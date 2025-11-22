import { ChevronLeft, ChevronRight, Copy, Database, Loader2 } from 'lucide-react'
import React, { useEffect, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import { useToast } from '@/hooks/use-toast'
import { QueryEditableMetadata } from '@/store/query-store'
import { CellValue, TableRow } from '@/types/table'

interface JsonRowViewerSidebarV2Props {
  open: boolean
  onClose: () => void
  rowData: TableRow | null
  rowId: string | null
  rowIndex: number
  totalRows: number
  onNavigate: (direction: 'prev' | 'next') => void
  columns?: string[]
  metadata?: QueryEditableMetadata | null
  connectionId?: string
  onSave?: (rowId: string, data: Record<string, CellValue>) => Promise<boolean>
}

interface FieldRowProps {
  fieldKey: string
  value: CellValue
  editable: boolean
  onUpdate: (key: string, value: CellValue) => void
  isPrimaryKey?: boolean
  isForeignKey?: boolean
}

interface RowNavigationProps {
  currentIndex: number
  totalRows: number
  onPrevious: () => void
  onNext: () => void
  hasPrevious: boolean
  hasNext: boolean
}

function RowNavigation({
  currentIndex,
  totalRows,
  onPrevious,
  onNext,
  hasPrevious,
  hasNext
}: RowNavigationProps) {
  return (
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        onClick={onPrevious}
        disabled={!hasPrevious}
        className="h-8 w-8 p-0"
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>
      <span className="text-sm text-muted-foreground">
        Row {currentIndex + 1} of {totalRows}
      </span>
      <Button
        variant="outline"
        size="sm"
        onClick={onNext}
        disabled={!hasNext}
        className="h-8 w-8 p-0"
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  )
}

function FieldRow({
  fieldKey,
  value,
  editable,
  onUpdate,
  isPrimaryKey,
  isForeignKey
}: FieldRowProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState<string>(String(value ?? ''))

  const handleSave = () => {
    let parsedValue: CellValue = editValue

    // Try to parse as number if original was number
    if (typeof value === 'number' && editValue !== '') {
      const num = Number(editValue)
      if (!isNaN(num)) {
        parsedValue = num
      }
    }

    // Handle boolean
    if (typeof value === 'boolean') {
      parsedValue = editValue === 'true'
    }

    // Handle null
    if (editValue === '' || editValue === 'null') {
      parsedValue = null
    }

    onUpdate(fieldKey, parsedValue)
    setIsEditing(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSave()
    } else if (e.key === 'Escape') {
      setEditValue(String(value ?? ''))
      setIsEditing(false)
    }
  }

  const displayValue = value === null ? 'null' : String(value)
  const isLongText = displayValue.length > 100

  return (
    <div className="grid grid-cols-[minmax(150px,1fr)_2fr] gap-3 py-2 border-b last:border-b-0">
      <div className="font-mono text-sm flex items-start gap-2 pt-2">
        <span className="text-muted-foreground">{fieldKey}</span>
        {isPrimaryKey && (
          <Badge variant="outline" className="text-xs">PK</Badge>
        )}
        {isForeignKey && (
          <Badge variant="secondary" className="text-xs">FK</Badge>
        )}
      </div>
      <div className="min-w-0">
        {!editable || isPrimaryKey ? (
          <div className="font-mono text-sm p-2 bg-muted/30 rounded border">
            {displayValue}
          </div>
        ) : isEditing ? (
          <div className="space-y-2">
            {isLongText ? (
              <Textarea
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onKeyDown={handleKeyDown}
                className="font-mono text-sm min-h-[100px]"
                autoFocus
              />
            ) : (
              <Input
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                onKeyDown={handleKeyDown}
                className="font-mono text-sm"
                autoFocus
              />
            )}
            <div className="flex gap-2">
              <Button size="sm" onClick={handleSave}>Save</Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setEditValue(String(value ?? ''))
                  setIsEditing(false)
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <div
            className="font-mono text-sm p-2 rounded border cursor-pointer hover:bg-muted/50 transition-colors"
            onClick={() => {
              setEditValue(String(value ?? ''))
              setIsEditing(true)
            }}
          >
            {displayValue}
          </div>
        )}
      </div>
    </div>
  )
}

function ForeignKeyBadges({
  metadata,
  rowData
}: {
  metadata?: QueryEditableMetadata | null
  rowData: TableRow
}) {
  if (!metadata?.columns) return null

  const foreignKeys = metadata.columns.filter(col => {
    const columnName = col.resultName || col.name
    return col.foreignKey && columnName in rowData && rowData[columnName] !== null
  })

  if (foreignKeys.length === 0) return null

  return (
    <div className="p-4 border-b bg-muted/30">
      <div className="flex items-center gap-2 mb-3 text-sm font-medium">
        <Database className="h-4 w-4" />
        Foreign Key Relationships
        <Badge variant="secondary" className="text-xs">
          {foreignKeys.length}
        </Badge>
      </div>
      <div className="flex flex-wrap gap-2">
        {foreignKeys.map(col => {
          const columnName = col.resultName || col.name
          const value = rowData[columnName]
          const fk = col.foreignKey!

          return (
            <Badge key={columnName} variant="outline" className="text-xs">
              {columnName}: {String(value)} → {fk.table}.{fk.column}
            </Badge>
          )
        })}
      </div>
    </div>
  )
}

export function JsonRowViewerSidebarV2({
  open,
  onClose,
  rowData,
  rowId,
  rowIndex,
  totalRows,
  onNavigate,
  metadata,
  onSave
}: JsonRowViewerSidebarV2Props) {
  const [editedData, setEditedData] = useState<Record<string, CellValue>>({})
  const [isSaving, setIsSaving] = useState(false)
  const { toast } = useToast()

  // Reset edited data when row changes
  useEffect(() => {
    if (rowData) {
      setEditedData({ ...rowData })
    }
  }, [rowData, rowId])

  // Keyboard shortcuts
  useEffect(() => {
    if (!open) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      } else if (e.key === 'ArrowUp' && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
        const activeElement = document.activeElement
        if (activeElement?.tagName !== 'INPUT' && activeElement?.tagName !== 'TEXTAREA') {
          e.preventDefault()
          if (rowIndex > 0) {
            onNavigate('prev')
          }
        }
      } else if (e.key === 'ArrowDown' && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
        const activeElement = document.activeElement
        if (activeElement?.tagName !== 'INPUT' && activeElement?.tagName !== 'TEXTAREA') {
          e.preventDefault()
          if (rowIndex < totalRows - 1) {
            onNavigate('next')
          }
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose, onNavigate, rowIndex, totalRows])

  const handleUpdateField = (key: string, value: CellValue) => {
    setEditedData(prev => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    if (!onSave || !rowId) return

    setIsSaving(true)
    try {
      const success = await onSave(rowId, editedData)
      if (success) {
        toast({
          title: 'Changes saved',
          description: 'Row updated successfully',
          duration: 2000
        })
      } else {
        toast({
          title: 'Save failed',
          description: 'Failed to update row',
          variant: 'destructive',
          duration: 3000
        })
      }
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
        duration: 3000
      })
    } finally {
      setIsSaving(false)
    }
  }

  const handleCopyJson = () => {
    if (!editedData) return

    const jsonString = JSON.stringify(editedData, null, 2)
    navigator.clipboard.writeText(jsonString)
      .then(() => {
        toast({
          title: 'Copied to clipboard',
          description: 'JSON data copied successfully',
          duration: 2000
        })
      })
      .catch(() => {
        toast({
          title: 'Copy failed',
          description: 'Failed to copy JSON to clipboard',
          variant: 'destructive',
          duration: 2000
        })
      })
  }

  if (!open || !rowData || !editedData) {
    return null
  }

  // Get primary keys from metadata
  const primaryKeys = new Set(metadata?.primaryKeys || [])

  // Get foreign key columns from metadata
  const foreignKeyColumns = new Set(
    metadata?.columns
      ?.filter(col => col.foreignKey)
      .map(col => col.resultName || col.name) || []
  )

  // Filter out internal fields
  const fields = Object.entries(editedData)
    .filter(([key]) => !key.startsWith('__'))
    .sort(([keyA], [keyB]) => {
      // Sort: PK first, then FK, then regular fields
      const aIsPK = primaryKeys.has(keyA)
      const bIsPK = primaryKeys.has(keyB)
      const aIsFK = foreignKeyColumns.has(keyA)
      const bIsFK = foreignKeyColumns.has(keyB)

      if (aIsPK && !bIsPK) return -1
      if (!aIsPK && bIsPK) return 1
      if (aIsFK && !bIsFK) return -1
      if (!aIsFK && bIsFK) return 1
      return keyA.localeCompare(keyB)
    })

  const hasChanges = JSON.stringify(rowData) !== JSON.stringify(editedData)
  const canSave = hasChanges && !isSaving && !!onSave

  return (
    <Sheet open={open} onOpenChange={onClose}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-2xl m-4 h-[calc(100vh-2rem)] rounded-xl shadow-2xl border overflow-hidden flex flex-col"
      >
        <SheetHeader className="border-b pb-4">
          <div className="flex items-center justify-between">
            <SheetTitle className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              Row Details
              {rowId && (
                <Badge variant="outline" className="text-xs">
                  {rowId}
                </Badge>
              )}
            </SheetTitle>
            <RowNavigation
              currentIndex={rowIndex}
              totalRows={totalRows}
              onPrevious={() => onNavigate('prev')}
              onNext={() => onNavigate('next')}
              hasPrevious={rowIndex > 0}
              hasNext={rowIndex < totalRows - 1}
            />
          </div>
          <SheetDescription>
            View and edit row data. Use ↑↓ arrows to navigate between rows.
          </SheetDescription>
        </SheetHeader>

        {/* Toolbar */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-2">
            {canSave && (
              <Button
                variant="default"
                size="sm"
                onClick={handleSave}
                disabled={isSaving}
              >
                {isSaving ? (
                  <>
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                    Saving...
                  </>
                ) : (
                  'Save Changes'
                )}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={handleCopyJson}
            >
              <Copy className="h-3 w-3 mr-1" />
              Copy JSON
            </Button>
          </div>
          <div className="text-xs text-muted-foreground">
            {fields.length} fields
          </div>
        </div>

        {/* Foreign Keys */}
        <ForeignKeyBadges metadata={metadata} rowData={rowData} />

        {/* Field List */}
        <div className="flex-1 min-h-0">
          <ScrollArea className="h-full">
            <div className="p-4">
              {fields.map(([key, value]) => (
                <FieldRow
                  key={key}
                  fieldKey={key}
                  value={value}
                  editable={metadata?.enabled ?? false}
                  onUpdate={handleUpdateField}
                  isPrimaryKey={primaryKeys.has(key)}
                  isForeignKey={foreignKeyColumns.has(key)}
                />
              ))}
            </div>
          </ScrollArea>
        </div>

        {/* Footer */}
        {hasChanges && (
          <div className="p-3 border-t bg-amber-50 text-amber-800 text-sm">
            You have unsaved changes
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}
