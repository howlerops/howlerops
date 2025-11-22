import { Eye } from 'lucide-react';
import React, { memo } from 'react';

import type { CellValue, TableColumn, TableRow } from '../../../types/table';
import { useTableContext } from '../context/table-context';
import { TableCell } from '../table-cell';

interface TableCellRendererProps {
  rowId: string;
  columnId: string;
  value: CellValue;
  column: TableColumn;
  rowData: TableRow;
}

export const TableCellRenderer = memo<TableCellRendererProps>(({
  rowId,
  columnId,
  value,
  column,
  rowData,
}) => {
  const { state, actions, onRowInspect, customCellRenderers } = useTableContext();

  // Check if there's a custom renderer for this column
  const customRenderer = customCellRenderers[columnId];
  if (customRenderer) {
    return (
      <div
        className="relative group h-full"
        data-row-id={rowId}
        data-column-id={columnId}
      >
        {customRenderer(value, rowData)}
        {onRowInspect && (
          <button
            type="button"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onRowInspect(rowId, rowData);
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
      data-row-id={rowId}
      data-column-id={columnId}
    >
      <TableCell
        value={value}
        rowId={rowId}
        columnId={columnId}
        column={column}
        isEditing={
          state.editingCell?.rowId === rowId &&
          state.editingCell?.columnId === columnId
        }
        isSelected={state.selectedRows.includes(rowId)}
        isDirty={state.dirtyRows.has(rowId)}
        isInvalid={state.invalidCells.has(`${rowId}|${columnId}`)}
        validationError={state.invalidCells.get(`${rowId}|${columnId}`)?.error}
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
}, (prevProps, nextProps) => {
  // Only re-render if these props change
  return (
    prevProps.rowId === nextProps.rowId &&
    prevProps.columnId === nextProps.columnId &&
    prevProps.value === nextProps.value &&
    prevProps.column === nextProps.column &&
    prevProps.rowData === nextProps.rowData
  );
});

TableCellRenderer.displayName = 'TableCellRenderer';
