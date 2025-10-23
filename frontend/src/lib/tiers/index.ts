/**
 * Tier Management System
 *
 * Central export for SQL Studio's tier management functionality.
 * This module provides a complete solution for managing the 3-tier product structure.
 *
 * @module lib/tiers
 */

// License validation and generation
export {
  validateLicenseKey,
  generateLicenseKey,
  extractTierFromLicense,
  isWellFormedLicense,
  devLicenses,
} from './license-validator'

// Re-export types
export type {
  TierLevel,
  TierFeatures,
  TierLimits,
  LicenseMetadata,
  LicenseValidationResult,
  LimitCheckResult,
  TierPersistence,
  TeamRole,
} from '@/types/tiers'

// Re-export configuration
export {
  TIER_LIMITS,
  TIER_FEATURES,
  TIER_METADATA,
  FEATURE_TIER_MAP,
  getRequiredTier,
  tierHasFeature,
  getTierLevel,
  isTierAtLeast,
} from '@/config/tier-limits'

// Re-export store
export {
  useTierStore,
  initializeTierStore,
  tierSelectors,
} from '@/store/tier-store'

// Re-export hooks
export {
  useFeatureGate,
  useMultiFeatureGate,
  useAvailableFeatures,
  useIsAtLeastTier,
  useFeatureRequirement,
} from '@/hooks/use-feature-gate'

export {
  useTierLimit,
  useCanExceedLimit,
  useCurrentLimits,
  useMultiLimitCheck,
  useLimitProgress,
} from '@/hooks/use-tier-limit'

// Re-export components
export { TierBadge, TierBadgeList } from '@/components/tier-badge'

export type {
  TierBadgeProps,
  TierBadgeVariant,
  TierBadgeListProps,
} from '@/components/tier-badge'
