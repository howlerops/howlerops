import { MousePointerClick } from 'lucide-react'
import { useMemo } from 'react'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { Tooltip as UITooltip, TooltipContent, TooltipTrigger, TooltipProvider } from '@/components/ui/tooltip'
import type { DrillDownConfig, DrillDownContext, ReportChartSettings } from '@/types/reports'

interface ChartRendererProps {
  data: {
    columns: string[]
    rows: unknown[][]
  }
  chartConfig?: ReportChartSettings
  drillDownConfig?: DrillDownConfig
  title?: string
  height?: number
  onDrillDown?: (context: DrillDownContext) => void
}

interface ChartDataPoint {
  [key: string]: string | number | null
}

// Color palette for multiple series (consistent with design system)
const CHART_COLORS = [
  'hsl(var(--primary))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
  '#8b5cf6', // purple
  '#ec4899', // pink
  '#f97316', // orange
  '#14b8a6', // teal
  '#6366f1', // indigo
]

// Pie chart colors
const PIE_COLORS = [
  'hsl(var(--primary))',
  '#8b5cf6',
  '#ec4899',
  '#f97316',
  '#14b8a6',
  '#6366f1',
  '#06b6d4',
  '#84cc16',
]

/**
 * Transform tabular data into chart-friendly format
 * Input: { columns: ["date", "revenue", "profit"], rows: [["2024-01", 1000, 200], ...] }
 * Output: [{ date: "2024-01", revenue: 1000, profit: 200 }, ...]
 */
function transformData(columns: string[], rows: unknown[][]): ChartDataPoint[] {
  return rows.map((row) => {
    const point: ChartDataPoint = {}
    columns.forEach((col, idx) => {
      const value = row[idx]
      if (value === null || value === undefined) {
        point[col] = null
      } else if (typeof value === 'number') {
        point[col] = value
      } else {
        point[col] = String(value)
      }
    })
    return point
  })
}

/**
 * Auto-detect axis fields from data
 */
function detectFields(columns: string[], data: ChartDataPoint[]) {
  if (columns.length === 0) {
    return { xField: '', yFields: [] }
  }

  // First non-numeric column becomes X-axis (typically date/category)
  const firstRow = data[0] || {}
  const xField =
    columns.find((col) => {
      const val = firstRow[col]
      return typeof val === 'string' || val === null
    }) || columns[0]

  // All numeric columns become Y-axis series
  const yFields = columns.filter((col) => {
    if (col === xField) return false
    const val = firstRow[col]
    return typeof val === 'number'
  })

  return { xField, yFields }
}

/**
 * Subsample data if too many points (performance optimization)
 */
function subsampleData(data: ChartDataPoint[], maxPoints: number = 1000): ChartDataPoint[] {
  if (data.length <= maxPoints) return data

  const step = Math.ceil(data.length / maxPoints)
  return data.filter((_, idx) => idx % step === 0)
}

/**
 * Format number for display in tooltips
 */
function formatNumber(value: number): string {
  if (Math.abs(value) >= 1e9) {
    return `${(value / 1e9).toFixed(2)}B`
  }
  if (Math.abs(value) >= 1e6) {
    return `${(value / 1e6).toFixed(2)}M`
  }
  if (Math.abs(value) >= 1e3) {
    return `${(value / 1e3).toFixed(2)}K`
  }
  return value.toFixed(2)
}

/**
 * Custom tooltip with better formatting and drill-down hints
 */
function CustomTooltip({ active, payload, label, drillDownEnabled }: any) {
  if (!active || !payload?.length) return null

  return (
    <div className="rounded-lg border bg-background p-3 shadow-lg">
      <p className="mb-2 font-medium">{label}</p>
      {payload.map((entry: any, index: number) => (
        <div key={index} className="flex items-center gap-2 text-sm">
          <div className="h-3 w-3 rounded" style={{ backgroundColor: entry.color }} />
          <span className="text-muted-foreground">{entry.name}:</span>
          <span className="font-medium">{typeof entry.value === 'number' ? formatNumber(entry.value) : entry.value}</span>
        </div>
      ))}
      {drillDownEnabled && (
        <div className="pt-2 mt-2 border-t">
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <MousePointerClick className="h-3 w-3" />
            Click to view details
          </p>
        </div>
      )}
    </div>
  )
}

/**
 * Main chart renderer component
 */
