# SQL Studio Cloud Sync System

Robust frontend synchronization service for SQL Studio's Individual tier users. Syncs local IndexedDB data with the backend while maintaining offline-first capabilities, conflict resolution, and data sanitization.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend Sync System                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │  UI Layer    │         │  Store Layer │                  │
│  │              │◄────────┤              │                  │
│  │ - Indicators │         │ Zustand      │                  │
│  │ - Conflicts  │         │ sync-store   │                  │
│  └──────────────┘         └──────┬───────┘                  │
│                                   │                          │
│                          ┌────────▼────────┐                 │
│                          │  Sync Service   │                 │
│                          │                 │                 │
│                          │ - Orchestration │                 │
│                          │ - Conflicts     │                 │
│                          │ - Progress      │                 │
│                          └───┬─────────┬───┘                 │
│                              │         │                     │
│              ┌───────────────┘         └──────────────┐      │
│              ▼                                        ▼      │
│     ┌────────────────┐                     ┌─────────────┐   │
│     │   IndexedDB    │                     │  Sync API   │   │
│     │   Client       │                     │  Client     │   │
│     │                │                     │             │   │
│     │ - Local        │                     │ - Upload    │   │
│     │ - Queries      │                     │ - Download  │   │
│     │ - History      │                     │ - Conflicts │   │
│     └────────────────┘                     └──────┬──────┘   │
│                                                   │          │
└───────────────────────────────────────────────────┼──────────┘
                                                    │
                                             ┌──────▼──────┐
                                             │   Backend   │
                                             │   REST API  │
                                             │             │
                                             │ - Storage   │
                                             │ - Sync      │
                                             └─────────────┘
```

## Key Features

### 1. Offline-First
- Queue changes when offline
- Automatic sync when back online
- Local-first operations with background sync

### 2. Data Sanitization
- **NEVER** uploads credentials (passwords, SSH keys)
- Strips sensitive data from connections
- Sanitizes query text (removes values)
- Safety validation before upload

### 3. Conflict Resolution
- Last-write-wins by default
- Manual resolution UI for conflicts
- Keep-both option for user choice
- Tracks sync versions

### 4. Progress Tracking
- Real-time progress updates
- Phase-based sync stages
- Item counts and percentages
- Estimated time remaining

### 5. Error Handling
- Exponential backoff retries
- Network error recovery
- Authentication failures
- User-friendly error messages

## Installation & Setup

### 1. Initialize Sync Store

In your main app component:

```typescript
import { initializeSyncStore } from '@/store/sync-store'

function App() {
  useEffect(() => {
    // Initialize sync store on app startup
    initializeSyncStore()
  }, [])

  return <YourApp />
}
```

### 2. Add UI Components

```typescript
import { SyncIndicator, ConflictResolver, SyncProgressBar } from '@/components/sync'

function Layout() {
  return (
    <div>
      <header>
        <SyncIndicator />
      </header>

      <main>
        {/* Your content */}
      </main>

      {/* Conflict resolution modal */}
      <ConflictResolver />

      {/* Progress bar */}
      <SyncProgressBar />
    </div>
  )
}
```

### 3. Enable Sync for User

```typescript
import { useSyncActions } from '@/store/sync-store'

function SyncSettings() {
  const { enableSync, disableSync } = useSyncActions()

  return (
    <button onClick={enableSync}>
      Enable Cloud Sync
    </button>
  )
}
```

## Usage Examples

### Manual Sync Trigger

```typescript
import { useSyncActions } from '@/store/sync-store'

function SyncButton() {
  const { syncNow } = useSyncActions()
  const [syncing, setSyncing] = useState(false)

  const handleSync = async () => {
    setSyncing(true)
    try {
      const result = await syncNow()
      console.log('Sync complete:', result)
    } catch (error) {
      console.error('Sync failed:', error)
    } finally {
      setSyncing(false)
    }
  }

  return (
    <button onClick={handleSync} disabled={syncing}>
      {syncing ? 'Syncing...' : 'Sync Now'}
    </button>
  )
}
```

### Monitor Sync Status

```typescript
import { useSyncStatus } from '@/store/sync-store'

