# Multi-Tab Synchronization System

Comprehensive multi-tab synchronization for Howlerops using the BroadcastChannel API.

## Features

- **Real-time State Sync**: Automatic synchronization of Zustand stores across browser tabs
- **Tab Lifecycle Management**: Track active tabs with heartbeat mechanism and primary tab election
- **Secure Password Sharing**: Ephemeral AES-256 encryption for password transfer between tabs
- **Type-Safe Messaging**: Fully typed BroadcastChannel communication
- **Conflict Resolution**: Last-write-wins merge strategy with customizable mergers
- **Performance Optimized**: Debounced broadcasts, selective field sync, message retry logic

## Browser Support

| Browser | Version | Support |
|---------|---------|---------|
| Chrome  | 54+     | ✅       |
| Firefox | 38+     | ✅       |
| Safari  | 15.4+   | ✅       |
| Edge    | 79+     | ✅       |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Multi-Tab Sync System                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐      ┌──────────────────┐            │
│  │  BroadcastSync   │◄────►│  Tab Lifecycle   │            │
│  │  (Core Channel)  │      │  (Heartbeat)     │            │
│  └────────┬─────────┘      └──────────────────┘            │
│           │                                                  │
│           ├──► Store Registry (Config Management)           │
│           │                                                  │
│           ├──► Zustand Middleware (State Sync)              │
│           │                                                  │
│           └──► Password Transfer (Secure Sharing)           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Initialize the System

In your root `App.tsx`:

```typescript
import { useEffect } from 'react'
import { initializeSyncRegistry } from '@/lib/sync'

function App() {
  useEffect(() => {
    // Initialize multi-tab sync on app mount
    initializeSyncRegistry()
  }, [])

  return <YourApp />
}
```

### 2. Add Middleware to Stores

Update your Zustand stores to use broadcast sync:

```typescript
import { create } from 'zustand'
import { persist, devtools } from 'zustand/middleware'
import { broadcastSync } from '@/lib/sync'

interface MyStore {
  count: number
  increment: () => void
}

export const useMyStore = create<MyStore>()(
  devtools(
    broadcastSync('my-store', {
      excludeFields: ['privateData'],
      debounceMs: 100
    })(
      persist(
        (set) => ({
          count: 0,
          increment: () => set((state) => ({ count: state.count + 1 }))
        }),
        { name: 'my-store' }
      )
    )
  )
)
```

**Important**: Place `broadcastSync` between `devtools` and `persist` for correct operation.

### 3. Use Multi-Tab Hook in Components

```typescript
import { useMultiTabSync } from '@/hooks/use-multi-tab-sync'
import { MultiTabIndicator } from '@/components/multi-tab-indicator'

function MyComponent() {
  const {
    isConnected,
    tabCount,
    isPrimaryTab,
    requestPasswordShare,
    broadcastLogout
  } = useMultiTabSync()

  return (
    <div>
      <MultiTabIndicator />
      {isPrimaryTab && <p>This is the primary tab</p>}
      <button onClick={broadcastLogout}>Logout All Tabs</button>
    </div>
  )
}
```

## Core Components

### BroadcastSync

Low-level BroadcastChannel wrapper with type safety and retry logic.

```typescript
import { getBroadcastSync } from '@/lib/sync'

const broadcast = getBroadcastSync()

// Listen for messages
const unsubscribe = broadcast.on('store-update', (message) => {
  console.log('Store updated:', message.storeName)
})

// Send messages
broadcast.send({
  type: 'store-update',
  storeName: 'my-store',
  patch: { count: 42 },
  senderId: broadcast.getTabId(),
  timestamp: Date.now()
})
```

### Tab Lifecycle Manager

Tracks active tabs and implements primary tab election.

```typescript
import { getTabLifecycleManager } from '@/lib/sync'

const lifecycle = getTabLifecycleManager()

// Get current tab ID
const tabId = lifecycle.getTabId()

// Check if primary
const isPrimary = lifecycle.getIsPrimary()

// Listen for tabs changed
lifecycle.onTabsChanged((tabs) => {
  console.log('Active tabs:', tabs.size)
})

// Listen for primary changed
lifecycle.onPrimaryChanged((isPrimary, primaryTabId) => {
  console.log('Primary tab:', primaryTabId)
})
```

**Heartbeat Configuration**:
- Interval: 10 seconds
- Stale timeout: 30 seconds (no heartbeat)
- Primary election: Oldest active tab

### Password Transfer

Secure password sharing between tabs using ephemeral encryption.

