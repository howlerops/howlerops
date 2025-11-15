# Auto-Connect Code Snippets Reference

## Core Implementation

### 1. Connection Store - State Definition
**File**: `/Users/jacob_1/projects/sql-studio/frontend/src/store/connection-store.ts`

```typescript
interface ConnectionState {
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  lastActiveConnectionId: string | null // NEW: Track last active for auto-reconnect
  autoConnectEnabled: boolean
  isConnecting: boolean
  // ... other fields
}
```

### 2. Connection Store - Initial State
```typescript
export const useConnectionStore = create<ConnectionState>()(
  devtools(
    persist(
      (set, get) => ({
        connections: [],
        activeConnection: null,
        lastActiveConnectionId: null, // NEW: Initialize as null
        autoConnectEnabled: true,
        isConnecting: false,
        activeEnvironmentFilter: null,
        availableEnvironments: [],
        // ... methods
      })
    )
  )
)
```

### 3. Connection Store - Track Active Connection
```typescript
setActiveConnection: (connection) => {
  set({
    activeConnection: connection,
    lastActiveConnectionId: connection?.id ?? null // NEW: Save ID for later
  })
},
```

### 4. Connection Store - Track on Connect
```typescript
connectToDatabase: async (connectionId) => {
  // ... connection logic

  set((currentState) => ({
    connections: currentState.connections.map((conn) =>
      conn.id === connectionId ? updatedConnection : conn
    ),
    activeConnection: updatedConnection,
    lastActiveConnectionId: connectionId, // NEW: Track on successful connect
  }))
}
```

### 5. Connection Store - Persist Configuration
```typescript
{
  name: 'connection-store',
  partialize: (state) => ({
    connections: /* ... */,
    lastActiveConnectionId: state.lastActiveConnectionId, // NEW: Persist to localStorage
    autoConnectEnabled: state.autoConnectEnabled,
    activeEnvironmentFilter: state.activeEnvironmentFilter,
  }),
  // ...
}
```

### 6. Connection Store - Initialization Function
```typescript
/**
 * Initialize connection store and auto-connect to last active connection
 * Call this on app startup after store hydration
 */
export async function initializeConnectionStore() {
  const state = useConnectionStore.getState()

  // Guard: Check if auto-connect is enabled
  if (!state.autoConnectEnabled) {
    console.debug('Auto-connect is disabled')
    return
  }

  // Guard: Get last active connection
  const lastConnectionId = state.lastActiveConnectionId
  if (!lastConnectionId) {
    console.debug('No last active connection found')
    return
  }

  // Guard: Find the connection
  const connection = state.connections.find(c => c.id === lastConnectionId)
  if (!connection) {
    console.debug('Last active connection no longer exists:', lastConnectionId)
    return
  }

  // Guard: Check if already connected
  if (connection.isConnected) {
    console.debug('Connection already active:', connection.name)
    return
  }

  // Auto-connect in background
  console.debug('Auto-connecting to:', connection.name)
  try {
    await state.connectToDatabase(lastConnectionId)
    console.debug('Auto-connect successful:', connection.name)
  } catch (error) {
    console.warn('Auto-connect failed:', connection.name, error)
    // Fail silently - don't block app startup
  }
}
```

### 7. App.tsx - Integration
**File**: `/Users/jacob_1/projects/sql-studio/frontend/src/app.tsx`

```typescript
import { initializeConnectionStore } from './store/connection-store'

function App() {
  useEffect(() => {
    // Initialize stores
    initializeAuthStore()
    initializeTierStore()
    initializeOrganizationStore().catch(/* ... */)

    // Migrate credentials
    import('./lib/migrate-credentials').then(/* ... */)

    // NEW: Auto-connect to last active connection
    // Add small delay to ensure store hydration is complete
    const autoConnectTimer = setTimeout(() => {
      initializeConnectionStore().catch(err => {
        console.error('Auto-connect failed:', err)
        // App continues normally even if auto-connect fails
      })
    }, 100)

    return () => clearTimeout(autoConnectTimer)
  }, [])

  // ... rest of app
}
```

## Usage Examples

### Example 1: Check Auto-Connect Status
```typescript
import { useConnectionStore } from './store/connection-store'

function ConnectionSettings() {
  const { autoConnectEnabled, lastActiveConnectionId } = useConnectionStore()

  return (
    <div>
      <p>Auto-connect: {autoConnectEnabled ? 'Enabled' : 'Disabled'}</p>
      <p>Last active: {lastActiveConnectionId || 'None'}</p>
    </div>
  )
}
```

