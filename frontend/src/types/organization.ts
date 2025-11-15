/**
 * Organization System Type Definitions
 *
 * Type-safe interfaces for team collaboration features in Howlerops.
 * Supports multi-user organizations, role-based access, invitations, and audit logging.
 *
 * @module types/organization
 */

// ============================================================================
// Enums and Constants
// ============================================================================

/**
 * Organization role levels with hierarchical permissions
 */
export enum OrganizationRole {
  /** Owner - Full control, can delete organization */
  Owner = 'owner',
  /** Admin - Can manage members and settings */
  Admin = 'admin',
  /** Member - Basic access, can view and edit resources */
  Member = 'member',
}

/**
 * Audit log action types
 */
export type AuditAction =
  | 'organization.created'
  | 'organization.updated'
  | 'organization.deleted'
  | 'member.added'
  | 'member.removed'
  | 'member.role_updated'
  | 'invitation.created'
  | 'invitation.accepted'
  | 'invitation.declined'
  | 'invitation.revoked'
  | 'connection.created'
  | 'connection.updated'
  | 'connection.deleted'
  | 'query.executed'
  | 'query.shared'

/**
 * Resource types for audit logging
 */
export type ResourceType =
  | 'organization'
  | 'member'
  | 'invitation'
  | 'connection'
  | 'query'
  | 'settings'

// ============================================================================
// Core Data Models
// ============================================================================

/**
 * Organization (team/workspace) entity
 */
export interface Organization {
  /** Unique organization identifier */
  id: string

  /** Organization name */
  name: string

  /** Optional description */
  description?: string

  /** User ID of the organization owner */
  owner_id: string

  /** When the organization was created */
  created_at: Date

  /** When the organization was last updated */
  updated_at: Date

  /** Soft delete timestamp (null if active) */
  deleted_at?: Date | null

  /** Maximum number of members allowed */
  max_members: number

  /** Additional organization settings (JSON) */
  settings?: Record<string, unknown>

  /** Current member count (computed) */
  member_count?: number
}

type OrganizationJson = Omit<Organization, 'created_at' | 'updated_at' | 'deleted_at'> & {
  created_at: string | Date
  updated_at: string | Date
  deleted_at?: string | Date | null
}

/**
 * Basic user information for organization members
 */
export interface UserInfo {
  /** User identifier */
  id: string

  /** User email address */
  email: string

  /** Username */
  username: string

  /** Optional display name */
  display_name?: string | null
}

/**
 * Organization member with role and user details
 */
export interface OrganizationMember {
  /** Unique member record identifier */
  id: string

  /** Organization this member belongs to */
  organization_id: string

  /** User identifier */
  user_id: string

  /** Member's role in the organization */
  role: OrganizationRole

  /** Who invited this member (user ID) */
  invited_by?: string | null

  /** When the member joined */
  joined_at: Date

  /** User details (joined from users table) */
  user?: UserInfo
}

type OrganizationMemberJson = Omit<OrganizationMember, 'joined_at'> & {
  joined_at: string | Date
}

/**
 * Organization invitation entity
 */
export interface OrganizationInvitation {
  /** Unique invitation identifier */
  id: string

  /** Organization the invitation is for */
  organization_id: string

  /** Email address of invitee */
  email: string

  /** Role the invitee will have when accepted */
  role: OrganizationRole

  /** User ID who created the invitation */
  invited_by: string

  /** Secure invitation token */
  token: string

  /** When the invitation expires */
  expires_at: Date

  /** When the invitation was accepted (null if pending) */
  accepted_at?: Date | null

  /** When the invitation was created */
  created_at: Date

  /** Organization details (optional, for pending invitations view) */
  organization?: Organization
}

type OrganizationInvitationJson = Omit<OrganizationInvitation, 'expires_at' | 'accepted_at' | 'created_at' | 'organization'> & {
  expires_at: string | Date
  accepted_at?: string | Date | null
  created_at: string | Date
  organization?: OrganizationJson
}

