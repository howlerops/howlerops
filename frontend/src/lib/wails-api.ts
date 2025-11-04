import * as App from '../../wailsjs/go/main/App'
import { waitForWails, isWailsReady } from './wails-runtime'

// Wails-based API client for desktop application
export class WailsApiClient {
  // Schema introspection methods
  async getSchemas(connectionId: string) {
    try {
      await waitForWails()
      
      if (!isWailsReady()) {
        throw new Error('Wails runtime not available')
      }
      
      const schemas = await App.GetSchemas(connectionId)

      return {
        data: schemas.map(schemaName => ({
          name: schemaName,
          owner: '',
          createdAt: '',
          tableCount: 0,
          sizeBytes: 0,
          metadata: {}
        })),
        success: true,
        message: 'Schemas retrieved successfully'
      }
    } catch (error) {
      return {
        data: [],
        success: false,
        message: error instanceof Error ? error.message : 'Failed to fetch schemas'
      }
    }
  }

  async getTables(connectionId: string, schemaName?: string) {
    try {
      await waitForWails()
      
      if (!isWailsReady()) {
        throw new Error('Wails runtime not available')
      }
      
      const tables = await App.GetTables(connectionId, schemaName || '')

      return {
        data: tables.map(table => ({
          name: table.name,
          schema: table.schema,
          type: table.type,
          comment: table.comment,
          createdAt: '',
          updatedAt: '',
          rowCount: table.rowCount,
          sizeBytes: table.sizeBytes,
          owner: '',
          metadata: {}
        })),
        success: true,
        message: 'Tables retrieved successfully'
      }
    } catch (error) {
      return {
        data: [],
        success: false,
        message: error instanceof Error ? error.message : 'Failed to fetch tables'
      }
    }
  }

