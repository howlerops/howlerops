/**
 * Connections Store
 *
 * Zustand store for managing database connections with organization sharing.
 * Provides CRUD operations, sharing/unsharing, and permission-aware filtering.
 *
 * @module store/connections-store
 */

import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import {
  getConnections,
  createConnection as apiCreateConnection,
  updateConnection as apiUpdateConnection,
  deleteConnection as apiDeleteConnection,
  shareConnection as apiShareConnection,
  unshareConnection as apiUnshareConnection,
  getOrganizationConnections,
  type Connection,
  type CreateConnectionInput,
  type UpdateConnectionInput,
} from '@/lib/api/connections'

/**
 * Store state interface
 */
interface ConnectionsState {
  /** Personal and accessible shared connections */
  connections: Connection[]

  /** Connections shared in current organization (cached) */
  sharedConnections: Connection[]

  /** Loading states */
  loading: boolean

  /** Error message if operation failed */
  error: string | null
}

/**
 * Store actions interface
 */
interface ConnectionsActions {
  // CRUD operations
  /**
   * Fetch all connections for current user
   */
  fetchConnections: () => Promise<void>

  /**
   * Fetch shared connections for an organization
   */
  fetchSharedConnections: (orgId: string) => Promise<void>

  /**
   * Create a new connection
   */
  createConnection: (input: CreateConnectionInput) => Promise<Connection>

  /**
   * Update an existing connection
   */
  updateConnection: (
    id: string,
    input: UpdateConnectionInput
  ) => Promise<void>

  /**
   * Delete a connection
   */
  deleteConnection: (id: string) => Promise<void>

  // Sharing operations
  /**
   * Share a connection with an organization
   */
  shareConnection: (id: string, orgId: string) => Promise<void>

  /**
   * Unshare a connection (make it personal)
   */
  unshareConnection: (id: string) => Promise<void>

  // Filtering
  /**
   * Get connections by organization ID
   */
  getConnectionsByOrg: (orgId: string) => Connection[]

  /**
   * Get only personal connections
   */
  getPersonalConnections: () => Connection[]

  // Utilities
  /**
   * Clear error message
   */
  clearError: () => void
}

type ConnectionsStore = ConnectionsState & ConnectionsActions

/**
 * Default initial state
 */
const DEFAULT_STATE: ConnectionsState = {
  connections: [],
  sharedConnections: [],
  loading: false,
  error: null,
}

/**
 * Connections Management Store
 *
 * Usage:
 * ```typescript
 * const { connections, shareConnection } = useConnectionsStore()
 *
 * // Share a connection
 * await shareConnection(connectionId, orgId)
 *
 * // Get org connections
 * const orgConnections = getConnectionsByOrg(orgId)
 * ```
 */
