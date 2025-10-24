# Phase 3: Team Collaboration - Detailed Task Breakdown

## Document Overview

This document provides a comprehensive breakdown of all tasks required for Phase 3 implementation. Each task includes effort estimates, dependencies, assignee roles, acceptance criteria, and priority levels.

**Phase Duration:** 6 weeks (January 16 - February 27, 2026)
**Total Estimated Hours:** 536 hours
**Total Person-Weeks:** 13.4

---

## Task Categorization

### Priority Levels
- **P0 (Must-Have):** Critical for Phase 3 completion, blocks other work
- **P1 (Must-Have):** Required for Phase 3 completion
- **P2 (Should-Have):** Important but can be deferred if needed
- **P3 (Could-Have):** Nice to have, defer to Phase 4 if time-constrained

### Effort Estimation
- **XS:** 1-2 hours
- **S:** 3-4 hours
- **M:** 5-8 hours
- **L:** 9-16 hours
- **XL:** 17-24 hours
- **XXL:** 25+ hours

---

## Sprint 1: Foundation (Week 13)

### Backend Tasks

#### TASK-001: Database Schema - Organizations Table
**Priority:** P0
**Effort:** M (8 hours)
**Assignee:** Backend Engineer
**Dependencies:** None

**Description:**
Create the core `organizations` table with all necessary fields for team management.

**Subtasks:**
1. Write SQL migration script
2. Add indexes for performance
3. Add constraints and foreign keys
4. Test migration locally
5. Document schema design

**Acceptance Criteria:**
- [ ] Organizations table created with all fields
- [ ] Indexes created: id (PK), owner_id, name, deleted_at
- [ ] Migration script runs without errors
- [ ] Can create, read, update, delete organizations via SQL
- [ ] Soft delete (deleted_at) working correctly

**SQL Schema:**
```sql
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    owner_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    max_members INTEGER DEFAULT 10,
    settings TEXT, -- JSON

    CONSTRAINT name_not_empty CHECK (LENGTH(name) > 0)
);

CREATE INDEX idx_organizations_owner ON organizations(owner_id, deleted_at);
CREATE INDEX idx_organizations_name ON organizations(name) WHERE deleted_at IS NULL;
```

---

