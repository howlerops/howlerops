/**
 * Sort and Limit Component for Visual Query Builder
 * Handles ORDER BY, LIMIT, and OFFSET clauses
 */

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Plus, X, ArrowUp, ArrowDown } from 'lucide-react'
import { SortLimitProps, ColumnInfo, OrderBy } from './types'

export function SortLimit({
  columns,
  orderBy,
  limit,
  offset,
  onOrderByChange,
  onLimitChange,
  onOffsetChange
}: SortLimitProps) {
  const [sortColumns, setSortColumns] = useState<OrderBy[]>(orderBy)
  const [limitValue, setLimitValue] = useState(limit?.toString() || '')
  const [offsetValue, setOffsetValue] = useState(offset?.toString() || '')

  // Add sort column
  const addSortColumn = () => {
    if (columns.length === 0) return
    
    const newSort: OrderBy = {
      column: columns[0].name,
      direction: 'asc'
    }
    
    const updatedSorts = [...sortColumns, newSort]
    setSortColumns(updatedSorts)
    onOrderByChange(updatedSorts)
  }

  // Remove sort column
  const removeSortColumn = (index: number) => {
    const updatedSorts = sortColumns.filter((_, i) => i !== index)
    setSortColumns(updatedSorts)
    onOrderByChange(updatedSorts)
  }

  // Update sort column
  const updateSortColumn = (index: number, updates: Partial<OrderBy>) => {
    const updatedSorts = sortColumns.map((sort, i) => 
      i === index ? { ...sort, ...updates } : sort
    )
    setSortColumns(updatedSorts)
    onOrderByChange(updatedSorts)
  }

  // Handle limit change
  const handleLimitChange = (value: string) => {
    setLimitValue(value)
    const numValue = parseInt(value, 10)
    if (!isNaN(numValue) && numValue > 0) {
      onLimitChange(numValue)
    } else if (value === '') {
      onLimitChange(undefined)
    }
  }

  // Handle offset change
  const handleOffsetChange = (value: string) => {
    setOffsetValue(value)
    const numValue = parseInt(value, 10)
    if (!isNaN(numValue) && numValue >= 0) {
      onOffsetChange(numValue)
    } else if (value === '') {
      onOffsetChange(0)
    }
  }

  // Get column info
  const getColumnInfo = (columnName: string): ColumnInfo | undefined => {
    return columns.find(col => col.name === columnName)
  }

  return (
    <div className="space-y-4">
      {/* Sort Columns */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium">Sort Order</h3>
          <Button variant="outline" size="sm" onClick={addSortColumn}>
            <Plus className="w-4 h-4 mr-1" />
            Add Sort
          </Button>
        </div>

        {sortColumns.length === 0 ? (
          <div className="text-center py-4 text-muted-foreground border-2 border-dashed rounded-md">
            <p className="text-sm">No sorting applied</p>
            <p className="text-xs mt-1">Results will be returned in default order</p>
          </div>
        ) : (
          <div className="space-y-2">
            {sortColumns.map((sort, index) => {
              const columnInfo = getColumnInfo(sort.column)
              
              return (
                <Card key={index} className="w-full">
                  <CardContent className="p-3">
                    <div className="flex items-center space-x-2">
                      <Badge variant="outline" className="text-xs">
                        {index + 1}
                      </Badge>
                      
                      <Select
                        value={sort.column}
                        onValueChange={(value) => updateSortColumn(index, { column: value })}
                      >
                        <SelectTrigger className="flex-1">
                          <SelectValue placeholder="Select column..." />
                        </SelectTrigger>
                        <SelectContent>
                          {columns.map(column => (
                            <SelectItem key={column.name} value={column.name}>
                              <div className="flex items-center space-x-2">
                                <span>{column.name}</span>
                                <Badge variant="outline" className="text-xs">
                                  {column.dataType}
                                </Badge>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>

                      <Select
                        value={sort.direction}
                        onValueChange={(value) => updateSortColumn(index, { direction: value as 'asc' | 'desc' })}
                      >
                        <SelectTrigger className="w-24">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="asc">
                            <div className="flex items-center space-x-1">
                              <ArrowUp className="w-3 h-3" />
                              <span>Asc</span>
                            </div>
                          </SelectItem>
                          <SelectItem value="desc">
                            <div className="flex items-center space-x-1">
                              <ArrowDown className="w-3 h-3" />
                              <span>Desc</span>
                            </div>
                          </SelectItem>
                        </SelectContent>
                      </Select>

                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => removeSortColumn(index)}
                      >
                        <X className="w-4 h-4" />
                      </Button>
                    </div>
                    
                    {columnInfo && (
                      <div className="mt-2 text-xs text-muted-foreground">
                        {columnInfo.dataType}
                        {columnInfo.isPrimaryKey && ' • Primary Key'}
                        {columnInfo.isForeignKey && ' • Foreign Key'}
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>

      {/* Limit and Offset */}
      <div>
        <h3 className="text-sm font-medium mb-3">Result Limits</h3>
        
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1 block">
              Limit
            </label>
            <Input
              type="number"
              value={limitValue}
              onChange={(e) => handleLimitChange(e.target.value)}
              placeholder="No limit"
              min="1"
              className="w-full"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Maximum number of rows to return
            </p>
          </div>
          
          <div>
            <label className="text-xs font-medium text-muted-foreground mb-1 block">
              Offset
            </label>
            <Input
              type="number"
              value={offsetValue}
              onChange={(e) => handleOffsetChange(e.target.value)}
              placeholder="0"
              min="0"
              className="w-full"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Number of rows to skip
            </p>
          </div>
        </div>

        {/* Summary */}
        {(limit || offset) && (
          <div className="mt-3 p-2 bg-muted rounded-md">
            <div className="text-xs font-medium mb-1">Query Summary</div>
            <div className="text-xs text-muted-foreground">
              {limit ? `Limit: ${limit} rows` : 'No limit'}
              {offset && offset > 0 && ` • Skip: ${offset} rows`}
              {sortColumns.length > 0 && ` • Sorted by: ${sortColumns.length} column(s)`}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
