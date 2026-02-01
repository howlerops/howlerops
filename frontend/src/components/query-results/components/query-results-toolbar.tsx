import { AlertCircle, Plus, Save, Trash2 } from 'lucide-react'

import { Button } from '../../ui/button'
import type { ToolbarProps } from '../types'
import { formatTimestamp } from '../utils'
import { ExportButton } from './export-button'

export function QueryResultsToolbar({
  context,
  rowCount: _rowCount,
  columnCount: _columnCount,
  executionTimeMs: _executionTimeMs,
  executedAt,
  dirtyCount,
  canSave,
  saving,
  onSave,
  onExport,
  metadata,
  hasEditableColumns,
  onDiscardChanges,
  onJumpToFirstError,
  canDeleteRows,
  onDeleteSelected,
  canInsertRows,
  onAddRow,
  databases: _databases, // Database selector currently disabled
  currentDatabase: _currentDatabase,
  onSelectDatabase: _onSelectDatabase,
  databaseLoading: _databaseLoading,
  databaseSwitching: _databaseSwitching,
}: ToolbarProps) {
  const invalidCellsCount = context.state.invalidCells.size
  const dirtyRowCount = dirtyCount ?? context.state.dirtyRows.size
  const hasValidationErrors = invalidCellsCount > 0
  const canSaveWithValidation = canSave && !hasValidationErrors
  const selectedCount = context.state.selectedRows.length

  return (
    <div className="flex flex-col gap-2 border-b border-gray-200 bg-background px-1 py-1">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-1 items-center gap-3 min-w-[220px]">
          {/* Database selector and search disabled for now */}
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground">{formatTimestamp(executedAt)}</span>

          {/* Unsaved changes indicator */}
          {dirtyRowCount > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 bg-accent/10 border border-accent rounded text-xs">
              <span className="text-accent-foreground">
                {dirtyRowCount} unsaved{dirtyRowCount === 1 ? '' : ''}
              </span>
            </div>
          )}

          {/* Validation errors indicator */}
          {invalidCellsCount > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 bg-destructive/10 border border-destructive rounded text-xs">
              <AlertCircle className="h-3 w-3 text-destructive" />
              <span className="text-destructive">
                {invalidCellsCount} error{invalidCellsCount === 1 ? '' : 's'}
              </span>
              {onJumpToFirstError && (
                <button
                  onClick={onJumpToFirstError}
                  className="text-destructive hover:text-destructive/80 underline ml-1"
                >
                  Jump
                </button>
              )}
            </div>
          )}

          {/* Add row */}
          {canInsertRows && onAddRow && (
            <Button
              variant="outline"
              size="sm"
              onClick={onAddRow}
              disabled={metadata?.pending || !metadata?.enabled}
              className="gap-2"
              title={metadata?.pending ? 'Checking table editability...' : undefined}
            >
              <Plus className="h-4 w-4" />
              Add Row
            </Button>
          )}

          {/* Delete selected rows */}
          {canDeleteRows && selectedCount > 0 && onDeleteSelected && (
            <Button
              variant="destructive"
              size="icon"
              onClick={onDeleteSelected}
              disabled={metadata?.pending || !metadata?.enabled}
              className="h-9 w-9"
              title={
                metadata?.pending
                  ? 'Checking table editability...'
                  : `Delete ${selectedCount} selected row${selectedCount === 1 ? '' : 's'}`
              }
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}

          {/* Export Button */}
          <ExportButton context={context} onExport={onExport} />

          {/* Show save controls when table has editable columns and metadata is available */}
          {metadata && hasEditableColumns && (
            <>
              {/* Discard Changes Button */}
              {dirtyRowCount > 0 && onDiscardChanges && metadata.enabled && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onDiscardChanges}
                  disabled={saving}
                  className="gap-2"
                >
                  Discard Changes
                </Button>
              )}

              {/* Save Button - always shown when metadata exists, disabled while pending */}
              <Button
                size="sm"
                onClick={onSave}
                disabled={!canSaveWithValidation || metadata.pending || !metadata.enabled}
                className="gap-2"
                title={
                  metadata.pending
                    ? 'Checking table editability...'
                    : !metadata.enabled
                      ? 'Table is not editable'
                      : hasValidationErrors
                        ? `Cannot save: ${invalidCellsCount} validation errors`
                        : undefined
                }
              >
                {saving ? (
                  <span className="flex items-center gap-2">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-b-transparent border-current" />
                    Saving...
                  </span>
                ) : metadata.pending ? (
                  <span className="flex items-center gap-2">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-b-transparent border-current" />
                    Checking...
                  </span>
                ) : (
                  <>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </>
                )}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
