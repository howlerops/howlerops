import { test, expect, Page } from '@playwright/test'

/**
 * E2E Tests for Shared Resources Feature
 *
 * Tests the complete workflow for sharing database connections and queries
 * within organizations, including permissions, conflicts, and sync.
 */

// ====================================================================
// Test Setup and Helpers
// ====================================================================

async function login(page: Page, email: string, password: string = 'password123') {
  await page.goto('/login')
  await page.fill('[name="email"]', email)
  await page.fill('[name="password"]', password)
  await page.click('button[type="submit"]')
  await page.waitForURL('/dashboard', { timeout: 5000 })
}

async function createOrganization(page: Page, name: string): Promise<string> {
  await page.goto('/organizations')
  await page.click('[data-testid="create-org-button"]')
  await page.fill('[name="org-name"]', name)
  await page.click('[data-testid="submit-org"]')

  // Wait for org to be created and get ID from URL
  await page.waitForURL(/\/organizations\/org-.*/)
  const url = page.url()
  const orgId = url.split('/').pop() || ''

  return orgId
}

async function inviteMember(page: Page, orgId: string, email: string, role: string = 'member') {
  await page.goto(`/organizations/${orgId}`)
  await page.click('[data-testid="invite-member-button"]')
  await page.fill('[name="member-email"]', email)
  await page.selectOption('[name="member-role"]', role)
  await page.click('[data-testid="send-invite"]')

  await expect(page.locator('text=Invitation sent')).toBeVisible({ timeout: 3000 })
}

// ====================================================================
// E2E Test: Share Connection Workflow
// ====================================================================

test('Share connection workflow - Admin creates and shares', async ({ page }) => {
  // Login as admin
  await login(page, 'admin@test.com')

  // Create organization
  const orgId = await createOrganization(page, 'E2E Test Org')

  // Navigate to connections
  await page.goto('/connections')
  await page.click('[data-testid="new-connection"]')

  // Fill connection form
  await page.fill('[name="name"]', 'Shared Production DB')
  await page.selectOption('[name="type"]', 'postgres')
  await page.fill('[name="host"]', 'prod.example.com')
  await page.fill('[name="port"]', '5432')
  await page.fill('[name="database"]', 'production')
  await page.fill('[name="username"]', 'produser')

  // Set visibility to shared and select organization
  await page.selectOption('[name="visibility"]', 'shared')
  await page.selectOption('[name="organization"]', orgId)

  // Save
  await page.click('[data-testid="save-connection"]')

  // Verify success message
  await expect(page.locator('text=Connection saved')).toBeVisible()

  // Verify shared badge is visible
  await expect(page.locator('[data-testid="shared-badge"]')).toBeVisible()

  // Verify connection appears in organization view
  await page.goto(`/organizations/${orgId}/connections`)
  await expect(page.locator('text=Shared Production DB')).toBeVisible()
})

// ====================================================================
// E2E Test: Member sees shared resources
// ====================================================================

test('Member sees shared resources from organization', async ({ page, context }) => {
  // Setup: Admin creates org and shares connection
  const adminPage = await context.newPage()
  await login(adminPage, 'admin@test.com')

  const orgId = await createOrganization(adminPage, 'Share Test Org')

  // Invite member
  await inviteMember(adminPage, orgId, 'member@test.com', 'member')

  // Admin creates shared connection
  await adminPage.goto('/connections')
  await adminPage.click('[data-testid="new-connection"]')
  await adminPage.fill('[name="name"]', 'Team Database')
  await adminPage.selectOption('[name="type"]', 'mysql')
  await adminPage.fill('[name="host"]', 'team-db.local')
  await adminPage.fill('[name="port"]', '3306')
  await adminPage.selectOption('[name="visibility"]', 'shared')
  await adminPage.selectOption('[name="organization"]', orgId)
  await adminPage.click('[data-testid="save-connection"]')

  await adminPage.close()

  // Test: Member logs in
  await login(page, 'member@test.com')

  // Accept invitation
  await page.goto('/invitations')
  await page.click(`[data-testid="accept-invite-${orgId}"]`)

  // Navigate to shared resources
  await page.goto('/shared-resources')

  // Verify member sees the shared connection
  await expect(page.locator('text=Team Database')).toBeVisible()
  await expect(page.locator(`[data-org-id="${orgId}"]`)).toBeVisible()
})

