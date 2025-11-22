/**
 * SQL Query Sanitization Module
 *
 * Removes sensitive data from SQL queries while preserving structure.
 * CRITICAL: Must prevent ALL credential leakage in queries.
 *
 * @module sanitization/query-sanitizer
 */

import {
  getGlobalConfig,
  PrivacyMode,
  QueryPrivacyLevel,
  SanitizationConfig,
  shouldExcludeSchema,
  shouldExcludeTable} from './config'
import { detectCredentials, mightBeCredential } from './credential-detector'

export interface QuerySanitizationResult {
  /**
   * The sanitized query with sensitive data removed
   */
  sanitizedQuery: string

  /**
   * Privacy level determined for this query
   */
  privacyLevel: QueryPrivacyLevel

  /**
   * Whether any modifications were made
   */
  wasModified: boolean

  /**
   * Reasons for the privacy level assigned
   */
  reasons: string[]

  /**
   * Statistics about what was sanitized
   */
  stats: {
    literalsRemoved: number
    credentialsDetected: number
    sensitiveTablesFound: number
    sensitiveOperationsFound: number
  }

  /**
   * Original query hash for tracking (not the query itself)
   */
  originalHash?: string
}

/**
 * Token types for SQL parsing
 */
enum TokenType {
  KEYWORD = 'keyword',
  IDENTIFIER = 'identifier',
  STRING_LITERAL = 'string_literal',
  NUMERIC_LITERAL = 'numeric_literal',
  OPERATOR = 'operator',
  COMMENT = 'comment',
  WHITESPACE = 'whitespace',
  OTHER = 'other'
}

interface Token {
  type: TokenType
  value: string
  start: number
  end: number
}

/**
 * Simple SQL tokenizer
 */
class SQLTokenizer {
  private query: string
  private position: number
  private tokens: Token[]

  constructor(query: string) {
    this.query = query
    this.position = 0
    this.tokens = []
  }

  tokenize(): Token[] {
    while (this.position < this.query.length) {
      this.skipWhitespace()
      if (this.position >= this.query.length) break

      if (this.tryComment()) continue
      if (this.tryStringLiteral()) continue
      if (this.tryNumericLiteral()) continue
      if (this.tryIdentifier()) continue
      if (this.tryOperator()) continue

      // Unknown character, treat as other
      this.tokens.push({
        type: TokenType.OTHER,
        value: this.query[this.position],
        start: this.position,
        end: this.position + 1
      })
      this.position++
    }

    return this.tokens
  }

  private skipWhitespace(): void {
    const start = this.position
    while (this.position < this.query.length && /\s/.test(this.query[this.position])) {
      this.position++
    }
    if (this.position > start) {
      this.tokens.push({
        type: TokenType.WHITESPACE,
        value: this.query.substring(start, this.position),
        start,
        end: this.position
      })
    }
  }

  private tryComment(): boolean {
    // Single line comment --
    if (this.query.substr(this.position, 2) === '--') {
      const start = this.position
      this.position += 2
      while (this.position < this.query.length && this.query[this.position] !== '\n') {
        this.position++
      }
      this.tokens.push({
        type: TokenType.COMMENT,
        value: this.query.substring(start, this.position),
        start,
        end: this.position
      })
      return true
    }

    // Multi-line comment /* */
    if (this.query.substr(this.position, 2) === '/*') {
      const start = this.position
      this.position += 2
      while (this.position < this.query.length - 1) {
        if (this.query.substr(this.position, 2) === '*/') {
          this.position += 2
          break
        }
        this.position++
      }
      this.tokens.push({
        type: TokenType.COMMENT,
        value: this.query.substring(start, this.position),
        start,
        end: this.position
      })
      return true
    }

    return false
  }

  private tryStringLiteral(): boolean {
    const quotes = ["'", '"', '`']
    const currentChar = this.query[this.position]

    if (!quotes.includes(currentChar)) return false

    const quote = currentChar
    const start = this.position
    this.position++ // Skip opening quote

    let escaped = false
    while (this.position < this.query.length) {
      const char = this.query[this.position]

      if (escaped) {
        escaped = false
        this.position++
        continue
      }

      if (char === '\\') {
        escaped = true
        this.position++
        continue
      }

      if (char === quote) {
        // Check for doubled quote (SQL escape)
        if (this.position + 1 < this.query.length && this.query[this.position + 1] === quote) {
          this.position += 2
          continue
        }
        this.position++ // Include closing quote
        break
      }

      this.position++
    }

    this.tokens.push({
      type: TokenType.STRING_LITERAL,
      value: this.query.substring(start, this.position),
      start,
      end: this.position
    })

    return true
  }

