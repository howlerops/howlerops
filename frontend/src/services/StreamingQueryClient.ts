import { EventEmitter } from 'events';
import { grpcWebClient } from '../lib/grpc-web-client';
import {
  StreamingQueryResponse,
  StreamResponseType,
  QueryOptions,
  DataFormat,
  // QueryProgress as GrpcQueryProgress,
  ColumnMetadata
  // QueryRow
} from '../generated/query';

export interface StreamingOptions {
  chunkSize?: number;
  maxRetries?: number;
  retryDelay?: number;
  timeout?: number;
  onProgress?: (progress: QueryProgress) => void;
  onChunk?: (chunk: unknown[]) => void;
  onMetadata?: (columns: ColumnMetadata[]) => void;
  signal?: AbortSignal;
}

export interface QueryProgress {
  rowsProcessed: number;
  bytesProcessed: number;
  elapsedTime: number;
  estimatedTotalRows?: number;
  percentComplete?: number;
  memoryUsage?: number;
  throughput?: number; // rows per second
  currentPhase?: string;
}

export interface QueryResult {
  rows: unknown[];
  metadata: {
    rowCount: number;
    executionTime: number;
    bytesTransferred: number;
    cached: boolean;
    columns?: ColumnMetadata[];
  };
}

export class StreamingQueryClient extends EventEmitter {
  private activeStreams: Map<string, {
    streamId: string;
    controller: AbortController;
    columns?: ColumnMetadata[];
    rows: unknown[][];
    startTime: number;
    progress: QueryProgress;
  }> = new Map();

  constructor() {
    super();
    this.setupMemoryMonitoring();
  }

  private handleStreamingMessage(
    queryId: string,
    message: StreamingQueryResponse
  ): void {
    const stream = this.activeStreams.get(queryId);
    if (!stream) return;

    switch (message.type) {
      case StreamResponseType.STREAM_RESPONSE_TYPE_METADATA:
        if (message.metadata?.columns) {
          stream.columns = message.metadata.columns;
          this.emit('metadata', queryId, message.metadata.columns);
        }
        break;

      case StreamResponseType.STREAM_RESPONSE_TYPE_DATA:
        if (message.data) {
          const rowValues = message.data.values?.map(val => {
            // Convert protobuf Any types to JavaScript values
            return (val as unknown as { value?: unknown })?.value || val;
          }) || [];

          stream.rows.push(rowValues);
          stream.progress.rowsProcessed++;

          // Update throughput
          const elapsedTime = Date.now() - stream.startTime;
          stream.progress.elapsedTime = elapsedTime;
          if (elapsedTime > 0) {
            stream.progress.throughput = (stream.progress.rowsProcessed / elapsedTime) * 1000;
          }

          this.emit('row', queryId, rowValues);

          // Emit chunk when buffer reaches chunk size
          const chunkSize = 1000; // Default chunk size
          if (stream.rows.length % chunkSize === 0) {
            const chunk = stream.rows.slice(-chunkSize);
            this.emit('chunk', queryId, chunk);
          }
        }
        break;

      case StreamResponseType.STREAM_RESPONSE_TYPE_PROGRESS:
        if (message.progress) {
          Object.assign(stream.progress, {
            rowsProcessed: Number(message.progress.rowsProcessed || 0),
            estimatedTotalRows: message.progress.totalRows ? Number(message.progress.totalRows) : undefined,
            percentComplete: message.progress.progressPercentage || 0,
            currentPhase: message.progress.currentPhase || 'Processing...',
          });

          this.emit('progress', queryId, stream.progress);
        }
        break;

      case StreamResponseType.STREAM_RESPONSE_TYPE_COMPLETE:
        this.emit('complete', queryId, stream.rows);
        this.activeStreams.delete(queryId);
        break;

      case StreamResponseType.STREAM_RESPONSE_TYPE_ERROR: {
        const error = new Error(message.error || 'Unknown error');
        this.emit('error', queryId, error);
        this.activeStreams.delete(queryId);
        break;
      }
    }
  }

