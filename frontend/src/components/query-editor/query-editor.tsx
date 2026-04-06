import { Loader2 } from "lucide-react"
import { forwardRef, lazy, Suspense, useCallback, useImperativeHandle, useMemo } from "react"

import { AIQueryTabView } from "@/components/ai-query-tab"
import { CodeMirrorEditor } from "@/components/codemirror-editor"
import { MultiDBDiagnostics } from "@/components/debug/multi-db-diagnostics"
import { MultiDBConnectionSelector } from "@/components/multi-db-connection-selector"
import { SavedQueriesPanel } from "@/components/saved-queries/SavedQueriesPanel"
import { SaveQueryDialog } from "@/components/saved-queries/SaveQueryDialog"
import { SelectDatabasePrompt } from "@/components/select-database-prompt"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useQueryMode } from "@/hooks/use-query-mode"
import { useSchemaIntrospection } from "@/hooks/use-schema-introspection"
import { useTheme } from "@/hooks/use-theme"
import { preloadComponent } from "@/lib/component-preload"
import { generateSQL as generateSQLFromIR, type QueryIR } from "@/lib/query-ir"
import { useAIStore } from "@/store/ai-store"
import { useAuthStore } from "@/store/auth-store"
import { useConnectionStore } from "@/store/connection-store"
import { useQueryEditorStore } from "@/store/query-editor-store"

import { AISidebar, EditorToolbar, EmptyState, HeaderBar, QueryTabs } from "./components"
import {
  useAIIntegration,
  useEditorState,
  useKeyboardShortcuts,
  useMultiDB,
  useQueryExecution,
} from "./hooks"
import type { QueryEditorHandle, QueryEditorProps, SqlDialect } from "./types"
import { convertSchemaNodes, getDialectFromConnectionType } from "./utils"

const VisualQueryBuilder = lazy(() => import("@/components/visual-query-builder").then(m => ({ default: m.VisualQueryBuilder })))
const preloadVisualQueryBuilder = () => import("@/components/visual-query-builder").then(m => ({ default: m.VisualQueryBuilder as React.ComponentType<unknown> }))

