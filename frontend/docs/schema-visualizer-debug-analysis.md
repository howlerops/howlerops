# Schema Visualizer ERD Mode - Debug Analysis

## Problem
Columns and foreign key relationships are not showing in the ERD mode of the schema visualizer.

## Data Flow Trace

### 1. Frontend Schema Loading (`schema-store.ts` lines 333-376)
**Status: ✅ Working - columns ARE being fetched**

```typescript
// Fetch columns for this table
const columnsResponse = await wailsEndpoints.schema.columns(
  sessionId,
  schemaInfo.name,
  tableInfo.name
)

if (columnsResponse.success && columnsResponse.data) {
  const normalizedForeignKeys = normalizeForeignKeys(
    columnsResponse.foreignKeys as RawForeignKeyInfo[] | undefined,
    schemaInfo.name,
    tableInfo.name
  )

  const foreignKeyByColumn = new Map<string, NormalizedForeignKeyInfo>()
  normalizedForeignKeys.forEach(fk => {
    if (!foreignKeyByColumn.has(fk.columnName)) {
      foreignKeyByColumn.set(fk.columnName, fk)
    }
  })

  tableNode.children = columnsResponse.data.map((columnInfo: RawColumnInfo, columnIndex: number) => {
    const normalizedColumn = normalizeColumnInfo(columnInfo, {
      foreignKey: foreignKeyByColumn.get(columnInfo.name)
    })

    return {
      id: `${tableId}.${columnInfo.name}.${columnIndex}`,
      name: formatColumnName(normalizedColumn),
      type: 'column' as const,
      metadata: normalizedColumn.metadata ?? normalizedColumn
    }
  })

  // Store foreign keys in table metadata
  if (normalizedForeignKeys.length > 0) {
    tableNode.metadata = {
      ...(tableNode.metadata || {}),
      foreignKeys: normalizedForeignKeys
    }
  }
}
```

**Key Points:**
- Columns are fetched via `wailsEndpoints.schema.columns()`
- Foreign keys come from `columnsResponse.foreignKeys`
- Foreign keys are attached to `tableNode.metadata.foreignKeys`
- Column data is stored in `tableNode.children[]`

### 2. Wails API Layer (`wails-api.ts` lines 73-130)
**Status: ✅ Working - passes through structure data**

```typescript
async getTableStructure(connectionId: string, schemaName: string, tableName: string) {
  const structure = await App.GetTableStructure(connectionId, schemaName, tableName)

  return {
    data: structure.columns?.map(column => ({
      name: column.name,
      dataType: column.data_type,
      nullable: column.nullable,
      defaultValue: column.default_value,
      primaryKey: column.primary_key,
      // ... other fields
    })) || [],
    foreignKeys: structure.foreign_keys || [],  // ✅ FK data is passed
    success: true
  }
}
```

### 3. Backend Go Layer (`app.go`)
**Status: ✅ Working - returns structure with FKs**

```go
func (a *App) GetTableStructure(connectionID, schema, table string) (*TableStructure, error) {
  structure, err := a.databaseService.GetTableStructure(connectionID, schema, table)
  if err != nil {
    return nil, err
  }

  fks := make([]ForeignKeyInfo, 0, len(structure.ForeignKeys))
  for _, fk := range structure.ForeignKeys {
    fks = append(fks, ForeignKeyInfo{
      Name:              fk.Name,
      Columns:           fk.Columns,
      ReferencedTable:   fk.ReferencedTable,
      ReferencedSchema:  fk.ReferencedSchema,
      ReferencedColumns: fk.ReferencedColumns,
      OnDelete:          fk.OnDelete,
      OnUpdate:          fk.OnUpdate,
    })
  }

  return &TableStructure{
    Columns:     columns,
    ForeignKeys: fks,  // ✅ FK data returned
    // ...
  }, nil
}
```

### 4. Schema Config Builder (`schema-config.ts`)
**Status: ⚠️ POTENTIAL ISSUE - Data transformation**

The `fromSchemaNodes()` function extracts columns and FK relationships:

```typescript
// Lines 29-73: Extract tables and columns
schemaNodes.forEach((schemaNode) => {
  if (schemaNode.children) {
    schemaNode.children.forEach((tableNode) => {
      if (tableNode.type === 'table' && tableNode.children) {
        const columns: ColumnConfig[] = tableNode.children
          .filter((col) => col.type === 'column')
          .map((col) => {
            // ... column mapping
          })

        const tableConfig: TableConfig = {
          id: tableNode.id,
          name: tableNode.name,
          schema: schemaNode.name,
          description: (tableNode.metadata as any)?.description,
          columns,  // ✅ Columns are included
        }

        tables.push(tableConfig)
      }
    })
  }
})

// Lines 75-76: Extract FK relationships
await this.extractForeignKeyRelationships(schemaNodes, tables, edges)
```

### 5. Foreign Key Extraction (`schema-config.ts` lines 241-358)
**Status: ⚠️ CRITICAL - This is where FK edges are created**

