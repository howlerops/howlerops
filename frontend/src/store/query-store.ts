import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { wailsEndpoints } from '@/lib/wails-api'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useConnectionStore, type DatabaseConnection } from './connection-store'

export type QueryTabType = 'sql' | 'ai'

export interface QueryTab {
  id: string
  title: string
  type: QueryTabType
  content: string
  isDirty: boolean
  isExecuting: boolean
  lastExecuted?: Date
  connectionId?: string // Per-tab connection support (single-DB mode)
  selectedConnectionIds?: string[] // Multi-select connections (multi-DB mode)
  environmentSnapshot?: string | null // Capture environment filter at creation
  aiSessionId?: string
}

export interface QueryEditableColumn {
  name: string
  resultName: string
  dataType: string
  editable: boolean
  primaryKey: boolean
  foreignKey?: {
    table: string
    column: string
    schema?: string
  }
}

export interface QueryEditableMetadata {
  enabled: boolean
  reason?: string
  schema?: string
  table?: string
  primaryKeys: string[]
  columns: QueryEditableColumn[]
  pending?: boolean
  jobId?: string
  job_id?: string
}

export interface QueryResultRow extends Record<string, unknown> {
  __rowId: string
}

export interface QueryResult {
  id: string
  tabId: string
  columns: string[]
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
  rowCount: number
  affectedRows: number
  executionTime: number
  error?: string
  timestamp: Date
  editable?: QueryEditableMetadata | null
  query: string
  connectionId?: string
}

interface QueryState {
  tabs: QueryTab[]
  activeTabId: string | null
  results: QueryResult[]

  // Actions
  createTab: (title?: string, options?: { connectionId?: string; type?: QueryTabType; aiSessionId?: string }) => string
  closeTab: (id: string) => void
  updateTab: (id: string, updates: Partial<QueryTab>) => void
  setActiveTab: (id: string) => void
  executeQuery: (tabId: string, query: string, connectionId?: string | null) => Promise<void>
  addResult: (result: Omit<QueryResult, 'id' | 'timestamp'>) => QueryResult
  clearResults: (tabId: string) => void
  updateResultRows: (resultId: string, rows: QueryResultRow[], newOriginalRows?: Record<string, QueryResultRow>) => void
  updateResultEditable: (resultId: string, metadata: QueryEditableMetadata | null) => void
}

interface NormalisedRowsResult {
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
}

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

function transformEditableColumn(raw: unknown): QueryEditableColumn {
  if (!raw || typeof raw !== 'object') {
    return {
      name: '',
      resultName: '',
      dataType: '',
      editable: false,
      primaryKey: false,
    }
  }

  const column = raw as Record<string, unknown>
  const name = typeof column.name === 'string'
    ? column.name
    : typeof column.Name === 'string'
      ? column.Name
      : ''

  const resultName = typeof column.resultName === 'string'
    ? column.resultName
    : typeof column.result_name === 'string'
      ? column.result_name
      : name

  return {
    name,
    resultName,
    dataType: typeof column.dataType === 'string'
      ? column.dataType
      : typeof column.data_type === 'string'
        ? column.data_type
        : '',
    editable: Boolean(column.editable),
    primaryKey: Boolean(column.primaryKey ?? column.primary_key),
  }
}

function transformEditableMetadata(raw: unknown): QueryEditableMetadata | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }

  const metadataRaw = raw as Record<string, unknown>

  const primaryKeys = Array.isArray(metadataRaw.primaryKeys)
    ? metadataRaw.primaryKeys.filter((value): value is string => typeof value === 'string')
    : Array.isArray(metadataRaw.primary_keys)
      ? metadataRaw.primary_keys.filter((value): value is string => typeof value === 'string')
      : []

  const metadata: QueryEditableMetadata = {
    enabled: Boolean(metadataRaw.enabled),
    reason: typeof metadataRaw.reason === 'string' ? metadataRaw.reason : undefined,
    schema: typeof metadataRaw.schema === 'string' ? metadataRaw.schema : undefined,
    table: typeof metadataRaw.table === 'string' ? metadataRaw.table : undefined,
    primaryKeys,
    columns: Array.isArray(metadataRaw.columns) ? metadataRaw.columns.map(transformEditableColumn) : [],
    pending: Boolean(metadataRaw.pending),
    jobId: (metadataRaw.jobId as string | undefined) ?? (metadataRaw.job_id as string | undefined),
    job_id: (metadataRaw.job_id as string | undefined) ?? (metadataRaw.jobId as string | undefined),
  }

  if (!metadata.primaryKeys.length && Array.isArray(metadataRaw.primary_keys)) {
    metadata.primaryKeys = metadataRaw.primary_keys.filter((value): value is string => typeof value === 'string')
  }

  if (!metadata.jobId && metadata.job_id) {
    metadata.jobId = metadata.job_id
  }
  if (!metadata.job_id && metadata.jobId) {
    metadata.job_id = metadata.jobId
  }

  return metadata
}

