/**
 * Connection Export/Import Type Definitions
 *
 * Defines the schema for exporting and importing database connections.
 * See ADR-027 for architecture decisions and rationale.
 *
 * @module lib/export-import/types
 */

import type { SSHAuthMethod } from '@/generated/database'
import type { DatabaseTypeString, SSHTunnelConfig, VPCConfig } from '@/store/connection-store'

// =============================================================================
// Export File Schema
// =============================================================================

/**
 * Root structure of a connection export file
 */
export interface ConnectionExportFile {
  /** Export metadata */
  metadata: ExportMetadata

  /** Exported connection data */
  connections: ExportedConnection[]
}

/**
 * Metadata about the export
 */
export interface ExportMetadata {
  /** Schema version for compatibility checking (semver) */
  version: string

  /** ISO 8601 timestamp of when the export was created */
  exportedAt: string

  /** HowlerOps application version */
  appVersion: string

  /** Optional anonymized user identifier */
  exportedBy?: string

  /** Number of connections in the export */
  connectionCount: number

  /** Whether passwords were included in export (security indicator) */
  includesPasswords: boolean
}

/**
 * A single connection as exported to file
 */
export interface ExportedConnection {
  // Core identification
  /** Original UUID - used for duplicate detection on import */
  id: string

  /** Human-readable connection name */
  name: string

  // Connection type
  /** Database type (postgresql, mysql, etc.) */
  type: DatabaseTypeString

  // Core connection details
  /** Database server hostname or IP */
  host?: string

  /** Database server port */
  port?: number

  /** Database name to connect to */
  database: string

  /** Username for authentication */
  username?: string

  /** SSL/TLS mode setting */
  sslMode?: string

  // Environment tags
  /** Environment labels (dev, staging, prod, etc.) */
  environments?: string[]

  // SSH Tunnel configuration
  /** Whether SSH tunnel is enabled */
  useTunnel?: boolean

  /** SSH tunnel settings (credentials excluded) */
  sshTunnel?: ExportedSSHTunnelConfig

  // VPC configuration
  /** Whether VPC is enabled */
  useVpc?: boolean

  /** VPC settings (sanitized) */
  vpcConfig?: ExportedVPCConfig

  // Database-specific parameters
  /** Additional connection parameters (sanitized) */
  parameters?: Record<string, string>

  // Optional sensitive data
  /**
   * Database password - ONLY included when explicitly requested by user.
   * SSH passwords and private keys are NEVER exported.
   */
  password?: string

  // Import tracking metadata
  /** Metadata about how this connection was imported (optional) */
  metadata?: {
    /** Source of import (env-file, manual, etc.) */
    importedFrom?: string
    /** AI extraction confidence level */
    extractionConfidence?: string
    /** Notes from AI extraction */
    extractionNotes?: string
    /** Detected environment from file */
    detectedEnvironment?: string
    /** How environment was detected */
    environmentSource?: string
  }
}

/**
 * SSH tunnel configuration for export
 * NOTE: password and privateKey content are NEVER exported
 */
export interface ExportedSSHTunnelConfig {
  /** SSH server hostname */
  host: string

  /** SSH server port */
  port: number

  /** SSH username */
  user: string

  /** Authentication method (password, key, agent) */
  authMethod: SSHAuthMethod

  /** Path to private key file (reference only, not content) */
  privateKeyPath?: string

  /** Path to known_hosts file */
  knownHostsPath?: string

  /** Whether to verify host key */
  strictHostKeyChecking: boolean

  /** Connection timeout in seconds */
  timeoutSeconds: number

  /** Keep-alive interval in seconds */
  keepAliveIntervalSeconds: number
}

/**
 * VPC configuration for export
 * NOTE: customConfig is sanitized before export
 */
export interface ExportedVPCConfig {
  /** VPC identifier */
  vpcId: string

  /** Subnet identifier */
  subnetId: string

  /** Security group IDs */
  securityGroupIds: string[]

  /** Private Link service name */
  privateLinkService?: string

