import { create } from 'zustand'
import { TableRow, CellValue } from '../types/table'
import { QueryEditableMetadata } from '../store/query-store'

export interface JsonViewerState {
  // Sidebar state
  isOpen: boolean
  currentRow: TableRow | null
  currentRowId: string | null
  
  // Edit mode
  isEditing: boolean
  editedData: Record<string, CellValue> | null
  validationErrors: Map<string, string>
  
  // Display options
  wordWrap: boolean
  expandedKeys: Set<string>
  collapsedKeys: Set<string>
  
  // Search
  searchQuery: string
  searchResults: {
    matches: any[]
    currentIndex: number
    totalMatches: number
  }
  useRegex: boolean
  searchKeys: boolean
  searchValues: boolean
  
  // Foreign key expansion
  expandedForeignKeys: Set<string>
  foreignKeyCache: Map<string, any>
  
  // Loading states
  isLoading: boolean
  isSaving: boolean
  saveError: string | null
}

export interface JsonViewerActions {
  // Sidebar control
  openRow: (rowId: string, rowData: TableRow, metadata?: QueryEditableMetadata | null) => void
  closeViewer: () => void
  
  // Edit mode
  toggleEdit: () => void
  startEdit: () => void
  cancelEdit: () => void
  updateField: (key: string, value: CellValue) => void
  validateField: (key: string, value: CellValue, metadata?: QueryEditableMetadata | null) => boolean
  
  // Display options
  toggleWordWrap: () => void
  toggleKeyExpansion: (key: string) => void
  expandAllKeys: () => void
  collapseAllKeys: () => void
  
  // Search
  setSearchQuery: (query: string) => void
  setSearchOptions: (options: { useRegex?: boolean; searchKeys?: boolean; searchValues?: boolean }) => void
  navigateToNextMatch: () => void
  navigateToPreviousMatch: () => void
  clearSearch: () => void
  
  // Foreign keys
  toggleForeignKey: (key: string) => void
  loadForeignKeyData: (key: string, connectionId: string, query: string) => Promise<void>
  clearForeignKeyCache: () => void
  
  // Save operations
  saveChanges: (onSave: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>) => Promise<boolean>
  setLoading: (loading: boolean) => void
  setSaving: (saving: boolean) => void
  setSaveError: (error: string | null) => void
  
  // Reset
  reset: () => void
}

const initialState: JsonViewerState = {
  isOpen: false,
  currentRow: null,
  currentRowId: null,
  isEditing: false,
  editedData: null,
  validationErrors: new Map(),
  wordWrap: true,
  expandedKeys: new Set(),
  collapsedKeys: new Set(),
  searchQuery: '',
  searchResults: {
    matches: [],
    currentIndex: -1,
    totalMatches: 0
  },
  useRegex: false,
  searchKeys: true,
  searchValues: true,
  expandedForeignKeys: new Set(),
  foreignKeyCache: new Map(),
  isLoading: false,
  isSaving: false,
  saveError: null
}

