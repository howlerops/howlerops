/**
 * AuditLogViewer Component
 *
 * Displays organization audit logs with filtering and export capabilities.
 * Only visible to owners and admins.
 *
 * Features:
 * - Filter by action type, user, date range
 * - Pagination (50 per page)
 * - Expandable rows for IP/user agent details
 * - Auto-refresh every 30 seconds
 * - Export to CSV
 */

import * as React from 'react'
import {
  FileText,
  Filter,
  Download,
  ChevronDown,
  ChevronRight,
  Loader2,
  Calendar,
  User,
  Activity,
  RefreshCw,
  Clock,
} from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import type {
  AuditLog,
  AuditAction,
  ResourceType,
  OrganizationMember,
} from '@/types/organization'
import { formatRelativeTime } from '@/types/organization'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

interface AuditLogViewerProps {
  organizationId: string
  members: OrganizationMember[]
  onFetchLogs: (params: {
    action?: AuditAction
    user_id?: string
    start_date?: Date
    end_date?: Date
    limit?: number
    offset?: number
  }) => Promise<AuditLog[]>
  className?: string
}

const LOGS_PER_PAGE = 50
const AUTO_REFRESH_INTERVAL = 30000 // 30 seconds

/**
 * AuditLogViewer - View and filter organization audit logs
 */
