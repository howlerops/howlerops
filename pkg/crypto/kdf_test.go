package crypto

import (
	"testing"
)

func TestDeriveKey(t *testing.T) {
	passphrase := "test-passphrase-123"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Derive key
	key, err := DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	// Verify key size
	if len(key) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key))
	}

	// Verify deterministic derivation
	key2, err := DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key second time: %v", err)
	}

	if string(key) != string(key2) {
		t.Error("Key derivation should be deterministic")
	}
}

func TestDeriveKeyWithSalt(t *testing.T) {
	passphrase := "test-passphrase-456"

	key, salt, err := DeriveKeyWithSalt(passphrase)
	if err != nil {
		t.Fatalf("Failed to derive key with salt: %v", err)
	}

	// Verify key size
	if len(key) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key))
	}

	// Verify salt size
	if len(salt) != 32 {
		t.Errorf("Expected salt size 32, got %d", len(salt))
	}

	// Verify we can derive the same key with the returned salt
	key2, err := DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key with returned salt: %v", err)
	}

	if string(key) != string(key2) {
		t.Error("Key derivation should be consistent")
	}
}

func TestDeriveKeyDifferentPassphrases(t *testing.T) {
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	passphrase1 := "passphrase-one"
	passphrase2 := "passphrase-two"

	key1, err := DeriveKey(passphrase1, salt)
	if err != nil {
		t.Fatalf("Failed to derive key for passphrase 1: %v", err)
	}

	key2, err := DeriveKey(passphrase2, salt)
	if err != nil {
		t.Fatalf("Failed to derive key for passphrase 2: %v", err)
	}

	// Keys should be different
	if string(key1) == string(key2) {
		t.Error("Different passphrases should produce different keys")
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	passphrase := "same-passphrase"

	salt1, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt 1: %v", err)
	}

	salt2, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt 2: %v", err)
	}

	key1, err := DeriveKey(passphrase, salt1)
	if err != nil {
		t.Fatalf("Failed to derive key with salt 1: %v", err)
	}

	key2, err := DeriveKey(passphrase, salt2)
	if err != nil {
		t.Fatalf("Failed to derive key with salt 2: %v", err)
	}

	// Keys should be different
	if string(key1) == string(key2) {
		t.Error("Different salts should produce different keys")
	}
}

func TestDeriveTeamKey(t *testing.T) {
	teamSecret, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate team secret: %v", err)
	}

	userKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	// Derive team key
	teamKey, err := DeriveTeamKey(teamSecret, userKey)
	if err != nil {
		t.Fatalf("Failed to derive team key: %v", err)
	}

	// Verify team key size
	if len(teamKey) != KeySize {
		t.Errorf("Expected team key size %d, got %d", KeySize, len(teamKey))
	}

	// Verify deterministic derivation
	teamKey2, err := DeriveTeamKey(teamSecret, userKey)
	if err != nil {
		t.Fatalf("Failed to derive team key second time: %v", err)
	}

	if string(teamKey) != string(teamKey2) {
		t.Error("Team key derivation should be deterministic")
	}
}

func TestDeriveTeamKeyDifferentInputs(t *testing.T) {
	teamSecret1, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate team secret 1: %v", err)
	}

	teamSecret2, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate team secret 2: %v", err)
	}

	userKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	teamKey1, err := DeriveTeamKey(teamSecret1, userKey)
	if err != nil {
		t.Fatalf("Failed to derive team key 1: %v", err)
	}

	teamKey2, err := DeriveTeamKey(teamSecret2, userKey)
	if err != nil {
		t.Fatalf("Failed to derive team key 2: %v", err)
	}

	// Different team secrets should produce different keys
	if string(teamKey1) == string(teamKey2) {
		t.Error("Different team secrets should produce different team keys")
	}
}

func TestDeriveTeamKeyInvalidInputs(t *testing.T) {
	userKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	// Test with empty team secret
	_, err = DeriveTeamKey([]byte{}, userKey)
	if err == nil {
		t.Error("Expected error for empty team secret")
	}

	// Test with nil team secret
	_, err = DeriveTeamKey(nil, userKey)
	if err == nil {
		t.Error("Expected error for nil team secret")
	}

	// Test with wrong user key size
	wrongUserKey := []byte("short")
	teamSecret, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate team secret: %v", err)
	}

	_, err = DeriveTeamKey(teamSecret, wrongUserKey)
	if err == nil {
		t.Error("Expected error for wrong user key size")
	}
}

func TestHashPassphrase(t *testing.T) {
	passphrase := "test-passphrase-hash"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Hash passphrase
	hash, err := HashPassphrase(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to hash passphrase: %v", err)
	}

	// Verify hash size
	if len(hash) != 32 {
		t.Errorf("Expected hash size 32, got %d", len(hash))
	}

	// Verify deterministic hashing
	hash2, err := HashPassphrase(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to hash passphrase second time: %v", err)
	}

	if string(hash) != string(hash2) {
		t.Error("Passphrase hashing should be deterministic")
	}
}

func TestHashPassphraseWithSalt(t *testing.T) {
	passphrase := "test-passphrase-with-salt"

	hash, salt, err := HashPassphraseWithSalt(passphrase)
	if err != nil {
		t.Fatalf("Failed to hash passphrase with salt: %v", err)
	}

	// Verify hash size
	if len(hash) != 32 {
		t.Errorf("Expected hash size 32, got %d", len(hash))
	}

	// Verify salt size
	if len(salt) != 32 {
		t.Errorf("Expected salt size 32, got %d", len(salt))
	}

	// Verify we can hash the same passphrase with the returned salt
	hash2, err := HashPassphrase(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to hash passphrase with returned salt: %v", err)
	}

	if string(hash) != string(hash2) {
		t.Error("Passphrase hashing should be consistent")
	}
}

func TestVerifyPassphrase(t *testing.T) {
	passphrase := "correct-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Hash the passphrase
	hash, err := HashPassphrase(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to hash passphrase: %v", err)
	}

	// Verify correct passphrase
	valid, err := VerifyPassphrase(passphrase, hash, salt)
	if err != nil {
		t.Fatalf("Failed to verify passphrase: %v", err)
	}

	if !valid {
		t.Error("Correct passphrase should be valid")
	}

	// Verify incorrect passphrase
	valid, err = VerifyPassphrase("wrong-passphrase", hash, salt)
	if err != nil {
		t.Fatalf("Failed to verify wrong passphrase: %v", err)
	}

	if valid {
		t.Error("Wrong passphrase should be invalid")
	}
}

func TestVerifyPassphraseInvalidInputs(t *testing.T) {
	passphrase := "test-passphrase"
	salt, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	hash, err := HashPassphrase(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to hash passphrase: %v", err)
	}

	// Test with empty salt
	_, err = VerifyPassphrase(passphrase, hash, []byte{})
	if err == nil {
		t.Error("Expected error for empty salt")
	}

	// Test with nil salt
	_, err = VerifyPassphrase(passphrase, hash, nil)
	if err == nil {
		t.Error("Expected error for nil salt")
	}
}
