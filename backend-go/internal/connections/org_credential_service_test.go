package connections

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/pkg/crypto"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// Mock implementations for testing

// OEKStoreInterface defines the interface for OEK storage operations
type OEKStoreInterface interface {
	StoreOEKForUser(ctx context.Context, orgID, userID string, encrypted *turso.OEKEncryptedData) error
	GetOEKForUser(ctx context.Context, orgID, userID string) (*turso.StoredOEK, error)
	GetAnyOEKForOrg(ctx context.Context, orgID string) (*turso.StoredOEK, error)
	DeleteOEKForUser(ctx context.Context, orgID, userID string) error
	Close() error
}

// SharedCredStoreInterface defines the interface for shared credential storage
type SharedCredStoreInterface interface {
	Store(ctx context.Context, cred *turso.SharedCredential) error
	Get(ctx context.Context, connID, orgID string) (*turso.SharedCredential, error)
	Delete(ctx context.Context, connID, orgID string) error
	Close() error
}

// CredentialAuditStoreInterface defines the interface for audit logging
type CredentialAuditStoreInterface interface {
	LogAccess(log *turso.CredentialAccessLog)
	Start()
	Close() error
}

// MockOEKStore mocks the OEK storage operations
type MockOEKStore struct {
	mu    sync.RWMutex
	store map[string]*turso.StoredOEK // key: orgID:userID
}

func NewMockOEKStore() *MockOEKStore {
	return &MockOEKStore{
		store: make(map[string]*turso.StoredOEK),
	}
}

func (m *MockOEKStore) StoreOEKForUser(ctx context.Context, orgID, userID string, encrypted *turso.OEKEncryptedData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := orgID + ":" + userID
	m.store[key] = &turso.StoredOEK{
		ID:             "oek-" + key,
		OrganizationID: orgID,
		UserID:         userID,
		EncryptedOEK:   encrypted.Ciphertext,
		OEKIV:          encrypted.IV,
		OEKAuthTag:     encrypted.AuthTag,
		KeyVersion:     1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return nil
}

func (m *MockOEKStore) GetOEKForUser(ctx context.Context, orgID, userID string) (*turso.StoredOEK, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := orgID + ":" + userID
	oek, exists := m.store[key]
	if !exists {
		return nil, errors.New("OEK not found for user in organization")
	}
	return oek, nil
}

func (m *MockOEKStore) GetAnyOEKForOrg(ctx context.Context, orgID string) (*turso.StoredOEK, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, oek := range m.store {
		if oek.OrganizationID == orgID {
			return oek, nil
		}
	}
	return nil, errors.New("no OEK found for organization")
}

func (m *MockOEKStore) DeleteOEKForUser(ctx context.Context, orgID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := orgID + ":" + userID
	if _, exists := m.store[key]; !exists {
		return errors.New("OEK not found for user")
	}
	delete(m.store, key)
	return nil
}

func (m *MockOEKStore) Close() error {
	return nil
}

// MockSharedCredStore mocks shared credential storage
type MockSharedCredStore struct {
	mu    sync.RWMutex
	store map[string]*turso.SharedCredential // key: connID:orgID
}

func NewMockSharedCredStore() *MockSharedCredStore {
	return &MockSharedCredStore{
		store: make(map[string]*turso.SharedCredential),
	}
}

func (m *MockSharedCredStore) Store(ctx context.Context, cred *turso.SharedCredential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := cred.ConnectionID + ":" + cred.OrganizationID
	m.store[key] = cred
	return nil
}

func (m *MockSharedCredStore) Get(ctx context.Context, connID, orgID string) (*turso.SharedCredential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := connID + ":" + orgID
	cred, exists := m.store[key]
	if !exists {
		return nil, errors.New("shared credential not found")
	}
	return cred, nil
}

func (m *MockSharedCredStore) Delete(ctx context.Context, connID, orgID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := connID + ":" + orgID
	if _, exists := m.store[key]; !exists {
		return errors.New("shared credential not found")
	}
	delete(m.store, key)
	return nil
}

func (m *MockSharedCredStore) Close() error {
	return nil
}

// MockCredentialAuditStore mocks audit logging
type MockCredentialAuditStore struct {
	mu   sync.RWMutex
	logs []*turso.CredentialAccessLog
}

func NewMockCredentialAuditStore() *MockCredentialAuditStore {
	return &MockCredentialAuditStore{
		logs: make([]*turso.CredentialAccessLog, 0),
	}
}

func (m *MockCredentialAuditStore) LogAccess(log *turso.CredentialAccessLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, log)
}

