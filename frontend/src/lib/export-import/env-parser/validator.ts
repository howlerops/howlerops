/**
 * Connection Validator
 *
 * Validates parsed .env connections before import.
 *
 * @module lib/export-import/env-parser/validator
 */

import type { ParsedEnvConnection } from './types'

/**
 * Validation result for a single connection
 */
export interface ValidationResult {
  /** Whether the connection is valid for import */
  isValid: boolean

  /** Validation errors (blocking) */
  errors: string[]

  /** Validation warnings (non-blocking) */
  warnings: string[]
}

/**
 * Required fields by database type
 */
const REQUIRED_FIELDS: Record<string, string[]> = {
  postgresql: ['host', 'database'],
  mysql: ['host', 'database'],
  mariadb: ['host', 'database'],
  mongodb: ['host'],
  redis: ['host'],
  elasticsearch: ['host'],
  opensearch: ['host'],
  clickhouse: ['host'],
  mssql: ['host', 'database'],
  sqlite: [], // SQLite only needs a file path, handled separately
}

/**
 * Validate a single parsed connection
 *
 * @param connection - Parsed connection to validate
 * @returns Validation result with errors and warnings
 */
export function validateParsedConnection(
  connection: ParsedEnvConnection
): ValidationResult {
  const errors: string[] = []
  const warnings: string[] = []

  // Check for name
  if (!connection.suggestedName || connection.suggestedName.trim() === '') {
    errors.push('Connection name is required')
  }

  // Check required fields based on database type
  const required = REQUIRED_FIELDS[connection.type] || ['host']

  for (const field of required) {
    const value = connection[field as keyof ParsedEnvConnection]
    if (value === undefined || value === null || value === '') {
      errors.push(`${capitalizeFirst(field)} is required for ${connection.type}`)
    }
  }

  // Validate host format
  if (connection.host) {
    if (!isValidHost(connection.host)) {
      errors.push('Invalid host format')
    }
  }

  // Validate port range
  if (connection.port !== undefined) {
    if (connection.port < 1 || connection.port > 65535) {
      errors.push('Port must be between 1 and 65535')
    }
  }

  // Warn about missing optional but recommended fields
  if (!connection.username) {
    warnings.push('No username specified - connection may fail if authentication is required')
  }

  if (!connection.password) {
    warnings.push('No password specified - you may need to add it manually')
  }

  // Warn about low confidence
  if (connection.overallConfidence === 'low') {
    warnings.push('Low confidence extraction - please review all fields carefully')
  }

  // Type-specific validations
  if (connection.type === 'mongodb' && connection.connectionString) {
    if (
      !connection.connectionString.startsWith('mongodb://') &&
      !connection.connectionString.startsWith('mongodb+srv://')
    ) {
      warnings.push('MongoDB connection string should start with mongodb:// or mongodb+srv://')
    }
  }

  // Redis validation removed - redis not yet supported

  return {
    isValid: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Validate all parsed connections
 *
 * @param connections - Array of parsed connections
 * @returns Map of tempId to validation result
 */
export function validateAllParsedConnections(
  connections: ParsedEnvConnection[]
): Map<string, ValidationResult> {
  const results = new Map<string, ValidationResult>()

  for (const conn of connections) {
    results.set(conn.tempId, validateParsedConnection(conn))
  }

  return results
}

/**
 * Update validation errors on parsed connections
 * Mutates the connections array to add validation errors
 *
 * @param connections - Array of parsed connections
 */
export function applyValidationToConnections(
  connections: ParsedEnvConnection[]
): void {
  const results = validateAllParsedConnections(connections)

  for (const conn of connections) {
    const result = results.get(conn.tempId)
    if (result) {
      conn.validationErrors = result.errors
    }
  }
}

/**
 * Check if all connections are valid for import
 *
 * @param connections - Array of parsed connections
 * @param skipInvalid - Whether to skip invalid connections
 * @returns Whether import can proceed
 */
export function canImportConnections(
  connections: ParsedEnvConnection[],
  skipInvalid: boolean = false
): boolean {
  const nonSkipped = connections.filter(c => !c.isSkipped)

  if (nonSkipped.length === 0) {
    return false
  }

  if (skipInvalid) {
    // At least one valid connection required
    return nonSkipped.some(c => c.validationErrors.length === 0)
  }

  // All non-skipped must be valid
  return nonSkipped.every(c => c.validationErrors.length === 0)
}

/**
 * Get summary of validation status
 */
export function getValidationSummary(connections: ParsedEnvConnection[]): {
  total: number
  valid: number
  invalid: number
  skipped: number
  warnings: number
} {
  let valid = 0
  let invalid = 0
  let skipped = 0
  let warnings = 0

  for (const conn of connections) {
    if (conn.isSkipped) {
      skipped++
      continue
    }

    const result = validateParsedConnection(conn)
    if (result.isValid) {
      valid++
      if (result.warnings.length > 0) {
        warnings++
      }
    } else {
      invalid++
    }
  }

  return {
    total: connections.length,
    valid,
    invalid,
    skipped,
    warnings,
  }
}

/**
 * Check if a host string is valid
 */
function isValidHost(host: string): boolean {
  // Allow localhost
  if (host === 'localhost') return true

  // Allow IP addresses
  const ipPattern =
    /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
  if (ipPattern.test(host)) return true

  // Allow hostnames
  const hostnamePattern =
    /^(?!-)[A-Za-z0-9-]{1,63}(?<!-)(?:\.(?!-)[A-Za-z0-9-]{1,63}(?<!-))*$/
  if (hostnamePattern.test(host)) return true

  return false
}

/**
 * Capitalize first letter
 */
function capitalizeFirst(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1)
}
