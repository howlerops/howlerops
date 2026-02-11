/**
 * .env File Parser
 *
 * Parses .env file content into structured entries.
 * Handles various .env formats including quoted values, multiline, and comments.
 *
 * @module lib/export-import/env-parser/parser
 */

import type {
  EnvEntry,
  EnvParseError,
  EnvParseResult,
  EnvEntryExtended,
  EnvParseResultExtended,
  CommentedConnectionGroup,
} from './types'
import {
  CONNECTION_RELATED_PATTERNS,
  EnvImportError,
  MAX_ENV_FILE_SIZE,
} from './types'

// =============================================================================
// Comment Parsing Patterns
// =============================================================================

/**
 * Patterns for detecting environment header comments
 */
const ENV_HEADER_PATTERNS = [
  /^#\s*(production|prod)\s*$/i,
  /^#\s*(staging|stg)\s*$/i,
  /^#\s*(development|dev)\s*$/i,
  /^#\s*(test|testing)\s*$/i,
  /^#\s*(local|localhost)\s*$/i,
  /^#\s*-+\s*(production|staging|development|test|local)\s*-*\s*$/i,
  /^#\s*(production|staging|development|test|local)\s*\(.*\)\s*$/i,
]

/**
 * Pattern for commented-out key=value
 */
const COMMENTED_KV_PATTERN = /^#+\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$/

/**
 * Detect if comment is an environment header
 */
function detectEnvironmentHeader(comment: string): string | null {
  for (const pattern of ENV_HEADER_PATTERNS) {
    const match = comment.match(pattern)
    if (match) {
      const env = match[1].toLowerCase()
      if (env === 'prod') return 'production'
      if (env === 'stg') return 'staging'
      if (env === 'dev') return 'development'
      return env
    }
  }
  return null
}

/**
 * Parse a .env file content into structured entries
 *
 * @param content - Raw .env file content
 * @returns Parsed result with entries, errors, and metadata
 */
export function parseEnvFile(content: string): EnvParseResult {
  const lines = content.split(/\r?\n/)
  const entries: EnvEntry[] = []
  const errors: EnvParseError[] = []
  let skippedLines = 0

  for (let i = 0; i < lines.length; i++) {
    const lineNumber = i + 1
    const raw = lines[i]
    const trimmed = raw.trim()

    // Skip empty lines
    if (!trimmed) {
      skippedLines++
      continue
    }

    // Skip comments
    if (trimmed.startsWith('#')) {
      skippedLines++
      continue
    }

    // Skip export prefix if present (common in shell scripts)
    const lineContent = trimmed.replace(/^export\s+/, '')

    // Parse key=value
    const equalsIndex = lineContent.indexOf('=')
    if (equalsIndex === -1) {
      errors.push({
        lineNumber,
        line: raw,
        message: 'Invalid format: missing "=" separator',
      })
      continue
    }

    const key = lineContent.slice(0, equalsIndex).trim()
    let value = lineContent.slice(equalsIndex + 1)

    // Validate key format
    if (!isValidEnvKey(key)) {
      errors.push({
        lineNumber,
        line: raw,
        message: `Invalid key format: "${key}"`,
      })
      continue
    }

    // Handle quoted values
    value = parseQuotedValue(value)

    entries.push({
      key,
      value,
      lineNumber,
      raw,
    })
  }

  return {
    entries,
    errors,
    metadata: {
      totalLines: lines.length,
      validEntries: entries.length,
      skippedLines,
    },
  }
}

/**
 * Check if a key is a valid environment variable name
 */
function isValidEnvKey(key: string): boolean {
  // Must start with letter or underscore, contain only alphanumeric and underscore
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(key)
}

/**
 * Parse a potentially quoted value
 */
