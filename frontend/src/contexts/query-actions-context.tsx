import { createContext, type ReactNode, useContext, useMemo } from 'react'

/**
 * QueryActionsContext provides query-related action callbacks to deeply nested components
 * without prop drilling through Dashboard -> ResultsPanel -> QueryResultsTable.
 *
 * This context handles:
 * - AI fix suggestions for failed queries
 * - Pagination changes for query results
 */

export interface QueryActionsContextType {
  /**
   * Callback to request AI assistance for fixing a failed query
   * @param error - The error message from the failed query
   * @param query - The SQL query that failed
   */
  onFixWithAI: ((error: string, query: string) => void) | null

  /**
   * Callback to handle pagination changes in query results
   * @param tabId - The ID of the tab containing the results
   * @param limit - Number of rows per page
   * @param offset - Starting row offset
   */
  onPageChange: ((tabId: string, limit: number, offset: number) => Promise<void>) | null
}

const QueryActionsContext = createContext<QueryActionsContextType | null>(null)

export interface QueryActionsProviderProps {
  children: ReactNode
  onFixWithAI?: (error: string, query: string) => void
  onPageChange?: (tabId: string, limit: number, offset: number) => Promise<void>
}

/**
 * Provider component that makes query action callbacks available to all descendants.
 *
 * Wrap this around the component tree that needs access to query actions:
 *
 * ```tsx
 * <QueryActionsProvider onFixWithAI={handleFix} onPageChange={handlePage}>
 *   <ResultsPanel />
 * </QueryActionsProvider>
 * ```
 */
export function QueryActionsProvider({
  children,
  onFixWithAI,
  onPageChange,
}: QueryActionsProviderProps) {
  const value = useMemo(
    () => ({
      onFixWithAI: onFixWithAI ?? null,
      onPageChange: onPageChange ?? null,
    }),
    [onFixWithAI, onPageChange]
  )

  return (
    <QueryActionsContext.Provider value={value}>
      {children}
    </QueryActionsContext.Provider>
  )
}

/**
 * Hook to access query action callbacks from any component within QueryActionsProvider.
 *
 * @throws Error if used outside of QueryActionsProvider
 *
 * ```tsx
 * function MyComponent() {
 *   const { onFixWithAI, onPageChange } = useQueryActions()
 *   // Use callbacks...
 * }
 * ```
 */
export function useQueryActions(): QueryActionsContextType {
  const context = useContext(QueryActionsContext)
  if (!context) {
    throw new Error('useQueryActions must be used within a QueryActionsProvider')
  }
  return context
}

/**
 * Hook to safely access query action callbacks - returns null values if outside provider.
 * Useful for components that may be rendered outside the provider context.
 */
export function useQueryActionsOptional(): QueryActionsContextType {
  const context = useContext(QueryActionsContext)
  return context ?? { onFixWithAI: null, onPageChange: null }
}
