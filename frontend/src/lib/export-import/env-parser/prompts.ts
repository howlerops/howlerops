/**
 * AI Prompts for .env Connection Extraction
 *
 * Defines the prompts used to instruct AI models to extract
 * database connection details from environment variables.
 *
 * Enhanced with:
 * - Contextual proximity matching for unprefixed HOST variables
 * - Commented connection extraction as alternate environments
 * - Multi-pass extraction for consensus-based accuracy
 *
 * @module lib/export-import/env-parser/prompts
 */

import type { EnvEntry } from './types'

/**
 * Mask password values for safe logging and AI prompts
 * Shows first 2 chars and last 2 chars, masks the rest
 */
export function maskPassword(value: string): string {
  if (!value || value.length < 8) {
    return '********'
  }
  return `${value.slice(0, 2)}${'*'.repeat(Math.min(value.length - 4, 20))}${value.slice(-2)}`
}

/**
 * Check if a value looks like a password (for masking)
 */
export function looksLikePassword(key: string): boolean {
  const passwordPatterns = [
    /password/i,
    /passwd/i,
    /pass$/i,
    /secret/i,
    /api_key/i,
    /apikey/i,
    /token/i,
    /credential/i,
  ]
  return passwordPatterns.some(pattern => pattern.test(key))
}

/**
 * Mask sensitive values in entries for AI prompt
 */
export function maskSensitiveEntries(entries: EnvEntry[]): EnvEntry[] {
  return entries.map(entry => {
    // Check if this looks like a password field
    if (looksLikePassword(entry.key)) {
      return {
        ...entry,
        value: maskPassword(entry.value),
      }
    }

    // Check if value is a connection string with password
    const connectionStringWithPassword = /(:\/\/[^:]+:)([^@]+)(@)/
    if (connectionStringWithPassword.test(entry.value)) {
      return {
        ...entry,
        value: entry.value.replace(connectionStringWithPassword, (_, prefix, password, suffix) =>
          `${prefix}${maskPassword(password)}${suffix}`
        ),
      }
    }

    return entry
  })
}

/**
 * System prompt for AI extraction
 * Establishes the AI's role and knowledge base
 */
