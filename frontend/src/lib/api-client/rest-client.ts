/**
 * REST API Client Implementation
 *
 * HTTP-based client for web deployment mode.
 * Uses the backend REST API endpoints.
 */

import { getApiBaseUrl } from '../platform'
import type {
  ApiClient,
  ApiResponse,
  CancelResult,
  ColumnInfo,
  ConnectionInfo,
  CreateConnectionRequest,
  DeleteRowsRequest,
  DeleteRowsResult,
  EditableMetadata,
  ExplainResult,
  InsertRowRequest,
  InsertRowResult,
  ListDatabasesResult,
  PaginatedResponse,
  QueryResult,
  SaveConnectionRequest,
  SchemaInfo,
  SwitchDatabaseResult,
  TableInfo,
  TableStructureResult,
  TestConnectionRequest,
  TestConnectionResult,
  UpdateRowRequest,
} from './types'

/**
 * Make an HTTP request to the REST API
 */
async function request<T>(
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  body?: unknown
): Promise<T> {
  const baseUrl = getApiBaseUrl()
  const url = `${baseUrl}${path}`

  const options: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    credentials: 'include', // Include cookies for auth
  }

  if (body !== undefined) {
    options.body = JSON.stringify(body)
  }

  const response = await fetch(url, options)

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`HTTP ${response.status}: ${errorText}`)
  }

  return response.json() as Promise<T>
}

/**
 * REST implementation of the unified API client.
 * Uses HTTP requests to the backend REST API.
 */
