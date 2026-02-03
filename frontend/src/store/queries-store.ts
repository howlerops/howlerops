/**
 * Unified Queries Store
 *
 * Zustand store for managing saved queries with both local storage and server sync.
 * Combines personal query library (IndexedDB) with organization sharing (API).
 *
 * Features:
 * - CRUD operations with IndexedDB for local-first storage
 * - Organization sharing via server API
 * - Search, filter, and folder organization
 * - Tier-aware limit checking
 * - Optimistic updates with rollback
 *
 * @module store/queries-store
 */

import { useEffect } from 'react'
import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

import {
  getOrganizationQueries,
  type SavedQuery as ServerSavedQuery,
  shareQuery as apiShareQuery,
  unshareQuery as apiUnshareQuery,
} from '@/lib/api/queries'
import { dedupedRequest } from '@/lib/request-deduplication'
import {
  getSavedQueryRepository,
  type SavedQuerySearchOptions,
} from '@/lib/storage'
import type { SavedQueryRecord } from '@/types/storage'

import { useTierStore } from './tier-store'

// ============================================================================
// Types
// ============================================================================

/**
 * Combined query type that works with both local and server data
 */
export type SavedQuery = SavedQueryRecord

/**
 * Re-export ServerSavedQuery for components that need it
 */
export type { ServerSavedQuery }

/**
 * Store state interface
 */
interface QueriesState {
  // Local queries (from IndexedDB)
  queries: SavedQueryRecord[]
  totalCount: number

  // Shared queries from organization (from server API)
  sharedQueries: ServerSavedQuery[]

  // Loading states
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

/**
 * Store actions interface
 */
interface QueriesActions {
  // Data loading
  loadQueries: (userId: string) => Promise<void>
  loadMetadata: (userId: string) => Promise<void>
  refresh: (userId: string) => Promise<void>
  fetchSharedQueries: (orgId: string) => Promise<void>

  // CRUD operations (local storage)
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

  // Sharing operations (server API)
  shareQuery: (id: string, orgId: string) => Promise<void>
  unshareQuery: (id: string) => Promise<void>

  // Search & filter
  setSearchText: (text: string) => void
  setSelectedFolder: (folder: string | null) => void
  setSelectedTags: (tags: string[]) => void
  toggleTag: (tag: string) => void
  setShowFavoritesOnly: (show: boolean) => void
  setSortBy: (sortBy: 'title' | 'created_at' | 'updated_at') => void
  setSortDirection: (direction: 'asc' | 'desc') => void
  clearFilters: () => void

  // Filtering helpers
  getQueriesByOrg: (orgId: string) => ServerSavedQuery[]
  getPersonalQueries: () => SavedQueryRecord[]
  getQueriesByTag: (tag: string) => SavedQueryRecord[]

  // Utility
  getQueryById: (id: string) => SavedQueryRecord | undefined
  canSaveMore: () => boolean
  getRemainingQuota: () => number | null
  clearError: () => void
  reset: () => void
}

type QueriesStore = QueriesState & QueriesActions

// ============================================================================
// Initial State
// ============================================================================

const initialState: QueriesState = {
  queries: [],
  totalCount: 0,
  sharedQueries: [],
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

// ============================================================================
// Store Implementation
// ============================================================================

/**
 * Unified Queries Store
 *
 * Usage:
 * ```typescript
 * const { queries, saveQuery, shareQuery } = useQueriesStore()
 *
 * // Save a new query locally
 * const query = await saveQuery({ user_id, title, query_text })
 *
 * // Share with organization
 * await shareQuery(query.id, orgId)
 *
 * // Get shared queries
 * await fetchSharedQueries(orgId)
 * ```
 */
export const useQueriesStore = create<QueriesStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // ================================================================
      // Data Loading
      // ================================================================

      loadQueries: async (userId: string) => {
        return dedupedRequest(`loadQueries-${userId}`, async () => {
          set({ isLoading: true, error: null }, false, 'loadQueries/start')

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

            set(
              {
                queries: result.items,
                totalCount: result.total ?? result.items.length,
                isLoading: false,
                isInitialized: true,
              },
              false,
              'loadQueries/success'
            )
          } catch (error) {
            console.error('Failed to load saved queries:', error)
            set(
              {
                error:
                  error instanceof Error
                    ? error.message
                    : 'Failed to load queries',
                isLoading: false,
              },
              false,
              'loadQueries/error'
            )
          }
        })
      },