function parseQuotedValue(value: string): string {
  const trimmed = value.trim()

  // Handle double-quoted strings
  if (trimmed.startsWith('"') && trimmed.endsWith('"')) {
    return trimmed
      .slice(1, -1)
      .replace(/\\n/g, '\n')
      .replace(/\\r/g, '\r')
      .replace(/\\t/g, '\t')
      .replace(/\\"/g, '"')
      .replace(/\\\\/g, '\\')
  }

  // Handle single-quoted strings (literal, no escape processing)
  if (trimmed.startsWith("'") && trimmed.endsWith("'")) {
    return trimmed.slice(1, -1)
  }

  // Unquoted value - strip inline comments
  const commentIndex = trimmed.indexOf(' #')
  if (commentIndex !== -1) {
    return trimmed.slice(0, commentIndex).trim()
  }

  return trimmed
}

/**
 * Filter entries to only those related to database connections
 *
 * @param entries - All parsed entries
 * @returns Entries that match connection-related patterns
 */
export function filterConnectionRelatedEntries(entries: EnvEntry[]): EnvEntry[] {
  const allPatterns = [
    ...CONNECTION_RELATED_PATTERNS.url,
    ...CONNECTION_RELATED_PATTERNS.host,
    ...CONNECTION_RELATED_PATTERNS.port,
    ...CONNECTION_RELATED_PATTERNS.database,
    ...CONNECTION_RELATED_PATTERNS.username,
    ...CONNECTION_RELATED_PATTERNS.password,
    ...CONNECTION_RELATED_PATTERNS.ssl,
    ...CONNECTION_RELATED_PATTERNS.databasePrefixes,
  ]

  return entries.filter(entry =>
    allPatterns.some(pattern => pattern.test(entry.key))
  )
}

/**
 * Read a .env file from a File object
 *
 * @param file - File object from file input or drag-drop
 * @returns File content as string
 * @throws EnvImportError if file is too large or cannot be read
 */
export async function readEnvFile(file: File): Promise<string> {
  // Check file size
  if (file.size > MAX_ENV_FILE_SIZE) {
    throw new EnvImportError(
      'FILE_TOO_LARGE',
      `File size (${Math.round(file.size / 1024)}KB) exceeds maximum allowed (${Math.round(MAX_ENV_FILE_SIZE / 1024)}KB)`,
      { fileSize: file.size, maxSize: MAX_ENV_FILE_SIZE }
    )
  }

  return new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
      } else {
        reject(
          new EnvImportError(
            'FILE_READ_ERROR',
            'Failed to read file as text'
          )
        )
      }
    }

    reader.onerror = () => {
      reject(
        new EnvImportError(
          'FILE_READ_ERROR',
          'Error reading file',
          { error: reader.error?.message }
        )
      )
    }

    reader.readAsText(file)
  })
}

/**
 * Validate that file has an accepted .env extension
 */
export function isAcceptedEnvFile(filename: string): boolean {
  const lowerName = filename.toLowerCase()

  // Check exact matches
  if (lowerName === '.env') return true

  // Check if ends with common .env patterns
  if (lowerName.endsWith('.env')) return true
  if (lowerName.includes('.env.')) return true

  return false
}

/**
 * Group entries by common prefix to help identify related variables
 *
 * @param entries - Filtered connection-related entries
 * @returns Map of prefix to entries with that prefix
 */
export function groupEntriesByPrefix(entries: EnvEntry[]): Map<string, EnvEntry[]> {
  const groups = new Map<string, EnvEntry[]>()

  for (const entry of entries) {
    // Extract prefix (e.g., "POSTGRES" from "POSTGRES_HOST")
    const match = entry.key.match(/^([A-Z]+(?:_[A-Z]+)?)_/)
    const prefix = match ? match[1] : 'OTHER'

    const existing = groups.get(prefix) || []
    existing.push(entry)
    groups.set(prefix, existing)
  }

  return groups
}

// =============================================================================
// Extended Parser with Comment Support
// =============================================================================

/**
 * Parse a .env file including commented-out entries
 *
 * Extracts both active and commented variables, grouping commented
 * entries by their environment context (e.g., # Production header).
 *
 * @param content - Raw .env file content
 * @returns Extended result with commented entries and groups
 */
