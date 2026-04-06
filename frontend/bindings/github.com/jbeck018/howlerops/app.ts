type UnknownRecord = Record<string, unknown>
type BindingResult = any
type BindingFn = (...args: unknown[]) => Promise<BindingResult>
type RuntimeCall = {
  ByName: (name: string, ...args: unknown[]) => Promise<BindingResult>
}

const defaultServicePrefixes = [
  'main.CatalogService',
  'main.ConnectionService',
  'main.QueryService',
  'main.SchemaDiffService',
  'main.StorageService',
  'main.UpdateService',
  'main.WailsAIService',
  'main.WailsAuthService',
  'main.WailsFileService',
  'main.WailsKeyboardService',
] as const

const servicePrefixesByMethod: Record<string, readonly string[]> = {
  AssignTableSteward: ['main.CatalogService'],
  CancelQueryStream: ['main.QueryService', 'main.WailsAIService'],
  CheckBiometricAvailability: ['main.WailsAuthService'],
  CheckForUpdates: ['main.UpdateService'],
  CheckStoredToken: ['main.WailsAuthService'],
  CompareConnectionSchemas: ['main.SchemaDiffService'],
  CompareWithSnapshot: ['main.SchemaDiffService'],
  ConfigureAIProvider: ['main.WailsAIService'],
  CreateCatalogTag: ['main.CatalogService'],
  CreateConnection: ['main.ConnectionService'],
  CreateSchemaSnapshot: ['main.SchemaDiffService'],
  DeleteAIMemorySession: ['main.WailsAIService'],
  DeleteCatalogTag: ['main.CatalogService'],
  DeletePassword: ['main.WailsAuthService'],
  DeleteQueryRows: ['main.QueryService'],
  DeleteReport: ['main.CatalogService'],
  DeleteSchemaSnapshot: ['main.SchemaDiffService'],
  DeleteSyntheticView: ['main.QueryService'],
  ExecuteMultiDatabaseQuery: ['main.QueryService'],
  ExecuteQuery: ['main.QueryService'],
  ExplainQuery: ['main.QueryService'],
  FixSQLErrorWithOptions: ['main.WailsAIService'],
  FinishWebAuthnAuthentication: ['main.WailsAuthService'],
  FinishWebAuthnRegistration: ['main.WailsAuthService'],
  GenerateSQLFromNaturalLanguage: ['main.WailsAIService'],
  GenerateMigrationSQL: ['main.SchemaDiffService'],
  GenerateMigrationSQLFromSnapshot: ['main.SchemaDiffService'],
  GenericChat: ['main.WailsAIService'],
  GetAvailableModels: ['main.WailsAIService'],
  GetCatalogStats: ['main.CatalogService'],
  GetConnectionHealth: ['main.ConnectionService'],
  GetConnectionStats: ['main.ConnectionService'],
  GetCurrentVersion: ['main.UpdateService'],
  GetDatabaseVersion: ['main.ConnectionService'],
  GetEditableMetadata: ['main.CatalogService'],
  GetMultiConnectionSchema: ['main.QueryService'],
  GetOAuthURL: ['main.WailsAuthService'],
  GetPassword: ['main.WailsAuthService'],
  GetReport: ['main.CatalogService'],
  GetSchemas: ['main.QueryService'],
  GetSyntheticView: ['main.QueryService'],
  GetTableStructure: ['main.QueryService'],
  GetTables: ['main.QueryService'],
  InsertQueryRow: ['main.QueryService'],
  ListCatalogTags: ['main.CatalogService'],
  ListColumnCatalogEntries: ['main.CatalogService'],
  ListConnectionDatabases: ['main.ConnectionService'],
  ListConnections: ['main.ConnectionService'],
  ListReports: ['main.CatalogService'],
  ListSchemaSnapshots: ['main.SchemaDiffService'],
  ListSyntheticViews: ['main.QueryService'],
  ListTableCatalogEntries: ['main.CatalogService'],
  LoadAIMemorySessions: ['main.WailsAIService'],
  MarkColumnAsPII: ['main.CatalogService'],
  OpenDownloadPage: ['main.UpdateService'],
  OpenEnvFileDialog: ['main.WailsFileService'],
  RecallAIMemorySessions: ['main.WailsAIService'],
  RefreshSchema: ['main.ConnectionService'],
  RemoveConnection: ['main.ConnectionService'],
  RunReport: ['main.CatalogService'],
  SaveToDownloads: ['main.WailsFileService'],
  SQLiteDeleteConnection: ['main.StorageService'],
  SQLiteDeleteQuery: ['main.StorageService'],
  SQLiteGetConnection: ['main.StorageService'],
  SQLiteGetConnections: ['main.StorageService'],
  SQLiteGetQueries: ['main.StorageService'],
  SQLiteGetQuery: ['main.StorageService'],
  SQLiteGetQueryHistory: ['main.StorageService'],
  SQLiteGetSetting: ['main.StorageService'],
  SQLiteSaveConnection: ['main.StorageService'],
  SQLiteSaveQuery: ['main.StorageService'],
  SQLiteSaveQueryHistory: ['main.StorageService'],
  SQLiteSetSetting: ['main.StorageService'],
  SaveAIMemorySessions: ['main.WailsAIService'],
  SaveConnection: ['main.ConnectionService'],
  SaveReport: ['main.CatalogService'],
  ShowNotification: ['main.WailsAIService'],
  StartClaudeCodeLogin: ['main.WailsAIService'],
  StartCodexLogin: ['main.WailsAIService'],
  StartWebAuthnAuthentication: ['main.WailsAuthService'],
  StartWebAuthnRegistration: ['main.WailsAuthService'],
  StorageCompleteMigration: ['main.StorageService'],
  StorageImportConnections: ['main.StorageService'],
  StorageImportHistory: ['main.StorageService'],
  StorageImportPreferences: ['main.StorageService'],
  StorageImportQueries: ['main.StorageService'],
  StorageMigrationStatus: ['main.StorageService'],
  StorePassword: ['main.WailsAuthService'],
  StreamAIQueryAgent: ['main.WailsAIService'],
  SwitchConnectionDatabase: ['main.ConnectionService'],
  SyncCatalogFromConnection: ['main.CatalogService'],
  TestAnthropicConnection: ['main.WailsAIService'],
  TestClaudeCodeConnection: ['main.WailsAIService'],
  TestCodexConnection: ['main.WailsAIService'],
  TestConnection: ['main.ConnectionService'],
  TestHuggingFaceConnection: ['main.WailsAIService'],
  TestOllamaConnection: ['main.WailsAIService'],
  TestOpenAIConnection: ['main.WailsAIService'],
  UpdateColumnCatalogEntry: ['main.CatalogService'],
  UpdateQueryRow: ['main.QueryService'],
  UpdateTableCatalogEntry: ['main.CatalogService'],
  ValidateMultiQuery: ['main.QueryService'],
}

function parseErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return typeof error === 'string' ? error : JSON.stringify(error)
}

function getLegacyBinding(method: string): ((...args: unknown[]) => BindingResult) | null {
  const legacyApp = (globalThis as {
    window?: { go?: { main?: { App?: Record<string, (...args: unknown[]) => BindingResult> } } }
  }).window?.go?.main?.App

  return typeof legacyApp?.[method] === 'function' ? legacyApp[method] : null
}

async function getRuntimeCall(): Promise<RuntimeCall> {
  const runtime = (await import('@wailsio/runtime')) as { Call: RuntimeCall }
  return runtime.Call
}

async function invokeBinding<T = BindingResult>(method: string, args: unknown[]): Promise<T> {
  const legacyBinding = getLegacyBinding(method)
  if (legacyBinding) {
    return await legacyBinding(...args) as T
  }

  const prefixes = servicePrefixesByMethod[method] ?? defaultServicePrefixes
  const candidates = [...prefixes.map(prefix => `${prefix}.${method}`), method]
  const failures: string[] = []
  const runtimeCall = await getRuntimeCall()

  for (const candidate of candidates) {
    try {
      return await (runtimeCall.ByName(candidate, ...args) as Promise<T>)
    } catch (error) {
      failures.push(`${candidate}: ${parseErrorMessage(error)}`)
    }
  }

  throw new Error(`Wails binding "${method}" unavailable. Attempts: ${failures.join(' | ')}`)
}

