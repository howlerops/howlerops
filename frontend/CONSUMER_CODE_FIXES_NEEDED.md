# Consumer Code Fixes Required for Async Secure Storage

## TypeScript Errors to Fix

The following TypeScript errors occur because the consuming code treats async methods as synchronous:

```
src/lib/sync/password-transfer.ts(457,33): error TS2339: Property 'password' does not exist on type 'Promise<SecureCredential | null>'.
src/lib/sync/password-transfer.ts(458,36): error TS2339: Property 'sshPassword' does not exist on type 'Promise<SecureCredential | null>'.
src/lib/sync/password-transfer.ts(459,38): error TS2339: Property 'sshPrivateKey' does not exist on type 'Promise<SecureCredential | null>'.
src/store/connection-store.ts(176,54): error TS2339: Property 'password' does not exist on type 'Promise<SecureCredential | null>'.
src/store/connection-store.ts(177,68): error TS2339: Property 'sshPassword' does not exist on type 'Promise<SecureCredential | null>'.
src/store/connection-store.ts(178,72): error TS2339: Property 'sshPrivateKey' does not exist on type 'Promise<SecureCredential | null>'.
src/store/connection-store.ts(272,38): error TS2339: Property 'password' does not exist on type 'Promise<SecureCredential | null>'.
```

## Files That Need Updates

### 1. `/Users/jacob_1/projects/sql-studio/frontend/src/store/connection-store.ts`

#### Fix 1: Line 136 - `addConnection` method
**Location:** In the `addConnection` action

```typescript
// BEFORE (Line 136)
addConnection: (connectionData) => {
  // ...
  secureStorage.setCredentials(newConnection.id, {
    password: connectionData.password,
    sshPassword: connectionData.sshTunnel?.password,
    sshPrivateKey: connectionData.sshTunnel?.privateKey
  })
  // ...
},

// AFTER
addConnection: async (connectionData) => {
  // ...
  await secureStorage.setCredentials(newConnection.id, {
    password: connectionData.password,
    sshPassword: connectionData.sshTunnel?.password,
    sshPrivateKey: connectionData.sshTunnel?.privateKey
  })
  // ...
},
```

**Changes:**
1. Add `async` keyword to method signature
2. Add `await` before `secureStorage.setCredentials`

#### Fix 2: Line 170 - `updateConnection` method
**Location:** Lines 174-179 in the `updateConnection` action

```typescript
// BEFORE (Lines 170-179)
updateConnection: (id, updates) => {
  if (updates.password || updates.sshTunnel?.password || updates.sshTunnel?.privateKey) {
    const secureStorage = getSecureStorage()
    const existing = secureStorage.getCredentials(id) || { connectionId: id }
    secureStorage.setCredentials(id, {
      password: updates.password ?? existing.password,
      sshPassword: updates.sshTunnel?.password ?? existing.sshPassword,
      sshPrivateKey: updates.sshTunnel?.privateKey ?? existing.sshPrivateKey
    })
    // ...
  }
},

// AFTER
updateConnection: async (id, updates) => {
  if (updates.password || updates.sshTunnel?.password || updates.sshTunnel?.privateKey) {
    const secureStorage = getSecureStorage()
    const existing = await secureStorage.getCredentials(id) || { connectionId: id }
    await secureStorage.setCredentials(id, {
      password: updates.password ?? existing.password,
      sshPassword: updates.sshTunnel?.password ?? existing.sshPassword,
      sshPrivateKey: updates.sshTunnel?.privateKey ?? existing.sshPrivateKey
    })
    // ...
  }
},
```

**Changes:**
1. Add `async` keyword to method signature
2. Add `await` before `secureStorage.getCredentials`
3. Add `await` before `secureStorage.setCredentials`

#### Fix 3: Line 214 - `removeConnection` method
**Location:** Line 217 in the `removeConnection` action

```typescript
// BEFORE (Line 214)
removeConnection: (id) => {
  const secureStorage = getSecureStorage()
  secureStorage.removeCredentials(id)
  // ...
},

// AFTER
removeConnection: async (id) => {
  const secureStorage = getSecureStorage()
  await secureStorage.removeCredentials(id)
  // ...
},
```

