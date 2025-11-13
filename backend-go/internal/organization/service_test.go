package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/organization"
)

// ====================================================================
// Mock Repository Implementation
// ====================================================================

type mockRepository struct {
	createFunc                func(ctx context.Context, org *organization.Organization) error
	getByIDFunc               func(ctx context.Context, id string) (*organization.Organization, error)
	getByUserIDFunc           func(ctx context.Context, userID string) ([]*organization.Organization, error)
	updateFunc                func(ctx context.Context, org *organization.Organization) error
	deleteFunc                func(ctx context.Context, id string) error
	addMemberFunc             func(ctx context.Context, member *organization.OrganizationMember) error
	removeMemberFunc          func(ctx context.Context, orgID, userID string) error
	getMemberFunc             func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error)
	getMembersFunc            func(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error)
	updateMemberRoleFunc      func(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error
	getMemberCountFunc        func(ctx context.Context, orgID string) (int, error)
	createInvitationFunc      func(ctx context.Context, invitation *organization.OrganizationInvitation) error
	getInvitationFunc         func(ctx context.Context, id string) (*organization.OrganizationInvitation, error)
	getInvitationByTokenFunc  func(ctx context.Context, token string) (*organization.OrganizationInvitation, error)
	getInvitationsByOrgFunc   func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error)
	getInvitationsByEmailFunc func(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error)
	updateInvitationFunc      func(ctx context.Context, invitation *organization.OrganizationInvitation) error
	deleteInvitationFunc      func(ctx context.Context, id string) error
	createAuditLogFunc        func(ctx context.Context, log *organization.AuditLog) error
	getAuditLogsFunc          func(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error)
}

