import { expect, Page,test } from '@playwright/test';

/**
 * Organization E2E Tests
 *
 * These tests cover the complete user journey for organization features:
 * - Creating organizations
 * - Inviting members
 * - Accepting invitations
 * - Managing members and roles
 * - Permission enforcement
 */

// Helper: Login helper function
async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('/dashboard');
}

// Helper: Create organization
async function createOrganization(page: Page, name: string, description: string) {
  await page.click('button:has-text("Create Organization")');
  await page.fill('input[name="name"]', name);
  await page.fill('textarea[name="description"]', description);
  await page.click('button[type="submit"]:has-text("Create")');

  // Wait for success toast or redirect
  await expect(page.locator('text=Organization created')).toBeVisible({ timeout: 5000 });
}

test.describe('Organization Management', () => {
  test.beforeEach(async ({ page }) => {
    // Start fresh for each test
    await page.goto('/');
  });

  test('Test 1: Create Organization Flow', async ({ page }) => {
    // Given: A logged-in user
    await login(page, 'owner@test.com', 'password123');

    // When: User creates a new organization
    await createOrganization(page, 'Test Company', 'A test organization');

    // Then: Organization appears in the list
    await expect(page.locator('text=Test Company')).toBeVisible();

    // And: User is redirected to organization dashboard
    await expect(page).toHaveURL(/\/organizations\/[^/]+/);

    // And: Organization name is displayed
    await expect(page.locator('h1:has-text("Test Company")')).toBeVisible();
  });

  test('Test 2: Invite Member Flow', async ({ page }) => {
    // Given: User is organization owner
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');
    await page.click('text=Test Company');

    // When: User navigates to members page
    await page.click('a:has-text("Members")');
    await expect(page).toHaveURL(/\/members$/);

    // And: Clicks invite member
    await page.click('button:has-text("Invite Member")');

    // And: Fills invitation form
    await page.fill('input[name="email"]', 'newmember@test.com');
    await page.selectOption('select[name="role"]', 'member');
    await page.click('button[type="submit"]:has-text("Send Invitation")');

    // Then: Invitation appears in pending list
    await expect(page.locator('text=newmember@test.com')).toBeVisible();
    await expect(page.locator('text=Pending')).toBeVisible();

    // And: Success message is shown
    await expect(page.locator('text=Invitation sent')).toBeVisible();
  });

  test('Test 3: Accept Invitation Flow (Multi-User)', async ({ page, context }) => {
    // Setup: Owner creates organization and invites member
    await login(page, 'owner@test.com', 'password123');
    await createOrganization(page, 'Collaborative Team', 'Team workspace');
    await page.click('a:has-text("Members")');
    await page.click('button:has-text("Invite Member")');
    await page.fill('input[name="email"]', 'member@test.com');
    await page.selectOption('select[name="role"]', 'member');
    await page.click('button[type="submit"]:has-text("Send Invitation")');

    // Get invitation token from the page (in real app, would be from email)
    const invitationLink = await page.locator('[data-testid="invitation-link"]').getAttribute('href');
    await expect(invitationLink).toBeTruthy();

    // Given: User B opens new browser context (simulate different user)
    const memberPage = await context.newPage();

    // When: User B logs in
    await login(memberPage, 'member@test.com', 'password123');

    // And: Navigates to invitations page
    await memberPage.goto('/invitations');

    // Then: User B sees the invitation notification
    await expect(memberPage.locator('text=Collaborative Team')).toBeVisible();
    await expect(memberPage.locator('text=Invitation from')).toBeVisible();

    // When: User B accepts the invitation
    await memberPage.click('button:has-text("Accept")');

    // Then: User B sees success message
    await expect(memberPage.locator('text=Joined organization')).toBeVisible();

    // And: Organization appears in User B's organization list
    await memberPage.goto('/organizations');
    await expect(memberPage.locator('text=Collaborative Team')).toBeVisible();

    // And: User A sees User B in members list (switch back to owner)
    await page.reload();
    await expect(page.locator('text=member@test.com')).toBeVisible();
    await expect(page.locator('text=Member')).toBeVisible(); // Role badge
  });

  test('Test 4: Permission Enforcement', async ({ page, context }) => {
    // Setup: Create org with owner and regular member
    await login(page, 'owner@test.com', 'password123');
    await createOrganization(page, 'Permission Test Org', 'Testing permissions');

    // Owner invites a member
    await page.click('a:has-text("Members")');
    await page.click('button:has-text("Invite Member")');
    await page.fill('input[name="email"]', 'member@test.com');
    await page.selectOption('select[name="role"]', 'member');
    await page.click('button[type="submit"]:has-text("Send Invitation")');

    // Member accepts (assume they've accepted)
    // ... (invitation acceptance flow)

    // Given: Member is logged in
    const memberPage = await context.newPage();
    await login(memberPage, 'member@test.com', 'password123');
    await memberPage.goto('/organizations');
    await memberPage.click('text=Permission Test Org');

    // When: Member tries to invite another member
    await memberPage.click('a:has-text("Members")');

    // Then: Invite button should be disabled or hidden
    const inviteButton = memberPage.locator('button:has-text("Invite Member")');
    await expect(inviteButton).toBeDisabled();
    // Or: await expect(inviteButton).not.toBeVisible();

    // When: Member tries to remove owner
    const ownerRow = memberPage.locator('[data-testid="member-row"]:has-text("owner@test.com")');
    const removeButton = ownerRow.locator('button:has-text("Remove")');

    // Then: Remove button should be disabled for owner
    await expect(removeButton).toBeDisabled();

    // Given: Owner promotes member to admin
    await page.click('button:has-text("owner@test.com")'); // Select member dropdown
    await page.selectOption('select[name="role"]', 'admin');
    await page.click('button:has-text("Update Role")');

    // When: Admin (formerly member) tries to invite
    await memberPage.reload();
    const adminInviteButton = memberPage.locator('button:has-text("Invite Member")');

    // Then: Admin can now invite members
    await expect(adminInviteButton).toBeEnabled();
  });

  test('Test 5: Organization Switching', async ({ page }) => {
    // Given: User with multiple organizations
    await login(page, 'multiorg@test.com', 'password123');

    // Create first organization
    await createOrganization(page, 'Company A', 'First company');

    // Create second organization
    await page.goto('/organizations');
    await createOrganization(page, 'Company B', 'Second company');

    // When: User switches to first organization
    await page.goto('/organizations');
    await page.click('text=Company A');

    // Then: Context changes to Company A
    await expect(page.locator('h1:has-text("Company A")')).toBeVisible();
    await page.click('a:has-text("Members")');

    // Members should be from Company A
    await expect(page.locator('text=multiorg@test.com')).toBeVisible();

    // When: User switches to second organization
    await page.click('[data-testid="org-switcher"]');
    await page.click('text=Company B');

    // Then: Context changes to Company B
    await expect(page.locator('h1:has-text("Company B")')).toBeVisible();
    await page.click('a:has-text("Members")');

    // Members list should update (only owner initially)
    const memberRows = page.locator('[data-testid="member-row"]');
    await expect(memberRows).toHaveCount(1);
  });
});

