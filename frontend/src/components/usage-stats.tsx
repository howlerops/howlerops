/**
 * Usage Stats Component
 *
 * Shows current usage across all tier limits with visual progress indicators.
 * Helps users understand their current usage and upgrade value.
 */

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Database,
  History,
  Brain,
  BookMarked,
  Sparkles,
  TrendingUp,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react'
import { useTierStore } from '@/store/tier-store'
import { useConnectionStore } from '@/store/connection-store'
import { useUpgradeModal } from '@/components/upgrade-modal'
import { getQueryHistoryRepository } from '@/lib/storage/repositories/query-history-repository'
import { cn } from '@/lib/utils'

/**
 * Usage stat item configuration
 */
interface UsageStat {
  name: string
  icon: React.ReactNode
  current: number
  limit: number | null
  color: 'green' | 'yellow' | 'orange' | 'red'
  percentage: number
  isUnlimited: boolean
}

/**
 * Get color based on usage percentage
 */
function getUsageColorClass(percentage: number): string {
  if (percentage >= 100) return 'text-red-600'
  if (percentage >= 90) return 'text-orange-600'
  if (percentage >= 70) return 'text-yellow-600'
  return 'text-green-600'
}

/**
 * Get background color for progress bar
 */
function getProgressColor(percentage: number): string {
  if (percentage >= 100) return 'bg-red-600'
  if (percentage >= 90) return 'bg-orange-500'
  if (percentage >= 70) return 'bg-yellow-500'
  return 'bg-green-500'
}

export interface UsageStatsProps {
  /**
   * Show upgrade CTA button
   * @default true
   */
  showUpgradeCTA?: boolean

  /**
   * Compact mode (less padding)
   * @default false
   */
  compact?: boolean

  /**
   * Additional CSS classes
   */
  className?: string
}

/**
 * Usage Stats Component
 *
 * @example
 * ```typescript
 * // In settings panel
 * <UsageStats showUpgradeCTA />
 *
 * // Compact mode in sidebar
 * <UsageStats compact />
 * ```
 */
