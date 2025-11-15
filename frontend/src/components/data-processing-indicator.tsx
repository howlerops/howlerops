import { Loader2 } from 'lucide-react'
import { Progress } from '@/components/ui/progress'

interface DataProcessingIndicatorProps {
  rowCount: number
  progress?: number // 0-100
  message?: string
}

export const DataProcessingIndicator = ({
  rowCount,
  progress = 0,
  message
}: DataProcessingIndicatorProps) => {
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
          <p className="text-sm text-muted-foreground mb-4">
            {getStageMessage()}
          </p>

          <div className="space-y-2">
            <Progress value={progress} className="h-2" />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>{Math.round(progress)}% complete</span>
              <span>{formatRowCount(Math.round((rowCount * progress) / 100))} / {formatRowCount(rowCount)}</span>
            </div>
          </div>
        </div>

        <div className="mt-2 text-xs text-muted-foreground text-center">
          Large datasets are processed in batches to keep the UI responsive.
          <br />
          This should only take a few seconds.
        </div>
      </div>
    </div>
  )
}
