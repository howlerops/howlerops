# Dashboard Canvas - Usage Examples

This document provides practical examples of using the Dashboard Canvas component in various scenarios.

## Table of Contents

1. [Basic Setup](#basic-setup)
2. [Creating a Simple Dashboard](#creating-a-simple-dashboard)
3. [Applying Layout Templates](#applying-layout-templates)
4. [Custom Component Handlers](#custom-component-handlers)
5. [View Mode Toggle](#view-mode-toggle)
6. [Programmatic Layout Updates](#programmatic-layout-updates)
7. [Responsive Layouts](#responsive-layouts)
8. [Advanced Patterns](#advanced-patterns)

---

## Basic Setup

### Minimal Working Example

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import type { ReportComponent, ReportLayoutSlot } from '@/types/reports'

function MyDashboard() {
  const [components, setComponents] = useState<ReportComponent[]>([
    {
      id: 'metric-1',
      title: 'Total Sales',
      type: 'metric',
      query: {
        mode: 'sql',
        sql: 'SELECT SUM(amount) as total FROM sales',
        connectionId: 'db-1',
      },
    },
  ])

  const [layout, setLayout] = useState<ReportLayoutSlot[]>([
    { componentId: 'metric-1', x: 0, y: 0, w: 6, h: 4 },
  ])

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={setLayout}
    />
  )
}
```

---

## Creating a Simple Dashboard

### Multi-Component Dashboard

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import type { ReportComponent, ReportLayoutSlot } from '@/types/reports'

function SalesDashboard() {
  const components: ReportComponent[] = [
    {
      id: 'sales-metric',
      title: 'Total Sales',
      type: 'metric',
      size: { minW: 3, minH: 2, maxW: 6, maxH: 4 },
      query: {
        mode: 'sql',
        sql: 'SELECT SUM(amount) as total FROM sales WHERE date >= CURRENT_DATE - INTERVAL 30 DAY',
        connectionId: 'analytics-db',
      },
    },
    {
      id: 'orders-metric',
      title: 'Total Orders',
      type: 'metric',
      size: { minW: 3, minH: 2, maxW: 6, maxH: 4 },
      query: {
        mode: 'sql',
        sql: 'SELECT COUNT(*) as count FROM orders WHERE created_at >= CURRENT_DATE - INTERVAL 30 DAY',
        connectionId: 'analytics-db',
      },
    },
    {
      id: 'sales-chart',
      title: 'Sales Trend',
      type: 'chart',
      size: { minW: 6, minH: 4, maxW: 12, maxH: 8 },
      query: {
        mode: 'sql',
        sql: 'SELECT DATE(created_at) as date, SUM(amount) as sales FROM orders GROUP BY DATE(created_at) ORDER BY date',
        connectionId: 'analytics-db',
      },
      chart: {
        variant: 'line',
        xField: 'date',
        yField: 'sales',
      },
    },
    {
      id: 'top-products',
      title: 'Top Products',
      type: 'table',
      size: { minW: 6, minH: 4, maxW: 12, maxH: 8 },
      query: {
        mode: 'sql',
        sql: 'SELECT product_name, SUM(quantity) as sold FROM order_items GROUP BY product_name ORDER BY sold DESC LIMIT 10',
        connectionId: 'analytics-db',
      },
    },
  ]

  const layout: ReportLayoutSlot[] = [
    // Top row: KPIs
    { componentId: 'sales-metric', x: 0, y: 0, w: 6, h: 3 },
    { componentId: 'orders-metric', x: 6, y: 0, w: 6, h: 3 },
    // Bottom row: Chart and table
    { componentId: 'sales-chart', x: 0, y: 3, w: 6, h: 5 },
    { componentId: 'top-products', x: 6, y: 3, w: 6, h: 5 },
  ]

  const [currentLayout, setCurrentLayout] = useState(layout)

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Sales Dashboard</h1>
      <DashboardCanvas
        components={components}
        layout={currentLayout}
        onLayoutChange={setCurrentLayout}
        editMode={true}
      />
    </div>
  )
}
```

---

## Applying Layout Templates

### Using Built-in Templates

```typescript
import { useState } from 'react'
import { DashboardCanvas, LAYOUT_TEMPLATES, LayoutTemplate } from '@/components/reports/dashboard-canvas'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

function DashboardWithTemplates() {
  const [components, setComponents] = useState<ReportComponent[]>([
    // ... your components
  ])

  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])

  const applyTemplate = (templateName: LayoutTemplate) => {
    const template = LAYOUT_TEMPLATES[templateName]
    const componentIds = components.map((c) => c.id)
    const newLayout = template.getLayout(componentIds)
    setLayout(newLayout)
  }

  return (
    <div className="space-y-4">
      {/* Template Selector */}
      <div className="flex items-center gap-4">
        <label className="font-medium">Quick Layout:</label>
        <Select onValueChange={(value) => applyTemplate(value as LayoutTemplate)}>
          <SelectTrigger className="w-64">
            <SelectValue placeholder="Choose a template" />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(LAYOUT_TEMPLATES).map(([key, template]) => (
              <SelectItem key={key} value={key}>
                {template.name} - {template.description}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <DashboardCanvas
        components={components}
        layout={layout}
        onLayoutChange={setLayout}
      />
    </div>
  )
}
```

---

## Custom Component Handlers

### Full Event Handling

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import { useToast } from '@/hooks/use-toast'

function InteractiveDashboard() {
  const [components, setComponents] = useState<ReportComponent[]>([])
  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])
  const [selectedComponentId, setSelectedComponentId] = useState<string | null>(null)
  const { toast } = useToast()

  const handleComponentClick = (componentId: string) => {
    setSelectedComponentId(componentId)
    console.log('Component clicked:', componentId)
  }

  const handleComponentEdit = (componentId: string) => {
    const component = components.find((c) => c.id === componentId)
    if (component) {
      // Open edit modal/sidebar
      openComponentEditor(component)
    }
  }

  const handleComponentRun = async (componentId: string) => {
    const component = components.find((c) => c.id === componentId)
    if (!component) return

    try {
      toast({ title: 'Running component...', duration: 2000 })
      await executeComponent(component)
      toast({ title: 'Component executed successfully', duration: 2000 })
    } catch (error) {
      toast({
        title: 'Execution failed',
        description: error.message,
        variant: 'destructive',
      })
    }
  }

  const handleComponentDelete = (componentId: string) => {
    setComponents((prev) => prev.filter((c) => c.id !== componentId))
    setLayout((prev) => prev.filter((l) => l.componentId !== componentId))
    toast({ title: 'Component deleted', variant: 'destructive' })
  }

  const handleAddComponent = (type: ReportComponentType) => {
    const newComponent: ReportComponent = {
      id: crypto.randomUUID(),
      title: `New ${type}`,
      type,
      query: {
        mode: 'sql',
        sql: 'SELECT 1',
        connectionId: '',
      },
    }

    setComponents((prev) => [...prev, newComponent])

    // Add to layout at bottom
    const maxY = Math.max(...layout.map((l) => l.y + l.h), 0)
    setLayout((prev) => [
      ...prev,
      { componentId: newComponent.id, x: 0, y: maxY, w: 6, h: 4 },
    ])

    toast({ title: `${type} component added`, duration: 2000 })
  }

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={setLayout}
      onComponentClick={handleComponentClick}
      onComponentEdit={handleComponentEdit}
      onComponentRun={handleComponentRun}
      onComponentDelete={handleComponentDelete}
      onAddComponent={handleAddComponent}
      editMode={true}
    />
  )
}
```

---

## View Mode Toggle

### Edit/View Mode with Permissions

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import { Button } from '@/components/ui/button'
import { Edit, Eye } from 'lucide-react'

function DashboardWithPermissions() {
  const [components, setComponents] = useState<ReportComponent[]>([])
  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])
  const [editMode, setEditMode] = useState(false)

  // Check user permissions
  const userCanEdit = useUserPermissions('dashboard.edit')

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1>Dashboard</h1>

        {userCanEdit && (
          <Button
            variant="outline"
            onClick={() => setEditMode(!editMode)}
          >
            {editMode ? (
              <>
                <Eye className="mr-2 h-4 w-4" /> View Mode
              </>
            ) : (
              <>
                <Edit className="mr-2 h-4 w-4" /> Edit Mode
              </>
            )}
          </Button>
        )}
      </div>

      <DashboardCanvas
        components={components}
        layout={layout}
        onLayoutChange={setLayout}
        editMode={editMode}
        // Only provide edit handlers if user has permission
        onComponentEdit={userCanEdit ? handleEdit : undefined}
        onComponentDelete={userCanEdit ? handleDelete : undefined}
        onAddComponent={userCanEdit ? handleAdd : undefined}
      />
    </div>
  )
}
```

---

## Programmatic Layout Updates

### Rearranging Components Programmatically

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import { Button } from '@/components/ui/button'

function ProgrammaticLayout() {
  const [components, setComponents] = useState<ReportComponent[]>([
    // ... components
  ])

  const [layout, setLayout] = useState<ReportLayoutSlot[]>([
    { componentId: 'metric-1', x: 0, y: 0, w: 6, h: 3 },
    { componentId: 'chart-1', x: 6, y: 0, w: 6, h: 3 },
  ])

  // Swap two components
  const swapComponents = () => {
    setLayout((prev) => {
      const newLayout = [...prev]
      const idx1 = newLayout.findIndex((l) => l.componentId === 'metric-1')
      const idx2 = newLayout.findIndex((l) => l.componentId === 'chart-1')

      if (idx1 !== -1 && idx2 !== -1) {
        // Swap positions
        const temp = { x: newLayout[idx1].x, y: newLayout[idx1].y }
        newLayout[idx1].x = newLayout[idx2].x
        newLayout[idx1].y = newLayout[idx2].y
        newLayout[idx2].x = temp.x
        newLayout[idx2].y = temp.y
      }

      return newLayout
    })
  }

  // Stack all components vertically
  const stackVertically = () => {
    setLayout((prev) => {
      return prev.map((slot, index) => ({
        ...slot,
        x: 0,
        y: index * 4, // 4 rows per component
        w: 12, // Full width
        h: 4,
      }))
    })
  }

  // Arrange in grid (2 columns)
  const arrangeGrid = () => {
    setLayout((prev) => {
      return prev.map((slot, index) => ({
        ...slot,
        x: (index % 2) * 6, // 0 or 6
        y: Math.floor(index / 2) * 4, // Row index
        w: 6,
        h: 4,
      }))
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button onClick={swapComponents}>Swap Components</Button>
        <Button onClick={stackVertically}>Stack Vertically</Button>
        <Button onClick={arrangeGrid}>2-Column Grid</Button>
      </div>

      <DashboardCanvas
        components={components}
        layout={layout}
        onLayoutChange={setLayout}
      />
    </div>
  )
}
```

---

## Responsive Layouts

### Different Layouts per Breakpoint

```typescript
import { useState } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'

function ResponsiveDashboard() {
  const [components, setComponents] = useState<ReportComponent[]>([
    // ... components
  ])

  // Desktop layout: 4-column grid
  const desktopLayout: ReportLayoutSlot[] = [
    { componentId: 'metric-1', x: 0, y: 0, w: 3, h: 3 },
    { componentId: 'metric-2', x: 3, y: 0, w: 3, h: 3 },
    { componentId: 'metric-3', x: 6, y: 0, w: 3, h: 3 },
    { componentId: 'metric-4', x: 9, y: 0, w: 3, h: 3 },
    { componentId: 'chart-1', x: 0, y: 3, w: 6, h: 5 },
    { componentId: 'table-1', x: 6, y: 3, w: 6, h: 5 },
  ]

  const [layout, setLayout] = useState(desktopLayout)

  // You can also store multiple layouts for different breakpoints
  const [responsiveLayouts, setResponsiveLayouts] = useState({
    lg: desktopLayout,
    md: [
      // Tablet: 2 columns
      { componentId: 'metric-1', x: 0, y: 0, w: 5, h: 3 },
      { componentId: 'metric-2', x: 5, y: 0, w: 5, h: 3 },
      { componentId: 'metric-3', x: 0, y: 3, w: 5, h: 3 },
      { componentId: 'metric-4', x: 5, y: 3, w: 5, h: 3 },
      { componentId: 'chart-1', x: 0, y: 6, w: 10, h: 5 },
      { componentId: 'table-1', x: 0, y: 11, w: 10, h: 5 },
    ],
    sm: [
      // Mobile: 1 column
      { componentId: 'metric-1', x: 0, y: 0, w: 6, h: 3 },
      { componentId: 'metric-2', x: 0, y: 3, w: 6, h: 3 },
      { componentId: 'metric-3', x: 0, y: 6, w: 6, h: 3 },
      { componentId: 'metric-4', x: 0, y: 9, w: 6, h: 3 },
      { componentId: 'chart-1', x: 0, y: 12, w: 6, h: 5 },
      { componentId: 'table-1', x: 0, y: 17, w: 6, h: 5 },
    ],
  })

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={setLayout}
    />
  )
}
```

---

## Advanced Patterns

### Persistent Layout with LocalStorage

```typescript
import { useState, useEffect } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'

function PersistentDashboard({ dashboardId }: { dashboardId: string }) {
  const [components, setComponents] = useState<ReportComponent[]>([])
  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])

  // Load layout from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem(`dashboard-layout-${dashboardId}`)
    if (saved) {
      try {
        const parsed = JSON.parse(saved)
        setLayout(parsed)
      } catch (error) {
        console.error('Failed to load saved layout:', error)
      }
    }
  }, [dashboardId])

  // Save layout to localStorage when it changes
  const handleLayoutChange = (newLayout: ReportLayoutSlot[]) => {
    setLayout(newLayout)
    localStorage.setItem(`dashboard-layout-${dashboardId}`, JSON.stringify(newLayout))
  }

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={handleLayoutChange}
    />
  )
}
```

### Dashboard with Real-time Updates

```typescript
import { useState, useEffect } from 'react'
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import type { ReportRunResult } from '@/types/reports'