export const ENV_EXTRACTION_SYSTEM_PROMPT = `You are a database connection configuration expert. Your task is to analyze environment variables and extract database connection details.

You understand these database types and their common environment variable patterns:

## PostgreSQL
- Connection strings: postgres://, postgresql://
- Common prefixes: POSTGRES_, PG_, PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
- Default port: 5432

## MySQL / MariaDB
- Connection strings: mysql://
- Common prefixes: MYSQL_, MARIADB_, DB_ (when type hints exist)
- Default port: 3306

## MongoDB
- Connection strings: mongodb://, mongodb+srv://
- Common prefixes: MONGO_, MONGODB_
- Default port: 27017

## Redis
- Connection strings: redis://, rediss://
- Common prefixes: REDIS_
- Default port: 6379

## Elasticsearch / OpenSearch
- URL patterns: http://, https:// (usually port 9200)
- Common prefixes: ELASTIC_, ELASTICSEARCH_, OPENSEARCH_, ES_
- Default port: 9200

## ClickHouse
- Connection strings: clickhouse://, ch://
- Common prefixes: CLICKHOUSE_, CH_
- Default port: 9000 (native), 8123 (HTTP)

## SQL Server (MSSQL)
- Connection strings: sqlserver://, mssql://
- Common prefixes: MSSQL_, SQLSERVER_, SQL_
- Default port: 1433

## SQLite
- File paths or :memory:
- Common prefixes: SQLITE_
- No host/port needed

## Connection String Formats
- PostgreSQL: postgres://user:pass@host:port/database?sslmode=require
- MySQL: mysql://user:pass@host:port/database
- MongoDB: mongodb://user:pass@host:port/database or mongodb+srv://...
- Redis: redis://user:pass@host:port/db_number

## Environment Detection
Look for environment prefixes in variable names:
- DEV_, DEVELOPMENT_ → development
- STAGING_, STG_ → staging
- PROD_, PRODUCTION_ → production
- TEST_ → test
- LOCAL_ → local

Examples:
- DEV_DATABASE_URL → environment: "development"
- PROD_POSTGRES_HOST → environment: "production"
- STAGING_DB_HOST → environment: "staging"

If multiple similar connections exist (same type, different environments), group them separately.

## CRITICAL: Contextual Proximity Matching

When you see a generic variable like HOST or HOST_READER without a database prefix, use CONTEXTUAL PROXIMITY to determine its purpose:

### Proximity Signals (within 5 lines):
1. **Adjacent DB_* variables** - If HOST appears near DB_PORT, DB_USER, DB_PASSWORD, etc., treat HOST as the database host
2. **Database hints in hostname** - Values containing "postgres", "mysql", "mongo", "redis", "db", "database" indicate database servers
3. **Standard database ports nearby** - If DB_PORT=5432 is nearby, HOST is likely a PostgreSQL host
4. **Related variable naming** - HOST + HOST_READER suggest primary/replica database setup

### Example: Unprefixed HOST
\`\`\`env
HOST=postgres.leviosa-backend.orb.local
HOST_READER=postgres.leviosa-backend.orb.local
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB=graphql
\`\`\`

Analysis:
- HOST has no DB_ prefix BUT:
  - Hostname contains "postgres" → strong PostgreSQL signal
  - Adjacent lines have DB_PORT=5432 (PostgreSQL default port)
  - DB_USER, DB_PASSWORD, DB variables present
- Conclusion: HOST is the PostgreSQL host (confidence: HIGH)
- HOST_READER is a read replica (same connection, note it)

### Proximity Rule
Variables within 5 lines of each other that share a logical grouping (all DB-related) should be grouped into ONE connection, even if prefixes are inconsistent.

## CRITICAL: Commented Connection Extraction

Commented-out lines (starting with #) often contain ALTERNATE or INACTIVE database configurations. You MUST extract these as separate connections with status markers.

### Comment Patterns to Extract:
1. **Section headers** - Lines like \`# Production\` or \`# Development (active)\` indicate environment labels
2. **Commented variables** - Lines like \`# PROD_DB_HOST=...\` are inactive configurations
3. **Commented connection strings** - Full URLs that are commented out

### Example: Commented Connections
\`\`\`env
# Production (currently inactive)
# PROD_DB_HOST=prod.database.internal
# PROD_DB_PORT=5432
# PROD_DB_USER=app_user
# PROD_DB_PASSWORD=prod_secret

# Development (active)
DB_HOST=localhost
DB_PORT=5432
DB_USER=dev_user
DB_PASSWORD=dev_pass
\`\`\`

You should extract TWO connections:
1. **Production PostgreSQL** (from commented lines) - mark as "commented: true"
2. **Development PostgreSQL** (from active lines) - mark as "commented: false"

### Comment Extraction Rules:
1. Look for comment headers that indicate environment (# Production, # Staging, etc.)
2. Extract key=value pairs from commented lines (strip leading # and whitespace)
3. Group commented variables by their header section
4. Mark commented connections with \`"isCommented": true\` in the response
5. If no header exists, infer from variable prefixes (PROD_, DEV_, etc.)

## Important Rules
1. Group related variables into single connections (e.g., DB_HOST, DB_PORT, DB_USER belong together)
2. If a connection string contains all details, prefer it over individual variables
3. Look for database type hints in variable names (POSTGRES_, MYSQL_, etc.)
4. Some passwords may be masked with asterisks - preserve this
5. Return valid JSON only, no explanation text outside the JSON
6. Detect environment from variable prefixes and include in response
7. USE CONTEXTUAL PROXIMITY for generic HOST variables
8. EXTRACT COMMENTED CONNECTIONS as alternate environments

## Confidence Scoring
- HIGH: Clear database-specific prefix (POSTGRES_HOST) or full connection URL
- HIGH: Generic HOST with strong contextual signals (db keyword in hostname + adjacent DB_* vars + matching port)
- MEDIUM: Generic prefix with type hints (DB_HOST with DATABASE_TYPE=postgres)
- MEDIUM: Commented connection with clear variable names
- LOW: Inferred from context or partial information
- LOW: Commented connection with ambiguous variables`

/**
 * Build the user prompt for connection extraction
 *
 * @param entries - Environment variable entries to analyze
 * @param maskedEntries - Entries with sensitive values masked (for the prompt)
 * @param includeCommented - Whether to include commented lines for extraction
 * @returns Formatted user prompt
 */
