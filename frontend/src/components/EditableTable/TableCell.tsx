import React, { memo, useCallback, useRef, useEffect } from 'react';
import { cn } from '../../utils/cn';
import { CellValue, TableColumn, CellEditState } from '../../types/table';
import { formatCellValue } from '../../utils/table';
import { CellEditor } from './CellEditor';

interface TableCellProps {
  value: CellValue;
  rowId: string;
  columnId: string;
  column: TableColumn;
  isEditing: boolean;
  isSelected: boolean;
  isDirty: boolean;
  onEdit: (rowId: string, columnId: string, value: CellValue) => void;
  onSave: () => Promise<boolean>;
  onCancel: () => void;
  onUpdateEdit: (value: CellValue, isValid: boolean, error?: string) => void;
  editingState: CellEditState | null;
}

export const TableCell = memo<TableCellProps>(({
  value,
  rowId,
  columnId,
  column,
  isEditing,
  isSelected,
  isDirty,
  onEdit,
  onSave,
  onCancel,
  onUpdateEdit,
  editingState,
}) => {
  const cellRef = useRef<HTMLDivElement>(null);
  const doubleClickTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleDoubleClick = useCallback(() => {
    if (!column.editable) return;
    onEdit(rowId, columnId, value);
  }, [column.editable, onEdit, rowId, columnId, value]);

  const handleSingleClick = useCallback(() => {
    // Clear any existing timeout
    if (doubleClickTimeoutRef.current) {
      clearTimeout(doubleClickTimeoutRef.current);
      doubleClickTimeoutRef.current = null;
    }

    // Set a timeout to handle single click
    doubleClickTimeoutRef.current = window.setTimeout(() => {
      // Single click logic can be added here if needed
      // For now, we only handle double-click for editing
    }, 200);
  }, []);

  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (!column.editable) return;

    switch (event.key) {
      case 'Enter':
      case 'F2':
        event.preventDefault();
        onEdit(rowId, columnId, value);
        break;
      case ' ':
        if (column.type === 'boolean') {
          event.preventDefault();
          onEdit(rowId, columnId, value);
        }
        break;
    }
  }, [column.editable, column.type, onEdit, rowId, columnId, value]);

  // Clean up timeout on unmount
  useEffect(() => {
    return () => {
      if (doubleClickTimeoutRef.current) {
        clearTimeout(doubleClickTimeoutRef.current);
        doubleClickTimeoutRef.current = null;
      }
    };
  }, []);

  const renderCellContent = () => {
    if (isEditing) {
      return (
        <CellEditor
          value={editingState?.value ?? value}
          type={column.type}
          onChange={(newValue) => {
            // Validate the new value here if needed
            onUpdateEdit(newValue, true); // Simplified validation for now
          }}
          onCancel={onCancel}
          onSave={onSave}
          validation={column.validation}
          options={column.options}
          required={column.required}
          autoFocus
        />
      );
    }

    const formattedValue = formatCellValue(value, column.type);

    // Special rendering for different types
    switch (column.type) {
      case 'boolean':
        return (
          <div className="flex items-center justify-center">
            <input
              type="checkbox"
              checked={Boolean(value)}
              readOnly
              className="rounded border-gray-300"
            />
          </div>
        );

      case 'number':
        return (
          <div className="text-right font-mono">
            {formattedValue}
          </div>
        );

      case 'date':
        return (
          <div className="font-mono text-gray-600">
            {formattedValue}
          </div>
        );

      case 'select':
        return (
          <div className="flex items-center">
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-gray-100 text-gray-800">
              {formattedValue}
            </span>
          </div>
        );

      default:
        return (
          <div className="truncate" title={formattedValue}>
            {formattedValue}
          </div>
        );
    }
  };

  const renderValidationError = () => {
    if (isEditing && editingState?.error) {
      return (
        <div className="absolute z-50 mt-1 p-2 bg-red-100 border border-red-300 rounded text-xs text-red-700 shadow-lg">
          {editingState.error}
        </div>
      );
    }
    return null;
  };

  return (
    <div
      ref={cellRef}
      className={cn(
        'relative h-full w-full flex items-center px-3 py-2 text-sm',
        'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-inset',
        'cursor-pointer select-none',
        {
          'bg-blue-50 border-blue-200': isSelected,
          'bg-yellow-50 border-yellow-200': isDirty,
          'bg-gray-50': isEditing,
          'cursor-not-allowed opacity-60': !column.editable,
          'hover:bg-gray-50': column.editable && !isEditing && !isSelected,
        }
      )}
      onClick={handleSingleClick}
      onDoubleClick={handleDoubleClick}
      onKeyDown={handleKeyDown}
      tabIndex={column.editable ? 0 : -1}
      role="gridcell"
      aria-selected={isSelected}
      aria-label={`${column.header}: ${formatCellValue(value, column.type)}`}
    >
      {renderCellContent()}
      {renderValidationError()}

      {/* Dirty indicator */}
      {isDirty && !isEditing && (
        <div className="absolute top-1 right-1 w-2 h-2 bg-orange-400 rounded-full" />
      )}

      {/* Required field indicator */}
      {column.required && (
        <div className="absolute top-1 left-1 text-red-500 text-xs">*</div>
      )}
    </div>
  );
});

TableCell.displayName = 'TableCell';
