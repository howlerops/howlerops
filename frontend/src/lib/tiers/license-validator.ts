/**
 * License Key Validator
 *
 * Validates and parses Howlerops license keys.
 *
 * License Key Format: SQL-{TIER}-{UUID}-{CHECKSUM}
 *
 * Example: SQL-INDIVIDUAL-550e8400-e29b-41d4-a716-446655440000-A3F2
 *
 * Components:
 * - Prefix: Always "SQL"
 * - Tier: LOCAL, INDIVIDUAL, or TEAM
 * - UUID: Unique identifier for the license
 * - Checksum: CRC16 checksum of tier+uuid for validation
 *
 * The checksum prevents tampering and ensures license integrity.
 */

import type { TierLevel, LicenseValidationResult } from '@/types/tiers'

/**
 * License key format regex
 * Matches: SQL-{TIER}-{UUID}-{CHECKSUM}
 */
const LICENSE_KEY_REGEX = /^SQL-([A-Z]+)-([0-9a-f-]+)-([A-F0-9]{4})$/i

/**
 * Valid tier names in license keys
 */
const TIER_MAP: Record<string, TierLevel> = {
  LOCAL: 'local',
  INDIVIDUAL: 'individual',
  TEAM: 'team',
}

/**
 * Calculate CRC16 checksum for license validation
 * Uses CRC-16-CCITT polynomial (0x1021)
 *
 * @param data - String to calculate checksum for
 * @returns 4-character hex checksum
 */
function calculateChecksum(data: string): string {
  let crc = 0xffff
  const bytes = new TextEncoder().encode(data)

  for (const byte of bytes) {
    crc ^= byte << 8
    for (let i = 0; i < 8; i++) {
      if (crc & 0x8000) {
        crc = (crc << 1) ^ 0x1021
      } else {
        crc = crc << 1
      }
    }
  }

  crc = crc & 0xffff
  return crc.toString(16).toUpperCase().padStart(4, '0')
}

/**
 * Validate UUID format (RFC 4122)
 *
 * @param uuid - UUID string to validate
 * @returns True if valid UUID format
 */
function isValidUUID(uuid: string): boolean {
  const uuidRegex =
    /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  return uuidRegex.test(uuid)
}

/**
 * Parse a license key into its components
 *
 * @param key - License key string
 * @returns Parsed license metadata or null if invalid format
 */
function parseLicenseKey(key: string): {
  tier: string
  uuid: string
  checksum: string
} | null {
  const match = key.match(LICENSE_KEY_REGEX)

  if (!match) {
    return null
  }

  const [, tier, uuid, checksum] = match

  return {
    tier: tier.toUpperCase(),
    uuid: uuid.toLowerCase(),
    checksum: checksum.toUpperCase(),
  }
}

/**
 * Verify license checksum
 *
 * @param tier - Tier string from license
 * @param uuid - UUID from license
 * @param checksum - Checksum from license
 * @returns True if checksum is valid
 */
function verifyChecksum(tier: string, uuid: string, checksum: string): boolean {
  const data = `${tier}-${uuid}`
  const calculatedChecksum = calculateChecksum(data)
  return calculatedChecksum === checksum
}

/**
 * Generate a license key (for testing/development)
 * In production, license keys should be generated server-side
 *
 * @param tier - The tier level
 * @param uuid - Unique identifier (or generate random)
 * @returns Generated license key
 */
export function generateLicenseKey(tier: TierLevel, uuid?: string): string {
  const tierString = tier.toUpperCase()
  const licenseUuid = uuid || crypto.randomUUID()
  const data = `${tierString}-${licenseUuid}`
  const checksum = calculateChecksum(data)

  return `SQL-${tierString}-${licenseUuid}-${checksum}`
}

/**
 * Validate a license key
 *
 * Performs the following checks:
 * 1. Format validation (SQL-{TIER}-{UUID}-{CHECKSUM})
 * 2. Tier validation (must be LOCAL, INDIVIDUAL, or TEAM)
 * 3. UUID format validation
 * 4. Checksum verification
 * 5. Optional: Server-side validation (if API is available)
 *
 * @param key - License key to validate
 * @returns Validation result with details
 */
