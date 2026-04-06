/**
 * Shared types for query stores
 * These types are used across query-editor-store, query-engine-store, and query-history-store
 */

export type QueryTabType = 'sql' | 'ai'

export interface QueryTab {
  id: string
  title: string
  type: QueryTabType
  content: string
  isDirty: boolean
  isExecuting: boolean
  executionStartTime?: Date
  lastExecuted?: Date
  connectionId?: string // Per-tab connection support (single-DB mode)
  selectedConnectionIds?: string[] // Multi-select connections (multi-DB mode)
  environmentSnapshot?: string | null // Capture environment filter at creation
  aiSessionId?: string
}

export interface QueryEditableColumn {
  name: string
  resultName: string
  dataType: string
  editable: boolean
  primaryKey: boolean
  foreignKey?: {
    table: string
    column: string
    schema?: string
  }
  hasDefault?: boolean
  defaultValue?: unknown
  defaultExpression?: string
  autoNumber?: boolean
  timeZone?: boolean
  precision?: number
}

export interface QueryEditableMetadata {
  enabled: boolean
  reason?: string
  schema?: string
  table?: string
  primaryKeys: string[]
  columns: QueryEditableColumn[]
  pending?: boolean
  jobId?: string
  job_id?: string
  capabilities?: {
    canInsert: boolean
    canUpdate: boolean
    canDelete: boolean
    reason?: string
  }
}

export interface QueryResultRow extends Record<string, unknown> {
  __rowId: string
  __isNewRow?: boolean
}

export interface QueryResult {
  id: string
  tabId: string
  columns: string[]
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
  rowCount: number
  affectedRows: number
  executionTime: number
  error?: string
  timestamp: Date
  editable?: QueryEditableMetadata | null
  query: string
  connectionId?: string
  isLarge?: boolean // true if stored in IndexedDB
  rowsLoaded?: number // number of rows loaded in memory (for large results)
  // Phase 2: Chunking support
  chunkingEnabled?: boolean
  loadedChunks?: Set<number>
  totalChunks?: number
  displayMode?: import('@/lib/query-result-storage').ResultDisplayMode
  // Data processing state (for large datasets)
  isProcessing?: boolean
  processingProgress?: number // 0-100
  // Pagination metadata (from backend)
  totalRows?: number // Total unpaginated rows
  pagedRows?: number // Rows in current page
  hasMore?: boolean // More data available
  offset?: number // Current offset
  limit?: number // Page size
  // Multi-database query metadata
  connectionsUsed?: string[] // Connection aliases used in federated query
  federationStrategy?: string // Query execution strategy (federated, union, etc.)
}

export interface NormalisedRowsResult {
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
}