  /** Endpoint service name */
  endpointServiceName?: string
}

// =============================================================================
// Export Options
// =============================================================================

/**
 * Options for configuring connection export
 */
export interface ExportOptions {
  /**
   * Whether to include database passwords in export.
   * Default: false (passwords excluded for security)
   *
   * SECURITY NOTE: When true, user must confirm via warning dialog
   */
  includePasswords: boolean

  /**
   * Specific connection IDs to export.
   * If undefined or empty, all connections are exported.
   */
  selectedConnectionIds?: string[]
}

// =============================================================================
// Import Options & Results
// =============================================================================

/**
 * Strategy for handling duplicate connections during import
 */
export type ConflictResolution = 'skip' | 'overwrite' | 'keep-both'

/**
 * Options for configuring connection import
 */
export interface ImportOptions {
  /**
   * How to handle connections with IDs that already exist:
   * - 'skip': Ignore duplicate connections
   * - 'overwrite': Replace existing with imported
   * - 'keep-both': Import with new UUID (creates duplicate)
   */
  conflictResolution: ConflictResolution
}

/**
 * Result of an import operation
 */
export interface ImportResult {
  /** Number of connections successfully imported */
  imported: number

  /** Number of connections skipped due to conflicts */
  skipped: number

  /** Number of existing connections overwritten */
  overwritten: number

  /** Connections that failed to import with reasons */
  failed: ImportFailure[]
}

/**
 * Details about a failed connection import
 */
export interface ImportFailure {
  /** Name of the connection that failed */
  connectionName: string

  /** Original ID from the export file */
  originalId: string

  /** Human-readable reason for failure */
  reason: string
}

// =============================================================================
// Validation Types
// =============================================================================

/**
 * Result of validating an export file or connection
 */
export interface ValidationResult {
  /** Whether validation passed */
  isValid: boolean

  /** Critical errors that prevent import */
  errors: string[]

  /** Non-critical issues (import can proceed) */
  warnings: string[]
}

/**
 * Fields required for a valid connection
 */
export const REQUIRED_CONNECTION_FIELDS = ['id', 'name', 'type', 'database'] as const

/**
 * Valid database type values
 */
export const VALID_DATABASE_TYPES: DatabaseTypeString[] = [
  'postgresql',
  'mysql',
  'sqlite',
  'mssql',
  'mariadb',
  'elasticsearch',
  'opensearch',
  'clickhouse',
  'mongodb',
  'tidb',
]

/**
 * Current export schema version
 */
export const CURRENT_SCHEMA_VERSION = '1.0.0'

// =============================================================================
// UI State Types
// =============================================================================

/**
 * State for the export dialog
 */
export interface ExportDialogState {
  /** Dialog open state */
  isOpen: boolean

  /** Connections available for export */
  availableConnections: Array<{ id: string; name: string; type: DatabaseTypeString }>

  /** Currently selected connection IDs */
  selectedIds: Set<string>

  /** Whether to include passwords */
  includePasswords: boolean

  /** Whether password warning has been acknowledged */
  passwordWarningAcknowledged: boolean

  /** Export in progress */
  isExporting: boolean

  /** Error message if export failed */
  error: string | null
}

/**
 * State for the import dialog
 */
export interface ImportDialogState {
  /** Dialog open state */
  isOpen: boolean

  /** Parsed export file (null if not yet loaded) */
  parsedFile: ConnectionExportFile | null

  /** File validation result */
  validation: ValidationResult | null

  /** IDs of connections that conflict with existing */
  conflictingIds: string[]

  /** Selected conflict resolution strategy */
  conflictResolution: ConflictResolution

  /** Import in progress */
  isImporting: boolean

  /** Import result (null if not yet imported) */
  result: ImportResult | null

  /** Error message if import failed */
  error: string | null
}

// =============================================================================
// Re-exports for convenience
// =============================================================================

export type { DatabaseTypeString, SSHTunnelConfig, VPCConfig, SSHAuthMethod }
