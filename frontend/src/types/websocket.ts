/**
 * Frontend WebSocket Types
 * Client-side types for WebSocket communication
 */

import { Socket } from 'socket.io-client';

// Re-export backend types for consistency
export interface BaseEvent {
  id: string;
  timestamp: number;
  userId?: string;
  connectionId: string;
}

// Connection states
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';

export interface ConnectionState {
  status: ConnectionStatus;
  lastConnected?: Date;
  reconnectAttempts: number;
  error?: string;
  serverInfo?: {
    version: string;
    capabilities: {
      streaming: boolean;
      compression: boolean;
      binaryProtocol: boolean;
    };
  };
}

// Query-related types
export interface QueryProgress {
  queryId: string;
  progress: number; // 0-100
  message: string;
  rowsProcessed?: number;
  estimatedTotal?: number;
}

export interface QueryResult {
  queryId: string;
  columns: Array<{
    name: string;
    type: string;
    nullable?: boolean;
  }>;
  rows: unknown[][];
  totalRows: number;
  executionTime: number;
  hasMore: boolean;
  nextCursor?: string;
}

export interface QueryError {
  queryId: string;
  error: {
    message: string;
    code?: string;
    line?: number;
    column?: number;
  };
}

export interface DataChunk {
  queryId: string;
  chunk: unknown[][];
  chunkIndex: number;
  isLast: boolean;
  compressed?: boolean;
}

// Table sync types
export interface TableEdit {
  editId: string;
  tableId: string;
  tableName: string;
  schema?: string;
  rowId: string | number;
  column: string;
  oldValue: unknown;
  newValue: unknown;
  version: number;
  status: 'pending' | 'applying' | 'applied' | 'rejected' | 'conflicted';
  optimistic?: boolean;
}

export interface TableEditConflict {
  editId: string;
  conflictingEdits: string[];
  resolution: 'merge' | 'reject' | 'manual';
  mergedValue?: unknown;
}

export interface TableRowChange {
  tableId: string;
  rowId: string | number;
  changes: Record<string, unknown>;
  version: number;
  operation: 'insert' | 'update' | 'delete';
}

// Room types
export interface Room {
  id: string;
  type: 'table' | 'connection' | 'query';
  metadata: Record<string, unknown>;
  memberCount?: number;
}

export interface RoomMember {
  userId?: string;
  username?: string;
  joinedAt: Date;
  isActive: boolean;
}

// Event handlers
export type EventHandler<T = unknown> = (event: T) => void;

export interface WebSocketEventHandlers {
  // Connection events
  onConnect?: EventHandler<{ connectionId: string; serverInfo: unknown }>;
  onDisconnect?: EventHandler<{ reason: string }>;
  onReconnect?: EventHandler<{ attempt: number }>;
  onError?: EventHandler<{ error: string }>;

  // Query events
  onQueryProgress?: EventHandler<QueryProgress>;
  onQueryResult?: EventHandler<QueryResult>;
  onQueryError?: EventHandler<QueryError>;
  onDataChunk?: EventHandler<DataChunk>;

  // Table events
  onTableEditApply?: EventHandler<{ editId: string; success: boolean; error?: string }>;
  onTableEditConflict?: EventHandler<TableEditConflict>;
  onTableRowUpdate?: EventHandler<TableRowChange>;
  onTableRowInsert?: EventHandler<TableRowChange>;
  onTableRowDelete?: EventHandler<TableRowChange>;

  // Room events
  onUserJoin?: EventHandler<{ userId: string; username: string; roomId: string }>;
  onUserLeave?: EventHandler<{ userId: string; username: string; roomId: string }>;
  onRoomUpdate?: EventHandler<{ roomId: string; metadata: Record<string, unknown> }>;
}

// Hook options
export interface UseWebSocketOptions {
  url?: string;
  autoConnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  enableCompression?: boolean;
  enableBatching?: boolean;
  eventHandlers?: WebSocketEventHandlers;
}

export interface UseRealtimeQueryOptions {
  connectionName: string;
  streaming?: boolean;
  autoExecute?: boolean;
  onProgress?: EventHandler<QueryProgress>;
  onResult?: EventHandler<QueryResult>;
  onError?: EventHandler<QueryError>;
  onChunk?: EventHandler<DataChunk>;
}

export interface UseTableSyncOptions {
  tableId: string;
  tableName: string;
  schema?: string;
  conflictResolution?: 'auto' | 'manual';
  optimisticUpdates?: boolean;
  onEditApply?: EventHandler<{ editId: string; success: boolean }>;
  onConflict?: EventHandler<TableEditConflict>;
  onRowChange?: EventHandler<TableRowChange>;
}

// Optimistic update types
export interface OptimisticUpdate {
  id: string;
  type: 'table_edit' | 'row_operation';
  tableId: string;
  rowId?: string | number;
  changes: Record<string, unknown>;
  originalData: Record<string, unknown>;
  timestamp: number;
  status: 'pending' | 'confirmed' | 'rejected';
}

export interface OptimisticState {
  updates: Map<string, OptimisticUpdate>;
  pendingCount: number;
  lastUpdate: Date | null;
}

// Message queue types
export interface QueuedMessage {
  id: string;
  type: string;
  data: unknown;
  priority: 'low' | 'normal' | 'high';
  timestamp: number;
  retries: number;
}

export interface MessageQueueConfig {
  enabled: boolean;
  batchSize: number;
  flushInterval: number;
  maxRetries: number;
}

// Compression types
export interface CompressionConfig {
  enabled: boolean;
  threshold: number;
  algorithm: 'gzip' | 'deflate';
}

// Reconnection types
export interface ReconnectionConfig {
  enabled: boolean;
  initialDelay: number;
  maxDelay: number;
  backoffFactor: number;
  maxAttempts: number;
}

// WebSocket context types
export interface WebSocketContextValue {
  // Connection
  socket: Socket | null;
  connectionState: ConnectionState;
  connect: () => Promise<void>;
  disconnect: () => void;

  // Rooms
  joinRoom: (roomId: string, roomType: 'table' | 'connection' | 'query', metadata?: Record<string, unknown>) => Promise<void>;
  leaveRoom: (roomId: string) => Promise<void>;
  getRooms: () => Room[];

  // Messaging
  sendMessage: (type: string, data: unknown, options?: { priority?: 'low' | 'normal' | 'high'; acknowledgment?: boolean }) => Promise<void>;

  // Event handling
  on: (event: string, handler: EventHandler) => void;
  off: (event: string, handler: EventHandler) => void;

  // Utilities
  getStats: () => Record<string, unknown>;
  healthCheck: () => Promise<boolean>;
}