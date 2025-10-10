/**
 * useWebSocketContext - Hook for accessing WebSocket context
 */

import { useContext } from 'react';
import { WebSocketContext } from './WebSocketContext';

/**
 * Hook to access WebSocket context
 * @throws Error if used outside WebSocketProvider
 */
export function useWebSocketContext() {
  const context = useContext(WebSocketContext);

  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider');
  }

  return context;
}