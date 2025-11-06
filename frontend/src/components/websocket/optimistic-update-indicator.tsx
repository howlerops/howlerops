/**
 * Optimistic Update Indicator - Shows pending optimistic updates
 * Provides visual feedback for optimistic operations
 */

import React, { useState, useCallback } from 'react';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import {
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  AlertTriangle,
  RefreshCw,
} from 'lucide-react';
import { useOptimisticUpdates } from '../../hooks/websocket';
import { OptimisticUpdate } from '../../types/websocket';

interface OptimisticUpdateIndicatorProps {
  tableId?: string;
  rowId?: string | number;
  className?: string;
}

export function OptimisticUpdateIndicator({
  tableId,
  rowId,
  className = '',
}: OptimisticUpdateIndicatorProps) {
  const {
    optimisticState,
    getPendingUpdates,
    rollbackUpdate,
    clearAllUpdates,
    getStats,
  } = useOptimisticUpdates();

  const [isPopoverOpen, setIsPopoverOpen] = useState(false);

  // Get relevant updates
  const pendingUpdates = getPendingUpdates(tableId).filter(
    update => rowId === undefined || update.rowId === rowId
  );

  const confirmedUpdates = Array.from(optimisticState.updates.values()).filter(
    update =>
      update.status === 'confirmed' &&
      (tableId === undefined || update.tableId === tableId) &&
      (rowId === undefined || update.rowId === rowId)
  );

  const rejectedUpdates = Array.from(optimisticState.updates.values()).filter(
    update =>
      update.status === 'rejected' &&
      (tableId === undefined || update.tableId === tableId) &&
      (rowId === undefined || update.rowId === rowId)
  );

  /**
   * Handle rollback of specific update
   */
  const handleRollback = useCallback(async (updateId: string) => {
    try {
      await rollbackUpdate(updateId);
    } catch (error) {
      console.error('Failed to rollback update:', error);
    }
  }, [rollbackUpdate]);

  /**
   * Format timestamp for display
   */
  const formatTimestamp = useCallback((timestamp: number): string => {
    const now = Date.now();
    const diff = now - timestamp;

    if (diff < 1000) return 'just now';
    if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    return `${Math.floor(diff / 3600000)}h ago`;
  }, []);

  /**
   * Get update status icon
   */
  const getUpdateStatusIcon = useCallback((update: OptimisticUpdate) => {
    switch (update.status) {
      case 'pending':
        return <Loader2 className="h-3 w-3 animate-spin text-primary" />;
      case 'confirmed':
        return <CheckCircle className="h-3 w-3 text-primary" />;
      case 'rejected':
        return <XCircle className="h-3 w-3 text-destructive" />;
      default:
        return <Clock className="h-3 w-3 text-muted-foreground" />;
    }
  }, []);

  /**
   * Get badge variant based on status
   */
  const getBadgeVariant = useCallback(() => {
    if (pendingUpdates.length > 0) return 'default';
    if (rejectedUpdates.length > 0) return 'destructive';
    if (confirmedUpdates.length > 0) return 'secondary';
    return 'outline';
  }, [pendingUpdates.length, rejectedUpdates.length, confirmedUpdates.length]);

  /**
   * Get badge text
   */
  const getBadgeText = useCallback(() => {
    const totalUpdates = pendingUpdates.length + confirmedUpdates.length + rejectedUpdates.length;

    if (totalUpdates === 0) return null;

    if (pendingUpdates.length > 0) {
      return `${pendingUpdates.length} pending`;
    }

    if (rejectedUpdates.length > 0) {
      return `${rejectedUpdates.length} failed`;
    }

    if (confirmedUpdates.length > 0) {
      return `${confirmedUpdates.length} synced`;
    }

    return null;
  }, [pendingUpdates.length, confirmedUpdates.length, rejectedUpdates.length]);

  /**
   * Render update item
   */
  const renderUpdateItem = useCallback((update: OptimisticUpdate) => {
    return (
      <div key={update.id} className="flex items-start gap-2 p-2 rounded border">
        <div className="mt-1">
          {getUpdateStatusIcon(update)}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 text-sm">
            <span className="font-medium">{update.type.replace('_', ' ')}</span>
            <span className="text-muted-foreground">
              {formatTimestamp(update.timestamp)}
            </span>
          </div>

          <div className="text-xs text-muted-foreground mt-1">
            {update.rowId && (
              <div>Row: {update.rowId}</div>
            )}
            <div>Changes: {Object.keys(update.changes).join(', ')}</div>
          </div>

          {update.status === 'pending' && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleRollback(update.id)}
              className="mt-2 h-6 text-xs"
            >
              <RefreshCw className="h-3 w-3 mr-1" />
              Rollback
            </Button>
          )}
        </div>
      </div>
    );
  }, [getUpdateStatusIcon, formatTimestamp, handleRollback]);

  const badgeText = getBadgeText();
  if (!badgeText) return null;

  const stats = getStats();

  return (
    <TooltipProvider>
      <Popover open={isPopoverOpen} onOpenChange={setIsPopoverOpen}>
        <PopoverTrigger asChild>
          <div className={className}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge
                  variant={getBadgeVariant()}
                  className="cursor-pointer hover:opacity-80 transition-opacity"
                >
                  {pendingUpdates.length > 0 && (
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  )}
                  {rejectedUpdates.length > 0 && (
                    <AlertTriangle className="h-3 w-3 mr-1" />
                  )}
                  {confirmedUpdates.length > 0 && pendingUpdates.length === 0 && rejectedUpdates.length === 0 && (
                    <CheckCircle className="h-3 w-3 mr-1" />
                  )}
                  {badgeText}
                </Badge>
              </TooltipTrigger>
              <TooltipContent>
                <div className="text-xs">
                  <div>Optimistic Updates:</div>
                  <div>• {pendingUpdates.length} pending</div>
                  <div>• {confirmedUpdates.length} confirmed</div>
                  <div>• {rejectedUpdates.length} rejected</div>
                </div>
              </TooltipContent>
            </Tooltip>
          </div>
        </PopoverTrigger>

        <PopoverContent className="w-80" align="end">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="font-medium">Optimistic Updates</h4>
              {stats.pending > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={clearAllUpdates}
                  className="h-6 text-xs"
                >
                  Clear All
                </Button>
              )}
            </div>

            {/* Statistics */}
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div className="text-center p-2 bg-blue-50 rounded">
                <div className="font-medium text-primary">{stats.pending}</div>
                <div className="text-primary">Pending</div>
              </div>
              <div className="text-center p-2 bg-green-50 rounded">
                <div className="font-medium text-primary">{stats.confirmed}</div>
                <div className="text-primary">Confirmed</div>
              </div>
              <div className="text-center p-2 bg-red-50 rounded">
                <div className="font-medium text-destructive">{stats.rejected}</div>
                <div className="text-destructive">Rejected</div>
              </div>
            </div>

            {/* Update List */}
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {/* Pending Updates */}
              {pendingUpdates.map(update => renderUpdateItem(update))}

              {/* Recent Confirmed Updates */}
              {confirmedUpdates.slice(0, 3).map(update => renderUpdateItem(update))}

              {/* Recent Rejected Updates */}
              {rejectedUpdates.slice(0, 3).map(update => renderUpdateItem(update))}

              {pendingUpdates.length === 0 && confirmedUpdates.length === 0 && rejectedUpdates.length === 0 && (
                <div className="text-center text-muted-foreground text-sm py-4">
                  No optimistic updates
                </div>
              )}
            </div>

            {/* Health Information */}
            {stats.pending > 0 && (
              <div className="text-xs text-muted-foreground bg-gray-50 p-2 rounded">
                Average age: {Math.round(stats.averageAge)}ms
                {stats.averageAge > 5000 && (
                  <div className="text-accent-foreground mt-1">
                    ⚠️ Some updates are taking longer than expected
                  </div>
                )}
              </div>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  );
}