export const restApiClient: ApiClient = {
  connections: {
    list: async (
      page: number = 1,
      pageSize: number = 50,
      _filter?: string
    ): Promise<PaginatedResponse<ConnectionInfo>> => {
      try {
        const result = await request<{ connections: ConnectionInfo[]; total?: number }>(
          'GET',
          `/api/connections?page=${page}&pageSize=${pageSize}`
        )

        return {
          data: result.connections || [],
          total: result.total || result.connections?.length || 0,
          page,
          pageSize,
        }
      } catch (error) {
        console.error('Failed to list connections:', error)
        return {
          data: [],
          total: 0,
          page,
          pageSize,
        }
      }
    },

    create: async (data: CreateConnectionRequest): Promise<ApiResponse<unknown>> => {
      try {
        const result = await request<unknown>('POST', '/api/connections', {
          type: data.type || 'postgresql',
          host: data.host || 'localhost',
          port: data.port || 5432,
          database: data.database || '',
          username: data.username || '',
          password: data.password || '',
          ssl_mode: data.ssl_mode || 'prefer',
          connection_timeout: data.connection_timeout || 30,
          parameters: data.parameters || {},
          name: data.name || '',
        })

        return {
          data: result,
          success: true,
          message: 'Connection created successfully',
        }
      } catch (error) {
        return {
          data: null,
          success: false,
          message: error instanceof Error ? error.message : 'Failed to create connection',
        }
      }
    },

    save: async (data: SaveConnectionRequest): Promise<ApiResponse<null>> => {
      try {
        await request('PUT', `/api/connections/${data.id}`, {
          type: data.type || 'postgresql',
          host: data.host || 'localhost',
          port: data.port || 5432,
          database: data.database || '',
          username: data.username || '',
          password: data.password || '',
          ssl_mode: data.ssl_mode || 'prefer',
          connection_timeout: data.connection_timeout || 30,
          parameters: data.parameters || {},
          name: data.name || '',
        })

        return {
          data: null,
          success: true,
          message: 'Connection saved successfully',
        }
      } catch (error) {
        return {
          data: null,
          success: false,
          message: error instanceof Error ? error.message : 'Failed to save connection',
        }
      }
    },

    test: async (data: TestConnectionRequest): Promise<ApiResponse<TestConnectionResult>> => {
      try {
        const result = await request<{ success: boolean; version?: string; responseTime?: number }>(
          'POST',
          '/api/connections/test',
          {
            type: data.type || 'postgresql',
            host: data.host || 'localhost',
            port: data.port || 5432,
            database: data.database || '',
            username: data.username || '',
            password: data.password || '',
            ssl_mode: data.ssl_mode || 'prefer',
            connection_timeout: data.connection_timeout || 30,
            parameters: data.parameters || {},
          }
        )

        return {
          data: {
            success: result.success,
            responseTime: result.responseTime || 0,
            version: result.version || '',
            serverInfo: {},
          },
          success: result.success,
          message: result.success ? 'Connection test successful' : 'Connection test failed',
        }
      } catch (error) {
        return {
          data: {
            success: false,
            responseTime: 0,
            version: '',
            serverInfo: {},
          },
          success: false,
          message: error instanceof Error ? error.message : 'Connection test failed',
        }
      }
    },

    remove: async (connectionId: string): Promise<ApiResponse<null>> => {
      try {
        await request('DELETE', `/api/connections/${connectionId}`)

        return {
          data: null,
          success: true,
          message: 'Connection removed successfully',
        }
      } catch (error) {
        return {
          data: null,
          success: false,
          message: error instanceof Error ? error.message : 'Failed to remove connection',
        }
      }
    },

    listDatabases: async (connectionId: string): Promise<ListDatabasesResult> => {
      try {
        const result = await request<{ databases: string[] }>(
          'GET',
          `/api/connections/${connectionId}/databases`
        )

        return {
          success: true,
          databases: result.databases || [],
        }
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to list databases',
          databases: [],
        }
      }
    },

    switchDatabase: async (connectionId: string, database: string): Promise<SwitchDatabaseResult> => {
      try {
        const result = await request<{ success: boolean; reconnected?: boolean }>(
          'POST',
          `/api/connections/${connectionId}/switch-database`,
          { database }
        )

        return {
          success: result.success,
          database,
          reconnected: result.reconnected || false,
        }
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to switch database',
          database,
          reconnected: false,
        }
      }
    },
  },

  queries: {
    execute: async (
      connectionId: string,
      sql: string,
      limit: number = 5000,
      offset: number = 0,
      timeout: number = 30,
      isExport: boolean = false
    ): Promise<ApiResponse<QueryResult>> => {
      try {
        const result = await request<{
          columns?: Array<{ name: string; dataType: string }>
          rows?: unknown[][]
          rowCount?: number
          duration?: number
          affected?: number
          error?: string
          editable?: EditableMetadata
          totalRows?: number
          hasMore?: boolean
        }>('POST', '/api/queries/execute', {
          connectionId,
          query: sql,
          limit,
          offset,
          timeout,
          isExport,
        })

        const hasError = Boolean(result.error)

        return {
          data: {
            queryId: `query-${Date.now()}`,
            success: !hasError,
            columns: result.columns || [],
            rows: result.rows || [],
            rowCount: result.rowCount || 0,
            stats: {
              duration: result.duration,
              affectedRows: result.affected,
            },
            warnings: [],
            editable: result.editable || null,
            totalRows: result.totalRows,
            hasMore: result.hasMore,
            offset,
          },
          success: !hasError,
          message: hasError ? result.error : undefined,
        }
      } catch (error) {
        return {
          data: {
            queryId: `query-${Date.now()}`,
            success: false,
            columns: [],
            rows: [],
            rowCount: 0,
            stats: {},
            warnings: [],
          },
          success: false,
          message: error instanceof Error ? error.message : 'Query execution failed',
        }
      }
    },

    getEditableMetadata: async (jobId: string): Promise<ApiResponse<EditableMetadata | null>> => {
      try {
        const result = await request<EditableMetadata>('GET', `/api/queries/${jobId}/editable`)

        return {
          data: result,
          success: true,
          message: 'Editable metadata retrieved successfully',
        }
      } catch (error) {
        return {
          data: null,
          success: false,
          message: error instanceof Error ? error.message : 'Failed to fetch editable metadata',
        }
      }
    },

    updateRow: async (payload: UpdateRowRequest): Promise<ApiResponse<null>> => {
      try {
        const result = await request<{ success: boolean; message?: string }>(
          'POST',
          '/api/queries/update-row',
          payload
        )

        return {
          data: null,
          success: result.success,
          message: result.message || (result.success ? 'Row updated successfully' : 'Failed to update row'),
        }
      } catch (error) {
        return {
          data: null,
          success: false,
          message: error instanceof Error ? error.message : 'Failed to save changes',
        }
      }
    },

    insertRow: async (payload: InsertRowRequest): Promise<InsertRowResult> => {
      try {
        const result = await request<{ success: boolean; message?: string; row?: unknown[] }>(
          'POST',
          '/api/queries/insert-row',
          payload
        )

        return {
          success: result.success,
          message: result.message,
          row: result.row,
        }
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to insert row',
        }
      }
    },

    deleteRows: async (payload: DeleteRowsRequest): Promise<DeleteRowsResult> => {
      try {
        const result = await request<{ success: boolean; message?: string; deleted?: number }>(
          'POST',
          '/api/queries/delete-rows',
          payload
        )

        return {
          success: result.success,
          message: result.message,
          deleted: result.deleted,
        }
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to delete rows',
        }
      }
    },

    explain: async (connectionId: string, query: string): Promise<ApiResponse<ExplainResult>> => {
      try {
        const result = await request<{ plan: string }>('POST', '/api/queries/explain', {
          connectionId,
          query,
        })

        return {
          data: {
            plan: result.plan,
            format: 'text',
            estimatedStats: {},
            warnings: [],
          },
          success: true,
          message: 'Query explanation retrieved successfully',
        }
      } catch (error) {
        return {
          data: {
            plan: '',
            format: '',
            estimatedStats: {},
            warnings: [],
          },
          success: false,
          message: error instanceof Error ? error.message : 'Failed to explain query',
        }
      }
    },

    cancel: async (streamId: string): Promise<CancelResult> => {
      try {
        await request('POST', `/api/queries/${streamId}/cancel`)

        return {
          success: true,
          message: 'Query cancelled successfully',
          wasRunning: true,
        }
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to cancel query',
          wasRunning: false,
        }
      }
    },
  },

  schema: {
    databases: async (connectionId: string): Promise<ApiResponse<SchemaInfo[]>> => {
      try {
        const result = await request<{ schemas: string[] }>(
          'GET',
          `/api/connections/${connectionId}/schemas`
        )

        return {
          data: (result.schemas || []).map((name) => ({
            name,
            owner: '',
            createdAt: '',
            tableCount: 0,
            sizeBytes: 0,
            metadata: {},
          })),
          success: true,
          message: 'Schemas retrieved successfully',
        }
      } catch (error) {
        return {
          data: [],
          success: false,
          message: error instanceof Error ? error.message : 'Failed to fetch schemas',
        }
      }
    },

    tables: async (connectionId: string, schemaName?: string): Promise<ApiResponse<TableInfo[]>> => {
      try {
        const query = schemaName ? `?schema=${encodeURIComponent(schemaName)}` : ''
        const result = await request<{
          tables: Array<{
            name: string
            schema: string
            type: string
            comment?: string
            rowCount?: number
            sizeBytes?: number
          }>
        }>('GET', `/api/connections/${connectionId}/tables${query}`)

        return {
          data: (result.tables || []).map((table) => ({
            name: table.name,
            schema: table.schema,
            type: table.type,
            comment: table.comment || '',
            createdAt: '',
            updatedAt: '',
            rowCount: table.rowCount || 0,
            sizeBytes: table.sizeBytes || 0,
            owner: '',
            metadata: {},
          })),
          success: true,
          message: 'Tables retrieved successfully',
        }
      } catch (error) {
        return {
          data: [],
          success: false,
          message: error instanceof Error ? error.message : 'Failed to fetch tables',
        }
      }
    },

    columns: async (
      connectionId: string,
      schemaName: string,
      tableName: string
    ): Promise<TableStructureResult> => {
      try {
        const result = await request<{
          columns?: Array<{
            name: string
            data_type: string
            nullable: boolean
            default_value?: string
            primary_key: boolean
            unique: boolean
            ordinal_position: number
            character_maximum_length?: number
            numeric_precision?: number
            numeric_scale?: number
          }>
          indexes?: unknown[]
          foreign_keys?: unknown[]
          triggers?: unknown[]
          statistics?: Record<string, unknown>
        }>(
          'GET',
          `/api/connections/${connectionId}/tables/${encodeURIComponent(schemaName)}.${encodeURIComponent(tableName)}/structure`
        )

        const columns: ColumnInfo[] = (result.columns || []).map((col) => ({
          name: col.name,
          dataType: col.data_type,
          nullable: col.nullable,
          defaultValue: col.default_value,
          primaryKey: col.primary_key,
          unique: col.unique,
          indexed: false,
          comment: '',
          ordinalPosition: col.ordinal_position,
          characterMaximumLength: col.character_maximum_length,
          numericPrecision: col.numeric_precision,
          numericScale: col.numeric_scale,
          metadata: {},
        }))

        return {
          data: columns,
          table: {
            name: tableName,
            schema: schemaName,
            type: 'TABLE',
            comment: '',
            createdAt: '',
            updatedAt: '',
            rowCount: 0,
            sizeBytes: 0,
            owner: '',
            metadata: {},
          },
          indexes: result.indexes || [],
          foreignKeys: result.foreign_keys || [],
          triggers: result.triggers || [],
          statistics: result.statistics || {},
          success: true,
          message: 'Table structure retrieved successfully',
        }
      } catch (error) {
        return {
          data: [],
          table: null,
          indexes: [],
          foreignKeys: [],
          triggers: [],
          statistics: {},
          success: false,
          message: error instanceof Error ? error.message : 'Failed to fetch table structure',
        }
      }
    },
  },
}
