package services

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the name used to identify this application in the keychain
	ServiceName = "HowlerOps Howlerops"
)

// Sentinel errors for credential operations
var (
	// ErrNotFound is returned when a credential is not found in the keychain
	ErrNotFound = errors.New("credential not found")

	// ErrPermissionDenied is returned when access to the keychain is denied
	ErrPermissionDenied = errors.New("permission denied to access keychain")

	// ErrUnavailable is returned when the keychain service is not available
	ErrUnavailable = errors.New("keychain service unavailable")

	// ErrInvalidInput is returned when invalid parameters are provided
	ErrInvalidInput = errors.New("invalid input parameters")
)

// CredentialService manages secure storage of database passwords using OS keychain.
// It supports:
//   - macOS: Keychain Access (automatic via go-keyring)
//   - Windows: Credential Manager (automatic via go-keyring)
//   - Linux: Secret Service API (requires libsecret)
//
// The service is thread-safe and handles all keychain operations with
// proper error handling and logging (without logging sensitive data).
type CredentialService struct {
	ctx    context.Context
	logger *logrus.Logger
	mu     sync.RWMutex // protects concurrent access
}

// NewCredentialService creates a new credential service instance.
// The logger is used for operational logging (never logs passwords or sensitive data).
func NewCredentialService(logger *logrus.Logger) *CredentialService {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.WarnLevel)
	}

	return &CredentialService{
		logger: logger,
	}
}

