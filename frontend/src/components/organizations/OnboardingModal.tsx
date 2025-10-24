/**
 * Organization Onboarding Modal
 *
 * Shows a brief onboarding tour after user accepts their first invitation.
 * Features:
 * - 3-step wizard: Welcome → Explore → Get Started
 * - Brief tour of organization features
 * - Skip Tour option
 * - Sets onboarding complete flag in localStorage
 */

import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import {
  Users,
  Shield,
  Database,
  CheckCircle,
  ArrowRight,
  ArrowLeft,
} from 'lucide-react'

const ONBOARDING_KEY = 'organization-onboarding-completed'

interface OnboardingModalProps {
  /** Whether the modal is open */
  open: boolean
  /** Callback when modal is closed */
  onClose: () => void
  /** Organization name for personalization */
  organizationName?: string
}

interface OnboardingStep {
  title: string
  description: string
  icon: React.ReactNode
  content: React.ReactNode
}

export function OnboardingModal({
  open,
  onClose,
  organizationName = 'the organization',
}: OnboardingModalProps) {
  const [currentStep, setCurrentStep] = useState(0)

  const steps: OnboardingStep[] = [
    {
      title: 'Welcome to Your Organization!',
      description: 'You are now a member of ' + organizationName,
      icon: <Users className="h-12 w-12 text-primary" />,
      content: (
        <div className="space-y-4">
          <p className="text-muted-foreground">
            Congratulations! You've successfully joined{' '}
            <strong>{organizationName}</strong>. You can now collaborate with
            your team on shared database connections and queries.
          </p>
          <div className="rounded-lg bg-muted p-4">
            <h4 className="mb-2 font-semibold">What's Next?</h4>
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li className="flex items-start gap-2">
                <CheckCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-green-600" />
                <span>Access shared database connections</span>
              </li>
              <li className="flex items-start gap-2">
                <CheckCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-green-600" />
                <span>Collaborate on SQL queries with your team</span>
              </li>
              <li className="flex items-start gap-2">
                <CheckCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-green-600" />
                <span>View activity logs and member information</span>
              </li>
            </ul>
          </div>
        </div>
      ),
    },
    {
      title: 'Explore Organization Features',
      description: 'Learn what you can do as a member',
      icon: <Shield className="h-12 w-12 text-primary" />,
      content: (
        <div className="space-y-4">
          <p className="text-muted-foreground">
            As a member, you have access to various organization features based
            on your role.
          </p>
          <div className="space-y-3">
            <div className="flex gap-3 rounded-lg border p-3">
              <Database className="h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <h4 className="font-semibold">Shared Connections</h4>
                <p className="text-sm text-muted-foreground">
                  Access databases shared by your organization. Switch between
                  personal and organization connections easily.
                </p>
              </div>
            </div>
            <div className="flex gap-3 rounded-lg border p-3">
              <Users className="h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <h4 className="font-semibold">Team Collaboration</h4>
                <p className="text-sm text-muted-foreground">
                  View members, see who has access, and collaborate on shared
                  resources with your team.
                </p>
              </div>
            </div>
            <div className="flex gap-3 rounded-lg border p-3">
              <Shield className="h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <h4 className="font-semibold">Role-Based Access</h4>
                <p className="text-sm text-muted-foreground">
                  Your permissions are based on your role. Admins can manage
                  members and settings.
                </p>
              </div>
            </div>
          </div>
        </div>
      ),
    },
    {
      title: 'Get Started!',
      description: 'You are all set to begin',
      icon: <CheckCircle className="h-12 w-12 text-green-600" />,
      content: (
        <div className="space-y-4">
          <p className="text-muted-foreground">
            You're ready to start working with your organization!
          </p>
          <div className="rounded-lg bg-gradient-to-br from-primary/10 to-primary/5 p-6">
            <h4 className="mb-3 text-lg font-semibold">Quick Tips:</h4>
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li className="flex items-start gap-2">
                <span className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">
                  1
                </span>
                <span>
                  Use the organization switcher in the sidebar to toggle
                  between personal and organization contexts
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">
                  2
                </span>
                <span>
                  Check the Connections page to see shared database connections
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">
                  3
                </span>
                <span>
                  Visit Settings to view members, invitations, and audit logs
                </span>
              </li>
            </ul>
          </div>
          <p className="text-center text-sm font-medium text-primary">
            Happy collaborating!
          </p>
        </div>
      ),
    },
  ]

  const currentStepData = steps[currentStep]
  const progress = ((currentStep + 1) / steps.length) * 100

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      handleComplete()
    }
  }

  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1)
    }
  }

  const handleSkip = () => {
    handleComplete()
  }

  const handleComplete = () => {
    // Mark onboarding as completed
    localStorage.setItem(ONBOARDING_KEY, 'true')
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader className="space-y-4">
          <div className="flex flex-col items-center gap-4">
            {currentStepData.icon}
            <div className="text-center">
              <DialogTitle className="text-2xl">
                {currentStepData.title}
              </DialogTitle>
              <DialogDescription className="mt-2">
                {currentStepData.description}
              </DialogDescription>
            </div>
          </div>
          <Progress value={progress} className="h-2" />
          <p className="text-center text-xs text-muted-foreground">
            Step {currentStep + 1} of {steps.length}
          </p>
        </DialogHeader>

        <div className="py-4">{currentStepData.content}</div>

        <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between">
          <Button
            variant="ghost"
            onClick={handleSkip}
            className="order-2 sm:order-1"
          >
            Skip Tour
          </Button>
          <div className="order-1 flex gap-2 sm:order-2">
            {currentStep > 0 && (
              <Button variant="outline" onClick={handleBack}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back
              </Button>
            )}
            <Button onClick={handleNext}>
              {currentStep < steps.length - 1 ? (
                <>
                  Next
                  <ArrowRight className="ml-2 h-4 w-4" />
                </>
              ) : (
                <>
                  Get Started
                  <CheckCircle className="ml-2 h-4 w-4" />
                </>
              )}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Check if user has completed onboarding
 */
export function hasCompletedOnboarding(): boolean {
  return localStorage.getItem(ONBOARDING_KEY) === 'true'
}

/**
 * Reset onboarding state (for testing)
 */
export function resetOnboarding(): void {
  localStorage.removeItem(ONBOARDING_KEY)
}
