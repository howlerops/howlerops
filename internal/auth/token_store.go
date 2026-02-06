package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TokenType represents the type of token
type TokenType string

const (
	// #nosec G101 - these are token type constants, not actual passwords
	TokenTypeEmailVerification TokenType = "email_verification"
	TokenTypePasswordReset     TokenType = "password_reset"
)

// Token represents a verification or reset token
type Token struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Token     string     `json:"token"`
	Type      TokenType  `json:"type"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenStore defines the interface for token storage
type TokenStore interface {
	CreateToken(ctx context.Context, token *Token) error
	GetToken(ctx context.Context, token string, tokenType TokenType) (*Token, error)
	MarkTokenUsed(ctx context.Context, token string) error
	DeleteToken(ctx context.Context, token string) error
	CleanupExpiredTokens(ctx context.Context) error
}

// InMemoryTokenStore provides an in-memory implementation of TokenStore
type InMemoryTokenStore struct {
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewInMemoryTokenStore creates a new in-memory token store
func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{
		tokens: make(map[string]*Token),
	}
}

// CreateToken creates a new token
func (s *InMemoryTokenStore) CreateToken(ctx context.Context, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if token already exists
	if _, exists := s.tokens[token.Token]; exists {
		return fmt.Errorf("token already exists")
	}

	// Store a copy
	tokenCopy := *token
	s.tokens[token.Token] = &tokenCopy

	return nil
}

// GetToken retrieves a token by its value and type
func (s *InMemoryTokenStore) GetToken(ctx context.Context, tokenValue string, tokenType TokenType) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[tokenValue]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if token.Type != tokenType {
		return nil, fmt.Errorf("invalid token type")
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	// Check if token is already used
	if token.UsedAt != nil {
		return nil, fmt.Errorf("token already used")
	}

	tokenCopy := *token
	return &tokenCopy, nil
}

// MarkTokenUsed marks a token as used
func (s *InMemoryTokenStore) MarkTokenUsed(ctx context.Context, tokenValue string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenValue]
	if !exists {
		return fmt.Errorf("token not found")
	}

	now := time.Now()
	token.UsedAt = &now

	return nil
}

// DeleteToken deletes a token
func (s *InMemoryTokenStore) DeleteToken(ctx context.Context, tokenValue string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tokens[tokenValue]; !exists {
		return fmt.Errorf("token not found")
	}

	delete(s.tokens, tokenValue)
	return nil
}

// CleanupExpiredTokens removes expired tokens
func (s *InMemoryTokenStore) CleanupExpiredTokens(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for tokenValue, token := range s.tokens {
		if now.After(token.ExpiresAt) {
			delete(s.tokens, tokenValue)
		}
	}

	return nil
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	if length < 16 {
		length = 32 // Default to 32 bytes (64 hex characters)
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
