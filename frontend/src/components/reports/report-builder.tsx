import { debounce } from 'lodash-es'
import { AlertCircle, BarChart3, Edit, Eye, FileQuestion, Filter, Grid3x3, Play, Plus } from 'lucide-react'
import React, { useMemo, useState } from 'react'

import { DashboardCanvas } from '@/components/reports/dashboard-canvas'
import { QueryModeSwitcher } from '@/components/reports/query-mode-switcher'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { useReportStore } from '@/store/report-store'
import type { QueryBuilderState, ReportComponent, ReportComponentType, ReportLayoutSlot,ReportRunComponentResult } from '@/types/reports'
import type { ReportRecord } from '@/types/storage'

interface ReportBuilderProps {
  report: ReportRecord
  disabled?: boolean
  onChange: (update: Partial<ReportRecord>) => void
  onRun?: () => void
}

export function ReportBuilder({ report, disabled, onChange, onRun }: ReportBuilderProps) {
  const components = report.definition?.components ?? []
  const layout = report.definition?.layout ?? []
  const lastRun = useReportStore((state) => state.lastRun)
  const [viewMode, setViewMode] = useState<'canvas' | 'list'>('canvas')
  const [editMode, setEditMode] = useState(true)
  const [selectedComponentId, setSelectedComponentId] = useState<string | null>(null)

  // Create lookup for component results
  const resultsByComponentId = useMemo(() => {
    if (!lastRun) return new Map()
    return new Map(
      lastRun.results.map((result) => [result.componentId, result])
    )
  }, [lastRun])

  const addComponent = (type: ReportComponentType) => {
    const newComponent: ReportComponent = {
      id: crypto.randomUUID(),
      title: `${type.charAt(0).toUpperCase()}${type.slice(1)} Block`,
      description: '',
      type,
      size: { minW: 4, minH: 2, maxW: 12, maxH: 8 },
      query: {
        mode: 'sql',
        sql: 'SELECT 1 as value',
        connectionId: '',
        limit: 500,
        topLevelFilter: [],
        parameters: {},
      },
      chart:
        type === 'chart' || type === 'combo'
          ? { variant: type === 'combo' ? 'combo' : 'line', xField: '', yField: '' }
          : undefined,
      llm:
        type === 'llm'
          ? {
              provider: 'openai',
              model: 'gpt-4o-mini',
              promptTemplate: 'Summarize the selected data set using {{component.previous}}',
              contextComponents: [],
              temperature: 0.2,
              maxTokens: 500,
            }
          : undefined,
    }

    const nextLayout = layout.concat({
      componentId: newComponent.id,
      x: 0,
      y: layout.length * 4,
      w: 6,
      h: 4,
    })

    onChange({
      definition: {
        layout: nextLayout,
        components: [...components, newComponent],
      },
    })
  }

  const updateComponent = (id: string, updates: Partial<ReportComponent>) => {
    onChange({
      definition: {
        layout,
        components: components.map((component) =>
          component.id === id ? { ...component, ...updates } : component
        ),
      },
    })
  }

  const removeComponent = (id: string) => {
    onChange({
      definition: {
        layout: layout.filter((item) => item.componentId !== id),
        components: components.filter((component) => component.id !== id),
      },
    })
    if (selectedComponentId === id) {
      setSelectedComponentId(null)
    }
  }

  const updateLayout = (newLayout: ReportLayoutSlot[]) => {
    onChange({
      definition: {
        layout: newLayout,
        components,
      },
    })
  }

  const handleComponentEdit = (id: string) => {
    setSelectedComponentId(id)
    setViewMode('list')
  }

  const handleComponentRun = (id: string) => {
    // For now, run all components. In future, could support running individual components
    onRun?.()
  }

  const updateFilterField = (index: number, updates: Record<string, unknown>) => {
    const fields = [...(report.filter?.fields ?? [])]
    fields[index] = { ...fields[index], ...updates }
    onChange({ filter: { fields } })
  }

  const addFilterField = () => {
    const fields = [...(report.filter?.fields ?? [])]
    fields.push({ key: `filter_${fields.length + 1}`, label: 'New Filter', type: 'text' })
    onChange({ filter: { fields } })
  }

  const removeFilterField = (index: number) => {
    const fields = [...(report.filter?.fields ?? [])]
    fields.splice(index, 1)
    onChange({ filter: { fields } })
  }

  const scheduleDescription = useMemo(() => {
    const options = report.sync_options ?? { enabled: false, cadence: '@every 1h', target: 'local' }
    if (!options.enabled) return 'Automatic refresh disabled'
    return `Runs ${options.cadence} (${options.target === 'remote' ? 'cloud' : 'local'})`
  }, [report.sync_options])

  const cadenceError = useMemo(() => {
    if (!report.sync_options?.enabled) {
      return undefined
    }
    const cadence = report.sync_options?.cadence ?? ''
    if (!cadence.trim()) {
      return 'Cadence required when auto refresh is enabled.'
    }
    const pattern = /^@every\s+\d+(ms|s|m|h|d)$|^(\S+\s+){4}\S+$/
    if (!pattern.test(cadence.trim())) {
      return 'Use @every intervals (e.g. @every 10m) or standard 5-field cron expressions.'
    }
    return undefined
  }, [report.sync_options])

  return (
    <div className="space-y-6">
      {/* Filters Section - Emphasized with colored accent */}
      <Card className="border-primary/20 bg-primary/5">
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-primary" />
            <div>
              <CardTitle>Top-level filters</CardTitle>
              <p className="text-sm text-muted-foreground">Filters are injected into component queries</p>
            </div>
          </div>
          <Button variant="outline" size="sm" onClick={addFilterField} disabled={disabled}>
            <Plus className="mr-2 h-4 w-4" /> Add Filter
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {(report.filter?.fields ?? []).length === 0 && (
            <p className="text-sm text-muted-foreground">No filters configured yet.</p>
          )}
          {(report.filter?.fields ?? []).map((field, index) => (
            <div key={field.key} className="grid grid-cols-12 gap-4 rounded-md border bg-background p-4">
              <div className="col-span-3 space-y-1">
                <Label htmlFor={`filter-key-${index}`}>Key</Label>
                <Input
                  id={`filter-key-${index}`}
                  value={field.key}
                  disabled={disabled}
                  onChange={(event) => updateFilterField(index, { key: event.target.value })}
                />
              </div>
              <div className="col-span-3 space-y-1">
                <Label htmlFor={`filter-label-${index}`}>Label</Label>
                <Input
                  id={`filter-label-${index}`}
                  value={field.label}
                  disabled={disabled}
                  onChange={(event) => updateFilterField(index, { label: event.target.value })}
                />
              </div>
              <div className="col-span-3 space-y-1">
                <Label>Type</Label>
                <Select
                  value={(field.type as string) ?? 'text'}
                  onValueChange={(value) => updateFilterField(index, { type: value })}
                  disabled={disabled}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="text">Text</SelectItem>
                    <SelectItem value="number">Number</SelectItem>
                    <SelectItem value="date">Date</SelectItem>
                    <SelectItem value="select">Select</SelectItem>
                    <SelectItem value="multi-select">Multi Select</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="col-span-2 flex items-center space-x-2">
                <Label>Required</Label>
                <Switch
                  checked={Boolean(field.required)}
                  onCheckedChange={(checked) => updateFilterField(index, { required: checked })}
                  disabled={disabled}
                />
              </div>
              <div className="col-span-1 flex items-start justify-end">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => removeFilterField(index)}
                  disabled={disabled}
                >
                  Remove
                </Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Components Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Components</h3>
            <p className="text-sm text-muted-foreground">Build your report with charts, tables, metrics, and AI summaries</p>
          </div>
          <div className="flex gap-2">
            {/* View Mode Toggle */}
            <div className="flex items-center gap-1 rounded-md border p-1">
              <Button
                variant={viewMode === 'canvas' ? 'secondary' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('canvas')}
                disabled={disabled}
              >
                <Grid3x3 className="mr-2 h-4 w-4" /> Canvas
              </Button>
              <Button
                variant={viewMode === 'list' ? 'secondary' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('list')}
                disabled={disabled}
              >
                <Edit className="mr-2 h-4 w-4" /> List
              </Button>
            </div>

            {/* Edit/View Mode Toggle (only in canvas view) */}
            {viewMode === 'canvas' && components.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setEditMode(!editMode)}
                disabled={disabled}
              >
                {editMode ? <Eye className="mr-2 h-4 w-4" /> : <Edit className="mr-2 h-4 w-4" />}
                {editMode ? 'View Mode' : 'Edit Mode'}
              </Button>
            )}

            <ComponentPicker onSelect={addComponent} disabled={disabled} />
            <Button variant="secondary" onClick={onRun} disabled={disabled}>
              <Play className="mr-2 h-4 w-4" /> Run All
            </Button>
          </div>
        </div>

        {/* Canvas View */}
        {viewMode === 'canvas' && (
          <DashboardCanvas
            components={components}
            layout={layout}
            results={lastRun ?? undefined}
            onLayoutChange={updateLayout}
            onComponentEdit={handleComponentEdit}
            onComponentRun={handleComponentRun}
            onComponentDelete={removeComponent}
            onAddComponent={addComponent}
            editMode={editMode}
          />
        )}

        {/* List View */}
        {viewMode === 'list' && (
          <>
            {components.length === 0 ? (
              <EmptyState
                icon={BarChart3}
                title="No components yet"
                description="Components are the building blocks of your report. Add charts, tables, metrics, or AI summaries to get started."
                action={
                  <Button onClick={() => addComponent('chart')}>
                    <Plus className="mr-2 h-4 w-4" /> Add Your First Component
                  </Button>
                }
              />
            ) : (
              <div className="grid gap-4 lg:grid-cols-2">
                {components.map((component) => (
                  <ComponentEditor
                    key={component.id}
                    component={component}
                    result={resultsByComponentId.get(component.id)}
                    disabled={disabled}
                    onChange={(updates) => updateComponent(component.id, updates)}
                    onRemove={() => removeComponent(component.id)}
                    onRun={onRun}
                    isSelected={selectedComponentId === component.id}
                  />
                ))}
              </div>
            )}
          </>
        )}
      </div>

      {/* Sync & Scheduling Section */}
      <Card className="border-muted">
        <CardHeader>
          <CardTitle>Sync & Scheduling</CardTitle>
          <p className="text-sm text-muted-foreground">{scheduleDescription}</p>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div className="space-y-2">
            <Label>Enable automatic refresh</Label>
            <Switch
              checked={Boolean(report.sync_options?.enabled)}
              onCheckedChange={(checked) =>
                onChange({ sync_options: { ...(report.sync_options ?? {}), enabled: checked } })
              }
              disabled={disabled}
            />
          </div>
          <div className="space-y-2">
            <Label>Cadence</Label>
            <Input
              value={report.sync_options?.cadence ?? '@every 1h'}
              onChange={(event) =>
                onChange({ sync_options: { ...(report.sync_options ?? {}), cadence: event.target.value } })
              }
              disabled={disabled}
            />
            <p className={cn('text-xs', cadenceError ? 'text-destructive' : 'text-muted-foreground')}>
              {cadenceError ?? 'Supports cron syntax or @every duration (e.g. @every 5m)'}
            </p>
          </div>
          <div className="space-y-2">
            <Label>Target</Label>
            <Select
              value={report.sync_options?.target ?? 'local'}
              onValueChange={(value) =>
                onChange({ sync_options: { ...(report.sync_options ?? {}), target: value as 'local' | 'remote' } })
              }
              disabled={disabled}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="local">Local (this device)</SelectItem>
                <SelectItem value="remote">Remote (cloud)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

interface ComponentPickerProps {
  onSelect: (type: ReportComponentType) => void
  disabled?: boolean
}

function ComponentPicker({ onSelect, disabled }: ComponentPickerProps) {
  return (
    <Select onValueChange={(value) => onSelect(value as ReportComponentType)} disabled={disabled}>
      <SelectTrigger className="w-[180px]">
        <SelectValue placeholder="Add component" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="chart">Chart</SelectItem>
        <SelectItem value="metric">Metric</SelectItem>
        <SelectItem value="table">Table</SelectItem>
        <SelectItem value="combo">Combo Chart</SelectItem>
        <SelectItem value="llm">LLM Summary</SelectItem>
      </SelectContent>
    </Select>
  )
}

interface ComponentEditorProps {
  component: ReportComponent
  result?: ReportRunComponentResult
  disabled?: boolean
  onChange: (updates: Partial<ReportComponent>) => void
  onRemove: () => void
  onRun?: () => void
  isSelected?: boolean
}

/**
 * Memoized component editor with debounced text inputs for optimal performance.
 * Only re-renders when component data, result, or disabled state changes.
 */
const ComponentEditor = React.memo(
  ({ component, result, disabled, onChange, onRemove, onRun, isSelected }: ComponentEditorProps) => {
    // Debounce title updates
    const debouncedTitleUpdate = useMemo(
      () =>
        debounce((title: string) => {
          onChange({ title })
        }, 300),
      [onChange]
    )

    // Debounce SQL query updates (longer delay for potentially larger text)
    const debouncedSqlUpdate = useMemo(
      () =>
        debounce((sql: string) => {
          onChange({
            query: {
              ...(component.query ?? { mode: 'sql' }),
              sql,
            },
          })
        }, 500),
      [onChange, component.query]
    )

    // Debounce LLM prompt updates
    const debouncedPromptUpdate = useMemo(
      () =>
        debounce((promptTemplate: string) => {
          onChange({
            llm: {
              ...(component.llm ?? {
                provider: 'openai',
                model: 'gpt-4o-mini',
                contextComponents: [],
              }),
              promptTemplate,
            },
          })
        }, 500),
      [onChange, component.llm]
    )

    // Cleanup debounced functions on unmount
    React.useEffect(() => {
      return () => {
        debouncedTitleUpdate.cancel()
        debouncedSqlUpdate.cancel()
        debouncedPromptUpdate.cancel()
      }
    }, [debouncedTitleUpdate, debouncedSqlUpdate, debouncedPromptUpdate])

    return (
    <Card className={cn('transition-shadow hover:shadow-md', isSelected && 'ring-2 ring-primary')}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{component.title}</CardTitle>
          <Button variant="ghost" size="sm" onClick={onRemove} disabled={disabled}>
            Remove
          </Button>
        </div>
      </CardHeader>

      <Tabs defaultValue="config" className="w-full">
        <TabsList className="w-full grid grid-cols-2">
          <TabsTrigger value="config">Configuration</TabsTrigger>
          <TabsTrigger value="preview">
            Preview
            {result && (
              <Badge variant={result.error ? 'destructive' : 'default'} className="ml-2">
                {result.error ? 'Error' : `${result.rows?.length || 0} rows`}
              </Badge>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="config" className="space-y-4 px-6 pb-6">
          {/* Title */}
          <div className="space-y-2">
            <Label>Title</Label>
            <Input
              defaultValue={component.title}
              disabled={disabled}
              onChange={(event) => debouncedTitleUpdate(event.target.value)}
              key={component.id} // Force re-render when component changes
            />
          </div>

          {/* Type */}
          <div className="space-y-2">
            <Label>Type</Label>
            <Select
              value={component.type}
              onValueChange={(value) => onChange({ type: value as ReportComponentType })}
              disabled={disabled}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="chart">Chart</SelectItem>
                <SelectItem value="metric">Metric</SelectItem>
                <SelectItem value="table">Table</SelectItem>
                <SelectItem value="combo">Combo</SelectItem>
                <SelectItem value="llm">LLM Summary</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Query Mode Switcher for non-LLM components */}
          {component.type !== 'llm' && (
            <QueryModeSwitcher
              mode={component.query?.mode || 'sql'}
              connectionId={component.query?.connectionId}
              sql={component.query?.sql}
              builderState={component.query?.builderState as QueryBuilderState | undefined}
              onChange={(updates) => {
                onChange({
                  query: {
                    ...(component.query ?? {}),
                    mode: updates.mode || component.query?.mode || 'sql',
                    sql: updates.sql !== undefined ? updates.sql : component.query?.sql,
                    builderState: updates.builderState || component.query?.builderState,
                  },
                })
              }}
              disabled={disabled}
            />
          )}

          {/* LLM Prompt */}
          {component.type === 'llm' && (
            <div className="space-y-2">
              <Label>Prompt template</Label>
              <Textarea
                className="min-h-[100px]"
                defaultValue={component.llm?.promptTemplate ?? ''}
                disabled={disabled}
                onChange={(event) => debouncedPromptUpdate(event.target.value)}
                key={`${component.id}-prompt`}
              />
            </div>
          )}
        </TabsContent>

        <TabsContent value="preview" className="px-6 pb-6">
          {!result ? (
            <EmptyState
              icon={FileQuestion}
              title="No preview available"
              description="Run this component to see results"
              action={
                <Button onClick={onRun} size="sm">
                  <Play className="mr-2 h-4 w-4" /> Run Component
                </Button>
              }
            />
          ) : result.error ? (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Query Failed</AlertTitle>
              <AlertDescription>{result.error}</AlertDescription>
            </Alert>
          ) : (
            <ComponentPreview type={component.type} result={result} />
          )}
        </TabsContent>
      </Tabs>

      <CardFooter className="border-t pt-4">
        <Button size="sm" onClick={onRun} disabled={disabled}>
          <Play className="h-4 w-4 mr-2" /> Run Component
        </Button>
      </CardFooter>
    </Card>
    )
  },
  (prev, next) => {
    // Custom comparison: only re-render if component data, result, disabled, or selected state changes
    return (
      prev.component.id === next.component.id &&
      prev.disabled === next.disabled &&
      prev.isSelected === next.isSelected &&
      JSON.stringify(prev.component) === JSON.stringify(next.component) &&
      JSON.stringify(prev.result) === JSON.stringify(next.result)
    )
  }
)

ComponentEditor.displayName = 'ComponentEditor'

interface ComponentPreviewProps {
  type: ReportComponentType
  result: ReportRunComponentResult
}

function ComponentPreview({ type, result }: ComponentPreviewProps) {
  if (result.content) {
    return (
      <div className="whitespace-pre-wrap rounded-md bg-muted p-4 text-sm">
        {result.content}
      </div>
    )
  }

  if (result.rows && result.columns) {
    return (
      <div className="overflow-auto rounded-md border">
        <table className="w-full min-w-[300px] text-sm">
          <thead>
            <tr className="border-b bg-muted/50 text-left">
              {result.columns.map((column) => (
                <th key={column} className="px-4 py-2 font-medium">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {result.rows.slice(0, 10).map((row, rowIndex) => (
              <tr key={rowIndex} className="border-b last:border-none hover:bg-muted/30">
                {row.map((value, colIndex) => (
                  <td key={`${rowIndex}-${colIndex}`} className="px-4 py-2">
                    {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {result.rows.length > 10 && (
          <div className="border-t bg-muted/30 px-4 py-2 text-xs text-muted-foreground">
            Showing first 10 of {result.rows.length} rows
          </div>
        )}
      </div>
    )
  }

  return <p className="text-sm text-muted-foreground">No preview data available</p>
}
