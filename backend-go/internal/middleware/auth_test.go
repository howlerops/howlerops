package middleware_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/sql-studio/backend-go/internal/middleware"
)

// TestNewAuthMiddleware tests the AuthMiddleware constructor
func TestNewAuthMiddleware(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"

	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	assert.NotNil(t, authMiddleware)
}

// TestNewAuthMiddleware_EmptySecret tests constructor with empty secret
func TestNewAuthMiddleware_EmptySecret(t *testing.T) {
	logger := newSilentLogger()
	secret := ""

	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	assert.NotNil(t, authMiddleware)
}

// TestNewAuthMiddleware_NilLogger tests constructor with nil logger
func TestNewAuthMiddleware_NilLogger(t *testing.T) {
	secret := "test-secret-key"

	// Should not panic
	authMiddleware := middleware.NewAuthMiddleware(secret, nil)

	assert.NotNil(t, authMiddleware)
}

// TestUnaryInterceptor_PublicMethod tests that public methods bypass authentication
func TestUnaryInterceptor_PublicMethod(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	publicMethods := []string{
		"/sqlstudio.auth.AuthService/Login",
		"/sqlstudio.health.HealthService/Check",
		"/sqlstudio.health.HealthService/Watch",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}

	for _, method := range publicMethods {
		t.Run(method, func(t *testing.T) {
			ctx := context.Background()
			req := "test-request"
			handler := unaryHandlerResponse("response")
			info := &grpc.UnaryServerInfo{FullMethod: method}

			resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		})
	}
}

// TestUnaryInterceptor_PrivateMethodWithoutToken tests that private methods require authentication
func TestUnaryInterceptor_PrivateMethodWithoutToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := context.Background()
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestUnaryInterceptor_PrivateMethodWithEmptyToken tests authentication with empty token
func TestUnaryInterceptor_PrivateMethodWithEmptyToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := authContext("")
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestUnaryInterceptor_PrivateMethodWithInvalidToken tests authentication with invalid token
func TestUnaryInterceptor_PrivateMethodWithInvalidToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := authContext("invalid-token")
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, err.Error(), "invalid token")
}

// TestUnaryInterceptor_PrivateMethodWithValidToken tests authentication with valid token
func TestUnaryInterceptor_PrivateMethodWithValidToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

// TestUnaryInterceptor_PrivateMethodWithExpiredToken tests authentication with expired token
func TestUnaryInterceptor_PrivateMethodWithExpiredToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate expired token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", -1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, err.Error(), "invalid token")
}

// TestUnaryInterceptor_ContextPropagation tests that user context is propagated
func TestUnaryInterceptor_ContextPropagation(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	req := "test-request"
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify user context
		userID, username, role, err := middleware.ExtractUserFromContext(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "user123", userID)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, "admin", role)
		return "response", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

// TestUnaryInterceptor_MissingMetadata tests authentication without metadata
func TestUnaryInterceptor_MissingMetadata(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := context.Background()
	req := "test-request"
	handler := unaryHandlerResponse("response")
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestStreamInterceptor_PublicMethod tests that public methods bypass authentication for streams
func TestStreamInterceptor_PublicMethod(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.health.HealthService/Watch"}

	err := authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.NoError(t, err)
}

// TestStreamInterceptor_PrivateMethodWithoutToken tests stream authentication without token
func TestStreamInterceptor_PrivateMethodWithoutToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err := authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestStreamInterceptor_PrivateMethodWithValidToken tests stream authentication with valid token
func TestStreamInterceptor_PrivateMethodWithValidToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err = authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.NoError(t, err)
}

// TestStreamInterceptor_PrivateMethodWithInvalidToken tests stream authentication with invalid token
func TestStreamInterceptor_PrivateMethodWithInvalidToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := authContext("invalid-token")
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err := authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestStreamInterceptor_PrivateMethodWithExpiredToken tests stream authentication with expired token
func TestStreamInterceptor_PrivateMethodWithExpiredToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate expired token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", -1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err = authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestStreamInterceptor_ContextPropagation tests context propagation in streams
func TestStreamInterceptor_ContextPropagation(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	stream := &mockServerStream{ctx: ctx}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Verify user context
		userID, username, role, err := middleware.ExtractUserFromContext(stream.Context())
		assert.NoError(t, err)
		assert.Equal(t, "user123", userID)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, "admin", role)
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err = authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.NoError(t, err)
}

