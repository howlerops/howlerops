/**
 * .env File Database Connection Import Module
 *
 * Provides functionality for importing database connections from .env files
 * with AI-assisted extraction and user confirmation workflow.
 *
 * See ADR-028 for architecture decisions.
 *
 * @module lib/export-import/env-parser
 *
 * @example
 * ```typescript
 * import {
 *   parseEnvFile,
 *   extractConnectionsWithAI,
 *   convertToExportedConnection,
 * } from '@/lib/export-import/env-parser'
 *
 * // Parse the .env file
 * const parseResult = parseEnvFile(fileContent)
 *
 * // Extract connections using AI
 * const extractionResult = await extractConnectionsWithAI(
 *   parseResult.entries,
 *   aiConfig
 * )
 *
 * // After user review, convert to importable format
 * const exportedConnection = convertToExportedConnection(parsedConnection)
 * ```
 */

// =============================================================================
// Type Exports
// =============================================================================

export type {
  // Raw parsing types
  EnvEntry,
  EnvParseResult,
  EnvParseError,

  // Extended parsing types (comment support)
  EnvEntryExtended,
  EnvParseResultExtended,
  CommentedConnectionGroup,

  // Extraction types
  ExtractionConfidence,
  FieldSource,
  ParsedEnvConnection,
  EnvConnectionExtractionResult,
  AIExtractionResponse,
  AIExtractedConnection,

  // Import types
  EnvImportOptions,
  EnvImportResult,
  EnvImportFailure,

  // UI state types
  EnvImportStep,
  EnvImportDialogState,
  EnvImportAction,

  // Error types
  EnvImportErrorCode,
} from './types'

export {
  // Error class
  EnvImportError,

  // Constants
  MAX_ENV_FILE_SIZE,
  ACCEPTED_ENV_EXTENSIONS,
  CONNECTION_RELATED_PATTERNS,

  // Default values
  DEFAULT_ENV_IMPORT_OPTIONS,
  INITIAL_ENV_IMPORT_STATE,
} from './types'

// =============================================================================
// Prompt Exports
// =============================================================================

export {
  // Prompts
  ENV_EXTRACTION_SYSTEM_PROMPT,
  buildExtractionPrompt,
  buildFocusedExtractionPrompt,
  buildFallbackExtractionPrompt,

  // Utilities
  maskPassword,
  looksLikePassword,
  maskSensitiveEntries,
  validateAIResponse,
} from './prompts'

// =============================================================================
// Parser Exports
// =============================================================================

export {
  parseEnvFile,
  parseEnvFileWithComments,
  filterConnectionRelatedEntries,
  filterConnectionRelatedEntriesWithProximity,
  readEnvFile,
  isAcceptedEnvFile,
  groupEntriesByPrefix,
} from './parser'

// =============================================================================
// AI Extractor Exports
// =============================================================================

export { extractConnectionsWithAI } from './ai-extractor'
export type { AIExtractorConfig } from './ai-extractor'

// =============================================================================
// Consensus Extractor Exports (Multi-Agent System)
// =============================================================================

export {
  extractConnectionsWithConsensus,
  runPatternMatcherAgent,
  runAIExtractorAgent,
  runValidatorAgent,
  runQueenEvaluator,
} from './consensus-extractor'

export type {
  AgentExtractionResult,
  AgentConnection,
  AgentAgreement,
  FieldAgreement,
  FieldConflict,
  QueenEvaluation,
  ConsensusExtractionResult,
} from './consensus-extractor'

// =============================================================================
// Converter Exports
// =============================================================================

export {
  convertToExportedConnection,
  convertToConnectionFormData,
  parseConnectionString,
  buildConnectionString,
} from './converter'

// =============================================================================
// Validator Exports
// =============================================================================

export {
  validateParsedConnection,
  validateAllParsedConnections,
  applyValidationToConnections,
  canImportConnections,
  getValidationSummary,
} from './validator'
export type { ValidationResult } from './validator'

// =============================================================================
// Reducer Exports
// =============================================================================

export {
  envImportReducer,
  envImportActions,
  envImportSelectors,
} from './reducer'

// =============================================================================
// Duplicate Detection Exports
// =============================================================================

export {
  detectDuplicates,
  detectEnvironmentFromFilename,
  applyFilenameEnvironment,
  groupByEnvironment,
  checkEnvironmentConflicts,
} from './duplicate-detector'
export type { ExistingConnection, DuplicateCheckResult } from './duplicate-detector'
