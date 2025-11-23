# Dashboard Canvas - Interactive Drag-and-Drop Report Builder

A professional, Tableau/Looker-style drag-and-drop dashboard layout builder for the HowlerOps Reports feature.

## Overview

The Dashboard Canvas transforms the report builder from a static component list into an interactive grid canvas where users can:

- **Drag** components to reposition them
- **Resize** components using corner and edge handles
- **Add** components via a floating palette
- **Edit** component configurations
- **Toggle** between Canvas and List views
- **Undo/Redo** layout changes with keyboard shortcuts
- **Auto-save** layout changes after 1 second of inactivity

## Features

### 1. Interactive Grid Layout

Built on `react-grid-layout` with:
- 12-column responsive grid system
- 60px row height for consistent spacing
- Breakpoints: lg (1200px), md (996px), sm (768px), xs (480px)
- Collision prevention and automatic compaction
- Snap-to-grid behavior

### 2. Edit vs View Modes

**Edit Mode (default):**
- Visible drag handles on component headers
- Resize handles on hover
- Component toolbars visible on hover
- Subtle grid background
- "Add Component" floating button

**View Mode:**
- Clean, polished interface
- No layout controls visible
- Focus on data visualization
- Click components to interact with charts/tables

### 3. Component Cards

Each component renders as a draggable card with:

**Visual Elements:**
- Icon and title in header
- Type badge
- Inline data preview/results
- Mini toolbar (edit, run, delete) on hover

**States:**
- Default: Subtle border, white background
- Hover: Shadow elevation, toolbar visible
- Dragging: Strong shadow, slight opacity
- Error: Red alert with error message

### 4. Undo/Redo

- Tracks last 10 layout changes
- Keyboard shortcuts:
  - `Cmd+Z` / `Ctrl+Z` - Undo
  - `Cmd+Shift+Z` / `Ctrl+Shift+Z` - Redo
- Visual indicators for undo/redo availability

### 5. Auto-Save

- Debounced layout updates (1 second delay)
- "Saving..." indicator when changes pending
- Automatic persistence to report definition
- Prevents accidental data loss

### 6. Layout Templates

Pre-built layouts for common dashboard patterns:

```typescript
import { LAYOUT_TEMPLATES } from '@/components/reports/dashboard-canvas'

// Available templates:
- single-kpi: 1 large metric centered
- dual-kpi: 2 metrics side-by-side
- dashboard: 4 KPIs top, 2 charts below
- report: Full-width table layout
- analytics: 3 KPIs, 1 large chart, 2 small charts
```

### 7. Responsive Breakpoints

The grid automatically adjusts for different screen sizes:

| Breakpoint | Width | Columns | Use Case |
|------------|-------|---------|----------|
| lg | 1200px+ | 12 | Desktop, large screens |
| md | 996px+ | 10 | Tablets (landscape) |
| sm | 768px+ | 6 | Tablets (portrait) |
| xs | 480px+ | 4 | Mobile devices |

## Usage

### Basic Integration

```typescript
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'

function ReportBuilder() {
  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])
  const [components, setComponents] = useState<ReportComponent[]>([])

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      results={runResults}
      onLayoutChange={setLayout}
      onComponentEdit={(id) => openEditor(id)}
      onComponentRun={(id) => runComponent(id)}
      onComponentDelete={(id) => removeComponent(id)}
      onAddComponent={(type) => addNewComponent(type)}
      editMode={true}
    />
  )
}
```

### Props Reference

```typescript
interface DashboardCanvasProps {
  // Required
  components: ReportComponent[]        // Array of report components
  layout: ReportLayoutSlot[]           // Grid layout positions
  onLayoutChange: (newLayout) => void  // Layout update callback

  // Optional
  results?: ReportRunResult            // Component execution results
  onComponentClick?: (id) => void      // Component click handler
  onComponentEdit?: (id) => void       // Edit button handler
  onComponentRun?: (id) => void        // Run button handler
  onComponentDelete?: (id) => void     // Delete button handler
  onAddComponent?: (type) => void      // Add component handler
  editMode?: boolean                   // Enable/disable editing (default: true)
}
```

### Layout Slot Format

```typescript
interface ReportLayoutSlot {
  componentId: string  // Unique component identifier
  x: number           // Grid column (0-11 in 12-col grid)
  y: number           // Grid row (0-based)
  w: number           // Width in grid columns (1-12)
  h: number           // Height in grid rows
}

// Example: Full-width component at top
{
  componentId: "abc123",
  x: 0,    // Start at first column
  y: 0,    // First row
  w: 12,   // Full width (all 12 columns)
  h: 4     // 4 rows tall (240px at 60px/row)
}
```

