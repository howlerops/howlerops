/**
 * Worker Client - Type-safe communication layer for Web Workers
 * Provides Promise-based API for main thread to communicate with workers
 */

import {
  AggregationConfig,
  ExportConfig,
  ExportFormat,
  FilterCondition,
  Priority,
  ProgressUpdate,
  QueryResult,
  SortCondition,
  TransformationRule,
  ValidationRule,
  WorkerError,
  WorkerMessage,
  WorkerMessageType,
  WorkerResponse} from './types';

export interface WorkerClientConfig {
  workerPath?: string;
  timeout?: number;
  retries?: number;
  onProgress?: (progress: ProgressUpdate) => void;
  onError?: (error: WorkerError) => void;
  enableSharedArrayBuffer?: boolean;
}

export class WorkerClient {
  private worker: Worker | null = null;
  private pendingMessages: Map<string, {
    resolve: (value: unknown) => void;
    reject: (error: unknown) => void;
    timeout: NodeJS.Timeout;
  }> = new Map();

  private config: Required<WorkerClientConfig>;
  private messageQueue: WorkerMessage[] = [];
  private isProcessing = false;

  constructor(config: WorkerClientConfig = {}) {
    this.config = {
      workerPath: new URL('./dataProcessor.worker.ts', import.meta.url).href,
      timeout: 60000, // 60 seconds default
      retries: 3,
      onProgress: () => {},
      onError: () => {},
      enableSharedArrayBuffer: false,
      ...config
    };

    this.initializeWorker();
  }

  private initializeWorker(): void {
    try {
      this.worker = new Worker(this.config.workerPath, {
        type: 'module'
      });

      this.worker.addEventListener('message', this.handleWorkerMessage.bind(this));
      this.worker.addEventListener('error', this.handleWorkerError.bind(this));
    } catch (error) {
      console.error('Failed to initialize worker:', error);
      throw new Error('Worker initialization failed');
    }
  }

  private handleWorkerMessage(event: MessageEvent<WorkerResponse>): void {
    const response = event.data;

    // Handle progress updates
    if (response.type === WorkerMessageType.PROGRESS_UPDATE && response.result) {
      this.config.onProgress(response.result as ProgressUpdate);
      return;
    }

    // Handle regular responses
    const pending = this.pendingMessages.get(response.id);
    if (pending) {
      clearTimeout(pending.timeout);
      this.pendingMessages.delete(response.id);

      if (response.success) {
        pending.resolve(response.result);
      } else {
        const error = response.error || { code: 'UNKNOWN', message: 'Unknown error' };
        this.config.onError(error);
        pending.reject(error);
      }
    }
  }

  private handleWorkerError(event: ErrorEvent): void {
    console.error('Worker error:', event);
    const error: WorkerError = {
      code: 'WORKER_ERROR',
      message: event.message,
      stack: event.error?.stack
    };
    this.config.onError(error);

    // Reject all pending messages
    for (const pending of this.pendingMessages.values()) {
      clearTimeout(pending.timeout);
      pending.reject(error);
    }
    this.pendingMessages.clear();

    // Attempt to restart worker
    this.restartWorker();
  }

  private restartWorker(): void {
    if (this.worker) {
      this.worker.terminate();
    }
    this.initializeWorker();
    this.processQueue();
  }

