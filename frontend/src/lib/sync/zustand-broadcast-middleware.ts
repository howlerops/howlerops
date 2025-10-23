/**
 * Zustand Broadcast Middleware
 *
 * Automatically synchronizes Zustand store state across browser tabs using BroadcastChannel.
 * Prevents broadcast loops, supports partial state updates, and handles merge conflicts.
 *
 * Features:
 * - Automatic state broadcasting on changes
 * - Selective field synchronization (exclude sensitive data)
 * - Last-write-wins conflict resolution
 * - Debounced broadcasting to reduce message frequency
 * - Loop prevention using sender ID tracking
 *
 * Usage:
 * ```typescript
 * const useStore = create(
 *   broadcastSync('store-name', {
 *     excludeFields: ['password', 'sessionId'],
 *     debounceMs: 100
 *   })(
 *     persist(
 *       (set, get) => ({
 *         // store implementation
 *       })
 *     )
 *   )
 * )
 * ```
 */

import { StateCreator, StoreMutatorIdentifier } from 'zustand'
import { getBroadcastSync } from './broadcast-sync'

/**
 * Configuration options for broadcast sync middleware
 */
export interface BroadcastSyncOptions {
  /**
   * Fields to exclude from broadcasting (e.g., passwords, session data)
   */
  excludeFields?: string[]

  /**
   * Debounce time in milliseconds for broadcasting changes
   * Reduces broadcast frequency for rapidly changing state
   * @default 50
   */
  debounceMs?: number

  /**
   * Whether to enable broadcast sync
   * @default true
   */
  enabled?: boolean

  /**
   * Custom state merger function
   * Default implementation uses last-write-wins
   */
  merger?: (currentState: any, incomingPatch: any) => any
}

/**
 * Middleware state extension for tracking broadcast metadata
 */
interface BroadcastSyncState {
  __broadcastSync?: {
    lastUpdate: number
    senderId: string | null
  }
}

/**
 * Deep merge with last-write-wins strategy
 */
function defaultMerger(currentState: any, incomingPatch: any): any {
  if (!incomingPatch || typeof incomingPatch !== 'object') {
    return currentState
  }

  const merged = { ...currentState }

  for (const key in incomingPatch) {
    if (key === '__broadcastSync') continue

    const value = incomingPatch[key]

    if (value && typeof value === 'object' && !Array.isArray(value)) {
      // Recursively merge nested objects
      merged[key] = typeof merged[key] === 'object' && merged[key] !== null
        ? defaultMerger(merged[key], value)
        : value
    } else {
      // Direct assignment for primitives and arrays (last-write-wins)
      merged[key] = value
    }
  }

  return merged
}

/**
 * Remove excluded fields from state before broadcasting
 */
function sanitizeState(state: any, excludeFields: string[]): any {
  if (!state || typeof state !== 'object') {
    return state
  }

  const sanitized: any = {}

  for (const key in state) {
    if (excludeFields.includes(key) || key === '__broadcastSync') {
      continue
    }

    const value = state[key]

    if (value && typeof value === 'object' && !Array.isArray(value)) {
      sanitized[key] = sanitizeState(value, excludeFields)
    } else {
      sanitized[key] = value
    }
  }

  return sanitized
}

/**
 * Create a broadcast sync middleware for Zustand stores
 */
export function broadcastSync<T extends object>(
  storeName: string,
  options: BroadcastSyncOptions = {}
) {
  const {
    excludeFields = [],
    debounceMs = 50,
    enabled = true,
    merger = defaultMerger
  } = options

  return <
    Mps extends [StoreMutatorIdentifier, unknown][] = [],
    Mcs extends [StoreMutatorIdentifier, unknown][] = []
  >(
    config: StateCreator<T & BroadcastSyncState, Mps, Mcs>
  ): StateCreator<T & BroadcastSyncState, Mps, Mcs> => {
    return (set, get, api) => {
      // Initialize broadcast channel
      const broadcast = getBroadcastSync()
      let debounceTimer: ReturnType<typeof setTimeout> | null = null
      let isApplyingRemoteUpdate = false

      // Listen for store updates from other tabs
      const unsubscribe = broadcast.on('store-update', (message) => {
        // Ignore updates for different stores
        if (message.storeName !== storeName) {
          return
        }

        // Ignore updates from this tab
        if (message.senderId === broadcast.getTabId()) {
          return
        }

        // Apply the incoming patch
        isApplyingRemoteUpdate = true

        try {
          const currentState = get()
          const mergedState = merger(currentState, message.patch)

          // Update broadcast sync metadata
          mergedState.__broadcastSync = {
            lastUpdate: message.timestamp,
            senderId: message.senderId
          }

          set(mergedState, true)
        } catch (error) {
          console.error(`[BroadcastSync] Failed to apply update for store "${storeName}":`, error)
        } finally {
          isApplyingRemoteUpdate = false
        }
      })

      // Cleanup function
      api.destroy = () => {
        if (debounceTimer) {
          clearTimeout(debounceTimer)
        }
        unsubscribe()
      }

      // Wrap the config with broadcast logic
      const wrappedConfig = config(
        (partial, replace, ...args) => {
          // Apply local update first
          set(partial, replace, ...args)

          // Don't broadcast if:
          // 1. Broadcasting is disabled
          // 2. We're applying a remote update
          // 3. This is a replace operation (we only sync partial updates)
          if (!enabled || isApplyingRemoteUpdate || replace) {
            return
          }

          // Debounce broadcasts to reduce message frequency
          if (debounceTimer) {
            clearTimeout(debounceTimer)
          }

          debounceTimer = setTimeout(() => {
            try {
              const state = get()

              // Create a patch with only the changed fields
              let patch: any

              if (typeof partial === 'function') {
                // For function updates, we need to broadcast the entire state
                patch = sanitizeState(state, excludeFields)
              } else {
                // For partial updates, only broadcast the changed fields
                patch = sanitizeState(partial, excludeFields)
              }

              // Broadcast the update
              broadcast.send({
                type: 'store-update',
                storeName,
                patch,
                senderId: broadcast.getTabId(),
                timestamp: Date.now()
              })
            } catch (error) {
              console.error(`[BroadcastSync] Failed to broadcast update for store "${storeName}":`, error)
            }
          }, debounceMs)
        },
        get,
        api
      )

      return wrappedConfig
    }
  }
}

/**
 * Broadcast a specific action to other tabs
 * Useful for triggering actions without state changes (e.g., logout)
 */
export function broadcastAction(
  actionType: string,
  payload: any = {}
): void {
  const broadcast = getBroadcastSync()

  broadcast.send({
    type: 'store-update' as any,
    storeName: '__actions__',
    patch: { actionType, payload },
    senderId: broadcast.getTabId(),
    timestamp: Date.now()
  })
}

/**
 * Listen for broadcasted actions from other tabs
 */
export function onBroadcastAction(
  actionType: string,
  handler: (payload: any) => void
): () => void {
  const broadcast = getBroadcastSync()

  return broadcast.on('store-update', (message) => {
    if (message.storeName === '__actions__' && message.patch.actionType === actionType) {
      handler(message.patch.payload)
    }
  })
}
