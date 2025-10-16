/**
 * HowlerOps Web Worker Infrastructure
 * Main export file providing unified API for worker functionality
 */

export * from './types';
export { WorkerClient } from './worker-client';
export { WorkerPool } from './worker-pool';
export { PerformanceMonitor } from './performance-monitor';
export { SharedBufferManager } from './shared-buffer';

// import { WorkerClient } from './worker-client';
import { WorkerPool } from './worker-pool';
import { PerformanceMonitor } from './performance-monitor';
import { SharedBufferManager } from './shared-buffer';
import {
  WorkerPoolConfig,
  QueryResult,
  FilterCondition,
  SortCondition,
  ExportConfig,
  AggregationConfig,
  ValidationRule,
  TransformationRule,
  Priority,
  WorkerMessageType
} from './types';

/**
 * HowlerOps Worker Manager
 * High-level API for managing workers and data processing
 */
export class SQLStudioWorkers {
  private static instance: SQLStudioWorkers | null = null;
  private workerPool: WorkerPool;
  private performanceMonitor: PerformanceMonitor;
  private sharedBufferManager: SharedBufferManager;
  private isInitialized = false;

  private constructor(config?: Partial<WorkerPoolConfig>) {
    this.workerPool = new WorkerPool(config);
    this.performanceMonitor = new PerformanceMonitor(this.workerPool);
    this.sharedBufferManager = new SharedBufferManager();
  }

  /**
   * Get singleton instance
   */
  static getInstance(config?: Partial<WorkerPoolConfig>): SQLStudioWorkers {
    if (!SQLStudioWorkers.instance) {
      SQLStudioWorkers.instance = new SQLStudioWorkers(config);
    }
    return SQLStudioWorkers.instance;
  }

  /**
   * Initialize workers
   */
  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    // Start performance monitoring
    this.performanceMonitor.startRecording();

    // Check SharedArrayBuffer support
    if (SharedBufferManager.checkSupport()) {
      console.log('SharedArrayBuffer is supported - enabling optimized data transfer');
    } else {
      console.log('SharedArrayBuffer not available - using standard message passing');
    }

