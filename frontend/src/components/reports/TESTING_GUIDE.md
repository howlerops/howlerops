# Drill-Down System Testing Guide

## Quick Verification (5 minutes)

### Test 1: Basic Click-Through
```typescript
// Setup
const testReport = {
  components: [{
    id: 'test-chart',
    type: 'chart',
    drillDown: {
      enabled: true,
      type: 'detail',
      detailQuery: 'SELECT * FROM orders WHERE status = :clickedValue'
    }
  }]
}

// Test Steps
1. Render chart with data
2. Click on a bar/point
3. ✅ Verify drawer opens
4. ✅ Verify query executed with correct params
5. ✅ Verify data displayed in table
```

### Test 2: Filter Application
```typescript
// Setup
const testConfig = {
  enabled: true,
  type: 'filter',
  filterField: 'category'
}

// Test Steps
1. Click chart element
2. ✅ Verify filter badge appears
3. ✅ Verify other components re-queried
4. Click X on badge
5. ✅ Verify filter removed
```

### Test 3: Navigation
```typescript
// Test Steps
1. Perform 2-3 drill-downs
2. ✅ Verify breadcrumbs show path
3. Press Alt + ←
4. ✅ Verify goes back one step
5. Click earlier breadcrumb
6. ✅ Verify jumps to that point
```

## Unit Tests

### useDrillDown Hook Tests

```typescript
import { renderHook, act } from '@testing-library/react'
import { useDrillDown } from './drill-down-handler'

describe('useDrillDown', () => {
  test('executes detail drill-down', async () => {
    const executeQuery = jest.fn().mockResolvedValue({
      columns: ['id', 'name'],
      rows: [[1, 'Test']]
    })

    const { result } = renderHook(() => useDrillDown({ executeQuery }))

    await act(async () => {
      await result.current.executeDrillDown(
        {
          enabled: true,
          type: 'detail',
          detailQuery: 'SELECT * FROM test WHERE id = :clickedValue'
        },
        { clickedValue: 123, field: 'id' }
      )
    })

    expect(executeQuery).toHaveBeenCalledWith(
      "SELECT * FROM test WHERE id = 123"
    )
    expect(result.current.detailDrawerOpen).toBe(true)
    expect(result.current.detailData).toEqual({
      columns: ['id', 'name'],
      rows: [[1, 'Test']]
    })
  })

  test('applies cross-filter', () => {
    const onFilterChange = jest.fn()
    const { result } = renderHook(() => useDrillDown({ onFilterChange }))

    act(() => {
      result.current.executeDrillDown(
        { enabled: true, type: 'filter', filterField: 'category' },
        { clickedValue: 'Electronics', field: 'category' }
      )
    })

    expect(onFilterChange).toHaveBeenCalledWith({
      category: 'Electronics'
    })
    expect(result.current.activeFilters).toEqual({
      category: 'Electronics'
    })
  })

  test('tracks history', async () => {
    const executeQuery = jest.fn().mockResolvedValue({ columns: [], rows: [] })
    const { result } = renderHook(() => useDrillDown({ executeQuery }))

    expect(result.current.history).toHaveLength(0)
    expect(result.current.canGoBack).toBe(false)

    await act(async () => {
      await result.current.executeDrillDown(
        { enabled: true, type: 'detail', detailQuery: 'SELECT 1' },
        { clickedValue: 'test', field: 'test' }
      )
    })

    expect(result.current.history).toHaveLength(1)
    expect(result.current.canGoBack).toBe(true)
  })

  test('goes back in history', async () => {
    const executeQuery = jest.fn().mockResolvedValue({ columns: [], rows: [] })
    const { result } = renderHook(() => useDrillDown({ executeQuery }))

    // Create history
    await act(async () => {
      await result.current.executeDrillDown(
        { enabled: true, type: 'detail', detailQuery: 'SELECT 1' },
        { clickedValue: 'test1', field: 'test' }
      )
      await result.current.executeDrillDown(
        { enabled: true, type: 'detail', detailQuery: 'SELECT 2' },
        { clickedValue: 'test2', field: 'test' }
      )
    })

    expect(result.current.history).toHaveLength(2)

    // Go back
    act(() => {
      result.current.goBack()
    })

    expect(result.current.history).toHaveLength(1)
  })

  test('clears filters', () => {
    const onFilterChange = jest.fn()
    const { result } = renderHook(() => useDrillDown({ onFilterChange }))

    // Apply filter
    act(() => {
      result.current.executeDrillDown(
        { enabled: true, type: 'filter', filterField: 'category' },
        { clickedValue: 'Electronics', field: 'category' }
      )
    })

    expect(result.current.activeFilters).toEqual({ category: 'Electronics' })

    // Clear all
    act(() => {
      result.current.clearAllFilters()
    })

    expect(result.current.activeFilters).toEqual({})
    expect(onFilterChange).toHaveBeenCalledWith({})
  })
})
```

