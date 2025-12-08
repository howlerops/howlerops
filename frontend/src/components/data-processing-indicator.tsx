import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'

import { Progress } from '@/components/ui/progress'

interface DataProcessingIndicatorProps {
  rowCount: number
  progress?: number // 0-100
  message?: string
  startTime?: number // Timestamp when processing started
}

export const DataProcessingIndicator = ({
  rowCount,
  progress = 0,
  message,
  startTime
}: DataProcessingIndicatorProps) => {
  const [elapsedSeconds, setElapsedSeconds] = useState(0)

  useEffect(() => {
    if (!startTime) {
      setElapsedSeconds(0)
      return
    }

    const interval = setInterval(() => {
      setElapsedSeconds(Math.floor((Date.now() - startTime) / 1000))
    }, 1000)

    return () => clearInterval(interval)
  }, [startTime])

  const formatRowCount = (count: number): string => {
    if (count < 1000) return count.toString()
    if (count < 1000000) return `${(count / 1000).toFixed(1)}K`
    return `${(count / 1000000).toFixed(1)}M`
  }

  // Determine the processing stage based on progress
  const getStageMessage = (): string => {
    if (message) return message
    return `Processing ${formatRowCount(rowCount)} rows...`
  }

  return (
    <div className="flex items-center justify-center h-full w-full p-8">
      <div className="flex flex-col items-center gap-4 max-w-md w-full">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />

        <div className="text-center w-full">
          <h3 className="text-lg font-semibold mb-1">Processing Data</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {getStageMessage()}
            {startTime && elapsedSeconds >= 2 && (
              <span className="ml-2 text-xs opacity-70">
                ({elapsedSeconds}s)
              </span>
            )}
          </p>

          {/* Bouncing dots indicator */}
          <div className="flex items-center justify-center gap-1 mb-4">
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

          <div className="space-y-2">
            <Progress value={progress} className="h-2" />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>{Math.round(progress)}% complete</span>
              <span>{formatRowCount(Math.round((rowCount * progress) / 100))} / {formatRowCount(rowCount)}</span>
            </div>
          </div>
        </div>

        {/* Slow processing warning */}
        {startTime && elapsedSeconds >= 15 && (
          <div className="mt-2 text-xs text-amber-500 animate-pulse">
            Taking longer than usual...
          </div>
        )}

        <div className="mt-2 text-xs text-muted-foreground text-center">
          Large datasets are processed in batches to keep the UI responsive.
          <br />
          This should only take a few seconds.
        </div>
      </div>
    </div>
  )
}