    this.isInitialized = true;
  }

  /**
   * Parse and validate query results
   */
  async parseQueryResults(data: unknown): Promise<QueryResult> {
    return this.workerPool.execute(
      WorkerMessageType.PARSE_QUERY_RESULTS,
      data,
      Priority.HIGH
    );
  }

  /**
   * Filter data with conditions
   */
  async filterData(
    data: QueryResult,
    filters: FilterCondition[]
  ): Promise<QueryResult> {
    // Use SharedArrayBuffer for large datasets
    if (data.rows.length > 10000 && SharedBufferManager.checkSupport()) {
      const buffer = this.sharedBufferManager.createBuffer({
        rowCount: data.rows.length,
        columns: data.columns
      });

      if (buffer) {
        this.sharedBufferManager.writeData(data);
        return this.workerPool.execute(
          WorkerMessageType.FILTER_DATA,
          { buffer, filters },
          Priority.NORMAL,
          [buffer]
        );
      }
    }

    return this.workerPool.execute(
      WorkerMessageType.FILTER_DATA,
      { data, filters },
      Priority.NORMAL
    );
  }

  /**
   * Sort data with conditions
   */
  async sortData(
    data: QueryResult,
    sorts: SortCondition[]
  ): Promise<QueryResult> {
    return this.workerPool.execute(
      WorkerMessageType.SORT_DATA,
      { data, sorts },
      Priority.NORMAL
    );
  }

  /**
   * Export data to various formats
   */
  async exportData(
    data: QueryResult,
    config: ExportConfig
  ): Promise<string | Blob> {
    let messageType: WorkerMessageType;

    switch (config.format) {
      case 'CSV':
        messageType = WorkerMessageType.EXPORT_CSV;
        break;
      case 'JSON':
        messageType = WorkerMessageType.EXPORT_JSON;
        break;
      case 'EXCEL':
        messageType = WorkerMessageType.EXPORT_EXCEL;
        break;
      default:
        throw new Error(`Unsupported export format: ${config.format}`);
    }

    const result = await this.workerPool.execute(
      messageType,
      { data, config },
      Priority.NORMAL
    );

    // Apply compression if requested
    if (config.compression && config.compression !== 'NONE') {
      return this.compressData(result, config.compression);
    }

    return result;
  }

  /**
   * Calculate aggregations
   */
  async calculateAggregations(
    data: QueryResult,
    config: AggregationConfig
  ): Promise<unknown[]> {
    return this.workerPool.execute(
      WorkerMessageType.CALCULATE_AGGREGATIONS,
      { data, config },
      Priority.NORMAL
    );
  }

  /**
   * Calculate statistics for columns
   */
  async calculateStatistics(
    data: QueryResult,
    columns?: string[]
  ): Promise<unknown> {
    const targetColumns = columns || data.columns.map(c => c.name);
    return this.workerPool.execute(
      WorkerMessageType.CALCULATE_STATISTICS,
      { data, columns: targetColumns },
      Priority.LOW
    );
  }

  /**
   * Validate data against rules
   */
  async validateData(
    data: QueryResult,
    rules: ValidationRule[]
  ): Promise<unknown> {
    return this.workerPool.execute(
      WorkerMessageType.VALIDATE_DATA,
      { data, rules },
      Priority.NORMAL
    );
  }

  /**
   * Transform data with rules
   */
  async transformData(
    data: QueryResult,
    transformations: TransformationRule[]
  ): Promise<QueryResult> {
    return this.workerPool.execute(
      WorkerMessageType.TRANSFORM_DATA,
      { data, transformations },
      Priority.NORMAL
    );
  }

  /**
   * Process large dataset in batches
   */
  async processBatchData<T = unknown>(
    data: unknown[],
    processor: (batch: unknown[]) => Promise<T[]>,
    batchSize = 10000
  ): Promise<T[]> {
    const results: T[] = [];
    const totalBatches = Math.ceil(data.length / batchSize);

    for (let i = 0; i < totalBatches; i++) {
      const start = i * batchSize;
      const end = Math.min(start + batchSize, data.length);
      const batch = data.slice(start, end);

      const batchResult = await processor(batch);
      results.push(...batchResult);

      // Report progress
      const progress = ((i + 1) / totalBatches) * 100;
      console.log(`Processed batch ${i + 1}/${totalBatches} (${progress.toFixed(1)}%)`);
    }

    return results;
  }

  /**
   * Cancel a running operation
   */
  async cancelOperation(operationId: string): Promise<boolean> {
    return this.workerPool.cancelTask(operationId);
  }

  /**
   * Get performance metrics
   */
  getPerformanceMetrics() {
    return {
      realtime: this.performanceMonitor.getRealtimeMetrics(),
      pool: this.workerPool.getMetrics(),
      report: this.performanceMonitor.generateReport()
    };
  }

  /**
   * Export performance report
   */
  exportPerformanceReport(format: 'json' | 'csv' = 'json'): string {
    return this.performanceMonitor.exportMetrics(format);
  }

  /**
   * Get worker pool status
   */
  getWorkerPoolStatus() {
    return {
      metrics: this.workerPool.getMetrics(),
      workers: this.workerPool.getWorkerStates(),
      queue: this.workerPool.getQueuedTasks()
    };
  }

  /**
   * Compress data (simplified implementation)
   */
  private async compressData(data: string, compression: string): Promise<Blob> {
    // In a real implementation, use CompressionStream API or a library
    const encoder = new TextEncoder();
    const encoded = encoder.encode(data);

    if ('CompressionStream' in window) {
      const stream = new ReadableStream({
        start(controller) {
          controller.enqueue(encoded);
          controller.close();
        }
      });

      const compressedStream = stream.pipeThrough(
        new (window as unknown as { CompressionStream: new (format: string) => CompressionStream }).CompressionStream(
          compression.toLowerCase() === 'gzip' ? 'gzip' : 'deflate'
        )
      );

      const chunks: Uint8Array[] = [];
      const reader = compressedStream.getReader();

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
      }

      return new Blob(chunks, { type: 'application/octet-stream' });
    }

    // Fallback: return uncompressed blob
    return new Blob([data], { type: 'text/plain' });
  }

  /**
   * Shutdown and cleanup
   */
  async shutdown(): Promise<void> {
    this.performanceMonitor.stopRecording();
    this.performanceMonitor.destroy();
    this.sharedBufferManager.destroy();
    await this.workerPool.shutdown();
    this.isInitialized = false;
    SQLStudioWorkers.instance = null;
  }
}

// Convenience functions

/**
 * Get default worker manager instance
 */
export function getWorkerManager(config?: Partial<WorkerPoolConfig>): SQLStudioWorkers {
  return SQLStudioWorkers.getInstance(config);
}

/**
 * Quick parse function
 */
export async function parseQueryResults(data: unknown): Promise<QueryResult> {
  const manager = getWorkerManager();
  await manager.initialize();
  return manager.parseQueryResults(data);
}

/**
 * Quick filter function
 */
export async function filterData(
  data: QueryResult,
  filters: FilterCondition[]
): Promise<QueryResult> {
  const manager = getWorkerManager();
  await manager.initialize();
  return manager.filterData(data, filters);
}

/**
 * Quick sort function
 */
export async function sortData(
  data: QueryResult,
  sorts: SortCondition[]
): Promise<QueryResult> {
  const manager = getWorkerManager();
  await manager.initialize();
  return manager.sortData(data, sorts);
}

/**
 * Quick export function
 */
export async function exportData(
  data: QueryResult,
  config: ExportConfig
): Promise<string | Blob> {
  const manager = getWorkerManager();
  await manager.initialize();
  return manager.exportData(data, config);
}

/**
 * Quick statistics function
 */
export async function calculateStatistics(
  data: QueryResult,
  columns?: string[]
): Promise<unknown> {
  const manager = getWorkerManager();
  await manager.initialize();
  return manager.calculateStatistics(data, columns);
}