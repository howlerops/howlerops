/**
 * Soft Limit Warning Component
 *
 * Shows warnings when approaching or exceeding soft limits.
 * Creates urgency and promotes upgrades at the right moment.
 *
 * Usage:
 * ```tsx
 * <SoftLimitWarning
 *   limit="connections"
 *   current={5}
 *   soft={5}
 *   message="You've added 5 connections. Upgrade for unlimited."
 *   variant="banner"
 * />
 * ```
 */

import * as React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { AlertCircle, AlertTriangle, Info, X, TrendingUp, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import { UpgradeButton, UpgradeLink } from './upgrade-button'
import { FeatureBadge } from './feature-badge'
import { Progress } from '@/components/ui/progress'
import type { TierLevel } from '@/types/tiers'

interface SoftLimitWarningProps {
  limit: string
  current: number
  soft: number
  hard?: number
  message?: string
  variant?: 'banner' | 'toast' | 'inline' | 'badge' | 'card'
  severity?: 'info' | 'warning' | 'critical'
  dismissible?: boolean
  requiredTier?: TierLevel
  className?: string
  onDismiss?: () => void
  onUpgrade?: () => void
}

const severityConfig = {
  info: {
    icon: Info,
    color: 'text-blue-600 dark:text-blue-400',
    bgColor: 'bg-blue-500/10',
    borderColor: 'border-blue-500/20',
    progressColor: 'bg-blue-500',
  },
  warning: {
    icon: AlertTriangle,
    color: 'text-yellow-600 dark:text-yellow-400',
    bgColor: 'bg-yellow-500/10',
    borderColor: 'border-yellow-500/20',
    progressColor: 'bg-yellow-500',
  },
  critical: {
    icon: AlertCircle,
    color: 'text-red-600 dark:text-red-400',
    bgColor: 'bg-red-500/10',
    borderColor: 'border-red-500/20',
    progressColor: 'bg-red-500',
  },
}

function calculateSeverity(current: number, soft: number, hard?: number): 'info' | 'warning' | 'critical' {
  const percentage = (current / soft) * 100

  if (hard && current >= hard) return 'critical'
  if (percentage >= 100) return 'critical'
  if (percentage >= 80) return 'warning'
  return 'info'
}

export function SoftLimitWarning({
  limit,
  current,
  soft,
  hard,
  message,
  variant = 'banner',
  severity: explicitSeverity,
  dismissible = true,
  requiredTier = 'individual',
  className,
  onDismiss,
  onUpgrade,
}: SoftLimitWarningProps) {
  const [isDismissed, setIsDismissed] = React.useState(false)
  const severity = explicitSeverity || calculateSeverity(current, soft, hard)
  const config = severityConfig[severity]
  const Icon = config.icon
  const percentage = Math.min(100, (current / soft) * 100)
  const remaining = Math.max(0, soft - current)

  const handleDismiss = React.useCallback(() => {
    setIsDismissed(true)
    onDismiss?.()
  }, [onDismiss])

  const defaultMessage = React.useMemo(() => {
    if (hard && current >= hard) {
      return `You've reached the hard limit of ${hard} ${limit}. Upgrade to continue.`
    }
    if (current >= soft) {
      return `You've reached the limit of ${soft} ${limit}. Upgrade for unlimited.`
    }
    if (remaining <= 2) {
      return `Only ${remaining} ${limit} remaining. Upgrade for unlimited.`
    }
    return `You're using ${current} of ${soft} ${limit}. Upgrade for more.`
  }, [current, soft, hard, limit, remaining])

  if (isDismissed) return null

  if (variant === 'badge') {
    return (
      <motion.div
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        exit={{ scale: 0 }}
        className={cn(
          'inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium',
          config.bgColor,
          config.color,
          'border',
          config.borderColor,
          className
        )}
      >
        <Icon className="h-3 w-3 shrink-0" />
        <span>
          {current}/{soft}
        </span>
      </motion.div>
    )
  }

  if (variant === 'inline') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={{ opacity: 0, height: 0 }}
          className={cn('flex items-center gap-2 text-sm', config.color, className)}
        >
          <Icon className="h-4 w-4 shrink-0" />
          <span>{message || defaultMessage}</span>
          <UpgradeLink
            trigger={`limit_${limit}`}
            feature={limit}
            requiredTier={requiredTier}
            className="ml-1"
          >
            Upgrade
          </UpgradeLink>
        </motion.div>
      </AnimatePresence>
    )
  }

  if (variant === 'toast') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, y: 50, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 50, scale: 0.95 }}
          className={cn(
            'flex items-start gap-3 p-4 rounded-lg shadow-lg border backdrop-blur-sm',
            'bg-card/95',
            config.borderColor,
            className
          )}
        >
          <div className={cn('flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center', config.bgColor)}>
            <Icon className={cn('h-4 w-4', config.color)} />
          </div>
          <div className="flex-1 min-w-0 space-y-2">
            <p className="text-sm font-medium">{message || defaultMessage}</p>
            <div className="flex items-center gap-2">
              <Progress value={percentage} className={cn('flex-1 h-1', config.progressColor)} />
              <span className="text-xs text-muted-foreground whitespace-nowrap">
                {current}/{soft}
              </span>
            </div>
            <UpgradeButton
              trigger={`limit_${limit}`}
              feature={limit}
              requiredTier={requiredTier}
              size="sm"
              onClick={onUpgrade}
            >
              Upgrade Now
            </UpgradeButton>
          </div>
          {dismissible && (
            <button
              onClick={handleDismiss}
              className="flex-shrink-0 p-1 hover:bg-accent rounded transition-colors"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </motion.div>
      </AnimatePresence>
    )
  }

  if (variant === 'card') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          className={cn(
            'relative overflow-hidden rounded-lg border shadow-lg',
            config.borderColor,
            'bg-card',
            className
          )}
        >
          {/* Gradient background */}
          <div className={cn('absolute inset-0 opacity-5', config.bgColor)} />

          <div className="relative p-6 space-y-4">
            {/* Header */}
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className={cn('w-10 h-10 rounded-full flex items-center justify-center', config.bgColor)}>
                  <Icon className={cn('h-5 w-5', config.color)} />
                </div>
                <div>
                  <h3 className="font-semibold">Usage Limit Approaching</h3>
                  <p className="text-sm text-muted-foreground mt-0.5">{message || defaultMessage}</p>
                </div>
              </div>
              {dismissible && (
                <button
                  onClick={handleDismiss}
                  className="flex-shrink-0 p-1 hover:bg-accent rounded transition-colors"
                  aria-label="Dismiss"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>

            {/* Progress */}
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Current Usage</span>
                <span className="font-medium">
                  {current} / {soft} {limit}
                </span>
              </div>
              <Progress value={percentage} className="h-2" />
              {hard && current < hard && (
                <p className="text-xs text-muted-foreground">
                  Hard limit: {hard} {limit}
                </p>
              )}
            </div>

            {/* CTA */}
            <div className="flex items-center gap-2">
              <UpgradeButton
                trigger={`limit_${limit}`}
                feature={limit}
                requiredTier={requiredTier}
                variant="gradient"
                className="flex-1"
                onClick={onUpgrade}
              >
                Upgrade for Unlimited
              </UpgradeButton>
              <FeatureBadge tier={requiredTier} variant="inline" />
            </div>
          </div>
        </motion.div>
      </AnimatePresence>
    )
  }

  // Default: banner variant
  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -10 }}
        className={cn(
          'flex items-center gap-3 p-4 rounded-lg border',
          config.bgColor,
          config.borderColor,
          className
        )}
      >
        <Icon className={cn('h-5 w-5 shrink-0', config.color)} />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium">{message || defaultMessage}</p>
          <div className="flex items-center gap-2 mt-2">
            <Progress value={percentage} className="flex-1 h-1.5" />
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {current}/{soft}
            </span>
          </div>
        </div>
        <UpgradeButton
          trigger={`limit_${limit}`}
          feature={limit}
          requiredTier={requiredTier}
          size="sm"
          variant="outline"
          onClick={onUpgrade}
        >
          Upgrade
        </UpgradeButton>
        {dismissible && (
          <button
            onClick={handleDismiss}
            className="flex-shrink-0 p-1 hover:bg-accent rounded transition-colors"
            aria-label="Dismiss"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </motion.div>
    </AnimatePresence>
  )
}

