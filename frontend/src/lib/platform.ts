/**
 * Platform Detection Utilities
 *
 * Detects whether the app is running in Wails v3 desktop mode or web deployment mode.
 * This determines which authentication flow to use:
 * - Wails mode: Direct Go backend calls via generated bindings
 * - Web mode: HTTP API endpoints via fetch
 */

import { isWailsReady } from './wails-runtime'

/**
 * Check if app is running in Wails v3 desktop environment
 */
export function isWailsApp(): boolean {
  return isWailsReady()
}

/**
 * Check if app is running in web deployment mode
 */
export function isWebApp(): boolean {
  return !isWailsApp()
}

/**
 * Get platform type as string for logging/debugging
 */
export function getPlatformType(): 'wails' | 'web' {
  return isWailsApp() ? 'wails' : 'web'
}

/**
 * Get API base URL based on platform
 * - Wails: Not used (direct Go calls via bindings)
 * - Web: From VITE_API_URL env var or default to localhost:8080
 */
export function getApiBaseUrl(): string {
  if (isWailsApp()) {
    return '' // Not used in Wails mode
  }
  return import.meta.env.VITE_API_URL || 'http://localhost:8080'
}
