/**
 * Realtime Table Editor - Interactive table with real-time synchronization
 * Combines optimistic updates, conflict resolution, and collaborative editing
 */

import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Badge } from '../ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../ui/table';
import {
  Edit3,
  Save,
  X,
  Users,
  AlertTriangle,
  Clock,
  CheckCircle,
  Loader2,
} from 'lucide-react';
import { useTableSync } from '../../hooks/websocket';
import { useConflictResolution } from '../../hooks/websocket';
import { OptimisticUpdateIndicator } from './OptimisticUpdateIndicator';
import { ConflictResolutionModal } from './ConflictResolutionModal';
// import { TableEdit, TableEditConflict } from '../../types/websocket'; // Will be used for future conflict resolution features

interface Column {
  key: string;
  name: string;
  type: string;
  editable?: boolean;
}

interface TableRowData {
  rowId: string | number;
  data: Record<string, unknown>;
}

interface RealtimeTableEditorProps {
  tableId: string;
  tableName: string;
  schema?: string;
  columns: Column[];
  initialData?: TableRowData[];
  onDataChange?: (data: TableRowData[]) => void;
  readOnly?: boolean;
}

interface EditingCell {
  rowId: string | number;
  column: string;
  value: string;
  originalValue: unknown;
}

export function RealtimeTableEditor({
  tableId,
  tableName,
  schema,
  columns,
  initialData = [],
  onDataChange,
  readOnly = false,
}: RealtimeTableEditorProps) {
  const {
    syncState,
    optimisticState,
    editCell,
    insertRow,
    updateRow, // eslint-disable-line @typescript-eslint/no-unused-vars
    deleteRow,
    getTableData,
    getRow,
    loadTableData,
  } = useTableSync({
    tableId,
    tableName,
    schema,
    optimisticUpdates: true,
    conflictResolution: 'auto',
    onConflict: (conflict) => {
      setActiveConflictId(conflict.editId);
      setIsConflictModalOpen(true);
    },
  });

  const { activeConflicts, hasConflicts } = useConflictResolution(); // eslint-disable-line @typescript-eslint/no-unused-vars

  // Local state
  const [editingCell, setEditingCell] = useState<EditingCell | null>(null);
  const [isConflictModalOpen, setIsConflictModalOpen] = useState(false);
  const [activeConflictId, setActiveConflictId] = useState<string | null>(null);

  // Load initial data
  useEffect(() => {
    if (initialData.length > 0) {
      const dataMap: Record<string | number, Record<string, unknown>> = {};
      initialData.forEach(({ rowId, data }) => {
        dataMap[rowId] = data;
      });
      loadTableData(dataMap);
    }
  }, [initialData, loadTableData]);

  // Get current table data
  const tableData = useMemo(() => getTableData(), [getTableData]);

  // Notify parent of data changes
  useEffect(() => {
    if (onDataChange) {
      onDataChange(tableData);
    }
  }, [tableData, onDataChange]);

  /**
   * Start editing a cell
   */
  const startEditing = useCallback((rowId: string | number, column: string) => {
    if (readOnly) return;

    const row = getRow(rowId);
    if (!row) return;

    const columnDef = columns.find(col => col.key === column);
    if (!columnDef?.editable) return;

    setEditingCell({
      rowId,
      column,
      value: String(row[column] || ''),
      originalValue: row[column],
    });
  }, [readOnly, getRow, columns]);

  /**
   * Cancel editing
   */
  const cancelEditing = useCallback(() => {
    setEditingCell(null);
  }, []);

  /**
   * Save cell edit
   */
  const saveEdit = useCallback(async () => {
    if (!editingCell) return;

    try {
      // Parse value based on column type
      const columnDef = columns.find(col => col.key === editingCell.column);
      let parsedValue: string | number | boolean = editingCell.value;

      if (columnDef?.type === 'number') {
        parsedValue = parseFloat(editingCell.value);
        if (isNaN(parsedValue)) {
          throw new Error('Invalid number');
        }
      } else if (columnDef?.type === 'boolean') {
        parsedValue = editingCell.value.toLowerCase() === 'true';
      } else if (columnDef?.type === 'json') {
        parsedValue = JSON.parse(editingCell.value);
      }

      await editCell(
        editingCell.rowId,
        editingCell.column,
        parsedValue,
        editingCell.originalValue
      );

      setEditingCell(null);

    } catch (error) {
      console.error('Failed to save edit:', error);
      // Could show error toast here
    }
  }, [editingCell, columns, editCell]);

  /**
   * Handle key press in editing cell
   */
  const handleKeyPress = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      saveEdit();
    } else if (e.key === 'Escape') {
      cancelEditing();
    }
  }, [saveEdit, cancelEditing]);

  /**
   * Add new row
   */
  const handleAddRow = useCallback(async () => {
    if (readOnly) return;

    const newRow: Record<string, unknown> = {};
    columns.forEach(col => {
      if (col.key !== 'id') {
        newRow[col.key] = '';
      }
    });

    try {
      await insertRow(newRow);
    } catch (error) {
      console.error('Failed to add row:', error);
    }
  }, [readOnly, columns, insertRow]);

  /**
   * Delete row
   */
  const handleDeleteRow = useCallback(async (rowId: string | number) => {
    if (readOnly) return;

    try {
      await deleteRow(rowId);
    } catch (error) {
      console.error('Failed to delete row:', error);
    }
  }, [readOnly, deleteRow]);

  /**
   * Get cell display value
   */
  const getCellDisplayValue = useCallback((value: unknown, columnType: string): string => {
    if (value === null || value === undefined) return '';
    if (columnType === 'json') return JSON.stringify(value);
    return String(value);
  }, []);

  /**
   * Check if cell has conflicts
   */
  const hasCellConflict = useCallback((rowId: string | number, column: string): boolean => {
    return hasConflicts(tableId, rowId, column);
  }, [hasConflicts, tableId]);

  /**
   * Get cell status (pending, confirmed, etc.)
   */
  const getCellStatus = useCallback((rowId: string | number, column: string) => {
    // Check optimistic updates
    const pendingUpdate = Array.from(optimisticState.updates.values()).find(
      update => update.rowId === rowId && column in update.changes
    );

    if (pendingUpdate) {
      return {
        status: pendingUpdate.status,
        icon: pendingUpdate.status === 'pending' ? (
          <Loader2 className="h-3 w-3 animate-spin text-blue-500" />
        ) : pendingUpdate.status === 'confirmed' ? (
          <CheckCircle className="h-3 w-3 text-green-500" />
        ) : (
          <AlertTriangle className="h-3 w-3 text-red-500" />
        ),
      };
    }

    // Check conflicts
    if (hasCellConflict(rowId, column)) {
      return {
        status: 'conflict',
        icon: <AlertTriangle className="h-3 w-3 text-orange-500" />,
      };
    }

    return null;
  }, [optimisticState.updates, hasCellConflict]);

  /**
   * Render table cell
   */
  const renderCell = useCallback((
    rowId: string | number,
    column: Column,
    value: unknown
  ) => {
    const isEditing = editingCell?.rowId === rowId && editingCell?.column === column.key;
    const cellStatus = getCellStatus(rowId, column.key);
    const displayValue = getCellDisplayValue(value, column.type);

    if (isEditing) {
      return (
        <div className="flex items-center gap-2">
          <Input
            value={editingCell.value}
            onChange={(e) => setEditingCell(prev => prev ? { ...prev, value: e.target.value } : null)}
            onKeyDown={handleKeyPress}
            onBlur={saveEdit}
            autoFocus
            className="h-8"
          />
          <Button size="sm" variant="ghost" onClick={saveEdit} className="h-6 w-6 p-0">
            <Save className="h-3 w-3" />
          </Button>
          <Button size="sm" variant="ghost" onClick={cancelEditing} className="h-6 w-6 p-0">
            <X className="h-3 w-3" />
          </Button>
        </div>
      );
    }

    return (
      <div
        className={`flex items-center gap-2 group ${
          column.editable && !readOnly ? 'cursor-pointer hover:bg-gray-50' : ''
        } ${cellStatus?.status === 'conflict' ? 'bg-orange-50 border border-orange-200' : ''}`}
        onClick={() => startEditing(rowId, column.key)}
      >
        <span className="flex-1 truncate">{displayValue}</span>

        {cellStatus && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                {cellStatus.icon}
              </TooltipTrigger>
              <TooltipContent>
                <div className="text-xs">
                  {cellStatus.status === 'pending' && 'Update pending...'}
                  {cellStatus.status === 'confirmed' && 'Update confirmed'}
                  {cellStatus.status === 'rejected' && 'Update failed'}
                  {cellStatus.status === 'conflict' && 'Conflict detected'}
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}

        {column.editable && !readOnly && (
          <Edit3 className="h-3 w-3 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
        )}
      </div>
    );
  }, [
    editingCell,
    getCellStatus,
    getCellDisplayValue,
    handleKeyPress,
    saveEdit,
    cancelEditing,
    startEditing,
    readOnly,
  ]);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-lg font-medium">{tableName}</h3>
          {syncState.isOnline ? (
            <Badge variant="outline" className="text-green-600 border-green-200">
              <Users className="h-3 w-3 mr-1" />
              Live
            </Badge>
          ) : (
            <Badge variant="outline" className="text-gray-600 border-gray-200">
              <Clock className="h-3 w-3 mr-1" />
              Offline
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <OptimisticUpdateIndicator tableId={tableId} />

          {!readOnly && (
            <Button size="sm" onClick={handleAddRow}>
              Add Row
            </Button>
          )}
        </div>
      </div>

      {/* Table */}
      <div className="border rounded-lg overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map(column => (
                <TableHead key={column.key} className="font-medium">
                  <div className="flex items-center gap-2">
                    {column.name}
                    {column.editable && (
                      <Edit3 className="h-3 w-3 text-gray-400" />
                    )}
                  </div>
                </TableHead>
              ))}
              {!readOnly && (
                <TableHead className="w-20">Actions</TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {tableData.map(({ rowId, data }) => (
              <TableRow key={rowId}>
                {columns.map(column => (
                  <TableCell key={column.key} className="py-2">
                    {renderCell(rowId, column, data[column.key])}
                  </TableCell>
                ))}
                {!readOnly && (
                  <TableCell className="py-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleDeleteRow(rowId)}
                      className="h-6 w-16 text-xs"
                    >
                      Delete
                    </Button>
                  </TableCell>
                )}
              </TableRow>
            ))}
            {tableData.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={columns.length + (readOnly ? 0 : 1)}
                  className="text-center text-gray-500 py-8"
                >
                  No data available
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Status Bar */}
      <div className="flex items-center justify-between text-sm text-gray-600">
        <div className="flex items-center gap-4">
          <span>{tableData.length} rows</span>
          {syncState.pendingEdits.size > 0 && (
            <span>{syncState.pendingEdits.size} pending edits</span>
          )}
          {syncState.conflicts.size > 0 && (
            <span className="text-orange-600">
              {syncState.conflicts.size} conflicts
            </span>
          )}
        </div>

        {syncState.lastSync && (
          <div className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            <span>Last sync: {syncState.lastSync.toLocaleTimeString()}</span>
          </div>
        )}
      </div>

      {/* Conflict Resolution Modal */}
      <ConflictResolutionModal
        isOpen={isConflictModalOpen}
        onClose={() => {
          setIsConflictModalOpen(false);
          setActiveConflictId(null);
        }}
        conflictId={activeConflictId}
      />
    </div>
  );
}