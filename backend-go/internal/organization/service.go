package organization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceInterface defines the interface for organization service operations
type ServiceInterface interface {
	// Organization operations
	CreateOrganization(ctx context.Context, userID string, input *CreateOrganizationInput) (*Organization, error)
	GetOrganization(ctx context.Context, orgID string, userID string) (*Organization, error)
	GetUserOrganizations(ctx context.Context, userID string) ([]*Organization, error)
	UpdateOrganization(ctx context.Context, orgID string, userID string, input *UpdateOrganizationInput) (*Organization, error)
	DeleteOrganization(ctx context.Context, orgID string, userID string) error

	// Member operations
	GetMembers(ctx context.Context, orgID string, userID string) ([]*OrganizationMember, error)
	UpdateMemberRole(ctx context.Context, orgID string, targetUserID string, actorUserID string, role OrganizationRole) error
	RemoveMember(ctx context.Context, orgID string, targetUserID string, actorUserID string) error

	// Invitation operations
	CreateInvitation(ctx context.Context, orgID string, actorUserID string, input *CreateInvitationInput) (*OrganizationInvitation, error)
	GetInvitations(ctx context.Context, orgID string, userID string) ([]*OrganizationInvitation, error)
	GetPendingInvitationsForEmail(ctx context.Context, email string) ([]*OrganizationInvitation, error)
	AcceptInvitation(ctx context.Context, token string, userID string) (*Organization, error)
	DeclineInvitation(ctx context.Context, token string) error
	RevokeInvitation(ctx context.Context, orgID string, invitationID string, userID string) error

	// Audit log operations
	GetAuditLogs(ctx context.Context, orgID string, userID string, limit, offset int) ([]*AuditLog, error)
	CreateAuditLog(ctx context.Context, log *AuditLog) error
}

// EmailService defines the interface for sending emails
type EmailService interface {
	SendOrganizationInvitationEmail(email, orgName, inviterName, role, invitationURL string) error
	SendOrganizationWelcomeEmail(email, name, orgName, role string) error
	SendMemberRemovedEmail(email, orgName string) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	CheckBothLimits(userID, orgID string) (allowed bool, reason string)
}

// Service handles business logic for organizations
type Service struct {
	repo        Repository
	logger      *logrus.Logger
	emailSvc    EmailService
	rateLimiter RateLimiter
}

// NewService creates a new organization service
func NewService(repo Repository, logger *logrus.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// SetEmailService sets the email service for the organization service
func (s *Service) SetEmailService(emailSvc EmailService) {
	s.emailSvc = emailSvc
}

// SetRateLimiter sets the rate limiter for the organization service
func (s *Service) SetRateLimiter(rateLimiter RateLimiter) {
	s.rateLimiter = rateLimiter
}

// ====================================================================
// Organization Operations
// ====================================================================

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, userID string, input *CreateOrganizationInput) (*Organization, error) {
	// Validate input
	if err := s.validateOrganizationName(input.Name); err != nil {
		return nil, err
	}

	// Create organization
	org := &Organization{
		Name:        input.Name,
		Description: input.Description,
		OwnerID:     userID,
		MaxMembers:  10, // Default for free tier
		Settings:    make(map[string]interface{}),
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": org.ID,
		"user_id":         userID,
		"name":            org.Name,
	}).Info("Organization created")

	return org, nil
}

