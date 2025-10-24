# Credential Migration Guide

## Overview

This guide covers the migration of database credentials from browser localStorage to the OS-level keychain for enhanced security.

## Architecture

### Components

1. **migrate-credentials.ts** - Core migration logic
2. **secure-storage.ts** - Keychain integration layer
3. **Wails Backend** - OS keychain access via Go

### Flow

```
localStorage → Migration Utility → Wails Backend → OS Keychain
     ↓                                                    ↓
 Cleared after                                    macOS Keychain
 successful                                       Windows Credential Manager
 migration                                        Linux Secret Service
```

## Migration Process

### Automatic Migration

The migration runs automatically on app startup via the `useMigrateCredentials()` hook:

```typescript
// In app.tsx or main.tsx
import { useMigrateCredentials } from './lib/migrate-credentials'

function App() {
  useMigrateCredentials()
  // ... rest of app
}
```

### Migration Steps

1. **Check Status** - Verifies if migration already completed
2. **Extract Credentials** - Reads from localStorage
3. **Store in Keychain** - Calls Wails backend for each credential
4. **Mark Complete** - Sets migration flag
5. **Clear localStorage** - Removes old credentials

### Credential Types Migrated

- **Database Passwords** - `password` field
- **SSH Passwords** - `sshPassword` field
- **SSH Private Keys** - `sshPrivateKey` field

## API Reference

### Functions

#### `migrateCredentialsToKeychain()`

Main migration function. Runs once per installation.

```typescript
const result: MigrationResult = await migrateCredentialsToKeychain()

// Result structure
{
  success: boolean
  migratedCount: number
  failedCount: number
  errors: Array<{ connectionId: string, error: string }>
  skipped: boolean
  reason?: string
}
```

#### `getMigrationStatus()`

Check migration status without performing migration.

```typescript
const status = getMigrationStatus()

// Status structure
{
  migrated: boolean
  version: string | null
  hasCredentials: boolean
  keychainAvailable: boolean
}
```

#### `retryMigration()`

Force retry migration (useful for debugging).

```typescript
const result = await retryMigration()
```

#### `clearMigrationFlag()`

Clear migration flag for testing.

```typescript
clearMigrationFlag()
```

### React Hook

#### `useMigrateCredentials()`

React hook for automatic migration on mount.

```typescript
function App() {
  useMigrateCredentials() // Runs once on mount
  return <div>...</div>
}
```

## Error Handling

### Strategy

- **Non-Blocking** - App starts even if migration fails
- **Graceful Degradation** - Falls back to localStorage if keychain unavailable
- **Individual Failures** - One failed credential doesn't stop others
- **Preservation** - localStorage kept intact on failure

### Error Scenarios

1. **Keychain Unavailable**
   - Result: Migration skipped, credentials stay in localStorage
   - Log: "Keychain API not yet available"

2. **Parse Error**
   - Result: Migration fails, localStorage preserved
   - Log: "Failed to parse credentials"

3. **Individual Credential Failure**
   - Result: Other credentials continue, localStorage preserved
   - Log: "Failed to migrate credentials for {connectionId}"

4. **Partial Success**
   - Result: Success count logged, localStorage preserved until all succeed
   - Log: "Partial migration: X succeeded, Y failed"

## Storage Keys

### Migration Status

- `credentials-migrated` - Boolean flag (`"true"` or `null`)
- `credentials-migration-version` - Version string (`"1.0"`)

### Credential Storage

- **localStorage Key**: `sql-studio-secure-credentials`
- **Keychain Service**: `sql-studio`
- **Keychain Account Format**: `{connectionId}-{type}`
  - Example: `conn-123-password`
  - Example: `conn-123-ssh_password`

## Testing

### Unit Tests

Run tests with Vitest:

```bash
npm test migrate-credentials
```

### Test Coverage

- Migration status checks
- Single credential migration
- Multiple credential migration
- All credential types (password, SSH password, SSH key)
- Error handling (parse errors, keychain failures)
- Partial migration scenarios
- Edge cases (empty arrays, special characters)

