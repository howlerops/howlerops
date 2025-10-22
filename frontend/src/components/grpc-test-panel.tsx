import React, { useState } from 'react'
import { Button } from './ui/button'
import { Card } from './ui/card'
import { Input } from './ui/input'
import { useGrpcConnections, useCreateGrpcConnection, useTestGrpcConnection } from '../hooks/use-grpc-connections'
import { useGrpcStreamingQuery } from '../hooks/use-grpc-streaming-query'

export function GrpcTestPanel() {
  const [connectionData, setConnectionData] = useState({
    name: 'Test Connection',
    type: 'postgresql',
    host: 'localhost',
    port: 5432,
    database: 'testdb',
    username: 'testuser',
    password: 'testpass',
  })

  const [testQuery, setTestQuery] = useState('SELECT * FROM users LIMIT 1000')

  // gRPC hooks
  const { data: connections, isLoading: connectionsLoading, error: connectionsError } = useGrpcConnections()
  const createConnection = useCreateGrpcConnection()
  const testConnection = useTestGrpcConnection()
  const streamingQuery = useGrpcStreamingQuery()

  const handleTestConnection = async () => {
    try {
      const result = await testConnection.mutateAsync(connectionData)
      console.log('Connection test result:', result)
    } catch (error) {
      console.error('Connection test failed:', error)
    }
  }

  const handleCreateConnection = async () => {
    try {
      const result = await createConnection.mutateAsync(connectionData)
      console.log('Connection created:', result)
    } catch (error) {
      console.error('Failed to create connection:', error)
    }
  }

  const handleExecuteStreamingQuery = async () => {
    if (!connections?.data?.length) {
      alert('No connections available. Create a connection first.')
      return
    }

    const connectionId = connections.data[0].id
    try {
      await streamingQuery.execute(connectionId, testQuery, {
        onProgress: (progress: unknown) => {
          console.log('Query progress:', progress)
        },
        onRow: (row: unknown) => {
          console.log('New row received:', row)
        },
        onMetadata: (columns: unknown) => {
          console.log('Query metadata:', columns)
        },
      })
    } catch (error) {
      console.error('Streaming query failed:', error)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">gRPC-Web Test Panel</h2>

      {/* Connection Test Section */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Connection Management</h3>

        <div className="grid grid-cols-2 gap-4 mb-4">
          <div>
            <label className="block text-sm font-medium mb-1">Host</label>
            <Input
              value={connectionData.host}
              onChange={(e) => setConnectionData({ ...connectionData, host: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Port</label>
            <Input
              type="number"
              value={connectionData.port}
              onChange={(e) => setConnectionData({ ...connectionData, port: parseInt(e.target.value) })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Database</label>
            <Input
              value={connectionData.database}
              onChange={(e) => setConnectionData({ ...connectionData, database: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Username</label>
            <Input
              value={connectionData.username}
              onChange={(e) => setConnectionData({ ...connectionData, username: e.target.value })}
            />
          </div>
        </div>

        <div className="flex gap-2 mb-4">
          <Button
            onClick={handleTestConnection}
            disabled={testConnection.isPending}
          >
            {testConnection.isPending ? 'Testing...' : 'Test Connection'}
          </Button>

          <Button
            onClick={handleCreateConnection}
            disabled={createConnection.isPending}
          >
            {createConnection.isPending ? 'Creating...' : 'Create Connection'}
          </Button>
        </div>

        {testConnection.data && (
          <div className="mt-4 p-3 bg-primary/10 border border-primary rounded">
            <h4 className="font-medium text-primary">Test Result:</h4>
            <pre className="text-sm text-primary mt-1">
              {JSON.stringify(testConnection.data, null, 2)}
            </pre>
          </div>
        )}

        {(testConnection.error || createConnection.error) && (
          <div className="mt-4 p-3 bg-destructive/10 border border-destructive rounded">
            <h4 className="font-medium text-destructive">Error:</h4>
            <p className="text-sm text-destructive mt-1">
              {(testConnection.error || createConnection.error)?.message}
            </p>
          </div>
        )}
      </Card>

      {/* Connections List */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Active Connections</h3>

        {connectionsLoading && <p>Loading connections...</p>}
        {connectionsError && <p className="text-destructive">Error: {connectionsError.message}</p>}

        {connections?.data && (
          <div className="space-y-2">
            {connections.data.map((conn) => (
              <div key={conn.id} className="p-3 border rounded">
                <div className="font-medium">{conn.name}</div>
                <div className="text-sm text-gray-600">
                  {conn.type} - {conn.host}:{conn.port}/{conn.database}
                </div>
                <div className="text-xs text-gray-500">
                  Status: {conn.active ? 'Active' : 'Inactive'} | ID: {conn.id}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Streaming Query Test */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Streaming Query Test</h3>

        <div className="mb-4">
          <label className="block text-sm font-medium mb-1">SQL Query</label>
          <textarea
            className="w-full p-2 border rounded"
            rows={3}
            value={testQuery}
            onChange={(e) => setTestQuery(e.target.value)}
          />
        </div>

        <div className="flex gap-2 mb-4">
          <Button
            onClick={handleExecuteStreamingQuery}
            disabled={streamingQuery.isLoading || streamingQuery.isStreaming}
          >
            {streamingQuery.isLoading ? 'Starting...' :
             streamingQuery.isStreaming ? 'Streaming...' : 'Execute Streaming Query'}
          </Button>

          {(streamingQuery.isLoading || streamingQuery.isStreaming) && (
            <Button variant="outline" onClick={streamingQuery.cancel}>
              Cancel
            </Button>
          )}
        </div>

        {streamingQuery.progress && (
          <div className="mb-4 p-3 bg-primary/10 border border-primary rounded">
            <h4 className="font-medium text-primary">Query Progress:</h4>
            <div className="text-sm text-primary mt-1">
              <p>Rows Processed: {streamingQuery.progress.rowsProcessed}</p>
              <p>Elapsed Time: {streamingQuery.progress.elapsedTime}ms</p>
              <p>Throughput: {streamingQuery.progress.throughput?.toFixed(2)} rows/sec</p>
              {streamingQuery.progress.percentComplete !== undefined && (
                <p>Progress: {streamingQuery.progress.percentComplete.toFixed(1)}%</p>
              )}
              {streamingQuery.progress.currentPhase && (
                <p>Phase: {streamingQuery.progress.currentPhase}</p>
              )}
            </div>
          </div>
        )}

        {streamingQuery.columns && (
          <div className="mb-4 p-3 bg-gray-50 border border-gray-200 rounded">
            <h4 className="font-medium text-gray-800">Column Metadata:</h4>
            <div className="text-sm text-gray-700 mt-1">
              {streamingQuery.columns.map((col, idx) => (
                <span key={idx} className="inline-block mr-4">
                  {col.name} ({col.type})
                </span>
              ))}
            </div>
          </div>
        )}

        {streamingQuery.result && (
          <div className="mt-4 p-3 bg-primary/10 border border-primary rounded">
            <h4 className="font-medium text-primary">Query Result:</h4>
            <div className="text-sm text-primary mt-1">
              <p>Total Rows: {streamingQuery.result.metadata.rowCount}</p>
              <p>Execution Time: {streamingQuery.result.metadata.executionTime}ms</p>
              <p>Bytes Transferred: {streamingQuery.result.metadata.bytesTransferred}</p>

              {streamingQuery.result.rows.length > 0 && (
                <div className="mt-2">
                  <p className="font-medium">Sample Rows (first 5):</p>
                  <div className="overflow-x-auto">
                    <table className="min-w-full text-xs">
                      <thead>
                        <tr>
                          {streamingQuery.columns?.map((col, idx) => (
                            <th key={idx} className="border px-1 py-1">{col.name}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {streamingQuery.result.rows.slice(0, 5).map((row: unknown, idx: number) => (
                          <tr key={idx}>
                            {(row as unknown[]).map((cell: unknown, cellIdx: number) => (
                              <td key={cellIdx} className="border px-1 py-1">
                                {String(cell)}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {streamingQuery.error && (
          <div className="mt-4 p-3 bg-destructive/10 border border-destructive rounded">
            <h4 className="font-medium text-destructive">Query Error:</h4>
            <p className="text-sm text-destructive mt-1">{streamingQuery.error.message}</p>
          </div>
        )}
      </Card>

      {/* Debug Information */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Debug Information</h3>

        <div className="text-sm space-y-2">
          <p><strong>Environment:</strong></p>
          <p>gRPC Endpoint: {import.meta.env.VITE_GRPC_ENDPOINT || 'http://localhost:9500'}</p>
          <p>HTTP Gateway: {import.meta.env.VITE_HTTP_GATEWAY_ENDPOINT || 'http://localhost:8500'}</p>

          <p className="mt-4"><strong>Current State:</strong></p>
          <p>Connections Loading: {connectionsLoading.toString()}</p>
          <p>Streaming Query Loading: {streamingQuery.isLoading.toString()}</p>
          <p>Streaming Query Active: {streamingQuery.isStreaming.toString()}</p>
          <p>Row Count: {streamingQuery.rowCount}</p>
        </div>
      </Card>
    </div>
  )
}