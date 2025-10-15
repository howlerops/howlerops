import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { testAIProviderConnection, showHybridNotification } from '@/lib/wails-ai-api'

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

const normalizeEndpoint = (endpoint?: string): string => {
  if (!endpoint || endpoint.trim() === '') {
    return 'http://localhost:11434'
  }

  if (/^https?:\/\//i.test(endpoint.trim())) {
    return endpoint.trim()
  }

  return `http://${endpoint.trim()}`
}

const isLocalOllamaEndpoint = (endpoint: string): boolean => {
  try {
    const url = new URL(normalizeEndpoint(endpoint))
    const hostname = url.hostname.toLowerCase()
    return ['localhost', '127.0.0.1', '0.0.0.0', '[::1]'].includes(hostname)
  } catch {
    return false
  }
}

const dispatchOllamaStatusRefresh = () => {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('ollama-status:refresh'))
  }
}

interface OllamaDetectionStatus {
  installed: boolean
  running: boolean
  available_models?: string[]
  endpoint?: string
  version?: string
  last_checked?: string
  error?: string
  backend_available?: boolean
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
}

export interface AIActions {
  updateConfig: (config: Partial<AIConfig>) => void
  setIsGenerating: (generating: boolean) => void
  addSuggestion: (suggestion: Omit<SQLSuggestion, 'id' | 'timestamp'>) => void
  clearSuggestions: () => void
  setLastError: (error: string | null) => void
  setConnectionStatus: (provider: string, status: string) => void
  testConnection: (provider: string) => Promise<boolean>
  generateSQL: (prompt: string, schema?: string, mode?: 'single' | 'multi', connections?: any[], schemasMap?: Map<string, any[]>) => Promise<string>
  fixSQL: (query: string, error: string, schema?: string, mode?: 'single' | 'multi', connections?: any[], schemasMap?: Map<string, any[]>) => Promise<string>
  resetConfig: () => void
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
}

// Secure storage for API keys
class SecureStorage {
  private static instance: SecureStorage
  private keyPrefix = 'ai-secure-'

  static getInstance(): SecureStorage {
    if (!SecureStorage.instance) {
      SecureStorage.instance = new SecureStorage()
    }
    return SecureStorage.instance
  }

  async setItem(key: string, value: string): Promise<void> {
    try {
      sessionStorage.setItem(this.keyPrefix + key, value)
    } catch (error) {
      console.error('Failed to store secure item:', error)
      throw new Error('Failed to store secure data')
    }
  }

  async getItem(key: string): Promise<string | null> {
    try {
      return sessionStorage.getItem(this.keyPrefix + key)
    } catch (error) {
      console.error('Failed to retrieve secure item:', error)
      return null
    }
  }

  async removeItem(key: string): Promise<void> {
    try {
      sessionStorage.removeItem(this.keyPrefix + key)
    } catch (error) {
      console.error('Failed to remove secure item:', error)
    }
  }

  async clear(): Promise<void> {
    try {
      const keys = Object.keys(sessionStorage).filter(key => key.startsWith(this.keyPrefix))
      keys.forEach(key => sessionStorage.removeItem(key))
    } catch (error) {
      console.error('Failed to clear secure storage:', error)
    }
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
        set(state => ({
          config: { ...state.config, ...configUpdate },
        }))
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

          // Show error notification using hybrid approach (toast + dialog for errors)
          const errorMessage = error instanceof Error ? error.message : 'Connection test failed'
          await showHybridNotification('Connection Test Failed', errorMessage, true, true)

          throw error // Re-throw so the frontend can handle it
        }
      },

      generateSQL: async (prompt: string, schema?: string, mode?: 'single' | 'multi', connections?: any[], schemasMap?: Map<string, any[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()

        if (!config.enabled) {
          throw new Error('AI features are disabled')
        }

        setIsGenerating(true)
        setLastError(null)

        try {
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
          
          if (effectiveMode === 'multi' && connections && schemasMap && connections.length > 1) {
            // Multi-database mode: build comprehensive context
            const multiContext = AISchemaContextBuilder.buildMultiDatabaseContext(
              connections.filter(c => c.isConnected),
              schemasMap,
              connections.find(c => c.isActive)?.id
            )
            context = AISchemaContextBuilder.generateCompactSchemaContext(multiContext)

            // Enhance prompt with multi-DB syntax instructions
            enhancedPrompt = `${prompt}\n\nIMPORTANT: This is a multi-database query. Use @connection_name.table syntax to reference tables from different databases. Available connections: ${connections.filter(c => c.isConnected).map(c => c.name).join(', ')}`
          } else if (schema) {
            // Single database mode: use simple schema context
            context = `Database: ${schema}`
          }

          // Call the Wails backend method
          const request = {
            prompt: enhancedPrompt,
            connectionId: connections?.find(c => c.isActive)?.id || '',
            context
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

          return result.sql
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setLastError(errorMessage)
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      fixSQL: async (query: string, error: string, schema?: string, mode?: 'single' | 'multi', connections?: any[], schemasMap?: Map<string, any[]>): Promise<string> => {
        const { config, setIsGenerating, addSuggestion, setLastError } = get()

        if (!config.enabled) {
          throw new Error('AI features are disabled')
        }

        setIsGenerating(true)
        setLastError(null)

        try {
          // Import Wails bindings
          const { FixSQLError } = await import('../../wailsjs/go/main/App')

          // Add context about multi-DB mode to the error if applicable
          let enhancedError = error
          if (mode === 'multi') {
            enhancedError = `${error}\n\nNote: This is a multi-database query. Tables should use @connection_name.table syntax. Available connections: ${connections?.filter(c => c.isConnected).map(c => c.name).join(', ') || 'none'}`
          }

          // Call the Wails backend method
          const connectionId = connections?.find(c => c.isActive)?.id || ''
          const result = await FixSQLError(query, enhancedError, connectionId)

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

          return result.sql
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setLastError(errorMessage)
          throw error
        } finally {
          setIsGenerating(false)
        }
      },

      resetConfig: () => {
        set({ config: defaultConfig })
        SecureStorage.getInstance().clear()
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
      }),
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
  const isGenerating = useAIStore(state => state.isGenerating)
  const lastError = useAIStore(state => state.lastError)
  const suggestions = useAIStore(state => state.suggestions)
  const clearSuggestions = useAIStore(state => state.clearSuggestions)

  return {
    generateSQL,
    fixSQL,
    isGenerating,
    lastError,
    suggestions,
    clearSuggestions,
  }
}
