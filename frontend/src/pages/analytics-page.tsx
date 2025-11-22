import { useState } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Progress } from '@/components/ui/progress'
import {
  AlertTriangle,
  Database,
  Users,
  Clock,
  Zap,
  CheckCircle,
  XCircle,
  AlertCircle,
  ArrowUp,
  ArrowDown,
  Minus,
  RefreshCw,
  Download
} from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { cn } from '@/lib/utils'

// Types
interface DashboardData {
  time_range: {
    start: string
    end: string
  }
  overview: {
    total_queries: number
    total_users: number
    total_connections: number
    active_users_24h: number
    query_success_rate: number
    avg_response_time_ms: number
  }
  query_stats: {
    top_queries: Array<{
      sql_hash: string
      sql: string
      execution_count: number
      avg_time_ms: number
      success_rate: number
    }>
    slow_queries: Array<{
      id: string
      sql: string
      execution_time_ms: number
      user_id: string
      status: string
      executed_at: string
    }>
    query_types: Record<string, number>
    error_queries: Array<{
      id: string
      sql: string
      error_message: string
      executed_at: string
    }>
  }
  user_activity: {
    query_count_by_user: Array<{
      user_id: string
      user_email?: string
      query_count: number
    }>
    peak_hours: Array<{
      hour: number
      query_count: number
      avg_time_ms: number
    }>
  }
  performance: {
    avg_query_time_ms: number
    p50_query_time_ms: number
    p95_query_time_ms: number
    p99_query_time_ms: number
    error_rate: number
    timeout_rate: number
    queries_per_second: number
  }
}

