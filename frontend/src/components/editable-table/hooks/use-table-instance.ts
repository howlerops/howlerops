import { type ColumnDef, getCoreRowModel, getFilteredRowModel, getSortedRowModel, type OnChangeFn, type RowSelectionState,useReactTable } from '@tanstack/react-table';

import type { EditableTableActions,TableRow, TableState } from '../../../types/table';

export interface UseTableInstanceProps {
  data: TableRow[];
  columns: ColumnDef<TableRow>[];
  state: TableState;
  actions: EditableTableActions;
  displayMode: 'view' | 'edit';
  enableMultiSelect: boolean;
  enableColumnResizing: boolean;
  enableGlobalFilter: boolean;
  internalRowSelection: RowSelectionState;
  setInternalRowSelection: OnChangeFn<RowSelectionState>;
}

export function useTableInstance(props: UseTableInstanceProps) {
  const {
    data,
    columns,
    state,
    actions,
    displayMode,
    enableMultiSelect,
    enableColumnResizing,
    enableGlobalFilter,
    internalRowSelection,
    setInternalRowSelection,
  } = props;

  const isVeryLarge = data.length > 10000;

  return useReactTable<TableRow>({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: displayMode === 'view' ? getSortedRowModel() : undefined,
    getFilteredRowModel: displayMode === 'view' ? getFilteredRowModel() : undefined,
    enableMultiSort: !isVeryLarge,
    enableGlobalFilter: !isVeryLarge && enableGlobalFilter,
    enableRowSelection: !isVeryLarge && enableMultiSelect,
    enableColumnResizing: !isVeryLarge && enableColumnResizing,
    columnResizeMode: 'onChange',
    getRowId: (row, index) => String(index),
    state: {
      sorting: state.sorting,
      columnFilters: state.columnFilters,
      globalFilter: state.globalFilter,
      rowSelection: internalRowSelection,
      columnVisibility: state.columnVisibility,
      columnOrder: state.columnOrder,
      columnSizing: state.columnSizing,
    },
    onSortingChange: (updater) => {
      const currentState = state.sorting;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateSorting(next);
    },
    onColumnFiltersChange: (updater) => {
      const currentState = state.columnFilters;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateColumnFilters(next);
    },
    onGlobalFilterChange: (updater) => {
      const currentState = state.globalFilter;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateGlobalFilter(next ?? '');
    },
    onRowSelectionChange: setInternalRowSelection,
    onColumnVisibilityChange: (updater) => {
      const currentState = state.columnVisibility;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateColumnVisibility(next);
    },
    onColumnOrderChange: (updater) => {
      const currentState = state.columnOrder;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateColumnOrder(next);
    },
    onColumnSizingChange: (updater) => {
      const currentState = state.columnSizing;
      const next = typeof updater === 'function' ? updater(currentState) : updater;
      actions.updateColumnSizing(next);
    },
  });
}
