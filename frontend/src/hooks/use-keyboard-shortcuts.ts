import { useCallback, useEffect, useState } from 'react'

import type { KeyboardShortcut, ShortcutCategory, ShortcutConfig } from '@/types/keyboard-shortcuts'
import { matchesShortcut } from '@/types/keyboard-shortcuts'

interface UseKeyboardShortcutsOptions {
  enabled?: boolean
  ignoreInputs?: boolean
}

export function useKeyboardShortcuts(options: UseKeyboardShortcutsOptions = {}) {
  const { enabled = true, ignoreInputs = true } = options
  const [shortcuts, setShortcuts] = useState<Map<string, KeyboardShortcut>>(new Map())
  const [helpVisible, setHelpVisible] = useState(false)

  const registerShortcut = useCallback((shortcut: KeyboardShortcut) => {
    setShortcuts((prev) => {
      const next = new Map(prev)
      next.set(shortcut.id, shortcut)
      return next
    })
  }, [])

  const unregisterShortcut = useCallback((id: string) => {
    setShortcuts((prev) => {
      const next = new Map(prev)
      next.delete(id)
      return next
    })
  }, [])

  const toggleHelp = useCallback(() => {
    setHelpVisible((prev) => !prev)
  }, [])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return

      // Check if we should ignore this event
      if (ignoreInputs) {
        const target = event.target as HTMLElement
        const tagName = target.tagName.toLowerCase()
        const isInput =
          tagName === 'input' ||
          tagName === 'textarea' ||
          tagName === 'select' ||
          target.contentEditable === 'true'

        if (isInput) {
          // Allow Cmd/Ctrl+Enter in textareas (common pattern)
          if (tagName === 'textarea' && (event.metaKey || event.ctrlKey) && event.key === 'Enter') {
            // Let this through to be handled by shortcuts
          } else {
            return
          }
        }
      }

      // Check for help shortcut (? key)
      if (event.key === '?' && !event.ctrlKey && !event.metaKey && !event.altKey) {
        event.preventDefault()
        toggleHelp()
        return
      }

      // Try to match and execute shortcuts
      for (const shortcut of shortcuts.values()) {
        if (shortcut.enabled === false) continue

        const config: ShortcutConfig = {
          key: shortcut.key,
          ctrlKey: shortcut.modifier === 'ctrl' || shortcut.modifiers?.includes('ctrl'),
          metaKey: shortcut.modifier === 'cmd' || shortcut.modifiers?.includes('cmd'),
          altKey: shortcut.modifier === 'alt' || shortcut.modifiers?.includes('alt'),
          shiftKey: shortcut.modifier === 'shift' || shortcut.modifiers?.includes('shift'),
        }

        if (matchesShortcut(event, config)) {
          event.preventDefault()
          event.stopPropagation()
          shortcut.handler()
          return
        }
      }
    },
    [enabled, ignoreInputs, shortcuts, toggleHelp]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const getShortcutsByCategory = useCallback((): ShortcutCategory[] => {
    const categories = new Map<string, ShortcutCategory>()

    for (const shortcut of shortcuts.values()) {
      const categoryId = shortcut.category
      if (!categories.has(categoryId)) {
        categories.set(categoryId, {
          id: categoryId,
          label: getCategoryLabel(categoryId),
          description: getCategoryDescription(categoryId),
          shortcuts: [],
        })
      }
      categories.get(categoryId)!.shortcuts.push(shortcut)
    }

    return Array.from(categories.values()).sort((a, b) => {
      const order = ['ai', 'query', 'navigation', 'editor', 'general']
      return order.indexOf(a.id) - order.indexOf(b.id)
    })
  }, [shortcuts])

  return {
    shortcuts: Array.from(shortcuts.values()),
    registerShortcut,
    unregisterShortcut,
    helpVisible,
    setHelpVisible,
    toggleHelp,
    getShortcutsByCategory,
  }
}

function getCategoryLabel(id: string): string {
  const labels: Record<string, string> = {
    ai: 'AI Features',
    navigation: 'Navigation',
    query: 'Query Execution',
    editor: 'Editor',
    general: 'General',
  }
  return labels[id] || id
}

function getCategoryDescription(id: string): string {
  const descriptions: Record<string, string> = {
    ai: 'AI-powered query generation and analysis',
    navigation: 'Navigate between tabs and panels',
    query: 'Execute and manage queries',
    editor: 'Code editor shortcuts',
    general: 'General application shortcuts',
  }
  return descriptions[id] || ''
}
