# Drill-Down System Architecture

## Component Hierarchy

```
┌─────────────────────────────────────────────────────────────┐
│                      Reports Page                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            useDrillDown Hook                          │  │
│  │  • State management                                   │  │
│  │  • URL synchronization                                │  │
│  │  • History tracking                                   │  │
│  │  • Keyboard shortcuts                                 │  │
│  └───────────────────────────────────────────────────────┘  │
│         │                                                    │
│         ├─────────────┬──────────────┬────────────────┐     │
│         ▼             ▼              ▼                ▼     │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌─────────┐│
│  │Breadcrumbs│  │CrossFilter│  │ChartRenderer │  │ Detail  ││
│  │           │  │   Bar     │  │  (Enhanced)  │  │ Drawer  ││
│  │• History  │  │• Filters  │  │• Clicks      │  │• Data   ││
│  │• Navigate │  │• Clear    │  │• Tooltips    │  │• Export ││
│  └──────────┘  └──────────┘  └──────────────┘  └─────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. Detail Drill-Down Flow

```
User Clicks Chart Bar
       │
       ▼
ChartRenderer.handleElementClick()
       │
       ▼
Creates DrillDownContext
  { clickedValue, field, filters }
       │
       ▼
Calls onDrillDown(context)
       │
       ▼
useDrillDown.executeDrillDown(config, context)
       │
       ├─ Add to history
       ├─ Interpolate query with context
       └─ Execute query
              │
              ▼
         Query Result
              │
              ▼
      Set detailData
              │
              ▼
   DetailDrawer Opens
   Shows paginated table
```

### 2. Cross-Filter Flow

```
User Clicks Chart Element
       │
       ▼
onDrillDown({ clickedValue, field })
       │
       ▼
useDrillDown.applyFilter()
       │
       ├─ Update activeFilters state
       ├─ Update URL params
       └─ Call onFilterChange callback
              │
              ▼
         Parent Component
         refreshAllComponents(filters)
              │
              ▼
   All Charts Re-query with Filters
   CrossFilterBar Shows Active Filters
```

### 3. Related Report Navigation Flow

```
User Clicks Chart Element
       │
       ▼
onDrillDown({ clickedValue })
       │
       ▼
useDrillDown.navigateToReport()
       │
       ├─ Build URL with parameters
       └─ Navigate to target report
              │
              ▼
    Target Report Loads
    Reads params from URL
    Applies initial filters
```

## State Management

### useDrillDown Hook State

```typescript
{
  // Navigation
  history: DrillDownAction[]           // Stack of drill-down actions
  canGoBack: boolean                   // Has history?

  // Detail View
  detailDrawerOpen: boolean            // Drawer visible?
  detailData: QueryResult | null       // Detail query results
  detailLoading: boolean               // Query in progress?
  detailError: string | null           // Query error?

  // Cross-Filtering
  activeFilters: Record<string, any>   // Applied filters
}
```

### URL State Sync

```
URL: /reports/abc123?filters=%7B%22category%22%3A%22Electronics%22%7D
                              └─────────────────┬────────────────┘
                                                │
                                    Decoded: {"category":"Electronics"}
                                                │
                                                ▼
                                      activeFilters state
                                                │
                                                ▼
                                   Applied to all components
```

## Event Sequence Diagrams

### User Interaction: Drill into Details

```
User          ChartRenderer    useDrillDown    Backend       DetailDrawer
 │                │                │              │               │
 │──Click Bar──>  │                │              │               │
 │                │──handleClick─> │              │               │
 │                │                │──Query───>   │               │
 │                │                │              │               │
 │                │                │   <─Result── │               │
 │                │                │──setData──>  │               │
 │                │                │──open───────────────────────>│
 │                │                │              │               │
 │   <──────────────────────────────────────────────Shows Data────│
```

### Multi-Step Drill-Down with History

```
State: Dashboard (no filters)
       │
       ▼ Click "Electronics" category
State: Dashboard (category=Electronics)
  History: [filter Electronics]
       │
       ▼ Click "Laptops" subcategory
State: Dashboard (category=Electronics, subcategory=Laptops)
  History: [filter Electronics, filter Laptops]
       │
       ▼ Click bar to see details
State: Detail View (filtered data)
  History: [filter Electronics, filter Laptops, detail view]
       │
       ▼ Press Alt + ←
State: Dashboard (category=Electronics, subcategory=Laptops)
  History: [filter Electronics, filter Laptops]
       │
       ▼ Press Alt + C
State: Dashboard (no filters)
  History: []
