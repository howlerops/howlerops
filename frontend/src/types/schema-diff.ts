/**
 * Schema Diff Types Module
 *
 * Type-safe definitions for schema versioning, comparison, and migration features.
 * These types mirror the Wails bindings but provide cleaner TypeScript interfaces
 * without the Wails class boilerplate.
 */

import type { database } from '../../wailsjs/go/models'

/**
 * Branded type for snapshot identifiers
 */
type Brand<T, B> = T & { readonly __brand: B }
export type SnapshotId = Brand<string, 'SnapshotId'>

/**
 * Creates a validated SnapshotId from a string
 */
export function createSnapshotId(id: string): SnapshotId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid SnapshotId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as SnapshotId
}

/**
 * Status of a schema element (table, column, index, foreign key)
 */
export type DiffStatus = 'added' | 'removed' | 'modified' | 'unchanged'

/**
 * Direction of migration script
 */
export type MigrationDirection = 'forward' | 'backward'

/**
 * Database types supported for schema diffing
 */
export type SchemaDatabaseType = 'postgres' | 'mysql' | 'sqlite' | 'mssql' | 'oracle'

// ============================================================================
// Schema Snapshot Types
// ============================================================================

/**
 * Complete schema snapshot capturing the state of a database at a point in time
 */
export interface SchemaSnapshot {
  /** Unique snapshot identifier */
  id: SnapshotId
  /** User-friendly snapshot name */
  name: string
  /** Connection this snapshot was taken from */
  connectionId: string
  /** Database engine type */
  databaseType: SchemaDatabaseType
  /** List of schemas included in the snapshot */
  schemas: string[]
  /** Tables grouped by schema (schema_name -> TableInfo[]) */
  tables: Record<string, database.TableInfo[]>
  /** Full table structures (table_key -> TableStructure) */
  structures: Record<string, database.TableStructure>
  /** When snapshot was created */
  createdAt: Date
  /** Hash of snapshot content for integrity verification */
  hash: string
}

/**
 * Lightweight snapshot metadata for listing and selection
 */
export interface SnapshotMetadata {
  /** Unique snapshot identifier */
  id: SnapshotId
  /** User-friendly snapshot name */
  name: string
  /** Connection this snapshot was taken from */
  connectionId: string
  /** Database engine type */
  databaseType: SchemaDatabaseType
  /** Total number of tables in snapshot */
  tableCount: number
  /** When snapshot was created */
  createdAt: Date
  /** Size of snapshot in bytes */
  sizeBytes: number
}

// ============================================================================
// Schema Diff Types
// ============================================================================

/**
 * High-level comparison result between two schemas
 */
export interface SchemaDiff {
  /** Source identifier (connection ID or snapshot ID) */
  sourceId: string
  /** Target identifier (connection ID or snapshot ID) */
  targetId: string
  /** When comparison was performed */
  timestamp: Date
  /** Summary statistics of changes */
  summary: DiffSummary
  /** Detailed table-level differences */
  tables: TableDiff[]
  /** Time taken to compute diff in milliseconds */
  duration: number
}

/**
 * Summary statistics for schema differences
 */
export interface DiffSummary {
  /** Number of tables added */
  tablesAdded: number
  /** Number of tables removed */
  tablesRemoved: number
  /** Number of tables with changes */
  tablesModified: number
  /** Number of columns added across all tables */
  columnsAdded: number
  /** Number of columns removed across all tables */
  columnsRemoved: number
  /** Number of columns modified across all tables */
  columnsModified: number
  /** Number of index changes across all tables */
  indexesChanged: number
  /** Number of foreign key changes across all tables */
  foreignKeysChanged: number
}

/**
 * Difference for a single table
 */
export interface TableDiff {
  /** Schema name */
  schema: string
  /** Table name */
  name: string
  /** Overall table status */
  status: DiffStatus
  /** Column-level differences */
  columns?: ColumnDiff[]
  /** Index-level differences */
  indexes?: IndexDiff[]
  /** Foreign key differences */
  foreignKeys?: ForeignKeyDiff[]
}

/**
 * Difference for a single column
 */
export interface ColumnDiff {
  /** Column name */
  name: string
  /** Column status */
  status: DiffStatus
  /** Previous data type (if modified or removed) */
  oldType?: string
  /** New data type (if modified or added) */
  newType?: string
  /** Previous nullable constraint */
  oldNullable?: boolean
  /** New nullable constraint */
  newNullable?: boolean
  /** Previous default value */
  oldDefault?: string
  /** New default value */
  newDefault?: string
}

/**
 * Difference for a single index
 */
