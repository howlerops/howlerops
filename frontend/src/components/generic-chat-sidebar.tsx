import { AlertCircle, BarChart3, Database, FileText, Lightbulb, Loader2, MessageCircle, Network, Pencil, Plus, Search, SendHorizontal, Sparkles, TrendingUp } from "lucide-react"
import { useEffect, useMemo, useState } from "react"

import { EmptyState, type ExampleQuery } from "@/components/empty-states/EmptyState"
import { MultiDBConnectionSelector } from "@/components/multi-db-connection-selector"
import { SchemaContextSelector } from "@/components/schema-context-selector"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import { VirtualMessageList } from "@/components/virtual-message-list"
import { useQueryMode } from "@/hooks/use-query-mode"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { AISchemaContextBuilder } from "@/lib/ai-schema-context"
import { cn } from "@/lib/utils"
import { useAIMemoryStore } from "@/store/ai-memory-store"
import { useAIConfig,useAIGeneration } from "@/store/ai-store"
import type { DatabaseConnection } from "@/store/connection-store"

const CHAT_EXAMPLES: ExampleQuery[] = [
  {
    label: "Explain query results",
    description: "Get a plain English explanation of your SQL results",
    query: "Explain the results of my last query in simple terms",
    icon: FileText,
  },
  {
    label: "Identify trends",
    description: "Find patterns and trends in your data",
    query: "What trends do you see in this data?",
    icon: TrendingUp,
  },
  {
    label: "Suggest analysis",
    description: "Get recommendations for further analysis",
    query: "What other questions should I ask about this dataset?",
    icon: Lightbulb,
  },
  {
    label: "Data insights",
    description: "Discover interesting insights from your results",
    query: "What are the most interesting insights from these results?",
    icon: Search,
  },
  {
    label: "Visualize data",
    description: "Get suggestions for how to visualize your data",
    query: "How should I visualize this data?",
    icon: BarChart3,
  },
]

interface GenericChatSidebarProps {
  open: boolean
  onClose: () => void
  connections: DatabaseConnection[]
  schemasMap: Map<string, SchemaNode[]>
}