  private async sendMessage<T = unknown>(
    type: WorkerMessageType,
    payload: unknown,
    priority: Priority = Priority.NORMAL,
    transferable?: Transferable[]
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      const id = this.generateId();
      const message: WorkerMessage = {
        id,
        type,
        payload,
        timestamp: Date.now(),
        priority,
        transferable
      };

      // Set up timeout
      const timeout = setTimeout(() => {
        this.pendingMessages.delete(id);
        reject(new Error(`Operation timed out after ${this.config.timeout}ms`));
      }, this.config.timeout);

      // Store pending message
      this.pendingMessages.set(id, { resolve: resolve as (value: unknown) => void, reject, timeout });

      // Send or queue message
      if (this.worker) {
        this.worker.postMessage(message, transferable || []);
      } else {
        this.messageQueue.push(message);
      }
    });
  }

  private async processQueue(): Promise<void> {
    if (this.isProcessing || !this.worker || this.messageQueue.length === 0) {
      return;
    }

    this.isProcessing = true;

    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      if (message) {
        this.worker.postMessage(message, message.transferable || []);
      }
    }

    this.isProcessing = false;
  }

  private generateId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  // Public API Methods

  async parseQueryResults(data: unknown): Promise<QueryResult> {
    return this.sendMessage<QueryResult>(
      WorkerMessageType.PARSE_QUERY_RESULTS,
      data,
      Priority.HIGH
    );
  }

  async filterData(
    data: QueryResult,
    filters: FilterCondition[]
  ): Promise<QueryResult> {
    return this.sendMessage<QueryResult>(
      WorkerMessageType.FILTER_DATA,
      { data, filters },
      Priority.NORMAL
    );
  }

  async sortData(
    data: QueryResult,
    sorts: SortCondition[]
  ): Promise<QueryResult> {
    return this.sendMessage<QueryResult>(
      WorkerMessageType.SORT_DATA,
      { data, sorts },
      Priority.NORMAL
    );
  }

  async exportCSV(
    data: QueryResult,
    config: ExportConfig = { format: ExportFormat.CSV }
  ): Promise<string> {
    return this.sendMessage<string>(
      WorkerMessageType.EXPORT_CSV,
      { data, config },
      Priority.NORMAL
    );
  }

  async exportJSON(
    data: QueryResult,
    config: ExportConfig = { format: ExportFormat.JSON }
  ): Promise<string> {
    return this.sendMessage<string>(
      WorkerMessageType.EXPORT_JSON,
      { data, config },
      Priority.NORMAL
    );
  }

  async exportExcel(
    data: QueryResult,
    config: ExportConfig = { format: ExportFormat.EXCEL }
  ): Promise<string> {
    return this.sendMessage<string>(
      WorkerMessageType.EXPORT_EXCEL,
      { data, config },
      Priority.NORMAL
    );
  }

  async calculateAggregations(
    data: QueryResult,
    config: AggregationConfig
  ): Promise<unknown[]> {
    return this.sendMessage<unknown[]>(
      WorkerMessageType.CALCULATE_AGGREGATIONS,
      { data, config },
      Priority.NORMAL
    );
  }

  async calculateStatistics(
    data: QueryResult,
    columns: string[]
  ): Promise<unknown> {
    return this.sendMessage<unknown>(
      WorkerMessageType.CALCULATE_STATISTICS,
      { data, columns },
      Priority.LOW
    );
  }

  async validateData(
    data: QueryResult,
    rules: ValidationRule[]
  ): Promise<unknown> {
    return this.sendMessage<unknown>(
      WorkerMessageType.VALIDATE_DATA,
      { data, rules },
      Priority.NORMAL
    );
  }

  async transformData(
    data: QueryResult,
    transformations: TransformationRule[]
  ): Promise<QueryResult> {
    return this.sendMessage<QueryResult>(
      WorkerMessageType.TRANSFORM_DATA,
      { data, transformations },
      Priority.NORMAL
    );
  }

  async cancelOperation(operationId: string): Promise<void> {
    return this.sendMessage<void>(
      WorkerMessageType.CANCEL_OPERATION,
      { operationId },
      Priority.CRITICAL
    );
  }

  // Utility Methods

  async processLargeDataset(
    data: unknown[],
    batchSize: number = 10000,
    processor: (batch: unknown[]) => Promise<unknown[]>
  ): Promise<unknown[]> {
    const results: unknown[] = [];
    const totalBatches = Math.ceil(data.length / batchSize);

    for (let i = 0; i < totalBatches; i++) {
      const start = i * batchSize;
      const end = Math.min(start + batchSize, data.length);
      const batch = data.slice(start, end);

      const batchResult = await processor(batch);
      results.push(...batchResult);

      // Report progress
      this.config.onProgress({
        id: 'batch-processing',
        current: i + 1,
        total: totalBatches,
        percentage: ((i + 1) / totalBatches) * 100,
        message: `Processing batch ${i + 1} of ${totalBatches}`
      });
    }

    return results;
  }

  // Cleanup
  terminate(): void {
    if (this.worker) {
      // Clear pending messages
      for (const pending of this.pendingMessages.values()) {
        clearTimeout(pending.timeout);
        pending.reject(new Error('Worker terminated'));
      }
      this.pendingMessages.clear();

      // Terminate worker
      this.worker.terminate();
      this.worker = null;
    }
  }

  // Performance Methods
  getMetrics(): {
    pendingMessages: number;
    queuedMessages: number;
    isWorkerActive: boolean;
  } {
    return {
      pendingMessages: this.pendingMessages.size,
      queuedMessages: this.messageQueue.length,
      isWorkerActive: this.worker !== null
    };
  }
}

// Singleton instance for convenience
let defaultClient: WorkerClient | null = null;

export function getDefaultWorkerClient(config?: WorkerClientConfig): WorkerClient {
  if (!defaultClient) {
    defaultClient = new WorkerClient(config);
  }
  return defaultClient;
}

export function terminateDefaultWorkerClient(): void {
  if (defaultClient) {
    defaultClient.terminate();
    defaultClient = null;
  }
}
