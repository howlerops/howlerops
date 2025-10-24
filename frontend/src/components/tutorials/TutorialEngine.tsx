import { useState, useEffect, useCallback } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { X, ArrowRight, ArrowLeft, CheckCircle2 } from "lucide-react"
import { Tutorial, TutorialStep, TutorialProgress } from "@/types/tutorial"
import { onboardingTracker } from "@/lib/analytics/onboarding-tracking"
import { cn } from "@/lib/utils"

interface TutorialEngineProps {
  tutorial: Tutorial
  open: boolean
  onOpenChange: (open: boolean) => void
  onComplete?: () => void
}

const TUTORIAL_STORAGE_KEY = "sql-studio-tutorials"

export function TutorialEngine({
  tutorial,
  open,
  onOpenChange,
  onComplete,
}: TutorialEngineProps) {
  const [currentStepIndex, setCurrentStepIndex] = useState(0)
  const [highlightedElement, setHighlightedElement] = useState<HTMLElement | null>(null)
  const [startTime] = useState(Date.now())

  const currentStep = tutorial.steps[currentStepIndex]
  const totalSteps = tutorial.steps.length
  const progress = ((currentStepIndex + 1) / totalSteps) * 100
  const isLastStep = currentStepIndex === totalSteps - 1

  // Highlight target element
  useEffect(() => {
    if (!open || !currentStep?.target) return

    const element = document.querySelector(currentStep.target) as HTMLElement
    if (element) {
      setHighlightedElement(element)

      // Add highlight styles
      element.style.position = "relative"
      element.style.zIndex = "9999"
      element.style.outline = "2px solid hsl(var(--primary))"
      element.style.outlineOffset = "4px"
      element.style.borderRadius = "4px"
      element.style.boxShadow = "0 0 0 9999px rgba(0, 0, 0, 0.5)"

      // Execute beforeShow hook
      currentStep.beforeShow?.()

      return () => {
        element.style.position = ""
        element.style.zIndex = ""
        element.style.outline = ""
        element.style.outlineOffset = ""
        element.style.borderRadius = ""
        element.style.boxShadow = ""
      }
    }
  }, [currentStep, open])

  // Track tutorial start
  useEffect(() => {
    if (open) {
      onboardingTracker.trackTutorialStarted(tutorial.id)
    }
  }, [open, tutorial.id])

  const handleNext = useCallback(async () => {
    // Execute onNext hook
    await currentStep?.onNext?.()

    // Track step completion
    onboardingTracker.trackTutorialStepCompleted(tutorial.id, currentStepIndex + 1)

    if (isLastStep) {
      handleComplete()
    } else {
      setCurrentStepIndex((prev) => prev + 1)
    }
  }, [currentStep, currentStepIndex, isLastStep, tutorial.id])

  const handlePrevious = useCallback(() => {
    setCurrentStepIndex((prev) => Math.max(0, prev - 1))
  }, [])

  const handleComplete = useCallback(() => {
    const duration = Date.now() - startTime
    onboardingTracker.trackTutorialCompleted(tutorial.id, duration)

    // Save completion to localStorage
    const saved = localStorage.getItem(TUTORIAL_STORAGE_KEY)
    const state = saved ? JSON.parse(saved) : { completedTutorials: [], progress: {} }

    if (!state.completedTutorials.includes(tutorial.id)) {
      state.completedTutorials.push(tutorial.id)
    }

    const progress: TutorialProgress = {
      tutorialId: tutorial.id,
      currentStep: totalSteps,
      completedSteps: Array.from({ length: totalSteps }, (_, i) => i),
      startedAt: new Date(startTime).toISOString(),
      completedAt: new Date().toISOString(),
    }

    state.progress[tutorial.id] = progress
    localStorage.setItem(TUTORIAL_STORAGE_KEY, JSON.stringify(state))

    onOpenChange(false)
    onComplete?.()
  }, [startTime, tutorial.id, totalSteps, onOpenChange, onComplete])

  const handleClose = useCallback(() => {
    onboardingTracker.trackTutorialAbandoned(tutorial.id, currentStepIndex + 1)
    onOpenChange(false)
  }, [tutorial.id, currentStepIndex, onOpenChange])

  // Keyboard navigation
  useEffect(() => {
    if (!open) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleClose()
      } else if (e.key === "ArrowRight" || e.key === "Enter") {
        handleNext()
      } else if (e.key === "ArrowLeft") {
        handlePrevious()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [open, handleClose, handleNext, handlePrevious])

  if (!open) return null

  return (
    <>
      {/* Tutorial Overlay */}
      <div className="fixed inset-0 z-[9998] pointer-events-none" />

      {/* Tutorial Dialog */}
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-md z-[10000]">
          <DialogHeader>
            <div className="flex items-start justify-between">
              <div className="flex-1 pr-8">
                <DialogTitle className="text-lg">{currentStep.title}</DialogTitle>
                <p className="text-xs text-muted-foreground mt-1">
                  {tutorial.name}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleClose}
                className="rounded-full"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          </DialogHeader>

          <div className="space-y-4">
            {/* Progress */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">
                  Step {currentStepIndex + 1} of {totalSteps}
                </span>
                <span className="text-sm font-medium">{Math.round(progress)}%</span>
              </div>
              <Progress value={progress} className="h-2" />
            </div>

            {/* Content */}
            <div className="prose prose-sm dark:prose-invert max-w-none">
              <div
                dangerouslySetInnerHTML={{ __html: currentStep.content }}
                className="text-sm"
              />
            </div>

            {/* Action instruction */}
            {currentStep.action && (
              <div className="p-3 rounded-lg bg-primary/10 border border-primary/20">
                <p className="text-sm font-medium">
                  {currentStep.action.type === "click" && "Click to continue"}
                  {currentStep.action.type === "input" && "Fill in the field"}
                  {currentStep.action.type === "wait" && "Wait for the action"}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {currentStep.action.instruction}
                </p>
              </div>
            )}

            {/* Navigation */}
            <div className="flex items-center justify-between pt-4">
              <Button
                variant="outline"
                onClick={handlePrevious}
                disabled={currentStepIndex === 0}
                size="sm"
              >
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>

              <div className="flex gap-1">
                {tutorial.steps.map((_, index) => (
                  <div
                    key={index}
                    className={cn(
                      "h-1.5 rounded-full transition-all",
                      index === currentStepIndex
                        ? "w-6 bg-primary"
                        : index < currentStepIndex
                        ? "w-1.5 bg-primary/50"
                        : "w-1.5 bg-muted"
                    )}
                  />
                ))}
              </div>

              <Button onClick={handleNext} size="sm">
                {isLastStep ? (
                  <>
                    Complete
                    <CheckCircle2 className="h-4 w-4 ml-2" />
                  </>
                ) : (
                  <>
                    Next
                    <ArrowRight className="h-4 w-4 ml-2" />
                  </>
                )}
              </Button>
            </div>

            {/* Keyboard shortcuts hint */}
            <div className="text-xs text-center text-muted-foreground pt-2 border-t">
              Use <kbd className="px-1.5 py-0.5 bg-muted rounded">←</kbd>{" "}
              <kbd className="px-1.5 py-0.5 bg-muted rounded">→</kbd> to navigate
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
