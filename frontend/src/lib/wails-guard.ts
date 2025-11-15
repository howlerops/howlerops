/**
 * Wails API Guard Utilities
 *
 * Provides type-safe wrappers for calling Wails backend methods with proper error handling.
 */

export function ensureWailsAPI() {
  if (!window.go?.main?.App) {
    throw new Error('Wails runtime not available')
  }
  return window.go.main.App
}

export function ensureWailsRuntime() {
  if (!window.runtime) {
    throw new Error('Wails runtime events not available')
  }
  return window.runtime
}

/**
 * Call a Wails method with automatic runtime checking
 */
export async function callWails<T>(
  method: (app: NonNullable<NonNullable<NonNullable<typeof window.go>['main']>['App']>) => Promise<T>
): Promise<T> {
  const app = ensureWailsAPI()
  return method(app)
}

/**
 * Subscribe to Wails runtime events with automatic cleanup
 */
export function subscribeToWailsEvent(
  eventName: string,
  callback: (data: any) => void
): () => void {
  const runtime = ensureWailsRuntime()

  if (!runtime.EventsOn) {
    throw new Error('Wails EventsOn not available')
  }

  const unsubscribe = runtime.EventsOn(eventName, callback)

  if (!unsubscribe) {
    throw new Error(`Failed to subscribe to event: ${eventName}`)
  }

  return unsubscribe
}
