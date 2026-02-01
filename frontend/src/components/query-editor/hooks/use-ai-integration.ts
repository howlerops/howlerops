import { useCallback, useEffect } from "react"

import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { useAIMemoryStore } from "@/store/ai-memory-store"
import { useAIQueryAgentStore } from "@/store/ai-query-agent-store"
import { useAIConfig, useAIGeneration, useAIStore } from "@/store/ai-store"
import type { DatabaseConnection } from "@/store/connection-store"

import type { QueryMode } from "../types"

interface UseAIIntegrationProps {
  mode: QueryMode
  activeConnection: DatabaseConnection | null
  schema: SchemaNode[]
  environmentFilteredConnections: DatabaseConnection[]
  multiDBSchemas: Map<string, SchemaNode[]>
}

/**
 * Hook that manages AI integration for the query editor
 */
export function useAIIntegration({
  mode,
  activeConnection,
  schema,
  environmentFilteredConnections,
  multiDBSchemas,
}: UseAIIntegrationProps) {
  // AI config and generation
  const { config: aiConfig, isEnabled: aiEnabled } = useAIConfig()
  const {
    generateSQL,
    fixSQL,
    isGenerating,
    lastError,
    suggestions,
    clearSuggestions,
    resetSession,
    hydrateMemoriesFromBackend,
    deleteMemorySession,
    persistMemoriesIfEnabled,
  } = useAIGeneration()

  // Memory store
  const memorySessionsMap = useAIMemoryStore(state => state.sessions)
  const activeMemorySessionId = useAIMemoryStore(state => state.activeSessionId)
  const setActiveMemorySession = useAIMemoryStore(state => state.setActiveSession)
  const startMemorySession = useAIMemoryStore(state => state.startNewSession)
  const renameMemorySession = useAIMemoryStore(state => state.renameSession)
  const clearAllMemorySessions = useAIMemoryStore(state => state.clearAll)

  // Agent store
  const createAgentSession = useAIQueryAgentStore(state => state.createSession)
  const ensureAgentSession = useAIQueryAgentStore(state => state.ensureSession)
  const setActiveAgentSession = useAIQueryAgentStore(state => state.setActiveSession)
  const syncAgentFromMemory = useAIQueryAgentStore(state => state.syncFromMemoryStore)
  const agentHydrated = useAIQueryAgentStore(state => state.isHydrated)
  const renameAgentSession = useAIQueryAgentStore(state => state.renameSession)

  // Memories hydration state
  const memoriesHydrated = useAIStore(state => state.memoriesHydrated)

  // Hydrate memories on mount
  useEffect(() => {
    if (!aiEnabled || !aiConfig.syncMemories) return

    hydrateMemoriesFromBackend().catch(error => {
      console.error('Failed to hydrate AI memories:', error)
    })
  }, [aiEnabled, aiConfig.syncMemories, hydrateMemoriesFromBackend])

  // Sync agent from memory when hydrated
  useEffect(() => {
    if (memoriesHydrated && !agentHydrated) {
      syncAgentFromMemory()
    }
  }, [memoriesHydrated, agentHydrated, syncAgentFromMemory])

  // Fix SQL with AI
  const handleFixQueryError = useCallback(async (error: string, query: string) => {
    if (!aiEnabled) return

    try {
      const schemaDatabase = activeConnection?.database || undefined

      let connections = undefined
      let schemasMap = undefined

      if (mode === 'multi') {
        connections = environmentFilteredConnections
        schemasMap = multiDBSchemas
      } else if (activeConnection && schema) {
        connections = [activeConnection]
        schemasMap = new Map([[activeConnection.id, schema]])
      }

      await fixSQL(query, error, schemaDatabase, mode, connections, schemasMap)
    } catch (error) {
      console.error('Failed to fix SQL:', error)
    }
  }, [aiEnabled, mode, environmentFilteredConnections, multiDBSchemas, activeConnection, schema, fixSQL])

  // Generate SQL from natural language
  const handleGenerateSQL = useCallback(async (naturalLanguagePrompt: string) => {
    if (!naturalLanguagePrompt.trim() || !aiEnabled) return

    try {
      const schemaDatabase = activeConnection?.database || undefined

      let connections = undefined
      let schemasMap = undefined

      if (mode === 'multi') {
        connections = environmentFilteredConnections
        schemasMap = multiDBSchemas
      } else if (activeConnection && schema) {
        connections = [activeConnection]
        schemasMap = new Map([[activeConnection.id, schema]])
      }

      await generateSQL(naturalLanguagePrompt, schemaDatabase, mode, connections, schemasMap)
    } catch (error) {
      console.error('Failed to generate SQL:', error)
      throw error
    }
  }, [aiEnabled, mode, environmentFilteredConnections, multiDBSchemas, activeConnection, schema, generateSQL])

  // Reset AI session
  const handleResetAISession = useCallback(() => {
    resetSession()
    clearSuggestions()
  }, [resetSession, clearSuggestions])

  // Memory session handlers
  const handleCreateMemorySession = useCallback(() => {
    const sessionId = startMemorySession({ title: `Session ${new Date().toLocaleString()}` })
    setActiveMemorySession(sessionId)
    void persistMemoriesIfEnabled()
    return sessionId
  }, [startMemorySession, setActiveMemorySession, persistMemoriesIfEnabled])

  const handleDeleteMemorySession = useCallback(async (sessionId: string) => {
    if (!sessionId) return
    if (!window.confirm('Delete this memory session? This cannot be undone.')) return
    await deleteMemorySession(sessionId)
  }, [deleteMemorySession])

  const handleClearAllMemories = useCallback(() => {
    if (!window.confirm('Clear all AI memory sessions? This cannot be undone.')) return
    clearAllMemorySessions()
    void persistMemoriesIfEnabled()
  }, [clearAllMemorySessions, persistMemoriesIfEnabled])

  const handleResumeMemorySession = useCallback((sessionId: string) => {
    setActiveMemorySession(sessionId)
  }, [setActiveMemorySession])

  const handleRenameMemorySession = useCallback((sessionId: string, title: string) => {
    if (!title.trim()) return
    renameMemorySession(sessionId, title.trim())
    void persistMemoriesIfEnabled()
  }, [renameMemorySession, persistMemoriesIfEnabled])

  return {
    // Config
    aiConfig,
    aiEnabled,

    // Generation state
    isGenerating,
    lastError,
    suggestions,

    // Memory sessions
    memorySessionsMap,
    activeMemorySessionId,
    setActiveMemorySession,

    // Agent sessions
    createAgentSession,
    ensureAgentSession,
    setActiveAgentSession,
    renameAgentSession,

    // Handlers
    handleFixQueryError,
    handleGenerateSQL,
    handleResetAISession,
    handleCreateMemorySession,
    handleDeleteMemorySession,
    handleClearAllMemories,
    handleResumeMemorySession,
    handleRenameMemorySession,
    clearSuggestions,
  }
}
