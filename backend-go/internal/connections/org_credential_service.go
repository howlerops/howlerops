package connections

import (
	"context"
	"fmt"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/crypto"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
	"github.com/sirupsen/logrus"
)

// OrgCredentialService manages shared credential operations using OEK (Organization Envelope Key)
// This service provides secure credential sharing within organizations where:
// - Each organization has a single OEK (Organization Envelope Key)
// - Each member has their own encrypted copy of the OEK (encrypted with their master key)
// - Shared credentials are encrypted once with the OEK
// - All operations use proper Go concurrency patterns and crypto cleanup
type OrgCredentialService struct {
	oekStore         *turso.OrgEnvelopeKeyStore
	sharedCredStore  *turso.SharedCredentialStore
	auditStore       *turso.CredentialAuditStore
	masterKeyStore   *turso.MasterKeyStore
	credentialStore  *turso.CredentialStore
	logger           *logrus.Logger
}

// NewOrgCredentialService creates a new organization credential service
func NewOrgCredentialService(
	oekStore *turso.OrgEnvelopeKeyStore,
	sharedCredStore *turso.SharedCredentialStore,
	auditStore *turso.CredentialAuditStore,
	masterKeyStore *turso.MasterKeyStore,
	credentialStore *turso.CredentialStore,
	logger *logrus.Logger,
) *OrgCredentialService {
	return &OrgCredentialService{
		oekStore:        oekStore,
		sharedCredStore: sharedCredStore,
		auditStore:      auditStore,
		masterKeyStore:  masterKeyStore,
		credentialStore: credentialStore,
		logger:          logger,
	}
}

// ShareCredential shares a connection credential with an organization
// This encrypts the password with the organization's OEK and stores it
// If the organization doesn't have an OEK yet, one is generated and encrypted for the user
func (s *OrgCredentialService) ShareCredential(
	ctx context.Context,
	connID, orgID, userID string,
	userMasterKey []byte,
	password []byte,
) error {
	// Ensure master key cleanup
	defer crypto.ClearBytes(userMasterKey)
	defer crypto.ClearBytes(password)

	// Get or create OEK for the organization
	oek, err := s.getOrCreateOEK(ctx, orgID, userID, userMasterKey)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "share", false, err.Error())
		return fmt.Errorf("failed to get or create OEK: %w", err)
	}
	defer crypto.ClearBytes(oek)

	// Encrypt password with OEK
	encryptedPassword, err := crypto.EncryptWithOEK(password, oek)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "share", false, err.Error())
		return fmt.Errorf("failed to encrypt password with OEK: %w", err)
	}

	// Store shared credential
	sharedCred := &turso.SharedCredential{
		ConnectionID:      connID,
		OrganizationID:    orgID,
		EncryptedPassword: encryptedPassword.Ciphertext,
		PasswordIV:        encryptedPassword.IV,
		PasswordAuthTag:   encryptedPassword.AuthTag,
		CreatedBy:         userID,
		CreatedAt:         time.Now(),
	}

	if err := s.sharedCredStore.Store(ctx, sharedCred); err != nil {
		s.logAuditAsync(connID, orgID, userID, "share", false, err.Error())
		return fmt.Errorf("failed to store shared credential: %w", err)
	}

	// Log successful share (async)
	s.logAuditAsync(connID, orgID, userID, "share", true, "")

	s.logger.WithFields(logrus.Fields{
		"connection_id":   connID,
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Credential shared successfully")

	return nil
}

// GetSharedCredentialPassword retrieves and decrypts a shared credential
// This requires the user to have access to the organization's OEK
func (s *OrgCredentialService) GetSharedCredentialPassword(
	ctx context.Context,
	connID, orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	// Ensure master key cleanup
	defer crypto.ClearBytes(userMasterKey)

	// Get decrypted OEK for this user
	oek, err := s.getDecryptedOEK(ctx, orgID, userID, userMasterKey)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to get OEK: %w", err)
	}
	defer crypto.ClearBytes(oek)

	// Get shared credential
	sharedCred, err := s.sharedCredStore.Get(ctx, connID, orgID)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to get shared credential: %w", err)
	}

	// Convert to crypto format
	encryptedData := &crypto.OEKEncryptedData{
		Ciphertext: sharedCred.EncryptedPassword,
		IV:         sharedCred.PasswordIV,
		AuthTag:    sharedCred.PasswordAuthTag,
	}

	// Decrypt password with OEK
	password, err := crypto.DecryptWithOEK(encryptedData, oek)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Log successful access (async)
	s.logAuditAsync(connID, orgID, userID, "decrypt", true, "")

	s.logger.WithFields(logrus.Fields{
		"connection_id":   connID,
		"organization_id": orgID,
		"user_id":         userID,
	}).Debug("Shared credential decrypted successfully")

	// Caller is responsible for clearing the returned password
	return password, nil
}