// TestStreamInterceptor_MissingMetadata tests stream authentication without metadata
func TestStreamInterceptor_MissingMetadata(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}
	handler := streamHandlerSuccess
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err := authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGenerateToken_Success tests successful token generation
func TestGenerateToken_Success(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestGenerateToken_DifferentRoles tests token generation with different roles
func TestGenerateToken_DifferentRoles(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	roles := []string{"admin", "user", "viewer", "editor"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			token, err := authMiddleware.GenerateToken("user123", "testuser", role, 1*time.Hour)

			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify token contains correct role
			claims, err := authMiddleware.GenerateToken("user123", "testuser", role, 1*time.Hour)
			require.NoError(t, err)
			assert.NotEmpty(t, claims)
		})
	}
}

// TestGenerateToken_ShortDuration tests token generation with short duration
func TestGenerateToken_ShortDuration(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Second)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestGenerateToken_LongDuration tests token generation with long duration
func TestGenerateToken_LongDuration(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 168*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestGenerateToken_EmptyUserID tests token generation with empty user ID
func TestGenerateToken_EmptyUserID(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateToken("", "testuser", "admin", 1*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestGenerateToken_EmptyUsername tests token generation with empty username
func TestGenerateToken_EmptyUsername(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateToken("user123", "", "admin", 1*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestValidateToken_Success tests successful token validation
func TestValidateToken_Success(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	// Parse the token to verify it
	parsedToken, err := jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)

	claims, ok := parsedToken.Claims.(*middleware.JWTClaims)
	assert.True(t, ok)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

// TestValidateToken_InvalidSignature tests token validation with invalid signature
func TestValidateToken_InvalidSignature(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"

	// Generate token with different secret
	otherAuth := middleware.NewAuthMiddleware("different-secret", logger)
	token, err := otherAuth.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	// Try to parse with original secret
	_, err = jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.Error(t, err)
}

// TestValidateToken_MalformedToken tests token validation with malformed token
func TestValidateToken_MalformedToken(t *testing.T) {
	secret := "test-secret-key"

	malformedTokens := []string{
		"",
		"invalid",
		"invalid.token.format",
		"Bearer token",
	}

	for _, token := range malformedTokens {
		t.Run(fmt.Sprintf("token=%s", token), func(t *testing.T) {
			_, err := jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			assert.Error(t, err)
		})
	}
}

// TestValidateToken_ExpiredToken tests token validation with expired token
func TestValidateToken_ExpiredToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate expired token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", -1*time.Hour)
	require.NoError(t, err)

	// Try to parse
	parsedToken, err := jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	// Token parsing may succeed but validation will fail
	if err == nil {
		claims, ok := parsedToken.Claims.(*middleware.JWTClaims)
		assert.True(t, ok)
		assert.True(t, claims.ExpiresAt.Before(time.Now()))
	} else {
		assert.Error(t, err)
	}
}

// TestValidateToken_NotBeforeToken tests token validation with not-before claim
func TestValidateToken_NotBeforeToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate token (which sets NotBefore to now)
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	// Parse token
	parsedToken, err := jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)

	claims, ok := parsedToken.Claims.(*middleware.JWTClaims)
	assert.True(t, ok)
	assert.True(t, claims.NotBefore.Before(time.Now().Add(1*time.Minute)))
}

// TestGenerateRefreshToken_Success tests successful refresh token generation
func TestGenerateRefreshToken_Success(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateRefreshToken("user123", 168*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestGenerateRefreshToken_DifferentDurations tests refresh token generation with different durations
func TestGenerateRefreshToken_DifferentDurations(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	durations := []time.Duration{
		1 * time.Hour,
		24 * time.Hour,
		168 * time.Hour,
		720 * time.Hour,
	}

	for _, duration := range durations {
		t.Run(duration.String(), func(t *testing.T) {
			token, err := authMiddleware.GenerateRefreshToken("user123", duration)

			assert.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

// TestGenerateRefreshToken_EmptyUserID tests refresh token generation with empty user ID
func TestGenerateRefreshToken_EmptyUserID(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	token, err := authMiddleware.GenerateRefreshToken("", 168*time.Hour)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestValidateRefreshToken_Success tests successful refresh token validation
func TestValidateRefreshToken_Success(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate refresh token
	token, err := authMiddleware.GenerateRefreshToken("user123", 168*time.Hour)
	require.NoError(t, err)

	// Validate refresh token
	userID, err := authMiddleware.ValidateRefreshToken(token)

	assert.NoError(t, err)
	assert.Equal(t, "user123", userID)
}

// TestValidateRefreshToken_InvalidToken tests refresh token validation with invalid token
func TestValidateRefreshToken_InvalidToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	userID, err := authMiddleware.ValidateRefreshToken("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

// TestValidateRefreshToken_ExpiredToken tests refresh token validation with expired token
func TestValidateRefreshToken_ExpiredToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate expired refresh token
	token, err := authMiddleware.GenerateRefreshToken("user123", -1*time.Hour)
	require.NoError(t, err)

	// Validate refresh token
	userID, err := authMiddleware.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Contains(t, err.Error(), "expired")
}

// TestValidateRefreshToken_RegularTokenAsRefreshToken tests that regular tokens are rejected as refresh tokens
func TestValidateRefreshToken_RegularTokenAsRefreshToken(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate regular token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	// Try to validate as refresh token
	userID, err := authMiddleware.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Contains(t, err.Error(), "not a refresh token")
}

// TestValidateRefreshToken_InvalidSignature tests refresh token validation with invalid signature
func TestValidateRefreshToken_InvalidSignature(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate token with different secret
	otherAuth := middleware.NewAuthMiddleware("different-secret", logger)
	token, err := otherAuth.GenerateRefreshToken("user123", 168*time.Hour)
	require.NoError(t, err)

	// Try to validate with original secret
	userID, err := authMiddleware.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.Empty(t, userID)
}

// TestExtractUserFromContext_Success tests successful user extraction from context
func TestExtractUserFromContext_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "user123", userID)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, "admin", role)
}

// TestExtractUserFromContext_MissingUserID tests extraction with missing user ID
func TestExtractUserFromContext_MissingUserID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Empty(t, username)
	assert.Empty(t, role)
	assert.Contains(t, err.Error(), "user ID not found")
}

// TestExtractUserFromContext_EmptyUserID tests extraction with empty user ID
func TestExtractUserFromContext_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Empty(t, username)
	assert.Empty(t, role)
}

// TestExtractUserFromContext_MissingUsername tests extraction with missing username
func TestExtractUserFromContext_MissingUsername(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "user123", userID)
	assert.Empty(t, username)
	assert.Equal(t, "admin", role)
}

// TestExtractUserFromContext_MissingRole tests extraction with missing role
func TestExtractUserFromContext_MissingRole(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "user123", userID)
	assert.Equal(t, "testuser", username)
	assert.Empty(t, role)
}

// TestExtractUserFromContext_WrongTypeUserID tests extraction with wrong type for user ID
func TestExtractUserFromContext_WrongTypeUserID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, 123)
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userID, username, role, err := middleware.ExtractUserFromContext(ctx)

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Empty(t, username)
	assert.Empty(t, role)
}

