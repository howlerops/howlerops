import React, { useState, useCallback } from 'react';
import { flexRender } from '@tanstack/react-table';
import { ChevronUp, ChevronDown, Filter, GripVertical } from 'lucide-react';
import { cn } from '../../utils/cn';
import { ColumnHeaderProps } from '../../types/table';

export const TableHeader: React.FC<ColumnHeaderProps> = ({
  header,
  canSort,
  canFilter,
  canResize,
  sortDirection,
}) => {
  const [showFilter, setShowFilter] = useState(false);
  const [filterValue, setFilterValue] = useState(
    (header.column.getFilterValue() as string) ?? ''
  );

  const handleSort = useCallback(() => {
    if (!canSort) return;
    header.column.toggleSorting();
  }, [canSort, header.column]);

  const handleFilterToggle = useCallback(() => {
    setShowFilter(!showFilter);
  }, [showFilter]);

  const handleFilterChange = useCallback((value: string) => {
    setFilterValue(value);
    header.column.setFilterValue(value || undefined);
  }, [header.column]);

  const handleFilterKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (event.key === 'Enter') {
      setShowFilter(false);
    }
    if (event.key === 'Escape') {
      setFilterValue('');
      header.column.setFilterValue(undefined);
      setShowFilter(false);
    }
  }, [header.column]);

  const renderSortIcon = () => {
    if (!canSort) return null;

    return (
      <span className="ml-1 flex flex-col">
        <ChevronUp
          className={cn(
            'h-3 w-3 transition-colors',
            sortDirection === 'asc' ? 'text-primary' : 'text-gray-400'
          )}
        />
        <ChevronDown
          className={cn(
            'h-3 w-3 -mt-1 transition-colors',
            sortDirection === 'desc' ? 'text-primary' : 'text-gray-400'
          )}
        />
      </span>
    );
  };

  const renderFilterButton = () => {
    if (!canFilter) return null;

    const hasFilter = header.column.getFilterValue();

    return (
      <button
        onClick={handleFilterToggle}
        className={cn(
          'ml-1 p-1 rounded hover:bg-gray-200 transition-colors',
          {
            'text-primary': hasFilter || showFilter,
            'text-gray-400': !hasFilter && !showFilter,
          }
        )}
        title="Filter column"
      >
        <Filter className="h-3 w-3" />
      </button>
    );
  };

  const renderResizeHandle = () => {
    if (!canResize) return null;

    return (
      <div
        onMouseDown={header.getResizeHandler()}
        onTouchStart={header.getResizeHandler()}
        className={cn(
          'absolute right-0 top-0 h-full w-1 cursor-col-resize select-none touch-none',
          'hover:bg-primary transition-colors',
          header.column.getIsResizing() && 'bg-primary'
        )}
        style={{
          transform: header.column.getIsResizing() ? 'scaleX(2)' : 'scaleX(1)',
        }}
      />
    );
  };

  const renderFilter = () => {
    if (!showFilter) return null;

    return (
      <div className="absolute top-full left-0 right-0 z-50 mt-1 p-2 bg-white border border-gray-300 rounded shadow-lg">
        <input
          type="text"
          value={filterValue}
          onChange={(e) => handleFilterChange(e.target.value)}
          onKeyDown={handleFilterKeyDown}
          onBlur={() => setShowFilter(false)}
          placeholder={`Filter ${header.column.columnDef.header}...`}
          className="w-full px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          autoFocus
        />
      </div>
    );
  };

  const sticky = (header.column.columnDef.meta as { sticky?: 'left' | 'right' } | undefined)?.sticky;

  return (
    <th
      className={cn(
        'relative bg-gray-50 border-b border-gray-200 px-3 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider',
        'select-none',
        {
          'cursor-pointer hover:bg-gray-100': canSort,
          'cursor-default': !canSort,
        },
        sticky && `sticky ${sticky === 'right' ? 'right-0' : 'left-0'} z-20 shadow-sm`
      )}
      style={{ width: header.getSize() }}
      onClick={handleSort}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center min-w-0 flex-1">
          <span className="truncate">
            {flexRender(header.column.columnDef.header, header.getContext())}
          </span>
          {renderSortIcon()}
        </div>

        <div className="flex items-center ml-2">
          {renderFilterButton()}
        </div>
      </div>

      {renderFilter()}
      {renderResizeHandle()}

      {/* Column reordering handle */}
      {header.column.getCanPin() && (
        <div className="absolute left-0 top-0 h-full w-4 cursor-move flex items-center justify-center hover:bg-gray-200">
          <GripVertical className="h-3 w-3 text-gray-400" />
        </div>
      )}
    </th>
  );
};