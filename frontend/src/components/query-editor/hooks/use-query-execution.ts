import type { RefObject } from "react"
import { useCallback, useState } from "react"

import type { CodeMirrorEditorRef } from "@/components/codemirror-editor"
import { useQueryStore } from "@/store/query-store"
import { buildExecutableSql } from "@/utils/sql"

import type { TabPaginationState } from "../types"

interface UseQueryExecutionProps {
  editorRef: RefObject<CodeMirrorEditorRef | null>
  editorContent: string
  setEditorContent: (content: string) => void
  setLastExecutionError: (error: string | null) => void
  setPendingQuery: (query: string | null) => void
  setShowDatabasePrompt: (show: boolean) => void
  flushTabUpdate: (tabId: string, value: string) => void
  pendingTabUpdateRef: RefObject<number | null>
}

/**
 * Hook that manages query execution logic
 */
export function useQueryExecution({
  editorRef,
  editorContent,
  setEditorContent,
  setLastExecutionError,
  setPendingQuery,
  setShowDatabasePrompt,
  flushTabUpdate,
  pendingTabUpdateRef,
}: UseQueryExecutionProps) {
  const { tabs, activeTabId, executeQuery, results } = useQueryStore()
  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Pagination state per tab
  const [tabPaginationState, setTabPaginationState] = useState<Record<string, TabPaginationState>>({})

  // Pagination handler - re-executes query with new limit/offset
  const handlePageChange = useCallback(async (tabId: string, limit: number, offset: number) => {
    const tab = tabs.find(t => t.id === tabId)
    if (!tab) return

    // Update pagination state
    setTabPaginationState(prev => ({
      ...prev,
      [tabId]: { limit, offset }
    }))

    // Find the last executed query for this tab
    const lastResult = results.find(r => r.tabId === tabId)
    if (!lastResult?.query) return

    // Re-execute query with new pagination
    await executeQuery(tabId, lastResult.query, tab.connectionId, limit, offset)
  }, [tabs, results, executeQuery])

  // Main execute query handler
  const handleExecuteQuery = useCallback(async () => {
    if (!activeTab) return

    const currentEditorValue = editorRef.current?.getValue() ?? editorContent
    const selectedText = editorRef.current?.getSelectedText?.() ?? ''
    const cursorOffset = editorRef.current?.getCursorOffset?.() ?? currentEditorValue.length

    // Determine the query to execute
    let queryToExecute: string
    if (selectedText.trim().length > 0) {
      queryToExecute = selectedText.trim()
    } else {
      const executableQuery = buildExecutableSql(currentEditorValue, {
        selectionText: selectedText,
        cursorOffset,
      })

      if (!executableQuery) {
        return
      }

      queryToExecute = executableQuery
    }

    // Check if we have a connection selected
    const hasConnection = activeTab.connectionId || (activeTab.selectedConnectionIds && activeTab.selectedConnectionIds.length > 0)

    if (!hasConnection) {
      // No connection selected - show the prompt
      setPendingQuery(queryToExecute)
      setShowDatabasePrompt(true)
      return
    }

    // We have a connection - proceed with execution
    setLastExecutionError(null)

    if (currentEditorValue !== editorContent) {
      setEditorContent(currentEditorValue)
    }

    if (pendingTabUpdateRef.current) {
      window.clearTimeout(pendingTabUpdateRef.current)
      // Note: We need to clear the ref value, but we can't mutate it directly
      // The parent component handles this through the pendingTabUpdateRef
    }

    flushTabUpdate(activeTab.id, currentEditorValue)

    // Reset pagination to first page when query changes
    const lastResult = results.find(r => r.tabId === activeTab.id)
    const queryChanged = !lastResult || lastResult.query !== queryToExecute

    if (queryChanged) {
      // Reset to first page
      setTabPaginationState(prev => ({
        ...prev,
        [activeTab.id]: { limit: 100, offset: 0 }
      }))
      await executeQuery(activeTab.id, queryToExecute, activeTab.connectionId, 100, 0)
    } else {
      // Use existing pagination state
      const paginationState = tabPaginationState[activeTab.id] || { limit: 100, offset: 0 }
      await executeQuery(activeTab.id, queryToExecute, activeTab.connectionId, paginationState.limit, paginationState.offset)
    }
  }, [
    activeTab,
    editorRef,
    editorContent,
    setEditorContent,
    setLastExecutionError,
    setPendingQuery,
    setShowDatabasePrompt,
    flushTabUpdate,
    pendingTabUpdateRef,
    results,
    executeQuery,
    tabPaginationState,
  ])

  // Handle database selected from prompt
  const handleDatabaseSelected = useCallback(async (
    connectionId: string,
    pendingQuery: string | null,
    updateTab: (id: string, updates: Record<string, unknown>) => void
  ) => {
    if (!activeTab || !pendingQuery) return

    // Update the tab with the selected connection
    updateTab(activeTab.id, {
      connectionId,
      selectedConnectionIds: [connectionId]
    })

    // Get pagination state for this tab (default to first page)
    const paginationState = tabPaginationState[activeTab.id] || { limit: 100, offset: 0 }

    // Execute the pending query with pagination
    await executeQuery(activeTab.id, pendingQuery, connectionId, paginationState.limit, paginationState.offset)
  }, [activeTab, executeQuery, tabPaginationState])

  return {
    tabPaginationState,
    setTabPaginationState,
    handlePageChange,
    handleExecuteQuery,
    handleDatabaseSelected,
  }
}
