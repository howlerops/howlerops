import { useState, useEffect } from "react";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { WelcomeStep } from "./steps/WelcomeStep";
import { ProfileStep } from "./steps/ProfileStep";
import { ConnectionStep } from "./steps/ConnectionStep";
import { TourStep } from "./steps/TourStep";
import { FirstQueryStep } from "./steps/FirstQueryStep";
import { FeaturesStep } from "./steps/FeaturesStep";
import { PathStep } from "./steps/PathStep";
import {
  OnboardingState,
  UserProfile,
  ONBOARDING_STEPS,
} from "@/types/onboarding";
import { onboardingTracker } from "@/lib/analytics/onboarding-tracking";

const STORAGE_KEY = "sql-studio-onboarding";

interface OnboardingWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete?: (path: string) => void;
}

export function OnboardingWizard({
  open,
  onOpenChange,
  onComplete,
}: OnboardingWizardProps) {
  const [currentStep, setCurrentStep] = useState(0);
  const [state, setState] = useState<OnboardingState>({
    isComplete: false,
    currentStep: 0,
    completedSteps: [],
    skippedSteps: [],
    startedAt: new Date().toISOString(),
  });

  // Load saved progress on mount
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        setState(parsed);
        setCurrentStep(parsed.currentStep);
      } catch (error) {
        console.error("Failed to load onboarding progress:", error);
      }
    } else {
      // Track onboarding start
      onboardingTracker.trackOnboardingStarted();
    }
  }, []);

  // Save progress whenever state changes
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  }, [state]);

  const totalSteps = ONBOARDING_STEPS.length;
  const progress = ((currentStep + 1) / totalSteps) * 100;

  const handleNext = () => {
    const stepName = ONBOARDING_STEPS[currentStep];
    onboardingTracker.trackOnboardingStepCompleted(currentStep + 1, stepName);

    setState((prev) => ({
      ...prev,
      currentStep: currentStep + 1,
      completedSteps: [...prev.completedSteps, currentStep],
    }));
    setCurrentStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setCurrentStep((prev) => Math.max(0, prev - 1));
    setState((prev) => ({
      ...prev,
      currentStep: Math.max(0, currentStep - 1),
    }));
  };

  const handleSkip = () => {
    const stepName = ONBOARDING_STEPS[currentStep];
    onboardingTracker.trackOnboardingStepSkipped(currentStep + 1, stepName);

    setState((prev) => ({
      ...prev,
      skippedSteps: [...prev.skippedSteps, currentStep],
    }));
    handleNext();
  };

  const handleProfileComplete = (profile: UserProfile) => {
    setState((prev) => ({ ...prev, profile }));
    handleNext();
  };

  const handleComplete = (path: string) => {
    const startTime = state.startedAt
      ? new Date(state.startedAt).getTime()
      : Date.now();
    const duration = Date.now() - startTime;

    onboardingTracker.trackOnboardingCompleted(duration);

    const completedState: OnboardingState = {
      ...state,
      isComplete: true,
      completedAt: new Date().toISOString(),
    };

    setState(completedState);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(completedState));

    onOpenChange(false);
    onComplete?.(path);
  };

  const handleClose = () => {
    const stepName = ONBOARDING_STEPS[currentStep];
    onboardingTracker.trackOnboardingAbandoned(currentStep + 1, stepName);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto p-0">
        {/* Header with progress */}
        <div className="sticky top-0 z-10 bg-background border-b border-border px-6 py-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-semibold">Welcome to Howlerops</h1>
              <span className="text-sm text-muted-foreground">
                Step {currentStep + 1} of {totalSteps}
              </span>
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
          <Progress value={progress} className="h-2" />
        </div>

        {/* Step content */}
        <div className="px-6 pb-6">
          {currentStep === 0 && (
            <WelcomeStep onNext={handleNext} onSkip={handleSkip} />
          )}
          {currentStep === 1 && (
            <ProfileStep
              onNext={handleProfileComplete}
              onBack={handleBack}
              initialProfile={state.profile}
            />
          )}
          {currentStep === 2 && (
            <ConnectionStep
              onNext={handleNext}
              onBack={handleBack}
              onSkip={handleSkip}
            />
          )}
          {currentStep === 3 && (
            <TourStep onNext={handleNext} onBack={handleBack} />
          )}
          {currentStep === 4 && (
            <FirstQueryStep onNext={handleNext} onBack={handleBack} />
          )}
          {currentStep === 5 && (
            <FeaturesStep onNext={handleNext} onBack={handleBack} />
          )}
          {currentStep === 6 && (
            <PathStep onComplete={handleComplete} onBack={handleBack} />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