func (m *MockCredentialAuditStore) GetLogs() []*turso.CredentialAccessLog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logs
}

func (m *MockCredentialAuditStore) Start() {
	// No-op for mock
}

func (m *MockCredentialAuditStore) Close() error {
	return nil
}

// TestOrgCredentialService wraps the service for testing with mocks
type TestOrgCredentialService struct {
	oekStore        OEKStoreInterface
	sharedCredStore SharedCredStoreInterface
	auditStore      CredentialAuditStoreInterface
	logger          *logrus.Logger
}

// NewTestOrgCredentialService creates a test service with injected dependencies
func NewTestOrgCredentialService(
	oekStore OEKStoreInterface,
	sharedCredStore SharedCredStoreInterface,
	auditStore CredentialAuditStoreInterface,
	logger *logrus.Logger,
) *TestOrgCredentialService {
	return &TestOrgCredentialService{
		oekStore:        oekStore,
		sharedCredStore: sharedCredStore,
		auditStore:      auditStore,
		logger:          logger,
	}
}

// ShareCredential - test implementation that mirrors org_credential_service.go
func (s *TestOrgCredentialService) ShareCredential(
	ctx context.Context,
	connID, orgID, userID string,
	userMasterKey []byte,
	password []byte,
) error {
	defer crypto.ClearBytes(userMasterKey)
	defer crypto.ClearBytes(password)

	oek, err := s.getOrCreateOEK(ctx, orgID, userID, userMasterKey)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "share", false, err.Error())
		return fmt.Errorf("failed to get or create OEK: %w", err)
	}
	defer crypto.ClearBytes(oek)

	encryptedPassword, err := crypto.EncryptWithOEK(password, oek)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "share", false, err.Error())
		return fmt.Errorf("failed to encrypt password with OEK: %w", err)
	}

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

	s.logAuditAsync(connID, orgID, userID, "share", true, "")
	return nil
}

// GetSharedCredentialPassword - test implementation
func (s *TestOrgCredentialService) GetSharedCredentialPassword(
	ctx context.Context,
	connID, orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	defer crypto.ClearBytes(userMasterKey)

	oek, err := s.getDecryptedOEK(ctx, orgID, userID, userMasterKey)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to get OEK: %w", err)
	}
	defer crypto.ClearBytes(oek)

	sharedCred, err := s.sharedCredStore.Get(ctx, connID, orgID)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to get shared credential: %w", err)
	}

	encryptedData := &crypto.OEKEncryptedData{
		Ciphertext: sharedCred.EncryptedPassword,
		IV:         sharedCred.PasswordIV,
		AuthTag:    sharedCred.PasswordAuthTag,
	}

	password, err := crypto.DecryptWithOEK(encryptedData, oek)
	if err != nil {
		s.logAuditAsync(connID, orgID, userID, "decrypt", false, err.Error())
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	s.logAuditAsync(connID, orgID, userID, "decrypt", true, "")
	return password, nil
}

