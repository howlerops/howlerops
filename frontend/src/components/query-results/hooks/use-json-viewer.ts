import { useCallback, useState } from 'react'

import type { QueryResultRow } from '../../../store/query-store'
import type { TableRow } from '../../../types/table'

interface UseJsonViewerOptions {
  rows: QueryResultRow[]
}

interface UseJsonViewerReturn {
  jsonViewerOpen: boolean
  selectedRowId: string | null
  selectedRowData: TableRow | null
  selectedRowIndex: number
  handleRowClick: (rowId: string, rowData: TableRow) => void
  handleCloseJsonViewer: () => void
  handleNavigateRow: (direction: 'prev' | 'next') => void
  clearSelection: () => void
}

export function useJsonViewer({ rows }: UseJsonViewerOptions): UseJsonViewerReturn {
  const [jsonViewerOpen, setJsonViewerOpen] = useState(false)
  const [selectedRowId, setSelectedRowId] = useState<string | null>(null)
  const [selectedRowData, setSelectedRowData] = useState<TableRow | null>(null)
  const [selectedRowIndex, setSelectedRowIndex] = useState<number>(0)

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

  const clearSelection = useCallback(() => {
    setJsonViewerOpen(false)
    setSelectedRowId(null)
    setSelectedRowData(null)
  }, [])

  return {
    jsonViewerOpen,
    selectedRowId,
    selectedRowData,
    selectedRowIndex,
    handleRowClick,
    handleCloseJsonViewer,
    handleNavigateRow,
    clearSelection,
  }
}
