import { useEffect, useMemo, useRef, useState } from "react"

import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { Input } from "@/components/ui/input"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { SchemaContextSelector } from "@/components/schema-context-selector"
import { useAIGeneration, useAIConfig } from "@/store/ai-store"
import { useAIMemoryStore } from "@/store/ai-memory-store"
import type { DatabaseConnection } from "@/store/connection-store"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { cn } from "@/lib/utils"
import { AlertCircle, Loader2, MessageCircle, Pencil, Plus, SendHorizontal, Sparkles } from "lucide-react"

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
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

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

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' })
    }
  }, [activeMessages])

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
            <ScrollArea className="flex-1 rounded-lg border bg-muted/20 p-3">
              <div className="space-y-3">
                {activeMessages.length === 0 && (
                  <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                    Start the conversation by asking a question or sharing context. Previous chat history will appear here.
                  </div>
                )}
                {activeMessages.map(messageEntry => (
                  <div
                    key={messageEntry.id}
                    className={cn(
                      "max-w-[85%] rounded-lg border px-3 py-2 text-sm shadow-sm",
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
                ))}
                <div ref={messagesEndRef} />
              </div>
            </ScrollArea>

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
    </Sheet>
  )
}
