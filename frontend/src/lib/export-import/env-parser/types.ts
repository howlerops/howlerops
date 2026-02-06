/**
 * .env File Database Connection Import Types
 *
 * Defines the data structures for importing database connections from .env files.
 * See ADR-028 for architecture decisions and rationale.
 *
 * @module lib/export-import/env-parser/types
 */

import type { ConflictResolution, DatabaseTypeString } from '../types'

// =============================================================================
// Raw Environment Variable Types
// =============================================================================

/**
 * Raw key-value pair from .env file
 */
export interface EnvEntry {
  /** Environment variable name */
  key: string

  /** Environment variable value */
  value: string

  /** Line number in source file (1-indexed) */
  lineNumber: number

  /** Original line from file (for debugging/display) */
  raw: string
}

/**
 * Result of parsing a .env file
 */
export interface EnvParseResult {
  /** Successfully parsed entries */
  entries: EnvEntry[]

  /** Errors encountered during parsing */
  errors: EnvParseError[]

  /** Parsing metadata */
  metadata: {
    /** Total lines in file */
    totalLines: number

    /** Number of valid entries extracted */
    validEntries: number

    /** Lines skipped (comments, blank lines) */
    skippedLines: number
  }
}

/**
 * Error encountered during .env parsing
 */
export interface EnvParseError {
  /** Line number where error occurred (1-indexed) */
  lineNumber: number

  /** Original line content */
  line: string

  /** Human-readable error message */
  message: string
}

// =============================================================================
// Extended Entry Types (for comment parsing)
// =============================================================================

/**
 * Extended entry that tracks comment status
 */
export interface EnvEntryExtended extends EnvEntry {
  /** Whether this line was commented out */
  isCommented: boolean

  /** Comment prefix (e.g., "# " or "## ") */
  commentPrefix?: string

  /** Detected environment from comment context (e.g., "production", "staging") */
  commentEnvironment?: string
}

/**
 * Result of parsing including commented entries
 */
export interface EnvParseResultExtended extends EnvParseResult {
  /** Commented-out entries (potential alternate connections) */
  commentedEntries: EnvEntryExtended[]

  /** Groups of related commented variables */
  commentedGroups: CommentedConnectionGroup[]
}

/**
 * A group of commented entries that form a potential connection
 */
export interface CommentedConnectionGroup {
  /** Entries in this group */
  entries: EnvEntryExtended[]

  /** Detected environment (Production, Staging, etc.) */
  environment: string

  /** Whether this appears to be a complete connection */
  isComplete: boolean

  /** Line number of environment header comment */
  headerLineNumber?: number
}

// =============================================================================
// AI Extraction Types
// =============================================================================

/**
 * Confidence level for AI extraction
 * - high: Clear pattern match, high certainty
 * - medium: Likely correct but some ambiguity
 * - low: Best guess, user should verify
 */
export type ExtractionConfidence = 'high' | 'medium' | 'low'

/**
 * Source info for an extracted field
 * Tracks which environment variable was used and confidence
 */
export interface FieldSource {
  /** Environment variable key that provided this value */
  envKey: string

  /** Line number in source file */
  lineNumber: number

  /** Confidence in this extraction */
  confidence: ExtractionConfidence

  /** Alternative keys that could have been used */
  alternativeKeys?: string[]
}

/**
 * A database connection extracted from .env file by AI
 *
 * This is the intermediate type used during the import flow.
 * Once confirmed by user, it's converted to ConnectionFormData for import.
 */
export interface ParsedEnvConnection {
  /** Unique ID for this parsed connection (transient, for UI tracking) */
  tempId: string

  /** AI-suggested connection name (user can edit) */
  suggestedName: string

  /** Detected database type */
  type: DatabaseTypeString

  /** Confidence in database type detection */
  typeConfidence: ExtractionConfidence

  /** Source info for type detection */
  typeSource?: FieldSource

  // ---------------------------------------------------------------------------
  // Connection Details (all optional - AI may not extract everything)
  // ---------------------------------------------------------------------------

  /** Database host/hostname */
  host?: string
  hostSource?: FieldSource

  /** Database port */
  port?: number
  portSource?: FieldSource

  /** Database name */
  database?: string
  databaseSource?: FieldSource

  /** Connection username */
  username?: string
  usernameSource?: FieldSource

  /** Connection password */
  password?: string
  passwordSource?: FieldSource

  /** SSL/TLS mode setting */
  sslMode?: string
  sslModeSource?: FieldSource

  // ---------------------------------------------------------------------------
  // Connection String Support
  // ---------------------------------------------------------------------------

  /** Full connection string if detected (e.g., DATABASE_URL) */
  connectionString?: string
  connectionStringSource?: FieldSource

  // ---------------------------------------------------------------------------
  // Status and Validation
  // ---------------------------------------------------------------------------

  /** Overall extraction confidence (minimum of all field confidences) */
  overallConfidence: ExtractionConfidence

  /** AI explanation of how this connection was detected */
  extractionNotes?: string

  /** Whether user has reviewed/edited this connection */
  isReviewed: boolean

  /** User chose to skip importing this connection */
  isSkipped: boolean

