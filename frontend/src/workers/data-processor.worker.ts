/**
 * Data Processing Web Worker
 * Handles heavy computations for HowlerOps in a separate thread
 */

import {
  WorkerMessage,
  WorkerResponse,
  WorkerMessageType,
  QueryResult,
  FilterCondition,
  FilterOperator,
  SortCondition,
  SortDirection,
  ExportConfig,
  AggregationConfig,
  AggregationOperation,
  ValidationRule,
  ValidationType,
  TransformationRule,
  TransformationType,
  ProgressUpdate,
  PerformanceMetrics,
  MemoryUsage
} from './types';

// Worker context
const ctx: Worker = self as unknown as Worker;

// Operation cancellation support
const abortControllers = new Map<string, AbortController>();

// Performance tracking
// const _performanceMetrics = new Map<string, PerformanceMetrics>();

// Message handler
ctx.addEventListener('message', async (event: MessageEvent<WorkerMessage>) => {
  const message = event.data;
  const startTime = performance.now();

  try {
    // Handle cancellation
    if (message.type === WorkerMessageType.CANCEL_OPERATION) {
      handleCancellation(message.payload.operationId);
      return;
    }

    // Create abort controller for this operation
    const abortController = new AbortController();
    abortControllers.set(message.id, abortController);

    // Process message based on type
    const result = await processMessage(message, abortController.signal);

    // Calculate execution metrics
    const endTime = performance.now();
    const metrics: PerformanceMetrics = {
      operationId: message.id,
      operationType: message.type,
      startTime,
      endTime,
      duration: endTime - startTime,
      memoryUsed: getMemoryUsage().used
    };

    // Send success response
    const response: WorkerResponse = {
      id: message.id,
      type: message.type,
      success: true,
      result,
      timestamp: Date.now(),
      executionTime: metrics.duration,
      memoryUsage: getMemoryUsage()
    };

    ctx.postMessage(response, message.transferable || []);

    // Cleanup
    abortControllers.delete(message.id);

  } catch (error) {
    // Send error response
    const errorResponse: WorkerResponse = {
      id: message.id,
      type: WorkerMessageType.ERROR,
      success: false,
      error: {
        code: 'WORKER_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined
      },
      timestamp: Date.now(),
      executionTime: performance.now() - startTime
    };

    ctx.postMessage(errorResponse);
    abortControllers.delete(message.id);
  }
});

async function processMessage(message: WorkerMessage, signal: AbortSignal): Promise<unknown> {
  switch (message.type) {
    case WorkerMessageType.PARSE_QUERY_RESULTS:
      return parseQueryResults(message.payload, signal, message.id);

    case WorkerMessageType.FILTER_DATA:
      return filterData(message.payload, signal, message.id);

    case WorkerMessageType.SORT_DATA:
      return sortData(message.payload, signal, message.id);

    case WorkerMessageType.EXPORT_CSV:
      return exportCSV(message.payload, signal, message.id);

    case WorkerMessageType.EXPORT_JSON:
      return exportJSON(message.payload, signal, message.id);

    case WorkerMessageType.EXPORT_EXCEL:
      return exportExcel(message.payload, signal, message.id);

    case WorkerMessageType.CALCULATE_AGGREGATIONS:
      return calculateAggregations(message.payload, signal, message.id);

    case WorkerMessageType.CALCULATE_STATISTICS:
      return calculateStatistics(message.payload, signal, message.id);

    case WorkerMessageType.VALIDATE_DATA:
      return validateData(message.payload, signal, message.id);

    case WorkerMessageType.TRANSFORM_DATA:
      return transformData(message.payload, signal, message.id);

    default:
      throw new Error(`Unknown message type: ${message.type}`);
  }
}

// Data Parsing
async function parseQueryResults(
  data: unknown,
  signal: AbortSignal,
  operationId: string
): Promise<QueryResult> {
  const rows = Array.isArray(data.rows) ? data.rows : [];
  const total = rows.length;
  let processed = 0;

  // Infer column definitions if not provided
  const columns = data.columns || inferColumns(rows);

  // Parse and validate each row
  const parsedRows = [];
  for (let i = 0; i < rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, i, total, 'Parsing rows...');
    }

    const parsedRow = parseRow(rows[i], columns);
    parsedRows.push(parsedRow);
    processed++;
  }

  return {
    columns,
    rows: parsedRows,
    metadata: {
      totalRows: processed,
      ...data.metadata
    }
  };
}