// SetContext sets the context for the credential service
func (s *CredentialService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// StorePassword securely stores a password for the given connection ID in the OS keychain.
//
// Parameters:
//   - connectionID: unique identifier for the database connection (used as keychain account)
//   - password: the password to store securely
//
// Returns:
//   - nil on success
//   - ErrInvalidInput if connectionID or password is empty
//   - ErrPermissionDenied if access to keychain is denied
//   - ErrUnavailable if keychain service is not available
//
// Platform behavior:
//   - macOS: Stores in Keychain Access under service name
//   - Windows: Stores in Windows Credential Manager
//   - Linux: Stores via Secret Service API (requires libsecret-1-0)
func (s *CredentialService) StorePassword(connectionID, password string) error {
	if connectionID == "" {
		return fmt.Errorf("%w: connectionID cannot be empty", ErrInvalidInput)
	}
	if password == "" {
		return fmt.Errorf("%w: password cannot be empty", ErrInvalidInput)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Debug("Storing password in OS keychain")

	err := keyring.Set(ServiceName, connectionID, password)
	if err != nil {
		return s.wrapKeychainError(err, "store password", connectionID)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Info("Password stored successfully in OS keychain")

	return nil
}

// GetPassword retrieves a password for the given connection ID from the OS keychain.
//
// Parameters:
//   - connectionID: unique identifier for the database connection
//
// Returns:
//   - password string and nil error on success
//   - empty string and ErrNotFound if credential doesn't exist
//   - empty string and ErrPermissionDenied if access is denied
//   - empty string and ErrUnavailable if keychain service is not available
//
// Note: Always check the error before using the returned password.
func (s *CredentialService) GetPassword(connectionID string) (string, error) {
	if connectionID == "" {
		return "", fmt.Errorf("%w: connectionID cannot be empty", ErrInvalidInput)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Debug("Retrieving password from OS keychain")

	password, err := keyring.Get(ServiceName, connectionID)
	if err != nil {
		return "", s.wrapKeychainError(err, "get password", connectionID)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Debug("Password retrieved successfully from OS keychain")

	return password, nil
}

// DeletePassword removes a password for the given connection ID from the OS keychain.
//
// Parameters:
//   - connectionID: unique identifier for the database connection
//
// Returns:
//   - nil on success or if credential doesn't exist (idempotent)
//   - ErrInvalidInput if connectionID is empty
//   - ErrPermissionDenied if access to keychain is denied
//   - ErrUnavailable if keychain service is not available
//
// This operation is idempotent - deleting a non-existent credential is not an error.
func (s *CredentialService) DeletePassword(connectionID string) error {
	if connectionID == "" {
		return fmt.Errorf("%w: connectionID cannot be empty", ErrInvalidInput)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Debug("Deleting password from OS keychain")

	err := keyring.Delete(ServiceName, connectionID)
	if err != nil {
		// If not found, treat as success (idempotent delete)
		if errors.Is(err, keyring.ErrNotFound) {
			s.logger.WithFields(logrus.Fields{
				"connection_id": connectionID,
			}).Debug("Password not found in keychain (already deleted)")
			return nil
		}

		return s.wrapKeychainError(err, "delete password", connectionID)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).Info("Password deleted successfully from OS keychain")

	return nil
}

// HasPassword checks if a password exists in the keychain for a given connection
func (s *CredentialService) HasPassword(connectionID string) bool {
	if connectionID == "" {
		return false
	}

	_, err := keyring.Get(ServiceName, connectionID)
	return err == nil
}

// UpdatePassword updates an existing password in the keychain
// This is essentially an alias for StorePassword as the keychain overwrites existing entries
func (s *CredentialService) UpdatePassword(connectionID, newPassword string) error {
	s.logger.WithField("connection_id", connectionID).Debug("Updating password in keychain")
	return s.StorePassword(connectionID, newPassword)
}

// GetPlatformInfo returns information about the current platform's keychain support.
// This can be used by clients to determine if OS keychain is available.
func (s *CredentialService) GetPlatformInfo() map[string]interface{} {
	platform := runtime.GOOS

	info := map[string]interface{}{
		"platform": platform,
		"service":  ServiceName,
	}

	switch platform {
	case "darwin":
		info["backend"] = "macOS Keychain"
		info["supported"] = true
		info["notes"] = "Uses Keychain Access.app - may prompt for password on first access"

	case "windows":
		info["backend"] = "Windows Credential Manager"
		info["supported"] = true
		info["notes"] = "Uses Windows Credential Manager - may prompt for access permission"

	case "linux":
		info["backend"] = "Secret Service API"
		info["supported"] = true
		info["notes"] = "Requires libsecret-1-0 package installed"
		info["install_hint"] = "sudo apt-get install libsecret-1-0"

	default:
		info["backend"] = "Unknown"
		info["supported"] = false
		info["notes"] = fmt.Sprintf("Platform %s not supported - use manual credential storage", platform)
	}

	return info
}

// HealthCheck performs a basic health check of the keychain service.
// It attempts to store and retrieve a test credential to verify functionality.
//
// Returns:
//   - nil if keychain is working properly
//   - error describing what's wrong if keychain is unavailable
func (s *CredentialService) HealthCheck() error {
	testKey := "__sqlstudio_health_check__"
	testValue := "test"

	// Try to store
	if err := s.StorePassword(testKey, testValue); err != nil {
		return fmt.Errorf("keychain health check failed (store): %w", err)
	}

	// Try to retrieve
	retrieved, err := s.GetPassword(testKey)
	if err != nil {
		// Clean up test credential if possible
		_ = s.DeletePassword(testKey)
		return fmt.Errorf("keychain health check failed (retrieve): %w", err)
	}

	// Verify value
	if retrieved != testValue {
		_ = s.DeletePassword(testKey)
		return fmt.Errorf("keychain health check failed: retrieved value doesn't match")
	}

	// Clean up
	if err := s.DeletePassword(testKey); err != nil {
		s.logger.Warn("Failed to clean up health check credential")
	}

	return nil
}

// wrapKeychainError wraps keyring errors with user-friendly messages and context.
// This internal helper ensures consistent error handling across all operations.
func (s *CredentialService) wrapKeychainError(err error, operation, connectionID string) error {
	if err == nil {
		return nil
	}

	// Log the internal error (but never log passwords)
	s.logger.WithFields(logrus.Fields{
		"operation":     operation,
		"connection_id": connectionID,
		"platform":      runtime.GOOS,
	}).WithError(err).Error("Keychain operation failed")

	// Map known keyring errors to our sentinel errors with helpful messages
	switch {
	case errors.Is(err, keyring.ErrNotFound):
		return fmt.Errorf("%w: credential for connection '%s' not found in keychain",
			ErrNotFound, connectionID)

	case errors.Is(err, keyring.ErrUnsupportedPlatform):
		return fmt.Errorf("%w: keychain not supported on %s - please store credentials manually",
			ErrUnavailable, runtime.GOOS)

	// Check for common permission/access denied patterns
	case containsAny(err.Error(),
		"denied", "permission", "access", "authorization", "unauthorized", "restricted"):
		return fmt.Errorf("%w: access to keychain denied - please check system permissions",
			ErrPermissionDenied)

	// Check for service unavailable patterns (especially Linux without libsecret)
	case containsAny(err.Error(),
		"unavailable", "not available", "not installed", "secret service", "dbus"):
		platform := runtime.GOOS
		msg := fmt.Sprintf("keychain service unavailable on %s", platform)
		if platform == "linux" {
			msg += " - install libsecret-1-0 or use manual credential storage"
		}
		return fmt.Errorf("%w: %s", ErrUnavailable, msg)

	default:
		// Return a wrapped generic error with operation context
		return fmt.Errorf("keychain %s failed for connection '%s': %w",
			operation, connectionID, err)
	}
}

// containsAny checks if a string contains any of the given substrings (case-insensitive)
func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, substr := range substrs {
		if strings.Contains(lower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
