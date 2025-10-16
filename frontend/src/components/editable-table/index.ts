// Main component
export { EditableTable } from './editable-table';
export { EditableTableDemo } from './editable-table-demo';

// Sub-components
export { TableCell } from './table-cell';
export { TableHeader } from './table-header';
export { TableToolbar } from './table-toolbar';
export { StatusBar } from './status-bar';
export { CellEditor } from './cell-editor';

// Hooks
export { useTableState } from '../../hooks/use-table-state';
export { useKeyboardNavigation } from '../../hooks/use-keyboard-navigation';
export { useOptimisticUpdates } from '../../hooks/use-optimistic-updates';
export { usePerformance, useRenderPerformance, useVirtualScrollingOptimization } from '../../hooks/use-performance';

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