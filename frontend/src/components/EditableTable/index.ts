// Main component
export { EditableTable } from './EditableTable';
export { EditableTableDemo } from './EditableTableDemo';

// Sub-components
export { TableCell } from './TableCell';
export { TableHeader } from './TableHeader';
export { TableToolbar } from './TableToolbar';
export { StatusBar } from './StatusBar';
export { CellEditor } from './CellEditor';

// Hooks
export { useTableState } from '../../hooks/useTableState';
export { useKeyboardNavigation } from '../../hooks/useKeyboardNavigation';
export { useOptimisticUpdates } from '../../hooks/useOptimisticUpdates';
export { usePerformance, useRenderPerformance, useVirtualScrollingOptimization } from '../../hooks/usePerformance';

// Utilities
export {
  validateCellValue,
  formatCellValue,
  parseCellValue,
  copyToClipboard,
  pasteFromClipboard,
  exportData,
  debounce,
  throttle,
  getColumnWidth,
  generateTableId,
  isEqual,
  cloneDeep,
} from '../../utils/table';

// Types
export type {
  EditableTableProps,
  TableRow,
  TableColumn,
  CellValue,
  TableState,
  TableAction,
  CellEditState,
  ValidationResult,
  ClipboardData,
  ExportOptions,
  FilterOption,
  TableToolbarProps,
  StatusBarProps,
  ColumnHeaderProps,
  CellEditorProps,
  KeyboardNavigationState,
  TableConfig,
  TableMetrics,
} from '../../types/table';