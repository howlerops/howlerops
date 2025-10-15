# Auto Schema Loading Strategy - Implementation Plan

## Problem Statement

Currently, users must manually click each database connection to:
1. Establish a connection
2. Load the schema
3. Enable autocomplete and query features

This creates friction when working with multiple databases, especially for multi-DB queries.

## Research Findings

### Current Flow
1. User adds connection via `ConnectionManager` → stored in Zustand store
2. Connection is NOT automatically connected (`isConnected: false`)
3. User must click connection in Sidebar → triggers `connectToDatabase()`
4. Only then is schema loaded via `useSchemaIntrospection()` hook
5. Schema is per-active-connection, not all connections

### Key Files
- `frontend/src/store/connection-store.ts` - Connection state management
- `frontend/src/components/layout/sidebar.tsx` - Connection UI with manual connect
- `frontend/src/hooks/useSchemaIntrospection.ts` - Schema loading hook
- `services/database.go` - Backend connection management
- `backend-go/pkg/database/schema_cache.go` - Schema caching (520x faster!)

### Constraints
- We already have smart schema caching (detects changes via migrations table + table hash)
- Backend is fast: cache hit = ~2ms, cache miss = ~1s
- Schemas are connection-specific in frontend state
- Multi-DB mode needs schemas for ALL connections simultaneously

## Solution Design

### 🎯 Goals
1. **Auto-connect on add**: When user adds connection, immediately test + connect + load schema
2. **Background refresh**: Periodically check for schema changes (use existing cache!)
3. **Multi-DB ready**: Load all connection schemas when 2+ connections exist
4. **Minimal overhead**: Leverage existing schema cache for performance
5. **Error resilience**: Failed connections don't block UI

### 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Connection Lifecycle                                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Add Connection                                          │
│     └─> Test (validate credentials)                         │
│         └─> Connect (establish session)                     │
│             └─> Load Schema (initial fetch)                 │
│                 └─> Cache in backend                        │
│                     └─> Store in frontend                   │
│                                                             │
│  2. Background Refresh (every 60s or manual)                │
│     └─> For each connected connection:                      │
│         └─> Check cache validity (backend)                  │
│             ├─> Valid: No-op (fast!)                        │
│             └─> Invalid: Fetch fresh schema                 │
│                                                             │
│  3. Multi-DB Mode Activation (2+ connections)               │
│     └─> Ensure all connections are connected                │
│         └─> Load schemas for all connections                │
│             └─> Store in multi-DB schema map                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 📦 Components

#### 1. Backend: Connection Auto-Management
**File**: `services/database.go`

Add method to auto-initialize connection:
```go
// InitializeConnection connects and loads schema in one call
func (s *DatabaseService) InitializeConnection(connectionID string) (*ConnectionInfo, error) {
    // 1. Test connection
    if err := s.TestConnection(config); err != nil {
        return nil, err
    }
    
    // 2. Create connection
    conn, err := s.CreateConnection(config)
    if err != nil {
        return nil, err
    }
    
    // 3. Pre-load schema (triggers cache)
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
    defer cancel()
    
    schemas, err := s.manager.GetMultiConnectionSchema(ctx, []string{connectionID})
    if err != nil {
        s.logger.WithError(err).Warn("Failed to pre-load schema, will load on-demand")
    }
    
    return conn, nil
}
```

#### 2. Backend: Health Check Endpoint
**File**: `app.go`

Add method for checking connection health:
```go
// CheckConnectionsHealth checks if connections are still valid
func (a *App) CheckConnectionsHealth() ([]ConnectionHealth, error) {
    connectionIDs := a.databaseService.ListConnections()
    health := make([]ConnectionHealth, 0, len(connectionIDs))
    
    for _, id := range connectionIDs {
        status := a.databaseService.CheckConnectionHealth(id)
        health = append(health, status)
    }
    
    return health, nil
}

type ConnectionHealth struct {
    ID            string `json:"id"`
    IsAlive       bool   `json:"is_alive"`
    SchemaValid   bool   `json:"schema_valid"`
    LastChecked   int64  `json:"last_checked"`
    Error         string `json:"error,omitempty"`
}
```

