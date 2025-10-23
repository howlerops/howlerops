/**
 * Upgrade Prompt Store
 *
 * Tracks when to show upgrade prompts using soft nudges instead of hard blocks.
 * Manages cooldown periods and dismissals to avoid nagging users.
 *
 * Philosophy:
 * - Never block users from working
 * - Show prompts at natural moments
 * - Respect dismissals and cooldown periods
 * - Be contextual and helpful
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'

/**
 * Trigger types for upgrade prompts
 */
export type UpgradeTrigger =
  | 'connections' // Reached connection limit
  | 'queryHistory' // Approaching query history limit
  | 'multiDevice' // Detected new device
  | 'aiMemory' // AI wants to remember context
  | 'export' // Large export file
  | 'manual' // User clicked upgrade button
  | 'periodic' // Periodic reminder for active users
  | 'feature' // Locked feature clicked

/**
 * Cooldown configuration (in milliseconds)
 */
export const COOLDOWN_PERIODS: Record<UpgradeTrigger, number> = {
  connections: 24 * 60 * 60 * 1000, // 24 hours
  queryHistory: 7 * 24 * 60 * 60 * 1000, // 7 days
  multiDevice: 30 * 24 * 60 * 60 * 1000, // 30 days
  aiMemory: 7 * 24 * 60 * 60 * 1000, // 7 days
  export: 24 * 60 * 60 * 1000, // 24 hours
  manual: 0, // No cooldown for manual triggers
  periodic: 7 * 24 * 60 * 60 * 1000, // 7 days
  feature: 24 * 60 * 60 * 1000, // 24 hours
}

/**
 * Dismissal durations (in hours)
 */
export const DISMISSAL_DURATIONS = {
  short: 24, // 1 day
  medium: 24 * 7, // 1 week
  long: 24 * 30, // 1 month
  permanent: 24 * 365, // 1 year (effectively permanent)
}

/**
 * Upgrade prompt state
 */
interface UpgradePromptState {
  /** Last time prompt was shown for each trigger */
  lastShown: Record<string, string> // ISO date strings

  /** User dismissed prompts (timestamp when dismissal expires) */
  dismissed: Record<string, string> // ISO date strings

  /** Device fingerprint for multi-device detection */
  deviceFingerprint?: string

  /** First app launch timestamp */
  firstLaunch?: string // ISO date string

  /** Total queries executed (for activity tracking) */
  totalQueries: number

  /** Total prompts shown (for A/B testing) */
  totalPromptsShown: number

  /** Total prompts dismissed */
  totalPromptsDismissed: number

  /** Total upgrades (conversions) */
  totalUpgrades: number
}

/**
 * Upgrade prompt actions
 */
interface UpgradePromptActions {
  /**
   * Check if a prompt should be shown for a trigger
   * @param trigger - The trigger type
   * @returns True if the prompt should be shown
   */
  shouldShowPrompt: (trigger: UpgradeTrigger) => boolean

  /**
   * Mark a prompt as shown
   * @param trigger - The trigger type
   */
  markShown: (trigger: UpgradeTrigger) => void

  /**
   * Dismiss a prompt for a period
   * @param trigger - The trigger type
   * @param hours - Hours to dismiss for (default: 24)
   */
  dismiss: (trigger: UpgradeTrigger, hours?: number) => void

  /**
   * Clear dismissal for a trigger
   * @param trigger - The trigger type
   */
  clearDismissal: (trigger: UpgradeTrigger) => void

  /**
   * Clear all dismissals
   */
  clearAllDismissals: () => void

  /**
   * Initialize device fingerprint
   */
  initializeDevice: () => void

  /**
   * Check if current device is new
   * @returns True if device is new (different from stored fingerprint)
   */
  isNewDevice: () => boolean

  /**
   * Increment query count (for activity tracking)
   */
  incrementQueryCount: () => void

  /**
   * Record an upgrade (conversion)
   */
  recordUpgrade: () => void

  /**
   * Get conversion metrics
   */
  getMetrics: () => {
    totalShown: number
    totalDismissed: number
    totalUpgrades: number
    conversionRate: number
    dismissRate: number
  }

  /**
   * Reset all prompt history (for testing)
   */
  resetHistory: () => void
}

type UpgradePromptStore = UpgradePromptState & UpgradePromptActions

/**
 * Default initial state
 */
const DEFAULT_STATE: UpgradePromptState = {
  lastShown: {},
  dismissed: {},
  totalQueries: 0,
  totalPromptsShown: 0,
  totalPromptsDismissed: 0,
  totalUpgrades: 0,
}

/**
 * Generate a simple device fingerprint
 * Based on screen size, timezone, and random ID
 */
function generateDeviceFingerprint(): string {
  const screen = `${window.screen.width}x${window.screen.height}`
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const random = localStorage.getItem('device-random-id') || Math.random().toString(36).substring(7)

  if (!localStorage.getItem('device-random-id')) {
    localStorage.setItem('device-random-id', random)
  }

  return `${screen}-${timezone}-${random}`
}

/**
 * Upgrade Prompt Store
 *
 * Usage:
 * ```typescript
 * const { shouldShowPrompt, markShown, dismiss } = useUpgradePromptStore()
 *
 * if (shouldShowPrompt('connections')) {
 *   // Show upgrade modal
 *   markShown('connections')
 * }
 * ```
 */
