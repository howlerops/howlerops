/**
 * AI Model Registry
 *
 * Single source of truth for all AI model configurations.
 * Update this file when new models are released.
 *
 * Last updated: 2026-02-06
 *
 * @module config/ai-models
 */

export interface AIModel {
  /** API model identifier */
  id: string
  /** Human-readable name */
  displayName: string
  /** Parent provider */
  provider: AIProvider
  /** Show as recommended in UI */
  isRecommended?: boolean
  /** Maps to "latest" alias */
  isLatest?: boolean
  /** Warn users to upgrade */
  isDeprecated?: boolean
  /** When it will be removed */
  deprecationDate?: string
  /** Max context window tokens */
  contextWindow?: number
  /** Model release date */
  releaseDate?: string
}

export type AIProvider =
  | 'openai'
  | 'anthropic'
  | 'ollama'
  | 'huggingface'
  | 'claudecode'
  | 'codex'

/** Version timestamp - update when models change */
export const MODEL_REGISTRY_VERSION = '2026-02-06'

/**
 * Complete model registry organized by provider
 */
export const AI_MODELS: Record<AIProvider, AIModel[]> = {
  openai: [
    {
      id: 'gpt-4o',
      displayName: 'GPT-4o',
      provider: 'openai',
      isRecommended: true,
      isLatest: true,
      contextWindow: 128000,
      releaseDate: '2024-05-13',
    },
    {
      id: 'gpt-4o-mini',
      displayName: 'GPT-4o Mini',
      provider: 'openai',
      contextWindow: 128000,
      releaseDate: '2024-07-18',
    },
    {
      id: 'gpt-4-turbo',
      displayName: 'GPT-4 Turbo',
      provider: 'openai',
      contextWindow: 128000,
    },
    {
      id: 'gpt-3.5-turbo',
      displayName: 'GPT-3.5 Turbo',
      provider: 'openai',
      isDeprecated: true,
      contextWindow: 16385,
    },
  ],

  anthropic: [
    // Claude 4.x family (Latest)
    {
      id: 'claude-opus-4-6-20260205',
      displayName: 'Claude Opus 4.6',
      provider: 'anthropic',
      isLatest: true,
      contextWindow: 200000,
      releaseDate: '2026-02-05',
    },
    {
      id: 'claude-opus-4-5-20251101',
      displayName: 'Claude Opus 4.5',
      provider: 'anthropic',
      contextWindow: 200000,
      releaseDate: '2025-11-01',
    },
    {
      id: 'claude-sonnet-4-5-20250929',
      displayName: 'Claude Sonnet 4.5',
      provider: 'anthropic',
      isRecommended: true,
      contextWindow: 200000,
      releaseDate: '2025-09-29',
    },
    {
      id: 'claude-haiku-4-5-20251001',
      displayName: 'Claude Haiku 4.5',
      provider: 'anthropic',
      contextWindow: 200000,
      releaseDate: '2025-10-01',
    },
    {
      id: 'claude-sonnet-4-20250514',
      displayName: 'Claude Sonnet 4',
      provider: 'anthropic',
      contextWindow: 200000,
      releaseDate: '2025-05-14',
    },
    // Claude 3.x family (Legacy)
    {
      id: 'claude-3-5-sonnet-20241022',
      displayName: 'Claude 3.5 Sonnet',
      provider: 'anthropic',
      isDeprecated: true,
      contextWindow: 200000,
      releaseDate: '2024-10-22',
    },
    {
      id: 'claude-3-5-haiku-20241022',
      displayName: 'Claude 3.5 Haiku',
      provider: 'anthropic',
      isDeprecated: true,
      contextWindow: 200000,
      releaseDate: '2024-10-22',
    },
    {
      id: 'claude-3-opus-20240229',
      displayName: 'Claude 3 Opus',
      provider: 'anthropic',
      isDeprecated: true,
      contextWindow: 200000,
      releaseDate: '2024-02-29',
    },
  ],

  ollama: [
    {
      id: 'llama3.2:latest',
      displayName: 'Llama 3.2',
      provider: 'ollama',
      isRecommended: true,
      isLatest: true,
    },
    {
      id: 'sqlcoder:7b',
      displayName: 'SQLCoder 7B',
      provider: 'ollama',
    },
    {
      id: 'codellama:7b',
      displayName: 'CodeLlama 7B',
      provider: 'ollama',
    },
    {
      id: 'mistral:7b',
      displayName: 'Mistral 7B',
      provider: 'ollama',
    },
  ],

  huggingface: [
    {
      id: 'meta-llama/Llama-3.2-3B',
      displayName: 'Llama 3.2 3B',
      provider: 'huggingface',
      isRecommended: true,
      isLatest: true,
    },
    {
      id: 'sqlcoder:7b',
      displayName: 'SQLCoder 7B',
      provider: 'huggingface',
    },
    {
      id: 'codellama:7b',
      displayName: 'CodeLlama 7B',
      provider: 'huggingface',
    },
  ],

  claudecode: [
    {
      id: 'opus',
      displayName: 'Opus (Most Capable)',
      provider: 'claudecode',
      isRecommended: true,
      isLatest: true,
    },
    {
      id: 'sonnet',
      displayName: 'Sonnet (Balanced)',
      provider: 'claudecode',
    },
    {
      id: 'haiku',
      displayName: 'Haiku (Fast)',
      provider: 'claudecode',
    },
  ],

  codex: [
    {
      id: 'code-davinci-002',
      displayName: 'Code Davinci 002',
      provider: 'codex',
      isDeprecated: true,
      deprecationDate: '2024-01-01',
    },
  ],
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Get all models for a provider
 */
export function getModelsForProvider(provider: AIProvider): AIModel[] {
  return AI_MODELS[provider] || []
}

/**
 * Get the recommended model for a provider
 */
export function getRecommendedModel(provider: AIProvider): AIModel | undefined {
  return AI_MODELS[provider]?.find((m) => m.isRecommended)
}

/**
 * Get the latest model for a provider
 */
export function getLatestModel(provider: AIProvider): AIModel | undefined {
  return AI_MODELS[provider]?.find((m) => m.isLatest) || getRecommendedModel(provider)
}

/**
 * Get the default model ID for a provider (recommended or first available)
 */
export function getDefaultModelId(provider: AIProvider): string {
  return getRecommendedModel(provider)?.id || AI_MODELS[provider]?.[0]?.id || ''
}

/**
 * Resolve "latest" alias to actual model ID
 */
export function resolveModelAlias(provider: AIProvider, modelId: string): string {
  if (modelId === 'latest') {
    return getLatestModel(provider)?.id || modelId
  }
  return modelId
}

/**
 * Check if a model is deprecated
 */
export function isModelDeprecated(provider: AIProvider, modelId: string): boolean {
  const model = AI_MODELS[provider]?.find((m) => m.id === modelId)
  return model?.isDeprecated || false
}

/**
 * Get a model by its ID
 */
export function getModelById(provider: AIProvider, modelId: string): AIModel | undefined {
  return AI_MODELS[provider]?.find((m) => m.id === modelId)
}

/**
 * Get non-deprecated models for a provider
 */
export function getActiveModels(provider: AIProvider): AIModel[] {
  return AI_MODELS[provider]?.filter((m) => !m.isDeprecated) || []
}

/**
 * Get the recommended model map (for backwards compatibility)
 */
export function getRecommendedModelMap(): Record<AIProvider, string> {
  const map: Partial<Record<AIProvider, string>> = {}
  for (const provider of Object.keys(AI_MODELS) as AIProvider[]) {
    map[provider] = getDefaultModelId(provider)
  }
  return map as Record<AIProvider, string>
}

// =============================================================================
// Dynamic Model Fetching (with static fallback + merge)
// =============================================================================

/** Response from the GetAvailableModels Wails binding */
interface DynamicModelInfo {
  id: string
  name: string
  provider: string
  description?: string
  maxTokens?: number
  source?: string
}

/** Cache for fetched models to avoid repeated API calls */
const modelCache: Partial<Record<AIProvider, { models: AIModel[]; fetchedAt: number }>> = {}
const CACHE_TTL_MS = 5 * 60 * 1000 // 5 minutes

/**
 * Fetch models dynamically from the backend (which queries provider APIs),
 * merge with static registry metadata, and return the combined list.
 *
 * Falls back to static registry if the backend call fails.
 */
export async function fetchModelsForProvider(provider: AIProvider): Promise<AIModel[]> {
  // Check cache
  const cached = modelCache[provider]
  if (cached && Date.now() - cached.fetchedAt < CACHE_TTL_MS) {
    return cached.models
  }

  try {
    const { GetAvailableModels } = await import(
      '../../bindings/github.com/jbeck018/howlerops/app'
    )
    const dynamicModels: DynamicModelInfo[] = await GetAvailableModels(provider)

    if (!dynamicModels || dynamicModels.length === 0) {
      return getModelsForProvider(provider)
    }

    // Merge: dynamic models enriched with static metadata
    const staticModels = getModelsForProvider(provider)
    const staticById = new Map(staticModels.map((m) => [m.id, m]))

    const merged: AIModel[] = dynamicModels.map((dm) => {
      const staticMatch = staticById.get(dm.id)

      if (staticMatch) {
        // Dynamic model exists in static registry - use static metadata
        return staticMatch
      }

      // New model from API not in static registry
      return {
        id: dm.id,
        displayName: dm.name || dm.id,
        provider,
        contextWindow: dm.maxTokens,
      }
    })

    // Also include static-only models not returned by API (e.g. deprecated ones users may still have selected)
    for (const sm of staticModels) {
      if (!merged.some((m) => m.id === sm.id)) {
        merged.push(sm)
      }
    }

    // Cache the result
    modelCache[provider] = { models: merged, fetchedAt: Date.now() }

    return merged
  } catch {
    // Fallback to static registry
    return getModelsForProvider(provider)
  }
}

/**
 * Invalidate the model cache for a provider (e.g. after config change)
 */
export function invalidateModelCache(provider?: AIProvider) {
  if (provider) {
    delete modelCache[provider]
  } else {
    for (const key of Object.keys(modelCache) as AIProvider[]) {
      delete modelCache[key]
    }
  }
}