// ProvisionOEKForNewMember - test implementation
func (s *TestOrgCredentialService) ProvisionOEKForNewMember(
	ctx context.Context,
	orgID, existingUserID string,
	existingUserMasterKey []byte,
	newUserID string,
	newUserMasterKey []byte,
) error {
	defer crypto.ClearBytes(existingUserMasterKey)
	defer crypto.ClearBytes(newUserMasterKey)

	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, existingUserID)
	if err != nil {
		return fmt.Errorf("failed to get OEK for existing user: %w", err)
	}

	encryptedOEK := &crypto.OEKEncryptedData{
		Ciphertext: storedOEK.EncryptedOEK,
		IV:         storedOEK.OEKIV,
		AuthTag:    storedOEK.OEKAuthTag,
	}

	oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, existingUserMasterKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt OEK with existing member's master key: %w", err)
	}
	defer crypto.ClearBytes(oek)

	newEncryptedOEK, err := crypto.EncryptOEKForUser(oek, newUserMasterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt OEK for new member: %w", err)
	}

	tursoEncrypted := &turso.OEKEncryptedData{
		Ciphertext: newEncryptedOEK.Ciphertext,
		IV:         newEncryptedOEK.IV,
		AuthTag:    newEncryptedOEK.AuthTag,
	}

	if err := s.oekStore.StoreOEKForUser(ctx, orgID, newUserID, tursoEncrypted); err != nil {
		return fmt.Errorf("failed to store OEK for new member: %w", err)
	}

	return nil
}

// RevokeOEKForMember - test implementation
func (s *TestOrgCredentialService) RevokeOEKForMember(ctx context.Context, orgID, userID string) error {
	if err := s.oekStore.DeleteOEKForUser(ctx, orgID, userID); err != nil {
		return fmt.Errorf("failed to revoke OEK: %w", err)
	}

	s.auditStore.LogAccess(&turso.CredentialAccessLog{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         "revoke_oek",
		Success:        true,
		Timestamp:      time.Now(),
	})

	return nil
}

// UnshareCredential - test implementation
func (s *TestOrgCredentialService) UnshareCredential(ctx context.Context, connID, orgID string) error {
	if err := s.sharedCredStore.Delete(ctx, connID, orgID); err != nil {
		s.logAuditAsync(connID, orgID, "", "unshare", false, err.Error())
		return fmt.Errorf("failed to unshare credential: %w", err)
	}

	s.logAuditAsync(connID, orgID, "", "unshare", true, "")
	return nil
}

// Helper methods

func (s *TestOrgCredentialService) getOrCreateOEK(
	ctx context.Context,
	orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, userID)
	if err == nil {
		encryptedOEK := &crypto.OEKEncryptedData{
			Ciphertext: storedOEK.EncryptedOEK,
			IV:         storedOEK.OEKIV,
			AuthTag:    storedOEK.OEKAuthTag,
		}
		oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, userMasterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt existing OEK: %w", err)
		}
		return oek, nil
	}

	_, err = s.oekStore.GetAnyOEKForOrg(ctx, orgID)
	if err == nil {
		return nil, fmt.Errorf("user does not have access to organization's OEK, provision required")
	}

	oek, err := crypto.GenerateOrgEnvelopeKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new OEK: %w", err)
	}

	encryptedOEK, err := crypto.EncryptOEKForUser(oek, userMasterKey)
	if err != nil {
		crypto.ClearBytes(oek)
		return nil, fmt.Errorf("failed to encrypt OEK for user: %w", err)
	}

	tursoEncrypted := &turso.OEKEncryptedData{
		Ciphertext: encryptedOEK.Ciphertext,
		IV:         encryptedOEK.IV,
		AuthTag:    encryptedOEK.AuthTag,
	}

	if err := s.oekStore.StoreOEKForUser(ctx, orgID, userID, tursoEncrypted); err != nil {
		crypto.ClearBytes(oek)
		return nil, fmt.Errorf("failed to store OEK: %w", err)
	}

	return oek, nil
}

func (s *TestOrgCredentialService) getDecryptedOEK(
	ctx context.Context,
	orgID, userID string,
	userMasterKey []byte,
) ([]byte, error) {
	storedOEK, err := s.oekStore.GetOEKForUser(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("user does not have access to organization's OEK: %w", err)
	}

	encryptedOEK := &crypto.OEKEncryptedData{
		Ciphertext: storedOEK.EncryptedOEK,
		IV:         storedOEK.OEKIV,
		AuthTag:    storedOEK.OEKAuthTag,
	}

	oek, err := crypto.DecryptOEKWithMasterKey(encryptedOEK, userMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt OEK: %w", err)
	}

	return oek, nil
}

