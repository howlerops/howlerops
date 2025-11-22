/**
 * Tier Management Store
 *
 * Central state management for Howlerops's tier system using Zustand.
 * Handles tier detection, license validation, feature gates, and usage limits.
 *
 * Features:
 * - Persistent storage of tier information in localStorage
 * - License key activation and validation
 * - Feature availability checks
 * - Usage limit enforcement
 * - Team information management
 * - Development mode override support
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'

import {
  getRequiredTier,
  isTierAtLeast,
  TIER_FEATURES,
  TIER_LIMITS,
  TIER_METADATA,
  tierHasFeature,
} from '@/config/tier-limits'
import { validateLicenseKey } from '@/lib/tiers/license-validator'
import type {
  LimitCheckResult,
  TeamRole,
  TierFeatures,
  TierLevel,
  TierLimits,
  TierPersistence,
} from '@/types/tiers'

/**
 * Tier store state interface
 */
interface TierState {
  /** Current active tier */
  currentTier: TierLevel
  /** Active license key (if any) */
  licenseKey?: string
  /** License expiration date */
  expiresAt?: Date
  /** Last validation timestamp */
  lastValidated?: Date
  /** Team ID (for team tier) */
  teamId?: string
  /** Team display name */
  teamName?: string
  /** Current user's team role */
  teamRole?: TeamRole
  /** Whether the store has been initialized */
  isInitialized: boolean
  /** Development mode override (ignores all limits/features) */
  devMode: boolean
}

/**
 * Tier store actions interface
 */
interface TierActions {
  /**
   * Set the current tier
   * @param tier - The tier to set
   */
  setTier: (tier: TierLevel) => void

  /**
   * Activate a license key
   * @param key - The license key string (format: SQL-{TIER}-{UUID}-{CHECKSUM})
   * @returns Validation result with success status and error message
   */
  activateLicense: (key: string) => Promise<{
    success: boolean
    error?: string
    tier?: TierLevel
  }>

  /**
   * Deactivate the current license and revert to local tier
   */
  deactivateLicense: () => void

  /**
   * Set team information (for team tier)
   * @param teamId - Team unique identifier
   * @param teamName - Team display name
   * @param teamRole - Current user's role in the team
   */
  setTeamInfo: (teamId: string, teamName: string, teamRole: TeamRole) => void

  /**
   * Clear team information
   */
  clearTeamInfo: () => void

  /**
   * Check if a feature is available in the current tier
   * @param feature - The feature name to check
   * @returns True if the feature is available or dev mode is enabled
   */
  hasFeature: (feature: keyof TierFeatures) => boolean

  /**
   * Check if usage is within limits for the current tier
   * @param limitName - The limit to check
   * @param currentUsage - Current usage value
   * @returns Detailed limit check result
   */
  checkLimit: (limitName: keyof TierLimits, currentUsage: number) => LimitCheckResult

  /**
   * Get the features available in the current tier
   * @returns Feature configuration object
   */
  getFeatures: () => TierFeatures

  /**
   * Get the limits for the current tier
   * @returns Limits configuration object
   */
  getLimits: () => TierLimits

  /**
   * Get metadata for the current tier
   * @returns Tier metadata (name, price, color, etc.)
   */
  getMetadata: () => (typeof TIER_METADATA)[TierLevel]

  /**
   * Check if the license has expired
   * @returns True if expired or invalid
   */
  isLicenseExpired: () => boolean

  /**
   * Get the minimum tier required for a feature
   * @param feature - The feature name
   * @returns Required tier level or null if feature doesn't exist
   */
  getRequiredTierForFeature: (feature: string) => TierLevel | null

  /**
   * Check if current tier is at least the specified tier
   * @param tier - The tier to compare against
   * @returns True if current tier >= specified tier
   */
  isAtLeastTier: (tier: TierLevel) => boolean

  /**
   * Validate and refresh license status
   * @returns True if license is still valid
   */
  validateLicense: () => Promise<boolean>

  /**
   * Initialize the tier store (called on app startup)
   */
  initialize: () => void

  /**
   * Enable development mode (bypasses all limits and feature gates)
   * Only available in development environment
   */
  enableDevMode: () => void

