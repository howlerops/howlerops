import { create } from 'zustand'
import { createJSONStorage,persist } from 'zustand/middleware'

import type { SchemaNode } from '@/hooks/use-schema-introspection'
import {
  buildFixSQLBackendRequest,
  buildGenerateSQLBackendRequest,
  buildGenericMessageBackendRequest,
  buildMemoryContext,
  ensureActiveSession,
  getRecallContext,
  normalizeError,
  parseFixSQLResponse,
  parseGenerateSQLResponse,
  parseGenericMessageResponse,
  recordAssistantMessage,
  recordUserMessage,
  validateAIEnabled,
  validateFixSQLRequest,
  validateGenerateSQLRequest,
  validateGenericMessageRequest,
} from '@/lib/ai'
import { handleVoidPromise } from '@/lib/ai-error-handling'
import { detectsMultiDB } from '@/lib/ai-schema-context-builder'
import { showHybridNotification,testAIProviderConnection } from '@/lib/wails-ai-api'
import { type AIMemorySession as MemorySession,estimateTokens as estimateMemoryTokens, useAIMemoryStore } from '@/store/ai-memory-store'
import type { DatabaseConnection } from '@/store/connection-store'
import type { AISessionId } from '@/types/ai'

import { ConfigureAIProvider,DeleteAIMemorySession, LoadAIMemorySessions, SaveAIMemorySessions } from '../../wailsjs/go/main/App'
import { main as wailsModels } from '../../wailsjs/go/models'

// Normalize endpoint URL
function normalizeEndpoint(endpoint: string | undefined): string {
  if (!endpoint) return 'http://localhost:11434'
  return endpoint.replace(/\/+$/, '') // Remove trailing slashes
}

function buildProviderConfig(provider: string, config: AIConfig) {
  const payload: {
    provider: string
    apiKey?: string
    endpoint?: string
    model?: string
    options?: Record<string, string>
  } = {
    provider,
    model: config.selectedModel || undefined,
  }

  switch (provider) {
    case 'openai':
      payload.apiKey = config.openaiApiKey
      break

    case 'anthropic':
      payload.apiKey = config.anthropicApiKey
      break

    case 'ollama':
      payload.endpoint = normalizeEndpoint(config.ollamaEndpoint)
      break

    case 'huggingface':
      payload.endpoint = normalizeEndpoint(config.huggingfaceEndpoint || config.ollamaEndpoint)
      break

    case 'claudecode': {
      const binaryPath = config.claudeCodePath || 'claude'
      payload.model = config.selectedModel || 'opus'
      payload.options = { binary_path: binaryPath }
      break
    }

    case 'codex': {
      payload.apiKey = config.codexApiKey
      payload.model = config.selectedModel || 'code-davinci-002'
      if (config.codexOrganization) {
        payload.options = { organization: config.codexOrganization }
      }
      break
    }
  }

  return payload
}

export interface AIConfig {
  enabled: boolean
  provider: 'openai' | 'anthropic' | 'ollama' | 'huggingface' | 'claudecode' | 'codex'
  openaiApiKey: string
  anthropicApiKey: string
  claudeCodePath: string  // Path to Claude CLI executable
  codexApiKey: string     // OpenAI Codex API key
  codexOrganization: string // OpenAI organization ID for Codex
  ollamaEndpoint: string
  huggingfaceEndpoint: string
  selectedModel: string
  maxTokens: number
  temperature: number
  autoFixEnabled: boolean
  suggestionThreshold: number
  syncMemories: boolean
}

export interface SQLSuggestion {
  id: string
  query: string
  explanation: string
  confidence: number
  timestamp: Date
  provider: string
  model: string
}

export interface AIState {
  config: AIConfig
  isGenerating: boolean
  suggestions: SQLSuggestion[]
  lastError: string | null
  connectionStatus: {
    openai: 'connected' | 'disconnected' | 'testing' | 'error'
    anthropic: 'connected' | 'disconnected' | 'testing' | 'error'
    ollama: 'connected' | 'disconnected' | 'testing' | 'error'
    huggingface: 'connected' | 'disconnected' | 'testing' | 'error'
    claudecode: 'connected' | 'disconnected' | 'testing' | 'error'
    codex: 'connected' | 'disconnected' | 'testing' | 'error'
  }
  memoriesHydrated: boolean
  providerSynced: boolean
}

