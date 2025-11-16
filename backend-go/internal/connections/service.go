package connections

import (
	"context"
	"fmt"

	"github.com/jbeck018/howlerops/backend-go/internal/organization"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
	"github.com/sirupsen/logrus"
)

// ConnectionStore defines the interface for connection storage operations
type ConnectionStore interface {
	Create(ctx context.Context, conn *turso.Connection) error
	GetByID(ctx context.Context, id string) (*turso.Connection, error)
	GetByUserID(ctx context.Context, userID string) ([]*turso.Connection, error)
	GetConnectionsByOrganization(ctx context.Context, orgID string) ([]*turso.Connection, error)
	GetSharedConnections(ctx context.Context, userID string) ([]*turso.Connection, error)
	Update(ctx context.Context, conn *turso.Connection) error
	UpdateConnectionVisibility(ctx context.Context, connID, userID string, visibility string) error
	Delete(ctx context.Context, id string) error
}

// Service handles business logic for connections
type Service struct {
	store   ConnectionStore
	orgRepo organization.Repository
	logger  *logrus.Logger
}

// NewService creates a new connections service
func NewService(store ConnectionStore, orgRepo organization.Repository, logger *logrus.Logger) *Service {
	return &Service{
		store:   store,
		orgRepo: orgRepo,
		logger:  logger,
	}
}

// ShareConnection changes a connection's visibility to 'shared' in an organization
// Validates: user has connections:update permission in the org
func (s *Service) ShareConnection(ctx context.Context, connID, userID, orgID string) error {
	// Get connection to verify it exists and belongs to user
	conn, err := s.store.GetByID(ctx, connID)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}

	// Verify user created this connection
	if conn.CreatedBy != userID {
		return fmt.Errorf("only the creator can share this connection")
	}

	// Verify user is a member of the organization
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("user not member of organization")
	}

	// Check permission
	if !organization.HasPermission(member.Role, organization.PermUpdateConnections) {
		// Log permission denial
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "connection",
			ResourceID:     &connID,
			Details: map[string]interface{}{
				"permission": string(organization.PermUpdateConnections),
				"role":       string(member.Role),
				"attempted":  "share_connection",
			},
		})
		return fmt.Errorf("insufficient permissions to share connections")
	}

	// Update connection to be shared in organization
	conn.OrganizationID = &orgID
	conn.Visibility = "shared"

	if err := s.store.Update(ctx, conn); err != nil {
		return fmt.Errorf("failed to share connection: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, &organization.AuditLog{
		OrganizationID: &orgID,
		UserID:         userID,
		Action:         "share_connection",
		ResourceType:   "connection",
		ResourceID:     &connID,
		Details: map[string]interface{}{
			"visibility":      "shared",
			"connection_name": conn.Name,
			"connection_type": conn.Type,
		},
	})

	s.logger.WithFields(logrus.Fields{
		"connection_id":   connID,
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Connection shared with organization")

	return nil
}

// UnshareConnection changes visibility back to 'personal'
func (s *Service) UnshareConnection(ctx context.Context, connID, userID string) error {
	// Get connection to verify it exists
	conn, err := s.store.GetByID(ctx, connID)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}

	// Verify user created this connection
	if conn.CreatedBy != userID {
		return fmt.Errorf("only the creator can unshare this connection")
	}

	// If connection is shared in an org, check permission
	if conn.OrganizationID != nil && conn.Visibility == "shared" {
		member, err := s.orgRepo.GetMember(ctx, *conn.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		if !organization.HasPermission(member.Role, organization.PermUpdateConnections) {
			return fmt.Errorf("insufficient permissions to unshare connections")
		}

		// Create audit log for unsharing
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: conn.OrganizationID,
			UserID:         userID,
			Action:         "unshare_connection",
			ResourceType:   "connection",
			ResourceID:     &connID,
			Details: map[string]interface{}{
				"visibility":      "personal",
				"connection_name": conn.Name,
			},
		})
	}

	// Update connection to be personal
	conn.OrganizationID = nil
	conn.Visibility = "personal"

	if err := s.store.Update(ctx, conn); err != nil {
		return fmt.Errorf("failed to unshare connection: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connID,
		"user_id":       userID,
	}).Info("Connection unshared")

	return nil
}