export function buildExtractionPrompt(
  entries: EnvEntry[],
  maskedEntries?: EnvEntry[],
  includeCommented: boolean = true
): string {
  const displayEntries = maskedEntries || maskSensitiveEntries(entries)

  const entriesText = displayEntries
    .map(e => `Line ${e.lineNumber}: ${e.key}=${e.value}`)
    .join('\n')

  return `Analyze these environment variables and extract database connections:

\`\`\`env
${entriesText}
\`\`\`

## CRITICAL INSTRUCTIONS

### 1. Contextual Proximity for HOST Variables
If you see a generic "HOST" or "HOST_*" variable WITHOUT a database prefix, check:
- Are there DB_*, DATABASE_*, or similar variables within 5 lines?
- Does the hostname contain database keywords (postgres, mysql, mongo, redis, db)?
- Is there a port variable nearby that matches a database default (5432, 3306, 27017)?

If YES to any of these, associate HOST with the database connection.

**Example to handle:**
\`\`\`
HOST=postgres.backend.local     <- No DB_ prefix, but...
HOST_READER=postgres.backend.local
DB_PORT=5432                    <- PostgreSQL port nearby!
DB_USER=postgres                <- DB_* variables adjacent
DB_PASSWORD=secret
DB=myapp                        <- Database name
\`\`\`
This is ONE PostgreSQL connection where HOST is the host (confidence: HIGH due to "postgres" in hostname + adjacent DB_* vars).

### 2. Extract Commented Connections${includeCommented ? '' : ' (DISABLED for this extraction)'}
${includeCommented ? `Look for commented-out database configurations and extract them as SEPARATE connections.

**Example:**
\`\`\`
# Production
# PROD_DB_HOST=prod.db.com
# PROD_DB_PORT=5432

# Development (active)
DB_HOST=localhost
DB_PORT=5432
\`\`\`
Extract BOTH: Production (isCommented: true) AND Development (isCommented: false).` : 'Skip commented lines for this extraction.'}

For each database connection you identify, return a JSON object with this structure:

\`\`\`json
{
  "connections": [
    {
      "name": "suggested connection name based on context",
      "type": "postgresql|mysql|mariadb|mongodb|redis|elasticsearch|opensearch|clickhouse|mssql|sqlite",
      "typeConfidence": "high|medium|low",
      "environment": "development|staging|production|test|local (if detected)",
      "isCommented": false,
      "host": "hostname or IP",
      "hostKey": "ENV_KEY_USED",
      "port": 5432,
      "portKey": "ENV_KEY_USED",
      "database": "database name",
      "databaseKey": "ENV_KEY_USED",
      "username": "username",
      "usernameKey": "ENV_KEY_USED",
      "password": "password (may be masked)",
      "passwordKey": "ENV_KEY_USED",
      "sslMode": "require|prefer|disable",
      "sslModeKey": "ENV_KEY_USED",
      "connectionString": "full connection URL if found",
      "connectionStringKey": "ENV_KEY_USED",
      "replicaHost": "read replica hostname if detected (e.g., from HOST_READER)",
      "replicaHostKey": "ENV_KEY_USED",
      "proximitySignals": ["list of contextual signals used for generic HOST matching"],
      "notes": "brief explanation of how you identified this connection"
    }
  ],
  "unusedKeys": ["list of env keys not used in any connection"]
}
\`\`\`

Rules:
1. Only include fields that you can extract - omit fields with no value
2. The *Key fields should reference actual environment variable names from the input
3. Group related variables (e.g., DB_HOST + DB_PORT + DB_USER) into single connections
4. **For generic HOST variables**: Use proximity analysis and document your signals
5. If parsing a connection string, extract individual components AND keep the connection string
6. Suggest descriptive names like "Production PostgreSQL" or "Local Development DB"
7. **For commented connections**: Set isCommented: true, infer environment from comment headers or prefixes
8. Return ONLY the JSON object, no additional text`
}

/**
 * Build a focused extraction prompt for a specific database type
 * Used when user indicates expected type or for retry with more context
 */
