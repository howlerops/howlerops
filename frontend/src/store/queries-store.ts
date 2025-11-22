/**
 * Queries Store
 *
 * Zustand store for managing saved queries with organization sharing.
 * Provides CRUD operations, sharing/unsharing, and permission-aware filtering.
 *
 * @module store/queries-store
 */

import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

import {
  createQuery as apiCreateQuery,
  type CreateQueryInput,
  deleteQuery as apiDeleteQuery,
  getOrganizationQueries,
  getQueries,
  type SavedQuery,
  shareQuery as apiShareQuery,
  unshareQuery as apiUnshareQuery,
  updateQuery as apiUpdateQuery,
  type UpdateQueryInput,
} from '@/lib/api/queries'

/**
 * Store state interface
 */
interface QueriesState {
  /** Personal and accessible shared queries */
  queries: SavedQuery[]

  /** Queries shared in current organization (cached) */
  sharedQueries: SavedQuery[]

  /** Loading states */
  loading: boolean

  /** Error message if operation failed */
  error: string | null
}

/**
 * Store actions interface
 */
interface QueriesActions {
  // CRUD operations
  /**
   * Fetch all queries for current user
   */
  fetchQueries: () => Promise<void>

  /**
   * Fetch shared queries for an organization
   */
  fetchSharedQueries: (orgId: string) => Promise<void>

  /**
   * Create a new query
   */
  createQuery: (input: CreateQueryInput) => Promise<SavedQuery>

  /**
   * Update an existing query
   */
  updateQuery: (id: string, input: UpdateQueryInput) => Promise<void>

  /**
   * Delete a query
   */
  deleteQuery: (id: string) => Promise<void>

  // Sharing operations
  /**
   * Share a query with an organization
   */
  shareQuery: (id: string, orgId: string) => Promise<void>

  /**
   * Unshare a query (make it personal)
   */
  unshareQuery: (id: string) => Promise<void>

  // Filtering
  /**
   * Get queries by organization ID
   */
  getQueriesByOrg: (orgId: string) => SavedQuery[]

  /**
   * Get only personal queries
   */
  getPersonalQueries: () => SavedQuery[]

  /**
   * Get queries by tag
   */
  getQueriesByTag: (tag: string) => SavedQuery[]

  // Utilities
  /**
   * Clear error message
   */
  clearError: () => void
}

type QueriesStore = QueriesState & QueriesActions

/**
 * Default initial state
 */
const DEFAULT_STATE: QueriesState = {
  queries: [],
  sharedQueries: [],
  loading: false,
  error: null,
}

/**
 * Queries Management Store
 *
 * Usage:
 * ```typescript
 * const { queries, shareQuery } = useQueriesStore()
 *
 * // Share a query
 * await shareQuery(queryId, orgId)
 *
 * // Get org queries
 * const orgQueries = getQueriesByOrg(orgId)
 * ```
 */
