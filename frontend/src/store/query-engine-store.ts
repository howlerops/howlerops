/**
 * Query Engine Store
 * Unified runtime state layer for frontend query execution.
 *
 * This store owns execution runtime state and delegates result shaping to the
 * pure TypeScript query engine.
 */

import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { useShallow } from 'zustand/react/shallow'

import { api } from '@/lib/api-client'
import {
  createErrorQueryResult,
  prepareLoadMoreResult,
  prepareQueryExecutionResult,
  resolveQueryConnection,
} from '@/lib/query-engine/engine'

import { useConnectionStore } from './connection-store'
import { useQueryEditorStore } from './query-editor-store'
import { useQueryHistoryStore } from './query-history-store'
import type { QueryEditableMetadata } from './query-types'
import { transformEditableMetadata } from './query-utils'

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

interface QueryEngineState {
  executingQueries: Map<string, { startTime: Date; query: string }>
  executeQuery: (tabId: string, query: string, connectionId?: string | null, limit?: number, offset?: number) => Promise<void>
  loadMoreRows: (resultId: string) => Promise<void>
  cancelQuery: (tabId: string) => void
}

export const useQueryEngineStore = create<QueryEngineState>()(
  devtools(
    (set) => {
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

          const tab = editorStore.tabs.find((candidate) => candidate.id === tabId)
          if (!tab || tab.type !== 'sql') {
            return
          }

          editorStore.updateTab(tabId, { isExecuting: true, executionStartTime: new Date() })

          set((state) => {
            const next = new Map(state.executingQueries)
            next.set(tabId, { startTime: new Date(), query })
            return { executingQueries: next }
          })

          const resolved = resolveQueryConnection(
            useConnectionStore.getState().connections,
            connectionId || tab.connectionId
          )

          if ('ok' in resolved) {
            historyStore.addResult(createErrorQueryResult({ tabId, query }, resolved.error, resolved.connectionId))
            editorStore.updateTab(tabId, { isExecuting: false, executionStartTime: undefined })
            set((state) => {
              const next = new Map(state.executingQueries)
              next.delete(tabId)
              return { executingQueries: next }
            })
            return
          }

          try {
            const response = await api.queries.execute(
              resolved.sessionId,
              query,
              limit,
              offset
            )

            const prepared = prepareQueryExecutionResult(
              {
                tabId,
                query,
                connectionId: resolved.connectionId,
                sessionId: resolved.sessionId,
                limit,
                offset,
              },
              response
            )

            if (!prepared.ok) {
              historyStore.addResult(createErrorQueryResult({ tabId, query }, prepared.error, prepared.connectionId))
              return
            }

            const savedResult = historyStore.addResult(prepared.result)
            const jobId = prepared.editable?.jobId || prepared.editable?.job_id
            if (prepared.editable?.pending && jobId) {
              scheduleEditablePoll(jobId, savedResult.id)
            }

            editorStore.updateTab(tabId, {
              lastExecuted: new Date(),
              isDirty: false,
            })
          } catch (error) {
            historyStore.addResult(
              createErrorQueryResult(
                { tabId, query },
                error instanceof Error ? error.message : 'Unknown error occurred',
                resolved.connectionId
              )
            )
          } finally {
            editorStore.updateTab(tabId, { isExecuting: false, executionStartTime: undefined })
            set((state) => {
              const next = new Map(state.executingQueries)
              next.delete(tabId)
              return { executingQueries: next }
            })
          }
        },

        loadMoreRows: async (resultId) => {
          const historyStore = useQueryHistoryStore.getState()
          const result = historyStore.results.find((candidate) => candidate.id === resultId)
          if (!result || !result.hasMore || !result.connectionId) {
            return
          }

          const resolved = resolveQueryConnection(
            useConnectionStore.getState().connections,
            result.connectionId
          )
          if ('ok' in resolved) {
            return
          }

          const currentOffset = result.offset ?? 0
          const pageSize = result.limit ?? 5000
          const nextOffset = currentOffset + pageSize

          try {
            historyStore.updateResultProcessing(resultId, true, 0)

            const response = await api.queries.execute(
              resolved.sessionId,
              result.query,
              pageSize,
              nextOffset
            )

            const prepared = prepareLoadMoreResult(result, response, nextOffset)
            if ('ok' in prepared) {
              historyStore.updateResultProcessing(resultId, false, 0)
              return
            }

            const fresh = historyStore.results.find((candidate) => candidate.id === resultId)
            if (!fresh) {
              historyStore.updateResultProcessing(resultId, false, 0)
              return
            }

            historyStore.updateResultRows(
              resultId,
              [...fresh.rows, ...prepared.rows],
              { ...fresh.originalRows, ...prepared.originalRows }
            )

            useQueryHistoryStore.setState((state) => ({
              results: state.results.map((candidate) => {
                if (candidate.id !== resultId) return candidate
                return {
                  ...candidate,
                  offset: prepared.offset,
                  hasMore: prepared.hasMore ?? false,
                  pagedRows: prepared.pagedRows ?? prepared.rows.length,
                  totalRows: prepared.totalRows ?? candidate.totalRows,
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
            const next = new Map(state.executingQueries)
            next.delete(tabId)
            return { executingQueries: next }
          })
        },
      }
    },
    { name: 'query-engine-store' }
  )
)

const hasWailsRuntime =
  typeof window !== 'undefined' &&
  typeof (window as { runtime?: { EventsOn?: unknown } }).runtime?.EventsOn === 'function'

if (hasWailsRuntime) {
  Events.On('query:editableMetadata', (payload: unknown) => {
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

export const useExecutingQueries = () =>
  useQueryEngineStore(useShallow((state) => state.executingQueries))

export const useIsExecuting = (tabId: string) =>
  useQueryEngineStore((state) => state.executingQueries.has(tabId))

export const useQueryEngineActions = () =>
  useQueryEngineStore(
    useShallow((state) => ({
      executeQuery: state.executeQuery,
      loadMoreRows: state.loadMoreRows,
      cancelQuery: state.cancelQuery,
    }))
  )