#### TASK-002: Database Schema - Organization Members Table
**Priority:** P0
**Effort:** M (6 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-001

**Description:**
Create `organization_members` table to track team membership and roles.

**Subtasks:**
1. Write SQL migration script
2. Define role enum (owner, admin, member)
3. Add composite indexes
4. Add foreign key constraints
5. Test membership operations

**Acceptance Criteria:**
- [ ] organization_members table created
- [ ] Unique constraint on (organization_id, user_id)
- [ ] Indexes on organization_id, user_id, role
- [ ] Can add/remove members via SQL
- [ ] Role values validated (owner/admin/member only)

**SQL Schema:**
```sql
CREATE TABLE organization_members (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    invited_by TEXT,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_role CHECK (role IN ('owner', 'admin', 'member')),
    UNIQUE(organization_id, user_id)
);

CREATE INDEX idx_org_members_org ON organization_members(organization_id);
CREATE INDEX idx_org_members_user ON organization_members(user_id);
```

---

#### TASK-003: Database Schema - Organization Invitations Table
**Priority:** P1
**Effort:** S (6 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-001

**Description:**
Create `organization_invitations` table for managing team invitation flow.

**Acceptance Criteria:**
- [ ] Invitations table created with token field
- [ ] Unique constraint on (organization_id, email)
- [ ] Expires_at field for invitation expiration
- [ ] Token field indexed and unique
- [ ] Can track invitation lifecycle

**SQL Schema:**
```sql
CREATE TABLE organization_invitations (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    invited_by TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_invite_role CHECK (role IN ('admin', 'member')),
    UNIQUE(organization_id, email)
);

CREATE INDEX idx_invitations_org ON organization_invitations(organization_id);
CREATE INDEX idx_invitations_email ON organization_invitations(email);
CREATE INDEX idx_invitations_token ON organization_invitations(token);
```

---

#### TASK-004: Organization Service - Repository Layer
**Priority:** P0
**Effort:** L (12 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-001, TASK-002, TASK-003

**Description:**
Implement repository pattern for organizations with full CRUD operations.

**Subtasks:**
1. Create OrganizationRepository interface
2. Implement Turso-based repository
3. Add error handling and validation
4. Implement soft delete
5. Add transaction support
6. Write unit tests

**Acceptance Criteria:**
- [ ] Repository interface defined
- [ ] CRUD methods implemented (Create, Read, Update, Delete)
- [ ] Soft delete (deleted_at) implemented
- [ ] Validation errors returned correctly
- [ ] Unit tests cover all methods (>90% coverage)
- [ ] Can handle concurrent operations safely

**Interface:**
```go
type OrganizationRepository interface {
    Create(ctx context.Context, org *Organization) error
    GetByID(ctx context.Context, id string) (*Organization, error)
    GetByUserID(ctx context.Context, userID string) ([]*Organization, error)
    Update(ctx context.Context, org *Organization) error
    Delete(ctx context.Context, id string) error
    AddMember(ctx context.Context, orgID, userID, role string) error
    RemoveMember(ctx context.Context, orgID, userID string) error
    GetMembers(ctx context.Context, orgID string) ([]*OrganizationMember, error)
}
```

---

#### TASK-005: Organization Service - Business Logic Layer
**Priority:** P0
**Effort:** L (10 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-004

**Description:**
Implement business logic for organization operations with validation and authorization.

**Subtasks:**
1. Create OrganizationService with business rules
2. Implement organization name validation
3. Add duplicate name checking
4. Implement member limit enforcement
5. Add owner validation
6. Write service tests

**Acceptance Criteria:**
- [ ] Service layer validates all inputs
- [ ] Organization names must be unique and non-empty
- [ ] Member limits enforced (max 10 for initial tier)
- [ ] Owner cannot be removed from organization
- [ ] Cannot delete organization with active members (unless forced)
- [ ] Service tests cover all business rules (>85% coverage)

---

#### TASK-006: Organization API - Create Endpoint
**Priority:** P0
**Effort:** M (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-005

**Description:**
Implement POST /api/organizations endpoint for creating new organizations.

**Acceptance Criteria:**
- [ ] POST /api/organizations endpoint implemented
- [ ] Requires authentication
- [ ] Validates organization name
- [ ] Creator automatically added as owner
- [ ] Returns 201 Created with organization object
- [ ] Returns 400 Bad Request for invalid input
- [ ] Returns 409 Conflict for duplicate name

**Request/Response:**
```json
POST /api/organizations
Authorization: Bearer <jwt>

{
  "name": "Acme Corp Engineering",
  "description": "Engineering team workspace"
}

Response 201:
{
  "id": "org-uuid",
  "name": "Acme Corp Engineering",
  "description": "Engineering team workspace",
  "owner_id": "user-uuid",
  "created_at": "2026-01-16T10:00:00Z",
  "member_count": 1
}
```

---

#### TASK-007: Organization API - List & Get Endpoints
**Priority:** P0
**Effort:** M (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-005

**Description:**
Implement GET /api/organizations and GET /api/organizations/:id endpoints.

**Acceptance Criteria:**
- [ ] GET /api/organizations lists user's organizations
- [ ] GET /api/organizations/:id returns org details
- [ ] Requires authentication
- [ ] Only returns organizations user is a member of
- [ ] Returns 404 Not Found for invalid org ID
- [ ] Returns 403 Forbidden if user not a member

---

#### TASK-008: Organization API - Update & Delete Endpoints
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-005

**Description:**
Implement PUT /api/organizations/:id and DELETE /api/organizations/:id endpoints.

**Acceptance Criteria:**
- [ ] PUT /api/organizations/:id updates organization
- [ ] DELETE /api/organizations/:id soft-deletes organization
- [ ] Requires owner or admin role (checked later in RBAC sprint)
- [ ] Update validates name uniqueness
- [ ] Delete checks for active members
- [ ] Returns appropriate error codes

---

#### TASK-009: Member Management - Add/Remove Member
**Priority:** P1
**Effort:** M (8 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-004

**Description:**
Implement member addition and removal logic.

**Acceptance Criteria:**
- [ ] Can add member to organization
- [ ] Can remove member from organization
- [ ] Cannot remove owner
- [ ] Cannot add duplicate members
- [ ] Member count enforced (max 10)
- [ ] Removal triggers cleanup (remove from shared resources)

---

#### TASK-010: Member Management - List Members
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-004

**Description:**
Implement GET /api/organizations/:id/members endpoint.

**Acceptance Criteria:**
- [ ] Returns all organization members
- [ ] Includes user details (name, email)
- [ ] Includes role and join date
- [ ] Supports pagination (limit/offset)
- [ ] Returns 403 if user not a member

---

### Frontend Tasks

#### TASK-011: TypeScript Types - Organization Models
**Priority:** P0
**Effort:** S (4 hours)
**Assignee:** Frontend Engineer
**Dependencies:** None

**Description:**
Define TypeScript interfaces and types for organizations, members, and invitations.

**Acceptance Criteria:**
- [ ] Organization interface defined
- [ ] OrganizationMember interface defined
- [ ] OrganizationInvitation interface defined
- [ ] OrganizationRole enum defined
- [ ] All types exported from central types file
- [ ] Zod schemas for validation

**Types:**
```typescript
export enum OrganizationRole {
  OWNER = 'owner',
  ADMIN = 'admin',
  MEMBER = 'member',
}

export interface Organization {
  id: string
  name: string
  description?: string
  owner_id: string
  created_at: string
  updated_at: string
  member_count: number
  max_members: number
  settings?: Record<string, any>
}

export interface OrganizationMember {
  id: string
  organization_id: string
  user_id: string
  role: OrganizationRole
  invited_by?: string
  joined_at: string
  user: {
    id: string
    email: string
    display_name?: string
  }
}

export interface OrganizationInvitation {
  id: string
  organization_id: string
  email: string
  role: OrganizationRole
  invited_by: string
  token: string
  expires_at: string
  accepted_at?: string
  created_at: string
  organization: Organization
}
```

---

#### TASK-012: Organization Store - Zustand State Management
**Priority:** P0
**Effort:** M (8 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-011

**Description:**
Create Zustand store for organization state management with sync integration.

**Acceptance Criteria:**
- [ ] Organization store created with Zustand
- [ ] State includes: organizations, currentOrg, loading, error
- [ ] Actions: createOrg, updateOrg, deleteOrg, switchOrg
- [ ] Integrates with IndexedDB for persistence
- [ ] Integrates with cloud sync
- [ ] Optimistic updates implemented

**Store Interface:**
```typescript
interface OrganizationStore {
  // State
  organizations: Organization[]
  currentOrgId: string | null
  loading: boolean
  error: string | null

  // Actions
  createOrganization: (data: CreateOrgInput) => Promise<Organization>
  updateOrganization: (id: string, data: UpdateOrgInput) => Promise<void>
  deleteOrganization: (id: string) => Promise<void>
  switchOrganization: (id: string | null) => void
  fetchOrganizations: () => Promise<void>

  // Computed
  currentOrg: () => Organization | null
}
```

---

#### TASK-013: Organization UI - List View
**Priority:** P1
**Effort:** M (8 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Create organization list page showing all user's organizations.

**Acceptance Criteria:**
- [ ] Displays all user's organizations in grid/list
- [ ] Shows organization name, description, member count
- [ ] Shows user's role in each organization
- [ ] Includes "Create Organization" button
- [ ] Clicking org navigates to org detail page
- [ ] Empty state for no organizations
- [ ] Loading and error states

---

#### TASK-014: Organization UI - Creation Modal
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Create modal/form for creating new organizations.

**Acceptance Criteria:**
- [ ] Modal with organization name input (required)
- [ ] Optional description textarea
- [ ] Validation: name 3-50 chars, no special chars
- [ ] Submit creates organization and closes modal
- [ ] Shows loading state during creation
- [ ] Shows error messages
- [ ] Success redirects to new org page

---

#### TASK-015: Organization UI - Detail/Settings Page
**Priority:** P1
**Effort:** M (8 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Create organization detail page with settings.

**Acceptance Criteria:**
- [ ] Displays organization details (name, description, created date)
- [ ] Edit name/description (owner/admin only)
- [ ] Shows member count
- [ ] Link to members page
- [ ] Delete organization button (owner only)
- [ ] Confirmation dialog for deletion
- [ ] Breadcrumb navigation

---

#### TASK-016: Organization UI - Switcher Component
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Create organization switcher component in top navigation.

**Acceptance Criteria:**
- [ ] Dropdown showing current organization
- [ ] List of all user's organizations
- [ ] Personal workspace option (no org)
- [ ] Clicking switches context immediately
- [ ] Shows indicator of current selection
- [ ] Accessible keyboard navigation

---

### Testing Tasks

#### TASK-017: Backend Unit Tests - Organization Service
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** QA Engineer / Backend Engineer
**Dependencies:** TASK-005

**Description:**
Write comprehensive unit tests for organization service layer.

**Acceptance Criteria:**
- [ ] Test organization creation with valid data
- [ ] Test validation errors (empty name, too long, etc.)
- [ ] Test duplicate name prevention
- [ ] Test member limit enforcement
- [ ] Test soft delete
- [ ] >85% code coverage

---

#### TASK-018: Backend Unit Tests - Organization Repository
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** QA Engineer / Backend Engineer
**Dependencies:** TASK-004

**Description:**
Write unit tests for organization repository CRUD operations.

**Acceptance Criteria:**
- [ ] Test create, read, update, delete operations
- [ ] Test member add/remove operations
- [ ] Test error handling (db errors, not found, etc.)
- [ ] Test transaction rollback on error
- [ ] >90% code coverage

---

#### TASK-019: Frontend Unit Tests - Organization Store
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Write unit tests for organization Zustand store.

**Acceptance Criteria:**
- [ ] Test organization creation action
- [ ] Test organization switching
- [ ] Test optimistic updates
- [ ] Test error handling
- [ ] >80% code coverage

---

#### TASK-020: Frontend Component Tests - Organization UI
**Priority:** P2
**Effort:** S (4 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-013, TASK-014, TASK-015

**Description:**
Write component tests for organization UI components.

**Acceptance Criteria:**
- [ ] Test organization list renders correctly
- [ ] Test creation modal opens/closes
- [ ] Test form validation
- [ ] Test empty states
- [ ] Test error states

---

### Cross-Cutting Tasks

#### TASK-021: Sprint 1 Planning & Kickoff
**Priority:** P0
**Effort:** S (4 hours)
**Assignee:** Product Manager, Team Leads
**Dependencies:** None

**Description:**
Plan Sprint 1, assign tasks, and conduct kickoff meeting.

**Acceptance Criteria:**
- [ ] All Sprint 1 tasks assigned
- [ ] Dependencies identified
- [ ] Kickoff meeting conducted
- [ ] Team has access to all resources
- [ ] Development environment set up

---

**Sprint 1 Total:** 150 hours (3.75 person-weeks)

---

## Sprint 2: Invitations & Onboarding (Week 14)

### Backend Tasks

#### TASK-022: Invitation Token Generation
**Priority:** P0
**Effort:** S (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-003

**Description:**
Implement secure token generation and validation for invitations.

**Acceptance Criteria:**
- [ ] Generates cryptographically secure tokens (32 bytes)
- [ ] Tokens are URL-safe
- [ ] Token validation function implemented
- [ ] Tokens expire after 7 days
- [ ] One-time use enforced

---

#### TASK-023: Invitation API - Create Invitation
**Priority:** P0
**Effort:** M (8 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-022

**Description:**
Implement POST /api/organizations/:id/invitations endpoint.

**Acceptance Criteria:**
- [ ] Validates email format
- [ ] Checks inviter has permission (owner/admin)
- [ ] Prevents duplicate invitations
- [ ] Generates unique token
- [ ] Sets expiration date (7 days)
- [ ] Returns invitation object

---

#### TASK-024: Invitation API - List, Accept, Decline, Revoke
**Priority:** P0
**Effort:** L (10 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-023

**Description:**
Implement remaining invitation endpoints.

**Acceptance Criteria:**
- [ ] GET /api/invitations - lists user's pending invitations
- [ ] POST /api/invitations/:id/accept - accepts invitation
- [ ] POST /api/invitations/:id/decline - declines invitation
- [ ] DELETE /api/organizations/:id/invitations/:id - revokes invitation
- [ ] Accept adds user as member
- [ ] All endpoints validate permissions

---

#### TASK-025: Email Service - Invitation Templates
**Priority:** P0
**Effort:** M (6 hours)
**Assignee:** Backend Engineer
**Dependencies:** None

**Description:**
Create HTML email templates for invitations.

**Acceptance Criteria:**
- [ ] Invitation email template (HTML + text)
- [ ] Welcome to team email template
- [ ] Member removed notification template
- [ ] Templates branded with SQL Studio style
- [ ] Templates tested in multiple email clients

---

#### TASK-026: Email Service - Send Invitation Emails
**Priority:** P0
**Effort:** M (8 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-025

**Description:**
Implement email sending for invitations using Resend API.

**Acceptance Criteria:**
- [ ] Sends invitation email when invitation created
- [ ] Includes magic link with token
- [ ] Includes organization details
- [ ] Handles email sending errors gracefully
- [ ] Logs email sending attempts

---

#### TASK-027: Invitation Rate Limiting
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-023

**Description:**
Implement rate limiting for invitation creation (prevent spam).

**Acceptance Criteria:**
- [ ] Max 20 invitations per hour per user
- [ ] Max 5 invitations per hour per organization
- [ ] Returns 429 Too Many Requests when exceeded
- [ ] Rate limit resets after time window

---

### Frontend Tasks

#### TASK-028: Invitation UI - Team Members Page
**Priority:** P0
**Effort:** L (10 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-012

**Description:**
Create team members page showing current members and pending invitations.

**Acceptance Criteria:**
- [ ] Table showing current members (name, email, role, joined date)
- [ ] Section for pending invitations
- [ ] Search/filter members
- [ ] "Invite Member" button opens modal
- [ ] Can revoke pending invitations
- [ ] Can remove members (with confirmation)

---

#### TASK-029: Invitation UI - Invite Modal
**Priority:** P0
**Effort:** M (8 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-028

**Description:**
Create modal for inviting new team members.

**Acceptance Criteria:**
- [ ] Email input with validation
- [ ] Role selector (Admin or Member)
- [ ] Bulk invite (multiple emails, comma-separated)
- [ ] Shows pending invitations count
- [ ] Submit sends invitations
- [ ] Success message with copy link option
- [ ] Error handling

---

#### TASK-030: Invitation UI - Accept/Decline Flow
**Priority:** P0
**Effort:** L (10 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-011

**Description:**
Create UI flow for accepting or declining invitations.

**Acceptance Criteria:**
- [ ] Invitation notification banner when logged in
- [ ] Invitations list page (/invitations)
- [ ] Shows organization details before accepting
- [ ] Accept button adds to organization
- [ ] Decline button removes invitation
- [ ] Success redirect to new organization
- [ ] Expired invitation handling

---

#### TASK-031: Invitation UI - Magic Link Landing Page
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-030

**Description:**
Create landing page for invitation magic links.

**Acceptance Criteria:**
- [ ] /invite/:token route created
- [ ] Validates token on load
- [ ] Shows organization preview
- [ ] Accept/Decline buttons
- [ ] Handles expired tokens
- [ ] Redirects to signup if not authenticated
- [ ] Redirects to org after acceptance

---

#### TASK-032: Member Management UI - Role Change
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-028

**Description:**
Add UI for changing member roles.

**Acceptance Criteria:**
- [ ] Role dropdown for each member
- [ ] Only visible to owner/admin
- [ ] Cannot change owner role
- [ ] Confirmation dialog for role changes
- [ ] Optimistic update with rollback on error

---

#### TASK-033: Member Management UI - Remove Member
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** Frontend Engineer
**Dependencies:** TASK-028

**Description:**
Add UI for removing team members.

**Acceptance Criteria:**
- [ ] Remove button for each member
- [ ] Only visible to owner/admin
- [ ] Cannot remove owner
- [ ] Confirmation dialog with warning
- [ ] Shows impact (removed from shared resources)
- [ ] Optimistic update

---

### Testing Tasks

#### TASK-034: Integration Tests - Invitation Flow
**Priority:** P1
**Effort:** M (8 hours)
**Assignee:** QA Engineer
**Dependencies:** TASK-024, TASK-026

**Description:**
Test complete invitation flow from creation to acceptance.

**Acceptance Criteria:**
- [ ] Test invitation creation (happy path)
- [ ] Test invitation acceptance
- [ ] Test invitation decline
- [ ] Test invitation expiration
- [ ] Test invitation revocation
- [ ] Test duplicate invitation prevention
- [ ] Test email sending (mocked)

---

#### TASK-035: E2E Tests - Member Onboarding
**Priority:** P1
**Effort:** M (6 hours)
**Assignee:** QA Engineer
**Dependencies:** TASK-030, TASK-031

**Description:**
End-to-end Playwright tests for member onboarding.

**Acceptance Criteria:**
- [ ] Test user receives invitation
- [ ] Test user accepts invitation
- [ ] Test user sees new organization
- [ ] Test user can switch to new organization
- [ ] Test user has correct role

---

#### TASK-036: Unit Tests - Email Service
**Priority:** P2
**Effort:** S (4 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-026

**Description:**
Unit tests for email sending service.

**Acceptance Criteria:**
- [ ] Test email template rendering
- [ ] Test email sending (mocked Resend API)
- [ ] Test error handling
- [ ] >85% coverage

---

### Design Tasks

#### TASK-037: Email Template Design
**Priority:** P1
**Effort:** S (4 hours)
**Assignee:** UI/UX Designer
**Dependencies:** None

**Description:**
Design email templates for invitation flow.

**Acceptance Criteria:**
- [ ] Invitation email designed (HTML mockup)
- [ ] Welcome email designed
- [ ] Templates match SQL Studio brand
- [ ] Templates are mobile-responsive
- [ ] Design approved by stakeholders

---

### Documentation Tasks

#### TASK-038: API Documentation - Invitations
**Priority:** P2
**Effort:** S (2 hours)
**Assignee:** Backend Engineer
**Dependencies:** TASK-024

**Description:**
Document invitation API endpoints.

**Acceptance Criteria:**
- [ ] OpenAPI spec for invitation endpoints
- [ ] Request/response examples
- [ ] Error codes documented

---

#### TASK-039: User Guide - Team Invitations
**Priority:** P2
**Effort:** XS (2 hours)
**Assignee:** Product Manager
**Dependencies:** TASK-030

**Description:**
Write user guide for inviting team members.

**Acceptance Criteria:**
- [ ] Step-by-step guide with screenshots
- [ ] FAQ section
- [ ] Published to help docs

---

**Sprint 2 Total:** 100 hours (2.5 person-weeks)

---

## Sprint 3: RBAC Implementation (Weeks 15-16)

**Note:** Sprint 3 spans 2 weeks with 150 hours total. See PHASE_3_IMPLEMENTATION_PLAN.md for detailed task breakdown.

### Week 15: Backend Permission System (75 hours)

#### TASK-040: Permission Matrix Design
**Priority:** P0
**Effort:** M (8 hours)

#### TASK-041: Permission Middleware Implementation
**Priority:** P0
**Effort:** L (12 hours)

#### TASK-042: Update Organization Endpoints with Permissions
**Priority:** P0
**Effort:** L (10 hours)

#### TASK-043: Shared Resource Schema Updates
**Priority:** P0
**Effort:** L (12 hours)

#### TASK-044: Audit Logging System
**Priority:** P0
**Effort:** L (12 hours)

#### TASK-045: Permission Utilities & Helpers
**Priority:** P1
**Effort:** M (6 hours)

#### TASK-046: Permission Caching (Redis)
**Priority:** P2
**Effort:** M (8 hours)

#### TASK-047: Permission Tests - Backend
**Priority:** P1
**Effort:** L (12 hours)

### Week 16: Frontend Permission UI (75 hours)

#### TASK-048: usePermissions Hook
**Priority:** P0
**Effort:** M (6 hours)

#### TASK-049: Permission-Based UI Rendering
**Priority:** P0
**Effort:** L (12 hours)

#### TASK-050: Role Management UI
**Priority:** P1
**Effort:** M (8 hours)

#### TASK-051: Organization Settings UI
**Priority:** P1
**Effort:** L (10 hours)

#### TASK-052: Transfer Ownership Flow
**Priority:** P1
**Effort:** M (6 hours)

#### TASK-053: Audit Log Viewer UI
**Priority:** P1
**Effort:** M (8 hours)

#### TASK-054: Permission Error Handling
**Priority:** P1
**Effort:** M (6 hours)

#### TASK-055: Security Audit (External)
**Priority:** P0
**Effort:** XL (16 hours)

#### TASK-056: E2E Permission Tests
**Priority:** P1
**Effort:** L (10 hours)

#### TASK-057: Documentation - Permissions
**Priority:** P1
**Effort:** M (8 hours)

---

## Sprint 4: Shared Resources (Week 17)

### Backend Tasks (58 hours)

#### TASK-058: Shared Connections - Schema Migration
**Priority:** P0
**Effort:** M (6 hours)

**Description:**
Update connections table to support organization ownership and visibility.

**Acceptance Criteria:**
- [ ] Add organization_id column (nullable, foreign key)
- [ ] Add visibility column (personal, shared)
- [ ] Add created_by column
- [ ] Migration script for existing connections
- [ ] Indexes on organization_id

---

#### TASK-059: Shared Connections - API Endpoints
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Implement API endpoints for shared connections.

**Acceptance Criteria:**
- [ ] POST /api/organizations/:id/connections
- [ ] GET /api/organizations/:id/connections
- [ ] PUT /api/connections/:id/share (change visibility)
- [ ] Permission checks: members can view, edit own
- [ ] Validation: no credentials synced

---

#### TASK-060: Shared Queries - Schema Migration
**Priority:** P0
**Effort:** M (6 hours)

**Description:**
Update saved_queries table to support organization ownership.

**Acceptance Criteria:**
- [ ] Add organization_id column
- [ ] Add visibility column
- [ ] Add created_by column
- [ ] Migration script
- [ ] Indexes added

---

#### TASK-061: Shared Queries - API Endpoints
**Priority:** P0
**Effort:** L (10 hours)

**Description:**
Implement API endpoints for shared queries.

**Acceptance Criteria:**
- [ ] POST /api/organizations/:id/queries
- [ ] GET /api/organizations/:id/queries
- [ ] PUT /api/queries/:id/share
- [ ] Permission checks
- [ ] Pagination support

---

#### TASK-062: Sync Protocol - Multi-User Updates
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Update sync protocol to handle shared resources.

**Acceptance Criteria:**
- [ ] Sync includes organization context
- [ ] Conflict resolution for multi-user edits
- [ ] Optimistic locking implemented
- [ ] Last-write-wins for shared resources
- [ ] Sync notifications for team changes

---

#### TASK-063: Shared Resources - Conflict Resolution
**Priority:** P1
**Effort:** M (8 hours)

**Description:**
Implement conflict resolution for simultaneous edits.

**Acceptance Criteria:**
- [ ] Detect concurrent edits
- [ ] Show conflict UI to users
- [ ] Allow user to choose resolution
- [ ] Preserve all data (no loss)
- [ ] Log conflicts for analysis

---

### Frontend Tasks (46 hours)

#### TASK-064: Shared Connections UI
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Update connections UI to show and manage shared connections.

**Acceptance Criteria:**
- [ ] Visibility toggle on connection form
- [ ] Organization connections section in sidebar
- [ ] "Shared" badge indicator
- [ ] Filter: personal/shared
- [ ] Cannot edit others' shared connections

---

#### TASK-065: Shared Queries UI
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Update queries UI to show and manage shared queries.

**Acceptance Criteria:**
- [ ] Visibility toggle on save query modal
- [ ] Organization queries in query library
- [ ] "Shared" badge
- [ ] Filter options
- [ ] View-only for others' queries

---

#### TASK-066: Multi-User Indicators
**Priority:** P1
**Effort:** L (10 hours)

**Description:**
Add visual indicators for multi-user collaboration.

**Acceptance Criteria:**
- [ ] "Currently editing" indicator
- [ ] Last modified by user display
- [ ] Real-time presence (optional)
- [ ] Conflict warning indicators

---

#### TASK-067: Resource Sharing Flow
**Priority:** P1
**Effort:** M (8 hours)

**Description:**
Create UI flow for sharing/unsharing resources.

**Acceptance Criteria:**
- [ ] Share button on connection/query cards
- [ ] Confirmation dialog
- [ ] Unshare action
- [ ] Shows who has access

---

#### TASK-068: Sync Store Updates
**Priority:** P1
**Effort:** S (4 hours)

**Description:**
Update sync store to handle shared resources.

**Acceptance Criteria:**
- [ ] Sync respects organization context
- [ ] Handles shared resource updates
- [ ] Merges changes from other users
- [ ] Conflict resolution UI trigger

---

### Testing Tasks (30 hours)

#### TASK-069: Multi-User Sync Tests
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Test multi-user concurrent editing scenarios.

**Acceptance Criteria:**
- [ ] Test 2 users editing same connection
- [ ] Test 2 users editing same query
- [ ] Test conflict detection
- [ ] Test conflict resolution
- [ ] No data loss in any scenario

---

#### TASK-070: Integration Tests - Sharing
**Priority:** P1
**Effort:** M (8 hours)

**Description:**
Integration tests for resource sharing.

**Acceptance Criteria:**
- [ ] Test share connection flow
- [ ] Test share query flow
- [ ] Test unshare flow
- [ ] Test permission enforcement
- [ ] Test visibility changes

---

#### TASK-071: Performance Tests - Shared Resources
**Priority:** P2
**Effort:** M (6 hours)

**Description:**
Performance testing with large teams.

**Acceptance Criteria:**
- [ ] Test with 50 members
- [ ] Test with 100 shared connections
- [ ] Test with 500 shared queries
- [ ] Measure load times
- [ ] Optimize slow queries

---

#### TASK-072: Documentation - Sharing
**Priority:** P2
**Effort:** S (4 hours)

**Description:**
Document resource sharing features.

**Acceptance Criteria:**
- [ ] User guide for sharing
- [ ] API documentation
- [ ] Sync protocol update

---

**Sprint 4 Total:** 134 hours (3.35 person-weeks)

---

## Sprint 5: Testing & Polish (Week 18)

### Testing Tasks (68 hours)

#### TASK-073: E2E Testing Suite - Complete Flows
**Priority:** P0
**Effort:** XL (16 hours)

**Description:**
Comprehensive E2E tests covering all Phase 3 features.

**Test Scenarios:**
- [ ] Create org → invite member → accept → collaborate
- [ ] Share connection → edit by multiple users → resolve conflict
- [ ] Change member role → verify permission changes
- [ ] Remove member → verify access revoked
- [ ] Delete organization → verify cleanup

---

#### TASK-074: Performance Testing
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Load and performance testing for team features.

**Test Scenarios:**
- [ ] 50 concurrent users in single org
- [ ] Permission check latency
- [ ] Sync performance with 20 member team
- [ ] Large organization list (100+ orgs)
- [ ] Audit log query performance (10K entries)

**Performance Targets:**
- [ ] Permission check < 50ms (p95)
- [ ] Org list load < 200ms (p95)
- [ ] Sync latency < 1s (p95)
- [ ] Audit log query < 500ms (p95)

---

#### TASK-075: Security Testing
**Priority:** P0
**Effort:** L (12 hours)

**Description:**
Security penetration testing and vulnerability assessment.

**Test Scenarios:**
- [ ] Attempt privilege escalation
- [ ] Attempt to access other orgs
- [ ] SQL injection attempts
- [ ] XSS vulnerability testing
- [ ] CSRF protection verification
- [ ] Authorization bypass attempts

**Acceptance Criteria:**
- [ ] Zero critical vulnerabilities
- [ ] Zero high-severity vulnerabilities
- [ ] All P1 medium vulnerabilities fixed

---

#### TASK-076: Data Integrity Testing
**Priority:** P0
**Effort:** M (8 hours)

**Description:**
Test data consistency and integrity under various scenarios.

**Test Scenarios:**
- [ ] Multi-user concurrent updates
- [ ] Network interruptions during sync
- [ ] Partial sync failures
- [ ] Org deletion with active members
- [ ] Member removal cleanup

**Acceptance Criteria:**
- [ ] No data loss scenarios found
- [ ] No orphaned records
- [ ] Referential integrity maintained

---

#### TASK-077: Cross-Browser Testing
**Priority:** P1
**Effort:** M (6 hours)

**Description:**
Test all features across major browsers.

**Browsers:**
- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)

**Acceptance Criteria:**
- [ ] All features work in all browsers
- [ ] UI renders correctly
- [ ] No console errors

---

#### TASK-078: Regression Testing
**Priority:** P1
**Effort:** M (6 hours)

**Description:**
Verify Phase 1 and Phase 2 features still work.

**Acceptance Criteria:**
- [ ] Local-first still works (Phase 1)
- [ ] Individual sync still works (Phase 2)
- [ ] No breaking changes
- [ ] All existing tests passing

---

#### TASK-079: Accessibility Testing
**Priority:** P2
**Effort:** S (4 hours)

**Description:**
Test accessibility compliance (WCAG 2.1 Level AA).

**Acceptance Criteria:**
- [ ] Keyboard navigation works
- [ ] Screen reader compatible
- [ ] Color contrast meets standards
- [ ] ARIA labels present
- [ ] No accessibility blockers

---

### Bug Fixes & Polish (32 hours)

#### TASK-080: Bug Triage & P0/P1 Fixes
**Priority:** P0
**Effort:** XL (16 hours)

**Description:**
Fix all high-priority bugs discovered during testing.

**Acceptance Criteria:**
- [ ] All P0 bugs fixed
- [ ] All P1 bugs fixed
- [ ] Fixes verified by QA

---

#### TASK-081: UI/UX Polish
**Priority:** P1
**Effort:** L (12 hours)

**Description:**
Polish UI for professional look and feel.

**Tasks:**
- [ ] Improve loading states
- [ ] Add empty states for all views
- [ ] Improve error messages
- [ ] Add smooth animations
- [ ] Consistent spacing/styling
- [ ] Tooltips for clarity

---

#### TASK-082: Performance Optimization
**Priority:** P1
**Effort:** L (10 hours)

**Description:**
Optimize performance based on testing results.

**Tasks:**
- [ ] Add database indexes where needed
- [ ] Optimize permission check queries
- [ ] Reduce frontend bundle size
- [ ] Add lazy loading for lists
- [ ] Implement pagination
- [ ] Add caching where appropriate

---

### Documentation (24 hours)

#### TASK-083: User Documentation - Complete Guide
**Priority:** P1
**Effort:** M (8 hours)

**Description:**
Complete user guide for team collaboration features.

**Sections:**
- [ ] Getting started with teams
- [ ] Inviting members
- [ ] Managing roles
- [ ] Sharing resources
- [ ] Best practices
- [ ] Troubleshooting

---

#### TASK-084: Admin Guide
**Priority:** P1
**Effort:** M (6 hours)

**Description:**
Guide for organization owners/admins.

**Sections:**
- [ ] Organization management
- [ ] Member management
- [ ] Audit logs
- [ ] Security best practices

---

#### TASK-085: API Documentation - Complete
**Priority:** P1
**Effort:** M (6 hours)

**Description:**
Complete API documentation for all Phase 3 endpoints.

**Acceptance Criteria:**
- [ ] OpenAPI spec complete
- [ ] All endpoints documented
- [ ] Request/response examples
- [ ] Error codes documented

---

#### TASK-086: Video Tutorials
**Priority:** P2
**Effort:** S (4 hours)

**Description:**
Record screen capture tutorials.

**Videos:**
- [ ] Creating and managing organizations
- [ ] Inviting team members
- [ ] Sharing connections and queries

---

### Deployment Preparation (20 hours)

#### TASK-087: Production Deployment Checklist
**Priority:** P0
**Effort:** M (6 hours)

**Description:**
Create and execute deployment checklist.

**Checklist:**
- [ ] Database migrations ready
- [ ] Environment variables configured
- [ ] Secrets rotated
- [ ] Monitoring configured
- [ ] Alerts configured
- [ ] Rollback plan tested

---

#### TASK-088: Monitoring & Alerting Setup
**Priority:** P0
**Effort:** M (6 hours)

**Description:**
Configure monitoring for team features.

**Metrics:**
- [ ] Organization creation rate
- [ ] Invitation acceptance rate
- [ ] Permission check errors
- [ ] Sync conflicts
- [ ] API error rates

**Alerts:**
- [ ] Permission check latency > 100ms
- [ ] Sync failure rate > 5%
- [ ] High number of 403 errors

---

#### TASK-089: Error Tracking Configuration
**Priority:** P1
**Effort:** S (4 hours)

**Description:**
Configure Sentry/error tracking for Phase 3.

**Acceptance Criteria:**
- [ ] Errors tagged with Phase 3 features
- [ ] Source maps uploaded
- [ ] Alert rules configured
- [ ] Team members added

---

#### TASK-090: Beta Testing Program Setup
**Priority:** P1
**Effort:** S (4 hours)

**Description:**
Prepare beta testing program for team features.

**Tasks:**
- [ ] Create beta testing plan
- [ ] Prepare feedback forms
- [ ] Set up analytics tracking
- [ ] Create beta announcement
- [ ] Identify beta testers (10-20 teams)

---

### Final Review (8 hours)

#### TASK-091: Stakeholder Demo
**Priority:** P0
**Effort:** S (2 hours)

**Description:**
Demo Phase 3 features to stakeholders.

**Acceptance Criteria:**
- [ ] Demo prepared
- [ ] All features showcased
- [ ] Stakeholder feedback collected

---

#### TASK-092: Engineering Sign-off
**Priority:** P0
**Effort:** XS (2 hours)

**Description:**
Engineering team reviews and approves Phase 3.

**Checklist:**
- [ ] All P0/P1 tasks complete
- [ ] All tests passing
- [ ] Performance targets met
- [ ] Security audit passed
- [ ] Documentation complete

---

#### TASK-093: Product Manager Sign-off
**Priority:** P0
**Effort:** XS (2 hours)

**Description:**
Product manager approves Phase 3 for launch.

**Acceptance Criteria:**
- [ ] All must-have features complete
- [ ] User experience acceptable
- [ ] Documentation sufficient

---

#### TASK-094: Security Sign-off
**Priority:** P0
**Effort:** XS (2 hours)

**Description:**
Security team approves Phase 3 for production.

**Acceptance Criteria:**
- [ ] Security audit passed
- [ ] No critical vulnerabilities
- [ ] RBAC correctly implemented

---

**Sprint 5 Total:** 152 hours (3.8 person-weeks)

---

## Task Summary

### By Sprint

| Sprint | Tasks | Hours | Person-Weeks |
|--------|-------|-------|--------------|
| Sprint 1: Foundation | 21 | 150 | 3.75 |
| Sprint 2: Invitations | 18 | 100 | 2.50 |
| Sprint 3: RBAC | 18 | 150 | 3.75 |
| Sprint 4: Sharing | 15 | 134 | 3.35 |
| Sprint 5: Polish | 22 | 152 | 3.80 |
| **Total** | **94** | **686** | **17.15** |

**Note:** Actual implementation plan estimates 536 hours. The difference (150 hours) represents buffer time and efficiency gains from parallel work.

### By Priority

| Priority | Tasks | Percentage |
|----------|-------|------------|
| P0 (Must-Have) | 42 | 45% |
| P1 (Must-Have) | 38 | 40% |
| P2 (Should-Have) | 12 | 13% |
| P3 (Could-Have) | 2 | 2% |

### By Role

| Role | Tasks | Hours | Percentage |
|------|-------|-------|------------|
| Backend Engineer | 35 | 298 | 43% |
| Frontend Engineer | 32 | 246 | 36% |
| QA Engineer | 17 | 98 | 14% |
| UI/UX Designer | 3 | 14 | 2% |
| Product Manager | 4 | 16 | 2% |
| Security Engineer | 3 | 16 | 2% |

---

## Dependencies Graph

**Critical Path:**

1. TASK-001 (Org schema) → TASK-004 (Repo) → TASK-005 (Service) → TASK-006-008 (APIs)
2. TASK-011 (Types) → TASK-012 (Store) → TASK-013-016 (UI)
3. TASK-022 (Token gen) → TASK-023-024 (Invite APIs) → TASK-028-031 (Invite UI)
4. TASK-040 (Permissions) → TASK-041 (Middleware) → TASK-042 (Endpoints) → TASK-048-051 (Permission UI)
5. TASK-058-062 (Shared resources backend) → TASK-064-068 (Shared resources frontend)

**Parallel Work Opportunities:**

- Backend and frontend can work in parallel once APIs are defined
- Testing can start early with unit tests
- Documentation can be written alongside development
- UI/UX design can be done ahead of implementation

---

## Risk Mitigation

### High-Risk Tasks

1. **TASK-041: Permission Middleware** (High complexity)
   - Mitigation: Security review, extensive testing
   - Fallback: Simplify to 2 roles if needed

2. **TASK-062: Sync Protocol Updates** (Multi-user conflicts)
   - Mitigation: Comprehensive conflict testing
   - Fallback: Pessimistic locking for shared resources

3. **TASK-055: Security Audit** (External dependency)
   - Mitigation: Schedule early, have backup auditor
   - Fallback: Internal security review if external delayed

4. **TASK-073: E2E Testing** (Time-consuming)
   - Mitigation: Start early, automate everything
   - Fallback: Manual testing for critical paths

---

## Success Metrics

### Development Metrics

- [ ] 100% of P0 tasks completed
- [ ] 95%+ of P1 tasks completed
- [ ] >85% backend test coverage
- [ ] >80% frontend test coverage
- [ ] All E2E tests passing

### Quality Metrics

- [ ] Zero P0 bugs at launch
- [ ] <5 P1 bugs at launch
- [ ] Security audit passed
- [ ] Performance targets met

### User Metrics (Beta)

- [ ] 50+ teams created
- [ ] 70%+ invitation acceptance rate
- [ ] <5% support tickets per user
- [ ] 90%+ user satisfaction

---

## Document Metadata

**Version:** 1.0
**Created:** 2025-10-23
**Last Updated:** 2025-10-23
**Owner:** Engineering Team
**Status:** Ready for Review

**Related Documents:**
- [PHASE_3_IMPLEMENTATION_PLAN.md](./PHASE_3_IMPLEMENTATION_PLAN.md)
- [PHASE_3_RISKS.md](./PHASE_3_RISKS.md)
- [PHASE_3_TESTING_STRATEGY.md](./PHASE_3_TESTING_STRATEGY.md)
- [PHASE_3_KICKOFF.md](./PHASE_3_KICKOFF.md)
