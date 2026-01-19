/**
 * Unified API Client Factory
 *
 * Provides a single entry point for API access that works in both
 * Wails desktop mode and web deployment mode.
 *
 * Usage:
 *   import { api } from '@/lib/api-client'
 *
 *   // Works in both desktop and web modes
 *   const connections = await api.connections.list()
 *   const result = await api.queries.execute(connectionId, sql)
 */

import { isWailsApp, getPlatformType } from '../platform'
import { wailsApiClient } from './wails-client'
import { restApiClient } from './rest-client'
import type { ApiClient, PlatformType } from './types'

// Re-export types for convenience
export * from './types'

/**
 * Get the current platform type
 */
export function getClientPlatform(): PlatformType {
  return getPlatformType()
}

/**
 * Get the appropriate API client based on the current platform.
 *
 * - In Wails desktop mode: Returns client that uses Go bindings
 * - In web mode: Returns client that uses REST API
 */
export function getApiClient(): ApiClient {
  if (isWailsApp()) {
    return wailsApiClient
  }
  return restApiClient
}

/**
 * Singleton API client instance.
 *
 * This is the primary export for most use cases.
 * The client automatically detects the platform and routes
 * calls to either Wails bindings or REST API.
 *
 * @example
 * ```typescript
 * import { api } from '@/lib/api-client'
 *
 * // List connections
 * const { data } = await api.connections.list()
 *
 * // Execute a query
 * const result = await api.queries.execute(connectionId, 'SELECT * FROM users')
 *
 * // Get table structure
 * const structure = await api.schema.columns(connectionId, 'public', 'users')
 * ```
 */
export const api: ApiClient = getApiClient()

/**
 * Force use of a specific client (for testing or special cases)
 */
export { wailsApiClient } from './wails-client'
export { restApiClient } from './rest-client'
