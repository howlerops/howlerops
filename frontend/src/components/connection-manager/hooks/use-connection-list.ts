import { useMemo, useState } from "react"

import type { ConnectionGroup, DatabaseConnection } from "../types"
import { UNASSIGNED_ENVIRONMENT_LABEL } from "../types"

interface UseConnectionListReturn {
  filteredConnections: DatabaseConnection[]
  groupedConnections: ConnectionGroup[]
  groupByEnvironment: boolean
  setGroupByEnvironment: (grouped: boolean) => void
}

interface UseConnectionListOptions {
  connections: DatabaseConnection[]
  activeEnvironmentFilter: string | null
  availableEnvironments: string[]
}

/**
 * Hook for filtering and grouping connections
 */
export function useConnectionList({
  connections,
  activeEnvironmentFilter,
  availableEnvironments,
}: UseConnectionListOptions): UseConnectionListReturn {
  const [groupByEnvironment, setGroupByEnvironment] = useState(false)

  // Filter connections by active environment
  const filteredConnections = useMemo(() => {
    if (!activeEnvironmentFilter) {
      return connections
    }

    return connections.filter((conn) => {
      if (!conn.environments || conn.environments.length === 0) {
        return false
      }
      return conn.environments.includes(activeEnvironmentFilter)
    })
  }, [connections, activeEnvironmentFilter])

  // Group connections by environment
  const groupedConnections = useMemo(() => {
    if (!groupByEnvironment) {
      return []
    }

    // Create environment order map for sorting
    const envOrder = new Map<string, number>()
    availableEnvironments.forEach((env, idx) => envOrder.set(env, idx))

    // Group connections by environment
    const groupMap = new Map<string, DatabaseConnection[]>()

    filteredConnections.forEach((conn) => {
      const connEnvs = conn.environments && conn.environments.length > 0
        ? conn.environments
        : [UNASSIGNED_ENVIRONMENT_LABEL]

      connEnvs.forEach((env) => {
        const key = env === UNASSIGNED_ENVIRONMENT_LABEL ? UNASSIGNED_ENVIRONMENT_LABEL : env
        if (!groupMap.has(key)) {
          groupMap.set(key, [])
        }
        groupMap.get(key)?.push(conn)
      })
    })

    // Convert to array and sort
    return Array.from(groupMap.entries())
      .map(([key, items]) => ({
        key,
        label: key === UNASSIGNED_ENVIRONMENT_LABEL ? UNASSIGNED_ENVIRONMENT_LABEL : key,
        connections: items,
      }))
      .sort((a, b) => {
        // Unassigned always at the end
        if (a.key === UNASSIGNED_ENVIRONMENT_LABEL) return 1
        if (b.key === UNASSIGNED_ENVIRONMENT_LABEL) return -1

        // Sort by environment order, then alphabetically
        const orderA = envOrder.get(a.key) ?? Number.MAX_SAFE_INTEGER
        const orderB = envOrder.get(b.key) ?? Number.MAX_SAFE_INTEGER

        if (orderA === orderB) {
          return a.label.localeCompare(b.label)
        }
        return orderA - orderB
      })
  }, [filteredConnections, groupByEnvironment, availableEnvironments])

  return {
    filteredConnections,
    groupedConnections,
    groupByEnvironment,
    setGroupByEnvironment,
  }
}