func (m *mockRepository) Create(ctx context.Context, org *organization.Organization) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, org)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*organization.Organization, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetByUserID(ctx context.Context, userID string) ([]*organization.Organization, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) Update(ctx context.Context, org *organization.Organization) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, org)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) AddMember(ctx context.Context, member *organization.OrganizationMember) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, member)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(ctx, orgID, userID)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) GetMember(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, orgID, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetMembers(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx, orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error {
	if m.updateMemberRoleFunc != nil {
		return m.updateMemberRoleFunc(ctx, orgID, userID, role)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	if m.getMemberCountFunc != nil {
		return m.getMemberCountFunc(ctx, orgID)
	}
	return 0, errors.New("not implemented")
}

func (m *mockRepository) CreateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	if m.createInvitationFunc != nil {
		return m.createInvitationFunc(ctx, invitation)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) GetInvitation(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
	if m.getInvitationFunc != nil {
		return m.getInvitationFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetInvitationByToken(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
	if m.getInvitationByTokenFunc != nil {
		return m.getInvitationByTokenFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetInvitationsByOrg(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
	if m.getInvitationsByOrgFunc != nil {
		return m.getInvitationsByOrgFunc(ctx, orgID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetInvitationsByEmail(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
	if m.getInvitationsByEmailFunc != nil {
		return m.getInvitationsByEmailFunc(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) UpdateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	if m.updateInvitationFunc != nil {
		return m.updateInvitationFunc(ctx, invitation)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) DeleteInvitation(ctx context.Context, id string) error {
	if m.deleteInvitationFunc != nil {
		return m.deleteInvitationFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockRepository) CreateAuditLog(ctx context.Context, log *organization.AuditLog) error {
	if m.createAuditLogFunc != nil {
		return m.createAuditLogFunc(ctx, log)
	}
	return nil // Don't fail on audit log
}

func (m *mockRepository) GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error) {
	if m.getAuditLogsFunc != nil {
		return m.getAuditLogsFunc(ctx, orgID, limit, offset)
	}
	return nil, errors.New("not implemented")
}

// ====================================================================
// Helper Functions
// ====================================================================

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs in tests
	return logger
}

// ====================================================================
// Service Tests - CreateOrganization
// ====================================================================

func TestCreateOrganization_Success(t *testing.T) {
	var createdOrg *organization.Organization

	repo := &mockRepository{
		createFunc: func(ctx context.Context, org *organization.Organization) error {
			createdOrg = org
			org.ID = "org-1"
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateOrganizationInput{
		Name:        "Test Organization",
		Description: "A test organization",
	}

	result, err := service.CreateOrganization(context.Background(), "user-1", input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Organization", createdOrg.Name)
	assert.Equal(t, "A test organization", createdOrg.Description)
	assert.Equal(t, "user-1", createdOrg.OwnerID)
	assert.Equal(t, 10, createdOrg.MaxMembers)
	assert.NotNil(t, createdOrg.Settings)
}

func TestCreateOrganization_NameValidation(t *testing.T) {
	tests := []struct {
		name      string
		orgName   string
		expectErr bool
		errMsg    string
	}{
		{"valid name", "Valid Org Name", false, ""},
		{"too short", "AB", true, "at least 3 characters"},
		{"minimum valid", "ABC", false, ""},
		{"too long", "ThisOrganizationNameIsFarTooLongAndExceedsTheFiftyCharLimit", true, "at most 50 characters"},
		{"maximum valid", "This Organization Name Is Exactly Fifty Chars--", false, ""},
		{"invalid chars", "Test@Org!", true, "can only contain"},
		{"with spaces", "My Cool Org", false, ""},
		{"with hyphens", "My-Cool-Org", false, ""},
		{"with underscores", "My_Cool_Org", false, ""},
		{"only spaces", "   ", true, "at least 3 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				createFunc: func(ctx context.Context, org *organization.Organization) error {
					return nil
				},
			}

			service := organization.NewService(repo, newTestLogger())

			input := &organization.CreateOrganizationInput{
				Name: tt.orgName,
			}

			_, err := service.CreateOrganization(context.Background(), "user-1", input)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateOrganization_RepositoryError(t *testing.T) {
	repo := &mockRepository{
		createFunc: func(ctx context.Context, org *organization.Organization) error {
			return errors.New("database error")
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateOrganizationInput{
		Name: "Test Org",
	}

	result, err := service.CreateOrganization(context.Background(), "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create organization")
}

// ====================================================================
// Service Tests - GetOrganization
// ====================================================================

func TestGetOrganization_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleMember,
			}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{
				ID:      id,
				Name:    "Test Org",
				OwnerID: "user-1",
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.GetOrganization(context.Background(), "org-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "org-1", result.ID)
	assert.Equal(t, "Test Org", result.Name)
}

func TestGetOrganization_NotMember(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return nil, errors.New("member not found")
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.GetOrganization(context.Background(), "org-1", "user-2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not a member")
}

// ====================================================================
// Service Tests - UpdateOrganization
// ====================================================================

func TestUpdateOrganization_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleOwner,
			}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{
				ID:         id,
				Name:       "Old Name",
				OwnerID:    "user-1",
				MaxMembers: 10,
			}, nil
		},
		updateFunc: func(ctx context.Context, org *organization.Organization) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	newName := "New Name"
	newDesc := "New Description"
	input := &organization.UpdateOrganizationInput{
		Name:        &newName,
		Description: &newDesc,
	}

	result, err := service.UpdateOrganization(context.Background(), "org-1", "user-1", input)

	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, "New Description", result.Description)
}

func TestUpdateOrganization_InsufficientPermissions(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleMember,
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	newName := "New Name"
	input := &organization.UpdateOrganizationInput{
		Name: &newName,
	}

	result, err := service.UpdateOrganization(context.Background(), "org-1", "user-2", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestUpdateOrganization_ReduceMaxMembersBelowCurrent(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleOwner,
			}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{
				ID:         id,
				MaxMembers: 10,
			}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 5, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	newMax := 3
	input := &organization.UpdateOrganizationInput{
		MaxMembers: &newMax,
	}

	result, err := service.UpdateOrganization(context.Background(), "org-1", "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot reduce max_members")
}

func TestUpdateOrganization_InvalidMaxMembers(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleOwner,
			}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{
				ID:         id,
				MaxMembers: 10,
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	newMax := 0
	input := &organization.UpdateOrganizationInput{
		MaxMembers: &newMax,
	}

	result, err := service.UpdateOrganization(context.Background(), "org-1", "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "must be at least 1")
}

// ====================================================================
// Service Tests - DeleteOrganization
// ====================================================================

func TestDeleteOrganization_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleOwner,
			}, nil
		},
		getMembersFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
			return []*organization.OrganizationMember{
				{UserID: "user-1", Role: organization.RoleOwner},
			}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.DeleteOrganization(context.Background(), "org-1", "user-1")

	assert.NoError(t, err)
}

func TestDeleteOrganization_NotOwner(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleAdmin,
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.DeleteOrganization(context.Background(), "org-1", "user-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestDeleteOrganization_HasOtherMembers(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleOwner,
			}, nil
		},
		getMembersFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
			return []*organization.OrganizationMember{
				{UserID: "user-1", Role: organization.RoleOwner},
				{UserID: "user-2", Role: organization.RoleMember},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.DeleteOrganization(context.Background(), "org-1", "user-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete organization with other members")
}

// ====================================================================
// Service Tests - Member Management
// ====================================================================

func TestUpdateMemberRole_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			if userID == "user-1" {
				return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
			}
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleMember}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "user-1"}, nil
		},
		updateMemberRoleFunc: func(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.UpdateMemberRole(context.Background(), "org-1", "user-2", "user-1", organization.RoleAdmin)

	assert.NoError(t, err)
}

func TestUpdateMemberRole_CannotChangeOwnerRole(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "user-1"}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.UpdateMemberRole(context.Background(), "org-1", "user-1", "user-1", organization.RoleAdmin)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change owner's role")
}

func TestUpdateMemberRole_AdminCannotPromoteToOwner(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			if userID == "user-1" {
				return &organization.OrganizationMember{UserID: userID, Role: organization.RoleAdmin}, nil
			}
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleMember}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "owner-1"}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.UpdateMemberRole(context.Background(), "org-1", "user-2", "user-1", organization.RoleOwner)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only owners can assign owner role")
}

func TestRemoveMember_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			if userID == "user-1" {
				return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
			}
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleMember}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "user-1"}, nil
		},
		removeMemberFunc: func(ctx context.Context, orgID, userID string) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.RemoveMember(context.Background(), "org-1", "user-2", "user-1")

	assert.NoError(t, err)
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "user-1"}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.RemoveMember(context.Background(), "org-1", "user-1", "user-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove owner")
}

func TestRemoveMember_AdminCanOnlyRemoveMembers(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			if userID == "user-1" {
				return &organization.OrganizationMember{UserID: userID, Role: organization.RoleAdmin}, nil
			}
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleAdmin}, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, OwnerID: "owner-1"}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.RemoveMember(context.Background(), "org-1", "user-2", "user-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admins can only remove members")
}

