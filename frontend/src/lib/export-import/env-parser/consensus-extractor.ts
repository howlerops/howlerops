/**
 * Multi-Agent Consensus Extraction System
 *
 * Implements a 3-tier agent architecture with Queen-based consensus for
 * high-confidence .env connection extraction.
 *
 * Architecture:
 * - Agent 1 (Pattern Matcher): Regex/heuristic-based extraction
 * - Agent 2 (AI Extractor): LLM-based extraction with enhanced prompts
 * - Agent 3 (Validator): Validates and scores both extractions
 * - Queen Evaluator: Merges, resolves conflicts, produces final extraction
 *
 * @module lib/export-import/env-parser/consensus-extractor
 */

import { v4 as uuidv4 } from 'uuid'
import type {
  AIExtractedConnection,
  EnvConnectionExtractionResult,
  EnvEntry,
  ExtractionConfidence,
  FieldSource,
  ParsedEnvConnection,
} from './types'
import { CONNECTION_RELATED_PATTERNS, EnvImportError } from './types'
import type { DatabaseTypeString } from '../types'
import {
  buildExtractionPrompt,
  buildFallbackExtractionPrompt,
  ENV_EXTRACTION_SYSTEM_PROMPT,
  maskSensitiveEntries,
  validateAIResponse,
} from './prompts'
import { groupEntriesByPrefix } from './parser'
import type { AIExtractorConfig } from './ai-extractor'

// =============================================================================
// Consensus Types
// =============================================================================

/**
 * Result from a single agent's extraction attempt
 */
export interface AgentExtractionResult {
  /** Agent identifier */
  agentId: 'pattern-matcher' | 'ai-extractor' | 'validator'

  /** Agent's display name */
  agentName: string

  /** Extracted connections from this agent */
  connections: AgentConnection[]

  /** Raw confidence score (0-1) for this agent's overall extraction */
  confidence: number

  /** Processing time in milliseconds */
  processingTime: number

  /** Any errors or warnings encountered */
  warnings: string[]

  /** Reasoning/notes about the extraction */
  reasoning: string
}

/**
 * A connection as extracted by an individual agent
 * Extends the base AIExtractedConnection with agent-specific metadata
 */
export interface AgentConnection extends AIExtractedConnection {
  /** Unique ID for tracking across agents */
  extractionId: string

  /** Numeric confidence score (0-1) */
  confidenceScore: number

  /** How this connection was identified */
  extractionMethod: 'regex' | 'prefix-grouping' | 'url-parsing' | 'ai-inference' | 'validation'

  /** Keys used to build this connection */
  sourceKeys: string[]

  /** Validation status from validator agent */
  validationStatus?: 'valid' | 'partial' | 'invalid'

  /** Validation notes */
  validationNotes?: string[]
}

/**
 * Agreement score between two agents on a specific connection
 */
export interface AgentAgreement {
  /** ID of first agent's connection */
  connectionId1: string

  /** ID of second agent's connection */
  connectionId2: string

  /** Agents being compared */
  agents: [string, string]

  /** Overall agreement score (0-1) */
  agreementScore: number

  /** Field-level agreement details */
  fieldAgreements: FieldAgreement[]

  /** Conflicts that need resolution */
  conflicts: FieldConflict[]
}

/**
 * Agreement status for a single field
 */
export interface FieldAgreement {
  field: string
  agent1Value: unknown
  agent2Value: unknown
  match: boolean
  similarity: number // 0-1 for partial matches
}

/**
 * A conflict between agent extractions that needs resolution
 */
export interface FieldConflict {
  field: string
  values: { agentId: string; value: unknown; confidence: number }[]
  resolution?: unknown
  resolutionReason?: string
}

/**
 * Queen's evaluation of all agent results
 */
export interface QueenEvaluation {
  /** Final merged connections */
  mergedConnections: ParsedEnvConnection[]

  /** Agreements found between agents */
  agreements: AgentAgreement[]

  /** Conflicts that were resolved */
  resolvedConflicts: FieldConflict[]

  /** Overall confidence in the consensus */
  consensusConfidence: ExtractionConfidence

  /** Numeric consensus score (0-1) */
  consensusScore: number

  /** Queen's reasoning for the final result */
  reasoning: string

  /** Per-connection agreement scores */
  connectionAgreementScores: Map<string, number>
}

/**
 * Full result with consensus metadata
 */
export interface ConsensusExtractionResult extends EnvConnectionExtractionResult {
  /** Individual agent results */
  agentResults: AgentExtractionResult[]

  /** Queen's evaluation */
  queenEvaluation: QueenEvaluation

  /** Per-connection agreement scores (tempId -> score) */
  agreementScores: Record<string, number>

  /** Whether consensus was reached */
  consensusReached: boolean

  /** Connections where agents disagreed significantly */
  lowAgreementConnections: string[]
}

// =============================================================================
// Agent 1: Pattern Matcher
// =============================================================================

/**
 * Pattern Matcher Agent
 * Uses regex and heuristics to extract connections without AI
 */
async function runPatternMatcherAgent(
  entries: EnvEntry[]
): Promise<AgentExtractionResult> {
  const startTime = Date.now()
  const connections: AgentConnection[] = []
  const warnings: string[] = []
  const usedKeys = new Set<string>()

  // Strategy 1: Find connection strings (URLs)
  const urlConnections = extractFromConnectionStrings(entries, usedKeys)
  connections.push(...urlConnections)

  // Strategy 2: Group by prefix and build connections
  const prefixConnections = extractFromPrefixGroups(entries, usedKeys)
  connections.push(...prefixConnections)

  // Strategy 3: Find standalone database variables
  const standaloneConnections = extractStandaloneVariables(entries, usedKeys)
  connections.push(...standaloneConnections)

  // Calculate overall confidence
  const avgConfidence =
    connections.length > 0
      ? connections.reduce((sum, c) => sum + c.confidenceScore, 0) / connections.length
      : 0

  return {
    agentId: 'pattern-matcher',
    agentName: 'Pattern Matcher Agent',
    connections,
    confidence: avgConfidence,
    processingTime: Date.now() - startTime,
    warnings,
    reasoning: `Extracted ${connections.length} connections using regex patterns, ` +
      `prefix grouping, and URL parsing. Used ${usedKeys.size} of ${entries.length} entries.`,
  }
}