export const useUpgradePromptStore = create<UpgradePromptStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...DEFAULT_STATE,

        shouldShowPrompt: (trigger: UpgradeTrigger): boolean => {
          const state = get()

          // Manual triggers always show
          if (trigger === 'manual') {
            return true
          }

          // Check if dismissed
          const dismissedUntil = state.dismissed[trigger]
          if (dismissedUntil) {
            const expiresAt = new Date(dismissedUntil)
            if (new Date() < expiresAt) {
              return false // Still dismissed
            }
          }

          // Check cooldown period
          const lastShownStr = state.lastShown[trigger]
          if (lastShownStr) {
            const lastShown = new Date(lastShownStr)
            const cooldown = COOLDOWN_PERIODS[trigger]
            const nextShowTime = new Date(lastShown.getTime() + cooldown)

            if (new Date() < nextShowTime) {
              return false // Still in cooldown
            }
          }

          return true
        },

        markShown: (trigger: UpgradeTrigger) => {
          const now = new Date().toISOString()

          set((state) => ({
            lastShown: {
              ...state.lastShown,
              [trigger]: now,
            },
            totalPromptsShown: state.totalPromptsShown + 1,
          }))
        },

        dismiss: (trigger: UpgradeTrigger, hours: number = DISMISSAL_DURATIONS.short) => {
          const dismissUntil = new Date()
          dismissUntil.setHours(dismissUntil.getHours() + hours)

          set((state) => ({
            dismissed: {
              ...state.dismissed,
              [trigger]: dismissUntil.toISOString(),
            },
            totalPromptsDismissed: state.totalPromptsDismissed + 1,
          }))
        },

        clearDismissal: (trigger: UpgradeTrigger) => {
          set((state) => {
            const { [trigger]: _, ...rest } = state.dismissed
            return { dismissed: rest }
          })
        },

        clearAllDismissals: () => {
          set({ dismissed: {} })
        },

        initializeDevice: () => {
          const state = get()

          // Set first launch if not already set
          if (!state.firstLaunch) {
            set({ firstLaunch: new Date().toISOString() })
          }

          // Generate device fingerprint if not exists
          if (!state.deviceFingerprint) {
            set({ deviceFingerprint: generateDeviceFingerprint() })
          }
        },

        isNewDevice: (): boolean => {
          const state = get()
          if (!state.deviceFingerprint) {
            return false // First time, not "new"
          }

          const currentFingerprint = generateDeviceFingerprint()
          return currentFingerprint !== state.deviceFingerprint
        },

        incrementQueryCount: () => {
          set((state) => ({
            totalQueries: state.totalQueries + 1,
          }))
        },

        recordUpgrade: () => {
          set((state) => ({
            totalUpgrades: state.totalUpgrades + 1,
          }))
        },

        getMetrics: () => {
          const state = get()
          return {
            totalShown: state.totalPromptsShown,
            totalDismissed: state.totalPromptsDismissed,
            totalUpgrades: state.totalUpgrades,
            conversionRate:
              state.totalPromptsShown > 0
                ? (state.totalUpgrades / state.totalPromptsShown) * 100
                : 0,
            dismissRate:
              state.totalPromptsShown > 0
                ? (state.totalPromptsDismissed / state.totalPromptsShown) * 100
                : 0,
          }
        },

        resetHistory: () => {
          set({
            ...DEFAULT_STATE,
            deviceFingerprint: get().deviceFingerprint,
            firstLaunch: get().firstLaunch,
          })
        },
      }),
      {
        name: 'sql-studio-upgrade-prompt-storage',
        version: 1,
      }
    ),
    {
      name: 'UpgradePromptStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Initialize the upgrade prompt store on app startup
 */
export const initializeUpgradePromptStore = () => {
  useUpgradePromptStore.getState().initializeDevice()
}

/**
 * Selectors for common checks
 */
export const upgradePromptSelectors = {
  canShowAnyPrompt: (state: UpgradePromptStore) => {
    const triggers: UpgradeTrigger[] = [
      'connections',
      'queryHistory',
      'multiDevice',
      'aiMemory',
      'export',
      'periodic',
      'feature',
    ]
    return triggers.some((trigger) => state.shouldShowPrompt(trigger))
  },

  hasBeenDismissed: (state: UpgradePromptStore, trigger: UpgradeTrigger) => {
    const dismissedUntil = state.dismissed[trigger]
    if (!dismissedUntil) return false

    const expiresAt = new Date(dismissedUntil)
    return new Date() < expiresAt
  },

  getDaysUntilNextShow: (state: UpgradePromptStore, trigger: UpgradeTrigger): number | null => {
    const lastShownStr = state.lastShown[trigger]
    if (!lastShownStr) return null

    const lastShown = new Date(lastShownStr)
    const cooldown = COOLDOWN_PERIODS[trigger]
    const nextShowTime = new Date(lastShown.getTime() + cooldown)
    const now = new Date()

    if (now >= nextShowTime) return 0

    const daysUntil = Math.ceil((nextShowTime.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
    return daysUntil
  },
}
