# Phase 3: Team Collaboration API Specification

**Version:** 1.0
**Base URL:** `https://api.sqlstudio.io` (Production) | `http://localhost:8500` (Development)
**Authentication:** Bearer JWT Token
**Content-Type:** `application/json`

---

## Table of Contents

1. [Authentication](#authentication)
2. [Rate Limits](#rate-limits)
3. [Error Responses](#error-responses)
4. [Organizations API](#organizations-api)
5. [Members API](#members-api)
6. [Shared Resources API](#shared-resources-api)
7. [Audit Logs API](#audit-logs-api)
8. [Data Models](#data-models)

---

## Authentication

All Phase 3 endpoints require authentication via JWT Bearer token.

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### JWT Claims

```json
{
  "user_id": "user_123abc",
  "email": "user@example.com",
  "organizations": ["org_personal", "org_team123"],
  "exp": 1705334400
}
```

---

## Rate Limits

| Endpoint Category | Rate Limit | Window |
|-------------------|------------|--------|
| Organization CRUD | 100 req | 1 hour |
| Member Management | 50 req | 1 hour |
| Invitations | 10 req | 1 hour |
| Resource Sharing | 100 req | 1 hour |
| Audit Logs | 200 req | 1 hour |

### Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705334400
```

### Rate Limit Exceeded Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 3600

{
  "error": true,
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded. Try again in 3600 seconds.",
  "retry_after": 3600
}
```

---

## Error Responses

All errors follow this standard format:

```json
{
  "error": true,
  "code": "ERROR_CODE",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional context (optional)"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request or validation error |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g., duplicate slug) |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

---

## Organizations API

### Create Organization

Create a new team organization.

```http
POST /api/organizations
```

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp (optional)",
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  }
}
```

**Validation Rules:**
- `name`: Required, 1-255 characters
- `slug`: Required, 3-100 characters, lowercase alphanumeric and hyphens only, unique
- `description`: Optional, max 1000 characters
- `settings`: Optional JSON object

**Success Response (201 Created):**
```json
{
  "id": "org_7x9k2m4n",
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp",
  "owner_id": "user_abc123",
  "plan": "team",
  "max_members": 10,
  "max_connections": null,
  "max_queries_per_month": null,
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  },
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Error Responses:**

```json
// 400 - Invalid slug format
{
  "error": true,
  "code": "INVALID_REQUEST",
  "message": "Invalid slug format",
  "details": {
    "field": "slug",
    "requirement": "Slug must be 3-100 lowercase alphanumeric characters and hyphens"
  }
}

// 409 - Slug already taken
{
  "error": true,
  "code": "CONFLICT",
  "message": "Organization slug already exists",
  "details": {
    "field": "slug",
    "value": "acme-engineering"
  }
}

// 403 - Limit reached
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Maximum organizations limit reached for your plan",
  "details": {
    "current_count": 3,
    "max_allowed": 3,
    "plan": "individual"
  }
}
```

---

### List Organizations

Get all organizations the authenticated user is a member of.

```http
GET /api/organizations
```

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
None

**Success Response (200 OK):**
```json
{
  "organizations": [
    {
      "id": "org_personal_user123",
      "name": "John Doe",
      "slug": "john-doe-personal",
      "description": null,
      "owner_id": "user_abc123",
      "plan": "individual",
      "role": "owner",
      "status": "active",
      "member_count": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "org_7x9k2m4n",
      "name": "Acme Corp Engineering",
      "slug": "acme-engineering",
      "description": "Engineering team at Acme Corp",
      "owner_id": "user_xyz789",
      "plan": "team",
      "role": "member",
      "status": "active",
      "member_count": 5,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 2
}
```

**Notes:**
- Returns both personal and team organizations
- Includes user's role in each organization
- Sorted by created_at descending

---

### Get Organization

Get details for a specific organization.

```http
GET /api/organizations/:id
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Success Response (200 OK):**
```json
{
  "id": "org_7x9k2m4n",
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp",
  "owner_id": "user_xyz789",
  "plan": "team",
  "max_members": 10,
  "max_connections": null,
  "max_queries_per_month": null,
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  },
  "stats": {
    "active_members": 5,
    "pending_invitations": 2,
    "shared_connections": 12,
    "shared_queries": 34,
    "total_queries_this_month": 1234
  },
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Error Responses:**

```json
// 403 - Not a member
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "You are not a member of this organization"
}

// 404 - Not found
{
  "error": true,
  "code": "NOT_FOUND",
  "message": "Organization not found"
}
```

---

### Update Organization

Update organization details. Requires admin or owner role.

```http
PUT /api/organizations/:id
PATCH /api/organizations/:id
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Acme Corp Engineering Team",
  "description": "Updated description",
  "settings": {
    "allow_public_sharing": true,
    "require_2fa": true,
    "new_setting": "value"
  }
}
```

**Notes:**
- All fields are optional
- Settings are merged (not replaced)
- Cannot update: `id`, `slug`, `owner_id`, `plan`

**Success Response (200 OK):**
```json
{
  "id": "org_7x9k2m4n",
  "name": "Acme Corp Engineering Team",
  "slug": "acme-engineering",
  "description": "Updated description",
  "owner_id": "user_xyz789",
  "plan": "team",
  "max_members": 10,
  "settings": {
    "allow_public_sharing": true,
    "require_2fa": true,
    "new_setting": "value"
  },
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T15:30:00Z"
}
```

**Error Responses:**

```json
// 403 - Insufficient permissions
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only owners and admins can update organization settings"
}
```

---

### Delete Organization

Permanently delete an organization. Requires owner role.

```http
DELETE /api/organizations/:id
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Organization deleted successfully",
  "deleted_at": "2024-01-15T16:00:00Z"
}
```

**Error Responses:**

```json
// 403 - Not owner
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only the organization owner can delete it"
}

// 409 - Has active members
{
  "error": true,
  "code": "CONFLICT",
  "message": "Cannot delete organization with active members",
  "details": {
    "active_members": 3,
    "requirement": "Remove all members before deleting organization"
  }
}

// 409 - Personal organization
{
  "error": true,
  "code": "CONFLICT",
  "message": "Cannot delete personal organization",
  "details": {
    "plan": "individual"
  }
}
```

---

## Members API

### Invite Member

Invite a new member to the organization. Requires admin or owner role.

```http
POST /api/organizations/:id/invite
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "newmember@example.com",
  "role": "member"
}
```

**Validation Rules:**
- `email`: Required, valid email format
- `role`: Required, one of: `admin`, `member`, `viewer` (cannot invite as `owner`)

**Success Response (200 OK):**
```json
{
  "success": true,
  "invitation": {
    "id": "inv_abc123",
    "organization_id": "org_7x9k2m4n",
    "email": "newmember@example.com",
    "role": "member",
    "invited_by": "user_abc123",
    "invited_at": "2024-01-15T10:30:00Z",
    "expires_at": "2024-01-22T10:30:00Z",
    "status": "pending",
    "invitation_url": "https://app.sqlstudio.io/accept-invite?token=inv_abc123_secret"
  },
  "message": "Invitation sent to newmember@example.com"
}
```

**Email Sent:**
```
Subject: You've been invited to join Acme Corp Engineering on Howlerops

Hi,

John Doe has invited you to join Acme Corp Engineering on Howlerops as a Member.

[Accept Invitation Button]

This invitation will expire in 7 days.

If you don't have a Howlerops account, you'll be prompted to create one.
```

**Error Responses:**

```json
// 403 - Insufficient permissions
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only owners and admins can invite members"
}

// 409 - Already a member
{
  "error": true,
  "code": "CONFLICT",
  "message": "User is already a member of this organization",
  "details": {
    "email": "newmember@example.com",
    "current_role": "member",
    "status": "active"
  }
}

// 409 - Pending invitation exists
{
  "error": true,
  "code": "CONFLICT",
  "message": "Pending invitation already exists for this email",
  "details": {
    "email": "newmember@example.com",
    "invited_at": "2024-01-14T10:30:00Z",
    "expires_at": "2024-01-21T10:30:00Z"
  }
}

// 409 - Member limit reached
{
  "error": true,
  "code": "CONFLICT",
  "message": "Organization has reached maximum member limit",
  "details": {
    "current_members": 10,
    "max_members": 10,
    "plan": "team"
  }
}

// 400 - Cannot invite as owner
{
  "error": true,
  "code": "INVALID_REQUEST",
  "message": "Cannot invite member with 'owner' role",
  "details": {
    "field": "role",
    "allowed_roles": ["admin", "member", "viewer"]
  }
}
```

---

### List Members

Get all members of an organization.

```http
GET /api/organizations/:id/members
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
- `status` (optional): Filter by status (`active`, `pending`, `suspended`)
- `role` (optional): Filter by role (`owner`, `admin`, `member`, `viewer`)

**Success Response (200 OK):**
```json
{
  "members": [
    {
      "id": "mem_123",
      "organization_id": "org_7x9k2m4n",
      "user_id": "user_xyz789",
      "username": "jane_admin",
      "email": "jane@example.com",
      "role": "owner",
      "status": "active",
      "invited_by": null,
      "invited_at": "2024-01-15T10:00:00Z",
      "joined_at": "2024-01-15T10:00:00Z",
      "last_active": "2024-01-15T14:30:00Z"
    },
    {
      "id": "mem_456",
      "organization_id": "org_7x9k2m4n",
      "user_id": "user_abc123",
      "username": "john_member",
      "email": "john@example.com",
      "role": "member",
      "status": "active",
      "invited_by": "user_xyz789",
      "invited_at": "2024-01-15T10:30:00Z",
      "joined_at": "2024-01-15T11:00:00Z",
      "last_active": "2024-01-15T15:00:00Z"
    },
    {
      "id": "mem_789",
      "organization_id": "org_7x9k2m4n",
      "user_id": null,
      "username": null,
      "email": "pending@example.com",
      "role": "member",
      "status": "pending",
      "invited_by": "user_xyz789",
      "invited_at": "2024-01-15T12:00:00Z",
      "joined_at": null,
      "last_active": null,
      "invitation_expires_at": "2024-01-22T12:00:00Z"
    }
  ],
  "total": 3,
  "active_count": 2,
  "pending_count": 1
}
```

**Error Responses:**

```json
// 403 - Not a member
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "You are not a member of this organization"
}
```

---

### Update Member Role

Update a member's role. Requires owner or admin role.

```http
PUT /api/organizations/:id/members/:user_id
PATCH /api/organizations/:id/members/:user_id
```

**Path Parameters:**
- `id` (string): Organization ID
- `user_id` (string): User ID of the member

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "role": "admin"
}
```

**Validation Rules:**
- `role`: Required, one of: `owner`, `admin`, `member`, `viewer`

**Success Response (200 OK):**
```json
{
  "success": true,
  "member": {
    "id": "mem_456",
    "organization_id": "org_7x9k2m4n",
    "user_id": "user_abc123",
    "username": "john_member",
    "email": "john@example.com",
    "role": "admin",
    "status": "active",
    "updated_at": "2024-01-15T16:00:00Z"
  },
  "message": "Member role updated successfully"
}
```

**Error Responses:**

```json
// 403 - Insufficient permissions
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only owners and admins can update member roles"
}

