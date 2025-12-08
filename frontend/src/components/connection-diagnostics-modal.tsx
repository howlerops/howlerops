import { Activity, AlertCircle, CheckCircle2, Clock, Database, Loader2, RefreshCw, Server, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'
import { toast } from '@/hooks/use-toast'
import type { DatabaseConnection } from '@/store/connection-store'
import {
  GetConnectionHealth,
  GetConnectionStats,
  GetDatabaseVersion,
} from '../../wailsjs/go/main/App'

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

interface HealthStatus {
  status: string
  message?: string
  response_time: number
  timestamp: string
}

interface DiagnosticsData {
  health: HealthStatus | null
  poolStats: PoolStats | null
  version: string | null
}

interface ConnectionDiagnosticsModalProps {
  connection: DatabaseConnection
  open: boolean
  onClose: () => void
}

function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(2)}m`
}

function formatNanoseconds(ns: number): string {
  const ms = ns / 1_000_000
  return formatDuration(ms)
}

export function ConnectionDiagnosticsModal({
  connection,
  open,
  onClose,
}: ConnectionDiagnosticsModalProps) {
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [data, setData] = useState<DiagnosticsData>({
    health: null,
    poolStats: null,
    version: null,
  })

  const fetchDiagnostics = useCallback(async () => {
    if (!connection.sessionId) {
      setLoading(false)
      return
    }

    try {
      const [healthResult, statsResult, versionResult] = await Promise.allSettled([
        GetConnectionHealth(connection.sessionId),
        GetConnectionStats(),
        GetDatabaseVersion(connection.sessionId),
      ])

      const health = healthResult.status === 'fulfilled' ? healthResult.value as HealthStatus : null
      const allStats = statsResult.status === 'fulfilled' ? statsResult.value as Record<string, PoolStats> : null
      const poolStats = allStats ? allStats[connection.sessionId] : null
      const version = versionResult.status === 'fulfilled' ? versionResult.value : null

      setData({ health, poolStats, version })
    } catch (error) {
      console.error('Failed to fetch diagnostics:', error)
      toast({
        title: 'Failed to load diagnostics',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [connection.sessionId])

  useEffect(() => {
    if (open && connection.sessionId) {
      setLoading(true)
      fetchDiagnostics()
    }
  }, [open, connection.sessionId, fetchDiagnostics])

  const handleRefresh = () => {
    setRefreshing(true)
    fetchDiagnostics()
  }

  const isHealthy = data.health?.status === 'healthy' || data.health?.status === 'ok'
  const poolUtilization = data.poolStats
    ? data.poolStats.open_connections > 0
      ? Math.round((data.poolStats.in_use / data.poolStats.open_connections) * 100)
      : 0
    : 0

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Connection Diagnostics
          </DialogTitle>
          <DialogDescription>
            Performance and health metrics for {connection.name}
          </DialogDescription>
        </DialogHeader>

        {!connection.isConnected || !connection.sessionId ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Database className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-sm text-muted-foreground">
              Connect to this database to view diagnostics
            </p>
          </div>
        ) : loading ? (
          <div className="flex flex-col items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="mt-2 text-sm text-muted-foreground">Loading diagnostics...</p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Health Status */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-medium">Health Status</h4>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleRefresh}
                  disabled={refreshing}
                  className="h-7 px-2"
                >
                  <RefreshCw className={`h-3 w-3 mr-1 ${refreshing ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              </div>

              <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
                {isHealthy ? (
                  <CheckCircle2 className="h-8 w-8 text-green-500" />
                ) : data.health ? (
                  <XCircle className="h-8 w-8 text-destructive" />
                ) : (
                  <AlertCircle className="h-8 w-8 text-yellow-500" />
                )}
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <Badge variant={isHealthy ? 'default' : 'destructive'}>
                      {data.health?.status || 'Unknown'}
                    </Badge>
                    {data.health?.response_time !== undefined && (
                      <span className="text-xs text-muted-foreground flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatDuration(data.health.response_time)}
                      </span>
                    )}
                  </div>
                  {data.health?.message && (
                    <p className="text-xs text-muted-foreground mt-1">{data.health.message}</p>
                  )}
                </div>
              </div>
            </div>

            <Separator />

            {/* Connection Info */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium">Connection Info</h4>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Host:</span>
                  <span className="font-mono text-xs truncate">{connection.host}:{connection.port}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Database className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Database:</span>
                  <span className="font-mono text-xs truncate">{connection.database || 'N/A'}</span>
                </div>
                {data.version && (
                  <div className="col-span-2 flex items-center gap-2">
                    <span className="text-muted-foreground">Version:</span>
                    <span className="font-mono text-xs">{data.version}</span>
                  </div>
                )}
              </div>
            </div>

            {data.poolStats && (
              <>
                <Separator />

                {/* Pool Statistics */}
                <div className="space-y-3">
                  <h4 className="text-sm font-medium">Connection Pool</h4>

                  {/* Pool Utilization */}
                  <div className="space-y-2">
                    <div className="flex justify-between text-xs">
                      <span className="text-muted-foreground">Pool Utilization</span>
                      <span>{poolUtilization}%</span>
                    </div>
                    <Progress value={poolUtilization} className="h-2" />
                  </div>

                  {/* Pool Stats Grid */}
                  <div className="grid grid-cols-3 gap-3">
                    <div className="rounded-lg bg-muted/50 p-3 text-center">
                      <div className="text-2xl font-bold text-primary">
                        {data.poolStats.open_connections}
                      </div>
                      <div className="text-xs text-muted-foreground">Open</div>
                    </div>
                    <div className="rounded-lg bg-muted/50 p-3 text-center">
                      <div className="text-2xl font-bold text-orange-500">
                        {data.poolStats.in_use}
                      </div>
                      <div className="text-xs text-muted-foreground">In Use</div>
                    </div>
                    <div className="rounded-lg bg-muted/50 p-3 text-center">
                      <div className="text-2xl font-bold text-green-500">
                        {data.poolStats.idle}
                      </div>
                      <div className="text-xs text-muted-foreground">Idle</div>
                    </div>
                  </div>

                  {/* Additional Stats */}
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div className="flex justify-between p-2 rounded bg-muted/30">
                      <span className="text-muted-foreground">Wait Count:</span>
                      <span>{data.poolStats.wait_count}</span>
                    </div>
                    <div className="flex justify-between p-2 rounded bg-muted/30">
                      <span className="text-muted-foreground">Wait Duration:</span>
                      <span>{formatNanoseconds(data.poolStats.wait_duration)}</span>
                    </div>
                    <div className="flex justify-between p-2 rounded bg-muted/30">
                      <span className="text-muted-foreground">Max Idle Closed:</span>
                      <span>{data.poolStats.max_idle_closed}</span>
                    </div>
                    <div className="flex justify-between p-2 rounded bg-muted/30">
                      <span className="text-muted-foreground">Max Lifetime Closed:</span>
                      <span>{data.poolStats.max_lifetime_closed}</span>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
