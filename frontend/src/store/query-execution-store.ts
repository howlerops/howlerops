/**
 * Query Execution Store
 * Manages query execution state: running queries, results processing, errors
 */

import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { useShallow } from 'zustand/react/shallow'

import { api } from '@/lib/api-client'

import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useConnectionStore } from './connection-store'
import { useQueryEditorStore } from './query-editor-store'
import { useQueryHistoryStore } from './query-history-store'
import type { QueryEditableMetadata } from './query-types'
import { normaliseRows, parseDurationMs, transformEditableMetadata } from './query-utils'

// Editable metadata polling configuration
const MAX_EDITABLE_METADATA_ATTEMPTS = 20
const editableMetadataTimers = new Map<string, number>()
const editableMetadataTargets = new Map<string, string>()

function cleanupEditableMetadataJob(jobId: string) {
  const timer = editableMetadataTimers.get(jobId)
  if (timer) {
    clearTimeout(timer)
    editableMetadataTimers.delete(jobId)
  }
  editableMetadataTargets.delete(jobId)
}

interface QueryExecutionState {
  // Execution tracking (could be expanded for concurrent query tracking)
  executingQueries: Map<string, { startTime: Date; query: string }>

  // Actions
  executeQuery: (tabId: string, query: string, connectionId?: string | null, limit?: number, offset?: number) => Promise<void>
  loadMoreRows: (resultId: string) => Promise<void>
  cancelQuery: (tabId: string) => void
}

