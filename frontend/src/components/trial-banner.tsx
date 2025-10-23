/**
 * Trial Banner Component
 *
 * Promotes free trial with compelling messaging.
 * Dismissible and tracks user interactions.
 *
 * Usage:
 * ```tsx
 * <TrialBanner
 *   daysRemaining={14}
 *   onStartTrial={() => handleStartTrial()}
 *   onDismiss={() => handleDismiss()}
 * />
 * ```
 */

import * as React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Sparkles, X, Clock, CheckCircle, ArrowRight, Zap, Gift } from 'lucide-react'
import { cn } from '@/lib/utils'
import { UpgradeButton, FloatingUpgradeButton } from './upgrade-button'
import { FeatureBadge } from './feature-badge'
import type { TierLevel } from '@/types/tiers'

interface TrialBannerProps {
  tier?: TierLevel
  daysRemaining?: number
  features?: string[]
  variant?: 'banner' | 'modal' | 'card' | 'floating'
  dismissible?: boolean
  showCountdown?: boolean
  position?: 'top' | 'bottom'
  className?: string
  onStartTrial?: () => void
  onDismiss?: () => void
}

const DISMISS_DURATION_DAYS = 7
const DISMISS_KEY = 'trial_banner_dismissed_at'

function getDismissedAt(): Date | null {
  const stored = localStorage.getItem(DISMISS_KEY)
  return stored ? new Date(stored) : null
}

function setDismissedAt() {
  localStorage.setItem(DISMISS_KEY, new Date().toISOString())
}

function shouldShowBanner(): boolean {
  const dismissedAt = getDismissedAt()
  if (!dismissedAt) return true

  const daysSinceDismiss = (Date.now() - dismissedAt.getTime()) / (1000 * 60 * 60 * 24)
  return daysSinceDismiss >= DISMISS_DURATION_DAYS
}

