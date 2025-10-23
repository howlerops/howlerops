/**
 * Feature Gating UI Components
 *
 * A comprehensive set of components for showing value instead of blocking users.
 * These components create desire by previewing locked features with upgrade prompts.
 *
 * @example
 * ```tsx
 * import {
 *   FeatureBadge,
 *   UpgradeButton,
 *   LockedFeatureOverlay,
 *   useFeatureGate
 * } from '@/components/feature-gating'
 *
 * function MyFeature() {
 *   const { allowed, renderLocked } = useFeatureGate('sync', 'soft')
 *
 *   if (!allowed) {
 *     return renderLocked(
 *       <SyncButton disabled />,
 *       "Cloud Sync",
 *       ["Sync across devices", "Auto-backup", "Never lose work"]
 *     )
 *   }
 *
 *   return <SyncButton />
 * }
 * ```
 */

// Badges
export {
  FeatureBadge,
  AnimatedFeatureBadge,
  GradientFeatureBadge,
} from '../feature-badge'

// Buttons
export {
  UpgradeButton,
  UpgradeLink,
  UpgradeBadgeButton,
  FloatingUpgradeButton,
  InlineUpgradePrompt,
} from '../upgrade-button'

// Overlays
export {
  LockedFeatureOverlay,
  MinimalLockedOverlay,
  BannerLockedOverlay,
  InlineLockedState,
} from '../locked-feature-overlay'

// Previews
export {
  FeaturePreview,
  GridFeaturePreview,
  CompactFeaturePreview,
} from '../feature-preview'

// Warnings
export {
  SoftLimitWarning,
  UsageStatsCard,
  UpgradePromptCard,
} from '../soft-limit-warning'

// Comparison
export {
  ValueComparisonTable,
  CompactComparisonCard,
  MobileComparison,
} from '../value-comparison-table'

// Trial
export {
  TrialBanner,
  TrialCountdown,
  TrialExpiredBanner,
} from '../trial-banner'

// Success
export {
  UpgradeSuccessAnimation,
  UpgradeSuccessToast,
  FeatureUnlockAnimation,
} from '../upgrade-success-animation'

// Hooks
export {
  useFeatureGate,
  useMultiFeatureGate,
  useAvailableFeatures,
  useIsAtLeastTier,
  useFeatureRequirement,
  type FeatureGateResult,
  type FeatureGateMode,
} from '@/hooks/use-feature-gate'

// Re-export types for convenience
export type { TierLevel, TierFeatures } from '@/types/tiers'
