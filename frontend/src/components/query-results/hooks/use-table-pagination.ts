import { useCallback, useEffect, useState } from 'react'

import { toast } from '../../../hooks/use-toast'

interface UseTablePaginationOptions {
  resultId: string
  offset?: number
  onPageChange?: (limit: number, offset: number) => void
}

interface UseTablePaginationReturn {
  currentPage: number
  pageSize: number
  isLoadingPage: boolean
  handlePageChange: (newPage: number) => Promise<void>
  handlePageSizeChange: (newPageSize: number) => Promise<void>
}

export function useTablePagination({
  resultId,
  offset = 0,
  onPageChange,
}: UseTablePaginationOptions): UseTablePaginationReturn {
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(100)
  const [isLoadingPage, setIsLoadingPage] = useState(false)

  // Sync current page with offset from backend
  useEffect(() => {
    if (offset !== undefined && pageSize > 0) {
      const calculatedPage = Math.floor(offset / pageSize) + 1
      if (calculatedPage !== currentPage) {
        setCurrentPage(calculatedPage)
      }
    }
  }, [offset, pageSize, currentPage])

  // Reset to page 1 when query changes
  useEffect(() => {
    setCurrentPage(1)
    setIsLoadingPage(false)
  }, [resultId])

  const handlePageChange = useCallback(async (newPage: number) => {
    if (!onPageChange || isLoadingPage) return

    const newOffset = (newPage - 1) * pageSize
    setIsLoadingPage(true)
    setCurrentPage(newPage)

    try {
      await onPageChange(pageSize, newOffset)
    } catch (error) {
      console.error('Page change failed:', error)
      toast({
        title: 'Page change failed',
        description: error instanceof Error ? error.message : 'Failed to load page',
        variant: 'destructive'
      })
      // Revert to previous page on error
      setCurrentPage(Math.floor(offset / pageSize) + 1)
    } finally {
      setIsLoadingPage(false)
    }
  }, [onPageChange, pageSize, offset, isLoadingPage])

  const handlePageSizeChange = useCallback(async (newPageSize: number) => {
    if (!onPageChange || isLoadingPage) return

    // Calculate what page we should be on to show similar rows
    const currentFirstRow = (currentPage - 1) * pageSize
    const newPage = Math.floor(currentFirstRow / newPageSize) + 1

    setPageSize(newPageSize)
    setIsLoadingPage(true)
    setCurrentPage(newPage)

    try {
      await onPageChange(newPageSize, (newPage - 1) * newPageSize)
    } catch (error) {
      console.error('Page size change failed:', error)
      toast({
        title: 'Page size change failed',
        description: error instanceof Error ? error.message : 'Failed to change page size',
        variant: 'destructive'
      })
      // Revert to previous page size on error
      setPageSize(pageSize)
      setCurrentPage(Math.floor(offset / pageSize) + 1)
    } finally {
      setIsLoadingPage(false)
    }
  }, [onPageChange, currentPage, pageSize, offset, isLoadingPage])

  return {
    currentPage,
    pageSize,
    isLoadingPage,
    handlePageChange,
    handlePageSizeChange,
  }
}