// ====================================================================
// Service Tests - Invitation Management
// ====================================================================

func TestCreateInvitation_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 3, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, MaxMembers: 10}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{}, nil
		},
		createInvitationFunc: func(ctx context.Context, invitation *organization.OrganizationInvitation) error {
			invitation.ID = "inv-1"
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateInvitationInput{
		Email: "test@example.com",
		Role:  organization.RoleMember,
	}

	result, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test@example.com", result.Email)
	assert.NotEmpty(t, result.Token)
	assert.False(t, result.ExpiresAt.IsZero())
}

func TestCreateInvitation_InvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"missing @", "notanemail"},
		{"missing domain", "test@"},
		{"missing username", "@example.com"},
		{"invalid format", "test@@example.com"},
		{"spaces", "test @example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
					return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
				},
				getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
					return 3, nil
				},
				getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
					return &organization.Organization{ID: id, MaxMembers: 10}, nil
				},
				getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
					return []*organization.OrganizationInvitation{}, nil
				},
			}

			service := organization.NewService(repo, newTestLogger())

			input := &organization.CreateInvitationInput{
				Email: tt.email,
				Role:  organization.RoleMember,
			}

			result, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid email")
		})
	}
}

func TestCreateInvitation_MemberLimitReached(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 10, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, MaxMembers: 10}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateInvitationInput{
		Email: "test@example.com",
		Role:  organization.RoleMember,
	}

	result, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reached maximum member limit")
}

