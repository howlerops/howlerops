import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  ChevronRight, 
  ChevronDown, 
  Edit3, 
  Check, 
  X, 
  AlertCircle,
  Copy
} from 'lucide-react'
import { JsonToken, getTokenClass } from '@/lib/json-formatter'
import { SearchMatch, highlightMatches } from '@/lib/json-search'
import { CellValue } from '@/types/table'
import { ForeignKeyResolver } from './foreign-key-resolver'
import { formatValue, getValueClass } from './json-editor-utils'
import type { QueryEditableMetadata } from '@/store/query-store'

interface JsonEditorProps {
  tokens: JsonToken[]
  data: Record<string, CellValue>
  isEditing: boolean
  validationErrors: Map<string, string>
  searchMatches: SearchMatch[]
  currentMatchIndex: number
  wordWrap: boolean
  expandedKeys: Set<string>
  collapsedKeys: Set<string>
  onToggleEdit: () => void
  onUpdateField: (key: string, value: CellValue) => void
  onToggleKeyExpansion: (key: string) => void
  onCopyJson: () => void
  metadata?: QueryEditableMetadata | null
  connectionId?: string
}

interface JsonNode {
  key: string
  value: CellValue
  path: string
  level: number
  isExpanded: boolean
  isEditing: boolean
  validationError?: string
}

export function JsonEditor({
  tokens,
  data,
  isEditing,
  validationErrors,
  searchMatches,
  currentMatchIndex,
  wordWrap,
  expandedKeys,
  collapsedKeys,
  onUpdateField,
  onToggleKeyExpansion,
  onCopyJson,
  metadata,
  connectionId
}: JsonEditorProps) {
  const [editingField, setEditingField] = useState<string | null>(null)
  const [editValue, setEditValue] = useState<string>('')
  const editInputRef = useRef<HTMLInputElement>(null)

  // Parse JSON data into hierarchical structure
  const jsonNodes = useMemo((): JsonNode[] => {
    const nodes: JsonNode[] = []
    
    const processObject = (obj: Record<string, CellValue>, path: string = '', level: number = 0) => {
      Object.entries(obj).forEach(([key, value]) => {
        if (key === '__rowId') return
        
        const currentPath = path ? `${path}.${key}` : key
        const isExpanded = expandedKeys.has('*') || expandedKeys.has(currentPath)
        const isCollapsed = collapsedKeys.has('*') || collapsedKeys.has(currentPath)
        const validationError = validationErrors.get(key)
        
        nodes.push({
          key,
          value,
          path: currentPath,
          level,
          isExpanded: isExpanded && !isCollapsed,
          isEditing: editingField === key,
          validationError
        })
      })
    }
    
    processObject(data)
    return nodes
  }, [data, expandedKeys, collapsedKeys, validationErrors, editingField])

  // Handle field editing
  const handleStartEdit = useCallback((key: string, currentValue: CellValue) => {
    setEditingField(key)
    setEditValue(String(currentValue ?? ''))
  }, [])

  const handleSaveEdit = useCallback(() => {
    if (!editingField) return
    
    onUpdateField(editingField, editValue)
    setEditingField(null)
    setEditValue('')
  }, [editingField, editValue, onUpdateField])

  const handleCancelEdit = useCallback(() => {
    setEditingField(null)
    setEditValue('')
  }, [])

  // Focus edit input when editing starts
  useEffect(() => {
    if (editingField && editInputRef.current) {
      editInputRef.current.focus()
      editInputRef.current.select()
    }
  }, [editingField])

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleCancelEdit()
      } else if (e.key === 'Enter' && editingField) {
        handleSaveEdit()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [editingField, handleSaveEdit, handleCancelEdit])

  // Render JSON tokens with syntax highlighting
  const renderTokens = useCallback(() => {
    if (searchMatches.length > 0) {
      // Highlight search matches
      const highlighted = highlightMatches(
        tokens.map(t => t.value).join(''),
        searchMatches,
        currentMatchIndex
      )
      return (
        <div 
          className="font-mono text-sm whitespace-pre"
          dangerouslySetInnerHTML={{ __html: highlighted }}
        />
      )
    }

    return (
      <div className="font-mono text-sm whitespace-pre">
        {tokens.map((token, index) => (
          <span
            key={index}
            className={getTokenClass(token)}
            style={{ wordBreak: wordWrap ? 'break-word' : 'normal' }}
          >
            {token.value}
          </span>
        ))}
      </div>
    )
  }, [tokens, searchMatches, currentMatchIndex, wordWrap])

  // Render individual field
  const renderField = useCallback((node: JsonNode) => {
    const { key, value, level, isExpanded, isEditing: nodeEditing, validationError } = node
    
    return (
      <div key={key} className="json-field" style={{ marginLeft: `${level * 20}px` }}>
        <div className="flex items-center gap-2 py-1">
          {/* Expand/collapse button for objects/arrays */}
          {(typeof value === 'object' && value !== null) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onToggleKeyExpansion(node.path)}
              className="h-4 w-4 p-0"
            >
              {isExpanded ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )}
            </Button>
          )}
          
          {/* Key name */}
          <span className="text-blue-600 font-medium">{key}:</span>
          
          {/* Value */}
          {nodeEditing ? (
            <div className="flex items-center gap-2 flex-1">
              <Input
                ref={editInputRef}
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                className="h-6 text-xs"
                size={1}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={handleSaveEdit}
                className="h-6 w-6 p-0 text-green-600"
              >
                <Check className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCancelEdit}
                className="h-6 w-6 p-0 text-red-600"
              >
                <X className="h-3 w-3" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2 flex-1">
              <span className={getValueClass(value)}>
                {formatValue(value)}
              </span>
              
              {/* Edit button */}
              {nodeEditing && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleStartEdit(key, value)}
                  className="h-4 w-4 p-0 opacity-0 group-hover:opacity-100"
                >
                  <Edit3 className="h-3 w-3" />
                </Button>
              )}
              
              {/* Validation error */}
              {validationError && (
                <Badge variant="destructive" className="text-xs">
                  <AlertCircle className="h-3 w-3 mr-1" />
                  {validationError}
                </Badge>
              )}
            </div>
          )}
        </div>
        
        {/* Foreign key resolver */}
          <ForeignKeyResolver
            key={`fk-${key}`}
            fieldKey={key}
            value={value}
            metadata={metadata}
            connectionId={connectionId}
            isExpanded={isExpanded}
            onToggle={onToggleKeyExpansion}
            onLoadData={() => Promise.resolve()}
          />
      </div>
    )
  }, [onToggleKeyExpansion, handleStartEdit, handleSaveEdit, handleCancelEdit, editValue, metadata, connectionId])

  return (
    <div className="json-editor">
      {/* Toolbar */}
      <div className="flex items-center justify-between p-2 border-b">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onCopyJson}
            className="h-7"
          >
            <Copy className="h-3 w-3 mr-1" />
            Copy JSON
          </Button>
        </div>
        
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {searchMatches.length > 0 && (
            <span>
              {currentMatchIndex + 1} of {searchMatches.length} matches
            </span>
          )}
        </div>
      </div>

      {/* JSON Content */}
      <ScrollArea className="h-full">
        <div className="p-4 space-y-1">
          {isEditing ? (
            // Tree view for editing
            <div className="space-y-1">
              {jsonNodes.map(renderField)}
            </div>
          ) : (
            // Syntax highlighted view
            <div className="json-content">
              {renderTokens()}
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  )
}
