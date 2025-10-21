/**
 * useConflictResolution Hook - Handles conflict detection and resolution UI
 * Manages conflict states and resolution strategies
 */

import { useState, useCallback, useRef, useEffect } from 'react';

interface ConflictResolutionStrategy {
  id: string;
  name: string;
  description: string;
  autoApply: boolean;
  handler: (conflict: ConflictData) => Promise<unknown>;
}

interface ConflictData {
  id: string;
  tableId: string;
  rowId: string | number;
  column: string;
  localValue: unknown;
  remoteValue: unknown;
  baseValue: unknown;
  timestamp: number;
  conflictType: 'value' | 'type' | 'structural';
  metadata?: Record<string, unknown>;
}

interface ConflictResolutionState {
  activeConflicts: Map<string, ConflictData>;
  resolvedConflicts: ConflictData[];
  strategies: ConflictResolutionStrategy[];
  defaultStrategy: string;
  autoResolveEnabled: boolean;
  resolutionHistory: Array<{
    conflictId: string;
    strategy: string;
    resolvedValue: unknown;
    timestamp: number;
  }>;
}

const DEFAULT_STRATEGIES: ConflictResolutionStrategy[] = [
  {
    id: 'last_write_wins',
    name: 'Last Write Wins',
    description: 'Use the most recent change',
    autoApply: true,
    handler: async (conflict) => conflict.remoteValue,
  },
  {
    id: 'first_write_wins',
    name: 'First Write Wins',
    description: 'Keep the original local change',
    autoApply: true,
    handler: async (conflict) => conflict.localValue,
  },
  {
    id: 'manual',
    name: 'Manual Resolution',
    description: 'Require user intervention',
    autoApply: false,
    handler: async () => {
      throw new Error('Manual resolution required');
    },
  },
  {
    id: 'merge_string',
    name: 'Merge Strings',
    description: 'Concatenate string values with delimiter',
    autoApply: true,
    handler: async (conflict) => {
      if (typeof conflict.localValue === 'string' && typeof conflict.remoteValue === 'string') {
        return `${conflict.localValue} | ${conflict.remoteValue}`;
      }
      return conflict.remoteValue;
    },
  },
  {
    id: 'merge_numeric',
    name: 'Average Numbers',
    description: 'Use average of numeric values',
    autoApply: true,
    handler: async (conflict) => {
      if (typeof conflict.localValue === 'number' && typeof conflict.remoteValue === 'number') {
        return (conflict.localValue + conflict.remoteValue) / 2;
      }
      return conflict.remoteValue;
    },
  },
];