// Data Filtering
async function filterData(
  payload: { data: QueryResult; filters: FilterCondition[] },
  signal: AbortSignal,
  operationId: string
): Promise<QueryResult> {
  const { data, filters } = payload;
  const total = data.rows.length;

  const filteredRows = [];
  for (let i = 0; i < data.rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, i, total, 'Filtering data...');
    }

    const row = data.rows[i];
    if (matchesFilters(row, filters)) {
      filteredRows.push(row);
    }
  }

  return {
    ...data,
    rows: filteredRows,
    metadata: {
      ...data.metadata,
      totalRows: filteredRows.length,
    }
  };
}

// Data Sorting
async function sortData(
  payload: { data: QueryResult; sorts: SortCondition[] },
  signal: AbortSignal,
  operationId: string
): Promise<QueryResult> {
  const { data, sorts } = payload;

  sendProgress(operationId, 0, 1, 'Sorting data...');

  const sortedRows = [...data.rows].sort((a, b) => {
    for (const sort of sorts) {
      if (signal.aborted) throw new Error('Operation cancelled');

      const aVal = a[sort.column];
      const bVal = b[sort.column];

      // Handle nulls
      if (aVal === null || aVal === undefined) {
        return sort.nullsFirst ? -1 : 1;
      }
      if (bVal === null || bVal === undefined) {
        return sort.nullsFirst ? 1 : -1;
      }

      // Compare values
      let comparison = 0;
      if (typeof aVal === 'string' && typeof bVal === 'string') {
        comparison = sort.collation === 'numeric'
          ? parseFloat(aVal) - parseFloat(bVal)
          : aVal.localeCompare(bVal);
      } else if (typeof aVal === 'number' && typeof bVal === 'number') {
        comparison = aVal - bVal;
      } else {
        comparison = String(aVal).localeCompare(String(bVal));
      }

      if (comparison !== 0) {
        return sort.direction === SortDirection.ASC ? comparison : -comparison;
      }
    }
    return 0;
  });

  sendProgress(operationId, 1, 1, 'Sorting complete');

  return {
    ...data,
    rows: sortedRows
  };
}

// CSV Export
async function exportCSV(
  payload: { data: QueryResult; config: ExportConfig },
  signal: AbortSignal,
  operationId: string
): Promise<string> {
  const { data, config } = payload;
  const delimiter = config.delimiter || ',';
  const quote = config.quote || '"';
  const escape = config.escape || '"';
  const includeHeaders = config.includeHeaders !== false;

  const rows = [];
  const total = data.rows.length + (includeHeaders ? 1 : 0);
  let processed = 0;

  // Add headers
  if (includeHeaders) {
    const headers = data.columns.map(col => escapeCSVValue(col.name, quote, escape, delimiter));
    rows.push(headers.join(delimiter));
    processed++;
  }

  // Add data rows
  for (let i = 0; i < data.rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, processed, total, 'Exporting to CSV...');
    }

    const row = data.rows[i];
    const values = data.columns.map(col => {
      const value = formatValue(row[col.name], config);
      return escapeCSVValue(value, quote, escape, delimiter);
    });

    rows.push(values.join(delimiter));
    processed++;
  }

  return rows.join('\n');
}

// JSON Export
async function exportJSON(
  payload: { data: QueryResult; config: ExportConfig },
  signal: AbortSignal,
  operationId: string
): Promise<string> {
  const { data, config } = payload;
  const formatted = [];
  const total = data.rows.length;

  for (let i = 0; i < data.rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, i, total, 'Exporting to JSON...');
    }

    const row = data.rows[i];
    const formattedRow: unknown = {};

    for (const col of data.columns) {
      formattedRow[col.name] = formatValue(row[col.name], config);
    }

    formatted.push(formattedRow);
  }

  return JSON.stringify(formatted, null, 2);
}

