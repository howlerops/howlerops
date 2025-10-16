/**
 * SharedArrayBuffer Support for Large Dataset Processing
 * Enables efficient memory sharing between workers and main thread
 */

import { QueryResult, ColumnDefinition, DataType } from './types';

export interface SharedBufferConfig {
  rowCount: number;
  columns: ColumnDefinition[];
  bufferSize?: number;
  enableCompression?: boolean;
}

export interface SharedBufferMetadata {
  bufferSize: number;
  rowCount: number;
  columnCount: number;
  columnOffsets: number[];
  columnSizes: number[];
  dataTypes: DataType[];
  currentRows: number;
  isLocked: boolean;
}

export class SharedBufferManager {
  private static isSupported = typeof SharedArrayBuffer !== 'undefined';
  private buffer?: SharedArrayBuffer;
  private metadata?: SharedBufferMetadata;
  private dataView?: DataView;
  private int32Array?: Int32Array;
  private float64Array?: Float64Array;
  private textDecoder = new TextDecoder();
  private textEncoder = new TextEncoder();

  constructor() {
    if (!SharedBufferManager.isSupported) {
      console.warn('SharedArrayBuffer is not supported in this environment');
    }
  }

  static checkSupport(): boolean {
    // Check if SharedArrayBuffer is available
    if (!SharedBufferManager.isSupported) {
      return false;
    }

    // Check for required headers (COOP and COEP)
    try {
      new SharedArrayBuffer(1);
      return true;
    } catch (e) {
      console.warn('SharedArrayBuffer requires proper COOP and COEP headers:', e);
      return false;
    }
  }

  createBuffer(config: SharedBufferConfig): SharedArrayBuffer | null {
    if (!SharedBufferManager.checkSupport()) {
      return null;
    }

    try {
      // Calculate required buffer size
      const bufferSize = this.calculateBufferSize(config);

      // Create SharedArrayBuffer
      this.buffer = new SharedArrayBuffer(bufferSize);

      // Create views
      this.dataView = new DataView(this.buffer);
      this.int32Array = new Int32Array(this.buffer);
      this.float64Array = new Float64Array(this.buffer);

      // Initialize metadata
      this.metadata = {
        bufferSize,
        rowCount: config.rowCount,
        columnCount: config.columns.length,
        columnOffsets: [],
        columnSizes: [],
        dataTypes: config.columns.map(c => c.type),
        currentRows: 0,
        isLocked: false
      };

      // Calculate column offsets
      this.calculateColumnOffsets(config.columns);

      // Write metadata to buffer
      this.writeMetadata();

      return this.buffer;

    } catch (error) {
      console.error('Failed to create SharedArrayBuffer:', error);
      return null;
    }
  }

  private calculateBufferSize(config: SharedBufferConfig): number {
    if (config.bufferSize) {
      return config.bufferSize;
    }

    let size = 0;
    const metadataSize = 1024; // Reserved for metadata

    for (const column of config.columns) {
      const columnSize = this.getColumnSize(column, config.rowCount);
      size += columnSize;
    }

    return metadataSize + size;
  }

  private getColumnSize(column: ColumnDefinition, rowCount: number): number {
    switch (column.type) {
      case DataType.INTEGER:
        return rowCount * 4; // 4 bytes per int32
      case DataType.FLOAT:
      case DataType.NUMBER:
        return rowCount * 8; // 8 bytes per float64
      case DataType.BOOLEAN:
        return Math.ceil(rowCount / 8); // 1 bit per boolean
      case DataType.STRING: {
        // Estimate string size
        const avgStringLength = column.maxLength || 50;
        return rowCount * avgStringLength * 2; // 2 bytes per character (UTF-16)
      }
      case DataType.DATE:
      case DataType.DATETIME:
        return rowCount * 8; // Store as timestamp (float64)
      default:
        return rowCount * 100; // Default estimate
    }
  }

  private calculateColumnOffsets(columns: ColumnDefinition[]): void {
    if (!this.metadata) return;

    let offset = 1024; // Start after metadata

    for (let i = 0; i < columns.length; i++) {
      this.metadata.columnOffsets.push(offset);
      const size = this.getColumnSize(columns[i], this.metadata.rowCount);
      this.metadata.columnSizes.push(size);
      offset += size;
    }
  }

  private writeMetadata(): void {
    if (!this.dataView || !this.metadata) return;

    // Write metadata header
    this.dataView.setInt32(0, this.metadata.bufferSize, true);
    this.dataView.setInt32(4, this.metadata.rowCount, true);
    this.dataView.setInt32(8, this.metadata.columnCount, true);
    this.dataView.setInt32(12, this.metadata.currentRows, true);

    // Write column offsets
    let offset = 16;
    for (let i = 0; i < this.metadata.columnCount; i++) {
      this.dataView.setInt32(offset, this.metadata.columnOffsets[i], true);
      offset += 4;
    }

    // Write column sizes
    for (let i = 0; i < this.metadata.columnCount; i++) {
      this.dataView.setInt32(offset, this.metadata.columnSizes[i], true);
      offset += 4;
    }

    // Write data types
    for (let i = 0; i < this.metadata.columnCount; i++) {
      this.dataView.setInt8(offset, this.metadata.dataTypes[i] as number);
      offset += 1;
    }
  }

