# Drill-Down System Guide

## Overview

The drill-down system transforms static reports into interactive exploration tools. Users can click chart elements to see underlying details, filter data dynamically, and navigate between related reports.

## Architecture

### Core Components

1. **useDrillDown Hook** (`drill-down-handler.tsx`)
   - Manages drill-down state and actions
   - Handles URL synchronization
   - Provides keyboard shortcuts
   - Coordinates all drill-down interactions

2. **DetailDrawer** (`detail-drawer.tsx`)
   - Slide-out panel for detail views
   - Shows filtered data in paginated table
   - Export functionality
   - Loading/error states

3. **CrossFilterBar** (`cross-filter-bar.tsx`)
   - Displays active filters
   - Quick filter removal
   - Clear all functionality

4. **DrillDownBreadcrumbs** (`drill-down-breadcrumbs.tsx`)
   - Shows navigation path
   - Back navigation support
   - Visual hierarchy

5. **Enhanced ChartRenderer** (`chart-renderer.tsx`)
   - Click handlers on all chart types
   - Visual feedback (cursor, active states)
   - Contextual tooltips

## Drill-Down Types

### 1. Detail View
Show underlying records for a clicked data point.

```typescript
const drillDownConfig: DrillDownConfig = {
  enabled: true,
  type: 'detail',
  detailQuery: `
    SELECT *
    FROM orders
    WHERE status = :clickedValue
    ORDER BY created_at DESC
  `
}
```

**Use Cases:**
- Click bar in status chart → see all orders with that status
- Click line point in trend → see transactions for that day
- Click pie slice → see detail for that category

### 2. Cross-Filter
Apply filter to other components on the same dashboard.

```typescript
const drillDownConfig: DrillDownConfig = {
  enabled: true,
  type: 'filter',
  filterField: 'product_category'
}
```

**Use Cases:**
- Click category in one chart → filter all other charts by that category
- Click region in map → show region-specific data everywhere
- Interactive exploration across multiple views

### 3. Related Report Navigation
Navigate to another report with context.

```typescript
const drillDownConfig: DrillDownConfig = {
  enabled: true,
  type: 'related-report',
  target: 'report-abc-123', // Report ID
  parameters: {
    'customer_id': 'clickedValue', // Map click value to param
    'date_from': 'startDate' // Map from context
  }
}
```

**Use Cases:**
- Click customer in summary → navigate to customer detail report
- Click product → see product performance report
- Hierarchical report navigation

### 4. External URL
Open external resource with context.

```typescript
const drillDownConfig: DrillDownConfig = {
  enabled: true,
  type: 'url',
  target: 'https://admin.example.com/orders/{clickedValue}'
}
```

**Use Cases:**
- Click order ID → open admin panel
- Click user → view profile in external system
- Link to documentation or help resources

## Implementation Examples

### Basic Detail Drill-Down

```typescript
import { useDrillDown } from '@/components/reports/drill-down-handler'
import { DetailDrawer } from '@/components/reports/detail-drawer'
import { ChartRenderer } from '@/components/reports/chart-renderer'

function MyReport() {
  const {
    executeDrillDown,
    detailDrawerOpen,
    detailData,
    detailLoading,
    detailError,
    closeDetailDrawer
  } = useDrillDown({
    executeQuery: async (sql) => {
      // Execute query and return results
      const response = await api.executeQuery({ sql })
      return response.data
    }
  })

  const handleChartClick = (context: DrillDownContext) => {
    const config: DrillDownConfig = {
      enabled: true,
      type: 'detail',
      detailQuery: `
        SELECT order_id, customer_name, amount, created_at
        FROM orders
        WHERE status = :clickedValue
        ORDER BY created_at DESC
      `
    }

    executeDrillDown(config, context)
  }

  return (
    <>
      <ChartRenderer
        data={chartData}
        drillDownConfig={{ enabled: true, type: 'detail' }}
        onDrillDown={handleChartClick}
      />

      <DetailDrawer
        open={detailDrawerOpen}
        onClose={closeDetailDrawer}
        title="Order Details"
        loading={detailLoading}
        data={detailData}
        error={detailError}
      />
    </>
  )
}
```

### Cross-Filtering Dashboard

```typescript
import { useDrillDown } from '@/components/reports/drill-down-handler'
import { CrossFilterBar } from '@/components/reports/cross-filter-bar'

function FilterableDashboard() {
  const {
    executeDrillDown,
    activeFilters,
    clearFilter,
    clearAllFilters
  } = useDrillDown({
    onFilterChange: (filters) => {
      // Filters changed - re-query all components
      refreshAllComponents(filters)
    }
  })

  const handleCategoryClick = (context: DrillDownContext) => {
    const config: DrillDownConfig = {
      enabled: true,
      type: 'filter',
      filterField: 'category'
    }

    executeDrillDown(config, context)
  }

  return (
    <>
      <CrossFilterBar
        activeFilters={activeFilters}
        onClearFilter={clearFilter}
        onClearAll={clearAllFilters}
      />

      {/* All charts respect activeFilters */}
      <ChartRenderer
        data={getFilteredData(activeFilters)}
        drillDownConfig={{ enabled: true, type: 'filter', filterField: 'category' }}
        onDrillDown={handleCategoryClick}
      />
    </>
  )
}
```

### Report Linking

```typescript
const handleProductClick = (context: DrillDownContext) => {
  const config: DrillDownConfig = {
    enabled: true,
    type: 'related-report',
    target: productDetailReportId,
    parameters: {
      'product_id': 'clickedValue',
      'date_range': 'last_30_days'
    }
  }

  executeDrillDown(config, context)
}
```

