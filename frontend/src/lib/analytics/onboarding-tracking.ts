/**
 * Onboarding Analytics Tracking
 *
 * Tracks user onboarding flow and tutorial interactions.
 * Replace the console.log calls with your actual analytics service
 * (e.g., PostHog, Mixpanel, Google Analytics, etc.)
 */

export type OnboardingEvent =
  | "onboarding_started"
  | "onboarding_step_completed"
  | "onboarding_step_skipped"
  | "onboarding_completed"
  | "onboarding_abandoned"
  | "tutorial_started"
  | "tutorial_step_completed"
  | "tutorial_completed"
  | "tutorial_abandoned"
  | "feature_discovered"
  | "help_searched"
  | "video_watched"
  | "video_completed"
  | "interactive_example_run"
  | "empty_state_action_clicked"

export interface OnboardingEventProperties {
  step_number?: number
  step_name?: string
  tutorial_id?: string
  tutorial_step?: number
  feature_id?: string
  search_query?: string
  video_id?: string
  duration_watched?: number
  total_duration?: number
  example_id?: string
  empty_state_type?: string
  action?: string
  timestamp?: string
  user_id?: string
  session_id?: string
  [key: string]: unknown
}

class OnboardingTracker {
  private sessionId: string
  private startTime: number

  constructor() {
    this.sessionId = this.generateSessionId()
    this.startTime = Date.now()
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  track(event: OnboardingEvent, properties: OnboardingEventProperties = {}) {
    const enrichedProperties = {
      ...properties,
      timestamp: new Date().toISOString(),
      session_id: this.sessionId,
      session_duration: Date.now() - this.startTime,
    }

    // Replace with your analytics service
    console.log(`[Analytics] ${event}`, enrichedProperties)

    // Example integrations:

    // PostHog
    // if (window.posthog) {
    //   window.posthog.capture(event, enrichedProperties)
    // }

    // Mixpanel
    // if (window.mixpanel) {
    //   window.mixpanel.track(event, enrichedProperties)
    // }

    // Google Analytics
    // if (window.gtag) {
    //   window.gtag('event', event, enrichedProperties)
    // }

    // Amplitude
    // if (window.amplitude) {
    //   window.amplitude.getInstance().logEvent(event, enrichedProperties)
    // }
  }

  // Convenience methods for common events

  trackOnboardingStarted() {
    this.track("onboarding_started")
  }

  trackOnboardingStepCompleted(stepNumber: number, stepName: string) {
    this.track("onboarding_step_completed", {
      step_number: stepNumber,
      step_name: stepName,
    })
  }

  trackOnboardingStepSkipped(stepNumber: number, stepName: string) {
    this.track("onboarding_step_skipped", {
      step_number: stepNumber,
      step_name: stepName,
    })
  }

  trackOnboardingCompleted(totalDuration: number) {
    this.track("onboarding_completed", {
      total_duration: totalDuration,
    })
  }

  trackOnboardingAbandoned(stepNumber: number, stepName: string) {
    this.track("onboarding_abandoned", {
      step_number: stepNumber,
      step_name: stepName,
    })
  }

  trackTutorialStarted(tutorialId: string) {
    this.track("tutorial_started", {
      tutorial_id: tutorialId,
    })
  }

  trackTutorialStepCompleted(tutorialId: string, stepNumber: number) {
    this.track("tutorial_step_completed", {
      tutorial_id: tutorialId,
      tutorial_step: stepNumber,
    })
  }

  trackTutorialCompleted(tutorialId: string, totalDuration: number) {
    this.track("tutorial_completed", {
      tutorial_id: tutorialId,
      total_duration: totalDuration,
    })
  }

  trackTutorialAbandoned(tutorialId: string, stepNumber: number) {
    this.track("tutorial_abandoned", {
      tutorial_id: tutorialId,
      tutorial_step: stepNumber,
    })
  }

  trackFeatureDiscovered(featureId: string) {
    this.track("feature_discovered", {
      feature_id: featureId,
    })
  }

  trackHelpSearched(query: string) {
    this.track("help_searched", {
      search_query: query,
    })
  }

  trackVideoWatched(
    videoId: string,
    durationWatched: number,
    totalDuration: number
  ) {
    this.track("video_watched", {
      video_id: videoId,
      duration_watched: durationWatched,
      total_duration: totalDuration,
    })
  }

  trackVideoCompleted(videoId: string, totalDuration: number) {
    this.track("video_completed", {
      video_id: videoId,
      total_duration: totalDuration,
    })
  }

  trackInteractiveExampleRun(exampleId: string) {
    this.track("interactive_example_run", {
      example_id: exampleId,
    })
  }

  trackEmptyStateActionClicked(emptyStateType: string, action: string) {
    this.track("empty_state_action_clicked", {
      empty_state_type: emptyStateType,
      action: action,
    })
  }
}

// Export singleton instance
export const onboardingTracker = new OnboardingTracker()
