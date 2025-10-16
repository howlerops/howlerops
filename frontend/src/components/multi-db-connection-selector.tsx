/**
 * Multi-Database Connection Selector
 * 
 * Allows users to select which connections to use in multi-DB mode
 * Defaults to all connections but allows selective override
 */

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { Network, CheckCircle2, Circle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useConnectionStore, type DatabaseConnection } from '@/store/connection-store'

export interface MultiDBConnectionSelectorProps {
  open: boolean
  onClose: () => void
  selectedConnectionIds: string[]
  onSelectionChange: (connectionIds: string[]) => void
  filteredConnections?: DatabaseConnection[]
}

export function MultiDBConnectionSelector({
  open,
  onClose,
  selectedConnectionIds,
  onSelectionChange,
  filteredConnections
}: MultiDBConnectionSelectorProps) {
  const { getFilteredConnections } = useConnectionStore()
  const connections = filteredConnections || getFilteredConnections()
  
  const [localSelection, setLocalSelection] = useState<string[]>(selectedConnectionIds)

  // Update local selection when prop changes (controlled component pattern)
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLocalSelection(selectedConnectionIds)
  }, [selectedConnectionIds, open])

  const selectedCount = localSelection.length
  const totalCount = connections.length

  const handleToggleConnection = (connectionId: string) => {
    setLocalSelection(prev => {
      if (prev.includes(connectionId)) {
        // Don't allow deselecting the last connection
        if (prev.length === 1) {
          return prev
        }
        return prev.filter(id => id !== connectionId)
      } else {
        return [...prev, connectionId]
      }
    })
  }

  const handleSelectAll = () => {
    setLocalSelection(connections.map(c => c.id))
  }

  const handleDeselectAll = () => {
    // Keep at least one selected
    if (connections.length > 0) {
      setLocalSelection([connections[0].id])
    }
  }

  const handleApply = () => {
    onSelectionChange(localSelection)
    onClose()
  }

  const handleCancel = () => {
    setLocalSelection(selectedConnectionIds)
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleCancel()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Network className="h-5 w-5 text-accent-foreground" />
            Select Connections
          </DialogTitle>
          <DialogDescription>
            Choose which databases to include in your multi-database queries.
            At least one connection must be selected.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Summary */}
          <div className="flex items-center justify-between p-3 bg-accent/10 border border-accent rounded-lg">
            <div className="flex items-center gap-2">
              <Network className="h-4 w-4 text-accent-foreground" />
              <span className="text-sm font-medium">
                {selectedCount} of {totalCount} databases selected
              </span>
            </div>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleSelectAll}
                disabled={selectedCount === totalCount}
              >
                Select All
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDeselectAll}
                disabled={selectedCount === 1}
              >
                Clear
              </Button>
            </div>
          </div>

          {/* Connection List */}
          <div className="space-y-2 max-h-[400px] overflow-y-auto">
            {connections.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground text-sm">
                No connections available
              </div>
            ) : (
              connections.map((connection) => {
                const isSelected = localSelection.includes(connection.id)
                const isOnlySelected = isSelected && selectedCount === 1
                
                return (
                  <div
                    key={connection.id}
                    className={cn(
                      'flex items-center gap-3 p-3 border rounded-lg transition-colors cursor-pointer',
                      isSelected
                        ? 'bg-accent/10 border-accent'
                        : 'bg-background hover:bg-muted/50 border-border',
                      isOnlySelected && 'cursor-not-allowed opacity-75'
                    )}
                    onClick={() => !isOnlySelected && handleToggleConnection(connection.id)}
                  >
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => !isOnlySelected && handleToggleConnection(connection.id)}
                      disabled={isOnlySelected}
                      className={cn(
                        isSelected && 'border-accent data-[state=checked]:bg-accent'
                      )}
                    />

                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{connection.name}</span>
                        <Badge variant="outline" className="text-xs">
                          {connection.type}
                        </Badge>
                        {connection.isConnected ? (
                          <div className="flex items-center gap-1 text-primary">
                            <CheckCircle2 className="h-3 w-3" />
                            <span className="text-xs">Connected</span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1 text-muted-foreground">
                            <Circle className="h-3 w-3" />
                            <span className="text-xs">Disconnected</span>
                          </div>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground truncate">
                        {connection.database}
                      </div>
                    </div>
                  </div>
                )
              })
            )}
          </div>

          {/* Warning */}
          {selectedCount === 1 && (
            <div className="p-2 bg-accent/10 border border-accent rounded text-xs text-accent-foreground">
              At least one connection must remain selected for multi-DB queries.
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-4 border-t">
          <Button variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button onClick={handleApply}>
            Apply Selection
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

