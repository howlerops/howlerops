import { useEffect, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ReportBuilder } from '@/components/reports/report-builder'
import { useReportStore } from '@/store/report-store'
import { cn } from '@/lib/utils'
import { Loader2, Play, Save, Trash2, PlusCircle } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

export function ReportsPage() {
  const summaries = useReportStore((state) => state.summaries)
  const activeReport = useReportStore((state) => state.activeReport)
  const loading = useReportStore((state) => state.loading)
  const error = useReportStore((state) => state.error)
  const lastRun = useReportStore((state) => state.lastRun)
  const fetchReports = useReportStore((state) => state.fetchReports)
  const selectReport = useReportStore((state) => state.selectReport)
  const updateActive = useReportStore((state) => state.updateActive)
  const saveActive = useReportStore((state) => state.saveActive)
  const runActive = useReportStore((state) => state.runActive)
  const createReport = useReportStore((state) => state.createReport)
  const deleteReport = useReportStore((state) => state.deleteReport)
  const topLevelFilters = useReportStore((state) => state.topLevelFilters)
  const setTopLevelFilters = useReportStore((state) => state.setTopLevelFilters)
  const { toast } = useToast()

  useEffect(() => {
    fetchReports().catch(console.error)
  }, [fetchReports])

  const handleCreate = async () => {
    await createReport()
  }

  const handleSave = async () => {
    await saveActive()
    toast({ title: 'Report saved', duration: 2000 })
  }

  const handleDelete = async () => {
    if (!activeReport) return
    await deleteReport(activeReport.id)
    toast({ title: 'Report deleted', variant: 'destructive', duration: 2000 })
  }

  const handleRun = async () => {
    try {
      await runActive()
      toast({ title: 'Report executed', duration: 2000 })
    } catch (err) {
      toast({
        title: 'Run failed',
        description: err instanceof Error ? err.message : 'Unable to execute report',
        variant: 'destructive',
      })
    }
  }

  const activeSummary = useMemo(() => {
    if (!activeReport) return undefined
    return summaries.find((summary) => summary.id === activeReport.id)
  }, [activeReport, summaries])

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="flex-1 space-y-6 overflow-y-auto p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Reports</h1>
            <p className="text-sm text-muted-foreground">
              Build drag-and-drop dashboards with reusable filters, charts, and AI summaries.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleCreate} disabled={loading}>
            <PlusCircle className="mr-2 h-4 w-4" /> New Report
          </Button>
          <Button variant="outline" onClick={handleSave} disabled={!activeReport || loading}>
            <Save className="mr-2 h-4 w-4" /> Save
          </Button>
          <Button variant="outline" onClick={handleRun} disabled={!activeReport || loading}>
            <Play className="mr-2 h-4 w-4" /> Run
          </Button>
          <Button variant="destructive" onClick={handleDelete} disabled={!activeReport || loading}>
            <Trash2 className="mr-2 h-4 w-4" /> Delete
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Unable to load reports</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
        <aside className="space-y-4">
          <Input placeholder="Search reports" disabled={loading} />
          <div className="space-y-2">
            {summaries.map((summary) => (
              <button
                key={summary.id}
                onClick={() => selectReport(summary.id)}
                className={cn(
                  'w-full rounded-md border px-3 py-2 text-left text-sm transition',
                  activeReport?.id === summary.id ? 'border-primary bg-primary/5' : 'hover:bg-muted'
                )}
                disabled={loading}
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium">{summary.name}</span>
                  <Badge variant={summary.lastRunStatus === 'ok' ? 'outline' : 'secondary'}>
                    {summary.lastRunStatus ?? 'idle'}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  Updated {summary.updatedAt.toLocaleString()}
                </p>
              </button>
            ))}
            {summaries.length === 0 && (
              <div className="rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground">
                No reports yet. Create your first report to get started.
              </div>
            )}
          </div>
        </aside>

        <section className="space-y-6">
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" /> Loading reports...
            </div>
          )}

          {!activeReport && (
            <Card className="border-dashed">
              <CardHeader>
                <CardTitle>Select or create a report</CardTitle>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                Choose a report from the left panel or click “New Report” to start designing a layout.
              </CardContent>
            </Card>
          )}

          {activeReport && (
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Report details</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Name</label>
                    <Input
                      value={activeReport.name}
                      onChange={(event) => updateActive({ name: event.target.value })}
                      disabled={loading}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Folder</label>
                    <Input
                      value={activeReport.folder}
                      onChange={(event) => updateActive({ folder: event.target.value })}
                      disabled={loading}
                    />
                  </div>
                  <div className="col-span-2 space-y-2">
                    <label className="text-sm font-medium">Description</label>
                    <Textarea
                      value={activeReport.description}
                      rows={3}
                      onChange={(event) => updateActive({ description: event.target.value })}
                      disabled={loading}
                    />
                  </div>
                </CardContent>
              </Card>

              {activeSummary && (
                <Card>
                  <CardHeader>
                    <CardTitle>Execution status</CardTitle>
                  </CardHeader>
                  <CardContent className="flex flex-wrap gap-4 text-sm">
                    <div>
                      <p className="text-muted-foreground">Last run</p>
                      <p className="font-medium">
                        {activeSummary.lastRunAt ? activeSummary.lastRunAt.toLocaleString() : 'Never'}
                      </p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Status</p>
                      <Badge variant={activeSummary.lastRunStatus === 'ok' ? 'outline' : 'secondary'}>
                        {activeSummary.lastRunStatus ?? 'idle'}
                      </Badge>
                    </div>
                    {lastRun && (
                      <div>
                        <p className="text-muted-foreground">Components</p>
                        <p>
                          {lastRun.results.filter((result) => !result.error).length} successful /
                          {lastRun.results.length} total
                        </p>
                      </div>
                    )}
                </CardContent>
              </Card>
              )}

              {activeReport.filter?.fields?.length ? (
                <Card>
                  <CardHeader>
                    <CardTitle>Top-level filters</CardTitle>
                    <p className="text-sm text-muted-foreground">
                      Values here will be substituted into component queries before execution.
                    </p>
                  </CardHeader>
                  <CardContent className="grid gap-4 md:grid-cols-2">
                    {activeReport.filter.fields.map((field) => (
                      <div key={field.key} className="space-y-2">
                        <label className="text-sm font-medium flex items-center justify-between">
                          <span>{field.label}</span>
                          {field.required && <Badge variant="outline">Required</Badge>}
                        </label>
                        {field.type === 'select' || field.type === 'multi-select' ? (
                          <Select
                            value={String(topLevelFilters[field.key] ?? '')}
                            onValueChange={(value) =>
                              setTopLevelFilters({ [field.key]: field.type === 'number' ? Number(value) : value })
                            }
                            disabled={loading}
                          >
                            <SelectTrigger>
                              <SelectValue placeholder="Select option" />
                            </SelectTrigger>
                            <SelectContent>
                              {(field.choices ?? []).map((choice) => (
                                <SelectItem key={choice} value={choice}>
                                  {choice}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Input
                            type={field.type === 'number' ? 'number' : field.type === 'date' ? 'date' : 'text'}
                            value={field.type === 'date'
                              ? String(topLevelFilters[field.key] ?? '').slice(0, 10)
                              : String(topLevelFilters[field.key] ?? '')}
                            onChange={(event) => {
                              const value = field.type === 'number'
                                ? Number(event.target.value)
                                : event.target.value
                              setTopLevelFilters({ [field.key]: value })
                            }}
                            placeholder={field.label}
                            disabled={loading}
                          />
                        )}
                      </div>
                    ))}
                  </CardContent>
                </Card>
              ) : null}

              {lastRun && lastRun.results.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle>Latest execution</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {lastRun.results.map((result) => (
                      <div key={result.componentId} className="rounded-md border p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="font-medium">Component {result.componentId}</p>
                            <p className="text-xs text-muted-foreground">{result.type}</p>
                          </div>
                          {result.error ? (
                            <Badge variant="destructive">Error</Badge>
                          ) : (
                            <Badge variant="outline">{result.rows?.length ?? 0} rows</Badge>
                          )}
                        </div>
                        {result.error ? (
                          <p className="mt-2 text-sm text-destructive">{result.error}</p>
                        ) : result.content ? (
                          <p className="mt-2 whitespace-pre-wrap text-sm">{result.content}</p>
                        ) : result.rows && result.columns ? (
                          <div className="mt-3 overflow-auto">
                            <table className="w-full min-w-[300px] text-sm">
                              <thead>
                                <tr className="border-b text-left">
                                  {result.columns.map((column) => (
                                    <th key={column} className="px-2 py-1 font-medium">
                                      {column}
                                    </th>
                                  ))}
                                </tr>
                              </thead>
                              <tbody>
                                {result.rows.slice(0, 5).map((row, rowIndex) => (
                                  <tr key={rowIndex} className="border-b last:border-none">
                                    {row.map((value, colIndex) => (
                                      <td key={`${rowIndex}-${colIndex}`} className="px-2 py-1">
                                        {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                                      </td>
                                    ))}
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                            {result.rows.length > 5 && (
                              <p className="mt-1 text-xs text-muted-foreground">
                                Showing first 5 of {result.rows.length} rows
                              </p>
                            )}
                          </div>
                        ) : (
                          <p className="mt-2 text-sm text-muted-foreground">No preview available</p>
                        )}
                      </div>
                    ))}
                  </CardContent>
                </Card>
              )}

              <ReportBuilder report={activeReport} disabled={loading} onChange={updateActive} onRun={handleRun} />
            </div>
          )}
        </section>
      </div>
    </div>
    </div>
  )
}

export default ReportsPage
