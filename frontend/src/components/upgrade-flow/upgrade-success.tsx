/**
 * Upgrade Success Component
 *
 * Celebration screen after successful upgrade/trial start.
 * Shows next steps and features to explore.
 */

import { motion } from 'framer-motion'
import {
  ArrowRight,
  Brain,
  Calendar,
  CheckCircle2,
  Cloud,
  Smartphone,
  Users,
} from 'lucide-react'
import React, { useEffect } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export interface UpgradeSuccessProps {
  /**
   * Selected plan
   */
  plan: 'individual' | 'team'

  /**
   * Whether this is a trial
   */
  isTrial?: boolean

  /**
   * Trial end date
   */
  trialEndDate?: Date

  /**
   * Callback when user clicks "Get Started"
   */
  onGetStarted: () => void

  /**
   * Show sync progress
   */
  showSyncProgress?: boolean
}

/**
 * Next steps for each plan
 */
const NEXT_STEPS = {
  individual: [
    {
      icon: <Cloud className="w-5 h-5" />,
      title: 'Enable Cloud Sync',
      description: 'Your workspace is being synced to the cloud',
    },
    {
      icon: <Smartphone className="w-5 h-5" />,
      title: 'Install on Other Devices',
      description: 'Access your workspace from anywhere',
    },
    {
      icon: <Brain className="w-5 h-5" />,
      title: 'Try AI Features',
      description: 'Let AI remember your database schema',
    },
  ],
  team: [
    {
      icon: <Cloud className="w-5 h-5" />,
      title: 'Enable Cloud Sync',
      description: 'Your team workspace is being synced',
    },
    {
      icon: <Users className="w-5 h-5" />,
      title: 'Invite Team Members',
      description: 'Collaborate with up to 5 members',
    },
    {
      icon: <Brain className="w-5 h-5" />,
      title: 'Set Up Permissions',
      description: 'Configure role-based access control',
    },
  ],
}

/**
 * Trigger confetti animation
 * Uses canvas-confetti if available, otherwise skips
 */
async function triggerConfetti() {
  try {
    // Dynamic import for optional dependency
    const confetti = (await import('canvas-confetti')).default

    const duration = 3000
    const animationEnd = Date.now() + duration

    const interval = setInterval(() => {
      const timeLeft = animationEnd - Date.now()

      if (timeLeft <= 0) {
        return clearInterval(interval)
      }

      const particleCount = 50 * (timeLeft / duration)

      confetti({
        particleCount,
        spread: 70,
        origin: { y: 0.6 },
        colors: ['#3B82F6', '#A855F7', '#EC4899', '#10B981'],
      })
    }, 250)
  } catch {
    // Confetti library not available, skip animation
    console.log('canvas-confetti not installed, skipping celebration animation')
  }
}

/**
 * Upgrade Success Component
 *
 * @example
 * ```typescript
 * <UpgradeSuccess
 *   plan="individual"
 *   isTrial={true}
 *   trialEndDate={new Date(Date.now() + 14 * 24 * 60 * 60 * 1000)}
 *   onGetStarted={handleGetStarted}
 * />
 * ```
 */
export function UpgradeSuccess({
  plan,
  isTrial = true,
  trialEndDate,
  onGetStarted,
  showSyncProgress = true,
}: UpgradeSuccessProps) {
  const steps = NEXT_STEPS[plan]

  // Trigger confetti on mount
  useEffect(() => {
    triggerConfetti()
  }, [])

  const trialDaysLeft = trialEndDate
    ? Math.ceil((trialEndDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    : 14

  return (
    <div className="space-y-8">
      {/* Success Header */}
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5 }}
        className="text-center space-y-4"
      >
        <div className="flex justify-center">
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: 'spring', stiffness: 200 }}
            className="p-4 rounded-full bg-gradient-to-br from-green-400 to-blue-500"
          >
            <CheckCircle2 className="w-16 h-16 text-white" />
          </motion.div>
        </div>

        <div>
          <h1 className="text-3xl font-bold">
            {isTrial ? 'Trial Started!' : 'Welcome to Pro!'}
          </h1>
          <p className="text-lg text-muted-foreground mt-2">
            {isTrial
              ? `You now have full access to ${plan === 'individual' ? 'Individual' : 'Team'} features`
              : 'Your workspace has been upgraded'}
          </p>
        </div>

        {isTrial && trialEndDate && (
          <Badge
            variant="outline"
            className="text-base px-4 py-1.5 bg-gradient-to-r from-blue-50 to-purple-50"
          >
            <Calendar className="w-4 h-4 mr-2" />
            {trialDaysLeft} days remaining in your trial
          </Badge>
        )}
      </motion.div>

      {/* Sync Progress */}
      {showSyncProgress && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <Card className="bg-gradient-to-r from-blue-50 to-purple-50 border-blue-200 dark:from-blue-950 dark:to-purple-950">
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
                >
                  <Cloud className="w-5 h-5 text-blue-600" />
                </motion.div>
                <div className="flex-1">
                  <p className="font-medium text-sm">Syncing your workspace...</p>
                  <p className="text-xs text-muted-foreground">
                    This may take a few moments
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Next Steps */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        className="space-y-4"
      >
        <h2 className="text-xl font-semibold">Next steps</h2>
        <div className="space-y-3">
          {steps.map((step, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.5 + index * 0.1 }}
            >
              <Card>
                <CardContent className="p-4">
                  <div className="flex items-start gap-3">
                    <div className="p-2 rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900">
                      {step.icon}
                    </div>
                    <div className="flex-1">
                      <h3 className="font-medium">{step.title}</h3>
                      <p className="text-sm text-muted-foreground mt-1">
                        {step.description}
                      </p>
                    </div>
                    <CheckCircle2 className="w-5 h-5 text-green-600 mt-1" />
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      </motion.div>

      {/* Get Started Button */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.8 }}
        className="flex justify-center"
      >
        <Button
          size="lg"
          className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white px-12"
          onClick={onGetStarted}
        >
          Get Started
          <ArrowRight className="w-4 h-4 ml-2" />
        </Button>
      </motion.div>

      {/* Footer */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1 }}
        className="text-center text-sm text-muted-foreground"
      >
        <p>
          Need help getting started?{' '}
          <a href="#" className="text-blue-600 hover:underline">
            View our guide
          </a>{' '}
          or{' '}
          <a href="#" className="text-blue-600 hover:underline">
            contact support
          </a>
        </p>
      </motion.div>
    </div>
  )
}
