import { Filter, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

interface CrossFilterBarProps {
  activeFilters: Record<string, unknown>
  onClearFilter: (field: string) => void
  onClearAll: () => void
  className?: string
}

/**
 * Display bar showing active cross-filters
 *
 * Features:
 * - Shows all active filters as badges
 * - Individual filter removal
 * - Clear all filters button
 * - Keyboard shortcut hint (Alt+C)
 */
export function CrossFilterBar({
  activeFilters,
  onClearFilter,
  onClearAll,
  className,
}: CrossFilterBarProps) {
  const filterCount = Object.keys(activeFilters).length

  if (filterCount === 0) return null

  return (
    <Card className={className}>
      <CardContent className="py-3">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 flex-wrap min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-shrink-0">
              <Filter className="h-4 w-4 text-primary" />
              <span className="text-sm font-medium">Active Filters:</span>
            </div>
            <div className="flex items-center gap-2 flex-wrap min-w-0">
              {Object.entries(activeFilters).map(([field, value]) => (
                <Badge key={field} variant="secondary" className="gap-1.5">
                  <span className="truncate max-w-[200px]">
                    {field} = {formatFilterValue(value)}
                  </span>
                  <button
                    type="button"
                    className="ml-1 hover:text-foreground"
                    onClick={() => onClearFilter(field)}
                    aria-label={`Remove ${field} filter`}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClearAll}
            className="flex-shrink-0"
            title="Alt+C to clear all"
          >
            Clear All
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Format filter value for display (truncate if too long)
 */
function formatFilterValue(value: unknown): string {
  if (value === null || value === undefined) {
    return 'null'
  }

  const str = String(value)
  const maxLength = 40

  if (str.length > maxLength) {
    return str.substring(0, maxLength - 3) + '...'
  }

  return str
}