export function parseEnvFileWithComments(content: string): EnvParseResultExtended {
  const lines = content.split(/\r?\n/)
  const entries: EnvEntry[] = []
  const commentedEntries: EnvEntryExtended[] = []
  const errors: EnvParseError[] = []
  let skippedLines = 0

  let currentCommentEnv: string | null = null
  let envHeaderLine: number | null = null

  for (let i = 0; i < lines.length; i++) {
    const lineNumber = i + 1
    const raw = lines[i]
    const trimmed = raw.trim()

    // Skip empty lines (also potentially reset comment context)
    if (!trimmed) {
      skippedLines++
      // Multiple blank lines reset context
      if (i > 0 && !lines[i - 1].trim()) {
        currentCommentEnv = null
        envHeaderLine = null
      }
      continue
    }

    // Handle comments
    if (trimmed.startsWith('#')) {
      // Check for environment header
      const envHeader = detectEnvironmentHeader(trimmed)
      if (envHeader) {
        currentCommentEnv = envHeader
        envHeaderLine = lineNumber
        skippedLines++
        continue
      }

      // Check for commented-out key=value
      const kvMatch = trimmed.match(COMMENTED_KV_PATTERN)
      if (kvMatch) {
        const [, key, value] = kvMatch
        if (isValidEnvKey(key)) {
          commentedEntries.push({
            key,
            value: parseQuotedValue(value),
            lineNumber,
            raw,
            isCommented: true,
            commentPrefix: trimmed.match(/^(#+\s*)/)?.[1] || '# ',
            commentEnvironment: currentCommentEnv || undefined,
          })
          continue
        }
      }

      skippedLines++
      continue
    }

    // Reset comment environment for uncommented lines
    currentCommentEnv = null
    envHeaderLine = null

    // Skip export prefix if present (common in shell scripts)
    const lineContent = trimmed.replace(/^export\s+/, '')

    // Parse key=value
    const equalsIndex = lineContent.indexOf('=')
    if (equalsIndex === -1) {
      errors.push({
        lineNumber,
        line: raw,
        message: 'Invalid format: missing "=" separator',
      })
      continue
    }

    const key = lineContent.slice(0, equalsIndex).trim()
    let value = lineContent.slice(equalsIndex + 1)

    // Validate key format
    if (!isValidEnvKey(key)) {
      errors.push({
        lineNumber,
        line: raw,
        message: `Invalid key format: "${key}"`,
      })
      continue
    }

    // Handle quoted values
    value = parseQuotedValue(value)

    entries.push({
      key,
      value,
      lineNumber,
      raw,
    })
  }

  // Group commented entries by environment
  const commentedGroups = groupCommentedEntries(commentedEntries)

  return {
    entries,
    errors,
    commentedEntries,
    commentedGroups,
    metadata: {
      totalLines: lines.length,
      validEntries: entries.length,
      skippedLines,
    },
  }
}

/**
 * Group commented entries by environment and proximity
 */
function groupCommentedEntries(
  entries: EnvEntryExtended[]
): CommentedConnectionGroup[] {
  const groups: CommentedConnectionGroup[] = []

  // Group by detected environment
  const byEnv = new Map<string, EnvEntryExtended[]>()

  for (const entry of entries) {
    const env = entry.commentEnvironment || 'unknown'
    const existing = byEnv.get(env) || []
    existing.push(entry)
    byEnv.set(env, existing)
  }

  for (const [environment, envEntries] of byEnv) {
    // Further group by line proximity
    const proximityGroups = groupByProximity(envEntries)

    for (const group of proximityGroups) {
      const isComplete = hasMinimumConnectionFields(group)

      groups.push({
        entries: group,
        environment,
        isComplete,
        headerLineNumber: group[0]?.lineNumber,
      })
    }
  }

  return groups
}

/**
 * Group entries by line proximity (within 3 lines of each other)
 */
function groupByProximity(entries: EnvEntryExtended[]): EnvEntryExtended[][] {
  if (entries.length === 0) return []

  const sorted = [...entries].sort((a, b) => a.lineNumber - b.lineNumber)
  const groups: EnvEntryExtended[][] = []
  let currentGroup: EnvEntryExtended[] = [sorted[0]]

  for (let i = 1; i < sorted.length; i++) {
    const gap = sorted[i].lineNumber - sorted[i - 1].lineNumber

    if (gap <= 3) {
      // Within proximity - same group
      currentGroup.push(sorted[i])
    } else {
      // Gap too large - new group
      groups.push(currentGroup)
      currentGroup = [sorted[i]]
    }
  }

  groups.push(currentGroup)
  return groups
}

/**
 * Check if entries have minimum fields for a connection
 */
function hasMinimumConnectionFields(entries: EnvEntryExtended[]): boolean {
  const keys = entries.map(e => e.key.toUpperCase())

  // Check for connection URL (complete by itself)
  if (keys.some(k => /_URL$|_URI$|_DSN$|_CONNECTION_STRING$/.test(k))) {
    return true
  }

  // Check for host+port minimum
  const hasHost = keys.some(k => /HOST|SERVER|HOSTNAME/.test(k))
  const hasPort = keys.some(k => /PORT/.test(k))

  return hasHost && hasPort
}

// =============================================================================
// Proximity-based Connection Grouping
// =============================================================================

/**
 * Window size for proximity grouping (number of lines to look ahead/behind)
 */
const PROXIMITY_WINDOW = 5

/**
 * Database-related key patterns for context detection
 */
const DB_CONTEXT_PATTERNS = [
  /^DB_/i,
  /^DATABASE_/i,
  /^POSTGRES/i,
  /^MYSQL/i,
  /^MONGO/i,
  /^PG/i,
  /_PORT$/i,
  /_USER$/i,
  /_PASSWORD$/i,
  /_NAME$/i,
]

/**
 * Check if a key indicates database context
 */
function isDbContextKey(key: string): boolean {
  return DB_CONTEXT_PATTERNS.some(pattern => pattern.test(key))
}

/**
 * Find entries within proximity window
 */
function getProximityNeighbors(
  entries: EnvEntry[],
  targetLineNumber: number,
  windowSize: number = PROXIMITY_WINDOW
): EnvEntry[] {
  return entries.filter(e =>
    Math.abs(e.lineNumber - targetLineNumber) <= windowSize
  )
}

/**
 * Check if standalone HOST variable is in database context
 */
function isHostInDbContext(
  hostEntry: EnvEntry,
  allEntries: EnvEntry[]
): boolean {
  // Only applies to standalone HOST patterns
  if (!/^HOST(_READER|_WRITER|_\d+)?$/i.test(hostEntry.key)) {
    return false
  }

  const neighbors = getProximityNeighbors(allEntries, hostEntry.lineNumber)
  const dbNeighbors = neighbors.filter(e => isDbContextKey(e.key))

  // Require at least 2 DB-context neighbors (e.g., DB_PORT and DB_USER)
  return dbNeighbors.length >= 2
}

/**
 * Enhanced entry filtering with proximity analysis
 *
 * Catches standalone HOST variables when they appear near DB_* variables
 */
export function filterConnectionRelatedEntriesWithProximity(
  entries: EnvEntry[]
): EnvEntry[] {
  const basicMatches = filterConnectionRelatedEntries(entries)

  // Find standalone HOST vars that need proximity check
  const standaloneHostPattern = /^HOST(_READER|_WRITER|_\d+)?$/i

  const additionalHosts = entries.filter(entry =>
    standaloneHostPattern.test(entry.key) &&
    !basicMatches.some(m => m.key === entry.key) &&
    isHostInDbContext(entry, entries)
  )

  return [...basicMatches, ...additionalHosts]
}