// 403 - Admin trying to modify admin/owner
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Admins cannot modify other admins or the owner",
  "details": {
    "target_role": "admin",
    "your_role": "admin"
  }
}

// 409 - Cannot change owner role
{
  "error": true,
  "code": "CONFLICT",
  "message": "Cannot change the owner's role",
  "details": {
    "requirement": "Transfer ownership first"
  }
}

// 409 - Cannot promote to owner
{
  "error": true,
  "code": "CONFLICT",
  "message": "Cannot promote member to owner",
  "details": {
    "requirement": "Use transfer ownership endpoint"
  }
}
```

---

### Remove Member

Remove a member from the organization. Requires owner or admin role.

```http
DELETE /api/organizations/:id/members/:user_id
```

**Path Parameters:**
- `id` (string): Organization ID
- `user_id` (string): User ID of the member to remove

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Member removed from organization",
  "removed_at": "2024-01-15T16:30:00Z"
}
```

**Error Responses:**

```json
// 403 - Insufficient permissions
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only owners and admins can remove members"
}

// 409 - Cannot remove owner
{
  "error": true,
  "code": "CONFLICT",
  "message": "Cannot remove the organization owner",
  "details": {
    "requirement": "Transfer ownership or delete organization"
  }
}

// 403 - Admin trying to remove admin
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Admins cannot remove other admins",
  "details": {
    "requirement": "Only the owner can remove admins"
  }
}

// 409 - Removing self
{
  "error": true,
  "code": "CONFLICT",
  "message": "Use leave organization endpoint to remove yourself"
}
```

