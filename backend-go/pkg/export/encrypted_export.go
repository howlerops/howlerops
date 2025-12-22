package export

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

// Cryptographic parameters (OWASP 2024 recommendations)
const (
	// Argon2id parameters - memory-hard to resist GPU attacks
	Argon2Memory      = 64 * 1024 // 64 MB
	Argon2Iterations  = 3
	Argon2Parallelism = 4
	Argon2KeyLength   = 32 // 256 bits for AES-256

	// Salt and nonce sizes
	SaltSize  = 16 // 128 bits
	NonceSize = 12 // 96 bits for AES-GCM

	// Minimum passphrase length
	MinPassphraseLength = 12

	// Export format identifier
	EncryptedExportFormat = "howlerops-encrypted-config-v1"
)

// Errors
var (
	ErrPassphraseTooShort   = errors.New("passphrase must be at least 12 characters")
	ErrPassphraseTooWeak    = errors.New("passphrase is too weak - use a mix of characters")
	ErrInvalidEncryptedData = errors.New("invalid encrypted data format")
	ErrDecryptionFailed     = errors.New("decryption failed - wrong passphrase or corrupted data")
	ErrUnsupportedAlgorithm = errors.New("unsupported encryption algorithm")
)

// EncryptedConfigExport is the structure for password-protected exports
type EncryptedConfigExport struct {
	// Format version for compatibility
	Format string `json:"format"`
	// Encryption algorithm identifier
	Algorithm string `json:"algorithm"`
	// Base64-encoded salt for key derivation
	Salt string `json:"salt"`
	// Base64-encoded nonce for AES-GCM
	Nonce string `json:"nonce"`
	// Base64-encoded encrypted payload
	Ciphertext string `json:"ciphertext"`
	// Export timestamp
	CreatedAt time.Time `json:"created_at"`
	// Export ID for audit correlation
	ExportID string `json:"export_id"`
	// Hint about contents (no sensitive data)
	Hint EncryptedExportHint `json:"hint"`
}

// EncryptedExportHint provides non-sensitive metadata about the export
type EncryptedExportHint struct {
	ConnectionCount int      `json:"connection_count"`
	QueryCount      int      `json:"query_count"`
	ExportedBy      string   `json:"exported_by,omitempty"`
	DatabaseTypes   []string `json:"database_types,omitempty"`
}

// ExportedCredential represents a connection WITH its password for encrypted export
type ExportedCredential struct {
	ExportedConnection
	// Password (only present in encrypted exports)
	Password string `json:"password,omitempty"`
}

// FullExportPayload is the decrypted content of an encrypted export
type FullExportPayload struct {
	// Credentials with passwords
	Credentials []ExportedCredential `json:"credentials"`
	// Saved queries (same as regular export)
	SavedQueries []ExportedSavedQuery `json:"saved_queries,omitempty"`
	// Tags and folders
	Tags    []string `json:"tags,omitempty"`
	Folders []string `json:"folders,omitempty"`
	// Original export metadata
	ExportedAt time.Time `json:"exported_at"`
	ExportedBy string    `json:"exported_by,omitempty"`
	AppVersion string    `json:"app_version,omitempty"`
}

// EncryptExport encrypts an export payload with a user-provided passphrase
func EncryptExport(payload *FullExportPayload, passphrase string) (*EncryptedConfigExport, error) {
	// Validate passphrase strength
	if err := validatePassphrase(passphrase); err != nil {
		return nil, err
	}

	// Generate cryptographic parameters
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Derive encryption key using Argon2id
	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		Argon2Iterations,
		Argon2Memory,
		Argon2Parallelism,
		Argon2KeyLength,
	)
	defer clearBytes(key)

	// Serialize payload to JSON
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}
	defer clearBytes(plaintext)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt with authenticated encryption
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Generate export ID
	exportIDBytes := make([]byte, 8)
	rand.Read(exportIDBytes)
	exportID := base64.URLEncoding.EncodeToString(exportIDBytes)

	// Build hint (non-sensitive metadata)
	dbTypes := make(map[string]bool)
	for _, cred := range payload.Credentials {
		dbTypes[cred.Type] = true
	}
	dbTypeList := make([]string, 0, len(dbTypes))
	for t := range dbTypes {
		dbTypeList = append(dbTypeList, t)
	}

	return &EncryptedConfigExport{
		Format:     EncryptedExportFormat,
		Algorithm:  "argon2id-aes256gcm",
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		CreatedAt:  time.Now().UTC(),
		ExportID:   exportID,
		Hint: EncryptedExportHint{
			ConnectionCount: len(payload.Credentials),
			QueryCount:      len(payload.SavedQueries),
			ExportedBy:      payload.ExportedBy,
			DatabaseTypes:   dbTypeList,
		},
	}, nil
}

// DecryptExport decrypts an encrypted export with the provided passphrase
func DecryptExport(encrypted *EncryptedConfigExport, passphrase string) (*FullExportPayload, error) {
	// Validate format
	if encrypted.Format != EncryptedExportFormat {
		return nil, fmt.Errorf("%w: %s", ErrInvalidEncryptedData, encrypted.Format)
	}

	if encrypted.Algorithm != "argon2id-aes256gcm" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, encrypted.Algorithm)
	}

	// Decode base64 values
	salt, err := base64.StdEncoding.DecodeString(encrypted.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	// Derive key using same Argon2id parameters
	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		Argon2Iterations,
		Argon2Memory,
		Argon2Parallelism,
		Argon2KeyLength,
	)
	defer clearBytes(key)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and authenticate
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	defer clearBytes(plaintext)

	// Parse JSON payload
	var payload FullExportPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted data: %w", err)
	}

	return &payload, nil
}

// validatePassphrase checks passphrase strength
func validatePassphrase(passphrase string) error {
	if len(passphrase) < MinPassphraseLength {
		return ErrPassphraseTooShort
	}

	// Check for basic complexity (at least 2 of: uppercase, lowercase, digit, special)
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range passphrase {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	complexity := 0
	if hasUpper {
		complexity++
	}
	if hasLower {
		complexity++
	}
	if hasDigit {
		complexity++
	}
	if hasSpecial {
		complexity++
	}

	if complexity < 2 {
		return ErrPassphraseTooWeak
	}

	return nil
}

// clearBytes overwrites a byte slice with zeros for security
func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ToJSON serializes the encrypted export to JSON
func (e *EncryptedConfigExport) ToJSON(pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(e, "", "  ")
	}
	return json.Marshal(e)
}

// ParseEncryptedExport parses an encrypted export from JSON
func ParseEncryptedExport(data []byte) (*EncryptedConfigExport, error) {
	var export EncryptedConfigExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("invalid encrypted export format: %w", err)
	}

	if export.Format != EncryptedExportFormat {
		return nil, fmt.Errorf("unsupported format: %s (expected %s)", export.Format, EncryptedExportFormat)
	}

	return &export, nil
}
