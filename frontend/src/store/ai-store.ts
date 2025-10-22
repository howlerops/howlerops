import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { testAIProviderConnection, showHybridNotification } from '@/lib/wails-ai-api'
import { LoadAIMemorySessions, SaveAIMemorySessions, RecallAIMemorySessions, DeleteAIMemorySession, ConfigureAIProvider } from '../../wailsjs/go/main/App'
import { main as wailsModels } from '../../wailsjs/go/models'
import type { SchemaNode } from '@/hooks/use-schema-introspection'
import type { DatabaseConnection } from '@/store/connection-store'
import { useAIMemoryStore, estimateTokens as estimateMemoryTokens, type AIMemorySession as MemorySession } from '@/store/ai-memory-store'

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

// Secure storage for API keys
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

  async setItem(key: string, value: string): Promise<void> {
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
      throw new Error('Failed to store secure data')
    }
  }

  async getItem(key: string): Promise<string | null> {
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

  async removeItem(key: string): Promise<void> {
    try {
      sessionStorage.removeItem(this.keyPrefix + key)
    } catch (error) {
      console.error('Failed to remove secure item:', error)
    }
    this.fallback.delete(this.keyPrefix + key)
  }

  async clear(): Promise<void> {
    try {
      const keys = Object.keys(sessionStorage).filter(key => key.startsWith(this.keyPrefix))
      keys.forEach(key => sessionStorage.removeItem(key))
    } catch (error) {
      console.error('Failed to clear secure storage:', error)
    }
    this.fallback.clear()
  }
}

