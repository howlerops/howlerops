# Secure Credentials Architecture

## Overview

This document describes the security architecture for storing and managing database connection credentials in SQL Studio. The system uses client-side encryption to ensure that sensitive data (passwords, SSH keys) is never stored in plaintext, either locally or in the cloud.

## Security Principles

### 1. Defense in Depth
- **Client-side encryption**: All secrets are encrypted before storage
- **Key derivation**: Strong passphrase-based key derivation using Argon2id
- **Authenticated encryption**: AES-256-GCM provides both confidentiality and integrity
- **Secure key management**: Keys are cached in memory only during active sessions

### 2. Zero-Knowledge Architecture
- **No server-side decryption**: The server never has access to plaintext secrets
- **Client-side key derivation**: Encryption keys are derived locally from user passphrases
- **No key escrow**: There is no way for the system to recover secrets without the user's passphrase

### 3. Forward Secrecy
- **Key rotation support**: The system supports rotating encryption keys
- **Per-secret encryption**: Each secret can be encrypted with different key versions
- **Team key isolation**: Team secrets use separate encryption keys

## Encryption Architecture

### Key Derivation

The system uses **Argon2id** for key derivation with the following parameters:

```go
const (
    Argon2Time    = 1      // 1 iteration (fast unlock)
    Argon2Memory  = 64 * 1024 // 64 MB memory
    Argon2Threads = 4      // 4 threads
    Argon2KeyLen  = 32     // 32 bytes for AES-256
)
```

**Why these parameters:**
- **Time: 1** - Balances security with user experience (< 500ms unlock time)
- **Memory: 64MB** - Reasonable for desktop applications, prevents GPU attacks
- **Threads: 4** - Utilizes modern multi-core processors
- **Key length: 32 bytes** - Required for AES-256

### Encryption Algorithm

**AES-256-GCM** is used for all secret encryption:

- **Key size**: 256 bits (32 bytes)
- **Nonce size**: 96 bits (12 bytes) - randomly generated for each encryption
- **Tag size**: 128 bits (16 bytes) - provides authentication
- **Mode**: Galois/Counter Mode - provides authenticated encryption

### Secret Storage Schema

```sql
CREATE TABLE connection_secrets (
    connection_id TEXT NOT NULL,
    secret_type TEXT NOT NULL,      -- 'db_password', 'ssh_password', 'ssh_private_key'
    ciphertext BLOB NOT NULL,        -- AES-256-GCM encrypted
    nonce BLOB NOT NULL,             -- GCM nonce (96 bits)
    salt BLOB,                       -- Argon2id salt (for key derivation)
    key_version INTEGER DEFAULT 1,   -- Support key rotation
    updated_at INTEGER NOT NULL,
    updated_by TEXT NOT NULL,
    team_id TEXT,                    -- NULL for local-only, set for team-shared
    PRIMARY KEY (connection_id, secret_type),
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
);
```

## Key Management

### User Key Store

The `KeyStore` manages encryption keys in memory:

```go
type KeyStore struct {
    userKey     []byte    // Cached user key
    userKeySalt []byte    // Salt used for key derivation
    teamKeys    map[string][]byte // Team-specific keys
    locked      bool      // Whether keys are loaded
}
```

**Key lifecycle:**
1. **Unlock**: User enters passphrase → Argon2id derives key → Key cached in memory
2. **Use**: Keys used for encryption/decryption operations
3. **Lock**: Keys cleared from memory → User must re-enter passphrase

### Team Key Derivation

For team-shared secrets, a team key is derived from:
- Team secret (shared among team members)
- User's personal key

```go
func DeriveTeamKey(teamSecret []byte, userKey []byte) ([]byte, error) {
    combined := append(teamSecret, userKey...)
    hash := sha256.Sum256(combined)
    return hash[:], nil
}
```

This ensures that:
- Team secrets can only be decrypted by team members
- Each user has a unique team key (prevents key sharing)
- Team admins can rotate team secrets to revoke access

## Secret Types

### Database Passwords
- **Type**: `db_password`
- **Usage**: Database connection authentication
- **Encryption**: User key
- **Storage**: Local SQLite + optional cloud sync

### SSH Passwords
- **Type**: `ssh_password`
- **Usage**: SSH tunnel authentication
- **Encryption**: User key
- **Storage**: Local SQLite + optional cloud sync

### SSH Private Keys
- **Type**: `ssh_private_key`
- **Usage**: SSH tunnel authentication with key-based auth
- **Encryption**: User key
- **Storage**: Local SQLite + optional cloud sync
- **Format**: PEM-encoded private keys (RSA, DSA, EC, Ed25519, OpenSSH)

## Security Features

