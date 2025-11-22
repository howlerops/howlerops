/**
 * SchedulesPage Component
 * Page for managing scheduled queries
 */

import { formatDistanceToNow } from 'date-fns'
import {
  AlertCircle,
  Calendar,
  CheckCircle,
  Clock,
  History,
  MoreVertical,
  Pause,
  Play,
  Plus,
  Trash2,
  XCircle,
} from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { ExecutionHistory } from '@/components/templates/ExecutionHistory'
import { ScheduleCreator } from '@/components/templates/ScheduleCreator'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cronToHumanReadable, getRelativeNextRun } from '@/lib/utils/cron'
import { useTemplatesStore } from '@/store/templates-store'
import type { QuerySchedule } from '@/types/templates'

const STATUS_CONFIG = {
  active: {
    label: 'Active',
    icon: CheckCircle,
    className: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  },
  paused: {
    label: 'Paused',
    icon: Pause,
    className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  },
  failed: {
    label: 'Failed',
    icon: XCircle,
    className: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  },
}

export function SchedulesPage() {
  const {
    schedules,
    templates,
    loading,
    error,
    fetchSchedules,
    fetchTemplates,
    pauseSchedule,
    resumeSchedule,
    deleteSchedule,
    runScheduleNow,
  } = useTemplatesStore()

  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [viewingHistory, setViewingHistory] = useState<string | null>(null)

  useEffect(() => {
    fetchSchedules()
    fetchTemplates()
  }, [fetchSchedules, fetchTemplates])

  const handlePauseResume = async (schedule: QuerySchedule) => {
    try {
      if (schedule.status === 'active') {
        await pauseSchedule(schedule.id)
      } else {
        await resumeSchedule(schedule.id)
      }
    } catch (error) {
      console.error('Failed to update schedule:', error)
    }
  }

  const handleDelete = async (schedule: QuerySchedule) => {
    if (confirm(`Are you sure you want to delete "${schedule.name}"?`)) {
      try {
        await deleteSchedule(schedule.id)
      } catch (error) {
        console.error('Failed to delete schedule:', error)
      }
    }
  }

  const handleRunNow = async (schedule: QuerySchedule) => {
    try {
      await runScheduleNow(schedule.id)
      alert(`Schedule "${schedule.name}" is now running`)
    } catch (error) {
      console.error('Failed to run schedule:', error)
    }
  }

  const handleViewHistory = (scheduleId: string) => {
    setViewingHistory(scheduleId)
  }

  // Get template name for a schedule
  const getTemplateName = (templateId: string) => {
    const template = templates.find((t) => t.id === templateId)
    return template?.name || 'Unknown Template'
  }

  const activeCount = schedules.filter((s) => s.status === 'active').length
  const pausedCount = schedules.filter((s) => s.status === 'paused').length
  const failedCount = schedules.filter((s) => s.status === 'failed').length

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b bg-card">
        <div className="container mx-auto px-6 py-8">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold flex items-center gap-3">
                <Calendar className="h-8 w-8" />
                Scheduled Queries
              </h1>
              <p className="text-muted-foreground mt-2">
                Manage automated query execution schedules
              </p>
            </div>

            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Schedule
            </Button>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <StatCard label="Total Schedules" value={schedules.length} />
            <StatCard
              label="Active"
              value={activeCount}
              className="text-green-600 dark:text-green-400"
            />
            <StatCard
              label="Paused"
              value={pausedCount}
              className="text-yellow-600 dark:text-yellow-400"
            />
            <StatCard
              label="Failed"
              value={failedCount}
              className="text-red-600 dark:text-red-400"
            />
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto px-6 py-8">
        {/* Error Alert */}
        {error && (
          <Alert variant="destructive" className="mb-6">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Loading State */}
        {loading && schedules.length === 0 && (
          <div className="grid gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <ScheduleCardSkeleton key={i} />
            ))}
          </div>
        )}

        {/* Empty State */}
        {!loading && schedules.length === 0 && (
          <div className="text-center py-12">
            <Calendar className="h-16 w-16 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No schedules yet</h3>
            <p className="text-muted-foreground mb-6">
              Create your first scheduled query to automate report generation
            </p>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Schedule
            </Button>
          </div>
        )}

        {/* Schedules List */}
        {!loading && schedules.length > 0 && (
          <div className="grid gap-4">
            {schedules.map((schedule) => (
              <ScheduleCard
                key={schedule.id}
                schedule={schedule}
                templateName={getTemplateName(schedule.template_id)}
                onPauseResume={handlePauseResume}
                onDelete={handleDelete}
                onRunNow={handleRunNow}
                onViewHistory={handleViewHistory}
              />
            ))}
          </div>
        )}
      </div>

      {/* Modals */}
      <ScheduleCreator open={showCreateDialog} onClose={() => setShowCreateDialog(false)} />

      {viewingHistory && (
        <ExecutionHistory
          scheduleId={viewingHistory}
          open={!!viewingHistory}
          onClose={() => setViewingHistory(null)}
        />
      )}
    </div>
  )
}

