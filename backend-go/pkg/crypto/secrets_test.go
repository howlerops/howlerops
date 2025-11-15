package crypto

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test fixtures
var (
	testConnectionID = "test-connection-123"
	testTeamID       = "test-team-456"
	testSecretType   = SecretTypeDBPassword
	testPlaintext    = []byte("my-super-secret-password")
	testUserKey      []byte
	testEncrypted    *EncryptedSecret
)

func init() {
	// Generate test user key (32 bytes for AES-256)
	var err error
	testUserKey, err = GenerateMasterKey()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate test user key: %v", err))
	}

	// Create test encrypted secret
	testEncrypted = &EncryptedSecret{
		ID:         "test-secret-id",
		OwnerID:    testConnectionID,
		Type:       testSecretType,
		Ciphertext: []byte("encrypted-data"),
		Salt:       []byte("test-salt"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// MockSecretStore implements the SecretStore interface for testing
type MockSecretStore struct {
	mock.Mock
}

func (m *MockSecretStore) StoreSecret(ctx context.Context, ownerID string, secretType SecretType, plaintext []byte, sessionKey []byte) (*EncryptedSecret, error) {
	args := m.Called(ctx, ownerID, secretType, plaintext, sessionKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EncryptedSecret), args.Error(1)
}

func (m *MockSecretStore) GetSecret(ctx context.Context, ownerID string, secretType SecretType) ([]byte, error) {
	args := m.Called(ctx, ownerID, secretType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockSecretStore) DeleteSecret(ctx context.Context, ownerID string, secretType SecretType) error {
	args := m.Called(ctx, ownerID, secretType)
	return args.Error(0)
}

func (m *MockSecretStore) ListSecrets(ctx context.Context, ownerID string) ([]*EncryptedSecret, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EncryptedSecret), args.Error(1)
}

// createTestKeyStore creates a KeyStore for testing with a pre-unlocked user key
func createTestKeyStore() (*KeyStore, []byte, error) {
	ks := NewKeyStore()
	salt, err := GenerateRandomBytes(PBKDF2SaltLength)
	if err != nil {
		return nil, nil, err
	}

	// Unlock with test passphrase
	err = ks.Unlock("test-passphrase-123", salt)
	if err != nil {
		return nil, nil, err
	}

	userKey, err := ks.GetUserKey()
	if err != nil {
		return nil, nil, err
	}

	return ks, userKey, nil
}

// createLockedKeyStore creates a locked KeyStore for testing error scenarios
func createLockedKeyStore() *KeyStore {
	return NewKeyStore()
}

// TestNewSecretManager tests the SecretManager constructor
func TestNewSecretManager(t *testing.T) {
	tests := []struct {
		name     string
		store    SecretStore
		keyStore *KeyStore
		wantNil  bool
	}{
		{
			name:     "valid inputs",
			store:    &MockSecretStore{},
			keyStore: NewKeyStore(),
			wantNil:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			keyStore: NewKeyStore(),
			wantNil:  false, // Constructor doesn't validate, SecretManager can be created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecretManager(tt.store, tt.keyStore)
			if tt.wantNil {
				assert.Nil(t, sm)
			} else {
				assert.NotNil(t, sm)
			}
		})
	}
}

// TestStoreSecret tests the StoreSecret method with various scenarios
func TestStoreSecret(t *testing.T) {
	tests := []struct {
		name          string
		connectionID  string
		secretType    SecretType
		plaintext     []byte
		useLocked     bool
		setupMocks    func(*MockSecretStore, []byte)
		expectedError bool
		errorContains string
	}{
		{
			name:         "success",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			plaintext:    testPlaintext,
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, testSecretType, testPlaintext, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
		{
			name:          "keystore locked error",
			connectionID:  testConnectionID,
			secretType:    testSecretType,
			plaintext:     testPlaintext,
			useLocked:     true,
			setupMocks:    nil,
			expectedError: true,
			errorContains: "failed to get user key",
		},
		{
			name:         "store error",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			plaintext:    testPlaintext,
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, testSecretType, testPlaintext, userKey).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to store secret",
		},
		{
			name:         "empty plaintext",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			plaintext:    []byte{},
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, testSecretType, []byte{}, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
		{
			name:         "large plaintext",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			plaintext:    make([]byte, 10000), // 10KB plaintext
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, testSecretType, mock.Anything, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
		{
			name:         "empty connection ID",
			connectionID: "",
			secretType:   testSecretType,
			plaintext:    testPlaintext,
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, "", testSecretType, testPlaintext, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false, // SecretManager doesn't validate connectionID
		},
		{
			name:         "SSH password type",
			connectionID: testConnectionID,
			secretType:   SecretTypeSSHPassword,
			plaintext:    testPlaintext,
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, SecretTypeSSHPassword, testPlaintext, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
		{
			name:         "API key type",
			connectionID: testConnectionID,
			secretType:   SecretTypeAPIKey,
			plaintext:    testPlaintext,
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, SecretTypeAPIKey, testPlaintext, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
		{
			name:         "SSH private key type",
			connectionID: testConnectionID,
			secretType:   SecretTypeSSHPrivateKey,
			plaintext:    []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"),
			useLocked:    false,
			setupMocks: func(store *MockSecretStore, userKey []byte) {
				store.On("StoreSecret", mock.Anything, testConnectionID, SecretTypeSSHPrivateKey, mock.Anything, userKey).
					Return(testEncrypted, nil)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSecretStore)

			var keyStore *KeyStore
			var userKey []byte
			var err error

			if tt.useLocked {
				keyStore = createLockedKeyStore()
			} else {
				keyStore, userKey, err = createTestKeyStore()
				require.NoError(t, err)
				if tt.setupMocks != nil {
					tt.setupMocks(mockStore, userKey)
				}
			}

			sm := NewSecretManager(mockStore, keyStore)
			err = sm.StoreSecret(context.Background(), tt.connectionID, tt.secretType, tt.plaintext)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			if !tt.useLocked && tt.setupMocks != nil {
				mockStore.AssertExpectations(t)
			}
		})
	}
}

// TestGetSecret tests the GetSecret method with various scenarios
func TestGetSecret(t *testing.T) {
	tests := []struct {
		name          string
		connectionID  string
		secretType    SecretType
		setupMocks    func(*MockSecretStore)
		expectedData  []byte
		expectedError bool
		errorContains string
	}{
		{
			name:         "success",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
					Return(testPlaintext, nil)
			},
			expectedData:  testPlaintext,
			expectedError: false,
		},
		{
			name:         "store error",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get secret from store",
		},
		{
			name:         "secret not found",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
					Return(nil, errors.New("not found"))
			},
			expectedError: true,
			errorContains: "failed to get secret from store",
		},
		{
			name:         "empty plaintext",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
					Return([]byte{}, nil)
			},
			expectedData:  []byte{},
			expectedError: false,
		},
		{
			name:         "empty connection ID",
			connectionID: "",
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, "", testSecretType).
					Return(testPlaintext, nil)
			},
			expectedData:  testPlaintext,
			expectedError: false,
		},
		{
			name:         "different secret type",
			connectionID: testConnectionID,
			secretType:   SecretTypeAPIKey,
			setupMocks: func(store *MockSecretStore) {
				store.On("GetSecret", mock.Anything, testConnectionID, SecretTypeAPIKey).
					Return([]byte("api-key-value"), nil)
			},
			expectedData:  []byte("api-key-value"),
			expectedError: false,
		},
		{
			name:         "large plaintext",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				largeData := make([]byte, 10000)
				store.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
					Return(largeData, nil)
			},
			expectedData:  make([]byte, 10000),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSecretStore)
			tt.setupMocks(mockStore)

			sm := NewSecretManager(mockStore, NewKeyStore())
			data, err := sm.GetSecret(context.Background(), tt.connectionID, tt.secretType)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedData, data)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// TestDeleteSecret tests the DeleteSecret method
