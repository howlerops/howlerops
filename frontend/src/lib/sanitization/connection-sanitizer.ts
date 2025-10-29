/**
 * Connection Sanitization Module
 *
 * Strips credentials and sensitive data from database connection objects.
 * CRITICAL: Ensures no passwords or private keys are ever synced to cloud.
 *
 * @module sanitization/connection-sanitizer
 */

import type { DatabaseConnection, SSHTunnelConfig, VPCConfig } from '@/store/connection-store'
import { SanitizationConfig, getGlobalConfig } from './config'
import { detectCredentials, type CredentialDetectionResult, CredentialType } from './credential-detector'

export interface SanitizedConnection extends Omit<DatabaseConnection, 'password' | 'sshTunnel'> {
  /**
   * Indicates that a password exists but was removed
   */
  passwordRequired: boolean

  /**
   * SSH tunnel config with credentials removed
   */
  sshTunnel?: SanitizedSSHTunnelConfig

  /**
   * Hash of the original connection for tracking
   */
  sanitizationHash?: string

  /**
   * Timestamp of sanitization
   */
  sanitizedAt: Date

  /**
   * Flags indicating what was removed
   */
  sanitizationFlags: {
    passwordRemoved: boolean
    sshPasswordRemoved: boolean
    sshPrivateKeyRemoved: boolean
    parametersModified: boolean
    vpcConfigModified: boolean
  }
}

export interface SanitizedSSHTunnelConfig extends Omit<SSHTunnelConfig, 'password' | 'privateKey'> {
  /**
   * Indicates SSH password was present but removed
   */
  passwordRequired: boolean

  /**
   * Indicates private key was present but removed
   */
  privateKeyRequired: boolean

  /**
   * Path to private key file (kept for reference)
   */
  privateKeyPath?: string
}

export interface ConnectionSanitizationResult {
  /**
   * The sanitized connection object
   */
  sanitizedConnection: SanitizedConnection

  /**
   * Whether any modifications were made
   */
  wasModified: boolean

  /**
   * What was removed/modified
   */
  removedFields: string[]

  /**
   * Any credentials detected in unexpected places
   */
  unexpectedCredentials: Array<{
    field: string
    type: string
    confidence: number
  }>

  /**
   * Whether this connection is safe to sync
   */
  isSafeToSync: boolean

  /**
   * Reasons why it might not be safe
   */
  safetyIssues: string[]
}

const detectCredentialInValue = (
  value: string,
  config: SanitizationConfig
): CredentialDetectionResult => {
  const direct = detectCredentials(value, config)
  let bestMatch = direct

  if (!direct.isCredential || direct.confidence < 0.8) {
    const tokens = value.split(/[\s,;:|]+/)
    for (const token of tokens) {
      const trimmed = token.trim()
      if (!trimmed) {
        continue
      }
      const check = detectCredentials(trimmed, config)
      if (check.isCredential && check.confidence > bestMatch.confidence) {
        bestMatch = check
      }
    }
  }

  const substringPatterns: Array<{ type: CredentialType; regex: RegExp; confidence: number }> = [
    { type: CredentialType.API_KEY, regex: /(sk|rk|pk)-[A-Za-z0-9]{8,}/i, confidence: 0.8 },
    { type: CredentialType.API_KEY, regex: /api[_-]?key[^A-Za-z0-9]*[A-Za-z0-9]{8,}/i, confidence: 0.75 },
    { type: CredentialType.JWT_TOKEN, regex: /eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/, confidence: 0.8 }
  ]

  for (const pattern of substringPatterns) {
    if (pattern.regex.test(value)) {
      const match: CredentialDetectionResult = {
        isCredential: true,
        type: pattern.type,
        confidence: pattern.confidence,
        reason: 'Credential substring detected'
      }
      if (match.confidence > bestMatch.confidence) {
        bestMatch = match
      }
    }
  }

  return bestMatch
}

/**
 * Main connection sanitization function
 */
