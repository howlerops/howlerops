import { useState, useCallback, useMemo, useRef, useEffect, type SetStateAction } from 'react';
import { SortingState, ColumnFiltersState } from '@tanstack/react-table';
import {
  TableState,
  TableAction,
  TableRow,
  CellValue,
  TableConfig,
  EditableTableActions,
} from '../types/table';
import { isEqual } from '../utils/table';

// Event-based table state management
class TableEventManager {
  private static instance: TableEventManager
  private listeners: Map<string, Set<(...args: unknown[]) => void>> = new Map()

  static getInstance(): TableEventManager {
    if (!TableEventManager.instance) {
      TableEventManager.instance = new TableEventManager()
    }
    return TableEventManager.instance
  }

  on(event: string, callback: (...args: unknown[]) => void) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  off(event: string, callback: (...args: unknown[]) => void) {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      eventListeners.delete(callback)
    }
  }

  emit(event: string, data?: unknown) {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      eventListeners.forEach(callback => {
        try {
          callback(data as never)
        } catch (error) {
          console.error('Error in table event listener:', error)
        }
      })
    }
  }
}

const generateRowId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

const assignRowIds = (rows: TableRow[]): TableRow[] =>
  rows.map((row, index) => {
    const id = row.__rowId ?? `${generateRowId()}-${index}`;
    if (row.__rowId === id) {
      return row;
    }
    return {
      ...row,
      __rowId: id,
    };
  });

const defaultConfig: TableConfig = {
  estimateSize: 35,
  overscan: 5,
  debounceMs: 300,
  maxUndoHistory: 50,
  autoSave: false,
  autoSaveInterval: 30000,
  enableVirtualization: true,
  enablePersistence: false,
};

