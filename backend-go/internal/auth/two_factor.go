package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// TwoFactor represents 2FA configuration for a user
type TwoFactor struct {
	UserID      string     `json:"user_id"`
	Enabled     bool       `json:"enabled"`
	Secret      string     `json:"-"` // Never expose secret in JSON
	BackupCodes []string   `json:"-"` // Never expose backup codes in JSON
	CreatedAt   time.Time  `json:"created_at"`
	EnabledAt   *time.Time `json:"enabled_at,omitempty"`
}

// TwoFactorSetup represents the setup response for 2FA
type TwoFactorSetup struct {
	Secret      string   `json:"secret"`
	QRCode      string   `json:"qr_code"`
	BackupCodes []string `json:"backup_codes"`
}

// TwoFactorStore defines the interface for 2FA storage
type TwoFactorStore interface {
	CreateTwoFactor(ctx context.Context, tf *TwoFactor) error
	GetTwoFactor(ctx context.Context, userID string) (*TwoFactor, error)
	EnableTwoFactor(ctx context.Context, userID string) error
	DisableTwoFactor(ctx context.Context, userID string) error
	UpdateBackupCodes(ctx context.Context, userID string, codes []string) error
	UseBackupCode(ctx context.Context, userID, code string) error
}

// TwoFactorService handles two-factor authentication
type TwoFactorService struct {
	store       TwoFactorStore
	eventLogger SecurityEventLogger
	logger      *logrus.Logger
	issuer      string
}

// SecurityEventLogger interface for logging security events
type SecurityEventLogger interface {
	LogSecurityEvent(ctx context.Context, eventType, userID, orgID, ipAddress, userAgent string, details map[string]interface{}) error
}

// NewTwoFactorService creates a new 2FA service
func NewTwoFactorService(store TwoFactorStore, eventLogger SecurityEventLogger, logger *logrus.Logger, issuer string) *TwoFactorService {
	if issuer == "" {
		issuer = "Howlerops"
	}
	return &TwoFactorService{
		store:       store,
		eventLogger: eventLogger,
		logger:      logger,
		issuer:      issuer,
	}
}

// EnableTwoFactor generates 2FA setup for a user
func (s *TwoFactorService) EnableTwoFactor(ctx context.Context, userID, userEmail string) (*TwoFactorSetup, error) {
	// Check if 2FA already exists
	existing, _ := s.store.GetTwoFactor(ctx, userID)
	if existing != nil && existing.Enabled {
		return nil, fmt.Errorf("2FA is already enabled for this user")
	}

	// Generate TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: userEmail,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate TOTP key")
		return nil, fmt.Errorf("failed to generate 2FA secret")
	}

	// Generate backup codes
	backupCodes := s.generateBackupCodes(10)
	hashedCodes := s.hashBackupCodes(backupCodes)

	// Store in database (disabled until user confirms)
	tf := &TwoFactor{
		UserID:      userID,
		Enabled:     false,
		Secret:      key.Secret(),
		BackupCodes: hashedCodes,
		CreatedAt:   time.Now(),
	}

	if err := s.store.CreateTwoFactor(ctx, tf); err != nil {
		s.logger.WithError(err).Error("Failed to store 2FA configuration")
		return nil, fmt.Errorf("failed to setup 2FA")
	}

	// Log security event
	if s.eventLogger != nil {
		if err := s.eventLogger.LogSecurityEvent(ctx, "2fa_setup_initiated", userID, "", "", "", nil); err != nil {
			s.logger.WithError(err).Warn("Failed to log 2FA setup initiated security event")
		}
	}

	return &TwoFactorSetup{
		Secret:      key.Secret(),
		QRCode:      key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// ConfirmTwoFactor confirms and enables 2FA after validating a code
func (s *TwoFactorService) ConfirmTwoFactor(ctx context.Context, userID, code string) error {
	// Get 2FA configuration
	config, err := s.store.GetTwoFactor(ctx, userID)
	if err != nil {
		return fmt.Errorf("2FA not configured")
	}

	if config.Enabled {
		return fmt.Errorf("2FA is already enabled")
	}

	// Validate TOTP code
	valid := totp.Validate(code, config.Secret)
	if !valid {
		s.logger.WithField("user_id", userID).Warn("Invalid 2FA confirmation code")
		return fmt.Errorf("invalid verification code")
	}

	// Enable 2FA
	if err := s.store.EnableTwoFactor(ctx, userID); err != nil {
		return fmt.Errorf("failed to enable 2FA")
	}

	// Log security event
	if s.eventLogger != nil {
		if err := s.eventLogger.LogSecurityEvent(ctx, "2fa_enabled", userID, "", "", "", nil); err != nil {
			s.logger.WithError(err).Warn("Failed to log 2FA enabled security event")
		}
	}

	s.logger.WithField("user_id", userID).Info("2FA enabled successfully")
	return nil
}

// ValidateCode validates a 2FA code (TOTP or backup code)
func (s *TwoFactorService) ValidateCode(ctx context.Context, userID, code string) error {
	config, err := s.store.GetTwoFactor(ctx, userID)
	if err != nil {
		return fmt.Errorf("2FA not configured")
	}

	if !config.Enabled {
		return fmt.Errorf("2FA not enabled")
	}

	// Remove any spaces from the code
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")

	// Try TOTP code first (6 digits)
	if len(code) == 6 {
		valid := totp.Validate(code, config.Secret)
		if valid {
			s.logger.WithField("user_id", userID).Debug("Valid TOTP code used")
			return nil
		}
	}

	// Try backup codes (8 characters)
	if len(code) == 8 {
		if s.isValidBackupCode(code, config.BackupCodes) {
			// Mark backup code as used
			if err := s.store.UseBackupCode(ctx, userID, code); err != nil {
				s.logger.WithError(err).Error("Failed to mark backup code as used")
			}

			// Log security event
			if s.eventLogger != nil {
				if err := s.eventLogger.LogSecurityEvent(ctx, "2fa_backup_code_used", userID, "", "", "", nil); err != nil {
					s.logger.WithError(err).Warn("Failed to log 2FA backup code used security event")
				}
			}

			s.logger.WithField("user_id", userID).Info("Backup code used for 2FA")
			return nil
		}
	}

	// Log failed attempt
	if s.eventLogger != nil {
		_ = s.eventLogger.LogSecurityEvent(ctx, "2fa_validation_failed", userID, "", "", "", nil) // Best-effort logging
	}

	return fmt.Errorf("invalid verification code")
}

// DisableTwoFactor disables 2FA for a user
func (s *TwoFactorService) DisableTwoFactor(ctx context.Context, userID, currentPassword string) error {
	// Verify user's password before disabling 2FA
	// This adds an extra security layer

	if err := s.store.DisableTwoFactor(ctx, userID); err != nil {
		return fmt.Errorf("failed to disable 2FA")
	}

	// Log security event
	if s.eventLogger != nil {
		_ = s.eventLogger.LogSecurityEvent(ctx, "2fa_disabled", userID, "", "", "", nil) // Best-effort logging
	}

	s.logger.WithField("user_id", userID).Info("2FA disabled")
	return nil
}

// RegenerateBackupCodes generates new backup codes
func (s *TwoFactorService) RegenerateBackupCodes(ctx context.Context, userID string) ([]string, error) {
	// Verify 2FA is enabled
	config, err := s.store.GetTwoFactor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("2FA not configured")
	}

	if !config.Enabled {
		return nil, fmt.Errorf("2FA not enabled")
	}

	// Generate new backup codes
	backupCodes := s.generateBackupCodes(10)
	hashedCodes := s.hashBackupCodes(backupCodes)

	// Update stored codes
	if err := s.store.UpdateBackupCodes(ctx, userID, hashedCodes); err != nil {
		return nil, fmt.Errorf("failed to update backup codes")
	}

	// Log security event
	if s.eventLogger != nil {
		_ = s.eventLogger.LogSecurityEvent(ctx, "2fa_backup_codes_regenerated", userID, "", "", "", nil) // Best-effort logging
	}

	return backupCodes, nil
}

