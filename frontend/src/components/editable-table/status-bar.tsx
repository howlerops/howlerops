import { AlertCircle,CheckCircle, Clock, Database, Filter } from 'lucide-react';
import React, { useDeferredValue } from 'react';

import { StatusBarProps } from '../../types/table';
import { cn } from '../../utils/cn';

export const StatusBar: React.FC<StatusBarProps> = ({
  totalRows,
  selectedRows,
  filteredRows,
  dirtyRows,
  loading = false,
  lastUpdated,
  customStatus,
}) => {
  // Defer values that change frequently to prevent re-renders during typing
  const deferredDirtyRows = useDeferredValue(dirtyRows);
  const deferredSelectedRows = useDeferredValue(selectedRows);

  // Track if we're showing stale data
  const isPending = dirtyRows !== deferredDirtyRows || selectedRows !== deferredSelectedRows;
  const formatTime = (date: Date) => {
    return new Intl.DateTimeFormat('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(date);
  };

  const formatNumber = (num: number) => {
    return num.toLocaleString();
  };

  return (
    <div className={cn(
      "flex items-center justify-between px-4 py-2 bg-gray-50 border-t border-gray-200 text-sm text-gray-600",
      isPending && "opacity-70 transition-opacity"
    )}>
      {/* Left side - Row information */}
      <div className="flex items-center gap-6">
        {/* Total rows */}
        <div className="flex items-center gap-1">
          <Database className="h-4 w-4" />
          <span>
            {formatNumber(totalRows)} row{totalRows !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Filtered rows (if different from total) */}
        {filteredRows !== undefined && filteredRows !== totalRows && (
          <div className="flex items-center gap-1 text-primary">
            <Filter className="h-4 w-4" />
            <span>
              {formatNumber(filteredRows)} filtered
            </span>
          </div>
        )}

        {/* Selected rows */}
        {deferredSelectedRows > 0 && (
          <div className="flex items-center gap-1 text-primary font-medium">
            <CheckCircle className="h-4 w-4" />
            <span>
              {formatNumber(deferredSelectedRows)} selected
            </span>
          </div>
        )}

        {/* Dirty rows */}
        {deferredDirtyRows > 0 && (
          <div className="flex items-center gap-1 text-accent-foreground font-medium">
            <AlertCircle className="h-4 w-4" />
            <span>
              {formatNumber(deferredDirtyRows)} modified
            </span>
          </div>
        )}

        {/* Loading indicator */}
        {loading && (
          <div className="flex items-center gap-1 text-primary">
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
            <span>Loading...</span>
          </div>
        )}
      </div>

      {/* Center - Custom status */}
      {customStatus && (
        <div className="flex-1 flex justify-center">
          {customStatus}
        </div>
      )}

      {/* Right side - Timestamps and actions */}
      <div className="flex items-center gap-4">
        {/* Last updated */}
        {lastUpdated && !loading && (
          <div className="flex items-center gap-1 text-gray-500">
            <Clock className="h-4 w-4" />
            <span>
              Updated {formatTime(lastUpdated)}
            </span>
          </div>
        )}

        {/* Performance indicator */}
        <div className="flex items-center gap-1">
          <div
            className={cn(
              'w-2 h-2 rounded-full',
              loading
                ? 'bg-accent'
                : deferredDirtyRows > 0
                ? 'bg-accent'
                : 'bg-primary'
            )}
          />
          <span className="text-xs">
            {loading
              ? 'Syncing'
              : deferredDirtyRows > 0
              ? 'Modified'
              : 'Synced'
            }
          </span>
        </div>
      </div>
    </div>
  );
};