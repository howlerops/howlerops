/**
 * Tier configuration and limits for SQL Studio
 * Defines features and resource limits for each tier
 */

import type { TierFeatures, TierLimits, TierLevel } from '@/types/tiers'

/**
 * Resource limits for each tier
 *
 * Local tier (Free):
 * - Limited connections, history, and exports
 * - Suitable for local development and testing
 *
 * Individual tier ($9/mo):
 * - Unlimited resources except export file size
 * - Cloud sync enabled
 *
 * Team tier ($29/mo):
 * - All Individual features plus team collaboration
 * - Larger export limits
 * - Team member management
 */
export const TIER_LIMITS: Record<TierLevel, TierLimits> = {
  local: {
    connections: null,
    queryHistory: null,
    aiMemories: null,
    savedQueries: null, // unlimited saved queries for local tier
    exportFileSize: 10 * 1024 * 1024, // 10MB
  },
  individual: {
    connections: null, // unlimited
    queryHistory: null, // unlimited
    aiMemories: null, // unlimited
    savedQueries: null, // unlimited
    exportFileSize: 100 * 1024 * 1024, // 100MB
  },
  team: {
    connections: null, // unlimited
    queryHistory: null, // unlimited
    aiMemories: null, // unlimited
    savedQueries: null, // unlimited
    exportFileSize: 500 * 1024 * 1024, // 500MB
    teamMembers: 5, // included seats, can add more
  },
} as const

/**
 * Feature availability for each tier
 *
 * Local tier:
 * - No cloud features
 * - Basic local functionality only
 *
 * Individual tier:
 * - Full cloud sync
 * - Multi-device support
 * - Priority support
 *
 * Team tier:
 * - All Individual features
 * - Team sharing and collaboration
 * - RBAC and audit logging
 * - SSO (future)
 */
export const TIER_FEATURES: Record<TierLevel, TierFeatures> = {
  local: {
    sync: false,
    multiDevice: false,
    aiMemorySync: false,
    teamSharing: false,
    prioritySupport: false,
    customThemes: false,
  },
  individual: {
    sync: true,
    multiDevice: true,
    aiMemorySync: true,
    teamSharing: false,
    prioritySupport: true,
    customThemes: true,
  },
  team: {
    sync: true,
    multiDevice: true,
    aiMemorySync: true,
    teamSharing: true,
    prioritySupport: true,
    customThemes: true,
    auditLog: true,
    rbac: true,
    sso: false, // future feature
  },
} as const

/**
 * Tier display metadata
 */
export const TIER_METADATA = {
  local: {
    name: 'Local',
    description: 'Free, local-only database client',
    price: 0,
    priceLabel: 'Free',
    color: 'gray',
  },
  individual: {
    name: 'Individual',
    description: 'Personal sync with unlimited storage',
    price: 9,
    priceLabel: '$9/month',
    color: 'blue',
  },
  team: {
    name: 'Team',
    description: 'Shared resources with RBAC and audit log',
    price: 29,
    priceLabel: '$29/month per team',
    color: 'purple',
  },
} as const

/**
 * Feature to tier mapping
 * Maps feature keys to the minimum required tier
 */
export const FEATURE_TIER_MAP: Record<string, TierLevel> = {
  sync: 'individual',
  multiDevice: 'individual',
  aiMemorySync: 'individual',
  prioritySupport: 'individual',
  customThemes: 'individual',
  teamSharing: 'team',
  auditLog: 'team',
  rbac: 'team',
  sso: 'team',
} as const

/**
 * Get the minimum tier required for a feature
 */
export function getRequiredTier(feature: string): TierLevel | null {
  return FEATURE_TIER_MAP[feature] || null
}

/**
 * Check if a tier has access to a feature
 */
export function tierHasFeature(tier: TierLevel, feature: keyof TierFeatures): boolean {
  return TIER_FEATURES[tier][feature] === true
}

/**
 * Get tier hierarchy level (for comparison)
 * local = 0, individual = 1, team = 2
 */
export function getTierLevel(tier: TierLevel): number {
  const levels: Record<TierLevel, number> = {
    local: 0,
    individual: 1,
    team: 2,
  }
  return levels[tier]
}

/**
 * Check if tierA is at least tierB
 */
export function isTierAtLeast(tierA: TierLevel, tierB: TierLevel): boolean {
  return getTierLevel(tierA) >= getTierLevel(tierB)
}
