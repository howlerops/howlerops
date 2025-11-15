package crypto

import (
	"encoding/base64"
	"testing"
)

// TestEncryptPasswordWithKey tests basic password encryption functionality
func TestEncryptPasswordWithKey(t *testing.T) {
	// Generate a master key
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	password := "MySecurePassword123!"

	// Encrypt the password
	encrypted, err := EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	// Verify all fields are populated
	if encrypted.Ciphertext == "" {
		t.Error("Ciphertext should not be empty")
	}
	if encrypted.IV == "" {
		t.Error("IV should not be empty")
	}
	if encrypted.AuthTag == "" {
		t.Error("AuthTag should not be empty")
	}

	// Verify fields are valid base64
	if _, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext); err != nil {
		t.Errorf("Ciphertext is not valid base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted.IV); err != nil {
		t.Errorf("IV is not valid base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted.AuthTag); err != nil {
		t.Errorf("AuthTag is not valid base64: %v", err)
	}
}

// TestDecryptPasswordWithKey tests basic password decryption functionality
func TestDecryptPasswordWithKey(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	password := "MySecurePassword123!"

	// Encrypt then decrypt
	encrypted, err := EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	decrypted, err := DecryptPasswordWithKey(encrypted, masterKey)
	if err != nil {
		t.Fatalf("Failed to decrypt password: %v", err)
	}

	// Verify decrypted password matches original
	if decrypted != password {
		t.Errorf("Expected %s, got %s", password, decrypted)
	}
}

// TestEncryptDecryptPasswordRoundTrip tests end-to-end encryption/decryption
func TestEncryptDecryptPasswordRoundTrip(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"complex password", "P@ssw0rd!#$%^&*()"},
		{"long password", "ThisIsAVeryLongPasswordThatContainsManyCharactersAndSymbols!@#$%^&*()123456789"},
		{"unicode password", "パスワード🔐💻"},
		{"empty password", ""},
		{"password with spaces", "my secure password"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			masterKey, err := GenerateMasterKey()
			if err != nil {
				t.Fatalf("Failed to generate master key: %v", err)
			}

			encrypted, err := EncryptPasswordWithKey(tc.password, masterKey)
			if err != nil {
				t.Fatalf("Failed to encrypt password: %v", err)
			}

			decrypted, err := DecryptPasswordWithKey(encrypted, masterKey)
			if err != nil {
				t.Fatalf("Failed to decrypt password: %v", err)
			}

			if decrypted != tc.password {
				t.Errorf("Expected %s, got %s", tc.password, decrypted)
			}
		})
	}
}

// TestDecryptPasswordWithWrongKey tests that decryption fails with wrong key
func TestDecryptPasswordWithWrongKey(t *testing.T) {
	correctKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate correct key: %v", err)
	}

	wrongKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate wrong key: %v", err)
	}

	password := "MySecurePassword123!"

	encrypted, err := EncryptPasswordWithKey(password, correctKey)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = DecryptPasswordWithKey(encrypted, wrongKey)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

// TestEncryptPasswordWithInvalidKey tests error handling for invalid keys
func TestEncryptPasswordWithInvalidKey(t *testing.T) {
	password := "MySecurePassword123!"

	testCases := []struct {
		name string
		key  []byte
	}{
		{"nil key", nil},
		{"short key", []byte("short")},
		{"wrong size key", make([]byte, 16)},
		{"empty key", []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EncryptPasswordWithKey(password, tc.key)
			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}
		})
	}
}

// TestDecryptPasswordWithInvalidKey tests error handling for invalid keys during decryption
func TestDecryptPasswordWithInvalidKey(t *testing.T) {
	// Create valid encrypted data first
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	password := "MySecurePassword123!"
	encrypted, err := EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	testCases := []struct {
		name string
		key  []byte
	}{
		{"nil key", nil},
		{"short key", []byte("short")},
		{"wrong size key", make([]byte, 16)},
		{"empty key", []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptPasswordWithKey(encrypted, tc.key)
			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}
		})
	}
}

