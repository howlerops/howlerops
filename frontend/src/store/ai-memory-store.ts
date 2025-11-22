import { create } from 'zustand'
import { createJSONStorage,persist } from 'zustand/middleware'

type AIMemoryRole = 'system' | 'user' | 'assistant'

export interface AIMemoryMessage {
  id: string
  role: AIMemoryRole
  content: string
  tokens: number
  timestamp: number
  metadata?: Record<string, unknown>
}

export interface AIMemorySession {
  id: string
  title: string
  createdAt: number
  updatedAt: number
  messages: AIMemoryMessage[]
  summary?: string
  summaryTokens?: number
  metadata?: Record<string, unknown>
}

interface BuildContextOptions {
  sessionId?: string
  provider?: string
  model?: string
  maxTokens?: number
  includeSummary?: boolean
}

interface EnsureSessionOptions {
  title?: string
  metadata?: Record<string, unknown>
}

interface AIMemoryState {
  sessions: Record<string, AIMemorySession>
  activeSessionId: string | null
  maxContextTokens: number
}

interface AIMemoryActions {
  startNewSession: (options?: EnsureSessionOptions) => string
  ensureActiveSession: (options?: EnsureSessionOptions) => string
  setActiveSession: (sessionId: string) => void
  recordMessage: (input: {
    sessionId: string
    role: AIMemoryRole
    content: string
    metadata?: Record<string, unknown>
    timestamp?: number
  }) => void
  buildContext: (options: BuildContextOptions) => string
  resetActiveSession: () => void
  pruneSessions: (limit: number) => void
  clearAll: () => void
  exportSessions: () => AIMemorySession[]
  importSessions: (sessions: AIMemorySession[], options?: { merge?: boolean }) => void
  deleteSession: (sessionId: string) => void
  renameSession: (sessionId: string, title: string) => void
}

type AIMemoryStore = AIMemoryState & AIMemoryActions

const DEFAULT_MAX_CONTEXT_TOKENS = 2000

export function estimateTokens(text: string): number {
  if (!text) return 0
  // Simple heuristic: 1 token ≈ 4 characters (English)
  return Math.ceil(text.length / 4)
}

function createMessageId(): string {
  return crypto.randomUUID()
}

function createSessionId(): string {
  return crypto.randomUUID()
}

function cloneSession(session: AIMemorySession): AIMemorySession {
  return {
    ...session,
    metadata: session.metadata ? { ...session.metadata } : undefined,
    messages: session.messages.map(message => ({
      ...message,
      metadata: message.metadata ? { ...message.metadata } : undefined,
    })),
  }
}

function summariseMessages(messages: AIMemoryMessage[]): { summary: string; tokens: number } {
  if (messages.length === 0) {
    return { summary: '', tokens: 0 }
  }

  const MAX_SNIPPET = 200

  const first = messages[0]
  const lastUser = [...messages].reverse().find(msg => msg.role === 'user')
  const lastAssistant = [...messages].reverse().find(msg => msg.role === 'assistant')

  const summaryParts: string[] = []
  summaryParts.push(`Conversation kicked off with: ${first.content.slice(0, MAX_SNIPPET)}${first.content.length > MAX_SNIPPET ? '…' : ''}`)

  if (lastUser) {
    summaryParts.push(`Latest user intent: ${lastUser.content.slice(0, MAX_SNIPPET)}${lastUser.content.length > MAX_SNIPPET ? '…' : ''}`)
  }

  if (lastAssistant) {
    summaryParts.push(`Assistant guidance: ${lastAssistant.content.slice(0, MAX_SNIPPET)}${lastAssistant.content.length > MAX_SNIPPET ? '…' : ''}`)
  }

  const summary = summaryParts.join('\n')
  return {
    summary,
    tokens: estimateTokens(summary),
  }
}

