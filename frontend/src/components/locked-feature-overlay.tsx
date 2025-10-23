/**
 * Locked Feature Overlay Component
 *
 * Semi-transparent overlay that shows locked features with upgrade prompt.
 * Displays benefits and allows users to preview the feature UI.
 *
 * Usage:
 * ```tsx
 * <LockedFeatureOverlay
 *   feature="queryHistorySearch"
 *   requiredTier="individual"
 *   title="Search your query history"
 *   benefits={["Full-text search", "Advanced filters", "Save favorites"]}
 * >
 *   <Input placeholder="Search queries..." disabled />
 * </LockedFeatureOverlay>
 * ```
 */

import * as React from 'react'
import { Lock, Sparkles, X, ChevronRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { FeatureBadge } from './feature-badge'
import { UpgradeButton } from './upgrade-button'
import type { TierLevel } from '@/types/tiers'

interface LockedFeatureOverlayProps {
  feature: string
  requiredTier: TierLevel
  title: string
  description?: string
  benefits?: string[]
  children: React.ReactNode
  dismissible?: boolean
  showPreview?: boolean
  className?: string
}

export function LockedFeatureOverlay({
  feature,
  requiredTier,
  title,
  description,
  benefits = [],
  children,
  dismissible = true,
  showPreview = true,
  className,
}: LockedFeatureOverlayProps) {
  const [isDismissed, setIsDismissed] = React.useState(false)

  return (
    <div className={cn('relative', className)}>
      {/* Disabled feature preview */}
      <div
        className={cn(
          'pointer-events-none transition-opacity duration-300',
          !isDismissed && 'opacity-50 blur-[1px]'
        )}
      >
        {children}
      </div>

      {/* Overlay */}
      <AnimatePresence>
        {!isDismissed && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          >
            {/* Lock Icon Background */}
            <div className="absolute inset-0 flex items-center justify-center opacity-5">
              <Lock className="h-32 w-32" />
            </div>

            {/* Content Card */}
            <motion.div
              initial={{ scale: 0.9, y: 20 }}
              animate={{ scale: 1, y: 0 }}
              exit={{ scale: 0.9, y: 20 }}
              className="relative z-10 max-w-md mx-4 p-6 bg-card border border-border rounded-lg shadow-2xl"
            >
              {/* Dismiss Button */}
              {dismissible && (
                <button
                  onClick={() => setIsDismissed(true)}
                  className="absolute top-3 right-3 p-1 rounded-full hover:bg-accent transition-colors"
                  aria-label="Dismiss overlay"
                >
                  <X className="h-4 w-4" />
                </button>
              )}

              {/* Header */}
              <div className="flex items-start gap-3 mb-4">
                <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                  <Lock className="h-5 w-5 text-white" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="font-semibold text-lg">{title}</h3>
                    <FeatureBadge tier={requiredTier} variant="inline" size="sm" />
                  </div>
                  {description && (
                    <p className="text-sm text-muted-foreground">{description}</p>
                  )}
                </div>
              </div>

              {/* Benefits List */}
              {benefits.length > 0 && (
                <ul className="space-y-2 mb-6">
                  {benefits.map((benefit, index) => (
                    <motion.li
                      key={index}
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: index * 0.1 }}
                      className="flex items-start gap-2 text-sm"
                    >
                      <ChevronRight className="h-4 w-4 shrink-0 mt-0.5 text-purple-500" />
                      <span>{benefit}</span>
                    </motion.li>
                  ))}
                </ul>
              )}

              {/* CTA */}
              <div className="flex flex-col gap-2">
                <UpgradeButton
                  trigger={feature}
                  feature={feature}
                  requiredTier={requiredTier}
                  variant="gradient"
                  icon="sparkles"
                  className="w-full"
                >
                  Unlock {requiredTier === 'individual' ? 'Pro' : 'Team'} Features
                </UpgradeButton>
                {showPreview && dismissible && (
                  <button
                    onClick={() => setIsDismissed(true)}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                  >
                    Preview feature
                  </button>
                )}
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

/**
 * Minimal Locked Overlay
 * Simpler version with just a lock icon and upgrade button
 */
export function MinimalLockedOverlay({
  feature,
  requiredTier,
  className,
  children,
}: Pick<LockedFeatureOverlayProps, 'feature' | 'requiredTier' | 'className' | 'children'>) {
  return (
    <div className={cn('relative', className)}>
      <div className="pointer-events-none opacity-40 blur-[0.5px]">{children}</div>
      <div className="absolute inset-0 flex items-center justify-center bg-background/60 backdrop-blur-[1px]">
        <div className="flex flex-col items-center gap-3 p-4">
          <div className="flex items-center gap-2 text-sm font-medium">
            <Lock className="h-4 w-4 text-purple-500" />
            <span>Locked</span>
          </div>
          <UpgradeButton
            trigger={feature}
            feature={feature}
            requiredTier={requiredTier}
            size="sm"
            variant="outline"
            icon="sparkles"
          >
            Unlock
          </UpgradeButton>
        </div>
      </div>
    </div>
  )
}

/**
 * Banner Locked Overlay
 * Shows as a banner above the locked content
 */
export function BannerLockedOverlay({
  feature,
  requiredTier,
  title,
  description,
  className,
  children,
}: Pick<
  LockedFeatureOverlayProps,
  'feature' | 'requiredTier' | 'title' | 'description' | 'className' | 'children'
>) {
  return (
    <div className={cn('space-y-3', className)}>
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center justify-between p-4 rounded-lg border border-purple-500/20 bg-gradient-to-r from-purple-500/5 to-pink-500/5"
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
            <Lock className="h-4 w-4 text-white" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-0.5">
              <h4 className="font-semibold text-sm">{title}</h4>
              <FeatureBadge tier={requiredTier} variant="inline" size="xs" />
            </div>
            {description && (
              <p className="text-xs text-muted-foreground truncate">{description}</p>
            )}
          </div>
        </div>
        <UpgradeButton
          trigger={feature}
          feature={feature}
          requiredTier={requiredTier}
          size="sm"
          variant="outline"
          icon="sparkles"
        >
          Unlock
        </UpgradeButton>
      </motion.div>
      <div className="opacity-50 pointer-events-none">{children}</div>
    </div>
  )
}

/**
 * Inline Locked State
 * Shows locked state inline with the content
 */
export function InlineLockedState({
  feature,
  requiredTier,
  message = 'This feature is locked',
  className,
}: Pick<LockedFeatureOverlayProps, 'feature' | 'requiredTier' | 'className'> & {
  message?: string
}) {
  return (
    <div
      className={cn(
        'flex items-center justify-center gap-3 p-6 rounded-lg border border-dashed border-purple-500/30 bg-purple-500/5',
        className
      )}
    >
      <Lock className="h-5 w-5 text-purple-500 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{message}</p>
        <p className="text-xs text-muted-foreground mt-0.5">
          Upgrade to {requiredTier === 'individual' ? 'Pro' : 'Team'} to unlock
        </p>
      </div>
      <UpgradeButton
        trigger={feature}
        feature={feature}
        requiredTier={requiredTier}
        size="sm"
        icon="sparkles"
      >
        Upgrade
      </UpgradeButton>
    </div>
  )
}