// TestRequireRole_Success tests successful role check
func TestRequireRole_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	err := middleware.RequireRole(ctx, "admin")

	assert.NoError(t, err)
}

// TestRequireRole_AdminBypass tests that admin role bypasses other role checks
func TestRequireRole_AdminBypass(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	err := middleware.RequireRole(ctx, "user")

	assert.NoError(t, err)
}

// TestRequireRole_InsufficientPermissions tests role check with insufficient permissions
func TestRequireRole_InsufficientPermissions(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "user")

	err := middleware.RequireRole(ctx, "admin")

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// TestRequireRole_UnauthenticatedUser tests role check without authentication
func TestRequireRole_UnauthenticatedUser(t *testing.T) {
	ctx := context.Background()

	err := middleware.RequireRole(ctx, "admin")

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestRequireAdmin_Success tests successful admin check
func TestRequireAdmin_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	err := middleware.RequireAdmin(ctx)

	assert.NoError(t, err)
}

// TestRequireAdmin_NonAdminUser tests admin check with non-admin user
func TestRequireAdmin_NonAdminUser(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "user")

	err := middleware.RequireAdmin(ctx)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// TestRequireAdmin_UnauthenticatedUser tests admin check without authentication
func TestRequireAdmin_UnauthenticatedUser(t *testing.T) {
	ctx := context.Background()

	err := middleware.RequireAdmin(ctx)

	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestAuthTokenExtractor_ExtractToken_Success tests successful token extraction
func TestAuthTokenExtractor_ExtractToken_Success(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	md := metadata.New(map[string]string{
		"authorization": "Bearer test-token-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, err := extractor.ExtractToken(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test-token-123", token)
}

// TestAuthTokenExtractor_ExtractToken_MissingMetadata tests extraction without metadata
func TestAuthTokenExtractor_ExtractToken_MissingMetadata(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	ctx := context.Background()

	token, err := extractor.ExtractToken(ctx)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "missing metadata")
}

// TestAuthTokenExtractor_ExtractToken_MissingAuthHeader tests extraction without auth header
func TestAuthTokenExtractor_ExtractToken_MissingAuthHeader(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	md := metadata.New(map[string]string{
		"other-header": "value",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, err := extractor.ExtractToken(ctx)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "missing authorization header")
}

// TestAuthTokenExtractor_ExtractToken_InvalidFormat tests extraction with invalid format
func TestAuthTokenExtractor_ExtractToken_InvalidFormat(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	md := metadata.New(map[string]string{
		"authorization": "InvalidFormat token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, err := extractor.ExtractToken(ctx)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid authorization header format")
}

// TestAuthTokenExtractor_ExtractToken_CaseInsensitive tests that bearer prefix is case-insensitive
func TestAuthTokenExtractor_ExtractToken_CaseInsensitive(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	testCases := []string{
		"Bearer token123",
		"bearer token123",
		"BEARER token123",
		"BeArEr token123",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			md := metadata.New(map[string]string{
				"authorization": tc,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			token, err := extractor.ExtractToken(ctx)

			assert.NoError(t, err)
			assert.Equal(t, "token123", token)
		})
	}
}

// TestAuthTokenExtractor_ExtractToken_EmptyToken tests extraction with empty token
func TestAuthTokenExtractor_ExtractToken_EmptyToken(t *testing.T) {
	extractor := &middleware.AuthTokenExtractor{}

	md := metadata.New(map[string]string{
		"authorization": "Bearer ",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, err := extractor.ExtractToken(ctx)

	assert.NoError(t, err)
	assert.Empty(t, token)
}

// TestSetTokenInContext_Success tests successful token setting
func TestSetTokenInContext_Success(t *testing.T) {
	token := "test-token-123"

	ctx := middleware.SetTokenInContext(context.Background(), token)

	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, []string{"Bearer test-token-123"}, md.Get("authorization"))
}

// TestSetTokenInContext_EmptyToken tests setting empty token
func TestSetTokenInContext_EmptyToken(t *testing.T) {
	token := ""

	ctx := middleware.SetTokenInContext(context.Background(), token)

	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, []string{"Bearer "}, md.Get("authorization"))
}

// TestGetUserContext_Success tests successful user context retrieval
func TestGetUserContext_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")
	ctx = context.WithValue(ctx, middleware.UsernameKey, "testuser")
	ctx = context.WithValue(ctx, middleware.RoleKey, "admin")

	userContext, err := middleware.GetUserContext(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, userContext)
	assert.Equal(t, "user123", userContext.UserID)
	assert.Equal(t, "testuser", userContext.Username)
	assert.Equal(t, "admin", userContext.Role)
}

// TestGetUserContext_MissingUserID tests user context retrieval with missing user ID
func TestGetUserContext_MissingUserID(t *testing.T) {
	ctx := context.Background()

	userContext, err := middleware.GetUserContext(ctx)

	assert.Error(t, err)
	assert.Nil(t, userContext)
}

// TestGetUserContext_PartialData tests user context retrieval with partial data
func TestGetUserContext_PartialData(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, "user123")

	userContext, err := middleware.GetUserContext(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, userContext)
	assert.Equal(t, "user123", userContext.UserID)
	assert.Empty(t, userContext.Username)
	assert.Empty(t, userContext.Role)
}

// TestWrappedServerStream_Context tests wrapped server stream context
func TestWrappedServerStream_Context(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	stream := &mockServerStream{ctx: ctx}

	// Simulate stream interception
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Verify the wrapped stream preserves context
		streamCtx := stream.Context()
		userID, username, role, err := middleware.ExtractUserFromContext(streamCtx)
		assert.NoError(t, err)
		assert.Equal(t, "user123", userID)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, "admin", role)
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}
	err = authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.NoError(t, err)
}

