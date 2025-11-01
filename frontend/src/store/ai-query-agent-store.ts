import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { StreamAIQueryAgent } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useAIMemoryStore } from '@/store/ai-memory-store'
import { showHybridNotification } from '@/lib/wails-ai-api'

export type AgentAttachmentType = 'sql' | 'result' | 'chart' | 'report' | 'insight' | string

export interface AgentSQLAttachment {
  query: string
  explanation?: string
  confidence?: number
  connectionId?: string
  warnings?: string[]
}

export interface AgentResultAttachment {
  columns: string[]
  rows: Record<string, unknown>[]
  rowCount: number
  executionTimeMs: number
  limited: boolean
  connectionId?: string
}

export interface AgentChartAttachment {
  type: string
  xField: string
  yFields: string[]
  seriesField?: string
  title?: string
  description?: string
  recommended?: boolean
}

export interface AgentReportAttachment {
  format: string
  body: string
  title?: string
}

export interface AgentInsightAttachment {
  highlights: string[]
}

export type AgentAttachment =
  | { type: 'sql'; sql: AgentSQLAttachment }
  | { type: 'result'; result: AgentResultAttachment }
  | { type: 'chart'; chart: AgentChartAttachment }
  | { type: 'report'; report: AgentReportAttachment }
  | { type: 'insight'; insight: AgentInsightAttachment }
  | { type: AgentAttachmentType; sql?: AgentSQLAttachment; result?: AgentResultAttachment; chart?: AgentChartAttachment; report?: AgentReportAttachment; insight?: AgentInsightAttachment; rawPayload?: Record<string, unknown> }

export interface AgentMessage {
  id: string
  agent: string
  role: 'user' | 'assistant'
  title?: string
  content: string
  createdAt: number
  attachments?: AgentAttachment[]
  metadata?: Record<string, unknown>
  warnings?: string[]
  error?: string
  provider?: string
  model?: string
  tokensUsed?: number
  elapsedMs?: number
}

export interface AgentSession {
  id: string
  title: string
  createdAt: number
  updatedAt: number
  messages: AgentMessage[]
  status: 'idle' | 'streaming' | 'error'
  lastError?: string
  turnId?: string
  provider?: string
  model?: string
}

interface StreamPayload {
  sessionId?: string
  turnId?: string
  status?: string
  message?: unknown
  error?: string
}

interface SendMessageOptions {
  sessionId: string
  message: string
  provider: string
  model: string
  connectionId?: string
  connectionIds?: string[]
  schemaContext?: string
  context?: string
  temperature?: number
  maxTokens?: number
  maxRows?: number
}

interface AIQueryAgentState {
  sessions: Record<string, AgentSession>
  activeSessionId: string | null
  streamingTurnId?: string
  isHydrated: boolean
  createSession: (options?: { title?: string; provider?: string; model?: string }) => string
  ensureSession: (sessionId: string) => void
  setActiveSession: (sessionId: string | null) => void
  receiveEvent: (payload: StreamPayload) => void
  sendMessage: (options: SendMessageOptions) => Promise<void>
  setSessionConnection: (sessionId: string, connectionId?: string) => void
  syncFromMemoryStore: () => void
  renameSession: (sessionId: string, title: string) => void
  clearSession: (sessionId: string) => void
}

function now(): number {
  return Date.now()
}