test.describe('Organization Settings', () => {
  test('Update organization details', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');
    await page.click('text=Test Company');
    await page.click('a:has-text("Settings")');

    // Update name and description
    await page.fill('input[name="name"]', 'Updated Company Name');
    await page.fill('textarea[name="description"]', 'Updated description');
    await page.click('button:has-text("Save Changes")');

    // Verify success
    await expect(page.locator('text=Settings updated')).toBeVisible();
    await expect(page.locator('h1:has-text("Updated Company Name")')).toBeVisible();
  });

  test('Delete organization', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await createOrganization(page, 'To Be Deleted', 'This will be deleted');

    await page.click('a:has-text("Settings")');
    await page.click('button:has-text("Delete Organization")');

    // Confirm deletion in modal
    await page.fill('input[name="confirmName"]', 'To Be Deleted');
    await page.click('button:has-text("Permanently Delete")');

    // Redirected to organizations list
    await expect(page).toHaveURL('/organizations');

    // Organization no longer in list
    await expect(page.locator('text=To Be Deleted')).not.toBeVisible();
  });
});

test.describe('Member Management', () => {
  test('Update member role', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');
    await page.click('text=Test Company');
    await page.click('a:has-text("Members")');

    // Find member and update role
    const memberRow = page.locator('[data-testid="member-row"]:has-text("member@test.com")');
    await memberRow.locator('button:has-text("Member")').click(); // Role dropdown
    await page.click('text=Admin');

    // Verify role updated
    await expect(memberRow.locator('text=Admin')).toBeVisible();
    await expect(page.locator('text=Role updated')).toBeVisible();
  });

  test('Remove member', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');
    await page.click('text=Test Company');
    await page.click('a:has-text("Members")');

    // Get initial member count
    const initialCount = await page.locator('[data-testid="member-row"]').count();

    // Remove a member
    const memberRow = page.locator('[data-testid="member-row"]:has-text("member@test.com")');
    await memberRow.locator('button:has-text("Remove")').click();

    // Confirm removal
    await page.click('button:has-text("Confirm")');

    // Verify member removed
    await expect(page.locator('text=Member removed')).toBeVisible();
    const newCount = await page.locator('[data-testid="member-row"]').count();
    expect(newCount).toBe(initialCount - 1);
  });
});

