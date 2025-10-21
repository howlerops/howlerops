/**
 * Filter Editor Component for Visual Query Builder
 * Handles WHERE clause construction with typed inputs
 */

import { useState, useEffect, startTransition } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Plus, X, Trash2 } from 'lucide-react'
import { FilterEditorProps, ColumnInfo, FilterCondition, FilterGroup } from './types'
import { FilterOperator } from '@/workers/types'
import { typeRegistry } from '@/lib/type-registry'
import { Expr, Predicate, Group } from '@/lib/query-ir'

const convertConditionToExpr = (condition: FilterCondition | FilterGroup): Expr => {
  if ('column' in condition) {
    return {
      column: condition.column,
      operator: condition.operator,
      value: condition.value,
      not: condition.not,
    } as Predicate
  }

  return {
    operator: condition.operator,
    conditions: condition.conditions
      .map(convertConditionToExpr)
      .filter((child): child is Expr => Boolean(child)),
    not: condition.not,
  } as Group
}

const convertConditionsToExpr = (conditions: (FilterCondition | FilterGroup)[]): Expr | undefined => {
  if (conditions.length === 0) {
    return undefined
  }

  if (conditions.length === 1) {
    return convertConditionToExpr(conditions[0])
  }

  return {
    operator: 'AND',
    conditions: conditions.map(convertConditionToExpr),
  } as Group
}

const buildConditionsFromExpr = (
  expr: Expr,
  startId = 1
): { conditions: (FilterCondition | FilterGroup)[]; nextId: number } => {
  let nextId = startId

  const createId = (prefix: 'condition' | 'group') => {
    const id = `${prefix}-${nextId}`
    nextId += 1
    return id
  }

  const convert = (node: Expr): FilterCondition | FilterGroup => {
    if ('column' in node) {
      return {
        id: createId('condition'),
        column: node.column,
        operator: node.operator,
        value: node.value,
        not: node.not,
      }
    }

    return {
      id: createId('group'),
      operator: node.operator,
      conditions: node.conditions.map(convert),
      not: node.not,
    }
  }

  const root = convert(expr)
  return {
    conditions: [root],
    nextId,
  }
}