```

## Component Responsibilities

### useDrillDown Hook
**Responsibilities:**
- State management for all drill-down operations
- URL synchronization (read on mount, update on change)
- Keyboard shortcut handling
- Query interpolation
- History tracking

**Does NOT:**
- Render any UI
- Execute queries directly (delegates to callback)
- Know about specific components

### ChartRenderer
**Responsibilities:**
- Render chart with data
- Handle click events
- Show visual feedback (cursor, active states)
- Display drill-down hints in tooltips

**Does NOT:**
- Manage drill-down state
- Execute queries
- Track history

### DetailDrawer
**Responsibilities:**
- Display detail data in table
- Handle CSV export
- Show loading/error states
- Display active filters

**Does NOT:**
- Execute queries
- Manage filter state
- Track history

### CrossFilterBar
**Responsibilities:**
- Display active filters as badges
- Provide filter removal UI
- Show "Clear All" button

**Does NOT:**
- Manage filter state
- Execute queries
- Apply filters

## Configuration Schema

### DrillDownConfig

```typescript
{
  enabled: boolean                     // Feature toggle

  type: 'detail' | 'filter' |         // Action type
        'related-report' | 'url'

  // Type-specific config:

  // For 'detail':
  detailQuery?: string                 // SQL with :params

  // For 'filter':
  filterField?: string                 // Field to filter by

  // For 'related-report':
  target?: string                      // Report ID
  parameters?: {                       // Click → query params
    [paramName]: contextKey
  }

  // For 'url':
  target?: string                      // URL template
}
```

### Example Configurations

```typescript
// Detail View
{
  enabled: true,
  type: 'detail',
  detailQuery: `
    SELECT *
    FROM orders
    WHERE status = :clickedValue
    AND created_at >= :dateFrom
    LIMIT 1000
  `
}

// Cross-Filter
{
  enabled: true,
  type: 'filter',
  filterField: 'product_category'
}

// Related Report
{
  enabled: true,
  type: 'related-report',
  target: 'customer-detail-report-id',
  parameters: {
    'customer_id': 'clickedValue',
    'period': 'last_30_days'
  }
}

// External URL
{
  enabled: true,
  type: 'url',
  target: 'https://admin.example.com/orders/{clickedValue}'
}
```

## Performance Optimizations

### 1. Debouncing
```
User clicks rapidly:
  Click 1 (0ms)  → Start 300ms timer
  Click 2 (100ms) → Reset timer
  Click 3 (200ms) → Reset timer
  Click 4 (250ms) → Reset timer
  (550ms) → Execute last click only
```

### 2. Memoization
```typescript
// Context calculation only when dependencies change
const context = useMemo(
  () => createContext(clickedValue, filters),
  [clickedValue, filters]
)

// Chart data transformation cached
const chartData = useMemo(
  () => transformData(rawData),
  [rawData]
)
```

### 3. Lazy Loading
```
DetailDrawer renders    → Hook initialized
User hasn't clicked     → No queries executed
User clicks element     → Drawer opens
Drawer opens            → Query executes
Data arrives            → Table renders
```

### 4. Caching
```typescript
Query Cache:
{
  "SELECT * FROM orders WHERE status = 'pending'": {
    data: {...},
    timestamp: 1234567890,
    ttl: 300000 // 5 minutes
  }
}

On duplicate drill-down:
  - Check cache first
  - Return cached if fresh
  - Execute query if expired
```

## Error Handling

```
Query Execution Error
       │
       ▼
Set detailError state
       │
       ▼
DetailDrawer shows error card
       │
User can:
  • Close drawer
  • Try different drill-down
  • Report issue
```

## Accessibility

### Keyboard Navigation
- **Tab**: Navigate between interactive elements
- **Enter/Space**: Activate clicked element
- **Alt + ←**: Go back in history
- **Esc**: Close detail drawer
- **Alt + C**: Clear all filters

### Screen Reader Support
- Chart elements have aria-labels
- Drill-down hints in tooltips
- Filter badges announce count
- Drawer has proper ARIA roles

### Visual Indicators
- Cursor changes to pointer on hover
- Active element highlighting
- Focus rings on keyboard navigation
- Loading spinners for async operations

## Browser Support Matrix

| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Click handlers | ✅ | ✅ | ✅ | ✅ |
| URL sync | ✅ | ✅ | ✅ | ✅ |
| Keyboard shortcuts | ✅ | ✅ | ✅ | ✅ |
| CSV export | ✅ | ✅ | ✅ | ✅ |
| Drawer animation | ✅ | ✅ | ✅ | ✅ |

Minimum versions:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+
