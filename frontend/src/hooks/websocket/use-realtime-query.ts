/**
 * useRealtimeQuery Hook - Real-time query execution with progress and streaming
 * Manages query state, progress updates, and result streaming
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { v4 as uuidv4 } from 'uuid';
import {
  QueryProgress,
  QueryResult,
  QueryError,
  DataChunk,
  UseRealtimeQueryOptions,
  EventHandler,
} from '../../types/websocket';
import { useWebSocket } from './use-websocket';

interface QueryState {
  queryId: string | null;
  status: 'idle' | 'executing' | 'streaming' | 'completed' | 'error' | 'cancelled';
  progress: number;
  progressMessage: string;
  result: QueryResult | null;
  error: QueryError | null;
  chunks: DataChunk[];
  totalRows: number;
  executionTime: number;
  startTime: number | null;
}

const initialState: QueryState = {
  queryId: null,
  status: 'idle',
  progress: 0,
  progressMessage: '',
  result: null,
  error: null,
  chunks: [],
  totalRows: 0,
  executionTime: 0,
  startTime: null,
};

export function useRealtimeQuery(options: UseRealtimeQueryOptions) {
  const {
    connectionName,
    streaming = false,
    autoExecute = false,
    onProgress,
    onResult,
    onError,
    onChunk,
  } = options;
  const { sendMessage, on, off, connectionState } = useWebSocket();

  // State
  const [queryState, setQueryState] = useState<QueryState>(initialState);
  const [queryHistory, setQueryHistory] = useState<QueryResult[]>([]);

  // Refs
  const currentQueryRef = useRef<string | null>(null);
  const chunksRef = useRef<DataChunk[]>([]);
  const abortControllerRef = useRef<AbortController | null>(null);

  /**
   * Reset query state
   */
  const resetState = useCallback(() => {
    setQueryState(initialState);
    chunksRef.current = [];
    currentQueryRef.current = null;
  }, []);

  const cancelQuery = useCallback(async () => {
    if (!currentQueryRef.current) return;

    const queryId = currentQueryRef.current;

    try {
      await sendMessage('cancel_query', { queryId });

      setQueryState(prev => ({
        ...prev,
        status: 'cancelled',
        progressMessage: 'Query cancelled',
      }));

      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }

      currentQueryRef.current = null;
    } catch (error) {
      console.error('Failed to cancel query:', error);
    }
  }, [sendMessage]);

  /**
   * Handle query progress updates
   */
  const handleProgress = useCallback((progress: QueryProgress) => {
    if (progress.queryId !== currentQueryRef.current) return;

    setQueryState(prev => ({
      ...prev,
      progress: progress.progress,
      progressMessage: progress.message,
    }));

    onProgress?.(progress);
  }, [onProgress]);

  /**
   * Handle query result
   */
  const handleResult = useCallback((result: QueryResult) => {
    if (result.queryId !== currentQueryRef.current) return;

    const executionTime = queryState.startTime
      ? Date.now() - queryState.startTime
      : result.executionTime;

    setQueryState(prev => ({
      ...prev,
      status: 'completed',
      result,
      executionTime,
      totalRows: result.totalRows,
      progress: 100,
      progressMessage: 'Query completed',
    }));

    // Add to history
    setQueryHistory(prev => [result, ...prev.slice(0, 9)]); // Keep last 10 queries

    currentQueryRef.current = null;
    onResult?.(result);
  }, [queryState.startTime, onResult]);

  /**
   * Handle query error
   */
  const handleError = useCallback((error: QueryError) => {
    if (error.queryId !== currentQueryRef.current) return;

    setQueryState(prev => ({
      ...prev,
      status: 'error',
      error,
      progressMessage: `Error: ${error.error.message}`,
    }));

    currentQueryRef.current = null;
    onError?.(error);
  }, [onError]);

  /**
   * Handle data chunk (for streaming queries)
   */
  const handleChunk = useCallback((chunk: DataChunk) => {
    if (chunk.queryId !== currentQueryRef.current) return;

    chunksRef.current.push(chunk);

    setQueryState(prev => ({
      ...prev,
      status: 'streaming',
      chunks: [...chunksRef.current],
      totalRows: prev.totalRows + chunk.chunk.length,
    }));

    onChunk?.(chunk);

    // If this is the last chunk, mark as completed
    if (chunk.isLast) {
      setQueryState(prev => ({
        ...prev,
        status: 'completed',
        progress: 100,
        progressMessage: 'Streaming completed',
      }));
      currentQueryRef.current = null;
    }
  }, [onChunk]);

  /**
   * Execute a SQL query
   */
  const executeQuery = useCallback(async (
    sql: string,
    queryOptions: {
      limit?: number;
      streaming?: boolean;
      timeout?: number;
    } = {}
  ) => {
    if (connectionState.status !== 'connected') {
      throw new Error('Not connected to WebSocket server');
    }

    // Cancel existing query if running
    if (currentQueryRef.current) {
      await cancelQuery();
    }

    const queryId = uuidv4();
    currentQueryRef.current = queryId;

    // Reset state
    resetState();

    // Create abort controller for cancellation
    abortControllerRef.current = new AbortController();

    setQueryState({
      ...initialState,
      queryId,
      status: 'executing',
      startTime: Date.now(),
      progressMessage: 'Preparing query...',
    });

    try {
      await sendMessage('execute_query', {
        queryId,
        sql,
        connectionName,
        streaming: queryOptions.streaming ?? streaming,
        limit: queryOptions.limit,
      });

      // Set timeout if specified
      if (queryOptions.timeout) {
        setTimeout(() => {
          if (currentQueryRef.current === queryId) {
            cancelQuery();
          }
        }, queryOptions.timeout);
      }

    } catch (error) {
      setQueryState(prev => ({
        ...prev,
        status: 'error',
        error: {
          queryId,
          error: {
            message: error instanceof Error ? error.message : 'Failed to execute query',
            code: 'EXECUTION_ERROR',
          },
        },
      }));
      currentQueryRef.current = null;
      throw error;
    }
  }, [connectionState.status, connectionName, streaming, sendMessage, resetState, cancelQuery]);

  /**
   * Cancel the current query
   */
  /**
   * Re-execute the last query
   */
  const reExecuteQuery = useCallback(async () => {
    if (!queryHistory.length) {
      throw new Error('No query history available');
    }

    // Note: We don't have the original SQL here, this would need to be stored
    // This is a simplified implementation
    throw new Error('Re-execution requires storing original SQL - not implemented');
  }, [queryHistory]);

  /**
   * Get combined result data (for streaming queries)
   */
  const getCombinedResult = useCallback((): unknown[][] => {
    if (queryState.result) {
      return queryState.result.rows;
    }

    // Combine chunks for streaming queries
    return chunksRef.current.reduce((combined, chunk) => {
      return [...combined, ...chunk.chunk];
    }, [] as unknown[][]);
  }, [queryState.result]);

  /**
   * Get query statistics
   */
  const getStats = useCallback(() => {
    return {
      currentQuery: {
        queryId: queryState.queryId,
        status: queryState.status,
        progress: queryState.progress,
        totalRows: queryState.totalRows,
        chunksReceived: queryState.chunks.length,
        executionTime: queryState.executionTime,
      },
      history: {
        count: queryHistory.length,
        lastExecution: queryHistory[0]?.executionTime || 0,
      },
    };
  }, [queryState, queryHistory]);

  /**
   * Check if query is currently running
   */
  const isExecuting = queryState.status === 'executing' || queryState.status === 'streaming';

  /**
   * Check if query can be cancelled
   */
  const canCancel = isExecuting && currentQueryRef.current !== null;

  // Set up event handlers
  useEffect(() => {
    on('query:progress', handleProgress as EventHandler);
    on('query:result', handleResult as EventHandler);
    on('query:error', handleError as EventHandler);
    on('data:chunk', handleChunk as EventHandler);

    return () => {
      off('query:progress', handleProgress as EventHandler);
      off('query:result', handleResult as EventHandler);
      off('query:error', handleError as EventHandler);
      off('data:chunk', handleChunk as EventHandler);
    };
  }, [on, off, handleProgress, handleResult, handleError, handleChunk]);

  // Auto-execute if specified
  useEffect(() => {
    if (autoExecute && connectionState.status === 'connected') {
      // This would require a default query to be specified
      // Implementation depends on specific requirements
    }
  }, [autoExecute, connectionState.status]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (currentQueryRef.current) {
        cancelQuery();
      }
    };
  }, [cancelQuery]);

  return {
    // State
    queryState,
    queryHistory,
    isExecuting,
    canCancel,

    // Actions
    executeQuery,
    cancelQuery,
    reExecuteQuery,
    resetState,

    // Data access
    getCombinedResult,

    // Utilities
    getStats,
  };
}
