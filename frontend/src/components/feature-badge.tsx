/**
 * Feature Badge Component
 *
 * Displays a badge indicating which tier a feature requires.
 * Supports multiple variants for different use cases.
 *
 * Usage:
 * ```tsx
 * <FeatureBadge tier="individual" variant="inline" />
 * <FeatureBadge tier="team" variant="tooltip">Hover for info</FeatureBadge>
 * ```
 */

import * as React from 'react'
import { Sparkles, Lock, Users, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { TierLevel } from '@/types/tiers'

interface FeatureBadgeProps {
  tier: TierLevel
  variant?: 'inline' | 'tooltip' | 'banner' | 'pill'
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
  children?: React.ReactNode
  showIcon?: boolean
}

const tierConfig = {
  local: {
    label: 'Free',
    color: 'bg-gray-500/10 text-gray-700 dark:text-gray-300 border-gray-500/20',
    gradient: 'from-gray-500 to-gray-600',
    icon: Lock,
  },
  individual: {
    label: 'Pro',
    color: 'bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/20',
    gradient: 'from-purple-500 to-pink-500',
    icon: Sparkles,
  },
  team: {
    label: 'Team',
    color: 'bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/20',
    gradient: 'from-blue-500 to-cyan-500',
    icon: Users,
  },
}

const sizeClasses = {
  xs: 'text-[10px] px-1.5 py-0.5 gap-0.5',
  sm: 'text-xs px-2 py-0.5 gap-1',
  md: 'text-sm px-2.5 py-1 gap-1.5',
  lg: 'text-base px-3 py-1.5 gap-2',
}

export function FeatureBadge({
  tier,
  variant = 'inline',
  size = 'sm',
  className,
  children,
  showIcon = true,
}: FeatureBadgeProps) {
  const config = tierConfig[tier]
  const Icon = config.icon

  const badgeContent = (
    <span
      className={cn(
        'inline-flex items-center font-semibold rounded-full border transition-all',
        config.color,
        sizeClasses[size],
        className
      )}
    >
      {showIcon && <Icon className={cn('shrink-0', size === 'xs' ? 'h-2.5 w-2.5' : 'h-3 w-3')} />}
      <span>{config.label}</span>
    </span>
  )

  if (variant === 'tooltip' && children) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex items-center gap-1.5">
              {children}
              {badgeContent}
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <p className="font-medium">Requires {config.label} tier</p>
            <p className="text-xs text-muted-foreground mt-1">
              {tier === 'individual' && 'Unlock with Individual plan ($9/mo)'}
              {tier === 'team' && 'Unlock with Team plan ($29/mo)'}
              {tier === 'local' && 'Available in free tier'}
            </p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  if (variant === 'banner') {
    return (
      <div
        className={cn(
          'flex items-center gap-2 px-4 py-2 rounded-lg border',
          config.color,
          'bg-gradient-to-r',
          config.gradient,
          'bg-opacity-5',
          className
        )}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <span className="font-semibold">{config.label} Feature</span>
        {children && <span className="text-sm opacity-90">{children}</span>}
      </div>
    )
  }

  if (variant === 'pill') {
    return (
      <span
        className={cn(
          'inline-flex items-center gap-2 px-3 py-1 rounded-full font-medium',
          config.color,
          'shadow-sm',
          className
        )}
      >
        <span className={cn('h-2 w-2 rounded-full bg-gradient-to-r', config.gradient)} />
        {config.label}
        {children && <span className="text-xs opacity-75">{children}</span>}
      </span>
    )
  }

  // inline variant
  return badgeContent
}

/**
 * Animated Feature Badge with glow effect
 */
export function AnimatedFeatureBadge({
  tier,
  className,
}: {
  tier: TierLevel
  className?: string
}) {
  const config = tierConfig[tier]
  const Icon = config.icon

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full font-semibold text-xs',
        config.color,
        'animate-pulse',
        'shadow-lg',
        className
      )}
    >
      <Icon className="h-3 w-3 shrink-0" />
      <span>{config.label}</span>
      <Zap className="h-2.5 w-2.5 shrink-0 opacity-75" />
    </span>
  )
}

/**
 * Gradient Feature Badge with enhanced styling
 */
export function GradientFeatureBadge({
  tier,
  className,
}: {
  tier: TierLevel
  className?: string
}) {
  const config = tierConfig[tier]
  const Icon = config.icon

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full font-bold text-xs text-white',
        'bg-gradient-to-r',
        config.gradient,
        'shadow-lg',
        'ring-2 ring-white/20',
        className
      )}
    >
      <Icon className="h-3.5 w-3.5 shrink-0" />
      <span className="uppercase tracking-wide">{config.label}</span>
    </span>
  )
}
