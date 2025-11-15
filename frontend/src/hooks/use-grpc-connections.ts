import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { wailsEndpoints } from '../lib/wails-api'

export interface Connection {
  id: string
  name: string
  description?: string
  type: string
  host?: string
  port?: number
  database?: string
  username?: string
  active: boolean
  createdAt?: string | Date
  updatedAt?: string | Date
  createdBy?: string
  tags?: Record<string, string>
}

export interface CreateConnectionData {
  name: string
  description?: string
  type: string
  host: string
  port: number
  database: string
  username: string
  password: string
  ssl_mode?: string
  connection_timeout?: number
  idle_timeout?: number
  max_connections?: number
  max_idle_connections?: number
  parameters?: Record<string, string>
  tags?: Record<string, string>
}

export function useGrpcConnections(page: number = 1, pageSize: number = 50, filter?: string) {
  return useQuery({
    queryKey: ['grpc-connections', page, pageSize, filter],
    queryFn: () => wailsEndpoints.connections.list(page, pageSize, filter),
    staleTime: 30 * 1000, // 30 seconds
  })
}

export function useCreateGrpcConnection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateConnectionData) => wailsEndpoints.connections.create(data),
    onSuccess: () => {
      // Invalidate and refetch connections
      queryClient.invalidateQueries({ queryKey: ['grpc-connections'] })
    },
  })
}

export function useUpdateGrpcConnection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ data }: { id: string; data: Partial<CreateConnectionData> }) =>
      wailsEndpoints.connections.create(data), // Note: Update not implemented in Wails API yet
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grpc-connections'] })
    },
  })
}

export function useDeleteGrpcConnection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => Promise.resolve({ success: true, message: 'Delete not implemented in Wails API' }), // Note: Delete not implemented in Wails API yet
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grpc-connections'] })
    },
  })
}

export function useTestGrpcConnection() {
  return useMutation({
    mutationFn: (data: CreateConnectionData) => wailsEndpoints.connections.test(data),
  })
}

export function useGrpcSchemas(connectionId: string) {
  return useQuery({
    queryKey: ['grpc-schemas', connectionId],
    queryFn: () => wailsEndpoints.schema.databases(connectionId),
    enabled: !!connectionId,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

export function useGrpcTables(connectionId: string, schemaName?: string, _tableType?: string) {
  return useQuery({
    queryKey: ['grpc-tables', connectionId, schemaName, _tableType],
    queryFn: () => wailsEndpoints.schema.tables(connectionId, schemaName),
    enabled: !!connectionId,
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

export function useGrpcTableMetadata(connectionId: string, schemaName: string, tableName: string) {
  return useQuery({
    queryKey: ['grpc-table-metadata', connectionId, schemaName, tableName],
    queryFn: () => wailsEndpoints.schema.columns(connectionId, schemaName, tableName),
    enabled: !!(connectionId && schemaName && tableName),
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

// Hook for executing queries with gRPC
export function useGrpcQuery() {
  return useMutation({
    mutationFn: ({
      connectionId,
      sql,
      options
    }: {
      connectionId: string;
      sql: string;
      options?: { limit?: number; timeout?: number }
    }) => {
      const limit = options?.limit
      const timeout = options?.timeout
      return wailsEndpoints.queries.execute(connectionId, sql, limit, undefined, timeout)
    },
  })
}

// Hook for getting query history
export function useGrpcQueryHistory(connectionId: string, limit: number = 50) {
  return useQuery({
    queryKey: ['grpc-query-history', connectionId, limit],
    queryFn: () => Promise.resolve({ data: [], hasMore: false, nextCursor: '' }), // Note: Query history not implemented in Wails API yet
    enabled: !!connectionId,
    staleTime: 1 * 60 * 1000, // 1 minute
  })
}

// Hook for explaining queries
export function useGrpcExplainQuery() {
  return useMutation({
    mutationFn: () =>
      Promise.resolve({ data: { plan: '', format: '', estimatedStats: {}, warnings: [] }, success: true, message: 'Explain not implemented in Wails API yet' }), // Note: Explain query not implemented in Wails API yet
  })
}

// Hook for cancelling queries
export function useGrpcCancelQuery() {
  return useMutation({
    mutationFn: () => Promise.resolve({ success: true, message: 'Cancel not implemented in Wails API yet', wasRunning: false }), // Note: Cancel query not implemented in Wails API yet
  })
}