  /**
   * Execute streaming query using gRPC-Web
   */
  async executeStreamingQuery(
    connectionId: string,
    query: string,
    options: StreamingOptions = {}
  ): Promise<QueryResult> {
    const queryId = this.generateQueryId();
    const controller = new AbortController();

    const startTime = Date.now();
    const progress: QueryProgress = {
      rowsProcessed: 0,
      bytesProcessed: 0,
      elapsedTime: 0,
      throughput: 0
    };

    const streamData = {
      streamId: '',
      controller,
      rows: [] as unknown[][],
      startTime,
      progress,
    };

    this.activeStreams.set(queryId, streamData);

    return new Promise((resolve, reject) => {
      const executeAsync = async () => {
        try {
          const queryOptions: QueryOptions = {
            limit: 0, // No limit for streaming
            timeoutSeconds: options.timeout || 300,
            readOnly: false,
            explain: false,
            format: DataFormat.DATA_FORMAT_JSON,
            includeMetadata: true,
            fetchSize: options.chunkSize || 10000,
          };

          const streamId = await grpcWebClient.executeStreamingQuery(
          {
            connectionId,
            sql: query,
            parameters: {},
            options: queryOptions,
            queryId,
          },
          (message: StreamingQueryResponse) => {
            this.handleStreamingMessage(queryId, message);
          },
          (error: Error) => {
            this.activeStreams.delete(queryId);
            this.emit('error', queryId, error);
            reject(error);
          },
          () => {
            // Stream completed
            const stream = this.activeStreams.get(queryId);
            if (stream) {
              const result: QueryResult = {
                rows: stream.rows,
                metadata: {
                  rowCount: stream.rows.length,
                  executionTime: stream.progress.elapsedTime,
                  bytesTransferred: stream.progress.bytesProcessed,
                  cached: false,
                  columns: stream.columns,
                }
              };

              this.activeStreams.delete(queryId);
              resolve(result);
            }
          }
        );

        streamData.streamId = streamId;

        // Setup event handlers for options callbacks
        this.once(`metadata-${queryId}`, (columns: ColumnMetadata[]) => {
          if (options.onMetadata) {
            options.onMetadata(columns);
          }
        });

        this.on(`progress-${queryId}`, (progress: QueryProgress) => {
          if (options.onProgress) {
            options.onProgress(progress);
          }
        });

        this.on(`chunk-${queryId}`, (chunk: unknown[]) => {
          if (options.onChunk) {
            options.onChunk(chunk);
          }
        });

        } catch (error) {
          this.activeStreams.delete(queryId);
          reject(error);
        }
      };

      executeAsync();
    });
  }

