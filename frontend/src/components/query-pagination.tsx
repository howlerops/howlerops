import { useState } from 'react'

import { Button } from './ui/button'
import { Input } from './ui/input'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationFirst,
  PaginationItem,
  PaginationLast,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from './ui/pagination'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'

interface QueryPaginationProps {
  currentPage: number
  totalPages: number
  pageSize: number
  totalRows: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  loading?: boolean
}

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500, 1000]

export function QueryPagination({
  currentPage,
  totalPages,
  pageSize,
  totalRows,
  onPageChange,
  onPageSizeChange,
  loading = false,
}: QueryPaginationProps) {
  const [jumpToPage, setJumpToPage] = useState('')

  const handleJumpToPage = () => {
    const page = parseInt(jumpToPage, 10)
    if (page >= 1 && page <= totalPages) {
      onPageChange(page)
      setJumpToPage('')
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleJumpToPage()
    }
  }

  // Calculate page numbers to show
  const getPageNumbers = () => {
    const pages: (number | 'ellipsis')[] = []
    const showEllipsisThreshold = 7 // Total number of pages to show before using ellipsis

    if (totalPages <= showEllipsisThreshold) {
      // Show all pages if total is small enough
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i)
      }
    } else {
      // Always show first page
      pages.push(1)

      if (currentPage > 3) {
        pages.push('ellipsis')
      }

      // Show pages around current page
      const start = Math.max(2, currentPage - 1)
      const end = Math.min(totalPages - 1, currentPage + 1)

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }

      if (currentPage < totalPages - 2) {
        pages.push('ellipsis')
      }

      // Always show last page
      if (totalPages > 1) {
        pages.push(totalPages)
      }
    }

    return pages
  }

  const pageNumbers = getPageNumbers()
  const startRow = (currentPage - 1) * pageSize + 1
  const endRow = Math.min(currentPage * pageSize, totalRows)

  return (
    <div className="flex items-center justify-between px-2 py-3 border-t border-border bg-background">
      <div className="flex items-center gap-4 text-sm text-muted-foreground">
        <span>
          Showing {startRow.toLocaleString()}-{endRow.toLocaleString()} of{' '}
          {totalRows.toLocaleString()} rows
        </span>
        <div className="flex items-center gap-2">
          <span>Rows per page:</span>
          <Select
            value={pageSize.toString()}
            onValueChange={(value) => onPageSizeChange(parseInt(value, 10))}
            disabled={loading}
          >
            <SelectTrigger className="h-8 w-20">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZE_OPTIONS.map((size) => (
                <SelectItem key={size} value={size.toString()}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex items-center gap-4">
        {/* Jump to page */}
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">Jump to:</span>
          <Input
            type="number"
            min={1}
            max={totalPages}
            value={jumpToPage}
            onChange={(e) => setJumpToPage(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="Page"
            className="h-8 w-20"
            disabled={loading}
          />
          <Button
            size="sm"
            variant="outline"
            onClick={handleJumpToPage}
            disabled={loading || !jumpToPage}
            className="h-8"
          >
            Go
          </Button>
        </div>

        {/* Pagination controls */}
        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationFirst
                onClick={() => onPageChange(1)}
                className={currentPage === 1 || loading ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationPrevious
                onClick={() => onPageChange(currentPage - 1)}
                className={currentPage === 1 || loading ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>

            {pageNumbers.map((page, index) => (
              <PaginationItem key={index}>
                {page === 'ellipsis' ? (
                  <PaginationEllipsis />
                ) : (
                  <PaginationLink
                    onClick={() => onPageChange(page)}
                    isActive={currentPage === page}
                    className={loading ? 'pointer-events-none opacity-50' : ''}
                  >
                    {page}
                  </PaginationLink>
                )}
              </PaginationItem>
            ))}

            <PaginationItem>
              <PaginationNext
                onClick={() => onPageChange(currentPage + 1)}
                className={currentPage === totalPages || loading ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationLast
                onClick={() => onPageChange(totalPages)}
                className={currentPage === totalPages || loading ? 'pointer-events-none opacity-50' : ''}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  )
}
