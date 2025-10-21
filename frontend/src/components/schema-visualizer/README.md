# Schema Visualizer - Interactive Features Guide

## Quick Reference

### Component Architecture

```
SchemaVisualizer (Main Container)
├── Custom Edge (Hover tooltips)
├── Table Node (Focus mode styling)
└── Relationship Inspector (Edge details panel)
```

## Features

### 1. Edge Hover Tooltips

Hover over any edge to see:
- Source: `table.column`
- Target: `table.column`
- Type: `(1:1)`, `(1:N)`, or `(N:1)`

**Implementation:** `custom-edge.tsx`

### 2. Table Focus Mode

Click a table to highlight its relationships:
- Focused table: Blue border + ring glow
- Connected edges: Full opacity
- Other elements: Dimmed

**Exit:** Click table again, click background, or press `Escape`

**Implementation:** `table-node.tsx` + `schema-visualizer.tsx`

### 3. Relationship Inspector

Click an edge to see:
- Relationship type and cardinality
- Source/target column details
- Constraint information
- Copy SQL button

**Implementation:** `relationship-inspector.tsx`

### 4. Performance Optimizations

- **>50 tables:** Animations disabled
- **>100 tables:** Warning banner shown
- **Debouncing:** 50ms hover, 300ms search
- **Smart rendering:** Only visible elements

## State Management

```typescript
// Focus mode
const [selectedTableId, setSelectedTableId] = useState<string | null>(null)

// Edge interaction
const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null)

// Inspector panel
const [selectedEdge, setSelectedEdge] = useState<EdgeData | null>(null)
```

## Event Handlers

```typescript
// Node click → Focus mode
onNodeClick={(event, node) => {
  setSelectedTableId(prevId => prevId === node.id ? null : node.id)
}}

// Edge click → Inspector
onEdgeClick={(event, edge) => {
  // Show relationship inspector panel
}}

// Pane click → Deselect
onPaneClick={() => {
  setSelectedTableId(null)
  setSelectedEdge(null)
}}
```

## Performance Thresholds

| Tables | Behavior |
|--------|----------|
| < 50   | All animations enabled |
| 50-100 | Animations disabled |
| > 100  | Warning banner + suggestions |
| > 200  | Auto-switch to grid layout |

## Keyboard Shortcuts

- `Escape` - Exit focus mode / Close inspector
- Click table - Toggle focus mode
- Click edge - Open inspector
- Click background - Deselect all

## Styling Classes

### Table Node States
```typescript
isFocused && 'border-blue-500 border-3 shadow-2xl ring-4 ring-blue-500/30'
isDimmed && 'opacity-40'
```

### Edge States
```typescript
isHighlighted && { strokeWidth: +1, opacity: 1 }
isDimmed && { opacity: 0.15 }
```

## Type Definitions

See `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/types/schema-visualizer.ts`

```typescript
interface TableConfig {
  // ... existing fields
  isFocused?: boolean
  isDimmed?: boolean
}

interface SchemaVisualizerEdge {
  // ... existing fields
  data?: {
    onEdgeHover?: (edgeId: string | null) => void
    isHighlighted?: boolean
    isDimmed?: boolean
  }
}
```

## Testing Checklist

- [ ] Edge hover shows tooltip
- [ ] Edge hover highlights connected tables
- [ ] Table click enters focus mode
- [ ] Focus mode dims unrelated elements
- [ ] Click table again exits focus mode
- [ ] Escape key exits focus mode
- [ ] Edge click shows inspector
- [ ] Inspector shows correct data
- [ ] Copy SQL works
- [ ] Inspector closes properly
- [ ] Performance warning appears (>100 tables)
- [ ] Animations disabled (>50 tables)
- [ ] Debouncing works smoothly

## Troubleshooting

### Edge tooltips not showing
- Check `edgeTypes` prop is set on ReactFlow
- Verify `CustomEdge` is imported correctly

### Focus mode not working
- Ensure `onNodeClick` handler is set
- Check `selectedTableId` state updates

### Inspector not appearing
- Verify `onEdgeClick` handler is set
- Check edge data contains `EdgeConfig`

### Performance issues
- Enable `onlyRenderVisibleElements={true}`
- Check debounce values (50ms/300ms)
- Verify animation disabling (>50 tables)

## File Locations

```
/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/components/schema-visualizer/
├── custom-edge.tsx              # Edge hover tooltips
├── relationship-inspector.tsx   # Edge click panel
├── schema-visualizer.tsx        # Main component
├── table-node.tsx              # Table styling
└── README.md                   # This file
```

## Next Steps

To use these features:

1. Import `SchemaVisualizerWrapper` in your app
2. Pass schema data from `use-schema-introspection`
3. All interactive features work automatically

```typescript
import { SchemaVisualizerWrapper } from '@/components/schema-visualizer'

<SchemaVisualizerWrapper
  schema={schemaData}
  onClose={() => setShowVisualizer(false)}
/>
```

## Support

For issues or questions:
1. Check TypeScript errors: `npx tsc --noEmit`
2. Review console for warnings
3. Verify ReactFlow version compatibility
4. Check dependency versions match requirements