function generateRowId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function normaliseRows(
  columns: string[],
  rows: unknown[],
  metadata?: QueryEditableMetadata | null
): NormalisedRowsResult {
  if (!Array.isArray(rows)) {
    return {
      rows: [],
      originalRows: {},
    }
  }

  const processedRows: QueryResultRow[] = []
  const originalRows: Record<string, QueryResultRow> = {}

  const columnLookup: Record<string, string> = {}
  columns.forEach((name) => {
    columnLookup[name.toLowerCase()] = name
  })

  const primaryKeyColumns = (metadata?.primaryKeys || []).map((pk) => {
    return columnLookup[pk.toLowerCase()] ?? pk
  })

  const assignValue = (target: Record<string, unknown>, columnName: string, value: unknown) => {
    if (value && typeof value === 'object' && 'String' in value && 'Valid' in value) {
      const sqlValue = value as { String: unknown; Valid: boolean }
      target[columnName] = sqlValue.Valid ? sqlValue.String : null
    } else {
      target[columnName] = value
    }
  }

  rows.forEach((row, rowIndex) => {
    const record: Record<string, unknown> = {}

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

    let rowId = ''
    if (primaryKeyColumns.length > 0) {
      const parts: string[] = []
      let allPresent = true
      primaryKeyColumns.forEach((pkColumn) => {
        const value = record[pkColumn]
        if (value === undefined) {
          allPresent = false
        } else {
          const serialised =
            value === null || value === undefined ? 'NULL' : String(value)
          parts.push(`${pkColumn}:${serialised}`)
        }
      })
      if (allPresent && parts.length > 0) {
        rowId = parts.join('|')
      }
    }

    if (!rowId) {
      rowId = `${generateRowId()}-${rowIndex}`
    }

    const completeRow: QueryResultRow = {
      ...record,
      __rowId: rowId,
    }

    processedRows.push(completeRow)
    originalRows[rowId] = { ...completeRow }
  })

  return {
    rows: processedRows,
    originalRows,
  }
}

