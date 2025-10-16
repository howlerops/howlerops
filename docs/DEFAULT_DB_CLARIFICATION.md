# Default Database - Clarified Behavior

## Correct Behavior

The **default database** is used for **tab initialization** when creating new query tabs, NOT for query execution.

### Use Case

User has 3 connections:
- Production (⭐ default)
- Staging
- Analytics

### Flow

1. **Creating New Tab**
   ```
   User clicks "New Tab" button
   
   If connections.length > 1 AND defaultConnectionId exists:
     → New tab is created with connectionId = defaultConnectionId
   Else:
     → New tab is created with connectionId = activeConnection?.id
   ```

2. **User Can Override**
   ```
   User can change tab connection via dropdown
   User can use @ syntax in any tab
   ```

3. **Query Execution**
   ```
   Tab has connectionId = "production-id"
   
   Query: SELECT * FROM users
   → Uses tab's connectionId (production)
   
   Query: SELECT * FROM @staging.users
   → Uses explicit @staging override
   
   Query: SELECT u.*, o.* FROM users u JOIN @staging.orders o
   → Multi-DB query: production (from tab) + staging (explicit)
   ```

## Implementation Changes

### Connection Store

```typescript
interface ConnectionState {
  defaultConnectionId: string | null
  
  setDefaultConnection: (connectionId: string | null) => void
  getDefaultConnection: () => DatabaseConnection | null
}
```

### Query Store - Tab Creation

**BEFORE:**
```typescript
createTab: (title = 'New Query', connectionId?: string) => {
  const newTab: QueryTab = {
    id: crypto.randomUUID(),
    title,
    content: '',
    isDirty: false,
    isExecuting: false,
    connectionId,  // undefined or explicitly passed
  }
}
```

**AFTER:**
```typescript
createTab: (title = 'New Query', connectionId?: string) => {
  // Determine initial connection
  let initialConnectionId = connectionId
  
  if (!initialConnectionId) {
    const { connections, defaultConnectionId, activeConnection } = useConnectionStore.getState()
    
    if (connections.length > 1 && defaultConnectionId) {
      // Multi-connection mode with default set → use default
      initialConnectionId = defaultConnectionId
    } else if (activeConnection) {
      // Use active connection
      initialConnectionId = activeConnection.id
    }
  }
  
  const newTab: QueryTab = {
    id: crypto.randomUUID(),
    title,
    content: '',
    isDirty: false,
    isExecuting: false,
    connectionId: initialConnectionId,
  }
}
```

### Query Execution - No Changes Needed!

Query execution already uses the tab's `connectionId`, so no changes needed there.

## UI Indicators

### Sidebar
```
┌──────────────────────────────┐
│ Connections                  │
├──────────────────────────────┤
│ ⭐ Production    [Connected] │ ← Click star = default for new tabs
│    Staging       [Connected] │
│    Analytics     [Connected] │
└──────────────────────────────┘
```

### Query Tab Bar
```
┌─────────────────────────────────────────────┐
│ [Query 1] [Production ▾] [×]                │ ← Tab uses Production
│ [Query 2] [Staging ▾]    [×]                │ ← Tab uses Staging
│ [+ New Tab] ← Would create with Production  │
└─────────────────────────────────────────────┘
```

### Multi-DB Indicator
```
┌─────────────────────────────────────────────┐
│ [🌐 Multi-DB Mode] Default: Production      │
│ (New tabs will use Production by default)   │
└─────────────────────────────────────────────┘
```

## Benefits

1. **Fast Workflow**: New tabs automatically use your primary database
2. **Flexible Override**: Can change per-tab or use @ syntax
3. **No Confusion**: Clear which connection each tab uses
4. **Multi-DB Queries**: Can still reference other DBs with @

## Example Scenarios

### Scenario 1: Primary Production, Occasional Staging Checks
```
Default: Production

Tab 1: SELECT * FROM users               → Production
Tab 2: SELECT * FROM @staging.users      → Staging (explicit)
Tab 3: (Change dropdown to Staging)
       SELECT * FROM users               → Staging (tab override)
```

### Scenario 2: Development Workflow
```
Default: Development

Tab 1: Working query                     → Development
Tab 2: SELECT * FROM @prod.users         → Check production
Tab 3: Compare data
       SELECT d.*, p.* 
       FROM users d                      → Development (tab)
       JOIN @prod.users p                → Production (explicit)
```

### Scenario 3: No Default Set
```
Default: None

Tab 1: Must select connection manually
Tab 2: Must select connection manually
```

---

This is the **correct** and **intuitive** behavior for default databases.
