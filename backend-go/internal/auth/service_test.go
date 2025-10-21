package auth_test

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
	"golang.org/x/crypto/bcrypt"

	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/internal/middleware"
)

// mockUserStore implements auth.UserStore for testing
type mockUserStore struct {
	getUserFunc          func(ctx context.Context, id string) (*auth.User, error)
	getUserByUsernameFunc func(ctx context.Context, username string) (*auth.User, error)
	getUserByEmailFunc    func(ctx context.Context, email string) (*auth.User, error)
	createUserFunc       func(ctx context.Context, user *auth.User) error
	updateUserFunc       func(ctx context.Context, user *auth.User) error
	deleteUserFunc       func(ctx context.Context, id string) error
	listUsersFunc        func(ctx context.Context, limit, offset int) ([]*auth.User, error)
}

func (m *mockUserStore) GetUser(ctx context.Context, id string) (*auth.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserStore) GetUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(ctx, username)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserStore) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserStore) CreateUser(ctx context.Context, user *auth.User) error {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, user)
	}
	return errors.New("not implemented")
}

func (m *mockUserStore) UpdateUser(ctx context.Context, user *auth.User) error {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, user)
	}
	return errors.New("not implemented")
}

func (m *mockUserStore) DeleteUser(ctx context.Context, id string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockUserStore) ListUsers(ctx context.Context, limit, offset int) ([]*auth.User, error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

// mockSessionStore implements auth.SessionStore for testing
type mockSessionStore struct {
	createSessionFunc          func(ctx context.Context, session *auth.Session) error
	getSessionFunc             func(ctx context.Context, token string) (*auth.Session, error)
	updateSessionFunc          func(ctx context.Context, session *auth.Session) error
	deleteSessionFunc          func(ctx context.Context, token string) error
	deleteUserSessionsFunc     func(ctx context.Context, userID string) error
	getUserSessionsFunc        func(ctx context.Context, userID string) ([]*auth.Session, error)
	cleanupExpiredSessionsFunc func(ctx context.Context) error
}

func (m *mockSessionStore) CreateSession(ctx context.Context, session *auth.Session) error {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, session)
	}
	return errors.New("not implemented")
}

func (m *mockSessionStore) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	if m.getSessionFunc != nil {
		return m.getSessionFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSessionStore) UpdateSession(ctx context.Context, session *auth.Session) error {
	if m.updateSessionFunc != nil {
		return m.updateSessionFunc(ctx, session)
	}
	return errors.New("not implemented")
}

func (m *mockSessionStore) DeleteSession(ctx context.Context, token string) error {
	if m.deleteSessionFunc != nil {
		return m.deleteSessionFunc(ctx, token)
	}
	return errors.New("not implemented")
}

func (m *mockSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	if m.deleteUserSessionsFunc != nil {
		return m.deleteUserSessionsFunc(ctx, userID)
	}
	return errors.New("not implemented")
}

func (m *mockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	if m.getUserSessionsFunc != nil {
		return m.getUserSessionsFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSessionStore) CleanupExpiredSessions(ctx context.Context) error {
	if m.cleanupExpiredSessionsFunc != nil {
		return m.cleanupExpiredSessionsFunc(ctx)
	}
	return errors.New("not implemented")
}

// mockLoginAttemptStore implements auth.LoginAttemptStore for testing
type mockLoginAttemptStore struct {
	recordAttemptFunc      func(ctx context.Context, attempt *auth.LoginAttempt) error
	getAttemptsFunc        func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error)
	cleanupOldAttemptsFunc func(ctx context.Context, before time.Time) error
}

func (m *mockLoginAttemptStore) RecordAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	if m.recordAttemptFunc != nil {
		return m.recordAttemptFunc(ctx, attempt)
	}
	return nil
}

func (m *mockLoginAttemptStore) GetAttempts(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
	if m.getAttemptsFunc != nil {
		return m.getAttemptsFunc(ctx, ip, username, since)
	}
	return []*auth.LoginAttempt{}, nil
}

func (m *mockLoginAttemptStore) CleanupOldAttempts(ctx context.Context, before time.Time) error {
	if m.cleanupOldAttemptsFunc != nil {
		return m.cleanupOldAttemptsFunc(ctx, before)
	}
	return nil
}

