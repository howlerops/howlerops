import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Card } from "@/components/ui/card"
import { ArrowRight, X } from "lucide-react"
import { ONBOARDING_STEPS } from "@/types/onboarding"

interface OnboardingProgressProps {
  currentStep: number
  completedSteps: number[]
  onContinue: () => void
  onDismiss: () => void
}

export function OnboardingProgress({
  currentStep,
  completedSteps,
  onContinue,
  onDismiss,
}: OnboardingProgressProps) {
  const totalSteps = ONBOARDING_STEPS.length
  const progress = (completedSteps.length / totalSteps) * 100

  return (
    <Card className="p-4 border-2 border-primary/20 bg-primary/5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold">Continue your setup</h3>
            <Button
              variant="ghost"
              size="icon"
              onClick={onDismiss}
              className="h-6 w-6 rounded-full"
            >
              <X className="h-3 w-3" />
            </Button>
          </div>

          <div className="space-y-2">
            <Progress value={progress} className="h-2" />
            <p className="text-sm text-muted-foreground">
              {completedSteps.length} of {totalSteps} steps completed
            </p>
          </div>

          <Button onClick={onContinue} size="sm" className="w-full">
            Resume Setup
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </Card>
  )
}
