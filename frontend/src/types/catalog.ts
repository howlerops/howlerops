/**
 * Data Catalog Types Module
 *
 * Type-safe definitions for the data governance catalog including:
 * - Table and column metadata
 * - PII detection and classification
 * - Tagging and categorization
 * - Search and discovery
 * - Data stewardship
 */

/**
 * Branded type for catalog identifiers
 */
type Brand<T, B> = T & { readonly __brand: B }

export type CatalogTagId = Brand<string, 'CatalogTagId'>
export type TableCatalogId = Brand<string, 'TableCatalogId'>
export type ColumnCatalogId = Brand<string, 'ColumnCatalogId'>
export type UserId = Brand<string, 'UserId'>

/**
 * Creates a validated CatalogTagId from a string
 */
export function createCatalogTagId(id: string): CatalogTagId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid CatalogTagId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as CatalogTagId
}

/**
 * Creates a validated TableCatalogId from a string
 */
export function createTableCatalogId(id: string): TableCatalogId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid TableCatalogId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as TableCatalogId
}

/**
 * Creates a validated ColumnCatalogId from a string
 */
export function createColumnCatalogId(id: string): ColumnCatalogId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid ColumnCatalogId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as ColumnCatalogId
}

// ============================================================================
// PII Types
// ============================================================================

/**
 * PII classification types based on common standards (GDPR, CCPA, etc.)
 */
export type PIIType =
  | 'email'
  | 'phone'
  | 'ssn'
  | 'credit_card'
  | 'ip_address'
  | 'postal_address'
  | 'name'
  | 'date_of_birth'
  | 'financial_account'
  | 'medical_record'
  | 'biometric'
  | 'government_id'
  | 'custom'

/**
 * PII confidence levels from detection algorithms
 */
export type PIIConfidence = 'low' | 'medium' | 'high' | 'confirmed'

/**
 * PII detection result
 */
export interface PIIDetectionResult {
  /** Column identifier */
  columnCatalogId: ColumnCatalogId
  /** Detected PII type */
  piiType: PIIType
  /** Detection confidence score (0-1) */
  confidence: number
  /** Confidence level category */
  confidenceLevel: PIIConfidence
  /** Evidence supporting the detection */
  evidence: string[]
  /** Whether manually confirmed by steward */
  manuallyConfirmed: boolean
}

// ============================================================================
// Tag Types
// ============================================================================

/**
 * Catalog tag for organizing and categorizing data assets
 */
export interface CatalogTag {
  /** Unique tag identifier */
  id: CatalogTagId
  /** Tag name (displayed to users) */
  name: string
  /** Tag color for visual organization (hex color) */
  color: string
  /** Optional tag description */
  description?: string
  /** Organization this tag belongs to */
  organizationId?: string
  /** Whether tag is system-defined (cannot be deleted) */
  isSystem: boolean
  /** When tag was created */
  createdAt: Date
}

/**
 * Tag creation input
 */
export interface CreateTagInput {
  /** Tag name */
  name: string
  /** Tag color (hex format) */
  color: string
  /** Optional description */
  description?: string
  /** Organization ID (defaults to current user's org) */
  organizationId?: string
}

/**
 * Tag update input
 */
export interface UpdateTagInput {
  /** New tag name */
  name?: string
  /** New tag color */
  color?: string
  /** New description */
  description?: string
}

// ============================================================================
// Table Catalog Types
// ============================================================================

/**
 * Table-level catalog entry with metadata and governance
 */
export interface TableCatalogEntry {
  /** Unique table catalog identifier */
  id: TableCatalogId
  /** Connection this table belongs to */
  connectionId: string
  /** Schema name */
  schemaName: string
  /** Table name */
  tableName: string
  /** Business-friendly description */
  description?: string
  /** Data steward responsible for this table */
  stewardUserId?: UserId
  /** Tags for categorization */
  tags?: string[]
  /** Organization this entry belongs to */
  organizationId?: string
  /** Column-level catalog entries */
  columns?: ColumnCatalogEntry[]
  /** When entry was created */
  createdAt: Date
  /** When entry was last updated */
  updatedAt: Date
  /** User who created this entry */
  createdBy: UserId
}

/**
 * Table catalog creation input
 */
export interface CreateTableCatalogInput {
  /** Connection ID */
  connectionId: string
  /** Schema name */
  schemaName: string
  /** Table name */
  tableName: string
  /** Optional description */
  description?: string
  /** Optional steward */
  stewardUserId?: UserId
  /** Optional tags */
  tags?: string[]
}

/**
 * Table catalog update input
 */
export interface UpdateTableCatalogInput {
  /** New description */
  description?: string
  /** New steward */
  stewardUserId?: UserId
  /** New tags (replaces existing) */
  tags?: string[]
}

// ============================================================================
// Column Catalog Types
// ============================================================================

/**
 * Column-level catalog entry with PII classification
 */
export interface ColumnCatalogEntry {
  /** Unique column catalog identifier */
  id: ColumnCatalogId
  /** Parent table catalog entry */
  tableCatalogId: TableCatalogId
  /** Column name */
  columnName: string
  /** Business-friendly description */
  description?: string
  /** Tags for categorization */
  tags?: string[]
  /** PII classification type */
  piiType?: PIIType
  /** PII detection confidence (0-1) */
  piiConfidence?: number
  /** When entry was created */
  createdAt: Date
  /** When entry was last updated */
  updatedAt: Date
}

