package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	// Default session expiration time
	defaultSessionExpiration = 5 * time.Minute
)

// sessionData wraps WebAuthn session data with expiration
type sessionData struct {
	session   *webauthn.SessionData
	expiresAt time.Time
}

// SessionStore handles temporary storage of WebAuthn session data
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionData
	ttl      time.Duration
	// Background cleanup
	stopCleanup chan struct{}
}

// NewSessionStore creates a new session store
func NewSessionStore() *SessionStore {
	ss := &SessionStore{
		sessions:    make(map[string]*sessionData),
		ttl:         defaultSessionExpiration,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup
	go ss.cleanupExpired()

	return ss
}

// NewSessionStoreWithTTL creates a new session store with custom TTL
func NewSessionStoreWithTTL(ttl time.Duration) *SessionStore {
	ss := &SessionStore{
		sessions:    make(map[string]*sessionData),
		ttl:         ttl,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup
	go ss.cleanupExpired()

	return ss
}

// StoreSession stores a WebAuthn session with automatic expiration
func (ss *SessionStore) StoreSession(sessionID string, session *webauthn.SessionData) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if session == nil {
		return fmt.Errorf("session data cannot be nil")
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions[sessionID] = &sessionData{
		session:   session,
		expiresAt: time.Now().Add(ss.ttl),
	}

	return nil
}

// GetSession retrieves a WebAuthn session
func (ss *SessionStore) GetSession(sessionID string) (*webauthn.SessionData, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	data, ok := ss.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Check if expired
	if time.Now().After(data.expiresAt) {
		// Expired, but don't delete here (let cleanup goroutine handle it)
		return nil, fmt.Errorf("session expired")
	}

	return data.session, nil
}

// DeleteSession removes a session
func (ss *SessionStore) DeleteSession(sessionID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.sessions, sessionID)
	return nil
}

// HasSession checks if a session exists and is not expired
func (ss *SessionStore) HasSession(sessionID string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	data, ok := ss.sessions[sessionID]
	if !ok {
		return false
	}

	// Check expiration
	return time.Now().Before(data.expiresAt)
}

// RefreshSession extends the session expiration
func (ss *SessionStore) RefreshSession(sessionID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	data, ok := ss.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Extend expiration
	data.expiresAt = time.Now().Add(ss.ttl)

	return nil
}

// Clear removes all sessions
func (ss *SessionStore) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions = make(map[string]*sessionData)
}

// Count returns the number of active sessions
func (ss *SessionStore) Count() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return len(ss.sessions)
}

// cleanupExpired removes expired sessions periodically
func (ss *SessionStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ss.mu.Lock()
			now := time.Now()
			for id, data := range ss.sessions {
				if now.After(data.expiresAt) {
					delete(ss.sessions, id)
				}
			}
			ss.mu.Unlock()

		case <-ss.stopCleanup:
			return
		}
	}
}

// Close stops the background cleanup goroutine
func (ss *SessionStore) Close() {
	close(ss.stopCleanup)
}
