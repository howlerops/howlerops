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
import type { ReactNode } from 'react'
import { useTierStore } from '@/store/tier-store'
import type { TierFeatures, TierLevel } from '@/types/tiers'
import { getRequiredTier } from '@/config/tier-limits'

/**
 * Feature gate mode
 * - 'hard': Completely blocks access (default)
 * - 'soft': Shows upgrade prompts but allows preview
 */
export type FeatureGateMode = 'hard' | 'soft'

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
  /** Feature gate mode (hard or soft) */
  mode: FeatureGateMode
  /** Function to show upgrade dialog */
  showUpgrade: (trigger?: string) => void
  /** Whether this is a team-only feature */
  isTeamFeature: boolean
  /** Whether this is an individual+ feature */
  isIndividualFeature: boolean
  /** Render feature preview with overlay (soft mode helper) */
  renderPreview: (children: ReactNode, options?: PreviewOptions) => ReactNode
  /** Render locked feature with overlay (soft mode helper) */
  renderLocked: (children: ReactNode, title: string, benefits: string[]) => ReactNode
  /** Render inline badge for locked feature */
  renderBadge: () => ReactNode
}

/**
 * Preview options for soft-gated features
 */
interface PreviewOptions {
  title?: string
  description?: string
  benefits?: string[]
  screenshot?: string
  variant?: 'card' | 'overlay' | 'inline'
}

/**
 * Check if a feature is available in the current tier
 *
 * @param feature - The feature name to check (e.g., 'sync', 'teamSharing')
 * @param mode - Gate mode: 'hard' (blocks completely) or 'soft' (shows preview with upgrade)
 * @returns Feature gate result with allowed status and upgrade helpers
 *
 * @example
 * ```typescript
 * // Hard gate (traditional blocking)
 * const { allowed } = useFeatureGate('sync', 'hard')
 * if (!allowed) return null
 *
 * // Soft gate with preview
 * const { allowed, renderLocked } = useFeatureGate('sync', 'soft')
 * if (!allowed) {
 *   return renderLocked(
 *     <SyncButton disabled />,
 *     "Cloud Sync",
 *     ["Sync across devices", "Never lose work", "Auto backups"]
 *   )
 * }
 *
 * // Soft gate with custom preview
 * const { allowed, renderPreview } = useFeatureGate('queryHistory', 'soft')
 * if (!allowed) {
 *   return renderPreview(
 *     <QueryHistoryUI />,
 *     {
 *       title: "Query History",
 *       description: "Access unlimited query history",
 *       benefits: ["Full-text search", "Advanced filters", "Save favorites"]
 *     }
 *   )
 * }
 * ```
 */
export function useFeatureGate(
  feature: keyof TierFeatures,
  mode: FeatureGateMode = 'hard'
): FeatureGateResult {
  const { currentTier, hasFeature, devMode } = useTierStore()

  const allowed = useMemo(() => hasFeature(feature), [feature, hasFeature])

  const requiredTier = useMemo(() => getRequiredTier(feature), [feature])

  const isTeamFeature = useMemo(() => requiredTier === 'team', [requiredTier])

  const isIndividualFeature = useMemo(() => requiredTier === 'individual', [requiredTier])

  const showUpgrade = useCallback(
    (trigger?: string) => {
      // Check cooldown to prevent spam
      const cooldownKey = `upgrade_prompt_${feature}_cooldown`
      const lastShown = localStorage.getItem(cooldownKey)
      if (lastShown) {
        const timeSinceShown = Date.now() - parseInt(lastShown, 10)
        const cooldownMs = 60 * 60 * 1000 // 1 hour
        if (timeSinceShown < cooldownMs) {
          console.log(`Upgrade prompt for ${feature} is in cooldown`)
          return
        }
      }

      // Set new cooldown
      localStorage.setItem(cooldownKey, Date.now().toString())

      // Dispatch custom event that upgrade modal listens to
      window.dispatchEvent(
        new CustomEvent('showUpgradeDialog', {
          detail: {
            trigger: trigger || feature,
            feature,
            currentTier,
            requiredTier,
            timestamp: new Date().toISOString(),
          },
        })
      )

      // Track analytics
      if (window.gtag) {
        window.gtag('event', 'upgrade_prompt_shown', {
          feature,
          current_tier: currentTier,
          required_tier: requiredTier,
          trigger: trigger || feature,
        })
      }
    },
    [feature, currentTier, requiredTier]
  )

  const renderPreview = useCallback(
    (children: ReactNode, options?: PreviewOptions): ReactNode => {
      // Use helper component to avoid circular dependencies
      const React = require('react')
      const { PreviewWrapper } = require('@/components/feature-gating-helpers')

      return React.createElement(PreviewWrapper, {
        feature,
        tier: requiredTier || 'individual',
        title: options?.title || feature,
        description: options?.description,
        screenshot: options?.screenshot,
        benefits: options?.benefits,
        variant: options?.variant || 'overlay',
        children,
      })
    },
    [feature, requiredTier]
  )

  const renderLocked = useCallback(
    (children: ReactNode, title: string, benefits: string[]): ReactNode => {
      // Use helper component to avoid circular dependencies
      const React = require('react')
      const { LockedWrapper } = require('@/components/feature-gating-helpers')

      return React.createElement(LockedWrapper, {
        feature,
        tier: requiredTier || 'individual',
        title,
        benefits,
        children,
      })
    },
    [feature, requiredTier]
  )

  const renderBadge = useCallback((): ReactNode => {
    // Use helper component to avoid circular dependencies
    const React = require('react')
    const { BadgeWrapper } = require('@/components/feature-gating-helpers')

    return React.createElement(BadgeWrapper, {
      tier: requiredTier || 'individual',
    })
  }, [requiredTier])

  return {
    allowed,
    tier: currentTier,
    requiredTier,
    isDevMode: devMode,
    mode,
    showUpgrade,
    isTeamFeature,
    isIndividualFeature,
    renderPreview,
    renderLocked,
    renderBadge,
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
