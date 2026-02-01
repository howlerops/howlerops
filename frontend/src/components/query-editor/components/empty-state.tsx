import { Database, Plus, Sparkles } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface EmptyStateProps {
  onCreateSqlTab: () => void
  onCreateAiTab: () => void
}

export function EmptyState({ onCreateSqlTab, onCreateAiTab }: EmptyStateProps) {
  return (
    <div className="flex-1 flex w-full items-center justify-center">
      <div className="text-center">
        <h3 className="text-lg font-medium mb-2">No query tabs open</h3>
        <p className="text-mute mb-4">Create a new tab to start writing SQL queries</p>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              New Query
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onClick={onCreateSqlTab}>
              <Database className="h-4 w-4 mr-2" />
              SQL Editor Tab
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onCreateAiTab}>
              <Sparkles className="h-4 w-4 mr-2" />
              AI Query Agent
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}
