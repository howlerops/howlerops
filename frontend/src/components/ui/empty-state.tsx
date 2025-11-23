import type { LucideIcon } from 'lucide-react'
import * as React from 'react'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description: string
  action?: React.ReactNode
  className?: string
  compact?: boolean
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
  compact = false,
}: EmptyStateProps) {
  if (compact) {
    return (
      <div className={`flex flex-col items-center justify-center p-6 text-center ${className || ''}`}>
        <Icon className="h-6 w-6 text-muted-foreground mb-2" />
        <p className="text-xs font-medium text-muted-foreground">{title}</p>
        <p className="text-xs text-muted-foreground/70 mt-1">{description}</p>
        {action && <div className="mt-3">{action}</div>}
      </div>
    )
  }

  return (
    <div
      className={`flex flex-col items-center justify-center p-12 text-center border-2 border-dashed rounded-lg ${className || ''}`}
    >
      <div className="rounded-full bg-primary/10 p-3 mb-4">
        <Icon className="h-8 w-8 text-primary" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground mb-4 max-w-sm">
        {description}
      </p>
      {action}
    </div>
  )
}
