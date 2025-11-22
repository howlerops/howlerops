/**
 * Query Result Storage Service
 *
 * Handles storage of query results with automatic fallback:
 * - Small results (< 100 rows): Stored in memory (Zustand)
 * - Large results (>= 100 rows): Stored in IndexedDB
 *
 * This prevents memory issues when loading very large datasets.
 */

import { createStore, del, get, set, UseStore } from 'idb-keyval'

// Custom IndexedDB store for query results
const queryResultStore: UseStore = createStore('sql-studio-db', 'query-results')

// Threshold for storing in IndexedDB vs memory (rows)
const LARGE_RESULT_THRESHOLD = 100

// Maximum results to keep in IndexedDB (LRU eviction)
const MAX_STORED_RESULTS = 50

export interface StoredQueryResult {
  id: string
  tabId: string
  columns: string[]
  rows: unknown[]
  originalRows: Record<string, unknown>
  rowCount: number
  affectedRows: number
  executionTime: number
  error?: string
  timestamp: Date
  editable?: unknown
  query: string
  connectionId?: string
}

export interface QueryResultMetadata {
  id: string
  tabId: string
  rowCount: number
  isLarge: boolean // true if stored in IndexedDB
  timestamp: Date
  lastAccessed: Date
}

/**
 * Store query result (automatically chooses memory vs IndexedDB based on size)
 */
export async function storeQueryResult(result: StoredQueryResult): Promise<QueryResultMetadata> {
  const isLarge = result.rows.length >= LARGE_RESULT_THRESHOLD

  const metadata: QueryResultMetadata = {
    id: result.id,
    tabId: result.tabId,
    rowCount: result.rows.length,
    isLarge,
    timestamp: result.timestamp,
    lastAccessed: new Date(),
  }

  if (isLarge) {
    // Store large results in IndexedDB
    await set(result.id, result, queryResultStore)

    // Update metadata for LRU eviction
    await updateResultsMetadata(metadata)

    // Enforce max results limit (LRU eviction)
    await enforceStorageLimit()
  }

  return metadata
}

/**
 * Get query result from IndexedDB
 */
export async function getQueryResult(resultId: string): Promise<StoredQueryResult | null> {
  try {
    const result = await get<StoredQueryResult>(resultId, queryResultStore)

    if (result) {
      // Update last accessed time for LRU
      await updateLastAccessed(resultId)
    }

    return result || null
  } catch (error) {
    console.error('Failed to get query result from IndexedDB:', error)
    return null
  }
}

/**
 * Get a range of rows from a stored result (for virtualization)
 */
export async function getQueryResultRows(
  resultId: string,
  startIndex: number,
  endIndex: number
): Promise<unknown[] | null> {
  try {
    const result = await get<StoredQueryResult>(resultId, queryResultStore)

    if (!result) {
      return null
    }

    // Return the requested slice of rows
    return result.rows.slice(startIndex, endIndex + 1)
  } catch (error) {
    console.error('Failed to get query result rows:', error)
    return null
  }
}

/**
 * Delete query result from IndexedDB
 */
export async function deleteQueryResult(resultId: string): Promise<void> {
  try {
    await del(resultId, queryResultStore)
    await removeResultMetadata(resultId)
  } catch (error) {
    console.error('Failed to delete query result:', error)
  }
}

/**
 * Delete all results for a specific tab
 */
export async function deleteTabResults(tabId: string): Promise<void> {
  try {
    const metadata = await getAllResultsMetadata()
    const tabResultIds = metadata
      .filter(m => m.tabId === tabId)
      .map(m => m.id)

    await Promise.all(tabResultIds.map(id => deleteQueryResult(id)))
  } catch (error) {
    console.error('Failed to delete tab results:', error)
  }
}

/**
 * Clear all stored results (useful for cleanup)
 */
export async function clearAllResults(): Promise<void> {
  try {
    // Get all metadata
    const metadata = await getAllResultsMetadata()

    // Delete all results
    await Promise.all(metadata.map(m => del(m.id, queryResultStore)))

    // Clear metadata
    await set('__metadata__', [], queryResultStore)
  } catch (error) {
    console.error('Failed to clear all results:', error)
  }
}

