import { Key, Link } from 'lucide-react'
import React from 'react'
import { Handle, NodeProps,Position } from 'reactflow'

import { cn } from '@/lib/utils'
import { ColumnConfig,TableConfig } from '@/types/schema-visualizer'

interface TableNodeData extends TableConfig {
  schemaColor?: string
  isHighlighted?: boolean
  isSelected?: boolean
  isFocused?: boolean
  isDimmed?: boolean
  detailLevel?: 'full' | 'compact'
  showPrimaryKeys?: boolean
}

function TableNodeComponent({ data, selected }: NodeProps<TableNodeData>) {
  const {
    name,
    schema,
    columns,
    schemaColor,
    isHighlighted,
    isSelected,
    isFocused,
    isDimmed,
    detailLevel = 'full',
    showPrimaryKeys = true,
  } = data

  const isCompact = detailLevel === 'compact'

  // Calculate header height (px-3 py-2 = 8px top + 8px bottom + text height ~20px = ~36px)
  const headerHeight = 36
  // Calculate padding top of columns section (p-2 = 8px)
  const columnsPaddingTop = 8
  // Height per column row (space-y-1 = 4px gap, text-xs = 16px height + padding = ~20px)
  const rowHeight = 20

  return (
    <div
      className={cn(
        'bg-background border-2 rounded-lg shadow-lg min-w-[200px] max-w-[300px] relative',
        'transition-all duration-200',
        selected || isSelected
          ? 'border-primary shadow-xl scale-105'
          : 'border-border hover:border-primary/50',
        isHighlighted && 'ring-2 ring-primary/30',
        isFocused && 'border-blue-500 border-3 shadow-2xl ring-4 ring-blue-500/30 scale-[1.02]',
        isDimmed && 'opacity-40'
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
      {!isCompact ? (
        <div className="p-2 space-y-1">
          {columns.map((column: ColumnConfig) => (
            <div
              key={column.id}
              className="flex items-center justify-between text-xs"
            >
              <div className="flex items-center space-x-1 flex-1 min-w-0">
                {/* Primary Key Indicator */}
                {showPrimaryKeys && column.isPrimaryKey && (
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
            </div>
          ))}
        </div>
      ) : (
        <div className="p-3 space-y-1 text-xs text-muted-foreground">
          <p className="font-medium text-foreground">{columns.length} columns</p>
          <p className="text-muted-foreground">
            Zoom in or switch to detailed mode to view column definitions.
          </p>
        </div>
      )}

      {/* Table Footer */}
      <div className="px-3 py-1 bg-muted/30 rounded-b-md text-xs text-muted-foreground">
        {columns.length} column{columns.length !== 1 ? 's' : ''}
      </div>

      {/* Handles positioned absolutely relative to node */}
      {!isCompact && columns.map((column: ColumnConfig, index: number) => {
        const topPosition = headerHeight + columnsPaddingTop + (index * rowHeight) + (rowHeight / 2)

        return (
          <React.Fragment key={`handles-${column.id}`}>
            <Handle
              type="source"
              position={Position.Right}
              id={`${column.id}-source`}
              className="!w-3 !h-3 !bg-blue-500 !border-2 !border-white hover:!bg-blue-600 transition-colors"
              style={{ top: topPosition }}
            />
            <Handle
              type="target"
              position={Position.Left}
              id={`${column.id}-target`}
              className="!w-3 !h-3 !bg-green-500 !border-2 !border-white hover:!bg-green-600 transition-colors"
              style={{ top: topPosition }}
            />
          </React.Fragment>
        )
      })}
    </div>
  )
}

export const TableNode = React.memo(TableNodeComponent)