      loadMetadata: async (userId: string) => {
        return dedupedRequest(`loadMetadata-${userId}`, async () => {
          try {
            const repo = getSavedQueryRepository()
            const [folders, tags] = await Promise.all([
              repo.getAllFolders(userId),
              repo.getAllTags(userId),
            ])

            set({ folders, tags }, false, 'loadMetadata/success')
          } catch (error) {
            console.error('Failed to load metadata:', error)
          }
        })
      },

      refresh: async (userId: string) => {
        await Promise.all([get().loadQueries(userId), get().loadMetadata(userId)])
      },

      fetchSharedQueries: async (orgId: string) => {
        return dedupedRequest(`fetchSharedQueries-${orgId}`, async () => {
          set({ isLoading: true, error: null }, false, 'fetchSharedQueries/start')

          try {
            const sharedQueries = await getOrganizationQueries(orgId)

            set(
              { sharedQueries, isLoading: false },
              false,
              'fetchSharedQueries/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof Error
                ? error.message
                : 'Failed to fetch shared queries'

            set(
              { error: errorMessage, isLoading: false },
              false,
              'fetchSharedQueries/error'
            )

            throw error
          }
        })
      },

      // ================================================================
      // CRUD Operations (Local Storage)
      // ================================================================

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
          set(
            (state) => ({
              queries: [query, ...state.queries],
              totalCount: state.totalCount + 1,
            }),
            false,
            'saveQuery/success'
          )

          // Reload metadata to update folders/tags
          await get().loadMetadata(data.user_id)

          return query
        } catch (error) {
          console.error('Failed to save query:', error)
          throw error
        }
      },

      updateQuery: async (id, updates) => {
        const repo = getSavedQueryRepository()

        // Capture current state BEFORE any modifications for rollback
        const previousQueries = get().queries

        try {
          // Optimistic update
          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? { ...q, ...updates } : q
              ),
            }),
            false,
            'updateQuery/optimistic'
          )

          const updated = await repo.update(id, updates)

          // Update with server response
          set(
            (state) => ({
              queries: state.queries.map((q) => (q.id === id ? updated : q)),
            }),
            false,
            'updateQuery/success'
          )

          // Reload metadata if folder or tags changed
          if (updates.folder !== undefined || updates.tags !== undefined) {
            if (updated?.user_id) {
              await get().loadMetadata(updated.user_id)
            }
          }
        } catch (error) {
          // Rollback to captured state
          set({ queries: previousQueries }, false, 'updateQuery/rollback')

          console.error('Failed to update query:', error)
          throw error
        }
      },

      deleteQuery: async (id) => {
        return dedupedRequest(`deleteQuery-${id}`, async () => {
          const repo = getSavedQueryRepository()

          try {
            // Optimistic delete
            const previousQueries = get().queries
            set(
              (state) => ({
                queries: state.queries.filter((q) => q.id !== id),
                totalCount: Math.max(0, state.totalCount - 1),
              }),
              false,
              'deleteQuery/optimistic'
            )

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
        })
      },

      duplicateQuery: async (id) => {
        const repo = getSavedQueryRepository()

        try {
          const duplicate = await repo.duplicate(id)

          // Add to state
          set(
            (state) => ({
              queries: [duplicate, ...state.queries],
              totalCount: state.totalCount + 1,
            }),
            false,
            'duplicateQuery/success'
          )

          return duplicate
        } catch (error) {
          console.error('Failed to duplicate query:', error)
          throw error
        }
      },

      toggleFavorite: async (id) => {
        const repo = getSavedQueryRepository()

        try {
          // Optimistic update
          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? { ...q, is_favorite: !q.is_favorite } : q
              ),
            }),
            false,
            'toggleFavorite/optimistic'
          )

          const updated = await repo.toggleFavorite(id)

          // Update with server response
          set(
            (state) => ({
              queries: state.queries.map((q) => (q.id === id ? updated : q)),
            }),
            false,
            'toggleFavorite/success'
          )
        } catch (error) {
          console.error('Failed to toggle favorite:', error)
          throw error
        }
      },

      // ================================================================
      // Sharing Operations (Server API)
      // ================================================================

      shareQuery: async (id, orgId) => {
        return dedupedRequest(`shareQuery-${id}-${orgId}`, async () => {
          set({ isLoading: true, error: null }, false, 'shareQuery/start')

          try {
            await apiShareQuery(id, orgId)

            set({ isLoading: false }, false, 'shareQuery/success')

            // Refresh shared queries for this org
            await get().fetchSharedQueries(orgId)
          } catch (error) {
            const errorMessage =
              error instanceof Error ? error.message : 'Failed to share query'

            set(
              { error: errorMessage, isLoading: false },
              false,
              'shareQuery/error'
            )

            throw error
          }
        })
      },

      unshareQuery: async (id) => {
        return dedupedRequest(`unshareQuery-${id}`, async () => {
          set({ isLoading: true, error: null }, false, 'unshareQuery/start')

          try {
            await apiUnshareQuery(id)

            // Remove from shared queries list
            set(
              (state) => ({
                sharedQueries: state.sharedQueries.filter((q) => q.id !== id),
                isLoading: false,
              }),
              false,
              'unshareQuery/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof Error ? error.message : 'Failed to unshare query'

            set(
              { error: errorMessage, isLoading: false },
              false,
              'unshareQuery/error'
            )

            throw error
          }
        })
      },

      // ================================================================
      // Search & Filter Actions
      // ================================================================

      setSearchText: (text) => {
        set({ searchText: text }, false, 'setSearchText')
      },

      setSelectedFolder: (folder) => {
        set({ selectedFolder: folder }, false, 'setSelectedFolder')
      },

      setSelectedTags: (tags) => {
        set({ selectedTags: tags }, false, 'setSelectedTags')
      },

      toggleTag: (tag) => {
        set(
          (state) => ({
            selectedTags: state.selectedTags.includes(tag)
              ? state.selectedTags.filter((t) => t !== tag)
              : [...state.selectedTags, tag],
          }),
          false,
          'toggleTag'
        )
      },

      setShowFavoritesOnly: (show) => {
        set({ showFavoritesOnly: show }, false, 'setShowFavoritesOnly')
      },

      setSortBy: (sortBy) => {
        set({ sortBy }, false, 'setSortBy')
      },

      setSortDirection: (direction) => {
        set({ sortDirection: direction }, false, 'setSortDirection')
      },

      clearFilters: () => {
        set(
          {
            searchText: '',
            selectedFolder: null,
            selectedTags: [],
            showFavoritesOnly: false,
          },
          false,
          'clearFilters'
        )
      },

      // ================================================================
      // Filtering Helpers
      // ================================================================

      getQueriesByOrg: (orgId) => {
        const state = get()
        return state.sharedQueries.filter((q) => q.organization_id === orgId)
      },

      getPersonalQueries: () => {
        const state = get()
        // Local queries are personal by default
        return state.queries
      },

      getQueriesByTag: (tag) => {
        const state = get()
        return state.queries.filter((q) => q.tags?.includes(tag))
      },

      // ================================================================
      // Utility Functions
      // ================================================================

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

      clearError: () => {
        set({ error: null }, false, 'clearError')
      },

      reset: () => {
        set(initialState, false, 'reset')
      },
    }),
    {
      name: 'QueriesStore',
      enabled: import.meta.env.DEV,
    }
  )
)

// ============================================================================
// Selectors
// ============================================================================

/**
 * Selectors for common queries
 */
export const queriesSelectors = {
  hasQueries: (state: QueriesStore) => state.queries.length > 0,
  isLoading: (state: QueriesStore) => state.isLoading,
  hasError: (state: QueriesStore) => !!state.error,
  getSharedCount: (state: QueriesStore) => state.sharedQueries.length,
  getPersonalCount: (state: QueriesStore) => state.queries.length,
  getAllTags: (state: QueriesStore) => {
    const tags = new Set<string>()
    state.queries.forEach((q) => {
      q.tags?.forEach((tag) => tags.add(tag))
    })
    return Array.from(tags).sort()
  },
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for loading saved queries on mount
 */
export function useLoadQueries(userId: string | null) {
  const store = useQueriesStore()

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

// Legacy alias for backward compatibility
export const useSavedQueriesStore = useQueriesStore
export const useLoadSavedQueries = useLoadQueries
