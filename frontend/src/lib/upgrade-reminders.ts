/**
 * Upgrade Reminders System
 *
 * Smart reminder system that shows upgrade prompts at natural moments.
 * Respects user workflow and never interrupts critical tasks.
 *
 * Features:
 * - Periodic reminders for active users
 * - Natural pause detection
 * - Activity tracking
 * - Respect dismissals and cooldowns
 */

import { useTierStore } from '@/store/tier-store'
import type { UpgradeTrigger } from '@/store/upgrade-prompt-store'
import { useUpgradePromptStore } from '@/store/upgrade-prompt-store'

/**
 * Reminder configuration
 */
interface ReminderConfig {
  /** Minimum time between reminders (in days) */
  minDaysBetween: number

  /** Minimum queries in session before showing */
  minQueriesInSession: number

  /** Only show on specific days (0 = Sunday, 5 = Friday) */
  preferredDays?: number[]

  /** Only show during specific hours (24-hour format) */
  preferredHours?: [number, number]

  /** Minimum days since first launch */
  minDaysSinceFirstLaunch?: number
}

const DEFAULT_CONFIG: ReminderConfig = {
  minDaysBetween: 7,
  minQueriesInSession: 10,
  preferredDays: [5], // Friday afternoons
  preferredHours: [14, 18], // 2 PM - 6 PM
  minDaysSinceFirstLaunch: 7,
}

/**
 * Check if current time is within preferred hours
 */
function isPreferredTime(config: ReminderConfig): boolean {
  const now = new Date()

  // Check day of week
  if (config.preferredDays && config.preferredDays.length > 0) {
    const dayOfWeek = now.getDay()
    if (!config.preferredDays.includes(dayOfWeek)) {
      return false
    }
  }

  // Check hour of day
  if (config.preferredHours) {
    const hour = now.getHours()
    const [startHour, endHour] = config.preferredHours
    if (hour < startHour || hour >= endHour) {
      return false
    }
  }

  return true
}

/**
 * Check if user has been active enough
 */
function isActiveUser(config: ReminderConfig): boolean {
  const promptStore = useUpgradePromptStore.getState()
  const { firstLaunch } = promptStore

  // Check session queries
  const sessionQueries = sessionStorage.getItem('session-query-count')
  const sessionQueryCount = sessionQueries ? parseInt(sessionQueries, 10) : 0

  if (sessionQueryCount < config.minQueriesInSession) {
    return false
  }

  // Check days since first launch
  if (config.minDaysSinceFirstLaunch && firstLaunch) {
    const daysSinceFirstLaunch = Math.floor(
      (new Date().getTime() - new Date(firstLaunch).getTime()) / (1000 * 60 * 60 * 24)
    )

    if (daysSinceFirstLaunch < config.minDaysSinceFirstLaunch) {
      return false
    }
  }

  return true
}

/**
 * Check if we're in a critical workflow
 * (Don't show reminders during these times)
 */
function isInCriticalWorkflow(): boolean {
  // Check if user is currently editing
  const isEditing = document.querySelector('[contenteditable="true"]') !== null

  // Check if there's a modal open
  const hasModal = document.querySelector('[role="dialog"]') !== null

  // Check if query is running
  const isQueryRunning = sessionStorage.getItem('query-running') === 'true'

  return isEditing || hasModal || isQueryRunning
}

/**
 * Should show periodic reminder
 */
export function shouldShowPeriodicReminder(config: Partial<ReminderConfig> = {}): boolean {
  const finalConfig = { ...DEFAULT_CONFIG, ...config }
  const tierStore = useTierStore.getState()
  const promptStore = useUpgradePromptStore.getState()

  // Don't show if already on paid tier
  if (tierStore.currentTier !== 'local') {
    return false
  }

  // Don't show if in critical workflow
  if (isInCriticalWorkflow()) {
    return false
  }

  // Check if user is active enough
  if (!isActiveUser(finalConfig)) {
    return false
  }

  // Check if it's a good time
  if (!isPreferredTime(finalConfig)) {
    return false
  }

  // Check if we should show prompt (respects cooldown and dismissals)
  if (!promptStore.shouldShowPrompt('periodic')) {
    return false
  }

  return true
}

/**
 * Show reminder at app launch
 */
