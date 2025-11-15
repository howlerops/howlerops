/**
 * Batch Processing Utilities
 *
 * Provides async batch processing for CPU-intensive operations
 * to keep the UI responsive during large data processing.
 */

export interface BatchProcessorOptions {
  batchSize?: number
  onProgress?: (processed: number, total: number) => void
  signal?: AbortSignal
}

export interface QueryResultRow {
  __rowId: string
  [key: string]: unknown
}

export interface QueryEditableMetadata {
  primaryKeys?: string[]
}

export interface NormalisedRowsResult {
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
}

/**
 * Process data in batches with yielding to keep UI responsive
 */
async function yieldToEventLoop(): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, 0))
}

/**
 * Generate a unique row ID
 */
function generateRowId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

/**
 * Assign value handling SQL null types
 */
function assignValue(target: Record<string, unknown>, columnName: string, value: unknown): void {
  if (value && typeof value === 'object' && 'String' in value && 'Valid' in value) {
    const sqlValue = value as { String: unknown; Valid: boolean }
    target[columnName] = sqlValue.Valid ? sqlValue.String : null
  } else {
    target[columnName] = value
  }
}

/**
 * Process a single row (extracted from normaliseRows for reusability)
 */
function processRow(
  row: unknown,
  rowIndex: number,
  columns: string[],
  primaryKeyColumns: string[]
): QueryResultRow {
  const record: Record<string, unknown> = {}

  // Handle different row formats (array or object)
  if (Array.isArray(row)) {
    row.forEach((value, index) => {
      const columnName = columns[index] ?? `col_${index}`
      assignValue(record, columnName, value)
    })
  } else if (row && typeof row === 'object') {
    const rowObject = row as Record<string | number, unknown>
    columns.forEach((columnName, index) => {
      if (columnName in rowObject) {
        assignValue(record, columnName, rowObject[columnName])
      } else if (index in rowObject) {
        assignValue(record, columnName, rowObject[index])
      } else if (String(index) in rowObject) {
        assignValue(record, columnName, rowObject[String(index)])
      } else {
        assignValue(record, columnName, undefined)
      }
    })
  } else {
    columns.forEach((columnName) => assignValue(record, columnName, undefined))
  }

  // Generate row ID from primary keys if available
  let rowId = ''
  if (primaryKeyColumns.length > 0) {
    const parts: string[] = []
    let allPresent = true
    primaryKeyColumns.forEach((pkColumn) => {
      const value = record[pkColumn]
      if (value === undefined) {
        allPresent = false
      } else {
        const serialised = value === null || value === undefined ? 'NULL' : String(value)
        parts.push(`${pkColumn}:${serialised}`)
      }
    })
    if (allPresent && parts.length > 0) {
      rowId = parts.join('|')
    }
  }

  // Fallback to generated ID
  if (!rowId) {
    rowId = `${generateRowId()}-${rowIndex}`
  }

  return {
    ...record,
    __rowId: rowId,
  }
}

/**
 * Batched async version of normaliseRows
 * Processes rows in chunks to keep UI responsive
 */
export async function normaliseRowsBatched(
  columns: string[],
  rows: unknown[],
  metadata?: QueryEditableMetadata | null,
  options: BatchProcessorOptions = {}
): Promise<NormalisedRowsResult> {
  const { batchSize = 200, onProgress, signal } = options

  if (!Array.isArray(rows)) {
    return {
      rows: [],
      originalRows: {},
    }
  }

  const processedRows: QueryResultRow[] = []
  const originalRows: Record<string, QueryResultRow> = {}

  // Create column lookup for case-insensitive matching
  const columnLookup: Record<string, string> = {}
  columns.forEach((name) => {
    columnLookup[name.toLowerCase()] = name
  })

  // Normalize primary key columns
  const primaryKeyColumns = (metadata?.primaryKeys || []).map((pk) => {
    return columnLookup[pk.toLowerCase()] ?? pk
  })

  const totalRows = rows.length
  let processedCount = 0

  // Process in batches
  for (let i = 0; i < totalRows; i += batchSize) {
    // Check for abort signal
    if (signal?.aborted) {
      throw new Error('Batch processing aborted')
    }

    const batchEnd = Math.min(i + batchSize, totalRows)
    const batch = rows.slice(i, batchEnd)

    // Process this batch synchronously (it's small enough)
    batch.forEach((row, batchIndex) => {
      const rowIndex = i + batchIndex
      const completeRow = processRow(row, rowIndex, columns, primaryKeyColumns)

      processedRows.push(completeRow)
      originalRows[completeRow.__rowId] = { ...completeRow }
    })

    processedCount = batchEnd

    // Report progress
    if (onProgress) {
      onProgress(processedCount, totalRows)
    }

    // Yield to event loop every batch to keep UI responsive
    // Skip yielding on the last batch for slightly better performance
    if (batchEnd < totalRows) {
      await yieldToEventLoop()
    }
  }

  return {
    rows: processedRows,
    originalRows,
  }
}

/**
 * Calculate optimal batch size based on row count
 * Larger datasets get smaller batches for more frequent UI updates
 */
export function calculateOptimalBatchSize(rowCount: number): number {
  if (rowCount < 1000) return 500
  if (rowCount < 5000) return 300
  if (rowCount < 10000) return 200
  return 100 // For very large datasets, smaller batches = more responsive
}
