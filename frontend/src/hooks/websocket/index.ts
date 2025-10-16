/**
 * WebSocket Hooks - Entry point for all WebSocket-related hooks
 * Exports all hooks and utilities for WebSocket functionality
 */

export { useWebSocket } from './use-websocket';
export { useRealtimeQuery } from './use-realtime-query';
export { useTableSync } from './use-table-sync';

// Re-export types for convenience
export * from '../../types/websocket';

// WebSocket Context Provider
export { WebSocketProvider } from './websocket-provider';
export { useWebSocketContext } from './use-websocket-context';

// Utility hooks
export { useOptimisticUpdates } from './use-optimistic-updates';
export { useConflictResolution } from './use-conflict-resolution';
export { useConnectionStatus } from './use-connection-status';