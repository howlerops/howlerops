/**
 * Config Export/Import API Client
 *
 * HTTP client for exporting and importing database configurations.
 * Allows users to backup/restore connections, saved queries, and tags.
 * Supports both unencrypted (no passwords) and encrypted (with passwords) exports.
 *
 * @module lib/api/config-export
 */

import { authFetch } from './auth-client'

// ============================================================================
// Encrypted Export Types (with passwords)
// ============================================================================

/**
 * Encrypted export format identifier
 */
export const ENCRYPTED_EXPORT_FORMAT = 'howlerops-encrypted-config-v1'

/**
 * Metadata hint for encrypted exports (visible without decryption)
 */
export interface EncryptedExportHint {
  connection_count: number
  query_count: number
  exported_by?: string
  database_types?: string[]
}

/**
 * Encrypted config export (passwords protected with passphrase)
 */
export interface EncryptedConfigExport {
  format: string
  algorithm: string
  salt: string
  nonce: string
  ciphertext: string
  created_at: string
  export_id: string
  hint: EncryptedExportHint
}

/**
 * Credential with password (only in decrypted exports)
 */
export interface ExportedCredential extends ExportedConnection {
  password?: string
}

/**
 * Decrypted payload from encrypted export
 */
export interface DecryptedExportPayload {
  credentials: ExportedCredential[]
  saved_queries?: ExportedSavedQuery[]
  tags?: string[]
  folders?: string[]
  exported_at: string
  exported_by?: string
  app_version?: string
}

// ============================================================================
// Types
// ============================================================================

/**
 * Options for configuring what to include in the export
 */
export interface ConfigExportOptions {
  /** Include connections (without passwords) */
  include_connections: boolean
  /** Include saved queries */
  include_saved_queries: boolean
  /** Include query history (sanitized) */
  include_query_history: boolean
  /** Only export specific connection IDs */
  connection_ids?: string[]
  /** Only export queries with these tags */
  query_tags?: string[]
  /** Include shared resources */
  include_shared: boolean
  /** Export metadata only (no query SQL) */
  metadata_only: boolean
  /** Anonymize hostnames */
  anonymize_hosts: boolean
}

/**
 * Conflict handling strategies for import
 */
export type ConflictStrategy = 'skip' | 'overwrite' | 'rename' | 'merge'

/**
 * Options for configuring the import behavior
 */
export interface ConfigImportOptions {
  /** The config data to import */
  config: ExportedConfig
  /** How to handle conflicts */
  conflict_strategy: ConflictStrategy
  /** Import connections */
  import_connections: boolean
  /** Import saved queries */
  import_saved_queries: boolean
  /** Only import specific connection export IDs */
  connection_export_ids?: string[]
  /** Only import queries with these tags */
  query_tags?: string[]
  /** Host overrides (export_id -> actual_host) */
  host_overrides?: Record<string, string>
  /** Share with this organization */
  share_with_organization?: string
  /** Dry run (validate only) */
  dry_run: boolean
}

/**
 * Exported connection (without password)
 */
export interface ExportedConnection {
  export_id: string
  original_id?: string
  name: string
  type: string
  host: string
  port: number
  database: string
  username: string
  environment?: string
  ssl_config_keys?: string[]
  metadata?: Record<string, string>
  was_shared: boolean
  created_at: string
}

/**
 * Exported saved query
 */
export interface ExportedSavedQuery {
  export_id: string
  original_id?: string
  name: string
  description?: string
  query?: string
  connection_export_id?: string
  connection_type?: string
  folder?: string
  tags?: string[]
  metadata?: Record<string, string>
  favorite: boolean
  was_shared: boolean
  created_at: string
  updated_at: string
}

/**
 * Exported query history (sanitized)
 */
export interface ExportedQueryHistory {
  connection_export_id?: string
  executed_at: string
  duration_ms: number
  rows_returned: number
  success: boolean
  error?: string
}

/**
 * Complete exported configuration
 */
export interface ExportedConfig {
  format: string
  exported_at: string
  exported_by?: string
  app_version?: string
  export_options: ConfigExportOptions
  connections?: ExportedConnection[]
  saved_queries?: ExportedSavedQuery[]
  query_history?: ExportedQueryHistory[]
  tags?: string[]
  folders?: string[]
}