  private tryNumericLiteral(): boolean {
    const start = this.position
    let hasDigit = false
    let hasDot = false
    let hasE = false

    // Check for negative sign
    if (this.query[this.position] === '-' || this.query[this.position] === '+') {
      this.position++
    }

    while (this.position < this.query.length) {
      const char = this.query[this.position]

      if (/\d/.test(char)) {
        hasDigit = true
        this.position++
      } else if (char === '.' && !hasDot && !hasE) {
        hasDot = true
        this.position++
      } else if ((char === 'e' || char === 'E') && hasDigit && !hasE) {
        hasE = true
        this.position++
        // Allow +/- after E
        if (this.position < this.query.length &&
            (this.query[this.position] === '+' || this.query[this.position] === '-')) {
          this.position++
        }
      } else {
        break
      }
    }

    if (hasDigit && this.position > start) {
      this.tokens.push({
        type: TokenType.NUMERIC_LITERAL,
        value: this.query.substring(start, this.position),
        start,
        end: this.position
      })
      return true
    }

    this.position = start // Reset if not a number
    return false
  }

  private tryIdentifier(): boolean {
    const start = this.position

    // Check for quoted identifier
    if (this.query[this.position] === '"' || this.query[this.position] === '`' ||
        this.query[this.position] === '[') {
      const closeChar = this.query[this.position] === '[' ? ']' : this.query[this.position]
      this.position++
      while (this.position < this.query.length) {
        if (this.query[this.position] === closeChar) {
          this.position++
          break
        }
        this.position++
      }
      const value = this.query.substring(start, this.position)
      const upperValue = value.toUpperCase()

      this.tokens.push({
        type: this.isKeyword(upperValue) ? TokenType.KEYWORD : TokenType.IDENTIFIER,
        value,
        start,
        end: this.position
      })
      return true
    }

    // Regular identifier
    if (/[a-zA-Z_]/.test(this.query[this.position])) {
      while (this.position < this.query.length &&
             /[a-zA-Z0-9_$]/.test(this.query[this.position])) {
        this.position++
      }

      const value = this.query.substring(start, this.position)
      const upperValue = value.toUpperCase()

      this.tokens.push({
        type: this.isKeyword(upperValue) ? TokenType.KEYWORD : TokenType.IDENTIFIER,
        value,
        start,
        end: this.position
      })
      return true
    }

    return false
  }

  private tryOperator(): boolean {
    const operators = ['<=', '>=', '<>', '!=', '||', '::', '->>', '->', '::']
    const singleOps = ['=', '<', '>', '+', '-', '*', '/', '%', '(', ')', ',', ';', '.', ':']

    // Try multi-character operators first
    for (const op of operators) {
      if (this.query.substr(this.position, op.length) === op) {
        this.tokens.push({
          type: TokenType.OPERATOR,
          value: op,
          start: this.position,
          end: this.position + op.length
        })
        this.position += op.length
        return true
      }
    }

    // Try single character operators
    const char = this.query[this.position]
    if (singleOps.includes(char)) {
      this.tokens.push({
        type: TokenType.OPERATOR,
        value: char,
        start: this.position,
        end: this.position + 1
      })
      this.position++
      return true
    }

    return false
  }

  private isKeyword(word: string): boolean {
    const keywords = new Set([
      'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER',
      'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE', 'CREATE', 'ALTER',
      'DROP', 'TABLE', 'DATABASE', 'SCHEMA', 'INDEX', 'VIEW', 'PROCEDURE',
      'FUNCTION', 'TRIGGER', 'GRANT', 'REVOKE', 'USER', 'ROLE', 'PASSWORD',
      'IDENTIFIED', 'BY', 'WITH', 'AS', 'ON', 'AND', 'OR', 'NOT', 'IN', 'EXISTS',
      'BETWEEN', 'LIKE', 'IS', 'NULL', 'TRUE', 'FALSE', 'CASE', 'WHEN', 'THEN',
      'ELSE', 'END', 'GROUP', 'ORDER', 'HAVING', 'LIMIT', 'OFFSET', 'UNION',
      'INTERSECT', 'EXCEPT', 'ALL', 'DISTINCT', 'UNIQUE', 'PRIMARY', 'KEY',
      'FOREIGN', 'REFERENCES', 'CASCADE', 'RESTRICT', 'DEFAULT', 'CHECK',
      'CONSTRAINT', 'BEGIN', 'COMMIT', 'ROLLBACK', 'TRANSACTION', 'IF',
      'USE', 'SHOW', 'DESCRIBE', 'DESC', 'EXPLAIN', 'ANALYZE'
    ])
    return keywords.has(word)
  }
}

