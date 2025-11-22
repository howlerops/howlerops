export interface KeyboardShortcut {
  id: string
  key: string
  label: string
  description: string
  category: 'ai' | 'navigation' | 'query' | 'editor' | 'general'
  handler: () => void
  enabled?: boolean
  modifier?: 'ctrl' | 'cmd' | 'alt' | 'shift'
  modifiers?: Array<'ctrl' | 'cmd' | 'alt' | 'shift'>
}

export interface ShortcutCategory {
  id: string
  label: string
  description: string
  shortcuts: KeyboardShortcut[]
}

export type ShortcutMap = Record<string, KeyboardShortcut>

export interface ShortcutConfig {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  altKey?: boolean
  shiftKey?: boolean
}

export function getModifierKey(): 'cmd' | 'ctrl' {
  return navigator.platform.toLowerCase().includes('mac') ? 'cmd' : 'ctrl'
}

export function formatShortcut(config: ShortcutConfig): string {
  const parts: string[] = []
  const isMac = navigator.platform.toLowerCase().includes('mac')

  if (config.ctrlKey || config.metaKey) {
    parts.push(isMac ? '⌘' : 'Ctrl')
  }
  if (config.altKey) {
    parts.push(isMac ? '⌥' : 'Alt')
  }
  if (config.shiftKey) {
    parts.push(isMac ? '⇧' : 'Shift')
  }

  parts.push(config.key.toUpperCase())

  return parts.join(isMac ? '' : '+')
}

export function matchesShortcut(event: KeyboardEvent, config: ShortcutConfig): boolean {
  const modifierPressed = event.ctrlKey || event.metaKey
  const modifierRequired = config.ctrlKey || config.metaKey

  return (
    event.key.toLowerCase() === config.key.toLowerCase() &&
    modifierPressed === modifierRequired &&
    event.altKey === (config.altKey ?? false) &&
    event.shiftKey === (config.shiftKey ?? false)
  )
}
