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
  AnimatedFeatureBadge,
  FeatureBadge,
  GradientFeatureBadge,
} from '../feature-badge'

// Buttons
export {
  FloatingUpgradeButton,
  InlineUpgradePrompt,
  UpgradeBadgeButton,
  UpgradeButton,
  UpgradeLink,
} from '../upgrade-button'

// Overlays
export {
  BannerLockedOverlay,
  InlineLockedState,
  LockedFeatureOverlay,
  MinimalLockedOverlay,
} from '../locked-feature-overlay'

// Previews
export {
  CompactFeaturePreview,
  FeaturePreview,
  GridFeaturePreview,
} from '../feature-preview'

// Warnings
export {
  SoftLimitWarning,
  UpgradePromptCard,
  UsageStatsCard,
} from '../soft-limit-warning'

// Comparison
export {
  CompactComparisonCard,
  MobileComparison,
  ValueComparisonTable,
} from '../value-comparison-table'

// Trial
export {
  TrialBanner,
  TrialCountdown,
  TrialExpiredBanner,
} from '../trial-banner'

// Success
export {
  FeatureUnlockAnimation,
  UpgradeSuccessAnimation,
  UpgradeSuccessToast,
} from '../upgrade-success-animation'

// Hooks
export {
  type FeatureGateMode,
  type FeatureGateResult,
  useAvailableFeatures,
  useFeatureGate,
  useFeatureRequirement,
  useIsAtLeastTier,
  useMultiFeatureGate,
} from '@/hooks/use-feature-gate'

// Re-export types for convenience
export type { TierFeatures,TierLevel } from '@/types/tiers'
