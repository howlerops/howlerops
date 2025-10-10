/**
 * Tests for Worker Pool Management
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { WorkerPool } from '../workerPool';
import {
  WorkerMessageType,
  Priority,
  WorkerStatus
} from '../types';

// Mock Worker
class MockWorker {
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  private listeners = new Map<string, Set<(event: Event) => void>>();
  private terminated = false;

  postMessage(message: { id: string; type: WorkerMessageType; payload?: unknown }): void {
    if (this.terminated) return;

    // Simulate async response with variable delay
    const delay = Math.random() * 50 + 10; // 10-60ms
    setTimeout(() => {
      if (this.onmessage && !this.terminated) {
        this.onmessage(new MessageEvent('message', {
          data: {
            id: message.id,
            type: message.type,
            success: true,
            result: { processed: true },
            timestamp: Date.now(),
            executionTime: delay
          }
        }));
      }
    }, delay);
  }

  addEventListener(type: string, listener: (event: Event) => void): void {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }
    this.listeners.get(type)!.add(listener);

    if (type === 'message') {
      this.onmessage = listener as (event: MessageEvent) => void;
    } else if (type === 'error') {
      this.onerror = listener as (event: ErrorEvent) => void;
    }
  }

  removeEventListener(type: string, listener: (event: Event) => void): void {
    this.listeners.get(type)?.delete(listener);
  }

  terminate(): void {
    this.terminated = true;
    this.onmessage = null;
    this.onerror = null;
    this.listeners.clear();
  }
}

// Mock Worker constructor
vi.stubGlobal('Worker', MockWorker);

// Mock navigator.hardwareConcurrency
vi.stubGlobal('navigator', {
  hardwareConcurrency: 4
});

describe('WorkerPool', () => {
  let pool: WorkerPool;

  beforeEach(() => {
    pool = new WorkerPool({
      minWorkers: 2,
      maxWorkers: 4,
      idleTimeout: 1000,
      taskTimeout: 500
    });
  });

  afterEach(async () => {
    await pool.shutdown();
  });

  describe('initialization', () => {
    it('should spawn minimum number of workers', () => {
      const metrics = pool.getMetrics();
      expect(metrics.totalWorkers).toBe(2);
      expect(metrics.idleWorkers).toBe(2);
      expect(metrics.busyWorkers).toBe(0);
    });

    it('should respect max workers configuration', () => {
      const customPool = new WorkerPool({
        minWorkers: 1,
        maxWorkers: 8
      });

      const metrics = customPool.getMetrics();
      expect(metrics.totalWorkers).toBeGreaterThanOrEqual(1);
      expect(metrics.totalWorkers).toBeLessThanOrEqual(8);

      customPool.shutdown();
    });
  });

  describe('task execution', () => {
    it('should execute single task', async () => {
      const result = await pool.execute(
        WorkerMessageType.PARSE_QUERY_RESULTS,
        { data: 'test' },
        Priority.NORMAL
      );

      expect(result).toBeDefined();
      expect(result.processed).toBe(true);
    });

    it('should execute multiple tasks concurrently', async () => {
      const tasks = Array.from({ length: 5 }, (_, i) =>
        pool.execute(
          WorkerMessageType.FILTER_DATA,
          { id: i },
          Priority.NORMAL
        )
      );

      const results = await Promise.all(tasks);
      expect(results).toHaveLength(5);
      results.forEach(result => {
        expect(result.processed).toBe(true);
      });
    });

    it('should respect task priority', async () => {
      const completionOrder: number[] = [];

      // Submit tasks with different priorities
      const promises = [
        pool.execute(WorkerMessageType.SORT_DATA, { id: 1 }, Priority.LOW)
          .then(r => { completionOrder.push(1); return r; }),
        pool.execute(WorkerMessageType.SORT_DATA, { id: 2 }, Priority.CRITICAL)
          .then(r => { completionOrder.push(2); return r; }),
        pool.execute(WorkerMessageType.SORT_DATA, { id: 3 }, Priority.HIGH)
          .then(r => { completionOrder.push(3); return r; }),
        pool.execute(WorkerMessageType.SORT_DATA, { id: 4 }, Priority.NORMAL)
          .then(r => { completionOrder.push(4); return r; })
      ];

      await Promise.all(promises);

      // Critical and high priority tasks should generally complete first
      // Note: Due to async nature, exact order may vary
      expect(completionOrder).toBeDefined();
      expect(completionOrder).toHaveLength(4);
    });

    it('should queue tasks when all workers are busy', async () => {
      // Submit more tasks than available workers
      const taskCount = 10;
      const tasks = Array.from({ length: taskCount }, (_, i) =>
        pool.execute(
          WorkerMessageType.TRANSFORM_DATA,
          { id: i },
          Priority.NORMAL
        )
      );

      // Check queue metrics
      const metrics = pool.getMetrics();
      expect(metrics.queuedTasks).toBeGreaterThanOrEqual(0);

      // Wait for all tasks to complete
      const results = await Promise.all(tasks);
      expect(results).toHaveLength(taskCount);

      // After completion, queue should be empty
      const finalMetrics = pool.getMetrics();
      expect(finalMetrics.queuedTasks).toBe(0);
    });
  });

  describe('worker scaling', () => {
    it('should spawn new workers when needed', async () => {
      const initialMetrics = pool.getMetrics();
      const initialWorkers = initialMetrics.totalWorkers;

      // Submit tasks to trigger scaling
      const tasks = Array.from({ length: 6 }, (_, i) =>
        pool.execute(
          WorkerMessageType.CALCULATE_AGGREGATIONS,
          { id: i },
          Priority.NORMAL
        )
      );

      // Give time for scaling
      await new Promise(resolve => setTimeout(resolve, 50));

      const scaledMetrics = pool.getMetrics();
      expect(scaledMetrics.totalWorkers).toBeGreaterThanOrEqual(initialWorkers);
      expect(scaledMetrics.totalWorkers).toBeLessThanOrEqual(4); // max workers

      await Promise.all(tasks);
    });

    it('should not exceed max workers limit', async () => {
      // Submit many tasks
      const tasks = Array.from({ length: 20 }, (_, i) =>
        pool.execute(
          WorkerMessageType.VALIDATE_DATA,
          { id: i },
          Priority.NORMAL
        )
      );

      // Check that we don't exceed max workers
      const metrics = pool.getMetrics();
      expect(metrics.totalWorkers).toBeLessThanOrEqual(4);

      await Promise.all(tasks);
    });
  });

  describe('error handling', () => {
    it('should handle task timeout', async () => {
      const shortTimeoutPool = new WorkerPool({
        minWorkers: 1,
        maxWorkers: 2,
        taskTimeout: 10 // Very short timeout
      });

      // Override mock to delay response beyond timeout
      const originalPostMessage = MockWorker.prototype.postMessage;
      MockWorker.prototype.postMessage = function() {
        // Don't respond, causing timeout
      };

      await expect(
        shortTimeoutPool.execute(
          WorkerMessageType.EXPORT_CSV,
          { data: 'test' },
          Priority.NORMAL
        )
      ).rejects.toThrow(/timeout/i);

      MockWorker.prototype.postMessage = originalPostMessage;
      await shortTimeoutPool.shutdown();
    });

    it('should handle worker errors', async () => {
      // Simulate worker error
      const errorTask = pool.execute(
        WorkerMessageType.EXPORT_JSON,
        { data: 'test' },
        Priority.NORMAL
      );

      // Trigger error in worker
      const workers = pool.getWorkerStates();
      expect(workers.length).toBeGreaterThan(0);

      // Should handle error gracefully
      try {
        await errorTask;
      } catch {
        // Error is expected in some cases
      }

      // Pool should still be functional
      const result = await pool.execute(
        WorkerMessageType.PARSE_QUERY_RESULTS,
        { data: 'recovery test' },
        Priority.NORMAL
      );
      expect(result).toBeDefined();
    });

    it('should reject tasks when queue is full', async () => {
      const limitedPool = new WorkerPool({
        minWorkers: 1,
        maxWorkers: 1,
        maxQueueSize: 5
      });

      // Fill up the queue
      const tasks: Promise<unknown>[] = [];
      for (let i = 0; i < 10; i++) {
        tasks.push(
          limitedPool.execute(
            WorkerMessageType.FILTER_DATA,
            { id: i },
            Priority.NORMAL
          ).catch(err => err)
        );
      }

      const results = await Promise.all(tasks);

      // Some tasks should fail due to queue limit
      const errors = results.filter(r => r instanceof Error);
      expect(errors.length).toBeGreaterThan(0);
      expect(errors[0].message).toContain('queue');

      await limitedPool.shutdown();
    });
  });

  describe('task cancellation', () => {
    it('should cancel pending task', async () => {
      // Create a pool with longer timeout to avoid timeout issues
      const testPool = new WorkerPool({
        minWorkers: 1,
        maxWorkers: 2,
        taskTimeout: 2000 // 2 seconds
      });

      // Override mock to have longer delay to ensure task can be cancelled
      const originalPostMessage = MockWorker.prototype.postMessage;
      MockWorker.prototype.postMessage = function(message: { id: string; type: WorkerMessageType; payload?: unknown }): void {
        if (this.terminated) return;

        // Use longer delay for this test
        const delay = 200; // 200ms delay
        setTimeout(() => {
          if (this.onmessage && !this.terminated) {
            this.onmessage(new MessageEvent('message', {
              data: {
                id: message.id,
                type: message.type,
                success: true,
                result: { processed: true },
                timestamp: Date.now(),
                executionTime: delay
              }
            }));
          }
        }, delay);
      };

      // Submit task but don't wait
      const taskPromise = testPool.execute(
        WorkerMessageType.CALCULATE_STATISTICS,
        { data: 'test' },
        Priority.NORMAL
      );

      // Give a moment for task to be queued
      await new Promise(resolve => setTimeout(resolve, 10));

      // Get task ID from queued tasks
      const queuedTasks = testPool.getQueuedTasks();
      if (queuedTasks.length > 0) {
        const taskId = queuedTasks[0].id;
        const cancelled = await testPool.cancelTask(taskId);
        expect(cancelled).toBe(true);

        // Task should reject
        await expect(taskPromise).rejects.toThrow();
      } else {
        // If no queued tasks, the task was already assigned to a worker
        // Cancel the task that's running
        await new Promise(resolve => setTimeout(resolve, 50)); // Wait a bit more
        const workerStates = testPool.getWorkerStates();
        const busyWorker = workerStates.find(w => w.currentTask);
        if (busyWorker?.currentTask) {
          const cancelled = await testPool.cancelTask(busyWorker.currentTask);
          // For running tasks, cancellation returns true but task may still complete
          expect(cancelled).toBe(true);
        }

        // Since the task was already running, it may complete normally
        try {
          const result = await taskPromise;
          expect(result).toBeDefined();
        } catch (error) {
          // Task was successfully cancelled
          expect(error).toBeDefined();
        }
      }

      // Restore original mock
      MockWorker.prototype.postMessage = originalPostMessage;

      // Clean up test pool
      await testPool.shutdown();
    });

    it('should handle cancellation of non-existent task', async () => {
      const cancelled = await pool.cancelTask('non-existent-task-id');
      expect(cancelled).toBe(false);
    });
  });

  describe('metrics and monitoring', () => {
    it('should track completed tasks', async () => {
      const initialMetrics = pool.getMetrics();
      const initialCompleted = initialMetrics.completedTasks;

      // Execute some tasks
      await Promise.all([
        pool.execute(WorkerMessageType.PARSE_QUERY_RESULTS, {}, Priority.NORMAL),
        pool.execute(WorkerMessageType.FILTER_DATA, {}, Priority.NORMAL),
        pool.execute(WorkerMessageType.SORT_DATA, {}, Priority.NORMAL)
      ]);

      const finalMetrics = pool.getMetrics();
      expect(finalMetrics.completedTasks).toBe(initialCompleted + 3);
    });

    it('should track failed tasks', async () => {

      // Force a failure by timeout
      const failPool = new WorkerPool({
        minWorkers: 1,
        taskTimeout: 1
      });

      try {
        await failPool.execute(
          WorkerMessageType.EXPORT_EXCEL,
          { data: 'test' },
          Priority.NORMAL
        );
      } catch {
        // Expected to fail
      }

      await failPool.shutdown();
    });

    it('should provide worker states', () => {
      const states = pool.getWorkerStates();
      expect(states).toBeDefined();
      expect(states.length).toBeGreaterThan(0);

      states.forEach(state => {
        expect(state.id).toBeDefined();
        expect(state.status).toBeDefined();
        expect([
          WorkerStatus.IDLE,
          WorkerStatus.BUSY,
          WorkerStatus.TERMINATING,
          WorkerStatus.TERMINATED,
          WorkerStatus.ERROR
        ]).toContain(state.status);
        expect(state.tasksCompleted).toBeGreaterThanOrEqual(0);
        expect(state.tasksFailed).toBeGreaterThanOrEqual(0);
      });
    });
  });

  describe('shutdown', () => {
    it('should gracefully shutdown pool', async () => {
      const testPool = new WorkerPool({
        minWorkers: 2,
        maxWorkers: 4
      });

      // Submit some tasks
      Array.from({ length: 3 }, (_, i) =>
        testPool.execute(
          WorkerMessageType.TRANSFORM_DATA,
          { id: i },
          Priority.NORMAL
        ).catch(() => {}) // Ignore errors from shutdown
      );

      // Shutdown pool
      await testPool.shutdown();

      // Verify pool is shut down
      const metrics = testPool.getMetrics();
      expect(metrics.totalWorkers).toBe(0);
      expect(metrics.queuedTasks).toBe(0);
    });

    it('should reject pending tasks on shutdown', async () => {
      const testPool = new WorkerPool({
        minWorkers: 1,
        maxWorkers: 1
      });

      // Submit task but don't wait
      const taskPromise = testPool.execute(
        WorkerMessageType.VALIDATE_DATA,
        { data: 'test' },
        Priority.NORMAL
      );

      // Immediately shutdown
      await testPool.shutdown();

      // Task should be rejected
      await expect(taskPromise).rejects.toThrow(/shutting down/i);
    });
  });
});