/**
 * Import error detail
 */
export interface ImportError {
  export_id: string
  name: string
  error: string
}

/**
 * Connection needing password
 */
export interface PasswordRequired {
  new_connection_id: string
  export_id: string
  name: string
  host: string
  database: string
}

/**
 * Result of an import operation
 */
export interface ImportResult {
  dry_run: boolean
  connections_imported: number
  connections_skipped: number
  connection_errors?: ImportError[]
  connection_id_map: Record<string, string>
  queries_imported: number
  queries_skipped: number
  query_errors?: ImportError[]
  new_tags?: string[]
  new_folders?: string[]
  connections_needing_passwords: PasswordRequired[]
  warnings?: string[]
}

/**
 * Validation result
 */
export interface ValidationResult {
  valid: boolean
  error?: string
  issues: string[]
  connections: number
  queries: number
  tags: string[]
  folders: string[]
  exported_at: string
  exported_by: string
}

/**
 * API response wrapper
 */
interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
  result?: T
  preview?: T
}

// ============================================================================
// Default Options
// ============================================================================

/**
 * Default export options
 */
export const defaultExportOptions: ConfigExportOptions = {
  include_connections: true,
  include_saved_queries: true,
  include_query_history: false,
  include_shared: false,
  metadata_only: false,
  anonymize_hosts: false,
}

/**
 * Default import options
 */
