package export

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/pkg/storage"
)

func TestEncryptDecryptExport(t *testing.T) {
	payload := &FullExportPayload{
		Credentials: []ExportedCredential{
			{
				ExportedConnection: ExportedConnection{
					ExportID: "conn_12345678",
					Name:     "Production DB",
					Type:     "postgres",
					Host:     "prod.example.com",
					Port:     5432,
					Database: "myapp",
					Username: "admin",
				},
				Password: "super-secret-password-123",
			},
			{
				ExportedConnection: ExportedConnection{
					ExportID: "conn_87654321",
					Name:     "Staging DB",
					Type:     "mysql",
					Host:     "staging.example.com",
					Port:     3306,
					Database: "myapp_staging",
					Username: "stageuser",
				},
				Password: "another-secret-pass!@#",
			},
		},
		SavedQueries: []ExportedSavedQuery{
			{
				ExportID: "query_11111111",
				Name:     "Get Active Users",
				Query:    "SELECT * FROM users WHERE active = true",
				Tags:     []string{"users", "reporting"},
			},
		},
		Tags:       []string{"users", "reporting"},
		Folders:    []string{"analytics"},
		ExportedAt: time.Now().UTC(),
		ExportedBy: "test@example.com",
		AppVersion: "1.0.0-test",
	}

	t.Run("encrypt and decrypt with correct passphrase", func(t *testing.T) {
		passphrase := "MySecurePassphrase123!"

		// Encrypt
		encrypted, err := EncryptExport(payload, passphrase)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.Equal(t, EncryptedExportFormat, encrypted.Format)
		assert.Equal(t, "argon2id-aes256gcm", encrypted.Algorithm)
		assert.NotEmpty(t, encrypted.Salt)
		assert.NotEmpty(t, encrypted.Nonce)
		assert.NotEmpty(t, encrypted.Ciphertext)
		assert.NotEmpty(t, encrypted.ExportID)

		// Check hint (non-sensitive metadata)
		assert.Equal(t, 2, encrypted.Hint.ConnectionCount)
		assert.Equal(t, 1, encrypted.Hint.QueryCount)
		assert.Equal(t, "test@example.com", encrypted.Hint.ExportedBy)
		assert.Contains(t, encrypted.Hint.DatabaseTypes, "postgres")
		assert.Contains(t, encrypted.Hint.DatabaseTypes, "mysql")

		// Decrypt
		decrypted, err := DecryptExport(encrypted, passphrase)
		require.NoError(t, err)
		assert.NotNil(t, decrypted)

		// Verify credentials with passwords
		assert.Len(t, decrypted.Credentials, 2)
		assert.Equal(t, "super-secret-password-123", decrypted.Credentials[0].Password)
		assert.Equal(t, "another-secret-pass!@#", decrypted.Credentials[1].Password)

		// Verify queries
		assert.Len(t, decrypted.SavedQueries, 1)
		assert.Equal(t, "Get Active Users", decrypted.SavedQueries[0].Name)

		// Verify metadata
		assert.Equal(t, "test@example.com", decrypted.ExportedBy)
	})

	t.Run("decrypt with wrong passphrase fails", func(t *testing.T) {
		passphrase := "CorrectPassphrase123!"
		wrongPassphrase := "WrongPassphrase456!"

		encrypted, err := EncryptExport(payload, passphrase)
		require.NoError(t, err)

		_, err = DecryptExport(encrypted, wrongPassphrase)
		assert.Error(t, err)
		assert.Equal(t, ErrDecryptionFailed, err)
	})

	t.Run("passphrase too short", func(t *testing.T) {
		shortPassphrase := "short"

		_, err := EncryptExport(payload, shortPassphrase)
		assert.Error(t, err)
		assert.Equal(t, ErrPassphraseTooShort, err)
	})

	t.Run("passphrase too weak", func(t *testing.T) {
		// All lowercase, no numbers or special chars
		weakPassphrase := "allllowercase"

		_, err := EncryptExport(payload, weakPassphrase)
		assert.Error(t, err)
		assert.Equal(t, ErrPassphraseTooWeak, err)
	})

	t.Run("different encryptions produce different ciphertext", func(t *testing.T) {
		passphrase := "SamePassphrase123!"

		encrypted1, err := EncryptExport(payload, passphrase)
		require.NoError(t, err)

		encrypted2, err := EncryptExport(payload, passphrase)
		require.NoError(t, err)

		// Salt should be different
		assert.NotEqual(t, encrypted1.Salt, encrypted2.Salt)
		// Nonce should be different
		assert.NotEqual(t, encrypted1.Nonce, encrypted2.Nonce)
		// Ciphertext should be different (due to different salt/nonce)
		assert.NotEqual(t, encrypted1.Ciphertext, encrypted2.Ciphertext)

		// But both should decrypt correctly
		decrypted1, err := DecryptExport(encrypted1, passphrase)
		require.NoError(t, err)
		decrypted2, err := DecryptExport(encrypted2, passphrase)
		require.NoError(t, err)

		assert.Equal(t, decrypted1.Credentials[0].Password, decrypted2.Credentials[0].Password)
	})
}

