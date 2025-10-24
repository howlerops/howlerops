# QueryCard Quick Start Guide

## 30-Second Integration

```tsx
import { QueryCard } from '@/components/saved-queries'

// In your component:
<QueryCard
  query={savedQuery}
  onLoad={(q) => setEditorContent(q.query_text)}
  onEdit={(q) => openEditDialog(q)}
  onDelete={(id) => deleteQuery(id)}
  onDuplicate={(id) => duplicateQuery(id)}
  onToggleFavorite={(id) => toggleFavorite(id)}
/>
```

## Props at a Glance

| Prop | What it does |
|------|-------------|
| `query` | The saved query data |
| `onLoad` | User clicked card → load query |
| `onEdit` | User clicked Edit → open edit dialog |
| `onDelete` | User confirmed delete → remove query |
| `onDuplicate` | User clicked Duplicate → copy query |
| `onToggleFavorite` | User clicked star → toggle favorite |
| `showSyncStatus?` | Show "Synced" badge (Individual tier) |

## Common Patterns

### List View
```tsx
<div className="space-y-3">
  {queries.map(q => <QueryCard key={q.id} query={q} {...handlers} />)}
</div>
```

### Grid View
```tsx
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  {queries.map(q => <QueryCard key={q.id} query={q} {...handlers} />)}
</div>
```

### With Sync Status
```tsx
<QueryCard query={q} {...handlers} showSyncStatus={true} />
```

## What it Displays

1. **Title** (truncated if too long)
2. **Description** (max 2 lines)
3. **Favorite Star** (filled if favorite)
4. **Folder Badge** (if query has folder)
5. **Tags** (as badges)
6. **Sync Status** (if showSyncStatus=true)
7. **Last Updated** (relative time like "2 hours ago")
8. **Actions Menu** (Load, Edit, Duplicate, Favorite, Delete)

## User Interactions

- **Click Card** → Loads query (calls `onLoad`)
- **Click Star** → Toggles favorite (calls `onToggleFavorite`)
- **Click Menu** → Shows actions dropdown
- **Click Delete** → Shows confirmation dialog

## Accessibility

- ✅ Keyboard: Tab, Enter, Space, Arrows all work
- ✅ Screen Reader: Full ARIA labels
- ✅ Focus: Visible focus rings
- ✅ Colors: High contrast, not sole indicator

## Performance

- Fast: ~8ms render per card
- Small: ~3KB memory per card
- Virtual: Use virtualization for 100+ cards

## Files Reference

- **Main**: `QueryCard.tsx` (component code)
- **Types**: `QueryCardProps` interface
- **Examples**: `QueryCard.example.tsx`
- **Tests**: `QueryCard.test.tsx`
- **A11y**: `QueryCard.a11y.md`
- **Perf**: `QueryCard.performance.md`
- **Full Docs**: `QueryCard.README.md`

## Troubleshooting

**Card not clickable?**
- Make sure you're not clicking inside `[data-no-propagate]` elements

**Star not showing correctly?**
- Check `query.is_favorite` boolean value

**Sync status not showing?**
- Set `showSyncStatus={true}` prop

**Delete not working?**
- Check `onDelete` handler is called with correct ID

**Handlers firing on every render?**
- Wrap handlers in `useCallback` for performance