/**
 * Extract connections from URL/connection string patterns
 */
function extractFromConnectionStrings(
  entries: EnvEntry[],
  usedKeys: Set<string>
): AgentConnection[] {
  const connections: AgentConnection[] = []

  // Connection string patterns with database type detection
  // Note: redis not yet supported, will be skipped
  const urlPatterns: { pattern: RegExp; dbType: DatabaseTypeString }[] = [
    { pattern: /^postgres(?:ql)?:\/\//i, dbType: 'postgresql' },
    { pattern: /^mysql:\/\//i, dbType: 'mysql' },
    { pattern: /^mariadb:\/\//i, dbType: 'mariadb' },
    { pattern: /^mongodb(?:\+srv)?:\/\//i, dbType: 'mongodb' },
    // Redis not yet supported - skip redis:// URLs
    { pattern: /^clickhouse:\/\//i, dbType: 'clickhouse' },
    { pattern: /^ch:\/\//i, dbType: 'clickhouse' },
    { pattern: /^sqlserver:\/\//i, dbType: 'mssql' },
    { pattern: /^mssql:\/\//i, dbType: 'mssql' },
  ]

  for (const entry of entries) {
    if (usedKeys.has(entry.key)) continue

    // Check if value is a connection string
    for (const { pattern, dbType } of urlPatterns) {
      if (pattern.test(entry.value)) {
        const parsed = parseConnectionStringValue(entry.value, dbType)
        if (parsed) {
          const environment = detectEnvironmentFromKey(entry.key)
          connections.push({
            extractionId: uuidv4(),
            name: generateConnectionName(dbType, environment, entry.key),
            type: dbType,
            typeConfidence: 'high',
            confidenceScore: 0.9,
            extractionMethod: 'url-parsing',
            sourceKeys: [entry.key],
            environment,
            connectionString: entry.value,
            connectionStringKey: entry.key,
            ...parsed,
            notes: `Parsed from connection URL in ${entry.key}`,
          })
          usedKeys.add(entry.key)
          break
        }
      }
    }
  }

  return connections
}

/**
 * Parse a connection string URL into components
 */
function parseConnectionStringValue(
  url: string,
  dbType: DatabaseTypeString
): Partial<AIExtractedConnection> | null {
  try {
    // Handle mongodb+srv specially
    if (url.startsWith('mongodb+srv://')) {
      const match = url.match(/mongodb\+srv:\/\/([^:]+):([^@]+)@([^/?]+)(?:\/([^?]+))?/)
      if (match) {
        return {
          username: match[1],
          host: match[3],
          database: match[4] || undefined,
        }
      }
    }

    // Standard URL parsing
    const parsed = new URL(url)
    const result: Partial<AIExtractedConnection> = {}

    if (parsed.hostname) result.host = parsed.hostname
    if (parsed.port) result.port = parseInt(parsed.port, 10)
    if (parsed.username) result.username = decodeURIComponent(parsed.username)
    if (parsed.password) result.password = decodeURIComponent(parsed.password)

    // Extract database from path
    const pathDb = parsed.pathname.slice(1).split('/')[0]
    if (pathDb) result.database = pathDb

    // Extract SSL mode from query params
    const sslMode = parsed.searchParams.get('sslmode') ||
      parsed.searchParams.get('ssl') ||
      parsed.searchParams.get('tls')
    if (sslMode) result.sslMode = sslMode

    return result
  } catch {
    return null
  }
}

/**
 * Extract connections by grouping variables with common prefixes
 */
function extractFromPrefixGroups(
  entries: EnvEntry[],
  usedKeys: Set<string>
): AgentConnection[] {
  const connections: AgentConnection[] = []
  const groups = groupEntriesByPrefix(entries)

  // Database-specific prefixes mapped to types
  // Note: REDIS prefix not included as redis not yet supported
  const prefixToType: Record<string, DatabaseTypeString> = {
    POSTGRES: 'postgresql',
    PG: 'postgresql',
    MYSQL: 'mysql',
    MARIADB: 'mariadb',
    MONGO: 'mongodb',
    MONGODB: 'mongodb',
    ELASTIC: 'elasticsearch',
    ELASTICSEARCH: 'elasticsearch',
    OPENSEARCH: 'opensearch',
    CLICKHOUSE: 'clickhouse',
    CH: 'clickhouse',
    MSSQL: 'mssql',
    SQLSERVER: 'mssql',
    SQLITE: 'sqlite',
  }

  for (const [prefix, groupEntries] of Array.from(groups.entries())) {
    // Skip entries already used
    const unusedEntries = groupEntries.filter(e => !usedKeys.has(e.key))
    if (unusedEntries.length === 0) continue

    // Determine database type from prefix
    const dbType = prefixToType[prefix] || detectTypeFromEntries(unusedEntries)
    if (!dbType) continue

    // Build connection from group
    const conn = buildConnectionFromGroup(prefix, unusedEntries, dbType)
    if (conn) {
      connections.push(conn)
      unusedEntries.forEach(e => usedKeys.add(e.key))
    }
  }

  return connections
}

/**
 * Try to detect database type from entry values
 */
function detectTypeFromEntries(entries: EnvEntry[]): DatabaseTypeString | null {
  for (const entry of entries) {
    const value = entry.value.toLowerCase()
    if (value.includes('postgres')) return 'postgresql'
    if (value.includes('mysql')) return 'mysql'
    if (value.includes('mongo')) return 'mongodb'
    // Redis not yet supported - skip detection
    if (value.includes('elastic')) return 'elasticsearch'
  }
  return null
}

/**
 * Build a connection from a group of related entries
 */
function buildConnectionFromGroup(
  prefix: string,
  entries: EnvEntry[],
  dbType: DatabaseTypeString
): AgentConnection | null {
  const connection: Partial<AgentConnection> = {
    extractionId: uuidv4(),
    type: dbType,
    typeConfidence: 'medium',
    confidenceScore: 0.7,
    extractionMethod: 'prefix-grouping',
    sourceKeys: entries.map(e => e.key),
  }

  const entryMap = new Map(entries.map(e => [e.key.toLowerCase(), e]))

  // Find host
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.host.some(p => p.test(entry.key))) {
      connection.host = entry.value
      connection.hostKey = entry.key
      break
    }
  }

  // Find port
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.port.some(p => p.test(entry.key))) {
      const port = parseInt(entry.value, 10)
      if (!isNaN(port)) {
        connection.port = port
        connection.portKey = entry.key
      }
      break
    }
  }

  // Find database name
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.database.some(p => p.test(entry.key))) {
      connection.database = entry.value
      connection.databaseKey = entry.key
      break
    }
  }

  // Find username
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.username.some(p => p.test(entry.key))) {
      connection.username = entry.value
      connection.usernameKey = entry.key
      break
    }
  }

  // Find password
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.password.some(p => p.test(entry.key))) {
      connection.password = entry.value
      connection.passwordKey = entry.key
      break
    }
  }

  // Find SSL mode
  for (const entry of entries) {
    if (CONNECTION_RELATED_PATTERNS.ssl.some(p => p.test(entry.key))) {
      connection.sslMode = entry.value
      connection.sslModeKey = entry.key
      break
    }
  }

  // Must have at least host or connection string to be valid
  if (!connection.host && !connection.connectionString) {
    return null
  }

  // Detect environment and generate name
  const environment = entries
    .map(e => detectEnvironmentFromKey(e.key))
    .find(env => env !== undefined)

  connection.environment = environment
  connection.name = generateConnectionName(dbType, environment, prefix)
  connection.notes = `Built from prefix group: ${prefix}`

  return connection as AgentConnection
}

