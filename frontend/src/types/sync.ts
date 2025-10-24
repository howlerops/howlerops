/**
 * Sync System Type Definitions
 *
 * Type-safe interfaces for cloud sync functionality in SQL Studio.
 * Supports offline-first, conflict resolution, and data sanitization.
 *
 * @module types/sync
 */

import type { ConnectionRecord, SavedQueryRecord, QueryHistoryRecord } from './storage'
import type { SanitizedConnection } from '@/lib/sanitization/connection-sanitizer'

/**
 * Sync operation types for change tracking
 */
export type SyncAction = 'create' | 'update' | 'delete'

/**
 * Entity types that can be synced
 */
export type SyncEntityType = 'connection' | 'saved_query' | 'query_history'

/**
 * Sync status indicators
 */
export type SyncStatus = 'idle' | 'syncing' | 'error' | 'conflict'

/**
 * Conflict resolution strategies
 */
export type ConflictResolution = 'local' | 'remote' | 'keep-both' | 'manual'

/**
 * Change set for a single entity
 */
export interface ChangeSet<T = unknown> {
  /** Unique change identifier */
  id: string

  /** Type of entity being changed */
  entityType: SyncEntityType

  /** Entity identifier */
  entityId: string

  /** Action type */
  action: SyncAction

  /** Entity data (sanitized for upload) */
  data?: T

  /** When the change occurred locally */
  timestamp: number

  /** Sync version for conflict detection */
  syncVersion: number

  /** Device that made the change */
  deviceId: string
}

/**
 * Request payload for uploading changes to server
 */
export interface UploadChangesRequest {
  /** Unique device identifier */
  deviceId: string

  /** Last successful sync timestamp */
  lastSyncAt: number

  /** Connection changes (sanitized) */
  connections: ChangeSet<SanitizedConnection>[]

  /** Saved query changes */
  savedQueries: ChangeSet<Omit<SavedQueryRecord, 'synced' | 'sync_version'>>[]

  /** Query history changes (optional, can be disabled) */
  queryHistory?: ChangeSet<Omit<QueryHistoryRecord, 'synced' | 'sync_version'>>[]
}

/**
 * Response from upload endpoint
 */
export interface UploadChangesResponse {
  /** Whether upload was successful */
  success: boolean

  /** Number of changes accepted */
  acceptedCount: number

  /** Number of changes rejected */
  rejectedCount: number

  /** Any conflicts detected during upload */
  conflicts?: Conflict[]

  /** Error message if failed */
  error?: string

  /** Server timestamp after upload */
  serverTimestamp: number
}

/**
 * Response from download endpoint
 */
export interface DownloadChangesResponse {
  /** Connections from server */
  connections: SanitizedConnection[]

  /** Saved queries from server */
  savedQueries: SavedQueryRecord[]

  /** Query history from server (optional) */
  queryHistory?: QueryHistoryRecord[]

  /** Server timestamp of this sync */
  serverTimestamp: number

  /** Whether more data is available (pagination) */
  hasMore: boolean

  /** Cursor for next page */
  nextCursor?: string
}

/**
 * Conflict between local and remote changes
 */
export interface Conflict<T = unknown> {
  /** Unique conflict identifier */
  id: string

  /** Type of entity in conflict */
  entityType: SyncEntityType

  /** Entity identifier */
  entityId: string

  /** Local version of the entity */
  localVersion: T

  /** Remote version of the entity */
  remoteVersion: T

  /** Local sync version */
  localSyncVersion: number

  /** Remote sync version */
  remoteSyncVersion: number

  /** When local change was made */
  localUpdatedAt: Date

  /** When remote change was made */
  remoteUpdatedAt: Date

  /** Recommended resolution (based on timestamps) */
  recommendedResolution: ConflictResolution

  /** Additional context about the conflict */
  reason: string
}

/**
 * Result of a sync operation
 */
export interface SyncResult {
  /** Whether sync completed successfully */
  success: boolean

  /** Number of items uploaded */
  uploaded: number

  /** Number of items downloaded */
  downloaded: number

  /** Any conflicts that need resolution */
  conflicts: Conflict[]

  /** Error message if sync failed */
  error?: string

  /** When sync completed */
  completedAt: Date

  /** How long sync took (ms) */
  durationMs: number

  /** Statistics about what was synced */
  stats: {
    connectionsUploaded: number
    connectionsDownloaded: number
    queriesUploaded: number
    queriesDownloaded: number
    historyUploaded: number
    historyDownloaded: number
  }
}

/**
 * Sync configuration options
 */
export interface SyncConfig {
  /** Enable automatic sync */
  autoSyncEnabled: boolean

  /** Auto-sync interval in milliseconds (default: 5 minutes) */
  syncIntervalMs: number

  /** Sync query history (can be disabled for privacy) */
  syncQueryHistory: boolean

  /** Maximum number of history items to sync */
  maxHistoryItems: number

  /** Enable conflict resolution UI */
  enableConflictResolution: boolean

  /** Default conflict resolution strategy */
  defaultConflictResolution: ConflictResolution

  /** Retry failed syncs automatically */
  autoRetry: boolean

