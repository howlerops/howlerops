/**
 * Query History Store
 * Manages query results, history, and result metadata
 */

import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { useShallow } from 'zustand/react/shallow'

import {
  CHUNK_CONFIG,
  deleteTabResults,
  determineDisplayMode,
  FEATURE_FLAGS,
  isLargeResult,
  type StoredQueryResult,
  storeQueryResult,
} from '@/lib/query-result-storage'

import type { QueryEditableMetadata, QueryResult, QueryResultRow } from './query-types'
import { generateRowId } from './query-utils'

interface QueryHistoryState {
  results: QueryResult[]

  // Actions
  addResult: (result: Omit<QueryResult, 'id' | 'timestamp'>) => QueryResult
  clearResults: (tabId: string) => void
  clearAllResults: () => void
  updateResultRows: (resultId: string, rows: QueryResultRow[], newOriginalRows?: Record<string, QueryResultRow>) => void
  updateResultEditable: (resultId: string, metadata: QueryEditableMetadata | null) => void
  updateResultProcessing: (resultId: string, isProcessing: boolean, progress?: number) => void
  getResultsForTab: (tabId: string) => QueryResult[]
  getLatestResult: (tabId: string) => QueryResult | undefined
}

export const useQueryHistoryStore = create<QueryHistoryState>()(
  devtools(
    (set, get) => ({
      results: [],

      addResult: (resultData) => {
        const newResult: QueryResult = {
          ...resultData,
          id: crypto.randomUUID(),
          timestamp: new Date(),
        }

        const rowCount = newResult.rows.length
        const displayMode = determineDisplayMode(rowCount, false)
        const isLarge = isLargeResult(rowCount)
        const enableChunking = FEATURE_FLAGS.ENABLE_CHUNKING && rowCount >= FEATURE_FLAGS.CHUNKING_THRESHOLD

        // Store large results in IndexedDB with optional chunking
        // Only use IndexedDB if chunking is enabled OR result is truly massive (> 50K rows)
        if (isLarge && (enableChunking || rowCount > 50000)) {
          const storedResult: StoredQueryResult = {
            id: newResult.id,
            tabId: newResult.tabId,
            columns: newResult.columns,
            rows: newResult.rows,
            originalRows: newResult.originalRows,
            rowCount: newResult.rowCount,
            affectedRows: newResult.affectedRows,
            executionTime: newResult.executionTime,
            error: newResult.error,
            timestamp: newResult.timestamp,
            editable: newResult.editable,
            query: newResult.query,
            connectionId: newResult.connectionId,
          }

          // Store in IndexedDB asynchronously
          storeQueryResult(storedResult).catch((error) => {
            console.error('Failed to store large result in IndexedDB:', error)
          })

          // If chunking is enabled, keep only first chunk in memory
          if (enableChunking) {
            const firstChunk = newResult.rows.slice(0, CHUNK_CONFIG.CHUNK_SIZE)
            const firstChunkOriginalRows: Record<string, QueryResultRow> = {}
            firstChunk.forEach((row) => {
              firstChunkOriginalRows[row.__rowId] = newResult.originalRows[row.__rowId]
            })

            const resultWithMetadata: QueryResult = {
              ...newResult,
              isLarge: true,
              chunkingEnabled: true,
              loadedChunks: new Set([0]),
              totalChunks: Math.ceil(rowCount / CHUNK_CONFIG.CHUNK_SIZE),
              rowsLoaded: firstChunk.length,
              rows: firstChunk,
              originalRows: firstChunkOriginalRows,
              displayMode,
            }

            set((state) => ({
              results: [...state.results, resultWithMetadata].slice(-20),
            }))

            return resultWithMetadata
          }

          // Phase 1 behavior: Keep first 100 rows for preview (no chunking)
          const previewRows = newResult.rows.slice(0, 100)
          const previewOriginalRows: Record<string, QueryResultRow> = {}
          previewRows.forEach((row) => {
            previewOriginalRows[row.__rowId] = newResult.originalRows[row.__rowId]
          })

          const resultWithMetadata: QueryResult = {
            ...newResult,
            isLarge: true,
            chunkingEnabled: false,
            rowsLoaded: previewRows.length,
            rows: previewRows,
            originalRows: previewOriginalRows,
            displayMode,
          }

          set((state) => ({
            results: [...state.results, resultWithMetadata].slice(-20),
          }))

          return resultWithMetadata
        }

        // Small results: store normally in memory
        newResult.displayMode = displayMode
        newResult.chunkingEnabled = false

        set((state) => ({
          results: [...state.results, newResult].slice(-20),
        }))

        return newResult
      },

      clearResults: (tabId) => {
        // Clean up IndexedDB results for this tab
        deleteTabResults(tabId).catch((error) => {
          console.error('Failed to clear tab results from IndexedDB:', error)
        })

        set((state) => ({
          results: state.results.filter((result) => result.tabId !== tabId),
        }))
      },

      clearAllResults: () => {
        set({ results: [] })
      },

      updateResultRows: (resultId, rows, newOriginalRows) => {
        set((state) => ({
          results: state.results.map((result) => {
            if (result.id !== resultId) {
              return result
            }

            return {
              ...result,
              rows,
              originalRows: newOriginalRows ?? result.originalRows,
            }
          }),
        }))
      },

      updateResultEditable: (resultId, metadata) => {
        set((state) => ({
          results: state.results.map((result) => {
            if (result.id !== resultId) {
              return result
            }

            const normalizedMetadata = metadata
              ? {
                  ...metadata,
                  primaryKeys: [...(metadata.primaryKeys || [])],
                  columns: (metadata.columns || []).map((column) => ({ ...column })),
                  jobId: metadata.jobId || metadata.job_id,
                  job_id: metadata.jobId || metadata.job_id,
                }
              : null

            let updatedRows = result.rows
            let updatedOriginalRows = result.originalRows

            if (normalizedMetadata && !normalizedMetadata.pending && normalizedMetadata.primaryKeys.length > 0) {
              const columnLookup: Record<string, string> = {}
              result.columns.forEach((name) => {
                columnLookup[name.toLowerCase()] = name
              })

              const pkColumns = normalizedMetadata.primaryKeys.map((pk) => columnLookup[pk.toLowerCase()] ?? pk)

              const recomputedRows: QueryResultRow[] = []
              const recomputedOriginal: Record<string, QueryResultRow> = {}

              result.rows.forEach((row, index) => {
                const existingOriginal = result.originalRows[row.__rowId] ?? row
                const nextRow: QueryResultRow = { ...row }

                let rowId = ''
                if (pkColumns.length > 0) {
                  const parts: string[] = []
                  let allPresent = true
                  pkColumns.forEach((pkColumn) => {
                    const value = nextRow[pkColumn]
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

                if (!rowId) {
                  rowId = `${generateRowId()}-${index}`
                }

                nextRow.__rowId = rowId
                recomputedRows.push(nextRow)
                recomputedOriginal[rowId] = { ...(existingOriginal as QueryResultRow), __rowId: rowId }
              })

              updatedRows = recomputedRows
              updatedOriginalRows = recomputedOriginal
            }

            return {
              ...result,
              editable: normalizedMetadata,
              rows: updatedRows,
              originalRows: updatedOriginalRows,
            }
          }),
        }))
      },

      updateResultProcessing: (resultId, isProcessing, progress) => {
        set((state) => ({
          results: state.results.map((result) => {
            if (result.id !== resultId) {
              return result
            }

            return {
              ...result,
              isProcessing,
              processingProgress: progress,
            }
          }),
        }))
      },

      getResultsForTab: (tabId) => {
        return get().results.filter((result) => result.tabId === tabId)
      },

      getLatestResult: (tabId) => {
        const tabResults = get().results.filter((result) => result.tabId === tabId)
        return tabResults.length > 0 ? tabResults[tabResults.length - 1] : undefined
      },
    }),
    {
      name: 'query-history-store',
    }
  )
)

// Selectors with useShallow for efficient subscriptions
export const useQueryResults = () =>
  useQueryHistoryStore(useShallow((state) => state.results))

export const useTabResults = (tabId: string) =>
  useQueryHistoryStore(useShallow((state) => state.results.filter((r) => r.tabId === tabId)))

export const useLatestTabResult = (tabId: string) =>
  useQueryHistoryStore(
    useShallow((state) => {
      const tabResults = state.results.filter((r) => r.tabId === tabId)
      return tabResults.length > 0 ? tabResults[tabResults.length - 1] : undefined
    })
  )

export const useQueryHistoryActions = () =>
  useQueryHistoryStore(
    useShallow((state) => ({
      addResult: state.addResult,
      clearResults: state.clearResults,
      clearAllResults: state.clearAllResults,
      updateResultRows: state.updateResultRows,
      updateResultEditable: state.updateResultEditable,
      updateResultProcessing: state.updateResultProcessing,
    }))
  )

// Re-export types
export type { QueryEditableMetadata, QueryResult, QueryResultRow } from './query-types'