export const useJsonViewerStore = create<JsonViewerState & JsonViewerActions>((set, get) => ({
  ...initialState,

  // Sidebar control
  openRow: (rowId: string, rowData: TableRow, metadata?: QueryEditableMetadata | null) => {
    set({
      isOpen: true,
      currentRow: rowData,
      currentRowId: rowId,
      editedData: null,
      validationErrors: new Map(),
      isEditing: false,
      expandedKeys: new Set(),
      collapsedKeys: new Set(),
      searchQuery: '',
      searchResults: {
        matches: [],
        currentIndex: -1,
        totalMatches: 0
      },
      expandedForeignKeys: new Set(),
      saveError: null
    })
  },

  closeViewer: () => {
    set({
      isOpen: false,
      currentRow: null,
      currentRowId: null,
      editedData: null,
      validationErrors: new Map(),
      isEditing: false,
      searchQuery: '',
      searchResults: {
        matches: [],
        currentIndex: -1,
        totalMatches: 0
      },
      expandedForeignKeys: new Set(),
      saveError: null
    })
  },

  // Edit mode
  toggleEdit: () => {
    const { isEditing, currentRow } = get()
    if (!currentRow) return

    if (isEditing) {
      // Cancel edit
      set({
        isEditing: false,
        editedData: null,
        validationErrors: new Map()
      })
    } else {
      // Start edit
      set({
        isEditing: true,
        editedData: { ...currentRow }
      })
    }
  },

  startEdit: () => {
    const { currentRow } = get()
    if (!currentRow) return

    set({
      isEditing: true,
      editedData: { ...currentRow },
      validationErrors: new Map()
    })
  },

  cancelEdit: () => {
    set({
      isEditing: false,
      editedData: null,
      validationErrors: new Map()
    })
  },

  updateField: (key: string, value: CellValue) => {
    const { editedData, validationErrors } = get()
    if (!editedData) return

    const newEditedData = { ...editedData, [key]: value }
    const newValidationErrors = new Map(validationErrors)
    
    // Clear validation error for this field
    newValidationErrors.delete(key)

    set({
      editedData: newEditedData,
      validationErrors: newValidationErrors
    })
  },

  validateField: (key: string, value: CellValue, metadata?: QueryEditableMetadata | null) => {
    const { validationErrors } = get()
    const newValidationErrors = new Map(validationErrors)

    // Basic validation
    if (value === null || value === undefined || value === '') {
      // Check if field is required
      const column = metadata?.columns?.find(col => 
        (col.name || col.resultName)?.toLowerCase() === key.toLowerCase()
      )
      
      if (column?.required) {
        newValidationErrors.set(key, 'This field is required')
        set({ validationErrors: newValidationErrors })
        return false
      }
    }

    // Type validation
    const column = metadata?.columns?.find(col => 
      (col.name || col.resultName)?.toLowerCase() === key.toLowerCase()
    )

    if (column) {
      const dataType = column.dataType?.toLowerCase() || ''
      
      if (dataType.includes('int') || dataType.includes('numeric') || dataType.includes('decimal')) {
        if (value !== null && value !== undefined && value !== '' && isNaN(Number(value))) {
          newValidationErrors.set(key, 'Must be a valid number')
          set({ validationErrors: newValidationErrors })
          return false
        }
      }
      
      if (dataType.includes('bool')) {
        if (value !== null && value !== undefined && value !== '' && 
            value !== true && value !== false && 
            value !== 'true' && value !== 'false' && 
            value !== '1' && value !== '0') {
          newValidationErrors.set(key, 'Must be a valid boolean')
          set({ validationErrors: newValidationErrors })
          return false
        }
      }
    }

    // Clear any existing error
    newValidationErrors.delete(key)
    set({ validationErrors: newValidationErrors })
    return true
  },

  // Display options
  toggleWordWrap: () => {
    set(state => ({ wordWrap: !state.wordWrap }))
  },

  toggleKeyExpansion: (key: string) => {
    set(state => {
      const newExpandedKeys = new Set(state.expandedKeys)
      const newCollapsedKeys = new Set(state.collapsedKeys)

      if (newExpandedKeys.has(key)) {
        newExpandedKeys.delete(key)
        newCollapsedKeys.add(key)
      } else {
        newExpandedKeys.add(key)
        newCollapsedKeys.delete(key)
      }

      return {
        expandedKeys: newExpandedKeys,
        collapsedKeys: newCollapsedKeys
      }
    })
  },

  expandAllKeys: () => {
    set(state => ({
      expandedKeys: new Set(['*']), // Use special key to indicate all expanded
      collapsedKeys: new Set()
    }))
  },

  collapseAllKeys: () => {
    set(state => ({
      expandedKeys: new Set(),
      collapsedKeys: new Set(['*']) // Use special key to indicate all collapsed
    }))
  },

  // Search
  setSearchQuery: (query: string) => {
    set({ searchQuery: query })
  },

  setSearchOptions: (options: { useRegex?: boolean; searchKeys?: boolean; searchValues?: boolean }) => {
    set(state => ({
      useRegex: options.useRegex ?? state.useRegex,
      searchKeys: options.searchKeys ?? state.searchKeys,
      searchValues: options.searchValues ?? state.searchValues
    }))
  },

  navigateToNextMatch: () => {
    set(state => {
      const { searchResults } = state
      if (searchResults.totalMatches === 0) return state

      const nextIndex = (searchResults.currentIndex + 1) % searchResults.totalMatches
      return {
        searchResults: {
          ...searchResults,
          currentIndex: nextIndex
        }
      }
    })
  },

  navigateToPreviousMatch: () => {
    set(state => {
      const { searchResults } = state
      if (searchResults.totalMatches === 0) return state

      const prevIndex = searchResults.currentIndex === 0 
        ? searchResults.totalMatches - 1 
        : searchResults.currentIndex - 1

      return {
        searchResults: {
          ...searchResults,
          currentIndex: prevIndex
        }
      }
    })
  },

  clearSearch: () => {
    set({
      searchQuery: '',
      searchResults: {
        matches: [],
        currentIndex: -1,
        totalMatches: 0
      }
    })
  },

  // Foreign keys
  toggleForeignKey: (key: string) => {
    set(state => {
      const newExpandedForeignKeys = new Set(state.expandedForeignKeys)
      
      if (newExpandedForeignKeys.has(key)) {
        newExpandedForeignKeys.delete(key)
      } else {
        newExpandedForeignKeys.add(key)
      }

      return { expandedForeignKeys: newExpandedForeignKeys }
    })
  },

  loadForeignKeyData: async (key: string, connectionId: string, query: string) => {
    // This would integrate with the existing query system
    // For now, we'll just mark it as loaded
    set(state => {
      const newCache = new Map(state.foreignKeyCache)
      newCache.set(key, { loaded: true, timestamp: Date.now() })
      return { foreignKeyCache: newCache }
    })
  },

  clearForeignKeyCache: () => {
    set({ foreignKeyCache: new Map() })
  },

  // Save operations
  saveChanges: async (onSave: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>) => {
    const { currentRowId, editedData, validationErrors } = get()
    
    if (!currentRowId || !editedData) return false
    
    // Check for validation errors
    if (validationErrors.size > 0) {
      set({ saveError: 'Please fix validation errors before saving' })
      return false
    }

    set({ isSaving: true, saveError: null })

    try {
      const success = await onSave(currentRowId, editedData)
      
      if (success) {
        set({
          isEditing: false,
          editedData: null,
          validationErrors: new Map(),
          currentRow: editedData as TableRow
        })
      } else {
        set({ saveError: 'Failed to save changes' })
      }
      
      return success
    } catch (error) {
      set({ saveError: error instanceof Error ? error.message : 'Failed to save changes' })
      return false
    } finally {
      set({ isSaving: false })
    }
  },

  setLoading: (loading: boolean) => {
    set({ isLoading: loading })
  },

  setSaving: (saving: boolean) => {
    set({ isSaving: saving })
  },

  setSaveError: (error: string | null) => {
    set({ saveError: error })
  },

  // Reset
  reset: () => {
    set(initialState)
  }
}))
