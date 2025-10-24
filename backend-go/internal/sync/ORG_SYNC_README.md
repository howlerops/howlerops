# Organization-Aware Sync Protocol

This document describes the updated sync protocol that supports organization-scoped resources (shared connections and queries).

## Overview

The sync protocol has been extended to handle:
- **Personal resources**: Owned by individual users, only synced to that user
- **Shared resources**: Owned by users but shared within organizations, synced to all organization members
- **Permission validation**: Ensuring users can only modify resources they have access to
- **Conflict resolution**: Handling concurrent edits to shared resources
- **Audit logging**: Tracking all sync operations

## Architecture

### Key Components

1. **OrgAwareHandler** (`org_handler.go`)
   - Handles HTTP requests for pull and push operations
   - Validates organization permissions
   - Integrates with ConflictResolver and MetadataTracker

2. **ConflictResolver** (`conflict.go`)
   - Detects conflicts when multiple users edit the same resource
   - Implements last-write-wins strategy
   - Returns conflict metadata to clients

3. **MetadataTracker** (`metadata.go`)
   - Logs all sync operations for audit trail
   - Tracks sync statistics per user
   - Manages sync metadata (last sync time, device info)

4. **TursoStore** (`turso_store.go`)
   - Extended database schema with organization fields
   - Organization-aware query methods
   - Sync log persistence

## Data Model

### Connection Template

```go
type ConnectionTemplate struct {
    ID             string
    Name           string
    Type           string
    // ... other fields ...

    // Organization fields
    UserID         string  // Owner of the connection
    OrganizationID *string // NULL for personal, set for shared
    Visibility     string  // "personal" or "shared"

    SyncVersion    int     // Incremented on each update
    UpdatedAt      time.Time
}
```

### Saved Query

```go
type SavedQuery struct {
    ID             string
    Name           string
    Query          string
    // ... other fields ...

    // Organization fields
    UserID         string  // Owner of the query
    OrganizationID *string // NULL for personal, set for shared
    Visibility     string  // "personal" or "shared"

    SyncVersion    int     // Incremented on each update
    UpdatedAt      time.Time
}
```

## API Endpoints

### Pull Endpoint

**GET** `/api/sync/pull?since={timestamp}&device_id={device}&org_id={optional}`

Retrieves resources the user has access to.

**Query Parameters:**
- `since` (required): ISO 8601 timestamp, only return resources updated after this time
- `device_id` (required): Unique device identifier
- `org_id` (optional): Filter by specific organization

**Response:**
```json
{
  "connections": [
    {
      "id": "conn-123",
      "name": "Production DB",
      "user_id": "user-456",
      "organization_id": "org-789",
      "visibility": "shared",
      "sync_version": 5,
      "updated_at": "2025-10-23T10:00:00Z"
    }
  ],
  "saved_queries": [...],
  "query_history": [...],
  "conflicts": [],
  "sync_timestamp": "2025-10-23T10:30:00Z",
  "has_more": false
}
```

**Access Rules:**
- Returns all personal resources (where `organization_id IS NULL` or `visibility='personal'`)
- Returns all shared resources in user's organizations (where `organization_id IN user_orgs` and `visibility='shared'`)

### Push Endpoint

**POST** `/api/sync/push`

Uploads local changes to the server.

**Request Body:**
```json
{
  "device_id": "device-abc",
  "last_sync_at": "2025-10-23T09:00:00Z",
  "changes": [
    {
      "id": "change-1",
      "item_type": "connection",
      "item_id": "conn-123",
      "action": "update",
      "sync_version": 5,
      "data": {
        "id": "conn-123",
        "name": "Production DB (Updated)",
        "user_id": "user-456",
        "organization_id": "org-789",
        "visibility": "shared",
        "updated_at": "2025-10-23T10:15:00Z"
      }
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "synced_at": "2025-10-23T10:30:00Z",
  "conflicts": [
    {
      "resource_type": "connection",
      "resource_id": "conn-123",
      "metadata": {
        "resolution": "server_wins",
        "reason": "server_newer",
        "server_version": 6,
        "client_version": 5,
        "conflicted_at": "2025-10-23T10:30:00Z"
      }
    }
  ],
  "rejected": [],
  "message": "Synced 3 items, 1 conflicts, 0 rejected"
}
```

## Permission Model

### Organization Roles

1. **Owner**
   - Can do everything
   - Can update/delete any resource in the organization
   - Can manage members and permissions

2. **Admin**
   - Can create, update, delete shared resources
   - Can update resources owned by others
   - Cannot delete the organization

3. **Member**
   - Can view shared resources
   - Can create new shared resources
   - Can only update/delete their own resources

### Permission Checks During Push

The system validates permissions before accepting changes:

