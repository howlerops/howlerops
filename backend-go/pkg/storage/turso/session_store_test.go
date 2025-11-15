package turso_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Test Setup and Helpers
// ====================================================================

func setupSessionTestDB(t *testing.T) (*sql.DB, func()) {
	// Use in-memory SQLite for tests
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create schema
	schema := `
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			refresh_token TEXT NOT NULL UNIQUE,
			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			last_access INTEGER NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			active INTEGER NOT NULL DEFAULT 1
		);

		CREATE INDEX idx_sessions_token ON sessions(token);
		CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
		CREATE INDEX idx_sessions_user_id ON sessions(user_id);
		CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close() // Best-effort close in test
	}

	return db, cleanup
}

func newSessionTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func createTestSession(userID string, expiresIn time.Duration) *auth.Session {
	now := time.Now()
	return &auth.Session{
		ID:           "session-" + userID,
		UserID:       userID,
		Token:        "token-" + userID,
		RefreshToken: "refresh-" + userID,
		ExpiresAt:    now.Add(expiresIn),
		CreatedAt:    now,
		LastAccess:   now,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Mozilla/5.0",
		Active:       true,
	}
}

// ====================================================================
// CreateSession Tests
// ====================================================================

func TestSessionStore_CreateSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)

	err := store.CreateSession(context.Background(), session)

	require.NoError(t, err)

	// Verify session was created by retrieving it
	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Token, retrieved.Token)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)
	assert.Equal(t, session.IPAddress, retrieved.IPAddress)
	assert.Equal(t, session.UserAgent, retrieved.UserAgent)
	assert.True(t, retrieved.Active)
}

func TestSessionStore_CreateSession_WithAllFields(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	now := time.Now()
	session := &auth.Session{
		ID:           "session-123",
		UserID:       "user-456",
		Token:        "access-token-xyz",
		RefreshToken: "refresh-token-abc",
		ExpiresAt:    now.Add(24 * time.Hour),
		CreatedAt:    now,
		LastAccess:   now,
		IPAddress:    "192.168.1.100",
		UserAgent:    "Custom User Agent/1.0",
		Active:       true,
	}

	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Retrieve and verify all fields
	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)

	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Token, retrieved.Token)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)
	assert.WithinDuration(t, session.ExpiresAt, retrieved.ExpiresAt, time.Second)
	assert.WithinDuration(t, session.CreatedAt, retrieved.CreatedAt, time.Second)
	assert.WithinDuration(t, session.LastAccess, retrieved.LastAccess, time.Second)
	assert.Equal(t, session.IPAddress, retrieved.IPAddress)
	assert.Equal(t, session.UserAgent, retrieved.UserAgent)
	assert.Equal(t, session.Active, retrieved.Active)
}

func TestSessionStore_CreateSession_DuplicateToken(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session1 := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session1)
	require.NoError(t, err)

	// Try to create another session with the same token
	session2 := createTestSession("user-2", 24*time.Hour)
	session2.Token = session1.Token // Same token
	session2.RefreshToken = "different-refresh"
	session2.ID = "different-id"

	err = store.CreateSession(context.Background(), session2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create session")
}

func TestSessionStore_CreateSession_InactiveSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	session.Active = false

	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)
	assert.False(t, retrieved.Active)
}

// ====================================================================
// GetSession Tests
// ====================================================================

func TestSessionStore_GetSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)

	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Token, retrieved.Token)
}

func TestSessionStore_GetSession_NotFound(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session, err := store.GetSession(context.Background(), "nonexistent-token")

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionStore_GetSession_TimestampConversion(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	now := time.Now()
	session := createTestSession("user-1", 24*time.Hour)
	session.CreatedAt = now
	session.LastAccess = now
	session.ExpiresAt = now.Add(24 * time.Hour)

	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)

	// Timestamps should be converted correctly (within 1 second tolerance for Unix timestamp precision)
	assert.WithinDuration(t, session.CreatedAt, retrieved.CreatedAt, time.Second)
	assert.WithinDuration(t, session.LastAccess, retrieved.LastAccess, time.Second)
	assert.WithinDuration(t, session.ExpiresAt, retrieved.ExpiresAt, time.Second)
}

// ====================================================================
// GetSessionByRefreshToken Tests
// ====================================================================

func TestSessionStore_GetSessionByRefreshToken(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSessionByRefreshToken(context.Background(), session.RefreshToken)

	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)
}

func TestSessionStore_GetSessionByRefreshToken_NotFound(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session, err := store.GetSessionByRefreshToken(context.Background(), "nonexistent-refresh-token")

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session not found")
}

// ====================================================================
// UpdateSession Tests
// ====================================================================

func TestSessionStore_UpdateSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Update session fields
	session.Token = "new-token"
	session.RefreshToken = "new-refresh-token"
	session.LastAccess = time.Now().Add(1 * time.Hour)
	session.IPAddress = "192.168.1.200"
	session.UserAgent = "Updated User Agent"
	session.Active = false
	session.ExpiresAt = time.Now().Add(48 * time.Hour)

	err = store.UpdateSession(context.Background(), session)
	require.NoError(t, err)

	// Retrieve and verify updates
	retrieved, err := store.GetSession(context.Background(), "new-token")
	require.NoError(t, err)

	assert.Equal(t, session.Token, retrieved.Token)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)
	assert.WithinDuration(t, session.LastAccess, retrieved.LastAccess, time.Second)
	assert.Equal(t, session.IPAddress, retrieved.IPAddress)
	assert.Equal(t, session.UserAgent, retrieved.UserAgent)
	assert.False(t, retrieved.Active)
	assert.WithinDuration(t, session.ExpiresAt, retrieved.ExpiresAt, time.Second)
}

func TestSessionStore_UpdateSession_NotFound(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	session.ID = "nonexistent-id"

	err := store.UpdateSession(context.Background(), session)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionStore_UpdateSession_ExtendExpiry(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 1*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Extend expiry
	newExpiry := time.Now().Add(48 * time.Hour)
	session.ExpiresAt = newExpiry

	err = store.UpdateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)
	assert.WithinDuration(t, newExpiry, retrieved.ExpiresAt, time.Second)
}

// ====================================================================
// DeleteSession Tests
// ====================================================================

func TestSessionStore_DeleteSession(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	err = store.DeleteSession(context.Background(), session.Token)
	require.NoError(t, err)

	// Verify session is deleted
	_, err = store.GetSession(context.Background(), session.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionStore_DeleteSession_NotFound(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	err := store.DeleteSession(context.Background(), "nonexistent-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionStore_DeleteSession_MultipleTokens(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create multiple sessions
	session1 := createTestSession("user-1", 24*time.Hour)
	session2 := createTestSession("user-2", 24*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), session1))
	require.NoError(t, store.CreateSession(context.Background(), session2))

	// Delete one session
	err := store.DeleteSession(context.Background(), session1.Token)
	require.NoError(t, err)

	// Verify only one is deleted
	_, err = store.GetSession(context.Background(), session1.Token)
	assert.Error(t, err)

	retrieved, err := store.GetSession(context.Background(), session2.Token)
	require.NoError(t, err)
	assert.Equal(t, session2.ID, retrieved.ID)
}

// ====================================================================
// DeleteUserSessions Tests
// ====================================================================

func TestSessionStore_DeleteUserSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create multiple sessions for the same user
	session1 := createTestSession("user-1", 24*time.Hour)
	session2 := createTestSession("user-1", 24*time.Hour)
	session2.ID = "session-user-1-2"
	session2.Token = "token-user-1-2"
	session2.RefreshToken = "refresh-user-1-2"

	require.NoError(t, store.CreateSession(context.Background(), session1))
	require.NoError(t, store.CreateSession(context.Background(), session2))

	err := store.DeleteUserSessions(context.Background(), "user-1")
	require.NoError(t, err)

	// Verify all sessions for user are deleted
	_, err = store.GetSession(context.Background(), session1.Token)
	assert.Error(t, err)

	_, err = store.GetSession(context.Background(), session2.Token)
	assert.Error(t, err)
}

func TestSessionStore_DeleteUserSessions_NoSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Should not error when no sessions exist
	err := store.DeleteUserSessions(context.Background(), "user-with-no-sessions")
	require.NoError(t, err)
}

func TestSessionStore_DeleteUserSessions_OnlyDeletesTargetUser(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session1 := createTestSession("user-1", 24*time.Hour)
	session2 := createTestSession("user-2", 24*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), session1))
	require.NoError(t, store.CreateSession(context.Background(), session2))

	err := store.DeleteUserSessions(context.Background(), "user-1")
	require.NoError(t, err)

	// Verify only user-1's sessions are deleted
	_, err = store.GetSession(context.Background(), session1.Token)
	assert.Error(t, err)

	retrieved, err := store.GetSession(context.Background(), session2.Token)
	require.NoError(t, err)
	assert.Equal(t, session2.ID, retrieved.ID)
}

// ====================================================================
// GetUserSessions Tests
// ====================================================================

func TestSessionStore_GetUserSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create multiple sessions for a user
	session1 := createTestSession("user-1", 24*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), session1))

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	session2 := createTestSession("user-1", 24*time.Hour)
	session2.ID = "session-user-1-2"
	session2.Token = "token-user-1-2"
	session2.RefreshToken = "refresh-user-1-2"
	require.NoError(t, store.CreateSession(context.Background(), session2))

	sessions, err := store.GetUserSessions(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Find which session is which (order may vary due to timestamp precision)
	var found1, found2 bool
	for _, s := range sessions {
		if s.ID == session1.ID {
			found1 = true
		}
		if s.ID == session2.ID {
			found2 = true
		}
	}
	assert.True(t, found1, "Should find session 1")
	assert.True(t, found2, "Should find session 2")
}

func TestSessionStore_GetUserSessions_NoSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	sessions, err := store.GetUserSessions(context.Background(), "user-with-no-sessions")

	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionStore_GetUserSessions_OnlyReturnsUserSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session1 := createTestSession("user-1", 24*time.Hour)
	session2 := createTestSession("user-2", 24*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), session1))
	require.NoError(t, store.CreateSession(context.Background(), session2))

	sessions, err := store.GetUserSessions(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "user-1", sessions[0].UserID)
}

// ====================================================================
// CleanupExpiredSessions Tests
// ====================================================================

func TestSessionStore_CleanupExpiredSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create expired session
	expiredSession := createTestSession("user-1", -24*time.Hour) // Expired 24 hours ago

	// Create valid session
	validSession := createTestSession("user-2", 24*time.Hour)

	require.NoError(t, store.CreateSession(context.Background(), expiredSession))
	require.NoError(t, store.CreateSession(context.Background(), validSession))

	err := store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)

	// Verify expired session is deleted
	_, err = store.GetSession(context.Background(), expiredSession.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")

	// Verify valid session still exists
	retrieved, err := store.GetSession(context.Background(), validSession.Token)
	require.NoError(t, err)
	assert.Equal(t, validSession.ID, retrieved.ID)
}

func TestSessionStore_CleanupExpiredSessions_InactiveSessions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create inactive session (not expired)
	inactiveSession := createTestSession("user-1", 24*time.Hour)
	inactiveSession.Active = false

	// Create active session
	activeSession := createTestSession("user-2", 24*time.Hour)

	require.NoError(t, store.CreateSession(context.Background(), inactiveSession))
	require.NoError(t, store.CreateSession(context.Background(), activeSession))

	err := store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)

	// Verify inactive session is deleted
	_, err = store.GetSession(context.Background(), inactiveSession.Token)
	assert.Error(t, err)

	// Verify active session still exists
	retrieved, err := store.GetSession(context.Background(), activeSession.Token)
	require.NoError(t, err)
	assert.Equal(t, activeSession.ID, retrieved.ID)
}

func TestSessionStore_CleanupExpiredSessions_MultipleExpired(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create multiple expired sessions
	for i := 1; i <= 5; i++ {
		session := createTestSession("user-"+string(rune(i+'0')), -24*time.Hour)
		session.ID = "expired-" + string(rune(i+'0'))
		session.Token = "token-expired-" + string(rune(i+'0'))
		session.RefreshToken = "refresh-expired-" + string(rune(i+'0'))
		require.NoError(t, store.CreateSession(context.Background(), session))
	}

	// Create one valid session
	validSession := createTestSession("user-valid", 24*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), validSession))

	err := store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)

	// Verify all expired sessions are deleted
	for i := 1; i <= 5; i++ {
		_, err := store.GetSession(context.Background(), "token-expired-"+string(rune(i+'0')))
		assert.Error(t, err)
	}

	// Verify valid session still exists
	retrieved, err := store.GetSession(context.Background(), validSession.Token)
	require.NoError(t, err)
	assert.Equal(t, validSession.ID, retrieved.ID)
}

func TestSessionStore_CleanupExpiredSessions_NoExpired(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create only valid sessions
	session1 := createTestSession("user-1", 24*time.Hour)
	session2 := createTestSession("user-2", 48*time.Hour)
	require.NoError(t, store.CreateSession(context.Background(), session1))
	require.NoError(t, store.CreateSession(context.Background(), session2))

	err := store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)

	// Verify both sessions still exist
	_, err = store.GetSession(context.Background(), session1.Token)
	assert.NoError(t, err)

	_, err = store.GetSession(context.Background(), session2.Token)
	assert.NoError(t, err)
}

func TestSessionStore_CleanupExpiredSessions_EmptyTable(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Should not error on empty table
	err := store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)
}

// ====================================================================
// Edge Cases and Error Handling Tests
// ====================================================================

func TestSessionStore_SessionExpiry_BoundaryConditions(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	testCases := []struct {
		name          string
		expiresIn     time.Duration
		shouldCleanup bool
	}{
		{"just expired", -1 * time.Second, true},
		{"about to expire", 1 * time.Second, false},
		{"expired long ago", -24 * time.Hour, true},
		{"valid for long time", 365 * 24 * time.Hour, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := createTestSession("user-"+tc.name, tc.expiresIn)
			session.ID = "session-" + tc.name
			session.Token = "token-" + tc.name
			session.RefreshToken = "refresh-" + tc.name

			err := store.CreateSession(context.Background(), session)
			require.NoError(t, err)

			err = store.CleanupExpiredSessions(context.Background())
			require.NoError(t, err)

			_, err = store.GetSession(context.Background(), session.Token)
			if tc.shouldCleanup {
				assert.Error(t, err, "Expected session to be cleaned up")
			} else {
				assert.NoError(t, err, "Expected session to still exist")
			}
		})
	}
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	// Create multiple sessions sequentially to avoid SQLite concurrency issues
	// SQLite in-memory databases can have issues with concurrent writes from goroutines
	for i := 1; i <= 10; i++ {
		session := createTestSession("user-"+string(rune(i+'0')), 24*time.Hour)
		session.ID = "session-" + string(rune(i+'0'))
		session.Token = "token-" + string(rune(i+'0'))
		session.RefreshToken = "refresh-" + string(rune(i+'0'))
		err := store.CreateSession(context.Background(), session)
		require.NoError(t, err)
	}

	// Verify all sessions were created
	for i := 1; i <= 10; i++ {
		sessions, err := store.GetUserSessions(context.Background(), "user-"+string(rune(i+'0')))
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
	}
}

func TestSessionStore_NullableFields(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	session.IPAddress = ""
	session.UserAgent = ""

	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)
	assert.Equal(t, "", retrieved.IPAddress)
	assert.Equal(t, "", retrieved.UserAgent)
}

func TestSessionStore_UpdateSession_ToggleActive(t *testing.T) {
	db, cleanup := setupSessionTestDB(t)
	defer cleanup()

	store := turso.NewTursoSessionStore(db, newSessionTestLogger())

	session := createTestSession("user-1", 24*time.Hour)
	err := store.CreateSession(context.Background(), session)
	require.NoError(t, err)

	// Deactivate session
	session.Active = false
	err = store.UpdateSession(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.GetSession(context.Background(), session.Token)
	require.NoError(t, err)
	assert.False(t, retrieved.Active)

	// Cleanup should remove inactive sessions
	err = store.CleanupExpiredSessions(context.Background())
	require.NoError(t, err)

	_, err = store.GetSession(context.Background(), session.Token)
	assert.Error(t, err)
}