test.describe('Invitation Management', () => {
  test('Revoke pending invitation', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');
    await page.click('text=Test Company');
    await page.click('a:has-text("Members")');

    // Create invitation
    await page.click('button:has-text("Invite Member")');
    await page.fill('input[name="email"]', 'temp@test.com');
    await page.click('button:has-text("Send Invitation")');

    // Revoke invitation
    const invitationRow = page.locator('[data-testid="invitation-row"]:has-text("temp@test.com")');
    await invitationRow.locator('button:has-text("Revoke")').click();

    // Verify revoked
    await expect(page.locator('text=Invitation revoked')).toBeVisible();
    await expect(invitationRow).not.toBeVisible();
  });

  test('Decline invitation', async ({ page }) => {
    // Assume invitation exists
    await login(page, 'invitee@test.com', 'password123');
    await page.goto('/invitations');

    // Decline invitation
    const invitationCard = page.locator('[data-testid="invitation-card"]:has-text("Test Company")');
    await invitationCard.locator('button:has-text("Decline")').click();

    // Verify declined
    await expect(page.locator('text=Invitation declined')).toBeVisible();
    await expect(invitationCard).not.toBeVisible();
  });
});

test.describe('Error Handling', () => {
  test('Handle network errors gracefully', async ({ page }) => {
    // Simulate offline mode
    await page.context().setOffline(true);

    await login(page, 'owner@test.com', 'password123');
    await page.click('button:has-text("Create Organization")');
    await page.fill('input[name="name"]', 'Offline Test');
    await page.click('button[type="submit"]');

    // Should show error message
    await expect(page.locator('text=Network error')).toBeVisible();

    // Restore connection
    await page.context().setOffline(false);
  });

  test('Validate form inputs', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.click('button:has-text("Create Organization")');

    // Try to submit with empty name
    await page.click('button[type="submit"]');
    await expect(page.locator('text=Name is required')).toBeVisible();

    // Try with name too short
    await page.fill('input[name="name"]', 'AB');
    await page.click('button[type="submit"]');
    await expect(page.locator('text=Name must be at least 3 characters')).toBeVisible();
  });
});

// Accessibility tests
test.describe('Accessibility', () => {
  test('Organization pages are keyboard navigable', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');

    // Tab through elements
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter'); // Should open organization

    // Verify navigation worked
    await expect(page).toHaveURL(/\/organizations\/[^/]+/);
  });

  test('Screen reader labels present', async ({ page }) => {
    await login(page, 'owner@test.com', 'password123');
    await page.goto('/organizations');

    // Check for proper ARIA labels
    await expect(page.locator('[aria-label="Create new organization"]')).toBeVisible();
    await expect(page.locator('[aria-label="Organization list"]')).toBeVisible();
  });
});