---

### Leave Organization

Leave an organization (remove yourself).

```http
POST /api/organizations/:id/leave
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "You have left the organization",
  "left_at": "2024-01-15T17:00:00Z"
}
```

**Error Responses:**

```json
// 409 - Cannot leave as owner
{
  "error": true,
  "code": "CONFLICT",
  "message": "Organization owner cannot leave",
  "details": {
    "requirement": "Transfer ownership or delete organization first"
  }
}
```

---

## Shared Resources API

### Share Connection

Share a database connection with an organization.

```http
POST /api/organizations/:id/connections/:conn_id/share
```

**Path Parameters:**
- `id` (string): Organization ID
- `conn_id` (string): Connection ID

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "permissions": ["read", "execute"],
  "notes": "Production database - read-only access for analysts"
}
```

**Validation Rules:**
- `permissions`: Required, array of one or more: `read`, `execute`, `modify`, `delete`
- `notes`: Optional, max 500 characters

**Permission Definitions:**
- `read`: View connection details (host, database, etc.)
- `execute`: Execute queries against the connection
- `modify`: Update connection settings
- `delete`: Delete the connection

**Success Response (200 OK):**
```json
{
  "success": true,
  "shared_connection": {
    "id": "shconn_abc123",
    "connection_id": "conn_xyz789",
    "connection_name": "Production PostgreSQL",
    "connection_type": "postgres",
    "organization_id": "org_7x9k2m4n",
    "shared_by": "user_abc123",
    "shared_by_username": "john_member",
    "permissions": ["read", "execute"],
    "notes": "Production database - read-only access for analysts",
    "shared_at": "2024-01-15T13:00:00Z"
  },
  "message": "Connection shared successfully"
}
```

**Error Responses:**

```json
// 403 - Not a member
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only organization members can share connections"
}