export function checkAppLaunchReminder(): boolean {
  const tierStore = useTierStore.getState()
  const promptStore = useUpgradePromptStore.getState()

  // Don't show if already on paid tier
  if (tierStore.currentTier !== 'local') {
    return false
  }

  // Check if enough time has passed since first launch
  if (promptStore.firstLaunch) {
    const daysSinceFirstLaunch = Math.floor(
      (new Date().getTime() - new Date(promptStore.firstLaunch).getTime()) / (1000 * 60 * 60 * 24)
    )

    if (daysSinceFirstLaunch < 7) {
      return false // Too soon
    }
  }

  // Check if we should show prompt
  if (!promptStore.shouldShowPrompt('periodic')) {
    return false
  }

  return true
}

/**
 * Track query execution for activity monitoring
 */
export function trackQueryExecution() {
  const promptStore = useUpgradePromptStore.getState()
  promptStore.incrementQueryCount()

  // Track session queries
  const sessionQueries = sessionStorage.getItem('session-query-count')
  const count = sessionQueries ? parseInt(sessionQueries, 10) : 0
  sessionStorage.setItem('session-query-count', (count + 1).toString())

  // Check if we should show reminder after Nth query
  const sessionQueryCount = count + 1
  if (sessionQueryCount === 10) {
    // After 10 queries, check if we should show reminder
    if (shouldShowPeriodicReminder({ minQueriesInSession: 10 })) {
      // Dispatch event for UI to handle
      window.dispatchEvent(
        new CustomEvent('show-periodic-upgrade-reminder', {
          detail: { trigger: 'query-milestone' },
        })
      )
    }
  }
}

/**
 * Check connection milestone
 */
export function checkConnectionMilestone(connectionCount: number) {
  const tierStore = useTierStore.getState()
  const promptStore = useUpgradePromptStore.getState()

  // Don't show if already on paid tier
  if (tierStore.currentTier !== 'local') {
    return false
  }

  // Check if at or near connection limit
  const limitCheck = tierStore.checkLimit('connections', connectionCount)

  if (limitCheck.percentage >= 80 && promptStore.shouldShowPrompt('connections')) {
    // Dispatch event for UI to handle
    window.dispatchEvent(
      new CustomEvent('show-upgrade-reminder', {
        detail: { trigger: 'connections', usage: connectionCount, limit: limitCheck.limit },
      })
    )
    return true
  }

  return false
}

/**
 * Initialize reminder system
 * Call this on app startup
 */
export function initializeReminderSystem() {
  const promptStore = useUpgradePromptStore.getState()
  promptStore.initializeDevice()

  // Check app launch reminder
  if (checkAppLaunchReminder()) {
    // Wait a bit before showing (let app load first)
    setTimeout(() => {
      window.dispatchEvent(
        new CustomEvent('show-periodic-upgrade-reminder', {
          detail: { trigger: 'app-launch' },
        })
      )
    }, 3000) // 3 seconds after launch
  }

  // Check for new device
  if (promptStore.isNewDevice() && promptStore.shouldShowPrompt('multiDevice')) {
    setTimeout(() => {
      window.dispatchEvent(
        new CustomEvent('show-upgrade-reminder', {
          detail: { trigger: 'multiDevice' },
        })
      )
    }, 5000) // 5 seconds after launch
  }
}

/**
 * Hook for reminder system
 */
export function useUpgradeReminders() {
  const trackQuery = () => {
    trackQueryExecution()
  }

  const checkConnection = (count: number) => {
    return checkConnectionMilestone(count)
  }

  const shouldShowPeriodic = (config?: Partial<ReminderConfig>) => {
    return shouldShowPeriodicReminder(config)
  }

  return {
    trackQuery,
    checkConnection,
    shouldShowPeriodic,
  }
}

/**
 * Register global event listeners for reminders
 * Call this in your app initialization
 */
export function registerReminderListeners(onShowUpgrade: (trigger: UpgradeTrigger) => void) {
  // Listen for periodic reminder events
  window.addEventListener('show-periodic-upgrade-reminder', ((_event: CustomEvent) => {
    onShowUpgrade('periodic')
  }) as EventListener)

  // Listen for specific upgrade reminder events
  window.addEventListener('show-upgrade-reminder', ((event: CustomEvent) => {
    const { trigger } = event.detail
    onShowUpgrade(trigger)
  }) as EventListener)
}