// GetOrganization retrieves an organization
func (s *Service) GetOrganization(ctx context.Context, orgID string, userID string) (*Organization, error) {
	// Check if user is a member
	if err := s.checkMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// GetUserOrganizations retrieves all organizations for a user
func (s *Service) GetUserOrganizations(ctx context.Context, userID string) ([]*Organization, error) {
	orgs, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	return orgs, nil
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, orgID string, userID string, input *UpdateOrganizationInput) (*Organization, error) {
	// Get member and check permission
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(member.Role, PermUpdateOrganization) {
		// Log permission denial
		_ = s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "organization",
			ResourceID:     &orgID,
			Details: map[string]interface{}{
				"permission": string(PermUpdateOrganization),
				"role":       string(member.Role),
				"attempted":  "update_organization",
			},
		})
		return nil, fmt.Errorf("insufficient permissions")
	}

	// Get existing organization
	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Update fields
	if input.Name != nil {
		if err := s.validateOrganizationName(*input.Name); err != nil {
			return nil, err
		}
		org.Name = *input.Name
	}

	if input.Description != nil {
		org.Description = *input.Description
	}

	if input.MaxMembers != nil {
		if *input.MaxMembers < 1 {
			return nil, fmt.Errorf("max_members must be at least 1")
		}

		// Check if reducing below current member count
		count, err := s.repo.GetMemberCount(ctx, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get member count: %w", err)
		}

		if *input.MaxMembers < count {
			return nil, fmt.Errorf("cannot reduce max_members below current member count (%d)", count)
		}

		org.MaxMembers = *input.MaxMembers
	}

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Organization updated")

	return org, nil
}

// DeleteOrganization deletes an organization (owner only)
func (s *Service) DeleteOrganization(ctx context.Context, orgID string, userID string) error {
	// Get member and check permission
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(member.Role, PermDeleteOrganization) {
		// Log permission denial
		_ = s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "organization",
			ResourceID:     &orgID,
			Details: map[string]interface{}{
				"permission": string(PermDeleteOrganization),
				"role":       string(member.Role),
				"attempted":  "delete_organization",
			},
		})
		return fmt.Errorf("insufficient permissions")
	}

	// Check if there are other members
	members, err := s.repo.GetMembers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}

	if len(members) > 1 {
		return fmt.Errorf("cannot delete organization with other members (remove members first)")
	}

	if err := s.repo.Delete(ctx, orgID); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Organization deleted")

	return nil
}

// ====================================================================
// Member Operations
// ====================================================================

// GetMembers retrieves all members of an organization
func (s *Service) GetMembers(ctx context.Context, orgID string, userID string) ([]*OrganizationMember, error) {
	// Check if user is a member
	if err := s.checkMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	members, err := s.repo.GetMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID string, targetUserID string, actorUserID string, role OrganizationRole) error {
	// Get member and check permission
	actorMember, err := s.repo.GetMember(ctx, orgID, actorUserID)
	if err != nil || actorMember == nil {
		return fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(actorMember.Role, PermUpdateMemberRoles) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         actorUserID,
			Action:         "permission_denied",
			ResourceType:   "member",
			ResourceID:     &targetUserID,
			Details: map[string]interface{}{
				"permission":   string(PermUpdateMemberRoles),
				"role":         string(actorMember.Role),
				"attempted":    "update_member_role",
				"target_user":  targetUserID,
				"desired_role": string(role),
			},
		})
		return fmt.Errorf("insufficient permissions")
	}

	// Cannot change owner's role
	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if targetUserID == org.OwnerID {
		return fmt.Errorf("cannot change owner's role")
	}

	// Admins cannot promote to owner
	if actorMember.Role == RoleAdmin && role == RoleOwner {
		return fmt.Errorf("only owners can assign owner role")
	}

	// Update role
	if err := s.repo.UpdateMemberRole(ctx, orgID, targetUserID, role); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"target_user_id":  targetUserID,
		"actor_user_id":   actorUserID,
		"new_role":        role,
	}).Info("Member role updated")

	return nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID string, targetUserID string, actorUserID string) error {
	// Get member and check permission
	actorMember, err := s.repo.GetMember(ctx, orgID, actorUserID)
	if err != nil || actorMember == nil {
		return fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(actorMember.Role, PermRemoveMembers) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         actorUserID,
			Action:         "permission_denied",
			ResourceType:   "member",
			ResourceID:     &targetUserID,
			Details: map[string]interface{}{
				"permission":  string(PermRemoveMembers),
				"role":        string(actorMember.Role),
				"attempted":   "remove_member",
				"target_user": targetUserID,
			},
		})
		return fmt.Errorf("insufficient permissions")
	}

	// Cannot remove owner
	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if targetUserID == org.OwnerID {
		return fmt.Errorf("cannot remove owner from organization")
	}

	// Get target member info before removal (for email notification)
	targetMember, err := s.repo.GetMember(ctx, orgID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target member: %w", err)
	}

	if actorMember.Role == RoleAdmin {
		if targetMember.Role != RoleMember {
			return fmt.Errorf("admins can only remove members")
		}
	}

	// Remove member
	if err := s.repo.RemoveMember(ctx, orgID, targetUserID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"target_user_id":  targetUserID,
		"actor_user_id":   actorUserID,
	}).Info("Member removed from organization")

	// Send removal notification email (non-blocking, log errors but don't fail)
	if s.emailSvc != nil && targetMember.User != nil {
		go func() {
			err := s.emailSvc.SendMemberRemovedEmail(
				targetMember.User.Email,
				org.Name,
			)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"user_id": targetUserID,
					"org_id":  orgID,
				}).Error("Failed to send member removed email")
			}
		}()
	}

	return nil
}