/**
 * Get storage statistics
 */
export async function getStorageStats(): Promise<{
  totalResults: number
  totalRows: number
  largeResults: number
  estimatedSizeMB: number
}> {
  try {
    const metadata = await getAllResultsMetadata()

    const totalResults = metadata.length
    const totalRows = metadata.reduce((sum, m) => sum + m.rowCount, 0)
    const largeResults = metadata.filter(m => m.isLarge).length

    // Rough estimate: 100 bytes per row on average
    const estimatedSizeMB = (totalRows * 100) / (1024 * 1024)

    return {
      totalResults,
      totalRows,
      largeResults,
      estimatedSizeMB,
    }
  } catch (error) {
    console.error('Failed to get storage stats:', error)
    return {
      totalResults: 0,
      totalRows: 0,
      largeResults: 0,
      estimatedSizeMB: 0,
    }
  }
}

// ============================================================================
// Internal Helper Functions
// ============================================================================

/**
 * Get all results metadata (for LRU eviction)
 */
async function getAllResultsMetadata(): Promise<QueryResultMetadata[]> {
  try {
    const metadata = await get<QueryResultMetadata[]>('__metadata__', queryResultStore)
    return metadata || []
  } catch {
    return []
  }
}

/**
 * Update results metadata (for LRU tracking)
 */
async function updateResultsMetadata(newMetadata: QueryResultMetadata): Promise<void> {
  try {
    const allMetadata = await getAllResultsMetadata()

    // Remove existing metadata for this result (if any)
    const filteredMetadata = allMetadata.filter(m => m.id !== newMetadata.id)

    // Add new metadata
    filteredMetadata.push(newMetadata)

    // Save back to IndexedDB
    await set('__metadata__', filteredMetadata, queryResultStore)
  } catch (error) {
    console.error('Failed to update results metadata:', error)
  }
}

/**
 * Update last accessed time for a result
 */
async function updateLastAccessed(resultId: string): Promise<void> {
  try {
    const allMetadata = await getAllResultsMetadata()
    const metadata = allMetadata.find(m => m.id === resultId)

    if (metadata) {
      metadata.lastAccessed = new Date()
      await set('__metadata__', allMetadata, queryResultStore)
    }
  } catch (error) {
    console.error('Failed to update last accessed:', error)
  }
}

/**
 * Remove metadata for a specific result
 */
async function removeResultMetadata(resultId: string): Promise<void> {
  try {
    const allMetadata = await getAllResultsMetadata()
    const filteredMetadata = allMetadata.filter(m => m.id !== resultId)
    await set('__metadata__', filteredMetadata, queryResultStore)
  } catch (error) {
    console.error('Failed to remove result metadata:', error)
  }
}

/**
 * Enforce maximum storage limit (LRU eviction)
 */
async function enforceStorageLimit(): Promise<void> {
  try {
    const allMetadata = await getAllResultsMetadata()

    if (allMetadata.length <= MAX_STORED_RESULTS) {
      return // Under limit, nothing to do
    }

    // Sort by last accessed (oldest first)
    const sorted = [...allMetadata].sort((a, b) =>
      a.lastAccessed.getTime() - b.lastAccessed.getTime()
    )

    // Calculate how many to evict
    const toEvict = sorted.length - MAX_STORED_RESULTS
    const evictIds = sorted.slice(0, toEvict).map(m => m.id)

    // Delete old results
    await Promise.all(evictIds.map(id => deleteQueryResult(id)))

    console.log(`[QueryResultStorage] Evicted ${toEvict} old results (LRU)`)
  } catch (error) {
    console.error('Failed to enforce storage limit:', error)
  }
}

/**
 * Check if a result is considered large
 */
export function isLargeResult(rowCount: number): boolean {
  return rowCount >= LARGE_RESULT_THRESHOLD
}