export function AuditLogViewer({
  organizationId,
  members,
  onFetchLogs,
  className,
}: AuditLogViewerProps) {
  const [logs, setLogs] = React.useState<AuditLog[]>([])
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [expandedLogId, setExpandedLogId] = React.useState<string | null>(null)
  const [page, setPage] = React.useState(0)
  const [hasMore, setHasMore] = React.useState(false)
  const [autoRefresh, setAutoRefresh] = React.useState(true)

  // Filters
  const [selectedAction, setSelectedAction] = React.useState<AuditAction | 'all'>(
    'all'
  )
  const [selectedUserId, setSelectedUserId] = React.useState<string>('all')
  const [startDate, setStartDate] = React.useState<string>('')
  const [endDate, setEndDate] = React.useState<string>('')

  // Fetch logs with current filters
  const fetchLogs = React.useCallback(
    async (offset = 0) => {
      setLoading(true)
      setError(null)

      try {
        const params: Parameters<typeof onFetchLogs>[0] = {
          limit: LOGS_PER_PAGE,
          offset,
        }

        if (selectedAction !== 'all') {
          params.action = selectedAction
        }

        if (selectedUserId !== 'all') {
          params.user_id = selectedUserId
        }

        if (startDate) {
          params.start_date = new Date(startDate)
        }

        if (endDate) {
          params.end_date = new Date(endDate)
        }

        const fetchedLogs = await onFetchLogs(params)

        setLogs(fetchedLogs)
        setHasMore(fetchedLogs.length === LOGS_PER_PAGE)
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to fetch audit logs'
        setError(errorMessage)
        toast.error('Failed to load audit logs', {
          description: errorMessage,
        })
      } finally {
        setLoading(false)
      }
    },
    [onFetchLogs, selectedAction, selectedUserId, startDate, endDate]
  )

  // Initial load
  React.useEffect(() => {
    fetchLogs(0)
  }, [fetchLogs])

  // Auto-refresh
  React.useEffect(() => {
    if (!autoRefresh) return

    const interval = setInterval(() => {
      fetchLogs(page * LOGS_PER_PAGE)
    }, AUTO_REFRESH_INTERVAL)

    return () => clearInterval(interval)
  }, [autoRefresh, fetchLogs, page])

  // Handle filter changes
  const handleFilterChange = () => {
    setPage(0)
    fetchLogs(0)
  }

  // Pagination
  const handlePreviousPage = () => {
    const newPage = Math.max(0, page - 1)
    setPage(newPage)
    fetchLogs(newPage * LOGS_PER_PAGE)
  }

  const handleNextPage = () => {
    const newPage = page + 1
    setPage(newPage)
    fetchLogs(newPage * LOGS_PER_PAGE)
  }

  // Export to CSV
  const handleExportCSV = () => {
    try {
      const headers = ['Timestamp', 'User', 'Action', 'Resource Type', 'Resource ID', 'IP Address', 'User Agent']
      const rows = logs.map((log) => {
        const member = members.find((m) => m.user_id === log.user_id)
        const userName = member?.user?.email || log.user_id
        return [
          new Date(log.created_at).toISOString(),
          userName,
          log.action,
          log.resource_type,
          log.resource_id || '',
          log.ip_address || '',
          log.user_agent || '',
        ]
      })

      const csvContent = [
        headers.join(','),
        ...rows.map((row) => row.map((cell) => `"${cell}"`).join(',')),
      ].join('\n')

      const blob = new Blob([csvContent], { type: 'text/csv' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `audit-logs-${organizationId}-${new Date().toISOString()}.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      toast.success('Audit logs exported', {
        description: `${logs.length} log entries exported to CSV`,
      })
    } catch (err) {
      toast.error('Failed to export logs', {
        description: err instanceof Error ? err.message : 'Unknown error',
      })
    }
  }

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Audit Logs
            </CardTitle>
            <CardDescription>
              View organization activity history and security events
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchLogs(page * LOGS_PER_PAGE)}
              disabled={loading}
            >
              <RefreshCw
                className={cn('h-4 w-4', loading && 'animate-spin')}
              />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleExportCSV}
              disabled={logs.length === 0}
            >
              <Download className="h-4 w-4 mr-2" />
              Export CSV
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Filters */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 p-4 rounded-lg border bg-muted/50">
          <div className="space-y-2">
            <Label htmlFor="action-filter" className="text-xs">
              Action Type
            </Label>
            <Select
              value={selectedAction}
              onValueChange={(value) => {
                setSelectedAction(value as AuditAction | 'all')
                handleFilterChange()
              }}
            >
              <SelectTrigger id="action-filter" className="h-9">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Actions</SelectItem>
                <SelectItem value="organization.created">Org Created</SelectItem>
                <SelectItem value="organization.updated">Org Updated</SelectItem>
                <SelectItem value="organization.deleted">Org Deleted</SelectItem>
                <SelectItem value="member.added">Member Added</SelectItem>
                <SelectItem value="member.removed">Member Removed</SelectItem>
                <SelectItem value="member.role_updated">Role Updated</SelectItem>
                <SelectItem value="invitation.created">Invite Created</SelectItem>
                <SelectItem value="invitation.accepted">Invite Accepted</SelectItem>
                <SelectItem value="connection.created">Connection Created</SelectItem>
                <SelectItem value="connection.deleted">Connection Deleted</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="user-filter" className="text-xs">
              User
            </Label>
            <Select
              value={selectedUserId}
              onValueChange={(value) => {
                setSelectedUserId(value)
                handleFilterChange()
              }}
            >
              <SelectTrigger id="user-filter" className="h-9">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Users</SelectItem>
                {members.map((member) => (
                  <SelectItem key={member.user_id} value={member.user_id}>
                    {member.user?.email || member.user?.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="start-date" className="text-xs">
              Start Date
            </Label>
            <Input
              id="start-date"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              onBlur={handleFilterChange}
              className="h-9"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="end-date" className="text-xs">
              End Date
            </Label>
            <Input
              id="end-date"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              onBlur={handleFilterChange}
              className="h-9"
            />
          </div>
        </div>

        {/* Auto-refresh toggle */}
        <div className="flex items-center justify-between text-sm">
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="w-4 h-4"
              />
              <span className="text-muted-foreground">
                Auto-refresh every 30 seconds
              </span>
            </label>
          </div>
          <span className="text-muted-foreground">
            {logs.length} {logs.length === 1 ? 'entry' : 'entries'}
          </span>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Logs Table */}
        {loading && logs.length === 0 ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : logs.length === 0 ? (
          <div className="text-center py-12">
            <Activity className="h-12 w-12 text-muted-foreground mx-auto mb-3" />
            <p className="text-muted-foreground">
              No audit logs found for the selected filters
            </p>
          </div>
        ) : (
          <>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[50px]"></TableHead>
                    <TableHead>Timestamp</TableHead>
                    <TableHead>User</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Resource</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => {
                    const isExpanded = expandedLogId === log.id
                    const member = members.find((m) => m.user_id === log.user_id)

                    return (
                      <React.Fragment key={log.id}>
                        <TableRow
                          className="cursor-pointer hover:bg-muted/50"
                          onClick={() =>
                            setExpandedLogId(isExpanded ? null : log.id)
                          }
                        >
                          <TableCell>
                            {isExpanded ? (
                              <ChevronDown className="h-4 w-4 text-muted-foreground" />
                            ) : (
                              <ChevronRight className="h-4 w-4 text-muted-foreground" />
                            )}
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            <div className="flex items-center gap-2">
                              <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                              {formatRelativeTime(new Date(log.created_at))}
                            </div>
                            <div className="text-xs text-muted-foreground mt-1">
                              {new Date(log.created_at).toLocaleString()}
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <User className="h-3.5 w-3.5 text-muted-foreground" />
                              {member?.user?.email || log.user_id}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline" className="font-mono text-xs">
                              {log.action}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <div>
                              <Badge variant="secondary" className="text-xs">
                                {log.resource_type}
                              </Badge>
                              {log.resource_id && (
                                <div className="text-xs text-muted-foreground mt-1 font-mono">
                                  {log.resource_id}
                                </div>
                              )}
                            </div>
                          </TableCell>
                        </TableRow>

                        {/* Expanded Details */}
                        {isExpanded && (
                          <TableRow>
                            <TableCell colSpan={5} className="bg-muted/30">
                              <div className="p-4 space-y-3">
                                <h4 className="font-semibold text-sm">Details</h4>
                                <div className="grid grid-cols-2 gap-4 text-sm">
                                  {log.ip_address && (
                                    <div>
                                      <span className="text-muted-foreground">
                                        IP Address:
                                      </span>{' '}
                                      <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                                        {log.ip_address}
                                      </code>
                                    </div>
                                  )}
                                  {log.user_agent && (
                                    <div className="col-span-2">
                                      <span className="text-muted-foreground">
                                        User Agent:
                                      </span>{' '}
                                      <code className="text-xs bg-muted px-1.5 py-0.5 rounded break-all">
                                        {log.user_agent}
                                      </code>
                                    </div>
                                  )}
                                  {log.details && (
                                    <div className="col-span-2">
                                      <span className="text-muted-foreground">
                                        Additional Details:
                                      </span>
                                      <pre className="text-xs bg-muted p-2 rounded mt-1 overflow-auto max-h-[200px]">
                                        {JSON.stringify(log.details, null, 2)}
                                      </pre>
                                    </div>
                                  )}
                                </div>
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </React.Fragment>
                    )
                  })}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                Page {page + 1} {hasMore && '(more available)'}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePreviousPage}
                  disabled={page === 0 || loading}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!hasMore || loading}
                >
                  Next
                </Button>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