  /** Maximum retry attempts */
  maxRetries: number

  /** Retry delay in milliseconds (exponential backoff) */
  retryDelayMs: number

  /** Sync only when online */
  requireOnline: boolean

  /** Batch size for uploads */
  uploadBatchSize: number

  /** Batch size for downloads */
  downloadBatchSize: number
}

/**
 * Device information for sync tracking
 */
export interface DeviceInfo {
  /** Unique device identifier (persisted) */
  deviceId: string

  /** Device name (user-editable) */
  deviceName: string

  /** Browser/platform information */
  userAgent: string

  /** When device was first registered */
  registeredAt: Date

  /** Last sync timestamp */
  lastSyncAt?: Date
}

/**
 * Sync queue entry for offline changes
 */
export interface SyncQueueEntry<T = unknown> {
  /** Unique queue entry ID */
  id: string

  /** Change set to sync */
  changeSet: ChangeSet<T>

  /** Number of upload attempts */
  attemptCount: number

  /** Last error message */
  lastError?: string

  /** When entry was added to queue */
  queuedAt: Date

  /** When last attempted */
  lastAttemptAt?: Date

  /** Next retry timestamp (for exponential backoff) */
  nextRetryAt?: Date
}

/**
 * Sync progress information
 */
export interface SyncProgress {
  /** Current phase of sync */
  phase: 'preparing' | 'uploading' | 'downloading' | 'resolving' | 'merging' | 'complete'

  /** Progress percentage (0-100) */
  percentage: number

  /** Current item being processed */
  currentItem?: string

  /** Total items to process */
  totalItems: number

  /** Items processed so far */
  processedItems: number

  /** Estimated time remaining (ms) */
  estimatedRemainingMs?: number
}

/**
 * Sync event types for listeners
 */
export type SyncEvent =
  | { type: 'sync-started' }
  | { type: 'sync-progress'; progress: SyncProgress }
  | { type: 'sync-completed'; result: SyncResult }
  | { type: 'sync-failed'; error: string }
  | { type: 'sync-conflict'; conflict: Conflict }
  | { type: 'sync-conflict-resolved'; conflictId: string; resolution: ConflictResolution }

/**
 * Sync state snapshot (for debugging/monitoring)
 */
export interface SyncStateSnapshot {
  /** Current sync status */
  status: SyncStatus

  /** Is sync currently in progress */
  isSyncing: boolean

  /** Last successful sync timestamp */
  lastSyncAt?: Date

  /** Last sync result */
  lastSyncResult?: SyncResult

  /** Number of pending changes in queue */
  pendingChanges: number

  /** Number of unresolved conflicts */
  unresolvedConflicts: number

  /** Current device info */
  deviceInfo: DeviceInfo

  /** Sync configuration */
  config: SyncConfig

  /** Whether user is authenticated */
  isAuthenticated: boolean

  /** Whether network is online */
  isOnline: boolean

  /** Whether sync is enabled for current tier */
  syncEnabled: boolean
}

/**
 * Merge strategy for resolving conflicts
 */
export interface MergeStrategy<T = unknown> {
  /** Merge function */
  merge: (local: T, remote: T, conflict: Conflict<T>) => T | null

  /** Whether this strategy can auto-resolve */
  canAutoResolve: boolean

  /** Strategy name */
  name: string
}

/**
 * Default sync configuration
 */
export const DEFAULT_SYNC_CONFIG: SyncConfig = {
  autoSyncEnabled: true,
  syncIntervalMs: 5 * 60 * 1000, // 5 minutes
  syncQueryHistory: true,
  maxHistoryItems: 1000,
  enableConflictResolution: true,
  defaultConflictResolution: 'remote', // Last-write-wins favoring remote
  autoRetry: true,
  maxRetries: 3,
  retryDelayMs: 1000, // 1 second, with exponential backoff
  requireOnline: true,
  uploadBatchSize: 100,
  downloadBatchSize: 100,
}

/**
 * Type guard for checking if a value is a valid sync entity type
 */
export function isSyncEntityType(value: string): value is SyncEntityType {
  return ['connection', 'saved_query', 'query_history'].includes(value)
}

/**
 * Type guard for checking if a value is a valid conflict resolution
 */
export function isConflictResolution(value: string): value is ConflictResolution {
  return ['local', 'remote', 'keep-both', 'manual'].includes(value)
}

/**
 * Create a unique device ID
 */
export function generateDeviceId(): string {
  // Combine multiple factors for a stable device ID
  const navigatorId = navigator.userAgent + navigator.language
  const screenId = `${screen.width}x${screen.height}x${screen.colorDepth}`
  const timezoneId = Intl.DateTimeFormat().resolvedOptions().timeZone

  const combined = `${navigatorId}|${screenId}|${timezoneId}`

  // Simple hash function
  let hash = 0
  for (let i = 0; i < combined.length; i++) {
    const char = combined.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash
  }

  // Convert to base36 and add a random component for uniqueness
  const hashStr = Math.abs(hash).toString(36)
  const random = Math.random().toString(36).substring(2, 8)

  return `device_${hashStr}_${random}`
}