**Changes:**
1. Add `async` keyword to method signature
2. Add `await` before `secureStorage.removeCredentials`

#### Fix 4: Line 233 - `connectToDatabase` method
**Location:** Line 245 in the `connectToDatabase` action (already async)

```typescript
// BEFORE (Line 245)
connectToDatabase: async (connectionId) => {
  // ...
  const credentials = secureStorage.getCredentials(connectionId)
  // ...
},

// AFTER
connectToDatabase: async (connectionId) => {
  // ...
  const credentials = await secureStorage.getCredentials(connectionId)
  // ...
},
```

**Changes:**
1. Method is already async, just add `await` before `secureStorage.getCredentials`

### 2. `/Users/jacob_1/projects/sql-studio/frontend/src/lib/sync/password-transfer.ts`

#### Fix 1: Line 160-163 - `receivePasswords` method
**Location:** In the passwords receive handler

```typescript
// BEFORE (Lines 160-163)
// Store in secure storage
const secureStorage = getSecureStorage()
passwords.forEach(pwd => {
  secureStorage.setCredentials(pwd.connectionId, pwd)
})

// AFTER
// Store in secure storage
const secureStorage = getSecureStorage()
await Promise.all(
  passwords.map(pwd =>
    secureStorage.setCredentials(pwd.connectionId, pwd)
  )
)
```

**Changes:**
1. Change `forEach` to `Promise.all` with `map`
2. Add `await` before `Promise.all`
3. Ensure parent function is async

#### Fix 2: Line 453 - Password export loop
**Location:** In the password collection loop

```typescript
// BEFORE (Line 453)
for (const connectionId of connectionIds) {
  const credentials = secureStorage.getCredentials(connectionId)
  if (credentials) {
    passwords.push({
      connectionId,
      password: credentials.password,
      sshPassword: credentials.sshPassword,
      sshPrivateKey: credentials.sshPrivateKey
    })
  }
}

// AFTER
for (const connectionId of connectionIds) {
  const credentials = await secureStorage.getCredentials(connectionId)
  if (credentials) {
    passwords.push({
      connectionId,
      password: credentials.password,
      sshPassword: credentials.sshPassword,
      sshPrivateKey: credentials.sshPrivateKey
    })
  }
}
```

**Changes:**
1. Add `await` before `secureStorage.getCredentials`
2. Ensure parent function is async

## Summary of Changes

### Method Signature Changes Required

In `connection-store.ts`:
- `addConnection: (connectionData) => {}` → `addConnection: async (connectionData) => {}`
- `updateConnection: (id, updates) => {}` → `updateConnection: async (id, updates) => {}`
- `removeConnection: (id) => {}` → `removeConnection: async (id) => {}`
- `connectToDatabase` - already async, just needs await added

### Await Calls Required

Total locations needing `await`:
- connection-store.ts: 5 locations (2 setCredentials, 2 getCredentials, 1 removeCredentials)
- password-transfer.ts: 2 locations (1 setCredentials batch, 1 getCredentials)

### Pattern for Batch Operations

When setting multiple credentials at once, use `Promise.all`:

```typescript
// ❌ WRONG - doesn't wait for all to complete
passwords.forEach(pwd => {
  secureStorage.setCredentials(pwd.connectionId, pwd)
})

// ✅ CORRECT - waits for all to complete
await Promise.all(
  passwords.map(pwd =>
    secureStorage.setCredentials(pwd.connectionId, pwd)
  )
)
```

## Testing Checklist

After making these changes, verify:

- [ ] `npm run typecheck` passes without secure-storage errors
- [ ] Adding a new connection works
- [ ] Updating connection credentials works
- [ ] Deleting a connection works
- [ ] Connecting to a database retrieves credentials correctly
- [ ] Password transfer/sync still works
- [ ] No regressions in connection management

## Notes

- All changes are backwards compatible if the backend doesn't have keychain methods yet
- The code will gracefully fall back to in-memory storage with helpful warnings
- Once backend implements keychain methods, credentials will automatically persist securely