export function UsageStats({
  showUpgradeCTA = true,
  compact = false,
  className,
}: UsageStatsProps) {
  const { getLimits, currentTier } = useTierStore()
  const { connections } = useConnectionStore()
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()

  const [queryCount, setQueryCount] = useState(0)
  const [isLoading, setIsLoading] = useState(true)

  const limits = getLimits()
  const isPaidTier = currentTier !== 'local'

  // Load query count
  useEffect(() => {
    const loadCounts = async () => {
      try {
        const repo = getQueryHistoryRepository()
        const count = await repo.count()
        setQueryCount(count)
      } catch (error) {
        console.error('Failed to load usage stats:', error)
      } finally {
        setIsLoading(false)
      }
    }
    loadCounts()
  }, [])

  // Calculate usage stats
  const stats: UsageStat[] = [
    {
      name: 'Database Connections',
      icon: <Database className="w-5 h-5" />,
      current: connections.length,
      limit: limits.connections,
      ...calculateUsageMetrics(connections.length, limits.connections),
    },
    {
      name: 'Query History',
      icon: <History className="w-5 h-5" />,
      current: queryCount,
      limit: limits.queryHistory,
      ...calculateUsageMetrics(queryCount, limits.queryHistory),
    },
    {
      name: 'AI Memories',
      icon: <Brain className="w-5 h-5" />,
      current: 0, // TODO: Get actual count
      limit: limits.aiMemories,
      ...calculateUsageMetrics(0, limits.aiMemories),
    },
    {
      name: 'Saved Queries',
      icon: <BookMarked className="w-5 h-5" />,
      current: 0, // TODO: Get actual count
      limit: limits.savedQueries,
      ...calculateUsageMetrics(0, limits.savedQueries),
    },
  ]

  const hasAnyWarnings = stats.some((stat) => stat.percentage >= 70 && !stat.isUnlimited)
  const hasAnyCritical = stats.some((stat) => stat.percentage >= 90 && !stat.isUnlimited)

  if (isLoading) {
    return (
      <Card className={className}>
        <CardContent className={cn('py-8', compact && 'py-4')}>
          <div className="text-center text-muted-foreground">Loading usage stats...</div>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card className={className}>
        <CardHeader className={compact ? 'pb-3' : undefined}>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className={compact ? 'text-base' : 'text-lg'}>
                Usage & Limits
              </CardTitle>
              <CardDescription className={cn('mt-1', compact && 'text-xs')}>
                Your current usage against tier limits
              </CardDescription>
            </div>
            {hasAnyCritical && (
              <Badge variant="destructive" className="flex items-center gap-1">
                <AlertCircle className="w-3 h-3" />
                Action Needed
              </Badge>
            )}
            {hasAnyWarnings && !hasAnyCritical && (
              <Badge className="flex items-center gap-1 bg-orange-100 text-orange-700 border-orange-200">
                <AlertCircle className="w-3 h-3" />
                Warning
              </Badge>
            )}
          </div>
        </CardHeader>

        <CardContent className={cn('space-y-6', compact && 'space-y-4')}>
          {/* Usage Stats */}
          {stats.map((stat, index) => (
            <div key={index} className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className={cn(getUsageColorClass(stat.percentage))}>
                    {stat.icon}
                  </div>
                  <span className={cn('font-medium', compact && 'text-sm')}>
                    {stat.name}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  {stat.isUnlimited ? (
                    <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                      <CheckCircle2 className="w-3 h-3 mr-1" />
                      Unlimited
                    </Badge>
                  ) : (
                    <span className={cn(
                      'text-sm font-mono font-semibold',
                      getUsageColorClass(stat.percentage)
                    )}>
                      {stat.current}/{stat.limit}
                    </span>
                  )}
                </div>
              </div>

              {!stat.isUnlimited && (
                <>
                  <div className="relative w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className={cn('h-full transition-all duration-300', getProgressColor(stat.percentage))}
                      style={{ width: `${Math.min(100, stat.percentage)}%` }}
                    />
                  </div>

                  {/* Warning messages */}
                  {stat.percentage >= 100 && (
                    <p className="text-xs text-red-600">
                      Limit reached. {stat.name === 'Database Connections' ? 'Remove a connection or ' : ''}Upgrade for unlimited.
                    </p>
                  )}
                  {stat.percentage >= 90 && stat.percentage < 100 && (
                    <p className="text-xs text-orange-600">
                      {Math.floor((stat.limit! - stat.current) || 0)} remaining. Close to limit.
                    </p>
                  )}
                  {stat.percentage >= 70 && stat.percentage < 90 && (
                    <p className="text-xs text-yellow-600">
                      {Math.floor((stat.limit! - stat.current) || 0)} remaining.
                    </p>
                  )}
                </>
              )}
            </div>
          ))}

          {/* Upgrade CTA */}
          {!isPaidTier && showUpgradeCTA && (
            <div className={cn(
              'pt-4 border-t space-y-3',
              compact && 'pt-3'
            )}>
              <div className="flex items-start gap-2 text-sm">
                <TrendingUp className="w-4 h-4 text-blue-600 mt-0.5" />
                <p className="text-muted-foreground">
                  Upgrade to unlock unlimited connections, query history, and AI memories.
                </p>
              </div>
              <Button
                className="w-full bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
                onClick={() => showUpgradeModal('manual')}
              >
                <Sparkles className="w-4 h-4 mr-2" />
                Upgrade to Pro
              </Button>
            </div>
          )}

          {/* Paid tier message */}
          {isPaidTier && (
            <div className="pt-4 border-t">
              <div className="flex items-center gap-2 text-sm text-green-600">
                <CheckCircle2 className="w-4 h-4" />
                <p className="font-medium">
                  You're on the {currentTier === 'individual' ? 'Individual' : 'Team'} plan with unlimited resources
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
      {UpgradeModalComponent}
    </>
  )
}

/**
 * Calculate usage metrics
 */
function calculateUsageMetrics(current: number, limit: number | null): {
  percentage: number
  color: 'green' | 'yellow' | 'orange' | 'red'
  isUnlimited: boolean
} {
  if (limit === null) {
    return {
      percentage: 0,
      color: 'green',
      isUnlimited: true,
    }
  }

  const percentage = limit > 0 ? Math.min(100, (current / limit) * 100) : 0

  let color: 'green' | 'yellow' | 'orange' | 'red' = 'green'
  if (percentage >= 100) color = 'red'
  else if (percentage >= 90) color = 'orange'
  else if (percentage >= 70) color = 'yellow'

  return {
    percentage,
    color,
    isUnlimited: false,
  }
}