  async getTableStructure(connectionId: string, schemaName: string, tableName: string) {
    try {
      await waitForWails()
      
      if (!isWailsReady()) {
        throw new Error('Wails runtime not available')
      }
      
      const structure = await App.GetTableStructure(connectionId, schemaName, tableName)

      return {
        data: structure.columns?.map(column => ({
          name: column.name,
          dataType: column.data_type,
          nullable: column.nullable,
          defaultValue: column.default_value,
          primaryKey: column.primary_key,
          unique: column.unique,
          indexed: false, // Not provided by the backend structure
          comment: '',
          ordinalPosition: column.ordinal_position,
          characterMaximumLength: column.character_maximum_length,
          numericPrecision: column.numeric_precision,
          numericScale: column.numeric_scale,
          metadata: {}
        })) || [],
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
          metadata: {}
        },
        indexes: structure.indexes || [],
        foreignKeys: structure.foreign_keys || [],
        triggers: structure.triggers || [],
        statistics: structure.statistics || {},
        success: true,
        message: 'Table structure retrieved successfully'
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
        message: error instanceof Error ? error.message : 'Failed to fetch table structure'
      }
    }
  }

  // Connection methods
  async createConnection(data: unknown) {
    try {
      const request = (data || {}) as {
        id?: string
        type?: string
        host?: string
        port?: number
        database?: string
        username?: string
        password?: string
        ssl_mode?: string
        connection_timeout?: number
        parameters?: Record<string, string>
        name?: string
      }

      const parameters: Record<string, string> = {
        ...(request.parameters ?? {})
      }

      const aliasSource = parameters.alias || request.name
      if (aliasSource && aliasSource.trim().length > 0) {
        const alias = aliasSource.trim()
        parameters.alias = alias

        const slug = alias.replace(/[^\w-]/g, '-')
        if (slug && slug !== alias && !parameters.alias_slug) {
          parameters.alias_slug = slug
        }

        const lower = alias.toLowerCase()
        if (lower !== alias && !parameters.alias_lower) {
          parameters.alias_lower = lower
        }
      }

      const result = await App.CreateConnection({
        id: request.id || '', // Pass stored connection ID for reconnecting
        type: request.type || 'postgresql',
        host: request.host || 'localhost',
        port: request.port || 5432,
        database: request.database || '',
        username: request.username || '',
        password: request.password || '',
        sslMode: request.ssl_mode || 'prefer',  // Default to 'prefer' for better security
        connectionTimeout: request.connection_timeout || 30,
        parameters
      })

      return {
        data: result,
        success: true,
        message: 'Connection created successfully'
      }
    } catch (error) {
      return {
        data: null,
        success: false,
        message: error instanceof Error ? error.message : 'Failed to create connection'
      }
    }
  }

  async saveConnection(data: unknown) {
    try {
      const request = (data || {}) as {
        id?: string
        name?: string
        type?: string
        host?: string
        port?: number
        database?: string
        username?: string
        password?: string
        ssl_mode?: string
        connection_timeout?: number
        parameters?: Record<string, string>
      }

      await App.SaveConnection({
        id: request.id || '',
        name: request.name || '',
        type: request.type || 'postgresql',
        host: request.host || 'localhost',
        port: request.port || 5432,
        database: request.database || '',
        username: request.username || '',
        password: request.password || '',
        sslMode: request.ssl_mode || 'prefer',  // Default to 'prefer' for better security
        connectionTimeout: request.connection_timeout || 30,
        parameters: request.parameters || {}
      })

      return {
        data: null,
        success: true,
        message: 'Connection saved successfully'
      }
    } catch (error) {
      return {
        data: null,
        success: false,
        message: error instanceof Error ? error.message : 'Failed to save connection'
      }
    }
  }

  async testConnection(data: unknown) {
    try {
      const connectionData = data as {
        type?: string
        host?: string
        port?: number
        database?: string
        username?: string
        password?: string
        ssl_mode?: string
        connection_timeout?: number
        parameters?: Record<string, string>
      }

      await App.TestConnection({
        type: connectionData.type || 'postgresql',
        host: connectionData.host || 'localhost',
        port: connectionData.port || 5432,
        database: connectionData.database || '',
        username: connectionData.username || '',
        password: connectionData.password || '',
        sslMode: connectionData.ssl_mode || 'prefer',  // Default to 'prefer' for better security
        connectionTimeout: connectionData.connection_timeout || 30,
        parameters: connectionData.parameters || {}
      })

      return {
        data: {
          success: true,
          responseTime: 0,
          version: '',
          serverInfo: {}
        },
        success: true,
        message: 'Connection test successful'
      }
    } catch (error) {
      return {
        data: {
          success: false,
          responseTime: 0,
          version: '',
          serverInfo: {}
        },
        success: false,
        message: error instanceof Error ? error.message : 'Connection test failed'
      }
    }
  }

  async listConnections() {
    try {
      const connections = await App.ListConnections()

      return {
        data: connections.map(id => ({
          id,
          name: `Connection ${id}`,
          description: '',
          type: 'postgresql',
          host: '',
          port: 5432,
          database: '',
          username: '',
          active: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          createdBy: '',
          tags: {}
        })),
        total: connections.length,
        page: 1,
        pageSize: 50
      }
    } catch {  
      return {
        data: [],
        total: 0,
        page: 1,
        pageSize: 50
      }
    }
  }

  async removeConnection(connectionId: string) {
    try {
      await App.RemoveConnection(connectionId)
      return {
        success: true,
        message: 'Connection removed successfully'
      }
    } catch (error) {
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Failed to remove connection'
      }
    }
  }

  async executeQuery(connectionId: string, sql: string, options?: { limit?: number; timeout?: number }) {
    try {
      // Check if query contains @ syntax for multi-database queries
      if (shouldUseMultiDatabasePath(sql)) {
        return await this.executeMultiDatabaseQuery(sql, options)
      }

      // Load defaults from preferences
      const { PreferenceRepository, _PreferenceCategory } = await import('@/lib/storage/repositories/preference-repository')
      const pref = new PreferenceRepository()
      const timeoutPref = await pref.getUserPreference('local-user', 'queryTimeoutSeconds')
      const limitPref = await pref.getUserPreference('local-user', 'defaultResultLimit')
      const timeoutSeconds = typeof options?.timeout === 'number' ? options.timeout : (typeof timeoutPref?.value === 'number' ? timeoutPref.value : 30)
      const limitRows = typeof options?.limit === 'number' ? options.limit : (typeof limitPref?.value === 'number' ? limitPref.value : 1000)

      // Note: Timeout is supported by backend; TS bindings may lag until regenerated
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const req: any = {
        connectionId,
        query: sql,
        limit: limitRows,
        timeout: timeoutSeconds
      }

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const result = await (App.ExecuteQuery as any)(req)

      const hasError = typeof result.error === 'string' ? result.error.length > 0 : Boolean(result.error)
      const success = !hasError

      return {
        data: {
          queryId: `query-${Date.now()}`,
          success,
          columns: result.columns || [],
          rows: result.rows || [],
          rowCount: result.rowCount || 0,
          stats: {
            duration: result.duration,
            affectedRows: result.affected
          },
          warnings: [],
          editable: result.editable || null
        },
        success,
        message: hasError ? result.error : undefined
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
          warnings: []
        },
        success: false,
        message: error instanceof Error ? error.message : 'Query execution failed'
      }
    }
  }

  async getEditableMetadata(jobId: string) {
    try {
      const result = await App.GetEditableMetadata(jobId)
      return {
        data: result,
        success: true,
        message: 'Editable metadata retrieved successfully'
      }
    } catch (error) {
      return {
        data: null,
        success: false,
        message: error instanceof Error ? error.message : 'Failed to fetch editable metadata'
      }
    }
  }

  async executeMultiDatabaseQuery(sql: string, options?: { limit?: number; timeout?: number }) {
    try {
      // Load timeout default if not provided
      const { PreferenceRepository } = await import('@/lib/storage/repositories/preference-repository')
      const pref = new PreferenceRepository()
      const timeoutPref = await pref.getUserPreference('local-user', 'queryTimeoutSeconds')
      const timeoutSeconds = typeof options?.timeout === 'number' ? options.timeout : (typeof timeoutPref?.value === 'number' ? timeoutPref.value : 30)

      const result = await App.ExecuteMultiDatabaseQuery({
        query: sql,
        timeout: timeoutSeconds, // seconds
        strategy: 'federated',
        limit: options?.limit || 1000
      })

      const hasError = typeof result.error === 'string' ? result.error.length > 0 : Boolean(result.error)
      const success = !hasError

      return {
        data: {
          queryId: `query-${Date.now()}`,
          success,
          columns: result.columns || [],
          rows: result.rows || [],
          rowCount: result.rowCount || 0,
          stats: {
            duration: result.duration,
            affectedRows: 0
          },
          warnings: [],
          editable: null,
          connectionsUsed: result.connectionsUsed || []
        },
        success,
        message: hasError ? result.error : undefined
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
          warnings: []
        },
        success: false,
        message: error instanceof Error ? error.message : 'Multi-database query execution failed'
      }
    }
  }

  async updateQueryRow(payload: unknown) {
    try {
      const requestPayload = payload as {
        connectionId: string
        query: string
        columns: string[]
        schema?: string
        table?: string
        primaryKey: Record<string, unknown>
        values: Record<string, unknown>
      }
      const response = await App.UpdateQueryRow(requestPayload)
      return {
        success: response.success,
        message: response.message
      }
    } catch (error) {
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Failed to save changes'
      }
    }
  }
}

