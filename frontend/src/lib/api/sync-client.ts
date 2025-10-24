/**
 * Sync HTTP Client
 *
 * Type-safe HTTP client for cloud sync API endpoints.
 * Handles authentication, retries, and error handling.
 *
 * @module lib/api/sync-client
 */

import type {
  UploadChangesRequest,
  UploadChangesResponse,
  DownloadChangesResponse,
  Conflict,
  ConflictResolution,
} from '@/types/sync'
import { useTierStore } from '@/store/tier-store'

/**
 * Sync API client errors
 */
export class SyncClientError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly statusCode?: number,
    public readonly cause?: Error
  ) {
    super(message)
    this.name = 'SyncClientError'
  }
}

/**
 * Authentication error (401/403)
 */
export class AuthenticationError extends SyncClientError {
  constructor(message = 'Authentication required', cause?: Error) {
    super(message, 'AUTH_REQUIRED', 401, cause)
    this.name = 'AuthenticationError'
  }
}

/**
 * Network error (connection failed)
 */
export class NetworkError extends SyncClientError {
  constructor(message = 'Network request failed', cause?: Error) {
    super(message, 'NETWORK_ERROR', undefined, cause)
    this.name = 'NetworkError'
  }
}

/**
 * Server error (5xx)
 */
export class ServerError extends SyncClientError {
  constructor(message = 'Server error', statusCode: number, cause?: Error) {
    super(message, 'SERVER_ERROR', statusCode, cause)
    this.name = 'ServerError'
  }
}

/**
 * Sync HTTP Client for backend communication
 */
export class SyncClient {
  private readonly baseUrl: string
  private readonly timeout: number
  private abortController: AbortController | null = null

  constructor(
    baseUrl: string = import.meta.env.VITE_API_URL || 'http://localhost:8080',
    timeout: number = 30000
  ) {
    this.baseUrl = baseUrl.replace(/\/$/, '') // Remove trailing slash
    this.timeout = timeout
  }

  /**
   * Get authentication token from tier store
   * Note: This assumes tier store will have authentication info
   * You may need to adjust this based on your auth implementation
   */
  private getAuthToken(): string | null {
    // For now, check if user has Individual or Team tier (implies authentication)
    const tierStore = useTierStore.getState()
    const licenseKey = tierStore.licenseKey

    // In production, you'd get a proper JWT token here
    // For now, we'll use the license key as a bearer token
    return licenseKey || null
  }

  /**
   * Check if user is authenticated and authorized for sync
   */
  private isAuthorized(): boolean {
    const tierStore = useTierStore.getState()
    return tierStore.hasFeature('sync')
  }

