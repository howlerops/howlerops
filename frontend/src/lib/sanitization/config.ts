/**
 * Sanitization Configuration Module
 *
 * User-configurable rules for data sanitization to prevent credential leakage.
 * This module defines privacy modes and configurable rules for Howlerops.
 *
 * @module sanitization/config
 */

export enum PrivacyMode {
  /**
   * Strict mode - Maximum privacy, never sync any potentially sensitive data
   * - All literals removed from queries
   * - All credentials stripped
   * - Minimal metadata shared
   */
  STRICT = 'strict',

  /**
   * Normal mode - Balanced privacy (default)
   * - Sensitive literals removed
   * - Credentials stripped
   * - Safe metadata shared
   */
  NORMAL = 'normal',

  /**
   * Permissive mode - Minimal sanitization
   * - Only obvious credentials removed
   * - More query context preserved
   * WARNING: Use only in trusted environments
   */
  PERMISSIVE = 'permissive'
}

export enum QueryPrivacyLevel {
  /**
   * Private - Never sync, contains sensitive data
   */
  PRIVATE = 'private',

  /**
   * Normal - Sync after sanitization
   */
  NORMAL = 'normal',

  /**
   * Shared - Full sync, no sanitization needed
   */
  SHARED = 'shared'
}

export interface SanitizationConfig {
  /**
   * Current privacy mode
   */
  privacyMode: PrivacyMode

  /**
   * SQL keywords that should trigger private mode
   * Default includes: CREATE USER, ALTER USER, GRANT, etc.
   */
  excludeKeywords: Set<string>

  /**
   * Table names that should trigger private mode
   * Default includes: users, passwords, credentials, tokens, etc.
   */
  excludeTables: Set<string>

  /**
   * Schemas that should trigger private mode
   * Default includes: auth, security, admin, etc.
   */
  excludeSchemas: Set<string>

  /**
   * Patterns for detecting sensitive column names
   * These are treated as regex patterns
   */
  sensitiveColumnPatterns: RegExp[]

  /**
   * Whether to log sanitization operations
   */
  enableLogging: boolean

  /**
   * Whether to fail closed (block) or open (allow) on errors
   * true = block on error (safer)
   * false = allow on error (more permissive)
   */
  failClosed: boolean

  /**
   * Maximum length for strings before they're considered potentially sensitive
   * Longer strings might be credentials or keys
   */
  maxSafeStringLength: number

  /**
   * Patterns for detecting various credential types
   */
  credentialPatterns: {
    apiKey: RegExp[]
    jwt: RegExp[]
    connectionString: RegExp[]
    privateKey: RegExp[]
    password: RegExp[]
  }
}

/**
 * Default excluded keywords that indicate sensitive operations
 */
const DEFAULT_EXCLUDE_KEYWORDS = new Set([
  'CREATE USER',
  'ALTER USER',
  'DROP USER',
  'CREATE ROLE',
  'ALTER ROLE',
  'DROP ROLE',
  'GRANT',
  'REVOKE',
  'SET PASSWORD',
  'IDENTIFIED BY',
  'ENCRYPTED PASSWORD',
  'CREATE LOGIN',
  'ALTER LOGIN',
  'CREATE CREDENTIAL',
  'ALTER CREDENTIAL',
  'CREATE CERTIFICATE',
  'BACKUP CERTIFICATE',
  'CREATE SYMMETRIC KEY',
  'CREATE ASYMMETRIC KEY',
  'CREATE MASTER KEY'
])

/**
 * Default excluded table names that typically contain sensitive data
 */
const DEFAULT_EXCLUDE_TABLES = new Set([
  'users',
  'user',
  'passwords',
  'password',
  'credentials',
  'credential',
  'tokens',
  'token',
  'api_keys',
  'api_key',
  'secrets',
  'secret',
  'auth',
  'authentication',
  'authorization',
  'sessions',
  'session',
  'login',
  'logins',
  'account',
  'accounts',
  'private_key',
  'private_keys',
  'ssh_keys',
  'oauth_tokens',
  'refresh_tokens',
  'access_tokens'
])

/**
 * Default excluded schemas that typically contain sensitive data
 */
const DEFAULT_EXCLUDE_SCHEMAS = new Set([
  'auth',
  'authentication',
  'authorization',
  'security',
  'admin',
  'sys',
  'system',
  'pg_catalog',
  'information_schema',
  'mysql',
  'performance_schema'
])

/**
 * Default patterns for detecting sensitive column names
 */
const DEFAULT_SENSITIVE_COLUMN_PATTERNS = [
  /^pass(word)?$/i,
  /^pwd$/i,
  /^secret$/i,
  /^token$/i,
  /^api_?key$/i,
  /^private_?key$/i,
  /^ssh_?key$/i,
  /^access_?token$/i,
  /^refresh_?token$/i,
  /^auth(entication)?$/i,
  /^credential$/i,
  /^salt$/i,
  /^hash$/i,
  /_pass(word)?$/i,
  /_pwd$/i,
  /_secret$/i,
  /_token$/i,
  /_key$/i
]

/**
 * Default credential detection patterns
 */