// Stat Card Component
function StatCard({
  title,
  value,
  change,
  trend,
  icon: Icon,
  format: formatValue = (v) => v.toString(),
}: {
  title: string
  value: number | string
  change?: string
  trend?: 'up' | 'down' | 'neutral'
  icon?: React.ElementType
  format?: (value: number | string) => string
}) {
  const TrendIcon = trend === 'up' ? ArrowUp : trend === 'down' ? ArrowDown : Minus

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{formatValue(value)}</div>
        {change && (
          <div className="flex items-center mt-1">
            <TrendIcon
              className={cn(
                "h-3 w-3 mr-1",
                trend === 'up' && 'text-green-600',
                trend === 'down' && 'text-red-600',
                trend === 'neutral' && 'text-gray-400'
              )}
            />
            <p className={cn(
              "text-xs",
              trend === 'up' && 'text-green-600',
              trend === 'down' && 'text-red-600',
              trend === 'neutral' && 'text-gray-400'
            )}>
              {change}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Simple Bar Chart Component
function SimpleBarChart({ data, maxValue }: { data: { label: string; value: number }[]; maxValue: number }) {
  return (
    <div className="space-y-2">
      {data.map((item) => (
        <div key={item.label}>
          <div className="flex justify-between mb-1">
            <span className="text-sm font-medium">{item.label}</span>
            <span className="text-sm text-muted-foreground">{item.value}</span>
          </div>
          <Progress value={(item.value / maxValue) * 100} className="h-2" />
        </div>
      ))}
    </div>
  )
}

// Truncate SQL query for display
function truncateSQL(sql: string, maxLength: number = 100): string {
  if (sql.length <= maxLength) return sql
  return sql.substring(0, maxLength) + '...'
}

// Format milliseconds to readable time
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}m`
}

// Format relative time
function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  return `${days}d ago`
}

// Mock data generator for demo purposes
function generateMockData(): DashboardData {
  return {
    time_range: {
      start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
      end: new Date().toISOString()
    },
    overview: {
      total_queries: 15234,
      total_users: 42,
      total_connections: 8,
      active_users_24h: 23,
      query_success_rate: 98.5,
      avg_response_time_ms: 145
    },
    query_stats: {
      top_queries: [
        { sql_hash: '1', sql: 'SELECT * FROM users WHERE active = true', execution_count: 523, avg_time_ms: 42, success_rate: 100 },
        { sql_hash: '2', sql: 'SELECT id, name FROM products ORDER BY created_at DESC', execution_count: 412, avg_time_ms: 78, success_rate: 99.5 },
        { sql_hash: '3', sql: 'INSERT INTO logs (action, user_id) VALUES (?, ?)', execution_count: 387, avg_time_ms: 15, success_rate: 100 }
      ],
      slow_queries: [
        { id: '1', sql: 'SELECT * FROM orders JOIN customers ON ...', execution_time_ms: 2340, user_id: 'user1', status: 'success', executed_at: new Date(Date.now() - 3600000).toISOString() },
        { id: '2', sql: 'UPDATE inventory SET quantity = quantity - 1 WHERE ...', execution_time_ms: 1890, user_id: 'user2', status: 'success', executed_at: new Date(Date.now() - 7200000).toISOString() }
      ],
      query_types: {
        'SELECT': 8234,
        'INSERT': 3421,
        'UPDATE': 2156,
        'DELETE': 423,
        'OTHER': 1000
      },
      error_queries: [
        { id: '1', sql: 'SELECT * FROM non_existent_table', error_message: 'Table not found', executed_at: new Date(Date.now() - 1800000).toISOString() }
      ]
    },
    user_activity: {
      query_count_by_user: [
        { user_id: 'user1', user_email: 'alice@example.com', query_count: 3421 },
        { user_id: 'user2', user_email: 'bob@example.com', query_count: 2156 },
        { user_id: 'user3', user_email: 'charlie@example.com', query_count: 1890 }
      ],
      peak_hours: [
        { hour: 9, query_count: 1523, avg_time_ms: 125 },
        { hour: 14, query_count: 2156, avg_time_ms: 142 },
        { hour: 16, query_count: 1890, avg_time_ms: 156 }
      ]
    },
    performance: {
      avg_query_time_ms: 145,
      p50_query_time_ms: 89,
      p95_query_time_ms: 423,
      p99_query_time_ms: 1245,
      error_rate: 1.5,
      timeout_rate: 0.2,
      queries_per_second: 2.5
    }
  }
}

// Main Analytics Page Component
export default function AnalyticsPage() {
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d'>('7d')
  const [autoRefresh, setAutoRefresh] = useState(false)

  // Fetch analytics data (using mock data for now)
  const { data, isLoading, error, refetch } = useQuery<DashboardData>({
    queryKey: ['analytics', timeRange],
    queryFn: async () => {
      try {
        const response = await fetch(`/api/analytics/dashboard?range=${timeRange}`)
        if (!response.ok) throw new Error('Failed to fetch analytics')
        return response.json()
      } catch {
        // Return mock data if API is not available
        return generateMockData()
      }
    },
    refetchInterval: autoRefresh ? 30000 : false, // Refresh every 30s if enabled
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive" className="m-6">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>Failed to load analytics data. Please try again.</AlertDescription>
      </Alert>
    )
  }

  if (!data) return null

  // Prepare data for simple charts
  const queryTypesData = Object.entries(data.query_stats.query_types || {})
    .map(([label, value]) => ({ label, value }))
    .sort((a, b) => b.value - a.value)

  const maxQueryType = Math.max(...queryTypesData.map(d => d.value))

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">Analytics Dashboard</h1>
        <div className="flex items-center gap-4">
          <Select value={timeRange} onValueChange={(v) => setTimeRange(v as '24h' | '7d' | '30d')}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="24h">Last 24h</SelectItem>
              <SelectItem value="7d">Last 7 days</SelectItem>
              <SelectItem value="30d">Last 30 days</SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant={autoRefresh ? 'default' : 'outline'}
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
          >
            <RefreshCw className={cn("h-4 w-4 mr-2", autoRefresh && "animate-spin")} />
            Auto Refresh
          </Button>

          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>

          <Button variant="outline" size="sm">
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      {/* Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Queries"
          value={data.overview.total_queries}
          change="+12.5%"
          trend="up"
          icon={Database}
          format={(v) => Number(v).toLocaleString()}
        />
        <StatCard
          title="Active Users (24h)"
          value={data.overview.active_users_24h}
          change="+5.2%"
          trend="up"
          icon={Users}
          format={(v) => Number(v).toLocaleString()}
        />
        <StatCard
          title="Avg Query Time"
          value={data.performance.avg_query_time_ms}
          change="-8.3%"
          trend="down"
          icon={Clock}
          format={(v) => formatDuration(Number(v))}
        />
        <StatCard
          title="Success Rate"
          value={data.overview.query_success_rate}
          change="+2.1%"
          trend="up"
          icon={CheckCircle}
          format={(v) => `${Number(v).toFixed(1)}%`}
        />
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="queries">Queries</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Query Types Distribution */}
            <Card>
              <CardHeader>
                <CardTitle>Query Types</CardTitle>
                <CardDescription>Distribution of query operations</CardDescription>
              </CardHeader>
              <CardContent>
                <SimpleBarChart data={queryTypesData} maxValue={maxQueryType} />
              </CardContent>
            </Card>

            {/* Performance Metrics */}
            <Card>
              <CardHeader>
                <CardTitle>Performance Percentiles</CardTitle>
                <CardDescription>Query execution time distribution</CardDescription>
              </CardHeader>
              <CardContent>
                <SimpleBarChart
                  data={[
                    { label: 'P50 (Median)', value: data.performance.p50_query_time_ms },
                    { label: 'P95', value: data.performance.p95_query_time_ms },
                    { label: 'P99', value: data.performance.p99_query_time_ms },
                  ]}
                  maxValue={data.performance.p99_query_time_ms}
                />
              </CardContent>
            </Card>
          </div>

          {/* Key Metrics */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Throughput</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{data.performance.queries_per_second.toFixed(2)} q/s</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Error Rate</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-red-600">{data.performance.error_rate.toFixed(2)}%</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Total Connections</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{data.overview.total_connections}</div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Queries Tab */}
        <TabsContent value="queries" className="space-y-4">
          {/* Top Queries */}
          <Card>
            <CardHeader>
              <CardTitle>Top Queries</CardTitle>
              <CardDescription>Most frequently executed queries</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Query</TableHead>
                    <TableHead className="text-right">Executions</TableHead>
                    <TableHead className="text-right">Avg Time</TableHead>
                    <TableHead className="text-right">Success Rate</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.query_stats.top_queries.map((query) => (
                    <TableRow key={query.sql_hash}>
                      <TableCell className="font-mono text-xs">
                        {truncateSQL(query.sql)}
                      </TableCell>
                      <TableCell className="text-right">
                        {query.execution_count.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatDuration(query.avg_time_ms)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Badge variant={query.success_rate > 95 ? "default" : "destructive"}>
                          {query.success_rate.toFixed(1)}%
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Slow Queries Alert */}
          {data.query_stats.slow_queries.length > 0 && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>Slow Queries Detected</AlertTitle>
              <AlertDescription>
                {data.query_stats.slow_queries.length} queries are taking longer than 1 second.
              </AlertDescription>
            </Alert>
          )}

          {/* Slow Queries Table */}
          <Card>
            <CardHeader>
              <CardTitle>Slow Queries</CardTitle>
              <CardDescription>Queries taking more than 1 second</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Query</TableHead>
                    <TableHead className="text-right">Execution Time</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Executed</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.query_stats.slow_queries.slice(0, 10).map((query) => (
                    <TableRow key={query.id}>
                      <TableCell className="font-mono text-xs">
                        {truncateSQL(query.sql)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Badge variant="destructive">
                          {formatDuration(query.execution_time_ms)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={query.status === 'success' ? 'default' : 'destructive'}>
                          {query.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatRelativeTime(query.executed_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Performance Tab */}
        <TabsContent value="performance" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <StatCard
              title="Throughput"
              value={data.performance.queries_per_second}
              icon={Zap}
              format={(v) => `${Number(v).toFixed(2)} q/s`}
            />
            <StatCard
              title="Error Rate"
              value={data.performance.error_rate}
              trend={data.performance.error_rate > 5 ? 'up' : 'neutral'}
              icon={XCircle}
              format={(v) => `${Number(v).toFixed(2)}%`}
            />
            <StatCard
              title="Timeout Rate"
              value={data.performance.timeout_rate}
              trend={data.performance.timeout_rate > 1 ? 'up' : 'neutral'}
              icon={AlertCircle}
              format={(v) => `${Number(v).toFixed(2)}%`}
            />
          </div>

          {/* Performance Summary */}
          <Card>
            <CardHeader>
              <CardTitle>Performance Summary</CardTitle>
              <CardDescription>Key performance indicators</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">Average Query Time</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDuration(data.performance.avg_query_time_ms)}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">P50 (Median)</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDuration(data.performance.p50_query_time_ms)}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">P95</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDuration(data.performance.p95_query_time_ms)}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">P99</span>
                  <span className="text-sm text-muted-foreground">
                    {formatDuration(data.performance.p99_query_time_ms)}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Users Tab */}
        <TabsContent value="users" className="space-y-4">
          {/* Top Users */}
          <Card>
            <CardHeader>
              <CardTitle>Top Users by Activity</CardTitle>
              <CardDescription>Most active users by query count</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead className="text-right">Queries</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.user_activity.query_count_by_user.slice(0, 10).map((user) => (
                    <TableRow key={user.user_id}>
                      <TableCell>
                        {user.user_email || user.user_id}
                      </TableCell>
                      <TableCell className="text-right">
                        {user.query_count.toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Peak Hours */}
          <Card>
            <CardHeader>
              <CardTitle>Peak Usage Hours</CardTitle>
              <CardDescription>Query activity by hour of day</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {data.user_activity.peak_hours.map((hour) => (
                  <div key={hour.hour}>
                    <div className="flex justify-between mb-1">
                      <span className="text-sm font-medium">{hour.hour}:00</span>
                      <span className="text-sm text-muted-foreground">{hour.query_count} queries</span>
                    </div>
                    <Progress value={(hour.query_count / Math.max(...data.user_activity.peak_hours.map(h => h.query_count))) * 100} className="h-2" />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}