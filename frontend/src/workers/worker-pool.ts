/**
 * Worker Pool Management System
 * Manages dynamic worker spawning, task queuing, and load balancing
 */

import {
  WorkerPoolConfig,
  WorkerState,
  WorkerStatus,
  QueuedTask,
  TaskStatus,
  Priority,
  WorkerResponse,
  MemoryUsage,
  WorkerMessageType
} from './types';

export interface PooledWorker {
  id: string;
  worker: Worker;
  state: WorkerState;
  messageHandler?: (event: MessageEvent) => void;
  errorHandler?: (event: ErrorEvent) => void;
}

export interface WorkerPoolMetrics {
  totalWorkers: number;
  idleWorkers: number;
  busyWorkers: number;
  queuedTasks: number;
  completedTasks: number;
  failedTasks: number;
  averageExecutionTime: number;
  averageThroughput: number;
  memoryUsage: MemoryUsage[];
  cpuUsage: number[];
}

export class WorkerPool {
  private config: Required<WorkerPoolConfig>;
  private workers: Map<string, PooledWorker> = new Map();
  private taskQueue: QueuedTask[] = [];
  private pendingTasks: Map<string, {
    resolve: (value: unknown) => void;
    reject: (error: unknown) => void;
    timeout?: NodeJS.Timeout;
  }> = new Map();

  private metrics: {
    tasksCompleted: number;
    tasksFailed: number;
    totalExecutionTime: number;
    totalThroughput: number;
  } = {
    tasksCompleted: 0,
    tasksFailed: 0,
    totalExecutionTime: 0,
    totalThroughput: 0
  };

  private idleCheckInterval?: NodeJS.Timeout;
  private metricsInterval?: NodeJS.Timeout;
  private isShuttingDown = false;

  constructor(config: Partial<WorkerPoolConfig> = {}) {
    this.config = {
      minWorkers: 2,
      maxWorkers: navigator.hardwareConcurrency || 4,
      idleTimeout: 30000, // 30 seconds
      taskTimeout: 60000, // 60 seconds
      maxQueueSize: 1000,
      enableSharedArrayBuffer: typeof SharedArrayBuffer !== 'undefined',
      workerScript: new URL('./dataProcessor.worker.ts', import.meta.url).href,
      ...config
    };

    this.initialize();
  }

  private async initialize(): Promise<void> {
    // Spawn initial workers
    for (let i = 0; i < this.config.minWorkers; i++) {
      await this.spawnWorker();
    }

    // Start idle check interval
    this.idleCheckInterval = setInterval(() => {
      this.checkIdleWorkers();
    }, 5000);

    // Start metrics collection interval
    this.metricsInterval = setInterval(() => {
      this.collectMetrics();
    }, 10000);
  }

  private async spawnWorker(): Promise<PooledWorker> {
    const workerId = this.generateWorkerId();

    try {
      const worker = new Worker(this.config.workerScript, {
        type: 'module',
        name: `sql-studio-worker-${workerId}`
      });

      const pooledWorker: PooledWorker = {
        id: workerId,
        worker,
        state: {
          id: workerId,
          status: WorkerStatus.IDLE,
          tasksCompleted: 0,
          tasksFailed: 0,
          createdAt: Date.now(),
          lastActivityAt: Date.now()
        }
      };

      // Set up message handler
      pooledWorker.messageHandler = (event: MessageEvent<WorkerResponse>) => {
        this.handleWorkerMessage(workerId, event.data);
      };

      // Set up error handler
      pooledWorker.errorHandler = (event: ErrorEvent) => {
        this.handleWorkerError(workerId, event);
      };

      worker.addEventListener('message', pooledWorker.messageHandler);
      worker.addEventListener('error', pooledWorker.errorHandler);

      this.workers.set(workerId, pooledWorker);

      console.log(`Worker ${workerId} spawned. Total workers: ${this.workers.size}`);

      return pooledWorker;
    } catch (error) {
      console.error(`Failed to spawn worker ${workerId}:`, error);
      throw error;
    }
  }

  private terminateWorker(workerId: string): void {
    const pooledWorker = this.workers.get(workerId);
    if (pooledWorker) {
      pooledWorker.state.status = WorkerStatus.TERMINATING;

      // Remove event listeners
      if (pooledWorker.messageHandler) {
        pooledWorker.worker.removeEventListener('message', pooledWorker.messageHandler);
      }
      if (pooledWorker.errorHandler) {
        pooledWorker.worker.removeEventListener('error', pooledWorker.errorHandler);
      }

      // Terminate worker
      pooledWorker.worker.terminate();
      pooledWorker.state.status = WorkerStatus.TERMINATED;

      this.workers.delete(workerId);

      console.log(`Worker ${workerId} terminated. Total workers: ${this.workers.size}`);
    }
  }

