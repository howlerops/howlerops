# Auto Schema Loading + Default Database - Implementation Plan

## Overview

This document covers two related features:
1. **Auto Schema Loading**: Automatically connect and load schemas for all databases
2. **Default Database**: Allow setting a "primary" database for queries without `@` syntax

## Part 1: Default Database Feature

### Problem
In multi-DB mode, users must use `@connection.table` syntax for every query. But often you're primarily working with one database and only occasionally referencing others.

### Solution: Default/Primary Database

```
User has 3 connections: Production, Staging, Analytics

Set Production as "default"

Query without @:
  SELECT * FROM users               → Uses Production (default)

Query with @:
  SELECT * FROM @staging.users      → Uses Staging (override)

Mixed query:
  SELECT u.name, s.status 
  FROM users u                      → Uses Production (default)
  JOIN @staging.sessions s          → Uses Staging (explicit)
```

### Architecture

```
Connection Store State:
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  defaultConnectionId: string | null        ← NEW!

Query Execution Logic:
  1. Parse query for @ references
  2. If @ found → Use specified connection
  3. If no @ found:
     a. In multi-DB mode → Use defaultConnectionId
     b. In single-DB mode → Use activeConnection
```

### UI Design

#### Sidebar - Connection List
```
┌─────────────────────────────────────┐
│ Connections                         │
├─────────────────────────────────────┤
│ ⭐ Production         [Connected] ✓ │  ← Star = default
│    Staging            [Connected] ✓ │
│    Analytics          [Disconnect]  │
│                                     │
│ [+ Add Connection]                  │
└─────────────────────────────────────┘
```

Click star to set as default. Shows in:
- Sidebar connection list (star icon)
- Query tab (badge: "Default: Production")
- Multi-DB indicator ("Using: Production (default) + 2 others")

### Implementation

#### 1. Backend - No Changes Needed!
The backend already supports executing queries on any connection. The default selection is purely a frontend concern.

#### 2. Frontend - Connection Store

**File**: `frontend/src/store/connection-store.ts`

```typescript
interface ConnectionState {
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  defaultConnectionId: string | null  // NEW
  autoConnectEnabled: boolean          // NEW (Phase 1)
  isConnecting: boolean

  // Existing methods
  addConnection: (connection: Omit<DatabaseConnection, 'id' | 'isConnected' | 'sessionId'>) => void
  updateConnection: (id: string, updates: Partial<DatabaseConnection>) => void
  removeConnection: (id: string) => void
  setActiveConnection: (connection: DatabaseConnection | null) => void
  connectToDatabase: (connectionId: string) => Promise<void>
  disconnectFromDatabase: (connectionId: string) => Promise<void>
  
  // NEW methods
  setDefaultConnection: (connectionId: string | null) => void
  setAutoConnect: (enabled: boolean) => void
  getDefaultConnection: () => DatabaseConnection | null
}
```

Implementation:
```typescript
setDefaultConnection: (connectionId) => {
  // Validate connection exists and is connected
  const conn = get().connections.find(c => c.id === connectionId)
  if (connectionId && (!conn || !conn.isConnected)) {
    console.warn('Cannot set default: connection not found or not connected')
    return
  }
  
  set({ defaultConnectionId: connectionId })
  
  // Emit event for UI updates
  window.dispatchEvent(new CustomEvent('connection:default-changed', {
    detail: { connectionId, connection: conn }
  }))
},

getDefaultConnection: () => {
  const state = get()
  if (!state.defaultConnectionId) return null
  return state.connections.find(c => c.id === state.defaultConnectionId) || null
},
```

Persistence (add to `partialize`):
```typescript
partialize: (state) => ({
  connections: state.connections.map(({ sessionId, isConnected, lastUsed, ...rest }) => rest),
  defaultConnectionId: state.defaultConnectionId,  // NEW
  autoConnectEnabled: state.autoConnectEnabled,     // NEW
}),
```

#### 3. Frontend - Query Execution

**File**: `frontend/src/components/query-editor.tsx`

Update `handleExecuteQuery` to use default connection:

```typescript
const handleExecuteQuery = async () => {
  if (!activeTab) return
  
  const query = editorContent.trim()
  if (!query) return
  
  // Determine which connection to use
  let targetConnectionId: string | undefined
  
  if (mode === 'multi') {
    // Check if query uses @ syntax
    const hasAtSyntax = /@[\w-]+\./.test(query)
    
    if (hasAtSyntax) {
      // Multi-DB query - backend will handle multiple connections
      // Use all connected connections
      targetConnectionId = undefined // Backend handles routing
    } else {
      // No @ syntax - use default connection
      const defaultConn = getDefaultConnection()
      if (!defaultConn?.sessionId) {
        setLastExecutionError('No default connection set. Set a default connection or use @connection.table syntax.')
        return
      }
      targetConnectionId = defaultConn.sessionId
    }
  } else {
    // Single-DB mode - use active connection
    if (!activeConnection?.sessionId) {
      setLastExecutionError('No active connection')
      return
    }
    targetConnectionId = activeConnection.sessionId
  }
  
  // Execute query
  await executeQuery(activeTab.id, query, targetConnectionId)
}
```

