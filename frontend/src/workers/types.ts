/**
 * Web Worker Types and Interfaces for HowlerOps
 * Provides type-safe communication between main thread and workers
 */

// Worker Message Types
export enum WorkerMessageType {
  // Data Processing
  PARSE_QUERY_RESULTS = 'PARSE_QUERY_RESULTS',
  FILTER_DATA = 'FILTER_DATA',
  SORT_DATA = 'SORT_DATA',
  EXPORT_CSV = 'EXPORT_CSV',
  EXPORT_JSON = 'EXPORT_JSON',
  EXPORT_EXCEL = 'EXPORT_EXCEL',

  // Aggregations
  CALCULATE_AGGREGATIONS = 'CALCULATE_AGGREGATIONS',
  CALCULATE_STATISTICS = 'CALCULATE_STATISTICS',

  // Validation
  VALIDATE_DATA = 'VALIDATE_DATA',
  TRANSFORM_DATA = 'TRANSFORM_DATA',

  // Control
  CANCEL_OPERATION = 'CANCEL_OPERATION',
  PROGRESS_UPDATE = 'PROGRESS_UPDATE',
  ERROR = 'ERROR',
  SUCCESS = 'SUCCESS',

  // Performance
  PERFORMANCE_METRICS = 'PERFORMANCE_METRICS',
  MEMORY_USAGE = 'MEMORY_USAGE'
}

// Data Types
export interface QueryResult {
  columns: ColumnDefinition[];
  rows: unknown[];
  metadata?: {
    totalRows?: number;
    executionTime?: number;
    affectedRows?: number;
  };
}

export interface ColumnDefinition {
  name: string;
  type: DataType;
  nullable?: boolean;
  primaryKey?: boolean;
  unique?: boolean;
  defaultValue?: unknown;
  maxLength?: number;
}

export enum DataType {
  STRING = 'STRING',
  NUMBER = 'NUMBER',
  INTEGER = 'INTEGER',
  FLOAT = 'FLOAT',
  BOOLEAN = 'BOOLEAN',
  DATE = 'DATE',
  DATETIME = 'DATETIME',
  TIME = 'TIME',
  JSON = 'JSON',
  BINARY = 'BINARY',
  UUID = 'UUID',
  ARRAY = 'ARRAY',
  UNKNOWN = 'UNKNOWN'
}

// Filter Operations
export interface FilterCondition {
  column: string;
  operator: FilterOperator;
  value: unknown;
  caseSensitive?: boolean;
}

export enum FilterOperator {
  EQUALS = 'EQUALS',
  NOT_EQUALS = 'NOT_EQUALS',
  GREATER_THAN = 'GREATER_THAN',
  GREATER_THAN_OR_EQUALS = 'GREATER_THAN_OR_EQUALS',
  LESS_THAN = 'LESS_THAN',
  LESS_THAN_OR_EQUALS = 'LESS_THAN_OR_EQUALS',
  CONTAINS = 'CONTAINS',
  NOT_CONTAINS = 'NOT_CONTAINS',
  STARTS_WITH = 'STARTS_WITH',
  ENDS_WITH = 'ENDS_WITH',
  IN = 'IN',
  NOT_IN = 'NOT_IN',
  IS_NULL = 'IS_NULL',
  IS_NOT_NULL = 'IS_NOT_NULL',
  REGEX = 'REGEX',
  BETWEEN = 'BETWEEN'
}

// Sort Operations
export interface SortCondition {
  column: string;
  direction: SortDirection;
  nullsFirst?: boolean;
  collation?: string;
}

export enum SortDirection {
  ASC = 'ASC',
  DESC = 'DESC'
}

// Aggregation Operations
export interface AggregationConfig {
  groupBy?: string[];
  aggregations: Aggregation[];
  having?: FilterCondition[];
}

export interface Aggregation {
  column: string;
  operation: AggregationOperation;
  alias?: string;
}

export enum AggregationOperation {
  COUNT = 'COUNT',
  COUNT_DISTINCT = 'COUNT_DISTINCT',
  SUM = 'SUM',
  AVG = 'AVG',
  MIN = 'MIN',
  MAX = 'MAX',
  MEDIAN = 'MEDIAN',
  MODE = 'MODE',
  STDDEV = 'STDDEV',
  VARIANCE = 'VARIANCE',
  PERCENTILE = 'PERCENTILE'
}

// Export Formats
export interface ExportConfig {
  format: ExportFormat;
  includeHeaders?: boolean;
  delimiter?: string; // for CSV
  quote?: string; // for CSV
  escape?: string; // for CSV
  dateFormat?: string;
  numberFormat?: string;
  nullValue?: string;
  booleanFormat?: 'true/false' | '1/0' | 'yes/no';
  compression?: CompressionType;
}

export enum ExportFormat {
  CSV = 'CSV',
  JSON = 'JSON',
  EXCEL = 'EXCEL',
  TSV = 'TSV',
  MARKDOWN = 'MARKDOWN',
  HTML = 'HTML',
  XML = 'XML',
  PARQUET = 'PARQUET'
}