### Component Size Constraints

Set min/max dimensions in component definition:

```typescript
const component: ReportComponent = {
  id: "chart-1",
  title: "Sales Chart",
  type: "chart",
  size: {
    minW: 4,   // Minimum 4 columns wide
    minH: 3,   // Minimum 3 rows tall
    maxW: 12,  // Maximum full width
    maxH: 8    // Maximum 8 rows tall
  },
  // ... other properties
}
```

### Using Layout Templates

```typescript
import { LAYOUT_TEMPLATES, LayoutTemplate } from '@/components/reports/dashboard-canvas'

function applyTemplate(template: LayoutTemplate) {
  const componentIds = components.map(c => c.id)
  const newLayout = LAYOUT_TEMPLATES[template].getLayout(componentIds)
  setLayout(newLayout)
}

// Apply dashboard template
applyTemplate('dashboard')
```

## Component Rendering

### Grid Component Structure

Each grid item contains:

```tsx
<Card>
  {/* Draggable header */}
  <CardHeader className="drag-handle cursor-move">
    <Icon />
    <CardTitle>{component.title}</CardTitle>
    <Badge>{component.type}</Badge>

    {/* Toolbar (visible on hover in edit mode) */}
    <div className="opacity-0 group-hover:opacity-100">
      <Button onClick={onEdit}><Settings /></Button>
      <Button onClick={onRun}><Play /></Button>
      <Button onClick={onDelete}><Trash2 /></Button>
    </div>
  </CardHeader>

  {/* Content area */}
  <CardContent>
    {result ? (
      <ComponentPreview type={type} result={result} />
    ) : (
      <EmptyState title="No data" />
    )}
  </CardContent>
</Card>
```

### Preview Rendering

Different component types render differently:

**Metric**: Large centered value
```tsx
<div className="text-4xl font-bold">
  {value.toLocaleString()}
</div>
```

**Table**: Data grid with pagination
```tsx
<table>
  <thead>...</thead>
  <tbody>
    {rows.slice(0, 10).map(...)}
  </tbody>
</table>
```

**Chart**: (Future) Recharts visualization
**LLM**: Formatted text content

## Performance Optimizations

### 1. Memoization

```typescript
// Grid items are memoized to prevent unnecessary re-renders
const GridComponent = React.memo(
  ({ component, result, ... }) => { ... },
  (prev, next) => {
    // Custom comparison logic
    return prev.component.id === next.component.id &&
           JSON.stringify(prev.component) === JSON.stringify(next.component)
  }
)
```

### 2. Debounced Callbacks

```typescript
// Layout changes debounced to 1 second
const handleLayoutChange = useCallback(
  debounce((newLayout) => {
    onLayoutChange(newLayout)
  }, 1000),
  [onLayoutChange]
)
```

### 3. Efficient Lookups

```typescript
// Results indexed by component ID for O(1) lookup
const resultsByComponentId = useMemo(() => {
  return new Map(results.map(r => [r.componentId, r]))
}, [results])
```

### 4. CSS Transforms

react-grid-layout uses CSS transforms for smooth 60fps animations:

```tsx
<ResponsiveGridLayout
  useCSSTransforms={true}  // Enable GPU-accelerated transforms
  // ...
/>
```

## Styling and Customization

### Custom CSS Classes

Override default react-grid-layout styles in `dashboard-canvas.css`:

```css
/* Dragging placeholder */
.react-grid-item.react-grid-placeholder {
  background: hsl(var(--primary) / 0.1);
  border: 2px dashed hsl(var(--primary) / 0.5);
  opacity: 0.5;
}

/* Resize handles */
.react-resizable-handle::after {
  border-color: hsl(var(--border));
}

/* Grid lines (visible on hover in edit mode) */
.layout::before {
  background-image: repeating-linear-gradient(...);
  opacity: 0;
}
.layout:hover::before {
  opacity: 0.5;
}
```

### Design System Integration

Uses shadcn/ui components throughout:
- `Card`, `CardHeader`, `CardTitle`, `CardContent`
- `Button`, `Badge`, `Alert`
- `DropdownMenu` for component palette
- CSS variables from global theme

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd+Z` / `Ctrl+Z` | Undo last layout change |
| `Cmd+Shift+Z` / `Ctrl+Shift+Z` | Redo layout change |
| `Tab` | Navigate between components (accessibility) |
| `Space` | Select focused component (accessibility) |

## Accessibility

The dashboard canvas is keyboard-accessible:

1. **Tab Navigation**: Move focus between components
2. **Space/Enter**: Select and interact with components
3. **Screen Reader Support**: ARIA labels on interactive elements
4. **Visual Indicators**: Clear focus states and hover effects

```tsx
<Button
  aria-label="Edit component"
  title="Edit component"
  onClick={onEdit}
