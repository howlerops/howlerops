/**
 * Example Integration of Keyboard Shortcuts Provider
 *
 * This file demonstrates how to integrate the KeyboardShortcutsProvider
 * into the main app.tsx file. Follow these steps:
 *
 * 1. Import the provider in app.tsx:
 *    import { KeyboardShortcutsProvider } from '@/components/keyboard-shortcuts-provider'
 *
 * 2. Create refs to access child component methods (in the main component that renders Dashboard):
 *    const queryEditorRef = useRef<QueryEditorHandle>(null)
 *    const [aiChatOpen, setAIChatOpen] = useState(false)
 *
 * 3. Wrap the NavigationProvider with KeyboardShortcutsProvider:
 */

import { useRef, useState } from 'react'

import { KeyboardShortcutsProvider } from '@/components/keyboard-shortcuts-provider'
import type { QueryEditorHandle } from '@/components/query-editor'
import { useQueryStore } from '@/store/query-store'

/**
 * Example parent component that wraps the app with keyboard shortcuts
 */
export function AppWithKeyboardShortcuts({ children }: { children: React.ReactNode }) {
  const queryEditorRef = useRef<QueryEditorHandle>(null)
  const [aiChatOpen, setAIChatOpen] = useState(false)

  const {
    createTab,
    closeTab,
    setActiveTab,
    tabs,
    activeTabId,
  } = useQueryStore()

  return (
    <KeyboardShortcutsProvider
      // AI Features
      onOpenAIChat={() => setAIChatOpen(true)}
      onExplainQuery={() => {
        // Trigger explain on selected SQL
        console.log('Explain query shortcut triggered')
      }}
      onFixSQL={() => {
        // Trigger AI SQL fix
        console.log('Fix SQL shortcut triggered')
      }}

      // Navigation
      onNewTab={() => createTab()}
      onCloseTab={() => {
        if (activeTabId) {
          closeTab(activeTabId)
        }
      }}
      onSwitchTab={(index) => {
        const tabIds = Object.keys(tabs)
        if (tabIds[index]) {
          setActiveTab(tabIds[index])
        }
      }}

      // Query Execution
      onExecuteQuery={() => {
        // Execute current query
        console.log('Execute query shortcut triggered')
      }}
    >
      {children}
    </KeyboardShortcutsProvider>
  )
}

/**
 * Integration in app.tsx should look like this:
 *
 * ```tsx
 * function App() {
 *   return (
 *     <ErrorBoundary>
 *       <QueryClientProvider client={queryClient}>
 *         <ThemeProvider>
 *           <Router>
 *             <KeyboardShortcutsProvider
 *               onGenerateSQL={() => { ... }}
 *               onOpenAIChat={() => { ... }}
 *               // ... other handlers
 *             >
 *               <NavigationProvider>
 *                 <Routes>
 *                   ...
 *                 </Routes>
 *               </NavigationProvider>
 *             </KeyboardShortcutsProvider>
 *           </Router>
 *         </ThemeProvider>
 *       </QueryClientProvider>
 *     </ErrorBoundary>
 *   )
 * }
 * ```
 */
