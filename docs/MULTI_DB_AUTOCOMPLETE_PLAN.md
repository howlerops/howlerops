# Multi-DB Autocomplete Implementation Plan

**Goal**: Enhance QueryEditor's Monaco autocomplete to support `@connection.schema.table` syntax

---

## Current State

QueryEditor (line 338) has basic SQL autocomplete:
- Keywords (SELECT, FROM, WHERE, etc.)
- Tables from active connection
- Columns from referenced tables

**Missing**: Support for `@` syntax in multi-DB mode

---

## Required Features

### 1. Connection Name Autocomplete

When user types `@`:
```sql
SELECT * FROM @|
                ↑ cursor
```

**Should show**:
```
@production  (PostgreSQL - howlerops_prod)
@staging     (PostgreSQL - howlerops_staging)
@analytics   (MySQL - analytics_db)
```

### 2. Schema/Table Autocomplete After Connection

When user types `@connection.`:
```sql
SELECT * FROM @production.|
                          ↑ cursor
```

**Should show**:
```
@production.public.users
@production.public.orders
@production.auth.sessions
```

### 3. Column Autocomplete

When user types `@connection.schema.table.`:
```sql
SELECT @production.users.|
                        ↑ cursor
```

**Should show**:
```
id
name
email
created_at
```

### 4. Smart Context Awareness

After FROM with alias:
```sql
SELECT u.|
       ↑ cursor
FROM @production.users u
```

**Should show** columns from `production.users` table

---

## Implementation Plan

### Step 1: Fetch All Connection Schemas

**Add to QueryEditor component**:

```typescript
// Add state for multi-DB schemas
const [multiDBSchemas, setMultiDBSchemas] = useState<Map<string, SchemaNode[]>>(new Map());

// Load schemas for all connections when mode='multi'
useEffect(() => {
  if (mode === 'multi' && connections.length > 1) {
    loadMultiDBSchemas();
  }
}, [mode, connections]);

const loadMultiDBSchemas = async () => {
  try {
    const connectionIds = connections.map(c => c.id);
    const combinedSchema = await GetMultiConnectionSchema(connectionIds);
    
    // Convert to SchemaNode format
    const schemasMap = new Map<string, SchemaNode[]>();
    
    Object.entries(combinedSchema.connections || {}).forEach(([connId, connSchema]) => {
      const conn = connections.find(c => c.id === connId);
      const schemaNodes = convertToSchemaNodes(connSchema, conn);
      schemasMap.set(connId, schemaNodes);
    });
    
    setMultiDBSchemas(schemasMap);
  } catch (error) {
    console.error('Failed to load multi-DB schemas:', error);
  }
};
```

### Step 2: Enhance Completion Provider

**Update the Monaco completion provider** (line 338):

```typescript
completionProviderRef.current = monaco.languages.registerCompletionItemProvider('sql', {
  triggerCharacters: ['.', ' ', '\n', '@'],  // Add '@'
  provideCompletionItems: (model, position, context) => {
    const lineContent = model.getLineContent(position.lineNumber);
    const textUntilPosition = lineContent.substring(0, position.column - 1);
    
    // CASE 1: User typed '@' - show connections
    if (textUntilPosition.endsWith('@')) {
      return {
        suggestions: connections.map((conn, idx) => ({
          label: `@${conn.id}`,
          kind: monaco.languages.CompletionItemKind.Module,
          detail: `${conn.type} - ${conn.database}`,
          insertText: conn.id,
          documentation: conn.name || conn.id,
          sortText: `0${idx}`,  // Sort by index
          range: {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: position.column,
            endColumn: position.column,
          }
        }))
      };
    }
    
    // CASE 2: User typed '@connection.' - show schemas/tables
    const multiDBMatch = textUntilPosition.match(/@(\w+)\.$/);
    if (multiDBMatch && mode === 'multi') {
      const connectionId = multiDBMatch[1];
      const schemas = multiDBSchemas.get(connectionId) || [];
      
      return {
        suggestions: buildMultiDBTableSuggestions(schemas, connectionId, monaco, position)
      };
    }
    
    // CASE 3: Regular SQL autocomplete (existing code)
    // ... rest of existing completion logic
  }
});
```

### Step 3: Helper Functions

**Add helper to build multi-DB suggestions**:

```typescript
const buildMultiDBTableSuggestions = (
  schemas: SchemaNode[], 
  connectionId: string,
  monaco: any,
  position: any
) => {
  const suggestions = [];
  
  schemas.forEach(schemaNode => {
    if (schemaNode.type !== 'schema' || !schemaNode.children) return;
    
    schemaNode.children.forEach(tableNode => {
      if (tableNode.type !== 'table') return;
      
      suggestions.push({
        label: `${schemaNode.name}.${tableNode.name}`,
        kind: monaco.languages.CompletionItemKind.Class,
        detail: `Table from @${connectionId}.${schemaNode.name}`,
        insertText: `${schemaNode.name}.${tableNode.name}`,
        documentation: `${tableNode.name} (${tableNode.children?.length || 0} columns)`,
        sortText: `1${tableNode.name}`,
        range: {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: position.column,
          endColumn: position.column,
        }
      });
    });
  });
  
  return suggestions;
};
```

