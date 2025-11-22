/**
 * AI Utilities Index
 *
 * Central export point for all AI-related utilities.
 * Import from here instead of individual modules.
 */

// Validators
export {
  createAIError,
  validateAIEnabled,
  validateProviderConfig,
  validateGenerateSQLRequest,
  validateFixSQLRequest,
  validateGenericMessageRequest,
} from './request-validator'

// Request builders
export {
  getPrimaryConnectionId,
  buildFullSchemaContext,
  buildGenerateSQLBackendRequest,
  buildFixSQLBackendRequest,
  buildGenericMessageBackendRequest,
} from './request-builder'

// Response parsers
export {
  parseGenerateSQLResponse,
  parseFixSQLResponse,
  parseGenericMessageResponse,
  extractErrorMessage,
  normalizeError,
} from './response-parser'

// Memory management
export {
  ensureActiveSession,
  buildMemoryContext,
  recordUserMessage,
  recordAssistantMessage,
  ensureSessionForChatType,
  exportSessions,
  importSessions,
} from './memory-manager'

// Recall management
export {
  recallRelatedSessions,
  buildRecallContext,
  getRecallContext,
} from './recall-manager'
