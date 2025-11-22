/**
 * Type Migration Utilities
 *
 * Provides utilities for migrating legacy code from plain strings to branded types.
 * These helpers ensure backward compatibility during the transition period and make
 * it easy to convert existing data structures to use type-safe branded types.
 *
 * @example
 * ```typescript
 * // Migrate legacy row IDs
 * const legacyIds = ['row1', 'row2', 'row3'];
 * const brandedIds = migrateRowIds(legacyIds);
 *
 * // Migrate table rows with string __rowId to branded RowId
 * const legacyRows = [
 *   { __rowId: 'row-1', name: 'Alice' },
 *   { name: 'Bob' }, // No ID - will generate one
 * ];
 * const migratedRows = migrateTableRows(legacyRows);
 *
 * // Migrate cell key maps
 * const legacyMap = new Map([
 *   ['row1|col1', { value: 42 }],
 *   ['row2|col2', { value: 99 }],
 * ]);
 * const brandedMap = migrateCellKeyMap(legacyMap);
 * ```
 */

import {
  type CellKey,
  type ColumnId,
  createCellKey,
  createColumnId,
  createRowId,
  type RowId,
} from '../types/branded';

/**
 * @deprecated Use branded RowId instead
 */
export type LegacyRowId = string;

/**
 * @deprecated Use branded ColumnId instead
 */
export type LegacyColumnId = string;

/**
 * @deprecated Use TableRow with branded RowId instead
 */
export interface LegacyTableRow {
  __rowId?: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Table row columns are dynamic database values
  [key: string]: any;
}

/**
 * TableRow interface with branded RowId
 */
export interface TableRow {
  __rowId: RowId;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Table row columns are dynamic database values
  [key: string]: any;
}

/**
 * Generate a unique row ID for rows without existing IDs.
 * Uses timestamp and random string for uniqueness.
 *
 * @returns A unique row identifier string
 *
 * @example
 * ```typescript
 * const newId = generateRowId(); // 'row_1705392847123_a7f9k2x'
 * ```
 */
