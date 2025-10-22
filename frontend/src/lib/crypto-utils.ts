// Client-side crypto utilities for validation and key management

export interface KeyFingerprint {
  type: string
  size: number
  fingerprint: string
  comment?: string
}

export interface PemKeyInfo {
  type: 'rsa' | 'dsa' | 'ec' | 'ed25519' | 'openssh' | 'unknown'
  size: number
  fingerprint: string
  isEncrypted: boolean
  comment?: string
}

/**
 * Validates PEM key format
 */
export function validatePemKey(content: string): { valid: boolean; error?: string; info?: PemKeyInfo } {
  if (!content.trim()) {
    return { valid: false, error: 'Please provide a private key' }
  }

  // Check for common PEM key formats
  const pemPatterns = {
    rsa: /-----BEGIN RSA PRIVATE KEY-----[\s\S]*?-----END RSA PRIVATE KEY-----/,
    pkcs8: /-----BEGIN PRIVATE KEY-----[\s\S]*?-----END PRIVATE KEY-----/,
    openssh: /-----BEGIN OPENSSH PRIVATE KEY-----[\s\S]*?-----END OPENSSH PRIVATE KEY-----/,
    ec: /-----BEGIN EC PRIVATE KEY-----[\s\S]*?-----END EC PRIVATE KEY-----/,
    dsa: /-----BEGIN DSA PRIVATE KEY-----[\s\S]*?-----END DSA PRIVATE KEY-----/,
  }

  let keyType: PemKeyInfo['type'] = 'unknown'
  let matchedPattern: RegExp | null = null

  for (const [type, pattern] of Object.entries(pemPatterns)) {
    if (pattern.test(content)) {
      keyType = type as PemKeyInfo['type']
      matchedPattern = pattern
      break
    }
  }

  if (!matchedPattern) {
    return { 
      valid: false, 
      error: 'Invalid PEM key format. Please ensure the key starts with -----BEGIN and ends with -----END' 
    }
  }

  // Extract key info
  const info = extractKeyInfo(content, keyType)
  
  return { valid: true, info }
}

/**
 * Extracts information from a PEM key
 */
function extractKeyInfo(content: string, type: PemKeyInfo['type']): PemKeyInfo {
  // Check if key is encrypted
  const isEncrypted = content.includes('Proc-Type: 4,ENCRYPTED') || 
                     content.includes('DEK-Info:') ||
                     content.includes('-----BEGIN ENCRYPTED')

  // Extract key data for fingerprint calculation
  const lines = content.split('\n').filter(line => 
    !line.startsWith('-----BEGIN') && 
    !line.startsWith('-----END') && 
    !line.startsWith('Proc-Type:') &&
    !line.startsWith('DEK-Info:') &&
    line.trim()
  )
  
  const keyData = lines.join('')
  const fingerprint = calculateFingerprint(keyData)
  
  // Estimate key size (rough approximation)
  const size = estimateKeySize(keyData, type)

  return {
    type,
    size,
    fingerprint,
    isEncrypted,
  }
}

/**
 * Calculates a simple fingerprint for the key
 */
function calculateFingerprint(keyData: string): string {
  // Simple hash-based fingerprint (in production, use proper SSH key fingerprint)
  let hash = 0
  for (let i = 0; i < keyData.length; i++) {
    const char = keyData.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash // Convert to 32-bit integer
  }
  
  // Convert to hex and take first 16 characters
  const hex = Math.abs(hash).toString(16).padStart(8, '0')
  return hex.substring(0, 16)
}

/**
 * Estimates key size based on content
 */
function estimateKeySize(keyData: string, type: PemKeyInfo['type']): number {
  // Rough estimation based on base64 content length
  const base64Length = keyData.length
  const estimatedBytes = (base64Length * 3) / 4
  
  // Adjust based on key type
  switch (type) {
    case 'rsa':
      return Math.round(estimatedBytes / 2) // RSA keys are roughly half the total length
    case 'dsa':
      return 1024 // DSA keys are typically 1024 bits
    case 'ec':
      return 256 // EC keys are typically 256 bits
    case 'ed25519':
      return 256 // Ed25519 keys are 256 bits
    case 'openssh':
      return Math.round(estimatedBytes / 2) // Rough estimate
    default:
      return Math.round(estimatedBytes / 2)
  }
}

