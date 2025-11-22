/**
 * Multi-Connection Query Executor
 * Handles parallel execution across multiple database connections
 */

import { generateSQL,QueryIR } from './query-ir'
import { wailsEndpoints } from './wails-api'

export interface MultiConnectionResult {
  connectionId: string
  connectionName: string
  success: boolean
  data?: {
    columns: string[]
    rows: unknown[][]
    rowCount: number
    executionTime: number
  }
  error?: string
}

export interface MergedResult {
  columns: string[]
  rows: unknown[][]
  rowCount: number
  totalExecutionTime: number
  connectionResults: MultiConnectionResult[]
  provenance?: string // Column name for connection tracking
}

export class MultiConnectionExecutor {
  private connections: Array<{
    id: string
    name: string
    type: string
    isConnected: boolean
  }>

  constructor(connections: Array<{ id: string; name: string; type: string; isConnected: boolean }>) {
    this.connections = connections
  }

  /**
   * Execute query across multiple connections in parallel
   */
  async executeQuery(
    queryIR: QueryIR,
    connectionIds: string[],
    options: {
      dialect?: 'postgres' | 'mysql' | 'sqlite' | 'mssql'
      addProvenance?: boolean
      timeout?: number
    } = {}
  ): Promise<MergedResult> {
    const { dialect = 'postgres', addProvenance = true, timeout = 30000 } = options
    
    // Filter to only connected connections
    const targetConnections = this.connections.filter(
      conn => connectionIds.includes(conn.id) && conn.isConnected
    )

    if (targetConnections.length === 0) {
      throw new Error('No connected connections available')
    }

    // Generate SQL for each connection
    const sql = generateSQL(queryIR, dialect)
    
    // Execute queries in parallel
    const startTime = Date.now()
    const promises = targetConnections.map(connection => 
      this.executeOnConnection(connection.id, connection.name, sql, timeout)
    )

    const results = await Promise.allSettled(promises)
    const connectionResults: MultiConnectionResult[] = results.map((result, index) => {
      const connection = targetConnections[index]
      
      if (result.status === 'fulfilled') {
        return result.value
      } else {
        return {
          connectionId: connection.id,
          connectionName: connection.name,
          success: false,
          error: result.reason?.message || 'Unknown error'
        }
      }
    })

    // Merge results
    return this.mergeResults(connectionResults, addProvenance, Date.now() - startTime)
  }

  /**
   * Execute query on a single connection
   */
  private async executeOnConnection(
    connectionId: string,
    connectionName: string,
    sql: string,
    timeout: number
  ): Promise<MultiConnectionResult> {
    try {
      const startTime = Date.now()
      
      // Create timeout promise
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('Query timeout')), timeout)
      })

      // Execute query with timeout
      const queryPromise = wailsEndpoints.queries.execute(connectionId, sql)
      const response = await Promise.race([queryPromise, timeoutPromise])

      if (!response.success || !response.data) {
        return {
          connectionId,
          connectionName,
          success: false,
          error: response.message || 'Query execution failed'
        }
      }

      const { columns = [], rows = [], rowCount = 0 } = response.data

      return {
        connectionId,
        connectionName,
        success: true,
        data: {
          columns,
          rows,
          rowCount,
          executionTime: Date.now() - startTime
        }
      }
    } catch (error) {
      return {
        connectionId,
        connectionName,
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error'
      }
    }
  }

  /**
   * Merge results from multiple connections
   */
  private mergeResults(
    results: MultiConnectionResult[],
    addProvenance: boolean,
    totalExecutionTime: number
  ): MergedResult {
    const successfulResults = results.filter(r => r.success && r.data)
    
    if (successfulResults.length === 0) {
      return {
        columns: [],
        rows: [],
        rowCount: 0,
        totalExecutionTime,
        connectionResults: results
      }
    }

    // Get common columns across all results
    const allColumns = successfulResults.flatMap(r => r.data!.columns)
    const uniqueColumns = Array.from(new Set(allColumns))
    
    // Add provenance column if requested
    const finalColumns = addProvenance 
      ? [...uniqueColumns, '__connection']
      : uniqueColumns

    // Merge rows
    const mergedRows: unknown[][] = []
    let totalRowCount = 0

    for (const result of successfulResults) {
      if (result.data) {
        const { columns, rows } = result.data
        
        for (const row of rows) {
          // Create row with all columns, filling missing ones with null
          const mergedRow: unknown[] = []
          
          for (const col of finalColumns) {
            if (col === '__connection') {
              mergedRow.push(result.connectionName)
            } else {
              const colIndex = columns.indexOf(col)
              mergedRow.push(colIndex >= 0 ? row[colIndex] : null)
            }
          }
          
          mergedRows.push(mergedRow)
        }
        
        totalRowCount += result.data.rowCount
      }
    }

    return {
      columns: finalColumns,
      rows: mergedRows,
      rowCount: totalRowCount,
      totalExecutionTime,
      connectionResults: results,
      provenance: addProvenance ? '__connection' : undefined
    }
  }

  /**
   * Get connection info by ID
   */
  getConnectionInfo(connectionId: string) {
    return this.connections.find(conn => conn.id === connectionId)
  }

  /**
   * Check if all specified connections are available
   */
  areConnectionsAvailable(connectionIds: string[]): boolean {
    return connectionIds.every(id => 
      this.connections.some(conn => conn.id === id && conn.isConnected)
    )
  }

  /**
   * Get available connection IDs
   */
  getAvailableConnectionIds(): string[] {
    return this.connections
      .filter(conn => conn.isConnected)
      .map(conn => conn.id)
  }
}

/**
 * Create a multi-connection executor
 */
export function createMultiConnectionExecutor(
  connections: Array<{ id: string; name: string; type: string; isConnected: boolean }>
): MultiConnectionExecutor {
  return new MultiConnectionExecutor(connections)
}
