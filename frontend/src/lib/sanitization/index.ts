/**
 * Howlerops Data Sanitization Library
 *
 * Comprehensive security module for preventing credential leakage
 * in Howlerops's sync and storage operations.
 *
 * @module sanitization
 */

import type { DatabaseConnection } from '@/store/connection-store'

// Configuration exports
export {
  createDefaultConfig,
  getGlobalConfig,
  isSensitiveColumn,
  mergeConfig,
  PrivacyMode,
  QueryPrivacyLevel,
  type SanitizationConfig,
  setGlobalConfig,
  shouldExcludeSchema,
  shouldExcludeTable} from './config'

// Credential detection exports
export {
  type CredentialDetectionResult,
  CredentialType,
  deepScanForCredentials,
  detectCredentials,
  detectCredentialsInBatch,
  mightBeCredential,
  redactCredentials
} from './credential-detector'

// Query sanitization exports
export {
  extractQueryMetadata,
  isPrivateQuery,
  type QueryMetadata,
  type QuerySanitizationResult,
  sanitizeQueries,
  sanitizeQuery} from './query-sanitizer'

// Connection sanitization exports
export {
  type ConnectionSanitizationResult,
  hasRequiredCredentials,
  prepareConnectionsForSync,
  restoreCredentials,
  sanitizeConnection,
  sanitizeConnections,
  type SanitizedConnection,
  type SanitizedSSHTunnelConfig,
  validateSanitization
} from './connection-sanitizer'

import type { SanitizationConfig } from './config'
import {
  getGlobalConfig,
  mergeConfig,
  QueryPrivacyLevel,
} from './config'
import type {
  ConnectionSanitizationResult,
  SanitizedConnection,
} from './connection-sanitizer'
import {
  prepareConnectionsForSync,
  sanitizeConnection,
} from './connection-sanitizer'
import type { CredentialDetectionResult } from './credential-detector'
import {
  deepScanForCredentials,
  detectCredentials,
} from './credential-detector'
import type { QuerySanitizationResult } from './query-sanitizer'
import { sanitizeQuery } from './query-sanitizer'

/**
 * High-level sanitization interface for common use cases
 */
export class Sanitizer {
  private config: SanitizationConfig

  constructor(config?: Partial<SanitizationConfig>) {
    this.config = config ? mergeConfig(config) : getGlobalConfig()
  }

  /**
   * Sanitize a SQL query
   */
  sanitizeQuery(query: string): QuerySanitizationResult {
    return sanitizeQuery(query, this.config)
  }

  /**
   * Sanitize a database connection
   */
  sanitizeConnection(
    connection: DatabaseConnection
  ): ConnectionSanitizationResult {
    return sanitizeConnection(connection, this.config)
  }

  /**
   * Check if a string contains credentials
   */
  detectCredentials(input: string): CredentialDetectionResult {
    return detectCredentials(input, this.config)
  }

  /**
   * Prepare data for cloud sync
   */
  prepareForSync(data: {
    queries?: string[]
    connections?: DatabaseConnection[]
  }): {
    sanitizedQueries: QuerySanitizationResult[]
    sanitizedConnections: SanitizedConnection[]
    issues: string[]
  } {
    const issues: string[] = []

    // Sanitize queries
    const sanitizedQueries = data.queries
      ? data.queries.map(q => this.sanitizeQuery(q))
      : []

    // Check for private queries
    const privateQueries = sanitizedQueries.filter(
      r => r.privacyLevel === QueryPrivacyLevel.PRIVATE
    )
    if (privateQueries.length > 0) {
      issues.push(`${privateQueries.length} queries marked as private and should not be synced`)
    }

    // Sanitize connections
    const connectionResults = data.connections
      ? prepareConnectionsForSync(data.connections, this.config)
      : { safeConnections: [], unsafeConnections: [] }

    if (connectionResults.unsafeConnections.length > 0) {
      connectionResults.unsafeConnections.forEach(({ connection, reasons }: { connection: any; reasons: string[] }) => {
        issues.push(`Connection "${connection.name}" is unsafe: ${reasons.join(', ')}`)
      })
    }

    return {
      sanitizedQueries,
      sanitizedConnections: connectionResults.safeConnections,
      issues
    }
  }

  /**
   * Validate that data is safe for sync
   */
  validateForSync(data: any): { isSafe: boolean; issues: string[] } {
    const scanResults = deepScanForCredentials(data, this.config)

    if (scanResults.length === 0) {
      return { isSafe: true, issues: [] }
    }

    const issues = scanResults.map(({ path, result }: { path: string; result: any }) =>
      `Credential detected at ${path}: ${result.type} (confidence: ${result.confidence})`
    )

    return { isSafe: false, issues }
  }

  /**
   * Update configuration
   */
  updateConfig(updates: Partial<SanitizationConfig>): void {
    this.config = mergeConfig({ ...this.config, ...updates })
  }

  /**
   * Get current configuration
   */
  getConfig(): SanitizationConfig {
    return { ...this.config }
  }
}

/**
 * Default sanitizer instance
 */
export const defaultSanitizer = new Sanitizer()

/**
 * Quick helper functions for common operations
 */

/**
 * Quick check if data is safe to sync
 */
export function isSafeToSync(data: any): boolean {
  const { isSafe } = defaultSanitizer.validateForSync(data)
  return isSafe
}

/**
 * Sanitize data before storing or syncing
 */
export function sanitizeForStorage(data: {
  query?: string
  connection?: import('@/store/connection-store').DatabaseConnection
}): {
  query?: string
  connection?: import('./connection-sanitizer').SanitizedConnection
} {
  const result: any = {}

  if (data.query) {
    const sanitized = defaultSanitizer.sanitizeQuery(data.query)
    result.query = sanitized.sanitizedQuery
  }

  if (data.connection) {
    const sanitized = defaultSanitizer.sanitizeConnection(data.connection)
    result.connection = sanitized.sanitizedConnection
  }

  return result
}

/**
 * Remove all credentials from an object
 */
export function stripCredentials(obj: any): any {
  const { deepScanForCredentials } = require('./credential-detector')
  const scanResults = deepScanForCredentials(obj)

  if (scanResults.length === 0) {
    return obj
  }

  // Create a deep copy
  const cleaned = JSON.parse(JSON.stringify(obj))

  // Remove detected credentials
  for (const { path } of scanResults) {
    const pathParts = path.split('.')
    let current = cleaned

    for (let i = 1; i < pathParts.length - 1; i++) {
      const part = pathParts[i]
      if (part.includes('[')) {
        const [arrayName, indexStr] = part.split('[')
        const index = parseInt(indexStr.replace(']', ''), 10)
        current = current[arrayName][index]
      } else {
        current = current[part]
      }
    }

    const lastPart = pathParts[pathParts.length - 1]
    if (lastPart.includes('[')) {
      const [arrayName, indexStr] = lastPart.split('[')
      const index = parseInt(indexStr.replace(']', ''), 10)
      current[arrayName][index] = '[REDACTED]'
    } else {
      current[lastPart] = '[REDACTED]'
    }
  }

  return cleaned
}
