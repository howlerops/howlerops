import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { wailsEndpoints } from '@/lib/wails-api'
import { useConnectionStore } from './connection-store'

export interface QueryTab {
  id: string
  title: string
  content: string
  isDirty: boolean
  isExecuting: boolean
  lastExecuted?: Date
  connectionId?: string // Per-tab connection support
}

export interface QueryEditableColumn {
  name: string
  resultName: string
  dataType: string
  editable: boolean
  primaryKey: boolean
}

export interface QueryEditableMetadata {
  enabled: boolean
  reason?: string
  schema?: string
  table?: string
  primaryKeys: string[]
  columns: QueryEditableColumn[]
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
  createTab: (title?: string) => string
  closeTab: (id: string) => void
  updateTab: (id: string, updates: Partial<QueryTab>) => void
  setActiveTab: (id: string) => void
  executeQuery: (tabId: string, query: string, connectionId?: string | null) => Promise<void>
  addResult: (result: Omit<QueryResult, 'id' | 'timestamp'>) => void
  clearResults: (tabId: string) => void
  updateResultRows: (resultId: string, rows: QueryResultRow[], newOriginalRows?: Record<string, QueryResultRow>) => void
}

interface NormalisedRowsResult {
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
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
    (set, get) => ({
      tabs: [],
      activeTabId: null,
      results: [],

      createTab: (title = 'New Query', connectionId?: string) => {
        const newTab: QueryTab = {
          id: crypto.randomUUID(),
          title,
          content: '',
          isDirty: false,
          isExecuting: false,
          connectionId,
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
          tabs: state.tabs.map((tab) =>
            tab.id === id ? { ...tab, ...updates } : tab
          ),
        }))
      },

      setActiveTab: (id) => {
        set({ activeTabId: id })
      },

      executeQuery: async (tabId, query, connectionId) => {
        get().updateTab(tabId, { isExecuting: true })

        // Use tab's connection if no connectionId provided
        const tab = get().tabs.find(t => t.id === tabId)
        const effectiveConnectionId = connectionId || tab?.connectionId

        if (!effectiveConnectionId) {
          get().addResult({
            tabId,
            columns: [],
            rows: [],
            originalRows: {},
            rowCount: 0,
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
            executionTime: 0,
            error: message,
            editable: null,
            query,
            connectionId,
          })
            return
          }

          const {
            columns = [],
            rows = [],
            rowCount = 0,
            stats = {},
            editable = null,
          } = response.data
          const { rows: normalisedRows, originalRows } = normaliseRows(columns, rows, editable)

          get().addResult({
            tabId,
            columns,
            rows: normalisedRows,
            originalRows,
            rowCount: rowCount || normalisedRows.length,
            executionTime: parseDurationMs(stats.duration),
            error: undefined,
            editable,
            query,
            connectionId,
          })

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
            executionTime: 0,
            error: error instanceof Error ? error.message : 'Unknown error occurred',
            editable: null,
            query,
            connectionId,
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
    }),
    {
      name: 'query-store',
    }
  )
)
