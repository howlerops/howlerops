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
  validateFixSQLRequest,
  validateGenerateSQLRequest,
  validateGenericMessageRequest,
  validateProviderConfig,
} from './request-validator'

// Request builders
export {
  buildFixSQLBackendRequest,
  buildFullSchemaContext,
  buildGenerateSQLBackendRequest,
  buildGenericMessageBackendRequest,
  getPrimaryConnectionId,
} from './request-builder'

// Response parsers
export {
  extractErrorMessage,
  normalizeError,
  parseFixSQLResponse,
  parseGenerateSQLResponse,
  parseGenericMessageResponse,
} from './response-parser'

// Memory management
export {
  buildMemoryContext,
  ensureActiveSession,
  ensureSessionForChatType,
  exportSessions,
  importSessions,
  recordAssistantMessage,
  recordUserMessage,
} from './memory-manager'

// Recall management
export {
  buildRecallContext,
  getRecallContext,
  recallRelatedSessions,
} from './recall-manager'