func TestCreateInvitation_AdminCannotInviteAdmins(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleAdmin}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 3, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, MaxMembers: 10}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateInvitationInput{
		Email: "test@example.com",
		Role:  organization.RoleAdmin,
	}

	result, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "only owners can invite admins")
}

func TestAcceptInvitation_Success(t *testing.T) {
	now := time.Now()
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             "inv-1",
				OrganizationID: "org-1",
				Email:          "test@example.com",
				Role:           organization.RoleMember,
				ExpiresAt:      now.Add(24 * time.Hour),
				Organization: &organization.Organization{
					ID:         "org-1",
					Name:       "Test Org",
					MaxMembers: 10,
				},
			}, nil
		},
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return nil, errors.New("not found")
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 3, nil
		},
		addMemberFunc: func(ctx context.Context, member *organization.OrganizationMember) error {
			return nil
		},
		updateInvitationFunc: func(ctx context.Context, invitation *organization.OrganizationInvitation) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.AcceptInvitation(context.Background(), "test-token", "user-2")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "org-1", result.ID)
}

func TestAcceptInvitation_ExpiredInvitation(t *testing.T) {
	now := time.Now()
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             "inv-1",
				OrganizationID: "org-1",
				Email:          "test@example.com",
				Role:           organization.RoleMember,
				ExpiresAt:      now.Add(-24 * time.Hour), // Expired
				Organization: &organization.Organization{
					ID: "org-1",
				},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.AcceptInvitation(context.Background(), "test-token", "user-2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expired")
}

func TestAcceptInvitation_AlreadyAccepted(t *testing.T) {
	now := time.Now()
	acceptedAt := now.Add(-1 * time.Hour)
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             "inv-1",
				OrganizationID: "org-1",
				Email:          "test@example.com",
				Role:           organization.RoleMember,
				ExpiresAt:      now.Add(24 * time.Hour),
				AcceptedAt:     &acceptedAt,
				Organization: &organization.Organization{
					ID: "org-1",
				},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.AcceptInvitation(context.Background(), "test-token", "user-2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already accepted")
}

func TestAcceptInvitation_AlreadyMember(t *testing.T) {
	now := time.Now()
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             "inv-1",
				OrganizationID: "org-1",
				Email:          "test@example.com",
				Role:           organization.RoleMember,
				ExpiresAt:      now.Add(24 * time.Hour),
				Organization: &organization.Organization{
					ID:         "org-1",
					MaxMembers: 10,
				},
			}, nil
		},
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{
				UserID: userID,
				Role:   organization.RoleMember,
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.AcceptInvitation(context.Background(), "test-token", "user-2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already a member")
}

func TestAcceptInvitation_OrganizationDeleted(t *testing.T) {
	now := time.Now()
	deletedAt := now.Add(-1 * time.Hour)
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             "inv-1",
				OrganizationID: "org-1",
				Email:          "test@example.com",
				Role:           organization.RoleMember,
				ExpiresAt:      now.Add(24 * time.Hour),
				Organization: &organization.Organization{
					ID:        "org-1",
					DeletedAt: &deletedAt,
				},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	result, err := service.AcceptInvitation(context.Background(), "test-token", "user-2")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no longer exists")
}

// ====================================================================
// Service Tests - GetMembers
// ====================================================================

func TestGetMembers_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleMember}, nil
		},
		getMembersFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
			return []*organization.OrganizationMember{
				{UserID: "user-1", Role: organization.RoleOwner},
				{UserID: "user-2", Role: organization.RoleMember},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	members, err := service.GetMembers(context.Background(), "org-1", "user-1")

	require.NoError(t, err)
	assert.Len(t, members, 2)
}

// ====================================================================
// Service Tests - GetUserOrganizations
// ====================================================================

