/**
 * Cloud Sync Integration Example
 *
 * Complete example showing how to integrate cloud sync into SQL Studio.
 * Copy and adapt this code to your application.
 *
 * @module lib/sync/integration-example
 */

import { useEffect } from 'react'
import { initializeSyncStore, useSyncStatus, useSyncActions } from '@/store/sync-store'
import { ConflictResolver, SyncIndicator, SyncProgressBar } from '@/components/sync'
import { useTierStore } from '@/store/tier-store'

/**
 * Main App Component - Initialize Sync
 */
export function App() {
  useEffect(() => {
    // Initialize sync store on app startup
    initializeSyncStore()
  }, [])

  return (
    <div className="app">
      <Header />
      <Main />

      {/* Global sync components */}
      <ConflictResolver />
      <SyncProgressBar />
    </div>
  )
}

/**
 * Header with Sync Indicator
 */
function Header() {
  return (
    <header className="flex items-center justify-between p-4">
      <h1>SQL Studio</h1>

      {/* Show sync status in header */}
      <div className="flex items-center gap-4">
        <TierBadge />
        <SyncIndicator />
      </div>
    </header>
  )
}

/**
 * Tier Badge Component
 */
function TierBadge() {
  const currentTier = useTierStore((state) => state.currentTier)

  return (
    <span className="px-2 py-1 text-xs bg-primary text-primary-foreground rounded">
      {currentTier.charAt(0).toUpperCase() + currentTier.slice(1)} Tier
    </span>
  )
}

/**
 * Main Content Area
 */
function Main() {
  return (
    <main className="p-4">
      <SyncSettings />
      <ConnectionList />
    </main>
  )
}

/**
 * Sync Settings Panel
 */
function SyncSettings() {
  const { syncEnabled, lastSyncAt, hasConflicts, conflictCount } = useSyncStatus()
  const { enableSync, disableSync, syncNow } = useSyncActions()
  const hasFeature = useTierStore((state) => state.hasFeature('sync'))

  if (!hasFeature) {
    return (
      <div className="p-4 bg-muted rounded-lg">
        <p className="text-sm">
          Cloud sync requires Individual or Team tier.{' '}
          <a href="/upgrade" className="text-primary underline">
            Upgrade now
          </a>
        </p>
      </div>
    )
  }

  return (
    <div className="p-4 border rounded-lg space-y-4">
      <h2 className="text-lg font-semibold">Cloud Sync</h2>

      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium">
            {syncEnabled ? 'Enabled' : 'Disabled'}
          </p>
          {lastSyncAt && (
            <p className="text-xs text-muted-foreground">
              Last sync: {lastSyncAt.toLocaleString()}
            </p>
          )}
        </div>

        <div className="flex gap-2">
          <button
            onClick={syncEnabled ? disableSync : enableSync}
            className="px-3 py-1 text-sm rounded border"
          >
            {syncEnabled ? 'Disable' : 'Enable'}
          </button>

          {syncEnabled && (
            <button
              onClick={() => syncNow()}
              className="px-3 py-1 text-sm rounded bg-primary text-primary-foreground"
            >
              Sync Now
            </button>
          )}
        </div>
      </div>

      {hasConflicts && (
        <div className="p-2 bg-yellow-50 border border-yellow-200 rounded">
          <p className="text-sm text-yellow-800">
            {conflictCount} conflict{conflictCount > 1 ? 's' : ''} need your attention
          </p>
        </div>
      )}
    </div>
  )
}

/**
 * Connection List with Sync Status
 */
function ConnectionList() {
  const connections = [
    { id: '1', name: 'Production DB', synced: true },
    { id: '2', name: 'Development DB', synced: false },
  ]

  return (
    <div className="mt-6 space-y-2">
      <h2 className="text-lg font-semibold">Connections</h2>

      {connections.map((conn) => (
        <div
          key={conn.id}
          className="flex items-center justify-between p-3 border rounded"
        >
          <span>{conn.name}</span>

          {conn.synced ? (
            <span className="text-xs text-green-600">✓ Synced</span>
          ) : (
            <span className="text-xs text-yellow-600">⏳ Pending</span>
          )}
        </div>
      ))}
    </div>
  )
}

/**
 * Programmatic Sync Example
 */
export async function performBackgroundSync() {
  const { syncNow } = useSyncActions()

  try {
    const result = await syncNow()

    if (result.success) {
      console.log('Sync completed:', {
        uploaded: result.uploaded,
        downloaded: result.downloaded,
        duration: result.durationMs,
      })

      if (result.conflicts.length > 0) {
        console.warn('Conflicts detected:', result.conflicts)
        // Conflicts will be shown in UI automatically
      }
    }
  } catch (error) {
    console.error('Sync failed:', error)
    // Error will be shown in UI automatically
  }
}

/**
 * Settings Page - Advanced Configuration
 */
export function AdvancedSyncSettings() {
  const { updateConfig } = useSyncActions()

  const handleUpdateConfig = () => {
    updateConfig({
      syncIntervalMs: 10 * 60 * 1000, // 10 minutes
      syncQueryHistory: false, // Privacy: don't sync history
      maxHistoryItems: 500,
      defaultConflictResolution: 'remote', // Auto-resolve to remote
      uploadBatchSize: 50, // Smaller batches for slower networks
    })
  }

  return (
    <div className="p-4 space-y-4">
      <h2 className="text-lg font-semibold">Advanced Sync Settings</h2>

      <button
        onClick={handleUpdateConfig}
        className="px-4 py-2 rounded bg-primary text-primary-foreground"
      >
        Apply Custom Configuration
      </button>

      <div className="text-sm text-muted-foreground space-y-1">
        <p>• Sync interval: 10 minutes</p>
        <p>• Query history: Disabled</p>
        <p>• Conflict resolution: Automatic (remote wins)</p>
        <p>• Batch size: 50 items</p>
      </div>
    </div>
  )
}

/**
 * React Hook - Custom Sync Logic
 */
export function useAutoSync(enabled: boolean) {
  const { syncNow } = useSyncActions()

  useEffect(() => {
    if (!enabled) return

    // Sync on mount
    syncNow().catch(console.error)

    // Sync on window focus
    const handleFocus = () => {
      syncNow().catch(console.error)
    }

    window.addEventListener('focus', handleFocus)

    return () => {
      window.removeEventListener('focus', handleFocus)
    }
  }, [enabled, syncNow])
}

/**
 * Example: Using in a Component
 */
export function MyComponent() {
  const hasSync = useTierStore((state) => state.hasFeature('sync'))

  // Enable auto-sync on focus
  useAutoSync(hasSync)

  return (
    <div>
      {/* Your component content */}
    </div>
  )
}