/**
 * Extract standalone database variables that don't fit patterns
 */
function extractStandaloneVariables(
  entries: EnvEntry[],
  usedKeys: Set<string>
): AgentConnection[] {
  const connections: AgentConnection[] = []

  // Look for DATABASE_URL or similar that might have been missed
  for (const entry of entries) {
    if (usedKeys.has(entry.key)) continue

    const keyLower = entry.key.toLowerCase()
    if (
      keyLower.includes('database_url') ||
      keyLower.includes('db_url') ||
      keyLower.endsWith('_uri') ||
      keyLower.endsWith('_dsn')
    ) {
      // Try to detect type from value
      const dbType = detectTypeFromUrl(entry.value)
      if (dbType) {
        const parsed = parseConnectionStringValue(entry.value, dbType)
        const environment = detectEnvironmentFromKey(entry.key)

        connections.push({
          extractionId: uuidv4(),
          name: generateConnectionName(dbType, environment, entry.key),
          type: dbType,
          typeConfidence: 'medium',
          confidenceScore: 0.6,
          extractionMethod: 'regex',
          sourceKeys: [entry.key],
          environment,
          connectionString: entry.value,
          connectionStringKey: entry.key,
          ...parsed,
          notes: `Standalone URL variable: ${entry.key}`,
        })
        usedKeys.add(entry.key)
      }
    }
  }

  return connections
}

/**
 * Detect database type from URL string
 */
function detectTypeFromUrl(url: string): DatabaseTypeString | null {
  const lower = url.toLowerCase()
  if (lower.includes('postgres')) return 'postgresql'
  if (lower.includes('mysql')) return 'mysql'
  if (lower.includes('mariadb')) return 'mariadb'
  if (lower.includes('mongo')) return 'mongodb'
  // Redis not yet supported - skip detection
  if (lower.includes('elastic')) return 'elasticsearch'
  if (lower.includes('clickhouse')) return 'clickhouse'
  if (lower.includes('sqlserver') || lower.includes('mssql')) return 'mssql'
  return null
}

/**
 * Detect environment from variable key prefix
 */
function detectEnvironmentFromKey(key: string): string | undefined {
  const keyLower = key.toLowerCase()
  if (keyLower.startsWith('dev_') || keyLower.startsWith('development_')) return 'development'
  if (keyLower.startsWith('stg_') || keyLower.startsWith('staging_')) return 'staging'
  if (keyLower.startsWith('prod_') || keyLower.startsWith('production_')) return 'production'
  if (keyLower.startsWith('test_')) return 'test'
  if (keyLower.startsWith('local_')) return 'local'
  return undefined
}

/**
 * Generate a descriptive connection name
 */
function generateConnectionName(
  dbType: DatabaseTypeString,
  environment: string | undefined,
  sourceKey: string
): string {
  const typeNames: Record<string, string> = {
    postgresql: 'PostgreSQL',
    mysql: 'MySQL',
    mariadb: 'MariaDB',
    mongodb: 'MongoDB',
    redis: 'Redis',
    elasticsearch: 'Elasticsearch',
    opensearch: 'OpenSearch',
    clickhouse: 'ClickHouse',
    mssql: 'SQL Server',
    sqlite: 'SQLite',
  }

  const typeName = typeNames[dbType] || dbType

  if (environment) {
    const envName = environment.charAt(0).toUpperCase() + environment.slice(1)
    return `${envName} ${typeName}`
  }

  // Try to extract meaningful name from source key
  const cleanKey = sourceKey
    .replace(/_URL$/i, '')
    .replace(/_URI$/i, '')
    .replace(/_DSN$/i, '')
    .replace(/^(DEV|STAGING|PROD|TEST|LOCAL)_/i, '')
    .replace(/_/g, ' ')
    .toLowerCase()

  if (cleanKey && cleanKey !== 'database' && cleanKey !== 'db') {
    return `${cleanKey.charAt(0).toUpperCase() + cleanKey.slice(1)} ${typeName}`
  }

  return typeName
}

// =============================================================================
// Agent 2: AI Extractor
// =============================================================================

/**
 * Enhanced AI prompt for consensus extraction
 */
const CONSENSUS_AI_SYSTEM_PROMPT = `${ENV_EXTRACTION_SYSTEM_PROMPT}

## Additional Instructions for Consensus Extraction

You are participating in a multi-agent consensus system. Your extraction will be compared with:
1. A pattern-matching agent (regex/heuristics)
2. A validation agent

To maximize accuracy:
1. Be explicit about your confidence - use "high" only when patterns are unambiguous
2. Include detailed notes explaining your reasoning for each connection
3. If a variable could belong to multiple connections, note the ambiguity
4. Flag any unusual patterns or potential data quality issues
5. Cross-reference related variables to validate your extractions

When you see variables like DB_HOST alongside POSTGRES_HOST, explain which you chose and why.`

