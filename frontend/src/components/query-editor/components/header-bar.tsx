import { Bug, ChevronDown, MessageCircle, Plug, Sparkles } from "lucide-react"

import { ModeSwitcher } from "@/components/mode-switcher"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { preloadComponent } from "@/lib/component-preload"

import type { AISidebarMode, QueryMode } from "../types"

// Preload function for GenericChatSidebar
const preloadGenericChatSidebar = () => import("@/components/generic-chat-sidebar").then(m => ({ default: m.GenericChatSidebar as React.ComponentType<unknown> }))

interface HeaderBarProps {
  mode: QueryMode
  canToggle: boolean
  connectionCount: number
  activeEnvironmentFilter: string | null
  connectedCount: number
  totalCount: number
  aiEnabled: boolean
  showAIDialog: boolean
  aiSidebarMode: AISidebarMode
  showDiagnostics: boolean
  onToggleMode: () => void
  onSetAISidebarMode: (mode: AISidebarMode) => void
  onSetShowAIDialog: (show: boolean) => void
  onToggleDiagnostics: () => void
  onSetIsFixMode: (mode: boolean) => void
  onSetAISheetTab: (tab: 'assistant' | 'memories') => void
}

export function HeaderBar({
  mode,
  canToggle,
  connectionCount,
  activeEnvironmentFilter,
  connectedCount,
  totalCount,
  aiEnabled,
  showAIDialog,
  aiSidebarMode,
  showDiagnostics,
  onToggleMode,
  onSetAISidebarMode,
  onSetShowAIDialog,
  onToggleDiagnostics,
  onSetIsFixMode,
  onSetAISheetTab,
}: HeaderBarProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/20">
      <div className="flex items-center gap-4">
        <ModeSwitcher
          mode={mode}
          canToggle={canToggle}
          toggleMode={onToggleMode}
          connectionCount={connectionCount}
        />

        {/* Environment and Connection Status */}
        <div className="flex items-center gap-2">
          {activeEnvironmentFilter && (
            <Badge variant="secondary" className="gap-1.5 font-medium">
              {activeEnvironmentFilter}
            </Badge>
          )}

          <Badge variant="secondary" className="gap-1.5 font-medium">
            <Plug className="h-3 w-3" />
            {connectedCount}/{totalCount} Connected
          </Badge>
        </div>
      </div>

      <div className="flex items-center gap-2">
        {/* AI Assistant Mode Selector */}
        {aiEnabled && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant={showAIDialog ? "default" : "ghost"}
                size="sm"
                className="gap-2"
              >
                {aiSidebarMode === 'generic' ? (
                  <MessageCircle className="h-4 w-4" />
                ) : (
                  <Sparkles className="h-4 w-4" />
                )}
                <span className="hidden text-xs sm:inline">
                  {aiSidebarMode === 'generic' ? 'Generic Chat' : 'AI Assistant'}
                </span>
                <ChevronDown className="h-3 w-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() => {
                  onSetAISidebarMode('sql')
                  onSetIsFixMode(false)
                  onSetAISheetTab('assistant')
                  onSetShowAIDialog(true)
                }}
              >
                <Sparkles className="mr-2 h-4 w-4" />
                SQL Assistant
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => {
                  onSetAISidebarMode('generic')
                  onSetIsFixMode(false)
                  onSetAISheetTab('assistant')
                  onSetShowAIDialog(true)
                }}
                onMouseEnter={() => void preloadComponent(preloadGenericChatSidebar)}
              >
                <MessageCircle className="mr-2 h-4 w-4" />
                Generic Chat
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}

        {/* Diagnostics Toggle Button */}
        <Button
          variant={showDiagnostics ? "default" : "ghost"}
          size="sm"
          onClick={onToggleDiagnostics}
          title="Toggle Diagnostics (Ctrl/Cmd+Shift+D)"
        >
          <Bug className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