/**
 * Main query sanitization function
 */
export function sanitizeQuery(
  query: string,
  config: SanitizationConfig = getGlobalConfig()
): QuerySanitizationResult {
  const result: QuerySanitizationResult = {
    sanitizedQuery: query,
    privacyLevel: QueryPrivacyLevel.NORMAL,
    wasModified: false,
    reasons: [],
    stats: {
      literalsRemoved: 0,
      credentialsDetected: 0,
      sensitiveTablesFound: 0,
      sensitiveOperationsFound: 0
    }
  }

  // Empty query is safe
  if (!query || query.trim().length === 0) {
    return result
  }

  // Tokenize the query
  const tokenizer = new SQLTokenizer(query)
  const tokens = tokenizer.tokenize()

  // Analyze tokens for sensitive content
  const sanitizedTokens: Token[] = []
  const clauseContextKeywords = new Set(['WHERE', 'HAVING', 'SET', 'VALUES', 'ON', 'USING'])
  const clauseTerminators = new Set(['GROUP', 'ORDER', 'LIMIT', 'RETURNING', 'UNION', 'EXCEPT', 'INTERSECT', 'END'])
  const tableContextKeywords = new Set(['FROM', 'JOIN', 'UPDATE', 'INTO', 'TABLE', 'DELETE', 'USING'])
  const sensitiveKeywords = new Set(['GRANT', 'REVOKE', 'PASSWORD', 'IDENTIFIED'])
  const sensitiveCombinations = new Set([
    'CREATE USER',
    'ALTER USER',
    'DROP USER',
    'CREATE ROLE',
    'ALTER ROLE',
    'DROP ROLE',
    'CREATE LOGIN',
    'ALTER LOGIN',
    'SET PASSWORD'
  ])
  const reasonsSeen = new Set<string>()

  const normalizeIdentifier = (raw: string): string => {
    let value = raw.trim()
    if (!value) {
      return value
    }
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith('`') && value.endsWith('`')) ||
      (value.startsWith('[') && value.endsWith(']'))
    ) {
      value = value.slice(1, -1)
    }
    return value
  }

  const sensitiveFragments = ['password', 'credential', 'secret', 'token', 'auth', 'key']

  const isSensitiveTableName = (name: string): boolean => {
    if (!name) {
      return false
    }
    if (shouldExcludeTable(name, config)) {
      return true
    }
    const lower = name.toLowerCase()
    if (sensitiveFragments.some(fragment => lower.includes(fragment))) {
      return true
    }
    return config.sensitiveColumnPatterns.some(pattern => pattern.test(name))
  }

  const isSensitiveSchemaName = (name: string): boolean => {
    if (!name) {
      return false
    }
    if (shouldExcludeSchema(name, config)) {
      return true
    }
    const lower = name.toLowerCase()
    return sensitiveFragments.some(fragment => lower.includes(fragment))
  }

  const markPrivate = (reason: string) => {
    if (!reasonsSeen.has(reason)) {
      reasonsSeen.add(reason)
      if (reason) {
        result.reasons.push(reason)
      }
    }
    result.privacyLevel = QueryPrivacyLevel.PRIVATE
  }

  let lastKeyword: string | null = null
  let clause: string | null = null
  let sensitiveContext = false
  let pendingSchema: string | null = null
  let lastTokenWasTable = false
  let lastIdentifierInfo: { name: string; sensitive: boolean } | null = null

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]
    const nextToken = i < tokens.length - 1 ? tokens[i + 1] : null
    const prevToken = i > 0 ? tokens[i - 1] : null

    if (token.type === TokenType.KEYWORD) {
      const currentKeyword = token.value.toUpperCase()
      const combined = lastKeyword ? `${lastKeyword} ${currentKeyword}` : ''
      let markedSensitive = false

      if (sensitiveCombinations.has(combined)) {
        markPrivate(`Sensitive operation detected: ${combined}`)
        result.stats.sensitiveOperationsFound++
        result.stats.credentialsDetected++
        sensitiveContext = true
        markedSensitive = true
      }

      if (!markedSensitive && sensitiveKeywords.has(currentKeyword)) {
        markPrivate(`Sensitive keyword: ${currentKeyword}`)
        result.stats.sensitiveOperationsFound++
        result.stats.credentialsDetected++
        sensitiveContext = true
        markedSensitive = true
      }

      if (clauseContextKeywords.has(currentKeyword)) {
        clause = currentKeyword
      } else if (clauseTerminators.has(currentKeyword)) {
        clause = null
        sensitiveContext = false
        pendingSchema = null
      }

      if (['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'WITH', 'CREATE'].includes(currentKeyword)) {
        if (!clauseContextKeywords.has(currentKeyword)) {
          clause = null
        }
        pendingSchema = null
        lastTokenWasTable = false
      }

      lastKeyword = currentKeyword
      lastTokenWasTable = false
      lastIdentifierInfo = null
      sanitizedTokens.push(token)
      continue
    }

    if (token.type === TokenType.IDENTIFIER) {
      const cleanName = normalizeIdentifier(token.value)

      if (nextToken?.type === TokenType.OPERATOR && nextToken.value === '.') {
        pendingSchema = cleanName
        sanitizedTokens.push(token)
        lastTokenWasTable = false
        lastIdentifierInfo = null
        continue
      }

      if (prevToken?.type === TokenType.OPERATOR && prevToken.value === '.') {
        if (pendingSchema && isSensitiveSchemaName(pendingSchema)) {
          markPrivate(`Sensitive schema: ${pendingSchema}`)
          result.stats.sensitiveTablesFound++
        }
        pendingSchema = null

        if (isSensitiveTableName(cleanName)) {
          markPrivate(`Sensitive table: ${cleanName}`)
          result.stats.sensitiveTablesFound++
        }

        sanitizedTokens.push(token)
        lastTokenWasTable = true
        lastIdentifierInfo = null
        continue
      }

      if (tableContextKeywords.has(lastKeyword ?? '')) {
        if (pendingSchema && isSensitiveSchemaName(pendingSchema)) {
          markPrivate(`Sensitive schema: ${pendingSchema}`)
          result.stats.sensitiveTablesFound++
        }
        pendingSchema = null

        if (isSensitiveTableName(cleanName)) {
          markPrivate(`Sensitive table: ${cleanName}`)
          result.stats.sensitiveTablesFound++
        }

        sanitizedTokens.push(token)
        lastTokenWasTable = true
        lastIdentifierInfo = null
        continue
      }

      if (lastTokenWasTable) {
        lastTokenWasTable = false
        sanitizedTokens.push(token)
        lastIdentifierInfo = null
        continue
      }

      const isSensitiveColumn = config.sensitiveColumnPatterns.some(pattern => pattern.test(cleanName))
      if (isSensitiveColumn) {
        markPrivate(`Sensitive column referenced: ${cleanName}`)
        result.stats.credentialsDetected++
        sensitiveContext = true
      }

      sanitizedTokens.push(token)
      lastTokenWasTable = false
      lastIdentifierInfo = { name: cleanName, sensitive: isSensitiveColumn }
      continue
    }

    if (token.type === TokenType.STRING_LITERAL) {
      const content = token.value.slice(1, -1)
      const credCheck = detectCredentials(content, config)
      if (credCheck.isCredential) {
        result.stats.credentialsDetected++
        markPrivate(`Credential detected in string literal: ${credCheck.type}`)
        sensitiveContext = true
      }

      let shouldSanitize = false
      if (config.privacyMode === PrivacyMode.STRICT) {
        shouldSanitize = true
      } else if (config.privacyMode === PrivacyMode.NORMAL) {
        shouldSanitize =
          credCheck.isCredential ||
          sensitiveContext ||
          (clause !== null && clauseContextKeywords.has(clause)) ||
          mightBeCredential(content)
      } else {
        shouldSanitize = credCheck.isCredential || sensitiveContext
      }

      if (shouldSanitize) {
        if (!credCheck.isCredential && lastIdentifierInfo?.sensitive) {
          result.stats.credentialsDetected++
        }
        sanitizedTokens.push({ ...token, value: '?' })
        result.wasModified = true
        result.stats.literalsRemoved++
      } else {
        sanitizedTokens.push(token)
      }

      lastTokenWasTable = false
      lastIdentifierInfo = null
      continue
    }

    if (token.type === TokenType.NUMERIC_LITERAL) {
      let shouldSanitize = false
      if (config.privacyMode === PrivacyMode.STRICT) {
        shouldSanitize = true
      } else if (config.privacyMode === PrivacyMode.NORMAL) {
        shouldSanitize =
          sensitiveContext ||
          (clause !== null && ['WHERE', 'HAVING', 'ON', 'USING'].includes(clause))
      }

      if (shouldSanitize) {
        sanitizedTokens.push({ ...token, value: '?' })
        result.wasModified = true
        result.stats.literalsRemoved++
      } else {
        sanitizedTokens.push(token)
      }

      lastTokenWasTable = false
      lastIdentifierInfo = null
      continue
    }

    if (token.type === TokenType.COMMENT) {
      const credCheck = detectCredentials(token.value, config)
      if (credCheck.isCredential) {
        result.wasModified = true
        result.stats.credentialsDetected++
        markPrivate('Credential detected in comment')
        continue
      }

      sanitizedTokens.push(token)
      lastTokenWasTable = false
      lastIdentifierInfo = null
      continue
    }

    if (token.type === TokenType.OPERATOR && token.value === ';') {
      clause = null
      sensitiveContext = false
      pendingSchema = null
      lastTokenWasTable = false
      lastIdentifierInfo = null
    }

    sanitizedTokens.push(token)
    lastTokenWasTable = false
    if (token.type !== TokenType.WHITESPACE) {
      lastIdentifierInfo = null
    }
  }

  // Reconstruct the query from sanitized tokens
  if (result.wasModified) {
    result.sanitizedQuery = sanitizedTokens.map(t => t.value).join('')
  }

  // Final check: scan entire query for missed credentials
  const fullScan = detectCredentials(result.sanitizedQuery, config)
  if (fullScan.isCredential && fullScan.confidence > 0.7) {
    result.privacyLevel = QueryPrivacyLevel.PRIVATE
    result.reasons.push('Query still contains potential credentials after sanitization')

    // If we still have credentials, return a heavily sanitized version
    if (config.failClosed) {
      result.sanitizedQuery = '-- Query blocked: contains credentials'
      result.wasModified = true
    }
  }

  // Generate a hash of the original query for tracking
  if (result.wasModified) {
    result.originalHash = hashQuery(query)
  }

  return result
}

