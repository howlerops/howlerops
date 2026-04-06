import { describe, expect, it } from 'vitest'

import type { ApiResponse, QueryResult as ApiQueryResult } from '@/lib/api-client'
import type { DatabaseConnection } from '@/store/connection-store'

import {
  createErrorQueryResult,
  prepareLoadMoreResult,
  prepareQueryExecutionResult,
  resolveQueryConnection,
} from './engine'

function makeConnection(overrides: Partial<DatabaseConnection> = {}): DatabaseConnection {
  return {
    id: 'conn-1',
    sessionId: 'session-1',
    name: 'Primary',
    type: 'postgresql',
    database: 'app',
    isConnected: true,
    ...overrides,
  }
}

function makeResponse(overrides: Partial<ApiQueryResult> = {}): ApiResponse<ApiQueryResult> {
  return {
    success: true,
    data: {
      queryId: 'query-1',
      success: true,
      columns: [{ name: 'id', dataType: 'int' }, { name: 'name', dataType: 'text' }],
      rows: [[1, 'Ada']],
      rowCount: 1,
      stats: { duration: '15ms', affectedRows: 0 },
      warnings: [],
      totalRows: 10,
      pagedRows: 1,
      hasMore: true,
      offset: 0,
      ...overrides,
    },
  }
}

describe('query engine', () => {
  it('resolves an active connection with session id', () => {
    const result = resolveQueryConnection([makeConnection()], 'conn-1')
    expect('ok' in result).toBe(false)
    expect(result).toEqual({ connectionId: 'conn-1', sessionId: 'session-1' })
  })

  it('returns a useful error when connection is missing', () => {
    const result = resolveQueryConnection([], 'conn-1')
    expect('ok' in result && result.ok === false).toBe(true)
    if ('ok' in result && result.ok === false) {
      expect(result.error).toContain('Connection not established')
    }
  })

  it('prepares normalized query results with pagination metadata', () => {
    const prepared = prepareQueryExecutionResult(
      {
        tabId: 'tab-1',
        query: 'select 1',
        connectionId: 'conn-1',
        sessionId: 'session-1',
        limit: 100,
        offset: 0,
      },
      makeResponse()
    )

    expect(prepared.ok).toBe(true)
    if (prepared.ok) {
      expect(prepared.result.columns).toEqual(['id', 'name'])
      expect(prepared.result.rows).toHaveLength(1)
      expect(prepared.result.totalRows).toBe(10)
      expect(prepared.result.hasMore).toBe(true)
      expect(prepared.result.limit).toBe(100)
      expect(prepared.result.executionTime).toBe(15)
    }
  })

  it('creates an error result payload for store consumption', () => {
    const result = createErrorQueryResult(
      { tabId: 'tab-1', query: 'select 1' },
      'boom',
      'conn-1'
    )

    expect(result.error).toBe('boom')
    expect(result.connectionId).toBe('conn-1')
    expect(result.rows).toEqual([])
  })

  it('prepares load-more payloads without mutating the existing result', () => {
    const existing = {
      id: 'result-1',
      tabId: 'tab-1',
      columns: ['id', 'name'],
      rows: [{ id: 1, name: 'Ada', __rowId: 'id:1' }],
      originalRows: { 'id:1': { id: 1, name: 'Ada', __rowId: 'id:1' } },
      rowCount: 1,
      affectedRows: 0,
      executionTime: 1,
      timestamp: new Date(),
      query: 'select * from users',
      connectionId: 'conn-1',
      hasMore: true,
      offset: 0,
      limit: 1,
      editable: {
        enabled: true,
        primaryKeys: ['id'],
        columns: [],
      },
    }

    const prepared = prepareLoadMoreResult(
      existing,
      makeResponse({ rows: [[2, 'Grace']], pagedRows: 1, totalRows: 2, hasMore: false, offset: 1 }),
      1
    )

    expect('ok' in prepared).toBe(false)
    expect(prepared.rows).toHaveLength(1)
    expect(prepared.offset).toBe(1)
    expect(prepared.hasMore).toBe(false)
  })
})