export interface AIActions {
  updateConfig: (config: Partial<AIConfig>) => void
  setIsGenerating: (generating: boolean) => void
  addSuggestion: (suggestion: Omit<SQLSuggestion, 'id' | 'timestamp'>) => void
  clearSuggestions: () => void
  setLastError: (error: string | null) => void
  setConnectionStatus: (provider: string, status: string) => void
  testConnection: (provider: string) => Promise<boolean>
  ensureProviderConfigured: () => Promise<void>
  generateSQL: (prompt: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>) => Promise<string>
  fixSQL: (query: string, error: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>) => Promise<string>
  sendGenericMessage: (prompt: string, options?: {
    context?: string
    systemPrompt?: string
    metadata?: Record<string, string>
  }) => Promise<{
    content: string
    provider: string
    model: string
    tokensUsed: number
    metadata?: Record<string, string>
  }>
  resetConfig: () => void
  resetSession: () => void
  hydrateMemoriesFromBackend: () => Promise<void>
  persistMemoriesIfEnabled: () => Promise<void>
  deleteMemorySession: (sessionId: string) => Promise<void>
}

const defaultConfig: AIConfig = {
  enabled: false,
  provider: 'openai',
  openaiApiKey: '',
  anthropicApiKey: '',
  claudeCodePath: '',
  codexApiKey: '',
  codexOrganization: '',
  ollamaEndpoint: 'http://localhost:11434',
  huggingfaceEndpoint: 'http://localhost:11434',
  selectedModel: 'gpt-4o-mini',
  maxTokens: 2048,
  temperature: 0.1,
  autoFixEnabled: true,
  suggestionThreshold: 0.7,
  syncMemories: true,
}

const defaultState: AIState = {
  config: defaultConfig,
  isGenerating: false,
  suggestions: [],
  lastError: null,
  connectionStatus: {
    openai: 'disconnected',
    anthropic: 'disconnected',
    ollama: 'disconnected',
    huggingface: 'disconnected',
    claudecode: 'disconnected',
    codex: 'disconnected',
  },
  memoriesHydrated: false,
  providerSynced: false,
}

type WailsMemorySession = InstanceType<typeof wailsModels.AIMemorySessionPayload>
type WailsMemoryMessage = InstanceType<typeof wailsModels.AIMemoryMessagePayload>

const serializeMemorySessions = (sessions: MemorySession[]): WailsMemorySession[] => {
  return sessions.map(session => wailsModels.AIMemorySessionPayload.createFrom({
    id: session.id,
    title: session.title,
    createdAt: session.createdAt,
    updatedAt: session.updatedAt,
    summary: session.summary ?? '',
    summaryTokens: session.summaryTokens ?? 0,
    metadata: session.metadata ?? {},
    messages: session.messages.map(message => wailsModels.AIMemoryMessagePayload.createFrom({
      role: message.role,
      content: message.content,
      timestamp: message.timestamp,
      metadata: message.metadata ?? {},
    })),
  }))
}

const deserializeMemorySessions = (payload: WailsMemorySession[]): MemorySession[] => {
  return payload.map(session => ({
    id: session.id,
    title: session.title,
    createdAt: session.createdAt,
    updatedAt: session.updatedAt,
    summary: session.summary || undefined,
    summaryTokens: session.summaryTokens || undefined,
    metadata: session.metadata && Object.keys(session.metadata).length ? session.metadata : undefined,
    messages: (session.messages || []).map((message: WailsMemoryMessage) => ({
      id: crypto.randomUUID(),
      role: (message.role as 'system' | 'user' | 'assistant') ?? 'user',
      content: message.content,
      tokens: estimateMemoryTokens(message.content),
      timestamp: message.timestamp,
      metadata: message.metadata && Object.keys(message.metadata).length ? message.metadata : undefined,
    })),
  }))
}

// Secure storage for API keys - Synchronous version for Zustand
class SecureStorage {
  private static instance: SecureStorage
  private keyPrefix = 'ai-secure-'
  private fallback = new Map<string, string>()

  static getInstance(): SecureStorage {
    if (!SecureStorage.instance) {
      SecureStorage.instance = new SecureStorage()
    }
    return SecureStorage.instance
  }

