/**
 * Credential Detection Module
 *
 * Detects various types of credentials and sensitive data in strings.
 * CRITICAL: Zero false negatives - we must catch ALL credentials.
 *
 * @module sanitization/credential-detector
 */

import { SanitizationConfig, getGlobalConfig } from './config'

export enum CredentialType {
  PASSWORD = 'password',
  API_KEY = 'api_key',
  JWT_TOKEN = 'jwt_token',
  SSH_PRIVATE_KEY = 'ssh_private_key',
  CONNECTION_STRING = 'connection_string',
  CERTIFICATE = 'certificate',
  HASH = 'hash',
  UNKNOWN = 'unknown'
}

export interface CredentialDetectionResult {
  isCredential: boolean
  type: CredentialType
  confidence: number // 0-1, where 1 is certain
  reason: string
  positions?: Array<{ start: number; end: number }>
}

/**
 * Enhanced entropy calculation for detecting high-entropy strings (likely credentials)
 */
function calculateShannonEntropy(str: string): number {
  if (!str || str.length === 0) return 0

  const frequencies = new Map<string, number>()
  for (const char of str) {
    frequencies.set(char, (frequencies.get(char) || 0) + 1)
  }

  let entropy = 0
  const len = str.length
  for (const freq of frequencies.values()) {
    const probability = freq / len
    entropy -= probability * Math.log2(probability)
  }

  return entropy
}

/**
 * Check if a string has characteristics of a randomly generated credential
 */
function hasRandomCharacteristics(str: string): boolean {
  if (str.length < 8) return false

  // Check for high entropy (randomness)
  const entropy = calculateShannonEntropy(str)
  if (entropy > 3.5) return true  // High entropy suggests randomness

  // Check for mixed character types
  const hasLower = /[a-z]/.test(str)
  const hasUpper = /[A-Z]/.test(str)
  const hasDigit = /\d/.test(str)
  const hasSpecial = /[^a-zA-Z0-9]/.test(str)
  const mixedTypes = [hasLower, hasUpper, hasDigit, hasSpecial].filter(Boolean).length

  // If it has 3+ character types and decent length, likely a credential
  if (mixedTypes >= 3 && str.length >= 12) return true

  // Check for patterns common in generated tokens
  const tokenPatterns = [
    /^[A-Za-z0-9+/]{20,}={0,2}$/,  // Base64
    /^[A-Fa-f0-9]{32,}$/,  // Hex (MD5, SHA, etc)
    /^[A-Za-z0-9_-]{20,}$/,  // URL-safe base64
  ]

  return tokenPatterns.some(pattern => pattern.test(str))
}

/**
 * Main credential detection function
 * CRITICAL: This must have ZERO false negatives for security
 */
