package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sql-studio/backend-go/internal/auth"
)

// InMemoryUserStore provides an in-memory implementation of UserStore
type InMemoryUserStore struct {
	users map[string]*auth.User
	mu    sync.RWMutex
}

// NewInMemoryUserStore creates a new in-memory user store
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*auth.User),
	}
}

// GetUser retrieves a user by ID
func (s *InMemoryUserStore) GetUser(ctx context.Context, id string) (*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Return a copy to prevent external modifications
	userCopy := *user
	return &userCopy, nil
}

// GetUserByUsername retrieves a user by username
func (s *InMemoryUserStore) GetUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Username == username {
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// GetUserByEmail retrieves a user by email
func (s *InMemoryUserStore) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// CreateUser creates a new user
func (s *InMemoryUserStore) CreateUser(ctx context.Context, user *auth.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	if _, exists := s.users[user.ID]; exists {
		return fmt.Errorf("user already exists")
	}

	// Check if username is taken
	for _, existingUser := range s.users {
		if existingUser.Username == user.Username {
			return fmt.Errorf("username already taken")
		}
		if existingUser.Email == user.Email {
			return fmt.Errorf("email already taken")
		}
	}

	// Store a copy
	userCopy := *user
	s.users[user.ID] = &userCopy

	return nil
}

// UpdateUser updates an existing user
func (s *InMemoryUserStore) UpdateUser(ctx context.Context, user *auth.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}

	// Store a copy
	userCopy := *user
	s.users[user.ID] = &userCopy

	return nil
}

// DeleteUser deletes a user
func (s *InMemoryUserStore) DeleteUser(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.users, id)
	return nil
}

// ListUsers returns a list of users
func (s *InMemoryUserStore) ListUsers(ctx context.Context, limit, offset int) ([]*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []*auth.User
	count := 0

	for _, user := range s.users {
		if count < offset {
			count++
			continue
		}

		if len(users) >= limit {
			break
		}

		userCopy := *user
		users = append(users, &userCopy)
		count++
	}

	return users, nil
}

// InMemorySessionStore provides an in-memory implementation of SessionStore
type InMemorySessionStore struct {
	sessions map[string]*auth.Session
	mu       sync.RWMutex
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*auth.Session),
	}
}

// CreateSession creates a new session
func (s *InMemorySessionStore) CreateSession(ctx context.Context, session *auth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store by token
	sessionCopy := *session
	s.sessions[session.Token] = &sessionCopy

	return nil
}

// GetSession retrieves a session by token
func (s *InMemorySessionStore) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	sessionCopy := *session
	return &sessionCopy, nil
}

// UpdateSession updates an existing session
func (s *InMemorySessionStore) UpdateSession(ctx context.Context, session *auth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.Token]; !exists {
		return fmt.Errorf("session not found")
	}

	sessionCopy := *session
	s.sessions[session.Token] = &sessionCopy

	return nil
}

// DeleteSession deletes a session
func (s *InMemorySessionStore) DeleteSession(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[token]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(s.sessions, token)
	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (s *InMemorySessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, token)
		}
	}

	return nil
}

// GetUserSessions returns all sessions for a user
func (s *InMemorySessionStore) GetUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*auth.Session
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessionCopy := *session
			sessions = append(sessions, &sessionCopy)
		}
	}

	return sessions, nil
}

// CleanupExpiredSessions removes expired sessions
func (s *InMemorySessionStore) CleanupExpiredSessions(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) || !session.Active {
			delete(s.sessions, token)
		}
	}

	return nil
}

// InMemoryLoginAttemptStore provides an in-memory implementation of LoginAttemptStore
type InMemoryLoginAttemptStore struct {
	attempts []auth.LoginAttempt
	mu       sync.RWMutex
}

// NewInMemoryLoginAttemptStore creates a new in-memory login attempt store
func NewInMemoryLoginAttemptStore() *InMemoryLoginAttemptStore {
	return &InMemoryLoginAttemptStore{
		attempts: make([]auth.LoginAttempt, 0),
	}
}

// RecordAttempt records a login attempt
func (s *InMemoryLoginAttemptStore) RecordAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts = append(s.attempts, *attempt)
	return nil
}

// GetAttempts retrieves login attempts for an IP/username since a given time
func (s *InMemoryLoginAttemptStore) GetAttempts(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []*auth.LoginAttempt
	for i := range s.attempts {
		attempt := &s.attempts[i]
		if (ip == "" || attempt.IP == ip) &&
			(username == "" || attempt.Username == username) &&
			attempt.Timestamp.After(since) {
			attemptCopy := *attempt
			matching = append(matching, &attemptCopy)
		}
	}

	return matching, nil
}

// CleanupOldAttempts removes login attempts older than the specified time
func (s *InMemoryLoginAttemptStore) CleanupOldAttempts(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []auth.LoginAttempt
	for _, attempt := range s.attempts {
		if attempt.Timestamp.After(before) {
			filtered = append(filtered, attempt)
		}
	}

	s.attempts = filtered
	return nil
}