// ====================================================================
// E2E Test: Member cannot share without permission
// ====================================================================

test('Member without permission cannot share connection', async ({ page }) => {
  await login(page, 'member@test.com')

  // Create personal connection
  await page.goto('/connections')
  await page.click('[data-testid="new-connection"]')
  await page.fill('[name="name"]', 'My Personal DB')
  await page.selectOption('[name="type"]', 'postgres')
  await page.fill('[name="host"]', 'localhost')
  await page.selectOption('[name="visibility"]', 'personal')
  await page.click('[data-testid="save-connection"]')

  // Try to change to shared
  await page.click('[data-testid="edit-connection"]')

  // Verify share option is disabled or shows permission error
  const shareSelect = page.locator('[name="visibility"]')

  // Try to select shared
  await shareSelect.selectOption('shared')
  await page.click('[data-testid="save-connection"]')

  // Verify error message
  await expect(page.locator('text=Insufficient permissions')).toBeVisible({ timeout: 3000 })

  // Or verify option is disabled
  // await expect(shareSelect.locator('option[value="shared"]')).toBeDisabled()
})

// ====================================================================
// E2E Test: Conflict resolution dialog
// ====================================================================

test('Conflict resolution for shared connection', async ({ page, context }) => {
  // Setup: Two users edit same shared connection
  const user1Page = page
  const user2Page = await context.newPage()

  await login(user1Page, 'admin@test.com')
  await login(user2Page, 'admin2@test.com')

  // Admin1 creates org and shares connection
  const orgId = await createOrganization(user1Page, 'Conflict Test Org')
  await inviteMember(user1Page, orgId, 'admin2@test.com', 'admin')

  await user1Page.goto('/connections')
  await user1Page.click('[data-testid="new-connection"]')
  await user1Page.fill('[name="name"]', 'Shared Conflict DB')
  await user1Page.selectOption('[name="type"]', 'postgres')
  await user1Page.selectOption('[name="visibility"]', 'shared')
  await user1Page.selectOption('[name="organization"]', orgId)
  await user1Page.click('[data-testid="save-connection"]')

  // Get connection ID
  const connId = await user1Page.locator('[data-testid="connection-id"]').textContent()

  // User 2 accepts invite
  await user2Page.goto('/invitations')
  await user2Page.click(`[data-testid="accept-invite-${orgId}"]`)

  // Both users edit the connection
  // User 1 edits
  await user1Page.goto(`/connections/${connId}/edit`)
  await user1Page.fill('[name="name"]', 'Updated by User 1')

  // User 2 edits (before user 1 saves)
  await user2Page.goto(`/connections/${connId}/edit`)
  await user2Page.fill('[name="name"]', 'Updated by User 2')

  // User 1 saves first
  await user1Page.click('[data-testid="save-connection"]')
  await expect(user1Page.locator('text=Saved successfully')).toBeVisible()

  // User 2 saves second - should trigger conflict
  await user2Page.click('[data-testid="save-connection"]')

  // Verify conflict dialog appears
  await expect(user2Page.locator('[data-testid="conflict-dialog"]')).toBeVisible({ timeout: 5000 })
  await expect(user2Page.locator('text=Conflict detected')).toBeVisible()

  // Verify both versions shown
  await expect(user2Page.locator('text=Updated by User 1')).toBeVisible()
  await expect(user2Page.locator('text=Updated by User 2')).toBeVisible()

  // User 2 chooses to keep their version
  await user2Page.click('[data-testid="choose-local"]')
  await user2Page.click('[data-testid="resolve-conflict"]')

  // Verify resolution succeeded
  await expect(user2Page.locator('text=Conflict resolved')).toBeVisible()

  await user2Page.close()
})

