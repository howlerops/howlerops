package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockTwoFactorStore is a mock implementation of TwoFactorStore
type MockTwoFactorStore struct {
	mock.Mock
}

func (m *MockTwoFactorStore) CreateTwoFactor(ctx context.Context, tf *TwoFactor) error {
	args := m.Called(ctx, tf)
	return args.Error(0)
}

func (m *MockTwoFactorStore) GetTwoFactor(ctx context.Context, userID string) (*TwoFactor, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TwoFactor), args.Error(1)
}

func (m *MockTwoFactorStore) EnableTwoFactor(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTwoFactorStore) DisableTwoFactor(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTwoFactorStore) UpdateBackupCodes(ctx context.Context, userID string, codes []string) error {
	args := m.Called(ctx, userID, codes)
	return args.Error(0)
}

func (m *MockTwoFactorStore) UseBackupCode(ctx context.Context, userID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

// MockSecurityEventLogger is a mock implementation
type MockSecurityEventLogger struct {
	mock.Mock
}

func (m *MockSecurityEventLogger) LogSecurityEvent(ctx context.Context, eventType, userID, orgID, ipAddress, userAgent string, details map[string]interface{}) error {
	args := m.Called(ctx, eventType, userID, orgID, ipAddress, userAgent, details)
	return args.Error(0)
}

func TestTwoFactorService_EnableTwoFactor(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockTwoFactorStore)
	mockLogger := new(MockSecurityEventLogger)
	service := NewTwoFactorService(mockStore, mockLogger, logger, "TestIssuer")

	ctx := context.Background()
	userID := "user123"
	userEmail := "test@example.com"

	t.Run("successful setup", func(t *testing.T) {
		mockStore.On("GetTwoFactor", ctx, userID).Return(nil, nil).Once()
		mockStore.On("CreateTwoFactor", ctx, mock.AnythingOfType("*auth.TwoFactor")).Return(nil).Once()
		mockLogger.On("LogSecurityEvent", ctx, "2fa_setup_initiated", userID, "", "", "", mock.Anything).Return(nil).Once()

		setup, err := service.EnableTwoFactor(ctx, userID, userEmail)
		assert.NoError(t, err)
		assert.NotNil(t, setup)
		assert.NotEmpty(t, setup.Secret)
		assert.NotEmpty(t, setup.QRCode)
		assert.Len(t, setup.BackupCodes, 10)

		mockStore.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("already enabled", func(t *testing.T) {
		existing := &TwoFactor{
			UserID:  userID,
			Enabled: true,
		}
		mockStore.On("GetTwoFactor", ctx, userID).Return(existing, nil).Once()

		_, err := service.EnableTwoFactor(ctx, userID, userEmail)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already enabled")

		mockStore.AssertExpectations(t)
	})
}

func TestTwoFactorService_ConfirmTwoFactor(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockTwoFactorStore)
	mockLogger := new(MockSecurityEventLogger)
	service := NewTwoFactorService(mockStore, mockLogger, logger, "TestIssuer")

	ctx := context.Background()
	userID := "user123"

	// Generate a real TOTP secret for testing
	key, _ := totp.Generate(totp.GenerateOpts{
		Issuer:      "TestIssuer",
		AccountName: "test@example.com",
	})

	t.Run("valid code", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: false,
			Secret:  key.Secret(),
		}

		validCode, _ := totp.GenerateCode(key.Secret(), time.Now())

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()
		mockStore.On("EnableTwoFactor", ctx, userID).Return(nil).Once()
		mockLogger.On("LogSecurityEvent", ctx, "2fa_enabled", userID, "", "", "", mock.Anything).Return(nil).Once()

		err := service.ConfirmTwoFactor(ctx, userID, validCode)
		assert.NoError(t, err)

		mockStore.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("invalid code", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: false,
			Secret:  key.Secret(),
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()

		err := service.ConfirmTwoFactor(ctx, userID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid verification code")

		mockStore.AssertExpectations(t)
	})

	t.Run("already enabled", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: true,
			Secret:  key.Secret(),
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()

		err := service.ConfirmTwoFactor(ctx, userID, "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already enabled")

		mockStore.AssertExpectations(t)
	})
}

