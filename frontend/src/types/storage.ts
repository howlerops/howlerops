/**
 * IndexedDB Storage Types for Howlerops
 *
 * Comprehensive type definitions for local-first storage including:
 * - Connection metadata (no passwords)
 * - Query history and saved queries
 * - AI conversation sessions
 * - Export files and blobs
 * - Offline sync queue
 * - UI preferences
 */

// ============================================================================
// Base Types and Enums
// ============================================================================

/**
 * Database engines supported by Howlerops
 */
export type DatabaseType = 'postgres' | 'mysql' | 'sqlite' | 'mssql' | 'oracle' | 'mongodb';

/**
 * Privacy modes for query history
 */
export type PrivacyMode = 'private' | 'normal' | 'shared';

/**
 * AI conversation roles
 */
export type AIRole = 'system' | 'user' | 'assistant';

/**
 * Export file formats
 */
export type ExportFormat = 'csv' | 'json' | 'sql' | 'xlsx' | 'parquet';

/**
 * Sync queue operations
 */
export type SyncOperation = 'create' | 'update' | 'delete';

/**
 * Entity types for sync queue
 */
export type EntityType = 'connection' | 'query' | 'preference' | 'ai_session';

/**
 * SSL/TLS modes for database connections
 */
export type SSLMode = 'disable' | 'require' | 'verify-ca' | 'verify-full';

// ============================================================================
// Connection Store
// ============================================================================

/**
 * Connection metadata stored in IndexedDB
 *
 * SECURITY: Passwords are NEVER stored in IndexedDB.
 * Use sessionStorage or secure credential manager.
 */
export interface ConnectionRecord {
  /** Unique connection identifier (UUID) */
  connection_id: string;

  /** User who created this connection */
  user_id: string;

  /** Display name for the connection */
  name: string;

  /** Database type */
  type: DatabaseType;

  /** Database host/server */
  host: string;

  /** Database port */
  port: number;

  /** Database name */
  database: string;

  /** Username (password stored separately) */
  username: string;

  /** SSL/TLS configuration */
  ssl_mode: SSLMode;

  /** Additional connection parameters (JSON) */
  parameters?: Record<string, unknown>;

  /** Environment tags for organization */
  environment_tags: string[];

  /** When connection was created */
  created_at: Date;

  /** When connection was last modified */
  updated_at: Date;

  /** When connection was last used */
  last_used_at: Date;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

// ============================================================================
// Query History Store
// ============================================================================

/**
 * Query execution history entry
 *
 * Stores sanitized query execution records for:
 * - Recent queries dropdown
 * - Query performance analytics
 * - Audit logging
 */
export interface QueryHistoryRecord {
  /** Unique history entry ID */
  id: string;

  /** User who executed the query */
  user_id: string;

  /** The SQL query text (sanitized, no passwords/secrets) */
  query_text: string;

  /** Connection used for execution */
  connection_id: string;

  /** Execution time in milliseconds */
  execution_time_ms: number;

  /** Number of rows returned */
  row_count: number;

  /** Error message if query failed */
  error?: string;

  /** Privacy mode for this query */
  privacy_mode: PrivacyMode;

  /** When query was executed */
  executed_at: Date;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

// ============================================================================
// Saved Queries Store
// ============================================================================

/**
 * User's saved query library
 *
 * Personal query collection with organization features
 */
export interface SavedQueryRecord {
  /** Unique query ID */
  id: string;

  /** User who owns this query */
  user_id: string;

  /** Query title/name */
  title: string;

  /** Optional description */
  description?: string;

  /** The SQL query text */
  query_text: string;

  /** Tags for categorization */
  tags: string[];

  /** Folder/path for organization */
  folder?: string;

  /** Whether marked as favorite */
  is_favorite: boolean;

  /** When query was created */
  created_at: Date;

  /** When query was last modified */
  updated_at: Date;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

// ============================================================================
// AI Session Store
// ============================================================================

/**
 * AI conversation session metadata
 *
 * High-level session information without detailed messages
 */
export interface AISessionRecord {
  /** Unique session ID */
  id: string;

  /** User who owns this session */
  user_id: string;

  /** Session title */
  title: string;

  /** Auto-generated summary */
  summary?: string;

  /** Total messages in session */
  message_count: number;

  /** Approximate total tokens used */
  token_count: number;

  /** When session was created */
  created_at: Date;

  /** When session was last updated */
  updated_at: Date;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

/**
 * Individual AI message in a conversation
 *
 * Detailed message content and metadata
 */
export interface AIMessageRecord {
  /** Unique message ID */
  id: string;

  /** Parent session ID */
  session_id: string;

  /** Message role */
  role: AIRole;

  /** Message content (can be large) */
  content: string;

  /** Tokens used for this message */
  tokens?: number;

  /** When message was sent */
  timestamp: Date;

  /** Additional metadata (model, temperature, etc.) */
  metadata?: Record<string, unknown>;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

// ============================================================================
// Export Files Store
// ============================================================================

/**
 * Temporary export file storage
 *
 * Stores large query result exports with automatic cleanup
 */
export interface ExportFileRecord {
  /** Unique export ID */
  id: string;

  /** Export filename */
  filename: string;

  /** Export format */
  format: ExportFormat;

  /** Binary data (File or Blob) */
  blob: Blob;

  /** Size in bytes */
  size_bytes: number;

  /** When export was created */
  created_at: Date;

  /** When export should be deleted */
  expires_at: Date;
}

// ============================================================================
// Sync Queue Store
// ============================================================================

/**
 * Offline change queue for sync
 *
 * Tracks local changes when offline for later sync
 */
export interface SyncQueueRecord {
  /** Unique queue entry ID */
  id: string;

