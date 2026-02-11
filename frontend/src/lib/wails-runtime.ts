/**
 * Wails v3 Runtime Utilities
 * Provides utilities for checking and waiting for Wails v3 runtime availability
 */

import { Call, Events, Flags,System } from '@wailsio/runtime'

// Check if Wails v3 runtime is ready
export function isWailsReady(): boolean {
  // In v3, the runtime is ready if Call is available
  return typeof window !== 'undefined' && typeof Call !== 'undefined'
}

// Wait for Wails v3 runtime to be ready
export async function waitForWails(timeoutMs: number = 5000): Promise<boolean> {
  if (isWailsReady()) return true

  return new Promise((resolve) => {
    const checkInterval = setInterval(() => {
      if (isWailsReady()) {
        clearInterval(checkInterval)
        resolve(true)
      }
    }, 100)

    // Timeout after specified milliseconds
    setTimeout(() => {
      clearInterval(checkInterval)
      resolve(false)
    }, timeoutMs)
  })
}

// Execute a Wails function with runtime check
export async function executeWailsFunction<T>(
  fn: () => Promise<T>,
  timeoutMs: number = 5000
): Promise<T> {
  const isReady = await waitForWails(timeoutMs)

  if (!isReady) {
    throw new Error('Wails v3 runtime not available after timeout')
  }

  return fn()
}

// Wrapper for Wails API calls with error handling
export async function safeWailsCall<T>(
  fn: () => Promise<T>,
  fallback?: T,
  timeoutMs: number = 5000
): Promise<T | undefined> {
  try {
    return await executeWailsFunction(fn, timeoutMs)
  } catch (error) {
    console.warn('Wails call failed:', error)
    return fallback
  }
}

// Check if we're running in a Wails environment
export function isWailsEnvironment(): boolean {
  return typeof window !== 'undefined' && isWailsReady()
}

// v3 compatibility helpers for IsDev/IsProduction
function isDev(): boolean {
  try {
    // Check debug flag or environment
    return System.IsDebug() || Flags.GetFlag('dev') === true
  } catch {
    return false
  }
}

function isProduction(): boolean {
  try {
    return !System.IsDebug() && Flags.GetFlag('dev') !== true
  } catch {
    return true
  }
}

function isDebug(): boolean {
  try {
    return System.IsDebug()
  } catch {
    return false
  }
}

// Get Wails runtime version info
export function getWailsInfo(): { version: string; ready: boolean; isDev: boolean; isProduction: boolean; isDebug: boolean } {
  if (!isWailsEnvironment()) {
    return { version: 'unknown', ready: false, isDev: false, isProduction: false, isDebug: false }
  }

  return {
    version: 'v3',
    ready: isWailsReady(),
    isDev: isDev(),
    isProduction: isProduction(),
    isDebug: isDebug()
  }
}

// Re-export Events for v3 event handling
export { Events }