function SyncStatus() {
  const {
    status,
    isSyncing,
    lastSyncAt,
    hasConflicts,
    conflictCount
  } = useSyncStatus()

  return (
    <div>
      <p>Status: {status}</p>
      {isSyncing && <p>Syncing...</p>}
      {lastSyncAt && <p>Last sync: {lastSyncAt.toLocaleString()}</p>}
      {hasConflicts && <p>{conflictCount} conflicts need resolution</p>}
    </div>
  )
}
```

### Resolve Conflicts

```typescript
import { useSyncStore, useSyncActions } from '@/store/sync-store'

function ConflictHandler() {
  const pendingConflicts = useSyncStore(state => state.pendingConflicts)
  const { resolveConflict } = useSyncActions()

  const handleResolve = async (conflictId: string) => {
    try {
      // Keep remote version (last-write-wins)
      await resolveConflict(conflictId, 'remote')
      console.log('Conflict resolved')
    } catch (error) {
      console.error('Failed to resolve:', error)
    }
  }

  return (
    <div>
      {pendingConflicts.map(conflict => (
        <div key={conflict.id}>
          <p>Conflict: {conflict.entityType}</p>
          <button onClick={() => handleResolve(conflict.id)}>
            Resolve
          </button>
        </div>
      ))}
    </div>
  )
}
```

### Custom Sync Configuration

```typescript
import { useSyncActions } from '@/store/sync-store'

function CustomConfig() {
  const { updateConfig } = useSyncActions()

  useEffect(() => {
    updateConfig({
      syncIntervalMs: 10 * 60 * 1000, // 10 minutes
      syncQueryHistory: false, // Don't sync history
      maxHistoryItems: 500,
      defaultConflictResolution: 'remote', // Auto-resolve to remote
    })
  }, [])

  return <div>Custom sync configuration applied</div>
}
```

### Progress Tracking

```typescript
import { SyncService } from '@/lib/sync/sync-service'

function ProgressTracker() {
  const [progress, setProgress] = useState<SyncProgress>()
  const service = new SyncService()

  useEffect(() => {
    const unsubscribe = service.onProgress((p) => {
      setProgress(p)
    })

    return unsubscribe
  }, [])

  if (!progress) return null

  return (
    <div>
      <p>Phase: {progress.phase}</p>
      <p>Progress: {progress.percentage}%</p>
      <p>{progress.processedItems} / {progress.totalItems}</p>
    </div>
  )
}
```

## Data Flow

### Upload Flow

1. **Collect Changes**: Get modified items from IndexedDB
2. **Sanitize**: Remove all credentials and sensitive data
3. **Validate**: Ensure safe to upload
4. **Batch**: Group into batches (default 100 items)
5. **Upload**: POST to `/api/sync/upload`
6. **Mark Synced**: Update `synced` flag locally

### Download Flow

1. **Request**: GET from `/api/sync/download?since={timestamp}`
2. **Receive**: Remote changes since last sync
3. **Detect Conflicts**: Compare sync versions
4. **Resolve**: Auto-resolve or prompt user
5. **Merge**: Apply changes to IndexedDB
6. **Update Timestamp**: Save last sync time

## Conflict Resolution Strategies

### 1. Last-Write-Wins (Default)
```typescript
recommendedResolution: localUpdatedAt > remoteUpdatedAt ? 'local' : 'remote'
```

### 2. Keep Local
```typescript
await resolveConflict(conflictId, 'local')
// Remote changes are discarded
```

### 3. Keep Remote
```typescript
await resolveConflict(conflictId, 'remote')
// Local changes are overwritten
```

### 4. Keep Both
```typescript
await resolveConflict(conflictId, 'keep-both')
// Creates duplicate: "Connection Name (remote)"
```

## Security Considerations

### What Gets Synced

✅ **Synced**:
- Connection metadata (host, port, database name)
- Saved query names and folders
- Query structure (sanitized)
- UI preferences
- Tags and labels

❌ **NEVER Synced**:
- Passwords
- SSH private keys
- API tokens
- Connection strings with credentials
- Query parameter values
- PII in query results

### Sanitization Example

```typescript
// Before sanitization
{
  connection_id: 'abc-123',
  name: 'Production DB',
  host: 'db.example.com',
  port: 5432,
  password: 'super-secret-123', // ❌ Will be removed
  sshTunnel: {
    password: 'ssh-password',     // ❌ Will be removed
    privateKey: '-----BEGIN...'   // ❌ Will be removed
  }
}

