# Secure Password Storage Design

## 🔐 Ultra-Deep Security Analysis

### Requirements
1. ✅ Store database passwords in Turso (cloud sync)
2. ✅ Access from multiple devices
3. ✅ Future team sharing capability
4. ✅ Top-notch encryption (industry standard)
5. ✅ Zero-knowledge architecture (server never sees plaintext)

---

## 🎯 Security Architecture: Master Key System

### Why Not Simple User-Password Encryption?

❌ **Bad Approach:**
```
Password → Encrypt with user's login password → Store in cloud
```

**Problems:**
- Changing password requires re-encrypting all passwords
- Can't share with team (everyone has different passwords)
- Vulnerable to password reuse attacks

✅ **Correct Approach: Master Key System**
```
1. Generate random Master Key (256-bit)
2. Encrypt database passwords with Master Key
3. Encrypt Master Key with User-Derived Key
4. Store encrypted Master Key in Turso
```

**Benefits:**
- ✅ Password changes only require re-encrypting Master Key
- ✅ Can share Master Key with team (encrypt with their keys)
- ✅ Industry-standard pattern (used by 1Password, Bitwarden, etc.)
- ✅ Zero-knowledge: Server never sees Master Key or passwords

---

## 🔑 Encryption Stack

### 1. Key Derivation: **PBKDF2-SHA256**
- **Input**: User's login password + unique salt
- **Output**: 256-bit encryption key
- **Iterations**: 600,000+ (OWASP 2023 recommendation)
- **Purpose**: Derive strong key from user password

### 2. Encryption Algorithm: **AES-256-GCM**
- **Key size**: 256 bits
- **Mode**: Galois/Counter Mode (authenticated encryption)
- **IV/Nonce**: 96 bits, unique per operation
- **Auth tag**: 128 bits (prevents tampering)
- **Purpose**: Encrypt passwords and master key

### 3. Encoding: **Base64**
- For storing binary data in database

---

## 📊 Database Schema

```sql
-- User's encrypted master key
CREATE TABLE user_master_keys (
    user_id TEXT PRIMARY KEY,
    encrypted_master_key TEXT NOT NULL,  -- AES-256-GCM encrypted, Base64
    key_iv TEXT NOT NULL,                -- IV for master key encryption
    key_auth_tag TEXT NOT NULL,          -- GCM auth tag
    pbkdf2_salt TEXT NOT NULL,           -- Unique per user
    pbkdf2_iterations INTEGER NOT NULL,  -- 600000+
    version INTEGER NOT NULL DEFAULT 1,  -- For future algorithm upgrades
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Encrypted database credentials
CREATE TABLE encrypted_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,         -- Reference to connection
    encrypted_username TEXT,             -- Optional encrypted username
    encrypted_password TEXT NOT NULL,    -- AES-256-GCM encrypted, Base64
    password_iv TEXT NOT NULL,           -- IV for password encryption
    password_auth_tag TEXT NOT NULL,     -- GCM auth tag
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, connection_id)
);

-- Future: Team shared credentials
CREATE TABLE team_shared_keys (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    user_id TEXT NOT NULL,              -- Team member
    encrypted_master_key TEXT NOT NULL, -- Master key encrypted with user's key
    key_iv TEXT NOT NULL,
    key_auth_tag TEXT NOT NULL,
    granted_at TIMESTAMP NOT NULL,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

---

## 🔄 Encryption Flow

### **Sign Up / First Login:**

```
User enters password "MySecurePassword123"
    ↓
1. Generate random salt (32 bytes)
2. PBKDF2(password, salt, 600000 iterations) → User Key (256 bits)
3. Generate random Master Key (256 bytes)
4. Encrypt Master Key with User Key (AES-256-GCM) → Encrypted Master Key
5. Store in Turso:
   - encrypted_master_key (Base64)
   - key_iv (Base64)
   - key_auth_tag (Base64)
   - pbkdf2_salt (Base64)
   - pbkdf2_iterations (600000)
```

### **Storing a Database Password:**

```
User adds connection with password "postgres_pass_123"
    ↓
