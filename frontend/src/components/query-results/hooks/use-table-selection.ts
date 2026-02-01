import { useCallback, useState } from 'react'

import { toast } from '../../../hooks/use-toast'
import { api } from '../../../lib/api-client'
import type { QueryEditableMetadata, QueryResultRow } from '../../../store/query-store'
import type { EditableTableContext } from '../../../types/table'
import { buildPrimaryKeyMap } from '../utils'

interface UseTableSelectionOptions {
  connectionId?: string
  query: string
  columnNames: string[]
  columnsLookup: Record<string, string>
  metadata?: QueryEditableMetadata | null
  originalRows: Record<string, QueryResultRow>
  resultId: string
  tableContextRef: React.MutableRefObject<EditableTableContext | null>
  resolveCurrentRows: () => QueryResultRow[]
  updateResultRows: (resultId: string, rows: QueryResultRow[], originalRows: Record<string, QueryResultRow>) => void
  setDirtyRowIds: React.Dispatch<React.SetStateAction<string[]>>
  onRowDeleted?: (deletedRowId: string) => void
}

interface UseTableSelectionReturn {
  pendingDeleteIds: string[]
  showDeleteDialog: boolean
  isDeleting: boolean
  canDeleteRows: boolean
  handleRequestDelete: () => void
  handleConfirmDelete: () => Promise<void>
  setShowDeleteDialog: (open: boolean) => void
  setPendingDeleteIds: React.Dispatch<React.SetStateAction<string[]>>
}

export function useTableSelection({
  connectionId,
  query,
  columnNames,
  columnsLookup,
  metadata,
  originalRows,
  resultId,
  tableContextRef,
  resolveCurrentRows,
  updateResultRows,
  setDirtyRowIds,
  onRowDeleted,
}: UseTableSelectionOptions): UseTableSelectionReturn {
  const [pendingDeleteIds, setPendingDeleteIds] = useState<string[]>([])
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const canDeleteRows = Boolean(
    metadata?.table &&
    metadata?.primaryKeys?.length &&
    metadata?.capabilities?.canDelete
  )

  const handleRequestDelete = useCallback(() => {
    const selected = tableContextRef.current?.state.selectedRows ?? []
    if (!selected.length) {
      return
    }
    setPendingDeleteIds(selected)
    setShowDeleteDialog(true)
  }, [tableContextRef])

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
        const response = await api.queries.deleteRows({
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

      // Notify about deleted rows for any cleanup needed
      pendingDeleteIds.forEach(id => {
        onRowDeleted?.(id)
      })

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
    setDirtyRowIds,
    tableContextRef,
    updateResultRows,
    onRowDeleted,
  ])

  return {
    pendingDeleteIds,
    showDeleteDialog,
    isDeleting,
    canDeleteRows,
    handleRequestDelete,
    handleConfirmDelete,
    setShowDeleteDialog,
    setPendingDeleteIds,
  }
}