export function buildFocusedExtractionPrompt(
  entries: EnvEntry[],
  focusType: string
): string {
  const maskedEntries = maskSensitiveEntries(entries)
  const entriesText = maskedEntries
    .map(e => `${e.key}=${e.value}`)
    .join('\n')

  return `Focus on extracting ${focusType.toUpperCase()} database connections from these environment variables:

\`\`\`env
${entriesText}
\`\`\`

Look specifically for:
${getFocusedPatterns(focusType)}

Return the same JSON format as before, focusing on ${focusType} connections.
If no ${focusType} connections are found, return {"connections": [], "unusedKeys": [...]}`
}

/**
 * Get database-specific patterns for focused extraction
 */
function getFocusedPatterns(dbType: string): string {
  switch (dbType.toLowerCase()) {
    case 'postgresql':
    case 'postgres':
      return `- POSTGRES_*, PG_*, PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
- DATABASE_URL with postgres:// or postgresql://
- Generic DB_* variables if context suggests PostgreSQL`

    case 'mysql':
      return `- MYSQL_*, MARIADB_*
- DATABASE_URL with mysql://
- Generic DB_* variables if context suggests MySQL`

    case 'mongodb':
    case 'mongo':
      return `- MONGO_*, MONGODB_*
- DATABASE_URL or MONGO_URI with mongodb:// or mongodb+srv://`

    case 'redis':
      return `- REDIS_*
- REDIS_URL with redis:// or rediss://`

    case 'elasticsearch':
    case 'opensearch':
      return `- ELASTIC_*, ELASTICSEARCH_*, OPENSEARCH_*, ES_*
- URLs with port 9200 or elasticsearch/opensearch in path`

    case 'clickhouse':
      return `- CLICKHOUSE_*, CH_*
- URLs with clickhouse:// or ch://`

    case 'mssql':
    case 'sqlserver':
      return `- MSSQL_*, SQLSERVER_*, SQL_*
- URLs with sqlserver:// or mssql://`

    case 'sqlite':
      return `- SQLITE_*
- DATABASE_PATH, DB_FILE
- Values that are file paths ending in .db, .sqlite, .sqlite3`

    default:
      return '- Any database-related environment variables'
  }
}

/**
 * Build a fallback prompt for when AI extraction fails
 * This prompt is simpler and focuses on basic pattern matching
 */
export function buildFallbackExtractionPrompt(entries: EnvEntry[]): string {
  const maskedEntries = maskSensitiveEntries(entries)
  const entriesText = maskedEntries
    .map(e => `${e.key}=${e.value}`)
    .join('\n')

  return `Extract database connections from these environment variables using simple pattern matching:

\`\`\`env
${entriesText}
\`\`\`

Look for:
1. Any *_URL or *_URI that looks like a database connection string
2. Groups of variables with matching prefixes (like DB_HOST, DB_PORT, DB_USER)
3. Variables with obvious database prefixes (POSTGRES_, MYSQL_, MONGO_, REDIS_)

Return minimal JSON:
{
  "connections": [
    {
      "name": "descriptive name",
      "type": "database type",
      "typeConfidence": "confidence level",
      "connectionString": "full URL if found",
      "connectionStringKey": "ENV_KEY",
      "host": "host if separate",
      "hostKey": "ENV_KEY",
      "notes": "what matched"
    }
  ]
}`
}

/**
 * Validate AI response structure
 * Returns null if valid, error message if invalid
 */
export function validateAIResponse(response: unknown): string | null {
  if (!response || typeof response !== 'object') {
    return 'Response is not an object'
  }

  const obj = response as Record<string, unknown>

  if (!('connections' in obj)) {
    return 'Response missing "connections" field'
  }

  if (!Array.isArray(obj.connections)) {
    return '"connections" is not an array'
  }

  for (let i = 0; i < obj.connections.length; i++) {
    const conn = obj.connections[i] as Record<string, unknown>

    if (!conn.type || typeof conn.type !== 'string') {
      return `Connection ${i}: missing or invalid "type" field`
    }

    if (!conn.name || typeof conn.name !== 'string') {
      return `Connection ${i}: missing or invalid "name" field`
    }
  }

  return null
}

// =============================================================================
// MULTI-PASS EXTRACTION SYSTEM
// =============================================================================

/**
 * Multi-pass extraction result for consensus building
 */
export interface MultiPassConnection {
  name: string
  type: string
  typeConfidence: 'high' | 'medium' | 'low'
  environment?: string
  isCommented?: boolean
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
  connectionString?: string
  connectionStringKey?: string
  replicaHost?: string
  replicaHostKey?: string
  proximitySignals?: string[]
  notes?: string
}

