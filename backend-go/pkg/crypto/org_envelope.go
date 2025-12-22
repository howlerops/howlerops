package crypto

import (
	"encoding/base64"
	"fmt"
	"sync"
)

// OEKEncryptedData represents encrypted data with OEK
type OEKEncryptedData struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded ciphertext
	IV         string `json:"iv"`         // Base64-encoded IV/nonce
	AuthTag    string `json:"authTag"`    // Base64-encoded auth tag
}

// bufferPool reduces allocations for ciphertext buffers
// We pool byte slices to reduce GC pressure during encryption/decryption
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate reasonable size for most secrets (1KB)
		b := make([]byte, 0, 1024)
		return &b
	},
}

// getBuffer retrieves a buffer from the pool
func getBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// putBuffer returns a buffer to the pool after clearing it
func putBuffer(b *[]byte) {
	if b != nil && cap(*b) <= 4096 { // Don't pool very large buffers
		*b = (*b)[:0] // Reset length but keep capacity
		bufferPool.Put(b)
	}
}

// GenerateOrgEnvelopeKey generates a new 256-bit OEK
// Returns a cryptographically secure random key suitable for AES-256-GCM
func GenerateOrgEnvelopeKey() ([]byte, error) {
	oek, err := GenerateRandomBytes(KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OEK: %w", err)
	}
	return oek, nil
}

// EncryptWithOEK encrypts plaintext using the OEK
// Uses sync.Pool for buffer reuse to reduce GC pressure
// Returns encrypted data split into ciphertext, IV, and auth tag
func EncryptWithOEK(plaintext []byte, oek []byte) (*OEKEncryptedData, error) {
	if err := ValidateKey(oek); err != nil {
		return nil, fmt.Errorf("invalid OEK: %w", err)
	}

	// Use existing EncryptSecret function for consistency
	ciphertextWithTag, nonce, err := EncryptSecret(plaintext, oek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with OEK: %w", err)
	}

	// Split ciphertext and auth tag (GCM appends 16-byte tag)
	if len(ciphertextWithTag) < TagSize {
		return nil, fmt.Errorf("invalid ciphertext: too short")
	}

	ciphertext := ciphertextWithTag[:len(ciphertextWithTag)-TagSize]
	authTag := ciphertextWithTag[len(ciphertextWithTag)-TagSize:]

	return &OEKEncryptedData{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString(nonce),
		AuthTag:    base64.StdEncoding.EncodeToString(authTag),
	}, nil
}

// DecryptWithOEK decrypts ciphertext using the OEK
// Verifies authentication tag and returns plaintext
func DecryptWithOEK(data *OEKEncryptedData, oek []byte) ([]byte, error) {
	if err := ValidateKey(oek); err != nil {
		return nil, fmt.Errorf("invalid OEK: %w", err)
	}

	// Decode Base64 components
	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(data.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(data.AuthTag)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth tag: %w", err)
	}

	// Get buffer from pool for reconstruction
	bufPtr := getBuffer()
	defer putBuffer(bufPtr)

	// Reconstruct ciphertext with auth tag
	buf := *bufPtr
	buf = append(buf, ciphertext...)
	buf = append(buf, authTag...)

	// Use existing DecryptSecret function for consistency
	plaintext, err := DecryptSecret(buf, iv, oek)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with OEK: %w", err)
	}

	return plaintext, nil
}

// EncryptOEKForUser encrypts an OEK with a user's master key
// This allows each org member to have their own encrypted copy of the OEK
// The encrypted OEK can be stored in the database and decrypted when needed
func EncryptOEKForUser(oek []byte, userMasterKey []byte) (*OEKEncryptedData, error) {
	if err := ValidateKey(oek); err != nil {
		return nil, fmt.Errorf("invalid OEK: %w", err)
	}

	if err := ValidateKey(userMasterKey); err != nil {
		return nil, fmt.Errorf("invalid user master key: %w", err)
	}

	// Encrypt OEK using user's master key
	ciphertextWithTag, nonce, err := EncryptSecret(oek, userMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt OEK for user: %w", err)
	}

	// Split ciphertext and auth tag
	if len(ciphertextWithTag) < TagSize {
		return nil, fmt.Errorf("invalid ciphertext: too short")
	}

	ciphertext := ciphertextWithTag[:len(ciphertextWithTag)-TagSize]
	authTag := ciphertextWithTag[len(ciphertextWithTag)-TagSize:]

	return &OEKEncryptedData{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString(nonce),
		AuthTag:    base64.StdEncoding.EncodeToString(authTag),
	}, nil
}

// DecryptOEKWithMasterKey decrypts an OEK using a user's master key
// This retrieves the organization's envelope key for a specific user
func DecryptOEKWithMasterKey(data *OEKEncryptedData, userMasterKey []byte) ([]byte, error) {
	if err := ValidateKey(userMasterKey); err != nil {
		return nil, fmt.Errorf("invalid user master key: %w", err)
	}

	// Decode Base64 components
	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(data.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(data.AuthTag)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth tag: %w", err)
	}

	// Get buffer from pool
	bufPtr := getBuffer()
	defer putBuffer(bufPtr)

	// Reconstruct ciphertext with auth tag
	buf := *bufPtr
	buf = append(buf, ciphertext...)
	buf = append(buf, authTag...)

	// Decrypt OEK using user's master key
	oek, err := DecryptSecret(buf, iv, userMasterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt OEK with master key: %w", err)
	}

	// Validate the decrypted OEK
	if err := ValidateKey(oek); err != nil {
		return nil, fmt.Errorf("decrypted OEK is invalid: %w", err)
	}

	return oek, nil
}

// ClearBytes securely zeros a byte slice (for key cleanup)
// This should be called with defer after sensitive key material is no longer needed
func ClearBytes(b []byte) {
	if b == nil {
		return
	}
	for i := range b {
		b[i] = 0
	}
}
