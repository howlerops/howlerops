import { ChevronDown, Database, Network, Plus, Sparkles, X } from "lucide-react"
import type { SyntheticEvent } from "react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"
import type { DatabaseConnection } from "@/store/connection-store"
import type { QueryTab } from "@/store/query-store"

import type { QueryMode } from "../types"

interface QueryTabsProps {
  tabs: QueryTab[]
  activeTabId: string | null
  mode: QueryMode
  connections: DatabaseConnection[]
  environmentFilteredConnections: DatabaseConnection[]
  environmentOptions: string[]
  activeEnvironmentFilter: string | null
  isConnecting: boolean
  openConnectionPopover: string | null
  lastConnectionError: string | null
  onTabClick: (tabId: string) => void
  onCloseTab: (tabId: string, e: SyntheticEvent) => void
  onConnectionChange: (tabId: string, connectionId: string) => void
  onConnectionPopoverToggle: (tabId: string, open: boolean) => void
  onSetEnvironmentFilter: (env: string | null) => void
  onOpenConnectionSelector: (tabId: string) => void
  onCreateSqlTab: () => void
  onCreateAiTab: () => void
  getConnectionLabelForTab: (tab: QueryTab) => string
  getActiveConnectionsForTab: (tab: QueryTab) => string[]
}

