import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';

import {
  AllCommunityModule,
  type CellValueChangedEvent,
  type ColDef,
  type ColumnMovedEvent,
  type ColumnResizedEvent,
  type ColumnVisibleEvent,
  type GetRowIdParams,
  type GridApi,
  type GridReadyEvent,
  ModuleRegistry,
  type RowClickedEvent,
  type SelectionColumnDef,
  type SortChangedEvent,
} from 'ag-grid-community';
import { AgGridReact } from 'ag-grid-react';
import { Eye } from 'lucide-react';

// Register AG Grid Community modules
ModuleRegistry.registerModules([AllCommunityModule]);
import './ag-grid-table.css';

import { type ColumnFiltersState, type SortingState } from '@tanstack/react-table';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  CellValue,
  EditableTableContext,
  EditableTableProps,
  EditableTableRenderer,
  TableColumn,
  TableRow,
} from '../../types/table';
import { cn } from '../../utils/cn';

/**
 * AG Grid-based Table Component
 *
 * Drop-in replacement for EditableTable using AG Grid Community for:
 * - Better virtualization (no white chunks during fast scrolling)
 * - Built-in column stability
 * - Less maintenance burden
 *
 * @component
 */
export const AGGridTable: React.FC<EditableTableProps> = ({
  data,
  columns: tableColumns,
  onDataChange,
  onCellEdit: _onCellEdit,
  onRowSelect,
  onRowClick,
  onRowInspect,
  onSort,
  onFilter,
  onExport: _onExport,
  onSelectAllPages,
  loading = false,
  error = null,
  virtualScrolling: _virtualScrolling = true,
  className,
  height = 600,
  enableMultiSelect = true,
  enableColumnResizing = true,
  enableColumnReordering = false,
  enableGlobalFilter = true,
  enableExport: _enableExport = true,
  toolbar,
  footer,
  onDirtyChange,
  customCellRenderers = {},
  isEditable = false,
  // Phase 2: Chunked data loading
  resultId: _resultId,
  totalRows: _totalRows,
  isLargeResult: _isLargeResult = false,
  chunkingEnabled: _chunkingEnabled = false,
  displayMode: _displayMode,
}) => {
  const gridRef = useRef<AgGridReact>(null);
  const [gridApi, setGridApi] = useState<GridApi | null>(null);

  // Ref to track current data for efficient updates (avoids iterating all grid nodes)
  const dataRef = useRef<TableRow[]>(data);
  useEffect(() => {
    dataRef.current = data;
  }, [data]);

  // Track dirty rows (edited but not saved)
  const [dirtyRows, setDirtyRows] = useState<Set<string>>(new Set());

  // Track selected rows
  const [selectedRows, setSelectedRows] = useState<string[]>([]);

  // Track sorting state
  const [sorting, setSorting] = useState<SortingState>([]);

  // Track filters
  const [filters, setFilters] = useState<ColumnFiltersState>([]);

  // Track global filter
  const [globalFilter, setGlobalFilter] = useState('');

  // Track select all pages mode
  const [selectAllPagesMode, setSelectAllPagesMode] = useState(false);

  // Track column state
  const [columnOrder, setColumnOrder] = useState<string[]>([]);
  const [columnVisibility, setColumnVisibility] = useState<Record<string, boolean>>({});
  const [columnSizing, setColumnSizing] = useState<Record<string, number>>({});

  /**
   * Map TableColumn type to AG Grid column type
   */
  const mapColumnType = (type: TableColumn['type']): string => {
    switch (type) {
      case 'number':
        return 'numericColumn';
      case 'date':
      case 'datetime':
        return 'dateColumn';
      case 'boolean':
        return 'booleanColumn';
      default:
        return 'textColumn';
    }
  };

  /**
   * Create cell renderer for custom types
   * Eye icon appears on hover at the end of each cell for row inspection
   */
  const createCellRenderer = (column: TableColumn) => {
    return (params: { value: CellValue; data: TableRow }) => {
      const { value, data: rowData } = params;

      // Use custom renderer if provided
      const customRenderer = customCellRenderers[column.id || column.accessorKey || ''];
      if (customRenderer) {
        return customRenderer(value, rowData);
      }

      // Eye icon component - appears on hover in every cell
      const eyeIcon = onRowInspect ? (
        <button
          className="inspect-row-btn opacity-0 group-hover:opacity-100 ml-1 p-0.5 rounded hover:bg-accent transition-opacity flex-shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            const rowId = rowData.__rowId;
            if (rowId && onRowInspect) {
              onRowInspect(rowId, rowData);
            }
          }}
          title="View row details"
        >
          <Eye className="h-3 w-3 text-muted-foreground hover:text-foreground" />
        </button>
      ) : null;

      // Boolean renderer - uses native checkbox with proper accessibility
      // Note: We use a custom renderer instead of agCheckboxCellRenderer to:
      // 1. Include the eye icon for row inspection (consistent with other columns)
      // 2. Have full control over styling and interaction
      if (column.type === 'boolean') {
        const isDisabled = !column.editable || !isEditable;
        const checkboxId = `bool-${rowData.__rowId}-${column.accessorKey || column.id}`;

        const handleBooleanChange = (e: React.ChangeEvent<HTMLInputElement>) => {
          e.stopPropagation();
          if (isDisabled) return;

          const newValue = e.target.checked;
          // Get the grid API from the ref and update the cell value directly
          const api = gridRef.current?.api;
          if (api && rowData.__rowId) {
            const rowNode = api.getRowNode(rowData.__rowId);
            if (rowNode) {
              const field = column.accessorKey || column.id || column.header;
              rowNode.setDataValue(field, newValue);
            }
          }
        };

        return (
          <div className="group flex items-center justify-between h-full w-full">
            <div className="flex items-center justify-center flex-1">
              <input
                id={checkboxId}
                type="checkbox"
                checked={Boolean(value)}
                onChange={handleBooleanChange}
                disabled={isDisabled}
                aria-label={`${column.header}: ${value ? 'checked' : 'unchecked'}`}
                aria-disabled={isDisabled}
                className="w-4 h-4 cursor-pointer rounded border-gray-400 text-primary focus:ring-2 focus:ring-primary focus:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            {eyeIcon}
          </div>
        );
      }

      // Select renderer
      if (column.type === 'select' && column.options) {
        return (
          <div className="group flex items-center h-full w-full">
            <select
              value={String(value || '')}
              disabled={!column.editable}
              className="flex-1 h-full bg-transparent border-none outline-none"
            >
              <option value="">Select...</option>
              {column.options.map(opt => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
            {eyeIcon}
          </div>
        );
      }

      // Default text renderer with monospace support
      // Always truncate to maintain consistent row height
      const textClasses = cn(
        'flex-1 truncate overflow-hidden whitespace-nowrap',
        column.monospace && 'font-mono'
      );

      return (
        <div className="group flex items-center h-full w-full">
          <div className={textClasses} title={String(value ?? '')}>
            {value === null || value === undefined ? (
              <span className="cell-null">NULL</span>
            ) : (
              String(value)
            )}
          </div>
          {eyeIcon}
        </div>
      );
    };
  };

  /**
   * Create cell editor for custom types
   */
  const createCellEditor = (column: TableColumn) => {
    if (!column.editable) return undefined;

    if (column.type === 'boolean') {
      return 'agCheckboxCellEditor';
    }

    if (column.type === 'select' && column.options) {
      return 'agSelectCellEditor';
    }

    if (column.type === 'number') {
      return 'agNumberCellEditor';
    }

    if (column.type === 'date' || column.type === 'datetime') {
      return 'agDateCellEditor';
    }

    return 'agTextCellEditor';
  };

  /**
   * Create cell editor params
   */
  const createCellEditorParams = (column: TableColumn) => {
    if (column.type === 'select' && column.options) {
      return {
        values: column.options,
      };
    }

    if (column.type === 'number') {
      return {
        min: column.validation?.min,
        max: column.validation?.max,
      };
    }

    return undefined;
  };

  /**
   * Convert TableColumn[] to AG Grid ColDef[]
   * CRITICAL: Do NOT include dirtyRows in dependencies - causes flicker on state changes
   */
  const columnDefs = useMemo<ColDef[]>(() => {
    const cols: ColDef[] = [];

    // Note: Checkbox column is now handled automatically by AG Grid v34's
    // rowSelection object API with { checkboxes: true }

    // Add data columns
    tableColumns.forEach((col, index) => {
      // Phase 1: Normalize to single source of truth - field and colId must match
      const field = col.accessorKey || col.id || col.header;
      const colId = field; // ALWAYS match field

      // Add dev-mode validation (only log once per column)
      if (process.env.NODE_ENV === 'development' && data.length > 0 && index === 0) {
        const sampleRow = data[0];
        if (field && !(field in sampleRow)) {
          console.warn(`[AG Grid] Column field "${field}" not found in row data. Available keys:`, Object.keys(sampleRow));
        }
      }

      // Determine optimal width based on column type
      // Strategy: Use flex for proportional sizing instead of fixed widths
      // - UUID/ID columns: fixed width (no flex) with reasonable max
      // - Number columns: narrow fixed width
      // - Text columns: flex to fill remaining space
      const isUuidColumn = field?.toLowerCase().includes('id') || field?.toLowerCase().includes('uuid');
      const isNumericColumn = col.type === 'number';
      const isLongText = col.longText || col.type === 'text' && (col.maxWidth ?? 0) > 400;

      // Width sizing strategy per column type
      let columnWidth: number | undefined;
      let columnFlex: number | undefined;
      let columnMinWidth: number;
      let columnMaxWidth: number | undefined;

      if (isUuidColumn) {
        // UUID columns: fixed reasonable width, no flex
        columnWidth = 300;
        columnMinWidth = 250;
        columnMaxWidth = 350;
        columnFlex = undefined; // No flex - fixed width
      } else if (isNumericColumn) {
        // Numeric columns: narrow fixed width
        columnWidth = 120;
        columnMinWidth = 100;
        columnMaxWidth = 150;
        columnFlex = undefined; // No flex - fixed width
      } else if (isLongText) {
        // Long text: flex with constraints
        columnWidth = undefined;
        columnFlex = 2; // Get more space than regular text
        columnMinWidth = 200;
        columnMaxWidth = 500;
      } else {
        // Regular text: flex proportionally
        columnWidth = undefined;
        columnFlex = 1;
        columnMinWidth = col.minWidth || 100;
        columnMaxWidth = col.maxWidth;
      }

      // Allow column config to override defaults
      if (col.width !== undefined) {
        columnWidth = col.width;
        columnFlex = undefined;
      }

      // Check if column has editable capability
      // We'll use a function for the editable property to access isEditable dynamically
      const columnCanBeEditable = !!col.editable;

      // Determine if this column should use popup editor (for text types that are editable)
      const isTextType = col.type === 'text' || col.type === undefined || col.longText;
      const usePopupEditor = columnCanBeEditable && isTextType;

      // Static cell classes - determined once at definition time to prevent re-renders
      const staticCellClasses: string[] = [];
      if (col.isPrimaryKey) {
        staticCellClasses.push('ag-cell-primary-key');
      }

      cols.push({
        field,
        headerName: col.header,
        colId,
        type: mapColumnType(col.type),
        // Column sizing: use width/flex strategy calculated above
        width: columnWidth,
        flex: columnFlex,
        minWidth: columnMinWidth,
        maxWidth: columnMaxWidth,
        sortable: col.sortable !== false,
        filter: col.filterable !== false,
        // CRITICAL: Use function to dynamically check isEditable at edit-time
        // This prevents column regeneration when global editable state changes
        editable: columnCanBeEditable ? () => isEditable : false,
        resizable: enableColumnResizing,
        initialPinned: col.sticky || undefined,
        // Never wrap text or auto-height - always truncate to maintain consistent row height
        wrapText: false,
        autoHeight: false,
        cellRenderer: createCellRenderer(col),
        // Only set editor for columns that can be editable
        cellEditor: columnCanBeEditable
          ? (usePopupEditor ? 'agLargeTextCellEditor' : createCellEditor(col))
          : undefined,
        cellEditorPopup: usePopupEditor,
        cellEditorParams: columnCanBeEditable
          ? (usePopupEditor ? { maxLength: 10000, rows: 6, cols: 40 } : createCellEditorParams(col))
          : undefined,
        // Use static classes only - no dynamic state to prevent flickering
        cellClass: staticCellClasses.length > 0 ? staticCellClasses : undefined,
        // Use cellClassRules for dynamic state instead - more efficient
        cellClassRules: {
          'ag-cell-dirty': (params) => {
            // Access dirtyRows from outer scope at render time, not definition time
            return !!(params.data?.__rowId && dirtyRows.has(params.data.__rowId));
          },
          'ag-row-new': (params) => {
            return !!params.data?.__isNewRow;
          },
        },
        valueGetter: col.type === 'boolean'
          ? (params) => Boolean(params.data?.[params.colDef.field || ''])
          : undefined,
        // Parse string input to correct type before valueSetter
        valueParser: (params) => {
          const value = params.newValue;

          // Parse number columns
          if (col.type === 'number') {
            if (value === null || value === undefined || value === '') {
              return null;
            }
            const parsed = parseFloat(value);
            if (isNaN(parsed)) {
              console.warn('Invalid number input:', value);
              return params.oldValue; // Revert to old value on parse error
            }
            return parsed;
          }

          // Parse boolean columns
          if (col.type === 'boolean') {
            if (typeof value === 'boolean') return value;
            if (typeof value === 'string') {
              return value.toLowerCase() === 'true';
            }
            return Boolean(value);
          }

          // Return as-is for other types
          return value;
        },
        valueSetter: (params) => {
          const field = params.colDef.field;
          if (!field) return false;

          let newValue: CellValue = params.newValue;

          // Type conversion (already done by valueParser, but keep for safety)
          if (col.type === 'number') {
            if (typeof newValue !== 'number') {
              newValue = newValue === null || newValue === '' ? null : Number(newValue);
            }
          } else if (col.type === 'boolean') {
            if (typeof newValue !== 'boolean') {
              newValue = Boolean(newValue);
            }
          }

          // Validation
          if (col.validation) {
            if (col.validation.pattern && typeof newValue === 'string') {
              if (!col.validation.pattern.test(newValue)) {
                console.warn(`Validation failed: ${col.validation.message || 'Invalid value'}`);
                return false;
              }
            }

            if (typeof newValue === 'number') {
              if (col.validation.min !== undefined && newValue < col.validation.min) {
                console.warn(`Value must be >= ${col.validation.min}`);
                return false;
              }
              if (col.validation.max !== undefined && newValue > col.validation.max) {
                console.warn(`Value must be <= ${col.validation.max}`);
                return false;
              }
            }
          }

          params.data[field] = newValue;
          return true;
        },
      });
    });

    return cols;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- INTENTIONAL: See comments below
  }, [tableColumns, enableColumnResizing, customCellRenderers, onRowInspect]);
  // CRITICAL: Do NOT include isEditable in dependencies!
  // Column editability is determined at render time via the editable property
  // which references the outer isEditable variable dynamically
  // This prevents column regeneration when global editable state changes
  // IMPORTANT: Do NOT add dirtyRows to dependencies - causes flicker
  // We use cellClassRules which accesses dirtyRows dynamically

  /**
   * Memoize row data to prevent unnecessary re-renders
   * CRITICAL: Only update when data array reference changes
   */
  const memoizedRowData = useMemo(() => data, [data]);

  /**
   * Get row ID for AG Grid
   * IMPORTANT: Rows MUST have __rowId or id for stable identification
   * Content-based fallback is safer than Math.random() which changes every render
   */
  const getRowId = useCallback((params: GetRowIdParams<TableRow>): string => {
    // Prefer __rowId (our internal ID) or id (database primary key)
    if (params.data.__rowId) return params.data.__rowId;
    if (params.data.id) return String(params.data.id);

    // Fallback: create stable ID from first few column values
    // This ensures same data produces same ID across renders
    const keys = Object.keys(params.data).filter(k => !k.startsWith('__')).slice(0, 3);
    const fallbackId = keys.map(k => String(params.data[k] ?? '')).join('-');
    if (fallbackId) {
      console.warn('Row missing __rowId and id, using content-based fallback:', fallbackId);
      return `fallback-${fallbackId}`;
    }

    // Last resort: use stringified data hash (still better than random)
    const dataHash = JSON.stringify(params.data).substring(0, 50);
    console.warn('Row missing __rowId, id, and has no key columns, using data hash');
    return `hash-${btoa(dataHash).substring(0, 20)}`;
  }, []);

  /**
   * Handle grid ready
   */
  const onGridReady = useCallback((params: GridReadyEvent) => {
    setGridApi(params.api);
  }, []);

  /**
   * Handle cell value changed (editing)
   * NOTE: This now ONLY updates local state and marks rows dirty
   * Actual save happens when user clicks "Save Changes" button
   * OPTIMIZED: Uses dataRef to update specific row instead of iterating all nodes
   */
  const onCellValueChanged = useCallback((event: CellValueChangedEvent<TableRow>) => {
    const rowId = event.data.__rowId;
    const columnId = event.colDef.field;

    if (!rowId || !columnId) return;

    // Mark row as dirty (but don't save yet)
    setDirtyRows(prev => new Set(prev).add(rowId));

    // Refresh the cell to update custom renderers (especially important for boolean checkboxes)
    if (gridApi && event.node) {
      gridApi.refreshCells({
        rowNodes: [event.node],
        columns: [columnId],
        force: true,
      });
    }

    // Call onDataChange with updated data array
    // OPTIMIZATION: Update specific row in dataRef instead of iterating all grid nodes
    if (onDataChange) {
      const updatedData = dataRef.current.map(row =>
        row.__rowId === rowId ? { ...row, ...event.data } : row
      );
      dataRef.current = updatedData;
      onDataChange(updatedData);
    }
  }, [onDataChange, gridApi]);

  /**
   * Handle row selection
   */
  const onSelectionChanged = useCallback(() => {
    if (!gridApi) return;

    const selectedNodes = gridApi.getSelectedNodes();
    const selectedRowIds = selectedNodes
      .map(node => node.data?.__rowId)
      .filter((id): id is string => Boolean(id));

    setSelectedRows(selectedRowIds);

    if (onRowSelect) {
      onRowSelect(selectedRowIds);
    }
  }, [gridApi, onRowSelect]);

  /**
   * Handle row click
   */
  const onRowClicked = useCallback((event: RowClickedEvent) => {
    const rowId = event.data?.__rowId;
    if (rowId && onRowClick) {
      onRowClick(rowId, event.data);
    }
  }, [onRowClick]);

  // Note: Row double-click handler removed - use eye icon in first column to open sidebar
  // This allows inline editing to work properly on double-click

  /**
   * Handle sort changed
   */
  const onSortChanged = useCallback((_event: SortChangedEvent) => {
    if (!gridApi || !onSort) return;

    const sortModel = gridApi.getColumnState()
      .filter(col => col.sort)
      .map(col => ({
        id: col.colId,
        desc: col.sort === 'desc',
      }));

    setSorting(sortModel);
    onSort(sortModel);
  }, [gridApi, onSort]);

  /**
   * Handle filter changed
   */
  const onFilterChanged = useCallback(() => {
    if (!gridApi || !onFilter) return;

    const filterModel = gridApi.getFilterModel();
    const filters: ColumnFiltersState = Object.keys(filterModel).map(key => ({
      id: key,
      value: filterModel[key],
    }));

    setFilters(filters);
    onFilter(filters);
  }, [gridApi, onFilter]);

  /**
   * Handle column moved (reordering)
   * Syncs column order changes to application state
   */
  const onColumnMoved = useCallback((event: ColumnMovedEvent) => {
    if (event.finished && event.api) {
      const newOrder = event.api.getAllDisplayedColumns()
        .map(col => col.getColId())
        .filter(Boolean) as string[];

      setColumnOrder(newOrder);
    }
  }, []);

  /**
   * Handle column visibility changed
   * Syncs column show/hide to application state
   */
  const onColumnVisible = useCallback((event: ColumnVisibleEvent) => {
    if (event.api) {
      const visibilityState: Record<string, boolean> = {};
      event.api.getAllGridColumns().forEach(col => {
        visibilityState[col.getColId()] = col.isVisible();
      });

      setColumnVisibility(visibilityState);
    }
  }, []);

  /**
   * Handle column resized
   * Syncs column width changes to application state
   */
  const onColumnResized = useCallback((event: ColumnResizedEvent) => {
    if (event.finished && event.api) {
      const sizingState: Record<string, number> = {};
      event.api.getAllDisplayedColumns().forEach(col => {
        sizingState[col.getColId()] = col.getActualWidth();
      });

      setColumnSizing(sizingState);
    }
  }, []);

  /**
   * Apply global filter
   */
  useEffect(() => {
    if (gridApi && enableGlobalFilter) {
      gridApi.setGridOption('quickFilterText', globalFilter);
    }
  }, [gridApi, globalFilter, enableGlobalFilter]);

  /**
   * Notify parent of dirty changes
   */
  useEffect(() => {
    if (onDirtyChange) {
      onDirtyChange(Array.from(dirtyRows));
    }
  }, [dirtyRows, onDirtyChange]);

  /**
   * Create table context for toolbar/footer renderers
   */
  const tableContext = useMemo<EditableTableContext>(() => ({
    data,
    state: {
      editingCell: null,
      selectedRows,
      selectAllPagesMode,
      sorting,
      columnFilters: filters,
      globalFilter,
      columnVisibility,
      columnOrder,
      columnSizing,
      dirtyRows,
      invalidCells: new Map(),
      undoStack: [],
      redoStack: [],
      hasUndoActions: false,
      hasRedoActions: false,
      hasSelection: selectedRows.length > 0,
      hasDirtyRows: dirtyRows.size > 0,
      hasInvalidCells: false,
      isEditing: false,
    },
    actions: {
      updateCell: () => false,
      startEditing: () => {},
      updateEditingCell: () => {},
      cancelEditing: () => {},
      saveEditing: async () => false,
      toggleRowSelection: (rowId: string, selected?: boolean) => {
        if (!gridApi) return;
        gridApi.forEachNode(node => {
          if (node.data?.__rowId === rowId) {
            node.setSelected(selected ?? !node.isSelected());
          }
        });
      },
      selectAllRows: (selected: boolean) => {
        if (!gridApi) return;
        if (selected) {
          gridApi.selectAll();
        } else {
          gridApi.deselectAll();
        }
      },
      setSelectedRows: (rowIds: string[]) => {
        if (!gridApi) return;
        gridApi.forEachNode(node => {
          const rowId = node.data?.__rowId;
          node.setSelected(rowId ? rowIds.includes(rowId) : false);
        });
      },
      setSelectAllPagesMode: (enabled: boolean) => {
        setSelectAllPagesMode(enabled);
        if (enabled && onSelectAllPages) {
          onSelectAllPages();
        }
      },
      updateSorting: (newSorting: SortingState) => {
        setSorting(newSorting);
        if (onSort) onSort(newSorting);
      },
      updateColumnFilters: (newFilters: ColumnFiltersState) => {
        setFilters(newFilters);
        if (onFilter) onFilter(newFilters);
      },
      updateGlobalFilter: (filter: string) => {
        setGlobalFilter(filter);
      },
      updateColumnVisibility: (visibility: Record<string, boolean>) => {
        setColumnVisibility(visibility);
      },
      updateColumnSizing: (sizing: Record<string, number>) => {
        setColumnSizing(sizing);
      },
      updateColumnOrder: (order: string[]) => {
        setColumnOrder(order);
      },
      undo: () => {},
      redo: () => {},
      clearDirtyRows: () => setDirtyRows(new Set()),
      resetTable: () => {
        if (!gridApi) return;
        gridApi.deselectAll();
        gridApi.setFilterModel(null);
        gridApi.setGridOption('columnDefs', columnDefs); // Reset sort via column defs
        setDirtyRows(new Set());
      },
      getInvalidCells: () => [],
      validateAllCells: () => true,
      clearInvalidCells: () => {},
      trackValidationError: () => {},
      clearValidationError: () => {},
    },
  }), [
    data,
    selectedRows,
    selectAllPagesMode,
    sorting,
    filters,
    globalFilter,
    columnOrder,
    columnVisibility,
    columnSizing,
    dirtyRows,
    gridApi,
    columnDefs,
    onSort,
    onFilter,
    onSelectAllPages,
  ]);

  /**
   * Render toolbar if provided
   */
  const renderToolbar = () => {
    if (!toolbar) return null;

    if (typeof toolbar === 'function') {
      return (toolbar as EditableTableRenderer)(tableContext);
    }

    return toolbar;
  };

  /**
   * Render footer if provided
   */
  const renderFooter = () => {
    if (!footer) return null;

    if (typeof footer === 'function') {
      return (footer as EditableTableRenderer)(tableContext);
    }

    return footer;
  };

  /**
   * Default grid options
   */
  const defaultColDef = useMemo<ColDef>(() => ({
    resizable: enableColumnResizing,
    sortable: true,
    filter: false, // Disable column filter icons - we use global filter instead
    editable: false,
    suppressMovable: !enableColumnReordering,
    suppressHeaderMenuButton: true, // Hide the menu button in headers
  }), [enableColumnResizing, enableColumnReordering]);

  /**
   * Row selection config - AG Grid v34 object-based API
   */
  const rowSelectionConfig = useMemo(() => {
    return enableMultiSelect
      ? { mode: 'multiRow' as const, checkboxes: true, headerCheckbox: true }
      : { mode: 'singleRow' as const };
  }, [enableMultiSelect]);

  /**
   * Selection column definition - customize the checkbox column
   * Note: Width needs to be at least 56px for header checkbox to render properly
   */
  const selectionColumnDef = useMemo<SelectionColumnDef>(() => ({
    width: 56,
    minWidth: 56,
    maxWidth: 56,
    suppressHeaderMenuButton: false, // Allow header checkbox to show
    pinned: 'left',
    lockPosition: 'left',
  }), []);

  /**
   * Auto-size strategy: undefined to disable auto-sizing
   * CRITICAL: Not using autoSizeStrategy prevents flicker from layout recalculations
   * Column widths are controlled via flex/width properties in columnDefs
   */
  const autoSizeStrategy = useMemo(() => undefined, []);

  /**
   * Container height calculation
   */
  const containerHeight = typeof height === 'number' ? `${height}px` : height;

  if (error) {
    return (
      <div className={cn('rounded-lg border border-destructive bg-destructive/10 p-4', className)}>
        <p className="text-sm text-destructive">{error}</p>
      </div>
    );
  }

  return (
    <div className={cn('flex flex-col', className)}>
      {renderToolbar()}

      <div
        className={cn(
          'ag-theme-quartz-dark',
          'border border-border',
          'font-size-10',
          'w-full'
        )}
        style={{ height: containerHeight }}
      >
        <AgGridReact
          ref={gridRef}
          rowData={memoizedRowData}
          columnDefs={columnDefs}
          defaultColDef={defaultColDef}
          autoSizeStrategy={autoSizeStrategy}
          getRowId={getRowId}
          onGridReady={onGridReady}
          onCellValueChanged={onCellValueChanged}
          onSelectionChanged={onSelectionChanged}
          onRowClicked={onRowClicked}
          onSortChanged={onSortChanged}
          onFilterChanged={onFilterChanged}
          onColumnMoved={onColumnMoved}
          onColumnVisible={onColumnVisible}
          onColumnResized={onColumnResized}
          rowSelection={rowSelectionConfig}
          selectionColumnDef={selectionColumnDef}
          // Performance: disable all animations to prevent flicker
          animateRows={false}
          enableCellTextSelection={false}
          loading={loading}
          suppressCellFocus={false}
          suppressMenuHide={true}
          domLayout="normal"
          rowHeight={31}
          headerHeight={36}
          suppressScrollOnNewData={true}
          // Performance: disable scrollbar debouncing for immediate scroll response
          debounceVerticalScrollbar={false}
          maintainColumnOrder={true}
          singleClickEdit={false}
          stopEditingWhenCellsLoseFocus={true}
          enterNavigatesVertically={true}
          enterNavigatesVerticallyAfterEdit={true}
          // Performance: pre-render 20 rows above/below viewport (620px buffer zone)
          // This creates a larger cushion to prevent white flash during fast scrolling
          rowBuffer={20}
          // Performance: reduce DOM overhead by removing row transform animations
          suppressRowTransform={true}
          // CRITICAL: Suppress layout recalculations that cause size changes/flicker
          suppressColumnVirtualisation={false}
          suppressRowVirtualisation={false}
          // Prevent automatic column sizing that causes flicker
          suppressAutoSize={true}
          // Skip header on horizontal scroll to prevent layout shift
          suppressHorizontalScroll={false}
          className="w-full h-full font-size-10"
        />
      </div>

      {renderFooter()}
    </div>
  );
};

AGGridTable.displayName = 'AGGridTable';