// ProvisionOEKForNewMember provisions access to the organization's OEK for a new member
// This requires an existing member to decrypt the OEK and re-encrypt it for the new member
func (s *OrgCredentialService) ProvisionOEKForNewMember(
	ctx context.Context,
	orgID, existingUserID string,
	existingUserMasterKey []byte,
	newUserID string,
	newUserMasterKey []byte,
) error {
	// Ensure key cleanup
	defer crypto.ClearBytes(existingUserMasterKey)
	defer crypto.ClearBytes(newUserMasterKey)

	// Get existing member's encrypted OEK
	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, existingUserID)
	if err != nil {
		return fmt.Errorf("failed to get OEK for existing user: %w", err)
	}

	// Convert stored OEK to crypto format
	encryptedOEK := s.convertStoredOEKToCrypto(storedOEK)

	// Decrypt OEK with existing member's master key
	oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, existingUserMasterKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt OEK with existing member's master key: %w", err)
	}
	defer crypto.ClearBytes(oek)

	// Re-encrypt OEK with new member's master key
	newEncryptedOEK, err := crypto.EncryptOEKForUser(oek, newUserMasterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt OEK for new member: %w", err)
	}

	// Store OEK for new member (convert crypto type to turso type)
	tursoEncryptedOEK := s.convertCryptoOEKToTurso(newEncryptedOEK)
	if err := s.oekStore.StoreOEKForUser(ctx, orgID, newUserID, tursoEncryptedOEK); err != nil {
		return fmt.Errorf("failed to store OEK for new member: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id":    orgID,
		"existing_user_id":   existingUserID,
		"new_user_id":        newUserID,
	}).Info("OEK provisioned for new organization member")

	return nil
}

// RevokeOEKForMember revokes a member's access to the organization's OEK
// This prevents them from accessing any shared credentials
// Note: This does NOT re-encrypt existing shared credentials with a new OEK
// For full security, rotate all shared credentials after revoking member access
func (s *OrgCredentialService) RevokeOEKForMember(ctx context.Context, orgID, userID string) error {
	// Delete user's OEK copy
	if err := s.oekStore.DeleteOEKForUser(ctx, orgID, userID); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"organization_id": orgID,
			"user_id":         userID,
		}).Warn("Failed to revoke OEK for member")
		return fmt.Errorf("failed to revoke OEK: %w", err)
	}

	// Log revocation (async)
	go func() {
		s.auditStore.LogAccess(&turso.CredentialAccessLog{
			OrganizationID: orgID,
			UserID:         userID,
			Action:         "revoke_oek",
			Success:        true,
			Timestamp:      time.Now(),
		})
	}()

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("OEK revoked for organization member")

	return nil
}

// UnshareCredential removes a shared credential from an organization
// This makes the connection personal again
func (s *OrgCredentialService) UnshareCredential(ctx context.Context, connID, orgID string) error {
	// Delete shared credential
	if err := s.sharedCredStore.Delete(ctx, connID, orgID); err != nil {
		s.logAuditAsync(connID, orgID, "", "unshare", false, err.Error())
		return fmt.Errorf("failed to unshare credential: %w", err)
	}

	// Log successful unshare (async)
	s.logAuditAsync(connID, orgID, "", "unshare", true, "")

	s.logger.WithFields(logrus.Fields{
		"connection_id":   connID,
		"organization_id": orgID,
	}).Info("Credential unshared successfully")

	return nil
}