/**
 * AI Extractor Agent
 * Uses LLM with enhanced prompts for intelligent extraction
 */
async function runAIExtractorAgent(
  entries: EnvEntry[],
  config: AIExtractorConfig
): Promise<AgentExtractionResult> {
  const startTime = Date.now()
  const warnings: string[] = []

  if (entries.length === 0) {
    return {
      agentId: 'ai-extractor',
      agentName: 'AI Extractor Agent',
      connections: [],
      confidence: 0,
      processingTime: Date.now() - startTime,
      warnings: ['No entries to process'],
      reasoning: 'No environment variables provided for analysis.',
    }
  }

  // Mask sensitive values for AI prompt
  const maskedEntries = maskSensitiveEntries(entries)
  const userPrompt = buildExtractionPrompt(entries, maskedEntries)
  const systemPrompt = config.systemPromptOverride || CONSENSUS_AI_SYSTEM_PROMPT

  let aiResponse: string

  try {
    aiResponse = await config.sendMessage(userPrompt, {
      systemPrompt,
      context: 'Multi-agent consensus extraction from .env file',
    })
  } catch (error) {
    // Try fallback prompt
    try {
      const fallbackPrompt = buildFallbackExtractionPrompt(entries)
      aiResponse = await config.sendMessage(fallbackPrompt, {
        systemPrompt,
        context: 'Consensus extraction (fallback)',
      })
      warnings.push('Used fallback prompt due to initial extraction failure')
    } catch (fallbackError) {
      return {
        agentId: 'ai-extractor',
        agentName: 'AI Extractor Agent',
        connections: [],
        confidence: 0,
        processingTime: Date.now() - startTime,
        warnings: [`AI extraction failed: ${String(error)}`],
        reasoning: 'AI extraction failed, no connections extracted.',
      }
    }
  }

  // Parse AI response
  const parsedResponse = parseAIResponseToJSON(aiResponse)
  const validationError = validateAIResponse(parsedResponse)

  if (validationError) {
    return {
      agentId: 'ai-extractor',
      agentName: 'AI Extractor Agent',
      connections: [],
      confidence: 0,
      processingTime: Date.now() - startTime,
      warnings: [`Invalid AI response: ${validationError}`],
      reasoning: `AI response validation failed: ${validationError}`,
    }
  }

  const typedResponse = parsedResponse as { connections: AIExtractedConnection[]; unusedKeys?: string[] }

  // Convert to AgentConnection format
  const connections: AgentConnection[] = typedResponse.connections.map(conn => ({
    ...conn,
    extractionId: uuidv4(),
    confidenceScore: confidenceToScore(conn.typeConfidence),
    extractionMethod: 'ai-inference' as const,
    sourceKeys: collectSourceKeys(conn),
  }))

  const avgConfidence =
    connections.length > 0
      ? connections.reduce((sum, c) => sum + c.confidenceScore, 0) / connections.length
      : 0

  return {
    agentId: 'ai-extractor',
    agentName: 'AI Extractor Agent',
    connections,
    confidence: avgConfidence,
    processingTime: Date.now() - startTime,
    warnings,
    reasoning: `AI extracted ${connections.length} connections. ` +
      `Unused keys: ${typedResponse.unusedKeys?.join(', ') || 'none'}`,
  }
}

/**
 * Parse AI response string to JSON (duplicated from ai-extractor for isolation)
 */
function parseAIResponseToJSON(response: string): unknown {
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
        throw new Error('Could not parse JSON from AI response')
      }
    }
    throw new Error('AI response did not contain valid JSON')
  }
}

/**
 * Convert confidence string to numeric score
 */
function confidenceToScore(confidence: string): number {
  switch (confidence?.toLowerCase()) {
    case 'high':
      return 0.9
    case 'medium':
      return 0.7
    case 'low':
      return 0.4
    default:
      return 0.5
  }
}

/**
 * Collect all source keys from an AI connection
 */
function collectSourceKeys(conn: AIExtractedConnection): string[] {
  const keys: string[] = []
  if (conn.hostKey) keys.push(conn.hostKey)
  if (conn.portKey) keys.push(conn.portKey)
  if (conn.databaseKey) keys.push(conn.databaseKey)
  if (conn.usernameKey) keys.push(conn.usernameKey)
  if (conn.passwordKey) keys.push(conn.passwordKey)
  if (conn.sslModeKey) keys.push(conn.sslModeKey)
  if (conn.connectionStringKey) keys.push(conn.connectionStringKey)
  return keys
}

// =============================================================================
// Agent 3: Validator
// =============================================================================

/**
 * Validator Agent
 * Validates and scores extractions from other agents
 */
async function runValidatorAgent(
  entries: EnvEntry[],
  otherAgentResults: AgentExtractionResult[]
): Promise<AgentExtractionResult> {
  const startTime = Date.now()
  const warnings: string[] = []
  const validatedConnections: AgentConnection[] = []

  // Collect all connections from other agents
  const allConnections = otherAgentResults.flatMap(r => r.connections)

  // Create entry lookup map
  const entryMap = new Map(entries.map(e => [e.key, e]))

  for (const conn of allConnections) {
    const validated = validateConnection(conn, entryMap)
    validatedConnections.push(validated)

    if (validated.validationStatus === 'invalid') {
      warnings.push(`Connection "${conn.name}" has validation issues: ${validated.validationNotes?.join(', ')}`)
    }
  }

  // Calculate validation-based confidence
  const validCount = validatedConnections.filter(c => c.validationStatus === 'valid').length
  const partialCount = validatedConnections.filter(c => c.validationStatus === 'partial').length
  const totalCount = validatedConnections.length

  const confidence = totalCount > 0
    ? (validCount + partialCount * 0.5) / totalCount
    : 0

  return {
    agentId: 'validator',
    agentName: 'Validator Agent',
    connections: validatedConnections,
    confidence,
    processingTime: Date.now() - startTime,
    warnings,
    reasoning: `Validated ${totalCount} connections: ${validCount} valid, ` +
      `${partialCount} partial, ${totalCount - validCount - partialCount} invalid.`,
  }
}

/**
 * Validate a single connection against source entries
 */
