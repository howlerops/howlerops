/**
 * Cloud Sync System Public API
 *
 * Main entry point for SQL Studio's cloud sync functionality.
 * Exports all public interfaces, services, and utilities for Individual tier.
 *
 * @module lib/sync/cloud-sync
 */

// Core sync service
export { SyncService } from './sync-service'

// Sync API client
export { getSyncClient, resetSyncClient, isSyncAvailable } from '@/lib/api/sync-client'
export {
  SyncClient,
  SyncClientError,
  AuthenticationError,
  NetworkError,
  ServerError,
} from '@/lib/api/sync-client'

// Type definitions
export type {
  SyncAction,
  SyncEntityType,
  SyncStatus,
  ConflictResolution,
  ChangeSet,
  UploadChangesRequest,
  UploadChangesResponse,
  DownloadChangesResponse,
  Conflict,
  SyncResult,
  SyncConfig,
  DeviceInfo,
  SyncQueueEntry,
  SyncProgress,
  SyncEvent,
  SyncStateSnapshot,
  MergeStrategy,
} from '@/types/sync'

export {
  DEFAULT_SYNC_CONFIG,
  isSyncEntityType,
  isConflictResolution,
  generateDeviceId,
} from '@/types/sync'

// Re-export sanitization utilities
export { sanitizeConnection, prepareConnectionsForSync } from '@/lib/sanitization/connection-sanitizer'
export type { SanitizedConnection } from '@/lib/sanitization/connection-sanitizer'

/**
 * Initialize cloud sync system
 * Call this once during app initialization
 */
export function initializeCloudSync(): void {
  console.log('Cloud sync system initialized')
}

/**
 * Check if cloud sync feature is available for current user
 */
export function canUseCloudSync(): boolean {
  if (typeof window === 'undefined') {
    return false
  }

  // Check if tier store is available
  try {
    const { useTierStore } = require('@/store/tier-store')
    const tierStore = useTierStore.getState()
    return tierStore.hasFeature('sync')
  } catch {
    return false
  }
}