function normaliseAttachment(payload: unknown): AgentAttachment {
  if (!payload || typeof payload !== 'object') {
    return { type: 'unknown' }
  }

  const data = payload as Record<string, unknown>
  const type = typeof data.type === 'string' ? data.type : 'unknown'
  const attachment: AgentAttachment = {
    type,
    rawPayload: data,
  }

  const sqlPayload = data.sql
  if (sqlPayload && typeof sqlPayload === 'object') {
    const sql = sqlPayload as Record<string, unknown>
    attachment.sql = {
      query: String(sql.query ?? ''),
      explanation: sql.explanation ? String(sql.explanation) : undefined,
      confidence: typeof sql.confidence === 'number' ? sql.confidence : undefined,
      connectionId: sql.connectionId ? String(sql.connectionId) : undefined,
      warnings: Array.isArray(sql.warnings) ? (sql.warnings as unknown[]).map(String) : undefined,
    }
  }

  const resultPayload = data.result
  if (resultPayload && typeof resultPayload === 'object') {
    const result = resultPayload as Record<string, unknown>
    attachment.result = {
      columns: Array.isArray(result.columns) ? (result.columns as unknown[]).map(String) : [],
      rows: Array.isArray(result.rows) ? (result.rows as Record<string, unknown>[]).map(row => ({ ...row })) : [],
      rowCount: Number(result.rowCount ?? 0),
      executionTimeMs: Number(result.executionTimeMs ?? 0),
      limited: Boolean(result.limited),
      connectionId: result.connectionId ? String(result.connectionId) : undefined,
    }
  }

  const chartPayload = data.chart
  if (chartPayload && typeof chartPayload === 'object') {
    const chart = chartPayload as Record<string, unknown>
    attachment.chart = {
      type: String(chart.type ?? ''),
      xField: String(chart.xField ?? ''),
      yFields: Array.isArray(chart.yFields) ? (chart.yFields as unknown[]).map(String) : [],
      seriesField: chart.seriesField ? String(chart.seriesField) : undefined,
      title: chart.title ? String(chart.title) : undefined,
      description: chart.description ? String(chart.description) : undefined,
      recommended: chart.recommended !== undefined ? Boolean(chart.recommended) : undefined,
    }
  }

  const reportPayload = data.report
  if (reportPayload && typeof reportPayload === 'object') {
    const report = reportPayload as Record<string, unknown>
    attachment.report = {
      format: String(report.format ?? 'markdown'),
      body: String(report.body ?? ''),
      title: report.title ? String(report.title) : undefined,
    }
  }

  const insightPayload = data.insight
  if (insightPayload && typeof insightPayload === 'object') {
    const insight = insightPayload as Record<string, unknown>
    attachment.insight = {
      highlights: Array.isArray(insight.highlights) ? (insight.highlights as unknown[]).map(String) : [],
    }
  }

  return attachment
}

function normaliseMessage(payload: unknown): AgentMessage | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  const data = payload as Record<string, unknown>
  const attachments = Array.isArray(data.attachments)
    ? (data.attachments as unknown[]).map(normaliseAttachment)
    : undefined

  return {
    id: String(data.id ?? crypto.randomUUID()),
    agent: String(data.agent ?? 'assistant'),
    role: (data.role === 'user' ? 'user' : 'assistant'),
    title: data.title ? String(data.title) : undefined,
    content: String(data.content ?? ''),
    createdAt: Number(data.createdAt ?? now()),
    attachments,
    metadata: data.metadata && typeof data.metadata === 'object' ? { ...(data.metadata as Record<string, unknown>) } : undefined,
    warnings: Array.isArray(data.warnings) ? (data.warnings as unknown[]).map(String) : undefined,
    error: data.error ? String(data.error) : undefined,
    provider: data.provider ? String(data.provider) : undefined,
    model: data.model ? String(data.model) : undefined,
    tokensUsed: typeof data.tokensUsed === 'number' ? data.tokensUsed : undefined,
    elapsedMs: typeof data.elapsedMs === 'number' ? data.elapsedMs : undefined,
  }
}

function appendMessage(state: AIQueryAgentState, sessionId: string, message: AgentMessage): AIQueryAgentState {
  const session = state.sessions[sessionId]
  if (!session) {
    return state
  }

  if (session.messages.some(existing => existing.id === message.id)) {
    return state
  }

  const updatedSession: AgentSession = {
    ...session,
    messages: [...session.messages, message],
    updatedAt: message.createdAt,
  }

  // Persist to memory store
  const memoryStore = useAIMemoryStore.getState()
  memoryStore.recordMessage({
    sessionId,
    role: message.role,
    content: message.content,
    metadata: {
      agent: message.agent,
      attachments: message.attachments,
      provider: message.provider,
      model: message.model,
      tokensUsed: message.tokensUsed,
    },
    timestamp: message.createdAt,
  })

  return {
    ...state,
    sessions: {
      ...state.sessions,
      [sessionId]: updatedSession,
    },
  }
}

