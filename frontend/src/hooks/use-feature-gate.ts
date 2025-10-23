/**
 * Feature Gate Hook
 *
 * React hook for checking feature availability based on the current tier.
 * Provides easy integration for gating features behind tier requirements.
 *
 * Usage:
 * ```typescript
 * const CloudSync = () => {
 *   const { allowed, tier, requiredTier, showUpgrade } = useFeatureGate('sync')
 *
 *   if (!allowed) {
 *     return (
 *       <div>
 *         <p>Cloud sync requires {requiredTier} tier</p>
 *         <button onClick={showUpgrade}>Upgrade Now</button>
 *       </div>
 *     )
 *   }
 *
 *   return <SyncComponent />
 * }
 * ```
 */

import { useMemo, useCallback } from 'react'
import { useTierStore } from '@/store/tier-store'
import type { TierFeatures, TierLevel } from '@/types/tiers'
import { getRequiredTier } from '@/config/tier-limits'

/**
 * Feature gate result interface
 */
export interface FeatureGateResult {
  /** Whether the feature is allowed in the current tier */
  allowed: boolean
  /** Current active tier */
  tier: TierLevel
  /** Minimum tier required for this feature (null if feature doesn't exist) */
  requiredTier: TierLevel | null
  /** Whether the user is in development mode (all features unlocked) */
  isDevMode: boolean
  /** Function to show upgrade dialog */
  showUpgrade: () => void
  /** Whether this is a team-only feature */
  isTeamFeature: boolean
  /** Whether this is an individual+ feature */
  isIndividualFeature: boolean
}

/**
 * Check if a feature is available in the current tier
 *
 * @param feature - The feature name to check (e.g., 'sync', 'teamSharing')
 * @returns Feature gate result with allowed status and upgrade helpers
 *
 * @example
 * ```typescript
 * // Basic usage
 * const { allowed } = useFeatureGate('sync')
 *
 * // With upgrade prompt
 * const { allowed, showUpgrade, requiredTier } = useFeatureGate('teamSharing')
 * if (!allowed) {
 *   return (
 *     <div>
 *       <p>This requires {requiredTier} tier</p>
 *       <button onClick={showUpgrade}>Upgrade</button>
 *     </div>
 *   )
 * }
 *
 * // Check multiple features
 * const syncGate = useFeatureGate('sync')
 * const teamGate = useFeatureGate('teamSharing')
 * const canCollaborate = syncGate.allowed && teamGate.allowed
 * ```
 */
export function useFeatureGate(feature: keyof TierFeatures): FeatureGateResult {
  const { currentTier, hasFeature, devMode } = useTierStore()

  const allowed = useMemo(() => hasFeature(feature), [feature, hasFeature])

  const requiredTier = useMemo(() => getRequiredTier(feature), [feature])

  const isTeamFeature = useMemo(() => requiredTier === 'team', [requiredTier])

  const isIndividualFeature = useMemo(() => requiredTier === 'individual', [requiredTier])

  const showUpgrade = useCallback(() => {
    // In a real implementation, this would open an upgrade dialog/modal
    // For now, we'll log to console and could integrate with a toast/modal system
    console.log(`Upgrade required: ${feature} needs ${requiredTier} tier`)

    // You can dispatch a custom event that a modal component listens to
    window.dispatchEvent(
      new CustomEvent('showUpgradeDialog', {
        detail: {
          feature,
          currentTier,
          requiredTier,
        },
      })
    )
  }, [feature, currentTier, requiredTier])

  return {
    allowed,
    tier: currentTier,
    requiredTier,
    isDevMode: devMode,
    showUpgrade,
    isTeamFeature,
    isIndividualFeature,
  }
}

/**
 * Check multiple features at once
 *
 * @param features - Array of feature names to check
 * @returns Object with feature names as keys and allowed status as values
 *
 * @example
 * ```typescript
 * const features = useMultiFeatureGate(['sync', 'teamSharing', 'auditLog'])
 * // { sync: true, teamSharing: false, auditLog: false }
 *
 * if (features.sync && features.teamSharing) {
 *   // Show collaborative sync features
 * }
 * ```
 */
export function useMultiFeatureGate(
  features: Array<keyof TierFeatures>
): Record<keyof TierFeatures, boolean> {
  const { hasFeature } = useTierStore()

  return useMemo(() => {
    const result: Partial<Record<keyof TierFeatures, boolean>> = {}
    for (const feature of features) {
      result[feature] = hasFeature(feature)
    }
    return result as Record<keyof TierFeatures, boolean>
  }, [features, hasFeature])
}

/**
 * Get all available features for the current tier
 *
 * @returns Object with all features and their availability
 *
 * @example
 * ```typescript
 * const availableFeatures = useAvailableFeatures()
 * // {
 * //   sync: true,
 * //   multiDevice: true,
 * //   teamSharing: false,
 * //   ...
 * // }
 * ```
 */
export function useAvailableFeatures(): TierFeatures {
  const { getFeatures } = useTierStore()
  return useMemo(() => getFeatures(), [getFeatures])
}

/**
 * Check if current tier is at least the specified tier
 *
 * @param tier - The tier to compare against
 * @returns True if current tier >= specified tier
 *
 * @example
 * ```typescript
 * const isPaidUser = useIsAtLeastTier('individual')
 * const isTeamUser = useIsAtLeastTier('team')
 * ```
 */
export function useIsAtLeastTier(tier: TierLevel): boolean {
  const { isAtLeastTier } = useTierStore()
  return useMemo(() => isAtLeastTier(tier), [tier, isAtLeastTier])
}

/**
 * Get feature requirement information without checking current tier
 * Useful for displaying upgrade prompts or feature marketing
 *
 * @param feature - Feature name
 * @returns Required tier and feature metadata
 *
 * @example
 * ```typescript
 * const TeamFeatureCard = ({ feature }) => {
 *   const { requiredTier, isTeamOnly } = useFeatureRequirement(feature)
 *
 *   return (
 *     <div>
 *       <h3>{feature}</h3>
 *       <Badge>{requiredTier}</Badge>
 *       {isTeamOnly && <span>Team exclusive</span>}
 *     </div>
 *   )
 * }
 * ```
 */
export function useFeatureRequirement(feature: string): {
  requiredTier: TierLevel | null
  isTeamOnly: boolean
  isIndividualPlus: boolean
  isFree: boolean
} {
  const requiredTier = useMemo(() => getRequiredTier(feature), [feature])

  return useMemo(
    () => ({
      requiredTier,
      isTeamOnly: requiredTier === 'team',
      isIndividualPlus: requiredTier === 'individual' || requiredTier === 'team',
      isFree: requiredTier === null || requiredTier === 'local',
    }),
    [requiredTier]
  )
}