  /**
   * Disable development mode
   */
  disableDevMode: () => void
}

type TierStore = TierState & TierActions

/**
 * Default initial state
 */
const DEFAULT_STATE: TierState = {
  currentTier: 'local',
  isInitialized: false,
  devMode: false,
}

/**
 * Tier Management Store
 *
 * Usage:
 * ```typescript
 * const { currentTier, hasFeature, checkLimit } = useTierStore()
 *
 * if (hasFeature('sync')) {
 *   // Enable sync functionality
 * }
 *
 * const { allowed, remaining } = checkLimit('connections', currentConnections)
 * if (!allowed) {
 *   // Show upgrade prompt
 * }
 * ```
 */
export const useTierStore = create<TierStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...DEFAULT_STATE,

        setTier: (tier: TierLevel) => {
          set({ currentTier: tier }, false, 'setTier')
        },

        activateLicense: async (key: string) => {
          try {
            const validation = await validateLicenseKey(key)

            if (!validation.valid) {
              return {
                success: false,
                error: validation.message || 'Invalid license key',
              }
            }

            if (!validation.tier) {
              return {
                success: false,
                error: 'License key does not specify a tier',
              }
            }

            set(
              {
                currentTier: validation.tier,
                licenseKey: key,
                expiresAt: validation.expiresAt,
                lastValidated: new Date(),
              },
              false,
              'activateLicense'
            )

            return {
              success: true,
              tier: validation.tier,
            }
          } catch (error) {
            return {
              success: false,
              error: error instanceof Error ? error.message : 'Unknown error',
            }
          }
        },

        deactivateLicense: () => {
          set(
            {
              currentTier: 'local',
              licenseKey: undefined,
              expiresAt: undefined,
              lastValidated: undefined,
              teamId: undefined,
              teamName: undefined,
              teamRole: undefined,
            },
            false,
            'deactivateLicense'
          )
        },

        setTeamInfo: (teamId: string, teamName: string, teamRole: TeamRole) => {
          set({ teamId, teamName, teamRole }, false, 'setTeamInfo')
        },

        clearTeamInfo: () => {
          set(
            {
              teamId: undefined,
              teamName: undefined,
              teamRole: undefined,
            },
            false,
            'clearTeamInfo'
          )
        },

        hasFeature: (feature: keyof TierFeatures) => {
          const state = get()

          // Development mode bypasses all feature gates
          if (state.devMode) {
            return true
          }

          // Check if license is expired
          if (state.licenseKey && get().isLicenseExpired()) {
            return TIER_FEATURES.local[feature] === true
          }

          return tierHasFeature(state.currentTier, feature)
        },

        checkLimit: (limitName: keyof TierLimits, currentUsage: number): LimitCheckResult => {
          const state = get()

          // Development mode bypasses all limits
          if (state.devMode) {
            return {
              allowed: true,
              remaining: null,
              limit: null,
              percentage: 0,
              isNearLimit: false,
              isAtLimit: false,
              isUnlimited: true,
            }
          }

          // If license is expired, use local tier limits
          const effectiveTier =
            state.licenseKey && get().isLicenseExpired() ? 'local' : state.currentTier

          const limits = TIER_LIMITS[effectiveTier]
          const limit = limits[limitName]

          // Ensure limit exists in the tier configuration
          if (limit === undefined) {
            throw new Error(`Unknown limit: ${limitName}`)
          }

          // null means unlimited
          if (limit === null) {
            return {
              allowed: true,
              remaining: null,
              limit: null,
              percentage: 0,
              isNearLimit: false,
              isAtLimit: false,
              isUnlimited: true,
            }
          }

          const remaining = Math.max(0, limit - currentUsage)
          const percentage = limit > 0 ? (currentUsage / limit) * 100 : 0
          const isNearLimit = percentage >= 80 && percentage < 100
          const isAtLimit = currentUsage >= limit

          return {
            allowed: currentUsage < limit,
            remaining,
            limit,
            percentage: Math.min(100, percentage),
            isNearLimit,
            isAtLimit,
            isUnlimited: false,
          }
        },

        getFeatures: () => {
          const state = get()
          const effectiveTier =
            state.licenseKey && get().isLicenseExpired() ? 'local' : state.currentTier
          return TIER_FEATURES[effectiveTier]
        },

        getLimits: () => {
          const state = get()
          const effectiveTier =
            state.licenseKey && get().isLicenseExpired() ? 'local' : state.currentTier
          return TIER_LIMITS[effectiveTier]
        },

        getMetadata: () => {
          const state = get()
          return TIER_METADATA[state.currentTier]
        },

        isLicenseExpired: () => {
          const state = get()

          if (!state.expiresAt) {
            return false
          }

          return new Date() > new Date(state.expiresAt)
        },

        getRequiredTierForFeature: (feature: string) => {
          return getRequiredTier(feature)
        },

        isAtLeastTier: (tier: TierLevel) => {
          const state = get()
          const effectiveTier =
            state.licenseKey && get().isLicenseExpired() ? 'local' : state.currentTier
          return isTierAtLeast(effectiveTier, tier)
        },

        validateLicense: async () => {
          const state = get()

          if (!state.licenseKey) {
            return true // No license to validate
          }

          try {
            const validation = await validateLicenseKey(state.licenseKey)

            if (!validation.valid) {
              // License is no longer valid, revert to local tier
              get().deactivateLicense()
              return false
            }

            // Update expiration date if it changed
            if (validation.expiresAt) {
              set(
                {
                  expiresAt: validation.expiresAt,
                  lastValidated: new Date(),
                },
                false,
                'validateLicense'
              )
            }

            return true
          } catch (error) {
            console.error('License validation error:', error)
            return false
          }
        },

        initialize: () => {
          const state = get()

          if (state.isInitialized) {
            return
          }

          // Check for development mode override
          if (import.meta.env.DEV && import.meta.env.VITE_TIER_DEV_MODE === 'true') {
            set({ devMode: true }, false, 'initialize')
          }

          // Validate license if one exists
          if (state.licenseKey) {
            get().validateLicense()
          }

          set({ isInitialized: true }, false, 'initialize')
        },

        enableDevMode: () => {
          if (import.meta.env.DEV) {
            set({ devMode: true }, false, 'enableDevMode')
            console.log('🔓 Development mode enabled: All features unlocked')
          } else {
            console.warn('Development mode can only be enabled in development environment')
          }
        },

        disableDevMode: () => {
          set({ devMode: false }, false, 'disableDevMode')
          console.log('🔒 Development mode disabled')
        },
      }),
      {
        name: 'sql-studio-tier-storage',
        version: 1,
        // Custom storage to handle Date serialization
        storage: {
          getItem: (name) => {
            const str = localStorage.getItem(name)
            if (!str) return null

            try {
              const { state } = JSON.parse(str)

              // Convert date strings back to Date objects
              if (state.expiresAt) {
                state.expiresAt = new Date(state.expiresAt)
              }
              if (state.lastValidated) {
                state.lastValidated = new Date(state.lastValidated)
              }

              return { state }
            } catch (error) {
              console.error('Failed to parse tier storage:', error)
              return null
            }
          },
          setItem: (name, value) => {
            localStorage.setItem(name, JSON.stringify(value))
          },
          removeItem: (name) => {
            localStorage.removeItem(name)
          },
        },
        // Only persist specific fields
        partialize: (state): TierPersistence => ({
          currentTier: state.currentTier,
          licenseKey: state.licenseKey,
          expiresAt: state.expiresAt?.toISOString(),
          lastValidated: state.lastValidated?.toISOString(),
          teamId: state.teamId,
          teamName: state.teamName,
          teamRole: state.teamRole,
        }),
      }
    ),
    {
      name: 'TierStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Initialize the tier store on app startup
 * Call this in your main App component
 */
export const initializeTierStore = () => {
  useTierStore.getState().initialize()
}

/**
 * Selectors for common tier checks
 */
export const tierSelectors = {
  isLocal: (state: TierStore) => state.currentTier === 'local',
  isIndividual: (state: TierStore) => state.currentTier === 'individual',
  isTeam: (state: TierStore) => state.currentTier === 'team',
  isPaid: (state: TierStore) => state.currentTier !== 'local',
  hasActiveTeam: (state: TierStore) => state.currentTier === 'team' && !!state.teamId,
}
