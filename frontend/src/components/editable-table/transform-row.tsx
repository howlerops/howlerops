/**
 * TransformRow Component
 *
 * GPU-accelerated virtual row using transform positioning (ag-Grid pattern)
 * - Uses `transform: translateY()` instead of spacer rows
 * - Maintains table semantics with nested structure
 * - Enables 60fps scrolling for large datasets
 */

import React, { memo, CSSProperties, useCallback } from 'react';
import { Row, flexRender } from '@tanstack/react-table';
import { VirtualItem } from '@tanstack/react-virtual';
import { TableRow } from '../../types/table';

interface TransformRowProps {
  row: Row<TableRow>;
  virtualItem: VirtualItem;
  columns: { id: string; getSize: () => number }[];
  columnWidths: number[];
  measureElement?: (node: Element | null) => void;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
}

export const TransformRow = memo(
  React.forwardRef<HTMLDivElement, TransformRowProps>(
    ({ row, virtualItem, columns, columnWidths, measureElement, onRowClick }, ref) => {
      const rowData = row.original;
      const rowId = rowData.__rowId || String(row.id);

      const handleRowClick = useCallback(() => {
        if (onRowClick && rowId) {
          onRowClick(rowId, rowData);
        }
      }, [onRowClick, rowId, rowData]);

      // Combine external ref with measurement
      const measureRef = useCallback(
        (node: HTMLDivElement | null) => {
          if (measureElement) {
            measureElement(node);
          }
          if (typeof ref === 'function') {
            ref(node);
          } else if (ref) {
            (ref as React.MutableRefObject<HTMLDivElement | null>).current = node;
          }
        },
        [measureElement, ref]
      );

      // GPU-accelerated wrapper positioning
      const wrapperStyle: CSSProperties = {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        // GPU-accelerated transform (ag-Grid pattern)
        transform: `translateY(${virtualItem.start}px)`,
        // Hint browser to use GPU layer
        willChange: 'transform',
      };

      // Inner table for proper column alignment
      const tableStyle: CSSProperties = {
        tableLayout: 'fixed',
        width: '100%',
        borderCollapse: 'collapse',
      };

      return (
        <div
          ref={measureRef}
          style={wrapperStyle}
          data-index={virtualItem.index}
          role="presentation"
        >
          <table style={tableStyle} role="presentation">
            <tbody role="presentation">
              <tr
                className="border-b border-border hover:bg-muted/50 transition-colors cursor-pointer"
                onClick={handleRowClick}
                data-row-id={rowId}
                role="row"
              >
                {row.getVisibleCells().map((cell, cellIndex) => {
                  const columnSize = columnWidths[cellIndex] || cell.column.getSize();
                  const sticky = (cell.column.columnDef.meta as { sticky?: 'left' | 'right' })?.sticky;

                  return (
                    <td
                      key={cell.id}
                      className={`px-3 py-2 text-sm ${
                        sticky
                          ? `sticky ${sticky === 'right' ? 'right-0' : 'left-0'} bg-background z-10 shadow-sm`
                          : ''
                      }`}
                      style={{
                        width: columnSize,
                        minWidth: columnSize,
                        maxWidth: columnSize,
                      }}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  );
                })}
              </tr>
            </tbody>
          </table>
        </div>
      );
    }
  )
);

TransformRow.displayName = 'TransformRow';