// Excel Export (simplified - returns CSV for now)
async function exportExcel(
  payload: { data: QueryResult; config: ExportConfig },
  signal: AbortSignal,
  operationId: string
): Promise<string> {
  // In a real implementation, this would use a library like ExcelJS
  // For now, we'll return tab-separated values
  return exportCSV({
    data: payload.data,
    config: { ...payload.config, delimiter: '\t' }
  }, signal, operationId);
}

// Aggregations
async function calculateAggregations(
  payload: { data: QueryResult; config: AggregationConfig },
  signal: AbortSignal,
  operationId: string
): Promise<unknown> {
  const { data, config } = payload;
  const groups = new Map<string, unknown[]>();
  const total = data.rows.length;

  // Group data
  if (config.groupBy && config.groupBy.length > 0) {
    for (let i = 0; i < data.rows.length; i++) {
      if (signal.aborted) throw new Error('Operation cancelled');

      if (i % 1000 === 0) {
        sendProgress(operationId, i, total, 'Grouping data...');
      }

      const row = data.rows[i];
      const key = config.groupBy.map(col => row[col]).join('|||');

      if (!groups.has(key)) {
        groups.set(key, []);
      }
      groups.get(key)!.push(row);
    }
  } else {
    groups.set('all', data.rows);
  }

  // Calculate aggregations for each group
  const results = [];
  for (const [groupKey, groupRows] of groups.entries()) {
    if (signal.aborted) throw new Error('Operation cancelled');

    const result: unknown = {};

    // Add group keys
    if (config.groupBy) {
      const keyParts = groupKey.split('|||');
      config.groupBy.forEach((col, idx) => {
        result[col] = keyParts[idx];
      });
    }

    // Calculate aggregations
    for (const agg of config.aggregations) {
      const values = groupRows.map(row => row[agg.column]).filter(v => v != null);
      const alias = agg.alias || `${agg.operation.toLowerCase()}_${agg.column}`;

      switch (agg.operation) {
        case AggregationOperation.COUNT:
          result[alias] = groupRows.length;
          break;
        case AggregationOperation.COUNT_DISTINCT:
          result[alias] = new Set(values).size;
          break;
        case AggregationOperation.SUM:
          result[alias] = values.reduce((sum, val) => sum + Number(val), 0);
          break;
        case AggregationOperation.AVG:
          result[alias] = values.length > 0
            ? values.reduce((sum, val) => sum + Number(val), 0) / values.length
            : null;
          break;
        case AggregationOperation.MIN:
          result[alias] = values.length > 0 ? Math.min(...values.map(Number)) : null;
          break;
        case AggregationOperation.MAX:
          result[alias] = values.length > 0 ? Math.max(...values.map(Number)) : null;
          break;
        case AggregationOperation.MEDIAN:
          result[alias] = calculateMedian(values.map(Number));
          break;
        case AggregationOperation.STDDEV:
          result[alias] = calculateStdDev(values.map(Number));
          break;
        case AggregationOperation.VARIANCE:
          result[alias] = calculateVariance(values.map(Number));
          break;
      }
    }

    // Apply having filters
    if (config.having && !matchesFilters(result, config.having)) {
      continue;
    }

    results.push(result);
  }

  return results;
}

// Statistics
async function calculateStatistics(
  payload: { data: QueryResult; columns: string[] },
  signal: AbortSignal,
  _operationId: string // eslint-disable-line @typescript-eslint/no-unused-vars
): Promise<unknown> {
  const { data, columns } = payload;
  const stats: unknown = {};

  for (const column of columns) {
    if (signal.aborted) throw new Error('Operation cancelled');

    const values = data.rows.map(row => row[column]).filter(v => v != null);
    const numericValues = values.filter(v => !isNaN(Number(v))).map(Number);

    stats[column] = {
      count: values.length,
      unique: new Set(values).size,
      nulls: data.rows.length - values.length,
      type: inferType(values[0])
    };

    if (numericValues.length > 0) {
      stats[column].numeric = {
        min: Math.min(...numericValues),
        max: Math.max(...numericValues),
        mean: numericValues.reduce((sum, val) => sum + val, 0) / numericValues.length,
        median: calculateMedian(numericValues),
        stddev: calculateStdDev(numericValues),
        variance: calculateVariance(numericValues),
        sum: numericValues.reduce((sum, val) => sum + val, 0)
      };
    }

    if (values.length > 0 && typeof values[0] === 'string') {
      stats[column].string = {
        minLength: Math.min(...values.map(v => String(v).length)),
        maxLength: Math.max(...values.map(v => String(v).length)),
        avgLength: values.reduce((sum, v) => sum + String(v).length, 0) / values.length
      };
    }
  }

  return stats;
}

