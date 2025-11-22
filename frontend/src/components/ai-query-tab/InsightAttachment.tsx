import { Sparkles } from "lucide-react"
import { memo } from "react"

import type { AgentAttachment } from "@/store/ai-query-agent-store"

interface InsightAttachmentProps {
  attachment: AgentAttachment
}

export const InsightAttachment = memo(function InsightAttachment({ attachment }: InsightAttachmentProps) {
  if (!attachment.insight) {
    return null
  }

  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary mb-2">
        <Sparkles className="h-4 w-4" />
        Key Insights
      </div>
      <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
        {attachment.insight.highlights.map((insight, idx) => (
          <li key={idx}>{insight}</li>
        ))}
      </ul>
    </div>
  )
})
