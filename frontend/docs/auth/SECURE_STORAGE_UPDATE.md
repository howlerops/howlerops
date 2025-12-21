# Secure Storage Update - Keychain Integration

## Summary

Updated `/Users/jacob_1/projects/sql-studio/frontend/src/lib/secure-storage.ts` to use Wails keychain API instead of localStorage for secure credential storage.

## Changes Made

### 1. Removed localStorage Dependencies
- Removed `loadFromStorage()` method
- Removed `saveToStorage()` method
- Removed all `localStorage.getItem()` and `localStorage.setItem()` calls
- Removed `STORAGE_KEY` constant

### 2. Added Wails Keychain Integration
- Imports Wails App methods: `import * as App from '../../wailsjs/go/main/App'`
- Uses `App.StorePassword()` to store credentials
- Uses `App.GetPassword()` to retrieve credentials
- Uses `App.DeletePassword()` to remove credentials
- Keychain keys are prefixed with `sql-studio-credentials-${connectionId}`

### 3. All Methods Now Async
Changed method signatures to async:
```typescript
// Before
setCredentials(connectionId: string, credentials: Omit<SecureCredential, 'connectionId'>): void

// After
async setCredentials(connectionId: string, credentials: Omit<SecureCredential, 'connectionId'>): Promise<void>

// Before
getCredentials(connectionId: string): SecureCredential | null

// After
async getCredentials(connectionId: string): Promise<SecureCredential | null>

// Before
removeCredentials(connectionId: string): void

// After
async removeCredentials(connectionId: string): Promise<void>

// Before
clearAll(): void

// After
async clearAll(): Promise<void>
```

### 4. Error Handling & Graceful Degradation
- Checks if keychain API methods exist before calling (handles case where backend not yet updated)
- Falls back to in-memory storage if keychain unavailable
- Logs helpful warnings when keychain API not available
- Doesn't throw errors - continues with in-memory cache

### 5. Performance Optimization
- Maintains in-memory cache (`credentials` Map) for fast reads
- Checks cache first before hitting keychain
- Updates cache immediately on writes

### 6. New Features
- Added `preloadCredentials(connectionIds: string[])` method for cache warming
- Better error messages distinguishing "not found" from actual errors
- Updated migration function to be async

## Breaking Changes for Consumers

### Files That Need Updates

The following files use secure storage and need to be updated to handle async operations:

#### 1. `/Users/jacob_1/projects/sql-studio/frontend/src/store/connection-store.ts`

**Lines that need updating:**

**Line 136** - `setCredentials` in `addConnection`:
```typescript
// Before (sync)
secureStorage.setCredentials(newConnection.id, {
  password: connectionData.password,
  sshPassword: connectionData.sshTunnel?.password,
  sshPrivateKey: connectionData.sshTunnel?.privateKey
})

// After (async) - needs await
await secureStorage.setCredentials(newConnection.id, {
  password: connectionData.password,
  sshPassword: connectionData.sshTunnel?.password,
  sshPrivateKey: connectionData.sshTunnel?.privateKey
})

// And change addConnection to async
addConnection: async (connectionData) => {
```

**Line 174-175** - `getCredentials` and `setCredentials` in `updateConnection`:
```typescript
// Before (sync)
const existing = secureStorage.getCredentials(id) || { connectionId: id }
secureStorage.setCredentials(id, {

// After (async) - needs await
const existing = await secureStorage.getCredentials(id) || { connectionId: id }
await secureStorage.setCredentials(id, {

// And change updateConnection to async
updateConnection: async (id, updates) => {
```

**Line 217** - `removeCredentials` in `removeConnection`:
```typescript
// Before (sync)
secureStorage.removeCredentials(id)

// After (async) - needs await
await secureStorage.removeCredentials(id)

// And change removeConnection to async
removeConnection: async (id) => {
```

**Line 245** - `getCredentials` in `connectToDatabase`:
```typescript
// Before (sync)
const credentials = secureStorage.getCredentials(connectionId)

// After (async) - needs await
const credentials = await secureStorage.getCredentials(connectionId)

// Note: connectToDatabase is already async, so just add await
```

#### 2. `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/password-transfer.ts`

**Line 162** - `setCredentials` in `receivePasswords`:
```typescript
// Before (sync)
passwords.forEach(pwd => {
  secureStorage.setCredentials(pwd.connectionId, pwd)
})

// After (async) - needs await + Promise.all
await Promise.all(
  passwords.map(pwd =>
    secureStorage.setCredentials(pwd.connectionId, pwd)
  )
)
```

**Line 453** - `getCredentials` in password export:
```typescript
// Before (sync)
for (const connectionId of connectionIds) {
  const credentials = secureStorage.getCredentials(connectionId)

// After (async) - needs await
for (const connectionId of connectionIds) {
  const credentials = await secureStorage.getCredentials(connectionId)
```

## Backend Requirements

The Wails backend (`app.go`) needs to implement three new methods:

```go
// StorePassword stores a password in the OS keychain
func (a *App) StorePassword(key string, value string) error {
  // Implementation using OS keychain
  // - macOS: use Keychain
  // - Windows: use Credential Manager
  // - Linux: use Secret Service
}

// GetPassword retrieves a password from the OS keychain
func (a *App) GetPassword(key string) (string, error) {
  // Implementation using OS keychain
  // Return error if not found
}

// DeletePassword removes a password from the OS keychain
func (a *App) DeletePassword(key string) error {
  // Implementation using OS keychain
  // Don't error if already doesn't exist
}
```

## Testing Checklist

- [ ] Backend implements StorePassword, GetPassword, DeletePassword
- [ ] Wails bindings regenerated (`wails generate module`)
- [ ] Update connection-store.ts to use async methods
- [ ] Update password-transfer.ts to use async methods
- [ ] Test adding new connection with password
- [ ] Test updating connection password
- [ ] Test removing connection
- [ ] Test connecting to database (password retrieval)
- [ ] Test app restart - credentials should persist
- [ ] Test fallback behavior if keychain unavailable
- [ ] Test migration from localStorage to keychain

## Migration Strategy

1. **Phase 1**: Backend implements keychain methods (separate PR)
2. **Phase 2**: Frontend updates consuming code (this PR can include these updates)
3. **Phase 3**: Run migration on app startup to move existing localStorage credentials to keychain
4. **Phase 4**: Remove old localStorage migration code after sufficient adoption

## Notes

- Keychain storage is more secure than localStorage
- Credentials now stored at OS level, not in app data
- In-memory cache provides same performance as before
- Graceful fallback ensures app works even if keychain unavailable
- Migration function updated to be async but not yet fully implemented
