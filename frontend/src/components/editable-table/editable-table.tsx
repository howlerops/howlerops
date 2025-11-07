import React, { useMemo, useCallback, useRef, useEffect, memo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  ColumnDef,
  flexRender,
} from '@tanstack/react-table';
import { useVirtualizer } from '@tanstack/react-virtual';
import { Eye } from 'lucide-react';
import { cn } from '../../utils/cn';
import { useTableState } from '../../hooks/use-table-state';
import { useKeyboardNavigation } from '../../hooks/use-keyboard-navigation';
import {
  EditableTableProps,
  TableRow,
  TableColumn,
  CellValue,
  EditableTableContext,
  EditableTableRenderer,
} from '../../types/table';
import { getColumnWidth } from '../../utils/table';
import { TableCell } from './table-cell';
import { TableHeader } from './table-header';
import { TableToolbar } from './table-toolbar';
import { StatusBar } from './status-bar';

// Simplified VirtualRow component following official pattern
const VirtualRow = memo(React.forwardRef<HTMLTableRowElement, {
  row: unknown;
  columns: ColumnDef<TableRow>[];
  state: unknown;
  actions: unknown;
  tableColumns: TableColumn[];
  isVirtual?: boolean;
  virtualItem?: unknown;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
}>(({ row, onRowClick }, ref) => {
  const rowData = row as { original: { __rowId: string }; getVisibleCells: () => unknown[] };
  
  const handleRowClick = useCallback(() => {
    if (onRowClick && rowData.original.__rowId) {
      onRowClick(rowData.original.__rowId, rowData.original);
    }
  }, [onRowClick, rowData.original]);
  
  return (
    <tr
      ref={ref}
      className="border-b border-border hover:bg-muted/50 cursor-pointer"
      onClick={handleRowClick}
    >
      {rowData.getVisibleCells().map((cell: unknown) => {
        const cellData = cell as { id: string; column: { getSize: () => number; columnDef: { cell: unknown; meta?: { sticky?: 'left' | 'right' } } }; getContext: () => object };
        const sticky = cellData.column.columnDef.meta?.sticky;
        const columnSize = cellData.column.getSize();
        return (
          <td
            key={cellData.id}
            className={`px-3 py-2 text-sm ${sticky ? `sticky ${sticky === 'right' ? 'right-0' : 'left-0'} bg-background z-10 shadow-sm` : ''}`}
            style={{
              width: columnSize,
              minWidth: columnSize,
              maxWidth: columnSize,
            }}
          >
            {flexRender(cellData.column.columnDef.cell as ((props: object) => React.ReactNode) | React.ReactNode, cellData.getContext())}
          </td>
        );
      })}
    </tr>
  );
}));

