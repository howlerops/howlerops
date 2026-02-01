/**
 * Query Store - Backwards Compatibility Layer
 *
 * This file re-exports from the split stores for backwards compatibility.
 * The query store has been split into three separate stores:
 * - query-editor-store.ts: Tabs, active query, editor state
 * - query-execution-store.ts: Query execution, running queries, errors
 * - query-history-store.ts: Query results and history
 *
 * New code should import directly from the specific stores.
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'

import { deleteTabResults } from '@/lib/query-result-storage'

import { useQueryEditorStore } from './query-editor-store'
import { useQueryExecutionStore } from './query-execution-store'
import { useQueryHistoryStore } from './query-history-store'
import type {
  QueryEditableColumn,
  QueryEditableMetadata,
  QueryResult,
  QueryResultRow,
  QueryTab,
  QueryTabType,
} from './query-types'

// Re-export types
export type {
  QueryEditableColumn,
  QueryEditableMetadata,
  QueryResult,
  QueryResultRow,
  QueryTab,
  QueryTabType,
}

// Re-export stores for direct use
export { useQueryEditorStore } from './query-editor-store'
export { useQueryExecutionStore } from './query-execution-store'
export { useQueryHistoryStore } from './query-history-store'

// Re-export selectors
export {
  useActiveTab,
  useActiveTabId,
  useQueryEditorActions,
  useQueryEditorTabs,
} from './query-editor-store'
export {
  useExecutingQueries,
  useIsExecuting,
  useQueryExecutionActions,
} from './query-execution-store'
export {
  useLatestTabResult,
  useQueryHistoryActions,
  useQueryResults,
  useTabResults,
} from './query-history-store'

// Re-export utilities for external consumers
export {
  generateRowId,
  normaliseRows,
  parseDurationMs,
  transformEditableColumn,
  transformEditableMetadata,
} from './query-utils'

/**
 * Combined query state interface for backwards compatibility
 */
interface QueryState {
  tabs: QueryTab[]
  activeTabId: string | null
  results: QueryResult[]

  // Actions
  createTab: (title?: string, options?: { connectionId?: string; type?: QueryTabType; aiSessionId?: string }) => string
  closeTab: (id: string) => void
  updateTab: (id: string, updates: Partial<QueryTab>) => void
  setActiveTab: (id: string) => void
  executeQuery: (tabId: string, query: string, connectionId?: string | null, limit?: number, offset?: number) => Promise<void>
  addResult: (result: Omit<QueryResult, 'id' | 'timestamp'>) => QueryResult
  clearResults: (tabId: string) => void
  updateResultRows: (resultId: string, rows: QueryResultRow[], newOriginalRows?: Record<string, QueryResultRow>) => void
  updateResultEditable: (resultId: string, metadata: QueryEditableMetadata | null) => void
  updateResultProcessing: (resultId: string, isProcessing: boolean, progress?: number) => void
  loadMoreRows: (resultId: string) => Promise<void>
}

/**
 * Unified query store for backwards compatibility
 *
 * This store combines state from the three split stores and delegates
 * actions to the appropriate store. New code should use the individual
 * stores directly (useQueryEditorStore, useQueryExecutionStore, useQueryHistoryStore).
 */
export const useQueryStore = create<QueryState>()(
  devtools(
    persist(
      (set, get) => ({
        // State is derived from individual stores via subscriptions
        tabs: [],
        activeTabId: null,
        results: [],

        createTab: (title, options) => {
          return useQueryEditorStore.getState().createTab(title, options)
        },

        closeTab: (id) => {
          // Clean up IndexedDB results for this tab
          deleteTabResults(id).catch((error) => {
            console.error('Failed to delete tab results from IndexedDB:', error)
          })

          // Clear results for this tab
          useQueryHistoryStore.getState().clearResults(id)

          // Close the tab
          useQueryEditorStore.getState().closeTab(id)
        },

        updateTab: (id, updates) => {
          useQueryEditorStore.getState().updateTab(id, updates)
        },

        setActiveTab: (id) => {
          useQueryEditorStore.getState().setActiveTab(id)
        },

        executeQuery: async (tabId, query, connectionId, limit, offset) => {
          await useQueryExecutionStore.getState().executeQuery(tabId, query, connectionId, limit, offset)
        },

        addResult: (result) => {
          return useQueryHistoryStore.getState().addResult(result)
        },

        clearResults: (tabId) => {
          useQueryHistoryStore.getState().clearResults(tabId)
        },

        updateResultRows: (resultId, rows, newOriginalRows) => {
          useQueryHistoryStore.getState().updateResultRows(resultId, rows, newOriginalRows)
        },

        updateResultEditable: (resultId, metadata) => {
          useQueryHistoryStore.getState().updateResultEditable(resultId, metadata)
        },

        updateResultProcessing: (resultId, isProcessing, progress) => {
          useQueryHistoryStore.getState().updateResultProcessing(resultId, isProcessing, progress)
        },

        loadMoreRows: async (resultId) => {
          await useQueryExecutionStore.getState().loadMoreRows(resultId)
        },
      }),
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
    }
  )
)

// Synchronize state from individual stores to the unified store
// This ensures backwards compatibility for consumers that read from useQueryStore
const syncFromEditorStore = () => {
  const editorState = useQueryEditorStore.getState()
  useQueryStore.setState({
    tabs: editorState.tabs,
    activeTabId: editorState.activeTabId,
  })
}

const syncFromHistoryStore = () => {
  const historyState = useQueryHistoryStore.getState()
  useQueryStore.setState({
    results: historyState.results,
  })
}

// Subscribe to changes in individual stores
useQueryEditorStore.subscribe(syncFromEditorStore)
useQueryHistoryStore.subscribe(syncFromHistoryStore)

// Initial sync
syncFromEditorStore()
syncFromHistoryStore()
