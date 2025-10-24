# SQL Studio - Turso Sync Protocol Specification

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Sync Protocol Algorithm](#sync-protocol-algorithm)
4. [Conflict Resolution](#conflict-resolution)
5. [Delta Sync Strategy](#delta-sync-strategy)
6. [Offline Queue Management](#offline-queue-management)
7. [Initial Sync (First Upload)](#initial-sync-first-upload)
8. [Security & Validation](#security--validation)
9. [Error Handling & Recovery](#error-handling--recovery)
10. [Performance Optimization](#performance-optimization)
11. [Implementation Details](#implementation-details)

## Overview

### Goal
Enable seamless bidirectional sync between IndexedDB (local) and Turso (cloud) for Individual tier users, supporting multi-device access and offline-first operation.

### Key Principles
- **Offline-First**: All operations work locally first, sync in background
- **Eventual Consistency**: Multi-device conflicts resolved deterministically
- **Never Lose Data**: Conflicts preserved, never silently overwritten
- **Security**: Zero credentials in cloud, all data sanitized
- **Performance**: Delta sync only changed records, batch operations
- **Privacy**: User controls sync per-entity (private queries excluded)

### Sync Scope
**Synced Entities:**
- User Preferences (UI settings)
- Connection Templates (metadata only, NO passwords)
- Query History (sanitized queries)
- Saved Queries (user's library)
- AI Memory Sessions (metadata)
- AI Memory Messages (full conversations)

**NOT Synced:**
- Passwords/credentials (sessionStorage only)
- Large query results (export files)
- Transient UI state (scroll position, hover)

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser Tab                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐         ┌──────────────┐                │
│  │   Zustand    │────────▶│  IndexedDB   │                │
│  │   Stores     │◀────────│   8 Stores   │                │
│  └──────────────┘         └──────┬───────┘                │
│        │                          │                         │
│        │                          │                         │
│        ▼                          ▼                         │
│  ┌──────────────────────────────────────┐                 │
│  │       Sync Manager                    │                 │
│  │  - Change Detection                   │                 │
│  │  - Conflict Resolution                │                 │
│  │  - Batch Processing                   │                 │
│  │  - Offline Queue                      │                 │
│  └──────────────┬───────────────────────┘                 │
│                 │                                           │
└─────────────────┼───────────────────────────────────────────┘
                  │
                  │ HTTPS (TLS 1.3)
                  │ Turso Client SDK
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    Turso Cloud (LibSQL)                     │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Replica    │  │   Replica    │  │   Replica    │    │
│  │   Primary    │──│   Region A   │──│   Region B   │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
Local Change → Sanitize → Batch → Upload → Turso
                                           │
                                           ▼
                                    Sync Metadata Update
                                           │
                                           ▼
                                    Broadcast to Tabs
```

```
Turso Change → Download Batch → Compare Version → Merge
                                                    │
                                                    ▼
                                            Apply to IndexedDB
                                                    │
                                                    ▼
                                            Broadcast to Tabs
```

## Sync Protocol Algorithm

### 1. Upload Flow (Local → Turso)

```typescript
/**
 * Upload local changes to Turso
 * Runs every 30 seconds when online, or on demand
 */
async function uploadChanges(): Promise<SyncResult> {
  // Step 1: Detect changes since last sync
  const lastSync = await getLastSyncTimestamp()
  const changes = await detectChanges(lastSync)

  if (changes.length === 0) {
    return { success: true, uploaded: 0 }
  }

  // Step 2: Group changes by entity type
  const batches = groupByEntityType(changes)

  // Step 3: Process each batch
  const results: SyncResult[] = []

  for (const batch of batches) {
    // Step 4: Sanitize data (remove credentials)
    const sanitized = await sanitizeBatch(batch)

    // Step 5: Validate no credentials leaked
    validateNoCredentials(sanitized)

    // Step 6: Upload to Turso with retry
    try {
      const result = await uploadBatchWithRetry(sanitized, {
        maxRetries: 3,
        timeout: 5000,
        batchSize: 50
      })

      results.push(result)

      // Step 7: Update sync metadata on success
      await updateSyncMetadata(batch, {
        synced: true,
        syncedAt: new Date(),
        syncVersion: result.newVersion
      })

    } catch (error) {
      // Step 8: Handle upload failure
      if (isConflict(error)) {
        // Conflict detected, trigger conflict resolution
        await handleConflict(batch, error)
      } else if (isRetryable(error)) {
        // Queue for retry
        await queueForRetry(batch, error)
      } else {
        // Permanent failure, log error
        await logSyncError(batch, error)
      }
    }
  }

  return aggregateResults(results)
}
```

#### Change Detection Algorithm

```typescript
/**
 * Detect changes in IndexedDB since last sync
 * Uses updated_at timestamp for efficient delta detection
 */
async function detectChanges(since: Date): Promise<ChangeSet[]> {
  const changes: ChangeSet[] = []

  // Query each store for changes
  const stores = [
    'connections',
    'query_history',
    'saved_queries',
    'ai_sessions',
    'ai_messages',
    'ui_preferences'
  ]

  for (const storeName of stores) {
    // Get records updated since last sync
    const records = await db.getAll(storeName, {
      index: 'updated_at',
      range: IDBKeyRange.lowerBound(since, true),
      limit: 50 // Process in batches
    })

    // Filter out records that shouldn't sync
    const syncable = records.filter(record => {
      // Exclude private queries
      if (record.privacy_mode === 'private') {
        return false
      }

      // Exclude device-specific preferences
      if (storeName === 'ui_preferences' && record.device_id) {
        return false
      }

      return true
    })

    // Mark change type (create, update, delete)
    for (const record of syncable) {
      changes.push({
        entityType: storeName,
        entityId: getEntityId(record),
        operation: detectOperation(record),
        data: record,
        timestamp: record.updated_at
      })
    }
  }

  // Sort by timestamp (oldest first for FIFO)
  changes.sort((a, b) => a.timestamp - b.timestamp)

  return changes
}
```

### 2. Download Flow (Turso → Local)

```typescript
/**
 * Download changes from Turso
 * Runs every 60 seconds when online, or on tab focus
 */
async function downloadChanges(): Promise<SyncResult> {
  // Step 1: Get last sync timestamp
  const lastSync = await getLastSyncTimestamp()
  const deviceId = await getDeviceId()

  // Step 2: Query Turso for changes
  // Parallel queries for each entity type
  const [
    preferences,
    connections,
    queries,
    savedQueries,
    aiSessions,
    aiMessages
  ] = await Promise.all([
    turso.query(`
      SELECT * FROM user_preferences
      WHERE user_id = ?
        AND updated_at > ?
        AND deleted_at IS NULL
      ORDER BY updated_at ASC
      LIMIT 50
    `, [userId, lastSync]),

    turso.query(`
      SELECT * FROM connection_templates
      WHERE user_id = ?
        AND updated_at > ?
        AND deleted_at IS NULL
      ORDER BY updated_at ASC
      LIMIT 50
    `, [userId, lastSync]),

    // ... similar queries for other entities
  ])

  // Step 3: Merge all changes
  const allChanges = [
    ...preferences.rows,
    ...connections.rows,
    ...queries.rows,
    ...savedQueries.rows,
    ...aiSessions.rows,
    ...aiMessages.rows
  ]

  if (allChanges.length === 0) {
    return { success: true, downloaded: 0 }
  }

  // Step 4: Process each change
  const results: ApplyResult[] = []

  for (const change of allChanges) {
    try {
      // Step 5: Check for conflicts
      const conflict = await detectConflict(change)

      if (conflict) {
        // Step 6: Resolve conflict
        const resolved = await resolveConflict(conflict, change)
        results.push(await applyChange(resolved))
      } else {
        // Step 7: Apply change directly
        results.push(await applyChange(change))
      }

    } catch (error) {
      // Step 8: Handle apply failure
      await logSyncError(change, error)
      results.push({ success: false, error })
    }
  }

  // Step 9: Update last sync timestamp
  await setLastSyncTimestamp(new Date())

  // Step 10: Broadcast changes to other tabs
  await broadcastSyncComplete()

  return aggregateResults(results)
}
```

#### Conflict Detection Algorithm

```typescript
/**
 * Detect if remote change conflicts with local state
 * Uses sync_version for optimistic locking
 */
async function detectConflict(
  remoteChange: EntityRecord
): Promise<Conflict | null> {
  const { entity_type, entity_id } = remoteChange

  // Get local record
  const local = await getLocalRecord(entity_type, entity_id)

  if (!local) {
    // No local record, no conflict
    return null
  }

  // Check if local was modified since last sync
  const lastSync = await getLastSyncTimestamp()

  if (local.updated_at <= lastSync) {
    // Local not modified, no conflict
    return null
  }

  // Both local and remote modified since last sync
  // This is a conflict
  return {
    entityType: entity_type,
    entityId: entity_id,
    localVersion: local.sync_version,
    remoteVersion: remoteChange.sync_version,
    localData: local,
    remoteData: remoteChange,
    detectedAt: new Date()
  }
}
```

## Conflict Resolution

### Strategy: Vector Clock with Last-Write-Wins (LWW)

```typescript
/**
 * Resolve conflict between local and remote versions
 * Default: Last-Write-Wins based on updated_at timestamp
 * Critical data: Prompt user for manual resolution
 */
async function resolveConflict(
  conflict: Conflict,
  remoteChange: EntityRecord
): Promise<EntityRecord> {
  const { localData, remoteData } = conflict

  // Step 1: Determine resolution strategy
  const strategy = getConflictStrategy(conflict.entityType)

  switch (strategy) {
    case 'last-write-wins':
      // Compare timestamps
      if (remoteData.updated_at > localData.updated_at) {
        // Remote is newer, accept remote
        return remoteData
      } else if (remoteData.updated_at < localData.updated_at) {
        // Local is newer, keep local (will upload on next sync)
        return localData
      } else {
        // Same timestamp, use device_id as tie-breaker
        return remoteData.device_id > localData.device_id
          ? remoteData
          : localData
      }

    case 'manual':
      // Critical data, prompt user
      return await promptUserForResolution(conflict)

    case 'merge':
      // Merge both versions
      return await mergeVersions(localData, remoteData)

    case 'keep-both':
      // Create duplicate with suffix
      const duplicate = {
        ...remoteData,
        id: generateId(),
        name: `${remoteData.name} (conflicted copy)`
      }
      await saveLocalRecord(duplicate)
      return localData

    default:
      // Default to last-write-wins
      return remoteData.updated_at > localData.updated_at
        ? remoteData
        : localData
  }
}
```

### Conflict Resolution Matrix

| Entity Type | Strategy | Reasoning |
|-------------|----------|-----------|
| User Preferences | last-write-wins | UI settings, low risk |
| Connection Templates | keep-both | Critical, user should decide |
| Query History | last-write-wins | Audit trail, remote is source of truth |
| Saved Queries | keep-both | Critical, don't lose work |
| AI Sessions | merge | Can combine messages |
| AI Messages | last-write-wins | Immutable, timestamp ordered |

### Conflict Resolution Examples

#### Example 1: Connection Template Conflict

```
Scenario:
- User edits connection "Production DB" on Device A (Laptop)
- User edits same connection on Device B (Desktop)
- Both devices sync to Turso

Resolution:
1. Detect conflict (both updated since last sync)
2. Strategy: keep-both
3. Create duplicate: "Production DB (conflicted copy - Desktop)"
4. Show notification: "Connection conflict detected, created copy"
5. User manually merges or deletes duplicate
```

#### Example 2: Query History Conflict

```
Scenario:
- User runs query on Device A at 10:00 AM
- User runs query on Device B at 10:05 AM
- Both devices sync

Resolution:
1. No conflict (different queries, different IDs)
2. Both queries added to history
3. Sorted by executed_at timestamp
```

#### Example 3: Preference Conflict

```
Scenario:
- User changes theme to "dark" on Device A at 10:00 AM
- User changes theme to "light" on Device B at 10:05 AM
- Device A syncs at 10:10 AM
- Device B syncs at 10:15 AM

Resolution:
1. Device A uploads: theme = "dark", updated_at = 10:00 AM
2. Device B detects conflict
3. Strategy: last-write-wins
4. Device B wins (10:05 AM > 10:00 AM)
5. Device A downloads: theme = "light"
6. Result: Both devices now have "light" theme
```

### Archive Resolved Conflicts

```typescript
/**
 * Archive conflict after resolution for audit
 */
async function archiveConflict(
  conflict: Conflict,
  resolution: EntityRecord,
  strategy: ConflictStrategy
): Promise<void> {
  await turso.execute(`
    INSERT INTO sync_conflicts_archive (
      id,
      entity_type,
      entity_id,
      user_id,
      local_version,
      remote_version,
      local_data,
      remote_data,
      resolution_strategy,
      resolved_data,
      resolved_by,
      resolved_at,
      conflict_detected_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `, [
    generateId(),
    conflict.entityType,
    conflict.entityId,
    userId,
    conflict.localVersion,
    conflict.remoteVersion,
    JSON.stringify(conflict.localData),
    JSON.stringify(conflict.remoteData),
    strategy,
    JSON.stringify(resolution),
    deviceId,
    new Date(),
    conflict.detectedAt
  ])
}
```

## Delta Sync Strategy

### Efficient Change Detection

Only sync records that changed since last sync using indexed queries:

```sql
-- Get changed preferences (upload)
SELECT * FROM user_preferences
WHERE updated_at > :last_sync_timestamp
  AND (deleted_at IS NULL OR deleted_at > :last_sync_timestamp)
ORDER BY updated_at ASC
LIMIT 50;

-- Get changed connections (upload)
SELECT * FROM connection_templates
WHERE updated_at > :last_sync_timestamp
  AND (deleted_at IS NULL OR deleted_at > :last_sync_timestamp)
ORDER BY updated_at ASC
LIMIT 50;

-- Get changed queries (download from Turso)
SELECT * FROM query_history
WHERE user_id = :user_id
  AND updated_at > :last_sync_timestamp
  AND deleted_at IS NULL
ORDER BY executed_at DESC
LIMIT 50;
```

### Batch Size Strategy

```typescript
const BATCH_SIZES = {
  // Small records (preferences, connections)
  small: 50,

  // Medium records (query history, saved queries)
  medium: 25,

  // Large records (AI messages with full content)
  large: 10,

  // Maximum bytes per batch (100 KB)
  maxBytes: 100 * 1024
}

function getBatchSize(entityType: EntityType): number {
  switch (entityType) {
    case 'preference':
    case 'connection':
      return BATCH_SIZES.small

    case 'query_history':
    case 'saved_query':
      return BATCH_SIZES.medium

    case 'ai_message':
      return BATCH_SIZES.large

    default:
      return BATCH_SIZES.medium
  }
}
```

### Pagination for Large Syncs

```typescript
/**
 * Paginate through large change sets
 * Prevents memory issues and timeout
 */
async function uploadChangesWithPagination(): Promise<SyncResult> {
  const lastSync = await getLastSyncTimestamp()
  let offset = 0
  let totalUploaded = 0
  let hasMore = true

  while (hasMore) {
    // Get next batch
    const changes = await detectChanges(lastSync, {
      offset,
      limit: 50
    })

    if (changes.length === 0) {
      hasMore = false
      break
    }

    // Upload batch
    const result = await uploadBatch(changes)
    totalUploaded += result.uploaded

    // Update offset
    offset += changes.length

    // Check if more batches exist
    hasMore = changes.length === 50

    // Yield to prevent blocking UI
    await sleep(100)
  }

  return {
    success: true,
    uploaded: totalUploaded
  }
}
```

## Offline Queue Management

### Queue Architecture

```typescript
interface SyncQueueItem {
  id: string
  entityType: EntityType
  entityId: string
  operation: 'create' | 'update' | 'delete'
  payload: Record<string, unknown>
  timestamp: Date
  retryCount: number
  lastError?: string
  priority: 'low' | 'medium' | 'high'
}
```

### Queue Processing

```typescript
/**
 * Process offline queue when connection restored
 */
async function processOfflineQueue(): Promise<void> {
  // Step 1: Check if online
  if (!navigator.onLine) {
    return
  }

  // Step 2: Get queued items (FIFO, prioritized)
  const queue = await db.getAll('sync_queue', {
    index: 'timestamp',
    direction: 'next'
  })

  // Sort by priority
  const prioritized = queue.sort((a, b) => {
    const priorityOrder = { high: 0, medium: 1, low: 2 }
    return priorityOrder[a.priority] - priorityOrder[b.priority]
  })

  // Step 3: Process each item
  for (const item of prioritized) {
    try {
      // Attempt upload
      await uploadQueueItem(item)

      // Success - remove from queue
      await db.delete('sync_queue', item.id)

    } catch (error) {
      // Increment retry count
      item.retryCount++
      item.lastError = error.message

      if (item.retryCount >= 3) {
        // Max retries exceeded, mark as failed
        await db.put('sync_queue', {
          ...item,
          lastError: `Max retries exceeded: ${error.message}`
        })

        // Notify user
        showNotification({
          type: 'error',
          message: `Failed to sync ${item.entityType}: ${error.message}`
        })
      } else {
        // Update retry count
        await db.put('sync_queue', item)
      }
    }
  }
}
```

### Online/Offline Event Handling

```typescript
/**
 * Set up online/offline event listeners
 */
function setupNetworkListeners(): void {
  window.addEventListener('online', async () => {
    console.log('[Sync] Network online, processing queue')

    // Process offline queue
    await processOfflineQueue()

    // Resume regular sync
    startSyncInterval()
  })

  window.addEventListener('offline', () => {
    console.log('[Sync] Network offline, pausing sync')

    // Stop sync interval
    stopSyncInterval()

    // Show offline indicator
    showOfflineIndicator()
  })
}
```

## Initial Sync (First Upload)

### First-Time Sync Flow

```typescript
/**
 * Initial sync when user first upgrades to Individual tier
 * Uploads all existing local data to Turso
 */
async function performInitialSync(): Promise<InitialSyncResult> {
  // Step 1: Count local records
  const counts = await countLocalRecords()
  const totalRecords = Object.values(counts).reduce((a, b) => a + b, 0)

  // Step 2: Show progress modal
  showInitialSyncModal({
    totalRecords,
    message: 'Syncing your data to cloud...'
  })

  // Step 3: Upload each entity type in order
  const results: SyncResult[] = []

  try {
    // Order matters: dependencies first
    results.push(await uploadAllConnections(
      progress => updateProgress('connections', progress)
    ))

    results.push(await uploadAllPreferences(
      progress => updateProgress('preferences', progress)
    ))

    results.push(await uploadAllSavedQueries(
      progress => updateProgress('saved_queries', progress)
    ))

    results.push(await uploadAllQueryHistory(
      progress => updateProgress('query_history', progress)
    ))

    results.push(await uploadAllAISessions(
      progress => updateProgress('ai_sessions', progress)
    ))

    results.push(await uploadAllAIMessages(
      progress => updateProgress('ai_messages', progress)
    ))

    // Step 4: Mark initial sync complete
    await setInitialSyncComplete()

    // Step 5: Show success message
    showSuccessNotification({
      message: `Successfully synced ${totalRecords} items to cloud!`
    })

    return {
      success: true,
      totalRecords,
      results
    }

  } catch (error) {
    // Step 6: Handle failure
    showErrorNotification({
      message: `Initial sync failed: ${error.message}`,
      action: 'Retry',
      onAction: () => performInitialSync()
    })

    return {
      success: false,
      error: error.message
    }
  } finally {
    // Step 7: Close modal
    closeInitialSyncModal()
  }
}
```

### Upload All Connections

```typescript
/**
 * Upload all connections with progress tracking
 */
async function uploadAllConnections(
  onProgress: (progress: Progress) => void
): Promise<SyncResult> {
  const connections = await db.getAll('connections')
  const total = connections.length
  let uploaded = 0

  for (const connection of connections) {
    // Sanitize (remove password)
    const sanitized = sanitizeConnection(connection)

    // Upload to Turso
    await turso.execute(`
      INSERT INTO connection_templates (
        connection_id, user_id, name, type, host, port,
        database, username, ssl_mode, parameters,
        environment_tags, last_used_at, created_at,
        updated_at, sync_version, synced_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(connection_id) DO UPDATE SET
        name = excluded.name,
        updated_at = excluded.updated_at,
        sync_version = sync_version + 1
    `, [
      connection.connection_id,
      connection.user_id,
      connection.name,
      connection.type,
      connection.host,
      connection.port,
      connection.database,
      connection.username,
      connection.ssl_mode,
      JSON.stringify(connection.parameters || {}),
      JSON.stringify(connection.environment_tags || []),
      connection.last_used_at.toISOString(),
      connection.created_at.toISOString(),
      connection.updated_at.toISOString(),
      1, // initial sync_version
      new Date().toISOString()
    ])

    // Update local sync metadata
    await db.put('connections', {
      ...connection,
      synced: true,
      sync_version: 1
    })

    // Report progress
    uploaded++
    onProgress({
      current: uploaded,
      total,
      percentage: Math.round((uploaded / total) * 100)
    })
  }

  return { success: true, uploaded }
}
```

## Security & Validation

### Data Sanitization

```typescript
/**
 * Sanitize connection before upload
 * REMOVES ALL CREDENTIALS
 */
function sanitizeConnection(connection: ConnectionRecord): ConnectionRecord {
  // Create copy
  const sanitized = { ...connection }

  // Remove password fields
  delete sanitized.password
  delete sanitized.sshTunnel?.password
  delete sanitized.sshTunnel?.privateKey

  // Remove session data
  delete sanitized.sessionId
  delete sanitized.activeConnection

  // Sanitize parameters
  if (sanitized.parameters) {
    const params = { ...sanitized.parameters }

    // Remove known credential keys
    const credentialKeys = [
      'password',
      'passwd',
      'pwd',
      'secret',
      'token',
      'api_key',
      'apiKey',
      'private_key',
      'privateKey',
      'certificate',
      'key',
      'passphrase'
    ]

    for (const key of credentialKeys) {
      delete params[key]
    }

    sanitized.parameters = params
  }

  return sanitized
}

/**
 * Sanitize query text before upload
 * REMOVES INLINE CREDENTIALS
 */
function sanitizeQuery(queryText: string): string {
  // Remove inline passwords
  let sanitized = queryText

  // Pattern: PASSWORD 'secret'
  sanitized = sanitized.replace(
    /PASSWORD\s+['"][^'"]+['"]/gi,
    "PASSWORD '[REDACTED]'"
  )

  // Pattern: IDENTIFIED BY 'secret'
  sanitized = sanitized.replace(
    /IDENTIFIED\s+BY\s+['"][^'"]+['"]/gi,
    "IDENTIFIED BY '[REDACTED]'"
  )

  // Pattern: user:pass@host
  sanitized = sanitized.replace(
    /([a-zA-Z0-9_]+):([^@\s]+)@/g,
    '$1:[REDACTED]@'
  )

  return sanitized
}
```

### Validation Before Upload

```typescript
/**
 * Validate data is safe to sync
 * Throws error if credentials detected
 */
function validateSyncSafety(data: unknown): void {
  const json = JSON.stringify(data)

  // Patterns that indicate credentials
  const credentialPatterns = [
    /password["\s:]+[^,}\]]+/i,
    /passwd["\s:]+[^,}\]]+/i,
    /secret["\s:]+[^,}\]]+/i,
    /api[-_]?key["\s:]+[^,}\]]+/i,
    /private[-_]?key["\s:]+[^,}\]]+/i,
    /bearer\s+[a-zA-Z0-9_\-\.]+/i,
    /authorization["\s:]+[^,}\]]+/i
  ]

  for (const pattern of credentialPatterns) {
    if (pattern.test(json)) {
      throw new Error(
        `SECURITY: Credential detected in sync data. ` +
        `Pattern: ${pattern.source}. ` +
        `Sync aborted.`
      )
    }
  }
}
```

## Error Handling & Recovery

### Error Categories

```typescript
enum SyncErrorType {
  // Network errors (retryable)
  NETWORK_ERROR = 'network_error',
  TIMEOUT = 'timeout',
  CONNECTION_REFUSED = 'connection_refused',

  // Conflict errors (resolvable)
  VERSION_CONFLICT = 'version_conflict',
  CONCURRENT_UPDATE = 'concurrent_update',

  // Data errors (not retryable)
  VALIDATION_ERROR = 'validation_error',
  CREDENTIAL_LEAKED = 'credential_leaked',
  QUOTA_EXCEEDED = 'quota_exceeded',

  // Auth errors (requires re-auth)
  UNAUTHORIZED = 'unauthorized',
  TOKEN_EXPIRED = 'token_expired',
  FORBIDDEN = 'forbidden',

  // Server errors (retryable with backoff)
  SERVER_ERROR = 'server_error',
  SERVICE_UNAVAILABLE = 'service_unavailable'
}
```

### Retry Strategy

```typescript
/**
 * Retry with exponential backoff
 */
async function uploadBatchWithRetry(
  batch: ChangeSet[],
  options: RetryOptions
): Promise<SyncResult> {
  const { maxRetries = 3, timeout = 5000 } = options

  let lastError: Error

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      // Attempt upload with timeout
      return await withTimeout(
        uploadBatch(batch),
        timeout
      )

    } catch (error) {
      lastError = error

      // Check if retryable
      if (!isRetryableError(error)) {
        throw error
      }

      // Last attempt failed
      if (attempt === maxRetries) {
        throw new Error(
          `Upload failed after ${maxRetries} retries: ${error.message}`
        )
      }

      // Calculate backoff delay: 100ms * 2^attempt
      const delay = 100 * Math.pow(2, attempt)

      console.log(
        `[Sync] Upload attempt ${attempt + 1} failed, ` +
        `retrying in ${delay}ms...`
      )

      await sleep(delay)
    }
  }

  throw lastError!
}

function isRetryableError(error: Error): boolean {
  const retryableTypes = [
    SyncErrorType.NETWORK_ERROR,
    SyncErrorType.TIMEOUT,
    SyncErrorType.CONNECTION_REFUSED,
    SyncErrorType.SERVER_ERROR,
    SyncErrorType.SERVICE_UNAVAILABLE
  ]

  return retryableTypes.includes(error.type)
}
```

## Performance Optimization

### Expected Performance Targets

| Operation | Target Latency | Target Throughput |
|-----------|----------------|-------------------|
| Upload 50 connections | < 500ms | 100 records/sec |
| Upload 50 queries | < 300ms | 150 records/sec |
| Download 50 changes | < 400ms | 125 records/sec |
| Conflict detection | < 50ms | 1000 records/sec |
| Initial sync (1000 items) | < 10s | 100 records/sec |

### Optimization Techniques

#### 1. Parallel Batch Processing

```typescript
/**
 * Upload multiple entity types in parallel
 */
async function uploadChangesParallel(): Promise<SyncResult> {
  const lastSync = await getLastSyncTimestamp()

  // Detect changes for each entity type
  const [
    preferenceChanges,
    connectionChanges,
    queryChanges,
    savedQueryChanges
  ] = await Promise.all([
    detectChanges('preferences', lastSync),
    detectChanges('connections', lastSync),
    detectChanges('query_history', lastSync),
    detectChanges('saved_queries', lastSync)
  ])

  // Upload all batches in parallel
  const results = await Promise.all([
    uploadBatch(preferenceChanges),
    uploadBatch(connectionChanges),
    uploadBatch(queryChanges),
    uploadBatch(savedQueryChanges)
  ])

  return aggregateResults(results)
}
```

#### 2. Connection Pooling

```typescript
/**
 * Maintain connection pool for Turso
 */
class TursoConnectionPool {
  private pool: TursoClient[] = []
  private maxConnections = 5

  async getConnection(): Promise<TursoClient> {
    // Reuse existing connection
    if (this.pool.length > 0) {
      return this.pool.pop()!
    }

    // Create new connection
    return createClient({
      url: process.env.TURSO_URL,
      authToken: await getAuthToken()
    })
  }

  releaseConnection(client: TursoClient): void {
    if (this.pool.length < this.maxConnections) {
      this.pool.push(client)
    } else {
      client.close()
    }
  }
}
```

#### 3. Prepared Statements

```typescript
/**
 * Use prepared statements for better performance
 */
const PREPARED_STATEMENTS = {
  upsertConnection: `
    INSERT INTO connection_templates (
      connection_id, user_id, name, type, host, port,
      database, username, ssl_mode, parameters,
      environment_tags, last_used_at, created_at,
      updated_at, sync_version, synced_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(connection_id) DO UPDATE SET
      name = excluded.name,
      updated_at = excluded.updated_at,
      sync_version = sync_version + 1
  `,

  upsertQuery: `
    INSERT INTO query_history (...)
    VALUES (...)
    ON CONFLICT(id) DO UPDATE SET ...
  `
}

// Prepare once, execute many times
const stmt = await turso.prepare(PREPARED_STATEMENTS.upsertConnection)
for (const connection of connections) {
  await stmt.execute([...params])
}
```

#### 4. Debounced Sync

```typescript
/**
 * Debounce sync to avoid excessive requests
 */
const debouncedSync = debounce(async () => {
  await uploadChanges()
}, 5000) // Wait 5s after last change

// Trigger sync on change
db.on('change', () => {
  debouncedSync()
})
```

## Implementation Details

### Turso Client Setup

```typescript
/**
 * Initialize Turso client
 */
import { createClient } from '@libsql/client'

const tursoClient = createClient({
  url: process.env.TURSO_DATABASE_URL!,
  authToken: await getUserAuthToken(),

  // Connection options
  fetch: globalThis.fetch,

  // Performance options
  intMode: 'number',

  // Sync options (for embedded replica - future)
  syncUrl: process.env.TURSO_SYNC_URL,
  syncInterval: 60 // seconds
})

export { tursoClient as turso }
```

### Sync Manager Class

```typescript
/**
 * Central sync manager
 */
export class SyncManager {
  private syncInterval: NodeJS.Timeout | null = null
  private isSyncing = false

  async start(): Promise<void> {
    // Initial sync on start
    await this.performSync()

    // Set up periodic sync
    this.syncInterval = setInterval(() => {
      this.performSync()
    }, 30000) // Every 30 seconds

    // Set up network listeners
    setupNetworkListeners()
  }

  async stop(): Promise<void> {
    if (this.syncInterval) {
      clearInterval(this.syncInterval)
      this.syncInterval = null
    }
  }

  private async performSync(): Promise<void> {
    if (this.isSyncing || !navigator.onLine) {
      return
    }

    this.isSyncing = true

    try {
      // Upload local changes
      await uploadChanges()

      // Download remote changes
      await downloadChanges()

      // Process offline queue
      await processOfflineQueue()

    } catch (error) {
      console.error('[Sync] Error:', error)
    } finally {
      this.isSyncing = false
    }
  }

  async forceSyncNow(): Promise<void> {
    await this.performSync()
  }
}

// Singleton instance
export const syncManager = new SyncManager()
```

### React Hook for Sync Status

```typescript
/**
 * Hook for sync status in React components
 */
export function useSyncStatus() {
  const [status, setStatus] = useState<SyncStatus>({
    isSyncing: false,
    lastSynced: null,
    pendingCount: 0,
    errorCount: 0
  })

  useEffect(() => {
    // Subscribe to sync events
    const unsubscribe = syncManager.on('statusChange', setStatus)
    return unsubscribe
  }, [])

  return {
    ...status,
    forceSyncNow: () => syncManager.forceSyncNow()
  }
}
```

### Usage Example

```typescript
/**
 * Use sync in React component
 */
export function SyncStatusIndicator() {
  const { isSyncing, lastSynced, pendingCount, forceSyncNow } = useSyncStatus()

  return (
    <div className="sync-indicator">
      {isSyncing && <Spinner />}

      {lastSynced && (
        <span>Last synced: {formatRelativeTime(lastSynced)}</span>
      )}

      {pendingCount > 0 && (
        <span>{pendingCount} pending changes</span>
      )}

      <button onClick={forceSyncNow}>
        Sync Now
      </button>
    </div>
  )
}
```

---

## Summary

This sync protocol provides:

1. **Bidirectional Sync**: Local ↔ Turso with conflict resolution
2. **Delta Sync**: Only changed records, efficient bandwidth
3. **Offline Support**: Queue changes, sync when online
4. **Conflict Resolution**: Multiple strategies, never lose data
5. **Security**: Zero credentials in cloud, strict validation
6. **Performance**: Parallel batches, prepared statements, connection pooling
7. **Reliability**: Retry with backoff, error handling, recovery

**Next Steps:**
1. Implement SyncManager class
2. Add Turso client integration
3. Create conflict resolution UI
4. Add sync status indicator
5. Test multi-device scenarios
6. Monitor performance metrics
