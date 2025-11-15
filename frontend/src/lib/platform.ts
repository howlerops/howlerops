/**
 * Platform Detection Utilities
 *
 * Detects whether the app is running in Wails desktop mode or web deployment mode.
 * This determines which authentication flow to use:
 * - Wails mode: Direct Go backend calls via window.go.main.App
 * - Web mode: HTTP API endpoints via fetch
 */

/**
 * Check if app is running in Wails desktop environment
 */
export function isWailsApp(): boolean {
  return typeof window !== 'undefined' && !!window.go?.main?.App
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
 * - Wails: Not used (direct Go calls)
 * - Web: From VITE_API_URL env var or default to localhost:8080
 */
export function getApiBaseUrl(): string {
  if (isWailsApp()) {
    return '' // Not used in Wails mode
  }
  return import.meta.env.VITE_API_URL || 'http://localhost:8080'
}
