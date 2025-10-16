import React from 'react'
import { Handle, Position, NodeProps } from 'reactflow'
import { Key, Link } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TableConfig, ColumnConfig } from '@/types/schema-visualizer'

interface TableNodeData extends TableConfig {
  schemaColor?: string
  isHighlighted?: boolean
  isSelected?: boolean
}

export function TableNode({ data, selected }: NodeProps<TableNodeData>) {
  const { name, schema, columns, schemaColor, isHighlighted, isSelected } = data

  return (
    <div
      className={cn(
        'bg-background border-2 rounded-lg shadow-lg min-w-[200px] max-w-[300px]',
        'transition-all duration-200',
        selected || isSelected
          ? 'border-primary shadow-xl scale-105'
          : 'border-border hover:border-primary/50',
        isHighlighted && 'ring-2 ring-primary/30'
      )}
    >
      {/* Table Header */}
      <div
        className="px-3 py-2 rounded-t-md font-semibold text-sm text-white"
        style={{ backgroundColor: schemaColor || '#6366f1' }}
      >
        <div className="flex items-center justify-between">
          <span className="truncate">{name}</span>
          <span className="text-xs opacity-75 ml-2">{schema}</span>
        </div>
      </div>

      {/* Columns */}
      <div className="p-2 space-y-1">
        {columns.map((column: ColumnConfig, index: number) => (
          <div
            key={column.id}
            className="flex items-center justify-between text-xs group"
          >
            <div className="flex items-center space-x-1 flex-1 min-w-0">
              {/* Primary Key Indicator */}
              {column.isPrimaryKey && (
                <Key className="h-3 w-3 text-accent-foreground flex-shrink-0" />
              )}
              
              {/* Foreign Key Indicator */}
              {column.isForeignKey && (
                <Link className="h-3 w-3 text-primary flex-shrink-0" />
              )}

              {/* Column Name */}
              <span className="font-medium truncate">{column.name}</span>
            </div>

            {/* Column Type */}
            <span className="text-muted-foreground text-xs ml-2 flex-shrink-0">
              {column.type}
            </span>

            {/* Handles for connections */}
            <Handle
              type="source"
              position={Position.Right}
              id={`${column.id}-source`}
              className="opacity-0 group-hover:opacity-100 transition-opacity"
              style={{ top: `${20 + index * 24}px` }}
            />
            <Handle
              type="target"
              position={Position.Left}
              id={`${column.id}-target`}
              className="opacity-0 group-hover:opacity-100 transition-opacity"
              style={{ top: `${20 + index * 24}px` }}
            />
          </div>
        ))}
      </div>

      {/* Table Footer */}
      <div className="px-3 py-1 bg-muted/30 rounded-b-md text-xs text-muted-foreground">
        {columns.length} column{columns.length !== 1 ? 's' : ''}
      </div>
    </div>
  )
}