// TestDecryptPasswordWithCorruptedData tests error handling for corrupted encrypted data
func TestDecryptPasswordWithCorruptedData(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	password := "MySecurePassword123!"
	encrypted, err := EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	testCases := []struct {
		name string
		data *EncryptedPasswordData
	}{
		{
			"corrupted ciphertext",
			&EncryptedPasswordData{
				Ciphertext: "invalid-base64-!!!",
				IV:         encrypted.IV,
				AuthTag:    encrypted.AuthTag,
			},
		},
		{
			"corrupted IV",
			&EncryptedPasswordData{
				Ciphertext: encrypted.Ciphertext,
				IV:         "invalid-base64-!!!",
				AuthTag:    encrypted.AuthTag,
			},
		},
		{
			"corrupted auth tag",
			&EncryptedPasswordData{
				Ciphertext: encrypted.Ciphertext,
				IV:         encrypted.IV,
				AuthTag:    "invalid-base64-!!!",
			},
		},
		{
			"empty ciphertext",
			&EncryptedPasswordData{
				Ciphertext: "",
				IV:         encrypted.IV,
				AuthTag:    encrypted.AuthTag,
			},
		},
		{
			"empty IV",
			&EncryptedPasswordData{
				Ciphertext: encrypted.Ciphertext,
				IV:         "",
				AuthTag:    encrypted.AuthTag,
			},
		},
		{
			"empty auth tag",
			&EncryptedPasswordData{
				Ciphertext: encrypted.Ciphertext,
				IV:         encrypted.IV,
				AuthTag:    "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptPasswordWithKey(tc.data, masterKey)
			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}
		})
	}
}

// TestGenerateMasterKey tests master key generation
func TestGenerateMasterKey(t *testing.T) {
	// Generate multiple keys to ensure they're different
	keys := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		key, err := GenerateMasterKey()
		if err != nil {
			t.Fatalf("Failed to generate master key %d: %v", i, err)
		}

		// Verify key size
		if len(key) != KeySize {
			t.Errorf("Expected key size %d, got %d", KeySize, len(key))
		}

		// Verify key is not all zeros
		allZeros := true
		for _, b := range key {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if allZeros {
			t.Error("Generated key is all zeros")
		}

		keys[i] = key
	}

	// Verify all keys are different (extremely unlikely to generate duplicates with crypto/rand)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if string(keys[i]) == string(keys[j]) {
				t.Errorf("Keys %d and %d are identical", i, j)
			}
		}
	}
}

// TestDeriveKeyFromPassword tests PBKDF2 key derivation
func TestDeriveKeyFromPassword(t *testing.T) {
	password := "MySecurePassword123!"
	salt := make([]byte, PBKDF2SaltLength)
	for i := range salt {
		salt[i] = byte(i)
	}

	key := DeriveKeyFromPassword(password, salt)

	// Verify key size
	if len(key) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key))
	}

	// Verify key is not all zeros
	allZeros := true
	for _, b := range key {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Derived key is all zeros")
	}

	// Verify same password + salt produces same key (deterministic)
	key2 := DeriveKeyFromPassword(password, salt)
	if string(key) != string(key2) {
		t.Error("Same password and salt should produce same key")
	}

	// Verify different password produces different key
	differentPassword := "DifferentPassword456!"
	key3 := DeriveKeyFromPassword(differentPassword, salt)
	if string(key) == string(key3) {
		t.Error("Different passwords should produce different keys")
	}

	// Verify different salt produces different key
	differentSalt := make([]byte, PBKDF2SaltLength)
	for i := range differentSalt {
		differentSalt[i] = byte(i + 1)
	}
	key4 := DeriveKeyFromPassword(password, differentSalt)
	if string(key) == string(key4) {
		t.Error("Different salts should produce different keys")
	}
}

