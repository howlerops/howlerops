/**
 * TypeScript declarations for Wails v3 generated app bindings
 * Auto-generated types for Go backend methods exposed to frontend
 */

import { CancellablePromise } from '@wailsio/runtime'

// Re-export model types
export * from './models'

// Connection types
export interface ConnectionConfig {
  id?: string
  type: string
  host: string
  port: number
  database: string
  username: string
  password?: string
  sslMode?: string
  connectionTimeout?: number
  parameters?: Record<string, string>
}

export interface SaveConnectionRequest {
  id: string
  name: string
  type: string
  host: string
  port: number
  database: string
  username: string
  password?: string
  sslMode?: string
  connectionTimeout?: number
  parameters?: Record<string, string>
}

export interface Connection {
  ID: string
  Name?: string
  Type?: string
  Host?: string
  Port?: number
  Database?: string
  Username?: string
}

export interface ConnectionInfo {
  id: string
  type: string
  host: string
  port: number
  database: string
  username: string
}

// Query types
export interface QueryRequest {
  connectionId: string
  query: string
  limit?: number
  offset?: number
  timeout?: number
  isExport?: boolean
}

export interface QueryResult {
  columns: string[]
  rows: unknown[][]
  rowCount: number
  affected?: number
  duration?: number
  error?: string
  editable?: EditableQueryMetadata | null
  totalRows?: number
  pagedRows?: number
  hasMore?: boolean
  offset?: number
}

export interface EditableQueryMetadata {
  enabled: boolean
  pending?: boolean
  schema?: string
  table?: string
  primaryKeyColumns?: string[]
}

export interface MultiQueryRequest {
  query: string
  timeout?: number
  strategy?: string
  limit?: number
}

export interface MultiQueryResult extends QueryResult {
  connectionsUsed?: string[]
}

export interface MultiQueryValidation {
  valid: boolean
  errors?: string[]
  estimatedStrategy?: string
}

// Row operations
export interface UpdateRowRequest {
  connectionId: string
  query: string
  columns: string[]
  schema?: string
  table?: string
  primaryKey: Record<string, unknown>
  values: Record<string, unknown>
}

export interface InsertRowRequest {
  connectionId: string
  query: string
  columns: string[]
  schema?: string
  table?: string
  values: Record<string, unknown>
}

export interface DeleteRowsRequest {
  connectionId: string
  query: string
  columns: string[]
  schema?: string
  table?: string
  primaryKeys: Record<string, unknown>[]
}

export interface RowOperationResponse {
  success: boolean
  message?: string
  row?: Record<string, unknown>
  deleted?: number
}

// Schema types
export interface TableInfo {
  name: string
  schema: string
  type?: string
  comment?: string
  rowCount?: number
  sizeBytes?: number
}

export interface TableStructure {
  columns?: ColumnInfo[]
  indexes?: IndexInfo[]
  foreign_keys?: ForeignKeyInfo[]
  triggers?: unknown[]
  statistics?: Record<string, unknown>
}

export interface ColumnInfo {
  name: string
  data_type: string
  dataType?: string
  nullable: boolean
  default_value?: string | null
  defaultValue?: string | null
  primary_key: boolean
  primaryKey?: boolean
  unique?: boolean
  indexed?: boolean
  comment?: string
  ordinal_position?: number
  ordinalPosition?: number
  character_maximum_length?: number | null
  characterMaximumLength?: number | null
  numeric_precision?: number | null
  numericPrecision?: number | null
  numeric_scale?: number | null
  numericScale?: number | null
}

export interface IndexInfo {
  name: string
  columns: string[]
  unique: boolean
  method?: string
}

export interface ForeignKeyInfo {
  name: string
  columns: string[]
  ref_table: string
  ref_columns: string[]
  on_delete?: string
  on_update?: string
}

// OAuth types
export interface OAuthURLResponse {
  authUrl?: string
  auth_url?: string
  state?: string
}

export interface BiometricAvailabilityResponse {
  available: boolean
  type?: string
}

// Database list response
export interface ListDatabasesResponse {
  success: boolean
  message?: string
  databases?: string[]
}

export interface SwitchDatabaseRequest {
  connectionId: string
  database: string
}

