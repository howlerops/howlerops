import { HelpCircle, Keyboard, X } from 'lucide-react'
import { useEffect } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import type { ShortcutCategory } from '@/types/keyboard-shortcuts'
import { formatShortcut, type ShortcutConfig } from '@/types/keyboard-shortcuts'

interface KeyboardShortcutsHelperProps {
  open: boolean
  onClose: () => void
  categories: ShortcutCategory[]
}

export function KeyboardShortcutsHelper({ open, onClose, categories }: KeyboardShortcutsHelperProps) {
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose()
      }
    }

    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [open, onClose])

  if (categories.length === 0) {
    return null
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl max-h-[80vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Keyboard className="h-5 w-5" />
            Keyboard Shortcuts
          </DialogTitle>
          <DialogDescription>
            Press <Badge variant="outline" className="mx-1">?</Badge> anytime to show this help
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh] pr-4">
          <div className="space-y-6">
            {categories.map((category, index) => (
              <div key={category.id}>
                {index > 0 && <Separator className="my-4" />}
                <div className="space-y-3">
                  <div>
                    <h3 className="text-sm font-semibold text-foreground">{category.label}</h3>
                    {category.description && (
                      <p className="text-xs text-muted-foreground mt-1">{category.description}</p>
                    )}
                  </div>

                  <div className="space-y-2">
                    {category.shortcuts
                      .filter((shortcut) => shortcut.enabled !== false)
                      .map((shortcut) => {
                        const config: ShortcutConfig = {
                          key: shortcut.key,
                          ctrlKey: shortcut.modifier === 'ctrl' || shortcut.modifiers?.includes('ctrl'),
                          metaKey: shortcut.modifier === 'cmd' || shortcut.modifiers?.includes('cmd'),
                          altKey: shortcut.modifier === 'alt' || shortcut.modifiers?.includes('alt'),
                          shiftKey: shortcut.modifier === 'shift' || shortcut.modifiers?.includes('shift'),
                        }

                        return (
                          <div
                            key={shortcut.id}
                            className="flex items-center justify-between gap-4 rounded-md border bg-muted/30 px-3 py-2"
                          >
                            <div className="flex-1 min-w-0">
                              <div className="text-sm font-medium text-foreground">{shortcut.label}</div>
                              {shortcut.description && (
                                <div className="text-xs text-muted-foreground mt-0.5">{shortcut.description}</div>
                              )}
                            </div>
                            <Badge variant="secondary" className="font-mono text-xs whitespace-nowrap">
                              {formatShortcut(config)}
                            </Badge>
                          </div>
                        )
                      })}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>

        <div className="flex items-center justify-between pt-4 border-t">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <HelpCircle className="h-3 w-3" />
            <span>Shortcuts work from anywhere in the app</span>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="h-4 w-4 mr-1" />
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
