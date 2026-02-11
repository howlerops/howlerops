# ADR-028: .env File Database Connection Import Feature

## Status
Proposed

## Context
Users managing multiple database connections often store connection details in `.env` files for their applications. Manually copying these details into HowlerOps is tedious and error-prone. We need a feature that allows users to import database connections directly from `.env` files with AI-assisted parsing.

## Decision

### Architecture Overview

```
+------------------+     +-------------------+     +------------------+
|   EnvImportDialog|     | EnvParserService  |     | AI Parser        |
|   (UI Component) |---->| (Parser + Types)  |---->| (Backend/Wails)  |
+------------------+     +-------------------+     +------------------+
        |                        |                        |
        v                        v                        v
+------------------+     +-------------------+     +------------------+
| File Drop Zone   |     | ParsedEnvConnection|    | Connection       |
| Preview/Edit     |     | (Intermediate Type)|    | Extraction       |
| Confirm Dialog   |     +-------------------+     | Prompt           |
+------------------+              |                +------------------+
        |                         |
        v                         v
+-----------------------------------+
|      ConnectionStore              |
|   (Existing Import Service)       |
+-----------------------------------+
```

### Component Hierarchy

```
EnvImportDialog (new)
├── EnvFileDropZone
├── EnvParsingProgress
├── ParsedConnectionsPreview
│   ├── ParsedConnectionCard (editable)
│   │   ├── ConnectionTypeSelector
│   │   ├── ConnectionFieldEditor
│   │   └── ConfidenceIndicator
│   └── ParsedConnectionCard (multiple)
├── ImportConfirmation
└── ImportResultSummary
```

## Data Structures

### 1. Raw Environment Variable Entry
```typescript
// src/lib/export-import/env-parser/types.ts

/**
 * Raw key-value pair from .env file
 */
export interface EnvEntry {
  key: string
  value: string
  lineNumber: number
  raw: string // Original line for debugging
}

/**
 * Result of parsing a .env file
 */
export interface EnvParseResult {
  entries: EnvEntry[]
  errors: EnvParseError[]
  metadata: {
    totalLines: number
    validEntries: number
    skippedLines: number // Comments, blank lines
  }
}

/**
 * Error encountered during .env parsing
 */
export interface EnvParseError {
  lineNumber: number
  line: string
  message: string
}
```

### 2. AI-Extracted Connection (Intermediate Type)
```typescript
// src/lib/export-import/env-parser/types.ts

/**
 * Confidence level for AI extraction
 */
export type ExtractionConfidence = 'high' | 'medium' | 'low'

/**
 * Source info for extracted field
 */
export interface FieldSource {
  envKey: string
  lineNumber: number
  confidence: ExtractionConfidence
  alternativeKeys?: string[] // Other keys that could match
}

/**
 * A connection extracted from .env file by AI
 * This is the intermediate type before becoming DatabaseConnection
 */
export interface ParsedEnvConnection {
  /** Unique ID for this parsed connection (transient) */
  tempId: string

  /** User-editable connection name (AI suggested) */
  suggestedName: string

  /** Detected database type */
  type: DatabaseTypeString
  typeConfidence: ExtractionConfidence
  typeSource?: FieldSource

  /** Connection details - all optional since AI may not extract all */
  host?: string
  hostSource?: FieldSource

  port?: number
  portSource?: FieldSource

  database?: string
  databaseSource?: FieldSource

  username?: string
  usernameSource?: FieldSource

  password?: string
  passwordSource?: FieldSource

  sslMode?: string
  sslModeSource?: FieldSource

  /** Full connection string if detected (e.g., DATABASE_URL) */
  connectionString?: string
  connectionStringSource?: FieldSource

  /** Overall extraction confidence */
  overallConfidence: ExtractionConfidence

  /** AI explanation of how connection was detected */
  extractionNotes?: string

  /** Whether user has reviewed/edited this connection */
  isReviewed: boolean

  /** User chose to skip this connection */
  isSkipped: boolean

  /** Validation errors (missing required fields, etc.) */
  validationErrors: string[]
}

/**
 * Result of AI connection extraction
 */
export interface EnvConnectionExtractionResult {
  connections: ParsedEnvConnection[]
  unusedEntries: EnvEntry[] // Entries not matched to any connection
  aiConfidence: ExtractionConfidence
  processingTime: number // milliseconds
}
```

