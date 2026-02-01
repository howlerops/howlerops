import { type RefObject,useEffect } from "react"

import type { CodeMirrorEditorRef } from "@/components/codemirror-editor"
import type { DatabaseConnection } from "@/store/connection-store"
import { buildExecutableSql } from "@/utils/sql"

import { isTypingTarget } from "../utils"

interface UseKeyboardShortcutsProps {
  editorRef: RefObject<CodeMirrorEditorRef | null>
  editorContentRef: RefObject<string>
  activeConnection: DatabaseConnection | null
  isExecuting: boolean
  user: { id: string } | null
  editorContent: string
  onExecuteQuery: () => void
  onToggleDiagnostics: () => void
  onToggleSavedQueries: () => void
  onOpenSaveQueryDialog: () => void
  onCreateSqlTab: () => void
  onCreateAiTab: () => void
}

/**
 * Hook that manages keyboard shortcuts for the query editor
 */
export function useKeyboardShortcuts({
  editorRef,
  editorContentRef,
  activeConnection,
  isExecuting,
  user,
  editorContent,
  onExecuteQuery,
  onToggleDiagnostics,
  onToggleSavedQueries,
  onOpenSaveQueryDialog,
  onCreateSqlTab,
  onCreateAiTab,
}: UseKeyboardShortcutsProps) {
  // Ctrl/Cmd+Shift+D: toggle diagnostics
  // Ctrl/Cmd+Shift+L: open Saved Queries library
  // Ctrl/Cmd+Shift+S: open Save Query dialog
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'D') {
        e.preventDefault()
        onToggleDiagnostics()
      }

      if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'L' || e.key === 'l')) {
        e.preventDefault()
        onToggleSavedQueries()
      }

      if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'S' || e.key === 's')) {
        if ((editorRef.current?.getValue() ?? editorContentRef.current).trim()) {
          e.preventDefault()
          onOpenSaveQueryDialog()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [editorRef, editorContentRef, onToggleDiagnostics, onToggleSavedQueries, onOpenSaveQueryDialog])

  // Ctrl/Cmd+Enter: execute query
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const view = editorRef.current?.getView()
        if (view && e.target instanceof Node && view.contentDOM.contains(e.target)) {
          // Let the editor's own keymap handle execution
          return
        }

        e.preventDefault()

        if (activeConnection?.isConnected && !isExecuting) {
          const currentEditorValue = editorRef.current?.getValue() ?? editorContent
          const selectedText = editorRef.current?.getSelectedText?.() ?? ''
          const cursorOffset = editorRef.current?.getCursorOffset?.() ?? currentEditorValue.length

          if (selectedText.trim().length > 0) {
            onExecuteQuery()
            return
          }

          const executableQuery = buildExecutableSql(currentEditorValue, {
            selectionText: selectedText,
            cursorOffset,
          })

          if (executableQuery) {
            onExecuteQuery()
          }
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [activeConnection, editorContent, isExecuting, onExecuteQuery, editorRef])

  // Ctrl/Cmd+Shift+S: save query (duplicate of above but separate for isolation)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'S') {
        e.preventDefault()
        if (user && editorContent.trim()) {
          onOpenSaveQueryDialog()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [user, editorContent, onOpenSaveQueryDialog])

  // Ctrl/Cmd+Shift+L: saved queries (duplicate of above)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toUpperCase() === 'L') {
        e.preventDefault()
        if (user) onToggleSavedQueries()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [user, onToggleSavedQueries])

  // Ctrl/Cmd+Shift+N: new SQL tab
  // Ctrl/Cmd+Shift+G: new AI tab
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (isTypingTarget(event.target)) return

      const modifierPressed = event.metaKey || event.ctrlKey
      if (!modifierPressed || !event.shiftKey) return

      const key = event.key.toLowerCase()
      if (key === 'n') {
        event.preventDefault()
        onCreateSqlTab()
      } else if (key === 'g') {
        event.preventDefault()
        onCreateAiTab()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onCreateSqlTab, onCreateAiTab])
}
