import { Command, Database, History, Search, Settings, Sparkles, Zap } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

export interface CommandAction {
  id: string
  label: string
  description?: string
  category: 'ai' | 'database' | 'settings' | 'navigation' | 'recent'
  icon?: React.ComponentType<{ className?: string }>
  keywords?: string[]
  handler: () => void | Promise<void>
  enabled?: boolean
  badge?: string
}

interface CommandPaletteProps {
  open: boolean
  onClose: () => void
  actions: CommandAction[]
  recentActions?: string[]
}

export function CommandPalette({ open, onClose, actions, recentActions = [] }: CommandPaletteProps) {
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)

  const filteredActions = useMemo(() => {
    if (!search.trim()) {
      return actions.filter((action) => action.enabled !== false)
    }

    const searchLower = search.toLowerCase().trim()
    const results = actions
      .filter((action) => {
        if (action.enabled === false) return false

        const labelMatch = action.label.toLowerCase().includes(searchLower)
        const descMatch = action.description?.toLowerCase().includes(searchLower)
        const keywordMatch = action.keywords?.some((kw) => kw.toLowerCase().includes(searchLower))
        const categoryMatch = action.category.toLowerCase().includes(searchLower)

        return labelMatch || descMatch || keywordMatch || categoryMatch
      })
      .map((action) => {
        // Calculate relevance score for better sorting
        let score = 0
        const searchLower = search.toLowerCase()

        if (action.label.toLowerCase().startsWith(searchLower)) score += 100
        if (action.label.toLowerCase().includes(searchLower)) score += 50
        if (action.description?.toLowerCase().includes(searchLower)) score += 20
        if (action.keywords?.some((kw) => kw.toLowerCase().includes(searchLower))) score += 30

        return { action, score }
      })
      .sort((a, b) => b.score - a.score)
      .map(({ action }) => action)

    return results
  }, [search, actions])

  const groupedActions = useMemo(() => {
    const groups = new Map<string, CommandAction[]>()

    // Add recent actions first if no search
    if (!search.trim() && recentActions.length > 0) {
      const recent = actions
        .filter((action) => recentActions.includes(action.id))
        .slice(0, 5)
      if (recent.length > 0) {
        groups.set('recent', recent)
      }
    }

    // Group filtered actions by category
    for (const action of filteredActions) {
      const category = action.category
      if (!groups.has(category)) {
        groups.set(category, [])
      }
      groups.get(category)!.push(action)
    }

    return Array.from(groups.entries())
  }, [filteredActions, search, recentActions, actions])

  const flatActions = useMemo(() => {
    return groupedActions.flatMap(([, actions]) => actions)
  }, [groupedActions])

  const handleExecute = useCallback(
    async (action: CommandAction) => {
      try {
        await action.handler()
        onClose()
        setSearch('')
        setSelectedIndex(0)
      } catch (error) {
        console.error('Command execution failed:', error)
      }
    },
    [onClose]
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex((prev) => (prev + 1) % flatActions.length)
          break
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex((prev) => (prev - 1 + flatActions.length) % flatActions.length)
          break
        case 'Enter':
          e.preventDefault()
          if (flatActions[selectedIndex]) {
            void handleExecute(flatActions[selectedIndex])
          }
          break
        case 'Escape':
          e.preventDefault()
          onClose()
          break
      }
    },
    [flatActions, selectedIndex, handleExecute, onClose]
  )

  useEffect(() => {
    if (open) {
      setSearch('')
      setSelectedIndex(0)
    }
  }, [open])

  useEffect(() => {
    setSelectedIndex(0)
  }, [search])

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl p-0 gap-0 overflow-hidden">
        <div className="flex items-center border-b px-4 py-3">
          <Search className="h-4 w-4 text-muted-foreground mr-2" />
          <Input
            placeholder="Search commands..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={handleKeyDown}
            className="border-0 focus-visible:ring-0 focus-visible:ring-offset-0 p-0 h-auto"
            autoFocus
          />
          <Badge variant="outline" className="ml-2">
            <Command className="h-3 w-3 mr-1" />K
          </Badge>
        </div>

        <ScrollArea className="max-h-[400px]">
          {groupedActions.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">
              <p className="text-sm">No commands found</p>
              <p className="text-xs mt-1">Try different keywords</p>
            </div>
          ) : (
            <div className="p-2">
              {groupedActions.map(([category, categoryActions], groupIndex) => (
                <div key={category}>
                  {groupIndex > 0 && <Separator className="my-2" />}
                  <div className="px-2 py-1.5">
                    <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      {getCategoryLabel(category)}
                    </p>
                  </div>
                  {categoryActions.map((action) => {
                    const globalIndex = flatActions.indexOf(action)
                    const isSelected = globalIndex === selectedIndex
                    const Icon = action.icon || getDefaultIcon(action.category)

                    return (
                      <button
                        key={action.id}
                        onClick={() => void handleExecute(action)}
                        onMouseEnter={() => setSelectedIndex(globalIndex)}
                        className={cn(
                          'w-full flex items-center gap-3 px-3 py-2 rounded-md text-left transition-colors',
                          isSelected ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50'
                        )}
                      >
                        <Icon className="h-4 w-4 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium truncate">{action.label}</div>
                          {action.description && (
                            <div className="text-xs text-muted-foreground truncate">{action.description}</div>
                          )}
                        </div>
                        {action.badge && (
                          <Badge variant="secondary" className="text-xs">
                            {action.badge}
                          </Badge>
                        )}
                      </button>
                    )
                  })}
                </div>
              ))}
            </div>
          )}
        </ScrollArea>

        <div className="border-t px-4 py-2 bg-muted/30">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1">
                <Badge variant="outline" className="text-xs">↑↓</Badge>
                Navigate
              </span>
              <span className="flex items-center gap-1">
                <Badge variant="outline" className="text-xs">Enter</Badge>
                Select
              </span>
              <span className="flex items-center gap-1">
                <Badge variant="outline" className="text-xs">Esc</Badge>
                Close
              </span>
            </div>
            <span>{flatActions.length} commands</span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function getCategoryLabel(category: string): string {
  const labels: Record<string, string> = {
    ai: 'AI Features',
    database: 'Database',
    settings: 'Settings',
    navigation: 'Navigation',
    recent: 'Recent Commands',
  }
  return labels[category] || category
}

function getDefaultIcon(category: string): React.ComponentType<{ className?: string }> {
  const icons: Record<string, React.ComponentType<{ className?: string }>> = {
    ai: Sparkles,
    database: Database,
    settings: Settings,
    navigation: Zap,
    recent: History,
  }
  return icons[category] || Command
}
