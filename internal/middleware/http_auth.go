package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// HTTPAuthMiddleware provides JWT authentication for HTTP handlers
// It extracts the JWT token from the Authorization header, validates it,
// and adds user information to the request context
func HTTPAuthMiddleware(authMiddleware *AuthMiddleware, logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.WithField("path", r.URL.Path).Warn("Missing authorization header")
				http.Error(w, `{"error": true, "message": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check if it's a Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				logger.WithField("path", r.URL.Path).Warn("Invalid authorization header format")
				http.Error(w, `{"error": true, "message": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]
			if token == "" {
				logger.WithField("path", r.URL.Path).Warn("Empty authorization token")
				http.Error(w, `{"error": true, "message": "empty authorization token"}`, http.StatusUnauthorized)
				return
			}

			// Validate the token
			claims, err := authMiddleware.validateToken(token)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"path":  r.URL.Path,
					"error": err.Error(),
				}).Warn("Token validation failed")
				http.Error(w, `{"error": true, "message": "invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Add user information to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Log successful authentication
			logger.WithFields(logrus.Fields{
				"path":    r.URL.Path,
				"user_id": claims.UserID,
				"method":  r.Method,
			}).Debug("Request authenticated")

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalHTTPAuthMiddleware provides optional JWT authentication for HTTP handlers
// If a token is provided, it validates it and adds user info to context
// If no token is provided, it allows the request to proceed without authentication
func OptionalHTTPAuthMiddleware(authMiddleware *AuthMiddleware, logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth header, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Check if it's a Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				// Invalid format, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			token := parts[1]
			if token == "" {
				// Empty token, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Validate the token
			claims, err := authMiddleware.validateToken(token)
			if err != nil {
				// Invalid token, continue without authentication
				logger.WithFields(logrus.Fields{
					"path":  r.URL.Path,
					"error": err.Error(),
				}).Debug("Optional auth: token validation failed")
				next.ServeHTTP(w, r)
				return
			}

			// Add user information to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Log successful authentication
			logger.WithFields(logrus.Fields{
				"path":    r.URL.Path,
				"user_id": claims.UserID,
				"method":  r.Method,
			}).Debug("Optional auth: request authenticated")

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext extracts the user ID from the request context
// Returns empty string if not found
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return ""
	}
	return userID
}

// GetUsernameFromContext extracts the username from the request context
// Returns empty string if not found
func GetUsernameFromContext(ctx context.Context) string {
	username, ok := ctx.Value("username").(string)
	if !ok {
		return ""
	}
	return username
}

// GetRoleFromContext extracts the role from the request context
// Returns empty string if not found
func GetRoleFromContext(ctx context.Context) string {
	role, ok := ctx.Value("role").(string)
	if !ok {
		return ""
	}
	return role
}
