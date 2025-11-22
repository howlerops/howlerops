import { ReactNode } from 'react';
import { SortingState, ColumnFiltersState, Header } from '@tanstack/react-table';
import type { ResultDisplayMode } from '../lib/query-result-storage';

export type CellValue = string | number | boolean | null | undefined;

export interface TableRow {
  __rowId?: string;
  __isNewRow?: boolean;
  [key: string]: CellValue | boolean | undefined;
}

export interface TableColumn {
  id?: string;
  accessorKey?: string;
  header: string;
  type: 'text' | 'number' | 'boolean' | 'date' | 'datetime' | 'select';
  width?: number;
  minWidth?: number;
  maxWidth?: number;
  preferredWidth?: number;
  sortable?: boolean;
  filterable?: boolean;
  editable?: boolean;
  required?: boolean;
  options?: string[]; // For select type
  sticky?: 'left' | 'right'; // For sticky columns
  longText?: boolean;
  wrapContent?: boolean;
  clipContent?: boolean;
  monospace?: boolean;
  validation?: {
    pattern?: RegExp;
    min?: number;
    max?: number;
    message?: string;
  };
  hasDefault?: boolean;
  defaultLabel?: string;
  defaultValue?: unknown;
  autoNumber?: boolean;
  isPrimaryKey?: boolean;
}

export interface EditableTableProps {
  data: TableRow[];
  columns: TableColumn[];
  onDataChange?: (data: TableRow[]) => void;
  onCellEdit?: (rowId: string, columnId: string, value: CellValue) => Promise<boolean>;
  onRowSelect?: (selectedRows: string[]) => void;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
  onRowInspect?: (rowId: string, rowData: TableRow) => void;
  onSort?: (sorting: SortingState) => void;
  onFilter?: (filters: ColumnFiltersState) => void;
  onExport?: (options: ExportOptions) => void;
  onSelectAllPages?: () => void; // Callback when user wants to select all rows across all pages
  loading?: boolean;
  error?: string | null;
  virtualScrolling?: boolean;
  estimateSize?: number;
  className?: string;
  height?: number | string;
  enableMultiSelect?: boolean;
  enableColumnResizing?: boolean;
  enableColumnReordering?: boolean;
  enableGlobalFilter?: boolean;
  enableExport?: boolean;
  toolbar?: ReactNode | EditableTableRenderer;
  footer?: ReactNode | EditableTableRenderer;
  onDirtyChange?: (dirtyRowIds: string[]) => void;
  customCellRenderers?: Record<string, (value: CellValue, row: TableRow) => ReactNode>;
  // Phase 2: Chunked data loading
  resultId?: string;
  totalRows?: number;
  isLargeResult?: boolean;
  chunkingEnabled?: boolean;
  displayMode?: ResultDisplayMode;
}

export interface EditableTableContext {
  data: TableRow[];
  state: TableState & {
    hasUndoActions: boolean;
    hasRedoActions: boolean;
    hasSelection: boolean;
    hasDirtyRows: boolean;
    hasInvalidCells: boolean;
    isEditing: boolean;
  };
  actions: EditableTableActions;
}

export type EditableTableRenderer = (context: EditableTableContext) => ReactNode;

export interface EditableTableActions {
  updateCell: (rowId: string, columnId: string, newValue: CellValue, addToHistory?: boolean) => boolean;
  startEditing: (rowId: string, columnId: string, value: CellValue) => void;
  updateEditingCell: (value: CellValue, isValid: boolean, error?: string) => void;
  cancelEditing: () => void;
  saveEditing: () => Promise<boolean>;
  toggleRowSelection: (rowId: string, selected?: boolean) => void;
  selectAllRows: (selected: boolean) => void;
  setSelectedRows: (rowIds: string[]) => void;
  setSelectAllPagesMode: (enabled: boolean) => void;
  updateSorting: (sorting: SortingState) => void;
  updateColumnFilters: (columnFilters: ColumnFiltersState) => void;
  updateGlobalFilter: (globalFilter: string) => void;
  updateColumnVisibility: (columnVisibility: Record<string, boolean>) => void;
  updateColumnSizing: (columnSizing: Record<string, number>) => void;
  updateColumnOrder: (columnOrder: string[]) => void;
  undo: () => void;
  redo: () => void;
  clearDirtyRows: () => void;
  resetTable: () => void;
  getInvalidCells: () => Array<{ rowId: string; columnId: string; error: string }>;
  validateAllCells: () => boolean;
  clearInvalidCells: () => void;
  trackValidationError: (rowId: string, columnId: string, error: string) => void;
  clearValidationError: (rowId: string, columnId: string) => void;
}