```typescript
import { getPasswordTransferManager } from '@/lib/sync'

const passwordTransfer = getPasswordTransferManager()

// Request passwords from other tabs
await passwordTransfer.requestPasswordShare(['conn-1', 'conn-2'])

// Listen for password requests (in existing tabs)
passwordTransfer.onPasswordRequest(
  (connectionIds, requesterId, approve, deny) => {
    // Show approval dialog to user
    if (userApproves) {
      const passwords = getPasswordsForConnections(connectionIds)
      approve(passwords)
    } else {
      deny()
    }
  }
)

// Listen for received passwords (in requesting tab)
passwordTransfer.onPasswordReceived((passwords) => {
  passwords.forEach(pwd => {
    storePasswordSecurely(pwd.connectionId, pwd.password)
  })
})
```

**Security Features**:
- Ephemeral AES-256-GCM keys
- 10-second key lifetime
- User approval required
- Encrypted in transit between tabs

### Store Registry

Central configuration for syncable stores.

```typescript
import { getStoreRegistry } from '@/lib/sync'

const registry = getStoreRegistry()

// Get store configuration
const config = registry.getConfig('connection-store')

// Check if store should sync
const shouldSync = registry.shouldSync('my-store')

// Broadcast events
registry.broadcastLogout()
registry.broadcastConnectionAdded('conn-id')
registry.broadcastTierChanged('team')
```

## Message Protocol

### Message Types

```typescript
type BroadcastMessage =
  | { type: 'store-update', storeName: string, patch: any, senderId: string, timestamp: number }
  | { type: 'logout', senderId: string, timestamp: number }
  | { type: 'sync-complete', timestamp: number, senderId: string }
  | { type: 'connection-added', connectionId: string, senderId: string, timestamp: number }
  | { type: 'tier-changed', newTier: string, senderId: string, timestamp: number }
  | { type: 'password-share-request', connectionId: string, requesterId: string, timestamp: number }
  | { type: 'password-share-response', connectionId: string, encryptedPassword: string, key: string, iv: string, senderId: string, timestamp: number }
  | { type: 'tab-alive', tabId: string, timestamp: number, isPrimary?: boolean }
  | { type: 'tab-closed', tabId: string, timestamp: number }
  | { type: 'request-password-share', connectionIds: string[], requesterId: string, timestamp: number }
```

### Message Flow

```
Tab A                          Tab B
  │                             │
  ├─── store-update ───────────►│ (receives & applies)
  │                             │
  │◄─── tab-alive ──────────────┤ (heartbeat)
  │                             │
  ├─── request-password-share ─►│ (shows approval dialog)
  │                             │
  │◄─── password-share-response─┤ (encrypted data)
  │                             │
```

## Store Synchronization

### Synchronized Stores

| Store | Fields Excluded | Priority |
|-------|----------------|----------|
| connection-store | password, sessionId, isConnecting, activeConnection, sshTunnel.password | high |
| query-store | results, isExecuting, activeTabId | medium |
| tier-store | isInitialized | high |
| ai-memory-store | fullTranscripts, isLoading | low |

### Excluded Stores

- `secrets-store`: Contains sensitive data
- `schema-store`: Large data, database-specific
- `json-viewer-store`: Transient UI state
- `ai-query-agent-store`: Session-specific
- `ai-store`: Session-specific with large data

### Custom Store Configuration

```typescript
import { getStoreRegistry } from '@/lib/sync'

const registry = getStoreRegistry()

registry.registerStore({
  name: 'my-custom-store',
  enabled: true,
  excludeFields: ['sensitiveField', 'largeData'],
  debounceMs: 200,
  description: 'My custom store description',
  priority: 'medium'
})
```

## UI Components

### MultiTabIndicator

Visual indicator showing sync status and active tabs.

```typescript
import { MultiTabIndicator } from '@/components/multi-tab-indicator'

<MultiTabIndicator
  onRequestPasswordShare={() => requestPasswordShare(connectionIds)}
  connectionIdsToShare={['conn-1']}
  showDetails={true}
  compact={false}
/>
```

**Features**:
- Connection status icon (Wifi/WifiOff)
- Active tab count badge
- Primary tab crown indicator
- Password share request button
- Detailed popover with tab list

### PasswordShareDialog

Dialog for approving password sharing requests.

```typescript
import { PasswordShareDialog } from '@/components/password-share-dialog'

<PasswordShareDialog
  request={passwordShareRequest}
  onApprove={approvePasswordShare}
  onDeny={denyPasswordShare}
  connections={connections}
/>
```

**Features**:
- Connection list with names
- Approve/Deny buttons
- Progress indicator
- Security notice
- Auto-dismiss on success

## Performance Considerations

### Message Size

- Target: < 1KB per broadcast
- Exclude large data (query results, transcripts)
- Use selective field sync

### Broadcast Latency

- Typical: < 10ms
- Debounce: 50-500ms depending on store
- Retry logic: 3 attempts with exponential backoff

### Heartbeat Interval

- Interval: 10 seconds
- Stale timeout: 30 seconds
- Minimal overhead (~100 bytes per heartbeat)

### Memory Usage

- Each tab maintains:
  - Active tab map (~1KB)
  - Message queue (dynamic)
  - Ephemeral keys (auto-cleanup)

