/**
 * Wails v3 API Guard Utilities
 *
 * Provides type-safe wrappers for calling Wails v3 backend methods with proper error handling.
 * In v3, we import functions directly from generated bindings rather than using window.go.
 */

import { Events } from '@wailsio/runtime'
import { isWailsReady } from './wails-runtime'

// Ensure Wails v3 runtime is available
export function ensureWailsRuntime(): boolean {
  if (!isWailsReady()) {
    throw new Error('Wails v3 runtime not available')
  }
  return true
}

// Legacy API guard - now just checks runtime availability
// In v3, functions are imported directly from bindings, not from window.go
export function ensureWailsAPI(): boolean {
  return ensureWailsRuntime()
}

/**
 * Call a Wails v3 method with automatic runtime checking
 * In v3, we use direct imports from bindings, so this is just a wrapper for error handling
 */
export async function callWails<T>(
  fn: () => Promise<T>
): Promise<T> {
  ensureWailsRuntime()
  return fn()
}

/**
 * Subscribe to Wails v3 runtime events with automatic cleanup
 */
export function subscribeToWailsEvent(
  eventName: string,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Wails runtime events emit untyped data payloads
  callback: (data: any) => void
): () => void {
  ensureWailsRuntime()

  // In v3, Events.On returns the cancel function directly
  const unsubscribe = Events.On(eventName, callback)

  return unsubscribe
}

/**
 * Emit an event to the Wails v3 runtime
 */
export function emitWailsEvent(eventName: string, data?: unknown): void {
  ensureWailsRuntime()
  Events.Emit(eventName, data)
}
