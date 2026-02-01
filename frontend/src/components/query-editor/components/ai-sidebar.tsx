import {
  AlertCircle,
  Database,
  Loader2,
  Network,
  Pencil,
  Plus,
  Sparkles,
  Trash2,
  Wand2,
} from "lucide-react"
import { lazy, type ReactNode,Suspense } from "react"

import { AISchemaDisplay } from "@/components/ai-schema-display"
import { AISuggestionCard } from "@/components/ai-suggestion-card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { cn } from "@/lib/utils"
import type { SQLSuggestion } from "@/store/ai-store"
import type { DatabaseConnection } from "@/store/connection-store"
import type { QueryTab } from "@/store/query-store"

import type { AISheetTab, QueryMode } from "../types"

const GenericChatSidebar = lazy(() => import("@/components/generic-chat-sidebar").then(m => ({ default: m.GenericChatSidebar })))

interface MemorySession {
  id: string
  title: string
  createdAt: number
  updatedAt?: number
  messages: unknown[]
  summary?: string
}

interface AISidebarProps {
  mode: QueryMode
  open: boolean
  aiSidebarMode: 'sql' | 'generic'
  isFixMode: boolean
  aiSheetTab: AISheetTab
  naturalLanguagePrompt: string
  lastExecutionError: string | null
  lastError: string | null
  isGenerating: boolean
  suggestions: SQLSuggestion[]
  appliedSuggestionId: string | null
  memorySessions: MemorySession[]
  activeMemorySessionId: string | null
  activeTab: QueryTab | undefined
  connections: DatabaseConnection[]
  environmentFilteredConnections: DatabaseConnection[]
  editorConnections: DatabaseConnection[]
  editorSchemas: Map<string, SchemaNode[]>
  multiDBSchemas: Map<string, SchemaNode[]>
  schema: SchemaNode[]
  activeConnection: DatabaseConnection | null
  canToggle: boolean
  isConnecting: boolean
  activeDatabaseSelector: ReactNode
  renameSessionId: string | null
  renameTitle: string
  onClose: () => void
  onSetIsFixMode: (mode: boolean) => void
  onSetAISheetTab: (tab: AISheetTab) => void
  onSetNaturalLanguagePrompt: (prompt: string) => void
  onGenerateSQL: () => void
  onApplySuggestion: (query: string, id: string) => void
  onResetAISession: () => void
  onCreateMemorySession: () => void
  onDeleteMemorySession: (id: string) => void
  onClearAllMemories: () => void
  onResumeMemorySession: (id: string) => void
  onOpenRenameDialog: (id: string, title: string) => void
  onCloseRenameDialog: () => void
  onConfirmRename: () => void
  onSetRenameTitle: (title: string) => void
  onTabConnectionChange: (tabId: string, connectionId: string) => void
  onShowConnectionSelector: () => void
  onToggleMode: () => void
}

