import { AlertCircle, Eye } from 'lucide-react';
import React, { memo, useCallback, useEffect, useRef, useState } from 'react';

import { CellEditState, CellValue, TableColumn, TableRow } from '../../types/table';
import { cn } from '../../utils/cn';
import { formatCellValue } from '../../utils/table';
import { Button } from '../ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { CellEditor } from './cell-editor';

interface TableCellProps {
  value: CellValue;
  rowId: string;
  columnId: string;
  column: TableColumn;
  isEditing: boolean;
  isSelected: boolean;
  isDirty: boolean;
  isInvalid: boolean;
  validationError?: string;
  onEdit: (rowId: string, columnId: string, value: CellValue) => void;
  onSave: () => Promise<boolean>;
  onCancel: () => void;
  onUpdateEdit: (value: CellValue, isValid: boolean, error?: string) => void;
  editingState: CellEditState | null;
  onInspectRow?: (rowId: string, rowData: TableRow) => void;
  rowData: TableRow;
}

export const TableCell = memo<TableCellProps>(({
  value,
  rowId,
  columnId,
  column,
  isEditing,
  isSelected,
  isDirty,
  isInvalid,
  validationError,
  onEdit,
  onSave,
  onCancel,
  onUpdateEdit,
  editingState,
  onInspectRow,
  rowData,
}) => {
  const cellRef = useRef<HTMLDivElement>(null);
  const doubleClickTimeoutRef = useRef<number | null>(null);
  const [showValueDialog, setShowValueDialog] = useState(false);

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
      const editorValue = editingState ? editingState.value : value;
      return (
        <CellEditor
          value={editorValue}
          type={column.type}
          onChange={(newValue, isValid, error) => {
            onUpdateEdit(newValue, isValid, error);
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

    if (!isEditing && value === undefined && column.hasDefault) {
      return (
        <span className="text-xs italic text-muted-foreground">
          {column.defaultLabel || '[default]'}
        </span>
      );
    }

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
      case 'datetime':
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
        {
          const shouldWrap = column.wrapContent ?? column.longText ?? false;
          const shouldTruncate = column.clipContent !== false && !shouldWrap;
          const textClassName = cn(
            'w-full',
            column.monospace && 'font-mono tabular-nums',
            shouldWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-nowrap',
            shouldTruncate ? 'truncate' : 'break-words'
          );

          return (
            <div className="flex w-full items-center gap-1">
              <div className={textClassName} title={shouldTruncate ? formattedValue : undefined}>
                {formattedValue}
              </div>
              {column.longText && !isEditing && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0 text-muted-foreground"
                  onClick={(event) => {
                    event.stopPropagation();
                    setShowValueDialog(true);
                  }}
                >
                  <Eye className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
          );
        }
    }
  };

  const renderValidationError = () => {
    const errorMessage = isEditing ? editingState?.error : validationError;
    if (errorMessage) {
      return (
        <div className="absolute z-50 mt-1 p-2 bg-destructive border border-destructive rounded text-xs text-destructive shadow-lg">
          {errorMessage}
        </div>
      );
    }
    return null;
  };

  return (
    <>
      <div
        ref={cellRef}
        className={cn(
          'relative group h-full w-full flex items-center px-3 py-2 text-sm',
          'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-inset',
          'cursor-pointer select-none',
          {
            'bg-primary/10 border-primary': isSelected,
            'bg-accent/10 border-accent': isDirty && !isInvalid,
            'bg-muted': isEditing,
            'cursor-not-allowed opacity-60': !column.editable,
            'hover:bg-muted/50': column.editable && !isEditing && !isSelected,
            // Invalid cell styling - red border takes precedence
            'border-2 border-destructive bg-destructive/5': isInvalid,
            'hover:bg-destructive/10': isInvalid && column.editable && !isEditing,
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

        {/* Error indicator */}
        {isInvalid && !isEditing && (
          <div className="absolute top-1 right-1">
            <AlertCircle className="h-3 w-3 text-destructive" />
          </div>
        )}

        {/* Dirty indicator */}
        {isDirty && !isEditing && !isInvalid && (
          <div className="absolute top-1 right-1 w-2 h-2 bg-accent rounded-full" />
        )}

        {/* Required field indicator */}
        {column.required && (
          <div className="absolute top-1 left-1 text-destructive text-xs">*</div>
        )}

        {/* JSON inspector trigger */}
        {onInspectRow && !isEditing && (
          <button
            type="button"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onInspectRow(rowId, rowData);
            }}
            className={cn(
              'absolute bottom-1 right-1 flex h-6 w-6 items-center justify-center rounded-full',
              'bg-background/80 text-muted-foreground shadow-sm',
              'opacity-0 transition-opacity duration-150',
              'group-hover:opacity-100 focus-visible:opacity-100'
            )}
            tabIndex={-1}
            aria-label="Open row JSON"
          >
            <Eye className="h-3.5 w-3.5" />
          </button>
        )}
      </div>
      {column.longText && !isEditing && (
        <Dialog open={showValueDialog} onOpenChange={setShowValueDialog}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>{column.header}</DialogTitle>
            </DialogHeader>
            <div className="max-h-[60vh] overflow-auto whitespace-pre-wrap text-sm font-mono text-foreground/90">
              {formatCellValue(value, column.type) || '∅'}
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}, (prevProps, nextProps) => {
  // Custom comparison function (ag-Grid pattern)
  // Only re-render if these critical props actually change
  return (
    prevProps.value === nextProps.value &&
    prevProps.isDirty === nextProps.isDirty &&
    prevProps.isInvalid === nextProps.isInvalid &&
    prevProps.isEditing === nextProps.isEditing &&
    prevProps.isSelected === nextProps.isSelected &&
    prevProps.validationError === nextProps.validationError &&
    prevProps.columnId === nextProps.columnId &&
    prevProps.rowId === nextProps.rowId
  );
});

TableCell.displayName = 'TableCell';