  /**
   * Execute regular (non-streaming) query
   */
  async executeQuery(
    query: string,
    params: unknown[] = [],
    options: { timeout?: number; signal?: AbortSignal } = {}
  ): Promise<QueryResult> {
    const response = await fetch(`${this.apiUrl}/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ query, params }),
      signal: options.signal
    });

    if (!response.ok) {
      throw new Error(`Query failed: ${response.statusText}`);
    }

    const result = await response.json();

    return {
      rows: result.rows,
      metadata: {
        rowCount: result.rows.length,
        executionTime: result.executionTime || 0,
        bytesTransferred: 0,
        cached: result.cached || false
      }
    };
  }

  /**
   * Cancel an active streaming query
   */
  cancelQuery(queryId: string): boolean {
    const controller = this.activeStreams.get(queryId);
    if (controller) {
      controller.abort();
      this.activeStreams.delete(queryId);
      this.emit('cancelled', queryId);
      return true;
    }
    return false;
  }

  /**
   * Fetch with automatic retry
   */
  private async fetchWithRetry(
    url: string,
    options: RequestInit,
    maxRetries: number,
    retryDelay: number
  ): Promise<Response> {
    let lastError: Error | null = null;

    for (let i = 0; i <= maxRetries; i++) {
      try {
        const response = await fetch(url, options);

        // Don't retry on client errors (4xx)
        if (response.status >= 400 && response.status < 500) {
          return response;
        }

        // Retry on server errors (5xx) or network errors
        if (response.ok) {
          return response;
        }

        lastError = new Error(`HTTP ${response.status}: ${response.statusText}`);

      } catch (error) {
        lastError = error as Error;

        // Don't retry on abort
        if (error.name === 'AbortError') {
          throw error;
        }
      }

      // Wait before retrying
      if (i < maxRetries) {
        await this.delay(retryDelay * Math.pow(2, i)); // Exponential backoff
      }
    }

    throw lastError || new Error('Failed to fetch after retries');
  }

  /**
   * Parse NDJSON stream using web streams API
   */
  private async* parseNDJSONStream(
    reader: ReadableStreamDefaultReader<Uint8Array>
  ): AsyncIterableIterator<unknown> {
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();

      if (done) {
        // Parse any remaining data
        if (buffer.trim()) {
          try {
            yield JSON.parse(buffer);
          } catch (_error) { // eslint-disable-line @typescript-eslint/no-unused-vars
            console.error('Failed to parse final buffer:', buffer);
          }
        }
        break;
      }

      buffer += decoder.decode(value, { stream: true });

      // Parse complete lines
      const lines = buffer.split('\n');
      buffer = lines.pop() || ''; // Keep incomplete line

      for (const line of lines) {
        if (line.trim()) {
          try {
            yield JSON.parse(line);
          } catch (_error) { // eslint-disable-line @typescript-eslint/no-unused-vars
            console.error('Failed to parse line:', line);
          }
        }
      }
    }
  }

  /**
   * Check if backpressure should be applied
   */
  private shouldApplyBackpressure(): boolean {
    // Check memory usage
    if ('memory' in performance) {
      const memoryInfo = (performance as unknown as { memory: { usedJSHeapSize: number; jsHeapSizeLimit: number } }).memory;
      const usedMemory = memoryInfo.usedJSHeapSize;
      const totalMemory = memoryInfo.jsHeapSizeLimit;

      // Apply backpressure if using more than 70% of available memory
      return usedMemory / totalMemory > 0.7;
    }

    // Check buffer sizes
    let totalBufferSize = 0;
    this.chunkBuffers.forEach(buffer => {
      totalBufferSize += buffer.length;
    });

    // Apply backpressure if total buffer size exceeds 10000 rows
    return totalBufferSize > 10000;
  }

  /**
   * Setup memory monitoring
   */
  private setupMemoryMonitoring(): void {
    if ('memory' in performance) {
      setInterval(() => {
        const memoryInfo = (performance as unknown as { memory: { usedJSHeapSize: number; jsHeapSizeLimit: number } }).memory;
        const usedMemory = memoryInfo.usedJSHeapSize;
        const totalMemory = memoryInfo.jsHeapSizeLimit;
        const usage = (usedMemory / totalMemory) * 100;

        if (usage > 80) {
          console.warn(`High memory usage: ${usage.toFixed(2)}%`);
          this.emit('memoryWarning', {
            used: usedMemory,
            total: totalMemory,
            percentage: usage
          });
        }
      }, 5000); // Check every 5 seconds
    }
  }

  /**
   * Generate unique query ID
   */
  private generateQueryId(): string {
    return `query_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Delay helper
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Get active query IDs
   */
  getActiveQueries(): string[] {
    return Array.from(this.activeStreams.keys());
  }

  /**
   * Get query metrics
   */
  getQueryMetrics(queryId: string): QueryProgress | undefined {
    return this.metrics.get(queryId);
  }

  /**
   * Clear all active streams
   */
  clearAllStreams(): void {
    this.activeStreams.forEach((controller, queryId) => {
      controller.abort();
      this.emit('cancelled', queryId);
    });

    this.activeStreams.clear();
    this.chunkBuffers.clear();
    this.metrics.clear();
  }
}

// Virtual table component for efficient rendering
export class VirtualTableRenderer {
  private container: HTMLElement;
  private rowHeight: number;
  private visibleRows: number;
  private totalRows: number;
  private data: unknown[];
  private renderCallback: (row: unknown, index: number) => HTMLElement;
  private scrollTop: number = 0;
  private renderedRange: { start: number; end: number } = { start: 0, end: 0 };

  constructor(
    container: HTMLElement,
    options: {
      rowHeight: number;
      renderRow: (row: unknown, index: number) => HTMLElement;
    }
  ) {
    this.container = container;
    this.rowHeight = options.rowHeight;
    this.renderCallback = options.renderRow;
    this.data = [];
    this.totalRows = 0;

    // Calculate visible rows
    this.visibleRows = Math.ceil(container.clientHeight / this.rowHeight);

    // Setup scroll listener
    this.setupScrollListener();
  }

  /**
   * Update data and render
   */
  setData(data: unknown[]): void {
    this.data = data;
    this.totalRows = data.length;
    this.render();
  }

  /**
   * Append new data (for streaming)
   */
  appendData(newData: unknown[]): void {
    this.data.push(...newData);
    this.totalRows = this.data.length;

    // Update virtual height
    this.updateVirtualHeight();

    // Re-render if new data is in visible range
    const visibleStart = Math.floor(this.scrollTop / this.rowHeight);
    const visibleEnd = visibleStart + this.visibleRows;

    if (this.totalRows <= visibleEnd + 10) {
      this.render();
    }
  }

  /**
   * Render visible rows
   */
  private render(): void {
    const scrollTop = this.container.scrollTop;
    const startIndex = Math.max(0, Math.floor(scrollTop / this.rowHeight) - 5);
    const endIndex = Math.min(
      this.totalRows,
      startIndex + this.visibleRows + 10
    );

    // Skip if range hasn't changed significantly
    if (
      Math.abs(startIndex - this.renderedRange.start) < 3 &&
      Math.abs(endIndex - this.renderedRange.end) < 3
    ) {
      return;
    }

    // Clear container
    this.container.innerHTML = '';

    // Create virtual spacer for rows above
    const spacerTop = document.createElement('div');
    spacerTop.style.height = `${startIndex * this.rowHeight}px`;
    this.container.appendChild(spacerTop);

    // Render visible rows
    for (let i = startIndex; i < endIndex; i++) {
      const row = this.data[i];
      if (row) {
        const element = this.renderCallback(row, i);
        this.container.appendChild(element);
      }
    }

    // Create virtual spacer for rows below
    const spacerBottom = document.createElement('div');
    const remainingRows = Math.max(0, this.totalRows - endIndex);
    spacerBottom.style.height = `${remainingRows * this.rowHeight}px`;
    this.container.appendChild(spacerBottom);

    this.renderedRange = { start: startIndex, end: endIndex };
  }

  /**
   * Update virtual height
   */
  private updateVirtualHeight(): void {
    const totalHeight = this.totalRows * this.rowHeight;
    this.container.style.height = `${totalHeight}px`;
  }

  /**
   * Setup scroll listener with debouncing
   */
  private setupScrollListener(): void {
    let scrollTimeout: NodeJS.Timeout;

    this.container.addEventListener('scroll', () => {
      this.scrollTop = this.container.scrollTop;

      // Debounce rendering
      clearTimeout(scrollTimeout);
      scrollTimeout = setTimeout(() => {
        this.render();
      }, 16); // ~60fps
    });
  }

  /**
   * Scroll to row
   */
  scrollToRow(index: number): void {
    const scrollTop = index * this.rowHeight;
    this.container.scrollTop = scrollTop;
  }

  /**
   * Get visible row indices
   */
  getVisibleRange(): { start: number; end: number } {
    const start = Math.floor(this.scrollTop / this.rowHeight);
    const end = start + this.visibleRows;
    return { start, end };
  }
}

// Create singleton instance
export const streamingQueryClient = new StreamingQueryClient();

export default StreamingQueryClient;