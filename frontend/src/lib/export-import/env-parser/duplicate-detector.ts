/**
 * Duplicate Connection Detector
 *
 * Detects potential duplicate connections by comparing
 * parsed .env connections against existing stored connections.
 *
 * @module lib/export-import/env-parser/duplicate-detector
 */

import type { ParsedEnvConnection } from './types'

/**
 * Existing connection info for duplicate detection
 */
export interface ExistingConnection {
  id: string
  name: string
  type: string
  host?: string
  port?: number
  database?: string
  environments?: string[]
}

/**
 * Result of duplicate detection
 */
export interface DuplicateCheckResult {
  /** Whether any duplicates were found */
  hasDuplicates: boolean

  /** Connections with duplicate info attached */
  connections: ParsedEnvConnection[]

  /** Summary of duplicate detection */
  summary: {
    total: number
    duplicates: number
    unique: number
  }
}

/**
 * Check parsed connections against existing connections for duplicates
 *
 * @param parsedConnections - Connections extracted from .env
 * @param existingConnections - Currently stored connections
 * @returns Parsed connections with duplicate info attached
 */
export function detectDuplicates(
  parsedConnections: ParsedEnvConnection[],
  existingConnections: ExistingConnection[]
): DuplicateCheckResult {
  let duplicateCount = 0

  const updatedConnections = parsedConnections.map((parsed) => {
    const duplicate = findDuplicate(parsed, existingConnections)

    if (duplicate) {
      duplicateCount++
      return {
        ...parsed,
        duplicateOfId: duplicate.id,
        duplicateOfName: duplicate.name,
      }
    }

    return parsed
  })

  return {
    hasDuplicates: duplicateCount > 0,
    connections: updatedConnections,
    summary: {
      total: parsedConnections.length,
      duplicates: duplicateCount,
      unique: parsedConnections.length - duplicateCount,
    },
  }
}

/**
 * Find a matching existing connection for a parsed connection
 */
function findDuplicate(
  parsed: ParsedEnvConnection,
  existing: ExistingConnection[]
): ExistingConnection | undefined {
  // Try exact match first (same host, port, database, type)
  const exactMatch = existing.find(
    (conn) =>
      conn.type === parsed.type &&
      conn.host === parsed.host &&
      conn.port === parsed.port &&
      conn.database === parsed.database
  )

  if (exactMatch) return exactMatch

  // Try host + database match (port might differ or be default)
  const hostDbMatch = existing.find(
    (conn) =>
      conn.type === parsed.type &&
      conn.host === parsed.host &&
      conn.database === parsed.database
  )

  if (hostDbMatch) return hostDbMatch

  // Try name similarity match (fuzzy)
  const nameMatch = existing.find((conn) =>
    areNamesSimilar(conn.name, parsed.suggestedName)
  )

  if (nameMatch) {
    // Only consider name match if type also matches
    if (nameMatch.type === parsed.type) {
      return nameMatch
    }
  }

  return undefined
}

/**
 * Check if two connection names are similar enough to be duplicates
 */
function areNamesSimilar(name1: string, name2: string): boolean {
  const normalize = (s: string) =>
    s
      .toLowerCase()
      .replace(/[^a-z0-9]/g, '')
      .replace(/(dev|development|staging|stg|prod|production|test|local)/g, '')

  const n1 = normalize(name1)
  const n2 = normalize(name2)

  // Exact match after normalization
  if (n1 === n2) return true

  // One contains the other
  if (n1.includes(n2) || n2.includes(n1)) return true

  // Calculate similarity ratio
  const similarity = calculateSimilarity(n1, n2)
  return similarity > 0.8
}

/**
 * Calculate similarity between two strings (0-1)
 * Uses Levenshtein distance
 */
function calculateSimilarity(s1: string, s2: string): number {
  if (s1 === s2) return 1
  if (s1.length === 0 || s2.length === 0) return 0

  const longer = s1.length > s2.length ? s1 : s2
  const shorter = s1.length > s2.length ? s2 : s1

  const longerLength = longer.length
  const distance = levenshteinDistance(longer, shorter)

  return (longerLength - distance) / longerLength
}

