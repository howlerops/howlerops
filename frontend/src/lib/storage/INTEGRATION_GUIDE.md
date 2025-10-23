# IndexedDB Integration Guide

Complete guide for integrating IndexedDB storage with existing Zustand stores.

## Quick Start

### 1. Initialize Storage on App Load

```typescript
// app.tsx or main entry point
import { initializeStorage, needsMigration, migrateFromLocalStorage } from '@/lib/storage'

async function initApp() {
  try {
    // Initialize IndexedDB
    await initializeStorage()
    console.log('Storage initialized')

    // Check if migration is needed
    if (needsMigration()) {
      const result = await migrateFromLocalStorage('current-user-id', false)
      console.log('Migration complete:', result)
    }
  } catch (error) {
    console.error('Storage initialization failed:', error)
  }
}

initApp()
```

## Integration with Existing Stores

### Connection Store Integration

Add IndexedDB persistence to the connection store:

```typescript
// store/connection-store.ts
import { getConnectionRepository } from '@/lib/storage'

export const useConnectionStore = create<ConnectionState>()(
  devtools(
    persist(
      (set, get) => ({
        connections: [],
        activeConnection: null,

        // Initialize from IndexedDB on mount
        loadConnections: async () => {
          const repo = getConnectionRepository()
          const userId = getCurrentUserId() // Your user ID getter
          const connections = await repo.getAllForUser(userId)

          set({ connections })
        },

        // Enhanced addConnection with IndexedDB
        addConnection: async (connectionData) => {
          const newConnection: DatabaseConnection = {
            ...connectionData,
            id: crypto.randomUUID(),
            isConnected: false,
          }

          // Add to state
          set((state) => ({
            connections: [...state.connections, newConnection],
          }))

          // Persist to IndexedDB
          const repo = getConnectionRepository()
          await repo.create({
            connection_id: newConnection.id,
            user_id: getCurrentUserId(),
            name: newConnection.name,
            type: mapToStorageType(newConnection.type),
            host: newConnection.host ?? '',
            port: newConnection.port ?? 5432,
            database: newConnection.database,
            username: newConnection.username ?? '',
            ssl_mode: newConnection.sslMode ?? 'disable',
            parameters: newConnection.parameters,
            environment_tags: newConnection.environments ?? [],
          })
        },

        // Enhanced updateConnection
        updateConnection: async (id, updates) => {
          // Update state
          set((state) => ({
            connections: state.connections.map((conn) =>
              conn.id === id ? { ...conn, ...updates } : conn
            ),
          }))

          // Update IndexedDB
          const repo = getConnectionRepository()
          await repo.update(id, updates)
        },

        // Enhanced removeConnection
        removeConnection: async (id) => {
          // Remove from state
          set((state) => ({
            connections: state.connections.filter((conn) => conn.id !== id),
          }))

          // Remove from IndexedDB
          const repo = getConnectionRepository()
          await repo.delete(id)
        },

        // Track connection usage
        connectToDatabase: async (connectionId) => {
          // ... existing connection logic ...

          // Update last used timestamp
          const repo = getConnectionRepository()
          await repo.updateLastUsed(connectionId)
        },
      }),
      {
        name: 'connection-store',
        partialize: (state) => ({
          // Keep localStorage for quick access
          connections: state.connections,
        }),
      }
    )
  )
)

// Helper function to map database types
function mapToStorageType(type: DatabaseTypeString): DatabaseType {
  switch (type) {
    case 'postgresql':
      return 'postgres'
    case 'mysql':
    case 'mariadb':
      return 'mysql'
    case 'sqlite':
      return 'sqlite'
    case 'mssql':
      return 'mssql'
    case 'mongodb':
      return 'mongodb'
    default:
      return 'postgres'
  }
}
```

### Query Store Integration

Add query history tracking:

