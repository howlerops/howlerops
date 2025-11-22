import { ColumnDef, flexRender } from '@tanstack/react-table';
import { VirtualItem } from '@tanstack/react-virtual';
import React, { memo, useCallback } from 'react';

import { TableRow } from '../../../../types/table';

interface VirtualRowProps {
  row: unknown;
  columns: ColumnDef<TableRow>[];
  state: unknown;
  actions: unknown;
  tableColumns: unknown;
  isVirtual?: boolean;
  virtualItem?: VirtualItem;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
}

// Comparison function for VirtualRow memo - only re-render if row data or virtual item changes
const arePropsEqual = (
  prev: { row: unknown; virtualItem?: VirtualItem; onRowClick?: (rowId: string, rowData: TableRow) => void },
  next: { row: unknown; virtualItem?: VirtualItem; onRowClick?: (rowId: string, rowData: TableRow) => void }
) => {
  const prevRow = prev.row as { original?: { __rowId?: string } } | null;
  const nextRow = next.row as { original?: { __rowId?: string } } | null;

  // Check if row ID changed (most important check)
  if (prevRow?.original?.__rowId !== nextRow?.original?.__rowId) {
    return false;
  }

  // Check if virtual item index changed
  if (prev.virtualItem?.index !== next.virtualItem?.index) {
    return false;
  }

  // Check if virtual item key changed (size/position updates)
  if (prev.virtualItem?.key !== next.virtualItem?.key) {
    return false;
  }

  // onRowClick is stable, no need to check
  return true;
};

// Simplified VirtualRow component following official pattern
const VirtualRow = memo(React.forwardRef<HTMLTableRowElement, VirtualRowProps>(
  ({ row, onRowClick, virtualItem }, ref) => {
    // Critical validation: Ensure row data exists and is valid (ag-Grid pattern)
    const rowData = row as { original?: { __rowId?: string }; getVisibleCells?: () => unknown[] } | null;

    const handleRowClick = useCallback(() => {
      if (!onRowClick || !rowData?.original?.__rowId) {
        return;
      }
      onRowClick(rowData.original.__rowId, rowData.original);
    }, [onRowClick, rowData]);

    // Return null if row data is invalid (prevents rendering dummy rows)
    if (!rowData || !rowData.original || !rowData.getVisibleCells) {
      return null;
    }

    return (
      <tr
        ref={ref}
        data-index={typeof virtualItem?.index === 'number' ? virtualItem.index : undefined}
        className="border-b border-border hover:bg-muted/50 cursor-pointer"
        onClick={handleRowClick}
      >
        {rowData.getVisibleCells().map((cell: unknown) => {
          const cellData = cell as {
            id: string;
            column: {
              getSize: () => number;
              columnDef: { cell: unknown; meta?: { sticky?: 'left' | 'right' } };
            };
            getContext: () => object;
          };
          const sticky = cellData.column.columnDef.meta?.sticky;
          const columnSize = cellData.column.getSize();
          return (
            <td
              key={cellData.id}
              className={`px-3 py-1 text-sm ${sticky ? `sticky ${sticky === 'right' ? 'right-0' : 'left-0'} bg-background z-10 shadow-sm` : ''}`}
              style={{
                width: columnSize,
                minWidth: columnSize,
                maxWidth: columnSize,
              }}
            >
              {flexRender(
                cellData.column.columnDef.cell as ((props: object) => React.ReactNode) | React.ReactNode,
                cellData.getContext()
              )}
            </td>
          );
        })}
      </tr>
    );
  }
));

VirtualRow.displayName = 'VirtualRow';

// Apply custom comparison to VirtualRow memo
export const MemoizedVirtualRow = memo(VirtualRow, arePropsEqual) as typeof VirtualRow;
