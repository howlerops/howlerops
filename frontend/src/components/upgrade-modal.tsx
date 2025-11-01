/**
 * Upgrade Modal Component
 *
 * Beautiful, value-focused upgrade modal with contextual messaging.
 * Uses soft nudges instead of hard blocks.
 *
 * Features:
 * - Context-aware messaging per trigger
 * - Plan comparison cards
 * - Start Free Trial CTA (14 days, no credit card)
 * - Dismissible with cooldown
 * - Smooth animations
 */

import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Sparkles,
  Cloud,
  Users,
  Zap,
  TrendingUp,
  Lock,
  Check,
  Clock,
  Shield,
  Database,
  History,
  Smartphone,
  Brain,
  FileDown,
  ArrowRight,
} from 'lucide-react'
import { useTierStore } from '@/store/tier-store'
import { useUpgradePromptStore, type UpgradeTrigger, DISMISSAL_DURATIONS } from '@/store/upgrade-prompt-store'
import { TIER_METADATA } from '@/config/tier-limits'
import { cn } from '@/lib/utils'

/**
 * Trigger-specific messages
 */
const TRIGGER_MESSAGES: Record<
  UpgradeTrigger,
  {
    title: string
    description: string
    value: string
    icon: React.ReactNode
  }
> = {
  connections: {
    title: 'Growing your database portfolio?',
    description: "You've added 5 connections. Upgrade for unlimited connections synced across all your devices.",
    value: 'Never worry about limits again',
    icon: <Database className="w-6 h-6" />,
  },
  queryHistory: {
    title: 'Your query history is filling up',
    description: "You're approaching your query limit. Upgrade for unlimited searchable history.",
    value: 'Never lose a query again',
    icon: <History className="w-6 h-6" />,
  },
  multiDevice: {
    title: 'Working from a new device?',
    description: 'Upgrade to Pro and your entire workspace syncs automatically across all devices.',
    value: 'Seamless multi-device workflow',
    icon: <Smartphone className="w-6 h-6" />,
  },
  aiMemory: {
    title: 'Let AI remember your context',
    description: 'Unlock unlimited AI memory to remember your database schemas and query patterns.',
    value: 'Smarter AI assistance',
    icon: <Brain className="w-6 h-6" />,
  },
  export: {
    title: 'Need to export larger files?',
    description: 'Upgrade to export files up to 100MB (Individual) or 500MB (Team).',
    value: 'Export any dataset',
    icon: <FileDown className="w-6 h-6" />,
  },
  manual: {
    title: 'Upgrade SQL Studio',
    description: 'Unlock unlimited connections, cloud sync, and powerful collaboration features.',
    value: 'Supercharge your workflow',
    icon: <Sparkles className="w-6 h-6" />,
  },
  periodic: {
    title: 'Ready to level up?',
    description: "You've been using SQL Studio for a while. Discover what Pro can do for you.",
    value: 'Take your workflow to the next level',
    icon: <TrendingUp className="w-6 h-6" />,
  },
  feature: {
    title: 'This feature requires an upgrade',
    description: 'Unlock this feature and many more with Individual or Team tier.',
    value: 'Access all features',
    icon: <Lock className="w-6 h-6" />,
  },
}

/**
 * Plan features comparison
 */
const PLAN_FEATURES = {
  individual: [
    { name: 'Unlimited connections', icon: <Database className="w-4 h-4" /> },
    { name: 'Unlimited query history', icon: <History className="w-4 h-4" /> },
    { name: 'Cloud sync', icon: <Cloud className="w-4 h-4" /> },
    { name: 'Multi-device support', icon: <Smartphone className="w-4 h-4" /> },
    { name: 'AI memory sync', icon: <Brain className="w-4 h-4" /> },
    { name: 'Priority support', icon: <Shield className="w-4 h-4" /> },
    { name: 'Custom themes', icon: <Sparkles className="w-4 h-4" /> },
  ],
  team: [
    { name: 'Everything in Individual', icon: <Check className="w-4 h-4" /> },
    { name: 'Team sharing', icon: <Users className="w-4 h-4" /> },
    { name: 'Role-based access', icon: <Shield className="w-4 h-4" /> },
    { name: 'Audit logging', icon: <History className="w-4 h-4" /> },
    { name: '500MB exports', icon: <FileDown className="w-4 h-4" /> },
    { name: '5 team members', icon: <Users className="w-4 h-4" /> },
  ],
}

