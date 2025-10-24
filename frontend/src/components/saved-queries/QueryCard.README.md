# QueryCard Component

A feature-rich card component for displaying saved SQL queries in a list or grid layout.

## Features

- **Rich Metadata Display**: Title, description (truncated to 2 lines), folder, tags, and relative timestamps
- **Visual Indicators**: Favorite star icon (filled when favorited), sync status for Individual tier
- **Interactive Actions**: Load, Edit, Duplicate, Toggle Favorite, Delete (with confirmation)
- **Click to Load**: Click anywhere on the card to load the query into the editor
- **Responsive Design**: Works seamlessly in list or grid layouts
- **Full Accessibility**: WCAG 2.1 Level AA compliant with keyboard navigation and ARIA labels
- **Performance Optimized**: Efficient rendering for large lists

## Installation

The component is already part of the saved-queries module. Import it from:

```tsx
import { QueryCard } from '@/components/saved-queries'
// or
import { QueryCard } from '@/components/saved-queries/QueryCard'
```

## Basic Usage

```tsx
import { QueryCard } from '@/components/saved-queries'
import type { SavedQueryRecord } from '@/types/storage'

function MyQueriesList() {
  const [queries, setQueries] = useState<SavedQueryRecord[]>([])

  return (
    <div className="space-y-3">
      {queries.map((query) => (
        <QueryCard
          key={query.id}
          query={query}
          onLoad={(q) => loadInEditor(q.query_text)}
          onEdit={(q) => openEditDialog(q)}
          onDelete={(id) => deleteQuery(id)}
          onDuplicate={(id) => duplicateQuery(id)}
          onToggleFavorite={(id) => toggleFavorite(id)}
        />
      ))}
    </div>
  )
}
```

## Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `query` | `SavedQueryRecord` | Yes | The saved query data to display |
| `onLoad` | `(query: SavedQueryRecord) => void` | Yes | Called when user loads the query |
| `onEdit` | `(query: SavedQueryRecord) => void` | Yes | Called when user edits the query |
| `onDelete` | `(id: string) => void` | Yes | Called when user confirms deletion |
| `onDuplicate` | `(id: string) => void` | Yes | Called when user duplicates the query |
| `onToggleFavorite` | `(id: string) => void` | Yes | Called when user toggles favorite status |
| `showSyncStatus` | `boolean` | No | Show sync status badge (default: false) |

## Examples

### With Sync Status (Individual Tier)

```tsx
<QueryCard
  query={query}
  onLoad={handleLoad}
  onEdit={handleEdit}
  onDelete={handleDelete}
  onDuplicate={handleDuplicate}
  onToggleFavorite={handleToggleFavorite}
  showSyncStatus={userTier === 'individual'}
/>
```

### Grid Layout

```tsx
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  {queries.map((query) => (
    <QueryCard key={query.id} query={query} {...handlers} />
  ))}
</div>
```

### Favorites Only

```tsx
const favoriteQueries = queries.filter(q => q.is_favorite)

<div className="space-y-3">
  {favoriteQueries.map((query) => (
    <QueryCard key={query.id} query={query} {...handlers} />
  ))}
</div>
```

## Accessibility

The QueryCard component is fully accessible and meets WCAG 2.1 Level AA standards:

### Keyboard Navigation

- **Tab**: Focus on the card
- **Enter / Space**: Load the query
- **Tab**: Move to favorite star button
- **Enter / Space**: Toggle favorite
- **Tab**: Move to actions dropdown
- **Enter / Space**: Open dropdown menu
- **Arrow Keys**: Navigate menu items
- **Enter**: Activate menu item
- **Escape**: Close dropdown

### Screen Reader Support

All interactive elements have proper ARIA labels:
- Card: "Load query: [Query Title]"
- Star button: "Add to favorites" / "Remove from favorites"
- Dropdown: "Query actions"
- Sync badge: "Synced to cloud" / "Not synced"
- Timestamp: "Updated [relative time]"

### Visual Accessibility

