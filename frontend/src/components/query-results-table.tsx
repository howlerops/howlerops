import { useEffect, useMemo, useRef, useState, useCallback } from 'react'
import { Database, Clock, Save, AlertCircle, Download, Search, Inbox, Loader2, CheckCircle2, Trash2 } from 'lucide-react'

import { EditableTable } from './editable-table/editable-table'
import { JsonRowViewerSidebar } from './json-row-viewer-sidebar'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog'
import { TableColumn, ExportOptions, TableRow } from '../types/table'
import { QueryEditableMetadata, QueryResultRow, useQueryStore } from '../store/query-store'
import { wailsEndpoints } from '../lib/wails-api'
import type { CellValue, EditableTableContext } from '../types/table'
import { toast } from '../hooks/use-toast'

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
  if (normalized.includes('date') || normalized.includes('time')) {
    return 'date'
  }

  return 'text'
}

const formatTimestamp = (value: Date) => {
  return value.toLocaleTimeString()
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

const quoteIdentifier = (identifier: string): string => {
  return `"${String(identifier).replace(/"/g, '""')}"`
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
      setShowExportDialog(false)
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
              Configure export options and download to your Downloads folder.
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
  executedAt,
  canSave,
  saving,
  onSave,
  onExport,
  metadata,
  onDiscardChanges,
  onJumpToFirstError,
  canDeleteRows,
  onDeleteSelected,
}: ToolbarProps) => {
  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    context.actions.updateGlobalFilter(event.target.value)
  }

  const invalidCellsCount = context.state.invalidCells.size
  const dirtyCount = context.state.dirtyRows.size
  const hasValidationErrors = invalidCellsCount > 0
  const canSaveWithValidation = canSave && !hasValidationErrors
  const selectedCount = context.state.selectedRows.length

  return (
    <div className="flex flex-col gap-2 border-b border-gray-200 bg-background px-1 py-1">
      {/* Show non-editable reason if applicable */}
      {!metadata?.enabled && metadata?.reason && (
        <div className="flex items-center gap-2 rounded border border-accent bg-accent/10 px-3 py-2 text-accent-foreground text-xs">
          <AlertCircle className="h-4 w-4" />
          <span>{metadata.reason}</span>
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="relative w-full max-w-xs">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={context.state.globalFilter}
            onChange={handleSearchChange}
            placeholder="Search…"
            className="h-9 w-full pl-9"
          />
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground">{formatTimestamp(executedAt)}</span>

          {/* Unsaved changes indicator */}
          {dirtyCount > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 bg-accent/10 border border-accent rounded text-xs">
              <span className="text-accent-foreground">
                {dirtyCount} unsaved{dirtyCount === 1 ? '' : ''}
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
              {dirtyCount > 0 && onDiscardChanges && (
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

const createOriginalMap = (rows: QueryResultRow[]): Record<string, QueryResultRow> => {
  return rows.reduce<Record<string, QueryResultRow>>((acc, row) => {
    acc[row.__rowId!] = { ...row }
    return acc
  }, {})
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
}: QueryResultsTableProps) => {
  const columnNames = Array.isArray(columns) ? columns : []
  const [dirtyRowIds, setDirtyRowIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  
  // JSON viewer state
  const [jsonViewerOpen, setJsonViewerOpen] = useState(false)
  const [selectedRowId, setSelectedRowId] = useState<string | null>(null)
  const [selectedRowData, setSelectedRowData] = useState<TableRow | null>(null)
  const [pendingDeleteIds, setPendingDeleteIds] = useState<string[]>([])
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const updateResultRows = useQueryStore((state) => state.updateResultRows)
  const columnsLookup = useMemo(() => buildColumnsLookup(metadata), [metadata])
  const tableContextRef = useRef<EditableTableContext | null>(null)
  const canDeleteRows = useMemo(() => {
    return Boolean(metadata?.enabled && metadata?.table && metadata?.primaryKeys?.length)
  }, [metadata])

  useEffect(() => {
    setDirtyRowIds([])
    tableContextRef.current?.actions.clearDirtyRows?.()
    tableContextRef.current?.actions.resetTable?.()
  }, [rows, resultId])

  const resolveCurrentRows = useCallback((): QueryResultRow[] => {
    const contextRows = tableContextRef.current?.data as QueryResultRow[] | undefined
    const source = contextRows ?? rows
    return source.map((row) => ({ ...row }))
  }, [rows])

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

    // Validate all cells before saving
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
      for (const rowId of dirtyRowIds) {
        const currentRow = currentRows.find((row) => row.__rowId === rowId)
        const originalRow = originalRows[rowId]

        if (!currentRow || !originalRow) {
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
      }

      const newOriginalRows = createOriginalMap(currentRows)
      updateResultRows(resultId, currentRows, newOriginalRows)
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

  const getMetaForColumn = useCallback((columnName: string) => {
    const lower = columnName.toLowerCase()
    return metadata?.columns?.find((col) => {
      const candidates = [col.name, col.resultName].filter(Boolean) as string[]
      return candidates.some((candidate) => candidate.toLowerCase() === lower)
    })
  }, [metadata])

  const formatValueForColumn = useCallback((columnName: string, value: unknown): string => {
    const meta = getMetaForColumn(columnName)
    const dataType = meta?.dataType?.toLowerCase() ?? ''

    if (value === null || value === undefined) {
      return 'NULL'
    }

    if (typeof value === 'number') {
      return Number.isFinite(value) ? String(value) : `'${String(value).replace(/'/g, "''")}'`
    }

    if (typeof value === 'boolean') {
      return value ? 'TRUE' : 'FALSE'
    }

    if (dataType.includes('bool')) {
      const normalized = String(value).toLowerCase()
      if (['true', 't', '1', 'yes'].includes(normalized)) return 'TRUE'
      if (['false', 'f', '0', 'no'].includes(normalized)) return 'FALSE'
    }

    if (dataType.match(/int|numeric|decimal|float|double|real|serial|money/)) {
      const numeric = Number(value)
      if (!Number.isNaN(numeric)) {
        return String(numeric)
      }
    }

    const stringValue = value instanceof Date ? value.toISOString() : String(value)
    return `'${stringValue.replace(/'/g, "''")}'`
  }, [getMetaForColumn])

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

      const whereClauses: string[] = []

      rowsToDelete.forEach((row) => {
        const primaryKey = buildPrimaryKeyMap(row, metadata, columnsLookup)
        if (!primaryKey) {
          throw new Error('Unable to determine primary key for one of the selected rows.')
        }

        const clauseParts = Object.entries(primaryKey).map(([pkColumn, pkValue]) => {
          const columnIdentifier = quoteIdentifier(pkColumn)
          if (pkValue === null || pkValue === undefined) {
            return `${columnIdentifier} IS NULL`
          }
          const valueLiteral = formatValueForColumn(pkColumn, pkValue)
          return `${columnIdentifier} = ${valueLiteral}`
        })

        if (clauseParts.length > 0) {
          whereClauses.push(`(${clauseParts.join(' AND ')})`)
        }
      })

      if (!whereClauses.length) {
        throw new Error('Unable to build a deletion condition for the selected rows.')
      }

      const tableIdentifier = metadata.schema
        ? `${quoteIdentifier(metadata.schema)}.${quoteIdentifier(metadata.table)}`
        : quoteIdentifier(metadata.table)

      const deleteStatement = `DELETE FROM ${tableIdentifier} WHERE ${whereClauses.join(' OR ')};`

      const response = await wailsEndpoints.queries.execute(connectionId, deleteStatement)
      if (!response.success) {
        throw new Error(response.message || 'Failed to delete selected rows.')
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
    connectionId,
    formatValueForColumn,
    metadata,
    originalRows,
    pendingDeleteIds,
    resolveCurrentRows,
    resultId,
    selectedRowId,
    updateResultRows,
  ])

  // Keyboard shortcut handler for Ctrl+S (Cmd+S on Mac)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
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
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [metadata?.enabled, dirtyRowIds.length, handleSave])

  const tableColumns: TableColumn[] = useMemo(() => {
    return columnNames.map<TableColumn>((columnName) => {
      const metaColumn = metadata?.columns?.find((col) => {
        const candidate = (col.resultName ?? col.name)?.toLowerCase()
        return candidate ? candidate === columnName.toLowerCase() : false
      })

      return {
        id: columnName,
        accessorKey: columnName,
        header: columnName,
        type: inferColumnType(metaColumn?.dataType),
        editable: Boolean(metadata?.enabled && metaColumn?.editable),
        sortable: true,
        filterable: true,
        minWidth: 120,
      }
    })
  }, [columnNames, metadata])

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
    const currentRows = resolveCurrentRows()
    let dataToExport = currentRows

    // Filter to selected rows only if requested
    if (options.selectedOnly && tableContextRef.current?.state.selectedRows.length && tableContextRef.current.state.selectedRows.length > 0) {
      const selectedIds = tableContextRef.current.state.selectedRows
      dataToExport = currentRows.filter(row => selectedIds.includes(row.__rowId!))
    }

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

    try {
      // Import the Wails function
      const { SaveToDownloads } = await import('../../wailsjs/go/main/App')
      const filePath = await SaveToDownloads(filename, content)
      
      // Show success notification with file path
      toast({
        title: 'Export successful',
        description: `File saved to: ${filePath}`,
        variant: 'default',
      })
    } catch (error) {
      console.error('Failed to save file to Downloads:', error)
      
      // Show error and fallback to browser download
      toast({
        title: 'Export failed',
        description: 'Falling back to browser download',
        variant: 'destructive',
      })
      
      // Fallback to browser download if Wails method fails
      const blob = new Blob([content], { 
        type: options.format === 'csv' ? 'text/csv;charset=utf-8;' : 'application/json;charset=utf-8;'
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', filename)
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    }
  }, [columnNames, resolveCurrentRows])

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
    setSelectedRowId(rowId)
    setSelectedRowData(rowData)
    setJsonViewerOpen(true)
  }, [])

  const handleCloseJsonViewer = useCallback(() => {
    setJsonViewerOpen(false)
    setSelectedRowId(null)
    setSelectedRowData(null)
  }, [])

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
      />
    )
  }, [rowCount, columnNames.length, executionTimeMs, executedAt, dirtyRowIds.length,
      canSave, saving, handleSave, handleExport, metadata,
      handleDiscardChanges, handleJumpToFirstError, canDeleteRows, handleRequestDelete])

  const safeAffectedRows = Number.isFinite(affectedRows) ? affectedRows : 0
  const hasTabularResults = columnNames.length > 0 && rows.length > 0
  const isModificationStatement = columnNames.length === 0
  const affectedRowsMessage =
    safeAffectedRows === 1
      ? '1 row affected.'
      : `${safeAffectedRows.toLocaleString()} rows affected.`
  const pendingDeleteCount = pendingDeleteIds.length

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
        <EditableTable
          data={rows as TableRow[]}
          columns={tableColumns}
          onDirtyChange={handleDirtyChange}
          enableMultiSelect={canDeleteRows}
          enableGlobalFilter={false}
          enableExport={true}
          loading={saving}
          className="flex-1 min-h-0"
          height="100%"
          onExport={handleExport}
          onCellEdit={handleCellEdit}
          onRowInspect={handleRowClick}
          toolbar={renderToolbar}
          footer={null}
        />
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
            {rowCount.toLocaleString()} rows • {columnNames.length} columns
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
      <JsonRowViewerSidebar
        open={jsonViewerOpen}
        onClose={handleCloseJsonViewer}
        rowData={selectedRowData}
        rowId={selectedRowId}
        columns={columnNames}
        metadata={metadata}
        connectionId={connectionId}
        onSave={handleJsonViewerSave}
      />
    </div>
  )
}
