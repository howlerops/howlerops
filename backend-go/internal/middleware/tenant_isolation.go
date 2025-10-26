package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	// Context keys
	UserOrganizationsKey contextKey = "user_organizations"
	CurrentOrgIDKey      contextKey = "current_org_id"
)

// TenantIsolationMiddleware ensures all database queries are scoped to the user's organizations
type TenantIsolationMiddleware struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTenantIsolationMiddleware creates a new tenant isolation middleware
func NewTenantIsolationMiddleware(db *sql.DB, logger *logrus.Logger) *TenantIsolationMiddleware {
	return &TenantIsolationMiddleware{
		db:     db,
		logger: logger,
	}
}

// EnforceTenantIsolation ensures all requests have organization context
func (m *TenantIsolationMiddleware) EnforceTenantIsolation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := getTenantUserIDFromContext(r.Context())
		if userID == "" {
			m.logger.Warn("Tenant isolation: no user ID in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user's organizations
		orgs, err := m.getUserOrganizations(r.Context(), userID)
		if err != nil {
			m.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user organizations")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(orgs) == 0 {
			m.logger.WithField("user_id", userID).Warn("User has no organizations")
			http.Error(w, "User has no organizations", http.StatusForbidden)
			return
		}

		// Add to context for query filtering
		ctx := context.WithValue(r.Context(), UserOrganizationsKey, orgs)

		// Set primary organization as current (first one, or from header/query param)
		currentOrgID := m.determineCurrentOrg(r, orgs)
		ctx = context.WithValue(ctx, CurrentOrgIDKey, currentOrgID)

		m.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"organizations":  len(orgs),
			"current_org_id": currentOrgID,
		}).Debug("Tenant isolation applied")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserOrganizations fetches all organizations the user belongs to
func (m *TenantIsolationMiddleware) getUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT o.id
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = ?
	`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user organizations: %w", err)
	}
	defer rows.Close()

	var orgIDs []string
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgIDs = append(orgIDs, orgID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}

	return orgIDs, nil
}

// determineCurrentOrg determines which organization context to use
func (m *TenantIsolationMiddleware) determineCurrentOrg(r *http.Request, orgs []string) string {
	// 1. Check X-Organization-ID header
	if orgID := r.Header.Get("X-Organization-ID"); orgID != "" {
		// Verify user has access to this org
		for _, o := range orgs {
			if o == orgID {
				return orgID
			}
		}
	}

	// 2. Check query parameter
	if orgID := r.URL.Query().Get("org_id"); orgID != "" {
		for _, o := range orgs {
			if o == orgID {
				return orgID
			}
		}
	}

	// 3. Default to first organization
	return orgs[0]
}

// GetUserOrganizationIDs extracts organization IDs from context
func GetUserOrganizationIDs(ctx context.Context) []string {
	orgs, ok := ctx.Value(UserOrganizationsKey).([]string)
	if !ok {
		return []string{}
	}
	return orgs
}

// GetCurrentOrgID gets the current organization ID from context
func GetCurrentOrgID(ctx context.Context) string {
	orgID, ok := ctx.Value(CurrentOrgIDKey).(string)
	if !ok {
		return ""
	}
	return orgID
}

// VerifyOrgAccess checks if user has access to specified organization
func VerifyOrgAccess(ctx context.Context, orgID string) error {
	userOrgs := GetUserOrganizationIDs(ctx)
	for _, o := range userOrgs {
		if o == orgID {
			return nil
		}
	}
	return fmt.Errorf("access denied: user not member of organization %s", orgID)
}

// BuildOrgFilterQuery creates SQL WHERE clause for organization filtering
func BuildOrgFilterQuery(ctx context.Context, orgColumnName string) (string, []interface{}) {
	orgs := GetUserOrganizationIDs(ctx)
	if len(orgs) == 0 {
		return fmt.Sprintf("%s = ?", orgColumnName), []interface{}{"__no_access__"}
	}

	if len(orgs) == 1 {
		return fmt.Sprintf("%s = ?", orgColumnName), []interface{}{orgs[0]}
	}

	// Multiple organizations: use IN clause
	placeholders := ""
	args := make([]interface{}, len(orgs))
	for i, org := range orgs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = org
	}

	return fmt.Sprintf("%s IN (%s)", orgColumnName, placeholders), args
}

// getTenantUserIDFromContext extracts user ID from context (set by auth middleware)
func getTenantUserIDFromContext(ctx context.Context) string {
	// This should be set by your authentication middleware
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return ""
	}
	return userID
}
