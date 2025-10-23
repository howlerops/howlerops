# Multi-Tab Sync Integration Guide

Step-by-step guide to integrate multi-tab synchronization into SQL Studio stores.

## Prerequisites

Ensure the sync system is initialized in your root component:

```typescript
// App.tsx
import { useEffect } from 'react'
import { initializeSyncRegistry } from '@/lib/sync'

function App() {
  useEffect(() => {
    initializeSyncRegistry()
  }, [])

  return <YourApp />
}
```

## Store Integration Examples

### 1. Connection Store

**File**: `frontend/src/store/connection-store.ts`

**Changes**:

```typescript
import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { broadcastSync } from '@/lib/sync'

export const useConnectionStore = create<ConnectionState>()(
  devtools(
    broadcastSync('connection-store', {
      excludeFields: [
        'password',
        'sessionId',
        'isConnecting',
        'activeConnection',
        'sshTunnel.password',
        'sshTunnel.privateKey'
      ],
      debounceMs: 100
    })(
      persist(
        (set, get) => ({
          // ... existing implementation
        }),
        {
          name: 'connection-store',
          // ... existing persist config
        }
      )
    )
  )
)
```

**What gets synced**:
- Connection list (name, host, port, type, etc.)
- Auto-connect settings
- Environment filters

**What doesn't get synced**:
- Passwords (kept in sessionStorage)
- Session IDs (tab-specific)
- Connection state (isConnecting)
- Active connection (tab-specific)

### 2. Query Store

**File**: `frontend/src/store/query-store.ts`

**Changes**:

```typescript
import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { broadcastSync } from '@/lib/sync'

export const useQueryStore = create<QueryState>()(
  devtools(
    broadcastSync('query-store', {
      excludeFields: [
        'results',
        'isExecuting',
        'activeTabId'
      ],
      debounceMs: 200
    })(
      persist(
        (set, get) => ({
          // ... existing implementation
        }),
        {
          name: 'query-store',
          // ... existing persist config
        }
      )
    )
  )
)
```

**What gets synced**:
- Query tabs (title, content, type)
- Tab connection assignments
- Environment snapshots

**What doesn't get synced**:
- Query results (too large)
- Execution state (tab-specific)
- Active tab ID (tab-specific)

### 3. Tier Store

**File**: `frontend/src/store/tier-store.ts`

**Changes**:

```typescript
import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { broadcastSync } from '@/lib/sync'

export const useTierStore = create<TierStore>()(
  devtools(
    broadcastSync('tier-store', {
      excludeFields: ['isInitialized'],
      debounceMs: 500
    })(
      persist(
        (set, get) => ({
          // ... existing implementation
        }),
        {
          name: 'sql-studio-tier-storage',
          // ... existing persist config
        }
      )
    ),
    {
      name: 'TierStore',
      enabled: import.meta.env.DEV,
    }
  )
)
```

**What gets synced**:
- Current tier
- License key
- Expiration date
- Team information

**What doesn't get synced**:
- Initialization state (tab-specific)

**Important**: When tier changes, broadcast the event:

```typescript
activateLicense: async (key: string) => {
  const validation = await validateLicenseKey(key)

  if (validation.valid && validation.tier) {
    set({
      currentTier: validation.tier,
      licenseKey: key,
      // ...
    })

    // Broadcast tier change to all tabs
    const { broadcastTierChanged } = await import('@/lib/sync')
    broadcastTierChanged(validation.tier)
  }

  // ...
}
```

### 4. AI Memory Store

**File**: `frontend/src/store/ai-memory-store.ts`

**Changes**:

```typescript
import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import { broadcastSync } from '@/lib/sync'

export const useAIMemoryStore = create<AIMemoryState>()(
  devtools(
    broadcastSync('ai-memory-store', {
      excludeFields: [
        'fullTranscripts',
        'isLoading',
        '__broadcastSync'
      ],
      debounceMs: 500
    })(
      persist(
        (set, get) => ({
          // ... existing implementation
        }),
        {
          name: 'ai-memory-store',
          // ... existing persist config
        }
      )
    )
  )
)
```

