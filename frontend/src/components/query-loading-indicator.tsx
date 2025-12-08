import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'

interface QueryLoadingIndicatorProps {
  startTime: Date
  query?: string
}

export const QueryLoadingIndicator = ({ startTime }: QueryLoadingIndicatorProps) => {
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    // Update elapsed time every 100ms for smooth updates
    const interval = setInterval(() => {
      const now = new Date()
      const elapsedMs = now.getTime() - startTime.getTime()
      setElapsed(elapsedMs)
    }, 100)

    return () => clearInterval(interval)
  }, [startTime])

  const formatElapsed = (ms: number): string => {
    const seconds = ms / 1000
    if (seconds < 60) {
      return `${seconds.toFixed(1)}s`
    }
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = Math.floor(seconds % 60)
    return `${minutes}m ${remainingSeconds}s`
  }

  const elapsedSeconds = Math.floor(elapsed / 1000)

  return (
    <div className="flex items-center justify-center h-full p-4">
      <div className="flex flex-col items-center gap-4 max-w-2xl">
        <Loader2 className="h-12 w-12 animate-spin text-primary" />

        <div className="text-center">
          <h3 className="text-lg font-semibold mb-1">Executing Query</h3>
          <p className="text-sm text-muted-foreground mb-3">
            Please wait while your query is being processed...
          </p>

          {/* Bouncing dots indicator */}
          <div className="flex items-center justify-center gap-1 mb-3">
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

          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-muted">
            <span className="text-2xl font-mono font-semibold tabular-nums">
              {formatElapsed(elapsed)}
            </span>
          </div>
        </div>

        {/* {query && (
          <div className="w-full mt-4 p-4 rounded-lg bg-muted/50 border border-border">
            <p className="text-xs text-muted-foreground mb-2 font-semibold">Query:</p>
            <pre className="text-sm font-mono overflow-x-auto max-h-32 overflow-y-auto">
              {query}
            </pre>
          </div>
        )} */}

        {elapsedSeconds >= 15 && (
          <div className="mt-2 text-xs text-amber-500 animate-pulse">
            Taking longer than usual...
          </div>
        )}

        {elapsedSeconds >= 30 && (
          <div className="mt-2 px-3 py-2 rounded-md bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 text-xs text-yellow-800 dark:text-yellow-200">
            Long-running query detected. Consider adding a LIMIT clause to improve performance.
          </div>
        )}
      </div>
    </div>
  )
}
