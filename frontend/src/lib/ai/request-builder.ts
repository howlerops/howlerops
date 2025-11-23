/**
 * AI Request Builder
 *
 * Builds properly formatted requests for the AI backend,
 * handling schema context, memory, and provider configuration.
 */

import type { SchemaNode } from '@/hooks/use-schema-introspection'
import { addMemoryContext, addRecallContext,buildSchemaContext, enhancePromptForMode } from '@/lib/ai-schema-context-builder'
import type { AIConfig } from '@/store/ai-store'
import type { DatabaseConnection } from '@/store/connection-store'
import type {
  AIBackendRequest,
  AIFixSQLRequest,
  AIGenerateSQLRequest,
  AIGenericMessageRequest,
  AIQueryMode,
} from '@/types/ai'

/**
 * Determines the effective connection ID from connections array
 */
export function getPrimaryConnectionId(connections?: DatabaseConnection[]): string {
  if (!connections || connections.length === 0) {
    return ''
  }

  // Prefer connected database with sessionId
  const primaryConn = connections.find(c => c.isConnected && c.sessionId) ||
                      connections.find(c => c.isConnected) ||
                      connections[0]

  return primaryConn?.sessionId || primaryConn?.id || ''
}

/**
 * Builds the full schema context for an AI request
 */
export function buildFullSchemaContext(options: {
  mode: AIQueryMode
  schema?: string
  connections?: DatabaseConnection[]
  schemasMap?: Map<string, SchemaNode[]>
  memoryContext?: string
  recallContext?: string
}): string {
  const { mode, schema, connections, schemasMap, memoryContext, recallContext } = options

  // Build base schema context
  let context = buildSchemaContext({
    mode,
    schema,
    connections,
    schemasMap,
  })

  // Add memory context if provided
  if (memoryContext) {
    context = addMemoryContext(context, memoryContext)
  }

  // Add recall context if provided
  if (recallContext) {
    context = addRecallContext(context, recallContext)
  }

  return context
}

/**
 * Builds a backend request for SQL generation
 */
export function buildGenerateSQLBackendRequest(
  request: AIGenerateSQLRequest,
  config: AIConfig
): AIBackendRequest {
  const mode = request.mode || 'single'
  const provider = (request.provider || config.provider || 'openai').toLowerCase()
  const model = request.model || config.selectedModel || 'gpt-4o-mini'

  // Enhance prompt for multi-DB mode
  const enhancedPrompt = enhancePromptForMode(
    request.prompt,
    mode,
    request.connections
  )

  // Build full context
  const context = buildFullSchemaContext({
    mode,
    schema: request.schema,
    connections: request.connections,
    schemasMap: request.schemasMap,
    memoryContext: request.memoryContext,
  })

  // Determine connection ID
  const connectionId = request.connectionId || getPrimaryConnectionId(request.connections)

  return {
    prompt: enhancedPrompt,
    connectionId,
    context,
    provider,
    model,
    maxTokens: request.maxTokens ?? config.maxTokens,
    temperature: request.temperature ?? config.temperature,
  }
}

/**
 * Builds a backend request for SQL fixing
 */
export function buildFixSQLBackendRequest(
  request: AIFixSQLRequest,
  config: AIConfig
): AIBackendRequest & { query: string; error: string } {
  const mode = request.mode || 'single'
  const provider = (request.provider || config.provider || 'openai').toLowerCase()
  const model = request.model || config.selectedModel || ''

  // Build full context
  const context = buildFullSchemaContext({
    mode,
    schema: request.schema,
    connections: request.connections,
    schemasMap: request.schemasMap,
    memoryContext: request.memoryContext,
  })

  // Enhance error message for multi-DB mode
  let enhancedError = request.error
  if (mode === 'multi' && request.connections && request.connections.length > 1) {
    const connectedNames = request.connections
      .filter(c => c.isConnected)
      .map(c => c.name)
      .join(', ')

    enhancedError = `${request.error}\n\nNote: This is a multi-database query. Tables should use @connection_name.table syntax. Available connections: ${connectedNames}`
  }

  // Determine connection ID
  const connectionId = request.connectionId || getPrimaryConnectionId(request.connections)

  return {
    query: request.query,
    error: enhancedError,
    prompt: `Fix this SQL query:\n${request.query}\n\nError:\n${enhancedError}`,
    connectionId,
    context,
    provider,
    model,
    maxTokens: request.maxTokens ?? config.maxTokens,
    temperature: request.temperature ?? config.temperature,
  }
}

/**
 * Builds a backend request for generic message
 */
export function buildGenericMessageBackendRequest(
  request: AIGenericMessageRequest,
  config: AIConfig
): {
  prompt: string
  context: string
  system?: string
  provider: string
  model: string
  maxTokens: number
  temperature: number
  metadata?: Record<string, string>
} {
  const provider = (request.provider || config.provider || 'openai').toLowerCase()
  const model = request.model || config.selectedModel

  // Combine base context with memory context
  let combinedContext = request.context?.trim() ?? ''
  if (request.memoryContext) {
    combinedContext = combinedContext
      ? `${combinedContext}\n\n---\n\nConversation Memory:\n${request.memoryContext}`
      : `Conversation Memory:\n${request.memoryContext}`
  }

  return {
    prompt: request.prompt,
    context: combinedContext,
    system: request.systemPrompt,
    provider,
    model,
    maxTokens: request.maxTokens ?? config.maxTokens,
    temperature: request.temperature ?? config.temperature,
    metadata: request.metadata,
  }
}
