/**
 * Tier Limit Hook
 *
 * React hook for checking and monitoring usage limits based on the current tier.
 * Provides real-time limit checking and usage tracking with helpful utilities.
 *
 * Usage:
 * ```typescript
 * const ConnectionManager = () => {
 *   const connections = useConnections()
 *   const limit = useTierLimit('connections', connections.length)
 *
 *   return (
 *     <div>
 *       <p>Connections: {limit.usage} / {limit.limit ?? '∞'}</p>
 *       {limit.isNearLimit && <Warning>Approaching limit</Warning>}
 *       {!limit.allowed && (
 *         <button onClick={limit.showUpgrade}>Upgrade for more</button>
 *       )}
 *     </div>
 *   )
 * }
 * ```
 */

import { useMemo, useCallback } from 'react'
import { useTierStore } from '@/store/tier-store'
import type { TierLimits, TierLevel } from '@/types/tiers'

/**
 * Extended limit check result with helper methods
 */
export interface TierLimitResult {
  /** Current usage value */
  usage: number
  /** Maximum allowed value (null = unlimited) */
  limit: number | null
  /** Remaining capacity (null = unlimited) */
  remaining: number | null
  /** Percentage used (0-100) */
  percentage: number
  /** Whether the limit allows current usage */
  allowed: boolean
  /** Whether usage is near the limit (>= 80%) */
  isNearLimit: boolean
  /** Whether usage has reached the limit */
  isAtLimit: boolean
  /** Whether the limit is unlimited */
  isUnlimited: boolean
  /** Current tier */
  tier: TierLevel
  /** Function to show upgrade dialog */
  showUpgrade: () => void
  /** Get a formatted display string (e.g., "3 / 5" or "3 / ∞") */
  displayString: string
  /** Get a color indicator based on usage (green, yellow, red) */
  colorIndicator: 'green' | 'yellow' | 'red'
}

/**
 * Monitor a usage limit for the current tier
 *
 * @param limitName - The limit to check (e.g., 'connections', 'queryHistory')
 * @param currentUsage - Current usage value
 * @returns Detailed limit information and helpers
 *
 * @example
 * ```typescript
 * // Basic usage
 * const { allowed, remaining } = useTierLimit('connections', connectionCount)
 *
 * // With visual indicators
 * const limit = useTierLimit('queryHistory', historyCount)
 * return (
 *   <div>
 *     <Progress value={limit.percentage} color={limit.colorIndicator} />
 *     <span>{limit.displayString}</span>
 *   </div>
 * )
 *
 * // With upgrade prompt
 * const limit = useTierLimit('savedQueries', savedCount)
 * if (!limit.allowed) {
 *   return <UpgradePrompt onUpgrade={limit.showUpgrade} />
 * }
 * ```
 */
export function useTierLimit(
  limitName: keyof TierLimits,
  currentUsage: number
): TierLimitResult {
  const { currentTier, checkLimit } = useTierStore()

  const limitCheck = useMemo(
    () => checkLimit(limitName, currentUsage),
    [limitName, currentUsage, checkLimit]
  )

  const isUnlimited = useMemo(() => limitCheck.limit === null, [limitCheck.limit])

  const displayString = useMemo(() => {
    if (isUnlimited) {
      return `${currentUsage} / ∞`
    }
    return `${currentUsage} / ${limitCheck.limit}`
  }, [currentUsage, limitCheck.limit, isUnlimited])

  const colorIndicator = useMemo((): 'green' | 'yellow' | 'red' => {
    if (isUnlimited) {
      return 'green'
    }
    if (limitCheck.isAtLimit) {
      return 'red'
    }
    if (limitCheck.isNearLimit) {
      return 'yellow'
    }
    return 'green'
  }, [isUnlimited, limitCheck.isAtLimit, limitCheck.isNearLimit])

  const showUpgrade = useCallback(() => {
    console.log(`Upgrade required: ${limitName} limit reached (${currentUsage} / ${limitCheck.limit})`)

    // Dispatch custom event for upgrade dialog
    window.dispatchEvent(
      new CustomEvent('showUpgradeDialog', {
        detail: {
          limit: limitName,
          currentTier,
          usage: currentUsage,
          limit: limitCheck.limit,
        },
      })
    )
  }, [limitName, currentTier, currentUsage, limitCheck.limit])

  return {
    usage: currentUsage,
    limit: limitCheck.limit,
    remaining: limitCheck.remaining,
    percentage: limitCheck.percentage,
    allowed: limitCheck.allowed,
    isNearLimit: limitCheck.isNearLimit,
    isAtLimit: limitCheck.isAtLimit,
    isUnlimited,
    tier: currentTier,
    showUpgrade,
    displayString,
    colorIndicator,
  }
}

