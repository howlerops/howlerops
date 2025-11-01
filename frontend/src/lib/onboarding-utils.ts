/**
 * Onboarding Utilities
 *
 * Utilities for managing organization onboarding state.
 */

const ONBOARDING_KEY = 'organization-onboarding-completed'

/**
 * Check if user has completed onboarding
 */
export function hasCompletedOnboarding(): boolean {
  return localStorage.getItem(ONBOARDING_KEY) === 'true'
}

/**
 * Mark onboarding as completed
 */
export function markOnboardingComplete(): void {
  localStorage.setItem(ONBOARDING_KEY, 'true')
}

/**
 * Reset onboarding state (for testing)
 */
export function resetOnboarding(): void {
  localStorage.removeItem(ONBOARDING_KEY)
}
