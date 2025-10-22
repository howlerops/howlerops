/**
 * HowlerOps Web Workers - Usage Examples
 * Demonstrates how to use the worker infrastructure for data processing
 */

import {
  getWorkerManager,
  parseQueryResults,
  filterData,
  sortData,
  exportData,
  calculateStatistics,
  SQLStudioWorkers,
  QueryResult,
  FilterCondition,
  FilterOperator,
  SortCondition,
  SortDirection,
  ExportConfig,
  ExportFormat,
  AggregationConfig,
  AggregationOperation,
  ValidationRule,
  ValidationType,
  TransformationRule,
  TransformationType,
  DataType
} from './index';

// Example 1: Basic Usage with Convenience Functions
// eslint-disable-next-line @typescript-eslint/no-unused-vars
async function basicUsageExample() {
  console.log('=== Basic Usage Example ===');

  // Parse raw query results
  const rawData = {
    columns: ['id', 'name', 'age', 'department', 'salary'],
    rows: [
      [1, 'Alice', 30, 'Engineering', 75000],
      [2, 'Bob', 25, 'Marketing', 60000],
      [3, 'Charlie', 35, 'Engineering', 85000],
      [4, 'Diana', 28, 'Sales', 65000],
      [5, 'Eve', 32, 'Engineering', 80000]
    ]
  };

  const parsedData = await parseQueryResults(rawData);
  console.log('Parsed data:', parsedData);

  // Filter data
  const filters: FilterCondition[] = [
    {
      column: 'department',
      operator: FilterOperator.EQUALS,
      value: 'Engineering'
    },
    {
      column: 'salary',
      operator: FilterOperator.GREATER_THAN,
      value: 70000
    }
  ];

  const filteredData = await filterData(parsedData, filters);
  console.log('Filtered data:', filteredData);

  // Sort data
  const sorts: SortCondition[] = [
    {
      column: 'salary',
      direction: SortDirection.DESC
    }
  ];

  const sortedData = await sortData(filteredData, sorts);
  console.log('Sorted data:', sortedData);

  // Export to CSV
  const csvConfig: ExportConfig = {
    format: ExportFormat.CSV,
    includeHeaders: true,
    delimiter: ',',
    quote: '"'
  };

  const csv = await exportData(sortedData, csvConfig);
  console.log('CSV export:', csv);

  // Calculate statistics
  const stats = await calculateStatistics(parsedData, ['age', 'salary']);
  console.log('Statistics:', stats);
}

// Example 2: Advanced Usage with Worker Manager
// eslint-disable-next-line @typescript-eslint/no-unused-vars
async function advancedUsageExample() {
  console.log('=== Advanced Usage Example ===');

  // Initialize worker manager with custom configuration
  const manager = SQLStudioWorkers.getInstance({
    minWorkers: 2,
    maxWorkers: 8,
    idleTimeout: 30000,
    enableSharedArrayBuffer: true
  });

  await manager.initialize();

  // Create sample dataset
  const data: QueryResult = {
    columns: [
      { name: 'id', type: DataType.INTEGER },
      { name: 'product', type: DataType.STRING },
      { name: 'category', type: DataType.STRING },
      { name: 'price', type: DataType.FLOAT },
      { name: 'quantity', type: DataType.INTEGER },
      { name: 'date', type: DataType.DATE }
    ],
    rows: generateSampleData(10000) // Large dataset
  };

  // 1. Data Validation
  const validationRules: ValidationRule[] = [
    {
      column: 'price',
      type: ValidationType.MIN_VALUE,
      config: 0,
      errorMessage: 'Price must be positive'
    },
    {
      column: 'quantity',
      type: ValidationType.REQUIRED,
      errorMessage: 'Quantity is required'
    },
    {
      column: 'product',
      type: ValidationType.MAX_LENGTH,
      config: 100,
      errorMessage: 'Product name too long'
    }
  ];

  const validationResult = await manager.validateData(data, validationRules) as {
    valid: boolean;
    errors: unknown[];
    summary: { errorRate: number };
  };
  console.log('Validation result:', {
    valid: validationResult.valid,
    errorCount: validationResult.errors.length,
    errorRate: validationResult.summary.errorRate
  });

  // 2. Data Transformation
  const transformations: TransformationRule[] = [
    {
      column: 'product',
      type: TransformationType.UPPERCASE
    },
    {
      column: 'price',
      type: TransformationType.NUMBER_FORMAT,
      config: { decimals: 2 }
    },
    {
      column: 'date',
      type: TransformationType.DATE_FORMAT,
      config: { year: 'numeric', month: 'short', day: 'numeric' }
    }
  ];

  const transformedData = await manager.transformData(data, transformations);
  console.log('Transformed data sample:', transformedData.rows.slice(0, 3));

  // 3. Aggregations
  const aggregationConfig: AggregationConfig = {
    groupBy: ['category'],
    aggregations: [
      {
        column: 'price',
        operation: AggregationOperation.AVG,
        alias: 'avg_price'
      },
      {
        column: 'quantity',
        operation: AggregationOperation.SUM,
        alias: 'total_quantity'
      },
      {
        column: 'id',
        operation: AggregationOperation.COUNT,
        alias: 'product_count'
      }
    ]
  };

  const aggregations = await manager.calculateAggregations(data, aggregationConfig);
  console.log('Aggregations:', aggregations);

  // 4. Batch Processing for Very Large Datasets
  const veryLargeDataset = generateSampleData(100000);

  const processedBatches = await manager.processBatchData(
    veryLargeDataset,
    async (batch: unknown[]) => {
      // Process each batch
      const typedBatch = batch as Array<{ price: number; quantity: number }>;
      return typedBatch.map(row => ({
        ...row,
        total: row.price * row.quantity
      }));
    },
    10000 // Process 10k rows at a time
  );

  console.log('Batch processing complete:', {
    totalRows: processedBatches.length
  });

  // 5. Performance Metrics
  const metrics = manager.getPerformanceMetrics();
  console.log('Performance metrics:', {
    realtime: metrics.realtime,
    poolStatus: metrics.pool,
    avgExecutionTime: metrics.report.averageExecutionTime,
    throughput: metrics.report.throughput
  });

  // Export performance report
  manager.exportPerformanceReport('json');
  console.log('Performance report exported');

  // Cleanup
  await manager.shutdown();
}

