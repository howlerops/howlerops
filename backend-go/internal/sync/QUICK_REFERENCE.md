# Organization Sync Protocol - Quick Reference

## Key Concepts

### Resource Visibility
- **Personal**: Only visible to the owner
- **Shared**: Visible to all members of the organization

### Permission Hierarchy
```
Owner > Admin > Member

Owner:  All permissions
Admin:  All except org deletion
Member: View + create own resources
```

### Conflict Resolution
**Last-Write-Wins** strategy based on `updated_at` timestamp

## API Quick Reference

### Pull (Download)
```
GET /api/sync/pull?since={ISO8601}&device_id={id}&org_id={optional}
```

Returns: Personal resources + Shared resources in user's orgs

### Push (Upload)
```
POST /api/sync/push
Body: { device_id, last_sync_at, changes[] }
```

Validates: Permissions, detects conflicts, applies changes

## Data Model Changes

### Connection Template
```go
// NEW FIELDS
UserID         string   // Owner
OrganizationID *string  // NULL = personal
Visibility     string   // "personal" | "shared"
SyncVersion    int      // Increments on update
```

### Saved Query
```go
// NEW FIELDS
UserID         string   // Owner
OrganizationID *string  // NULL = personal
Visibility     string   // "personal" | "shared"
SyncVersion    int      // Increments on update
```

## Permission Matrix

| Action | Personal | Shared (Owner) | Shared (Admin) | Shared (Member) |
|--------|----------|----------------|----------------|-----------------|
| View   | ✓        | ✓              | ✓              | ✓               |
| Create | ✓        | ✓              | ✓              | ✓               |
| Update Own | ✓    | ✓              | ✓              | ✓               |
| Update Others | ✗ | ✓              | ✓              | ✗               |
| Delete Own | ✓    | ✓              | ✓              | ✓               |
| Delete Others | ✗ | ✓              | ✓              | ✗               |

## Code Examples

### Create Shared Connection
```go
conn := &ConnectionTemplate{
    ID:             uuid.New().String(),
    Name:           "Shared DB",
    UserID:         userID,
    OrganizationID: &orgID,
    Visibility:     "shared",
    SyncVersion:    1,
}
```

### Filter Accessible Connections
```go
// Get user's org IDs
orgs, _ := orgRepo.GetByUserID(ctx, userID)
orgIDs := extractOrgIDs(orgs)

// Get accessible connections
conns, _ := store.ListAccessibleConnections(ctx, userID, orgIDs, since)
```

### Handle Conflict
```go
if conflict.Metadata.Resolution == "server_wins" {
    // Update local with server version
    localDB.Update(conflict.ResourceID, serverVersion)
    notify("Conflict resolved: server version kept")
}
```

### Validate Permission
```go
member, _ := orgRepo.GetMember(ctx, orgID, userID)

if !organization.HasPermission(member.Role, organization.PermCreateConnections) {
    return errors.New("insufficient permissions")
}

if !organization.CanUpdateResource(member.Role, resourceOwnerID, userID) {
    return errors.New("cannot update other user's resource")
}
```

## Testing Checklist

- [ ] User A creates personal connection - only A can see it
- [ ] User A creates shared connection in Org1 - User B (in Org1) can see it
- [ ] User C (not in Org1) cannot see Org1 shared connection
- [ ] Member cannot update Admin's shared connection
- [ ] Admin can update Member's shared connection
- [ ] Two users edit same resource - conflict detected and resolved
- [ ] Conflict metadata returned to client
- [ ] Sync logs created for all operations
- [ ] Permission denied returns proper error
- [ ] Sync version increments correctly

## Common SQL Queries

### Get User's Accessible Connections
```sql
SELECT * FROM connections
WHERE deleted_at IS NULL
  AND updated_at > ?
  AND (
    -- Personal connections
    (user_id = ? AND (visibility = 'personal' OR organization_id IS NULL))
    OR
    -- Shared in user's orgs
    (organization_id IN (?, ?, ...) AND visibility = 'shared')
  )
ORDER BY updated_at DESC;
```

### Get Sync Statistics
```sql
SELECT
    COUNT(*) as total_syncs,
    SUM(CASE WHEN action = 'pull' THEN 1 ELSE 0 END) as pulls,
    SUM(CASE WHEN action = 'push' THEN 1 ELSE 0 END) as pushes,
    SUM(conflict_count) as total_conflicts,
    COUNT(DISTINCT device_id) as unique_devices
FROM sync_logs
WHERE user_id = ?
  AND synced_at > ?;
```

## Troubleshooting Commands

### Check User's Organizations
```sql
SELECT o.id, o.name, om.role
FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = ?;
```

### Find Conflicted Resources
```sql
SELECT item_type, item_id, detected_at
FROM conflicts
WHERE user_id = ?
  AND resolved_at IS NULL
ORDER BY detected_at DESC;
```

### Audit Recent Sync Activity
```sql
SELECT
    user_id,
    action,
    resource_count,
    conflict_count,
    device_id,
    synced_at
FROM sync_logs
WHERE synced_at > datetime('now', '-1 day')
ORDER BY synced_at DESC
LIMIT 100;
```

## Performance Tips

1. **Index organization_id columns** for fast filtering
2. **Batch changes** in push requests (max 100 per request)
3. **Use since parameter** to only fetch recent changes
4. **Cache organization memberships** on client
5. **Implement exponential backoff** on sync failures

## Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| 401 | Unauthorized | Re-authenticate |
| 403 | Permission denied | Check role/permissions |
| 409 | Conflict | Pull latest and retry |
| 422 | Validation error | Fix data and retry |
| 429 | Rate limit | Wait and retry |
| 500 | Server error | Log and report |

## Migration Checklist

When deploying:

1. [ ] Run database migrations (add org fields)
2. [ ] Update existing data (set visibility='personal')
3. [ ] Deploy backend with new handlers
4. [ ] Update client sync logic
5. [ ] Test with multiple users/orgs
6. [ ] Monitor sync logs
7. [ ] Set up alerts for permission errors
8. [ ] Document for team

## Monitoring Queries

### Sync Health
```sql
-- Sync operations in last hour
SELECT COUNT(*) FROM sync_logs
WHERE synced_at > datetime('now', '-1 hour');

-- Average resources per sync
SELECT AVG(resource_count) FROM sync_logs
WHERE synced_at > datetime('now', '-1 day');

-- Conflict rate
SELECT
    SUM(conflict_count) * 100.0 / COUNT(*) as conflict_rate_percent
FROM sync_logs
WHERE synced_at > datetime('now', '-1 day');
```

### Performance
```sql
-- Largest syncs
SELECT user_id, resource_count, synced_at
FROM sync_logs
WHERE resource_count > 100
ORDER BY resource_count DESC
LIMIT 20;

-- Most active users
SELECT user_id, COUNT(*) as sync_count
FROM sync_logs
WHERE synced_at > datetime('now', '-1 day')
GROUP BY user_id
ORDER BY sync_count DESC
LIMIT 10;
```

## Contact

Questions? Check:
- Full docs: `ORG_SYNC_README.md`
- Code: `org_handler.go`, `conflict.go`, `metadata.go`
- Tests: `conflict_test.go`, `org_sync_integration_test.go`
