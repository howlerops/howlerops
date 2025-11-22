/**
 * WebSocket Hooks - Entry point for all WebSocket-related hooks
 * Exports all hooks and utilities for WebSocket functionality
 */

export { useRealtimeQuery } from './use-realtime-query';
export { useTableSync } from './use-table-sync';
export { useWebSocket } from './use-websocket';

// Re-export types for convenience
export * from '../../types/websocket';

// WebSocket Context Provider
export { useWebSocketContext } from './use-websocket-context';
export { WebSocketProvider } from './websocket-provider';

// Utility hooks
export { useConflictResolution } from './use-conflict-resolution';
export { useConnectionStatus } from './use-connection-status';
export { useOptimisticUpdates } from './use-optimistic-updates';