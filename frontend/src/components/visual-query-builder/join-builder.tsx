/**
 * Join Builder Component for Visual Query Builder
 * Handles JOIN clause construction for single-connection queries
 */

import { ArrowRight,Link, Plus, X } from 'lucide-react'
import { useEffect,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Expr, Predicate,TableRef } from '@/lib/query-ir'
import { FilterOperator } from '@/workers/types'

import { JoinBuilderProps, TableInfo } from './types'

export function JoinBuilder({
  availableTables,
  existingJoins,
  onJoinsChange
}: JoinBuilderProps) {
  const [joins, setJoins] = useState<Array<{
    id: string
    type: 'inner' | 'left' | 'right' | 'full'
    table: TableRef
    on: Expr
  }>>(existingJoins)
  const [nextId, setNextId] = useState(1)

  // Update parent when joins change
  useEffect(() => {
    onJoinsChange(joins)
  }, [joins, onJoinsChange])

  // Add new join
  const addJoin = () => {
    if (availableTables.length === 0) return
    
    const newJoin = {
      id: `join-${nextId}`,
      type: 'inner' as const,
      table: {
        schema: availableTables[0].schema,
        table: availableTables[0].name,
        alias: availableTables[0].name
      },
      on: {
        column: availableTables[0].columns[0]?.name || '',
        operator: FilterOperator.EQUALS,
        value: ''
      } as Predicate
    }
    
    setJoins(prev => [...prev, newJoin])
    setNextId(prev => prev + 1)
  }

  // Remove join
  const removeJoin = (id: string) => {
    setJoins(prev => prev.filter(join => join.id !== id))
  }

  // Update join
  const updateJoin = (id: string, updates: Partial<typeof joins[0]>) => {
    setJoins(prev => prev.map(join => 
      join.id === id ? { ...join, ...updates } : join
    ))
  }

  // Get table info
  const getTableInfo = (tableName: string): TableInfo | undefined => {
    return availableTables.find(table => table.name === tableName)
  }

  // Get join type label
  const getJoinTypeLabel = (type: string) => {
    switch (type) {
      case 'inner': return 'INNER JOIN'
      case 'left': return 'LEFT JOIN'
      case 'right': return 'RIGHT JOIN'
      case 'full': return 'FULL JOIN'
      default: return type.toUpperCase()
    }
  }

  // Render join condition
  const renderJoinCondition = (join: typeof joins[0]) => {
    const table = getTableInfo(join.table.table)
    if (!table) return null

    return (
      <div className="space-y-2">
        <div className="text-sm font-medium">Join Condition</div>
        
        <div className="grid grid-cols-3 gap-2">
          <Select
            value={'column' in join.on ? join.on.column : ''}
            onValueChange={(value) => updateJoin(join.id, {
              on: 'column' in join.on ? { ...join.on, column: value } as Expr : join.on
            })}
          >
            <SelectTrigger>
              <SelectValue placeholder="Left column" />
            </SelectTrigger>
            <SelectContent>
              {table.columns.map(column => (
                <SelectItem key={column.name} value={column.name}>
                  {column.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select
            value={'operator' in join.on ? join.on.operator : ''}
            onValueChange={(value) => updateJoin(join.id, {
              on: 'operator' in join.on ? { ...join.on, operator: value as FilterOperator } as Expr : join.on
            })}
          >
            <SelectTrigger>
              <SelectValue placeholder="Operator" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={FilterOperator.EQUALS}>=</SelectItem>
              <SelectItem value={FilterOperator.NOT_EQUALS}>!=</SelectItem>
              <SelectItem value={FilterOperator.GREATER_THAN}>&gt;</SelectItem>
              <SelectItem value={FilterOperator.LESS_THAN}>&lt;</SelectItem>
            </SelectContent>
          </Select>

          <Input
            value={'value' in join.on ? String(join.on.value || '') : ''}
            onChange={(e) => updateJoin(join.id, {
              on: 'value' in join.on ? { ...join.on, value: e.target.value } as Expr : join.on
            })}
            placeholder="Right column or value"
          />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Table Joins</h3>
        <Button variant="outline" size="sm" onClick={addJoin}>
          <Plus className="w-4 h-4 mr-1" />
          Add Join
        </Button>
      </div>

      {joins.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          <Link className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p>No joins configured</p>
          <p className="text-xs mt-1">Add joins to combine data from multiple tables</p>
        </div>
      ) : (
        <div className="space-y-3">
          {joins.map(join => {
            const table = getTableInfo(join.table.table)
            
            return (
              <Card key={join.id} className="w-full">
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <Badge variant="outline">
                        {getJoinTypeLabel(join.type)}
                      </Badge>
                      <ArrowRight className="w-4 h-4" />
                      <span className="font-medium">
                        {join.table.schema}.{join.table.table}
                      </span>
                      {join.table.alias && join.table.alias !== join.table.table && (
                        <Badge variant="secondary" className="text-xs">
                          as {join.table.alias}
                        </Badge>
                      )}
                    </div>
                    
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => removeJoin(join.id)}
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  </div>
                </CardHeader>
                
                <CardContent className="pt-0">
                  <div className="space-y-3">
                    {/* Join Type */}
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-1 block">
                        Join Type
                      </label>
                      <Select
                        value={join.type}
                        onValueChange={(value) => updateJoin(join.id, { 
                          type: value as 'inner' | 'left' | 'right' | 'full' 
                        })}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="inner">INNER JOIN</SelectItem>
                          <SelectItem value="left">LEFT JOIN</SelectItem>
                          <SelectItem value="right">RIGHT JOIN</SelectItem>
                          <SelectItem value="full">FULL JOIN</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    {/* Table Selection */}
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-1 block">
                        Table
                      </label>
                      <Select
                        value={join.table.table}
                        onValueChange={(value) => {
                          const selectedTable = availableTables.find(t => t.name === value)
                          if (selectedTable) {
                            updateJoin(join.id, {
                              table: {
                                schema: selectedTable.schema,
                                table: selectedTable.name,
                                alias: selectedTable.name
                              }
                            })
                          }
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select table..." />
                        </SelectTrigger>
                        <SelectContent>
                          {availableTables.map(table => (
                            <SelectItem key={table.name} value={table.name}>
                              <div className="flex items-center space-x-2">
                                <span>{table.name}</span>
                                <Badge variant="outline" className="text-xs">
                                  {table.schema}
                                </Badge>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    {/* Table Alias */}
                    <div>
                      <label className="text-xs font-medium text-muted-foreground mb-1 block">
                        Alias (optional)
                      </label>
                      <Input
                        value={join.table.alias || ''}
                        onChange={(e) => updateJoin(join.id, {
                          table: { ...join.table, alias: e.target.value }
                        })}
                        placeholder={join.table.table}
                      />
                    </div>

                    {/* Join Condition */}
                    {table && renderJoinCondition(join)}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {/* Join Summary */}
      {joins.length > 0 && (
        <div className="p-3 bg-muted rounded-md">
          <div className="text-sm font-medium mb-2">
            Join Summary ({joins.length})
          </div>
          <div className="space-y-1">
            {joins.map((join, index) => (
              <div key={join.id} className="text-xs text-muted-foreground">
                {index === 0 ? 'FROM' : getJoinTypeLabel(join.type)} {join.table.schema}.{join.table.table}
                {join.table.alias && join.table.alias !== join.table.table && ` AS ${join.table.alias}`}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
