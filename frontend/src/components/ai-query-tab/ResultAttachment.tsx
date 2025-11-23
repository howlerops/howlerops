import { Download, Table2 } from "lucide-react"
import { memo } from "react"

import { Button } from "@/components/ui/button"
import type { AgentResultAttachment } from "@/store/ai-query-agent-store"

interface ResultAttachmentProps {
  result: AgentResultAttachment
  onExport?: (columns: string[], rows: Record<string, unknown>[]) => void
}

export const ResultAttachment = memo(function ResultAttachment({
  result,
  onExport
}: ResultAttachmentProps) {
  return (
    <div className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm space-y-3">
      <div className="flex items-center justify-between text-sm font-medium">
        <span className="flex items-center gap-2 text-primary">
          <Table2 className="h-4 w-4" />
          Result Preview ({result.rowCount} rows)
        </span>
        {onExport && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onExport(result.columns, result.rows)}
          >
            <Download className="h-4 w-4 mr-2" />
            Export CSV
          </Button>
        )}
      </div>
      <div className="overflow-x-auto rounded-md border border-border/40">
        <table className="w-full text-xs">
          <thead className="bg-muted/60">
            <tr>
              {result.columns.map((column) => (
                <th key={column} className="px-2 py-1 text-left font-semibold text-muted-foreground">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {result.rows.slice(0, 20).map((row, rowIndex) => (
              <tr key={rowIndex} className="border-t border-border/30">
                {result.columns.map((column) => (
                  <td key={column} className="px-2 py-1 text-muted-foreground">
                    {String(row[column] ?? '')}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {result.rows.length > 20 && (
        <p className="text-xs text-muted-foreground">
          Showing the first 20 rows. Export for the full result set.
        </p>
      )}
    </div>
  )
})