function validateConnection(
  conn: AgentConnection,
  entryMap: Map<string, EnvEntry>
): AgentConnection {
  const notes: string[] = []
  let issueCount = 0

  // Validate referenced keys exist
  for (const key of conn.sourceKeys) {
    if (!entryMap.has(key)) {
      notes.push(`Referenced key "${key}" not found in entries`)
      issueCount++
    }
  }

  // Validate host key matches host value
  if (conn.hostKey && conn.host) {
    const entry = entryMap.get(conn.hostKey)
    if (entry && entry.value !== conn.host) {
      notes.push(`Host value mismatch: expected "${entry.value}", got "${conn.host}"`)
      issueCount++
    }
  }

  // Validate port is valid number
  if (conn.port !== undefined) {
    if (isNaN(conn.port) || conn.port < 1 || conn.port > 65535) {
      notes.push(`Invalid port number: ${conn.port}`)
      issueCount++
    }
  }

  // Validate database type is known
  const validTypes = [
    'postgresql', 'mysql', 'mariadb', 'mongodb', 'redis',
    'elasticsearch', 'opensearch', 'clickhouse', 'mssql', 'sqlite',
  ]
  if (!validTypes.includes(conn.type.toLowerCase())) {
    notes.push(`Unknown database type: ${conn.type}`)
    issueCount++
  }

  // Validate connection string format if present
  if (conn.connectionString) {
    try {
      // Check for valid URL-like format
      if (!conn.connectionString.includes('://')) {
        notes.push('Connection string missing protocol')
        issueCount++
      }
    } catch {
      notes.push('Invalid connection string format')
      issueCount++
    }
  }

  // Validate has minimum required fields
  if (!conn.host && !conn.connectionString) {
    notes.push('Missing both host and connection string')
    issueCount++
  }

  // Determine validation status
  let validationStatus: 'valid' | 'partial' | 'invalid'
  if (issueCount === 0) {
    validationStatus = 'valid'
  } else if (issueCount <= 2) {
    validationStatus = 'partial'
  } else {
    validationStatus = 'invalid'
  }

  // Adjust confidence based on validation
  let adjustedConfidence = conn.confidenceScore
  if (validationStatus === 'partial') {
    adjustedConfidence *= 0.8
  } else if (validationStatus === 'invalid') {
    adjustedConfidence *= 0.5
  }

  return {
    ...conn,
    confidenceScore: adjustedConfidence,
    extractionMethod: 'validation',
    validationStatus,
    validationNotes: notes.length > 0 ? notes : undefined,
  }
}

// =============================================================================
// Queen Evaluator
// =============================================================================

/**
 * Queen Evaluator
 * Merges agent results, resolves conflicts, produces final consensus
 */
function runQueenEvaluator(
  entries: EnvEntry[],
  agentResults: AgentExtractionResult[]
): QueenEvaluation {
  const patternResult = agentResults.find(r => r.agentId === 'pattern-matcher')
  const aiResult = agentResults.find(r => r.agentId === 'ai-extractor')
  const validatorResult = agentResults.find(r => r.agentId === 'validator')

  // Build connection matching matrix
  const agreements = computeAgreements(
    patternResult?.connections || [],
    aiResult?.connections || []
  )

  // Merge connections with conflict resolution
  const { mergedConnections, resolvedConflicts, connectionScores } = mergeConnections(
    patternResult?.connections || [],
    aiResult?.connections || [],
    validatorResult?.connections || [],
    agreements,
    entries
  )

  // Calculate consensus confidence
  const { consensusConfidence, consensusScore, reasoning } = evaluateConsensus(
    mergedConnections,
    agreements,
    agentResults
  )

  return {
    mergedConnections,
    agreements,
    resolvedConflicts,
    consensusConfidence,
    consensusScore,
    reasoning,
    connectionAgreementScores: connectionScores,
  }
}

/**
 * Compute agreements between pattern and AI agent connections
 */
function computeAgreements(
  patternConnections: AgentConnection[],
  aiConnections: AgentConnection[]
): AgentAgreement[] {
  const agreements: AgentAgreement[] = []

  for (const pConn of patternConnections) {
    for (const aConn of aiConnections) {
      // Check if connections likely refer to the same database
      const sourceOverlap = computeSourceKeyOverlap(pConn.sourceKeys, aConn.sourceKeys)
      const typeMatch = pConn.type.toLowerCase() === aConn.type.toLowerCase()

      if (sourceOverlap > 0 || (typeMatch && areHostsEqual(pConn, aConn))) {
        const fieldAgreements = computeFieldAgreements(pConn, aConn)
        const conflicts = findFieldConflicts(pConn, aConn, 'pattern-matcher', 'ai-extractor')

        const agreementScore = fieldAgreements.reduce(
          (sum, fa) => sum + fa.similarity,
          0
        ) / Math.max(fieldAgreements.length, 1)

        agreements.push({
          connectionId1: pConn.extractionId,
          connectionId2: aConn.extractionId,
          agents: ['pattern-matcher', 'ai-extractor'],
          agreementScore,
          fieldAgreements,
          conflicts,
        })
      }
    }
  }

  return agreements
}

/**
 * Compute overlap between source key sets
 */
function computeSourceKeyOverlap(keys1: string[], keys2: string[]): number {
  const set1 = new Set(keys1.map(k => k.toLowerCase()))
  const set2 = new Set(keys2.map(k => k.toLowerCase()))
  let overlap = 0
  Array.from(set1).forEach(key => {
    if (set2.has(key)) overlap++
  })
  return overlap
}

/**
 * Check if two connections have equivalent hosts
 */
function areHostsEqual(conn1: AgentConnection, conn2: AgentConnection): boolean {
  if (!conn1.host && !conn2.host) return true
  if (!conn1.host || !conn2.host) return false
  return conn1.host.toLowerCase() === conn2.host.toLowerCase()
}

/**
 * Compute field-level agreements between two connections
 */
function computeFieldAgreements(
  conn1: AgentConnection,
  conn2: AgentConnection
): FieldAgreement[] {
  const fields = ['type', 'host', 'port', 'database', 'username', 'sslMode', 'environment']
  const agreements: FieldAgreement[] = []

  for (const field of fields) {
    const val1 = getConnectionField(conn1, field)
    const val2 = getConnectionField(conn2, field)

    if (val1 === undefined && val2 === undefined) {
      continue // Skip if neither has the field
    }

    const match = val1 === val2 ||
      (typeof val1 === 'string' && typeof val2 === 'string' &&
        val1.toLowerCase() === val2.toLowerCase())

    let similarity = match ? 1 : 0
    if (!match && typeof val1 === 'string' && typeof val2 === 'string') {
      // Calculate string similarity for partial matches
      similarity = computeStringSimilarity(val1, val2)
    }

    agreements.push({
      field,
      agent1Value: val1,
      agent2Value: val2,
      match,
      similarity,
    })
  }

  return agreements
}

