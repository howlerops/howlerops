# IndexedDB Storage Infrastructure

Complete local-first storage system for SQL Studio built on IndexedDB.

## Architecture

```
lib/storage/
├── index.ts                    # Main export file
├── schema.ts                   # Database schema and migrations
├── indexeddb-client.ts         # Low-level IndexedDB wrapper
└── repositories/               # Repository pattern implementations
    ├── index.ts                # Repository exports
    ├── query-history-repository.ts
    ├── connection-repository.ts
    ├── preference-repository.ts
    └── sync-queue-repository.ts
```

## Features

- **Type-safe**: Full TypeScript support with strict typing
- **Performance**: Optimized indexes for fast queries
- **Offline-first**: Works without network connection
- **Sync queue**: Automatic offline change tracking
- **Error handling**: Comprehensive error types and retry logic
- **Memory efficient**: Cursor-based pagination for large datasets
- **Browser compatible**: Chrome, Firefox, Safari, Edge

## Database Schema

### Object Stores

1. **connections** - Database connection metadata (NO passwords)
2. **query_history** - Query execution history with performance metrics
3. **saved_queries** - User's saved query library
4. **ai_sessions** - AI conversation sessions
5. **ai_messages** - Detailed AI messages
6. **export_files** - Temporary export file storage
7. **sync_queue** - Offline change queue for sync
8. **ui_preferences** - UI settings and preferences

### Indexes

Each store has optimized indexes for common query patterns:

- **connections**: user_id, last_used_at, type, environment_tags
- **query_history**: user_id, connection_id, executed_at, compound indexes
- **ui_preferences**: user_id + key, device_id + key for fast lookups

## Usage

### Basic Example

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

// Get repository instance
const queryRepo = getQueryHistoryRepository()

// Create a query history record
const record = await queryRepo.create({
  user_id: 'user-123',
  query_text: 'SELECT * FROM users',
  connection_id: 'conn-456',
  execution_time_ms: 125,
  row_count: 100,
  privacy_mode: 'normal',
})

// Search with filters
const results = await queryRepo.search({
  userId: 'user-123',
  connectionId: 'conn-456',
  startDate: new Date('2024-01-01'),
  limit: 50,
})

// Get statistics
const stats = await queryRepo.getStatistics('user-123')
console.log(`Average execution time: ${stats.averageExecutionTime}ms`)
```

### Connection Repository

```typescript
import { getConnectionRepository } from '@/lib/storage'

const connRepo = getConnectionRepository()

// Create connection (NO passwords)
const connection = await connRepo.create({
  user_id: 'user-123',
  name: 'Production DB',
  type: 'postgres',
  host: 'db.example.com',
  port: 5432,
  database: 'myapp',
  username: 'dbuser',
  ssl_mode: 'require',
  environment_tags: ['production'],
})

// Update last used timestamp
await connRepo.updateLastUsed(connection.connection_id)

// Get by environment
const prodDbs = await connRepo.getByEnvironment('production')
```

### Preference Repository

```typescript
import {
  getPreferenceRepository,
  PreferenceCategory
} from '@/lib/storage'

const prefRepo = getPreferenceRepository()

// Set user preference
await prefRepo.setUserPreference(
  'user-123',
  'theme',
  'dark',
  PreferenceCategory.THEME
)

// Get preference value (typed)
const theme = await prefRepo.getUserPreferenceValue<string>(
  'user-123',
  'theme',
  'light' // default
)

// Set device-specific preference
await prefRepo.setDevicePreference(
  'editor.fontSize',
  14,
  PreferenceCategory.EDITOR
)

// Export preferences for backup
const backup = await prefRepo.exportPreferences('user-123')
```

### Sync Queue Repository

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

const syncRepo = getSyncQueueRepository()

// Queue a change
await syncRepo.queueChange(
  'connection',
  'conn-123',
  'update',
  { name: 'New Name', updated_at: new Date() }
)

// Process sync queue
const { successful, failed } = await syncRepo.processBatch(
  async (record) => {
    // Send to server
    await fetch('/api/sync', {
      method: 'POST',
      body: JSON.stringify(record),
    })
  },
  10 // batch size
)

// Get statistics
const stats = await syncRepo.getStatistics()
console.log(`${stats.totalQueued} items queued`)
```

## Advanced Features

### Pagination

```typescript
const queryRepo = getQueryHistoryRepository()

// Paginated search
const page1 = await queryRepo.search({
  userId: 'user-123',
  limit: 50,
  offset: 0,
})

const page2 = await queryRepo.search({
  userId: 'user-123',
  limit: 50,
  offset: 50,
})
```

### Text Search

```typescript
const queryRepo = getQueryHistoryRepository()

// Search query text
const results = await queryRepo.search({
  userId: 'user-123',
  searchText: 'SELECT', // case-insensitive
  limit: 20,
})
```

### Performance Analytics

```typescript
const queryRepo = getQueryHistoryRepository()

// Get slow queries
const slowQueries = await queryRepo.getSlowQueries(
  1000, // threshold in ms
  'user-123',
  20
)

// Get failed queries
const failed = await queryRepo.getFailedQueries('user-123')

// Get statistics
const stats = await queryRepo.getStatistics('user-123', {
  startDate: new Date('2024-01-01'),
  endDate: new Date('2024-12-31'),
})
```