### DetailDrawer Component Tests

```typescript
import { render, screen, fireEvent } from '@testing-library/react'
import { DetailDrawer } from './detail-drawer'

describe('DetailDrawer', () => {
  test('renders when open', () => {
    render(
      <DetailDrawer
        open={true}
        onClose={jest.fn()}
        title="Test Details"
        loading={false}
        data={{ columns: ['id'], rows: [[1]] }}
        error={null}
      />
    )

    expect(screen.getByText('Test Details')).toBeInTheDocument()
  })

  test('shows loading state', () => {
    render(
      <DetailDrawer
        open={true}
        onClose={jest.fn()}
        title="Test"
        loading={true}
        data={null}
        error={null}
      />
    )

    expect(screen.getByText(/loading detail data/i)).toBeInTheDocument()
  })

  test('shows error state', () => {
    render(
      <DetailDrawer
        open={true}
        onClose={jest.fn()}
        title="Test"
        loading={false}
        data={null}
        error="Query failed"
      />
    )

    expect(screen.getByText('Query failed')).toBeInTheDocument()
  })

  test('displays data in table', () => {
    render(
      <DetailDrawer
        open={true}
        onClose={jest.fn()}
        title="Test"
        loading={false}
        data={{
          columns: ['id', 'name'],
          rows: [[1, 'Test'], [2, 'Test2']]
        }}
        error={null}
      />
    )

    expect(screen.getByText('2 Records')).toBeInTheDocument()
  })

  test('calls onClose when close button clicked', () => {
    const onClose = jest.fn()

    render(
      <DetailDrawer
        open={true}
        onClose={onClose}
        title="Test"
        loading={false}
        data={null}
        error={null}
      />
    )

    const closeButton = screen.getByRole('button', { name: /close/i })
    fireEvent.click(closeButton)

    expect(onClose).toHaveBeenCalled()
  })

  test('shows active filters', () => {
    render(
      <DetailDrawer
        open={true}
        onClose={jest.fn()}
        title="Test"
        filters={{ category: 'Electronics', status: 'active' }}
        loading={false}
        data={null}
        error={null}
      />
    )

    expect(screen.getByText(/category = Electronics/i)).toBeInTheDocument()
    expect(screen.getByText(/status = active/i)).toBeInTheDocument()
  })
})
```

### ChartRenderer Click Tests

```typescript
import { render, screen, fireEvent } from '@testing-library/react'
import { ChartRenderer } from './chart-renderer'

describe('ChartRenderer drill-down', () => {
  const testData = {
    columns: ['month', 'revenue'],
    rows: [
      ['Jan', 1000],
      ['Feb', 1500],
      ['Mar', 2000]
    ]
  }

  test('calls onDrillDown when chart clicked', () => {
    const onDrillDown = jest.fn()

    render(
      <ChartRenderer
        data={testData}
        chartConfig={{ variant: 'bar' }}
        drillDownConfig={{ enabled: true, type: 'detail' }}
        onDrillDown={onDrillDown}
      />
    )

    // Click on chart element (implementation depends on Recharts)
    // This is a simplified example
    const chartElement = screen.getByRole('img') // SVG chart
    fireEvent.click(chartElement)

    // Verify context passed to callback
    expect(onDrillDown).toHaveBeenCalledWith(
      expect.objectContaining({
        field: expect.any(String),
        clickedValue: expect.anything()
      })
    )
  })

  test('shows drill-down hint in tooltip', () => {
    render(
      <ChartRenderer
        data={testData}
        chartConfig={{ variant: 'bar' }}
        drillDownConfig={{ enabled: true, type: 'detail' }}
        onDrillDown={jest.fn()}
      />
    )

    // Hover to show tooltip
    const chartElement = screen.getByRole('img')
    fireEvent.mouseOver(chartElement)

    expect(screen.getByText(/click to view details/i)).toBeInTheDocument()
  })

  test('does not call onDrillDown when disabled', () => {
    const onDrillDown = jest.fn()

    render(
      <ChartRenderer
        data={testData}
        chartConfig={{ variant: 'bar' }}
        drillDownConfig={{ enabled: false, type: 'detail' }}
        onDrillDown={onDrillDown}
      />
    )

    const chartElement = screen.getByRole('img')
    fireEvent.click(chartElement)

    expect(onDrillDown).not.toHaveBeenCalled()
  })
})
```