  writeData(data: QueryResult): boolean {
    if (!this.buffer || !this.metadata || !this.dataView) {
      return false;
    }

    try {
      // Lock buffer for writing
      this.lock();

      for (let rowIndex = 0; rowIndex < data.rows.length; rowIndex++) {
        const row = data.rows[rowIndex];

        for (let colIndex = 0; colIndex < data.columns.length; colIndex++) {
          const column = data.columns[colIndex];
          const value = row[column.name];

          this.writeValue(value, column.type, rowIndex, colIndex);
        }
      }

      // Update current rows
      this.metadata.currentRows = data.rows.length;
      this.dataView.setInt32(12, this.metadata.currentRows, true);

      // Unlock buffer
      this.unlock();

      return true;

    } catch (error) {
      console.error('Failed to write data to SharedArrayBuffer:', error);
      this.unlock();
      return false;
    }
  }

  private writeValue(value: unknown, type: DataType, rowIndex: number, colIndex: number): void {
    if (!this.dataView || !this.metadata) return;

    const offset = this.metadata.columnOffsets[colIndex];
    const position = this.getValuePosition(offset, type, rowIndex);

    switch (type) {
      case DataType.INTEGER:
        this.dataView.setInt32(position, value || 0, true);
        break;

      case DataType.FLOAT:
      case DataType.NUMBER:
        this.dataView.setFloat64(position, value || 0, true);
        break;

      case DataType.BOOLEAN:
        this.writeBooleanBit(position, rowIndex, value);
        break;

      case DataType.STRING:
        this.writeString(position, value || '');
        break;

      case DataType.DATE:
      case DataType.DATETIME: {
        const timestamp = value instanceof Date ? value.getTime() : new Date(value).getTime();
        this.dataView.setFloat64(position, timestamp, true);
        break;
      }

      default:
        // Handle as JSON string
        this.writeString(position, JSON.stringify(value));
    }
  }

  private getValuePosition(baseOffset: number, type: DataType, rowIndex: number): number {
    switch (type) {
      case DataType.INTEGER:
        return baseOffset + (rowIndex * 4);
      case DataType.FLOAT:
      case DataType.NUMBER:
      case DataType.DATE:
      case DataType.DATETIME:
        return baseOffset + (rowIndex * 8);
      case DataType.BOOLEAN:
        return baseOffset + Math.floor(rowIndex / 8);
      case DataType.STRING:
      default:
        return baseOffset + (rowIndex * 100); // Assuming fixed string allocation
    }
  }

  private writeBooleanBit(position: number, bitIndex: number, value: boolean): void {
    if (!this.dataView) return;

    const _byteIndex = Math.floor(bitIndex / 8); // eslint-disable-line @typescript-eslint/no-unused-vars
    const bit = bitIndex % 8;
    let byte = this.dataView.getUint8(position);

    if (value) {
      byte |= (1 << bit);
    } else {
      byte &= ~(1 << bit);
    }

    this.dataView.setUint8(position, byte);
  }

  private writeString(position: number, value: string): void {
    if (!this.dataView) return;

    const encoded = this.textEncoder.encode(value);
    const maxLength = 100; // Fixed allocation per string

    // Write string length
    this.dataView.setInt32(position, encoded.length, true);

    // Write string bytes
    for (let i = 0; i < Math.min(encoded.length, maxLength - 4); i++) {
      this.dataView.setUint8(position + 4 + i, encoded[i]);
    }
  }

  readData(columns: ColumnDefinition[]): QueryResult | null {
    if (!this.buffer || !this.metadata || !this.dataView) {
      return null;
    }

    try {
      const rows = [];

      for (let rowIndex = 0; rowIndex < this.metadata.currentRows; rowIndex++) {
        const row: unknown = {};

        for (let colIndex = 0; colIndex < columns.length; colIndex++) {
          const column = columns[colIndex];
          const value = this.readValue(column.type, rowIndex, colIndex);
          row[column.name] = value;
        }

        rows.push(row);
      }

      return {
        columns,
        rows,
        metadata: {
          totalRows: this.metadata.currentRows
        }
      };

    } catch (error) {
      console.error('Failed to read data from SharedArrayBuffer:', error);
      return null;
    }
  }

