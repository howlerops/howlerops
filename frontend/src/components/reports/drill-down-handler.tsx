import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import type {
  DrillDownAction,
  DrillDownConfig,
  DrillDownContext,
  ReportRunComponentResult,
} from '@/types/reports'

interface UseDrillDownOptions {
  onFilterChange?: (filters: Record<string, unknown>) => void
  executeQuery?: (sql: string, connectionId?: string) => Promise<ReportRunComponentResult>
}

interface UseDrillDownReturn {
  executeDrillDown: (config: DrillDownConfig, context: DrillDownContext) => Promise<void>
  goBack: () => void
  canGoBack: boolean
  history: DrillDownAction[]
  detailDrawerOpen: boolean
  detailData: ReportRunComponentResult | null
  detailLoading: boolean
  detailError: string | null
  closeDetailDrawer: () => void
  activeFilters: Record<string, unknown>
  clearFilter: (field: string) => void
  clearAllFilters: () => void
}

/**
 * Hook for managing drill-down interactions across report components
 *
 * Handles:
 * - Click-through to detail views
 * - Cross-filtering between components
 * - Navigation to related reports
 * - External URL linking
 * - History tracking for back navigation
 * - URL state synchronization
 */
export function useDrillDown(options: UseDrillDownOptions = {}): UseDrillDownReturn {
  const { onFilterChange, executeQuery } = options
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const [history, setHistory] = useState<DrillDownAction[]>([])
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false)
  const [detailData, setDetailData] = useState<ReportRunComponentResult | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [activeFilters, setActiveFilters] = useState<Record<string, unknown>>({})

  // Initialize filters from URL on mount
  useEffect(() => {
    const filtersParam = searchParams.get('filters')
    if (filtersParam) {
      try {
        const parsed = JSON.parse(decodeURIComponent(filtersParam))
        setActiveFilters(parsed)
        onFilterChange?.(parsed)
      } catch (err) {
        console.error('Failed to parse filters from URL:', err)
      }
    }
  }, []) // Only run on mount

  // Update URL when filters change
  const updateUrlFilters = useCallback(
    (filters: Record<string, unknown>) => {
      const params = new URLSearchParams(searchParams)
      if (Object.keys(filters).length > 0) {
        params.set('filters', encodeURIComponent(JSON.stringify(filters)))
      } else {
        params.delete('filters')
      }
      setSearchParams(params, { replace: true })
    },
    [searchParams, setSearchParams]
  )

  // Execute drill-down action
  const executeDrillDown = useCallback(
    async (config: DrillDownConfig, context: DrillDownContext) => {
      const action: DrillDownAction = {
        type: config.type,
        componentId: context.componentId || '',
        context,
        timestamp: new Date(),
        config,
      }

      setHistory((prev) => [...prev, action])

      try {
        switch (config.type) {
          case 'detail':
            await showDetail(config, context)
            break

          case 'filter':
            applyFilter(config, context)
            break

          case 'related-report':
            navigateToReport(config.target!, context, config)
            break

          case 'url':
            openUrl(config.target!, context)
            break
        }
      } catch (err) {
        console.error('Drill-down execution failed:', err)
        setDetailError(err instanceof Error ? err.message : 'Unknown error occurred')
      }
    },
    [executeQuery, onFilterChange, navigate]
  )

  // Show detail drawer with query results
  const showDetail = async (config: DrillDownConfig, context: DrillDownContext) => {
    if (!config.detailQuery || !executeQuery) {
      throw new Error('Detail query or executeQuery function not provided')
    }

    setDetailDrawerOpen(true)
    setDetailLoading(true)
    setDetailError(null)
    setDetailData(null)

    try {
      // Interpolate clicked value into query
      const interpolatedSql = interpolateQuery(config.detailQuery, {
        clickedValue: context.clickedValue,
        ...context.filters,
      })

      const result = await executeQuery(interpolatedSql)

      if (result.error) {
        throw new Error(result.error)
      }

      setDetailData(result)
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to load detail data'
      setDetailError(errorMsg)
      throw err
    } finally {
      setDetailLoading(false)
    }
  }

  // Apply cross-filter to other components
  const applyFilter = (config: DrillDownConfig, context: DrillDownContext) => {
    if (!config.filterField) {
      throw new Error('Filter field not specified in drill-down config')
    }

    const newFilters = {
      ...activeFilters,
      [config.filterField]: context.clickedValue,
    }

    setActiveFilters(newFilters)
    updateUrlFilters(newFilters)
    onFilterChange?.(newFilters)
  }

  // Navigate to related report
  const navigateToReport = (reportId: string, context: DrillDownContext, config: DrillDownConfig) => {
    const params = new URLSearchParams()

    // Pass context as URL parameters
    if (config.parameters) {
      Object.entries(config.parameters).forEach(([paramKey, contextKey]) => {
        const value = contextKey === 'clickedValue' ? context.clickedValue : context.filters?.[contextKey]
        if (value !== undefined) {
          params.set(paramKey, String(value))
        }
      })
    }

    const url = `/reports/${reportId}${params.toString() ? `?${params.toString()}` : ''}`
    navigate(url)
  }

  // Open external URL with interpolated parameters
  const openUrl = (urlTemplate: string, context: DrillDownContext) => {
    const interpolatedUrl = interpolateQuery(urlTemplate, {
      clickedValue: context.clickedValue,
      ...context.filters,
    })

    window.open(interpolatedUrl, '_blank', 'noopener,noreferrer')
  }

  // Go back in drill-down history
  const goBack = useCallback(() => {
    if (history.length === 0) return

    // Remove last action
    setHistory((prev) => prev.slice(0, -1))

    // If we were showing details, close the drawer
    if (detailDrawerOpen) {
      setDetailDrawerOpen(false)
      setDetailData(null)
      setDetailError(null)
    }

    // If the last action was a filter, remove it
    const lastAction = history[history.length - 1]
    if (lastAction?.type === 'filter' && lastAction.config.filterField) {
      const newFilters = { ...activeFilters }
      delete newFilters[lastAction.config.filterField]
      setActiveFilters(newFilters)
      updateUrlFilters(newFilters)
      onFilterChange?.(newFilters)
    }
  }, [history, detailDrawerOpen, activeFilters, updateUrlFilters, onFilterChange])

  // Clear single filter
  const clearFilter = useCallback(
    (field: string) => {
      const newFilters = { ...activeFilters }
      delete newFilters[field]
      setActiveFilters(newFilters)
      updateUrlFilters(newFilters)
      onFilterChange?.(newFilters)
    },
    [activeFilters, updateUrlFilters, onFilterChange]
  )

  // Clear all filters
  const clearAllFilters = useCallback(() => {
    setActiveFilters({})
    updateUrlFilters({})
    onFilterChange?.({})
  }, [updateUrlFilters, onFilterChange])

  // Close detail drawer
  const closeDetailDrawer = useCallback(() => {
    setDetailDrawerOpen(false)
    setDetailData(null)
    setDetailError(null)
  }, [])

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyboard = (e: KeyboardEvent) => {
      // Alt + ← = Go back in drill-down history
      if (e.altKey && e.key === 'ArrowLeft' && history.length > 0) {
        e.preventDefault()
        goBack()
      }

      // Esc = Close detail drawer
      if (e.key === 'Escape' && detailDrawerOpen) {
        closeDetailDrawer()
      }

      // Alt + C = Clear all filters
      if (e.altKey && e.key === 'c' && Object.keys(activeFilters).length > 0) {
        e.preventDefault()
        clearAllFilters()
      }
    }

    window.addEventListener('keydown', handleKeyboard)
    return () => window.removeEventListener('keydown', handleKeyboard)
  }, [history.length, detailDrawerOpen, activeFilters, goBack, closeDetailDrawer, clearAllFilters])

  return {
    executeDrillDown,
    goBack,
    canGoBack: history.length > 0,
    history,
    detailDrawerOpen,
    detailData,
    detailLoading,
    detailError,
    closeDetailDrawer,
    activeFilters,
    clearFilter,
    clearAllFilters,
  }
}

