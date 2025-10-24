# SavedQueriesPanel - Quick Reference

## Import

```tsx
import { SavedQueriesPanel } from '@/components/saved-queries'
```

## Basic Usage

```tsx
<SavedQueriesPanel
  open={isPanelOpen}
  onClose={() => setIsPanelOpen(false)}
  userId="user-123"
  onLoadQuery={(query) => loadIntoEditor(query)}
/>
```

## Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `open` | `boolean` | Yes | Controls panel visibility |
| `onClose` | `() => void` | Yes | Called when panel should close |
| `userId` | `string` | Yes | User ID for query filtering |
| `onLoadQuery` | `(query: SavedQueryRecord) => void` | Yes | Called when user loads a query |

## Features at a Glance

### Search & Filter
- **Search:** Debounced 300ms, searches title/description/SQL/tags
- **Folder:** Dropdown with "All Folders" option
- **Tags:** Multi-select badges
- **Favorites:** Toggle button
- **Sort:** By title/created/updated, asc/desc toggle
- **Clear:** One-click clear all filters

### Stats Display
- Total query count
- Tier limits (Local: max 20)
- Progress bar with color coding:
  - Green: 0-80%
  - Orange: 80-99%
  - Red: 100%

### Empty States
1. No queries saved
2. No search results → "Clear filters" button
3. No favorites → "Show all queries" button

### CRUD Operations
- **Load:** Click card or use dropdown
- **Edit:** Opens SaveQueryDialog
- **Delete:** Shows confirmation dialog
- **Duplicate:** Creates copy with "(Copy)" suffix
- **Favorite:** Toggle star icon

## Store Integration

The panel automatically integrates with:

```tsx
// Saved queries store
const {
  queries,        // Current filtered list
  isLoading,      // Loading state
  error,          // Error message
  folders,        // Available folders
  tags,           // Available tags
  // ... filters and actions
} = useSavedQueriesStore()

// Tier store
const tierStore = useTierStore()
const limitCheck = tierStore.checkLimit('savedQueries', count)
```

## Common Patterns

### With Editor Integration

```tsx
function QueryEditor() {
  const [panelOpen, setPanelOpen] = useState(false)
  const [editorContent, setEditorContent] = useState('')

  return (
    <>
      <Button onClick={() => setPanelOpen(true)}>
        Browse Queries
      </Button>

      <CodeMirror value={editorContent} onChange={setEditorContent} />

      <SavedQueriesPanel
        open={panelOpen}
        onClose={() => setPanelOpen(false)}
        userId={currentUser.id}
        onLoadQuery={(query) => {
          setEditorContent(query.query_text)
          setPanelOpen(false)
        }}
      />
    </>
  )
}
```

### With Keyboard Shortcuts

```tsx
useEffect(() => {
  const handleKeyDown = (e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      setPanelOpen(true)
    }
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [])
```

### With Confirmation

```tsx
onLoadQuery={(query) => {
  if (hasUnsavedChanges) {
    if (confirm('Discard unsaved changes?')) {
      loadQuery(query)
    }
  } else {
    loadQuery(query)
  }
}}
```

## Styling Customization

### Custom Width

```tsx
<SavedQueriesPanel
  // Default is sm:max-w-2xl (672px)
  // Override via SheetContent className
/>
```

### Theme Colors

The panel uses CSS variables from your theme:
- `--primary` - Progress bar, active states
- `--destructive` - Delete actions, errors
- `--muted` - Backgrounds, disabled states
- `--border` - Borders and dividers

## Performance Tips

1. **Memoize onLoadQuery** if it has dependencies:
   ```tsx
   const handleLoad = useCallback((query) => {
     editor.setValue(query.query_text)
   }, [editor])
   ```

2. **Lazy load the panel** if not immediately needed:
   ```tsx
   {isPanelOpen && (
     <SavedQueriesPanel {...props} />
   )}
   ```

3. **Debounce filter changes** are already built-in (search: 300ms)

## Accessibility

- Keyboard: ESC closes, Tab navigates
- Screen readers: Semantic HTML + ARIA labels
- Focus management: Auto-focus search on open
- Color contrast: WCAG AA compliant

## Troubleshooting

### Panel doesn't open
- Check `open` prop is true
- Verify Sheet component renders (check dev tools)

### Queries not loading
- Verify `userId` is valid
- Check network tab for API calls
- Look for console errors

### Filters not working
- Store may not be initialized
- Check `loadQueries(userId)` is called
- Verify filter state in Redux DevTools

### Tier limits not showing
- Check `useTierStore` is providing tier data
- Verify tier is 'local' (Individual/Team are unlimited)

## Related Components

- **SaveQueryDialog** - Create/edit queries
- **QueryCard** - Individual query display
- **useSavedQueriesStore** - State management
- **useTierStore** - Tier limits

## File Locations

```
/frontend/src/components/saved-queries/
├── SavedQueriesPanel.tsx    ← Main component
├── QueryCard.tsx             ← Query card
├── SaveQueryDialog.tsx       ← Save/edit dialog
├── index.ts                  ← Exports
├── README.md                 ← Full docs
├── USAGE_EXAMPLE.tsx         ← Examples
└── QUICK_REFERENCE.md        ← This file
```

## Need Help?

1. See USAGE_EXAMPLE.tsx for complete examples
2. Read README.md for detailed documentation
3. Check component source code (well-commented)
4. Review store documentation in /src/store/saved-queries-store.ts
