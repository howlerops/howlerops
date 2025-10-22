import { useCallback } from 'react'

/**
 * Error reporting utilities for development and production
 */

export interface ErrorReport {
  message: string
  stack?: string
  componentStack?: string
  errorId: string
  timestamp: string
  url: string
  userAgent: string
  userId?: string
  sessionId?: string
}

/**
 * Create a standardized error report
 */
export function createErrorReport(
  error: Error,
  errorInfo?: { componentStack?: string },
  additionalData?: Record<string, unknown>
): ErrorReport {
  return {
    message: error.message,
    stack: error.stack,
    componentStack: errorInfo?.componentStack,
    errorId: `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    timestamp: new Date().toISOString(),
    url: window.location.href,
    userAgent: navigator.userAgent,
    ...additionalData
  }
}

/**
 * Log error to console with formatting
 */
export function logError(error: Error, context?: string, additionalData?: Record<string, unknown>) {
  const prefix = context ? `[${context}]` : '[Error]'
  
  console.group(`🚨 ${prefix} ${error.message}`)
  console.error('Error:', error)
  console.error('Stack:', error.stack)
  
  if (additionalData) {
    console.error('Additional Data:', additionalData)
  }
  
  console.groupEnd()
}

/**
 * Send error report to external service
 */
export async function reportError(
  errorReport: ErrorReport,
  endpoint?: string
): Promise<boolean> {
  try {
    const url = endpoint || '/api/errors'
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(errorReport)
    })
    
    return response.ok
  } catch (reportingError) {
    console.error('Failed to report error:', reportingError)
    return false
  }
}

/**
 * Development-only error simulation for testing error boundaries
 */
export function simulateError(context: string = 'Test') {
  if (process.env.NODE_ENV === 'development') {
    throw new Error(`Simulated error in ${context} for testing error boundary`)
  }
}

/**
 * Safe error handler that won't throw
 */
export function safeErrorHandler(
  error: Error,
  context?: string,
  onError?: (error: Error, context?: string) => void
) {
  try {
    logError(error, context)
    
    if (onError) {
      onError(error, context)
    }
    
    // Report to external service in production
    if (process.env.NODE_ENV === 'production') {
      const errorReport = createErrorReport(error)
      reportError(errorReport).catch(() => {
        // Silently fail if error reporting fails
      })
    }
  } catch (handlingError) {
    // Last resort - just log to console
    console.error('Error in error handler:', handlingError)
    console.error('Original error:', error)
  }
}

/**
 * Hook for manual error reporting
 */
export function useErrorReporter() {
  const reportError = useCallback((error: Error, context?: string) => {
    safeErrorHandler(error, context)
  }, [])
  
  const simulateError = useCallback((context?: string) => {
    if (process.env.NODE_ENV === 'development') {
      throw new Error(`Simulated error in ${context || 'component'} for testing`)
    }
  }, [])
  
  return { reportError, simulateError }
}
