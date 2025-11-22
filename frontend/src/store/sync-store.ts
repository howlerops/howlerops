/**
 * Sync State Management Store
 *
 * Zustand store for managing sync state, conflicts, and configuration.
 * Provides reactive state for UI components and sync orchestration.
 *
 * @module store/sync-store
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'

import { SyncService } from '@/lib/sync/sync-service'
import type {
  Conflict,
  ConflictResolution,
  DeviceInfo,
  SyncConfig,
  SyncProgress,
  SyncResult,
  SyncStatus,
} from '@/types/sync'
import { DEFAULT_SYNC_CONFIG } from '@/types/sync'

/**
 * Sync store state interface
 */
interface SyncState {
  // State
  /** Current sync status */
  status: SyncStatus

  /** Is sync currently in progress */
  isSyncing: boolean

  /** Last successful sync timestamp */
  lastSyncAt?: Date

  /** Last sync result */
  lastSyncResult?: SyncResult

  /** Unresolved conflicts that need user attention */
  pendingConflicts: Conflict[]

  /** Whether sync is enabled */
  syncEnabled: boolean

  /** Sync configuration */
  config: SyncConfig

  /** Device information */
  deviceInfo?: DeviceInfo

  /** Current sync progress (if syncing) */
  progress?: SyncProgress

  /** Error message if sync failed */
  lastError?: string
}

/**
 * Sync store actions interface
 */
interface SyncActions {
  /**
   * Enable automatic sync
   */
  enableSync: () => Promise<void>

  /**
   * Disable automatic sync
   */
  disableSync: () => void

  /**
   * Trigger an immediate sync
   */
  syncNow: () => Promise<SyncResult>

  /**
   * Resolve a specific conflict
   */
  resolveConflict: (
    conflictId: string,
    resolution: ConflictResolution
  ) => Promise<void>

  /**
   * Clear all conflicts (after resolution)
   */
  clearConflicts: () => void

  /**
   * Update sync configuration
   */
  updateConfig: (updates: Partial<SyncConfig>) => void

  /**
   * Get current device info
   */
  getDeviceInfo: () => DeviceInfo | undefined

  /**
   * Clear last error
   */
  clearError: () => void

  /**
   * Reset sync state (nuclear option)
   */
  resetSync: () => Promise<void>

  /**
   * Get sync statistics
   */
  getSyncStats: () => {
    totalSyncs: number
    successfulSyncs: number
    failedSyncs: number
    totalConflicts: number
    lastSyncDuration?: number
  }
}

type SyncStore = SyncState & SyncActions

/**
 * Singleton sync service instance
 */
let syncServiceInstance: SyncService | null = null

/**
 * Get or create sync service instance
 */
function getSyncService(): SyncService {
  if (!syncServiceInstance) {
    syncServiceInstance = new SyncService()
  }
  return syncServiceInstance
}

/**
 * Default initial state
 */
const DEFAULT_STATE: SyncState = {
  status: 'idle',
  isSyncing: false,
  pendingConflicts: [],
  syncEnabled: false,
  config: DEFAULT_SYNC_CONFIG,
}

/**
 * Sync Management Store
 *
 * Usage:
 * ```typescript
 * const { syncEnabled, syncNow, pendingConflicts } = useSyncStore()
 *
 * // Enable sync
 * await enableSync()
 *
 * // Trigger manual sync
 * const result = await syncNow()
 *
 * // Resolve conflict
 * await resolveConflict(conflictId, 'remote')
 * ```
 */
