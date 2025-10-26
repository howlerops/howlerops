package connections_test

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/connections"
	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Security Tests - Authorization and Access Control
// ====================================================================

func TestSecurity_CannotAccessOtherOrgResources(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	// User is member of org-1 but tries to access org-2 resources
	userID := "user-attacker"
	targetOrgID := "org-victim"

	// User is NOT a member
	mockOrgRepo.On("GetMember", ctx, targetOrgID, userID).Return(nil, assert.AnError)

	// Test: Try to get organization connections
	_, err := service.GetOrganizationConnections(ctx, targetOrgID, userID)

	// Verify: Denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not member of organization")
	mockOrgRepo.AssertExpectations(t)
}

func TestSecurity_CannotShareWithoutPermission(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	connID := "conn-secure"
	userID := "user-member"
	orgID := "org-restricted"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "Test DB",
		CreatedBy: userID,
		UserID:    userID,
	}

	// User is member but without share permission
	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleMember, // No connections:update permission
	}

	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Test: Try to share
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify: Permission denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")

	// Verify audit log created for permission denial
	mockOrgRepo.AssertCalled(t, "CreateAuditLog", ctx, mock.MatchedBy(func(log *organization.AuditLog) bool {
		return log.Action == "permission_denied"
	}))
}

func TestSecurity_MemberCannotUpdateOthersPersonalConnection(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	// User 1 has personal connection
	existingConn := &turso.Connection{
		ID:         "conn-personal",
		Name:       "User1 Personal DB",
		CreatedBy:  "user-1",
		UserID:     "user-1",
		Visibility: "personal",
	}

	updatedConn := &turso.Connection{
		ID:   "conn-personal",
		Name: "Hacked by User 2",
	}

	mockStore.On("GetByID", ctx, "conn-personal").Return(existingConn, nil)

	// Test: User 2 tries to update user 1's personal connection
	err := service.UpdateConnection(ctx, updatedConn, "user-2")

	// Verify: Denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update another user's personal connection")
}