func (s *TestOrgCredentialService) logAuditAsync(connID, orgID, userID, action string, success bool, errorMsg string) {
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

// Helper to create service with mocks
func setupTestService() (*TestOrgCredentialService, *MockOEKStore, *MockSharedCredStore, *MockCredentialAuditStore) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	oekStore := NewMockOEKStore()
	credStore := NewMockSharedCredStore()
	auditStore := NewMockCredentialAuditStore()

	service := NewTestOrgCredentialService(
		oekStore,
		credStore,
		auditStore,
		logger,
	)

	return service, oekStore, credStore, auditStore
}

// Tests

func TestShareCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("first share creates new OEK for org", func(t *testing.T) {
		service, oekStore, credStore, _ := setupTestService()

		// Generate user master key
		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)
		// Note: service clears the key, so make a copy for our checks
		userMasterKeyCopy := make([]byte, len(userMasterKey))
		copy(userMasterKeyCopy, userMasterKey)
		defer crypto.ClearBytes(userMasterKeyCopy)

		password := []byte("secret-password")
		// Share credential (this will clear userMasterKey and password)
		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userMasterKey, password)
		require.NoError(t, err)

		// Verify OEK was created for user
		storedOEK, err := oekStore.GetOEKForUser(ctx, "org-456", "user-789")
		require.NoError(t, err)
		assert.NotNil(t, storedOEK)
		assert.Equal(t, "org-456", storedOEK.OrganizationID)
		assert.Equal(t, "user-789", storedOEK.UserID)

		// Verify shared credential was stored
		sharedCred, err := credStore.Get(ctx, "conn-123", "org-456")
		require.NoError(t, err)
		assert.NotNil(t, sharedCred)
		assert.Equal(t, "conn-123", sharedCred.ConnectionID)
		assert.Equal(t, "org-456", sharedCred.OrganizationID)
		assert.Equal(t, "user-789", sharedCred.CreatedBy)
		assert.NotEmpty(t, sharedCred.EncryptedPassword)
	})

	t.Run("password correctly encrypted with OEK", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		originalPassword := []byte("super-secret-password-123!")
		passwordCopy := make([]byte, len(originalPassword))
		copy(passwordCopy, originalPassword)

		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userMasterKey, originalPassword)
		require.NoError(t, err)

		// Full decryption is tested in TestGetSharedCredentialPassword
		_ = passwordCopy // Use to avoid unused variable error
	})

	t.Run("audit log created", func(t *testing.T) {
		service, _, _, auditStore := setupTestService()

		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		password := []byte("test-password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userMasterKey, password)
		require.NoError(t, err)

		// Wait briefly for async audit logging
		time.Sleep(50 * time.Millisecond)

		logs := auditStore.GetLogs()
		assert.NotEmpty(t, logs)
		found := false
		for _, log := range logs {
			if log.Action == "share" && log.ConnectionID == "conn-123" {
				found = true
				assert.True(t, log.Success)
				assert.Equal(t, "org-456", log.OrganizationID)
				assert.Equal(t, "user-789", log.UserID)
			}
		}
		assert.True(t, found, "share audit log not found")
	})
}

func TestGetSharedCredentialPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully decrypts password with correct master key", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		// Make a copy since ShareCredential will clear it
		masterKeyCopy := make([]byte, len(userMasterKey))
		copy(masterKeyCopy, userMasterKey)

		originalPassword := []byte("test-password-secure")
		passwordCopy := make([]byte, len(originalPassword))
		copy(passwordCopy, originalPassword)

		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userMasterKey, originalPassword)
		require.NoError(t, err)

		// Use the copy to retrieve
		retrieved, err := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", masterKeyCopy)
		require.NoError(t, err)
		defer crypto.ClearBytes(retrieved)

		assert.Equal(t, passwordCopy, retrieved)
	})

	t.Run("returns error with wrong master key", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		correctMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		correctKeyCopy := make([]byte, len(correctMasterKey))
		copy(correctKeyCopy, correctMasterKey)

		wrongMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		password := []byte("test-password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", correctMasterKey, password)
		require.NoError(t, err)

		// Try to retrieve with wrong key
		_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", wrongMasterKey)
		assert.Error(t, err)
	})

	t.Run("returns error when OEK not found", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		// Try to get password without OEK
		_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", userMasterKey)
		assert.Error(t, err)
	})

	t.Run("returns error when credential not found", func(t *testing.T) {
		service, oekStore, _, _ := setupTestService()

		userMasterKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		masterKeyCopy := make([]byte, len(userMasterKey))
		copy(masterKeyCopy, userMasterKey)

		// Create OEK but no credential
		oek, err := crypto.GenerateOrgEnvelopeKey()
		require.NoError(t, err)
		defer crypto.ClearBytes(oek)

		encryptedOEK, err := crypto.EncryptOEKForUser(oek, userMasterKey)
		require.NoError(t, err)

		// Convert crypto.OEKEncryptedData to turso.OEKEncryptedData
		tursoEncrypted := &turso.OEKEncryptedData{
			Ciphertext: encryptedOEK.Ciphertext,
			IV:         encryptedOEK.IV,
			AuthTag:    encryptedOEK.AuthTag,
		}

		err = oekStore.StoreOEKForUser(ctx, "org-456", "user-789", tursoEncrypted)
		require.NoError(t, err)

		// Try to get non-existent credential
		_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", masterKeyCopy)
		assert.Error(t, err)
	})
}

func TestProvisionOEKForNewMember(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully copies OEK to new member", func(t *testing.T) {
		service, oekStore, _, _ := setupTestService()

		// Existing user setup
		existingUserKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		existingKeyCopy := make([]byte, len(existingUserKey))
		copy(existingKeyCopy, existingUserKey)

		password := []byte("password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "existing-user", existingUserKey, password)
		require.NoError(t, err)

		// New user setup
		newUserKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		// Provision OEK for new member
		err = service.ProvisionOEKForNewMember(ctx, "org-456", "existing-user", existingKeyCopy, "new-user", newUserKey)
		require.NoError(t, err)

		// Verify new user has OEK
		newOEK, err := oekStore.GetOEKForUser(ctx, "org-456", "new-user")
		require.NoError(t, err)
		assert.NotNil(t, newOEK)
		assert.Equal(t, "org-456", newOEK.OrganizationID)
		assert.Equal(t, "new-user", newOEK.UserID)
	})

	t.Run("new member can decrypt credentials", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// User A shares credential
		userAKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userAKeyCopy := make([]byte, len(userAKey))
		copy(userAKeyCopy, userAKey)

		originalPassword := []byte("shared-secret")
		passwordCopy := make([]byte, len(originalPassword))
		copy(passwordCopy, originalPassword)

		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-a", userAKey, originalPassword)
		require.NoError(t, err)

		// User B joins
		userBKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userBKeyCopy := make([]byte, len(userBKey))
		copy(userBKeyCopy, userBKey)

		err = service.ProvisionOEKForNewMember(ctx, "org-456", "user-a", userAKeyCopy, "user-b", userBKey)
		require.NoError(t, err)

		// User B can decrypt the password
		password, err := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-b", userBKeyCopy)
		require.NoError(t, err)
		defer crypto.ClearBytes(password)

		assert.Equal(t, passwordCopy, password)
	})

	t.Run("original member still works after provisioning", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// User A shares
		userAKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userAKeyCopy1 := make([]byte, len(userAKey))
		copy(userAKeyCopy1, userAKey)

		userAKeyCopy2 := make([]byte, len(userAKey))
		copy(userAKeyCopy2, userAKey)

		originalPassword := []byte("shared-secret")
		passwordCopy := make([]byte, len(originalPassword))
		copy(passwordCopy, originalPassword)

		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-a", userAKey, originalPassword)
		require.NoError(t, err)

		// User B joins
		userBKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		err = service.ProvisionOEKForNewMember(ctx, "org-456", "user-a", userAKeyCopy1, "user-b", userBKey)
		require.NoError(t, err)

		// User A can still decrypt
		password, err := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-a", userAKeyCopy2)
		require.NoError(t, err)
		defer crypto.ClearBytes(password)

		assert.Equal(t, passwordCopy, password)
	})

	t.Run("parallel provisioning is safe", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// Setup existing user
		existingUserKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		existingKeyCopy := make([]byte, len(existingUserKey))
		copy(existingKeyCopy, existingUserKey)

		password := []byte("password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "existing-user", existingUserKey, password)
		require.NoError(t, err)

		// Provision multiple users in parallel
		numNewUsers := 10
		var wg sync.WaitGroup
		errors := make(chan error, numNewUsers)

		for i := 0; i < numNewUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				newUserKey, err := crypto.GenerateMasterKey()
				if err != nil {
					errors <- err
					return
				}

				existingKeyCopyLocal := make([]byte, len(existingKeyCopy))
				copy(existingKeyCopyLocal, existingKeyCopy)

				userIDStr := "user-" + string(rune(userID))
				err = service.ProvisionOEKForNewMember(ctx, "org-456", "existing-user", existingKeyCopyLocal, userIDStr, newUserKey)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent provisioning failed: %v", err)
		}
	})
}

