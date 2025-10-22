package crypto

import (
	"testing"
)

func TestKeyStore(t *testing.T) {
	ks := NewKeyStore()

	// Test initial state
	if !ks.IsLocked() {
		t.Error("Key store should start locked")
	}

	// Test unlocking
	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	if ks.IsLocked() {
		t.Error("Key store should be unlocked")
	}

	// Test getting user key
	userKey, err := ks.GetUserKey()
	if err != nil {
		t.Fatalf("Failed to get user key: %v", err)
	}

	if len(userKey) != KeySize {
		t.Errorf("Expected user key size %d, got %d", KeySize, len(userKey))
	}

	// Test getting user key salt
	userKeySalt, err := ks.GetUserKeySalt()
	if err != nil {
		t.Fatalf("Failed to get user key salt: %v", err)
	}

	if len(userKeySalt) != 32 {
		t.Errorf("Expected salt size 32, got %d", len(userKeySalt))
	}

	// Test locking
	ks.Lock()
	if !ks.IsLocked() {
		t.Error("Key store should be locked after Lock()")
	}

	// Test getting key when locked
	_, err = ks.GetUserKey()
	if err == nil {
		t.Error("Expected error when getting key while locked")
	}
}

func TestKeyStoreUnlockTwice(t *testing.T) {
	ks := NewKeyStore()

	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Unlock first time
	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock first time: %v", err)
	}

	// Try to unlock again
	err = ks.Unlock(passphrase, salt)
	if err == nil {
		t.Error("Expected error when unlocking already unlocked store")
	}
}

func TestKeyStoreTeamKeys(t *testing.T) {
	ks := NewKeyStore()

	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Test setting team key
	teamID := "team-123"
	teamKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate team key: %v", err)
	}

	err = ks.SetTeamKey(teamID, teamKey)
	if err != nil {
		t.Fatalf("Failed to set team key: %v", err)
	}

	// Test getting team key
	retrievedKey, err := ks.GetTeamKey(teamID)
	if err != nil {
		t.Fatalf("Failed to get team key: %v", err)
	}

	if string(retrievedKey) != string(teamKey) {
		t.Error("Retrieved team key doesn't match original")
	}

	// Test getting non-existent team key
	_, err = ks.GetTeamKey("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent team key")
	}

	// Test removing team key
	ks.RemoveTeamKey(teamID)
	_, err = ks.GetTeamKey(teamID)
	if err == nil {
		t.Error("Expected error after removing team key")
	}
}

func TestKeyStoreTeamKeysWhenLocked(t *testing.T) {
	ks := NewKeyStore()

	teamID := "team-123"
	teamKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate team key: %v", err)
	}

	// Test setting team key when locked
	err = ks.SetTeamKey(teamID, teamKey)
	if err == nil {
		t.Error("Expected error when setting team key while locked")
	}

	// Test getting team key when locked
	_, err = ks.GetTeamKey(teamID)
	if err == nil {
		t.Error("Expected error when getting team key while locked")
	}
}

func TestKeyStoreInvalidTeamKeySize(t *testing.T) {
	ks := NewKeyStore()

	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Test with wrong team key size
	wrongKey := []byte("short")
	err = ks.SetTeamKey("team-123", wrongKey)
	if err == nil {
		t.Error("Expected error for wrong team key size")
	}
}

func TestKeyStoreClearTeamKeys(t *testing.T) {
	ks := NewKeyStore()

	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Set multiple team keys
	teamKeys := []string{"team-1", "team-2", "team-3"}
	for _, teamID := range teamKeys {
		teamKey, err := GenerateRandomBytes(KeySize)
		if err != nil {
			t.Fatalf("Failed to generate team key for %s: %v", teamID, err)
		}

		err = ks.SetTeamKey(teamID, teamKey)
		if err != nil {
			t.Fatalf("Failed to set team key for %s: %v", teamID, err)
		}
	}

	// Verify all keys are set
	for _, teamID := range teamKeys {
		_, err := ks.GetTeamKey(teamID)
		if err != nil {
			t.Errorf("Failed to get team key for %s: %v", teamID, err)
		}
	}

	// Clear all team keys
	ks.ClearTeamKeys()

	// Verify all keys are cleared
	for _, teamID := range teamKeys {
		_, err := ks.GetTeamKey(teamID)
		if err == nil {
			t.Errorf("Expected error for cleared team key %s", teamID)
		}
	}
}

func TestKeyStoreInfo(t *testing.T) {
	ks := NewKeyStore()

	// Test info when locked
	info := ks.GetInfo()
	if !info.IsLocked {
		t.Error("Info should show locked when locked")
	}
	if info.HasUserKey {
		t.Error("Info should not show user key when locked")
	}
	if info.TeamKeyCount != 0 {
		t.Errorf("Expected 0 team keys, got %d", info.TeamKeyCount)
	}

	// Unlock and set team keys
	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	err = ks.Unlock(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Set a team key
	teamKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate team key: %v", err)
	}

	err = ks.SetTeamKey("team-123", teamKey)
	if err != nil {
		t.Fatalf("Failed to set team key: %v", err)
	}

	// Test info when unlocked
	info = ks.GetInfo()
	if info.IsLocked {
		t.Error("Info should show unlocked when unlocked")
	}
	if !info.HasUserKey {
		t.Error("Info should show user key when unlocked")
	}
	if info.TeamKeyCount != 1 {
		t.Errorf("Expected 1 team key, got %d", info.TeamKeyCount)
	}
}
