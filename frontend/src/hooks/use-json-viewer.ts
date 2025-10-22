import { useCallback, useMemo, useEffect } from 'react'
import { useJsonViewerStore } from '../store/json-viewer-store'
import { formatJson, rowToJson, JsonToken } from '../lib/json-formatter'
import { searchJson, SearchResult, isRegexQuery } from '../lib/json-search'
import { TableRow, CellValue } from '../types/table'
import { QueryEditableMetadata } from '../store/query-store'

export interface UseJsonViewerOptions {
  rowData: TableRow | null
  columns?: string[]
  metadata?: QueryEditableMetadata | null
  connectionId?: string
  onSave?: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>
}

export function useJsonViewer({
  rowData,
  columns,
  metadata,
  connectionId,
  onSave
}: UseJsonViewerOptions) {
  const store = useJsonViewerStore()

  // Convert row data to JSON
  const jsonData = useMemo(() => {
    if (!rowData) return null
    return rowToJson(rowData, columns)
  }, [rowData, columns])

  // Format JSON with syntax highlighting
  const formattedJson = useMemo(() => {
    if (!jsonData) return null
    return formatJson(jsonData)
  }, [jsonData])

  // Search functionality
  const searchResult = useMemo((): SearchResult => {
    if (!formattedJson || !store.searchQuery) {
      return {
        matches: [],
        totalMatches: 0,
        currentMatchIndex: -1
      }
    }

    return searchJson(formattedJson.tokens, store.searchQuery, {
      caseSensitive: false,
      useRegex: store.useRegex,
      searchKeys: store.searchKeys,
      searchValues: store.searchValues
    })
  }, [formattedJson, store.searchQuery, store.useRegex, store.searchKeys, store.searchValues])

  // Update search results when search changes
  useEffect(() => {
    if (searchResult.totalMatches > 0) {
      useJsonViewerStore.setState({
        searchResults: {
          matches: searchResult.matches,
          currentIndex: searchResult.currentMatchIndex,
          totalMatches: searchResult.totalMatches
        }
      })
    } else {
      useJsonViewerStore.setState({
        searchResults: {
          matches: [],
          currentIndex: -1,
          totalMatches: 0
        }
      })
    }
  }, [searchResult])

  // Actions
  const openRow = useCallback((rowId: string, row: TableRow) => {
    store.openRow(rowId, row, metadata)
  }, [store, metadata])

  const closeViewer = useCallback(() => {
    store.closeViewer()
  }, [store])

  const toggleEdit = useCallback(() => {
    store.toggleEdit()
  }, [store])

  const updateField = useCallback((key: string, value: CellValue) => {
    store.updateField(key, value)
    
    // Validate field if metadata is available
    if (metadata) {
      store.validateField(key, value, metadata)
    }
  }, [store, metadata])

  const saveChanges = useCallback(async (): Promise<boolean> => {
    if (!onSave || !store.currentRowId || !store.editedData) {
      return false
    }

    return store.saveChanges(onSave)
  }, [store, onSave])

  const handleSearch = useCallback((query: string) => {
    store.setSearchQuery(query)
    
    // Auto-detect regex pattern
    if (isRegexQuery(query)) {
      store.setSearchOptions({ useRegex: true })
    }
  }, [store])

  const navigateToNextMatch = useCallback(() => {
    store.navigateToNextMatch()
  }, [store])

  const navigateToPreviousMatch = useCallback(() => {
    store.navigateToPreviousMatch()
  }, [store])

  const clearSearch = useCallback(() => {
    store.clearSearch()
  }, [store])

  const toggleKeyExpansion = useCallback((key: string) => {
    store.toggleKeyExpansion(key)
  }, [store])

  const expandAllKeys = useCallback(() => {
    store.expandAllKeys()
  }, [store])

  const collapseAllKeys = useCallback(() => {
    store.collapseAllKeys()
  }, [store])

  const toggleWordWrap = useCallback(() => {
    store.toggleWordWrap()
  }, [store])

  const toggleForeignKey = useCallback((key: string) => {
    store.toggleForeignKey(key)
  }, [store])

  const loadForeignKeyData = useCallback(async (key: string) => {
    if (!connectionId) return
    
    // This would integrate with the existing query system
    // For now, we'll use a placeholder
    await store.loadForeignKeyData(key, connectionId, '')
  }, [store, connectionId])

  // Computed values
  const hasChanges = useMemo(() => {
    if (!store.isEditing || !store.editedData || !rowData) return false
    
    return Object.keys(store.editedData).some(key => {
      if (key === '__rowId') return false
      return store.editedData![key] !== rowData[key]
    })
  }, [store.isEditing, store.editedData, rowData])

  const hasValidationErrors = useMemo(() => {
    return store.validationErrors.size > 0
  }, [store.validationErrors])

  const canSave = useMemo(() => {
    return store.isEditing && hasChanges && !hasValidationErrors && !store.isSaving
  }, [store.isEditing, hasChanges, hasValidationErrors, store.isSaving])

  const currentMatch = useMemo(() => {
    if (store.searchResults.currentIndex < 0 || store.searchResults.currentIndex >= store.searchResults.totalMatches) {
      return null
    }
    return store.searchResults.matches[store.searchResults.currentIndex]
  }, [store.searchResults])

  const isKeyExpanded = useCallback((key: string) => {
    if (store.expandedKeys.has('*')) return true
    if (store.collapsedKeys.has('*')) return false
    return store.expandedKeys.has(key)
  }, [store.expandedKeys, store.collapsedKeys])

  const isForeignKeyExpanded = useCallback((key: string) => {
    return store.expandedForeignKeys.has(key)
  }, [store.expandedForeignKeys])

  return {
    // State
    isOpen: store.isOpen,
    currentRow: store.currentRow,
    currentRowId: store.currentRowId,
    isEditing: store.isEditing,
    editedData: store.editedData,
    validationErrors: store.validationErrors,
    wordWrap: store.wordWrap,
    searchQuery: store.searchQuery,
    searchResults: store.searchResults,
    useRegex: store.useRegex,
    searchKeys: store.searchKeys,
    searchValues: store.searchValues,
    isLoading: store.isLoading,
    isSaving: store.isSaving,
    saveError: store.saveError,
    
    // Computed
    jsonData,
    formattedJson,
    hasChanges,
    hasValidationErrors,
    canSave,
    currentMatch,
    
    // Actions
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
    
    // Search options
    setSearchOptions: store.setSearchOptions,
    
    // Utility functions
    getTokenClass: (token: JsonToken) => {
      // This would be imported from json-formatter
      switch (token.type) {
        case 'key':
          return 'text-blue-600 font-medium'
        case 'string':
          return 'text-green-600'
        case 'number':
          return 'text-purple-600'
        case 'boolean':
          return 'text-orange-600'
        case 'null':
          return 'text-gray-500'
        case 'punctuation':
          return 'text-gray-700'
        case 'whitespace':
          return ''
        default:
          return 'text-gray-900'
      }
    }
  }
}