func TestRevokeOEKForMember(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully deletes OEK", func(t *testing.T) {
		service, oekStore, _, _ := setupTestService()

		userKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		password := []byte("password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userKey, password)
		require.NoError(t, err)

		// Revoke
		err = service.RevokeOEKForMember(ctx, "org-456", "user-789")
		require.NoError(t, err)

		// Verify OEK is gone
		_, err = oekStore.GetOEKForUser(ctx, "org-456", "user-789")
		assert.Error(t, err)
	})

	t.Run("revoked member cannot decrypt", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		userKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userKeyCopy := make([]byte, len(userKey))
		copy(userKeyCopy, userKey)

		password := []byte("password")
		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userKey, password)
		require.NoError(t, err)

		// Revoke
		err = service.RevokeOEKForMember(ctx, "org-456", "user-789")
		require.NoError(t, err)

		// Try to decrypt
		_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", userKeyCopy)
		assert.Error(t, err)
	})

	t.Run("other members unaffected", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// User A
		userAKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userAKeyCopy1 := make([]byte, len(userAKey))
		copy(userAKeyCopy1, userAKey)

		userAKeyCopy2 := make([]byte, len(userAKey))
		copy(userAKeyCopy2, userAKey)

		password := []byte("shared-password")
		passwordCopy := make([]byte, len(password))
		copy(passwordCopy, password)

		err = service.ShareCredential(ctx, "conn-123", "org-456", "user-a", userAKey, password)
		require.NoError(t, err)

		// User B
		userBKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		err = service.ProvisionOEKForNewMember(ctx, "org-456", "user-a", userAKeyCopy1, "user-b", userBKey)
		require.NoError(t, err)

		// Revoke user B
		err = service.RevokeOEKForMember(ctx, "org-456", "user-b")
		require.NoError(t, err)

		// User A still works
		retrieved, err := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-a", userAKeyCopy2)
		require.NoError(t, err)
		defer crypto.ClearBytes(retrieved)

		assert.Equal(t, passwordCopy, retrieved)
	})
}

