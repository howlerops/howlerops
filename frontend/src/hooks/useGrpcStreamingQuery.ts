import { useState, useCallback, useRef, useEffect, useMemo } from 'react'
import { streamingQueryClient, QueryResult, QueryProgress } from '../services/StreamingQueryClient'
import { ColumnMetadata } from '../generated/query'

export interface StreamingQueryState {
  isLoading: boolean
  isStreaming: boolean
  error: Error | null
  result: QueryResult | null
  progress: QueryProgress | null
  columns: ColumnMetadata[] | null
  rowCount: number
  cancel: () => void
  execute: (connectionId: string, sql: string, options?: unknown) => Promise<void>
}

export function useGrpcStreamingQuery(): StreamingQueryState {
  const [isLoading, setIsLoading] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [result, setResult] = useState<QueryResult | null>(null)
  const [progress, setProgress] = useState<QueryProgress | null>(null)
  const [columns, setColumns] = useState<ColumnMetadata[] | null>(null)
  const [rowCount, setRowCount] = useState(0)

  const currentQueryIdRef = useRef<string | null>(null)

  const cancel = useCallback(() => {
    if (currentQueryIdRef.current) {
      streamingQueryClient.cancelQuery(currentQueryIdRef.current)
      currentQueryIdRef.current = null
      setIsLoading(false)
      setIsStreaming(false)
    }
  }, [])

  const execute = useCallback(async (
    connectionId: string,
    sql: string,
    options: {
      chunkSize?: number
      timeout?: number
      readOnly?: boolean
      onProgress?: (progress: QueryProgress) => void
      onRow?: (row: unknown[]) => void
      onMetadata?: (columns: ColumnMetadata[]) => void
    } = {}
  ) => {
    // Cancel any existing query
    cancel()

    setIsLoading(true)
    setIsStreaming(false)
    setError(null)
    setResult(null)
    setProgress(null)
    setColumns(null)
    setRowCount(0)

    try {
      const queryResult = await streamingQueryClient.executeStreamingQuery(
        connectionId,
        sql,
        {
          ...options,
          onProgress: (prog) => {
            setProgress(prog)
            setRowCount(prog.rowsProcessed)
            setIsStreaming(true)
            if (options.onProgress) {
              options.onProgress(prog)
            }
          },
          onRow: (row) => {
            setRowCount(prev => prev + 1)
            if (options.onRow) {
              options.onRow(row)
            }
          },
          onMetadata: (cols) => {
            setColumns(cols)
            if (options.onMetadata) {
              options.onMetadata(cols)
            }
          },
        }
      )

      setResult(queryResult)
      setIsLoading(false)
      setIsStreaming(false)
    } catch (err) {
      setError(err as Error)
      setIsLoading(false)
      setIsStreaming(false)
    }
  }, [cancel])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cancel()
    }
  }, [cancel])

  return {
    isLoading,
    isStreaming,
    error,
    result,
    progress,
    columns,
    rowCount,
    cancel,
    execute,
  }
}

// Hook for managing multiple streaming queries
export function useGrpcMultipleStreamingQueries() {
  const [queryStates, setQueryStates] = useState<Map<string, {
    isLoading: boolean
    isStreaming: boolean
    error: Error | null
    result: QueryResult | null
    progress: QueryProgress | null
    columns: ColumnMetadata[] | null
    rowCount: number
  }>>(new Map())
  const queryClientsRef = useRef<Map<string, {
    cancel: () => void
    execute: (connectionId: string, sql: string, options?: unknown) => Promise<void>
  }>>(new Map())

  const createQuery = useCallback((queryId: string) => {
    // Create a new streaming query client without calling the hook
    const queryState = {
      isLoading: false,
      isStreaming: false,
      error: null,
      result: null,
      progress: null,
      columns: null,
      rowCount: 0,
    }

    const queryClient = {
      cancel: () => {
        streamingQueryClient.cancelQuery(queryId)
        setQueryStates(prev => {
          const newMap = new Map(prev)
          const current = newMap.get(queryId)
          if (current) {
            newMap.set(queryId, { ...current, isLoading: false, isStreaming: false })
          }
          return newMap
        })
      },
      execute: async (connectionId: string, sql: string, options: unknown = {}) => {
        setQueryStates(prev => new Map(prev.set(queryId, {
          ...queryState,
          isLoading: true,
          error: null,
        })))

        try {
          const result = await streamingQueryClient.executeStreamingQuery(connectionId, sql, options)
          setQueryStates(prev => {
            const newMap = new Map(prev)
            newMap.set(queryId, {
              isLoading: false,
              isStreaming: false,
              error: null,
              result,
              progress: null,
              columns: null,
              rowCount: 0,
            })
            return newMap
          })
        } catch (error) {
          setQueryStates(prev => {
            const newMap = new Map(prev)
            newMap.set(queryId, {
              isLoading: false,
              isStreaming: false,
              error: error as Error,
              result: null,
              progress: null,
              columns: null,
              rowCount: 0,
            })
            return newMap
          })
        }
      }
    }

    setQueryStates(prev => new Map(prev.set(queryId, queryState)))
    queryClientsRef.current.set(queryId, queryClient)

    return {
      ...queryState,
      cancel: queryClient.cancel,
      execute: queryClient.execute,
    }
  }, [])

  const removeQuery = useCallback((queryId: string) => {
    const queryClient = queryClientsRef.current.get(queryId)
    if (queryClient) {
      queryClient.cancel()
      queryClientsRef.current.delete(queryId)
      setQueryStates(prev => {
        const newMap = new Map(prev)
        newMap.delete(queryId)
        return newMap
      })
    }
  }, [])

  const cancelAll = useCallback(() => {
    queryClientsRef.current.forEach(queryClient => queryClient.cancel())
    queryClientsRef.current.clear()
    setQueryStates(new Map())
  }, [])

  const queries = useMemo(() => {
    const result = new Map<string, StreamingQueryState>()
    queryStates.forEach((state, queryId) => {
      const client = queryClientsRef.current.get(queryId)
      if (client) {
        result.set(queryId, {
          ...state,
          cancel: client.cancel,
          execute: client.execute,
        })
      }
    })
    return result
  }, [queryStates])

  return {
    queries,
    createQuery,
    removeQuery,
    cancelAll,
    activeCount: queries.size,
  }
}

// Hook for real-time query monitoring
export function useGrpcQueryMonitor() {
  const [activeQueries, setActiveQueries] = useState<string[]>([])
  const [queryMetrics, setQueryMetrics] = useState<Map<string, QueryProgress>>(new Map())

  useEffect(() => {
    const updateMetrics = () => {
      const active = streamingQueryClient.getActiveQueries()
      setActiveQueries(active)

      const metrics = new Map<string, QueryProgress>()
      active.forEach(queryId => {
        const metric = streamingQueryClient.getQueryMetrics?.(queryId)
        if (metric) {
          metrics.set(queryId, metric)
        }
      })
      setQueryMetrics(metrics)
    }

    const interval = setInterval(updateMetrics, 1000) // Update every second
    updateMetrics() // Initial update

    return () => clearInterval(interval)
  }, [])

  return {
    activeQueries,
    queryMetrics,
    totalActiveQueries: activeQueries.length,
  }
}