func TestEncryptedExportSerialization(t *testing.T) {
	payload := &FullExportPayload{
		Credentials: []ExportedCredential{
			{
				ExportedConnection: ExportedConnection{
					ExportID: "conn_12345678",
					Name:     "Test DB",
					Type:     "postgres",
				},
				Password: "test-password",
			},
		},
		ExportedAt: time.Now().UTC(),
	}

	passphrase := "TestPassphrase123!"

	t.Run("serialize and parse encrypted export", func(t *testing.T) {
		encrypted, err := EncryptExport(payload, passphrase)
		require.NoError(t, err)

		// Serialize to JSON
		jsonData, err := encrypted.ToJSON(true)
		require.NoError(t, err)

		// Verify password is NOT in plaintext in JSON
		assert.NotContains(t, string(jsonData), "test-password")

		// Parse back
		parsed, err := ParseEncryptedExport(jsonData)
		require.NoError(t, err)
		assert.Equal(t, encrypted.Format, parsed.Format)
		assert.Equal(t, encrypted.Salt, parsed.Salt)
		assert.Equal(t, encrypted.Ciphertext, parsed.Ciphertext)

		// Decrypt the parsed export
		decrypted, err := DecryptExport(parsed, passphrase)
		require.NoError(t, err)
		assert.Equal(t, "test-password", decrypted.Credentials[0].Password)
	})
}

func TestConfigExporterWithPasswords(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	exporter := NewConfigExporter(logger, "1.0.0-test")

	connections := []storage.Connection{
		{
			ID:          "conn-12345678",
			Name:        "Production DB",
			Type:        "postgres",
			Host:        "prod.example.com",
			Port:        5432,
			Database:    "myapp",
			Username:    "admin",
			Password:    "prod-secret-password",
			Environment: "production",
		},
		{
			ID:       "conn-87654321",
			Name:     "Dev DB",
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "myapp_dev",
			Username: "dev",
			Password: "dev-password",
		},
	}

	queries := []storage.SavedQuery{
		{
			ID:           "query-11111111",
			Name:         "User Count",
			Query:        "SELECT COUNT(*) FROM users",
			ConnectionID: "conn-12345678",
			Tags:         []string{"metrics"},
		},
	}

	t.Run("export with passwords", func(t *testing.T) {
		options := DefaultConfigExportOptions()
		passphrase := "SecureExportPass123!"

		encrypted, err := exporter.ExportWithPasswords(
			context.Background(),
			"test@example.com",
			connections,
			queries,
			options,
			passphrase,
		)
		require.NoError(t, err)
		assert.NotNil(t, encrypted)
		assert.Equal(t, 2, encrypted.Hint.ConnectionCount)
		assert.Equal(t, 1, encrypted.Hint.QueryCount)

		// Decrypt and verify passwords are included
		decrypted, err := DecryptExport(encrypted, passphrase)
		require.NoError(t, err)
		assert.Len(t, decrypted.Credentials, 2)

		// Find production credential and verify password
		var prodCred *ExportedCredential
		for _, c := range decrypted.Credentials {
			if c.Name == "Production DB" {
				prodCred = &c
				break
			}
		}
		require.NotNil(t, prodCred)
		assert.Equal(t, "prod-secret-password", prodCred.Password)
	})

	t.Run("export with passwords respects filters", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections:  true,
			IncludeSavedQueries: true,
			ConnectionIDs:       []string{"conn-12345678"}, // Only prod
		}
		passphrase := "SecureExportPass123!"

		encrypted, err := exporter.ExportWithPasswords(
			context.Background(),
			"test@example.com",
			connections,
			queries,
			options,
			passphrase,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, encrypted.Hint.ConnectionCount) // Only 1 due to filter

		decrypted, err := DecryptExport(encrypted, passphrase)
		require.NoError(t, err)
		assert.Len(t, decrypted.Credentials, 1)
		assert.Equal(t, "Production DB", decrypted.Credentials[0].Name)
	})

	t.Run("export with passwords and anonymized hosts", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections: true,
			AnonymizeHosts:     true,
		}
		passphrase := "SecureExportPass123!"

		encrypted, err := exporter.ExportWithPasswords(
			context.Background(),
			"test@example.com",
			connections,
			queries,
			options,
			passphrase,
		)
		require.NoError(t, err)

		decrypted, err := DecryptExport(encrypted, passphrase)
		require.NoError(t, err)

		// Hosts should be placeholders
		for _, cred := range decrypted.Credentials {
			assert.Contains(t, cred.Host, "{{")
			assert.Contains(t, cred.Host, "}}")
		}

		// But passwords should still be real
		assert.NotEmpty(t, decrypted.Credentials[0].Password)
	})
}