### 3. Import Options and Result
```typescript
// src/lib/export-import/env-parser/types.ts

/**
 * Options for .env import process
 */
export interface EnvImportOptions {
  /** Skip connections with low confidence */
  skipLowConfidence: boolean

  /** Auto-generate names for unnamed connections */
  autoGenerateNames: boolean

  /** How to handle duplicate connections */
  conflictResolution: ConflictResolution

  /** Which connections to import (by tempId) */
  selectedConnectionIds: string[]
}

/**
 * Result of .env import operation
 */
export interface EnvImportResult {
  imported: number
  skipped: number
  failed: EnvImportFailure[]

  /** IDs of successfully imported connections */
  importedConnectionIds: string[]
}

/**
 * Details about a failed .env import
 */
export interface EnvImportFailure {
  tempId: string
  suggestedName: string
  reason: string
}
```

## UI Flow States

```typescript
// src/lib/export-import/env-parser/types.ts

/**
 * UI state machine for .env import dialog
 */
export type EnvImportStep =
  | 'file-select'      // Initial: waiting for file
  | 'parsing'          // Reading and AI-parsing file
  | 'preview'          // Showing extracted connections for review
  | 'editing'          // User is editing a specific connection
  | 'confirming'       // Final confirmation before import
  | 'importing'        // Import in progress
  | 'complete'         // Import finished, showing results
  | 'error'            // Error state

/**
 * Dialog state
 */
export interface EnvImportDialogState {
  step: EnvImportStep

  /** Source file info */
  file: File | null

  /** Raw parse result */
  envParseResult: EnvParseResult | null

  /** AI extraction result */
  extractionResult: EnvConnectionExtractionResult | null

  /** Connection currently being edited (by tempId) */
  editingConnectionId: string | null

  /** Import options */
  importOptions: EnvImportOptions

  /** Import result */
  importResult: EnvImportResult | null

  /** Error message */
  error: string | null
}
```

## Service Layer

### 1. Env File Parser (Client-side)
```typescript
// src/lib/export-import/env-parser/parser.ts

/**
 * Parse a .env file into key-value entries
 * This is a pure function that runs client-side
 */
export function parseEnvFile(content: string): EnvParseResult {
  // Implementation handles:
  // - Standard KEY=VALUE format
  // - Quoted values (single and double)
  // - Multiline values
  // - Comments (#)
  // - Export prefix handling (export KEY=VALUE)
  // - Empty lines
}

/**
 * Detect potential connection patterns in entries
 * Returns entries that look like they might be connection-related
 */
export function filterConnectionRelatedEntries(
  entries: EnvEntry[]
): EnvEntry[] {
  // Patterns to match:
  // - *_URL, *_URI, *_DSN
  // - DATABASE_*, DB_*, POSTGRES_*, MYSQL_*, etc.
  // - *_HOST, *_PORT, *_USER, *_PASSWORD, *_NAME
  // - REDIS_*, MONGO_*, ELASTIC_*, etc.
}
```

### 2. AI Extraction Service
```typescript
// src/lib/export-import/env-parser/ai-extractor.ts

/**
 * Use AI to extract database connections from .env entries
 */
export async function extractConnectionsWithAI(
  entries: EnvEntry[],
  config: AIConfig
): Promise<EnvConnectionExtractionResult> {
  // 1. Filter to connection-related entries
  // 2. Build prompt for AI
  // 3. Call AI service
  // 4. Parse AI response into ParsedEnvConnection[]
  // 5. Validate extracted connections
}

/**
 * Build the prompt for AI extraction
 */
export function buildExtractionPrompt(entries: EnvEntry[]): string {
  // See AI Prompt Design section below
}

/**
 * Parse AI response into structured connections
 */
export function parseAIResponse(
  response: string,
  entries: EnvEntry[]
): ParsedEnvConnection[] {
  // Parse JSON from AI response
  // Map to ParsedEnvConnection with source tracking
  // Calculate confidence scores
}
```

### 3. Connection Converter
```typescript
// src/lib/export-import/env-parser/converter.ts

/**
 * Convert a validated ParsedEnvConnection to ConnectionFormData
 */
export function convertToConnectionFormData(
  parsed: ParsedEnvConnection
): ConnectionFormData {
  // Map fields with defaults for missing values
}

/**
 * Validate a ParsedEnvConnection for import readiness
 */
export function validateParsedConnection(
  connection: ParsedEnvConnection
): ValidationResult {
  // Check required fields: name, type, database
  // Validate port range
  // Check for common issues
}
```

## AI Prompt Design

### System Prompt
```typescript
// src/lib/export-import/env-parser/prompts.ts

export const ENV_EXTRACTION_SYSTEM_PROMPT = `You are a database connection configuration expert. Your task is to analyze environment variables and extract database connection details.

