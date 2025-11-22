/**
 * AI Error Handling Utilities
 *
 * Provides error classification, retry logic, and user-friendly error messages
 * for AI-related operations.
 */

export enum AIErrorType {
  // Provider/Network errors
  PROVIDER_UNAVAILABLE = 'PROVIDER_UNAVAILABLE',
  NETWORK_ERROR = 'NETWORK_ERROR',
  TIMEOUT = 'TIMEOUT',
  RATE_LIMIT = 'RATE_LIMIT',

  // Configuration errors
  INVALID_CONFIG = 'INVALID_CONFIG',
  MISSING_API_KEY = 'MISSING_API_KEY',
  INVALID_MODEL = 'INVALID_MODEL',

  // Request errors
  INVALID_REQUEST = 'INVALID_REQUEST',
  CONTEXT_TOO_LARGE = 'CONTEXT_TOO_LARGE',

  // Backend errors
  BACKEND_ERROR = 'BACKEND_ERROR',

  // Unknown errors
  UNKNOWN = 'UNKNOWN',
}

export interface ClassifiedError {
  type: AIErrorType
  message: string
  userMessage: string
  isRetryable: boolean
  originalError: Error
}

/**
 * Classify an error and determine if it's retryable
 */
export function classifyAIError(error: Error): ClassifiedError {
  const errorMessage = error.message.toLowerCase()

  // Network/Connection errors
  if (errorMessage.includes('network') || errorMessage.includes('fetch')) {
    return {
      type: AIErrorType.NETWORK_ERROR,
      message: error.message,
      userMessage: 'Network error. Please check your connection and try again.',
      isRetryable: true,
      originalError: error,
    }
  }

  // Timeout errors
  if (errorMessage.includes('timeout') || errorMessage.includes('timed out')) {
    return {
      type: AIErrorType.TIMEOUT,
      message: error.message,
      userMessage: 'Request timed out. The AI provider may be experiencing delays.',
      isRetryable: true,
      originalError: error,
    }
  }

  // Rate limiting
  if (errorMessage.includes('rate limit') || errorMessage.includes('too many requests')) {
    return {
      type: AIErrorType.RATE_LIMIT,
      message: error.message,
      userMessage: 'Rate limit exceeded. Please wait a moment and try again.',
      isRetryable: true,
      originalError: error,
    }
  }

  // Provider unavailable
  if (errorMessage.includes('connection refused') ||
      errorMessage.includes('unavailable') ||
      errorMessage.includes('unreachable')) {
    return {
      type: AIErrorType.PROVIDER_UNAVAILABLE,
      message: error.message,
      userMessage: 'AI provider is unavailable. Please check your connection settings.',
      isRetryable: true,
      originalError: error,
    }
  }

  // Configuration errors
  if (errorMessage.includes('api key') || errorMessage.includes('unauthorized')) {
    return {
      type: AIErrorType.MISSING_API_KEY,
      message: error.message,
      userMessage: 'Invalid or missing API key. Please check your AI configuration.',
      isRetryable: false,
      originalError: error,
    }
  }

  if (errorMessage.includes('model') && errorMessage.includes('not found')) {
    return {
      type: AIErrorType.INVALID_MODEL,
      message: error.message,
      userMessage: 'Invalid model selected. Please choose a different model.',
      isRetryable: false,
      originalError: error,
    }
  }

  // Context size errors
  if (errorMessage.includes('context') &&
      (errorMessage.includes('too large') || errorMessage.includes('exceeded'))) {
    return {
      type: AIErrorType.CONTEXT_TOO_LARGE,
      message: error.message,
      userMessage: 'Context is too large. Try simplifying your request or reducing schema size.',
      isRetryable: false,
      originalError: error,
    }
  }

  // Default to unknown
  return {
    type: AIErrorType.UNKNOWN,
    message: error.message,
    userMessage: `An error occurred: ${error.message}`,
    isRetryable: false,
    originalError: error,
  }
}

/**
 * Retry an async operation with exponential backoff
 */
export async function retryWithBackoff<T>(
  operation: () => Promise<T>,
  options: {
    maxRetries?: number
    initialDelay?: number
    maxDelay?: number
    shouldRetry?: (error: ClassifiedError) => boolean
  } = {}
): Promise<T> {
  const {
    maxRetries = 3,
    initialDelay = 1000,
    maxDelay = 10000,
    shouldRetry = (error) => error.isRetryable,
  } = options

  let lastError: ClassifiedError | null = null
  let delay = initialDelay

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await operation()
    } catch (error) {
      const classified = classifyAIError(error instanceof Error ? error : new Error(String(error)))
      lastError = classified

      // Don't retry if we've exhausted attempts or error isn't retryable
      if (attempt === maxRetries || !shouldRetry(classified)) {
        throw error
      }

      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, delay))

      // Exponential backoff with max delay
      delay = Math.min(delay * 2, maxDelay)
    }
  }

  // This should never be reached, but TypeScript needs it
  throw lastError?.originalError || new Error('Retry failed')
}

/**
 * Handle void promises with proper error logging
 */
export function handleVoidPromise(
  promise: Promise<unknown>,
  context: string
): void {
  promise.catch((error) => {
    const classified = classifyAIError(error instanceof Error ? error : new Error(String(error)))
    console.error(`[${context}] ${classified.type}:`, classified.message)
  })
}