/**
 * Batch sanitization for multiple queries
 */
export function sanitizeQueries(
  queries: string[],
  config?: SanitizationConfig
): QuerySanitizationResult[] {
  return queries.map(query => sanitizeQuery(query, config))
}

/**
 * Check if a query should be considered private
 */
export function isPrivateQuery(query: string, config?: SanitizationConfig): boolean {
  const result = sanitizeQuery(query, config)
  return result.privacyLevel === QueryPrivacyLevel.PRIVATE
}

/**
 * Generate a hash for query tracking (not reversible)
 */
function hashQuery(query: string): string {
  // Simple hash function for demo - in production, use crypto
  let hash = 0
  for (let i = 0; i < query.length; i++) {
    const char = query.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash // Convert to 32-bit integer
  }
  return hash.toString(36)
}

/**
 * Extract and sanitize query metadata
 */
export interface QueryMetadata {
  tables: string[]
  schemas: string[]
  operations: string[]
  hasCredentials: boolean
  privacyLevel: QueryPrivacyLevel
}

export function extractQueryMetadata(
  query: string,
  config?: SanitizationConfig
): QueryMetadata {
  const sanitizationResult = sanitizeQuery(query, config)
  const tokenizer = new SQLTokenizer(query)
  const tokens = tokenizer.tokenize()

  const metadata: QueryMetadata = {
    tables: [],
    schemas: [],
    operations: [],
    hasCredentials: sanitizationResult.stats.credentialsDetected > 0,
    privacyLevel: sanitizationResult.privacyLevel
  }

  let previousKeyword = ''
  const operations = new Set<string>()

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]
    const nextToken = i < tokens.length - 1 ? tokens[i + 1] : null
    const prevToken = i > 0 ? tokens[i - 1] : null

    if (token.type === TokenType.KEYWORD) {
      const keyword = token.value.toUpperCase()

      // Track main operations
      if (['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'ALTER', 'DROP', 'GRANT', 'REVOKE'].includes(keyword)) {
        operations.add(keyword)
      }

      previousKeyword = keyword
    }

    if (token.type === TokenType.IDENTIFIER) {
      // Extract table names
      if (['FROM', 'JOIN', 'INTO', 'TABLE', 'UPDATE'].includes(previousKeyword)) {
        if (nextToken?.value === '.') {
          // This is a schema name
          metadata.schemas.push(token.value)
        } else if (prevToken?.value === '.') {
          // This is a table name with schema prefix
          metadata.tables.push(token.value)
        } else {
          // This is just a table name
          metadata.tables.push(token.value)
        }
      }
    }
  }

  metadata.operations = Array.from(operations)

  // Remove duplicates
  metadata.tables = [...new Set(metadata.tables)]
  metadata.schemas = [...new Set(metadata.schemas)]

  return metadata
}
