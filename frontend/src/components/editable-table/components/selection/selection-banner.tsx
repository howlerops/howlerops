import React from 'react';

export interface SelectionBannerProps {
  mode: 'offer' | 'active';
  currentPageCount: number;
  totalCount: number;
  onSelectAllPages: () => void;
  onClearSelection: () => void;
}

export const SelectionBanner: React.FC<SelectionBannerProps> = ({
  mode,
  currentPageCount,
  totalCount,
  onSelectAllPages,
  onClearSelection,
}) => {
  if (mode === 'offer') {
    return (
      <div className="bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-3 mb-4">
        <div className="flex items-center justify-between">
          <span className="text-sm text-blue-700 dark:text-blue-300">
            All {currentPageCount.toLocaleString()} rows on this page are selected.
          </span>
          <button
            onClick={onSelectAllPages}
            className="text-sm font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200 underline"
          >
            Select all {totalCount.toLocaleString()} rows
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-blue-100 dark:bg-blue-900/40 border-l-4 border-blue-600 p-3 mb-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
          All {totalCount.toLocaleString()} rows are selected.
        </span>
        <button
          onClick={onClearSelection}
          className="text-sm font-medium text-blue-700 dark:text-blue-300 hover:text-blue-900 dark:hover:text-blue-100 underline"
        >
          Clear selection
        </button>
      </div>
    </div>
  );
};
