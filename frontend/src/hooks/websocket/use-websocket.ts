/**
 * useWebSocket Hook - Core WebSocket connection management
 * Provides connection state, auto-reconnection, and event handling
 */

import { useCallback,useEffect, useRef, useState } from 'react';
import { io, Socket } from 'socket.io-client';

import {
  ConnectionState,
  ConnectionStatus,
  DataChunk,
  EventHandler,
  QueryError,
  QueryProgress,
  QueryResult,
  Room,
  TableEditConflict,
  TableRowChange,
  UseWebSocketOptions,
} from '../../types/websocket';

const DEFAULT_OPTIONS: UseWebSocketOptions = {
  url: import.meta.env.VITE_WEBSOCKET_URL || 'http://localhost:8000',
  autoConnect: true,
  reconnectInterval: 1000,
  maxReconnectAttempts: 10,
  enableCompression: true,
  enableBatching: true,
};

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const opts = { ...DEFAULT_OPTIONS, ...options };

  // State
  const [connectionState, setConnectionState] = useState<ConnectionState>({
    status: 'disconnected',
    reconnectAttempts: 0,
  });

  const [rooms, setRooms] = useState<Room[]>([]);

  // Refs
  const socketRef = useRef<Socket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const eventHandlersRef = useRef<Map<string, Set<EventHandler>>>(new Map());
  const messageQueueRef = useRef<Array<{ message: unknown; options?: unknown }>>([]);
  const isManuallyDisconnectedRef = useRef(false);
  const connectRef = useRef<(() => Promise<void>) | null>(null);

  /**
   * Update connection status
   */
  const updateConnectionStatus = useCallback((status: ConnectionStatus, error?: string) => {
    setConnectionState(prev => ({
      ...prev,
      status,
      error,
      lastConnected: status === 'connected' ? new Date() : prev.lastConnected,
    }));
  }, []);

  /**
   * Flush queued messages
   */
  const flushMessageQueue = useCallback(() => {
    if (!socketRef.current?.connected) return;

    const queue = messageQueueRef.current;
    messageQueueRef.current = [];

    queue.forEach(({ message }) => {
      try {
        socketRef.current!.emit('message', message);
      } catch (error) {
        console.error('Failed to send queued message:', error);
        // Re-queue failed message
        messageQueueRef.current.push({ message });
      }
    });
  }, []);

  /**
   * Handle incoming events
   */
  const handleIncomingEvent = useCallback((event: { type: string; data?: unknown }) => {
    // Route to specific handlers
    const handlers = eventHandlersRef.current.get(event.type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(event.data || event);
        } catch (error) {
          console.error('Error in event handler:', error);
        }
      });
    }

    // Route to global event handlers
    switch (event.type) {
      case 'query:progress':
        opts.eventHandlers?.onQueryProgress?.(event.data as QueryProgress);
        break;
      case 'query:result':
        opts.eventHandlers?.onQueryResult?.(event.data as QueryResult);
        break;
      case 'query:error':
        opts.eventHandlers?.onQueryError?.(event.data as QueryError);
        break;
      case 'data:chunk':
        opts.eventHandlers?.onDataChunk?.(event.data as DataChunk);
        break;
      case 'table:edit:apply':
        opts.eventHandlers?.onTableEditApply?.(event.data as { editId: string; success: boolean; error?: string });
        break;
      case 'table:edit:conflict':
        opts.eventHandlers?.onTableEditConflict?.(event.data as TableEditConflict);
        break;
      case 'table:row:update':
        opts.eventHandlers?.onTableRowUpdate?.(event.data as TableRowChange);
        break;
      case 'table:row:insert':
        opts.eventHandlers?.onTableRowInsert?.(event.data as TableRowChange);
        break;
      case 'table:row:delete':
        opts.eventHandlers?.onTableRowDelete?.(event.data as TableRowChange);
        break;
      case 'user:join':
        opts.eventHandlers?.onUserJoin?.(event.data as { userId: string; username: string; roomId: string });
        break;
      case 'user:leave':
        opts.eventHandlers?.onUserLeave?.(event.data as { userId: string; username: string; roomId: string });
        break;
    }
  }, [opts.eventHandlers]);

  /**
   * Handle reconnection with exponential backoff
   */
  const scheduleReconnect = useCallback(() => {
    if (isManuallyDisconnectedRef.current) return;
    if (connectionState.reconnectAttempts >= opts.maxReconnectAttempts!) return;

    const delay = Math.min(
      opts.reconnectInterval! * Math.pow(2, connectionState.reconnectAttempts),
      30000 // Max 30 seconds
    );

    reconnectTimeoutRef.current = setTimeout(() => {
      setConnectionState(prev => ({
        ...prev,
        reconnectAttempts: prev.reconnectAttempts + 1,
      }));

      updateConnectionStatus('reconnecting');
      if (connectRef.current) {
        connectRef.current();
      }
    }, delay);
  }, [connectionState.reconnectAttempts, opts.reconnectInterval, opts.maxReconnectAttempts, updateConnectionStatus]);

  /**
   * Connect to WebSocket server
   */
  const connect = useCallback(async () => {
    if (socketRef.current?.connected) return;

    try {
      updateConnectionStatus('connecting');

      const socket = io(opts.url!, {
        transports: ['websocket', 'polling'],
        upgrade: true,
        timeout: 20000,
        forceNew: true,
      });

      socketRef.current = socket;

      // Connection event handlers
      socket.on('connect', () => {
        console.log('WebSocket connected');
        updateConnectionStatus('connected');

        setConnectionState(prev => ({
          ...prev,
          reconnectAttempts: 0,
          error: undefined,
        }));

        // Clear reconnect timeout
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current);
          reconnectTimeoutRef.current = null;
        }

        // Flush queued messages
        flushMessageQueue();

        // Call user handler
        opts.eventHandlers?.onConnect?.({
          connectionId: socket.id || 'unknown',
          serverInfo: {},
        });
      });

      socket.on('disconnect', (reason: string) => {
        console.log('WebSocket disconnected:', reason);
        updateConnectionStatus('disconnected');

        // Call user handler
        opts.eventHandlers?.onDisconnect?.({ reason });

        // Auto-reconnect if not manually disconnected
        if (!isManuallyDisconnectedRef.current && reason !== 'io client disconnect') {
          scheduleReconnect();
        }
      });

      socket.on('connect_error', (error: Error) => {
        console.error('WebSocket connection error:', error);
        updateConnectionStatus('error', error.message);

        // Call user handler
        opts.eventHandlers?.onError?.({ error: error.message });

        // Schedule reconnect
        scheduleReconnect();
      });

      // Event message handler
      socket.on('event', (event: unknown) => {
        if (typeof event === 'object' && event !== null && 'type' in event) {
          handleIncomingEvent(event as { type: string; data?: unknown });
        }
      });

      // Batch message handler
      socket.on('batch', (events: unknown[]) => {
        events.forEach(event => {
          if (typeof event === 'object' && event !== null && 'type' in event) {
            handleIncomingEvent(event as { type: string; data?: unknown });
          }
        });
      });

      // Error handler
      socket.on('error', (error: unknown) => {
        console.error('WebSocket error:', error);
        const errorMessage = error instanceof Error ? error.message : 'Unknown error';
        opts.eventHandlers?.onError?.({ error: errorMessage });
      });

      // Acknowledgment handler
      socket.on('ack', (ack: unknown) => {
        console.log('Message acknowledged:', ack);
      });

      // Connection established handler
      socket.on('connection:established', (data: { serverInfo: { version: string; capabilities: { streaming: boolean; compression: boolean; binaryProtocol: boolean; } } }) => {
        setConnectionState(prev => ({
          ...prev,
          serverInfo: data.serverInfo,
        }));
      });

      isManuallyDisconnectedRef.current = false;

    } catch (error) {
      console.error('Failed to connect:', error);
      updateConnectionStatus('error', error instanceof Error ? error.message : 'Connection failed');
      scheduleReconnect();
    }
  }, [opts.url, opts.eventHandlers, scheduleReconnect, updateConnectionStatus, flushMessageQueue, handleIncomingEvent]);

  /**
   * Disconnect from WebSocket server
   */
  const disconnect = useCallback(() => {
    isManuallyDisconnectedRef.current = true;

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (socketRef.current) {
      socketRef.current.disconnect();
      socketRef.current = null;
    }

    updateConnectionStatus('disconnected');
    setRooms([]);
  }, [updateConnectionStatus]);

  /**
   * Send message to server
   */
  const sendMessage = useCallback(async (
    type: string,
    data: unknown,
    options: { priority?: 'low' | 'normal' | 'high'; acknowledgment?: boolean } = {}
  ) => {
    const message = {
      id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      type,
      data,
      acknowledgment: options.acknowledgment || false,
    };

    if (!socketRef.current?.connected) {
      // Queue message for later
      messageQueueRef.current.push({ message, options });
      return;
    }

    try {
      socketRef.current.emit('message', message);
    } catch (error) {
      console.error('Failed to send message:', error);
      throw error;
    }
  }, []);

  /**
   * Join a room
   */
  const joinRoom = useCallback(async (
    roomId: string,
    roomType: 'table' | 'connection' | 'query',
    metadata: Record<string, unknown> = {}
  ) => {
    if (!socketRef.current?.connected) {
      throw new Error('Not connected to WebSocket server');
    }

    const data = { roomId, roomType, metadata };

    socketRef.current.emit('join_room', data);

    // Add room to local state
    setRooms(prev => {
      const existing = prev.find(room => room.id === roomId);
      if (existing) return prev;

      return [...prev, { id: roomId, type: roomType, metadata }];
    });
  }, []);

  /**
   * Leave a room
   */
  const leaveRoom = useCallback(async (roomId: string) => {
    if (!socketRef.current?.connected) {
      throw new Error('Not connected to WebSocket server');
    }

    socketRef.current.emit('leave_room', { roomId });

    // Remove room from local state
    setRooms(prev => prev.filter(room => room.id !== roomId));
  }, []);

  /**
   * Subscribe to events
   */
  const on = useCallback((event: string, handler: EventHandler) => {
    if (!eventHandlersRef.current.has(event)) {
      eventHandlersRef.current.set(event, new Set());
    }
    eventHandlersRef.current.get(event)!.add(handler);
  }, []);

  /**
   * Unsubscribe from events
   */
  const off = useCallback((event: string, handler: EventHandler) => {
    const handlers = eventHandlersRef.current.get(event);
    if (handlers) {
      handlers.delete(handler);
      if (handlers.size === 0) {
        eventHandlersRef.current.delete(event);
      }
    }
  }, []);

  /**
   * Get current rooms
   */
  const getRooms = useCallback(() => rooms, [rooms]);

  /**
   * Get connection statistics
   */
  const getStats = useCallback(() => {
    return {
      connectionState,
      rooms: rooms.length,
      queuedMessages: messageQueueRef.current.length,
      eventHandlers: Array.from(eventHandlersRef.current.keys()),
      socketConnected: socketRef.current?.connected || false,
    };
  }, [connectionState, rooms]);

  /**
   * Health check
   */
  const healthCheck = useCallback(async (): Promise<boolean> => {
    if (!socketRef.current?.connected) return false;

    try {
      // Send ping and wait for pong
      const pingStart = Date.now();

      return new Promise((resolve) => {
        const timeout = setTimeout(() => resolve(false), 5000);

        socketRef.current!.emit('ping');
        socketRef.current!.once('pong', () => {
          clearTimeout(timeout);
          const latency = Date.now() - pingStart;
          console.log('WebSocket latency:', latency, 'ms');
          resolve(latency < 1000); // Consider healthy if latency < 1s
        });
      });
    } catch {
      return false;
    }
  }, []);

  // Keep connect ref up to date
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  // Auto-connect on mount
  useEffect(() => {
    if (opts.autoConnect) {
       
      connect();
    }

    return () => {
      disconnect();
    };
  }, [opts.autoConnect, connect, disconnect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, []);

  /**
   * Get socket instance (safe to call during render)
   */
  const getSocket = useCallback(() => socketRef.current, []);

  return {
    // Connection
    getSocket,
    connectionState,
    connect,
    disconnect,

    // Rooms
    joinRoom,
    leaveRoom,
    getRooms,

    // Messaging
    sendMessage,

    // Event handling
    on,
    off,

    // Utilities
    getStats,
    healthCheck,
  };
}