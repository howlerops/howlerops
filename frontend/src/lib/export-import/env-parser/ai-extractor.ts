/**
 * AI-Powered Connection Extraction
 *
 * Uses the app's AI capabilities to intelligently extract
 * database connection details from environment variables.
 *
 * @module lib/export-import/env-parser/ai-extractor
 */

import { v4 as uuidv4 } from 'uuid'
import type {
  AIExtractedConnection,
  AIExtractionResponse,
  EnvConnectionExtractionResult,
  EnvEntry,
  ExtractionConfidence,
  FieldSource,
  ParsedEnvConnection,
} from './types'
import { EnvImportError } from './types'
import type { DatabaseTypeString } from '../types'
import {
  buildExtractionPrompt,
  buildFallbackExtractionPrompt,
  ENV_EXTRACTION_SYSTEM_PROMPT,
  maskSensitiveEntries,
  validateAIResponse,
} from './prompts'

/**
 * Configuration for AI extraction
 */
export interface AIExtractorConfig {
  /** Function to send message to AI */
  sendMessage: (prompt: string, options?: {
    systemPrompt?: string
    context?: string
  }) => Promise<string>

  /** Optional: Focus on specific database type */
  focusType?: string

  /** Optional: Custom system prompt override */
  systemPromptOverride?: string
}

/**
 * Extract database connections from environment variables using AI
 *
 * @param entries - Filtered connection-related entries
 * @param config - AI configuration with sendMessage function
 * @returns Extraction result with parsed connections
 */
export async function extractConnectionsWithAI(
  entries: EnvEntry[],
  config: AIExtractorConfig
): Promise<EnvConnectionExtractionResult> {
  const startTime = Date.now()

  if (entries.length === 0) {
    return {
      connections: [],
      unusedEntries: [],
      aiConfidence: 'low',
      processingTime: Date.now() - startTime,
    }
  }

  // Mask sensitive values for AI prompt
  const maskedEntries = maskSensitiveEntries(entries)

  // Build the prompt
  const userPrompt = buildExtractionPrompt(entries, maskedEntries)
  const systemPrompt = config.systemPromptOverride || ENV_EXTRACTION_SYSTEM_PROMPT

  let aiResponse: string

  try {
    aiResponse = await config.sendMessage(userPrompt, {
      systemPrompt,
      context: 'Extracting database connections from .env file',
    })
  } catch (error) {
    // Try fallback prompt if first attempt fails
    try {
      const fallbackPrompt = buildFallbackExtractionPrompt(entries)
      aiResponse = await config.sendMessage(fallbackPrompt, {
        systemPrompt,
        context: 'Extracting database connections (fallback)',
      })
    } catch (fallbackError) {
      throw new EnvImportError(
        'AI_EXTRACTION_FAILED',
        'AI could not process the environment variables',
        { originalError: String(error), fallbackError: String(fallbackError) }
      )
    }
  }

  // Parse AI response
  const parsedResponse = parseAIResponseToJSON(aiResponse)
  const validationError = validateAIResponse(parsedResponse)

  if (validationError) {
    throw new EnvImportError(
      'AI_EXTRACTION_FAILED',
      `Invalid AI response: ${validationError}`,
      { response: aiResponse }
    )
  }

  const typedResponse = parsedResponse as AIExtractionResponse

  // Convert AI response to ParsedEnvConnection objects
  const connections = typedResponse.connections.map(conn =>
    convertAIConnectionToParsed(conn, entries)
  )

  // Find unused entries
  const usedKeys = new Set<string>()
  for (const conn of typedResponse.connections) {
    collectUsedKeys(conn, usedKeys)
  }

  const unusedEntries = entries.filter(e => !usedKeys.has(e.key))

  // Calculate overall confidence
  const aiConfidence = calculateOverallConfidence(connections)

  return {
    connections,
    unusedEntries,
    aiConfidence,
    processingTime: Date.now() - startTime,
  }
}

/**
 * Parse AI response string to JSON
 */
function parseAIResponseToJSON(response: string): unknown {
  // Try to extract JSON from response (AI might include markdown)
  let jsonStr = response.trim()

  // Remove markdown code fences if present
  const jsonMatch = response.match(/```(?:json)?\s*([\s\S]*?)```/)
  if (jsonMatch) {
    jsonStr = jsonMatch[1].trim()
  }

  try {
    return JSON.parse(jsonStr)
  } catch {
    // Try to find JSON object in response
    const objectMatch = response.match(/\{[\s\S]*\}/)
    if (objectMatch) {
      try {
        return JSON.parse(objectMatch[0])
      } catch {
        throw new EnvImportError(
          'AI_EXTRACTION_FAILED',
          'Could not parse JSON from AI response',
          { response }
        )
      }
    }
    throw new EnvImportError(
      'AI_EXTRACTION_FAILED',
      'AI response did not contain valid JSON',
      { response }
    )
  }
}

/**
 * Convert AI-extracted connection to ParsedEnvConnection
 */