// ============================================================================
// Schedule Card Component
// ============================================================================

interface ScheduleCardProps {
  schedule: QuerySchedule
  templateName: string
  onPauseResume: (schedule: QuerySchedule) => void
  onDelete: (schedule: QuerySchedule) => void
  onRunNow: (schedule: QuerySchedule) => void
  onViewHistory: (scheduleId: string) => void
}

function ScheduleCard({
  schedule,
  templateName,
  onPauseResume,
  onDelete,
  onRunNow,
  onViewHistory,
}: ScheduleCardProps) {
  const statusConfig = STATUS_CONFIG[schedule.status]
  const StatusIcon = statusConfig.icon

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <CardTitle className="flex items-center gap-2">
              {schedule.name}
              <Badge variant="secondary" className={statusConfig.className}>
                <StatusIcon className="h-3 w-3 mr-1" />
                {statusConfig.label}
              </Badge>
            </CardTitle>
            <CardDescription className="mt-1">Template: {templateName}</CardDescription>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onRunNow(schedule)}>
                <Play className="mr-2 h-4 w-4" />
                Run Now
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onPauseResume(schedule)}>
                {schedule.status === 'active' ? (
                  <>
                    <Pause className="mr-2 h-4 w-4" />
                    Pause
                  </>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    Resume
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onViewHistory(schedule.id)}>
                <History className="mr-2 h-4 w-4" />
                View History
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => onDelete(schedule)} className="text-red-600">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-muted-foreground mb-1">Schedule</p>
            <p className="font-medium">{cronToHumanReadable(schedule.frequency)}</p>
          </div>

          <div>
            <p className="text-muted-foreground mb-1">Next Run</p>
            <p className="font-medium flex items-center gap-1">
              <Clock className="h-4 w-4" />
              {schedule.next_run_at
                ? getRelativeNextRun(schedule.next_run_at)
                : 'Not scheduled'}
            </p>
          </div>

          <div>
            <p className="text-muted-foreground mb-1">Last Run</p>
            <p className="font-medium">
              {schedule.last_run_at
                ? formatDistanceToNow(new Date(schedule.last_run_at), { addSuffix: true })
                : 'Never'}
            </p>
          </div>
        </div>

        {schedule.notification_email && (
          <div className="mt-4 text-xs text-muted-foreground">
            Notifications: {schedule.notification_email}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Utility Components
// ============================================================================

function StatCard({
  label,
  value,
  className,
}: {
  label: string
  value: number
  className?: string
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="text-sm text-muted-foreground">{label}</div>
        <div className={`text-3xl font-bold mt-2 ${className || ''}`}>{value}</div>
      </CardContent>
    </Card>
  )
}

function ScheduleCardSkeleton() {
  return (
    <Card>
      <CardHeader>
        <div className="space-y-2">
          <div className="h-6 bg-muted rounded animate-pulse w-2/3" />
          <div className="h-4 bg-muted rounded animate-pulse w-1/3" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="h-20 bg-muted rounded animate-pulse" />
      </CardContent>
    </Card>
  )
}