You understand these database types and their common environment variable patterns:
- PostgreSQL: DATABASE_URL, POSTGRES_*, PG_*, postgres://
- MySQL: MYSQL_*, mysql://
- MariaDB: MARIADB_*, mysql://
- MongoDB: MONGO_*, mongodb://
- Redis: REDIS_*, redis://
- Elasticsearch: ELASTIC_*, ELASTICSEARCH_*, http(s)://
- ClickHouse: CLICKHOUSE_*, clickhouse://
- SQL Server: MSSQL_*, sqlserver://

Connection string formats you recognize:
- PostgreSQL: postgres://user:pass@host:port/database?sslmode=require
- MySQL: mysql://user:pass@host:port/database
- MongoDB: mongodb://user:pass@host:port/database
- Redis: redis://user:pass@host:port

Common variable patterns:
- Full URL: *_URL, *_URI, *_DSN, *_CONNECTION_STRING
- Host: *_HOST, *_HOSTNAME, *_SERVER
- Port: *_PORT
- Database: *_DATABASE, *_DB, *_NAME, *_DBNAME
- Username: *_USER, *_USERNAME
- Password: *_PASSWORD, *_PASS, *_SECRET
- SSL: *_SSL, *_SSLMODE, *_SSL_MODE

Output JSON only, no explanation text.`

export const ENV_EXTRACTION_USER_PROMPT = (entries: EnvEntry[]) => `
Analyze these environment variables and extract database connections:

${entries.map(e => `${e.key}=${maskPassword(e.value)}`).join('\n')}

For each database connection you identify, return:
{
  "connections": [
    {
      "name": "suggested connection name",
      "type": "postgresql|mysql|mariadb|mongodb|redis|elasticsearch|clickhouse|mssql|sqlite",
      "typeConfidence": "high|medium|low",
      "host": "hostname",
      "hostKey": "ENV_KEY_USED",
      "port": 5432,
      "portKey": "ENV_KEY_USED",
      "database": "database name",
      "databaseKey": "ENV_KEY_USED",
      "username": "username",
      "usernameKey": "ENV_KEY_USED",
      "password": "password",
      "passwordKey": "ENV_KEY_USED",
      "sslMode": "require|prefer|disable",
      "sslModeKey": "ENV_KEY_USED",
      "connectionString": "full connection string if found",
      "connectionStringKey": "ENV_KEY_USED",
      "notes": "how you identified this connection"
    }
  ],
  "unusedKeys": ["keys not used in any connection"]
}

Group related variables into single connections. If a connection string contains all details, prefer it over individual variables.`
```

## Component Design

### EnvImportDialog (Main Container)
```tsx
// src/components/connection-manager/components/env-import-dialog.tsx

interface EnvImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportComplete?: (result: EnvImportResult) => void
}

export function EnvImportDialog({
  open,
  onOpenChange,
  onImportComplete
}: EnvImportDialogProps) {
  const [state, dispatch] = useReducer(envImportReducer, initialState)

  // Render based on step:
  // - file-select: EnvFileDropZone
  // - parsing: Loading spinner with progress
  // - preview: ParsedConnectionsPreview
  // - editing: ConnectionEditor (full form)
  // - confirming: ConfirmationSummary
  // - importing: Progress indicator
  // - complete: ResultSummary
  // - error: ErrorDisplay with retry
}
```

### EnvFileDropZone
```tsx
// src/components/connection-manager/components/env-file-drop-zone.tsx

interface EnvFileDropZoneProps {
  onFileSelect: (file: File) => void
  isLoading?: boolean
}

export function EnvFileDropZone({ onFileSelect, isLoading }: EnvFileDropZoneProps) {
  // Visual drop zone similar to existing import dialog
  // Accept .env, .env.local, .env.development, etc.
  // File icon for .env files
}
```

### ParsedConnectionCard
```tsx
// src/components/connection-manager/components/parsed-connection-card.tsx

interface ParsedConnectionCardProps {
  connection: ParsedEnvConnection
  onEdit: () => void
  onToggleSkip: () => void
  onFieldChange: (field: string, value: string) => void
}

export function ParsedConnectionCard({
  connection,
  onEdit,
  onToggleSkip,
  onFieldChange
}: ParsedConnectionCardProps) {
  // Display extracted connection with:
  // - Confidence indicator (color-coded)
  // - Editable name field
  // - Database type dropdown
  // - Quick-edit for host, port, database, username
  // - Password field (masked, with reveal toggle)
  // - Source tracking (which env vars were used)
  // - Validation errors
  // - Skip checkbox
  // - Full edit button (opens existing connection form)
}
```

## Error Handling