func TestValidatePassphrase(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		wantErr    error
	}{
		{"valid passphrase", "MySecure123Pass!", nil},
		{"too short", "short", ErrPassphraseTooShort},
		{"too short 11 chars", "12345678901", ErrPassphraseTooShort},
		{"min length 12 chars", "123456789012", ErrPassphraseTooWeak}, // 12 chars but weak
		{"all lowercase", "alllowercase", ErrPassphraseTooWeak},
		{"all uppercase", "ALLUPPERCASE", ErrPassphraseTooWeak},
		{"all numbers", "123456789012", ErrPassphraseTooWeak},
		{"lowercase with number", "lowercase123", nil},  // 2 types
		{"uppercase with special", "UPPERCASE!!!", nil}, // 2 types
		{"mixed case", "MixedCasePass", nil},            // 2 types
		{"complex passphrase", "MyP@ssw0rd!23", nil},    // All types
		{"unicode passphrase", "Пароль123!test", nil},   // Works with unicode
		{"spaces allowed", "Has Spaces 123!", nil},      // Spaces count as special
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassphrase(tt.passphrase)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseEncryptedExportErrors(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseEncryptedExport([]byte("not json"))
		assert.Error(t, err)
	})

	t.Run("wrong format version", func(t *testing.T) {
		wrongFormat := `{"format": "wrong-format-v1"}`
		_, err := ParseEncryptedExport([]byte(wrongFormat))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})
}

func TestDecryptExportErrors(t *testing.T) {
	t.Run("unsupported algorithm", func(t *testing.T) {
		encrypted := &EncryptedConfigExport{
			Format:    EncryptedExportFormat,
			Algorithm: "unknown-algorithm",
		}
		_, err := DecryptExport(encrypted, "passphrase")
		assert.ErrorIs(t, err, ErrUnsupportedAlgorithm)
	})

	t.Run("invalid base64 salt", func(t *testing.T) {
		encrypted := &EncryptedConfigExport{
			Format:    EncryptedExportFormat,
			Algorithm: "argon2id-aes256gcm",
			Salt:      "not-valid-base64!!!",
		}
		_, err := DecryptExport(encrypted, "passphrase")
		assert.Error(t, err)
	})
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	payload := &FullExportPayload{
		Credentials: make([]ExportedCredential, 10),
		ExportedAt:  time.Now().UTC(),
	}
	for i := 0; i < 10; i++ {
		payload.Credentials[i] = ExportedCredential{
			ExportedConnection: ExportedConnection{
				ExportID: "conn_12345678",
				Name:     "Test DB",
				Type:     "postgres",
			},
			Password: "benchmark-password-12345",
		}
	}

	passphrase := "BenchmarkPass123!"

	b.Run("encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := EncryptExport(payload, passphrase)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	encrypted, _ := EncryptExport(payload, passphrase)
	b.Run("decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := DecryptExport(encrypted, passphrase)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