// Helper functions
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func newTestConfig() auth.Config {
	return auth.Config{
		BcryptCost:        bcrypt.MinCost,
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 24 * time.Hour,
		MaxLoginAttempts:  5,
		LockoutDuration:   15 * time.Minute,
	}
}

func newTestAuthMiddleware() *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware("test-jwt-secret", newTestLogger())
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

// TestNewService tests the service constructor
func TestNewService(t *testing.T) {
	userStore := &mockUserStore{}
	sessionStore := &mockSessionStore{}
	attemptStore := &mockLoginAttemptStore{}
	authMiddleware := newTestAuthMiddleware()
	config := newTestConfig()
	logger := newTestLogger()

	service := auth.NewService(userStore, sessionStore, attemptStore, authMiddleware, config, logger)

	assert.NotNil(t, service)
}

func TestNewService_NilStores(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	config := newTestConfig()
	logger := newTestLogger()

	tests := []struct {
		name         string
		userStore    auth.UserStore
		sessionStore auth.SessionStore
		attemptStore auth.LoginAttemptStore
	}{
		{"nil user store", nil, &mockSessionStore{}, &mockLoginAttemptStore{}},
		{"nil session store", &mockUserStore{}, nil, &mockLoginAttemptStore{}},
		{"nil attempt store", &mockUserStore{}, &mockSessionStore{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := auth.NewService(tt.userStore, tt.sessionStore, tt.attemptStore, authMiddleware, config, logger)
			assert.NotNil(t, service)
		})
	}
}

func TestNewService_NilAuthMiddleware(t *testing.T) {
	userStore := &mockUserStore{}
	sessionStore := &mockSessionStore{}
	attemptStore := &mockLoginAttemptStore{}
	config := newTestConfig()
	logger := newTestLogger()

	service := auth.NewService(userStore, sessionStore, attemptStore, nil, config, logger)

	assert.NotNil(t, service)
}

// TestLogin tests the login functionality
func TestLogin_Success(t *testing.T) {
	hashedPassword := hashPassword("password123")
	now := time.Now()

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:        "user-1",
				Username:  "testuser",
				Password:  hashedPassword,
				Active:    true,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "testuser", resp.User.Username)
	assert.Empty(t, resp.User.Password, "password should be cleared")
}

func TestLogin_InvalidUsername(t *testing.T) {
	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return nil, errors.New("user not found")
		},
	}

	sessionStore := &mockSessionStore{}
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
		recordAttemptFunc: func(ctx context.Context, attempt *auth.LoginAttempt) error {
			return nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "nonexistent",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid username or password")
}

func TestLogin_InvalidPassword(t *testing.T) {
	hashedPassword := hashPassword("correctpassword")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
	}

	sessionStore := &mockSessionStore{}
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
		recordAttemptFunc: func(ctx context.Context, attempt *auth.LoginAttempt) error {
			return nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "wrongpassword",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid username or password")
}

func TestLogin_InactiveUser(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   false,
			}, nil
		},
	}

	sessionStore := &mockSessionStore{}
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
		recordAttemptFunc: func(ctx context.Context, attempt *auth.LoginAttempt) error {
			return nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user account is disabled")
}

func TestLogin_AccountLocked(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			attempts := make([]*auth.LoginAttempt, 5)
			for i := 0; i < 5; i++ {
				attempts[i] = &auth.LoginAttempt{
					IP:        ip,
					Username:  username,
					Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
					Success:   false,
				}
			}
			return attempts, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "locked")
}

func TestLogin_SessionCreationError(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return errors.New("database error")
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create session")
}

