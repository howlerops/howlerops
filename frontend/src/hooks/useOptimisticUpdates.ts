import { useState, useCallback, useRef, useEffect } from 'react';
import { TableRow, TableAction } from '../types/table';
import { cloneDeep } from '../utils/table';

interface OptimisticUpdate {
  id: string;
  type: 'update' | 'create' | 'delete';
  timestamp: number;
  rollbackData: unknown;
  retryCount: number;
  maxRetries: number;
  promise: Promise<boolean>;
  status: 'pending' | 'success' | 'failed' | 'rolledback';
}

interface UseOptimisticUpdatesOptions {
  maxRetries?: number;
  retryDelay?: number;
  autoRollbackDelay?: number;
  onSuccess?: (update: OptimisticUpdate) => void;
  onError?: (update: OptimisticUpdate, error: unknown) => void;
  onRollback?: (update: OptimisticUpdate) => void;
}

export const useOptimisticUpdates = (
  data: TableRow[],
  setData: (data: TableRow[]) => void,
  options: UseOptimisticUpdatesOptions = {}
) => {
  const {
    maxRetries = 3,
    retryDelay = 1000,
    autoRollbackDelay = 5000,
    onSuccess,
    onError,
    onRollback,
  } = options;

  const [pendingUpdates, setPendingUpdates] = useState<Map<string, OptimisticUpdate>>(new Map());
  const [failedUpdates, setFailedUpdates] = useState<Set<string>>(new Set());
  const originalDataRef = useRef<TableRow[]>(data);
  const timeoutsRef = useRef<Map<string, NodeJS.Timeout>>(new Map());

  // Update original data reference when data changes from external source
  useEffect(() => {
    if (pendingUpdates.size === 0) {
      originalDataRef.current = cloneDeep(data);
    }
  }, [data, pendingUpdates.size]);

  const generateUpdateId = () => {
    return `update_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  };

  const rollbackUpdate = useCallback((updateId: string) => {
    const update = pendingUpdates.get(updateId);
    if (!update) return;

    // Clear any pending timeout
    const timeout = timeoutsRef.current.get(updateId);
    if (timeout) {
      clearTimeout(timeout);
      timeoutsRef.current.delete(updateId);
    }

    // Restore original data
    switch (update.type) {
      case 'update': {
        setData(prevData => {
          const newData = [...prevData];
          const rollbackData = update.rollbackData as { rowId: string; columnId: string; originalValue: unknown };
          const index = newData.findIndex(row => row.id === rollbackData.rowId);
          if (index !== -1) {
            newData[index] = { ...newData[index], [rollbackData.columnId]: rollbackData.originalValue };
          }
          return newData;
        });
        break;
      }

      case 'create': {
        const rollbackData = update.rollbackData as { id: string };
        setData(prevData => prevData.filter(row => row.id !== rollbackData.id));
        break;
      }

      case 'delete': {
        const rollbackData = update.rollbackData as TableRow;
        setData(prevData => [...prevData, rollbackData]);
        break;
      }
    }

    // Update state
    setPendingUpdates(prev => {
      const newMap = new Map(prev);
      const rolledBackUpdate = { ...update, status: 'rolledback' as const };
      newMap.set(updateId, rolledBackUpdate);
      return newMap;
    });

    setFailedUpdates(prev => new Set([...prev, updateId]));

    onRollback?.(update);
  }, [pendingUpdates, setData, onRollback]);

  const retryUpdate = useCallback(async (updateId: string) => {
    const update = pendingUpdates.get(updateId);
    if (!update || update.retryCount >= update.maxRetries) {
      rollbackUpdate(updateId);
      return;
    }

    // Wait for retry delay
    await new Promise(resolve => setTimeout(resolve, retryDelay * (update.retryCount + 1)));

    try {
      const success = await update.promise;
      if (success) {
        setPendingUpdates(prev => {
          const newMap = new Map(prev);
          newMap.set(updateId, { ...update, status: 'success', retryCount: update.retryCount + 1 });
          return newMap;
        });
        onSuccess?.(update);
      } else {
        throw new Error('Update failed');
      }
    } catch (error) {
      const updatedUpdate = { ...update, retryCount: update.retryCount + 1 };
      setPendingUpdates(prev => {
        const newMap = new Map(prev);
        newMap.set(updateId, updatedUpdate);
        return newMap;
      });

      if (updatedUpdate.retryCount >= maxRetries) {
        rollbackUpdate(updateId);
      } else {
        retryUpdate(updateId);
      }

      onError?.(updatedUpdate, error);
    }
  }, [pendingUpdates, maxRetries, retryDelay, rollbackUpdate, onSuccess, onError]);

  const scheduleAutoRollback = useCallback((updateId: string) => {
    const timeout = setTimeout(() => {
      const update = pendingUpdates.get(updateId);
      if (update && update.status === 'pending') {
        rollbackUpdate(updateId);
      }
    }, autoRollbackDelay);

    timeoutsRef.current.set(updateId, timeout);
  }, [autoRollbackDelay, pendingUpdates, rollbackUpdate]);

  const optimisticUpdate = useCallback(async (
    updatePromise: Promise<boolean>,
    action: TableAction
  ): Promise<boolean> => {
    const updateId = generateUpdateId();

    let rollbackData: unknown;
    let newData: TableRow[];

    // Apply optimistic update immediately
    switch (action.type) {
      case 'edit': {
        const { rowId, columnId, oldValue, newValue } = action.payload;
        rollbackData = { rowId, columnId, originalValue: oldValue };

        newData = data.map(row =>
          row.id === rowId ? { ...row, [columnId!]: newValue } : row
        );
        setData(newData);
        break;
      }

      case 'add': {
        const newRow = action.payload.rows![0];
        rollbackData = { id: newRow.id };
        newData = [...data, newRow];
        setData(newData);
        break;
      }

      case 'delete': {
        const rowToDelete = data.find(row => row.id === action.payload.rowId);
        rollbackData = rowToDelete;
        newData = data.filter(row => row.id !== action.payload.rowId);
        setData(newData);
        break;
      }

      default:
        return false;
    }

    // Track the optimistic update
    const update: OptimisticUpdate = {
      id: updateId,
      type: action.type === 'edit' ? 'update' : action.type === 'add' ? 'create' : 'delete',
      timestamp: Date.now(),
      rollbackData,
      retryCount: 0,
      maxRetries,
      promise: updatePromise,
      status: 'pending',
    };

    setPendingUpdates(prev => new Map([...prev, [updateId, update]]));

    // Schedule auto-rollback
    scheduleAutoRollback(updateId);

    try {
      const success = await updatePromise;

      if (success) {
        // Update succeeded, mark as successful
        setPendingUpdates(prev => {
          const newMap = new Map(prev);
          newMap.set(updateId, { ...update, status: 'success' });
          return newMap;
        });

        // Clear timeout
        const timeout = timeoutsRef.current.get(updateId);
        if (timeout) {
          clearTimeout(timeout);
          timeoutsRef.current.delete(updateId);
        }

        onSuccess?.(update);
        return true;
      } else {
        // Update failed, start retry process
        retryUpdate(updateId);
        return false;
      }
    } catch (error) {
      // Update failed, start retry process
      setPendingUpdates(prev => {
        const newMap = new Map(prev);
        newMap.set(updateId, { ...update, status: 'failed' });
        return newMap;
      });

      retryUpdate(updateId);
      onError?.(update, error);
      return false;
    }
  }, [data, setData, maxRetries, scheduleAutoRollback, retryUpdate, onSuccess, onError]);

  const manualRollback = useCallback((updateId: string) => {
    rollbackUpdate(updateId);
  }, [rollbackUpdate]);

  const clearCompletedUpdates = useCallback(() => {
    setPendingUpdates(prev => {
      const newMap = new Map();
      for (const [id, update] of prev) {
        if (update.status === 'pending' || update.status === 'failed') {
          newMap.set(id, update);
        }
      }
      return newMap;
    });

    setFailedUpdates(new Set());
  }, []);

  const getPendingUpdatesCount = useCallback(() => {
    return Array.from(pendingUpdates.values()).filter(
      update => update.status === 'pending'
    ).length;
  }, [pendingUpdates]);

  const getFailedUpdatesCount = useCallback(() => {
    return failedUpdates.size;
  }, [failedUpdates]);

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      timeoutsRef.current.forEach(timeout => clearTimeout(timeout));
      timeoutsRef.current.clear();
    };
  }, []);

  return {
    optimisticUpdate,
    manualRollback,
    clearCompletedUpdates,
    getPendingUpdatesCount,
    getFailedUpdatesCount,
    pendingUpdates: Array.from(pendingUpdates.values()),
    failedUpdates: Array.from(failedUpdates),
  };
};