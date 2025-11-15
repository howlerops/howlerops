import axios from 'axios'
import { QueryClient } from '@tanstack/react-query'
import { grpcWebClient } from './grpc-web-client'
import { navigationService } from './navigation'
import {
  DatabaseType,
  ConnectionConfig,
} from '../generated/database'
import { QueryOptions, DataFormat } from '../generated/query'

// Create axios instance with base configuration
export const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('auth-token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized access
      localStorage.removeItem('auth-token')
      navigationService.to('/login')
    }
    return Promise.reject(error)
  }
)

// Create React Query client
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
      retry: (failureCount, error: unknown) => {
        // Type guard for axios error
        interface AxiosError {
          response?: {
            status: number
          }
        }
        if ((error as AxiosError)?.response?.status === 404) return false
        return failureCount < 3
      },
    },
    mutations: {
      retry: false,
    },
  },
})

// Helper function to convert REST connection data to gRPC format
function convertToGrpcConnectionConfig(data: unknown): ConnectionConfig {
  const dataObj = data as {
    type?: string;
    host?: string;
    port?: number;
    database?: string;
    username?: string;
    password?: string;
    ssl_mode?: string;
    connection_timeout?: number;
    idle_timeout?: number;
    max_connections?: number;
    max_idle_connections?: number;
    parameters?: Record<string, unknown>;
  };

  const config: ConnectionConfig = {
    type: mapDatabaseType(dataObj.type || 'postgresql'),
    host: dataObj.host || '',
    port: dataObj.port || 0,
    database: dataObj.database || '',
    username: dataObj.username || '',
    password: dataObj.password || '',
    sslMode: dataObj.ssl_mode || 'prefer',  // Default to 'prefer' for better security
    connectionTimeout: dataObj.connection_timeout || 30,
    idleTimeout: dataObj.idle_timeout || 300,
    maxConnections: dataObj.max_connections || 10,
    maxIdleConnections: dataObj.max_idle_connections || 5,
    parameters: (dataObj.parameters as Record<string, string>) || {},
    useTunnel: false,
    sshTunnel: undefined,
    useVpc: false,
    vpcConfig: undefined,
  };
  return config;
}

// Helper function to map database types
function mapDatabaseType(type: string): DatabaseType {
  switch (type?.toLowerCase()) {
    case 'postgresql':
    case 'postgres':
      return DatabaseType.DATABASE_TYPE_POSTGRESQL;
    case 'mysql':
      return DatabaseType.DATABASE_TYPE_MYSQL;
    case 'sqlite':
      return DatabaseType.DATABASE_TYPE_SQLITE;
    case 'mariadb':
      return DatabaseType.DATABASE_TYPE_MARIADB;
    default:
      return DatabaseType.DATABASE_TYPE_UNSPECIFIED;
  }
}

// Helper function to convert gRPC database type back to string
function convertFromGrpcDatabaseType(type: DatabaseType): string {
  switch (type) {
    case DatabaseType.DATABASE_TYPE_POSTGRESQL:
      return 'postgresql';
    case DatabaseType.DATABASE_TYPE_MYSQL:
      return 'mysql';
    case DatabaseType.DATABASE_TYPE_SQLITE:
      return 'sqlite';
    case DatabaseType.DATABASE_TYPE_MARIADB:
      return 'mariadb';
    default:
      return 'postgresql';
  }
}