export interface CellEditState {
  rowId: string;
  columnId: string;
  value: CellValue;
  originalValue: CellValue;
  isValid: boolean;
  error?: string;
}

export interface TableState {
  editingCell: CellEditState | null;
  selectedRows: string[];
  selectAllPagesMode: boolean; // True when user wants to select all rows across all pages
  sorting: SortingState;
  columnFilters: ColumnFiltersState;
  globalFilter: string;
  columnVisibility: Record<string, boolean>;
  columnOrder: string[];
  columnSizing: Record<string, number>;
  dirtyRows: Set<string>;
  invalidCells: Map<string, { columnId: string; error: string }>;
  undoStack: TableAction[];
  redoStack: TableAction[];
}

export interface TableAction {
  type: 'edit' | 'delete' | 'add' | 'bulk_edit';
  payload: {
    rowId?: string;
    columnId?: string;
    oldValue?: CellValue;
    newValue?: CellValue;
    rows?: TableRow[];
  };
  timestamp: number;
}

export interface ValidationResult {
  isValid: boolean;
  error?: string;
}

export interface ClipboardData {
  rows: number;
  columns: number;
  data: CellValue[][];
}

export interface ExportOptions {
  format: 'csv' | 'json' | 'xlsx';
  filename?: string;
  selectedOnly?: boolean;
  includeHeaders?: boolean;
}

export interface FilterOption {
  label: string;
  value: string;
  count?: number;
}

export interface TableToolbarProps {
  searchValue: string;
  onSearchChange: (value: string) => void;
  onExport?: (options: ExportOptions) => void;
  onImport?: () => void;
  onRefresh?: () => void;
  selectedCount: number;
  totalCount: number;
  loading?: boolean;
  showExport?: boolean;
  showImport?: boolean;
  customActions?: ReactNode;
}

export interface StatusBarProps {
  totalRows: number;
  selectedRows: number;
  filteredRows?: number;
  dirtyRows: number;
  loading?: boolean;
  lastUpdated?: Date;
  customStatus?: ReactNode;
}

export interface ColumnHeaderProps {
  header: Header<TableRow, unknown>;
  canSort?: boolean;
  canFilter?: boolean;
  canResize?: boolean;
  sortDirection?: 'asc' | 'desc' | false;
}

export interface CellEditorProps {
  value: CellValue;
  type: TableColumn['type'];
  onChange: (value: CellValue, isValid: boolean, error?: string) => void;
  onCancel: () => void;
  onSave: () => void;
  validation?: TableColumn['validation'];
  options?: string[];
  required?: boolean;
  autoFocus?: boolean;
  className?: string;
}

export interface KeyboardNavigationState {
  focusedCell: {
    rowIndex: number;
    columnIndex: number;
  } | null;
  isEditing: boolean;
  selection: {
    start: { rowIndex: number; columnIndex: number };
    end: { rowIndex: number; columnIndex: number };
  } | null;
}

export interface TableConfig {
  estimateSize: number;
  overscan: number;
  debounceMs: number;
  maxUndoHistory: number;
  autoSave: boolean;
  autoSaveInterval: number;
  enableVirtualization: boolean;
  enablePersistence: boolean;
  persistenceKey?: string;
}

export interface TableMetrics {
  totalRows: number;
  visibleRows: number;
  filteredRows: number;
  selectedRows: number;
  dirtyRows: number;
  renderTime: number;
  scrollPosition: number;
  virtualItems: {
    start: number;
    end: number;
  };
}

export interface SearchMatch {
  path: string;
  key: string;
  value: CellValue;
}

export interface JsonViewerState {
  isOpen: boolean;
  currentRow: TableRow | null;
  currentRowId: string | null;
  isEditing: boolean;
  editedData: Record<string, CellValue> | null;
  validationErrors: Map<string, string>;
  wordWrap: boolean;
  expandedKeys: Set<string>;
  collapsedKeys: Set<string>;
  searchQuery: string;
  searchResults: {
    matches: SearchMatch[];
    currentIndex: number;
    totalMatches: number;
  };
  useRegex: boolean;
  searchKeys: boolean;
  searchValues: boolean;
  expandedForeignKeys: Set<string>;
  foreignKeyCache: Map<string, unknown>;
  isLoading: boolean;
  isSaving: boolean;
  saveError: string | null;
}
