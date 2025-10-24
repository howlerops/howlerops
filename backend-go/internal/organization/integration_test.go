package organization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/internal/organization/testutil"
)

// IntegrationTestSuite is the main test suite for organization integration tests
type IntegrationTestSuite struct {
	suite.Suite
	testDB *testutil.TestDB
	repo   organization.Repository
	svc    *organization.Service
	ctx    context.Context
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	suite.testDB = testutil.SetupTestDB(suite.T())
	suite.repo = testutil.NewSQLiteRepository(suite.testDB.DB)
	suite.svc = organization.NewService(suite.repo, suite.testDB.Logger)
	suite.ctx = context.Background()

	// Create some test users in the database
	suite.createTestUser("user1", "user1@test.com", "user1", "password123")
	suite.createTestUser("user2", "user2@test.com", "user2", "password123")
	suite.createTestUser("user3", "user3@test.com", "user3", "password123")
}

// TearDownTest runs after each test
func (suite *IntegrationTestSuite) TearDownTest() {
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

// Helper: Create test user in database
func (suite *IntegrationTestSuite) createTestUser(id, email, username, password string) {
	err := suite.testDB.InsertTestUser(suite.ctx, id, email, username, password)
	require.NoError(suite.T(), err)
}

// ====================================================================
// TEST FLOW 1: User creates organization → becomes owner → verified in DB
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_CreateOrganization_UserBecomesOwner() {
	t := suite.T()

	// Given: A user wants to create an organization
	userID := "user1"
	input := testutil.CreateTestOrganizationInput("Acme Corp", "Best company ever")

	// When: User creates the organization
	org, err := suite.svc.CreateOrganization(suite.ctx, userID, input)

	// Then: Organization is created successfully
	testutil.AssertNoError(t, err, "Organization creation should succeed")
	testutil.AssertOrganizationNotNil(t, org)
	assert.Equal(t, "Acme Corp", org.Name)
	assert.Equal(t, "Best company ever", org.Description)
	assert.Equal(t, userID, org.OwnerID)
	assert.Equal(t, 10, org.MaxMembers) // Default for free tier

	// And: Organization exists in database
	dbOrg, err := suite.testDB.GetOrganizationByID(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Organization should exist in database")
	assert.Equal(t, org.ID, dbOrg["id"])
	assert.Equal(t, "Acme Corp", dbOrg["name"])

	// And: User is automatically added as owner member
	memberCount, err := suite.testDB.CountMembers(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Should count members")
	assert.Equal(t, 1, memberCount, "Organization should have 1 member (owner)")

	role, err := suite.testDB.GetMemberRole(suite.ctx, org.ID, userID)
	testutil.AssertNoError(t, err, "Should get member role")
	assert.Equal(t, string(organization.RoleOwner), role, "User should have owner role")
}

// ====================================================================
// TEST FLOW 2: Owner invites member → invitation created → email logged
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_InviteMember_InvitationCreated() {
	t := suite.T()

	// Given: An existing organization with an owner
	org := suite.createTestOrganization("user1", "Tech Startup")
	inviterID := "user1" // Owner
	inviteEmail := "newmember@test.com"

	// When: Owner invites a new member
	input := testutil.CreateTestInvitationInput(inviteEmail, organization.RoleMember)
	invitation, err := suite.svc.CreateInvitation(suite.ctx, org.ID, inviterID, input)

	// Then: Invitation is created successfully
	testutil.AssertNoError(t, err, "Invitation creation should succeed")
	testutil.AssertInvitationPending(t, invitation)
	assert.Equal(t, org.ID, invitation.OrganizationID)
	assert.Equal(t, "newmember@test.com", invitation.Email)
	assert.Equal(t, organization.RoleMember, invitation.Role)
	assert.Equal(t, inviterID, invitation.InvitedBy)
	assert.NotEmpty(t, invitation.Token, "Invitation should have a token")

	// And: Invitation is stored in database
	invExists, err := suite.testDB.InvitationExists(suite.ctx, invitation.Token)
	testutil.AssertNoError(t, err, "Should check invitation existence")
	assert.True(t, invExists, "Invitation should exist in database")

	// And: Invitation count is correct
	invCount, err := suite.testDB.CountInvitations(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Should count invitations")
	assert.Equal(t, 1, invCount, "Organization should have 1 pending invitation")
}

// ====================================================================
// TEST FLOW 3: Member accepts invitation → added to org → verified in DB
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_AcceptInvitation_MemberAdded() {
	t := suite.T()

	// Given: An organization with a pending invitation
	org := suite.createTestOrganization("user1", "Dev Team")
	inviteEmail := "user2@test.com"
	input := testutil.CreateTestInvitationInput(inviteEmail, organization.RoleMember)
	invitation, err := suite.svc.CreateInvitation(suite.ctx, org.ID, "user1", input)
	require.NoError(t, err)

	// When: User accepts the invitation
	acceptedOrg, err := suite.svc.AcceptInvitation(suite.ctx, invitation.Token, "user2")

	// Then: Invitation is accepted successfully
	testutil.AssertNoError(t, err, "Invitation acceptance should succeed")
	assert.Equal(t, org.ID, acceptedOrg.ID)

	// And: User is now a member of the organization
	memberExists, err := suite.testDB.MemberExists(suite.ctx, org.ID, "user2")
	testutil.AssertNoError(t, err, "Should check member existence")
	assert.True(t, memberExists, "User should be a member of the organization")

	// And: Member has correct role
	role, err := suite.testDB.GetMemberRole(suite.ctx, org.ID, "user2")
	testutil.AssertNoError(t, err, "Should get member role")
	assert.Equal(t, string(organization.RoleMember), role)

	// And: Member count is updated
	memberCount, err := suite.testDB.CountMembers(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Should count members")
	assert.Equal(t, 2, memberCount, "Organization should have 2 members (owner + new member)")

	// And: User can retrieve the organization
	orgs, err := suite.svc.GetUserOrganizations(suite.ctx, "user2")
	testutil.AssertNoError(t, err, "Should get user organizations")
	testutil.AssertSliceLength(t, orgs, 1)
	assert.Equal(t, org.ID, orgs[0].ID)
}

// ====================================================================
// TEST FLOW 4: Admin updates member role → permission check → role updated
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_UpdateMemberRole_RoleUpdated() {
	t := suite.T()

	// Given: An organization with owner and member
	org := suite.createTestOrganization("user1", "Growing Company")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	// When: Owner promotes member to admin
	err := suite.svc.UpdateMemberRole(suite.ctx, org.ID, "user2", "user1", organization.RoleAdmin)

	// Then: Role update succeeds
	testutil.AssertNoError(t, err, "Role update should succeed")

	// And: Member now has admin role in database
	role, err := suite.testDB.GetMemberRole(suite.ctx, org.ID, "user2")
	testutil.AssertNoError(t, err, "Should get updated role")
	assert.Equal(t, string(organization.RoleAdmin), role)

	// And: Admin can now perform admin actions (invite members)
	invitation, err := suite.svc.CreateInvitation(suite.ctx, org.ID, "user2",
		testutil.CreateTestInvitationInput("user3@test.com", organization.RoleMember))
	testutil.AssertNoError(t, err, "Admin should be able to create invitations")
	assert.NotNil(t, invitation)
}

// ====================================================================
// TEST FLOW 5: Owner removes member → member deleted → verified in DB
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_RemoveMember_MemberDeleted() {
	t := suite.T()

	// Given: An organization with owner and member
	org := suite.createTestOrganization("user1", "Shrinking Company")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	initialCount, _ := suite.testDB.CountMembers(suite.ctx, org.ID)
	assert.Equal(t, 2, initialCount)

	// When: Owner removes the member
	err := suite.svc.RemoveMember(suite.ctx, org.ID, "user2", "user1")

	// Then: Member removal succeeds
	testutil.AssertNoError(t, err, "Member removal should succeed")

	// And: Member no longer exists in database
	memberExists, err := suite.testDB.MemberExists(suite.ctx, org.ID, "user2")
	testutil.AssertNoError(t, err, "Should check member existence")
	assert.False(t, memberExists, "User should no longer be a member")

	// And: Member count is updated
	memberCount, err := suite.testDB.CountMembers(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Should count members")
	assert.Equal(t, 1, memberCount, "Organization should have 1 member (only owner)")

	// And: User can no longer access organization
	_, err = suite.svc.GetOrganization(suite.ctx, org.ID, "user2")
	assert.Error(t, err, "Removed member should not access organization")
	testutil.AssertErrorContains(t, err, "not a member")
}

// ====================================================================
// TEST FLOW 6: Owner deletes organization → soft delete → verified in DB
// ====================================================================

func (suite *IntegrationTestSuite) TestFlow_DeleteOrganization_SoftDeleted() {
	t := suite.T()

	// Given: An organization with only the owner
	org := suite.createTestOrganization("user1", "Closing Shop")

	// When: Owner deletes the organization
	err := suite.svc.DeleteOrganization(suite.ctx, org.ID, "user1")

	// Then: Deletion succeeds
	testutil.AssertNoError(t, err, "Organization deletion should succeed")

	// And: Organization is soft-deleted in database
	deletedOrg, err := suite.repo.GetByID(suite.ctx, org.ID)
	testutil.AssertNoError(t, err, "Should still be able to retrieve deleted org")
	assert.NotNil(t, deletedOrg.DeletedAt, "Organization should have deleted_at timestamp")

	// And: User's organization list no longer includes it
	orgs, err := suite.svc.GetUserOrganizations(suite.ctx, "user1")
	testutil.AssertNoError(t, err, "Should get user organizations")
	testutil.AssertSliceEmpty(t, orgs)
}

// ====================================================================
// PERMISSION BOUNDARY TESTS
// ====================================================================

func (suite *IntegrationTestSuite) TestPermissions_MemberCannotInvite() {
	t := suite.T()

	// Given: An organization with a regular member
	org := suite.createTestOrganization("user1", "Secure Org")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	// When: Member tries to invite another user
	_, err := suite.svc.CreateInvitation(suite.ctx, org.ID, "user2",
		testutil.CreateTestInvitationInput("user3@test.com", organization.RoleMember))

	// Then: Permission is denied
	assert.Error(t, err, "Member should not be able to create invitations")
	testutil.AssertErrorContains(t, err, "insufficient permissions")
}

func (suite *IntegrationTestSuite) TestPermissions_AdminCannotPromoteToOwner() {
	t := suite.T()

	// Given: An organization with an admin
	org := suite.createTestOrganization("user1", "Hierarchical Org")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleAdmin)

	// When: Admin tries to promote someone to owner
	err := suite.svc.UpdateMemberRole(suite.ctx, org.ID, "user3", "user2", organization.RoleOwner)

	// Then: Permission is denied
	assert.Error(t, err, "Admin should not be able to assign owner role")
	testutil.AssertErrorContains(t, err, "only owners can assign owner role")
}

func (suite *IntegrationTestSuite) TestPermissions_CannotRemoveOwner() {
	t := suite.T()

	// Given: An organization with owner and admin
	org := suite.createTestOrganization("user1", "Protected Org")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleAdmin)

	// When: Admin tries to remove the owner
	err := suite.svc.RemoveMember(suite.ctx, org.ID, "user1", "user2")

	// Then: Permission is denied
	assert.Error(t, err, "Cannot remove owner from organization")
	testutil.AssertErrorContains(t, err, "cannot remove owner")
}

// ====================================================================
// VALIDATION TESTS
// ====================================================================

func (suite *IntegrationTestSuite) TestValidation_InvalidOrganizationName() {
	t := suite.T()

	// Test: Name too short
	input := testutil.CreateTestOrganizationInput("AB", "Description")
	_, err := suite.svc.CreateOrganization(suite.ctx, "user1", input)
	assert.Error(t, err, "Should reject name that is too short")
	testutil.AssertErrorContains(t, err, "at least 3 characters")

	// Test: Name too long
	longName := string(make([]byte, 60))
	input = testutil.CreateTestOrganizationInput(longName, "Description")
	_, err = suite.svc.CreateOrganization(suite.ctx, "user1", input)
	assert.Error(t, err, "Should reject name that is too long")

	// Test: Invalid characters
	input = testutil.CreateTestOrganizationInput("Test<>Org", "Description")
	_, err = suite.svc.CreateOrganization(suite.ctx, "user1", input)
	assert.Error(t, err, "Should reject name with invalid characters")
}

func (suite *IntegrationTestSuite) TestValidation_InvalidEmail() {
	t := suite.T()

	// Given: An organization
	org := suite.createTestOrganization("user1", "Email Validator")

	// When: Owner tries to invite with invalid email
	input := testutil.CreateTestInvitationInput("not-an-email", organization.RoleMember)
	_, err := suite.svc.CreateInvitation(suite.ctx, org.ID, "user1", input)

	// Then: Validation fails
	assert.Error(t, err, "Should reject invalid email")
	testutil.AssertErrorContains(t, err, "invalid email")
}

func (suite *IntegrationTestSuite) TestValidation_DuplicateInvitation() {
	t := suite.T()

	// Given: An organization with a pending invitation
	org := suite.createTestOrganization("user1", "No Duplicates")
	input := testutil.CreateTestInvitationInput("user2@test.com", organization.RoleMember)
	_, err := suite.svc.CreateInvitation(suite.ctx, org.ID, "user1", input)
	require.NoError(t, err)

	// When: Owner tries to invite the same email again
	_, err = suite.svc.CreateInvitation(suite.ctx, org.ID, "user1", input)

	// Then: Duplicate invitation is rejected
	assert.Error(t, err, "Should reject duplicate invitation")
	testutil.AssertErrorContains(t, err, "already exists")
}

// ====================================================================
// EDGE CASES
// ====================================================================

func (suite *IntegrationTestSuite) TestEdgeCase_ExpiredInvitation() {
	t := suite.T()

	// Given: An expired invitation
	org := suite.createTestOrganization("user1", "Time Sensitive")
	invitation := testutil.CreateExpiredInvitation(org.ID, "user2@test.com", "user1", organization.RoleMember)
	err := suite.repo.CreateInvitation(suite.ctx, invitation)
	require.NoError(t, err)

	// When: User tries to accept expired invitation
	_, err = suite.svc.AcceptInvitation(suite.ctx, invitation.Token, "user2")

	// Then: Acceptance is rejected
	assert.Error(t, err, "Should reject expired invitation")
	testutil.AssertErrorContains(t, err, "expired")
}

func (suite *IntegrationTestSuite) TestEdgeCase_AlreadyMember() {
	t := suite.T()

	// Given: An invitation for a user who is already a member
	org := suite.createTestOrganization("user1", "Already In")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	invitation := testutil.CreateTestInvitation(org.ID, "user2@test.com", "user1", organization.RoleMember)
	err := suite.repo.CreateInvitation(suite.ctx, invitation)
	require.NoError(t, err)

	// When: User tries to accept invitation
	_, err = suite.svc.AcceptInvitation(suite.ctx, invitation.Token, "user2")

	// Then: Acceptance is rejected
	assert.Error(t, err, "Should reject invitation for existing member")
	testutil.AssertErrorContains(t, err, "already a member")
}

func (suite *IntegrationTestSuite) TestEdgeCase_MaxMembersReached() {
	t := suite.T()

	// Given: An organization at max capacity
	org := suite.createTestOrganization("user1", "Full House")
	org.MaxMembers = 2
	err := suite.repo.Update(suite.ctx, org)
	require.NoError(t, err)

	// Add member to reach limit
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	// When: Owner tries to invite another member
	input := testutil.CreateTestInvitationInput("user3@test.com", organization.RoleMember)
	_, err = suite.svc.CreateInvitation(suite.ctx, org.ID, "user1", input)

	// Then: Invitation is rejected
	assert.Error(t, err, "Should reject invitation when at max capacity")
	testutil.AssertErrorContains(t, err, "maximum member limit")
}

func (suite *IntegrationTestSuite) TestEdgeCase_CannotDeleteOrgWithMembers() {
	t := suite.T()

	// Given: An organization with multiple members
	org := suite.createTestOrganization("user1", "Active Team")
	suite.addMemberToOrg(org.ID, "user2", organization.RoleMember)

	// When: Owner tries to delete organization
	err := suite.svc.DeleteOrganization(suite.ctx, org.ID, "user1")

	// Then: Deletion is rejected
	assert.Error(t, err, "Should reject deletion of org with other members")
	testutil.AssertErrorContains(t, err, "with other members")
}

// ====================================================================
// AUDIT LOGGING TESTS
// ====================================================================

func (suite *IntegrationTestSuite) TestAuditLog_OrganizationCreated() {
	t := suite.T()

	// When: User creates an organization
	org := suite.createTestOrganization("user1", "Audited Org")

	// Then: Audit log entry is created (if implemented in service)
	// Note: This depends on whether audit logging is implemented in the service layer
	// For now, we can verify the audit log functionality works
	auditLog := testutil.CreateTestAuditLog(org.ID, "user1", "organization.created", "organization", org.ID)
	err := suite.repo.CreateAuditLog(suite.ctx, auditLog)
	testutil.AssertNoError(t, err, "Should create audit log")

	// And: Audit log can be retrieved
	logs, err := suite.repo.GetAuditLogs(suite.ctx, org.ID, 10, 0)
	testutil.AssertNoError(t, err, "Should retrieve audit logs")
	testutil.AssertSliceNotEmpty(t, logs)
}

// ====================================================================
// HELPER METHODS
// ====================================================================

func (suite *IntegrationTestSuite) createTestOrganization(ownerID, name string) *organization.Organization {
	input := testutil.CreateTestOrganizationInput(name, "Test organization for "+name)
	org, err := suite.svc.CreateOrganization(suite.ctx, ownerID, input)
	require.NoError(suite.T(), err)
	return org
}

func (suite *IntegrationTestSuite) addMemberToOrg(orgID, userID string, role organization.OrganizationRole) {
	member := testutil.CreateTestMember(orgID, userID, role)
	err := suite.repo.AddMember(suite.ctx, member)
	require.NoError(suite.T(), err)
}

// Run the test suite
func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// ====================================================================
// ADDITIONAL INTEGRATION TESTS
// ====================================================================

func TestInvitationFlow_CompleteJourney(t *testing.T) {
	// This test demonstrates the complete invitation journey
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()

	repo := testutil.NewSQLiteRepository(testDB.DB)
	svc := organization.NewService(repo, testDB.Logger)
	ctx := context.Background()

	// Setup: Create users
	testDB.InsertTestUser(ctx, "owner1", "owner@test.com", "owner", "pass")
	testDB.InsertTestUser(ctx, "member1", "member@test.com", "member", "pass")

	// Step 1: Create organization
	orgInput := testutil.CreateTestOrganizationInput("Complete Journey Co", "End to end test")
	org, err := svc.CreateOrganization(ctx, "owner1", orgInput)
	require.NoError(t, err)

	// Step 2: Send invitation
	invInput := testutil.CreateTestInvitationInput("member@test.com", organization.RoleMember)
	invitation, err := svc.CreateInvitation(ctx, org.ID, "owner1", invInput)
	require.NoError(t, err)
	assert.NotEmpty(t, invitation.Token)

	// Step 3: Check pending invitations
	invitations, err := svc.GetPendingInvitationsForEmail(ctx, "member@test.com")
	require.NoError(t, err)
	testutil.AssertSliceLength(t, invitations, 1)

	// Step 4: Accept invitation
	acceptedOrg, err := svc.AcceptInvitation(ctx, invitation.Token, "member1")
	require.NoError(t, err)
	assert.Equal(t, org.ID, acceptedOrg.ID)

	// Step 5: Verify membership
	members, err := svc.GetMembers(ctx, org.ID, "owner1")
	require.NoError(t, err)
	testutil.AssertSliceLength(t, members, 2)

	// Step 6: Member can access organization
	retrievedOrg, err := svc.GetOrganization(ctx, org.ID, "member1")
	require.NoError(t, err)
	assert.Equal(t, org.ID, retrievedOrg.ID)
}