// 403 - Insufficient role (viewer cannot share)
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Viewers cannot share connections",
  "details": {
    "your_role": "viewer",
    "required_role": "member"
  }
}

// 404 - Connection not found or not owned
{
  "error": true,
  "code": "NOT_FOUND",
  "message": "Connection not found or you don't have permission to share it"
}

// 409 - Already shared
{
  "error": true,
  "code": "CONFLICT",
  "message": "Connection is already shared with this organization",
  "details": {
    "shared_by": "user_xyz789",
    "shared_at": "2024-01-14T10:00:00Z",
    "current_permissions": ["read"]
  }
}

// 400 - Invalid permissions
{
  "error": true,
  "code": "INVALID_REQUEST",
  "message": "Invalid permissions specified",
  "details": {
    "field": "permissions",
    "allowed_values": ["read", "execute", "modify", "delete"]
  }
}
```

---

### List Shared Connections

Get all connections shared with an organization.

```http
GET /api/organizations/:id/connections
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
- `permissions` (optional): Filter by permission (e.g., `?permissions=execute`)
- `shared_by` (optional): Filter by who shared (user ID)

**Success Response (200 OK):**
```json
{
  "connections": [
    {
      "id": "shconn_abc123",
      "connection_id": "conn_xyz789",
      "connection_name": "Production PostgreSQL",
      "connection_type": "postgres",
      "connection_host": "db.example.com",
      "connection_database": "prod_db",
      "organization_id": "org_7x9k2m4n",
      "shared_by": "user_abc123",
      "shared_by_username": "john_member",
      "permissions": ["read", "execute"],
      "notes": "Production database - read-only access",
      "shared_at": "2024-01-15T13:00:00Z",
      "last_used_at": "2024-01-15T15:00:00Z"
    },
    {
      "id": "shconn_def456",
      "connection_id": "conn_uvw123",
      "connection_name": "Analytics MySQL",
      "connection_type": "mysql",
      "connection_host": "analytics.example.com",
      "connection_database": "analytics",
      "organization_id": "org_7x9k2m4n",
      "shared_by": "user_xyz789",
      "shared_by_username": "jane_admin",
      "permissions": ["read", "execute", "modify"],
      "notes": "Analytics database - full access for engineers",
      "shared_at": "2024-01-15T14:00:00Z",
      "last_used_at": null
    }
  ],
  "total": 2
}
```

