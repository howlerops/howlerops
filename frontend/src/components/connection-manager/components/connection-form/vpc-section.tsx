import { ChevronDown, ChevronRight, Cloud } from "lucide-react"

import { Checkbox } from "@/components/ui/checkbox"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import type { ConnectionFormData } from "../../types"

interface VPCSectionProps {
  formData: ConnectionFormData
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  onFormDataChange: (data: Partial<ConnectionFormData>) => void
}

/**
 * VPC configuration section
 */
export function VPCSection({
  formData,
  isOpen,
  onOpenChange,
  onFormDataChange,
}: VPCSectionProps) {
  return (
    <Collapsible open={isOpen} onOpenChange={onOpenChange}>
      <div className="border rounded-lg p-4 mt-2">
        <CollapsibleTrigger className="flex items-center justify-between w-full">
          <div className="flex items-center space-x-2">
            <Cloud className="h-4 w-4" />
            <Label className="text-sm font-semibold">VPC Configuration</Label>
          </div>
          {isOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </CollapsibleTrigger>

        <CollapsibleContent className="mt-4 space-y-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="useVpc"
              checked={formData.useVpc}
              onCheckedChange={(checked) => onFormDataChange({ useVpc: checked === true })}
            />
            <Label htmlFor="useVpc">Enable VPC configuration</Label>
          </div>

          {formData.useVpc && (
            <>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="vpcId" className="text-right">
                  VPC ID
                </Label>
                <Input
                  id="vpcId"
                  value={formData.vpcId}
                  onChange={(e) => onFormDataChange({ vpcId: e.target.value })}
                  className="col-span-3"
                  placeholder="vpc-xxxxxxxxx"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="subnetId" className="text-right">
                  Subnet ID
                </Label>
                <Input
                  id="subnetId"
                  value={formData.subnetId}
                  onChange={(e) => onFormDataChange({ subnetId: e.target.value })}
                  className="col-span-3"
                  placeholder="subnet-xxxxxxxxx"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="securityGroupIds" className="text-right">
                  Security Group IDs
                </Label>
                <Input
                  id="securityGroupIds"
                  value={formData.securityGroupIds}
                  onChange={(e) => onFormDataChange({ securityGroupIds: e.target.value })}
                  className="col-span-3"
                  placeholder="sg-xxx, sg-yyy"
                  required
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="privateLinkService" className="text-right">
                  Private Link Service
                </Label>
                <Input
                  id="privateLinkService"
                  value={formData.privateLinkService}
                  onChange={(e) => onFormDataChange({ privateLinkService: e.target.value })}
                  className="col-span-3"
                />
              </div>

              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="endpointServiceName" className="text-right">
                  Endpoint Service Name
                </Label>
                <Input
                  id="endpointServiceName"
                  value={formData.endpointServiceName}
                  onChange={(e) => onFormDataChange({ endpointServiceName: e.target.value })}
                  className="col-span-3"
                />
              </div>
            </>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}
