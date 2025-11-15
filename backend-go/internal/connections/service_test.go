package connections

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/internal/organization"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ====================================================================
// Mock Implementations
// ====================================================================

type MockConnectionStore struct {
	mock.Mock
}

func (m *MockConnectionStore) Create(ctx context.Context, conn *turso.Connection) error {
	args := m.Called(ctx, conn)
	return args.Error(0)
}

func (m *MockConnectionStore) GetByID(ctx context.Context, id string) (*turso.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*turso.Connection), args.Error(1)
}

func (m *MockConnectionStore) GetByUserID(ctx context.Context, userID string) ([]*turso.Connection, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*turso.Connection), args.Error(1)
}

func (m *MockConnectionStore) GetConnectionsByOrganization(ctx context.Context, orgID string) ([]*turso.Connection, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*turso.Connection), args.Error(1)
}

func (m *MockConnectionStore) GetSharedConnections(ctx context.Context, userID string) ([]*turso.Connection, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*turso.Connection), args.Error(1)
}

func (m *MockConnectionStore) Update(ctx context.Context, conn *turso.Connection) error {
	args := m.Called(ctx, conn)
	return args.Error(0)
}

func (m *MockConnectionStore) UpdateConnectionVisibility(ctx context.Context, connID, userID string, visibility string) error {
	args := m.Called(ctx, connID, userID, visibility)
	return args.Error(0)
}

func (m *MockConnectionStore) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockOrgRepository struct {
	mock.Mock
}

func (m *MockOrgRepository) GetMember(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*organization.OrganizationMember), args.Error(1)
}

func (m *MockOrgRepository) CreateAuditLog(ctx context.Context, log *organization.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

// Implement other required methods with no-ops for testing
func (m *MockOrgRepository) Create(ctx context.Context, org *organization.Organization) error {
	return nil
}
func (m *MockOrgRepository) GetByID(ctx context.Context, id string) (*organization.Organization, error) {
	return nil, nil
}
func (m *MockOrgRepository) GetByUserID(ctx context.Context, userID string) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *MockOrgRepository) Update(ctx context.Context, org *organization.Organization) error {
	return nil
}
func (m *MockOrgRepository) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *MockOrgRepository) AddMember(ctx context.Context, member *organization.OrganizationMember) error {
	return nil
}
func (m *MockOrgRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	return nil
}
func (m *MockOrgRepository) GetMembers(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
	return nil, nil
}
func (m *MockOrgRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error {
	return nil
}
func (m *MockOrgRepository) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}
func (m *MockOrgRepository) CreateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	return nil
}
func (m *MockOrgRepository) GetInvitation(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
	return nil, nil
}
func (m *MockOrgRepository) GetInvitationByToken(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
	return nil, nil
}
func (m *MockOrgRepository) GetInvitationsByOrg(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
	return nil, nil
}
func (m *MockOrgRepository) GetInvitationsByEmail(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
	return nil, nil
}
func (m *MockOrgRepository) UpdateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	return nil
}
func (m *MockOrgRepository) DeleteInvitation(ctx context.Context, id string) error {
	return nil
}
func (m *MockOrgRepository) GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error) {
	return nil, nil
}

// ====================================================================
// Service Tests - ShareConnection
// ====================================================================

func TestShareConnection_Success(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	connID := "conn-1"
	userID := "user-1"
	orgID := "org-1"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "Test DB",
		CreatedBy: userID,
		UserID:    userID,
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleAdmin,
	}

	// Setup expectations
	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockStore.On("Update", ctx, mock.MatchedBy(func(c *turso.Connection) bool {
		return c.ID == connID && c.Visibility == "shared" && *c.OrganizationID == orgID
	})).Return(nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Execute
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify
	require.NoError(t, err)
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestShareConnection_NotCreator(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	connID := "conn-1"
	userID := "user-2"
	orgID := "org-1"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "Test DB",
		CreatedBy: "user-1", // Different creator
		UserID:    "user-1",
	}

	mockStore.On("GetByID", ctx, connID).Return(conn, nil)

	// Execute
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only the creator can share")
	mockStore.AssertExpectations(t)
}