export const useQueryExecutionStore = create<QueryExecutionState>()(
  devtools(
    (set, get) => {
      // Schedule polling for editable metadata
      const scheduleEditablePoll = (jobId: string, resultId: string, attempt = 0) => {
        const historyStore = useQueryHistoryStore.getState()
        const resultExists = historyStore.results.some(result => result.id === resultId)
        if (!resultExists) {
          cleanupEditableMetadataJob(jobId)
          return
        }

        const delay = Math.min(1000, 250 * Math.max(1, attempt + 1))

        const timer = window.setTimeout(async () => {
          try {
            const response = await api.queries.getEditableMetadata(jobId)

            if (!response.success || !response.data) {
              if (attempt + 1 >= MAX_EDITABLE_METADATA_ATTEMPTS) {
                cleanupEditableMetadataJob(jobId)
                historyStore.updateResultEditable(resultId, {
                  enabled: false,
                  reason: response.message || 'Editable metadata unavailable',
                  schema: undefined,
                  table: undefined,
                  primaryKeys: [],
                  columns: [],
                  pending: false,
                  jobId,
                  job_id: jobId,
                })
                return
              }

              scheduleEditablePoll(jobId, resultId, attempt + 1)
              return
            }

            const jobData = response.data as { status?: string; metadata?: unknown; error?: string; id?: string }
            const status = (jobData.status || '').toLowerCase()

            if (status === 'completed' && jobData.metadata) {
              const metadata = transformEditableMetadata(jobData.metadata)
              if (metadata) {
                metadata.pending = false
                metadata.jobId = metadata.jobId || jobData.id || jobId
                metadata.job_id = metadata.jobId
              }

              cleanupEditableMetadataJob(jobId)
              historyStore.updateResultEditable(resultId, metadata)
              return
            }

            if (status === 'failed') {
              const metadata = transformEditableMetadata(jobData.metadata) || {
                enabled: false,
                reason: jobData.error || 'Editable metadata unavailable',
                schema: undefined,
                table: undefined,
                primaryKeys: [],
                columns: [],
                pending: false,
                jobId: jobData.id || jobId,
                job_id: jobData.id || jobId,
              }

              metadata.pending = false
              metadata.reason = jobData.error || metadata.reason
              metadata.jobId = metadata.jobId || jobData.id || jobId
              metadata.job_id = metadata.jobId

              cleanupEditableMetadataJob(jobId)
              historyStore.updateResultEditable(resultId, metadata)
              return
            }

            if (attempt + 1 >= MAX_EDITABLE_METADATA_ATTEMPTS) {
              const fallback = transformEditableMetadata(jobData.metadata) || {
                enabled: false,
                primaryKeys: [],
                columns: [],
              } as QueryEditableMetadata

              fallback.pending = false
              fallback.reason = jobData.error || fallback.reason || 'Editable metadata timed out'
              fallback.jobId = fallback.jobId || jobData.id || jobId
              fallback.job_id = fallback.jobId

              cleanupEditableMetadataJob(jobId)
              historyStore.updateResultEditable(resultId, fallback)
              return
            }

            scheduleEditablePoll(jobId, resultId, attempt + 1)
          } catch (pollError) {
            if (attempt + 1 >= MAX_EDITABLE_METADATA_ATTEMPTS) {
              cleanupEditableMetadataJob(jobId)
              historyStore.updateResultEditable(resultId, {
                enabled: false,
                reason: pollError instanceof Error ? pollError.message : 'Editable metadata unavailable',
                schema: undefined,
                table: undefined,
                primaryKeys: [],
                columns: [],
                pending: false,
                jobId,
                job_id: jobId,
              })
              return
            }

            scheduleEditablePoll(jobId, resultId, attempt + 1)
          }
        }, delay)

        const existingTimer = editableMetadataTimers.get(jobId)
        if (existingTimer) {
          clearTimeout(existingTimer)
        }

        editableMetadataTimers.set(jobId, timer)
        editableMetadataTargets.set(jobId, resultId)
      }

      return {
        executingQueries: new Map(),

        executeQuery: async (tabId, query, connectionId, limit = 5000, offset = 0) => {
          const editorStore = useQueryEditorStore.getState()
          const historyStore = useQueryHistoryStore.getState()

          const tab = editorStore.tabs.find(t => t.id === tabId)
          if (!tab || tab.type !== 'sql') {
            return
          }

          editorStore.updateTab(tabId, { isExecuting: true, executionStartTime: new Date() })

          // Track executing query
          set((state) => {
            const newMap = new Map(state.executingQueries)
            newMap.set(tabId, { startTime: new Date(), query })
            return { executingQueries: newMap }
          })

          // Use tab's connection if no connectionId provided
          const effectiveConnectionId = connectionId || tab.connectionId

          if (!effectiveConnectionId) {
            historyStore.addResult({
              tabId,
              columns: [],
              rows: [],
              originalRows: {},
              rowCount: 0,
              affectedRows: 0,
              executionTime: 0,
              error: 'No connection selected for this tab',
              editable: null,
              query,
            })
            editorStore.updateTab(tabId, { isExecuting: false })
            set((state) => {
              const newMap = new Map(state.executingQueries)
              newMap.delete(tabId)
              return { executingQueries: newMap }
            })
            return
          }

          // Get the actual session ID from the connection store
          const { connections } = useConnectionStore.getState()
          const connection = connections.find(conn => conn.id === effectiveConnectionId)

          if (!connection?.sessionId) {
            historyStore.addResult({
              tabId,
              columns: [],
              rows: [],
              originalRows: {},
              rowCount: 0,
              affectedRows: 0,
              executionTime: 0,
              error: 'Connection not established. Please connect to the database first.',
              editable: null,
              query,
            })
            editorStore.updateTab(tabId, { isExecuting: false })
            set((state) => {
              const newMap = new Map(state.executingQueries)
              newMap.delete(tabId)
              return { executingQueries: newMap }
            })
            return
          }

          try {
            const response = await api.queries.execute(connection.sessionId, query, limit, offset)

            if (!response.success || !response.data) {
              const message = response.message || 'Query execution failed'
              historyStore.addResult({
                tabId,
                columns: [],
                rows: [],
                originalRows: {},
                rowCount: 0,
                affectedRows: 0,
                executionTime: 0,
                error: message,
                editable: null,
                query,
                connectionId: effectiveConnectionId,
              })
              return
            }

            const {
              columns: rawColumns = [],
              rows = [],
              rowCount = 0,
              stats = {},
              editable: rawEditable = null,
            } = response.data

            // Convert QueryColumn[] to string[] for column names
            const columns = rawColumns.map((col: unknown) =>
              typeof col === 'string' ? col : (col as { name: string }).name
            )

            // Extract pagination metadata
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendTotalRows = (response.data as any).totalRows
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendPagedRows = (response.data as any).pagedRows
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendHasMore = (response.data as any).hasMore
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendOffset = (response.data as any).offset

            // Extract multi-database query metadata
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const connectionsUsed = (response.data as any).connectionsUsed as string[] | undefined

            const statsRecord = (stats ?? {}) as Record<string, unknown>
            const affectedRows =
              typeof statsRecord.affectedRows === 'number'
                ? statsRecord.affectedRows
                : typeof statsRecord.affected_rows === 'number'
                  ? statsRecord.affected_rows
                  : 0
            const durationValue =
              typeof statsRecord.duration === 'string'
                ? statsRecord.duration
                : undefined

            const editableMetadata = transformEditableMetadata(rawEditable)

            // Process rows synchronously (Go backend already normalized data)
            const { rows: normalisedRows, originalRows } = normaliseRows(columns, rows, editableMetadata)

            const savedResult = historyStore.addResult({
              tabId,
              columns,
              rows: normalisedRows,
              originalRows,
              rowCount: rowCount || normalisedRows.length,
              affectedRows,
              executionTime: parseDurationMs(durationValue),
              error: undefined,
              editable: editableMetadata,
              query,
              connectionId: effectiveConnectionId,
              // Pagination metadata
              totalRows: typeof backendTotalRows === 'number' ? backendTotalRows : undefined,
              pagedRows: typeof backendPagedRows === 'number' ? backendPagedRows : undefined,
              hasMore: typeof backendHasMore === 'boolean' ? backendHasMore : undefined,
              offset: typeof backendOffset === 'number' ? backendOffset : offset,
              limit,
              // Multi-database query metadata
              connectionsUsed,
            })

            const jobId = editableMetadata?.jobId || editableMetadata?.job_id
            if (editableMetadata?.pending && jobId) {
              scheduleEditablePoll(jobId, savedResult.id)
            }

            editorStore.updateTab(tabId, {
              lastExecuted: new Date(),
              isDirty: false,
            })
          } catch (error) {
            historyStore.addResult({
              tabId,
              columns: [],
              rows: [],
              originalRows: {},
              rowCount: 0,
              affectedRows: 0,
              executionTime: 0,
              error: error instanceof Error ? error.message : 'Unknown error occurred',
              editable: null,
              query,
              connectionId: effectiveConnectionId,
            })
          } finally {
            editorStore.updateTab(tabId, { isExecuting: false, executionStartTime: undefined })
            set((state) => {
              const newMap = new Map(state.executingQueries)
              newMap.delete(tabId)
              return { executingQueries: newMap }
            })
          }
        },

        loadMoreRows: async (resultId) => {
          const historyStore = useQueryHistoryStore.getState()
          const result = historyStore.results.find((r) => r.id === resultId)

          if (!result || !result.hasMore || !result.connectionId) {
            return
          }

          const currentOffset = result.offset ?? 0
          const pageSize = result.limit ?? 5000
          const nextOffset = currentOffset + pageSize

          // Get the connection session ID
          const { connections } = useConnectionStore.getState()
          const connection = connections.find((conn) => conn.id === result.connectionId)

          if (!connection?.sessionId) {
            console.error('Connection not found for loadMoreRows')
            return
          }

          try {
            // Set loading state
            historyStore.updateResultProcessing(resultId, true, 0)

            const response = await api.queries.execute(
              connection.sessionId,
              result.query,
              pageSize,
              nextOffset
            )

            if (!response.success || !response.data) {
              console.error('Failed to load more rows:', response.message)
              historyStore.updateResultProcessing(resultId, false, 0)
              return
            }

            const { rows = [] } = response.data

            // Extract pagination metadata
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendTotalRows = (response.data as any).totalRows
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendPagedRows = (response.data as any).pagedRows
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendHasMore = (response.data as any).hasMore
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const backendOffset = (response.data as any).offset

            // Process new rows
            const { rows: normalisedRows, originalRows: newOriginalRows } = normaliseRows(
              result.columns,
              rows,
              result.editable
            )

            // Get fresh result state and append new rows
            const freshResult = historyStore.results.find((r) => r.id === resultId)
            if (!freshResult) {
              historyStore.updateResultProcessing(resultId, false, 0)
              return
            }

            const updatedRows = [...freshResult.rows, ...normalisedRows]
            const updatedOriginalRows = { ...freshResult.originalRows, ...newOriginalRows }

            // Update result with new data using direct state update
            historyStore.updateResultRows(resultId, updatedRows, updatedOriginalRows)

            // Update pagination metadata separately via the store's internal state
            // This requires accessing the store directly since we need to update multiple fields
            useQueryHistoryStore.setState((state) => ({
              results: state.results.map((r) => {
                if (r.id !== resultId) return r
                return {
                  ...r,
                  offset: typeof backendOffset === 'number' ? backendOffset : nextOffset,
                  hasMore: typeof backendHasMore === 'boolean' ? backendHasMore : false,
                  pagedRows: typeof backendPagedRows === 'number' ? backendPagedRows : rows.length,
                  totalRows: typeof backendTotalRows === 'number' ? backendTotalRows : r.totalRows,
                  isProcessing: false,
                  processingProgress: 0,
                }
              }),
            }))
          } catch (error) {
            console.error('Error loading more rows:', error)
            historyStore.updateResultProcessing(resultId, false, 0)
          }
        },

        cancelQuery: (tabId) => {
          const editorStore = useQueryEditorStore.getState()
          editorStore.updateTab(tabId, { isExecuting: false, executionStartTime: undefined })
          set((state) => {
            const newMap = new Map(state.executingQueries)
            newMap.delete(tabId)
            return { executingQueries: newMap }
          })
        },
      }
    },
    {
      name: 'query-execution-store',
    }
  )
)