export const useConnectionsStore = create<ConnectionsStore>()(
  devtools(
    (set, get) => ({
      ...DEFAULT_STATE,

      // ================================================================
      // CRUD Operations
      // ================================================================

      fetchConnections: async () => {
        set({ loading: true, error: null }, false, 'fetchConnections/start')

        try {
          const connections = await getConnections()

          set(
            { connections, loading: false },
            false,
            'fetchConnections/success'
          )
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to fetch connections'

          set(
            { error: errorMessage, loading: false },
            false,
            'fetchConnections/error'
          )

          throw error
        }
      },

      fetchSharedConnections: async (orgId: string) => {
        set({ loading: true, error: null }, false, 'fetchSharedConnections/start')

        try {
          const sharedConnections = await getOrganizationConnections(orgId)

          set(
            { sharedConnections, loading: false },
            false,
            'fetchSharedConnections/success'
          )
        } catch (error) {
          const errorMessage =
            error instanceof Error
              ? error.message
              : 'Failed to fetch shared connections'

          set(
            { error: errorMessage, loading: false },
            false,
            'fetchSharedConnections/error'
          )

          throw error
        }
      },

      createConnection: async (input) => {
        set({ loading: true, error: null }, false, 'createConnection/start')

        try {
          const newConnection = await apiCreateConnection(input)

          set(
            (state) => ({
              connections: [...state.connections, newConnection],
              loading: false,
            }),
            false,
            'createConnection/success'
          )

          return newConnection
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to create connection'

          set(
            { error: errorMessage, loading: false },
            false,
            'createConnection/error'
          )

          throw error
        }
      },

      updateConnection: async (id, input) => {
        const state = get()
        const originalConnection = state.connections.find((c) => c.id === id)

        if (!originalConnection) {
          throw new Error('Connection not found')
        }

        // Optimistic update
        set(
          (state) => ({
            connections: state.connections.map((c) =>
              c.id === id ? { ...c, ...input } : c
            ),
            loading: true,
            error: null,
          }),
          false,
          'updateConnection/optimistic'
        )

        try {
          const updatedConnection = await apiUpdateConnection(id, input)

          set(
            (state) => ({
              connections: state.connections.map((c) =>
                c.id === id ? updatedConnection : c
              ),
              loading: false,
            }),
            false,
            'updateConnection/success'
          )
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to update connection'

          // Rollback optimistic update
          set(
            (state) => ({
              connections: state.connections.map((c) =>
                c.id === id ? originalConnection : c
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'updateConnection/rollback'
          )

          throw error
        }
      },

      deleteConnection: async (id) => {
        const state = get()
        const originalConnections = [...state.connections]

        // Optimistic removal
        set(
          (state) => ({
            connections: state.connections.filter((c) => c.id !== id),
            loading: true,
            error: null,
          }),
          false,
          'deleteConnection/optimistic'
        )

        try {
          await apiDeleteConnection(id)

          set({ loading: false }, false, 'deleteConnection/success')
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to delete connection'

          // Rollback optimistic removal
          set(
            {
              connections: originalConnections,
              error: errorMessage,
              loading: false,
            },
            false,
            'deleteConnection/rollback'
          )

          throw error
        }
      },

      // ================================================================
      // Sharing Operations
      // ================================================================

      shareConnection: async (id, orgId) => {
        const state = get()
        const originalConnection = state.connections.find((c) => c.id === id)

        if (!originalConnection) {
          throw new Error('Connection not found')
        }

        // Optimistic update
        set(
          (state) => ({
            connections: state.connections.map((c) =>
              c.id === id
                ? { ...c, visibility: 'shared', organization_id: orgId }
                : c
            ),
            loading: true,
            error: null,
          }),
          false,
          'shareConnection/optimistic'
        )

        try {
          await apiShareConnection(id, orgId)

          set({ loading: false }, false, 'shareConnection/success')

          // Refresh shared connections for this org
          await get().fetchSharedConnections(orgId)
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to share connection'

          // Rollback optimistic update
          set(
            (state) => ({
              connections: state.connections.map((c) =>
                c.id === id ? originalConnection : c
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'shareConnection/rollback'
          )

          throw error
        }
      },

      unshareConnection: async (id) => {
        const state = get()
        const originalConnection = state.connections.find((c) => c.id === id)

        if (!originalConnection) {
          throw new Error('Connection not found')
        }

        // Optimistic update
        set(
          (state) => ({
            connections: state.connections.map((c) =>
              c.id === id
                ? { ...c, visibility: 'personal', organization_id: null }
                : c
            ),
            loading: true,
            error: null,
          }),
          false,
          'unshareConnection/optimistic'
        )

        try {
          await apiUnshareConnection(id)

          set({ loading: false }, false, 'unshareConnection/success')

          // Refresh shared connections
          if (originalConnection.organization_id) {
            await get().fetchSharedConnections(originalConnection.organization_id)
          }
        } catch (error) {
          const errorMessage =
            error instanceof Error ? error.message : 'Failed to unshare connection'

          // Rollback optimistic update
          set(
            (state) => ({
              connections: state.connections.map((c) =>
                c.id === id ? originalConnection : c
              ),
              error: errorMessage,
              loading: false,
            }),
            false,
            'unshareConnection/rollback'
          )

          throw error
        }
      },

      // ================================================================
      // Filtering
      // ================================================================

      getConnectionsByOrg: (orgId) => {
        const state = get()
        return state.connections.filter((c) => c.organization_id === orgId)
      },

      getPersonalConnections: () => {
        const state = get()
        return state.connections.filter((c) => c.visibility === 'personal')
      },

      // ================================================================
      // Utilities
      // ================================================================

      clearError: () => {
        set({ error: null }, false, 'clearError')
      },
    }),
    {
      name: 'ConnectionsStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Selectors for common queries
 */
export const connectionsSelectors = {
  hasConnections: (state: ConnectionsStore) => state.connections.length > 0,
  isLoading: (state: ConnectionsStore) => state.loading,
  hasError: (state: ConnectionsStore) => !!state.error,
  getSharedCount: (state: ConnectionsStore) =>
    state.connections.filter((c) => c.visibility === 'shared').length,
  getPersonalCount: (state: ConnectionsStore) =>
    state.connections.filter((c) => c.visibility === 'personal').length,
}