export interface SwitchDatabaseResponse {
  success: boolean
  message?: string
  database?: string
  reconnected?: boolean
}

// Update checker
export interface UpdateInfo {
  version?: string
  releaseDate?: string
  downloadUrl: string
  releaseNotes: string
  isRequired?: boolean
  available: boolean
  currentVersion: string
  latestVersion: string
  publishedAt: string
}

// Keyboard
export interface KeyboardAction {
  action: string
  description?: string
}

export interface KeyboardEvent {
  key: string
  modifiers?: string[]
}

// Health status
export interface HealthStatus {
  healthy?: boolean
  status: string
  message?: string
  response_time: number
  timestamp: string
  latency?: number
  error?: string
  lastChecked?: string
}

// Pool stats
export interface PoolStats {
  // snake_case fields (Go backend)
  open_connections: number
  in_use: number
  idle: number
  wait_count: number
  wait_duration: number
  max_idle_closed: number
  max_idle_time_closed: number
  max_lifetime_closed: number
  // camelCase aliases
  totalConnections?: number
  idleConnections?: number
  usedConnections?: number
  waitCount?: number
  waitDuration?: number
}

// Schema diff types
export interface SnapshotMetadata {
  id: string
  name: string
  connection_id: string
  database_type: string
  table_count: number
  created_at: string
  size_bytes: number
}

export interface SchemaSnapshot {
  id: string
  name: string
  connection_id: string
  database_type: string
  schemas: string[]
  tables: Record<string, TableInfo[]>
  structures: Record<string, TableStructure>
  created_at: string
  hash: string
}

export interface SchemaDiff {
  source_id: string
  target_id: string
  timestamp: string
  summary: DiffSummary
  tables: TableDiff[]
  duration: number
}

export interface DiffSummary {
  tables_added: number
  tables_removed: number
  tables_modified: number
  columns_added: number
  columns_removed: number
  columns_modified: number
  indexes_changed: number
  fks_changed: number
}

export type DiffStatus = 'added' | 'removed' | 'modified' | 'unchanged'

export interface TableDiff {
  schema: string
  name: string
  status: DiffStatus
  columns?: ColumnDiff[]
  indexes?: IndexDiff[]
  foreign_keys?: FKDiff[]
}

export interface ColumnDiff {
  name: string
  status: DiffStatus
  old_type?: string
  new_type?: string
  old_nullable?: boolean
  new_nullable?: boolean
  old_default?: string
  new_default?: string
}

export interface IndexDiff {
  name: string
  status: DiffStatus
  old_columns?: string[]
  new_columns?: string[]
  old_unique?: boolean
  new_unique?: boolean
  old_method?: string
  new_method?: string
}

export interface FKDiff {
  name: string
  status: DiffStatus
  old_columns?: string[]
  new_columns?: string[]
  old_ref_table?: string
  new_ref_table?: string
  old_ref_columns?: string[]
  new_ref_columns?: string[]
  old_on_delete?: string
  new_on_delete?: string
  old_on_update?: string
  new_on_update?: string
}

export interface MigrationScript {
  sql: string
  statements?: string[]
  warnings?: string[]
}

// Catalog types
export interface TableCatalogEntry {
  id: string
  connection_id: string
  schema_name: string
  table_name: string
  description?: string
  steward_user_id?: string | null
  tags?: string[]
  organization_id?: string | null
  columns?: ColumnCatalogEntry[]
  created_at: string
  updated_at: string
  created_by: string
}

export interface ColumnCatalogEntry {
  id: string
  table_catalog_id: string
  column_name: string
  description?: string
  tags?: string[]
  pii_type?: string | null
  pii_confidence?: number | null
  created_at: string
  updated_at: string
}

export interface CatalogTag {
  id: string
  name: string
  color: string
  description?: string
  organization_id?: string | null
  is_system: boolean
  created_at: string
}

export interface CatalogStats {
  total_tables: number
  total_columns: number
  tagged_tables: number
  pii_columns: number
}

export interface SearchFilters {
  connectionId?: string
  schema?: string
  hasPii?: boolean
  tags?: string[]
}

export interface SearchResults {
  tables: TableCatalogEntry[]
  columns: ColumnCatalogEntry[]
  total: number
}