export interface PassOneResult {
  connections: MultiPassConnection[]
  unusedKeys: string[]
  ambiguousVariables: string[]
}

export interface PassTwoResult {
  connections: MultiPassConnection[]
  resolvedAmbiguities: Array<{
    variable: string
    resolution: string
    confidence: 'high' | 'medium' | 'low'
  }>
  newConnectionsFound: MultiPassConnection[]
}

export interface PassThreeResult {
  finalConnections: MultiPassConnection[]
  mergedConnections: Array<{
    mergedFrom: string[]
    into: string
    reason: string
  }>
  discardedConnections: Array<{
    name: string
    reason: string
  }>
  consensusConfidence: 'high' | 'medium' | 'low'
}

/**
 * System prompt for Pass 1: Initial Obvious Extraction
 */
export const MULTI_PASS_SYSTEM_PROMPT_PASS_1 = `You are Agent Alpha, the first of three extraction agents analyzing environment variables for database connections.

Your role in Pass 1 is CONSERVATIVE EXTRACTION:
- Extract ONLY connections you are highly confident about
- Flag ambiguous variables for the next pass to review
- Do NOT guess - if unsure, mark it as ambiguous

${ENV_EXTRACTION_SYSTEM_PROMPT}

## Pass 1 Specific Rules
1. Extract connections with HIGH confidence signals only:
   - Full connection strings (DATABASE_URL, etc.)
   - Clear database-specific prefixes (POSTGRES_HOST, MYSQL_USER, etc.)
   - Standard PG* variables (PGHOST, PGPORT, etc.)

2. Flag as AMBIGUOUS (for Pass 2 review):
   - Generic HOST without clear DB context
   - DB_* variables without clear type indicators
   - Variables that could belong to multiple connections
   - Commented variables (leave for Pass 2)

3. Be explicit about what you're NOT extracting and why`

/**
 * Build Pass 1 extraction prompt
 */
export function buildMultiPassPromptPass1(entries: EnvEntry[]): string {
  const maskedEntries = maskSensitiveEntries(entries)
  const entriesText = maskedEntries
    .map(e => `Line ${e.lineNumber}: ${e.key}=${e.value}`)
    .join('\n')

  return `## PASS 1: Conservative Initial Extraction

Analyze these environment variables. Extract ONLY high-confidence database connections.
Flag anything ambiguous for Pass 2 review.

\`\`\`env
${entriesText}
\`\`\`

Return JSON:
\`\`\`json
{
  "connections": [
    {
      "name": "connection name",
      "type": "postgresql|mysql|...",
      "typeConfidence": "high",
      "host": "...",
      "hostKey": "ENV_KEY",
      "port": 5432,
      "portKey": "ENV_KEY",
      "database": "...",
      "databaseKey": "ENV_KEY",
      "username": "...",
      "usernameKey": "ENV_KEY",
      "password": "...",
      "passwordKey": "ENV_KEY",
      "notes": "why this is high confidence"
    }
  ],
  "unusedKeys": ["keys not matched to any connection"],
  "ambiguousVariables": [
    "HOST - no clear DB context, needs proximity analysis",
    "DB_PORT - unclear which connection this belongs to",
    "# PROD_DB_HOST - commented, needs evaluation"
  ]
}
\`\`\`

Rules:
- Only extract connections where typeConfidence would be "high"
- List ALL ambiguous variables that need Pass 2 analysis
- Include commented lines in ambiguousVariables if they look like database configs`
}

/**
 * System prompt for Pass 2: Contextual Deep Analysis
 */
export const MULTI_PASS_SYSTEM_PROMPT_PASS_2 = `You are Agent Beta, the second extraction agent. You receive:
1. The original environment variables
2. Pass 1 results (high-confidence extractions)
3. Ambiguous variables flagged by Pass 1

Your role in Pass 2 is DEEP CONTEXTUAL ANALYSIS:
- Analyze ambiguous variables using proximity and context
- Extract commented-out connections as alternate environments
- Find connections Pass 1 may have missed
- Resolve which connection ambiguous variables belong to

${ENV_EXTRACTION_SYSTEM_PROMPT}

## Pass 2 Specific Rules

### Proximity Analysis for Ambiguous HOST Variables
When you see a flagged HOST variable:
1. Look at lines within 5 positions (before and after)
2. Check for DB_*, DATABASE_*, or specific db type variables
3. Analyze the hostname value for database keywords
4. Consider the port if one exists nearby

### Commented Connection Extraction
For commented variables:
1. Look for section headers (# Production, # Staging, etc.)
2. Strip # prefix and extract the key=value
3. Group by section headers or variable prefixes
4. Mark all extracted commented connections with isCommented: true

### Variable Attribution
If a variable could belong to multiple connections:
1. Use proximity (nearest related variables)
2. Use naming consistency (same prefix pattern)
3. If still ambiguous, duplicate to both with notes`

