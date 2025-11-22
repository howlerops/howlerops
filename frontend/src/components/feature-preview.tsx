/**
 * Feature Preview Component
 *
 * Shows locked features as interactive previews with upgrade prompts.
 * Creates desire by showing what users are missing.
 *
 * Usage:
 * ```tsx
 * <FeaturePreview
 *   feature="sync"
 *   title="Cloud Sync"
 *   description="Keep your workspace in sync across devices"
 *   screenshot="/images/sync-preview.png"
 *   tier="individual"
 * >
 *   <SyncStatusIndicator disabled />
 * </FeaturePreview>
 * ```
 */

import { motion } from 'framer-motion'
import { Eye, EyeOff,Lock, Sparkles } from 'lucide-react'
import * as React from 'react'

import { cn } from '@/lib/utils'
import type { TierLevel } from '@/types/tiers'

import { FeatureBadge, GradientFeatureBadge } from './feature-badge'
import { FloatingUpgradeButton,UpgradeButton } from './upgrade-button'

interface FeaturePreviewProps {
  feature: string
  tier: TierLevel
  title: string
  description?: string
  screenshot?: string
  benefits?: string[]
  children?: React.ReactNode
  className?: string
  variant?: 'card' | 'overlay' | 'inline'
}

export function FeaturePreview({
  feature,
  tier,
  title,
  description,
  screenshot,
  benefits = [],
  children,
  className,
  variant = 'overlay',
}: FeaturePreviewProps) {
  const [isHovered, setIsHovered] = React.useState(false)
  const [showPreview, setShowPreview] = React.useState(false)

  if (variant === 'card') {
    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className={cn(
          'group relative overflow-hidden rounded-lg border border-border bg-card shadow-lg',
          'hover:shadow-2xl hover:border-purple-500/50',
          'transition-all duration-300',
          className
        )}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        {/* Gradient Background */}
        <div className="absolute inset-0 bg-gradient-to-br from-purple-500/10 via-transparent to-pink-500/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

        {/* Content */}
        <div className="relative p-6 space-y-4">
          {/* Header */}
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2">
                <h3 className="font-semibold text-lg">{title}</h3>
                <GradientFeatureBadge tier={tier} />
              </div>
              {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            <Lock className="h-5 w-5 text-purple-500 shrink-0" />
          </div>

          {/* Screenshot Preview */}
          {screenshot && (
            <div className="relative rounded-md overflow-hidden border border-border bg-muted">
              <img
                src={screenshot}
                alt={`${title} preview`}
                className={cn(
                  'w-full h-48 object-cover transition-all duration-300',
                  !isHovered && 'blur-sm'
                )}
              />
              {!isHovered && (
                <div className="absolute inset-0 flex items-center justify-center bg-background/60">
                  <Eye className="h-8 w-8 text-muted-foreground" />
                </div>
              )}
            </div>
          )}

          {/* Children Preview */}
          {children && (
            <div
              className={cn(
                'relative rounded-md border border-dashed border-purple-500/30 bg-purple-500/5 p-4',
                'transition-all duration-300',
                !isHovered && 'opacity-50 blur-[1px]'
              )}
            >
              {children}
            </div>
          )}

          {/* Benefits */}
          {benefits.length > 0 && (
            <ul className="space-y-1.5">
              {benefits.map((benefit, index) => (
                <motion.li
                  key={index}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.1 }}
                  className="flex items-center gap-2 text-sm"
                >
                  <Sparkles className="h-3.5 w-3.5 shrink-0 text-purple-500" />
                  <span>{benefit}</span>
                </motion.li>
              ))}
            </ul>
          )}

          {/* CTA */}
          <UpgradeButton
            trigger={feature}
            feature={feature}
            requiredTier={tier}
            variant="gradient"
            className="w-full"
          >
            Unlock {title}
          </UpgradeButton>
        </div>
      </motion.div>
    )
  }

  if (variant === 'inline') {
    return (
      <div className={cn('relative inline-flex items-center gap-2', className)}>
        <button
          onClick={() => setShowPreview(!showPreview)}
          className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/20 hover:border-purple-500/40 transition-all"
        >
          <Lock className="h-3.5 w-3.5 text-purple-500" />
          <span className="text-sm font-medium">{title}</span>
          <FeatureBadge tier={tier} variant="inline" size="xs" />
        </button>

        {showPreview && (
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            className="absolute top-full left-0 mt-2 w-80 p-4 rounded-lg border border-border bg-card shadow-xl z-50"
          >
            <h4 className="font-semibold mb-2">{title}</h4>
            {description && <p className="text-sm text-muted-foreground mb-3">{description}</p>}
            <UpgradeButton
              trigger={feature}
              feature={feature}
              requiredTier={tier}
              size="sm"
              className="w-full"
            >
              Learn More
            </UpgradeButton>
          </motion.div>
        )}
      </div>
    )
  }

  // Default: overlay variant
  return (
    <div className={cn('relative group', className)}>
      {/* Background feature UI */}
      <div
        className={cn(
          'transition-all duration-300',
          'opacity-40 blur-[2px] grayscale',
          'group-hover:opacity-60 group-hover:blur-[1px]'
        )}
      >
        {children}
      </div>

      {/* Overlay */}
      <motion.div
        className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-background/95 via-background/90 to-background/95 backdrop-blur-sm"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        {/* Decorative gradient circles */}
        <div className="absolute inset-0 overflow-hidden pointer-events-none">
          <div className="absolute top-0 left-1/4 w-64 h-64 bg-purple-500/10 rounded-full blur-3xl animate-pulse" />
          <div className="absolute bottom-0 right-1/4 w-64 h-64 bg-pink-500/10 rounded-full blur-3xl animate-pulse delay-1000" />
        </div>

        {/* Content */}
        <motion.div
          initial={{ scale: 0.9, y: 20 }}
          animate={{ scale: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="relative z-10 max-w-lg mx-4 text-center"
        >
          {/* Icon */}
          <div className="flex items-center justify-center mb-4">
            <div className="relative">
              <div className="absolute inset-0 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full blur-lg opacity-50" />
              <div className="relative w-16 h-16 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full flex items-center justify-center">
                <Lock className="h-8 w-8 text-white" />
              </div>
            </div>
          </div>

          {/* Title & Badge */}
          <div className="flex items-center justify-center gap-2 mb-2">
            <h3 className="font-bold text-2xl">{title}</h3>
            <GradientFeatureBadge tier={tier} />
          </div>

          {/* Description */}
          {description && <p className="text-muted-foreground mb-6 max-w-md mx-auto">{description}</p>}

          {/* Benefits */}
          {benefits.length > 0 && (
            <ul className="inline-flex flex-col items-start gap-2 mb-6 text-left">
              {benefits.map((benefit, index) => (
                <motion.li
                  key={index}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: 0.2 + index * 0.1 }}
                  className="flex items-center gap-2 text-sm"
                >
                  <Sparkles className="h-4 w-4 shrink-0 text-purple-500" />
                  <span>{benefit}</span>
                </motion.li>
              ))}
            </ul>
          )}

          {/* CTA */}
          <FloatingUpgradeButton trigger={feature} feature={feature} requiredTier={tier}>
            Unlock {title}
          </FloatingUpgradeButton>

          {/* Preview Toggle */}
          <button
            onClick={() => setShowPreview(!showPreview)}
            className="mt-4 text-xs text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
          >
            {showPreview ? (
              <>
                <EyeOff className="h-3 w-3" />
                Hide Preview
              </>
            ) : (
              <>
                <Eye className="h-3 w-3" />
                Show Preview
              </>
            )}
          </button>
        </motion.div>
      </motion.div>

      {/* Show preview when toggled */}
      {showPreview && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="absolute inset-0 pointer-events-none"
        >
          {children}
        </motion.div>
      )}
    </div>
  )
}