  /** Validation errors (missing required fields, invalid values, etc.) */
  validationErrors: string[]

  /** Detected environment (dev, staging, prod, etc.) */
  detectedEnvironment?: string

  /** Source of environment detection */
  environmentSource?: 'variable-prefix' | 'file-name' | 'ai-inference'

  /** ID of existing connection this might duplicate */
  duplicateOfId?: string

  /** Name of existing connection this might duplicate */
  duplicateOfName?: string
}

/**
 * Result of AI connection extraction
 */
export interface EnvConnectionExtractionResult {
  /** Extracted database connections */
  connections: ParsedEnvConnection[]

  /** Environment entries not matched to any connection */
  unusedEntries: EnvEntry[]

  /** Overall AI confidence in extraction */
  aiConfidence: ExtractionConfidence

  /** Time taken to process (milliseconds) */
  processingTime: number
}

// =============================================================================
// AI Response Types (for parsing AI output)
// =============================================================================

/**
 * Structure expected from AI response
 * This matches the JSON format requested in the prompt
 */
export interface AIExtractionResponse {
  connections: AIExtractedConnection[]
  unusedKeys: string[]
}

/**
 * Single connection as returned by AI
 */
export interface AIExtractedConnection {
  name: string
  type: string
  typeConfidence: 'high' | 'medium' | 'low'
  environment?: string
  host?: string
  hostKey?: string
  port?: number
  portKey?: string
  database?: string
  databaseKey?: string
  username?: string
  usernameKey?: string
  password?: string
  passwordKey?: string
  sslMode?: string
  sslModeKey?: string
  connectionString?: string
  connectionStringKey?: string
  notes?: string
}

// =============================================================================
// Import Options and Result Types
// =============================================================================

/**
 * Options for .env import process
 */
export interface EnvImportOptions {
  /** Skip connections with low confidence without asking */
  skipLowConfidence: boolean

  /** Auto-generate names for connections without suggested names */
  autoGenerateNames: boolean

  /** How to handle connections that conflict with existing */
  conflictResolution: ConflictResolution

  /** Which connections to import (by tempId), empty = all non-skipped */
  selectedConnectionIds: string[]
}

/**
 * Default import options
 */
export const DEFAULT_ENV_IMPORT_OPTIONS: EnvImportOptions = {
  skipLowConfidence: false,
  autoGenerateNames: true,
  conflictResolution: 'skip',
  selectedConnectionIds: [],
}

/**
 * Result of .env import operation
 */
export interface EnvImportResult {
  /** Number of connections successfully imported */
  imported: number

  /** Number of connections skipped */
  skipped: number

  /** Connections that failed to import */
  failed: EnvImportFailure[]

  /** IDs of successfully imported connections (in connection store) */
  importedConnectionIds: string[]
}

/**
 * Details about a failed .env import
 */
export interface EnvImportFailure {
  /** Temporary ID from parsed connection */
  tempId: string

  /** Suggested name for this connection */
  suggestedName: string

  /** Reason for failure */
  reason: string
}

// =============================================================================
// UI State Types
// =============================================================================

/**
 * Steps in the .env import flow
 */
export type EnvImportStep =
  | 'file-select'   // Initial: waiting for file upload
  | 'parsing'       // Reading file and AI extraction in progress
  | 'preview'       // Showing extracted connections for review
  | 'editing'       // User is editing a specific connection
  | 'confirming'    // Final confirmation before import
  | 'importing'     // Import in progress
  | 'complete'      // Import finished, showing results
  | 'error'         // Error state

/**
 * Full state for .env import dialog
 */
export interface EnvImportDialogState {
  /** Current step in the import flow */
  step: EnvImportStep

  /** Source file (null if not yet selected) */
  file: File | null

  /** Raw parse result from .env file */
  envParseResult: EnvParseResult | null

  /** AI extraction result */
  extractionResult: EnvConnectionExtractionResult | null

  /** ID of connection currently being edited (null if none) */
  editingConnectionId: string | null

  /** Import options */
  importOptions: EnvImportOptions

  /** Import result (available after import completes) */
  importResult: EnvImportResult | null

  /** Error message (when step is 'error') */
  error: string | null
}

/**
 * Initial state for .env import dialog
 */
export const INITIAL_ENV_IMPORT_STATE: EnvImportDialogState = {
  step: 'file-select',
  file: null,
  envParseResult: null,
  extractionResult: null,
  editingConnectionId: null,
  importOptions: DEFAULT_ENV_IMPORT_OPTIONS,
  importResult: null,
  error: null,
}

// =============================================================================
// Error Types
// =============================================================================

/**
 * Error codes for .env import failures
 */
export type EnvImportErrorCode =
  | 'FILE_READ_ERROR'        // Could not read file
  | 'FILE_TOO_LARGE'         // File exceeds size limit
  | 'PARSE_ERROR'            // .env parsing failed
  | 'AI_EXTRACTION_FAILED'   // AI could not extract connections
  | 'NO_CONNECTIONS_FOUND'   // No database connections detected
  | 'VALIDATION_ERROR'       // Connection validation failed
  | 'IMPORT_ERROR'           // Error during connection import
  | 'NETWORK_ERROR'          // Network/API error
  | 'AI_NOT_CONFIGURED'      // AI provider not set up

