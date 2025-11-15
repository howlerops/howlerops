package services

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/pkg/crypto"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// PasswordManager implements a hybrid dual-read system for password storage
// that enables zero-downtime migration from OS keychain to encrypted database storage.
//
// Read Priority:
//  1. Try encrypted_credentials table (Turso) - NEW SYSTEM
//  2. Fall back to OS keychain - LEGACY SYSTEM
//  3. Opportunistically migrate in background
//
// Write Strategy:
//  1. Store in encrypted_credentials (if master key available)
//  2. ALSO store in keychain (backup during transition)
//  3. Success if EITHER location works
type PasswordManager struct {
	credentialService *CredentialService     // OS keychain (legacy)
	credentialStore   *turso.CredentialStore // Encrypted DB (new)
	connectionStore   *turso.ConnectionStore // For migration status tracking
	logger            *logrus.Logger
	mu                sync.RWMutex
}

// NewPasswordManager creates a new password manager with hybrid storage
func NewPasswordManager(
	credentialService *CredentialService,
	credentialStore *turso.CredentialStore,
	connectionStore *turso.ConnectionStore,
	logger *logrus.Logger,
) *PasswordManager {
	return &PasswordManager{
		credentialService: credentialService,
		credentialStore:   credentialStore,
		connectionStore:   connectionStore,
		logger:            logger,
	}
}

// GetPassword retrieves a password using the hybrid dual-read approach
func (pm *PasswordManager) GetPassword(
	ctx context.Context,
	userID, connectionID string,
	masterKey []byte,
) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Priority 1: Try encrypted DB if master key is available
	if masterKey != nil {
		encryptedData, err := pm.credentialStore.GetCredential(ctx, userID, connectionID)
		if err != nil {
			pm.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":       userID,
				"connection_id": connectionID,
			}).Debug("Failed to retrieve password from encrypted DB, falling back to keychain")
		} else if encryptedData != nil {
			// Found in encrypted DB, decrypt it
			password, err := crypto.DecryptPasswordWithKey(encryptedData, masterKey)
			if err != nil {
				pm.logger.WithError(err).WithFields(logrus.Fields{
					"user_id":       userID,
					"connection_id": connectionID,
				}).Error("Failed to decrypt password from encrypted DB, falling back to keychain")
			} else {
				pm.logger.WithFields(logrus.Fields{
					"user_id":       userID,
					"connection_id": connectionID,
					"source":        "encrypted_db",
				}).Debug("Retrieved password from encrypted DB")
				return password, nil
			}
		}
	}

	// Priority 2: Fall back to keychain
	password, err := pm.credentialService.GetPassword(connectionID)
	if err != nil {
		pm.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Error("Failed to retrieve password from keychain")
		return "", err
	}

	pm.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"connection_id": connectionID,
		"source":        "keychain",
	}).Debug("Retrieved password from keychain")

	// Priority 3: Opportunistically migrate in background if we have master key
	if masterKey != nil {
		go pm.migratePasswordAsync(context.Background(), userID, connectionID, password, masterKey)
	}

	return password, nil
}

// StorePassword stores a password using the hybrid dual-write approach
func (pm *PasswordManager) StorePassword(
	ctx context.Context,
	userID, connectionID, password string,
	masterKey []byte,
) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var keychainErr, encryptedErr error

	// Strategy 1: Always store in keychain (backup during transition)
	keychainErr = pm.credentialService.StorePassword(connectionID, password)
	if keychainErr != nil {
		pm.logger.WithError(keychainErr).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Warn("Failed to store password in keychain")
	} else {
		pm.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Debug("Stored password in keychain")
	}

	// Strategy 2: If master key available, also store encrypted in DB
	if masterKey != nil {
		encryptedData, err := crypto.EncryptPasswordWithKey(password, masterKey)
		if err != nil {
			pm.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":       userID,
				"connection_id": connectionID,
			}).Error("Failed to encrypt password")
			encryptedErr = err
		} else {
			encryptedErr = pm.credentialStore.StoreCredential(ctx, userID, connectionID, encryptedData)
			if encryptedErr != nil {
				pm.logger.WithError(encryptedErr).WithFields(logrus.Fields{
					"user_id":       userID,
					"connection_id": connectionID,
				}).Error("Failed to store encrypted password in DB")
			} else {
				pm.logger.WithFields(logrus.Fields{
					"user_id":       userID,
					"connection_id": connectionID,
				}).Debug("Stored encrypted password in DB")

				// Update migration status to "migrated"
				if err := pm.connectionStore.UpdateMigrationStatus(ctx, connectionID, "migrated"); err != nil {
					pm.logger.WithError(err).WithFields(logrus.Fields{
						"user_id":       userID,
						"connection_id": connectionID,
					}).Warn("Failed to update migration status")
				}
			}
		}
	}

	// Success if EITHER location worked
	if keychainErr != nil && encryptedErr != nil {
		pm.logger.WithFields(logrus.Fields{
			"user_id":         userID,
			"connection_id":   connectionID,
			"keychain_error":  keychainErr.Error(),
			"encrypted_error": encryptedErr.Error(),
		}).Error("Failed to store password in both locations")
		return keychainErr // Return keychain error as primary
	}

	return nil
}

