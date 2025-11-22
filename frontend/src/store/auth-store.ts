/**
 * Authentication Store
 *
 * Central state management for user authentication using Zustand.
 * Handles JWT-based authentication with the Go backend.
 *
 * Features:
 * - Sign up, sign in, sign out
 * - OAuth authentication (Google, GitHub)
 * - Biometric authentication (WebAuthn)
 * - Automatic token refresh
 * - Persistent authentication state
 * - Integration with tier store
 * - Master key management for password encryption
 * - Error handling and loading states
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import * as authApi from '@/lib/auth-api'
import { importMasterKeyFromBase64 } from '@/lib/crypto/encryption'
import { isWailsApp } from '@/lib/platform'
import { subscribeToWailsEvent } from '@/lib/wails-guard'
import type { AuthRestoredEvent,AuthSuccessEvent } from '@/types/wails-auth'

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
  signInWithOAuth: (provider: 'google' | 'github') => Promise<void>
  signInWithBiometric: () => Promise<void>
  signOut: () => Promise<void>
  refreshToken: () => Promise<boolean>
  clearError: () => void
  checkStoredAuth: () => Promise<void>

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

      // Sign in with OAuth
      signInWithOAuth: async (provider: 'google' | 'github') => {
        set({ isLoading: true, error: null })

        try {
          // Get OAuth URL from backend (works in both Wails and web mode)
          const { authUrl, state } = await authApi.getOAuthURL(provider)

          if (isWailsApp()) {
            // Desktop mode: Open in new window
            // Callback handled by OS custom protocol handler → auth:success event
            window.open(authUrl, '_blank')
          } else {
            // Web mode: Redirect current window
            // Store state in sessionStorage for verification after redirect
            if (state) {
              sessionStorage.setItem('oauth_state', state)
            }
            window.location.href = authUrl
          }

          // Note: Authentication completion handled differently per mode:
          // - Desktop: auth:success event listener
          // - Web: /auth/callback route processes the redirect
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'OAuth login failed',
            isLoading: false,
          })
          throw error
        }
      },

      // Sign in with biometric authentication
      signInWithBiometric: async () => {
        set({ isLoading: true, error: null })

        try {
          const { parsePublicKeyRequestOptions, serializeCredentialAssertion } = await import(
            '@/lib/utils/webauthn'
          )

          // Get WebAuthn challenge from backend (works in both modes)
          const optionsJSON = await authApi.startWebAuthnAuthentication()
          const options = parsePublicKeyRequestOptions(optionsJSON)

          // Trigger browser's native biometric prompt
          const credential = await navigator.credentials.get({
            publicKey: options,
          })

          if (!credential || credential.type !== 'public-key') {
            throw new Error('Invalid credential type')
          }

          // Send credential to backend for verification (works in both modes)
          const assertionJSON = serializeCredentialAssertion(credential as PublicKeyCredential)
          const token = await authApi.finishWebAuthnAuthentication(assertionJSON)

          if (!token) {
            throw new Error('Authentication failed')
          }

          // Set authenticated state
          set({
            tokens: {
              access_token: token,
              refresh_token: '', // WebAuthn doesn't provide refresh token
              expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
            },
            isAuthenticated: true,
            isLoading: false,
          })
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : 'Biometric authentication failed',
            isLoading: false,
          })
          throw error
        }
      },

      // Check for stored authentication on app startup
      checkStoredAuth: async () => {
        // Only applicable in desktop mode - web mode uses session cookies
        if (!isWailsApp()) {
          return
        }

        try {
          // Check for OAuth tokens in backend keychain (desktop only)
          const providers: Array<'google' | 'github'> = ['google', 'github']

          for (const provider of providers) {
            const hasToken = await authApi.checkStoredToken(provider)
            if (hasToken) {
              console.log(`Found stored ${provider} token`)
              // Token will be restored via auth:restored event
              break
            }
          }
        } catch (error) {
          console.error('Failed to check stored auth:', error)
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
  const { isAuthenticated, refreshToken, checkStoredAuth } = useAuthStore.getState()

  // Set up event listeners for OAuth authentication (desktop mode only)
  if (isWailsApp() && typeof window !== 'undefined' && window.runtime) {
    try {
      // Listen for successful OAuth authentication
      subscribeToWailsEvent('auth:success', (data: AuthSuccessEvent) => {
        console.log('OAuth authentication successful:', data)

        useAuthStore.setState({
          user: {
            id: data.user.id,
            username: data.user.name || data.user.email,
            email: data.user.email,
            role: 'user',
            created_at: new Date().toISOString(),
          },
          tokens: {
            access_token: data.token,
            refresh_token: '', // OAuth tokens stored in backend keyring
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
          },
          isAuthenticated: true,
          isLoading: false,
          error: null,
        })
      })

      // Listen for OAuth authentication errors
      subscribeToWailsEvent('auth:error', (message: string) => {
        console.error('OAuth authentication error:', message)

        useAuthStore.setState({
          error: message,
          isLoading: false,
        })
      })

      // Listen for restored authentication on app startup
      subscribeToWailsEvent('auth:restored', (data: AuthRestoredEvent) => {
        console.log('Authentication restored from keyring')

        useAuthStore.setState({
          tokens: {
            access_token: data.token,
            refresh_token: '',
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
          },
          isAuthenticated: true,
          isLoading: false,
        })
      })
    } catch (err) {
      console.error('Failed to subscribe to Wails auth events:', err)
    }
  }

  if (isAuthenticated) {
    // Validate token on startup
    refreshToken()
  } else {
    // Check for stored OAuth tokens (desktop only)
    checkStoredAuth()
  }
}
