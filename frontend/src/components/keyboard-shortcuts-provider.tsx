import { createContext, useCallback, useContext, useEffect, useMemo } from 'react'

import { CommandPalette } from '@/components/command-palette'
import { KeyboardShortcutsHelper } from '@/components/keyboard-shortcuts-helper'
import { useCommandPalette } from '@/hooks/use-command-palette'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'
import type { KeyboardShortcut } from '@/types/keyboard-shortcuts'
import { getModifierKey } from '@/types/keyboard-shortcuts'

interface KeyboardShortcutsContextValue {
  registerShortcut: (shortcut: KeyboardShortcut) => void
  unregisterShortcut: (id: string) => void
  showHelp: () => void
  showCommandPalette: () => void
}

const KeyboardShortcutsContext = createContext<KeyboardShortcutsContextValue | null>(null)

export function useKeyboardShortcutsContext() {
  const context = useContext(KeyboardShortcutsContext)
  if (!context) {
    throw new Error('useKeyboardShortcutsContext must be used within KeyboardShortcutsProvider')
  }
  return context
}

interface KeyboardShortcutsProviderProps {
  children: React.ReactNode
  onGenerateSQL?: () => void
  onExplainQuery?: () => void
  onFixSQL?: () => void
  onOptimizeQuery?: () => void
  onOpenAIChat?: () => void
  onAddConnection?: () => void
  onSwitchDatabase?: () => void
  onOpenSettings?: () => void
  onNewTab?: () => void
  onCloseTab?: () => void
  onSwitchTab?: (index: number) => void
  onExecuteQuery?: () => void
  onExecuteAndExplain?: () => void
}

export function KeyboardShortcutsProvider({
  children,
  onGenerateSQL,
  onExplainQuery,
  onFixSQL,
  onOptimizeQuery,
  onOpenAIChat,
  onAddConnection,
  onSwitchDatabase,
  onOpenSettings,
  onNewTab,
  onCloseTab,
  onSwitchTab,
  onExecuteQuery,
  onExecuteAndExplain,
}: KeyboardShortcutsProviderProps) {
  const {
    registerShortcut,
    unregisterShortcut,
    helpVisible,
    setHelpVisible,
    toggleHelp,
    getShortcutsByCategory,
  } = useKeyboardShortcuts({ enabled: true, ignoreInputs: true })

  const commandPalette = useCommandPalette({
    onGenerateSQL,
    onExplainQuery,
    onFixSQL,
    onOptimizeQuery,
    onOpenAIChat,
    onAddConnection,
    onSwitchDatabase,
    onOpenSettings,
    onNewTab,
    onCloseTab,
  })

  const modKey = getModifierKey()

  // Register all global shortcuts
  useEffect(() => {
    const shortcuts: KeyboardShortcut[] = [
      // Command Palette
      {
        id: 'command-palette',
        key: 'k',
        label: 'Command Palette',
        description: 'Open command palette for quick actions',
        category: 'general',
        modifiers: [modKey],
        handler: commandPalette.toggle,
      },

      // AI Features
      {
        id: 'ai-chat-toggle',
        key: '/',
        label: 'Toggle AI Chat',
        description: 'Open or close AI chat sidebar',
        category: 'ai',
        modifiers: [modKey],
        handler: () => onOpenAIChat?.(),
        enabled: !!onOpenAIChat,
      },
      {
        id: 'ai-explain-selected',
        key: 'e',
        label: 'Explain Selected SQL',
        description: 'Get AI explanation of highlighted SQL',
        category: 'ai',
        modifiers: [modKey, 'shift'],
        handler: () => onExplainQuery?.(),
        enabled: !!onExplainQuery,
      },
      {
        id: 'ai-fix-selected',
        key: 'f',
        label: 'Fix Selected SQL',
        description: 'Let AI fix errors in highlighted SQL',
        category: 'ai',
        modifiers: [modKey, 'shift'],
        handler: () => onFixSQL?.(),
        enabled: !!onFixSQL,
      },

      // Navigation - Tabs
      {
        id: 'new-tab',
        key: 'n',
        label: 'New Query Tab',
        description: 'Create a new query editor tab',
        category: 'navigation',
        modifiers: [modKey],
        handler: () => onNewTab?.(),
        enabled: !!onNewTab,
      },
      {
        id: 'close-tab',
        key: 'w',
        label: 'Close Tab',
        description: 'Close the current tab',
        category: 'navigation',
        modifiers: [modKey],
        handler: () => onCloseTab?.(),
        enabled: !!onCloseTab,
      },

      // Query Execution
      {
        id: 'execute-query',
        key: 'Enter',
        label: 'Execute Query',
        description: 'Run the current SQL query',
        category: 'query',
        modifiers: [modKey],
        handler: () => onExecuteQuery?.(),
        enabled: !!onExecuteQuery,
      },
      {
        id: 'execute-and-explain',
        key: 'Enter',
        label: 'Execute and Explain',
        description: 'Run query and get AI explanation',
        category: 'query',
        modifiers: [modKey, 'shift'],
        handler: () => onExecuteAndExplain?.(),
        enabled: !!onExecuteAndExplain,
      },
    ]

    // Register tab switching shortcuts (Cmd/Ctrl+1-9)
    for (let i = 1; i <= 9; i++) {
      shortcuts.push({
        id: `switch-tab-${i}`,
        key: String(i),
        label: `Switch to Tab ${i}`,
        description: `Activate query tab ${i}`,
        category: 'navigation',
        modifiers: [modKey],
        handler: () => onSwitchTab?.(i - 1),
        enabled: !!onSwitchTab,
      })
    }

    // Register all shortcuts
    shortcuts.forEach((shortcut) => {
      if (shortcut.enabled !== false) {
        registerShortcut(shortcut)
      }
    })

    // Cleanup
    return () => {
      shortcuts.forEach((shortcut) => {
        unregisterShortcut(shortcut.id)
      })
    }
  }, [
    registerShortcut,
    unregisterShortcut,
    modKey,
    commandPalette.toggle,
    onOpenAIChat,
    onExplainQuery,
    onFixSQL,
    onNewTab,
    onCloseTab,
    onSwitchTab,
    onExecuteQuery,
    onExecuteAndExplain,
  ])

  const showHelp = useCallback(() => {
    setHelpVisible(true)
  }, [setHelpVisible])

  const showCommandPalette = useCallback(() => {
    commandPalette.open()
  }, [commandPalette])

  const contextValue = useMemo(
    () => ({
      registerShortcut,
      unregisterShortcut,
      showHelp,
      showCommandPalette,
    }),
    [registerShortcut, unregisterShortcut, showHelp, showCommandPalette]
  )

  return (
    <KeyboardShortcutsContext.Provider value={contextValue}>
      {children}

      <KeyboardShortcutsHelper
        open={helpVisible}
        onClose={() => setHelpVisible(false)}
        categories={getShortcutsByCategory()}
      />

      <CommandPalette
        open={commandPalette.isOpen}
        onClose={commandPalette.close}
        actions={commandPalette.actions}
        recentActions={commandPalette.recentActions}
      />
    </KeyboardShortcutsContext.Provider>
  )
}