export function sanitizeConnection(
  connection: DatabaseConnection,
  config: SanitizationConfig = getGlobalConfig()
): ConnectionSanitizationResult {
  if (!connection || typeof connection !== 'object') {
    const fallback: SanitizedConnection = {
      id: '',
      name: '',
      type: 'unknown' as DatabaseConnection['type'],
      database: '',
      isConnected: false,
      passwordRequired: false,
      sanitizedAt: new Date(),
      sanitizationFlags: {
        passwordRemoved: false,
        sshPasswordRemoved: false,
        sshPrivateKeyRemoved: false,
        parametersModified: false,
        vpcConfigModified: false,
      },
    }

    return {
      sanitizedConnection: fallback,
      wasModified: false,
      removedFields: [],
      unexpectedCredentials: [],
      isSafeToSync: false,
      safetyIssues: ['Invalid connection object'],
    }
  }

  const result: ConnectionSanitizationResult = {
    sanitizedConnection: null as any, // Will be set below
    wasModified: false,
    removedFields: [],
    unexpectedCredentials: [],
    isSafeToSync: true,
    safetyIssues: []
  }

  // Create a deep copy to avoid modifying the original
  const sanitized: any = JSON.parse(JSON.stringify(connection))

  // Track what we remove
  const flags = {
    passwordRemoved: false,
    sshPasswordRemoved: false,
    sshPrivateKeyRemoved: false,
    parametersModified: false,
    vpcConfigModified: false
  }

  // Remove main password
  if (connection.password) {
    flags.passwordRemoved = true
    result.removedFields.push('password')
    result.wasModified = true
    delete sanitized.password
  }

  // Add password required flag
  sanitized.passwordRequired = flags.passwordRemoved

  // Sanitize SSH tunnel configuration
  if (connection.sshTunnel) {
    const sshSanitized: any = { ...connection.sshTunnel }

    // Remove SSH password
    if (connection.sshTunnel?.password) {
      flags.sshPasswordRemoved = true
      result.removedFields.push('sshTunnel.password')
      result.wasModified = true
      delete sshSanitized.password
    }

    // Remove SSH private key content (but keep the path)
    if (connection.sshTunnel?.privateKey) {
      flags.sshPrivateKeyRemoved = true
      result.removedFields.push('sshTunnel.privateKey')
      result.wasModified = true
      delete sshSanitized.privateKey
    }

    // Add requirement flags
    sshSanitized.passwordRequired = flags.sshPasswordRemoved
    sshSanitized.privateKeyRequired = flags.sshPrivateKeyRemoved

    sanitized.sshTunnel = sshSanitized as SanitizedSSHTunnelConfig
  }

  // Check parameters for embedded credentials
  if (connection.parameters) {
    const sanitizedParams: Record<string, string> = {}
    let paramsModified = false

    for (const [key, value] of Object.entries(connection.parameters)) {
      // Check if the parameter name suggests credentials
      const suspiciousKeys = ['password', 'pwd', 'pass', 'secret', 'token', 'key', 'auth']
      const keyLower = key.toLowerCase()

      if (suspiciousKeys.some(s => keyLower.includes(s))) {
        result.removedFields.push(`parameters.${key}`)
        result.safetyIssues.push(`Parameter '${key}' appears to contain credentials`)
        paramsModified = true
        result.wasModified = true
        flags.parametersModified = true
        // Don't include this parameter
        continue
      }

      // Check if the value looks like a credential
      const credCheck = detectCredentialInValue(value, config)
      if (credCheck.isCredential) {
        result.unexpectedCredentials.push({
          field: `parameters.${key}`,
          type: credCheck.type,
          confidence: credCheck.confidence
        })
        result.removedFields.push(`parameters.${key}`)
        paramsModified = true
        result.wasModified = true
        flags.parametersModified = true
        // Don't include this parameter
        continue
      }

      // Keep the parameter if it passed checks
      sanitizedParams[key] = value
    }

    sanitized.parameters = paramsModified ? sanitizedParams : connection.parameters
  }

  // Sanitize VPC configuration
  if (connection.vpcConfig) {
    const vpcSanitized = { ...connection.vpcConfig }
    let vpcModified = false

    // Check custom config for credentials
    if (connection.vpcConfig.customConfig) {
      const sanitizedCustom: Record<string, string> = {}

      for (const [key, value] of Object.entries(connection.vpcConfig.customConfig)) {
        const credCheck = detectCredentialInValue(value, config ?? getGlobalConfig())
        if (credCheck.isCredential) {
          result.unexpectedCredentials.push({
            field: `vpcConfig.customConfig.${key}`,
            type: credCheck.type,
            confidence: credCheck.confidence
          })
          result.removedFields.push(`vpcConfig.customConfig.${key}`)
          vpcModified = true
          result.wasModified = true
          flags.vpcConfigModified = true
          // Don't include this config
          continue
        }
        sanitizedCustom[key] = value
      }

      if (vpcModified) {
        vpcSanitized.customConfig = sanitizedCustom
      }
    }

    sanitized.vpcConfig = vpcSanitized
  }

  // Deep scan for any missed credentials in other fields
  const deepScanFields = ['name', 'host', 'database', 'username']
  for (const field of deepScanFields) {
    if (connection[field as keyof DatabaseConnection]) {
      const value = String(connection[field as keyof DatabaseConnection])
      const credCheck = detectCredentialInValue(value, config)

      if (credCheck.isCredential && credCheck.confidence > 0.7) {
        result.unexpectedCredentials.push({
          field,
          type: credCheck.type,
          confidence: credCheck.confidence
        })
        result.safetyIssues.push(`Field '${field}' contains potential credentials`)
        result.isSafeToSync = false

        // For high confidence credentials in unexpected places, redact the value
        if (credCheck.confidence > 0.8) {
          sanitized[field] = '[REDACTED]'
          result.wasModified = true
          result.removedFields.push(field)
        }
      }
    }
  }

  // Add metadata
  sanitized.sanitizationFlags = flags
  sanitized.sanitizedAt = new Date()
  sanitized.sanitizationHash = generateConnectionHash(connection)

  // Final safety check
  if (result.unexpectedCredentials.length > 0) {
    result.isSafeToSync = false
    result.safetyIssues.push(
      `Found ${result.unexpectedCredentials.length} unexpected credential(s)`
    )
  }

  // Check if critical fields were removed
  if (flags.passwordRemoved || flags.sshPasswordRemoved || flags.sshPrivateKeyRemoved) {
    // This is expected and safe
  }

  result.sanitizedConnection = sanitized as SanitizedConnection

  return result
}

