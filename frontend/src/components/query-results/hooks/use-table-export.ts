import { useCallback } from 'react'

import { toast } from '../../../hooks/use-toast'
import type { QueryResultRow } from '../../../store/query-store'
import type { EditableTableContext, ExportOptions } from '../../../types/table'
import { serialiseCsvValue } from '../utils'

interface UseTableExportOptions {
  connectionId?: string
  query: string
  columnNames: string[]
  tableContextRef: React.MutableRefObject<EditableTableContext | null>
  resolveCurrentRows: () => QueryResultRow[]
}

interface UseTableExportReturn {
  handleExport: (options: ExportOptions) => Promise<void>
}

export function useTableExport({
  connectionId,
  query,
  columnNames,
  tableContextRef,
  resolveCurrentRows,
}: UseTableExportOptions): UseTableExportReturn {
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

        const { SaveToDownloads } = await import('../../../../bindings/github.com/jbeck018/howlerops/app')
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

      const { wailsApiClient } = await import('../../../lib/wails-api')
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

      const { SaveToDownloads } = await import('../../../../bindings/github.com/jbeck018/howlerops/app')
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
  }, [connectionId, query, columnNames, resolveCurrentRows, tableContextRef])

  return {
    handleExport,
  }
}
