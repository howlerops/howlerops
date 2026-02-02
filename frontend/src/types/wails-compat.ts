/**
 * Wails v3 Compatibility Types
 *
 * These types provide backwards compatibility for v2 namespaced types
 * that were previously imported as catalog.*, schemadiff.*, etc.
 *
 * In v3, types are flat exports but some frontend code still uses
 * the namespaced pattern. This module provides those namespaces.
 */

// =============================================================================
// CATALOG NAMESPACE
// =============================================================================

export namespace catalog {
  /** Represents catalog metadata for a table */
  export interface TableCatalogEntry {
    id: string
    connection_id: string
    schema_name: string
    table_name: string
    description?: string
    steward_user_id?: string | null
    tags?: string[]
    organization_id?: string | null
    columns?: ColumnCatalogEntry[]
    created_at: string
    updated_at: string
    created_by: string
  }

  /** Represents catalog metadata for a column */
  export interface ColumnCatalogEntry {
    id: string
    table_catalog_id: string
    column_name: string
    description?: string
    tags?: string[]
    pii_type?: string | null
    pii_confidence?: number | null
    created_at: string
    updated_at: string
  }

  /** Represents a reusable tag */
  export interface CatalogTag {
    id: string
    name: string
    color: string
    description?: string
    organization_id?: string | null
    is_system: boolean
    created_at: string
  }

  /** Factory functions for creating instances */
  export const TableCatalogEntry = {
    createFrom(data: Partial<TableCatalogEntry>): TableCatalogEntry {
      return {
        id: data.id ?? '',
        connection_id: data.connection_id ?? '',
        schema_name: data.schema_name ?? '',
        table_name: data.table_name ?? '',
        description: data.description ?? '',
        steward_user_id: data.steward_user_id ?? null,
        tags: data.tags ?? [],
        organization_id: data.organization_id ?? null,
        columns: data.columns ?? [],
        created_at: data.created_at ?? new Date().toISOString(),
        updated_at: data.updated_at ?? new Date().toISOString(),
        created_by: data.created_by ?? '',
      }
    }
  }

  export const ColumnCatalogEntry = {
    createFrom(data: Partial<ColumnCatalogEntry>): ColumnCatalogEntry {
      return {
        id: data.id ?? '',
        table_catalog_id: data.table_catalog_id ?? '',
        column_name: data.column_name ?? '',
        description: data.description ?? '',
        tags: data.tags ?? [],
        pii_type: data.pii_type ?? null,
        pii_confidence: data.pii_confidence ?? null,
        created_at: data.created_at ?? new Date().toISOString(),
        updated_at: data.updated_at ?? new Date().toISOString(),
      }
    }
  }

  export const CatalogTag = {
    createFrom(data: Partial<CatalogTag>): CatalogTag {
      return {
        id: data.id ?? '',
        name: data.name ?? '',
        color: data.color ?? '#808080',
        description: data.description ?? '',
        organization_id: data.organization_id ?? null,
        is_system: data.is_system ?? false,
        created_at: data.created_at ?? new Date().toISOString(),
      }
    }
  }
}

// =============================================================================
// SCHEMADIFF NAMESPACE
// =============================================================================

export namespace schemadiff {
  export type DiffStatus = 'added' | 'removed' | 'modified' | 'unchanged'

  /** Provides lightweight snapshot info for listing */
  export interface SnapshotMetadata {
    id: string
    name: string
    connection_id: string
    database_type: string
    table_count: number
    created_at: string
    size_bytes: number
  }

  /** Represents a saved point-in-time snapshot of a database schema */
  export interface SchemaSnapshot {
    id: string
    name: string
    connection_id: string
    database_type: string
    schemas: string[]
    tables: Record<string, TableInfo[]>
    structures: Record<string, TableStructure>
    created_at: string
    hash: string
  }

  /** Represents the complete diff between two database schemas */
  export interface SchemaDiff {
    source_id: string
    target_id: string
    timestamp: string
    summary: DiffSummary
    tables: TableDiff[]
    duration: number
  }

  /** Provides aggregate statistics about the diff */
  export interface DiffSummary {
    tables_added: number
    tables_removed: number
    tables_modified: number
    columns_added: number
    columns_removed: number
    columns_modified: number
    indexes_changed: number
    fks_changed: number
  }

  /** Represents the diff for a single table */
  export interface TableDiff {
    schema: string
    name: string
    status: DiffStatus
    columns?: ColumnDiff[]
    indexes?: IndexDiff[]
    foreign_keys?: FKDiff[]
  }

  /** Represents the diff for a single column */
  export interface ColumnDiff {
    name: string
    status: DiffStatus
    old_type?: string
    new_type?: string
    old_nullable?: boolean
    new_nullable?: boolean
    old_default?: string
    new_default?: string
  }

  /** Represents the diff for a single index */
  export interface IndexDiff {
    name: string
    status: DiffStatus
    old_columns?: string[]
    new_columns?: string[]
    old_unique?: boolean
    new_unique?: boolean
    old_method?: string
    new_method?: string
  }

  /** Represents the diff for a single foreign key */
  export interface FKDiff {
    name: string
    status: DiffStatus
    old_columns?: string[]
    new_columns?: string[]
    old_ref_table?: string
    new_ref_table?: string
    old_ref_columns?: string[]
    new_ref_columns?: string[]
    old_on_delete?: string
    new_on_delete?: string
    old_on_update?: string
    new_on_update?: string
  }

  // Supporting types
  export interface TableInfo {
    name: string
    schema: string
  }

  export interface TableStructure {
    columns: ColumnInfo[]
    indexes: IndexInfo[]
    foreign_keys: ForeignKeyInfo[]
  }

  export interface ColumnInfo {
    name: string
    data_type: string
    nullable: boolean
    default_value?: string | null
    primary_key: boolean
  }

  export interface IndexInfo {
    name: string
    columns: string[]
    unique: boolean
    method?: string
  }

  export interface ForeignKeyInfo {
    name: string
    columns: string[]
    ref_table: string
    ref_columns: string[]
    on_delete?: string
    on_update?: string
  }
}

// =============================================================================
// MAIN NAMESPACE (for CatalogStats etc.)
// =============================================================================

export namespace main {
  /** Statistics about the data catalog (matches v3 bindings) */
  export interface CatalogStats {
    total_tables: number
    total_columns: number
    tagged_tables: number
    pii_columns: number
  }
}

// =============================================================================
// DATABASE NAMESPACE
// =============================================================================

export namespace database {
  export type DatabaseType = 'postgresql' | 'mysql' | 'sqlite' | 'mariadb'

  export interface TableInfo {
    name: string
    schema: string
  }

  export interface TableStructure {
    columns: ColumnInfo[]
    indexes: IndexInfo[]
    foreign_keys: ForeignKeyInfo[]
  }

  export interface ColumnInfo {
    name: string
    data_type: string
    nullable: boolean
    default_value?: string | null
    primary_key: boolean
    unique: boolean
    indexed: boolean
    comment?: string
    ordinal_position: number
    character_maximum_length?: number | null
    numeric_precision?: number | null
    numeric_scale?: number | null
  }

  export interface IndexInfo {
    name: string
    columns: string[]
    unique: boolean
    method?: string
  }

  export interface ForeignKeyInfo {
    name: string
    columns: string[]
    ref_table: string
    ref_columns: string[]
    on_delete?: string
    on_update?: string
  }
}
