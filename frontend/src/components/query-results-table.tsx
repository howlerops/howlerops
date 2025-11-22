import { AlertCircle, CheckCircle2, Clock, Database, Download, Inbox, Loader2, Plus,Save, Trash2 } from 'lucide-react'
import { useCallback,useEffect, useMemo, useRef, useState } from 'react'

import { toast } from '../hooks/use-toast'
import { wailsEndpoints } from '../lib/wails-api'
import { useConnectionStore } from '../store/connection-store'
import { type QueryEditableColumn,QueryEditableMetadata, QueryResultRow, useQueryStore } from '../store/query-store'
import type { CellValue, EditableTableContext } from '../types/table'
import { ExportOptions, TableColumn, TableRow } from '../types/table'
import { EditableTable } from './editable-table/editable-table'
import { JsonRowViewerSidebarV2 } from './json-row-viewer-sidebar-v2'
import { PaginationControls } from './pagination-controls'
import { Button } from './ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog'

interface QueryResultsTableProps {
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
  displayMode?: import('../lib/query-result-storage').ResultDisplayMode
  // Pagination metadata
  totalRows?: number
  hasMore?: boolean
  offset?: number
  // Pagination callback
  onPageChange?: (limit: number, offset: number) => void
}

interface ToolbarProps {
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

const inferColumnType = (dataType?: string): TableColumn['type'] => {
  if (!dataType) return 'text'
  const normalized = dataType.toLowerCase()

  if (normalized.includes('int') || normalized.includes('numeric') || normalized.includes('decimal') || normalized.includes('double') || normalized.includes('real')) {
    return 'number'
  }
  if (normalized.includes('bool')) {
    return 'boolean'
  }
  if (normalized.includes('timestamp') || normalized.includes('time')) {
    return 'datetime'
  }
  if (normalized.includes('date')) {
    return 'date'
  }

  return 'text'
}

type EditableColumnMeta = QueryEditableMetadata['columns'] extends Array<infer C> ? C : never

interface ColumnDisplayTraits {
  minWidth: number
  maxWidth?: number
  preferredWidth?: number
  longText: boolean
  wrapContent: boolean
  clipContent: boolean
  monospace: boolean
}

const deriveColumnDisplayTraits = (
  columnName: string,
  metaColumn: EditableColumnMeta | undefined,
  columnType: TableColumn['type']
): ColumnDisplayTraits => {
  const normalizedName = columnName.toLowerCase()
  const dataType = metaColumn?.dataType?.toLowerCase() ?? ''
  const precision = typeof metaColumn?.precision === 'number' ? metaColumn.precision : undefined

  const isUUIDLike =
    dataType.includes('uuid') ||
    normalizedName.endsWith('_uuid') ||
    normalizedName.endsWith('_guid') ||
    ((normalizedName === 'id' || normalizedName.endsWith('_id')) && (precision ?? 0) >= 24)

  const isJsonLike = dataType.includes('json')
  const isTextLike = dataType.includes('text') || dataType.includes('clob') || dataType.includes('xml')
  const isBinaryLike = dataType.includes('blob') || dataType.includes('binary')
  const isLongCharacter = typeof precision === 'number' && precision >= 512
  const isNumeric = columnType === 'number'
  const isTemporal = columnType === 'datetime' || columnType === 'date'
  const isBoolean = columnType === 'boolean'

  const longText = isJsonLike || isTextLike || isBinaryLike || isLongCharacter
  const wrapContent = isUUIDLike
  const monospace = isUUIDLike || isTemporal || isNumeric

  const minWidth = longText
    ? 220
    : isUUIDLike
      ? 240
      : isTemporal
        ? 200
        : isNumeric
          ? 150
          : isBoolean
            ? 110
            : 120

  const maxWidth = longText
    ? 620
    : isUUIDLike
      ? 460
      : undefined

  const preferredWidth = longText
    ? 520
    : isUUIDLike
      ? 320
      : isTemporal
        ? 280
        : undefined

  return {
    minWidth,
    maxWidth,
    preferredWidth,
    longText,
    wrapContent,
    clipContent: !wrapContent,
    monospace,
  }
}

const formatTimestamp = (value: Date) => value.toLocaleString()

const createRowId = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const serialiseCsvValue = (value: unknown): string => {
  if (value === null || value === undefined) {
    return ''
  }

  const stringValue = String(value)
  if (stringValue.includes('"') || stringValue.includes(',') || stringValue.includes('\n')) {
    return `"${stringValue.replace(/"/g, '""')}"`
  }
  return stringValue
}

const ExportButton = ({ context, onExport }: { context: EditableTableContext; onExport: (options: ExportOptions) => Promise<void> }) => {
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [isExporting, setIsExporting] = useState(false)
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    includeHeaders: true,
    selectedOnly: false,
  })

  const handleExport = async () => {
    setIsExporting(true)
    try {
      await onExport(exportOptions)
      // Keep dialog open briefly to show success message (handled via toast)
      setTimeout(() => setShowExportDialog(false), 500)
    } catch {
      // Error is handled by onExport, just reset state
      setIsExporting(false)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowExportDialog(true)}
        className="gap-2"
      >
        <Download className="h-4 w-4" />
        Export
      </Button>

      <Dialog open={showExportDialog} onOpenChange={setShowExportDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Export Data</DialogTitle>
            <DialogDescription>
              Export will fetch ALL results from the database (up to 1M rows). Configure options and download to your Downloads folder.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Format selection */}
            <div className="space-y-2">
              <label className="text-sm font-medium">Format</label>
              <select
                value={exportOptions.format}
                onChange={(e) => setExportOptions(prev => ({ ...prev, format: e.target.value as 'csv' | 'json' }))}
                className="w-full px-3 py-2 border border-input rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring"
              >
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
              </select>
            </div>

            {/* Options */}
            <div className="space-y-3">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={exportOptions.includeHeaders}
                  onChange={(e) => setExportOptions(prev => ({ ...prev, includeHeaders: e.target.checked }))}
                  className="rounded border-input focus:ring-2 focus:ring-ring"
                />
                <span className="text-sm">Include headers</span>
              </label>
              {context.state.selectedRows.length > 0 && (
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={exportOptions.selectedOnly}
                    onChange={(e) => setExportOptions(prev => ({ ...prev, selectedOnly: e.target.checked }))}
                    className="rounded border-input focus:ring-2 focus:ring-ring"
                  />
                  <span className="text-sm">
                    Selected only ({context.state.selectedRows.length} rows)
                  </span>
                </label>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowExportDialog(false)}
              disabled={isExporting}
            >
              Cancel
            </Button>
            <Button onClick={handleExport} disabled={isExporting}>
              {isExporting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Exporting...
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 mr-2" />
                  Export to Downloads
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

const QueryResultsToolbar = ({
  context,
  rowCount: _rowCount,
  columnCount: _columnCount,
  executionTimeMs: _executionTimeMs,
  executedAt,
  dirtyCount,
  canSave,
  saving,
  onSave,
  onExport,
  metadata,
  onDiscardChanges,
  onJumpToFirstError,
  canDeleteRows,
  onDeleteSelected,
  canInsertRows,
  onAddRow,
  databases: _databases, // Database selector currently disabled
  currentDatabase: _currentDatabase,
  onSelectDatabase: _onSelectDatabase,
  databaseLoading: _databaseLoading,
  databaseSwitching: _databaseSwitching,
}: ToolbarProps) => {
  // Search is currently disabled in UI
  const _handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    context.actions.updateGlobalFilter(event.target.value)
  }

  const invalidCellsCount = context.state.invalidCells.size
  const dirtyRowCount = dirtyCount ?? context.state.dirtyRows.size
  const hasValidationErrors = invalidCellsCount > 0
  const canSaveWithValidation = canSave && !hasValidationErrors
  const selectedCount = context.state.selectedRows.length

  return (
    <div className="flex flex-col gap-2 border-b border-gray-200 bg-background px-1 py-1">
      {/* Show non-editable reason if applicable */}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-1 items-center gap-3 min-w-[220px]">
          {/* {onSelectDatabase && databases && databases.length > 1 && (
            <div className="flex items-center gap-2">
              <Select
                value={currentDatabase ?? ''}
                onValueChange={onSelectDatabase}
                disabled={databaseLoading || databaseSwitching}
              >
                <SelectTrigger className="h-9 w-48 text-sm">
                  <SelectValue placeholder={databaseLoading ? 'Loading databases…' : 'Select database'} />
                </SelectTrigger>
                <SelectContent>
                  {databases.map((dbName) => (
                    <SelectItem key={dbName} value={dbName}>
                      {dbName}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {(databaseLoading || databaseSwitching) && (
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              )}
            </div>
          )} */}
          {/* <div className="relative w-full max-w-xs">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={context.state.globalFilter}
              onChange={handleSearchChange}
              placeholder="Search…"
              className="h-9 w-full pl-9"
            />
          </div> */}
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground">{formatTimestamp(executedAt)}</span>

          {/* Unsaved changes indicator */}
          {dirtyRowCount > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 bg-accent/10 border border-accent rounded text-xs">
              <span className="text-accent-foreground">
                {dirtyRowCount} unsaved{dirtyRowCount === 1 ? '' : ''}
              </span>
            </div>
          )}

          {/* Validation errors indicator */}
          {invalidCellsCount > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 bg-destructive/10 border border-destructive rounded text-xs">
              <AlertCircle className="h-3 w-3 text-destructive" />
              <span className="text-destructive">
                {invalidCellsCount} error{invalidCellsCount === 1 ? '' : 's'}
              </span>
              {onJumpToFirstError && (
                <button
                  onClick={onJumpToFirstError}
                  className="text-destructive hover:text-destructive/80 underline ml-1"
                >
                  Jump
                </button>
              )}
            </div>
          )}

          {/* Add row */}
          {canInsertRows && onAddRow && (
            <Button
              variant="outline"
              size="sm"
              onClick={onAddRow}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              Add Row
            </Button>
          )}

          {/* Delete selected rows */}
          {canDeleteRows && selectedCount > 0 && onDeleteSelected && (
            <Button
              variant="destructive"
              size="icon"
              onClick={onDeleteSelected}
              className="h-9 w-9"
              title={`Delete ${selectedCount} selected row${selectedCount === 1 ? '' : 's'}`}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
          
          {/* Export Button */}
          <ExportButton context={context} onExport={onExport} />
          
          {metadata?.enabled && (
            <>
              {/* Discard Changes Button */}
              {dirtyRowCount > 0 && onDiscardChanges && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onDiscardChanges}
                  disabled={saving}
                  className="gap-2"
                >
                  Discard Changes
                </Button>
              )}
              
              {/* Save Button */}
              <Button
                size="sm"
                onClick={onSave}
                disabled={!canSaveWithValidation}
                className="gap-2"
                title={hasValidationErrors ? `Cannot save: ${invalidCellsCount} validation errors` : undefined}
              >
                {saving ? (
                  <span className="flex items-center gap-2">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-b-transparent border-current" />
                    Saving…
                  </span>
                ) : (
                  <>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </>
                )}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

const buildPrimaryKeyMap = (
  row: QueryResultRow,
  metadata?: QueryEditableMetadata | null,
  columnsLookup?: Record<string, string>
): Record<string, unknown> | null => {
  if (!metadata?.primaryKeys?.length) {
    return null
  }

  const primaryKey: Record<string, unknown> = {}
  let allPresent = true

  metadata.primaryKeys.forEach((pk) => {
    const resultColumnName =
      columnsLookup?.[pk.toLowerCase()] ??
      metadata.columns?.find((col) => (col.name ?? col.resultName)?.toLowerCase() === pk.toLowerCase())?.resultName ??
      pk
    const value = row[resultColumnName]
    if (value === undefined) {
      allPresent = false
    } else {
      primaryKey[pk] = value
    }
  })

  return allPresent ? primaryKey : null
}

const buildColumnsLookup = (metadata?: QueryEditableMetadata | null) => {
  const lookup: Record<string, string> = {}
  metadata?.columns?.forEach((column) => {
    const baseName = column.name ?? column.resultName
    if (!baseName) return
    const key = baseName.toLowerCase()
    const resultName = column.resultName ?? column.name ?? baseName
    lookup[key] = resultName
  })
  return lookup
}

const buildMetadataLookup = (metadata?: QueryEditableMetadata | null) => {
  const map = new Map<string, QueryEditableColumn>()
  metadata?.columns?.forEach((column) => {
    const key = (column.resultName ?? column.name ?? '').toLowerCase()
    if (!key) {
      return
    }
    map.set(key, column)
  })
  return map
}

export const QueryResultsTable = ({
  resultId,
  columns = [],
  rows,
  originalRows,
  metadata,
  query,
  connectionId,
  executionTimeMs,
  rowCount,
  executedAt,
  affectedRows,
  isLarge = false,
  chunkingEnabled = false,
  displayMode,
  totalRows,
  hasMore: _hasMore = false, // Reserved for future infinite scroll feature
  offset = 0,
  onPageChange,
}: QueryResultsTableProps) => {
  const columnNames = useMemo(
    () => (Array.isArray(columns) ? columns : []),
    [columns]
  )
  const [dirtyRowIds, setDirtyRowIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(100)
  const [isLoadingPage, setIsLoadingPage] = useState(false)

  // Sync current page with offset from backend
  useEffect(() => {
    if (offset !== undefined && pageSize > 0) {
      const calculatedPage = Math.floor(offset / pageSize) + 1
      if (calculatedPage !== currentPage) {
        setCurrentPage(calculatedPage)
      }
    }
  }, [offset, pageSize, currentPage])

  // Reset to page 1 when query changes
  useEffect(() => {
    setCurrentPage(1)
    setIsLoadingPage(false)
  }, [resultId])

  // Pagination handlers
  const handlePageChange = useCallback(async (newPage: number) => {
    if (!onPageChange || isLoadingPage) return

    const newOffset = (newPage - 1) * pageSize
    setIsLoadingPage(true)
    setCurrentPage(newPage)

    try {
      await onPageChange(pageSize, newOffset)
    } catch (error) {
      console.error('Page change failed:', error)
      toast({
        title: 'Page change failed',
        description: error instanceof Error ? error.message : 'Failed to load page',
        variant: 'destructive'
      })
      // Revert to previous page on error
      setCurrentPage(Math.floor(offset / pageSize) + 1)
    } finally {
      setIsLoadingPage(false)
    }
  }, [onPageChange, pageSize, offset, isLoadingPage])

  const handlePageSizeChange = useCallback(async (newPageSize: number) => {
    if (!onPageChange || isLoadingPage) return

    // Calculate what page we should be on to show similar rows
    const currentFirstRow = (currentPage - 1) * pageSize
    const newPage = Math.floor(currentFirstRow / newPageSize) + 1

    setPageSize(newPageSize)
    setIsLoadingPage(true)
    setCurrentPage(newPage)

    try {
      await onPageChange(newPageSize, (newPage - 1) * newPageSize)
    } catch (error) {
      console.error('Page size change failed:', error)
      toast({
        title: 'Page size change failed',
        description: error instanceof Error ? error.message : 'Failed to change page size',
        variant: 'destructive'
      })
      // Revert to previous page size on error
      setPageSize(pageSize)
      setCurrentPage(Math.floor(offset / pageSize) + 1)
    } finally {
      setIsLoadingPage(false)
    }
  }, [onPageChange, currentPage, pageSize, offset, isLoadingPage])

  // JSON viewer state
  const [jsonViewerOpen, setJsonViewerOpen] = useState(false)
  const [selectedRowId, setSelectedRowId] = useState<string | null>(null)
  const [selectedRowData, setSelectedRowData] = useState<TableRow | null>(null)
  const [selectedRowIndex, setSelectedRowIndex] = useState<number>(0)
  const [pendingDeleteIds, setPendingDeleteIds] = useState<string[]>([])
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const fetchDatabases = useConnectionStore((state) => state.fetchDatabases)
  const switchConnectionDatabase = useConnectionStore((state) => state.switchDatabase)
  const activeDatabase = useConnectionStore(
    useCallback((state) => {
      if (connectionId) {
        const connection = state.connections.find((conn) => conn.id === connectionId)
        if (connection) {
          return connection.database
        }
      }
      return state.activeConnection?.database
    }, [connectionId])
  )
  const [databaseList, setDatabaseList] = useState<string[]>([])
  const [databaseLoading, setDatabaseLoading] = useState(false)
  const [databaseSelectorEnabled, setDatabaseSelectorEnabled] = useState(false)
  const [isSwitchingDatabase, setIsSwitchingDatabase] = useState(false)

  const updateResultRows = useQueryStore((state) => state.updateResultRows)
  const columnsLookup = useMemo(() => buildColumnsLookup(metadata), [metadata])
  const metadataLookup = useMemo(() => buildMetadataLookup(metadata), [metadata])
  const firstEditableColumnId = useMemo(() => {
    for (const columnName of columnNames) {
      const metaColumn = metadataLookup.get(columnName.toLowerCase())
      if (metaColumn?.editable) {
        return columnName
      }
    }
    return columnNames[0]
  }, [columnNames, metadataLookup])
  const tableContextRef = useRef<EditableTableContext | null>(null)
  const canInsertRows = useMemo(() => {
    if (!metadata?.enabled || !metadata.table) {
      return false
    }
    return Boolean(metadata.capabilities?.canInsert)
  }, [metadata])
  const canDeleteRows = useMemo(() => {
    if (!metadata?.enabled || !metadata?.table || !metadata?.primaryKeys?.length) {
      return false
    }
    return Boolean(metadata.capabilities?.canDelete)
  }, [metadata])

  useEffect(() => {
    setDirtyRowIds([])
    tableContextRef.current?.actions.clearDirtyRows?.()
    tableContextRef.current?.actions.resetTable?.()
  }, [resultId])

  useEffect(() => {
    if (!connectionId) {
      setDatabaseList([])
      setDatabaseSelectorEnabled(false)
      return
    }

    let cancelled = false
    setDatabaseLoading(true)

    fetchDatabases(connectionId)
      .then((databases) => {
        if (cancelled) {
          return
        }
        setDatabaseList(databases)
        setDatabaseSelectorEnabled(databases.length > 1)
      })
      .catch((error) => {
        if (cancelled) {
          return
        }
        setDatabaseList([])
        setDatabaseSelectorEnabled(false)
        if (error instanceof Error && !error.message.toLowerCase().includes('not supported')) {
          console.warn('Failed to load databases for connection', error)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setDatabaseLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [connectionId, fetchDatabases])

  const resolveCurrentRows = useCallback((): QueryResultRow[] => {
    const contextRows = tableContextRef.current?.data as QueryResultRow[] | undefined
    const source = contextRows ?? rows
    return source.map((row) => ({ ...row }))
  }, [rows])

  const handleAddRow = useCallback(() => {
    if (!canInsertRows) {
      return
    }

    const newRowId = createRowId()
    const emptyRow: QueryResultRow = {
      __rowId: newRowId,
      __isNewRow: true,
    }

    columnNames.forEach((columnName) => {
      const key = columnName.toLowerCase()
      const metaColumn = metadataLookup.get(key)
      if (metaColumn?.defaultValue !== undefined) {
        emptyRow[columnName] = metaColumn.defaultValue as CellValue
      } else {
        emptyRow[columnName] = undefined
      }
    })

    const nextRows = [...rows, emptyRow]
    updateResultRows(resultId, nextRows, originalRows)
    setDirtyRowIds((prev) => [...new Set([...prev, newRowId])])

    const targetColumn = firstEditableColumnId || columnNames[0]
    if (targetColumn && tableContextRef.current?.actions?.startEditing) {
      requestAnimationFrame(() => {
        tableContextRef.current?.actions.startEditing(newRowId, targetColumn, emptyRow[targetColumn] as CellValue)
      })
    }
  }, [
    canInsertRows,
    columnNames,
    metadataLookup,
    rows,
    originalRows,
    resultId,
    updateResultRows,
    firstEditableColumnId,
  ])

  const handleDatabaseSelection = useCallback(async (nextDatabase: string) => {
    if (!connectionId || !nextDatabase || nextDatabase === (activeDatabase ?? '')) {
      return
    }

    setIsSwitchingDatabase(true)
    try {
      await switchConnectionDatabase(connectionId, nextDatabase)
      setDirtyRowIds([])
      setPendingDeleteIds([])
      tableContextRef.current?.actions.clearDirtyRows?.()
      tableContextRef.current?.actions.clearInvalidCells?.()
      tableContextRef.current?.actions.resetTable?.()
      updateResultRows(resultId, [], {})
      setJsonViewerOpen(false)
      setSelectedRowId(null)
      setSelectedRowData(null)

      toast({
        title: 'Database switched',
        description: `Active database is now ${nextDatabase}.`,
        variant: 'default'
      })
    } catch (error) {
      toast({
        title: 'Failed to switch database',
        description: error instanceof Error ? error.message : 'Unable to switch database',
        variant: 'destructive'
      })
    } finally {
      setIsSwitchingDatabase(false)
    }
  }, [
    connectionId,
    activeDatabase,
    switchConnectionDatabase,
    setDirtyRowIds,
    setPendingDeleteIds,
    updateResultRows,
    resultId
  ])

  const handleSave = useCallback(async () => {
    const currentRows = resolveCurrentRows()

    if (!metadata?.enabled || !metadata || rows.length === 0) {
      return
    }
    if (!connectionId) {
      toast({
        title: 'No active connection',
        description: 'Please select a connection and try again.',
        variant: 'destructive'
      })
      return
    }
    if (dirtyRowIds.length === 0) {
      return
    }

    if (tableContextRef.current) {
      const isValid = tableContextRef.current.actions.validateAllCells()
      if (!isValid) {
        const invalidCells = tableContextRef.current.actions.getInvalidCells()
        toast({
          title: 'Validation errors',
          description: `Cannot save: ${invalidCells.length} validation error${invalidCells.length === 1 ? '' : 's'} found. Please fix all errors before saving.`,
          variant: 'destructive'
        })
        return
      }
    }

    setSaving(true)

    try {
      const updatedRows = [...currentRows]
      const newOriginalRows: Record<string, QueryResultRow> = { ...originalRows }

      for (const rowId of dirtyRowIds) {
        const rowIndex = updatedRows.findIndex((row) => row.__rowId === rowId)
        if (rowIndex === -1) {
          continue
        }

        const currentRow = updatedRows[rowIndex]
        const originalRow = originalRows[rowId]
        const isNewRow = currentRow.__isNewRow || !originalRow

        if (isNewRow) {
          const insertValues: Record<string, unknown> = {}
          columnNames.forEach((columnName) => {
            const value = currentRow[columnName]
            if (value !== undefined) {
              insertValues[columnName] = value
            }
          })

          const response = await wailsEndpoints.queries.insertRow({
            connectionId,
            query,
            columns: columnNames,
            schema: metadata?.schema,
            table: metadata?.table,
            values: insertValues,
          })

          if (!response.success) {
            throw new Error(response.message || 'Failed to insert row')
          }

          const returnedValues = response.row || {}
          const persistedRow: QueryResultRow = { ...currentRow, __isNewRow: false }
          columnNames.forEach((columnName) => {
            if (returnedValues[columnName] !== undefined) {
              persistedRow[columnName] = returnedValues[columnName]
            }
          })

          updatedRows[rowIndex] = persistedRow
          const snapshot = { ...persistedRow }
          delete snapshot.__isNewRow
          newOriginalRows[rowId] = snapshot
          continue
        }

        const primaryKey = buildPrimaryKeyMap(originalRow, metadata, columnsLookup)
        if (!primaryKey) {
          throw new Error('Unable to determine primary key for the selected row.')
        }

        const changedValues: Record<string, unknown> = {}
        columnNames.forEach((columnName) => {
          const currentValue = currentRow[columnName]
          const originalValue = originalRow[columnName]

          const valuesAreEqual =
            currentValue === originalValue ||
            (currentValue == null && originalValue == null)

          const metaColumn = metadata?.columns?.find((col) => {
            const candidate = col.resultName ?? col.name
            return candidate ? candidate.toLowerCase() === columnName.toLowerCase() : false
          })

          if (!valuesAreEqual && metaColumn?.editable) {
            changedValues[columnName] = currentValue
          }
        })

        if (Object.keys(changedValues).length === 0) {
          continue
        }

        const response = await wailsEndpoints.queries.updateRow({
          connectionId,
          query,
          columns: columnNames,
          schema: metadata?.schema,
          table: metadata?.table,
          primaryKey,
          values: changedValues,
        })

        if (!response.success) {
          throw new Error(response.message || 'Failed to save changes')
        }

        const snapshot = { ...currentRow }
        delete snapshot.__isNewRow
        newOriginalRows[rowId] = snapshot
      }

      updateResultRows(resultId, updatedRows, newOriginalRows)
      setDirtyRowIds([])
      tableContextRef.current?.actions.clearDirtyRows()
      tableContextRef.current?.actions.clearInvalidCells()

      toast({
        title: 'Success',
        description: 'Changes saved successfully.',
        variant: 'default'
      })
    } catch (error) {
      toast({
        title: 'Save failed',
        description: error instanceof Error ? error.message : 'Failed to save changes',
        variant: 'destructive'
      })
    } finally {
      setSaving(false)
    }
  }, [
    connectionId,
    columnNames,
    columnsLookup,
    dirtyRowIds,
    metadata,
    originalRows,
    query,
    resultId,
    resolveCurrentRows,
    rows,
    updateResultRows,
  ])

  const handleRequestDelete = useCallback(() => {
    const selected = tableContextRef.current?.state.selectedRows ?? []
    if (!selected.length) {
      return
    }
    setPendingDeleteIds(selected)
    setShowDeleteDialog(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (!canDeleteRows || pendingDeleteIds.length === 0) {
      setShowDeleteDialog(false)
      return
    }

    if (!connectionId || !metadata?.table) {
      toast({
        title: 'Delete failed',
        description: 'Missing connection or table information for deletion.',
        variant: 'destructive'
      })
      setShowDeleteDialog(false)
      return
    }

    setIsDeleting(true)

    try {
      const currentRows = resolveCurrentRows()
      const rowsToDelete = currentRows.filter(row => pendingDeleteIds.includes(row.__rowId))

      if (!rowsToDelete.length) {
        throw new Error('No matching rows found to delete.')
      }

      const primaryKeysPayload: Record<string, unknown>[] = []

      rowsToDelete.forEach((row) => {
        const originalRow = originalRows[row.__rowId]
        if (!originalRow) {
          return
        }
        const primaryKey = buildPrimaryKeyMap(originalRow, metadata, columnsLookup)
        if (!primaryKey) {
          throw new Error('Unable to determine primary key for one of the selected rows.')
        }
        primaryKeysPayload.push(primaryKey)
      })

      if (primaryKeysPayload.length > 0) {
        const response = await wailsEndpoints.queries.deleteRows({
          connectionId,
          query,
          columns: columnNames,
          schema: metadata?.schema,
          table: metadata?.table,
          primaryKeys: primaryKeysPayload,
        })

        if (!response.success) {
          throw new Error(response.message || 'Failed to delete selected rows.')
        }
      }

      const remainingRows = currentRows.filter(row => !pendingDeleteIds.includes(row.__rowId))
      const updatedOriginalRows = { ...originalRows }
      pendingDeleteIds.forEach(id => {
        delete updatedOriginalRows[id]
      })

      updateResultRows(resultId, remainingRows, updatedOriginalRows)
      setDirtyRowIds(prev => prev.filter(id => !pendingDeleteIds.includes(id)))
      tableContextRef.current?.actions.selectAllRows(false)
      tableContextRef.current?.actions.clearInvalidCells()

      if (selectedRowId && pendingDeleteIds.includes(selectedRowId)) {
        setJsonViewerOpen(false)
        setSelectedRowId(null)
        setSelectedRowData(null)
      }

      toast({
        title: 'Rows deleted',
        description: `${pendingDeleteIds.length} row${pendingDeleteIds.length === 1 ? '' : 's'} deleted successfully.`,
        variant: 'default'
      })

      setPendingDeleteIds([])
    } catch (error) {
      toast({
        title: 'Delete failed',
        description: error instanceof Error ? error.message : 'Failed to delete selected rows.',
        variant: 'destructive'
      })
    } finally {
      setIsDeleting(false)
      setShowDeleteDialog(false)
    }
  }, [
    canDeleteRows,
    columnsLookup,
    columnNames,
    connectionId,
    metadata,
    originalRows,
    pendingDeleteIds,
    query,
    resolveCurrentRows,
    resultId,
    selectedRowId,
    updateResultRows,
  ])

  const tableColumns: TableColumn[] = useMemo(() => {
    return columnNames.map<TableColumn>((columnName) => {
      const metaColumn = metadataLookup.get(columnName.toLowerCase())
      const columnType = inferColumnType(metaColumn?.dataType)
      const traits = deriveColumnDisplayTraits(columnName, metaColumn, columnType)

      return {
        id: columnName,
        accessorKey: columnName,
        header: columnName,
        type: columnType,
        editable: Boolean(metadata?.enabled && metaColumn?.editable),
        sortable: true,
        filterable: true,
        minWidth: traits.minWidth,
        maxWidth: traits.maxWidth,
        preferredWidth: traits.preferredWidth,
        longText: traits.longText,
        wrapContent: traits.wrapContent,
        clipContent: traits.clipContent,
        monospace: traits.monospace,
        hasDefault: Boolean(metaColumn?.hasDefault),
        defaultLabel: metaColumn?.defaultExpression || '[default]',
        defaultValue: metaColumn?.defaultValue,
        autoNumber: Boolean(metaColumn?.autoNumber),
        isPrimaryKey: Boolean(metaColumn?.primaryKey),
      }
    })
  }, [columnNames, metadataLookup, metadata?.enabled])

  // const handleExportCsv = useCallback(() => {
  //   const currentRows = resolveCurrentRows()
  //   const header = columns.join(',')
  //   const records = currentRows.map((row) =>
  //     columnNames.map((column) => serialiseCsvValue(row[column])).join(',')
  //   )

  //   const csv = [header, ...records].join('\n')
  //   const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  //   const url = URL.createObjectURL(blob)
  //   const link = document.createElement('a')
  //   link.href = url
  //   link.setAttribute('download', `query-results-${Date.now()}.csv`)
  //   document.body.appendChild(link)
  //   link.click()
  //   document.body.removeChild(link)
  //   URL.revokeObjectURL(url)
  // }, [columnNames, resolveCurrentRows])

  const handleExport = useCallback(async (options: ExportOptions) => {
    if (!connectionId) {
      toast({
        title: 'Export failed',
        description: 'No active connection',
        variant: 'destructive',
      })
      return
    }

    try {
      // For selected rows only, export the current loaded data
      if (options.selectedOnly && tableContextRef.current?.state.selectedRows.length && tableContextRef.current.state.selectedRows.length > 0) {
        const currentRows = resolveCurrentRows()
        const selectedIds = tableContextRef.current.state.selectedRows
        const dataToExport = currentRows.filter(row => selectedIds.includes(row.__rowId!))

        const timestamp = Date.now()
        let filename: string
        let content: string

        if (options.format === 'csv') {
          filename = `query-results-${timestamp}.csv`
          const header = options.includeHeaders ? columnNames.join(',') : ''
          const records = dataToExport.map((row) =>
            columnNames.map((column) => serialiseCsvValue(row[column])).join(',')
          )
          content = options.includeHeaders ? [header, ...records].join('\n') : records.join('\n')
        } else {
          filename = `query-results-${timestamp}.json`
          content = JSON.stringify(dataToExport, null, 2)
        }

        const { SaveToDownloads } = await import('../../wailsjs/go/main/App')
        const filePath = await SaveToDownloads(filename, content)

        toast({
          title: 'Export successful',
          description: `File saved to: ${filePath}`,
          variant: 'default',
        })
        return
      }

      // For full export, re-query with isExport=true to get ALL rows
      toast({
        title: 'Export starting',
        description: 'Fetching all results from database...',
        variant: 'default',
      })

      const { wailsApiClient } = await import('../lib/wails-api')
      const result = await wailsApiClient.executeQuery(
        connectionId,
        query,
        0, // limit=0 triggers unlimited export (backend handles max 1M rows)
        0, // offset
        300, // 5 minute timeout
        true // isExport = true
      )

      if (!result.success || !result.data) {
        throw new Error(result.message || 'Failed to fetch export data')
      }

      // Prepare export data
      const exportRows = result.data.rows || []
      const exportColumns = result.data.columns || []

      // Show warning if hitting max export limit (1M rows)
      if (exportRows.length >= 1000000) {
        toast({
          title: 'Export limit reached',
          description: 'Export limited to 1 million rows. Consider filtering your query.',
          variant: 'default',
        })
      }

      const timestamp = Date.now()
      let filename: string
      let content: string

      if (options.format === 'csv') {
        filename = `query-results-${timestamp}.csv`
        const header = options.includeHeaders ? exportColumns.join(',') : ''
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Backend returns rows as any[] arrays from Wails
        const records = exportRows.map((row: any[]) =>
          row.map((cell) => serialiseCsvValue(cell)).join(',')
        )
        content = options.includeHeaders ? [header, ...records].join('\n') : records.join('\n')
      } else {
        filename = `query-results-${timestamp}.json`
        // Convert rows array to objects for JSON export
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Backend returns rows as any[] arrays and cells as any from Wails
        const jsonData = exportRows.map((row: any[]) => {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Cell values are unknown types from database
          const obj: Record<string, any> = {}
          exportColumns.forEach((col: string, idx: number) => {
            obj[col] = row[idx]
          })
          return obj
        })
        content = JSON.stringify(jsonData, null, 2)
      }

      const { SaveToDownloads } = await import('../../wailsjs/go/main/App')
      const filePath = await SaveToDownloads(filename, content)

      toast({
        title: 'Export successful',
        description: `${exportRows.length.toLocaleString()} rows saved to: ${filePath}`,
        variant: 'default',
      })
    } catch (error) {
      console.error('Failed to export:', error)

      toast({
        title: 'Export failed',
        description: error instanceof Error ? error.message : 'Failed to export data',
        variant: 'destructive',
      })
    }
  }, [connectionId, query, columnNames, resolveCurrentRows])

  const handleDiscardChanges = useCallback(() => {
    if (tableContextRef.current) {
      tableContextRef.current.actions.resetTable()
      setDirtyRowIds([])
    }
  }, [])

  const handleJumpToFirstError = useCallback(() => {
    if (!tableContextRef.current) return
    
    const invalidCells = tableContextRef.current.actions.getInvalidCells()
    if (invalidCells.length === 0) return
    
    const firstError = invalidCells[0]
    // Find the cell element and scroll to it
    const cellElement = document.querySelector(`[data-row-id="${firstError.rowId}"][data-column-id="${firstError.columnId}"]`)
    if (cellElement) {
      cellElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // Focus the cell for better visibility
      ;(cellElement as HTMLElement).focus()
    }
  }, [])

  // JSON viewer handlers
  const handleRowClick = useCallback((rowId: string, rowData: TableRow) => {
    // Find the index of this row in the current rows array
    const rowIndex = rows.findIndex(row => row.__rowId === rowId)

    setSelectedRowId(rowId)
    setSelectedRowData(rowData)
    setSelectedRowIndex(rowIndex >= 0 ? rowIndex : 0)
    setJsonViewerOpen(true)
  }, [rows])

  const handleCloseJsonViewer = useCallback(() => {
    setJsonViewerOpen(false)
    setSelectedRowId(null)
    setSelectedRowData(null)
    setSelectedRowIndex(0)
  }, [])

  const handleNavigateRow = useCallback((direction: 'prev' | 'next') => {
    const newIndex = direction === 'prev' ? selectedRowIndex - 1 : selectedRowIndex + 1

    // Bounds check
    if (newIndex < 0 || newIndex >= rows.length) return

    const newRow = rows[newIndex]
    if (!newRow) return

    setSelectedRowIndex(newIndex)
    setSelectedRowId(newRow.__rowId)
    // Type assertion is safe because QueryResultRow extends Record<string, unknown>
    // and we're using it as TableRow which has the same shape
    setSelectedRowData(newRow as TableRow)
  }, [selectedRowIndex, rows])

  const handleJsonViewerSave = useCallback(async (rowId: string, data: Record<string, CellValue>): Promise<boolean> => {
    if (!connectionId || !metadata?.enabled) return false

    try {
      // Build primary key for the update
      const originalRow = originalRows[rowId]
      if (!originalRow) return false

      const primaryKey = buildPrimaryKeyMap(originalRow, metadata, columnsLookup)
      if (!primaryKey) return false

      // Prepare the update
      const changedValues: Record<string, unknown> = {}
      columnNames.forEach((columnName) => {
        const currentValue = data[columnName]
        const originalValue = originalRow[columnName]

        const valuesAreEqual =
          currentValue === originalValue ||
          (currentValue == null && originalValue == null)

        const metaColumn = metadata?.columns?.find((col) => {
          const candidate = col.resultName ?? col.name
          return candidate ? candidate.toLowerCase() === columnName.toLowerCase() : false
        })

        if (!valuesAreEqual && metaColumn?.editable) {
          changedValues[columnName] = currentValue
        }
      })

      if (Object.keys(changedValues).length === 0) return true

      const response = await wailsEndpoints.queries.updateRow({
        connectionId,
        query,
        columns: columnNames,
        schema: metadata?.schema,
        table: metadata?.table,
        primaryKey,
        values: changedValues,
      })

      if (!response.success) {
        throw new Error(response.message || 'Failed to save changes')
      }

      // Update the table data
      const currentRows = resolveCurrentRows()
      const updatedRows = currentRows.map(row => 
        row.__rowId === rowId 
          ? { ...row, ...changedValues }
          : row
      )
      
      updateResultRows(resultId, updatedRows, originalRows)
      
      return true
    } catch (error) {
      console.error('JSON viewer save failed:', error)
      return false
    }
  }, [connectionId, metadata, originalRows, columnsLookup, columnNames, query, resolveCurrentRows, updateResultRows, resultId])

  const handleCellEdit = useCallback(async (
    rowId: string,
    columnId: string,
    value: unknown
  ): Promise<boolean> => {
    // Early return if no rows or metadata is not properly initialized
    if (!metadata?.enabled || !metadata || rows.length === 0) {
      return false
    }

    try {
      // Find the row and column info
      const currentRows = resolveCurrentRows()
      const currentRow = currentRows.find((row) => row.__rowId === rowId)
      const originalRow = originalRows[rowId]

      if (!currentRow || !originalRow) {
        return false
      }

      // Check if the column is editable
      const metaColumn = metadata?.columns?.find((col) => {
        const candidate = col.resultName ?? col.name
        return candidate ? candidate.toLowerCase() === columnId.toLowerCase() : false
      })

      if (!metaColumn?.editable) {
        return false
      }

      // Build primary key for the update
      const primaryKey = buildPrimaryKeyMap(originalRow, metadata, columnsLookup)
      if (!primaryKey) {
        throw new Error('Unable to determine primary key for the selected row.')
      }

      // Prepare the update
      const updateData = {
        connectionId,
        query,
        columns: columnNames,
        schema: metadata?.schema,
        table: metadata?.table,
        primaryKey,
        values: { [columnId]: value },
      }

      // Optimistic update - update UI immediately
      const updatedRows = currentRows.map(row => 
        row.__rowId === rowId 
          ? { ...row, [columnId]: value }
          : row
      )
      
      // Update the table data optimistically
      updateResultRows(resultId, updatedRows, originalRows)
      
      // Mark row as dirty
      setDirtyRowIds(prev => [...new Set([...prev, rowId])])

      // Save to database in background
      const response = await wailsEndpoints.queries.updateRow(updateData)
      
      if (!response.success) {
        // Rollback on failure
        updateResultRows(resultId, currentRows, originalRows)
        setDirtyRowIds(prev => prev.filter(id => id !== rowId))
        throw new Error(response.message || 'Failed to save changes')
      }

      // Update original rows to reflect the successful save
      const newOriginalRows = { ...originalRows }
      newOriginalRows[rowId] = { ...originalRows[rowId], [columnId]: value }
      
      // Remove from dirty list since it's now saved
      setDirtyRowIds(prev => prev.filter(id => id !== rowId))
      
      return true
    } catch (error) {
      console.error('Cell edit failed:', error)
      return false
    }
  }, [
    metadata,
    rows,
    resolveCurrentRows,
    originalRows,
    columnsLookup,
    connectionId,
    query,
    columnNames,
    resultId,
    updateResultRows,
  ])

  const canSave = Boolean(metadata?.enabled && dirtyRowIds.length > 0 && !saving)

  // Memoize onDirtyChange to prevent infinite re-renders
  const handleDirtyChange = useCallback((ids: string[]) => {
    setDirtyRowIds(ids)
  }, [])

  // Memoize toolbar function to prevent infinite re-renders
  const renderToolbar = useCallback((context: EditableTableContext) => {
    // Capture context in ref outside of render
    tableContextRef.current = context
    
    return (
      <QueryResultsToolbar
        context={context}
        rowCount={rowCount}
        columnCount={columnNames.length}
        executionTimeMs={executionTimeMs}
        executedAt={executedAt}
        dirtyCount={dirtyRowIds.length}
        canSave={canSave}
        saving={saving}
        onSave={handleSave}
        onExport={handleExport}
        metadata={metadata}
        onDiscardChanges={handleDiscardChanges}
        onJumpToFirstError={handleJumpToFirstError}
        canDeleteRows={canDeleteRows}
        onDeleteSelected={canDeleteRows ? handleRequestDelete : undefined}
        canInsertRows={canInsertRows}
        onAddRow={canInsertRows ? handleAddRow : undefined}
        databases={databaseSelectorEnabled ? databaseList : undefined}
        currentDatabase={activeDatabase}
        onSelectDatabase={databaseSelectorEnabled ? handleDatabaseSelection : undefined}
        databaseLoading={databaseLoading}
        databaseSwitching={isSwitchingDatabase}
      />
    )
  }, [rowCount, columnNames.length, executionTimeMs, executedAt, dirtyRowIds.length,
      canSave, saving, handleSave, handleExport, metadata,
      handleDiscardChanges, handleJumpToFirstError, canDeleteRows, handleRequestDelete,
      canInsertRows, handleAddRow, databaseSelectorEnabled, databaseList,
      activeDatabase, handleDatabaseSelection, databaseLoading, isSwitchingDatabase])

  const handleSelectAllPages = useCallback(() => {
    // When user clicks "Select all X rows", we need to mark that all pages are selected
    // For now, we'll show a toast. In a full implementation, this would:
    // 1. Make a backend call to fetch all row IDs (or keep a flag that all rows are selected)
    // 2. Update the selection state to include all row IDs
    // 3. When performing actions (delete, export), use the "selectAllPagesMode" flag

    toast({
      title: 'All rows selected',
      description: `All ${totalRows?.toLocaleString()} rows across all pages are now selected. Bulk actions will apply to all rows.`,
      variant: 'default',
    })
  }, [totalRows])

  const safeAffectedRows = Number.isFinite(affectedRows) ? affectedRows : 0
  const hasTabularResults = columnNames.length > 0 && rows.length > 0
  const isModificationStatement = columnNames.length === 0
  const affectedRowsMessage =
    safeAffectedRows === 1
      ? '1 row affected.'
      : `${safeAffectedRows.toLocaleString()} rows affected.`
  const pendingDeleteCount = pendingDeleteIds.length
  const effectiveTotalRows = totalRows !== undefined ? totalRows : rowCount
  const showPagination = onPageChange && effectiveTotalRows > 0 && hasTabularResults

  // Keyboard shortcut handler for Ctrl+S (Cmd+S on Mac) and pagination
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Ignore if user is typing in an input/textarea
      const target = event.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return
      }

      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
      const isSaveShortcut = isMac
        ? (event.metaKey && event.key === 's')
        : (event.ctrlKey && event.key === 's')

      if (isSaveShortcut) {
        event.preventDefault()

        // Only save if there are dirty rows and metadata is enabled
        if (metadata?.enabled && dirtyRowIds.length > 0) {
          handleSave()
        }
        return
      }

      // Pagination keyboard navigation (only when pagination is enabled)
      if (showPagination && !isLoadingPage && !saving) {
        const totalPages = Math.ceil(effectiveTotalRows / pageSize)

        // Alt+Left Arrow or Alt+PageUp - Previous page
        if (event.altKey && (event.key === 'ArrowLeft' || event.key === 'PageUp')) {
          event.preventDefault()
          if (currentPage > 1) {
            handlePageChange(currentPage - 1)
          }
          return
        }

        // Alt+Right Arrow or Alt+PageDown - Next page
        if (event.altKey && (event.key === 'ArrowRight' || event.key === 'PageDown')) {
          event.preventDefault()
          if (currentPage < totalPages) {
            handlePageChange(currentPage + 1)
          }
          return
        }

        // Alt+Home - First page
        if (event.altKey && event.key === 'Home') {
          event.preventDefault()
          if (currentPage > 1) {
            handlePageChange(1)
          }
          return
        }

        // Alt+End - Last page
        if (event.altKey && event.key === 'End') {
          event.preventDefault()
          if (currentPage < totalPages) {
            handlePageChange(totalPages)
          }
          return
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [metadata?.enabled, dirtyRowIds.length, handleSave, showPagination, isLoadingPage, saving, effectiveTotalRows, pageSize, currentPage, handlePageChange])

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      {!hasTabularResults ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            {isModificationStatement ? (
              <>
                <CheckCircle2 className="h-12 w-12 mx-auto mb-4 text-primary" />
                <p className="text-lg font-medium mb-1">Statement executed successfully</p>
                <p className="text-sm">{affectedRowsMessage}</p>
              </>
            ) : (
              <>
                <Inbox className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium mb-1">No results found</p>
                <p className="text-sm">Your query returned 0 rows</p>
              </>
            )}
          </div>
        </div>
      ) : (
        <>
          <EditableTable
            data={rows as TableRow[]}
            columns={tableColumns}
            onDirtyChange={handleDirtyChange}
            enableMultiSelect={canDeleteRows}
            enableGlobalFilter={false}
            enableExport={true}
            loading={saving || isLoadingPage}
            className="flex-1 min-h-0"
            height="100%"
            onExport={handleExport}
            onCellEdit={handleCellEdit}
            onRowInspect={handleRowClick}
            onSelectAllPages={effectiveTotalRows > rows.length ? handleSelectAllPages : undefined}
            toolbar={renderToolbar}
            footer={null}
            // Phase 2: Chunked data loading
            resultId={resultId}
            totalRows={effectiveTotalRows}
            isLargeResult={isLarge}
            chunkingEnabled={chunkingEnabled}
            displayMode={displayMode}
          />

          {/* Bottom pagination controls */}
          {showPagination && (
            <div className="border-t border-border bg-muted/20 px-4 py-2">
              <PaginationControls
                currentPage={currentPage}
                pageSize={pageSize}
                totalRows={effectiveTotalRows}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
                disabled={isLoadingPage || saving}
                compact
              />
            </div>
          )}
        </>
      )}

      <Dialog
        open={showDeleteDialog}
        onOpenChange={(open) => {
          if (isDeleting) return
          if (!open) {
            setShowDeleteDialog(false)
            setPendingDeleteIds([])
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete selected rows?</DialogTitle>
            <DialogDescription>
              {pendingDeleteCount === 1
                ? 'This will permanently delete the selected row.'
                : `This will permanently delete ${pendingDeleteCount} rows.`}
              {' '}This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowDeleteDialog(false)
                setPendingDeleteIds([])
              }}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={isDeleting}
              className="gap-2"
            >
              {isDeleting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Deleting…
                </>
              ) : (
                `Delete ${pendingDeleteCount} row${pendingDeleteCount === 1 ? '' : 's'}`
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <div className="flex-shrink-0 border-t border-border bg-muted/40 px-4 py-2 text-xs text-muted-foreground flex items-center justify-between">
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-1.5">
            <Database className="h-3.5 w-3.5" />
            {(totalRows !== undefined ? totalRows : rowCount).toLocaleString()} rows
            {totalRows !== undefined && rowCount < totalRows && (
              <span className="text-muted-foreground/60"> ({rowCount.toLocaleString()} shown)</span>
            )}
            {' • '}{columnNames.length} columns
          </span>
          {safeAffectedRows > 0 && (
            <span className="flex items-center gap-1.5">
              <CheckCircle2 className="h-3.5 w-3.5" />
              {safeAffectedRows.toLocaleString()} affected
            </span>
          )}
          <span className="flex items-center gap-1.5">
            <Clock className="h-3.5 w-3.5" />
            {executionTimeMs.toFixed(2)} ms
          </span>
        </div>
        <span>
          {dirtyRowIds.length > 0
            ? `${dirtyRowIds.length} pending change${dirtyRowIds.length === 1 ? '' : 's'}`
            : 'No pending changes'}
        </span>
      </div>

      {/* JSON Row Viewer Sidebar */}
      <JsonRowViewerSidebarV2
        open={jsonViewerOpen}
        onClose={handleCloseJsonViewer}
        rowData={selectedRowData}
        rowId={selectedRowId}
        rowIndex={selectedRowIndex}
        totalRows={rows.length}
        onNavigate={handleNavigateRow}
        columns={columnNames}
        metadata={metadata}
        connectionId={connectionId}
        onSave={handleJsonViewerSave}
      />
    </div>
  )
}