#### 3. Frontend: Auto-Connect Hook
**File**: `frontend/src/hooks/useAutoConnect.ts` (NEW)

```typescript
import { useEffect, useRef } from 'react'
import { useConnectionStore } from '@/store/connection-store'

export function useAutoConnect() {
  const { connections, connectToDatabase } = useConnectionStore()
  const connectedRef = useRef(new Set<string>())
  
  useEffect(() => {
    // Auto-connect to any new connections
    connections.forEach(async (conn) => {
      if (!conn.isConnected && !connectedRef.current.has(conn.id)) {
        connectedRef.current.add(conn.id)
        
        try {
          await connectToDatabase(conn.id)
          console.log(`✓ Auto-connected to: ${conn.name}`)
        } catch (error) {
          console.warn(`✗ Failed to auto-connect to ${conn.name}:`, error)
          connectedRef.current.delete(conn.id)
        }
      }
    })
  }, [connections, connectToDatabase])
}
```

#### 4. Frontend: Background Schema Refresh
**File**: `frontend/src/hooks/useSchemaRefresh.ts` (NEW)

```typescript
import { useEffect } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { RefreshSchema } from '@/wailsjs/go/main/App'

export function useSchemaRefresh(intervalMs = 60000) {
  const { connections } = useConnectionStore()
  
  useEffect(() => {
    const interval = setInterval(async () => {
      const connected = connections.filter(c => c.isConnected)
      
      if (connected.length === 0) return
      
      console.log('🔄 Background schema refresh for', connected.length, 'connections')
      
      // Refresh schemas (backend cache will make this fast)
      await Promise.allSettled(
        connected.map(async (conn) => {
          try {
            await RefreshSchema(conn.sessionId!)
          } catch (error) {
            console.debug(`Schema refresh skipped for ${conn.name}`)
          }
        })
      )
    }, intervalMs)
    
    return () => clearInterval(interval)
  }, [connections, intervalMs])
}
```

#### 5. Frontend: Enhanced Connection Store
**File**: `frontend/src/store/connection-store.ts` (MODIFY)

Add auto-connect option:
```typescript
interface ConnectionState {
  // ... existing fields
  autoConnectEnabled: boolean
  setAutoConnect: (enabled: boolean) => void
}

// In addConnection:
addConnection: (connectionData) => {
  const newConnection = { ... }
  
  set((state) => ({
    connections: [...state.connections, newConnection],
  }))
  
  // Auto-connect if enabled
  if (get().autoConnectEnabled) {
    setTimeout(() => get().connectToDatabase(newConnection.id), 100)
  }
}
```

#### 6. Frontend: Multi-DB Schema Loader
**File**: `frontend/src/components/query-editor.tsx` (ENHANCE)

Optimize multi-DB schema loading:
```typescript
const loadMultiDBSchemas = async () => {
  console.log('🔄 Loading multi-DB schemas...')
  
  try {
    // Ensure all connections are connected
    const disconnected = connections.filter(c => !c.isConnected)
    if (disconnected.length > 0) {
      console.log(`⚡ Auto-connecting ${disconnected.length} connections...`)
      await Promise.allSettled(
        disconnected.map(c => connectToDatabase(c.id))
      )
    }
    
    // Now load schemas (will use backend cache!)
    const connectionIds = connections
      .filter(c => c.isConnected && c.sessionId)
      .map(c => c.sessionId!)
      
    if (connectionIds.length === 0) {
      console.warn('⚠️ No connected connections for multi-DB')
      return
    }
    
    // ... rest of existing logic
  } catch (error) {
    console.error('❌ Failed to load multi-DB schemas:', error)
  }
}
```

## Implementation Phases

### Phase 1: Auto-Connect on Add (Week 1)
**Priority: HIGH - Immediate UX improvement**

