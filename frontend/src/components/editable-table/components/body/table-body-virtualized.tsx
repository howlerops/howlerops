import { ColumnDef,Row } from '@tanstack/react-table';
import { VirtualItem } from '@tanstack/react-virtual';
import React from 'react';

import { TableRow } from '../../../../types/table';
import { MemoizedVirtualRow } from './virtual-row';

interface TableBodyVirtualizedProps {
  virtualItems: VirtualItem[];
  rows: Row<TableRow>[];
  columns: ColumnDef<TableRow>[];
  totalSize: number | undefined;
  rowCount: number;
  measureElement: (element: Element | null) => void;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
}

export const TableBodyVirtualized: React.FC<TableBodyVirtualizedProps> = ({
  virtualItems,
  rows,
  columns,
  totalSize,
  rowCount,
  measureElement,
  onRowClick,
}) => {
  return (
    <tbody>
      {/* Top spacer */}
      {virtualItems.length > 0 && virtualItems[0].index > 0 && (
        <tr>
          <td
            colSpan={columns.length}
            style={{ height: virtualItems[0].start, padding: 0, border: 'none' }}
          />
        </tr>
      )}

      {/* Virtual rows */}
      {virtualItems.map(virtualItem => {
        // Critical: Bounds check BEFORE array access (ag-Grid pattern)
        // Prevents accessing undefined when virtualizer is out of sync
        if (virtualItem.index < 0 || virtualItem.index >= rows.length) {
          return null;
        }

        const row = rows[virtualItem.index];
        // Double-check row exists and has required data
        if (!row || !row.id) {
          return null;
        }

        return (
          <MemoizedVirtualRow
            key={row.id}
            row={row}
            columns={columns}
            state={undefined}
            actions={undefined}
            tableColumns={undefined}
            isVirtual={true}
            virtualItem={virtualItem}
            onRowClick={onRowClick}
            ref={measureElement}
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
                height: totalSize! - virtualItems[virtualItems.length - 1].end,
                padding: 0,
                border: 'none',
              }}
            />
          </tr>
        )}
    </tbody>
  );
};