**Notes:**
- Connection credentials (passwords, SSH keys) are NEVER returned
- Only includes connections the user has permission to see based on their role
- Sorted by shared_at descending

---

### Update Shared Connection Permissions

Update permissions for a shared connection.

```http
PUT /api/organizations/:id/connections/:conn_id/permissions
PATCH /api/organizations/:id/connections/:conn_id/permissions
```

**Path Parameters:**
- `id` (string): Organization ID
- `conn_id` (string): Connection ID

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "permissions": ["read", "execute", "modify"],
  "notes": "Updated: now engineers can modify connection settings"
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "shared_connection": {
    "id": "shconn_abc123",
    "connection_id": "conn_xyz789",
    "organization_id": "org_7x9k2m4n",
    "permissions": ["read", "execute", "modify"],
    "notes": "Updated: now engineers can modify connection settings",
    "updated_at": "2024-01-15T16:00:00Z"
  },
  "message": "Permissions updated successfully"
}
```

**Error Responses:**

```json
// 403 - Only owner of connection or org admin can update
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only the connection owner or organization admins can update permissions"
}
```

---

### Unshare Connection

Remove a connection from organization sharing.

```http
DELETE /api/organizations/:id/connections/:conn_id/share
```

**Path Parameters:**
- `id` (string): Organization ID
- `conn_id` (string): Connection ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Connection unshared from organization",
  "unshared_at": "2024-01-15T17:00:00Z"
}
```

**Error Responses:**

```json
// 403 - Only owner or admin can unshare
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only the connection owner or organization admins can unshare connections"
}

// 404 - Not shared
{
  "error": true,
  "code": "NOT_FOUND",
  "message": "Connection is not shared with this organization"
}
```

---

### Share Query

Share a saved query with an organization.

```http
POST /api/organizations/:id/queries/:query_id/share
```

**Path Parameters:**
- `id` (string): Organization ID
- `query_id` (string): Query ID

**Request Headers:**
```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "permissions": ["read", "execute", "modify"],
  "notes": "Useful analytics query for daily reports"
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "shared_query": {
    "id": "shquery_abc123",
    "query_id": "query_xyz789",
    "query_name": "Daily Active Users",
    "query_description": "Calculate DAU from events table",
    "organization_id": "org_7x9k2m4n",
    "shared_by": "user_abc123",
    "shared_by_username": "john_member",
    "permissions": ["read", "execute", "modify"],
    "notes": "Useful analytics query for daily reports",
    "shared_at": "2024-01-15T14:00:00Z"
  },
  "message": "Query shared successfully"
}
```

**Permission Definitions:**
- `read`: View query text and metadata
- `execute`: Run the query
- `modify`: Edit query text and save changes
- `delete`: Delete the query

**Error Responses:**
(Similar to connection sharing errors)

---

### List Shared Queries

Get all queries shared with an organization.

