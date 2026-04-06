import { api } from '@/lib/api-client'
import { generateSQL, QueryIR } from '@/lib/query-ir'

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
  provenance?: string
}

export class MultiConnectionQueryEngine {
  private connections: Array<{
    id: string
    name: string
    type: string
    isConnected: boolean
  }>

  constructor(connections: Array<{ id: string; name: string; type: string; isConnected: boolean }>) {
    this.connections = connections
  }

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

    const targetConnections = this.connections.filter(
      conn => connectionIds.includes(conn.id) && conn.isConnected
    )

    if (targetConnections.length === 0) {
      throw new Error('No connected connections available')
    }

    const sql = generateSQL(queryIR, dialect)
    const startTime = Date.now()
    const promises = targetConnections.map((connection) =>
      this.executeOnConnection(connection.id, connection.name, sql, timeout)
    )

    const results = await Promise.allSettled(promises)
    const connectionResults: MultiConnectionResult[] = results.map((result, index) => {
      const connection = targetConnections[index]

      if (result.status === 'fulfilled') {
        return result.value
      }

      return {
        connectionId: connection.id,
        connectionName: connection.name,
        success: false,
        error: result.reason?.message || 'Unknown error',
      }
    })

    return this.mergeResults(connectionResults, addProvenance, Date.now() - startTime)
  }

  private async executeOnConnection(
    connectionId: string,
    connectionName: string,
    sql: string,
    timeout: number
  ): Promise<MultiConnectionResult> {
    try {
      const startTime = Date.now()
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => reject(new Error('Query timeout')), timeout)
      })

      const queryPromise = api.queries.execute(connectionId, sql)
      const response = await Promise.race([queryPromise, timeoutPromise])

      if (!response.success || !response.data) {
        return {
          connectionId,
          connectionName,
          success: false,
          error: response.message || 'Query execution failed',
        }
      }

      const { columns: rawColumns = [], rows = [], rowCount = 0 } = response.data
      const columns = rawColumns.map((column) =>
        typeof column === 'string' ? column : column.name
      )

      return {
        connectionId,
        connectionName,
        success: true,
        data: {
          columns,
          rows,
          rowCount,
          executionTime: Date.now() - startTime,
        },
      }
    } catch (error) {
      return {
        connectionId,
        connectionName,
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      }
    }
  }

  private mergeResults(
    results: MultiConnectionResult[],
    addProvenance: boolean,
    totalExecutionTime: number
  ): MergedResult {
    const successfulResults = results.filter(result => result.success && result.data)

    if (successfulResults.length === 0) {
      return {
        columns: [],
        rows: [],
        rowCount: 0,
        totalExecutionTime,
        connectionResults: results,
      }
    }

    const allColumns = successfulResults.flatMap(result => result.data!.columns)
    const uniqueColumns = Array.from(new Set(allColumns))
    const finalColumns = addProvenance ? [...uniqueColumns, '__connection'] : uniqueColumns

    const mergedRows: unknown[][] = []
    let totalRowCount = 0

    for (const result of successfulResults) {
      if (!result.data) continue

      const { columns, rows } = result.data

      for (const row of rows) {
        const mergedRow: unknown[] = []

        for (const column of finalColumns) {
          if (column === '__connection') {
            mergedRow.push(result.connectionName)
            continue
          }

          const columnIndex = columns.indexOf(column)
          mergedRow.push(columnIndex >= 0 ? row[columnIndex] : null)
        }

        mergedRows.push(mergedRow)
      }

      totalRowCount += result.data.rowCount
    }

    return {
      columns: finalColumns,
      rows: mergedRows,
      rowCount: totalRowCount,
      totalExecutionTime,
      connectionResults: results,
      provenance: addProvenance ? '__connection' : undefined,
    }
  }

  getConnectionInfo(connectionId: string) {
    return this.connections.find(conn => conn.id === connectionId)
  }

  areConnectionsAvailable(connectionIds: string[]): boolean {
    return connectionIds.every((id) =>
      this.connections.some(conn => conn.id === id && conn.isConnected)
    )
  }

  getAvailableConnectionIds(): string[] {
    return this.connections
      .filter(conn => conn.isConnected)
      .map(conn => conn.id)
  }
}

export function createMultiConnectionQueryEngine(
  connections: Array<{ id: string; name: string; type: string; isConnected: boolean }>
): MultiConnectionQueryEngine {
  return new MultiConnectionQueryEngine(connections)
}
