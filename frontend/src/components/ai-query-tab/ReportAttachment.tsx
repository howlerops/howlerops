import { MessageSquare } from "lucide-react"
import { memo } from "react"

import type { AgentReportAttachment } from "@/store/ai-query-agent-store"

interface ReportAttachmentProps {
  report: AgentReportAttachment
}

export const ReportAttachment = memo(function ReportAttachment({ report }: ReportAttachmentProps) {
  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm text-sm text-muted-foreground">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary mb-2">
        <MessageSquare className="h-4 w-4" />
        Report Draft
      </div>
      <pre className="whitespace-pre-wrap font-sans">{report.body}</pre>
    </div>
  )
})