export const QueryEditor = forwardRef<QueryEditorHandle, QueryEditorProps>(({ mode: propMode = 'single' }, ref) => {
  const { theme } = useTheme()
  const { mode, canToggle, toggleMode, connectionCount } = useQueryMode(propMode)
  const {
    activeConnection,
    connections,
    connectToDatabase,
    isConnecting,
    activeEnvironmentFilter,
    availableEnvironments,
    setEnvironmentFilter,
    setActiveConnection: setGlobalActiveConnection,
  } = useConnectionStore()
  const {
    tabs,
    activeTabId,
    createTab,
    closeTab,
    updateTab,
    setActiveTab,
  } = useQueryEditorStore()
  const { schema } = useSchemaIntrospection()
  const user = useAuthStore(state => state.user)

  // Editor state hook
  const editorState = useEditorState()
  const {
    editorRef,
    editorContentRef,
    pendingTabUpdateRef,
    editorContent,
    setEditorContent,
    naturalLanguagePrompt,
    setNaturalLanguagePrompt,
    showAIDialog,
    setShowAIDialog,
    aiSidebarMode,
    setAISidebarMode,
    aiSheetTab,
    setAISheetTab,
    isFixMode,
    setIsFixMode,
    appliedSuggestionId,
    setAppliedSuggestionId,
    isVisualMode,
    setIsVisualMode,
    visualQueryIR,
    setVisualQueryIR,
    showSavedQueries,
    setShowSavedQueries,
    showDiagnostics,
    setShowDiagnostics,
    showConnectionSelector,
    setShowConnectionSelector,
    showDatabasePrompt,
    setShowDatabasePrompt,
    showSaveQueryDialog,
    setShowSaveQueryDialog,
    lastExecutionError,
    setLastExecutionError,
    lastConnectionError,
    setLastConnectionError,
    pendingQuery,
    setPendingQuery,
    openConnectionPopover,
    setOpenConnectionPopover,
    renameSessionId,
    renameTitle,
    setRenameTitle,
    openRenameDialog,
    closeRenameDialog,
    flushTabUpdate,
    scheduleTabUpdate,
  } = editorState

  // Multi-DB hook
  const multiDB = useMultiDB({
    mode,
    connections,
    activeConnection,
    schema,
  })
  const {
    multiDBSchemas,
    multiDBSchemasRef,
    editorSchemas,
    columnCacheRef,
    columnLoader,
    connectionMap,
    connectionDatabases,
    connectionDbLoading,
    connectionDbSwitching,
    ensureConnectionDatabases,
    handleConnectionDatabaseChange,
    loadMultiDBSchemas,
  } = multiDB

  // Environment filtered connections
  const environmentFilteredConnections = useMemo(() => {
    if (!activeEnvironmentFilter) {
      return connections
    }
    return connections.filter((conn) => conn.environments?.includes(activeEnvironmentFilter))
  }, [connections, activeEnvironmentFilter])

  // AI integration hook
  const aiIntegration = useAIIntegration({
    mode,
    activeConnection,
    schema,
    environmentFilteredConnections,
    multiDBSchemas,
  })
  const {
    aiConfig,
    aiEnabled,
    isGenerating,
    lastError,
    suggestions,
    memorySessionsMap,
    activeMemorySessionId,
    createAgentSession,
    setActiveAgentSession,
    renameAgentSession,
    handleFixQueryError,
    handleGenerateSQL,
    handleResetAISession,
    handleCreateMemorySession,
    handleDeleteMemorySession,
    handleClearAllMemories,
    handleResumeMemorySession,
    handleRenameMemorySession,
    clearSuggestions,
  } = aiIntegration

  // Query execution hook
  const queryExecution = useQueryExecution({
    editorRef,
    editorContent,
    setEditorContent,
    setLastExecutionError,
    setPendingQuery,
    setShowDatabasePrompt,
    flushTabUpdate,
    pendingTabUpdateRef,
  })
  const { handlePageChange, handleExecuteQuery, handleDatabaseSelected } = queryExecution

  // Active tab
  const activeTab = tabs.find(tab => tab.id === activeTabId)

  // Get SQL dialect from active tab's connection
  const activeDialect = useMemo((): SqlDialect => {
    if (!activeTab) return 'postgres'

    if (activeTab.connectionId) {
      const conn = connections.find(c => c.id === activeTab.connectionId)
      if (conn) return getDialectFromConnectionType(conn.type)
    }

    if (activeTab.selectedConnectionIds?.length) {
      const conn = connections.find(c => c.id === activeTab.selectedConnectionIds![0])
      if (conn) return getDialectFromConnectionType(conn.type)
    }

    if (activeConnection) {
      return getDialectFromConnectionType(activeConnection.type)
    }

    return 'postgres'
  }, [activeTab, connections, activeConnection])

  // Memory sessions
  const memorySessions = useMemo(() =>
    Object.values(memorySessionsMap).sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0)),
    [memorySessionsMap]
  )

  // Editor connections for CodeMirror
  const editorConnections = useMemo(() => {
    if (mode === 'multi') {
      return connections.filter(conn => conn.isConnected)
    }
    if (activeConnection?.isConnected) {
      return [activeConnection]
    }
    return []
  }, [mode, connections, activeConnection])

  const codeMirrorConnections = useMemo(
    () => editorConnections.map(conn => ({
      id: conn.id,
      name: conn.name,
      type: conn.type,
      database: conn.database,
      sessionId: conn.sessionId,
      isConnected: conn.isConnected,
      alias: conn.parameters?.alias || conn.name,
    })),
    [editorConnections]
  )

  // AI tab connections
  const aiTabConnections = useMemo(() => {
    if (environmentFilteredConnections.length > 0) {
      return environmentFilteredConnections
    }
    return connections
  }, [environmentFilteredConnections, connections])

  // Environment options
  const environmentOptions = useMemo(() => {
    const envSet = new Set(availableEnvironments)
    if (activeEnvironmentFilter) {
      envSet.add(activeEnvironmentFilter)
    }
    return Array.from(envSet).sort((a, b) => a.localeCompare(b))
  }, [availableEnvironments, activeEnvironmentFilter])

  // Visual builder schemas
  const visualBuilderSchemas = useMemo(() => {
    const converted = new Map()
    for (const [connId, schemaNodes] of editorSchemas.entries()) {
      converted.set(connId, convertSchemaNodes(schemaNodes))
    }
    return converted
  }, [editorSchemas])

  // Active database selector for single mode
  const activeDatabaseSelector = useMemo(() => {
    if (mode !== 'single') return null
    if (!activeTab?.connectionId) return null

    const connection = connectionMap.get(activeTab.connectionId)
    if (!connection?.isConnected) return null

    const databases = connectionDatabases[activeTab.connectionId]
    const loading = connectionDbLoading[activeTab.connectionId]
    const switching = connectionDbSwitching[activeTab.connectionId]

    if (!databases) {
      if (loading) {
        return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
      }
      return null
    }

    if (databases.length <= 1) return null

    return (
      <div className="flex items-center gap-2">
        <Select
          value={connection.database}
          onValueChange={(value: string) => handleConnectionDatabaseChange(connection.id, value)}
          disabled={loading || switching}
        >
          <SelectTrigger className="h-8 w-44 text-xs">
            <SelectValue placeholder={loading ? 'Loading databases...' : 'Select database'} />
          </SelectTrigger>
          <SelectContent>
            {databases.map((dbName: string) => (
              <SelectItem key={dbName} value={dbName}>
                <span className="text-xs">{dbName}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {(loading || switching) && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
      </div>
    )
  }, [
    mode,
    activeTab?.connectionId,
    connectionDatabases,
    connectionDbLoading,
    connectionDbSwitching,
    connectionMap,
    handleConnectionDatabaseChange,
  ])

  // Tab creation handlers
  const handleCreateSqlTab = useCallback(() => {
    const tabId = createTab('New Query', {
      type: 'sql',
      connectionId: mode === 'single' ? activeConnection?.id : undefined,
    })

    if (mode === 'multi') {
      const allFilteredIds = environmentFilteredConnections.map(c => c.id)
      updateTab(tabId, {
        selectedConnectionIds: allFilteredIds,
        connectionId: undefined,
      })
    }

    setActiveTab(tabId)
    if (activeConnection && mode === 'single') {
      setGlobalActiveConnection(activeConnection)
    }
  }, [mode, createTab, activeConnection, environmentFilteredConnections, updateTab, setActiveTab, setGlobalActiveConnection])

  const handleCreateAiTab = useCallback(() => {
    const sessionId = createAgentSession({
      title: `AI Query ${new Date().toLocaleTimeString()}`,
      provider: aiConfig.provider,
      model: aiConfig.selectedModel,
    })

    const tabId = createTab('AI Query Agent', {
      type: 'ai',
      connectionId: activeConnection?.id,
      aiSessionId: sessionId,
    })

    updateTab(tabId, {
      connectionId: activeConnection?.id,
      selectedConnectionIds: activeConnection?.id ? [activeConnection.id] : [],
    })

    setActiveTab(tabId)
    setActiveAgentSession(sessionId)
    if (activeConnection) {
      setGlobalActiveConnection(activeConnection)
    }
  }, [createAgentSession, aiConfig.provider, aiConfig.selectedModel, createTab, activeConnection, updateTab, setActiveTab, setActiveAgentSession, setGlobalActiveConnection])

  // Tab connection change handler
  const handleTabConnectionChange = useCallback(
    async (tabId: string, connectionId: string) => {
      setLastConnectionError(null)
      updateTab(tabId, { connectionId, selectedConnectionIds: [connectionId] })

      const connection = connections.find(conn => conn.id === connectionId)
      if (!connection) {
        setGlobalActiveConnection(null)
        return
      }

      if (!connection.isConnected) {
        try {
          await connectToDatabase(connectionId)
        } catch (error) {
          console.error('Failed to connect to database:', error)
          const errorMessage = error instanceof Error ? error.message : 'Failed to connect to database'
          setLastConnectionError(errorMessage)
          return
        }
      }

      const updatedConnection = useConnectionStore.getState().connections.find(conn => conn.id === connectionId)
      if (updatedConnection && updatedConnection.isConnected) {
        setGlobalActiveConnection(updatedConnection)
      }
    },
    [connections, connectToDatabase, setGlobalActiveConnection, updateTab, setLastConnectionError]
  )

  // Handle use SQL from AI agent
  const handleUseSQLFromAgent = useCallback((sql: string, connectionId?: string) => {
    const targetConnectionId = connectionId ?? activeConnection?.id ?? undefined
    const tabId = createTab('AI Generated Query', {
      type: 'sql',
      connectionId: targetConnectionId,
    })

    updateTab(tabId, {
      content: sql,
      isDirty: true,
      connectionId: targetConnectionId,
      selectedConnectionIds: targetConnectionId ? [targetConnectionId] : [],
    })

    setActiveTab(tabId)
    setEditorContent(sql)
    if (targetConnectionId) {
      const connection = useConnectionStore.getState().connections.find(conn => conn.id === targetConnectionId)
      if (connection) {
        setGlobalActiveConnection(connection)
      }
    }
  }, [createTab, activeConnection?.id, updateTab, setActiveTab, setEditorContent, setGlobalActiveConnection])

  // Visual query builder handlers
  const handleVisualQueryChange = useCallback((queryIR: QueryIR) => {
    setVisualQueryIR(queryIR)

    try {
      const sql = generateSQLFromIR(queryIR, activeDialect)
      setEditorContent(sql)

      if (activeTab) {
        updateTab(activeTab.id, {
          content: sql,
          isDirty: sql !== activeTab.content
        })
      }
    } catch (error) {
      console.error('Failed to generate SQL from visual query:', error)
    }
  }, [activeTab, updateTab, activeDialect, setVisualQueryIR, setEditorContent])

  const handleVisualSQLChange = useCallback((sql: string) => {
    setEditorContent(sql)

    if (activeTab) {
      updateTab(activeTab.id, {
        content: sql,
        isDirty: sql !== activeTab.content
      })
    }
  }, [activeTab, updateTab, setEditorContent])

  const handleVisualModeToggle = useCallback(() => {
    setIsVisualMode(prev => !prev)
    setVisualQueryIR(null)
  }, [setIsVisualMode, setVisualQueryIR])

  // Editor change handler
  const handleEditorChange = useCallback((value: string) => {
    setEditorContent(value)

    if (activeTab?.id) {
      scheduleTabUpdate(activeTab.id, value)
    }
  }, [activeTab?.id, scheduleTabUpdate, setEditorContent])

  // Tab click handler
  const handleTabClick = useCallback((tabId: string) => {
    setActiveTab(tabId)
    const tab = tabs.find(t => t.id === tabId)
    if (tab) {
      setEditorContent(tab.content)
    }
  }, [tabs, setActiveTab, setEditorContent])

  // Get connection label for tab
  const getConnectionLabelForTab = useCallback((tab: { connectionId?: string }) => {
    if (!tab.connectionId) {
      return 'Select DB'
    }
    const connection = connections.find((conn) => conn.id === tab.connectionId)
    return connection?.name || 'Select DB'
  }, [connections])

  // Get active connections for tab
  const getActiveConnectionsForTab = useCallback((tab: { connectionId?: string; selectedConnectionIds?: string[] }) => {
    if (mode === 'single') {
      return tab.connectionId ? [tab.connectionId] : []
    }
    if (tab.selectedConnectionIds && tab.selectedConnectionIds.length > 0) {
      return tab.selectedConnectionIds
    }
    return environmentFilteredConnections.map(c => c.id)
  }, [mode, environmentFilteredConnections])

  // Connection popover toggle
  const handleConnectionPopoverToggle = useCallback((tabId: string, open: boolean) => {
    setOpenConnectionPopover(open ? tabId : null)
    if (open) {
      const targetTab = tabs.find((tab) => tab.id === tabId)
      if (targetTab?.connectionId) {
        void ensureConnectionDatabases(targetTab.connectionId)
      }
    }
  }, [tabs, ensureConnectionDatabases, setOpenConnectionPopover])

  // Multi-DB connections change
  const handleMultiDBConnectionsChange = useCallback((tabId: string, connectionIds: string[]) => {
    updateTab(tabId, { selectedConnectionIds: connectionIds })
  }, [updateTab])

  // AI handlers
  const onGenerateSQL = useCallback(async () => {
    try {
      await handleGenerateSQL(naturalLanguagePrompt)
      setNaturalLanguagePrompt('')
    } catch (error) {
      setLastExecutionError(error instanceof Error ? error.message : 'Failed to generate SQL')
    }
  }, [handleGenerateSQL, naturalLanguagePrompt, setNaturalLanguagePrompt, setLastExecutionError])

  const onFixWithAI = useCallback(async () => {
    if (!lastExecutionError || !editorContent.trim() || !aiEnabled) return

    setIsFixMode(true)
    setShowAIDialog(true)
    await handleFixQueryError(lastExecutionError, editorContent)
  }, [lastExecutionError, editorContent, aiEnabled, setIsFixMode, setShowAIDialog, handleFixQueryError])

  const onApplySuggestion = useCallback((suggestionQuery: string, suggestionId: string) => {
    if (activeTab) {
      setEditorContent(suggestionQuery)
      updateTab(activeTab.id, {
        content: suggestionQuery,
        isDirty: true
      })
      if (editorRef.current) {
        editorRef.current.setValue(suggestionQuery)
      }
      setAppliedSuggestionId(suggestionId)
      setShowAIDialog(false)
      setIsFixMode(false)
    }
  }, [activeTab, updateTab, editorRef, setEditorContent, setAppliedSuggestionId, setShowAIDialog, setIsFixMode])

  const onConfirmRename = useCallback(() => {
    if (!renameSessionId) return
    handleRenameMemorySession(renameSessionId, renameTitle)
    closeRenameDialog()
  }, [renameSessionId, renameTitle, handleRenameMemorySession, closeRenameDialog])

  // Keyboard shortcuts
  useKeyboardShortcuts({
    editorRef,
    editorContentRef,
    activeConnection,
    isExecuting: !!activeTab?.isExecuting,
    user,
    editorContent,
    onExecuteQuery: handleExecuteQuery,
    onToggleDiagnostics: () => setShowDiagnostics(prev => !prev),
    onToggleSavedQueries: () => setShowSavedQueries(prev => !prev),
    onOpenSaveQueryDialog: () => setShowSaveQueryDialog(true),
    onCreateSqlTab: handleCreateSqlTab,
    onCreateAiTab: handleCreateAiTab,
  })

  // Expose methods to parent
  useImperativeHandle(ref, () => ({
    openAIFix: (error: string, query: string) => {
      setIsFixMode(true)
      setLastExecutionError(error)
      if (activeTab) {
        setEditorContent(query)
        updateTab(activeTab.id, { content: query })
      }
      setShowAIDialog(true)
      handleFixQueryError(error, query)
    },
    handlePageChange
  }), [activeTab, handleFixQueryError, updateTab, handlePageChange, setIsFixMode, setLastExecutionError, setEditorContent, setShowAIDialog])

  // Empty state
  if (tabs.length === 0) {
    return <EmptyState onCreateSqlTab={handleCreateSqlTab} onCreateAiTab={handleCreateAiTab} />
  }

  const isAiTab = !!activeTab && activeTab.type === 'ai'

  return (
    <div className="flex-1 flex h-full min-h-0 w-full flex-col">
      {/* Diagnostics Panel */}
      {showDiagnostics && (
        <MultiDBDiagnostics
          multiDBSchemas={multiDBSchemas}
          columnCache={columnCacheRef.current}
          onRefreshSchemas={loadMultiDBSchemas}
          onTestAutocomplete={() => {
            const testInput = "@Prod-Leviosa."
            const pattern = /@([\w-]+)\.(\w*)$/
            const match = testInput.match(pattern)
            if (match) {
              multiDBSchemasRef.current.get(match[1])
            }
          }}
        />
      )}

      {/* Header */}
      <div className="border-b bg-background">
        <HeaderBar
          mode={mode}
          canToggle={canToggle}
          connectionCount={connectionCount}
          activeEnvironmentFilter={activeEnvironmentFilter}
          connectedCount={environmentFilteredConnections.filter(c => c.isConnected).length}
          totalCount={environmentFilteredConnections.length}
          aiEnabled={aiEnabled}
          showAIDialog={showAIDialog}
          aiSidebarMode={aiSidebarMode}
          showDiagnostics={showDiagnostics}
          onToggleMode={toggleMode}
          onSetAISidebarMode={setAISidebarMode}
          onSetShowAIDialog={setShowAIDialog}
          onToggleDiagnostics={() => setShowDiagnostics(prev => !prev)}
          onSetIsFixMode={setIsFixMode}
          onSetAISheetTab={setAISheetTab}
        />

        {/* Tabs */}
        <QueryTabs
          tabs={tabs}
          activeTabId={activeTabId}
          mode={mode}
          connections={connections}
          environmentFilteredConnections={environmentFilteredConnections}
          environmentOptions={environmentOptions}
          activeEnvironmentFilter={activeEnvironmentFilter}
          isConnecting={isConnecting}
          openConnectionPopover={openConnectionPopover}
          lastConnectionError={lastConnectionError}
          onTabClick={handleTabClick}
          onCloseTab={(tabId, e) => {
            e.stopPropagation()
            closeTab(tabId)
          }}
          onConnectionChange={(tabId, connId) => {
            handleTabConnectionChange(tabId, connId)
            setOpenConnectionPopover(null)
          }}
          onConnectionPopoverToggle={handleConnectionPopoverToggle}
          onSetEnvironmentFilter={setEnvironmentFilter}
          onOpenConnectionSelector={(tabId) => {
            setActiveTab(tabId)
            setShowConnectionSelector(true)
          }}
          onCreateSqlTab={handleCreateSqlTab}
          onCreateAiTab={handleCreateAiTab}
          getConnectionLabelForTab={getConnectionLabelForTab}
          getActiveConnectionsForTab={getActiveConnectionsForTab}
        />
      </div>

      {isAiTab && activeTab ? (
        <AIQueryTabView
          tab={activeTab}
          connections={aiTabConnections}
          schemasMap={editorSchemas}
          onSelectConnection={(connectionId) => handleTabConnectionChange(activeTab.id, connectionId)}
          onUseSQL={handleUseSQLFromAgent}
          onRenameSession={renameAgentSession}
        />
      ) : (
        <>
          {/* AI Sidebar */}
          {aiEnabled && (
            <AISidebar
              mode={mode}
              open={showAIDialog}
              aiSidebarMode={aiSidebarMode}
              isFixMode={isFixMode}
              aiSheetTab={aiSheetTab}
              naturalLanguagePrompt={naturalLanguagePrompt}
              lastExecutionError={lastExecutionError}
              lastError={lastError}
              isGenerating={isGenerating}
              suggestions={suggestions}
              appliedSuggestionId={appliedSuggestionId}
              memorySessions={memorySessions}
              activeMemorySessionId={activeMemorySessionId}
              activeTab={activeTab}
              connections={connections}
              environmentFilteredConnections={environmentFilteredConnections}
              editorConnections={editorConnections}
              editorSchemas={editorSchemas}
              multiDBSchemas={multiDBSchemas}
              schema={schema}
              activeConnection={activeConnection}
              canToggle={canToggle}
              isConnecting={isConnecting}
              activeDatabaseSelector={activeDatabaseSelector}
              renameSessionId={renameSessionId}
              renameTitle={renameTitle}
              onClose={() => setShowAIDialog(false)}
              onSetIsFixMode={setIsFixMode}
              onSetAISheetTab={setAISheetTab}
              onSetNaturalLanguagePrompt={setNaturalLanguagePrompt}
              onGenerateSQL={onGenerateSQL}
              onApplySuggestion={onApplySuggestion}
              onResetAISession={() => {
                handleResetAISession()
                clearSuggestions()
                setAppliedSuggestionId(null)
              }}
              onCreateMemorySession={() => {
                const sessionId = handleCreateMemorySession()
                setAISheetTab('assistant')
                return sessionId
              }}
              onDeleteMemorySession={handleDeleteMemorySession}
              onClearAllMemories={handleClearAllMemories}
              onResumeMemorySession={(sessionId) => {
                handleResumeMemorySession(sessionId)
                setAISheetTab('assistant')
              }}
              onOpenRenameDialog={openRenameDialog}
              onCloseRenameDialog={closeRenameDialog}
              onConfirmRename={onConfirmRename}
              onSetRenameTitle={setRenameTitle}
              onTabConnectionChange={handleTabConnectionChange}
              onShowConnectionSelector={() => setShowConnectionSelector(true)}
              onToggleMode={toggleMode}
            />
          )}

          {/* Save Query Dialog */}
          <SaveQueryDialog
            open={showSaveQueryDialog}
            onClose={() => setShowSaveQueryDialog(false)}
            userId={user?.id ?? 'local-user'}
            initialQuery={editorRef.current?.getValue() ?? editorContent}
            onSaved={(query) => {
              console.log('Query saved:', query)
              setShowSaveQueryDialog(false)
            }}
          />

          {/* Toolbar */}
          <EditorToolbar
            mode={mode}
            editorContent={editorContent}
            isExecuting={!!activeTab?.isExecuting}
            isVisualMode={isVisualMode}
            isGenerating={isGenerating}
            hasExecutionError={!!lastExecutionError}
            aiEnabled={aiEnabled}
            connectionId={activeTab?.connectionId}
            onExecute={handleExecuteQuery}
            onToggleVisualMode={() => {
              handleVisualModeToggle()
              void preloadComponent(preloadVisualQueryBuilder)
            }}
            onFixWithAI={onFixWithAI}
            onSaveQuery={() => setShowSaveQueryDialog(true)}
            onOpenQueryLibrary={() => setShowSavedQueries(true)}
          />

          {/* Editor */}
          <div className="flex-1 min-h-0 overflow-y-auto">
            {isVisualMode ? (
              <Suspense fallback={
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
                </div>
              }>
                <div className="h-full p-4">
                  <VisualQueryBuilder
                    connections={connections.map(conn => ({
                      id: conn.id,
                      name: conn.name,
                      type: conn.type,
                      isConnected: conn.isConnected
                    }))}
                    schemas={visualBuilderSchemas}
                    onQueryChange={handleVisualQueryChange}
                    onSQLChange={handleVisualSQLChange}
                    initialQuery={visualQueryIR || undefined}
                  />
                </div>
              </Suspense>
            ) : (
              <CodeMirrorEditor
                ref={editorRef}
                value={editorContent}
                onChange={handleEditorChange}
                onMount={() => {}}
                onExecute={handleExecuteQuery}
                theme={theme === 'dark' ? 'dark' : 'light'}
                height="100%"
                connections={codeMirrorConnections}
                schemas={editorSchemas}
                mode={mode}
                columnLoader={columnLoader}
                className="h-full"
                aiEnabled={useAIStore.getState().config.enabled && useAIStore.getState().providerSynced}
                aiLanguage={'sql'}
              />
            )}
          </div>

          {/* Saved Queries Panel */}
          <SavedQueriesPanel
            open={showSavedQueries}
            onClose={() => setShowSavedQueries(false)}
            userId={user?.id ?? 'local-user'}
            onLoadQuery={(q) => {
              setEditorContent(q.query_text)
              setShowSavedQueries(false)
            }}
          />

          {/* Multi-DB Connection Selector */}
          {mode === 'multi' && activeTab && (
            <MultiDBConnectionSelector
              open={showConnectionSelector}
              onClose={() => setShowConnectionSelector(false)}
              selectedConnectionIds={getActiveConnectionsForTab(activeTab)}
              onSelectionChange={(connectionIds) => handleMultiDBConnectionsChange(activeTab.id, connectionIds)}
              filteredConnections={environmentFilteredConnections}
            />
          )}

          {/* Database Selection Prompt */}
          <SelectDatabasePrompt
            isOpen={showDatabasePrompt}
            onClose={() => {
              setShowDatabasePrompt(false)
              setPendingQuery(null)
            }}
            onSelect={(connectionId) => {
              handleDatabaseSelected(connectionId, pendingQuery, updateTab)
              setPendingQuery(null)
            }}
            connections={connections}
            currentConnectionId={activeTab?.connectionId}
          />
        </>
      )}
    </div>
  )
})

QueryEditor.displayName = 'QueryEditor'
