import { ChevronRight, Home } from 'lucide-react'
import { Fragment } from 'react'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { DrillDownAction } from '@/types/reports'

interface DrillDownBreadcrumbsProps {
  history: DrillDownAction[]
  onNavigate: (index: number) => void
  className?: string
}

/**
 * Breadcrumb navigation showing drill-down path
 *
 * Allows users to navigate back through drill-down history
 * Shows clear path from dashboard to current detail view
 */
export function DrillDownBreadcrumbs({ history, onNavigate, className }: DrillDownBreadcrumbsProps) {
  if (history.length === 0) return null

  return (
    <div className={cn('flex items-center gap-1 text-sm overflow-x-auto py-2', className)}>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => onNavigate(-1)}
        className="h-7 px-2 hover:bg-muted"
      >
        <Home className="h-3.5 w-3.5 mr-1.5" />
        Dashboard
      </Button>

      {history.map((action, idx) => (
        <Fragment key={action.timestamp.toISOString()}>
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onNavigate(idx)}
            className={cn(
              'h-7 px-2 hover:bg-muted',
              idx === history.length - 1 && 'font-medium text-foreground'
            )}
          >
            {formatActionLabel(action)}
          </Button>
        </Fragment>
      ))}
    </div>
  )
}

/**
 * Format drill-down action for display in breadcrumbs
 */
function formatActionLabel(action: DrillDownAction): string {
  const maxLength = 30

  let label = ''
  switch (action.type) {
    case 'detail':
      label = `${action.context.field}: ${String(action.context.clickedValue)}`
      break
    case 'filter':
      label = `Filter: ${action.config.filterField}`
      break
    case 'related-report':
      label = action.config.target || 'Related Report'
      break
    case 'url':
      label = 'External Link'
      break
    default:
      label = 'Unknown'
  }

  // Truncate if too long
  if (label.length > maxLength) {
    return label.substring(0, maxLength - 3) + '...'
  }

  return label
}
