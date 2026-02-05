/**
 * Connection Export/Import Validation
 *
 * Validation functions for export files and connections.
 * Ensures imported data meets schema requirements before use.
 *
 * @module lib/export-import/validation
 */

import {
  ConnectionExportFile,
  CURRENT_SCHEMA_VERSION,
  ExportedConnection,
  REQUIRED_CONNECTION_FIELDS,
  VALID_DATABASE_TYPES,
  ValidationResult,
} from './types'

// =============================================================================
// File Validation
// =============================================================================

/**
 * Validate an export file structure
 *
 * @param data - Parsed JSON data to validate
 * @returns Validation result with errors and warnings
 */
export function validateExportFile(data: unknown): ValidationResult {
  const errors: string[] = []
  const warnings: string[] = []

  // Check if data is an object
  if (!data || typeof data !== 'object') {
    return {
      isValid: false,
      errors: ['Export file must be a JSON object'],
      warnings: [],
    }
  }

  const file = data as Record<string, unknown>

  // Check for required top-level fields
  if (!file.metadata) {
    errors.push('Missing required field: metadata')
  } else {
    validateMetadata(file.metadata, errors, warnings)
  }

  if (!file.connections) {
    errors.push('Missing required field: connections')
  } else if (!Array.isArray(file.connections)) {
    errors.push('Field "connections" must be an array')
  }

  return {
    isValid: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Validate export file metadata
 */
function validateMetadata(
  metadata: unknown,
  errors: string[],
  warnings: string[]
): void {
  if (!metadata || typeof metadata !== 'object') {
    errors.push('Metadata must be an object')
    return
  }

  const meta = metadata as Record<string, unknown>

  // Required metadata fields
  if (typeof meta.version !== 'string') {
    errors.push('Metadata missing required field: version')
  } else {
    // Check version compatibility
    const [majorCurrent] = CURRENT_SCHEMA_VERSION.split('.').map(Number)
    const [majorFile] = meta.version.split('.').map(Number)

    if (majorFile > majorCurrent) {
      warnings.push(
        `Export file version (${meta.version}) is newer than supported (${CURRENT_SCHEMA_VERSION}). ` +
        'Some features may not import correctly.'
      )
    }
  }

  if (typeof meta.exportedAt !== 'string') {
    warnings.push('Metadata missing field: exportedAt')
  }

  if (typeof meta.connectionCount !== 'number') {
    warnings.push('Metadata missing field: connectionCount')
  }

  if (typeof meta.includesPasswords !== 'boolean') {
    warnings.push('Metadata missing field: includesPasswords')
  }
}

// =============================================================================
// Connection Validation
// =============================================================================

/**
 * Validate a single connection from an export file
 *
 * @param connection - Connection data to validate
 * @returns Validation result with errors and warnings
 */
export function validateConnection(connection: unknown): ValidationResult {
  const errors: string[] = []
  const warnings: string[] = []

  // Check if connection is an object
  if (!connection || typeof connection !== 'object') {
    return {
      isValid: false,
      errors: ['Connection must be an object'],
      warnings: [],
    }
  }

  const conn = connection as Record<string, unknown>

  // Check required fields
  for (const field of REQUIRED_CONNECTION_FIELDS) {
    if (!conn[field]) {
      errors.push(`Missing required field: ${field}`)
    } else if (typeof conn[field] !== 'string') {
      errors.push(`Field "${field}" must be a string`)
    }
  }

  // Validate ID format (should be UUID-like)
  if (conn.id && typeof conn.id === 'string') {
    if (!isValidUUID(conn.id)) {
      warnings.push('Connection ID is not a valid UUID format')
    }
  }

  // Validate database type
  if (conn.type && typeof conn.type === 'string') {
    if (!VALID_DATABASE_TYPES.includes(conn.type as typeof VALID_DATABASE_TYPES[number])) {
      errors.push(
        `Invalid database type: "${conn.type}". ` +
        `Valid types: ${VALID_DATABASE_TYPES.join(', ')}`
      )
    }
  }

  // Validate port if present
  if (conn.port !== undefined) {
    if (typeof conn.port !== 'number') {
      errors.push('Field "port" must be a number')
    } else if (conn.port < 1 || conn.port > 65535) {
      errors.push('Port must be between 1 and 65535')
    }
  }

  // Validate environments if present
  if (conn.environments !== undefined) {
    if (!Array.isArray(conn.environments)) {
      errors.push('Field "environments" must be an array')
    } else {
      const invalidEnvs = conn.environments.filter(
        (e: unknown) => typeof e !== 'string'
      )
      if (invalidEnvs.length > 0) {
        errors.push('All environment values must be strings')
      }
    }
  }

  // Validate SSH tunnel config if tunnel is enabled
  if (conn.useTunnel === true) {
    if (!conn.sshTunnel) {
      errors.push('SSH tunnel enabled but sshTunnel config is missing')
    } else {
      validateSSHTunnelConfig(conn.sshTunnel, errors, warnings)
    }
  }

  // Validate VPC config if VPC is enabled
  if (conn.useVpc === true) {
    if (!conn.vpcConfig) {
      errors.push('VPC enabled but vpcConfig is missing')
    } else {
      validateVPCConfig(conn.vpcConfig, errors, warnings)
    }
  }

  // Validate parameters if present
  if (conn.parameters !== undefined) {
    if (typeof conn.parameters !== 'object' || Array.isArray(conn.parameters)) {
      errors.push('Field "parameters" must be an object')
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Validate SSH tunnel configuration
 */
function validateSSHTunnelConfig(
  config: unknown,
  errors: string[],
  warnings: string[]
): void {
  if (!config || typeof config !== 'object') {
    errors.push('SSH tunnel config must be an object')
    return
  }

  const ssh = config as Record<string, unknown>

  // Required SSH fields
  if (typeof ssh.host !== 'string' || !ssh.host) {
    errors.push('SSH tunnel missing required field: host')
  }

  if (typeof ssh.port !== 'number') {
    errors.push('SSH tunnel missing required field: port')
  } else if (ssh.port < 1 || ssh.port > 65535) {
    errors.push('SSH port must be between 1 and 65535')
  }

  if (typeof ssh.user !== 'string' || !ssh.user) {
    errors.push('SSH tunnel missing required field: user')
  }

  // Auth method validation
  const validAuthMethods = [0, 1, 2, 3] // SSHAuthMethod enum values
  if (ssh.authMethod !== undefined && !validAuthMethods.includes(ssh.authMethod as number)) {
    warnings.push('Invalid SSH auth method')
  }

  // Warn if password or privateKey content is present (should not be exported)
  if (ssh.password) {
    warnings.push('SSH password found in export - this should not normally be included')
  }
  if (ssh.privateKey) {
    warnings.push('SSH private key content found in export - this is a security concern')
  }
}

/**
 * Validate VPC configuration
 */
function validateVPCConfig(
  config: unknown,
  errors: string[],
  warnings: string[]
): void {
  if (!config || typeof config !== 'object') {
    errors.push('VPC config must be an object')
    return
  }

  const vpc = config as Record<string, unknown>

  // Required VPC fields
  if (typeof vpc.vpcId !== 'string' || !vpc.vpcId) {
    warnings.push('VPC config missing field: vpcId')
  }

  if (typeof vpc.subnetId !== 'string' || !vpc.subnetId) {
    warnings.push('VPC config missing field: subnetId')
  }

  // Security group IDs
  if (vpc.securityGroupIds !== undefined) {
    if (!Array.isArray(vpc.securityGroupIds)) {
      errors.push('VPC securityGroupIds must be an array')
    }
  }
}

// =============================================================================
// Batch Validation
// =============================================================================

/**
 * Validate all connections in an export file
 *
 * @param file - Parsed export file
 * @returns Array of validation results for each connection
 */
export function validateAllConnections(
  file: ConnectionExportFile
): Array<{ connection: ExportedConnection; validation: ValidationResult }> {
  return file.connections.map((connection) => ({
    connection,
    validation: validateConnection(connection),
  }))
}

/**
 * Get summary of validation results
 */
export function getValidationSummary(
  results: Array<{ connection: ExportedConnection; validation: ValidationResult }>
): {
  totalConnections: number
  validConnections: number
  invalidConnections: number
  connectionsWithWarnings: number
} {
  const valid = results.filter((r) => r.validation.isValid)
  const withWarnings = results.filter(
    (r) => r.validation.isValid && r.validation.warnings.length > 0
  )

  return {
    totalConnections: results.length,
    validConnections: valid.length,
    invalidConnections: results.length - valid.length,
    connectionsWithWarnings: withWarnings.length,
  }
}

// =============================================================================
// Utility Functions
// =============================================================================

/**
 * Check if a string is a valid UUID format
 */
function isValidUUID(str: string): boolean {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  return uuidRegex.test(str)
}

/**
 * Check if the parsed file contains passwords
 */
export function fileContainsPasswords(file: ConnectionExportFile): boolean {
  // Check metadata flag first
  if (file.metadata.includesPasswords) {
    return true
  }

  // Double-check by scanning connections
  return file.connections.some((conn) => !!conn.password)
}

/**
 * Find connections in import that conflict with existing connections
 *
 * @param importConnections - Connections from import file
 * @param existingIds - Set of existing connection IDs
 * @returns Array of conflicting connection IDs
 */
export function findConflictingConnections(
  importConnections: ExportedConnection[],
  existingIds: Set<string>
): string[] {
  return importConnections
    .filter((conn) => existingIds.has(conn.id))
    .map((conn) => conn.id)
}
