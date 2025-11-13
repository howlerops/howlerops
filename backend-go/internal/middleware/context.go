package middleware

// contextKey is the type for context keys used in middleware
type contextKey string

const (
	// User authentication context keys
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
	roleKey     contextKey = "role"

	// Tenant isolation context keys
	UserOrganizationsKey contextKey = "user_organizations"
	CurrentOrgIDKey      contextKey = "current_org_id"
)
