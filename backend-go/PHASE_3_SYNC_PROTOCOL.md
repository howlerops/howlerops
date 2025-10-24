# Phase 3: Organization-Scoped Sync Protocol

**Version:** 1.0
**Status:** Design Phase
**Backward Compatible:** Yes (Phase 1 & 2 sync continues to work)

---

## Table of Contents

1. [Overview](#overview)
2. [Protocol Changes](#protocol-changes)
3. [Personal vs Organization Sync](#personal-vs-organization-sync)
4. [Shared Resources Sync](#shared-resources-sync)
5. [Permission-Aware Operations](#permission-aware-operations)
6. [Conflict Resolution](#conflict-resolution)
7. [Multi-User Scenarios](#multi-user-scenarios)
8. [WebSocket Protocol](#websocket-protocol-optional)
9. [Implementation Examples](#implementation-examples)

---

## Overview

Phase 3 extends the sync protocol to support organization-scoped syncing while maintaining full backward compatibility with Phase 1 & 2 (Individual tier).

### Key Concepts

1. **Dual Context**: Resources can exist in personal or organization context
2. **Explicit Scoping**: All sync operations explicitly specify context (user-only or org-scoped)
3. **Shared Resources**: Special handling for resources shared across org members
4. **Permission Enforcement**: Sync respects RBAC and resource-level permissions
5. **Backward Compatibility**: Existing Individual tier sync works unchanged

### Sync Contexts

```
User's Sync Space
├── Personal Resources (organization_id = NULL)
│   ├── Connections (owned by user)
│   ├── Saved Queries (owned by user)
│   └── Query History (user's executions)
│
└── Organization Resources (per organization)
    ├── Organization A
    │   ├── Connections (owned by user, shared with org)
    │   ├── Saved Queries (owned by user, shared with org)
    │   ├── Shared Connections (owned by others)
    │   └── Shared Queries (owned by others)
    │
    └── Organization B
        ├── ... (same structure)
```

---

## Protocol Changes

### Request Format Changes

#### SyncUploadRequest (Updated)

```typescript
interface SyncUploadRequest {
  // Existing fields
  device_id: string;
  last_sync_at: string; // ISO8601
  changes: SyncChange[];

  // NEW: Organization context (optional)
  organization_id?: string;
}
```

**Behavior:**
- If `organization_id` is `null` or omitted: Syncs personal resources only
- If `organization_id` is specified: Syncs resources for that organization

#### SyncChange (Updated)

```typescript
interface SyncChange {
  // Existing fields
  id: string;
  item_type: 'connection' | 'saved_query' | 'query_history';
  item_id: string;
  action: 'create' | 'update' | 'delete';
  data: ConnectionTemplate | SavedQuery | QueryHistory;
  updated_at: string; // ISO8601
  sync_version: number;
  device_id: string;

  // NEW: Organization context (optional)
  organization_id?: string; // NULL for personal, set for org resources

  // NEW: Sharing metadata (only for shared resources)
  sharing_metadata?: {
    is_shared: boolean;
    shared_with_orgs?: string[]; // Organization IDs this resource is shared with
    permissions?: Permission[]; // User's permissions on this resource
  };
}
```

#### SyncDownloadRequest (Updated)

```typescript
interface SyncDownloadRequest {
  // Existing fields
  since: string; // ISO8601
  device_id: string;

  // NEW: Organization filter (optional)
  organization_id?: string; // NULL = personal only, set = specific org
  include_shared?: boolean; // Include resources shared by others (default: true)
}
```

**Behavior:**
- `organization_id = null`: Download only personal resources
- `organization_id = "org_123"`: Download resources for org_123
- `organization_id = "org_123", include_shared = true`: Download both owned and shared resources

#### SyncDownloadResponse (Updated)

```typescript
interface SyncDownloadResponse {
  // Existing fields (owned resources)
  connections: ConnectionTemplate[];
  saved_queries: SavedQuery[];
  query_history: QueryHistory[];
  sync_timestamp: string; // ISO8601
  has_more: boolean;

  // NEW: Shared resources metadata
  shared_resources?: {
    connections: SharedResourceInfo[];
    queries: SharedResourceInfo[];
  };

  // NEW: Organization sync metadata
  organization_metadata?: {
    organization_id: string;
    organization_name: string;
    user_role: OrganizationRole;
    last_sync_at: string;
  };
}

interface SharedResourceInfo {
  resource_id: string; // Connection or query ID
  resource_type: 'connection' | 'saved_query';
  owner_id: string;
  owner_username: string;
  shared_by: string;
  shared_at: string; // ISO8601
  permissions: Permission[]; // What user can do with this resource
  notes?: string;

  // Resource preview (no credentials)
  preview: {
    name: string;
    type?: string; // For connections: postgres, mysql, etc.
    description?: string;
    tags?: string[];
  };
}
```

---

## Personal vs Organization Sync

### Personal Sync (Unchanged from Phase 2)

**Use Case:** Individual tier users, or team users syncing personal resources

```typescript
// Upload personal connections
const uploadReq: SyncUploadRequest = {
  device_id: "device_abc123",
  last_sync_at: "2024-01-15T10:00:00Z",
  organization_id: null, // or omit entirely
  changes: [
    {
      id: "change_1",
      item_type: "connection",
      item_id: "conn_xyz",
      action: "create",
      organization_id: null, // Personal resource
      data: {
        id: "conn_xyz",
        name: "My Local PostgreSQL",
        type: "postgres",
        // ... connection details (NO password)
      },
      updated_at: "2024-01-15T10:05:00Z",
      sync_version: 1,
      device_id: "device_abc123"
    }
  ]
};

// Response
const uploadResp: SyncUploadResponse = {
  success: true,
  synced_at: "2024-01-15T10:10:00Z",
  conflicts: [],
  rejected: []
};
```

```typescript
// Download personal resources
const downloadReq: SyncDownloadRequest = {
  since: "2024-01-15T09:00:00Z",
  device_id: "device_abc123",
  organization_id: null // Personal only
};

const downloadResp: SyncDownloadResponse = {
  connections: [
    {
      id: "conn_xyz",
      name: "My Local PostgreSQL",
      // ... connection details
      organization_id: null // Personal
    }
  ],
  saved_queries: [],
  query_history: [],
  sync_timestamp: "2024-01-15T10:15:00Z",
  has_more: false
};
```

### Organization Sync

**Use Case:** Team members syncing organization resources

```typescript
// Upload organization connection
const uploadReq: SyncUploadRequest = {
  device_id: "device_abc123",
  last_sync_at: "2024-01-15T10:00:00Z",
  organization_id: "org_team123", // Organization context
  changes: [
    {
      id: "change_2",
      item_type: "connection",
      item_id: "conn_prod",
      action: "create",
      organization_id: "org_team123", // Organization resource
      data: {
        id: "conn_prod",
        name: "Production DB",
        type: "postgres",
        // ... connection details
      },
      updated_at: "2024-01-15T11:00:00Z",
      sync_version: 1,
      device_id: "device_abc123",
      sharing_metadata: {
        is_shared: true,
        shared_with_orgs: ["org_team123"],
        permissions: ["read", "execute"] // Permissions when sharing
      }
    }
  ]
};
```

```typescript
// Download organization resources (owned + shared)
const downloadReq: SyncDownloadRequest = {
  since: "2024-01-15T09:00:00Z",
  device_id: "device_abc123",
  organization_id: "org_team123",
  include_shared: true // Include shared resources
};

const downloadResp: SyncDownloadResponse = {
  // Resources owned by this user (shared with org)
  connections: [
    {
      id: "conn_prod",
      name: "Production DB",
      type: "postgres",
      organization_id: "org_team123",
      // ... details
    }
  ],
  saved_queries: [],
  query_history: [],

  // Resources shared by OTHER users
  shared_resources: {
    connections: [
      {
        resource_id: "conn_analytics",
        resource_type: "connection",
        owner_id: "user_jane",
        owner_username: "jane_admin",
        shared_by: "user_jane",
        shared_at: "2024-01-14T10:00:00Z",
        permissions: ["read", "execute"],
        notes: "Analytics DB - read-only access",
        preview: {
          name: "Analytics PostgreSQL",
          type: "postgres",
          description: "Analytics database for reports"
        }
      }
    ],
    queries: [
      {
        resource_id: "query_dau",
        resource_type: "saved_query",
        owner_id: "user_john",
        owner_username: "john_member",
        shared_by: "user_john",
        shared_at: "2024-01-15T08:00:00Z",
        permissions: ["read", "execute", "modify"],
        preview: {
          name: "Daily Active Users",
          description: "Calculate DAU from events",
          tags: ["analytics", "metrics"]
        }
      }
    ]
  },

  organization_metadata: {
    organization_id: "org_team123",
    organization_name: "Acme Corp Engineering",
    user_role: "member",
    last_sync_at: "2024-01-15T11:30:00Z"
  },

  sync_timestamp: "2024-01-15T11:30:00Z",
  has_more: false
};
```

---

## Shared Resources Sync

### Shared Resource Lifecycle

1. **Owner creates resource** (personal or org-scoped)
2. **Owner shares with organization** (via API, not sync)
3. **Other members download shared resource** (via sync)
4. **Members use resource** (based on permissions)
5. **Owner updates resource** → sync propagates changes
6. **Owner unshares** → members receive deletion in next sync

### Sync Behavior for Shared Resources

#### Owner's Perspective

```typescript
// Owner creates and shares a connection
// Step 1: Create connection locally
const localConnection = {
  id: "conn_shared_db",
  name: "Shared Analytics DB",
  type: "postgres",
  // ... details
};

// Step 2: Sync to cloud (normal sync)
await syncService.upload({
  device_id: "device_owner",
  organization_id: "org_team123",
  changes: [{
    item_type: "connection",
    item_id: "conn_shared_db",
    action: "create",
    organization_id: "org_team123",
    data: localConnection,
    // ...
  }]
});

// Step 3: Share with organization (separate API call, NOT sync)
await api.post(`/api/organizations/org_team123/connections/conn_shared_db/share`, {
  permissions: ["read", "execute"],
  notes: "Analytics DB for team"
});

// Step 4: Update connection → syncs to all members
localConnection.name = "Updated Analytics DB";
await syncService.upload({
  // ... change with action: "update"
});
// Members will receive update in their next sync
```

#### Member's Perspective

```typescript
// Member downloads shared resources
const resp = await syncService.download({
  since: lastSyncTime,
  device_id: "device_member",
  organization_id: "org_team123",
  include_shared: true
});

// Process shared connections
for (const sharedConn of resp.shared_resources.connections) {
  // Store in local DB with metadata
  await db.saveSharedConnection({
    id: sharedConn.resource_id,
    name: sharedConn.preview.name,
    type: sharedConn.preview.type,
    owner: sharedConn.owner_username,
    permissions: sharedConn.permissions,
    is_shared: true,
    read_only: !sharedConn.permissions.includes("modify")
  });

  // UI shows: "Analytics PostgreSQL (shared by jane_admin)"
}

// Member cannot modify (without "modify" permission)
if (sharedConn.permissions.includes("modify")) {
  // Show edit button
} else {
  // Read-only indicator
}
```

#### Shared Resource Updates

When owner updates a shared resource:

```typescript
// Server-side sync logic
async function handleSharedResourceUpdate(
  change: SyncChange,
  orgID: string
): Promise<void> {
  // 1. Apply change to database
  await db.updateConnection(change.item_id, change.data);

  // 2. Increment sync_version
  await db.incrementSyncVersion(change.item_id);

  // 3. Notify other org members (via sync)
  // When they next sync, they'll receive the updated resource

  // 4. (Optional) Real-time notification via WebSocket
  await notifyOrgMembers(orgID, {
    type: "resource_updated",
    resource_type: "connection",
    resource_id: change.item_id,
    updated_by: change.user_id,
    message: `${change.data.name} was updated`
  });
}
```

---

## Permission-Aware Operations

### Upload with Permission Checks

```typescript
// Server-side upload handler
async function handleSyncUpload(
  req: SyncUploadRequest,
  userID: string
): Promise<SyncUploadResponse> {
  const resp: SyncUploadResponse = {
    success: true,
    conflicts: [],
    rejected: []
  };

  for (const change of req.changes) {
    // Check organization membership
    if (change.organization_id) {
      const isMember = await checkOrgMembership(userID, change.organization_id);
      if (!isMember) {
        resp.rejected.push({
          change,
          reason: "not_a_member",
          message: "You are not a member of this organization"
        });
        continue;
      }
    }

    // Check permissions for updates/deletes
    if (change.action === "update" || change.action === "delete") {
      const hasPermission = await checkResourcePermission(
        userID,
        change.organization_id,
        change.item_id,
        change.item_type,
        "modify" // or "delete"
      );

      if (!hasPermission) {
        resp.rejected.push({
          change,
          reason: "insufficient_permissions",
          message: "You don't have permission to modify this resource"
        });
        continue;
      }
    }

    // Process change
    const conflict = await processChange(userID, change);
    if (conflict) {
      resp.conflicts.push(conflict);
    }
  }

  resp.synced_at = new Date().toISOString();
  return resp;
}
```

### Download with Permission Filtering

```typescript
// Server-side download handler
async function handleSyncDownload(
  req: SyncDownloadRequest,
  userID: string
): Promise<SyncDownloadResponse> {
  const resp: SyncDownloadResponse = {
    connections: [],
    saved_queries: [],
    query_history: [],
    sync_timestamp: new Date().toISOString(),
    has_more: false
  };

  if (req.organization_id) {
    // Check membership
    const member = await getOrgMember(userID, req.organization_id);
    if (!member) {
      throw new Error("Not a member of this organization");
    }

    // Get owned resources in this org
    const ownedConns = await db.getConnectionsSince(
      userID,
      req.organization_id,
      req.since
    );
    resp.connections = ownedConns;

    // Get shared resources (if requested)
    if (req.include_shared) {
      const sharedConns = await db.getSharedConnections(
        req.organization_id,
        userID
      );

      // Filter by permissions
      resp.shared_resources = {
        connections: sharedConns
          .filter(sc => sc.permissions.includes("read"))
          .map(sc => ({
            resource_id: sc.connection_id,
            resource_type: "connection",
            owner_id: sc.owner_id,
            owner_username: sc.owner_username,
            shared_by: sc.shared_by,
            shared_at: sc.shared_at,
            permissions: sc.permissions,
            notes: sc.notes,
            preview: {
              name: sc.connection_name,
              type: sc.connection_type,
              description: sc.connection_description
            }
          })),
        queries: [] // Similar for queries
      };
    }

    // Add org metadata
    resp.organization_metadata = {
      organization_id: req.organization_id,
      organization_name: member.organization_name,
      user_role: member.role,
      last_sync_at: new Date().toISOString()
    };
  } else {
    // Personal resources only
    resp.connections = await db.getConnectionsSince(userID, null, req.since);
    // ... similar for queries and history
  }

  return resp;
}
```

---

## Conflict Resolution

### Conflict Types in Organization Context

1. **Personal Resource Conflict**: Same as Phase 2 (last write wins, keep both, user choice)
2. **Owned Org Resource Conflict**: User conflicts with their own changes across devices
3. **Shared Resource Conflict**: User modifies shared resource, owner also modifies

### Shared Resource Conflict Resolution

**Scenario:** User with "modify" permission edits shared resource, owner also edits

```typescript
interface SharedResourceConflict extends Conflict {
  item_type: "connection" | "saved_query";
  item_id: string;

  // Local version (user's edit)
  local_version: {
    data: any;
    updated_at: string;
    sync_version: number;
    user_id: string; // Current user
    device_id: string;
  };

  // Remote version (owner's edit)
  remote_version: {
    data: any;
    updated_at: string;
    sync_version: number;
    user_id: string; // Resource owner
    is_owner: true;
  };

  // Metadata
  resource_owner: string;
  shared_with_org: string;
  current_user_permissions: Permission[];
}
```

**Resolution Strategy:**

```typescript
async function resolveSharedResourceConflict(
  conflict: SharedResourceConflict
): Promise<ConflictResolution> {
  // 1. Owner always wins (by default)
  if (conflict.remote_version.is_owner) {
    return {
      strategy: "owner_wins",
      winner: conflict.remote_version,
      message: "Resource owner's changes take precedence"
    };
  }

  // 2. If both have modify permission, use last write wins
  if (conflict.current_user_permissions.includes("modify")) {
    if (conflict.local_version.updated_at > conflict.remote_version.updated_at) {
      return {
        strategy: "last_write_wins",
        winner: conflict.local_version
      };
    } else {
      return {
        strategy: "last_write_wins",
        winner: conflict.remote_version
      };
    }
  }

  // 3. If user doesn't have modify permission, remote wins
  return {
    strategy: "remote_wins",
    winner: conflict.remote_version,
    message: "You don't have permission to modify this resource"
  };
}
```

### Multi-Device Conflict (Same User, Different Devices)

```typescript
// User edits on Device A and Device B before syncing
// Device A syncs first (succeeds)
// Device B syncs second (detects conflict)

// Server detects conflict
const conflict: Conflict = {
  id: "conflict_123",
  item_type: "connection",
  item_id: "conn_xyz",
  local_version: {
    data: { name: "Updated from Device B" },
    updated_at: "2024-01-15T10:05:00Z",
    sync_version: 1,
    device_id: "device_b"
  },
  remote_version: {
    data: { name: "Updated from Device A" },
    updated_at: "2024-01-15T10:03:00Z",
    sync_version: 2, // Already incremented by Device A sync
    device_id: "device_a"
  },
  detected_at: "2024-01-15T10:06:00Z"
};

// Default resolution: Last write wins
if (conflict.local_version.updated_at > conflict.remote_version.updated_at) {
  // Device B's change is newer, accept it
  await resolveConflict(conflict.id, "last_write_wins", "local");
} else {
  // Device A's change is newer, reject Device B's change
  await resolveConflict(conflict.id, "last_write_wins", "remote");
}
```

---

## Multi-User Scenarios

### Scenario 1: Two Team Members Edit Same Shared Query

**Setup:**
- Alice (owner) creates and shares query "Daily Revenue"
- Bob (member with "modify" permission) edits it
- Alice also edits it (before Bob syncs)

**Flow:**

```typescript
// 1. Alice creates and shares query
await alice.sync.upload({
  changes: [{
    item_type: "saved_query",
    item_id: "query_revenue",
    action: "create",
    organization_id: "org_acme",
    data: {
      name: "Daily Revenue",
      query: "SELECT SUM(amount) FROM orders WHERE date = CURRENT_DATE"
    }
  }]
});

await alice.api.shareQuery("org_acme", "query_revenue", {
  permissions: ["read", "execute", "modify"]
});

// 2. Bob downloads shared query
const bobDownload = await bob.sync.download({
  organization_id: "org_acme",
  include_shared: true
});
// Bob sees "Daily Revenue" in shared_resources.queries

// 3. Bob edits query locally (offline)
const bobEdit = {
  name: "Daily Revenue (Updated by Bob)",
  query: "SELECT SUM(amount) FROM orders WHERE date = CURRENT_DATE AND status = 'paid'"
};

// 4. Alice ALSO edits query (while Bob is offline)
const aliceEdit = {
  name: "Daily Revenue - Paid Orders",
  query: "SELECT SUM(amount) FROM orders WHERE date = CURRENT_DATE AND status IN ('paid', 'completed')"
};

await alice.sync.upload({
  changes: [{
    item_id: "query_revenue",
    action: "update",
    data: aliceEdit,
    sync_version: 1
  }]
});
// Server: sync_version → 2

// 5. Bob comes online and syncs
await bob.sync.upload({
  changes: [{
    item_id: "query_revenue",
    action: "update",
    data: bobEdit,
    sync_version: 1 // Stale version!
  }]
});

// Server response: CONFLICT
{
  success: true,
  conflicts: [{
    id: "conflict_456",
    item_type: "saved_query",
    item_id: "query_revenue",
    local_version: {
      data: bobEdit,
      updated_at: "2024-01-15T11:00:00Z",
      sync_version: 1,
      user_id: "user_bob"
    },
    remote_version: {
      data: aliceEdit,
      updated_at: "2024-01-15T10:50:00Z",
      sync_version: 2,
      user_id: "user_alice",
      is_owner: true
    },
    resource_owner: "user_alice"
  }]
}

// 6. Bob resolves conflict
// Option A: Owner wins (Alice's edit)
await bob.sync.resolveConflict("conflict_456", {
  strategy: "owner_wins"
});

// Option B: User choice (Bob chooses to keep his edit)
await bob.sync.resolveConflict("conflict_456", {
  strategy: "user_choice",
  chosen_version: "local"
});

// Option C: Manual merge (Bob manually combines changes)
const mergedEdit = {
  name: "Daily Revenue - Paid Orders (Combined)",
  query: aliceEdit.query // Use Alice's query logic
};
await bob.sync.upload({
  changes: [{
    item_id: "query_revenue",
    action: "update",
    data: mergedEdit,
    sync_version: 2 // Use latest version
  }]
});
```

### Scenario 2: Owner Unshares While Member is Offline

```typescript
// 1. Alice shares connection with team
await alice.api.shareConnection("org_acme", "conn_prod", {
  permissions: ["read", "execute"]
});

// 2. Bob downloads it
const bobDownload = await bob.sync.download({
  organization_id: "org_acme",
  include_shared: true
});
// Bob has conn_prod in shared_resources

// 3. Alice unshares (while Bob is offline)
await alice.api.unshareConnection("org_acme", "conn_prod");

// 4. Bob comes online and syncs
const bobDownloadAgain = await bob.sync.download({
  organization_id: "org_acme",
  include_shared: true
});

// Server response: conn_prod is NOT in shared_resources anymore
// Bob's client detects removal
if (!newShared.includes("conn_prod") && oldShared.includes("conn_prod")) {
  // Remove from local DB
  await db.removeSharedConnection("conn_prod");

  // Show notification
  showNotification({
    type: "info",
    message: "Production DB is no longer shared with your team"
  });
}
```

---

## WebSocket Protocol (Optional)

For real-time collaboration, Phase 3 can optionally use WebSockets to notify users of changes.

### WebSocket Connection

```typescript
// Client establishes WebSocket connection
const ws = new WebSocket("wss://api.sqlstudio.io/ws");

ws.onopen = () => {
  // Authenticate
  ws.send(JSON.stringify({
    type: "auth",
    token: jwtToken
  }));

  // Subscribe to organization updates
  ws.send(JSON.stringify({
    type: "subscribe",
    channels: [
      `org:org_acme`,
      `user:user_bob` // Personal notifications
    ]
  }));
};
```

### Real-Time Notifications

```typescript
// Server sends notification when resource is updated
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  switch (msg.type) {
    case "resource_updated":
      handleResourceUpdate(msg);
      break;
    case "resource_shared":
      handleResourceShared(msg);
      break;
    case "resource_unshared":
      handleResourceUnshared(msg);
      break;
    case "member_joined":
      handleMemberJoined(msg);
      break;
  }
};

function handleResourceUpdate(msg: ResourceUpdateMessage) {
  // {
  //   type: "resource_updated",
  //   organization_id: "org_acme",
  //   resource_type: "connection",
  //   resource_id: "conn_prod",
  //   updated_by: "user_alice",
  //   updated_by_username: "alice_admin",
  //   updated_at: "2024-01-15T12:00:00Z",
  //   message: "Production DB was updated by alice_admin"
  // }

  // Show notification
  showNotification({
    type: "info",
    message: msg.message
  });

  // Trigger background sync to get latest changes
  syncService.backgroundSync();
}
```

---

## Implementation Examples

### Complete Sync Flow (Frontend)

```typescript
class OrganizationSyncService {
  private db: LocalDatabase;
  private api: APIClient;
  private deviceId: string;

  async syncOrganization(orgId: string): Promise<SyncResult> {
    // 1. Get last sync time for this org
    const lastSync = await this.db.getLastSyncTime(orgId);

    // 2. Collect local changes since last sync
    const localChanges = await this.collectLocalChanges(orgId, lastSync);

    // 3. Upload local changes
    const uploadResp = await this.api.syncUpload({
      device_id: this.deviceId,
      organization_id: orgId,
      last_sync_at: lastSync,
      changes: localChanges
    });

    // 4. Handle conflicts
    if (uploadResp.conflicts.length > 0) {
      await this.handleConflicts(uploadResp.conflicts);
    }

    // 5. Download remote changes
    const downloadResp = await this.api.syncDownload({
      device_id: this.deviceId,
      organization_id: orgId,
      since: lastSync,
      include_shared: true
    });

    // 6. Apply remote changes to local DB
    await this.applyRemoteChanges(downloadResp);

    // 7. Apply shared resources
    await this.applySharedResources(downloadResp.shared_resources);

    // 8. Update last sync time
    await this.db.setLastSyncTime(orgId, downloadResp.sync_timestamp);

    return {
      success: true,
      uploaded: localChanges.length,
      downloaded: downloadResp.connections.length +
                  downloadResp.saved_queries.length,
      conflicts: uploadResp.conflicts.length,
      shared: (downloadResp.shared_resources?.connections.length ?? 0) +
              (downloadResp.shared_resources?.queries.length ?? 0)
    };
  }

  private async applySharedResources(
    sharedResources?: { connections: SharedResourceInfo[], queries: SharedResourceInfo[] }
  ): Promise<void> {
    if (!sharedResources) return;

    // Apply shared connections
    for (const sharedConn of sharedResources.connections) {
      await this.db.saveSharedConnection({
        id: sharedConn.resource_id,
        name: sharedConn.preview.name,
        type: sharedConn.preview.type,
        owner_id: sharedConn.owner_id,
        owner_username: sharedConn.owner_username,
        shared_by: sharedConn.shared_by,
        permissions: sharedConn.permissions,
        notes: sharedConn.notes,
        is_shared: true,
        read_only: !sharedConn.permissions.includes("modify")
      });
    }

    // Similar for queries...
  }

  private async handleConflicts(conflicts: Conflict[]): Promise<void> {
    for (const conflict of conflicts) {
      // Show conflict UI to user
      const resolution = await this.showConflictDialog(conflict);

      // Resolve based on user choice
      await this.api.resolveConflict(conflict.id, resolution);
    }
  }
}
```

### Backend Sync Handler

```go
package sync

func (s *Service) HandleSyncUpload(ctx context.Context, req *SyncUploadRequest) (*SyncUploadResponse, error) {
    userID := getUserIDFromContext(ctx)

    resp := &SyncUploadResponse{
        Success: true,
        Conflicts: []Conflict{},
        Rejected: []RejectedChange{},
    }

    // Check organization membership if org context provided
    if req.OrganizationID != nil {
        if err := s.permChecker.CheckMembership(ctx, userID, *req.OrganizationID); err != nil {
            return nil, fmt.Errorf("not a member: %w", err)
        }
    }

    for _, change := range req.Changes {
        // Validate org context matches request
        if !matchesOrgContext(change.OrganizationID, req.OrganizationID) {
            resp.Rejected = append(resp.Rejected, RejectedChange{
                Change: change,
                Reason: "organization_id_mismatch",
            })
            continue
        }

        // Check permissions for updates/deletes
        if change.Action == SyncActionUpdate || change.Action == SyncActionDelete {
            hasPermission, err := s.checkPermission(ctx, userID, change)
            if err != nil || !hasPermission {
                resp.Rejected = append(resp.Rejected, RejectedChange{
                    Change: change,
                    Reason: "insufficient_permissions",
                })
                continue
            }
        }

        // Process change
        conflict, err := s.processChange(ctx, userID, change)
        if err != nil {
            return nil, err
        }
        if conflict != nil {
            resp.Conflicts = append(resp.Conflicts, *conflict)
        }
    }

    resp.SyncedAt = time.Now().Format(time.RFC3339)

    // Log sync event
    s.auditService.LogSync(ctx, userID, req.OrganizationID, len(req.Changes), len(resp.Conflicts))

    return resp, nil
}
```

---

## Migration from Phase 2 to Phase 3

### Automatic Migration for Individual Users

When a user first accesses Phase 3 features:

1. Backend automatically creates "personal" organization
2. All existing personal resources remain unchanged (organization_id = NULL)
3. Sync protocol continues to work exactly as before

```typescript
// Phase 2 sync (continues to work)
const resp = await sync.upload({
  device_id: "device_abc",
  changes: [...]
  // No organization_id = personal resources
});

// Phase 3 sync (new)
const respOrg = await sync.upload({
  device_id: "device_abc",
  organization_id: "org_team123",
  changes: [...]
});
```

### Frontend Sync Strategy

```typescript
class SyncManager {
  async performFullSync(): Promise<void> {
    // 1. Sync personal resources (Phase 2 compatible)
    await this.syncPersonal();

    // 2. Sync each organization (Phase 3)
    const orgs = await this.getOrganizations();
    for (const org of orgs) {
      if (org.id !== this.personalOrgId) {
        await this.syncOrganization(org.id);
      }
    }
  }

  private async syncPersonal(): Promise<void> {
    await this.orgSyncService.syncOrganization(null); // null = personal
  }

  private async syncOrganization(orgId: string): Promise<void> {
    await this.orgSyncService.syncOrganization(orgId);
  }
}
```

---

## Summary

Phase 3 sync protocol extends Phase 2 with:

1. **Organization Context**: All sync operations can specify organization scope
2. **Shared Resources**: Special handling for resources shared across team members
3. **Permission Enforcement**: Sync respects RBAC and resource-level permissions
4. **Backward Compatibility**: Phase 1 & 2 sync continues to work unchanged
5. **Conflict Resolution**: Enhanced strategies for multi-user scenarios
6. **Real-Time Updates**: Optional WebSocket support for live collaboration

**Key Principles:**
- Explicit is better than implicit (always specify context)
- Security first (permission checks on every operation)
- Graceful degradation (works offline, syncs when online)
- User control (conflicts always require user decision when ambiguous)

---

**Document Status**: Ready for Implementation
**Last Updated**: 2024-01-15
**Version**: 1.0