```typescript
// store/query-store.ts
import { getQueryHistoryRepository } from '@/lib/storage'

export const useQueryStore = create<QueryState>()(
  devtools(
    persist(
      (set, get) => ({
        tabs: [],
        activeTabId: null,
        results: [],
        queryHistory: [], // New: Recent queries

        // Load recent queries on mount
        loadQueryHistory: async () => {
          const repo = getQueryHistoryRepository()
          const userId = getCurrentUserId()
          const history = await repo.getRecent(userId, 50)

          set({ queryHistory: history })
        },

        // Enhanced executeQuery with history tracking
        executeQuery: async (tabId, query, connectionId) => {
          const startTime = Date.now()

          get().updateTab(tabId, { isExecuting: true })

          try {
            // Execute query
            const response = await wailsEndpoints.queries.execute(sessionId, query)
            const executionTime = Date.now() - startTime

            if (response.success && response.data) {
              // Add result to state
              const result = get().addResult({
                tabId,
                columns: response.data.columns,
                rows: response.data.rows,
                rowCount: response.data.rowCount,
                executionTime,
                query,
                connectionId,
              })

              // Save to query history
              const repo = getQueryHistoryRepository()
              await repo.create({
                user_id: getCurrentUserId(),
                query_text: query,
                connection_id: connectionId || 'unknown',
                execution_time_ms: executionTime,
                row_count: response.data.rowCount || 0,
                privacy_mode: 'normal',
              })

              // Refresh history
              get().loadQueryHistory()
            } else {
              // Save failed query
              const repo = getQueryHistoryRepository()
              await repo.create({
                user_id: getCurrentUserId(),
                query_text: query,
                connection_id: connectionId || 'unknown',
                execution_time_ms: Date.now() - startTime,
                row_count: 0,
                error: response.message,
                privacy_mode: 'normal',
              })

              throw new Error(response.message || 'Query execution failed')
            }
          } catch (error) {
            // Handle error
            console.error('Query execution failed:', error)
            throw error
          } finally {
            get().updateTab(tabId, { isExecuting: false })
          }
        },

        // Get query history with filters
        searchQueryHistory: async (options) => {
          const repo = getQueryHistoryRepository()
          const result = await repo.search(options)
          return result.items
        },

        // Get query statistics
        getQueryStatistics: async (connectionId?: string) => {
          const repo = getQueryHistoryRepository()
          const userId = getCurrentUserId()
          return repo.getStatistics(userId, { connectionId })
        },
      }),
      {
        name: 'query-store',
      }
    )
  )
)
```

### UI Preferences Store Integration

Create a new preferences store or integrate with existing:

```typescript
// store/preferences-store.ts
import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { getPreferenceRepository, PreferenceCategory } from '@/lib/storage'

interface PreferencesState {
  // Theme
  theme: 'light' | 'dark'

  // Editor
  editorFontSize: number
  editorTabSize: number
  showLineNumbers: boolean
  wordWrap: boolean

  // Behavior
  autoConnect: boolean
  autoSave: boolean

  // Actions
  loadPreferences: () => Promise<void>
  setTheme: (theme: 'light' | 'dark') => Promise<void>
  setEditorFontSize: (size: number) => Promise<void>
  setEditorTabSize: (size: number) => Promise<void>
  setShowLineNumbers: (show: boolean) => Promise<void>
  setWordWrap: (wrap: boolean) => Promise<void>
  setAutoConnect: (auto: boolean) => Promise<void>
  setAutoSave: (auto: boolean) => Promise<void>
}

export const usePreferencesStore = create<PreferencesState>()(
  devtools((set) => ({
    // Defaults
    theme: 'dark',
    editorFontSize: 14,
    editorTabSize: 2,
    showLineNumbers: true,
    wordWrap: false,
    autoConnect: true,
    autoSave: true,

    // Load from IndexedDB
    loadPreferences: async () => {
      const repo = getPreferenceRepository()
      const userId = getCurrentUserId()

      const theme = await repo.getUserPreferenceValue<string>(userId, 'theme', 'dark')
      const editorFontSize = await repo.getUserPreferenceValue<number>(userId, 'editor.fontSize', 14)
      const editorTabSize = await repo.getUserPreferenceValue<number>(userId, 'editor.tabSize', 2)
      const showLineNumbers = await repo.getUserPreferenceValue<boolean>(userId, 'editor.showLineNumbers', true)
      const wordWrap = await repo.getUserPreferenceValue<boolean>(userId, 'editor.wordWrap', false)
      const autoConnect = await repo.getUserPreferenceValue<boolean>(userId, 'behavior.autoConnect', true)
      const autoSave = await repo.getUserPreferenceValue<boolean>(userId, 'behavior.autoSave', true)

      set({
        theme: theme as 'light' | 'dark',
        editorFontSize,
        editorTabSize,
        showLineNumbers,
        wordWrap,
        autoConnect,
        autoSave,
      })
    },

    // Setters
    setTheme: async (theme) => {
      set({ theme })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'theme', theme, PreferenceCategory.THEME)
    },

    setEditorFontSize: async (size) => {
      set({ editorFontSize: size })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'editor.fontSize', size, PreferenceCategory.EDITOR)
    },

    setEditorTabSize: async (size) => {
      set({ editorTabSize: size })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'editor.tabSize', size, PreferenceCategory.EDITOR)
    },

    setShowLineNumbers: async (show) => {
      set({ showLineNumbers: show })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'editor.showLineNumbers', show, PreferenceCategory.EDITOR)
    },

    setWordWrap: async (wrap) => {
      set({ wordWrap: wrap })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'editor.wordWrap', wrap, PreferenceCategory.EDITOR)
    },

    setAutoConnect: async (auto) => {
      set({ autoConnect: auto })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'behavior.autoConnect', auto, PreferenceCategory.BEHAVIOR)
    },

    setAutoSave: async (auto) => {
      set({ autoSave: auto })
      const repo = getPreferenceRepository()
      await repo.setUserPreference(getCurrentUserId(), 'behavior.autoSave', auto, PreferenceCategory.BEHAVIOR)
    },
  }))
)
```