#### 4. Frontend - UI Components

**File**: `frontend/src/components/layout/sidebar.tsx`

Add default connection indicator:

```tsx
const Sidebar = () => {
  const {
    connections,
    activeConnection,
    defaultConnectionId,
    setActiveConnection,
    setDefaultConnection,
    connectToDatabase,
    isConnecting,
  } = useConnectionStore()
  
  const handleSetDefault = (connectionId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    
    // Toggle: if already default, unset; otherwise set
    if (defaultConnectionId === connectionId) {
      setDefaultConnection(null)
    } else {
      setDefaultConnection(connectionId)
    }
  }
  
  return (
    <div>
      {connections.map((conn) => (
        <div key={conn.id} className="connection-item">
          {/* Star icon for default */}
          <button
            onClick={(e) => handleSetDefault(conn.id, e)}
            className={cn(
              "star-button",
              defaultConnectionId === conn.id && "active"
            )}
            title={defaultConnectionId === conn.id ? "Default connection" : "Set as default"}
          >
            {defaultConnectionId === conn.id ? <StarFilled /> : <Star />}
          </button>
          
          {/* Connection info */}
          <div onClick={() => handleConnectionSelect(conn)}>
            <Database />
            <span>{conn.name}</span>
            {conn.isConnected && <Badge>Connected</Badge>}
          </div>
        </div>
      ))}
    </div>
  )
}
```

**File**: `frontend/src/components/query-editor.tsx`

Add default connection badge to tab bar:

```tsx
{mode === 'multi' && (
  <div className="multi-db-indicator">
    <Network className="w-4 h-4" />
    <span>Multi-DB Mode</span>
    {defaultConnectionId && (
      <Badge variant="secondary">
        Default: {getDefaultConnection()?.name}
      </Badge>
    )}
  </div>
)}
```

## Part 2: Auto Schema Loading

### Phase 1: Auto-Connect on Add (HIGH PRIORITY)

#### Backend Changes

**File**: `services/database.go`

Add helper method to preload schema:

```go
// PreloadSchema loads and caches schema for a connection
func (s *DatabaseService) PreloadSchema(connectionID string) error {
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
    defer cancel()
    
    // Get schema (will populate cache)
    _, err := s.manager.GetMultiConnectionSchema(ctx, []string{connectionID})
    if err != nil {
        s.logger.WithError(err).Warn("Failed to preload schema, will load on-demand")
        return err
    }
    
    s.logger.WithField("connection_id", connectionID).Info("Schema preloaded successfully")
    return nil
}
```

Modify `CreateConnection` to preload schema:

```go
func (s *DatabaseService) CreateConnection(config database.ConnectionConfig) (*database.Connection, error) {
    connection, err := s.manager.CreateConnection(s.ctx, config)
    if err != nil {
        s.logger.WithError(err).Error("Failed to create database connection")
        return nil, err
    }

    // Preload schema in background (don't block)
    go func() {
        if err := s.PreloadSchema(connection.ID); err != nil {
            s.logger.WithError(err).Warn("Background schema preload failed")
        }
    }()

    // Emit connection created event
    runtime.EventsEmit(s.ctx, "connection:created", map[string]interface{}{
        "id":   connection.ID,
        "type": config.Type,
        "name": config.Database,
    })

    s.logger.WithFields(logrus.Fields{
        "connection_id": connection.ID,
        "type":          config.Type,
        "database":      config.Database,
    }).Info("Database connection created successfully")

    return connection, nil
}
```

#### Frontend Changes

**File**: `frontend/src/store/connection-store.ts`

Add `autoConnectEnabled` state:

```typescript
// In store definition
autoConnectEnabled: true,  // Default to enabled

setAutoConnect: (enabled) => {
  set({ autoConnectEnabled: enabled })
},
```

Modify `addConnection` to auto-connect:

