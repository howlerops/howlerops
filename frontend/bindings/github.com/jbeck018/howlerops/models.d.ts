/**
 * TypeScript declarations for Wails v3 generated model bindings
 *
 * These types include factory classes (createFrom) to match Wails v2 patterns
 * used throughout the codebase.
 */

export interface UpdateInfo {
  version: string
  releaseDate: string
  downloadUrl: string
  releaseNotes: string
  isRequired: boolean
}

export interface AITestResponse {
  success: boolean
  message: string
  model?: string
  provider?: string
  error?: string
}

export interface ChatMessage {
  role: string
  content: string
}

// AI Memory Message Payload interface
export interface AIMemoryMessagePayloadData {
  id?: string
  session_id?: string
  role: string
  content: string
  timestamp: string
  metadata?: Record<string, unknown>
}

// AI Memory Session Payload interface
export interface AIMemorySessionPayloadData {
  id: string
  title: string
  messages: (ChatMessage | AIMemoryMessagePayloadData)[]
  createdAt: string
  updatedAt: string
  summary?: string
  summaryTokens?: number
  metadata?: Record<string, unknown>
}

// Factory class for AIMemoryMessagePayload (Wails v2 pattern)
export class AIMemoryMessagePayload implements AIMemoryMessagePayloadData {
  id?: string
  session_id?: string
  role: string
  content: string
  timestamp: string
  metadata?: Record<string, unknown>

  constructor(data?: Partial<AIMemoryMessagePayloadData>)

  static createFrom(data: Partial<AIMemoryMessagePayloadData>): AIMemoryMessagePayload
}

// Factory class for AIMemorySessionPayload (Wails v2 pattern)
export class AIMemorySessionPayload implements AIMemorySessionPayloadData {
  id: string
  title: string
  messages: (ChatMessage | AIMemoryMessagePayload)[]
  createdAt: string
  updatedAt: string
  summary?: string
  summaryTokens?: number
  metadata?: Record<string, unknown>

  constructor(data?: Partial<AIMemorySessionPayloadData>)

  static createFrom(data: Partial<AIMemorySessionPayloadData>): AIMemorySessionPayload
}

export interface AIMemoryRecallResult {
  session: AIMemorySessionPayloadData
  relevance: number
  title?: string
  summary?: string
  content?: string
}

// AI Query Agent Request interface
export interface AIQueryAgentRequestData {
  connectionId: string
  message: string
  sessionId?: string
  mode?: string
  history?: ChatMessage[]
  // Extended fields for full agent functionality
  provider?: string
  model?: string
  connectionIds?: string[]
  schemaContext?: string
  context?: string
  temperature?: number
  maxTokens?: number
  maxRows?: number
  page?: number
  pageSize?: number
}

// Factory class for AIQueryAgentRequest (Wails v2 pattern)
export class AIQueryAgentRequest implements AIQueryAgentRequestData {
  connectionId: string
  message: string
  sessionId?: string
  mode?: string
  history?: ChatMessage[]
  provider?: string
  model?: string
  connectionIds?: string[]
  schemaContext?: string
  context?: string
  temperature?: number
  maxTokens?: number
  maxRows?: number
  page?: number
  pageSize?: number

  constructor(data?: Partial<AIQueryAgentRequestData>)

  static createFrom(data: Partial<AIQueryAgentRequestData>): AIQueryAgentRequest
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

// Re-export from app.d.ts for convenience
export type {
  Connection,
  ConnectionConfig,
  QueryResult,
  QueryRequest,
  TableInfo,
  TableStructure,
  ColumnInfo,
  IndexInfo,
  ForeignKeyInfo,
  EditableQueryMetadata,
} from './app'
