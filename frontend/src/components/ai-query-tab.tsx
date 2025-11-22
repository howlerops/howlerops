import { BarChart3, Copy, Download, GitCompare, Loader2, MessageSquare, Pencil, Play, Search, SendHorizontal, Sparkles, Table2, TrendingUp, Wand2 } from "lucide-react"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import { EmptyState, type ExampleQuery } from "@/components/empty-states/EmptyState"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { VirtualMessageList } from "@/components/virtual-message-list"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { AISchemaContextBuilder } from "@/lib/ai-schema-context"
import { cn } from "@/lib/utils"
import { showHybridNotification } from "@/lib/wails-ai-api"
import { type AgentAttachment, type AgentMessage, type AgentResultAttachment,useAIQueryAgentStore } from "@/store/ai-query-agent-store"
import { useAIConfig } from "@/store/ai-store"
import type { DatabaseConnection } from "@/store/connection-store"

import { SQLAttachment } from "./ai-query-tab/SQLAttachment"
import { ResultAttachment } from "./ai-query-tab/ResultAttachment"
import { ChartAttachment } from "./ai-query-tab/ChartAttachment"
import { ReportAttachment } from "./ai-query-tab/ReportAttachment"
import { InsightAttachment } from "./ai-query-tab/InsightAttachment"

const DEFAULT_EXAMPLES: ExampleQuery[] = [
  {
    label: "Show all tables",
    description: "Get a list of all tables in the database",
    query: "Show me all tables in this database",
    icon: Table2,
  },
  {
    label: "Top customers by revenue",
    description: "Find the highest revenue generating customers",
    query: "Find the top 10 customers by total revenue",
    icon: TrendingUp,
  },
  {
    label: "Table relationships",
    description: "Understand how tables connect via foreign keys",
    query: "Explain the relationship between users and orders tables",
    icon: GitCompare,
  },
  {
    label: "Recent activity",
    description: "Query recent records or transactions",
    query: "Show me the most recent 20 orders from the last 7 days",
    icon: Search,
  },
  {
    label: "Data statistics",
    description: "Get aggregated statistics about your data",
    query: "What are the total sales by product category this month?",
    icon: BarChart3,
  },
]

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

  const activeConnection = useMemo(
    () => connections.find(connection => connection.id === tab.connectionId),
    [connections, tab.connectionId],
  )

  useEffect(() => {
    if (session?.id) {
      setActiveSession(session.id)
    }
  }, [session?.id, setActiveSession])

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

  const handleSend = useCallback(async () => {
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
  }, [tab.aiSessionId, tab.connectionId, tab.selectedConnectionIds, input, sendMessage, aiConfig, schemaContext])

  const handleCopySQL = useCallback(async (sql: string) => {
    try {
      await navigator.clipboard.writeText(sql)
      showHybridNotification('SQL Copied', 'The generated SQL has been copied to your clipboard.', false)
    } catch (error) {
      const description = error instanceof Error ? error.message : 'Unable to access clipboard'
      showHybridNotification('Copy failed', description, true)
    }
  }, [])

  const handleExportResult = useCallback((attachment: AgentResultAttachment) => {
    const filename = `ai-query-result-${new Date().toISOString().replace(/[:.]/g, '-')}.csv`
    downloadCSV(attachment, filename)
  }, [])

  const handleRename = useCallback(() => {
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
  }, [tab.aiSessionId, draftTitle, onRenameSession])

  const renderAttachment = useCallback((attachment: AgentAttachment, index: number) => {
    switch (attachment.type) {
      case 'sql':
        return <SQLAttachment key={`sql-${index}`} attachment={attachment} onCopySQL={handleCopySQL} onUseSQL={onUseSQL} />
      case 'result':
        return <ResultAttachment key={`result-${index}`} attachment={attachment} onExport={handleExportResult} />
      case 'chart':
        return <ChartAttachment key={`chart-${index}`} attachment={attachment} />
      case 'report':
        return <ReportAttachment key={`report-${index}`} attachment={attachment} />
      case 'insight':
        return <InsightAttachment key={`insight-${index}`} attachment={attachment} />
      default:
        return null
    }
  }, [handleCopySQL, handleExportResult, onUseSQL])

  const renderMessage = useCallback((message: AgentMessage) => {
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
  }, [renderAttachment])

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
        {session && session.messages.length === 0 ? (
          <div className="h-full">
            <EmptyState
              icon={Sparkles}
              title="Start your AI conversation"
              description="Ask natural language questions and I'll generate SQL, execute it safely in read-only mode, and provide insights about your data."
              examples={DEFAULT_EXAMPLES}
              onExampleClick={(query) => setInput(query)}
              className="h-full"
            />
          </div>
        ) : (
          <VirtualMessageList
            messages={session?.messages ?? []}
            renderMessage={(message) => (
              <div className="px-4 py-2">
                {renderMessage(message)}
              </div>
            )}
            getMessageKey={(message) => message.id}
            estimateSize={150}
            overscan={3}
            className="h-full"
            autoScroll={true}
          />
        )}
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