export interface CatalogSyncResult {
  // snake_case (used in data-catalog.tsx)
  tables_added: number
  tables_updated: number
  columns_added: number
  // camelCase aliases
  tablesAdded?: number
  tablesUpdated?: number
  columnsAdded?: number
  errors?: string[]
}

// Storage types (SQLite)
export interface SQLiteConnection {
  id: string
  name: string
  type: string
  host: string
  port: number
  database: string
  username: string
  ssl_config?: Record<string, string>
  environments?: string[]
  created_at: string
  updated_at: string
}

export interface SQLiteQueryHistory {
  id: string
  connection_id: string
  query: string
  duration_ms: number
  row_count: number
  success: boolean
  error?: string
  executed_at: string
}

export interface SQLiteSavedQuery {
  id: string
  title: string
  description?: string
  query: string
  connection_id?: string
  folder?: string
  tags?: string[]
  created_at: string
  updated_at: string
}

// AI types
export interface AITestResponse {
  success: boolean
  message: string
  model?: string
  provider?: string
  error?: string
}

export interface NLQueryRequest {
  prompt: string
  connectionId: string
  context?: string
}

export interface GeneratedSQLResponse {
  sql: string
  explanation?: string
  confidence?: number
}

export interface GenericChatRequest {
  message?: string
  prompt?: string
  context?: string
  system?: string
  provider?: string
  model?: string
  maxTokens?: number
  temperature?: number
  metadata?: Record<string, string>
  history?: ChatMessage[]
}

export interface GenericChatResponse {
  message: string
  content?: string
  error?: string
}

export interface ChatMessage {
  role: string
  content: string
}

export interface ProviderConfig {
  provider: string
  apiKey?: string
  endpoint?: string
  model?: string
  organization?: string
  binaryPath?: string
}

export interface ProviderStatus {
  available: boolean
  configured: boolean
  error?: string
  models?: string[]
}

// Multi-connection types
export interface SchemaConflict {
  tableName: string
  connections: {
    connectionId: string
    tableName: string
    schema: string
  }[]
  resolution: string
}

export interface CombinedSchema {
  connections: Record<string, ConnectionSchema>
  crossReferences?: CrossReference[]
  conflicts: SchemaConflict[]
}

export interface ConnectionSchema {
  connectionId: string
  name: string
  type: string
  schemas: string[]
  tables: Record<string, TableInfo[]> | TableInfo[]
}

export interface CrossReference {
  sourceConnection: string
  sourceTable: string
  sourceColumn: string
  targetConnection: string
  targetTable: string
  targetColumn: string
}

export interface ValidationResult {
  valid: boolean
  errors?: string[]
  estimatedStrategy?: string
}

// File service types
export interface FileInfo {
  name: string
  path: string
  size: number
  isDir: boolean
  modTime: string
}

export interface RecentFile {
  path: string
  name: string
  lastAccessed: string
}

// Report types
export interface Report {
  id: string
  name: string
  description?: string
  query: string
  connectionId: string
  columns?: ReportColumn[]
  createdAt: string
  updatedAt: string
}

export interface ReportSummary {
  id: string
  name: string
  description?: string
  connectionId: string
  updatedAt: string
}

export interface ReportColumn {
  name: string
  type: string
  label?: string
  format?: string
}

export interface ReportRunRequest {
  reportId: string
  parameters?: Record<string, unknown>
}

export interface ReportRunResponse {
  columns: string[]
  rows: unknown[][]
  rowCount: number
  duration: number
}

// Synthetic view types
export interface ViewDefinition {
  id: string
  name: string
  description?: string
  sql: string
  connectionIds: string[]
  createdAt: string
  updatedAt: string
}

export interface SyntheticViewSummary {
  id: string
  name: string
  description?: string
  connectionIds: string[]
  updatedAt: string
}

// Migration types
export interface MigrationStatus {
  // snake_case (used in migrate-to-sqlite.ts)
  sqlite_has_data: boolean
  migration_done: boolean
  connection_count: number
  query_count: number
  history_count: number
  preferences_count: number
  // camelCase aliases
  completed?: boolean
  inProgress?: boolean
  startedAt?: string
  completedAt?: string
  connectionsImported?: number
  queriesImported?: number
  historyImported?: number
  errors?: string[]
}

export interface ImportConnectionsResult {
  imported: number
  errors?: string[]
}