```go
// For shared resources
if visibility == "shared" && organization_id != nil {
    // 1. Verify user is member of organization
    member := orgRepo.GetMember(ctx, organization_id, user_id)

    // 2. Check permission based on action
    if action == "create" {
        require: PermCreateConnections (or PermCreateQueries)
    } else if action == "update" {
        require: PermUpdateConnections (or PermUpdateQueries)

        // 3. Check if user can update resource owned by another user
        if resource.user_id != user_id {
            // Only Owner and Admin can do this
            if member.role == "member" {
                reject: "Cannot update resource owned by another user"
            }
        }
    }
}
```

## Conflict Resolution

### Conflict Detection

Conflicts occur when:
1. Server has a newer `sync_version` than client expected
2. Multiple users edit the same shared resource between syncs

### Resolution Strategy: Last-Write-Wins

The system uses timestamp-based conflict resolution:

```
if client.updated_at > server.updated_at:
    winner = client
    resolution = "client_wins"
else:
    winner = server
    resolution = "server_wins"
```

### Conflict Metadata

When a conflict is detected and resolved, the system returns metadata:

```go
type ConflictMetadata struct {
    Resolution    string    // "client_wins" | "server_wins"
    Reason        string    // "client_newer" | "server_newer"
    ServerVersion int       // Server's sync version
    ClientVersion int       // Client's sync version
    ConflictedAt  time.Time // When conflict was detected
}
```

Clients should:
1. Display conflict information to users
2. Update local data with the winning version
3. Log conflict events for debugging

## Sync Logging

All sync operations are logged for audit purposes:

```go
type SyncLog struct {
    ID             string
    UserID         string
    OrganizationID *string   // If syncing org-specific resources
    Action         string    // "pull" | "push"
    ResourceCount  int       // Number of resources synced
    ConflictCount  int       // Number of conflicts detected
    DeviceID       string
    ClientVersion  string
    SyncedAt       time.Time
}
```

Logs can be queried via:
```go
logs := metadataTracker.GetSyncHistory(ctx, userID, limit)
```

## Database Schema

### Updated Tables

```sql
-- Connections table
CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    -- ... other fields ...
    organization_id TEXT,              -- NEW
    visibility TEXT DEFAULT 'personal', -- NEW
    sync_version INTEGER DEFAULT 1,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_connections_org_id ON connections(organization_id);

-- Saved queries table
CREATE TABLE saved_queries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    query TEXT NOT NULL,
    -- ... other fields ...
    organization_id TEXT,              -- NEW
    visibility TEXT DEFAULT 'personal', -- NEW
    sync_version INTEGER DEFAULT 1,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_saved_queries_org_id ON saved_queries(organization_id);

-- Sync logs table (NEW)
CREATE TABLE sync_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    organization_id TEXT,
    action TEXT NOT NULL,
    resource_count INTEGER DEFAULT 0,
    conflict_count INTEGER DEFAULT 0,
    device_id TEXT NOT NULL,
    client_version TEXT,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sync_logs_user_id ON sync_logs(user_id);
CREATE INDEX idx_sync_logs_synced_at ON sync_logs(synced_at);
```

## Usage Examples

### Backend: Setup

```go
import (
    "github.com/sql-studio/internal/sync"
    "github.com/sql-studio/internal/organization"
)

// Initialize components
store := sync.NewTursoStore(dbURL, authToken, logger)
orgRepo := organization.NewRepository(db)
service := sync.NewService(store, config, logger)
handler := sync.NewOrgAwareHandler(service, orgRepo, logger)

// Register routes
router.HandleFunc("/api/sync/pull", handler.HandlePull).Methods("GET")
router.HandleFunc("/api/sync/push", handler.HandlePush).Methods("POST")
```

### Backend: Create Shared Connection

```go
conn := &sync.ConnectionTemplate{
    ID:             uuid.New().String(),
    Name:           "Shared Production DB",
    Type:           "postgres",
    Database:       "production",
    UserID:         userID,
    OrganizationID: &orgID,
    Visibility:     "shared",
    UpdatedAt:      time.Now(),
    SyncVersion:    1,
}

err := store.SaveConnection(ctx, userID, conn)
```

### Client: Pull Resources

```javascript
const response = await fetch('/api/sync/pull?' + new URLSearchParams({
    since: lastSyncTime.toISOString(),
    device_id: deviceId,
    org_id: currentOrgId // optional
}), {
    headers: {
        'Authorization': `Bearer ${authToken}`,
        'X-Client-Version': '1.0.0'
    }
});

const data = await response.json();

// Handle conflicts
if (data.conflicts.length > 0) {
    data.conflicts.forEach(conflict => {
        console.log(`Conflict on ${conflict.resource_type} ${conflict.resource_id}`);
        console.log(`Resolution: ${conflict.metadata.resolution}`);
        console.log(`Reason: ${conflict.metadata.reason}`);
    });
}

// Update local database
await updateLocalConnections(data.connections);
await updateLocalQueries(data.saved_queries);
```

### Client: Push Changes

