/**
 * IndexedDB Schema for Howlerops
 *
 * Defines the complete database schema including:
 * - Object stores (tables)
 * - Indexes for query performance
 * - Schema versioning and migrations
 *
 * @module lib/storage/schema
 */

import {
  STORE_NAMES,
  type SchemaVersion,
  type StoreConfig,
} from '@/types/storage'

/**
 * Current database version
 * Increment this when making schema changes
 */
export const CURRENT_VERSION = 2

/**
 * Database name
 */
export const DB_NAME = 'sql-studio-db'

/**
 * Connection store configuration
 *
 * Stores connection metadata (NO passwords)
 */
const connectionsStore: StoreConfig = {
  name: STORE_NAMES.CONNECTIONS,
  keyPath: 'connection_id',
  indexes: [
    {
      name: 'user_id',
      keyPath: 'user_id',
      unique: false,
    },
    {
      name: 'last_used_at',
      keyPath: 'last_used_at',
      unique: false,
    },
    {
      name: 'type',
      keyPath: 'type',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    {
      name: 'environment_tags',
      keyPath: 'environment_tags',
      unique: false,
      multiEntry: true, // Allows querying by individual tag
    },
  ],
}

/**
 * Query history store configuration
 *
 * Stores query execution history with performance metrics
 */
const queryHistoryStore: StoreConfig = {
  name: STORE_NAMES.QUERY_HISTORY,
  keyPath: 'id',
  indexes: [
    {
      name: 'user_id',
      keyPath: 'user_id',
      unique: false,
    },
    {
      name: 'connection_id',
      keyPath: 'connection_id',
      unique: false,
    },
    {
      name: 'executed_at',
      keyPath: 'executed_at',
      unique: false,
    },
    {
      name: 'privacy_mode',
      keyPath: 'privacy_mode',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    // Compound index for user + time queries
    {
      name: 'user_executed',
      keyPath: ['user_id', 'executed_at'],
      unique: false,
    },
    // Compound index for connection + time queries
    {
      name: 'connection_executed',
      keyPath: ['connection_id', 'executed_at'],
      unique: false,
    },
  ],
}

/**
 * Saved queries store configuration
 *
 * User's personal query library
 */
const savedQueriesStore: StoreConfig = {
  name: STORE_NAMES.SAVED_QUERIES,
  keyPath: 'id',
  indexes: [
    {
      name: 'user_id',
      keyPath: 'user_id',
      unique: false,
    },
    {
      name: 'is_favorite',
      keyPath: 'is_favorite',
      unique: false,
    },
    {
      name: 'folder',
      keyPath: 'folder',
      unique: false,
    },
    {
      name: 'tags',
      keyPath: 'tags',
      unique: false,
      multiEntry: true, // Allows querying by individual tag
    },
    {
      name: 'created_at',
      keyPath: 'created_at',
      unique: false,
    },
    {
      name: 'updated_at',
      keyPath: 'updated_at',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    // Compound index for user + favorites
    {
      name: 'user_favorites',
      keyPath: ['user_id', 'is_favorite'],
      unique: false,
    },
  ],
}

/**
 * Reports store configuration
 */
const reportsStore: StoreConfig = {
  name: STORE_NAMES.REPORTS,
  keyPath: 'id',
  indexes: [
    {
      name: 'name',
      keyPath: 'name',
      unique: false,
    },
    {
      name: 'folder',
      keyPath: 'folder',
      unique: false,
    },
    {
      name: 'tags',
      keyPath: 'tags',
      unique: false,
      multiEntry: true,
    },
    {
      name: 'updated_at',
      keyPath: 'updated_at',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
  ],
}

/**
 * AI sessions store configuration
 *
 * High-level AI conversation sessions
 */
const aiSessionsStore: StoreConfig = {
  name: STORE_NAMES.AI_SESSIONS,
  keyPath: 'id',
  indexes: [
    {
      name: 'user_id',
      keyPath: 'user_id',
      unique: false,
    },
    {
      name: 'created_at',
      keyPath: 'created_at',
      unique: false,
    },
    {
      name: 'updated_at',
      keyPath: 'updated_at',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    // Compound index for user + time queries
    {
      name: 'user_updated',
      keyPath: ['user_id', 'updated_at'],
      unique: false,
    },
  ],
}

/**
 * AI messages store configuration
 *
 * Detailed AI conversation messages
 */
const aiMessagesStore: StoreConfig = {
  name: STORE_NAMES.AI_MESSAGES,
  keyPath: 'id',
  indexes: [
    {
      name: 'session_id',
      keyPath: 'session_id',
      unique: false,
    },
    {
      name: 'timestamp',
      keyPath: 'timestamp',
      unique: false,
    },
    {
      name: 'role',
      keyPath: 'role',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    // Compound index for session + time queries
    {
      name: 'session_timestamp',
      keyPath: ['session_id', 'timestamp'],
      unique: false,
    },
  ],
}

/**
 * Export files store configuration
 *
 * Temporary storage for large query exports
 */
const exportFilesStore: StoreConfig = {
  name: STORE_NAMES.EXPORT_FILES,
  keyPath: 'id',
  indexes: [
    {
      name: 'created_at',
      keyPath: 'created_at',
      unique: false,
    },
    {
      name: 'expires_at',
      keyPath: 'expires_at',
      unique: false,
    },
    {
      name: 'format',
      keyPath: 'format',
      unique: false,
    },
  ],
}

/**
 * Sync queue store configuration
 *
 * Offline changes queue for server sync
 */
const syncQueueStore: StoreConfig = {
  name: STORE_NAMES.SYNC_QUEUE,
  keyPath: 'id',
  indexes: [
    {
      name: 'entity_type',
      keyPath: 'entity_type',
      unique: false,
    },
    {
      name: 'entity_id',
      keyPath: 'entity_id',
      unique: false,
    },
    {
      name: 'operation',
      keyPath: 'operation',
      unique: false,
    },
    {
      name: 'timestamp',
      keyPath: 'timestamp',
      unique: false,
    },
    {
      name: 'retry_count',
      keyPath: 'retry_count',
      unique: false,
    },
    // Compound index for entity queries
    {
      name: 'entity_lookup',
      keyPath: ['entity_type', 'entity_id'],
      unique: false,
    },
  ],
}

/**
 * UI preferences store configuration
 *
 * User interface settings and preferences
 */
const uiPreferencesStore: StoreConfig = {
  name: STORE_NAMES.UI_PREFERENCES,
  keyPath: 'id',
  indexes: [
    {
      name: 'user_id',
      keyPath: 'user_id',
      unique: false,
    },
    {
      name: 'key',
      keyPath: 'key',
      unique: false,
    },
    {
      name: 'category',
      keyPath: 'category',
      unique: false,
    },
    {
      name: 'device_id',
      keyPath: 'device_id',
      unique: false,
    },
    {
      name: 'synced',
      keyPath: 'synced',
      unique: false,
    },
    // Compound index for key lookups
    {
      name: 'user_key',
      keyPath: ['user_id', 'key'],
      unique: false,
    },
    {
      name: 'device_key',
      keyPath: ['device_id', 'key'],
      unique: false,
    },
  ],
}

/**
 * Schema version 1 - Initial schema
 */
const schemaV1: SchemaVersion = {
  version: 1,
  stores: [
    connectionsStore,
    queryHistoryStore,
    savedQueriesStore,
    aiSessionsStore,
    aiMessagesStore,
    exportFilesStore,
    syncQueueStore,
    uiPreferencesStore,
  ],
  migrate: async (db: IDBDatabase, transaction: IDBTransaction) => {
    // V1 is initial schema, no migration needed
    console.log('Initializing Howlerops database v1', db.name, transaction.mode)
  },
}

/**
 * Schema version 2 - Adds report definitions store
 */
const schemaV2: SchemaVersion = {
  version: 2,
  stores: [
    connectionsStore,
    queryHistoryStore,
    savedQueriesStore,
    reportsStore,
    aiSessionsStore,
    aiMessagesStore,
    exportFilesStore,
    syncQueueStore,
    uiPreferencesStore,
  ],
  migrate: async (db: IDBDatabase, transaction: IDBTransaction) => {
    console.log('Migrating Howlerops database to v2 (reports)', db.name, transaction.mode)
    if (!db.objectStoreNames.contains(STORE_NAMES.REPORTS)) {
      const store = db.createObjectStore(STORE_NAMES.REPORTS, { keyPath: 'id' })
      reportsStore.indexes?.forEach((index) => {
        store.createIndex(index.name, index.keyPath, {
          unique: index.unique ?? false,
          multiEntry: index.multiEntry ?? false,
        })
      })
    }
  },
}

/**
 * All schema versions
 * Add new versions here when making schema changes
 */
export const SCHEMA_VERSIONS: SchemaVersion[] = [schemaV1, schemaV2]

/**
 * Get the current schema version
 */
export function getCurrentSchema(): SchemaVersion {
  return SCHEMA_VERSIONS[SCHEMA_VERSIONS.length - 1]
}

/**
 * Get schema version by number
 */
export function getSchemaVersion(version: number): SchemaVersion | undefined {
  return SCHEMA_VERSIONS.find((s) => s.version === version)
}

/**
 * Validate that a schema version exists
 */
export function isValidVersion(version: number): boolean {
  return SCHEMA_VERSIONS.some((s) => s.version === version)
}