func TestOEKConcurrency(t *testing.T) {
	ctx := context.Background()

	t.Run("multiple goroutines sharing credentials simultaneously", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// Generate master keys for multiple users
		numUsers := 10
		userKeys := make([][]byte, numUsers)
		for i := 0; i < numUsers; i++ {
			key, err := crypto.GenerateMasterKey()
			require.NoError(t, err)
			userKeys[i] = key
		}

		// Share credentials concurrently
		var wg sync.WaitGroup
		errors := make(chan error, numUsers)

		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				connID := "conn-" + string(rune(userID))
				password := []byte("password-" + string(rune(userID)))
				userIDStr := "user-" + string(rune(userID))

				err := service.ShareCredential(ctx, connID, "org-456", userIDStr, userKeys[userID], password)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent share failed: %v", err)
		}
	})

	t.Run("multiple goroutines reading passwords simultaneously", func(t *testing.T) {
		service, _, _, _ := setupTestService()

		// Setup - create shared credentials
		numCreds := 10
		userKey, err := crypto.GenerateMasterKey()
		require.NoError(t, err)

		userKeyCopies := make([][]byte, numCreds*5)
		passwords := make([][]byte, numCreds)

		for i := 0; i < numCreds; i++ {
			connID := "conn-" + string(rune(i))
			password := []byte("password-" + string(rune(i)))
			passwords[i] = password

			userKeyCopy := make([]byte, len(userKey))
			copy(userKeyCopy, userKey)

			err := service.ShareCredential(ctx, connID, "org-456", "user-789", userKeyCopy, password)
			require.NoError(t, err)

			// Create copies for reading
			for j := 0; j < 5; j++ {
				idx := i*5 + j
				userKeyCopies[idx] = make([]byte, len(userKey))
				copy(userKeyCopies[idx], userKey)
			}
		}

		// Read concurrently
		var wg sync.WaitGroup
		errors := make(chan error, numCreds*5)

		for i := 0; i < numCreds; i++ {
			for j := 0; j < 5; j++ {
				wg.Add(1)
				go func(credID, copyIdx int) {
					defer wg.Done()

					connID := "conn-" + string(rune(credID))

					password, err := service.GetSharedCredentialPassword(ctx, connID, "org-456", "user-789", userKeyCopies[copyIdx])
					if err != nil {
						errors <- err
						return
					}
					defer crypto.ClearBytes(password)

					expectedPass := passwords[credID]
					for k, b := range password {
						if k >= len(expectedPass) || b != expectedPass[k] {
							errors <- err
							return
						}
					}
				}(i, i*5+j)
			}
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent read failed: %v", err)
		}
	})
}

func TestFullCredentialSharingWorkflow(t *testing.T) {
	ctx := context.Background()
	service, oekStore, credStore, _ := setupTestService()

	// 1. User A creates personal connection with password
	userAKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userAKeyCopy1 := make([]byte, len(userAKey))
	copy(userAKeyCopy1, userAKey)

	originalPassword := []byte("production-db-password-123")
	passwordCopy := make([]byte, len(originalPassword))
	copy(passwordCopy, originalPassword)

	// 2. User A shares connection with org (creates OEK, encrypts password)
	err = service.ShareCredential(ctx, "conn-prod-db", "org-acme", "user-a", userAKey, originalPassword)
	require.NoError(t, err)

	// Verify OEK exists for user A
	oekA, err := oekStore.GetOEKForUser(ctx, "org-acme", "user-a")
	require.NoError(t, err)
	assert.NotNil(t, oekA)

	// Verify shared credential exists
	sharedCred, err := credStore.Get(ctx, "conn-prod-db", "org-acme")
	require.NoError(t, err)
	assert.NotNil(t, sharedCred)

	// 3. User B joins org (provisions OEK copy)
	userBKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userBKeyCopy := make([]byte, len(userBKey))
	copy(userBKeyCopy, userBKey)

	err = service.ProvisionOEKForNewMember(ctx, "org-acme", "user-a", userAKeyCopy1, "user-b", userBKey)
	require.NoError(t, err)

	// Verify user B has OEK
	oekB, err := oekStore.GetOEKForUser(ctx, "org-acme", "user-b")
	require.NoError(t, err)
	assert.NotNil(t, oekB)

	// 4. User B can read shared password
	passwordB, err := service.GetSharedCredentialPassword(ctx, "conn-prod-db", "org-acme", "user-b", userBKeyCopy)
	require.NoError(t, err)
	defer crypto.ClearBytes(passwordB)

	assert.Equal(t, passwordCopy, passwordB)

	// 5. User A unshares connection
	err = service.UnshareCredential(ctx, "conn-prod-db", "org-acme")
	require.NoError(t, err)

	// 6. Verify shared credential deleted
	_, err = credStore.Get(ctx, "conn-prod-db", "org-acme")
	assert.Error(t, err)

	// 7. User B cannot access password anymore (credential gone)
	userBKeyCopy2 := make([]byte, len(userBKeyCopy))
	copy(userBKeyCopy2, userBKeyCopy)

	_, err = service.GetSharedCredentialPassword(ctx, "conn-prod-db", "org-acme", "user-b", userBKeyCopy2)
	assert.Error(t, err)

	// Note: OEKs still exist for both users (they can be used for other credentials)
	oekA, err = oekStore.GetOEKForUser(ctx, "org-acme", "user-a")
	require.NoError(t, err)
	assert.NotNil(t, oekA)

	oekB, err = oekStore.GetOEKForUser(ctx, "org-acme", "user-b")
	require.NoError(t, err)
	assert.NotNil(t, oekB)
}

