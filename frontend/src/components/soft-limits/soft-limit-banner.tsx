/**
 * Soft Limit Banner
 *
 * Persistent banner shown when soft limits are exceeded.
 * Non-blocking but persistent reminder with upgrade CTA.
 */

import { AnimatePresence,motion } from 'framer-motion'
import { AlertTriangle, Sparkles, TrendingUp,X } from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { useUpgradeModal } from '@/components/upgrade-modal'
import { cn } from '@/lib/utils'
import type { UpgradeTrigger } from '@/store/upgrade-prompt-store'

export interface SoftLimitBannerProps {
  /**
   * Trigger type for upgrade modal
   */
  trigger: UpgradeTrigger

  /**
   * Current usage value
   */
  usage: number

  /**
   * Soft limit value
   */
  softLimit: number

  /**
   * Banner title
   */
  title: string

  /**
   * Banner message
   */
  message: string

  /**
   * Call-to-action text
   * @default "Start Free Trial"
   */
  cta?: string

  /**
   * Whether the banner is dismissible
   * @default true
   */
  dismissible?: boolean

  /**
   * Storage key for dismissed state
   * If not provided, banner won't persist dismissal
   */
  storageKey?: string

  /**
   * Show usage progress bar
   * @default true
   */
  showProgress?: boolean

  /**
   * Additional CSS classes
   */
  className?: string
}

/**
 * Soft Limit Banner Component
 *
 * @example
 * ```typescript
 * <SoftLimitBanner
 *   trigger="queryHistory"
 *   usage={52}
 *   softLimit={50}
 *   title="You're storing more history than the free tier suggests"
 *   message="Upgrade to Pro for unlimited searchable history with advanced filters."
 *   cta="Start Free Trial"
 *   dismissible={true}
 *   storageKey="query-history-banner-dismissed"
 * />
 * ```
 */
export function SoftLimitBanner({
  trigger,
  usage,
  softLimit,
  title,
  message,
  cta = 'Start Free Trial',
  dismissible = true,
  storageKey,
  showProgress = true,
  className,
}: SoftLimitBannerProps) {
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()
  const [isDismissed, setIsDismissed] = useState(false)

  // Check if banner was dismissed
  useEffect(() => {
    if (storageKey) {
      const dismissed = localStorage.getItem(storageKey)
      if (dismissed) {
        setIsDismissed(true)
      }
    }
  }, [storageKey])

  const handleDismiss = () => {
    setIsDismissed(true)
    if (storageKey) {
      localStorage.setItem(storageKey, new Date().toISOString())
    }
  }

  const handleUpgrade = () => {
    showUpgradeModal(trigger)
  }

  const percentage = Math.min(100, (usage / softLimit) * 100)
  const isOverLimit = usage > softLimit

  if (isDismissed) {
    return <>{UpgradeModalComponent}</>
  }

  return (
    <>
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.3 }}
          className={cn(
            'border-b bg-gradient-to-r',
            isOverLimit
              ? 'from-orange-50 to-red-50 border-orange-200 dark:from-orange-950 dark:to-red-950 dark:border-orange-800'
              : 'from-yellow-50 to-orange-50 border-yellow-200 dark:from-yellow-950 dark:to-orange-950 dark:border-yellow-800',
            className
          )}
        >
          <div className="container mx-auto px-4 py-3">
            <div className="flex items-start gap-4">
              {/* Icon */}
              <div className={cn(
                'p-2 rounded-lg mt-0.5',
                isOverLimit
                  ? 'bg-orange-200 text-orange-700 dark:bg-orange-900'
                  : 'bg-yellow-200 text-yellow-700 dark:bg-yellow-900'
              )}>
                <AlertTriangle className="w-5 h-5" />
              </div>

              {/* Content */}
              <div className="flex-1 space-y-2">
                <div>
                  <h4 className="font-semibold text-sm">{title}</h4>
                  <p className="text-xs text-muted-foreground mt-1">{message}</p>
                </div>

                {/* Progress */}
                {showProgress && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-muted-foreground">
                        {usage} / {softLimit} (soft limit)
                      </span>
                      <span className={cn(
                        'font-medium',
                        isOverLimit ? 'text-orange-600' : 'text-yellow-600'
                      )}>
                        {percentage.toFixed(0)}%
                      </span>
                    </div>
                    <Progress
                      value={percentage}
                      className={cn(
                        'h-1.5',
                        isOverLimit ? 'bg-orange-200' : 'bg-yellow-200'
                      )}
                    />
                  </div>
                )}

                {/* Value Proposition */}
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <TrendingUp className="w-3 h-3" />
                  <span>
                    {isOverLimit
                      ? 'Upgrade now to avoid any interruptions'
                      : 'Upgrade for unlimited usage and premium features'}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
                  onClick={handleUpgrade}
                >
                  <Sparkles className="w-3 h-3 mr-1" />
                  {cta}
                </Button>
                {dismissible && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={handleDismiss}
                    className="h-8 w-8 p-0"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                )}
              </div>
            </div>
          </div>
        </motion.div>
      </AnimatePresence>
      {UpgradeModalComponent}
    </>
  )
}
