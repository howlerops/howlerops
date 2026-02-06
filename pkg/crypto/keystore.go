package crypto

import (
	"fmt"
	"sync"
	"time"
)

// KeyStore manages encryption keys in memory with caching
type KeyStore struct {
	// User key cache
	userKey     []byte
	userKeySalt []byte
	locked      bool

	// Team key cache
	teamKeys map[string][]byte // teamID -> teamKey

	// Thread safety
	mu sync.RWMutex
}

// NewKeyStore creates a new key store
func NewKeyStore() *KeyStore {
	return &KeyStore{
		teamKeys: make(map[string][]byte),
		locked:   true, // Start locked
	}
}

// Unlock derives and caches the user key from passphrase
func (ks *KeyStore) Unlock(passphrase string, salt []byte) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.locked {
		return fmt.Errorf("key store is already unlocked")
	}

	// Derive user key
	key, err := DeriveKey(passphrase, salt)
	if err != nil {
		return fmt.Errorf("failed to derive user key: %w", err)
	}

	// Cache the key and salt
	ks.userKey = make([]byte, len(key))
	copy(ks.userKey, key)
	ks.userKeySalt = make([]byte, len(salt))
	copy(ks.userKeySalt, salt)
	ks.locked = false

	return nil
}

// Lock clears all cached keys from memory
func (ks *KeyStore) Lock() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Clear user key
	if ks.userKey != nil {
		ks.userKey = nil
	}
	if ks.userKeySalt != nil {
		ks.userKeySalt = nil
	}

	// Clear team keys
	ks.teamKeys = make(map[string][]byte)

	ks.locked = true
}

// IsLocked returns whether the key store is locked
func (ks *KeyStore) IsLocked() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.locked
}

// GetUserKey returns the cached user key
func (ks *KeyStore) GetUserKey() ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.locked {
		return nil, fmt.Errorf("key store is locked")
	}

	if ks.userKey == nil {
		return nil, fmt.Errorf("user key not available")
	}

	// Return a copy to prevent external modification
	key := make([]byte, len(ks.userKey))
	copy(key, ks.userKey)
	return key, nil
}

// GetUserKeySalt returns the cached user key salt
func (ks *KeyStore) GetUserKeySalt() ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.locked {
		return nil, fmt.Errorf("key store is locked")
	}

	if ks.userKeySalt == nil {
		return nil, fmt.Errorf("user key salt not available")
	}

	// Return a copy to prevent external modification
	salt := make([]byte, len(ks.userKeySalt))
	copy(salt, ks.userKeySalt)
	return salt, nil
}

// SetTeamKey caches a team key for the given team ID
func (ks *KeyStore) SetTeamKey(teamID string, teamKey []byte) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.locked {
		return fmt.Errorf("key store is locked")
	}

	if len(teamKey) != KeySize {
		return fmt.Errorf("invalid team key size: expected %d, got %d", KeySize, len(teamKey))
	}

	// Cache a copy
	ks.teamKeys[teamID] = make([]byte, len(teamKey))
	copy(ks.teamKeys[teamID], teamKey)

	return nil
}

// GetTeamKey returns the cached team key for the given team ID
func (ks *KeyStore) GetTeamKey(teamID string) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.locked {
		return nil, fmt.Errorf("key store is locked")
	}

	teamKey, exists := ks.teamKeys[teamID]
	if !exists {
		return nil, fmt.Errorf("team key not found for team: %s", teamID)
	}

	// Return a copy to prevent external modification
	key := make([]byte, len(teamKey))
	copy(key, teamKey)
	return key, nil
}

// RemoveTeamKey removes a team key from cache
func (ks *KeyStore) RemoveTeamKey(teamID string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	delete(ks.teamKeys, teamID)
}

// ClearTeamKeys removes all team keys from cache
func (ks *KeyStore) ClearTeamKeys() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.teamKeys = make(map[string][]byte)
}

// KeyInfo contains information about a cached key
type KeyInfo struct {
	IsLocked     bool
	HasUserKey   bool
	TeamKeyCount int
	LastAccess   time.Time
}

// GetInfo returns information about the key store state
func (ks *KeyStore) GetInfo() KeyInfo {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return KeyInfo{
		IsLocked:     ks.locked,
		HasUserKey:   ks.userKey != nil,
		TeamKeyCount: len(ks.teamKeys),
		LastAccess:   time.Now(),
	}
}
