# Organization Components

React UI components for the team collaboration features in SQL Studio Phase 3.

## Components Overview

### 1. OrganizationList.tsx (252 lines)
**Purpose:** Grid/list view of user's organizations

**Features:**
- Displays organization cards with name, description, member count, and user's role
- "Create Organization" button
- Click handler to view organization details
- Empty state when no organizations exist
- Mobile-friendly variant (`OrganizationListMobile`)

**Props:**
```typescript
interface OrganizationListProps {
  organizations: OrganizationWithMembership[]
  loading?: boolean
  error?: string | null
  onCreateClick: () => void
  onOrganizationClick: (org: OrganizationWithMembership) => void
  className?: string
}
```

**Usage:**
```tsx
import { OrganizationList } from '@/components/organizations'

<OrganizationList
  organizations={organizations}
  loading={isLoading}
  error={error}
  onCreateClick={() => setShowCreateModal(true)}
  onOrganizationClick={(org) => navigate(`/orgs/${org.id}`)}
/>
```

---

### 2. OrganizationCreateModal.tsx (215 lines)
**Purpose:** Modal for creating new organizations

**Features:**
- Form with name (required, 3-50 chars) and description (optional, max 500 chars)
- Real-time validation with error messages
- Character counters for both fields
- Loading state during creation
- Error handling with alert display
- Auto-reset on close

**Props:**
```typescript
interface OrganizationCreateModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (data: CreateOrganizationInput) => Promise<void>
  loading?: boolean
  error?: string | null
}
```

**Usage:**
```tsx
import { OrganizationCreateModal } from '@/components/organizations'

<OrganizationCreateModal
  open={showModal}
  onOpenChange={setShowModal}
  onCreate={handleCreate}
  loading={isCreating}
  error={createError}
/>
```

---

### 3. OrganizationSettings.tsx (314 lines)
**Purpose:** Organization settings page

**Features:**
- Edit name and description (owner/admin only)
- View-only mode for members
- Display member count, max members, creation date
- Navigate to members management
- Delete organization (owner only) with confirmation dialog
- Unsaved changes tracking with save/cancel buttons
- Permission-based UI (hides/disables actions based on role)

**Props:**
```typescript
interface OrganizationSettingsProps {
  organization: Organization
  currentUserRole: OrganizationRole
  onUpdate: (data: UpdateOrganizationInput) => Promise<void>
  onDelete: () => Promise<void>
  onNavigateToMembers: () => void
  loading?: boolean
  error?: string | null
  className?: string
}
```

**Usage:**
```tsx
import { OrganizationSettings } from '@/components/organizations'

<OrganizationSettings
  organization={currentOrg}
  currentUserRole={userRole}
  onUpdate={handleUpdate}
  onDelete={handleDelete}
  onNavigateToMembers={() => navigate('members')}
  loading={isUpdating}
  error={updateError}
/>
```

---

### 4. OrganizationSwitcher.tsx (211 lines)
**Purpose:** Dropdown for switching between organizations

**Features:**
- Shows current organization or "Personal Workspace"
- Lists all user's organizations
- Quick switch between orgs
- "Create Organization" option
- Keyboard accessible
- Compact mobile variant (`OrganizationSwitcherCompact`)

**Props:**
```typescript
interface OrganizationSwitcherProps {
  organizations: OrganizationWithMembership[]
  currentOrganizationId: string | null
  onOrganizationChange: (organizationId: string | null) => void
  onCreateClick?: () => void
  loading?: boolean
  className?: string
}
```

**Usage:**
```tsx
import { OrganizationSwitcher } from '@/components/organizations'

<OrganizationSwitcher
  organizations={organizations}
  currentOrganizationId={currentOrgId}
  onOrganizationChange={handleOrgChange}
  onCreateClick={() => setShowCreateModal(true)}
/>
```

---

### 5. MembersList.tsx (397 lines)
**Purpose:** Team members management table