// ====================================================================
// Invitation Operations
// ====================================================================

// CreateInvitation creates an invitation to join an organization
func (s *Service) CreateInvitation(ctx context.Context, orgID string, actorUserID string, input *CreateInvitationInput) (*OrganizationInvitation, error) {
	// Check rate limits first
	if s.rateLimiter != nil {
		allowed, reason := s.rateLimiter.CheckBothLimits(actorUserID, orgID)
		if !allowed {
			s.logger.WithFields(logrus.Fields{
				"user_id": actorUserID,
				"org_id":  orgID,
				"reason":  reason,
			}).Warn("Invitation rate limit exceeded")
			return nil, fmt.Errorf("rate limit exceeded: %s", reason)
		}
	}

	// Get member and check permission
	actorMember, err := s.repo.GetMember(ctx, orgID, actorUserID)
	if err != nil || actorMember == nil {
		return nil, fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(actorMember.Role, PermInviteMembers) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         actorUserID,
			Action:         "permission_denied",
			ResourceType:   "invitation",
			Details: map[string]interface{}{
				"permission": string(PermInviteMembers),
				"role":       string(actorMember.Role),
				"attempted":  "create_invitation",
				"email":      input.Email,
			},
		})
		return nil, fmt.Errorf("insufficient permissions")
	}

	// Check if organization has reached member limit
	count, err := s.repo.GetMemberCount(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member count: %w", err)
	}

	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if count >= org.MaxMembers {
		return nil, fmt.Errorf("organization has reached maximum member limit (%d)", org.MaxMembers)
	}

	// Validate email
	if !isValidEmail(input.Email) {
		return nil, fmt.Errorf("invalid email address")
	}

	// Admins cannot invite admins
	if actorMember.Role == RoleAdmin && input.Role == RoleAdmin {
		return nil, fmt.Errorf("only owners can invite admins")
	}

	// Check for existing pending invitation for this email in this organization
	normalizedEmail := strings.ToLower(input.Email)
	existingInvitations, err := s.repo.GetInvitationsByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invitations: %w", err)
	}

	for _, existing := range existingInvitations {
		if strings.ToLower(existing.Email) == normalizedEmail && !existing.IsAccepted() && !existing.IsExpired() {
			return nil, fmt.Errorf("invitation already exists for this email")
		}
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create invitation
	invitation := &OrganizationInvitation{
		OrganizationID: orgID,
		Email:          normalizedEmail,
		Role:           input.Role,
		InvitedBy:      actorUserID,
		Token:          token,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.repo.CreateInvitation(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"invitation_id":   invitation.ID,
		"organization_id": orgID,
		"email":           invitation.Email,
		"invited_by":      actorUserID,
	}).Info("Invitation created")

	// Send invitation email (non-blocking, log errors but don't fail)
	if s.emailSvc != nil {
		go func() {
			// Build invitation URL (this would be configurable in production)
			invitationURL := fmt.Sprintf("https://sqlstudio.io/invitations/accept?token=%s", token)

			// Get inviter name from member info
			inviterName := actorMember.User.Email
			if actorMember.User.DisplayName != nil {
				inviterName = *actorMember.User.DisplayName
			} else if actorMember.User.Username != "" {
				inviterName = actorMember.User.Username
			}

			err := s.emailSvc.SendOrganizationInvitationEmail(
				invitation.Email,
				org.Name,
				inviterName,
				string(invitation.Role),
				invitationURL,
			)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"invitation_id": invitation.ID,
					"email":         invitation.Email,
				}).Error("Failed to send invitation email")
			}
		}()
	}

	return invitation, nil
}