export function FilterEditor({
  columns,
  where,
  onWhereChange
}: FilterEditorProps) {
  const [conditions, setConditions] = useState<(FilterCondition | FilterGroup)[]>([])
  const [nextId, setNextId] = useState(1)

  useEffect(() => {
    startTransition(() => {
      if (!where) {
        setConditions([])
        setNextId(1)
        return
      }

      const { conditions: initialConditions, nextId: generatedNextId } = buildConditionsFromExpr(where)
      setConditions(initialConditions)
      setNextId(generatedNextId)
    })
  }, [where])

  useEffect(() => {
    onWhereChange(convertConditionsToExpr(conditions))
  }, [conditions, onWhereChange])

  // Convert Expr to UI state
  // Add new condition
  const addCondition = () => {
    const newCondition: FilterCondition = {
      id: `condition-${nextId}`,
      column: columns[0]?.name || '',
      operator: FilterOperator.EQUALS,
      value: ''
    }
    setConditions(prev => [...prev, newCondition])
    setNextId(prev => prev + 1)
  }

  // Add new group
  const addGroup = () => {
    const newGroup: FilterGroup = {
      id: `group-${nextId}`,
      operator: 'AND',
      conditions: []
    }
    setConditions(prev => [...prev, newGroup])
    setNextId(prev => prev + 1)
  }

  // Remove condition/group
  const removeCondition = (id: string) => {
    setConditions(prev => prev.filter(condition => condition.id !== id))
  }

  // Update condition
  const updateCondition = (id: string, updates: Partial<FilterCondition>) => {
    setConditions(prev => prev.map(condition => 
      condition.id === id ? { ...condition, ...updates } : condition
    ))
  }

  // Update group
  const updateGroup = (id: string, updates: Partial<FilterGroup>) => {
    setConditions(prev => prev.map(condition => 
      condition.id === id ? { ...condition, ...updates } : condition
    ))
  }

  // Get available operators for a column
  const getOperatorsForColumn = (columnName: string): FilterOperator[] => {
    const column = columns.find(col => col.name === columnName)
    if (!column) return []

    const fieldType = typeRegistry.inferFieldType({
      name: column.name,
      dataType: column.dataType,
      isNullable: column.isNullable,
      isPrimaryKey: column.isPrimaryKey,
      isForeignKey: column.isForeignKey,
      enumValues: column.enumValues
    })

    return typeRegistry.getOperators(fieldType)
  }

  // Get column info
  const getColumnInfo = (columnName: string): ColumnInfo | undefined => {
    return columns.find(col => col.name === columnName)
  }

  // Render condition input based on operator and column type
  const renderValueInput = (condition: FilterCondition) => {
    const column = getColumnInfo(condition.column)
    if (!column) return null

    const fieldType = typeRegistry.inferFieldType({
      name: column.name,
      dataType: column.dataType,
      isNullable: column.isNullable,
      isPrimaryKey: column.isPrimaryKey,
      isForeignKey: column.isForeignKey,
      enumValues: column.enumValues
    })

    // Handle null checks
    if (condition.operator === FilterOperator.IS_NULL || condition.operator === FilterOperator.IS_NOT_NULL) {
      return null
    }

    // Handle BETWEEN operator
    if (condition.operator === FilterOperator.BETWEEN) {
      const values = Array.isArray(condition.value) ? condition.value : ['', '']
      return (
        <div className="flex items-center space-x-2">
          <Input
            type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
            value={values[0] || ''}
            onChange={(e) => updateCondition(condition.id, {
              value: [e.target.value, values[1] || '']
            })}
            placeholder="Min"
            className="w-20"
          />
          <span className="text-sm text-muted-foreground">and</span>
          <Input
            type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
            value={values[1] || ''}
            onChange={(e) => updateCondition(condition.id, {
              value: [values[0] || '', e.target.value]
            })}
            placeholder="Max"
            className="w-20"
          />
        </div>
      )
    }

    // Handle IN operator
    if (condition.operator === FilterOperator.IN || condition.operator === FilterOperator.NOT_IN) {
      const values = Array.isArray(condition.value) ? condition.value : []
      return (
        <div className="space-y-2">
          {values.map((value, index) => (
            <div key={index} className="flex items-center space-x-2">
              <Input
                type={fieldType === 'number' ? 'number' : 'text'}
                value={value}
                onChange={(e) => {
                  const newValues = [...values]
                  newValues[index] = e.target.value
                  updateCondition(condition.id, { value: newValues })
                }}
                placeholder="Value"
                className="flex-1"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  const newValues = values.filter((_, i) => i !== index)
                  updateCondition(condition.id, { value: newValues })
                }}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          ))}
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              const newValues = [...values, '']
              updateCondition(condition.id, { value: newValues })
            }}
          >
            <Plus className="w-4 h-4 mr-1" />
            Add Value
          </Button>
        </div>
      )
    }

    // Handle enum/select columns
    if (fieldType === 'enum' && column.enumValues?.length) {
      return (
        <Select
          value={String(condition.value || '')}
          onValueChange={(value) => updateCondition(condition.id, { value })}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select value..." />
          </SelectTrigger>
          <SelectContent>
            {column.enumValues.map(value => (
              <SelectItem key={value} value={value}>
                {value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    }

    // Handle boolean columns
    if (fieldType === 'boolean') {
      return (
        <Select
          value={String(condition.value || '')}
          onValueChange={(value) => updateCondition(condition.id, { value: value === 'true' })}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select value..." />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="true">True</SelectItem>
            <SelectItem value="false">False</SelectItem>
          </SelectContent>
        </Select>
      )
    }

    // Default input
    return (
      <Input
        type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
        value={String(condition.value || '')}
        onChange={(e) => updateCondition(condition.id, { value: e.target.value })}
        placeholder="Enter value..."
      />
    )
  }

  // Render condition
  const renderCondition = (condition: FilterCondition) => (
    <Card key={condition.id} className="w-full">
      <CardContent className="p-4">
        <div className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={condition.not}
            onChange={(e) => updateCondition(condition.id, { not: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm font-medium">NOT</span>
        </div>
        
        <div className="grid grid-cols-4 gap-2 mt-3">
          <Select
            value={condition.column}
            onValueChange={(value) => updateCondition(condition.id, { column: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder="Column" />
            </SelectTrigger>
            <SelectContent>
              {columns.map(column => (
                <SelectItem key={column.name} value={column.name}>
                  {column.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select
            value={condition.operator}
            onValueChange={(value) => updateCondition(condition.id, { operator: value as FilterOperator })}
          >
            <SelectTrigger>
              <SelectValue placeholder="Operator" />
            </SelectTrigger>
            <SelectContent>
              {getOperatorsForColumn(condition.column).map(operator => (
                <SelectItem key={operator} value={operator}>
                  {operator.replace(/_/g, ' ').toLowerCase()}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="col-span-2">
            {renderValueInput(condition)}
          </div>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => removeCondition(condition.id)}
          className="mt-2"
        >
          <Trash2 className="w-4 h-4 mr-1" />
          Remove
        </Button>
      </CardContent>
    </Card>
  )

  // Render group
  const renderGroup = (group: FilterGroup) => (
    <Card key={group.id} className="w-full">
      <CardHeader className="pb-2">
        <div className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={group.not}
            onChange={(e) => updateGroup(group.id, { not: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm font-medium">NOT</span>
          
          <Select
            value={group.operator}
            onValueChange={(value) => updateGroup(group.id, { operator: value as 'AND' | 'OR' })}
          >
            <SelectTrigger className="w-20">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="AND">AND</SelectItem>
              <SelectItem value="OR">OR</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <div className="space-y-2">
          {group.conditions.map(condition => 
            'column' in condition 
              ? renderCondition(condition)
              : renderGroup(condition)
          )}
        </div>
        
        <Button
          variant="outline"
          size="sm"
          onClick={() => removeCondition(group.id)}
          className="mt-2"
        >
          <Trash2 className="w-4 h-4 mr-1" />
          Remove Group
        </Button>
      </CardContent>
    </Card>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Filters</h3>
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" onClick={addCondition}>
            <Plus className="w-4 h-4 mr-1" />
            Add Condition
          </Button>
          <Button variant="outline" size="sm" onClick={addGroup}>
            <Plus className="w-4 h-4 mr-1" />
            Add Group
          </Button>
        </div>
      </div>

      {conditions.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          <p>No filters applied</p>
          <p className="text-xs mt-1">Add conditions to filter your results</p>
        </div>
      ) : (
        <div className="space-y-2">
          {conditions.map(condition => 
            'column' in condition 
              ? renderCondition(condition)
              : renderGroup(condition)
          )}
        </div>
      )}
    </div>
  )
}