/**
 * Build Pass 2 extraction prompt with Pass 1 results
 */
export function buildMultiPassPromptPass2(
  entries: EnvEntry[],
  passOneResult: PassOneResult
): string {
  const maskedEntries = maskSensitiveEntries(entries)
  const entriesText = maskedEntries
    .map(e => `Line ${e.lineNumber}: ${e.key}=${e.value}`)
    .join('\n')

  return `## PASS 2: Contextual Deep Analysis

You have Pass 1 results and must now analyze ambiguous variables and find additional connections.

### Original Environment Variables:
\`\`\`env
${entriesText}
\`\`\`

### Pass 1 Results (High-Confidence Connections Found):
\`\`\`json
${JSON.stringify(passOneResult.connections, null, 2)}
\`\`\`

### Variables Pass 1 Did NOT Use:
${passOneResult.unusedKeys.map(k => `- ${k}`).join('\n') || '(none)'}

### Ambiguous Variables Flagged by Pass 1:
${passOneResult.ambiguousVariables.map(v => `- ${v}`).join('\n') || '(none)'}

## Your Tasks:

1. **Resolve Ambiguous Variables**: For each flagged variable, determine:
   - Does it belong to an existing Pass 1 connection?
   - Does it form a NEW connection (possibly with unused keys)?
   - Is it truly unrelated to databases?

2. **Extract Commented Connections**: Look for commented-out database configs:
   - Find section headers (# Production, # Development, etc.)
   - Extract key=value pairs from comment lines
   - Create separate connections marked with isCommented: true

3. **Find Missed Connections**: Check if Pass 1 missed any connections due to:
   - Non-standard naming (HOST instead of DB_HOST)
   - Mixed prefixes (HOST + DB_PORT + DB_USER)
   - Subtle type indicators

Return JSON:
\`\`\`json
{
  "connections": [
    {
      "name": "connection name",
      "type": "postgresql|mysql|...",
      "typeConfidence": "high|medium|low",
      "environment": "production|development|...",
      "isCommented": false,
      "host": "...",
      "hostKey": "ENV_KEY",
      "port": 5432,
      "portKey": "ENV_KEY",
      "proximitySignals": ["adjacent DB_PORT=5432", "hostname contains 'postgres'"],
      "notes": "explanation of extraction logic"
    }
  ],
  "resolvedAmbiguities": [
    {
      "variable": "HOST",
      "resolution": "Assigned to 'Primary PostgreSQL' - hostname contains 'postgres' and adjacent to DB_PORT=5432",
      "confidence": "high"
    }
  ],
  "newConnectionsFound": [
    {
      "name": "Production PostgreSQL (Commented)",
      "type": "postgresql",
      "typeConfidence": "medium",
      "isCommented": true,
      "notes": "Extracted from commented section"
    }
  ]
}
\`\`\`

Focus on:
1. Using PROXIMITY signals (within 5 lines) for generic HOST variables
2. Extracting ALL commented database configurations
3. Documenting your reasoning in notes and proximitySignals`
}

/**
 * System prompt for Pass 3: Consensus and Finalization
 */
