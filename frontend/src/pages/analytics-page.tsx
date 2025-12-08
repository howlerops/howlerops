import { useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  CheckCircle,
  Clock,
  Database,
  Minus,
  RefreshCw,
  XCircle,
  Zap
} from 'lucide-react'
import { useState } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import {
  GetConnectionCount,
  GetConnectionIDs,
  GetConnectionStats,
  HealthCheckAll
} from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

// Types for connection pool stats (from Go backend)
interface PoolStats {
  open_connections: number
  in_use: number
  idle: number
  wait_count: number
  wait_duration: number
  max_idle_closed: number
  max_idle_time_closed: number
  max_lifetime_closed: number
}

// Analytics data aggregated from real APIs
interface AnalyticsData {
  connectionCount: number
  connectionIds: string[]
  healthStatuses: Record<string, main.HealthStatus>
  poolStats: Record<string, PoolStats>
  summary: {
    totalConnections: number
    healthyConnections: number
    unhealthyConnections: number
    totalOpenConnections: number
    totalInUse: number
    totalIdle: number
    avgResponseTime: number
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
          <Progress value={maxValue > 0 ? (item.value / maxValue) * 100 : 0} className="h-2" />
        </div>
      ))}
    </div>
  )
}

// Format milliseconds to readable time
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}m`
}

// Fetch real analytics data from Wails APIs
async function fetchAnalyticsData(): Promise<AnalyticsData> {
  const [connectionCount, connectionIds, healthStatuses, poolStats] = await Promise.all([
    GetConnectionCount(),
    GetConnectionIDs(),
    HealthCheckAll(),
    GetConnectionStats()
  ])

  // Calculate summary stats from real data
  let healthyConnections = 0
  let unhealthyConnections = 0
  let totalResponseTime = 0
  let responseTimeCount = 0

  Object.values(healthStatuses).forEach((status: main.HealthStatus) => {
    if (status.status === 'healthy' || status.status === 'ok') {
      healthyConnections++
    } else {
      unhealthyConnections++
    }
    if (status.response_time > 0) {
      totalResponseTime += status.response_time
      responseTimeCount++
    }
  })

  let totalOpenConnections = 0
  let totalInUse = 0
  let totalIdle = 0

  Object.values(poolStats as Record<string, PoolStats>).forEach((stats: PoolStats) => {
    totalOpenConnections += stats.open_connections || 0
    totalInUse += stats.in_use || 0
    totalIdle += stats.idle || 0
  })

  return {
    connectionCount,
    connectionIds,
    healthStatuses,
    poolStats: poolStats as Record<string, PoolStats>,
    summary: {
      totalConnections: connectionCount,
      healthyConnections,
      unhealthyConnections,
      totalOpenConnections,
      totalInUse,
      totalIdle,
      avgResponseTime: responseTimeCount > 0 ? totalResponseTime / responseTimeCount : 0
    }
  }
}

// Main Analytics Page Component
export default function AnalyticsPage() {
  const [autoRefresh, setAutoRefresh] = useState(false)

  // Fetch real analytics data from Wails APIs
  const { data, isLoading, error, refetch } = useQuery<AnalyticsData>({
    queryKey: ['analytics'],
    queryFn: fetchAnalyticsData,
    refetchInterval: autoRefresh ? 10000 : false, // Refresh every 10s if enabled
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

  // Prepare data for pool stats chart
  const poolStatsData = [
    { label: 'Open Connections', value: data.summary.totalOpenConnections },
    { label: 'In Use', value: data.summary.totalInUse },
    { label: 'Idle', value: data.summary.totalIdle },
  ]
  const maxPoolStat = Math.max(...poolStatsData.map(d => d.value), 1)

  // Prepare health distribution data
  const healthData = [
    { label: 'Healthy', value: data.summary.healthyConnections },
    { label: 'Unhealthy', value: data.summary.unhealthyConnections },
  ]
  const maxHealthStat = Math.max(...healthData.map(d => d.value), 1)

  // Calculate health percentage
  const healthPercentage = data.summary.totalConnections > 0
    ? (data.summary.healthyConnections / data.summary.totalConnections) * 100
    : 0

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <div className="flex-1 space-y-6 overflow-y-auto p-6">
        {/* Header */}
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold">Analytics Dashboard</h1>
          <div className="flex items-center gap-4">
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
          </div>
        </div>

        {/* Overview Stats */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            title="Total Connections"
            value={data.summary.totalConnections}
            icon={Database}
            format={(v) => Number(v).toLocaleString()}
          />
          <StatCard
            title="Healthy Connections"
            value={data.summary.healthyConnections}
            trend={data.summary.healthyConnections === data.summary.totalConnections ? 'up' : 'neutral'}
            icon={CheckCircle}
            format={(v) => Number(v).toLocaleString()}
          />
          <StatCard
            title="Avg Response Time"
            value={data.summary.avgResponseTime}
            icon={Clock}
            format={(v) => formatDuration(Number(v))}
          />
          <StatCard
            title="Health Rate"
            value={healthPercentage}
            trend={healthPercentage >= 90 ? 'up' : healthPercentage >= 50 ? 'neutral' : 'down'}
            icon={Zap}
            format={(v) => `${Number(v).toFixed(1)}%`}
          />
        </div>

        <Tabs defaultValue="overview" className="space-y-4">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="connections">Connections</TabsTrigger>
            <TabsTrigger value="pool">Connection Pool</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-4">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Connection Pool Distribution */}
              <Card>
                <CardHeader>
                  <CardTitle>Connection Pool</CardTitle>
                  <CardDescription>Current connection pool utilization</CardDescription>
                </CardHeader>
                <CardContent>
                  <SimpleBarChart data={poolStatsData} maxValue={maxPoolStat} />
                </CardContent>
              </Card>

              {/* Health Distribution */}
              <Card>
                <CardHeader>
                  <CardTitle>Connection Health</CardTitle>
                  <CardDescription>Health status distribution</CardDescription>
                </CardHeader>
                <CardContent>
                  <SimpleBarChart data={healthData} maxValue={maxHealthStat} />
                </CardContent>
              </Card>
            </div>

            {/* Key Metrics */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Open Connections</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{data.summary.totalOpenConnections}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Active (In Use)</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-blue-600">{data.summary.totalInUse}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Idle Connections</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-600">{data.summary.totalIdle}</div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Connections Tab */}
          <TabsContent value="connections" className="space-y-4">
            {data.connectionIds.length === 0 ? (
              <Alert>
                <Database className="h-4 w-4" />
                <AlertTitle>No Connections</AlertTitle>
                <AlertDescription>
                  No database connections are currently configured. Add a connection to see analytics.
                </AlertDescription>
              </Alert>
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle>Connection Health Status</CardTitle>
                  <CardDescription>Real-time health status of all database connections</CardDescription>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Connection ID</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="text-right">Response Time</TableHead>
                        <TableHead>Last Check</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.connectionIds.map((connectionId) => {
                        const health = data.healthStatuses[connectionId]
                        const isHealthy = health?.status === 'healthy' || health?.status === 'ok'
                        return (
                          <TableRow key={connectionId}>
                            <TableCell className="font-mono text-sm">
                              {connectionId}
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                {isHealthy ? (
                                  <CheckCircle className="h-4 w-4 text-green-600" />
                                ) : (
                                  <XCircle className="h-4 w-4 text-red-600" />
                                )}
                                <Badge variant={isHealthy ? "default" : "destructive"}>
                                  {health?.status || 'unknown'}
                                </Badge>
                              </div>
                            </TableCell>
                            <TableCell className="text-right">
                              {health?.response_time ? formatDuration(health.response_time) : '-'}
                            </TableCell>
                            <TableCell className="text-muted-foreground text-sm">
                              {health?.timestamp ? new Date(health.timestamp).toLocaleTimeString() : '-'}
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          {/* Pool Stats Tab */}
          <TabsContent value="pool" className="space-y-4">
            {Object.keys(data.poolStats).length === 0 ? (
              <Alert>
                <Database className="h-4 w-4" />
                <AlertTitle>No Pool Statistics</AlertTitle>
                <AlertDescription>
                  No connection pool statistics are available. Connect to a database to see pool metrics.
                </AlertDescription>
              </Alert>
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle>Connection Pool Statistics</CardTitle>
                  <CardDescription>Detailed pool metrics per connection</CardDescription>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Connection</TableHead>
                        <TableHead className="text-right">Open</TableHead>
                        <TableHead className="text-right">In Use</TableHead>
                        <TableHead className="text-right">Idle</TableHead>
                        <TableHead className="text-right">Wait Count</TableHead>
                        <TableHead className="text-right">Wait Duration</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {Object.entries(data.poolStats).map(([connectionId, stats]) => (
                        <TableRow key={connectionId}>
                          <TableCell className="font-mono text-sm">
                            {connectionId}
                          </TableCell>
                          <TableCell className="text-right">
                            {stats.open_connections}
                          </TableCell>
                          <TableCell className="text-right">
                            <Badge variant={stats.in_use > 0 ? "default" : "secondary"}>
                              {stats.in_use}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right">
                            {stats.idle}
                          </TableCell>
                          <TableCell className="text-right">
                            {stats.wait_count}
                          </TableCell>
                          <TableCell className="text-right">
                            {stats.wait_duration > 0 ? formatDuration(stats.wait_duration / 1000000) : '-'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )}

            {/* Pool Summary */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Total Open</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{data.summary.totalOpenConnections}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Total In Use</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-blue-600">{data.summary.totalInUse}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Total Idle</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-600">{data.summary.totalIdle}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Utilization</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {data.summary.totalOpenConnections > 0
                      ? `${((data.summary.totalInUse / data.summary.totalOpenConnections) * 100).toFixed(1)}%`
                      : '0%'}
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
