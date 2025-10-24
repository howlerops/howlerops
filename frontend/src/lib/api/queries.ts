/**
 * Queries API Client
 *
 * HTTP client for saved query management with organization sharing support.
 * Provides type-safe functions for CRUD operations and sharing queries.
 *
 * @module lib/api/queries
 */

import { authFetch } from './auth-client'

/**
 * Query visibility modes
 */
export type QueryVisibility = 'personal' | 'shared'

/**
 * Saved query entity (server response)
 */
export interface SavedQuery {
  id: string
  user_id: string
  organization_id?: string | null
  title: string
  description?: string
  sql_content: string
  database_type: string
  tags?: string[]
  visibility: QueryVisibility
  created_at: string
  updated_at: string
  created_by_email?: string
  last_executed?: string
}

/**
 * Input for creating a saved query
 */
export interface CreateQueryInput {
  title: string
  description?: string
  sql_content: string
  database_type: string
  tags?: string[]
  organization_id?: string
  visibility?: QueryVisibility
}

/**
 * Input for updating a saved query
 */
export interface UpdateQueryInput {
  title?: string
  description?: string
  sql_content?: string
  tags?: string[]
  visibility?: QueryVisibility
  organization_id?: string
}

/**
 * API response wrapper
 */
interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

// ============================================================================
// Saved Query CRUD Operations
// ============================================================================

/**
 * Create a new saved query
 */
export async function createQuery(
  input: CreateQueryInput
): Promise<SavedQuery> {
  const response = await authFetch<ApiResponse<SavedQuery>>(
    '/api/queries',
    {
      method: 'POST',
      body: JSON.stringify(input),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to create query')
  }

  return response.data
}

/**
 * Get all saved queries for current user
 */
export async function getQueries(): Promise<SavedQuery[]> {
  const response = await authFetch<ApiResponse<{ queries: SavedQuery[] }>>(
    '/api/queries'
  )

  return response.data?.queries || []
}

/**
 * Get a specific query by ID
 */
export async function getQuery(id: string): Promise<SavedQuery> {
  const response = await authFetch<ApiResponse<SavedQuery>>(
    `/api/queries/${id}`
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to fetch query')
  }

  return response.data
}

/**
 * Update a saved query
 */
export async function updateQuery(
  id: string,
  input: UpdateQueryInput
): Promise<SavedQuery> {
  const response = await authFetch<ApiResponse<SavedQuery>>(
    `/api/queries/${id}`,
    {
      method: 'PUT',
      body: JSON.stringify(input),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to update query')
  }

  return response.data
}

/**
 * Delete a saved query
 */
export async function deleteQuery(id: string): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/queries/${id}`,
    {
      method: 'DELETE',
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete query')
  }
}

// ============================================================================
// Sharing Operations
// ============================================================================

/**
 * Share a query within an organization
 */
export async function shareQuery(
  queryId: string,
  organizationId: string
): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/queries/${queryId}/share`,
    {
      method: 'POST',
      body: JSON.stringify({ organization_id: organizationId }),
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to share query')
  }
}

/**
 * Unshare a query (make it personal)
 */
export async function unshareQuery(queryId: string): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/queries/${queryId}/unshare`,
    {
      method: 'POST',
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to unshare query')
  }
}

/**
 * Get all shared queries in an organization
 */
export async function getOrganizationQueries(
  orgId: string
): Promise<SavedQuery[]> {
  const response = await authFetch<ApiResponse<{ queries: SavedQuery[] }>>(
    `/api/organizations/${orgId}/queries`
  )

  return response.data?.queries || []
}

// ============================================================================
// Query Execution (via existing connection)
// ============================================================================

/**
 * Execute a saved query
 * Note: Actual execution happens via Wails backend with active connection
 */
export async function executeQuery(queryId: string, connectionId: string): Promise<{
  columns: string[]
  rows: unknown[]
  rowCount: number
  executionTime: number
}> {
  const response = await authFetch<ApiResponse<{
    columns: string[]
    rows: unknown[]
    rowCount: number
    executionTime: number
  }>>(`/api/queries/${queryId}/execute`, {
    method: 'POST',
    body: JSON.stringify({ connection_id: connectionId }),
  })

  if (!response.data) {
    throw new Error(response.message || 'Failed to execute query')
  }

  return response.data
}