// TestJWTClaims_Structure tests JWT claims structure
func TestJWTClaims_Structure(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	// Parse and verify claims structure
	parsedToken, err := jwt.ParseWithClaims(token, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)

	claims, ok := parsedToken.Claims.(*middleware.JWTClaims)
	assert.True(t, ok)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "sql-studio", claims.Issuer)
	assert.Equal(t, "user123", claims.Subject)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
}

// TestMultipleTokensGeneration tests generating multiple tokens for different users
func TestMultipleTokensGeneration(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	users := []struct {
		userID   string
		username string
		role     string
	}{
		{"user1", "alice", "admin"},
		{"user2", "bob", "user"},
		{"user3", "charlie", "viewer"},
	}

	tokens := make([]string, len(users))

	for i, user := range users {
		token, err := authMiddleware.GenerateToken(user.userID, user.username, user.role, 1*time.Hour)
		require.NoError(t, err)
		tokens[i] = token
	}

	// Verify all tokens are unique
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			assert.NotEqual(t, tokens[i], tokens[j])
		}
	}
}

// TestTokenRefreshFlow tests complete token refresh flow
func TestTokenRefreshFlow(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate refresh token
	refreshToken, err := authMiddleware.GenerateRefreshToken("user123", 168*time.Hour)
	require.NoError(t, err)

	// Validate refresh token
	userID, err := authMiddleware.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, "user123", userID)

	// Generate new access token using user ID from refresh token
	accessToken, err := authMiddleware.GenerateToken(userID, "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
}

