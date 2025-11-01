/**
 * Organization State Management Store
 *
 * Zustand store for managing organization state, members, and invitations.
 * Provides optimistic updates, error handling, and rollback on failures.
 *
 * @module store/organization-store
 */

import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { authFetch, AuthApiError } from '@/lib/api/auth-client'
import type {
  Organization,
  OrganizationMember,
  OrganizationInvitation,
  AuditLog,
  CreateOrganizationInput,
  UpdateOrganizationInput,
  CreateInvitationInput,
  UpdateMemberRoleInput,
  OrganizationRole,
  AuditLogQueryParams,
} from '@/types/organization'

// Import date parsers
import {
  parseOrganizationDates as parseOrgDates,
  parseMemberDates as parseMbrDates,
  parseInvitationDates as parseInvDates,
  parseAuditLogDates as parseLogDates,
} from '@/types/organization'

/**
 * Organization store state interface
 */
interface OrganizationState {
  // State
  /** List of organizations the user is a member of */
  organizations: Organization[]

  /** Currently selected organization ID */
  currentOrgId: string | null

  /** Members of current organization (cached) */
  currentOrgMembers: OrganizationMember[]

  /** Invitations for current organization (cached) */
  currentOrgInvitations: OrganizationInvitation[]

  /** Pending invitations for current user */
  pendingInvitations: OrganizationInvitation[]

  /** Loading states */
  loading: {
    organizations: boolean
    members: boolean
    invitations: boolean
    creating: boolean
    updating: boolean
    deleting: boolean
  }

  /** Error message if operation failed */
  error: string | null
}

/**
 * Organization store actions interface
 */
interface OrganizationActions {
  // Organization CRUD
  /**
   * Fetch all organizations for current user
   */
  fetchOrganizations: () => Promise<void>

  /**
   * Create a new organization
   */
  createOrganization: (
    input: CreateOrganizationInput
  ) => Promise<Organization>

  /**
   * Update an organization
   */
  updateOrganization: (
    id: string,
    input: UpdateOrganizationInput
  ) => Promise<void>

  /**
   * Delete an organization
   */
  deleteOrganization: (id: string) => Promise<void>

  /**
   * Switch to a different organization context
   */
  switchOrganization: (id: string | null) => void

  // Members
  /**
   * Fetch members of an organization
   */
  fetchMembers: (orgId: string) => Promise<OrganizationMember[]>

  /**
   * Update a member's role
   */
  updateMemberRole: (
    orgId: string,
    userId: string,
    role: OrganizationRole
  ) => Promise<void>

  /**
   * Remove a member from an organization
   */
  removeMember: (orgId: string, userId: string) => Promise<void>

  // Invitations
  /**
   * Create a new invitation
   */
  createInvitation: (
    orgId: string,
    input: CreateInvitationInput
  ) => Promise<OrganizationInvitation>

  /**
   * Fetch invitations for an organization
   */
  fetchInvitations: (orgId: string) => Promise<OrganizationInvitation[]>

  /**
   * Fetch pending invitations for current user
   */
  fetchPendingInvitations: () => Promise<OrganizationInvitation[]>

  /**
   * Accept an invitation
   */
  acceptInvitation: (inviteId: string) => Promise<void>

  /**
   * Decline an invitation
   */
  declineInvitation: (inviteId: string) => Promise<void>

  /**
   * Revoke an invitation
   */
  revokeInvitation: (orgId: string, inviteId: string) => Promise<void>

  // Audit Logs
  /**
   * Fetch audit logs for an organization
   */
  fetchAuditLogs: (
    orgId: string,
    params?: AuditLogQueryParams
  ) => Promise<AuditLog[]>

  // Utilities
  /**
   * Clear error message
   */
  clearError: () => void

  /**
   * Get current organization object
   */
  currentOrg: () => Organization | null

  /**
   * Get user's role in current organization
   */
  currentUserRole: () => OrganizationRole | null

  /**
   * Refresh current organization data
   */
  refreshCurrentOrg: () => Promise<void>
}

type OrganizationStore = OrganizationState & OrganizationActions

/**
 * Default initial state
 */