1. Retrieve encrypted Master Key from Turso
2. Decrypt Master Key using User Key (from session)
3. Generate random IV for password encryption
4. Encrypt password with Master Key (AES-256-GCM)
5. Store in Turso:
   - encrypted_password (Base64)
   - password_iv (Base64)
   - password_auth_tag (Base64)
   - connection_id (reference)
```

### **Retrieving a Database Password:**

```
User opens connection
    ↓
1. Fetch encrypted password from Turso
2. Get Master Key from memory/session
   (if not in memory, decrypt from Turso using User Key)
3. Decrypt password using Master Key
4. Use password to connect to database
```

### **Changing User Password:**

```
User changes login password
    ↓
1. Derive NEW User Key from new password
2. Decrypt Master Key with OLD User Key
3. Re-encrypt Master Key with NEW User Key
4. Update encrypted_master_key in Turso
5. Database passwords unchanged (still encrypted with same Master Key)
```

---

## 🔒 Security Properties

### ✅ Zero-Knowledge
- Server (Turso) never sees:
  - User's login password
  - User-derived encryption key
  - Master key (plaintext)
  - Database passwords (plaintext)

### ✅ Data Breach Protection
If Turso is compromised:
- ❌ Attacker gets: Encrypted data, salts, IVs
- ✅ Attacker needs: User's login password (not stored)
- ✅ PBKDF2 makes brute force very slow (600k iterations)
- ✅ AES-256-GCM prevents tampering (auth tags)

### ✅ Multi-Device Support
- Same user key derived on each device
- Same master key decrypted on each device
- Passwords accessible everywhere

### ✅ Team Sharing (Future)
```
Share password with teammate:
1. Encrypt Master Key with teammate's public key
2. Teammate decrypts with their private key
3. Both can decrypt passwords with same Master Key
```

---

## 💻 Implementation

### Frontend (TypeScript)

**File:** `frontend/src/lib/crypto/encryption.ts`

```typescript
/**
 * Cryptographic utilities for password encryption
 * Uses Web Crypto API (browser standard)
 */

const PBKDF2_ITERATIONS = 600_000; // OWASP 2023 recommendation
const KEY_LENGTH = 256; // bits
const IV_LENGTH = 12; // bytes (96 bits for GCM)
const SALT_LENGTH = 32; // bytes

// Convert between string and ArrayBuffer
function str2ab(str: string): ArrayBuffer {
  return new TextEncoder().encode(str);
}

function ab2str(buf: ArrayBuffer): string {
  return new TextDecoder().decode(buf);
}

function ab2base64(buf: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buf)));
}

function base642ab(base64: string): ArrayBuffer {
  return Uint8Array.from(atob(base64), c => c.charCodeAt(0)).buffer;
}

/**
 * Generate random salt for PBKDF2
 */
export function generateSalt(): ArrayBuffer {
  return crypto.getRandomValues(new Uint8Array(SALT_LENGTH)).buffer;
}

/**
 * Generate random IV for AES-GCM
 */
export function generateIV(): ArrayBuffer {
  return crypto.getRandomValues(new Uint8Array(IV_LENGTH)).buffer;
}

/**
 * Derive encryption key from user password using PBKDF2
 */
export async function deriveKeyFromPassword(
  password: string,
  salt: ArrayBuffer
): Promise<CryptoKey> {
  // Import password as key material
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    str2ab(password),
    'PBKDF2',
    false,
    ['deriveKey']
  );

  // Derive actual encryption key
  return await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: PBKDF2_ITERATIONS,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: KEY_LENGTH },
    false, // not extractable
    ['encrypt', 'decrypt']
  );
}

/**
 * Generate random master key (for encrypting passwords)
 */
export async function generateMasterKey(): Promise<CryptoKey> {
  return await crypto.subtle.generateKey(
    { name: 'AES-GCM', length: KEY_LENGTH },
    true, // extractable (so we can encrypt it)
    ['encrypt', 'decrypt']
  );
}

/**
 * Export master key to raw bytes (for encryption)
 */
export async function exportMasterKey(key: CryptoKey): Promise<ArrayBuffer> {
  return await crypto.subtle.exportKey('raw', key);
}

