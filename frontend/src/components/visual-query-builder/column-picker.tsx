/**
 * Column Picker Component for Visual Query Builder
 * Handles column selection and aliasing
 */

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus, X, Hash, Text, Calendar, ToggleLeft, Key } from 'lucide-react'
import { ColumnPickerProps, ColumnInfo, SelectItem as SelectItemType } from './types'
import { typeRegistry } from '@/lib/type-registry'

export function ColumnPicker({
  table,
  columns,
  selectedColumns,
  onColumnsChange
}: ColumnPickerProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [showAll, setShowAll] = useState(false)

  // Filter columns based on search term
  const filteredColumns = columns.filter(column =>
    column.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    column.dataType.toLowerCase().includes(searchTerm.toLowerCase())
  )

  // Get column type icon
  const getColumnIcon = (column: ColumnInfo) => {
    const fieldType = typeRegistry.inferFieldType({
      name: column.name,
      dataType: column.dataType,
      isNullable: column.isNullable,
      isPrimaryKey: column.isPrimaryKey,
      isForeignKey: column.isForeignKey,
      enumValues: column.enumValues
    })

    switch (fieldType) {
      case 'number':
        return <Hash className="w-4 h-4" />
      case 'date':
      case 'datetime':
        return <Calendar className="w-4 h-4" />
      case 'boolean':
        return <ToggleLeft className="w-4 h-4" />
      default:
        return <Text className="w-4 h-4" />
    }
  }

  // Get column type badge
  const getColumnTypeBadge = (column: ColumnInfo) => {
    const fieldType = typeRegistry.inferFieldType({
      name: column.name,
      dataType: column.dataType,
      isNullable: column.isNullable,
      isPrimaryKey: column.isPrimaryKey,
      isForeignKey: column.isForeignKey,
      enumValues: column.enumValues
    })

    const typeColors: Record<string, string> = {
      text: 'bg-blue-100 text-blue-800',
      number: 'bg-green-100 text-green-800',
      boolean: 'bg-purple-100 text-purple-800',
      date: 'bg-orange-100 text-orange-800',
      datetime: 'bg-orange-100 text-orange-800',
      enum: 'bg-pink-100 text-pink-800',
      uuid: 'bg-gray-100 text-gray-800',
      json: 'bg-yellow-100 text-yellow-800',
      array: 'bg-indigo-100 text-indigo-800'
    }

    return (
      <Badge variant="outline" className={`text-xs ${typeColors[fieldType] || 'bg-gray-100 text-gray-800'}`}>
        {fieldType}
      </Badge>
    )
  }

  // Handle column selection
  const handleColumnToggle = (column: ColumnInfo) => {
    const isSelected = selectedColumns.some(sel => sel.column === column.name)
    
    if (isSelected) {
      onColumnsChange(selectedColumns.filter(sel => sel.column !== column.name))
    } else {
      onColumnsChange([...selectedColumns, {
        column: column.name,
        alias: column.name
      }])
    }
  }

  // Handle alias change
  const handleAliasChange = (columnName: string, alias: string) => {
    onColumnsChange(selectedColumns.map(sel => 
      sel.column === columnName 
        ? { ...sel, alias: alias || columnName }
        : sel
    ))
  }

  // Handle aggregate change
  const handleAggregateChange = (columnName: string, aggregate: string) => {
    onColumnsChange(selectedColumns.map(sel => 
      sel.column === columnName 
        ? { ...sel, aggregate: aggregate as any }
        : sel
    ))
  }

  // Select all columns
  const handleSelectAll = () => {
    const allColumns = columns.map(column => ({
      column: column.name,
      alias: column.name
    }))
    onColumnsChange(allColumns)
  }

  // Clear all columns
  const handleClearAll = () => {
    onColumnsChange([])
  }

  if (!table) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <p>Select a table to choose columns</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Columns</h3>
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" onClick={handleSelectAll}>
            Select All
          </Button>
          <Button variant="outline" size="sm" onClick={handleClearAll}>
            Clear All
          </Button>
        </div>
      </div>

      {/* Search */}
      <Input
        placeholder="Search columns..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        className="w-full"
      />

      {/* Column List */}
      <ScrollArea className="h-64">
        <div className="space-y-2">
          {filteredColumns.map(column => {
            const isSelected = selectedColumns.some(sel => sel.column === column.name)
            const selectedItem = selectedColumns.find(sel => sel.column === column.name)
            
            return (
              <div key={column.name} className="space-y-2">
                <div className="flex items-center space-x-2 p-2 border rounded-md">
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onChange={() => handleColumnToggle(column)}
                    className="rounded"
                  />
                  
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center space-x-2">
                      {getColumnIcon(column)}
                      <span className="font-medium truncate">{column.name}</span>
                      {column.isPrimaryKey && (
                        <Badge variant="outline" className="text-xs">
                          <Key className="w-3 h-3 mr-1" />
                          PK
                        </Badge>
                      )}
                      {column.isForeignKey && (
                        <Badge variant="outline" className="text-xs">
                          FK
                        </Badge>
                      )}
                    </div>
                    
                    <div className="flex items-center space-x-2 mt-1">
                      {getColumnTypeBadge(column)}
                      <span className="text-xs text-muted-foreground">
                        {column.dataType}
                        {column.isNullable && ' (nullable)'}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Column Configuration */}
                {isSelected && selectedItem && (
                  <div className="ml-6 p-3 bg-muted rounded-md space-y-3">
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">
                        Alias
                      </label>
                      <Input
                        value={selectedItem.alias || ''}
                        onChange={(e) => handleAliasChange(column.name, e.target.value)}
                        placeholder={column.name}
                        className="mt-1"
                      />
                    </div>
                    
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">
                        Aggregate Function
                      </label>
                      <Select
                        value={selectedItem.aggregate || ''}
                        onValueChange={(value) => handleAggregateChange(column.name, value)}
                      >
                        <SelectTrigger className="mt-1">
                          <SelectValue placeholder="None" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="">None</SelectItem>
                          <SelectItem value="count">Count</SelectItem>
                          <SelectItem value="sum">Sum</SelectItem>
                          <SelectItem value="avg">Average</SelectItem>
                          <SelectItem value="min">Min</SelectItem>
                          <SelectItem value="max">Max</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </ScrollArea>

      {/* Selected Columns Summary */}
      {selectedColumns.length > 0 && (
        <div className="p-3 bg-muted rounded-md">
          <div className="text-sm font-medium mb-2">
            Selected Columns ({selectedColumns.length})
          </div>
          <div className="flex flex-wrap gap-2">
            {selectedColumns.map((item, index) => (
              <Badge key={index} variant="secondary" className="text-xs">
                {item.column}
                {item.alias && item.alias !== item.column && ` as ${item.alias}`}
                {item.aggregate && ` (${item.aggregate})`}
              </Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
