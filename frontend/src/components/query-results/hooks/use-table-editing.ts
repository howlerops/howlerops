import { useCallback, useMemo, useState } from 'react'

import { toast } from '../../../hooks/use-toast'
import { api } from '../../../lib/api-client'
import type { QueryEditableColumn, QueryEditableMetadata, QueryResultRow } from '../../../store/query-store'
import type { CellValue, EditableTableContext, TableColumn } from '../../../types/table'
import { buildColumnsLookup, buildMetadataLookup, buildPrimaryKeyMap, createRowId, deriveColumnDisplayTraits, inferColumnType } from '../utils'

interface UseTableEditingOptions {
  resultId: string
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
  columnNames: string[]
  metadata?: QueryEditableMetadata | null
  connectionId?: string
  query: string
  tableContextRef: React.MutableRefObject<EditableTableContext | null>
  updateResultRows: (resultId: string, rows: QueryResultRow[], originalRows: Record<string, QueryResultRow>) => void
}

interface UseTableEditingReturn {
  dirtyRowIds: string[]
  saving: boolean
  canSave: boolean
  hasEditableColumns: boolean
  canInsertRows: boolean
  tableColumns: TableColumn[]
  columnsLookup: Record<string, string>
  metadataLookup: Map<string, QueryEditableColumn>
  firstEditableColumnId: string
  setDirtyRowIds: React.Dispatch<React.SetStateAction<string[]>>
  handleSave: () => Promise<void>
  handleAddRow: () => void
  handleDiscardChanges: () => void
  handleJumpToFirstError: () => void
  handleCellEdit: (rowId: string, columnId: string, value: unknown) => Promise<boolean>
  handleDirtyChange: (ids: string[]) => void
  resolveCurrentRows: () => QueryResultRow[]
  handleJsonViewerSave: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>
}

export function useTableEditing({
  resultId,
  rows,
  originalRows,
  columnNames,
  metadata,
  connectionId,
  query,
  tableContextRef,
  updateResultRows,
}: UseTableEditingOptions): UseTableEditingReturn {
  const [dirtyRowIds, setDirtyRowIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

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

  const hasEditableColumns = useMemo(() => {
    return columnNames.some(columnName => {
      const metaColumn = metadataLookup.get(columnName.toLowerCase())
      return Boolean(metaColumn?.editable)
    })
  }, [columnNames, metadataLookup])

  const canInsertRows = useMemo(() => {
    if (!metadata || !metadata.table) {
      return false
    }
    return Boolean(metadata.capabilities?.canInsert && hasEditableColumns)
  }, [metadata, hasEditableColumns])

  const canSave = Boolean(metadata?.enabled && dirtyRowIds.length > 0 && !saving)

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
        editable: Boolean(metaColumn?.editable),
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
  }, [columnNames, metadataLookup])

  const resolveCurrentRows = useCallback((): QueryResultRow[] => {
    const contextRows = tableContextRef.current?.data as QueryResultRow[] | undefined
    const source = contextRows ?? rows
    return source.map((row) => ({ ...row }))
  }, [rows, tableContextRef])

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
    tableContextRef,
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

          const response = await api.queries.insertRow({
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

          const returnedValues = (response.row || {}) as Record<string, unknown>
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

        const response = await api.queries.updateRow({
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
    tableContextRef,
    updateResultRows,
  ])

  const handleDiscardChanges = useCallback(() => {
    if (tableContextRef.current) {
      tableContextRef.current.actions.resetTable()
      setDirtyRowIds([])
    }
  }, [tableContextRef])

  const handleJumpToFirstError = useCallback(() => {
    if (!tableContextRef.current) return

    const invalidCells = tableContextRef.current.actions.getInvalidCells()
    if (invalidCells.length === 0) return

    const firstError = invalidCells[0]
    const cellElement = document.querySelector(`[data-row-id="${firstError.rowId}"][data-column-id="${firstError.columnId}"]`)
    if (cellElement) {
      cellElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      ;(cellElement as HTMLElement).focus()
    }
  }, [tableContextRef])

  const handleCellEdit = useCallback(async (
    _rowId: string,
    _columnId: string,
    _value: unknown
  ): Promise<boolean> => {
    // NOTE: This is now a no-op since we don't auto-save on cell edit
    // All saving happens via the "Save Changes" button (handleSave)
    return true
  }, [])

  const handleDirtyChange = useCallback((ids: string[]) => {
    setDirtyRowIds(ids)
  }, [])

  const handleJsonViewerSave = useCallback(async (rowId: string, data: Record<string, CellValue>): Promise<boolean> => {
    if (!connectionId || !metadata?.enabled) return false

    try {
      const originalRow = originalRows[rowId]
      if (!originalRow) return false

      const primaryKey = buildPrimaryKeyMap(originalRow, metadata, columnsLookup)
      if (!primaryKey) return false

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

      const response = await api.queries.updateRow({
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

  return {
    dirtyRowIds,
    saving,
    canSave,
    hasEditableColumns,
    canInsertRows,
    tableColumns,
    columnsLookup,
    metadataLookup,
    firstEditableColumnId,
    setDirtyRowIds,
    handleSave,
    handleAddRow,
    handleDiscardChanges,
    handleJumpToFirstError,
    handleCellEdit,
    handleDirtyChange,
    resolveCurrentRows,
    handleJsonViewerSave,
  }
}