/**
 * Import master key from raw bytes
 */
export async function importMasterKey(keyData: ArrayBuffer): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    'raw',
    keyData,
    'AES-GCM',
    true,
    ['encrypt', 'decrypt']
  );
}

/**
 * Encrypt data using AES-256-GCM
 */
export async function encrypt(
  data: string,
  key: CryptoKey,
  iv: ArrayBuffer
): Promise<{ ciphertext: ArrayBuffer; authTag: ArrayBuffer }> {
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    key,
    str2ab(data)
  );

  // GCM mode appends auth tag to ciphertext
  // Extract auth tag (last 16 bytes)
  const authTag = encrypted.slice(-16);
  const ciphertext = encrypted.slice(0, -16);

  return { ciphertext, authTag };
}

/**
 * Decrypt data using AES-256-GCM
 */
export async function decrypt(
  ciphertext: ArrayBuffer,
  authTag: ArrayBuffer,
  key: CryptoKey,
  iv: ArrayBuffer
): Promise<string> {
  // Combine ciphertext and auth tag
  const combined = new Uint8Array(ciphertext.byteLength + authTag.byteLength);
  combined.set(new Uint8Array(ciphertext), 0);
  combined.set(new Uint8Array(authTag), ciphertext.byteLength);

  const decrypted = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    key,
    combined
  );

  return ab2str(decrypted);
}

/**
 * Complete flow: Encrypt master key with user password
 */
export async function encryptMasterKey(
  masterKey: CryptoKey,
  userPassword: string
): Promise<{
  encryptedKey: string;
  iv: string;
  authTag: string;
  salt: string;
  iterations: number;
}> {
  // Generate salt and IV
  const salt = generateSalt();
  const iv = generateIV();

  // Derive user key from password
  const userKey = await deriveKeyFromPassword(userPassword, salt);

  // Export master key to raw bytes
  const masterKeyBytes = await exportMasterKey(masterKey);

  // Encrypt master key with user key
  const { ciphertext, authTag } = await encrypt(
    ab2base64(masterKeyBytes), // Convert to string for encryption
    userKey,
    iv
  );

  return {
    encryptedKey: ab2base64(ciphertext),
    iv: ab2base64(iv),
    authTag: ab2base64(authTag),
    salt: ab2base64(salt),
    iterations: PBKDF2_ITERATIONS,
  };
}

/**
 * Complete flow: Decrypt master key with user password
 */
export async function decryptMasterKey(
  encryptedKey: string,
  iv: string,
  authTag: string,
  salt: string,
  userPassword: string
): Promise<CryptoKey> {
  // Derive user key from password
  const userKey = await deriveKeyFromPassword(userPassword, base642ab(salt));

  // Decrypt master key
  const masterKeyBase64 = await decrypt(
    base642ab(encryptedKey),
    base642ab(authTag),
    userKey,
    base642ab(iv)
  );

  // Import master key
  const masterKeyBytes = base642ab(masterKeyBase64);
  return await importMasterKey(masterKeyBytes);
}

/**
 * Encrypt a database password with master key
 */
export async function encryptPassword(
  password: string,
  masterKey: CryptoKey
): Promise<{
  encryptedPassword: string;
  iv: string;
  authTag: string;
}> {
  const iv = generateIV();
  const { ciphertext, authTag } = await encrypt(password, masterKey, iv);

  return {
    encryptedPassword: ab2base64(ciphertext),
    iv: ab2base64(iv),
    authTag: ab2base64(authTag),
  };
}

/**
 * Decrypt a database password with master key
 */
export async function decryptPassword(
  encryptedPassword: string,
  iv: string,
  authTag: string,
  masterKey: CryptoKey
): Promise<string> {
  return await decrypt(
    base642ab(encryptedPassword),
    base642ab(authTag),
    masterKey,
    base642ab(iv)
  );
}
```

---

### Backend (Go)

**File:** `backend-go/pkg/crypto/encryption.go`

```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Iterations = 600000
	KeyLength        = 32  // 256 bits
	IVLength         = 12  // 96 bits for GCM
	SaltLength       = 32  // bytes
	AuthTagLength    = 16  // 128 bits
)

