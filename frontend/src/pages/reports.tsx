import { debounce } from 'lodash-es'
import { Copy, Download, FileText, Loader2, MoreVertical, Play, PlusCircle, Save, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { ReportBuilder } from '@/components/reports/report-builder'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'
import { useReportStore } from '@/store/report-store'

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

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

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
    setDeleteDialogOpen(true)
  }

  const confirmDelete = async () => {
    if (!activeReport) return
    await deleteReport(activeReport.id)
    toast({ title: 'Report deleted', variant: 'destructive', duration: 2000 })
    setDeleteDialogOpen(false)
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

  // Optimize: Only re-compute when activeReport ID changes, not whole object
  const activeSummary = useMemo(() => {
    if (!activeReport?.id) return undefined
    return summaries.find((summary) => summary.id === activeReport.id)
  }, [activeReport?.id, summaries])

  // Debounce text input updates to reduce re-renders and store updates
  const debouncedUpdateName = useMemo(
    () =>
      debounce((name: string) => {
        updateActive({ name })
      }, 300),
    [updateActive]
  )

  const debouncedUpdateFolder = useMemo(
    () =>
      debounce((folder: string) => {
        updateActive({ folder })
      }, 300),
    [updateActive]
  )

  const debouncedUpdateDescription = useMemo(
    () =>
      debounce((description: string) => {
        updateActive({ description })
      }, 500), // Longer debounce for longer text
    [updateActive]
  )

  // Filter reports by search query
  const filteredSummaries = useMemo(() => {
    if (!searchQuery.trim()) return summaries
    const query = searchQuery.toLowerCase()
    return summaries.filter(
      (summary) =>
        summary.name.toLowerCase().includes(query) ||
        (activeReport?.description?.toLowerCase().includes(query) ?? false)
    )
  }, [summaries, searchQuery, activeReport])

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="flex-1 space-y-6 overflow-y-auto p-6">
        {/* Header */}
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Reports</h1>
            <p className="text-sm text-muted-foreground">
              Build drag-and-drop dashboards with reusable filters, charts, and AI summaries.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button onClick={handleCreate} disabled={loading}>
              <PlusCircle className="mr-2 h-4 w-4" /> New Report
            </Button>
            <Button variant="outline" onClick={handleSave} disabled={!activeReport || loading}>
              <Save className="mr-2 h-4 w-4" /> Save
            </Button>
            <Button onClick={handleRun} disabled={!activeReport || loading}>
              <Play className="mr-2 h-4 w-4" /> Run
            </Button>

            {/* Action overflow menu */}
            {activeReport && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon">
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => toast({ title: 'Duplicate feature coming soon' })}>
                    <Copy className="mr-2 h-4 w-4" /> Duplicate
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => toast({ title: 'Export feature coming soon' })}>
                    <Download className="mr-2 h-4 w-4" /> Export...
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={handleDelete} className="text-destructive focus:text-destructive">
                    <Trash2 className="mr-2 h-4 w-4" /> Delete Report
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertTitle>Unable to load reports</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
          {/* Sidebar */}
          <aside className="space-y-4">
            <Input
              placeholder="Search reports"
              disabled={loading}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {loading && summaries.length === 0 ? (
              <div className="space-y-2">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
              </div>
            ) : (
              <div className="space-y-2">
                {filteredSummaries.map((summary) => (
                  <button
                    key={summary.id}
                    onClick={() => selectReport(summary.id)}
                    className={cn(
                      'w-full rounded-md border px-3 py-2 text-left text-sm transition hover:bg-muted',
                      activeReport?.id === summary.id ? 'border-primary bg-primary/5' : ''
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
                {filteredSummaries.length === 0 && !loading && (
                  <EmptyState
                    icon={FileText}
                    title={searchQuery ? 'No matches' : 'No reports yet'}
                    description={
                      searchQuery
                        ? 'Try a different search term'
                        : 'Create your first report to get started.'
                    }
                  />
                )}
              </div>
            )}
          </aside>

          {/* Main content */}
          <section className="space-y-6">
            {loading && !activeReport ? (
              <div className="space-y-4">
                <Skeleton className="h-12 w-full" />
                <div className="grid gap-4 md:grid-cols-2">
                  <Skeleton className="h-64 w-full" />
                  <Skeleton className="h-64 w-full" />
                </div>
              </div>
            ) : !activeReport ? (
              <EmptyState
                icon={FileText}
                title="Select or create a report"
                description="Choose a report from the left panel or click 'New Report' to start designing a layout."
              />
            ) : (
              <div className="space-y-6">
                {/* Report details */}
                <Card className="border-muted">
                  <CardHeader>
                    <CardTitle>Report details</CardTitle>
                  </CardHeader>
                  <CardContent className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Name</label>
                      <Input
                        defaultValue={activeReport.name}
                        onChange={(event) => debouncedUpdateName(event.target.value)}
                        disabled={loading}
                        key={activeReport.id} // Force re-render when activeReport changes
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Folder</label>
                      <Input
                        defaultValue={activeReport.folder}
                        onChange={(event) => debouncedUpdateFolder(event.target.value)}
                        disabled={loading}
                        key={`${activeReport.id}-folder`}
                      />
                    </div>
                    <div className="col-span-2 space-y-2">
                      <label className="text-sm font-medium">Description</label>
                      <Textarea
                        defaultValue={activeReport.description}
                        rows={3}
                        onChange={(event) => debouncedUpdateDescription(event.target.value)}
                        disabled={loading}
                        key={`${activeReport.id}-desc`}
                      />
                    </div>
                  </CardContent>
                </Card>

                {/* Execution status */}
                {activeSummary && (
                  <Card className="border-muted">
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

                {/* Top-level filters */}
                {activeReport.filter?.fields?.length ? (
                  <Card className="border-primary/20 bg-primary/5">
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
                              value={
                                field.type === 'date'
                                  ? String(topLevelFilters[field.key] ?? '').slice(0, 10)
                                  : String(topLevelFilters[field.key] ?? '')
                              }
                              onChange={(event) => {
                                const value =
                                  field.type === 'number' ? Number(event.target.value) : event.target.value
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

                {/* Report builder */}
                <ReportBuilder report={activeReport} disabled={loading} onChange={updateActive} onRun={handleRun} />
              </div>
            )}
          </section>
        </div>
      </div>

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        title="Delete Report?"
        description={`"${activeReport?.name}" will be permanently deleted. This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={confirmDelete}
        onCancel={() => setDeleteDialogOpen(false)}
      />
    </div>
  )
}

export default ReportsPage