/**
 * Get the threshold for large results
 */
export function getLargeResultThreshold(): number {
  return LARGE_RESULT_THRESHOLD
}

// ============================================================================
// Chunked Data Loading (Phase 2)
// ============================================================================

// Chunk configuration
export const CHUNK_CONFIG = {
  CHUNK_SIZE: 500, // Rows per chunk
  MAX_CHUNKS_IN_MEMORY: 5, // Keep 2,500 rows max (5 * 500)
  PRELOAD_CHUNKS: 1, // Preload 1 chunk ahead/behind
} as const

// Feature flags
export const FEATURE_FLAGS = {
  ENABLE_CHUNKING: true, // ✅ ENABLED: ag-Grid-style chunked loading
  CHUNKING_THRESHOLD: 1000, // Enable for 1K+ rows (reduced from 5K)
  MAX_CLIENT_SORT_ROWS: 5000, // Disable client-side sorting above 5K rows
  WARN_THRESHOLD: 1000, // Show performance warning
} as const

/**
 * Display mode for large datasets
 */
export type DataMode = 'small' | 'large-disabled' | 'large-server'

export interface ResultDisplayMode {
  mode: DataMode
  canSort: boolean
  canFilter: boolean
  reason?: string
}

/**
 * Determine the appropriate display mode based on dataset size
 */
export function determineDisplayMode(
  rowCount: number,
  serverSortingAvailable: boolean
): ResultDisplayMode {
  if (rowCount < 100) {
    return {
      mode: 'small',
      canSort: true,
      canFilter: true,
    }
  }

  if (rowCount >= 100 && rowCount < 10000) {
    // Medium: Load all, but warn if > 1000
    return {
      mode: 'small',
      canSort: true,
      canFilter: true,
      reason: rowCount >= FEATURE_FLAGS.WARN_THRESHOLD
        ? 'Large dataset: sorting/filtering may be slow'
        : undefined,
    }
  }

  if (rowCount >= 10000 && !serverSortingAvailable) {
    return {
      mode: 'large-disabled',
      canSort: false,
      canFilter: false,
      reason: 'Sorting/filtering disabled for large datasets. Export data to sort/filter.',
    }
  }

  return {
    mode: 'large-server',
    canSort: true,
    canFilter: true,
    reason: 'Server-side sorting/filtering required',
  }
}

/**
 * Calculate which chunks are needed for a visible range
 */
export function calculateRequiredChunks(
  visibleStartIndex: number,
  visibleEndIndex: number,
  preloadCount: number = CHUNK_CONFIG.PRELOAD_CHUNKS
): number[] {
  const startChunk = Math.floor(visibleStartIndex / CHUNK_CONFIG.CHUNK_SIZE)
  const endChunk = Math.floor(visibleEndIndex / CHUNK_CONFIG.CHUNK_SIZE)

  const preloadStart = Math.max(0, startChunk - preloadCount)
  const preloadEnd = endChunk + preloadCount

  const chunks: number[] = []
  for (let i = preloadStart; i <= preloadEnd; i++) {
    chunks.push(i)
  }

  return chunks
}

/**
 * Load specific chunks from IndexedDB
 */
export async function loadChunks(
  resultId: string,
  chunkIndices: number[]
): Promise<Map<number, unknown[]>> {
  try {
    const result = await get<StoredQueryResult>(resultId, queryResultStore)
    if (!result) {
      return new Map()
    }

    const chunks = new Map<number, unknown[]>()

    for (const chunkIndex of chunkIndices) {
      const startRow = chunkIndex * CHUNK_CONFIG.CHUNK_SIZE
      const endRow = Math.min(
        startRow + CHUNK_CONFIG.CHUNK_SIZE,
        result.rows.length
      )

      const chunkRows = result.rows.slice(startRow, endRow)
      chunks.set(chunkIndex, chunkRows)
    }

    // Update last accessed time
    await updateLastAccessed(resultId)

    return chunks
  } catch (error) {
    console.error('Failed to load chunks:', error)
    return new Map()
  }
}
