export interface OnboardingState {
  isComplete: boolean
  currentStep: number
  completedSteps: number[]
  skippedSteps: number[]
  profile?: UserProfile
  startedAt?: string
  completedAt?: string
}

export interface UserProfile {
  name?: string
  useCases: UseCase[]
  role: UserRole
}

export type UseCase =
  | "personal_projects"
  | "team_collaboration"
  | "data_analysis"
  | "database_administration"
  | "learning_sql"

export type UserRole =
  | "developer"
  | "data_analyst"
  | "dba"
  | "student"
  | "other"

export type OnboardingStep =
  | "welcome"
  | "profile"
  | "connection"
  | "tour"
  | "first_query"
  | "features"
  | "path"

export interface OnboardingProgress {
  step: OnboardingStep
  total: number
  completed: number
  percentage: number
}

export interface TutorialState {
  completedTutorials: string[]
  activeTutorial?: string
  tutorialProgress: Record<string, number>
}

export const ONBOARDING_STEPS: OnboardingStep[] = [
  "welcome",
  "profile",
  "connection",
  "tour",
  "first_query",
  "features",
  "path",
]

export const USE_CASE_LABELS: Record<UseCase, string> = {
  personal_projects: "Personal projects",
  team_collaboration: "Team collaboration",
  data_analysis: "Data analysis",
  database_administration: "Database administration",
  learning_sql: "Learning SQL",
}

export const USER_ROLE_LABELS: Record<UserRole, string> = {
  developer: "Developer",
  data_analyst: "Data Analyst",
  dba: "Database Administrator",
  student: "Student",
  other: "Other",
}