// GetAccessibleConnections returns all connections the user can access
// Personal + shared in user's organizations
func (s *Service) GetAccessibleConnections(ctx context.Context, userID string) ([]*turso.Connection, error) {
	connections, err := s.store.GetSharedConnections(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible connections: %w", err)
	}

	return connections, nil
}

// GetOrganizationConnections returns all shared connections in an organization
// Validates: user is a member of the organization
func (s *Service) GetOrganizationConnections(ctx context.Context, orgID, userID string) ([]*turso.Connection, error) {
	// Verify user is a member
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("user not member of organization")
	}

	// Check permission to view connections
	if !organization.HasPermission(member.Role, organization.PermViewConnections) {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "connection",
			Details: map[string]interface{}{
				"permission": string(organization.PermViewConnections),
				"role":       string(member.Role),
				"attempted":  "view_org_connections",
			},
		})
		return nil, fmt.Errorf("insufficient permissions to view connections")
	}

	connections, err := s.store.GetConnectionsByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization connections: %w", err)
	}

	return connections, nil
}

// CreateConnection creates a new connection
func (s *Service) CreateConnection(ctx context.Context, conn *turso.Connection, userID string) error {
	conn.UserID = userID
	conn.CreatedBy = userID

	// If organization is specified, validate membership and permissions
	if conn.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *conn.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		if !organization.HasPermission(member.Role, organization.PermCreateConnections) {
			return fmt.Errorf("insufficient permissions to create connections in organization")
		}
	}

	if err := s.store.Create(ctx, conn); err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}

	// Create audit log if in organization
	if conn.OrganizationID != nil {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: conn.OrganizationID,
			UserID:         userID,
			Action:         "create_connection",
			ResourceType:   "connection",
			ResourceID:     &conn.ID,
			Details: map[string]interface{}{
				"connection_name": conn.Name,
				"connection_type": conn.Type,
				"visibility":      conn.Visibility,
			},
		})
	}

	return nil
}

// UpdateConnection updates a connection with permission checks
func (s *Service) UpdateConnection(ctx context.Context, conn *turso.Connection, userID string) error {
	// Get existing connection
	existing, err := s.store.GetByID(ctx, conn.ID)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}

	// Check if user can update
	if existing.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *existing.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Check permission based on role and ownership
		if !organization.CanUpdateResource(member.Role, existing.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to update this connection")
		}
	} else if existing.UserID != userID {
		return fmt.Errorf("cannot update another user's personal connection")
	}

	if err := s.store.Update(ctx, conn); err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}

	// Create audit log if in organization
	if conn.OrganizationID != nil {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: conn.OrganizationID,
			UserID:         userID,
			Action:         "update_connection",
			ResourceType:   "connection",
			ResourceID:     &conn.ID,
			Details: map[string]interface{}{
				"connection_name": conn.Name,
			},
		})
	}

	return nil
}

// DeleteConnection deletes a connection with permission checks
func (s *Service) DeleteConnection(ctx context.Context, connID, userID string) error {
	// Get existing connection
	existing, err := s.store.GetByID(ctx, connID)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}

	// Check if user can delete
	if existing.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *existing.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Check permission based on role and ownership
		if !organization.CanDeleteResource(member.Role, existing.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to delete this connection")
		}

		// Create audit log
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: existing.OrganizationID,
			UserID:         userID,
			Action:         "delete_connection",
			ResourceType:   "connection",
			ResourceID:     &connID,
			Details: map[string]interface{}{
				"connection_name": existing.Name,
			},
		})
	} else if existing.UserID != userID {
		return fmt.Errorf("cannot delete another user's personal connection")
	}

	if err := s.store.Delete(ctx, connID); err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

// createAuditLog is a helper to create audit logs (non-blocking)
func (s *Service) createAuditLog(ctx context.Context, log *organization.AuditLog) {
	if err := s.orgRepo.CreateAuditLog(ctx, log); err != nil {
		s.logger.WithError(err).Warn("Failed to create audit log")
	}
}
