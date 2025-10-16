import { useEffect, useMemo, useRef, useState, useCallback } from 'react'
import { Database, Clock, Save, AlertCircle, Download, Search, Inbox, Loader2 } from 'lucide-react'

import { EditableTable } from './editable-table/editable-table'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog'
import { TableColumn, ExportOptions, TableRow } from '../types/table'
import { QueryEditableMetadata, QueryResultRow, useQueryStore } from '../store/query-store'
import { wailsEndpoints } from '../lib/wails-api'
import type { EditableTableContext } from '../types/table'
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
  saveError?: string | null
  saveSuccess?: string | null
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
  saveError,
  saveSuccess,
}: ToolbarProps) => {
  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    context.actions.updateGlobalFilter(event.target.value)
  }

  return (
    <div className="flex flex-col gap-3 border-b border-gray-200 bg-background px-1 py-1">
      {(saveError || saveSuccess || (!metadata?.enabled && metadata?.reason)) && (
        <div className="flex flex-col gap-2 text-xs">
          {saveError && (
            <div className="flex items-center gap-2 rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-destructive">
              <AlertCircle className="h-4 w-4" />
              <span>{saveError}</span>
            </div>
          )}
          {!saveError && saveSuccess && (
            <div className="rounded border border-primary bg-primary/10 px-3 py-2 text-primary">
              {saveSuccess}
            </div>
          )}
          {!metadata?.enabled && metadata?.reason && (
            <div className="flex items-center gap-2 rounded border border-accent bg-accent/10 px-3 py-2 text-accent-foreground">
              <AlertCircle className="h-4 w-4" />
              <span>{metadata.reason}</span>
            </div>
          )}
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
          
          {/* Export Button */}
          <ExportButton context={context} onExport={onExport} />
          
          {metadata?.enabled && (
            <Button
              size="sm"
              onClick={onSave}
              disabled={!canSave}
              className="gap-2"
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
  columns,
  rows,
  originalRows,
  metadata,
  query,
  connectionId,
  executionTimeMs,
  rowCount,
  executedAt,
}: QueryResultsTableProps) => {
  const [dirtyRowIds, setDirtyRowIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null)

  const updateResultRows = useQueryStore((state) => state.updateResultRows)
  const columnsLookup = useMemo(() => buildColumnsLookup(metadata), [metadata])
  const tableContextRef = useRef<EditableTableContext | null>(null)

  useEffect(() => {
    setDirtyRowIds([])
    setSaveError(null)
    tableContextRef.current?.actions.clearDirtyRows?.()
    tableContextRef.current?.actions.resetTable?.()
  }, [rows, resultId])

  useEffect(() => {
    if (!saveSuccess) return
    const timeout = window.setTimeout(() => {
      setSaveSuccess(null)
    }, 4000)
    return () => window.clearTimeout(timeout)
  }, [saveSuccess])

  const tableColumns: TableColumn[] = useMemo(() => {
    return columns.map<TableColumn>((columnName) => {
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
  }, [columns, metadata])

  const resolveCurrentRows = useCallback((): QueryResultRow[] => {
    const contextRows = tableContextRef.current?.data as QueryResultRow[] | undefined
    const source = contextRows ?? rows
    return source.map((row) => ({ ...row }))
  }, [rows])

  // const handleExportCsv = useCallback(() => {
  //   const currentRows = resolveCurrentRows()
  //   const header = columns.join(',')
  //   const records = currentRows.map((row) =>
  //     columns.map((column) => serialiseCsvValue(row[column])).join(',')
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
  // }, [columns, resolveCurrentRows])

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
      const header = options.includeHeaders ? columns.join(',') : ''
      const records = dataToExport.map((row) =>
        columns.map((column) => serialiseCsvValue(row[column])).join(',')
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
  }, [columns, resolveCurrentRows])

  const handleCellEdit = useCallback(async (
    rowId: string,
    columnId: string,
    value: unknown
  ): Promise<boolean> => {
    if (!metadata?.enabled) {
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
      const metaColumn = metadata.columns?.find((col) => {
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
        columns,
        schema: metadata.schema,
        table: metadata.table,
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
    resolveCurrentRows,
    originalRows,
    columnsLookup,
    connectionId,
    query,
    columns,
    resultId,
    updateResultRows,
  ])

  const handleSave = useCallback(async () => {
    const currentRows = resolveCurrentRows()

    if (!metadata?.enabled) {
      return
    }
    if (!connectionId) {
      setSaveError('No active connection. Please select a connection and try again.')
      return
    }
    if (dirtyRowIds.length === 0) {
      return
    }

    setSaving(true)
    setSaveError(null)
    setSaveSuccess(null)

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
        columns.forEach((columnName) => {
          const currentValue = currentRow[columnName]
          const originalValue = originalRow[columnName]

          const valuesAreEqual =
            currentValue === originalValue ||
            (currentValue == null && originalValue == null)

          const metaColumn = metadata.columns?.find((col) => {
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
          columns,
          schema: metadata.schema,
          table: metadata.table,
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
      setSaveSuccess('Changes saved successfully.')
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : 'Failed to save changes')
    } finally {
      setSaving(false)
    }
  }, [
    connectionId,
    columns,
    columnsLookup,
    dirtyRowIds,
    metadata,
    originalRows,
    query,
    resultId,
    resolveCurrentRows,
    updateResultRows,
  ])

  const canSave = Boolean(metadata?.enabled && dirtyRowIds.length > 0 && !saving)

  // Memoize onDirtyChange to prevent infinite re-renders
  const handleDirtyChange = useCallback((ids: string[]) => {
    setDirtyRowIds(ids)
    if (ids.length > 0) {
      setSaveSuccess(null)
    }
  }, [])

  // Memoize toolbar function to prevent infinite re-renders
  const renderToolbar = useCallback((context: EditableTableContext) => {
    // Capture context in ref outside of render
    tableContextRef.current = context
    
    return (
      <QueryResultsToolbar
        context={context}
        rowCount={rowCount}
        columnCount={columns.length}
        executionTimeMs={executionTimeMs}
        executedAt={executedAt}
        dirtyCount={dirtyRowIds.length}
        canSave={canSave}
        saving={saving}
        onSave={handleSave}
        onExport={handleExport}
        metadata={metadata}
        saveError={saveError}
        saveSuccess={saveSuccess}
      />
    )
  }, [rowCount, columns.length, executionTimeMs, executedAt, dirtyRowIds.length, 
      canSave, saving, handleSave, handleExport, metadata, saveError, saveSuccess])

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      {rows.length === 0 ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            <Inbox className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p className="text-lg font-medium mb-1">No results found</p>
            <p className="text-sm">Your query returned 0 rows</p>
          </div>
        </div>
      ) : (
        <EditableTable
          data={rows as TableRow[]}
          columns={tableColumns}
          onDirtyChange={handleDirtyChange}
          enableMultiSelect={false}
          enableGlobalFilter={false}
          enableExport={true}
          loading={saving}
          className="flex-1 min-h-0"
          height="100%"
          onExport={handleExport}
          onCellEdit={handleCellEdit}
          toolbar={renderToolbar}
          footer={null}
        />
      )}

      <div className="flex-shrink-0 border-t border-border bg-muted/40 px-4 py-2 text-xs text-muted-foreground flex items-center justify-between">
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-1.5">
            <Database className="h-3.5 w-3.5" />
            {rowCount.toLocaleString()} rows • {columns.length} columns
          </span>
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
    </div>
  )
}
