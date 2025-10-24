# Credential Migration - Quick Start

## TL;DR

Migration utility to move credentials from localStorage to OS keychain. Non-blocking, runs once per installation.

## Integration (2 lines)

```typescript
// In app.tsx
import { useMigrateCredentials } from './lib/migrate-credentials'

function App() {
  useMigrateCredentials() // ← Add this line
  // ... rest of app
}
```

## What It Does

1. ✓ Runs automatically on app startup
2. ✓ Checks if already migrated (runs once)
3. ✓ Extracts credentials from localStorage
4. ✓ Stores in OS keychain via Wails
5. ✓ Clears localStorage on success
6. ✓ Non-blocking (app starts even if fails)

## Credential Types Migrated

- Database passwords
- SSH passwords
- SSH private keys

## When Migration Runs

- **First app start after update** - Migrates existing credentials
- **Subsequent starts** - Skipped (already migrated)
- **No credentials** - Skipped (nothing to migrate)
- **Keychain unavailable** - Skipped (falls back to localStorage)

## Backend Requirements

The Go backend needs these functions (via `github.com/zalando/go-keyring`):

```go
// app.go
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

Until these are implemented, migration is **automatically skipped** and credentials remain in localStorage.

## Error Handling

**Migration never blocks app startup.** All errors are caught and logged.

| Scenario | Behavior |
|----------|----------|
| Keychain unavailable | Skip migration, use localStorage |
| Individual credential fails | Continue with others |
| Parse error | Skip migration, keep localStorage |
| Partial success | Keep localStorage until all succeed |

## Status Check

```typescript
import { getMigrationStatus } from './lib/migrate-credentials'

const status = getMigrationStatus()
// {
//   migrated: boolean
//   version: string | null
//   hasCredentials: boolean
//   keychainAvailable: boolean
// }
```

## Manual Retry

```typescript
import { retryMigration } from './lib/migrate-credentials'

const result = await retryMigration()
console.log(result.migratedCount, result.failedCount)
```

## Testing

```bash
# Run tests
npm test migrate-credentials

# Check migration status in browser console
getMigrationStatus()

# Force retry (dev only)
clearMigrationFlag()
await migrateCredentialsToKeychain()
```

## Storage Keys

- **localStorage**: `sql-studio-secure-credentials`
- **Keychain Service**: `sql-studio`
- **Keychain Account**: `{connectionId}-{type}`
  - Example: `conn-123-password`
  - Example: `conn-456-ssh_password`

## Platform Support

| Platform | Keychain |
|----------|----------|
| macOS | Keychain Access |
| Windows | Credential Manager |
| Linux | Secret Service (GNOME Keyring/KWallet) |

## Rollback

If issues occur:

```typescript
// Option 1: Disable migration (comment out in app.tsx)
// useMigrateCredentials()

// Option 2: Clear flag and retry later
clearMigrationFlag()
```

Credentials remain in localStorage if migration fails, so app continues working.

## Files Created

- `/frontend/src/lib/migrate-credentials.ts` - Core logic
- `/frontend/src/lib/migrate-credentials.test.ts` - Tests
- `/frontend/src/lib/MIGRATION_GUIDE.md` - Full documentation
- `/frontend/src/lib/MIGRATION_QUICKSTART.md` - This file
- `/frontend/src/lib/INTEGRATION_EXAMPLE.tsx` - Integration examples

## Next Steps

1. ✓ Add `useMigrateCredentials()` to app.tsx
2. ✓ Tests already passing (19/19)
3. ⏳ Implement Wails keychain functions (backend)
4. ⏳ Test on all platforms
5. ⏳ Deploy and monitor

## Questions?

See `/frontend/src/lib/MIGRATION_GUIDE.md` for detailed documentation.
