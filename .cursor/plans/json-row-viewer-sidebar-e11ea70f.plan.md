<!-- e11ea70f-252b-4cb4-9cbc-37fe06df191c 695ffbd0-c2c1-481a-a78b-9d5d4bb04270 -->
# JSON Row Viewer Sidebar Implementation

## Overview

Create a slide-out sidebar component that displays table row data as formatted JSON with advanced features including search, foreign key expansion, editing with validation, and real-time updates.

## Architecture

### Core Components

**1. `json-row-viewer-sidebar.tsx`** (New)

- Main sidebar component using Sheet from `@/components/ui/sheet`
- Props: `open`, `onClose`, `rowData`, `columns`, `metadata`, `connectionId`, `onSave`
- Features:
  - Formatted JSON display with syntax highlighting
  - Expand/collapse all controls
  - Copy visible JSON button
  - Word wrap toggle
  - Search by key name with regex support
  - Foreign key inline expansion
  - Edit mode with validation

**2. `json-editor.tsx`** (New)

- Reusable JSON editor component with validation
- Features:
  - Syntax highlighting for JSON
  - Line numbers
  - Collapsible sections for nested objects/arrays
  - Inline error indicators
  - Real-time validation
  - Undo/redo support

**3. `foreign-key-resolver.tsx`** (New)

- Component to fetch and display related data
- Uses existing schema introspection to identify foreign keys
- Displays related rows inline as nested JSON
- Lazy loading with caching

### Integration Points

**4. Update `editable-table.tsx`**

- Add row click handler to open JSON viewer
- Pass row data, columns, and metadata to sidebar
- Add visual indicator (eye icon) on row hover
- Wire up save callback to update table data

**5. Update `query-results-table.tsx`**

- Similar integration as editable-table
- Use existing metadata for foreign key detection
- Wire up to existing save handlers

### State Management

**6. Create `json-viewer-store.ts`** (New)

- Zustand store for JSON viewer state
- Track: open state, current row, edit mode, expanded keys, search query
- Actions: openRow, closeViewer, toggleEdit, updateRow, setSearch

### Utilities

**7. `json-formatter.ts`** (New)

- Format JSON with proper indentation
- Syntax highlighting tokens
- Handle circular references
- Detect and format special types (dates, buffers, etc.)

**8. `json-search.ts`** (New)

- Search by key name
- Regex pattern matching
- Highlight matches in JSON
- Navigate between matches

## Implementation Details

### JSON Display Features

```typescript
// Collapsible JSON tree structure
- Root level always expanded
- Objects/arrays collapsible by default
- Remember expansion state per row
- Keyboard shortcuts: Ctrl+E (expand all), Ctrl+C (collapse all)
```

### Foreign Key Resolution

```typescript
// Use existing schema metadata
- Detect foreign keys from QueryEditableMetadata
- Fetch related rows on-demand
- Cache results per session
- Display as expandable nested objects
- Format: { _related: { tableName: [rows] } }
```

### Edit Mode

```typescript
// Validation rules
- Respect column data types from metadata
- Required field validation
- Type coercion (string → number, etc.)
- Show inline errors next to invalid fields
- Disable save button until valid
```

### Search Implementation

```typescript
// Search features
- Filter by key name (case-insensitive)
- Regex support with /pattern/ syntax
- Highlight matches in yellow
- Show match count (e.g., "3 matches")
- Next/previous navigation buttons
```

## Files to Create

1. `/frontend/src/components/json-row-viewer-sidebar.tsx` - Main sidebar component
2. `/frontend/src/components/json-editor.tsx` - JSON editor with validation
3. `/frontend/src/components/foreign-key-resolver.tsx` - Foreign key expansion
4. `/frontend/src/store/json-viewer-store.ts` - State management
5. `/frontend/src/lib/json-formatter.ts` - JSON formatting utilities
6. `/frontend/src/lib/json-search.ts` - Search functionality
7. `/frontend/src/hooks/use-json-viewer.ts` - Custom hook for sidebar logic

## Files to Modify

1. `/frontend/src/components/editable-table/editable-table.tsx`

   - Add row click handler at line ~407
   - Import and render JsonRowViewerSidebar
   - Pass row data and callbacks

2. `/frontend/src/components/query-results-table.tsx`

   - Add row click handler in EditableTable props
   - Import and render JsonRowViewerSidebar
   - Wire up to existing save handlers

3. `/frontend/src/types/table.ts`

   - Add `onRowClick?: (rowId: string, rowData: TableRow) => void` to EditableTableProps
   - Add JsonViewerState interface

## Key Technical Decisions

- Use `react-json-view` or build custom JSON tree component (recommend custom for control)
- Leverage existing `QueryEditableMetadata` for foreign key detection
- Reuse validation logic from `table-cell.tsx`
- Use `Sheet` component with `sm:max-w-2xl` for wider sidebar
- Implement virtual scrolling for large JSON objects
- Cache foreign key lookups in memory (Map<tableId, Map<pkValue, row>>)

## Performance Optimizations

- Lazy render collapsed JSON sections
- Debounce search input (300ms)
- Memoize formatted JSON output
- Virtual scrolling for arrays with 100+ items
- Limit foreign key expansion depth to 2 levels

## Testing Considerations

- Test with wide rows (50+ columns)
- Test with nested JSON columns
- Test with circular foreign key references
- Test search with special regex characters
- Test edit mode validation edge cases

### To-dos

- [ ] Create json-formatter.ts utility with syntax highlighting and special type handling
- [ ] Create json-search.ts utility for key name search and regex matching
- [ ] Create json-viewer-store.ts Zustand store for sidebar state management
- [ ] Create json-editor.tsx component with collapsible tree, validation, and editing
- [ ] Create foreign-key-resolver.tsx for inline foreign key expansion
- [ ] Create use-json-viewer.ts custom hook for sidebar logic
- [ ] Create json-row-viewer-sidebar.tsx main sidebar component with all features
- [ ] Update editable-table.tsx to add row click handler and render JsonRowViewerSidebar
- [ ] Update query-results-table.tsx to integrate JsonRowViewerSidebar with existing save handlers
- [ ] Update types/table.ts to add onRowClick callback and JsonViewerState interface