export interface ImportQueriesResult {
  imported: number
  errors?: string[]
}

export interface ImportHistoryResult {
  imported: number
  errors?: string[]
}

// AI Memory types
export interface AIMemorySessionPayload {
  id: string
  title: string
  messages: ChatMessage[] | AIMemoryMessagePayload[]
  createdAt: string
  updatedAt: string
  summary?: string
  summaryTokens?: number
  metadata?: Record<string, unknown>
}

export interface AIMemoryMessagePayload {
  id?: string
  session_id?: string
  role: string
  content: string
  timestamp: string
  metadata?: Record<string, unknown>
}

export interface AIMemoryRecallResult {
  session: AIMemorySessionPayload
  relevance: number
  // Direct fields for recall-manager.ts
  title: string
  summary?: string
  content: string
}

// Query suggestion types
export interface Suggestion {
  text: string
  type: string
  description?: string
}

export interface ResultData {
  columns: string[]
  rows: unknown[][]
}

export interface VizSuggestion {
  type: string
  config: Record<string, unknown>
}

// Fix SQL types
export interface FixSQLRequest {
  query: string
  error: string
  connectionId: string
  context?: string
}

export interface FixedSQLResponse {
  sql: string
  explanation?: string
  changes?: string[]
}

export interface OptimizationResponse {
  sql: string
  explanation?: string
  improvements?: string[]
}

// ============================================================================
// Function declarations
// ============================================================================

// Connection management
export function CreateConnection(config: ConnectionConfig): CancellablePromise<ConnectionInfo>
export function TestConnection(config: ConnectionConfig): CancellablePromise<void>
export function SaveConnection(request: SaveConnectionRequest): CancellablePromise<void>
export function ListConnections(): CancellablePromise<string[]>
export function RemoveConnection(connectionId: string): CancellablePromise<void>
export function ListConnectionDatabases(connectionId: string): CancellablePromise<ListDatabasesResponse>
export function SwitchConnectionDatabase(request: SwitchDatabaseRequest): CancellablePromise<SwitchDatabaseResponse>
export function GetConnectionHealth(connectionId: string): CancellablePromise<HealthStatus>
export function GetConnectionStats(): CancellablePromise<Record<string, PoolStats>>
export function HealthCheckAll(): CancellablePromise<Record<string, HealthStatus>>
export function GetConnectionCount(): CancellablePromise<number>
export function GetConnectionIDs(): CancellablePromise<string[]>
export function GetDatabaseVersion(connectionId: string): CancellablePromise<string>

// Query execution
export function ExecuteQuery(request: QueryRequest): CancellablePromise<QueryResult>
export function ExecuteQueryStream(connectionId: string, query: string, batchSize: number): CancellablePromise<string>
export function CancelQueryStream(streamId: string): CancellablePromise<void>
export function ExplainQuery(connectionId: string, query: string): CancellablePromise<string>
export function GetEditableMetadata(jobId: string): CancellablePromise<EditableQueryMetadata | null>

// Row operations
export function UpdateQueryRow(request: UpdateRowRequest): CancellablePromise<RowOperationResponse>
export function InsertQueryRow(request: InsertRowRequest): CancellablePromise<RowOperationResponse>
export function DeleteQueryRows(request: DeleteRowsRequest): CancellablePromise<RowOperationResponse>

// Schema operations
export function GetSchemas(connectionId: string): CancellablePromise<string[]>
export function GetTables(connectionId: string, schemaName: string): CancellablePromise<TableInfo[]>
export function GetTableStructure(connectionId: string, schemaName: string, tableName: string): CancellablePromise<TableStructure>
export function InvalidateSchemaCache(connectionId: string): CancellablePromise<void>
export function InvalidateAllSchemas(): CancellablePromise<void>
export function RefreshSchema(connectionId: string): CancellablePromise<void>
export function GetSchemaCacheStats(): CancellablePromise<Record<string, unknown>>

// Multi-database query
export function ExecuteMultiDatabaseQuery(request: MultiQueryRequest): CancellablePromise<MultiQueryResult>
export function ValidateMultiQuery(query: string): CancellablePromise<ValidationResult>
export function GetMultiConnectionSchema(connectionIds: string[]): CancellablePromise<CombinedSchema>
export function ParseQueryConnections(query: string): CancellablePromise<string[]>