export interface UpgradeModalProps {
  /**
   * Whether the modal is open
   */
  open: boolean

  /**
   * Callback when modal is closed
   */
  onOpenChange: (open: boolean) => void

  /**
   * Trigger type for contextual messaging
   */
  trigger?: UpgradeTrigger

  /**
   * Recommended tier to highlight
   */
  recommendedTier?: 'individual' | 'team'
}

/**
 * Upgrade Modal Component
 *
 * @example
 * ```typescript
 * const [showUpgrade, setShowUpgrade] = useState(false)
 *
 * <UpgradeModal
 *   open={showUpgrade}
 *   onOpenChange={setShowUpgrade}
 *   trigger="connections"
 *   recommendedTier="individual"
 * />
 * ```
 */
export function UpgradeModal({
  open,
  onOpenChange,
  trigger = 'manual',
  recommendedTier = 'individual',
}: UpgradeModalProps) {
  const { _currentTier } = useTierStore()
  const { markShown, dismiss } = useUpgradePromptStore()
  const [selectedPlan, setSelectedPlan] = useState<'individual' | 'team'>(recommendedTier)

  const message = TRIGGER_MESSAGES[trigger]
  const individualMetadata = TIER_METADATA.individual
  const teamMetadata = TIER_METADATA.team

  // Mark as shown when opened
  useEffect(() => {
    if (open) {
      markShown(trigger)
    }
  }, [open, trigger, markShown])

  const handleDismiss = (hours?: number) => {
    if (hours !== undefined) {
      dismiss(trigger, hours)
    }
    onOpenChange(false)
  }

  const handleStartTrial = () => {
    // TODO: Implement trial start flow
    console.log('Starting trial for:', selectedPlan)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <motion.div
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="flex items-center gap-3"
          >
            <div className="p-3 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 text-white">
              {message.icon}
            </div>
            <div>
              <DialogTitle className="text-2xl">{message.title}</DialogTitle>
              <DialogDescription className="text-base mt-1">
                {message.description}
              </DialogDescription>
            </div>
          </motion.div>
        </DialogHeader>

        {/* Value Proposition */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="mt-6 p-4 rounded-lg bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-200 dark:from-blue-950 dark:to-purple-950 dark:border-blue-800"
        >
          <div className="flex items-center gap-3">
            <Zap className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <p className="text-lg font-semibold text-blue-900 dark:text-blue-100">
              {message.value}
            </p>
          </div>
        </motion.div>

        {/* Plan Selection */}
        <div className="mt-8 space-y-4">
          <h3 className="text-lg font-semibold">Choose your plan</h3>
          <div className="grid gap-4 md:grid-cols-2">
            {/* Individual Plan */}
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.3, delay: 0.2 }}
            >
              <Card
                className={cn(
                  'cursor-pointer transition-all hover:shadow-lg',
                  selectedPlan === 'individual'
                    ? 'ring-2 ring-blue-500 shadow-lg'
                    : 'hover:border-blue-300'
                )}
                onClick={() => setSelectedPlan('individual')}
              >
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Cloud className="w-5 h-5 text-blue-600" />
                      <CardTitle className="text-xl">{individualMetadata.name}</CardTitle>
                    </div>
                    {selectedPlan === 'individual' && (
                      <Badge className="bg-blue-600 text-white">
                        <Check className="w-3 h-3 mr-1" />
                        Selected
                      </Badge>
                    )}
                  </div>
                  <div className="mt-2">
                    <span className="text-3xl font-bold text-blue-600">$9</span>
                    <span className="text-muted-foreground">/month</span>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {PLAN_FEATURES.individual.map((feature, idx) => (
                    <div key={idx} className="flex items-center gap-2 text-sm">
                      <div className="text-green-600">{feature.icon}</div>
                      <span>{feature.name}</span>
                    </div>
                  ))}
                </CardContent>
              </Card>
            </motion.div>

            {/* Team Plan */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.3, delay: 0.3 }}
            >
              <Card
                className={cn(
                  'cursor-pointer transition-all hover:shadow-lg relative',
                  selectedPlan === 'team'
                    ? 'ring-2 ring-purple-500 shadow-lg'
                    : 'hover:border-purple-300'
                )}
                onClick={() => setSelectedPlan('team')}
              >
                <div className="absolute -top-3 -right-3">
                  <Badge className="bg-gradient-to-r from-purple-600 to-pink-600 text-white">
                    <Sparkles className="w-3 h-3 mr-1" />
                    Popular
                  </Badge>
                </div>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Users className="w-5 h-5 text-purple-600" />
                      <CardTitle className="text-xl">{teamMetadata.name}</CardTitle>
                    </div>
                    {selectedPlan === 'team' && (
                      <Badge className="bg-purple-600 text-white">
                        <Check className="w-3 h-3 mr-1" />
                        Selected
                      </Badge>
                    )}
                  </div>
                  <div className="mt-2">
                    <span className="text-3xl font-bold text-purple-600">$29</span>
                    <span className="text-muted-foreground">/month</span>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {PLAN_FEATURES.team.map((feature, idx) => (
                    <div key={idx} className="flex items-center gap-2 text-sm">
                      <div className="text-green-600">{feature.icon}</div>
                      <span>{feature.name}</span>
                    </div>
                  ))}
                </CardContent>
              </Card>
            </motion.div>
          </div>
        </div>

        {/* Trial Info */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.4 }}
          className="mt-6 p-4 rounded-lg bg-green-50 border border-green-200 dark:bg-green-950 dark:border-green-800"
        >
          <div className="flex items-start gap-3">
            <Clock className="w-5 h-5 text-green-600 mt-0.5" />
            <div>
              <p className="font-semibold text-green-900 dark:text-green-100">
                14-day free trial
              </p>
              <p className="text-sm text-green-700 dark:text-green-300 mt-1">
                No credit card required. Cancel anytime.
              </p>
            </div>
          </div>
        </motion.div>

        {/* Action Buttons */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.5 }}
          className="mt-8 flex items-center justify-between gap-4"
        >
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              onClick={() => handleDismiss(DISMISSAL_DURATIONS.short)}
            >
              Maybe Later
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleDismiss(DISMISSAL_DURATIONS.long)}
              className="text-xs text-muted-foreground"
            >
              Don't show for 30 days
            </Button>
          </div>
          <Button
            size="lg"
            className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
            onClick={handleStartTrial}
          >
            Start Free Trial
            <ArrowRight className="w-4 h-4 ml-2" />
          </Button>
        </motion.div>

        {/* Footer */}
        <div className="mt-6 pt-6 border-t text-center text-sm text-muted-foreground">
          <p>
            Questions? <a href="#" className="text-blue-600 hover:underline">Contact us</a> or{' '}
            <a href="#" className="text-blue-600 hover:underline">view pricing details</a>
          </p>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Hook to easily show upgrade modal - tightly coupled with UpgradeModal component
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useUpgradeModal() {
  const [open, setOpen] = useState(false)
  const [trigger, setTrigger] = useState<UpgradeTrigger>('manual')
  const { shouldShowPrompt } = useUpgradePromptStore()

  const showUpgradeModal = (triggerType: UpgradeTrigger) => {
    if (shouldShowPrompt(triggerType)) {
      setTrigger(triggerType)
      setOpen(true)
      return true
    }
    return false
  }

  const UpgradeModalComponent = (
    <UpgradeModal
      open={open}
      onOpenChange={setOpen}
      trigger={trigger}
    />
  )

  return {
    showUpgradeModal,
    UpgradeModalComponent,
  }
}