/**
 * Grid Feature Preview
 * Shows multiple feature previews in a grid layout
 */
export function GridFeaturePreview({
  features,
  className,
}: {
  features: Array<Omit<FeaturePreviewProps, 'variant'>>
  className?: string
}) {
  return (
    <div className={cn('grid gap-6 md:grid-cols-2 lg:grid-cols-3', className)}>
      {features.map((feature) => (
        <FeaturePreview key={feature.feature} {...feature} variant="card" />
      ))}
    </div>
  )
}

/**
 * Compact Feature Preview
 * Minimal preview for tight spaces
 */
export function CompactFeaturePreview({
  feature,
  tier,
  title,
  className,
}: Pick<FeaturePreviewProps, 'feature' | 'tier' | 'title' | 'className'>) {
  return (
    <motion.button
      onClick={() => {
        window.dispatchEvent(
          new CustomEvent('showUpgradeDialog', {
            detail: { trigger: feature, feature, requiredTier: tier },
          })
        )
      }}
      className={cn(
        'w-full flex items-center justify-between p-3 rounded-lg border border-dashed border-purple-500/30',
        'bg-purple-500/5 hover:bg-purple-500/10',
        'transition-colors',
        className
      )}
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
    >
      <div className="flex items-center gap-2">
        <Lock className="h-4 w-4 text-purple-500 shrink-0" />
        <span className="text-sm font-medium">{title}</span>
      </div>
      <FeatureBadge tier={tier} variant="inline" size="xs" />
    </motion.button>
  )
}
