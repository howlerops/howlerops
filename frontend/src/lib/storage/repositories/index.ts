/**
 * Storage Repositories Index
 *
 * Centralized export of all storage repositories and utilities
 *
 * @module lib/storage/repositories
 */

// Repository exports
export {
  QueryHistoryRepository,
  getQueryHistoryRepository,
  type QueryHistorySearchOptions,
  type QueryStatistics,
} from './query-history-repository'

export {
  ConnectionRepository,
  getConnectionRepository,
  type ConnectionSearchOptions,
} from './connection-repository'

export {
  PreferenceRepository,
  getPreferenceRepository,
  PreferenceCategory,
  type PreferenceValue,
} from './preference-repository'

export {
  SyncQueueRepository,
  getSyncQueueRepository,
  type SyncQueueSearchOptions,
  type SyncStatistics,
} from './sync-queue-repository'

// Re-export common types
export type {
  ConnectionRecord,
  QueryHistoryRecord,
  UIPreferenceRecord,
  SyncQueueRecord,
  CreateInput,
  UpdateInput,
} from '@/types/storage'