// Schema diff
export function CompareConnectionSchemas(sourceId: string, targetId: string): CancellablePromise<SchemaDiff>
export function CompareWithSnapshot(connectionId: string, snapshotId: string): CancellablePromise<SchemaDiff>
export function CreateSchemaSnapshot(connectionId: string, name: string): CancellablePromise<SnapshotMetadata>
export function ListSchemaSnapshots(): CancellablePromise<SnapshotMetadata[]>
export function GetSchemaSnapshot(snapshotId: string): CancellablePromise<SchemaSnapshot>
export function DeleteSchemaSnapshot(id: string): CancellablePromise<void>
export function GenerateMigrationSQL(sourceId: string, targetId: string, allowDestructive: boolean): CancellablePromise<MigrationScript>
export function GenerateMigrationSQLFromSnapshot(connectionId: string, snapshotId: string, allowDestructive: boolean): CancellablePromise<MigrationScript>

// Catalog operations
export function CreateTableCatalogEntry(entry: TableCatalogEntry): CancellablePromise<void>
export function GetTableCatalogEntry(connectionId: string, schema: string, table: string): CancellablePromise<TableCatalogEntry>
export function UpdateTableCatalogEntry(entry: TableCatalogEntry): CancellablePromise<void>
export function DeleteTableCatalogEntry(id: string): CancellablePromise<void>
export function ListTableCatalogEntries(connectionId: string): CancellablePromise<TableCatalogEntry[]>
export function CreateColumnCatalogEntry(entry: ColumnCatalogEntry): CancellablePromise<void>
export function GetColumnCatalogEntry(tableId: string, column: string): CancellablePromise<ColumnCatalogEntry>
export function UpdateColumnCatalogEntry(entry: ColumnCatalogEntry): CancellablePromise<void>
export function ListColumnCatalogEntries(tableId: string): CancellablePromise<ColumnCatalogEntry[]>
export function CreateCatalogTag(tag: CatalogTag): CancellablePromise<void>
export function ListCatalogTags(orgId: string | null): CancellablePromise<CatalogTag[]>
export function DeleteCatalogTag(id: string): CancellablePromise<void>
export function SearchCatalog(query: string, filters: SearchFilters | null): CancellablePromise<SearchResults>
export function AssignTableSteward(tableId: string, userId: string): CancellablePromise<void>
export function MarkColumnAsPII(tableId: string, columnName: string, piiType: string, confidence: number): CancellablePromise<void>
export function GetCatalogStats(connectionId: string): CancellablePromise<CatalogStats>
export function SyncCatalogFromConnection(connectionId: string): CancellablePromise<CatalogSyncResult>

// File operations
export function OpenFileDialog(): CancellablePromise<string>
export function SaveFileDialog(): CancellablePromise<string>
export function ReadFile(filePath: string): CancellablePromise<string>
export function WriteFile(filePath: string, content: string): CancellablePromise<void>
export function GetFileInfo(filePath: string): CancellablePromise<FileInfo>
export function FileExists(filePath: string): CancellablePromise<boolean>
export function GetRecentFiles(): CancellablePromise<RecentFile[]>
export function ClearRecentFiles(): CancellablePromise<void>
export function RemoveFromRecentFiles(filePath: string): CancellablePromise<void>
export function GetWorkspaceFiles(dirPath: string, extensions: string[]): CancellablePromise<FileInfo[]>
export function CreateDirectory(dirPath: string): CancellablePromise<void>
export function DeleteFile(filePath: string): CancellablePromise<void>
export function CopyFile(srcPath: string, destPath: string): CancellablePromise<void>
export function GetTempDir(): CancellablePromise<string>
export function CreateTempFile(content: string, prefix: string, suffix: string): CancellablePromise<string>
export function GetDownloadsPath(): CancellablePromise<string>
export function SaveToDownloads(filename: string, content: string): CancellablePromise<string>
export function GetHomePath(): CancellablePromise<string>

// Password management
export function StorePassword(connectionId: string, password: string, masterKeyBase64: string): CancellablePromise<void>
export function GetPassword(connectionId: string, masterKeyBase64: string): CancellablePromise<string>
export function DeletePassword(connectionId: string): CancellablePromise<void>
export function HasPassword(connectionId: string): CancellablePromise<boolean>

