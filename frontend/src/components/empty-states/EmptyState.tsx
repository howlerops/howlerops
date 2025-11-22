import { Lightbulb, LucideIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface ExampleQuery {
  label: string
  description?: string
  query: string
  icon?: LucideIcon
}

interface EmptyStateProps {
  icon?: LucideIcon
  illustration?: string
  title: string
  description: string
  primaryAction?: {
    label: string
    onClick: () => void
  }
  secondaryAction?: {
    label: string
    onClick: () => void
  }
  examples?: ExampleQuery[]
  onExampleClick?: (query: string) => void
  className?: string
}

export function EmptyState({
  icon: Icon,
  illustration,
  title,
  description,
  primaryAction,
  secondaryAction,
  examples,
  onExampleClick,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center py-12 px-4",
        className
      )}
    >
      {/* Illustration or Icon */}
      {illustration ? (
        <img
          src={illustration}
          alt={title}
          className="w-64 h-64 object-contain mb-6 opacity-80"
        />
      ) : Icon ? (
        <div className="w-24 h-24 rounded-full bg-muted flex items-center justify-center mb-6">
          <Icon className="w-12 h-12 text-muted-foreground" />
        </div>
      ) : null}

      {/* Content */}
      <div className="max-w-md space-y-2 mb-6">
        <h3 className="text-xl font-semibold">{title}</h3>
        <p className="text-muted-foreground">{description}</p>
      </div>

      {/* Actions */}
      {(primaryAction || secondaryAction) && (
        <div className="flex items-center gap-3 mb-6">
          {primaryAction && (
            <Button onClick={primaryAction.onClick}>
              {primaryAction.label}
            </Button>
          )}
          {secondaryAction && (
            <Button variant="outline" onClick={secondaryAction.onClick}>
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}

      {/* Example Queries */}
      {examples && examples.length > 0 && onExampleClick && (
        <div className="w-full max-w-2xl space-y-3">
          <div className="flex items-center justify-center gap-2 text-sm font-medium text-muted-foreground mb-4">
            <Lightbulb className="h-4 w-4" />
            <span>Try these examples</span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {examples.map((example, index) => {
              const ExampleIcon = example.icon
              return (
                <Card
                  key={index}
                  className="p-4 hover:bg-muted/50 transition-colors cursor-pointer group text-left"
                  onClick={() => onExampleClick(example.query)}
                >
                  <div className="flex items-start gap-3">
                    {ExampleIcon && (
                      <div className="flex-shrink-0 w-8 h-8 rounded-md bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                        <ExampleIcon className="h-4 w-4 text-primary" />
                      </div>
                    )}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium mb-1 group-hover:text-primary transition-colors">
                        {example.label}
                      </p>
                      {example.description && (
                        <p className="text-xs text-muted-foreground line-clamp-2">
                          {example.description}
                        </p>
                      )}
                    </div>
                  </div>
                </Card>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