/**
 * Check if an action would exceed the limit
 *
 * @param limitName - The limit to check
 * @param currentUsage - Current usage value
 * @param increment - Amount to add (default: 1)
 * @returns Whether the action is allowed and result information
 *
 * @example
 * ```typescript
 * const handleAddConnection = () => {
 *   const { allowed, showUpgrade } = useCanExceedLimit(
 *     'connections',
 *     connections.length,
 *     1
 *   )
 *
 *   if (!allowed) {
 *     showUpgrade()
 *     return
 *   }
 *
 *   addConnection(...)
 * }
 * ```
 */
export function useCanExceedLimit(
  limitName: keyof TierLimits,
  currentUsage: number,
  increment: number = 1
): {
  allowed: boolean
  wouldExceed: boolean
  afterUsage: number
  showUpgrade: () => void
} {
  const { checkLimit, currentTier } = useTierStore()

  const afterUsage = currentUsage + increment

  const result = useMemo(() => {
    const check = checkLimit(limitName, afterUsage)
    return {
      allowed: check.allowed,
      wouldExceed: !check.allowed,
      afterUsage,
    }
  }, [limitName, afterUsage, checkLimit])

  const showUpgrade = useCallback(() => {
    console.log(
      `Upgrade required: Adding ${increment} would exceed ${limitName} limit`
    )

    window.dispatchEvent(
      new CustomEvent('showUpgradeDialog', {
        detail: {
          limit: limitName,
          currentTier,
          usage: currentUsage,
          increment,
        },
      })
    )
  }, [limitName, currentTier, currentUsage, increment])

  return {
    ...result,
    showUpgrade,
  }
}

/**
 * Get all limits for the current tier
 *
 * @returns All limits configuration for current tier
 *
 * @example
 * ```typescript
 * const limits = useCurrentLimits()
 * // {
 * //   connections: 5,
 * //   queryHistory: 50,
 * //   ...
 * // }
 * ```
 */
export function useCurrentLimits(): TierLimits {
  const { getLimits } = useTierStore()
  return useMemo(() => getLimits(), [getLimits])
}

/**
 * Check multiple limits at once
 *
 * @param limits - Object mapping limit names to current usage
 * @returns Object with limit check results for each limit
 *
 * @example
 * ```typescript
 * const checks = useMultiLimitCheck({
 *   connections: connectionCount,
 *   queryHistory: historyCount,
 *   savedQueries: savedCount,
 * })
 *
 * const allWithinLimits = Object.values(checks).every(c => c.allowed)
 * ```
 */
export function useMultiLimitCheck(
  limits: Partial<Record<keyof TierLimits, number>>
): Record<string, TierLimitResult> {
  const { checkLimit, currentTier } = useTierStore()

  return useMemo(() => {
    const results: Record<string, TierLimitResult> = {}

    for (const [limitName, usage] of Object.entries(limits)) {
      if (usage !== undefined) {
        const limitCheck = checkLimit(limitName as keyof TierLimits, usage)
        const isUnlimited = limitCheck.limit === null

        results[limitName] = {
          usage,
          limit: limitCheck.limit,
          remaining: limitCheck.remaining,
          percentage: limitCheck.percentage,
          allowed: limitCheck.allowed,
          isNearLimit: limitCheck.isNearLimit,
          isAtLimit: limitCheck.isAtLimit,
          isUnlimited,
          tier: currentTier,
          showUpgrade: () => {
            window.dispatchEvent(
              new CustomEvent('showUpgradeDialog', {
                detail: { limit: limitName, currentTier, usage },
              })
            )
          },
          displayString: isUnlimited ? `${usage} / ∞` : `${usage} / ${limitCheck.limit}`,
          colorIndicator: isUnlimited
            ? 'green'
            : limitCheck.isAtLimit
              ? 'red'
              : limitCheck.isNearLimit
                ? 'yellow'
                : 'green',
        }
      }
    }

    return results
  }, [limits, checkLimit, currentTier])
}

/**
 * Get a percentage-based progress indicator for a limit
 *
 * @param limitName - The limit to check
 * @param currentUsage - Current usage value
 * @returns Progress information suitable for progress bars
 *
 * @example
 * ```typescript
 * const progress = useLimitProgress('connections', connectionCount)
 * return (
 *   <ProgressBar
 *     value={progress.percentage}
 *     max={100}
 *     color={progress.color}
 *   />
 * )
 * ```
 */
export function useLimitProgress(
  limitName: keyof TierLimits,
  currentUsage: number
): {
  percentage: number
  color: 'green' | 'yellow' | 'red'
  isUnlimited: boolean
} {
  const limitResult = useTierLimit(limitName, currentUsage)

  return useMemo(
    () => ({
      percentage: limitResult.percentage,
      color: limitResult.colorIndicator,
      isUnlimited: limitResult.isUnlimited,
    }),
    [limitResult.percentage, limitResult.colorIndicator, limitResult.isUnlimited]
  )
}
