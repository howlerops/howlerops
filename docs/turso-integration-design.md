# Turso Database Integration & Sync Strategy for Howlerops

## Executive Summary

This document outlines a comprehensive strategy for integrating Turso (libSQL) as a cloud sync backend for Howlerops's tiered architecture. The design optimizes for minimal row writes, efficient sync patterns, and scalable multi-tenancy while leveraging Turso's edge replication capabilities.

**Key Metrics:**
- Individual tier: ~$29/month Turso cost (est. 500K rows written/month)
- Team tier: ~$29-89/month depending on team size and activity
- Sync latency: <500ms for incremental updates
- Offline support: Full offline queue with conflict resolution
- Data privacy: Zero credential sync, optional query redaction

---

## Table of Contents

1. [Database Schema Design](#1-database-schema-design)
2. [Sync Architecture](#2-sync-architecture)
3. [Query Optimization](#3-query-optimization)
4. [Data Isolation & Security](#4-data-isolation--security)
5. [Turso-Specific Features](#5-turso-specific-features)
6. [Performance Considerations](#6-performance-considerations)
7. [Cost Optimization](#7-cost-optimization)
8. [Authentication & Authorization](#8-authentication--authorization)
9. [Data Privacy](#9-data-privacy)
10. [Migration & Operations](#10-migration--operations)
11. [Implementation Roadmap](#11-implementation-roadmap)

---

## 1. Database Schema Design

### 1.1 Individual Tier Schema

```sql
-- Schema version tracking
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- User profile and sync metadata
CREATE TABLE users (
    user_id TEXT PRIMARY KEY,           -- UUID from auth provider
    email TEXT UNIQUE NOT NULL,
    display_name TEXT,
    tier TEXT NOT NULL DEFAULT 'individual', -- 'individual' | 'team'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_sync_at DATETIME,
    sync_enabled BOOLEAN DEFAULT 1,
    settings_json TEXT,                 -- JSON: { theme, preferences }
    deleted_at DATETIME                 -- Soft delete for GDPR
);

-- Connection metadata (NO PASSWORDS)
CREATE TABLE connections (
    connection_id TEXT PRIMARY KEY,     -- UUID from client
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    db_type TEXT NOT NULL,              -- postgresql, mysql, etc.
    host TEXT,
    port INTEGER,
    database_name TEXT NOT NULL,
    username TEXT,
    -- NO password field
    ssl_mode TEXT,
    use_tunnel BOOLEAN DEFAULT 0,
    use_vpc BOOLEAN DEFAULT 0,
    environments TEXT,                  -- JSON array: ["dev", "prod"]
    parameters TEXT,                    -- JSON object for db-specific params
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_connections_user ON connections(user_id, deleted_at);
CREATE INDEX idx_connections_last_used ON connections(user_id, last_used_at DESC);

-- Query tabs state
CREATE TABLE query_tabs (
    tab_id TEXT PRIMARY KEY,            -- UUID from client
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    tab_type TEXT NOT NULL DEFAULT 'sql', -- 'sql' | 'ai'
    content TEXT,                       -- The actual query/content
    connection_id TEXT,                 -- Single connection mode
    selected_connection_ids TEXT,       -- JSON array for multi-DB mode
    environment_snapshot TEXT,          -- Environment filter at creation
    ai_session_id TEXT,                 -- Reference to AI session
    position INTEGER,                   -- Tab order
    is_pinned BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES connections(connection_id) ON DELETE SET NULL
);

CREATE INDEX idx_query_tabs_user ON query_tabs(user_id, deleted_at, position);
CREATE INDEX idx_query_tabs_updated ON query_tabs(user_id, updated_at DESC);

-- Query history (with optional redaction)
CREATE TABLE query_history (
    history_id TEXT PRIMARY KEY,        -- UUID
    user_id TEXT NOT NULL,
    tab_id TEXT,                        -- Nullable for ad-hoc queries
    connection_id TEXT,
    query_text TEXT,                    -- Optionally redacted
    query_hash TEXT,                    -- SHA256 for dedup, even if redacted
    is_redacted BOOLEAN DEFAULT 0,
    row_count INTEGER,
    affected_rows INTEGER,
    execution_time_ms INTEGER,
    error_message TEXT,
    executed_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (tab_id) REFERENCES query_tabs(tab_id) ON DELETE SET NULL,
    FOREIGN KEY (connection_id) REFERENCES connections(connection_id) ON DELETE SET NULL
);

CREATE INDEX idx_query_history_user ON query_history(user_id, executed_at DESC);
CREATE INDEX idx_query_history_connection ON query_history(connection_id, executed_at DESC);
CREATE INDEX idx_query_history_hash ON query_history(user_id, query_hash); -- Dedup

-- Saved queries library
CREATE TABLE saved_queries (
    saved_query_id TEXT PRIMARY KEY,    -- UUID
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    query_text TEXT NOT NULL,
    tags TEXT,                          -- JSON array: ["optimization", "reporting"]
    connection_id TEXT,                 -- Optional default connection
    is_favorite BOOLEAN DEFAULT 0,
    execution_count INTEGER DEFAULT 0,
    last_executed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES connections(connection_id) ON DELETE SET NULL
);

CREATE INDEX idx_saved_queries_user ON saved_queries(user_id, deleted_at);
CREATE INDEX idx_saved_queries_favorite ON saved_queries(user_id, is_favorite DESC, updated_at DESC);
CREATE INDEX idx_saved_queries_tags ON saved_queries(user_id, tags); -- JSON1 extension

-- AI conversation sessions
CREATE TABLE ai_sessions (
    session_id TEXT PRIMARY KEY,        -- UUID
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,                       -- Compressed conversation summary
    summary_tokens INTEGER,
    message_count INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_ai_sessions_user ON ai_sessions(user_id, updated_at DESC);

-- AI conversation messages (with aggressive pruning)
CREATE TABLE ai_messages (
    message_id TEXT PRIMARY KEY,        -- UUID
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,                 -- 'system' | 'user' | 'assistant'
    content TEXT NOT NULL,
    tokens INTEGER NOT NULL,
    metadata TEXT,                      -- JSON: additional context
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (session_id) REFERENCES ai_sessions(session_id) ON DELETE CASCADE
);

CREATE INDEX idx_ai_messages_session ON ai_messages(session_id, created_at);

-- UI preferences and settings
CREATE TABLE ui_preferences (
    user_id TEXT PRIMARY KEY,
    theme TEXT DEFAULT 'dark',          -- 'dark' | 'light' | 'system'
    font_size INTEGER DEFAULT 14,
    auto_connect BOOLEAN DEFAULT 1,
    active_environment TEXT,            -- Current environment filter
    sidebar_width INTEGER DEFAULT 240,
    result_limit INTEGER DEFAULT 1000,
    enable_query_history_sync BOOLEAN DEFAULT 1,
    query_history_redaction BOOLEAN DEFAULT 0, -- Privacy option
    preferences_json TEXT,              -- Extensible JSON blob
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Sync conflict resolution log
CREATE TABLE sync_conflicts (
    conflict_id TEXT PRIMARY KEY,       -- UUID
    user_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,          -- 'tab' | 'connection' | 'saved_query'
    entity_id TEXT NOT NULL,
    client_version TEXT,                -- JSON: client state
    server_version TEXT,                -- JSON: server state
    resolution TEXT,                    -- 'client_wins' | 'server_wins' | 'merged'
    resolved_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_sync_conflicts_user ON sync_conflicts(user_id, resolved_at DESC);

-- Client device registration for sync
CREATE TABLE devices (
    device_id TEXT PRIMARY KEY,         -- UUID from client
    user_id TEXT NOT NULL,
    device_name TEXT,
    device_type TEXT,                   -- 'desktop' | 'web'
    os_type TEXT,                       -- 'darwin' | 'windows' | 'linux'
    last_sync_at DATETIME,
    last_sync_vector TEXT,              -- JSON: sync clock for devices
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_devices_user ON devices(user_id);
```

### 1.2 Team Tier Schema (Additional Tables)

```sql
-- Team/Organization management
CREATE TABLE organizations (
    org_id TEXT PRIMARY KEY,            -- UUID
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    owner_user_id TEXT NOT NULL,
    tier TEXT NOT NULL DEFAULT 'team',
    max_members INTEGER DEFAULT 10,
    settings_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (owner_user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_organizations_slug ON organizations(slug, deleted_at);

-- Organization membership
CREATE TABLE organization_members (
    member_id TEXT PRIMARY KEY,         -- UUID
    org_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member', -- 'owner' | 'admin' | 'member' | 'viewer'
    invited_by TEXT,
    invited_at DATETIME,
    joined_at DATETIME,
    status TEXT DEFAULT 'active',       -- 'active' | 'invited' | 'suspended'

    FOREIGN KEY (org_id) REFERENCES organizations(org_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    UNIQUE (org_id, user_id)
);

CREATE INDEX idx_org_members_org ON organization_members(org_id, status);
CREATE INDEX idx_org_members_user ON organization_members(user_id);

-- Shared connections (team-level)
CREATE TABLE shared_connections (
    shared_connection_id TEXT PRIMARY KEY, -- UUID
    org_id TEXT NOT NULL,
    name TEXT NOT NULL,
    db_type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT NOT NULL,
    username TEXT,
    ssl_mode TEXT,
    use_tunnel BOOLEAN DEFAULT 0,
    environments TEXT,
    parameters TEXT,
    created_by TEXT NOT NULL,
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (org_id) REFERENCES organizations(org_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_shared_connections_org ON shared_connections(org_id, deleted_at);

-- Connection permissions (RBAC)
CREATE TABLE connection_permissions (
    permission_id TEXT PRIMARY KEY,     -- UUID
    shared_connection_id TEXT,          -- Null for org-level permissions
    org_id TEXT NOT NULL,
    user_id TEXT,                       -- Null for role-based permissions
    role TEXT,                          -- Null for user-specific permissions
    can_read BOOLEAN DEFAULT 1,
    can_write BOOLEAN DEFAULT 0,
    can_execute BOOLEAN DEFAULT 1,
    can_share BOOLEAN DEFAULT 0,
    can_delete BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (shared_connection_id) REFERENCES shared_connections(shared_connection_id) ON DELETE CASCADE,
    FOREIGN KEY (org_id) REFERENCES organizations(org_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK ((user_id IS NOT NULL) OR (role IS NOT NULL))
);

CREATE INDEX idx_permissions_connection ON connection_permissions(shared_connection_id);
CREATE INDEX idx_permissions_user ON connection_permissions(org_id, user_id);
CREATE INDEX idx_permissions_role ON connection_permissions(org_id, role);

-- Shared saved queries (team library)
CREATE TABLE shared_queries (
    shared_query_id TEXT PRIMARY KEY,   -- UUID
    org_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    query_text TEXT NOT NULL,
    tags TEXT,
    created_by TEXT NOT NULL,
    is_template BOOLEAN DEFAULT 0,
    is_public BOOLEAN DEFAULT 0,        -- Public to all org members
    execution_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (org_id) REFERENCES organizations(org_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_shared_queries_org ON shared_queries(org_id, deleted_at, is_public);

-- Team audit log
CREATE TABLE audit_logs (
    log_id TEXT PRIMARY KEY,            -- UUID
    org_id TEXT NOT NULL,
    user_id TEXT,
    action TEXT NOT NULL,               -- 'create' | 'update' | 'delete' | 'execute'
    entity_type TEXT NOT NULL,          -- 'connection' | 'query' | 'member'
    entity_id TEXT,
    details TEXT,                       -- JSON: action details
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (org_id) REFERENCES organizations(org_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_audit_logs_org ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
```

### 1.3 Schema Versioning Strategy

```sql
-- Migration script example
INSERT INTO schema_migrations (version, description) VALUES
    (1, 'Initial schema - individual tier'),
    (2, 'Add team tier tables'),
    (3, 'Add query_history.query_hash index'),
    (4, 'Add ui_preferences.query_history_redaction');

-- Version check query for clients
SELECT MAX(version) as current_version FROM schema_migrations;
```

**Migration Approach:**
- Server-side migrations using Go migrate library
- Client checks schema version on connect
- Backward compatibility for 2 schema versions
- Graceful degradation if client is outdated

---

## 2. Sync Architecture

### 2.1 Sync Patterns

**Hybrid Push-Pull Strategy:**

1. **Real-time Push (WebSocket)**
   - Active tab content changes (debounced 2s)
   - Query execution events
   - New saved queries
   - AI message streaming

2. **Periodic Pull (HTTP)**
   - Full sync on app startup
   - Incremental sync every 30s for inactive tabs
   - Background sync every 5min for history/settings

3. **On-Demand Pull**
   - User-triggered "Sync Now"
   - After network reconnection
   - Before critical operations (delete, share)

### 2.2 Sync Algorithm Pseudocode

```typescript
// Client-side sync manager

interface SyncVector {
  deviceId: string
  lastSyncAt: string // ISO timestamp
  entityVersions: Record<string, number> // entity_id -> version
}

class TursoSyncManager {
  private syncVector: SyncVector
  private offlineQueue: PendingChange[]
  private wsConnection: WebSocket | null

  // Initialize sync on app startup
  async initializeSync(userId: string, deviceId: string) {
    // 1. Load local sync vector from localStorage
    this.syncVector = this.loadSyncVector()

    // 2. Register device if new
    if (!this.syncVector.deviceId) {
      await this.registerDevice(deviceId)
    }

    // 3. Perform full sync
    await this.performFullSync(userId)

    // 4. Establish WebSocket for real-time updates
    this.connectWebSocket(userId)

    // 5. Start periodic background sync
    this.startPeriodicSync()
  }

  // Full sync - pull all changes since last sync
  async performFullSync(userId: string) {
    const lastSyncAt = this.syncVector.lastSyncAt

    // Pull changes from server
    const serverChanges = await tursoClient.query(`
      SELECT entity_type, entity_id, action, data, updated_at
      FROM sync_changes
      WHERE user_id = ? AND updated_at > ?
      ORDER BY updated_at ASC
    `, [userId, lastSyncAt])

    // Apply server changes to local stores
    for (const change of serverChanges) {
      await this.applyServerChange(change)
    }

    // Push pending offline changes
    if (this.offlineQueue.length > 0) {
      await this.flushOfflineQueue()
    }

    // Update sync vector
    this.syncVector.lastSyncAt = new Date().toISOString()
    this.saveSyncVector()
  }

  // Incremental sync - delta changes only
  async performIncrementalSync(entities: string[]) {
    const changes = []

    for (const entityType of entities) {
      const localVersion = this.syncVector.entityVersions[entityType] || 0

      const deltas = await tursoClient.query(`
        SELECT * FROM ${entityType}
        WHERE user_id = ? AND version > ?
      `, [this.userId, localVersion])

      changes.push(...deltas)
    }

    return this.applyChanges(changes)
  }

  // Push local change to server
  async pushChange(change: LocalChange) {
    if (!navigator.onLine) {
      // Add to offline queue
      this.offlineQueue.push(change)
      return
    }

    try {
      // Optimistic update - update local immediately
      this.applyLocalChange(change)

      // Push to server
      const response = await tursoClient.execute(`
        INSERT INTO ${change.entityType} (...) VALUES (...)
        ON CONFLICT (${change.entityType}_id) DO UPDATE SET ...
        RETURNING version
      `, change.data)

      // Update sync vector with server version
      this.syncVector.entityVersions[change.entityId] = response.version

    } catch (error) {
      // Rollback local change
      this.rollbackLocalChange(change)

      // Check for conflict
      if (error.code === 'CONFLICT') {
        await this.resolveConflict(change)
      } else {
        // Add to retry queue
        this.offlineQueue.push(change)
      }
    }
  }

  // Conflict resolution: Last-Write-Wins by default
  async resolveConflict(localChange: LocalChange) {
    // Fetch server version
    const serverVersion = await this.fetchServerVersion(
      localChange.entityType,
      localChange.entityId
    )

    // Compare timestamps
    if (localChange.updatedAt > serverVersion.updatedAt) {
      // Local is newer - force push
      await this.forcePushChange(localChange)

      // Log conflict resolution
      await this.logConflict(localChange, serverVersion, 'client_wins')
    } else {
      // Server is newer - accept server version
      await this.applyServerChange(serverVersion)

      // Log conflict resolution
      await this.logConflict(localChange, serverVersion, 'server_wins')

      // Notify user of conflict
      this.notifyConflictResolution(localChange.entityType, 'server_wins')
    }
  }

  // WebSocket real-time sync
  connectWebSocket(userId: string) {
    this.wsConnection = new WebSocket(
      `wss://sync.sqlstudio.app/ws?userId=${userId}`
    )

    this.wsConnection.on('message', (data) => {
      const change = JSON.parse(data)

      // Apply real-time change from server
      this.applyServerChange(change)
    })

    this.wsConnection.on('close', () => {
      // Reconnect with exponential backoff
      setTimeout(() => this.connectWebSocket(userId), 5000)
    })
  }

  // Offline queue flush
  async flushOfflineQueue() {
    const queue = [...this.offlineQueue]
    this.offlineQueue = []

    for (const change of queue) {
      try {
        await this.pushChange(change)
      } catch (error) {
        // Re-queue failed changes
        this.offlineQueue.push(change)
      }
    }
  }

  // Periodic background sync
  startPeriodicSync() {
    setInterval(async () => {
      if (navigator.onLine) {
        await this.performIncrementalSync([
          'query_history',
          'ui_preferences',
          'ai_sessions'
        ])
      }
    }, 30000) // Every 30 seconds
  }
}
```

### 2.3 Conflict Resolution Strategies

**Default: Last-Write-Wins (LWW)**
- Compare `updated_at` timestamps
- Newer version always wins
- Log conflicts for audit

**Alternative: Three-Way Merge (for specific entities)**
- Detect concurrent edits
- Merge non-conflicting fields
- Prompt user for conflicting fields

**Example: Query Tab Merge**
```typescript
function mergeQueryTab(
  local: QueryTab,
  server: QueryTab,
  base: QueryTab | null
): QueryTab {
  // If base is null, cannot determine merge strategy - use LWW
  if (!base) {
    return local.updatedAt > server.updatedAt ? local : server
  }

  // Three-way merge
  const merged = { ...server }

  // Merge content if both modified
  if (local.content !== base.content && server.content !== base.content) {
    // Both modified - user must resolve
    merged.content = local.content // Or prompt user
    merged.isDirty = true
  } else if (local.content !== base.content) {
    // Only local modified
    merged.content = local.content
  }

  // Merge other fields independently
  if (local.title !== base.title) merged.title = local.title
  if (local.connectionId !== base.connectionId) merged.connectionId = local.connectionId

  return merged
}
```

---

## 3. Query Optimization

### 3.1 Query Patterns with Indexes

**Common Query: Get Recent Query History**
```sql
-- Query: Get last 100 queries for user
SELECT
  history_id,
  query_text,
  connection_id,
  execution_time_ms,
  executed_at
FROM query_history
WHERE user_id = ? AND is_redacted = 0
ORDER BY executed_at DESC
LIMIT 100;

-- Index used: idx_query_history_user (user_id, executed_at DESC)
-- Execution time: ~5ms for 10K rows
```

**Query: Search Query History by Text**
```sql
-- Enable FTS5 for full-text search
CREATE VIRTUAL TABLE query_history_fts USING fts5(
  query_text,
  content='query_history',
  content_rowid='rowid'
);

-- Trigger to keep FTS index in sync
CREATE TRIGGER query_history_fts_insert AFTER INSERT ON query_history BEGIN
  INSERT INTO query_history_fts(rowid, query_text)
  VALUES (new.rowid, new.query_text);
END;

-- Search query
SELECT
  h.history_id,
  h.query_text,
  h.executed_at,
  rank
FROM query_history h
JOIN query_history_fts fts ON h.rowid = fts.rowid
WHERE fts.query_text MATCH ?
  AND h.user_id = ?
ORDER BY rank, h.executed_at DESC
LIMIT 50;

-- Execution time: ~15ms for 10K rows
```

**Query: Get User's Active Tabs**
```sql
SELECT
  tab_id,
  title,
  content,
  connection_id,
  position
FROM query_tabs
WHERE user_id = ?
  AND deleted_at IS NULL
ORDER BY position ASC;

-- Index used: idx_query_tabs_user (user_id, deleted_at, position)
-- Execution time: ~2ms
```

**Query: Get Saved Queries with Tags**
```sql
-- Using JSON1 extension for tag filtering
SELECT
  saved_query_id,
  name,
  query_text,
  tags
FROM saved_queries
WHERE user_id = ?
  AND deleted_at IS NULL
  AND json_extract(tags, '$') LIKE '%' || ? || '%'
ORDER BY is_favorite DESC, updated_at DESC;

-- Execution time: ~10ms for 1K saved queries
```

### 3.2 Pagination Strategies

**Cursor-Based Pagination (Recommended)**
```sql
-- First page
SELECT * FROM query_history
WHERE user_id = ?
ORDER BY executed_at DESC
LIMIT 100;

-- Next page
SELECT * FROM query_history
WHERE user_id = ?
  AND executed_at < ?  -- Cursor from last row
ORDER BY executed_at DESC
LIMIT 100;
```

**Offset-Based Pagination (Avoid for large datasets)**
```sql
-- Works but slow for large offsets
SELECT * FROM query_history
WHERE user_id = ?
ORDER BY executed_at DESC
LIMIT 100 OFFSET 5000;  -- Scans 5100 rows
```

### 3.3 Bulk Operations

**Batch Insert Saved Queries**
```sql
-- Use single INSERT with multiple VALUES
INSERT INTO saved_queries (
  saved_query_id, user_id, name, query_text, created_at
) VALUES
  (?, ?, ?, ?, ?),
  (?, ?, ?, ?, ?),
  (?, ?, ?, ?, ?);

-- 50x faster than individual INSERTs
```

**Bulk Delete Old History**
```sql
-- Soft delete history older than 90 days
UPDATE query_history
SET deleted_at = CURRENT_TIMESTAMP
WHERE user_id = ?
  AND executed_at < datetime('now', '-90 days')
  AND deleted_at IS NULL;

-- Hard delete (run periodically)
DELETE FROM query_history
WHERE deleted_at < datetime('now', '-30 days');
```

---

## 4. Data Isolation & Security

### 4.1 Row-Level Security (Individual Tier)

**Approach: Application-Level RLS**
- Every query includes `WHERE user_id = ?`
- Enforced in application layer (Go backend)
- Turso doesn't have native RLS - app-level is sufficient

**Example: Safe Query Pattern**
```go
// Backend Go code
func GetUserQueryHistory(ctx context.Context, userId string, limit int) ([]QueryHistory, error) {
    // Always include user_id filter
    query := `
        SELECT history_id, query_text, executed_at
        FROM query_history
        WHERE user_id = ? AND deleted_at IS NULL
        ORDER BY executed_at DESC
        LIMIT ?
    `

    rows, err := db.QueryContext(ctx, query, userId, limit)
    // ... handle results
}
```

### 4.2 Row-Level Security (Team Tier)

**Multi-Tenancy Strategy: Shared Database with Org Isolation**

```sql
-- All team tables include org_id
-- Queries always filter by org_id AND check permissions

-- Example: Get shared connections for user
SELECT sc.*
FROM shared_connections sc
JOIN organization_members om ON sc.org_id = om.org_id
JOIN connection_permissions cp ON (
    sc.shared_connection_id = cp.shared_connection_id
    AND (
        (cp.user_id = ? AND om.user_id = ?)
        OR (cp.role = om.role)
    )
)
WHERE om.user_id = ?
  AND om.status = 'active'
  AND sc.deleted_at IS NULL
  AND cp.can_read = 1;
```

**Permission Check Function**
```go
func HasConnectionPermission(
    ctx context.Context,
    userId, connectionId, permission string,
) (bool, error) {
    query := `
        SELECT 1
        FROM connection_permissions cp
        JOIN organization_members om ON cp.org_id = om.org_id
        WHERE cp.shared_connection_id = ?
          AND om.user_id = ?
          AND om.status = 'active'
          AND (
              (cp.user_id = ? AND cp.can_read = 1)
              OR (cp.role = om.role AND cp.can_read = 1)
          )
        LIMIT 1
    `

    var exists int
    err := db.QueryRowContext(ctx, query, connectionId, userId, userId).Scan(&exists)
    return exists == 1, err
}
```

### 4.3 Tenant Isolation Strategies

**Option 1: Shared Database (Recommended)**
- Single Turso database per tier
- All tables include `user_id` or `org_id`
- Application enforces isolation
- Most cost-effective
- Simpler schema migrations

**Option 2: Database-per-Tenant (Not Recommended)**
- Separate Turso database per user/org
- Complete isolation
- Higher complexity
- Higher Turso costs ($29/database)
- Difficult to manage at scale

**Recommendation: Shared Database**
- Use for both individual and team tiers
- Individual tier: filter by `user_id`
- Team tier: filter by `org_id` + RBAC checks

---

## 5. Turso-Specific Features

### 5.1 Edge Replication Benefits

**Turso Edge Architecture:**
- Primary database in closest region (e.g., us-west-2)
- Read replicas at edge locations
- Writes go to primary, reads from nearest replica
- Eventual consistency (typically <100ms)

**Howlerops Benefits:**
- Query history reads from edge (fast)
- Tab content reads from edge (fast)
- Writes to primary (slightly slower, acceptable)
- Global users get low-latency reads

**Configuration:**
```bash
# Create Turso database with replicas
turso db create sql-studio-sync \
  --location iad \
  --enable-replicas

# Add replicas in other regions
turso db replicate sql-studio-sync add lax
turso db replicate sql-studio-sync add lhr
turso db replicate sql-studio-sync add syd
```

### 5.2 libSQL Extensions Usage

**JSON1 Extension (Built-in)**
```sql
-- Query saved queries by tag
SELECT * FROM saved_queries
WHERE json_extract(tags, '$[0]') = 'optimization';

-- Extract specific preference
SELECT json_extract(preferences_json, '$.theme') as theme
FROM ui_preferences
WHERE user_id = ?;
```

**FTS5 Extension (Full-Text Search)**
```sql
-- Already covered in section 3.1
-- Use for query history search
```

**Possible: Vector Extension (Future)**
```sql
-- If Turso adds vector support in future
-- Could enable semantic query search

CREATE TABLE query_embeddings (
  query_id TEXT PRIMARY KEY,
  embedding BLOB,  -- Vector representation
  FOREIGN KEY (query_id) REFERENCES query_history(history_id)
);

-- Semantic search
SELECT query_text, vector_distance(embedding, ?) as distance
FROM query_embeddings
ORDER BY distance ASC
LIMIT 10;
```

### 5.3 Embedded Replica Strategy

**Local Replica for Offline Support:**

Turso supports embedded replicas - a local SQLite file that syncs with remote.

```typescript
// Client-side embedded replica setup
import { createClient } from '@libsql/client'

const client = createClient({
  url: 'file:///Users/jacob/Library/Application Support/sql-studio/sync.db',
  syncUrl: 'libsql://sql-studio-sync-username.turso.io',
  authToken: process.env.TURSO_AUTH_TOKEN,
  syncInterval: 60, // Sync every 60 seconds
})

// Writes go to local + queued for remote
await client.execute(`
  INSERT INTO query_history (...) VALUES (...)
`)

// Reads from local (instant)
const rows = await client.execute(`
  SELECT * FROM query_history WHERE user_id = ? LIMIT 100
`)
```

**Benefits:**
- Full offline support
- Instant reads from local database
- Background sync when online
- Conflict resolution handled by Turso

**Tradeoffs:**
- Larger app size (~5MB for libSQL)
- Desktop-only (Wails already includes SQLite)
- More complex client setup

**Recommendation:** Use embedded replica for Howlerops desktop app.

### 5.4 Primary/Replica Sync Patterns

**Write Path:**
```
Client -> Primary DB -> Replicas (async)
         ^-- Writes acknowledged immediately
         |
         +-- Background replication to edges (<100ms)
```

**Read Path:**
```
Client -> Nearest Replica (edge) -> Return data
         ^-- Sub-10ms latency
         |
         +-- May be slightly stale (eventual consistency)
```

**Handling Consistency:**
```typescript
// For critical reads after write, read from primary
await client.execute('INSERT INTO saved_queries (...) VALUES (...)')

// Read from primary to ensure consistency
const result = await client.execute(
  'SELECT * FROM saved_queries WHERE saved_query_id = ?',
  [savedQueryId],
  { readYourWrites: true } // Turso option (if available)
)

// Alternative: Wait for replication
await new Promise(resolve => setTimeout(resolve, 200))
const result = await client.execute('SELECT ...')
```

---

## 6. Performance Considerations

### 6.1 Network Overhead

**Minimizing Network Roundtrips:**

1. **Batch Queries**
   ```typescript
   // Bad: Multiple roundtrips
   const tabs = await turso.query('SELECT * FROM query_tabs WHERE user_id = ?', [userId])
   const history = await turso.query('SELECT * FROM query_history WHERE user_id = ?', [userId])

   // Good: Single batch request
   const results = await turso.batch([
     { sql: 'SELECT * FROM query_tabs WHERE user_id = ?', args: [userId] },
     { sql: 'SELECT * FROM query_history WHERE user_id = ? LIMIT 100', args: [userId] },
   ])
   ```

2. **Compress Large Payloads**
   ```typescript
   // Compress query content before storing
   import pako from 'pako'

   const compressed = pako.gzip(queryContent)
   await turso.execute('INSERT INTO query_tabs (..., content) VALUES (..., ?)', [compressed])

   // Decompress on read
   const row = await turso.execute('SELECT content FROM query_tabs WHERE tab_id = ?', [tabId])
   const content = pako.ungzip(row[0].content, { to: 'string' })
   ```

3. **Use WebSocket for Real-Time Sync**
   - Avoid polling for updates
   - Server pushes changes to connected clients
   - Reduces network traffic by 80%

### 6.2 Sync Impact on UI Responsiveness

**Strategies to Avoid UI Blocking:**

1. **Debouncing Writes**
   ```typescript
   // Debounce tab content sync
   const debouncedSyncTab = debounce(async (tabId: string, content: string) => {
     await syncManager.pushChange({
       entityType: 'query_tabs',
       entityId: tabId,
       data: { content },
     })
   }, 2000) // 2 second debounce

   // On every keystroke
   editor.on('change', (content) => {
     debouncedSyncTab(activeTab.id, content)
   })
   ```

2. **Optimistic Updates**
   ```typescript
   // Update local state immediately
   queryStore.updateTab(tabId, { content: newContent })

   // Sync to server in background
   syncManager.pushChange({ ... }).catch(error => {
     // Rollback on error
     queryStore.updateTab(tabId, { content: oldContent })
     toast.error('Sync failed')
   })
   ```

3. **Background Sync Workers**
   ```typescript
   // Use Web Worker for heavy sync operations
   const syncWorker = new Worker('/sync-worker.js')

   syncWorker.postMessage({
     action: 'syncHistory',
     userId: currentUser.id,
   })

   syncWorker.onmessage = (event) => {
     if (event.data.type === 'syncComplete') {
       queryStore.mergeHistory(event.data.history)
     }
   }
   ```

### 6.3 Background Sync Workers

**Go Backend Sync Service:**
```go
// Separate goroutine for background sync
func StartBackgroundSync(ctx context.Context, userId string) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Sync non-critical data
            go syncQueryHistory(ctx, userId)
            go syncAISessions(ctx, userId)

        case <-ctx.Done():
            return
        }
    }
}
```

### 6.4 Compression Strategies

**Compress Large Fields:**

1. **Query Content (typically small, skip compression)**
2. **AI Session Messages (compress)**
   ```sql
   -- Store compressed content
   CREATE TABLE ai_messages (
     message_id TEXT PRIMARY KEY,
     content BLOB,  -- Compressed with gzip
     content_encoding TEXT DEFAULT 'gzip',
     -- ...
   );
   ```

3. **Query History (conditionally compress)**
   ```go
   func StoreQueryHistory(query string) error {
       var content []byte
       var encoding string

       if len(query) > 1024 {
           // Compress large queries
           content = gzipCompress(query)
           encoding = "gzip"
       } else {
           content = []byte(query)
           encoding = "plain"
       }

       return db.Exec(`
           INSERT INTO query_history (query_text, encoding)
           VALUES (?, ?)
       `, content, encoding)
   }
   ```

### 6.5 Batch Operations

**Batch Insert Pattern:**
```typescript
// Accumulate changes for batch
const batchQueue: Change[] = []

function queueChange(change: Change) {
  batchQueue.push(change)

  if (batchQueue.length >= 50) {
    flushBatch()
  }
}

async function flushBatch() {
  if (batchQueue.length === 0) return

  const batch = [...batchQueue]
  batchQueue.length = 0

  // Single batch request
  await turso.batch(
    batch.map(change => ({
      sql: `INSERT INTO ${change.entityType} (...) VALUES (...)`,
      args: change.data,
    }))
  )
}

// Flush on interval
setInterval(flushBatch, 5000)
```

---

## 7. Cost Optimization

### 7.1 Turso Pricing Model

**Turso Pricing (as of 2024):**
- **Free Tier:** 9 GB total storage, 500K rows written/month
- **Scaler Plan:** $29/month
  - 25 GB storage included
  - 500M rows written/month
  - $0.01 per additional GB
  - $0.00001 per row written

**Key Metric: Rows Written**
- Reads are free
- Writes are metered
- Updates count as writes
- Deletes count as writes

### 7.2 Minimize Row Writes

**Strategies:**

1. **Batch Writes**
   - Accumulate changes, flush every 5s
   - Reduces 100 writes to 1 batch write
   - Saves 99% of write costs

2. **Debounce Frequent Updates**
   - Don't sync on every keystroke
   - Debounce tab content sync (2s)
   - Reduces writes by 95%

3. **Skip Redundant Writes**
   ```typescript
   // Check if data actually changed
   function shouldSync(local: QueryTab, remote: QueryTab): boolean {
     return local.updatedAt > remote.updatedAt &&
            local.content !== remote.content
   }
   ```

4. **Compress Data**
   - Smaller payloads = fewer bytes written
   - Not directly billed, but improves performance

5. **Soft Deletes**
   - Use `deleted_at` instead of `DELETE`
   - Same write cost, but preserves audit trail
   - Periodic hard delete (cleanup job)

6. **Selective Sync**
   - Allow users to disable history sync
   - Sync only tabs, not history
   - Reduces writes by 80% for heavy users

### 7.3 Storage Efficiency

**JSON vs Normalized:**

**Option 1: Store preferences as JSON**
```sql
-- Single row per user
CREATE TABLE ui_preferences (
  user_id TEXT PRIMARY KEY,
  preferences_json TEXT  -- All prefs in one JSON blob
);

-- Pros: 1 write to update any preference
-- Cons: Full JSON rewrite on any change
```

**Option 2: Normalize preferences**
```sql
-- Multiple rows per user
CREATE TABLE user_preferences (
  preference_id TEXT PRIMARY KEY,
  user_id TEXT,
  key TEXT,
  value TEXT
);

-- Pros: Granular updates
-- Cons: More writes to update multiple prefs
```

**Recommendation: Hybrid**
- Use JSON for rarely-changed settings
- Use normalized for frequently-changed settings

```sql
CREATE TABLE ui_preferences (
  user_id TEXT PRIMARY KEY,
  theme TEXT,                    -- Frequently changed
  font_size INTEGER,             -- Frequently changed
  other_prefs TEXT               -- JSON blob for rare settings
);
```

### 7.4 Read vs Write Patterns

**Optimize for Read-Heavy Workloads:**

Howlerops is read-heavy:
- 90% reads (query history, tabs, saved queries)
- 10% writes (new queries, tab updates)

**Strategy:**
- Cache aggressively on client
- Use embedded replica for reads
- Only write to Turso on significant changes

### 7.5 Retention Policies

**Data Lifecycle Management:**

1. **Query History**
   - Keep 90 days in Turso
   - Archive older queries locally
   - Reduces storage by 70%

2. **AI Messages**
   - Keep only last 20 messages per session in Turso
   - Older messages archived/deleted
   - Reduces storage by 90%

3. **Audit Logs (Team Tier)**
   - Keep 1 year in Turso
   - Export older logs to S3
   - Reduces storage by 50%

**Cleanup Job (Weekly):**
```sql
-- Soft delete old history
UPDATE query_history
SET deleted_at = CURRENT_TIMESTAMP
WHERE user_id = ?
  AND executed_at < datetime('now', '-90 days')
  AND deleted_at IS NULL;

-- Hard delete after 30 day grace period
DELETE FROM query_history
WHERE deleted_at < datetime('now', '-30 days');

-- Trim AI sessions
DELETE FROM ai_messages
WHERE session_id IN (
  SELECT session_id FROM ai_sessions
  WHERE updated_at < datetime('now', '-180 days')
);
```

### 7.6 Cost Estimation

**Individual Tier (Single User):**

Assumptions:
- 50 query executions/day
- 10 tab updates/day
- 5 AI messages/day
- 30 days/month

```
Query history writes:  50/day × 30 days = 1,500 rows
Tab updates:          10/day × 30 days = 300 rows
AI messages:          5/day × 30 days = 150 rows
Settings updates:     2/day × 30 days = 60 rows
---------------------------------------------------------
Total:                                   ~2,000 rows/month
```

**Cost:** Free tier (500K rows/month limit)

**Team Tier (10 Users):**

```
Per-user writes:      2,000 rows/month
Team members:         10 users
---------------------------------------------------------
Total:                20,000 rows/month
```

**Cost:** Free tier (well under 500K limit)

**Heavy User (Individual):**

Assumptions:
- 500 query executions/day (power user)
- 100 tab updates/day
- 20 AI messages/day

```
Query history:        500 × 30 = 15,000 rows
Tab updates:          100 × 30 = 3,000 rows
AI messages:          20 × 30 = 600 rows
---------------------------------------------------------
Total:                         ~18,600 rows/month
```

**Cost:** Free tier

**Scaling to 1,000 Users (SaaS):**

```
Average user writes:  2,000 rows/month
Users:                1,000
---------------------------------------------------------
Total:                2,000,000 rows/month
```

**Cost:** Scaler Plan ($29/month, covers 500M rows)

**Conclusion:** Even at scale, Howlerops stays within free or basic tier.

---

## 8. Authentication & Authorization

### 8.1 User Authentication Flow

**Recommended: OAuth2 + JWT**

```typescript
// 1. User logs in via OAuth provider (GitHub, Google, etc.)
const { user, accessToken } = await authProvider.login()

// 2. Exchange access token for Howlerops JWT
const response = await fetch('https://api.sqlstudio.app/auth/token', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${accessToken}` },
  body: JSON.stringify({ provider: 'github' }),
})

const { jwtToken, tursoToken } = await response.json()

// 3. Store tokens securely
await secureStorage.set('jwt', jwtToken)
await secureStorage.set('turso_token', tursoToken)

// 4. Use Turso token for database access
const tursoClient = createClient({
  url: 'libsql://sql-studio-sync.turso.io',
  authToken: tursoToken,
})
```

### 8.2 Turso Database Tokens

**Token Types:**

1. **User-Scoped Token (Individual Tier)**
   - Token scoped to single user
   - All queries filtered by `user_id`
   - Generated on login

2. **Org-Scoped Token (Team Tier)**
   - Token scoped to organization
   - Access to shared resources
   - RBAC enforced in app layer

**Token Generation (Backend):**
```go
func GenerateTursoToken(userId string, tier string) (string, error) {
    // Create Turso JWT with claims
    claims := jwt.MapClaims{
        "sub": userId,
        "tier": tier,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(tursoSecret)
}
```

**Token Refresh:**
- JWT expires after 24 hours
- Refresh silently in background
- Fallback to offline mode if refresh fails

### 8.3 Team Member Permissions

**RBAC Roles:**

| Role     | Permissions                                    |
|----------|------------------------------------------------|
| Owner    | Full access, manage members, billing          |
| Admin    | Manage connections, queries, members          |
| Member   | Read/write own queries, read shared resources |
| Viewer   | Read-only access to shared resources          |

**Permission Check Middleware:**
```go
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userId := c.GetString("user_id")
        orgId := c.Param("org_id")

        hasPermission, err := CheckOrgPermission(userId, orgId, permission)
        if err != nil || !hasPermission {
            c.JSON(403, gin.H{"error": "Forbidden"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// Usage
router.POST("/api/orgs/:org_id/connections",
    RequirePermission("create_connection"),
    CreateSharedConnection)
```

### 8.4 API Key Management

**API Keys for Turso Access:**

```sql
CREATE TABLE api_keys (
    key_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key_hash TEXT NOT NULL,          -- SHA256 hash of key
    name TEXT,
    scopes TEXT,                     -- JSON: ["read:history", "write:queries"]
    expires_at DATETIME,
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id, revoked_at);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

**API Key Usage:**
```bash
# User-generated API key for CLI tools
export SQL_STUDIO_API_KEY="sskey_abc123..."

# CLI tool uses API key
sql-studio sync pull --api-key $SQL_STUDIO_API_KEY
```

---

## 9. Data Privacy

### 9.1 What NOT to Sync

**Never Sync:**
1. Database passwords
2. SSH private keys
3. API secrets
4. Personal identifiable info in queries (optional redaction)

**Stored Locally Only:**
```typescript
// Passwords stored in sessionStorage (desktop app)
// or OS keychain (Wails integration)
import { crypto } from '@wailsjs/go/models'

// Store password securely
await crypto.StorePassword(connectionId, password)

// Connection synced WITHOUT password
const connectionToSync = {
  ...connection,
  password: undefined,
  sshTunnel: {
    ...connection.sshTunnel,
    password: undefined,
    privateKey: undefined,
  }
}
```

### 9.2 Encryption at Rest/In Transit

**In Transit:**
- All Turso connections use TLS 1.3
- WebSocket connections use WSS
- No additional encryption needed

**At Rest:**
- Turso encrypts data at rest by default (AES-256)
- No additional encryption needed
- For extra paranoia: client-side encryption

**Optional: Client-Side Encryption**
```typescript
// Encrypt sensitive query text before syncing
import { encryptAES256 } from './crypto'

const encryptedQuery = await encryptAES256(
  queryText,
  userEncryptionKey  // Derived from user's passphrase
)

await turso.execute(`
  INSERT INTO query_history (query_text, is_encrypted)
  VALUES (?, 1)
`, [encryptedQuery])
```

### 9.3 User Opt-Out Options

**Privacy Settings:**

```typescript
interface PrivacySettings {
  enableSync: boolean                 // Global sync toggle
  syncQueryHistory: boolean           // Sync query history
  syncAIConversations: boolean        // Sync AI chats
  redactSensitiveQueries: boolean     // Auto-redact queries with keywords
  sensitiveKeywords: string[]         // ['password', 'secret', 'token']
}

// User can disable sync entirely
if (!privacySettings.enableSync) {
  // Use local-only storage (IndexedDB)
  return localStorageAdapter
}

// Or selectively sync
if (!privacySettings.syncQueryHistory) {
  // Skip query history sync
  syncManager.skipEntity('query_history')
}
```

**Query Redaction:**
```typescript
function redactQuery(query: string, keywords: string[]): string {
  let redacted = query

  for (const keyword of keywords) {
    const regex = new RegExp(`(${keyword}\\s*=\\s*)['"]([^'"]+)['"]`, 'gi')
    redacted = redacted.replace(regex, `$1'[REDACTED]'`)
  }

  return redacted
}

// Example
const original = "UPDATE users SET password = 'secret123' WHERE id = 1"
const redacted = redactQuery(original, ['password'])
// Result: "UPDATE users SET password = '[REDACTED]' WHERE id = 1"
```

### 9.4 GDPR Compliance

**User Data Export:**
```typescript
// Export all user data
async function exportUserData(userId: string): Promise<UserDataExport> {
  const [connections, tabs, history, savedQueries, aiSessions] = await Promise.all([
    turso.query('SELECT * FROM connections WHERE user_id = ?', [userId]),
    turso.query('SELECT * FROM query_tabs WHERE user_id = ?', [userId]),
    turso.query('SELECT * FROM query_history WHERE user_id = ?', [userId]),
    turso.query('SELECT * FROM saved_queries WHERE user_id = ?', [userId]),
    turso.query('SELECT * FROM ai_sessions WHERE user_id = ?', [userId]),
  ])

  return {
    connections: connections.rows,
    tabs: tabs.rows,
    history: history.rows,
    savedQueries: savedQueries.rows,
    aiSessions: aiSessions.rows,
    exportedAt: new Date().toISOString(),
  }
}
```

**User Data Deletion:**
```sql
-- Soft delete user (30-day grace period)
UPDATE users
SET deleted_at = CURRENT_TIMESTAMP
WHERE user_id = ?;

-- Hard delete (after grace period)
DELETE FROM users WHERE user_id = ?;
-- Cascades to all related tables via foreign keys
```

**GDPR Data Processing Agreement:**
- Turso is GDPR-compliant
- Howlerops acts as data controller
- Users control their data (export, delete)
- No data sold to third parties
- Audit trail via `audit_logs` table

---

## 10. Migration & Operations

### 10.1 Bulk Import from Local Storage

**Migration Script (Client-Side):**

```typescript
async function migrateToTurso(userId: string, tursoClient: TursoClient) {
  // 1. Load local data from localStorage/IndexedDB
  const localConnections = await localDB.getAllConnections()
  const localTabs = await localDB.getAllTabs()
  const localHistory = await localDB.getQueryHistory()
  const localSavedQueries = await localDB.getSavedQueries()

  console.log('Starting migration...', {
    connections: localConnections.length,
    tabs: localTabs.length,
    history: localHistory.length,
    savedQueries: localSavedQueries.length,
  })

  // 2. Batch insert connections
  await tursoClient.batch(
    localConnections.map(conn => ({
      sql: `
        INSERT INTO connections (
          connection_id, user_id, name, db_type, host, port, database_name, username, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (connection_id) DO NOTHING
      `,
      args: [
        conn.id,
        userId,
        conn.name,
        conn.type,
        conn.host,
        conn.port,
        conn.database,
        conn.username,
        conn.createdAt || new Date().toISOString(),
      ],
    }))
  )

  // 3. Batch insert tabs
  await tursoClient.batch(
    localTabs.map(tab => ({
      sql: `
        INSERT INTO query_tabs (
          tab_id, user_id, title, tab_type, content, connection_id, position, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (tab_id) DO NOTHING
      `,
      args: [
        tab.id,
        userId,
        tab.title,
        tab.type,
        tab.content,
        tab.connectionId,
        tab.position || 0,
        tab.createdAt || new Date().toISOString(),
      ],
    }))
  )

  // 4. Batch insert history (limit to last 1000)
  const recentHistory = localHistory
    .sort((a, b) => b.executedAt - a.executedAt)
    .slice(0, 1000)

  await tursoClient.batch(
    recentHistory.map(h => ({
      sql: `
        INSERT INTO query_history (
          history_id, user_id, connection_id, query_text, query_hash,
          row_count, execution_time_ms, executed_at, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (history_id) DO NOTHING
      `,
      args: [
        crypto.randomUUID(),
        userId,
        h.connectionId,
        h.query,
        sha256(h.query),
        h.rowCount,
        h.executionTime,
        h.executedAt,
        h.executedAt,
      ],
    }))
  )

  // 5. Clear local data after successful migration
  await localDB.clear()

  console.log('Migration completed successfully!')
}
```

### 10.2 Export for Backup

**Export Functionality:**

```typescript
// Export all user data to JSON file
async function exportBackup(userId: string): Promise<void> {
  const data = await exportUserData(userId)

  const blob = new Blob([JSON.stringify(data, null, 2)], {
    type: 'application/json',
  })

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sql-studio-backup-${Date.now()}.json`
  a.click()

  URL.revokeObjectURL(url)
}
```

**Import Backup:**

```typescript
// Import data from backup JSON file
async function importBackup(file: File, userId: string): Promise<void> {
  const text = await file.text()
  const backup: UserDataExport = JSON.parse(text)

  // Validate backup format
  if (!backup.connections || !backup.tabs) {
    throw new Error('Invalid backup format')
  }

  // Re-import using migration logic
  await migrateToTurso(userId, tursoClient)
}
```

### 10.3 Account Deletion Data Cleanup

**Soft Delete (30-day grace period):**

```sql
-- Mark user and all data as deleted
BEGIN TRANSACTION;

UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ?;
UPDATE connections SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ?;
UPDATE query_tabs SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ?;
UPDATE saved_queries SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ?;
UPDATE ai_sessions SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ?;

COMMIT;
```

**Hard Delete (after grace period):**

```sql
-- Permanent deletion (run by scheduled job)
BEGIN TRANSACTION;

DELETE FROM connections WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');
DELETE FROM query_tabs WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');
DELETE FROM query_history WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');
DELETE FROM saved_queries WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');
DELETE FROM ai_sessions WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');
DELETE FROM users WHERE user_id = ? AND deleted_at < datetime('now', '-30 days');

COMMIT;
```

**Cleanup Job (Go):**

```go
// Scheduled job (runs daily)
func CleanupDeletedData(ctx context.Context) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Delete data older than 30 days
    tables := []string{
        "connections",
        "query_tabs",
        "query_history",
        "saved_queries",
        "ai_sessions",
        "ai_messages",
        "users",
    }

    for _, table := range tables {
        query := fmt.Sprintf(`
            DELETE FROM %s
            WHERE deleted_at IS NOT NULL
              AND deleted_at < datetime('now', '-30 days')
        `, table)

        result, err := tx.ExecContext(ctx, query)
        if err != nil {
            return err
        }

        rowsDeleted, _ := result.RowsAffected()
        log.Printf("Deleted %d rows from %s", rowsDeleted, table)
    }

    return tx.Commit()
}
```

---

## 11. Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)

**Goals:**
- Set up Turso database
- Implement basic schema (individual tier)
- Build sync manager infrastructure

**Tasks:**
1. Create Turso database with replicas
2. Implement schema migrations
3. Build Go backend API for sync
4. Create client-side sync manager
5. Implement basic push/pull sync
6. Add offline queue

**Deliverables:**
- Working Turso database
- Basic sync for connections and tabs
- Offline support

### Phase 2: Core Sync Features (Weeks 3-4)

**Goals:**
- Implement all sync entities
- Add conflict resolution
- Real-time sync via WebSocket

**Tasks:**
1. Sync query history
2. Sync saved queries
3. Sync AI sessions (basic)
4. Implement conflict resolution
5. Add WebSocket real-time sync
6. Build sync UI (sync status, conflicts)

**Deliverables:**
- Complete sync for individual tier
- Real-time updates
- Conflict resolution UI

### Phase 3: Performance & Privacy (Weeks 5-6)

**Goals:**
- Optimize sync performance
- Add privacy features
- Implement data retention

**Tasks:**
1. Add query redaction
2. Implement privacy settings
3. Add compression for large fields
4. Batch write optimizations
5. Implement retention policies
6. Add data export/import

**Deliverables:**
- Privacy-compliant sync
- Optimized for cost
- Backup/restore functionality

### Phase 4: Team Tier (Weeks 7-9)

**Goals:**
- Implement team tier schema
- Add RBAC
- Shared resources

**Tasks:**
1. Implement org management
2. Add member permissions
3. Sync shared connections
4. Sync shared queries
5. Build team management UI
6. Add audit logging

**Deliverables:**
- Working team tier
- RBAC implementation
- Team collaboration features

### Phase 5: Polish & Launch (Weeks 10-12)

**Goals:**
- Production readiness
- Monitoring & observability
- Documentation

**Tasks:**
1. Add monitoring (Sentry, Prometheus)
2. Implement rate limiting
3. Build admin dashboard
4. Write user documentation
5. Beta testing with users
6. Production deployment

**Deliverables:**
- Production-ready sync
- Monitored and observable
- User documentation

---

## Performance Benchmarks

### Expected Performance

| Operation                    | Latency | Notes                          |
|------------------------------|---------|--------------------------------|
| Full sync (startup)          | <2s     | 100 tabs, 1K history           |
| Incremental sync             | <200ms  | Delta changes only             |
| Real-time update (WebSocket) | <50ms   | Push from server to client     |
| Query history search         | <100ms  | FTS5 index, 10K queries        |
| Tab content sync             | <500ms  | Debounced, background          |
| Offline queue flush          | <1s     | 100 queued changes             |
| Conflict resolution          | <100ms  | Automatic LWW                  |
| Export backup                | <5s     | Full user data export          |

### Scalability Targets

| Metric                       | Target  | Notes                          |
|------------------------------|---------|--------------------------------|
| Users per database           | 100K+   | Shared database, app-level RLS |
| Query history per user       | 10K     | 90-day retention               |
| Tabs per user                | 100     | Reasonable limit               |
| Saved queries per user       | 1K      | Personal library               |
| Team members per org         | 50      | Team tier limit                |
| Concurrent sync connections  | 10K     | WebSocket connections          |

---

## Cost Analysis Summary

### Individual Tier

**Turso Free Tier:**
- Storage: Up to 9 GB
- Rows written: 500K/month
- Reads: Unlimited

**Howlerops Usage:**
- Average user: 2K rows/month
- Heavy user: 20K rows/month
- Storage: <10 MB/user

**Conclusion:** Individual users stay on free tier indefinitely.

### Team Tier (10 Users)

**Turso Free Tier:**
- Usage: 20K rows/month (well under 500K limit)
- Storage: <100 MB total

**Conclusion:** Small teams stay on free tier.

### SaaS (1000 Users)

**Turso Scaler Plan: $29/month**
- Storage: 25 GB included (plenty)
- Rows written: 2M/month (well under 500M limit)

**Conclusion:** Even at scale, Turso costs are minimal ($29/month for 1000 users).

---

## Security Model Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Howlerops Client                        │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Zustand    │  │ Sync Manager │  │ Turso Client │         │
│  │    Stores    │←→│  (debounce)  │←→│  (libSQL)    │         │
│  └──────────────┘  └──────────────┘  └──────┬───────┘         │
│         ↕                                    │                 │
│  ┌──────────────┐                           │ TLS 1.3         │
│  │   Local DB   │                           │                 │
│  │  (IndexedDB) │                           │                 │
│  └──────────────┘                           │                 │
└─────────────────────────────────────────────┼─────────────────┘
                                              │
                                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Turso Cloud (libSQL)                       │
│                                                                 │
│  ┌──────────────┐                                              │
│  │   Primary    │ ← Writes go here                             │
│  │   Database   │                                              │
│  └──────┬───────┘                                              │
│         │                                                       │
│         ├─────→ Edge Replica (us-west-2)                       │
│         ├─────→ Edge Replica (us-east-1)                       │
│         └─────→ Edge Replica (eu-west-1)                       │
│                                                                 │
│  [Encryption at rest: AES-256]                                 │
└─────────────────────────────────────────────────────────────────┘
                                              ↑
                                              │ Row-level filtering
                                              │
┌─────────────────────────────────────────────┼─────────────────┐
│                     Backend API (Go)        │                 │
│                                             │                 │
│  ┌──────────────┐  ┌──────────────┐  ┌─────┴──────┐          │
│  │     Auth     │→│  Middleware  │→│  Turso API   │          │
│  │   (JWT)      │  │  (user_id)   │  │  (filtered)  │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                 │
│  [OAuth2 + JWT + Turso tokens]                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Conclusion

This comprehensive design provides Howlerops with:

1. **Scalable sync infrastructure** using Turso's libSQL
2. **Cost-effective solution** (free for most users, $29/month at scale)
3. **Privacy-first approach** (no credentials synced, optional query redaction)
4. **Offline-first architecture** (full offline support with sync when online)
5. **Team collaboration** (RBAC, shared resources, audit logging)
6. **Production-ready** (conflict resolution, monitoring, GDPR compliance)

The design leverages Turso's strengths:
- Edge replication for low-latency reads
- libSQL compatibility (SQLite syntax)
- Embedded replicas for offline support
- Simple pricing model (rows written)

Next steps:
1. Review and approve this design
2. Begin Phase 1 implementation (foundation)
3. Set up development Turso database
4. Build sync manager prototype

---

**Document Version:** 1.0
**Last Updated:** 2025-10-23
**Author:** Howlerops Team
**Status:** Draft for Review
