# Backend Keychain API Specification

## Overview

The frontend secure storage has been updated to use Wails keychain API instead of localStorage. The backend needs to implement three new methods in `app.go` that interface with the OS keychain.

## Required Methods

### 1. StorePassword

**Signature:**
```go
func (a *App) StorePassword(key string, value string) error
```

**Purpose:** Store a credential securely in the OS keychain

**Parameters:**
- `key`: Unique identifier for the credential (e.g., `sql-studio-credentials-{connectionId}`)
- `value`: The credential value as a JSON string (contains password, sshPassword, sshPrivateKey)

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- Store the key-value pair in the OS keychain
- Overwrite if key already exists
- On macOS: use Keychain
- On Windows: use Credential Manager
- On Linux: use Secret Service (libsecret)

**Example Call from Frontend:**
```typescript
await App.StorePassword(
  'sql-studio-credentials-abc123',
  '{"connectionId":"abc123","password":"secret","sshPassword":"sshsecret"}'
)
```

### 2. GetPassword

**Signature:**
```go
func (a *App) GetPassword(key string) (string, error)
```

**Purpose:** Retrieve a credential from the OS keychain

**Parameters:**
- `key`: Unique identifier for the credential

**Returns:**
- `string`: The credential value as a JSON string
- `error`: nil on success, error if not found or other failure

**Behavior:**
- Retrieve the value associated with the key from OS keychain
- Return error if key doesn't exist (frontend expects this)
- Error message should include "not found" or "NotFound" for missing keys

**Example Call from Frontend:**
```typescript
const value = await App.GetPassword('sql-studio-credentials-abc123')
// value = '{"connectionId":"abc123","password":"secret","sshPassword":"sshsecret"}'
```

### 3. DeletePassword

**Signature:**
```go
func (a *App) DeletePassword(key string) error
```

**Purpose:** Remove a credential from the OS keychain

**Parameters:**
- `key`: Unique identifier for the credential

**Returns:**
- `error`: nil on success or if key doesn't exist, error on other failures

**Behavior:**
- Delete the key-value pair from OS keychain
- Don't return error if key doesn't exist (idempotent operation)
- Only return error on actual failure (permissions, keychain locked, etc.)

**Example Call from Frontend:**
```typescript
await App.DeletePassword('sql-studio-credentials-abc123')
```

## Recommended Go Library

Use the `github.com/zalando/go-keyring` library which provides cross-platform keychain access:

```bash
go get github.com/zalando/go-keyring
```

### Example Implementation

```go
package main

import (
    "fmt"
    "github.com/zalando/go-keyring"
)

const serviceName = "Howlerops"

// StorePassword stores a password in the OS keychain
func (a *App) StorePassword(key string, value string) error {
    err := keyring.Set(serviceName, key, value)
    if err != nil {
        return fmt.Errorf("failed to store password in keychain: %w", err)
    }
    return nil
}

// GetPassword retrieves a password from the OS keychain
func (a *App) GetPassword(key string) (string, error) {
    value, err := keyring.Get(serviceName, key)
    if err != nil {
        if err == keyring.ErrNotFound {
            return "", fmt.Errorf("password not found in keychain")
        }
        return "", fmt.Errorf("failed to get password from keychain: %w", err)
    }
    return value, nil
}

// DeletePassword removes a password from the OS keychain
func (a *App) DeletePassword(key string) error {
    err := keyring.Delete(serviceName, key)
    if err != nil {
        // Don't return error if key doesn't exist
        if err == keyring.ErrNotFound {
            return nil
        }
        return fmt.Errorf("failed to delete password from keychain: %w", err)
    }
    return nil
}
```

## Key Format

Keys follow this pattern:
```
sql-studio-credentials-{connectionId}
```

Where `{connectionId}` is a UUID identifying the database connection.

## Value Format

Values are JSON strings containing:

```json
{
  "connectionId": "abc-123-def-456",
  "password": "database_password",
  "sshPassword": "ssh_password",
  "sshPrivateKey": "-----BEGIN RSA PRIVATE KEY-----\n..."
}
```

**Note:** Not all fields are always present. Only populated fields are included.

## Security Considerations

1. **Service Name**: Use a consistent service name ("Howlerops") for all keychain entries
2. **Encryption**: The OS keychain handles encryption automatically
3. **Permissions**: Entries are accessible only to the current user
4. **Keychain Lock**: If keychain is locked, operations may fail - handle gracefully
5. **User Prompts**: On some systems, first access may prompt user for permission

## Error Handling

Frontend expects specific error patterns:

1. **Not Found**: Error message should contain "not found" or "NotFound"
   - Frontend treats this as normal (credential doesn't exist yet)
   - Don't log as error, just return the error

2. **Other Errors**: Any other error is logged as a problem
   - Keychain locked
   - Permission denied
   - Service unavailable

## Testing

### Manual Testing

```go
// Test storing
err := app.StorePassword("test-key", "test-value")
if err != nil {
    log.Fatal(err)
}

// Test retrieving
value, err := app.GetPassword("test-key")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Retrieved:", value) // Should print "test-value"

// Test deleting
err = app.DeletePassword("test-key")
if err != nil {
    log.Fatal(err)
}

// Test getting after delete (should get not found error)
_, err = app.GetPassword("test-key")
fmt.Println("Expected not found error:", err)
```

### Platform-Specific Verification

**macOS:**
```bash
# View in Keychain Access.app
# Look for service "Howlerops"
open -a "Keychain Access"
```

**Windows:**
```powershell
# View in Credential Manager
control /name Microsoft.CredentialManager
```

**Linux:**
```bash
# Using seahorse (GNOME Keyring viewer)
seahorse
# Look for "Howlerops" entries
```

## Integration Checklist

- [ ] Add `github.com/zalando/go-keyring` dependency
- [ ] Implement `StorePassword(key, value) error`
- [ ] Implement `GetPassword(key) (string, error)`
- [ ] Implement `DeletePassword(key) error`
- [ ] Test on macOS
- [ ] Test on Windows
- [ ] Test on Linux
- [ ] Regenerate Wails bindings: `wails generate module`
- [ ] Verify frontend sees new methods in `App.d.ts`
- [ ] Test with frontend integration

## After Implementation

Once backend is implemented and Wails bindings regenerated:

1. The frontend will automatically start using keychain storage
2. Console warnings about "Keychain API not available" will stop
3. Credentials will persist securely across app restarts
4. Old localStorage credentials can be migrated to keychain

## Alternative Libraries

If `go-keyring` doesn't meet requirements, alternatives include:

- **macOS only**: Use `security` command-line tool via `exec.Command`
- **Windows only**: Use `github.com/danieljoos/wincred`
- **Linux only**: Use `github.com/godbus/dbus` with Secret Service API
- **Cross-platform**: Implement custom solution per platform

## Support

For questions or issues with this specification:
1. Check frontend implementation: `/frontend/src/lib/secure-storage.ts`
2. See usage examples: `/frontend/src/store/connection-store.ts`
3. Review integration docs: `/frontend/SECURE_STORAGE_UPDATE.md`