// Data Validation
async function validateData(
  payload: { data: QueryResult; rules: ValidationRule[] },
  signal: AbortSignal,
  operationId: string
): Promise<unknown> {
  const { data, rules } = payload;
  const errors = [];
  const total = data.rows.length;

  for (let i = 0; i < data.rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, i, total, 'Validating data...');
    }

    const row = data.rows[i];

    for (const rule of rules) {
      const value = row[rule.column];
      const error = validateValue(value, rule);

      if (error) {
        errors.push({
          row: i,
          column: rule.column,
          value,
          error: error,
          rule: rule.type
        });
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors,
    summary: {
      totalRows: data.rows.length,
      errorCount: errors.length,
      errorRate: errors.length / data.rows.length
    }
  };
}

// Data Transformation
async function transformData(
  payload: { data: QueryResult; transformations: TransformationRule[] },
  signal: AbortSignal,
  operationId: string
): Promise<QueryResult> {
  const { data, transformations } = payload;
  const transformedRows = [];
  const total = data.rows.length;

  // Determine new columns
  const newColumns = [...data.columns];
  for (const transform of transformations) {
    if (transform.targetColumn && !newColumns.find(c => c.name === transform.targetColumn)) {
      newColumns.push({
        name: transform.targetColumn,
        type: inferTransformationType(transform)
      });
    }
  }

  // Transform each row
  for (let i = 0; i < data.rows.length; i++) {
    if (signal.aborted) throw new Error('Operation cancelled');

    if (i % 1000 === 0) {
      sendProgress(operationId, i, total, 'Transforming data...');
    }

    const row = { ...data.rows[i] };

    for (const transform of transformations) {
      const value = row[transform.column];
      const transformed = applyTransformation(value, transform);
      const targetCol = transform.targetColumn || transform.column;
      row[targetCol] = transformed;
    }

    transformedRows.push(row);
  }

  return {
    columns: newColumns,
    rows: transformedRows,
    metadata: data.metadata
  };
}

// Helper Functions

function inferColumns(rows: unknown[]): unknown[] {
  if (rows.length === 0) return [];

  const firstRow = rows[0];
  return Object.keys(firstRow).map(key => ({
    name: key,
    type: inferType(firstRow[key])
  }));
}

function inferType(value: unknown): string {
  if (value === null || value === undefined) return 'UNKNOWN';
  if (typeof value === 'boolean') return 'BOOLEAN';
  if (typeof value === 'number') {
    return Number.isInteger(value) ? 'INTEGER' : 'FLOAT';
  }
  if (typeof value === 'string') {
    // Try to detect date/time
    if (/^\d{4}-\d{2}-\d{2}/.test(value)) return 'DATE';
    if (/^\d{2}:\d{2}/.test(value)) return 'TIME';
    return 'STRING';
  }
  if (typeof value === 'object') {
    if (value instanceof Date) return 'DATETIME';
    if (Array.isArray(value)) return 'ARRAY';
    return 'JSON';
  }
  return 'UNKNOWN';
}

function inferTransformationType(transform: TransformationRule): unknown {
  switch (transform.type) {
    case TransformationType.UPPERCASE:
    case TransformationType.LOWERCASE:
    case TransformationType.TRIM:
    case TransformationType.REPLACE:
    case TransformationType.SUBSTRING:
    case TransformationType.CONCAT:
    case TransformationType.DATE_FORMAT:
      return 'STRING';
    case TransformationType.SPLIT:
      return 'ARRAY';
    case TransformationType.NUMBER_FORMAT:
      return 'NUMBER';
    default:
      return 'STRING';
  }
}

