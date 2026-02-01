import { CheckCircle2, Clock, Database } from 'lucide-react'
import React from 'react'

import type { StatusBarProps } from '../types'

export const StatusBar = React.memo(function StatusBar({
  totalRows,
  rowCount,
  columnCount,
  affectedRows,
  executionTimeMs,
  dirtyRowCount,
}: StatusBarProps) {
  const safeAffectedRows = Number.isFinite(affectedRows) ? affectedRows : 0

  return (
    <div className="flex-shrink-0 border-t border-border bg-muted/40 px-4 py-2 text-xs text-muted-foreground flex items-center justify-between">
      <div className="flex items-center gap-4">
        <span className="flex items-center gap-1.5">
          <Database className="h-3.5 w-3.5" />
          {(totalRows !== undefined ? totalRows : rowCount).toLocaleString()} rows
          {totalRows !== undefined && rowCount < totalRows && (
            <span className="text-muted-foreground/60"> ({rowCount.toLocaleString()} shown)</span>
          )}
          {' • '}{columnCount} columns
        </span>
        {safeAffectedRows > 0 && (
          <span className="flex items-center gap-1.5">
            <CheckCircle2 className="h-3.5 w-3.5" />
            {safeAffectedRows.toLocaleString()} affected
          </span>
        )}
        <span className="flex items-center gap-1.5">
          <Clock className="h-3.5 w-3.5" />
          {executionTimeMs.toFixed(2)} ms
        </span>
      </div>
      <span>
        {dirtyRowCount > 0
          ? `${dirtyRowCount} pending change${dirtyRowCount === 1 ? '' : 's'}`
          : 'No pending changes'}
      </span>
    </div>
  )
})
