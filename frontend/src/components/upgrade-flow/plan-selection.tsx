/**
 * Plan Selection Component
 *
 * Step 1 of upgrade flow: Choose between Individual and Team plans.
 * Beautiful comparison with feature highlights.
 */

import React from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  Check,
  Cloud,
  Users,
  Sparkles,
  Database,
  History,
  Smartphone,
  Brain,
  Shield,
  FileDown,
  BookMarked,
  TrendingUp,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const FEATURES = {
  individual: [
    { icon: <Database className="w-4 h-4" />, name: 'Unlimited connections' },
    { icon: <History className="w-4 h-4" />, name: 'Unlimited query history' },
    { icon: <Cloud className="w-4 h-4" />, name: 'Cloud sync' },
    { icon: <Smartphone className="w-4 h-4" />, name: 'Multi-device support' },
    { icon: <Brain className="w-4 h-4" />, name: 'AI memory sync' },
    { icon: <BookMarked className="w-4 h-4" />, name: 'Unlimited saved queries' },
    { icon: <Shield className="w-4 h-4" />, name: 'Priority support' },
    { icon: <Sparkles className="w-4 h-4" />, name: 'Custom themes' },
    { icon: <FileDown className="w-4 h-4" />, name: '100MB exports' },
  ],
  team: [
    { icon: <Check className="w-4 h-4" />, name: 'Everything in Individual' },
    { icon: <Users className="w-4 h-4" />, name: 'Team sharing & collaboration' },
    { icon: <Shield className="w-4 h-4" />, name: 'Role-based access control' },
    { icon: <History className="w-4 h-4" />, name: 'Audit logging' },
    { icon: <FileDown className="w-4 h-4" />, name: '500MB exports' },
    { icon: <Users className="w-4 h-4" />, name: '5 included team members' },
    { icon: <TrendingUp className="w-4 h-4" />, name: 'Priority onboarding' },
  ],
}

export interface PlanSelectionProps {
  /**
   * Selected plan
   */
  selectedPlan: 'individual' | 'team'

  /**
   * Callback when plan changes
   */
  onPlanChange: (plan: 'individual' | 'team') => void

  /**
   * Billing period
   */
  billingPeriod: 'monthly' | 'annual'

  /**
   * Callback when billing period changes
   */
  onBillingPeriodChange: (period: 'monthly' | 'annual') => void

  /**
   * Callback for continuing to next step
   */
  onContinue: () => void

  /**
   * Show annual discount
   */
  showAnnualDiscount?: boolean
}

/**
 * Calculate price with discount
 */
function calculatePrice(
  basePrice: number,
  _billingPeriod: 'monthly' | 'annual'
): { monthly: number; annual: number; discount: number } {
  const annual = basePrice * 12 * 0.8 // 20% discount
  const discount = basePrice * 12 - annual

  return {
    monthly: basePrice,
    annual,
    discount,
  }
}

/**
 * Plan Selection Component
 *
 * @example
 * ```typescript
 * <PlanSelection
 *   selectedPlan={plan}
 *   onPlanChange={setPlan}
 *   billingPeriod={billing}
 *   onBillingPeriodChange={setBilling}
 *   onContinue={handleContinue}
 * />
 * ```
 */