const DEFAULT_CREDENTIAL_PATTERNS = {
  // API Keys - long alphanumeric strings with specific patterns
  apiKey: [
    /^[A-Za-z0-9]{32,}$/,  // Generic long alphanumeric
    /^sk-[A-Za-z0-9]{48}$/,  // OpenAI style
    /^[A-Za-z0-9-_]{40,}$/,  // AWS/Google style
    /^Bearer\s+[A-Za-z0-9-_]+/i,
    /^[A-Fa-f0-9]{64}$/,  // SHA256 hash
  ],

  // JWT tokens
  jwt: [
    /^eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$/,  // Standard JWT
    /^Bearer\s+eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$/i,
  ],

  // Database connection strings
  connectionString: [
    /^(postgres|postgresql|mysql|mariadb|mongodb|redis|mssql|oracle):\/\/[^:]+:[^@]+@/i,
    /Data Source=.*password=/i,
    /Server=.*Password=/i,
    /mongodb(\+srv)?:\/\/[^:]+:[^@]+@/i,
  ],

  // SSH Private keys
  privateKey: [
    /-----BEGIN\s+(RSA|DSA|EC|OPENSSH|ENCRYPTED)?\s*PRIVATE KEY-----/,
    /-----BEGIN PGP PRIVATE KEY BLOCK-----/,
  ],

  // Password patterns (complex passwords)
  password: [
    /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&#])[A-Za-z\d@$!%*?&#]{8,}$/,  // Complex password
    /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[A-Za-z\d]{8,}$/,  // Medium complexity
  ]
}

/**
 * Create a default configuration
 */
export function createDefaultConfig(): SanitizationConfig {
  return {
    privacyMode: PrivacyMode.NORMAL,
    excludeKeywords: new Set(DEFAULT_EXCLUDE_KEYWORDS),
    excludeTables: new Set(DEFAULT_EXCLUDE_TABLES),
    excludeSchemas: new Set(DEFAULT_EXCLUDE_SCHEMAS),
    sensitiveColumnPatterns: [...DEFAULT_SENSITIVE_COLUMN_PATTERNS],
    enableLogging: true,
    failClosed: true,
    maxSafeStringLength: 100,
    credentialPatterns: {
      apiKey: [...DEFAULT_CREDENTIAL_PATTERNS.apiKey],
      jwt: [...DEFAULT_CREDENTIAL_PATTERNS.jwt],
      connectionString: [...DEFAULT_CREDENTIAL_PATTERNS.connectionString],
      privateKey: [...DEFAULT_CREDENTIAL_PATTERNS.privateKey],
      password: [...DEFAULT_CREDENTIAL_PATTERNS.password]
    }
  }
}

/**
 * Merge user configuration with defaults
 */
export function mergeConfig(userConfig: Partial<SanitizationConfig>): SanitizationConfig {
  const defaultConfig = createDefaultConfig()

  return {
    privacyMode: userConfig.privacyMode ?? defaultConfig.privacyMode,
    excludeKeywords: userConfig.excludeKeywords
      ? new Set([...DEFAULT_EXCLUDE_KEYWORDS, ...userConfig.excludeKeywords])
      : defaultConfig.excludeKeywords,
    excludeTables: userConfig.excludeTables
      ? new Set([...DEFAULT_EXCLUDE_TABLES, ...userConfig.excludeTables])
      : defaultConfig.excludeTables,
    excludeSchemas: userConfig.excludeSchemas
      ? new Set([...DEFAULT_EXCLUDE_SCHEMAS, ...userConfig.excludeSchemas])
      : defaultConfig.excludeSchemas,
    sensitiveColumnPatterns: userConfig.sensitiveColumnPatterns
      ? [...DEFAULT_SENSITIVE_COLUMN_PATTERNS, ...userConfig.sensitiveColumnPatterns]
      : defaultConfig.sensitiveColumnPatterns,
    enableLogging: userConfig.enableLogging ?? defaultConfig.enableLogging,
    failClosed: userConfig.failClosed ?? defaultConfig.failClosed,
    maxSafeStringLength: userConfig.maxSafeStringLength ?? defaultConfig.maxSafeStringLength,
    credentialPatterns: userConfig.credentialPatterns ? {
      apiKey: [...DEFAULT_CREDENTIAL_PATTERNS.apiKey, ...(userConfig.credentialPatterns.apiKey || [])],
      jwt: [...DEFAULT_CREDENTIAL_PATTERNS.jwt, ...(userConfig.credentialPatterns.jwt || [])],
      connectionString: [...DEFAULT_CREDENTIAL_PATTERNS.connectionString, ...(userConfig.credentialPatterns.connectionString || [])],
      privateKey: [...DEFAULT_CREDENTIAL_PATTERNS.privateKey, ...(userConfig.credentialPatterns.privateKey || [])],
      password: [...DEFAULT_CREDENTIAL_PATTERNS.password, ...(userConfig.credentialPatterns.password || [])]
    } : defaultConfig.credentialPatterns
  }
}

/**
 * Check if a table or column should be excluded based on config
 */
export function shouldExcludeTable(tableName: string, config: SanitizationConfig): boolean {
  const lowerTable = tableName.toLowerCase()
  return config.excludeTables.has(lowerTable)
}

export function shouldExcludeSchema(schemaName: string, config: SanitizationConfig): boolean {
  const lowerSchema = schemaName.toLowerCase()
  return config.excludeSchemas.has(lowerSchema)
}

export function isSensitiveColumn(columnName: string, config: SanitizationConfig): boolean {
  return config.sensitiveColumnPatterns.some(pattern => pattern.test(columnName))
}

/**
 * Singleton instance for global configuration
 */
let globalConfig: SanitizationConfig | null = null

export function getGlobalConfig(): SanitizationConfig {
  if (!globalConfig) {
    globalConfig = createDefaultConfig()
  }
  return globalConfig
}

export function setGlobalConfig(config: Partial<SanitizationConfig>): void {
  globalConfig = mergeConfig(config)
}