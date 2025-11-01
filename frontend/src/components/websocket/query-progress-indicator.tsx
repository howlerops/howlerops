/**
 * Query Progress Indicator - Shows real-time query execution progress
 * Displays progress, status, and allows cancellation
 */

import React, { useState, useCallback } from 'react';
import { Progress } from '@/components/ui/progress';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Play,
  Square,
  CheckCircle,
  XCircle,
  Clock,
  Database,
  Zap,
  AlertTriangle,
} from 'lucide-react';
import { useRealtimeQuery } from '../../hooks/websocket';
import { QueryResult, QueryError } from '../../types/websocket';

interface QueryProgressIndicatorProps {
  connectionName: string;
  sql?: string;
  streaming?: boolean;
  onQueryComplete?: (result: QueryResult) => void;
  onQueryError?: (error: QueryError) => void;
  className?: string;
}

export function QueryProgressIndicator({
  connectionName,
  sql,
  streaming = false,
  onQueryComplete,
  onQueryError,
  className = '',
}: QueryProgressIndicatorProps) {
  const {
    queryState,
    isExecuting,
    canCancel,
    executeQuery,
    cancelQuery,
    _getCombinedResult,  
    getStats,
  } = useRealtimeQuery({
    connectionName,
    streaming,
    onResult: onQueryComplete,
    onError: onQueryError,
  });

  const [showDetails, setShowDetails] = useState(false);

  /**
   * Handle query execution
   */
  const handleExecute = useCallback(async () => {
    if (!sql?.trim()) return;

    try {
      await executeQuery(sql, { streaming });
    } catch (error) {
      console.error('Failed to execute query:', error);
    }
  }, [sql, streaming, executeQuery]);

  /**
   * Handle query cancellation
   */
  const handleCancel = useCallback(async () => {
    try {
      await cancelQuery();
    } catch (error) {
      console.error('Failed to cancel query:', error);
    }
  }, [cancelQuery]);

  /**
   * Get status icon and color
   */
  const getStatusDisplay = useCallback(() => {
    switch (queryState.status) {
      case 'executing':
        return {
          icon: <Play className="h-4 w-4 animate-pulse" />,
          color: 'text-primary',
          bgColor: 'bg-blue-50',
          text: 'Executing',
        };
      case 'streaming':
        return {
          icon: <Zap className="h-4 w-4" />,
          color: 'text-accent-foreground',
          bgColor: 'bg-purple-50',
          text: 'Streaming',
        };
      case 'completed':
        return {
          icon: <CheckCircle className="h-4 w-4" />,
          color: 'text-primary',
          bgColor: 'bg-green-50',
          text: 'Completed',
        };
      case 'error':
        return {
          icon: <XCircle className="h-4 w-4" />,
          color: 'text-destructive',
          bgColor: 'bg-red-50',
          text: 'Error',
        };
      case 'cancelled':
        return {
          icon: <Square className="h-4 w-4" />,
          color: 'text-muted-foreground',
          bgColor: 'bg-gray-50',
          text: 'Cancelled',
        };
      default:
        return {
          icon: <Database className="h-4 w-4" />,
          color: 'text-muted-foreground',
          bgColor: 'bg-gray-50',
          text: 'Ready',
        };
    }
  }, [queryState.status]);

  /**
   * Format execution time
   */
  const formatExecutionTime = useCallback((ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  }, []);

  /**
   * Format row count
   */
  const formatRowCount = useCallback((count: number): string => {
    if (count < 1000) return count.toString();
    if (count < 1000000) return `${(count / 1000).toFixed(1)}K`;
    return `${(count / 1000000).toFixed(1)}M`;
  }, []);

  const statusDisplay = getStatusDisplay();
  const stats = getStats();

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Main Status Bar */}
      <div className="flex items-center gap-3">
        {/* Status Badge */}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge
                variant="outline"
                className={`${statusDisplay.bgColor} ${statusDisplay.color} border-current cursor-pointer`}
                onClick={() => setShowDetails(!showDetails)}
              >
                {statusDisplay.icon}
                <span className="ml-2">{statusDisplay.text}</span>
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              <div className="text-xs space-y-1">
                <div>Status: {statusDisplay.text}</div>
                {queryState.queryId && (
                  <div>Query ID: {queryState.queryId}</div>
                )}
                {queryState.executionTime > 0 && (
                  <div>Time: {formatExecutionTime(queryState.executionTime)}</div>
                )}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>

        {/* Progress Bar */}
        {isExecuting && (
          <div className="flex-1">
            <Progress
              value={queryState.progress >= 0 ? queryState.progress : undefined}
              className="h-2"
            />
          </div>
        )}

        {/* Row Count */}
        {queryState.totalRows > 0 && (
          <Badge variant="secondary" className="text-xs">
            {formatRowCount(queryState.totalRows)} rows
          </Badge>
        )}

        {/* Execution Time */}
        {queryState.executionTime > 0 && (
          <Badge variant="outline" className="text-xs">
            <Clock className="h-3 w-3 mr-1" />
            {formatExecutionTime(queryState.executionTime)}
          </Badge>
        )}

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          {!isExecuting && sql && (
            <Button size="sm" onClick={handleExecute} className="h-7 text-xs">
              <Play className="h-3 w-3 mr-1" />
              Execute
            </Button>
          )}

          {canCancel && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleCancel}
              className="h-7 text-xs"
            >
              <Square className="h-3 w-3 mr-1" />
              Cancel
            </Button>
          )}
        </div>
      </div>

      {/* Progress Message */}
      {isExecuting && queryState.progressMessage && (
        <div className="text-sm text-muted-foreground bg-gray-50 px-3 py-2 rounded">
          {queryState.progressMessage}
        </div>
      )}

      {/* Error Display */}
      {queryState.error && (
        <div className="bg-red-50 border border-destructive rounded p-3">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
            <div className="space-y-1">
              <div className="text-sm font-medium text-destructive">
                Query Error
              </div>
              <div className="text-sm text-destructive">
                {queryState.error.error.message}
              </div>
              {queryState.error.error.line && (
                <div className="text-xs text-destructive">
                  Line {queryState.error.error.line}
                  {queryState.error.error.column && `, Column ${queryState.error.error.column}`}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Detailed Information */}
      {showDetails && (
        <div className="bg-gray-50 border rounded p-3 space-y-3">
          <div className="text-sm font-medium text-muted-foreground">Query Details</div>

          <div className="grid grid-cols-2 gap-4 text-xs">
            <div>
              <div className="text-muted-foreground">Connection</div>
              <div className="font-medium">{connectionName}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Mode</div>
              <div className="font-medium">
                {streaming ? 'Streaming' : 'Standard'}
              </div>
            </div>
            {queryState.startTime && (
              <div>
                <div className="text-muted-foreground">Started</div>
                <div className="font-medium">
                  {new Date(queryState.startTime).toLocaleTimeString()}
                </div>
              </div>
            )}
            {queryState.chunks.length > 0 && (
              <div>
                <div className="text-muted-foreground">Chunks</div>
                <div className="font-medium">{queryState.chunks.length}</div>
              </div>
            )}
          </div>

          {/* Streaming Progress */}
          {streaming && queryState.chunks.length > 0 && (
            <div className="space-y-2">
              <div className="text-muted-foreground text-xs">Streaming Progress</div>
              <div className="space-y-1">
                {queryState.chunks.slice(-3).map((chunk, index) => (
                  <div key={index} className="flex items-center justify-between text-xs">
                    <span>Chunk {chunk.chunkIndex + 1}</span>
                    <span>{chunk.chunk.length} rows</span>
                    {chunk.isLast && (
                      <Badge variant="outline" className="text-xs h-4">
                        Last
                      </Badge>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Statistics */}
          <div className="space-y-2">
            <div className="text-muted-foreground text-xs">Statistics</div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <span className="text-muted-foreground">History: </span>
                <span className="font-medium">{stats.history.count} queries</span>
              </div>
              <div>
                <span className="text-muted-foreground">Status: </span>
                <span className="font-medium">{stats.currentQuery.status}</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}