// TestEncryptMasterKeyWithPassword tests master key encryption with password
func TestEncryptMasterKeyWithPassword(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	userPassword := "MyUserPassword123!"

	encrypted, err := EncryptMasterKeyWithPassword(masterKey, userPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt master key: %v", err)
	}

	// Verify all fields are populated
	if encrypted.Ciphertext == "" {
		t.Error("Ciphertext should not be empty")
	}
	if encrypted.IV == "" {
		t.Error("IV should not be empty")
	}
	if encrypted.AuthTag == "" {
		t.Error("AuthTag should not be empty")
	}
	if encrypted.Salt == "" {
		t.Error("Salt should not be empty")
	}
	if encrypted.Iterations != PBKDF2Iterations {
		t.Errorf("Expected %d iterations, got %d", PBKDF2Iterations, encrypted.Iterations)
	}

	// Verify fields are valid base64
	if _, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext); err != nil {
		t.Errorf("Ciphertext is not valid base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted.IV); err != nil {
		t.Errorf("IV is not valid base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted.AuthTag); err != nil {
		t.Errorf("AuthTag is not valid base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted.Salt); err != nil {
		t.Errorf("Salt is not valid base64: %v", err)
	}
}

// TestDecryptMasterKeyWithPassword tests master key decryption with password
func TestDecryptMasterKeyWithPassword(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	userPassword := "MyUserPassword123!"

	// Encrypt then decrypt
	encrypted, err := EncryptMasterKeyWithPassword(masterKey, userPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt master key: %v", err)
	}

	decrypted, err := DecryptMasterKeyWithPassword(encrypted, userPassword)
	if err != nil {
		t.Fatalf("Failed to decrypt master key: %v", err)
	}

	// Verify decrypted key matches original
	if string(decrypted) != string(masterKey) {
		t.Error("Decrypted master key does not match original")
	}
}

// TestEncryptDecryptMasterKeyRoundTrip tests end-to-end master key encryption/decryption
func TestEncryptDecryptMasterKeyRoundTrip(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"complex password", "P@ssw0rd!#$%^&*()"},
		{"long password", "ThisIsAVeryLongPasswordThatContainsManyCharactersAndSymbols!@#$%^&*()123456789"},
		{"unicode password", "パスワード🔐💻"},
		{"password with spaces", "my secure password"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			masterKey, err := GenerateMasterKey()
			if err != nil {
				t.Fatalf("Failed to generate master key: %v", err)
			}

			encrypted, err := EncryptMasterKeyWithPassword(masterKey, tc.password)
			if err != nil {
				t.Fatalf("Failed to encrypt master key: %v", err)
			}

			decrypted, err := DecryptMasterKeyWithPassword(encrypted, tc.password)
			if err != nil {
				t.Fatalf("Failed to decrypt master key: %v", err)
			}

			if string(decrypted) != string(masterKey) {
				t.Error("Decrypted master key does not match original")
			}
		})
	}
}

// TestDecryptMasterKeyWithWrongPassword tests that decryption fails with wrong password
func TestDecryptMasterKeyWithWrongPassword(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	correctPassword := "MyCorrectPassword123!"
	wrongPassword := "MyWrongPassword456!"

	encrypted, err := EncryptMasterKeyWithPassword(masterKey, correctPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt master key: %v", err)
	}

	// Try to decrypt with wrong password
	_, err = DecryptMasterKeyWithPassword(encrypted, wrongPassword)
	if err == nil {
		t.Error("Expected error when decrypting with wrong password")
	}
}

// TestDecryptMasterKeyWithCorruptedData tests error handling for corrupted encrypted master key
func TestDecryptMasterKeyWithCorruptedData(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	userPassword := "MyUserPassword123!"
	encrypted, err := EncryptMasterKeyWithPassword(masterKey, userPassword)
	if err != nil {
		t.Fatalf("Failed to encrypt master key: %v", err)
	}

	testCases := []struct {
		name string
		data *EncryptedMasterKey
	}{
		{
			"corrupted ciphertext",
			&EncryptedMasterKey{
				Ciphertext: "invalid-base64-!!!",
				IV:         encrypted.IV,
				AuthTag:    encrypted.AuthTag,
				Salt:       encrypted.Salt,
				Iterations: encrypted.Iterations,
			},
		},
		{
			"corrupted salt",
			&EncryptedMasterKey{
				Ciphertext: encrypted.Ciphertext,
				IV:         encrypted.IV,
				AuthTag:    encrypted.AuthTag,
				Salt:       "invalid-base64-!!!",
				Iterations: encrypted.Iterations,
			},
		},
		{
			"empty salt",
			&EncryptedMasterKey{
				Ciphertext: encrypted.Ciphertext,
				IV:         encrypted.IV,
				AuthTag:    encrypted.AuthTag,
				Salt:       "",
				Iterations: encrypted.Iterations,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptMasterKeyWithPassword(tc.data, userPassword)
			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}
		})
	}
}

// TestPBKDF2DeterministicKeyDerivation tests that PBKDF2 is deterministic
func TestPBKDF2DeterministicKeyDerivation(t *testing.T) {
	password := "MySecurePassword123!"
	salt := make([]byte, PBKDF2SaltLength)
	for i := range salt {
		salt[i] = byte(i)
	}

	// Derive key multiple times
	key1 := DeriveKeyFromPassword(password, salt)
	key2 := DeriveKeyFromPassword(password, salt)
	key3 := DeriveKeyFromPassword(password, salt)

	// All keys should be identical
	if string(key1) != string(key2) || string(key1) != string(key3) {
		t.Error("PBKDF2 should be deterministic - same password and salt should produce same key")
	}
}
