import type { ApiResponse, EditableMetadata, QueryResult as ApiQueryResult } from '@/lib/api-client'
import type { DatabaseConnection } from '@/store/connection-store'
import type { QueryEditableMetadata, QueryResult, QueryResultRow } from '@/store/query-types'
import { normaliseRows, parseDurationMs, transformEditableMetadata } from '@/store/query-utils'

export interface QueryEngineTransport {
  execute: (
    sessionId: string,
    sql: string,
    limit?: number,
    offset?: number,
    timeout?: number
  ) => Promise<ApiResponse<ApiQueryResult>>
}

export interface ResolvedQueryConnection {
  connectionId: string
  sessionId: string
}

export interface QueryExecutionInput {
  query: string
  connectionId: string
  sessionId: string
  tabId: string
  limit: number
  offset: number
  timeout?: number
}

export interface QueryExecutionFailure {
  ok: false
  error: string
  connectionId?: string
}

export interface QueryExecutionSuccess {
  ok: true
  result: Omit<QueryResult, 'id' | 'timestamp'>
  editable: QueryEditableMetadata | null
}

export type QueryExecutionOutcome = QueryExecutionFailure | QueryExecutionSuccess

export interface LoadMoreSuccess {
  rows: QueryResultRow[]
  originalRows: Record<string, QueryResultRow>
  totalRows?: number
  pagedRows?: number
  hasMore?: boolean
  offset: number
}

export function resolveQueryConnection(
  connections: DatabaseConnection[],
  connectionId?: string | null
): ResolvedQueryConnection | QueryExecutionFailure {
  if (!connectionId) {
    return {
      ok: false,
      error: 'No connection selected for this tab',
    }
  }

  const connection = connections.find((conn) => conn.id === connectionId)
  if (!connection?.sessionId) {
    return {
      ok: false,
      error: 'Connection not established. Please connect to the database first.',
      connectionId,
    }
  }

  return {
    connectionId,
    sessionId: connection.sessionId,
  }
}

function extractColumns(columns: unknown[]): string[] {
  return columns.map((column) =>
    typeof column === 'string' ? column : (column as { name?: string }).name ?? ''
  ).filter(Boolean)
}

export function createErrorQueryResult(
  input: Pick<QueryExecutionInput, 'tabId' | 'query'>,
  error: string,
  connectionId?: string
): Omit<QueryResult, 'id' | 'timestamp'> {
  return {
    tabId: input.tabId,
    columns: [],
    rows: [],
    originalRows: {},
    rowCount: 0,
    affectedRows: 0,
    executionTime: 0,
    error,
    editable: null,
    query: input.query,
    connectionId,
  }
}

export function prepareQueryExecutionResult(
  input: QueryExecutionInput,
  response: ApiResponse<ApiQueryResult>
): QueryExecutionOutcome {
  if (!response.success || !response.data) {
    return {
      ok: false,
      error: response.message || 'Query execution failed',
      connectionId: input.connectionId,
    }
  }

  const {
    columns: rawColumns = [],
    rows = [],
    rowCount = 0,
    stats = {},
    editable: rawEditable = null,
    totalRows,
    pagedRows,
    hasMore,
    offset,
    connectionsUsed,
  } = response.data

  const columns = extractColumns(rawColumns)
  const statsRecord = (stats ?? {}) as Record<string, unknown>
  const affectedRows =
    typeof statsRecord.affectedRows === 'number'
      ? statsRecord.affectedRows
      : typeof statsRecord.affected_rows === 'number'
        ? statsRecord.affected_rows
        : 0
  const durationValue =
    typeof statsRecord.duration === 'string'
      ? statsRecord.duration
      : undefined

  const editableMetadata = transformEditableMetadata(rawEditable as EditableMetadata | null)
  const { rows: normalisedRows, originalRows } = normaliseRows(columns, rows, editableMetadata)

  return {
    ok: true,
    editable: editableMetadata,
    result: {
      tabId: input.tabId,
      columns,
      rows: normalisedRows,
      originalRows,
      rowCount: rowCount || normalisedRows.length,
      affectedRows,
      executionTime: parseDurationMs(durationValue),
      error: undefined,
      editable: editableMetadata,
      query: input.query,
      connectionId: input.connectionId,
      totalRows,
      pagedRows,
      hasMore,
      offset: typeof offset === 'number' ? offset : input.offset,
      limit: input.limit,
      connectionsUsed,
    },
  }
}

export function prepareLoadMoreResult(
  existingResult: QueryResult,
  response: ApiResponse<ApiQueryResult>,
  nextOffset: number
): QueryExecutionFailure | LoadMoreSuccess {
  if (!response.success || !response.data) {
    return {
      ok: false,
      error: response.message || 'Failed to load more rows',
      connectionId: existingResult.connectionId,
    }
  }

  const { rows = [], totalRows, pagedRows, hasMore, offset } = response.data
  const { rows: normalisedRows, originalRows } = normaliseRows(
    existingResult.columns,
    rows,
    existingResult.editable
  )

  return {
    rows: normalisedRows,
    originalRows,
    totalRows,
    pagedRows,
    hasMore,
    offset: typeof offset === 'number' ? offset : nextOffset,
  }
}
