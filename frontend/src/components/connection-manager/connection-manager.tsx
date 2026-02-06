import { ChevronDown, Download, File, Plus, Upload } from "lucide-react"
import { useEffect, useState } from "react"

import { ConnectionDiagnosticsModal } from "@/components/connection-diagnostics-modal"
import { Button } from "@/components/ui/button"
import { Dialog, DialogTrigger } from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import { ConnectionForm, ConnectionList, EnvironmentFilter, EnvImportDialog, ExportDialog, ImportDialog } from "./components"
import { useConnectionActions, useConnectionForm, useConnectionList } from "./hooks"
import type { ConnectionFormData, DatabaseConnection } from "./types"

interface ConnectionManagerProps {
  hideHeader?: boolean
}

/**
 * Main connection manager component - orchestrates sub-components and hooks
 */
export function ConnectionManager({ hideHeader = false }: ConnectionManagerProps) {
  // Connection actions (CRUD operations, store access)
  const {
    connections,
    isConnecting,
    availableEnvironments,
    activeEnvironmentFilter,
    setEnvironmentFilter,
    refreshAvailableEnvironments,
    handleSubmit,
    handleConnect,
    handleDelete,
  } = useConnectionActions()

  // Form state management
  const {
    formData,
    setFormData,
    editingConnectionId,
    isDialogOpen,
    setIsDialogOpen,
    submitError,
    setSubmitError,
    isTestingConnection,
    setIsTestingConnection,
    isSshSectionOpen,
    setIsSshSectionOpen,
    isVpcSectionOpen,
    setIsVpcSectionOpen,
    isAdvancedSshOpen,
    setIsAdvancedSshOpen,
    newEnvironment,
    setNewEnvironment,
    environmentOptions,
    handleTypeChange,
    handleEnvironmentToggle,
    handleAddEnvironment,
    handleRemoveEnvironment,
    handleEditConnection,
    handleCloseDialog,
  } = useConnectionForm({ availableEnvironments })

  // List filtering and grouping
  const {
    filteredConnections,
    groupedConnections,
    groupByEnvironment,
    setGroupByEnvironment,
  } = useConnectionList({
    connections,
    activeEnvironmentFilter,
    availableEnvironments,
  })

  // Diagnostics modal state
  const [diagnosticsConnection, setDiagnosticsConnection] = useState<DatabaseConnection | null>(null)

  // Export/Import dialog state
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false)
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false)
  const [isEnvImportDialogOpen, setIsEnvImportDialogOpen] = useState(false)

  // Refresh environments on mount
  useEffect(() => {
    refreshAvailableEnvironments()
  }, [refreshAvailableEnvironments])

  // Form submission handler
  const onFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitError(null)

    await handleSubmit(
      formData,
      editingConnectionId,
      () => {
        handleCloseDialog()
        refreshAvailableEnvironments()
      },
      setSubmitError,
      setIsTestingConnection
    )
  }

  // Form data change handler
  const onFormDataChange = (data: Partial<ConnectionFormData>) => {
    setFormData(prev => ({ ...prev, ...data }))
  }

  return (
    <div className={hideHeader ? "" : "p-6"}>
      <div className="flex flex-col gap-4 mb-6">
        {/* Header */}
        {!hideHeader && (
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div>
              <h1 className="text-2xl font-bold">Database Connections</h1>
              <p className="text-muted-foreground">Manage your database connections</p>
            </div>
          </div>
        )}

        {/* Add Connection Dialog + Export/Import */}
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-2">
            <Dialog
              open={isDialogOpen}
              onOpenChange={(open) => {
                if (open) {
                  setIsDialogOpen(true)
                } else {
                  handleCloseDialog()
                }
              }}
            >
              <DialogTrigger asChild>
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Add Connection
                </Button>
              </DialogTrigger>
              <ConnectionForm
              formData={formData}
              editingConnectionId={editingConnectionId}
              environmentOptions={environmentOptions}
              newEnvironment={newEnvironment}
              submitError={submitError}
              isTestingConnection={isTestingConnection}
              isSshSectionOpen={isSshSectionOpen}
              isVpcSectionOpen={isVpcSectionOpen}
              isAdvancedSshOpen={isAdvancedSshOpen}
              onSubmit={onFormSubmit}
              onFormDataChange={onFormDataChange}
              onTypeChange={handleTypeChange}
              onEnvironmentToggle={handleEnvironmentToggle}
              onAddEnvironment={handleAddEnvironment}
              onRemoveEnvironment={handleRemoveEnvironment}
              onNewEnvironmentChange={setNewEnvironment}
              onSshSectionOpenChange={setIsSshSectionOpen}
              onVpcSectionOpenChange={setIsVpcSectionOpen}
              onAdvancedSshOpenChange={setIsAdvancedSshOpen}
            />
            </Dialog>

            {/* Export/Import buttons */}
            <Button variant="outline" onClick={() => setIsExportDialogOpen(true)}>
              <Download className="h-4 w-4 mr-2" />
              Export
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  <Upload className="h-4 w-4 mr-2" />
                  Import
                  <ChevronDown className="h-4 w-4 ml-2" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setIsImportDialogOpen(true)}>
                  <Upload className="h-4 w-4 mr-2" />
                  Import from JSON
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setIsEnvImportDialogOpen(true)}>
                  <File className="h-4 w-4 mr-2" />
                  Import from .env
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Export Dialog */}
        <ExportDialog
          open={isExportDialogOpen}
          onOpenChange={setIsExportDialogOpen}
        />

        {/* Import Dialog */}
        <ImportDialog
          open={isImportDialogOpen}
          onOpenChange={setIsImportDialogOpen}
          onImportComplete={refreshAvailableEnvironments}
        />

        {/* Env Import Dialog */}
        <EnvImportDialog
          open={isEnvImportDialogOpen}
          onOpenChange={setIsEnvImportDialogOpen}
          onImportComplete={refreshAvailableEnvironments}
        />

        {/* Environment Filter */}
        <EnvironmentFilter
          availableEnvironments={availableEnvironments}
          activeEnvironmentFilter={activeEnvironmentFilter}
          groupByEnvironment={groupByEnvironment}
          onFilterChange={setEnvironmentFilter}
          onGroupByChange={setGroupByEnvironment}
        />
      </div>

      {/* Connection List */}
      <ConnectionList
        connections={filteredConnections}
        groupedConnections={groupedConnections}
        groupByEnvironment={groupByEnvironment}
        isConnecting={isConnecting}
        hasConnections={connections.length > 0}
        activeEnvironmentFilter={activeEnvironmentFilter}
        onAddConnection={() => setIsDialogOpen(true)}
        onEditConnection={handleEditConnection}
        onDeleteConnection={handleDelete}
        onConnectConnection={handleConnect}
        onDiagnosticsConnection={setDiagnosticsConnection}
        onClearEnvironmentFilter={() => setEnvironmentFilter(null)}
      />

      {/* Diagnostics Modal */}
      {diagnosticsConnection && (
        <ConnectionDiagnosticsModal
          connection={diagnosticsConnection}
          open={!!diagnosticsConnection}
          onClose={() => setDiagnosticsConnection(null)}
        />
      )}
    </div>
  )
}
