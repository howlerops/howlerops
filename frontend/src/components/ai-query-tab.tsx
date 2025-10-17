import { useEffect, useMemo, useRef, useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { AISchemaContextBuilder } from "@/lib/ai-schema-context"
import { cn } from "@/lib/utils"
import { useAIConfig } from "@/store/ai-store"
import { useAIQueryAgentStore, type AgentAttachment, type AgentMessage, type AgentResultAttachment } from "@/store/ai-query-agent-store"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import type { DatabaseConnection } from "@/store/connection-store"
import { showHybridNotification } from "@/lib/wails-ai-api"

import { Download, Loader2, MessageSquare, Pencil, Play, SendHorizontal, Sparkles, Table2, Wand2, Copy } from "lucide-react"

interface AIQueryTabViewProps {
  tab: {
    id: string
    title: string
    aiSessionId?: string
    connectionId?: string
    selectedConnectionIds?: string[]
  }
  connections: DatabaseConnection[]
  schemasMap: Map<string, SchemaNode[]>
  onSelectConnection: (connectionId: string) => void
  onUseSQL: (sql: string, connectionId?: string) => void
  onRenameSession: (sessionId: string, title: string) => void
}

function downloadCSV(result: AgentResultAttachment, filename: string) {
  const escape = (value: unknown) => {
    if (value === null || value === undefined) {
      return ''
    }
    const str = String(value)
    if (/[",\n]/.test(str)) {
      return `"${str.replace(/"/g, '""')}"`
    }
    return str
  }

  const header = result.columns.map(col => escape(col)).join(',')
  const rows = result.rows.map(row =>
    result.columns.map(col => escape(row[col])).join(',')
  )

  const csv = [header, ...rows].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function formatTimestamp(timestamp: number): string {
  try {
    return new Date(timestamp).toLocaleTimeString()
  } catch {
    return ''
  }
}

export function AIQueryTabView({
  tab,
  connections,
  schemasMap,
  onSelectConnection,
  onUseSQL,
  onRenameSession,
}: AIQueryTabViewProps) {
  const { config: aiConfig } = useAIConfig()
  const sendMessage = useAIQueryAgentStore(state => state.sendMessage)
  const setActiveSession = useAIQueryAgentStore(state => state.setActiveSession)
  const session = useAIQueryAgentStore(state => (tab.aiSessionId ? state.sessions[tab.aiSessionId] : undefined))
  const streamingTurnId = useAIQueryAgentStore(state => state.streamingTurnId)

  const [input, setInput] = useState("")
  const [isRenaming, setIsRenaming] = useState(false)
  const [draftTitle, setDraftTitle] = useState<string | null>(null)

  const endRef = useRef<HTMLDivElement | null>(null)

  const activeConnection = useMemo(
    () => connections.find(connection => connection.id === tab.connectionId),
    [connections, tab.connectionId],
  )

  useEffect(() => {
    if (session?.id) {
      setActiveSession(session.id)
    }
  }, [session?.id, setActiveSession])

  useEffect(() => {
    if (endRef.current) {
      endRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' })
    }
  }, [session?.messages.length, streamingTurnId])

  const schemaContext = useMemo(() => {
    if (tab.selectedConnectionIds && tab.selectedConnectionIds.length > 1) {
      const context = AISchemaContextBuilder.buildMultiDatabaseContext(
        connections.filter(conn => tab.selectedConnectionIds?.includes(conn.id)),
        schemasMap,
        tab.connectionId,
      )
      return AISchemaContextBuilder.generateCompactSchemaContext(context)
    }

    if (activeConnection) {
      const schemas = schemasMap.get(activeConnection.id) || schemasMap.get(activeConnection.name) || []
      if (schemas.length > 0) {
        const context = AISchemaContextBuilder.buildSingleDatabaseContext(activeConnection, schemas)
        return AISchemaContextBuilder.generateCompactSchemaContext(context)
      }
    }

    return ''
  }, [activeConnection, connections, schemasMap, tab.connectionId, tab.selectedConnectionIds])

  const handleSend = async () => {
    if (!tab.aiSessionId || !input.trim()) {
      return
    }

    try {
      await sendMessage({
        sessionId: tab.aiSessionId,
        message: input.trim(),
        provider: aiConfig.provider,
        model: aiConfig.selectedModel,
        connectionId: tab.connectionId,
        connectionIds: tab.selectedConnectionIds,
        schemaContext: schemaContext || undefined,
        maxRows: 200,
        temperature: aiConfig.temperature,
        maxTokens: aiConfig.maxTokens,
      })
      setInput("")
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Failed to send AI query'
      showHybridNotification('AI Query Agent Error', description, true)
    }
  }

  const handleCopySQL = async (sql: string) => {
    try {
      await navigator.clipboard.writeText(sql)
      showHybridNotification('SQL Copied', 'The generated SQL has been copied to your clipboard.', false)
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Unable to access clipboard'
      showHybridNotification('Copy failed', description, true)
    }
  }

  const handleExportResult = (attachment: AgentResultAttachment) => {
    const filename = `ai-query-result-${new Date().toISOString().replace(/[:.]/g, '-')}.csv`
    downloadCSV(attachment, filename)
  }

  const handleRename = () => {
    if (!tab.aiSessionId) {
      return
    }

    const trimmed = (draftTitle ?? '').trim()
    if (trimmed.length === 0) {
      return
    }

    onRenameSession(tab.aiSessionId, trimmed)
    setIsRenaming(false)
    setDraftTitle(null)
  }

  const renderAttachment = (attachment: AgentAttachment, index: number) => {
    switch (attachment.type) {
      case 'sql':
        if (!attachment.sql) {
          return null
        }
        return (
          <div key={`sql-${index}`} className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm space-y-3">
            <div className="flex items-center justify-between text-sm font-medium">
              <span className="flex items-center gap-2 text-primary">
                <Sparkles className="h-4 w-4" />
                Generated SQL
              </span>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="icon" onClick={() => handleCopySQL(attachment.sql!.query)} title="Copy SQL">
                  <Copy className="h-4 w-4" />
                </Button>
                <Button size="sm" onClick={() => onUseSQL(attachment.sql!.query, attachment.sql?.connectionId)}>
                  <Play className="h-4 w-4 mr-2" />
                  Use in Editor
                </Button>
              </div>
            </div>
            <pre className="rounded-md bg-muted/60 p-3 text-xs font-mono whitespace-pre-wrap border border-border/40">
              {attachment.sql.query}
            </pre>
            {attachment.sql.explanation && (
              <p className="text-xs text-muted-foreground">
                {attachment.sql.explanation}
              </p>
            )}
          </div>
        )
      case 'result':
        if (!attachment.result) {
          return null
        }
        return (
          <div key={`result-${index}`} className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm space-y-3">
            <div className="flex items-center justify-between text-sm font-medium">
              <span className="flex items-center gap-2 text-primary">
                <Table2 className="h-4 w-4" />
                Result Preview ({attachment.result.rowCount} rows)
              </span>
              <Button variant="outline" size="sm" onClick={() => handleExportResult(attachment.result!)}>
                <Download className="h-4 w-4 mr-2" />
                Export CSV
              </Button>
            </div>
            <div className="overflow-x-auto rounded-md border border-border/40">
              <table className="w-full text-xs">
                <thead className="bg-muted/60">
                  <tr>
                    {attachment.result.columns.map(column => (
                      <th key={column} className="px-2 py-1 text-left font-semibold text-muted-foreground">
                        {column}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {attachment.result.rows.slice(0, 20).map((row, rowIndex) => (
                    <tr key={rowIndex} className="border-t border-border/30">
                      {attachment.result!.columns.map(column => (
                        <td key={column} className="px-2 py-1 text-muted-foreground">
                          {String(row[column] ?? '')}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {attachment.result.rows.length > 20 && (
              <p className="text-xs text-muted-foreground">
                Showing the first 20 rows. Export for the full result set.
              </p>
            )}
          </div>
        )
      case 'chart':
        if (!attachment.chart) {
          return null
        }
        return (
          <div key={`chart-${index}`} className="rounded-xl border border-border/60 bg-background/70 p-3 shadow-sm text-xs text-muted-foreground space-y-1">
            <div className="flex items-center gap-2 text-sm font-semibold text-primary">
              <Wand2 className="h-4 w-4" />
              Suggested Chart
            </div>
            <p><strong>Type:</strong> {attachment.chart.type}</p>
            <p><strong>X Axis:</strong> {attachment.chart.xField}</p>
            <p><strong>Y Axis:</strong> {attachment.chart.yFields.join(', ')}</p>
            {attachment.chart.seriesField && <p><strong>Series:</strong> {attachment.chart.seriesField}</p>}
            {attachment.chart.description && <p>{attachment.chart.description}</p>}
            <p className="italic">Use this as guidance when building a visualization.</p>
          </div>
        )
      case 'report':
        if (!attachment.report) {
          return null
        }
        return (
          <div key={`report-${index}`} className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm text-sm text-muted-foreground">
            <div className="flex items-center gap-2 text-sm font-semibold text-primary mb-2">
              <MessageSquare className="h-4 w-4" />
              Report Draft
            </div>
            <pre className="whitespace-pre-wrap font-sans">{attachment.report.body}</pre>
          </div>
        )
      case 'insight':
        if (!attachment.insight) {
          return null
        }
        return (
          <div key={`insight-${index}`} className="rounded-xl border border-border/60 bg-background/80 p-3 shadow-sm">
            <div className="flex items-center gap-2 text-sm font-semibold text-primary mb-2">
              <Sparkles className="h-4 w-4" />
              Key Insights
            </div>
            <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
              {attachment.insight.highlights.map((insight, idx) => (
                <li key={idx}>{insight}</li>
              ))}
            </ul>
          </div>
        )
      default:
        return null
    }
  }

  const renderMessage = (message: AgentMessage) => {
    const isUser = message.role === 'user'
    const bubbleClasses = isUser
      ? "bg-primary text-primary-foreground rounded-2xl rounded-tr-none"
      : "bg-muted text-foreground rounded-2xl rounded-tl-none"

    const metaLabel = isUser ? "You" : message.agent.replace(/_/g, " ")

    return (
      <div key={message.id} className={cn("flex w-full", isUser ? "justify-end" : "justify-start")}>
        <div className={cn("max-w-[80%] space-y-2", isUser ? "items-end text-right" : "items-start text-left")}>
          <div className="flex items-center gap-2 text-[11px] uppercase tracking-wide text-muted-foreground">
            <span>{metaLabel}</span>
            <span>•</span>
            <span>{formatTimestamp(message.createdAt)}</span>
            {!isUser && message.provider && (
              <Badge variant="outline" className="text-[10px] capitalize">
                {message.provider}
              </Badge>
            )}
            {!isUser && message.model && (
              <Badge variant="outline" className="text-[10px]">
                {message.model}
              </Badge>
            )}
          </div>

          <div className={cn("px-4 py-3 shadow-sm whitespace-pre-wrap text-sm", bubbleClasses)}>
            {message.content}
          </div>

          {message.attachments && message.attachments.length > 0 && (
            <div className={cn("space-y-2", isUser ? "items-end text-right" : "items-start text-left")}>
              {message.attachments.map((attachment, index) => renderAttachment(attachment, index))}
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b bg-muted/40 px-4 py-3 flex items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" />
            <span className="text-sm font-semibold">AI Query Agent</span>
          </div>
          <div className="flex items-center gap-2">
            {isRenaming ? (
              <>
                <Input
                  value={draftTitle ?? session?.title ?? tab.title}
                  onChange={(event) => setDraftTitle(event.target.value)}
                  className="h-8 w-64"
                  autoFocus
                />
                <Button size="sm" onClick={handleRename}>
                  Save
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setIsRenaming(false)
                    setDraftTitle(null)
                  }}
                >
                  Cancel
                </Button>
              </>
            ) : (
              <>
                <p className="text-sm font-medium text-muted-foreground">{session?.title ?? tab.title}</p>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => {
                    setDraftTitle(session?.title ?? tab.title)
                    setIsRenaming(true)
                  }}
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="capitalize gap-1">
            <Sparkles className="h-3 w-3" />
            {aiConfig.provider}
          </Badge>
          <Badge variant="outline" className="gap-1">
            {aiConfig.selectedModel || 'model'}
          </Badge>
          <Select
            value={tab.connectionId ?? ''}
            onValueChange={value => onSelectConnection(value)}
            disabled={connections.length === 0}
          >
            <SelectTrigger className="w-[220px]" disabled={connections.length === 0}>
              <SelectValue placeholder="Select connection" />
            </SelectTrigger>
            <SelectContent>
              {connections.length === 0 ? (
                <SelectItem value="__no-connection" disabled>
                  No connections available
                </SelectItem>
              ) : (
                connections.map(connection => (
                  <SelectItem key={connection.id} value={connection.id}>
                    {connection.name || connection.database}
                  </SelectItem>
                ))
              )}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex-1 overflow-hidden">
        <ScrollArea className="h-full px-4 py-6">
          <div className="space-y-4">
            {session && session.messages.length === 0 && (
              <div className="rounded-lg border bg-muted/30 p-6 text-sm text-muted-foreground">
                <p className="font-medium">Start the conversation</p>
                <p className="mt-2">
                  Ask natural language questions like "Show the top performing products over the last 7 days"
                  and I'll generate SQL, run it safely in read-only mode, and provide insights.
                </p>
              </div>
            )}

            {session?.messages.map(renderMessage)}
            <div ref={endRef} />
          </div>
        </ScrollArea>
      </div>

      <div className="border-t bg-background/80 px-4 py-3">
        <div className="space-y-3">
          <Textarea
            value={input}
            onChange={event => setInput(event.target.value)}
            placeholder={activeConnection
              ? `Ask about ${activeConnection.name || activeConnection.database}...`
              : 'Select a connection to get started'}
            className="min-h-[120px] resize-none text-sm"
            disabled={session?.status === 'streaming'}
            onKeyDown={event => {
              if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                event.preventDefault()
                void handleSend()
              }
            }}
          />
          <div className="flex items-center justify-between">
            <div className="text-xs text-muted-foreground">
              {schemaContext ? 'Schema context attached for the AI request.' : 'Schema context will be added automatically when available.'}
            </div>
            <Button
              onClick={handleSend}
              disabled={session?.status === 'streaming' || !input.trim()}
            >
              {session?.status === 'streaming' ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Thinking...
                </>
              ) : (
                <>
                  <SendHorizontal className="h-4 w-4 mr-2" />
                  Send
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
