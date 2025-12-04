# AG Grid Table - Quick Start Guide

## 5-Minute Setup

### 1. Import the Component

```typescript
import { AGGridTable } from '@/components/ag-grid-table';
import { TableColumn, TableRow } from '@/types/table';
```

### 2. Define Your Columns

```typescript
const columns: TableColumn[] = [
  {
    id: 'id',
    header: 'ID',
    type: 'number',
    width: 80,
    editable: false,
  },
  {
    id: 'name',
    header: 'Name',
    type: 'text',
    editable: true,
  },
  {
    id: 'active',
    header: 'Active',
    type: 'boolean',
    editable: true,
  },
];
```

### 3. Prepare Your Data

```typescript
const data: TableRow[] = [
  { __rowId: '1', id: 1, name: 'John Doe', active: true },
  { __rowId: '2', id: 2, name: 'Jane Smith', active: false },
];
```

**Important**: Each row MUST have a `__rowId` field!

### 4. Render the Table

```typescript
function MyTable() {
  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={600}
      onCellEdit={async (rowId, columnId, value) => {
        // Save to backend
        await api.updateCell(rowId, columnId, value);
        return true; // Return true on success
      }}
    />
  );
}
```

## Common Use Cases

### Handling Row Selection

```typescript
<AGGridTable
  data={data}
  columns={columns}
  enableMultiSelect={true}
  onRowSelect={(selectedRowIds) => {
    console.log('Selected:', selectedRowIds);
  }}
/>
```

### Adding a Toolbar

```typescript
<AGGridTable
  data={data}
  columns={columns}
  toolbar={(context) => (
    <div className="flex gap-2 p-2">
      <button onClick={() => context.actions.selectAllRows(true)}>
        Select All
      </button>
      <span>Selected: {context.state.selectedRows.length}</span>
    </div>
  )}
/>
```

### Custom Cell Rendering

```typescript
<AGGridTable
  data={data}
  columns={columns}
  customCellRenderers={{
    status: (value) => (
      <span className={`badge ${value === 'active' ? 'success' : 'warning'}`}>
        {value}
      </span>
    ),
  }}
/>
```

### Tracking Dirty Rows

```typescript
<AGGridTable
  data={data}
  columns={columns}
  onDirtyChange={(dirtyRowIds) => {
    console.log('Unsaved changes in rows:', dirtyRowIds);
  }}
  onCellEdit={async (rowId, columnId, value) => {
    const success = await saveToBackend(rowId, columnId, value);
    return success; // Row removed from dirty state if true
  }}
/>
```

## Column Types Reference

### Text Column
```typescript
{
  type: 'text',
  monospace: true,      // Use monospace font
  wrapContent: true,    // Allow text wrapping
  clipContent: true,    // Truncate long text
}
```

### Number Column
```typescript
{
  type: 'number',
  validation: {
    min: 0,
    max: 100,
    message: 'Must be 0-100',
  },
}
```

### Boolean Column
```typescript
{
  type: 'boolean',
  // Renders as checkbox automatically
}
```

### Select Column
```typescript
{
  type: 'select',
  options: ['Option A', 'Option B', 'Option C'],
}
```

### Date/DateTime Column
```typescript
{
  type: 'date',      // or 'datetime'
  // Uses AG Grid's built-in date picker
}
```

## Styling Options

### Sticky Columns
```typescript
{
  id: 'id',
  header: 'ID',
  sticky: 'left',  // or 'right'
}
```

### Primary Key
```typescript
{
  id: 'id',
  header: 'ID',
  isPrimaryKey: true,  // Adds special styling
}
```

### Column Width
```typescript
{
  width: 150,          // Preferred width
  minWidth: 100,       // Minimum width
  maxWidth: 300,       // Maximum width
}
```

## Pro Tips

### 1. Always Set __rowId
```typescript
// ✅ Good
const data = [{ __rowId: '1', ...fields }];

// ❌ Bad
const data = [{ id: 1, ...fields }]; // Will generate random IDs
```

### 2. Return Boolean from onCellEdit
```typescript
// ✅ Good - clears dirty state on success
onCellEdit={async (rowId, columnId, value) => {
  try {
    await api.save(rowId, columnId, value);
    return true;
  } catch (error) {
    console.error(error);
    return false;
  }
}}

// ❌ Bad - doesn't return anything
onCellEdit={async (rowId, columnId, value) => {
  await api.save(rowId, columnId, value);
  // Missing return!
}}
```

### 3. Use Height Prop
```typescript
// ✅ Good - visible table
<AGGridTable height={600} />

// ❌ Bad - might be collapsed
<AGGridTable /> // Defaults to 600px, but be explicit!
```

### 4. Enable Features Explicitly
```typescript
<AGGridTable
  enableMultiSelect={true}
  enableColumnResizing={true}
  enableGlobalFilter={true}
/>
```

## Next Steps

- 📖 Read the full [README.md](./README.md) for detailed documentation
- 💡 Check [examples](./ag-grid-table.example.tsx) for advanced usage
- 📝 Review [IMPLEMENTATION.md](./IMPLEMENTATION.md) for technical details
- 🧪 Run tests: `npm run test src/components/ag-grid-table/`

## Need Help?

Common issues:
- **No data showing**: Check that rows have `__rowId`
- **Edits not saving**: Check that `onCellEdit` returns `true`
- **Styles broken**: Import AG Grid CSS files
- **Performance issues**: Use `enableMultiSelect={false}` for huge datasets

---

**Ready to migrate from EditableTable?** Just swap the import - it's a drop-in replacement! 🚀