### Manual Testing

```typescript
// 1. Clear migration flag
clearMigrationFlag()

// 2. Add test credentials to localStorage
localStorage.setItem('sql-studio-secure-credentials', JSON.stringify([
  { connectionId: 'test-1', password: 'test123' }
]))

// 3. Check status
console.log(getMigrationStatus())

// 4. Run migration
const result = await migrateCredentialsToKeychain()
console.log(result)

// 5. Verify keychain (via Wails API)
// Check that credentials are in OS keychain
```

## Backend Integration

### Required Wails Functions

The Go backend must expose these functions:

```go
// Store password in OS keychain
func (a *App) StorePassword(service, account, password string) error

// Get password from OS keychain
func (a *App) GetPassword(service, account string) (string, error)

// Delete password from OS keychain
func (a *App) DeletePassword(service, account string) error
```

### Keychain Libraries

Recommended Go library: `github.com/zalando/go-keyring`

```go
import "github.com/zalando/go-keyring"

func (a *App) StorePassword(service, account, password string) error {
    return keyring.Set(service, account, password)
}

func (a *App) GetPassword(service, account string) (string, error) {
    return keyring.Get(service, account)
}

func (a *App) DeletePassword(service, account string) error {
    return keyring.Delete(service, account)
}
```

## Security Considerations

### Benefits

- **OS-Level Security** - Credentials encrypted by OS
- **User Authentication** - May require user login/biometrics
- **Secure Persistence** - Survives app uninstall (user choice)
- **No Plain Text** - Never stored in plain text on disk

### Limitations

- **Availability** - Requires OS keychain support
- **Permissions** - May require user approval
- **Portability** - Credentials don't transfer between machines

### Best Practices

1. **Fallback Strategy** - Always support in-memory fallback
2. **Error Logging** - Log failures for debugging
3. **User Communication** - Inform users about keychain access
4. **Testing** - Test on all platforms (macOS, Windows, Linux)

## Rollback Plan

If migration causes issues:

1. **Disable Migration**
   ```typescript
   // Comment out in app.tsx
   // useMigrateCredentials()
   ```

2. **Clear Migration Flag**
   ```typescript
   clearMigrationFlag()
   ```

3. **Revert to localStorage**
   - Credentials remain in localStorage if migration failed
   - App continues to work with localStorage-based storage

## Platform Support

### macOS
- Keychain Access
- System Keychain
- User may need to approve access

### Windows
- Credential Manager
- Windows Credential Vault
- Integrated with Windows Hello

### Linux
- Secret Service (GNOME Keyring, KWallet)
- May require additional packages
- Varies by desktop environment

## Monitoring

### Success Metrics

- Migration success rate
- Number of credentials migrated
- Time to complete migration

### Error Tracking

```typescript
// Log to analytics/monitoring service
const result = await migrateCredentialsToKeychain()
if (!result.success) {
  analytics.track('Migration Failed', {
    failedCount: result.failedCount,
    errors: result.errors
  })
}
```

## FAQ

### Q: What if keychain is not available?

A: Migration is skipped. App continues using localStorage.

### Q: What if migration fails?

A: Credentials remain in localStorage. App functions normally.

### Q: Can users retry migration?

A: Yes, via `retryMigration()` or by clearing the flag.

### Q: Are credentials deleted from localStorage?

A: Only after ALL credentials successfully migrate.

### Q: What about partial migration?

A: localStorage preserved until all succeed to prevent data loss.

### Q: How to test migration?

A: Use `clearMigrationFlag()` and `migrateCredentialsToKeychain()`.

## Changelog

### Version 1.0 (Current)
- Initial migration implementation
- Support for password, SSH password, SSH private key
- Non-blocking with graceful degradation
- Comprehensive error handling
- Full test coverage

## Future Enhancements

1. **Progress Indication** - Show UI during migration
2. **Batch Processing** - Migrate in chunks for large datasets
3. **Encryption** - Additional encryption layer before keychain
4. **Audit Log** - Track all keychain operations
5. **Migration Analytics** - Detailed success/failure metrics