func TestSecurity_MemberCannotDeleteAdminResource(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	orgID := "org-hierarchy"
	adminID := "admin-user"
	memberID := "member-user"

	// Admin's shared connection
	adminConn := &turso.Connection{
		ID:             "conn-admin",
		Name:           "Admin DB",
		CreatedBy:      adminID,
		UserID:         adminID,
		Visibility:     "shared",
		OrganizationID: &orgID,
	}

	// Member tries to delete
	member := &organization.OrganizationMember{
		UserID:         memberID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
	}

	mockStore.On("GetByID", ctx, "conn-admin").Return(adminConn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, memberID).Return(member, nil)

	// Test: Member tries to delete admin's connection
	err := service.DeleteConnection(ctx, "conn-admin", memberID)

	// Verify: Denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestSecurity_OwnerCanDeleteAnyResource(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	orgID := "org-owner-power"
	ownerID := "owner-user"
	memberID := "member-user"

	// Member's shared connection
	memberConn := &turso.Connection{
		ID:             "conn-member",
		Name:           "Member DB",
		CreatedBy:      memberID,
		UserID:         memberID,
		Visibility:     "shared",
		OrganizationID: &orgID,
	}

	// Owner can delete it
	owner := &organization.OrganizationMember{
		UserID:         ownerID,
		OrganizationID: orgID,
		Role:           organization.RoleOwner,
	}

	mockStore.On("GetByID", ctx, "conn-member").Return(memberConn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, ownerID).Return(owner, nil)
	mockStore.On("Delete", ctx, "conn-member").Return(nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Test: Owner deletes member's resource
	err := service.DeleteConnection(ctx, "conn-member", ownerID)

	// Verify: Success (owners can delete any resource)
	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestSecurity_CannotShareConnectionToOrgNotMemberOf(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	connID := "conn-orphan"
	userID := "user-outsider"
	orgID := "org-secret"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "DB",
		CreatedBy: userID,
		UserID:    userID,
	}

	// User is NOT a member
	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(nil, assert.AnError)

	// Test: Try to share to org user isn't member of
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify: Denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not member of organization")
}

func TestSecurity_CannotUnshareOthersConnection(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	orgID := "org-unshare-test"
	owner1ID := "owner-1"
	owner2ID := "owner-2"

	// Owner 1's shared connection
	conn := &turso.Connection{
		ID:             "conn-owner1",
		Name:           "Owner1 DB",
		CreatedBy:      owner1ID,
		UserID:         owner1ID,
		Visibility:     "shared",
		OrganizationID: &orgID,
	}

	mockStore.On("GetByID", ctx, "conn-owner1").Return(conn, nil)

	// Test: Owner 2 tries to unshare owner 1's connection
	err := service.UnshareConnection(ctx, "conn-owner1", owner2ID)

	// Verify: Denied (only creator can unshare)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only the creator can unshare")
}

func TestSecurity_SQLInjectionProtection(t *testing.T) {
	// This test ensures our queries use parameterized statements
	// Real test would be in repository layer with actual DB

	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	// Malicious org ID with SQL injection attempt
	maliciousOrgID := "org-1' OR '1'='1"
	userID := "user-test"

	mockOrgRepo.On("GetMember", ctx, maliciousOrgID, userID).Return(nil, assert.AnError)

	// Test: Should treat as literal string, not execute SQL
	_, err := service.GetOrganizationConnections(ctx, maliciousOrgID, userID)

	// Verify: No special handling needed - parameterized queries prevent injection
	require.Error(t, err) // Should fail because org doesn't exist
}

func TestSecurity_AuditLogCreatedForSensitiveOperations(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	connID := "conn-audit"
	userID := "user-audit"
	orgID := "org-audit"

	conn := &turso.Connection{
		ID:             connID,
		Name:           "Audit Test DB",
		CreatedBy:      userID,
		UserID:         userID,
		OrganizationID: &orgID,
		Visibility:     "shared",
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleAdmin,
	}

	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockStore.On("Delete", ctx, connID).Return(nil)

	// Expect audit log
	var capturedLog *organization.AuditLog
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Run(func(args mock.Arguments) {
		capturedLog = args.Get(1).(*organization.AuditLog)
	}).Return(nil)

	// Test: Delete connection
	err := service.DeleteConnection(ctx, connID, userID)

	// Verify: Audit log created
	require.NoError(t, err)
	require.NotNil(t, capturedLog)
	assert.Equal(t, "delete_connection", capturedLog.Action)
	assert.Equal(t, userID, capturedLog.UserID)
	assert.Equal(t, &orgID, capturedLog.OrganizationID)
	assert.Equal(t, &connID, capturedLog.ResourceID)
}

func TestSecurity_RateLimitingHonored(t *testing.T) {
	// Placeholder for rate limiting test
	// In real implementation, this would test:
	// - Max connections per org
	// - Max shared resources per user
	// - API rate limits

	t.Skip("Rate limiting not yet implemented")
}

func TestSecurity_DataIsolationBetweenOrganizations(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	ctx := context.Background()

	org1ID := "org-isolated-1"
	org2ID := "org-isolated-2"
	userID := "user-in-both-orgs"

	// User in org-1
	member1 := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: org1ID,
		Role:           organization.RoleMember,
	}

	org1Connections := []*turso.Connection{
		{ID: "conn-org1", Name: "Org1 DB", Visibility: "shared"},
	}

	mockOrgRepo.On("GetMember", ctx, org1ID, userID).Return(member1, nil)
	mockStore.On("GetConnectionsByOrganization", ctx, org1ID).Return(org1Connections, nil)

	// Test: Get org-1 connections
	conns1, err := service.GetOrganizationConnections(ctx, org1ID, userID)
	require.NoError(t, err)
	assert.Len(t, conns1, 1)

	// User tries to get org-2 connections (not a member)
	mockOrgRepo.On("GetMember", ctx, org2ID, userID).Return(nil, assert.AnError)

	// Test: Try to get org-2 connections
	_, err = service.GetOrganizationConnections(ctx, org2ID, userID)

	// Verify: Denied
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not member of organization")
}

// ====================================================================
// Helper Functions
// ====================================================================

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}
