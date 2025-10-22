import React, { useCallback, useMemo } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { ErrorBoundary } from '@/components/error-boundary'
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Copy,
  Save,
  X,
  Eye,
  EyeOff,
  Expand,
  ChevronDown,
  WrapText,
  Database,
  AlertCircle,
  Loader2,
  CheckCircle2
} from 'lucide-react'
import { TableRow, CellValue } from '@/types/table'
import { QueryEditableMetadata } from '@/store/query-store'
import { useJsonViewer } from '@/hooks/use-json-viewer'
import { JsonEditor } from './json-editor'
import { isRegexQuery } from '@/lib/json-search'

interface JsonRowViewerSidebarProps {
  open: boolean
  onClose: () => void
  rowData: TableRow | null
  rowId: string | null
  columns?: string[]
  metadata?: QueryEditableMetadata | null
  connectionId?: string
  onSave?: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>
}

export function JsonRowViewerSidebar({
  open,
  onClose,
  rowData,
  rowId,
  columns,
  metadata,
  connectionId,
  onSave
}: JsonRowViewerSidebarProps) {
  const {
    isOpen,
    currentRow,
    currentRowId,
    isEditing,
    editedData,
    validationErrors,
    wordWrap,
    searchQuery,
    searchResults,
    useRegex,
    searchKeys,
    searchValues,
    isLoading,
    isSaving,
    saveError,
    jsonData,
    formattedJson,
    hasChanges,
    hasValidationErrors,
    canSave,
    currentMatch,
    openRow,
    closeViewer,
    toggleEdit,
    updateField,
    saveChanges,
    handleSearch,
    navigateToNextMatch,
    navigateToPreviousMatch,
    clearSearch,
    toggleKeyExpansion,
    expandAllKeys,
    collapseAllKeys,
    toggleWordWrap,
    toggleForeignKey,
    loadForeignKeyData,
    isKeyExpanded,
    isForeignKeyExpanded,
    setSearchOptions,
    getTokenClass
  } = useJsonViewer({
    rowData,
    columns,
    metadata,
    connectionId,
    onSave
  })

  // Open row when props change
  React.useEffect(() => {
    if (open && rowData && rowId) {
      openRow(rowId, rowData)
    } else if (!open) {
      closeViewer()
    }
  }, [open, rowData, rowId, openRow, closeViewer])

  const handleSave = useCallback(async () => {
    if (!onSave) return
    
    const success = await saveChanges()
    if (success) {
      // Success feedback could be shown here
    }
  }, [saveChanges, onSave])

  const handleCopyJson = useCallback(() => {
    if (!formattedJson) return
    
    navigator.clipboard.writeText(formattedJson.formatted)
      .then(() => {
        // Success feedback could be shown here
      })
      .catch(err => {
        console.error('Failed to copy JSON:', err)
      })
  }, [formattedJson])

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const query = e.target.value
    handleSearch(query)
    
    // Auto-detect regex
    if (isRegexQuery(query)) {
      setSearchOptions({ useRegex: true })
    }
  }, [handleSearch, setSearchOptions])

  const handleSearchKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      if (e.shiftKey) {
        navigateToPreviousMatch()
      } else {
        navigateToNextMatch()
      }
    }
  }, [navigateToNextMatch, navigateToPreviousMatch])

  const searchStats = useMemo(() => {
    return {
      totalMatches: searchResults.totalMatches,
      currentMatch: searchResults.currentIndex + 1,
      hasMatches: searchResults.totalMatches > 0
    }
  }, [searchResults])

  if (!isOpen || !currentRow || !formattedJson) {
    return null
  }

  return (
    <ErrorBoundary
      fallback={
        <div className="p-4 text-center">
          <AlertCircle className="h-8 w-8 mx-auto mb-2 text-destructive" />
          <p className="text-sm text-muted-foreground mb-4">
            Error loading JSON viewer
          </p>
          <Button variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      }
      onError={(error, errorInfo) => {
        console.error('JSON Row Viewer Sidebar error:', error, errorInfo)
      }}
    >
      <Sheet open={isOpen} onOpenChange={closeViewer}>
      <SheetContent side="right" className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <Database className="h-4 w-4" />
            Row Details
            {currentRowId && (
              <Badge variant="outline" className="text-xs">
                {currentRowId}
              </Badge>
            )}
          </SheetTitle>
          <SheetDescription>
            View and edit row data as JSON with foreign key relationships
          </SheetDescription>
        </SheetHeader>

        <div className="flex flex-col h-full">
          {/* Toolbar */}
          <div className="flex items-center justify-between p-4 border-b">
            <div className="flex items-center gap-2">
              {/* Edit Mode Toggle */}
              <Button
                variant={isEditing ? "default" : "outline"}
                size="sm"
                onClick={toggleEdit}
                disabled={isSaving}
              >
                {isEditing ? (
                  <>
                    <EyeOff className="h-3 w-3 mr-1" />
                    Edit Mode
                  </>
                ) : (
                  <>
                    <Eye className="h-3 w-3 mr-1" />
                    View Mode
                  </>
                )}
              </Button>

              {/* Save Button */}
              {isEditing && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={handleSave}
                  disabled={!canSave}
                >
                  {isSaving ? (
                    <>
                      <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    <>
                      <Save className="h-3 w-3 mr-1" />
                      Save Changes
                    </>
                  )}
                </Button>
              )}

              {/* Copy JSON */}
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopyJson}
              >
                <Copy className="h-3 w-3 mr-1" />
                Copy JSON
              </Button>
            </div>

            <div className="flex items-center gap-2">
              {/* Word Wrap Toggle */}
              <Button
                variant="outline"
                size="sm"
                onClick={toggleWordWrap}
                className={wordWrap ? "bg-muted" : ""}
              >
                <WrapText className="h-3 w-3" />
              </Button>

              {/* Expand/Collapse All */}
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={expandAllKeys}
                  title="Expand All"
                >
                  <Expand className="h-3 w-3" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={collapseAllKeys}
                  title="Collapse All"
                >
                  <ChevronDown className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>

          {/* Search Bar */}
          <div className="p-4 border-b">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search by key name or use /regex/ pattern..."
                value={searchQuery}
                onChange={handleSearchChange}
                onKeyDown={handleSearchKeyDown}
                className="pl-9 pr-20"
              />
              {searchStats.hasMatches && (
                <div className="absolute right-2 top-1/2 transform -translate-y-1/2 flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={navigateToPreviousMatch}
                    className="h-6 w-6 p-0"
                  >
                    <ChevronLeft className="h-3 w-3" />
                  </Button>
                  <span className="text-xs text-muted-foreground">
                    {searchStats.currentMatch}/{searchStats.totalMatches}
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={navigateToNextMatch}
                    className="h-6 w-6 p-0"
                  >
                    <ChevronRight className="h-3 w-3" />
                  </Button>
                </div>
              )}
            </div>

            {/* Search Options */}
            <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  checked={useRegex}
                  onChange={(e) => setSearchOptions({ useRegex: e.target.checked })}
                  className="rounded"
                />
                Regex
              </label>
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  checked={searchKeys}
                  onChange={(e) => setSearchOptions({ searchKeys: e.target.checked })}
                  className="rounded"
                />
                Keys
              </label>
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  checked={searchValues}
                  onChange={(e) => setSearchOptions({ searchValues: e.target.checked })}
                  className="rounded"
                />
                Values
              </label>
              {searchQuery && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={clearSearch}
                  className="h-6 px-2 text-xs"
                >
                  Clear
                </Button>
              )}
            </div>
          </div>

          {/* Status Messages */}
          {(hasChanges || hasValidationErrors || saveError) && (
            <div className="p-4 border-b">
              {hasChanges && (
                <div className="flex items-center gap-2 text-sm text-amber-600 bg-amber-50 p-2 rounded">
                  <AlertCircle className="h-4 w-4" />
                  Unsaved changes
                </div>
              )}
              {hasValidationErrors && (
                <div className="flex items-center gap-2 text-sm text-red-600 bg-red-50 p-2 rounded">
                  <AlertCircle className="h-4 w-4" />
                  {validationErrors.size} validation error{validationErrors.size === 1 ? '' : 's'}
                </div>
              )}
              {saveError && (
                <div className="flex items-center gap-2 text-sm text-red-600 bg-red-50 p-2 rounded">
                  <AlertCircle className="h-4 w-4" />
                  {saveError}
                </div>
              )}
            </div>
          )}

          {/* JSON Content */}
          <div className="flex-1 min-h-0">
            <ScrollArea className="h-full">
              <div className="p-4">
                <JsonEditor
                  tokens={formattedJson.tokens}
                  data={jsonData!}
                  isEditing={isEditing}
                  validationErrors={validationErrors}
                  searchMatches={searchResults.matches}
                  currentMatchIndex={searchResults.currentIndex}
                  wordWrap={wordWrap}
                  expandedKeys={new Set()}
                  collapsedKeys={new Set()}
                  onToggleEdit={toggleEdit}
                  onUpdateField={updateField}
                  onToggleKeyExpansion={toggleKeyExpansion}
                  onCopyJson={handleCopyJson}
                  metadata={metadata}
                  connectionId={connectionId}
                />
              </div>
            </ScrollArea>
          </div>

          {/* Footer */}
          <div className="p-4 border-t bg-muted/30">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-4">
                <span>
                  {Object.keys(jsonData || {}).length} fields
                </span>
                {formattedJson.hasCircularRefs && (
                  <Badge variant="outline" className="text-xs">
                    Circular References Detected
                  </Badge>
                )}
              </div>
              <div className="flex items-center gap-2">
                {isEditing && (
                  <span className="text-amber-600">
                    Edit Mode
                  </span>
                )}
                {isLoading && (
                  <span className="flex items-center gap-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Loading...
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
    </ErrorBoundary>
  )
}