VirtualRow.displayName = 'VirtualRow';

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
  loading = false,
  error = null,
  virtualScrolling = true,
  estimateSize = 35,
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
}) => {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const {
    data,
    setData,
    state,
    actions,
  } = useTableState(initialData);

  useEffect(() => {
    setData(initialData);
  }, [initialData, setData]);

  // Create TanStack Table columns
  const columns = useMemo<ColumnDef<TableRow>[]>(() => {
    const baseColumns: ColumnDef<TableRow>[] = tableColumns.map(col => {
      const columnId = col.id ?? (typeof col.accessorKey === 'string' ? col.accessorKey : col.header);
      return {
        id: columnId,
        accessorKey: col.accessorKey ?? columnId,
        header: col.header,
        size: getColumnWidth(col, data),
        minSize: col.minWidth || 80,
        maxSize: col.maxWidth || 400,
        enableSorting: col.sortable !== false,
        enableColumnFilter: col.filterable !== false,
        enableResizing: enableColumnResizing,
        meta: {
          sticky: col.sticky,
          originalColumn: col,
        },
        cell: ({ row, column, getValue }) => {
          const rawColumnId = column.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } }).columnDef?.id ?? (column as unknown as { columnDef?: { id?: string; accessorKey?: string } }).columnDef?.accessorKey;
          const currentColumnId = String(rawColumnId ?? columnId);
          const rowData = row.original as TableRow;

          // Check if there's a custom renderer for this column
          const customRenderer = customCellRenderers[currentColumnId];
          if (customRenderer) {
            return (
              <div
                className="relative group h-full"
                data-row-id={row.original.__rowId!}
                data-column-id={currentColumnId}
              >
                {customRenderer(getValue() as CellValue, row.original)}
                {onRowInspect && (
                  <button
                    type="button"
                    onClick={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      onRowInspect(row.original.__rowId!, row.original);
                    }}
                    className="absolute bottom-1 right-1 flex h-6 w-6 items-center justify-center rounded-full bg-background/80 text-muted-foreground opacity-0 shadow-sm transition-opacity duration-150 group-hover:opacity-100 focus-visible:opacity-100"
                    tabIndex={-1}
                    aria-label="Open row JSON"
                  >
                    <Eye className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            );
          }

          return (
            <div
              data-row-id={row.original.__rowId!}
              data-column-id={currentColumnId}
            >
              <TableCell
                value={getValue() as CellValue}
                rowId={row.original.__rowId!}
                columnId={currentColumnId}
                column={col}
                isEditing={
                  state.editingCell?.rowId === row.original.__rowId &&
                  state.editingCell?.columnId === currentColumnId
                }
                isSelected={state.selectedRows.includes(row.original.__rowId!)}
                isDirty={state.dirtyRows.has(row.original.__rowId!)}
                isInvalid={state.invalidCells.has(`${row.original.__rowId!}|${currentColumnId}`)}
                validationError={state.invalidCells.get(`${row.original.__rowId!}|${currentColumnId}`)?.error}
                onEdit={actions.startEditing}
                onSave={actions.saveEditing}
                onCancel={actions.cancelEditing}
                onUpdateEdit={actions.updateEditingCell}
                editingState={state.editingCell}
                onInspectRow={onRowInspect}
                rowData={rowData}
              />
            </div>
          );
        },
      };
    });

    if (enableMultiSelect) {
      baseColumns.unshift({
        id: 'select',
        header: ({ table }) => (
          <input
            type="checkbox"
            checked={table.getIsAllRowsSelected()}
            onChange={table.getToggleAllRowsSelectedHandler()}
            className="rounded border-border focus:ring-2 focus:ring-ring"
            aria-label="Select all rows"
          />
        ),
        cell: ({ row }) => (
          <input
            type="checkbox"
            checked={row.getIsSelected()}
            onChange={row.getToggleSelectedHandler()}
            className="rounded border-border focus:ring-2 focus:ring-ring"
            aria-label={`Select row ${row.index + 1}`}
          />
        ),
        size: 40,
        enableSorting: false,
        enableColumnFilter: false,
        enableResizing: false,
      });
    }

    return baseColumns;
  }, [
    tableColumns,
    data,
    enableMultiSelect,
    enableColumnResizing,
    state,
    actions,
    customCellRenderers,
    onRowInspect,
  ]);

  // TanStack Table returns mutable helpers; safe to instantiate per render.
  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    state: {
      sorting: state.sorting,
      columnFilters: state.columnFilters,
      globalFilter: state.globalFilter,
      columnVisibility: state.columnVisibility,
      columnSizing: state.columnSizing,
      rowSelection: state.selectedRows.reduce((acc, id) => {
        const rowIndex = data.findIndex(row => row.__rowId === id);
        if (rowIndex !== -1) {
          acc[rowIndex] = true;
        }
        return acc;
      }, {} as Record<string, boolean>),
    },
    onSortingChange: (updater) => {
      const next = typeof updater === 'function' ? updater(table.getState().sorting) : updater
      actions.updateSorting(next)
    },
    onColumnFiltersChange: (updater) => {
      const next = typeof updater === 'function' ? updater(table.getState().columnFilters) : updater
      actions.updateColumnFilters(next)
    },
    onGlobalFilterChange: (updater) => {
      const prev = table.getState().globalFilter
      const next = typeof updater === 'function' ? updater(prev) : updater
      actions.updateGlobalFilter(next ?? '')
    },
    onColumnVisibilityChange: (updater) => {
      const next = typeof updater === 'function' ? updater(table.getState().columnVisibility) : updater
      actions.updateColumnVisibility(next)
    },
    onColumnSizingChange: (updater) => {
      const next = typeof updater === 'function' ? updater(table.getState().columnSizing) : updater
      actions.updateColumnSizing(next)
    },
    onRowSelectionChange: (updater) => {
      const newSelection = typeof updater === 'function'
        ? updater(table.getState().rowSelection)
        : updater;

      const selectedIds = Object.keys(newSelection)
        .filter(key => newSelection[key])
        .map((index) => data[parseInt(index)]?.__rowId)
        .filter((id): id is string => Boolean(id));

      actions.selectAllRows(false);
      selectedIds.forEach(id => actions.toggleRowSelection(id, true));
    },
    enableRowSelection: enableMultiSelect,
    enableColumnResizing,
    columnResizeMode: 'onChange',
  });

  const { rows } = table.getRowModel();

  const rowCount = rows.length;
  const shouldVirtualize = virtualScrolling && rowCount > 0;

  const virtualizer = useVirtualizer({
    count: shouldVirtualize ? rowCount : 0,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => estimateSize,
    overscan: 12,
    measureElement: (element) => element?.getBoundingClientRect().height ?? estimateSize,
  });

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

  const renderedToolbar = typeof toolbar === 'function'
    ? (toolbar as EditableTableRenderer)(tableContext)
    : toolbar;

  const renderedFooter = typeof footer === 'function'
    ? (footer as EditableTableRenderer)(tableContext)
    : footer;

  const shouldShowDefaultToolbar = !renderedToolbar && (enableGlobalFilter || enableExport);
  const shouldRenderToolbar = Boolean(renderedToolbar || shouldShowDefaultToolbar);

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

  // Get virtual items following official pattern
  const virtualItems = virtualizerWorking ? virtualizer.getVirtualItems() : null;
  const totalSize = virtualizerWorking ? virtualizer.getTotalSize() : undefined;

  return (
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

      {/* Table Container */}
      <div
        ref={containerRef}
        className="flex-1 overflow-hidden min-h-0"
        tabIndex={0}
        style={{ outline: 'none' }}
      >
        <div
          ref={tableContainerRef}
          className="relative overflow-auto"
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
            <tbody>
              {virtualizerWorking && virtualItems ? (
                <>
                  {/* Top spacer */}
                  {virtualItems.length > 0 && virtualItems[0].index > 0 && (
                    <tr>
                      <td
                        colSpan={columns.length}
                        style={{ height: virtualItems[0].start }}
                      />
                    </tr>
                  )}
                  {/* Virtual rows */}
                  {virtualItems.map(virtualItem => {
                    const row = rows[virtualItem.index];
                    if (!row) return null;
                    
                    return (
                      <VirtualRow
                        key={row.id}
                        row={row}
                        columns={columns}
                        state={state}
                        actions={actions}
                        tableColumns={tableColumns}
                        isVirtual={true}
                        virtualItem={virtualItem}
                        onRowClick={onRowClick}
                        ref={virtualizer.measureElement}
                      />
                    );
                  })}
                  {/* Bottom spacer */}
                  {virtualItems.length > 0 &&
                   virtualItems[virtualItems.length - 1].index < rowCount - 1 && (
                    <tr>
                      <td
                        colSpan={columns.length}
                        style={{
                          height: totalSize! - virtualItems[virtualItems.length - 1].end
                        }}
                      />
                    </tr>
                  )}
                </>
              ) : (
                rows.map(row => (
                  <VirtualRow
                    key={row.id}
                    row={row}
                    columns={columns}
                    state={state}
                    actions={actions}
                    tableColumns={tableColumns}
                    isVirtual={false}
                    onRowClick={onRowClick}
                  />
                ))
              )}
            </tbody>
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
  );
};

export default EditableTable;
