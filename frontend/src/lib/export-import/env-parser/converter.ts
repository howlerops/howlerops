/**
 * Connection Converter
 *
 * Converts parsed .env connections to formats compatible
 * with the connection store and import system.
 *
 * @module lib/export-import/env-parser/converter
 */

import type { ParsedEnvConnection } from './types'
import type { ExportedConnection } from '../types'

/**
 * Default ports for each database type
 */
const DEFAULT_PORTS: Record<string, number> = {
  postgresql: 5432,
  mysql: 3306,
  mariadb: 3306,
  mongodb: 27017,
  redis: 6379,
  elasticsearch: 9200,
  opensearch: 9200,
  clickhouse: 9000,
  mssql: 1433,
  sqlite: 0,
}

/**
 * Convert a parsed .env connection to ExportedConnection format
 * for use with the existing import system
 *
 * @param parsed - Parsed connection from AI extraction
 * @returns ExportedConnection ready for import
 */
export function convertToExportedConnection(
  parsed: ParsedEnvConnection
): ExportedConnection {
  // Use detected environment or default to development
  const environment = parsed.detectedEnvironment || 'development'

  return {
    id: parsed.tempId,
    name: parsed.suggestedName,
    type: parsed.type,
    host: parsed.host || 'localhost',
    port: parsed.port || DEFAULT_PORTS[parsed.type] || 5432,
    database: parsed.database || '',
    username: parsed.username,
    password: parsed.password,
    sslMode: mapSslMode(parsed.sslMode),
    environments: [environment],
    metadata: {
      importedFrom: 'env-file',
      extractionConfidence: parsed.overallConfidence,
      extractionNotes: parsed.extractionNotes,
      detectedEnvironment: parsed.detectedEnvironment,
      environmentSource: parsed.environmentSource,
    },
  }
}

/**
 * Convert a parsed .env connection to form data format
 * for use in connection edit forms
 *
 * @param parsed - Parsed connection from AI extraction
 * @returns Form data object for connection form
 */
export function convertToConnectionFormData(parsed: ParsedEnvConnection): {
  name: string
  type: string
  host: string
  port: number
  database: string
  username: string
  password: string
  sslMode: string
  environments: string[]
} {
  return {
    name: parsed.suggestedName,
    type: parsed.type,
    host: parsed.host || 'localhost',
    port: parsed.port || DEFAULT_PORTS[parsed.type] || 5432,
    database: parsed.database || '',
    username: parsed.username || '',
    password: parsed.password || '',
    sslMode: mapSslMode(parsed.sslMode) || 'prefer',
    environments: ['development'],
  }
}

/**
 * Map SSL mode string from .env to standard values
 */
function mapSslMode(
  sslMode: string | undefined
): 'require' | 'prefer' | 'disable' | undefined {
  if (!sslMode) return undefined

  const lower = sslMode.toLowerCase()

  // PostgreSQL SSL modes
  if (lower === 'require' || lower === 'verify-ca' || lower === 'verify-full') {
    return 'require'
  }
  if (lower === 'prefer' || lower === 'allow') {
    return 'prefer'
  }
  if (lower === 'disable' || lower === 'false' || lower === '0' || lower === 'no') {
    return 'disable'
  }

  // Boolean-like values
  if (lower === 'true' || lower === '1' || lower === 'yes') {
    return 'require'
  }

  return 'prefer'
}

/**
 * Parse a connection string URL to extract components
 *
 * @param connectionString - Full connection URL
 * @returns Parsed components or null if invalid
 */
export function parseConnectionString(connectionString: string): {
  type: string
  host: string
  port: number
  database: string
  username?: string
  password?: string
  options?: Record<string, string>
} | null {
  try {
    // Handle mongodb+srv:// specially
    if (connectionString.startsWith('mongodb+srv://')) {
      return parseMongoDBSRV(connectionString)
    }

    // Standard URL parsing
    const url = new URL(connectionString)

    // Determine database type from protocol
    const protocol = url.protocol.replace(':', '')
    const type = mapProtocolToType(protocol)

    // Extract components
    const host = url.hostname
    const port = url.port ? parseInt(url.port, 10) : DEFAULT_PORTS[type] || 5432
    const database = url.pathname.slice(1) // Remove leading /

    // Parse query params as options
    const options: Record<string, string> = {}
    url.searchParams.forEach((value, key) => {
      options[key] = value
    })

    return {
      type,
      host,
      port,
      database,
      username: url.username || undefined,
      password: url.password || undefined,
      options: Object.keys(options).length > 0 ? options : undefined,
    }
  } catch {
    return null
  }
}

/**
 * Parse MongoDB SRV connection string
 */
function parseMongoDBSRV(connectionString: string): {
  type: string
  host: string
  port: number
  database: string
  username?: string
  password?: string
  options?: Record<string, string>
} | null {
  try {
    // Convert mongodb+srv:// to https:// for URL parsing
    const url = new URL(connectionString.replace('mongodb+srv://', 'https://'))

    const options: Record<string, string> = {}
    url.searchParams.forEach((value, key) => {
      options[key] = value
    })

    return {
      type: 'mongodb',
      host: url.hostname,
      port: 27017, // SRV uses DNS to discover actual ports
      database: url.pathname.slice(1) || 'admin',
      username: url.username || undefined,
      password: url.password || undefined,
      options: Object.keys(options).length > 0 ? options : undefined,
    }
  } catch {
    return null
  }
}

/**
 * Map URL protocol to database type
 */
function mapProtocolToType(protocol: string): string {
  switch (protocol.toLowerCase()) {
    case 'postgres':
    case 'postgresql':
      return 'postgresql'
    case 'mysql':
      return 'mysql'
    case 'mariadb':
      return 'mariadb'
    case 'mongodb':
      return 'mongodb'
    case 'redis':
    case 'rediss':
      return 'redis'
    case 'clickhouse':
    case 'ch':
      return 'clickhouse'
    case 'sqlserver':
    case 'mssql':
      return 'mssql'
    default:
      return 'postgresql'
  }
}

/**
 * Build a connection string from parsed components
 *
 * @param parsed - Parsed connection from AI extraction
 * @returns Connection string URL
 */
export function buildConnectionString(parsed: ParsedEnvConnection): string {
  const type = parsed.type
  const host = parsed.host || 'localhost'
  const port = parsed.port || DEFAULT_PORTS[type] || 5432
  const database = parsed.database || ''

  let protocol: string
  switch (type) {
    case 'postgresql':
      protocol = 'postgresql'
      break
    case 'mysql':
    case 'mariadb':
      protocol = 'mysql'
      break
    case 'mongodb':
      protocol = 'mongodb'
      break
    // Redis not yet supported
    case 'clickhouse':
      protocol = 'clickhouse'
      break
    case 'mssql':
      protocol = 'sqlserver'
      break
    default:
      protocol = 'postgresql'
  }

  let auth = ''
  if (parsed.username) {
    auth = parsed.password
      ? `${encodeURIComponent(parsed.username)}:${encodeURIComponent(parsed.password)}@`
      : `${encodeURIComponent(parsed.username)}@`
  }

  return `${protocol}://${auth}${host}:${port}/${database}`
}