export const useAIQueryAgentStore = create<AIQueryAgentState>()(
  persist(
    (set, get) => ({
  sessions: {},
  activeSessionId: null,
  streamingTurnId: undefined,
  isHydrated: false,

  createSession: (options) => {
    const title = options?.title ?? `AI Query ${new Date().toLocaleString()}`
    const memoryStore = useAIMemoryStore.getState()
    const sessionId = memoryStore.startNewSession({
      title,
      metadata: { chatType: 'query-agent' },
    })

    set(state => ({
      sessions: {
        ...state.sessions,
        [sessionId]: {
          id: sessionId,
          title,
          createdAt: now(),
          updatedAt: now(),
          messages: [],
          status: 'idle',
          provider: options?.provider,
          model: options?.model,
        },
      },
      activeSessionId: sessionId,
    }))

    memoryStore.setActiveSession(sessionId)
    return sessionId
  },

  ensureSession: (sessionId) => {
    set(state => {
      if (state.sessions[sessionId]) {
        return state
      }

      return {
        ...state,
        sessions: {
          ...state.sessions,
          [sessionId]: {
            id: sessionId,
            title: `AI Query ${new Date().toLocaleString()}`,
            createdAt: now(),
            updatedAt: now(),
            messages: [],
            status: 'idle',
          },
        },
      }
    })
  },

  setActiveSession: (sessionId) => {
    set({ activeSessionId: sessionId ?? null })
    const memoryStore = useAIMemoryStore.getState()
    if (sessionId) {
      memoryStore.setActiveSession(sessionId)
    }
  },

  setSessionConnection: (sessionId, connectionId) => {
    set(state => {
      const session = state.sessions[sessionId]
      if (!session) return state
      const memoryStore = useAIMemoryStore.getState()
      const existing = memoryStore.sessions[sessionId]
      if (existing) {
        memoryStore.sessions[sessionId] = {
          ...existing,
          metadata: {
            ...(existing.metadata ?? {}),
            connectionId,
          },
        }
      }
      return state
    })
  },

  renameSession: (sessionId, title) => {
    set(state => {
      const session = state.sessions[sessionId]
      if (!session) return state
      const memoryStore = useAIMemoryStore.getState()
      memoryStore.renameSession(sessionId, title)
      return {
        ...state,
        sessions: {
          ...state.sessions,
          [sessionId]: {
            ...session,
            title,
          },
        },
      }
    })
  },

  clearSession: (sessionId) => {
    set(state => {
      const rest = { ...state.sessions }
      delete rest[sessionId]
      const memoryStore = useAIMemoryStore.getState()
      memoryStore.deleteSession(sessionId)
      return {
        ...state,
        sessions: rest,
        activeSessionId: state.activeSessionId === sessionId ? null : state.activeSessionId,
      }
    })
  },

  receiveEvent: (payload) => {
    if (!payload || typeof payload !== 'object') {
      return
    }

    const sessionId = payload.sessionId
    if (!sessionId) {
      return
    }

    const status = (payload.status ?? '').toLowerCase()
    const message = normaliseMessage(payload.message)

    set(state => {
      const session = state.sessions[sessionId]
      if (!session) {
        return state
      }

      let nextState = state

      if (status === 'started') {
        nextState = {
          ...state,
          streamingTurnId: payload.turnId,
          sessions: {
            ...state.sessions,
            [sessionId]: {
              ...session,
              status: 'streaming',
              lastError: undefined,
              turnId: payload.turnId,
            },
          },
        }
      } else if (status === 'message' && message) {
        nextState = appendMessage(state, sessionId, message)
      } else if (status === 'error') {
        nextState = {
          ...state,
          streamingTurnId: undefined,
          sessions: {
            ...state.sessions,
            [sessionId]: {
              ...session,
              status: 'error',
              lastError: payload.error ?? 'Unknown error',
              turnId: undefined,
            },
          },
        }
      } else if (status === 'completed') {
        nextState = {
          ...state,
          streamingTurnId: undefined,
          sessions: {
            ...state.sessions,
            [sessionId]: {
              ...session,
              status: 'idle',
              turnId: undefined,
              lastError: undefined,
            },
          },
        }
      }

      return nextState
    })
  },

  sendMessage: async (options) => {
    const session = get().sessions[options.sessionId]
    if (!session) {
      throw new Error('Session not found')
    }

    set(state => ({
      ...state,
      streamingTurnId: undefined,
      sessions: {
        ...state.sessions,
        [options.sessionId]: {
          ...session,
          status: 'streaming',
          lastError: undefined,
        },
      },
    }))

    try {
      const response = await StreamAIQueryAgent(
        main.AIQueryAgentRequest.createFrom({
          sessionId: options.sessionId,
          message: options.message,
          provider: options.provider,
          model: options.model,
          connectionId: options.connectionId,
          connectionIds: options.connectionIds,
          schemaContext: options.schemaContext,
          context: options.context,
          temperature: options.temperature,
          maxTokens: options.maxTokens,
          maxRows: options.maxRows,
        })
      )

      if (response?.error) {
        throw new Error(response.error)
      }

      if (response?.messages) {
        set(state => {
          let nextState = state
          for (const raw of response.messages) {
            const normalised = normaliseMessage(raw)
            if (normalised) {
              nextState = appendMessage(nextState, options.sessionId, normalised)
            }
          }
          return {
            ...nextState,
            streamingTurnId: undefined,
            sessions: {
              ...nextState.sessions,
              [options.sessionId]: {
                ...nextState.sessions[options.sessionId],
                status: 'idle',
                lastError: undefined,
              },
            },
          }
        })
      } else {
        set(state => ({
          ...state,
          streamingTurnId: undefined,
          sessions: {
            ...state.sessions,
            [options.sessionId]: {
              ...state.sessions[options.sessionId],
              status: 'idle',
              lastError: undefined,
            },
          },
        }))
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'AI query failed'
      showHybridNotification('AI Query Agent Error', message, true)

      set(state => ({
        ...state,
        streamingTurnId: undefined,
        sessions: {
          ...state.sessions,
          [options.sessionId]: {
            ...state.sessions[options.sessionId],
            status: 'error',
            lastError: message,
          },
        },
      }))
    }
  },

  syncFromMemoryStore: () => {
    const memoryStore = useAIMemoryStore.getState()
    const sessions: Record<string, AgentSession> = {}
    Object.values(memoryStore.sessions).forEach(session => {
      if (session.metadata?.chatType === 'query-agent') {
        sessions[session.id] = {
          id: session.id,
          title: session.title,
          createdAt: session.createdAt,
          updatedAt: session.updatedAt,
          messages: (session.messages || [])
            .filter(msg => msg.role !== 'system') // Filter out system messages
            .map(msg => ({
              id: msg.id,
              agent: typeof msg.metadata?.agent === 'string' ? (msg.metadata.agent as string) : msg.role === 'user' ? 'user' : 'assistant',
              role: msg.role === 'user' ? 'user' : 'assistant', // Ensure only user or assistant
              title: typeof msg.metadata?.title === 'string' ? (msg.metadata.title as string) : undefined,
              content: msg.content,
              createdAt: msg.timestamp,
              attachments: Array.isArray(msg.metadata?.attachments)
                ? (msg.metadata.attachments as AgentAttachment[])
                : undefined,
              provider: typeof msg.metadata?.provider === 'string' ? (msg.metadata.provider as string) : undefined,
              model: typeof msg.metadata?.model === 'string' ? (msg.metadata.model as string) : undefined,
              tokensUsed: typeof msg.metadata?.tokensUsed === 'number' ? (msg.metadata.tokensUsed as number) : undefined,
            })),
          status: 'idle',
        }
      }
    })

    set(state => ({
      ...state,
      sessions: {
        ...sessions,
        ...state.sessions,
      },
      isHydrated: true,
    }))
  },
}),
{
  name: 'ai-query-agent-store',
  partialize: (state) => ({
    sessions: state.sessions,
    activeSessionId: state.activeSessionId,
    isHydrated: state.isHydrated,
  }),
}
))

const hasWailsRuntime =
  typeof window !== 'undefined' &&
  typeof (window as { runtime?: { EventsOnMultiple?: unknown } }).runtime?.EventsOnMultiple === 'function'

if (hasWailsRuntime) {
  EventsOn('ai:query-agent:stream', (payload: unknown) => {
    useAIQueryAgentStore.getState().receiveEvent((payload ?? {}) as StreamPayload)
  })
}