/**
 * Interpolate query template with values from context
 *
 * Supports:
 * - :paramName - Named parameters
 * - {paramName} - Brace-wrapped parameters
 *
 * Example:
 *   interpolateQuery("SELECT * FROM users WHERE id = :clickedValue", { clickedValue: 123 })
 *   => "SELECT * FROM users WHERE id = 123"
 */
function interpolateQuery(template: string, values: Record<string, unknown>): string {
  let result = template

  // Replace :paramName style
  Object.entries(values).forEach(([key, value]) => {
    const placeholder = `:${key}`
    if (result.includes(placeholder)) {
      const interpolated = typeof value === 'string' ? `'${value}'` : String(value)
      result = result.replaceAll(placeholder, interpolated)
    }
  })

  // Replace {paramName} style
  Object.entries(values).forEach(([key, value]) => {
    const placeholder = `{${key}}`
    if (result.includes(placeholder)) {
      const interpolated = typeof value === 'string' ? `'${value}'` : String(value)
      result = result.replaceAll(placeholder, interpolated)
    }
  })

  return result
}

/**
 * Format drill-down action for display in breadcrumbs
 */
export function formatActionLabel(action: DrillDownAction): string {
  switch (action.type) {
    case 'detail':
      return `Details: ${action.context.field} = ${String(action.context.clickedValue)}`
    case 'filter':
      return `Filter: ${action.config.filterField} = ${String(action.context.clickedValue)}`
    case 'related-report':
      return `Report: ${action.config.target}`
    case 'url':
      return 'External Link'
    default:
      return 'Unknown Action'
  }
}