/**
 * Calculate Levenshtein distance between two strings
 */
function levenshteinDistance(s1: string, s2: string): number {
  const costs: number[] = []

  for (let i = 0; i <= s1.length; i++) {
    let lastValue = i
    for (let j = 0; j <= s2.length; j++) {
      if (i === 0) {
        costs[j] = j
      } else if (j > 0) {
        let newValue = costs[j - 1]
        if (s1.charAt(i - 1) !== s2.charAt(j - 1)) {
          newValue = Math.min(Math.min(newValue, lastValue), costs[j]) + 1
        }
        costs[j - 1] = lastValue
        lastValue = newValue
      }
    }
    if (i > 0) {
      costs[s2.length] = lastValue
    }
  }

  return costs[s2.length]
}

/**
 * Detect environment from .env filename
 *
 * @param filename - Name of the .env file
 * @returns Detected environment or undefined
 */
export function detectEnvironmentFromFilename(
  filename: string
): string | undefined {
  const lower = filename.toLowerCase()

  if (lower.includes('.development') || lower.includes('.dev')) {
    return 'development'
  }
  if (lower.includes('.staging') || lower.includes('.stg')) {
    return 'staging'
  }
  if (lower.includes('.production') || lower.includes('.prod')) {
    return 'production'
  }
  if (lower.includes('.test')) {
    return 'test'
  }
  if (lower.includes('.local')) {
    return 'local'
  }

  return undefined
}

/**
 * Apply environment from filename to connections that don't have one detected
 *
 * @param connections - Parsed connections
 * @param filename - Name of the source .env file
 * @returns Updated connections with environment from filename
 */
export function applyFilenameEnvironment(
  connections: ParsedEnvConnection[],
  filename: string
): ParsedEnvConnection[] {
  const fileEnv = detectEnvironmentFromFilename(filename)

  if (!fileEnv) return connections

  return connections.map((conn) => {
    // Only apply if no environment already detected
    if (!conn.detectedEnvironment) {
      return {
        ...conn,
        detectedEnvironment: fileEnv,
        environmentSource: 'file-name' as const,
      }
    }
    return conn
  })
}

/**
 * Group connections by detected environment
 */
export function groupByEnvironment(
  connections: ParsedEnvConnection[]
): Map<string, ParsedEnvConnection[]> {
  const groups = new Map<string, ParsedEnvConnection[]>()

  for (const conn of connections) {
    const env = conn.detectedEnvironment || 'unknown'
    const existing = groups.get(env) || []
    existing.push(conn)
    groups.set(env, existing)
  }

  return groups
}

/**
 * Check if importing these connections would create environment conflicts
 * (e.g., importing a "production" connection when one already exists)
 */
export function checkEnvironmentConflicts(
  parsedConnections: ParsedEnvConnection[],
  existingConnections: ExistingConnection[]
): Array<{
  parsed: ParsedEnvConnection
  existing: ExistingConnection
  conflictType: 'same-environment' | 'same-name-different-env'
}> {
  const conflicts: Array<{
    parsed: ParsedEnvConnection
    existing: ExistingConnection
    conflictType: 'same-environment' | 'same-name-different-env'
  }> = []

  for (const parsed of parsedConnections) {
    for (const existing of existingConnections) {
      // Skip if different database types
      if (parsed.type !== existing.type) continue

      // Check for same environment with similar config
      const parsedEnv = parsed.detectedEnvironment
      const existingEnvs = existing.environments || []

      if (parsedEnv && existingEnvs.includes(parsedEnv)) {
        // Same type and environment - check if same host/db
        if (
          parsed.host === existing.host &&
          parsed.database === existing.database
        ) {
          conflicts.push({
            parsed,
            existing,
            conflictType: 'same-environment',
          })
        }
      }

      // Check for similar names with different environments
      if (areNamesSimilar(parsed.suggestedName, existing.name)) {
        if (parsedEnv && !existingEnvs.includes(parsedEnv)) {
          conflicts.push({
            parsed,
            existing,
            conflictType: 'same-name-different-env',
          })
        }
      }
    }
  }

  return conflicts
}