func TestLogin_UserUpdateError(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return errors.New("update error")
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestLogin_AttemptStoreError(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return nil, errors.New("database error")
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to check account lockout")
}

func TestLogin_RememberMe(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:   "testuser",
		Password:   "password123",
		RememberMe: true,
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestLogin_PasswordComparison(t *testing.T) {
	password := "testpassword123"
	hashedPassword := hashPassword(password)

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  password,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestLogin_TokenGeneration(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Role:     "user",
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			assert.NotEmpty(t, session.Token)
			assert.NotEmpty(t, session.RefreshToken)
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestLogin_SessionFields(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			assert.NotEmpty(t, session.ID)
			assert.Equal(t, "user-1", session.UserID)
			assert.Equal(t, "127.0.0.1", session.IPAddress)
			assert.Equal(t, "test-agent", session.UserAgent)
			assert.True(t, session.Active)
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	_, err := service.Login(context.Background(), req)
	require.NoError(t, err)
}

func TestLogin_LastLoginUpdate(t *testing.T) {
	hashedPassword := hashPassword("password123")
	var lastLogin *time.Time

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			lastLogin = user.LastLogin
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	_, err := service.Login(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, lastLogin)
}

func TestLogin_SuccessfulAttemptRecorded(t *testing.T) {
	hashedPassword := hashPassword("password123")
	var recordedAttempt *auth.LoginAttempt

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
		recordAttemptFunc: func(ctx context.Context, attempt *auth.LoginAttempt) error {
			recordedAttempt = attempt
			return nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	_, err := service.Login(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, recordedAttempt)
	assert.True(t, recordedAttempt.Success)
	assert.Equal(t, "127.0.0.1", recordedAttempt.IP)
	assert.Equal(t, "testuser", recordedAttempt.Username)
}

// TestLogout tests the logout functionality
func TestLogout_Success(t *testing.T) {
	sessionStore := &mockSessionStore{
		deleteSessionFunc: func(ctx context.Context, token string) error {
			return nil
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.Logout(context.Background(), "test-token")

	assert.NoError(t, err)
}

func TestLogout_SessionNotFound(t *testing.T) {
	sessionStore := &mockSessionStore{
		deleteSessionFunc: func(ctx context.Context, token string) error {
			return errors.New("session not found")
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.Logout(context.Background(), "nonexistent-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete session")
}

func TestLogout_EmptyToken(t *testing.T) {
	sessionStore := &mockSessionStore{
		deleteSessionFunc: func(ctx context.Context, token string) error {
			assert.Empty(t, token)
			return errors.New("empty token")
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.Logout(context.Background(), "")

	assert.Error(t, err)
}

func TestLogout_DatabaseError(t *testing.T) {
	sessionStore := &mockSessionStore{
		deleteSessionFunc: func(ctx context.Context, token string) error {
			return errors.New("database connection failed")
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.Logout(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete session")
}

// TestRefreshToken tests the refresh token functionality
func TestRefreshToken_Success(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", 24*time.Hour)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Role:     "user",
				Active:   true,
			}, nil
		},
	}

	sessionStore := &mockSessionStore{
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshToken_ExpiredToken(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", -time.Hour)

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefreshToken_InactiveUser(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", 24*time.Hour)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   false,
			}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user account is disabled")
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", 24*time.Hour)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return nil, errors.New("user not found")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user not found")
}

func TestRefreshToken_SessionNotFound(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", 24*time.Hour)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   true,
			}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "session not found")
}

func TestRefreshToken_UpdateSessionError(t *testing.T) {
	authMiddleware := newTestAuthMiddleware()
	refreshToken, _ := authMiddleware.GenerateRefreshToken("user-1", 24*time.Hour)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   true,
			}, nil
		},
	}

	sessionStore := &mockSessionStore{
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return errors.New("update failed")
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, authMiddleware, newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), refreshToken)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	resp, err := service.RefreshToken(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestGetProfile tests getting user profile
func TestGetProfile_Success(t *testing.T) {
	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "hashed-password",
			}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.GetProfile(context.Background(), "user-1")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user-1", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Empty(t, user.Password, "password should be cleared")
}

func TestGetProfile_UserNotFound(t *testing.T) {
	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return nil, errors.New("user not found")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.GetProfile(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetProfile_EmptyUserID(t *testing.T) {
	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			if id == "" {
				return nil, errors.New("empty user id")
			}
			return nil, errors.New("user not found")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.GetProfile(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGetProfile_DatabaseError(t *testing.T) {
	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return nil, errors.New("database connection failed")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.GetProfile(context.Background(), "user-1")

	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestVerifyToken tests token verification
func TestVerifyToken_Success(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:         "session-1",
				UserID:     "user-1",
				Token:      token,
				ExpiresAt:  time.Now().Add(time.Hour),
				Active:     true,
				LastAccess: time.Now(),
			}, nil
		},
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   true,
				Password: "hashed",
			}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "test-token")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Empty(t, user.Password)
}

func TestVerifyToken_SessionNotFound(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return nil, errors.New("session not found")
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "session not found")
}

func TestVerifyToken_ExpiredSession(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(-time.Hour),
				Active:    true,
			}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "expired-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "expired or inactive")
}

func TestVerifyToken_InactiveSession(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(time.Hour),
				Active:    false,
			}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "inactive-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "expired or inactive")
}

func TestVerifyToken_InactiveUser(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(time.Hour),
				Active:    true,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   false,
			}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user account is disabled")
}

func TestVerifyToken_UserNotFound(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(time.Hour),
				Active:    true,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return nil, errors.New("user not found")
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "test-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestVerifyToken_UpdateSessionError(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(time.Hour),
				Active:    true,
			}, nil
		},
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return errors.New("update failed")
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   true,
				Password: "hashed",
			}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.VerifyToken(context.Background(), "test-token")

	require.NoError(t, err)
	assert.NotNil(t, user)
}

func TestVerifyToken_LastAccessUpdated(t *testing.T) {
	var lastAccess time.Time

	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:         "session-1",
				UserID:     "user-1",
				Token:      token,
				ExpiresAt:  time.Now().Add(time.Hour),
				Active:     true,
				LastAccess: time.Now().Add(-time.Hour),
			}, nil
		},
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			lastAccess = session.LastAccess
			return nil
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Active:   true,
				Password: "hashed",
			}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	_, err := service.VerifyToken(context.Background(), "test-token")
	require.NoError(t, err)
	assert.True(t, time.Since(lastAccess) < time.Second)
}