/**
 * Audit log entry for tracking organization activities
 */
export interface AuditLog {
  /** Unique log entry identifier */
  id: string

  /** Organization context (null for user-level actions) */
  organization_id?: string | null

  /** User who performed the action */
  user_id: string

  /** Action performed */
  action: AuditAction

  /** Type of resource affected */
  resource_type: ResourceType

  /** Specific resource identifier */
  resource_id?: string | null

  /** IP address of the user */
  ip_address?: string | null

  /** User agent string */
  user_agent?: string | null

  /** Additional action details (JSON) */
  details?: Record<string, unknown>

  /** When the action occurred */
  created_at: Date
}

type AuditLogJson = Omit<AuditLog, 'created_at'> & {
  created_at: string | Date
}

// ============================================================================
// API Request/Response Types
// ============================================================================

/**
 * Input for creating a new organization
 */
export interface CreateOrganizationInput {
  /** Organization name (3-50 characters) */
  name: string

  /** Optional description (max 500 characters) */
  description?: string
}

/**
 * Input for updating an organization
 */
export interface UpdateOrganizationInput {
  /** New organization name (3-50 characters) */
  name?: string

  /** New description (max 500 characters) */
  description?: string

  /** New maximum member count (1-1000) */
  max_members?: number
}

/**
 * Input for creating an invitation
 */
export interface CreateInvitationInput {
  /** Email address to invite */
  email: string

  /** Role for the new member (cannot be 'owner') */
  role: OrganizationRole.Admin | OrganizationRole.Member
}

/**
 * Input for updating a member's role
 */
export interface UpdateMemberRoleInput {
  /** New role for the member */
  role: OrganizationRole
}

/**
 * Response from creating an organization
 */
export interface CreateOrganizationResponse {
  /** The created organization */
  organization: Organization

  /** Success message */
  message?: string
}

/**
 * Response from organization list endpoint
 */
export interface ListOrganizationsResponse {
  /** Array of organizations */
  organizations: Organization[]

  /** Total count (for pagination) */
  total?: number
}

/**
 * Response from members list endpoint
 */
export interface ListMembersResponse {
  /** Array of members */
  members: OrganizationMember[]

  /** Total count (for pagination) */
  total?: number
}

/**
 * Response from invitations list endpoint
 */
export interface ListInvitationsResponse {
  /** Array of invitations */
  invitations: OrganizationInvitation[]

  /** Total count (for pagination) */
  total?: number
}

/**
 * Response from audit logs endpoint
 */
export interface ListAuditLogsResponse {
  /** Array of audit log entries */
  logs: AuditLog[]

  /** Total count (for pagination) */
  total?: number

  /** Whether more logs are available */
  has_more?: boolean
}

/**
 * Response from creating an invitation
 */
export interface CreateInvitationResponse {
  /** The created invitation */
  invitation: OrganizationInvitation

  /** Success message */
  message?: string
}

/**
 * Query parameters for listing audit logs
 */
export interface AuditLogQueryParams {
  /** Filter by action type */
  action?: AuditAction

  /** Filter by resource type */
  resource_type?: ResourceType

  /** Filter by user ID */
  user_id?: string

  /** Start date for time range */
  start_date?: Date

  /** End date for time range */
  end_date?: Date

  /** Number of results per page */
  limit?: number

  /** Offset for pagination */
  offset?: number
}

/**
 * Organization with current user's membership info
 */
export interface OrganizationWithMembership extends Organization {
  /** Current user's role in this organization */
  current_user_role: OrganizationRole
}

// ============================================================================
// Validation Types
// ============================================================================

/**
 * Validation error type
 */
export interface ValidationError {
  field: string
  message: string
}

/**
 * Validation result type
 */
export interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
}

// ============================================================================
// Validation Functions
// ============================================================================

/**
 * Validate email format
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

/**
 * Validate organization role
 */
