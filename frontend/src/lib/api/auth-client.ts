/**
 * Authentication API Client
 *
 * Centralized HTTP client for authenticated API requests.
 * Automatically includes JWT tokens and handles token refresh.
 */

import { useAuthStore } from '@/store/auth-store'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export interface ApiError {
  message: string
  status: number
  code?: string
}

export class AuthApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string
  ) {
    super(message)
    this.name = 'AuthApiError'
  }
}

interface FetchOptions extends RequestInit {
  skipAuth?: boolean
}

/**
 * Authenticated fetch wrapper
 * Automatically includes JWT token and handles refresh
 */
export async function authFetch<T = unknown>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { skipAuth = false, ...fetchOptions } = options

  // Build headers
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string>),
  }

  // Add auth token if not skipped
  if (!skipAuth) {
    const { tokens } = useAuthStore.getState()
    if (tokens?.access_token) {
      headers['Authorization'] = `Bearer ${tokens.access_token}`
    }
  }

  // Make request
  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`
  let response = await fetch(url, {
    ...fetchOptions,
    headers,
  })

  // Handle 401 - try to refresh token
  if (response.status === 401 && !skipAuth) {
    const refreshSuccess = await useAuthStore.getState().refreshToken()

    if (refreshSuccess) {
      // Retry with new token
      const { tokens } = useAuthStore.getState()
      if (tokens?.access_token) {
        headers['Authorization'] = `Bearer ${tokens.access_token}`
      }

      response = await fetch(url, {
        ...fetchOptions,
        headers,
      })
    } else {
      // Refresh failed, user will be logged out by auth store
      throw new AuthApiError('Authentication required', 401, 'AUTH_REQUIRED')
    }
  }

  // Handle errors
  if (!response.ok) {
    let errorMessage = 'Request failed'
    let errorCode: string | undefined

    try {
      const errorData = await response.json()
      errorMessage = errorData.message || errorData.error || errorMessage
      errorCode = errorData.code
    } catch {
      // Failed to parse error response
      errorMessage = response.statusText || errorMessage
    }

    throw new AuthApiError(errorMessage, response.status, errorCode)
  }

  // Parse response
  const contentType = response.headers.get('content-type')
  if (contentType?.includes('application/json')) {
    return (await response.json()) as T
  }

  return (await response.text()) as T
}

/**
 * Auth API endpoints
 */
export const authApi = {
  /**
   * Sign up a new user
   */
  signup: async (username: string, email: string, password: string) => {
    return authFetch<{
      user: {
        id: string
        username: string
        email: string
        role: string
        created_at: string
      }
      token: string
      refresh_token: string
      expires_at: string
    }>('/api/auth/signup', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
      skipAuth: true,
    })
  },

  /**
   * Sign in existing user
   */
  login: async (username: string, password: string) => {
    return authFetch<{
      user: {
        id: string
        username: string
        email: string
        role: string
        created_at: string
      }
      token: string
      refresh_token: string
      expires_at: string
    }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
      skipAuth: true,
    })
  },

  /**
   * Sign out current user
   */
  logout: async () => {
    return authFetch<void>('/api/auth/logout', {
      method: 'POST',
    })
  },

  /**
   * Refresh access token
   */
  refresh: async (refreshToken: string) => {
    return authFetch<{
      token: string
      refresh_token: string
      expires_at: string
    }>('/api/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
      skipAuth: true,
    })
  },

  /**
   * Get current user profile
   */
  getProfile: async () => {
    return authFetch<{
      id: string
      username: string
      email: string
      role: string
      created_at: string
    }>('/api/auth/me')
  },
}
