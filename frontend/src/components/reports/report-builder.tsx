import { useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'
import type { ReportComponent, ReportComponentType } from '@/types/reports'
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
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Top-level filters</CardTitle>
            <p className="text-sm text-muted-foreground">Filters are injected into component queries</p>
          </div>
          <Button variant="outline" size="sm" onClick={addFilterField} disabled={disabled}>
            Add Filter
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {(report.filter?.fields ?? []).length === 0 && (
            <p className="text-sm text-muted-foreground">No filters configured yet.</p>
          )}
          {(report.filter?.fields ?? []).map((field, index) => (
            <div key={field.key} className="grid grid-cols-12 gap-4 rounded-md border p-4">
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

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Components</CardTitle>
            <p className="text-sm text-muted-foreground">Drag-and-drop builder coming soon</p>
          </div>
          <div className="flex space-x-2">
            <ComponentPicker onSelect={addComponent} disabled={disabled} />
            <Button variant="secondary" onClick={onRun} disabled={disabled}>
              Preview Report
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-2">
          {components.length === 0 && (
            <div className="col-span-2 rounded-lg border border-dashed p-6 text-center text-muted-foreground">
              Add your first component (chart, metric, LLM summary, etc.)
            </div>
          )}
          {components.map((component) => (
            <ComponentEditor
              key={component.id}
              component={component}
              disabled={disabled}
              onChange={(updates) => updateComponent(component.id, updates)}
              onRemove={() => removeComponent(component.id)}
            />
          ))}
        </CardContent>
      </Card>

      <Card>
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
  disabled?: boolean
  onChange: (updates: Partial<ReportComponent>) => void
  onRemove: () => void
}

function ComponentEditor({ component, disabled, onChange, onRemove }: ComponentEditorProps) {
  return (
    <Card className="h-full border border-dashed">
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base">{component.title}</CardTitle>
        <Button variant="ghost" size="sm" onClick={onRemove} disabled={disabled}>
          Remove
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label>Title</Label>
          <Input
            value={component.title}
            disabled={disabled}
            onChange={(event) => onChange({ title: event.target.value })}
          />
        </div>
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
        {component.type !== 'llm' && (
          <div className="space-y-2">
            <Label>SQL (or builder output)</Label>
            <Textarea
              className="min-h-[120px]"
              value={component.query?.sql ?? ''}
              disabled={disabled}
              onChange={(event) =>
                onChange({
                  query: {
                    ...(component.query ?? { mode: 'sql' }),
                    sql: event.target.value,
                  },
                })
              }
            />
          </div>
        )}
        {component.type === 'llm' && (
          <div className="space-y-2">
            <Label>Prompt template</Label>
            <Textarea
              className="min-h-[100px]"
              value={component.llm?.promptTemplate ?? ''}
              disabled={disabled}
              onChange={(event) =>
                onChange({
                  llm: {
                    ...(component.llm ?? {
                      provider: 'openai',
                      model: 'gpt-4o-mini',
                      contextComponents: [],
                    }),
                    promptTemplate: event.target.value,
                  },
                })
              }
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