**Tasks:**
- [ ] Add `InitializeConnection` method to `services/database.go`
- [ ] Create `useAutoConnect` hook in frontend
- [ ] Add `autoConnectEnabled` to connection store (default: true)
- [ ] Update `ConnectionManager` to use new initialize endpoint
- [ ] Test: Add connection → auto-connects → schema loads

**Acceptance Criteria:**
- When user adds connection, it auto-connects within 2-3 seconds
- Schema is available immediately after connection
- Errors are gracefully handled (show toast, don't block)

### Phase 2: Background Refresh (Week 1-2)
**Priority: MEDIUM - Nice to have, leverages existing cache**

**Tasks:**
- [ ] Add `CheckConnectionsHealth` to `app.go`
- [ ] Create `useSchemaRefresh` hook
- [ ] Integrate hook in Dashboard component
- [ ] Add manual "Refresh All Schemas" button to UI
- [ ] Test: Schema updates detected within 60s

**Acceptance Criteria:**
- Schemas refresh every 60s in background
- Uses backend cache (fast, minimal overhead)
- Only refreshes if migrations table or tables changed
- Manual refresh button forces immediate cache invalidation

### Phase 3: Multi-DB Optimization (Week 2)
**Priority: HIGH - Critical for multi-DB queries**

**Tasks:**
- [ ] Enhance `loadMultiDBSchemas` to auto-connect
- [ ] Add connection status indicator in multi-DB UI
- [ ] Pre-load schemas when entering multi-DB mode
- [ ] Add loading states for each connection
- [ ] Test: Multi-DB autocomplete works immediately

**Acceptance Criteria:**
- When 2+ connections exist, all auto-connect
- Multi-DB autocomplete works without manual intervention
- Loading states show which connections are initializing
- Errors don't block other connections from loading

### Phase 4: Settings & Polish (Week 2)
**Priority: LOW - User preferences**

**Tasks:**
- [ ] Add settings page with "Auto-connect" toggle
- [ ] Add "Schema refresh interval" setting (30s, 60s, 5m, manual)
- [ ] Add connection health indicators in sidebar
- [ ] Add "Reconnect" button for failed connections
- [ ] Test: Settings persist across app restarts

**Acceptance Criteria:**
- User can disable auto-connect if desired
- User can customize refresh interval
- Health indicators show connection status
- Failed connections can be manually reconnected

## Configuration

### Backend Config
**File**: `backend-go/configs/config.yaml`

```yaml
database:
  # Connection management
  auto_initialize: true
  health_check_interval: "60s"
  connection_timeout: "30s"
  
  # Schema caching (already exists!)
  schema_cache:
    enabled: true
    ttl: "1h"
    check_migrations: true
```

### Frontend Settings
**File**: `frontend/src/lib/config.ts`

```typescript
export const connectionConfig = {
  autoConnect: true,
  schemaRefreshInterval: 60000, // 60s
  healthCheckInterval: 120000,  // 2m
  maxRetries: 3,
  retryDelay: 5000, // 5s
}
```

## Performance Impact

### Before (Current)
- User adds connection: Instant (no network)
- User clicks connection: 1-2s (establish + load schema)
- Schema refresh: Manual only
- Multi-DB setup: 2-5s per connection (sequential clicking)

### After (Optimized)
- User adds connection: 2-3s (test + connect + load schema)
- User clicks connection: <100ms (already connected!)
- Schema refresh: Automatic every 60s (cache makes it ~2ms)
- Multi-DB setup: 2-3s total (parallel auto-connect)

**Net Result:**
- Initial add is slower (2-3s vs instant)
- Everything else is MUCH faster
- Multi-DB queries work immediately
- Schemas stay fresh automatically

## Edge Cases & Error Handling

### Connection Failures
- **Problem**: Network down, wrong credentials
- **Solution**: Retry with exponential backoff (3 attempts), show error toast, allow manual retry

### Slow Connections
- **Problem**: Large schemas (1000+ tables) take 5-10s
- **Solution**: Show progress indicator, load in background, allow queries before schema loads

### Connection Drops
- **Problem**: Database server restarts, network interruption
- **Solution**: Health check detects failure, auto-reconnect, notify user

### Rate Limiting
- **Problem**: Too many connections, health checks, schema fetches
- **Solution**: Debounce health checks, use exponential backoff, respect backend cache

### Multi-User Scenarios
- **Problem**: Team member changes schema
- **Solution**: Background refresh detects changes via migrations table + table hash

## Testing Strategy

### Unit Tests
- Connection store auto-connect logic
- Schema refresh hook timing
- Health check debouncing
- Error handling and retries

### Integration Tests
- End-to-end connection lifecycle
- Multi-DB mode with auto-connect
- Schema refresh triggers cache invalidation
- Concurrent connections

### Manual Testing Checklist
- [ ] Add connection → auto-connects → schema loads
- [ ] Add multiple connections → all auto-connect in parallel
- [ ] Leave app open → schemas refresh in background
- [ ] Disconnect network → graceful error handling
- [ ] Reconnect network → auto-reconnects
- [ ] Add table to database → detected within 60s
- [ ] Run migration → schema updates automatically
- [ ] Multi-DB query → all schemas available immediately

## Rollout Plan

### Week 1: Foundation
- Implement Phase 1 (auto-connect on add)
- Add backend `InitializeConnection` endpoint
- Create frontend hooks
- Basic error handling

### Week 2: Enhancement
- Implement Phase 2 (background refresh)
- Implement Phase 3 (multi-DB optimization)
- Add health checks
- Comprehensive error handling

### Week 3: Polish
- Implement Phase 4 (settings)
- Add UI indicators
- Performance optimization
- Documentation

### Week 4: Testing & Release
- Integration testing
- Performance benchmarking
- User acceptance testing
- Documentation updates
- Release

## Success Metrics

- **UX**: Users don't need to manually click connections (0 manual clicks for multi-DB)
- **Performance**: Schema loading 10x faster for subsequent connections (cache hit)
- **Reliability**: 99%+ auto-connect success rate
- **Freshness**: Schemas update within 60s of database changes
- **Error Rate**: <1% connection failures after retries

## Open Questions

1. **Should auto-connect be opt-in or opt-out?**
   - Recommendation: Opt-out (default enabled) for best UX
   
2. **What should the default refresh interval be?**
   - Recommendation: 60s (good balance of freshness vs overhead)
   
3. **Should we auto-reconnect on connection failure?**
   - Recommendation: Yes, with exponential backoff (3 retries max)
   
4. **Should we pre-fetch columns for all tables or on-demand?**
   - Current: On-demand via `GetTableStructure`
   - Recommendation: Keep on-demand, but cache aggressively
   
5. **How to handle 10+ connections?**
   - Recommendation: Connect in batches of 3-5, show progress indicator

## Alternative Approaches Considered

### 1. Connect All on App Startup
**Pros**: Everything ready immediately
**Cons**: Slow startup (5-10s for 5 connections), unnecessary if user only uses 1 connection
**Decision**: Rejected - too slow, wasteful

### 2. Connect on First Query
**Pros**: Lazy loading, minimal overhead
**Cons**: Bad UX (query fails, user waits), multi-DB still requires manual setup
**Decision**: Rejected - poor UX

### 3. Manual Mode Only
**Pros**: User has full control
**Cons**: Current problem persists, annoying for multi-DB
**Decision**: Rejected - doesn't solve the problem

### 4. Auto-Connect + Background Refresh (CHOSEN)
**Pros**: Best UX, leverages cache, fresh schemas, works for multi-DB
**Cons**: Slight overhead on add, background polling
**Decision**: **ACCEPTED** - Best balance of UX and performance

## Next Steps

1. ✅ Create this plan
2. ⏭️ Review with team/user
3. ⏭️ Start Phase 1 implementation
4. ⏭️ Iterate based on feedback

---

**Document Version**: 1.0  
**Author**: AI Assistant  
**Date**: 2025-10-14  
**Status**: DRAFT - Awaiting Review