  setItem(key: string, value: string): void {
    try {
      sessionStorage.setItem(this.keyPrefix + key, value)
      this.fallback.delete(this.keyPrefix + key)
    } catch (error) {
      this.fallback.set(this.keyPrefix + key, value)
      if (error instanceof DOMException && error.name === 'QuotaExceededError') {
        console.warn('Session storage quota exceeded; falling back to in-memory secure storage')
        return
      }
      console.error('Failed to store secure item:', error)
    }
  }

  getItem(key: string): string | null {
    try {
      const value = sessionStorage.getItem(this.keyPrefix + key)
      if (value !== null) {
        return value
      }
    } catch (error) {
      console.error('Failed to retrieve secure item:', error)
    }
    return this.fallback.get(this.keyPrefix + key) ?? null
  }

  removeItem(key: string): void {
    try {
      sessionStorage.removeItem(this.keyPrefix + key)
    } catch (error) {
      console.error('Failed to remove secure item:', error)
    }
    this.fallback.delete(this.keyPrefix + key)
  }

  clear(): void {
    try {
      const keys = Object.keys(sessionStorage).filter(key => key.startsWith(this.keyPrefix))
      keys.forEach(key => sessionStorage.removeItem(key))
    } catch (error) {
      console.error('Failed to clear secure storage:', error)
    }
    this.fallback.clear()
  }

  // Helper methods for API key management
  storeApiKey(provider: string, apiKey: string): void {
    this.setItem(`${provider}-api-key`, apiKey)
  }

  getApiKey(provider: string): string | null {
    return this.getItem(`${provider}-api-key`)
  }

  removeApiKey(provider: string): void {
    this.removeItem(`${provider}-api-key`)
  }
}

// Custom storage for sensitive data - SYNCHRONOUS to match localStorage behavior
const createSecureStorage = () => {
  const storage = SecureStorage.getInstance()

  return {
    getItem: (name: string): string | null => {
      const value = localStorage.getItem(name)
      if (!value) return null

      try {
        const parsed = JSON.parse(value)

        if (parsed.state?.config) {
          const config = parsed.state.config
          if (config.openaiApiKey === 'STORED_SECURELY') {
            config.openaiApiKey = storage.getItem('openai-api-key') || ''
          }
          if (config.anthropicApiKey === 'STORED_SECURELY') {
            config.anthropicApiKey = storage.getItem('anthropic-api-key') || ''
          }
          if (config.codexApiKey === 'STORED_SECURELY') {
            config.codexApiKey = storage.getItem('codex-api-key') || ''
          }
        }

        return JSON.stringify(parsed)
      } catch {
        return value
      }
    },

    setItem: (name: string, value: string): void => {
      try {
        const parsed = JSON.parse(value)

        if (parsed.state?.config) {
          const config = parsed.state.config

          if (config.openaiApiKey && config.openaiApiKey !== 'STORED_SECURELY') {
            storage.setItem('openai-api-key', config.openaiApiKey)
            config.openaiApiKey = 'STORED_SECURELY'
          }

          if (config.anthropicApiKey && config.anthropicApiKey !== 'STORED_SECURELY') {
            storage.setItem('anthropic-api-key', config.anthropicApiKey)
            config.anthropicApiKey = 'STORED_SECURELY'
          }

          if (config.codexApiKey && config.codexApiKey !== 'STORED_SECURELY') {
            storage.setItem('codex-api-key', config.codexApiKey)
            config.codexApiKey = 'STORED_SECURELY'
          }
        }

        localStorage.setItem(name, JSON.stringify(parsed))
      } catch {
        localStorage.setItem(name, value)
      }
    },

    removeItem: (name: string): void => {
      localStorage.removeItem(name)
    },
  }
}

