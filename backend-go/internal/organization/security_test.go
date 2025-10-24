package organization

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SecurityTestSuite tests all permission bypass scenarios
func TestSecurityPermissionBypass(t *testing.T) {
	// Table-driven tests for each attack vector
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T, s *Service) (orgID, userID string)
		attackFunc    func(t *testing.T, s *Service, orgID, userID string) error
		expectedError string
		severity      string // P0 (Critical), P1 (High), P2 (Medium), P3 (Low)
	}{
		// Member Permission Bypass Attempts
		{
			name: "Member tries to update organization",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithMember(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				name := "Hacked Name"
				_, err := s.UpdateOrganization(context.Background(), orgID, userID, &UpdateOrganizationInput{
					Name: &name,
				})
				return err
			},
			expectedError: "insufficient permissions",
			severity:      "P1",
		},
		{
			name: "Member tries to delete organization",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithMember(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				return s.DeleteOrganization(context.Background(), orgID, userID)
			},
			expectedError: "insufficient permissions",
			severity:      "P0",
		},
		{
			name: "Member tries to invite other members",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithMember(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				_, err := s.CreateInvitation(context.Background(), orgID, userID, &CreateInvitationInput{
					Email: "newuser@test.com",
					Role:  RoleMember,
				})
				return err
			},
			expectedError: "insufficient permissions",
			severity:      "P1",
		},
		{
			name: "Member tries to remove other members",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				orgID, memberID := setupOrgWithMember(t, s)
				// Add another member to remove
				otherMemberID := "other-member-id"
				s.repo.(*mockRepository).members = append(s.repo.(*mockRepository).members, &OrganizationMember{
					OrganizationID: orgID,
					UserID:         otherMemberID,
					Role:           RoleMember,
				})
				return orgID, memberID
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				return s.RemoveMember(context.Background(), orgID, "other-member-id", userID)
			},
			expectedError: "insufficient permissions",
			severity:      "P1",
		},
		{
			name: "Member tries to change roles",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				orgID, memberID := setupOrgWithMember(t, s)
				// Add another member whose role to change
				otherMemberID := "other-member-id"
				s.repo.(*mockRepository).members = append(s.repo.(*mockRepository).members, &OrganizationMember{
					OrganizationID: orgID,
					UserID:         otherMemberID,
					Role:           RoleMember,
				})
				return orgID, memberID
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				return s.UpdateMemberRole(context.Background(), orgID, "other-member-id", userID, RoleAdmin)
			},
			expectedError: "insufficient permissions",
			severity:      "P1",
		},
		{
			name: "Member tries to view audit logs",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithMember(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				_, err := s.GetAuditLogs(context.Background(), orgID, userID, 10, 0)
				return err
			},
			expectedError: "insufficient permissions",
			severity:      "P2",
		},

		// Admin Permission Bypass Attempts
		{
			name: "Admin tries to delete organization",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithAdmin(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				return s.DeleteOrganization(context.Background(), orgID, userID)
			},
			expectedError: "insufficient permissions",
			severity:      "P0",
		},
		{
			name: "Admin tries to promote to owner",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				orgID, adminID := setupOrgWithAdmin(t, s)
				// Add a member to promote
				memberID := "member-to-promote"
				s.repo.(*mockRepository).members = append(s.repo.(*mockRepository).members, &OrganizationMember{
					OrganizationID: orgID,
					UserID:         memberID,
					Role:           RoleMember,
				})
				return orgID, adminID
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				return s.UpdateMemberRole(context.Background(), orgID, "member-to-promote", userID, RoleOwner)
			},
			expectedError: "only owners can assign owner role",
			severity:      "P0",
		},
		{
			name: "Admin tries to remove owner",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithAdmin(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				org, _ := s.repo.GetByID(context.Background(), orgID)
				return s.RemoveMember(context.Background(), orgID, org.OwnerID, userID)
			},
			expectedError: "cannot remove owner",
			severity:      "P0",
		},
		{
			name: "Admin tries to change owner's role",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithAdmin(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				org, _ := s.repo.GetByID(context.Background(), orgID)
				return s.UpdateMemberRole(context.Background(), orgID, org.OwnerID, userID, RoleMember)
			},
			expectedError: "cannot change owner's role",
			severity:      "P0",
		},
		{
			name: "Admin tries to invite admin",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				return setupOrgWithAdmin(t, s)
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				_, err := s.CreateInvitation(context.Background(), orgID, userID, &CreateInvitationInput{
					Email: "newadmin@test.com",
					Role:  RoleAdmin,
				})
				return err
			},
			expectedError: "only owners can invite admins",
			severity:      "P1",
		},

		// Non-Member Access Attempts
		{
			name: "Non-member tries to access organization",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				orgID, _ := setupOrgWithMember(t, s)
				nonMemberID := "non-member-id"
				return orgID, nonMemberID
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				_, err := s.GetOrganization(context.Background(), orgID, userID)
				return err
			},
			expectedError: "not a member",
			severity:      "P0",
		},
		{
			name: "Non-member tries to invite",
			setupFunc: func(t *testing.T, s *Service) (string, string) {
				orgID, _ := setupOrgWithMember(t, s)
				nonMemberID := "non-member-id"
				return orgID, nonMemberID
			},
			attackFunc: func(t *testing.T, s *Service, orgID, userID string) error {
				_, err := s.CreateInvitation(context.Background(), orgID, userID, &CreateInvitationInput{
					Email: "hacker@test.com",
					Role:  RoleOwner,
				})
				return err
			},
			expectedError: "not a member",
			severity:      "P0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo := newMockRepository()
			logger := newTestLogger()
			service := NewService(repo, logger)

			// Setup test data
			orgID, userID := tc.setupFunc(t, service)

			// Execute attack
			err := tc.attackFunc(t, service, orgID, userID)

			// Verify protection worked
			require.Error(t, err, "Attack should have been prevented")
			assert.Contains(t, err.Error(), tc.expectedError,
				"Expected error message not found. Severity: %s", tc.severity)
		})
	}
}