/**
 * Column catalog creation input
 */
export interface CreateColumnCatalogInput {
  /** Parent table catalog ID */
  tableCatalogId: TableCatalogId
  /** Column name */
  columnName: string
  /** Optional description */
  description?: string
  /** Optional tags */
  tags?: string[]
  /** Optional PII type */
  piiType?: PIIType
  /** Optional PII confidence */
  piiConfidence?: number
}

/**
 * Column catalog update input
 */
export interface UpdateColumnCatalogInput {
  /** New description */
  description?: string
  /** New tags (replaces existing) */
  tags?: string[]
  /** New PII type */
  piiType?: PIIType
  /** New PII confidence */
  piiConfidence?: number
}

// ============================================================================
// Search Types
// ============================================================================

/**
 * Search filters for catalog discovery
 */
export interface SearchFilters {
  /** Filter by connection ID */
  connectionId?: string
  /** Filter by schema name (supports wildcards) */
  schemaPattern?: string
  /** Filter by table name (supports wildcards) */
  tablePattern?: string
  /** Filter by tags (any match) */
  tags?: string[]
  /** Filter by PII presence */
  hasPII?: boolean
  /** Filter by steward assignment */
  hasSteward?: boolean
  /** Filter by organization */
  organizationId?: string
  /** Maximum results to return */
  limit?: number
}

/**
 * Search result type discriminator
 */
export type SearchResultType = 'table' | 'column'

/**
 * Individual search result
 */
export interface SearchResult {
  /** Result type */
  type: SearchResultType
  /** Catalog entry ID */
  id: string
  /** Connection ID */
  connectionId: string
  /** Schema name */
  schemaName: string
  /** Table name */
  tableName: string
  /** Column name (if type is 'column') */
  columnName?: string
  /** Description */
  description?: string
  /** Tags */
  tags?: string[]
  /** Relevance score (0-1) */
  relevanceScore: number
}

/**
 * Search results with metadata
 */
export interface SearchResults {
  /** Matching results */
  results: SearchResult[]
  /** Total number of matches */
  total: number
  /** Original search query */
  query: string
}

// ============================================================================
// Statistics Types
// ============================================================================

/**
 * Catalog statistics for a connection
 */
export interface CatalogStats {
  /** Total tables cataloged */
  totalTables: number
  /** Total columns cataloged */
  totalColumns: number
  /** Tables with descriptions */
  taggedTables: number
  /** Columns marked as PII */
  piiColumns: number
  /** Tables with assigned stewards */
  stewardedTables: number
  /** Catalog coverage percentage (0-100) */
  coveragePercentage: number
  /** PII coverage percentage (0-100) */
  piiCoveragePercentage: number
}

/**
 * PII statistics breakdown
 */
export interface PIIStats {
  /** Total columns with PII */
  totalPII: number
  /** Breakdown by PII type */
  byType: Record<PIIType, number>
  /** Breakdown by confidence level */
  byConfidence: Record<PIIConfidence, number>
  /** Manually confirmed PII columns */
  confirmedCount: number
  /** Unconfirmed PII columns */
  unconfirmedCount: number
}

/**
 * Tag usage statistics
 */
export interface TagStats {
  /** Tag ID */
  tagId: CatalogTagId
  /** Tag name */
  tagName: string
  /** Number of tables using this tag */
  tableCount: number
  /** Number of columns using this tag */
  columnCount: number
  /** Total usage count */
  totalUsage: number
}

// ============================================================================
// Sync and Import Types
// ============================================================================

/**
 * Catalog sync options
 */
export interface CatalogSyncOptions {
  /** Connection to sync from */
  connectionId: string
  /** Schemas to include (empty = all) */
  includedSchemas?: string[]
  /** Schemas to exclude */
  excludedSchemas?: string[]
  /** Whether to run PII detection */
  detectPII?: boolean
  /** Whether to preserve existing descriptions */
  preserveDescriptions?: boolean
  /** Whether to preserve existing tags */
  preserveTags?: boolean
}

/**
 * Catalog sync result
 */
export interface CatalogSyncResult {
  /** Number of tables added to catalog */
  tablesAdded: number
  /** Number of tables updated in catalog */
  tablesUpdated: number
  /** Number of tables removed from catalog */
  tablesRemoved: number
  /** Number of columns added to catalog */
  columnsAdded: number
  /** Number of columns updated in catalog */
  columnsUpdated: number
  /** Number of columns removed from catalog */
  columnsRemoved: number
  /** PII detection results (if enabled) */
  piiDetections?: PIIDetectionResult[]
  /** Sync duration in milliseconds */
  duration: number
  /** Any errors encountered */
  errors?: string[]
}

// ============================================================================
// Stewardship Types
// ============================================================================

/**
 * Data steward assignment
 */
export interface StewardAssignment {
  /** Table catalog ID */
  tableCatalogId: TableCatalogId
  /** Steward user ID */
  stewardUserId: UserId
  /** When assignment was made */
  assignedAt: Date
  /** User who made the assignment */
  assignedBy: UserId
}

/**
 * Steward responsibilities summary
 */
export interface StewardResponsibilities {
  /** Steward user ID */
  stewardUserId: UserId
  /** Tables under stewardship */
  tables: TableCatalogEntry[]
  /** Total table count */
  tableCount: number
  /** Total column count across all tables */
  columnCount: number
  /** Number of tables with PII */
  piiTableCount: number
}
