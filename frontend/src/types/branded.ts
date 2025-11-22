/**
 * Branded Types Module
 *
 * Provides type-safe branded types to prevent ID confusion bugs at compile time.
 * Using branded types ensures that IDs cannot be accidentally swapped (e.g., using
 * a ColumnId where a RowId is expected).
 *
 * @example
 * ```typescript
 * const rowId = createRowId('123');
 * const colId = createColumnId('name');
 *
 * // ❌ Compile error - can't swap!
 * updateCell(colId, rowId, value);
 *
 * // ✅ Type-safe
 * updateCell(rowId, colId, value);
 *
 * // ✅ Type-safe cell keys
 * const key = createCellKey(rowId, colId);
 * const { rowId: r, columnId: c } = parseCellKey(key);
 * ```
 */

/**
 * Core branded type helper.
 * Creates a nominal type by attaching a unique brand to a base type.
 *
 * @template T - The base type (e.g., string, number)
 * @template B - The brand identifier (unique symbol or string literal)
 */
type Brand<T, B> = T & { readonly __brand: B };

/**
 * Branded type for row identifiers.
 * Prevents accidental mixing with column IDs or plain strings.
 */
export type RowId = Brand<string, 'RowId'>;

/**
 * Branded type for column identifiers.
 * Prevents accidental mixing with row IDs or plain strings.
 */
export type ColumnId = Brand<string, 'ColumnId'>;

/**
 * Branded type for cell map keys.
 * Composite key in format "rowId:columnId" that ensures type-safe cell access.
 */
export type CellKey = Brand<string, 'CellKey'>;

/**
 * Creates a validated RowId from a string.
 * This is the only safe way to construct a RowId.
 *
 * @param id - The row identifier string
 * @returns A branded RowId
 * @throws {TypeError} If id is not a non-empty string
 *
 * @example
 * ```typescript
 * const rowId = createRowId('user-123');
 * ```
 */
export function createRowId(id: string): RowId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid RowId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`);
  }
  return id as RowId;
}

/**
 * Creates a validated ColumnId from a string.
 * This is the only safe way to construct a ColumnId.
 *
 * @param id - The column identifier string
 * @returns A branded ColumnId
 * @throws {TypeError} If id is not a non-empty string
 *
 * @example
 * ```typescript
 * const colId = createColumnId('name');
 * ```
 */
export function createColumnId(id: string): ColumnId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid ColumnId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`);
  }
  return id as ColumnId;
}

/**
 * Creates a composite CellKey from a RowId and ColumnId.
 * The key format is "rowId:columnId".
 *
 * @param rowId - The branded row identifier
 * @param columnId - The branded column identifier
 * @returns A branded CellKey
 *
 * @example
 * ```typescript
 * const rowId = createRowId('123');
 * const colId = createColumnId('name');
 * const key = createCellKey(rowId, colId); // "123:name"
 * ```
 */
export function createCellKey(rowId: RowId, columnId: ColumnId): CellKey {
  return `${rowId}:${columnId}` as CellKey;
}

/**
 * Parses a CellKey back into its constituent RowId and ColumnId.
 *
 * @param key - The branded cell key to parse
 * @returns An object containing the separated rowId and columnId
 * @throws {TypeError} If key format is invalid (missing separator)
 *
 * @example
 * ```typescript
 * const key = createCellKey(createRowId('123'), createColumnId('name'));
 * const { rowId, columnId } = parseCellKey(key);
 * // rowId: RowId = '123'
 * // columnId: ColumnId = 'name'
 * ```
 */
export function parseCellKey(key: CellKey): { rowId: RowId; columnId: ColumnId } {
  const parts = (key as string).split(':');
  if (parts.length !== 2) {
    throw new TypeError(`Invalid CellKey format: expected "rowId:columnId", got "${key}"`);
  }
  return {
    rowId: parts[0] as RowId,
    columnId: parts[1] as ColumnId,
  };
}

/**
 * Type guard to check if a value is a RowId.
 * Note: This performs runtime validation but cannot verify the brand at runtime.
 *
 * @param value - The value to check
 * @returns True if the value appears to be a valid RowId
 *
 * @example
 * ```typescript
 * if (isRowId(someValue)) {
 *   // TypeScript knows someValue is RowId here
 *   updateRow(someValue);
 * }
 * ```
 */
export function isRowId(value: unknown): value is RowId {
  return typeof value === 'string' && value.trim() !== '';
}

/**
 * Type guard to check if a value is a ColumnId.
 * Note: This performs runtime validation but cannot verify the brand at runtime.
 *
 * @param value - The value to check
 * @returns True if the value appears to be a valid ColumnId
 *
 * @example
 * ```typescript
 * if (isColumnId(someValue)) {
 *   // TypeScript knows someValue is ColumnId here
 *   updateColumn(someValue);
 * }
 * ```
 */
export function isColumnId(value: unknown): value is ColumnId {
  return typeof value === 'string' && value.trim() !== '';
}

/**
 * Unwraps a RowId to its underlying string value.
 * Use sparingly - prefer keeping the branded type when possible.
 *
 * @param id - The branded RowId
 * @returns The raw string value
 *
 * @example
 * ```typescript
 * const rowId = createRowId('123');
 * const rawId = unwrapRowId(rowId); // '123' as string
 * ```
 */
export function unwrapRowId(id: RowId): string {
  return id as string;
}

/**
 * Unwraps a ColumnId to its underlying string value.
 * Use sparingly - prefer keeping the branded type when possible.
 *
 * @param id - The branded ColumnId
 * @returns The raw string value
 *
 * @example
 * ```typescript
 * const colId = createColumnId('name');
 * const rawId = unwrapColumnId(colId); // 'name' as string
 * ```
 */
export function unwrapColumnId(id: ColumnId): string {
  return id as string;
}