// getOrCreateOEK gets an existing OEK or creates a new one for the organization
// This is used when sharing the first credential in an organization
func (s *OrgCredentialService) getOrCreateOEK(
	ctx context.Context,
	orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	// Try to get existing OEK for this user
	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, userID)
	if err == nil {
		// User already has OEK - decrypt and return it
		encryptedOEK := s.convertStoredOEKToCrypto(storedOEK)
		oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, userMasterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt existing OEK: %w", err)
		}
		return oek, nil
	}

	// Check if any other member has the OEK
	_, err = s.oekStore.GetAnyOEKForOrg(ctx, orgID)
	if err == nil {
		// Organization has OEK but this user doesn't
		return nil, fmt.Errorf("user does not have access to organization's OEK, provision required")
	}

	// No OEK exists for organization - generate new one
	oek, err := crypto.GenerateOrgEnvelopeKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new OEK: %w", err)
	}

	// Encrypt OEK with user's master key
	encryptedOEK, err := crypto.EncryptOEKForUser(oek, userMasterKey)
	if err != nil {
		crypto.ClearBytes(oek)
		return nil, fmt.Errorf("failed to encrypt OEK for user: %w", err)
	}

	// Store encrypted OEK (convert crypto type to turso type)
	tursoOEK := s.convertCryptoOEKToTurso(encryptedOEK)
	if err := s.oekStore.StoreOEKForUser(ctx, orgID, userID, tursoOEK); err != nil {
		crypto.ClearBytes(oek)
		return nil, fmt.Errorf("failed to store OEK: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("New OEK generated and stored for organization")

	return oek, nil
}

// getDecryptedOEK is a helper method to retrieve and decrypt a user's OEK
// This implements the DRY principle for common OEK decryption operations
func (s *OrgCredentialService) getDecryptedOEK(
	ctx context.Context,
	orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	// Get user's encrypted OEK
	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("user does not have access to organization's OEK: %w", err)
	}

	// Convert to crypto format
	encryptedOEK := s.convertStoredOEKToCrypto(storedOEK)

	// Decrypt OEK with user's master key
	oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, userMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt OEK: %w", err)
	}

	return oek, nil
}

// convertStoredOEKToCrypto converts a stored OEK to crypto.OEKEncryptedData format
// This implements the DRY principle for format conversion
func (s *OrgCredentialService) convertStoredOEKToCrypto(stored *turso.StoredOEK) *crypto.OEKEncryptedData {
	return &crypto.OEKEncryptedData{
		Ciphertext: stored.EncryptedOEK,
		IV:         stored.OEKIV,
		AuthTag:    stored.OEKAuthTag,
	}
}

// convertCryptoOEKToTurso converts crypto.OEKEncryptedData to turso.OEKEncryptedData format
// This bridges the two packages' type definitions
func (s *OrgCredentialService) convertCryptoOEKToTurso(encrypted *crypto.OEKEncryptedData) *turso.OEKEncryptedData {
	return &turso.OEKEncryptedData{
		Ciphertext: encrypted.Ciphertext,
		IV:         encrypted.IV,
		AuthTag:    encrypted.AuthTag,
	}
}

// logAuditAsync logs an audit event asynchronously (fire-and-forget pattern)
// This prevents audit logging from blocking credential operations
func (s *OrgCredentialService) logAuditAsync(connID, orgID, userID, action string, success bool, errorMsg string) {
	go func() {
		s.auditStore.LogAccess(&turso.CredentialAccessLog{
			ConnectionID:   connID,
			OrganizationID: orgID,
			UserID:         userID,
			Action:         action,
			Success:        success,
			ErrorMessage:   errorMsg,
			Timestamp:      time.Now(),
		})
	}()
}