```typescript
addConnection: (connectionData) => {
  const newConnection: DatabaseConnection = {
    ...connectionData,
    id: crypto.randomUUID(),
    isConnected: false,
  }

  set((state) => ({
    connections: [...state.connections, newConnection],
  }))
  
  // Auto-connect if enabled
  if (get().autoConnectEnabled) {
    console.log(`⚡ Auto-connecting to: ${newConnection.name}`)
    
    // Delay slightly to ensure state is updated
    setTimeout(async () => {
      try {
        await get().connectToDatabase(newConnection.id)
        console.log(`✓ Auto-connected to: ${newConnection.name}`)
      } catch (error) {
        console.warn(`✗ Failed to auto-connect to ${newConnection.name}:`, error)
      }
    }, 100)
  }
},
```

### Phase 2: Background Schema Refresh (MEDIUM PRIORITY)

#### Backend Changes

**File**: `app.go`

Add health check endpoint:

```go
type ConnectionHealth struct {
    ID          string `json:"id"`
    IsAlive     bool   `json:"is_alive"`
    SchemaValid bool   `json:"schema_valid"`
    LastChecked int64  `json:"last_checked"`
    Error       string `json:"error,omitempty"`
}

// CheckConnectionsHealth checks all connections and returns health status
func (a *App) CheckConnectionsHealth() ([]ConnectionHealth, error) {
    connectionIDs := a.databaseService.ListConnections()
    health := make([]ConnectionHealth, 0, len(connectionIDs))
    
    for _, id := range connectionIDs {
        status := ConnectionHealth{
            ID:          id,
            LastChecked: time.Now().Unix(),
        }
        
        // Check if connection is alive
        ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
        err := a.databaseService.TestConnectionByID(ctx, id)
        cancel()
        
        if err != nil {
            status.IsAlive = false
            status.Error = err.Error()
        } else {
            status.IsAlive = true
            // Schema validity could be checked here
            status.SchemaValid = true
        }
        
        health = append(health, status)
    }
    
    return health, nil
}
```

**File**: `services/database.go`

Add method to test existing connection:

```go
// TestConnectionByID tests an existing connection
func (s *DatabaseService) TestConnectionByID(ctx context.Context, connectionID string) error {
    // Get connection from manager
    conn := s.manager.GetConnection(connectionID)
    if conn == nil {
        return fmt.Errorf("connection not found: %s", connectionID)
    }
    
    // Simple ping/query to test
    _, err := conn.Execute(ctx, "SELECT 1", nil)
    return err
}
```

#### Frontend Changes

**File**: `frontend/src/hooks/useSchemaRefresh.ts` (NEW)

```typescript
import { useEffect, useRef } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { RefreshSchema } from '../../wailsjs/go/main/App'

export function useSchemaRefresh(intervalMs = 60000) {
  const { connections } = useConnectionStore()
  const lastRefreshRef = useRef<Map<string, number>>(new Map())
  
  useEffect(() => {
    const interval = setInterval(async () => {
      const connected = connections.filter(c => c.isConnected && c.sessionId)
      
      if (connected.length === 0) return
      
      console.log('🔄 Background schema refresh for', connected.length, 'connections')
      
      // Refresh schemas (backend cache makes this fast!)
      const results = await Promise.allSettled(
        connected.map(async (conn) => {
          try {
            await RefreshSchema(conn.sessionId!)
            lastRefreshRef.current.set(conn.id, Date.now())
            console.log(`  ✓ ${conn.name}: Schema refreshed`)
          } catch (error) {
            console.debug(`  ⊘ ${conn.name}: Schema unchanged (cache valid)`)
          }
        })
      )
      
      const successful = results.filter(r => r.status === 'fulfilled').length
      console.log(`🔄 Background refresh complete: ${successful}/${connected.length} successful`)
    }, intervalMs)
    
    return () => clearInterval(interval)
  }, [connections, intervalMs])
  
  return {
    lastRefresh: lastRefreshRef.current,
    forceRefresh: async () => {
      const connected = connections.filter(c => c.isConnected && c.sessionId)
      await Promise.all(connected.map(c => RefreshSchema(c.sessionId!)))
    }
  }
}
```

**File**: `frontend/src/pages/dashboard.tsx`

Integrate the hook:

```typescript
import { useSchemaRefresh } from '@/hooks/useSchemaRefresh'

export function Dashboard() {
  const { mode } = useQueryMode('auto')
  
  // Enable background schema refresh
  const { forceRefresh } = useSchemaRefresh(60000) // 60s interval
  
  return (
    <div>
      <QueryEditor mode={mode} />
      
      {/* Optional: Add manual refresh button */}
      <Button onClick={forceRefresh}>
        <RefreshCw /> Refresh All Schemas
      </Button>
    </div>
  )
}
```

### Phase 3: Multi-DB Optimization (HIGH PRIORITY)

**File**: `frontend/src/components/query-editor.tsx`