// TestCreateUser tests user creation
func TestCreateUser_Success(t *testing.T) {
	var createdUser *auth.User

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			createdUser = user
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "password123",
		Role:     "user",
	}

	err := service.CreateUser(context.Background(), user)

	require.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.NotEmpty(t, createdUser.ID)
	assert.NotEqual(t, "password123", createdUser.Password, "password should be hashed")
	assert.True(t, createdUser.Active)
	assert.NotNil(t, createdUser.Metadata)
}

func TestCreateUser_PasswordHashing(t *testing.T) {
	plainPassword := "testpassword123"
	var hashedPassword string

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			hashedPassword = user.Password
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Password: plainPassword,
	}

	err := service.CreateUser(context.Background(), user)

	require.NoError(t, err)
	assert.NotEqual(t, plainPassword, hashedPassword)
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	assert.NoError(t, err, "hashed password should match original")
}

func TestCreateUser_StoreError(t *testing.T) {
	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			return errors.New("database error")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Password: "password123",
	}

	err := service.CreateUser(context.Background(), user)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestCreateUser_MetadataInitialization(t *testing.T) {
	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			assert.NotNil(t, user.Metadata)
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Password: "password123",
	}

	err := service.CreateUser(context.Background(), user)

	require.NoError(t, err)
}

func TestCreateUser_ExistingMetadata(t *testing.T) {
	metadata := map[string]string{"key": "value"}

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			assert.Equal(t, "value", user.Metadata["key"])
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Password: "password123",
		Metadata: metadata,
	}

	err := service.CreateUser(context.Background(), user)

	require.NoError(t, err)
}

func TestCreateUser_TimestampsSet(t *testing.T) {
	var timestamps struct {
		createdAt time.Time
		updatedAt time.Time
	}

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			timestamps.createdAt = user.CreatedAt
			timestamps.updatedAt = user.UpdatedAt
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "newuser",
		Password: "password123",
	}

	err := service.CreateUser(context.Background(), user)

	require.NoError(t, err)
	assert.False(t, timestamps.createdAt.IsZero())
	assert.False(t, timestamps.updatedAt.IsZero())
}

// TestChangePassword tests password change functionality
func TestChangePassword_Success(t *testing.T) {
	oldPassword := "oldpassword"
	hashedOldPassword := hashPassword(oldPassword)
	var newHashedPassword string

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedOldPassword,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			newHashedPassword = user.Password
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", oldPassword, "newpassword")

	require.NoError(t, err)
	assert.NotEqual(t, hashedOldPassword, newHashedPassword)
	err = bcrypt.CompareHashAndPassword([]byte(newHashedPassword), []byte("newpassword"))
	assert.NoError(t, err)
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	hashedPassword := hashPassword("correctpassword")

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
			}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", "wrongpassword", "newpassword")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid current password")
}