### Step 4: Alias Resolution for Multi-DB

**Enhance alias map building** to understand `@connection` references:

```typescript
const buildAliasMap = (query: string, schemaIndex: SchemaIndex) => {
  const aliasMap: Record<string, TableEntry> = {};
  
  // Existing regex for normal tables
  const normalPattern = /FROM\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi;
  
  // NEW: Pattern for multi-DB references
  const multiDBPattern = /FROM\s+@(\w+)\.(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi;
  
  let match;
  
  // Handle multi-DB references
  while ((match = multiDBPattern.exec(query)) !== null) {
    const [_, connectionId, schema, table, alias] = match;
    const aliasName = alias || table;
    
    // Look up table from multi-DB schemas
    const connectionSchemas = multiDBSchemas.get(connectionId);
    if (connectionSchemas) {
      const tableEntry = findTableInSchemas(connectionSchemas, schema || 'public', table);
      if (tableEntry) {
        aliasMap[aliasName.toLowerCase()] = tableEntry;
      }
    }
  }
  
  // ... rest of existing alias logic for normal tables
  
  return aliasMap;
};
```

---

## UI Enhancements

### Visual Indicators

**1. Connection Badges in Autocomplete**

```typescript
label: `@${conn.id}`,
detail: `🔗 ${conn.type} - ${conn.database}`,
```

**2. Schema Context**

```typescript
detail: `📁 @${connectionId}.${schemaName}`,
```

**3. Table Info**

```typescript
documentation: `${tableNode.name}\n${tableNode.children?.length || 0} columns\nConnection: @${connectionId}`,
```

---

## Performance Optimizations

### 1. Lazy Loading

Only fetch multi-DB schemas when:
- Mode switches to 'multi'
- User adds a new connection
- User types '@' for the first time

### 2. Caching

```typescript
const schemaCache = useRef<Map<string, {
  schemas: SchemaNode[];
  fetchedAt: number;
}>>(new Map());

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes
```

### 3. Debounced Fetching

```typescript
const debouncedLoadSchemas = useMemo(
  () => debounce(loadMultiDBSchemas, 500),
  [connections]
);
```

---

## Example User Flow

### Scenario 1: Basic Multi-DB Query

```sql
1. User types: SELECT * FROM @
   → Autocomplete shows: [production, staging, analytics]

2. User selects: production
   → Query: SELECT * FROM @production

3. User types: .
   → Autocomplete shows: [public.users, public.orders, auth.sessions]

4. User selects: public.users
   → Query: SELECT * FROM @production.public.users

5. User types:  u JOIN @
   → Autocomplete shows: [production, staging, analytics]

6. User selects: staging
   → Query: SELECT * FROM @production.public.users u JOIN @staging

7. User types: .public.orders o ON u.id = o.user_id
   → Autocomplete helps with: .public.orders
```

### Scenario 2: Column Autocomplete with Aliases

```sql
1. Query: SELECT u.| FROM @prod.users u
                  ↑
2. Autocomplete shows:
   - id
   - name
   - email
   - created_at
   (columns from @prod.users)
```

---

## Testing Checklist

- [ ] `@` triggers connection list
- [ ] Connection list shows all connections
- [ ] `@connection.` shows schemas/tables
- [ ] Table autocomplete works for multi-DB
- [ ] Column autocomplete works with aliases
- [ ] Performance: schemas fetch only when needed
- [ ] Cache invalidation works on connection changes
- [ ] Works in both light and dark themes
- [ ] Keyboard navigation works
- [ ] Documentation strings are helpful

---

## Files to Modify

1. **`frontend/src/components/query-editor.tsx`**
   - Add multi-DB schemas state
   - Enhance completion provider
   - Add helper functions

2. **`frontend/src/hooks/useMultiDBSchemas.ts`** (NEW)
   - Custom hook for schema fetching
   - Caching logic
   - Connection change detection

3. **`frontend/src/lib/monaco-multi-db.ts`**
   - Already has some helpers
   - Can be reused or enhanced

---

## Estimated Implementation Time

- **Schema fetching logic**: 30 minutes
- **Completion provider enhancement**: 1 hour  
- **Helper functions**: 30 minutes
- **Alias resolution**: 30 minutes
- **Testing**: 1 hour
- **Total**: ~3.5 hours

---

## Alternative: Use Existing monaco-multi-db.ts

The `monaco-multi-db.ts` file already has:
- `configureMultiDBLanguage()` function
- Connection/schema/table completion logic
- Custom tokenizer for @ syntax

**Option**: Integrate that logic into QueryEditor's completion provider rather than rebuilding.

---

## Next Steps

1. Review existing `monaco-multi-db.ts` capabilities
2. Decide: enhance QueryEditor directly OR use monaco-multi-db helpers
3. Implement schema fetching for all connections
4. Update completion provider with @ syntax support
5. Test with real multi-DB queries
6. Add visual polish (icons, better descriptions)

---

**Status**: READY TO IMPLEMENT  
**Priority**: HIGH (core multi-DB UX feature)  
**Complexity**: MEDIUM