**What gets synced**:
- Session metadata
- Session summaries
- Settings

**What doesn't get synced**:
- Full transcripts (too large)
- Loading state (transient)

## Event Broadcasting

### Connection Added

When a new connection is added, broadcast to other tabs:

```typescript
// In connection-store.ts addConnection method
addConnection: (connectionData) => {
  const newConnection = {
    // ... create connection
  }

  set((state) => ({
    connections: [...state.connections, newConnection]
  }))

  // Broadcast to other tabs
  const { broadcastConnectionAdded } = await import('@/lib/sync')
  broadcastConnectionAdded(newConnection.id)
}
```

### Logout

When user logs out, broadcast to all tabs:

```typescript
// In your auth/logout handler
const handleLogout = async () => {
  // Clear local state
  clearAuthState()

  // Broadcast logout to all tabs
  const { broadcastLogout } = await import('@/lib/sync')
  broadcastLogout()

  // Redirect
  navigate('/login')
}

// Listen for logout broadcasts from other tabs
useEffect(() => {
  const handleMultiTabLogout = (event: CustomEvent) => {
    console.log('Logout broadcast received from', event.detail.senderId)
    clearAuthState()
    navigate('/login')
  }

  window.addEventListener('multi-tab-logout', handleMultiTabLogout as EventListener)

  return () => {
    window.removeEventListener('multi-tab-logout', handleMultiTabLogout as EventListener)
  }
}, [])
```

## UI Integration

### Add Multi-Tab Indicator

**File**: `frontend/src/components/layout/header.tsx` or similar

```typescript
import { MultiTabIndicator } from '@/components/multi-tab-indicator'
import { useMultiTabSync } from '@/hooks/use-multi-tab-sync'
import { useConnectionStore } from '@/store/connection-store'

function Header() {
  const { requestPasswordShare } = useMultiTabSync()
  const connections = useConnectionStore(state => state.connections)

  const handleRequestPasswordShare = () => {
    const connectionIds = connections.map(c => c.id)
    requestPasswordShare(connectionIds)
  }

  return (
    <header>
      {/* ... other header content */}

      <MultiTabIndicator
        onRequestPasswordShare={handleRequestPasswordShare}
        connectionIdsToShare={connections.map(c => c.id)}
      />
    </header>
  )
}
```

### Add Password Share Dialog

**File**: `frontend/src/components/layout/root-layout.tsx` or `App.tsx`

```typescript
import { PasswordShareDialog } from '@/components/password-share-dialog'
import { useMultiTabSync } from '@/hooks/use-multi-tab-sync'
import { useConnectionStore } from '@/store/connection-store'

function RootLayout() {
  const {
    passwordShareRequest,
    approvePasswordShare,
    denyPasswordShare
  } = useMultiTabSync()

  const connections = useConnectionStore(state => state.connections)

  return (
    <div>
      {/* ... your layout */}

      <PasswordShareDialog
        request={passwordShareRequest}
        onApprove={approvePasswordShare}
        onDeny={denyPasswordShare}
        connections={connections}
      />
    </div>
  )
}
```

## Password Handling

### On App Load (New Tab)

When a new tab loads and connections exist but passwords are missing:

```typescript
// In your connection initialization code
useEffect(() => {
  const connections = useConnectionStore.getState().connections
  const secureStorage = getSecureStorage()

  // Check if we have connections but no passwords
  const missingPasswords = connections.filter(conn =>
    !secureStorage.hasCredentials(conn.id)
  )

  if (missingPasswords.length > 0) {
    // Show prompt to request passwords
    setShowPasswordPrompt(true)
    setMissingConnectionIds(missingPasswords.map(c => c.id))
  }
}, [])
```

### Password Received Handler

Listen for password broadcasts:

```typescript
import { useBroadcastEvent } from '@/hooks/use-multi-tab-sync'

function ConnectionManager() {
  useBroadcastEvent('passwords-received', (detail: { passwords: PasswordData[] }) => {
    console.log('Received passwords for', detail.passwords.length, 'connections')

    // Passwords are already stored in secure storage by the hook
    // Optionally trigger connection attempts
    detail.passwords.forEach(pwd => {
      const { connectToDatabase } = useConnectionStore.getState()
      connectToDatabase(pwd.connectionId)
    })
  })

  // ...
}
```

