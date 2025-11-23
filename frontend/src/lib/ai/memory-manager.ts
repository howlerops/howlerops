/**
 * AI Memory Manager
 *
 * Manages AI memory and history operations, including
 * session management and message recording.
 */

import type { AIMemorySession } from '@/store/ai-memory-store'
import { useAIMemoryStore } from '@/store/ai-memory-store'
import type { AISessionId } from '@/types/ai'

/**
 * Ensures an active session exists, creating one if needed
 */
export function ensureActiveSession(options?: {
  title?: string
  metadata?: Record<string, unknown>
}): AISessionId {
  const memoryStore = useAIMemoryStore.getState()

  const sessionId = memoryStore.ensureActiveSession({
    title: options?.title || `Session ${new Date().toLocaleString()}`,
    metadata: options?.metadata,
  })

  return sessionId as AISessionId
}

/**
 * Builds memory context for an AI request
 */
export function buildMemoryContext(options: {
  sessionId: AISessionId
  provider: string
  model: string
  maxTokens: number
}): string | undefined {
  const memoryStore = useAIMemoryStore.getState()

  return memoryStore.buildContext({
    sessionId: options.sessionId,
    provider: options.provider,
    model: options.model,
    maxTokens: options.maxTokens,
  })
}

/**
 * Records a user message in memory
 */
export function recordUserMessage(options: {
  sessionId: AISessionId
  content: string
  metadata?: Record<string, unknown>
}): void {
  const memoryStore = useAIMemoryStore.getState()

  memoryStore.recordMessage({
    sessionId: options.sessionId,
    role: 'user',
    content: options.content,
    metadata: options.metadata,
  })
}

/**
 * Records an assistant message in memory
 */
export function recordAssistantMessage(options: {
  sessionId: AISessionId
  content: string
  metadata?: Record<string, unknown>
}): void {
  const memoryStore = useAIMemoryStore.getState()

  memoryStore.recordMessage({
    sessionId: options.sessionId,
    role: 'assistant',
    content: options.content,
    metadata: options.metadata,
  })
}

/**
 * Ensures the active session is appropriate for the given chat type
 * Creates a new session if the current one doesn't match
 */
export function ensureSessionForChatType(options: {
  chatType: string
  title: string
}): AISessionId {
  const memoryStore = useAIMemoryStore.getState()
  const activeSessionId = memoryStore.activeSessionId

  // Check if active session matches chat type
  if (activeSessionId) {
    const activeSession = memoryStore.sessions[activeSessionId]
    if (activeSession?.metadata?.chatType === options.chatType) {
      return activeSessionId as AISessionId
    }
  }

  // Create new session for this chat type
  const sessionId = memoryStore.startNewSession({
    title: options.title,
    metadata: { chatType: options.chatType },
  })

  memoryStore.setActiveSession(sessionId)
  return sessionId as AISessionId
}

/**
 * Exports all sessions for persistence
 */
export function exportSessions(): AIMemorySession[] {
  const memoryStore = useAIMemoryStore.getState()
  return memoryStore.exportSessions()
}

/**
 * Imports sessions from persistence
 */
export function importSessions(sessions: AIMemorySession[], options?: { merge?: boolean }): void {
  const memoryStore = useAIMemoryStore.getState()
  memoryStore.importSessions(sessions, options)
}
