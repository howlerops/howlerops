import { Checkbox } from "@/components/ui/checkbox"
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

interface DatabaseSpecificFieldsProps {
  formData: ConnectionFormData
  onFormDataChange: (data: Partial<ConnectionFormData>) => void
}

/**
 * Database-specific configuration fields (MongoDB, Elasticsearch, ClickHouse)
 */
export function DatabaseSpecificFields({
  formData,
  onFormDataChange,
}: DatabaseSpecificFieldsProps) {
  // MongoDB-specific fields
  if (formData.type === 'mongodb') {
    return (
      <>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor="mongoConnectionString" className="text-right">
            Connection String (optional)
          </Label>
          <Input
            id="mongoConnectionString"
            value={formData.mongoConnectionString}
            onChange={(e) => onFormDataChange({ mongoConnectionString: e.target.value })}
            className="col-span-3"
            placeholder="mongodb://..."
          />
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor="mongoAuthDatabase" className="text-right">
            Auth Database
          </Label>
          <Input
            id="mongoAuthDatabase"
            value={formData.mongoAuthDatabase}
            onChange={(e) => onFormDataChange({ mongoAuthDatabase: e.target.value })}
            className="col-span-3"
            placeholder="admin"
          />
        </div>
      </>
    )
  }

  // Elasticsearch/OpenSearch-specific fields
  if (formData.type === 'elasticsearch' || formData.type === 'opensearch') {
    return (
      <>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor="elasticScheme" className="text-right">
            Scheme
          </Label>
          <Select value={formData.elasticScheme} onValueChange={(value) => onFormDataChange({ elasticScheme: value })}>
            <SelectTrigger className="col-span-3">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="http">HTTP</SelectItem>
              <SelectItem value="https">HTTPS</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor="elasticApiKey" className="text-right">
            API Key (optional)
          </Label>
          <Input
            id="elasticApiKey"
            type="password"
            value={formData.elasticApiKey}
            onChange={(e) => onFormDataChange({ elasticApiKey: e.target.value })}
            className="col-span-3"
          />
        </div>
      </>
    )
  }

  // ClickHouse-specific fields
  if (formData.type === 'clickhouse') {
    return (
      <div className="grid grid-cols-4 items-center gap-4">
        <Label htmlFor="clickhouseNativeProtocol" className="text-right">
          Native Protocol
        </Label>
        <div className="col-span-3 flex items-center space-x-2">
          <Checkbox
            id="clickhouseNativeProtocol"
            checked={formData.clickhouseNativeProtocol}
            onCheckedChange={(checked) => onFormDataChange({ clickhouseNativeProtocol: checked === true })}
          />
          <Label htmlFor="clickhouseNativeProtocol" className="text-sm text-muted-foreground">
            Use native protocol (port 9000) instead of HTTP
          </Label>
        </div>
      </div>
    )
  }

  return null
}