export function ChartRenderer({
  data,
  chartConfig,
  drillDownConfig,
  title,
  height = 400,
  onDrillDown,
}: ChartRendererProps) {
  const chartData = useMemo(() => {
    const transformed = transformData(data.columns, data.rows)
    return subsampleData(transformed)
  }, [data.columns, data.rows])

  const { xField: detectedX, yFields: detectedY } = useMemo(
    () => detectFields(data.columns, chartData),
    [data.columns, chartData]
  )

  // Use config or fall back to auto-detected fields
  const variant = chartConfig?.variant || 'line'
  const xField = chartConfig?.xField || detectedX
  const yFields = chartConfig?.series?.length ? chartConfig.series : detectedY

  // Drill-down enabled flag
  const drillDownEnabled = drillDownConfig?.enabled && !!onDrillDown

  // Handle click on chart element
  const handleElementClick = (dataPoint: ChartDataPoint) => {
    if (!drillDownEnabled || !onDrillDown) return

    const context: DrillDownContext = {
      clickedValue: dataPoint[xField],
      field: xField,
      filters: dataPoint,
      additionalData: dataPoint,
    }

    onDrillDown(context)
  }

  // Handle empty data
  if (chartData.length === 0) {
    return (
      <div className="flex h-full items-center justify-center rounded-md border border-dashed">
        <div className="text-center">
          <p className="text-sm font-medium text-muted-foreground">No data to display</p>
          <p className="text-xs text-muted-foreground">Run the query to see results</p>
        </div>
      </div>
    )
  }

  // Handle missing fields
  if (!xField || yFields.length === 0) {
    return (
      <div className="flex h-full items-center justify-center rounded-md border border-dashed">
        <div className="text-center">
          <p className="text-sm font-medium text-muted-foreground">Unable to render chart</p>
          <p className="text-xs text-muted-foreground">
            Configure X-axis ({!xField ? 'missing' : 'ok'}) and Y-axis fields ({yFields.length === 0 ? 'missing' : 'ok'})
          </p>
        </div>
      </div>
    )
  }

  const commonProps = {
    data: chartData,
    margin: { top: 10, right: 30, left: 0, bottom: 0 },
  }

  return (
    <div className="space-y-2">
      {title && <h3 className="text-sm font-medium">{title}</h3>}
      <ResponsiveContainer width="100%" height={height}>
        {variant === 'pie' ? (
          <PieChart>
            <Pie
              data={chartData}
              dataKey={yFields[0]}
              nameKey={xField}
              cx="50%"
              cy="50%"
              outerRadius={Math.min(height / 2.5, 120)}
              label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(0)}%`}
              onClick={drillDownEnabled ? handleElementClick : undefined}
              cursor={drillDownEnabled ? 'pointer' : 'default'}
            >
              {chartData.map((_, index) => (
                <Cell key={`cell-${index}`} fill={PIE_COLORS[index % PIE_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip content={<CustomTooltip drillDownEnabled={drillDownEnabled} />} />
            <Legend />
          </PieChart>
        ) : variant === 'bar' ? (
          <BarChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey={xField} className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} />
            <YAxis className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} tickFormatter={formatNumber} />
            <Tooltip content={<CustomTooltip drillDownEnabled={drillDownEnabled} />} />
            {yFields.length > 1 && <Legend />}
            {yFields.map((field, idx) => (
              <Bar
                key={field}
                dataKey={field}
                fill={CHART_COLORS[idx % CHART_COLORS.length]}
                radius={[4, 4, 0, 0]}
                onClick={drillDownEnabled ? handleElementClick : undefined}
                cursor={drillDownEnabled ? 'pointer' : 'default'}
                activeBar={drillDownEnabled ? { fill: 'hsl(var(--primary))' } : undefined}
              />
            ))}
          </BarChart>
        ) : variant === 'area' ? (
          <AreaChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey={xField} className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} />
            <YAxis className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} tickFormatter={formatNumber} />
            <Tooltip content={<CustomTooltip drillDownEnabled={drillDownEnabled} />} />
            {yFields.length > 1 && <Legend />}
            {yFields.map((field, idx) => (
              <Area
                key={field}
                type="monotone"
                dataKey={field}
                stroke={CHART_COLORS[idx % CHART_COLORS.length]}
                fill={CHART_COLORS[idx % CHART_COLORS.length]}
                fillOpacity={0.6}
                onClick={drillDownEnabled ? handleElementClick : undefined}
                cursor={drillDownEnabled ? 'pointer' : 'default'}
              />
            ))}
          </AreaChart>
        ) : variant === 'combo' ? (
          <AreaChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey={xField} className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} />
            <YAxis className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} tickFormatter={formatNumber} />
            <Tooltip content={<CustomTooltip drillDownEnabled={drillDownEnabled} />} />
            {yFields.length > 1 && <Legend />}
            {yFields.map((field, idx) =>
              idx === 0 ? (
                <Area
                  key={field}
                  type="monotone"
                  dataKey={field}
                  stroke={CHART_COLORS[idx]}
                  fill={CHART_COLORS[idx]}
                  fillOpacity={0.6}
                  onClick={drillDownEnabled ? handleElementClick : undefined}
                  cursor={drillDownEnabled ? 'pointer' : 'default'}
                />
              ) : (
                <Line
                  key={field}
                  type="monotone"
                  dataKey={field}
                  stroke={CHART_COLORS[idx % CHART_COLORS.length]}
                  strokeWidth={2}
                  dot={false}
                  onClick={drillDownEnabled ? handleElementClick : undefined}
                  cursor={drillDownEnabled ? 'pointer' : 'default'}
                  activeDot={drillDownEnabled ? { r: 6 } : undefined}
                />
              )
            )}
          </AreaChart>
        ) : (
          <LineChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey={xField} className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} />
            <YAxis className="text-xs" tick={{ fill: 'hsl(var(--muted-foreground))' }} tickFormatter={formatNumber} />
            <Tooltip content={<CustomTooltip drillDownEnabled={drillDownEnabled} />} />
            {yFields.length > 1 && <Legend />}
            {yFields.map((field, idx) => (
              <Line
                key={field}
                type="monotone"
                dataKey={field}
                stroke={CHART_COLORS[idx % CHART_COLORS.length]}
                strokeWidth={2}
                dot={false}
                onClick={drillDownEnabled ? handleElementClick : undefined}
                cursor={drillDownEnabled ? 'pointer' : 'default'}
                activeDot={drillDownEnabled ? { r: 6 } : undefined}
              />
            ))}
          </LineChart>
        )}
      </ResponsiveContainer>
      <p className="text-xs text-muted-foreground">
        Showing {chartData.length} of {data.rows.length} data points
        {chartData.length < data.rows.length && ' (subsampled for performance)'}
      </p>
    </div>
  )
}