// gRPC-Web API endpoints (replacing REST endpoints)
export const grpcEndpoints = {
  // Connection endpoints
  connections: {
    list: async (page: number = 1, pageSize: number = 50, filter?: string) => {
      const response = await grpcWebClient.listConnections({
        page,
        pageSize,
        filter: filter || '',
        sort: '',
        includeInactive: false,
      });

      return {
        data: response.connections?.map(conn => ({
          id: conn.id,
          name: conn.name,
          description: conn.description,
          type: convertFromGrpcDatabaseType(conn.config?.type || DatabaseType.DATABASE_TYPE_POSTGRESQL),
          host: conn.config?.host,
          port: conn.config?.port,
          database: conn.config?.database,
          username: conn.config?.username,
          active: conn.active,
          createdAt: conn.createdAt,
          updatedAt: conn.updatedAt,
          createdBy: conn.createdBy,
          tags: conn.tags,
        })) || [],
        total: response.total,
        page: response.page,
        pageSize: response.pageSize,
      };
    },

    create: async (data: unknown) => {
      const dataObj = data as {
        name?: string;
        description?: string;
        tags?: Record<string, unknown>;
        [key: string]: unknown;
      };
      const config = convertToGrpcConnectionConfig(data);
      const response = await grpcWebClient.createConnection({
        name: dataObj.name || '',
        description: dataObj.description || '',
        config,
        tags: (dataObj.tags as Record<string, string>) || {},
      });

      return {
        data: response.connection ? {
          id: response.connection.id,
          name: response.connection.name,
          description: response.connection.description,
          type: convertFromGrpcDatabaseType(response.connection.config?.type || DatabaseType.DATABASE_TYPE_POSTGRESQL),
          // ... other fields
        } : null,
        success: response.success,
        message: response.message,
      };
    },

    update: async (id: string, data: unknown) => {
      const dataObj = data as {
        name?: string;
        description?: string;
        tags?: Record<string, string>;
      };
      const config = convertToGrpcConnectionConfig(data);
      const response = await grpcWebClient.updateConnection({
        id,
        name: dataObj.name || '',
        description: dataObj.description || '',
        config,
        tags: dataObj.tags || {},
      });

      return {
        data: response.connection,
        success: response.success,
        message: response.message,
      };
    },

    delete: async (id: string) => {
      const response = await grpcWebClient.deleteConnection({ id });
      return {
        success: response.success,
        message: response.message,
      };
    },

    test: async (data: unknown) => {
      const config = convertToGrpcConnectionConfig(data);
      const response = await grpcWebClient.testConnection({
        connectionId: '',
        config,
      });

      return {
        data: {
          success: response.success,
          responseTime: response.responseTimeMs,
          version: response.version,
          serverInfo: response.serverInfo,
        },
        success: response.success,
        message: response.message,
      };
    },
  },

  // Query endpoints
  queries: {
    execute: async (connectionId: string, sql: string, options?: Partial<QueryOptions>) => {
      const queryOptions: QueryOptions = {
        limit: options?.limit || 5000,
        timeoutSeconds: options?.timeoutSeconds || 30,
        readOnly: options?.readOnly || false,
        explain: options?.explain || false,
        format: options?.format || DataFormat.DATA_FORMAT_JSON,
        includeMetadata: options?.includeMetadata || true,
        fetchSize: options?.fetchSize || 1000,
      };

      const response = await grpcWebClient.executeQuery({
        connectionId,
        sql,
        parameters: {},
        options: queryOptions,
        queryId: `query-${Date.now()}`,
      });

      return {
        data: {
          queryId: response.queryId,
          success: response.success,
          columns: response.result?.columns || [],
          rows: response.result?.rows || [],
          rowCount: response.result?.rowCount || 0,
          stats: response.stats,
          warnings: response.warnings,
        },
        success: response.success,
        message: response.message,
      };
    },

    executeStreaming: async (
      connectionId: string,
      sql: string,
      onMessage: (data: unknown) => void,
      onError?: (error: Error) => void,
      onComplete?: () => void,
      options?: Partial<QueryOptions>
    ) => {
      const queryOptions: QueryOptions = {
        limit: options?.limit || 0, // 0 means no limit for streaming
        timeoutSeconds: options?.timeoutSeconds || 300, // 5 minutes for large queries
        readOnly: options?.readOnly || false,
        explain: options?.explain || false,
        format: options?.format || DataFormat.DATA_FORMAT_JSON,
        includeMetadata: options?.includeMetadata || true,
        fetchSize: options?.fetchSize || 10000, // Larger fetch size for streaming
      };

      return grpcWebClient.executeStreamingQuery(
        {
          connectionId,
          sql,
          parameters: {},
          options: queryOptions,
          queryId: `streaming-query-${Date.now()}`,
        },
        onMessage,
        onError,
        onComplete
      );
    },

    history: async (connectionId: string, limit: number = 50) => {
      const response = await grpcWebClient.getQueryHistory({
        connectionId,
        userId: '', // Will be filled by backend from auth
        fromTime: undefined,
        toTime: undefined,
        limit,
        cursor: '',
      });

      return {
        data: response.entries?.map(entry => ({
          queryId: entry.queryId,
          connectionId: entry.connectionId,
          sql: entry.sql,
          status: entry.status,
          startedAt: entry.startedAt,
          completedAt: entry.completedAt,
          stats: entry.stats,
          errorMessage: entry.errorMessage,
        })) || [],
        hasMore: response.hasMore,
        nextCursor: response.nextCursor,
      };
    },

    explain: async (connectionId: string, sql: string) => {
      const response = await grpcWebClient.explainQuery({
        connectionId,
        sql,
        parameters: {},
        analyze: false,
        format: 'JSON',
      });

      return {
        data: {
          plan: response.plan?.plan,
          format: response.plan?.format,
          estimatedStats: response.plan?.estimatedStats,
          warnings: response.plan?.warnings,
        },
        success: response.success,
        message: response.message,
      };
    },

    cancel: async (queryId: string) => {
      const response = await grpcWebClient.cancelQuery({ queryId });
      return {
        success: response.success,
        message: response.message,
        wasRunning: response.wasRunning,
      };
    },
  },

  // Schema endpoints
  schema: {
    databases: async (connectionId: string) => {
      const response = await grpcWebClient.getSchemas({ connectionId });
      return {
        data: response.schemas?.map(schema => ({
          name: schema.name,
          owner: schema.owner,
          createdAt: schema.createdAt,
          tableCount: schema.tableCount,
          sizeBytes: schema.sizeBytes,
          metadata: schema.metadata,
        })) || [],
        success: response.success,
        message: response.message,
      };
    },

    tables: async (connectionId: string, schemaName?: string, tableType?: string) => {
      const response = await grpcWebClient.getTables({
        connectionId,
        schemaName: schemaName || '',
        tableType: tableType || 'TABLE',
      });

      return {
        data: response.tables?.map(table => ({
          name: table.name,
          schema: table.schema,
          type: table.type,
          comment: table.comment,
          createdAt: table.createdAt,
          updatedAt: table.updatedAt,
          rowCount: table.rowCount,
          sizeBytes: table.sizeBytes,
          owner: table.owner,
          metadata: table.metadata,
        })) || [],
        success: response.success,
        message: response.message,
      };
    },

    columns: async (connectionId: string, schemaName: string, tableName: string) => {
      const response = await grpcWebClient.getTableMetadata({
        connectionId,
        schemaName,
        tableName,
      });

      return {
        data: response.metadata?.columns?.map(column => ({
          name: column.name,
          dataType: column.dataType,
          nullable: column.nullable,
          defaultValue: column.defaultValue,
          primaryKey: column.primaryKey,
          unique: column.unique,
          indexed: column.indexed,
          comment: column.comment,
          ordinalPosition: column.ordinalPosition,
          characterMaximumLength: column.characterMaximumLength,
          numericPrecision: column.numericPrecision,
          numericScale: column.numericScale,
          metadata: column.metadata,
        })) || [],
        table: response.metadata?.table,
        indexes: response.metadata?.indexes,
        foreignKeys: response.metadata?.foreignKeys,
        triggers: response.metadata?.triggers,
        statistics: response.metadata?.statistics,
        success: response.success,
        message: response.message,
      };
    },
  },
};

