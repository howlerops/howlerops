import { useEffect, useMemo, useRef, useState, useCallback } from 'react'
import { Database, Clock, Save, AlertCircle, Download, Search } from 'lucide-react'

import { EditableTable } from './EditableTable/EditableTable'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { TableColumn, ExportOptions } from '../types/table'
import { QueryEditableMetadata, QueryResultRow, useQueryStore } from '../store/query-store'
import { wailsEndpoints } from '../lib/wails-api'
import type { EditableTableContext } from '../types/table'

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
  onExport: (options: ExportOptions) => void
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

const ExportButton = ({ context, onExport }: { context: EditableTableContext; onExport: (options: unknown) => void }) => {
  const [showExportMenu, setShowExportMenu] = useState(false)
  const [exportOptions, setExportOptions] = useState({
    format: 'csv' as 'csv' | 'json',
    includeHeaders: true,
    selectedOnly: false,
  })

  const handleExport = (format: 'csv' | 'json') => {
    const options = { ...exportOptions, format }
    onExport(options)
    setShowExportMenu(false)
  }

  return (
    <div className="relative">
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowExportMenu(!showExportMenu)}
        className="gap-2"
      >
        <Download className="h-4 w-4" />
        Export
      </Button>

      {showExportMenu && (
        <div className="absolute right-0 top-full mt-1 w-64 bg-white border border-gray-300 rounded-md shadow-lg z-50">
          <div className="p-3">
            <h3 className="font-medium text-gray-900 mb-3">Export Options</h3>

            {/* Format selection */}
            <div className="mb-3">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Format
              </label>
              <select
                value={exportOptions.format}
                onChange={(e) => setExportOptions(prev => ({ ...prev, format: e.target.value as 'csv' | 'json' }))}
                className="w-full px-3 py-1 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
              </select>
            </div>

            {/* Options */}
            <div className="space-y-2 mb-3">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={exportOptions.includeHeaders}
                  onChange={(e) => setExportOptions(prev => ({ ...prev, includeHeaders: e.target.checked }))}
                  className="rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">Include headers</span>
              </label>
              {context.state.selectedRows.length > 0 && (
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={exportOptions.selectedOnly}
                    onChange={(e) => setExportOptions(prev => ({ ...prev, selectedOnly: e.target.checked }))}
                    className="rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Selected only ({context.state.selectedRows.length} rows)
                  </span>
                </label>
              )}
            </div>

            {/* Export buttons */}
            <div className="flex gap-2">
              <button
                onClick={() => handleExport(exportOptions.format)}
                className="flex-1 px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors text-sm"
              >
                Export
              </button>
              <button
                onClick={() => setShowExportMenu(false)}
                className="px-3 py-2 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-sm"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const QueryResultsToolbar = ({
  context,
  rowCount,
  columnCount,
  executionTimeMs,
  executedAt,
  dirtyCount,
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
    <div className="flex flex-col gap-3 border-b border-gray-200 bg-background px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
          <span className="flex items-center gap-2">
            <Database className="h-4 w-4" />
            {rowCount} rows
          </span>
          <span className="flex items-center gap-2">
            <Clock className="h-4 w-4" />
            {executionTimeMs.toFixed(2)} ms
          </span>
          {dirtyCount > 0 && (
            <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-800">
              {dirtyCount} pending change{dirtyCount > 1 ? 's' : ''}
            </span>
          )}
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

      {(saveError || saveSuccess || (!metadata?.enabled && metadata?.reason)) && (
        <div className="flex flex-col gap-2 text-xs">
          {saveError && (
            <div className="flex items-center gap-2 rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-destructive">
              <AlertCircle className="h-4 w-4" />
              <span>{saveError}</span>
            </div>
          )}
          {!saveError && saveSuccess && (
            <div className="rounded border border-emerald-300 bg-emerald-50 px-3 py-2 text-emerald-700">
              {saveSuccess}
            </div>
          )}
          {!metadata?.enabled && metadata?.reason && (
            <div className="flex items-center gap-2 rounded border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800">
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

        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span>
            {rowCount} rows • {columnCount} columns
          </span>
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

  const handleExport = useCallback((options: ExportOptions) => {
    const currentRows = resolveCurrentRows()
    let dataToExport = currentRows

    // Filter to selected rows only if requested
    if (options.selectedOnly && tableContextRef.current?.state.selectedRows.length && tableContextRef.current.state.selectedRows.length > 0) {
      const selectedIds = tableContextRef.current.state.selectedRows
      dataToExport = currentRows.filter(row => selectedIds.includes(row.__rowId!))
    }

    if (options.format === 'csv') {
      const header = options.includeHeaders ? columns.join(',') : ''
      const records = dataToExport.map((row) =>
        columns.map((column) => serialiseCsvValue(row[column])).join(',')
      )

      const csv = options.includeHeaders ? [header, ...records].join('\n') : records.join('\n')
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', `query-results-${Date.now()}.csv`)
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } else if (options.format === 'json') {
      const json = JSON.stringify(dataToExport, null, 2)
      const blob = new Blob([json], { type: 'application/json;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', `query-results-${Date.now()}.json`)
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

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <EditableTable
        data={rows}
        columns={tableColumns}
        onDirtyChange={(ids) => {
          setDirtyRowIds(ids)
          if (ids.length > 0) {
            setSaveSuccess(null)
          }
        }}
        enableMultiSelect={false}
        enableGlobalFilter={false}
        enableExport={true}
        loading={saving}
        className="flex-1 min-h-0"
        height="100%"
        onExport={handleExport}
        onCellEdit={handleCellEdit}
        toolbar={(context) => {
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
        }}
        footer={null}
      />

      <div className="flex-shrink-0 border-t border-border bg-muted/40 px-4 py-2 text-xs text-muted-foreground flex items-center justify-between">
        <span>
          {rowCount.toLocaleString()} rows • {columns.length} columns
        </span>
        <span>
          {dirtyRowIds.length > 0
            ? `${dirtyRowIds.length} pending change${dirtyRowIds.length === 1 ? '' : 's'}`
            : 'No pending changes'}
        </span>
      </div>
    </div>
  )
}