export function detectCredentials(
  input: string,
  config: SanitizationConfig = getGlobalConfig()
): CredentialDetectionResult {
  // Null/empty check
  if (!input || typeof input !== 'string') {
    return {
      isCredential: false,
      type: CredentialType.UNKNOWN,
      confidence: 0,
      reason: 'Empty or invalid input'
    }
  }

  // Trim for analysis (but preserve original for position tracking)
  const trimmed = input.trim()

  // Check for SSH private keys (highest priority)
  for (const pattern of config.credentialPatterns.privateKey) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.SSH_PRIVATE_KEY,
        confidence: 1.0,
        reason: 'SSH private key detected'
      }
      return result
    }
  }

  // Check for certificates
  const certPatterns = [
    /-----BEGIN CERTIFICATE-----/,
    /-----BEGIN TRUSTED CERTIFICATE-----/,
    /-----BEGIN X509 CERTIFICATE-----/,
    /-----BEGIN PUBLIC KEY-----/
  ]
  for (const pattern of certPatterns) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.CERTIFICATE,
        confidence: 1.0,
        reason: 'Certificate detected'
      }
      return result
    }
  }

  // Check for JWT tokens
  for (const pattern of config.credentialPatterns.jwt) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.JWT_TOKEN,
        confidence: 1.0,
        reason: 'JWT token detected'
      }
      return result
    }
  }

  // Check for connection strings
  for (const pattern of config.credentialPatterns.connectionString) {
    if (pattern.test(trimmed)) {
      return {
        isCredential: true,
        type: CredentialType.CONNECTION_STRING,
        confidence: 1.0,
        reason: 'Database connection string detected'
      }
    }
  }

  // Additional connection string patterns
  const connStringIndicators = [
    /mongodb:\/\//i,
    /mongodb\+srv:\/\//i,
    /postgres(?:ql)?:\/\//i,
    /mysql:\/\//i,
    /redis:\/\//i,
    /amqp:\/\//i,
    /kafka:\/\//i,
    /elasticsearch:\/\//i,
    /https?:\/\/[^\s]+:[^\s]+@/i,
  ]
  for (const indicator of connStringIndicators) {
    if (indicator.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.CONNECTION_STRING,
        confidence: 0.9,
        reason: 'Connection string indicator detected',
      }
      return result
    }
  }

  // Check for common hash formats before other heuristics to avoid misclassification
  const hashPatterns = [
    { pattern: /^[a-f0-9]{32}$/i, name: 'MD5' },
    { pattern: /^[a-f0-9]{40}$/i, name: 'SHA1' },
    { pattern: /^[a-f0-9]{56}$/i, name: 'SHA224' },
    { pattern: /^[a-f0-9]{64}$/i, name: 'SHA256' },
    { pattern: /^[a-f0-9]{96}$/i, name: 'SHA384' },
    { pattern: /^[a-f0-9]{128}$/i, name: 'SHA512' },
    { pattern: /^\$2[aby]\$\d{2}\$[./A-Za-z0-9]{53}$/, name: 'bcrypt' },
    { pattern: /^\$argon2(i|d|id)\$/, name: 'Argon2' },
    { pattern: /^pbkdf2/i, name: 'PBKDF2' },
  ]
  for (const { pattern, name } of hashPatterns) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.HASH,
        confidence: 0.95,
        reason: `${name} hash detected`,
      }
      return result
    }
  }

  // Quick length check - very long strings are suspicious (after explicit pattern checks)
  if (trimmed.length > config.maxSafeStringLength) {
    const result: CredentialDetectionResult = {
      isCredential: true,
      type: CredentialType.UNKNOWN,
      confidence: 0.7,
      reason: `String exceeds safe length (${trimmed.length} > ${config.maxSafeStringLength})`,
    }
    return result
  }

  // Check for API keys
  for (const pattern of config.credentialPatterns.apiKey) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.API_KEY,
        confidence: 0.9,
        reason: 'API key pattern detected'
      }
      return result
    }
  }

  // Known API key prefixes
  const apiKeyPrefixes = [
    'sk-',  // OpenAI secret key
    'pk-',  // Public key (various)
    'rk-',  // Restricted key
    'live_',  // Stripe live key
    'test_',  // Stripe test key
    'api-',
    'key-',
    'token-',
    'bearer ',
    'basic ',
    'AIza',  // Google API key prefix
    'AKIA',  // AWS Access Key ID prefix
  ]
  for (const prefix of apiKeyPrefixes) {
    if (trimmed.toLowerCase().startsWith(prefix.toLowerCase())) {
      if (trimmed.length < 16) {
        continue
      }
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.API_KEY,
        confidence: 0.85,
        reason: `API key prefix detected: ${prefix}`
      }
      return result
    }
  }

  // Check for password patterns
  for (const pattern of config.credentialPatterns.password) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.PASSWORD,
        confidence: 0.8,
        reason: 'Password pattern detected'
      }
      return result
    }
  }

  // Check for high entropy / random characteristics
  if (trimmed.length >= 16 && hasRandomCharacteristics(trimmed)) {
    // Additional checks for common non-credentials that might have high entropy
    const likelyNotCredential = [
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,  // UUID
      /^\d{4}-\d{2}-\d{2}/,  // Dates
      /^[A-Z0-9]{2,}-\d+$/,  // Order numbers, reference codes
      /^v?\d+\.\d+\.\d+/,  // Version numbers
      /^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$/i,  // Email addresses
    ]

    if (!likelyNotCredential.some(p => p.test(trimmed))) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.UNKNOWN,
        confidence: 0.6,
        reason: 'High entropy string detected (possible credential)'
      }
      return result
      }
    }

  // Check for suspicious key-value patterns in the string
  const keyValuePatterns = [
    /password\s*[:=]\s*["']?([^"'\s]+)/i,
    /api[_-]?key\s*[:=]\s*["']?([^"'\s]+)/i,
    /token\s*[:=]\s*["']?([^"'\s]+)/i,
    /secret\s*[:=]\s*["']?([^"'\s]+)/i,
    /auth\s*[:=]\s*["']?([^"'\s]+)/i,
    /bearer\s+([A-Za-z0-9_-]+)/i,
    /basic\s+([A-Za-z0-9+/=]+)/i
  ]

  for (const pattern of keyValuePatterns) {
    const match = pattern.exec(trimmed)
    if (match) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.UNKNOWN,
        confidence: 0.75,
        reason: `Credential key-value pattern detected`,
        positions: [{ start: match.index, end: match.index + match[0].length }]
      }
      return result
    }
  }

  // Environment variable references that likely contain credentials
  const envVarPatterns = [
    /\$\{?[A-Z_]*PASSWORD/i,
    /\$\{?[A-Z_]*SECRET/i,
    /\$\{?[A-Z_]*TOKEN/i,
    /\$\{?[A-Z_]*KEY/i,
    /\$\{?[A-Z_]*APIKEY/i,
    /\$\{?[A-Z_]*API_KEY/i
  ]

  for (const pattern of envVarPatterns) {
    if (pattern.test(trimmed)) {
      const result: CredentialDetectionResult = {
        isCredential: true,
        type: CredentialType.UNKNOWN,
        confidence: 0.5,
        reason: 'Environment variable reference to credential'
      }
      return result
    }
  }

  // No credentials detected
  return {
    isCredential: false,
    type: CredentialType.UNKNOWN,
    confidence: 0,
    reason: 'No credential patterns detected'
  }
}

/**
 * Batch detection for multiple strings
 */
export function detectCredentialsInBatch(
  inputs: string[],
  config?: SanitizationConfig
): CredentialDetectionResult[] {
  return inputs.map(input => detectCredentials(input, config))
}

/**
 * Deep scan an object for credentials
 */
export function deepScanForCredentials(
  obj: unknown,
  config?: SanitizationConfig,
  visitedRefs = new WeakSet()
): Array<{ path: string; result: CredentialDetectionResult }> {
  const results: Array<{ path: string; result: CredentialDetectionResult }> = []

  function scan(value: unknown, path: string): void {
    // Handle null/undefined
    if (value === null || value === undefined) return

    // Handle circular references
    if (typeof value === 'object') {
      if (visitedRefs.has(value as object)) return
      visitedRefs.add(value as object)
    }

    // Handle strings
    if (typeof value === 'string') {
      const result = detectCredentials(value, config)
      if (result.isCredential) {
        results.push({ path, result })
      }
      return
    }

    // Handle arrays
    if (Array.isArray(value)) {
      value.forEach((item, index) => {
        scan(item, `${path}[${index}]`)
      })
      return
    }

    // Handle objects
    if (typeof value === 'object') {
      Object.entries(value as Record<string, unknown>).forEach(([key, val]) => {
        // Check the key name itself for sensitive indicators
        const keysToFlag = [
          'password', 'pass', 'pwd', 'secret', 'token', 'apikey',
          'api_key', 'privatekey', 'private_key', 'auth', 'authorization',
          'credential', 'ssh_key', 'access_token', 'refresh_token'
        ]

        const lowerKey = key.toLowerCase()
        if (keysToFlag.some(flag => lowerKey.includes(flag))) {
          if (typeof val === 'string' && val) {
            const result = detectCredentials(val, config)
            result.confidence = Math.min(1, result.confidence + 0.2)
            results.push({ path: `${path}.${key}`, result })
          } else {
            scan(val, `${path}.${key}`)
          }
        } else {
          scan(val, `${path}.${key}`)
        }
      })
    }
  }

  scan(obj, 'root')
  return results
}

/**
 * Quick check if a string might be a credential (optimized for performance)
 */
export function mightBeCredential(str: string): boolean {
  if (!str || str.length < 8) return false

  // Quick checks for obvious patterns
  const quickPatterns = [
    /^[A-Za-z0-9+/]{20,}={0,2}$/,  // Base64
    /^[A-Fa-f0-9]{32,}$/,  // Hex
    /^eyJ/,  // JWT start
    /-----BEGIN/,  // Keys/certs
    /^(sk|pk|api|key|token|bearer|basic)[-_]/i,  // Common prefixes
  ]

  return quickPatterns.some(p => p.test(str)) || hasRandomCharacteristics(str)
}

/**
 * Redact credentials in a string
 */
const SENSITIVE_KEY_PATTERN = /(password|pass|pwd|secret|token|key|credential|auth)/i

export function redactCredentials(
  input: string,
  config?: SanitizationConfig
): { redacted: string; detections: CredentialDetectionResult[] } {
  const detections: CredentialDetectionResult[] = []

  const maskValue = (value: string): string => {
    if (value.length <= 2) {
      return '*'.repeat(value.length)
    }
    return value[0] + '*'.repeat(value.length - 2) + value[value.length - 1]
  }

  const segments = input.split(/(\s+|[,;:|])/g)
  const processed = segments.map((segment) => {
    if (!segment || /^\s*$/.test(segment)) {
      return segment
    }

    const eqIndex = segment.indexOf('=')
    if (eqIndex > 0 && eqIndex < segment.length - 1) {
      const key = segment.slice(0, eqIndex + 1)
      const value = segment.slice(eqIndex + 1)
      const detection = detectCredentials(value, config)
      if (detection.isCredential && detection.confidence > 0.5) {
        detections.push(detection)
        return key + maskValue(value)
      }

      if (SENSITIVE_KEY_PATTERN.test(key)) {
        detections.push({
          isCredential: true,
          type: CredentialType.UNKNOWN,
          confidence: 0.6,
          reason: `Sensitive key '${key.slice(0, -1)}' detected`,
        })
        return key + maskValue(value)
      }
      return segment
    }

    const detection = detectCredentials(segment, config)
    if (detection.isCredential && detection.confidence > 0.5) {
      detections.push(detection)
      return maskValue(segment)
    }

    return segment
  })

  return { redacted: processed.join(''), detections }
}