## Query Interpolation

The system supports two parameter styles:

### Named Parameters (`:paramName`)
```sql
SELECT * FROM orders WHERE status = :clickedValue
```

### Brace Parameters (`{paramName}`)
```sql
SELECT * FROM orders WHERE customer_id = {customerId}
```

**Available Variables:**
- `clickedValue` - The clicked data point value
- `field` - The field name that was clicked
- Any values from `context.filters`
- Any values from `context.additionalData`

## Keyboard Shortcuts

- **Alt + ←** - Go back in drill-down history
- **Esc** - Close detail drawer
- **Alt + C** - Clear all filters

## URL State

Filters are synchronized to URL for:
- Shareable links with active filters
- Browser back/forward support
- Bookmark-able filtered views

Example URL:
```
/reports/abc123?filters=%7B%22category%22%3A%22Electronics%22%7D
```

## Performance Considerations

### Debouncing
- Filter changes are debounced (300ms) before re-querying
- Reduces server load during rapid filter changes

### Caching
- Detail queries are cached by query hash
- Prevents redundant queries for same drill-down

### Lazy Loading
- Detail data only loaded when drawer opens
- Charts render before detail queries execute

### Memoization
- Drill-down context calculations memoized
- Re-renders minimized during filter changes

## Best Practices

### 1. Design for Exploration
- Make interactive elements obvious (cursor changes, hover states)
- Show hints in tooltips ("Click to view details")
- Provide clear navigation paths (breadcrumbs)

### 2. Limit Detail Query Scope
```typescript
// Good: Specific, fast query
detailQuery: `
  SELECT order_id, amount, status
  FROM orders
  WHERE status = :clickedValue
  LIMIT 1000
`

// Bad: Unbounded query
detailQuery: `SELECT * FROM orders`
```

### 3. Handle Errors Gracefully
- Show clear error messages in DetailDrawer
- Don't break entire dashboard on drill-down failure
- Provide retry options

### 4. Progressive Disclosure
- Start with summary views
- Drill into details on demand
- Don't overwhelm with data upfront

### 5. Preserve Context
- Show active filters prominently
- Maintain breadcrumb trail
- Enable easy back navigation

## Testing

### Unit Tests
```typescript
test('drill-down executes detail query', async () => {
  const executeQuery = jest.fn()
  const { executeDrillDown } = useDrillDown({ executeQuery })

  await executeDrillDown(
    { enabled: true, type: 'detail', detailQuery: 'SELECT * FROM test' },
    { clickedValue: 'test', field: 'status' }
  )

  expect(executeQuery).toHaveBeenCalledWith('SELECT * FROM test')
})
```

### Integration Tests
```typescript
test('cross-filtering updates all components', async () => {
  render(<Dashboard />)

  // Click chart element
  fireEvent.click(screen.getByText('Electronics'))

  // Verify filter applied
  expect(screen.getByText('category = Electronics')).toBeInTheDocument()

  // Verify all charts re-queried
  expect(mockExecuteQuery).toHaveBeenCalledTimes(3)
})
```

## Common Patterns

### Conditional Drill-Down
```typescript
const drillDownConfig = useMemo(() => {
  if (userRole !== 'admin') return undefined

  return {
    enabled: true,
    type: 'detail',
    detailQuery: adminDetailQuery
  }
}, [userRole])
```

### Multi-Level Drill-Down
```typescript
// Level 1: Region → States
const regionConfig = {
  enabled: true,
  type: 'filter',
  filterField: 'region'
}

// Level 2: State → Cities (maintains region filter)
const stateConfig = {
  enabled: true,
  type: 'detail',
  detailQuery: `
    SELECT * FROM locations
    WHERE region = :region
    AND state = :clickedValue
  `
}
```

### Dynamic Detail Queries
```typescript
const getDetailConfig = (chartType: string): DrillDownConfig => {
  const queries = {
    'orders': 'SELECT * FROM orders WHERE status = :clickedValue',
    'revenue': 'SELECT * FROM transactions WHERE category = :clickedValue',
    'users': 'SELECT * FROM users WHERE segment = :clickedValue'
  }

  return {
    enabled: true,
    type: 'detail',
    detailQuery: queries[chartType]
  }
}
```

## Troubleshooting

### Issue: Click not triggering drill-down
**Check:**
- `drillDownConfig.enabled` is true
- `onDrillDown` callback provided to ChartRenderer
- Chart type supports click events (all types do)

### Issue: Query interpolation not working
**Check:**
- Parameter names match exactly (`:clickedValue` vs `clickedValue`)
- Values exist in context object
- SQL syntax correct for your database

### Issue: Filters not applying
**Check:**
- `onFilterChange` callback provided to useDrillDown
- Components using activeFilters to query data
- URL params being read on mount

### Issue: Detail drawer not showing data
**Check:**
- `executeQuery` function properly implemented
- Query returning expected format (columns + rows)
- Error state being handled

## Future Enhancements

Potential additions to consider:

1. **Drill-Through Templates**
   - Pre-configured drill patterns
   - One-click setup for common scenarios

2. **Drill-Down Analytics**
   - Track most-used drill paths
   - Optimize popular queries

3. **Collaborative Filtering**
   - Share filtered views with team
   - Saved filter sets

4. **Smart Suggestions**
   - Auto-suggest related reports
   - Recommend drill-down paths

5. **Export Drill Path**
   - Save exploration path as report
   - Replay user journey

## Support

For questions or issues:
1. Check this guide
2. Review component TypeScript types
3. Examine example implementations
4. Consult team documentation
