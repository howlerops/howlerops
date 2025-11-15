# Backend Keychain Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing OS keychain integration in the Go backend to support secure credential storage.

## Prerequisites

- Go 1.21 or later
- Wails v2 installed
- Platform-specific keychain libraries (automatically handled by go-keyring)

## Implementation Steps

### 1. Install go-keyring Library

```bash
cd backend-go
go get github.com/zalando/go-keyring
```

This library provides cross-platform keychain access:
- **macOS**: Security Framework (Keychain)
- **Windows**: Windows Credential Manager (WAPI)
- **Linux**: Secret Service API (GNOME Keyring, KWallet)

### 2. Add Keychain Methods to App Struct

Add these three methods to your `app.go` file:

```go
// app.go
package main

import (
    "github.com/zalando/go-keyring"
    // ... other imports
)

// StorePassword stores a password securely in the OS keychain
//
// Parameters:
//   - service: Service name (e.g., "sql-studio")
//   - account: Account/key name (e.g., "conn-123-password")
//   - password: Password or sensitive value to store
//
// Returns:
//   - error: Error if keychain is locked, unavailable, or operation fails
//
// Example:
//   err := app.StorePassword("sql-studio", "conn-123-password", "secret123")
func (a *App) StorePassword(service, account, password string) error {
    a.logger.Debugf("[Keychain] Storing password for %s/%s", service, account)

    err := keyring.Set(service, account, password)
    if err != nil {
        a.logger.Errorf("[Keychain] Failed to store password: %v", err)
        return fmt.Errorf("failed to store password in keychain: %w", err)
    }

    a.logger.Debugf("[Keychain] Successfully stored password for %s/%s", service, account)
    return nil
}

// GetPassword retrieves a password from the OS keychain
//
// Parameters:
//   - service: Service name (e.g., "sql-studio")
//   - account: Account/key name (e.g., "conn-123-password")
//
// Returns:
//   - string: The retrieved password
//   - error: Error if not found, keychain locked, or operation fails
//
// Example:
//   password, err := app.GetPassword("sql-studio", "conn-123-password")
func (a *App) GetPassword(service, account string) (string, error) {
    a.logger.Debugf("[Keychain] Retrieving password for %s/%s", service, account)

    password, err := keyring.Get(service, account)
    if err != nil {
        // Don't log "not found" as error - it's expected
        if err == keyring.ErrNotFound {
            a.logger.Debugf("[Keychain] Password not found for %s/%s", service, account)
        } else {
            a.logger.Errorf("[Keychain] Failed to retrieve password: %v", err)
        }
        return "", fmt.Errorf("failed to get password from keychain: %w", err)
    }

    a.logger.Debugf("[Keychain] Successfully retrieved password for %s/%s", service, account)
    return password, nil
}

// DeletePassword removes a password from the OS keychain
//
// Parameters:
//   - service: Service name (e.g., "sql-studio")
//   - account: Account/key name (e.g., "conn-123-password")
//
// Returns:
//   - error: Error if not found, keychain locked, or operation fails
//
// Example:
//   err := app.DeletePassword("sql-studio", "conn-123-password")
func (a *App) DeletePassword(service, account string) error {
    a.logger.Debugf("[Keychain] Deleting password for %s/%s", service, account)

    err := keyring.Delete(service, account)
    if err != nil {
        // Don't log "not found" as error - it's expected
        if err == keyring.ErrNotFound {
            a.logger.Debugf("[Keychain] Password not found for %s/%s (already deleted)", service, account)
        } else {
            a.logger.Errorf("[Keychain] Failed to delete password: %v", err)
        }
        return fmt.Errorf("failed to delete password from keychain: %w", err)
    }

    a.logger.Debugf("[Keychain] Successfully deleted password for %s/%s", service, account)
    return nil
}
```

### 3. Rebuild Wails Bindings

After adding the methods, regenerate the Wails bindings:

```bash
cd /path/to/sql-studio
wails dev  # This will regenerate bindings automatically
```

Or manually:

```bash
wails generate bindings
```

This will update `/frontend/wailsjs/go/main/App.d.ts` with the new functions.

### 4. Testing the Implementation

#### Unit Tests (Go)

Create `app_keychain_test.go`:

```go
package main

import (
    "testing"
    "github.com/zalando/go-keyring"
)

func TestKeychainOperations(t *testing.T) {
    app := NewApp()
    service := "sql-studio-test"
    account := "test-account"
    password := "test-password"

    // Clean up before test
    keyring.Delete(service, account)

    // Test Store
    err := app.StorePassword(service, account, password)
    if err != nil {
        t.Fatalf("Failed to store password: %v", err)
    }

    // Test Get
    retrieved, err := app.GetPassword(service, account)
    if err != nil {
        t.Fatalf("Failed to get password: %v", err)
    }

    if retrieved != password {
        t.Errorf("Expected %s, got %s", password, retrieved)
    }

    // Test Delete
    err = app.DeletePassword(service, account)
    if err != nil {
        t.Fatalf("Failed to delete password: %v", err)
    }

    // Verify deleted
    _, err = app.GetPassword(service, account)
    if err == nil {
        t.Error("Expected error for deleted password")
    }

    // Clean up
    keyring.Delete(service, account)
}
```

