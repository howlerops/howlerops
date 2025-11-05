import { NodeProps } from 'reactflow'
import { FolderOpen, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface SchemaSummaryNodeData {
  schema: string
  tableCount: number
  color?: string
  onExpand: (schema: string) => void
}

export function SchemaSummaryNode({ data }: NodeProps<SchemaSummaryNodeData>) {
  const { schema, tableCount, color, onExpand } = data

  return (
    <div
      className={cn(
        'min-w-[180px] max-w-[220px] rounded-lg border-2 border-dashed border-border bg-background/90 shadow-sm',
        'flex flex-col overflow-hidden'
      )}
    >
      <div
        className="px-3 py-2 text-white text-sm font-semibold flex items-center gap-2"
        style={{ backgroundColor: color || '#6b7280' }}
      >
        <FolderOpen className="h-4 w-4" />
        <span className="truncate">{schema}</span>
      </div>
      <div className="p-3 space-y-2 text-sm">
        <p className="text-foreground font-medium">
          {tableCount} table{tableCount !== 1 ? 's' : ''}
        </p>
        <p className="text-muted-foreground text-xs leading-snug">
          Collapsed for clarity. Expand to inspect individual tables and relationships.
        </p>
        <button
          type="button"
          className="w-full inline-flex items-center justify-center rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-muted transition-colors"
          onClick={() => onExpand(schema)}
        >
          <Plus className="h-3 w-3 mr-1" />
          Expand Schema
        </button>
      </div>
    </div>
  )
}