// Example 3: Real-time Data Processing with Progress
// eslint-disable-next-line @typescript-eslint/no-unused-vars
async function realtimeProcessingExample() {
  console.log('=== Real-time Processing Example ===');

  const manager = getWorkerManager();
  await manager.initialize();

  // Monitor performance in real-time
  const monitoringInterval = setInterval(() => {
    const status = manager.getWorkerPoolStatus();
    console.log('Worker Pool Status:', {
      activeWorkers: status.metrics.busyWorkers,
      queueLength: status.metrics.queuedTasks,
      completed: status.metrics.completedTasks
    });
  }, 1000);

  // Process streaming data
  const streamData = async () => {
    for (let i = 0; i < 10; i++) {
      const batchData: QueryResult = {
        columns: [
          { name: 'timestamp', type: DataType.DATETIME },
          { name: 'value', type: DataType.FLOAT },
          { name: 'sensor', type: DataType.STRING }
        ],
        rows: generateStreamData(1000)
      };

      // Process each batch
      const stats = await manager.calculateStatistics(batchData, ['value']) as {
        value: {
          numeric?: {
            mean?: number;
            max?: number;
            min?: number;
          };
        };
      };
      console.log(`Batch ${i + 1} stats:`, {
        mean: stats.value.numeric?.mean,
        max: stats.value.numeric?.max,
        min: stats.value.numeric?.min
      });

      // Simulate real-time delay
      await new Promise(resolve => setTimeout(resolve, 500));
    }
  };

  await streamData();

  // Stop monitoring
  clearInterval(monitoringInterval);

  // Final metrics
  const finalMetrics = manager.getPerformanceMetrics();
  console.log('Final processing metrics:', {
    totalOperations: finalMetrics.report.operationsCount,
    averageLatency: finalMetrics.report.averageExecutionTime,
    p95Latency: finalMetrics.report.p95ExecutionTime
  });

  await manager.shutdown();
}

// Example 4: Error Handling and Recovery
// eslint-disable-next-line @typescript-eslint/no-unused-vars
async function errorHandlingExample() {
  console.log('=== Error Handling Example ===');

  const manager = getWorkerManager();
  await manager.initialize();

  try {
    // Attempt to process invalid data
    const invalidData = {
      columns: null,
      rows: 'invalid'
    };

    await manager.parseQueryResults(invalidData);
  } catch {
    console.log('Caught error');
    console.log('Worker pool is still operational');
  }

  // Cancel long-running operation
  const longOperation = manager.calculateStatistics(
    generateLargeDataset(50000),
    ['col1', 'col2', 'col3']
  );

  // Cancel after 100ms
  setTimeout(async () => {
    // Note: Need to track operation ID for cancellation
    console.log('Cancelling operation...');
  }, 100);

  try {
    await longOperation;
  } catch {
    console.log('Operation cancelled or failed');
  }

  await manager.shutdown();
}

// Helper Functions

function generateSampleData(count: number): Array<{
  id: number;
  product: string;
  category: string;
  price: number;
  quantity: number;
  date: Date;
}> {
  const categories = ['Electronics', 'Clothing', 'Food', 'Books', 'Sports'];
  const products = ['Product A', 'Product B', 'Product C', 'Product D', 'Product E'];

  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    product: products[Math.floor(Math.random() * products.length)],
    category: categories[Math.floor(Math.random() * categories.length)],
    price: Math.random() * 1000,
    quantity: Math.floor(Math.random() * 100),
    date: new Date(2024, Math.floor(Math.random() * 12), Math.floor(Math.random() * 28) + 1)
  }));
}

function generateStreamData(count: number): Array<{
  timestamp: Date;
  value: number;
  sensor: string;
}> {
  const sensors = ['sensor-1', 'sensor-2', 'sensor-3'];

  return Array.from({ length: count }, () => ({
    timestamp: new Date(),
    value: Math.random() * 100,
    sensor: sensors[Math.floor(Math.random() * sensors.length)]
  }));
}

function generateLargeDataset(rows: number): QueryResult {
  const columns = Array.from({ length: 10 }, (_, i) => ({
    name: `col${i + 1}`,
    type: i % 2 === 0 ? DataType.NUMBER : DataType.STRING
  }));

  const data = Array.from({ length: rows }, () => {
    const row: Record<string, string | number> = {};
    columns.forEach((col, i) => {
      row[col.name] = i % 2 === 0 ? Math.random() * 1000 : `value-${Math.random()}`;
    });
    return row;
  });

  return { columns, rows: data };
}

// Run examples (uncomment to test)
// basicUsageExample().catch(console.error);
// advancedUsageExample().catch(console.error);
// realtimeProcessingExample().catch(console.error);
// errorHandlingExample().catch(console.error);