Run tests:

```bash
cd backend-go
go test -v -run TestKeychainOperations
```

#### Integration Tests (Frontend)

```typescript
// In browser console or test file
import * as App from '../../wailsjs/go/main/App'

// Test storing
await App.StorePassword("sql-studio", "test-conn-password", "secret123")
console.log("✓ Store succeeded")

// Test retrieving
const password = await App.GetPassword("sql-studio", "test-conn-password")
console.log("✓ Get succeeded:", password === "secret123")

// Test deleting
await App.DeletePassword("sql-studio", "test-conn-password")
console.log("✓ Delete succeeded")

// Verify deleted
try {
  await App.GetPassword("sql-studio", "test-conn-password")
  console.log("✗ Should have thrown error")
} catch (err) {
  console.log("✓ Correctly throws error for missing password")
}
```

### 5. Platform-Specific Testing

#### macOS
```bash
# View stored credentials
security find-generic-password -s "sql-studio" -a "test-conn-password"

# Delete test credentials
security delete-generic-password -s "sql-studio" -a "test-conn-password"
```

#### Windows
```powershell
# View stored credentials
cmdkey /list | findstr "sql-studio"

# Delete test credentials
cmdkey /delete:"sql-studio:test-conn-password"
```

#### Linux (GNOME)
```bash
# Using secret-tool
secret-tool lookup service sql-studio account test-conn-password

# Delete test credentials
secret-tool clear service sql-studio account test-conn-password
```

### 6. Error Handling

Common errors and how to handle them:

```go
import "github.com/zalando/go-keyring"

err := keyring.Get(service, account)
if err != nil {
    switch {
    case err == keyring.ErrNotFound:
        // Credential doesn't exist - this is expected
        return "", nil
    case strings.Contains(err.Error(), "locked"):
        // Keychain is locked - user needs to unlock
        return "", fmt.Errorf("keychain is locked, please unlock and try again")
    default:
        // Other error - log and return
        a.logger.Errorf("Keychain error: %v", err)
        return "", err
    }
}
```

### 7. Security Considerations

#### Logging
```go
// NEVER log passwords
a.logger.Debugf("[Keychain] Storing password for %s/%s", service, account) // ✓ Good
a.logger.Debugf("[Keychain] Storing: %s", password) // ✗ BAD - Never do this
```

#### Service Names
```go
// Use consistent service names
const KeychainService = "sql-studio"

// Use descriptive account names
account := fmt.Sprintf("%s-%s", connectionId, "password")
```

#### User Permissions
- macOS: User may see system prompt to allow access
- Windows: Usually seamless
- Linux: May require keyring daemon running

### 8. Migration Verification

After implementing the backend functions, verify the frontend migration:

```typescript
// In browser console
import { getMigrationStatus, retryMigration } from './lib/migrate-credentials'

// Check status
const status = getMigrationStatus()
console.log(status)
// {
//   migrated: false,
//   version: null,
//   hasCredentials: true,
//   keychainAvailable: true  // Should now be true!
// }

// Run migration
const result = await retryMigration()
console.log(result)
// {
//   success: true,
//   migratedCount: 5,
//   failedCount: 0,
//   errors: [],
//   skipped: false
// }
```

### 9. Deployment Checklist

- [ ] go-keyring installed (`go get github.com/zalando/go-keyring`)
- [ ] Three methods added to app.go (StorePassword, GetPassword, DeletePassword)
- [ ] Wails bindings regenerated (`wails generate bindings`)
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] Platform-specific testing complete (macOS/Windows/Linux)
- [ ] Error handling verified
- [ ] Logging verified (no password leaks)
- [ ] Frontend migration tested
- [ ] Production build successful

### 10. Troubleshooting

#### Issue: "keychain not available"
**Solution**: Ensure keyring daemon is running (Linux) or keychain is unlocked (macOS)

#### Issue: "permission denied"
**Solution**:
- macOS: Check System Preferences > Security & Privacy
- Windows: Check User Account Control settings
- Linux: Ensure user is in correct groups (`groups $USER`)

#### Issue: Migration not activating
**Solution**: Check browser console for logs. Verify `keychainAvailable: true` in status.

#### Issue: Credentials not persisting
**Solution**: Verify keychain service is running and accessible.

## Performance Considerations

- Keychain access is typically 1-5ms per operation
- Consider caching in-memory after retrieval
- Batch operations when possible
- Don't call on every render - use context/state

## Security Best Practices

1. **Never log passwords or sensitive values**
2. **Use consistent service/account naming**
3. **Handle keychain locked state gracefully**
4. **Clear in-memory caches on logout**
5. **Test on all platforms before release**

## Support

- **go-keyring**: https://github.com/zalando/go-keyring
- **Wails**: https://wails.io/docs/reference/runtime/intro
- **Frontend Migration**: `/frontend/src/lib/MIGRATION_GUIDE.md`

## Example: Complete Implementation

See the code blocks above for a complete, production-ready implementation.

## Next Steps

After implementing:
1. Test thoroughly on all platforms
2. Add to CI/CD pipeline
3. Update user documentation
4. Monitor error logs for keychain issues
5. Consider adding telemetry for migration success rate