```http
GET /api/organizations/:id/queries
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
- `permissions` (optional): Filter by permission
- `shared_by` (optional): Filter by who shared
- `tags` (optional): Filter by query tags

**Success Response (200 OK):**
```json
{
  "queries": [
    {
      "id": "shquery_abc123",
      "query_id": "query_xyz789",
      "query_name": "Daily Active Users",
      "query_description": "Calculate DAU from events table",
      "query_tags": ["analytics", "metrics"],
      "organization_id": "org_7x9k2m4n",
      "shared_by": "user_abc123",
      "shared_by_username": "john_member",
      "permissions": ["read", "execute", "modify"],
      "notes": "Useful analytics query for daily reports",
      "shared_at": "2024-01-15T14:00:00Z",
      "last_executed_at": "2024-01-15T16:30:00Z",
      "execution_count": 23
    }
  ],
  "total": 1
}
```

---

### Update Shared Query Permissions

Update permissions for a shared query.

```http
PUT /api/organizations/:id/queries/:query_id/permissions
PATCH /api/organizations/:id/queries/:query_id/permissions
```

(Similar to connection permission updates)

---

### Unshare Query

Remove a query from organization sharing.

```http
DELETE /api/organizations/:id/queries/:query_id/share
```

(Similar to connection unsharing)

---

## Audit Logs API

### Get Audit Logs

Get audit logs for an organization. Requires admin or owner role.

```http
GET /api/organizations/:id/audit
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
- `limit` (integer, optional): Max results (1-100, default: 50)
- `offset` (integer, optional): Pagination offset (default: 0)
- `action` (string, optional): Filter by action type
- `resource_type` (string, optional): Filter by resource type
- `user_id` (string, optional): Filter by user
- `start_time` (ISO8601, optional): Filter by start time
- `end_time` (ISO8601, optional): Filter by end time

**Success Response (200 OK):**
```json
{
  "logs": [
    {
      "id": "audit_123abc",
      "organization_id": "org_7x9k2m4n",
      "user_id": "user_abc123",
      "username": "john_member",
      "email": "john@example.com",
      "action": "connection_shared",
      "resource_type": "connection",
      "resource_id": "conn_xyz789",
      "metadata": {
        "connection_name": "Production PostgreSQL",
        "permissions": ["read", "execute"],
        "notes": "Production database - read-only access"
      },
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0 ...",
      "timestamp": "2024-01-15T13:00:00Z"
    },
    {
      "id": "audit_456def",
      "organization_id": "org_7x9k2m4n",
      "user_id": "user_xyz789",
      "username": "jane_admin",
      "email": "jane@example.com",
      "action": "member_invited",
      "resource_type": "member",
      "resource_id": "inv_ghi789",
      "metadata": {
        "invited_email": "newmember@example.com",
        "role": "member"
      },
      "ip_address": "192.168.1.101",
      "user_agent": "Mozilla/5.0 ...",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 234,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

**Audit Actions:**

| Action | Resource Type | Description |
|--------|---------------|-------------|
| `organization_created` | organization | Organization created |
| `organization_updated` | organization | Organization settings updated |
| `organization_deleted` | organization | Organization deleted |
| `member_invited` | member | Member invitation sent |
| `member_joined` | member | Member accepted invitation |
| `member_role_updated` | member | Member role changed |
| `member_removed` | member | Member removed from org |
| `connection_shared` | connection | Connection shared |
| `connection_unshared` | connection | Connection unshared |
| `connection_permissions_updated` | connection | Shared connection permissions changed |
| `query_shared` | query | Query shared |
| `query_unshared` | query | Query unshared |
| `query_permissions_updated` | query | Shared query permissions changed |
| `query_executed` | query | Shared query executed |

**Error Responses:**

```json
// 403 - Not admin/owner
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only owners and admins can view audit logs"
}
```

---

### Export Audit Logs

Export audit logs as CSV or JSON. Requires owner role.

```http
GET /api/organizations/:id/audit/export
```

**Path Parameters:**
- `id` (string): Organization ID

**Request Headers:**
```http
Authorization: Bearer <token>
```

**Query Parameters:**
- `format` (string, required): `csv` or `json`
- `start_time` (ISO8601, required): Export start time
- `end_time` (ISO8601, required): Export end time
- `action` (string, optional): Filter by action
- `resource_type` (string, optional): Filter by resource type

**Success Response (200 OK):**

**CSV Format:**
```csv
timestamp,user_id,username,action,resource_type,resource_id,ip_address
2024-01-15T13:00:00Z,user_abc123,john_member,connection_shared,connection,conn_xyz789,192.168.1.100
2024-01-15T10:30:00Z,user_xyz789,jane_admin,member_invited,member,inv_ghi789,192.168.1.101
```

**JSON Format:**
```json
{
  "export_info": {
    "organization_id": "org_7x9k2m4n",
    "exported_by": "user_xyz789",
    "exported_at": "2024-01-15T18:00:00Z",
    "start_time": "2024-01-01T00:00:00Z",
    "end_time": "2024-01-15T23:59:59Z",
    "total_logs": 234
  },
  "logs": [
    {
      "timestamp": "2024-01-15T13:00:00Z",
      "user_id": "user_abc123",
      "username": "john_member",
      "action": "connection_shared",
      "resource_type": "connection",
      "resource_id": "conn_xyz789",
      "ip_address": "192.168.1.100"
    }
  ]
}
```

**Error Responses:**

```json
// 403 - Only owner can export
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "Only the organization owner can export audit logs"
}