func TestChangePassword_UserNotFound(t *testing.T) {
	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return nil, errors.New("user not found")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "nonexistent", "oldpass", "newpass")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestChangePassword_UpdateError(t *testing.T) {
	oldPassword := "oldpassword"
	hashedPassword := hashPassword(oldPassword)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Password: hashedPassword,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return errors.New("database error")
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", oldPassword, "newpassword")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update password")
}

func TestChangePassword_SamePassword(t *testing.T) {
	password := "samepassword"
	hashedPassword := hashPassword(password)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Password: hashedPassword,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", password, password)

	require.NoError(t, err)
}

func TestChangePassword_UpdatedAtSet(t *testing.T) {
	oldPassword := "oldpassword"
	hashedPassword := hashPassword(oldPassword)
	var updatedAt time.Time

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:        "user-1",
				Password:  hashedPassword,
				UpdatedAt: time.Now().Add(-time.Hour),
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			updatedAt = user.UpdatedAt
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", oldPassword, "newpassword")

	require.NoError(t, err)
	assert.True(t, time.Since(updatedAt) < time.Second)
}

func TestChangePassword_EmptyNewPassword(t *testing.T) {
	oldPassword := "oldpassword"
	hashedPassword := hashPassword(oldPassword)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Password: hashedPassword,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.ChangePassword(context.Background(), "user-1", oldPassword, "")

	require.NoError(t, err)
}

// TestAccountLockout tests account lockout functionality
func TestAccountLockout_NotLocked(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{
				{Success: false},
				{Success: false},
			}, nil
		},
	}

	config := newTestConfig()
	config.MaxLoginAttempts = 5

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "locked")
}

func TestAccountLockout_Locked(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			attempts := make([]*auth.LoginAttempt, 5)
			for i := range attempts {
				attempts[i] = &auth.LoginAttempt{Success: false}
			}
			return attempts, nil
		},
	}

	config := newTestConfig()
	config.MaxLoginAttempts = 5

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestAccountLockout_MixedAttempts(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{
				{Success: false},
				{Success: true},
				{Success: false},
				{Success: false},
			}, nil
		},
	}

	config := newTestConfig()
	config.MaxLoginAttempts = 5

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "locked")
}

func TestAccountLockout_ExactThreshold(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			attempts := make([]*auth.LoginAttempt, 3)
			for i := range attempts {
				attempts[i] = &auth.LoginAttempt{Success: false}
			}
			return attempts, nil
		},
	}

	config := newTestConfig()
	config.MaxLoginAttempts = 3

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestAccountLockout_BelowThreshold(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			attempts := make([]*auth.LoginAttempt, 2)
			for i := range attempts {
				attempts[i] = &auth.LoginAttempt{Success: false}
			}
			return attempts, nil
		},
	}

	config := newTestConfig()
	config.MaxLoginAttempts = 3

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "locked")
}

func TestAccountLockout_WindowExpired(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "locked")
}

func TestAccountLockout_ChecksIPAndUsername(t *testing.T) {
	var checkedIP, checkedUsername string

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			checkedIP = ip
			checkedUsername = username
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "192.168.1.1",
	}

	service.Login(context.Background(), req)

	assert.Equal(t, "192.168.1.1", checkedIP)
	assert.Equal(t, "testuser", checkedUsername)
}

func TestAccountLockout_TimeWindow(t *testing.T) {
	var sinceDuration time.Duration

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			sinceDuration = time.Since(since)
			return []*auth.LoginAttempt{}, nil
		},
	}

	config := newTestConfig()
	config.LockoutDuration = 30 * time.Minute

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	service.Login(context.Background(), req)

	assert.InDelta(t, 30*time.Minute, sinceDuration, float64(time.Second))
}