// GenerateSalt creates a random salt for PBKDF2
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateIV creates a random IV for AES-GCM
func GenerateIV() ([]byte, error) {
	iv := make([]byte, IVLength)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	return iv, nil
}

// DeriveKeyFromPassword uses PBKDF2 to derive encryption key
func DeriveKeyFromPassword(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeyLength, sha256.New)
}

// GenerateMasterKey creates a random 256-bit key
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, KeyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptedData holds encrypted content with metadata
type EncryptedData struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	AuthTag    string `json:"authTag"`
}

// Encrypt data using AES-256-GCM
func Encrypt(plaintext string, key []byte) (*EncryptedData, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	iv, err := GenerateIV()
	if err != nil {
		return nil, err
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	// Extract auth tag (last 16 bytes)
	authTag := ciphertext[len(ciphertext)-AuthTagLength:]
	ciphertext = ciphertext[:len(ciphertext)-AuthTagLength]

	return &EncryptedData{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString(iv),
		AuthTag:    base64.StdEncoding.EncodeToString(authTag),
	}, nil
}

// Decrypt data using AES-256-GCM
func Decrypt(data *EncryptedData, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return "", err
	}

	iv, err := base64.StdEncoding.DecodeString(data.IV)
	if err != nil {
		return "", err
	}

	authTag, err := base64.StdEncoding.DecodeString(data.AuthTag)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Combine ciphertext and auth tag
	combined := append(ciphertext, authTag...)

	plaintext, err := gcm.Open(nil, iv, combined, nil)
	if err != nil {
		return "", errors.New("decryption failed: invalid key or corrupted data")
	}

	return string(plaintext), nil
}
```

---

## 🚀 Usage Example

```typescript
// On user signup/login
const masterKey = await generateMasterKey();
const encrypted = await encryptMasterKey(masterKey, userPassword);

// Store in Turso
await tursoClient.createUserMasterKey({
  userId: user.id,
  encryptedKey: encrypted.encryptedKey,
  iv: encrypted.iv,
  authTag: encrypted.authTag,
  salt: encrypted.salt,
  iterations: encrypted.iterations,
});

// Store a database password
const encryptedPass = await encryptPassword(dbPassword, masterKey);
await tursoClient.createEncryptedCredential({
  userId: user.id,
  connectionId: connection.id,
  encryptedPassword: encryptedPass.encryptedPassword,
  iv: encryptedPass.iv,
  authTag: encryptedPass.authTag,
});

// Retrieve a database password
const creds = await tursoClient.getEncryptedCredential(connectionId);
const password = await decryptPassword(
  creds.encryptedPassword,
  creds.iv,
  creds.authTag,
  masterKey
);
```

---

## 📋 Migration Path

1. ✅ Add encryption utilities (frontend + backend)
2. ✅ Add database tables to Turso
3. ✅ Update auth flow to generate master key
4. ✅ Update connection storage to encrypt passwords
5. ✅ Update connection retrieval to decrypt passwords
6. ✅ Add master key caching in session (don't re-derive every time)
7. ✅ Test encryption/decryption flows
8. ✅ Deploy with database migration

---

## 🔐 Security Checklist

- ✅ AES-256-GCM (authenticated encryption)
- ✅ PBKDF2 with 600k iterations (OWASP 2023)
- ✅ Unique salt per user
- ✅ Unique IV per encryption
- ✅ Authentication tags prevent tampering
- ✅ Zero-knowledge (server never sees plaintext)
- ✅ Master key system (supports password changes)
- ✅ Team sharing ready (future)
- ✅ Web Crypto API (browser standard)
- ✅ constant-time operations (prevents timing attacks)

---

## ⚠️ Important Notes

1. **Master key caching**: Keep in memory for session duration
2. **Re-authentication**: Require password for sensitive operations
3. **Key rotation**: Plan for future key upgrades (version field)
4. **Backup**: User must remember password (no recovery without it)
5. **Team sharing**: Requires public key infrastructure (Phase 2)

This is industry-standard, battle-tested encryption used by password managers like 1Password and Bitwarden.