- High contrast focus rings (3:1 minimum)
- Text contrast meets WCAG AA (4.5:1)
- Clear hover states
- Color is not the only indicator (icons + text)

See [QueryCard.a11y.md](./QueryCard.a11y.md) for full accessibility checklist.

## Performance

The QueryCard component is optimized for rendering in large lists:

### Metrics

- **Render time**: ~8ms per card
- **Memory**: ~3KB per card
- **Bundle size**: ~20KB (including dependencies)

### For Large Lists (100+ queries)

Use virtualization to maintain smooth scrolling:

```tsx
import { useVirtualizer } from '@tanstack/react-virtual'

function VirtualQueryList({ queries }: { queries: SavedQueryRecord[] }) {
  const parentRef = useRef<HTMLDivElement>(null)

  const virtualizer = useVirtualizer({
    count: queries.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 120,
  })

  return (
    <div ref={parentRef} className="h-[600px] overflow-auto">
      <div style={{ height: `${virtualizer.getTotalSize()}px`, position: 'relative' }}>
        {virtualizer.getVirtualItems().map((item) => (
          <div
            key={queries[item.index].id}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              transform: `translateY(${item.start}px)`,
            }}
          >
            <QueryCard query={queries[item.index]} {...handlers} />
          </div>
        ))}
      </div>
    </div>
  )
}
```

See [QueryCard.performance.md](./QueryCard.performance.md) for optimization guide.

## Styling

The component uses Tailwind CSS and respects your theme configuration:

- **Background**: Uses `bg-card` and `text-card-foreground`
- **Hover**: Uses `hover:bg-accent/50` for subtle feedback
- **Borders**: Uses `border` color from theme
- **Focus**: Uses `ring` color from theme

### Dark Mode

Automatically adapts to dark mode via Tailwind's dark mode support.

### Customization

Override styles using className:

```tsx
<QueryCard
  query={query}
  {...handlers}
  className="border-2 border-blue-500" // Custom border
/>
```

## Component Structure

```
QueryCard
├── Card (container)
│   ├── CardHeader
│   │   ├── Title + Favorite Star
│   │   ├── Description (truncated)
│   │   └── Actions Dropdown
│   └── CardContent
│       └── Metadata (folder, tags, sync, timestamp)
└── Delete Confirmation Dialog
```

## Testing

### Unit Tests

Run the test suite:

```bash
npm test -- QueryCard.test.tsx
```

See [QueryCard.test.tsx](./QueryCard.test.tsx) for test examples.

### Manual Testing

1. **Card Click**: Click card body to load query
2. **Star Click**: Click star to toggle favorite (doesn't load query)
3. **Dropdown Click**: Click dropdown to see actions (doesn't load query)
4. **Keyboard**: Tab through elements and use Enter/Space
5. **Delete**: Click delete and confirm in dialog
6. **Hover**: Hover to see background change

## Dependencies

- **React**: ^18.0.0
- **date-fns**: For relative time formatting
- **lucide-react**: For icons
- **@radix-ui/react-dropdown-menu**: For dropdown menu
- **@radix-ui/react-dialog**: For delete confirmation
- **Tailwind CSS**: For styling
- **class-variance-authority**: For badge variants

## Browser Support

- Chrome/Edge: ✅ Latest 2 versions
- Firefox: ✅ Latest 2 versions
- Safari: ✅ Latest 2 versions
- Mobile Safari: ✅ iOS 12+
- Chrome Android: ✅ Latest 2 versions

## Related Components

- **SaveQueryDialog**: For creating/editing saved queries
- **SavedQueriesPanel**: Container for list of QueryCards

## Contributing

When modifying QueryCard:

1. **Maintain Accessibility**: Test with keyboard and screen reader
2. **Performance**: Profile with 100+ cards
3. **Tests**: Update tests for new features
4. **Documentation**: Update this README

## License

Part of SQL Studio - see project license.

## Support

For issues or questions:
- Check the [accessibility checklist](./QueryCard.a11y.md)
- Review [performance guide](./QueryCard.performance.md)
- See [usage examples](./QueryCard.example.tsx)
- File an issue in the main repository