## Testing Integration

### Test Checklist

1. **Basic Sync**
   - [ ] Open two tabs
   - [ ] Modify connection in Tab 1
   - [ ] Verify Tab 2 updates
   - [ ] Verify passwords NOT synced

2. **Password Transfer**
   - [ ] Open Tab 1 with connections + passwords
   - [ ] Open Tab 2 (new tab, no passwords)
   - [ ] Click "Request Passwords" in Tab 2
   - [ ] Approve in Tab 1
   - [ ] Verify Tab 2 receives passwords
   - [ ] Verify Tab 2 can connect

3. **Tab Lifecycle**
   - [ ] Open 3 tabs
   - [ ] Verify oldest is primary (crown icon)
   - [ ] Close primary tab
   - [ ] Verify next oldest becomes primary

4. **Logout**
   - [ ] Open multiple tabs
   - [ ] Logout in one tab
   - [ ] Verify all tabs logout
   - [ ] Verify secure storage cleared

5. **Tier Changes**
   - [ ] Open multiple tabs
   - [ ] Activate license in one tab
   - [ ] Verify all tabs update tier
   - [ ] Check feature availability syncs

### Debug Commands

Open browser console and run:

```javascript
// Check sync status
const { getBroadcastSync } = await import('./src/lib/sync/broadcast-sync')
const broadcast = getBroadcastSync()
console.log('Connected:', broadcast.isChannelConnected())
console.log('Tab ID:', broadcast.getTabId())

// Check tab lifecycle
const { getTabLifecycleManager } = await import('./src/lib/sync/tab-lifecycle')
const lifecycle = getTabLifecycleManager()
console.log('Is Primary:', lifecycle.getIsPrimary())
console.log('Tab Count:', lifecycle.getTabCount())
console.log('All Tabs:', lifecycle.getTabs())

// Check store registry
const { getStoreRegistry } = await import('./src/lib/sync/store-registry')
const registry = getStoreRegistry()
console.log('Initialized:', registry.isInitialized())
console.log('Configs:', registry.getAllConfigs())
```

## Migration Notes

### Existing Users

For users upgrading to the multi-tab sync version:

1. Existing localStorage data is preserved
2. Passwords are automatically migrated to sessionStorage on first load
3. No data loss during upgrade
4. Each tab initializes independently

### Breaking Changes

None! The system is fully backwards compatible:
- Existing stores work without modification
- Adding broadcast middleware is opt-in per store
- System gracefully degrades if BroadcastChannel unavailable

## Performance Monitoring

Track sync performance:

```typescript
// Add to your analytics
const { getTabLifecycleManager, getBroadcastSync } = await import('@/lib/sync')

const lifecycle = getTabLifecycleManager()
const broadcast = getBroadcastSync()

analytics.track('multi_tab_sync', {
  tab_count: lifecycle.getTabCount(),
  is_primary: lifecycle.getIsPrimary(),
  is_connected: broadcast.isChannelConnected(),
  pending_messages: broadcast.getPendingMessageCount()
})
```

## Troubleshooting

### Common Issues

**Issue**: Changes not syncing between tabs
- Check browser console for errors
- Verify store is registered in registry
- Check `excludeFields` configuration
- Ensure middleware order is correct

**Issue**: Password transfer fails
- Verify HTTPS or localhost (required for crypto)
- Check browser supports Web Crypto API
- Ensure approval dialog appears
- Check console for encryption errors

**Issue**: Performance degradation
- Increase debounce time
- Add more fields to `excludeFields`
- Check message sizes
- Monitor broadcast frequency

## Support

For issues or questions:
1. Check the main [README](./README.md)
2. Review browser console logs
3. Test with debug commands above
4. Check browser compatibility

## Next Steps

1. Implement store integrations as shown above
2. Add UI components to header/layout
3. Test all scenarios from checklist
4. Monitor performance in production
5. Gather user feedback on sync experience
