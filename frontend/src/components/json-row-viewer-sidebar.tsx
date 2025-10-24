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
  Edit,
  Expand,
  ChevronDown,
  WrapText,
  Database,
  AlertCircle,
  Loader2,
  CheckCircle2
} from 'lucide-react'
import { TableRow, CellValue } from '@/types/table'
import { QueryEditableMetadata, QueryEditableColumn } from '@/store/query-store'
import { useJsonViewer } from '@/hooks/use-json-viewer'
import { JsonEditor } from './json-editor'
import { ForeignKeySection } from './foreign-key-card'
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
    expandedForeignKeys,
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
  }, [open, rowData, rowId])

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

  // Simple pluralization helper
  const pluralize = useCallback((word: string): string => {
    // Common irregular plurals
    const irregulars: Record<string, string> = {
      'person': 'people',
      'child': 'children',
      'category': 'categories',
      'company': 'companies'
    }

    if (irregulars[word]) return irregulars[word]

    // Simple pluralization rules
    if (word.endsWith('y') && !['a', 'e', 'i', 'o', 'u'].includes(word[word.length - 2])) {
      return word.slice(0, -1) + 'ies'
    }
    if (word.endsWith('s') || word.endsWith('x') || word.endsWith('z') || word.endsWith('ch') || word.endsWith('sh')) {
      return word + 'es'
    }
    return word + 's'
  }, [])

  // Extract foreign key fields from row data
  const getForeignKeyFields = useCallback(() => {
    if (!rowData) return []

    const foreignKeys: Array<{key: string, value: CellValue, metadata: QueryEditableMetadata}> = []
    const addedKeys = new Set<string>() // Track added keys to avoid duplicates

    // Strategy 1: Use actual foreign key metadata from the query result (most accurate)
    if (metadata?.columns) {
      metadata.columns.forEach(column => {
        const columnName = column.resultName || column.name
        if (column.foreignKey && columnName in rowData) {
          const value = rowData[columnName]
          // Only include if the value exists and isn't null
          if (value !== null && value !== undefined && !addedKeys.has(columnName)) {
            // Create a metadata object with this specific FK info for the ForeignKeyCard
            const fkMetadata: QueryEditableMetadata = {
              ...metadata,
              columns: [column] // Only pass the relevant column
            }
            foreignKeys.push({
              key: columnName,
              value,
              metadata: fkMetadata
            })
            addedKeys.add(columnName)
          }
        }
      })
    }

    // Strategy 2: Pattern-based detection for common foreign key naming conventions
    // Only use this if we didn't find any FK metadata
    if (foreignKeys.length === 0) {
      const detectedKeys = new Map<string, {tableName: string, columnName: string}>()

      Object.entries(rowData).forEach(([key, value]) => {
        if (value === null || value === undefined || key === '__rowId') return
        if (!(typeof value === 'string' || typeof value === 'number')) return

        const lowerKey = key.toLowerCase()

        // Pattern 1: Explicit *_id suffix (snake_case)
        if (lowerKey.endsWith('_id') && !detectedKeys.has(key)) {
          const baseName = lowerKey.slice(0, -3) // Remove '_id'
          if (baseName.length > 0) {
            const tableName = pluralize(baseName)
            detectedKeys.set(key, { tableName, columnName: 'id' })
          }
        }

        // Pattern 2: CamelCase *Id suffix
        else if (lowerKey.endsWith('id') && lowerKey !== 'id' && lowerKey.length > 2 && !detectedKeys.has(key)) {
          const baseName = lowerKey.slice(0, -2) // Remove 'id'
          if (baseName.length > 0 && !lowerKey.includes('_')) {
            const tableName = pluralize(baseName)
            detectedKeys.set(key, { tableName, columnName: 'id' })
          }
        }
      })

      // Convert detected keys to foreign key objects
      detectedKeys.forEach((fkInfo, key) => {
        const value = rowData[key]

        // Create synthetic metadata with foreign key info
        const syntheticColumn: QueryEditableColumn = {
          name: key,
          resultName: key,
          dataType: typeof value === 'number' ? 'integer' : 'text',
          editable: false,
          primaryKey: false,
          foreignKey: {
            table: fkInfo.tableName,
            column: fkInfo.columnName,
            schema: metadata?.schema
          }
        }

        const fkMetadata: QueryEditableMetadata = {
          enabled: false,
          schema: metadata?.schema,
          table: metadata?.table,
          primaryKeys: [],
          columns: [syntheticColumn]
        }

        foreignKeys.push({
          key,
          value,
          metadata: fkMetadata
        })
      })
    }

    return foreignKeys
  }, [rowData, metadata, pluralize])

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
      <SheetContent side="right" className="w-full sm:max-w-2xl m-4 h-[calc(100vh-2rem)] rounded-xl shadow-2xl border overflow-hidden flex flex-col">
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
                    <Eye className="h-3 w-3 mr-1" />
                    Switch to View
                  </>
                ) : (
                  <>
                    <Edit className="h-3 w-3 mr-1" />
                    Switch to Edit
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

          {/* Content */}
          <div className="flex-1 min-h-0">
            <ScrollArea className="h-full">
              <div className="p-4 space-y-4">
                {/* Foreign Key Relationships */}
                {(() => {
                  const fkFields = getForeignKeyFields()
                  return fkFields.length > 0 && (
                    <ForeignKeySection
                      foreignKeys={fkFields}
                      connectionId={connectionId || ''}
                      expandedKeys={expandedForeignKeys}
                      onToggleKey={toggleForeignKey}
                      onLoadData={loadForeignKeyData}
                    />
                  )
                })()}

                {/* JSON Content */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-sm font-medium">Row Data</h3>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleCopyJson}
                      className="h-6 px-2 text-xs"
                    >
                      <Copy className="h-3 w-3 mr-1" />
                      Copy JSON
                    </Button>
                  </div>
                  <JsonEditor
                    tokens={formattedJson.tokens}
                    data={jsonData! as Record<string, CellValue>}
                    isEditing={isEditing}
                    validationErrors={validationErrors}
                    searchMatches={searchResults.matches}
                    currentMatchIndex={searchResults.currentIndex}
                    wordWrap={wordWrap}
                    expandedKeys={new Set(['*'])}
                    collapsedKeys={new Set()}
                    onToggleEdit={toggleEdit}
                    onUpdateField={updateField}
                    onToggleKeyExpansion={toggleKeyExpansion}
                    onCopyJson={handleCopyJson}
                    metadata={metadata}
                    connectionId={connectionId}
                  />
                </div>
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