```typescript
private static async extractForeignKeyRelationships(
  schemaNodes: SchemaNode[],
  tables: TableConfig[],
  edges: EdgeConfig[]
): Promise<void> {
  // ... processes each schema and table

  // Check table metadata for foreign keys
  const tableMetadata = tableNode.metadata as { foreignKeys?: ForeignKeyInfo[] }
  if (tableMetadata?.foreignKeys) {  // ⚠️ DEPENDS ON METADATA
    for (const fk of tableMetadata.foreignKeys) {
      // Find source and target tables
      const sourceTable = tableMap.get(tableNode.id)
      const targetTable = tables.find(t =>
        t.name === fk.referencedTableName && t.schema === fk.referencedSchemaName
      )

      if (!sourceTable || !targetTable) continue

      // Create edge configuration
      const edgeConfig: EdgeConfig = {
        id: `${tableNode.id}_${fk.name}`,
        source: tableNode.id,
        sourceKey: fk.columnName,
        target: targetTable.id,
        targetKey: fk.referencedColumnName,
        relation: relationType,
        label: fk.name,
      }

      edges.push(edgeConfig)  // ✅ Edge is created IF foreignKeys exists
    }
  }
}
```

### 6. Schema Visualizer Loading (`schema-visualizer.tsx` lines 131-192)
**Status: ⚠️ POTENTIAL ISSUE - Data passed to visualizer**

```typescript
useEffect(() => {
  const initializeSchema = async () => {
    if (schema.length > 0) {
      try {
        const config = await SchemaConfigBuilder.fromSchemaNodes(schema)
        setSchemaConfig(config)

        console.log('Schema config created:', {
          tables: config.tables.length,
          edges: config.edges.length,
          edgeDetails: config.edges.map(e => ({
            id: e.id,
            source: e.source,
            target: e.target,
            relation: e.relation,
            label: e.label
          }))
        })

        const { nodes: flowNodes, edges: flowEdges } = SchemaConfigBuilder.toReactFlowNodes(config)
        setNodes(flowNodes as Node[])
        setEdges(flowEdges as Edge[])
      }
    }
  }

  initializeSchema()
}, [schema, setNodes, setEdges])
```

## Root Cause Analysis

### Issue 1: Columns Not Showing
**Location:** ERD table node rendering

The columns are loaded but may not be rendered in ERD mode. Check:
- `ERDTableNode` component - does it read `data.columns`?
- Compare with `TableNode` component

### Issue 2: Foreign Keys Not Showing
**Location:** Foreign key edge creation in `schema-config.ts`

The FK extraction relies on `tableNode.metadata.foreignKeys` being set correctly. Two potential failure points:

1. **Schema Store (lines 367-372):** FK metadata attachment
   ```typescript
   if (normalizedForeignKeys.length > 0) {
     tableNode.metadata = {
       ...(tableNode.metadata || {}),
       foreignKeys: normalizedForeignKeys
     }
   }
   ```
   ⚠️ If `normalizedForeignKeys.length === 0`, no metadata is attached

2. **Schema Config Builder (line 261):** FK metadata check
   ```typescript
   const tableMetadata = tableNode.metadata as { foreignKeys?: ForeignKeyInfo[] }
   if (tableMetadata?.foreignKeys) {  // ⚠️ Fails if undefined
   ```

## Recommended Fixes

### Fix 1: Add Debug Logging
Add console.log statements to trace data flow:

**In `schema-store.ts` (line 340):**
```typescript
const normalizedForeignKeys = normalizeForeignKeys(
  columnsResponse.foreignKeys as RawForeignKeyInfo[] | undefined,
  schemaInfo.name,
  tableInfo.name
)

console.log(`[SchemaStore] Foreign keys for ${tableInfo.name}:`, {
  rawFKs: columnsResponse.foreignKeys,
  normalized: normalizedForeignKeys
})
```

**In `schema-config.ts` (line 246):**
```typescript
for (const tableNode of schemaNode.children) {
  if (tableNode.type !== 'table' || !tableNode.children) continue

  const tableMetadata = tableNode.metadata as { foreignKeys?: ForeignKeyInfo[] }
  console.log(`[SchemaConfig] Processing table ${tableNode.name}:`, {
    hasMetadata: !!tableMetadata,
    hasForeignKeys: !!tableMetadata?.foreignKeys,
    fkCount: tableMetadata?.foreignKeys?.length || 0,
    metadata: tableNode.metadata
  })
}
```

### Fix 2: Check ERD Components
Verify that ERD mode components read column data:
- `/Users/jacob_1/projects/howlerops/frontend/src/components/schema-visualizer/erd-table-node.tsx`
- Compare with `/Users/jacob_1/projects/howlerops/frontend/src/components/schema-visualizer/table-node.tsx`

### Fix 3: Verify Data Structure
Check the actual data returned from backend:
- Add logging in `wails-api.ts` to see raw `structure.foreign_keys`
- Check if the backend database driver supports FK introspection
- Verify the database actually has foreign keys defined

## Testing Steps

1. Open browser console
2. Connect to a database with known foreign keys
3. Open schema visualizer in ERD mode
4. Look for console.log output showing:
   - `[SchemaStore] Foreign keys for ...`
   - `[SchemaConfig] Processing table ...`
   - `Schema config created: { tables: X, edges: Y }`
5. Check if `edges.length > 0`
6. Verify columns array exists in table node data

## Expected Behavior

### Columns
- Each table should have `data.columns` array
- Columns should be rendered in ERD table cards
- Primary keys should be highlighted

### Foreign Keys
- `config.edges` should contain EdgeConfig objects
- Each edge connects two tables via column handles
- Edge visualization should show relationship type
- Hovering should highlight related tables
