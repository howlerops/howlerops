/**
 * Auth System Integration Tests
 *
 * Tests the complete authentication flow including:
 * - User signup
 * - User login
 * - Token refresh
 * - Logout
 * - Tier integration
 */

const { fetchMock, storage } = vi.hoisted(() => {
  const store = new Map<string, string>()
  const storageMock: Storage = {
    get length() {
      return store.size
    },
    clear: () => {
      store.clear()
    },
    getItem: (key: string) => store.get(key) ?? null,
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    removeItem: (key: string) => {
      store.delete(key)
    },
    setItem: (key: string, value: string) => {
      store.set(key, value)
    },
  }

  const mockFetch = vi.fn()
  Object.defineProperty(globalThis, 'localStorage', { value: storageMock, configurable: true })
  Object.defineProperty(globalThis, 'sessionStorage', { value: storageMock, configurable: true })
  Object.defineProperty(globalThis, 'fetch', { value: mockFetch, configurable: true })

  return { fetchMock: mockFetch, storage: storageMock }
})

import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/lib/wails-runtime', () => ({
  isWailsReady: () => false,
}))

vi.mock('@/lib/wails-guard', () => ({
  callWails: (fn: () => Promise<unknown>) => fn(),
  subscribeToWailsEvent: vi.fn(() => () => {}),
}))

import { useAuthStore } from '@/store/auth-store'

describe('Auth System Integration', () => {
  beforeEach(() => {
    // Clear store
    storage.clear()
    useAuthStore.getState().reset()
    fetchMock.mockReset()
  })

  describe('Sign Up Flow', () => {
    it('should successfully sign up a new user', async () => {
      const mockResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'access-token-123',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signUp('testuser', 'test@example.com', 'Password123')
      })

      expect(result.current.isAuthenticated).toBe(true)
      expect(result.current.user?.username).toBe('testuser')
      expect(result.current.tokens?.access_token).toBe('access-token-123')
      expect(result.current.error).toBeNull()
    })

    it('should handle signup errors', async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ message: 'Username already exists' }),
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        try {
          await result.current.signUp('testuser', 'test@example.com', 'Password123')
        } catch (error) {
          // Expected error
        }
      })

      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.error).toBe('Username already exists')
    })
  })

  describe('Sign In Flow', () => {
    it('should successfully sign in an existing user', async () => {
      const mockResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'access-token-123',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      expect(result.current.isAuthenticated).toBe(true)
      expect(result.current.user?.username).toBe('testuser')
      expect(result.current.tokens?.access_token).toBe('access-token-123')
    })

    it('should handle invalid credentials', async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ message: 'Invalid credentials' }),
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        try {
          await result.current.signIn('testuser', 'wrongpassword')
        } catch (error) {
          // Expected error
        }
      })

      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.error).toBe('Invalid credentials')
    })
  })

  describe('Token Refresh', () => {
    it('should successfully refresh access token', async () => {
      // First, sign in
      const signInResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'old-access-token',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => signInResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      // Now refresh the token
      const refreshResponse = {
        token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => refreshResponse,
      })

      let refreshSuccess = false
      await act(async () => {
        refreshSuccess = await result.current.refreshToken()
      })

      expect(refreshSuccess).toBe(true)
      expect(result.current.tokens?.access_token).toBe('new-access-token')
      expect(result.current.isAuthenticated).toBe(true)
    })

    it('should sign out on refresh failure', async () => {
      // First, sign in
      const signInResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'access-token',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => signInResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      // Refresh fails
      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ message: 'Invalid refresh token' }),
      })

      // Mock logout endpoint
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      })

      let refreshSuccess = false
      await act(async () => {
        refreshSuccess = await result.current.refreshToken()
      })

      expect(refreshSuccess).toBe(false)
      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.user).toBeNull()
    })
  })

  describe('Sign Out', () => {
    it('should successfully sign out', async () => {
      // First, sign in
      const signInResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'access-token',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => signInResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      expect(result.current.isAuthenticated).toBe(true)

      // Mock logout endpoint
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      })

      // Now sign out
      await act(async () => {
        await result.current.signOut()
      })

      expect(result.current.isAuthenticated).toBe(false)
      expect(result.current.user).toBeNull()
      expect(result.current.tokens).toBeNull()
    })
  })

  describe('Persistence', () => {
    it('should persist authentication state', async () => {
      const mockResponse = {
        user: {
          id: 'user-123',
          username: 'testuser',
          email: 'test@example.com',
          role: 'user',
          created_at: new Date().toISOString(),
        },
        token: 'access-token-123',
        refresh_token: 'refresh-token-123',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      }

      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      expect(result.current.isAuthenticated).toBe(true)

      // Verify localStorage has the data
      const stored = localStorage.getItem('auth-storage')
      expect(stored).toBeTruthy()

      const parsed = JSON.parse(stored!)
      expect(parsed.state.user.username).toBe('testuser')
      expect(parsed.state.isAuthenticated).toBe(true)
    })
  })
})