/**
 * Batch sanitization for multiple connections
 */
export function sanitizeConnections(
  connections: DatabaseConnection[],
  config?: SanitizationConfig
): ConnectionSanitizationResult[] {
  return connections.map(conn => sanitizeConnection(conn, config))
}

/**
 * Prepare connections for sync (removes all sensitive data)
 */
export function prepareConnectionsForSync(
  connections: DatabaseConnection[],
  config?: SanitizationConfig
): {
  safeConnections: SanitizedConnection[]
  unsafeConnections: Array<{ connection: DatabaseConnection; reasons: string[] }>
} {
  const safeConnections: SanitizedConnection[] = []
  const unsafeConnections: Array<{ connection: DatabaseConnection; reasons: string[] }> = []
  for (const connection of connections) {
    const result = sanitizeConnection(connection, config)

    if (result.isSafeToSync) {
      safeConnections.push(result.sanitizedConnection)
    } else {
      unsafeConnections.push({
        connection,
        reasons: result.safetyIssues
      })
    }
  }

  return { safeConnections, unsafeConnections }
}

/**
 * Check if a connection has all required credentials
 */
export function hasRequiredCredentials(connection: DatabaseConnection | SanitizedConnection): boolean {
  // Check if it's a sanitized connection
  if ('passwordRequired' in connection) {
    const sanitized = connection as SanitizedConnection

    // If password is required but not present, credentials are missing
    if (sanitized.passwordRequired) {
      return false
    }

    // Check SSH credentials if tunnel is used
    if (sanitized.useTunnel && sanitized.sshTunnel) {
      if (sanitized.sshTunnel.passwordRequired || sanitized.sshTunnel.privateKeyRequired) {
        return false
      }
    }

    return true
  }

  // For unsanitized connections, check if credentials exist
  const unsanitized = connection as DatabaseConnection

  // Some database types don't require passwords (e.g., SQLite)
  if (unsanitized.type === 'sqlite') {
    return true
  }

  // Check main password (might be using other auth methods)
  // We can't determine if password is actually required without trying to connect

  // Check SSH credentials if tunnel is used
  if (unsanitized.useTunnel && unsanitized.sshTunnel) {
    const hasSSHAuth = unsanitized.sshTunnel?.password ||
                       unsanitized.sshTunnel?.privateKey ||
                       unsanitized.sshTunnel?.privateKeyPath

    if (!hasSSHAuth) {
      return false
    }
  }

  return true
}