function maybeSummarizeSession(initialSession: AIMemorySession, maxTokens: number): AIMemorySession {
  let session = initialSession
  let totalTokens = session.messages.reduce((sum, msg) => sum + msg.tokens, 0)

  while (totalTokens > maxTokens && session.messages.length > 4) {
    const retainCount = Math.min(8, Math.max(4, session.messages.length - 4))
    const retainedMessages = session.messages.slice(-retainCount)
    const archivedMessages = session.messages.slice(0, session.messages.length - retainedMessages.length)

    const { summary, tokens } = summariseMessages(archivedMessages)
    const mergedSummary = [session.summary, summary].filter(Boolean).join('\n')

    session = {
      ...session,
      summary: mergedSummary,
      summaryTokens: (session.summaryTokens || 0) + tokens,
      messages: retainedMessages,
      updatedAt: Date.now(),
    }

    totalTokens = session.messages.reduce((sum, msg) => sum + msg.tokens, 0)
  }

  return session
}

export const useAIMemoryStore = create<AIMemoryStore>()(
  persist(
    (set, get) => ({
      sessions: {},
      activeSessionId: null,
      maxContextTokens: DEFAULT_MAX_CONTEXT_TOKENS,

      startNewSession: (options?: EnsureSessionOptions) => {
        const id = createSessionId()
        const now = Date.now()
        const title = options?.title || `Session ${new Date(now).toLocaleString()}`

        const session: AIMemorySession = {
          id,
          title,
          createdAt: now,
          updatedAt: now,
          messages: [],
          metadata: options?.metadata,
        }

        set(state => ({
          sessions: {
            ...state.sessions,
            [id]: session,
          },
          activeSessionId: id,
        }))

        get().pruneSessions(10)

        return id
      },

      ensureActiveSession: (options?: EnsureSessionOptions) => {
        const { activeSessionId, sessions, startNewSession } = get()
        if (activeSessionId && sessions[activeSessionId]) {
          return activeSessionId
        }
        return startNewSession(options)
      },

      setActiveSession: (sessionId: string) => {
        set(state => ({
          activeSessionId: state.sessions[sessionId] ? sessionId : state.activeSessionId,
        }))
      },

      recordMessage: ({ sessionId, role, content, metadata, timestamp }: {
        sessionId: string
        role: AIMemoryRole
        content: string
        metadata?: Record<string, unknown>
        timestamp?: number
      }) => {
        set(state => {
          const session = state.sessions[sessionId]
          if (!session) {
            return state
          }

          const message: AIMemoryMessage = {
            id: createMessageId(),
            role,
            content,
            tokens: estimateTokens(content),
            metadata,
            timestamp: timestamp ?? Date.now(),
          }

          const withMessage: AIMemorySession = {
            ...session,
            messages: [...session.messages, message],
            updatedAt: Date.now(),
          }

          const normalized = maybeSummarizeSession(withMessage, state.maxContextTokens)

          return {
            ...state,
            sessions: {
              ...state.sessions,
              [sessionId]: normalized,
            },
          }
        })
      },

      buildContext: ({ sessionId, provider, model, maxTokens, includeSummary = true }: BuildContextOptions) => {
        const { sessions, maxContextTokens } = get()
        const id = sessionId || get().activeSessionId
        if (!id) return ''

        const session = sessions[id]
        if (!session) return ''

        const tokenBudget = Math.max(200, Math.min(maxTokens ?? maxContextTokens, maxContextTokens))
        const selected: AIMemoryMessage[] = []
        let tokensUsed = 0

        // Walk messages from end (latest) backwards until we fill the budget
        for (let i = session.messages.length - 1; i >= 0; i--) {
          const message = session.messages[i]
          const nextTotal = tokensUsed + message.tokens
          if (selected.length > 0 && nextTotal > tokenBudget) {
            break
          }
          selected.push(message)
          tokensUsed = nextTotal
        }

        selected.reverse()

        const lines: string[] = []
        lines.push(`Conversation context (provider: ${provider || 'default'}, model: ${model || 'n/a'})`)
        lines.push(`Messages included: ${selected.length}, tokens ~${tokensUsed}/${tokenBudget}`)

        if (includeSummary && session.summary) {
          lines.push('')
          lines.push(`Session summary: ${session.summary}`)
        }

        if (session.messages.length > selected.length) {
          lines.push('')
          lines.push(`Note: ${session.messages.length - selected.length} earlier messages omitted to respect token budget.`)
        }

        selected.forEach(message => {
          lines.push('')
          lines.push(`[${message.role.toUpperCase()} @ ${new Date(message.timestamp).toLocaleTimeString()}]`)
          lines.push(message.content)
        })

        return lines.join('\n')
      },

      resetActiveSession: () => {
        const { activeSessionId, sessions } = get()
        if (!activeSessionId || !sessions[activeSessionId]) {
          return
        }

        set(state => ({
          sessions: {
            ...state.sessions,
            [activeSessionId]: {
              ...state.sessions[activeSessionId],
              messages: [],
              updatedAt: Date.now(),
            },
          },
        }))
      },

      pruneSessions: (limit: number) => {
        set(state => {
          const entries = Object.values(state.sessions)
            .sort((a, b) => b.updatedAt - a.updatedAt)
          if (entries.length <= limit) {
            return state
          }

          const keep = new Set(entries.slice(0, limit).map(session => session.id))
          const pruned: Record<string, AIMemorySession> = {}
          entries.forEach(session => {
            if (keep.has(session.id)) {
              pruned[session.id] = session
            }
          })

          const activeSessionId = state.activeSessionId && keep.has(state.activeSessionId)
            ? state.activeSessionId
            : entries[0]?.id ?? null

          return {
            ...state,
            sessions: pruned,
            activeSessionId,
          }
        })
      },

      clearAll: () => {
        set({
          sessions: {},
          activeSessionId: null,
        })
      },

      exportSessions: () => {
        const { sessions } = get()
        return Object.values(sessions).map(cloneSession)
      },

      importSessions: (sessions, options) => {
        const merge = options?.merge ?? true
        set(state => {
          const nextSessions = merge ? { ...state.sessions } : {}
          sessions.forEach(session => {
            nextSessions[session.id] = cloneSession(session)
          })

          const activeSessionId = nextSessions[state.activeSessionId ?? ''] ? state.activeSessionId : sessions[0]?.id ?? state.activeSessionId

          return {
            ...state,
            sessions: nextSessions,
            activeSessionId,
          }
        })
      },

      deleteSession: (sessionId: string) => {
        if (!sessionId) return

        set(state => {
          if (!state.sessions[sessionId]) {
            return state
          }

          const sessions = { ...state.sessions }
          delete sessions[sessionId]

          let activeID = state.activeSessionId
          if (activeID === sessionId) {
            activeID = Object.keys(sessions)[0] ?? null
          }

          return {
            ...state,
            sessions,
            activeSessionId: activeID,
          }
        })
      },

      renameSession: (sessionId: string, title: string) => {
        set(state => {
          const session = state.sessions[sessionId]
          if (!session) {
            return state
          }

          const updated = cloneSession(session)
          updated.title = title
          updated.updatedAt = Date.now()

          return {
            ...state,
            sessions: {
              ...state.sessions,
              [sessionId]: updated,
            },
          }
        })
      },
    }),
    {
      name: 'ai-memory-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        activeSessionId: state.activeSessionId,
        maxContextTokens: state.maxContextTokens,
      }),
      version: 1,
      migrate: (persistedState: unknown, version: number) => {
        if (version !== 0 || typeof persistedState !== 'object' || !persistedState) {
          return persistedState
        }

        const record = persistedState as { state?: Partial<AIMemoryStore> }
        if (!record.state || !record.state.sessions) {
          return persistedState
        }

        return {
          ...record,
          state: {
            ...record.state,
            sessions: {},
          },
        }
      },
    },
  ),
)
