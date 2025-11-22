/**
 * Saved Queries Store
 *
 * Zustand store for managing saved query state
 *
 * Features:
 * - CRUD operations with repository integration
 * - Search and filter state
 * - Tier-aware limit checking
 * - Optimistic updates with rollback
 *
 * @module store/saved-queries-store
 */

import { useEffect } from 'react'
import { create } from 'zustand'

import {
  getSavedQueryRepository,
  type SavedQuerySearchOptions,
} from '@/lib/storage'
import type { SavedQueryRecord } from '@/types/storage'

import { useTierStore } from './tier-store'

interface SavedQueriesState {
  // Data
  queries: SavedQueryRecord[]
  totalCount: number
  isLoading: boolean
  error: string | null

  // Search & filter state
  searchText: string
  selectedFolder: string | null
  selectedTags: string[]
  showFavoritesOnly: boolean
  sortBy: 'title' | 'created_at' | 'updated_at'
  sortDirection: 'asc' | 'desc'

  // Metadata
  folders: string[]
  tags: string[]
  isInitialized: boolean
}

interface SavedQueriesActions {
  // Data loading
  loadQueries: (userId: string) => Promise<void>
  loadMetadata: (userId: string) => Promise<void>
  refresh: (userId: string) => Promise<void>

  // CRUD operations
  saveQuery: (data: {
    user_id: string
    title: string
    description?: string
    query_text: string
    tags?: string[]
    folder?: string
    is_favorite?: boolean
  }) => Promise<SavedQueryRecord>
  updateQuery: (
    id: string,
    updates: {
      title?: string
      description?: string
      query_text?: string
      tags?: string[]
      folder?: string
      is_favorite?: boolean
    }
  ) => Promise<void>
  deleteQuery: (id: string) => Promise<void>
  duplicateQuery: (id: string) => Promise<SavedQueryRecord>
  toggleFavorite: (id: string) => Promise<void>

  // Search & filter
  setSearchText: (text: string) => void
  setSelectedFolder: (folder: string | null) => void
  setSelectedTags: (tags: string[]) => void
  toggleTag: (tag: string) => void
  setShowFavoritesOnly: (show: boolean) => void
  setSortBy: (sortBy: 'title' | 'created_at' | 'updated_at') => void
  setSortDirection: (direction: 'asc' | 'desc') => void
  clearFilters: () => void

  // Utility
  getQueryById: (id: string) => SavedQueryRecord | undefined
  canSaveMore: () => boolean
  getRemainingQuota: () => number | null
  reset: () => void
}

type SavedQueriesStore = SavedQueriesState & SavedQueriesActions

const initialState: SavedQueriesState = {
  queries: [],
  totalCount: 0,
  isLoading: false,
  error: null,
  searchText: '',
  selectedFolder: null,
  selectedTags: [],
  showFavoritesOnly: false,
  sortBy: 'updated_at',
  sortDirection: 'desc',
  folders: [],
  tags: [],
  isInitialized: false,
}

