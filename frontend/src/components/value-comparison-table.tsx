/**
 * Value Comparison Table Component
 *
 * Feature comparison table showing differences between tiers.
 * Highlights current tier and provides upgrade paths.
 *
 * Usage:
 * ```tsx
 * <ValueComparisonTable
 *   currentTier="local"
 *   highlightTier="individual"
 *   features={[
 *     { name: 'Connections', local: '5', individual: 'Unlimited', team: 'Unlimited' },
 *     { name: 'Cloud Sync', local: false, individual: true, team: true },
 *   ]}
 * />
 * ```
 */

import * as React from 'react'
import { motion } from 'framer-motion'
import { Check, X, Sparkles, Users, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import { UpgradeButton } from './upgrade-button'
import type { TierLevel } from '@/types/tiers'

type FeatureValue = string | number | boolean

interface Feature {
  name: string
  description?: string
  local: FeatureValue
  individual: FeatureValue
  team: FeatureValue
  category?: string
}

interface ValueComparisonTableProps {
  currentTier: TierLevel
  highlightTier?: TierLevel
  features: Feature[]
  showPricing?: boolean
  compact?: boolean
  className?: string
}

const tierConfig = {
  local: {
    name: 'Free',
    description: 'Perfect for getting started',
    price: '$0',
    priceLabel: 'Forever',
    icon: Zap,
    color: 'gray',
    gradient: 'from-gray-500 to-gray-600',
  },
  individual: {
    name: 'Pro',
    description: 'For power users',
    price: '$9',
    priceLabel: '/month',
    icon: Sparkles,
    color: 'purple',
    gradient: 'from-purple-500 to-pink-500',
  },
  team: {
    name: 'Team',
    description: 'For collaborative teams',
    price: '$29',
    priceLabel: '/month per team',
    icon: Users,
    color: 'blue',
    gradient: 'from-blue-500 to-cyan-500',
  },
}

function renderFeatureValue(value: FeatureValue, isAvailable: boolean) {
  if (typeof value === 'boolean') {
    return isAvailable ? (
      <Check className="h-5 w-5 text-green-500" />
    ) : (
      <X className="h-5 w-5 text-muted-foreground/30" />
    )
  }
  return <span className={cn('font-medium', !isAvailable && 'text-muted-foreground')}>{value}</span>
}

export function ValueComparisonTable({
  currentTier,
  highlightTier,
  features,
  showPricing = true,
  compact = false,
  className,
}: ValueComparisonTableProps) {
  const tiers: TierLevel[] = ['local', 'individual', 'team']
  const groupedFeatures = React.useMemo(() => {
    const groups = new Map<string, Feature[]>()
    features.forEach((feature) => {
      const category = feature.category || 'Features'
      if (!groups.has(category)) {
        groups.set(category, [])
      }
      groups.get(category)!.push(feature)
    })
    return groups
  }, [features])

  return (
    <div className={cn('overflow-hidden rounded-lg border border-border bg-card shadow-lg', className)}>
      {/* Header with tier names */}
      <div className="grid grid-cols-4 gap-4 p-4 border-b border-border bg-muted/30">
        <div className="font-semibold text-sm">Features</div>
        {tiers.map((tier) => {
          const config = tierConfig[tier]
          const Icon = config.icon
          const isHighlighted = tier === highlightTier
          const isCurrent = tier === currentTier

          return (
            <motion.div
              key={tier}
              className={cn(
                'text-center rounded-lg p-3 transition-all',
                isHighlighted && 'ring-2 ring-purple-500 shadow-lg',
                isCurrent && 'bg-purple-500/10'
              )}
              animate={isHighlighted ? { scale: [1, 1.02, 1] } : {}}
              transition={{ duration: 2, repeat: Infinity }}
            >
              <div className="flex items-center justify-center gap-2 mb-1">
                <Icon className={cn('h-4 w-4', `text-${config.color}-500`)} />
                <span className="font-bold">{config.name}</span>
              </div>
              {showPricing && !compact && (
                <div className="text-xs text-muted-foreground">
                  <div className="font-semibold text-foreground">{config.price}</div>
                  <div>{config.priceLabel}</div>
                </div>
              )}
              {isCurrent && (
                <div className="mt-2 text-xs font-medium text-purple-600 dark:text-purple-400">
                  Current Plan
                </div>
              )}
            </motion.div>
          )
        })}
      </div>

      {/* Feature rows */}
      <div className={cn('divide-y divide-border', compact ? 'text-sm' : '')}>
        {Array.from(groupedFeatures.entries()).map(([category, categoryFeatures]) => (
          <div key={category}>
            {/* Category header */}
            {groupedFeatures.size > 1 && (
              <div className="px-4 py-2 bg-muted/50 font-semibold text-xs uppercase tracking-wide text-muted-foreground">
                {category}
              </div>
            )}

            {/* Features in category */}
            {categoryFeatures.map((feature, index) => (
              <motion.div
                key={feature.name}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.05 }}
                className={cn(
                  'grid grid-cols-4 gap-4 p-4 hover:bg-muted/30 transition-colors',
                  compact && 'py-2'
                )}
              >
                <div>
                  <div className="font-medium">{feature.name}</div>
                  {feature.description && !compact && (
                    <div className="text-xs text-muted-foreground mt-0.5">{feature.description}</div>
                  )}
                </div>
                {tiers.map((tier) => (
                  <div key={tier} className="flex items-center justify-center">
                    {renderFeatureValue(feature[tier], feature[tier] !== false)}
                  </div>
                ))}
              </motion.div>
            ))}
          </div>
        ))}
      </div>

      {/* Footer with CTAs */}
      {!compact && (
        <div className="grid grid-cols-4 gap-4 p-4 border-t border-border bg-muted/30">
          <div />
          {tiers.map((tier) => {
            const isHighlighted = tier === highlightTier
            const isCurrent = tier === currentTier

            if (isCurrent || tier === 'local') {
              return <div key={tier} />
            }

            return (
              <div key={tier} className="flex justify-center">
                <UpgradeButton
                  trigger="comparison_table"
                  requiredTier={tier}
                  size="sm"
                  variant={isHighlighted ? 'gradient' : 'outline'}
                >
                  {isHighlighted ? 'Get Started' : 'Learn More'}
                </UpgradeButton>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

/**
 * Compact Comparison Card
 * Simplified comparison for smaller spaces
 */
export function CompactComparisonCard({
  _currentTier,
  targetTier,
  features,
  className,
}: {
  currentTier: TierLevel
  targetTier: TierLevel
  features: Array<{ name: string; current: FeatureValue; target: FeatureValue }>
  className?: string
}) {
  const targetConfig = tierConfig[targetTier]
  const Icon = targetConfig.icon

  return (
    <div className={cn('rounded-lg border border-border bg-card p-4 space-y-4', className)}>
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className={cn('w-10 h-10 rounded-full bg-gradient-to-br flex items-center justify-center', targetConfig.gradient)}>
          <Icon className="h-5 w-5 text-white" />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-bold">Upgrade to {targetConfig.name}</h3>
          <p className="text-sm text-muted-foreground">
            {targetConfig.price} {targetConfig.priceLabel}
          </p>
        </div>
      </div>

      {/* Feature comparison */}
      <div className="space-y-2">
        {features.map((feature, index) => (
          <motion.div
            key={feature.name}
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.1 }}
            className="flex items-center justify-between text-sm"
          >
            <span>{feature.name}</span>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">
                {typeof feature.current === 'boolean' ? (
                  feature.current ? (
                    <Check className="h-4 w-4 text-green-500" />
                  ) : (
                    <X className="h-4 w-4" />
                  )
                ) : (
                  feature.current
                )}
              </span>
              <span className="text-muted-foreground">→</span>
              <span className="font-semibold text-purple-600 dark:text-purple-400">
                {typeof feature.target === 'boolean' ? (
                  feature.target ? (
                    <Check className="h-4 w-4 text-green-500" />
                  ) : (
                    <X className="h-4 w-4" />
                  )
                ) : (
                  feature.target
                )}
              </span>
            </div>
          </motion.div>
        ))}
      </div>

      {/* CTA */}
      <UpgradeButton
        trigger="compact_comparison"
        requiredTier={targetTier}
        variant="gradient"
        className="w-full"
      >
        Upgrade Now
      </UpgradeButton>
    </div>
  )
}

/**
 * Mobile-Friendly Comparison
 * Stacked layout for mobile devices
 */
export function MobileComparison({
  currentTier,
  features,
  className,
}: {
  currentTier: TierLevel
  features: Feature[]
  className?: string
}) {
  const [selectedTier, setSelectedTier] = React.useState<TierLevel>('individual')
  const tiers: TierLevel[] = ['local', 'individual', 'team']

  return (
    <div className={cn('space-y-4', className)}>
      {/* Tier selector */}
      <div className="flex gap-2">
        {tiers.map((tier) => {
          const config = tierConfig[tier]
          const Icon = config.icon
          const isSelected = tier === selectedTier
          const isCurrent = tier === currentTier

          return (
            <button
              key={tier}
              onClick={() => setSelectedTier(tier)}
              className={cn(
                'flex-1 flex flex-col items-center gap-2 p-3 rounded-lg border transition-all',
                isSelected
                  ? 'border-purple-500 bg-purple-500/10 shadow-lg'
                  : 'border-border hover:border-purple-500/50',
                isCurrent && 'ring-2 ring-purple-500/20'
              )}
            >
              <Icon className={cn('h-5 w-5', isSelected ? 'text-purple-500' : 'text-muted-foreground')} />
              <span className={cn('text-sm font-medium', isSelected && 'text-purple-600 dark:text-purple-400')}>
                {config.name}
              </span>
              <span className="text-xs text-muted-foreground">{config.price}</span>
            </button>
          )
        })}
      </div>

      {/* Feature list for selected tier */}
      <div className="rounded-lg border border-border bg-card divide-y divide-border">
        {features.map((feature, index) => (
          <motion.div
            key={feature.name}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: index * 0.05 }}
            className="p-4"
          >
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium">{feature.name}</div>
                {feature.description && (
                  <div className="text-xs text-muted-foreground mt-0.5">{feature.description}</div>
                )}
              </div>
              <div className="flex-shrink-0 ml-3">
                {renderFeatureValue(feature[selectedTier], feature[selectedTier] !== false)}
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {/* CTA */}
      {selectedTier !== currentTier && selectedTier !== 'local' && (
        <UpgradeButton
          trigger="mobile_comparison"
          requiredTier={selectedTier}
          variant="gradient"
          className="w-full"
        >
          Upgrade to {tierConfig[selectedTier].name}
        </UpgradeButton>
      )}
    </div>
  )
}
