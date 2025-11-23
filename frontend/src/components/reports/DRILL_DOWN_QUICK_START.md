# Drill-Down Quick Start

## 5-Minute Integration Guide

### 1. Import Dependencies

```typescript
import { useDrillDown } from '@/components/reports/drill-down-handler'
import { DetailDrawer } from '@/components/reports/detail-drawer'
import { CrossFilterBar } from '@/components/reports/cross-filter-bar'
import { DrillDownBreadcrumbs } from '@/components/reports/drill-down-breadcrumbs'
import { ChartRenderer } from '@/components/reports/chart-renderer'
```

### 2. Initialize Hook

```typescript
const {
  executeDrillDown,
  detailDrawerOpen,
  detailData,
  detailLoading,
  detailError,
  closeDetailDrawer,
  activeFilters,
  clearFilter,
  clearAllFilters,
  history,
  goBack,
  canGoBack
} = useDrillDown({
  executeQuery: async (sql) => {
    // Your query execution
    const result = await reportsService.executeQuery({ sql })
    return result
  },
  onFilterChange: (filters) => {
    // Re-query with filters
    refreshData(filters)
  }
})
```

### 3. Create Drill-Down Handler

```typescript
const handleChartClick = (context: DrillDownContext) => {
  const config: DrillDownConfig = {
    enabled: true,
    type: 'detail', // or 'filter', 'related-report', 'url'
    detailQuery: `
      SELECT * FROM orders
      WHERE status = :clickedValue
      LIMIT 1000
    `
  }

  executeDrillDown(config, context)
}
```

### 4. Add to JSX

```typescript
return (
  <>
    {/* Breadcrumbs */}
    {history.length > 0 && (
      <DrillDownBreadcrumbs
        history={history}
        onNavigate={(idx) => idx === -1 ? goBack() : goBack()}
      />
    )}

    {/* Filter Bar */}
    <CrossFilterBar
      activeFilters={activeFilters}
      onClearFilter={clearFilter}
      onClearAll={clearAllFilters}
    />

    {/* Interactive Chart */}
    <ChartRenderer
      data={chartData}
      chartConfig={{ variant: 'bar' }}
      drillDownConfig={{ enabled: true, type: 'detail' }}
      onDrillDown={handleChartClick}
    />

    {/* Detail Drawer */}
    <DetailDrawer
      open={detailDrawerOpen}
      onClose={closeDetailDrawer}
      title="Details"
      filters={activeFilters}
      loading={detailLoading}
      data={detailData}
      error={detailError}
    />
  </>
)
```

## Common Patterns

### Detail View (Most Common)

```typescript
// Click chart → See underlying records
const config: DrillDownConfig = {
  enabled: true,
  type: 'detail',
  detailQuery: `
    SELECT *
    FROM orders
    WHERE category = :clickedValue
    ORDER BY created_at DESC
    LIMIT 500
  `
}
```

### Cross-Filter

```typescript
// Click chart → Filter entire dashboard
const config: DrillDownConfig = {
  enabled: true,
  type: 'filter',
  filterField: 'product_category'
}
```

### Related Report

```typescript
// Click chart → Navigate to related report
const config: DrillDownConfig = {
  enabled: true,
  type: 'related-report',
  target: customerDetailReportId,
  parameters: {
    'customer_id': 'clickedValue'
  }
}
```

### External Link

```typescript
// Click chart → Open external resource
const config: DrillDownConfig = {
  enabled: true,
  type: 'url',
  target: 'https://admin.example.com/orders/{clickedValue}'
}
```

## Query Interpolation

### Available Variables
- `:clickedValue` - The value user clicked
- `:field` - The field name
- Any value from `context.filters`

### Examples

```sql
-- Named parameters
SELECT * FROM orders WHERE status = :clickedValue

-- Brace parameters
SELECT * FROM orders WHERE id = {orderId}

-- Multiple parameters
SELECT *
FROM orders
WHERE category = :clickedValue
  AND created_at >= :dateFrom
  AND created_at <= :dateTo
```

## Keyboard Shortcuts

- **Alt + ←** - Go back in drill-down history
- **Esc** - Close detail drawer
- **Alt + C** - Clear all filters

## TypeScript Types

```typescript
interface DrillDownConfig {
  enabled: boolean
  type: 'detail' | 'related-report' | 'filter' | 'url'
  target?: string // Report ID or URL
  filterField?: string // Field to filter by
  detailQuery?: string // SQL for detail view
  parameters?: Record<string, string> // Click → query params
}

interface DrillDownContext {
  clickedValue: unknown // The clicked value
  field: string // Field name
  filters?: Record<string, unknown> // Active filters
  additionalData?: Record<string, unknown> // Extra context
  componentId?: string // Source component
}
```

## Troubleshooting

### Click Not Working
✅ Check `drillDownConfig.enabled === true`
✅ Check `onDrillDown` prop passed to ChartRenderer
✅ Verify chart type supports clicks (all do)

### Query Not Executing
✅ Check `executeQuery` function provided to hook
✅ Verify SQL syntax is correct
✅ Check parameter names match (`:clickedValue`)

### Filters Not Applying
✅ Check `onFilterChange` callback provided
✅ Verify components using `activeFilters` to query
✅ Check filter field name matches data

### Drawer Not Showing Data
✅ Check query returns `{ columns: [...], rows: [[...]] }`
✅ Verify no errors in `detailError`
✅ Check network tab for query response

## Performance Tips

1. **Limit Detail Queries**
   ```sql
   -- Good
   SELECT * FROM orders WHERE ... LIMIT 1000

   -- Bad
   SELECT * FROM orders -- No limit!
   ```

2. **Debounce Filter Changes**
   - Already built-in (300ms)
   - Don't need to implement yourself

3. **Memoize Expensive Calculations**
   ```typescript
   const chartData = useMemo(
     () => transformData(rawData, activeFilters),
     [rawData, activeFilters]
   )
   ```

4. **Cache Query Results**
   - useDrillDown already caches detail queries
   - No need to implement caching

## Complete Minimal Example

```typescript
import { useDrillDown } from '@/components/reports/drill-down-handler'
import { DetailDrawer } from '@/components/reports/detail-drawer'
import { ChartRenderer } from '@/components/reports/chart-renderer'

function SimpleReport() {
  const { executeDrillDown, ...drawer } = useDrillDown({
    executeQuery: async (sql) => api.query(sql)
  })

  return (
    <>
      <ChartRenderer
        data={{ columns: ['status', 'count'], rows: [['pending', 10]] }}
        drillDownConfig={{ enabled: true, type: 'detail' }}
        onDrillDown={(ctx) => executeDrillDown({
          enabled: true,
          type: 'detail',
          detailQuery: `SELECT * FROM orders WHERE status = :clickedValue`
        }, ctx)}
      />
      <DetailDrawer {...drawer} />
    </>
  )
}
```

That's it! 🎉

## Next Steps

- Read full guide: `DRILL_DOWN_GUIDE.md`
- Check TypeScript types: `types/reports.ts`
- See examples: `DRILL_DOWN_IMPLEMENTATION.md`