export function QueryTabs({
  tabs,
  activeTabId,
  mode,
  connections: _connections,
  environmentFilteredConnections,
  environmentOptions,
  activeEnvironmentFilter,
  isConnecting,
  openConnectionPopover,
  lastConnectionError,
  onTabClick,
  onCloseTab,
  onConnectionChange,
  onConnectionPopoverToggle,
  onSetEnvironmentFilter,
  onOpenConnectionSelector,
  onCreateSqlTab,
  onCreateAiTab,
  getConnectionLabelForTab,
  getActiveConnectionsForTab,
}: QueryTabsProps) {
  const hasEnvironmentFilters = environmentOptions.length > 0

  return (
    <div className="flex items-center">
      <div className="flex-1 flex items-center overflow-x-auto">
        <Tabs value={activeTabId || ''} className="w-full">
          <TabsList className="h-10 bg-transparent border-0 rounded-none p-0">
            {tabs.map((tab) => {
              const hasNoConnection = !tab.connectionId

              return (
                <div key={tab.id} className="flex items-center border-r">
                  <TabsTrigger
                    value={tab.id}
                    onClick={() => onTabClick(tab.id)}
                    className={cn(
                      "h-10 px-3 rounded-none border-0 data-[state=active]:bg-background",
                      "data-[state=active]:border-b-2 data-[state=active]:border-primary",
                      "flex items-center space-x-2 min-w-[120px] max-w-[200px]",
                      mode === 'single' && hasNoConnection && "border-b-2 border-accent/50"
                    )}
                  >
                    <span className={cn(
                      "truncate",
                      mode === 'single' && hasNoConnection && "text-accent-foreground"
                    )}>
                      {tab.title}
                      {tab.isDirty && <span className="ml-1">*</span>}
                      {mode === 'single' && hasNoConnection && <span className="ml-1" title="No connection selected">!</span>}
                    </span>
                  </TabsTrigger>

                  {/* Connection Selector & Environment Filter */}
                  <div className="px-2">
                    {mode === 'single' ? (
                      <>
                        <Popover
                          open={openConnectionPopover === tab.id}
                          onOpenChange={(open) => onConnectionPopoverToggle(tab.id, open)}
                        >
                          <PopoverTrigger asChild>
                            <Button
                              variant="outline"
                              size="sm"
                              className={cn(
                                "h-7 w-36 justify-between",
                                !tab.connectionId && "border-accent text-accent-foreground bg-accent/10"
                              )}
                              onClick={(e) => e.stopPropagation()}
                              disabled={isConnecting}
                            >
                              <span className="truncate text-xs">{getConnectionLabelForTab(tab)}</span>
                              <ChevronDown className="h-3 w-3 opacity-70" />
                            </Button>
                          </PopoverTrigger>
                          <PopoverContent className="w-64 space-y-3" align="start" onOpenAutoFocus={(e) => e.preventDefault()}>
                            {hasEnvironmentFilters && (
                              <div className="space-y-2">
                                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Environment</p>
                                <div className="flex flex-wrap gap-2">
                                  <Button
                                    type="button"
                                    variant={!activeEnvironmentFilter ? "secondary" : "ghost"}
                                    size="sm"
                                    className="h-6 px-2 text-xs"
                                    onClick={() => void onSetEnvironmentFilter(null)}
                                  >
                                    All
                                  </Button>
                                  {environmentOptions.map((env) => (
                                    <Button
                                      key={env}
                                      type="button"
                                      variant={activeEnvironmentFilter === env ? "secondary" : "ghost"}
                                      size="sm"
                                      className="h-6 px-2 text-xs"
                                      onClick={() => void onSetEnvironmentFilter(env)}
                                    >
                                      {env}
                                    </Button>
                                  ))}
                                </div>
                              </div>
                            )}

                            <div className="space-y-2">
                              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Connections</p>
                              <div className="max-h-60 space-y-1 overflow-y-auto pr-1">
                                {environmentFilteredConnections.length === 0 ? (
                                  <div className="text-xs text-muted-foreground py-6 text-center border rounded-md">
                                    No connections in this environment
                                  </div>
                                ) : (
                                  environmentFilteredConnections.map((conn) => (
                                    <button
                                      key={conn.id}
                                      type="button"
                                      className={cn(
                                        "w-full rounded-md border px-2 py-2 text-left text-sm hover:bg-accent",
                                        tab.connectionId === conn.id && "border-primary bg-accent/40"
                                      )}
                                      onClick={() => onConnectionChange(tab.id, conn.id)}
                                    >
                                      <div className="flex items-center justify-between">
                                        <div className="flex items-center gap-2">
                                          <Database className="h-3.5 w-3.5" />
                                          <span className="truncate">{conn.name}</span>
                                        </div>
                                        {conn.isConnected ? (
                                          <span className="text-xs text-green-500 font-semibold">*</span>
                                        ) : (
                                          <span className="text-xs text-muted-foreground">o</span>
                                        )}
                                      </div>
                                    </button>
                                  ))
                                )}
                              </div>
                            </div>
                          </PopoverContent>
                        </Popover>
                        {lastConnectionError && (
                          <div className="text-xs text-destructive mt-1 max-w-32 truncate" title={lastConnectionError}>
                            {lastConnectionError}
                          </div>
                        )}
                      </>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          onOpenConnectionSelector(tab.id)
                        }}
                        className="h-6 px-2 text-xs bg-accent/10 border-accent hover:bg-accent/20"
                        disabled={environmentFilteredConnections.length === 0}
                      >
                        <Network className="h-3 w-3 mr-1 text-accent-foreground" />
                        {(() => {
                          const activeConnections = getActiveConnectionsForTab(tab)
                          return `${activeConnections.length}/${environmentFilteredConnections.length} DBs`
                        })()}
                      </Button>
                    )}
                  </div>

                  {/* Close Button */}
                  {tabs.length > 1 && (
                    <span
                      role="button"
                      tabIndex={0}
                      className="px-1 inline-flex h-4 w-4 cursor-pointer items-center justify-center rounded hover:bg-destructive/10 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-destructive"
                      onClick={(e) => onCloseTab(tab.id, e)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          onCloseTab(tab.id, e)
                        }
                      }}
                    >
                      <X className="h-3 w-3" />
                    </span>
                  )}
                </div>
              )
            })}
          </TabsList>
        </Tabs>
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="ml-2"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={onCreateSqlTab}>
            <Database className="h-4 w-4 mr-2" />
            SQL Editor Tab
          </DropdownMenuItem>
          <DropdownMenuItem onClick={onCreateAiTab}>
            <Sparkles className="h-4 w-4 mr-2" />
            AI Query Agent
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
