# Saved Queries Components

Comprehensive components for managing saved SQL queries with search, filtering, and organization features.

## Components

### SavedQueriesPanel

A slide-in sidebar panel for browsing and managing saved queries.

**Location:** `/Users/jacob_1/projects/sql-studio/frontend/src/components/saved-queries/SavedQueriesPanel.tsx`

#### Features

- **Sheet drawer** - Slides in from right side with smooth animations
- **Search** - Debounced search input (300ms delay) for finding queries
- **Filters:**
  - Folder dropdown with "All Folders" option
  - Tags multi-select with badge display
  - Favorites-only toggle
- **Sort controls:**
  - Sort by: title, created date, or updated date
  - Direction toggle (ascending/descending)
- **Stats display:**
  - Shows total query count
  - For Local tier: "Y/20 remaining" with progress indicator
  - Visual warnings when approaching limits
- **Scrollable query list** - Uses ScrollArea for smooth scrolling
- **Empty states:**
  - No queries saved
  - No search results
  - No favorites (when favorites filter is active)
- **Integration:**
  - SaveQueryDialog for editing queries
  - useSavedQueriesStore for state management
  - useTierStore for limit checking

#### Props

```typescript
interface SavedQueriesPanelProps {
  open: boolean                                    // Whether panel is open
  onClose: () => void                              // Callback when panel closes
  userId: string                                   // User ID for filtering queries
  onLoadQuery: (query: SavedQueryRecord) => void  // Callback when user loads a query
}
```

#### Usage

```tsx
import { SavedQueriesPanel } from '@/components/saved-queries'

function MyComponent() {
  const [isPanelOpen, setIsPanelOpen] = useState(false)
  const { userId } = useAuth()

  const handleLoadQuery = (query) => {
    editor.setValue(query.query_text)
    setIsPanelOpen(false)
  }

  return (
    <>
      <Button onClick={() => setIsPanelOpen(true)}>
        Saved Queries
      </Button>

      <SavedQueriesPanel
        open={isPanelOpen}
        onClose={() => setIsPanelOpen(false)}
        userId={userId}
        onLoadQuery={handleLoadQuery}
      />
    </>
  )
}
```

### QueryCard

Displays an individual saved query with metadata and action menu.

**Location:** `/Users/jacob_1/projects/sql-studio/frontend/src/components/saved-queries/QueryCard.tsx`

#### Features

- Query title, description, and folder/tags display
- Favorite star indicator (clickable)
- Click card to load query
- Dropdown menu with actions:
  - Load Query
  - Edit
  - Duplicate
  - Toggle Favorite
  - Delete (with confirmation dialog)
- Sync status badge (for Individual tier)
- Last updated timestamp
- Hover states and transitions

#### Props

```typescript
interface QueryCardProps {
  query: SavedQueryRecord             // The saved query to display
  onLoad: (query: SavedQueryRecord) => void
  onEdit: (query: SavedQueryRecord) => void
  onDelete: (id: string) => void
  onDuplicate: (id: string) => void
  onToggleFavorite: (id: string) => void
  showSyncStatus?: boolean           // Show sync indicator (default: false)
}
```

### SaveQueryDialog

Modal dialog for creating and editing saved queries.

**Location:** `/Users/jacob_1/projects/sql-studio/frontend/src/components/saved-queries/SaveQueryDialog.tsx`

#### Features

- Create new or edit existing queries
- Form fields:
  - Title (required)
  - Description (optional)
  - Query text (read-only display)
  - Folder selector with "Create New" option
  - Tags with add/remove and suggestions
  - Favorite toggle
- Validation and error handling
- Optimistic updates

#### Props

```typescript
interface SaveQueryDialogProps {
  open: boolean
  onClose: () => void
  userId: string
  initialQuery?: string              // For new queries
  existingQuery?: SavedQueryRecord   // For editing
  onSaved?: (query: SavedQueryRecord) => void
}
```

## State Management

All components use `useSavedQueriesStore` from Zustand for state management:

```typescript
const {
  queries,           // Array of SavedQueryRecord
  isLoading,         // Loading state
  error,             // Error message
  folders,           // Available folders
  tags,              // Available tags
  searchText,        // Current search text
  selectedFolder,    // Selected folder filter
  selectedTags,      // Selected tags filter
  showFavoritesOnly, // Favorites filter toggle
  sortBy,            // Sort field
  sortDirection,     // Sort direction
  // ... actions
} = useSavedQueriesStore()
```

## Tier Limits

The panel integrates with `useTierStore` to check and display query limits:

- **Local tier:** Max 20 saved queries
- **Individual tier:** Unlimited queries
- **Team tier:** Unlimited queries

Progress indicators and warnings appear when:
- Near limit (< 20% remaining): Orange progress bar
- At limit (100%): Red progress bar + blocking message

## Styling

The component uses:
- **Tailwind CSS** for utility classes
- **CSS custom properties** for theming
- **Radix UI primitives** for Sheet, ScrollArea, Select, etc.
- **Lucide React** for icons
- **shadcn/ui** component patterns

### Key Design Elements

- **Spacing:** Consistent 4px grid system
- **Borders:** Subtle borders with `border` color
- **Hover effects:** Smooth transitions on cards and buttons
- **Typography:** Clear hierarchy with proper text sizing
- **Colors:** Semantic colors for states (destructive, warning, success)

## Accessibility

All components are fully accessible:

- Keyboard navigation support
- Proper ARIA labels and roles
- Focus management
- Screen reader friendly
- Semantic HTML structure

## Testing

See `USAGE_EXAMPLE.tsx` for integration examples:

- Basic integration
- Full query editor integration
- Keyboard shortcuts
- Custom actions

## Performance

Optimizations:
- Debounced search (300ms)
- Optimistic updates for CRUD operations
- Virtualized scrolling via ScrollArea
- Efficient re-renders with Zustand selectors
- Memoized filter counts

## Browser Support

Tested and working in:
- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)

## Dependencies

Required packages:
- `@radix-ui/react-dialog` - Sheet/Dialog components
- `@radix-ui/react-scroll-area` - Scrollable area
- `@radix-ui/react-select` - Select dropdowns
- `@radix-ui/react-progress` - Progress bar
- `lucide-react` - Icons
- `zustand` - State management
- `class-variance-authority` - Variant styling
- `clsx` - Class name utility

## File Structure

```
src/components/saved-queries/
├── SavedQueriesPanel.tsx    # Main panel component
├── QueryCard.tsx             # Individual query card
├── SaveQueryDialog.tsx       # Save/edit dialog
├── index.ts                  # Public exports
├── USAGE_EXAMPLE.tsx         # Usage examples
└── README.md                 # This file
```

## Future Enhancements

Potential improvements:
- [ ] Bulk operations (delete multiple, move to folder)
- [ ] Export/import query collections
- [ ] Query templates with parameters
- [ ] Sharing queries with team members
- [ ] Query execution history per saved query
- [ ] AI-powered query suggestions
- [ ] Advanced search with filters (SQL keywords, complexity)
- [ ] Keyboard shortcuts for quick access
- [ ] Drag and drop to reorder favorites
- [ ] Query versioning/history

## Related Components

- `QueryEditor` - Main SQL editor component
- `ConnectionSelector` - Database connection picker
- `ResultsTable` - Query results display

## Support

For issues or questions, check:
- Component source code and inline documentation
- `USAGE_EXAMPLE.tsx` for integration patterns
- Store documentation: `/src/store/saved-queries-store.ts`
- Type definitions: `/src/types/storage.ts`