func TestShareConnection_InsufficientPermissions(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	connID := "conn-1"
	userID := "user-1"
	orgID := "org-1"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "Test DB",
		CreatedBy: userID,
		UserID:    userID,
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleMember, // Member doesn't have update permission
	}

	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Execute
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestShareConnection_NotMember(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	connID := "conn-1"
	userID := "user-1"
	orgID := "org-1"

	conn := &turso.Connection{
		ID:        connID,
		Name:      "Test DB",
		CreatedBy: userID,
		UserID:    userID,
	}

	mockStore.On("GetByID", ctx, connID).Return(conn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(nil, errors.New("not found"))

	// Execute
	err := service.ShareConnection(ctx, connID, userID, orgID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not member of organization")
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

// ====================================================================
// Service Tests - GetOrganizationConnections
// ====================================================================

func TestGetOrganizationConnections_Success(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	orgID := "org-1"
	userID := "user-1"

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
	}

	connections := []*turso.Connection{
		{ID: "conn-1", Name: "DB 1", Visibility: "shared"},
		{ID: "conn-2", Name: "DB 2", Visibility: "shared"},
	}

	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockStore.On("GetConnectionsByOrganization", ctx, orgID).Return(connections, nil)

	// Execute
	result, err := service.GetOrganizationConnections(ctx, orgID, userID)

	// Verify
	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestGetOrganizationConnections_NoPermission(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	orgID := "org-1"
	userID := "user-1"

	// Member role does have view permission, so this shouldn't fail
	// Let's test with nil member instead
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(nil, errors.New("not found"))

	// Execute
	_, err := service.GetOrganizationConnections(ctx, orgID, userID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not member of organization")
	mockOrgRepo.AssertExpectations(t)
}

// ====================================================================
// Service Tests - UpdateConnection
// ====================================================================

func TestUpdateConnection_OwnerCanUpdate(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	userID := "user-1"
	orgID := "org-1"

	existingConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "Old Name",
		CreatedBy:      userID,
		OrganizationID: &orgID,
	}

	updatedConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "New Name",
		OrganizationID: &orgID,
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleMember,
	}

	mockStore.On("GetByID", ctx, "conn-1").Return(existingConn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockStore.On("Update", ctx, updatedConn).Return(nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Execute
	err := service.UpdateConnection(ctx, updatedConn, userID)

	// Verify
	require.NoError(t, err)
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestUpdateConnection_AdminCanUpdateOthersResources(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	userID := "user-2" // Admin user
	orgID := "org-1"

	existingConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "Old Name",
		CreatedBy:      "user-1", // Different owner
		OrganizationID: &orgID,
	}

	updatedConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "New Name",
		OrganizationID: &orgID,
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleAdmin, // Admin can update others' resources
	}

	mockStore.On("GetByID", ctx, "conn-1").Return(existingConn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)
	mockStore.On("Update", ctx, updatedConn).Return(nil)
	mockOrgRepo.On("CreateAuditLog", ctx, mock.Anything).Return(nil)

	// Execute
	err := service.UpdateConnection(ctx, updatedConn, userID)

	// Verify
	require.NoError(t, err)
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestUpdateConnection_MemberCannotUpdateOthersResources(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewService(mockStore, mockOrgRepo, logger)

	ctx := context.Background()
	userID := "user-2"
	orgID := "org-1"

	existingConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "Old Name",
		CreatedBy:      "user-1", // Different owner
		OrganizationID: &orgID,
	}

	updatedConn := &turso.Connection{
		ID:             "conn-1",
		Name:           "New Name",
		OrganizationID: &orgID,
	}

	member := &organization.OrganizationMember{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           organization.RoleMember, // Member cannot update others' resources
	}

	mockStore.On("GetByID", ctx, "conn-1").Return(existingConn, nil)
	mockOrgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)

	// Execute
	err := service.UpdateConnection(ctx, updatedConn, userID)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}
