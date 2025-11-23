import { useVirtualizer } from '@tanstack/react-virtual'
import { useMemo, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationFirst,
  PaginationItem,
  PaginationLast,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface PaginatedTableProps {
  columns: string[]
  rows: unknown[][]
  pageSize?: number
  useVirtualScrolling?: boolean
  className?: string
}

function formatCell(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

/**
 * High-performance table component with pagination and optional virtual scrolling.
 *
 * Performance characteristics:
 * - Pagination mode: Renders only visible page (default 100 rows)
 * - Virtual scrolling mode: Renders only visible rows + overscan (10-20 rows)
 * - Memory usage: O(visible rows) instead of O(total rows)
 * - Re-render cost: Minimal - only page/scroll state changes trigger updates
 */
export function PaginatedTable({
  columns,
  rows,
  pageSize = 100,
  useVirtualScrolling = false,
  className = '',
}: PaginatedTableProps) {
  const [currentPage, setCurrentPage] = useState(1)
  const [itemsPerPage, setItemsPerPage] = useState(pageSize)

  // Memoize pagination calculations to avoid recalculating on every render
  const { visibleRows, totalPages, startRow, endRow } = useMemo(() => {
    const start = (currentPage - 1) * itemsPerPage
    const end = start + itemsPerPage
    return {
      visibleRows: rows.slice(start, end),
      totalPages: Math.ceil(rows.length / itemsPerPage),
      startRow: start + 1,
      endRow: Math.min(end, rows.length),
    }
  }, [rows, currentPage, itemsPerPage])

  // Reset to page 1 when rows change (e.g., new query results)
  useMemo(() => {
    if (currentPage > totalPages && totalPages > 0) {
      setCurrentPage(1)
    }
  }, [totalPages, currentPage])

  if (useVirtualScrolling) {
    return (
      <VirtualizedTable
        columns={columns}
        rows={visibleRows}
        totalRows={rows.length}
        startRow={startRow}
        endRow={endRow}
        currentPage={currentPage}
        totalPages={totalPages}
        itemsPerPage={itemsPerPage}
        onPageChange={setCurrentPage}
        onItemsPerPageChange={setItemsPerPage}
        className={className}
      />
    )
  }

  return (
    <div className={className}>
      <div className="overflow-auto rounded-md border">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/50">
            <tr>
              {columns.map((col) => (
                <th key={col} className="px-3 py-2 text-left font-medium">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visibleRows.map((row, idx) => (
              <tr key={idx} className="border-b last:border-none hover:bg-muted/50">
                {row.map((cell, cellIdx) => (
                  <td key={cellIdx} className="px-3 py-2">
                    {formatCell(cell)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <TablePagination
        currentPage={currentPage}
        totalPages={totalPages}
        totalRows={rows.length}
        startRow={startRow}
        endRow={endRow}
        itemsPerPage={itemsPerPage}
        onPageChange={setCurrentPage}
        onItemsPerPageChange={setItemsPerPage}
      />
    </div>
  )
}

interface VirtualizedTableProps {
  columns: string[]
  rows: unknown[][]
  totalRows: number
  startRow: number
  endRow: number
  currentPage: number
  totalPages: number
  itemsPerPage: number
  onPageChange: (page: number) => void
  onItemsPerPageChange: (itemsPerPage: number) => void
  className?: string
}

/**
 * Virtualized table implementation using @tanstack/react-virtual.
 * Only renders visible rows plus overscan for smooth scrolling.
 * Ideal for very large datasets (10k+ rows).
 */
function VirtualizedTable({
  columns,
  rows,
  totalRows,
  startRow,
  endRow,
  currentPage,
  totalPages,
  itemsPerPage,
  onPageChange,
  onItemsPerPageChange,
  className = '',
}: VirtualizedTableProps) {
  const parentRef = useRef<HTMLDivElement>(null)

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 35, // Row height in pixels
    overscan: 10, // Number of rows to render outside viewport
  })

  const virtualItems = virtualizer.getVirtualItems()

  return (
    <div className={className}>
      <div ref={parentRef} className="h-[500px] overflow-auto rounded-md border">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10 border-b bg-background">
            <tr className="bg-muted/50">
              {columns.map((col) => (
                <th key={col} className="px-3 py-2 text-left font-medium">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody style={{ height: `${virtualizer.getTotalSize()}px` }} className="relative">
            {virtualItems.map((virtualRow) => {
              const row = rows[virtualRow.index]
              return (
                <tr
                  key={virtualRow.key}
                  className="absolute left-0 top-0 w-full border-b hover:bg-muted/50"
                  style={{
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  {row.map((cell, idx) => (
                    <td key={idx} className="px-3 py-2">
                      {formatCell(cell)}
                    </td>
                  ))}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <TablePagination
        currentPage={currentPage}
        totalPages={totalPages}
        totalRows={totalRows}
        startRow={startRow}
        endRow={endRow}
        itemsPerPage={itemsPerPage}
        onPageChange={onPageChange}
        onItemsPerPageChange={onItemsPerPageChange}
      />
    </div>
  )
}

interface TablePaginationProps {
  currentPage: number
  totalPages: number
  totalRows: number
  startRow: number
  endRow: number
  itemsPerPage: number
  onPageChange: (page: number) => void
  onItemsPerPageChange: (itemsPerPage: number) => void
}

function TablePagination({
  currentPage,
  totalPages,
  totalRows,
  startRow,
  endRow,
  itemsPerPage,
  onPageChange,
  onItemsPerPageChange,
}: TablePaginationProps) {
  const pageNumbers = useMemo(() => {
    const pages: (number | 'ellipsis')[] = []
    const maxVisible = 7 // Maximum page numbers to show

    if (totalPages <= maxVisible) {
      // Show all pages if there aren't many
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i)
      }
    } else {
      // Always show first page
      pages.push(1)

      // Calculate range around current page
      const startPage = Math.max(2, currentPage - 2)
      const endPage = Math.min(totalPages - 1, currentPage + 2)

      // Add ellipsis if needed
      if (startPage > 2) {
        pages.push('ellipsis')
      }

      // Add middle pages
      for (let i = startPage; i <= endPage; i++) {
        pages.push(i)
      }

      // Add ellipsis if needed
      if (endPage < totalPages - 1) {
        pages.push('ellipsis')
      }

      // Always show last page
      pages.push(totalPages)
    }

    return pages
  }, [currentPage, totalPages])

  return (
    <div className="mt-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        <p className="text-sm text-muted-foreground">
          Showing {startRow.toLocaleString()}-{endRow.toLocaleString()} of {totalRows.toLocaleString()} rows
        </p>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Rows per page:</span>
          <Select
            value={String(itemsPerPage)}
            onValueChange={(value) => {
              onItemsPerPageChange(Number(value))
              onPageChange(1) // Reset to first page when changing page size
            }}
          >
            <SelectTrigger className="h-8 w-[70px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="50">50</SelectItem>
              <SelectItem value="100">100</SelectItem>
              <SelectItem value="200">200</SelectItem>
              <SelectItem value="500">500</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {totalPages > 1 && (
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationFirst
                onClick={() => onPageChange(1)}
                className={currentPage === 1 ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationPrevious
                onClick={() => onPageChange(Math.max(1, currentPage - 1))}
                className={currentPage === 1 ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>

            {pageNumbers.map((page, idx) =>
              page === 'ellipsis' ? (
                <PaginationItem key={`ellipsis-${idx}`}>
                  <PaginationEllipsis />
                </PaginationItem>
              ) : (
                <PaginationItem key={page}>
                  <Button
                    variant={currentPage === page ? 'outline' : 'ghost'}
                    size="icon"
                    onClick={() => onPageChange(page)}
                  >
                    {page}
                  </Button>
                </PaginationItem>
              )
            )}

            <PaginationItem>
              <PaginationNext
                onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
                className={currentPage === totalPages ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationLast
                onClick={() => onPageChange(totalPages)}
                className={currentPage === totalPages ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  )
}