// Custom storage for sensitive data
const createSecureStorage = () => {
  const storage = SecureStorage.getInstance()

  return {
    getItem: async (name: string): Promise<string | null> => {
      const value = localStorage.getItem(name)
      if (!value) return null

      try {
        const parsed = JSON.parse(value)

        if (parsed.state?.config) {
          const config = parsed.state.config
          if (config.openaiApiKey === 'STORED_SECURELY') {
            config.openaiApiKey = await storage.getItem('openai-api-key') || ''
          }
          if (config.anthropicApiKey === 'STORED_SECURELY') {
            config.anthropicApiKey = await storage.getItem('anthropic-api-key') || ''
          }
          if (config.codexApiKey === 'STORED_SECURELY') {
            config.codexApiKey = await storage.getItem('codex-api-key') || ''
          }
        }

        return JSON.stringify(parsed)
      } catch {
        return value
      }
    },

    setItem: async (name: string, value: string): Promise<void> => {
      try {
        const parsed = JSON.parse(value)

        if (parsed.state?.config) {
          const config = parsed.state.config

          if (config.openaiApiKey && config.openaiApiKey !== 'STORED_SECURELY') {
            await storage.setItem('openai-api-key', config.openaiApiKey)
            config.openaiApiKey = 'STORED_SECURELY'
          }

          if (config.anthropicApiKey && config.anthropicApiKey !== 'STORED_SECURELY') {
            await storage.setItem('anthropic-api-key', config.anthropicApiKey)
            config.anthropicApiKey = 'STORED_SECURELY'
          }

          if (config.codexApiKey && config.codexApiKey !== 'STORED_SECURELY') {
            await storage.setItem('codex-api-key', config.codexApiKey)
            config.codexApiKey = 'STORED_SECURELY'
          }
        }

        localStorage.setItem(name, JSON.stringify(parsed))
      } catch {
        localStorage.setItem(name, value)
      }
    },

    removeItem: async (name: string): Promise<void> => {
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
            get().hydrateMemoriesFromBackend()
              .catch((error) => console.error('Failed to hydrate AI memories:', error))
              .finally(() => {
                void get().persistMemoriesIfEnabled()
              })
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

      generateSQL: async (prompt: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()
        const memoryStore = useAIMemoryStore.getState()

        const provider = (config.provider || 'openai').toLowerCase()
        const sessionId = memoryStore.ensureActiveSession({
          title: `Session ${new Date().toLocaleString()}`,
        })

        if (!config.enabled) {
          throw new Error('AI features are disabled')
        }

        setIsGenerating(true)
        setLastError(null)

        try {
          await get().ensureProviderConfigured()

          // Import Wails bindings
          const { GenerateSQLFromNaturalLanguage } = await import('../../wailsjs/go/main/App')
          const { AISchemaContextBuilder } = await import('@/lib/ai-schema-context')

          // Auto-detect if user wants multi-DB query based on prompt
          const detectsMultiDB = (prompt: string): boolean => {
            const multiDBKeywords = [
              /join.*from.*and.*from/i,
              /across.*database/i,
              /between.*database/i,
              /from.*database.*and.*database/i,
              /@\w+\./,  // Already using @connection syntax
              /compare.*from.*and/i,
              /merge.*from.*and/i,
              /combine.*from.*and/i,
              /different.*database/i,
              /multiple.*database/i,
            ]
            return multiDBKeywords.some(pattern => pattern.test(prompt))
          }

          // Determine effective mode: auto-detect or use provided mode
          const wantsMultiDB = detectsMultiDB(prompt)
          const effectiveMode = wantsMultiDB ? 'multi' : (mode || 'single')
          
          console.log(`🤖 AI Query Mode Detection:`, {
            providedMode: mode,
            detectedMultiDB: wantsMultiDB,
            effectiveMode,
            hasConnections: !!connections && connections.length > 0
          })

          // Build schema context based on effective mode
          let context = ''
          let enhancedPrompt = prompt

          const memoryContext = memoryStore.buildContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          memoryStore.recordMessage({
            sessionId,
            role: 'user',
            content: prompt,
            metadata: {
              mode: effectiveMode,
              provider,
            },
          })
          
          if (effectiveMode === 'multi' && connections && schemasMap && connections.length > 1) {
            // Multi-database mode: build comprehensive context
            const multiContext = AISchemaContextBuilder.buildMultiDatabaseContext(
              connections.filter(c => c.isConnected),
              schemasMap,
              undefined // Active connection ID not available here
            )
            context = AISchemaContextBuilder.generateCompactSchemaContext(multiContext)

            // Enhance prompt with multi-DB syntax instructions
            enhancedPrompt = `${prompt}\n\nIMPORTANT: This is a multi-database query. Use @connection_name.table syntax to reference tables from different databases. Available connections: ${connections.filter(c => c.isConnected).map(c => c.name).join(', ')}`
          } else if (schema) {
            // Single database mode: use simple schema context
            context = `Database: ${schema}`
          }

          if (memoryContext) {
            context = context
              ? `${context}\n\n---\n\nConversation Memory:\n${memoryContext}`
              : `Conversation Memory:\n${memoryContext}`
          }

          if (config.syncMemories) {
            try {
              const recalled = await RecallAIMemorySessions(prompt, 5)
              if (Array.isArray(recalled) && recalled.length > 0) {
                const recallContext = recalled
                  .map((item) => {
                    const summary = item.summary ? `Summary: ${item.summary}\n` : ''
                    return `Session: ${item.title}\n${summary}Memory:\n${item.content}`
                  })
                  .join('\n---\n')

                context = context
                  ? `${context}\n\n---\n\nRelated Sessions:\n${recallContext}`
                  : `Related Sessions:\n${recallContext}`
              }
            } catch (error) {
              console.error('Failed to recall AI memories for fix:', error)
            }
          }

          let recallContext = ''
          if (config.syncMemories) {
            try {
              const recalled = await RecallAIMemorySessions(prompt, 5)
              if (Array.isArray(recalled) && recalled.length > 0) {
                recallContext = recalled
                  .map((item) => {
                    const summary = item.summary ? `Summary: ${item.summary}\n` : ''
                    return `Session: ${item.title}\n${summary}Memory:\n${item.content}`
                  })
                  .join('\n---\n')
              }
            } catch (error) {
              console.error('Failed to recall AI memories:', error)
            }
          }

          if (recallContext) {
            context = context
              ? `${context}\n\n---\n\nRelated Sessions:\n${recallContext}`
              : `Related Sessions:\n${recallContext}`
          }

          // Call the Wails backend method
          const model = config.selectedModel || 'gpt-4o-mini'

          const request = {
            prompt: enhancedPrompt,
            connectionId: connections?.[0]?.id || '', // Use first connection if available
            context,
            provider,
            model,
            maxTokens: config.maxTokens,
            temperature: config.temperature,
          }

          const result = await GenerateSQLFromNaturalLanguage(request)

          if (!result) {
            throw new Error('No response from AI service')
          }

          addSuggestion({
            query: result.sql,
            explanation: result.explanation || '',
            confidence: result.confidence || 0.8,
            provider: config.provider,
            model: config.selectedModel,
          })

          memoryStore.recordMessage({
            sessionId,
            role: 'assistant',
            content: result.sql,
            metadata: {
              type: 'generation',
              explanation: result.explanation,
            },
          })

          void get().persistMemoriesIfEnabled()
          return result.sql
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setLastError(errorMessage)
          void get().persistMemoriesIfEnabled()
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      fixSQL: async (query: string, error: string, schema?: string, mode?: 'single' | 'multi', connections?: DatabaseConnection[], schemasMap?: Map<string, SchemaNode[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()
        const memoryStore = useAIMemoryStore.getState()
        const provider = (config.provider || 'openai').toLowerCase()
        const sessionId = memoryStore.ensureActiveSession({
          title: `Session ${new Date().toLocaleString()}`,
        })

        if (!config.enabled) {
          throw new Error('AI features are disabled')
        }

        setIsGenerating(true)
        setLastError(null)

        try {
          await get().ensureProviderConfigured()

          // Import Wails bindings
          const { FixSQLErrorWithOptions } = await import('../../wailsjs/go/main/App')
          const { AISchemaContextBuilder } = await import('@/lib/ai-schema-context')

          // Build schema context for RAG
          let context = ''
          let enhancedError = error

          if (mode === 'multi' && connections && schemasMap && connections.length > 1) {
            // Multi-database mode: build comprehensive context
            const multiContext = AISchemaContextBuilder.buildMultiDatabaseContext(
              connections.filter(c => c.isConnected),
              schemasMap,
              undefined
            )
            context = AISchemaContextBuilder.generateCompactSchemaContext(multiContext)

            enhancedError = `${error}\n\nNote: This is a multi-database query. Tables should use @connection_name.table syntax. Available connections: ${connections.filter(c => c.isConnected).map(c => c.name).join(', ')}`
          } else if (schema) {
            // Single database mode: use simple schema context
            context = `Database: ${schema}`
          }

          const memoryContext = memoryStore.buildContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          if (memoryContext) {
            context = context
              ? `${context}\n\n---\n\nConversation Memory:\n${memoryContext}`
              : `Conversation Memory:\n${memoryContext}`
          }

          memoryStore.recordMessage({
            sessionId,
            role: 'user',
            content: `Fix query:\n${query}\n\nError:\n${error}`,
            metadata: {
              type: 'fix-request',
              provider,
            },
          })

          // Call the Wails backend method with context
          const connectionId = connections?.[0]?.id || ''
          const model = config.selectedModel || ''
          const request = {
            query,
            error: enhancedError,
            connectionId,
            context,
            provider,
            model,
            maxTokens: config.maxTokens,
            temperature: config.temperature,
          }

          const result = await FixSQLErrorWithOptions(request)

          if (!result) {
            throw new Error('No response from AI service')
          }

          addSuggestion({
            query: result.sql,
            explanation: result.explanation || 'Fixed SQL query',
            confidence: 0.9,
            provider: config.provider,
            model: config.selectedModel,
          })

          memoryStore.recordMessage({
            sessionId,
            role: 'assistant',
            content: result.sql,
            metadata: {
              type: 'fix-response',
              explanation: result.explanation,
            },
          })

          void get().persistMemoriesIfEnabled()
          return result.sql
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setLastError(errorMessage)
          void get().persistMemoriesIfEnabled()
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      sendGenericMessage: async (prompt: string, options?: {
        context?: string
        systemPrompt?: string
        metadata?: Record<string, string>
      }) => {
        const { config, setIsGenerating, setLastError } = get()
        const memoryStore = useAIMemoryStore.getState()

        if (!config.enabled) {
          throw new Error('AI features are disabled')
        }

        const provider = (config.provider || 'openai').toLowerCase()
        let sessionId = memoryStore.ensureActiveSession({
          title: `Chat Session ${new Date().toLocaleString()}`,
          metadata: {
            chatType: 'generic',
          },
        })

        const activeSession = memoryStore.sessions[sessionId]
        if (activeSession?.metadata?.chatType !== 'generic') {
          sessionId = memoryStore.startNewSession({
            title: `Chat Session ${new Date().toLocaleString()}`,
            metadata: { chatType: 'generic' },
          })
          memoryStore.setActiveSession(sessionId)
        }

        setIsGenerating(true)
        setLastError(null)

        try {
          await get().ensureProviderConfigured()

          const baseContext = options?.context?.trim() ?? ''
          const memoryContext = memoryStore.buildContext({
            sessionId,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
          })

          let combinedContext = baseContext
          if (memoryContext) {
            combinedContext = combinedContext
              ? `${combinedContext}\n\n---\n\nConversation Memory:\n${memoryContext}`
              : `Conversation Memory:\n${memoryContext}`
          }

          memoryStore.recordMessage({
            sessionId,
            role: 'user',
            content: prompt,
            metadata: {
              type: 'chat:user',
              provider,
            },
          })

          const { GenericChat } = await import('../../wailsjs/go/main/App')
          const response = await GenericChat({
            prompt,
            context: combinedContext,
            system: options?.systemPrompt,
            provider,
            model: config.selectedModel,
            maxTokens: config.maxTokens,
            temperature: config.temperature,
            metadata: {
              sessionId,
              chatType: 'generic',
              ...options?.metadata,
            },
          })

          const assistantContent = response?.content ?? ''

          memoryStore.recordMessage({
            sessionId,
            role: 'assistant',
            content: assistantContent,
            metadata: {
              type: 'chat:assistant',
              provider: response?.provider || provider,
              tokensUsed: response?.tokensUsed ?? 0,
            },
          })

          return {
            content: assistantContent,
            provider: response?.provider || provider,
            model: response?.model || config.selectedModel,
            tokensUsed: response?.tokensUsed ?? 0,
            metadata: response?.metadata,
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setLastError(errorMessage)
          throw error
        } finally {
          void get().persistMemoriesIfEnabled()
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

        void get().persistMemoriesIfEnabled()
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