export function GenericChatSidebar({ open, onClose, connections, schemasMap }: GenericChatSidebarProps) {
  const { sendGenericMessage, isGenerating, lastError, hydrateMemoriesFromBackend } = useAIGeneration()
  const { config } = useAIConfig()
  const memorySessions = useAIMemoryStore(state => state.sessions)
  const activeSessionId = useAIMemoryStore(state => state.activeSessionId)
  const setActiveSession = useAIMemoryStore(state => state.setActiveSession)
  const startNewSession = useAIMemoryStore(state => state.startNewSession)
  const renameSession = useAIMemoryStore(state => state.renameSession)

  const [message, setMessage] = useState("")
  const [schemaContext, setSchemaContext] = useState("")
  const [systemPrompt, setSystemPrompt] = useState("")
  const [isRenaming, setIsRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState("")
  const { mode, canToggle, toggleMode } = useQueryMode('auto')
  const [singleConnectionId, setSingleConnectionId] = useState<string>("")
  const [selectedConnectionIds, setSelectedConnectionIds] = useState<string[]>([])
  const [showSelector, setShowSelector] = useState(false)

  const genericSessions = useMemo(() =>
    Object.values(memorySessions)
      .filter(session => session.metadata?.chatType === 'generic')
      .sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0)),
  [memorySessions]
  )

  const activeSession = activeSessionId ? memorySessions[activeSessionId] : undefined
  const activeMessages = useMemo(
    () => activeSession?.messages ?? [],
    [activeSession?.messages]
  )

  // Ensure a generic session exists when the sidebar opens
  useEffect(() => {
    if (!open) {
      return
    }

    void hydrateMemoriesFromBackend()

    if (genericSessions.length === 0) {
      const id = startNewSession({
        title: `Chat Session ${new Date().toLocaleString()}`,
        metadata: { chatType: 'generic' },
      })
      setActiveSession(id)
      return
    }

    if (!activeSession || activeSession.metadata?.chatType !== 'generic') {
      setActiveSession(genericSessions[0].id)
    }
  }, [open, activeSession, genericSessions, hydrateMemoriesFromBackend, setActiveSession, startNewSession])

  // Initialize selections when opening
  useEffect(() => {
    if (!open) return
    if (connections.length > 0 && !singleConnectionId) {
      setSingleConnectionId(connections[0].id)
    }
    if (connections.length > 0 && selectedConnectionIds.length === 0) {
      setSelectedConnectionIds(connections.map(c => c.id))
    }
  }, [open, connections, singleConnectionId, selectedConnectionIds.length])

  // Auto-generate schema context based on selection
  useEffect(() => {
    try {
      if (mode === 'multi' && selectedConnectionIds.length > 0) {
        const conns = connections.filter(c => selectedConnectionIds.includes(c.id))
        const ctx = AISchemaContextBuilder.buildMultiDatabaseContext(conns, schemasMap, undefined)
        setSchemaContext(AISchemaContextBuilder.generateCompactSchemaContext(ctx))
      } else if (mode === 'single' && singleConnectionId) {
        const conn = connections.find(c => c.id === singleConnectionId)
        const schemas = conn ? (schemasMap.get(conn.id) || schemasMap.get(conn.name) || []) : []
        if (conn && schemas.length > 0) {
          const ctx = AISchemaContextBuilder.buildSingleDatabaseContext(conn, schemas)
          setSchemaContext(AISchemaContextBuilder.generateCompactSchemaContext(ctx))
        }
      }
    } catch {
      // ignore
    }
  }, [mode, singleConnectionId, selectedConnectionIds, connections, schemasMap])

  const handleCreateSession = () => {
    const id = startNewSession({
      title: `Chat Session ${new Date().toLocaleString()}`,
      metadata: { chatType: 'generic' },
    })
    setActiveSession(id)
  }

  const handleRenameSession = () => {
    if (activeSession && renameValue.trim()) {
      renameSession(activeSession.id, renameValue.trim())
      setIsRenaming(false)
    }
  }

  const handleSendMessage = async () => {
    if (!message.trim()) {
      return
    }

    const combinedContext = [schemaContext.trim()]
      .filter(Boolean)
      .join("\n\n")

    try {
      await sendGenericMessage(message.trim(), {
        context: combinedContext,
        systemPrompt: systemPrompt.trim() || undefined,
        metadata: {
          source: 'generic-chat-sidebar',
        },
      })
      setMessage("")
    } catch {
      // Errors are surfaced via lastError
    }
  }

  const allowSend = message.trim().length > 0 && !isGenerating

  const handleClose = () => {
    setSchemaContext("")
    setSystemPrompt("")
    setMessage("")
    setIsRenaming(false)
    setRenameValue("")
    onClose()
  }

  const handleToggleRename = () => {
    if (!activeSession) {
      return
    }
    if (!isRenaming) {
      setRenameValue(activeSession.title)
      setIsRenaming(true)
    } else {
      setIsRenaming(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={isOpen => {
      if (!isOpen) {
        handleClose()
      }
    }}>
      <SheetContent
        side="right"
        className="w-[600px] sm:max-w-[600px] m-4 h-[calc(100vh-2rem)] rounded-xl shadow-2xl border overflow-hidden flex flex-col"
      >
        <SheetHeader className="border-b pb-4 space-y-2">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 space-y-1">
              <SheetTitle className="flex items-center gap-2 text-left">
                <MessageCircle className="h-5 w-5 text-primary" />
                Generic AI Chat
              </SheetTitle>
              <SheetDescription className="text-left">
                Ask questions, brainstorm ideas, or discuss SQL outputs with the configured AI provider.
              </SheetDescription>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="gap-1">
                <Sparkles className="h-3 w-3" />
                {config.provider}
              </Badge>
              <Badge variant="secondary" className="gap-1">
                {config.selectedModel || 'model'}
              </Badge>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Select
              value={activeSession?.id ?? ''}
              onValueChange={value => {
                setActiveSession(value)
              }}
            >
              <SelectTrigger className="w-full" disabled={genericSessions.length === 0}>
                <SelectValue placeholder="Select session" />
              </SelectTrigger>
              <SelectContent>
                {genericSessions.map(session => (
                  <SelectItem key={session.id} value={session.id}>
                    {session.title}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              size="icon"
              variant="ghost"
              onClick={handleToggleRename}
              disabled={!activeSession}
              title="Rename session"
            >
              <Pencil className="h-4 w-4" />
            </Button>
            <Button size="icon" onClick={handleCreateSession} title="New chat session">
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          {/* Mode & database selectors */}
          <div className="flex items-center gap-2">
            {canToggle && (
              <div className="flex items-center rounded-md border bg-background overflow-hidden">
                <Button
                  variant={mode === 'single' ? 'default' : 'ghost'}
                  size="sm"
                  className="h-8 px-2 text-xs"
                  onClick={() => mode === 'multi' && toggleMode()}
                >
                  Single
                </Button>
                <Button
                  variant={mode === 'multi' ? 'default' : 'ghost'}
                  size="sm"
                  className="h-8 px-2 text-xs"
                  onClick={() => mode === 'single' && toggleMode()}
                >
                  Multi
                </Button>
              </div>
            )}

            {mode === 'single' ? (
              <Select
                value={singleConnectionId}
                onValueChange={(value) => setSingleConnectionId(value)}
                disabled={connections.length === 0}
              >
                <SelectTrigger className="h-8 w-44 text-xs">
                  <SelectValue placeholder="Select database" />
                </SelectTrigger>
                <SelectContent>
                  {connections.map(conn => (
                    <SelectItem key={conn.id} value={conn.id}>
                      <div className="flex items-center gap-2 text-xs">
                        <Database className="h-3 w-3" />
                        <span className="flex-1">{conn.name || conn.database}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="h-8 px-2 text-xs"
                onClick={() => setShowSelector(true)}
              >
                <Network className="h-3 w-3 mr-1" />
                {selectedConnectionIds.length}/{connections.length} DBs
              </Button>
            )}
          </div>

          {isRenaming && activeSession && (
            <div className="flex items-center gap-2">
              <Input value={renameValue} onChange={event => setRenameValue(event.target.value)} placeholder="Rename session" />
              <Button size="sm" onClick={handleRenameSession}>
                Save
              </Button>
            </div>
          )}
        </SheetHeader>

        <div className="flex flex-1 min-h-0">
          <div className="flex w-full flex-col gap-4 p-4">
            {activeMessages.length === 0 ? (
              <div className="flex-1 rounded-lg border bg-muted/20 p-3">
                <EmptyState
                  icon={MessageCircle}
                  title="Start a conversation"
                  description="Ask questions, brainstorm ideas, or discuss your SQL query results with AI."
                  examples={CHAT_EXAMPLES}
                  onExampleClick={(query) => setMessage(query)}
                  className="h-full py-6"
                />
              </div>
            ) : (
              <VirtualMessageList
                messages={activeMessages}
                renderMessage={(messageEntry) => (
                  <div
                    className={cn(
                      "max-w-[85%] rounded-lg border px-3 py-2 text-sm shadow-sm mx-3 my-2",
                      messageEntry.role === 'assistant'
                        ? "ml-auto bg-primary/5 border-primary/30"
                        : "mr-auto bg-background"
                    )}
                  >
                    <div className="mb-1 flex items-center gap-2 text-xs uppercase text-muted-foreground">
                      <span>{messageEntry.role}</span>
                      {messageEntry.metadata?.provider ? (
                        <Badge variant="outline" className="text-[10px]">
                          {String(messageEntry.metadata.provider)}
                        </Badge>
                      ) : null}
                    </div>
                    <div className="whitespace-pre-wrap leading-relaxed text-sm">
                      {messageEntry.content}
                    </div>
                  </div>
                )}
                getMessageKey={(messageEntry) => messageEntry.id}
                estimateSize={80}
                overscan={3}
                className="flex-1 rounded-lg border bg-muted/20"
                autoScroll={true}
              />
            )}

            <div className="space-y-4">
              <SchemaContextSelector
                connections={connections}
                schemasMap={schemasMap}
                onChange={setSchemaContext}
              />

              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground" htmlFor="generic-chat-system">
                  Optional system instructions
                </label>
                <Input
                  id="generic-chat-system"
                  placeholder="Guide the assistant's behaviour (e.g. 'Explain results for a non-technical stakeholder')"
                  value={systemPrompt}
                  onChange={event => setSystemPrompt(event.target.value)}
                />
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground" htmlFor="generic-chat-message">
                  Ask a question or share instructions
                </label>
                <Textarea
                  id="generic-chat-message"
                  value={message}
                  onChange={event => setMessage(event.target.value)}
                  placeholder="e.g. Summarise the latest generated SQL query in plain English"
                  onKeyDown={event => {
                    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                      event.preventDefault()
                      if (allowSend) {
                        void handleSendMessage()
                      }
                    }
                  }}
                  className="min-h-[120px]"
                />
              </div>

              {lastError && (
                <Alert variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertTitle>Chat error</AlertTitle>
                  <AlertDescription>{lastError}</AlertDescription>
                </Alert>
              )}

              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground">
                  Press Cmd/Ctrl + Enter to send
                </span>
                <Button onClick={() => void handleSendMessage()} disabled={!allowSend}>
                  {isGenerating ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <SendHorizontal className="mr-2 h-4 w-4" />
                  )}
                  Send
                </Button>
              </div>
            </div>
          </div>
        </div>
      </SheetContent>
      {mode === 'multi' && (
        <MultiDBConnectionSelector
          open={showSelector}
          onClose={() => setShowSelector(false)}
          selectedConnectionIds={selectedConnectionIds}
          onSelectionChange={(ids) => setSelectedConnectionIds(ids)}
          filteredConnections={connections}
        />
      )}
    </Sheet>
  )
}
