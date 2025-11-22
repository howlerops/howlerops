import { CheckCircle2, Circle } from "lucide-react"

import { Card } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface ChecklistItem {
  id: string
  label: string
  completed: boolean
  onClick?: () => void
}

interface OnboardingChecklistProps {
  items: ChecklistItem[]
  title?: string
  className?: string
}

export function OnboardingChecklist({
  items,
  title = "Getting Started",
  className,
}: OnboardingChecklistProps) {
  const completedCount = items.filter((item) => item.completed).length
  const totalCount = items.length
  const allCompleted = completedCount === totalCount

  return (
    <Card className={cn("p-4", className)}>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">{title}</h3>
          <span className="text-xs text-muted-foreground">
            {completedCount}/{totalCount}
          </span>
        </div>

        <div className="space-y-2">
          {items.map((item) => (
            <button
              key={item.id}
              onClick={item.onClick}
              disabled={!item.onClick}
              className={cn(
                "w-full flex items-center gap-3 p-2 rounded-md transition-colors",
                item.onClick && "hover:bg-muted cursor-pointer",
                !item.onClick && "cursor-default"
              )}
            >
              {item.completed ? (
                <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
              ) : (
                <Circle className="h-5 w-5 text-muted-foreground flex-shrink-0" />
              )}
              <span
                className={cn(
                  "text-sm text-left",
                  item.completed
                    ? "text-muted-foreground line-through"
                    : "text-foreground"
                )}
              >
                {item.label}
              </span>
            </button>
          ))}
        </div>

        {allCompleted && (
          <div className="pt-2 border-t border-border">
            <p className="text-sm text-green-600 dark:text-green-400 font-medium text-center">
              All done! You're ready to go!
            </p>
          </div>
        )}
      </div>
    </Card>
  )
}