export const useAIStore = create<AIState & AIActions>()(
  persist(
    (set, get) => ({
      ...defaultState,

      updateConfig: (configUpdate: Partial<AIConfig>) => {
        const previousConfig = get().config
        set(state => ({
          config: { ...state.config, ...configUpdate },
          providerSynced: false,
        }))

        if (configUpdate.syncMemories !== undefined && configUpdate.syncMemories !== previousConfig.syncMemories) {
          if (configUpdate.syncMemories) {
            // Use handleVoidPromise to properly handle async operations
            handleVoidPromise(
              get().hydrateMemoriesFromBackend()
                .then(() => get().persistMemoriesIfEnabled()),
              'updateConfig: syncMemories enabled'
            )
          } else {
            set({ memoriesHydrated: false })
          }
        }
      },

      setIsGenerating: (generating: boolean) => {
        set({ isGenerating: generating })
      },

      addSuggestion: (suggestion: Omit<SQLSuggestion, 'id' | 'timestamp'>) => {
        const newSuggestion: SQLSuggestion = {
          ...suggestion,
          id: crypto.randomUUID(),
          timestamp: new Date(),
        }

        set(state => ({
          suggestions: [newSuggestion, ...state.suggestions].slice(0, 10),
        }))
      },

      clearSuggestions: () => {
        set({ suggestions: [] })
      },

      setLastError: (error: string | null) => {
        set({ lastError: error })
      },

      setConnectionStatus: (provider: string, status: string) => {
        set(state => ({
          connectionStatus: {
            ...state.connectionStatus,
            [provider]: status as 'connected' | 'disconnected' | 'testing' | 'error',
          },
        }))
      },

      testConnection: async (provider: string): Promise<boolean> => {
        const { config, setConnectionStatus } = get()

        setConnectionStatus(provider, 'testing')

        try {
          // Prepare parameters for the AI test
          const testParams = {
            provider,
            apiKey: '',
            model: config.selectedModel,
            endpoint: '',
            organization: '',
            binaryPath: '',
          }

          // Set provider-specific parameters
          switch (provider) {
            case 'openai':
              if (!config.openaiApiKey) {
                throw new Error('OpenAI API key not configured')
              }
              testParams.apiKey = config.openaiApiKey
              break

            case 'anthropic':
              if (!config.anthropicApiKey) {
                throw new Error('Anthropic API key not configured')
              }
              testParams.apiKey = config.anthropicApiKey
              break

            case 'ollama':
              if (!config.selectedModel) {
                throw new Error('Select a model before testing the Ollama connection.')
              }
              testParams.endpoint = normalizeEndpoint(config.ollamaEndpoint)
              break

            case 'claudecode':
              testParams.binaryPath = config.claudeCodePath || 'claude'
              testParams.model = config.selectedModel || 'opus'
              break

            case 'codex':
              if (!config.codexApiKey) {
                throw new Error('Codex API key not configured')
              }
              testParams.apiKey = config.codexApiKey
              testParams.organization = config.codexOrganization
              break

            case 'huggingface':
              if (!config.selectedModel) {
                throw new Error('Select a model before testing the Hugging Face connection.')
              }
              testParams.endpoint = normalizeEndpoint(config.huggingfaceEndpoint || config.ollamaEndpoint)
              break

            default:
              throw new Error('Unknown provider')
          }

          // Call Wails AI test method
          const response = await testAIProviderConnection(testParams)

          if (response.success) {
            const providerConfig = buildProviderConfig(provider, config)
            await ConfigureAIProvider(providerConfig)
            set({ providerSynced: true })

            setConnectionStatus(provider, 'connected')

            // Show success notification using hybrid approach (toast + optional dialog)
            const providerName = provider === 'claudecode' ? 'Claude Code' :
                               provider === 'codex' ? 'Codex' :
                               provider === 'huggingface' ? 'Hugging Face' :
                               provider === 'ollama' ? 'Ollama' :
                               provider === 'openai' ? 'OpenAI' :
                               provider === 'anthropic' ? 'Anthropic' : provider

            await showHybridNotification('Connection Test', `${providerName} connection successful!`, false, false)
            return true
          } else {
            throw new Error(response.error || 'Connection test failed')
          }
        } catch (error) {
          console.error(`${provider} connection test failed:`, error)
          setConnectionStatus(provider, 'error')
          set({ providerSynced: false })

          // Show error notification using hybrid approach (toast + dialog for errors)
          const errorMessage = error instanceof Error ? error.message : 'Connection test failed'
          await showHybridNotification('Connection Test Failed', errorMessage, true, true)

          throw error // Re-throw so the frontend can handle it
        }
      },

      ensureProviderConfigured: async () => {
        const { config, providerSynced } = get()
        if (providerSynced || !config.enabled) {
          return
        }

        const providerConfig = buildProviderConfig(config.provider, config)

        // Skip if provider requires credentials that are missing
        if ((config.provider === 'openai' && !providerConfig.apiKey) ||
            (config.provider === 'anthropic' && !providerConfig.apiKey) ||
            (config.provider === 'codex' && !providerConfig.apiKey)) {
          return
        }

        try {
          await ConfigureAIProvider(providerConfig)
          set({ providerSynced: true })
        } catch (error) {
          console.error('Failed to configure AI provider', error)
          set({ providerSynced: false })
          throw error
        }
      },

      /**
       * Generates SQL from natural language prompt
       *
       * This is the main entry point for AI-powered SQL generation.
       * It validates the request, builds context, calls the AI provider,
       * and records the interaction in memory.
       *
       * @param prompt - Natural language description of desired query
       * @param schema - Optional schema context string
       * @param mode - Query mode (single/multi database)
       * @param connections - Active database connections
       * @param schemasMap - Schema information for each connection
       * @returns Generated SQL query string
       * @throws {AIError} If validation fails or AI request fails
       */
      generateSQL: async (prompt: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()

        // Determine effective mode: auto-detect or use provided mode
        const wantsMultiDB = detectsMultiDB(prompt)
        const effectiveMode = wantsMultiDB ? 'multi' : (mode || 'single')

        console.log(`🤖 AI Query Mode Detection:`, {
          providedMode: mode,
          detectedMultiDB: wantsMultiDB,
          effectiveMode,
          hasConnections: !!connections && connections.length > 0
        })

        setIsGenerating(true)
        setLastError(null)

        try {
          // Validate configuration and request
          validateAIEnabled(config)
          validateGenerateSQLRequest({ prompt, mode: effectiveMode, connections, schema, schemasMap })
          await get().ensureProviderConfigured()

          // Set up session and memory
          const sessionId = ensureActiveSession({
            title: `Session ${new Date().toLocaleString()}`,
          }) as AISessionId

          const provider = (config.provider || 'openai').toLowerCase()

          // Record user message
          recordUserMessage({
            sessionId,
            content: prompt,
            metadata: {
              mode: effectiveMode,
              provider,
            },
          })

          // Build memory context
          const memoryContext = buildMemoryContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          // Get recall context if enabled
          const recallContext = config.syncMemories
            ? await getRecallContext(prompt, 5)
            : undefined

          // Build backend request
          const request = buildGenerateSQLBackendRequest(
            {
              prompt,
              schema,
              mode: effectiveMode,
              connections,
              schemasMap,
              sessionId,
              memoryContext,
            },
            config
          )

          // Add recall context to request if available
          if (recallContext) {
            const { addRecallContext } = await import('@/lib/ai-schema-context-builder')
            request.context = addRecallContext(request.context, recallContext)
          }

          // Import and execute Wails binding
          const { GenerateSQLFromNaturalLanguage } = await import('../../wailsjs/go/main/App')
          const rawResult = await GenerateSQLFromNaturalLanguage(request)

          // Parse and validate response
          const result = parseGenerateSQLResponse(rawResult)

          // Record results in suggestions
          addSuggestion({
            query: result.sql,
            explanation: result.explanation || '',
            confidence: result.confidence ?? 0.8,
            provider: config.provider,
            model: config.selectedModel,
          })

          // Record assistant response in memory
          recordAssistantMessage({
            sessionId,
            content: result.sql,
            metadata: {
              type: 'generation',
              explanation: result.explanation,
            },
          })

          handleVoidPromise(get().persistMemoriesIfEnabled(), 'generateSQL: success')
          return result.sql
        } catch (error) {
          const normalizedError = normalizeError(error)
          setLastError(normalizedError.message)
          handleVoidPromise(get().persistMemoriesIfEnabled(), 'generateSQL: error')
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      /**
       * Fixes SQL query based on error message
       *
       * Takes a failed query and error message, then uses AI to generate
       * a corrected version of the query.
       *
       * @param query - The SQL query that failed
       * @param error - Error message from database
       * @param schema - Optional schema context string
       * @param mode - Query mode (single/multi database)
       * @param connections - Active database connections
       * @param schemasMap - Schema information for each connection
       * @returns Fixed SQL query string
       * @throws {AIError} If validation fails or AI request fails
       */
      fixSQL: async (query: string, error: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()

        setIsGenerating(true)
        setLastError(null)

        try {
          // Validate configuration and request
          validateAIEnabled(config)
          validateFixSQLRequest({ query, error, mode, connections, schema, schemasMap })
          await get().ensureProviderConfigured()

          // Set up session and memory
          const sessionId = ensureActiveSession({
            title: `Session ${new Date().toLocaleString()}`,
          }) as AISessionId

          const provider = (config.provider || 'openai').toLowerCase()

          // Record user message
          recordUserMessage({
            sessionId,
            content: `Fix query:\n${query}\n\nError:\n${error}`,
            metadata: {
              type: 'fix-request',
              provider,
            },
          })

          // Build memory context
          const memoryContext = buildMemoryContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          // Build backend request
          const request = buildFixSQLBackendRequest(
            {
              query,
              error,
              schema,
              mode: mode || 'single',
              connections,
              schemasMap,
              sessionId,
              memoryContext,
            },
            config
          )

          // Import and execute Wails binding
          const { FixSQLErrorWithOptions } = await import('../../wailsjs/go/main/App')
          const rawResult = await FixSQLErrorWithOptions(request)

          // Parse and validate response
          const result = parseFixSQLResponse(rawResult)

          // Record results in suggestions
          addSuggestion({
            query: result.sql,
            explanation: result.explanation ?? 'Fixed SQL query',
            confidence: result.confidence ?? 0.9,
            provider: config.provider,
            model: config.selectedModel,
          })

          // Record assistant response in memory
          recordAssistantMessage({
            sessionId,
            content: result.sql,
            metadata: {
              type: 'fix-response',
              explanation: result.explanation,
            },
          })

          handleVoidPromise(get().persistMemoriesIfEnabled(), 'fixSQL: success')
          return result.sql
        } catch (error) {
          const normalizedError = normalizeError(error)
          setLastError(normalizedError.message)
          handleVoidPromise(get().persistMemoriesIfEnabled(), 'fixSQL: error')
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      /**
       * Sends a generic message to the AI (non-SQL use case)
       *
       * Used for general chat or non-database-specific AI interactions.
       * Automatically manages chat-specific session context.
       *
       * @param prompt - User message/question
       * @param options - Additional options (context, system prompt, metadata)
       * @returns AI response with metadata
       * @throws {AIError} If validation fails or AI request fails
       */
      sendGenericMessage: async (prompt: string, options?: {
        context?: string
        systemPrompt?: string
        metadata?: Record<string, string>
      }) => {
        const { config, setIsGenerating, setLastError } = get()

        setIsGenerating(true)
        setLastError(null)

        try {
          // Validate configuration and request
          validateAIEnabled(config)
          validateGenericMessageRequest({ prompt, ...options })
          await get().ensureProviderConfigured()

          // Ensure appropriate session for generic chat
          const { ensureSessionForChatType } = await import('@/lib/ai')
          const sessionId = ensureSessionForChatType({
            chatType: 'generic',
            title: `Chat Session ${new Date().toLocaleString()}`,
          }) as AISessionId

          const provider = (config.provider || 'openai').toLowerCase()

          // Record user message
          recordUserMessage({
            sessionId,
            content: prompt,
            metadata: {
              type: 'chat:user',
              provider,
            },
          })

          // Build memory context
          const memoryContext = buildMemoryContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          // Build backend request
          const request = buildGenericMessageBackendRequest(
            {
              prompt,
              context: options?.context,
              systemPrompt: options?.systemPrompt,
              metadata: {
                sessionId,
                chatType: 'generic',
                ...options?.metadata,
              },
              sessionId,
              memoryContext,
            },
            config
          )

          // Import and execute Wails binding
          const { GenericChat } = await import('../../wailsjs/go/main/App')
          const rawResponse = await GenericChat(request)

          // Parse and validate response
          const response = parseGenericMessageResponse(rawResponse, provider, config.selectedModel)

          // Record assistant response in memory
          recordAssistantMessage({
            sessionId,
            content: response.content,
            metadata: {
              type: 'chat:assistant',
              provider: response.provider,
              tokensUsed: response.tokensUsed,
            },
          })

          return response
        } catch (error) {
          const normalizedError = normalizeError(error)
          setLastError(normalizedError.message)
          throw error
        } finally {
          handleVoidPromise(get().persistMemoriesIfEnabled(), 'sendGenericMessage: finally')
          setIsGenerating(false)
        }
      },

      resetConfig: () => {
        set({ config: defaultConfig })
        SecureStorage.getInstance().clear()
      },

      resetSession: () => {
        const memoryStore = useAIMemoryStore.getState()
        memoryStore.startNewSession({
          title: `Session ${new Date().toLocaleString()}`,
        })

        set({
          suggestions: [],
          lastError: null,
          isGenerating: false,
        })

        handleVoidPromise(get().persistMemoriesIfEnabled(), 'resetSession')
      },

      hydrateMemoriesFromBackend: async () => {
        const { config, memoriesHydrated } = get()
        if (!config.syncMemories) {
          set({ memoriesHydrated: true })
          return
        }

        if (memoriesHydrated) {
          return
        }

        try {
          const payload = await LoadAIMemorySessions()
          if (Array.isArray(payload) && payload.length > 0) {
            const sessions = deserializeMemorySessions(payload as WailsMemorySession[])
            const memoryStore = useAIMemoryStore.getState()
            memoryStore.importSessions(sessions, { merge: true })
          }
        } catch (error) {
          console.error('Failed to hydrate memories from backend:', error)
        } finally {
          set({ memoriesHydrated: true })
        }
      },

      persistMemoriesIfEnabled: async () => {
        const { config } = get()
        if (!config.syncMemories) {
          return
        }

        const sessions = useAIMemoryStore.getState().exportSessions()
        try {
          await SaveAIMemorySessions(serializeMemorySessions(sessions))
        } catch (error) {
          console.error('Failed to persist AI memories:', error)
        }
      },

      deleteMemorySession: async (sessionId: string) => {
        const memoryStore = useAIMemoryStore.getState()
        memoryStore.deleteSession(sessionId)
        await DeleteAIMemorySession(sessionId)
        await get().persistMemoriesIfEnabled()
      },
    }),
    {
      name: 'ai-store',
      storage: createJSONStorage(() => createSecureStorage()),
      partialize: (state) => ({
        config: {
          ...state.config,
          openaiApiKey: state.config.openaiApiKey ? 'STORED_SECURELY' : '',
          anthropicApiKey: state.config.anthropicApiKey ? 'STORED_SECURELY' : '',
          codexApiKey: state.config.codexApiKey ? 'STORED_SECURELY' : '',
        },
        memoriesHydrated: state.memoriesHydrated,
      }),
      version: 1,
      migrate: (persistedState: unknown) => {
        if (typeof persistedState !== 'object' || !persistedState) {
          return persistedState
        }

        const record = persistedState as { state?: { config?: Partial<AIConfig> } }
        if (!record.state?.config || record.state.config.syncMemories !== undefined) {
          return persistedState
        }

        return {
          ...record,
          state: {
            ...record.state,
            config: {
              ...record.state.config,
              syncMemories: true,
            },
          },
        }
      },
    },
  ),
)

export const useAIConfig = () => {
  const config = useAIStore(state => state.config)
  const updateConfig = useAIStore(state => state.updateConfig)
  const testConnection = useAIStore(state => state.testConnection)
  const connectionStatus = useAIStore(state => state.connectionStatus)

  return {
    config,
    updateConfig,
    testConnection,
    connectionStatus,
    isEnabled: config.enabled,
    hasValidApiKey: () => {
      switch (config.provider) {
        case 'openai':
          return !!config.openaiApiKey
        case 'anthropic':
          return !!config.anthropicApiKey
        case 'ollama':
          return !!config.ollamaEndpoint
        case 'huggingface':
          return !!config.huggingfaceEndpoint
        case 'claudecode':
          return !!config.claudeCodePath
        case 'codex':
          return !!config.codexApiKey
        default:
          return false
      }
    },
  }
}

export const useAIGeneration = () => {
  const generateSQL = useAIStore(state => state.generateSQL)
  const fixSQL = useAIStore(state => state.fixSQL)
  const sendGenericMessage = useAIStore(state => state.sendGenericMessage)
  const isGenerating = useAIStore(state => state.isGenerating)
  const lastError = useAIStore(state => state.lastError)
  const suggestions = useAIStore(state => state.suggestions)
  const clearSuggestions = useAIStore(state => state.clearSuggestions)
  const resetSession = useAIStore(state => state.resetSession)
  const hydrateMemoriesFromBackend = useAIStore(state => state.hydrateMemoriesFromBackend)
  const deleteMemorySession = useAIStore(state => state.deleteMemorySession)
  const persistMemoriesIfEnabled = useAIStore(state => state.persistMemoriesIfEnabled)

  return {
    generateSQL,
    fixSQL,
    sendGenericMessage,
    isGenerating,
    lastError,
    suggestions,
    clearSuggestions,
    resetSession,
    hydrateMemoriesFromBackend,
    deleteMemorySession,
    persistMemoriesIfEnabled,
  }
}