function parseRow(row: unknown, columns: unknown[]): unknown {
  const parsed: unknown = {};

  for (const col of columns) {
    const value = row[col.name];
    parsed[col.name] = parseValue(value, col.type);
  }

  return parsed;
}

function parseValue(value: unknown, type: string): unknown {
  if (value === null || value === undefined) return null;

  switch (type) {
    case 'INTEGER':
      return parseInt(value, 10);
    case 'FLOAT':
    case 'NUMBER':
      return parseFloat(value);
    case 'BOOLEAN':
      return Boolean(value);
    case 'DATE':
    case 'DATETIME':
      return new Date(value);
    default:
      return value;
  }
}

function matchesFilters(row: unknown, filters: FilterCondition[]): boolean {
  for (const filter of filters) {
    const value = row[filter.column];

    if (!matchesFilter(value, filter)) {
      return false;
    }
  }
  return true;
}

function matchesFilter(value: unknown, filter: FilterCondition): boolean {
  switch (filter.operator) {
    case FilterOperator.EQUALS:
      return value === filter.value;
    case FilterOperator.NOT_EQUALS:
      return value !== filter.value;
    case FilterOperator.GREATER_THAN:
      return value > filter.value;
    case FilterOperator.GREATER_THAN_OR_EQUALS:
      return value >= filter.value;
    case FilterOperator.LESS_THAN:
      return value < filter.value;
    case FilterOperator.LESS_THAN_OR_EQUALS:
      return value <= filter.value;
    case FilterOperator.CONTAINS:
      return String(value).includes(String(filter.value));
    case FilterOperator.NOT_CONTAINS:
      return !String(value).includes(String(filter.value));
    case FilterOperator.STARTS_WITH:
      return String(value).startsWith(String(filter.value));
    case FilterOperator.ENDS_WITH:
      return String(value).endsWith(String(filter.value));
    case FilterOperator.IN:
      return Array.isArray(filter.value) && filter.value.includes(value);
    case FilterOperator.NOT_IN:
      return Array.isArray(filter.value) && !filter.value.includes(value);
    case FilterOperator.IS_NULL:
      return value === null || value === undefined;
    case FilterOperator.IS_NOT_NULL:
      return value !== null && value !== undefined;
    case FilterOperator.REGEX:
      return new RegExp(filter.value).test(String(value));
    case FilterOperator.BETWEEN:
      return Array.isArray(filter.value) &&
        value >= filter.value[0] &&
        value <= filter.value[1];
    default:
      return false;
  }
}

function formatValue(value: unknown, config: ExportConfig): string {
  if (value === null || value === undefined) {
    return config.nullValue || '';
  }

  if (typeof value === 'boolean') {
    switch (config.booleanFormat) {
      case '1/0':
        return value ? '1' : '0';
      case 'yes/no':
        return value ? 'yes' : 'no';
      default:
        return value ? 'true' : 'false';
    }
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (typeof value === 'object') {
    return JSON.stringify(value);
  }

  return String(value);
}

function escapeCSVValue(value: string, quote: string, escape: string, delimiter: string): string {
  if (value.includes(delimiter) || value.includes(quote) || value.includes('\n')) {
    return quote + value.replace(new RegExp(quote, 'g'), escape + quote) + quote;
  }
  return value;
}

function calculateMedian(values: number[]): number | null {
  if (values.length === 0) return null;

  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);

  if (sorted.length % 2 === 0) {
    return (sorted[mid - 1] + sorted[mid]) / 2;
  }

  return sorted[mid];
}

function calculateStdDev(values: number[]): number | null {
  const variance = calculateVariance(values);
  return variance !== null ? Math.sqrt(variance) : null;
}

function calculateVariance(values: number[]): number | null {
  if (values.length === 0) return null;

  const mean = values.reduce((sum, val) => sum + val, 0) / values.length;
  const squaredDiffs = values.map(val => Math.pow(val - mean, 2));
  return squaredDiffs.reduce((sum, val) => sum + val, 0) / values.length;
}

