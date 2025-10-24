/**
 * Connections API Client
 *
 * HTTP client for database connection management with organization sharing support.
 * Provides type-safe functions for CRUD operations and sharing connections.
 *
 * @module lib/api/connections
 */

import { authFetch } from './auth-client'

/**
 * Database connection visibility modes
 */
export type ConnectionVisibility = 'personal' | 'shared'

/**
 * Database connection entity (server response)
 */
export interface Connection {
  id: string
  user_id: string
  organization_id?: string | null
  name: string
  description?: string
  database_type: string
  host: string
  port: number
  database_name: string
  username: string
  // Password is NEVER returned from server
  ssl_enabled: boolean
  visibility: ConnectionVisibility
  created_at: string
  updated_at: string
  created_by_email?: string
  last_used?: string
}

/**
 * Input for creating a connection
 */
export interface CreateConnectionInput {
  name: string
  description?: string
  database_type: string
  host: string
  port: number
  database_name: string
  username: string
  password: string
  ssl_enabled?: boolean
  organization_id?: string
  visibility?: ConnectionVisibility
}

/**
 * Input for updating a connection
 */
export interface UpdateConnectionInput {
  name?: string
  description?: string
  host?: string
  port?: number
  database_name?: string
  username?: string
  password?: string
  ssl_enabled?: boolean
  visibility?: ConnectionVisibility
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
// Connection CRUD Operations
// ============================================================================

/**
 * Create a new database connection
 */
export async function createConnection(
  input: CreateConnectionInput
): Promise<Connection> {
  const response = await authFetch<ApiResponse<Connection>>(
    '/api/connections',
    {
      method: 'POST',
      body: JSON.stringify(input),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to create connection')
  }

  return response.data
}

/**
 * Get all connections for current user
 */
export async function getConnections(): Promise<Connection[]> {
  const response = await authFetch<ApiResponse<{ connections: Connection[] }>>(
    '/api/connections'
  )

  return response.data?.connections || []
}

/**
 * Get a specific connection by ID
 */
export async function getConnection(id: string): Promise<Connection> {
  const response = await authFetch<ApiResponse<Connection>>(
    `/api/connections/${id}`
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to fetch connection')
  }

  return response.data
}

/**
 * Update a connection
 */
export async function updateConnection(
  id: string,
  input: UpdateConnectionInput
): Promise<Connection> {
  const response = await authFetch<ApiResponse<Connection>>(
    `/api/connections/${id}`,
    {
      method: 'PUT',
      body: JSON.stringify(input),
    }
  )

  if (!response.data) {
    throw new Error(response.message || 'Failed to update connection')
  }

  return response.data
}

/**
 * Delete a connection
 */
export async function deleteConnection(id: string): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/connections/${id}`,
    {
      method: 'DELETE',
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete connection')
  }
}

// ============================================================================
// Sharing Operations
// ============================================================================

/**
 * Share a connection within an organization
 */
export async function shareConnection(
  connectionId: string,
  organizationId: string
): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/connections/${connectionId}/share`,
    {
      method: 'POST',
      body: JSON.stringify({ organization_id: organizationId }),
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to share connection')
  }
}

/**
 * Unshare a connection (make it personal)
 */
export async function unshareConnection(connectionId: string): Promise<void> {
  const response = await authFetch<ApiResponse<void>>(
    `/api/connections/${connectionId}/unshare`,
    {
      method: 'POST',
    }
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to unshare connection')
  }
}

/**
 * Get all shared connections in an organization
 */
export async function getOrganizationConnections(
  orgId: string
): Promise<Connection[]> {
  const response = await authFetch<ApiResponse<{ connections: Connection[] }>>(
    `/api/organizations/${orgId}/connections`
  )

  return response.data?.connections || []
}

// ============================================================================
// Test Connection
// ============================================================================

/**
 * Test a connection without saving it
 */
export async function testConnection(
  input: Omit<CreateConnectionInput, 'name' | 'organization_id' | 'visibility'>
): Promise<{ success: boolean; message: string }> {
  const response = await authFetch<
    ApiResponse<{ success: boolean; message: string }>
  >('/api/connections/test', {
    method: 'POST',
    body: JSON.stringify(input),
  })

  return (
    response.data || { success: false, message: 'Connection test failed' }
  )
}