function RealTimeDashboard() {
  const [components, setComponents] = useState<ReportComponent[]>([])
  const [layout, setLayout] = useState<ReportLayoutSlot[]>([])
  const [results, setResults] = useState<ReportRunResult | undefined>()

  // Auto-refresh every 30 seconds
  useEffect(() => {
    const refreshData = async () => {
      try {
        const newResults = await fetchDashboardData(components)
        setResults(newResults)
      } catch (error) {
        console.error('Failed to refresh:', error)
      }
    }

    // Initial fetch
    refreshData()

    // Set up interval
    const interval = setInterval(refreshData, 30000)

    return () => clearInterval(interval)
  }, [components])

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1>Real-Time Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Last updated: {results?.completedAt.toLocaleTimeString()}
        </p>
      </div>

      <DashboardCanvas
        components={components}
        layout={layout}
        results={results}
        onLayoutChange={setLayout}
        editMode={false}
      />
    </div>
  )
}
```

### Custom Component Preview

```typescript
import { DashboardCanvas } from '@/components/reports/dashboard-canvas'

// Extend the ComponentPreview to support custom visualization
function DashboardWithCustomPreviews() {
  const [components, setComponents] = useState<ReportComponent[]>([
    {
      id: 'custom-1',
      title: 'Custom Viz',
      type: 'chart',
      options: {
        customRenderer: 'heatmap', // Custom type
      },
    },
  ])

  // ... rest of implementation

  return (
    <DashboardCanvas
      components={components}
      layout={layout}
      onLayoutChange={setLayout}
      // You can wrap DashboardCanvas with a custom provider
      // that handles custom visualization types
    />
  )
}
```

---

## Tips and Best Practices

### 1. Component ID Generation

Always use stable IDs for components:

```typescript
// Good: Stable UUID
const newComponent = {
  id: crypto.randomUUID(),
  // ...
}

