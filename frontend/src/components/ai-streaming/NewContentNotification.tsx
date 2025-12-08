import { ArrowDown, Sparkles } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface NewContentNotificationProps {
  show: boolean
  onClick: () => void
  newMessageCount?: number
  className?: string
}

/**
 * NewContentNotification - Floating "Jump to latest" button
 *
 * Appears when:
 * - User has scrolled up and new messages arrive
 * - New content is below the visible viewport
 *
 * Features:
 * - Smooth slide-in animation from bottom
 * - Shows new message count badge
 * - Click scrolls to latest content with smooth behavior
 */
export function NewContentNotification({
  show,
  onClick,
  newMessageCount = 0,
  className,
}: NewContentNotificationProps) {
  if (!show) {
    return null
  }

  return (
    <div
      className={cn(
        "absolute bottom-4 left-1/2 -translate-x-1/2 z-50",
        "animate-in slide-in-from-bottom-4 fade-in-0 duration-300",
        className
      )}
    >
      <Button
        variant="default"
        size="sm"
        onClick={onClick}
        className="gap-2 shadow-lg rounded-full px-4 py-2 bg-primary hover:bg-primary/90"
      >
        <Sparkles className="h-4 w-4" />
        <span>
          {newMessageCount > 0
            ? `${newMessageCount} new response${newMessageCount > 1 ? 's' : ''}`
            : 'New response'}
        </span>
        <ArrowDown className="h-4 w-4" />
      </Button>
    </div>
  )
}

/**
 * NewContentBadge - Subtle inline badge for new content indicator
 */
export function NewContentBadge({
  show,
  className,
}: {
  show: boolean
  className?: string
}) {
  if (!show) {
    return null
  }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full",
        "bg-primary/10 text-primary border border-primary/20",
        "animate-pulse",
        className
      )}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-primary" />
      New
    </span>
  )
}

/**
 * HighlightedMessage - Wrapper to highlight newly arrived messages
 */
export function HighlightedMessage({
  children,
  isNew,
  className,
}: {
  children: React.ReactNode
  isNew: boolean
  className?: string
}) {
  return (
    <div
      className={cn(
        "transition-all duration-500",
        isNew && "animate-highlight-fade",
        className
      )}
    >
      {children}
    </div>
  )
}