export async function validateLicenseKey(key: string): Promise<LicenseValidationResult> {
  // 1. Format validation
  const parsed = parseLicenseKey(key)
  if (!parsed) {
    return {
      valid: false,
      message: 'Invalid license key format. Expected: SQL-{TIER}-{UUID}-{CHECKSUM}',
    }
  }

  const { tier, uuid, checksum } = parsed

  // 2. Tier validation
  const tierLevel = TIER_MAP[tier]
  if (!tierLevel) {
    return {
      valid: false,
      message: `Invalid tier: ${tier}. Must be LOCAL, INDIVIDUAL, or TEAM`,
    }
  }

  // 3. UUID validation
  if (!isValidUUID(uuid)) {
    return {
      valid: false,
      message: 'Invalid UUID format in license key',
    }
  }

  // 4. Checksum verification
  if (!verifyChecksum(tier, uuid, checksum)) {
    return {
      valid: false,
      message: 'License key checksum verification failed. The key may be corrupted or tampered with',
    }
  }

  // 5. Optional: Server-side validation
  // In a production environment, you would validate against a license server
  // to check for revocation, expiration, and usage limits
  try {
    const serverValidation = await validateWithServer(key, tierLevel, uuid)
    if (!serverValidation.valid) {
      return serverValidation
    }

    // Return successful validation with metadata
    return {
      valid: true,
      tier: tierLevel,
      expiresAt: serverValidation.expiresAt,
      metadata: {
        tier: tierLevel,
        uuid,
        checksum,
        issuedAt: serverValidation.issuedAt,
        expiresAt: serverValidation.expiresAt,
      },
    }
  } catch (error) {
    // If server validation fails (network error, server down, etc.),
    // fall back to offline validation
    console.warn('Server validation failed, using offline validation:', error)

    return {
      valid: true,
      tier: tierLevel,
      message: 'License validated offline (server unavailable)',
      metadata: {
        tier: tierLevel,
        uuid,
        checksum,
      },
    }
  }
}

/**
 * Validate license with server
 * This would make a request to your license server API
 *
 * @param key - License key
 * @param tier - Tier level
 * @param uuid - License UUID
 * @returns Server validation result
 */
async function validateWithServer(
  _key: string,
  _tier: TierLevel,
  _uuid: string
): Promise<{
  valid: boolean
  message?: string
  expiresAt?: Date
  issuedAt?: Date
}> {
  // Check if we're in development mode
  if (import.meta.env.DEV) {
    // In development, skip server validation
    return {
      valid: true,
      issuedAt: new Date(),
      // Set expiration to 1 year from now for testing
      expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
    }
  }

  // In production, this would make an API call to validate the license
  // Example implementation:
  /*
  const response = await fetch('/api/licenses/validate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ key, tier, uuid }),
  })

  if (!response.ok) {
    return {
      valid: false,
      message: 'Server validation failed',
    }
  }

  const data = await response.json()

  return {
    valid: data.valid,
    message: data.message,
    expiresAt: data.expiresAt ? new Date(data.expiresAt) : undefined,
    issuedAt: data.issuedAt ? new Date(data.issuedAt) : undefined,
  }
  */

  // For now, return a placeholder implementation
  return {
    valid: true,
    issuedAt: new Date(),
  }
}

/**
 * Extract tier from license key without full validation
 * Useful for quick tier detection
 *
 * @param key - License key
 * @returns Tier level or null if invalid format
 */
export function extractTierFromLicense(key: string): TierLevel | null {
  const parsed = parseLicenseKey(key)
  if (!parsed) {
    return null
  }

  return TIER_MAP[parsed.tier] || null
}

/**
 * Check if a license key is well-formed (format only, no checksum verification)
 *
 * @param key - License key
 * @returns True if format is valid
 */
export function isWellFormedLicense(key: string): boolean {
  const parsed = parseLicenseKey(key)
  if (!parsed) {
    return false
  }

  const { tier, uuid } = parsed
  return !!TIER_MAP[tier] && isValidUUID(uuid)
}

/**
 * Development helper: Generate test licenses
 * Only available in development mode
 */
export const devLicenses = import.meta.env.DEV
  ? {
      local: generateLicenseKey('local'),
      individual: generateLicenseKey('individual'),
      team: generateLicenseKey('team'),
    }
  : undefined

// Log development licenses in console for easy testing
if (import.meta.env.DEV && devLicenses) {
  console.group('🔑 Development License Keys')
  console.log('Local:', devLicenses.local)
  console.log('Individual:', devLicenses.individual)
  console.log('Team:', devLicenses.team)
  console.groupEnd()
}