// Bad: Using index or timestamp
const newComponent = {
  id: `component-${Date.now()}`, // Could cause collisions
  // ...
}
```

### 2. Layout Validation

Validate layouts before applying:

```typescript
function isValidLayout(layout: ReportLayoutSlot[]): boolean {
  // Check for duplicates
  const ids = layout.map((l) => l.componentId)
  if (new Set(ids).size !== ids.length) return false

  // Check bounds
  return layout.every((slot) => {
    return (
      slot.x >= 0 &&
      slot.y >= 0 &&
      slot.w > 0 &&
      slot.h > 0 &&
      slot.x + slot.w <= 12 // 12-column grid
    )
  })
}
```

### 3. Performance with Large Dashboards

For dashboards with 20+ components:

```typescript
import { useMemo } from 'react'

function LargeDashboard() {
  // Memoize expensive computations
  const componentMap = useMemo(() => {
    return new Map(components.map((c) => [c.id, c]))
  }, [components])

  const resultMap = useMemo(() => {
    return new Map(results?.results.map((r) => [r.componentId, r]))
  }, [results])

  // ... rest of implementation
}
```

### 4. Error Boundaries

Wrap in error boundary for resilience:

```typescript
import { ErrorBoundary } from 'react-error-boundary'

function ResilientDashboard() {
  return (
    <ErrorBoundary
      fallback={
        <div className="p-12 text-center">
          <p>Dashboard failed to load</p>
          <Button onClick={() => window.location.reload()}>Reload</Button>
        </div>
      }
    >
      <DashboardCanvas
        components={components}
        layout={layout}
        onLayoutChange={setLayout}
      />
    </ErrorBoundary>
  )
}
```

---

## Need More Help?

- Check the [main documentation](./README-DASHBOARD-CANVAS.md)
- See [component types reference](/docs/reports/component-types.md)
- Review [react-grid-layout docs](https://github.com/react-grid-layout/react-grid-layout)
