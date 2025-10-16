/**
 * Connection Status Indicator - Shows WebSocket connection health
 * Displays connection status, metrics, and health information
 */

import React, { useState } from 'react';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import { Progress } from '../ui/progress';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '../ui/popover';
import {
  Wifi,
  WifiOff,
  AlertTriangle,
  RefreshCw,
  Activity,
  Clock,
  Signal,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { useConnectionStatus } from '../../hooks/websocket';
import { ConnectionStatus } from '../../types/websocket';

interface ConnectionStatusIndicatorProps {
  showDetails?: boolean;
  className?: string;
}

export function ConnectionStatusIndicator({
  showDetails = false,
  className = '',
}: ConnectionStatusIndicatorProps) {
  const {
    status,
    metrics,
    health,
    networkQuality,
    performHealthCheck,
    formatDuration,
    isHealthy,
    isDegraded,
    isUnhealthy, // eslint-disable-line @typescript-eslint/no-unused-vars
    hasIssues,
  } = useConnectionStatus();

  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isPerformingHealthCheck, setIsPerformingHealthCheck] = useState(false);

  /**
   * Get status icon and color
   */
  const getStatusDisplay = (connectionStatus: ConnectionStatus) => {
    switch (connectionStatus) {
      case 'connected':
        return {
          icon: <Wifi className="h-4 w-4" />,
          color: isHealthy ? 'text-primary' : isDegraded ? 'text-accent-foreground' : 'text-destructive',
          bgColor: isHealthy ? 'bg-primary/10' : isDegraded ? 'bg-accent/10' : 'bg-destructive/10',
          text: 'Connected',
        };
      case 'connecting':
        return {
          icon: <RefreshCw className="h-4 w-4 animate-spin" />,
          color: 'text-primary',
          bgColor: 'bg-primary/10',
          text: 'Connecting',
        };
      case 'reconnecting':
        return {
          icon: <RefreshCw className="h-4 w-4 animate-spin" />,
          color: 'text-accent-foreground',
          bgColor: 'bg-accent/10',
          text: 'Reconnecting',
        };
      case 'disconnected':
        return {
          icon: <WifiOff className="h-4 w-4" />,
          color: 'text-muted-foreground',
          bgColor: 'bg-muted',
          text: 'Disconnected',
        };
      case 'error':
        return {
          icon: <AlertTriangle className="h-4 w-4" />,
          color: 'text-destructive',
          bgColor: 'bg-destructive/10',
          text: 'Error',
        };
      default:
        return {
          icon: <WifiOff className="h-4 w-4" />,
          color: 'text-muted-foreground',
          bgColor: 'bg-muted',
          text: 'Unknown',
        };
    }
  };

  /**
   * Get network quality indicator
   */
  const getNetworkQualityColor = (level: string) => {
    switch (level) {
      case 'excellent':
        return 'text-primary';
      case 'good':
        return 'text-primary';
      case 'fair':
        return 'text-accent-foreground';
      case 'poor':
        return 'text-destructive';
      default:
        return 'text-muted-foreground';
    }
  };

  /**
   * Handle health check
   */
  const handleHealthCheck = async () => {
    setIsPerformingHealthCheck(true);
    try {
      await performHealthCheck();
    } finally {
      setIsPerformingHealthCheck(false);
    }
  };

  const statusDisplay = getStatusDisplay(status);

  return (
    <TooltipProvider>
      <Popover open={isPopoverOpen} onOpenChange={setIsPopoverOpen}>
        <PopoverTrigger asChild>
          <div className={`cursor-pointer ${className}`}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge
                  variant="outline"
                  className={`${statusDisplay.bgColor} ${statusDisplay.color} border-current hover:opacity-80 transition-opacity`}
                >
                  {statusDisplay.icon}
                  {showDetails && (
                    <span className="ml-2">{statusDisplay.text}</span>
                  )}
                  {hasIssues && (
                    <AlertTriangle className="h-3 w-3 ml-1" />
                  )}
                </Badge>
              </TooltipTrigger>
              <TooltipContent>
                <div className="text-xs">
                  <div>Status: {statusDisplay.text}</div>
                  {metrics.latency && (
                    <div>Latency: {Math.round(metrics.latency)}ms</div>
                  )}
                  <div>Health: {health.overall}</div>
                </div>
              </TooltipContent>
            </Tooltip>
          </div>
        </PopoverTrigger>

        <PopoverContent className="w-80" align="end">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="font-medium flex items-center gap-2">
                <Activity className="h-4 w-4" />
                Connection Status
              </h4>
              <Button
                variant="outline"
                size="sm"
                onClick={handleHealthCheck}
                disabled={isPerformingHealthCheck}
                className="h-6 text-xs"
              >
                {isPerformingHealthCheck ? (
                  <RefreshCw className="h-3 w-3 animate-spin" />
                ) : (
                  'Check Health'
                )}
              </Button>
            </div>

            {/* Status Overview */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">Connection</div>
                <div className={`flex items-center gap-2 ${statusDisplay.color}`}>
                  {statusDisplay.icon}
                  <span className="font-medium">{statusDisplay.text}</span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">Health Score</div>
                <div className="flex items-center gap-2">
                  <Progress value={health.score} className="flex-1 h-2" />
                  <span className="text-sm font-medium">{health.score}/100</span>
                </div>
              </div>
            </div>

            {/* Network Quality */}
            <div className="space-y-2">
              <div className="text-xs text-muted-foreground">Network Quality</div>
              <div className="flex items-center gap-2">
                <Signal className={`h-4 w-4 ${getNetworkQualityColor(networkQuality.level)}`} />
                <span className={`font-medium ${getNetworkQualityColor(networkQuality.level)}`}>
                  {networkQuality.level.charAt(0).toUpperCase() + networkQuality.level.slice(1)}
                </span>
                {metrics.latency && (
                  <span className="text-xs text-muted-foreground">
                    ({Math.round(metrics.latency)}ms)
                  </span>
                )}
              </div>
              <div className="text-xs text-muted-foreground">
                {networkQuality.description}
              </div>
            </div>

            {/* Metrics */}
            <div className="space-y-3">
              <div className="text-xs text-muted-foreground">Metrics</div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="space-y-1">
                  <div className="text-muted-foreground">Uptime</div>
                  <div className="font-medium">
                    {metrics.uptime ? formatDuration(metrics.uptime) : 'N/A'}
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-muted-foreground">Stability</div>
                  <div className="font-medium">
                    {Math.round(metrics.connectionStability * 100)}%
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-muted-foreground">Reconnects</div>
                  <div className="font-medium">
                    {metrics.totalReconnects}
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-muted-foreground">Messages</div>
                  <div className="font-medium">
                    {metrics.messagesSent + metrics.messagesReceived}
                  </div>
                </div>
              </div>

              {metrics.lastConnectedAt && (
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  <span>
                    Last connected: {formatDuration(Date.now() - metrics.lastConnectedAt)} ago
                  </span>
                </div>
              )}
            </div>

            {/* Health Issues */}
            {health.issues.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">Issues</div>
                <div className="space-y-1">
                  {health.issues.map((issue, index) => (
                    <div key={index} className="flex items-start gap-2 text-xs">
                      <XCircle className="h-3 w-3 text-destructive mt-0.5 flex-shrink-0" />
                      <span className="text-destructive">{issue}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Recommendations */}
            {health.recommendations.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground">Recommendations</div>
                <div className="space-y-1">
                  {health.recommendations.map((recommendation, index) => (
                    <div key={index} className="flex items-start gap-2 text-xs">
                      <CheckCircle className="h-3 w-3 text-primary mt-0.5 flex-shrink-0" />
                      <span className="text-primary">{recommendation}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Connection Attempts */}
            {metrics.reconnectAttempts > 0 && (
              <div className="bg-accent/10 border border-accent rounded p-2">
                <div className="text-xs text-accent-foreground">
                  Reconnection attempt {metrics.reconnectAttempts}
                </div>
              </div>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  );
}