// DeletePassword removes a password from both storage locations
func (pm *PasswordManager) DeletePassword(
	ctx context.Context,
	userID, connectionID string,
) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var keychainErr, encryptedErr error

	// Delete from keychain
	keychainErr = pm.credentialService.DeletePassword(connectionID)
	if keychainErr != nil {
		pm.logger.WithError(keychainErr).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Warn("Failed to delete password from keychain")
	} else {
		pm.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Debug("Deleted password from keychain")
	}

	// Delete from encrypted DB
	encryptedErr = pm.credentialStore.DeleteCredential(ctx, userID, connectionID)
	if encryptedErr != nil {
		pm.logger.WithError(encryptedErr).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Warn("Failed to delete password from encrypted DB")
	} else {
		pm.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Debug("Deleted password from encrypted DB")
	}

	// Success if at least one deletion worked (or both failed with not-found)
	if keychainErr != nil && encryptedErr != nil {
		pm.logger.WithFields(logrus.Fields{
			"user_id":         userID,
			"connection_id":   connectionID,
			"keychain_error":  keychainErr.Error(),
			"encrypted_error": encryptedErr.Error(),
		}).Warn("Failed to delete password from both locations")
		// Still return nil if both failed - password might not exist in either location
	}

	return nil
}

// GetMigrationStatus returns the migration status for a connection
func (pm *PasswordManager) GetMigrationStatus(
	ctx context.Context,
	connectionID string,
) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status, err := pm.connectionStore.GetMigrationStatus(ctx, connectionID)
	if err != nil {
		pm.logger.WithError(err).WithField("connection_id", connectionID).
			Error("Failed to get migration status")
		return "not_migrated", err
	}

	return status, nil
}

// migratePasswordAsync performs opportunistic background migration of a password
// from keychain to encrypted DB storage
func (pm *PasswordManager) migratePasswordAsync(
	ctx context.Context,
	userID, connectionID, password string,
	masterKey []byte,
) {
	pm.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"connection_id": connectionID,
	}).Debug("Starting opportunistic password migration")

	// Check if already migrated
	encryptedData, err := pm.credentialStore.GetCredential(ctx, userID, connectionID)
	if err != nil {
		pm.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Debug("Failed to check if password already migrated")
	}

	if encryptedData != nil {
		pm.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Debug("Password already migrated, skipping")
		return
	}

	// Encrypt password
	encryptedData, err = crypto.EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		pm.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Error("Failed to encrypt password during migration")
		return
	}

	// Store in encrypted DB
	if err := pm.credentialStore.StoreCredential(ctx, userID, connectionID, encryptedData); err != nil {
		pm.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Error("Failed to store encrypted password during migration")
		return
	}

	// Update migration status
	if err := pm.connectionStore.UpdateMigrationStatus(ctx, connectionID, "migrated"); err != nil {
		pm.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":       userID,
			"connection_id": connectionID,
		}).Warn("Failed to update migration status")
		// Don't return - migration still succeeded
	}

	pm.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"connection_id": connectionID,
	}).Info("Successfully migrated password to encrypted DB")
}
