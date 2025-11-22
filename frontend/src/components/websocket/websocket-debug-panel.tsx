/**
 * WebSocket Debug Panel - Development tool for monitoring WebSocket activity
 * Shows connection status, events, and performance metrics
 */

import {
  Bug,
  Download,
  Pause,
  // RefreshCw,
  Play,
  // Activity,
  // BarChart3,
  // Settings,
  Trash2,
} from 'lucide-react';
import React, { useCallback, useEffect,useState } from 'react';

import { useWebSocketContext } from '../../hooks/websocket';
import { useConnectionStatus } from '../../hooks/websocket';
import { useOptimisticUpdates } from '../../hooks/websocket';
import { useConflictResolution } from '../../hooks/websocket';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../ui/card';
import { ScrollArea } from '../ui/scroll-area';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs';

interface WebSocketEvent {
  id: string;
  timestamp: number;
  type: string;
  direction: 'in' | 'out';
  data: unknown;
  size?: number;
}

interface WebSocketDebugPanelProps {
  isVisible?: boolean;
  onToggle?: () => void;
  maxEvents?: number;
  className?: string;
}

export function WebSocketDebugPanel({
  isVisible = false,
  onToggle,
  maxEvents = 100,
  className = '',
}: WebSocketDebugPanelProps) {
  const { connectionState, getStats: getWSStats } = useWebSocketContext();
  const { getStatusReport } = useConnectionStatus();
  const { getStats: getOptimisticStats } = useOptimisticUpdates();
  const { getStats: getConflictStats } = useConflictResolution();

  // Local state
  const [events, setEvents] = useState<WebSocketEvent[]>([]);
  const [isRecording, setIsRecording] = useState(true);
  const [selectedEvent, setSelectedEvent] = useState<WebSocketEvent | null>(null);
  const [activeTab, setActiveTab] = useState('events');

  /**
   * Add event to the log
   */
  const addEvent = useCallback((
    type: string,
    direction: 'in' | 'out',
    data: unknown,
    size?: number
  ) => {
    if (!isRecording) return;

    const event: WebSocketEvent = {
      id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      type,
      direction,
      data,
      size,
    };

    setEvents(prev => {
      const newEvents = [event, ...prev];
      return newEvents.slice(0, maxEvents);
    });
  }, [isRecording, maxEvents]);

  /**
   * Clear all events
   */
  const clearEvents = useCallback(() => {
    setEvents([]);
    setSelectedEvent(null);
  }, []);

  /**
   * Export events as JSON
   */
  const exportEvents = useCallback(() => {
    const data = {
      timestamp: new Date().toISOString(),
      connectionState,
      events,
      stats: {
        websocket: getWSStats(),
        optimistic: getOptimisticStats(),
        conflicts: getConflictStats(),
      },
    };

    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: 'application/json',
    });

    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `websocket-debug-${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [connectionState, events, getWSStats, getOptimisticStats, getConflictStats]);

  /**
   * Format event data for display
   */
  const formatEventData = useCallback((data: unknown): string => {
    try {
      return JSON.stringify(data, null, 2);
    } catch {  
      return String(data);
    }
  }, []);

  /**
   * Format timestamp
   */
  const formatTimestamp = useCallback((timestamp: number): string => {
    return new Date(timestamp).toLocaleTimeString();
  }, []);

  /**
   * Get event type color
   */
  const getEventTypeColor = useCallback((type: string) => {
    if (type.includes('query')) return 'text-primary bg-primary/10';
    if (type.includes('table')) return 'text-primary bg-primary/10';
    if (type.includes('connection')) return 'text-accent-foreground bg-accent/10';
    if (type.includes('error')) return 'text-destructive bg-destructive/10';
    return 'text-muted-foreground bg-muted';
  }, []);

  // Mock event listening (in real implementation, this would connect to actual WebSocket events)
  useEffect(() => {
    // This is a placeholder - in real implementation, you would:
    // 1. Listen to WebSocket events from the socket instance
    // 2. Intercept messages being sent/received
    // 3. Track performance metrics

    const interval = setInterval(() => {
      if (connectionState.status === 'connected' && Math.random() > 0.8) {
        // Simulate random events for demo
        const eventTypes = [
          'query:progress',
          'table:edit:apply',
          'connection:status',
          'data:chunk',
        ];
        const randomType = eventTypes[Math.floor(Math.random() * eventTypes.length)];

        addEvent(randomType, 'in', {
          simulated: true,
          timestamp: Date.now(),
          data: `Sample ${randomType} event`,
        });
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [connectionState.status, addEvent]);

  if (!isVisible) return null;

  const statusReport = getStatusReport();
  const _wsStats = getWSStats();  
  const optimisticStats = getOptimisticStats();
  const conflictStats = getConflictStats();

  return (
    <Card className={`fixed bottom-4 right-4 w-96 h-[500px] shadow-lg z-50 ${className}`}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bug className="h-5 w-5" />
            <CardTitle className="text-base">WebSocket Debug</CardTitle>
          </div>
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setIsRecording(!isRecording)}
              className="h-6 w-6 p-0"
            >
              {isRecording ? (
                <Pause className="h-3 w-3" />
              ) : (
                <Play className="h-3 w-3" />
              )}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={exportEvents}
              className="h-6 w-6 p-0"
            >
              <Download className="h-3 w-3" />
            </Button>
            {onToggle && (
              <Button
                size="sm"
                variant="ghost"
                onClick={onToggle}
                className="h-6 w-6 p-0"
              >
                ×
              </Button>
            )}
          </div>
        </div>
        <CardDescription className="text-xs">
          Monitor WebSocket activity and performance
        </CardDescription>
      </CardHeader>

      <CardContent className="pt-0">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="h-full">
          <TabsList className="grid w-full grid-cols-4 h-8 text-xs">
            <TabsTrigger value="events" className="text-xs">Events</TabsTrigger>
            <TabsTrigger value="stats" className="text-xs">Stats</TabsTrigger>
            <TabsTrigger value="status" className="text-xs">Status</TabsTrigger>
            <TabsTrigger value="config" className="text-xs">Config</TabsTrigger>
          </TabsList>

          <TabsContent value="events" className="space-y-2 h-[380px]">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-xs">
                  {events.length} events
                </Badge>
                <Badge
                  variant={isRecording ? 'default' : 'secondary'}
                  className="text-xs"
                >
                  {isRecording ? 'Recording' : 'Paused'}
                </Badge>
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={clearEvents}
                className="h-6 text-xs"
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>

            <ScrollArea className="h-[340px]">
              <div className="space-y-1">
                {events.map(event => (
                  <div
                    key={event.id}
                    className={`p-2 rounded text-xs cursor-pointer border transition-colors ${
                      selectedEvent?.id === event.id
                        ? 'border-primary bg-primary/10'
                        : 'border-border hover:bg-muted/50'
                    }`}
                    onClick={() => setSelectedEvent(event)}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="outline"
                          className={`h-4 text-xs ${getEventTypeColor(event.type)}`}
                        >
                          {event.direction === 'in' ? '←' : '→'}
                        </Badge>
                        <span className="font-medium">{event.type}</span>
                      </div>
                      <span className="text-muted-foreground">
                        {formatTimestamp(event.timestamp)}
                      </span>
                    </div>
                    {event.size && (
                      <div className="text-muted-foreground mt-1">
                        {event.size} bytes
                      </div>
                    )}
                  </div>
                ))}
                {events.length === 0 && (
                  <div className="text-center text-muted-foreground py-8">
                    No events recorded
                  </div>
                )}
              </div>
            </ScrollArea>

            {selectedEvent && (
              <div className="border-t pt-2">
                <div className="text-xs font-medium mb-1">Event Data:</div>
                <ScrollArea className="h-20">
                  <pre className="text-xs text-muted-foreground whitespace-pre-wrap">
                    {formatEventData(selectedEvent.data)}
                  </pre>
                </ScrollArea>
              </div>
            )}
          </TabsContent>

          <TabsContent value="stats" className="space-y-3 h-[380px] overflow-y-auto">
            <div className="space-y-3">
              {/* Connection Stats */}
              <div>
                <div className="text-xs font-medium mb-2">Connection</div>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div className="bg-muted p-2 rounded">
                    <div className="text-muted-foreground">Status</div>
                    <div className="font-medium">{connectionState.status}</div>
                  </div>
                  <div className="bg-muted p-2 rounded">
                    <div className="text-muted-foreground">Reconnects</div>
                    <div className="font-medium">{connectionState.reconnectAttempts}</div>
                  </div>
                </div>
              </div>

              {/* Optimistic Updates */}
              <div>
                <div className="text-xs font-medium mb-2">Optimistic Updates</div>
                <div className="grid grid-cols-3 gap-2 text-xs">
                  <div className="bg-primary/10 p-2 rounded">
                    <div className="text-primary">Pending</div>
                    <div className="font-medium">{optimisticStats.pending}</div>
                  </div>
                  <div className="bg-primary/10 p-2 rounded">
                    <div className="text-primary">Confirmed</div>
                    <div className="font-medium">{optimisticStats.confirmed}</div>
                  </div>
                  <div className="bg-destructive/10 p-2 rounded">
                    <div className="text-destructive">Rejected</div>
                    <div className="font-medium">{optimisticStats.rejected}</div>
                  </div>
                </div>
              </div>

              {/* Conflicts */}
              <div>
                <div className="text-xs font-medium mb-2">Conflicts</div>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div className="bg-accent/10 p-2 rounded">
                    <div className="text-accent-foreground">Active</div>
                    <div className="font-medium">{conflictStats.activeConflicts}</div>
                  </div>
                  <div className="bg-muted p-2 rounded">
                    <div className="text-muted-foreground">Resolved</div>
                    <div className="font-medium">{conflictStats.totalResolved}</div>
                  </div>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="status" className="space-y-3 h-[380px] overflow-y-auto">
            <ScrollArea className="h-full">
              <pre className="text-xs whitespace-pre-wrap">
                {JSON.stringify(statusReport, null, 2)}
              </pre>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="config" className="space-y-3 h-[380px]">
            <div className="space-y-3 text-xs">
              <div>
                <div className="font-medium mb-2">Recording Settings</div>
                <div className="space-y-2">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={isRecording}
                      onChange={(e) => setIsRecording(e.target.checked)}
                      className="rounded"
                    />
                    <span>Record events</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <span>Max events:</span>
                    <input
                      type="number"
                      value={maxEvents}
                      onChange={(e) => {
                        // In real implementation, this would update the maxEvents prop
                        console.log('Max events:', e.target.value);
                      }}
                      className="w-16 px-1 py-0.5 border rounded text-xs"
                      min="10"
                      max="1000"
                    />
                  </div>
                </div>
              </div>

              <div>
                <div className="font-medium mb-2">Actions</div>
                <div className="space-y-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={clearEvents}
                    className="w-full h-7 text-xs"
                  >
                    <Trash2 className="h-3 w-3 mr-1" />
                    Clear Events
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={exportEvents}
                    className="w-full h-7 text-xs"
                  >
                    <Download className="h-3 w-3 mr-1" />
                    Export Debug Data
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}