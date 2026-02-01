import { useCallback, useEffect, useState } from 'react'

import { toast } from '../../../hooks/use-toast'
import { useConnectionStore } from '../../../store/connection-store'
import type { QueryResultRow } from '../../../store/query-store'
import type { EditableTableContext } from '../../../types/table'

interface UseDatabaseSelectorOptions {
  connectionId?: string
  resultId: string
  tableContextRef: React.MutableRefObject<EditableTableContext | null>
  updateResultRows: (resultId: string, rows: QueryResultRow[], originalRows: Record<string, QueryResultRow>) => void
  setDirtyRowIds: React.Dispatch<React.SetStateAction<string[]>>
  setPendingDeleteIds: React.Dispatch<React.SetStateAction<string[]>>
  clearJsonViewer: () => void
}

interface UseDatabaseSelectorReturn {
  databaseList: string[]
  databaseLoading: boolean
  databaseSelectorEnabled: boolean
  isSwitchingDatabase: boolean
  activeDatabase: string | undefined
  handleDatabaseSelection: (nextDatabase: string) => Promise<void>
}

export function useDatabaseSelector({
  connectionId,
  resultId,
  tableContextRef,
  updateResultRows,
  setDirtyRowIds,
  setPendingDeleteIds,
  clearJsonViewer,
}: UseDatabaseSelectorOptions): UseDatabaseSelectorReturn {
  const fetchDatabases = useConnectionStore((state) => state.fetchDatabases)
  const switchConnectionDatabase = useConnectionStore((state) => state.switchDatabase)
  const activeDatabase = useConnectionStore(
    useCallback((state) => {
      if (connectionId) {
        const connection = state.connections.find((conn) => conn.id === connectionId)
        if (connection) {
          return connection.database
        }
      }
      return state.activeConnection?.database
    }, [connectionId])
  )

  const [databaseList, setDatabaseList] = useState<string[]>([])
  const [databaseLoading, setDatabaseLoading] = useState(false)
  const [databaseSelectorEnabled, setDatabaseSelectorEnabled] = useState(false)
  const [isSwitchingDatabase, setIsSwitchingDatabase] = useState(false)

  useEffect(() => {
    if (!connectionId) {
      setDatabaseList([])
      setDatabaseSelectorEnabled(false)
      return
    }

    let cancelled = false
    setDatabaseLoading(true)

    fetchDatabases(connectionId)
      .then((databases) => {
        if (cancelled) {
          return
        }
        setDatabaseList(databases)
        setDatabaseSelectorEnabled(databases.length > 1)
      })
      .catch((error) => {
        if (cancelled) {
          return
        }
        setDatabaseList([])
        setDatabaseSelectorEnabled(false)
        if (error instanceof Error && !error.message.toLowerCase().includes('not supported')) {
          console.warn('Failed to load databases for connection', error)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setDatabaseLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [connectionId, fetchDatabases])

  const handleDatabaseSelection = useCallback(async (nextDatabase: string) => {
    if (!connectionId || !nextDatabase || nextDatabase === (activeDatabase ?? '')) {
      return
    }

    setIsSwitchingDatabase(true)
    try {
      await switchConnectionDatabase(connectionId, nextDatabase)
      setDirtyRowIds([])
      setPendingDeleteIds([])
      tableContextRef.current?.actions.clearDirtyRows?.()
      tableContextRef.current?.actions.clearInvalidCells?.()
      tableContextRef.current?.actions.resetTable?.()
      updateResultRows(resultId, [], {})
      clearJsonViewer()

      toast({
        title: 'Database switched',
        description: `Active database is now ${nextDatabase}.`,
        variant: 'default'
      })
    } catch (error) {
      toast({
        title: 'Failed to switch database',
        description: error instanceof Error ? error.message : 'Unable to switch database',
        variant: 'destructive'
      })
    } finally {
      setIsSwitchingDatabase(false)
    }
  }, [
    connectionId,
    activeDatabase,
    switchConnectionDatabase,
    setDirtyRowIds,
    setPendingDeleteIds,
    tableContextRef,
    updateResultRows,
    resultId,
    clearJsonViewer,
  ])

  return {
    databaseList,
    databaseLoading,
    databaseSelectorEnabled,
    isSwitchingDatabase,
    activeDatabase,
    handleDatabaseSelection,
  }
}
