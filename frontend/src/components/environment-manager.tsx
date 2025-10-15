import { useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import { useConnectionStore } from "@/store/connection-store"
import { Plus, X, Tag } from "lucide-react"
import { cn } from "@/lib/utils"

interface EnvironmentManagerProps {
  open: boolean
  onClose: () => void
  connectionId?: string // If provided, manage only this connection
}

export function EnvironmentManager({ open, onClose, connectionId }: EnvironmentManagerProps) {
  const {
    connections,
    availableEnvironments,
    addEnvironmentToConnection,
    removeEnvironmentFromConnection,
    refreshAvailableEnvironments,
  } = useConnectionStore()

  const [newEnvName, setNewEnvName] = useState("")
  
  // Filter connections if specific connectionId provided
  const targetConnections = connectionId
    ? connections.filter((c) => c.id === connectionId)
    : connections

  const handleCreateEnvironment = () => {
    if (!newEnvName.trim()) return
    
    const envName = newEnvName.trim()
    
    // Add to all selected connections or first connection
    if (targetConnections.length > 0) {
      targetConnections.forEach((conn) => {
        if (!conn.environments?.includes(envName)) {
          addEnvironmentToConnection(conn.id, envName)
        }
      })
    }
    
    setNewEnvName("")
    refreshAvailableEnvironments()
  }

  const handleToggleEnvironment = (connId: string, env: string) => {
    const connection = connections.find((c) => c.id === connId)
    if (!connection) return

    if (connection.environments?.includes(env)) {
      removeEnvironmentFromConnection(connId, env)
    } else {
      addEnvironmentToConnection(connId, env)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tag className="h-5 w-5" />
            Manage Environments
          </DialogTitle>
          <DialogDescription>
            Create and assign environment tags to organize your database connections.
          </DialogDescription>
        </DialogHeader>

        {/* Create New Environment */}
        <div className="space-y-2">
          <Label>Create New Environment</Label>
          <div className="flex gap-2">
            <Input
              placeholder="e.g., production, staging, local"
              value={newEnvName}
              onChange={(e) => setNewEnvName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault()
                  handleCreateEnvironment()
                }
              }}
            />
            <Button
              onClick={handleCreateEnvironment}
              disabled={!newEnvName.trim()}
              size="sm"
            >
              <Plus className="h-4 w-4 mr-1" />
              Create
            </Button>
          </div>
        </div>

        {/* Available Environments */}
        {availableEnvironments.length > 0 && (
          <div className="space-y-2">
            <Label>Available Environments</Label>
            <div className="flex flex-wrap gap-2">
              {availableEnvironments.map((env) => (
                <Badge key={env} variant="secondary" className="text-sm">
                  {env}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Connection Matrix */}
        <div className="space-y-3 mt-4">
          <Label>Assign to Connections</Label>
          {targetConnections.length === 0 ? (
            <div className="text-sm text-muted-foreground text-center py-8">
              No connections available
            </div>
          ) : (
            <div className="space-y-2">
              {targetConnections.map((connection) => (
                <div
                  key={connection.id}
                  className="border rounded-lg p-3 space-y-2"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm">{connection.name}</span>
                      <Badge variant="outline" className="text-xs">
                        {connection.type}
                      </Badge>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {connection.environments?.length || 0} environments
                    </span>
                  </div>

                  {/* Environment chips for this connection */}
                  <div className="flex flex-wrap gap-2">
                    {availableEnvironments.length === 0 ? (
                      <span className="text-xs text-muted-foreground">
                        Create an environment to get started
                      </span>
                    ) : (
                      availableEnvironments.map((env) => {
                        const isAssigned = connection.environments?.includes(env)
                        return (
                          <button
                            key={env}
                            onClick={() => handleToggleEnvironment(connection.id, env)}
                            className={cn(
                              "inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium transition-colors",
                              isAssigned
                                ? "bg-primary text-primary-foreground hover:bg-primary/90"
                                : "bg-muted text-muted-foreground hover:bg-muted/70"
                            )}
                          >
                            {env}
                            {isAssigned && <X className="h-3 w-3" />}
                          </button>
                        )
                      })
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button onClick={onClose} variant="outline">
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

