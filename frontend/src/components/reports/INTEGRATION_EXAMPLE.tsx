/**
 * Example: How to Integrate Drill-Down into ReportBuilder
 *
 * This file shows how to add drill-down configuration UI to the component editor
 * and integrate the drill-down system into the report rendering flow.
 *
 * NOTE: This is an example/template, not production code.
 */

import { ExternalLink, Filter, Globe, List } from 'lucide-react'
import { useState } from 'react'

import { CrossFilterBar } from '@/components/reports/cross-filter-bar'
import { DetailDrawer } from '@/components/reports/detail-drawer'
import { DrillDownBreadcrumbs } from '@/components/reports/drill-down-breadcrumbs'
import { useDrillDown } from '@/components/reports/drill-down-handler'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import type { DrillDownConfig, ReportComponent } from '@/types/reports'

/**
 * PART 1: Add Drill-Down Configuration UI to Component Editor
 *
 * Add this section to your component editor panel in ReportBuilder
 */
export function DrillDownConfigEditor({
  component,
  allReports,
  onUpdate,
}: {
  component: ReportComponent
  allReports: Array<{ id: string; name: string }>
  onUpdate: (drillDown: DrillDownConfig) => void
}) {
  const drillDown = component.drillDown || { enabled: false, type: 'detail' }

  const updateDrillDown = (updates: Partial<DrillDownConfig>) => {
    onUpdate({ ...drillDown, ...updates })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Drill-Down Settings</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Enable/Disable Toggle */}
        <div className="flex items-center justify-between">
          <Label htmlFor="drill-down-enabled">Enable drill-down</Label>
          <Switch
            id="drill-down-enabled"
            checked={drillDown.enabled}
            onCheckedChange={(checked) => updateDrillDown({ enabled: checked })}
          />
        </div>

        {drillDown.enabled && (
          <>
            {/* Action Type Selector */}
            <div className="space-y-2">
              <Label>Action Type</Label>
              <Select
                value={drillDown.type}
                onValueChange={(type) => updateDrillDown({ type: type as any })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="detail">
                    <div className="flex items-center gap-2">
                      <List className="h-4 w-4" />
                      <div>
                        <p className="font-medium">Show Detail</p>
                        <p className="text-xs text-muted-foreground">Display underlying records</p>
                      </div>
                    </div>
                  </SelectItem>
                  <SelectItem value="related-report">
                    <div className="flex items-center gap-2">
                      <ExternalLink className="h-4 w-4" />
                      <div>
                        <p className="font-medium">Navigate to Report</p>
                        <p className="text-xs text-muted-foreground">Link to another report</p>
                      </div>
                    </div>
                  </SelectItem>
                  <SelectItem value="filter">
                    <div className="flex items-center gap-2">
                      <Filter className="h-4 w-4" />
                      <div>
                        <p className="font-medium">Apply Filter</p>
                        <p className="text-xs text-muted-foreground">Filter other components</p>
                      </div>
                    </div>
                  </SelectItem>
                  <SelectItem value="url">
                    <div className="flex items-center gap-2">
                      <Globe className="h-4 w-4" />
                      <div>
                        <p className="font-medium">Open URL</p>
                        <p className="text-xs text-muted-foreground">Link to external resource</p>
                      </div>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Detail Type: SQL Editor */}
            {drillDown.type === 'detail' && (
              <div className="space-y-2">
                <Label htmlFor="detail-query">Detail Query (SQL)</Label>
                <Textarea
                  id="detail-query"
                  className="font-mono text-sm"
                  rows={6}
                  value={drillDown.detailQuery || ''}
                  onChange={(e) => updateDrillDown({ detailQuery: e.target.value })}
                  placeholder="SELECT * FROM table WHERE field = :clickedValue"
                />
                <p className="text-xs text-muted-foreground">
                  Use <code className="px-1 py-0.5 bg-muted rounded">:clickedValue</code> to
                  reference the clicked data point
                </p>
              </div>
            )}

            {/* Filter Type: Field Selector */}
            {drillDown.type === 'filter' && (
              <div className="space-y-2">
                <Label htmlFor="filter-field">Filter Field</Label>
                <Input
                  id="filter-field"
                  value={drillDown.filterField || ''}
                  onChange={(e) => updateDrillDown({ filterField: e.target.value })}
                  placeholder="e.g., product_category"
                />
                <p className="text-xs text-muted-foreground">
                  Other components will be filtered by this field
                </p>
              </div>
            )}

            {/* Related Report Type: Report Selector */}
            {drillDown.type === 'related-report' && (
              <div className="space-y-2">
                <Label htmlFor="target-report">Target Report</Label>
                <Select
                  value={drillDown.target}
                  onValueChange={(target) => updateDrillDown({ target })}
                >
                  <SelectTrigger id="target-report">
                    <SelectValue placeholder="Select report..." />
                  </SelectTrigger>
                  <SelectContent>
                    {allReports.map((report) => (
                      <SelectItem key={report.id} value={report.id}>
                        {report.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* URL Type: Template Editor */}
            {drillDown.type === 'url' && (
              <div className="space-y-2">
                <Label htmlFor="url-template">URL Template</Label>
                <Input
                  id="url-template"
                  value={drillDown.target || ''}
                  onChange={(e) => updateDrillDown({ target: e.target.value })}
                  placeholder="https://example.com/detail?id={clickedValue}"
                />
                <p className="text-xs text-muted-foreground">
                  Use{' '}
                  <code className="px-1 py-0.5 bg-muted rounded">{'{clickedValue}'}</code> to
                  insert the clicked data
                </p>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * PART 2: Integrate Drill-Down into Report Rendering
 *
 * Add this to your main report rendering component
 */
export function DrillDownEnabledReport() {
  // Your existing report state
  const [report, setReport] = useState<any>(null)
  const [components, setComponents] = useState<ReportComponent[]>([])
  const [lastRun, setLastRun] = useState<any>(null)

  // Initialize drill-down hook
  const drillDown = useDrillDown({
    executeQuery: async (sql) => {
      // Execute detail query
      const response = await fetch('/api/reports/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sql }),
      })
      return response.json()
    },
    onFilterChange: (filters) => {
      // Re-run report with filters
      console.log('Filters changed:', filters)
      // refreshReport(filters)
    },
  })

  return (
    <div className="space-y-6">
      {/* Navigation Breadcrumbs */}
      {drillDown.history.length > 0 && (
        <DrillDownBreadcrumbs
          history={drillDown.history}
          onNavigate={(idx) => {
            if (idx === -1) {
              drillDown.goBack()
            } else {
              // Navigate to specific breadcrumb
              // Could implement going to specific history point
            }
          }}
        />
      )}

      {/* Active Filters Bar */}
      <CrossFilterBar
        activeFilters={drillDown.activeFilters}
        onClearFilter={drillDown.clearFilter}
        onClearAll={drillDown.clearAllFilters}
      />

      {/* Report Components */}
      <div className="grid gap-6">
        {components.map((component) => {
          const result = lastRun?.results?.find((r: any) => r.componentId === component.id)

          if (component.type === 'chart' && result?.columns && result?.rows) {
            return (
              <Card key={component.id}>
                <CardHeader>
                  <CardTitle>{component.title}</CardTitle>
                </CardHeader>
                <CardContent>
                  {/* Pass drill-down config and handler to chart */}
                  <ChartRenderer
                    data={{ columns: result.columns, rows: result.rows }}
                    chartConfig={component.chart}
                    drillDownConfig={component.drillDown}
                    onDrillDown={(context) => {
                      if (!component.drillDown) return

                      // Add component ID to context
                      const contextWithId = {
                        ...context,
                        componentId: component.id,
                      }

                      drillDown.executeDrillDown(component.drillDown, contextWithId)
                    }}
                  />
                </CardContent>
              </Card>
            )
          }

          return null
        })}
      </div>

      {/* Detail Drawer */}
      <DetailDrawer
        open={drillDown.detailDrawerOpen}
        onClose={drillDown.closeDetailDrawer}
        title="Details"
        filters={drillDown.activeFilters}
        loading={drillDown.detailLoading}
        data={drillDown.detailData}
        error={drillDown.detailError}
      />
    </div>
  )
}

/**
 * PART 3: Example ChartRenderer Usage (Already Implemented)
 *
 * ChartRenderer automatically supports drill-down when you pass:
 * - drillDownConfig: Configuration object
 * - onDrillDown: Click handler callback
 *
 * Example from above shows the pattern:
 */
import { ChartRenderer } from '@/components/reports/chart-renderer'

function ExampleChartWithDrillDown() {
  const { executeDrillDown } = useDrillDown({
    executeQuery: async (sql) => {
      /* ... */
      return null as any
    },
  })

  return (
    <ChartRenderer
      data={{
        columns: ['month', 'revenue'],
        rows: [
          ['Jan', 1000],
          ['Feb', 1500],
        ],
      }}
      chartConfig={{ variant: 'bar', xField: 'month', series: ['revenue'] }}
      drillDownConfig={{
        enabled: true,
        type: 'detail',
        detailQuery: 'SELECT * FROM transactions WHERE month = :clickedValue',
      }}
      onDrillDown={(context) => {
        executeDrillDown(
          {
            enabled: true,
            type: 'detail',
            detailQuery: 'SELECT * FROM transactions WHERE month = :clickedValue',
          },
          context
        )
      }}
    />
  )
}