/**
 * Simple string similarity (Jaccard-like)
 */
function computeStringSimilarity(s1: string, s2: string): number {
  const set1 = new Set(s1.toLowerCase().split(''))
  const set2 = new Set(s2.toLowerCase().split(''))
  let intersection = 0
  Array.from(set1).forEach(char => {
    if (set2.has(char)) intersection++
  })
  const union = set1.size + set2.size - intersection
  return union > 0 ? intersection / union : 0
}

/**
 * Type-safe field accessor for AgentConnection
 */
function getConnectionField(conn: AgentConnection, field: string): unknown {
  switch (field) {
    case 'type': return conn.type
    case 'host': return conn.host
    case 'port': return conn.port
    case 'database': return conn.database
    case 'username': return conn.username
    case 'password': return conn.password
    case 'sslMode': return conn.sslMode
    case 'environment': return conn.environment
    case 'connectionString': return conn.connectionString
    default: return undefined
  }
}

/**
 * Find field-level conflicts between connections
 */
function findFieldConflicts(
  conn1: AgentConnection,
  conn2: AgentConnection,
  agent1Id: string,
  agent2Id: string
): FieldConflict[] {
  const fields = ['type', 'host', 'port', 'database', 'username', 'sslMode']
  const conflicts: FieldConflict[] = []

  for (const field of fields) {
    const val1 = getConnectionField(conn1, field)
    const val2 = getConnectionField(conn2, field)

    // Only conflict if both have different non-null values
    if (val1 !== undefined && val2 !== undefined && val1 !== val2) {
      const match = typeof val1 === 'string' && typeof val2 === 'string' &&
        val1.toLowerCase() === val2.toLowerCase()

      if (!match) {
        conflicts.push({
          field,
          values: [
            { agentId: agent1Id, value: val1, confidence: conn1.confidenceScore },
            { agentId: agent2Id, value: val2, confidence: conn2.confidenceScore },
          ],
        })
      }
    }
  }

  return conflicts
}

/**
 * Merge connections from all agents with conflict resolution
 */
function mergeConnections(
  patternConnections: AgentConnection[],
  aiConnections: AgentConnection[],
  validatorConnections: AgentConnection[],
  agreements: AgentAgreement[],
  entries: EnvEntry[]
): {
  mergedConnections: ParsedEnvConnection[]
  resolvedConflicts: FieldConflict[]
  connectionScores: Map<string, number>
} {
  const mergedConnections: ParsedEnvConnection[] = []
  const resolvedConflicts: FieldConflict[] = []
  const connectionScores = new Map<string, number>()
  const processedAiIds = new Set<string>()

  // Get validated connection map
  const validatedMap = new Map(
    validatorConnections.map(c => [c.extractionId, c])
  )

  // Process agreements first (matched connections)
  for (const agreement of agreements) {
    const patternConn = patternConnections.find(c => c.extractionId === agreement.connectionId1)
    const aiConn = aiConnections.find(c => c.extractionId === agreement.connectionId2)

    if (!patternConn || !aiConn) continue

    processedAiIds.add(aiConn.extractionId)

    // Resolve conflicts
    for (const conflict of agreement.conflicts) {
      const resolved = resolveConflict(conflict, patternConn, aiConn, entries)
      resolvedConflicts.push(resolved)
    }

    // Merge the connections
    const merged = mergeMatchedConnections(
      patternConn,
      aiConn,
      validatedMap.get(patternConn.extractionId),
      validatedMap.get(aiConn.extractionId),
      resolvedConflicts.filter(c =>
        agreement.conflicts.some(ac => ac.field === c.field)
      )
    )

    mergedConnections.push(merged)
    connectionScores.set(merged.tempId, agreement.agreementScore)
  }

  // Add unmatched pattern connections
  for (const conn of patternConnections) {
    const wasMatched = agreements.some(a => a.connectionId1 === conn.extractionId)
    if (!wasMatched) {
      const validated = validatedMap.get(conn.extractionId)
      const merged = convertToFinalConnection(conn, validated)
      mergedConnections.push(merged)
      connectionScores.set(merged.tempId, conn.confidenceScore * 0.8) // Lower score for unmatched
    }
  }

  // Add unmatched AI connections
  for (const conn of aiConnections) {
    if (processedAiIds.has(conn.extractionId)) continue

    const validated = validatedMap.get(conn.extractionId)
    const merged = convertToFinalConnection(conn, validated)
    mergedConnections.push(merged)
    connectionScores.set(merged.tempId, conn.confidenceScore * 0.8) // Lower score for unmatched
  }

  return { mergedConnections, resolvedConflicts, connectionScores }
}

/**
 * Resolve a field conflict between agents
 */
function resolveConflict(
  conflict: FieldConflict,
  patternConn: AgentConnection,
  aiConn: AgentConnection,
  entries: EnvEntry[]
): FieldConflict {
  // Strategy: Prefer value with higher confidence, verified against entries
  const entryMap = new Map(entries.map(e => [e.key, e.value]))

  let bestValue: unknown
  let bestReason: string

  // Get the key for this field
  const fieldKey = `${conflict.field}Key` as keyof AgentConnection
  const patternKey = patternConn[fieldKey] as string | undefined
  const aiKey = aiConn[fieldKey] as string | undefined

  // Check if values match actual entry values
  const patternVal = conflict.values.find(v => v.agentId === 'pattern-matcher')
  const aiVal = conflict.values.find(v => v.agentId === 'ai-extractor')

  // Prefer value that matches source entry
  if (patternKey && entryMap.get(patternKey) === patternVal?.value) {
    bestValue = patternVal?.value
    bestReason = 'Pattern matcher value matches source entry'
  } else if (aiKey && entryMap.get(aiKey) === aiVal?.value) {
    bestValue = aiVal?.value
    bestReason = 'AI extractor value matches source entry'
  } else if ((patternVal?.confidence || 0) >= (aiVal?.confidence || 0)) {
    bestValue = patternVal?.value
    bestReason = 'Pattern matcher has higher confidence'
  } else {
    bestValue = aiVal?.value
    bestReason = 'AI extractor has higher confidence'
  }

  return {
    ...conflict,
    resolution: bestValue,
    resolutionReason: bestReason,
  }
}

