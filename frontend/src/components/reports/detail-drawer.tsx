import { Download, FileQuestion, Loader2, X } from 'lucide-react'
import { useMemo, useState } from 'react'

import { PaginatedTable } from '@/components/reports/paginated-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import type { ReportRunComponentResult } from '@/types/reports'

interface DetailDrawerProps {
  open: boolean
  onClose: () => void
  title: string
  filters?: Record<string, unknown>
  loading: boolean
  data: ReportRunComponentResult | null
  error: string | null
}

/**
 * Slide-out panel showing drill-down details
 *
 * Features:
 * - Displays active filters
 * - Shows detailed data in paginated table
 * - Export functionality
 * - Loading and error states
 * - Keyboard shortcuts (Esc to close)
 */
export function DetailDrawer({
  open,
  onClose,
  title,
  filters,
  loading,
  data,
  error,
}: DetailDrawerProps) {
  const [exportLoading, setExportLoading] = useState(false)

  // Transform data for table display
  const tableData = useMemo(() => {
    if (!data || !data.columns || !data.rows) return null

    return {
      columns: data.columns,
      rows: data.rows,
    }
  }, [data])

  // Export detail data as CSV
  const handleExport = async () => {
    if (!tableData) return

    setExportLoading(true)
    try {
      const csv = generateCsv(tableData.columns, tableData.rows)
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)

      const link = document.createElement('a')
      link.href = url
      link.download = `drill-down-${Date.now()}.csv`
      link.click()

      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Export failed:', err)
    } finally {
      setExportLoading(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onClose}>
      <SheetContent side="right" className="w-[600px] sm:w-[800px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8">
                <X className="h-4 w-4" />
              </Button>
              <span>{title}</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleExport}
              disabled={!tableData || loading || exportLoading}
            >
              {exportLoading ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Download className="h-4 w-4 mr-2" />
              )}
              Export
            </Button>
          </SheetTitle>
          <SheetDescription>
            Showing detailed breakdown for selected segment
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-4">
          {/* Active filters */}
          {filters && Object.keys(filters).length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Active Filters</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex gap-2 flex-wrap">
                  {Object.entries(filters).map(([key, value]) => (
                    <Badge key={key} variant="outline">
                      {key} = {String(value)}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Error state */}
          {error && (
            <Card className="border-destructive">
              <CardHeader>
                <CardTitle className="text-sm text-destructive">Error Loading Details</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{error}</p>
              </CardContent>
            </Card>
          )}

          {/* Loading state */}
          {loading && (
            <div className="flex items-center justify-center h-64">
              <div className="text-center space-y-4">
                <Loader2 className="h-8 w-8 animate-spin mx-auto text-primary" />
                <p className="text-sm text-muted-foreground">Loading detail data...</p>
              </div>
            </div>
          )}

          {/* Detail data */}
          {!loading && !error && tableData && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">
                  {tableData.rows.length.toLocaleString()} Record{tableData.rows.length !== 1 ? 's' : ''}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <PaginatedTable
                  columns={tableData.columns}
                  rows={tableData.rows}
                  pageSize={50}
                />
              </CardContent>
            </Card>
          )}

          {/* Empty state */}
          {!loading && !error && !tableData && (
            <EmptyState
              icon={FileQuestion}
              title="No detail data"
              description="No records found matching the selected criteria"
            />
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

/**
 * Generate CSV from table data
 */
function generateCsv(columns: string[], rows: unknown[][]): string {
  const lines: string[] = []

  // Header row
  lines.push(columns.map((col) => escapeCsvValue(col)).join(','))

  // Data rows
  rows.forEach((row) => {
    const values = row.map((cell) => escapeCsvValue(cell))
    lines.push(values.join(','))
  })

  return lines.join('\n')
}

/**
 * Escape CSV value (handle quotes, commas, newlines)
 */
function escapeCsvValue(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }

  const str = String(value)

  // If value contains comma, quote, or newline, wrap in quotes and escape quotes
  if (str.includes(',') || str.includes('"') || str.includes('\n')) {
    return `"${str.replaceAll('"', '""')}"`
  }

  return str
}
