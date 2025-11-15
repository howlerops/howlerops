import { useCallback, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react'
import { Button } from './ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Input } from './ui/input'

export interface PaginationControlsProps {
  currentPage: number
  pageSize: number
  totalRows: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  disabled?: boolean
  compact?: boolean
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 500, 1000]

export const PaginationControls = ({
  currentPage,
  pageSize,
  totalRows,
  onPageChange,
  onPageSizeChange,
  disabled = false,
  compact = false,
}: PaginationControlsProps) => {
  const [jumpToPage, setJumpToPage] = useState('')

  const totalPages = useMemo(() => Math.max(1, Math.ceil(totalRows / pageSize)), [totalRows, pageSize])
  const startRow = useMemo(() => Math.min(totalRows, (currentPage - 1) * pageSize + 1), [currentPage, pageSize, totalRows])
  const endRow = useMemo(() => Math.min(totalRows, currentPage * pageSize), [currentPage, pageSize, totalRows])

  const canGoPrevious = currentPage > 1 && !disabled
  const canGoNext = currentPage < totalPages && !disabled

  const handleFirstPage = useCallback(() => {
    if (canGoPrevious) {
      onPageChange(1)
    }
  }, [canGoPrevious, onPageChange])

  const handlePreviousPage = useCallback(() => {
    if (canGoPrevious) {
      onPageChange(currentPage - 1)
    }
  }, [canGoPrevious, currentPage, onPageChange])

  const handleNextPage = useCallback(() => {
    if (canGoNext) {
      onPageChange(currentPage + 1)
    }
  }, [canGoNext, currentPage, onPageChange])

  const handleLastPage = useCallback(() => {
    if (canGoNext) {
      onPageChange(totalPages)
    }
  }, [canGoNext, totalPages, onPageChange])

  const handleJumpToPage = useCallback(() => {
    const page = parseInt(jumpToPage, 10)
    if (!isNaN(page) && page >= 1 && page <= totalPages) {
      onPageChange(page)
      setJumpToPage('')
    }
  }, [jumpToPage, totalPages, onPageChange])

  const handleJumpKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleJumpToPage()
    } else if (e.key === 'Escape') {
      setJumpToPage('')
      ;(e.target as HTMLInputElement).blur()
    }
  }, [handleJumpToPage])

  const handlePageSizeChange = useCallback((value: string) => {
    const newSize = parseInt(value, 10)
    if (!isNaN(newSize)) {
      onPageSizeChange(newSize)
      // Adjust current page if needed to stay within bounds
      const newTotalPages = Math.ceil(totalRows / newSize)
      if (currentPage > newTotalPages) {
        onPageChange(Math.max(1, newTotalPages))
      }
    }
  }, [onPageSizeChange, totalRows, currentPage, onPageChange])

  if (totalRows === 0) {
    return null
  }

  return (
    <div className={`flex items-center ${compact ? 'gap-2' : 'gap-4'} ${compact ? 'text-xs' : 'text-sm'}`}>
      {/* Page size selector */}
      <div className="flex items-center gap-2">
        <span className="text-muted-foreground whitespace-nowrap">Rows per page:</span>
        <Select
          value={pageSize.toString()}
          onValueChange={handlePageSizeChange}
          disabled={disabled}
        >
          <SelectTrigger className={`${compact ? 'h-8 w-20' : 'h-9 w-24'}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PAGE_SIZE_OPTIONS.map((size) => (
              <SelectItem key={size} value={size.toString()}>
                {size.toLocaleString()}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Row range display */}
      <div className="text-muted-foreground whitespace-nowrap">
        {startRow.toLocaleString()}–{endRow.toLocaleString()} of {totalRows.toLocaleString()}
      </div>

      {/* Navigation buttons */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size={compact ? 'sm' : 'icon'}
          onClick={handleFirstPage}
          disabled={!canGoPrevious}
          title="First page"
          className={compact ? 'h-8 w-8' : 'h-9 w-9'}
        >
          <ChevronsLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size={compact ? 'sm' : 'icon'}
          onClick={handlePreviousPage}
          disabled={!canGoPrevious}
          title="Previous page"
          className={compact ? 'h-8 w-8' : 'h-9 w-9'}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>

        {/* Page indicator / jump to page */}
        <div className="flex items-center gap-2 mx-2">
          <span className="text-muted-foreground">Page</span>
          <Input
            type="text"
            value={jumpToPage || currentPage.toString()}
            onChange={(e) => setJumpToPage(e.target.value)}
            onKeyDown={handleJumpKeyDown}
            onBlur={() => setJumpToPage('')}
            disabled={disabled}
            className={`${compact ? 'h-8 w-16' : 'h-9 w-20'} text-center`}
            title="Enter page number and press Enter"
          />
          <span className="text-muted-foreground">of {totalPages.toLocaleString()}</span>
        </div>

        <Button
          variant="ghost"
          size={compact ? 'sm' : 'icon'}
          onClick={handleNextPage}
          disabled={!canGoNext}
          title="Next page"
          className={compact ? 'h-8 w-8' : 'h-9 w-9'}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size={compact ? 'sm' : 'icon'}
          onClick={handleLastPage}
          disabled={!canGoNext}
          title="Last page"
          className={compact ? 'h-8 w-8' : 'h-9 w-9'}
        >
          <ChevronsRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