export function TrialBanner({
  tier = 'individual',
  daysRemaining = 14,
  features = ['Full access to all features', 'No credit card required', 'Cancel anytime'],
  variant = 'banner',
  dismissible = true,
  showCountdown = true,
  position = 'top',
  className,
  onStartTrial,
  onDismiss,
}: TrialBannerProps) {
  const [isVisible, setIsVisible] = React.useState(shouldShowBanner())

  const handleDismiss = React.useCallback(() => {
    setIsVisible(false)
    setDismissedAt()
    onDismiss?.()
  }, [onDismiss])

  const handleStartTrial = React.useCallback(() => {
    onStartTrial?.()
  }, [onStartTrial])

  if (!isVisible) return null

  if (variant === 'floating') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, scale: 0.9, y: position === 'bottom' ? 100 : -100 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.9, y: position === 'bottom' ? 100 : -100 }}
          className={cn(
            'fixed z-50 max-w-sm mx-4',
            position === 'top' ? 'top-4 right-4' : 'bottom-4 right-4',
            className
          )}
        >
          <div className="relative overflow-hidden rounded-lg border border-purple-500/20 bg-card shadow-2xl">
            {/* Animated background */}
            <div className="absolute inset-0 bg-gradient-to-br from-purple-500/10 via-transparent to-pink-500/10 animate-pulse" />

            {/* Dismiss button */}
            {dismissible && (
              <button
                onClick={handleDismiss}
                className="absolute top-2 right-2 z-10 p-1 rounded-full hover:bg-accent transition-colors"
                aria-label="Dismiss"
              >
                <X className="h-4 w-4" />
              </button>
            )}

            <div className="relative p-4 space-y-3">
              {/* Header */}
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center animate-pulse">
                  <Gift className="h-5 w-5 text-white" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="font-bold text-lg">Try Pro Free!</h3>
                  {showCountdown && (
                    <p className="text-sm text-muted-foreground flex items-center gap-1 mt-0.5">
                      <Clock className="h-3 w-3" />
                      {daysRemaining}-day trial
                    </p>
                  )}
                </div>
              </div>

              {/* Features */}
              <ul className="space-y-1.5">
                {features.slice(0, 3).map((feature, index) => (
                  <li key={index} className="flex items-start gap-2 text-sm">
                    <CheckCircle className="h-4 w-4 shrink-0 mt-0.5 text-green-500" />
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>

              {/* CTA */}
              <button
                onClick={handleStartTrial}
                className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 rounded-lg font-semibold text-sm text-white bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 shadow-lg hover:shadow-xl transform hover:scale-105 transition-all"
              >
                <Sparkles className="h-4 w-4" />
                Start Free Trial
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </motion.div>
      </AnimatePresence>
    )
  }

  if (variant === 'modal') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          onClick={dismissible ? handleDismiss : undefined}
        >
          <motion.div
            initial={{ scale: 0.9, y: 20 }}
            animate={{ scale: 1, y: 0 }}
            exit={{ scale: 0.9, y: 20 }}
            onClick={(e) => e.stopPropagation()}
            className={cn('relative max-w-md w-full mx-4', className)}
          >
            <div className="relative overflow-hidden rounded-lg border border-border bg-card shadow-2xl">
              {/* Gradient background */}
              <div className="absolute inset-0 bg-gradient-to-br from-purple-500/20 via-transparent to-pink-500/20" />

              {/* Dismiss button */}
              {dismissible && (
                <button
                  onClick={handleDismiss}
                  className="absolute top-3 right-3 z-10 p-1 rounded-full hover:bg-accent transition-colors"
                  aria-label="Dismiss"
                >
                  <X className="h-4 w-4" />
                </button>
              )}

              <div className="relative p-6 space-y-6">
                {/* Header */}
                <div className="text-center">
                  <div className="flex items-center justify-center mb-4">
                    <div className="relative">
                      <div className="absolute inset-0 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full blur-xl opacity-50 animate-pulse" />
                      <div className="relative w-16 h-16 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full flex items-center justify-center">
                        <Gift className="h-8 w-8 text-white" />
                      </div>
                    </div>
                  </div>
                  <h2 className="text-2xl font-bold mb-2">Try Pro for Free</h2>
                  {showCountdown && (
                    <p className="text-muted-foreground flex items-center justify-center gap-2">
                      <Clock className="h-4 w-4" />
                      {daysRemaining} days • No credit card required
                    </p>
                  )}
                </div>

                {/* Features */}
                <div className="space-y-3">
                  {features.map((feature, index) => (
                    <motion.div
                      key={index}
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: index * 0.1 }}
                      className="flex items-start gap-3 p-3 rounded-lg bg-muted/30"
                    >
                      <CheckCircle className="h-5 w-5 shrink-0 mt-0.5 text-green-500" />
                      <span className="text-sm">{feature}</span>
                    </motion.div>
                  ))}
                </div>

                {/* CTA */}
                <div className="space-y-2">
                  <FloatingUpgradeButton
                    trigger="trial_modal"
                    requiredTier={tier}
                    className="w-full"
                  >
                    Start Your Free Trial
                  </FloatingUpgradeButton>
                  {dismissible && (
                    <button
                      onClick={handleDismiss}
                      className="w-full text-sm text-muted-foreground hover:text-foreground transition-colors"
                    >
                      Maybe later
                    </button>
                  )}
                </div>
              </div>
            </div>
          </motion.div>
        </motion.div>
      </AnimatePresence>
    )
  }

  if (variant === 'card') {
    return (
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          className={cn('relative overflow-hidden rounded-lg border border-purple-500/20 bg-card shadow-lg', className)}
        >
          {/* Gradient background */}
          <div className="absolute inset-0 bg-gradient-to-br from-purple-500/10 via-transparent to-pink-500/10" />

          {/* Dismiss button */}
          {dismissible && (
            <button
              onClick={handleDismiss}
              className="absolute top-3 right-3 z-10 p-1 rounded-full hover:bg-accent transition-colors"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          )}

          <div className="relative p-6 space-y-4">
            {/* Header */}
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0 w-12 h-12 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                <Gift className="h-6 w-6 text-white" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-bold text-xl">Try Pro Free</h3>
                  <FeatureBadge tier={tier} variant="inline" />
                </div>
                {showCountdown && (
                  <p className="text-sm text-muted-foreground flex items-center gap-1">
                    <Clock className="h-3.5 w-3.5" />
                    {daysRemaining}-day free trial • No credit card required
                  </p>
                )}
              </div>
            </div>

            {/* Features */}
            <ul className="space-y-2">
              {features.map((feature, index) => (
                <motion.li
                  key={index}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.1 }}
                  className="flex items-start gap-2 text-sm"
                >
                  <CheckCircle className="h-4 w-4 shrink-0 mt-0.5 text-green-500" />
                  <span>{feature}</span>
                </motion.li>
              ))}
            </ul>

            {/* CTA */}
            <UpgradeButton
              trigger="trial_card"
              requiredTier={tier}
              variant="gradient"
              className="w-full"
              onClick={handleStartTrial}
            >
              Start Free Trial
            </UpgradeButton>
          </div>
        </motion.div>
      </AnimatePresence>
    )
  }

  // Default: banner variant
  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: position === 'top' ? -100 : 100 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: position === 'top' ? -100 : 100 }}
        className={cn(
          'relative overflow-hidden border-b border-purple-500/20 bg-gradient-to-r from-purple-500/10 via-transparent to-pink-500/10',
          className
        )}
      >
        <div className="container mx-auto px-4 py-3">
          <div className="flex items-center justify-between gap-4">
            {/* Left: Icon and message */}
            <div className="flex items-center gap-3 flex-1 min-w-0">
              <Sparkles className="h-5 w-5 text-purple-500 shrink-0 animate-pulse" />
              <div className="flex-1 min-w-0">
                <span className="font-semibold">Try Pro free for {daysRemaining} days</span>
                <span className="hidden sm:inline text-sm text-muted-foreground ml-2">
                  • All features unlocked • No credit card required
                </span>
              </div>
            </div>

            {/* Right: CTA and dismiss */}
            <div className="flex items-center gap-2 shrink-0">
              <UpgradeButton
                trigger="trial_banner"
                requiredTier={tier}
                size="sm"
                onClick={handleStartTrial}
              >
                Start Trial
              </UpgradeButton>
              {dismissible && (
                <button
                  onClick={handleDismiss}
                  className="p-1 rounded-full hover:bg-accent transition-colors"
                  aria-label="Dismiss"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  )
}