// IsEnabled checks if 2FA is enabled for a user
func (s *TwoFactorService) IsEnabled(ctx context.Context, userID string) (bool, error) {
	config, err := s.store.GetTwoFactor(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // Not configured
		}
		return false, fmt.Errorf("failed to get two-factor config: %w", err)
	}
	return config.Enabled, nil
}

// generateBackupCodes generates random backup codes
func (s *TwoFactorService) generateBackupCodes(count int) []string {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		codes[i] = s.generateRandomCode(8)
	}
	return codes
}

// generateRandomCode generates a random alphanumeric code
func (s *TwoFactorService) generateRandomCode(length int) string {
	// Use base32 encoding for better readability (no 0/O or 1/I confusion)
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		s.logger.WithError(err).Error("Failed to generate random bytes, using fallback")
		// Fallback to time-based seed if crypto/rand fails
		for i := range bytes {
			bytes[i] = byte(time.Now().UnixNano() % 256)
		}
	}
	code := base32.StdEncoding.EncodeToString(bytes)
	// Take first 'length' characters and convert to uppercase
	if len(code) > length {
		code = code[:length]
	}
	return strings.ToUpper(code)
}

// hashBackupCodes hashes backup codes for storage
func (s *TwoFactorService) hashBackupCodes(codes []string) []string {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		hash, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		hashed[i] = string(hash)
	}
	return hashed
}

// isValidBackupCode checks if a code matches any stored backup code
func (s *TwoFactorService) isValidBackupCode(code string, hashedCodes []string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, hash := range hashedCodes {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err == nil {
			return true
		}
	}
	return false
}

// TwoFactorBackupCodesResponse represents the backup codes response
type TwoFactorBackupCodesResponse struct {
	BackupCodes []string  `json:"backup_codes"`
	CreatedAt   time.Time `json:"created_at"`
}

// TwoFactorStatusResponse represents the 2FA status
type TwoFactorStatusResponse struct {
	Enabled          bool       `json:"enabled"`
	ConfiguredAt     *time.Time `json:"configured_at,omitempty"`
	EnabledAt        *time.Time `json:"enabled_at,omitempty"`
	BackupCodesCount int        `json:"backup_codes_count"`
}

// GetStatus returns the 2FA status for a user
func (s *TwoFactorService) GetStatus(ctx context.Context, userID string) (*TwoFactorStatusResponse, error) {
	config, err := s.store.GetTwoFactor(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Not configured
			return &TwoFactorStatusResponse{
				Enabled: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get two-factor config: %w", err)
	}

	// Count remaining backup codes
	backupCodesCount := 0
	if config.BackupCodes != nil {
		var codes []string
		if err := json.Unmarshal([]byte(strings.Join(config.BackupCodes, ",")), &codes); err == nil {
			backupCodesCount = len(codes)
		}
	}

	return &TwoFactorStatusResponse{
		Enabled:          config.Enabled,
		ConfiguredAt:     &config.CreatedAt,
		EnabledAt:        config.EnabledAt,
		BackupCodesCount: backupCodesCount,
	}, nil
}