function parseDurationMs(duration?: string): number {
  if (!duration) return 0
  const value = duration.toLowerCase()

  if (value.endsWith('ms')) {
    return parseFloat(value.replace('ms', ''))
  }
  if (value.endsWith('s')) {
    return parseFloat(value.replace('s', '')) * 1000
  }
  if (value.endsWith('µs') || value.endsWith('us')) {
    return parseFloat(value.replace('µs', '').replace('us', '')) / 1000
  }
  if (value.endsWith('ns')) {
    return parseFloat(value.replace('ns', '')) / 1e6
  }

  const parsed = parseFloat(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

export const useQueryStore = create<QueryState>()(
  devtools(
    persist(
      (set, get) => {
    const scheduleEditablePoll = (jobId: string, resultId: string, attempt = 0) => {
      const resultExists = get().results.some(result => result.id === resultId)
      if (!resultExists) {
        cleanupEditableMetadataJob(jobId)
        return
      }

      const delay = Math.min(1000, 250 * Math.max(1, attempt + 1))

      const timer = window.setTimeout(async () => {
        try {
          const response = await wailsEndpoints.queries.getEditableMetadata(jobId)

          if (!response.success || !response.data) {
            if (attempt + 1 >= MAX_EDITABLE_METADATA_ATTEMPTS) {
              cleanupEditableMetadataJob(jobId)
              get().updateResultEditable(resultId, {
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
            get().updateResultEditable(resultId, metadata)
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
            get().updateResultEditable(resultId, metadata)
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
            get().updateResultEditable(resultId, fallback)
            return
          }

          scheduleEditablePoll(jobId, resultId, attempt + 1)
        } catch (pollError) {
          if (attempt + 1 >= MAX_EDITABLE_METADATA_ATTEMPTS) {
            cleanupEditableMetadataJob(jobId)
            get().updateResultEditable(resultId, {
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
      tabs: [],
      activeTabId: null,
      results: [],

      createTab: (title = 'New Query', options?: { connectionId?: string; type?: QueryTabType; aiSessionId?: string }) => {
        const desiredType = options?.type ?? 'sql'
        let initialConnectionId = options?.connectionId
        let environmentSnapshot: string | null = null

        // Get connection state for both connectionId and selectedConnectionIds
        const connectionState = window.__connectionStore?.getState?.()
        
        if (!initialConnectionId) {
          if (connectionState) {
            const { connections, activeConnection, activeEnvironmentFilter } = connectionState
            environmentSnapshot = activeEnvironmentFilter

            if (activeConnection) {
              initialConnectionId = activeConnection.id
            } else if (connections.length > 0) {
              const firstConnected = connections.find((c: DatabaseConnection) => c.isConnected)
              if (firstConnected) {
                initialConnectionId = firstConnected.id
              }
            }
          }
        }

        const newTab: QueryTab = {
          id: crypto.randomUUID(),
          title,
          type: desiredType,
          content: '',
          isDirty: false,
          isExecuting: false,
          connectionId: initialConnectionId,
          selectedConnectionIds: initialConnectionId ? [initialConnectionId] : [],
          environmentSnapshot,
          aiSessionId: options?.aiSessionId,
        }

        set((state) => ({
          tabs: [...state.tabs, newTab],
          activeTabId: newTab.id,
        }))

        return newTab.id
      },

      closeTab: (id) => {
        set((state) => {
          const newTabs = state.tabs.filter((tab) => tab.id !== id)
          const wasActive = state.activeTabId === id

          return {
            tabs: newTabs,
            activeTabId: wasActive
              ? newTabs.length > 0
                ? newTabs[newTabs.length - 1].id
                : null
              : state.activeTabId,
            results: state.results.filter((result) => result.tabId !== id),
          }
        })
      },

      updateTab: (id, updates) => {
        set((state) => ({
          tabs: state.tabs.map((tab) => {
            if (tab.id !== id) {
              return tab
            }

            return {
              ...tab,
              ...updates,
              type: tab.type,
              aiSessionId: tab.aiSessionId,
            }
          }),
        }))
      },

      setActiveTab: (id) => {
        set({ activeTabId: id })
      },

      executeQuery: async (tabId, query, connectionId) => {
        const tab = get().tabs.find(t => t.id === tabId)
        if (!tab || tab.type !== 'sql') {
          return
        }

        get().updateTab(tabId, { isExecuting: true })

        // Use tab's connection if no connectionId provided
        const effectiveConnectionId = connectionId || tab.connectionId

        if (!effectiveConnectionId) {
          get().addResult({
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
          get().updateTab(tabId, { isExecuting: false })
          return
        }

        // Get the actual session ID from the connection store
        const { connections } = useConnectionStore.getState()
        const connection = connections.find(conn => conn.id === effectiveConnectionId)
        
        if (!connection?.sessionId) {
          get().addResult({
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
          get().updateTab(tabId, { isExecuting: false })
          return
        }

        try {
          const response = await wailsEndpoints.queries.execute(connection.sessionId, query)

          if (!response.success || !response.data) {
            const message = response.message || 'Query execution failed'
            get().addResult({
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
              connectionId: effectiveConnectionId, // Use the actual connection that executed the query
            })
            return
          }

          const {
            columns = [],
            rows = [],
            rowCount = 0,
            stats = {},
            editable: rawEditable = null,
          } = response.data

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
          const { rows: normalisedRows, originalRows } = normaliseRows(columns, rows, editableMetadata)

          const savedResult = get().addResult({
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
            connectionId: effectiveConnectionId, // Use the actual connection that executed the query
          })

          const jobId = editableMetadata?.jobId || editableMetadata?.job_id
          if (editableMetadata?.pending && jobId) {
            scheduleEditablePoll(jobId, savedResult.id)
          }

          get().updateTab(tabId, {
            lastExecuted: new Date(),
            isDirty: false,
          })
        } catch (error) {
          get().addResult({
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
            connectionId: effectiveConnectionId, // Use the actual connection that executed the query
          })
        } finally {
          get().updateTab(tabId, { isExecuting: false })
        }
      },

      addResult: (resultData) => {
        const newResult: QueryResult = {
          ...resultData,
          id: crypto.randomUUID(),
          timestamp: new Date(),
        }

        set((state) => ({
          results: [...state.results, newResult].slice(-20),
        }))

        return newResult
      },

      clearResults: (tabId) => {
        set((state) => ({
          results: state.results.filter((result) => result.tabId !== tabId),
        }))
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
    }
  },
  {
    name: 'query-store',
    partialize: (state) => ({
      tabs: state.tabs,
      activeTabId: state.activeTabId,
    }),
  }
),
{
  name: 'query-store',
})
)

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
        // Nothing to update; ensure we clear any timers
        cleanupEditableMetadataJob(jobId)
        return
      }

      const status = String(data.status ?? '').toLowerCase()
      const metadataPayload = data.metadata
      const errorMessage = (data.error as string) || ''

      const store = useQueryStore.getState()
      const resultExists = store.results.some(result => result.id === resultId)
      if (!resultExists) {
        cleanupEditableMetadataJob(jobId)
        return
      }

      const applyMetadata = (metadata: QueryEditableMetadata | null) => {
        cleanupEditableMetadataJob(jobId)
        store.updateResultEditable(resultId, metadata)
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
        // Update the UI to reflect pending status but keep polling as fallback
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

        store.updateResultEditable(resultId, metadata)
        editableMetadataTargets.set(jobId, resultId)
        return
      }

      // Unknown status, treat as failure but keep fallback polling just in case
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