/**
 * Validates passphrase strength
 */
export function validatePassphrase(passphrase: string): { 
  valid: boolean; 
  score: number; 
  feedback: string[] 
} {
  const feedback: string[] = []
  let score = 0

  if (passphrase.length < 8) {
    feedback.push('Passphrase should be at least 8 characters long')
  } else {
    score += 1
  }

  if (passphrase.length >= 12) {
    score += 1
  }

  if (/[a-z]/.test(passphrase)) {
    score += 1
  } else {
    feedback.push('Add lowercase letters')
  }

  if (/[A-Z]/.test(passphrase)) {
    score += 1
  } else {
    feedback.push('Add uppercase letters')
  }

  if (/[0-9]/.test(passphrase)) {
    score += 1
  } else {
    feedback.push('Add numbers')
  }

  if (/[^a-zA-Z0-9]/.test(passphrase)) {
    score += 1
  } else {
    feedback.push('Add special characters')
  }

  // Check for common patterns
  const commonPatterns = [
    /password/i,
    /123456/,
    /qwerty/i,
    /abc123/i,
  ]

  if (commonPatterns.some(pattern => pattern.test(passphrase))) {
    score -= 2
    feedback.push('Avoid common patterns')
  }

  const valid = score >= 4 && passphrase.length >= 8

  if (valid && feedback.length === 0) {
    feedback.push('Strong passphrase')
  }

  return { valid, score, feedback }
}

/**
 * Generates a random passphrase suggestion
 */
export function generatePassphraseSuggestion(): string {
  const words = [
    'apple', 'banana', 'cherry', 'dragon', 'eagle', 'forest', 'garden', 'house',
    'island', 'jungle', 'knight', 'lizard', 'mountain', 'ocean', 'palace', 'queen',
    'river', 'sunset', 'tiger', 'umbrella', 'violet', 'wizard', 'yellow', 'zebra'
  ]
  
  const numbers = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0']
  const symbols = ['!', '@', '#', '$', '%', '^', '&', '*']
  
  // Pick 3 random words
  const selectedWords = []
  for (let i = 0; i < 3; i++) {
    const word = words[Math.floor(Math.random() * words.length)]
    selectedWords.push(word)
  }
  
  // Add a number and symbol
  const number = numbers[Math.floor(Math.random() * numbers.length)]
  const symbol = symbols[Math.floor(Math.random() * symbols.length)]
  
  return selectedWords.join('') + number + symbol
}

/**
 * Masks sensitive data for display
 */
export function maskSecret(secret: string, visibleChars: number = 4): string {
  if (secret.length <= visibleChars) {
    return '•'.repeat(secret.length)
  }
  
  const start = secret.substring(0, Math.floor(visibleChars / 2))
  const end = secret.substring(secret.length - Math.ceil(visibleChars / 2))
  const middle = '•'.repeat(secret.length - visibleChars)
  
  return start + middle + end
}

/**
 * Checks if a string looks like a private key
 */
export function isPrivateKey(content: string): boolean {
  const privateKeyPatterns = [
    /-----BEGIN.*PRIVATE KEY-----/,
    /-----BEGIN.*RSA PRIVATE KEY-----/,
    /-----BEGIN.*DSA PRIVATE KEY-----/,
    /-----BEGIN.*EC PRIVATE KEY-----/,
  ]
  
  return privateKeyPatterns.some(pattern => pattern.test(content))
}

/**
 * Checks if a string looks like a public key
 */
export function isPublicKey(content: string): boolean {
  const publicKeyPatterns = [
    /-----BEGIN.*PUBLIC KEY-----/,
    /ssh-rsa/,
    /ssh-dss/,
    /ssh-ed25519/,
    /ecdsa-sha2-/,
  ]
  
  return publicKeyPatterns.some(pattern => pattern.test(content))
}