// GetInvitations retrieves invitations for an organization
func (s *Service) GetInvitations(ctx context.Context, orgID string, userID string) ([]*OrganizationInvitation, error) {
	// Get member and check permission
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(member.Role, PermInviteMembers) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "invitation",
			Details: map[string]interface{}{
				"permission": string(PermInviteMembers),
				"role":       string(member.Role),
				"attempted":  "view_invitations",
			},
		})
		return nil, fmt.Errorf("insufficient permissions")
	}

	invitations, err := s.repo.GetInvitationsByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}

	return invitations, nil
}

// GetPendingInvitationsForEmail retrieves pending invitations for an email
func (s *Service) GetPendingInvitationsForEmail(ctx context.Context, email string) ([]*OrganizationInvitation, error) {
	invitations, err := s.repo.GetInvitationsByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}

	return invitations, nil
}

// AcceptInvitation accepts an invitation
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID string) (*Organization, error) {
	// Get invitation
	invitation, err := s.repo.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invitation not found or expired")
	}

	// Check if already accepted
	if invitation.IsAccepted() {
		return nil, fmt.Errorf("invitation already accepted")
	}

	// Check if expired
	if invitation.IsExpired() {
		return nil, fmt.Errorf("invitation has expired")
	}

	// Check if organization still exists
	if invitation.Organization.DeletedAt != nil {
		return nil, fmt.Errorf("organization no longer exists")
	}

	// Check if user is already a member
	existing, err := s.repo.GetMember(ctx, invitation.OrganizationID, userID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("already a member of this organization")
	}

	// Check member limit
	count, err := s.repo.GetMemberCount(ctx, invitation.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member count: %w", err)
	}

	if count >= invitation.Organization.MaxMembers {
		return nil, fmt.Errorf("organization has reached maximum member limit")
	}

	// Add member
	member := &OrganizationMember{
		OrganizationID: invitation.OrganizationID,
		UserID:         userID,
		Role:           invitation.Role,
		InvitedBy:      &invitation.InvitedBy,
		JoinedAt:       time.Now(),
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Mark invitation as accepted
	now := time.Now()
	invitation.AcceptedAt = &now
	if err := s.repo.UpdateInvitation(ctx, invitation); err != nil {
		s.logger.WithError(err).Warn("Failed to update invitation status")
	}

	s.logger.WithFields(logrus.Fields{
		"invitation_id":   invitation.ID,
		"organization_id": invitation.OrganizationID,
		"user_id":         userID,
	}).Info("Invitation accepted")

	// Send welcome email (non-blocking, log errors but don't fail)
	if s.emailSvc != nil {
		go func() {
			// Get member info to get user email and name
			newMember, err := s.repo.GetMember(context.Background(), invitation.OrganizationID, userID)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to get member info for welcome email")
				return
			}

			userName := newMember.User.Email
			if newMember.User.DisplayName != nil {
				userName = *newMember.User.DisplayName
			} else if newMember.User.Username != "" {
				userName = newMember.User.Username
			}

			err = s.emailSvc.SendOrganizationWelcomeEmail(
				newMember.User.Email,
				userName,
				invitation.Organization.Name,
				string(invitation.Role),
			)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"user_id": userID,
					"org_id":  invitation.OrganizationID,
				}).Error("Failed to send welcome email")
			}
		}()
	}

	return invitation.Organization, nil
}

