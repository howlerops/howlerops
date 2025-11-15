package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

// newTestAuthMiddleware creates an AuthMiddleware for testing
func newTestAuthMiddleware() *AuthMiddleware {
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during tests
	return NewAuthMiddleware("test-secret-key-must-be-at-least-32-chars-long!!", logger)
}

// newTestLogger creates a logger for testing
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during tests
	return logger
}

// generateValidToken generates a valid JWT token for testing
func generateValidToken(authMW *AuthMiddleware) string {
	token, err := authMW.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	if err != nil {
		panic(err)
	}
	return token
}

// generateExpiredToken generates an expired JWT token for testing
func generateExpiredToken(authMW *AuthMiddleware) string {
	token, err := authMW.GenerateToken("user123", "testuser", "admin", -1*time.Hour)
	if err != nil {
		panic(err)
	}
	return token
}

// testHandler is a simple handler that returns 200 OK
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// contextCapturingHandler captures the context for verification
func contextCapturingHandler(capturedCtx *context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// HTTPAuthMiddleware Tests

func TestHTTPAuthMiddleware_ValidToken(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()
	token := generateValidToken(authMW)

	// Create middleware and handler
	middleware := HTTPAuthMiddleware(authMW, logger)
	handler := middleware(testHandler())

	// Create request with valid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestHTTPAuthMiddleware_MissingHeader(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	// Create middleware and handler
	middleware := HTTPAuthMiddleware(authMW, logger)
	handler := middleware(testHandler())

	// Create request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing authorization header")
}

func TestHTTPAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	tests := []struct {
		name        string
		authHeader  string
		description string
	}{
		{
			name:        "NoBearer",
			authHeader:  "abc123",
			description: "Token without Bearer prefix",
		},
		{
			name:        "WrongScheme",
			authHeader:  "Basic abc123",
			description: "Wrong authentication scheme",
		},
		{
			name:        "OnlyBearer",
			authHeader:  "Bearer",
			description: "Bearer without token",
		},
		{
			name:        "BearerWithSpace",
			authHeader:  "Bearer ",
			description: "Bearer with only space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware and handler
			middleware := HTTPAuthMiddleware(authMW, logger)
			handler := middleware(testHandler())

			// Create request with invalid header format
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			body := rec.Body.String()
			// Should be either "invalid authorization header format" or "empty authorization token"
			assert.True(t,
				strings.Contains(body, "invalid authorization header format") ||
					strings.Contains(body, "empty authorization token"),
				"Expected error message about invalid format or empty token, got: %s", body)
		})
	}
}

func TestHTTPAuthMiddleware_ExpiredToken(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()
	token := generateExpiredToken(authMW)

	// Create middleware and handler
	middleware := HTTPAuthMiddleware(authMW, logger)
	handler := middleware(testHandler())

	// Create request with expired token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired token")
}

func TestHTTPAuthMiddleware_InvalidToken(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	tests := []struct {
		name        string
		token       string
		description string
	}{
		{
			name:        "MalformedToken",
			token:       "not.a.jwt",
			description: "Malformed JWT token",
		},
		{
			name:        "RandomString",
			token:       "random-string-not-jwt",
			description: "Random string as token",
		},
		{
			name:        "WrongSignature",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdCJ9.wrong_signature",
			description: "Valid JWT structure but wrong signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware and handler
			middleware := HTTPAuthMiddleware(authMW, logger)
			handler := middleware(testHandler())

			// Create request with invalid token
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "invalid or expired token")
		})
	}
}

func TestHTTPAuthMiddleware_ContextPropagation(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()
	token := generateValidToken(authMW)

	// Create middleware with context-capturing handler
	var capturedCtx context.Context
	middleware := HTTPAuthMiddleware(authMW, logger)
	handler := middleware(contextCapturingHandler(&capturedCtx))

	// Create request with valid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify context values using ContextKey (as set by middleware)
	userID, ok := capturedCtx.Value(UserIDKey).(string)
	require.True(t, ok, "UserID should be in context")
	assert.Equal(t, "user123", userID, "UserID should be set in context")

	username, ok := capturedCtx.Value(UsernameKey).(string)
	require.True(t, ok, "Username should be in context")
	assert.Equal(t, "testuser", username, "Username should be set in context")

	role, ok := capturedCtx.Value(RoleKey).(string)
	require.True(t, ok, "Role should be in context")
	assert.Equal(t, "admin", role, "Role should be set in context")
}

// OptionalHTTPAuthMiddleware Tests

func TestOptionalHTTPAuthMiddleware_NoAuth(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	// Create middleware and handler
	middleware := OptionalHTTPAuthMiddleware(authMW, logger)
	handler := middleware(testHandler())

	// Create request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response - should allow through
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestOptionalHTTPAuthMiddleware_InvalidToken(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	tests := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "MalformedToken",
			authHeader: "Bearer not.a.jwt",
		},
		{
			name:       "ExpiredToken",
			authHeader: "Bearer " + generateExpiredToken(authMW),
		},
		{
			name:       "InvalidFormat",
			authHeader: "Basic xyz123",
		},
		{
			name:       "EmptyToken",
			authHeader: "Bearer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware with context-capturing handler
			var capturedCtx context.Context
			middleware := OptionalHTTPAuthMiddleware(authMW, logger)
			handler := middleware(contextCapturingHandler(&capturedCtx))

			// Create request with invalid token
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response - should allow through
			assert.Equal(t, http.StatusOK, rec.Code)

			// Verify context has no user info (check with ContextKey)
			userID, ok := capturedCtx.Value(UserIDKey).(string)
			assert.False(t, ok, "UserID should not be set for invalid token")
			assert.Empty(t, userID, "UserID should be empty for invalid token")
		})
	}
}

