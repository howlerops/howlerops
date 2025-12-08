import { Loader2 } from "lucide-react"
import { useEffect, useState } from "react"

import { cn } from "@/lib/utils"

interface StreamingIndicatorProps {
  isStreaming: boolean
  startTime?: number
  stage?: string
  className?: string
}

/**
 * StreamingIndicator - Shows streaming status with elapsed time
 *
 * Features:
 * - Animated typing indicator with bouncing dots
 * - Elapsed time display (shows after 2s)
 * - Optional stage display (e.g., "Generating SQL", "Executing query")
 * - Smooth fade-in/out transitions
 */
export function StreamingIndicator({
  isStreaming,
  startTime,
  stage,
  className,
}: StreamingIndicatorProps) {
  const [elapsedSeconds, setElapsedSeconds] = useState(0)

  useEffect(() => {
    if (!isStreaming) {
      setElapsedSeconds(0)
      return
    }

    const start = startTime ?? Date.now()
    const interval = setInterval(() => {
      setElapsedSeconds(Math.floor((Date.now() - start) / 1000))
    }, 1000)

    return () => clearInterval(interval)
  }, [isStreaming, startTime])

  if (!isStreaming) {
    return null
  }

  return (
    <div
      className={cn(
        "flex items-center gap-3 px-4 py-2 rounded-lg bg-primary/5 border border-primary/20 animate-in fade-in-0 slide-in-from-bottom-2 duration-300",
        className
      )}
      role="status"
      aria-live="polite"
    >
      {/* Animated typing dots */}
      <div className="flex items-center gap-1">
        <span
          className="h-2 w-2 rounded-full bg-primary animate-bounce"
          style={{ animationDelay: "0ms" }}
        />
        <span
          className="h-2 w-2 rounded-full bg-primary animate-bounce"
          style={{ animationDelay: "150ms" }}
        />
        <span
          className="h-2 w-2 rounded-full bg-primary animate-bounce"
          style={{ animationDelay: "300ms" }}
        />
      </div>

      {/* Status text */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin text-primary" />
        <span className="font-medium">
          {stage || "AI is thinking"}
          {elapsedSeconds >= 2 && (
            <span className="ml-2 text-xs opacity-70">
              ({elapsedSeconds}s)
            </span>
          )}
        </span>
      </div>

      {/* Slow response warning */}
      {elapsedSeconds >= 15 && (
        <span className="text-xs text-amber-500 animate-pulse">
          Taking longer than usual...
        </span>
      )}
    </div>
  )
}

/**
 * Compact streaming indicator for inline use
 */
export function StreamingIndicatorCompact({
  isStreaming,
  className,
}: {
  isStreaming: boolean
  className?: string
}) {
  if (!isStreaming) {
    return null
  }

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 text-sm text-muted-foreground",
        className
      )}
      role="status"
      aria-label="AI is responding"
    >
      <span
        className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce"
        style={{ animationDelay: "0ms" }}
      />
      <span
        className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce"
        style={{ animationDelay: "150ms" }}
      />
      <span
        className="h-1.5 w-1.5 rounded-full bg-primary animate-bounce"
        style={{ animationDelay: "300ms" }}
      />
    </div>
  )
}
