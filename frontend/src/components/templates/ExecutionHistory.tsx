/**
 * ExecutionHistory Component
 * Display execution history for scheduled queries with timeline and charts
 */

import React, { useEffect, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  CheckCircle,
  XCircle,
  Clock,
  AlertCircle,
  TrendingUp,
  Database,
  BarChart3,
} from 'lucide-react'
import { useTemplatesStore } from '@/store/templates-store'
import type { ScheduleExecution } from '@/types/templates'
import { formatDistanceToNow, format } from 'date-fns'

interface ExecutionHistoryProps {
  scheduleId: string
  open: boolean
  onClose: () => void
}

const STATUS_CONFIG = {
  success: {
    label: 'Success',
    icon: CheckCircle,
    className: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  },
  failed: {
    label: 'Failed',
    icon: XCircle,
    className: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  },
  timeout: {
    label: 'Timeout',
    icon: Clock,
    className: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
  },
}

export function ExecutionHistory({ scheduleId, open, onClose }: ExecutionHistoryProps) {
  const { schedules, executions, fetchExecutions, loading } = useTemplatesStore()
  const [selectedExecution, setSelectedExecution] = useState<ScheduleExecution | null>(null)

  const schedule = schedules.find((s) => s.id === scheduleId)

  // Memoize scheduleExecutions to prevent creating new array reference on every render
  const scheduleExecutions = React.useMemo(
    () => executions.get(scheduleId) || [],
    [executions, scheduleId]
  )

  useEffect(() => {
    if (open && scheduleId) {
      fetchExecutions(scheduleId)
    }
  }, [open, scheduleId, fetchExecutions])

  // Calculate statistics
  const stats = React.useMemo(() => {
    const total = scheduleExecutions.length
    const successful = scheduleExecutions.filter((e) => e.status === 'success').length
    const failed = scheduleExecutions.filter((e) => e.status === 'failed').length
    const timedOut = scheduleExecutions.filter((e) => e.status === 'timeout').length
    const successRate = total > 0 ? (successful / total) * 100 : 0

    const avgDuration =
      successful > 0
        ? scheduleExecutions
            .filter((e) => e.status === 'success')
            .reduce((sum, e) => sum + e.duration_ms, 0) / successful
        : 0

    const totalRows = scheduleExecutions
      .filter((e) => e.rows_returned !== undefined)
      .reduce((sum, e) => sum + (e.rows_returned || 0), 0)

    return {
      total,
      successful,
      failed,
      timedOut,
      successRate,
      avgDuration,
      totalRows,
    }
  }, [scheduleExecutions])

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-5xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Execution History
          </DialogTitle>
          <DialogDescription>
            {schedule?.name || 'Schedule'} - Last {scheduleExecutions.length} executions
          </DialogDescription>
        </DialogHeader>

        {/* Statistics */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard
            label="Total Runs"
            value={stats.total}
            icon={BarChart3}
          />
          <StatCard
            label="Success Rate"
            value={`${stats.successRate.toFixed(1)}%`}
            icon={TrendingUp}
            className={stats.successRate >= 90 ? 'text-green-600' : 'text-yellow-600'}
          />
          <StatCard
            label="Avg Duration"
            value={`${stats.avgDuration.toFixed(0)}ms`}
            icon={Clock}
          />
          <StatCard
            label="Total Rows"
            value={stats.totalRows.toLocaleString()}
            icon={Database}
          />
        </div>

        {/* Status Breakdown */}
        <div className="flex gap-4 text-sm">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500" />
            <span>{stats.successful} Successful</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-red-500" />
            <span>{stats.failed} Failed</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-orange-500" />
            <span>{stats.timedOut} Timeout</span>
          </div>
        </div>

        {/* Executions Timeline */}
        <div className="flex-1 overflow-hidden flex flex-col">
          <h3 className="font-semibold mb-3">Timeline</h3>

          {loading && scheduleExecutions.length === 0 ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <ExecutionItemSkeleton key={i} />
              ))}
            </div>
          ) : scheduleExecutions.length === 0 ? (
            <Alert>
              <AlertDescription>No execution history available yet.</AlertDescription>
            </Alert>
          ) : (
            <ScrollArea className="flex-1">
              <div className="space-y-2 pr-4">
                {scheduleExecutions.map((execution) => (
                  <ExecutionItem
                    key={execution.id}
                    execution={execution}
                    isSelected={selectedExecution?.id === execution.id}
                    onClick={() => setSelectedExecution(execution)}
                  />
                ))}
              </div>
            </ScrollArea>
          )}
        </div>

        {/* Execution Details */}
        {selectedExecution && (
          <div className="border-t pt-4">
            <h3 className="font-semibold mb-3">Execution Details</h3>
            <ExecutionDetails execution={selectedExecution} />
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

// ============================================================================
// Sub-Components
// ============================================================================

interface StatCardProps {
  label: string
  value: string | number
  icon: React.ComponentType<{ className?: string }>
  className?: string
}

function StatCard({ label, value, icon: Icon, className }: StatCardProps) {
  return (
    <div className="border rounded-lg p-4">
      <div className="flex items-center gap-2 text-muted-foreground text-sm mb-1">
        <Icon className="h-4 w-4" />
        {label}
      </div>
      <div className={`text-2xl font-bold ${className || ''}`}>{value}</div>
    </div>
  )
}

interface ExecutionItemProps {
  execution: ScheduleExecution
  isSelected: boolean
  onClick: () => void
}

function ExecutionItem({ execution, isSelected, onClick }: ExecutionItemProps) {
  const statusConfig = STATUS_CONFIG[execution.status]
  const StatusIcon = statusConfig.icon

  return (
    <button
      onClick={onClick}
      className={`w-full text-left p-4 rounded-lg border transition-colors ${
        isSelected ? 'border-primary bg-accent' : 'hover:bg-accent/50'
      }`}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <Badge variant="secondary" className={statusConfig.className}>
              <StatusIcon className="h-3 w-3 mr-1" />
              {statusConfig.label}
            </Badge>
            <span className="text-sm text-muted-foreground">
              {formatDistanceToNow(new Date(execution.executed_at), { addSuffix: true })}
            </span>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mt-2 text-sm">
            <div>
              <span className="text-muted-foreground">Time:</span>{' '}
              <span className="font-medium">
                {format(new Date(execution.executed_at), 'MMM d, HH:mm')}
              </span>
            </div>
            <div>
              <span className="text-muted-foreground">Duration:</span>{' '}
              <span className="font-medium">{execution.duration_ms}ms</span>
            </div>
            {execution.rows_returned !== undefined && (
              <div>
                <span className="text-muted-foreground">Rows:</span>{' '}
                <span className="font-medium">{execution.rows_returned}</span>
              </div>
            )}
          </div>

          {execution.error_message && (
            <p className="text-xs text-red-500 mt-2 truncate" title={execution.error_message}>
              {execution.error_message}
            </p>
          )}
        </div>
      </div>
    </button>
  )
}

function ExecutionDetails({ execution }: { execution: ScheduleExecution }) {
  return (
    <div className="space-y-3 text-sm">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <span className="text-muted-foreground">Execution ID:</span>
          <p className="font-mono text-xs mt-1">{execution.id}</p>
        </div>
        <div>
          <span className="text-muted-foreground">Executed At:</span>
          <p className="font-medium mt-1">
            {format(new Date(execution.executed_at), 'PPpp')}
          </p>
        </div>
        <div>
          <span className="text-muted-foreground">Duration:</span>
          <p className="font-medium mt-1">{execution.duration_ms}ms</p>
        </div>
        {execution.rows_returned !== undefined && (
          <div>
            <span className="text-muted-foreground">Rows Returned:</span>
            <p className="font-medium mt-1">{execution.rows_returned.toLocaleString()}</p>
          </div>
        )}
      </div>

      {execution.error_message && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="text-xs font-mono">
            {execution.error_message}
          </AlertDescription>
        </Alert>
      )}

      {execution.result_preview && execution.result_preview.length > 0 && (
        <div>
          <span className="text-muted-foreground">Result Preview:</span>
          <div className="border rounded-lg overflow-auto mt-2 max-h-48">
            <table className="w-full text-xs">
              <thead className="bg-muted sticky top-0">
                <tr>
                  {Object.keys(execution.result_preview[0]).map((key) => (
                    <th key={key} className="px-3 py-2 text-left font-medium">
                      {key}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {execution.result_preview.slice(0, 5).map((row, i) => (
                  <tr key={i} className="border-t">
                    {Object.values(row).map((val, j) => (
                      <td key={j} className="px-3 py-2">
                        {val === null ? (
                          <span className="text-muted-foreground italic">null</span>
                        ) : (
                          String(val)
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {execution.result_preview.length > 5 && (
            <p className="text-xs text-muted-foreground mt-2">
              Showing 5 of {execution.result_preview.length} rows
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function ExecutionItemSkeleton() {
  return (
    <div className="p-4 rounded-lg border">
      <div className="space-y-2">
        <div className="h-5 bg-muted rounded animate-pulse w-1/4" />
        <div className="h-4 bg-muted rounded animate-pulse w-full" />
        <div className="h-4 bg-muted rounded animate-pulse w-2/3" />
      </div>
    </div>
  )
}