func TestDeleteSecret(t *testing.T) {
	tests := []struct {
		name          string
		connectionID  string
		secretType    SecretType
		setupMocks    func(*MockSecretStore)
		expectedError bool
		errorContains string
	}{
		{
			name:         "success",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("DeleteSecret", mock.Anything, testConnectionID, testSecretType).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name:         "store error",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("DeleteSecret", mock.Anything, testConnectionID, testSecretType).
					Return(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "database error",
		},
		{
			name:         "secret not found",
			connectionID: testConnectionID,
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("DeleteSecret", mock.Anything, testConnectionID, testSecretType).
					Return(errors.New("not found"))
			},
			expectedError: true,
			errorContains: "not found",
		},
		{
			name:         "empty connection ID",
			connectionID: "",
			secretType:   testSecretType,
			setupMocks: func(store *MockSecretStore) {
				store.On("DeleteSecret", mock.Anything, "", testSecretType).
					Return(nil)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSecretStore)
			tt.setupMocks(mockStore)

			sm := NewSecretManager(mockStore, NewKeyStore())
			err := sm.DeleteSecret(context.Background(), tt.connectionID, tt.secretType)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// TestListSecrets tests the ListSecrets method
func TestListSecrets(t *testing.T) {
	tests := []struct {
		name          string
		connectionID  string
		setupMocks    func(*MockSecretStore)
		expectedCount int
		expectedError bool
		errorContains string
	}{
		{
			name:         "success with multiple secrets",
			connectionID: testConnectionID,
			setupMocks: func(store *MockSecretStore) {
				secrets := []*EncryptedSecret{
					{ID: "1", OwnerID: testConnectionID, Type: SecretTypeDBPassword},
					{ID: "2", OwnerID: testConnectionID, Type: SecretTypeSSHPassword},
					{ID: "3", OwnerID: testConnectionID, Type: SecretTypeAPIKey},
				}
				store.On("ListSecrets", mock.Anything, testConnectionID).
					Return(secrets, nil)
			},
			expectedCount: 3,
			expectedError: false,
		},
		{
			name:         "success with no secrets",
			connectionID: testConnectionID,
			setupMocks: func(store *MockSecretStore) {
				store.On("ListSecrets", mock.Anything, testConnectionID).
					Return([]*EncryptedSecret{}, nil)
			},
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:         "store error",
			connectionID: testConnectionID,
			setupMocks: func(store *MockSecretStore) {
				store.On("ListSecrets", mock.Anything, testConnectionID).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "database error",
		},
		{
			name:         "empty connection ID",
			connectionID: "",
			setupMocks: func(store *MockSecretStore) {
				store.On("ListSecrets", mock.Anything, "").
					Return([]*EncryptedSecret{}, nil)
			},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSecretStore)
			tt.setupMocks(mockStore)

			sm := NewSecretManager(mockStore, NewKeyStore())
			secrets, err := sm.ListSecrets(context.Background(), tt.connectionID)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(secrets))
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// TestStoreTeamSecret tests the StoreTeamSecret method (currently delegates to StoreSecret)
func TestStoreTeamSecret(t *testing.T) {
	mockStore := new(MockSecretStore)

	keyStore, userKey, err := createTestKeyStore()
	require.NoError(t, err)

	// Setup mocks - StoreTeamSecret calls StoreSecret internally
	mockStore.On("StoreSecret", mock.Anything, testConnectionID, testSecretType, testPlaintext, userKey).
		Return(testEncrypted, nil)

	sm := NewSecretManager(mockStore, keyStore)
	err = sm.StoreTeamSecret(context.Background(), testConnectionID, testSecretType, testPlaintext, testTeamID)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

// TestGetTeamSecret tests the GetTeamSecret method (currently delegates to GetSecret)
func TestGetTeamSecret(t *testing.T) {
	mockStore := new(MockSecretStore)

	// Setup mocks - GetTeamSecret calls GetSecret internally
	mockStore.On("GetSecret", mock.Anything, testConnectionID, testSecretType).
		Return(testPlaintext, nil)

	sm := NewSecretManager(mockStore, NewKeyStore())
	data, err := sm.GetTeamSecret(context.Background(), testConnectionID, testSecretType, testTeamID)

	require.NoError(t, err)
	assert.Equal(t, testPlaintext, data)
	mockStore.AssertExpectations(t)
}

// TestReencryptAllSecrets tests that ReencryptAllSecrets returns not implemented error
func TestReencryptAllSecrets(t *testing.T) {
	mockStore := new(MockSecretStore)
	sm := NewSecretManager(mockStore, NewKeyStore())

	oldKey, err := GenerateMasterKey()
	require.NoError(t, err)

	newKey, err := GenerateMasterKey()
	require.NoError(t, err)

	err = sm.ReencryptAllSecrets(context.Background(), oldKey, newKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented yet")
}

// TestSecretTypes tests that all secret type constants are defined
func TestSecretTypes(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		expected   string
	}{
		{"DB password", SecretTypeDBPassword, "db_password"},
		{"SSH password", SecretTypeSSHPassword, "ssh_password"},
		{"SSH private key", SecretTypeSSHPrivateKey, "ssh_private_key"},
		{"API key", SecretTypeAPIKey, "api_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, SecretType(tt.expected), tt.secretType)
		})
	}
}

// TestEncryptedSecret_Structure tests the EncryptedSecret struct
func TestEncryptedSecret_Structure(t *testing.T) {
	secret := &EncryptedSecret{
		ID:         "test-id",
		OwnerID:    "owner-123",
		Type:       SecretTypeDBPassword,
		Ciphertext: []byte("encrypted"),
		Salt:       []byte("salt"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	assert.NotEmpty(t, secret.ID)
	assert.NotEmpty(t, secret.OwnerID)
	assert.NotEmpty(t, secret.Type)
	assert.NotEmpty(t, secret.Ciphertext)
	assert.NotEmpty(t, secret.Salt)
	assert.False(t, secret.CreatedAt.IsZero())
	assert.False(t, secret.UpdatedAt.IsZero())
}

// TestStoreSecret_ContextPropagation tests that context is properly propagated
func TestStoreSecret_ContextPropagation(t *testing.T) {
	mockStore := new(MockSecretStore)

	keyStore, userKey, err := createTestKeyStore()
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	mockStore.On("StoreSecret", ctx, testConnectionID, testSecretType, testPlaintext, userKey).
		Return(testEncrypted, nil)

	sm := NewSecretManager(mockStore, keyStore)
	err = sm.StoreSecret(ctx, testConnectionID, testSecretType, testPlaintext)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

// TestGetSecret_ContextPropagation tests that context is properly propagated
func TestGetSecret_ContextPropagation(t *testing.T) {
	mockStore := new(MockSecretStore)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	mockStore.On("GetSecret", ctx, testConnectionID, testSecretType).
		Return(testPlaintext, nil)

	sm := NewSecretManager(mockStore, NewKeyStore())
	data, err := sm.GetSecret(ctx, testConnectionID, testSecretType)

	require.NoError(t, err)
	assert.Equal(t, testPlaintext, data)
	mockStore.AssertExpectations(t)
}