// TestCleanupExpiredSessions tests session cleanup
func TestCleanupExpiredSessions_Success(t *testing.T) {
	sessionStore := &mockSessionStore{
		cleanupExpiredSessionsFunc: func(ctx context.Context) error {
			return nil
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.CleanupExpiredSessions(context.Background())

	assert.NoError(t, err)
}

func TestCleanupExpiredSessions_Error(t *testing.T) {
	sessionStore := &mockSessionStore{
		cleanupExpiredSessionsFunc: func(ctx context.Context) error {
			return errors.New("cleanup failed")
		},
	}

	service := auth.NewService(&mockUserStore{}, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.CleanupExpiredSessions(context.Background())

	assert.Error(t, err)
}

// TestCleanupOldLoginAttempts tests login attempt cleanup
func TestCleanupOldLoginAttempts_Success(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		cleanupOldAttemptsFunc: func(ctx context.Context, before time.Time) error {
			return nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.CleanupOldLoginAttempts(context.Background())

	assert.NoError(t, err)
}

func TestCleanupOldLoginAttempts_Error(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		cleanupOldAttemptsFunc: func(ctx context.Context, before time.Time) error {
			return errors.New("cleanup failed")
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	err := service.CleanupOldLoginAttempts(context.Background())

	assert.Error(t, err)
}

func TestCleanupOldLoginAttempts_TimeWindow(t *testing.T) {
	var beforeTime time.Time

	attemptStore := &mockLoginAttemptStore{
		cleanupOldAttemptsFunc: func(ctx context.Context, before time.Time) error {
			beforeTime = before
			return nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	now := time.Now()
	err := service.CleanupOldLoginAttempts(context.Background())

	require.NoError(t, err)
	expectedBefore := now.Add(-24 * time.Hour)
	assert.InDelta(t, expectedBefore.Unix(), beforeTime.Unix(), 2)
}

func TestCleanupOldLoginAttempts_ContextCancellation(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		cleanupOldAttemptsFunc: func(ctx context.Context, before time.Time) error {
			return ctx.Err()
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.CleanupOldLoginAttempts(ctx)

	assert.Error(t, err)
}

// TestGenerateAPIKey tests API key generation
func TestGenerateAPIKey_Success(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	apiKey, err := service.GenerateAPIKey(context.Background(), "user-1")

	require.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.Equal(t, 64, len(apiKey), "API key should be 64 hex characters (32 bytes)")
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := service.GenerateAPIKey(context.Background(), fmt.Sprintf("user-%d", i))
		require.NoError(t, err)
		assert.False(t, keys[key], "duplicate API key generated")
		keys[key] = true
	}
}

func TestGenerateAPIKey_HexEncoding(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	apiKey, err := service.GenerateAPIKey(context.Background(), "user-1")

	require.NoError(t, err)
	for _, c := range apiKey {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "API key should only contain hex characters")
	}
}

func TestGenerateAPIKey_DifferentUsers(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	key1, err1 := service.GenerateAPIKey(context.Background(), "user-1")
	key2, err2 := service.GenerateAPIKey(context.Background(), "user-2")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, key1, key2)
}

func TestGenerateAPIKey_EmptyUserID(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	apiKey, err := service.GenerateAPIKey(context.Background(), "")

	require.NoError(t, err)
	assert.NotEmpty(t, apiKey)
}

// TestValidateAPIKey tests API key validation
func TestValidateAPIKey_NotImplemented(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.ValidateAPIKey(context.Background(), "test-api-key")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestValidateAPIKey_EmptyKey(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.ValidateAPIKey(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user, err := service.ValidateAPIKey(context.Background(), "invalid-key")

	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestConcurrency tests concurrent operations
func TestConcurrency_Login(t *testing.T) {
	hashedPassword := hashPassword("password123")

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: username,
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			req := &auth.LoginRequest{
				Username:  fmt.Sprintf("user-%d", n),
				Password:  "password123",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			}
			_, err := service.Login(context.Background(), req)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestConcurrency_CreateUser(t *testing.T) {
	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			time.Sleep(time.Millisecond)
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			user := &auth.User{
				Username: fmt.Sprintf("user-%d", n),
				Password: "password123",
			}
			err := service.CreateUser(context.Background(), user)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestConcurrency_VerifyToken(t *testing.T) {
	sessionStore := &mockSessionStore{
		getSessionFunc: func(ctx context.Context, token string) (*auth.Session, error) {
			return &auth.Session{
				ID:        "session-1",
				UserID:    "user-1",
				Token:     token,
				ExpiresAt: time.Now().Add(time.Hour),
				Active:    true,
			}, nil
		},
		updateSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       id,
				Username: "testuser",
				Active:   true,
				Password: "hashed",
			}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := service.VerifyToken(context.Background(), "test-token")
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestConcurrency_ChangePassword(t *testing.T) {
	oldPassword := "oldpassword"
	hashedPassword := hashPassword(oldPassword)

	userStore := &mockUserStore{
		getUserFunc: func(ctx context.Context, id string) (*auth.User, error) {
			return &auth.User{
				ID:       id,
				Password: hashedPassword,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			time.Sleep(time.Millisecond)
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			err := service.ChangePassword(context.Background(), fmt.Sprintf("user-%d", n), oldPassword, "newpassword")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestConcurrency_GenerateAPIKey(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	var wg sync.WaitGroup
	numGoroutines := 50
	keys := make(chan string, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key, err := service.GenerateAPIKey(context.Background(), fmt.Sprintf("user-%d", n))
			assert.NoError(t, err)
			keys <- key
		}(i)
	}

	wg.Wait()
	close(keys)

	keySet := make(map[string]bool)
	for key := range keys {
		assert.False(t, keySet[key], "duplicate key generated in concurrent test")
		keySet[key] = true
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCase_NilContext(t *testing.T) {
	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(nil, req)
	assert.Error(t, err)
}

func TestEdgeCase_EmptyLoginRequest(t *testing.T) {
	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
}

func TestEdgeCase_VeryLongUsername(t *testing.T) {
	longUsername := string(make([]byte, 10000))

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			assert.Equal(t, longUsername, username)
			return nil, errors.New("user not found")
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  longUsername,
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
}

func TestEdgeCase_VeryLongPassword(t *testing.T) {
	longPassword := string(make([]byte, 10000))

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			assert.NotEmpty(t, user.Password)
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	user := &auth.User{
		Username: "testuser",
		Password: longPassword,
	}

	err := service.CreateUser(context.Background(), user)
	require.NoError(t, err)
}

func TestEdgeCase_SpecialCharactersInUsername(t *testing.T) {
	specialUsername := "user@#$%^&*()"

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			assert.Equal(t, specialUsername, username)
			return nil, errors.New("user not found")
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  specialUsername,
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
}

func TestEdgeCase_UnicodePassword(t *testing.T) {
	unicodePassword := "密码🔐🔑"
	hashedPassword := hashPassword(unicodePassword)

	userStore := &mockUserStore{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*auth.User, error) {
			return &auth.User{
				ID:       "user-1",
				Username: "testuser",
				Password: hashedPassword,
				Active:   true,
			}, nil
		},
		updateUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	sessionStore := &mockSessionStore{
		createSessionFunc: func(ctx context.Context, session *auth.Session) error {
			return nil
		},
	}

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{}, nil
		},
	}

	service := auth.NewService(userStore, sessionStore, attemptStore, newTestAuthMiddleware(), newTestConfig(), newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  unicodePassword,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	resp, err := service.Login(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestEdgeCase_ZeroBcryptCost(t *testing.T) {
	config := newTestConfig()
	config.BcryptCost = 0

	userStore := &mockUserStore{
		createUserFunc: func(ctx context.Context, user *auth.User) error {
			return nil
		},
	}

	service := auth.NewService(userStore, &mockSessionStore{}, &mockLoginAttemptStore{}, newTestAuthMiddleware(), config, newTestLogger())

	user := &auth.User{
		Username: "testuser",
		Password: "password",
	}

	err := service.CreateUser(context.Background(), user)
	assert.Error(t, err)
}

func TestEdgeCase_NegativeMaxLoginAttempts(t *testing.T) {
	config := newTestConfig()
	config.MaxLoginAttempts = -1

	attemptStore := &mockLoginAttemptStore{
		getAttemptsFunc: func(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
			return []*auth.LoginAttempt{{Success: false}}, nil
		},
	}

	service := auth.NewService(&mockUserStore{}, &mockSessionStore{}, attemptStore, newTestAuthMiddleware(), config, newTestLogger())

	req := &auth.LoginRequest{
		Username:  "testuser",
		Password:  "password",
		IPAddress: "127.0.0.1",
	}

	_, err := service.Login(context.Background(), req)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "locked")
}