function makeBinding(method: string): BindingFn {
  return (...args) => invokeBinding(method, args)
}

export const AssignTableSteward = makeBinding('AssignTableSteward')
export const CancelQueryStream = makeBinding('CancelQueryStream')
export const CheckBiometricAvailability = makeBinding('CheckBiometricAvailability')
export const CheckForUpdates = makeBinding('CheckForUpdates')
export const CheckStoredToken = makeBinding('CheckStoredToken')
export const CompareConnectionSchemas = makeBinding('CompareConnectionSchemas')
export const CompareWithSnapshot = makeBinding('CompareWithSnapshot')
export const ConfigureAIProvider = makeBinding('ConfigureAIProvider')
export const CreateCatalogTag = makeBinding('CreateCatalogTag')
export const CreateConnection = makeBinding('CreateConnection')
export const CreateSchemaSnapshot = makeBinding('CreateSchemaSnapshot')
export const DeleteAIMemorySession = makeBinding('DeleteAIMemorySession')
export const DeleteCatalogTag = makeBinding('DeleteCatalogTag')
export const DeletePassword = makeBinding('DeletePassword')
export const DeleteQueryRows = makeBinding('DeleteQueryRows')
export const DeleteReport = makeBinding('DeleteReport')
export const DeleteSchemaSnapshot = makeBinding('DeleteSchemaSnapshot')
export const DeleteSyntheticView = makeBinding('DeleteSyntheticView')
export const ExecuteMultiDatabaseQuery = makeBinding('ExecuteMultiDatabaseQuery')
export const ExecuteQuery = makeBinding('ExecuteQuery')
export const ExplainQuery = makeBinding('ExplainQuery')
export const FixSQLErrorWithOptions = makeBinding('FixSQLErrorWithOptions')
export const FinishWebAuthnAuthentication = makeBinding('FinishWebAuthnAuthentication')
export const FinishWebAuthnRegistration = makeBinding('FinishWebAuthnRegistration')
export const GenerateSQLFromNaturalLanguage = makeBinding('GenerateSQLFromNaturalLanguage')
export const GenerateMigrationSQL = makeBinding('GenerateMigrationSQL')
export const GenerateMigrationSQLFromSnapshot = makeBinding('GenerateMigrationSQLFromSnapshot')
export const GenericChat = makeBinding('GenericChat')
export const GetAvailableModels = makeBinding('GetAvailableModels')
export const GetCatalogStats = makeBinding('GetCatalogStats')
export const GetConnectionHealth = makeBinding('GetConnectionHealth')
export const GetConnectionStats = makeBinding('GetConnectionStats')
export const GetCurrentVersion = makeBinding('GetCurrentVersion')
export const GetDatabaseVersion = makeBinding('GetDatabaseVersion')
export const GetEditableMetadata = makeBinding('GetEditableMetadata')
export const GetMultiConnectionSchema = makeBinding('GetMultiConnectionSchema')
export const GetOAuthURL = makeBinding('GetOAuthURL')
export const GetPassword = makeBinding('GetPassword')
export const GetReport = makeBinding('GetReport')
export const GetSchemas = makeBinding('GetSchemas')
export const GetSyntheticView = makeBinding('GetSyntheticView')
export const GetTableStructure = makeBinding('GetTableStructure')
export const GetTables = makeBinding('GetTables')
export const InsertQueryRow = makeBinding('InsertQueryRow')
export const ListCatalogTags = makeBinding('ListCatalogTags')
export const ListColumnCatalogEntries = makeBinding('ListColumnCatalogEntries')
export const ListConnectionDatabases = makeBinding('ListConnectionDatabases')
export const ListConnections = makeBinding('ListConnections')
export const ListReports = makeBinding('ListReports')
export const ListSchemaSnapshots = makeBinding('ListSchemaSnapshots')
export const ListSyntheticViews = makeBinding('ListSyntheticViews')
export const ListTableCatalogEntries = makeBinding('ListTableCatalogEntries')
export const LoadAIMemorySessions = makeBinding('LoadAIMemorySessions')
export const MarkColumnAsPII = makeBinding('MarkColumnAsPII')
export const OpenDownloadPage = makeBinding('OpenDownloadPage')
export const OpenEnvFileDialog = makeBinding('OpenEnvFileDialog')
export const RecallAIMemorySessions = makeBinding('RecallAIMemorySessions')
export const RefreshSchema = makeBinding('RefreshSchema')
export const RemoveConnection = makeBinding('RemoveConnection')
export const RunReport = makeBinding('RunReport')
export const SaveToDownloads = makeBinding('SaveToDownloads')
export const SQLiteDeleteConnection = makeBinding('SQLiteDeleteConnection')
export const SQLiteDeleteQuery = makeBinding('SQLiteDeleteQuery')
export const SQLiteGetConnection = makeBinding('SQLiteGetConnection')
export const SQLiteGetConnections = makeBinding('SQLiteGetConnections')
export const SQLiteGetQueries = makeBinding('SQLiteGetQueries')
export const SQLiteGetQuery = makeBinding('SQLiteGetQuery')
export const SQLiteGetQueryHistory = makeBinding('SQLiteGetQueryHistory')
export const SQLiteGetSetting = makeBinding('SQLiteGetSetting')
export const SQLiteSaveConnection = makeBinding('SQLiteSaveConnection')
export const SQLiteSaveQuery = makeBinding('SQLiteSaveQuery')
export const SQLiteSaveQueryHistory = makeBinding('SQLiteSaveQueryHistory')
export const SQLiteSetSetting = makeBinding('SQLiteSetSetting')
export const SaveAIMemorySessions = makeBinding('SaveAIMemorySessions')
export const SaveConnection = makeBinding('SaveConnection')
export const SaveReport = makeBinding('SaveReport')
export const ShowNotification = makeBinding('ShowNotification')
export const StartClaudeCodeLogin = makeBinding('StartClaudeCodeLogin')
export const StartCodexLogin = makeBinding('StartCodexLogin')
export const StartWebAuthnAuthentication = makeBinding('StartWebAuthnAuthentication')
export const StartWebAuthnRegistration = makeBinding('StartWebAuthnRegistration')
export const StorageCompleteMigration = makeBinding('StorageCompleteMigration')
export const StorageImportConnections = makeBinding('StorageImportConnections')
export const StorageImportHistory = makeBinding('StorageImportHistory')
export const StorageImportPreferences = makeBinding('StorageImportPreferences')
export const StorageImportQueries = makeBinding('StorageImportQueries')
export const StorageMigrationStatus = makeBinding('StorageMigrationStatus')
export const StorePassword = makeBinding('StorePassword')
export const StreamAIQueryAgent = makeBinding('StreamAIQueryAgent')
export const SwitchConnectionDatabase = makeBinding('SwitchConnectionDatabase')
export const SyncCatalogFromConnection = makeBinding('SyncCatalogFromConnection')
export const TestAnthropicConnection = makeBinding('TestAnthropicConnection')
export const TestClaudeCodeConnection = makeBinding('TestClaudeCodeConnection')
export const TestCodexConnection = makeBinding('TestCodexConnection')
export const TestConnection = makeBinding('TestConnection')
export const TestHuggingFaceConnection = makeBinding('TestHuggingFaceConnection')
export const TestOllamaConnection = makeBinding('TestOllamaConnection')
export const TestOpenAIConnection = makeBinding('TestOpenAIConnection')
export const UpdateColumnCatalogEntry = makeBinding('UpdateColumnCatalogEntry')
export const UpdateQueryRow = makeBinding('UpdateQueryRow')
export const UpdateTableCatalogEntry = makeBinding('UpdateTableCatalogEntry')
export const ValidateMultiQuery = makeBinding('ValidateMultiQuery')

export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue }
export type JSONRecord = Record<string, JSONValue>
export type BindingObject = UnknownRecord
