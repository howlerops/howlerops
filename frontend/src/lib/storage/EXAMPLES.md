# IndexedDB Storage - Usage Examples

Real-world examples demonstrating all features of the storage infrastructure.

## Table of Contents
- [Basic Operations](#basic-operations)
- [Query History](#query-history)
- [Connection Management](#connection-management)
- [Preferences](#preferences)
- [Sync Queue](#sync-queue)
- [Advanced Patterns](#advanced-patterns)

## Basic Operations

### Initialize Storage

```typescript
import { initializeStorage, getStorageInfo } from '@/lib/storage'

// On app startup
async function init() {
  try {
    await initializeStorage()

    const info = await getStorageInfo()
    console.log(`Storage: ${info.percentage}% used`)
  } catch (error) {
    console.error('Storage init failed:', error)
  }
}
```

### Check Storage Support

```typescript
import { IndexedDBClient } from '@/lib/storage'

if (!IndexedDBClient.isSupported()) {
  alert('This browser does not support IndexedDB')
}
```

## Query History

### Save Query Execution

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

async function executeAndSaveQuery(query: string, connectionId: string) {
  const repo = getQueryHistoryRepository()
  const startTime = Date.now()

  try {
    // Execute query
    const result = await database.execute(query)
    const executionTime = Date.now() - startTime

    // Save to history
    await repo.create({
      user_id: getCurrentUserId(),
      query_text: query,
      connection_id: connectionId,
      execution_time_ms: executionTime,
      row_count: result.rowCount,
      privacy_mode: 'normal',
    })

    return result
  } catch (error) {
    // Save failed query
    await repo.create({
      user_id: getCurrentUserId(),
      query_text: query,
      connection_id: connectionId,
      execution_time_ms: Date.now() - startTime,
      row_count: 0,
      error: error.message,
      privacy_mode: 'normal',
    })

    throw error
  }
}
```

### Search Query History

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

const repo = getQueryHistoryRepository()

// Recent queries
const recent = await repo.getRecent('user-123', 20)

// Search by text
const searchResults = await repo.search({
  userId: 'user-123',
  searchText: 'SELECT',
  limit: 50,
})

// Get slow queries
const slowQueries = await repo.getSlowQueries(1000, 'user-123')

// Get failed queries
const failed = await repo.getFailedQueries('user-123')

// Search by date range
const thisMonth = await repo.search({
  userId: 'user-123',
  startDate: new Date('2024-10-01'),
  endDate: new Date('2024-10-31'),
})

// Search by connection
const connQueries = await repo.search({
  userId: 'user-123',
  connectionId: 'conn-456',
  limit: 100,
})
```

### Query Analytics

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

const repo = getQueryHistoryRepository()

// Get overall statistics
const stats = await repo.getStatistics('user-123')
console.log(`
  Total queries: ${stats.totalQueries}
  Average time: ${stats.averageExecutionTime}ms
  Slowest query: ${stats.slowestQueryTime}ms
  Success rate: ${(stats.successRate * 100).toFixed(1)}%
  Total rows: ${stats.totalRows}
`)

// Get statistics for specific connection
const connStats = await repo.getStatistics('user-123', {
  connectionId: 'conn-456',
})

// Get statistics for date range
const monthStats = await repo.getStatistics('user-123', {
  startDate: new Date('2024-10-01'),
  endDate: new Date('2024-10-31'),
})
```

### Cleanup Old Queries

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

const repo = getQueryHistoryRepository()

// Delete queries older than 30 days
const thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
const deleted = await repo.deleteOlderThan(thirtyDaysAgo)
console.log(`Deleted ${deleted} old queries`)

// Clear all history for a user
const cleared = await repo.clearUserHistory('user-123')
console.log(`Cleared ${cleared} queries`)
```

## Connection Management

### Create Connection

```typescript
import { getConnectionRepository } from '@/lib/storage'

const repo = getConnectionRepository()

const connection = await repo.create({
  user_id: 'user-123',
  name: 'Production Database',
  type: 'postgres',
  host: 'db.example.com',
  port: 5432,
  database: 'myapp',
  username: 'dbuser',
  ssl_mode: 'require',
  environment_tags: ['production', 'primary'],
})

console.log('Created connection:', connection.connection_id)
```

### Load Connections

```typescript
import { getConnectionRepository } from '@/lib/storage'

const repo = getConnectionRepository()

// All connections for user
const allConnections = await repo.getAllForUser('user-123')

// Recently used connections
const recent = await repo.getRecentlyUsed('user-123', 5)

// By environment
const prodDbs = await repo.getByEnvironment('production', 'user-123')

// By database type
const postgresDbs = await repo.getByType('postgres', 'user-123')
```

### Update Connection

```typescript
import { getConnectionRepository } from '@/lib/storage'

const repo = getConnectionRepository()

// Update connection
await repo.update('conn-123', {
  name: 'New Name',
  port: 5433,
})

// Update last used
await repo.updateLastUsed('conn-123')

// Add environment tag
await repo.addEnvironmentTag('conn-123', 'staging')

// Remove environment tag
await repo.removeEnvironmentTag('conn-123', 'staging')
```

### Environment Tags

```typescript
import { getConnectionRepository } from '@/lib/storage'

const repo = getConnectionRepository()

// Get all unique environment tags
const environments = await repo.getAllEnvironmentTags('user-123')
console.log('Environments:', environments) // ['production', 'staging', 'dev']

// Get connections by environment
const prodConns = await repo.getByEnvironment('production', 'user-123')
```

### Delete Connection

```typescript
import { getConnectionRepository } from '@/lib/storage'

const repo = getConnectionRepository()

// Delete single connection
await repo.delete('conn-123')

// Clear all connections for user
const deleted = await repo.clearUserConnections('user-123')
console.log(`Deleted ${deleted} connections`)
```

## Preferences

### User Preferences

```typescript
import { getPreferenceRepository, PreferenceCategory } from '@/lib/storage'

const repo = getPreferenceRepository()

// Set theme preference
await repo.setUserPreference(
  'user-123',
  'theme',
  'dark',
  PreferenceCategory.THEME
)

// Set editor preferences
await repo.setUserPreference(
  'user-123',
  'editor.fontSize',
  14,
  PreferenceCategory.EDITOR
)

await repo.setUserPreference(
  'user-123',
  'editor.tabSize',
  2,
  PreferenceCategory.EDITOR
)

// Get preference value (with default)
const theme = await repo.getUserPreferenceValue<string>(
  'user-123',
  'theme',
  'light' // default
)

const fontSize = await repo.getUserPreferenceValue<number>(
  'user-123',
  'editor.fontSize',
  12 // default
)
```

### Device Preferences

```typescript
import { getPreferenceRepository } from '@/lib/storage'

const repo = getPreferenceRepository()

// Set device-specific preference
await repo.setDevicePreference('window.width', 1920)
await repo.setDevicePreference('window.height', 1080)

// Get device preference
const width = await repo.getDevicePreferenceValue<number>('window.width', 1024)
```

### Bulk Preferences

```typescript
import { getPreferenceRepository, PreferenceCategory } from '@/lib/storage'

const repo = getPreferenceRepository()

// Set multiple preferences at once
await repo.bulkSet([
  { key: 'theme', value: 'dark', category: PreferenceCategory.THEME, userId: 'user-123' },
  { key: 'editor.fontSize', value: 14, category: PreferenceCategory.EDITOR, userId: 'user-123' },
  { key: 'editor.tabSize', value: 2, category: PreferenceCategory.EDITOR, userId: 'user-123' },
  { key: 'editor.wordWrap', value: true, category: PreferenceCategory.EDITOR, userId: 'user-123' },
])
```

### Preference Export/Import

```typescript
import { getPreferenceRepository } from '@/lib/storage'

const repo = getPreferenceRepository()

// Export preferences
const backup = await repo.exportPreferences('user-123')
console.log('Backup:', backup)

// Save to file
const json = JSON.stringify(backup, null, 2)
const blob = new Blob([json], { type: 'application/json' })
const url = URL.createObjectURL(blob)
// Trigger download...

// Import preferences
const imported = await repo.importPreferences(backup, 'user-123')
console.log(`Imported ${imported} preferences`)
```

### Get Preferences Map

```typescript
import { getPreferenceRepository } from '@/lib/storage'

const repo = getPreferenceRepository()

// Get all preferences as key-value map
const prefs = await repo.getPreferencesMap('user-123')
console.log(prefs)
// {
//   'theme': 'dark',
//   'editor.fontSize': 14,
//   'editor.tabSize': 2,
//   ...
// }
```

## Sync Queue

### Queue Changes

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

const repo = getSyncQueueRepository()

// Queue a create operation
await repo.queueChange(
  'connection',
  'conn-123',
  'create',
  {
    name: 'New Connection',
    type: 'postgres',
    // ... other fields
  }
)

// Queue an update
await repo.queueChange(
  'connection',
  'conn-123',
  'update',
  {
    name: 'Updated Name',
  }
)

// Queue a delete
await repo.queueChange(
  'connection',
  'conn-123',
  'delete',
  {}
)
```

### Process Sync Queue

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

const repo = getSyncQueueRepository()

// Process pending items
const { successful, failed } = await repo.processBatch(
  async (record) => {
    // Send to server
    const response = await fetch('/api/sync', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        entityType: record.entity_type,
        entityId: record.entity_id,
        operation: record.operation,
        payload: record.payload,
      }),
    })

    if (!response.ok) {
      throw new Error(`Sync failed: ${response.statusText}`)
    }
  },
  20 // batch size
)

console.log(`Synced ${successful.length} items`)
console.log(`Failed ${failed.length} items`)
```

### Monitor Sync Queue

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

const repo = getSyncQueueRepository()

// Get statistics
const stats = await repo.getStatistics()
console.log(`
  Total queued: ${stats.totalQueued}
  Failed items: ${stats.failedItems}
  Average retries: ${stats.averageRetries.toFixed(1)}

  By operation:
    Creates: ${stats.byOperation.create}
    Updates: ${stats.byOperation.update}
    Deletes: ${stats.byOperation.delete}

  By entity:
    Connections: ${stats.byEntity.connection}
    Queries: ${stats.byEntity.query}
    Preferences: ${stats.byEntity.preference}
`)

// Get pending items
const pending = await repo.getPending(50)
console.log(`${pending.length} items pending`)

// Get failed items
const failed = await repo.getFailed(20)
console.log(`${failed.length} items failed`)
```

### Retry Failed Syncs

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

const repo = getSyncQueueRepository()

// Retry all failed items
const retried = await repo.retryFailed()
console.log(`Retrying ${retried.length} failed items`)

// Clear items that exceeded max retries
const cleared = await repo.clearExceededRetries(5)
console.log(`Cleared ${cleared} items that failed too many times`)
```

## Advanced Patterns

### Offline-First Pattern

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

async function saveConnection(data: ConnectionData) {
  const repo = getSyncQueueRepository()

  // Save locally first
  const localConn = await saveToIndexedDB(data)

  // Try to sync to server
  try {
    if (navigator.onLine) {
      await saveToServer(data)
      console.log('Synced to server immediately')
    } else {
      throw new Error('Offline')
    }
  } catch (error) {
    // Queue for later sync
    await repo.queueChange(
      'connection',
      localConn.id,
      'create',
      data
    )
    console.log('Queued for sync when online')
  }

  return localConn
}

// Listen for online event
window.addEventListener('online', async () => {
  console.log('Back online, syncing...')
  const repo = getSyncQueueRepository()
  await repo.processBatch(syncToServer, 20)
})
```

### Optimistic Updates

```typescript
import { getConnectionRepository, getSyncQueueRepository } from '@/lib/storage'

async function updateConnectionOptimistic(id: string, updates: Partial<Connection>) {
  const connRepo = getConnectionRepository()
  const syncRepo = getSyncQueueRepository()

  // Update locally immediately
  await connRepo.update(id, updates)

  // Queue sync in background
  await syncRepo.queueChange('connection', id, 'update', updates)

  // Trigger background sync
  syncInBackground()
}
```

### Pagination with Infinite Scroll

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

async function* getQueryHistoryPages(userId: string, pageSize = 50) {
  const repo = getQueryHistoryRepository()
  let offset = 0

  while (true) {
    const page = await repo.search({
      userId,
      limit: pageSize,
      offset,
      sortDirection: 'desc',
    })

    if (page.items.length === 0) {
      break
    }

    yield page.items

    if (!page.hasMore) {
      break
    }

    offset += pageSize
  }
}

// Usage with React
async function loadMore() {
  const pages = getQueryHistoryPages('user-123')

  for await (const items of pages) {
    setQueries(prev => [...prev, ...items])

    // Stop if we have enough
    if (queries.length >= 200) {
      break
    }
  }
}
```

### Cache-First with Background Sync

```typescript
import { getConnectionRepository } from '@/lib/storage'

class ConnectionCache {
  private cache = new Map<string, Connection>()
  private repo = getConnectionRepository()

  async get(id: string): Promise<Connection | null> {
    // Check memory cache first
    if (this.cache.has(id)) {
      return this.cache.get(id)!
    }

    // Check IndexedDB
    const conn = await this.repo.get(id)
    if (conn) {
      this.cache.set(id, conn)
      return conn
    }

    // Fetch from server in background
    this.fetchFromServer(id)

    return null
  }

  private async fetchFromServer(id: string) {
    try {
      const response = await fetch(`/api/connections/${id}`)
      const conn = await response.json()

      // Update IndexedDB
      await this.repo.update(id, conn)

      // Update cache
      this.cache.set(id, conn)
    } catch (error) {
      console.error('Failed to fetch from server:', error)
    }
  }
}
```

### Storage Quota Management

```typescript
import { getStorageInfo, getQueryHistoryRepository } from '@/lib/storage'

async function manageStorageQuota() {
  const info = await getStorageInfo()

  if (!info.quota) {
    return
  }

  const percentage = info.percentage!

  if (percentage > 90) {
    console.warn('Storage almost full, cleaning up...')

    const queryRepo = getQueryHistoryRepository()

    // Delete queries older than 7 days
    const sevenDaysAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
    const deleted = await queryRepo.deleteOlderThan(sevenDaysAgo)

    console.log(`Freed up space by deleting ${deleted} old queries`)
  } else if (percentage > 95) {
    // Critical - request persistent storage
    if (navigator.storage && navigator.storage.persist) {
      const granted = await navigator.storage.persist()
      console.log('Persistent storage granted:', granted)
    }
  }
}

// Run periodically
setInterval(manageStorageQuota, 60 * 60 * 1000) // Every hour
```

### Migration from LocalStorage

```typescript
import {
  needsMigration,
  migrateFromLocalStorage,
  getMigrationStatus
} from '@/lib/storage'

async function checkAndMigrate() {
  const status = getMigrationStatus()

  if (!status.needed) {
    console.log('No migration needed')
    return
  }

  console.log('Migration status:', status)

  const result = await migrateFromLocalStorage('user-123', false)

  console.log(`
    Migrated:
    - ${result.connections} connections
    - ${result.queryHistory} queries
    - ${result.preferences} preferences
  `)

  if (result.errors.length > 0) {
    console.error('Migration errors:', result.errors)
  }
}
```

### Error Handling

```typescript
import {
  getQueryHistoryRepository,
  QuotaExceededError,
  NotFoundError,
  StorageError
} from '@/lib/storage'

async function safeQuerySave(data: QueryData) {
  const repo = getQueryHistoryRepository()

  try {
    await repo.create(data)
  } catch (error) {
    if (error instanceof QuotaExceededError) {
      // Handle quota exceeded
      alert('Storage full. Please clear old data.')

      // Auto-cleanup
      const thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
      await repo.deleteOlderThan(thirtyDaysAgo)

      // Retry
      await repo.create(data)
    } else if (error instanceof NotFoundError) {
      console.log('Record not found')
    } else if (error instanceof StorageError) {
      console.error('Storage error:', error.code, error.message)

      // Log to error tracking
      trackError(error)
    } else {
      throw error
    }
  }
}
```