function convertAIConnectionToParsed(
  aiConn: AIExtractedConnection,
  entries: EnvEntry[]
): ParsedEnvConnection {
  const entryMap = new Map(entries.map(e => [e.key, e]))

  // Helper to create FieldSource from key
  const createSource = (
    key: string | undefined,
    confidence: ExtractionConfidence
  ): FieldSource | undefined => {
    if (!key) return undefined
    const entry = entryMap.get(key)
    if (!entry) return undefined
    return {
      envKey: key,
      lineNumber: entry.lineNumber,
      confidence,
    }
  }

  const typeConfidence = normalizeConfidence(aiConn.typeConfidence)

  // Map database type string to our enum
  const dbType = mapDatabaseType(aiConn.type)

  const parsed: ParsedEnvConnection = {
    tempId: uuidv4(),
    suggestedName: aiConn.name,
    type: dbType,
    typeConfidence,
    overallConfidence: typeConfidence,
    isReviewed: false,
    isSkipped: false,
    validationErrors: [],
  }

  // Add optional fields with sources
  if (aiConn.host) {
    parsed.host = aiConn.host
    parsed.hostSource = createSource(aiConn.hostKey, typeConfidence)
  }

  if (aiConn.port !== undefined) {
    parsed.port = aiConn.port
    parsed.portSource = createSource(aiConn.portKey, typeConfidence)
  }

  if (aiConn.database) {
    parsed.database = aiConn.database
    parsed.databaseSource = createSource(aiConn.databaseKey, typeConfidence)
  }

  if (aiConn.username) {
    parsed.username = aiConn.username
    parsed.usernameSource = createSource(aiConn.usernameKey, typeConfidence)
  }

  if (aiConn.password) {
    parsed.password = aiConn.password
    parsed.passwordSource = createSource(aiConn.passwordKey, typeConfidence)
  }

  if (aiConn.sslMode) {
    parsed.sslMode = aiConn.sslMode
    parsed.sslModeSource = createSource(aiConn.sslModeKey, typeConfidence)
  }

  if (aiConn.connectionString) {
    parsed.connectionString = aiConn.connectionString
    parsed.connectionStringSource = createSource(
      aiConn.connectionStringKey,
      typeConfidence
    )
  }

  if (aiConn.notes) {
    parsed.extractionNotes = aiConn.notes
  }

  // Add detected environment
  if (aiConn.environment) {
    parsed.detectedEnvironment = normalizeEnvironment(aiConn.environment)
    parsed.environmentSource = 'ai-inference'
  }

  // Calculate overall confidence based on all fields
  parsed.overallConfidence = calculateConnectionConfidence(parsed)

  return parsed
}

/**
 * Normalize environment string to standard values
 */
function normalizeEnvironment(env: string): string {
  const lower = env.toLowerCase().trim()

  if (lower === 'dev' || lower === 'development') return 'development'
  if (lower === 'stg' || lower === 'staging') return 'staging'
  if (lower === 'prod' || lower === 'production') return 'production'
  if (lower === 'test' || lower === 'testing') return 'test'
  if (lower === 'local' || lower === 'localhost') return 'local'

  return lower
}

/**
 * Map AI database type string to our DatabaseTypeString enum
 */
function mapDatabaseType(type: string): DatabaseTypeString {
  const normalized = type.toLowerCase().trim()

  switch (normalized) {
    case 'postgresql':
    case 'postgres':
      return 'postgresql'
    case 'mysql':
      return 'mysql'
    case 'mariadb':
      return 'mariadb'
    case 'mongodb':
    case 'mongo':
      return 'mongodb'
    case 'redis':
      // Redis not yet supported - map to postgresql and let user correct
      return 'postgresql'
    case 'elasticsearch':
      return 'elasticsearch'
    case 'opensearch':
      return 'opensearch'
    case 'clickhouse':
      return 'clickhouse'
    case 'mssql':
    case 'sqlserver':
    case 'sql server':
      return 'mssql'
    case 'sqlite':
      return 'sqlite'
    default:
      // Default to postgresql if unknown
      return 'postgresql'
  }
}

/**
 * Normalize confidence string from AI
 */
function normalizeConfidence(confidence: string): ExtractionConfidence {
  const lower = confidence?.toLowerCase() || 'low'
  if (lower === 'high') return 'high'
  if (lower === 'medium') return 'medium'
  return 'low'
}

/**
 * Calculate overall confidence for a parsed connection
 */
function calculateConnectionConfidence(
  conn: ParsedEnvConnection
): ExtractionConfidence {
  const confidences: ExtractionConfidence[] = [conn.typeConfidence]

  if (conn.hostSource) confidences.push(conn.hostSource.confidence)
  if (conn.portSource) confidences.push(conn.portSource.confidence)
  if (conn.databaseSource) confidences.push(conn.databaseSource.confidence)
  if (conn.usernameSource) confidences.push(conn.usernameSource.confidence)
  if (conn.passwordSource) confidences.push(conn.passwordSource.confidence)
  if (conn.connectionStringSource) {
    confidences.push(conn.connectionStringSource.confidence)
  }

  // Return minimum confidence
  if (confidences.includes('low')) return 'low'
  if (confidences.includes('medium')) return 'medium'
  return 'high'
}

/**
 * Calculate overall confidence for extraction result
 */
function calculateOverallConfidence(
  connections: ParsedEnvConnection[]
): ExtractionConfidence {
  if (connections.length === 0) return 'low'

  const confidences = connections.map(c => c.overallConfidence)

  // Return minimum confidence
  if (confidences.includes('low')) return 'low'
  if (confidences.includes('medium')) return 'medium'
  return 'high'
}

/**
 * Collect all environment variable keys used by a connection
 */
function collectUsedKeys(conn: AIExtractedConnection, usedKeys: Set<string>): void {
  if (conn.hostKey) usedKeys.add(conn.hostKey)
  if (conn.portKey) usedKeys.add(conn.portKey)
  if (conn.databaseKey) usedKeys.add(conn.databaseKey)
  if (conn.usernameKey) usedKeys.add(conn.usernameKey)
  if (conn.passwordKey) usedKeys.add(conn.passwordKey)
  if (conn.sslModeKey) usedKeys.add(conn.sslModeKey)
  if (conn.connectionStringKey) usedKeys.add(conn.connectionStringKey)
}