export function useConflictResolution() {
  // State
  const [state, setState] = useState<ConflictResolutionState>({
    activeConflicts: new Map(),
    resolvedConflicts: [],
    strategies: DEFAULT_STRATEGIES,
    defaultStrategy: 'last_write_wins',
    autoResolveEnabled: true,
    resolutionHistory: [],
  });

  // Refs
  const autoResolveTimeoutRef = useRef<Map<string, NodeJS.Timeout>>(new Map());

  /**
   * Detect conflict type
   */
  const detectConflictType = useCallback((
    localValue: unknown,
    remoteValue: unknown,
    _baseValue: unknown // eslint-disable-line @typescript-eslint/no-unused-vars
  ): 'value' | 'type' | 'structural' => {
    const localType = typeof localValue;
    const remoteType = typeof remoteValue;

    // Type conflict
    if (localType !== remoteType) {
      return 'type';
    }

    // Structural conflict (for objects/arrays)
    if (localType === 'object' && localValue !== null && remoteValue !== null && 
        typeof localValue === 'object' && typeof remoteValue === 'object') {
      const localKeys = Object.keys(localValue as Record<string, unknown>);
      const remoteKeys = Object.keys(remoteValue as Record<string, unknown>);

      if (localKeys.length !== remoteKeys.length ||
          !localKeys.every(key => remoteKeys.includes(key))) {
        return 'structural';
      }
    }

    return 'value';
  }, []);

  /**
   * Levenshtein distance calculation
   */
  const levenshteinDistance = useCallback((str1: string, str2: string): number => {
    const matrix = Array(str2.length + 1).fill(null).map(() => Array(str1.length + 1).fill(null));

    for (let i = 0; i <= str1.length; i++) matrix[0][i] = i;
    for (let j = 0; j <= str2.length; j++) matrix[j][0] = j;

    for (let j = 1; j <= str2.length; j++) {
      for (let i = 1; i <= str1.length; i++) {
        const indicator = str1[i - 1] === str2[j - 1] ? 0 : 1;
        matrix[j][i] = Math.min(
          matrix[j][i - 1] + 1, // deletion
          matrix[j - 1][i] + 1, // insertion
          matrix[j - 1][i - 1] + indicator // substitution
        );
      }
    }

    return matrix[str2.length][str1.length];
  }, []);

  /**
   * Calculate string similarity (simple implementation)
   */
  const calculateStringSimilarity = useCallback((str1: string, str2: string): number => {
    const longer = str1.length > str2.length ? str1 : str2;
    const shorter = str1.length > str2.length ? str2 : str1;

    if (longer.length === 0) return 1.0;

    const editDistance = levenshteinDistance(longer, shorter);
    return (longer.length - editDistance) / longer.length;
  }, [levenshteinDistance]);

  /**
   * Resolve a conflict using a specific strategy
   */
  const resolveConflict = useCallback(async (
    conflictId: string,
    strategyId: string,
    customValue?: unknown
  ): Promise<unknown> => {
    const conflict = state.activeConflicts.get(conflictId);
    if (!conflict) {
      throw new Error(`Conflict ${conflictId} not found`);
    }

    // Clear auto-resolve timeout
    const timeoutId = autoResolveTimeoutRef.current.get(conflictId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      autoResolveTimeoutRef.current.delete(conflictId);
    }

    const strategy = state.strategies.find(s => s.id === strategyId);
    if (!strategy) {
      throw new Error(`Strategy ${strategyId} not found`);
    }

    let resolvedValue: unknown;

    try {
      if (strategyId === 'custom' && customValue !== undefined) {
        resolvedValue = customValue;
      } else {
        resolvedValue = await strategy.handler(conflict);
      }

      // Move conflict to resolved
      setState(prev => {
        const newActiveConflicts = new Map(prev.activeConflicts);
        newActiveConflicts.delete(conflictId);

        const resolvedConflict = { ...conflict };
        const newResolvedConflicts = [resolvedConflict, ...prev.resolvedConflicts.slice(0, 99)]; // Keep last 100

        const newResolutionHistory = [
          {
            conflictId,
            strategy: strategyId,
            resolvedValue,
            timestamp: Date.now(),
          },
          ...prev.resolutionHistory.slice(0, 99), // Keep last 100
        ];

        return {
          ...prev,
          activeConflicts: newActiveConflicts,
          resolvedConflicts: newResolvedConflicts,
          resolutionHistory: newResolutionHistory,
        };
      });

      return resolvedValue;

    } catch (error) {
      console.error(`Failed to resolve conflict ${conflictId}:`, error);
      throw error;
    }
  }, [state.activeConflicts, state.strategies]);

  /**
   * Add a conflict to the resolution queue
   */
  const addConflict = useCallback((
    conflictId: string,
    tableId: string,
    rowId: string | number,
    column: string,
    localValue: unknown,
    remoteValue: unknown,
    baseValue: unknown,
    metadata?: Record<string, unknown>
  ) => {
    const conflictType = detectConflictType(localValue, remoteValue, baseValue);

    const conflict: ConflictData = {
      id: conflictId,
      tableId,
      rowId,
      column,
      localValue,
      remoteValue,
      baseValue,
      timestamp: Date.now(),
      conflictType,
      metadata,
    };

    setState(prev => ({
      ...prev,
      activeConflicts: new Map(prev.activeConflicts).set(conflictId, conflict),
    }));

    // Auto-resolve if enabled and strategy supports it
    if (state.autoResolveEnabled) {
      const strategy = state.strategies.find(s => s.id === state.defaultStrategy);
      if (strategy?.autoApply) {
        const timeoutId = setTimeout(() => {
          resolveConflict(conflictId, state.defaultStrategy);
        }, 2000); // Wait 2 seconds for manual intervention

        autoResolveTimeoutRef.current.set(conflictId, timeoutId);
      }
    }

    return conflict;
  }, [detectConflictType, state.autoResolveEnabled, state.defaultStrategy, state.strategies, resolveConflict]);

  /**
   * Cancel conflict resolution (remove from queue)
   */
  const cancelConflict = useCallback((conflictId: string) => {
    // Clear auto-resolve timeout
    const timeoutId = autoResolveTimeoutRef.current.get(conflictId);
    if (timeoutId) {
      clearTimeout(timeoutId);
      autoResolveTimeoutRef.current.delete(conflictId);
    }

    setState(prev => {
      const newActiveConflicts = new Map(prev.activeConflicts);
      newActiveConflicts.delete(conflictId);

      return {
        ...prev,
        activeConflicts: newActiveConflicts,
      };
    });
  }, []);

  /**
   * Add a custom resolution strategy
   */
  const addStrategy = useCallback((strategy: ConflictResolutionStrategy) => {
    setState(prev => ({
      ...prev,
      strategies: [...prev.strategies.filter(s => s.id !== strategy.id), strategy],
    }));
  }, []);

  /**
   * Remove a resolution strategy
   */
  const removeStrategy = useCallback((strategyId: string) => {
    setState(prev => ({
      ...prev,
      strategies: prev.strategies.filter(s => s.id !== strategyId),
    }));
  }, []);

  /**
   * Set default resolution strategy
   */
  const setDefaultStrategy = useCallback((strategyId: string) => {
    setState(prev => ({
      ...prev,
      defaultStrategy: strategyId,
    }));
  }, []);

  /**
   * Enable/disable auto-resolution
   */
  const setAutoResolveEnabled = useCallback((enabled: boolean) => {
    setState(prev => ({
      ...prev,
      autoResolveEnabled: enabled,
    }));
  }, []);

  /**
   * Get conflicts for a specific table
   */
  const getTableConflicts = useCallback((tableId: string): ConflictData[] => {
    return Array.from(state.activeConflicts.values())
      .filter(conflict => conflict.tableId === tableId);
  }, [state.activeConflicts]);

  /**
   * Get conflicts for a specific row
   */
  const getRowConflicts = useCallback((
    tableId: string,
    rowId: string | number
  ): ConflictData[] => {
    return getTableConflicts(tableId)
      .filter(conflict => conflict.rowId === rowId);
  }, [getTableConflicts]);

  /**
   * Check if a cell has conflicts
   */
  const hasConflicts = useCallback((
    tableId: string,
    rowId: string | number,
    column?: string
  ): boolean => {
    const rowConflicts = getRowConflicts(tableId, rowId);

    if (column) {
      return rowConflicts.some(conflict => conflict.column === column);
    }

    return rowConflicts.length > 0;
  }, [getRowConflicts]);

  /**
   * Get suggested resolution for a conflict
   */
  const getSuggestedResolution = useCallback((conflictId: string): {
    strategyId: string;
    value: unknown;
    confidence: number;
  } | null => {
    const conflict = state.activeConflicts.get(conflictId);
    if (!conflict) return null;

    // Simple heuristics for suggestion
    const { localValue, remoteValue, conflictType } = conflict;

    // Type conflicts usually need manual resolution
    if (conflictType === 'type') {
      return {
        strategyId: 'manual',
        value: remoteValue,
        confidence: 0.3,
      };
    }

    // For numbers, suggest averaging if values are close
    if (typeof localValue === 'number' && typeof remoteValue === 'number') {
      const diff = Math.abs(localValue - remoteValue);
      const average = (localValue + remoteValue) / 2;

      if (diff / average < 0.1) { // Less than 10% difference
        return {
          strategyId: 'merge_numeric',
          value: average,
          confidence: 0.8,
        };
      }
    }

    // For strings, suggest merging if not too different
    if (typeof localValue === 'string' && typeof remoteValue === 'string') {
      const similarity = calculateStringSimilarity(localValue, remoteValue);

      if (similarity > 0.5) {
        return {
          strategyId: 'merge_string',
          value: `${localValue} | ${remoteValue}`,
          confidence: similarity,
        };
      }
    }

    // Default to last write wins
    return {
      strategyId: 'last_write_wins',
      value: remoteValue,
      confidence: 0.6,
    };
  }, [state.activeConflicts, calculateStringSimilarity]);

  /**
   * Get resolution statistics
   */
  const getStats = useCallback(() => {
    const totalResolved = state.resolutionHistory.length;
    const strategyUsage = state.resolutionHistory.reduce((acc, resolution) => {
      acc[resolution.strategy] = (acc[resolution.strategy] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const averageResolutionTime = state.resolutionHistory.length > 0
      ? state.resolutionHistory.reduce((sum, resolution) => {
          const conflict = state.resolvedConflicts.find(c => c.id === resolution.conflictId);
          return sum + (conflict ? resolution.timestamp - conflict.timestamp : 0);
        }, 0) / state.resolutionHistory.length
      : 0;

    return {
      activeConflicts: state.activeConflicts.size,
      totalResolved,
      strategyUsage,
      averageResolutionTime,
      autoResolveEnabled: state.autoResolveEnabled,
      availableStrategies: state.strategies.length,
    };
  }, [state]);

  /**
   * Clear resolved conflicts history
   */
  const clearHistory = useCallback(() => {
    setState(prev => ({
      ...prev,
      resolvedConflicts: [],
      resolutionHistory: [],
    }));
  }, []);

  // Cleanup timeouts on unmount
  useEffect(() => {
    const timeoutMap = autoResolveTimeoutRef.current;
    return () => {
      Array.from(timeoutMap.values()).forEach(clearTimeout);
      timeoutMap.clear();
    };
  }, []);

  return {
    // State
    activeConflicts: Array.from(state.activeConflicts.values()),
    resolvedConflicts: state.resolvedConflicts,
    strategies: state.strategies,
    defaultStrategy: state.defaultStrategy,
    autoResolveEnabled: state.autoResolveEnabled,

    // Actions
    addConflict,
    resolveConflict,
    cancelConflict,
    addStrategy,
    removeStrategy,
    setDefaultStrategy,
    setAutoResolveEnabled,

    // Queries
    getTableConflicts,
    getRowConflicts,
    hasConflicts,
    getSuggestedResolution,

    // Utilities
    getStats,
    clearHistory,
  };
}