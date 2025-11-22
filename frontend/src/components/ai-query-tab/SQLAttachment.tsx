import { Copy, Play, Sparkles } from "lucide-react"
import { memo } from "react"

import { ConfidenceIndicator } from "@/components/ConfidenceIndicator"
import { Button } from "@/components/ui/button"
import type { AgentAttachment } from "@/store/ai-query-agent-store"

interface SQLAttachmentProps {
  attachment: AgentAttachment
  onCopySQL: (sql: string) => void
  onUseSQL: (sql: string, connectionId?: string) => void
}

export const SQLAttachment = memo(function SQLAttachment({
  attachment,
  onCopySQL,
  onUseSQL
}: SQLAttachmentProps) {
  if (!attachment.sql) {
    return null
  }

  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm space-y-3">
      <div className="flex items-center justify-between text-sm font-medium">
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-2 text-primary">
            <Sparkles className="h-4 w-4" />
            Generated SQL
          </span>
          {attachment.sql.confidence !== undefined && (
            <ConfidenceIndicator confidence={attachment.sql.confidence} size="sm" />
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onCopySQL(attachment.sql!.query)}
            title="Copy SQL"
          >
            <Copy className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            onClick={() => onUseSQL(attachment.sql!.query, attachment.sql?.connectionId)}
          >
            <Play className="h-4 w-4 mr-2" />
            Use in Editor
          </Button>
        </div>
      </div>
      <pre className="rounded-md bg-muted/60 p-3 text-xs font-mono whitespace-pre-wrap border border-border/40">
        {attachment.sql.query}
      </pre>
      {attachment.sql.explanation && (
        <p className="text-xs text-muted-foreground">
          {attachment.sql.explanation}
        </p>
      )}
    </div>
  )
})
