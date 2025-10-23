/**
 * Upgrade Button Component
 *
 * Contextual button that triggers upgrade modal with feature context.
 * Tracks which feature triggered the upgrade for analytics.
 *
 * Usage:
 * ```tsx
 * <UpgradeButton trigger="queryHistorySearch" size="sm">
 *   Unlock Search
 * </UpgradeButton>
 * ```
 */

import * as React from 'react'
import { Sparkles, Lock, Zap, Crown, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { TierLevel } from '@/types/tiers'
import { cn } from '@/lib/utils'

interface UpgradeButtonProps {
  trigger: string
  feature?: string
  requiredTier?: TierLevel
  size?: 'xs' | 'sm' | 'md' | 'lg'
  variant?: 'default' | 'outline' | 'ghost' | 'link' | 'gradient'
  icon?: 'sparkles' | 'lock' | 'zap' | 'crown' | 'arrow' | 'none'
  className?: string
  children?: React.ReactNode
  onClick?: () => void
}

const iconMap = {
  sparkles: Sparkles,
  lock: Lock,
  zap: Zap,
  crown: Crown,
  arrow: ArrowRight,
  none: null,
}

const sizeMap = {
  xs: 'sm',
  sm: 'sm',
  md: 'default',
  lg: 'lg',
} as const

export function UpgradeButton({
  trigger,
  feature,
  requiredTier = 'individual',
  size = 'md',
  variant = 'default',
  icon = 'sparkles',
  className,
  children = 'Upgrade',
  onClick,
}: UpgradeButtonProps) {
  const Icon = iconMap[icon]
  const buttonSize = sizeMap[size]

  const handleClick = React.useCallback(() => {
    // Dispatch custom event for upgrade modal
    window.dispatchEvent(
      new CustomEvent('showUpgradeDialog', {
        detail: {
          trigger,
          feature,
          requiredTier,
          timestamp: new Date().toISOString(),
        },
      })
    )

    // Track analytics
    if (window.gtag) {
      window.gtag('event', 'upgrade_button_click', {
        trigger,
        feature,
        required_tier: requiredTier,
      })
    }

    // Call custom onClick if provided
    onClick?.()
  }, [trigger, feature, requiredTier, onClick])

  if (variant === 'gradient') {
    return (
      <button
        onClick={handleClick}
        className={cn(
          'inline-flex items-center gap-2 px-4 py-2 rounded-lg font-semibold text-sm text-white',
          'bg-gradient-to-r from-purple-500 to-pink-500',
          'hover:from-purple-600 hover:to-pink-600',
          'shadow-lg hover:shadow-xl',
          'transform hover:scale-105',
          'transition-all duration-200',
          'ring-2 ring-purple-500/20 hover:ring-purple-500/40',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
          size === 'xs' && 'px-2 py-1 text-xs',
          size === 'sm' && 'px-3 py-1.5 text-sm',
          size === 'lg' && 'px-6 py-3 text-base',
          className
        )}
      >
        {Icon && <Icon className={cn('shrink-0', size === 'xs' ? 'h-3 w-3' : 'h-4 w-4')} />}
        <span>{children}</span>
      </button>
    )
  }

  return (
    <Button
      onClick={handleClick}
      variant={variant as any}
      size={buttonSize}
      className={cn(
        'gap-2',
        variant === 'default' && 'bg-purple-600 hover:bg-purple-700',
        className
      )}
    >
      {Icon && <Icon className={cn('shrink-0', size === 'xs' ? 'h-3 w-3' : 'h-4 w-4')} />}
      <span>{children}</span>
    </Button>
  )
}

/**
 * Compact Upgrade Link
 * Minimal link-style upgrade trigger
 */
export function UpgradeLink({
  trigger,
  feature,
  requiredTier = 'individual',
  className,
  children = 'Upgrade to unlock',
}: Omit<UpgradeButtonProps, 'size' | 'variant' | 'icon'>) {
  return (
    <UpgradeButton
      trigger={trigger}
      feature={feature}
      requiredTier={requiredTier}
      variant="link"
      icon="arrow"
      size="sm"
      className={cn('h-auto p-0 text-purple-600 dark:text-purple-400', className)}
    >
      {children}
    </UpgradeButton>
  )
}

/**
 * Upgrade Badge Button
 * Small badge-style upgrade button
 */
export function UpgradeBadgeButton({
  trigger,
  feature,
  requiredTier = 'individual',
  className,
}: Omit<UpgradeButtonProps, 'size' | 'variant' | 'icon' | 'children'>) {
  return (
    <button
      onClick={() => {
        window.dispatchEvent(
          new CustomEvent('showUpgradeDialog', {
            detail: { trigger, feature, requiredTier },
          })
        )
      }}
      className={cn(
        'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold',
        'bg-purple-500/10 text-purple-700 dark:text-purple-300',
        'border border-purple-500/20',
        'hover:bg-purple-500/20 hover:border-purple-500/40',
        'transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-purple-500',
        className
      )}
    >
      <Lock className="h-2.5 w-2.5 shrink-0" />
      <span>Pro</span>
    </button>
  )
}

/**
 * Floating Upgrade Button
 * Eye-catching floating button with animation
 */
export function FloatingUpgradeButton({
  trigger,
  feature,
  requiredTier = 'individual',
  className,
  children = 'Unlock Pro Features',
}: Omit<UpgradeButtonProps, 'size' | 'variant' | 'icon'>) {
  return (
    <button
      onClick={() => {
        window.dispatchEvent(
          new CustomEvent('showUpgradeDialog', {
            detail: { trigger, feature, requiredTier },
          })
        )
      }}
      className={cn(
        'group relative inline-flex items-center gap-2 px-6 py-3 rounded-full font-bold text-white',
        'bg-gradient-to-r from-purple-500 via-pink-500 to-purple-500',
        'bg-size-200 bg-pos-0 hover:bg-pos-100',
        'shadow-2xl hover:shadow-purple-500/50',
        'transform hover:scale-105',
        'transition-all duration-300',
        'ring-4 ring-purple-500/20',
        'animate-pulse',
        className
      )}
      style={{
        backgroundSize: '200% 100%',
        backgroundPosition: '0% 0%',
      }}
    >
      <Sparkles className="h-5 w-5 shrink-0 animate-spin" style={{ animationDuration: '3s' }} />
      <span>{children}</span>
      <ArrowRight className="h-5 w-5 shrink-0 group-hover:translate-x-1 transition-transform" />
    </button>
  )
}

/**
 * Inline Upgrade Prompt
 * Subtle inline text with upgrade link
 */
export function InlineUpgradePrompt({
  trigger,
  feature,
  requiredTier = 'individual',
  message = 'This feature requires',
  tierLabel = 'Pro',
  className,
}: Omit<UpgradeButtonProps, 'size' | 'variant' | 'icon' | 'children'> & {
  message?: string
  tierLabel?: string
}) {
  return (
    <p className={cn('text-sm text-muted-foreground', className)}>
      {message}{' '}
      <button
        onClick={() => {
          window.dispatchEvent(
            new CustomEvent('showUpgradeDialog', {
              detail: { trigger, feature, requiredTier },
            })
          )
        }}
        className="inline-flex items-center gap-1 font-semibold text-purple-600 dark:text-purple-400 hover:underline"
      >
        <Sparkles className="h-3 w-3 shrink-0" />
        {tierLabel}
      </button>
    </p>
  )
}
