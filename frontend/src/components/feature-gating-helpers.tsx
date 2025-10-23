/**
 * Feature Gating Helper Components
 *
 * Wrapper components for use in hooks to avoid circular dependencies.
 * These are used by useFeatureGate's render helpers.
 */

import * as React from 'react'
import type { ReactNode } from 'react'
import type { TierLevel } from '@/types/tiers'

// Lazy load components to avoid circular dependencies
const FeaturePreviewLazy = React.lazy(() =>
  import('./feature-preview').then((module) => ({ default: module.FeaturePreview }))
)

const LockedFeatureOverlayLazy = React.lazy(() =>
  import('./locked-feature-overlay').then((module) => ({ default: module.LockedFeatureOverlay }))
)

const FeatureBadgeLazy = React.lazy(() =>
  import('./feature-badge').then((module) => ({ default: module.FeatureBadge }))
)

interface PreviewWrapperProps {
  feature: string
  tier: TierLevel
  title: string
  description?: string
  screenshot?: string
  benefits?: string[]
  variant?: 'card' | 'overlay' | 'inline'
  children: ReactNode
}

export function PreviewWrapper({
  feature,
  tier,
  title,
  description,
  screenshot,
  benefits,
  variant = 'overlay',
  children,
}: PreviewWrapperProps) {
  return (
    <React.Suspense fallback={<div>{children}</div>}>
      <FeaturePreviewLazy
        feature={feature}
        tier={tier}
        title={title}
        description={description}
        screenshot={screenshot}
        benefits={benefits}
        variant={variant}
      >
        {children}
      </FeaturePreviewLazy>
    </React.Suspense>
  )
}

interface LockedWrapperProps {
  feature: string
  tier: TierLevel
  title: string
  benefits: string[]
  children: ReactNode
}

export function LockedWrapper({ feature, tier, title, benefits, children }: LockedWrapperProps) {
  return (
    <React.Suspense fallback={<div className="opacity-50">{children}</div>}>
      <LockedFeatureOverlayLazy
        feature={feature}
        requiredTier={tier}
        title={title}
        benefits={benefits}
      >
        {children}
      </LockedFeatureOverlayLazy>
    </React.Suspense>
  )
}

interface BadgeWrapperProps {
  tier: TierLevel
}

export function BadgeWrapper({ tier }: BadgeWrapperProps) {
  return (
    <React.Suspense fallback={null}>
      <FeatureBadgeLazy tier={tier} variant="inline" size="sm" />
    </React.Suspense>
  )
}