const DEFAULT_STATE: OrganizationState = {
  organizations: [],
  currentOrgId: null,
  currentOrgMembers: [],
  currentOrgInvitations: [],
  pendingInvitations: [],
  loading: {
    organizations: false,
    members: false,
    invitations: false,
    creating: false,
    updating: false,
    deleting: false,
  },
  error: null,
}

/**
 * Organization Management Store
 *
 * Usage:
 * ```typescript
 * const { organizations, currentOrg, createOrganization } = useOrganizationStore()
 *
 * // Fetch organizations
 * await fetchOrganizations()
 *
 * // Create organization
 * const org = await createOrganization({ name: 'My Team' })
 *
 * // Switch context
 * switchOrganization(org.id)
 * ```
 */
export const useOrganizationStore = create<OrganizationStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...DEFAULT_STATE,

        // ================================================================
        // Organization CRUD Operations
        // ================================================================

        fetchOrganizations: async () => {
          set(
            { loading: { ...get().loading, organizations: true }, error: null },
            false,
            'fetchOrganizations/start'
          )

          try {
            const response = await authFetch<{ organizations: Organization[] }>(
              '/api/organizations'
            )

            const organizations = (response.organizations ?? []).map(parseOrgDates)

            set(
              {
                organizations,
                loading: { ...get().loading, organizations: false },
              },
              false,
              'fetchOrganizations/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to fetch organizations'

            set(
              {
                error: errorMessage,
                loading: { ...get().loading, organizations: false },
              },
              false,
              'fetchOrganizations/error'
            )

            throw error
          }
        },

        createOrganization: async (input) => {
          const state = get()

          // Optimistic update
          const tempId = `temp-${Date.now()}`
          const optimisticOrg: Organization = {
            id: tempId,
            name: input.name,
            description: input.description,
            owner_id: '', // Will be filled by server
            created_at: new Date(),
            updated_at: new Date(),
            max_members: 10, // Default
            member_count: 1,
          }

          set(
            {
              organizations: [...state.organizations, optimisticOrg],
              loading: { ...state.loading, creating: true },
              error: null,
            },
            false,
            'createOrganization/optimistic'
          )

          try {
            const response = await authFetch<{ organization: Organization }>(
              '/api/organizations',
              {
                method: 'POST',
                body: JSON.stringify(input),
              }
            )

            const newOrg = parseOrgDates(response.organization)

            // Replace optimistic with real data
            set(
              {
                organizations: state.organizations.map((org) =>
                  org.id === tempId ? newOrg : org
                ),
                currentOrgId: newOrg.id,
                loading: { ...state.loading, creating: false },
              },
              false,
              'createOrganization/success'
            )

            return newOrg
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to create organization'

            // Rollback optimistic update
            set(
              {
                organizations: state.organizations.filter(
                  (org) => org.id !== tempId
                ),
                error: errorMessage,
                loading: { ...state.loading, creating: false },
              },
              false,
              'createOrganization/rollback'
            )

            throw error
          }
        },

        updateOrganization: async (id, input) => {
          const state = get()
          const originalOrg = state.organizations.find((org) => org.id === id)

          if (!originalOrg) {
            throw new Error('Organization not found')
          }

          // Optimistic update
          const optimisticOrg: Organization = {
            ...originalOrg,
            ...input,
            updated_at: new Date(),
          }

          set(
            {
              organizations: state.organizations.map((org) =>
                org.id === id ? optimisticOrg : org
              ),
              loading: { ...state.loading, updating: true },
              error: null,
            },
            false,
            'updateOrganization/optimistic'
          )

          try {
            const response = await authFetch<{ organization: Organization }>(
              `/api/organizations/${id}`,
              {
                method: 'PUT',
                body: JSON.stringify(input),
              }
            )

            const updatedOrg = parseOrgDates(response.organization)

            set(
              {
                organizations: state.organizations.map((org) =>
                  org.id === id ? updatedOrg : org
                ),
                loading: { ...state.loading, updating: false },
              },
              false,
              'updateOrganization/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to update organization'

            // Rollback optimistic update
            set(
              {
                organizations: state.organizations.map((org) =>
                  org.id === id ? originalOrg : org
                ),
                error: errorMessage,
                loading: { ...state.loading, updating: false },
              },
              false,
              'updateOrganization/rollback'
            )

            throw error
          }
        },

        deleteOrganization: async (id) => {
          const state = get()
          const originalOrgs = [...state.organizations]

          // Optimistic removal
          set(
            {
              organizations: state.organizations.filter((org) => org.id !== id),
              currentOrgId: state.currentOrgId === id ? null : state.currentOrgId,
              loading: { ...state.loading, deleting: true },
              error: null,
            },
            false,
            'deleteOrganization/optimistic'
          )

          try {
            await authFetch(`/api/organizations/${id}`, {
              method: 'DELETE',
            })

            set(
              { loading: { ...state.loading, deleting: false } },
              false,
              'deleteOrganization/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to delete organization'

            // Rollback optimistic removal
            set(
              {
                organizations: originalOrgs,
                currentOrgId: state.currentOrgId,
                error: errorMessage,
                loading: { ...state.loading, deleting: false },
              },
              false,
              'deleteOrganization/rollback'
            )

            throw error
          }
        },

        switchOrganization: (id) => {
          set({ currentOrgId: id }, false, 'switchOrganization')

          // Fetch members and invitations for the new organization
          if (id) {
            get().fetchMembers(id).catch((error) => {
              console.error('Failed to fetch members:', error)
            })

            get().fetchInvitations(id).catch((error) => {
              console.error('Failed to fetch invitations:', error)
            })
          } else {
            // Clear cached data when switching to no organization
            set(
              { currentOrgMembers: [], currentOrgInvitations: [] },
              false,
              'switchOrganization/clearCache'
            )
          }
        },

        // ================================================================
        // Member Management
        // ================================================================

        fetchMembers: async (orgId) => {
          set(
            { loading: { ...get().loading, members: true }, error: null },
            false,
            'fetchMembers/start'
          )

          try {
            const response = await authFetch<{ members: OrganizationMember[] }>(
              `/api/organizations/${orgId}/members`
            )

            const members = (response.members ?? []).map(parseMbrDates)

            // Update cache if this is the current org
            const updates: Partial<OrganizationState> = {
              loading: { ...get().loading, members: false },
            }

            if (get().currentOrgId === orgId) {
              updates.currentOrgMembers = members
            }

            set(updates, false, 'fetchMembers/success')

            return members
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to fetch members'

            set(
              {
                error: errorMessage,
                loading: { ...get().loading, members: false },
              },
              false,
              'fetchMembers/error'
            )

            throw error
          }
        },

        updateMemberRole: async (orgId, userId, role) => {
          const state = get()
          const originalMembers = [...state.currentOrgMembers]
          const memberIndex = originalMembers.findIndex(
            (m) => m.user_id === userId
          )

          if (memberIndex === -1) {
            throw new Error('Member not found')
          }

          // Optimistic update
          const optimisticMembers = [...originalMembers]
          optimisticMembers[memberIndex] = {
            ...optimisticMembers[memberIndex],
            role,
          }

          set(
            {
              currentOrgMembers: optimisticMembers,
              loading: { ...state.loading, updating: true },
              error: null,
            },
            false,
            'updateMemberRole/optimistic'
          )

          try {
            await authFetch(
              `/api/organizations/${orgId}/members/${userId}/role`,
              {
                method: 'PUT',
                body: JSON.stringify({ role }),
              }
            )

            set(
              { loading: { ...state.loading, updating: false } },
              false,
              'updateMemberRole/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to update member role'

            // Rollback optimistic update
            set(
              {
                currentOrgMembers: originalMembers,
                error: errorMessage,
                loading: { ...state.loading, updating: false },
              },
              false,
              'updateMemberRole/rollback'
            )

            throw error
          }
        },

        removeMember: async (orgId, userId) => {
          const state = get()
          const originalMembers = [...state.currentOrgMembers]

          // Optimistic removal
          set(
            {
              currentOrgMembers: originalMembers.filter(
                (m) => m.user_id !== userId
              ),
              loading: { ...state.loading, deleting: true },
              error: null,
            },
            false,
            'removeMember/optimistic'
          )

          try {
            await authFetch(`/api/organizations/${orgId}/members/${userId}`, {
              method: 'DELETE',
            })

            // Update member count in organization
            set(
              {
                organizations: state.organizations.map((org) =>
                  org.id === orgId && org.member_count
                    ? { ...org, member_count: org.member_count - 1 }
                    : org
                ),
                loading: { ...state.loading, deleting: false },
              },
              false,
              'removeMember/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to remove member'

            // Rollback optimistic removal
            set(
              {
                currentOrgMembers: originalMembers,
                error: errorMessage,
                loading: { ...state.loading, deleting: false },
              },
              false,
              'removeMember/rollback'
            )

            throw error
          }
        },

        // ================================================================
        // Invitation Management
        // ================================================================

        createInvitation: async (orgId, input) => {
          const state = get()

          set(
            { loading: { ...state.loading, creating: true }, error: null },
            false,
            'createInvitation/start'
          )

          try {
            const response = await authFetch<{ invitation: OrganizationInvitation }>(
              `/api/organizations/${orgId}/invitations`,
              {
                method: 'POST',
                body: JSON.stringify(input),
              }
            )

            const newInvitation = parseInvDates(response.invitation)

            // Add to current org invitations if this is the current org
            if (state.currentOrgId === orgId) {
              set(
                {
                  currentOrgInvitations: [
                    ...state.currentOrgInvitations,
                    newInvitation,
                  ],
                  loading: { ...state.loading, creating: false },
                },
                false,
                'createInvitation/success'
              )
            } else {
              set(
                { loading: { ...state.loading, creating: false } },
                false,
                'createInvitation/success'
              )
            }

            return newInvitation
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to create invitation'

            set(
              {
                error: errorMessage,
                loading: { ...state.loading, creating: false },
              },
              false,
              'createInvitation/error'
            )

            throw error
          }
        },

        fetchInvitations: async (orgId) => {
          set(
            { loading: { ...get().loading, invitations: true }, error: null },
            false,
            'fetchInvitations/start'
          )

          try {
            const response = await authFetch<{ invitations: OrganizationInvitation[] }>(
              `/api/organizations/${orgId}/invitations`
            )

            const invitations = (response.invitations ?? []).map(parseInvDates)

            // Update cache if this is the current org
            const updates: Partial<OrganizationState> = {
              loading: { ...get().loading, invitations: false },
            }

            if (get().currentOrgId === orgId) {
              updates.currentOrgInvitations = invitations
            }

            set(updates, false, 'fetchInvitations/success')

            return invitations
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to fetch invitations'

            set(
              {
                error: errorMessage,
                loading: { ...get().loading, invitations: false },
              },
              false,
              'fetchInvitations/error'
            )

            throw error
          }
        },

        fetchPendingInvitations: async () => {
          set(
            { loading: { ...get().loading, invitations: true }, error: null },
            false,
            'fetchPendingInvitations/start'
          )

          try {
            const response = await authFetch<{ invitations: OrganizationInvitation[] }>(
              '/api/invitations/pending'
            )

            const invitations = (response.invitations ?? []).map(parseInvDates)

            set(
              {
                pendingInvitations: invitations,
                loading: { ...get().loading, invitations: false },
              },
              false,
              'fetchPendingInvitations/success'
            )

            return invitations
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to fetch pending invitations'

            set(
              {
                error: errorMessage,
                loading: { ...get().loading, invitations: false },
              },
              false,
              'fetchPendingInvitations/error'
            )

            throw error
          }
        },

        acceptInvitation: async (inviteId) => {
          const state = get()
          const originalInvitations = [...state.pendingInvitations]

          // Optimistic removal from pending
          set(
            {
              pendingInvitations: originalInvitations.filter(
                (inv) => inv.id !== inviteId
              ),
              loading: { ...state.loading, updating: true },
              error: null,
            },
            false,
            'acceptInvitation/optimistic'
          )

          try {
            await authFetch(`/api/invitations/${inviteId}/accept`, {
              method: 'POST',
            })

            set(
              { loading: { ...state.loading, updating: false } },
              false,
              'acceptInvitation/success'
            )

            // Refresh organizations list as user now has a new org
            await get().fetchOrganizations()
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to accept invitation'

            // Rollback optimistic removal
            set(
              {
                pendingInvitations: originalInvitations,
                error: errorMessage,
                loading: { ...state.loading, updating: false },
              },
              false,
              'acceptInvitation/rollback'
            )

            throw error
          }
        },

        declineInvitation: async (inviteId) => {
          const state = get()
          const originalInvitations = [...state.pendingInvitations]

          // Optimistic removal from pending
          set(
            {
              pendingInvitations: originalInvitations.filter(
                (inv) => inv.id !== inviteId
              ),
              loading: { ...state.loading, deleting: true },
              error: null,
            },
            false,
            'declineInvitation/optimistic'
          )

          try {
            await authFetch(`/api/invitations/${inviteId}/decline`, {
              method: 'POST',
            })

            set(
              { loading: { ...state.loading, deleting: false } },
              false,
              'declineInvitation/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to decline invitation'

            // Rollback optimistic removal
            set(
              {
                pendingInvitations: originalInvitations,
                error: errorMessage,
                loading: { ...state.loading, deleting: false },
              },
              false,
              'declineInvitation/rollback'
            )

            throw error
          }
        },

        revokeInvitation: async (orgId, inviteId) => {
          const state = get()
          const originalInvitations = [...state.currentOrgInvitations]

          // Optimistic removal
          set(
            {
              currentOrgInvitations: originalInvitations.filter(
                (inv) => inv.id !== inviteId
              ),
              loading: { ...state.loading, deleting: true },
              error: null,
            },
            false,
            'revokeInvitation/optimistic'
          )

          try {
            await authFetch(
              `/api/organizations/${orgId}/invitations/${inviteId}`,
              {
                method: 'DELETE',
              }
            )

            set(
              { loading: { ...state.loading, deleting: false } },
              false,
              'revokeInvitation/success'
            )
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to revoke invitation'

            // Rollback optimistic removal
            set(
              {
                currentOrgInvitations: originalInvitations,
                error: errorMessage,
                loading: { ...state.loading, deleting: false },
              },
              false,
              'revokeInvitation/rollback'
            )

            throw error
          }
        },

        // ================================================================
        // Audit Logs
        // ================================================================

        fetchAuditLogs: async (orgId, params = {}) => {
          try {
            const queryParams = new URLSearchParams()

            if (params.action) queryParams.append('action', params.action)
            if (params.resource_type)
              queryParams.append('resource_type', params.resource_type)
            if (params.user_id) queryParams.append('user_id', params.user_id)
            if (params.start_date)
              queryParams.append(
                'start_date',
                params.start_date.toISOString()
              )
            if (params.end_date)
              queryParams.append('end_date', params.end_date.toISOString())
            if (params.limit) queryParams.append('limit', params.limit.toString())
            if (params.offset)
              queryParams.append('offset', params.offset.toString())

            const query = queryParams.toString()
            const endpoint = `/api/organizations/${orgId}/audit-logs${
              query ? `?${query}` : ''
            }`

            const response = await authFetch<{ logs: AuditLog[] }>(endpoint)

            return (response.logs ?? []).map(parseLogDates)
          } catch (error) {
            const errorMessage =
              error instanceof AuthApiError
                ? error.message
                : 'Failed to fetch audit logs'

            set({ error: errorMessage }, false, 'fetchAuditLogs/error')

            throw error
          }
        },

        // ================================================================
        // Utility Methods
        // ================================================================

        clearError: () => {
          set({ error: null }, false, 'clearError')
        },

        currentOrg: () => {
          const state = get()
          if (!state.currentOrgId) return null

          return (
            state.organizations.find((org) => org.id === state.currentOrgId) ||
            null
          )
        },

        currentUserRole: () => {
          const state = get()
          const org = get().currentOrg()

          if (!org) return null

          // Find user's membership in the organization
          const member = state.currentOrgMembers.find(
            (m) => m.user_id === org.owner_id // This is simplified, should use actual user ID
          )

          return member?.role || null
        },

        refreshCurrentOrg: async () => {
          const state = get()

          if (!state.currentOrgId) return

          await get().fetchMembers(state.currentOrgId)
          await get().fetchInvitations(state.currentOrgId)
        },
      }),
      {
        name: 'sql-studio-organization-storage',
        version: 1,
        // Only persist specific fields
        partialize: (state) => ({
          currentOrgId: state.currentOrgId,
          // Don't persist organizations array - fetch fresh on load
          // Don't persist: loading states, errors, cached members/invitations
        }),
      }
    ),
    {
      name: 'OrganizationStore',
      enabled: import.meta.env.DEV,
    }
  )
)

/**
 * Initialize organization store on app startup
 * Call this in your main App component
 */
export const initializeOrganizationStore = async () => {
  const state = useOrganizationStore.getState()

  try {
    // Fetch organizations on startup
    await state.fetchOrganizations()

    // Fetch pending invitations
    await state.fetchPendingInvitations()

    // If there's a current org, fetch its data
    if (state.currentOrgId) {
      await state.refreshCurrentOrg()
    }
  } catch (error) {
    console.error('Failed to initialize organization store:', error)
  }
}

/**
 * Selectors for common organization checks
 */
export const organizationSelectors = {
  hasOrganizations: (state: OrganizationStore) =>
    state.organizations.length > 0,
  isLoading: (state: OrganizationStore) =>
    Object.values(state.loading).some((loading) => loading),
  hasError: (state: OrganizationStore) => !!state.error,
  hasPendingInvitations: (state: OrganizationStore) =>
    state.pendingInvitations.length > 0,
  getCurrentOrgMemberCount: (state: OrganizationStore) =>
    state.currentOrgMembers.length,
}

/**
 * Hook for organization state
 */
export const useOrganization = () => {
  const organizations = useOrganizationStore((state) => state.organizations)
  const currentOrgId = useOrganizationStore((state) => state.currentOrgId)
  const currentOrg = useOrganizationStore((state) => state.currentOrg())
  const loading = useOrganizationStore((state) => state.loading)
  const error = useOrganizationStore((state) => state.error)

  return {
    organizations,
    currentOrgId,
    currentOrg,
    loading,
    error,
    hasOrganizations: organizations.length > 0,
    isLoading: Object.values(loading).some((l) => l),
  }
}

/**
 * Hook for organization actions
 */
export const useOrganizationActions = () => {
  const fetchOrganizations = useOrganizationStore(
    (state) => state.fetchOrganizations
  )
  const createOrganization = useOrganizationStore(
    (state) => state.createOrganization
  )
  const updateOrganization = useOrganizationStore(
    (state) => state.updateOrganization
  )
  const deleteOrganization = useOrganizationStore(
    (state) => state.deleteOrganization
  )
  const switchOrganization = useOrganizationStore(
    (state) => state.switchOrganization
  )
  const clearError = useOrganizationStore((state) => state.clearError)

  return {
    fetchOrganizations,
    createOrganization,
    updateOrganization,
    deleteOrganization,
    switchOrganization,
    clearError,
  }
}

/**
 * Hook for member management
 */
export const useOrganizationMembers = () => {
  const members = useOrganizationStore((state) => state.currentOrgMembers)
  const fetchMembers = useOrganizationStore((state) => state.fetchMembers)
  const updateMemberRole = useOrganizationStore(
    (state) => state.updateMemberRole
  )
  const removeMember = useOrganizationStore((state) => state.removeMember)
  const loading = useOrganizationStore((state) => state.loading.members)

  return {
    members,
    fetchMembers,
    updateMemberRole,
    removeMember,
    loading,
  }
}

/**
 * Hook for invitation management
 */
export const useOrganizationInvitations = () => {
  const invitations = useOrganizationStore(
    (state) => state.currentOrgInvitations
  )
  const pendingInvitations = useOrganizationStore(
    (state) => state.pendingInvitations
  )
  const createInvitation = useOrganizationStore(
    (state) => state.createInvitation
  )
  const fetchInvitations = useOrganizationStore(
    (state) => state.fetchInvitations
  )
  const fetchPendingInvitations = useOrganizationStore(
    (state) => state.fetchPendingInvitations
  )
  const acceptInvitation = useOrganizationStore(
    (state) => state.acceptInvitation
  )
  const declineInvitation = useOrganizationStore(
    (state) => state.declineInvitation
  )
  const revokeInvitation = useOrganizationStore(
    (state) => state.revokeInvitation
  )
  const loading = useOrganizationStore((state) => state.loading.invitations)

  return {
    invitations,
    pendingInvitations,
    createInvitation,
    fetchInvitations,
    fetchPendingInvitations,
    acceptInvitation,
    declineInvitation,
    revokeInvitation,
    loading,
    hasPendingInvitations: pendingInvitations.length > 0,
  }
}
