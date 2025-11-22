/**
 * Query History Indicator
 *
 * Shows current query history usage vs limit.
 * Only shows when approaching limit (40+ queries).
 */

import { History, Sparkles } from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { useUpgradeModal } from '@/components/upgrade-modal'
import { getQueryHistoryRepository } from '@/lib/storage/repositories/query-history-repository'
import { cn } from '@/lib/utils'
import { useTierStore } from '@/store/tier-store'

export interface QueryHistoryIndicatorProps {
  /**
   * Display variant
   */
  variant?: 'badge' | 'banner' | 'inline'

  /**
   * Minimum usage percentage to show indicator
   */
  showThreshold?: number

  /**
   * Show upgrade CTA
   */
  showUpgradeCTA?: boolean

  /**
   * Additional CSS classes
   */
  className?: string
}

/**
 * Query History Indicator Component
 *
 * @example
 * ```typescript
 * // Show in query history panel
 * <QueryHistoryIndicator variant="banner" showThreshold={80} />
 * ```
 */
export function QueryHistoryIndicator({
  variant = 'badge',
  showThreshold = 80,
  showUpgradeCTA = true,
  className,
}: QueryHistoryIndicatorProps) {
  const { checkLimit } = useTierStore()
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()
  const [queryCount, setQueryCount] = useState(0)
  const [isLoading, setIsLoading] = useState(true)

  // Load query count
  useEffect(() => {
    const loadCount = async () => {
      try {
        const repo = getQueryHistoryRepository()
        const count = await repo.count()
        setQueryCount(count)
      } catch (error) {
        console.error('Failed to load query count:', error)
      } finally {
        setIsLoading(false)
      }
    }
    loadCount()
  }, [])

  const limitCheck = checkLimit('queryHistory', queryCount)

  // Don't show if unlimited or below threshold
  if (limitCheck.isUnlimited || limitCheck.percentage < showThreshold) {
    return null
  }

  const { limit, percentage, remaining } = limitCheck
  const displayString = `${queryCount}/${limit}`
  const isWarning = percentage >= 90

  const handleUpgradeClick = () => {
    showUpgradeModal('queryHistory')
  }

  if (isLoading) {
    return null
  }

  // Badge variant
  if (variant === 'badge') {
    return (
      <>
        <Badge
          variant={isWarning ? 'destructive' : 'default'}
          className={cn('flex items-center gap-1.5 cursor-pointer', className)}
          onClick={handleUpgradeClick}
        >
          <History className="w-3 h-3" />
          <span className="font-mono text-xs">{displayString}</span>
        </Badge>
        {UpgradeModalComponent}
      </>
    )
  }

  // Banner variant
  if (variant === 'banner') {
    return (
      <>
        <div
          className={cn(
            'p-3 rounded-lg border flex items-center justify-between gap-4',
            isWarning
              ? 'bg-orange-50 border-orange-200 dark:bg-orange-950 dark:border-orange-800'
              : 'bg-yellow-50 border-yellow-200 dark:bg-yellow-950 dark:border-yellow-800',
            className
          )}
        >
          <div className="flex-1 space-y-2">
            <div className="flex items-center gap-2">
              <History className={cn('w-4 h-4', isWarning ? 'text-orange-600' : 'text-yellow-600')} />
              <span className="text-sm font-medium">
                {isWarning
                  ? 'Query history almost full'
                  : 'Query history filling up'}
              </span>
            </div>
            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{displayString} queries stored</span>
                {remaining !== null && (
                  <span>{remaining} remaining</span>
                )}
              </div>
              <Progress value={percentage} className="h-1.5" />
            </div>
            <p className="text-xs text-muted-foreground">
              {isWarning
                ? 'Older queries will be auto-deleted when limit is reached.'
                : 'Upgrade for unlimited searchable history.'}
            </p>
          </div>
          {showUpgradeCTA && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleUpgradeClick}
            >
              <Sparkles className="w-3 h-3 mr-1" />
              Upgrade
            </Button>
          )}
        </div>
        {UpgradeModalComponent}
      </>
    )
  }

  // Inline variant
  if (variant === 'inline') {
    return (
      <>
        <div className={cn('flex items-center gap-3', className)}>
          <div className="flex items-center gap-2">
            <History className={cn('w-4 h-4', isWarning ? 'text-orange-600' : 'text-yellow-600')} />
            <span className="text-sm">{displayString} queries</span>
          </div>
          {showUpgradeCTA && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 text-xs text-blue-600"
              onClick={handleUpgradeClick}
            >
              Upgrade for unlimited
            </Button>
          )}
        </div>
        {UpgradeModalComponent}
      </>
    )
  }

  return null
}