## Advanced Features

### Offline Sync Queue

Automatically queue changes when offline:

```typescript
import { getSyncQueueRepository } from '@/lib/storage'

// Queue changes when offline
async function saveWithSync(entityType, entityId, operation, data) {
  try {
    // Try to save to server
    await saveToServer(data)
  } catch (error) {
    // If offline, queue for later
    const syncRepo = getSyncQueueRepository()
    await syncRepo.queueChange(entityType, entityId, operation, data)
    console.log('Queued for sync:', entityId)
  }
}

// Process sync queue when back online
window.addEventListener('online', async () => {
  console.log('Back online, processing sync queue...')

  const syncRepo = getSyncQueueRepository()
  const { successful, failed } = await syncRepo.processBatch(
    async (record) => {
      await saveToServer(record.payload)
    },
    20 // batch size
  )

  console.log(`Synced ${successful.length} records, ${failed.length} failed`)
})
```

### Storage Management UI

Show storage usage in settings:

```typescript
import { getStorageInfo } from '@/lib/storage'

function StorageSettings() {
  const [info, setInfo] = useState(null)

  useEffect(() => {
    getStorageInfo().then(setInfo)
  }, [])

  if (!info?.supported) {
    return <div>IndexedDB not supported</div>
  }

  const usageMB = (info.usage / 1024 / 1024).toFixed(2)
  const quotaMB = (info.quota / 1024 / 1024).toFixed(2)

  return (
    <div>
      <h3>Storage Usage</h3>
      <p>{usageMB} MB / {quotaMB} MB ({info.percentage.toFixed(1)}%)</p>
      <progress value={info.percentage} max={100} />
    </div>
  )
}
```

## Best Practices

1. **Always use user IDs**: Filter data by user_id for multi-user support
2. **Batch operations**: Use transactions for multiple writes
3. **Error handling**: Always catch and handle storage errors
4. **Cleanup**: Regularly delete old data to maintain performance
5. **Loading states**: Show loading indicators for async operations
6. **Offline support**: Queue changes when offline
7. **Type safety**: Use TypeScript types from `@/types/storage`

## Migration Checklist

- [ ] Initialize storage on app load
- [ ] Run migration from localStorage
- [ ] Update connection store to use ConnectionRepository
- [ ] Update query store to use QueryHistoryRepository
- [ ] Create/update preferences store to use PreferenceRepository
- [ ] Add storage usage UI to settings
- [ ] Implement offline sync queue
- [ ] Add error handling for storage operations
- [ ] Test with multiple users
- [ ] Test quota exceeded scenarios
- [ ] Test offline/online transitions

## Troubleshooting

### "QuotaExceededError"
Clean up old data or request persistent storage:
```typescript
if (navigator.storage && navigator.storage.persist) {
  const granted = await navigator.storage.persist()
  console.log('Persistent storage:', granted)
}
```

### "Database version mismatch"
Clear database and reload:
```typescript
import { IndexedDBClient } from '@/lib/storage'
await IndexedDBClient.deleteDatabase()
location.reload()
```

### Slow queries
Ensure you're using indexes:
```typescript
// Good - uses index
await repo.search({ userId: 'user-123' })

// Bad - full scan
const all = await repo.getAll()
const filtered = all.filter(x => x.user_id === 'user-123')
```

## Support

For issues or questions, check:
- README.md for general usage
- TypeScript types in `@/types/storage`
- Repository documentation in each file