// DeclineInvitation declines an invitation
func (s *Service) DeclineInvitation(ctx context.Context, token string) error {
	// Get invitation
	invitation, err := s.repo.GetInvitationByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invitation not found")
	}

	// Delete invitation
	if err := s.repo.DeleteInvitation(ctx, invitation.ID); err != nil {
		return fmt.Errorf("failed to decline invitation: %w", err)
	}

	s.logger.WithField("invitation_id", invitation.ID).Info("Invitation declined")
	return nil
}

// RevokeInvitation revokes an invitation
func (s *Service) RevokeInvitation(ctx context.Context, orgID string, invitationID string, userID string) error {
	// Get member and check permission
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(member.Role, PermInviteMembers) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "invitation",
			ResourceID:     &invitationID,
			Details: map[string]interface{}{
				"permission": string(PermInviteMembers),
				"role":       string(member.Role),
				"attempted":  "revoke_invitation",
			},
		})
		return fmt.Errorf("insufficient permissions")
	}

	// Get invitation to verify it belongs to this org
	invitation, err := s.repo.GetInvitation(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("invitation not found")
	}

	if invitation.OrganizationID != orgID {
		return fmt.Errorf("invitation does not belong to this organization")
	}

	// Delete invitation
	if err := s.repo.DeleteInvitation(ctx, invitationID); err != nil {
		return fmt.Errorf("failed to revoke invitation: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"invitation_id":   invitationID,
		"organization_id": orgID,
		"revoked_by":      userID,
	}).Info("Invitation revoked")

	return nil
}

// ====================================================================
// Audit Log Operations
// ====================================================================

// GetAuditLogs retrieves audit logs for an organization
func (s *Service) GetAuditLogs(ctx context.Context, orgID string, userID string, limit, offset int) ([]*AuditLog, error) {
	// Get member and check permission
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("not a member of this organization")
	}

	if !HasPermission(member.Role, PermViewAuditLogs) {
		// Log permission denial
		s.CreateAuditLog(ctx, &AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "audit_log",
			Details: map[string]interface{}{
				"permission": string(PermViewAuditLogs),
				"role":       string(member.Role),
				"attempted":  "view_audit_logs",
			},
		})
		return nil, fmt.Errorf("insufficient permissions")
	}

	logs, err := s.repo.GetAuditLogs(ctx, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, nil
}

// CreateAuditLog creates an audit log entry
func (s *Service) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	if err := s.repo.CreateAuditLog(ctx, log); err != nil {
		s.logger.WithError(err).Warn("Failed to create audit log")
		// Don't fail the request if audit logging fails
	}
	return nil
}

// ====================================================================
// Helper Functions
// ====================================================================

// checkMembership checks if a user is a member of an organization
func (s *Service) checkMembership(ctx context.Context, orgID string, userID string) error {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("not a member of this organization")
	}
	return nil
}

// checkPermission checks if a user has one of the required roles
func (s *Service) checkPermission(ctx context.Context, orgID string, userID string, allowedRoles []OrganizationRole) error {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("not a member of this organization")
	}

	for _, role := range allowedRoles {
		if member.Role == role {
			return nil
		}
	}

	return fmt.Errorf("insufficient permissions")
}

// validateOrganizationName validates an organization name
func (s *Service) validateOrganizationName(name string) error {
	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return fmt.Errorf("organization name must be at least 3 characters")
	}

	if len(name) > 50 {
		return fmt.Errorf("organization name must be at most 50 characters")
	}

	// Allow alphanumeric, spaces, hyphens, and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("organization name can only contain letters, numbers, spaces, hyphens, and underscores")
	}

	return nil
}

// isValidEmail validates an email address
func isValidEmail(email string) bool {
	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
