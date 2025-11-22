import type { ColumnDef } from '@tanstack/react-table';
import { useMemo } from 'react';

import type { CellValue, TableColumn, TableRow } from '../../../types/table';
import { TableCellRenderer } from '../components/table-cell-renderer';

export interface UseTableColumnsProps {
  tableColumns: TableColumn[];
  enableMultiSelect: boolean;
  enableColumnResizing: boolean;
}

export function useTableColumns(props: UseTableColumnsProps): ColumnDef<TableRow>[] {
  const { tableColumns, enableMultiSelect, enableColumnResizing } = props;

  return useMemo<ColumnDef<TableRow>[]>(() => {
    const baseColumns: ColumnDef<TableRow>[] = tableColumns.map(col => {
      const columnId = col.id ?? (typeof col.accessorKey === 'string' ? col.accessorKey : col.header);
      return {
        id: columnId,
        accessorKey: col.accessorKey ?? columnId,
        header: col.header,
        size: col.width || col.preferredWidth || col.minWidth || 150,
        minSize: col.minWidth || 80,
        maxSize: col.maxWidth || 400,
        enableSorting: col.sortable !== false,
        enableColumnFilter: col.filterable !== false,
        enableResizing: enableColumnResizing,
        meta: {
          sticky: col.sticky,
          originalColumn: col,
        },
        cell: ({ row, getValue }) => {
          return (
            <TableCellRenderer
              rowId={row.original.__rowId!}
              columnId={columnId}
              value={getValue() as CellValue}
              column={col}
              rowData={row.original}
            />
          );
        },
      };
    });

    if (enableMultiSelect) {
      baseColumns.unshift({
        id: 'select',
        header: ({ table }) => {
          const allSelected = table.getIsAllRowsSelected();
          const someSelected = table.getIsSomeRowsSelected();

          return (
            <input
              type="checkbox"
              checked={allSelected}
              ref={(el) => {
                if (el) {
                  el.indeterminate = someSelected && !allSelected;
                }
              }}
              onChange={table.getToggleAllRowsSelectedHandler()}
              className="rounded border-border focus:ring-2 focus:ring-ring"
              aria-label="Select all rows"
            />
          );
        },
        cell: ({ row }) => {
          const isSelected = row.getIsSelected();

          return (
            <input
              type="checkbox"
              checked={isSelected}
              onChange={row.getToggleSelectedHandler()}
              className="rounded border-border focus:ring-2 focus:ring-ring"
              aria-label={`Select row ${row.index + 1}`}
            />
          );
        },
        size: 40,
        enableSorting: false,
        enableColumnFilter: false,
        enableResizing: false,
      });
    }

    return baseColumns;
  }, [
    tableColumns,
    enableMultiSelect,
    enableColumnResizing,
  ]);
}
