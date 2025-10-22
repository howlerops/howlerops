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
import { QueryEditableMetadata } from '@/store/query-store'
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

  // Extract foreign key fields from row data
  const getForeignKeyFields = useCallback(() => {
    if (!rowData || !connectionId) return []
    
    // Look for potential foreign key fields based on common patterns
    const potentialForeignKeys: Array<{key: string, value: any, tableName: string, columnName: string}> = []
    
    Object.entries(rowData).forEach(([key, value]) => {
      if (value === null || value === undefined) return
      
      // Common foreign key patterns
      const lowerKey = key.toLowerCase()
      
      // Pattern 1: *_id fields (snake_case)
      if (lowerKey.endsWith('_id') && (typeof value === 'string' || typeof value === 'number')) {
        const tableName = lowerKey.replace('_id', '') + 's' // pluralize
        potentialForeignKeys.push({
          key,
          value,
          tableName,
          columnName: 'id'
        })
      }
      
      // Pattern 2: *Id fields (camelCase)
      if (lowerKey.endsWith('id') && lowerKey !== 'id' && (typeof value === 'string' || typeof value === 'number')) {
        const baseName = lowerKey.replace('id', '')
        if (baseName.length > 0) {
          const tableName = baseName + 's' // pluralize
          potentialForeignKeys.push({
            key,
            value,
            tableName,
            columnName: 'id'
          })
        }
      }
      
      // Pattern 3: user_id, account_id, etc. (snake_case with underscore)
      if (lowerKey.includes('_id') && (typeof value === 'string' || typeof value === 'number')) {
        const parts = lowerKey.split('_id')
        if (parts.length === 2 && parts[0].length > 0) {
          const tableName = parts[0] + 's' // pluralize
          potentialForeignKeys.push({
            key,
            value,
            tableName,
            columnName: 'id'
          })
        }
      }
      
      // Pattern 4: Common foreign key prefixes
      const commonPrefixes = ['user', 'account', 'organization', 'project', 'team', 'company', 'customer', 'order', 'product', 'category']
      commonPrefixes.forEach(prefix => {
        if (lowerKey === `${prefix}_id` || lowerKey === `${prefix}id` || lowerKey === `${prefix}Id`) {
          const tableName = prefix + 's' // pluralize
          potentialForeignKeys.push({
            key,
            value,
            tableName,
            columnName: 'id'
          })
        }
      })
      
      // Pattern 5: UUID-like values that might be foreign keys
      if (typeof value === 'string' && value.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i)) {
        // Try common table names
        const commonTables = ['users', 'accounts', 'organizations', 'projects', 'teams', 'companies', 'customers', 'orders', 'products', 'categories']
        commonTables.forEach(tableName => {
          potentialForeignKeys.push({
            key,
            value,
            tableName,
            columnName: 'id'
          })
        })
      }
      
      // Pattern 6: Numeric IDs that might be foreign keys
      if (typeof value === 'number' && value > 0 && value < 1000000) { // Reasonable range for IDs
        // Check if the key suggests it's a foreign key
        if (lowerKey.includes('id') || lowerKey.includes('ref') || lowerKey.includes('fk')) {
          const commonTables = ['users', 'accounts', 'organizations', 'projects', 'teams', 'companies', 'customers', 'orders', 'products', 'categories']
          commonTables.forEach(tableName => {
            potentialForeignKeys.push({
              key,
              value,
              tableName,
              columnName: 'id'
            })
          })
        }
      }
    })
    
    // Remove duplicates based on key
    const uniqueForeignKeys = potentialForeignKeys.filter((fk, index, self) => 
      index === self.findIndex(f => f.key === fk.key)
    )
    
    return uniqueForeignKeys.map(fk => ({
      key: fk.key,
      value: fk.value,
      metadata: metadata!
    }))
  }, [rowData, connectionId, metadata])

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
                {metadata && connectionId && (
                  <ForeignKeySection
                    foreignKeys={getForeignKeyFields()}
                    connectionId={connectionId}
                    expandedKeys={new Set()}
                    onToggleKey={toggleForeignKey}
                    onLoadData={loadForeignKeyData}
                  />
                )}

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
