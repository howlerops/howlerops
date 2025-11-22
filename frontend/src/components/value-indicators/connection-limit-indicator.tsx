/**
 * Connection Limit Indicator
 *
 * Shows current connection usage vs limit with visual feedback.
 * Provides gentle upgrade prompts when approaching limit.
 */

import { Database, Sparkles } from 'lucide-react'
import React from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useUpgradeModal } from '@/components/upgrade-modal'
import { cn } from '@/lib/utils'
import { useConnectionStore } from '@/store/connection-store'
import { useTierStore } from '@/store/tier-store'

export interface ConnectionLimitIndicatorProps {
  /**
   * Display variant
   * - badge: Small badge indicator
   * - inline: Inline text with icon
   * - full: Full card with progress
   */
  variant?: 'badge' | 'inline' | 'full'

  /**
   * Show upgrade CTA when near limit
   */
  showUpgradeCTA?: boolean

  /**
   * Additional CSS classes
   */
  className?: string
}

/**
 * Get color based on usage percentage
 */
function getUsageColor(percentage: number): string {
  if (percentage >= 100) return 'text-red-600'
  if (percentage >= 80) return 'text-orange-600'
  if (percentage >= 60) return 'text-yellow-600'
  return 'text-green-600'
}

/**
 * Get badge variant based on usage
 */
function getBadgeVariant(percentage: number): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (percentage >= 100) return 'destructive'
  if (percentage >= 80) return 'default'
  return 'secondary'
}

/**
 * Connection Limit Indicator Component
 *
 * @example
 * ```typescript
 * // Badge variant in header
 * <ConnectionLimitIndicator variant="badge" />
 *
 * // Inline variant in settings
 * <ConnectionLimitIndicator variant="inline" showUpgradeCTA />
 * ```
 */
export function ConnectionLimitIndicator({
  variant = 'badge',
  showUpgradeCTA = true,
  className,
}: ConnectionLimitIndicatorProps) {
  const { connections } = useConnectionStore()
  const { checkLimit } = useTierStore()
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()

  const currentUsage = connections.length
  const limitCheck = checkLimit('connections', currentUsage)

  // Don't show if unlimited
  if (limitCheck.isUnlimited) {
    return null
  }

  const { limit, percentage, isNearLimit, isAtLimit } = limitCheck
  const displayString = `${currentUsage}/${limit}`
  const colorClass = getUsageColor(percentage)

  const handleUpgradeClick = () => {
    showUpgradeModal('connections')
  }

  // Badge variant - minimal display
  if (variant === 'badge') {
    return (
      <>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge
                variant={getBadgeVariant(percentage)}
                className={cn('flex items-center gap-1.5 cursor-pointer', className)}
                onClick={handleUpgradeClick}
              >
                <Database className="w-3 h-3" />
                <span className="font-mono text-xs">{displayString}</span>
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              <p className="text-sm">
                {isAtLimit
                  ? 'Connection limit reached'
                  : isNearLimit
                    ? 'Approaching connection limit'
                    : 'Database connections'}
              </p>
              {(isNearLimit || isAtLimit) && (
                <p className="text-xs text-muted-foreground mt-1">
                  Click to upgrade for unlimited connections
                </p>
              )}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
        {UpgradeModalComponent}
      </>
    )
  }

  // Inline variant - text with icon
  if (variant === 'inline') {
    return (
      <>
        <div className={cn('flex items-center gap-2', className)}>
          <div className="flex items-center gap-1.5">
            <Database className={cn('w-4 h-4', colorClass)} />
            <span className={cn('text-sm font-medium', colorClass)}>
              {displayString} connections
            </span>
          </div>
          {(isNearLimit || isAtLimit) && showUpgradeCTA && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 text-xs text-blue-600 hover:text-blue-700"
              onClick={handleUpgradeClick}
            >
              <Sparkles className="w-3 h-3 mr-1" />
              Upgrade for unlimited
            </Button>
          )}
        </div>
        {UpgradeModalComponent}
      </>
    )
  }

  // Full variant - card with details
  if (variant === 'full') {
    return (
      <>
        <div className={cn('p-4 rounded-lg border bg-card', className)}>
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <Database className={cn('w-5 h-5', colorClass)} />
              <span className="font-semibold">Connections</span>
            </div>
            <span className={cn('text-2xl font-bold', colorClass)}>{displayString}</span>
          </div>

          {/* Progress bar */}
          <div className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
            <div
              className={cn(
                'h-full transition-all duration-300',
                percentage >= 100
                  ? 'bg-red-600'
                  : percentage >= 80
                    ? 'bg-orange-500'
                    : percentage >= 60
                      ? 'bg-yellow-500'
                      : 'bg-green-500'
              )}
              style={{ width: `${Math.min(100, percentage)}%` }}
            />
          </div>

          {/* Warning message */}
          {isAtLimit && (
            <p className="text-sm text-red-600 mt-3">
              Connection limit reached. Remove a connection or upgrade for unlimited.
            </p>
          )}
          {isNearLimit && !isAtLimit && (
            <p className="text-sm text-orange-600 mt-3">
              You're approaching your connection limit.
            </p>
          )}

          {/* Upgrade CTA */}
          {(isNearLimit || isAtLimit) && showUpgradeCTA && (
            <Button
              size="sm"
              className="w-full mt-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              onClick={handleUpgradeClick}
            >
              <Sparkles className="w-4 h-4 mr-2" />
              Upgrade for Unlimited Connections
            </Button>
          )}
        </div>
        {UpgradeModalComponent}
      </>
    )
  }

  return null
}
