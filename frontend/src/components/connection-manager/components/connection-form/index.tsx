import { Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

import type { ConnectionFormData, DatabaseTypeString } from "../../types"
import { requiresHostPort } from "../../utils"
import { BasicFields } from "./basic-fields"
import { DatabaseSpecificFields } from "./database-specific-fields"
import { SSHTunnelSection } from "./ssh-tunnel-section"
import { VPCSection } from "./vpc-section"

interface ConnectionFormProps {
  formData: ConnectionFormData
  editingConnectionId: string | null
  environmentOptions: string[]
  newEnvironment: string
  submitError: string | null
  isTestingConnection: boolean
  isSshSectionOpen: boolean
  isVpcSectionOpen: boolean
  isAdvancedSshOpen: boolean
  onSubmit: (e: React.FormEvent) => void
  onFormDataChange: (data: Partial<ConnectionFormData>) => void
  onTypeChange: (type: DatabaseTypeString) => void
  onEnvironmentToggle: (env: string) => void
  onAddEnvironment: () => void
  onRemoveEnvironment: (env: string) => void
  onNewEnvironmentChange: (value: string) => void
  onSshSectionOpenChange: (open: boolean) => void
  onVpcSectionOpenChange: (open: boolean) => void
  onAdvancedSshOpenChange: (open: boolean) => void
}

/**
 * Connection form dialog content
 */
export function ConnectionForm({
  formData,
  editingConnectionId,
  environmentOptions,
  newEnvironment,
  submitError,
  isTestingConnection,
  isSshSectionOpen,
  isVpcSectionOpen,
  isAdvancedSshOpen,
  onSubmit,
  onFormDataChange,
  onTypeChange,
  onEnvironmentToggle,
  onAddEnvironment,
  onRemoveEnvironment,
  onNewEnvironmentChange,
  onSshSectionOpenChange,
  onVpcSectionOpenChange,
  onAdvancedSshOpenChange,
}: ConnectionFormProps) {
  return (
    <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
      <form onSubmit={onSubmit}>
        <DialogHeader>
          <DialogTitle>{editingConnectionId ? 'Edit Connection' : 'Add New Connection'}</DialogTitle>
          <DialogDescription>
            {editingConnectionId ? 'Update the details for your database connection.' : 'Enter the details for your database connection.'}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* Basic Connection Info */}
          <BasicFields
            formData={formData}
            environmentOptions={environmentOptions}
            newEnvironment={newEnvironment}
            onFormDataChange={onFormDataChange}
            onTypeChange={onTypeChange}
            onEnvironmentToggle={onEnvironmentToggle}
            onAddEnvironment={onAddEnvironment}
            onRemoveEnvironment={onRemoveEnvironment}
            onNewEnvironmentChange={onNewEnvironmentChange}
          />

          {/* Database-specific fields */}
          <DatabaseSpecificFields
            formData={formData}
            onFormDataChange={onFormDataChange}
          />

          {/* SSH Tunnel Configuration */}
          {requiresHostPort(formData.type) && (
            <SSHTunnelSection
              formData={formData}
              isOpen={isSshSectionOpen}
              isAdvancedOpen={isAdvancedSshOpen}
              onOpenChange={onSshSectionOpenChange}
              onAdvancedOpenChange={onAdvancedSshOpenChange}
              onFormDataChange={onFormDataChange}
            />
          )}

          {/* VPC Configuration */}
          {requiresHostPort(formData.type) && (
            <VPCSection
              formData={formData}
              isOpen={isVpcSectionOpen}
              onOpenChange={onVpcSectionOpenChange}
              onFormDataChange={onFormDataChange}
            />
          )}
        </div>

        <DialogFooter>
          <div className="flex flex-col items-start gap-2 w-full">
            {submitError && (
              <p className="text-sm text-destructive">{submitError}</p>
            )}
            <Button type="submit" disabled={isTestingConnection}>
              {isTestingConnection ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Testing...
                </>
              ) : (
                editingConnectionId ? 'Update Connection' : 'Add Connection'
              )}
            </Button>
          </div>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}

export { BasicFields } from "./basic-fields"
export { DatabaseSpecificFields } from "./database-specific-fields"
export { SSHTunnelSection } from "./ssh-tunnel-section"
export { VPCSection } from "./vpc-section"