/**
 * Custom error class for .env import failures
 */
export class EnvImportError extends Error {
  constructor(
    public readonly code: EnvImportErrorCode,
    message: string,
    public readonly details?: Record<string, unknown>
  ) {
    super(message)
    this.name = 'EnvImportError'
  }

  /**
   * Create a user-friendly error message
   */
  toUserMessage(): string {
    switch (this.code) {
      case 'FILE_READ_ERROR':
        return 'Could not read the file. Please try again or use a different file.'
      case 'FILE_TOO_LARGE':
        return 'The file is too large. Maximum file size is 1MB.'
      case 'PARSE_ERROR':
        return 'Could not parse the .env file. Please check the file format.'
      case 'AI_EXTRACTION_FAILED':
        return 'AI could not extract connections. Try using the manual import option.'
      case 'NO_CONNECTIONS_FOUND':
        return 'No database connections were found in this file.'
      case 'VALIDATION_ERROR':
        return 'Some connection details are invalid or missing.'
      case 'IMPORT_ERROR':
        return 'Failed to import one or more connections.'
      case 'NETWORK_ERROR':
        return 'Network error occurred. Please check your connection.'
      case 'AI_NOT_CONFIGURED':
        return 'AI is not configured. Please set up an AI provider in settings.'
      default:
        return this.message
    }
  }
}

// =============================================================================
// Constants
// =============================================================================

/**
 * Maximum file size for .env import (1MB)
 */
export const MAX_ENV_FILE_SIZE = 1024 * 1024

/**
 * Accepted file extensions for .env import
 */
export const ACCEPTED_ENV_EXTENSIONS = [
  '.env',
  '.env.local',
  '.env.development',
  '.env.staging',
  '.env.production',
  '.env.test',
  '.env.example',
]

/**
 * Common patterns for connection-related environment variables
 */
export const CONNECTION_RELATED_PATTERNS = {
  // Full connection strings
  url: [
    /DATABASE_URL/i,
    /DB_URL/i,
    /.*_URI$/i,
    /.*_DSN$/i,
    /.*_CONNECTION_STRING$/i,
    /POSTGRES_URL/i,
    /MYSQL_URL/i,
    /MONGO_URL/i,
    /REDIS_URL/i,
  ],

  // Host patterns
  host: [
    /.*_HOST$/i,
    /.*_HOSTNAME$/i,
    /.*_SERVER$/i,
    /DB_HOST/i,
    /DATABASE_HOST/i,
  ],

  // Port patterns
  port: [
    /.*_PORT$/i,
    /DB_PORT/i,
    /DATABASE_PORT/i,
  ],

  // Database name patterns
  database: [
    /.*_DATABASE$/i,
    /.*_DB$/i,
    /.*_DBNAME$/i,
    /.*_NAME$/i,
    /DB_NAME/i,
  ],

  // Username patterns
  username: [
    /.*_USER$/i,
    /.*_USERNAME$/i,
    /DB_USER/i,
    /DATABASE_USER/i,
  ],

  // Password patterns
  password: [
    /.*_PASSWORD$/i,
    /.*_PASS$/i,
    /.*_SECRET$/i,
    /DB_PASSWORD/i,
    /DATABASE_PASSWORD/i,
  ],

  // SSL patterns
  ssl: [
    /.*_SSL$/i,
    /.*_SSL_MODE$/i,
    /.*_SSLMODE$/i,
  ],

  // Database-specific prefixes
  databasePrefixes: [
    /^POSTGRES/i,
    /^PG_/i,
    /^PGHOST/i,
    /^MYSQL/i,
    /^MARIADB/i,
    /^MONGODB/i,
    /^MONGO_/i,
    /^REDIS_/i,
    /^ELASTICSEARCH/i,
    /^ELASTIC_/i,
    /^CLICKHOUSE/i,
    /^MSSQL/i,
    /^SQLSERVER/i,
  ],
}

// =============================================================================
// Utility Types
// =============================================================================

/**
 * Action types for state reducer
 */
export type EnvImportAction =
  | { type: 'SET_FILE'; file: File }
  | { type: 'SET_PARSING' }
  | { type: 'SET_PARSE_RESULT'; result: EnvParseResult }
  | { type: 'SET_EXTRACTION_RESULT'; result: EnvConnectionExtractionResult }
  | { type: 'SET_STEP'; step: EnvImportStep }
  | { type: 'START_EDITING'; connectionId: string }
  | { type: 'STOP_EDITING' }
  | { type: 'UPDATE_CONNECTION'; connectionId: string; updates: Partial<ParsedEnvConnection> }
  | { type: 'TOGGLE_SKIP'; connectionId: string }
  | { type: 'REMOVE_CONNECTION'; connectionId: string }
  | { type: 'SET_OPTIONS'; options: Partial<EnvImportOptions> }
  | { type: 'SET_IMPORT_RESULT'; result: EnvImportResult }
  | { type: 'SET_ERROR'; error: string }
  | { type: 'RESET' }
