/**
 * Storage Module Index
 *
 * Complete IndexedDB infrastructure for Howlerops
 *
 * Features:
 * - Type-safe IndexedDB client
 * - Repository pattern for data access
 * - Schema versioning and migrations
 * - Offline sync queue
 * - Query history and analytics
 * - Connection management
 * - UI preferences
 *
 * Usage:
 * ```typescript
 * import { getQueryHistoryRepository } from '@/lib/storage'
 *
 * const repo = getQueryHistoryRepository()
 * const recent = await repo.getRecent(userId, 20)
 * ```
 *
 * @module lib/storage
 */

import { getIndexedDBClient, IndexedDBClient } from './indexeddb-client'

// Schema and client
export { CURRENT_VERSION, DB_NAME, getCurrentSchema } from './schema'

// Repositories
export {
  ConnectionRepository,
  getConnectionRepository,
  getPreferenceRepository,
  getQueryHistoryRepository,
  getSavedQueryRepository,
  getSyncQueueRepository,
  PreferenceCategory,
  PreferenceRepository,
  QueryHistoryRepository,
  SavedQueryRepository,
  SyncQueueRepository,
} from './repositories'

// Repository types
export type {
  ConnectionSearchOptions,
  PreferenceValue,
  QueryHistorySearchOptions,
  QueryStatistics,
  SavedQuerySearchOptions,
  SyncQueueSearchOptions,
  SyncStatistics,
} from './repositories'

// Storage types
export type {
  AIMessageRecord,
  AISessionRecord,
  ConnectionRecord,
  CreateInput,
  DatabaseType,
  EntityType,
  ExportFileRecord,
  PaginatedResult,
  PrivacyMode,
  QueryHistoryRecord,
  QueryOptions,
  SavedQueryRecord,
  SSLMode,
  StoreName,
  SyncOperation,
  SyncQueueRecord,
  UIPreferenceRecord,
  UpdateInput,
} from '@/types/storage'

// Error types
export {
  NotFoundError,
  QuotaExceededError,
  StorageError,
  TransactionError,
  VersionMismatchError,
} from '@/types/storage'

// Migration utilities
export {
  getMigrationStatus,
  migrateFromLocalStorage,
  type MigrationResult,
  needsMigration,
} from './migrate-from-localstorage'

/**
 * Initialize the storage system
 *
 * This function ensures the database is ready to use
 */
export async function initializeStorage(): Promise<void> {
  const client = getIndexedDBClient()
  // Opening the database triggers schema creation/migration
  await client.get('connections', 'init') // Dummy operation to ensure DB is ready
}

/**
 * Get storage usage information
 */
export async function getStorageInfo(): Promise<{
  supported: boolean
  usage?: number
  quota?: number
  percentage?: number
}> {
  const supported = IndexedDBClient.isSupported()

  if (!supported) {
    return { supported: false }
  }

  const estimate = await IndexedDBClient.getStorageEstimate()

  if (!estimate) {
    return { supported: true }
  }

  const usage = estimate.usage ?? 0
  const quota = estimate.quota ?? 0
  const percentage = quota > 0 ? (usage / quota) * 100 : 0

  return {
    supported: true,
    usage,
    quota,
    percentage,
  }
}

/**
 * Clear all storage data (use with caution)
 */
export async function clearAllStorage(): Promise<void> {
  const client = getIndexedDBClient()

  await Promise.all([
    client.clear('connections'),
    client.clear('query_history'),
    client.clear('saved_queries'),
    client.clear('ai_sessions'),
    client.clear('ai_messages'),
    client.clear('export_files'),
    client.clear('sync_queue'),
    client.clear('ui_preferences'),
  ])
}
