# AG Grid Table Component

A drop-in replacement for `EditableTable` using **AG Grid Community** for better virtualization, built-in column stability, and reduced maintenance.

## Features

### Core Functionality
- ✅ Virtual scrolling (AG Grid's built-in virtualization)
- ✅ Cell editing with dirty state tracking
- ✅ Row selection (single and multi-select)
- ✅ Sorting and filtering
- ✅ Column resizing
- ✅ Custom cell renderers
- ✅ Keyboard navigation
- ✅ Column pinning (sticky columns)
- ✅ Dark theme support

### Migration Benefits
1. **Better Virtualization**: No white chunks during fast scrolling
2. **Built-in Column Stability**: Columns maintain width and position
3. **Less Maintenance**: Battle-tested library with extensive community support
4. **Performance**: Optimized for large datasets (1M+ rows)

## Usage

### Basic Example

```typescript
import { AGGridTable } from '@/components/ag-grid-table';
import { TableColumn, TableRow } from '@/types/table';

const columns: TableColumn[] = [
  {
    id: 'id',
    header: 'ID',
    type: 'number',
    editable: false,
    isPrimaryKey: true,
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

const data: TableRow[] = [
  { __rowId: '1', id: 1, name: 'John Doe', active: true },
  { __rowId: '2', id: 2, name: 'Jane Smith', active: false },
];

function MyTable() {
  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={600}
      enableMultiSelect={true}
      enableColumnResizing={true}
      onCellEdit={async (rowId, columnId, value) => {
        // Save to backend
        console.log('Edit:', { rowId, columnId, value });
        return true; // Return true on success
      }}
      onRowSelect={(selectedRows) => {
        console.log('Selected:', selectedRows);
      }}
    />
  );
}
```

### Advanced Features

#### Custom Cell Renderers

```typescript
<AGGridTable
  data={data}
  columns={columns}
  customCellRenderers={{
    status: (value, row) => (
      <div className={`badge ${value === 'active' ? 'success' : 'warning'}`}>
        {value}
      </div>
    ),
  }}
/>
```

#### Column Configuration

```typescript
const columns: TableColumn[] = [
  {
    id: 'code',
    header: 'Code',
    type: 'text',
    monospace: true,        // Use monospace font
    clipContent: true,      // Truncate long text
    sticky: 'left',         // Pin to left side
  },
  {
    id: 'amount',
    header: 'Amount',
    type: 'number',
    validation: {
      min: 0,
      max: 1000000,
      message: 'Amount must be between 0 and 1,000,000',
    },
  },
  {
    id: 'category',
    header: 'Category',
    type: 'select',
    options: ['A', 'B', 'C'],
  },
];
```

#### Toolbar and Footer

```typescript
<AGGridTable
  data={data}
  columns={columns}
  toolbar={(context) => (
    <div className="flex gap-2 p-2">
      <button onClick={() => context.actions.selectAllRows(true)}>
        Select All
      </button>
      <button onClick={() => context.actions.clearDirtyRows()}>
        Clear Changes
      </button>
      <span>Selected: {context.state.selectedRows.length}</span>
    </div>
  )}
  footer={(context) => (
    <div className="p-2 text-sm">
      Total: {context.data.length} rows
      {context.state.hasDirtyRows && (
        <span className="text-warning ml-4">
          {context.state.dirtyRows.size} unsaved changes
        </span>
      )}
    </div>
  )}
/>
```

## Props Interface

All props from `EditableTableProps` are supported:

```typescript
interface EditableTableProps {
  // Data
  data: TableRow[];
  columns: TableColumn[];

  // Callbacks
  onDataChange?: (data: TableRow[]) => void;
  onCellEdit?: (rowId: string, columnId: string, value: CellValue) => Promise<boolean>;
  onRowSelect?: (selectedRows: string[]) => void;
  onRowClick?: (rowId: string, rowData: TableRow) => void;
  onRowInspect?: (rowId: string, rowData: TableRow) => void;
  onSort?: (sorting: SortingState) => void;
  onFilter?: (filters: ColumnFiltersState) => void;
  onDirtyChange?: (dirtyRowIds: string[]) => void;

  // Display
  height?: number | string;
  className?: string;
  loading?: boolean;
  error?: string | null;

  // Features
  enableMultiSelect?: boolean;
  enableColumnResizing?: boolean;
  enableColumnReordering?: boolean;
  enableGlobalFilter?: boolean;
  enableExport?: boolean;

  // Customization
  toolbar?: ReactNode | EditableTableRenderer;
  footer?: ReactNode | EditableTableRenderer;
  customCellRenderers?: Record<string, (value: CellValue, row: TableRow) => ReactNode>;

  // Phase 2 (chunked data)
  resultId?: string;
  totalRows?: number;
  isLargeResult?: boolean;
  chunkingEnabled?: boolean;
}
```

## Column Types

### Text Column
```typescript
{
  type: 'text',
  monospace?: boolean,    // Use monospace font
  wrapContent?: boolean,  // Allow text wrapping
  clipContent?: boolean,  // Truncate long text
  longText?: boolean,     // Optimize for long text
  validation?: {
    pattern?: RegExp,
    message?: string,
  }
}
```

### Number Column
```typescript
{
  type: 'number',
  validation?: {
    min?: number,
    max?: number,
    message?: string,
  }
}
```

### Boolean Column
```typescript
{
  type: 'boolean',
  // Renders as checkbox
}
```

### Select Column
```typescript
{
  type: 'select',
  options: string[],  // Required
}
```

### Date/DateTime Column
```typescript
{
  type: 'date' | 'datetime',
  // Uses AG Grid's date editor
}
```

## Dirty State Tracking

The component automatically tracks edited cells:

```typescript
<AGGridTable
  data={data}
  columns={columns}
  onDirtyChange={(dirtyRowIds) => {
    console.log('Dirty rows:', dirtyRowIds);
  }}
  onCellEdit={async (rowId, columnId, value) => {
    // Save to backend
    const success = await saveToBackend(rowId, columnId, value);

    // If success = true, row is automatically removed from dirty state
    // If success = false, row remains dirty
    return success;
  }}
/>
```

Visual indicators:
- Dirty cells: Orange triangle in top-left corner
- New rows: Green left border
- Primary key columns: Blue background

## Performance

AG Grid Community is optimized for large datasets:

- **Virtualization**: Only renders visible rows (handles 1M+ rows)
- **Column Virtualization**: Only renders visible columns
- **Debounced Scrolling**: Reduces scroll event processing
- **Efficient Updates**: Only re-renders changed cells

### Performance Tips

1. **Set fixed row height**: Improves scroll performance
   ```typescript
   rowHeight={31}  // Default value
   ```

2. **Use column types**: Enables AG Grid optimizations
   ```typescript
   type: 'number'  // vs generic 'text'
   ```

3. **Disable animations for large datasets**: Set in CSS
   ```css
   @media (prefers-reduced-motion: reduce) {
     --ag-transition-speed: 0ms;
   }
   ```

## Styling

The component uses CSS custom properties for theming:

```css
.ag-theme-alpine-dark {
  --ag-background-color: hsl(var(--background));
  --ag-foreground-color: hsl(var(--foreground));
  --ag-border-color: hsl(var(--border));
  /* ... more theme variables */
}
```

Custom cell classes:
- `.ag-cell-dirty`: Edited but not saved
- `.ag-row-new`: New row
- `.ag-cell-primary-key`: Primary key column

## Migration from EditableTable

The component is a **drop-in replacement**. Simply swap imports:

```typescript
// Before
import { EditableTable } from '@/components/editable-table';

// After
import { AGGridTable } from '@/components/ag-grid-table';

// Usage (no changes needed)
<AGGridTable {...props} />
```

### Feature Parity

| Feature | EditableTable | AGGridTable | Notes |
|---------|--------------|-------------|-------|
| Virtualization | TanStack Virtual | AG Grid | Better performance |
| Cell editing | Custom | AG Grid | Built-in editors |
| Row selection | Custom | AG Grid | Multi-range selection |
| Sorting | TanStack Table | AG Grid | Multi-column sort |
| Filtering | TanStack Table | AG Grid | Advanced filters |
| Column resizing | Custom | AG Grid | Smoother UX |
| Dirty tracking | Custom | Custom | Maintained |
| Keyboard nav | Custom | AG Grid | More features |

### Not Yet Implemented

Some advanced features from EditableTable are not yet implemented:

- [ ] Undo/Redo (planned)
- [ ] Copy/Paste (AG Grid has built-in, needs integration)
- [ ] Column reordering (partially supported)
- [ ] Export to CSV/JSON/XLSX (planned)

## Troubleshooting

### White screen or no data

Check that `data` prop has `__rowId` field:

```typescript
const data = [
  { __rowId: '1', ...otherFields },
  { __rowId: '2', ...otherFields },
];
```

### Edits not saving

Ensure `onCellEdit` returns `true` on success:

```typescript
onCellEdit={async (rowId, columnId, value) => {
  try {
    await api.update(rowId, columnId, value);
    return true;  // Important!
  } catch (error) {
    console.error(error);
    return false;
  }
}}
```

### Styles not applied

Import AG Grid CSS in your component or global CSS:

```typescript
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
```

## License

AG Grid Community is MIT licensed. No enterprise features are used.

## Resources

- [AG Grid Documentation](https://www.ag-grid.com/react-data-grid/)
- [AG Grid API Reference](https://www.ag-grid.com/react-data-grid/reference/)
- [AG Grid Examples](https://www.ag-grid.com/react-data-grid/examples/)
