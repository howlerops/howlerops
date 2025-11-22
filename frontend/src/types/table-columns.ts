/**
 * Type-Safe Table Column Definitions
 *
 * This module provides discriminated union types for table columns to ensure
 * compile-time type safety and prevent property misuse (e.g., putting `options`
 * on a NumberColumn).
 *
 * Key features:
 * - Each column type has specific validation properties
 * - SelectColumn REQUIRES options array
 * - Type guards enable safe property access
 * - Value types are inferred from column types
 *
 * @example
 * ```typescript
 * // ✅ Type-safe column definition
 * const nameColumn: TextColumn = {
 *   type: 'text',
 *   id: createColumnId('name'),
 *   accessorKey: 'name',
 *   header: 'Name',
 *   validation: { minLength: 1, maxLength: 100 }
 * };
 *
 * // ❌ Compile error - options only on SelectColumn!
 * const ageColumn: NumberColumn = {
 *   type: 'number',
 *   options: ['1', '2', '3'] // TypeScript error!
 * };
 *
 * // ✅ Type narrowing works
 * if (isSelectColumn(column)) {
 *   column.options.forEach(...); // TypeScript knows options exists!
 * }
 * ```
 */

import type { ColumnId } from './branded';

/**
 * Base properties shared by all column types
 */
interface BaseColumn {
  /** Unique identifier for the column */
  id: ColumnId;
  /** Key to access data in the row object */
  accessorKey: string;
  /** Display header text */
  header: string;

  // Sizing
  /** Fixed width in pixels */
  width?: number;
  /** Minimum width in pixels */
  minWidth?: number;
  /** Maximum width in pixels */
  maxWidth?: number;
  /** Preferred width (used for initial sizing) */
  preferredWidth?: number;

  // Behavior
  /** Whether column can be sorted */
  sortable?: boolean;
  /** Whether column can be filtered */
  filterable?: boolean;
  /** Whether cells in this column can be edited */
  editable?: boolean;
  /** Whether this field is required (non-null) */
  required?: boolean;

  // Layout
  /** Pin column to left or right side */
  sticky?: 'left' | 'right';
  /** Whether this column is the primary key */
  isPrimaryKey?: boolean;

  // Display hints
  /** Display long text content */
  longText?: boolean;
  /** Wrap text content */
  wrapContent?: boolean;
  /** Clip content with ellipsis */
  clipContent?: boolean;
  /** Use monospace font */
  monospace?: boolean;

  // Default values
  /** Whether column has a default value */
  hasDefault?: boolean;
  /** Label to display for default value */
  defaultLabel?: string;
  /** The actual default value */
  defaultValue?: unknown;
  /** Auto-increment number column */
  autoNumber?: boolean;
}

/**
 * Text column with string validation
 */
export interface TextColumn extends BaseColumn {
  type: 'text';
  validation?: {
    /** Regex pattern the text must match */
    pattern?: RegExp;
    /** Minimum string length */
    minLength?: number;
    /** Maximum string length */
    maxLength?: number;
    /** Custom validation error message */
    message?: string;
  };
}

/**
 * Number column with numeric validation
 */
export interface NumberColumn extends BaseColumn {
  type: 'number';
  validation?: {
    /** Minimum numeric value */
    min?: number;
    /** Maximum numeric value */
    max?: number;
    /** Step increment for number input */
    step?: number;
    /** Number of decimal places */
    precision?: number;
    /** Custom validation error message */
    message?: string;
  };
}

/**
 * Boolean column (checkbox)
 */
export interface BooleanColumn extends BaseColumn {
  type: 'boolean';
}

/**
 * Date column (date only, no time)
 */
export interface DateColumn extends BaseColumn {
  type: 'date';
  validation?: {
    /** Minimum allowed date (ISO string) */
    minDate?: string;
    /** Maximum allowed date (ISO string) */
    maxDate?: string;
    /** Custom validation error message */
    message?: string;
  };
}

/**
 * DateTime column (date and time)
 */
export interface DateTimeColumn extends BaseColumn {
  type: 'datetime';
  validation?: {
    /** Minimum allowed datetime (ISO string) */
    minDate?: string;
    /** Maximum allowed datetime (ISO string) */
    maxDate?: string;
    /** Custom validation error message */
    message?: string;
  };
}

/**
 * Select column with predefined options
 *
 * NOTE: `options` is REQUIRED for SelectColumn!
 */
export interface SelectColumn extends BaseColumn {
  type: 'select';
  /** Available options for selection (REQUIRED!) */
  options: readonly string[];
}

/**
 * Discriminated union of all column types
 *
 * This enables type-safe column handling with TypeScript's type narrowing.
 */
export type TableColumn =
  | TextColumn
  | NumberColumn
  | BooleanColumn
  | DateColumn
  | DateTimeColumn
  | SelectColumn;

/**
 * Extract the value type for a given column type
 *
 * @example
 * ```typescript
 * type NameValue = ColumnValueType<TextColumn>; // string
 * type AgeValue = ColumnValueType<NumberColumn>; // number
 * type ActiveValue = ColumnValueType<BooleanColumn>; // boolean
 * ```
 */
export type ColumnValueType<T extends TableColumn> =
  T extends TextColumn ? string :
  T extends NumberColumn ? number :
  T extends BooleanColumn ? boolean :
  T extends DateColumn ? string :
  T extends DateTimeColumn ? string :
  T extends SelectColumn ? string :
  never;

/**
 * Type-safe cell value that can be null or undefined
 */
export type TypedCellValue<T extends TableColumn = TableColumn> =
  ColumnValueType<T> | null | undefined;

// =============================================================================
// Type Guards
// =============================================================================

/**
 * Type guard for TextColumn
 */
export function isTextColumn(column: TableColumn): column is TextColumn {
  return column.type === 'text';
}

/**
 * Type guard for NumberColumn
 */
export function isNumberColumn(column: TableColumn): column is NumberColumn {
  return column.type === 'number';
}

/**
 * Type guard for BooleanColumn
 */
export function isBooleanColumn(column: TableColumn): column is BooleanColumn {
  return column.type === 'boolean';
}

/**
 * Type guard for DateColumn
 */
export function isDateColumn(column: TableColumn): column is DateColumn {
  return column.type === 'date';
}

/**
 * Type guard for DateTimeColumn
 */
export function isDateTimeColumn(column: TableColumn): column is DateTimeColumn {
  return column.type === 'datetime';
}

/**
 * Type guard for SelectColumn
 */
export function isSelectColumn(column: TableColumn): column is SelectColumn {
  return column.type === 'select';
}

// =============================================================================
// Utility Types
// =============================================================================

/**
 * Extract columns of a specific type from a union
 *
 * @example
 * ```typescript
 * type EditableColumns = ExtractColumnType<TextColumn | NumberColumn>;
 * ```
 */
export type ExtractColumnType<T extends TableColumn> = Extract<TableColumn, { type: T['type'] }>;

/**
 * Get the validation type for a column
 */
export type ColumnValidation<T extends TableColumn> =
  T extends TextColumn ? TextColumn['validation'] :
  T extends NumberColumn ? NumberColumn['validation'] :
  T extends DateColumn ? DateColumn['validation'] :
  T extends DateTimeColumn ? DateColumn['validation'] :
  undefined;

/**
 * Helper to create type-safe column definitions
 *
 * @example
 * ```typescript
 * const column = createColumn<TextColumn>({
 *   type: 'text',
 *   id: createColumnId('name'),
 *   accessorKey: 'name',
 *   header: 'Name'
 * });
 * ```
 */
export function createColumn<T extends TableColumn>(column: T): T {
  return column;
}
