# Schema Visualizer Debug Findings & Fixes

## Investigation Summary

I've traced the complete data flow for columns and foreign key relationships in the schema visualizer ERD mode.

## Key Findings

### ✅ Components Are Working Correctly
Both `ERDTableNode` and `TableNode` components properly render columns:
- **ERDTableNode** (lines 64-72): Iterates over `columns` array and renders each column row
- **TableNode** (lines 66-94): Same functionality with different styling
- Both components create per-column handles for FK connections (lines 75-98 in ERDTableNode, 110-131 in TableNode)

### ✅ Data Flow is Complete
The data flows correctly through all layers:

1. **Backend (Go)**: `GetTableStructure()` returns columns and foreign keys
2. **Wails API**: Passes structure data including `foreign_keys` array
3. **Schema Store**: Fetches columns, normalizes foreign keys, attaches to metadata
4. **Schema Config Builder**: Extracts columns and builds FK edges
5. **Schema Visualizer**: Renders nodes and edges

### ⚠️ Potential Issues Identified

#### Issue 1: Silent FK Failures
If foreign keys are not returned from the backend database driver, the code fails silently without warnings. The extraction logic depends on:

```typescript
// In schema-store.ts
const normalizedForeignKeys = normalizeForeignKeys(
  columnsResponse.foreignKeys,  // ⚠️ May be undefined/empty
  schemaInfo.name,
  tableInfo.name
)
```

If `columnsResponse.foreignKeys` is `undefined` or `[]`, no edges are created, but there's no visibility into WHY.

#### Issue 2: Mock FKs Only as Fallback
The code adds mock FK relationships ONLY when `edges.length === 0`:

```typescript
// In schema-config.ts
if (edges.length === 0 && tables.length > 1) {
  this.addMockForeignKeyRelationships(tables, edges)
}
```

This means if even ONE real FK is found, no mock FKs are added, but other FKs might still be missing.

## Fixes Applied

### Debug Logging Added

I've added comprehensive console logging throughout the data flow to make debugging visible:

#### 1. Schema Store (`schema-store.ts` line 347)
```typescript
console.log(`[SchemaStore] 🔍 Table: ${schemaInfo.name}.${tableInfo.name}`, {
  columnsCount: columnsResponse.data.length,
  rawForeignKeys: columnsResponse.foreignKeys,
  normalizedFKs: normalizedForeignKeys,
  fkCount: normalizedForeignKeys.length
})
```

This will show:
- How many columns were fetched
- Raw FK data from backend
- Normalized FK data
- Count of FKs found

#### 2. Schema Config Builder - Entry Point (`schema-config.ts` line 18)
```typescript
console.log('[SchemaConfig] 🚀 fromSchemaNodes called with', schemaNodes.length, 'schemas')
```

#### 3. Schema Config Builder - Table Processing (`schema-config.ts` line 71)
```typescript
console.log(`[SchemaConfig] 📊 Table: ${schemaNode.name}.${tableNode.name}`, {
  id: tableConfig.id,
  columnsCount: columns.length,
  hasChildren: !!tableNode.children,
  childrenCount: tableNode.children?.length || 0,
  metadata: tableNode.metadata
})
```

This shows:
- Each table being processed
- Column count
- Whether children exist
- Full metadata including foreignKeys array

#### 4. Schema Config Builder - FK Extraction (`schema-config.ts` line 278)
```typescript
console.log(`[SchemaConfig] 🔍 Checking FK metadata for ${schemaNode.name}.${tableNode.name}`, {
  hasMetadata: !!tableMetadata,
  hasForeignKeys: !!tableMetadata?.foreignKeys,
  fkCount: tableMetadata?.foreignKeys?.length || 0,
  metadata: tableMetadata
})
```

This shows for each table:
- Whether metadata exists
- Whether foreignKeys exist in metadata
- FK count
- Full metadata object

#### 5. Schema Config Builder - Edge Creation (`schema-config.ts` line 319)
```typescript
console.log(`[SchemaConfig] ✅ Created FK edge:`, {
  from: `${sourceTable.name}.${fk.columnName}`,
  to: `${targetTable.name}.${fk.referencedColumnName}`,
  relation: relationType
})
```

This shows each FK edge that is successfully created.

#### 6. Schema Config Builder - FK Summary (`schema-config.ts` line 88)
```typescript
console.log('[SchemaConfig] 🔗 FK extraction complete', {
  edgesCreated: edges.length,
  tablesProcessed: tables.length
})
```

