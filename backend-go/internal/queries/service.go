package queries

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/internal/organization"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// QueryStore defines the interface for query storage operations
type QueryStore interface {
	Create(ctx context.Context, query *turso.SavedQuery) error
	GetByID(ctx context.Context, id string) (*turso.SavedQuery, error)
	GetByUserID(ctx context.Context, userID string) ([]*turso.SavedQuery, error)
	GetQueriesByOrganization(ctx context.Context, orgID string) ([]*turso.SavedQuery, error)
	GetSharedQueries(ctx context.Context, userID string) ([]*turso.SavedQuery, error)
	Update(ctx context.Context, query *turso.SavedQuery) error
	UpdateQueryVisibility(ctx context.Context, queryID, userID string, visibility string) error
	Delete(ctx context.Context, id string) error
}

// Service handles business logic for saved queries
type Service struct {
	store   QueryStore
	orgRepo organization.Repository
	logger  *logrus.Logger
}

// NewService creates a new queries service
func NewService(store QueryStore, orgRepo organization.Repository, logger *logrus.Logger) *Service {
	return &Service{
		store:   store,
		orgRepo: orgRepo,
		logger:  logger,
	}
}

// ShareQuery changes a query's visibility to 'shared' in an organization
// Validates: user has queries:update permission in the org
func (s *Service) ShareQuery(ctx context.Context, queryID, userID, orgID string) error {
	// Get query to verify it exists and belongs to user
	query, err := s.store.GetByID(ctx, queryID)
	if err != nil {
		return fmt.Errorf("query not found: %w", err)
	}

	// Verify user created this query
	if query.CreatedBy != userID {
		return fmt.Errorf("only the creator can share this query")
	}

	// Verify user is a member of the organization
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return fmt.Errorf("user not member of organization")
	}

	// Check permission
	if !organization.HasPermission(member.Role, organization.PermUpdateQueries) {
		// Log permission denial
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "query",
			ResourceID:     &queryID,
			Details: map[string]interface{}{
				"permission": string(organization.PermUpdateQueries),
				"role":       string(member.Role),
				"attempted":  "share_query",
			},
		})
		return fmt.Errorf("insufficient permissions to share queries")
	}

	// Update query to be shared in organization
	query.OrganizationID = &orgID
	query.Visibility = "shared"

	if err := s.store.Update(ctx, query); err != nil {
		return fmt.Errorf("failed to share query: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, &organization.AuditLog{
		OrganizationID: &orgID,
		UserID:         userID,
		Action:         "share_query",
		ResourceType:   "query",
		ResourceID:     &queryID,
		Details: map[string]interface{}{
			"visibility":  "shared",
			"query_name":  query.Name,
			"is_favorite": query.Favorite,
		},
	})

	s.logger.WithFields(logrus.Fields{
		"query_id":        queryID,
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Query shared with organization")

	return nil
}

// UnshareQuery changes visibility back to 'personal'
func (s *Service) UnshareQuery(ctx context.Context, queryID, userID string) error {
	// Get query to verify it exists
	query, err := s.store.GetByID(ctx, queryID)
	if err != nil {
		return fmt.Errorf("query not found: %w", err)
	}

	// Verify user created this query
	if query.CreatedBy != userID {
		return fmt.Errorf("only the creator can unshare this query")
	}

	// If query is shared in an org, check permission
	if query.OrganizationID != nil && query.Visibility == "shared" {
		member, err := s.orgRepo.GetMember(ctx, *query.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		if !organization.HasPermission(member.Role, organization.PermUpdateQueries) {
			return fmt.Errorf("insufficient permissions to unshare queries")
		}

		// Create audit log for unsharing
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: query.OrganizationID,
			UserID:         userID,
			Action:         "unshare_query",
			ResourceType:   "query",
			ResourceID:     &queryID,
			Details: map[string]interface{}{
				"visibility": "personal",
				"query_name": query.Name,
			},
		})
	}

	// Update query to be personal
	query.OrganizationID = nil
	query.Visibility = "personal"

	if err := s.store.Update(ctx, query); err != nil {
		return fmt.Errorf("failed to unshare query: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"query_id": queryID,
		"user_id":  userID,
	}).Info("Query unshared")

	return nil
}

