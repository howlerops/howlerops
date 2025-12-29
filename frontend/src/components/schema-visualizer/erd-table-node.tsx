import { Grid3X3, KeyRound } from 'lucide-react'
import React, { useMemo } from 'react'
import { Handle, NodeProps, Position } from 'reactflow'

import { cn } from '@/lib/utils'
import { ColumnConfig, TableConfig } from '@/types/schema-visualizer'

interface ERDTableNodeData extends TableConfig {
  isSelected?: boolean
  isFocused?: boolean
  isDimmed?: boolean
  isRelated?: boolean
}

/**
 * ERD Table Node - Modern ERD design with light/dark theme support
 * - Themed background with dashed borders for related tables
 * - Solid highlighted border for selected/focused table
 * - Grid icon in header
 * - Key icons for PK/FK columns
 * - Right-aligned data types
 */
function ERDTableNodeComponent({ data, selected }: NodeProps<ERDTableNodeData>) {
  const {
    name,
    columns,
    isSelected,
    isFocused,
    isDimmed,
    isRelated,
  } = data

  // Determine border style based on state
  const borderStyle = useMemo(() => {
    if (isFocused || isSelected || selected) {
      return 'border-solid border-yellow-500 dark:border-yellow-500'
    }
    if (isRelated) {
      return 'border-dashed border-border'
    }
    return 'border-dashed border-border'
  }, [isFocused, isSelected, selected, isRelated])

  return (
    <div
      className={cn(
        'bg-card border-2 rounded-lg min-w-[220px] max-w-[280px] relative shadow-lg',
        'transition-[opacity,border-color] duration-100',
        borderStyle,
        isDimmed && 'opacity-30'
      )}
    >
      {/* Table Header */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border bg-muted/50">
        <Grid3X3 className="h-4 w-4 text-muted-foreground flex-shrink-0" />
        <span className="font-medium text-sm text-foreground truncate">{name}</span>
      </div>

      {/* Columns */}
      <div className="py-1">
        {columns.map((column: ColumnConfig) => (
          <ERDColumnRow
            key={column.id}
            column={column}
          />
        ))}
      </div>

      {/* Fallback table-level handles for edges without column info */}
      <Handle
        type="target"
        position={Position.Left}
        id="table-target"
        className="!w-2 !h-2 !bg-muted-foreground !border-0 !-left-1 !top-1/2"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="table-source"
        className="!w-2 !h-2 !bg-muted-foreground !border-0 !-right-1 !top-1/2"
      />
    </div>
  )
}

interface ERDColumnRowProps {
  column: ColumnConfig
}

/**
 * Individual column row in ERD table
 * Handles are positioned relative to the row itself, ensuring perfect alignment
 */
function ERDColumnRow({ column }: ERDColumnRowProps) {
  const isPK = column.isPrimaryKey
  const isFK = column.isForeignKey

  return (
    <div
      className={cn(
        'flex items-center justify-between px-3 py-1 text-xs relative',
        'hover:bg-muted/50 transition-colors'
      )}
    >
      {/* Column name with key icons */}
      <div className="flex items-center gap-2 flex-1 min-w-0">
        {/* Key indicators */}
        {(isPK || isFK) && (
          <div className="flex items-center gap-0.5 flex-shrink-0">
            {isPK && (
              <KeyRound className="h-3 w-3 text-amber-500 dark:text-amber-400" />
            )}
            {isFK && (
              <KeyRound className="h-3 w-3 text-amber-600 dark:text-amber-500 rotate-45" />
            )}
          </div>
        )}

        {/* Column name */}
        <span
          className={cn(
            'truncate',
            (isPK || isFK) ? 'text-foreground font-medium' : 'text-foreground/80'
          )}
        >
          {column.name}
        </span>
      </div>

      {/* Data type */}
      <span className="text-muted-foreground ml-3 flex-shrink-0 uppercase text-[10px]">
        {formatDataType(column.type)}
      </span>

      {/* Connection handles - positioned relative to this row (50% vertical center) */}
      <Handle
        type="target"
        position={Position.Left}
        id={`${column.id}-target`}
        className="!w-1.5 !h-1.5 !bg-muted-foreground/60 !border-0 !-left-1 !top-1/2 !-translate-y-1/2"
      />
      <Handle
        type="source"
        position={Position.Right}
        id={`${column.id}-source`}
        className="!w-1.5 !h-1.5 !bg-muted-foreground/60 !border-0 !-right-1 !top-1/2 !-translate-y-1/2"
      />
    </div>
  )
}

/**
 * Format data type for display - uppercase and simplified
 */
function formatDataType(type: string): string {
  if (!type) return 'UNKNOWN'

  // Extract base type (remove size specifications)
  const baseType = type
    .toUpperCase()
    .replace(/\(.*\)/, '')
    .replace(/\s+/g, '')
    .trim()

  // Map common types to display names
  const typeMap: Record<string, string> = {
    'INT': 'INTEGER',
    'INT4': 'INTEGER',
    'INT8': 'BIGINT',
    'BIGINT': 'BIGINT',
    'SMALLINT': 'SMALLINT',
    'INTEGER': 'INTEGER',
    'SERIAL': 'INTEGER',
    'BIGSERIAL': 'BIGINT',
    'VARCHAR': 'TEXT',
    'CHARACTER': 'TEXT',
    'CHARACTERVARYING': 'TEXT',
    'TEXT': 'TEXT',
    'BOOLEAN': 'BOOLEAN',
    'BOOL': 'BOOLEAN',
    'TIMESTAMP': 'TIMESTAMP',
    'TIMESTAMPTZ': 'TIMESTAMP',
    'TIMESTAMPWITHTIMEZONE': 'TIMESTAMP',
    'DATE': 'DATE',
    'TIME': 'TIME',
    'TIMETZ': 'TIME',
    'JSON': 'JSON',
    'JSONB': 'JSONB',
    'UUID': 'UUID',
    'NUMERIC': 'NUMERIC',
    'DECIMAL': 'DECIMAL',
    'REAL': 'REAL',
    'FLOAT': 'FLOAT',
    'FLOAT4': 'REAL',
    'FLOAT8': 'DOUBLE',
    'DOUBLE': 'DOUBLE',
    'DOUBLEPRECISION': 'DOUBLE',
    'BYTEA': 'BLOB',
    'BLOB': 'BLOB',
  }

  return typeMap[baseType] || baseType
}

export const ERDTableNode = React.memo(ERDTableNodeComponent)