/**
 * Trial Countdown Component
 * Shows remaining trial time with urgency
 */
export function TrialCountdown({
  daysRemaining,
  className,
}: {
  daysRemaining: number
  className?: string
}) {
  const isUrgent = daysRemaining <= 3
  const isCritical = daysRemaining <= 1

  return (
    <div
      className={cn(
        'inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium',
        isCritical
          ? 'bg-red-500/10 text-red-700 dark:text-red-300 border border-red-500/20'
          : isUrgent
            ? 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-300 border border-yellow-500/20'
            : 'bg-blue-500/10 text-blue-700 dark:text-blue-300 border border-blue-500/20',
        className
      )}
    >
      <Clock className={cn('h-4 w-4 shrink-0', (isUrgent || isCritical) && 'animate-pulse')} />
      <span>
        {daysRemaining} {daysRemaining === 1 ? 'day' : 'days'} left in trial
      </span>
    </div>
  )
}

/**
 * Trial Expired Banner
 * Shows when trial has ended
 */
export function TrialExpiredBanner({
  tier = 'individual',
  className,
}: {
  tier?: TierLevel
  className?: string
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        'relative overflow-hidden border-b border-red-500/20 bg-gradient-to-r from-red-500/10 to-orange-500/10',
        className
      )}
    >
      <div className="container mx-auto px-4 py-3">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <Zap className="h-5 w-5 text-red-500 shrink-0" />
            <div className="flex-1 min-w-0">
              <span className="font-semibold">Your trial has ended</span>
              <span className="hidden sm:inline text-sm text-muted-foreground ml-2">
                • Subscribe now to keep your Pro features
              </span>
            </div>
          </div>
          <UpgradeButton trigger="trial_expired" requiredTier={tier} size="sm" variant="gradient">
            Subscribe Now
          </UpgradeButton>
        </div>
      </div>
    </motion.div>
  )
}