### Bulk Operations

```typescript
const prefRepo = getPreferenceRepository()

// Bulk set preferences
await prefRepo.bulkSet([
  { key: 'theme', value: 'dark', category: 'theme', userId: 'user-123' },
  { key: 'editor.fontSize', value: 14, category: 'editor', userId: 'user-123' },
  { key: 'editor.tabSize', value: 2, category: 'editor', userId: 'user-123' },
])
```

### Cleanup

```typescript
const queryRepo = getQueryHistoryRepository()

// Delete old history
const deleted = await queryRepo.deleteOlderThan(
  new Date(Date.now() - 30 * 24 * 60 * 60 * 1000) // 30 days ago
)

// Clear user history
await queryRepo.clearUserHistory('user-123')
```

## Integration with Zustand Stores

### Connection Store Integration

```typescript
import { getConnectionRepository } from '@/lib/storage'
import { useConnectionStore } from '@/store/connection-store'

// In connection store
const repo = getConnectionRepository()

// Save to IndexedDB when creating connection
addConnection: async (data) => {
  // Create in memory (current behavior)
  const connection = { ...data, id: crypto.randomUUID() }
  set((state) => ({ connections: [...state.connections, connection] }))

  // Persist to IndexedDB
  await repo.create({
    connection_id: connection.id,
    user_id: getCurrentUserId(),
    name: connection.name,
    type: connection.type,
    // ... other fields (NO password)
  })
}

// Load from IndexedDB on startup
const loadConnections = async () => {
  const userId = getCurrentUserId()
  const connections = await repo.getAllForUser(userId)
  set({ connections })
}
```

### Query Store Integration

```typescript
import { getQueryHistoryRepository } from '@/lib/storage'

const repo = getQueryHistoryRepository()

// Save query after execution
executeQuery: async (query, connectionId) => {
  const startTime = Date.now()

  try {
    const result = await wailsEndpoints.queries.execute(query)
    const executionTime = Date.now() - startTime

    // Save to IndexedDB
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

## Error Handling

```typescript
import {
  StorageError,
  QuotaExceededError,
  NotFoundError
} from '@/lib/storage'

try {
  await repo.create(data)
} catch (error) {
  if (error instanceof QuotaExceededError) {
    // Handle quota exceeded
    alert('Storage quota exceeded. Please clear old data.')
  } else if (error instanceof NotFoundError) {
    // Handle not found
    console.log('Record not found')
  } else if (error instanceof StorageError) {
    // Handle generic storage error
    console.error('Storage error:', error.code, error.message)
  }
}
```

## Storage Management

```typescript
import { getStorageInfo, clearAllStorage } from '@/lib/storage'

// Check storage usage
const info = await getStorageInfo()
console.log(`Using ${info.percentage}% of ${info.quota} bytes`)

// Clear all data (use with caution)
if (confirm('Clear all local data?')) {
  await clearAllStorage()
}
```

## Schema Migrations

When adding new fields or stores, update the schema version:

```typescript
// schema.ts
export const CURRENT_VERSION = 2

const schemaV2: SchemaVersion = {
  version: 2,
  stores: [
    // ... existing stores
    newStore, // new object store
  ],
  migrate: async (db, transaction) => {
    // Migration logic
    const store = transaction.objectStore('connections')
    const cursor = await store.openCursor()

    while (cursor) {
      const record = cursor.value
      // Update record with new field
      record.new_field = 'default_value'
      cursor.update(record)
      cursor.continue()
    }
  },
}

export const SCHEMA_VERSIONS = [schemaV1, schemaV2]
```

## Performance Tips

1. **Use indexes**: Query on indexed fields for best performance
2. **Batch operations**: Use `putMany()` for multiple records
3. **Pagination**: Use `limit` and `offset` for large result sets
4. **Compound indexes**: Use for multi-field queries
5. **Cleanup**: Regularly delete old data to maintain performance

## Security

- **NO passwords in IndexedDB**: Use sessionStorage for credentials
- **Sanitize queries**: Remove sensitive data from query text
- **Privacy modes**: Support private/shared query history
- **Encryption**: Consider encrypting sensitive preference values

## Browser Support

- Chrome 24+
- Firefox 16+
- Safari 10+
- Edge 12+

## Testing

```typescript
import { resetIndexedDBClient } from '@/lib/storage'

// Reset for testing
afterEach(async () => {
  resetIndexedDBClient()
  await IndexedDBClient.deleteDatabase()
})
```

## Troubleshooting

### Quota Exceeded

```typescript
// Check quota
const info = await getStorageInfo()
if (info.percentage > 90) {
  // Clean up old data
  await queryRepo.deleteOlderThan(oldDate)
}
```

### Version Mismatch

Clear the database and reload:

```typescript
await IndexedDBClient.deleteDatabase()
location.reload()
```

### Slow Queries

Ensure you're using the right indexes:

```typescript
// Good - uses index
await queryRepo.search({ userId: 'user-123' })

// Bad - full table scan
const all = await client.getAll('query_history')
const filtered = all.filter(q => q.user_id === 'user-123')
```

## License

Part of SQL Studio - see main LICENSE file.
