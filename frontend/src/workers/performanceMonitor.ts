/**
 * Performance Monitoring and Metrics System
 * Tracks and analyzes worker performance for optimization
 */

import {
  PerformanceMetrics,
  MemoryUsage,
  WorkerMessageType,
  WorkerPoolMetrics
} from './types';
import { WorkerPool } from './workerPool';

export interface PerformanceSnapshot {
  timestamp: number;
  metrics: PerformanceMetrics;
  memoryUsage: MemoryUsage;
  workerPoolMetrics?: WorkerPoolMetrics;
}

export interface PerformanceReport {
  startTime: number;
  endTime: number;
  duration: number;
  operationsCount: number;
  averageExecutionTime: number;
  medianExecutionTime: number;
  p95ExecutionTime: number;
  p99ExecutionTime: number;
  maxExecutionTime: number;
  minExecutionTime: number;
  throughput: number;
  memoryPeakUsage: number;
  memoryAverageUsage: number;
  errorRate: number;
  operationBreakdown: Map<WorkerMessageType, OperationStats>;
  timeSeriesData: TimeSeriesData[];
}

export interface OperationStats {
  count: number;
  totalTime: number;
  averageTime: number;
  minTime: number;
  maxTime: number;
  errors: number;
  throughput: number;
}

export interface TimeSeriesData {
  timestamp: number;
  executionTime: number;
  memoryUsage: number;
  queueLength: number;
  activeWorkers: number;
}

export class PerformanceMonitor {
  private snapshots: PerformanceSnapshot[] = [];
  private operationMetrics: Map<string, PerformanceMetrics> = new Map();
  private startTime: number = Date.now();
  private maxSnapshots: number = 10000;
  private isRecording: boolean = false;
  private workerPool?: WorkerPool;

  // Performance observers (if available)
  private performanceObserver?: PerformanceObserver;
  private resourceObserver?: PerformanceObserver;

  constructor(workerPool?: WorkerPool) {
    this.workerPool = workerPool;
    this.initializeObservers();
  }

  private initializeObservers(): void {
    // Initialize Performance Observer for measuring operations
    if (typeof PerformanceObserver !== 'undefined') {
      try {
        this.performanceObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (entry.name.startsWith('worker-operation-')) {
              this.recordPerformanceEntry(entry);
            }
          }
        });