### Example 2: Toggle Auto-Connect
```typescript
import { useConnectionStore } from './store/connection-store'

function AutoConnectToggle() {
  const { autoConnectEnabled, setAutoConnect } = useConnectionStore()

  return (
    <button onClick={() => setAutoConnect(!autoConnectEnabled)}>
      {autoConnectEnabled ? 'Disable' : 'Enable'} Auto-Connect
    </button>
  )
}
```

### Example 3: Clear Last Active Connection
```typescript
import { useConnectionStore } from './store/connection-store'

function ClearAutoConnect() {
  const { setActiveConnection } = useConnectionStore()

  return (
    <button onClick={() => setActiveConnection(null)}>
      Clear Auto-Connect
    </button>
  )
}
```

### Example 4: Programmatic Auto-Connect
```typescript
import { useConnectionStore } from './store/connection-store'

async function connectToLastActive() {
  const state = useConnectionStore.getState()
  const lastId = state.lastActiveConnectionId

  if (lastId) {
    try {
      await state.connectToDatabase(lastId)
      console.log('Reconnected successfully')
    } catch (error) {
      console.error('Failed to reconnect:', error)
    }
  }
}
```

## Testing Snippets

### Manual Test in Browser Console
```javascript
// Check current state
const state = window.__connectionStore?.getState()
console.log('Auto-connect enabled:', state?.autoConnectEnabled)
console.log('Last active ID:', state?.lastActiveConnectionId)
console.log('Active connection:', state?.activeConnection?.name)

// Disable auto-connect
window.__connectionStore?.getState().setAutoConnect(false)

// Enable auto-connect
window.__connectionStore?.getState().setAutoConnect(true)

// Force auto-connect now
import('/src/store/connection-store').then(module => {
  module.initializeConnectionStore()
})
```

### Simulate Fresh Start
```javascript
// Clear localStorage and reload
localStorage.removeItem('connection-store')
location.reload()
```

### Check Persistence
```javascript
// View persisted data
const stored = JSON.parse(localStorage.getItem('connection-store') || '{}')
console.log('Persisted state:', stored.state)
console.log('Last active ID:', stored.state?.lastActiveConnectionId)
```

## Common Patterns

### Pattern 1: Conditional Auto-Connect Based on Environment
```typescript
export async function initializeConnectionStore() {
  const state = useConnectionStore.getState()

  // Only auto-connect in development
  if (import.meta.env.PROD && !state.autoConnectEnabled) {
    return
  }

  // ... rest of auto-connect logic
}
```

### Pattern 2: Auto-Connect with User Notification
```typescript
export async function initializeConnectionStore() {
  const state = useConnectionStore.getState()

  if (!state.autoConnectEnabled || !state.lastActiveConnectionId) {
    return
  }

  const connection = state.connections.find(c => c.id === state.lastActiveConnectionId)
  if (!connection || connection.isConnected) {
    return
  }

  try {
    await state.connectToDatabase(state.lastActiveConnectionId)

    // Show success toast
    window.dispatchEvent(new CustomEvent('showToast', {
      detail: {
        message: `Reconnected to ${connection.name}`,
        variant: 'success'
      }
    }))
  } catch (error) {
    console.warn('Auto-connect failed:', error)

    // Optional: Show warning toast
    window.dispatchEvent(new CustomEvent('showToast', {
      detail: {
        message: `Failed to reconnect to ${connection.name}`,
        variant: 'warning'
      }
    }))
  }
}
```

### Pattern 3: Auto-Connect to Multiple Connections
```typescript
export async function initializeConnectionStore() {
  const state = useConnectionStore.getState()

  if (!state.autoConnectEnabled) {
    return
  }

  // Get all previously connected connections
  const previouslyConnected = state.connections.filter(c => c.lastUsed)

  // Sort by most recently used
  const sorted = previouslyConnected.sort((a, b) =>
    (b.lastUsed?.getTime() ?? 0) - (a.lastUsed?.getTime() ?? 0)
  )

  // Connect to top 3 most recent
  const toConnect = sorted.slice(0, 3)

  await Promise.allSettled(
    toConnect.map(conn => state.connectToDatabase(conn.id))
  )
}
```

## Debug Helpers

### Enable Debug Logging
```typescript
// Add to initializeConnectionStore() for verbose logging
const DEBUG = true

if (DEBUG) {
  console.log('=== Auto-Connect Debug ===')
  console.log('Enabled:', state.autoConnectEnabled)
  console.log('Last ID:', state.lastActiveConnectionId)
  console.log('All connections:', state.connections.map(c => ({
    id: c.id,
    name: c.name,
    isConnected: c.isConnected
  })))
}
```

### Performance Monitoring
```typescript
export async function initializeConnectionStore() {
  const start = performance.now()

  // ... auto-connect logic

  const end = performance.now()
  console.log(`Auto-connect took ${end - start}ms`)
}
```