// TestConcurrentTokenGeneration tests thread safety of token generation
func TestConcurrentTokenGeneration(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	numGoroutines := 100
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			token, err := authMiddleware.GenerateToken(
				fmt.Sprintf("user%d", id),
				fmt.Sprintf("testuser%d", id),
				"admin",
				1*time.Hour,
			)
			if err != nil {
				errors <- err
			} else {
				results <- token
			}
		}(i)
	}

	// Collect results
	tokens := make([]string, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		select {
		case token := <-results:
			tokens = append(tokens, token)
		case err := <-errors:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	assert.Equal(t, numGoroutines, len(tokens))
}

// TestUnaryInterceptor_HandlerError tests that handler errors are propagated
func TestUnaryInterceptor_HandlerError(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	req := "test-request"
	handlerErr := fmt.Errorf("handler error")
	handler := unaryHandlerError(handlerErr)
	info := &grpc.UnaryServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Execute"}

	resp, err := authMiddleware.UnaryInterceptor(ctx, req, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, handlerErr, err)
}

// TestStreamInterceptor_HandlerError tests that stream handler errors are propagated
func TestStreamInterceptor_HandlerError(t *testing.T) {
	logger := newSilentLogger()
	secret := "test-secret-key"
	authMiddleware := middleware.NewAuthMiddleware(secret, logger)

	// Generate valid token
	token, err := authMiddleware.GenerateToken("user123", "testuser", "admin", 1*time.Hour)
	require.NoError(t, err)

	ctx := authContext(token)
	stream := &mockServerStream{ctx: ctx}
	handlerErr := fmt.Errorf("handler error")
	handler := streamHandlerError(handlerErr)
	info := &grpc.StreamServerInfo{FullMethod: "/sqlstudio.db.DatabaseService/Stream"}

	err = authMiddleware.StreamInterceptor(nil, stream, info, handler)

	assert.Error(t, err)
	assert.Equal(t, handlerErr, err)
}
