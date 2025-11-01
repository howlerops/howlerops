import { test, expect, Page } from '@playwright/test'

/**
 * RBAC Permission System E2E Tests
 * Tests role-based access control across Owner, Admin, and Member roles
 */

// Test data setup
const TEST_ORG = {
  name: 'Test Organization',
  description: 'Organization for E2E permission testing'
}

const TEST_USERS = {
  owner: {
    email: 'owner@test.com',
    password: 'OwnerPass123!',
    role: 'owner'
  },
  admin: {
    email: 'admin@test.com',
    password: 'AdminPass123!',
    role: 'admin'
  },
  member: {
    email: 'member@test.com',
    password: 'MemberPass123!',
    role: 'member'
  },
  nonMember: {
    email: 'nonmember@test.com',
    password: 'NonMemberPass123!'
  }
}

// Helper functions
async function loginAs(page: Page, user: typeof TEST_USERS.owner) {
  await page.goto('/login')
  await page.fill('[data-testid="email-input"]', user.email)
  await page.fill('[data-testid="password-input"]', user.password)
  await page.click('[data-testid="login-button"]')
  await page.waitForURL('/dashboard')
}

async function navigateToOrganization(page: Page, orgName: string) {
  await page.click(`[data-testid="org-${orgName}"]`)
  await page.waitForSelector('[data-testid="org-header"]')
}

async function navigateToMembers(page: Page) {
  await page.click('[data-testid="members-tab"]')
  await page.waitForSelector('[data-testid="members-list"]')
}

async function navigateToSettings(page: Page) {
  await page.click('[data-testid="settings-tab"]')
  await page.waitForSelector('[data-testid="org-settings"]')
}

async function makeAPICall(page: Page, method: string, endpoint: string, body?: Record<string, unknown>) {
  return await page.evaluate(async ({ method, endpoint, body }) => {
    const token = localStorage.getItem('auth_token')
    const response = await fetch(endpoint, {
      method,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: body ? JSON.stringify(body) : undefined
    })
    return {
      status: response.status,
      data: await response.json().catch(() => null)
    }
  }, { method, endpoint, body })
}

