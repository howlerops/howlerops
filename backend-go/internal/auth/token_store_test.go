package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/auth"
)

// ====================================================================
// InMemoryTokenStore Tests
// ====================================================================

func TestInMemoryTokenStore_CreateToken(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Verify token can be retrieved
	retrieved, err := store.GetToken(ctx, "abc123", auth.TokenTypeEmailVerification)
	require.NoError(t, err)
	assert.Equal(t, token.UserID, retrieved.UserID)
	assert.Equal(t, token.Type, retrieved.Type)
}

func TestInMemoryTokenStore_CreateToken_Duplicate(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Create first time
	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Try to create duplicate
	err = store.CreateToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInMemoryTokenStore_GetToken_Success(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Retrieve token
	retrieved, err := store.GetToken(ctx, "abc123", auth.TokenTypePasswordReset)
	require.NoError(t, err)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, auth.TokenTypePasswordReset, retrieved.Type)
	assert.Nil(t, retrieved.UsedAt)
}

func TestInMemoryTokenStore_GetToken_NotFound(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	_, err := store.GetToken(ctx, "nonexistent", auth.TokenTypeEmailVerification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTokenStore_GetToken_WrongType(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Try to get with wrong type
	_, err = store.GetToken(ctx, "abc123", auth.TokenTypePasswordReset)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

func TestInMemoryTokenStore_GetToken_Expired(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Try to get expired token
	_, err = store.GetToken(ctx, "abc123", auth.TokenTypePasswordReset)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInMemoryTokenStore_GetToken_AlreadyUsed(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Mark token as used
	err = store.MarkTokenUsed(ctx, "abc123")
	require.NoError(t, err)

	// Try to get used token
	_, err = store.GetToken(ctx, "abc123", auth.TokenTypeEmailVerification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestInMemoryTokenStore_MarkTokenUsed(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Mark as used
	err = store.MarkTokenUsed(ctx, "abc123")
	require.NoError(t, err)

	// Verify token is marked as used
	_, err = store.GetToken(ctx, "abc123", auth.TokenTypePasswordReset)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestInMemoryTokenStore_MarkTokenUsed_NotFound(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	err := store.MarkTokenUsed(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTokenStore_DeleteToken(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Delete token
	err = store.DeleteToken(ctx, "abc123")
	require.NoError(t, err)

	// Verify token is deleted
	_, err = store.GetToken(ctx, "abc123", auth.TokenTypeEmailVerification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTokenStore_DeleteToken_NotFound(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	err := store.DeleteToken(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInMemoryTokenStore_CleanupExpiredTokens(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	// Create expired token
	expiredToken := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "expired",
		Type:      auth.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	// Create valid token
	validToken := &auth.Token{
		ID:        "token-2",
		UserID:    "user-456",
		Token:     "valid",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, expiredToken)
	require.NoError(t, err)

	err = store.CreateToken(ctx, validToken)
	require.NoError(t, err)

	// Cleanup expired tokens
	err = store.CleanupExpiredTokens(ctx)
	require.NoError(t, err)

	// Verify expired token is deleted
	_, err = store.GetToken(ctx, "expired", auth.TokenTypePasswordReset)
	assert.Error(t, err)

	// Verify valid token still exists
	retrieved, err := store.GetToken(ctx, "valid", auth.TokenTypeEmailVerification)
	require.NoError(t, err)
	assert.Equal(t, "user-456", retrieved.UserID)
}

func TestInMemoryTokenStore_CleanupExpiredTokens_Empty(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	// Cleanup on empty store should not error
	err := store.CleanupExpiredTokens(ctx)
	assert.NoError(t, err)
}

func TestInMemoryTokenStore_CleanupExpiredTokens_AllValid(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	// Create multiple valid tokens
	for i := 0; i < 5; i++ {
		token := &auth.Token{
			ID:        "token-" + string(rune(i)),
			UserID:    "user-123",
			Token:     "token" + string(rune(i)),
			Type:      auth.TokenTypeEmailVerification,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}
		err := store.CreateToken(ctx, token)
		require.NoError(t, err)
	}

	// Cleanup should not remove any tokens
	err := store.CleanupExpiredTokens(ctx)
	require.NoError(t, err)

	// Verify all tokens still exist
	for i := 0; i < 5; i++ {
		_, err := store.GetToken(ctx, "token"+string(rune(i)), auth.TokenTypeEmailVerification)
		assert.NoError(t, err, "Token %d should still exist", i)
	}
}

// ====================================================================
// GenerateSecureToken Tests
// ====================================================================

func TestGenerateSecureToken_Success(t *testing.T) {
	token, err := auth.GenerateSecureToken(32)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 64, len(token)) // 32 bytes = 64 hex characters
}

func TestGenerateSecureToken_DefaultLength(t *testing.T) {
	// Test with length < 16 to verify default
	token, err := auth.GenerateSecureToken(8)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 64, len(token)) // Should use default 32 bytes = 64 hex characters
}

func TestGenerateSecureToken_CustomLength(t *testing.T) {
	token, err := auth.GenerateSecureToken(64)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 128, len(token)) // 64 bytes = 128 hex characters
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	// Generate multiple tokens and verify they're unique
	tokens := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, err := auth.GenerateSecureToken(32)
		require.NoError(t, err)

		// Check for duplicates
		assert.False(t, tokens[token], "Generated duplicate token")
		tokens[token] = true
	}

	assert.Equal(t, 100, len(tokens))
}

// ====================================================================
// Token Type Tests
// ====================================================================

func TestTokenType_Constants(t *testing.T) {
	assert.Equal(t, auth.TokenType("email_verification"), auth.TokenTypeEmailVerification)
	assert.Equal(t, auth.TokenType("password_reset"), auth.TokenTypePasswordReset)
}

func TestInMemoryTokenStore_MultipleTokenTypes(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	// Create email verification token
	emailToken := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "email-token",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Create password reset token
	resetToken := &auth.Token{
		ID:        "token-2",
		UserID:    "user-123",
		Token:     "reset-token",
		Type:      auth.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, emailToken)
	require.NoError(t, err)

	err = store.CreateToken(ctx, resetToken)
	require.NoError(t, err)

	// Verify both tokens can be retrieved with correct types
	retrieved1, err := store.GetToken(ctx, "email-token", auth.TokenTypeEmailVerification)
	require.NoError(t, err)
	assert.Equal(t, auth.TokenTypeEmailVerification, retrieved1.Type)

	retrieved2, err := store.GetToken(ctx, "reset-token", auth.TokenTypePasswordReset)
	require.NoError(t, err)
	assert.Equal(t, auth.TokenTypePasswordReset, retrieved2.Type)

	// Verify type checking works
	_, err = store.GetToken(ctx, "email-token", auth.TokenTypePasswordReset)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

// ====================================================================
// Concurrency Tests
// ====================================================================

func TestInMemoryTokenStore_ConcurrentAccess(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	// Create initial token
	token := &auth.Token{
		ID:        "token-1",
		UserID:    "user-123",
		Token:     "abc123",
		Type:      auth.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateToken(ctx, token)
	require.NoError(t, err)

	// Run multiple goroutines reading the token concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := store.GetToken(ctx, "abc123", auth.TokenTypeEmailVerification)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestInMemoryTokenStore_ConcurrentCreateAndCleanup(t *testing.T) {
	store := auth.NewInMemoryTokenStore()
	ctx := context.Background()

	done := make(chan bool)

	// Create tokens concurrently
	for i := 0; i < 5; i++ {
		go func(idx int) {
			token := &auth.Token{
				ID:        "token-" + string(rune(idx)),
				UserID:    "user-123",
				Token:     "token" + string(rune(idx)),
				Type:      auth.TokenTypeEmailVerification,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				CreatedAt: time.Now(),
			}
			err := store.CreateToken(ctx, token)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Run cleanup concurrently
	go func() {
		time.Sleep(10 * time.Millisecond)
		err := store.CleanupExpiredTokens(ctx)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 6; i++ {
		<-done
	}
}
