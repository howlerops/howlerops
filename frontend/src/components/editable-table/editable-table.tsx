import { useVirtualizer, type VirtualItem } from '@tanstack/react-virtual';
import React, { useCallback, useEffect, useMemo,useRef, useState } from 'react';

import { useChunkedData } from '../../hooks/use-chunked-data';
import { useKeyboardNavigation } from '../../hooks/use-keyboard-navigation';
import { useTableState } from '../../hooks/use-table-state';
import {
  CellValue,
  EditableTableContext,
  EditableTableProps,
  EditableTableRenderer,
  TableColumn,
  TableRow,
} from '../../types/table';
import { cn } from '../../utils/cn';
import { TableBodyStatic } from './components/body/table-body-static';
import { TableBodyVirtualized } from './components/body/table-body-virtualized';
import { SelectionBanner } from './components/selection/selection-banner';
import { EditStateContext } from './context/edit-state-context';
import { SelectionStateContext } from './context/selection-state-context';
import { TableConfigContext } from './context/table-config-context';
import { TableContext } from './context/table-context';
import { useTableColumns } from './hooks/use-table-columns';
import { useTableInstance } from './hooks/use-table-instance';
import { StatusBar } from './status-bar';
import { TableHeader } from './table-header';
import { TableToolbar } from './table-toolbar';

const EMPTY_VIRTUAL_ITEMS: VirtualItem[] = [];

// Debug logging only in development
const DEBUG = import.meta.env.MODE === 'development';
const debug = DEBUG ? console.log.bind(console) : () => {};

