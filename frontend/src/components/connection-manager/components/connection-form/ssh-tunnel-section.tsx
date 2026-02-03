import { ChevronDown, ChevronRight, Lock } from "lucide-react"

import { PemKeyUpload } from "@/components/pem-key-upload"
import { SecretInput } from "@/components/secret-input"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import type { ConnectionFormData } from "../../types"
import { SSHAuthMethod } from "../../types"

interface SSHTunnelSectionProps {
  formData: ConnectionFormData
  isOpen: boolean
  isAdvancedOpen: boolean
  onOpenChange: (open: boolean) => void
  onAdvancedOpenChange: (open: boolean) => void
  onFormDataChange: (data: Partial<ConnectionFormData>) => void
}

/**
 * SSH Tunnel configuration section
 */
export function SSHTunnelSection({
  formData,
  isOpen,
  isAdvancedOpen,
  onOpenChange,
  onAdvancedOpenChange,
  onFormDataChange,
}: SSHTunnelSectionProps) {
  return (
    <Collapsible open={isOpen} onOpenChange={onOpenChange}>
      <div className="border rounded-lg p-4 mt-2">
        <CollapsibleTrigger className="flex items-center justify-between w-full">
          <div className="flex items-center space-x-2">
            <Lock className="h-4 w-4" />
            <Label className="text-sm font-semibold">SSH Tunnel Configuration</Label>
          </div>
          {isOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </CollapsibleTrigger>

        <CollapsibleContent className="mt-4 space-y-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="useTunnel"
              checked={formData.useTunnel}
              onCheckedChange={(checked) => onFormDataChange({ useTunnel: checked === true })}
            />
            <Label htmlFor="useTunnel">Enable SSH tunnel</Label>
          </div>

          {formData.useTunnel && (
            <>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="sshHost" className="text-right">
                  SSH Host
                </Label>
                <Input
                  id="sshHost"
                  value={formData.sshHost}
                  onChange={(e) => onFormDataChange({ sshHost: e.target.value })}
                  className="col-span-3"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="sshPort" className="text-right">
                  SSH Port
                </Label>
                <Input
                  id="sshPort"
                  type="number"
                  value={formData.sshPort}
                  onChange={(e) => onFormDataChange({ sshPort: e.target.value })}
                  className="col-span-3"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="sshUser" className="text-right">
                  SSH User
                </Label>
                <Input
                  id="sshUser"
                  value={formData.sshUser}
                  onChange={(e) => onFormDataChange({ sshUser: e.target.value })}
                  className="col-span-3"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="sshAuthMethod" className="text-right">
                  Auth Method
                </Label>
                <Select
                  value={formData.sshAuthMethod}
                  onValueChange={(value) => onFormDataChange({ sshAuthMethod: value as SSHAuthMethod })}
                >
                  <SelectTrigger className="col-span-3">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD}>Password</SelectItem>
                    <SelectItem value={SSHAuthMethod.SSH_AUTH_METHOD_PRIVATE_KEY}>Private Key</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD && (
                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="sshPassword" className="text-right">
                    SSH Password
                  </Label>
                  <div className="col-span-3">
                    <SecretInput
                      id="sshPassword"
                      value={formData.sshPassword}
                      onChange={(value) => onFormDataChange({ sshPassword: value })}
                      placeholder="Enter SSH password"
                      required
                    />
                  </div>
                </div>
              )}

              {formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PRIVATE_KEY && (
                <div className="grid grid-cols-4 items-start gap-4">
                  <Label className="text-right pt-2">
                    Private Key
                  </Label>
                  <div className="col-span-3">
                    <PemKeyUpload
                      onUpload={(keyContent) => onFormDataChange({ sshPrivateKey: keyContent })}
                      onError={(error) => console.error('PEM key error:', error)}
                    />
                    <div className="mt-2">
                      <SecretInput
                        value={formData.sshPrivateKeyPassphrase}
                        onChange={(value) => onFormDataChange({ sshPrivateKeyPassphrase: value })}
                        placeholder="Key passphrase (if encrypted)"
                        label="Key Passphrase (Optional)"
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* Advanced SSH Options */}
              <Collapsible open={isAdvancedOpen} onOpenChange={onAdvancedOpenChange}>
                <CollapsibleTrigger className="flex items-center space-x-2 text-sm text-muted-foreground hover:text-foreground">
                  {isAdvancedOpen ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                  <span>Advanced SSH Options</span>
                </CollapsibleTrigger>

                <CollapsibleContent className="mt-4 space-y-4">
                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="sshKnownHostsPath" className="text-right">
                      Known Hosts Path
                    </Label>
                    <Input
                      id="sshKnownHostsPath"
                      value={formData.sshKnownHostsPath}
                      onChange={(e) => onFormDataChange({ sshKnownHostsPath: e.target.value })}
                      className="col-span-3"
                      placeholder="~/.ssh/known_hosts"
                    />
                  </div>

                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="sshStrictHostKeyChecking" className="text-right">
                      Strict Host Key Checking
                    </Label>
                    <div className="col-span-3 flex items-center space-x-2">
                      <Checkbox
                        id="sshStrictHostKeyChecking"
                        checked={formData.sshStrictHostKeyChecking}
                        onCheckedChange={(checked) => onFormDataChange({ sshStrictHostKeyChecking: checked === true })}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="sshTimeoutSeconds" className="text-right">
                      Timeout (seconds)
                    </Label>
                    <Input
                      id="sshTimeoutSeconds"
                      type="number"
                      value={formData.sshTimeoutSeconds}
                      onChange={(e) => onFormDataChange({ sshTimeoutSeconds: e.target.value })}
                      className="col-span-3"
                    />
                  </div>

                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="sshKeepAliveIntervalSeconds" className="text-right">
                      Keep-Alive (seconds)
                    </Label>
                    <Input
                      id="sshKeepAliveIntervalSeconds"
                      type="number"
                      value={formData.sshKeepAliveIntervalSeconds}
                      onChange={(e) => onFormDataChange({ sshKeepAliveIntervalSeconds: e.target.value })}
                      className="col-span-3"
                    />
                  </div>
                </CollapsibleContent>
              </Collapsible>
            </>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}
