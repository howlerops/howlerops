/**
 * Tier Management System
 *
 * Central export for Howlerops's tier management functionality.
 * This module provides a complete solution for managing the 3-tier product structure.
 *
 * @module lib/tiers
 */

// License validation and generation
export {
  devLicenses,
  extractTierFromLicense,
  generateLicenseKey,
  isWellFormedLicense,
  validateLicenseKey,
} from './license-validator'

// Re-export types
export type {
  LicenseMetadata,
  LicenseValidationResult,
  LimitCheckResult,
  TeamRole,
  TierFeatures,
  TierLevel,
  TierLimits,
  TierPersistence,
} from '@/types/tiers'

// Re-export configuration
export {
  FEATURE_TIER_MAP,
  getRequiredTier,
  getTierLevel,
  isTierAtLeast,
  TIER_FEATURES,
  TIER_LIMITS,
  TIER_METADATA,
  tierHasFeature,
} from '@/config/tier-limits'

// Re-export store
export {
  initializeTierStore,
  tierSelectors,
  useTierStore,
} from '@/store/tier-store'

// Re-export hooks
export {
  useAvailableFeatures,
  useFeatureGate,
  useFeatureRequirement,
  useIsAtLeastTier,
  useMultiFeatureGate,
} from '@/hooks/use-feature-gate'
export {
  useCanExceedLimit,
  useCurrentLimits,
  useLimitProgress,
  useMultiLimitCheck,
  useTierLimit,
} from '@/hooks/use-tier-limit'

// Re-export components
export type {
  TierBadgeListProps,
  TierBadgeProps,
  TierBadgeVariant,
} from '@/components/tier-badge'
export { TierBadge, TierBadgeList } from '@/components/tier-badge'