// Set up Wails runtime event listener for editable metadata
const hasWailsRuntime =
  typeof window !== 'undefined' &&
  typeof (window as { runtime?: { EventsOnMultiple?: unknown } }).runtime?.EventsOnMultiple === 'function'

if (hasWailsRuntime) {
  EventsOn('query:editableMetadata', (payload: unknown) => {
    try {
      const data = (payload ?? {}) as Record<string, unknown>
      const jobId = (data.jobId as string) ?? (data.job_id as string)
      if (!jobId) {
        return
      }

      const resultId = editableMetadataTargets.get(jobId)
      if (!resultId) {
        cleanupEditableMetadataJob(jobId)
        return
      }

      const status = String(data.status ?? '').toLowerCase()
      const metadataPayload = data.metadata
      const errorMessage = (data.error as string) || ''

      const historyStore = useQueryHistoryStore.getState()
      const resultExists = historyStore.results.some(result => result.id === resultId)
      if (!resultExists) {
        cleanupEditableMetadataJob(jobId)
        return
      }

      const applyMetadata = (metadata: QueryEditableMetadata | null) => {
        cleanupEditableMetadataJob(jobId)
        historyStore.updateResultEditable(resultId, metadata)
      }

      if (status === 'completed') {
        const metadata = transformEditableMetadata(metadataPayload)
        if (metadata) {
          metadata.pending = false
          metadata.jobId = metadata.jobId || jobId
          metadata.job_id = metadata.jobId
        }
        applyMetadata(metadata ?? {
          enabled: false,
          reason: 'Editable metadata unavailable',
          schema: undefined,
          table: undefined,
          primaryKeys: [],
          columns: [],
          pending: false,
          jobId,
          job_id: jobId,
        })
        return
      }

      if (status === 'failed') {
        const metadata = transformEditableMetadata(metadataPayload) ?? {
          enabled: false,
          reason: errorMessage || 'Editable metadata unavailable',
          schema: undefined,
          table: undefined,
          primaryKeys: [],
          columns: [],
          pending: false,
          jobId,
          job_id: jobId,
        }

        metadata.pending = false
        metadata.reason = errorMessage || metadata.reason
        metadata.jobId = metadata.jobId || jobId
        metadata.job_id = metadata.jobId

        applyMetadata(metadata)
        return
      }

      if (status === 'pending') {
        const metadata = transformEditableMetadata(metadataPayload) ?? {
          enabled: false,
          reason: errorMessage || 'Loading editable metadata',
          schema: undefined,
          table: undefined,
          primaryKeys: [],
          columns: [],
          pending: true,
          jobId,
          job_id: jobId,
        }

        metadata.pending = true
        metadata.reason = errorMessage || metadata.reason || 'Loading editable metadata'
        metadata.jobId = metadata.jobId || jobId
        metadata.job_id = metadata.jobId

        historyStore.updateResultEditable(resultId, metadata)
        editableMetadataTargets.set(jobId, resultId)
        return
      }

      // Unknown status, treat as failure
      const metadata = transformEditableMetadata(metadataPayload) ?? {
        enabled: false,
        reason: errorMessage || 'Editable metadata unavailable',
        schema: undefined,
        table: undefined,
        primaryKeys: [],
        columns: [],
        pending: false,
        jobId,
        job_id: jobId,
      }
      metadata.pending = false
      metadata.jobId = metadata.jobId || jobId
      metadata.job_id = metadata.jobId

      applyMetadata(metadata)
    } catch (eventError) {
      console.error('Failed to process editable metadata event:', eventError)
    }
  })
}

// Selectors
export const useExecutingQueries = () =>
  useQueryExecutionStore(useShallow((state) => state.executingQueries))

export const useIsExecuting = (tabId: string) =>
  useQueryExecutionStore((state) => state.executingQueries.has(tabId))

export const useQueryExecutionActions = () =>
  useQueryExecutionStore(
    useShallow((state) => ({
      executeQuery: state.executeQuery,
      loadMoreRows: state.loadMoreRows,
      cancelQuery: state.cancelQuery,
    }))
  )