// Dialog operations
export function ShowInfoDialog(title: string, message: string): CancellablePromise<void>
export function ShowErrorDialog(title: string, message: string): CancellablePromise<void>
export function ShowQuestionDialog(title: string, message: string): CancellablePromise<boolean>
export function ShowNotification(title: string, message: string, isError: boolean): CancellablePromise<void>

// Keyboard operations
export function HandleKeyboardEvent(event: KeyboardEvent): CancellablePromise<void>
export function GetAllKeyboardBindings(): CancellablePromise<Record<string, KeyboardAction>>
export function GetKeyboardBindingsByCategory(): CancellablePromise<Record<string, KeyboardAction[]>>
export function AddKeyboardBinding(key: string, action: KeyboardAction): CancellablePromise<void>
export function RemoveKeyboardBinding(key: string): CancellablePromise<void>
export function ResetKeyboardBindings(): CancellablePromise<void>
export function ExportKeyboardBindings(): CancellablePromise<Record<string, KeyboardAction>>
export function ImportKeyboardBindings(bindings: Record<string, KeyboardAction>): CancellablePromise<void>

// App info
export function GetAppVersion(): CancellablePromise<string>
export function GetAppIcon(): CancellablePromise<Uint8Array>
export function GetLightIcon(): CancellablePromise<Uint8Array>
export function GetDarkIcon(): CancellablePromise<Uint8Array>
export function GetSupportedDatabaseTypes(): CancellablePromise<string[]>
export function GetDatabaseTypeInfo(dbType: string): CancellablePromise<Record<string, unknown>>
export function GetAvailableEnvironments(): CancellablePromise<string[]>

// Update checker
export function CheckForUpdates(): CancellablePromise<UpdateInfo | null>
export function GetCurrentVersion(): CancellablePromise<string>
export function OpenDownloadPage(): CancellablePromise<void>

// OAuth authentication
export function GetOAuthURL(provider: string): CancellablePromise<OAuthURLResponse>
export function CheckStoredToken(provider: string): CancellablePromise<boolean>
export function GetStoredUserInfo(provider: string): CancellablePromise<Record<string, unknown>>
export function Logout(provider: string): CancellablePromise<void>

// WebAuthn / Biometric authentication
export function CheckBiometricAvailability(): CancellablePromise<BiometricAvailabilityResponse>
export function StartWebAuthnRegistration(userId: string, username: string): CancellablePromise<string>
export function FinishWebAuthnRegistration(userId: string, credentialJSON: string): CancellablePromise<boolean>
export function StartWebAuthnAuthentication(userId: string): CancellablePromise<string>
export function FinishWebAuthnAuthentication(userId: string, assertionJSON: string): CancellablePromise<string>
export function DeleteWebAuthnCredential(userId: string): CancellablePromise<void>
export function HasWebAuthnCredential(userId: string): CancellablePromise<boolean>

// AI operations
export function TestOpenAIConnection(apiKey: string, model: string): CancellablePromise<AITestResponse>
export function TestAnthropicConnection(apiKey: string, model: string): CancellablePromise<AITestResponse>
export function TestOllamaConnection(endpoint: string, model: string): CancellablePromise<AITestResponse>
export function TestClaudeCodeConnection(binaryPath: string, model: string): CancellablePromise<AITestResponse>
export function StartClaudeCodeLogin(binaryPath: string): CancellablePromise<AITestResponse>
export function StartCodexLogin(binaryPath: string): CancellablePromise<AITestResponse>
export function TestCodexConnection(apiKey: string, model: string, organization: string): CancellablePromise<AITestResponse>
export function TestHuggingFaceConnection(endpoint: string, model: string): CancellablePromise<AITestResponse>
export function GenerateSQLFromNaturalLanguage(req: NLQueryRequest): CancellablePromise<GeneratedSQLResponse>
export function GenericChat(req: GenericChatRequest): CancellablePromise<GenericChatResponse>
export function FixSQLError(query: string, error: string, connectionId: string): CancellablePromise<FixedSQLResponse>
export function OptimizeQuery(query: string, connectionId: string): CancellablePromise<OptimizationResponse>
export function FixSQLErrorWithOptions(req: FixSQLRequest): CancellablePromise<FixedSQLResponse>
export function GetQuerySuggestions(partialQuery: string, connectionId: string): CancellablePromise<Suggestion[]>
export function SuggestVisualization(resultData: ResultData): CancellablePromise<VizSuggestion>
export function GetAIProviderStatus(): CancellablePromise<Record<string, ProviderStatus>>
export function ConfigureAIProvider(config: ProviderConfig): CancellablePromise<void>
export function GetAIConfiguration(): CancellablePromise<ProviderConfig>
export function TestAIProvider(config: ProviderConfig): CancellablePromise<ProviderStatus>

