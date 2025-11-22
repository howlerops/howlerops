/**
 * Storage Repositories Index
 *
 * Centralized export of all storage repositories and utilities
 *
 * @module lib/storage/repositories
 */

// Repository exports
export {
  ConnectionRepository,
  type ConnectionSearchOptions,
  getConnectionRepository,
} from './connection-repository'
export {
  getPreferenceRepository,
  PreferenceCategory,
  PreferenceRepository,
  type PreferenceValue,
} from './preference-repository'
export {
  getQueryHistoryRepository,
  QueryHistoryRepository,
  type QueryHistorySearchOptions,
  type QueryStatistics,
} from './query-history-repository'
export {
  getSavedQueryRepository,
  SavedQueryRepository,
  type SavedQuerySearchOptions,
} from './saved-query-repository'
export {
  getSyncQueueRepository,
  SyncQueueRepository,
  type SyncQueueSearchOptions,
  type SyncStatistics,
} from './sync-queue-repository'

// Re-export common types
export type {
  ConnectionRecord,
  CreateInput,
  QueryHistoryRecord,
  SyncQueueRecord,
  UIPreferenceRecord,
  UpdateInput,
} from '@/types/storage'