### Error Categories
```typescript
// src/lib/export-import/env-parser/errors.ts

export type EnvImportErrorCode =
  | 'FILE_READ_ERROR'
  | 'PARSE_ERROR'
  | 'AI_EXTRACTION_FAILED'
  | 'NO_CONNECTIONS_FOUND'
  | 'VALIDATION_ERROR'
  | 'IMPORT_ERROR'
  | 'NETWORK_ERROR'
  | 'AI_NOT_CONFIGURED'

export class EnvImportError extends Error {
  constructor(
    public code: EnvImportErrorCode,
    message: string,
    public details?: Record<string, unknown>
  ) {
    super(message)
    this.name = 'EnvImportError'
  }
}
```

### Error Recovery Strategies
1. **FILE_READ_ERROR**: Show file format hints, suggest re-upload
2. **PARSE_ERROR**: Show problematic lines, allow manual entry
3. **AI_EXTRACTION_FAILED**: Fall back to pattern-based extraction
4. **NO_CONNECTIONS_FOUND**: Show all entries, allow manual selection
5. **VALIDATION_ERROR**: Highlight missing fields, allow user to fix
6. **AI_NOT_CONFIGURED**: Prompt user to configure AI settings

## Integration Points

### 1. Connection Manager
```tsx
// Modify src/components/connection-manager/connection-manager.tsx

// Add import option to the connection manager header
<DropdownMenu>
  <DropdownMenuTrigger>
    <Button variant="outline">Import</Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent>
    <DropdownMenuItem onClick={() => setShowJsonImport(true)}>
      From JSON Export
    </DropdownMenuItem>
    <DropdownMenuItem onClick={() => setShowEnvImport(true)}>
      From .env File
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

### 2. Existing Import Service
Reuse `importSingleConnection` from existing import-service.ts after converting `ParsedEnvConnection` to `ExportedConnection`.

### 3. AI Store
Use existing `useAIStore.sendGenericMessage` or create dedicated method for structured extraction.

## Security Considerations

1. **Password Handling**
   - Never log passwords
   - Mask passwords in AI prompts (send hashed/truncated versions)
   - Store in secure storage immediately after import
   - Clear from memory after import completes

2. **File Content**
   - Validate file size limits (prevent DoS)
   - Sanitize file content before AI processing
   - Never send full file to telemetry/analytics

3. **AI Processing**
   - Truncate sensitive values in prompts
   - Validate AI response schema before parsing
   - Rate limit AI calls

## File Structure

```
src/lib/export-import/
├── env-parser/
│   ├── index.ts                 # Public exports
│   ├── types.ts                 # Type definitions
│   ├── parser.ts                # .env file parser
│   ├── ai-extractor.ts          # AI extraction logic
│   ├── converter.ts             # Convert to ConnectionFormData
│   ├── validator.ts             # Validation logic
│   ├── prompts.ts               # AI prompts
│   └── errors.ts                # Error types
├── docs/
│   └── ADR-028-env-file-import.md  # This document
└── ... (existing files)

src/components/connection-manager/
├── components/
│   ├── env-import-dialog.tsx    # Main dialog component
│   ├── env-file-drop-zone.tsx   # Drop zone component
│   ├── parsed-connections-preview.tsx  # Preview container
│   ├── parsed-connection-card.tsx      # Individual connection card
│   └── ... (existing files)
└── ... (existing files)
```

## Testing Strategy

### Unit Tests
- `parser.ts`: Test various .env formats, edge cases
- `ai-extractor.ts`: Mock AI responses, test parsing
- `converter.ts`: Test conversion accuracy
- `validator.ts`: Test validation rules

### Integration Tests
- Full flow from file upload to connection store
- AI service integration with mock responses

### E2E Tests
- File drag-and-drop functionality
- Edit and confirm flow
- Error recovery flows

## Performance Considerations

1. **File Size Limits**: Cap at 1MB to prevent UI freeze
2. **AI Batching**: If many entries, batch AI calls
3. **Debounced Validation**: Validate on blur, not every keystroke
4. **Lazy Rendering**: Virtualize if many connections found

## Rollout Plan

1. **Phase 1**: Core parser and AI extraction (no UI)
2. **Phase 2**: Basic dialog with preview
3. **Phase 3**: Inline editing and confidence indicators
4. **Phase 4**: Error handling and recovery
5. **Phase 5**: Fallback pattern-based extraction (no AI)

## Consequences

### Positive
- Significantly faster onboarding for users with existing .env files
- Reduces manual data entry errors
- AI provides intelligent grouping of related variables
- Supports all common database types

### Negative
- Requires AI configuration for best results
- Complex state management for multi-step flow
- Potential for AI extraction errors requiring user correction

### Risks
- AI may misidentify connection types
- Complex connection strings may not parse correctly
- Multi-tenant environments may have conflicting patterns

## References
- ADR-027: Connection Export/Import (existing)
- Existing export-import service implementation
- AI store implementation for prompt patterns