// AI Memory operations
export function SaveAIMemorySessions(sessions: AIMemorySessionPayload[]): CancellablePromise<void>
export function LoadAIMemorySessions(): CancellablePromise<AIMemorySessionPayload[]>
export function ClearAIMemorySessions(): CancellablePromise<void>
export function RecallAIMemorySessions(prompt: string, limit: number): CancellablePromise<AIMemoryRecallResult[]>
export function DeleteAIMemorySession(sessionId: string): CancellablePromise<void>

// SQLite storage operations
export function SQLiteGetConnections(): CancellablePromise<SQLiteConnection[]>
export function SQLiteGetConnection(id: string): CancellablePromise<SQLiteConnection | null>
export function SQLiteSaveConnection(connectionJSON: string): CancellablePromise<void>
export function SQLiteDeleteConnection(id: string): CancellablePromise<void>
export function SQLiteGetQueries(): CancellablePromise<SQLiteSavedQuery[]>
export function SQLiteGetQuery(id: string): CancellablePromise<SQLiteSavedQuery | null>
export function SQLiteSaveQuery(queryJSON: string): CancellablePromise<void>
export function SQLiteDeleteQuery(id: string): CancellablePromise<void>
export function SQLiteGetQueryHistory(connectionId: string, limit: number): CancellablePromise<SQLiteQueryHistory[]>
export function SQLiteSaveQueryHistory(historyJSON: string): CancellablePromise<void>
export function SQLiteGetSetting(key: string): CancellablePromise<string>
export function SQLiteSetSetting(key: string, value: string): CancellablePromise<void>

// Storage migration
export function StorageMigrationStatus(): CancellablePromise<MigrationStatus>
export function StorageImportConnections(connectionsJSON: string): CancellablePromise<ImportConnectionsResult>
export function StorageImportQueries(queriesJSON: string): CancellablePromise<ImportQueriesResult>
export function StorageImportHistory(historyJSON: string): CancellablePromise<ImportHistoryResult>
export function StorageImportPreferences(preferencesJSON: string): CancellablePromise<void>
export function StorageCompleteMigration(): CancellablePromise<void>

// Synthetic views
export function SaveSyntheticView(viewDef: ViewDefinition): CancellablePromise<string>
export function ListSyntheticViews(): CancellablePromise<SyntheticViewSummary[]>
export function GetSyntheticView(id: string): CancellablePromise<ViewDefinition>
export function DeleteSyntheticView(id: string): CancellablePromise<void>
export function ExecuteSyntheticQuery(sql: string): CancellablePromise<QueryResult>
export function GetSyntheticSchema(): CancellablePromise<Record<string, unknown>>

// Reports
export function ListReports(): CancellablePromise<ReportSummary[]>
export function GetReport(id: string): CancellablePromise<Report>
export function SaveReport(report: Report): CancellablePromise<Report>
export function DeleteReport(id: string): CancellablePromise<void>
export function RunReport(req: ReportRunRequest): CancellablePromise<ReportRunResponse>

// AI Query Agent
export interface AIQueryAgentRequest {
  connectionId: string
  message: string
  sessionId?: string
  mode?: string
  history?: ChatMessage[]
}

export interface AIQueryAgentResponse {
  sessionId: string
  messages: AIQueryAgentMessage[]
  error?: string
}

export interface AIQueryAgentMessage {
  id: string
  role: string
  content: string
  type?: string
  sql?: string
  data?: unknown
}

export function StreamAIQueryAgent(req: AIQueryAgentRequest): CancellablePromise<AIQueryAgentResponse>
