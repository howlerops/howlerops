/**
 * Wails Runtime Utilities
 * Provides utilities for checking and waiting for Wails runtime availability
 *
 * Note: Window interface extensions are defined in @/types/wails-auth.d.ts
 */

// Check if Wails runtime is ready
export function isWailsReady(): boolean {
  return typeof window !== 'undefined' && 
         !!window.go && 
         !!window.go.main && 
         !!window.go.main.App &&
         typeof window.go.main.App.GetTableStructure === 'function'
}

// Wait for Wails runtime to be ready
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
    throw new Error('Wails runtime not available after timeout')
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
  return typeof window !== 'undefined' && 
         window.go !== undefined
}

// Get Wails runtime version info
export function getWailsInfo(): { version?: string; ready: boolean } {
  if (!isWailsEnvironment()) {
    return { ready: false }
  }
  
  return {
    version: window.go?.version,
    ready: isWailsReady()
  }
}
