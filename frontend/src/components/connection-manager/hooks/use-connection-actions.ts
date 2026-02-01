import { useCallback } from "react"
import { useShallow } from "zustand/react/shallow"

import { api } from "@/lib/api-client"
import { useConnectionStore } from "@/store/connection-store"

import type { ConnectionFormData, DatabaseConnection } from "../types"
import { buildConnectionPayload } from "../utils"

interface UseConnectionActionsReturn {
  connections: DatabaseConnection[]
  isConnecting: boolean
  availableEnvironments: string[]
  activeEnvironmentFilter: string | null
  setEnvironmentFilter: (filter: string | null) => void
  refreshAvailableEnvironments: () => void
  handleSubmit: (
    formData: ConnectionFormData,
    editingConnectionId: string | null,
    onSuccess: () => void,
    onError: (error: string) => void,
    setIsTestingConnection: (testing: boolean) => void
  ) => Promise<void>
  handleConnect: (connection: DatabaseConnection) => Promise<void>
  handleDelete: (connection: DatabaseConnection) => Promise<void>
}

/**
 * Hook for connection CRUD operations
 */
export function useConnectionActions(): UseConnectionActionsReturn {
  const {
    connections,
    addConnection,
    updateConnection,
    removeConnection,
    connectToDatabase,
    disconnectFromDatabase,
    isConnecting,
    availableEnvironments,
    refreshAvailableEnvironments,
    activeEnvironmentFilter,
    setEnvironmentFilter,
  } = useConnectionStore(useShallow((state) => ({
    connections: state.connections,
    addConnection: state.addConnection,
    updateConnection: state.updateConnection,
    removeConnection: state.removeConnection,
    connectToDatabase: state.connectToDatabase,
    disconnectFromDatabase: state.disconnectFromDatabase,
    isConnecting: state.isConnecting,
    availableEnvironments: state.availableEnvironments,
    refreshAvailableEnvironments: state.refreshAvailableEnvironments,
    activeEnvironmentFilter: state.activeEnvironmentFilter,
    setEnvironmentFilter: state.setEnvironmentFilter,
  })))

  // Handle form submission (create or update connection)
  const handleSubmit = useCallback(async (
    formData: ConnectionFormData,
    editingConnectionId: string | null,
    onSuccess: () => void,
    onError: (error: string) => void,
    setIsTestingConnection: (testing: boolean) => void
  ) => {
    setIsTestingConnection(true)

    const connectionData = buildConnectionPayload(formData)

    try {
      // Test connection first
      const result = await api.connections.test({
        ...connectionData,
        ssl_mode: connectionData.sslMode,
        connection_timeout: 30,
      })

      if (!result.success) {
        throw new Error(result.message || 'Connection test failed')
      }

      let connectionId: string

      if (editingConnectionId) {
        // Update existing connection
        await updateConnection(editingConnectionId, connectionData)
        connectionId = editingConnectionId
      } else {
        // Add new connection
        await addConnection(connectionData)
        // Get the ID of the newly added connection
        const state = useConnectionStore.getState()
        const newConnection = state.connections[state.connections.length - 1]
        connectionId = newConnection.id
      }

      refreshAvailableEnvironments()
      onSuccess()

      // Auto-connect after successful test
      try {
        await connectToDatabase(connectionId)
      } catch (connectError) {
        console.error('Failed to auto-connect:', connectError)
        // Don't show error since connection was tested successfully
      }
    } catch (error) {
      onError(error instanceof Error ? error.message : 'Failed to validate connection')
    } finally {
      setIsTestingConnection(false)
    }
  }, [addConnection, updateConnection, refreshAvailableEnvironments, connectToDatabase])

  // Toggle connection state
  const handleConnect = useCallback(async (connection: DatabaseConnection) => {
    try {
      if (connection.isConnected) {
        await disconnectFromDatabase(connection.id)
      } else {
        await connectToDatabase(connection.id)
      }
    } catch (error) {
      console.error('Connection toggle failed:', error)
    }
  }, [connectToDatabase, disconnectFromDatabase])

  // Delete connection
  const handleDelete = useCallback(async (connection: DatabaseConnection) => {
    if (connection.isConnected) {
      await disconnectFromDatabase(connection.id)
    }
    removeConnection(connection.id)
  }, [disconnectFromDatabase, removeConnection])

  return {
    connections,
    isConnecting,
    availableEnvironments,
    activeEnvironmentFilter,
    setEnvironmentFilter,
    refreshAvailableEnvironments,
    handleSubmit,
    handleConnect,
    handleDelete,
  }
}
