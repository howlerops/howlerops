/**
 * WebSocket Provider - React provider component for WebSocket context
 */

import React, { useMemo } from 'react';
import { useWebSocket } from './useWebSocket';
import { WebSocketContext } from './WebSocketContext';
import {
  WebSocketContextValue,
  UseWebSocketOptions,
} from '../../types/websocket';

interface WebSocketProviderProps {
  children: React.ReactNode;
  options?: UseWebSocketOptions;
}

/**
 * WebSocket Provider Component
 */
export function WebSocketProvider({ children, options = {} }: WebSocketProviderProps) {
  const webSocket = useWebSocket(options);

  const contextValue: WebSocketContextValue = useMemo(() => ({
    // Connection
    socket: webSocket.socket,
    connectionState: webSocket.connectionState,
    connect: webSocket.connect,
    disconnect: webSocket.disconnect,

    // Rooms
    joinRoom: webSocket.joinRoom,
    leaveRoom: webSocket.leaveRoom,
    getRooms: webSocket.getRooms,

    // Messaging
    sendMessage: webSocket.sendMessage,

    // Event handling
    on: webSocket.on,
    off: webSocket.off,

    // Utilities
    getStats: webSocket.getStats,
    healthCheck: webSocket.healthCheck,
  }), [webSocket]);

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
}