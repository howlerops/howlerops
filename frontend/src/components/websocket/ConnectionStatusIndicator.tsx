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
          color: isHealthy ? 'text-green-500' : isDegraded ? 'text-yellow-500' : 'text-red-500',
          bgColor: isHealthy ? 'bg-green-100' : isDegraded ? 'bg-yellow-100' : 'bg-red-100',
          text: 'Connected',
        };
      case 'connecting':
        return {
          icon: <RefreshCw className="h-4 w-4 animate-spin" />,
          color: 'text-blue-500',
          bgColor: 'bg-blue-100',
          text: 'Connecting',
        };
      case 'reconnecting':
        return {
          icon: <RefreshCw className="h-4 w-4 animate-spin" />,
          color: 'text-orange-500',
          bgColor: 'bg-orange-100',
          text: 'Reconnecting',
        };
      case 'disconnected':
        return {
          icon: <WifiOff className="h-4 w-4" />,
          color: 'text-gray-500',
          bgColor: 'bg-gray-100',
          text: 'Disconnected',
        };
      case 'error':
        return {
          icon: <AlertTriangle className="h-4 w-4" />,
          color: 'text-red-500',
          bgColor: 'bg-red-100',
          text: 'Error',
        };
      default:
        return {
          icon: <WifiOff className="h-4 w-4" />,
          color: 'text-gray-500',
          bgColor: 'bg-gray-100',
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
        return 'text-green-600';
      case 'good':
        return 'text-blue-600';
      case 'fair':
        return 'text-yellow-600';
      case 'poor':
        return 'text-red-600';
      default:
        return 'text-gray-600';
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
                <div className="text-xs text-gray-600">Connection</div>
                <div className={`flex items-center gap-2 ${statusDisplay.color}`}>
                  {statusDisplay.icon}
                  <span className="font-medium">{statusDisplay.text}</span>
                </div>
              </div>
              <div className="space-y-2">
                <div className="text-xs text-gray-600">Health Score</div>
                <div className="flex items-center gap-2">
                  <Progress value={health.score} className="flex-1 h-2" />
                  <span className="text-sm font-medium">{health.score}/100</span>
                </div>
              </div>
            </div>

            {/* Network Quality */}
            <div className="space-y-2">
              <div className="text-xs text-gray-600">Network Quality</div>
              <div className="flex items-center gap-2">
                <Signal className={`h-4 w-4 ${getNetworkQualityColor(networkQuality.level)}`} />
                <span className={`font-medium ${getNetworkQualityColor(networkQuality.level)}`}>
                  {networkQuality.level.charAt(0).toUpperCase() + networkQuality.level.slice(1)}
                </span>
                {metrics.latency && (
                  <span className="text-xs text-gray-500">
                    ({Math.round(metrics.latency)}ms)
                  </span>
                )}
              </div>
              <div className="text-xs text-gray-600">
                {networkQuality.description}
              </div>
            </div>

            {/* Metrics */}
            <div className="space-y-3">
              <div className="text-xs text-gray-600">Metrics</div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="space-y-1">
                  <div className="text-gray-600">Uptime</div>
                  <div className="font-medium">
                    {metrics.uptime ? formatDuration(metrics.uptime) : 'N/A'}
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-gray-600">Stability</div>
                  <div className="font-medium">
                    {Math.round(metrics.connectionStability * 100)}%
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-gray-600">Reconnects</div>
                  <div className="font-medium">
                    {metrics.totalReconnects}
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-gray-600">Messages</div>
                  <div className="font-medium">
                    {metrics.messagesSent + metrics.messagesReceived}
                  </div>
                </div>
              </div>

              {metrics.lastConnectedAt && (
                <div className="flex items-center gap-2 text-xs text-gray-600">
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
                <div className="text-xs text-gray-600">Issues</div>
                <div className="space-y-1">
                  {health.issues.map((issue, index) => (
                    <div key={index} className="flex items-start gap-2 text-xs">
                      <XCircle className="h-3 w-3 text-red-500 mt-0.5 flex-shrink-0" />
                      <span className="text-red-700">{issue}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Recommendations */}
            {health.recommendations.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs text-gray-600">Recommendations</div>
                <div className="space-y-1">
                  {health.recommendations.map((recommendation, index) => (
                    <div key={index} className="flex items-start gap-2 text-xs">
                      <CheckCircle className="h-3 w-3 text-blue-500 mt-0.5 flex-shrink-0" />
                      <span className="text-blue-700">{recommendation}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Connection Attempts */}
            {metrics.reconnectAttempts > 0 && (
              <div className="bg-yellow-50 border border-yellow-200 rounded p-2">
                <div className="text-xs text-yellow-800">
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