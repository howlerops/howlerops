package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtSecret []byte
	logger    *logrus.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
		logger:    logger,
	}
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UnaryInterceptor provides authentication for unary gRPC calls
func (a *AuthMiddleware) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Skip authentication for certain methods
	if a.isPublicMethod(info.FullMethod) {
		return handler(ctx, req)
	}

	// Authenticate the request
	newCtx, err := a.authenticate(ctx)
	if err != nil {
		a.logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
			"error":  err,
		}).Warn("Authentication failed")
		return nil, err
	}

	return handler(newCtx, req)
}

// StreamInterceptor provides authentication for streaming gRPC calls
func (a *AuthMiddleware) StreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Skip authentication for certain methods
	if a.isPublicMethod(info.FullMethod) {
		return handler(srv, stream)
	}

	// Authenticate the request
	newCtx, err := a.authenticate(stream.Context())
	if err != nil {
		a.logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
			"error":  err,
		}).Warn("Authentication failed")
		return err
	}

	// Wrap the stream with the new context
	wrappedStream := &wrappedServerStream{
		ServerStream: stream,
		ctx:          newCtx,
	}

	return handler(srv, wrappedStream)
}

// authenticate validates the JWT token and extracts user information
func (a *AuthMiddleware) authenticate(ctx context.Context) (context.Context, error) {
	// Extract token from metadata
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
	}

	if token == "" {
		return nil, status.Errorf(codes.Unauthenticated, "empty authorization token")
	}

	// Parse and validate the token
	claims, err := a.validateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Add user information to context
	newCtx := context.WithValue(ctx, "user_id", claims.UserID)
	newCtx = context.WithValue(newCtx, "username", claims.Username)
	newCtx = context.WithValue(newCtx, "role", claims.Role)

	return newCtx, nil
}

// validateToken parses and validates a JWT token
func (a *AuthMiddleware) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token is expired")
	}

	return claims, nil
}

// isPublicMethod checks if a method is public (doesn't require authentication)
func (a *AuthMiddleware) isPublicMethod(method string) bool {
	publicMethods := map[string]bool{
		"/sqlstudio.auth.AuthService/Login":                              true,
		"/sqlstudio.health.HealthService/Check":                          true,
		"/sqlstudio.health.HealthService/Watch":                          true,
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	}

	return publicMethods[method]
}

// GenerateToken generates a JWT token for a user
func (a *AuthMiddleware) GenerateToken(userID, username, role string, duration time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "sql-studio",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// GenerateRefreshToken generates a refresh token
func (a *AuthMiddleware) GenerateRefreshToken(userID string, duration time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "sql-studio",
		Subject:   userID,
		ID:        "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateRefreshToken validates a refresh token
func (a *AuthMiddleware) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return "", fmt.Errorf("token is expired")
	}

	// Check if it's a refresh token
	if claims.ID != "refresh" {
		return "", fmt.Errorf("not a refresh token")
	}

	return claims.Subject, nil
}

// ExtractUserFromContext extracts user information from context
func ExtractUserFromContext(ctx context.Context) (string, string, string, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return "", "", "", fmt.Errorf("user ID not found in context")
	}

	username, ok := ctx.Value("username").(string)
	if !ok {
		username = ""
	}

	role, ok := ctx.Value("role").(string)
	if !ok {
		role = ""
	}

	return userID, username, role, nil
}

// RequireRole checks if the user has the required role
func RequireRole(ctx context.Context, requiredRole string) error {
	_, _, role, err := ExtractUserFromContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	if role != requiredRole && role != "admin" {
		return status.Errorf(codes.PermissionDenied, "insufficient permissions")
	}

	return nil
}

// RequireAdmin checks if the user is an admin
func RequireAdmin(ctx context.Context) error {
	return RequireRole(ctx, "admin")
}

// wrappedServerStream wraps grpc.ServerStream with a new context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// AuthTokenExtractor extracts auth token from gRPC metadata
type AuthTokenExtractor struct{}

// ExtractToken extracts the bearer token from gRPC metadata
func (a *AuthTokenExtractor) ExtractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	authorization := md.Get("authorization")
	if len(authorization) == 0 {
		return "", fmt.Errorf("missing authorization header")
	}

	token := authorization[0]
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return token[7:], nil // Remove "Bearer " prefix
}

// SetTokenInContext sets the auth token in outgoing context
func SetTokenInContext(ctx context.Context, token string) context.Context {
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	return metadata.NewOutgoingContext(ctx, md)
}

// UserContext holds user information
type UserContext struct {
	UserID   string
	Username string
	Role     string
}

// GetUserContext extracts user context from gRPC context
func GetUserContext(ctx context.Context) (*UserContext, error) {
	userID, username, role, err := ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &UserContext{
		UserID:   userID,
		Username: username,
		Role:     role,
	}, nil
}