export const useQueriesStore = create<QueriesStore>()(
  devtools(
    (set, get) => ({
      ...DEFAULT_STATE,

      // ================================================================
      // CRUD Operations
      // ================================================================

      fetchQueries: async () => {
        set({ loading: true, error: null }, false, 'fetchQueries/start')

        try {
          const queries = await getQueries()

          set({ queries, loading: false }, false, 'fetchQueries/success')
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to fetch queries'

          set(
            { error: errorMessage, loading: false },
            false,
            'fetchQueries/error'
          )

          throw error
        }
      },

      fetchSharedQueries: async (orgId: string) => {
        set({ loading: true, error: null }, false, 'fetchSharedQueries/start')

        try {
          const sharedQueries = await getOrganizationQueries(orgId)

          set(
            { sharedQueries, loading: false },
            false,
            'fetchSharedQueries/success'
          )
        } catch (error) {
          const errorMessage =
            error instanceof Error
              ? error.message
              : 'Failed to fetch shared queries'

          set(
            { error: errorMessage, loading: false },
            false,
            'fetchSharedQueries/error'
          )

          throw error
        }
      },

      createQuery: async (input) => {
        set({ loading: true, error: null }, false, 'createQuery/start')

        try {
          const newQuery = await apiCreateQuery(input)

          set(
            (state) => ({
              queries: [...state.queries, newQuery],
              loading: false,
            }),
            false,
            'createQuery/success'
          )

          return newQuery
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to create query'

          set(
            { error: errorMessage, loading: false },
            false,
            'createQuery/error'
          )

          throw error
        }
      },

      updateQuery: async (id, input) => {
        const state = get()
        const originalQuery = state.queries.find((q) => q.id === id)

        if (!originalQuery) {
          throw new Error('Query not found')
        }

        // Optimistic update
        set(
          (state) => ({
            queries: state.queries.map((q) =>
              q.id === id ? { ...q, ...input } : q
            ),
            loading: true,
            error: null,
          }),
          false,
          'updateQuery/optimistic'
        )

        try {
          const updatedQuery = await apiUpdateQuery(id, input)

          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? updatedQuery : q
              ),
              loading: false,
            }),
            false,
            'updateQuery/success'
          )
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to update query'

          // Rollback optimistic update
          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? originalQuery : q
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'updateQuery/rollback'
          )

          throw error
        }
      },

      deleteQuery: async (id) => {
        const state = get()
        const originalQueries = [...state.queries]

        // Optimistic removal
        set(
          (state) => ({
            queries: state.queries.filter((q) => q.id !== id),
            loading: true,
            error: null,
          }),
          false,
          'deleteQuery/optimistic'
        )

        try {
          await apiDeleteQuery(id)

          set({ loading: false }, false, 'deleteQuery/success')
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to delete query'

          // Rollback optimistic removal
          set(
            {
              queries: originalQueries,
              error: errorMessage,
              loading: false,
            },
            false,
            'deleteQuery/rollback'
          )

          throw error
        }
      },

      // ================================================================
      // Sharing Operations
      // ================================================================

      shareQuery: async (id, orgId) => {
        const state = get()
        const originalQuery = state.queries.find((q) => q.id === id)

        if (!originalQuery) {
          throw new Error('Query not found')
        }

        // Optimistic update
        set(
          (state) => ({
            queries: state.queries.map((q) =>
              q.id === id
                ? { ...q, visibility: 'shared', organization_id: orgId }
                : q
            ),
            loading: true,
            error: null,
          }),
          false,
          'shareQuery/optimistic'
        )

        try {
          await apiShareQuery(id, orgId)

          set({ loading: false }, false, 'shareQuery/success')

          // Refresh shared queries for this org
          await get().fetchSharedQueries(orgId)
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to share query'

          // Rollback optimistic update
          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? originalQuery : q
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'shareQuery/rollback'
          )

          throw error
        }
      },

      unshareQuery: async (id) => {
        const state = get()
        const originalQuery = state.queries.find((q) => q.id === id)

        if (!originalQuery) {
          throw new Error('Query not found')
        }

        // Optimistic update
        set(
          (state) => ({
            queries: state.queries.map((q) =>
              q.id === id
                ? { ...q, visibility: 'personal', organization_id: null }
                : q
            ),
            loading: true,
            error: null,
          }),
          false,
          'unshareQuery/optimistic'
        )

        try {
          await apiUnshareQuery(id)

          set({ loading: false }, false, 'unshareQuery/success')

          // Refresh shared queries
          if (originalQuery.organization_id) {
            await get().fetchSharedQueries(originalQuery.organization_id)
          }
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to unshare query'

          // Rollback optimistic update
          set(
            (state) => ({
              queries: state.queries.map((q) =>
                q.id === id ? originalQuery : q
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'unshareQuery/rollback'
          )

          throw error
        }
      },

      // ================================================================
      // Filtering
      // ================================================================

      getQueriesByOrg: (orgId) => {
        const state = get()
        return state.queries.filter((q) => q.organization_id === orgId)
      },

      getPersonalQueries: () => {
        const state = get()
        return state.queries.filter((q) => q.visibility === 'personal')
      },

      getQueriesByTag: (tag) => {
        const state = get()
        return state.queries.filter((q) => q.tags?.includes(tag))
      },

      // ================================================================
      // Utilities
      // ================================================================

      clearError: () => {
        set({ error: null }, false, 'clearError')
      },
    }),
    {
      name: 'QueriesStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Selectors for common queries
 */
export const queriesSelectors = {
  hasQueries: (state: QueriesStore) => state.queries.length > 0,
  isLoading: (state: QueriesStore) => state.loading,
  hasError: (state: QueriesStore) => !!state.error,
  getSharedCount: (state: QueriesStore) =>
    state.queries.filter((q) => q.visibility === 'shared').length,
  getPersonalCount: (state: QueriesStore) =>
    state.queries.filter((q) => q.visibility === 'personal').length,
  getAllTags: (state: QueriesStore) => {
    const tags = new Set<string>()
    state.queries.forEach((q) => {
      q.tags?.forEach((tag) => tags.add(tag))
    })
    return Array.from(tags).sort()
  },
}
