/**
 * Wails API Client Implementation
 *
 * Adapts the existing wailsEndpoints for the unified ApiClient interface.
 * Handles type transformations between Wails backend responses and the interface.
 */

import { wailsEndpoints } from '../wails-api'
import type {
  ApiClient,
  ApiResponse,
  CancelResult,
  DeleteRowsRequest,
  DeleteRowsResult,
  EditableMetadata,
  ExplainResult,
  InsertRowRequest,
  InsertRowResult,
  ListDatabasesResult,
  PaginatedResponse,
  ConnectionInfo,
  QueryColumn,
  QueryResult,
  SaveConnectionRequest,
  SwitchDatabaseResult,
  TestConnectionResult,
  UpdateRowRequest,
  CreateConnectionRequest,
  TestConnectionRequest,
  SchemaInfo,
  TableInfo,
  TableStructureResult,
} from './types'

/**
 * Convert string column names to QueryColumn objects.
 * The Go backend returns columns as strings, but the interface expects objects.
 */
function transformColumns(columns: unknown[]): QueryColumn[] {
  if (!columns || !Array.isArray(columns)) return []

  return columns.map((col) => {
    // If already a QueryColumn object, return as-is
    if (typeof col === 'object' && col !== null && 'name' in col) {
      return col as QueryColumn
    }
    // Otherwise, convert string to QueryColumn
    return {
      name: String(col),
      dataType: 'unknown',
      nullable: true,
    }
  })
}

/**
 * Wails implementation of the unified API client.
 * Adapts wailsEndpoints responses to match the ApiClient interface.
 */
export const wailsApiClient: ApiClient = {
  connections: {
    list: async (
      page?: number,
      pageSize?: number,
      filter?: string
    ): Promise<PaginatedResponse<ConnectionInfo>> => {
      const result = await wailsEndpoints.connections.list(page, pageSize, filter)
      return {
        data: result.data || [],
        total: result.total || 0,
        page: result.page || 1,
        pageSize: result.pageSize || 50,
      }
    },

    create: async (data: CreateConnectionRequest): Promise<ApiResponse<unknown>> => {
      const result = await wailsEndpoints.connections.create(data)
      return {
        data: result.data,
        success: result.success,
        message: result.message,
      }
    },

    save: async (data: SaveConnectionRequest): Promise<ApiResponse<null>> => {
      const result = await wailsEndpoints.connections.save(data)
      return {
        data: null,
        success: result.success,
        message: result.message,
      }
    },

    test: async (data: TestConnectionRequest): Promise<ApiResponse<TestConnectionResult>> => {
      const result = await wailsEndpoints.connections.test(data)
      return {
        data: result.data || {
          success: result.success,
          responseTime: 0,
          version: '',
          serverInfo: {},
        },
        success: result.success,
        message: result.message,
      }
    },

    remove: async (connectionId: string): Promise<ApiResponse<null>> => {
      const result = await wailsEndpoints.connections.remove(connectionId)
      return {
        data: null,
        success: result.success,
        message: result.message,
      }
    },

    listDatabases: async (connectionId: string): Promise<ListDatabasesResult> => {
      const result = await wailsEndpoints.connections.listDatabases(connectionId)
      return {
        success: result.success,
        message: result.message,
        databases: result.databases || [],
      }
    },

    switchDatabase: async (connectionId: string, database: string): Promise<SwitchDatabaseResult> => {
      const result = await wailsEndpoints.connections.switchDatabase(connectionId, database)
      return {
        success: result.success,
        message: result.message,
        database: result.database || database,
        reconnected: result.reconnected || false,
      }
    },
  },

  queries: {
    execute: async (
      connectionId: string,
      sql: string,
      limit?: number,
      offset?: number,
      timeout?: number
    ): Promise<ApiResponse<QueryResult>> => {
      const result = await wailsEndpoints.queries.execute(connectionId, sql, limit, offset, timeout)
      const data = result.data || {}

      return {
        data: {
          queryId: data.queryId || `query-${Date.now()}`,
          success: data.success ?? result.success,
          columns: transformColumns(data.columns || []),
          rows: data.rows || [],
          rowCount: data.rowCount || 0,
          stats: data.stats || {},
          warnings: data.warnings || [],
          editable: data.editable || null,
          totalRows: data.totalRows,
          pagedRows: data.pagedRows,
          hasMore: data.hasMore,
          offset: data.offset,
          connectionsUsed: data.connectionsUsed,
        },
        success: result.success,
        message: result.message,
      }
    },

    getEditableMetadata: async (jobId: string): Promise<ApiResponse<EditableMetadata | null>> => {
      const result = await wailsEndpoints.queries.getEditableMetadata(jobId)
      return {
        data: result.data || null,
        success: result.success,
        message: result.message,
      }
    },

    updateRow: async (payload: UpdateRowRequest): Promise<ApiResponse<null>> => {
      const result = await wailsEndpoints.queries.updateRow(payload)
      return {
        data: null,
        success: result.success,
        message: result.message,
      }
    },

    insertRow: async (payload: InsertRowRequest): Promise<InsertRowResult> => {
      const result = await wailsEndpoints.queries.insertRow(payload)
      // Transform row from Record<string, unknown> to unknown[] if needed
      let row: unknown[] | undefined
      if (result.row) {
        if (Array.isArray(result.row)) {
          row = result.row
        } else if (typeof result.row === 'object') {
          // Convert object to array of values
          row = Object.values(result.row)
        }
      }
      return {
        success: result.success,
        message: result.message,
        row,
      }
    },

    deleteRows: async (payload: DeleteRowsRequest): Promise<DeleteRowsResult> => {
      const result = await wailsEndpoints.queries.deleteRows(payload)
      return {
        success: result.success,
        message: result.message,
        deleted: result.deleted,
      }
    },

    explain: async (connectionId: string, query: string): Promise<ApiResponse<ExplainResult>> => {
      const result = await wailsEndpoints.queries.explain(connectionId, query)
      return {
        data: result.data || {
          plan: '',
          format: '',
          estimatedStats: {},
          warnings: [],
        },
        success: result.success,
        message: result.message,
      }
    },

    cancel: async (streamId: string): Promise<CancelResult> => {
      const result = await wailsEndpoints.queries.cancel(streamId)
      return {
        success: result.success,
        message: result.message,
        wasRunning: result.wasRunning || false,
      }
    },
  },

  schema: {
    databases: async (connectionId: string): Promise<ApiResponse<SchemaInfo[]>> => {
      const result = await wailsEndpoints.schema.databases(connectionId)
      return {
        data: result.data || [],
        success: result.success,
        message: result.message,
      }
    },

    tables: async (connectionId: string, schemaName?: string): Promise<ApiResponse<TableInfo[]>> => {
      const result = await wailsEndpoints.schema.tables(connectionId, schemaName)
      return {
        data: result.data || [],
        success: result.success,
        message: result.message,
      }
    },

    columns: async (
      connectionId: string,
      schemaName: string,
      tableName: string
    ): Promise<TableStructureResult> => {
      const result = await wailsEndpoints.schema.columns(connectionId, schemaName, tableName)
      return {
        data: result.data || [],
        table: result.table || null,
        indexes: result.indexes || [],
        foreignKeys: result.foreignKeys || [],
        triggers: result.triggers || [],
        statistics: result.statistics || {},
        success: result.success,
        message: result.message,
      }
    },
  },
}
