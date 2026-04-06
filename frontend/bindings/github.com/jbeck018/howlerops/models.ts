export * from './pkg/catalog/models'
export * from './pkg/database/models'
export * from './pkg/schemadiff/models'
export * from './pkg/storage/models'
export * from './services/models'

type UnknownRecord = Record<string, unknown>

function parseSource<T extends UnknownRecord>(source: unknown): T {
  if (typeof source === 'string') {
    return JSON.parse(source) as T
  }
  return ((source ?? {}) as T)
}

class BaseModel {
  constructor(source: UnknownRecord = {}) {
    Object.assign(this, source)
  }

  static createFrom(source: unknown = {}): unknown {
    return new (this as unknown as { new(source?: UnknownRecord): unknown })(parseSource<UnknownRecord>(source))
  }
}

export class AITestResponse extends BaseModel {
  success = false
  message = ''
  error?: string
}

export class AIMemoryMessagePayload {
  role: string
  content: string
  timestamp: string
  metadata: Record<string, unknown>

  constructor(source: Partial<AIMemoryMessagePayload> = {}) {
    this.role = source.role ?? ''
    this.content = source.content ?? ''
    this.timestamp = source.timestamp ?? ''
    this.metadata = source.metadata ?? {}
  }

  static createFrom(source: unknown = {}): AIMemoryMessagePayload {
    return new AIMemoryMessagePayload(parseSource<Partial<AIMemoryMessagePayload>>(source))
  }
}

export class AIMemorySessionPayload {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  summary: string
  summaryTokens: number
  metadata: Record<string, unknown>
  messages: AIMemoryMessagePayload[]

  constructor(source: Partial<AIMemorySessionPayload> = {}) {
    this.id = source.id ?? ''
    this.title = source.title ?? ''
    this.createdAt = source.createdAt ?? ''
    this.updatedAt = source.updatedAt ?? ''
    this.summary = source.summary ?? ''
    this.summaryTokens = source.summaryTokens ?? 0
    this.metadata = source.metadata ?? {}
    this.messages = (source.messages ?? []).map(message =>
      message instanceof AIMemoryMessagePayload ? message : new AIMemoryMessagePayload(message)
    )
  }

  static createFrom(source: unknown = {}): AIMemorySessionPayload {
    const parsed = parseSource<Partial<AIMemorySessionPayload>>(source)
    return new AIMemorySessionPayload({
      ...parsed,
      messages: (parsed.messages ?? []).map(message => AIMemoryMessagePayload.createFrom(message)),
    })
  }
}

export class AIQueryAgentRequest {
  sessionId?: string
  message?: string
  provider?: string
  model?: string
  connectionId?: string
  connectionIds?: string[]
  schemaContext?: string
  context?: string
  temperature?: number
  maxTokens?: number
  maxRows?: number
  page?: number
  pageSize?: number

  constructor(source: Partial<AIQueryAgentRequest> = {}) {
    Object.assign(this, source)
  }

  static createFrom(source: unknown = {}): AIQueryAgentRequest {
    return new AIQueryAgentRequest(parseSource<Partial<AIQueryAgentRequest>>(source))
  }
}

export class AIMemoryRecallResult extends BaseModel {}
export class AIQueryAgentAttachment extends BaseModel {}
export class AIQueryAgentChartAttachment extends BaseModel {}
export class AIQueryAgentInsightAttachment extends BaseModel {}
export class AIQueryAgentMessage extends BaseModel {}
export class AIQueryAgentReportAttachment extends BaseModel {}
export class AIQueryAgentResponse extends BaseModel {}
export class AIQueryAgentResultAttachment extends BaseModel {}
export class AIQueryAgentSQLAttachment extends BaseModel {}
export class AlternativeQuery extends BaseModel {}
export class CatalogStats extends BaseModel {}
export class CatalogSyncResult extends BaseModel {}
export class CombinedSchema extends BaseModel {}
export class ConflictingTable extends BaseModel {}
export class ConnectionInfo extends BaseModel {}
export class ConnectionRequest extends BaseModel {}
export class ConnectionSchema extends BaseModel {}
export class EditableMetadataJobResponse extends BaseModel {}
export class FixSQLRequest extends BaseModel {}
export class FixedSQLResponse extends BaseModel {}
export class GeneratedSQLResponse extends BaseModel {}
export class GenericChatRequest extends BaseModel {}
export class GenericChatResponse extends BaseModel {}
export class HealthStatus extends BaseModel {}
export class ListDatabasesResponse extends BaseModel {}
export class ModelInfoResponse extends BaseModel {}
export class MultiQueryRequest extends BaseModel {}
export class MultiQueryResponse extends BaseModel {}
export class NLQueryRequest extends BaseModel {}
export class OptimizationResponse extends BaseModel {}
export class ProviderConfig extends BaseModel {}
export class ProviderStatus extends BaseModel {}
export class QueryRequest extends BaseModel {}
export class QueryResponse extends BaseModel {}
export class QueryRowDeleteRequest extends BaseModel {}
export class QueryRowDeleteResponse extends BaseModel {}
export class QueryRowInsertRequest extends BaseModel {}
export class QueryRowInsertResponse extends BaseModel {}
export class QueryRowUpdateRequest extends BaseModel {}
export class QueryRowUpdateResponse extends BaseModel {}
export class ReadOnlyQueryResult extends BaseModel {}
export class ResultData extends BaseModel {}
export class SQLiteConnection extends BaseModel {}
export class SQLiteQueryHistory extends BaseModel {}
export class SQLiteSavedQuery extends BaseModel {}
export class SchemaConflict extends BaseModel {}
export class Suggestion extends BaseModel {}
export class SwitchDatabaseRequest extends BaseModel {}
export class SwitchDatabaseResponse extends BaseModel {}
export class SyntheticViewSummary extends BaseModel {}
export class UpdateInfo extends BaseModel {}
export class ValidationResult extends BaseModel {}
export class VizSuggestion extends BaseModel {}