const multiDbPattern = /@[\w-]+\./i
const singleQuotedString = /'(?:''|[^'])*'/g
const doubleQuotedString = /"(?:\\"|[^"])*"/g
const backtickQuotedString = /`(?:``|[^`])*`/g

function stripQuotedLiterals(sql: string): string {
  return sql
    .replace(singleQuotedString, "''")
    .replace(doubleQuotedString, '""')
    .replace(backtickQuotedString, '``')
}

function shouldUseMultiDatabasePath(sql: string): boolean {
  if (!sql.includes('@')) {
    return false
  }

  const sanitized = stripQuotedLiterals(sql)
  return multiDbPattern.test(sanitized)
}

// Create singleton instance
export const wailsApiClient = new WailsApiClient()

// Updated endpoints using Wails API
export const wailsEndpoints = {
  // Connection endpoints
  connections: {
    list: async (
      _page: number = 1,  
      _pageSize: number = 50,  
      _filter?: string  
    ) => {
      return wailsApiClient.listConnections()
    },

    create: async (data: unknown) => {
      return wailsApiClient.createConnection(data)
    },

    save: async (data: unknown) => {
      return wailsApiClient.saveConnection(data)
    },

    test: async (data: unknown) => {
      return wailsApiClient.testConnection(data)
    },

    remove: async (connectionId: string) => {
      return wailsApiClient.removeConnection(connectionId)
    }
  },

  // Query endpoints
  queries: {
    execute: async (connectionId: string, sql: string, options?: { limit?: number; timeout?: number }) => {
      return wailsApiClient.executeQuery(connectionId, sql, options)
    },
    getEditableMetadata: async (jobId: string) => {
      return wailsApiClient.getEditableMetadata(jobId)
    },
    updateRow: async (payload: unknown) => {
      return wailsApiClient.updateQueryRow(payload)
    }
  },

  // Schema endpoints
  schema: {
    databases: async (connectionId: string) => {
      return wailsApiClient.getSchemas(connectionId)
    },

    tables: async (connectionId: string, schemaName?: string) => {
      return wailsApiClient.getTables(connectionId, schemaName)
    },

    columns: async (connectionId: string, schemaName: string, tableName: string) => {
      return wailsApiClient.getTableStructure(connectionId, schemaName, tableName)
    }
  }
}