func TestTwoFactorService_ValidateCode(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockTwoFactorStore)
	mockLogger := new(MockSecurityEventLogger)
	service := NewTwoFactorService(mockStore, mockLogger, logger, "TestIssuer")

	ctx := context.Background()
	userID := "user123"

	// Generate a real TOTP secret for testing
	key, _ := totp.Generate(totp.GenerateOpts{
		Issuer:      "TestIssuer",
		AccountName: "test@example.com",
	})

	t.Run("valid TOTP code", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: true,
			Secret:  key.Secret(),
		}

		validCode, _ := totp.GenerateCode(key.Secret(), time.Now())

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()

		err := service.ValidateCode(ctx, userID, validCode)
		assert.NoError(t, err)

		mockStore.AssertExpectations(t)
	})

	t.Run("valid backup code", func(t *testing.T) {
		backupCode := "TESTCODE"
		hashedCode, _ := bcrypt.GenerateFromPassword([]byte(strings.ToUpper(backupCode)), bcrypt.DefaultCost)

		tf := &TwoFactor{
			UserID:      userID,
			Enabled:     true,
			Secret:      key.Secret(),
			BackupCodes: []string{string(hashedCode)},
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()
		mockStore.On("UseBackupCode", ctx, userID, backupCode).Return(nil).Once()
		mockLogger.On("LogSecurityEvent", ctx, "2fa_backup_code_used", userID, "", "", "", mock.Anything).Return(nil).Once()

		err := service.ValidateCode(ctx, userID, backupCode)
		assert.NoError(t, err)

		mockStore.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("invalid code", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:      userID,
			Enabled:     true,
			Secret:      key.Secret(),
			BackupCodes: []string{},
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()
		mockLogger.On("LogSecurityEvent", ctx, "2fa_validation_failed", userID, "", "", "", mock.Anything).Return(nil).Once()

		err := service.ValidateCode(ctx, userID, "999999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid verification code")

		mockStore.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("2FA not enabled", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: false,
			Secret:  key.Secret(),
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()

		err := service.ValidateCode(ctx, userID, "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2FA not enabled")

		mockStore.AssertExpectations(t)
	})
}

func TestTwoFactorService_BackupCodes(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockTwoFactorStore)
	mockLogger := new(MockSecurityEventLogger)
	service := NewTwoFactorService(mockStore, mockLogger, logger, "TestIssuer")

	t.Run("generate backup codes", func(t *testing.T) {
		codes := service.generateBackupCodes(10)
		assert.Len(t, codes, 10)

		// Check that all codes are unique
		codeMap := make(map[string]bool)
		for _, code := range codes {
			assert.Len(t, code, 8)
			assert.False(t, codeMap[code], "Duplicate code found")
			codeMap[code] = true
		}
	})

	t.Run("hash backup codes", func(t *testing.T) {
		codes := []string{"CODE1234", "CODE5678"}
		hashed := service.hashBackupCodes(codes)
		assert.Len(t, hashed, 2)

		// Verify hashes are valid bcrypt
		for i, hash := range hashed {
			err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(codes[i]))
			assert.NoError(t, err)
		}
	})

	t.Run("validate backup code", func(t *testing.T) {
		code := "TESTCODE"
		hashedCode, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		hashedCodes := []string{string(hashedCode)}

		valid := service.isValidBackupCode(code, hashedCodes)
		assert.True(t, valid)

		invalid := service.isValidBackupCode("WRONGCODE", hashedCodes)
		assert.False(t, invalid)
	})
}

func TestTwoFactorService_RegenerateBackupCodes(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockTwoFactorStore)
	mockLogger := new(MockSecurityEventLogger)
	service := NewTwoFactorService(mockStore, mockLogger, logger, "TestIssuer")

	ctx := context.Background()
	userID := "user123"

	t.Run("successful regeneration", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: true,
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()
		mockStore.On("UpdateBackupCodes", ctx, userID, mock.AnythingOfType("[]string")).Return(nil).Once()
		mockLogger.On("LogSecurityEvent", ctx, "2fa_backup_codes_regenerated", userID, "", "", "", mock.Anything).Return(nil).Once()

		codes, err := service.RegenerateBackupCodes(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, codes, 10)

		mockStore.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("2FA not enabled", func(t *testing.T) {
		tf := &TwoFactor{
			UserID:  userID,
			Enabled: false,
		}

		mockStore.On("GetTwoFactor", ctx, userID).Return(tf, nil).Once()

		_, err := service.RegenerateBackupCodes(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2FA not enabled")

		mockStore.AssertExpectations(t)
	})
}