import { Wand2 } from "lucide-react"
import { memo } from "react"

import type { AgentAttachment } from "@/store/ai-query-agent-store"

interface ChartAttachmentProps {
  attachment: AgentAttachment
}

export const ChartAttachment = memo(function ChartAttachment({ attachment }: ChartAttachmentProps) {
  if (!attachment.chart) {
    return null
  }

  return (
    <div className="rounded-xl border border-border/60 bg-background/70 p-3 shadow-sm text-xs text-muted-foreground space-y-1">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary">
        <Wand2 className="h-4 w-4" />
        Suggested Chart
      </div>
      <p><strong>Type:</strong> {attachment.chart.type}</p>
      <p><strong>X Axis:</strong> {attachment.chart.xField}</p>
      <p><strong>Y Axis:</strong> {attachment.chart.yFields.join(', ')}</p>
      {attachment.chart.seriesField && <p><strong>Series:</strong> {attachment.chart.seriesField}</p>}
      {attachment.chart.description && <p>{attachment.chart.description}</p>}
      <p className="italic">Use this as guidance when building a visualization.</p>
    </div>
  )
})