export const MULTI_PASS_SYSTEM_PROMPT_PASS_3 = `You are Agent Gamma, the final arbiter. You receive:
1. Original environment variables
2. Pass 1 results (conservative extraction)
3. Pass 2 results (deep analysis with resolved ambiguities)

Your role in Pass 3 is CONSENSUS AND FINALIZATION:
- Merge duplicate connections found by both passes
- Resolve any conflicts between Pass 1 and Pass 2
- Assign final confidence scores
- Produce the definitive connection list

${ENV_EXTRACTION_SYSTEM_PROMPT}

## Pass 3 Specific Rules

### Merging Connections
If Pass 1 and Pass 2 found the same connection:
1. Keep the more complete version
2. Merge any additional fields from the other
3. Use the higher confidence if they differ
4. Document that it was confirmed by both passes

### Conflict Resolution
If passes disagree:
1. Prefer explicit DB-prefixed variables over generic ones
2. Trust Pass 2's proximity analysis for ambiguous variables
3. If still conflicting, include both but mark lower confidence

### Final Confidence Assignment
- HIGH: Confirmed by both passes with consistent data
- HIGH: Clear connection string or explicit DB prefix
- MEDIUM: Found by Pass 2 using proximity analysis with strong signals
- MEDIUM: Commented connection with clear variable names
- LOW: Inferred with weak signals or conflicts between passes

### Quality Checks
Before finalizing, verify:
1. No duplicate connections (same host+port+database)
2. All connections have at least host OR connectionString
3. Commented connections are clearly marked
4. proximitySignals documented for non-obvious matches`

/**
 * Build Pass 3 consensus prompt
 */
export function buildMultiPassPromptPass3(
  entries: EnvEntry[],
  passOneResult: PassOneResult,
  passTwoResult: PassTwoResult
): string {
  const maskedEntries = maskSensitiveEntries(entries)
  const entriesText = maskedEntries
    .map(e => `Line ${e.lineNumber}: ${e.key}=${e.value}`)
    .join('\n')

  return `## PASS 3: Consensus and Finalization

You must now produce the FINAL, DEFINITIVE list of database connections.

### Original Environment Variables:
\`\`\`env
${entriesText}
\`\`\`

### Pass 1 Results (Conservative):
\`\`\`json
${JSON.stringify(passOneResult.connections, null, 2)}
\`\`\`
Unused keys: ${passOneResult.unusedKeys.join(', ') || '(none)'}

### Pass 2 Results (Deep Analysis):
\`\`\`json
${JSON.stringify(passTwoResult.connections, null, 2)}
\`\`\`

### Pass 2 Resolved Ambiguities:
\`\`\`json
${JSON.stringify(passTwoResult.resolvedAmbiguities, null, 2)}
\`\`\`

### Pass 2 New Connections Found:
\`\`\`json
${JSON.stringify(passTwoResult.newConnectionsFound, null, 2)}
\`\`\`

## Your Tasks:

1. **Merge Duplicates**: If both passes found the same connection, merge them
2. **Resolve Conflicts**: If passes disagree, use the rules to determine truth
3. **Assign Final Confidence**: Based on confirmation across passes
4. **Quality Check**: Ensure no duplicates, all have host/connectionString

Return the FINAL JSON:
\`\`\`json
{
  "finalConnections": [
    {
      "name": "Primary PostgreSQL",
      "type": "postgresql",
      "typeConfidence": "high",
      "environment": "development",
      "isCommented": false,
      "host": "postgres.backend.local",
      "hostKey": "HOST",
      "port": 5432,
      "portKey": "DB_PORT",
      "database": "myapp",
      "databaseKey": "DB",
      "username": "postgres",
      "usernameKey": "DB_USER",
      "password": "********",
      "passwordKey": "DB_PASSWORD",
      "replicaHost": "postgres.backend.local",
      "replicaHostKey": "HOST_READER",
      "proximitySignals": [
        "HOST hostname contains 'postgres'",
        "Adjacent DB_PORT=5432 (PostgreSQL default)",
        "Confirmed by Pass 1 and Pass 2"
      ],
      "notes": "Primary connection. HOST associated via proximity analysis."
    }
  ],
  "mergedConnections": [
    {
      "mergedFrom": ["Pass 1: PostgreSQL Connection", "Pass 2: Primary PostgreSQL"],
      "into": "Primary PostgreSQL",
      "reason": "Same host and port, merged additional fields from Pass 2"
    }
  ],
  "discardedConnections": [
    {
      "name": "Possible Redis (Pass 2)",
      "reason": "Insufficient evidence - only had CACHE_HOST without Redis indicators"
    }
  ],
  "consensusConfidence": "high"
}
\`\`\`

The consensusConfidence should reflect overall extraction quality:
- "high": Most connections confirmed by both passes, clear patterns
- "medium": Some connections required inference, minor conflicts resolved
- "low": Significant ambiguity, user should carefully verify`
}

