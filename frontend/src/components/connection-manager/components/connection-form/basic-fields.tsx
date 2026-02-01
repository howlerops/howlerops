import { Plus, X } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { SecretInput } from "@/components/secret-input"
import type { ConnectionFormData, DatabaseTypeString } from "../../types"
import { DATABASE_TYPE_OPTIONS } from "../../types"
import { requiresHostPort, supportsSSL, getDatabaseLabel, isDatabaseRequired, isUsernameRequired } from "../../utils"

interface BasicFieldsProps {
  formData: ConnectionFormData
  environmentOptions: string[]
  newEnvironment: string
  onFormDataChange: (data: Partial<ConnectionFormData>) => void
  onTypeChange: (type: DatabaseTypeString) => void
  onEnvironmentToggle: (env: string) => void
  onAddEnvironment: () => void
  onRemoveEnvironment: (env: string) => void
  onNewEnvironmentChange: (value: string) => void
}

/**
 * Basic connection form fields (name, type, host, port, credentials, environments)
 */
export function BasicFields({
  formData,
  environmentOptions,
  newEnvironment,
  onFormDataChange,
  onTypeChange,
  onEnvironmentToggle,
  onAddEnvironment,
  onRemoveEnvironment,
  onNewEnvironmentChange,
}: BasicFieldsProps) {
  return (
    <>
      {/* Connection Name */}
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="name" className="text-right">
          Name
        </Label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => onFormDataChange({ name: e.target.value })}
          className="col-span-3"
          required
        />
      </div>

      {/* Database Type */}
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="type" className="text-right">
          Type
        </Label>
        <Select value={formData.type} onValueChange={onTypeChange}>
          <SelectTrigger className="col-span-3">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {DATABASE_TYPE_OPTIONS.map(option => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Environments */}
      <div className="grid grid-cols-4 items-start gap-4">
        <Label className="text-right pt-2">
          Environments
        </Label>
        <div className="col-span-3 space-y-2">
          {environmentOptions.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {environmentOptions.map((env) => {
                const isSelected = formData.environments.includes(env)
                return (
                  <Button
                    key={env}
                    type="button"
                    variant={isSelected ? "default" : "outline"}
                    size="sm"
                    onClick={() => onEnvironmentToggle(env)}
                    className={isSelected ? undefined : "text-muted-foreground"}
                  >
                    {env}
                  </Button>
                )
              })}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No environments yet. Create one below.
            </p>
          )}

          <div className="flex gap-2">
            <Input
              id="new-environment"
              placeholder="e.g., production, staging"
              value={newEnvironment}
              onChange={(e) => onNewEnvironmentChange(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  onAddEnvironment()
                }
              }}
            />
            <Button
              type="button"
              onClick={onAddEnvironment}
              disabled={!newEnvironment.trim()}
              variant="secondary"
            >
              <Plus className="h-4 w-4 mr-1" />
              Add
            </Button>
          </div>

          {formData.environments.length > 0 && (
            <div className="flex flex-wrap gap-2 pt-1">
              {formData.environments.map((env) => (
                <Badge key={env} variant="secondary" className="flex items-center gap-1">
                  {env}
                  <button
                    type="button"
                    onClick={() => onRemoveEnvironment(env)}
                    className="rounded-full p-0.5 hover:bg-muted"
                    aria-label={`Remove ${env}`}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Host and Port */}
      {requiresHostPort(formData.type) && (
        <>
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="host" className="text-right">
              Host
            </Label>
            <Input
              id="host"
              value={formData.host}
              onChange={(e) => onFormDataChange({ host: e.target.value })}
              className="col-span-3"
              required
            />
          </div>

          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="port" className="text-right">
              Port
            </Label>
            <Input
              id="port"
              type="number"
              value={formData.port}
              onChange={(e) => onFormDataChange({ port: e.target.value })}
              className="col-span-3"
              required
            />
          </div>
        </>
      )}

      {/* Database */}
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="database" className="text-right">
          {getDatabaseLabel(formData.type)}
        </Label>
        <Input
          id="database"
          value={formData.database}
          onChange={(e) => onFormDataChange({ database: e.target.value })}
          className="col-span-3"
          required={isDatabaseRequired(formData.type)}
        />
      </div>

      {/* Username and Password */}
      {requiresHostPort(formData.type) && (
        <>
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="username" className="text-right">
              Username
            </Label>
            <Input
              id="username"
              value={formData.username}
              onChange={(e) => onFormDataChange({ username: e.target.value })}
              className="col-span-3"
              required={isUsernameRequired(formData.type)}
            />
          </div>

          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="password" className="text-right">
              Password
            </Label>
            <div className="col-span-3">
              <SecretInput
                id="password"
                value={formData.password}
                onChange={(value) => onFormDataChange({ password: value })}
                placeholder="Enter database password"
              />
            </div>
          </div>
        </>
      )}

      {/* SSL Mode */}
      {supportsSSL(formData.type) && (
        <div className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor="sslMode" className="text-right">
            SSL Mode
          </Label>
          <Select value={formData.sslMode} onValueChange={(value) => onFormDataChange({ sslMode: value })}>
            <SelectTrigger className="col-span-3">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="disable">Disable</SelectItem>
              <SelectItem value="allow">Allow</SelectItem>
              <SelectItem value="prefer">Prefer</SelectItem>
              <SelectItem value="require">Require</SelectItem>
              <SelectItem value="verify-ca">Verify CA</SelectItem>
              <SelectItem value="verify-full">Verify Full</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}
    </>
  )
}