**Features:**
- Table with member details (name, email, role, joined date)
- Role dropdown for changing roles (owner/admin only)
- Remove member button (owner/admin only, can't remove self or owner)
- Tooltips explaining disabled actions
- "Invite Member" button
- Relative time display with hover tooltips showing exact dates
- Mobile-friendly variant (`MembersListMobile`)

**Props:**
```typescript
interface MembersListProps {
  members: OrganizationMember[]
  currentUserId: string
  currentUserRole: OrganizationRole
  onRoleChange: (memberId: string, data: UpdateMemberRoleInput) => Promise<void>
  onRemoveMember: (memberId: string) => Promise<void>
  onInviteClick: () => void
  loading?: boolean
  error?: string | null
  className?: string
}
```

**Usage:**
```tsx
import { MembersList } from '@/components/organizations'

<MembersList
  members={members}
  currentUserId={userId}
  currentUserRole={userRole}
  onRoleChange={handleRoleChange}
  onRemoveMember={handleRemove}
  onInviteClick={() => setShowInviteModal(true)}
  loading={isLoading}
  error={error}
/>
```

---

### 6. InviteMemberModal.tsx (328 lines)
**Purpose:** Modal for inviting new members

**Features:**
- Email input with validation
- Role selector (Admin or Member, based on permissions)
- Bulk invite support (comma-separated emails)
- Real-time email parsing and validation
- Shows pending invitations with ability to revoke
- Visual feedback with email badges
- Character limit enforcement

**Props:**
```typescript
interface InviteMemberModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onInvite: (invitations: CreateInvitationInput[]) => Promise<void>
  pendingInvitations?: OrganizationInvitation[]
  onRevokeInvitation?: (invitationId: string) => Promise<void>
  loading?: boolean
  error?: string | null
  canInviteAdmin?: boolean
}
```

**Usage:**
```tsx
import { InviteMemberModal } from '@/components/organizations'

<InviteMemberModal
  open={showModal}
  onOpenChange={setShowModal}
  onInvite={handleInvite}
  pendingInvitations={pendingInvites}
  onRevokeInvitation={handleRevoke}
  loading={isInviting}
  error={inviteError}
  canInviteAdmin={canInviteAdmin}
/>
```

---

## Type Definitions

All components use types from `/Users/jacob_1/projects/sql-studio/frontend/src/types/organization.ts`:

```typescript
export enum OrganizationRole {
  Owner = 'owner',
  Admin = 'admin',
  Member = 'member',
}

export interface Organization {
  id: string
  name: string
  description?: string
  owner_id: string
  created_at: Date
  updated_at: Date
  deleted_at?: Date | null
  max_members: number
  settings?: Record<string, unknown>
  member_count?: number
}

export interface OrganizationMember {
  id: string
  organization_id: string
  user_id: string
  role: OrganizationRole
  invited_by?: string | null
  joined_at: Date
  user?: UserInfo
}

export interface OrganizationInvitation {
  id: string
  organization_id: string
  email: string
  role: OrganizationRole
  invited_by: string
  token: string
  expires_at: Date
  accepted_at?: Date | null
  created_at: Date
  organization?: Organization
}
```

---

## Permission Helpers

The type definitions include permission helper functions:

```typescript
canInviteMembers(role: OrganizationRole): boolean
canRemoveMembers(role: OrganizationRole): boolean
canUpdateSettings(role: OrganizationRole): boolean
canDeleteOrganization(role: OrganizationRole): boolean
canChangeRole(currentRole: OrganizationRole, targetRole: OrganizationRole): boolean
```

---

## Design Patterns

### 1. Consistent Error Handling
All components accept optional `error` and `loading` props for consistent state management.

### 2. Permission-Based UI
Components automatically hide/disable actions based on user permissions using helper functions.

### 3. Accessibility
- ARIA labels on all interactive elements
- Keyboard navigation support (Enter/Space on cards)
- Screen reader friendly tooltips
- Proper focus management

### 4. Mobile Responsiveness
- Desktop components use tables, grids, and full modals
- Mobile variants provided for compact screens
- Touch-friendly tap targets

### 5. Loading States
- Skeleton states for initial loads
- Inline spinners for actions
- Disabled states during operations

### 6. Empty States
- Helpful messages when no data exists
- Clear call-to-action buttons
- Iconography for visual context

---

## Integration with Store

These components are designed to work with an organization store (to be created):

```typescript
// Example store integration
import { useOrganizationStore } from '@/store/organization-store'
import { OrganizationList } from '@/components/organizations'

function OrganizationsPage() {
  const {
    organizations,
    loading,
    error,
    createOrganization,
  } = useOrganizationStore()

  return (
    <OrganizationList
      organizations={organizations}
      loading={loading}
      error={error}
      onCreateClick={() => setShowModal(true)}
      onOrganizationClick={(org) => navigate(`/orgs/${org.id}`)}
    />
  )
}
```

---

## Styling

All components use:
- **Tailwind CSS** for styling
- **shadcn/ui** components (Button, Card, Dialog, Table, etc.)
- **lucide-react** for icons
- Consistent color scheme with existing UI
- Dark mode support via CSS variables

---

## TypeScript Compliance

All components:
- Have proper TypeScript types
- Pass type checking (verified with `npm run typecheck`)
- Export type definitions for props
- Use strict null checks

---

## Files Created

1. `/Users/jacob_1/projects/sql-studio/frontend/src/types/organization.ts` (768 lines)
2. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/OrganizationList.tsx` (252 lines)
3. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/OrganizationCreateModal.tsx` (215 lines)
4. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/OrganizationSettings.tsx` (314 lines)
5. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/OrganizationSwitcher.tsx` (211 lines)
6. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/MembersList.tsx` (397 lines)
7. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/InviteMemberModal.tsx` (328 lines)
8. `/Users/jacob_1/projects/sql-studio/frontend/src/components/organizations/index.ts` (12 lines)

**Total:** 2,497 lines of production-ready TypeScript React code

---

## Next Steps

1. Create organization store (`/store/organization-store.ts`)
2. Create API client (`/lib/api/organization-client.ts`)
3. Add organization routes to the router
4. Create organization pages using these components
5. Add tests for components
