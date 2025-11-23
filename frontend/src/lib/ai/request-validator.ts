/**
 * AI Request Validator
 *
 * Validates AI requests before sending to ensure all required
 * parameters are present and valid.
 */

import type { AIConfig } from '@/store/ai-store'
import type { AIError, AIErrorType, AIFixSQLRequest, AIGenerateSQLRequest, AIGenericMessageRequest } from '@/types/ai'

/**
 * Creates a structured AI error
 */
export function createAIError(
  type: AIErrorType,
  message: string,
  originalError?: Error,
  context?: Record<string, unknown>
): AIError {
  return {
    type,
    message,
    originalError,
    context,
  }
}

/**
 * Validates that AI features are enabled
 * @throws {AIError} If AI is disabled
 */
export function validateAIEnabled(config: AIConfig): void {
  if (!config.enabled) {
    throw createAIError('disabled', 'AI features are disabled')
  }
}

/**
 * Validates that a provider is configured
 * @throws {AIError} If provider requires credentials that are missing
 */
export function validateProviderConfig(config: AIConfig): void {
  const { provider, openaiApiKey, anthropicApiKey, codexApiKey, ollamaEndpoint, huggingfaceEndpoint, selectedModel } = config

  switch (provider) {
    case 'openai':
      if (!openaiApiKey) {
        throw createAIError('no-api-key', 'OpenAI API key not configured')
      }
      break

    case 'anthropic':
      if (!anthropicApiKey) {
        throw createAIError('no-api-key', 'Anthropic API key not configured')
      }
      break

    case 'codex':
      if (!codexApiKey) {
        throw createAIError('no-api-key', 'Codex API key not configured')
      }
      break

    case 'ollama':
      if (!ollamaEndpoint) {
        throw createAIError('no-provider', 'Ollama endpoint not configured')
      }
      if (!selectedModel) {
        throw createAIError('no-model', 'Select a model before using Ollama')
      }
      break

    case 'huggingface':
      if (!huggingfaceEndpoint) {
        throw createAIError('no-provider', 'Hugging Face endpoint not configured')
      }
      if (!selectedModel) {
        throw createAIError('no-model', 'Select a model before using Hugging Face')
      }
      break

    case 'claudecode':
      // claudecode doesn't require strict validation
      break

    default:
      throw createAIError('no-provider', `Unknown provider: ${provider}`)
  }
}

/**
 * Validates a SQL generation request
 * @throws {AIError} If request is invalid
 */
export function validateGenerateSQLRequest(request: AIGenerateSQLRequest): void {
  if (!request.prompt || request.prompt.trim() === '') {
    throw createAIError('request-failed', 'Prompt cannot be empty')
  }

  // Multi-DB mode validation
  if (request.mode === 'multi') {
    if (!request.connections || request.connections.length === 0) {
      throw createAIError('request-failed', 'Multi-DB mode requires connections')
    }

    const connectedDbs = request.connections.filter(c => c.isConnected)
    if (connectedDbs.length === 0) {
      throw createAIError('request-failed', 'No connected databases in multi-DB mode')
    }
  }
}

/**
 * Validates a SQL fix request
 * @throws {AIError} If request is invalid
 */
export function validateFixSQLRequest(request: AIFixSQLRequest): void {
  if (!request.query || request.query.trim() === '') {
    throw createAIError('request-failed', 'Query cannot be empty')
  }

  if (!request.error || request.error.trim() === '') {
    throw createAIError('request-failed', 'Error message cannot be empty')
  }
}

/**
 * Validates a generic message request
 * @throws {AIError} If request is invalid
 */
export function validateGenericMessageRequest(request: AIGenericMessageRequest): void {
  if (!request.prompt || request.prompt.trim() === '') {
    throw createAIError('request-failed', 'Prompt cannot be empty')
  }
}