export const useSyncStore = create<SyncStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...DEFAULT_STATE,

        enableSync: async () => {
          const service = getSyncService()

          // Update device info
          const deviceInfo = service.getDeviceInfo()
          set({ deviceInfo }, false, 'enableSync/deviceInfo')

          // Register progress callback
          service.onProgress((progress) => {
            set({ progress }, false, 'enableSync/progress')
          })

          // Start automatic sync
          service.startSync()

          set({ syncEnabled: true, status: 'idle' }, false, 'enableSync')

          // Perform initial sync
          try {
            await get().syncNow()
          } catch (error) {
            console.error('Initial sync failed:', error)
          }
        },

        disableSync: () => {
          const service = getSyncService()
          service.stopSync()
          set(
            { syncEnabled: false, status: 'idle', progress: undefined },
            false,
            'disableSync'
          )
        },

        syncNow: async () => {
          const state = get()

          if (state.isSyncing) {
            throw new Error('Sync already in progress')
          }

          set(
            { isSyncing: true, status: 'syncing', lastError: undefined },
            false,
            'syncNow/start'
          )

          try {
            const service = getSyncService()
            const result = await service.syncNow()

            // Update state with results
            const updates: Partial<SyncState> = {
              isSyncing: false,
              status: result.conflicts.length > 0 ? 'conflict' : 'idle',
              lastSyncAt: new Date(),
              lastSyncResult: result,
              pendingConflicts: result.conflicts,
              progress: undefined,
            }

            if (!result.success) {
              updates.status = 'error'
              updates.lastError = result.error
            }

            set(updates, false, 'syncNow/complete')

            return result
          } catch (error) {
            const errorMessage =
              error instanceof Error ? error.message : 'Unknown sync error'

            set(
              {
                isSyncing: false,
                status: 'error',
                lastError: errorMessage,
                progress: undefined,
              },
              false,
              'syncNow/error'
            )

            throw error
          }
        },

        resolveConflict: async (conflictId, resolution) => {
          const state = get()
          const conflict = state.pendingConflicts.find((c) => c.id === conflictId)

          if (!conflict) {
            throw new Error(`Conflict ${conflictId} not found`)
          }

          try {
            const service = getSyncService()
            await service.resolveConflict(conflictId, resolution, conflict)

            // Remove resolved conflict from pending
            set(
              {
                pendingConflicts: state.pendingConflicts.filter(
                  (c) => c.id !== conflictId
                ),
                status:
                  state.pendingConflicts.length === 1 ? 'idle' : 'conflict',
              },
              false,
              'resolveConflict'
            )
          } catch (error) {
            console.error('Failed to resolve conflict:', error)
            throw error
          }
        },

        clearConflicts: () => {
          set(
            { pendingConflicts: [], status: 'idle' },
            false,
            'clearConflicts'
          )
        },

        updateConfig: (updates) => {
          const currentConfig = get().config
          const newConfig = { ...currentConfig, ...updates }

          // Update service config
          const service = getSyncService()
          service.updateConfig(updates)

          set({ config: newConfig }, false, 'updateConfig')
        },

        getDeviceInfo: () => {
          const state = get()

          if (!state.deviceInfo) {
            const service = getSyncService()
            const deviceInfo = service.getDeviceInfo()
            set({ deviceInfo }, false, 'getDeviceInfo')
            return deviceInfo
          }

          return state.deviceInfo
        },

        clearError: () => {
          set({ lastError: undefined }, false, 'clearError')
        },

        resetSync: async () => {
          // Stop sync first
          get().disableSync()

          // Clear all local state
          set(
            {
              ...DEFAULT_STATE,
              config: get().config, // Keep config
            },
            false,
            'resetSync'
          )

          // Clear stored timestamps
          localStorage.removeItem('sync-last-timestamp')

          // Note: We don't reset device info to avoid creating multiple devices
        },

        getSyncStats: () => {
          const state = get()
          const lastResult = state.lastSyncResult

          return {
            totalSyncs: lastResult ? 1 : 0, // Would track this over time in production
            successfulSyncs: lastResult?.success ? 1 : 0,
            failedSyncs: lastResult?.success ? 0 : 1,
            totalConflicts: state.pendingConflicts.length,
            lastSyncDuration: lastResult?.durationMs,
          }
        },
      }),
      {
        name: 'sql-studio-sync-storage',
        version: 1,
        // Custom storage to handle Date serialization
        storage: {
          getItem: (name) => {
            const str = localStorage.getItem(name)
            if (!str) return null

            try {
              const { state } = JSON.parse(str)

              // Convert date strings back to Date objects
              if (state.lastSyncAt) {
                state.lastSyncAt = new Date(state.lastSyncAt)
              }
              if (state.lastSyncResult?.completedAt) {
                state.lastSyncResult.completedAt = new Date(
                  state.lastSyncResult.completedAt
                )
              }
              if (state.deviceInfo?.registeredAt) {
                state.deviceInfo.registeredAt = new Date(
                  state.deviceInfo.registeredAt
                )
              }
              if (state.deviceInfo?.lastSyncAt) {
                state.deviceInfo.lastSyncAt = new Date(
                  state.deviceInfo.lastSyncAt
                )
              }

              // Convert conflict dates
              if (state.pendingConflicts) {
                interface SerializedConflict {
                  localUpdatedAt: string
                  remoteUpdatedAt: string
                  [key: string]: unknown
                }
                state.pendingConflicts = state.pendingConflicts.map(
                  (c: SerializedConflict) => ({
                    ...c,
                    localUpdatedAt: new Date(c.localUpdatedAt),
                    remoteUpdatedAt: new Date(c.remoteUpdatedAt),
                  })
                )
              }

              return { state }
            } catch (error) {
              console.error('Failed to parse sync storage:', error)
              return null
            }
          },
          setItem: (name, value) => {
            localStorage.setItem(name, JSON.stringify(value))
          },
          removeItem: (name) => {
            localStorage.removeItem(name)
          },
        },
        // Only persist specific fields
        partialize: (state) => ({
          syncEnabled: state.syncEnabled,
          lastSyncAt: state.lastSyncAt?.toISOString(),
          config: state.config,
          deviceInfo: state.deviceInfo
            ? {
                ...state.deviceInfo,
                registeredAt: state.deviceInfo.registeredAt.toISOString(),
                lastSyncAt: state.deviceInfo.lastSyncAt?.toISOString(),
              }
            : undefined,
          // Don't persist: isSyncing, status, progress, errors, conflicts
        }),
      }
    ),
    {
      name: 'SyncStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Initialize sync store on app startup
 * Call this in your main App component
 */
export const initializeSyncStore = () => {
  const state = useSyncStore.getState()

  // Auto-enable sync if it was previously enabled
  if (state.syncEnabled) {
    state.enableSync().catch((error) => {
      console.error('Failed to initialize sync:', error)
    })
  }
}

/**
 * Selectors for common sync checks
 */
export const syncSelectors = {
  isActive: (state: SyncStore) => state.syncEnabled && !state.isSyncing,
  hasConflicts: (state: SyncStore) => state.pendingConflicts.length > 0,
  hasError: (state: SyncStore) => state.status === 'error',
  isOnline: () => navigator.onLine,
  canSync: (state: SyncStore) => state.syncEnabled && navigator.onLine,
  needsAttention: (state: SyncStore) =>
    state.status === 'error' || state.status === 'conflict',
}

/**
 * Hook for sync status indicator
 */
export const useSyncStatus = () => {
  const status = useSyncStore((state) => state.status)
  const isSyncing = useSyncStore((state) => state.isSyncing)
  const lastSyncAt = useSyncStore((state) => state.lastSyncAt)
  const syncEnabled = useSyncStore((state) => state.syncEnabled)
  const pendingConflicts = useSyncStore((state) => state.pendingConflicts)
  const lastError = useSyncStore((state) => state.lastError)
  const progress = useSyncStore((state) => state.progress)

  return {
    status,
    isSyncing,
    lastSyncAt,
    syncEnabled,
    hasConflicts: pendingConflicts.length > 0,
    conflictCount: pendingConflicts.length,
    hasError: !!lastError,
    errorMessage: lastError,
    progress,
  }
}

/**
 * Hook for sync actions
 */
export const useSyncActions = () => {
  const enableSync = useSyncStore((state) => state.enableSync)
  const disableSync = useSyncStore((state) => state.disableSync)
  const syncNow = useSyncStore((state) => state.syncNow)
  const resolveConflict = useSyncStore((state) => state.resolveConflict)
  const clearConflicts = useSyncStore((state) => state.clearConflicts)
  const updateConfig = useSyncStore((state) => state.updateConfig)
  const clearError = useSyncStore((state) => state.clearError)
  const resetSync = useSyncStore((state) => state.resetSync)

  return {
    enableSync,
    disableSync,
    syncNow,
    resolveConflict,
    clearConflicts,
    updateConfig,
    clearError,
    resetSync,
  }
}
