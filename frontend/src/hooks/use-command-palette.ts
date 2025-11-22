import { useCallback, useMemo, useState } from 'react'

import type { CommandAction } from '@/components/command-palette'

interface UseCommandPaletteOptions {
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
}

export function useCommandPalette(options: UseCommandPaletteOptions = {}) {
  const [open, setOpen] = useState(false)
  const [recentActions, setRecentActions] = useState<string[]>([])

  const actions = useMemo((): CommandAction[] => {
    return [
      // AI Features
      {
        id: 'ai-generate-sql',
        label: 'Generate SQL from natural language',
        description: 'Use AI to convert your question into SQL',
        category: 'ai',
        keywords: ['generate', 'create', 'natural', 'language', 'nl', 'ai'],
        handler: () => options.onGenerateSQL?.(),
        enabled: !!options.onGenerateSQL,
      },
      {
        id: 'ai-explain-query',
        label: 'Explain this query',
        description: 'Get AI explanation of selected SQL',
        category: 'ai',
        keywords: ['explain', 'describe', 'what', 'does'],
        handler: () => options.onExplainQuery?.(),
        enabled: !!options.onExplainQuery,
      },
      {
        id: 'ai-fix-sql',
        label: 'Fix SQL errors',
        description: 'Let AI fix syntax or logic errors',
        category: 'ai',
        keywords: ['fix', 'repair', 'correct', 'error', 'debug'],
        handler: () => options.onFixSQL?.(),
        enabled: !!options.onFixSQL,
      },
      {
        id: 'ai-optimize-query',
        label: 'Optimize query performance',
        description: 'Get AI suggestions for query optimization',
        category: 'ai',
        keywords: ['optimize', 'performance', 'speed', 'fast', 'improve'],
        handler: () => options.onOptimizeQuery?.(),
        enabled: !!options.onOptimizeQuery,
      },
      {
        id: 'ai-chat',
        label: 'Open AI Chat',
        description: 'Chat with AI about your database',
        category: 'ai',
        keywords: ['chat', 'talk', 'ask', 'question'],
        handler: () => options.onOpenAIChat?.(),
        enabled: !!options.onOpenAIChat,
      },

      // Database
      {
        id: 'db-add-connection',
        label: 'Add connection',
        description: 'Connect to a new database',
        category: 'database',
        keywords: ['add', 'new', 'connect', 'connection', 'database'],
        handler: () => options.onAddConnection?.(),
        enabled: !!options.onAddConnection,
      },
      {
        id: 'db-switch',
        label: 'Switch database',
        description: 'Change active database connection',
        category: 'database',
        keywords: ['switch', 'change', 'database', 'connection'],
        handler: () => options.onSwitchDatabase?.(),
        enabled: !!options.onSwitchDatabase,
      },

      // Navigation
      {
        id: 'nav-new-tab',
        label: 'New query tab',
        description: 'Open a new query editor tab',
        category: 'navigation',
        keywords: ['new', 'tab', 'query', 'editor'],
        handler: () => options.onNewTab?.(),
        enabled: !!options.onNewTab,
      },
      {
        id: 'nav-close-tab',
        label: 'Close current tab',
        description: 'Close the active query tab',
        category: 'navigation',
        keywords: ['close', 'tab', 'exit'],
        handler: () => options.onCloseTab?.(),
        enabled: !!options.onCloseTab,
      },

      // Settings
      {
        id: 'settings-open',
        label: 'Open Settings',
        description: 'Configure application settings',
        category: 'settings',
        keywords: ['settings', 'preferences', 'config', 'configure'],
        handler: () => options.onOpenSettings?.(),
        enabled: !!options.onOpenSettings,
      },
    ].filter((action) => action.enabled !== false)
  }, [options])

  const open = useCallback(() => {
    setOpen(true)
  }, [])

  const close = useCallback(() => {
    setOpen(false)
  }, [])

  const toggle = useCallback(() => {
    setOpen((prev) => !prev)
  }, [])

  const recordAction = useCallback((actionId: string) => {
    setRecentActions((prev) => {
      const filtered = prev.filter((id) => id !== actionId)
      return [actionId, ...filtered].slice(0, 10)
    })
  }, [])

  return {
    isOpen: open,
    open: open,
    close,
    toggle,
    actions,
    recentActions,
    recordAction,
  }
}