export enum CompressionType {
  NONE = 'NONE',
  GZIP = 'GZIP',
  ZIP = 'ZIP',
  BROTLI = 'BROTLI'
}

// Validation Rules
export interface ValidationRule {
  column: string;
  type: ValidationType;
  config?: unknown;
  errorMessage?: string;
}

export enum ValidationType {
  REQUIRED = 'REQUIRED',
  MIN_LENGTH = 'MIN_LENGTH',
  MAX_LENGTH = 'MAX_LENGTH',
  MIN_VALUE = 'MIN_VALUE',
  MAX_VALUE = 'MAX_VALUE',
  PATTERN = 'PATTERN',
  EMAIL = 'EMAIL',
  URL = 'URL',
  UUID = 'UUID',
  CUSTOM = 'CUSTOM'
}

// Worker Messages
export interface WorkerMessage<T = unknown> {
  id: string;
  type: WorkerMessageType;
  payload: T;
  timestamp: number;
  priority?: Priority;
  transferable?: Transferable[];
}

export enum Priority {
  LOW = 0,
  NORMAL = 1,
  HIGH = 2,
  CRITICAL = 3
}

// Worker Responses
export interface WorkerResponse<T = unknown> {
  id: string;
  type: WorkerMessageType;
  success: boolean;
  result?: T;
  error?: WorkerError;
  timestamp: number;
  executionTime?: number;
  memoryUsage?: MemoryUsage;
}

export interface WorkerError {
  code: string;
  message: string;
  stack?: string;
  details?: unknown;
}

// Progress Reporting
export interface ProgressUpdate {
  id: string;
  current: number;
  total: number;
  percentage: number;
  message?: string;
  estimatedTimeRemaining?: number;
  throughput?: number;
}

// Performance Metrics
export interface PerformanceMetrics {
  operationId: string;
  operationType: WorkerMessageType;
  startTime: number;
  endTime: number;
  duration: number;
  rowsProcessed?: number;
  throughput?: number; // rows/second
  memoryUsed?: number;
  cpuUsage?: number;
}

export interface MemoryUsage {
  used: number;
  peak: number;
  limit?: number;
  percentage?: number;
}

// Worker Pool Configuration
export interface WorkerPoolConfig {
  minWorkers: number;
  maxWorkers: number;
  idleTimeout: number; // ms before idle worker is terminated
  taskTimeout: number; // ms before task is considered failed
  maxQueueSize: number;
  enableSharedArrayBuffer?: boolean;
  workerScript?: string;
}

// Task Queue
export interface QueuedTask<T = unknown> {
  id: string;
  message: WorkerMessage<T>;
  priority: Priority;
  retries: number;
  maxRetries: number;
  createdAt: number;
  startedAt?: number;
  completedAt?: number;
  status: TaskStatus;
  assignedWorker?: string;
}

export enum TaskStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  COMPLETED = 'COMPLETED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED',
  TIMEOUT = 'TIMEOUT'
}

// Worker State
export interface WorkerState {
  id: string;
  status: WorkerStatus;
  currentTask?: string;
  tasksCompleted: number;
  tasksFailed: number;
  createdAt: number;
  lastActivityAt: number;
  memoryUsage?: MemoryUsage;
  cpuUsage?: number;
}

export enum WorkerStatus {
  IDLE = 'IDLE',
  BUSY = 'BUSY',
  TERMINATING = 'TERMINATING',
  TERMINATED = 'TERMINATED',
  ERROR = 'ERROR'
}

// Data Transformation
export interface TransformationRule {
  column: string;
  type: TransformationType;
  config?: unknown;
  targetColumn?: string; // for creating new column
}

export enum TransformationType {
  UPPERCASE = 'UPPERCASE',
  LOWERCASE = 'LOWERCASE',
  TRIM = 'TRIM',
  REPLACE = 'REPLACE',
  SUBSTRING = 'SUBSTRING',
  CONCAT = 'CONCAT',
  SPLIT = 'SPLIT',
  DATE_FORMAT = 'DATE_FORMAT',
  NUMBER_FORMAT = 'NUMBER_FORMAT',
  CAST = 'CAST',
  CUSTOM = 'CUSTOM'
}

// Shared Array Buffer Support
export interface SharedDataConfig {
  useSharedArrayBuffer: boolean;
  bufferSize?: number;
  dataFormat?: 'row' | 'column';
}

// Type Guards
export function isWorkerMessage(obj: unknown): obj is WorkerMessage {
  return obj &&
    typeof obj.id === 'string' &&
    typeof obj.type === 'string' &&
    typeof obj.timestamp === 'number' &&
    obj.payload !== undefined;
}

export function isWorkerResponse(obj: unknown): obj is WorkerResponse {
  return obj &&
    typeof obj.id === 'string' &&
    typeof obj.type === 'string' &&
    typeof obj.success === 'boolean' &&
    typeof obj.timestamp === 'number';
}

export function isProgressUpdate(obj: unknown): obj is ProgressUpdate {
  return obj &&
    typeof obj.id === 'string' &&
    typeof obj.current === 'number' &&
    typeof obj.total === 'number' &&
    typeof obj.percentage === 'number';
}