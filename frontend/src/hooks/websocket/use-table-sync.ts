/**
 * useTableSync Hook - Real-time table synchronization with optimistic updates
 * Manages table edits, conflict resolution, and collaborative editing
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { v4 as uuidv4 } from 'uuid';
import {
  TableEdit,
  TableEditConflict,
  TableRowChange,
  UseTableSyncOptions,
  OptimisticUpdate,
  OptimisticState,
  EventHandler,
} from '../../types/websocket';
import { useWebSocket } from './use-websocket';

interface TableSyncState {
  tableData: Map<string | number, Record<string, unknown>>;
  pendingEdits: Map<string, TableEdit>;
  conflicts: Map<string, TableEditConflict>;
  version: number;
  lastSync: Date | null;
  isOnline: boolean;
}

const initialState: TableSyncState = {
  tableData: new Map(),
  pendingEdits: new Map(),
  conflicts: new Map(),
  version: 0,
  lastSync: null,
  isOnline: false,
};

export function useTableSync(options: UseTableSyncOptions) {
  const {
    tableId,
    tableName,
    schema,
    conflictResolution = 'auto',
    optimisticUpdates = true,
  } = options;

  const { sendMessage, on, off, connectionState, joinRoom, leaveRoom } = useWebSocket();

  // State
  const [syncState, setSyncState] = useState<TableSyncState>(initialState);
  const [optimisticState, setOptimisticState] = useState<OptimisticState>({
    updates: new Map(),
    pendingCount: 0,
    lastUpdate: null,
  });

  // Refs
  const versionRef = useRef(0);
  const lockTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Apply optimistic update
   */
  const applyOptimisticUpdate = useCallback((update: OptimisticUpdate) => {
    if (!optimisticUpdates) return;

    setOptimisticState(prev => {
      const newUpdates = new Map(prev.updates);
      newUpdates.set(update.id, update);

      return {
        updates: newUpdates,
        pendingCount: newUpdates.size,
        lastUpdate: new Date(),
      };
    });

    // Apply to local table data
    setSyncState(prev => {
      const newTableData = new Map(prev.tableData);

      if (update.type === 'table_edit' && update.rowId) {
        const existingRow = newTableData.get(update.rowId) || {};
        newTableData.set(update.rowId, { ...existingRow, ...update.changes });
      }

      return {
        ...prev,
        tableData: newTableData,
      };
    });
  }, [optimisticUpdates]);

  /**
   * Confirm optimistic update
   */
  const confirmOptimisticUpdate = useCallback((updateId: string) => {
    setOptimisticState(prev => {
      const newUpdates = new Map(prev.updates);
      const update = newUpdates.get(updateId);

      if (update) {
        update.status = 'confirmed';
        newUpdates.set(updateId, update);

        // Remove confirmed update after a delay
        setTimeout(() => {
          setOptimisticState(current => {
            const updates = new Map(current.updates);
            updates.delete(updateId);
            return {
              ...current,
              updates,
              pendingCount: updates.size,
            };
          });
        }, 1000);
      }

      return {
        ...prev,
        updates: newUpdates,
        pendingCount: Array.from(newUpdates.values()).filter(u => u.status === 'pending').length,
      };
    });
  }, []);

  /**
   * Reject optimistic update and rollback
   */
  const rejectOptimisticUpdate = useCallback((updateId: string) => {
    setOptimisticState(prev => {
      const update = prev.updates.get(updateId);
      if (!update) return prev;

      // Rollback the change
      setSyncState(current => {
        const newTableData = new Map(current.tableData);

        if (update.type === 'table_edit' && update.rowId) {
          const currentRow = newTableData.get(update.rowId);
          if (currentRow) {
            // Restore original values
            const restoredRow = { ...currentRow };
            Object.keys(update.changes).forEach(key => {
              if (Object.prototype.hasOwnProperty.call(update.originalData, key)) {
                restoredRow[key] = update.originalData[key];
              } else {
                delete restoredRow[key];
              }
            });
            newTableData.set(update.rowId, restoredRow);
          }
        }

        return {
          ...current,
          tableData: newTableData,
        };
      });

      // Remove from optimistic updates
      const newUpdates = new Map(prev.updates);
      newUpdates.delete(updateId);

      return {
        updates: newUpdates,
        pendingCount: newUpdates.size,
        lastUpdate: new Date(),
      };
    });
  }, []);

  /**
   * Edit a table cell
   */
  const editCell = useCallback(async (
    rowId: string | number,
    column: string,
    newValue: unknown,
    oldValue?: unknown
  ) => {
    const editId = uuidv4();
    const currentRow = syncState.tableData.get(rowId);

    if (!currentRow && oldValue === undefined) {
      throw new Error('Row not found and no old value provided');
    }

    const actualOldValue = oldValue !== undefined ? oldValue : currentRow?.[column];

    const edit: TableEdit = {
      editId,
      tableId,
      tableName,
      schema,
      rowId,
      column,
      oldValue: actualOldValue,
      newValue,
      version: versionRef.current,
      status: 'pending',
      optimistic: optimisticUpdates,
    };

    // Apply optimistic update
    if (optimisticUpdates) {
      const optimisticUpdate: OptimisticUpdate = {
        id: editId,
        type: 'table_edit',
        tableId,
        rowId,
        changes: { [column]: newValue },
        originalData: { [column]: actualOldValue },
        timestamp: Date.now(),
        status: 'pending',
      };

      applyOptimisticUpdate(optimisticUpdate);
    }

    // Add to pending edits
    setSyncState(prev => ({
      ...prev,
      pendingEdits: new Map(prev.pendingEdits).set(editId, edit),
    }));

    try {
      await sendMessage('table_edit', {
        tableId,
        tableName,
        schema,
        rowId,
        column,
        oldValue: actualOldValue,
        newValue,
        editId,
        version: versionRef.current,
      });

    } catch (error) {
      // Reject optimistic update on error
      if (optimisticUpdates) {
        rejectOptimisticUpdate(editId);
      }

      // Remove from pending edits
      setSyncState(prev => {
        const newPendingEdits = new Map(prev.pendingEdits);
        newPendingEdits.delete(editId);
        return { ...prev, pendingEdits: newPendingEdits };
      });

      throw error;
    }
  }, [
    syncState.tableData,
    tableId,
    tableName,
    schema,
    optimisticUpdates,
    sendMessage,
    applyOptimisticUpdate,
    rejectOptimisticUpdate,
  ]);

  /**
   * Insert a new row
   */
  const insertRow = useCallback(async (
    row: Record<string, unknown>,
    _position?: number  
  ) => {
    const operationId = uuidv4();
    const tempRowId = `temp_${Date.now()}`;

    // Apply optimistic update
    if (optimisticUpdates) {
      const optimisticUpdate: OptimisticUpdate = {
        id: operationId,
        type: 'row_operation',
        tableId,
        rowId: tempRowId,
        changes: row,
        originalData: {},
        timestamp: Date.now(),
        status: 'pending',
      };

      applyOptimisticUpdate(optimisticUpdate);
    }

    try {
      await sendMessage('table_row_operation', {
        operation: 'insert',
        tableId,
        tableName,
        schema,
        row,
        version: versionRef.current,
      });

    } catch (error) {
      if (optimisticUpdates) {
        rejectOptimisticUpdate(operationId);
      }
      throw error;
    }
  }, [tableId, tableName, schema, optimisticUpdates, sendMessage, applyOptimisticUpdate, rejectOptimisticUpdate]);

  /**
   * Update an entire row
   */
  const updateRow = useCallback(async (
    rowId: string | number,
    changes: Record<string, unknown>
  ) => {
    const operationId = uuidv4();
    const currentRow = syncState.tableData.get(rowId);

    if (!currentRow) {
      throw new Error('Row not found');
    }

    // Apply optimistic update
    if (optimisticUpdates) {
      const originalData: Record<string, unknown> = {};
      Object.keys(changes).forEach(key => {
        originalData[key] = currentRow[key];
      });

      const optimisticUpdate: OptimisticUpdate = {
        id: operationId,
        type: 'row_operation',
        tableId,
        rowId,
        changes,
        originalData,
        timestamp: Date.now(),
        status: 'pending',
      };

      applyOptimisticUpdate(optimisticUpdate);
    }

    try {
      await sendMessage('table_row_operation', {
        operation: 'update',
        tableId,
        tableName,
        schema,
        rowId,
        changes,
        version: versionRef.current,
      });

    } catch (error) {
      if (optimisticUpdates) {
        rejectOptimisticUpdate(operationId);
      }
      throw error;
    }
  }, [
    syncState.tableData,
    tableId,
    tableName,
    schema,
    optimisticUpdates,
    sendMessage,
    applyOptimisticUpdate,
    rejectOptimisticUpdate,
  ]);

  /**
   * Delete a row
   */
  const deleteRow = useCallback(async (rowId: string | number) => {
    const currentRow = syncState.tableData.get(rowId);

    if (!currentRow) {
      throw new Error('Row not found');
    }

    // Apply optimistic update (remove from local state)
    if (optimisticUpdates) {
      setSyncState(prev => {
        const newTableData = new Map(prev.tableData);
        newTableData.delete(rowId);
        return { ...prev, tableData: newTableData };
      });
    }

    try {
      await sendMessage('table_row_operation', {
        operation: 'delete',
        tableId,
        tableName,
        schema,
        rowId,
        version: versionRef.current,
      });

    } catch (error) {
      // Restore row on error
      if (optimisticUpdates) {
        setSyncState(prev => {
          const newTableData = new Map(prev.tableData);
          newTableData.set(rowId, currentRow);
          return { ...prev, tableData: newTableData };
        });
      }
      throw error;
    }
  }, [syncState.tableData, tableId, tableName, schema, optimisticUpdates, sendMessage]);

  /**
   * Resolve a conflict manually
   */
  const resolveConflict = useCallback(async (
    conflictId: string,
    resolution: 'accept_local' | 'accept_remote' | 'custom',
    customValue?: unknown
  ) => {
    const conflict = syncState.conflicts.get(conflictId);
    if (!conflict) {
      throw new Error('Conflict not found');
    }

    let resolvedValue: unknown;

    switch (resolution) {
      case 'accept_local': {
        // Use the local value (from optimistic update)
        const optimisticUpdate = optimisticState.updates.get(conflict.editId);
        resolvedValue = optimisticUpdate?.changes || conflict.mergedValue;
        break;
      }
      case 'accept_remote':
        resolvedValue = conflict.mergedValue;
        break;
      case 'custom':
        resolvedValue = customValue;
        break;
    }

    try {
      await sendMessage('resolve_conflict', {
        conflictId,
        resolution,
        resolvedValue,
      });

      // Remove conflict from local state
      setSyncState(prev => {
        const newConflicts = new Map(prev.conflicts);
        newConflicts.delete(conflictId);
        return { ...prev, conflicts: newConflicts };
      });

    } catch (error) {
      console.error('Failed to resolve conflict:', error);
      throw error;
    }
  }, [syncState.conflicts, optimisticState.updates, sendMessage]);

  /**
   * Handle edit apply event
   */
  const handleEditApply = useCallback((event: { editId: string; success: boolean; error?: string }) => {
    const edit = syncState.pendingEdits.get(event.editId);
    if (!edit) return;

    if (event.success) {
      // Confirm optimistic update
      confirmOptimisticUpdate(event.editId);

      // Update version
      versionRef.current += 1;
      setSyncState(prev => ({ ...prev, version: versionRef.current }));

    } else {
      // Reject optimistic update
      rejectOptimisticUpdate(event.editId);
    }

    // Remove from pending edits
    setSyncState(prev => {
      const newPendingEdits = new Map(prev.pendingEdits);
      newPendingEdits.delete(event.editId);
      return { ...prev, pendingEdits: newPendingEdits };
    });

    options.onEditApply?.(event);
  }, [syncState.pendingEdits, confirmOptimisticUpdate, rejectOptimisticUpdate, options]);

  /**
   * Handle edit conflict event
   */
  const handleConflict = useCallback((conflict: TableEditConflict) => {
    setSyncState(prev => ({
      ...prev,
      conflicts: new Map(prev.conflicts).set(conflict.editId, conflict),
    }));

    // Auto-resolve if configured
    if (conflictResolution === 'auto') {
      setTimeout(() => {
        resolveConflict(conflict.editId, 'accept_remote', conflict.mergedValue);
      }, 1000);
    }

    options.onConflict?.(conflict);
  }, [conflictResolution, resolveConflict, options]);

  /**
   * Handle row change event
   */
  const handleRowChange = useCallback((change: TableRowChange) => {
    if (change.tableId !== tableId) return;

    setSyncState(prev => {
      const newTableData = new Map(prev.tableData);

      switch (change.operation) {
        case 'insert':
          newTableData.set(change.rowId, change.changes);
          break;
        case 'update': {
          const existingRow = newTableData.get(change.rowId) || {};
          newTableData.set(change.rowId, { ...existingRow, ...change.changes });
          break;
        }
        case 'delete':
          newTableData.delete(change.rowId);
          break;
      }

      return {
        ...prev,
        tableData: newTableData,
        version: Math.max(prev.version, change.version),
        lastSync: new Date(),
      };
    });

    versionRef.current = Math.max(versionRef.current, change.version);
    options.onRowChange?.(change);
  }, [tableId, options]);

  /**
   * Load initial table data
   */
  const loadTableData = useCallback(async (data: Record<string | number, Record<string, unknown>>) => {
    setSyncState(prev => ({
      ...prev,
      tableData: new Map(Object.entries(data)),
      lastSync: new Date(),
    }));
  }, []);

  /**
   * Get current table data as array
   */
  const getTableData = useCallback((): Array<{ rowId: string | number; data: Record<string, unknown> }> => {
    return Array.from(syncState.tableData.entries()).map(([rowId, data]) => ({ rowId, data }));
  }, [syncState.tableData]);

  /**
   * Get row by ID
   */
  const getRow = useCallback((rowId: string | number): Record<string, unknown> | null => {
    return syncState.tableData.get(rowId) || null;
  }, [syncState.tableData]);

  /**
   * Check if table has pending changes
   */
  const hasPendingChanges = syncState.pendingEdits.size > 0 || optimisticState.pendingCount > 0;

  /**
   * Get sync statistics
   */
  const getStats = useCallback(() => {
    return {
      tableId,
      rowCount: syncState.tableData.size,
      version: syncState.version,
      pendingEdits: syncState.pendingEdits.size,
      conflicts: syncState.conflicts.size,
      optimisticUpdates: optimisticState.pendingCount,
      lastSync: syncState.lastSync,
      isOnline: syncState.isOnline,
    };
  }, [tableId, syncState, optimisticState]);

  // Set up event handlers
  useEffect(() => {
    on('table:edit:apply', handleEditApply as EventHandler);
    on('table:edit:conflict', handleConflict as EventHandler);
    on('table:row:update', handleRowChange as EventHandler);
    on('table:row:insert', handleRowChange as EventHandler);
    on('table:row:delete', handleRowChange as EventHandler);

    return () => {
      off('table:edit:apply', handleEditApply as EventHandler);
      off('table:edit:conflict', handleConflict as EventHandler);
      off('table:row:update', handleRowChange as EventHandler);
      off('table:row:insert', handleRowChange as EventHandler);
      off('table:row:delete', handleRowChange as EventHandler);
    };
  }, [on, off, handleEditApply, handleConflict, handleRowChange]);

  // Join table room when connected
  useEffect(() => {
    if (connectionState.status === 'connected') {
      joinRoom(tableId, 'table', { tableName, schema });
       
      setSyncState(prev => ({ ...prev, isOnline: true }));
    } else {
      setSyncState(prev => ({ ...prev, isOnline: false }));
    }
  }, [connectionState.status, tableId, tableName, schema, joinRoom]);

  // Cleanup on unmount
  useEffect(() => {
    const currentLockTimeout = lockTimeoutRef.current;
    return () => {
      leaveRoom(tableId);
      if (currentLockTimeout) {
        clearTimeout(currentLockTimeout);
      }
    };
  }, [tableId, leaveRoom]);

  return {
    // State
    syncState,
    optimisticState,
    hasPendingChanges,

    // Actions
    editCell,
    insertRow,
    updateRow,
    deleteRow,
    resolveConflict,
    loadTableData,

    // Data access
    getTableData,
    getRow,

    // Utilities
    getStats,
  };
}