export const useTableState = (
  initialData: TableRow[],
  config: Partial<TableConfig> = {}
) => {
  const mergedConfig = { ...defaultConfig, ...config };
  const eventManager = useRef(TableEventManager.getInstance());

  const [data, setDataState] = useState<TableRow[]>(assignRowIds(initialData));
  const setData = useCallback(
    (updater: SetStateAction<TableRow[]>) => {
      setDataState((prev) => {
        const next = typeof updater === 'function' ? (updater as (prev: TableRow[]) => TableRow[])(prev) : updater;
        return assignRowIds(next);
      });
    },
    []
  );
  const [state, setState] = useState<TableState>({
    editingCell: null,
    selectedRows: [],
    sorting: [],
    columnFilters: [],
    globalFilter: '',
    columnVisibility: {},
    columnOrder: [],
    columnSizing: {},
    dirtyRows: new Set(),
    invalidCells: new Map(),
    undoStack: [],
    redoStack: [],
  });

  const dataRef = useRef(data);
  
  // Update ref in effect (not during render)
  useEffect(() => {
    dataRef.current = data;
  }, [data]);

  const addToUndoStack = useCallback((action: TableAction) => {
    setState(prev => {
      const newUndoStack = [...prev.undoStack, action];
      if (newUndoStack.length > mergedConfig.maxUndoHistory) {
        newUndoStack.shift();
      }
      return {
        ...prev,
        undoStack: newUndoStack,
        redoStack: [], // Clear redo stack when new action is performed
      };
    });
  }, [mergedConfig.maxUndoHistory]);

  const updateCell = useCallback((
    rowId: string,
    columnId: string,
    newValue: CellValue,
    addToHistory = true
  ) => {
    const rowIndex = dataRef.current.findIndex(row => row.__rowId === rowId);
    if (rowIndex === -1) return false;

    const oldValue = dataRef.current[rowIndex][columnId];

    if (isEqual(oldValue, newValue)) {
      return true;
    }

    setData(prevData => {
      const newData = [...prevData];
      newData[rowIndex] = { ...newData[rowIndex], [columnId]: newValue };
      return newData;
    });

    setState(prev => ({
      ...prev,
      dirtyRows: new Set([...prev.dirtyRows, rowId]),
      editingCell: null,
    }));

    if (addToHistory) {
      addToUndoStack({
        type: 'edit',
        payload: { rowId, columnId, oldValue, newValue },
        timestamp: Date.now(),
      });
    }

    return true;
  }, [addToUndoStack, setData]);

  const startEditing = useCallback((
    rowId: string,
    columnId: string,
    value: CellValue
  ) => {
    setState(prev => ({
      ...prev,
      editingCell: {
        rowId,
        columnId,
        value,
        originalValue: value,
        isValid: true,
      },
    }));
  }, []);

  const updateEditingCell = useCallback((
    value: CellValue,
    isValid: boolean,
    error?: string
  ) => {
    setState(prev => {
      if (!prev.editingCell) return prev;
      const { rowId, columnId } = prev.editingCell;

      let nextInvalidCells = prev.invalidCells;
      let invalidChanged = false;

      if (rowId && columnId) {
        const cellKey = `${rowId}|${columnId}`;
        const existingInvalid = prev.invalidCells.get(cellKey);

        if (!isValid && error) {
          if (!existingInvalid || existingInvalid.error !== error) {
            nextInvalidCells = new Map(prev.invalidCells);
            nextInvalidCells.set(cellKey, { columnId, error });
            invalidChanged = true;
          }
        } else if (existingInvalid) {
          nextInvalidCells = new Map(prev.invalidCells);
          nextInvalidCells.delete(cellKey);
          invalidChanged = true;
        }
      }

      const valueChanged = !isEqual(prev.editingCell.value, value);
      const validityChanged = prev.editingCell.isValid !== isValid;
      const errorChanged = prev.editingCell.error !== error;

      if (!valueChanged && !validityChanged && !errorChanged && !invalidChanged) {
        return prev;
      }

      return {
        ...prev,
        editingCell: {
          ...prev.editingCell,
          value,
          isValid,
          error,
        },
        invalidCells: nextInvalidCells,
      };
    });
  }, []);

  const cancelEditing = useCallback(() => {
    setState(prev => ({
      ...prev,
      editingCell: null,
    }));
  }, []);

  const saveEditing = useCallback(async (): Promise<boolean> => {
    if (!state.editingCell || !state.editingCell.isValid) {
      return false;
    }

    const { rowId, columnId, value } = state.editingCell;
    return updateCell(rowId, columnId, value);
  }, [state.editingCell, updateCell]);

  const toggleRowSelection = useCallback((rowId: string, selected?: boolean) => {
    setState(prev => {
      const isSelected = prev.selectedRows.includes(rowId);
      const shouldSelect = selected !== undefined ? selected : !isSelected;

      if (shouldSelect && !isSelected) {
        return {
          ...prev,
          selectedRows: [...prev.selectedRows, rowId],
        };
      } else if (!shouldSelect && isSelected) {
        return {
          ...prev,
          selectedRows: prev.selectedRows.filter(id => id !== rowId),
        };
      }
      return prev;
    });
  }, []);

  const selectAllRows = useCallback((selected: boolean) => {
    setState(prev => ({
      ...prev,
      selectedRows: selected
        ? dataRef.current
            .map(row => row.__rowId)
            .filter((id): id is string => Boolean(id))
        : [],
    }));
  }, []);

  const updateSorting = useCallback((sorting: SortingState) => {
    setState(prev => ({ ...prev, sorting }));
  }, []);

  const updateColumnFilters = useCallback((columnFilters: ColumnFiltersState) => {
    setState(prev => ({ ...prev, columnFilters }));
  }, []);

  const updateGlobalFilter = useCallback((globalFilter: string) => {
    setState(prev => ({ ...prev, globalFilter }));
  }, []);

  const updateColumnVisibility = useCallback((columnVisibility: Record<string, boolean>) => {
    setState(prev => ({ ...prev, columnVisibility }));
  }, []);

  const updateColumnSizing = useCallback((columnSizing: Record<string, number>) => {
    setState(prev => ({ ...prev, columnSizing }));
  }, []);

  const updateColumnOrder = useCallback((columnOrder: string[]) => {
    setState(prev => ({ ...prev, columnOrder }));
  }, []);

  const undo = useCallback(() => {
    setState(prev => {
      if (prev.undoStack.length === 0) return prev;

      const action = prev.undoStack[prev.undoStack.length - 1];
      const newUndoStack = prev.undoStack.slice(0, -1);

      // Perform the undo operation
      if (action.type === 'edit' && action.payload.rowId && action.payload.columnId) {
        updateCell(
          action.payload.rowId,
          action.payload.columnId,
          action.payload.oldValue,
          false
        );
      }

      return {
        ...prev,
        undoStack: newUndoStack,
        redoStack: [...prev.redoStack, action],
      };
    });
  }, [updateCell]);

  const redo = useCallback(() => {
    setState(prev => {
      if (prev.redoStack.length === 0) return prev;

      const action = prev.redoStack[prev.redoStack.length - 1];
      const newRedoStack = prev.redoStack.slice(0, -1);

      // Perform the redo operation
      if (action.type === 'edit' && action.payload.rowId && action.payload.columnId) {
        updateCell(
          action.payload.rowId,
          action.payload.columnId,
          action.payload.newValue,
          false
        );
      }

      return {
        ...prev,
        redoStack: newRedoStack,
        undoStack: [...prev.undoStack, action],
      };
    });
  }, [updateCell]);

  const clearDirtyRows = useCallback(() => {
    setState(prev => ({ ...prev, dirtyRows: new Set() }));
  }, []);

  const getInvalidCells = useCallback(() => {
    const invalidCellsArray: Array<{ rowId: string; columnId: string; error: string }> = [];
    state.invalidCells.forEach((errorInfo, cellKey) => {
      const [rowId, columnId] = cellKey.split('|');
      invalidCellsArray.push({ rowId, columnId, error: errorInfo.error });
    });
    return invalidCellsArray;
  }, [state.invalidCells]);

  const validateAllCells = useCallback(() => {
    // Clear existing invalid cells
    setState(prev => ({ ...prev, invalidCells: new Map() }));
    
    // Validate all dirty cells
    const newInvalidCells = new Map<string, { columnId: string; error: string }>();
    
    state.dirtyRows.forEach(rowId => {
      const rowIndex = dataRef.current.findIndex(row => row.__rowId === rowId);
      if (rowIndex === -1) return;
      
      const row = dataRef.current[rowIndex];
      // For now, we'll do basic validation - this can be enhanced later
      // with proper column validation rules
      Object.keys(row).forEach(columnId => {
        if (columnId === '__rowId') return;
        
        const value = row[columnId];
        // Basic validation - can be enhanced with column-specific rules
        if (value === null || value === undefined || value === '') {
          // This is a placeholder - real validation should check column rules
          // For now, we'll consider empty values as valid
        }
      });
    });
    
    setState(prev => ({ ...prev, invalidCells: newInvalidCells }));
    return newInvalidCells.size === 0;
  }, [state.dirtyRows]);

  const clearInvalidCells = useCallback(() => {
    setState(prev => ({ ...prev, invalidCells: new Map() }));
  }, []);

  const trackValidationError = useCallback((
    rowId: string,
    columnId: string,
    error: string
  ) => {
    const cellKey = `${rowId}|${columnId}`;
    setState(prev => {
      const newInvalidCells = new Map(prev.invalidCells);
      newInvalidCells.set(cellKey, { columnId, error });
      return { ...prev, invalidCells: newInvalidCells };
    });
  }, []);

  const clearValidationError = useCallback((
    rowId: string,
    columnId: string
  ) => {
    const cellKey = `${rowId}|${columnId}`;
    setState(prev => {
      const newInvalidCells = new Map(prev.invalidCells);
      newInvalidCells.delete(cellKey);
      return { ...prev, invalidCells: newInvalidCells };
    });
  }, []);

  const resetTable = useCallback(() => {
    setData(assignRowIds(initialData));
    setState({
      editingCell: null,
      selectedRows: [],
      sorting: [],
      columnFilters: [],
      globalFilter: '',
      columnVisibility: {},
      columnOrder: [],
      columnSizing: {},
      dirtyRows: new Set(),
      invalidCells: new Map(),
      undoStack: [],
      redoStack: [],
    });
  }, [initialData, setData]);

  // Event-based state clearing
  useEffect(() => {
    const handleTableReset = () => {
      resetTable();
    };

    const handleDataRefresh = (...args: unknown[]) => {
      const newData = args[0] as TableRow[];
      setData(assignRowIds(newData));
      setState(prev => ({
        ...prev,
        dirtyRows: new Set(),
        invalidCells: new Map(),
        undoStack: [],
        redoStack: [],
      }));
    };

    const handleQueryChange = () => {
      setState(prev => ({
        ...prev,
        editingCell: null,
        selectedRows: [],
        dirtyRows: new Set(),
        invalidCells: new Map(),
        undoStack: [],
        redoStack: [],
      }));
    };

    const manager = eventManager.current;
    manager.on('table:reset', handleTableReset);
    manager.on('table:data:refresh', handleDataRefresh);
    manager.on('query:change', handleQueryChange);

    return () => {
      manager.off('table:reset', handleTableReset);
      manager.off('table:data:refresh', handleDataRefresh);
      manager.off('query:change', handleQueryChange);
    };
  }, [resetTable, setData]);

  const tableActions = useMemo<EditableTableActions>(() => ({
    updateCell,
    startEditing,
    updateEditingCell,
    cancelEditing,
    saveEditing,
    toggleRowSelection,
    selectAllRows,
    updateSorting,
    updateColumnFilters,
    updateGlobalFilter,
    updateColumnVisibility,
    updateColumnSizing,
    updateColumnOrder,
    undo,
    redo,
    clearDirtyRows,
    resetTable,
    getInvalidCells,
    validateAllCells,
    clearInvalidCells,
    trackValidationError,
    clearValidationError,
  }), [
    updateCell,
    startEditing,
    updateEditingCell,
    cancelEditing,
    saveEditing,
    toggleRowSelection,
    selectAllRows,
    updateSorting,
    updateColumnFilters,
    updateGlobalFilter,
    updateColumnVisibility,
    updateColumnSizing,
    updateColumnOrder,
    undo,
    redo,
    clearDirtyRows,
    resetTable,
    getInvalidCells,
    validateAllCells,
    clearInvalidCells,
    trackValidationError,
    clearValidationError,
  ]);

  const computedState = useMemo(() => ({
    ...state,
    hasUndoActions: state.undoStack.length > 0,
    hasRedoActions: state.redoStack.length > 0,
    hasSelection: state.selectedRows.length > 0,
    hasDirtyRows: state.dirtyRows.size > 0,
    hasInvalidCells: state.invalidCells.size > 0,
    isEditing: state.editingCell !== null,
  }), [state]);

  // Safe getter for event manager (avoids ref access during render)
  const getEventManager = useCallback(() => eventManager.current, []);

  return {
    data,
    setData,
    state: computedState,
    actions: tableActions,
    config: mergedConfig,
    getEventManager,
  };
};

// Utility function to emit table events
export const emitTableEvent = (event: string, data?: unknown) => {
  TableEventManager.getInstance().emit(event, data);
};
