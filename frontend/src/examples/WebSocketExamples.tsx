/**
 * WebSocket Examples - Demo components showing WebSocket functionality
 * Provides examples of all WebSocket features and components
 */

import React, { useState } from 'react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Textarea } from '../components/ui/textarea';
import { Input } from '../components/ui/input';
import { Badge } from '../components/ui/badge';
import {
  WebSocketProvider,
  useWebSocketContext,
  // useRealtimeQuery,
  // useTableSync,
  ConnectionStatusIndicator,
  OptimisticUpdateIndicator,
  RealtimeTableEditor,
  QueryProgressIndicator,
  WebSocketDebugPanel,
  ConflictResolutionModal,
} from '../hooks/websocket';

/**
 * Main WebSocket Examples Component
 */
export function WebSocketExamples() {
  const [showDebugPanel, setShowDebugPanel] = useState(false);

  return (
    <WebSocketProvider options={{ autoConnect: true }}>
      <div className="container mx-auto p-6 space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">WebSocket Examples</h1>
          <div className="flex items-center gap-2">
            <ConnectionStatusIndicator showDetails />
            <Button
              variant="outline"
              onClick={() => setShowDebugPanel(!showDebugPanel)}
            >
              Debug Panel
            </Button>
          </div>
        </div>

        <Tabs defaultValue="connection" className="space-y-6">
          <TabsList>
            <TabsTrigger value="connection">Connection</TabsTrigger>
            <TabsTrigger value="queries">Real-time Queries</TabsTrigger>
            <TabsTrigger value="tables">Table Sync</TabsTrigger>
            <TabsTrigger value="optimistic">Optimistic Updates</TabsTrigger>
            <TabsTrigger value="conflicts">Conflict Resolution</TabsTrigger>
          </TabsList>

          <TabsContent value="connection">
            <ConnectionExample />
          </TabsContent>

          <TabsContent value="queries">
            <QueryExample />
          </TabsContent>

          <TabsContent value="tables">
            <TableSyncExample />
          </TabsContent>

          <TabsContent value="optimistic">
            <OptimisticUpdatesExample />
          </TabsContent>

          <TabsContent value="conflicts">
            <ConflictResolutionExample />
          </TabsContent>
        </Tabs>

        <WebSocketDebugPanel
          isVisible={showDebugPanel}
          onToggle={() => setShowDebugPanel(!showDebugPanel)}
        />
      </div>
    </WebSocketProvider>
  );
}

/**
 * Connection Management Example
 */