// ====================================================================
// E2E Test: Share saved query workflow
// ====================================================================

test('Share saved query with organization', async ({ page }) => {
  await login(page, 'admin@test.com')

  const orgId = await createOrganization(page, 'Query Share Org')

  // Create and share a query
  await page.goto('/queries')
  await page.click('[data-testid="new-query"]')

  await page.fill('[name="query-name"]', 'Shared Report Query')
  await page.fill('[name="query-text"]', 'SELECT * FROM users WHERE active = true')

  // Share with organization
  await page.selectOption('[name="visibility"]', 'shared')
  await page.selectOption('[name="organization"]', orgId)

  await page.click('[data-testid="save-query"]')

  // Verify success
  await expect(page.locator('text=Query saved')).toBeVisible()
  await expect(page.locator('[data-testid="shared-badge"]')).toBeVisible()

  // Verify in org queries
  await page.goto(`/organizations/${orgId}/queries`)
  await expect(page.locator('text=Shared Report Query')).toBeVisible()
})

// ====================================================================
// E2E Test: Unshare resource
// ====================================================================

test('Admin unshares connection from organization', async ({ page }) => {
  await login(page, 'admin@test.com')

  const orgId = await createOrganization(page, 'Unshare Test Org')

  // Create shared connection
  await page.goto('/connections')
  await page.click('[data-testid="new-connection"]')
  await page.fill('[name="name"]', 'Will Unshare')
  await page.selectOption('[name="type"]', 'postgres')
  await page.selectOption('[name="visibility"]', 'shared')
  await page.selectOption('[name="organization"]', orgId)
  await page.click('[data-testid="save-connection"]')

  // Verify it's in org connections
  await page.goto(`/organizations/${orgId}/connections`)
  await expect(page.locator('text=Will Unshare')).toBeVisible()

  // Unshare
  await page.click('[data-testid="connection-menu"]')
  await page.click('[data-testid="unshare-connection"]')

  // Confirm
  await page.click('[data-testid="confirm-unshare"]')

  // Verify success
  await expect(page.locator('text=Connection unshared')).toBeVisible()

  // Verify no longer in org connections
  await page.goto(`/organizations/${orgId}/connections`)
  await expect(page.locator('text=Will Unshare')).not.toBeVisible()

  // Verify still in personal connections
  await page.goto('/connections')
  await expect(page.locator('text=Will Unshare')).toBeVisible()
  await expect(page.locator('[data-testid="personal-badge"]')).toBeVisible()
})

// ====================================================================
// E2E Test: Multi-device sync of shared resources
// ====================================================================

test('Shared resources sync across devices', async ({ page, context }) => {
  // Simulate two devices for same user
  const device1 = page
  const device2 = await context.newPage()

  // Login on both devices
  await login(device1, 'admin@test.com')
  await login(device2, 'admin@test.com')

  const orgId = await createOrganization(device1, 'Sync Test Org')

  // Device 1: Create shared connection
  await device1.goto('/connections')
  await device1.click('[data-testid="new-connection"]')
  await device1.fill('[name="name"]', 'Synced DB')
  await device1.selectOption('[name="type"]', 'postgres')
  await device1.selectOption('[name="visibility"]', 'shared')
  await device1.selectOption('[name="organization"]', orgId)
  await device1.click('[data-testid="save-connection"]')

  // Wait for sync
  await device1.waitForTimeout(2000)

  // Device 2: Trigger sync
  await device2.goto('/connections')
  await device2.click('[data-testid="sync-button"]')

  // Wait for sync to complete
  await expect(device2.locator('[data-testid="sync-status"]')).toHaveText('Synced', { timeout: 5000 })

  // Verify connection appears on device 2
  await expect(device2.locator('text=Synced DB')).toBeVisible()
  await expect(device2.locator('[data-testid="shared-badge"]')).toBeVisible()

  await device2.close()
})