// After sanitization
{
  connection_id: 'abc-123',
  name: 'Production DB',
  host: 'db.example.com',
  port: 5432,
  passwordRequired: true,          // ✅ Indicator only
  sshTunnel: {
    passwordRequired: true,
    privateKeyRequired: true,
    privateKeyPath: '/path/to/key' // ✅ Path is safe
  },
  sanitizedAt: '2024-01-15T10:30:00Z'
}
```

## API Endpoints

### Upload Changes
```
POST /api/sync/upload
Content-Type: application/json
Authorization: Bearer {license-key}

{
  "deviceId": "device_abc123",
  "lastSyncAt": 1705320000000,
  "connections": [...],
  "savedQueries": [...],
  "queryHistory": [...]
}
```

### Download Changes
```
GET /api/sync/download?since={timestamp}&limit=100
Authorization: Bearer {license-key}

Response:
{
  "connections": [...],
  "savedQueries": [...],
  "queryHistory": [...],
  "serverTimestamp": 1705320100000,
  "hasMore": false
}
```

### Resolve Conflict
```
POST /api/sync/resolve
Content-Type: application/json
Authorization: Bearer {license-key}

{
  "conflict_id": "conflict-123",
  "resolution": "remote",
  "merged_data": {...}  // Optional
}
```

### Sync Status
```
GET /api/sync/status
Authorization: Bearer {license-key}

Response:
{
  "lastSyncAt": 1705320100000,
  "pendingChanges": 3,
  "conflicts": 1
}
```

## Configuration Options

```typescript
interface SyncConfig {
  autoSyncEnabled: boolean           // Enable automatic sync (default: true)
  syncIntervalMs: number             // Sync interval (default: 5 minutes)
  syncQueryHistory: boolean          // Sync query history (default: true)
  maxHistoryItems: number            // Max history items (default: 1000)
  enableConflictResolution: boolean  // Enable UI resolution (default: true)
  defaultConflictResolution: ConflictResolution // Default strategy (default: 'remote')
  autoRetry: boolean                 // Retry on failure (default: true)
  maxRetries: number                 // Max retry attempts (default: 3)
  retryDelayMs: number               // Retry delay (default: 1000ms)
  requireOnline: boolean             // Require online (default: true)
  uploadBatchSize: number            // Upload batch size (default: 100)
  downloadBatchSize: number          // Download batch size (default: 100)
}
```

## Testing

Run the test suite:

```bash
npm test -- lib/sync/__tests__/sync-service.test.ts
```

Tests cover:
- Initialization and configuration
- Periodic sync scheduling
- Complete sync cycle
- Data sanitization
- Conflict detection and resolution
- Progress tracking
- Error handling

## Performance Considerations

### Batch Processing
- Upload/download in batches of 100 items
- Prevents memory issues with large datasets
- Configurable via `uploadBatchSize` and `downloadBatchSize`

### Incremental Sync
- Only syncs changes since last sync
- Uses timestamps for efficient filtering
- Reduces bandwidth and processing time

### Background Sync
- Sync runs in background (non-blocking)
- Progress updates via callbacks
- Cancellable operations

## Troubleshooting

### Sync Not Starting
- Check tier: `useTierStore.getState().hasFeature('sync')`
- Verify authentication: `useTierStore.getState().licenseKey`
- Check online status: `navigator.onLine`

### Conflicts Not Resolving
- Check conflict count: `useSyncStore.getState().pendingConflicts.length`
- Verify resolution strategy: `config.defaultConflictResolution`
- Enable conflict UI: `config.enableConflictResolution = true`

### Upload Failures
- Check network: `await isSyncAvailable()`
- Verify sanitization: No credentials in upload payload
- Review server logs for rejection reasons

### Performance Issues
- Reduce batch size: `updateConfig({ uploadBatchSize: 50 })`
- Disable history sync: `updateConfig({ syncQueryHistory: false })`
- Increase sync interval: `updateConfig({ syncIntervalMs: 10 * 60 * 1000 })`

## Future Enhancements

- [ ] Differential sync (only changed fields)
- [ ] Compression for large payloads
- [ ] Conflict prediction
- [ ] Selective sync (choose what to sync)
- [ ] Sync analytics dashboard
- [ ] Background sync API integration
- [ ] WebSocket real-time sync
- [ ] P2P sync for team collaboration

## License

Part of SQL Studio - see main project license.
