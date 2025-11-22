/**
 * useOptimisticUpdates Hook - Manages optimistic UI updates
 * Handles local state updates before server confirmation
 */

import { useCallback, useMemo,useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

import { OptimisticState,OptimisticUpdate } from '../../types/websocket';

interface OptimisticUpdateConfig {
  enabled: boolean;
  maxPendingUpdates: number;
  timeoutMs: number;
  rollbackOnError: boolean;
}

const DEFAULT_CONFIG: OptimisticUpdateConfig = {
  enabled: true,
  maxPendingUpdates: 100,
  timeoutMs: 10000, // 10 seconds
  rollbackOnError: true,
};

export function useOptimisticUpdates(config: Partial<OptimisticUpdateConfig> = {}) {
  const fullConfig = useMemo(() => ({ ...DEFAULT_CONFIG, ...config }), [config]);

  // State
  const [optimisticState, setOptimisticState] = useState<OptimisticState>({
    updates: new Map(),
    pendingCount: 0,
    lastUpdate: null,
  });

  // Refs
  const timeoutRefs = useRef<Map<string, NodeJS.Timeout>>(new Map());
  const originalDataRef = useRef<Map<string, Record<string, unknown>>>(new Map());

  /**
   * Reject and rollback an optimistic update
   */
  const rollbackUpdate = useCallback((updateId: string, reason: 'error' | 'conflict' | 'timeout' = 'error') => {
    // Clear timeout
    const timeoutId = timeoutRefs.current.get(updateId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutRefs.current.delete(updateId);
    }

    const update = optimisticState.updates.get(updateId);
    if (!update) return null;

    // Update status
    setOptimisticState(prev => {
      const newUpdates = new Map(prev.updates);

      if (fullConfig.rollbackOnError) {
        // Remove the update entirely
        newUpdates.delete(updateId);
      } else {
        // Mark as rejected but keep for debugging
        update.status = 'rejected';
        newUpdates.set(updateId, update);
      }

      return {
        updates: newUpdates,
        pendingCount: Array.from(newUpdates.values()).filter(u => u.status === 'pending').length,
        lastUpdate: new Date(),
      };
    });

    // Get original data for rollback
    const originalData = originalDataRef.current.get(updateId);
    originalDataRef.current.delete(updateId);

    console.log(`Optimistic update ${updateId} rolled back due to: ${reason}`);

    return originalData;
  }, [optimisticState.updates, fullConfig.rollbackOnError]);

  /**
   * Apply an optimistic update
   */
  const applyUpdate = useCallback((
    id: string,
    type: 'table_edit' | 'row_operation',
    tableId: string,
    changes: Record<string, unknown>,
    originalData: Record<string, unknown>,
    rowId?: string | number
  ): string => {
    if (!fullConfig.enabled) {
      return id;
    }

    // Check if we're at max pending updates
    if (optimisticState.pendingCount >= fullConfig.maxPendingUpdates) {
      console.warn('Maximum pending optimistic updates reached');
      return id;
    }

    const updateId = id || uuidv4();

    const update: OptimisticUpdate = {
      id: updateId,
      type,
      tableId,
      rowId,
      changes,
      originalData,
      timestamp: Date.now(),
      status: 'pending',
    };

    setOptimisticState(prev => {
      const newUpdates = new Map(prev.updates);
      newUpdates.set(updateId, update);

      return {
        updates: newUpdates,
        pendingCount: newUpdates.size,
        lastUpdate: new Date(),
      };
    });

    // Store original data for potential rollback
    originalDataRef.current.set(updateId, originalData);

    // Set timeout for automatic rollback
    const timeoutId = setTimeout(() => {
      rollbackUpdate(updateId, 'timeout');
    }, fullConfig.timeoutMs);

    timeoutRefs.current.set(updateId, timeoutId);

    return updateId;
  }, [fullConfig.enabled, fullConfig.maxPendingUpdates, fullConfig.timeoutMs, optimisticState.pendingCount, rollbackUpdate]);

  /**
   * Confirm an optimistic update
   */
  const confirmUpdate = useCallback((updateId: string) => {
    // Clear timeout
    const timeoutId = timeoutRefs.current.get(updateId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutRefs.current.delete(updateId);
    }

    // Update status
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
              pendingCount: Array.from(updates.values()).filter(u => u.status === 'pending').length,
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

    // Clean up original data
    originalDataRef.current.delete(updateId);
  }, []);

  /**
   * Get pending updates for a specific table
   */
  const getPendingUpdates = useCallback((tableId?: string): OptimisticUpdate[] => {
    const updates = Array.from(optimisticState.updates.values())
      .filter(update => update.status === 'pending');

    if (tableId) {
      return updates.filter(update => update.tableId === tableId);
    }

    return updates;
  }, [optimisticState.updates]);

  /**
   * Get specific update by ID
   */
  const getUpdate = useCallback((updateId: string): OptimisticUpdate | null => {
    return optimisticState.updates.get(updateId) || null;
  }, [optimisticState.updates]);

  /**
   * Check if there are pending updates for a table/row
   */
  const hasPendingUpdates = useCallback((tableId?: string, rowId?: string | number): boolean => {
    const updates = getPendingUpdates(tableId);

    if (rowId !== undefined) {
      return updates.some(update => update.rowId === rowId);
    }

    return updates.length > 0;
  }, [getPendingUpdates]);

  /**
   * Apply optimistic changes to data
   */
  const applyOptimisticChangesToData = useCallback(<T extends Record<string, unknown>>(
    data: T,
    tableId: string,
    rowId?: string | number
  ): T => {
    if (!fullConfig.enabled) return data;

    const pendingUpdates = getPendingUpdates(tableId);
    let modifiedData = { ...data };

    for (const update of pendingUpdates) {
      if (rowId !== undefined && update.rowId !== rowId) continue;

      // Apply changes
      modifiedData = { ...modifiedData, ...update.changes };
    }

    return modifiedData;
  }, [fullConfig.enabled, getPendingUpdates]);

  /**
   * Clear all pending updates (useful for reset scenarios)
   */
  const clearAllUpdates = useCallback(() => {
    // Clear all timeouts
    for (const timeoutId of timeoutRefs.current.values()) {
      clearTimeout(timeoutId);
    }
    timeoutRefs.current.clear();
    originalDataRef.current.clear();

    setOptimisticState({
      updates: new Map(),
      pendingCount: 0,
      lastUpdate: null,
    });
  }, []);

  /**
   * Get statistics about optimistic updates
   */
  const getStats = useCallback(() => {
    const updates = Array.from(optimisticState.updates.values());
    const now = Date.now();

    return {
      total: updates.length,
      pending: updates.filter(u => u.status === 'pending').length,
      confirmed: updates.filter(u => u.status === 'confirmed').length,
      rejected: updates.filter(u => u.status === 'rejected').length,
      averageAge: updates.length > 0
        ? updates.reduce((sum, u) => sum + (now - u.timestamp), 0) / updates.length
        : 0,
      oldestUpdate: updates.length > 0
        ? Math.min(...updates.map(u => u.timestamp))
        : null,
      config: fullConfig,
    };
  }, [optimisticState.updates, fullConfig]);

  /**
   * Health check for optimistic updates
   */
  const healthCheck = useCallback(() => {
    const stats = getStats();
    const issues: string[] = [];
    let healthy = true;

    // Check for too many pending updates
    if (stats.pending > fullConfig.maxPendingUpdates * 0.8) {
      issues.push(`High number of pending updates: ${stats.pending}`);
      healthy = false;
    }

    // Check for old pending updates
    if (stats.averageAge > fullConfig.timeoutMs * 0.8) {
      issues.push(`Pending updates are aging: ${Math.round(stats.averageAge)}ms average`);
    }

    // Check rejection rate
    const rejectionRate = stats.total > 0 ? stats.rejected / stats.total : 0;
    if (rejectionRate > 0.1) { // 10% rejection rate threshold
      issues.push(`High rejection rate: ${Math.round(rejectionRate * 100)}%`);
    }

    return {
      healthy,
      issues,
      stats,
    };
  }, [getStats, fullConfig]);

  return {
    // State
    optimisticState,

    // Actions
    applyUpdate,
    confirmUpdate,
    rollbackUpdate,
    clearAllUpdates,

    // Queries
    getPendingUpdates,
    getUpdate,
    hasPendingUpdates,
    applyOptimisticChangesToData,

    // Utilities
    getStats,
    healthCheck,
  };
}