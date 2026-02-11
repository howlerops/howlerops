package crypto

import (
	"bytes"
	"sync"
	"testing"
)

func TestGenerateOrgEnvelopeKey(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "Generate valid OEK"},
		{name: "Generate second OEK (should differ)"},
	}

	var firstKey []byte

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oek, err := GenerateOrgEnvelopeKey()
			if err != nil {
				t.Fatalf("GenerateOrgEnvelopeKey() error = %v", err)
			}

			// Verify key size
			if len(oek) != KeySize {
				t.Errorf("expected key size %d, got %d", KeySize, len(oek))
			}

			// Verify key is not all zeros
			allZeros := true
			for _, b := range oek {
				if b != 0 {
					allZeros = false
					break
				}
			}
			if allZeros {
				t.Error("generated key is all zeros")
			}

			// Verify uniqueness
			if i == 0 {
				firstKey = oek
			} else {
				if bytes.Equal(oek, firstKey) {
					t.Error("generated keys should be unique")
				}
			}
		})
	}
}

func TestEncryptDecryptWithOEK(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "Simple string",
			plaintext: []byte("Hello, World!"),
			wantErr:   false,
		},
		{
			name:      "Empty plaintext",
			plaintext: []byte(""),
			wantErr:   false,
		},
		{
			name:      "Large plaintext",
			plaintext: bytes.Repeat([]byte("A"), 10000),
			wantErr:   false,
		},
		{
			name:      "Binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate OEK
			oek, err := GenerateOrgEnvelopeKey()
			if err != nil {
				t.Fatalf("GenerateOrgEnvelopeKey() error = %v", err)
			}
			defer ClearBytes(oek)

			// Encrypt
			encrypted, err := EncryptWithOEK(tt.plaintext, oek)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptWithOEK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify structure (ciphertext can be empty for empty plaintext, but IV and AuthTag must exist)
			if encrypted.IV == "" || encrypted.AuthTag == "" {
				t.Error("encrypted data has empty IV or AuthTag")
			}

			// Decrypt
			decrypted, err := DecryptWithOEK(encrypted, oek)
			if err != nil {
				t.Errorf("DecryptWithOEK() error = %v", err)
				return
			}

			// Verify plaintext matches
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("decrypted plaintext doesn't match original\ngot:  %v\nwant: %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDecryptOEKForUser(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Encrypt and decrypt OEK for user",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate OEK
			oek, err := GenerateOrgEnvelopeKey()
			if err != nil {
				t.Fatalf("GenerateOrgEnvelopeKey() error = %v", err)
			}
			defer ClearBytes(oek)

			// Generate user master key
			userMasterKey, err := GenerateMasterKey()
			if err != nil {
				t.Fatalf("GenerateMasterKey() error = %v", err)
			}
			defer ClearBytes(userMasterKey)

			// Encrypt OEK for user
			encrypted, err := EncryptOEKForUser(oek, userMasterKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptOEKForUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify structure
			if encrypted.Ciphertext == "" || encrypted.IV == "" || encrypted.AuthTag == "" {
				t.Error("encrypted OEK has empty fields")
			}

			// Decrypt OEK
			decryptedOEK, err := DecryptOEKWithMasterKey(encrypted, userMasterKey)
			if err != nil {
				t.Errorf("DecryptOEKWithMasterKey() error = %v", err)
				return
			}
			defer ClearBytes(decryptedOEK)

			// Verify OEK matches
			if !bytes.Equal(decryptedOEK, oek) {
				t.Errorf("decrypted OEK doesn't match original")
			}

			// Verify decrypted OEK can decrypt data
			testData := []byte("Test secret data")
			encData, err := EncryptWithOEK(testData, oek)
			if err != nil {
				t.Fatalf("EncryptWithOEK() error = %v", err)
			}

			decData, err := DecryptWithOEK(encData, decryptedOEK)
			if err != nil {
				t.Fatalf("DecryptWithOEK() error = %v", err)
			}

			if !bytes.Equal(decData, testData) {
				t.Error("decrypted data doesn't match using recovered OEK")
			}
		})
	}
}