        this.performanceObserver.observe({
          entryTypes: ['measure']
        });
      } catch (error) {
        console.warn('PerformanceObserver not available:', error);
      }
    }

    // Initialize Resource Timing Observer
    if (typeof PerformanceObserver !== 'undefined') {
      try {
        this.resourceObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (entry.entryType === 'resource') {
              this.recordResourceTiming(entry as PerformanceResourceTiming);
            }
          }
        });

        this.resourceObserver.observe({
          entryTypes: ['resource']
        });
      } catch (error) {
        console.warn('Resource timing not available:', error);
      }
    }
  }

  private recordPerformanceEntry(entry: PerformanceEntry): void {
    const metrics: PerformanceMetrics = {
      operationId: entry.name.replace('worker-operation-', ''),
      operationType: WorkerMessageType.PARSE_QUERY_RESULTS, // Would need to parse from name
      startTime: entry.startTime,
      endTime: entry.startTime + entry.duration,
      duration: entry.duration,
      memoryUsed: this.getCurrentMemoryUsage().used
    };

    this.recordMetrics(metrics);
  }

  private recordResourceTiming(entry: PerformanceResourceTiming): void {
    // Track resource loading performance (e.g., worker scripts)
    console.debug('Resource timing:', {
      name: entry.name,
      duration: entry.duration,
      transferSize: entry.transferSize
    });
  }

  // Start/Stop recording
  startRecording(): void {
    this.isRecording = true;
    this.startTime = Date.now();
    this.snapshots = [];
    this.operationMetrics.clear();
  }

  stopRecording(): void {
    this.isRecording = false;
  }

  // Record metrics
  recordMetrics(metrics: PerformanceMetrics): void {
    if (!this.isRecording) return;

    // Store operation metrics
    this.operationMetrics.set(metrics.operationId, metrics);

    // Create snapshot
    const snapshot: PerformanceSnapshot = {
      timestamp: Date.now(),
      metrics,
      memoryUsage: this.getCurrentMemoryUsage(),
      workerPoolMetrics: this.workerPool?.getMetrics()
    };

    this.snapshots.push(snapshot);

    // Trim old snapshots if needed
    if (this.snapshots.length > this.maxSnapshots) {
      this.snapshots.shift();
    }

    // Mark performance measure for PerformanceObserver
    if (typeof performance !== 'undefined' && performance.mark) {
      const markName = `worker-${metrics.operationId}`;
      performance.mark(`${markName}-start`);

      setTimeout(() => {
        performance.mark(`${markName}-end`);
        performance.measure(
          `worker-operation-${metrics.operationId}`,
          `${markName}-start`,
          `${markName}-end`
        );
      }, 0);
    }
  }

  // Generate performance report
  generateReport(): PerformanceReport {
    const endTime = Date.now();
    const duration = endTime - this.startTime;

    // Calculate execution times
    const executionTimes = this.snapshots
      .map(s => s.metrics.duration)
      .filter(d => d !== undefined)
      .sort((a, b) => a - b);

    // Calculate memory usage
    const memoryUsages = this.snapshots.map(s => s.memoryUsage.used);

    // Calculate operation breakdown
    const operationBreakdown = this.calculateOperationBreakdown();

    // Calculate time series data
    const timeSeriesData = this.generateTimeSeriesData();

    // Calculate error rate
    const totalOperations = this.snapshots.length;
    const errors = this.snapshots.filter(s =>
      s.metrics.operationType === WorkerMessageType.ERROR
    ).length;

    return {
      startTime: this.startTime,
      endTime,
      duration,
      operationsCount: totalOperations,
      averageExecutionTime: this.calculateAverage(executionTimes),
      medianExecutionTime: this.calculateMedian(executionTimes),
      p95ExecutionTime: this.calculatePercentile(executionTimes, 95),
      p99ExecutionTime: this.calculatePercentile(executionTimes, 99),
      maxExecutionTime: Math.max(...executionTimes, 0),
      minExecutionTime: Math.min(...executionTimes, Infinity),
      throughput: totalOperations / (duration / 1000), // ops/second
      memoryPeakUsage: Math.max(...memoryUsages, 0),
      memoryAverageUsage: this.calculateAverage(memoryUsages),
      errorRate: totalOperations > 0 ? errors / totalOperations : 0,
      operationBreakdown,
      timeSeriesData
    };
  }

  private calculateOperationBreakdown(): Map<WorkerMessageType, OperationStats> {
    const breakdown = new Map<WorkerMessageType, OperationStats>();

    // Group snapshots by operation type
    const grouped = new Map<WorkerMessageType, PerformanceSnapshot[]>();

    for (const snapshot of this.snapshots) {
      const type = snapshot.metrics.operationType;
      if (!grouped.has(type)) {
        grouped.set(type, []);
      }
      grouped.get(type)!.push(snapshot);
    }

    // Calculate stats for each operation type
    for (const [type, snapshots] of grouped) {
      const times = snapshots.map(s => s.metrics.duration).filter(d => d !== undefined);
      const errors = snapshots.filter(s =>
        s.metrics.operationType === WorkerMessageType.ERROR
      ).length;

      breakdown.set(type, {
        count: snapshots.length,
        totalTime: times.reduce((sum, t) => sum + t, 0),
        averageTime: this.calculateAverage(times),
        minTime: Math.min(...times, Infinity),
        maxTime: Math.max(...times, 0),
        errors,
        throughput: snapshots.length / ((Date.now() - this.startTime) / 1000)
      });
    }

    return breakdown;
  }

  private generateTimeSeriesData(): TimeSeriesData[] {
    const interval = 1000; // 1 second intervals
    const data: TimeSeriesData[] = [];
    const startTime = this.snapshots[0]?.timestamp || this.startTime;
    const endTime = this.snapshots[this.snapshots.length - 1]?.timestamp || Date.now();

    for (let time = startTime; time <= endTime; time += interval) {
      const windowSnapshots = this.snapshots.filter(s =>
        s.timestamp >= time && s.timestamp < time + interval
      );

      if (windowSnapshots.length > 0) {
        const avgExecutionTime = this.calculateAverage(
          windowSnapshots.map(s => s.metrics.duration).filter(d => d !== undefined)
        );

        const avgMemoryUsage = this.calculateAverage(
          windowSnapshots.map(s => s.memoryUsage.used)
        );

        const poolMetrics = windowSnapshots[windowSnapshots.length - 1].workerPoolMetrics;

        data.push({
          timestamp: time,
          executionTime: avgExecutionTime,
          memoryUsage: avgMemoryUsage,
          queueLength: poolMetrics?.queuedTasks || 0,
          activeWorkers: poolMetrics?.busyWorkers || 0
        });
      }
    }

    return data;
  }

  // Utility methods
  private getCurrentMemoryUsage(): MemoryUsage {
    if ('memory' in performance) {
      const memory = (performance as unknown as { memory: { usedJSHeapSize: number; totalJSHeapSize: number; jsHeapSizeLimit: number } }).memory;
      return {
        used: memory.usedJSHeapSize,
        peak: memory.totalJSHeapSize,
        limit: memory.jsHeapSizeLimit,
        percentage: (memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100
      };
    }

    return {
      used: 0,
      peak: 0,
      percentage: 0
    };
  }

  private calculateAverage(values: number[]): number {
    if (values.length === 0) return 0;
    return values.reduce((sum, v) => sum + v, 0) / values.length;
  }

  private calculateMedian(values: number[]): number {
    if (values.length === 0) return 0;
    const sorted = [...values].sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);

    if (sorted.length % 2 === 0) {
      return (sorted[mid - 1] + sorted[mid]) / 2;
    }

    return sorted[mid];
  }

  private calculatePercentile(values: number[], percentile: number): number {
    if (values.length === 0) return 0;
    const sorted = [...values].sort((a, b) => a - b);
    const index = Math.ceil((percentile / 100) * sorted.length) - 1;
    return sorted[Math.max(0, Math.min(index, sorted.length - 1))];
  }

  // Export methods
  exportMetrics(format: 'json' | 'csv' = 'json'): string {
    const report = this.generateReport();

    if (format === 'json') {
      return JSON.stringify(report, (key, value) => {
        if (value instanceof Map) {
          return Array.from(value.entries());
        }
        return value;
      }, 2);
    }

    // CSV format
    const lines: string[] = [
      'Metric,Value',
      `Start Time,${report.startTime}`,
      `End Time,${report.endTime}`,
      `Duration (ms),${report.duration}`,
      `Operations Count,${report.operationsCount}`,
      `Average Execution Time (ms),${report.averageExecutionTime}`,
      `Median Execution Time (ms),${report.medianExecutionTime}`,
      `P95 Execution Time (ms),${report.p95ExecutionTime}`,
      `P99 Execution Time (ms),${report.p99ExecutionTime}`,
      `Max Execution Time (ms),${report.maxExecutionTime}`,
      `Min Execution Time (ms),${report.minExecutionTime}`,
      `Throughput (ops/sec),${report.throughput}`,
      `Memory Peak Usage (bytes),${report.memoryPeakUsage}`,
      `Memory Average Usage (bytes),${report.memoryAverageUsage}`,
      `Error Rate,${report.errorRate}`
    ];

    return lines.join('\n');
  }

  // Real-time monitoring
  getRealtimeMetrics(): {
    currentThroughput: number;
    averageLatency: number;
    memoryUsage: MemoryUsage;
    workerPoolStatus?: WorkerPoolMetrics;
  } {
    const recentWindow = 5000; // Last 5 seconds
    const now = Date.now();
    const recentSnapshots = this.snapshots.filter(s =>
      s.timestamp > now - recentWindow
    );

    const latencies = recentSnapshots
      .map(s => s.metrics.duration)
      .filter(d => d !== undefined);

    return {
      currentThroughput: recentSnapshots.length / (recentWindow / 1000),
      averageLatency: this.calculateAverage(latencies),
      memoryUsage: this.getCurrentMemoryUsage(),
      workerPoolStatus: this.workerPool?.getMetrics()
    };
  }

  // Cleanup
  clear(): void {
    this.snapshots = [];
    this.operationMetrics.clear();
    this.startTime = Date.now();
  }

  destroy(): void {
    this.performanceObserver?.disconnect();
    this.resourceObserver?.disconnect();
    this.clear();
  }
}

// Singleton monitor instance
let defaultMonitor: PerformanceMonitor | null = null;

export function getDefaultPerformanceMonitor(workerPool?: WorkerPool): PerformanceMonitor {
  if (!defaultMonitor) {
    defaultMonitor = new PerformanceMonitor(workerPool);
  }
  return defaultMonitor;
}

export function destroyDefaultPerformanceMonitor(): void {
  if (defaultMonitor) {
    defaultMonitor.destroy();
    defaultMonitor = null;
  }
}