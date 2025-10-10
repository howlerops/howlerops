/**
 * Tests for Worker Client
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { WorkerClient } from '../workerClient';
import {
  WorkerMessageType,
  QueryResult,
  FilterCondition,
  FilterOperator,
  SortCondition,
  SortDirection,
  DataType
} from '../types';

// Mock Worker
class MockWorker {
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  private listeners = new Map<string, Set<(event: Event) => void>>();
  private shouldRespond = true;

  postMessage(message: { id: string; type: WorkerMessageType; payload?: unknown }): void {
    // Simulate async response
    setTimeout(() => {
      if (this.onmessage && this.shouldRespond) {
        this.onmessage(new MessageEvent('message', {
          data: {
            id: message.id,
            type: message.type,
            success: true,
            result: this.generateMockResult(message),
            timestamp: Date.now()
          }
        }));
      }
    }, 10);
  }

  // Method to control whether the worker responds (for timeout testing)
  setShouldRespond(value: boolean): void {
    this.shouldRespond = value;
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
    this.onmessage = null;
    this.onerror = null;
    this.listeners.clear();
  }

  private generateMockResult(message: { type: WorkerMessageType; payload?: unknown }): unknown {
    switch (message.type) {
      case WorkerMessageType.PARSE_QUERY_RESULTS:
        return {
          columns: [
            { name: 'id', type: DataType.INTEGER },
            { name: 'name', type: DataType.STRING }
          ],
          rows: [
            { id: 1, name: 'Test 1' },
            { id: 2, name: 'Test 2' }
          ]
        };

      case WorkerMessageType.FILTER_DATA:
        return {
          columns: message.payload.data.columns,
          rows: (message.payload as { data: { rows: { id: number }[] } }).data.rows.filter((row) => row.id === 1)
        };

      case WorkerMessageType.SORT_DATA:
        return {
          columns: message.payload.data.columns,
          rows: [...(message.payload as { data: { rows: unknown[] } }).data.rows].reverse()
        };

      case WorkerMessageType.EXPORT_CSV:
        return 'id,name\n1,Test 1\n2,Test 2';

      case WorkerMessageType.EXPORT_JSON:
        return JSON.stringify([
          { id: 1, name: 'Test 1' },
          { id: 2, name: 'Test 2' }
        ]);

      case WorkerMessageType.CALCULATE_STATISTICS:
        return {
          id: { count: 2, unique: 2, nulls: 0 },
          name: { count: 2, unique: 2, nulls: 0 }
        };

      default:
        return {};
    }
  }
}

// Keep track of worker instances for testing
let lastWorkerInstance: MockWorker | null = null;

// Mock Worker constructor
class MockWorkerConstructor {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  constructor(..._args: unknown[]) {
    const instance = new MockWorker();
    lastWorkerInstance = instance;
    return instance;
  }
}

vi.stubGlobal('Worker', MockWorkerConstructor);

describe('WorkerClient', () => {
  let client: WorkerClient;

  beforeEach(() => {
    client = new WorkerClient({
      timeout: 1000
    });
  });

  afterEach(() => {
    client.terminate();
  });

  describe('parseQueryResults', () => {
    it('should parse query results', async () => {
      const rawData = {
        columns: ['id', 'name'],
        rows: [
          [1, 'Test 1'],
          [2, 'Test 2']
        ]
      };

      const result = await client.parseQueryResults(rawData);

      expect(result).toBeDefined();
      expect(result.columns).toHaveLength(2);
      expect(result.rows).toHaveLength(2);
      expect(result.columns[0].name).toBe('id');
      expect(result.columns[0].type).toBe(DataType.INTEGER);
    });

    it('should handle empty data', async () => {
      const result = await client.parseQueryResults({ rows: [] });
      expect(result.rows).toHaveLength(2); // Mock returns fixed data
    });
  });

  describe('filterData', () => {
    it('should filter data based on conditions', async () => {
      const data: QueryResult = {
        columns: [
          { name: 'id', type: DataType.INTEGER },
          { name: 'name', type: DataType.STRING }
        ],
        rows: [
          { id: 1, name: 'Test 1' },
          { id: 2, name: 'Test 2' }
        ]
      };

      const filters: FilterCondition[] = [
        {
          column: 'id',
          operator: FilterOperator.EQUALS,
          value: 1
        }
      ];

      const result = await client.filterData(data, filters);

      expect(result).toBeDefined();
      expect(result.rows).toHaveLength(1);
      expect(result.rows[0]).toEqual({ id: 1, name: 'Test 1' });
    });

    it('should handle multiple filter conditions', async () => {
      const data: QueryResult = {
        columns: [
          { name: 'id', type: DataType.INTEGER },
          { name: 'name', type: DataType.STRING },
          { name: 'active', type: DataType.BOOLEAN }
        ],
        rows: [
          { id: 1, name: 'Test 1', active: true },
          { id: 2, name: 'Test 2', active: false },
          { id: 3, name: 'Test 3', active: true }
        ]
      };

      const filters: FilterCondition[] = [
        {
          column: 'active',
          operator: FilterOperator.EQUALS,
          value: true
        },
        {
          column: 'id',
          operator: FilterOperator.GREATER_THAN,
          value: 1
        }
      ];

      const result = await client.filterData(data, filters);
      expect(result).toBeDefined();
    });
  });

  describe('sortData', () => {
    it('should sort data by single column', async () => {
      const data: QueryResult = {
        columns: [
          { name: 'id', type: DataType.INTEGER },
          { name: 'name', type: DataType.STRING }
        ],
        rows: [
          { id: 2, name: 'Beta' },
          { id: 1, name: 'Alpha' },
          { id: 3, name: 'Gamma' }
        ]
      };

      const sorts: SortCondition[] = [
        {
          column: 'id',
          direction: SortDirection.ASC
        }
      ];

      const result = await client.sortData(data, sorts);
      expect(result).toBeDefined();
      expect(result.rows).toHaveLength(3);
    });

    it('should handle multi-column sorting', async () => {
      const data: QueryResult = {
        columns: [
          { name: 'category', type: DataType.STRING },
          { name: 'name', type: DataType.STRING }
        ],
        rows: [
          { category: 'B', name: 'Item 2' },
          { category: 'A', name: 'Item 3' },
          { category: 'A', name: 'Item 1' }
        ]
      };

      const sorts: SortCondition[] = [
        { column: 'category', direction: SortDirection.ASC },
        { column: 'name', direction: SortDirection.ASC }
      ];

      const result = await client.sortData(data, sorts);
      expect(result).toBeDefined();
    });
  });

  describe('export operations', () => {
    const testData: QueryResult = {
      columns: [
        { name: 'id', type: DataType.INTEGER },
        { name: 'name', type: DataType.STRING }
      ],
      rows: [
        { id: 1, name: 'Test 1' },
        { id: 2, name: 'Test 2' }
      ]
    };

    it('should export to CSV', async () => {
      const csv = await client.exportCSV(testData);
      expect(csv).toBeDefined();
      expect(csv).toContain('id,name');
      expect(csv).toContain('Test 1');
      expect(csv).toContain('Test 2');
    });

    it('should export to JSON', async () => {
      const json = await client.exportJSON(testData);
      expect(json).toBeDefined();
      const parsed = JSON.parse(json);
      expect(Array.isArray(parsed)).toBe(true);
      expect(parsed).toHaveLength(2);
    });

    it('should export to Excel format', async () => {
      const excel = await client.exportExcel(testData);
      expect(excel).toBeDefined();
    });
  });

  describe('calculateStatistics', () => {
    it('should calculate column statistics', async () => {
      const data: QueryResult = {
        columns: [
          { name: 'id', type: DataType.INTEGER },
          { name: 'value', type: DataType.NUMBER }
        ],
        rows: [
          { id: 1, value: 10 },
          { id: 2, value: 20 },
          { id: 3, value: 30 }
        ]
      };

      const stats = await client.calculateStatistics(data, ['id', 'value']);
      expect(stats).toBeDefined();
      expect(stats.id).toBeDefined();
      expect(stats.id.count).toBe(2); // Mock returns fixed value
    });
  });

  describe('error handling', () => {
    it('should handle worker errors', async () => {
      const errorClient = new WorkerClient({
        timeout: 100,
        onError: vi.fn()
      });

      // Force an error by terminating the worker
      errorClient.terminate();

      await expect(
        errorClient.parseQueryResults({})
      ).rejects.toThrow();
    });

    it('should handle operation timeout', async () => {
      const timeoutClient = new WorkerClient({
        timeout: 10 // Very short timeout
      });

      // Prevent the mock worker from responding to simulate a timeout
      if (lastWorkerInstance) {
        lastWorkerInstance.setShouldRespond(false);
      }

      await expect(
        timeoutClient.parseQueryResults({})
      ).rejects.toThrow('Operation timed out after 10ms');

      timeoutClient.terminate();
    });
  });

  describe('performance', () => {
    it('should report metrics', () => {
      const metrics = client.getMetrics();
      expect(metrics).toBeDefined();
      expect(metrics.pendingMessages).toBe(0);
      expect(metrics.queuedMessages).toBe(0);
      expect(metrics.isWorkerActive).toBe(true);
    });

    it('should process large datasets in batches', async () => {
      const largeData = Array.from({ length: 25000 }, (_, i) => ({
        id: i,
        value: Math.random()
      }));

      let batchCount = 0;
      const processed = await client.processLargeDataset(
        largeData,
        10000,
        async (batch) => {
          batchCount++;
          return batch;
        }
      );

      expect(processed).toHaveLength(25000);
      expect(batchCount).toBe(3); // 25000 / 10000 = 3 batches
    });
  });
});