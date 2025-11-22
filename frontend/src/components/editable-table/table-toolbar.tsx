import {
  Copy,
  Download,
  Filter,
  MoreHorizontal,
  Redo,
  RefreshCw,
  Trash,
  Undo,
  Upload} from 'lucide-react';
import React, { useCallback, useMemo, useState } from 'react';

import { ExportOptions,TableToolbarProps } from '../../types/table';
import { cn } from '../../utils/cn';
import { debounce } from '../../utils/table';

export const TableToolbar: React.FC<TableToolbarProps> = ({
  searchValue,
  onSearchChange,
  onExport,
  onImport,
  onRefresh,
  selectedCount,
  totalCount,
  loading = false,
  showExport = true,
  showImport = false,
  customActions,
}) => {
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    includeHeaders: true,
    selectedOnly: false,
  });

  // Debounced search to avoid excessive API calls
  const debouncedSearch = useMemo(
    () => debounce((value: unknown) => {
      onSearchChange(value as string);
    }, 300),
    [onSearchChange]
  );

  // Search is currently disabled in UI, but function kept for future use
  const _handleSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    debouncedSearch(value);
  }, [debouncedSearch]);

  const handleExport = useCallback((format: 'csv' | 'json' | 'xlsx') => {
    const options: ExportOptions = {
      ...exportOptions,
      format,
    };
    onExport?.(options);
    setShowExportMenu(false);
  }, [exportOptions, onExport]);

  const handleExportOptionChange = useCallback((
    key: keyof ExportOptions,
    value: boolean | string
  ) => {
    setExportOptions(prev => ({ ...prev, [key]: value }));
  }, []);

  return (
    <div className="flex items-center justify-between p-4 bg-background border-b border-border">
      {/* Left side - Search and filters */}
      <div className="flex items-center gap-4 flex-1">
        {/* Search */}
        {/* <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search table..."
            defaultValue={searchValue}
            onChange={handleSearchChange}
            className="pl-10 pr-4 py-2 w-64 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent"
          />
        </div> */}

        {/* Filter indicator */}
        {searchValue && (
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-primary" />
            <span className="text-sm text-muted-foreground">
              Filtered
            </span>
          </div>
        )}

        {/* Selection info */}
        {selectedCount > 0 && (
          <div className="flex items-center gap-2 px-3 py-1 bg-primary/10 border border-primary rounded">
            <span className="text-sm text-primary font-medium">
              {selectedCount} selected
            </span>
            <button
              className="text-primary hover:text-primary/80"
              title="Clear selection"
            >
              ×
            </button>
          </div>
        )}
      </div>

      {/* Right side - Actions */}
      <div className="flex items-center gap-2">
        {/* Custom actions */}
        {customActions}

        {/* Bulk actions for selected rows */}
        {selectedCount > 0 && (
          <>
            <button
              className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors"
              title="Copy selected"
            >
              <Copy className="h-4 w-4" />
            </button>
            <button
              className="p-2 text-destructive hover:text-destructive/80 hover:bg-destructive/10 rounded transition-colors"
              title="Delete selected"
            >
              <Trash className="h-4 w-4" />
            </button>
            <div className="w-px h-6 bg-border mx-1" />
          </>
        )}

        {/* Undo/Redo */}
        <button
          className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title="Undo"
          disabled={loading}
        >
          <Undo className="h-4 w-4" />
        </button>
        <button
          className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title="Redo"
          disabled={loading}
        >
          <Redo className="h-4 w-4" />
        </button>

        <div className="w-px h-6 bg-border mx-1" />

        {/* Import */}
        {showImport && (
          <button
            onClick={onImport}
            disabled={loading}
            className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Import data"
          >
            <Upload className="h-4 w-4" />
          </button>
        )}

        {/* Export */}
        {showExport && (
          <div className="relative">
            <button
              onClick={() => setShowExportMenu(!showExportMenu)}
              disabled={loading || totalCount === 0}
              className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Export data"
            >
              <Download className="h-4 w-4" />
            </button>

            {/* Export menu */}
            {showExportMenu && (
              <div className="absolute right-0 top-full mt-1 w-64 bg-card border border-border rounded-md shadow-lg z-50">
                <div className="p-3">
                  <h3 className="font-medium text-foreground mb-3">Export Options</h3>

                  {/* Format selection */}
                  <div className="mb-3">
                    <label className="block text-sm font-medium text-foreground mb-1">
                      Format
                    </label>
                    <select
                      value={exportOptions.format}
                      onChange={(e) => handleExportOptionChange('format', e.target.value)}
                      className="w-full px-3 py-1 border border-border rounded focus:outline-none focus:ring-2 focus:ring-ring"
                    >
                      <option value="csv">CSV</option>
                      <option value="json">JSON</option>
                      <option value="xlsx" disabled>Excel (Coming soon)</option>
                    </select>
                  </div>

                  {/* Options */}
                  <div className="space-y-2 mb-3">
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={exportOptions.includeHeaders}
                        onChange={(e) => handleExportOptionChange('includeHeaders', e.target.checked)}
                        className="rounded border-border focus:ring-2 focus:ring-ring"
                      />
                      <span className="ml-2 text-sm text-foreground">Include headers</span>
                    </label>
                    {selectedCount > 0 && (
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={exportOptions.selectedOnly}
                          onChange={(e) => handleExportOptionChange('selectedOnly', e.target.checked)}
                          className="rounded border-border focus:ring-2 focus:ring-ring"
                        />
                        <span className="ml-2 text-sm text-foreground">
                          Selected only ({selectedCount} rows)
                        </span>
                      </label>
                    )}
                  </div>

                  {/* Export buttons */}
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleExport(exportOptions.format)}
                      className="flex-1 px-3 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors text-sm"
                    >
                      Export
                    </button>
                    <button
                      onClick={() => setShowExportMenu(false)}
                      className="px-3 py-2 border border-border rounded hover:bg-muted/50 transition-colors text-sm"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Refresh */}
        <button
          onClick={onRefresh}
          disabled={loading}
          className={cn(
            'p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            loading && 'animate-spin'
          )}
          title="Refresh data"
        >
          <RefreshCw className="h-4 w-4" />
        </button>

        {/* More actions */}
        <button
          className="p-2 text-foreground hover:text-foreground/80 hover:bg-muted rounded transition-colors"
          title="More actions"
        >
          <MoreHorizontal className="h-4 w-4" />
        </button>
      </div>

      {/* Click outside to close export menu */}
      {showExportMenu && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => setShowExportMenu(false)}
        />
      )}
    </div>
  );
};