function validateValue(value: unknown, rule: ValidationRule): string | null {
  switch (rule.type) {
    case ValidationType.REQUIRED:
      if (value === null || value === undefined || value === '') {
        return rule.errorMessage || 'Value is required';
      }
      break;
    case ValidationType.MIN_LENGTH:
      if (String(value).length < rule.config) {
        return rule.errorMessage || `Minimum length is ${rule.config}`;
      }
      break;
    case ValidationType.MAX_LENGTH:
      if (String(value).length > rule.config) {
        return rule.errorMessage || `Maximum length is ${rule.config}`;
      }
      break;
    case ValidationType.MIN_VALUE:
      if (Number(value) < rule.config) {
        return rule.errorMessage || `Minimum value is ${rule.config}`;
      }
      break;
    case ValidationType.MAX_VALUE:
      if (Number(value) > rule.config) {
        return rule.errorMessage || `Maximum value is ${rule.config}`;
      }
      break;
    case ValidationType.PATTERN:
      if (!new RegExp(rule.config).test(String(value))) {
        return rule.errorMessage || 'Value does not match required pattern';
      }
      break;
    case ValidationType.EMAIL:
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(String(value))) {
        return rule.errorMessage || 'Invalid email address';
      }
      break;
    case ValidationType.URL:
      try {
        new URL(String(value));
      } catch {
        return rule.errorMessage || 'Invalid URL';
      }
      break;
    case ValidationType.UUID:
      if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(String(value))) {
        return rule.errorMessage || 'Invalid UUID';
      }
      break;
  }

  return null;
}

function applyTransformation(value: unknown, transform: TransformationRule): unknown {
  if (value === null || value === undefined) return null;

  switch (transform.type) {
    case TransformationType.UPPERCASE:
      return String(value).toUpperCase();
    case TransformationType.LOWERCASE:
      return String(value).toLowerCase();
    case TransformationType.TRIM:
      return String(value).trim();
    case TransformationType.REPLACE:
      return String(value).replace(
        new RegExp(transform.config.search, 'g'),
        transform.config.replace
      );
    case TransformationType.SUBSTRING:
      return String(value).substring(transform.config.start, transform.config.end);
    case TransformationType.CONCAT:
      return transform.config.prefix + String(value) + transform.config.suffix;
    case TransformationType.SPLIT:
      return String(value).split(transform.config.delimiter);
    case TransformationType.DATE_FORMAT:
      return new Date(value).toLocaleDateString(undefined, transform.config);
    case TransformationType.NUMBER_FORMAT:
      return Number(value).toFixed(transform.config.decimals || 2);
    case TransformationType.CAST:
      return parseValue(value, transform.config.targetType);
    default:
      return value;
  }
}

function sendProgress(operationId: string, current: number, total: number, message?: string) {
  const progress: ProgressUpdate = {
    id: operationId,
    current,
    total,
    percentage: (current / total) * 100,
    message
  };

  const response: WorkerResponse<ProgressUpdate> = {
    id: operationId,
    type: WorkerMessageType.PROGRESS_UPDATE,
    success: true,
    result: progress,
    timestamp: Date.now()
  };

  ctx.postMessage(response);
}

function handleCancellation(operationId: string) {
  const controller = abortControllers.get(operationId);
  if (controller) {
    controller.abort();
    abortControllers.delete(operationId);
  }
}

function getMemoryUsage(): MemoryUsage {
  // Use performance.memory if available (Chrome/Edge)
  if ('memory' in performance) {
    const memory = (performance as unknown as { memory: { usedJSHeapSize: number; totalJSHeapSize: number; jsHeapSizeLimit: number } }).memory;
    return {
      used: memory.usedJSHeapSize,
      peak: memory.totalJSHeapSize,
      limit: memory.jsHeapSizeLimit,
      percentage: (memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100
    };
  }

  // Fallback for other browsers
  return {
    used: 0,
    peak: 0
  };
}

// Export for testing
export {};
