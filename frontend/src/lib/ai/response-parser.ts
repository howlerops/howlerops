/**
 * AI Response Parser
 *
 * Parses and validates responses from the AI backend.
 */

import type { AIGenerateSQLResponse, AIFixSQLResponse, AIGenericMessageResponse, AIError } from '@/types/ai'
import { createAIError } from './request-validator'

/**
 * Parses a SQL generation response from the backend
 * @throws {AIError} If response is invalid
 */
export function parseGenerateSQLResponse(response: unknown): AIGenerateSQLResponse {
  if (!response || typeof response !== 'object') {
    throw createAIError('response-invalid', 'No response from AI service')
  }

  const data = response as Record<string, unknown>

  if (!data.sql || typeof data.sql !== 'string') {
    throw createAIError('response-invalid', 'Invalid SQL in response')
  }

  return {
    sql: data.sql,
    explanation: typeof data.explanation === 'string' ? data.explanation : undefined,
    confidence: typeof data.confidence === 'number' ? data.confidence : 0.8,
    warnings: Array.isArray(data.warnings) ? (data.warnings as string[]) : undefined,
  }
}

/**
 * Parses a SQL fix response from the backend
 * @throws {AIError} If response is invalid
 */
export function parseFixSQLResponse(response: unknown): AIFixSQLResponse {
  if (!response || typeof response !== 'object') {
    throw createAIError('response-invalid', 'No response from AI service')
  }

  const data = response as Record<string, unknown>

  if (!data.sql || typeof data.sql !== 'string') {
    throw createAIError('response-invalid', 'Invalid SQL in response')
  }

  return {
    sql: data.sql,
    explanation: typeof data.explanation === 'string' ? data.explanation : 'Fixed SQL query',
    confidence: typeof data.confidence === 'number' ? data.confidence : 0.9,
    warnings: Array.isArray(data.warnings) ? (data.warnings as string[]) : undefined,
  }
}

/**
 * Parses a generic message response from the backend
 * @throws {AIError} If response is invalid
 */
export function parseGenericMessageResponse(
  response: unknown,
  fallbackProvider: string,
  fallbackModel: string
): AIGenericMessageResponse {
  if (!response || typeof response !== 'object') {
    throw createAIError('response-invalid', 'No response from AI service')
  }

  const data = response as Record<string, unknown>

  const content = typeof data.content === 'string' ? data.content : ''
  const provider = typeof data.provider === 'string' ? data.provider : fallbackProvider
  const model = typeof data.model === 'string' ? data.model : fallbackModel
  const tokensUsed = typeof data.tokensUsed === 'number' ? data.tokensUsed : 0

  return {
    content,
    provider,
    model,
    tokensUsed,
    metadata: data.metadata && typeof data.metadata === 'object'
      ? (data.metadata as Record<string, string>)
      : undefined,
  }
}

/**
 * Extracts error message from various error types
 */
export function extractErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }

  if (typeof error === 'string') {
    return error
  }

  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message: unknown }).message)
  }

  return 'Unknown error'
}

/**
 * Converts any error to a structured AIError
 */
export function normalizeError(error: unknown): AIError {
  // Already an AIError
  if (error && typeof error === 'object' && 'type' in error && 'message' in error) {
    return error as AIError
  }

  // Convert to AIError
  const message = extractErrorMessage(error)
  const originalError = error instanceof Error ? error : undefined

  return createAIError('unknown', message, originalError)
}