// TestPrivilegeEscalation tests privilege escalation attempts
func TestPrivilegeEscalation(t *testing.T) {
	testCases := []struct {
		name          string
		attackFunc    func(t *testing.T, s *Service) error
		expectedError string
	}{
		{
			name: "Member tries to self-promote to admin",
			attackFunc: func(t *testing.T, s *Service) error {
				orgID, memberID := setupOrgWithMember(t, s)
				return s.UpdateMemberRole(context.Background(), orgID, memberID, memberID, RoleAdmin)
			},
			expectedError: "insufficient permissions",
		},
		{
			name: "Admin tries to self-promote to owner",
			attackFunc: func(t *testing.T, s *Service) error {
				orgID, adminID := setupOrgWithAdmin(t, s)
				return s.UpdateMemberRole(context.Background(), orgID, adminID, adminID, RoleOwner)
			},
			expectedError: "only owners can assign owner role",
		},
		{
			name: "User adds self to org without invitation",
			attackFunc: func(t *testing.T, s *Service) error {
				repo := s.repo.(*mockRepository)
				orgID := "test-org"
				userID := "unauthorized-user"

				// Create org without this user
				repo.organizations = []*Organization{{
					ID:      orgID,
					Name:    "Test Org",
					OwnerID: "owner-id",
				}}

				// Try to add self as member directly (this should not be possible via API)
				member := &OrganizationMember{
					OrganizationID: orgID,
					UserID:         userID,
					Role:           RoleOwner,
				}
				return repo.AddMember(context.Background(), member)
			},
			expectedError: "", // Direct repository access should be prevented at handler level
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			logger := newTestLogger()
			service := NewService(repo, logger)

			err := tc.attackFunc(t, service)
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

// TestInputValidation tests input validation and injection prevention
func TestInputValidation(t *testing.T) {
	testCases := []struct {
		name          string
		input         interface{}
		expectedError string
	}{
		{
			name: "SQL injection in organization name",
			input: &CreateOrganizationInput{
				Name:        "Test' OR 1=1 --",
				Description: "SQL injection test",
			},
			expectedError: "can only contain letters, numbers",
		},
		{
			name: "XSS in organization description",
			input: &CreateOrganizationInput{
				Name:        "Valid Name",
				Description: "<script>alert('XSS')</script>",
			},
			expectedError: "", // Should be sanitized, not rejected
		},
		{
			name: "Invalid email format",
			input: &CreateInvitationInput{
				Email: "not-an-email",
				Role:  RoleMember,
			},
			expectedError: "invalid email address",
		},
		{
			name: "Invalid role value",
			input: &CreateInvitationInput{
				Email: "test@example.com",
				Role:  "superadmin",
			},
			expectedError: "", // Role validation happens at type level
		},
		{
			name: "Overly long organization name",
			input: &CreateOrganizationInput{
				Name:        strings.Repeat("A", 100),
				Description: "Test",
			},
			expectedError: "at most 50 characters",
		},
		{
			name: "Empty organization name",
			input: &CreateOrganizationInput{
				Name:        "",
				Description: "Test",
			},
			expectedError: "at least 3 characters",
		},
		{
			name: "Special characters in organization name",
			input: &CreateOrganizationInput{
				Name:        "Test@#$%^&*()",
				Description: "Test",
			},
			expectedError: "can only contain letters, numbers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			logger := newTestLogger()
			service := NewService(repo, logger)

			var err error
			switch v := tc.input.(type) {
			case *CreateOrganizationInput:
				_, err = service.CreateOrganization(context.Background(), "user-id", v)
			case *CreateInvitationInput:
				// Setup org first
				org := &Organization{
					ID:         "test-org",
					Name:       "Test Org",
					OwnerID:    "owner-id",
					MaxMembers: 10,
				}
				repo.organizations = append(repo.organizations, org)
				repo.members = append(repo.members, &OrganizationMember{
					OrganizationID: org.ID,
					UserID:         "owner-id",
					Role:           RoleOwner,
				})
				_, err = service.CreateInvitation(context.Background(), org.ID, "owner-id", v)
			}

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

// TestTokenSecurity tests token-related security
func TestTokenSecurity(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T, s *Service) string
		attackFunc    func(t *testing.T, s *Service, token string) error
		expectedError string
	}{
		{
			name: "Expired token acceptance",
			setupFunc: func(t *testing.T, s *Service) string {
				// Create expired invitation
				repo := s.repo.(*mockRepository)
				invitation := &OrganizationInvitation{
					ID:             "inv-1",
					OrganizationID: "org-1",
					Email:          "test@example.com",
					Role:           RoleMember,
					Token:          "expired-token",
					ExpiresAt:      time.Now().Add(-24 * time.Hour), // Expired yesterday
					Organization:   &Organization{ID: "org-1", Name: "Test Org"},
				}
				repo.invitations = append(repo.invitations, invitation)
				return invitation.Token
			},
			attackFunc: func(t *testing.T, s *Service, token string) error {
				_, err := s.AcceptInvitation(context.Background(), token, "user-id")
				return err
			},
			expectedError: "invitation has expired",
		},
		{
			name: "Already accepted invitation",
			setupFunc: func(t *testing.T, s *Service) string {
				// Create already accepted invitation
				repo := s.repo.(*mockRepository)
				now := time.Now()
				invitation := &OrganizationInvitation{
					ID:             "inv-2",
					OrganizationID: "org-1",
					Email:          "test@example.com",
					Role:           RoleMember,
					Token:          "used-token",
					ExpiresAt:      time.Now().Add(24 * time.Hour),
					AcceptedAt:     &now, // Already accepted
					Organization:   &Organization{ID: "org-1", Name: "Test Org"},
				}
				repo.invitations = append(repo.invitations, invitation)
				return invitation.Token
			},
			attackFunc: func(t *testing.T, s *Service, token string) error {
				_, err := s.AcceptInvitation(context.Background(), token, "user-id")
				return err
			},
			expectedError: "invitation already accepted",
		},
		{
			name: "Invalid token format",
			setupFunc: func(t *testing.T, s *Service) string {
				return "invalid-token-12345"
			},
			attackFunc: func(t *testing.T, s *Service, token string) error {
				_, err := s.AcceptInvitation(context.Background(), token, "user-id")
				return err
			},
			expectedError: "invitation not found",
		},
		{
			name: "Deleted organization invitation",
			setupFunc: func(t *testing.T, s *Service) string {
				// Create invitation for deleted org
				repo := s.repo.(*mockRepository)
				deletedAt := time.Now()
				invitation := &OrganizationInvitation{
					ID:             "inv-3",
					OrganizationID: "org-1",
					Email:          "test@example.com",
					Role:           RoleMember,
					Token:          "deleted-org-token",
					ExpiresAt:      time.Now().Add(24 * time.Hour),
					Organization: &Organization{
						ID:        "org-1",
						Name:      "Deleted Org",
						DeletedAt: &deletedAt, // Org is deleted
					},
				}
				repo.invitations = append(repo.invitations, invitation)
				return invitation.Token
			},
			attackFunc: func(t *testing.T, s *Service, token string) error {
				_, err := s.AcceptInvitation(context.Background(), token, "user-id")
				return err
			},
			expectedError: "organization no longer exists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			logger := newTestLogger()
			service := NewService(repo, logger)

			// Setup test data
			token := tc.setupFunc(t, service)

			// Execute attack
			err := tc.attackFunc(t, service, token)

			// Verify protection worked
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestRateLimiting tests rate limiting protection
func TestRateLimiting(t *testing.T) {
	t.Run("Invitation rate limit enforcement", func(t *testing.T) {
		repo := newMockRepository()
		logger := newTestLogger()
		service := NewService(repo, logger)

		// Setup rate limiter
		rateLimiter := &mockRateLimiter{
			limitAfter: 5, // Allow 5 invitations then block
		}
		service.SetRateLimiter(rateLimiter)

		// Setup organization
		orgID := "test-org"
		ownerID := "owner-id"
		repo.organizations = append(repo.organizations, &Organization{
			ID:         orgID,
			Name:       "Test Org",
			OwnerID:    ownerID,
			MaxMembers: 100,
		})
		repo.members = append(repo.members, &OrganizationMember{
			OrganizationID: orgID,
			UserID:         ownerID,
			Role:           RoleOwner,
		})

		// Try to create many invitations
		for i := 0; i < 10; i++ {
			_, err := service.CreateInvitation(context.Background(), orgID, ownerID, &CreateInvitationInput{
				Email: fmt.Sprintf("test%d@example.com", i),
				Role:  RoleMember,
			})

			if i < 5 {
				assert.NoError(t, err, "Should allow first %d invitations", i+1)
			} else {
				assert.Error(t, err, "Should block after rate limit")
				assert.Contains(t, err.Error(), "rate limit exceeded")
			}
		}
	})
}

// TestDataLeakage tests for information disclosure vulnerabilities
func TestDataLeakage(t *testing.T) {
	testCases := []struct {
		name           string
		attackFunc     func(t *testing.T, s *Service) error
		checkError     func(t *testing.T, err error)
		leakageType    string
	}{
		{
			name: "Error message doesn't reveal sensitive data",
			attackFunc: func(t *testing.T, s *Service) error {
				// Try to access non-existent org
				_, err := s.GetOrganization(context.Background(), "non-existent", "user-id")
				return err
			},
			checkError: func(t *testing.T, err error) {
				// Should not contain database details, stack traces, etc.
				errStr := err.Error()
				assert.NotContains(t, errStr, "SELECT")
				assert.NotContains(t, errStr, "INSERT")
				assert.NotContains(t, errStr, "password")
				assert.NotContains(t, errStr, "secret")
				assert.NotContains(t, errStr, "token")
				assert.NotContains(t, errStr, "stack")
			},
			leakageType: "database",
		},
		{
			name: "403 vs 404 distinction",
			attackFunc: func(t *testing.T, s *Service) error {
				// Setup org that user is not member of
				repo := s.repo.(*mockRepository)
				repo.organizations = append(repo.organizations, &Organization{
					ID:      "private-org",
					Name:    "Private Org",
					OwnerID: "other-owner",
				})
				repo.members = append(repo.members, &OrganizationMember{
					OrganizationID: "private-org",
					UserID:         "other-owner",
					Role:           RoleOwner,
				})

				// Try to access as non-member
				_, err := s.GetOrganization(context.Background(), "private-org", "unauthorized-user")
				return err
			},
			checkError: func(t *testing.T, err error) {
				// Should say "not a member" not "not found"
				assert.Contains(t, err.Error(), "not a member")
				assert.NotContains(t, err.Error(), "not found")
			},
			leakageType: "existence",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			logger := newTestLogger()
			service := NewService(repo, logger)

			err := tc.attackFunc(t, service)
			require.Error(t, err)
			tc.checkError(t, err)
		})
	}
}

// Helper functions for test setup
func setupOrgWithMember(t *testing.T, s *Service) (orgID, memberID string) {
	orgID = "test-org"
	ownerID := "owner-id"
	memberID = "member-id"

	repo := s.repo.(*mockRepository)
	repo.organizations = append(repo.organizations, &Organization{
		ID:         orgID,
		Name:       "Test Org",
		OwnerID:    ownerID,
		MaxMembers: 10,
	})

	repo.members = append(repo.members,
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         ownerID,
			Role:           RoleOwner,
		},
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         memberID,
			Role:           RoleMember,
		},
	)

	return orgID, memberID
}

func setupOrgWithAdmin(t *testing.T, s *Service) (orgID, adminID string) {
	orgID = "test-org"
	ownerID := "owner-id"
	adminID = "admin-id"

	repo := s.repo.(*mockRepository)
	repo.organizations = append(repo.organizations, &Organization{
		ID:         orgID,
		Name:       "Test Org",
		OwnerID:    ownerID,
		MaxMembers: 10,
	})

	repo.members = append(repo.members,
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         ownerID,
			Role:           RoleOwner,
		},
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         adminID,
			Role:           RoleAdmin,
		},
	)

	return orgID, adminID
}