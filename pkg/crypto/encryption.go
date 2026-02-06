package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// AES-256-GCM key size
	KeySize = 32
	// GCM nonce size (96 bits)
	NonceSize = 12
	// GCM tag size (128 bits)
	TagSize = 16
)

// EncryptSecret encrypts plaintext using AES-256-GCM
// Returns ciphertext (including auth tag) and nonce
func EncryptSecret(plaintext []byte, key []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != KeySize {
		return nil, nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce = make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// #nosec G407 -- False positive: nonce is randomly generated above using crypto/rand,
	// not hardcoded. gosec doesn't track the data flow from lines 44-47.
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// DecryptSecret decrypts ciphertext using AES-256-GCM
// Expects ciphertext to include the auth tag
func DecryptSecret(ciphertext []byte, nonce []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: expected %d, got %d", NonceSize, len(nonce))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// ValidateKey validates that a key has the correct size
func ValidateKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}
	return nil
}

// PBKDF2 Password Encryption Functions
// These are separate from the Argon2-based team secret encryption above
// and are specifically for encrypting database passwords using user passwords

const (
	// PBKDF2Iterations is the OWASP 2023 recommended iteration count for PBKDF2-SHA256
	PBKDF2Iterations = 600_000
	// PBKDF2SaltLength is 256 bits
	PBKDF2SaltLength = 32
)

// EncryptedPasswordData represents encrypted password data with IV and authentication tag
// This is used for database password encryption
type EncryptedPasswordData struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded ciphertext
	IV         string `json:"iv"`         // Base64-encoded IV/nonce
	AuthTag    string `json:"authTag"`    // Base64-encoded auth tag
}

// EncryptedMasterKey represents an encrypted master key with PBKDF2 parameters
// This is the master key encrypted with a password-derived key
type EncryptedMasterKey struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded ciphertext
	IV         string `json:"iv"`         // Base64-encoded IV/nonce
	AuthTag    string `json:"authTag"`    // Base64-encoded auth tag
	Salt       string `json:"salt"`       // Base64-encoded PBKDF2 salt
	Iterations int    `json:"iterations"` // PBKDF2 iteration count
}

// DeriveKeyFromPassword derives a cryptographic key from a password using PBKDF2-SHA256
func DeriveKeyFromPassword(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
}

// GenerateMasterKey generates a random master key for encrypting database passwords
func GenerateMasterKey() ([]byte, error) {
	return GenerateRandomBytes(KeySize)
}

// EncryptPasswordWithKey encrypts a database password using AES-256-GCM
// Returns encrypted data split into ciphertext, IV, and auth tag
func EncryptPasswordWithKey(password string, key []byte) (*EncryptedPasswordData, error) {
	// Use existing EncryptSecret function
	ciphertextWithTag, nonce, err := EncryptSecret([]byte(password), key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Split ciphertext and auth tag (GCM appends 16-byte tag)
	if len(ciphertextWithTag) < TagSize {
		return nil, fmt.Errorf("invalid ciphertext: too short")
	}

	ciphertext := ciphertextWithTag[:len(ciphertextWithTag)-TagSize]
	authTag := ciphertextWithTag[len(ciphertextWithTag)-TagSize:]

	return &EncryptedPasswordData{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString(nonce),
		AuthTag:    base64.StdEncoding.EncodeToString(authTag),
	}, nil
}

// DecryptPasswordWithKey decrypts a database password using AES-256-GCM
func DecryptPasswordWithKey(data *EncryptedPasswordData, key []byte) (string, error) {
	// Decode Base64
	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(data.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(data.AuthTag)
	if err != nil {
		return "", fmt.Errorf("failed to decode auth tag: %w", err)
	}

	// Reconstruct ciphertext with auth tag
	ciphertextWithTag := append(ciphertext, authTag...)

	// Use existing DecryptSecret function
	plaintext, err := DecryptSecret(ciphertextWithTag, iv, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return string(plaintext), nil
}

// EncryptMasterKeyWithPassword encrypts a master key using a password-derived key
func EncryptMasterKeyWithPassword(masterKey []byte, userPassword string) (*EncryptedMasterKey, error) {
	// Generate salt for PBKDF2
	salt, err := GenerateRandomBytes(PBKDF2SaltLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from password
	userKey := DeriveKeyFromPassword(userPassword, salt)

	// Encrypt master key (encode as base64 first)
	masterKeyBase64 := base64.StdEncoding.EncodeToString(masterKey)
	encrypted, err := EncryptPasswordWithKey(masterKeyBase64, userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt master key: %w", err)
	}

	return &EncryptedMasterKey{
		Ciphertext: encrypted.Ciphertext,
		IV:         encrypted.IV,
		AuthTag:    encrypted.AuthTag,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Iterations: PBKDF2Iterations,
	}, nil
}

// DecryptMasterKeyWithPassword decrypts a master key using a password-derived key
func DecryptMasterKeyWithPassword(encryptedKey *EncryptedMasterKey, userPassword string) ([]byte, error) {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(encryptedKey.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Derive key from password
	userKey := DeriveKeyFromPassword(userPassword, salt)

	// Decrypt master key
	encryptedData := &EncryptedPasswordData{
		Ciphertext: encryptedKey.Ciphertext,
		IV:         encryptedKey.IV,
		AuthTag:    encryptedKey.AuthTag,
	}

	masterKeyBase64, err := DecryptPasswordWithKey(encryptedData, userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt master key: %w", err)
	}

	// Decode master key from Base64
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	return masterKey, nil
}
