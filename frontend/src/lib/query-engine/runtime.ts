import { api } from '@/lib/api-client'
import { useConnectionStore } from '@/store/connection-store'
import type { QueryResult } from '@/store/query-types'

import { prepareQueryExecutionResult, resolveQueryConnection } from './engine'

interface ExecuteQueryByConnectionIdOptions {
  limit?: number
  offset?: number
  timeout?: number
  isExport?: boolean
}

export async function executeQueryByConnectionId(
  connectionId: string,
  sql: string,
  options: ExecuteQueryByConnectionIdOptions = {}
): Promise<Omit<QueryResult, 'id' | 'timestamp'>> {
  const resolved = resolveQueryConnection(useConnectionStore.getState().connections, connectionId)
  if ('ok' in resolved) {
    throw new Error(resolved.error)
  }
  const activeConnection = resolved

  const limit = options.limit ?? 5000
  const offset = options.offset ?? 0
  const response = await api.queries.execute(
    activeConnection.sessionId,
    sql,
    limit,
    offset,
    options.timeout,
    options.isExport
  )

  const prepared = prepareQueryExecutionResult(
    {
      tabId: '__adhoc__',
      query: sql,
      connectionId: activeConnection.connectionId,
      sessionId: activeConnection.sessionId,
      limit,
      offset,
      timeout: options.timeout,
    },
    response
  )

  if (!prepared.ok) {
    throw new Error(prepared.error)
  }

  return prepared.result
}

export function queryResultRowsToMatrix(result: Pick<QueryResult, 'columns' | 'rows'>): unknown[][] {
  return result.rows.map((row) => result.columns.map((column) => row[column]))
}