/**
 * Orchestrate multi-pass extraction
 * This is the main entry point for multi-pass extraction
 */
export interface MultiPassExtractionConfig {
  entries: EnvEntry[]
  /** Callback to execute AI extraction for each pass */
  executePass: (systemPrompt: string, userPrompt: string) => Promise<string>
}

/**
 * Type guard for PassOneResult
 */
export function isPassOneResult(obj: unknown): obj is PassOneResult {
  if (!obj || typeof obj !== 'object') return false
  const o = obj as Record<string, unknown>
  return Array.isArray(o.connections) && Array.isArray(o.unusedKeys) && Array.isArray(o.ambiguousVariables)
}

/**
 * Type guard for PassTwoResult
 */
export function isPassTwoResult(obj: unknown): obj is PassTwoResult {
  if (!obj || typeof obj !== 'object') return false
  const o = obj as Record<string, unknown>
  return Array.isArray(o.connections) && Array.isArray(o.resolvedAmbiguities) && Array.isArray(o.newConnectionsFound)
}

/**
 * Type guard for PassThreeResult
 */
export function isPassThreeResult(obj: unknown): obj is PassThreeResult {
  if (!obj || typeof obj !== 'object') return false
  const o = obj as Record<string, unknown>
  return (
    Array.isArray(o.finalConnections) &&
    Array.isArray(o.mergedConnections) &&
    Array.isArray(o.discardedConnections) &&
    typeof o.consensusConfidence === 'string'
  )
}

/**
 * Execute multi-pass extraction with consensus building
 *
 * @param config - Configuration including entries and AI execution callback
 * @returns Final extraction result after consensus
 */
export async function executeMultiPassExtraction(
  config: MultiPassExtractionConfig
): Promise<PassThreeResult> {
  const { entries, executePass } = config

  // Pass 1: Conservative extraction
  const pass1Response = await executePass(
    MULTI_PASS_SYSTEM_PROMPT_PASS_1,
    buildMultiPassPromptPass1(entries)
  )

  let passOneResult: PassOneResult
  try {
    const parsed = JSON.parse(pass1Response)
    if (!isPassOneResult(parsed)) {
      throw new Error('Invalid Pass 1 response structure')
    }
    passOneResult = parsed
  } catch {
    // Fallback if parsing fails
    passOneResult = {
      connections: [],
      unusedKeys: entries.map(e => e.key),
      ambiguousVariables: ['All variables - Pass 1 failed to extract'],
    }
  }

  // Pass 2: Deep contextual analysis
  const pass2Response = await executePass(
    MULTI_PASS_SYSTEM_PROMPT_PASS_2,
    buildMultiPassPromptPass2(entries, passOneResult)
  )

  let passTwoResult: PassTwoResult
  try {
    const parsed = JSON.parse(pass2Response)
    if (!isPassTwoResult(parsed)) {
      throw new Error('Invalid Pass 2 response structure')
    }
    passTwoResult = parsed
  } catch {
    // Fallback if parsing fails
    passTwoResult = {
      connections: passOneResult.connections,
      resolvedAmbiguities: [],
      newConnectionsFound: [],
    }
  }

  // Pass 3: Consensus and finalization
  const pass3Response = await executePass(
    MULTI_PASS_SYSTEM_PROMPT_PASS_3,
    buildMultiPassPromptPass3(entries, passOneResult, passTwoResult)
  )

  let passThreeResult: PassThreeResult
  try {
    const parsed = JSON.parse(pass3Response)
    if (!isPassThreeResult(parsed)) {
      throw new Error('Invalid Pass 3 response structure')
    }
    passThreeResult = parsed
  } catch {
    // Fallback: merge Pass 1 and Pass 2 results
    const allConnections = [
      ...passOneResult.connections,
      ...passTwoResult.connections,
      ...passTwoResult.newConnectionsFound,
    ]

    // Simple deduplication by host+port
    const seen = new Set<string>()
    const dedupedConnections = allConnections.filter(conn => {
      const key = `${conn.host}:${conn.port}:${conn.database}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })

    passThreeResult = {
      finalConnections: dedupedConnections,
      mergedConnections: [],
      discardedConnections: [],
      consensusConfidence: 'low',
    }
  }

  return passThreeResult
}
