import type { ResultDisplayMode } from '../../lib/query-result-storage'
import type { QueryEditableColumn, QueryEditableMetadata, QueryResultRow } from '../../store/query-store'
import type { CellValue, EditableTableContext, ExportOptions, TableColumn, TableRow } from '../../types/table'

export interface QueryResultsTableProps {
  resultId: string
  columns: string[]
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
  metadata?: QueryEditableMetadata | null
  query: string
  connectionId?: string
  executionTimeMs: number
  rowCount: number
  executedAt: Date
  affectedRows: number
  // Phase 2: Chunking metadata
  isLarge?: boolean
  chunkingEnabled?: boolean
  displayMode?: ResultDisplayMode
  // Pagination metadata
  totalRows?: number
  hasMore?: boolean
  offset?: number
  // Pagination callback
  onPageChange?: (limit: number, offset: number) => void
}

export interface ToolbarProps {
  context: EditableTableContext
  rowCount: number
  columnCount: number
  executionTimeMs: number
  executedAt: Date
  dirtyCount: number
  canSave: boolean
  saving: boolean
  onSave: () => void
  onExport: (options: ExportOptions) => Promise<void>
  metadata?: QueryEditableMetadata | null
  hasEditableColumns?: boolean
  onDiscardChanges?: () => void
  onJumpToFirstError?: () => void
  canDeleteRows?: boolean
  onDeleteSelected?: () => void
  canInsertRows?: boolean
  onAddRow?: () => void
  databases?: string[]
  currentDatabase?: string
  onSelectDatabase?: (database: string) => void
  databaseLoading?: boolean
  databaseSwitching?: boolean
}

export interface ColumnDisplayTraits {
  minWidth: number
  maxWidth?: number
  preferredWidth?: number
  longText: boolean
  wrapContent: boolean
  clipContent: boolean
  monospace: boolean
}

export interface ExportButtonProps {
  context: EditableTableContext
  onExport: (options: ExportOptions) => Promise<void>
}

export interface DeleteConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  pendingDeleteCount: number
  isDeleting: boolean
  onConfirm: () => void
  onCancel: () => void
}

export interface StatusBarProps {
  totalRows?: number
  rowCount: number
  columnCount: number
  affectedRows: number
  executionTimeMs: number
  dirtyRowCount: number
}

export type {
  CellValue,
  EditableTableContext,
  ExportOptions,
  QueryEditableColumn,
  QueryEditableMetadata,
  QueryResultRow,
  ResultDisplayMode,
  TableColumn,
  TableRow}