/**
 * Restore credentials to a sanitized connection
 * This should only be used in secure contexts where credentials are needed
 */
export function restoreCredentials(
  sanitized: SanitizedConnection,
  credentials: {
    password?: string
    sshPassword?: string
    sshPrivateKey?: string
  }
): DatabaseConnection {
  const restored: any = { ...sanitized }

  // Remove sanitization metadata
  delete restored.passwordRequired
  delete restored.sanitizationFlags
  delete restored.sanitizedAt
  delete restored.sanitizationHash

  // Restore main password
  if (credentials?.password) {
    restored.password = credentials.password
  }

  // Restore SSH credentials
  if (sanitized.sshTunnel) {
    const sshRestored: any = { ...sanitized.sshTunnel }
    delete sshRestored.passwordRequired
    delete sshRestored.privateKeyRequired

    if (credentials?.sshPassword) {
      sshRestored.password = credentials.sshPassword
    }
    if (credentials?.sshPrivateKey) {
      sshRestored.privateKey = credentials.sshPrivateKey
    }

    restored.sshTunnel = sshRestored as SSHTunnelConfig
  }

  return restored as DatabaseConnection
}

/**
 * Generate a hash for connection tracking (not reversible)
 */
function generateConnectionHash(connection: DatabaseConnection): string {
  // Create a stable string representation of the connection
  const parts = [
    connection.type,
    connection.host,
    connection.port?.toString(),
    connection.database,
    connection.username,
    connection.useTunnel ? 'tunnel' : 'direct',
    connection.useVpc ? 'vpc' : 'public'
  ].filter(Boolean).join('|')

  // Simple hash function
  let hash = 0
  for (let i = 0; i < parts.length; i++) {
    const char = parts.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash
  }
  return hash.toString(36)
}

/**
 * Validate that a connection is properly sanitized
 */
export function validateSanitization(
  connection: any,
  config?: SanitizationConfig
): { isValid: boolean; issues: string[] } {
  const issues: string[] = []

  // Check for password field
  if ('password' in connection && connection.password) {
    issues.push('Password field still present')
  }

  // Check SSH tunnel
  if (connection.sshTunnel) {
    if ('password' in connection.sshTunnel && connection.sshTunnel.password) {
      issues.push('SSH password still present')
    }
    if ('privateKey' in connection.sshTunnel && connection.sshTunnel.privateKey) {
      issues.push('SSH private key still present')
    }
  }

  // Deep scan for credentials
  const credentialScan = deepScanConnection(connection, config)
  if (credentialScan.length > 0) {
    credentialScan.forEach(finding => {
      issues.push(`Credential found at ${finding.path}: ${finding.type}`)
    })
  }

  return {
    isValid: issues.length === 0,
    issues
  }
}

/**
 * Deep scan a connection object for any remaining credentials
 */
function deepScanConnection(
  connection: any,
  config?: SanitizationConfig
): Array<{ path: string; type: string; confidence: number }> {
  const findings: Array<{ path: string; type: string; confidence: number }> = []

  function scan(obj: any, path: string): void {
    if (!obj || typeof obj !== 'object') return

    for (const [key, value] of Object.entries(obj)) {
      const currentPath = path ? `${path}.${key}` : key

      if (typeof value === 'string' && value) {
        const credCheck = detectCredentialInValue(value, config ?? getGlobalConfig())
        if (credCheck.isCredential && credCheck.confidence > 0.5) {
          findings.push({
            path: currentPath,
            type: credCheck.type,
            confidence: credCheck.confidence
          })
        }
      } else if (typeof value === 'object' && value !== null) {
        scan(value, currentPath)
      }
    }
  }

  scan(connection, '')
  return findings
}
