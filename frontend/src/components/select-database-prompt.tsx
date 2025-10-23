import React, { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertCircle, Database, Check } from 'lucide-react'
import { DatabaseConnection } from '@/store/connection-store'
import { cn } from '@/lib/utils'

interface SelectDatabasePromptProps {
  isOpen: boolean
  onClose: () => void
  onSelect: (connectionId: string) => void
  connections: DatabaseConnection[]
  currentConnectionId?: string | null
}

export function SelectDatabasePrompt({
  isOpen,
  onClose,
  onSelect,
  connections,
  currentConnectionId
}: SelectDatabasePromptProps) {
  const [selectedConnectionId, setSelectedConnectionId] = useState<string>(
    currentConnectionId || ''
  )

  // Filter to only show connected databases
  const connectedDatabases = connections.filter(conn => conn.isConnected)

  const handleConfirm = () => {
    if (selectedConnectionId) {
      onSelect(selectedConnectionId)
      onClose()
    }
  }

  const handleCancel = () => {
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Select Database
          </DialogTitle>
          <DialogDescription>
            Please select a database connection to execute this query.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {connectedDatabases.length === 0 ? (
            <div className="flex items-start gap-3 rounded-lg border border-destructive bg-destructive/10 p-4">
              <AlertCircle className="h-5 w-5 text-destructive mt-0.5" />
              <div className="flex-1">
                <p className="font-medium text-sm text-destructive">
                  No Connected Databases
                </p>
                <p className="text-sm text-muted-foreground mt-1">
                  Please connect to a database before executing queries.
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {connectedDatabases.map((connection) => (
                <button
                  key={connection.id}
                  onClick={() => setSelectedConnectionId(connection.id)}
                  className={cn(
                    "w-full flex items-start gap-3 rounded-lg border p-3 transition-colors text-left",
                    selectedConnectionId === connection.id
                      ? "border-primary bg-primary/10"
                      : "border-border bg-muted/30 hover:bg-muted/50"
                  )}
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <Database className="h-4 w-4 text-blue-500" />
                      <span className="font-medium text-sm">
                        {connection.name}
                      </span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-1">
                      {connection.database} • {connection.host}
                      {connection.environments && connection.environments.length > 0 && (
                        <span className="ml-2 px-1.5 py-0.5 rounded bg-accent/20 text-accent-foreground">
                          {connection.environments.join(', ')}
                        </span>
                      )}
                    </div>
                  </div>
                  {selectedConnectionId === connection.id && (
                    <Check className="h-5 w-5 text-primary flex-shrink-0" />
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={!selectedConnectionId || connectedDatabases.length === 0}
          >
            Execute Query
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