func TestOptionalHTTPAuthMiddleware_ValidToken(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()
	token := generateValidToken(authMW)

	// Create middleware with context-capturing handler
	var capturedCtx context.Context
	middleware := OptionalHTTPAuthMiddleware(authMW, logger)
	handler := middleware(contextCapturingHandler(&capturedCtx))

	// Create request with valid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify context values are set (using ContextKey)
	userID, ok := capturedCtx.Value(UserIDKey).(string)
	require.True(t, ok, "UserID should be in context")
	assert.Equal(t, "user123", userID, "UserID should be set in context")

	username, ok := capturedCtx.Value(UsernameKey).(string)
	require.True(t, ok, "Username should be in context")
	assert.Equal(t, "testuser", username, "Username should be set in context")

	role, ok := capturedCtx.Value(RoleKey).(string)
	require.True(t, ok, "Role should be in context")
	assert.Equal(t, "admin", role, "Role should be set in context")
}

// Context Extraction Function Tests

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "ValidUserID",
			ctx:      context.WithValue(context.Background(), "user_id", "user123"),
			expected: "user123",
		},
		{
			name:     "MissingValue",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "WrongType",
			ctx:      context.WithValue(context.Background(), "user_id", 123),
			expected: "",
		},
		{
			name:     "EmptyString",
			ctx:      context.WithValue(context.Background(), "user_id", ""),
			expected: "",
		},
		{
			name:     "WrongKey",
			ctx:      context.WithValue(context.Background(), "wrong_key", "user123"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserIDFromContext(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUsernameFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "ValidUsername",
			ctx:      context.WithValue(context.Background(), "username", "testuser"),
			expected: "testuser",
		},
		{
			name:     "MissingValue",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "WrongType",
			ctx:      context.WithValue(context.Background(), "username", 456),
			expected: "",
		},
		{
			name:     "EmptyString",
			ctx:      context.WithValue(context.Background(), "username", ""),
			expected: "",
		},
		{
			name:     "WrongKey",
			ctx:      context.WithValue(context.Background(), "wrong_key", "testuser"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUsernameFromContext(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRoleFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "ValidRole",
			ctx:      context.WithValue(context.Background(), "role", "admin"),
			expected: "admin",
		},
		{
			name:     "UserRole",
			ctx:      context.WithValue(context.Background(), "role", "user"),
			expected: "user",
		},
		{
			name:     "MissingValue",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "WrongType",
			ctx:      context.WithValue(context.Background(), "role", 789),
			expected: "",
		},
		{
			name:     "EmptyString",
			ctx:      context.WithValue(context.Background(), "role", ""),
			expected: "",
		},
		{
			name:     "WrongKey",
			ctx:      context.WithValue(context.Background(), "wrong_key", "admin"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRoleFromContext(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration Tests

func TestHTTPAuthMiddleware_Integration_FullFlow(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()

	// Generate different tokens
	adminToken := generateValidToken(authMW)
	userToken, err := authMW.GenerateToken("user456", "regularuser", "user", 1*time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedUserID string
		expectedRole   string
	}{
		{
			name:           "AdminToken",
			token:          adminToken,
			expectedStatus: http.StatusOK,
			expectedUserID: "user123",
			expectedRole:   "admin",
		},
		{
			name:           "UserToken",
			token:          userToken,
			expectedStatus: http.StatusOK,
			expectedUserID: "user456",
			expectedRole:   "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware with context-capturing handler
			var capturedCtx context.Context
			middleware := HTTPAuthMiddleware(authMW, logger)
			handler := middleware(contextCapturingHandler(&capturedCtx))

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Verify context (using ContextKey)
			userID, ok := capturedCtx.Value(UserIDKey).(string)
			require.True(t, ok, "UserID should be in context")
			assert.Equal(t, tt.expectedUserID, userID)

			role, ok := capturedCtx.Value(RoleKey).(string)
			require.True(t, ok, "Role should be in context")
			assert.Equal(t, tt.expectedRole, role)
		})
	}
}

func TestOptionalHTTPAuthMiddleware_Integration_MixedAuth(t *testing.T) {
	authMW := newTestAuthMiddleware()
	logger := newTestLogger()
	validToken := generateValidToken(authMW)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectContext  bool
		expectedUserID string
	}{
		{
			name:           "NoAuth",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectContext:  false,
			expectedUserID: "",
		},
		{
			name:           "ValidAuth",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectContext:  true,
			expectedUserID: "user123",
		},
		{
			name:           "InvalidAuth",
			authHeader:     "Bearer invalid",
			expectedStatus: http.StatusOK,
			expectContext:  false,
			expectedUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware with context-capturing handler
			var capturedCtx context.Context
			middleware := OptionalHTTPAuthMiddleware(authMW, logger)
			handler := middleware(contextCapturingHandler(&capturedCtx))

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Verify context (using ContextKey)
			if tt.expectContext {
				userID, ok := capturedCtx.Value(UserIDKey).(string)
				require.True(t, ok, "UserID should be in context")
				assert.Equal(t, tt.expectedUserID, userID)
			} else {
				userID, ok := capturedCtx.Value(UserIDKey).(string)
				assert.False(t, ok || userID != "", "UserID should not be in context")
			}
		})
	}
}