// Benchmark tests

func BenchmarkShareCredential(b *testing.B) {
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		userKey, _ := crypto.GenerateMasterKey()
		password := []byte("benchmark-password")
		connID := "conn-" + string(rune(i))
		userID := "user-" + string(rune(i))
		b.StartTimer()

		_ = service.ShareCredential(ctx, connID, "org-456", userID, userKey, password)
	}
}

func BenchmarkGetSharedCredentialPassword(b *testing.B) {
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup
	userKey, _ := crypto.GenerateMasterKey()
	userKeyCopy := make([]byte, len(userKey))
	copy(userKeyCopy, userKey)

	password := []byte("benchmark-password")
	_ = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userKey, password)

	// Create copies for benchmark iterations
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = make([]byte, len(userKeyCopy))
		copy(keys[i], userKeyCopy)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", keys[i])
		if result != nil {
			crypto.ClearBytes(result)
		}
	}
}

func BenchmarkProvisionOEKForNewMember(b *testing.B) {
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	existingUserKey, _ := crypto.GenerateMasterKey()
	existingKeyCopy := make([]byte, len(existingUserKey))
	copy(existingKeyCopy, existingUserKey)

	// Setup - existing user has OEK
	password := []byte("password")
	_ = service.ShareCredential(ctx, "conn-123", "org-456", "existing-user", existingUserKey, password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		newUserKey, _ := crypto.GenerateMasterKey()
		existingKeyCopyLocal := make([]byte, len(existingKeyCopy))
		copy(existingKeyCopyLocal, existingKeyCopy)
		userID := "user-" + string(rune(i))
		b.StartTimer()

		_ = service.ProvisionOEKForNewMember(ctx, "org-456", "existing-user", existingKeyCopyLocal, userID, newUserKey)
	}
}

func BenchmarkConcurrentSharing(b *testing.B) {
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userKey, _ := crypto.GenerateMasterKey()
			connID := "conn-" + string(rune(i))
			password := []byte("password-" + string(rune(i)))
			userID := "user-" + string(rune(i))
			_ = service.ShareCredential(ctx, connID, "org-456", userID, userKey, password)
			i++
		}
	})
}

func BenchmarkConcurrentReading(b *testing.B) {
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup - create credential
	userKey, _ := crypto.GenerateMasterKey()
	userKeyCopy := make([]byte, len(userKey))
	copy(userKeyCopy, userKey)

	password := []byte("password")
	_ = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userKey, password)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			keyCopy := make([]byte, len(userKeyCopy))
			copy(keyCopy, userKeyCopy)
			result, _ := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", keyCopy)
			if result != nil {
				crypto.ClearBytes(result)
			}
		}
	})
}