export function PlanSelection({
  selectedPlan,
  onPlanChange,
  billingPeriod,
  onBillingPeriodChange,
  onContinue,
  showAnnualDiscount = true,
}: PlanSelectionProps) {
  const individualPrice = calculatePrice(9, billingPeriod)
  const teamPrice = calculatePrice(29, billingPeriod)

  return (
    <div className="space-y-8">
      {/* Billing Toggle */}
      {showAnnualDiscount && (
        <div className="flex items-center justify-center gap-4">
          <Label htmlFor="billing-toggle" className={cn(billingPeriod === 'monthly' && 'font-semibold')}>
            Monthly
          </Label>
          <Switch
            id="billing-toggle"
            checked={billingPeriod === 'annual'}
            onCheckedChange={(checked) => onBillingPeriodChange(checked ? 'annual' : 'monthly')}
          />
          <Label htmlFor="billing-toggle" className={cn('flex items-center gap-2', billingPeriod === 'annual' && 'font-semibold')}>
            Annual
            <Badge className="bg-green-100 text-green-700 border-green-200">
              Save 20%
            </Badge>
          </Label>
        </div>
      )}

      {/* Plan Cards */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Individual Plan */}
        <motion.div
          whileHover={{ scale: 1.02 }}
          transition={{ duration: 0.2 }}
        >
          <Card
            className={cn(
              'cursor-pointer transition-all h-full',
              selectedPlan === 'individual'
                ? 'ring-2 ring-blue-500 shadow-lg'
                : 'hover:border-blue-300'
            )}
            onClick={() => onPlanChange('individual')}
          >
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Cloud className="w-6 h-6 text-blue-600" />
                  <CardTitle className="text-xl">Individual</CardTitle>
                </div>
                {selectedPlan === 'individual' && (
                  <Badge className="bg-blue-600 text-white">
                    <Check className="w-3 h-3 mr-1" />
                    Selected
                  </Badge>
                )}
              </div>
              <div className="mt-4">
                <div className="flex items-baseline gap-2">
                  <span className="text-4xl font-bold text-blue-600">
                    ${billingPeriod === 'monthly' ? individualPrice.monthly : Math.floor(individualPrice.annual / 12)}
                  </span>
                  <span className="text-muted-foreground">/month</span>
                </div>
                {billingPeriod === 'annual' && (
                  <p className="text-sm text-green-600 mt-1">
                    ${individualPrice.annual}/year · Save ${individualPrice.discount.toFixed(0)}
                  </p>
                )}
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Perfect for individual developers and data analysts
              </p>
              <div className="space-y-2">
                {FEATURES.individual.map((feature, idx) => (
                  <div key={idx} className="flex items-center gap-2 text-sm">
                    <div className="text-green-600">{feature.icon}</div>
                    <span>{feature.name}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>

        {/* Team Plan */}
        <motion.div
          whileHover={{ scale: 1.02 }}
          transition={{ duration: 0.2 }}
        >
          <Card
            className={cn(
              'cursor-pointer transition-all h-full relative',
              selectedPlan === 'team'
                ? 'ring-2 ring-purple-500 shadow-lg'
                : 'hover:border-purple-300'
            )}
            onClick={() => onPlanChange('team')}
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
                  <Users className="w-6 h-6 text-purple-600" />
                  <CardTitle className="text-xl">Team</CardTitle>
                </div>
                {selectedPlan === 'team' && (
                  <Badge className="bg-purple-600 text-white">
                    <Check className="w-3 h-3 mr-1" />
                    Selected
                  </Badge>
                )}
              </div>
              <div className="mt-4">
                <div className="flex items-baseline gap-2">
                  <span className="text-4xl font-bold text-purple-600">
                    ${billingPeriod === 'monthly' ? teamPrice.monthly : Math.floor(teamPrice.annual / 12)}
                  </span>
                  <span className="text-muted-foreground">/month</span>
                </div>
                {billingPeriod === 'annual' && (
                  <p className="text-sm text-green-600 mt-1">
                    ${teamPrice.annual}/year · Save ${teamPrice.discount.toFixed(0)}
                  </p>
                )}
                <p className="text-xs text-muted-foreground mt-1">per team</p>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                For teams that need collaboration and advanced features
              </p>
              <div className="space-y-2">
                {FEATURES.team.map((feature, idx) => (
                  <div key={idx} className="flex items-center gap-2 text-sm">
                    <div className="text-green-600">{feature.icon}</div>
                    <span>{feature.name}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* Continue Button */}
      <div className="flex justify-center">
        <Button
          size="lg"
          className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white px-12"
          onClick={onContinue}
        >
          Continue to Trial
          <Sparkles className="w-4 h-4 ml-2" />
        </Button>
      </div>

      {/* Footer */}
      <div className="text-center space-y-2">
        <p className="text-sm text-muted-foreground">
          14-day free trial · No credit card required · Cancel anytime
        </p>
        <p className="text-xs text-muted-foreground">
          Questions? <a href="#" className="text-blue-600 hover:underline">Contact sales</a>
        </p>
      </div>
    </div>
  )
}