/**
 * Merge two matched connections into final format
 */
function mergeMatchedConnections(
  patternConn: AgentConnection,
  aiConn: AgentConnection,
  patternValidated: AgentConnection | undefined,
  aiValidated: AgentConnection | undefined,
  resolvedConflicts: FieldConflict[]
): ParsedEnvConnection {
  const tempId = uuidv4()

  // Build conflict resolution map
  const resolutions = new Map(
    resolvedConflicts
      .filter(c => c.resolution !== undefined)
      .map(c => [c.field, c.resolution])
  )

  // Prefer AI name but use pattern if AI didn't provide one
  const name = aiConn.name || patternConn.name || 'Unnamed Connection'

  // Prefer validated type, otherwise use consensus
  const type = (resolutions.get('type') as DatabaseTypeString) ||
    aiConn.type || patternConn.type

  // Calculate merged confidence
  const avgConfidence = (patternConn.confidenceScore + aiConn.confidenceScore) / 2
  const validationBonus = (patternValidated?.validationStatus === 'valid' ||
    aiValidated?.validationStatus === 'valid') ? 0.1 : 0

  const overallConfidence = scoreToConfidence(avgConfidence + validationBonus)

  // Merge field sources (prefer AI's key references)
  const createSource = (
    key: string | undefined,
    conn: AgentConnection
  ): FieldSource | undefined => {
    if (!key) return undefined
    return {
      envKey: key,
      lineNumber: 0, // Would need entries to get line number
      confidence: scoreToConfidence(conn.confidenceScore),
    }
  }

  const result: ParsedEnvConnection = {
    tempId,
    suggestedName: name,
    type: type as DatabaseTypeString,
    typeConfidence: overallConfidence,
    overallConfidence,
    isReviewed: false,
    isSkipped: false,
    validationErrors: [],
  }

  // Merge fields with conflict resolution
  const host = resolutions.get('host') || aiConn.host || patternConn.host
  if (host) {
    result.host = host as string
    result.hostSource = createSource(aiConn.hostKey || patternConn.hostKey, aiConn)
  }

  const port = resolutions.get('port') || aiConn.port || patternConn.port
  if (port !== undefined) {
    result.port = port as number
    result.portSource = createSource(aiConn.portKey || patternConn.portKey, aiConn)
  }

  const database = resolutions.get('database') || aiConn.database || patternConn.database
  if (database) {
    result.database = database as string
    result.databaseSource = createSource(aiConn.databaseKey || patternConn.databaseKey, aiConn)
  }

  const username = resolutions.get('username') || aiConn.username || patternConn.username
  if (username) {
    result.username = username as string
    result.usernameSource = createSource(aiConn.usernameKey || patternConn.usernameKey, aiConn)
  }

  // Always prefer pattern matcher for password (has actual value)
  const password = patternConn.password || aiConn.password
  if (password) {
    result.password = password as string
    result.passwordSource = createSource(patternConn.passwordKey || aiConn.passwordKey, patternConn)
  }

  const sslMode = resolutions.get('sslMode') || aiConn.sslMode || patternConn.sslMode
  if (sslMode) {
    result.sslMode = sslMode as string
    result.sslModeSource = createSource(aiConn.sslModeKey || patternConn.sslModeKey, aiConn)
  }

  const connectionString = aiConn.connectionString || patternConn.connectionString
  if (connectionString) {
    result.connectionString = connectionString as string
    result.connectionStringSource = createSource(
      aiConn.connectionStringKey || patternConn.connectionStringKey,
      aiConn
    )
  }

  // Environment
  const environment = aiConn.environment || patternConn.environment
  if (environment) {
    result.detectedEnvironment = environment
    result.environmentSource = 'ai-inference'
  }

  // Notes
  const notes: string[] = []
  if (aiConn.notes) notes.push(`AI: ${aiConn.notes}`)
  if (patternConn.notes) notes.push(`Pattern: ${patternConn.notes}`)
  if (notes.length > 0) {
    result.extractionNotes = notes.join(' | ')
  }

  return result
}

/**
 * Convert single agent connection to final format
 */
function convertToFinalConnection(
  conn: AgentConnection,
  validated: AgentConnection | undefined
): ParsedEnvConnection {
  const tempId = uuidv4()

  const confidence = scoreToConfidence(
    validated?.confidenceScore || conn.confidenceScore
  )

  const createSource = (key: string | undefined): FieldSource | undefined => {
    if (!key) return undefined
    return {
      envKey: key,
      lineNumber: 0,
      confidence,
    }
  }

  const result: ParsedEnvConnection = {
    tempId,
    suggestedName: conn.name || 'Unnamed Connection',
    type: conn.type as DatabaseTypeString,
    typeConfidence: confidence,
    overallConfidence: confidence,
    isReviewed: false,
    isSkipped: false,
    validationErrors: validated?.validationNotes || [],
  }

  if (conn.host) {
    result.host = conn.host
    result.hostSource = createSource(conn.hostKey)
  }

  if (conn.port !== undefined) {
    result.port = conn.port
    result.portSource = createSource(conn.portKey)
  }

  if (conn.database) {
    result.database = conn.database
    result.databaseSource = createSource(conn.databaseKey)
  }

  if (conn.username) {
    result.username = conn.username
    result.usernameSource = createSource(conn.usernameKey)
  }

  if (conn.password) {
    result.password = conn.password
    result.passwordSource = createSource(conn.passwordKey)
  }

  if (conn.sslMode) {
    result.sslMode = conn.sslMode
    result.sslModeSource = createSource(conn.sslModeKey)
  }

  if (conn.connectionString) {
    result.connectionString = conn.connectionString
    result.connectionStringSource = createSource(conn.connectionStringKey)
  }

  if (conn.environment) {
    result.detectedEnvironment = conn.environment
    result.environmentSource = 'ai-inference'
  }

  if (conn.notes) {
    result.extractionNotes = conn.notes
  }

  return result
}