  private readValue(type: DataType, rowIndex: number, colIndex: number): unknown {
    if (!this.dataView || !this.metadata) return null;

    const offset = this.metadata.columnOffsets[colIndex];
    const position = this.getValuePosition(offset, type, rowIndex);

    switch (type) {
      case DataType.INTEGER:
        return this.dataView.getInt32(position, true);

      case DataType.FLOAT:
      case DataType.NUMBER:
        return this.dataView.getFloat64(position, true);

      case DataType.BOOLEAN:
        return this.readBooleanBit(position, rowIndex);

      case DataType.STRING:
        return this.readString(position);

      case DataType.DATE:
      case DataType.DATETIME: {
        const timestamp = this.dataView.getFloat64(position, true);
        return new Date(timestamp);
      }

      default: {
        const jsonString = this.readString(position);
        try {
          return JSON.parse(jsonString);
        } catch {
          return jsonString;
        }
      }
    }
  }

  private readBooleanBit(position: number, bitIndex: number): boolean {
    if (!this.dataView) return false;

    const _byteIndex = Math.floor(bitIndex / 8); // eslint-disable-line @typescript-eslint/no-unused-vars
    const bit = bitIndex % 8;
    const byte = this.dataView.getUint8(position);

    return (byte & (1 << bit)) !== 0;
  }

  private readString(position: number): string {
    if (!this.dataView) return '';

    const length = this.dataView.getInt32(position, true);
    const bytes = new Uint8Array(length);

    for (let i = 0; i < length; i++) {
      bytes[i] = this.dataView.getUint8(position + 4 + i);
    }

    return this.textDecoder.decode(bytes);
  }

  // Atomic operations for thread safety
  private lock(): void {
    if (!this.int32Array || !this.metadata) return;

    // Use Atomics for thread-safe locking
    const lockIndex = 0; // First int32 position for lock

    while (Atomics.compareExchange(this.int32Array, lockIndex, 0, 1) !== 0) {
      // Wait until lock is available
      Atomics.wait(this.int32Array, lockIndex, 1);
    }

    this.metadata.isLocked = true;
  }

  private unlock(): void {
    if (!this.int32Array || !this.metadata) return;

    const lockIndex = 0;
    Atomics.store(this.int32Array, lockIndex, 0);
    Atomics.notify(this.int32Array, lockIndex, 1);

    this.metadata.isLocked = false;
  }

  // Utility methods

  getBuffer(): SharedArrayBuffer | undefined {
    return this.buffer;
  }

  getMetadata(): SharedBufferMetadata | undefined {
    return this.metadata;
  }

  clear(): void {
    if (!this.buffer || !this.metadata) return;

    this.lock();

    // Clear data area (skip metadata)
    const dataStart = 1024;
    const dataView = new DataView(this.buffer);

    for (let i = dataStart; i < this.metadata.bufferSize; i++) {
      dataView.setUint8(i, 0);
    }

    this.metadata.currentRows = 0;
    this.dataView?.setInt32(12, 0, true);

    this.unlock();
  }

  destroy(): void {
    this.buffer = undefined;
    this.metadata = undefined;
    this.dataView = undefined;
    this.int32Array = undefined;
    this.float64Array = undefined;
  }
}

// Helper function to transfer data using SharedArrayBuffer
export function createSharedDataTransfer(data: QueryResult): {
  buffer: SharedArrayBuffer | null;
  metadata: unknown;
} {
  const manager = new SharedBufferManager();

  const buffer = manager.createBuffer({
    rowCount: data.rows.length,
    columns: data.columns
  });

  if (buffer) {
    manager.writeData(data);
  }

  return {
    buffer,
    metadata: manager.getMetadata()
  };
}

// Helper function to read data from SharedArrayBuffer
export function readSharedDataTransfer(
  buffer: SharedArrayBuffer,
  columns: ColumnDefinition[]
): QueryResult | null {
  const manager = new SharedBufferManager();

  // Reconstruct manager with existing buffer
  (manager as unknown as { buffer: SharedArrayBuffer }).buffer = buffer;
  (manager as unknown as { dataView: DataView }).dataView = new DataView(buffer);
  (manager as unknown as { int32Array: Int32Array }).int32Array = new Int32Array(buffer);
  (manager as unknown as { float64Array: Float64Array }).float64Array = new Float64Array(buffer);

  // Read metadata from buffer
  const dataView = new DataView(buffer);
  (manager as unknown as { metadata: unknown }).metadata = {
    bufferSize: dataView.getInt32(0, true),
    rowCount: dataView.getInt32(4, true),
    columnCount: dataView.getInt32(8, true),
    currentRows: dataView.getInt32(12, true),
    columnOffsets: [],
    columnSizes: [],
    dataTypes: [],
    isLocked: false
  };

  // Read column offsets
  let offset = 16;
  const metadata = (manager as unknown as { metadata: { columnCount: number; columnOffsets: number[] } }).metadata;
  for (let i = 0; i < metadata.columnCount; i++) {
    metadata.columnOffsets.push(dataView.getInt32(offset, true));
    offset += 4;
  }

  return manager.readData(columns);
}