// GetAccessibleQueries returns all queries the user can access
// Personal + shared in user's organizations
func (s *Service) GetAccessibleQueries(ctx context.Context, userID string) ([]*turso.SavedQuery, error) {
	queries, err := s.store.GetSharedQueries(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible queries: %w", err)
	}

	return queries, nil
}

// GetOrganizationQueries returns all shared queries in an organization
// Validates: user is a member of the organization
func (s *Service) GetOrganizationQueries(ctx context.Context, orgID, userID string) ([]*turso.SavedQuery, error) {
	// Verify user is a member
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("user not member of organization")
	}

	// Check permission to view queries
	if !organization.HasPermission(member.Role, organization.PermViewQueries) {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         "permission_denied",
			ResourceType:   "query",
			Details: map[string]interface{}{
				"permission": string(organization.PermViewQueries),
				"role":       string(member.Role),
				"attempted":  "view_org_queries",
			},
		})
		return nil, fmt.Errorf("insufficient permissions to view queries")
	}

	queries, err := s.store.GetQueriesByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization queries: %w", err)
	}

	return queries, nil
}

// CreateQuery creates a new saved query
func (s *Service) CreateQuery(ctx context.Context, query *turso.SavedQuery, userID string) error {
	query.UserID = userID
	query.CreatedBy = userID

	// If organization is specified, validate membership and permissions
	if query.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *query.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		if !organization.HasPermission(member.Role, organization.PermCreateQueries) {
			return fmt.Errorf("insufficient permissions to create queries in organization")
		}
	}

	if err := s.store.Create(ctx, query); err != nil {
		return fmt.Errorf("failed to create query: %w", err)
	}

	// Create audit log if in organization
	if query.OrganizationID != nil {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: query.OrganizationID,
			UserID:         userID,
			Action:         "create_query",
			ResourceType:   "query",
			ResourceID:     &query.ID,
			Details: map[string]interface{}{
				"query_name":  query.Name,
				"visibility":  query.Visibility,
				"is_favorite": query.Favorite,
			},
		})
	}

	return nil
}

// UpdateQuery updates a saved query with permission checks
func (s *Service) UpdateQuery(ctx context.Context, query *turso.SavedQuery, userID string) error {
	// Get existing query
	existing, err := s.store.GetByID(ctx, query.ID)
	if err != nil {
		return fmt.Errorf("query not found: %w", err)
	}

	// Check if user can update
	if existing.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *existing.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Check permission based on role and ownership
		if !organization.CanUpdateResource(member.Role, existing.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to update this query")
		}
	} else if existing.UserID != userID {
		return fmt.Errorf("cannot update another user's personal query")
	}

	if err := s.store.Update(ctx, query); err != nil {
		return fmt.Errorf("failed to update query: %w", err)
	}

	// Create audit log if in organization
	if query.OrganizationID != nil {
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: query.OrganizationID,
			UserID:         userID,
			Action:         "update_query",
			ResourceType:   "query",
			ResourceID:     &query.ID,
			Details: map[string]interface{}{
				"query_name": query.Name,
			},
		})
	}

	return nil
}

// DeleteQuery deletes a saved query with permission checks
func (s *Service) DeleteQuery(ctx context.Context, queryID, userID string) error {
	// Get existing query
	existing, err := s.store.GetByID(ctx, queryID)
	if err != nil {
		return fmt.Errorf("query not found: %w", err)
	}

	// Check if user can delete
	if existing.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *existing.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Check permission based on role and ownership
		if !organization.CanDeleteResource(member.Role, existing.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to delete this query")
		}

		// Create audit log
		s.createAuditLog(ctx, &organization.AuditLog{
			OrganizationID: existing.OrganizationID,
			UserID:         userID,
			Action:         "delete_query",
			ResourceType:   "query",
			ResourceID:     &queryID,
			Details: map[string]interface{}{
				"query_name": existing.Name,
			},
		})
	} else if existing.UserID != userID {
		return fmt.Errorf("cannot delete another user's personal query")
	}

	if err := s.store.Delete(ctx, queryID); err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}

	return nil
}

// createAuditLog is a helper to create audit logs (non-blocking)
func (s *Service) createAuditLog(ctx context.Context, log *organization.AuditLog) {
	if err := s.orgRepo.CreateAuditLog(ctx, log); err != nil {
		s.logger.WithError(err).Warn("Failed to create audit log")
	}
}
