// Main component
export { EditableTable } from './editable-table';

// Sub-components
export { CellEditor } from './cell-editor';
export { StatusBar } from './status-bar';
export { TableCell } from './table-cell';
export { TableHeader } from './table-header';
export { TableToolbar } from './table-toolbar';

// Hooks
export { useKeyboardNavigation } from '../../hooks/use-keyboard-navigation';
export { useOptimisticUpdates } from '../../hooks/use-optimistic-updates';
export { usePerformance, useRenderPerformance, useVirtualScrollingOptimization } from '../../hooks/use-performance';
export { useTableState } from '../../hooks/use-table-state';

// Utilities
export {
  cloneDeep,
  copyToClipboard,
  debounce,
  exportData,
  formatCellValue,
  generateTableId,
  getColumnWidth,
  isEqual,
  parseCellValue,
  pasteFromClipboard,
  throttle,
  validateCellValue,
} from '../../utils/table';

// Types
export type {
  CellEditorProps,
  CellEditState,
  CellValue,
  ClipboardData,
  ColumnHeaderProps,
  EditableTableProps,
  ExportOptions,
  FilterOption,
  KeyboardNavigationState,
  StatusBarProps,
  TableAction,
  TableColumn,
  TableConfig,
  TableMetrics,
  TableRow,
  TableState,
  TableToolbarProps,
  ValidationResult,
} from '../../types/table';