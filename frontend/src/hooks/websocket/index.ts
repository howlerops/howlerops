/**
 * WebSocket Hooks - Entry point for all WebSocket-related hooks
 * Exports all hooks and utilities for WebSocket functionality
 */

export { useWebSocket } from './useWebSocket';
export { useRealtimeQuery } from './useRealtimeQuery';
export { useTableSync } from './useTableSync';

// Re-export types for convenience
export * from '../../types/websocket';

// WebSocket Context Provider
export { WebSocketProvider } from './WebSocketProvider';
export { useWebSocketContext } from './useWebSocketContext';

// Utility hooks
export { useOptimisticUpdates } from './useOptimisticUpdates';
export { useConflictResolution } from './useConflictResolution';
export { useConnectionStatus } from './useConnectionStatus';