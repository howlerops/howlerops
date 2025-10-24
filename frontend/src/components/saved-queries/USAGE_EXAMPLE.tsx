/**
 * SavedQueriesPanel Usage Example
 *
 * Complete example showing how to integrate the SavedQueriesPanel
 * into your application with proper state management.
 *
 * @module components/saved-queries/USAGE_EXAMPLE
 */

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { SavedQueriesPanel } from './SavedQueriesPanel'
import { SaveQueryDialog } from './SaveQueryDialog'
import type { SavedQueryRecord } from '@/types/storage'
import { BookMarked, Save } from 'lucide-react'

/**
 * Example: Basic Integration
 *
 * Shows the minimal setup needed to use SavedQueriesPanel
 */
export function BasicExample() {
  const [isPanelOpen, setIsPanelOpen] = useState(false)
  const userId = 'user-123' // Get from auth

  const handleLoadQuery = (query: SavedQueryRecord) => {
    console.log('Loading query:', query)
    // Load the query into your editor
    // editor.setValue(query.query_text)
  }

  return (
    <div>
      <Button onClick={() => setIsPanelOpen(true)}>
        <BookMarked className="mr-2 h-4 w-4" />
        Saved Queries
      </Button>

      <SavedQueriesPanel
        open={isPanelOpen}
        onClose={() => setIsPanelOpen(false)}
        userId={userId}
        onLoadQuery={handleLoadQuery}
      />
    </div>
  )
}

/**
 * Example: Full Query Editor Integration
 *
 * Shows complete integration with editor and save functionality
 */
export function FullQueryEditorExample() {
  const [isPanelOpen, setIsPanelOpen] = useState(false)
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false)
  const [currentQuery, setCurrentQuery] = useState('')
  const userId = 'user-123'

  const handleLoadQuery = (query: SavedQueryRecord) => {
    setCurrentQuery(query.query_text)
    setIsPanelOpen(false)
    // Optionally show a toast notification
    console.log(`Loaded: ${query.title}`)
  }

  const handleSaveQuery = () => {
    if (!currentQuery.trim()) {
      alert('No query to save')
      return
    }
    setIsSaveDialogOpen(true)
  }

  return (
    <div className="flex flex-col h-screen">
      {/* Toolbar */}
      <div className="flex items-center gap-2 p-4 border-b">
        <Button onClick={() => setIsPanelOpen(true)} variant="outline">
          <BookMarked className="mr-2 h-4 w-4" />
          Browse Saved Queries
        </Button>
        <Button onClick={handleSaveQuery} variant="default">
          <Save className="mr-2 h-4 w-4" />
          Save Query
        </Button>
      </div>

      {/* Editor */}
      <div className="flex-1 p-4">
        <textarea
          value={currentQuery}
          onChange={(e) => setCurrentQuery(e.target.value)}
          className="w-full h-full font-mono p-4 border rounded-md"
          placeholder="Write your SQL query here..."
        />
      </div>

      {/* Saved Queries Panel */}
      <SavedQueriesPanel
        open={isPanelOpen}
        onClose={() => setIsPanelOpen(false)}
        userId={userId}
        onLoadQuery={handleLoadQuery}
      />

      {/* Save Query Dialog */}
      <SaveQueryDialog
        open={isSaveDialogOpen}
        onClose={() => setIsSaveDialogOpen(false)}
        userId={userId}
        initialQuery={currentQuery}
        onSaved={(query) => {
          console.log('Query saved:', query)
          setIsSaveDialogOpen(false)
        }}
      />
    </div>
  )
}

/**
 * Example: With Keyboard Shortcuts
 *
 * Shows how to add keyboard shortcuts for common actions
 */
export function KeyboardShortcutExample() {
  const [isPanelOpen, setIsPanelOpen] = useState(false)
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false)
  const [currentQuery, setCurrentQuery] = useState('')
  const userId = 'user-123'

  // Set up keyboard shortcuts
  useState(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K: Open saved queries panel
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setIsPanelOpen(true)
      }

      // Cmd/Ctrl + S: Save query
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault()
        setIsSaveDialogOpen(true)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  })

  return (
    <div className="p-4">
      <div className="mb-4 text-sm text-muted-foreground">
        Keyboard shortcuts:
        <ul className="mt-2 space-y-1">
          <li><kbd className="px-2 py-1 bg-muted rounded">Cmd/Ctrl + K</kbd> - Open saved queries</li>
          <li><kbd className="px-2 py-1 bg-muted rounded">Cmd/Ctrl + S</kbd> - Save current query</li>
        </ul>
      </div>

      <textarea
        value={currentQuery}
        onChange={(e) => setCurrentQuery(e.target.value)}
        className="w-full h-64 font-mono p-4 border rounded-md"
        placeholder="Type your query... (Use Cmd/Ctrl+K to browse saved queries)"
      />

      <SavedQueriesPanel
        open={isPanelOpen}
        onClose={() => setIsPanelOpen(false)}
        userId={userId}
        onLoadQuery={(query) => {
          setCurrentQuery(query.query_text)
          setIsPanelOpen(false)
        }}
      />

      <SaveQueryDialog
        open={isSaveDialogOpen}
        onClose={() => setIsSaveDialogOpen(false)}
        userId={userId}
        initialQuery={currentQuery}
        onSaved={() => setIsSaveDialogOpen(false)}
      />
    </div>
  )
}

/**
 * Example: With Custom Actions
 *
 * Shows how to handle custom actions like copying to clipboard
 */
export function CustomActionsExample() {
  const [isPanelOpen, setIsPanelOpen] = useState(false)
  const userId = 'user-123'

  const handleLoadQuery = async (query: SavedQueryRecord) => {
    // Copy query to clipboard
    await navigator.clipboard.writeText(query.query_text)

    // Show toast notification
    console.log(`Copied "${query.title}" to clipboard`)

    // Close panel
    setIsPanelOpen(false)
  }

  return (
    <div>
      <Button onClick={() => setIsPanelOpen(true)}>
        Copy Query from Library
      </Button>

      <SavedQueriesPanel
        open={isPanelOpen}
        onClose={() => setIsPanelOpen(false)}
        userId={userId}
        onLoadQuery={handleLoadQuery}
      />
    </div>
  )
}

/**
 * Tips for Integration:
 *
 * 1. State Management:
 *    - The panel uses useSavedQueriesStore internally
 *    - No need to manage query state in parent component
 *    - Store is initialized on mount via useLoadSavedQueries hook
 *
 * 2. Filtering and Search:
 *    - All filter state is managed in the store
 *    - Search is debounced automatically (300ms)
 *    - Filters persist during the session
 *
 * 3. Performance:
 *    - Query list is virtualized via ScrollArea
 *    - Search/filter operations happen in-memory
 *    - Only renders visible query cards
 *
 * 4. Accessibility:
 *    - All interactive elements are keyboard accessible
 *    - Proper ARIA labels and roles
 *    - Screen reader friendly
 *
 * 5. Tier Limits:
 *    - Automatically checks tier limits via useTierStore
 *    - Shows progress bar for Local tier (max 20 queries)
 *    - Displays warnings when approaching limits
 *
 * 6. Empty States:
 *    - No queries saved
 *    - No search results
 *    - No favorites (when favorites filter is active)
 *    - Each state has helpful messaging and actions
 *
 * 7. Loading & Error States:
 *    - Loading spinner while fetching queries
 *    - Error message with retry button
 *    - Graceful handling of network issues
 */
