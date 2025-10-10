import * as App from '../../wailsjs/go/main/App'

// Wails-based API client for desktop application
export class WailsApiClient {
  // Schema introspection methods
  async getSchemas(connectionId: string) {
    try {
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
      const structure = await App.GetTableStructure(connectionId, schemaName, tableName)

      return {
        data: structure.columns?.map(column => ({
          name: column.name,
          dataType: column.dataType,
          nullable: column.nullable,
          defaultValue: column.defaultValue,
          primaryKey: column.primaryKey,
          unique: column.unique,
          indexed: false, // Not provided by the backend structure
          comment: '',
          ordinalPosition: column.ordinalPosition,
          characterMaximumLength: column.characterMaxLength,
          numericPrecision: column.numericPrecision,
          numericScale: column.numericScale,
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
        foreignKeys: structure.foreignKeys || [],
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
      const result = await App.CreateConnection({
        type: data.type,
        host: data.host,
        port: data.port,
        database: data.database,
        username: data.username,
        password: data.password,
        sslMode: data.ssl_mode || 'disable',
        connectionTimeout: data.connection_timeout || 30,
        parameters: data.parameters || {}
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

  async testConnection(data: unknown) {
    try {
      await App.TestConnection({
        type: data.type,
        host: data.host,
        port: data.port,
        database: data.database,
        username: data.username,
        password: data.password,
        sslMode: data.ssl_mode || 'disable',
        connectionTimeout: data.connection_timeout || 30,
        parameters: data.parameters || {}
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
    } catch (_error) { // eslint-disable-line @typescript-eslint/no-unused-vars
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

  async executeQuery(connectionId: string, sql: string, options?: unknown) {
    try {
      const result = await App.ExecuteQuery({
        connectionId,
        query: sql,
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

  async updateQueryRow(payload: unknown) {
    try {
      const response = await App.UpdateQueryRow(payload)
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

// Create singleton instance
export const wailsApiClient = new WailsApiClient()

// Updated endpoints using Wails API
export const wailsEndpoints = {
  // Connection endpoints
  connections: {
    list: async (
      _page: number = 1, // eslint-disable-line @typescript-eslint/no-unused-vars
      _pageSize: number = 50, // eslint-disable-line @typescript-eslint/no-unused-vars
      _filter?: string // eslint-disable-line @typescript-eslint/no-unused-vars
    ) => {
      return wailsApiClient.listConnections()
    },

    create: async (data: unknown) => {
      return wailsApiClient.createConnection(data)
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
    execute: async (connectionId: string, sql: string, options?: unknown) => {
      return wailsApiClient.executeQuery(connectionId, sql, options)
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
