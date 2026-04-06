import { useCallback, useEffect, useRef, useState } from "react"

import type { CodeMirrorEditorRef } from "@/components/codemirror-editor"
import type { QueryIR } from "@/lib/query-ir"
import { useQueryEditorStore } from "@/store/query-editor-store"

import type { AISheetTab,AISidebarMode } from "../types"

/**
 * Hook that manages all the UI state for the query editor
 */
export function useEditorState() {
  const { tabs, activeTabId, updateTab } = useQueryEditorStore()
  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Editor content state
  const editorRef = useRef<CodeMirrorEditorRef>(null)
  const [editorContent, setEditorContent] = useState("")
  const editorContentRef = useRef(editorContent)
  const pendingTabUpdateRef = useRef<number | null>(null)

  // AI state
  const [naturalLanguagePrompt, setNaturalLanguagePrompt] = useState("")
  const [showAIDialog, setShowAIDialog] = useState(false)
  const [aiSidebarMode, setAISidebarMode] = useState<AISidebarMode>('sql')
  const [aiSheetTab, setAISheetTab] = useState<AISheetTab>('assistant')
  const [isFixMode, setIsFixMode] = useState(false)
  const [appliedSuggestionId, setAppliedSuggestionId] = useState<string | null>(null)

  // Visual mode state
  const [isVisualMode, setIsVisualMode] = useState(false)
  const [visualQueryIR, setVisualQueryIR] = useState<QueryIR | null>(null)

  // Dialog/panel state
  const [showSavedQueries, setShowSavedQueries] = useState(false)
  const [showDiagnostics, setShowDiagnostics] = useState(false)
  const [showConnectionSelector, setShowConnectionSelector] = useState(false)
  const [showDatabasePrompt, setShowDatabasePrompt] = useState(false)
  const [showSaveQueryDialog, setShowSaveQueryDialog] = useState(false)

  // Error state
  const [lastExecutionError, setLastExecutionError] = useState<string | null>(null)
  const [lastConnectionError, setLastConnectionError] = useState<string | null>(null)

  // Pending query for database selection prompt
  const [pendingQuery, setPendingQuery] = useState<string | null>(null)

  // Connection popover state
  const [openConnectionPopover, setOpenConnectionPopover] = useState<string | null>(null)

  // Rename dialog state
  const [renameSessionId, setRenameSessionId] = useState<string | null>(null)
  const [renameTitle, setRenameTitle] = useState('')

  // Keep editorContentRef in sync
  useEffect(() => {
    editorContentRef.current = editorContent
  }, [editorContent])

  // Flush pending tab updates
  const flushTabUpdate = useCallback((tabId: string, value: string) => {
    if (!tabId) return
    const snapshot = useQueryEditorStore.getState().tabs.find(tab => tab.id === tabId)
    const baselineContent = snapshot?.content ?? ''
    updateTab(tabId, {
      content: value,
      isDirty: value !== baselineContent,
    })
  }, [updateTab])

  // Schedule debounced tab updates
  const scheduleTabUpdate = useCallback((tabId: string, value: string) => {
    if (!tabId) return

    if (typeof window === 'undefined') {
      flushTabUpdate(tabId, value)
      return
    }

    if (pendingTabUpdateRef.current) {
      window.clearTimeout(pendingTabUpdateRef.current)
    }

    pendingTabUpdateRef.current = window.setTimeout(() => {
      flushTabUpdate(tabId, value)
      pendingTabUpdateRef.current = null
    }, 140)
  }, [flushTabUpdate])

  // Restore editor content when active tab changes
  useEffect(() => {
    if (activeTab?.content !== undefined) {
      setEditorContent(activeTab.content)
    }
  }, [activeTab?.id, activeTab?.content])

  // Cleanup pending updates on unmount or tab change
  useEffect(() => {
    const tabIdAtRegistration = activeTab?.id
    return () => {
      if (pendingTabUpdateRef.current && tabIdAtRegistration) {
        window.clearTimeout(pendingTabUpdateRef.current)
        flushTabUpdate(tabIdAtRegistration, editorContentRef.current)
        pendingTabUpdateRef.current = null
      }
    }
  }, [activeTab?.id, flushTabUpdate])

  // Rename dialog helpers
  const openRenameDialog = useCallback((sessionId: string, currentTitle: string) => {
    setRenameSessionId(sessionId)
    setRenameTitle(currentTitle)
  }, [])

  const closeRenameDialog = useCallback(() => {
    setRenameSessionId(null)
    setRenameTitle('')
  }, [])

  return {
    // Refs
    editorRef,
    editorContentRef,
    pendingTabUpdateRef,

    // Editor content
    editorContent,
    setEditorContent,

    // AI state
    naturalLanguagePrompt,
    setNaturalLanguagePrompt,
    showAIDialog,
    setShowAIDialog,
    aiSidebarMode,
    setAISidebarMode,
    aiSheetTab,
    setAISheetTab,
    isFixMode,
    setIsFixMode,
    appliedSuggestionId,
    setAppliedSuggestionId,

    // Visual mode
    isVisualMode,
    setIsVisualMode,
    visualQueryIR,
    setVisualQueryIR,

    // Dialogs/panels
    showSavedQueries,
    setShowSavedQueries,
    showDiagnostics,
    setShowDiagnostics,
    showConnectionSelector,
    setShowConnectionSelector,
    showDatabasePrompt,
    setShowDatabasePrompt,
    showSaveQueryDialog,
    setShowSaveQueryDialog,

    // Errors
    lastExecutionError,
    setLastExecutionError,
    lastConnectionError,
    setLastConnectionError,

    // Pending query
    pendingQuery,
    setPendingQuery,

    // Connection popover
    openConnectionPopover,
    setOpenConnectionPopover,

    // Rename dialog
    renameSessionId,
    setRenameSessionId,
    renameTitle,
    setRenameTitle,
    openRenameDialog,
    closeRenameDialog,

    // Tab update helpers
    flushTabUpdate,
    scheduleTabUpdate,
  }
}
