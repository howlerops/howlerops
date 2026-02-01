import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { toast } from "@/hooks/use-toast"
import type { ColumnLoader } from "@/lib/codemirror-sql"
import { waitForWails } from "@/lib/wails-runtime"
import { type DatabaseConnection,useConnectionStore } from "@/store/connection-store"

import type { QueryMode } from "../types"
import { getConnectionKeys,shouldExcludeTable } from "../utils"

interface UseMultiDBProps {
  mode: QueryMode
  connections: DatabaseConnection[]
  activeConnection: DatabaseConnection | null
  schema: SchemaNode[]
}

/**
 * Hook that manages multi-database schemas and column loading
 */
export function useMultiDB({ mode, connections, activeConnection, schema }: UseMultiDBProps) {
  const { connectToDatabase } = useConnectionStore()

  // Multi-DB schemas state
  const [multiDBSchemas, setMultiDBSchemas] = useState<Map<string, SchemaNode[]>>(new Map())
  const multiDBSchemasRef = useRef<Map<string, SchemaNode[]>>(new Map())

  // Column cache for lazy loading
  const columnCacheRef = useRef<Map<string, SchemaNode[]>>(new Map())

  // Connection database state
  const [connectionDatabases, setConnectionDatabases] = useState<Record<string, string[]>>({})
  const [connectionDbLoading, setConnectionDbLoading] = useState<Record<string, boolean>>({})
  const [connectionDbSwitching, setConnectionDbSwitching] = useState<Record<string, boolean>>({})

  // Keep ref in sync with state
  useEffect(() => {
    multiDBSchemasRef.current = multiDBSchemas
  }, [multiDBSchemas])

  // Build single connection schemas map
  const singleConnectionSchemas = useMemo(() => {
    if (!activeConnection?.id || schema.length === 0) {
      return new Map<string, SchemaNode[]>()
    }

    const map = new Map<string, SchemaNode[]>()

    map.set(activeConnection.id, schema)
    if (activeConnection.name && activeConnection.name !== activeConnection.id) {
      map.set(activeConnection.name, schema)
    }

    if (activeConnection.name) {
      const slug = activeConnection.name.replace(/[^\w-]/g, '-')
      if (slug && slug !== activeConnection.name) {
        map.set(slug, schema)
      }
    }

    return map
  }, [activeConnection?.id, activeConnection?.name, schema])

  // Connection map for quick lookups
  const connectionMap = useMemo(() => {
    const map = new Map<string, DatabaseConnection>()
    connections.forEach((conn) => map.set(conn.id, conn))
    return map
  }, [connections])

  // Ensure connection databases are loaded
  const ensureConnectionDatabases = useCallback(async (connectionId: string) => {
    if (!connectionId) return
    if (connectionDatabases[connectionId] || connectionDbLoading[connectionId]) return

    const connection = connectionMap.get(connectionId)
    if (!connection || !connection.sessionId || !connection.isConnected) return

    setConnectionDbLoading((prev) => ({ ...prev, [connectionId]: true }))
    try {
      const dbs = await useConnectionStore.getState().fetchDatabases(connectionId)
      setConnectionDatabases((prev) => ({ ...prev, [connectionId]: dbs }))
    } catch (error) {
      toast({
        title: 'Unable to load databases',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        variant: 'destructive',
      })
    } finally {
      setConnectionDbLoading((prev) => ({ ...prev, [connectionId]: false }))
    }
  }, [connectionDatabases, connectionDbLoading, connectionMap])

  // Handle database change for a connection
  const handleConnectionDatabaseChange = useCallback(async (connectionId: string, database: string) => {
    if (!connectionId || !database) return

    const connection = connectionMap.get(connectionId)
    if (!connection || connection.database === database) return

    setConnectionDbSwitching((prev) => ({ ...prev, [connectionId]: true }))
    try {
      await useConnectionStore.getState().switchDatabase(connectionId, database)
      toast({
        title: 'Database switched',
        description: `${connection.name || 'Connection'} is now using ${database}.`,
      })
    } catch (error) {
      toast({
        title: 'Failed to switch database',
        description: error instanceof Error ? error.message : 'Unable to switch database',
        variant: 'destructive',
      })
    } finally {
      setConnectionDbSwitching((prev) => ({ ...prev, [connectionId]: false }))
    }
  }, [connectionMap])

  // Load schemas for all connections in multi-DB mode
  const loadMultiDBSchemas = useCallback(async () => {
    const relevantConnections = mode === 'multi' ? connections.filter(c => c.isConnected) : connections

    try {
      // Step 1: Auto-connect disconnected connections
      const disconnected = relevantConnections.filter(c => !c.isConnected)

      if (disconnected.length > 0) {
        await Promise.allSettled(
          disconnected.map(async (conn) => {
            await connectToDatabase(conn.id)
          })
        )

        // Wait for state to update
        await new Promise(resolve => setTimeout(resolve, 100))
      }

      // Step 2: Get session IDs
      const connectedWithSessions = relevantConnections.filter(c => c.isConnected && c.sessionId)
      const sessionIds = connectedWithSessions.map(c => c.sessionId!)

      if (sessionIds.length === 0) {
        setMultiDBSchemas(new Map())
        return
      }

      // Step 3: Load schemas
      try {
        const { GetMultiConnectionSchema } = await import('../../../../wailsjs/go/main/App')
        const combined = await GetMultiConnectionSchema(sessionIds)

        if (!combined || !combined.connections) {
          setMultiDBSchemas(new Map())
          return
        }

        // Convert to SchemaNode format
        const schemasMap = new Map<string, SchemaNode[]>()

        for (const [sessionId, connSchemaRaw] of Object.entries(combined.connections || {})) {
          const connSchema = connSchemaRaw as { schemas?: string[]; tables?: Array<{ name: string; schema: string }> }
          const schemaNodes: SchemaNode[] = []
          const connection = connectedWithSessions.find(c => c.sessionId === sessionId)

          const schemaNames = connSchema.schemas || []
          const tables = connSchema.tables || []

          for (const schemaName of schemaNames) {
            const schemaTables = tables.filter(t => t.schema === schemaName)
            const nonMigrationTables = schemaTables.filter(t => !shouldExcludeTable(t.name, schemaName))

            if (nonMigrationTables.length === 0) continue

            const tablesWithColumns: SchemaNode[] = nonMigrationTables.map(table => ({
              id: `${sessionId}-${schemaName}-${table.name}`,
              name: table.name,
              type: 'table' as const,
              schema: table.schema,
              sessionId,
              children: []
            }))

            schemaNodes.push({
              id: `${sessionId}-${schemaName}`,
              name: schemaName,
              type: 'schema' as const,
              children: tablesWithColumns
            })
          }

          // Store by connection keys
          if (connection) {
            const keys = getConnectionKeys(connection.id, connection.name)
            keys.forEach(key => {
              schemasMap.set(key, schemaNodes)
            })

            // Update both state and ref immediately
            const newMap = new Map(schemasMap)
            setMultiDBSchemas(newMap)
            multiDBSchemasRef.current = newMap
          }
        }

        // Final update
        setMultiDBSchemas(schemasMap)
        multiDBSchemasRef.current = schemasMap
      } catch {
        setMultiDBSchemas(new Map())
      }
    } catch {
      setMultiDBSchemas(new Map())
    }
  }, [mode, connections, connectToDatabase])

  // Load schemas when in multi-DB mode
  useEffect(() => {
    if (mode !== 'multi') return

    const connectedConnections = connections.filter(c => c.isConnected)
    if (connectedConnections.length === 0) {
      const emptyMap = new Map<string, SchemaNode[]>()
      setMultiDBSchemas(emptyMap)
      multiDBSchemasRef.current = emptyMap
      return
    }

    loadMultiDBSchemas()
  }, [mode, connections, loadMultiDBSchemas])

  // Column loader for CodeMirror
  const columnLoader: ColumnLoader = useCallback(async (sessionId: string, schema: string, tableName: string) => {
    try {
      const isReady = await waitForWails(2000)

      if (!isReady) {
        console.warn('[ColumnLoader] Wails runtime not ready')
        return []
      }

      const { GetTableStructure } = await import('../../../../wailsjs/go/main/App')
      const structure = await GetTableStructure(sessionId, schema, tableName)

      if (!structure || !structure.columns || structure.columns.length === 0) {
        return []
      }

      return structure.columns.map((col: { name: string; data_type?: string; nullable?: boolean; primary_key?: boolean }) => ({
        name: col.name,
        dataType: col.data_type || 'unknown',
        nullable: col.nullable,
        primaryKey: col.primary_key
      }))
    } catch (error) {
      console.error('[ColumnLoader] Failed to load columns:', {
        sessionId,
        schema,
        tableName,
        error: error instanceof Error ? error.message : String(error)
      })
      return []
    }
  }, [])

  // Get editor schemas based on mode
  const editorSchemas = mode === 'multi' ? multiDBSchemas : singleConnectionSchemas

  return {
    // Schemas
    multiDBSchemas,
    setMultiDBSchemas,
    multiDBSchemasRef,
    singleConnectionSchemas,
    editorSchemas,

    // Column cache
    columnCacheRef,
    columnLoader,

    // Connection databases
    connectionMap,
    connectionDatabases,
    setConnectionDatabases,
    connectionDbLoading,
    setConnectionDbLoading,
    connectionDbSwitching,
    setConnectionDbSwitching,

    // Actions
    ensureConnectionDatabases,
    handleConnectionDatabaseChange,
    loadMultiDBSchemas,
  }
}