func TestWrongKeyFails(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Wrong OEK for decryption",
			testFunc: func(t *testing.T) {
				oek1, _ := GenerateOrgEnvelopeKey()
				defer ClearBytes(oek1)
				oek2, _ := GenerateOrgEnvelopeKey()
				defer ClearBytes(oek2)

				plaintext := []byte("Secret message")
				encrypted, err := EncryptWithOEK(plaintext, oek1)
				if err != nil {
					t.Fatalf("EncryptWithOEK() error = %v", err)
				}

				// Try to decrypt with wrong key
				_, err = DecryptWithOEK(encrypted, oek2)
				if err == nil {
					t.Error("DecryptWithOEK() should fail with wrong key")
				}
			},
		},
		{
			name: "Wrong master key for OEK decryption",
			testFunc: func(t *testing.T) {
				oek, _ := GenerateOrgEnvelopeKey()
				defer ClearBytes(oek)
				masterKey1, _ := GenerateMasterKey()
				defer ClearBytes(masterKey1)
				masterKey2, _ := GenerateMasterKey()
				defer ClearBytes(masterKey2)

				encrypted, err := EncryptOEKForUser(oek, masterKey1)
				if err != nil {
					t.Fatalf("EncryptOEKForUser() error = %v", err)
				}

				// Try to decrypt with wrong master key
				_, err = DecryptOEKWithMasterKey(encrypted, masterKey2)
				if err == nil {
					t.Error("DecryptOEKWithMasterKey() should fail with wrong key")
				}
			},
		},
		{
			name: "Tampered ciphertext",
			testFunc: func(t *testing.T) {
				oek, _ := GenerateOrgEnvelopeKey()
				defer ClearBytes(oek)

				plaintext := []byte("Secret message")
				encrypted, err := EncryptWithOEK(plaintext, oek)
				if err != nil {
					t.Fatalf("EncryptWithOEK() error = %v", err)
				}

				// Tamper with ciphertext (flip a bit)
				tamperedCiphertext := encrypted.Ciphertext
				if len(tamperedCiphertext) > 0 {
					// Modify last character
					encrypted.Ciphertext = tamperedCiphertext[:len(tamperedCiphertext)-1] + "X"
				}

				// Should fail due to authentication
				_, err = DecryptWithOEK(encrypted, oek)
				if err == nil {
					t.Error("DecryptWithOEK() should fail with tampered ciphertext")
				}
			},
		},
		{
			name: "Invalid key size",
			testFunc: func(t *testing.T) {
				invalidKey := []byte("short key")
				plaintext := []byte("test")

				_, err := EncryptWithOEK(plaintext, invalidKey)
				if err == nil {
					t.Error("EncryptWithOEK() should fail with invalid key size")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func TestConcurrentEncryption(t *testing.T) {
	// Generate shared OEK
	oek, err := GenerateOrgEnvelopeKey()
	if err != nil {
		t.Fatalf("GenerateOrgEnvelopeKey() error = %v", err)
	}
	defer ClearBytes(oek)

	// Number of concurrent operations
	concurrency := 100

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	// Test concurrent encryption/decryption
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			plaintext := []byte("Test message " + string(rune(id)))

			// Encrypt
			encrypted, err := EncryptWithOEK(plaintext, oek)
			if err != nil {
				errors <- err
				return
			}

			// Decrypt
			decrypted, err := DecryptWithOEK(encrypted, oek)
			if err != nil {
				errors <- err
				return
			}

			// Verify
			if !bytes.Equal(decrypted, plaintext) {
				errors <- err
				return
			}
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

func TestConcurrentOEKEncryption(t *testing.T) {
	// Generate master keys for multiple users
	numUsers := 50
	userKeys := make([][]byte, numUsers)
	for i := 0; i < numUsers; i++ {
		key, err := GenerateMasterKey()
		if err != nil {
			t.Fatalf("GenerateMasterKey() error = %v", err)
		}
		userKeys[i] = key
		defer ClearBytes(userKeys[i])
	}

	// Generate OEK
	oek, err := GenerateOrgEnvelopeKey()
	if err != nil {
		t.Fatalf("GenerateOrgEnvelopeKey() error = %v", err)
	}
	defer ClearBytes(oek)

	var wg sync.WaitGroup
	errors := make(chan error, numUsers)

	// Encrypt OEK for all users concurrently
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			// Encrypt OEK for this user
			encrypted, err := EncryptOEKForUser(oek, userKeys[userID])
			if err != nil {
				errors <- err
				return
			}

			// Decrypt and verify
			decryptedOEK, err := DecryptOEKWithMasterKey(encrypted, userKeys[userID])
			if err != nil {
				errors <- err
				return
			}
			defer ClearBytes(decryptedOEK)

			if !bytes.Equal(decryptedOEK, oek) {
				errors <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent OEK encryption failed: %v", err)
	}
}

func TestClearBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Clear non-empty slice",
			input: []byte{1, 2, 3, 4, 5},
		},
		{
			name:  "Clear empty slice",
			input: []byte{},
		},
		{
			name:  "Clear nil slice",
			input: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to verify clearing
			original := make([]byte, len(tt.input))
			copy(original, tt.input)

			ClearBytes(tt.input)

			// Verify all bytes are zero (except for nil)
			if tt.input != nil {
				for i, b := range tt.input {
					if b != 0 {
						t.Errorf("byte at index %d not cleared: got %d", i, b)
					}
				}
			}
		})
	}
}

func BenchmarkEncryptWithOEK(b *testing.B) {
	oek, _ := GenerateOrgEnvelopeKey()
	defer ClearBytes(oek)
	plaintext := []byte("Test secret data for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptWithOEK(plaintext, oek)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptWithOEK(b *testing.B) {
	oek, _ := GenerateOrgEnvelopeKey()
	defer ClearBytes(oek)
	plaintext := []byte("Test secret data for benchmarking")
	encrypted, _ := EncryptWithOEK(plaintext, oek)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptWithOEK(encrypted, oek)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncryptOEKForUser(b *testing.B) {
	oek, _ := GenerateOrgEnvelopeKey()
	defer ClearBytes(oek)
	userKey, _ := GenerateMasterKey()
	defer ClearBytes(userKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptOEKForUser(oek, userKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentEncryption(b *testing.B) {
	oek, _ := GenerateOrgEnvelopeKey()
	defer ClearBytes(oek)
	plaintext := []byte("Test secret data for concurrent benchmarking")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			encrypted, err := EncryptWithOEK(plaintext, oek)
			if err != nil {
				b.Fatal(err)
			}
			_, err = DecryptWithOEK(encrypted, oek)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
