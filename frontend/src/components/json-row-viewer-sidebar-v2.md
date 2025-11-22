# JsonRowViewerSidebarV2 - Simplified JSON Row Viewer

A radically simplified row viewer following the "ruthless simplicity" philosophy.

## Key Improvements

### Removed Complexity
- ❌ Custom JSON tokenizer/formatter (562 lines → 380 lines)
- ❌ View/Edit mode toggle (always editable)
- ❌ Complex search with regex/toggles (use browser Ctrl+F instead)
- ❌ Word wrap toggle
- ❌ FK auto-detection with pluralization logic
- ❌ Validation map system
- ❌ Multiple copy buttons

### Added Features
- ✅ Row navigation (prev/next arrows in header)
- ✅ Keyboard shortcuts (↑↓ for navigation, Esc to close)
- ✅ Toast notifications for user feedback
- ✅ Clean, simple inline editing

## Usage Example

```tsx
import { JsonRowViewerSidebarV2 } from '@/components/json-row-viewer-sidebar-v2'

function MyTable() {
  const [selectedRowIndex, setSelectedRowIndex] = useState<number>(-1)
  const [isViewerOpen, setIsViewerOpen] = useState(false)

  const data = [
    { id: 1, name: 'Alice', email: 'alice@example.com' },
    { id: 2, name: 'Bob', email: 'bob@example.com' },
    // ...
  ]

  const handleRowClick = (index: number) => {
    setSelectedRowIndex(index)
    setIsViewerOpen(true)
  }

  const handleNavigate = (direction: 'prev' | 'next') => {
    setSelectedRowIndex(prev => {
      if (direction === 'prev') return Math.max(0, prev - 1)
      return Math.min(data.length - 1, prev + 1)
    })
  }

  const handleSave = async (rowId: string, newData: Record<string, CellValue>) => {
    // Save logic here
    console.log('Saving row:', rowId, newData)
    return true // Return true on success
  }

  const selectedRow = selectedRowIndex >= 0 ? data[selectedRowIndex] : null
  const rowId = selectedRow?.id?.toString() ?? null

  return (
    <>
      {/* Your table here */}

      <JsonRowViewerSidebarV2
        open={isViewerOpen}
        onClose={() => setIsViewerOpen(false)}
        rowData={selectedRow}
        rowId={rowId}
        rowIndex={selectedRowIndex}
        totalRows={data.length}
        onNavigate={handleNavigate}
        metadata={metadata} // Optional: for FK display and PK detection
        onSave={handleSave}
      />
    </>
  )
}
```

## Props

### Required Props

| Prop | Type | Description |
|------|------|-------------|
| `open` | `boolean` | Whether the sidebar is open |
| `onClose` | `() => void` | Callback when sidebar should close |
| `rowData` | `TableRow \| null` | The row data to display |
| `rowId` | `string \| null` | The row identifier |
| `rowIndex` | `number` | Current row index (0-based) |
| `totalRows` | `number` | Total number of rows |
| `onNavigate` | `(direction: 'prev' \| 'next') => void` | Navigation callback |

### Optional Props

| Prop | Type | Description |
|------|------|-------------|
| `columns` | `string[]` | Column names (currently unused) |
| `metadata` | `QueryEditableMetadata \| null` | Metadata for PK/FK detection |
| `connectionId` | `string` | Database connection ID |
| `onSave` | `(rowId: string, data: Record<string, CellValue>) => Promise<boolean>` | Save callback |

## Features

### Row Navigation
- Use prev/next arrows in header to navigate between rows
- Keyboard shortcuts: ↑ for previous, ↓ for next
- Navigation disabled at boundaries (first/last row)

### Field Editing
- Click any field value to edit it inline
- Primary keys are read-only (shown with PK badge)
- Press Enter to save, Escape to cancel
- Long text (>100 chars) uses textarea, short text uses input
- Type coercion: numbers, booleans, and null are handled automatically

### Foreign Keys
- Foreign key relationships shown at top (from metadata only)
- No auto-detection - relies on query metadata
- Displays as badges: `fieldName: value → table.column`

### Keyboard Shortcuts
- `↑` - Previous row (when not editing)
- `↓` - Next row (when not editing)
- `Esc` - Close sidebar
- `Enter` - Save field edit
- `Esc` (in field) - Cancel field edit

### Toast Notifications
- Save success: "Changes saved"
- Save failure: "Save failed"
- Copy success: "Copied to clipboard"
- Copy failure: "Copy failed"

## Design Philosophy

This component embodies "ruthless simplicity":

1. **Simple state** - Just `editedData` and `isSaving`, no complex hooks
2. **Direct editing** - No mode toggle, click to edit
3. **Browser features** - Use Ctrl+F for search instead of building it
4. **Clear feedback** - Toast notifications instead of inline errors
5. **Keyboard first** - Navigation and editing optimized for keyboard
6. **Minimal abstractions** - One component file, sub-components inline

## Migration from v1

To migrate from the old `JsonRowViewerSidebar`:

1. Add row navigation props:
   ```tsx
   // Add these state variables
   const [selectedRowIndex, setSelectedRowIndex] = useState(0)

   // Add navigation handler
   const handleNavigate = (direction: 'prev' | 'next') => {
     // Update selectedRowIndex
   }
   ```

2. Update the component usage:
   ```tsx
   <JsonRowViewerSidebarV2
     // ... existing props
     rowIndex={selectedRowIndex}
     totalRows={data.length}
     onNavigate={handleNavigate}
   />
   ```

3. Remove custom hook usage:
   ```tsx
   // DELETE: const { ... } = useJsonViewer(...)
   // Component handles state internally
   ```

## File Size Comparison

- **Original**: 562 lines (with complex hook + tokenizer)
- **Simplified**: 380 lines (complete, no external dependencies)
- **Reduction**: 32% smaller, much simpler

## Performance

- No complex JSON formatting - just `Object.entries()`
- No search indexing overhead
- Simple re-renders on data changes
- Minimal state management

## Limitations (By Design)

1. **No search** - Use browser's Ctrl+F to search within the sidebar
2. **No word wrap toggle** - Text wraps naturally based on container
3. **No FK auto-detection** - Only shows FKs from query metadata
4. **No validation UI** - Relies on save callback to handle validation
5. **Basic type coercion** - Simple string→number/boolean parsing

These limitations are intentional trade-offs for simplicity and maintainability.
