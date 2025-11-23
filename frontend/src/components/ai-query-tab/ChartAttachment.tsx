import { Wand2 } from "lucide-react"
import { memo } from "react"

import type { AgentChartAttachment } from "@/store/ai-query-agent-store"

interface ChartAttachmentProps {
  chart: AgentChartAttachment
}

export const ChartAttachment = memo(function ChartAttachment({ chart }: ChartAttachmentProps) {
  return (
    <div className="rounded-xl border border-border/60 bg-background/70 p-3 shadow-sm text-xs text-muted-foreground space-y-1">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary">
        <Wand2 className="h-4 w-4" />
        Suggested Chart
      </div>
      <p><strong>Type:</strong> {chart.type}</p>
      <p><strong>X Axis:</strong> {chart.xField}</p>
      <p><strong>Y Axis:</strong> {chart.yFields.join(', ')}</p>
      {chart.seriesField && <p><strong>Series:</strong> {chart.seriesField}</p>}
      {chart.description && <p>{chart.description}</p>}
      <p className="italic">Use this as guidance when building a visualization.</p>
    </div>
  )
})
