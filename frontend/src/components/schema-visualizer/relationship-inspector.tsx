import React, { useEffect, useCallback } from 'react'
import { X, Copy, CheckCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EdgeConfig, TableConfig } from '@/types/schema-visualizer'

interface RelationshipInspectorProps {
  edge: EdgeConfig
  sourceTable: TableConfig
  targetTable: TableConfig
  position: { x: number; y: number }
  onClose: () => void
}

export function RelationshipInspector({
  edge,
  sourceTable,
  targetTable,
  position,
  onClose,
}: RelationshipInspectorProps) {
  const [copied, setCopied] = React.useState(false)

  // Close on Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Get relationship type display
  const getRelationshipType = () => {
    switch (edge.relation) {
      case 'hasOne':
        return { label: 'One-to-One', badge: '1:1' }
      case 'hasMany':
        return { label: 'One-to-Many', badge: '1:N' }
      case 'belongsTo':
        return { label: 'Many-to-One', badge: 'N:1' }
      default:
        return { label: 'Unknown', badge: '?' }
    }
  }

  // Find column details
  const sourceColumn = sourceTable.columns.find((c) => c.name === edge.sourceKey)
  const targetColumn = targetTable.columns.find((c) => c.name === edge.targetKey)

  // Generate SQL for the foreign key constraint
  const generateSQL = useCallback(() => {
    const constraintName = edge.label || `fk_${sourceTable.name}_${edge.sourceKey}`

    return `ALTER TABLE ${sourceTable.schema}.${sourceTable.name}
  ADD CONSTRAINT ${constraintName}
  FOREIGN KEY (${edge.sourceKey})
  REFERENCES ${targetTable.schema}.${targetTable.name}(${edge.targetKey})
  ON DELETE CASCADE
  ON UPDATE RESTRICT;`
  }, [edge, sourceTable, targetTable])

  // Copy SQL to clipboard
  const handleCopySQL = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(generateSQL())
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy SQL:', error)
    }
  }, [generateSQL])

  const relationshipType = getRelationshipType()

  // Calculate position to keep panel in viewport
  const panelStyle: React.CSSProperties = {
    position: 'fixed',
    left: Math.min(position.x, window.innerWidth - 420),
    top: Math.min(position.y, window.innerHeight - 400),
    zIndex: 1000,
    maxWidth: '400px',
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-background/20 backdrop-blur-[2px] z-[999]"
        onClick={onClose}
      />

      {/* Inspector Panel */}
      <Card style={panelStyle} className="shadow-2xl border-2">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Relationship Details</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
              className="h-6 w-6 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Relationship Type */}
          <div className="space-y-1">
            <div className="text-xs font-medium text-muted-foreground">
              Type
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold">
                {relationshipType.label}
              </span>
              <Badge variant="secondary" className="text-xs">
                {relationshipType.badge}
              </Badge>
            </div>
          </div>

          {/* Source Column */}
          <div className="space-y-1">
            <div className="text-xs font-medium text-muted-foreground">
              Source
            </div>
            <div className="bg-muted/50 rounded-md p-2 text-sm">
              <div className="font-mono">
                {sourceTable.schema}.{sourceTable.name}.{edge.sourceKey}
              </div>
              {sourceColumn && (
                <div className="text-xs text-muted-foreground mt-1">
                  {sourceColumn.type}
                  {sourceColumn.isPrimaryKey && (
                    <Badge variant="outline" className="ml-2 text-[10px] h-4">
                      PK
                    </Badge>
                  )}
                  {sourceColumn.isForeignKey && (
                    <Badge variant="outline" className="ml-2 text-[10px] h-4">
                      FK
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Target Column */}
          <div className="space-y-1">
            <div className="text-xs font-medium text-muted-foreground">
              Target
            </div>
            <div className="bg-muted/50 rounded-md p-2 text-sm">
              <div className="font-mono">
                {targetTable.schema}.{targetTable.name}.{edge.targetKey}
              </div>
              {targetColumn && (
                <div className="text-xs text-muted-foreground mt-1">
                  {targetColumn.type}
                  {targetColumn.isPrimaryKey && (
                    <Badge variant="outline" className="ml-2 text-[10px] h-4">
                      PK
                    </Badge>
                  )}
                  {targetColumn.isForeignKey && (
                    <Badge variant="outline" className="ml-2 text-[10px] h-4">
                      FK
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Constraint Name */}
          {edge.label && (
            <div className="space-y-1">
              <div className="text-xs font-medium text-muted-foreground">
                Constraint Name
              </div>
              <div className="text-sm font-mono bg-muted/50 rounded-md p-2">
                {edge.label}
              </div>
            </div>
          )}

          {/* Referential Actions */}
          <div className="space-y-1">
            <div className="text-xs font-medium text-muted-foreground">
              Referential Actions
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div className="bg-muted/50 rounded-md p-2">
                <div className="text-muted-foreground mb-1">ON DELETE</div>
                <div className="font-medium">CASCADE</div>
              </div>
              <div className="bg-muted/50 rounded-md p-2">
                <div className="text-muted-foreground mb-1">ON UPDATE</div>
                <div className="font-medium">RESTRICT</div>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <Button
              variant="default"
              size="sm"
              className="flex-1"
              onClick={handleCopySQL}
            >
              {copied ? (
                <>
                  <CheckCheck className="h-4 w-4 mr-2" />
                  Copied!
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4 mr-2" />
                  Copy SQL
                </>
              )}
            </Button>
            <Button variant="outline" size="sm" onClick={onClose}>
              Close
            </Button>
          </div>
        </CardContent>
      </Card>
    </>
  )
}