// 400 - Time range too large
{
  "error": true,
  "code": "INVALID_REQUEST",
  "message": "Time range cannot exceed 90 days for exports",
  "details": {
    "max_days": 90,
    "requested_days": 180
  }
}
```

---

## Data Models

### Organization

```typescript
interface Organization {
  id: string;
  name: string;
  slug: string;
  description?: string;
  owner_id: string;
  plan: 'individual' | 'team' | 'enterprise';
  max_members: number;
  max_connections?: number;
  max_queries_per_month?: number;
  settings: Record<string, any>;
  created_at: string; // ISO8601
  updated_at: string; // ISO8601
  deleted_at?: string; // ISO8601
}
```

### OrganizationMember

```typescript
interface OrganizationMember {
  id: string;
  organization_id: string;
  user_id: string;
  username: string;
  email: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  status: 'pending' | 'active' | 'suspended';
  invited_by?: string;
  invited_at: string; // ISO8601
  invitation_accepted_at?: string; // ISO8601
  invitation_expires_at?: string; // ISO8601
  joined_at?: string; // ISO8601
  last_active?: string; // ISO8601
  created_at: string; // ISO8601
  updated_at: string; // ISO8601
  removed_at?: string; // ISO8601
}
```

### SharedConnection

```typescript
interface SharedConnection {
  id: string;
  connection_id: string;
  connection_name: string;
  connection_type: string; // postgres, mysql, etc.
  connection_host?: string;
  connection_database: string;
  organization_id: string;
  shared_by: string;
  shared_by_username: string;
  permissions: Permission[];
  notes?: string;
  shared_at: string; // ISO8601
  last_used_at?: string; // ISO8601
  unshared_at?: string; // ISO8601
}

type Permission = 'read' | 'execute' | 'modify' | 'delete';
```

### SharedQuery

```typescript
interface SharedQuery {
  id: string;
  query_id: string;
  query_name: string;
  query_description?: string;
  query_tags?: string[];
  organization_id: string;
  shared_by: string;
  shared_by_username: string;
  permissions: Permission[];
  notes?: string;
  shared_at: string; // ISO8601
  last_executed_at?: string; // ISO8601
  execution_count: number;
  unshared_at?: string; // ISO8601
}
```

### AuditLog

```typescript
interface AuditLog {
  id: string;
  organization_id: string;
  user_id?: string;
  username?: string;
  email?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  metadata: Record<string, any>;
  ip_address?: string;
  user_agent?: string;
  timestamp: string; // ISO8601
}
```

---

## OpenAPI 3.0 Specification

For a complete OpenAPI 3.0 specification, see: `/api/openapi.json`

```yaml
openapi: 3.0.0
info:
  title: Howlerops Team Collaboration API
  version: 3.0.0
  description: Team collaboration features for Howlerops
servers:
  - url: https://api.sqlstudio.io
    description: Production
  - url: http://localhost:8500
    description: Development
security:
  - BearerAuth: []
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
# ... (Full spec available at /api/openapi.json)
```

---

## Changelog

### Version 1.0 (2024-01-15)
- Initial Phase 3 API specification
- Organizations CRUD
- Member management
- Resource sharing (connections & queries)
- Audit logging

---

**Document Status**: Ready for Implementation
**Last Updated**: 2024-01-15
**Version**: 1.0
