/**
 * Tier system type definitions for Howlerops
 * Defines the three-tier product structure: Local, Individual, and Team
 */

/**
 * Available product tiers
 * - local: Free, local-only, no sync
 * - individual: $9/mo, personal sync via Turso, unlimited storage
 * - team: $29/mo per team, shared resources, RBAC, audit log
 */
export type TierLevel = 'local' | 'individual' | 'team'

/**
 * Team role hierarchy for Team tier
 */
export type TeamRole = 'owner' | 'admin' | 'member' | 'viewer'

/**
 * Feature flags available across tiers
 */
export interface TierFeatures {
  // Sync & Multi-Device
  sync: boolean
  multiDevice: boolean
  aiMemorySync: boolean

  // Collaboration
  teamSharing: boolean

  // Support & Customization
  prioritySupport: boolean
  customThemes: boolean

  // Team-only features
  auditLog?: boolean
  rbac?: boolean
  sso?: boolean
}

/**
 * Resource limits per tier
 * null = unlimited
 */
export interface TierLimits {
  connections: number | null
  queryHistory: number | null
  aiMemories: number | null
  savedQueries: number | null
  exportFileSize: number // in bytes
  teamMembers?: number // Team tier only
}

/**
 * License key metadata
 */
export interface LicenseMetadata {
  tier: TierLevel
  uuid: string
  checksum: string
  issuedAt?: Date
  expiresAt?: Date
}

/**
 * Persistence format for tier data in localStorage
 */
export interface TierPersistence {
  currentTier: TierLevel
  licenseKey?: string
  expiresAt?: string // ISO date string
  lastValidated?: string // ISO date string
  teamId?: string
  teamName?: string
  teamRole?: TeamRole
}

/**
 * License validation result
 */
export interface LicenseValidationResult {
  valid: boolean
  tier?: TierLevel
  message?: string
  expiresAt?: Date
  metadata?: Partial<LicenseMetadata>
}

/**
 * Limit check result
 */
export interface LimitCheckResult {
  allowed: boolean
  remaining: number | null // null = unlimited
  limit: number | null // null = unlimited
  percentage: number // 0-100
  isNearLimit: boolean // < 20% remaining
  isAtLimit: boolean // >= 100%
  isUnlimited: boolean // true if limit is null
}