## Testing Instructions

To use the debug logging:

1. **Open Browser Console** (F12 or Cmd+Option+I)
2. **Connect to Database** with known foreign keys
3. **Open Schema Visualizer** in ERD mode
4. **Check Console Output** - you should see:

```
[SchemaStore] 🔍 Table: public.users
  columnsCount: 5
  rawForeignKeys: []  // ⚠️ If empty, backend isn't returning FKs
  normalizedFKs: []
  fkCount: 0

[SchemaConfig] 🚀 fromSchemaNodes called with 1 schemas

[SchemaConfig] 📊 Table: public.users
  id: "..."
  columnsCount: 5
  hasChildren: true
  childrenCount: 5
  metadata: { foreignKeys: [] }  // ⚠️ Should contain FKs if they exist

[SchemaConfig] 🔍 Checking FK metadata for public.users
  hasMetadata: true
  hasForeignKeys: false  // ⚠️ Problem here if FKs should exist
  fkCount: 0
  metadata: { foreignKeys: [] }

[SchemaConfig] 🔗 FK extraction complete
  edgesCreated: 0  // ⚠️ No edges = no FK relationships visualized
  tablesProcessed: 10

[SchemaConfig] ⚠️ No FK edges found, adding mock relationships
```

## Diagnosis Guide

Use the console output to diagnose where the issue occurs:

### Scenario 1: No FKs from Backend
```
rawForeignKeys: undefined  or  []
```
**Problem**: Backend database driver not returning FK information
**Solution**: Check backend Go implementation for the specific database type

### Scenario 2: FKs Lost in Normalization
```
rawForeignKeys: [{...}]
normalizedFKs: []
```
**Problem**: `normalizeForeignKeys()` function failing
**Solution**: Check field name mapping in schema-store.ts lines 186-216

### Scenario 3: FKs Not in Metadata
```
normalizedFKs: [{...}]
metadata: {}  or  { foreignKeys: undefined }
```
**Problem**: FKs not being attached to table metadata
**Solution**: Check schema-store.ts lines 367-372

### Scenario 4: Metadata Present but No Edges
```
metadata: { foreignKeys: [{...}] }
edgesCreated: 0
```
**Problem**: FK extraction logic not finding matching tables
**Solution**: Check table name/schema matching in schema-config.ts lines 286-295

### Scenario 5: Columns Not Showing
Check the console for:
```
columnsCount: 0
```
If columns are fetched but not showing, check:
- `tableNode.children` array in Schema Store
- `columns` array passed to node component
- ERD component rendering logic

## Expected Working Output

When everything works correctly, you should see:

```
[SchemaStore] 🔍 Table: public.posts
  columnsCount: 7
  rawForeignKeys: [{name: "posts_user_id_fkey", columnName: "user_id", ...}]
  normalizedFKs: [{name: "posts_user_id_fkey", columnName: "user_id", ...}]
  fkCount: 1

[SchemaConfig] 📊 Table: public.posts
  columnsCount: 7
  metadata: {
    foreignKeys: [{
      name: "posts_user_id_fkey",
      columnName: "user_id",
      referencedTableName: "users",
      referencedSchemaName: "public",
      referencedColumnName: "id"
    }]
  }

[SchemaConfig] 🔍 Checking FK metadata for public.posts
  hasMetadata: true
  hasForeignKeys: true
  fkCount: 1

[SchemaConfig] ✅ Created FK edge:
  from: "posts.user_id"
  to: "users.id"
  relation: "belongsTo"

[SchemaConfig] 🔗 FK extraction complete
  edgesCreated: 1
  tablesProcessed: 2
```

## Next Steps

1. **Test with Real Database**: Connect to a database with known FK relationships
2. **Review Console Output**: Use the guide above to identify where data is lost
3. **Check Backend**: If no FKs from backend, investigate database driver implementation
4. **Report Findings**: Share console output to help identify the root cause

## Files Modified

- `/Users/jacob_1/projects/howlerops/frontend/src/store/schema-store.ts` - Added FK logging at line 347
- `/Users/jacob_1/projects/howlerops/frontend/src/lib/schema-config.ts` - Added comprehensive logging at lines 18, 71, 88, 278, 319

All logging uses descriptive emojis for easy scanning:
- 🔍 = Inspection/checking
- 📊 = Table processing
- 🔗 = FK/edge operations
- ✅ = Success
- ⚠️ = Warning/fallback
- 🚀 = Start/initialization