export const EditableTable: React.FC<EditableTableProps> = ({
  data: initialData,
  columns: tableColumns,
  onDataChange,
  onCellEdit: _onCellEdit,
  onRowSelect,
  onRowClick,
  onRowInspect,
  onSort,
  onFilter,
  onExport,
  onSelectAllPages,
  loading = false,
  error = null,
  virtualScrolling = true,
  estimateSize = 31,
  className,
  height = 600,
  enableMultiSelect = true,
  enableColumnResizing = true,
  enableGlobalFilter = true,
  enableExport = true,
  toolbar,
  footer,
  onDirtyChange,
  customCellRenderers = {},
  // Phase 2: Chunked data loading
  resultId,
  totalRows,
  isLargeResult = false,
  chunkingEnabled = false,
  displayMode,
}) => {
  const tableContainerRef = useRef<HTMLDivElement>(null);

  // Phase 2: Load chunked data on-demand from IndexedDB
  const {
    data: chunkedData,
    ensureRangeLoaded,
  } = useChunkedData({
    resultId: resultId || '',
    totalRows: totalRows || initialData.length,
    isLarge: chunkingEnabled && isLargeResult,
    initialData,
  });

  // Use chunked data if chunking is enabled, otherwise use initial data
  // REMOVED useDeferredValue - it causes stale data and scroll jumping
  const effectiveData = (chunkingEnabled && isLargeResult) ? chunkedData as TableRow[] : initialData;

  const {
    data,
    setData,
    state,
    actions,
    editStateValue,
    selectionStateValue,
  } = useTableState(effectiveData);

  useEffect(() => {
    // Update data synchronously when effective data changes
    setData(effectiveData);
  }, [effectiveData, setData]);

  // Create TanStack Table columns using extracted hook
  const columns = useTableColumns({
    tableColumns,
    enableMultiSelect,
    enableColumnResizing,
  });

  // TanStack Table's row selection state - this is the SINGLE source of truth
  const [internalRowSelection, setInternalRowSelection] = useState<Record<string, boolean>>({});

  // Sync internal selection to external state (for callbacks like onRowSelect)
  useEffect(() => {
    const selectedIds = Object.keys(internalRowSelection)
      .filter(key => internalRowSelection[key] === true)
      .map((indexStr) => {
        const index = parseInt(indexStr, 10);
        return data[index]?.__rowId;
      })
      .filter((id): id is string => Boolean(id));

    debug('[Effect] Syncing internal selection to external state:', selectedIds);
    actions.setSelectedRows(selectedIds);
  }, [internalRowSelection, data, actions]);

  // Create TanStack Table instance using extracted hook
  const table = useTableInstance({
    data,
    columns,
    state,
    actions,
    displayMode: displayMode?.canSort !== false ? 'view' : 'edit',
    enableMultiSelect,
    enableColumnResizing,
    enableGlobalFilter,
    internalRowSelection,
    setInternalRowSelection,
  });

  const { rows } = table.getRowModel();

  const rowCount = data.length;
  const visibleRowCount = rows.length;
  const shouldVirtualize = virtualScrolling && visibleRowCount > 0;

  const virtualizer = useVirtualizer({
    count: shouldVirtualize ? visibleRowCount : 0,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => estimateSize,
    // Reduced overscan: 5 rows is sufficient for smooth scrolling
    // 20 was too aggressive and caused unnecessary renders
    overscan: 5,
    measureElement: (element) => element?.getBoundingClientRect().height ?? estimateSize,
    // Enable horizontal overscan for wide tables
    horizontal: false,
  });

  // Preserve scroll position during data updates
  const scrollPositionRef = useRef({ top: 0, left: 0 });

  useEffect(() => {
    const scrollElement = tableContainerRef.current;
    if (!scrollElement) return;

    // Save scroll position before data updates
    const saveScrollPosition = () => {
      scrollPositionRef.current = {
        top: scrollElement.scrollTop,
        left: scrollElement.scrollLeft,
      };
    };

    // Restore scroll position after data updates
    const restoreScrollPosition = () => {
      if (scrollPositionRef.current.top > 0) {
        scrollElement.scrollTop = scrollPositionRef.current.top;
        scrollElement.scrollLeft = scrollPositionRef.current.left;
      }
    };

    saveScrollPosition();
    restoreScrollPosition();
  }, [data]);

  const virtualizerWorking = shouldVirtualize && Boolean(tableContainerRef.current);

  // Optimized keyboard navigation with virtualization support
  const {
    containerRef,
  } = useKeyboardNavigation({
    rowCount: rowCount,
    columnCount: columns.length,
    onCellFocus: useCallback((rowIndex: number, _columnIndex: number) => {
      // Smooth scroll to focused cell with virtualization
      if (virtualizerWorking && rowIndex >= 0 && rowIndex < rowCount) {
        virtualizer.scrollToIndex(rowIndex, {
          align: 'center',
          behavior: 'smooth',
        });
      }
    }, [virtualizerWorking, rowCount, virtualizer]),
    onCellEdit: useCallback((rowIndex: number, columnIndex: number) => {
      const row = rows[rowIndex];
      const column = columns[columnIndex];
      const columnId = (column?.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } })?.columnDef?.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } })?.columnDef?.accessorKey) as string | undefined;
      if (row && column && columnId && columnId !== 'select') {
        const rowData = row as { getValue: (columnId: string) => unknown; original: { __rowId: string } };
        const value = rowData.getValue(columnId) as CellValue;
        actions.startEditing(rowData.original.__rowId, columnId, value);
      }
    }, [rows, columns, actions]),
    onDelete: useCallback((rowIndex: number, columnIndex: number) => {
      const row = rows[rowIndex];
      const column = columns[columnIndex];
      const columnId = (column?.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } })?.columnDef?.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } })?.columnDef?.accessorKey) as string | undefined;
      if (!row || !column || !columnId || columnId === 'select') {
        return;
      }

      const metaColumn = (column as unknown as { columnDef?: { meta?: { originalColumn?: TableColumn } } })?.columnDef?.meta?.originalColumn;
      if (metaColumn && metaColumn.editable === false) {
        return;
      }

      const rowData = row as { original: { __rowId: string } };
      if (!rowData.original.__rowId) {
        return;
      }

      actions.updateCell(rowData.original.__rowId, columnId, null);
    }, [rows, columns, actions]),
    onUndo: actions.undo,
    onRedo: actions.redo,
    disabled: loading,
  });

  // Handle data changes
  useEffect(() => {
    onDataChange?.(data);
  }, [data, onDataChange]);

  // Handle row selection changes
  useEffect(() => {
    onRowSelect?.(state.selectedRows);
  }, [state.selectedRows, onRowSelect]);

  // Handle sorting changes
  useEffect(() => {
    onSort?.(state.sorting);
  }, [state.sorting, onSort]);

  // Handle filter changes
  useEffect(() => {
    onFilter?.(state.columnFilters);
  }, [state.columnFilters, onFilter]);

  useEffect(() => {
    if (onDirtyChange) {
      onDirtyChange(Array.from(state.dirtyRows));
    }
  }, [state.dirtyRows, onDirtyChange]);


  const tableContext = useMemo<EditableTableContext>(() => ({
    data,
    state,
    actions,
  }), [data, state, actions]);

  // Create stable TableConfig value
  const tableConfigValue = useMemo(() => ({
    onRowInspect,
    customCellRenderers,
  }), [onRowInspect, customCellRenderers]);

  const renderedToolbar = typeof toolbar === 'function'
    ? (toolbar as EditableTableRenderer)(tableContext)
    : toolbar;

  const renderedFooter = typeof footer === 'function'
    ? (footer as EditableTableRenderer)(tableContext)
    : footer;

  const shouldShowDefaultToolbar = !renderedToolbar && (enableGlobalFilter || enableExport);
  const shouldRenderToolbar = Boolean(renderedToolbar || shouldShowDefaultToolbar);

  // Determine if we should show the "Select All Pages" banner
  const allVisibleRowsSelected = rows.length > 0 && table.getIsAllRowsSelected();
  const hasPaginatedData = totalRows && totalRows > data.length;
  const shouldShowSelectAllBanner = allVisibleRowsSelected && hasPaginatedData && !state.selectAllPagesMode && enableMultiSelect;

  const handleSelectAllPages = useCallback(() => {
    actions.setSelectAllPagesMode(true);
    onSelectAllPages?.();
  }, [actions, onSelectAllPages]);

  const handleClearSelection = useCallback(() => {
    actions.selectAllRows(false);
    actions.setSelectAllPagesMode(false);
  }, [actions]);

  // Get virtual items following official pattern
  const virtualItems = virtualizerWorking ? virtualizer.getVirtualItems() : EMPTY_VIRTUAL_ITEMS;
  const totalSize = virtualizerWorking ? virtualizer.getTotalSize() : undefined;

  // Cleanup virtualizer when switching modes or unmounting
  useEffect(() => {
    return () => {
      // No explicit cleanup needed - virtualizer handles its own lifecycle
    };
  }, [shouldVirtualize]);

  useEffect(() => {
    if (!chunkingEnabled || !isLargeResult) {
      return;
    }
    if (!virtualizerWorking) {
      return;
    }
    if (virtualItems === EMPTY_VIRTUAL_ITEMS || virtualItems.length === 0) {
      return;
    }

    const startIndex = virtualItems[0].index;
    const endIndex = virtualItems[virtualItems.length - 1].index;
    ensureRangeLoaded(startIndex, endIndex);
  }, [chunkingEnabled, isLargeResult, virtualItems, virtualizerWorking, ensureRangeLoaded]);

  if (error) {
    return (
      <div className="flex items-center justify-center h-64 text-destructive">
        <div className="text-center">
          <div className="text-lg font-semibold">Error loading table</div>
          <div className="text-sm">{error}</div>
        </div>
      </div>
    );
  }

  return (
    <TableContext.Provider value={{
      state,
      actions,
      onRowInspect,
      customCellRenderers,
    }}>
      <TableConfigContext.Provider value={tableConfigValue}>
        <EditStateContext.Provider value={editStateValue}>
          <SelectionStateContext.Provider value={selectionStateValue}>
            <div className={cn('flex flex-col h-full min-h-0', className)}>
        {/* Toolbar */}
        {shouldRenderToolbar && (
        <div className="flex-shrink-0 border-b border-border">
          {renderedToolbar ?? (
            <TableToolbar
              searchValue={state.globalFilter}
              onSearchChange={actions.updateGlobalFilter}
              onExport={onExport}
              selectedCount={state.selectedRows.length}
              totalCount={data.length}
              loading={loading}
              showExport={enableExport}
            />
          )}
        </div>
      )}

      {/* Select All Pages Banner */}
      {shouldShowSelectAllBanner && (
        <SelectionBanner
          mode="offer"
          currentPageCount={data.length}
          totalCount={totalRows || data.length}
          onSelectAllPages={handleSelectAllPages}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* Select All Pages Active Banner */}
      {state.selectAllPagesMode && (
        <SelectionBanner
          mode="active"
          currentPageCount={data.length}
          totalCount={totalRows || data.length}
          onSelectAllPages={handleSelectAllPages}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* Table Container */}
      <div
        ref={containerRef}
        className="flex-1 overflow-hidden min-h-0"
        tabIndex={0}
        style={{ outline: 'none' }}
      >
        <div
          ref={tableContainerRef}
          className="relative overflow-auto virtual-scroll-container"
          style={{
            height: typeof height === 'number' ? `${height}px` : height || '400px',
          }}
        >
          <table className="w-full border-collapse table-auto">
            {/* Header */}
            <thead className="sticky top-0 z-10 bg-background border-b border-border">
              {table.getHeaderGroups().map(headerGroup => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map(header => (
                    <TableHeader
                      key={header.id}
                      header={header}
                      canSort={header.column.getCanSort()}
                      canFilter={header.column.getCanFilter()}
                      canResize={header.column.getCanResize()}
                      sortDirection={header.column.getIsSorted()}
                    />
                  ))}
                </tr>
              ))}
            </thead>

            {/* Body */}
            {virtualizerWorking ? (
              <TableBodyVirtualized
                virtualItems={virtualItems}
                rows={rows}
                columns={columns}
                totalSize={totalSize}
                rowCount={rowCount}
                measureElement={virtualizer.measureElement}
                onRowClick={onRowClick}
              />
            ) : (
              <TableBodyStatic
                rows={rows}
                columns={columns}
                onRowClick={onRowClick}
              />
            )}
          </table>

          {loading && (
            <div className="absolute inset-0 bg-background bg-opacity-75 flex items-center justify-center">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          )}
        </div>
      </div>

      {/* Footer */}
      {renderedFooter === null ? null : (
        <div className="flex-shrink-0 border-t border-border">
          {renderedFooter ?? (
            <StatusBar
              totalRows={data.length}
              selectedRows={state.selectedRows.length}
              filteredRows={table.getRowModel().rows.length}
              dirtyRows={state.dirtyRows.size}
              loading={loading}
            />
          )}
        </div>
      )}
            </div>
          </SelectionStateContext.Provider>
        </EditStateContext.Provider>
      </TableConfigContext.Provider>
    </TableContext.Provider>
  );
};

export default EditableTable;
