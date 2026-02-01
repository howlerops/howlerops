import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

import { ALL_ENVIRONMENTS_FILTER } from "../types"

interface EnvironmentFilterProps {
  availableEnvironments: string[]
  activeEnvironmentFilter: string | null
  groupByEnvironment: boolean
  onFilterChange: (filter: string | null) => void
  onGroupByChange: (grouped: boolean) => void
}

/**
 * Component for filtering and grouping connections by environment
 */
export function EnvironmentFilter({
  availableEnvironments,
  activeEnvironmentFilter,
  groupByEnvironment,
  onFilterChange,
  onGroupByChange,
}: EnvironmentFilterProps) {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div className="flex flex-col gap-1">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
          Environment Filter
        </span>
        <Select
          value={activeEnvironmentFilter ?? ALL_ENVIRONMENTS_FILTER}
          onValueChange={(value) => onFilterChange(value === ALL_ENVIRONMENTS_FILTER ? null : value)}
        >
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="All environments" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_ENVIRONMENTS_FILTER}>All environments</SelectItem>
            {availableEnvironments.map((env) => (
              <SelectItem key={env} value={env}>
                {env}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center gap-2">
        <Switch
          id="group-by-environment"
          checked={groupByEnvironment}
          onCheckedChange={(checked) => onGroupByChange(!!checked)}
        />
        <Label htmlFor="group-by-environment" className="text-sm">
          Group by environment
        </Label>
      </div>
    </div>
  )
}