/**
 * Convert numeric score to confidence level
 */
function scoreToConfidence(score: number): ExtractionConfidence {
  if (score >= 0.8) return 'high'
  if (score >= 0.5) return 'medium'
  return 'low'
}

/**
 * Evaluate overall consensus quality
 */
function evaluateConsensus(
  connections: ParsedEnvConnection[],
  agreements: AgentAgreement[],
  agentResults: AgentExtractionResult[]
): {
  consensusConfidence: ExtractionConfidence
  consensusScore: number
  reasoning: string
} {
  if (connections.length === 0) {
    return {
      consensusConfidence: 'low',
      consensusScore: 0,
      reasoning: 'No connections were extracted by any agent.',
    }
  }

  // Calculate average agreement score
  const avgAgreement = agreements.length > 0
    ? agreements.reduce((sum, a) => sum + a.agreementScore, 0) / agreements.length
    : 0

  // Calculate agent confidence agreement
  const agentConfidences = agentResults.map(r => r.confidence)
  const avgAgentConfidence = agentConfidences.reduce((a, b) => a + b, 0) / agentConfidences.length

  // Calculate connection confidence
  const connectionConfidences = connections.map(c =>
    c.overallConfidence === 'high' ? 1 : c.overallConfidence === 'medium' ? 0.7 : 0.4
  )
  const avgConnectionConfidence = connectionConfidences.reduce((a, b) => a + b, 0) / connectionConfidences.length

  // Weighted consensus score
  const consensusScore = (
    avgAgreement * 0.4 +
    avgAgentConfidence * 0.3 +
    avgConnectionConfidence * 0.3
  )

  const consensusConfidence = scoreToConfidence(consensusScore)

  // Build reasoning
  const matchedCount = agreements.length
  const totalConnections = connections.length
  const patternCount = agentResults.find(r => r.agentId === 'pattern-matcher')?.connections.length || 0
  const aiCount = agentResults.find(r => r.agentId === 'ai-extractor')?.connections.length || 0

  const reasoning = [
    `Consensus reached on ${totalConnections} connection(s).`,
    `Pattern matcher found ${patternCount}, AI found ${aiCount}.`,
    `${matchedCount} connection(s) matched between agents with ${(avgAgreement * 100).toFixed(0)}% average agreement.`,
    `Overall consensus confidence: ${(consensusScore * 100).toFixed(0)}%.`,
  ].join(' ')

  return { consensusConfidence, consensusScore, reasoning }
}

// =============================================================================
// Main Export Function
// =============================================================================

/**
 * Extract database connections using multi-agent consensus
 *
 * This function orchestrates three agents:
 * 1. Pattern Matcher: Fast regex/heuristic extraction
 * 2. AI Extractor: LLM-based intelligent extraction
 * 3. Validator: Validates and scores extractions
 *
 * A Queen evaluator then merges results, resolves conflicts,
 * and produces a high-confidence final extraction.
 *
 * @param entries - Filtered connection-related entries
 * @param config - AI configuration with sendMessage function
 * @returns Consensus extraction result with agreement scores
 */
export async function extractConnectionsWithConsensus(
  entries: EnvEntry[],
  config: AIExtractorConfig
): Promise<ConsensusExtractionResult> {
  const startTime = Date.now()

  if (entries.length === 0) {
    return {
      connections: [],
      unusedEntries: [],
      aiConfidence: 'low',
      processingTime: Date.now() - startTime,
      agentResults: [],
      queenEvaluation: {
        mergedConnections: [],
        agreements: [],
        resolvedConflicts: [],
        consensusConfidence: 'low',
        consensusScore: 0,
        reasoning: 'No entries to process.',
        connectionAgreementScores: new Map(),
      },
      agreementScores: {},
      consensusReached: false,
      lowAgreementConnections: [],
    }
  }

  // Phase 1: Run Pattern Matcher and AI Extractor in parallel
  const [patternResult, aiResult] = await Promise.all([
    runPatternMatcherAgent(entries),
    runAIExtractorAgent(entries, config),
  ])

  // Phase 2: Run Validator on results
  const validatorResult = await runValidatorAgent(entries, [patternResult, aiResult])

  const agentResults = [patternResult, aiResult, validatorResult]

  // Phase 3: Queen evaluates and merges
  const queenEvaluation = runQueenEvaluator(entries, agentResults)

  // Calculate unused entries
  const usedKeys = new Set<string>()
  for (const conn of queenEvaluation.mergedConnections) {
    if (conn.hostSource?.envKey) usedKeys.add(conn.hostSource.envKey)
    if (conn.portSource?.envKey) usedKeys.add(conn.portSource.envKey)
    if (conn.databaseSource?.envKey) usedKeys.add(conn.databaseSource.envKey)
    if (conn.usernameSource?.envKey) usedKeys.add(conn.usernameSource.envKey)
    if (conn.passwordSource?.envKey) usedKeys.add(conn.passwordSource.envKey)
    if (conn.sslModeSource?.envKey) usedKeys.add(conn.sslModeSource.envKey)
    if (conn.connectionStringSource?.envKey) usedKeys.add(conn.connectionStringSource.envKey)
  }

  const unusedEntries = entries.filter(e => !usedKeys.has(e.key))

  // Convert agreement scores map to record
  const agreementScores: Record<string, number> = {}
  Array.from(queenEvaluation.connectionAgreementScores.entries()).forEach(([tempId, score]) => {
    agreementScores[tempId] = score
  })

  // Find low agreement connections
  const lowAgreementConnections = Object.entries(agreementScores)
    .filter(([_, score]) => score < 0.5)
    .map(([tempId]) => tempId)

  // Determine if consensus was reached
  const consensusReached = queenEvaluation.consensusScore >= 0.5

  return {
    connections: queenEvaluation.mergedConnections,
    unusedEntries,
    aiConfidence: queenEvaluation.consensusConfidence,
    processingTime: Date.now() - startTime,
    agentResults,
    queenEvaluation,
    agreementScores,
    consensusReached,
    lowAgreementConnections,
  }
}

// =============================================================================
// Utility Exports
// =============================================================================

export {
  runPatternMatcherAgent,
  runAIExtractorAgent,
  runValidatorAgent,
  runQueenEvaluator,
}