>
  <Settings />
</Button>
```

## Mobile Responsiveness

On small screens (< 768px):
- Grid automatically stacks to single column
- Components become full-width
- Touch-friendly drag handles
- Simplified toolbar (critical actions only)

## Error Handling

### Component Errors

```tsx
{result?.error ? (
  <Alert variant="destructive">
    <AlertTitle>Error</AlertTitle>
    <AlertDescription>{result.error}</AlertDescription>
  </Alert>
) : (
  <ComponentPreview result={result} />
)}
```

### Empty States

```tsx
{!result ? (
  <EmptyState
    icon={FileQuestion}
    title="No data"
    description="Run this component to see results"
    compact
  />
) : (
  <ComponentPreview result={result} />
)}
```

## Testing

### Unit Tests

```typescript
import { render, screen } from '@testing-library/react'
import { DashboardCanvas } from './dashboard-canvas'

test('renders components in grid layout', () => {
  const components = [
    { id: '1', title: 'Chart 1', type: 'chart' },
    { id: '2', title: 'Metric 1', type: 'metric' },
  ]

  const layout = [
    { componentId: '1', x: 0, y: 0, w: 6, h: 4 },
    { componentId: '2', x: 6, y: 0, w: 6, h: 4 },
  ]

  render(
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={jest.fn()}
    />
  )

  expect(screen.getByText('Chart 1')).toBeInTheDocument()
  expect(screen.getByText('Metric 1')).toBeInTheDocument()
})
```

### Integration Tests

```typescript
test('undo/redo functionality works', () => {
  const onLayoutChange = jest.fn()
  const { rerender } = render(<DashboardCanvas ... />)

  // Make a layout change
  // ... simulate drag

  // Undo
  fireEvent.keyDown(window, { key: 'z', metaKey: true })
  expect(onLayoutChange).toHaveBeenCalledWith(originalLayout)

  // Redo
  fireEvent.keyDown(window, { key: 'z', metaKey: true, shiftKey: true })
  expect(onLayoutChange).toHaveBeenCalledWith(newLayout)
})
```

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Drag start latency | < 16ms | 60fps requirement |
| Layout recalculation | < 50ms | Smooth repositioning |
| Render 20 components | < 200ms | Initial page load |
| Memory usage | < 100MB | For 50-component dashboard |

## Browser Support

Tested and supported on:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

React-grid-layout uses CSS transforms and flexbox, so modern browsers required.

## Troubleshooting

### Components not draggable

Ensure `editMode={true}` and `draggableHandle=".drag-handle"` is set:

```tsx
<DashboardCanvas editMode={true} ... />

// In component card:
<CardHeader className="drag-handle cursor-move">
```

### Layout changes not persisting

Check `onLayoutChange` callback is wired up:

```tsx
const handleLayoutChange = (newLayout) => {
  // Update report definition
  onChange({
    definition: {
      layout: newLayout,
      components,
    },
  })
}

<DashboardCanvas onLayoutChange={handleLayoutChange} />
```

### Resize handles not visible

Ensure custom CSS is imported:

```tsx
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './dashboard-canvas.css'
```

### Grid not responsive

Verify breakpoints configuration:

```tsx
<ResponsiveGridLayout
  breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480 }}
  cols={{ lg: 12, md: 10, sm: 6, xs: 4 }}
  // ...
/>
```

## Future Enhancements

Potential improvements for future iterations:

1. **Chart Rendering**: Integrate Recharts for live chart visualization
2. **Real-time Collaboration**: Multi-user editing with conflict resolution
3. **Layout Sharing**: Export/import layouts as JSON templates
4. **Component Library**: Pre-built component templates
5. **Advanced Filters**: Filter components on canvas by type/tag
6. **Performance Mode**: Virtual scrolling for 100+ component dashboards
7. **Mobile Editor**: Touch-optimized layout editor for tablets
8. **Snapshot Comparison**: Visual diff between layout versions

## Related Documentation

- [Report Builder Guide](/docs/reports/report-builder.md)
- [Component Types Reference](/docs/reports/component-types.md)
- [react-grid-layout Documentation](https://github.com/react-grid-layout/react-grid-layout)
- [shadcn/ui Components](https://ui.shadcn.com)

## License

MIT
