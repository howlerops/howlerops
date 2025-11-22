import { MessageSquare } from "lucide-react"
import { memo } from "react"

import type { AgentAttachment } from "@/store/ai-query-agent-store"

interface ReportAttachmentProps {
  attachment: AgentAttachment
}

export const ReportAttachment = memo(function ReportAttachment({ attachment }: ReportAttachmentProps) {
  if (!attachment.report) {
    return null
  }

  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm text-sm text-muted-foreground">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary mb-2">
        <MessageSquare className="h-4 w-4" />
        Report Draft
      </div>
      <pre className="whitespace-pre-wrap font-sans">{attachment.report.body}</pre>
    </div>
  )
})
