import { useMemo, useState, useEffect } from "react"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import type { DatabaseConnection } from "@/store/connection-store"
import type { SchemaNode } from "@/hooks/use-schema-introspection"
import { ChevronDown, ChevronRight, Database, Table, Filter } from "lucide-react"

interface SchemaContextSelectorProps {
  connections: DatabaseConnection[]
  schemasMap: Map<string, SchemaNode[]>
  onChange?: (contextText: string) => void
  className?: string
}

interface SelectedTable {
  key: string
  connectionName: string
  schemaName: string
  tableName: string
}

export function SchemaContextSelector({ connections, schemasMap, onChange, className }: SchemaContextSelectorProps) {
  const [expandedConnections, setExpandedConnections] = useState<Record<string, boolean>>({})
  const [expandedSchemas, setExpandedSchemas] = useState<Record<string, boolean>>({})
  const [selectedTables, setSelectedTables] = useState<Record<string, SelectedTable>>({})
  const [customNotes, setCustomNotes] = useState("")

  const connectedConnections = useMemo(
    () => connections.filter(connection => connection.isConnected),
    [connections]
  )

  useEffect(() => {
    const lines: string[] = []
    const tables = Object.values(selectedTables)
    if (tables.length > 0) {
      lines.push("Use the following schema references:")
      tables.forEach(table => {
        const schemaSegment = table.schemaName ? `${table.schemaName}.` : ""
        lines.push(`- ${table.connectionName}.${schemaSegment}${table.tableName}`)
      })
    }
    if (customNotes.trim()) {
      if (lines.length > 0) {
        lines.push("\nAdditional notes:")
      }
      lines.push(customNotes.trim())
    }

    onChange?.(lines.join("\n"))
  }, [customNotes, selectedTables, onChange])

  const toggleConnection = (connectionId: string) => {
    setExpandedConnections(prev => ({
      ...prev,
      [connectionId]: !prev[connectionId],
    }))
  }

  const toggleSchema = (schemaKey: string) => {
    setExpandedSchemas(prev => ({
      ...prev,
      [schemaKey]: !prev[schemaKey],
    }))
  }

  const handleTableToggle = (table: SelectedTable, checked: boolean) => {
    setSelectedTables(prev => {
      const next = { ...prev }
      if (checked) {
        next[table.key] = table
      } else {
        delete next[table.key]
      }
      return next
    })
  }

  const handleClearSelection = () => {
    setSelectedTables({})
    setCustomNotes("")
  }

  if (connectedConnections.length === 0) {
    return (
      <div className={cn("rounded-lg border border-dashed p-4 text-sm text-muted-foreground", className)}>
        <Database className="mr-2 inline h-4 w-4" />
        Connect to a database to include schema context.
      </div>
    )
  }

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Schema Context</span>
          {Object.keys(selectedTables).length > 0 && (
            <Badge variant="secondary">{Object.keys(selectedTables).length} selected</Badge>
          )}
        </div>
        <Button variant="ghost" size="sm" onClick={handleClearSelection} disabled={Object.keys(selectedTables).length === 0 && !customNotes.trim()}>
          Clear
        </Button>
      </div>

      <ScrollArea className="max-h-64 rounded-md border">
        <div className="p-2 space-y-1">
          {connectedConnections.map(connection => {
            const schemas = schemasMap.get(connection.id) || schemasMap.get(connection.name) || []
            const isExpanded = expandedConnections[connection.id] ?? true
            return (
              <Collapsible key={connection.id} open={isExpanded} onOpenChange={() => toggleConnection(connection.id)}>
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" size="sm" className="w-full justify-start px-2 py-1 font-normal">
                    {isExpanded ? <ChevronDown className="mr-2 h-3 w-3" /> : <ChevronRight className="mr-2 h-3 w-3" />}
                    <Database className="mr-2 h-3 w-3" />
                    <span className="truncate text-sm">{connection.name || connection.id}</span>
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent className="pl-4">
                  {schemas.length === 0 ? (
                    <p className="px-2 py-1 text-xs text-muted-foreground">No schemas available</p>
                  ) : (
                    schemas.map((schemaNode, schemaIndex) => {
                      const schemaKey = `${connection.id}-${schemaNode.name}-${schemaIndex}`
                      const schemaExpanded = expandedSchemas[schemaKey] ?? schemaIndex === 0
                      const tables = schemaNode.children || []
                      return (
                        <Collapsible
                          key={schemaKey}
                          open={schemaExpanded}
                          onOpenChange={() => toggleSchema(schemaKey)}
                        >
                          <CollapsibleTrigger asChild>
                            <Button variant="ghost" size="sm" className="w-full justify-start px-2 py-1 font-normal text-xs">
                              {schemaExpanded ? <ChevronDown className="mr-2 h-3 w-3" /> : <ChevronRight className="mr-2 h-3 w-3" />}
                              <span className="truncate">{schemaNode.name}</span>
                              <Badge variant="outline" className="ml-auto text-[10px]">
                                {tables.length}
                              </Badge>
                            </Button>
                          </CollapsibleTrigger>
                          <CollapsibleContent className="pl-4 space-y-1">
                            {tables.map(tableNode => {
                              const key = `${connection.id}::${schemaNode.name || ""}::${tableNode.name}`
                              const selected = Boolean(selectedTables[key])
                              return (
                                <label
                                  key={key}
                                  className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-xs hover:bg-muted/50"
                                >
                                  <Checkbox
                                    checked={selected}
                                    onCheckedChange={checked =>
                                      handleTableToggle(
                                        {
                                          key,
                                          connectionName: connection.name || connection.id,
                                          schemaName: schemaNode.name || "",
                                          tableName: tableNode.name,
                                        },
                                        Boolean(checked)
                                      )
                                    }
                                  />
                                  <Table className="h-3 w-3 text-muted-foreground" />
                                  <span className="truncate">{tableNode.name}</span>
                                </label>
                              )
                            })}
                          </CollapsibleContent>
                        </Collapsible>
                      )
                    })
                  )}
                </CollapsibleContent>
              </Collapsible>
            )
          })}
        </div>
      </ScrollArea>

      <div className="space-y-2">
        <label className="text-xs font-medium text-muted-foreground" htmlFor="chat-custom-notes">
          Additional notes or context
        </label>
        <Textarea
          id="chat-custom-notes"
          placeholder="Mention business rules, filters, or clarification for the assistant"
          value={customNotes}
          onChange={event => setCustomNotes(event.target.value)}
          className="min-h-[80px]"
        />
      </div>
    </div>
  )
}
