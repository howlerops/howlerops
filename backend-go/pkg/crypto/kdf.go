package crypto

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters for key derivation
	Argon2Time    = 1         // 1 iteration (fast unlock)
	Argon2Memory  = 64 * 1024 // 64 MB memory
	Argon2Threads = 4         // 4 threads
	Argon2KeyLen  = KeySize   // 32 bytes for AES-256
)

// DeriveKey derives an encryption key from a passphrase using Argon2id
// Returns the derived key and salt used for derivation
func DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	// Derive key using Argon2id
	key := argon2.IDKey([]byte(passphrase), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	return key, nil
}

// DeriveKeyWithSalt generates a new salt and derives a key
// Returns the derived key and the generated salt
func DeriveKeyWithSalt(passphrase string) (key []byte, salt []byte, err error) {
	// Generate random salt
	salt, err = GenerateRandomBytes(32) // 256-bit salt
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key
	key, err = DeriveKey(passphrase, salt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, salt, nil
}

// DeriveTeamKey derives a team encryption key from team secret and user key
// This enables team members to share encrypted secrets
func DeriveTeamKey(teamSecret []byte, userKey []byte) ([]byte, error) {
	if len(teamSecret) == 0 {
		return nil, fmt.Errorf("team secret cannot be empty")
	}
	if len(userKey) != KeySize {
		return nil, fmt.Errorf("invalid user key size: expected %d, got %d", KeySize, len(userKey))
	}

	// Combine team secret and user key
	combined := append(teamSecret, userKey...)

	// Hash to get deterministic team key
	hash := sha256.Sum256(combined)
	return hash[:], nil
}

// HashPassphrase creates a hash of the passphrase for verification
// Uses Argon2id with different parameters than key derivation
func HashPassphrase(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	// Use stronger parameters for passphrase hashing
	hash := argon2.IDKey([]byte(passphrase), salt, 3, 128*1024, 4, 32)
	return hash, nil
}

// HashPassphraseWithSalt generates a new salt and hashes the passphrase
func HashPassphraseWithSalt(passphrase string) (hash []byte, salt []byte, err error) {
	// Generate random salt
	salt, err = GenerateRandomBytes(32)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash passphrase
	hash, err = HashPassphrase(passphrase, salt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash passphrase: %w", err)
	}

	return hash, salt, nil
}

// VerifyPassphrase verifies a passphrase against its hash
func VerifyPassphrase(passphrase string, hash []byte, salt []byte) (bool, error) {
	computedHash, err := HashPassphrase(passphrase, salt)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}

	// Constant-time comparison
	if len(hash) != len(computedHash) {
		return false, nil
	}

	result := 0
	for i := 0; i < len(hash); i++ {
		result |= int(hash[i]) ^ int(computedHash[i])
	}

	return result == 0, nil
}