/**
 * Usage Stats Card
 * Shows current usage with progress bars for multiple limits
 */
export function UsageStatsCard({
  limits,
  className,
}: {
  limits: Array<{
    name: string
    current: number
    max: number | null
    label?: string
  }>
  className?: string
}) {
  return (
    <div className={cn('rounded-lg border border-border bg-card p-4 space-y-4', className)}>
      <h3 className="font-semibold flex items-center gap-2">
        <TrendingUp className="h-4 w-4" />
        Usage Statistics
      </h3>
      <div className="space-y-3">
        {limits.map((limit) => {
          const isUnlimited = limit.max === null
          const percentage = isUnlimited || !limit.max ? 0 : Math.min(100, (limit.current / limit.max) * 100)
          const severity = calculateSeverity(limit.current, limit.max || Infinity)
          const _config = severityConfig[severity]

          return (
            <div key={limit.name} className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">{limit.label || limit.name}</span>
                <span className="text-muted-foreground">
                  {limit.current} / {isUnlimited ? '∞' : limit.max}
                </span>
              </div>
              {!isUnlimited && (
                <Progress value={percentage} className="h-1.5" />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

/**
 * Upgrade Prompt Card
 * Shows when user hits a limit with strong CTA
 */
export function UpgradePromptCard({
  title = 'Upgrade to Continue',
  description = "You've reached your limit. Upgrade to unlock unlimited access.",
  requiredTier = 'individual',
  benefits = [],
  className,
}: {
  title?: string
  description?: string
  requiredTier?: TierLevel
  benefits?: string[]
  className?: string
}) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      className={cn(
        'relative overflow-hidden rounded-lg border border-purple-500/20 bg-gradient-to-br from-purple-500/10 via-transparent to-pink-500/10 p-6 shadow-xl',
        className
      )}
    >
      <div className="relative z-10 space-y-4">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
            <Zap className="h-5 w-5 text-white" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="font-bold text-lg">{title}</h3>
            <p className="text-sm text-muted-foreground mt-1">{description}</p>
          </div>
        </div>

        {benefits.length > 0 && (
          <ul className="space-y-1.5 pl-13">
            {benefits.map((benefit, index) => (
              <li key={index} className="text-sm flex items-center gap-2">
                <span className="h-1 w-1 rounded-full bg-purple-500" />
                {benefit}
              </li>
            ))}
          </ul>
        )}

        <UpgradeButton
          trigger="limit_reached"
          requiredTier={requiredTier}
          variant="gradient"
          className="w-full"
        >
          Upgrade Now
        </UpgradeButton>
      </div>
    </motion.div>
  )
}