// ====================================================================
// E2E Test: Filter shared resources by organization
// ====================================================================

test('Filter shared resources by organization', async ({ page }) => {
  await login(page, 'admin@test.com')

  // Create 2 organizations
  const org1Id = await createOrganization(page, 'Org Alpha')
  const org2Id = await createOrganization(page, 'Org Beta')

  // Create connections in different orgs
  await page.goto('/connections')

  // Org 1 connection
  await page.click('[data-testid="new-connection"]')
  await page.fill('[name="name"]', 'Alpha DB')
  await page.selectOption('[name="type"]', 'postgres')
  await page.selectOption('[name="visibility"]', 'shared')
  await page.selectOption('[name="organization"]', org1Id)
  await page.click('[data-testid="save-connection"]')

  // Org 2 connection
  await page.click('[data-testid="new-connection"]')
  await page.fill('[name="name"]', 'Beta DB')
  await page.selectOption('[name="type"]', 'mysql')
  await page.selectOption('[name="visibility"]', 'shared')
  await page.selectOption('[name="organization"]', org2Id)
  await page.click('[data-testid="save-connection"]')

  // Go to shared resources
  await page.goto('/shared-resources')

  // Filter by Org Alpha
  await page.selectOption('[data-testid="org-filter"]', org1Id)

  // Verify only Alpha DB visible
  await expect(page.locator('text=Alpha DB')).toBeVisible()
  await expect(page.locator('text=Beta DB')).not.toBeVisible()

  // Filter by Org Beta
  await page.selectOption('[data-testid="org-filter"]', org2Id)

  // Verify only Beta DB visible
  await expect(page.locator('text=Beta DB')).toBeVisible()
  await expect(page.locator('text=Alpha DB')).not.toBeVisible()

  // Show all
  await page.selectOption('[data-testid="org-filter"]', 'all')

  // Verify both visible
  await expect(page.locator('text=Alpha DB')).toBeVisible()
  await expect(page.locator('text=Beta DB')).toBeVisible()
})

// ====================================================================
// E2E Test: Permission upgrade allows sharing
// ====================================================================

test('Member promoted to admin can share resources', async ({ page, context }) => {
  const adminPage = await context.newPage()
  const memberPage = page

  await login(adminPage, 'admin@test.com')
  await login(memberPage, 'member@test.com')

  // Admin creates org and invites member
  const orgId = await createOrganization(adminPage, 'Permission Test Org')
  await inviteMember(adminPage, orgId, 'member@test.com', 'member')

  // Member accepts
  await memberPage.goto('/invitations')
  await memberPage.click(`[data-testid="accept-invite-${orgId}"]`)

  // Member creates connection
  await memberPage.goto('/connections')
  await memberPage.click('[data-testid="new-connection"]')
  await memberPage.fill('[name="name"]', 'Member DB')
  await memberPage.selectOption('[name="type"]', 'postgres')

  // Try to share - should fail
  await memberPage.selectOption('[name="visibility"]', 'shared')
  await memberPage.click('[data-testid="save-connection"]')
  await expect(memberPage.locator('text=Insufficient permissions')).toBeVisible()

  // Admin promotes member to admin
  await adminPage.goto(`/organizations/${orgId}/members`)
  await adminPage.click(`[data-testid="member-menu-member@test.com"]`)
  await adminPage.click('[data-testid="change-role"]')
  await adminPage.selectOption('[name="new-role"]', 'admin')
  await adminPage.click('[data-testid="confirm-role-change"]')

  // Member refreshes and tries again
  await memberPage.reload()
  await memberPage.goto('/connections')
  await memberPage.click('[data-testid="edit-connection"]')
  await memberPage.selectOption('[name="visibility"]', 'shared')
  await memberPage.selectOption('[name="organization"]', orgId)
  await memberPage.click('[data-testid="save-connection"]')

  // Should succeed now
  await expect(memberPage.locator('text=Connection saved')).toBeVisible()

  await adminPage.close()
})