  /** Type of entity being synced */
  entity_type: EntityType;

  /** ID of the entity */
  entity_id: string;

  /** Operation type */
  operation: SyncOperation;

  /** Full entity payload */
  payload: Record<string, unknown>;

  /** When change was queued */
  timestamp: Date;

  /** Number of sync attempts */
  retry_count: number;

  /** Last error message if sync failed */
  last_error?: string;
}

// ============================================================================
// UI Preferences Store
// ============================================================================

/**
 * User interface preferences and settings
 *
 * Device-specific and synced settings
 */
export interface UIPreferenceRecord {
  /** Unique preference ID */
  id: string;

  /** User ID (for synced preferences) */
  user_id?: string;

  /** Preference key (e.g., 'theme', 'editor.fontSize') */
  key: string;

  /** Preference value (any JSON-serializable type) */
  value: unknown;

  /** Category for organization */
  category: string;

  /** Device ID (for device-specific preferences) */
  device_id?: string;

  /** When preference was last updated */
  updated_at: Date;

  /** Whether synced to server */
  synced: boolean;

  /** Server sync version */
  sync_version: number;
}

// ============================================================================
// Database Schema and Indexes
// ============================================================================

/**
 * Object store configuration
 */
export interface StoreConfig {
  /** Store name */
  name: string;

  /** Key path for primary key */
  keyPath: string;

  /** Whether to auto-increment keys */
  autoIncrement?: boolean;

  /** Index configurations */
  indexes?: IndexConfig[];
}

/**
 * Index configuration for queries
 */
export interface IndexConfig {
  /** Index name */
  name: string;

  /** Key path (can be array for compound index) */
  keyPath: string | string[];

  /** Whether index allows duplicate values */
  unique?: boolean;

  /** Whether index supports multi-entry */
  multiEntry?: boolean;
}

/**
 * Database schema version with migration
 */
export interface SchemaVersion {
  /** Version number */
  version: number;

  /** Stores to create/modify */
  stores: StoreConfig[];

  /** Migration function */
  migrate?: (db: IDBDatabase, transaction: IDBTransaction) => void | Promise<void>;
}

// ============================================================================
// Query and Pagination Types
// ============================================================================

/**
 * Query options for repository methods
 */
export interface QueryOptions {
  /** Index to query on */
  index?: string;

  /** Key range for query */
  range?: IDBKeyRange;

  /** Maximum number of results */
  limit?: number;

  /** Number of results to skip */
  offset?: number;

  /** Sort direction */
  direction?: 'next' | 'prev';
}

/**
 * Paginated query result
 */
export interface PaginatedResult<T> {
  /** Result items */
  items: T[];

  /** Total count (if available) */
  total?: number;

  /** Whether more results exist */
  hasMore: boolean;

  /** Cursor for next page */
  nextCursor?: IDBValidKey;
}

// ============================================================================
// Error Types
// ============================================================================

/**
 * Custom error types for IndexedDB operations
 */
export class StorageError extends Error {
  constructor(message: string, public readonly code: string, public readonly cause?: Error) {
    super(message);
    this.name = 'StorageError';
  }
}

export class QuotaExceededError extends StorageError {
  constructor(message = 'Storage quota exceeded', cause?: Error) {
    super(message, 'QUOTA_EXCEEDED', cause);
    this.name = 'QuotaExceededError';
  }
}

export class VersionMismatchError extends StorageError {
  constructor(message = 'Database version mismatch', cause?: Error) {
    super(message, 'VERSION_MISMATCH', cause);
    this.name = 'VersionMismatchError';
  }
}

export class TransactionError extends StorageError {
  constructor(message = 'Transaction failed', cause?: Error) {
    super(message, 'TRANSACTION_ERROR', cause);
    this.name = 'TransactionError';
  }
}

export class NotFoundError extends StorageError {
  constructor(message = 'Record not found', cause?: Error) {
    super(message, 'NOT_FOUND', cause);
    this.name = 'NotFoundError';
  }
}

// ============================================================================
// Utility Types
// ============================================================================

/**
 * Extract the primary key type from a record
 */
export type PrimaryKey<T> = T extends { id: infer K } ? K :
  T extends { connection_id: infer K } ? K : IDBValidKey;

/**
 * Make all Date fields and ID optional for input
 *
 * This allows repositories to auto-generate IDs when not provided.
 * Removes required status from: id, connection_id, timestamp, retry_count,
 * created_at, updated_at, synced, sync_version
 */
export type CreateInput<T> = Omit<
  T,
  'id' | 'connection_id' | 'timestamp' | 'retry_count' | 'created_at' | 'updated_at' | 'synced' | 'sync_version'
> & {
  id?: string;
  connection_id?: string;
  timestamp?: Date;
  retry_count?: number;
  created_at?: Date;
  updated_at?: Date;
  synced?: boolean;
  sync_version?: number;
};

/**
 * Make all fields except key optional for updates
 */
export type UpdateInput<T> = Partial<Omit<T, keyof PrimaryKey<T>>> & {
  updated_at?: Date;
};

/**
 * Store names as const for type safety
 */
export const STORE_NAMES = {
  CONNECTIONS: 'connections',
  QUERY_HISTORY: 'query_history',
  SAVED_QUERIES: 'saved_queries',
  AI_SESSIONS: 'ai_sessions',
  AI_MESSAGES: 'ai_messages',
  EXPORT_FILES: 'export_files',
  SYNC_QUEUE: 'sync_queue',
  UI_PREFERENCES: 'ui_preferences',
} as const;

export type StoreName = typeof STORE_NAMES[keyof typeof STORE_NAMES];
