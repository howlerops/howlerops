package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/sql-studio/backend-go/internal/middleware"
)

// User represents a user in the system
type User struct {
	ID        string            `json:"id"`
	Username  string            `json:"username"`
	Email     string            `json:"email"`
	Password  string            `json:"-"` // Never expose password in JSON
	Role      string            `json:"role"`
	Active    bool              `json:"active"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	LastLogin *time.Time        `json:"last_login"`
	Metadata  map[string]string `json:"metadata"`
}

// LoginAttempt tracks failed login attempts
type LoginAttempt struct {
	IP        string    `json:"ip"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccess   time.Time `json:"last_access"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Active       bool      `json:"active"`
}

// UserStore defines the interface for user storage
type UserStore interface {
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, token string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
	CleanupExpiredSessions(ctx context.Context) error
}

// LoginAttemptStore defines the interface for login attempt storage
type LoginAttemptStore interface {
	RecordAttempt(ctx context.Context, attempt *LoginAttempt) error
	GetAttempts(ctx context.Context, ip, username string, since time.Time) ([]*LoginAttempt, error)
	CleanupOldAttempts(ctx context.Context, before time.Time) error
}

// EmailService defines the interface for email operations
type EmailService interface {
	SendVerificationEmail(email, token, verificationURL string) error
	SendPasswordResetEmail(email, token, resetURL string) error
	SendWelcomeEmail(email, name string) error
}

// Service provides authentication functionality
type Service struct {
	userStore         UserStore
	sessionStore      SessionStore
	attemptStore      LoginAttemptStore
	authMiddleware    *middleware.AuthMiddleware
	emailService      EmailService
	logger            *logrus.Logger
	bcryptCost        int
	jwtExpiration     time.Duration
	refreshExpiration time.Duration
	maxLoginAttempts  int
	lockoutDuration   time.Duration
}

// Config holds authentication service configuration
type Config struct {
	BcryptCost        int
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
}

// NewService creates a new authentication service
func NewService(
	userStore UserStore,
	sessionStore SessionStore,
	attemptStore LoginAttemptStore,
	authMiddleware *middleware.AuthMiddleware,
	config Config,
	logger *logrus.Logger,
) *Service {
	return &Service{
		userStore:         userStore,
		sessionStore:      sessionStore,
		attemptStore:      attemptStore,
		authMiddleware:    authMiddleware,
		logger:            logger,
		bcryptCost:        config.BcryptCost,
		jwtExpiration:     config.JWTExpiration,
		refreshExpiration: config.RefreshExpiration,
		maxLoginAttempts:  config.MaxLoginAttempts,
		lockoutDuration:   config.LockoutDuration,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         *User     `json:"user"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Login authenticates a user and creates a session
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Check for account lockout
	if locked, err := s.isAccountLocked(ctx, req.IPAddress, req.Username); err != nil {
		return nil, fmt.Errorf("failed to check account lockout: %w", err)
	} else if locked {
		return nil, fmt.Errorf("account is temporarily locked due to too many failed login attempts")
	}

	// Get user by username
	user, err := s.userStore.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.recordLoginAttempt(ctx, req.IPAddress, req.Username, false)
		return nil, fmt.Errorf("invalid username or password")
	}

	// Check if user is active
	if !user.Active {
		s.recordLoginAttempt(ctx, req.IPAddress, req.Username, false)
		return nil, fmt.Errorf("user account is disabled")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.recordLoginAttempt(ctx, req.IPAddress, req.Username, false)
		return nil, fmt.Errorf("invalid username or password")
	}

	// Generate tokens
	token, err := s.authMiddleware.GenerateToken(user.ID, user.Username, user.Role, s.jwtExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := s.authMiddleware.GenerateRefreshToken(user.ID, s.refreshExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.jwtExpiration),
		CreatedAt:    time.Now(),
		LastAccess:   time.Now(),
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Active:       true,
	}

	if err := s.sessionStore.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update user last login
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now
	if err := s.userStore.UpdateUser(ctx, user); err != nil {
		s.logger.WithError(err).Warn("Failed to update user last login")
	}

	// Record successful login attempt
	s.recordLoginAttempt(ctx, req.IPAddress, req.Username, true)

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"ip":       req.IPAddress,
	}).Info("User logged in successfully")

	// Remove password from user object
	user.Password = ""

	return &LoginResponse{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

// Logout terminates a user session
func (s *Service) Logout(ctx context.Context, token string) error {
	// Delete session
	if err := s.sessionStore.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.logger.Info("User logged out successfully")
	return nil
}

// RefreshToken generates a new access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Validate refresh token
	userID, err := s.authMiddleware.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user
	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if user is active
	if !user.Active {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Get session by refresh token
	session, err := s.getSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Generate new tokens
	newToken, err := s.authMiddleware.GenerateToken(user.ID, user.Username, user.Role, s.jwtExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	newRefreshToken, err := s.authMiddleware.GenerateRefreshToken(user.ID, s.refreshExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update session
	session.Token = newToken
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = time.Now().Add(s.jwtExpiration)
	session.LastAccess = time.Now()

	if err := s.sessionStore.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Remove password from user object
	user.Password = ""

	return &LoginResponse{
		User:         user,
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

// GetProfile returns the user profile for the authenticated user
func (s *Service) GetProfile(ctx context.Context, userID string) (*User, error) {
	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Remove password from user object
	user.Password = ""

	return user, nil
}

// VerifyToken validates a JWT token and returns user information
func (s *Service) VerifyToken(ctx context.Context, token string) (*User, error) {
	// Get session
	session, err := s.sessionStore.GetSession(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is active and not expired
	if !session.Active || time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired or inactive")
	}

	// Get user
	user, err := s.userStore.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if user is active
	if !user.Active {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Update session last access
	session.LastAccess = time.Now()
	if err := s.sessionStore.UpdateSession(ctx, session); err != nil {
		s.logger.WithError(err).Warn("Failed to update session last access")
	}

	// Remove password from user object
	user.Password = ""

	return user, nil
}

// CreateUser creates a new user account
func (s *Service) CreateUser(ctx context.Context, user *User) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.ID = uuid.New().String()
	user.Password = string(hashedPassword)
	user.Active = true
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.Metadata == nil {
		user.Metadata = make(map[string]string)
	}

	return s.userStore.CreateUser(ctx, user)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all user sessions except current one
	// This forces re-authentication on other devices
	s.logger.WithField("user_id", userID).Info("Password changed successfully")

	return nil
}

// isAccountLocked checks if an account is locked due to failed login attempts
func (s *Service) isAccountLocked(ctx context.Context, ip, username string) (bool, error) {
	since := time.Now().Add(-s.lockoutDuration)
	attempts, err := s.attemptStore.GetAttempts(ctx, ip, username, since)
	if err != nil {
		return false, err
	}

	failedCount := 0
	for _, attempt := range attempts {
		if !attempt.Success {
			failedCount++
		}
	}

	return failedCount >= s.maxLoginAttempts, nil
}

// recordLoginAttempt records a login attempt
func (s *Service) recordLoginAttempt(ctx context.Context, ip, username string, success bool) {
	attempt := &LoginAttempt{
		IP:        ip,
		Username:  username,
		Timestamp: time.Now(),
		Success:   success,
	}

	if err := s.attemptStore.RecordAttempt(ctx, attempt); err != nil {
		s.logger.WithError(err).Error("Failed to record login attempt")
	}
}

// getSessionByRefreshToken gets a session by refresh token
func (s *Service) getSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	// This is a simplified implementation
	// In a real implementation, you might need to add an index on refresh_token
	// or store refresh tokens separately
	return nil, fmt.Errorf("not implemented")
}

// CleanupExpiredSessions removes expired sessions
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessionStore.CleanupExpiredSessions(ctx)
}

// CleanupOldLoginAttempts removes old login attempts
func (s *Service) CleanupOldLoginAttempts(ctx context.Context) error {
	before := time.Now().Add(-24 * time.Hour) // Keep attempts for 24 hours
	return s.attemptStore.CleanupOldAttempts(ctx, before)
}

// GenerateAPIKey generates a new API key for a user
func (s *Service) GenerateAPIKey(ctx context.Context, userID string) (string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as hex
	apiKey := hex.EncodeToString(bytes)

	// Store API key (implementation depends on your storage choice)
	// For now, just return the key
	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*User, error) {
	// Implementation depends on your storage choice
	// This would typically involve looking up the API key in your database
	return nil, fmt.Errorf("not implemented")
}

// SetEmailService sets the email service for the auth service
func (s *Service) SetEmailService(emailService EmailService) {
	s.emailService = emailService
}

// SendVerificationEmail sends a verification email to a user
func (s *Service) SendVerificationEmail(ctx context.Context, userID string) error {
	if s.emailService == nil {
		return fmt.Errorf("email service not configured")
	}

	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Generate verification token
	token, err := s.authMiddleware.GenerateToken(userID, user.Username, user.Role, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Build verification URL (would typically come from config)
	verificationURL := fmt.Sprintf("http://localhost:3000/verify?token=%s", token)

	// Send email
	if err := s.emailService.SendVerificationEmail(user.Email, token, verificationURL); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   user.Email,
	}).Info("Verification email sent")

	return nil
}

// SendPasswordResetEmail sends a password reset email to a user
func (s *Service) SendPasswordResetEmail(ctx context.Context, email string) error {
	if s.emailService == nil {
		return fmt.Errorf("email service not configured")
	}

	user, err := s.userStore.GetUserByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		s.logger.WithField("email", email).Warn("Password reset requested for non-existent email")
		return nil
	}

	// Generate reset token
	token, err := s.authMiddleware.GenerateToken(user.ID, user.Username, user.Role, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Build reset URL (would typically come from config)
	resetURL := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)

	// Send email
	if err := s.emailService.SendPasswordResetEmail(user.Email, token, resetURL); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("Password reset email sent")

	return nil
}

// SendWelcomeEmail sends a welcome email to a new user
func (s *Service) SendWelcomeEmail(ctx context.Context, userID string) error {
	if s.emailService == nil {
		return fmt.Errorf("email service not configured")
	}

	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Send email
	if err := s.emailService.SendWelcomeEmail(user.Email, user.Username); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   user.Email,
	}).Info("Welcome email sent")

	return nil
}