// Legacy REST API endpoints (kept for backward compatibility)
export const endpoints = {
  // Connection endpoints
  connections: {
    list: () => api.get('/connections'),
    create: (data: unknown) => api.post('/connections', data),
    update: (id: string, data: unknown) => api.put(`/connections/${id}`, data),
    delete: (id: string) => api.delete(`/connections/${id}`),
    test: (data: unknown) => api.post('/connections/test', data),
    connect: (id: string) => api.post(`/connections/${id}/connect`),
    disconnect: (id: string) => api.post(`/connections/${id}/disconnect`),
  },

  // Query endpoints
  queries: {
    execute: (connectionId: string, query: string) =>
      api.post(`/queries/execute`, { connectionId, query }),
    history: (connectionId: string) =>
      api.get(`/queries/history?connectionId=${connectionId}`),
    explain: (connectionId: string, query: string) =>
      api.post(`/queries/explain`, { connectionId, query }),
  },

  // Schema endpoints
  schema: {
    databases: (connectionId: string) =>
      api.get(`/schema/databases?connectionId=${connectionId}`),
    tables: (connectionId: string, database?: string) =>
      api.get(`/schema/tables?connectionId=${connectionId}${database ? `&database=${database}` : ''}`),
    columns: (connectionId: string, table: string, database?: string) =>
      api.get(`/schema/columns?connectionId=${connectionId}&table=${table}${database ? `&database=${database}` : ''}`),
  },
}

// React Query hooks
export const useConnections = () => {
  return {
    // Add query hooks here as needed
  }
}