  /**
   * Build request headers with authentication
   */
  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }

    const token = this.getAuthToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    return headers
  }

  /**
   * Execute HTTP request with timeout and error handling
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    // Check authorization first
    if (!this.isAuthorized()) {
      throw new AuthenticationError('Sync feature requires Individual or Team tier')
    }

    // Create abort controller for timeout
    this.abortController = new AbortController()
    const timeoutId = setTimeout(() => {
      this.abortController?.abort()
    }, this.timeout)

    const url = `${this.baseUrl}${endpoint}`

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          ...this.getHeaders(),
          ...options.headers,
        },
        signal: this.abortController.signal,
      })

      clearTimeout(timeoutId)

      // Handle HTTP errors
      if (!response.ok) {
        const errorBody = await response.text().catch(() => '')

        if (response.status === 401 || response.status === 403) {
          throw new AuthenticationError(
            errorBody || 'Authentication required'
          )
        }

        if (response.status >= 500) {
          throw new ServerError(
            errorBody || 'Server error',
            response.status
          )
        }

        throw new SyncClientError(
          errorBody || `Request failed with status ${response.status}`,
          'REQUEST_FAILED',
          response.status
        )
      }

      // Parse JSON response
      const data = await response.json()
      return data as T
    } catch (error) {
      clearTimeout(timeoutId)

      // Re-throw our custom errors
      if (error instanceof SyncClientError) {
        throw error
      }

      // Handle abort (timeout)
      if (error instanceof Error && error.name === 'AbortError') {
        throw new NetworkError('Request timeout')
      }

      // Handle network errors
      if (error instanceof TypeError) {
        throw new NetworkError('Network request failed', error)
      }

      // Unknown error
      throw new SyncClientError(
        error instanceof Error ? error.message : 'Unknown error',
        'UNKNOWN_ERROR',
        undefined,
        error instanceof Error ? error : undefined
      )
    }
  }

  /**
   * Upload local changes to server
   */
  async uploadChanges(
    data: UploadChangesRequest
  ): Promise<UploadChangesResponse> {
    return this.request<UploadChangesResponse>('/api/sync/upload', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  /**
   * Download remote changes from server
   */
  async downloadChanges(options?: {
    since?: Date
    cursor?: string
    limit?: number
  }): Promise<DownloadChangesResponse> {
    const params = new URLSearchParams()

    if (options?.since) {
      params.append('since', options.since.getTime().toString())
    }

    if (options?.cursor) {
      params.append('cursor', options.cursor)
    }

    if (options?.limit) {
      params.append('limit', options.limit.toString())
    }

    const query = params.toString()
    const endpoint = `/api/sync/download${query ? `?${query}` : ''}`

    return this.request<DownloadChangesResponse>(endpoint, {
      method: 'GET',
    })
  }

  /**
   * Get conflicts that need resolution
   */
  async getConflicts(): Promise<Conflict[]> {
    return this.request<Conflict[]>('/api/sync/conflicts', {
      method: 'GET',
    })
  }

  /**
   * Resolve a specific conflict
   */
  async resolveConflict(
    conflictId: string,
    resolution: ConflictResolution,
    mergedData?: unknown
  ): Promise<void> {
    await this.request<void>('/api/sync/resolve', {
      method: 'POST',
      body: JSON.stringify({
        conflict_id: conflictId,
        resolution,
        merged_data: mergedData,
      }),
    })
  }

  /**
   * Get sync status from server
   */
  async getSyncStatus(): Promise<{
    lastSyncAt: number
    pendingChanges: number
    conflicts: number
  }> {
    return this.request<{
      lastSyncAt: number
      pendingChanges: number
      conflicts: number
    }>('/api/sync/status', {
      method: 'GET',
    })
  }

  /**
   * Reset sync state on server (nuclear option)
   */
  async resetSync(): Promise<void> {
    await this.request<void>('/api/sync/reset', {
      method: 'POST',
    })
  }

  /**
   * Cancel current request
   */
  cancel(): void {
    if (this.abortController) {
      this.abortController.abort()
      this.abortController = null
    }
  }

  /**
   * Health check for sync service
   */
  async healthCheck(): Promise<boolean> {
    try {
      await this.request<{ status: string }>('/api/sync/health', {
        method: 'GET',
      })
      return true
    } catch {
      return false
    }
  }
}

/**
 * Singleton instance of sync client
 */
let syncClientInstance: SyncClient | null = null

/**
 * Get the singleton sync client instance
 */
export function getSyncClient(): SyncClient {
  if (!syncClientInstance) {
    syncClientInstance = new SyncClient()
  }
  return syncClientInstance
}

/**
 * Reset the singleton instance (useful for testing)
 */
export function resetSyncClient(): void {
  if (syncClientInstance) {
    syncClientInstance.cancel()
    syncClientInstance = null
  }
}

/**
 * Check if sync is available (network + authentication)
 */
export async function isSyncAvailable(): Promise<boolean> {
  if (!navigator.onLine) {
    return false
  }

  const tierStore = useTierStore.getState()
  if (!tierStore.hasFeature('sync')) {
    return false
  }

  const client = getSyncClient()
  return client.healthCheck()
}