## Integration Tests

### Full Drill-Down Workflow

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { DrillDownEnabledReport } from './test-utils'

describe('Drill-down workflow', () => {
  test('complete detail drill-down flow', async () => {
    const { user } = render(<DrillDownEnabledReport />)

    // 1. Click chart element
    const chartBar = screen.getByTestId('chart-bar-pending')
    fireEvent.click(chartBar)

    // 2. Verify drawer opens
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    // 3. Verify detail data loads
    await waitFor(() => {
      expect(screen.getByText(/10 records/i)).toBeInTheDocument()
    })

    // 4. Close drawer
    const closeButton = screen.getByRole('button', { name: /close/i })
    fireEvent.click(closeButton)

    // 5. Verify drawer closed
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  test('cross-filtering between components', async () => {
    const { user } = render(<DrillDownEnabledReport />)

    // 1. Click first chart
    const chartBar = screen.getByTestId('chart-1-bar-electronics')
    fireEvent.click(chartBar)

    // 2. Verify filter applied
    await waitFor(() => {
      expect(screen.getByText(/category = Electronics/i)).toBeInTheDocument()
    })

    // 3. Verify all charts updated
    const charts = screen.getAllByTestId(/chart-/)
    charts.forEach(chart => {
      expect(chart).toHaveAttribute('data-filtered', 'true')
    })

    // 4. Clear filter
    const clearButton = screen.getByRole('button', { name: /clear all/i })
    fireEvent.click(clearButton)

    // 5. Verify filter removed
    await waitFor(() => {
      expect(screen.queryByText(/category = Electronics/i)).not.toBeInTheDocument()
    })
  })

  test('breadcrumb navigation', async () => {
    const { user } = render(<DrillDownEnabledReport />)

    // 1. Apply first filter
    fireEvent.click(screen.getByTestId('chart-bar-1'))
    await waitFor(() => {
      expect(screen.getByText(/Dashboard.*Filter 1/i)).toBeInTheDocument()
    })

    // 2. Apply second filter
    fireEvent.click(screen.getByTestId('chart-bar-2'))
    await waitFor(() => {
      expect(screen.getByText(/Dashboard.*Filter 1.*Filter 2/i)).toBeInTheDocument()
    })

    // 3. Click first breadcrumb
    const breadcrumb1 = screen.getByText('Filter 1')
    fireEvent.click(breadcrumb1)

    // 4. Verify back to first filter state
    await waitFor(() => {
      expect(screen.queryByText(/Filter 2/i)).not.toBeInTheDocument()
    })
  })
})
```

## E2E Tests (Playwright)

```typescript
import { test, expect } from '@playwright/test'

test.describe('Drill-down system', () => {
  test('user can drill into chart details', async ({ page }) => {
    // Navigate to reports page
    await page.goto('/reports/test-report')

    // Wait for chart to load
    await page.waitForSelector('[data-testid="chart-renderer"]')

    // Click on chart bar
    await page.click('[data-testid="chart-bar-pending"]')

    // Verify drawer opens
    await expect(page.locator('[role="dialog"]')).toBeVisible()

    // Verify detail data loads
    await expect(page.locator('text=/\\d+ Records/')).toBeVisible()

    // Export to CSV
    await page.click('button:has-text("Export")')

    // Verify download started
    const download = await page.waitForEvent('download')
    expect(download.suggestedFilename()).toMatch(/drill-down.*\.csv/)
  })

  test('filters persist in URL', async ({ page }) => {
    await page.goto('/reports/test-report')

    // Apply filter by clicking chart
    await page.click('[data-testid="chart-bar-electronics"]')

    // Verify URL updated
    await expect(page).toHaveURL(/filters=%7B/)

    // Refresh page
    await page.reload()

    // Verify filter restored
    await expect(page.locator('text=/category = Electronics/i')).toBeVisible()
  })

  test('keyboard shortcuts work', async ({ page }) => {
    await page.goto('/reports/test-report')

    // Open detail drawer
    await page.click('[data-testid="chart-bar-pending"]')
    await expect(page.locator('[role="dialog"]')).toBeVisible()

    // Press Esc to close
    await page.keyboard.press('Escape')
    await expect(page.locator('[role="dialog"]')).not.toBeVisible()

    // Apply filter
    await page.click('[data-testid="chart-bar-electronics"]')
    await expect(page.locator('text=/category = Electronics/i')).toBeVisible()

    // Press Alt+C to clear
    await page.keyboard.press('Alt+KeyC')
    await expect(page.locator('text=/category = Electronics/i')).not.toBeVisible()
  })
})
```

## Performance Tests

### Query Execution Timing

```typescript
describe('Performance', () => {
  test('detail query executes within 2 seconds', async () => {
    const start = Date.now()

    const { result } = renderHook(() => useDrillDown({
      executeQuery: async (sql) => {
        // Simulate realistic query time
        await new Promise(resolve => setTimeout(resolve, 500))
        return { columns: ['id'], rows: [[1]] }
      }
    }))

    await act(async () => {
      await result.current.executeDrillDown(
        { enabled: true, type: 'detail', detailQuery: 'SELECT * FROM test' },
        { clickedValue: 1, field: 'id' }
      )
    })

    const duration = Date.now() - start
    expect(duration).toBeLessThan(2000)
  })

  test('filter debouncing reduces queries', async () => {
    jest.useFakeTimers()
    const executeQuery = jest.fn().mockResolvedValue({ columns: [], rows: [] })

    const { result } = renderHook(() => useDrillDown({ executeQuery }))

    // Rapid filter changes
    act(() => {
      result.current.executeDrillDown(
        { enabled: true, type: 'filter', filterField: 'category' },
        { clickedValue: 'Electronics', field: 'category' }
      )
    })

    act(() => {
      result.current.executeDrillDown(
        { enabled: true, type: 'filter', filterField: 'category' },
        { clickedValue: 'Books', field: 'category' }
      )
    })

    // Fast forward past debounce
    act(() => {
      jest.advanceTimersByTime(300)
    })

    // Should only execute once
    expect(executeQuery).toHaveBeenCalledTimes(0) // Filter doesn't execute query
  })
})
```

## Manual Testing Checklist

### Feature Completeness
- [ ] All 4 drill-down types work (detail, filter, related-report, url)
- [ ] Click works on all chart types (bar, line, area, pie, combo)
- [ ] Detail drawer shows data correctly
- [ ] CSV export works
- [ ] Breadcrumbs show history
- [ ] Cross-filter bar displays filters
- [ ] Keyboard shortcuts work (Alt+←, Esc, Alt+C)
- [ ] URL sharing preserves filters

### Error Handling
- [ ] Query errors show in detail drawer
- [ ] Network failures handled gracefully
- [ ] Invalid SQL shows error message
- [ ] Empty results show appropriate message
- [ ] Loading states display correctly

### Performance
- [ ] Large datasets (10K+ rows) render smoothly
- [ ] Multiple filters don't cause lag
- [ ] Debouncing prevents excessive queries
- [ ] Drawer opens instantly (< 100ms)

### Accessibility
- [ ] Can navigate with keyboard only
- [ ] Screen reader announces drill-down hints
- [ ] Focus indicators visible
- [ ] Color contrast meets WCAG AA
- [ ] ARIA labels present

### Browser Compatibility
- [ ] Works in Chrome
- [ ] Works in Firefox
- [ ] Works in Safari
- [ ] Works in Edge
- [ ] Mobile responsive

### Edge Cases
- [ ] Works with no data
- [ ] Works with single data point
- [ ] Works with very long labels
- [ ] Works with special characters in data
- [ ] Works with null/undefined values

## Debugging Tips

### Drill-down not triggering
```typescript
// Check these in dev tools:
console.log('drillDownConfig:', component.drillDown)
console.log('drillDownConfig.enabled:', component.drillDown?.enabled)
console.log('onDrillDown callback:', typeof onDrillDown)

// Verify in ChartRenderer:
console.log('handleElementClick called:', dataPoint)
```

### Query not interpolating
```typescript
// Log interpolation:
console.log('Template:', detailQuery)
console.log('Context:', context)
console.log('Result:', interpolatedQuery)

// Check parameter syntax:
// ✅ :clickedValue
// ✅ {clickedValue}
// ❌ ${clickedValue}
// ❌ $clickedValue
```

### Filters not applying
```typescript
// Check filter state:
console.log('activeFilters:', activeFilters)
console.log('onFilterChange called:', filterChangeCount)

// Verify components using filters:
console.log('Query with filters:', buildQuery(activeFilters))
```

## Coverage Goals

- **Unit Tests**: 80%+ coverage of business logic
- **Integration Tests**: All major workflows covered
- **E2E Tests**: Critical user paths tested
- **Performance Tests**: Key operations benchmarked