export const useSavedQueriesStore = create<SavedQueriesStore>((set, get) => ({
  ...initialState,

  // Load all queries for a user
  loadQueries: async (userId: string) => {
    set({ isLoading: true, error: null })

    try {
      const repo = getSavedQueryRepository()
      const state = get()

      // Build search options from current filter state
      const searchOptions: SavedQuerySearchOptions = {
        userId,
        searchText: state.searchText || undefined,
        folder: state.selectedFolder || undefined,
        tags: state.selectedTags.length > 0 ? state.selectedTags : [],
        favoritesOnly: state.showFavoritesOnly,
        sortBy: state.sortBy,
        sortDirection: state.sortDirection,
        limit: 1000, // Get all for now, implement pagination later
      }

      const result = await repo.search(searchOptions)

      set({
        queries: result.items,
        totalCount: result.total ?? result.items.length,
        isLoading: false,
        isInitialized: true,
      })
    } catch (error) {
      console.error('Failed to load saved queries:', error)
      set({
        error: error instanceof Error ? error.message : 'Failed to load queries',
        isLoading: false,
      })
    }
  },

  // Load metadata (folders and tags)
  loadMetadata: async (userId: string) => {
    try {
      const repo = getSavedQueryRepository()
      const [folders, tags] = await Promise.all([
        repo.getAllFolders(userId),
        repo.getAllTags(userId),
      ])

      set({ folders, tags })
    } catch (error) {
      console.error('Failed to load metadata:', error)
    }
  },

  // Refresh both queries and metadata
  refresh: async (userId: string) => {
    await Promise.all([
      get().loadQueries(userId),
      get().loadMetadata(userId),
    ])
  },

  // Save a new query
  saveQuery: async (data) => {
    const repo = getSavedQueryRepository()

    try {
      const query = await repo.create({
        user_id: data.user_id,
        title: data.title,
        description: data.description,
        query_text: data.query_text,
        tags: data.tags ?? [],
        folder: data.folder,
        is_favorite: data.is_favorite ?? false,
      })

      // Add to state optimistically
      set((state) => ({
        queries: [query, ...state.queries],
        totalCount: state.totalCount + 1,
      }))

      // Reload metadata to update folders/tags
      await get().loadMetadata(data.user_id)

      return query
    } catch (error) {
      console.error('Failed to save query:', error)
      throw error
    }
  },

  // Update an existing query
  updateQuery: async (id, updates) => {
    const repo = getSavedQueryRepository()

    // Capture current state BEFORE any modifications for rollback
    const previousQueries = get().queries

    try {
      // Optimistic update
      set((state) => ({
        queries: state.queries.map((q) =>
          q.id === id ? { ...q, ...updates } : q
        ),
      }))

      const updated = await repo.update(id, updates)

      // Update with server response
      set((state) => ({
        queries: state.queries.map((q) => (q.id === id ? updated : q)),
      }))

      // Reload metadata if folder or tags changed
      if (updates.folder !== undefined || updates.tags !== undefined) {
        const query = updated
        if (query?.user_id) {
          await get().loadMetadata(query.user_id)
        }
      }
    } catch (error) {
      // Rollback to captured state
      set({ queries: previousQueries })

      console.error('Failed to update query:', error)
      throw error
    }
  },

  // Delete a query
  deleteQuery: async (id) => {
    const repo = getSavedQueryRepository()

    try {
      // Optimistic delete
      const previousQueries = get().queries
      set((state) => ({
        queries: state.queries.filter((q) => q.id !== id),
        totalCount: Math.max(0, state.totalCount - 1),
      }))

      await repo.delete(id)

      // Reload metadata in case folder/tags are now empty
      const userId = previousQueries.find((q) => q.id === id)?.user_id
      if (userId) {
        await get().loadMetadata(userId)
      }
    } catch (error) {
      console.error('Failed to delete query:', error)
      throw error
    }
  },

  // Duplicate a query
  duplicateQuery: async (id) => {
    const repo = getSavedQueryRepository()

    try {
      const duplicate = await repo.duplicate(id)

      // Add to state
      set((state) => ({
        queries: [duplicate, ...state.queries],
        totalCount: state.totalCount + 1,
      }))

      return duplicate
    } catch (error) {
      console.error('Failed to duplicate query:', error)
      throw error
    }
  },

  // Toggle favorite status
  toggleFavorite: async (id) => {
    const repo = getSavedQueryRepository()

    try {
      // Optimistic update
      set((state) => ({
        queries: state.queries.map((q) =>
          q.id === id ? { ...q, is_favorite: !q.is_favorite } : q
        ),
      }))

      const updated = await repo.toggleFavorite(id)

      // Update with server response
      set((state) => ({
        queries: state.queries.map((q) => (q.id === id ? updated : q)),
      }))
    } catch (error) {
      console.error('Failed to toggle favorite:', error)
      throw error
    }
  },

  // Search & filter actions
  setSearchText: (text) => {
    set({ searchText: text })
  },

  setSelectedFolder: (folder) => {
    set({ selectedFolder: folder })
  },

  setSelectedTags: (tags) => {
    set({ selectedTags: tags })
  },

  toggleTag: (tag) => {
    set((state) => ({
      selectedTags: state.selectedTags.includes(tag)
        ? state.selectedTags.filter((t) => t !== tag)
        : [...state.selectedTags, tag],
    }))
  },

  setShowFavoritesOnly: (show) => {
    set({ showFavoritesOnly: show })
  },

  setSortBy: (sortBy) => {
    set({ sortBy })
  },

  setSortDirection: (direction) => {
    set({ sortDirection: direction })
  },

  clearFilters: () => {
    set({
      searchText: '',
      selectedFolder: null,
      selectedTags: [],
      showFavoritesOnly: false,
    })
  },

  // Utility functions
  getQueryById: (id) => {
    return get().queries.find((q) => q.id === id)
  },

  canSaveMore: () => {
    const tierStore = useTierStore.getState()
    const currentCount = get().queries.length
    const limitCheck = tierStore.checkLimit('savedQueries', currentCount + 1)
    return limitCheck.allowed
  },

  getRemainingQuota: () => {
    const tierStore = useTierStore.getState()
    const currentCount = get().queries.length
    const limitCheck = tierStore.checkLimit('savedQueries', currentCount)

    if (limitCheck.isUnlimited) {
      return null // Unlimited
    }

    return limitCheck.remaining
  },

  reset: () => {
    set(initialState)
  },
}))

/**
 * Hook for loading saved queries on mount
 */
export function useLoadSavedQueries(userId: string | null) {
  const store = useSavedQueriesStore()

  // Load queries when userId becomes available
  useEffect(() => {
    if (userId && !store.isInitialized) {
      store.refresh(userId).catch((error) => {
        console.error('Failed to load saved queries:', error)
      })
    }
  }, [userId, store.isInitialized, store])

  return store
}