export const generateRowId = (): string => {
  return `row_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
};

/**
 * Convert an array of plain string row IDs to branded RowId.
 * Validates each ID and throws if any are invalid.
 *
 * @param ids - Array of plain string row identifiers
 * @returns Array of branded RowId values
 * @throws {TypeError} If any ID is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacyIds = ['row1', 'row2', 'row3'];
 * const brandedIds = migrateRowIds(legacyIds);
 * // brandedIds: RowId[] with type safety
 * ```
 */
export const migrateRowIds = (ids: string[]): RowId[] => {
  return ids.map(createRowId);
};

/**
 * Convert an array of plain string column IDs to branded ColumnId.
 * Validates each ID and throws if any are invalid.
 *
 * @param ids - Array of plain string column identifiers
 * @returns Array of branded ColumnId values
 * @throws {TypeError} If any ID is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacyIds = ['name', 'email', 'status'];
 * const brandedIds = migrateColumnIds(legacyIds);
 * // brandedIds: ColumnId[] with type safety
 * ```
 */
export const migrateColumnIds = (ids: string[]): ColumnId[] => {
  return ids.map(createColumnId);
};

/**
 * Convert a legacy TableRow with string __rowId to branded RowId.
 * Handles missing __rowId by generating a new unique ID.
 *
 * @param row - Legacy table row with optional string __rowId
 * @returns TableRow with branded RowId
 *
 * @example
 * ```typescript
 * const legacyRow = { __rowId: 'row-1', name: 'Alice', age: 30 };
 * const migratedRow = migrateTableRow(legacyRow);
 * // migratedRow.__rowId is now RowId type
 *
 * const rowWithoutId = { name: 'Bob', age: 25 };
 * const migratedRowWithId = migrateTableRow(rowWithoutId);
 * // Automatically gets a generated RowId
 * ```
 */
export const migrateTableRow = (row: LegacyTableRow): TableRow => {
  return {
    ...row,
    __rowId: row.__rowId ? createRowId(row.__rowId) : createRowId(generateRowId()),
  };
};

/**
 * Batch convert multiple legacy TableRows to use branded RowId.
 * Processes an array of rows, handling missing IDs automatically.
 *
 * @param rows - Array of legacy table rows
 * @returns Array of TableRows with branded RowId
 *
 * @example
 * ```typescript
 * const legacyRows = [
 *   { __rowId: 'row-1', name: 'Alice' },
 *   { name: 'Bob' }, // No ID - will generate one
 *   { __rowId: 'row-3', name: 'Charlie' },
 * ];
 * const migratedRows = migrateTableRows(legacyRows);
 * // All rows now have branded RowId
 * ```
 */
export const migrateTableRows = (rows: LegacyTableRow[]): TableRow[] => {
  return rows.map(migrateTableRow);
};

/**
 * Convert a Map with plain string keys to Map with branded CellKey.
 * Expects legacy keys in format "rowId|columnId".
 * Invalid keys (missing separator or empty parts) are silently skipped.
 *
 * @param map - Map with string keys in "rowId|columnId" format
 * @returns Map with branded CellKey keys
 *
 * @example
 * ```typescript
 * const legacyMap = new Map([
 *   ['row1|name', { value: 'Alice', dirty: false }],
 *   ['row1|email', { value: 'alice@example.com', dirty: true }],
 *   ['row2|name', { value: 'Bob', dirty: false }],
 * ]);
 * const brandedMap = migrateCellKeyMap(legacyMap);
 * // Keys are now CellKey type with format "rowId:columnId"
 *
 * // Access with type-safe keys
 * const key = createCellKey(createRowId('row1'), createColumnId('name'));
 * const cell = brandedMap.get(key);
 * ```
 */
export const migrateCellKeyMap = <V>(
  map: Map<string, V>
): Map<CellKey, V> => {
  const newMap = new Map<CellKey, V>();
  for (const [key, value] of map.entries()) {
    const parts = key.split('|');
    if (parts.length === 2) {
      const rowId = parts[0]?.trim();
      const columnId = parts[1]?.trim();
      if (rowId && columnId) {
        newMap.set(
          createCellKey(createRowId(rowId), createColumnId(columnId)),
          value
        );
      }
    }
  }
  return newMap;
};

/**
 * Convert a Set of plain string row IDs to Set of branded RowId.
 *
 * @param set - Set of plain string row identifiers
 * @returns Set of branded RowId values
 * @throws {TypeError} If any ID is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacySet = new Set(['row1', 'row2', 'row3']);
 * const brandedSet = migrateRowIdSet(legacySet);
 * // brandedSet: Set<RowId> with type safety
 * ```
 */
export const migrateRowIdSet = (set: Set<string>): Set<RowId> => {
  return new Set(Array.from(set).map(createRowId));
};

/**
 * Convert a Set of plain string column IDs to Set of branded ColumnId.
 *
 * @param set - Set of plain string column identifiers
 * @returns Set of branded ColumnId values
 * @throws {TypeError} If any ID is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacySet = new Set(['name', 'email', 'status']);
 * const brandedSet = migrateColumnIdSet(legacySet);
 * // brandedSet: Set<ColumnId> with type safety
 * ```
 */
export const migrateColumnIdSet = (set: Set<string>): Set<ColumnId> => {
  return new Set(Array.from(set).map(createColumnId));
};

/**
 * Convert an object with string keys representing row IDs to use branded RowId.
 * Creates a new Map with branded keys.
 *
 * @param obj - Object with string keys as row identifiers
 * @returns Map with branded RowId keys
 * @throws {TypeError} If any key is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacyObj = {
 *   'row1': { name: 'Alice', status: 'active' },
 *   'row2': { name: 'Bob', status: 'inactive' },
 * };
 * const brandedMap = migrateRowIdObject(legacyObj);
 * // brandedMap: Map<RowId, any> with type safety
 * ```
 */
export const migrateRowIdObject = <V>(obj: Record<string, V>): Map<RowId, V> => {
  const map = new Map<RowId, V>();
  for (const [key, value] of Object.entries(obj)) {
    map.set(createRowId(key), value);
  }
  return map;
};

/**
 * Convert an object with string keys representing column IDs to use branded ColumnId.
 * Creates a new Map with branded keys.
 *
 * @param obj - Object with string keys as column identifiers
 * @returns Map with branded ColumnId keys
 * @throws {TypeError} If any key is not a non-empty string
 *
 * @example
 * ```typescript
 * const legacyObj = {
 *   'name': { width: 200, visible: true },
 *   'email': { width: 300, visible: true },
 *   'status': { width: 100, visible: false },
 * };
 * const brandedMap = migrateColumnIdObject(legacyObj);
 * // brandedMap: Map<ColumnId, any> with type safety
 * ```
 */
export const migrateColumnIdObject = <V>(obj: Record<string, V>): Map<ColumnId, V> => {
  const map = new Map<ColumnId, V>();
  for (const [key, value] of Object.entries(obj)) {
    map.set(createColumnId(key), value);
  }
  return map;
};