function ConnectionExample() {
  const { connectionState, connect, disconnect, joinRoom, leaveRoom, getRooms } = useWebSocketContext();

  const [roomId, setRoomId] = useState('');
  const [roomType, setRoomType] = useState<'table' | 'connection' | 'query'>('table');

  const rooms = getRooms();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Connection Status</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <span>Status:</span>
            <Badge variant={connectionState.status === 'connected' ? 'default' : 'secondary'}>
              {connectionState.status}
            </Badge>
          </div>

          {connectionState.error && (
            <div className="text-red-600 text-sm">
              Error: {connectionState.error}
            </div>
          )}

          <div className="flex gap-2">
            <Button
              onClick={connect}
              disabled={connectionState.status === 'connected'}
            >
              Connect
            </Button>
            <Button
              onClick={disconnect}
              disabled={connectionState.status === 'disconnected'}
              variant="outline"
            >
              Disconnect
            </Button>
          </div>

          {connectionState.serverInfo && (
            <div className="space-y-2 text-sm">
              <div>Server Version: {connectionState.serverInfo.version}</div>
              <div>Capabilities:</div>
              <ul className="list-disc list-inside ml-4">
                <li>Streaming: {connectionState.serverInfo.capabilities.streaming ? '✓' : '✗'}</li>
                <li>Compression: {connectionState.serverInfo.capabilities.compression ? '✓' : '✗'}</li>
                <li>Binary Protocol: {connectionState.serverInfo.capabilities.binaryProtocol ? '✓' : '✗'}</li>
              </ul>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Room Management</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="Room ID"
              value={roomId}
              onChange={(e) => setRoomId(e.target.value)}
            />
            <select
              value={roomType}
              onChange={(e) => setRoomType(e.target.value as 'table' | 'connection' | 'query')}
              className="px-3 py-2 border rounded"
            >
              <option value="table">Table</option>
              <option value="connection">Connection</option>
              <option value="query">Query</option>
            </select>
          </div>

          <div className="flex gap-2">
            <Button
              onClick={() => joinRoom(roomId, roomType)}
              disabled={!roomId || connectionState.status !== 'connected'}
            >
              Join Room
            </Button>
            <Button
              onClick={() => leaveRoom(roomId)}
              disabled={!roomId}
              variant="outline"
            >
              Leave Room
            </Button>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Active Rooms:</div>
            {rooms.length > 0 ? (
              <div className="space-y-1">
                {rooms.map(room => (
                  <div key={room.id} className="flex items-center justify-between text-sm">
                    <span>{room.id}</span>
                    <Badge variant="outline">{room.type}</Badge>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-gray-500 text-sm">No active rooms</div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Real-time Query Example
 */
function QueryExample() {
  const [sql, setSql] = useState('SELECT * FROM users LIMIT 100;');
  const [streaming, setStreaming] = useState(false);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Real-time Query Execution</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Textarea
            placeholder="Enter your SQL query..."
            value={sql}
            onChange={(e) => setSql(e.target.value)}
            rows={4}
          />

          <div className="flex items-center gap-2">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={streaming}
                onChange={(e) => setStreaming(e.target.checked)}
              />
              <span>Enable streaming</span>
            </label>
          </div>

          <QueryProgressIndicator
            connectionName="main"
            sql={sql}
            streaming={streaming}
            onQueryComplete={(result) => {
              console.log('Query completed:', result);
            }}
            onQueryError={(error) => {
              console.error('Query failed:', error);
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Table Synchronization Example
 */
function TableSyncExample() {
  const sampleColumns = [
    { key: 'id', name: 'ID', type: 'number', editable: false },
    { key: 'name', name: 'Name', type: 'string', editable: true },
    { key: 'email', name: 'Email', type: 'string', editable: true },
    { key: 'role', name: 'Role', type: 'string', editable: true },
    { key: 'active', name: 'Active', type: 'boolean', editable: true },
  ];

  const sampleData = [
    { rowId: 1, data: { id: 1, name: 'John Doe', email: 'john@example.com', role: 'Admin', active: true } },
    { rowId: 2, data: { id: 2, name: 'Jane Smith', email: 'jane@example.com', role: 'User', active: true } },
    { rowId: 3, data: { id: 3, name: 'Bob Wilson', email: 'bob@example.com', role: 'User', active: false } },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Real-time Table Editor</CardTitle>
      </CardHeader>
      <CardContent>
        <RealtimeTableEditor
          tableId="users"
          tableName="Users"
          columns={sampleColumns}
          initialData={sampleData}
          onDataChange={(data) => {
            console.log('Table data changed:', data);
          }}
        />
      </CardContent>
    </Card>
  );
}

/**
 * Optimistic Updates Example
 */
function OptimisticUpdatesExample() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Optimistic Updates</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <span>Updates Status:</span>
            <OptimisticUpdateIndicator className="inline-block" />
          </div>

          <div className="text-sm text-gray-600">
            Optimistic updates provide immediate feedback to users by applying changes
            locally before server confirmation. The indicator above shows pending,
            confirmed, and rejected updates.
          </div>

          <div className="space-y-2">
            <h4 className="font-medium">How it works:</h4>
            <ul className="list-disc list-inside text-sm space-y-1">
              <li>User makes an edit (e.g., changes a table cell)</li>
              <li>Change is applied immediately in the UI (optimistic update)</li>
              <li>Request is sent to server in the background</li>
              <li>If successful, the optimistic update is confirmed</li>
              <li>If failed, the change is rolled back and user is notified</li>
              <li>If there's a conflict, conflict resolution is triggered</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Conflict Resolution Example
 */
function ConflictResolutionExample() {
  const [showConflictModal, setShowConflictModal] = useState(false);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Conflict Resolution</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-sm text-gray-600">
            When multiple users edit the same data simultaneously, conflicts can occur.
            The system provides automatic and manual conflict resolution strategies.
          </div>

          <div className="space-y-4">
            <div>
              <h4 className="font-medium mb-2">Resolution Strategies:</h4>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="bg-gray-50 p-2 rounded">
                  <div className="font-medium">Last Write Wins</div>
                  <div className="text-gray-600">Use the most recent change</div>
                </div>
                <div className="bg-gray-50 p-2 rounded">
                  <div className="font-medium">First Write Wins</div>
                  <div className="text-gray-600">Keep the original change</div>
                </div>
                <div className="bg-gray-50 p-2 rounded">
                  <div className="font-medium">Smart Merge</div>
                  <div className="text-gray-600">Automatically merge compatible changes</div>
                </div>
                <div className="bg-gray-50 p-2 rounded">
                  <div className="font-medium">Manual Resolution</div>
                  <div className="text-gray-600">Let user choose the resolution</div>
                </div>
              </div>
            </div>

            <Button
              onClick={() => setShowConflictModal(true)}
              variant="outline"
            >
              Demo Conflict Resolution Modal
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConflictResolutionModal
        isOpen={showConflictModal}
        onClose={() => setShowConflictModal(false)}
        conflictId={null} // Would be a real conflict ID in practice
      />
    </div>
  );
}