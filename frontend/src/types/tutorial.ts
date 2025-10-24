export interface Tutorial {
  id: string
  name: string
  description: string
  category: TutorialCategory
  difficulty: "beginner" | "intermediate" | "advanced"
  estimatedMinutes: number
  steps: TutorialStep[]
  trigger?: "manual" | "auto" | "feature_discovery"
  requiredFeatures?: string[]
}

export interface TutorialStep {
  target: string // CSS selector
  title: string
  content: string // Supports markdown
  placement: "top" | "bottom" | "left" | "right" | "center"
  highlightClass?: string
  beforeShow?: () => void | Promise<void>
  onNext?: () => void | Promise<void>
  action?: {
    type: "click" | "input" | "wait"
    instruction: string
  }
}

export type TutorialCategory =
  | "basics"
  | "queries"
  | "collaboration"
  | "advanced"
  | "ai"
  | "optimization"

export interface TutorialProgress {
  tutorialId: string
  currentStep: number
  completedSteps: number[]
  startedAt: string
  completedAt?: string
}

export interface TutorialState {
  activeTutorial?: string
  completedTutorials: string[]
  progress: Record<string, TutorialProgress>
}
