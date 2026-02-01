import { useCallback,useMemo, useState } from "react"

import type { ConnectionFormData, DatabaseConnection,DatabaseTypeString } from "../types"
import { createDefaultFormData, getDefaultPort, populateFormFromConnection } from "../utils"

interface UseConnectionFormReturn {
  formData: ConnectionFormData
  setFormData: React.Dispatch<React.SetStateAction<ConnectionFormData>>
  editingConnectionId: string | null
  setEditingConnectionId: (id: string | null) => void
  isDialogOpen: boolean
  setIsDialogOpen: (open: boolean) => void
  submitError: string | null
  setSubmitError: (error: string | null) => void
  isTestingConnection: boolean
  setIsTestingConnection: (testing: boolean) => void
  isSshSectionOpen: boolean
  setIsSshSectionOpen: (open: boolean) => void
  isVpcSectionOpen: boolean
  setIsVpcSectionOpen: (open: boolean) => void
  isAdvancedSshOpen: boolean
  setIsAdvancedSshOpen: (open: boolean) => void
  newEnvironment: string
  setNewEnvironment: (env: string) => void
  environmentOptions: string[]
  handleTypeChange: (type: DatabaseTypeString) => void
  handleEnvironmentToggle: (env: string) => void
  handleAddEnvironment: () => void
  handleRemoveEnvironment: (env: string) => void
  handleEditConnection: (connection: DatabaseConnection) => void
  handleCloseDialog: () => void
  resetForm: () => void
}

interface UseConnectionFormOptions {
  availableEnvironments: string[]
}

/**
 * Hook for managing connection form state and operations
 */
export function useConnectionForm({ availableEnvironments }: UseConnectionFormOptions): UseConnectionFormReturn {
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingConnectionId, setEditingConnectionId] = useState<string | null>(null)
  const [formData, setFormData] = useState<ConnectionFormData>(createDefaultFormData())
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [isTestingConnection, setIsTestingConnection] = useState(false)
  const [isSshSectionOpen, setIsSshSectionOpen] = useState(false)
  const [isVpcSectionOpen, setIsVpcSectionOpen] = useState(false)
  const [isAdvancedSshOpen, setIsAdvancedSshOpen] = useState(false)
  const [newEnvironment, setNewEnvironment] = useState('')

  // Compute available environment options
  const environmentOptions = useMemo(() => {
    const envSet = new Set<string>(availableEnvironments)
    formData.environments.forEach((env) => envSet.add(env))
    return Array.from(envSet).sort((a, b) => a.localeCompare(b))
  }, [availableEnvironments, formData.environments])

  // Handle database type change with default port update
  const handleTypeChange = useCallback((type: DatabaseTypeString) => {
    setFormData(prev => ({
      ...prev,
      type,
      port: getDefaultPort(type)
    }))
  }, [])

  // Toggle environment selection
  const handleEnvironmentToggle = useCallback((env: string) => {
    setFormData(prev => ({
      ...prev,
      environments: prev.environments.includes(env)
        ? prev.environments.filter((e) => e !== env)
        : [...prev.environments, env],
    }))
  }, [])

  // Add new environment
  const handleAddEnvironment = useCallback(() => {
    const trimmed = newEnvironment.trim()
    if (!trimmed) return

    setFormData(prev => ({
      ...prev,
      environments: prev.environments.includes(trimmed)
        ? prev.environments
        : [...prev.environments, trimmed],
    }))
    setNewEnvironment('')
  }, [newEnvironment])

  // Remove environment from selection
  const handleRemoveEnvironment = useCallback((env: string) => {
    setFormData(prev => ({
      ...prev,
      environments: prev.environments.filter((e) => e !== env),
    }))
  }, [])

  // Populate form for editing existing connection
  const handleEditConnection = useCallback((connection: DatabaseConnection) => {
    setEditingConnectionId(connection.id)
    setFormData(populateFormFromConnection(connection))

    // Open sections if they have data
    if (connection.useTunnel) {
      setIsSshSectionOpen(true)
    }
    if (connection.useVpc) {
      setIsVpcSectionOpen(true)
    }

    setIsDialogOpen(true)
  }, [])

  // Reset form to initial state
  const resetForm = useCallback(() => {
    setFormData(createDefaultFormData())
    setSubmitError(null)
    setIsSshSectionOpen(false)
    setIsVpcSectionOpen(false)
    setIsAdvancedSshOpen(false)
    setNewEnvironment('')
  }, [])

  // Close dialog and reset state
  const handleCloseDialog = useCallback(() => {
    setIsDialogOpen(false)
    setEditingConnectionId(null)
    resetForm()
  }, [resetForm])

  return {
    formData,
    setFormData,
    editingConnectionId,
    setEditingConnectionId,
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
    resetForm,
  }
}