export interface IndexDiff {
  /** Index name */
  name: string
  /** Index status */
  status: DiffStatus
  /** Previous indexed columns */
  oldColumns?: string[]
  /** New indexed columns */
  newColumns?: string[]
  /** Previous unique constraint */
  oldUnique?: boolean
  /** New unique constraint */
  newUnique?: boolean
  /** Previous index method (btree, hash, etc.) */
  oldMethod?: string
  /** New index method */
  newMethod?: string
}

/**
 * Difference for a foreign key constraint
 */
export interface ForeignKeyDiff {
  /** Foreign key constraint name */
  name: string
  /** Foreign key status */
  status: DiffStatus
  /** Previous source columns */
  oldColumns?: string[]
  /** New source columns */
  newColumns?: string[]
  /** Previous referenced table */
  oldRefTable?: string
  /** New referenced table */
  newRefTable?: string
  /** Previous referenced columns */
  oldRefColumns?: string[]
  /** New referenced columns */
  newRefColumns?: string[]
  /** Previous ON DELETE action */
  oldOnDelete?: string
  /** New ON DELETE action */
  newOnDelete?: string
  /** Previous ON UPDATE action */
  oldOnUpdate?: string
  /** New ON UPDATE action */
  newOnUpdate?: string
}

// ============================================================================
// Migration Script Types
// ============================================================================

/**
 * Generated SQL migration script
 */
export interface MigrationScript {
  /** Migration direction */
  direction: MigrationDirection
  /** Target database type */
  databaseType: SchemaDatabaseType
  /** Generated SQL statements */
  sql: string
  /** When script was generated */
  generatedAt: Date
  /** Source schema identifier */
  sourceId: string
  /** Target schema identifier */
  targetId: string
}

/**
 * Migration script with validation warnings
 */
export interface MigrationScriptWithWarnings extends MigrationScript {
  /** Warnings about potentially destructive operations */
  warnings: MigrationWarning[]
}

/**
 * Warning about a migration operation
 */
export interface MigrationWarning {
  /** Warning severity */
  severity: 'info' | 'warning' | 'error'
  /** Affected object type */
  objectType: 'table' | 'column' | 'index' | 'foreign_key'
  /** Affected object name */
  objectName: string
  /** Warning message */
  message: string
  /** Whether this is a destructive operation */
  isDestructive: boolean
}

// ============================================================================
// UI Helper Types
// ============================================================================

/**
 * Filter options for snapshot list
 */
export interface SnapshotListFilter {
  /** Filter by connection ID */
  connectionId?: string
  /** Filter by database type */
  databaseType?: SchemaDatabaseType
  /** Filter by date range */
  dateFrom?: Date
  /** Filter by date range */
  dateTo?: Date
  /** Search by name */
  searchQuery?: string
}

/**
 * Sort options for snapshot list
 */
export interface SnapshotListSort {
  /** Field to sort by */
  field: 'name' | 'createdAt' | 'tableCount' | 'sizeBytes'
  /** Sort direction */
  direction: 'asc' | 'desc'
}

/**
 * Comparison request between two schemas
 */
export interface ComparisonRequest {
  /** Source (can be connection or snapshot) */
  sourceId: string
  /** Target (can be connection or snapshot) */
  targetId: string
  /** Whether to include unchanged items in results */
  includeUnchanged?: boolean
  /** Whether to compute detailed column diffs */
  includeColumnDetails?: boolean
}

/**
 * Statistics for a single diff status category
 */
export interface DiffStatusStats {
  /** Number of items with this status */
  count: number
  /** Percentage of total items */
  percentage: number
}

/**
 * Statistics breakdown for all diff statuses
 */
export interface DiffStatsBreakdown {
  /** Statistics for added items */
  added: DiffStatusStats
  /** Statistics for removed items */
  removed: DiffStatusStats
  /** Statistics for modified items */
  modified: DiffStatusStats
  /** Statistics for unchanged items */
  unchanged: DiffStatusStats
  /** Total number of items */
  total: number
}

/**
 * Migration preview with impact assessment
 */
export interface MigrationPreview {
  /** Generated migration script */
  script: MigrationScript
  /** Warnings about the migration */
  warnings: MigrationWarning[]
  /** Impact assessment */
  impact: MigrationImpact
  /** Estimated execution time in seconds */
  estimatedDuration?: number
}

/**
 * Impact assessment for a migration
 */
export interface MigrationImpact {
  /** Number of affected tables */
  tablesAffected: number
  /** Number of affected rows (estimated) */
  estimatedRowsAffected?: number
  /** Whether migration is reversible */
  isReversible: boolean
  /** Whether migration requires downtime */
  requiresDowntime: boolean
  /** Risk level assessment */
  riskLevel: 'low' | 'medium' | 'high'
}