export function isValidRole(role: string): role is OrganizationRole {
  return Object.values(OrganizationRole).includes(role as OrganizationRole)
}

/**
 * Validate CreateOrganizationInput
 */
export function validateCreateOrganizationInput(
  input: CreateOrganizationInput
): ValidationResult {
  const errors: ValidationError[] = []

  if (!input.name || input.name.trim().length < 3) {
    errors.push({
      field: 'name',
      message: 'Organization name must be at least 3 characters',
    })
  }

  if (input.name && input.name.length > 50) {
    errors.push({
      field: 'name',
      message: 'Organization name must not exceed 50 characters',
    })
  }

  if (input.description && input.description.length > 500) {
    errors.push({
      field: 'description',
      message: 'Description must not exceed 500 characters',
    })
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * Validate UpdateOrganizationInput
 */
export function validateUpdateOrganizationInput(
  input: UpdateOrganizationInput
): ValidationResult {
  const errors: ValidationError[] = []

  if (input.name !== undefined) {
    if (!input.name || input.name.trim().length < 3) {
      errors.push({
        field: 'name',
        message: 'Organization name must be at least 3 characters',
      })
    }

    if (input.name.length > 50) {
      errors.push({
        field: 'name',
        message: 'Organization name must not exceed 50 characters',
      })
    }
  }

  if (
    input.description !== undefined &&
    input.description &&
    input.description.length > 500
  ) {
    errors.push({
      field: 'description',
      message: 'Description must not exceed 500 characters',
    })
  }

  if (input.max_members !== undefined) {
    if (input.max_members < 1) {
      errors.push({
        field: 'max_members',
        message: 'Maximum members must be at least 1',
      })
    }

    if (input.max_members > 1000) {
      errors.push({
        field: 'max_members',
        message: 'Maximum members cannot exceed 1000',
      })
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * Validate CreateInvitationInput
 */
export function validateCreateInvitationInput(
  input: CreateInvitationInput
): ValidationResult {
  const errors: ValidationError[] = []

  if (!input.email) {
    errors.push({
      field: 'email',
      message: 'Email is required',
    })
  } else if (!isValidEmail(input.email)) {
    errors.push({
      field: 'email',
      message: 'Invalid email format',
    })
  }

  if (!input.role) {
    errors.push({
      field: 'role',
      message: 'Role is required',
    })
  } else if (!isValidRole(input.role as string)) {
    errors.push({
      field: 'role',
      message: 'Invalid role',
    })
  }
  // Note: TypeScript already prevents Owner role in CreateInvitationInput

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * Validate UpdateMemberRoleInput
 */
export function validateUpdateMemberRoleInput(
  input: UpdateMemberRoleInput
): ValidationResult {
  const errors: ValidationError[] = []

  if (!input.role) {
    errors.push({
      field: 'role',
      message: 'Role is required',
    })
  } else if (!isValidRole(input.role as string)) {
    errors.push({
      field: 'role',
      message: 'Invalid role',
    })
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

// ============================================================================
// Permission Helper Functions
// ============================================================================

/**
 * Check if a user has a specific role or higher
 */
export function hasRoleOrHigher(
  userRole: OrganizationRole,
  requiredRole: OrganizationRole
): boolean {
  const roleHierarchy = {
    [OrganizationRole.Owner]: 3,
    [OrganizationRole.Admin]: 2,
    [OrganizationRole.Member]: 1,
  }

  return roleHierarchy[userRole] >= roleHierarchy[requiredRole]
}

/**
 * Check if a member can invite other members
 */
export function canInviteMembers(role: OrganizationRole): boolean {
  return role === OrganizationRole.Owner || role === OrganizationRole.Admin
}

/**
 * Check if a member can remove other members
 */
export function canRemoveMembers(role: OrganizationRole): boolean {
  return role === OrganizationRole.Owner || role === OrganizationRole.Admin
}

/**
 * Check if a member can update organization settings
 */
export function canUpdateSettings(role: OrganizationRole): boolean {
  return role === OrganizationRole.Owner || role === OrganizationRole.Admin
}

/**
 * Check if a member can delete the organization
 */
export function canDeleteOrganization(role: OrganizationRole): boolean {
  return role === OrganizationRole.Owner
}

/**
 * Check if current user can change target member's role
 */
export function canChangeRole(
  currentRole: OrganizationRole,
  targetRole: OrganizationRole
): boolean {
  // Owners can change anyone's role
  if (currentRole === OrganizationRole.Owner) return true

  // Admins can change member roles but not owner roles
  if (
    currentRole === OrganizationRole.Admin &&
    targetRole !== OrganizationRole.Owner
  )
    return true

  return false
}

/**
 * Check if an invitation is still valid (not expired, not accepted)
 */
export function isInvitationValid(
  invitation: OrganizationInvitation
): boolean {
  if (invitation.accepted_at) {
    return false
  }

  return new Date() < new Date(invitation.expires_at)
}

/**
 * Check if a user is the owner of an organization
 */
export function isOwner(organization: Organization, userId: string): boolean {
  return organization.owner_id === userId
}

// ============================================================================
// Display Helper Functions
// ============================================================================

/**
 * Get display name for a role
 */
export function getRoleDisplayName(role: OrganizationRole): string {
  const displayNames = {
    [OrganizationRole.Owner]: 'Owner',
    [OrganizationRole.Admin]: 'Admin',
    [OrganizationRole.Member]: 'Member',
  }

  return displayNames[role] || 'Unknown'
}

/**
 * Get badge variant for a role (for UI components)
 */
export function getRoleBadgeVariant(
  role: OrganizationRole
): 'default' | 'secondary' | 'outline' {
  switch (role) {
    case OrganizationRole.Owner:
      return 'default'
    case OrganizationRole.Admin:
      return 'secondary'
    case OrganizationRole.Member:
      return 'outline'
  }
}

/**
 * Get color for a role (for UI badges)
 */
export function getRoleColor(role: OrganizationRole): string {
  const colors = {
    [OrganizationRole.Owner]: 'purple',
    [OrganizationRole.Admin]: 'blue',
    [OrganizationRole.Member]: 'gray',
  }

  return colors[role] || 'gray'
}

// ============================================================================
// Date Parsing Utilities
// ============================================================================

/**
 * Parse JSON date strings to Date objects for Organization
 */
export function parseOrganizationDates(org: OrganizationJson): Organization {
  return {
    ...org,
    created_at: new Date(org.created_at),
    updated_at: new Date(org.updated_at),
    deleted_at: org.deleted_at ? new Date(org.deleted_at) : null,
  }
}

/**
 * Parse JSON date strings for OrganizationMember
 */
export function parseMemberDates(member: OrganizationMemberJson): OrganizationMember {
  return {
    ...member,
    joined_at: new Date(member.joined_at),
  }
}

/**
 * Parse JSON date strings for OrganizationInvitation
 */
export function parseInvitationDates(
  invitation: OrganizationInvitationJson
): OrganizationInvitation {
  return {
    ...invitation,
    expires_at: new Date(invitation.expires_at),
    accepted_at: invitation.accepted_at
      ? new Date(invitation.accepted_at)
      : null,
    created_at: new Date(invitation.created_at),
    organization: invitation.organization
      ? parseOrganizationDates(invitation.organization)
      : undefined,
  }
}

/**
 * Parse JSON date strings for AuditLog
 */
export function parseAuditLogDates(log: AuditLogJson): AuditLog {
  return {
    ...log,
    created_at: new Date(log.created_at),
  }
}

/**
 * Format relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const months = Math.floor(days / 30)
  const years = Math.floor(days / 365)

  if (years > 0) return `${years} year${years > 1 ? 's' : ''} ago`
  if (months > 0) return `${months} month${months > 1 ? 's' : ''} ago`
  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`
  return 'just now'
}