  private handleWorkerMessage(workerId: string, response: WorkerResponse): void {
    const pooledWorker = this.workers.get(workerId);
    if (!pooledWorker) return;

    // Update worker state
    pooledWorker.state.lastActivityAt = Date.now();

    // Handle progress updates
    if (response.type === WorkerMessageType.PROGRESS_UPDATE) {
      // Forward progress to the original task requester
      // This could be implemented with an event emitter
      return;
    }

    // Handle task completion
    const pending = this.pendingTasks.get(response.id);
    if (pending) {
      if (pending.timeout) {
        clearTimeout(pending.timeout);
      }
      this.pendingTasks.delete(response.id);

      if (response.success) {
        pooledWorker.state.tasksCompleted++;
        this.metrics.tasksCompleted++;
        pending.resolve(response.result);
      } else {
        pooledWorker.state.tasksFailed++;
        this.metrics.tasksFailed++;
        pending.reject(response.error || new Error('Task failed'));
      }

      // Update metrics
      if (response.executionTime) {
        this.metrics.totalExecutionTime += response.executionTime;
      }

      // Mark worker as idle and process next task
      pooledWorker.state.status = WorkerStatus.IDLE;
      pooledWorker.state.currentTask = undefined;
      this.processNextTask();
    }
  }

  private handleWorkerError(workerId: string, event: ErrorEvent): void {
    console.error(`Worker ${workerId} error:`, event);

    const pooledWorker = this.workers.get(workerId);
    if (pooledWorker) {
      pooledWorker.state.status = WorkerStatus.ERROR;

      // Fail current task if any
      if (pooledWorker.state.currentTask) {
        const pending = this.pendingTasks.get(pooledWorker.state.currentTask);
        if (pending) {
          pending.reject(new Error(`Worker error: ${event.message}`));
          this.pendingTasks.delete(pooledWorker.state.currentTask);
        }
      }

      // Terminate and respawn worker
      this.terminateWorker(workerId);
      if (this.workers.size < this.config.minWorkers && !this.isShuttingDown) {
        this.spawnWorker();
      }
    }
  }