## Testing Scenarios

### Basic Sync Test

1. Open two tabs
2. Change theme in Tab 1
3. Verify Tab 2 updates immediately
4. Close Tab 1
5. Verify Tab 2 becomes primary

### Connection Sync Test

1. Open Tab 1 with existing connections
2. Open Tab 2 (no passwords)
3. Click "Request Passwords" in Tab 2
4. Approve in Tab 1
5. Verify Tab 2 receives passwords
6. Add new connection in Tab 2
7. Verify Tab 1 sees new connection

### Tier Change Test

1. Open multiple tabs
2. Activate license in one tab
3. Verify all tabs update tier status
4. Check feature availability syncs

### Logout Test

1. Open multiple tabs
2. Logout in one tab
3. Verify all tabs logout simultaneously
4. Check sessionStorage cleared

### Primary Tab Election

1. Open 3 tabs (A, B, C)
2. Verify oldest (A) is primary
3. Close A
4. Verify B becomes primary
5. Close B
6. Verify C becomes primary

## Advanced Usage

### Custom Merge Strategy

```typescript
import { broadcastSync } from '@/lib/sync'

const customMerger = (currentState, incomingPatch) => {
  // Implement custom conflict resolution
  if (incomingPatch.timestamp > currentState.timestamp) {
    return { ...currentState, ...incomingPatch }
  }
  return currentState
}

const useStore = create(
  broadcastSync('store', {
    merger: customMerger
  })(/* store config */)
)
```

### Broadcasting Custom Events

```typescript
import { broadcastAction, onBroadcastAction } from '@/lib/sync'

// Send custom action
broadcastAction('custom-event', { data: 'value' })

// Listen for custom action
const unsubscribe = onBroadcastAction('custom-event', (payload) => {
  console.log('Received:', payload)
})
```

### Conditional Sync

```typescript
import { getStoreRegistry } from '@/lib/sync'

const registry = getStoreRegistry()

// Disable sync for a store temporarily
registry.updateConfig('my-store', { enabled: false })

// Re-enable later
registry.updateConfig('my-store', { enabled: true })
```

## Debugging

### Enable Debug Logging

All components log to console with prefixes:
- `[BroadcastSync]`: Core broadcast operations
- `[TabLifecycle]`: Tab management
- `[PasswordTransfer]`: Password sharing
- `[StoreRegistry]`: Store registration

### Check Connection Status

```typescript
import { getBroadcastSync } from '@/lib/sync'

const broadcast = getBroadcastSync()
console.log('Connected:', broadcast.isChannelConnected())
console.log('Pending messages:', broadcast.getPendingMessageCount())
```

### Monitor Tab State

```typescript
import { getTabLifecycleManager } from '@/lib/sync'

const lifecycle = getTabLifecycleManager()
console.log('Tab ID:', lifecycle.getTabId())
console.log('Is Primary:', lifecycle.getIsPrimary())
console.log('Tab Count:', lifecycle.getTabCount())
console.log('All Tabs:', lifecycle.getTabs())
```

## Troubleshooting

### Messages Not Syncing

1. Check if BroadcastChannel is supported: `typeof BroadcastChannel !== 'undefined'`
2. Verify store is registered in registry
3. Check `excludeFields` configuration
4. Ensure middleware is in correct position (between devtools and persist)

### Password Sharing Fails

1. Check if Web Crypto API is available: `typeof crypto.subtle !== 'undefined'`
2. Verify HTTPS or localhost (required for crypto operations)
3. Check browser console for encryption errors
4. Ensure secure storage is initialized

### Primary Tab Not Elected

1. Check heartbeat interval (should see logs every 10s)
2. Verify tabs are sending heartbeats
3. Check for stale tab cleanup (30s timeout)
4. Monitor tab lifecycle events

### Performance Issues

1. Reduce debounce time if updates are too slow
2. Increase debounce time if too many messages
3. Add more fields to `excludeFields`
4. Check message sizes in network tab

## Security Considerations

### Password Transfer

- Ephemeral keys valid for 10 seconds only
- AES-256-GCM encryption
- User approval required
- Keys never persisted
- Auto-cleanup on expiration

### Data Exclusion

- Never sync passwords or credentials
- Exclude session IDs and tokens
- Keep sensitive data in sessionStorage
- Use `excludeFields` liberally

### Tab Verification

- User controls password sharing
- Visual confirmation in both tabs
- Clear security notices in UI
- Deny by default

## API Reference

See individual module documentation:
- [BroadcastSync API](./broadcast-sync.ts)
- [Zustand Middleware API](./zustand-broadcast-middleware.ts)
- [Tab Lifecycle API](./tab-lifecycle.ts)
- [Password Transfer API](./password-transfer.ts)
- [Store Registry API](./store-registry.ts)

## License

Part of Howlerops. See main LICENSE file.