export function AISidebar({
  mode,
  open,
  aiSidebarMode,
  isFixMode,
  aiSheetTab,
  naturalLanguagePrompt,
  lastExecutionError,
  lastError,
  isGenerating,
  suggestions,
  appliedSuggestionId,
  memorySessions,
  activeMemorySessionId,
  activeTab,
  connections,
  environmentFilteredConnections,
  editorConnections,
  editorSchemas,
  multiDBSchemas,
  schema,
  activeConnection,
  canToggle,
  isConnecting,
  activeDatabaseSelector,
  renameSessionId,
  renameTitle,
  onClose,
  onSetIsFixMode,
  onSetAISheetTab,
  onSetNaturalLanguagePrompt,
  onGenerateSQL,
  onApplySuggestion,
  onResetAISession,
  onCreateMemorySession,
  onDeleteMemorySession,
  onClearAllMemories,
  onResumeMemorySession,
  onOpenRenameDialog,
  onCloseRenameDialog,
  onConfirmRename,
  onSetRenameTitle,
  onTabConnectionChange,
  onShowConnectionSelector,
  onToggleMode,
}: AISidebarProps) {
  if (aiSidebarMode === 'generic') {
    return (
      <Suspense fallback={null}>
        <GenericChatSidebar
          open={open}
          onClose={onClose}
          connections={editorConnections}
          schemasMap={editorSchemas}
        />
      </Suspense>
    )
  }

  return (
    <>
      <Sheet open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
        <SheetContent
          side="right"
          className="w-[600px] sm:max-w-[600px] m-4 h-[calc(100vh-2rem)] rounded-xl shadow-2xl border overflow-y-auto flex flex-col p-4"
        >
          <SheetHeader className="space-y-4 border-b pb-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3 pr-10">
                <div className="flex items-center gap-2 min-w-0">
                  <Sparkles className="h-5 w-5 text-primary flex-shrink-0" />
                  <SheetTitle className="truncate">
                    {isFixMode ? 'AI Query Fixer' : 'AI SQL Assistant'}
                  </SheetTitle>
                </div>
                {!isFixMode && (
                  <div className="flex items-center gap-2">
                    {/* Mode toggle */}
                    {canToggle && (
                      <div className="flex items-center rounded-md border bg-background overflow-hidden">
                        <Button
                          variant={mode === 'single' ? 'default' : 'ghost'}
                          size="sm"
                          className="h-8 px-2 text-xs"
                          onClick={() => mode === 'multi' && onToggleMode()}
                        >
                          Single
                        </Button>
                        <Button
                          variant={mode === 'multi' ? 'default' : 'ghost'}
                          size="sm"
                          className="h-8 px-2 text-xs"
                          onClick={() => mode === 'single' && onToggleMode()}
                        >
                          Multi
                        </Button>
                      </div>
                    )}

                    {/* Connection controls per mode */}
                    {mode === 'single' ? (
                      <div className="flex items-center gap-2">
                        <Select
                          value={activeTab?.connectionId || ''}
                          onValueChange={(value) => activeTab && onTabConnectionChange(activeTab.id, value)}
                          disabled={isConnecting}
                        >
                          <SelectTrigger className="h-8 w-44 text-xs" title={!activeTab?.connectionId ? 'Select a database' : undefined}>
                            <SelectValue placeholder={isConnecting ? 'Connecting...' : 'Select database'} />
                          </SelectTrigger>
                          <SelectContent>
                            {connections.map((conn) => (
                              <SelectItem key={conn.id} value={conn.id}>
                                <div className="flex items-center gap-2 text-xs">
                                  <Database className="h-3 w-3" />
                                  <span className="flex-1">{conn.name}</span>
                                </div>
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        {activeDatabaseSelector}
                      </div>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 px-2 text-xs"
                        onClick={onShowConnectionSelector}
                      >
                        <Network className="h-3 w-3 mr-1" />
                        {(() => {
                          const count = activeTab?.selectedConnectionIds?.length || 0
                          const total = environmentFilteredConnections.length
                          return `${count}/${total} DBs`
                        })()}
                      </Button>
                    )}

                    <Button
                      size="sm"
                      variant="outline"
                      onClick={onCreateMemorySession}
                      className="h-8 px-2 text-xs"
                    >
                      <Plus className="h-3 w-3 mr-1" />
                      New Session
                    </Button>
                  </div>
                )}
              </div>
              <SheetDescription className="mt-2 text-left">
                {isFixMode ? (
                  <>
                    The AI will analyze the error and suggest fixes for your query.
                    {lastExecutionError && (
                      <Alert variant="destructive" className="mt-2">
                        <AlertCircle className="h-4 w-4" />
                        <AlertTitle>Query Error</AlertTitle>
                        <AlertDescription className="text-xs whitespace-pre-wrap">
                          {lastExecutionError}
                        </AlertDescription>
                      </Alert>
                    )}
                  </>
                ) : (
                  <>
                    Describe what you want to query in natural language, and I'll generate the SQL for you.

                    {mode === 'multi' ? (
                      <Alert className="mt-2">
                        <Network className="h-4 w-4" />
                        <AlertTitle>Multi-Database Mode Active</AlertTitle>
                        <AlertDescription>
                          The AI can generate queries across multiple databases. Use @connectionName.table syntax in your descriptions.
                        </AlertDescription>
                      </Alert>
                    ) : (
                      <Alert className="mt-2">
                        <Database className="h-4 w-4" />
                        <AlertTitle>Single Database Mode</AlertTitle>
                        <AlertDescription>
                          Tip: Mention multiple databases or "compare" to trigger multi-database mode automatically.
                        </AlertDescription>
                      </Alert>
                    )}
                  </>
                )}
              </SheetDescription>
            </div>
            {!isFixMode && (
              <Tabs
                value={aiSheetTab}
                onValueChange={(value) => onSetAISheetTab(value as AISheetTab)}
                className="w-full"
              >
                <TabsList className="grid w-full grid-cols-2 bg-muted/40 p-1 rounded-lg">
                  <TabsTrigger value="assistant" className="h-8 text-sm">
                    Assistant
                  </TabsTrigger>
                  <TabsTrigger value="memories" className="h-8 text-sm">
                    Memories
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            )}
          </SheetHeader>

          <div className="flex-1 overflow-hidden">
            {isFixMode || aiSheetTab === 'assistant' ? (
              <div className="grid h-full gap-4 py-4 px-2 overflow-y-auto">
                {!isFixMode && (
                  <>
                    <div className="space-y-2">
                      <label htmlFor="ai-prompt" className="text-sm font-medium">
                        What would you like to query?
                      </label>
                      <textarea
                        id="ai-prompt"
                        placeholder="e.g., 'Show me all users who signed up last month with their total orders'"
                        value={naturalLanguagePrompt}
                        onChange={(e) => onSetNaturalLanguagePrompt(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                            e.preventDefault()
                            onGenerateSQL()
                          }
                        }}
                        className="w-full h-32 p-3 text-sm bg-background border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent"
                        disabled={isGenerating}
                      />
                      <p className="text-xs text-muted-foreground">
                        Press Ctrl+Enter (or Cmd+Enter on Mac) to generate SQL
                      </p>
                    </div>

                    <div className="space-y-2">
                      <label className="text-sm font-medium">Available Databases & Tables:</label>
                      <AISchemaDisplay
                        mode={mode}
                        connections={mode === 'multi' ? environmentFilteredConnections : (activeConnection ? [activeConnection] : [])}
                        schemasMap={mode === 'multi' ? multiDBSchemas : (activeConnection && schema ? new Map([[activeConnection.id, schema]]) : new Map())}
                        onTableClick={(connName, tableName, schemaName) => {
                          const tablePath = mode === 'multi'
                            ? (schemaName === 'public' ? `@${connName}.${tableName}` : `@${connName}.${schemaName}.${tableName}`)
                            : (schemaName === 'public' ? tableName : `${schemaName}.${tableName}`)

                          const currentPrompt = naturalLanguagePrompt
                          const newPrompt = currentPrompt
                            ? `${currentPrompt} ${tablePath}`
                            : `Query the ${tablePath} table`
                          onSetNaturalLanguagePrompt(newPrompt)
                        }}
                        className="border rounded-lg"
                      />
                    </div>
                  </>
                )}

                {lastError && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>Error</AlertTitle>
                    <AlertDescription>{lastError}</AlertDescription>
                  </Alert>
                )}

                {suggestions.length > 0 && (
                  <div className="border-t pt-4 space-y-3">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-semibold">
                        {isFixMode ? 'Suggested Fixes' : 'Generated Queries'}
                      </h3>
                      <span className="text-xs text-muted-foreground">
                        {suggestions.length} suggestion{suggestions.length !== 1 ? 's' : ''}
                      </span>
                    </div>
                    <div className="space-y-3 max-h-[400px] overflow-y-auto">
                      {suggestions.map((suggestion) => (
                        <AISuggestionCard
                          key={suggestion.id}
                          suggestion={suggestion}
                          onApply={(query) => onApplySuggestion(query, suggestion.id)}
                          isApplied={appliedSuggestionId === suggestion.id}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex h-full flex-col py-4">
                <div className="flex flex-col gap-2 border-b pb-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h3 className="text-sm font-semibold">Memory Sessions</h3>
                    <p className="text-xs text-muted-foreground">
                      Switch between saved assistant context or start fresh sessions.
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={onClearAllMemories}
                      disabled={memorySessions.length === 0}
                    >
                      Clear All
                    </Button>
                    <Button size="sm" onClick={onCreateMemorySession}>
                      <Plus className="h-4 w-4 mr-2" />
                      New Session
                    </Button>
                  </div>
                </div>

                <ScrollArea className="flex-1 pr-4">
                  <div className="space-y-2 py-4">
                    {memorySessions.length === 0 ? (
                      <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
                        No memory sessions yet. Create one to let the assistant remember context.
                      </div>
                    ) : (
                      memorySessions.map((session) => (
                        <div
                          key={session.id}
                          className={cn(
                            "rounded-lg border p-3 transition-colors",
                            session.id === activeMemorySessionId
                              ? "border-primary bg-primary/5"
                              : "hover:bg-muted/50"
                          )}
                        >
                          <div className="flex items-start justify-between gap-2">
                            <div className="min-w-0">
                              <p className="text-sm font-semibold flex items-center gap-2">
                                <span className="truncate">{session.title}</span>
                                {session.id === activeMemorySessionId && (
                                  <Badge variant="secondary" className="text-[10px] uppercase tracking-wide">
                                    Active
                                  </Badge>
                                )}
                              </p>
                              <p className="text-xs text-muted-foreground mt-1">
                                Updated {new Date((session.updatedAt || session.createdAt)).toLocaleString()}
                              </p>
                              <p className="text-xs text-muted-foreground mt-1">
                                {session.messages.length} message{session.messages.length === 1 ? '' : 's'}
                              </p>
                              {session.summary && (
                                <p className="text-xs text-muted-foreground mt-2 whitespace-pre-wrap">
                                  {session.summary}
                                </p>
                              )}
                            </div>
                            <div className="flex items-center gap-1">
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={() => onOpenRenameDialog(session.id, session.title)}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                                <span className="sr-only">Rename session</span>
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-destructive"
                                onClick={() => { void onDeleteMemorySession(session.id) }}
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                                <span className="sr-only">Delete session</span>
                              </Button>
                            </div>
                          </div>
                          <div className="mt-3 flex flex-wrap gap-2">
                            <Button
                              size="sm"
                              variant={session.id === activeMemorySessionId ? "default" : "outline"}
                              onClick={() => onResumeMemorySession(session.id)}
                            >
                              {session.id === activeMemorySessionId ? 'Continue in Assistant' : 'Resume in Assistant'}
                            </Button>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </ScrollArea>
              </div>
            )}
          </div>

          <div className="flex flex-wrap justify-end gap-2 border-t pt-4">
            {(isFixMode || aiSheetTab === 'assistant') && (
              <Button
                variant="outline"
                onClick={() => {
                  onResetAISession()
                  onSetNaturalLanguagePrompt('')
                }}
                disabled={isGenerating}
              >
                Reset Session
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => {
                onClose()
                onSetIsFixMode(false)
              }}
              disabled={isGenerating}
            >
              Close
            </Button>
            {!isFixMode && aiSheetTab === 'assistant' && (
              <Button
                onClick={onGenerateSQL}
                disabled={!naturalLanguagePrompt.trim() || isGenerating}
              >
                {isGenerating ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Wand2 className="h-4 w-4 mr-2" />
                    Generate SQL
                  </>
                )}
              </Button>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <Dialog
        open={!!renameSessionId}
        onOpenChange={(isOpen) => {
          if (!isOpen) {
            onCloseRenameDialog()
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename Memory Session</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <Input
              autoFocus
              value={renameTitle}
              onChange={(e) => onSetRenameTitle(e.target.value)}
              placeholder="Session title"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={onCloseRenameDialog}>
              Cancel
            </Button>
            <Button onClick={onConfirmRename} disabled={!renameTitle.trim()}>
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