  public async execute<T = unknown>(
    type: WorkerMessageType,
    payload: unknown,
    priority: Priority = Priority.NORMAL,
    transferable?: Transferable[]
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      // Check queue size
      if (this.taskQueue.length >= this.config.maxQueueSize) {
        reject(new Error('Task queue is full'));
        return;
      }

      const taskId = this.generateTaskId();

      // Create queued task
      const task: QueuedTask = {
        id: taskId,
        message: {
          id: taskId,
          type,
          payload,
          timestamp: Date.now(),
          priority,
          transferable
        },
        priority,
        retries: 0,
        maxRetries: 3,
        createdAt: Date.now(),
        status: TaskStatus.PENDING
      };

      // Store pending promise handlers
      this.pendingTasks.set(taskId, {
        resolve,
        reject,
        timeout: setTimeout(() => {
          this.handleTaskTimeout(taskId);
        }, this.config.taskTimeout)
      });

      // Add to queue
      this.enqueueTask(task);

      // Try to process immediately
      this.processNextTask();
    });
  }

  private enqueueTask(task: QueuedTask): void {
    // Insert task based on priority
    let inserted = false;
    for (let i = 0; i < this.taskQueue.length; i++) {
      if (task.priority > this.taskQueue[i].priority) {
        this.taskQueue.splice(i, 0, task);
        inserted = true;
        break;
      }
    }

    if (!inserted) {
      this.taskQueue.push(task);
    }
  }

  private async processNextTask(): Promise<void> {
    if (this.isShuttingDown || this.taskQueue.length === 0) {
      return;
    }

    // Find idle worker
    const idleWorker = this.findIdleWorker();

    if (!idleWorker) {
      // Try to spawn new worker if below max and not shutting down
      if (this.workers.size < this.config.maxWorkers && !this.isShuttingDown) {
        try {
          const newWorker = await this.spawnWorker();
          const task = this.taskQueue.shift();
          if (task) {
            this.assignTaskToWorker(task, newWorker);
          }
        } catch (error) {
          console.error('Failed to spawn new worker:', error);
        }
      }
      return;
    }

    // Assign task to idle worker
    const task = this.taskQueue.shift();
    if (task) {
      this.assignTaskToWorker(task, idleWorker);
    }
  }

  private assignTaskToWorker(task: QueuedTask, pooledWorker: PooledWorker): void {
    task.status = TaskStatus.RUNNING;
    task.startedAt = Date.now();
    task.assignedWorker = pooledWorker.id;

    pooledWorker.state.status = WorkerStatus.BUSY;
    pooledWorker.state.currentTask = task.id;
    pooledWorker.state.lastActivityAt = Date.now();

    // Send message to worker
    pooledWorker.worker.postMessage(
      task.message,
      task.message.transferable || []
    );
  }

  private findIdleWorker(): PooledWorker | undefined {
    for (const [, worker] of this.workers) {
      if (worker.state.status === WorkerStatus.IDLE) {
        return worker;
      }
    }
    return undefined;
  }

  private handleTaskTimeout(taskId: string): void {
    const pending = this.pendingTasks.get(taskId);
    if (pending) {
      this.pendingTasks.delete(taskId);
      this.metrics.tasksFailed++;
      pending.reject(new Error('Task timeout'));
    }

    // Remove from queue if still pending
    const queueIndex = this.taskQueue.findIndex(t => t.id === taskId);
    if (queueIndex !== -1) {
      this.taskQueue.splice(queueIndex, 1);
    }

    // Find worker with this task and reset it
    for (const [, worker] of this.workers) {
      if (worker.state.currentTask === taskId) {
        worker.state.status = WorkerStatus.IDLE;
        worker.state.currentTask = undefined;
        worker.state.tasksFailed++;
        break;
      }
    }
  }

  private checkIdleWorkers(): void {
    const now = Date.now();
    const idleWorkers: string[] = [];

    for (const [, worker] of this.workers) {
      if (
        worker.state.status === WorkerStatus.IDLE &&
        now - worker.state.lastActivityAt > this.config.idleTimeout &&
        this.workers.size > this.config.minWorkers
      ) {
        idleWorkers.push(worker.id);
      }
    }

    // Terminate excess idle workers
    for (const workerId of idleWorkers) {
      this.terminateWorker(workerId);
    }
  }

  private collectMetrics(): void {
    // Collect memory usage from workers if available
    for (const [, worker] of this.workers) {
      // This would require workers to report their memory usage
      // For now, we'll use placeholder values
      if (worker.state.memoryUsage) {
        // Store or aggregate memory usage
      }
    }
  }

  // Public methods

  public getMetrics(): WorkerPoolMetrics {
    const idleWorkers = Array.from(this.workers.values()).filter(
      w => w.state.status === WorkerStatus.IDLE
    ).length;

    const busyWorkers = Array.from(this.workers.values()).filter(
      w => w.state.status === WorkerStatus.BUSY
    ).length;

    const avgExecutionTime = this.metrics.tasksCompleted > 0
      ? this.metrics.totalExecutionTime / this.metrics.tasksCompleted
      : 0;

    const avgThroughput = this.metrics.tasksCompleted > 0
      ? this.metrics.totalThroughput / this.metrics.tasksCompleted
      : 0;

    return {
      totalWorkers: this.workers.size,
      idleWorkers,
      busyWorkers,
      queuedTasks: this.taskQueue.length,
      completedTasks: this.metrics.tasksCompleted,
      failedTasks: this.metrics.tasksFailed,
      averageExecutionTime: avgExecutionTime,
      averageThroughput: avgThroughput,
      memoryUsage: [],
      cpuUsage: []
    };
  }

  public getWorkerStates(): WorkerState[] {
    return Array.from(this.workers.values()).map(w => w.state);
  }

  public getQueuedTasks(): QueuedTask[] {
    return [...this.taskQueue];
  }

  public async cancelTask(taskId: string): Promise<boolean> {
    // Remove from queue if pending
    const queueIndex = this.taskQueue.findIndex(t => t.id === taskId);
    if (queueIndex !== -1) {
      this.taskQueue.splice(queueIndex, 1);
      const pending = this.pendingTasks.get(taskId);
      if (pending) {
        if (pending.timeout) clearTimeout(pending.timeout);
        pending.reject(new Error('Task cancelled'));
        this.pendingTasks.delete(taskId);
      }
      return true;
    }

    // Send cancellation message to worker if running
    for (const [, worker] of this.workers) {
      if (worker.state.currentTask === taskId) {
        worker.worker.postMessage({
          id: taskId,
          type: WorkerMessageType.CANCEL_OPERATION,
          payload: { operationId: taskId },
          timestamp: Date.now()
        });
        return true;
      }
    }

    return false;
  }

  public async shutdown(): Promise<void> {
    this.isShuttingDown = true;

    // Clear intervals
    if (this.idleCheckInterval) {
      clearInterval(this.idleCheckInterval);
    }
    if (this.metricsInterval) {
      clearInterval(this.metricsInterval);
    }

    // Reject all pending tasks
    for (const [, pending] of this.pendingTasks) {
      if (pending.timeout) clearTimeout(pending.timeout);
      pending.reject(new Error('Worker pool shutting down'));
    }
    this.pendingTasks.clear();

    // Clear queue
    this.taskQueue = [];

    // Terminate all workers
    const workerIds = Array.from(this.workers.keys());
    for (const id of workerIds) {
      this.terminateWorker(id);
    }

    // Wait a brief moment for any async termination to complete
    await new Promise(resolve => setTimeout(resolve, 10));

    // Ensure workers map is cleared
    this.workers.clear();
  }

  private generateWorkerId(): string {
    return `worker-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateTaskId(): string {
    return `task-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Singleton pool instance
let defaultPool: WorkerPool | null = null;

export function getDefaultWorkerPool(config?: Partial<WorkerPoolConfig>): WorkerPool {
  if (!defaultPool) {
    defaultPool = new WorkerPool(config);
  }
  return defaultPool;
}

export async function shutdownDefaultWorkerPool(): Promise<void> {
  if (defaultPool) {
    await defaultPool.shutdown();
    defaultPool = null;
  }
}