test.describe('RBAC Permission System', () => {
  test.beforeEach(async ({ page: _page }) => {
    // Setup test data if needed
    // This could involve API calls to create test organization and users
  })

  test.describe('Owner Permissions', () => {
    test('Owner can perform all actions', async ({ page }) => {
      await loginAs(page, TEST_USERS.owner)
      await navigateToOrganization(page, TEST_ORG.name)

      // Test: Can update organization
      await navigateToSettings(page)
      const updateOrgButton = page.locator('[data-testid="update-org-button"]')
      await expect(updateOrgButton).toBeVisible()
      await expect(updateOrgButton).toBeEnabled()

      // Test: Can delete organization
      const deleteOrgButton = page.locator('[data-testid="delete-org-button"]')
      await expect(deleteOrgButton).toBeVisible()
      await expect(deleteOrgButton).toBeEnabled()

      // Test: Can invite members
      await navigateToMembers(page)
      const inviteButton = page.locator('[data-testid="invite-member-button"]')
      await expect(inviteButton).toBeVisible()
      await expect(inviteButton).toBeEnabled()

      // Test: Can remove members
      const removeMemberButtons = page.locator('[data-testid^="remove-member-"]')
      const count = await removeMemberButtons.count()
      if (count > 0) {
        await expect(removeMemberButtons.first()).toBeEnabled()
      }

      // Test: Can change member roles
      const roleDropdowns = page.locator('[data-testid^="role-dropdown-"]')
      if (await roleDropdowns.count() > 0) {
        await expect(roleDropdowns.first()).toBeEnabled()

        // Verify owner role is available in dropdown
        await roleDropdowns.first().click()
        const ownerOption = page.locator('[data-testid="role-option-owner"]')
        await expect(ownerOption).toBeVisible()
      }

      // Test: Can view audit logs
      await page.click('[data-testid="audit-logs-tab"]')
      await expect(page.locator('[data-testid="audit-logs-list"]')).toBeVisible()
    })

    test('Owner can transfer ownership', async ({ page }) => {
      await loginAs(page, TEST_USERS.owner)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Find an admin user to transfer ownership to
      const adminRow = page.locator('[data-testid="member-row-admin"]').first()
      if (await adminRow.count() > 0) {
        const transferButton = adminRow.locator('[data-testid="transfer-ownership-button"]')
        await expect(transferButton).toBeVisible()
        await expect(transferButton).toBeEnabled()
      }
    })
  })

  test.describe('Admin Permissions', () => {
    test('Admin cannot delete organization', async ({ page }) => {
      await loginAs(page, TEST_USERS.admin)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToSettings(page)

      // Test: Delete button should be hidden or disabled
      const deleteOrgButton = page.locator('[data-testid="delete-org-button"]')
      const isVisible = await deleteOrgButton.isVisible().catch(() => false)

      if (isVisible) {
        await expect(deleteOrgButton).toBeDisabled()

        // Verify tooltip explains why
        await deleteOrgButton.hover()
        const tooltip = page.locator('[role="tooltip"]')
        await expect(tooltip).toContainText(/only.*owner/i)
      }

      // Test: API call should fail with 403
      const response = await makeAPICall(page, 'DELETE', `/api/organizations/${TEST_ORG.name}`)
      expect(response.status).toBe(403)
    })

    test('Admin cannot promote to owner', async ({ page }) => {
      await loginAs(page, TEST_USERS.admin)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Find a member to try to promote
      const memberRow = page.locator('[data-testid="member-row-member"]').first()
      if (await memberRow.count() > 0) {
        const roleDropdown = memberRow.locator('[data-testid^="role-dropdown-"]')
        await roleDropdown.click()

        // Owner option should not be available
        const ownerOption = page.locator('[data-testid="role-option-owner"]')
        await expect(ownerOption).not.toBeVisible()

        // Admin option should be available
        const adminOption = page.locator('[data-testid="role-option-admin"]')
        await expect(adminOption).toBeVisible()
      }

      // Test: API call should fail
      const response = await makeAPICall(page, 'PUT', `/api/organizations/${TEST_ORG.name}/members/test-user`, {
        role: 'owner'
      })
      expect(response.status).toBe(400)
    })

    test('Admin can invite members but not admins', async ({ page }) => {
      await loginAs(page, TEST_USERS.admin)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Click invite button
      await page.click('[data-testid="invite-member-button"]')

      // Check available roles in invite modal
      const roleSelect = page.locator('[data-testid="invite-role-select"]')
      await roleSelect.click()

      // Member role should be available
      await expect(page.locator('[data-testid="role-option-member"]')).toBeVisible()

      // Admin role should not be available for admin users
      await expect(page.locator('[data-testid="role-option-admin"]')).not.toBeVisible()
    })

    test('Admin can view audit logs', async ({ page }) => {
      await loginAs(page, TEST_USERS.admin)
      await navigateToOrganization(page, TEST_ORG.name)

      // Audit logs tab should be visible and accessible
      const auditTab = page.locator('[data-testid="audit-logs-tab"]')
      await expect(auditTab).toBeVisible()
      await auditTab.click()
      await expect(page.locator('[data-testid="audit-logs-list"]')).toBeVisible()
    })
  })

  test.describe('Member Permissions', () => {
    test('Member cannot invite other members', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Invite button should be hidden or disabled
      const inviteButton = page.locator('[data-testid="invite-member-button"]')
      const isVisible = await inviteButton.isVisible().catch(() => false)

      if (isVisible) {
        await expect(inviteButton).toBeDisabled()

        // Verify tooltip explains why
        await inviteButton.hover()
        const tooltip = page.locator('[role="tooltip"]')
        await expect(tooltip).toContainText(/permission/i)
      }

      // Test: API call should fail with 403
      const response = await makeAPICall(page, 'POST', `/api/organizations/${TEST_ORG.name}/invitations`, {
        email: 'newuser@test.com',
        role: 'member'
      })
      expect(response.status).toBe(403)
    })

    test('Member cannot remove other members', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Remove buttons should be hidden or disabled
      const removeMemberButtons = page.locator('[data-testid^="remove-member-"]')
      const count = await removeMemberButtons.count()

      for (let i = 0; i < count; i++) {
        const button = removeMemberButtons.nth(i)
        const isVisible = await button.isVisible().catch(() => false)

        if (isVisible) {
          await expect(button).toBeDisabled()
        }
      }

      // Test: API call should fail with 403
      const response = await makeAPICall(page, 'DELETE', `/api/organizations/${TEST_ORG.name}/members/test-user`)
      expect(response.status).toBe(403)
    })

    test('Member cannot change roles', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Role dropdowns should be disabled or hidden
      const roleDropdowns = page.locator('[data-testid^="role-dropdown-"]')
      const count = await roleDropdowns.count()

      for (let i = 0; i < count; i++) {
        const dropdown = roleDropdowns.nth(i)
        const isVisible = await dropdown.isVisible().catch(() => false)

        if (isVisible) {
          await expect(dropdown).toBeDisabled()
        }
      }

      // Test: API call should fail with 403
      const response = await makeAPICall(page, 'PUT', `/api/organizations/${TEST_ORG.name}/members/test-user`, {
        role: 'admin'
      })
      expect(response.status).toBe(403)
    })

    test('Member cannot update organization settings', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToSettings(page)

      // Update button should be disabled or hidden
      const updateButton = page.locator('[data-testid="update-org-button"]')
      const isVisible = await updateButton.isVisible().catch(() => false)

      if (isVisible) {
        await expect(updateButton).toBeDisabled()
      }

      // Form fields should be read-only
      const nameInput = page.locator('[data-testid="org-name-input"]')
      await expect(nameInput).toHaveAttribute('readonly', '')
    })

    test('Member cannot view audit logs', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)

      // Audit logs tab should be hidden
      const auditTab = page.locator('[data-testid="audit-logs-tab"]')
      await expect(auditTab).not.toBeVisible()

      // Direct API call should fail
      const response = await makeAPICall(page, 'GET', `/api/organizations/${TEST_ORG.name}/audit-logs`)
      expect(response.status).toBe(403)
    })

    test('Member can view organization and members', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)

      // Should be able to view organization details
      await expect(page.locator('[data-testid="org-header"]')).toBeVisible()

      // Should be able to view members list
      await navigateToMembers(page)
      await expect(page.locator('[data-testid="members-list"]')).toBeVisible()
    })
  })

  test.describe('Non-Member Access Control', () => {
    test('Non-member cannot access organization', async ({ page }) => {
      await loginAs(page, TEST_USERS.nonMember)

      // Organization should not appear in dashboard
      const orgCard = page.locator(`[data-testid="org-${TEST_ORG.name}"]`)
      await expect(orgCard).not.toBeVisible()

      // Direct navigation should redirect or show error
      await page.goto(`/organizations/${TEST_ORG.name}`)

      // Should either redirect to dashboard or show access denied
      const url = page.url()
      const isRedirected = url.includes('/dashboard') || url.includes('/login')
      const hasErrorMessage = await page.locator('[data-testid="access-denied"]').isVisible().catch(() => false)

      expect(isRedirected || hasErrorMessage).toBeTruthy()

      // API call should fail with 403
      const response = await makeAPICall(page, 'GET', `/api/organizations/${TEST_ORG.name}`)
      expect(response.status).toBe(403)
    })

    test('Non-member cannot view organization members', async ({ page }) => {
      await loginAs(page, TEST_USERS.nonMember)

      // Direct API call should fail
      const response = await makeAPICall(page, 'GET', `/api/organizations/${TEST_ORG.name}/members`)
      expect(response.status).toBe(403)
    })

    test('Non-member cannot create invitations', async ({ page }) => {
      await loginAs(page, TEST_USERS.nonMember)

      // API call should fail with 403
      const response = await makeAPICall(page, 'POST', `/api/organizations/${TEST_ORG.name}/invitations`, {
        email: 'hacker@test.com',
        role: 'owner'
      })
      expect(response.status).toBe(403)
    })
  })

  test.describe('Permission Error Messages', () => {
    test('Permission denied shows helpful message', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Try to click disabled invite button
      const inviteButton = page.locator('[data-testid="invite-member-button"]')
      if (await inviteButton.isVisible()) {
        await inviteButton.hover()

        // Tooltip should explain permission issue
        const tooltip = page.locator('[role="tooltip"]')
        await expect(tooltip).toBeVisible()
        await expect(tooltip).toContainText(/permission|authorized|owner|admin/i)
      }
    })

    test('API error messages are user-friendly', async ({ page }) => {
      await loginAs(page, TEST_USERS.member)

      // Make forbidden API call
      const response = await makeAPICall(page, 'DELETE', `/api/organizations/${TEST_ORG.name}`)

      expect(response.status).toBe(403)
      if (response.data && response.data.error) {
        expect(response.data.error).toMatch(/permission|authorized|forbidden/i)
        // Should not contain technical details
        expect(response.data.error).not.toContain('stack')
        expect(response.data.error).not.toContain('sql')
      }
    })
  })

  test.describe('Real-time Permission Updates', () => {
    test('Role changes are reflected immediately', async ({ browser }) => {
      // Create two browser contexts for owner and member
      const ownerContext = await browser.newContext()
      const memberContext = await browser.newContext()

      const ownerPage = await ownerContext.newPage()
      const memberPage = await memberContext.newPage()

      // Login as owner and member in separate contexts
      await loginAs(ownerPage, TEST_USERS.owner)
      await loginAs(memberPage, TEST_USERS.member)

      // Both navigate to the same organization
      await navigateToOrganization(ownerPage, TEST_ORG.name)
      await navigateToOrganization(memberPage, TEST_ORG.name)

      // Member navigates to members page
      await navigateToMembers(memberPage)

      // Verify member cannot see admin controls initially
      const inviteButton = memberPage.locator('[data-testid="invite-member-button"]')
      await expect(inviteButton).toBeDisabled()

      // Owner promotes member to admin
      await navigateToMembers(ownerPage)
      const memberRow = ownerPage.locator(`[data-testid="member-${TEST_USERS.member.email}"]`)
      const roleDropdown = memberRow.locator('[data-testid^="role-dropdown-"]')
      await roleDropdown.click()
      await ownerPage.click('[data-testid="role-option-admin"]')

      // Wait for update to complete
      await ownerPage.waitForResponse(resp => resp.url().includes('/members') && resp.status() === 200)

      // Member page should reflect new permissions (may need to refresh or poll)
      await memberPage.reload()
      await navigateToMembers(memberPage)

      // Now member (promoted to admin) should see enabled invite button
      const updatedInviteButton = memberPage.locator('[data-testid="invite-member-button"]')
      await expect(updatedInviteButton).toBeEnabled()

      // Cleanup
      await ownerContext.close()
      await memberContext.close()
    })

    test('Removed member loses access immediately', async ({ browser }) => {
      // Create two browser contexts
      const ownerContext = await browser.newContext()
      const memberContext = await browser.newContext()

      const ownerPage = await ownerContext.newPage()
      const memberPage = await memberContext.newPage()

      // Login as owner and member
      await loginAs(ownerPage, TEST_USERS.owner)
      await loginAs(memberPage, TEST_USERS.member)

      // Both navigate to organization
      await navigateToOrganization(ownerPage, TEST_ORG.name)
      await navigateToOrganization(memberPage, TEST_ORG.name)

      // Member should have access initially
      await expect(memberPage.locator('[data-testid="org-header"]')).toBeVisible()

      // Owner removes member
      await navigateToMembers(ownerPage)
      const memberRow = ownerPage.locator(`[data-testid="member-${TEST_USERS.member.email}"]`)
      await memberRow.locator('[data-testid^="remove-member-"]').click()

      // Confirm removal
      await ownerPage.click('[data-testid="confirm-remove-button"]')
      await ownerPage.waitForResponse(resp => resp.url().includes('/members') && resp.status() === 200)

      // Member should lose access (page should redirect or show error)
      await memberPage.waitForTimeout(1000) // Give time for websocket update if implemented

      // Try to navigate or refresh
      await memberPage.reload()

      // Should be redirected away or show access denied
      const url = memberPage.url()
      const isRedirected = url.includes('/dashboard') || url.includes('/login')
      const hasErrorMessage = await memberPage.locator('[data-testid="access-denied"]').isVisible().catch(() => false)

      expect(isRedirected || hasErrorMessage).toBeTruthy()

      // Cleanup
      await ownerContext.close()
      await memberContext.close()
    })
  })

  test.describe('Invitation Security', () => {
    test('Expired invitation cannot be accepted', async ({ page }) => {
      // This would require setting up an expired invitation in test data
      const expiredToken = 'expired-test-token'

      await page.goto(`/invitations/accept?token=${expiredToken}`)

      // Should show error message
      await expect(page.locator('[data-testid="invitation-expired"]')).toBeVisible()

      // API call should fail
      const response = await makeAPICall(page, 'POST', `/api/invitations/${expiredToken}/accept`)
      expect(response.status).toBe(400)
    })

    test('Already accepted invitation cannot be reused', async ({ page }) => {
      const usedToken = 'already-used-token'

      await page.goto(`/invitations/accept?token=${usedToken}`)

      // Should show error message
      await expect(page.locator('[data-testid="invitation-already-used"]')).toBeVisible()

      // API call should fail
      const response = await makeAPICall(page, 'POST', `/api/invitations/${usedToken}/accept`)
      expect(response.status).toBe(400)
    })

    test('Invalid invitation token shows error', async ({ page }) => {
      const invalidToken = 'completely-invalid-token-12345'

      await page.goto(`/invitations/accept?token=${invalidToken}`)

      // Should show error message
      await expect(page.locator('[data-testid="invitation-invalid"]')).toBeVisible()

      // API call should fail
      const response = await makeAPICall(page, 'POST', `/api/invitations/${invalidToken}/accept`)
      expect([400, 404]).toContain(response.status)
    })
  })

  test.describe('Organization Limits', () => {
    test('Cannot invite when at member limit', async ({ page }) => {
      // This test assumes the organization is at its member limit
      await loginAs(page, TEST_USERS.owner)
      await navigateToOrganization(page, TEST_ORG.name)
      await navigateToMembers(page)

      // Check if at limit (this would need to be set up in test data)
      const memberCount = await page.locator('[data-testid="member-count"]').textContent()
      const maxMembers = await page.locator('[data-testid="max-members"]').textContent()

      if (memberCount === maxMembers) {
        // Invite button should show limit reached
        const inviteButton = page.locator('[data-testid="invite-member-button"]')
        await inviteButton.hover()

        const tooltip = page.locator('[role="tooltip"]')
        await expect(tooltip).toContainText(/limit|maximum/i)

        // API call should fail
        const response = await makeAPICall(page, 'POST', `/api/organizations/${TEST_ORG.name}/invitations`, {
          email: 'newuser@test.com',
          role: 'member'
        })
        expect(response.status).toBe(400)
      }
    })

    test('Rate limiting prevents invitation spam', async ({ page }) => {
      await loginAs(page, TEST_USERS.owner)
      await navigateToOrganization(page, TEST_ORG.name)

      // Try to send many invitations rapidly
      const responses = []
      for (let i = 0; i < 25; i++) {
        const response = await makeAPICall(page, 'POST', `/api/organizations/${TEST_ORG.name}/invitations`, {
          email: `test${i}@example.com`,
          role: 'member'
        })
        responses.push(response.status)

        // If we hit rate limit, stop
        if (response.status === 429) {
          break
        }
      }

      // Should have hit rate limit (429) at some point
      expect(responses).toContain(429)
    })
  })
})