```javascript
const changes = [
    {
        id: uuid(),
        item_type: 'connection',
        item_id: connectionId,
        action: 'update',
        sync_version: currentSyncVersion,
        updated_at: new Date().toISOString(),
        data: {
            id: connectionId,
            name: 'Updated Connection Name',
            user_id: userId,
            organization_id: orgId,
            visibility: 'shared',
            updated_at: new Date().toISOString()
        }
    }
];

const response = await fetch('/api/sync/push', {
    method: 'POST',
    headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json',
        'X-Client-Version': '1.0.0'
    },
    body: JSON.stringify({
        device_id: deviceId,
        last_sync_at: lastSyncTime.toISOString(),
        changes: changes
    })
});

const result = await response.json();

// Handle conflicts and rejections
if (!result.success) {
    console.error('Some changes were rejected:', result.rejected);
}

if (result.conflicts.length > 0) {
    // Handle conflicts - may need to pull latest and re-apply
    console.warn('Conflicts detected:', result.conflicts);
}
```

## Testing

### Unit Tests

Run conflict resolution tests:
```bash
go test -v ./internal/sync -run TestConflictResolver
```

### Integration Tests

Run organization sync integration tests:
```bash
go test -v ./internal/sync -run TestMultiUser
go test -v ./internal/sync -run TestOrganizationPermission
go test -v ./internal/sync -run TestSyncFiltering
```

### Test Coverage

Generate coverage report:
```bash
go test -coverprofile=coverage.out ./internal/sync
go tool cover -html=coverage.out
```

## Best Practices

### For Backend Developers

1. **Always validate permissions** before accepting changes to shared resources
2. **Log all sync operations** for audit trail and debugging
3. **Handle conflicts gracefully** - don't lose data
4. **Test multi-user scenarios** thoroughly
5. **Monitor sync performance** - watch for slow queries with many organizations

### For Frontend Developers

1. **Pull before push** - get latest changes before submitting
2. **Handle conflict responses** - inform users when conflicts occur
3. **Retry on failures** - transient network issues are common
4. **Batch changes** - send multiple changes in one push
5. **Track sync version** locally - include in push requests

### For Database Administrators

1. **Monitor index usage** on organization_id columns
2. **Archive old sync logs** periodically
3. **Optimize queries** for users in many organizations
4. **Set up replication** for high availability

## Troubleshooting

### Common Issues

**Issue**: User can't see shared connection
- Check: Is user member of the organization?
- Check: Is connection visibility set to 'shared'?
- Check: Is organization_id set correctly?

**Issue**: Permission denied on push
- Check: User's role in the organization
- Check: Required permissions for the action
- Check: Resource ownership if updating

**Issue**: Conflicts on every sync
- Check: Client sync_version tracking
- Check: Clock synchronization between devices
- Check: Multiple users editing same resource

**Issue**: Missing resources after sync
- Check: `since` parameter - might be too recent
- Check: Soft delete handling (deleted_at field)
- Check: Database indexes on updated_at columns

## Performance Considerations

### Optimization Tips

1. **Index Strategy**
   - Index on (user_id, updated_at) for personal resources
   - Index on (organization_id, visibility, updated_at) for shared resources
   - Composite indexes for common query patterns

2. **Query Limits**
   - Default limit of 1000 items per sync
   - Paginate large result sets
   - Use `has_more` flag to indicate additional data

3. **Sync Frequency**
   - Recommended: 5-15 minute intervals for background sync
   - Use websockets/push notifications for real-time updates
   - Implement exponential backoff on errors

4. **Database Connection Pooling**
   - Configure appropriate pool size
   - Monitor connection usage
   - Handle connection timeouts gracefully

## Security Considerations

1. **Authentication**
   - Always validate auth token before processing sync requests
   - Use secure token storage on clients

2. **Authorization**
   - Verify organization membership before returning resources
   - Check permissions on every push operation
   - Don't trust client-provided organization_id

3. **Data Validation**
   - Sanitize all input data
   - Never sync credentials (passwords, SSH keys)
   - Validate data sizes to prevent DoS

4. **Audit Logging**
   - Log all permission denials
   - Track suspicious sync patterns
   - Monitor for data exfiltration attempts

## Future Enhancements

Potential improvements to consider:

1. **Delta Sync**: Send only changed fields, not entire resources
2. **Compression**: Compress large sync payloads
3. **Webhooks**: Notify clients when shared resources change
4. **Conflict UI**: Built-in conflict resolution interface
5. **Sync History**: Track resource history for rollback
6. **Selective Sync**: Let users choose which orgs to sync
7. **Offline Mode**: Queue changes when offline, sync when online

## Support

For questions or issues:
- GitHub Issues: [sql-studio/issues](https://github.com/sql-studio/issues)
- Documentation: [sql-studio/docs](https://github.com/sql-studio/docs)
- Discord: [sql-studio community](https://discord.gg/sql-studio)
