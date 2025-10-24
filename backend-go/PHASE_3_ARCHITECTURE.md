# Phase 3: Team Collaboration Architecture

**Version:** 1.0
**Status:** Design Phase
**Target:** Production-Ready Implementation

## Overview

Phase 3 introduces team collaboration features to SQL Studio, transforming it from an individual-only tool to a team-enabled platform. This architecture maintains backward compatibility with Phase 1 & 2 (Individual tier) while adding organization-level features for team collaboration.

**Key Features:**
- Organizations with RBAC (Owner, Admin, Member, Viewer)
- Shared database connections across team members
- Shared saved queries and query history
- Audit logging for compliance and security
- Team member invitations and management
- Permission-aware sync protocol

## Architecture Principles

1. **Backward Compatibility**: All existing Individual tier users continue to work unchanged
2. **Default Personal Organization**: Every user gets a personal org for seamless transition
3. **Explicit Sharing**: Resources are private by default, shared explicitly
4. **Security First**: All operations require authorization checks
5. **Scalability**: Designed to support 1000s of organizations with 100s of members each
6. **Data Isolation**: Complete separation between organizations

---

## Table of Contents

1. [Database Schema Design](#database-schema-design)
2. [Data Models (Go Structs)](#data-models-go-structs)
3. [API Design](#api-design)
4. [Permission System](#permission-system)
5. [Sync Protocol Changes](#sync-protocol-changes)
6. [Migration Strategy](#migration-strategy)
7. [Security Considerations](#security-considerations)
8. [Performance Considerations](#performance-considerations)
9. [Implementation Checklist](#implementation-checklist)

---

## Database Schema Design

### Schema Overview

Phase 3 adds organization-specific tables to PostgreSQL (auth database) and extends Turso schema (sync database) to support organization context.

### PostgreSQL Schema (Auth & Organizations)

```sql
-- ============================================================================
-- Organizations Table
-- ============================================================================
-- Central table for team organizations

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Organization Details
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,

    -- Ownership
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

    -- Subscription
    plan VARCHAR(50) NOT NULL DEFAULT 'team', -- team, enterprise
    max_members INTEGER NOT NULL DEFAULT 10,
    max_connections INTEGER,
    max_queries_per_month INTEGER,

    -- Settings (JSON)
    settings JSONB NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CHECK (slug ~* '^[a-z0-9-]+$'),
    CHECK (char_length(slug) >= 3 AND char_length(slug) <= 100),
    CHECK (char_length(name) >= 1 AND char_length(name) <= 255),
    CHECK (plan IN ('team', 'enterprise'))
);

-- Indexes
CREATE INDEX idx_organizations_owner_id ON organizations(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_organizations_plan ON organizations(plan) WHERE deleted_at IS NULL;
CREATE INDEX idx_organizations_created_at ON organizations(created_at DESC);

-- Auto-update timestamp trigger
CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Organization Members Table
-- ============================================================================
-- Maps users to organizations with roles

CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Role (RBAC)
    role VARCHAR(50) NOT NULL DEFAULT 'member',

    -- Invitation Tracking
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    invited_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    invitation_accepted_at TIMESTAMP WITH TIME ZONE,
    invitation_token VARCHAR(255) UNIQUE,
    invitation_expires_at TIMESTAMP WITH TIME ZONE,

    -- Membership Status
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- pending, active, suspended

    -- Timestamps
    joined_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    CHECK (status IN ('pending', 'active', 'suspended')),
    UNIQUE(organization_id, user_id)
);

-- Indexes
CREATE INDEX idx_org_members_org_id ON organization_members(organization_id) WHERE removed_at IS NULL;
CREATE INDEX idx_org_members_user_id ON organization_members(user_id) WHERE removed_at IS NULL;
CREATE INDEX idx_org_members_role ON organization_members(organization_id, role) WHERE removed_at IS NULL;
CREATE INDEX idx_org_members_status ON organization_members(organization_id, status);
CREATE INDEX idx_org_members_invitation_token ON organization_members(invitation_token) WHERE invitation_token IS NOT NULL;

-- Auto-update timestamp trigger
CREATE TRIGGER update_organization_members_updated_at
    BEFORE UPDATE ON organization_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Shared Connections Table
-- ============================================================================
-- Tracks which connections are shared with which organizations

CREATE TABLE shared_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    connection_id VARCHAR(255) NOT NULL, -- References Turso connection_templates.connection_id
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Sharing Details
    shared_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    shared_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Permissions (JSON array of permissions)
    -- ["read", "execute", "modify", "delete"]
    permissions JSONB NOT NULL DEFAULT '["read", "execute"]',

    -- Metadata
    notes TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    unshared_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(connection_id, organization_id)
);

-- Indexes
CREATE INDEX idx_shared_connections_org_id ON shared_connections(organization_id) WHERE unshared_at IS NULL;
CREATE INDEX idx_shared_connections_connection_id ON shared_connections(connection_id) WHERE unshared_at IS NULL;
CREATE INDEX idx_shared_connections_shared_by ON shared_connections(shared_by);

-- ============================================================================
-- Shared Queries Table
-- ============================================================================
-- Tracks which saved queries are shared with organizations

CREATE TABLE shared_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    query_id VARCHAR(255) NOT NULL, -- References Turso saved_queries.id
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Sharing Details
    shared_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    shared_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Permissions (JSON array of permissions)
    -- ["read", "execute", "modify", "delete"]
    permissions JSONB NOT NULL DEFAULT '["read", "execute"]',

    -- Metadata
    notes TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    unshared_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(query_id, organization_id)
);

-- Indexes
CREATE INDEX idx_shared_queries_org_id ON shared_queries(organization_id) WHERE unshared_at IS NULL;
CREATE INDEX idx_shared_queries_query_id ON shared_queries(query_id) WHERE unshared_at IS NULL;
CREATE INDEX idx_shared_queries_shared_by ON shared_queries(shared_by);

-- ============================================================================
-- Audit Logs Table
-- ============================================================================
-- Comprehensive audit trail for organization activities

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Context
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Action Details
    action VARCHAR(100) NOT NULL, -- e.g., member_invited, connection_shared, query_executed
    resource_type VARCHAR(50) NOT NULL, -- organization, member, connection, query
    resource_id VARCHAR(255), -- ID of the affected resource

    -- Additional Data (JSON)
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Request Context
    ip_address INET,
    user_agent TEXT,

    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (action != ''),
    CHECK (resource_type != '')
);

-- Indexes (optimized for time-based queries and filtering)
CREATE INDEX idx_audit_logs_org_id ON audit_logs(organization_id, timestamp DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, timestamp DESC);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);

-- Partition by month for better performance (optional, for large deployments)
-- ALTER TABLE audit_logs PARTITION BY RANGE (timestamp);

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user is member of organization
CREATE OR REPLACE FUNCTION is_organization_member(
    p_user_id UUID,
    p_organization_id UUID
) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM organization_members
        WHERE user_id = p_user_id
        AND organization_id = p_organization_id
        AND status = 'active'
        AND removed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql;

-- Function to check if user has specific role in organization
CREATE OR REPLACE FUNCTION has_organization_role(
    p_user_id UUID,
    p_organization_id UUID,
    p_required_role VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_user_role VARCHAR;
    v_role_hierarchy INTEGER;
    v_required_hierarchy INTEGER;
BEGIN
    -- Get user's role
    SELECT role INTO v_user_role
    FROM organization_members
    WHERE user_id = p_user_id
    AND organization_id = p_organization_id
    AND status = 'active'
    AND removed_at IS NULL;

    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Role hierarchy: owner(4) > admin(3) > member(2) > viewer(1)
    v_user_role_hierarchy := CASE v_user_role
        WHEN 'owner' THEN 4
        WHEN 'admin' THEN 3
        WHEN 'member' THEN 2
        WHEN 'viewer' THEN 1
        ELSE 0
    END;

    v_required_hierarchy := CASE p_required_role
        WHEN 'owner' THEN 4
        WHEN 'admin' THEN 3
        WHEN 'member' THEN 2
        WHEN 'viewer' THEN 1
        ELSE 0
    END;

    RETURN v_user_role_hierarchy >= v_required_hierarchy;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Views
-- ============================================================================

-- Active organization members with user details
CREATE OR REPLACE VIEW v_active_organization_members AS
SELECT
    om.id,
    om.organization_id,
    om.user_id,
    om.role,
    om.status,
    om.joined_at,
    u.username,
    u.email,
    u.created_at as user_created_at
FROM organization_members om
JOIN users u ON om.user_id = u.id
WHERE om.removed_at IS NULL
AND om.status = 'active';

-- Organization statistics
CREATE OR REPLACE VIEW v_organization_stats AS
SELECT
    o.id as organization_id,
    o.name,
    o.slug,
    COUNT(DISTINCT om.user_id) FILTER (WHERE om.status = 'active' AND om.removed_at IS NULL) as active_members,
    COUNT(DISTINCT sc.connection_id) FILTER (WHERE sc.unshared_at IS NULL) as shared_connections,
    COUNT(DISTINCT sq.query_id) FILTER (WHERE sq.unshared_at IS NULL) as shared_queries,
    o.created_at,
    o.updated_at
FROM organizations o
LEFT JOIN organization_members om ON o.id = om.organization_id
LEFT JOIN shared_connections sc ON o.id = sc.organization_id
LEFT JOIN shared_queries sq ON o.id = sq.organization_id
WHERE o.deleted_at IS NULL
GROUP BY o.id, o.name, o.slug, o.created_at, o.updated_at;
```

### Turso Schema Extensions (Sync Database)

```sql
-- ============================================================================
-- Extend existing tables to support organization context
-- ============================================================================

-- Add organization_id to connection_templates
ALTER TABLE connection_templates ADD COLUMN organization_id TEXT;
CREATE INDEX idx_connection_templates_org_id
    ON connection_templates(organization_id) WHERE organization_id IS NOT NULL;

-- Add organization_id to saved_queries
ALTER TABLE saved_queries ADD COLUMN organization_id TEXT;
CREATE INDEX idx_saved_queries_org_id
    ON saved_queries(organization_id) WHERE organization_id IS NOT NULL;

-- Add organization_id to query_history
ALTER TABLE query_history ADD COLUMN organization_id TEXT;
CREATE INDEX idx_query_history_org_id
    ON query_history(organization_id) WHERE organization_id IS NOT NULL;

-- ============================================================================
-- Organization Sync Metadata
-- ============================================================================
-- Tracks last sync time per organization

CREATE TABLE IF NOT EXISTS organization_sync_metadata (
    organization_id TEXT PRIMARY KEY,
    last_sync_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_members INTEGER NOT NULL DEFAULT 0,
    total_shared_connections INTEGER NOT NULL DEFAULT 0,
    total_shared_queries INTEGER NOT NULL DEFAULT 0,
    sync_version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) STRICT;

CREATE INDEX idx_org_sync_metadata_updated
    ON organization_sync_metadata(updated_at DESC);

-- ============================================================================
-- Shared Resource Access Log (for analytics)
-- ============================================================================
-- Tracks who accesses shared resources

CREATE TABLE IF NOT EXISTS shared_resource_access (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    resource_type TEXT NOT NULL, -- connection, query
    resource_id TEXT NOT NULL,
    action TEXT NOT NULL, -- view, execute, modify
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CHECK (resource_type IN ('connection', 'query')),
    CHECK (action IN ('view', 'execute', 'modify'))
) STRICT;

CREATE INDEX idx_shared_resource_access_org
    ON shared_resource_access(organization_id, accessed_at DESC);
CREATE INDEX idx_shared_resource_access_user
    ON shared_resource_access(user_id, accessed_at DESC);
CREATE INDEX idx_shared_resource_access_resource
    ON shared_resource_access(resource_type, resource_id);
```

---

## Data Models (Go Structs)

### Organization Models

```go
package teams

import (
    "time"
    "database/sql"
)

// Organization represents a team organization
type Organization struct {
    ID                 string                 `json:"id"`
    Name               string                 `json:"name"`
    Slug               string                 `json:"slug"`
    Description        string                 `json:"description,omitempty"`
    OwnerID            string                 `json:"owner_id"`
    Plan               string                 `json:"plan"` // team, enterprise
    MaxMembers         int                    `json:"max_members"`
    MaxConnections     *int                   `json:"max_connections,omitempty"`
    MaxQueriesPerMonth *int                   `json:"max_queries_per_month,omitempty"`
    Settings           map[string]interface{} `json:"settings"`
    CreatedAt          time.Time              `json:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at"`
    DeletedAt          *time.Time             `json:"deleted_at,omitempty"`
}

// OrganizationRole represents a user's role in an organization
type OrganizationRole string

const (
    RoleOwner  OrganizationRole = "owner"
    RoleAdmin  OrganizationRole = "admin"
    RoleMember OrganizationRole = "member"
    RoleViewer OrganizationRole = "viewer"
)

// MemberStatus represents the status of an organization member
type MemberStatus string

const (
    MemberStatusPending   MemberStatus = "pending"
    MemberStatusActive    MemberStatus = "active"
    MemberStatusSuspended MemberStatus = "suspended"
)

// OrganizationMember represents a member of an organization
type OrganizationMember struct {
    ID                     string          `json:"id"`
    OrganizationID         string          `json:"organization_id"`
    UserID                 string          `json:"user_id"`
    Role                   OrganizationRole `json:"role"`
    InvitedBy              *string         `json:"invited_by,omitempty"`
    InvitedAt              time.Time       `json:"invited_at"`
    InvitationAcceptedAt   *time.Time      `json:"invitation_accepted_at,omitempty"`
    InvitationToken        *string         `json:"invitation_token,omitempty"`
    InvitationExpiresAt    *time.Time      `json:"invitation_expires_at,omitempty"`
    Status                 MemberStatus    `json:"status"`
    JoinedAt               *time.Time      `json:"joined_at,omitempty"`
    CreatedAt              time.Time       `json:"created_at"`
    UpdatedAt              time.Time       `json:"updated_at"`
    RemovedAt              *time.Time      `json:"removed_at,omitempty"`

    // Populated via JOIN
    Username               string          `json:"username,omitempty"`
    Email                  string          `json:"email,omitempty"`
}

// SharedConnection represents a connection shared with an organization
type SharedConnection struct {
    ID             string    `json:"id"`
    ConnectionID   string    `json:"connection_id"`
    OrganizationID string    `json:"organization_id"`
    SharedBy       string    `json:"shared_by"`
    SharedAt       time.Time `json:"shared_at"`
    Permissions    []string  `json:"permissions"` // read, execute, modify, delete
    Notes          string    `json:"notes,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
    UnsharedAt     *time.Time `json:"unshared_at,omitempty"`
}

// SharedQuery represents a saved query shared with an organization
type SharedQuery struct {
    ID             string    `json:"id"`
    QueryID        string    `json:"query_id"`
    OrganizationID string    `json:"organization_id"`
    SharedBy       string    `json:"shared_by"`
    SharedAt       time.Time `json:"shared_at"`
    Permissions    []string  `json:"permissions"` // read, execute, modify, delete
    Notes          string    `json:"notes,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
    UnsharedAt     *time.Time `json:"unshared_at,omitempty"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
    ID             string                 `json:"id"`
    OrganizationID string                 `json:"organization_id"`
    UserID         *string                `json:"user_id,omitempty"`
    Action         string                 `json:"action"`
    ResourceType   string                 `json:"resource_type"`
    ResourceID     string                 `json:"resource_id,omitempty"`
    Metadata       map[string]interface{} `json:"metadata"`
    IPAddress      string                 `json:"ip_address,omitempty"`
    UserAgent      string                 `json:"user_agent,omitempty"`
    Timestamp      time.Time              `json:"timestamp"`
}

// Permission represents a granular permission
type Permission string

const (
    PermissionRead   Permission = "read"
    PermissionExecute Permission = "execute"
    PermissionModify  Permission = "modify"
    PermissionDelete  Permission = "delete"
)

// OrganizationStats represents organization statistics
type OrganizationStats struct {
    OrganizationID      string    `json:"organization_id"`
    Name                string    `json:"name"`
    Slug                string    `json:"slug"`
    ActiveMembers       int       `json:"active_members"`
    SharedConnections   int       `json:"shared_connections"`
    SharedQueries       int       `json:"shared_queries"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}
```

### Request/Response Models

```go
package teams

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
    Name        string                 `json:"name" validate:"required,min=1,max=255"`
    Slug        string                 `json:"slug" validate:"required,min=3,max=100,alphanum"`
    Description string                 `json:"description,omitempty"`
    Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
    Name        *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
    Description *string                `json:"description,omitempty"`
    Settings    map[string]interface{} `json:"settings,omitempty"`
}

// InviteMemberRequest represents a request to invite a member
type InviteMemberRequest struct {
    Email string           `json:"email" validate:"required,email"`
    Role  OrganizationRole `json:"role" validate:"required,oneof=admin member viewer"`
}

// UpdateMemberRoleRequest represents a request to update a member's role
type UpdateMemberRoleRequest struct {
    Role OrganizationRole `json:"role" validate:"required,oneof=owner admin member viewer"`
}

// ShareConnectionRequest represents a request to share a connection
type ShareConnectionRequest struct {
    ConnectionID string       `json:"connection_id" validate:"required"`
    Permissions  []Permission `json:"permissions" validate:"required,min=1,dive,oneof=read execute modify delete"`
    Notes        string       `json:"notes,omitempty"`
}

// ShareQueryRequest represents a request to share a query
type ShareQueryRequest struct {
    QueryID     string       `json:"query_id" validate:"required"`
    Permissions []Permission `json:"permissions" validate:"required,min=1,dive,oneof=read execute modify delete"`
    Notes       string       `json:"notes,omitempty"`
}

// ListAuditLogsRequest represents a request to list audit logs
type ListAuditLogsRequest struct {
    Limit        int       `json:"limit" validate:"min=1,max=100"`
    Offset       int       `json:"offset" validate:"min=0"`
    Action       string    `json:"action,omitempty"`
    ResourceType string    `json:"resource_type,omitempty"`
    UserID       string    `json:"user_id,omitempty"`
    StartTime    time.Time `json:"start_time,omitempty"`
    EndTime      time.Time `json:"end_time,omitempty"`
}

// ListAuditLogsResponse represents a response containing audit logs
type ListAuditLogsResponse struct {
    Logs       []AuditLog `json:"logs"`
    Total      int        `json:"total"`
    Limit      int        `json:"limit"`
    Offset     int        `json:"offset"`
    HasMore    bool       `json:"has_more"`
}
```

---

## API Design

### Organization Endpoints

#### Create Organization
```
POST /api/organizations
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp",
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  }
}

Response (201 Created):
{
  "id": "org_123abc",
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp",
  "owner_id": "user_456def",
  "plan": "team",
  "max_members": 10,
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  },
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}

Errors:
- 400 Bad Request: Invalid slug format, name too long, slug already taken
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User already has max organizations for their plan
```

#### List User's Organizations
```
GET /api/organizations
Authorization: Bearer <token>

Response (200 OK):
{
  "organizations": [
    {
      "id": "org_personal",
      "name": "John Doe",
      "slug": "john-doe-personal",
      "owner_id": "user_456def",
      "role": "owner",
      "plan": "individual",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "org_123abc",
      "name": "Acme Corp Engineering",
      "slug": "acme-engineering",
      "owner_id": "user_789ghi",
      "role": "member",
      "plan": "team",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 2
}
```

#### Get Organization Details
```
GET /api/organizations/:id
Authorization: Bearer <token>

Response (200 OK):
{
  "id": "org_123abc",
  "name": "Acme Corp Engineering",
  "slug": "acme-engineering",
  "description": "Engineering team at Acme Corp",
  "owner_id": "user_789ghi",
  "plan": "team",
  "max_members": 10,
  "settings": {
    "allow_public_sharing": false,
    "require_2fa": true
  },
  "stats": {
    "active_members": 5,
    "shared_connections": 12,
    "shared_queries": 34
  },
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not a member of this organization
- 404 Not Found: Organization does not exist
```

#### Update Organization
```
PUT /api/organizations/:id
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "name": "Acme Corp Engineering Team",
  "description": "Updated description",
  "settings": {
    "allow_public_sharing": true
  }
}

Response (200 OK):
{
  "id": "org_123abc",
  "name": "Acme Corp Engineering Team",
  "description": "Updated description",
  ...
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not owner/admin
- 404 Not Found: Organization does not exist
```

#### Delete Organization
```
DELETE /api/organizations/:id
Authorization: Bearer <token>

Response (200 OK):
{
  "success": true,
  "message": "Organization deleted successfully"
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not owner
- 404 Not Found: Organization does not exist
- 409 Conflict: Cannot delete organization with active members (must remove all members first)
```

### Member Management Endpoints

#### Invite Member
```
POST /api/organizations/:id/invite
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "email": "newmember@example.com",
  "role": "member"
}

Response (200 OK):
{
  "success": true,
  "invitation": {
    "id": "inv_123abc",
    "email": "newmember@example.com",
    "role": "member",
    "invited_by": "user_456def",
    "invited_at": "2024-01-15T10:30:00Z",
    "expires_at": "2024-01-22T10:30:00Z",
    "status": "pending"
  },
  "message": "Invitation sent to newmember@example.com"
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User does not have permission to invite members
- 409 Conflict: User already a member, or max members reached
- 400 Bad Request: Invalid email or role
```

#### List Members
```
GET /api/organizations/:id/members
Authorization: Bearer <token>

Response (200 OK):
{
  "members": [
    {
      "id": "mem_123",
      "user_id": "user_789ghi",
      "username": "jane_admin",
      "email": "jane@example.com",
      "role": "owner",
      "status": "active",
      "joined_at": "2024-01-15T10:00:00Z"
    },
    {
      "id": "mem_456",
      "user_id": "user_456def",
      "username": "john_member",
      "email": "john@example.com",
      "role": "member",
      "status": "active",
      "joined_at": "2024-01-15T11:00:00Z"
    }
  ],
  "total": 2
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not a member
- 404 Not Found: Organization does not exist
```

#### Update Member Role
```
PUT /api/organizations/:id/members/:user_id
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "role": "admin"
}

Response (200 OK):
{
  "success": true,
  "member": {
    "id": "mem_456",
    "user_id": "user_456def",
    "role": "admin",
    "updated_at": "2024-01-15T12:00:00Z"
  }
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User does not have permission to update roles
- 404 Not Found: Organization or member does not exist
- 400 Bad Request: Cannot change owner role, or invalid role
```

#### Remove Member
```
DELETE /api/organizations/:id/members/:user_id
Authorization: Bearer <token>

Response (200 OK):
{
  "success": true,
  "message": "Member removed from organization"
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User does not have permission to remove members
- 404 Not Found: Organization or member does not exist
- 409 Conflict: Cannot remove owner (must transfer ownership first)
```

### Shared Resource Endpoints

#### Share Connection
```
POST /api/organizations/:id/connections/:conn_id/share
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "permissions": ["read", "execute"],
  "notes": "Production database - read-only access"
}

Response (200 OK):
{
  "success": true,
  "shared_connection": {
    "id": "shconn_123",
    "connection_id": "conn_abc",
    "organization_id": "org_123abc",
    "shared_by": "user_456def",
    "permissions": ["read", "execute"],
    "notes": "Production database - read-only access",
    "shared_at": "2024-01-15T13:00:00Z"
  }
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User does not have permission to share connections
- 404 Not Found: Organization or connection does not exist
- 409 Conflict: Connection already shared with this organization
```

#### List Shared Connections
```
GET /api/organizations/:id/connections
Authorization: Bearer <token>

Response (200 OK):
{
  "connections": [
    {
      "id": "shconn_123",
      "connection_id": "conn_abc",
      "connection_name": "Production DB",
      "connection_type": "postgres",
      "shared_by": "user_456def",
      "shared_by_username": "john_member",
      "permissions": ["read", "execute"],
      "notes": "Production database - read-only access",
      "shared_at": "2024-01-15T13:00:00Z"
    }
  ],
  "total": 1
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not a member
- 404 Not Found: Organization does not exist
```

#### Unshare Connection
```
DELETE /api/organizations/:id/connections/:conn_id/share
Authorization: Bearer <token>

Response (200 OK):
{
  "success": true,
  "message": "Connection unshared from organization"
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User does not have permission to unshare
- 404 Not Found: Shared connection does not exist
```

#### Share Query
```
POST /api/organizations/:id/queries/:query_id/share
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "permissions": ["read", "execute", "modify"],
  "notes": "Useful analytics query"
}

Response (200 OK):
{
  "success": true,
  "shared_query": {
    "id": "shquery_456",
    "query_id": "query_xyz",
    "organization_id": "org_123abc",
    "shared_by": "user_456def",
    "permissions": ["read", "execute", "modify"],
    "notes": "Useful analytics query",
    "shared_at": "2024-01-15T14:00:00Z"
  }
}
```

#### List Shared Queries
```
GET /api/organizations/:id/queries
Authorization: Bearer <token>

Response (200 OK):
{
  "queries": [
    {
      "id": "shquery_456",
      "query_id": "query_xyz",
      "query_name": "Daily Active Users",
      "shared_by": "user_456def",
      "shared_by_username": "john_member",
      "permissions": ["read", "execute", "modify"],
      "notes": "Useful analytics query",
      "shared_at": "2024-01-15T14:00:00Z"
    }
  ],
  "total": 1
}
```

#### Unshare Query
```
DELETE /api/organizations/:id/queries/:query_id/share
Authorization: Bearer <token>

Response (200 OK):
{
  "success": true,
  "message": "Query unshared from organization"
}
```

### Audit Log Endpoints

#### Get Audit Logs
```
GET /api/organizations/:id/audit
Authorization: Bearer <token>
Query Parameters:
  - limit (default: 50, max: 100)
  - offset (default: 0)
  - action (optional filter)
  - resource_type (optional filter)
  - user_id (optional filter)
  - start_time (optional filter, ISO8601)
  - end_time (optional filter, ISO8601)

Response (200 OK):
{
  "logs": [
    {
      "id": "audit_123",
      "organization_id": "org_123abc",
      "user_id": "user_456def",
      "username": "john_member",
      "action": "connection_shared",
      "resource_type": "connection",
      "resource_id": "conn_abc",
      "metadata": {
        "connection_name": "Production DB",
        "permissions": ["read", "execute"]
      },
      "ip_address": "192.168.1.100",
      "timestamp": "2024-01-15T13:00:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0,
  "has_more": false
}

Errors:
- 401 Unauthorized: Missing/invalid token
- 403 Forbidden: User is not admin/owner
- 404 Not Found: Organization does not exist
```

---

## Permission System

### Role-Based Access Control (RBAC) Matrix

| Action | Owner | Admin | Member | Viewer |
|--------|-------|-------|--------|--------|
| **Organization Management** |
| View organization details | ✓ | ✓ | ✓ | ✓ |
| Update organization settings | ✓ | ✓ | ✗ | ✗ |
| Delete organization | ✓ | ✗ | ✗ | ✗ |
| View audit logs | ✓ | ✓ | ✗ | ✗ |
| **Member Management** |
| Invite members | ✓ | ✓ | ✗ | ✗ |
| View members list | ✓ | ✓ | ✓ | ✓ |
| Update member roles | ✓ | ✓* | ✗ | ✗ |
| Remove members | ✓ | ✓* | ✗ | ✗ |
| **Connection Sharing** |
| Share own connections | ✓ | ✓ | ✓ | ✗ |
| View shared connections | ✓ | ✓ | ✓ | ✓ |
| Use shared connections | ✓ | ✓ | ✓** | ✓** |
| Modify shared connections | ✓ | ✓ | ✗*** | ✗ |
| Unshare connections | ✓ | ✓ | Own only | ✗ |
| **Query Sharing** |
| Share own queries | ✓ | ✓ | ✓ | ✗ |
| View shared queries | ✓ | ✓ | ✓ | ✓ |
| Execute shared queries | ✓ | ✓ | ✓** | ✓** |
| Modify shared queries | ✓ | ✓ | ✗*** | ✗ |
| Unshare queries | ✓ | ✓ | Own only | ✗ |

*Admins cannot modify/remove Owners or other Admins
**Based on resource-level permissions
***Only if granted "modify" permission on specific resource

### Permission Check Implementation

```go
package teams

import (
    "context"
    "errors"
)

var (
    ErrNotMember        = errors.New("user is not a member of this organization")
    ErrInsufficientRole = errors.New("user does not have required role")
    ErrInsufficientPerm = errors.New("user does not have required permission")
)

// PermissionChecker handles authorization checks
type PermissionChecker struct {
    store OrganizationStore
}

// CheckMembership verifies user is a member of organization
func (pc *PermissionChecker) CheckMembership(ctx context.Context, userID, orgID string) error {
    member, err := pc.store.GetMember(ctx, orgID, userID)
    if err != nil {
        return err
    }
    if member == nil || member.Status != MemberStatusActive {
        return ErrNotMember
    }
    return nil
}

// CheckRole verifies user has at least the required role
func (pc *PermissionChecker) CheckRole(ctx context.Context, userID, orgID string, requiredRole OrganizationRole) error {
    member, err := pc.store.GetMember(ctx, orgID, userID)
    if err != nil {
        return err
    }
    if member == nil || member.Status != MemberStatusActive {
        return ErrNotMember
    }

    if !hasRole(member.Role, requiredRole) {
        return ErrInsufficientRole
    }

    return nil
}

// CheckResourcePermission verifies user has specific permission on a resource
func (pc *PermissionChecker) CheckResourcePermission(
    ctx context.Context,
    userID, orgID, resourceID string,
    resourceType string,
    requiredPerm Permission,
) error {
    // First check membership
    if err := pc.CheckMembership(ctx, userID, orgID); err != nil {
        return err
    }

    // Check if resource is shared with org
    var permissions []Permission
    var err error

    switch resourceType {
    case "connection":
        sc, err := pc.store.GetSharedConnection(ctx, orgID, resourceID)
        if err != nil || sc == nil {
            return ErrInsufficientPerm
        }
        permissions = sc.Permissions
    case "query":
        sq, err := pc.store.GetSharedQuery(ctx, orgID, resourceID)
        if err != nil || sq == nil {
            return ErrInsufficientPerm
        }
        permissions = sq.Permissions
    default:
        return errors.New("invalid resource type")
    }

    // Check if user has required permission
    for _, perm := range permissions {
        if perm == requiredPerm {
            return nil
        }
    }

    return ErrInsufficientPerm
}

// hasRole checks if user's role meets or exceeds required role
func hasRole(userRole, requiredRole OrganizationRole) bool {
    hierarchy := map[OrganizationRole]int{
        RoleViewer: 1,
        RoleMember: 2,
        RoleAdmin:  3,
        RoleOwner:  4,
    }

    return hierarchy[userRole] >= hierarchy[requiredRole]
}
```

### Middleware for Authorization

```go
package middleware

import (
    "net/http"
    "github.com/gorilla/mux"
    "github.com/sql-studio/backend-go/internal/teams"
)

// RequireOrgMembership middleware verifies user is a member
func RequireOrgMembership(checker *teams.PermissionChecker) mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Context().Value("user_id").(string)
            orgID := mux.Vars(r)["id"]

            if err := checker.CheckMembership(r.Context(), userID, orgID); err != nil {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// RequireOrgRole middleware verifies user has required role
func RequireOrgRole(checker *teams.PermissionChecker, role teams.OrganizationRole) mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Context().Value("user_id").(string)
            orgID := mux.Vars(r)["id"]

            if err := checker.CheckRole(r.Context(), userID, orgID, role); err != nil {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Sync Protocol Changes

### Organization-Scoped Sync

Phase 3 introduces organization context to the sync protocol. Users can sync:
1. **Personal resources** (user_id scoped, organization_id = NULL)
2. **Organization resources** (organization_id scoped)

### Sync Request Format Changes

```typescript
// Updated SyncUploadRequest
interface SyncUploadRequest {
  device_id: string;
  last_sync_at: string; // ISO8601
  organization_id?: string; // NEW: Optional organization context
  changes: SyncChange[];
}

// Updated SyncChange
interface SyncChange {
  id: string;
  item_type: 'connection' | 'saved_query' | 'query_history';
  item_id: string;
  action: 'create' | 'update' | 'delete';
  data: ConnectionTemplate | SavedQuery | QueryHistory;
  updated_at: string;
  sync_version: number;
  device_id: string;
  organization_id?: string; // NEW: Organization context
}

// Updated SyncDownloadRequest
interface SyncDownloadRequest {
  since: string; // ISO8601
  device_id: string;
  organization_id?: string; // NEW: Optional organization filter
}

// Updated SyncDownloadResponse
interface SyncDownloadResponse {
  connections: ConnectionTemplate[];
  saved_queries: SavedQuery[];
  query_history: QueryHistory[];
  shared_connections?: SharedConnectionInfo[]; // NEW
  shared_queries?: SharedQueryInfo[]; // NEW
  sync_timestamp: string;
  has_more: boolean;
}

// NEW: Shared resource metadata
interface SharedConnectionInfo {
  connection_id: string;
  organization_id: string;
  organization_name: string;
  permissions: Permission[];
  shared_by: string;
  shared_at: string;
}

interface SharedQueryInfo {
  query_id: string;
  organization_id: string;
  organization_name: string;
  permissions: Permission[];
  shared_by: string;
  shared_at: string;
}
```

### Permission-Aware Sync Logic

```go
// Sync service modifications for organization support

func (s *Service) Download(ctx context.Context, req *SyncDownloadRequest) (*SyncDownloadResponse, error) {
    userID := getUserIDFromContext(ctx)

    resp := &SyncDownloadResponse{}

    // Download personal resources (always)
    personalConns, err := s.store.GetConnectionsSince(ctx, userID, nil, req.Since)
    if err != nil {
        return nil, err
    }
    resp.Connections = append(resp.Connections, personalConns...)

    // Download organization resources (if org specified)
    if req.OrganizationID != nil {
        // Check membership
        if err := s.permChecker.CheckMembership(ctx, userID, *req.OrganizationID); err != nil {
            return nil, err
        }

        // Get org connections
        orgConns, err := s.store.GetConnectionsSince(ctx, userID, req.OrganizationID, req.Since)
        if err != nil {
            return nil, err
        }
        resp.Connections = append(resp.Connections, orgConns...)

        // Get shared connections
        sharedConns, err := s.store.GetSharedConnectionsForOrg(ctx, *req.OrganizationID)
        if err != nil {
            return nil, err
        }
        resp.SharedConnections = sharedConns

        // Get shared queries
        sharedQueries, err := s.store.GetSharedQueriesForOrg(ctx, *req.OrganizationID)
        if err != nil {
            return nil, err
        }
        resp.SharedQueries = sharedQueries
    }

    // Similar logic for queries and history...

    resp.SyncTimestamp = time.Now().Format(time.RFC3339)
    return resp, nil
}

func (s *Service) Upload(ctx context.Context, req *SyncUploadRequest) (*SyncUploadResponse, error) {
    userID := getUserIDFromContext(ctx)

    // If organization context provided, verify membership and permissions
    if req.OrganizationID != nil {
        if err := s.permChecker.CheckMembership(ctx, userID, *req.OrganizationID); err != nil {
            return nil, err
        }
    }

    resp := &SyncUploadResponse{
        Success: true,
        Conflicts: []Conflict{},
        Rejected: []RejectedChange{},
    }

    for _, change := range req.Changes {
        // Validate organization context matches
        if change.OrganizationID != req.OrganizationID {
            resp.Rejected = append(resp.Rejected, RejectedChange{
                Change: change,
                Reason: "organization_id mismatch",
            })
            continue
        }

        // Check permissions for org resources
        if change.OrganizationID != nil {
            // Check if user can modify this resource
            if change.Action == SyncActionUpdate || change.Action == SyncActionDelete {
                // Verify ownership or modify permission
                hasPermission, err := s.checkModifyPermission(ctx, userID, change)
                if err != nil || !hasPermission {
                    resp.Rejected = append(resp.Rejected, RejectedChange{
                        Change: change,
                        Reason: "insufficient_permissions",
                    })
                    continue
                }
            }
        }

        // Process change (existing logic)
        conflict, err := s.processChange(ctx, userID, change)
        if err != nil {
            return nil, err
        }
        if conflict != nil {
            resp.Conflicts = append(resp.Conflicts, *conflict)
        }
    }

    resp.SyncedAt = time.Now().Format(time.RFC3339)
    return resp, nil
}
```

### Conflict Resolution for Shared Resources

Shared resources have special conflict handling:

1. **Owner always wins**: If the resource owner makes a change, it takes precedence
2. **Permission-based**: Users with "modify" permission can change shared resources
3. **Version tracking**: Each change increments sync_version
4. **Audit trail**: All changes to shared resources are logged

```go
func (s *Service) resolveSharedResourceConflict(
    ctx context.Context,
    userID string,
    local, remote *SyncChange,
) (*ConflictResolution, error) {
    // Check if user is the resource owner
    isOwner, err := s.isResourceOwner(ctx, userID, local.ItemID)
    if err != nil {
        return nil, err
    }

    if isOwner {
        // Owner always wins
        return &ConflictResolution{
            Strategy: "owner_wins",
            Winner: local,
        }, nil
    }

    // Check permissions
    hasModifyPerm, err := s.hasModifyPermission(ctx, userID, local.ItemID)
    if err != nil {
        return nil, err
    }

    if !hasModifyPerm {
        // User doesn't have modify permission, remote wins
        return &ConflictResolution{
            Strategy: "remote_wins",
            Winner: remote,
        }, nil
    }

    // Both have modify permission, use last write wins
    if local.UpdatedAt.After(remote.UpdatedAt) {
        return &ConflictResolution{
            Strategy: "last_write_wins",
            Winner: local,
        }, nil
    }

    return &ConflictResolution{
        Strategy: "last_write_wins",
        Winner: remote,
    }, nil
}
```

---

## Migration Strategy

### Phase 1: Database Migration

```sql
-- Migration: Add Phase 3 tables to PostgreSQL
-- Version: 3.0.0
-- Applied: 2024-XX-XX

BEGIN;

-- Create all Phase 3 tables (organizations, organization_members, etc.)
-- See "Database Schema Design" section above for full SQL

-- Create default "personal" organizations for existing users
INSERT INTO organizations (id, name, slug, owner_id, plan, max_members, created_at, updated_at)
SELECT
    'org_personal_' || id,
    COALESCE(username, email),
    'personal-' || LOWER(REGEXP_REPLACE(COALESCE(username, email), '[^a-zA-Z0-9]', '-', 'g')),
    id,
    'individual',
    1,
    created_at,
    CURRENT_TIMESTAMP
FROM users
WHERE NOT EXISTS (
    SELECT 1 FROM organizations WHERE owner_id = users.id AND plan = 'individual'
);

-- Add user as owner of their personal organization
INSERT INTO organization_members (organization_id, user_id, role, status, joined_at, created_at)
SELECT
    'org_personal_' || u.id,
    u.id,
    'owner',
    'active',
    u.created_at,
    CURRENT_TIMESTAMP
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM organization_members
    WHERE user_id = u.id AND organization_id = 'org_personal_' || u.id
);

COMMIT;
```

### Phase 2: Turso Schema Migration

```sql
-- Migration: Add organization support to Turso
-- Version: 3.0.0

BEGIN;

-- Add organization_id columns (already shown in schema section)
ALTER TABLE connection_templates ADD COLUMN organization_id TEXT;
ALTER TABLE saved_queries ADD COLUMN organization_id TEXT;
ALTER TABLE query_history ADD COLUMN organization_id TEXT;

-- Create indexes
CREATE INDEX idx_connection_templates_org_id
    ON connection_templates(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_saved_queries_org_id
    ON saved_queries(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_query_history_org_id
    ON query_history(organization_id) WHERE organization_id IS NOT NULL;

-- Create new tables for org metadata
CREATE TABLE organization_sync_metadata (...);
CREATE TABLE shared_resource_access (...);

COMMIT;
```

### Phase 3: Data Migration Script

```go
package migrations

import (
    "context"
    "database/sql"
    "fmt"
)

// MigrateToPhase3 migrates existing users to have personal organizations
func MigrateToPhase3(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get all users without a personal organization
    rows, err := tx.QueryContext(ctx, `
        SELECT id, username, email, created_at
        FROM users
        WHERE NOT EXISTS (
            SELECT 1 FROM organizations
            WHERE owner_id = users.id AND plan = 'individual'
        )
    `)
    if err != nil {
        return fmt.Errorf("query users: %w", err)
    }
    defer rows.Close()

    migrated := 0
    for rows.Next() {
        var userID, username, email string
        var createdAt time.Time

        if err := rows.Scan(&userID, &username, &email, &createdAt); err != nil {
            return fmt.Errorf("scan user: %w", err)
        }

        // Create personal organization
        orgID := "org_personal_" + userID
        orgName := username
        if orgName == "" {
            orgName = email
        }
        orgSlug := "personal-" + sanitizeSlug(orgName)

        _, err = tx.ExecContext(ctx, `
            INSERT INTO organizations (id, name, slug, owner_id, plan, max_members)
            VALUES ($1, $2, $3, $4, 'individual', 1)
        `, orgID, orgName, orgSlug, userID)
        if err != nil {
            return fmt.Errorf("create org for user %s: %w", userID, err)
        }

        // Add user as owner
        _, err = tx.ExecContext(ctx, `
            INSERT INTO organization_members (organization_id, user_id, role, status, joined_at)
            VALUES ($1, $2, 'owner', 'active', $3)
        `, orgID, userID, createdAt)
        if err != nil {
            return fmt.Errorf("add member for user %s: %w", userID, err)
        }

        migrated++
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }

    fmt.Printf("Migrated %d users to Phase 3\n", migrated)
    return nil
}

func sanitizeSlug(s string) string {
    // Convert to lowercase and replace non-alphanumeric with dash
    // Implementation omitted for brevity
    return strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(s, "-"))
}
```

### Phase 4: Backward Compatibility

```go
// Ensure existing Individual tier API endpoints continue to work

// Legacy endpoint: POST /api/sync/upload
// Now internally scopes to user's personal organization
func (h *Handler) HandleLegacySyncUpload(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)

    // Get user's personal organization
    personalOrg, err := h.orgStore.GetPersonalOrganization(r.Context(), userID)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Parse request
    var req SyncUploadRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Add organization context (transparent to client)
    req.OrganizationID = &personalOrg.ID

    // Process sync (existing logic)
    resp, err := h.syncService.Upload(r.Context(), &req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(resp)
}
```

---

## Security Considerations

### 1. Authorization Checks

Every organization endpoint MUST verify:
1. User is authenticated (JWT token valid)
2. User is a member of the organization
3. User has required role for the operation
4. User has required resource-level permissions (for shared resources)

```go
// Example authorization check in handler
func (h *Handler) HandleShareConnection(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := ctx.Value("user_id").(string)
    orgID := mux.Vars(r)["id"]
    connID := mux.Vars(r)["conn_id"]

    // 1. Check membership
    if err := h.permChecker.CheckMembership(ctx, userID, orgID); err != nil {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // 2. Check role (members can share their own connections)
    if err := h.permChecker.CheckRole(ctx, userID, orgID, teams.RoleMember); err != nil {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // 3. Verify user owns the connection
    conn, err := h.syncStore.GetConnection(ctx, userID, connID)
    if err != nil || conn == nil {
        http.Error(w, "Connection not found", http.StatusNotFound)
        return
    }

    // Proceed with sharing...
}
```

### 2. Audit Logging

ALL organization operations must be logged:

```go
func (s *AuditService) LogAction(ctx context.Context, log *AuditLog) error {
    // Enrich with request context
    if req, ok := ctx.Value("request").(*http.Request); ok {
        log.IPAddress = getClientIP(req)
        log.UserAgent = req.UserAgent()
    }

    // Write to database
    return s.store.CreateAuditLog(ctx, log)
}

// Usage in handlers
func (h *Handler) HandleInviteMember(w http.ResponseWriter, r *http.Request) {
    // ... invitation logic ...

    // Log the action
    h.auditService.LogAction(r.Context(), &AuditLog{
        OrganizationID: orgID,
        UserID: &userID,
        Action: "member_invited",
        ResourceType: "member",
        ResourceID: invitationID,
        Metadata: map[string]interface{}{
            "invited_email": req.Email,
            "role": req.Role,
        },
    })
}
```

### 3. Rate Limiting (Per Organization)

```go
// Organization-scoped rate limiter
type OrgRateLimiter struct {
    limiter *redis.Client
}

func (rl *OrgRateLimiter) CheckLimit(ctx context.Context, orgID, action string) error {
    key := fmt.Sprintf("ratelimit:org:%s:%s", orgID, action)

    // Different limits per action
    limits := map[string]int{
        "invite_member": 10,  // 10 invites per hour
        "share_connection": 50, // 50 shares per hour
        "audit_query": 100, // 100 audit queries per hour
    }

    limit := limits[action]
    if limit == 0 {
        limit = 100 // default
    }

    // Sliding window rate limit
    count, err := rl.limiter.Incr(ctx, key).Result()
    if err != nil {
        return err
    }

    if count == 1 {
        rl.limiter.Expire(ctx, key, time.Hour)
    }

    if count > int64(limit) {
        return errors.New("rate limit exceeded")
    }

    return nil
}
```

### 4. Data Isolation

All queries MUST include organization_id filter:

```go
// CORRECT: Scoped query
func (s *Store) GetSharedConnections(ctx context.Context, orgID string) ([]*SharedConnection, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, connection_id, organization_id, shared_by, permissions, shared_at
        FROM shared_connections
        WHERE organization_id = $1
        AND unshared_at IS NULL
    `, orgID)
    // ...
}

// INCORRECT: Missing org filter (security vulnerability!)
func (s *Store) GetSharedConnections(ctx context.Context) ([]*SharedConnection, error) {
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, connection_id, organization_id, shared_by, permissions, shared_at
        FROM shared_connections
        WHERE unshared_at IS NULL
    `)
    // This would leak data across organizations!
}
```

### 5. Access Token Scoping

JWTs should include organization context:

```go
type JWTClaims struct {
    UserID         string   `json:"user_id"`
    Email          string   `json:"email"`
    Organizations  []string `json:"organizations"` // NEW: List of org IDs user is member of
    jwt.StandardClaims
}

// When user switches organization in UI, issue new token with org context
func (s *AuthService) IssueOrgToken(ctx context.Context, userID, orgID string) (string, error) {
    // Verify membership
    if !s.isMember(ctx, userID, orgID) {
        return "", errors.New("not a member")
    }

    claims := &JWTClaims{
        UserID: userID,
        Organizations: []string{orgID},
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}
```

---

## Performance Considerations

### 1. Database Indexes

All critical queries are indexed:

```sql
-- Organization membership checks (most frequent query)
CREATE INDEX idx_org_members_user_org ON organization_members(user_id, organization_id)
    WHERE removed_at IS NULL AND status = 'active';

-- Shared resource lookups
CREATE INDEX idx_shared_connections_org_unshared ON shared_connections(organization_id, unshared_at);
CREATE INDEX idx_shared_queries_org_unshared ON shared_queries(organization_id, unshared_at);

-- Audit log queries (time-range scans)
CREATE INDEX idx_audit_logs_org_time ON audit_logs(organization_id, timestamp DESC);

-- Composite index for filtered audit queries
CREATE INDEX idx_audit_logs_filtered ON audit_logs(organization_id, action, resource_type, timestamp DESC);
```

### 2. Caching Strategy

```go
type OrgCache struct {
    redis *redis.Client
}

// Cache membership checks (5 minute TTL)
func (c *OrgCache) GetMembership(ctx context.Context, userID, orgID string) (*OrganizationMember, error) {
    key := fmt.Sprintf("membership:%s:%s", userID, orgID)

    // Try cache first
    var member OrganizationMember
    err := c.redis.Get(ctx, key).Scan(&member)
    if err == nil {
        return &member, nil
    }

    // Cache miss, query database
    member, err = c.store.GetMember(ctx, orgID, userID)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    c.redis.Set(ctx, key, member, 5*time.Minute)

    return &member, nil
}

// Invalidate cache on membership changes
func (c *OrgCache) InvalidateMembership(ctx context.Context, userID, orgID string) {
    key := fmt.Sprintf("membership:%s:%s", userID, orgID)
    c.redis.Del(ctx, key)
}

// Cache organization details (longer TTL)
func (c *OrgCache) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
    key := fmt.Sprintf("org:%s", orgID)

    var org Organization
    err := c.redis.Get(ctx, key).Scan(&org)
    if err == nil {
        return &org, nil
    }

    org, err = c.store.GetOrganization(ctx, orgID)
    if err != nil {
        return nil, err
    }

    // Cache for 15 minutes
    c.redis.Set(ctx, key, org, 15*time.Minute)

    return &org, nil
}
```

### 3. Batch Operations

```go
// Batch load organization members to avoid N+1 queries
func (s *Store) GetOrganizationsWithMembers(ctx context.Context, userID string) ([]*Organization, error) {
    // Single query with JOIN
    rows, err := s.db.QueryContext(ctx, `
        SELECT
            o.id, o.name, o.slug, o.plan,
            om.role, om.status,
            COUNT(om2.id) as member_count
        FROM organizations o
        JOIN organization_members om ON o.id = om.organization_id
        LEFT JOIN organization_members om2 ON o.id = om2.organization_id
            AND om2.status = 'active' AND om2.removed_at IS NULL
        WHERE om.user_id = $1
        AND om.status = 'active'
        AND om.removed_at IS NULL
        AND o.deleted_at IS NULL
        GROUP BY o.id, o.name, o.slug, o.plan, om.role, om.status
    `, userID)

    // Parse into structs...
}
```

### 4. Pagination for Audit Logs

```go
func (s *Store) GetAuditLogs(ctx context.Context, req *ListAuditLogsRequest) (*ListAuditLogsResponse, error) {
    // Count total (with filters)
    countQuery := `SELECT COUNT(*) FROM audit_logs WHERE organization_id = $1`
    args := []interface{}{req.OrganizationID}

    var total int
    err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
    if err != nil {
        return nil, err
    }

    // Get page of results
    query := `
        SELECT id, user_id, action, resource_type, resource_id, metadata, timestamp
        FROM audit_logs
        WHERE organization_id = $1
        ORDER BY timestamp DESC
        LIMIT $2 OFFSET $3
    `

    rows, err := s.db.QueryContext(ctx, query, req.OrganizationID, req.Limit, req.Offset)
    // Parse logs...

    return &ListAuditLogsResponse{
        Logs: logs,
        Total: total,
        Limit: req.Limit,
        Offset: req.Offset,
        HasMore: req.Offset + req.Limit < total,
    }, nil
}
```

### 5. Connection Pooling

```go
// Separate connection pools for read-heavy vs write-heavy operations

type DatabasePools struct {
    ReadPool  *sql.DB
    WritePool *sql.DB
}

func NewDatabasePools(connString string) (*DatabasePools, error) {
    // Read pool (larger, more connections)
    readPool, err := sql.Open("postgres", connString)
    if err != nil {
        return nil, err
    }
    readPool.SetMaxOpenConns(50)
    readPool.SetMaxIdleConns(25)

    // Write pool (smaller, fewer connections)
    writePool, err := sql.Open("postgres", connString)
    if err != nil {
        return nil, err
    }
    writePool.SetMaxOpenConns(20)
    writePool.SetMaxIdleConns(10)

    return &DatabasePools{
        ReadPool: readPool,
        WritePool: writePool,
    }, nil
}

// Use appropriate pool
func (s *Store) GetOrganization(ctx context.Context, id string) (*Organization, error) {
    // Use read pool for queries
    row := s.pools.ReadPool.QueryRowContext(ctx, `SELECT ... FROM organizations WHERE id = $1`, id)
    // ...
}

func (s *Store) CreateOrganization(ctx context.Context, org *Organization) error {
    // Use write pool for mutations
    _, err := s.pools.WritePool.ExecContext(ctx, `INSERT INTO organizations ...`)
    return err
}
```

---

## Implementation Checklist

### Database & Schema
- [ ] Create PostgreSQL migration for Phase 3 tables
- [ ] Create Turso migration for organization support
- [ ] Add indexes for performance
- [ ] Create database helper functions
- [ ] Create database views for common queries
- [ ] Write migration script for existing users

### Backend Services
- [ ] Create `teams` package with organization models
- [ ] Implement `OrganizationStore` interface
- [ ] Implement `OrganizationService` with business logic
- [ ] Create `PermissionChecker` for RBAC
- [ ] Extend `SyncService` for organization-scoped sync
- [ ] Create `AuditService` for logging
- [ ] Implement email templates for invitations

### API Endpoints
- [ ] Implement organization CRUD endpoints
- [ ] Implement member management endpoints
- [ ] Implement shared resource endpoints
- [ ] Implement audit log endpoints
- [ ] Add authorization middleware
- [ ] Add rate limiting middleware
- [ ] Write OpenAPI/Swagger documentation

### Sync Protocol
- [ ] Update sync types to support organization context
- [ ] Implement permission-aware sync download
- [ ] Implement permission-aware sync upload
- [ ] Add conflict resolution for shared resources
- [ ] Update sync metadata tracking

### Security
- [ ] Add authorization checks to all endpoints
- [ ] Implement audit logging for all operations
- [ ] Add rate limiting per organization
- [ ] Implement data isolation checks
- [ ] Add organization context to JWT tokens
- [ ] Security audit of all queries

### Testing
- [ ] Unit tests for organization service
- [ ] Unit tests for permission checker
- [ ] Integration tests for organization endpoints
- [ ] Integration tests for member management
- [ ] Integration tests for sharing workflows
- [ ] Integration tests for sync with organizations
- [ ] Load tests for 1000+ organizations
- [ ] Security penetration testing

### Frontend Integration
- [ ] Organization selection UI
- [ ] Organization settings page
- [ ] Member management UI
- [ ] Invitation acceptance flow
- [ ] Shared resources list
- [ ] Permission indicators in UI
- [ ] Audit log viewer

### Documentation
- [ ] API documentation (OpenAPI spec)
- [ ] Sync protocol documentation
- [ ] Migration guide for existing users
- [ ] Admin guide for organization management
- [ ] Security best practices guide

### Deployment
- [ ] Database migration scripts
- [ ] Rollback plan
- [ ] Monitoring and alerting setup
- [ ] Performance benchmarks
- [ ] Gradual rollout plan

---

## Next Steps

1. **Review and Approve**: Review this architecture with the team
2. **Create Issues**: Break down into implementable tasks
3. **API Spec**: Create detailed OpenAPI spec (PHASE_3_API_SPEC.md)
4. **Sync Protocol**: Document sync protocol changes (PHASE_3_SYNC_PROTOCOL.md)
5. **Prototype**: Build proof-of-concept for critical paths
6. **Implement**: Execute implementation in phases
7. **Test**: Comprehensive testing before production
8. **Deploy**: Gradual rollout with monitoring

---

## Timeline Estimate

**Assuming 2 full-time engineers:**

- **Week 1-2**: Database schema, migrations, basic services
- **Week 3-4**: Organization CRUD, member management
- **Week 5-6**: Shared resources, permissions
- **Week 7-8**: Sync protocol updates, conflict resolution
- **Week 9-10**: Audit logging, security hardening
- **Week 11-12**: Testing, documentation, deployment prep
- **Week 13-14**: Gradual rollout, monitoring, bug fixes

**Total: 14 weeks (3.5 months)**

---

**Document Status**: Ready for Review
**Last Updated**: 2024-01-15
**Version**: 1.0