### 1. Passphrase Validation

The system validates passphrase strength:

```typescript
function validatePassphrase(passphrase: string): {
  valid: boolean;
  score: number;
  feedback: string[];
}
```

**Requirements:**
- Minimum 8 characters
- Mix of uppercase, lowercase, numbers, symbols
- Avoids common patterns (password, 123456, etc.)

### 2. Key Rotation

The system supports key rotation for enhanced security:

- **Key versioning**: Each secret stores its encryption key version
- **Re-encryption**: All secrets can be re-encrypted with new keys
- **Gradual migration**: Old and new keys can coexist during transition

### 3. Audit Trail

All secret operations are logged:

```sql
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    actor_user_id TEXT NOT NULL,
    action TEXT NOT NULL,           -- 'create', 'read', 'update', 'delete'
    resource TEXT NOT NULL,         -- 'connection_secret'
    resource_id TEXT NOT NULL,      -- connection_id
    meta_json TEXT,                 -- Additional metadata
    created_at INTEGER NOT NULL
);
```

### 4. Session Management

- **Automatic lock**: Keys are cleared after inactivity
- **Manual lock**: Users can manually lock the key store
- **Session timeout**: Configurable timeout for key cache

## Threat Model

### Threats Mitigated

1. **Database compromise**: Encrypted secrets cannot be decrypted without user keys
2. **Cloud sync compromise**: Only ciphertext is synced, never plaintext
3. **Memory dumps**: Keys are cleared on lock/exit
4. **Shoulder surfing**: Passphrase input is masked
5. **Keyloggers**: Passphrase is only entered once per session

### Threats Not Mitigated

1. **Compromised client**: If the client machine is compromised, keys in memory can be extracted
2. **Social engineering**: Users can be tricked into revealing passphrases
3. **Weak passphrases**: Users may choose weak passphrases despite validation

## Implementation Details

### Frontend Components

#### PassphrasePrompt
- Modal dialog for passphrase entry
- Visual feedback for passphrase strength
- Secure input with show/hide toggle

#### SecretInput
- Secure password input component
- Masks sensitive data by default
- Supports both input and textarea modes

#### PemKeyUpload
- File upload for SSH private keys
- Client-side PEM validation
- Support for drag-and-drop and paste

### Backend Services

#### SecretStore
- Implements the `crypto.SecretStore` interface
- Handles database operations for encrypted secrets
- Provides migration support for existing data

#### SecretManager
- High-level API for secret operations
- Handles encryption/decryption with proper key management
- Supports both user and team secret operations

#### MigrationManager
- Handles migration of existing plaintext passwords
- Provides rollback capabilities
- Tracks migration status and history

## Best Practices

### For Users

1. **Strong passphrases**: Use long, complex passphrases
2. **Regular rotation**: Change passphrases periodically
3. **Secure storage**: Don't write down passphrases
4. **Lock when away**: Manually lock the key store when stepping away

### For Developers

1. **Never log secrets**: Ensure no plaintext secrets appear in logs
2. **Clear memory**: Explicitly clear sensitive data from memory
3. **Validate input**: Always validate PEM keys and passphrases
4. **Handle errors**: Don't leak information in error messages

### For Administrators

1. **Monitor access**: Review audit logs regularly
2. **Team management**: Rotate team keys when members leave
3. **Backup strategy**: Ensure encrypted backups are available
4. **Incident response**: Have procedures for compromised keys

## Compliance Considerations

### GDPR
- **Data minimization**: Only necessary secrets are stored
- **Right to erasure**: Secrets can be deleted with connection removal
- **Data portability**: Encrypted secrets can be exported

### SOC 2
- **Access controls**: Role-based access to team secrets
- **Audit logging**: All secret operations are logged
- **Encryption**: All sensitive data is encrypted at rest

### HIPAA (if applicable)
- **Encryption**: Meets encryption requirements for PHI
- **Access controls**: Proper authentication and authorization
- **Audit trails**: Complete audit trail of access

## Future Enhancements

### Hardware Security Modules (HSM)
- Integration with hardware security modules
- Key generation and storage in secure hardware
- Enhanced protection against key extraction

### Multi-Factor Authentication
- Integration with TOTP/HOTP authenticators
- Hardware security keys (FIDO2/WebAuthn)
- Biometric authentication for key unlock

### Advanced Key Management
- Hierarchical key derivation
- Time-based key rotation
- Quantum-resistant algorithms (post-quantum cryptography)

## Conclusion

The secure credentials architecture provides enterprise-grade security for database connection credentials while maintaining usability. The zero-knowledge design ensures that even if the system is compromised, user secrets remain protected. The modular design allows for future enhancements while maintaining backward compatibility.