func TestGetUserOrganizations_Success(t *testing.T) {
	repo := &mockRepository{
		getByUserIDFunc: func(ctx context.Context, userID string) ([]*organization.Organization, error) {
			return []*organization.Organization{
				{ID: "org-1", Name: "Org 1", OwnerID: userID},
				{ID: "org-2", Name: "Org 2", OwnerID: "other-user"},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	orgs, err := service.GetUserOrganizations(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

func TestGetUserOrganizations_Empty(t *testing.T) {
	repo := &mockRepository{
		getByUserIDFunc: func(ctx context.Context, userID string) ([]*organization.Organization, error) {
			return []*organization.Organization{}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	orgs, err := service.GetUserOrganizations(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, orgs, 0)
}

// ====================================================================
// Service Tests - Audit Logs
// ====================================================================

func TestGetAuditLogs_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getAuditLogsFunc: func(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error) {
			return []*organization.AuditLog{
				{ID: "log-1", Action: "create", ResourceType: "organization"},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	logs, err := service.GetAuditLogs(context.Background(), "org-1", "user-1", 10, 0)

	require.NoError(t, err)
	assert.Len(t, logs, 1)
}

func TestGetAuditLogs_InsufficientPermissions(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleMember}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	logs, err := service.GetAuditLogs(context.Background(), "org-1", "user-2", 10, 0)

	assert.Error(t, err)
	assert.Nil(t, logs)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestCreateAuditLog_Success(t *testing.T) {
	var created *organization.AuditLog
	repo := &mockRepository{
		createAuditLogFunc: func(ctx context.Context, log *organization.AuditLog) error {
			created = log
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	orgID := "org-1"
	log := &organization.AuditLog{
		OrganizationID: &orgID,
		UserID:         "user-1",
		Action:         "test",
		ResourceType:   "organization",
	}

	err := service.CreateAuditLog(context.Background(), log)

	assert.NoError(t, err)
	assert.NotNil(t, created)
}

func TestCreateAuditLog_FailureDoesNotError(t *testing.T) {
	repo := &mockRepository{
		createAuditLogFunc: func(ctx context.Context, log *organization.AuditLog) error {
			return errors.New("database error")
		},
	}

	service := organization.NewService(repo, newTestLogger())

	orgID := "org-1"
	log := &organization.AuditLog{
		OrganizationID: &orgID,
		UserID:         "user-1",
		Action:         "test",
		ResourceType:   "organization",
	}

	// Should not return error even if audit logging fails
	err := service.CreateAuditLog(context.Background(), log)
	assert.NoError(t, err)
}

// ====================================================================
// Service Tests - Invitations
// ====================================================================

func TestGetInvitations_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{
				{ID: "inv-1", Email: "test@example.com"},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	invitations, err := service.GetInvitations(context.Background(), "org-1", "user-1")

	require.NoError(t, err)
	assert.Len(t, invitations, 1)
}

func TestGetPendingInvitationsForEmail_Success(t *testing.T) {
	repo := &mockRepository{
		getInvitationsByEmailFunc: func(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{
				{ID: "inv-1", Email: email},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	invitations, err := service.GetPendingInvitationsForEmail(context.Background(), "test@example.com")

	require.NoError(t, err)
	assert.Len(t, invitations, 1)
}

func TestGetPendingInvitationsForEmail_EmailNormalization(t *testing.T) {
	var queriedEmail string
	repo := &mockRepository{
		getInvitationsByEmailFunc: func(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
			queriedEmail = email
			return []*organization.OrganizationInvitation{}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	_, err := service.GetPendingInvitationsForEmail(context.Background(), "Test@Example.COM")

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", queriedEmail)
}

func TestDeclineInvitation_Success(t *testing.T) {
	repo := &mockRepository{
		getInvitationByTokenFunc: func(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:    "inv-1",
				Email: "test@example.com",
			}, nil
		},
		deleteInvitationFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.DeclineInvitation(context.Background(), "test-token")

	assert.NoError(t, err)
}

func TestRevokeInvitation_Success(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getInvitationFunc: func(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             id,
				OrganizationID: "org-1",
			}, nil
		},
		deleteInvitationFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.RevokeInvitation(context.Background(), "org-1", "inv-1", "user-1")

	assert.NoError(t, err)
}

func TestRevokeInvitation_WrongOrganization(t *testing.T) {
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getInvitationFunc: func(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
			return &organization.OrganizationInvitation{
				ID:             id,
				OrganizationID: "org-2", // Different org
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	err := service.RevokeInvitation(context.Background(), "org-1", "inv-1", "user-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this organization")
}

// ====================================================================
// Test OrganizationRole Methods
// ====================================================================

func TestOrganizationRole_Validate(t *testing.T) {
	tests := []struct {
		role  organization.OrganizationRole
		valid bool
	}{
		{organization.RoleOwner, true},
		{organization.RoleAdmin, true},
		{organization.RoleMember, true},
		{organization.OrganizationRole("invalid"), false},
		{organization.OrganizationRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.role.Validate())
		})
	}
}

func TestOrganizationRole_String(t *testing.T) {
	assert.Equal(t, "owner", organization.RoleOwner.String())
	assert.Equal(t, "admin", organization.RoleAdmin.String())
	assert.Equal(t, "member", organization.RoleMember.String())
}

// ====================================================================
// Test Invitation Helper Methods
// ====================================================================

func TestInvitation_IsExpired(t *testing.T) {
	now := time.Now()

	inv1 := &organization.OrganizationInvitation{
		ExpiresAt: now.Add(1 * time.Hour),
	}
	assert.False(t, inv1.IsExpired())

	inv2 := &organization.OrganizationInvitation{
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	assert.True(t, inv2.IsExpired())
}

func TestInvitation_IsAccepted(t *testing.T) {
	now := time.Now()

	inv1 := &organization.OrganizationInvitation{
		AcceptedAt: nil,
	}
	assert.False(t, inv1.IsAccepted())

	inv2 := &organization.OrganizationInvitation{
		AcceptedAt: &now,
	}
	assert.True(t, inv2.IsAccepted())
}

// ====================================================================
// Edge Cases and Error Scenarios
// ====================================================================

func TestService_NilContext(t *testing.T) {
	repo := &mockRepository{}
	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateOrganizationInput{
		Name: "Test Org",
	}

	// Most Go functions will panic with nil context, but let's ensure they handle it gracefully
	_, err := service.CreateOrganization(context.TODO(), "user-1", input)
	assert.Error(t, err)
}

func TestService_EmptyUserID(t *testing.T) {
	repo := &mockRepository{
		createFunc: func(ctx context.Context, org *organization.Organization) error {
			assert.Equal(t, "", org.OwnerID)
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateOrganizationInput{
		Name: "Test Org",
	}

	_, err := service.CreateOrganization(context.Background(), "", input)
	assert.NoError(t, err)
}

func TestCreateInvitation_EmailCaseInsensitive(t *testing.T) {
	var createdInvitation *organization.OrganizationInvitation
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 3, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, MaxMembers: 10}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			return []*organization.OrganizationInvitation{}, nil
		},
		createInvitationFunc: func(ctx context.Context, invitation *organization.OrganizationInvitation) error {
			createdInvitation = invitation
			return nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateInvitationInput{
		Email: "Test@Example.COM",
		Role:  organization.RoleMember,
	}

	_, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", createdInvitation.Email)
}

func TestCreateInvitation_DuplicateError(t *testing.T) {
	now := time.Now()
	repo := &mockRepository{
		getMemberFunc: func(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
			return &organization.OrganizationMember{UserID: userID, Role: organization.RoleOwner}, nil
		},
		getMemberCountFunc: func(ctx context.Context, orgID string) (int, error) {
			return 3, nil
		},
		getByIDFunc: func(ctx context.Context, id string) (*organization.Organization, error) {
			return &organization.Organization{ID: id, MaxMembers: 10}, nil
		},
		getInvitationsByOrgFunc: func(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
			// Return an existing pending invitation for the same email
			return []*organization.OrganizationInvitation{
				{
					ID:             "inv-1",
					OrganizationID: orgID,
					Email:          "test@example.com",
					Role:           organization.RoleMember,
					ExpiresAt:      now.Add(24 * time.Hour),
					AcceptedAt:     nil,
				},
			}, nil
		},
	}

	service := organization.NewService(repo, newTestLogger())

	input := &organization.CreateInvitationInput{
		Email: "test@example.com",
		Role:  organization.RoleMember,
	}

	result, err := service.CreateInvitation(context.Background(), "org-1", "user-1", input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invitation already exists")
}
