import { CheckCircle2, Inbox } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { toast } from '../../hooks/use-toast'
import { useQueryStore } from '../../store/query-store'
import type { EditableTableContext, TableRow } from '../../types/table'
import { AGGridTable } from '../ag-grid-table'
import { JsonRowViewerSidebarV2 } from '../json-row-viewer-sidebar-v2'
import { PaginationControls } from '../pagination-controls'
import { DeleteConfirmDialog, QueryResultsToolbar, StatusBar } from './components'
import {
  useDatabaseSelector,
  useJsonViewer,
  useTableEditing,
  useTableExport,
  useTablePagination,
  useTableSelection,
} from './hooks'
import type { QueryResultsTableProps } from './types'

export function QueryResultsTable({
  resultId,
  columns = [],
  rows,
  originalRows,
  metadata,
  query,
  connectionId,
  executionTimeMs,
  rowCount,
  executedAt,
  affectedRows,
  isLarge = false,
  chunkingEnabled = false,
  displayMode,
  totalRows,
  hasMore: _hasMore = false, // Reserved for future infinite scroll feature
  offset = 0,
  onPageChange,
}: QueryResultsTableProps) {
  const columnNames = useMemo(
    () => (Array.isArray(columns) ? columns : []),
    [columns]
  )

  const updateResultRows = useQueryStore((state) => state.updateResultRows)
  const tableContextRef = useRef<EditableTableContext | null>(null)


  // Pagination hook
  const pagination = useTablePagination({
    resultId,
    offset,
    onPageChange,
  })

  // Editing hook
  const editing = useTableEditing({
    resultId,
    rows,
    originalRows,
    columnNames,
    metadata,
    connectionId,
    query,
    tableContextRef,
    updateResultRows,
  })

  // JSON viewer hook
  const jsonViewer = useJsonViewer({ rows })

  // Database selector hook
  const databaseSelector = useDatabaseSelector({
    connectionId,
    resultId,
    tableContextRef,
    updateResultRows,
    setDirtyRowIds: editing.setDirtyRowIds,
    setPendingDeleteIds: (ids) => selection.setPendingDeleteIds(ids),
    clearJsonViewer: jsonViewer.clearSelection,
  })

  // Selection/deletion hook
  const selection = useTableSelection({
    connectionId,
    query,
    columnNames,
    columnsLookup: editing.columnsLookup,
    metadata,
    originalRows,
    resultId,
    tableContextRef,
    resolveCurrentRows: editing.resolveCurrentRows,
    updateResultRows,
    setDirtyRowIds: editing.setDirtyRowIds,
    onRowDeleted: (deletedRowId) => {
      if (jsonViewer.selectedRowId === deletedRowId) {
        jsonViewer.clearSelection()
      }
    },
  })

  // Export hook
  const tableExport = useTableExport({
    connectionId,
    query,
    columnNames,
    tableContextRef,
    resolveCurrentRows: editing.resolveCurrentRows,
  })

  // Reset state when result changes
  useEffect(() => {
    editing.setDirtyRowIds([])
    tableContextRef.current?.actions.clearDirtyRows?.()
    tableContextRef.current?.actions.resetTable?.()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Only reset on resultId change
  }, [resultId])

  // Memoize toolbar render function
  const renderToolbar = useCallback((context: EditableTableContext) => {
    tableContextRef.current = context

    return (
      <QueryResultsToolbar
        context={context}
        rowCount={rowCount}
        columnCount={columnNames.length}
        executionTimeMs={executionTimeMs}
        executedAt={executedAt}
        dirtyCount={editing.dirtyRowIds.length}
        canSave={editing.canSave}
        saving={editing.saving}
        onSave={editing.handleSave}
        onExport={tableExport.handleExport}
        metadata={metadata}
        hasEditableColumns={editing.hasEditableColumns}
        onDiscardChanges={editing.handleDiscardChanges}
        onJumpToFirstError={editing.handleJumpToFirstError}
        canDeleteRows={selection.canDeleteRows}
        onDeleteSelected={selection.canDeleteRows ? selection.handleRequestDelete : undefined}
        canInsertRows={editing.canInsertRows}
        onAddRow={editing.canInsertRows ? editing.handleAddRow : undefined}
        databases={databaseSelector.databaseSelectorEnabled ? databaseSelector.databaseList : undefined}
        currentDatabase={databaseSelector.activeDatabase}
        onSelectDatabase={databaseSelector.databaseSelectorEnabled ? databaseSelector.handleDatabaseSelection : undefined}
        databaseLoading={databaseSelector.databaseLoading}
        databaseSwitching={databaseSelector.isSwitchingDatabase}
      />
    )
  }, [
    rowCount, columnNames.length, executionTimeMs, executedAt,
    editing.dirtyRowIds.length, editing.canSave, editing.saving,
    editing.handleSave, tableExport.handleExport, metadata,
    editing.hasEditableColumns, editing.handleDiscardChanges,
    editing.handleJumpToFirstError, selection.canDeleteRows,
    selection.handleRequestDelete, editing.canInsertRows, editing.handleAddRow,
    databaseSelector.databaseSelectorEnabled, databaseSelector.databaseList,
    databaseSelector.activeDatabase, databaseSelector.handleDatabaseSelection,
    databaseSelector.databaseLoading, databaseSelector.isSwitchingDatabase,
  ])

  // Derived values (must be before useEffect that uses them)
  const safeAffectedRows = Number.isFinite(affectedRows) ? affectedRows : 0
  const hasTabularResults = columnNames.length > 0 && rows.length > 0
  const isModificationStatement = columnNames.length === 0
  const affectedRowsMessage =
    safeAffectedRows === 1
      ? '1 row affected.'
      : `${safeAffectedRows.toLocaleString()} rows affected.`
  const effectiveTotalRows = totalRows !== undefined ? totalRows : rowCount
  const showPagination = Boolean(onPageChange && effectiveTotalRows > 0 && hasTabularResults)

  const handleSelectAllPages = useCallback(() => {
    toast({
      title: 'All rows selected',
      description: `All ${totalRows?.toLocaleString()} rows across all pages are now selected. Bulk actions will apply to all rows.`,
      variant: 'default',
    })
  }, [totalRows])

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return
      }

      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
      const isSaveShortcut = isMac
        ? (event.metaKey && event.key === 's')
        : (event.ctrlKey && event.key === 's')

      if (isSaveShortcut) {
        event.preventDefault()
        if (metadata?.enabled && editing.dirtyRowIds.length > 0) {
          editing.handleSave()
        }
        return
      }

      // Pagination keyboard navigation
      if (showPagination && !pagination.isLoadingPage && !editing.saving) {
        const totalPages = Math.ceil(effectiveTotalRows / pagination.pageSize)

        if (event.altKey && (event.key === 'ArrowLeft' || event.key === 'PageUp')) {
          event.preventDefault()
          if (pagination.currentPage > 1) {
            pagination.handlePageChange(pagination.currentPage - 1)
          }
          return
        }

        if (event.altKey && (event.key === 'ArrowRight' || event.key === 'PageDown')) {
          event.preventDefault()
          if (pagination.currentPage < totalPages) {
            pagination.handlePageChange(pagination.currentPage + 1)
          }
          return
        }

        if (event.altKey && event.key === 'Home') {
          event.preventDefault()
          if (pagination.currentPage > 1) {
            pagination.handlePageChange(1)
          }
          return
        }

        if (event.altKey && event.key === 'End') {
          event.preventDefault()
          if (pagination.currentPage < totalPages) {
            pagination.handlePageChange(totalPages)
          }
          return
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- We include all used properties from editing/pagination
  }, [
    metadata?.enabled, editing.dirtyRowIds.length, editing.handleSave,
    pagination.isLoadingPage, editing.saving, pagination.pageSize,
    pagination.currentPage, pagination.handlePageChange, showPagination,
    effectiveTotalRows,
  ])

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      {!hasTabularResults ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            {isModificationStatement ? (
              <>
                <CheckCircle2 className="h-12 w-12 mx-auto mb-4 text-primary" />
                <p className="text-lg font-medium mb-1">Statement executed successfully</p>
                <p className="text-sm">{affectedRowsMessage}</p>
              </>
            ) : (
              <>
                <Inbox className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium mb-1">No results found</p>
                <p className="text-sm">Your query returned 0 rows</p>
              </>
            )}
          </div>
        </div>
      ) : (
        <>
          <AGGridTable
            data={rows as TableRow[]}
            columns={editing.tableColumns}
            onDirtyChange={editing.handleDirtyChange}
            enableMultiSelect={selection.canDeleteRows}
            enableGlobalFilter={false}
            enableExport={true}
            loading={editing.saving || pagination.isLoadingPage}
            className="flex-1 min-h-0"
            height="100%"
            onExport={tableExport.handleExport}
            onCellEdit={editing.handleCellEdit}
            onRowInspect={jsonViewer.handleRowClick}
            onSelectAllPages={effectiveTotalRows > rows.length ? handleSelectAllPages : undefined}
            toolbar={renderToolbar}
            footer={null}
            isEditable={editing.hasEditableColumns}
            resultId={resultId}
            totalRows={effectiveTotalRows}
            isLargeResult={isLarge}
            chunkingEnabled={chunkingEnabled}
            displayMode={displayMode}
          />

          {showPagination && (
            <div className="border-t border-border bg-muted/20 px-4 py-2">
              <PaginationControls
                currentPage={pagination.currentPage}
                pageSize={pagination.pageSize}
                totalRows={effectiveTotalRows}
                onPageChange={pagination.handlePageChange}
                onPageSizeChange={pagination.handlePageSizeChange}
                disabled={pagination.isLoadingPage || editing.saving}
                compact
              />
            </div>
          )}
        </>
      )}

      <DeleteConfirmDialog
        open={selection.showDeleteDialog}
        onOpenChange={(open) => {
          if (selection.isDeleting) return
          if (!open) {
            selection.setShowDeleteDialog(false)
            selection.setPendingDeleteIds([])
          }
        }}
        pendingDeleteCount={selection.pendingDeleteIds.length}
        isDeleting={selection.isDeleting}
        onConfirm={selection.handleConfirmDelete}
        onCancel={() => {
          selection.setShowDeleteDialog(false)
          selection.setPendingDeleteIds([])
        }}
      />

      <StatusBar
        totalRows={totalRows}
        rowCount={rowCount}
        columnCount={columnNames.length}
        affectedRows={affectedRows}
        executionTimeMs={executionTimeMs}
        dirtyRowCount={editing.dirtyRowIds.length}
      />

      <JsonRowViewerSidebarV2
        open={jsonViewer.jsonViewerOpen}
        onClose={jsonViewer.handleCloseJsonViewer}
        rowData={jsonViewer.selectedRowData}
        rowId={jsonViewer.selectedRowId}
        rowIndex={jsonViewer.selectedRowIndex}
        totalRows={rows.length}
        onNavigate={jsonViewer.handleNavigateRow}
        columns={columnNames}
        metadata={metadata}
        connectionId={connectionId}
        onSave={editing.handleJsonViewerSave}
      />
    </div>
  )
}
