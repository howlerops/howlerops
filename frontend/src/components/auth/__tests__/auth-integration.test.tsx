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

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAuthStore } from '@/store/auth-store'

// Mock fetch
global.fetch = vi.fn()

describe('Auth System Integration', () => {
  beforeEach(() => {
    // Clear store
    useAuthStore.getState().reset()
    vi.clearAllMocks()
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

      ;(global.fetch as any).mockResolvedValueOnce({
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
      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
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
      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => signInResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      // Refresh fails
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        json: async () => ({ message: 'Invalid refresh token' }),
      })

      // Mock logout endpoint
      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => signInResponse,
      })

      const { result } = renderHook(() => useAuthStore())

      await act(async () => {
        await result.current.signIn('testuser', 'Password123')
      })

      expect(result.current.isAuthenticated).toBe(true)

      // Mock logout endpoint
      ;(global.fetch as any).mockResolvedValueOnce({
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

      ;(global.fetch as any).mockResolvedValueOnce({
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
