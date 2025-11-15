/**
 * Authentication Store
 *
 * Central state management for user authentication using Zustand.
 * Handles JWT-based authentication with the Go backend.
 *
 * Features:
 * - Sign up, sign in, sign out
 * - Automatic token refresh
 * - Persistent authentication state
 * - Integration with tier store
 * - Master key management for password encryption
 * - Error handling and loading states
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { importMasterKeyFromBase64 } from '@/lib/crypto/encryption'

export interface User {
  id: string
  username: string
  email: string
  role: string
  created_at: string
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  expires_at: string
}

// In-memory master key cache (never persisted to disk!)
let sessionMasterKey: CryptoKey | null = null

interface AuthState {
  // State
  user: User | null
  tokens: AuthTokens | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null

  // Actions
  signUp: (username: string, email: string, password: string) => Promise<void>
  signIn: (username: string, password: string) => Promise<void>
  signOut: () => Promise<void>
  refreshToken: () => Promise<boolean>
  clearError: () => void

  // Master key management
  getMasterKey: () => CryptoKey | null
  hasMasterKey: () => boolean

  // Internal
  setUser: (user: User | null) => void
  setTokens: (tokens: AuthTokens | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  reset: () => void
}

// API base URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      tokens: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      // Sign up
      signUp: async (username: string, email: string, password: string) => {
        set({ isLoading: true, error: null })

        try {
          const response = await fetch(`${API_BASE_URL}/api/auth/signup`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password }),
          })

          if (!response.ok) {
            const error = await response.json()
            throw new Error(error.message || 'Signup failed')
          }

          const data = await response.json()

          // Import and cache master key in memory
          if (data.master_key) {
            try {
              sessionMasterKey = await importMasterKeyFromBase64(data.master_key)
            } catch (err) {
              console.error('Failed to import master key:', err)
            }
          }

          set({
            user: data.user,
            tokens: {
              access_token: data.token,
              refresh_token: data.refresh_token,
              expires_at: data.expires_at,
            },
            isAuthenticated: true,
            isLoading: false,
          })

          // Start token refresh timer
          startTokenRefreshTimer(get().refreshToken)

          // Update tier store based on user data
          try {
            const { useTierStore } = await import('./tier-store')
            // Check if user has any license/tier info
            if (data.user.license_key) {
              await useTierStore.getState().activateLicense(data.user.license_key)
            }
          } catch (err) {
            console.error('Failed to update tier store:', err)
          }
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Signup failed',
            isLoading: false,
          })
          throw error
        }
      },

      // Sign in
      signIn: async (username: string, password: string) => {
        set({ isLoading: true, error: null })

        try {
          const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
          })

          if (!response.ok) {
            const error = await response.json()
            throw new Error(error.message || 'Login failed')
          }

          const data = await response.json()

          // Import and cache master key in memory
          if (data.master_key) {
            try {
              sessionMasterKey = await importMasterKeyFromBase64(data.master_key)
            } catch (err) {
              console.error('Failed to import master key:', err)
            }
          }

          set({
            user: data.user,
            tokens: {
              access_token: data.token,
              refresh_token: data.refresh_token,
              expires_at: data.expires_at,
            },
            isAuthenticated: true,
            isLoading: false,
          })

          // Start token refresh timer
          startTokenRefreshTimer(get().refreshToken)

          // Update tier store based on user data
          try {
            const { useTierStore } = await import('./tier-store')
            // Check if user has any license/tier info
            if (data.user.license_key) {
              await useTierStore.getState().activateLicense(data.user.license_key)
            }
          } catch (err) {
            console.error('Failed to update tier store:', err)
          }
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Login failed',
            isLoading: false,
          })
          throw error
        }
      },

      // Sign out
      signOut: async () => {
        const { tokens } = get()

        if (tokens) {
          try {
            await fetch(`${API_BASE_URL}/api/auth/logout`, {
              method: 'POST',
              headers: {
                Authorization: `Bearer ${tokens.access_token}`,
                'Content-Type': 'application/json',
              },
            })
          } catch (error) {
            console.error('Logout request failed:', error)
            // Continue with local logout even if backend fails
          }
        }

        // CRITICAL: Clear master key from memory
        sessionMasterKey = null

        set({
          user: null,
          tokens: null,
          isAuthenticated: false,
          error: null,
        })

        // Stop token refresh timer
        stopTokenRefreshTimer()

        // Reset tier store
        try {
          const { useTierStore } = await import('./tier-store')
          useTierStore.getState().setTier('local')
        } catch (err) {
          console.error('Failed to reset tier store:', err)
        }
      },

      // Refresh access token
      refreshToken: async () => {
        const { tokens } = get()

        if (!tokens?.refresh_token) {
          return false
        }

        try {
          const response = await fetch(`${API_BASE_URL}/api/auth/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: tokens.refresh_token }),
          })

          if (!response.ok) {
            throw new Error('Token refresh failed')
          }

          const data = await response.json()

          set({
            tokens: {
              access_token: data.token,
              refresh_token: data.refresh_token,
              expires_at: data.expires_at,
            },
          })

          return true
        } catch (error) {
          console.error('Token refresh failed:', error)
          // Sign out on refresh failure
          get().signOut()
          return false
        }
      },

      // Clear error
      clearError: () => set({ error: null }),

      // Master key management
      getMasterKey: () => sessionMasterKey,
      hasMasterKey: () => sessionMasterKey !== null,

      // Setters
      setUser: (user) => set({ user }),
      setTokens: (tokens) => set({ tokens }),
      setLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),

      // Reset store
      reset: () => {
        // Clear master key from memory
        sessionMasterKey = null

        set({
          user: null,
          tokens: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
        })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        tokens: state.tokens,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)

// Token refresh timer
let refreshTimerId: NodeJS.Timeout | null = null

function startTokenRefreshTimer(refreshFn: () => Promise<boolean>) {
  stopTokenRefreshTimer()

  // Refresh token 5 minutes before expiration
  const REFRESH_BUFFER_MS = 5 * 60 * 1000

  const scheduleRefresh = () => {
    const { tokens } = useAuthStore.getState()
    if (!tokens) return

    const expiresAt = new Date(tokens.expires_at).getTime()
    const now = Date.now()
    const timeUntilRefresh = expiresAt - now - REFRESH_BUFFER_MS

    if (timeUntilRefresh > 0) {
      refreshTimerId = setTimeout(async () => {
        const success = await refreshFn()
        if (success) {
          scheduleRefresh() // Schedule next refresh
        }
      }, timeUntilRefresh)
    } else {
      // Token already expired or about to expire, refresh immediately
      refreshFn().then((success) => {
        if (success) {
          scheduleRefresh()
        }
      })
    }
  }

  scheduleRefresh()
}

function stopTokenRefreshTimer() {
  if (refreshTimerId) {
    clearTimeout(refreshTimerId)
    refreshTimerId = null
  }
}

// Initialize token refresh on app load
if (useAuthStore.getState().isAuthenticated) {
  startTokenRefreshTimer(useAuthStore.getState().refreshToken)
}

/**
 * Helper to get authorization header
 */
export const getAuthHeader = (): Record<string, string> => {
  const { tokens } = useAuthStore.getState()

  if (!tokens?.access_token) {
    return {}
  }

  return {
    Authorization: `Bearer ${tokens.access_token}`,
  }
}

/**
 * Initialize auth store on app startup
 */
export const initializeAuthStore = () => {
  const { isAuthenticated, refreshToken } = useAuthStore.getState()

  if (isAuthenticated) {
    // Validate token on startup
    refreshToken()
  }
}
