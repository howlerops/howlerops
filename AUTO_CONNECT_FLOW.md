# Auto-Connect Flow Diagram

## User Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER CONNECTS TO DATABASE                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  connectToDatabase(connectionId)                                 │
│  └─ Retrieves credentials from secure storage                   │
│  └─ Connects to database                                         │
│  └─ Updates state:                                               │
│     • activeConnection = updatedConnection                       │
│     • lastActiveConnectionId = connectionId  ◄─── TRACKED        │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  Zustand persist middleware saves to localStorage:               │
│  {                                                                │
│    connections: [...],                                           │
│    lastActiveConnectionId: "abc-123-def",  ◄──── PERSISTED       │
│    autoConnectEnabled: true                                      │
│  }                                                                │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
                    [User reloads app]
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      APP STARTS (app.tsx)                        │
│                                                                   │
│  useEffect(() => {                                               │
│    initializeAuthStore()                                         │
│    initializeTierStore()                                         │
│    initializeOrganizationStore()                                 │
│    migrateCredentialsToKeychain()                                │
│                                                                   │
│    setTimeout(() => {                                            │
│      initializeConnectionStore()  ◄──── AUTO-CONNECT TRIGGERED   │
│    }, 100)  // Wait for store hydration                          │
│  }, [])                                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  initializeConnectionStore()                                     │
│                                                                   │
│  1. Check autoConnectEnabled                                     │
│     ├─ false → return (exit silently)                           │
│     └─ true → continue                                           │
│                                                                   │
│  2. Get lastActiveConnectionId                                   │
│     ├─ null → return (no previous connection)                   │
│     └─ "abc-123-def" → continue                                  │
│                                                                   │
│  3. Find connection in connections array                         │
│     ├─ not found → return (connection deleted)                  │
│     └─ found → continue                                          │
│                                                                   │
│  4. Check if already connected                                   │
│     ├─ isConnected = true → return (already active)             │
│     └─ isConnected = false → continue                            │
│                                                                   │
│  5. Auto-connect                                                 │
│     try {                                                         │
│       await connectToDatabase(lastActiveConnectionId)            │
│       console.debug('Auto-connect successful')                   │
│     } catch (error) {                                            │
│       console.warn('Auto-connect failed')  ◄─── FAIL GRACEFULLY │
│       // App continues normally                                  │
│     }                                                             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              USER SEES DATABASE CONNECTED                        │
│                    (No manual action needed)                     │
└─────────────────────────────────────────────────────────────────┘
```

## State Tracking

### When Connection Becomes Active

```typescript
// In setActiveConnection()
set({
  activeConnection: connection,
  lastActiveConnectionId: connection?.id ?? null  // ← Tracked here
})

// In connectToDatabase() on success
set({
  connections: [...],
  activeConnection: updatedConnection,
  lastActiveConnectionId: connectionId  // ← Also tracked here
})
```

### Persistence Layer

```typescript
// Zustand persist configuration
partialize: (state) => ({
  connections: [...],
  lastActiveConnectionId: state.lastActiveConnectionId,  // ← Persisted
  autoConnectEnabled: state.autoConnectEnabled,
  activeEnvironmentFilter: state.activeEnvironmentFilter,
})
```

## Security Model

```
┌──────────────────────┐
│  localStorage        │     ┌────────────────────┐
│  (Non-sensitive)     │     │  OS Keychain       │
│                      │     │  (Sensitive)       │
│  • Connection ID     │     │                    │
│  • Connection name   │     │  • password        │
│  • Host/port         │     │  • sshPassword     │
│  • Database name     │     │  • sshPrivateKey   │
│  • lastActiveConnId  │     │                    │
│  • autoConnectFlag   │     │  Retrieved at      │
│                      │     │  connect time      │
└──────────────────────┘     └────────────────────┘
         │                            │
         └────────┬───────────────────┘
                  │
                  ▼
         ┌────────────────┐
         │ connectToDb()  │
         └────────────────┘
```

## Error Scenarios

### Scenario 1: Invalid Credentials
```
initializeConnectionStore()
  └─ connectToDatabase(id)
       └─ Retrieves credentials from keychain
       └─ Attempts connection
       └─ ❌ Auth failed
       └─ catch block: console.warn()
       └─ App continues, user can try again
```

### Scenario 2: Server Down
```
initializeConnectionStore()
  └─ connectToDatabase(id)
       └─ Retrieves credentials
       └─ Attempts connection
       └─ ❌ Network timeout
       └─ catch block: console.warn()
       └─ App continues, user can try later
```

### Scenario 3: Connection Deleted
```
initializeConnectionStore()
  └─ Find connection by ID
  └─ ❌ Not found
  └─ return early
  └─ App continues normally
```

### Scenario 4: Auto-Connect Disabled
```
initializeConnectionStore()
  └─ Check autoConnectEnabled
  └─ ❌ false
  └─ return early
  └─ App continues, no auto-connect
```

## Benefits Summary

1. **Seamless UX**: Users stay connected between sessions
2. **Non-blocking**: Runs in background, doesn't delay app startup
3. **Safe**: Multiple guard clauses prevent errors
4. **Secure**: Works with encrypted credential storage
5. **Configurable**: Can be disabled via `autoConnectEnabled`
6. **Resilient**: Handles all error cases gracefully

## Configuration

### Enable/Disable Auto-Connect
```typescript
// Disable auto-connect
useConnectionStore.getState().setAutoConnect(false)

// Enable auto-connect
useConnectionStore.getState().setAutoConnect(true)
```

### Check Current State
```typescript
const { lastActiveConnectionId, autoConnectEnabled } = useConnectionStore()

console.log('Last active:', lastActiveConnectionId)
console.log('Auto-connect enabled:', autoConnectEnabled)
```

### Manual Clear
```typescript
// Clear last active connection (won't auto-connect on next reload)
useConnectionStore.getState().setActiveConnection(null)
```