export const defaultImportOptions: Omit<ConfigImportOptions, 'config'> = {
  conflict_strategy: 'skip',
  import_connections: true,
  import_saved_queries: true,
  dry_run: false,
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * Export user configuration to JSON
 */
export async function exportConfig(
  options: Partial<ConfigExportOptions> = {}
): Promise<ExportedConfig> {
  const mergedOptions = { ...defaultExportOptions, ...options }

  const response = await authFetch<ExportedConfig>(
    '/api/config/export',
    {
      method: 'POST',
      body: JSON.stringify(mergedOptions),
    }
  )

  return response
}

/**
 * Download config as a file
 */
export async function downloadConfigFile(
  options: Partial<ConfigExportOptions> = {}
): Promise<void> {
  const config = await exportConfig(options)
  const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `howlerops-config-${new Date().toISOString().split('T')[0]}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Validate a config file before import
 */
export async function validateConfig(
  config: ExportedConfig | string
): Promise<ValidationResult> {
  const configData = typeof config === 'string' ? config : JSON.stringify(config)

  const response = await authFetch<ValidationResult>(
    '/api/config/validate',
    {
      method: 'POST',
      body: configData,
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )

  return response
}

/**
 * Preview what an import would do (dry run)
 */
export async function previewImport(
  config: ExportedConfig,
  options: Partial<Omit<ConfigImportOptions, 'config' | 'dry_run'>> = {}
): Promise<ImportResult> {
  const response = await authFetch<ApiResponse<ImportResult>>(
    '/api/config/preview',
    {
      method: 'POST',
      body: JSON.stringify({
        config,
        ...defaultImportOptions,
        ...options,
        dry_run: true,
      }),
    }
  )

  if (!response.preview) {
    throw new Error('Failed to preview import')
  }

  return response.preview
}

/**
 * Import configuration
 */
export async function importConfig(
  config: ExportedConfig,
  options: Partial<Omit<ConfigImportOptions, 'config'>> = {}
): Promise<ImportResult> {
  const response = await authFetch<ApiResponse<ImportResult>>(
    '/api/config/import',
    {
      method: 'POST',
      body: JSON.stringify({
        config,
        ...defaultImportOptions,
        ...options,
      }),
    }
  )

  if (!response.result) {
    throw new Error(response.message || 'Failed to import config')
  }

  return response.result
}

/**
 * Parse a config file from string
 */
export function parseConfigFile(content: string): ExportedConfig {
  try {
    const config = JSON.parse(content) as ExportedConfig
    if (config.format !== 'howlerops-config-v1') {
      throw new Error(`Unsupported config format: ${config.format}`)
    }
    return config
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error('Invalid JSON format')
    }
    throw error
  }
}

/**
 * Read a config file from a File object
 */
export async function readConfigFile(file: File): Promise<ExportedConfig> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string
        resolve(parseConfigFile(content))
      } catch (error) {
        reject(error)
      }
    }
    reader.onerror = () => reject(new Error('Failed to read file'))
    reader.readAsText(file)
  })
}

// ============================================================================
// Encrypted Export Functions (with passwords)
// ============================================================================

/**
 * Export configuration WITH passwords (encrypted with passphrase)
 *
 * SECURITY: The passphrase must be strong (12+ chars, mixed types).
 * The exported file will be encrypted with Argon2id + AES-256-GCM.
 */
export async function exportEncryptedConfig(
  options: Partial<ConfigExportOptions> = {},
  passphrase: string
): Promise<EncryptedConfigExport> {
  const mergedOptions = { ...defaultExportOptions, ...options }

  const response = await authFetch<ApiResponse<EncryptedConfigExport>>(
    '/api/config/export/encrypted',
    {
      method: 'POST',
      body: JSON.stringify({
        ...mergedOptions,
        passphrase,
      }),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to export encrypted config')
  }

  return response.data
}

/**
 * Download encrypted config as a file
 */
export async function downloadEncryptedConfigFile(
  options: Partial<ConfigExportOptions> = {},
  passphrase: string
): Promise<void> {
  const config = await exportEncryptedConfig(options, passphrase)
  const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `howlerops-encrypted-config-${new Date().toISOString().split('T')[0]}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Decrypt an encrypted config export
 */
export async function decryptConfig(
  encrypted: EncryptedConfigExport,
  passphrase: string
): Promise<DecryptedExportPayload> {
  const response = await authFetch<ApiResponse<DecryptedExportPayload>>(
    '/api/config/decrypt',
    {
      method: 'POST',
      body: JSON.stringify({
        encrypted,
        passphrase,
      }),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Decryption failed - wrong passphrase?')
  }

  return response.data
}

/**
 * Import an encrypted config (will prompt for passwords to re-encrypt)
 */
export async function importEncryptedConfig(
  encrypted: EncryptedConfigExport,
  passphrase: string,
  options: Partial<Omit<ConfigImportOptions, 'config'>> = {}
): Promise<ImportResult> {
  const response = await authFetch<ApiResponse<ImportResult>>(
    '/api/config/import/encrypted',
    {
      method: 'POST',
      body: JSON.stringify({
        encrypted,
        passphrase,
        ...defaultImportOptions,
        ...options,
      }),
    }
  )

  if (!response.result) {
    throw new Error(response.message || 'Failed to import encrypted config')
  }

  return response.result
}

/**
 * Parse an encrypted config file from string
 */
export function parseEncryptedConfigFile(content: string): EncryptedConfigExport {
  try {
    const config = JSON.parse(content) as EncryptedConfigExport
    if (config.format !== ENCRYPTED_EXPORT_FORMAT) {
      throw new Error(`Unsupported format: ${config.format}`)
    }
    return config
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error('Invalid JSON format')
    }
    throw error
  }
}

/**
 * Read an encrypted config file from a File object
 */
export async function readEncryptedConfigFile(file: File): Promise<EncryptedConfigExport> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string
        resolve(parseEncryptedConfigFile(content))
      } catch (error) {
        reject(error)
      }
    }
    reader.onerror = () => reject(new Error('Failed to read file'))
    reader.readAsText(file)
  })
}

/**
 * Check if a file is an encrypted export
 */
export function isEncryptedExport(content: string): boolean {
  try {
    const parsed = JSON.parse(content)
    return parsed.format === ENCRYPTED_EXPORT_FORMAT
  } catch {
    return false
  }
}

/**
 * Validate passphrase strength (client-side check)
 */
export function validatePassphraseStrength(passphrase: string): {
  valid: boolean
  errors: string[]
} {
  const errors: string[] = []

  if (passphrase.length < 12) {
    errors.push('Passphrase must be at least 12 characters')
  }

  const hasUpper = /[A-Z]/.test(passphrase)
  const hasLower = /[a-z]/.test(passphrase)
  const hasDigit = /[0-9]/.test(passphrase)
  const hasSpecial = /[^A-Za-z0-9]/.test(passphrase)

  const complexity = [hasUpper, hasLower, hasDigit, hasSpecial].filter(Boolean).length
  if (complexity < 2) {
    errors.push('Use at least 2 of: uppercase, lowercase, numbers, special characters')
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}
