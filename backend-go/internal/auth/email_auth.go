package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// ExtendedService extends the base auth service with email functionality
type ExtendedService struct {
	*Service
	emailService EmailService
	tokenStore   TokenStore
	baseURL      string
}

// NewExtendedService creates a new extended authentication service with email support
func NewExtendedService(
	baseService *Service,
	emailService EmailService,
	tokenStore TokenStore,
	baseURL string,
) *ExtendedService {
	return &ExtendedService{
		Service:      baseService,
		emailService: emailService,
		tokenStore:   tokenStore,
		baseURL:      baseURL,
	}
}

// RegisterUser creates a new user and sends a verification email
func (s *ExtendedService) RegisterUser(ctx context.Context, user *User, sendWelcome bool) error {
	// Create user account
	if err := s.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Generate verification token
	token, err := s.GenerateVerificationToken(ctx, user.ID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to generate verification token")
		// Don't fail registration if email fails
	} else {
		// Send verification email
		verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)
		if err := s.emailService.SendVerificationEmail(user.Email, token, verificationURL); err != nil {
			s.logger.WithError(err).Warn("Failed to send verification email")
		}
	}

	// Send welcome email if requested
	if sendWelcome {
		if err := s.emailService.SendWelcomeEmail(user.Email, user.Username); err != nil {
			s.logger.WithError(err).Warn("Failed to send welcome email")
		}
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User registered successfully")

	return nil
}

// GenerateVerificationToken generates a new email verification token
func (s *ExtendedService) GenerateVerificationToken(ctx context.Context, userID string) (string, error) {
	// Generate secure token
	tokenValue, err := GenerateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create token record
	token := &Token{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     tokenValue,
		Type:      TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours expiration
		CreatedAt: time.Now(),
	}

	// Store token
	if err := s.tokenStore.CreateToken(ctx, token); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"expires_at": token.ExpiresAt,
	}).Info("Verification token generated")

	return tokenValue, nil
}

// VerifyEmail verifies a user's email address using a token
func (s *ExtendedService) VerifyEmail(ctx context.Context, tokenValue string) error {
	// Get token
	token, err := s.tokenStore.GetToken(ctx, tokenValue, TokenTypeEmailVerification)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token: %w", err)
	}

	// Get user
	user, err := s.userStore.GetUser(ctx, token.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update user metadata to mark email as verified
	if user.Metadata == nil {
		user.Metadata = make(map[string]string)
	}
	user.Metadata["email_verified"] = "true"
	user.Metadata["email_verified_at"] = time.Now().Format(time.RFC3339)
	user.UpdatedAt = time.Now()

	// Update user
	if err := s.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Mark token as used
	if err := s.tokenStore.MarkTokenUsed(ctx, tokenValue); err != nil {
		s.logger.WithError(err).Warn("Failed to mark token as used")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("Email verified successfully")

	return nil
}

// RequestEmailVerification sends a new verification email to a user
func (s *ExtendedService) RequestEmailVerification(ctx context.Context, userID string) error {
	// Get user
	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if already verified
	if user.Metadata != nil {
		if verified, exists := user.Metadata["email_verified"]; exists && verified == "true" {
			return fmt.Errorf("email already verified")
		}
	}

	// Generate verification token
	token, err := s.GenerateVerificationToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Send verification email
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)
	if err := s.emailService.SendVerificationEmail(user.Email, token, verificationURL); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   user.Email,
	}).Info("Verification email sent")

	return nil
}

// GenerateResetToken generates a new password reset token
func (s *ExtendedService) GenerateResetToken(ctx context.Context, userID string) (string, error) {
	// Generate secure token
	tokenValue, err := GenerateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create token record
	token := &Token{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     tokenValue,
		Type:      TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour expiration for security
		CreatedAt: time.Now(),
	}

	// Store token
	if err := s.tokenStore.CreateToken(ctx, token); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"expires_at": token.ExpiresAt,
	}).Info("Password reset token generated")

	return tokenValue, nil
}

// RequestPasswordReset initiates a password reset flow
func (s *ExtendedService) RequestPasswordReset(ctx context.Context, email string) error {
	// Get user by email
	user, err := s.userStore.GetUserByEmail(ctx, email)
	if err != nil {
		// For security, don't reveal if email exists
		s.logger.WithField("email", email).Info("Password reset requested for non-existent email")
		return nil
	}

	// Generate reset token
	token, err := s.GenerateResetToken(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Send password reset email
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	if err := s.emailService.SendPasswordResetEmail(user.Email, token, resetURL); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("Password reset email sent")

	return nil
}

// ResetPassword resets a user's password using a reset token
func (s *ExtendedService) ResetPassword(ctx context.Context, tokenValue, newPassword string) error {
	// Get token
	token, err := s.tokenStore.GetToken(ctx, tokenValue, TokenTypePasswordReset)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token: %w", err)
	}

	// Get user
	user, err := s.userStore.GetUser(ctx, token.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Validate new password
	if err := s.validatePassword(newPassword); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.tokenStore.MarkTokenUsed(ctx, tokenValue); err != nil {
		s.logger.WithError(err).Warn("Failed to mark token as used")
	}

	// Invalidate all existing sessions for security
	if err := s.sessionStore.DeleteUserSessions(ctx, user.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to delete user sessions")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
	}).Info("Password reset successfully")

	return nil
}

// validatePassword validates password strength
func (s *ExtendedService) validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Add more validation rules as needed
	hasUpper := false
	hasLower := false
	hasDigit := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain uppercase, lowercase, and numeric characters")
	}

	return nil
}

// IsEmailVerified checks if a user's email is verified
func (s *ExtendedService) IsEmailVerified(ctx context.Context, userID string) (bool, error) {
	user, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	if user.Metadata == nil {
		return false, nil
	}

	verified, exists := user.Metadata["email_verified"]
	return exists && verified == "true", nil
}

// CleanupExpiredTokens removes expired tokens
func (s *ExtendedService) CleanupExpiredTokens(ctx context.Context) error {
	return s.tokenStore.CleanupExpiredTokens(ctx)
}
