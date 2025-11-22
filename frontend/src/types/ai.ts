/**
 * AI Types Module
 *
 * Provides type-safe types and interfaces for AI operations.
 * This module centralizes all AI-related type definitions to improve
 * type safety and maintainability across the AI store and related components.
 */

import type { SchemaNode } from '@/hooks/use-schema-introspection'
import type { DatabaseConnection } from '@/store/connection-store'

/**
 * Branded Types
 * Using branded types ensures IDs cannot be accidentally swapped
 */
type Brand<T, B> = T & { readonly __brand: B }

/** Branded type for AI session identifiers */
export type AISessionId = Brand<string, 'AISessionId'>

/** Branded type for database connection identifiers */
export type ConnectionId = Brand<string, 'ConnectionId'>

/** Branded type for AI turn identifiers (for streaming) */
export type TurnId = Brand<string, 'TurnId'>

/**
 * Creates a validated AISessionId from a string
 */
export function createAISessionId(id: string): AISessionId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid AISessionId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as AISessionId
}

/**
 * Creates a validated ConnectionId from a string
 */
export function createConnectionId(id: string): ConnectionId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid ConnectionId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as ConnectionId
}

/**
 * Creates a validated TurnId from a string
 */
export function createTurnId(id: string): TurnId {
  if (typeof id !== 'string' || id.trim() === '') {
    throw new TypeError(`Invalid TurnId: expected non-empty string, got ${typeof id === 'string' ? `"${id}"` : typeof id}`)
  }
  return id as TurnId
}

/**
 * AI query mode - single database or multi-database
 */
export type AIQueryMode = 'single' | 'multi'

/**
 * AI provider types
 */
export type AIProvider = 'openai' | 'anthropic' | 'ollama' | 'huggingface' | 'claudecode' | 'codex'

/**
 * Request options for AI SQL generation
 */
export interface AIGenerateSQLRequest {
  /** Natural language prompt */
  prompt: string
  /** Database schema context (optional) */
  schema?: string
  /** Query mode */
  mode?: AIQueryMode
  /** Connected databases (for multi-DB mode) */
  connections?: DatabaseConnection[]
  /** Schema map for multi-DB mode */
  schemasMap?: Map<string, SchemaNode[]>
  /** Primary connection ID */
  connectionId?: string
  /** AI session ID */
  sessionId?: AISessionId
  /** Memory context */
  memoryContext?: string
  /** Provider override */
  provider?: string
  /** Model override */
  model?: string
  /** Max tokens override */
  maxTokens?: number
  /** Temperature override */
  temperature?: number
}

/**
 * Request options for AI SQL fixing
 */
export interface AIFixSQLRequest {
  /** SQL query to fix */
  query: string
  /** Error message to fix */
  error: string
  /** Database schema context (optional) */
  schema?: string
  /** Query mode */
  mode?: AIQueryMode
  /** Connected databases (for multi-DB mode) */
  connections?: DatabaseConnection[]
  /** Schema map for multi-DB mode */
  schemasMap?: Map<string, SchemaNode[]>
  /** Primary connection ID */
  connectionId?: string
  /** AI session ID */
  sessionId?: AISessionId
  /** Memory context */
  memoryContext?: string
  /** Provider override */
  provider?: string
  /** Model override */
  model?: string
  /** Max tokens override */
  maxTokens?: number
  /** Temperature override */
  temperature?: number
}

/**
 * Response from AI SQL generation
 */
export interface AIGenerateSQLResponse {
  /** Generated SQL query */
  sql: string
  /** Explanation of the query */
  explanation?: string
  /** Confidence score (0-1) */
  confidence?: number
  /** Warnings about the query */
  warnings?: string[]
}

/**
 * Response from AI SQL fixing
 */
export interface AIFixSQLResponse {
  /** Fixed SQL query */
  sql: string
  /** Explanation of the fix */
  explanation?: string
  /** Confidence score (0-1) */
  confidence?: number
  /** Warnings about the fix */
  warnings?: string[]
}

/**
 * Generic AI message request
 */
export interface AIGenericMessageRequest {
  /** User prompt */
  prompt: string
  /** Additional context */
  context?: string
  /** System prompt */
  systemPrompt?: string
  /** Metadata to attach */
  metadata?: Record<string, string>
  /** AI session ID */
  sessionId?: AISessionId
  /** Memory context */
  memoryContext?: string
  /** Provider override */
  provider?: string
  /** Model override */
  model?: string
  /** Max tokens override */
  maxTokens?: number
  /** Temperature override */
  temperature?: number
}

/**
 * Generic AI message response
 */
export interface AIGenericMessageResponse {
  /** Response content */
  content: string
  /** Provider used */
  provider: string
  /** Model used */
  model: string
  /** Tokens consumed */
  tokensUsed: number
  /** Additional metadata */
  metadata?: Record<string, string>
}

/**
 * Validated and enhanced request ready for backend
 */
export interface AIBackendRequest {
  /** Enhanced prompt */
  prompt: string
  /** Connection ID */
  connectionId: string
  /** Full context string */
  context: string
  /** Provider */
  provider: string
  /** Model */
  model: string
  /** Max tokens */
  maxTokens: number
  /** Temperature */
  temperature: number
}

/**
 * Error types for AI operations
 */
export type AIErrorType =
  | 'disabled' // AI features are disabled
  | 'no-provider' // No provider configured
  | 'no-api-key' // Missing API key
  | 'no-model' // No model selected
  | 'connection-failed' // Provider connection failed
  | 'request-failed' // Request to provider failed
  | 'response-invalid' // Invalid response from provider
  | 'unknown' // Unknown error

/**
 * Structured AI error
 */
export interface AIError {
  /** Error type */
  type: AIErrorType
  /** Error message */
  message: string
  /** Original error (if applicable) */
  originalError?: Error
  /** Additional context */
  context?: Record<string, unknown>
}

/**
 * AI memory recall item
 */
export interface AIRecallItem {
  /** Session title */
  title: string
  /** Session summary */
  summary?: string
  /** Memory content */
  content: string
}
