import { Copy, Play, Sparkles } from "lucide-react"
import { memo } from "react"

import { ConfidenceIndicator } from "@/components/ConfidenceIndicator"
import { Button } from "@/components/ui/button"
import type { AgentSQLAttachment } from "@/store/ai-query-agent-store"

interface SQLAttachmentProps {
  sql: AgentSQLAttachment
  onCopy?: (sql: string) => void
  onUse?: (sql: string, connectionId?: string) => void
}

export const SQLAttachment = memo(function SQLAttachment({
  sql,
  onCopy,
  onUse
}: SQLAttachmentProps) {
  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm space-y-3">
      <div className="flex items-center justify-between text-sm font-medium">
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-2 text-primary">
            <Sparkles className="h-4 w-4" />
            Generated SQL
          </span>
          {sql.confidence !== undefined && (
            <ConfidenceIndicator confidence={sql.confidence} size="sm" />
          )}
        </div>
        <div className="flex items-center gap-2">
          {onCopy && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onCopy(sql.query)}
              title="Copy SQL"
            >
              <Copy className="h-4 w-4" />
            </Button>
          )}
          {onUse && (
            <Button
              size="sm"
              onClick={() => onUse(sql.query, sql.connectionId)}
            >
              <Play className="h-4 w-4 mr-2" />
              Use in Editor
            </Button>
          )}
        </div>
      </div>
      <pre className="rounded-md bg-muted/60 p-3 text-xs font-mono whitespace-pre-wrap border border-border/40">
        {sql.query}
      </pre>
      {sql.explanation && (
        <p className="text-xs text-muted-foreground">
          {sql.explanation}
        </p>
      )}
    </div>
  )
})