Enhance `loadMultiDBSchemas` to auto-connect:

```typescript
const loadMultiDBSchemas = async () => {
  console.log('🔄 Loading multi-DB schemas...', { 
    mode, 
    connectionCount: connections.length,
    connectionIds: connections.map(c => c.id)
  })
  
  try {
    // Step 1: Ensure ALL connections are connected
    const disconnected = connections.filter(c => !c.isConnected)
    
    if (disconnected.length > 0) {
      console.log(`⚡ Auto-connecting ${disconnected.length} connections for multi-DB...`)
      
      const connectResults = await Promise.allSettled(
        disconnected.map(async (conn) => {
          try {
            await connectToDatabase(conn.id)
            console.log(`  ✓ Connected: ${conn.name}`)
          } catch (error) {
            console.warn(`  ✗ Failed: ${conn.name}`, error)
            throw error
          }
        })
      )
      
      const successful = connectResults.filter(r => r.status === 'fulfilled').length
      console.log(`⚡ Auto-connect complete: ${successful}/${disconnected.length} successful`)
    }
    
    // Step 2: Get all connected session IDs
    const connectionIds = connections
      .filter(c => c.isConnected && c.sessionId)
      .map(c => c.sessionId!)
      
    if (connectionIds.length === 0) {
      console.warn('⚠️ No connected connections for multi-DB')
      setMultiDBSchemas(new Map())
      return
    }
    
    // Step 3: Load schemas using GetMultiConnectionSchema (uses cache!)
    const { GetMultiConnectionSchema, GetTableStructure } = await import('../../wailsjs/go/main/App')
    const combined = await GetMultiConnectionSchema(connectionIds)
    
    // ... rest of existing schema loading logic
    
  } catch (error) {
    console.error('❌ Failed to load multi-DB schemas:', error)
    setMultiDBSchemas(new Map())
  }
}
```

## Implementation Order

### Week 1: Foundation
1. ✅ Implement default database feature (frontend only, fast!)
2. ✅ Implement Phase 1 (auto-connect on add)
3. ✅ Test basic flow: add connection → auto-connects → schema loads

### Week 2: Enhancement
4. ✅ Implement Phase 3 (multi-DB optimization)
5. ✅ Test multi-DB: 3+ connections → all auto-connect → @ autocomplete works
6. ✅ Implement Phase 2 (background refresh)
7. ✅ Add manual refresh button

### Week 3: Polish
8. Add settings UI for auto-connect toggle
9. Add settings UI for refresh interval
10. Add connection health indicators
11. Performance optimization

## Testing Checklist

### Default Database
- [ ] Set connection as default → star icon appears
- [ ] Query without @ → uses default connection
- [ ] Query with @ → overrides default
- [ ] Mixed query → default + explicit connections work
- [ ] Unset default → star icon disappears
- [ ] Default persists across app restart

### Auto-Connect
- [ ] Add connection → auto-connects within 2-3s
- [ ] Add connection → schema loads automatically
- [ ] Add multiple connections → all connect in parallel
- [ ] Connection failure → error handled gracefully
- [ ] Disable auto-connect → manual connect still works

### Background Refresh
- [ ] Leave app open → schemas refresh every 60s
- [ ] Schema unchanged → cache used (fast)
- [ ] Schema changed → detects and refetches
- [ ] Manual refresh button → forces immediate refresh

### Multi-DB Optimization
- [ ] Add 3 connections → all auto-connect
- [ ] Multi-DB mode activated → schemas loaded
- [ ] Type @ → shows all connections immediately
- [ ] Type @connection. → shows tables immediately
- [ ] Type alias. → shows columns immediately

## Configuration

Add to `frontend/src/lib/config.ts`:

```typescript
export const connectionConfig = {
  // Auto-connect settings
  autoConnect: true,                // Auto-connect on add
  autoConnectDelay: 100,            // ms delay before auto-connect
  
  // Schema refresh settings
  schemaRefreshEnabled: true,       // Background refresh
  schemaRefreshInterval: 60000,     // 60s
  
  // Health check settings
  healthCheckEnabled: false,        // Disabled by default
  healthCheckInterval: 120000,      // 2m
  
  // Connection settings
  connectionTimeout: 30000,         // 30s
  maxRetries: 3,
  retryDelay: 5000,                 // 5s
}
```

## Success Metrics

- **Default DB**: Users can query without @ syntax 90% of the time
- **Auto-Connect**: 0 manual clicks required for multi-DB queries
- **Schema Loading**: 10x faster for subsequent connections (cache)
- **Background Refresh**: Schemas fresh within 60s of changes
- **Error Rate**: <1% connection failures

---

**Ready to implement!** 🚀
