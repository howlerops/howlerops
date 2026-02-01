import type { SyntheticEvent } from "react"

import type { ColumnInfo, SchemaInfo, TableInfo } from "@/components/visual-query-builder/types"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import type { ColumnLoader } from "@/lib/codemirror-sql"
import type { QueryIR } from "@/lib/query-ir"
import type { DatabaseConnection } from "@/store/connection-store"
import type { QueryTab } from "@/store/query-store"

/**
 * SQL dialect types for query generation
 */
export type SqlDialect = 'postgres' | 'mysql' | 'sqlite' | 'mssql'

/**
 * Query mode - single database or multi-database
 */
export type QueryMode = 'single' | 'multi'

/**
 * AI sidebar mode - sql assistant or generic chat
 */
export type AISidebarMode = 'sql' | 'generic'

/**
 * AI sheet tab types
 */
export type AISheetTab = 'assistant' | 'memories'

/**
 * Props for the main QueryEditor component
 */
export interface QueryEditorProps {
  mode?: QueryMode
}

/**
 * Handle exposed by QueryEditor via forwardRef
 */
export interface QueryEditorHandle {
  openAIFix: (error: string, query: string) => void
  handlePageChange: (tabId: string, limit: number, offset: number) => Promise<void>
}

/**
 * Pagination state for a tab
 */
export interface TabPaginationState {
  limit: number
  offset: number
}

/**
 * Connection info for CodeMirror editor
 */
export interface CodeMirrorConnection {
  id: string
  name: string
  type: string
  database?: string
  sessionId?: string
  isConnected: boolean
  alias: string
}

/**
 * Editor state managed by the use-editor-state hook
 */
export interface EditorState {
  editorContent: string
  setEditorContent: (content: string) => void
  naturalLanguagePrompt: string
  setNaturalLanguagePrompt: (prompt: string) => void
  isVisualMode: boolean
  setIsVisualMode: (mode: boolean) => void
  visualQueryIR: QueryIR | null
  setVisualQueryIR: (ir: QueryIR | null) => void
  showAIDialog: boolean
  setShowAIDialog: (show: boolean) => void
  showSavedQueries: boolean
  setShowSavedQueries: (show: boolean) => void
  showDiagnostics: boolean
  setShowDiagnostics: (show: boolean) => void
  showConnectionSelector: boolean
  setShowConnectionSelector: (show: boolean) => void
  showDatabasePrompt: boolean
  setShowDatabasePrompt: (show: boolean) => void
  showSaveQueryDialog: boolean
  setShowSaveQueryDialog: (show: boolean) => void
  aiSidebarMode: AISidebarMode
  setAISidebarMode: (mode: AISidebarMode) => void
  aiSheetTab: AISheetTab
  setAISheetTab: (tab: AISheetTab) => void
  isFixMode: boolean
  setIsFixMode: (mode: boolean) => void
  lastExecutionError: string | null
  setLastExecutionError: (error: string | null) => void
  lastConnectionError: string | null
  setLastConnectionError: (error: string | null) => void
  appliedSuggestionId: string | null
  setAppliedSuggestionId: (id: string | null) => void
  pendingQuery: string | null
  setPendingQuery: (query: string | null) => void
  openConnectionPopover: string | null
  setOpenConnectionPopover: (id: string | null) => void
}

/**
 * Multi-DB state managed by the use-multi-db hook
 */
export interface MultiDBState {
  multiDBSchemas: Map<string, SchemaNode[]>
  setMultiDBSchemas: (schemas: Map<string, SchemaNode[]>) => void
  columnCache: Map<string, SchemaNode[]>
  connectionDatabases: Record<string, string[]>
  setConnectionDatabases: (databases: Record<string, string[]>) => void
  connectionDbLoading: Record<string, boolean>
  setConnectionDbLoading: (loading: Record<string, boolean>) => void
  connectionDbSwitching: Record<string, boolean>
  setConnectionDbSwitching: (switching: Record<string, boolean>) => void
  loadMultiDBSchemas: () => Promise<void>
  columnLoader: ColumnLoader
}

/**
 * Props for tab-related components
 */
export interface TabProps {
  tab: QueryTab
  isActive: boolean
  mode: QueryMode
  connections: DatabaseConnection[]
  onTabClick: (tabId: string) => void
  onCloseTab: (tabId: string, e: SyntheticEvent) => void
  onConnectionChange: (tabId: string, connectionId: string) => void
  onMultiDBConnectionsChange: (tabId: string, connectionIds: string[]) => void
}

/**
 * Props for the editor toolbar component
 */
export interface EditorToolbarProps {
  mode: QueryMode
  editorContent: string
  isExecuting: boolean
  isVisualMode: boolean
  isGenerating: boolean
  hasExecutionError: boolean
  aiEnabled: boolean
  canSaveQuery: boolean
  onExecute: () => void
  onToggleVisualMode: () => void
  onFixWithAI: () => void
  onSaveQuery: () => void
  onOpenQueryLibrary: () => void
}

/**
 * Props for the header bar component
 */
export interface HeaderBarProps {
  mode: QueryMode
  canToggle: boolean
  connectionCount: number
  activeEnvironmentFilter: string | null
  connectedCount: number
  totalCount: number
  aiEnabled: boolean
  showAIDialog: boolean
  aiSidebarMode: AISidebarMode
  showDiagnostics: boolean
  onToggleMode: () => void
  onSetAISidebarMode: (mode: AISidebarMode) => void
  onSetShowAIDialog: (show: boolean) => void
  onToggleDiagnostics: () => void
}

/**
 * Props for the AI sidebar component
 */
export interface AISidebarProps {
  mode: QueryMode
  open: boolean
  isFixMode: boolean
  aiSheetTab: AISheetTab
  naturalLanguagePrompt: string
  lastExecutionError: string | null
  lastError: string | null
  isGenerating: boolean
  suggestions: Array<{
    id: string
    query: string
    description?: string
    explanation?: string
  }>
  appliedSuggestionId: string | null
  memorySessions: Array<{
    id: string
    title: string
    createdAt: number
    updatedAt?: number
    messages: unknown[]
    summary?: string
  }>
  activeMemorySessionId: string | null
  activeTab: QueryTab | undefined
  connections: DatabaseConnection[]
  environmentFilteredConnections: DatabaseConnection[]
  multiDBSchemas: Map<string, SchemaNode[]>
  schema: SchemaNode[]
  activeConnection: DatabaseConnection | null
  canToggle: boolean
  isConnecting: boolean
  activeDatabaseSelector: React.ReactNode
  onClose: () => void
  onSetAISheetTab: (tab: AISheetTab) => void
  onSetNaturalLanguagePrompt: (prompt: string) => void
  onGenerateSQL: () => void
  onApplySuggestion: (query: string, id: string) => void
  onResetAISession: () => void
  onCreateMemorySession: () => void
  onDeleteMemorySession: (id: string) => void
  onClearAllMemories: () => void
  onResumeMemorySession: (id: string) => void
  onRenameSession: (id: string, title: string) => void
  onTabConnectionChange: (tabId: string, connectionId: string) => void
  onShowConnectionSelector: () => void
  onToggleMode: () => void
}

/**
 * Props for the empty state component
 */
export interface EmptyStateProps {
  onCreateSqlTab: () => void
  onCreateAiTab: () => void
}

/**
 * Re-export commonly used external types
 */
export type { ColumnInfo, ColumnLoader,DatabaseConnection, QueryIR, QueryTab, SchemaInfo, SchemaNode, TableInfo }
