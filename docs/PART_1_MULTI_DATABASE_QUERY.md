# Part 1: Multi-Database Query Support Implementation

## Executive Summary

This document outlines the implementation plan for adding multi-database query support to HowlerOps Howlerops, enabling users to query across multiple database connections in a single query with full autocomplete, visualization, and schema management support.

## Core Concept

Enable queries like:
```sql
SELECT u.name, o.total, p.amount
FROM @users_db.public.users u
JOIN @orders_db.sales.orders o ON u.id = o.user_id
JOIN @payments_db.transactions p ON o.id = p.order_id
WHERE u.created_at > '2024-01-01'
```

## Implementation Approach

### 1. Query Syntax Design

#### Connection Reference Format
```sql
-- Full syntax: @connection_alias.schema.table
@production.public.users

-- Default schema: @connection_alias.table
@production.users

-- With table alias
@production.users AS u

-- Complex example with multiple databases
SELECT
    prod.user_id,
    prod.total_orders,
    staging.test_flag
FROM @production.order_stats prod
LEFT JOIN @staging.test_users staging
    ON prod.user_id = staging.user_id
WHERE prod.created_date >= CURRENT_DATE - 30
```

#### Why This Syntax?
- **@ prefix**: Clearly distinguishes connection references from regular identifiers
- **Backward compatible**: Queries without @ work as before
- **Intuitive**: Similar to email/social media conventions
- **Non-conflicting**: @ is rarely used in SQL

### 2. HowlerOps Architecture Integration

#### Overview

HowlerOps follows a layered architecture pattern:
- **Core Logic**: `backend-go/pkg/database/` - Reusable database operations
- **Internal Services**: `backend-go/internal/` - AI, RAG, business logic
- **Service Wrappers**: `services/` - Thin Wails integration layer
- **App Layer**: `app.go` - Exported methods for frontend
- **Frontend**: `frontend/src/` - React/TypeScript with Wails bindings

#### Integration Points

Multi-database query support will integrate with:
1. Existing `database.Manager` (`backend-go/pkg/database/manager.go`)
2. Existing query parser utilities (`backend-go/pkg/database/queryparser.go`)
3. Database service wrapper (`services/database.go`)
4. Wails app methods (`app.go`)
5. Frontend query editor (`frontend/src/components/query-editor.tsx`)

### 3. Backend Architecture

#### A. Multi-Query Parser Package

**Location**: `backend-go/pkg/database/multiquery/`

Create a new package alongside existing database packages to handle multi-database query parsing and execution.

**Files to Create**:
- `parser.go` - Parse @connection syntax
- `executor.go` - Cross-database execution
- `merger.go` - Result set merging
- `types.go` - Multi-query types
- `validator.go` - Query validation
- `parser_test.go` - Unit tests

```go
// backend-go/pkg/database/multiquery/parser.go
package multiquery

type QueryParser struct {
    connections map[string]*ConnectionInfo
}

type ParsedMultiQuery struct {
    OriginalSQL     string
    QuerySegments   []QuerySegment
    RequiredConns   []string
    ExecutionPlan   *ExecutionPlan
}

type QuerySegment struct {
    ConnectionID    string
    SQL            string
    Tables         []TableRef
    OutputColumns  []ColumnDef
}

func (p *QueryParser) Parse(sql string) (*ParsedMultiQuery, error) {
    // Step 1: Tokenize and find @connection references
    // Step 2: Split query into connection-specific segments
    // Step 3: Validate all connections exist
    // Step 4: Generate execution plan
}
```

#### B. Cross-Database Executor

```go
// backend-go/pkg/database/multiquery/executor.go
package multiquery

type CrossDBExecutor struct {
    connManager  *ConnectionManager
    tempStore    *TempStorage
    resultMerger *ResultMerger
}

type ExecutionStrategy int
const (
    FEDERATED   ExecutionStrategy = iota // Execute in parts, merge results
    PUSH_DOWN                            // Push to capable database
    MATERIALIZE                          // Create temp tables
)

func (e *CrossDBExecutor) Execute(parsed *ParsedMultiQuery) (*QueryResult, error) {
    // Step 1: Acquire connections
    // Step 2: Execute segments based on strategy
    // Step 3: Merge results
    // Step 4: Return unified result set
}
```

#### C. Extend Existing Database Manager

**Location**: `backend-go/pkg/database/manager.go`

Add multi-database query methods to the existing `Manager` struct:

```go
// backend-go/pkg/database/manager.go
package database

import (
    "github.com/sql-studio/backend-go/pkg/database/multiquery"
)

// Add to Manager struct:
type Manager struct {
    connections    map[string]Database
    mu             sync.RWMutex
    logger         *logrus.Logger
    multiQueryParser *multiquery.QueryParser  // NEW
    multiQueryExec   *multiquery.Executor     // NEW
}

// Add multi-query methods to Manager:

// ExecuteMultiQuery executes a query spanning multiple connections
func (m *Manager) ExecuteMultiQuery(ctx context.Context, query string, options *multiquery.Options) (*multiquery.Result, error) {
    // Parse query to identify connections
    parsed, err := m.multiQueryParser.Parse(query)
    if err != nil {
        return nil, fmt.Errorf("failed to parse multi-query: %w", err)
    }

    // Validate all connections exist
    if err := m.validateConnections(parsed.RequiredConnections); err != nil {
        return nil, err
    }

    // Execute using appropriate strategy
    result, err := m.multiQueryExec.Execute(ctx, parsed, m.connections, options)
    if err != nil {
        return nil, fmt.Errorf("failed to execute multi-query: %w", err)
    }

    m.logger.WithFields(logrus.Fields{
        "connections": parsed.RequiredConnections,
        "duration":    result.Duration,
        "row_count":   result.RowCount,
    }).Info("Multi-query executed successfully")

    return result, nil
}

// ParseMultiQuery parses a query to identify connections without executing
func (m *Manager) ParseMultiQuery(query string) (*multiquery.ParsedQuery, error) {
    return m.multiQueryParser.Parse(query)
}

// ValidateMultiQuery validates a parsed multi-query
func (m *Manager) ValidateMultiQuery(parsed *multiquery.ParsedQuery) error {
    return m.validateConnections(parsed.RequiredConnections)
}

// GetMultiConnectionSchema returns combined schema for multiple connections
func (m *Manager) GetMultiConnectionSchema(ctx context.Context, connectionIDs []string) (*multiquery.CombinedSchema, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    combined := &multiquery.CombinedSchema{
        Connections: make(map[string]*multiquery.ConnectionSchema),
        Conflicts:   []multiquery.SchemaConflict{},
    }

    // Fetch schema for each connection
    for _, connID := range connectionIDs {
        db, exists := m.connections[connID]
        if !exists {
            return nil, fmt.Errorf("connection not found: %s", connID)
        }

        schemas, err := db.GetSchemas(ctx)
        if err != nil {
            m.logger.WithError(err).Warnf("Failed to get schemas for connection %s", connID)
            continue
        }

        connSchema := &multiquery.ConnectionSchema{
            ConnectionID: connID,
            Schemas:      schemas,
        }

        // Get tables for each schema
        for _, schema := range schemas {
            tables, err := db.GetTables(ctx, schema)
            if err != nil {
                continue
            }
            connSchema.Tables = append(connSchema.Tables, tables...)
        }

        combined.Connections[connID] = connSchema
    }

    // Detect naming conflicts
    combined.Conflicts = m.detectSchemaConflicts(combined.Connections)

    return combined, nil
}

func (m *Manager) validateConnections(connectionIDs []string) error {
    m.mu.RLock()
    defer m.mu.RUnlock()

    for _, connID := range connectionIDs {
        if _, exists := m.connections[connID]; !exists {
            return fmt.Errorf("connection not found: %s", connID)
        }
    }
    return nil
}

func (m *Manager) detectSchemaConflicts(schemas map[string]*multiquery.ConnectionSchema) []multiquery.SchemaConflict {
    // Track table names across connections
    tableMap := make(map[string][]multiquery.ConflictingTable)

    for connID, schema := range schemas {
        for _, table := range schema.Tables {
            key := table.Name
            tableMap[key] = append(tableMap[key], multiquery.ConflictingTable{
                ConnectionID: connID,
                TableName:    table.Name,
                Schema:       table.Schema,
            })
        }
    }

    // Identify conflicts (tables with same name in multiple connections)
    var conflicts []multiquery.SchemaConflict
    for tableName, tables := range tableMap {
        if len(tables) > 1 {
            conflicts = append(conflicts, multiquery.SchemaConflict{
                TableName:   tableName,
                Connections: tables,
                Resolution:  fmt.Sprintf("Use @connection.%s syntax to disambiguate", tableName),
            })
        }
    }

    return conflicts
}
```

#### D. Service Wrapper Layer

**Location**: `services/database.go`

Extend the existing `DatabaseService` with multi-query wrapper methods:

```go
// services/database.go
package services

import (
    "github.com/sql-studio/backend-go/pkg/database/multiquery"
)

// Add to DatabaseService:

// ExecuteMultiDatabaseQuery executes a query across multiple connections
func (s *DatabaseService) ExecuteMultiDatabaseQuery(query string, options *multiquery.Options) (*MultiQueryResponse, error) {
    // Apply default options
    if options == nil {
        options = &multiquery.Options{
            Timeout:  30 * time.Second,
            Strategy: multiquery.StrategyFederated,
        }
    }

    // Execute via manager
    result, err := s.manager.ExecuteMultiQuery(s.ctx, query, options)
    if err != nil {
        s.logger.WithError(err).Error("Multi-query execution failed")
    return &MultiQueryResponse{
            Error: err.Error(),
    }, nil
    }

    // Emit multi-query executed event
    runtime.EventsEmit(s.ctx, "multiquery:executed", map[string]interface{}{
        "connections": result.ConnectionsUsed,
        "duration":    result.Duration.String(),
        "rowCount":    result.RowCount,
    })

    return &MultiQueryResponse{
        Columns:         result.Columns,
        Rows:            result.Rows,
        RowCount:        result.RowCount,
        Duration:        result.Duration.String(),
        ConnectionsUsed: result.ConnectionsUsed,
        Strategy:        string(result.Strategy),
    }, nil
}

// ValidateMultiQuery validates a multi-database query
func (s *DatabaseService) ValidateMultiQuery(query string) (*MultiQueryValidation, error) {
    parsed, err := s.manager.ParseMultiQuery(query)
    if err != nil {
        return &MultiQueryValidation{
            Valid:  false,
            Errors: []string{err.Error()},
        }, nil
    }

    if err := s.manager.ValidateMultiQuery(parsed); err != nil {
        return &MultiQueryValidation{
            Valid:  false,
            Errors: []string{err.Error()},
        }, nil
    }

    return &MultiQueryValidation{
        Valid:               true,
        RequiredConnections: parsed.RequiredConnections,
        Tables:              parsed.Tables,
        EstimatedStrategy:   string(parsed.SuggestedStrategy),
    }, nil
}

// GetCombinedSchema returns combined schema for selected connections
func (s *DatabaseService) GetCombinedSchema(connectionIDs []string) (*CombinedSchemaResponse, error) {
    schema, err := s.manager.GetMultiConnectionSchema(s.ctx, connectionIDs)
    if err != nil {
        return nil, err
    }

    return &CombinedSchemaResponse{
        Connections: schema.Connections,
        Conflicts:   schema.Conflicts,
    }, nil
}

// Response types for service layer
type MultiQueryResponse struct {
    Columns         []string          `json:"columns"`
    Rows            [][]interface{}   `json:"rows"`
    RowCount        int64             `json:"rowCount"`
    Duration        string            `json:"duration"`
    ConnectionsUsed []string          `json:"connectionsUsed"`
    Strategy        string            `json:"strategy"`
    Error           string            `json:"error,omitempty"`
}

type MultiQueryValidation struct {
    Valid               bool     `json:"valid"`
    Errors              []string `json:"errors,omitempty"`
    RequiredConnections []string `json:"requiredConnections,omitempty"`
    Tables              []string `json:"tables,omitempty"`
    EstimatedStrategy   string   `json:"estimatedStrategy,omitempty"`
}

type CombinedSchemaResponse struct {
    Connections map[string]*multiquery.ConnectionSchema `json:"connections"`
    Conflicts   []multiquery.SchemaConflict             `json:"conflicts"`
}
```

#### E. Wails App Layer

**Location**: `app.go`

Add Wails-exported methods to expose multi-query functionality to the frontend:

```go
// app.go
package main

// Add request/response types for Wails bindings
type MultiQueryRequest struct {
    Query       string            `json:"query"`
    Limit       int               `json:"limit,omitempty"`
    Timeout     int               `json:"timeout,omitempty"` // seconds
    Strategy    string            `json:"strategy,omitempty"` // "auto", "federated", "push_down"
    Options     map[string]string `json:"options,omitempty"`
}

type MultiQueryResponse struct {
    Columns         []string          `json:"columns"`
    Rows            [][]interface{}   `json:"rows"`
    RowCount        int64             `json:"rowCount"`
    Duration        string            `json:"duration"`
    ConnectionsUsed []string          `json:"connectionsUsed"`
    Strategy        string            `json:"strategy"`
    Error           string            `json:"error,omitempty"`
}

type ValidationResult struct {
    Valid               bool     `json:"valid"`
    Errors              []string `json:"errors,omitempty"`
    RequiredConnections []string `json:"requiredConnections,omitempty"`
    Tables              []string `json:"tables,omitempty"`
    EstimatedStrategy   string   `json:"estimatedStrategy,omitempty"`
}

type CombinedSchema struct {
    Connections map[string]ConnectionSchema `json:"connections"`
    Conflicts   []SchemaConflict            `json:"conflicts"`
}

type ConnectionSchema struct {
    ConnectionID string      `json:"connectionId"`
    Name         string      `json:"name"`
    Type         string      `json:"type"`
    Schemas      []string    `json:"schemas"`
    Tables       []TableInfo `json:"tables"`
}

type SchemaConflict struct {
    TableName   string              `json:"tableName"`
    Connections []ConflictingTable  `json:"connections"`
    Resolution  string              `json:"resolution"`
}

type ConflictingTable struct {
    ConnectionID string `json:"connectionId"`
    TableName    string `json:"tableName"`
    Schema       string `json:"schema"`
}

// Wails-exported methods:

// ExecuteMultiDatabaseQuery executes a query that spans multiple database connections
func (a *App) ExecuteMultiDatabaseQuery(req MultiQueryRequest) (*MultiQueryResponse, error) {
    a.logger.WithFields(logrus.Fields{
        "query_length": len(req.Query),
        "limit":        req.Limit,
    }).Info("Executing multi-database query")

    // Parse strategy
    var strategy multiquery.ExecutionStrategy
    switch req.Strategy {
    case "federated":
        strategy = multiquery.StrategyFederated
    case "push_down":
        strategy = multiquery.StrategyPushDown
    default:
        strategy = multiquery.StrategyAuto
    }

    options := &multiquery.Options{
        Timeout:  time.Duration(req.Timeout) * time.Second,
        Strategy: strategy,
        Limit:    req.Limit,
    }

    if options.Timeout == 0 {
        options.Timeout = 30 * time.Second
    }
    if options.Limit == 0 {
        options.Limit = 1000
    }

    // Execute via database service
    result, err := a.databaseService.ExecuteMultiDatabaseQuery(req.Query, options)
    if err != nil {
        return &MultiQueryResponse{
            Error: err.Error(),
        }, nil
    }

    return result, nil
}

// ValidateMultiQuery validates a multi-database query without executing it
func (a *App) ValidateMultiQuery(query string) (*ValidationResult, error) {
    a.logger.WithField("query_length", len(query)).Debug("Validating multi-query")

    validation, err := a.databaseService.ValidateMultiQuery(query)
    if err != nil {
        return &ValidationResult{
            Valid:  false,
            Errors: []string{err.Error()},
        }, nil
    }

    return &ValidationResult{
        Valid:               validation.Valid,
        Errors:              validation.Errors,
        RequiredConnections: validation.RequiredConnections,
        Tables:              validation.Tables,
        EstimatedStrategy:   validation.EstimatedStrategy,
    }, nil
}

// GetMultiConnectionSchema returns combined schema information for multiple connections
func (a *App) GetMultiConnectionSchema(connectionIDs []string) (*CombinedSchema, error) {
    a.logger.WithField("connection_count", len(connectionIDs)).Debug("Fetching combined schema")

    schema, err := a.databaseService.GetCombinedSchema(connectionIDs)
    if err != nil {
        return nil, err
    }

    // Convert to Wails response types
    result := &CombinedSchema{
        Connections: make(map[string]ConnectionSchema),
        Conflicts:   make([]SchemaConflict, len(schema.Conflicts)),
    }

    for connID, connSchema := range schema.Connections {
        result.Connections[connID] = ConnectionSchema{
            ConnectionID: connSchema.ConnectionID,
            Schemas:      connSchema.Schemas,
            Tables:       connSchema.Tables,
        }
    }

    for i, conflict := range schema.Conflicts {
        result.Conflicts[i] = SchemaConflict{
            TableName:   conflict.TableName,
            Connections: conflict.Connections,
            Resolution:  conflict.Resolution,
        }
    }

    return result, nil
}

// ParseQueryConnections extracts connection IDs from a query without validating
func (a *App) ParseQueryConnections(query string) ([]string, error) {
    a.logger.Debug("Parsing query for connections")

    parsed, err := a.databaseService.manager.ParseMultiQuery(query)
    if err != nil {
        return nil, err
    }

    return parsed.RequiredConnections, nil
}
```

### 4. Frontend Implementation

#### A. Enhanced Query Editor Component

**Location**: `frontend/src/components/multi-db-query-editor.tsx`

Create a new component that extends the existing query editor with multi-database support:

```tsx
// frontend/src/components/multi-db-query-editor.tsx
import { useState, useEffect, useCallback } from 'react'
import { ExecuteMultiDatabaseQuery, ValidateMultiQuery, ListConnections } from '../../wailsjs/go/main/App'
import { editor } from 'monaco-editor'
import { configureMultiDBLanguage } from '../lib/monaco-multi-db'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { AlertCircle, CheckCircle, Database } from 'lucide-react'

interface MultiDBQueryEditorProps {
  onQueryResult?: (result: MultiQueryResponse) => void
}

export function MultiDBQueryEditor({ onQueryResult }: MultiDBQueryEditorProps) {
  const [query, setQuery] = useState('')
  const [activeConnections, setActiveConnections] = useState<string[]>([])
  const [selectedConnections, setSelectedConnections] = useState<string[]>([])
  const [validation, setValidation] = useState<ValidationResult | null>(null)
  const [executing, setExecuting] = useState(false)
  const [editorInstance, setEditorInstance] = useState<editor.IStandaloneCodeEditor | null>(null)

  // Load active connections on mount
  useEffect(() => {
    ListConnections().then(setActiveConnections).catch(console.error)
  }, [])

  // Validate query as user types (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      if (query.includes('@')) {
        ValidateMultiQuery(query)
          .then(setValidation)
          .catch(console.error)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [query])

  // Configure Monaco editor with multi-DB language support
  useEffect(() => {
    if (editorInstance) {
      configureMultiDBLanguage(editorInstance, activeConnections)
    }
  }, [editorInstance, activeConnections])

  const handleExecute = async () => {
    setExecuting(true)
    try {
      const result = await ExecuteMultiDatabaseQuery({
        query,
        limit: 1000,
        timeout: 30,
        strategy: 'auto',
      })

      if (result.error) {
        // Handle error
        console.error(result.error)
      } else {
        onQueryResult?.(result)
      }
    } catch (error) {
      console.error('Query execution failed:', error)
    } finally {
      setExecuting(false)
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Connection Pills Bar */}
      <div className="border-b p-3 bg-muted/30">
        <div className="flex items-center gap-2 flex-wrap">
          <Database className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Active Connections:</span>
          {activeConnections.map(connId => (
            <Badge
              key={connId}
              variant={selectedConnections.includes(connId) ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => {
                setSelectedConnections(prev =>
                  prev.includes(connId)
                    ? prev.filter(id => id !== connId)
                    : [...prev, connId]
                )
              }}
            >
              {connId}
            </Badge>
          ))}
        </div>
      </div>

      {/* Validation Status Bar */}
      {validation && query.includes('@') && (
        <div className={`px-3 py-2 flex items-center gap-2 text-sm ${
          validation.valid ? 'bg-green-500/10 text-green-700' : 'bg-red-500/10 text-red-700'
        }`}>
          {validation.valid ? (
            <>
              <CheckCircle className="h-4 w-4" />
              <span>Valid multi-database query</span>
              {validation.requiredConnections && (
                <span className="ml-2 text-xs">
                  ({validation.requiredConnections.length} connections)
                </span>
              )}
            </>
          ) : (
            <>
              <AlertCircle className="h-4 w-4" />
              <span>{validation.errors?.[0] || 'Invalid query'}</span>
            </>
          )}
        </div>
      )}

      {/* Query Editor with Monaco */}
      <div className="flex-1 relative">
        <MonacoEditor
          value={query}
          onChange={setQuery}
          onMount={setEditorInstance}
          language="sql"
          theme="vs-dark"
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            automaticLayout: true,
          }}
        />
      </div>

      {/* Toolbar */}
      <div className="border-t p-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          {validation?.estimatedStrategy && (
            <span className="text-xs text-muted-foreground">
              Strategy: {validation.estimatedStrategy}
            </span>
          )}
        </div>
        <Button
          onClick={handleExecute}
          disabled={executing || (validation && !validation.valid)}
        >
          {executing ? 'Executing...' : 'Execute Query'}
        </Button>
      </div>
    </div>
  )
}
```

#### B. Monaco Editor Multi-DB Language Support

```tsx
// frontend/src/components/MultiDatabaseQueryEditor.tsx
import { useState } from 'react'
import MonacoEditor from '@monaco-editor/react'

interface MultiQueryEditorProps {
  activeConnections: Connection[]
  onExecute: (query: string, connections: string[]) => void
}

export function MultiDatabaseQueryEditor({ activeConnections, onExecute }: MultiQueryEditorProps) {
  const [query, setQuery] = useState('')
  const [selectedConnections, setSelectedConnections] = useState<string[]>([])

  return (
    <div className="flex h-full">
      {/* Connection Sidebar */}
      <div className="w-64 border-r p-4">
        <h3 className="font-semibold mb-3">Active Connections</h3>
        <div className="space-y-2">
          {activeConnections.map(conn => (
            <ConnectionCard
              key={conn.id}
              connection={conn}
              isActive={selectedConnections.includes(conn.id)}
              onToggle={() => toggleConnection(conn.id)}
            />
          ))}
        </div>
      </div>

      {/* Query Editor */}
      <div className="flex-1 flex flex-col">
        {/* Connection Pills */}
        <div className="p-2 bg-muted/30 border-b">
          <div className="flex gap-2">
            {selectedConnections.map(connId => {
              const conn = activeConnections.find(c => c.id === connId)
              return (
                <div key={connId} className="px-3 py-1 rounded-full bg-primary/10 text-sm">
                  {conn?.name}
                </div>
              )
            })}
          </div>
        </div>

        {/* Monaco Editor with Multi-DB Support */}
        <div className="flex-1">
          <MonacoEditor
            value={query}
            onChange={setQuery}
            language="sql"
            options={{
              minimap: { enabled: false },
              fontSize: 14,
            }}
            beforeMount={configureMultiDatabaseSupport}
          />
        </div>
      </div>
    </div>
  )
}
```

#### B. Multi-Database Autocomplete Provider

```tsx
// frontend/src/services/multiDatabaseAutocomplete.ts
export class MultiDatabaseCompletionProvider {
  constructor(
    private connections: Map<string, ConnectionSchema>,
    private monaco: any
  ) {}

  provideCompletionItems(model: any, position: any) {
    const suggestions = []
    const textUntilPosition = model.getValueInRange({
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: position.lineNumber,
      endColumn: position.column,
    })

    // Check if we're after an @ symbol
    if (textUntilPosition.match(/@\w*$/)) {
      // Suggest connection names
      this.connections.forEach((schema, connId) => {
        suggestions.push({
          label: `@${schema.alias}`,
          kind: this.monaco.languages.CompletionItemKind.Module,
          insertText: `@${schema.alias}.`,
          detail: `${schema.name} (${schema.type})`,
          documentation: `Tables: ${schema.tableCount}`,
        })
      })
    }

    // Check if we're after @connection.
    const connMatch = textUntilPosition.match(/@(\w+)\.(\w*)$/)
    if (connMatch) {
      const connAlias = connMatch[1]
      const partialTable = connMatch[2] || ''

      const schema = this.getSchemaByAlias(connAlias)
      if (schema) {
        // Suggest tables from this connection
        schema.tables.forEach(table => {
          if (table.name.toLowerCase().startsWith(partialTable.toLowerCase())) {
            suggestions.push({
              label: table.name,
              kind: this.monaco.languages.CompletionItemKind.Class,
              insertText: table.name,
              detail: `Table in ${schema.name}`,
              documentation: `Columns: ${table.columns.length}`,
            })
          }
        })
      }
    }

    return { suggestions }
  }
}
```

#### C. Connection Status Indicator

```tsx
// frontend/src/components/ConnectionStatusBar.tsx
export function ConnectionStatusBar({ activeConnections }: { activeConnections: Connection[] }) {
  return (
    <div className="flex items-center gap-4 px-4 py-2 bg-muted/50 border-t">
      <div className="flex items-center gap-2">
        <Database className="h-4 w-4" />
        <span className="text-sm font-medium">Active Connections:</span>
      </div>
      <div className="flex gap-2">
        {activeConnections.map(conn => (
          <div
            key={conn.id}
            className="flex items-center gap-1 px-2 py-1 rounded bg-background"
          >
            <div
              className={`w-2 h-2 rounded-full ${
                conn.isConnected ? 'bg-green-500' : 'bg-red-500'
              }`}
            />
            <span className="text-xs">{conn.name}</span>
            <span className="text-xs text-muted-foreground">
              ({conn.type})
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
```

### 4. Schema Management

#### A. Multi-Connection Schema Service

```go
// backend-go/pkg/database/schema/multi_schema.go
package schema

type MultiSchemaService struct {
    schemas map[string]*DatabaseSchema
    mu      sync.RWMutex
}

type DatabaseSchema struct {
    ConnectionID   string        `json:"connection_id"`
    ConnectionName string        `json:"connection_name"`
    Tables        []TableSchema `json:"tables"`
    LastUpdated   time.Time     `json:"last_updated"`
}

type TableSchema struct {
    Name        string         `json:"name"`
    Schema      string         `json:"schema"`
    Columns     []ColumnSchema `json:"columns"`
    PrimaryKeys []string       `json:"primary_keys"`
    ForeignKeys []ForeignKey   `json:"foreign_keys"`
}

func (s *MultiSchemaService) GetCombinedSchema(connectionIDs []string) *CombinedSchema {
    combined := &CombinedSchema{
        Connections: make(map[string]*DatabaseSchema),
        Conflicts:   []SchemaConflict{},
    }

    for _, connID := range connectionIDs {
        if schema, exists := s.schemas[connID]; exists {
            combined.Connections[connID] = schema
        }
    }

    // Detect naming conflicts
    combined.Conflicts = s.detectConflicts(combined.Connections)

    return combined
}
```

#### B. Conflict Resolution

```go
// backend-go/pkg/database/schema/conflicts.go
type SchemaConflict struct {
    TableName    string              `json:"table_name"`
    Connections  []ConflictingTable  `json:"connections"`
    Resolution   string              `json:"resolution"`
}

type ConflictingTable struct {
    ConnectionID   string `json:"connection_id"`
    ConnectionName string `json:"connection_name"`
    FullPath       string `json:"full_path"` // @conn.schema.table
}

func (s *MultiSchemaService) detectConflicts(schemas map[string]*DatabaseSchema) []SchemaConflict {
    tableMap := make(map[string][]ConflictingTable)

    for connID, schema := range schemas {
        for _, table := range schema.Tables {
            key := table.Name
            tableMap[key] = append(tableMap[key], ConflictingTable{
                ConnectionID:   connID,
                ConnectionName: schema.ConnectionName,
                FullPath:      fmt.Sprintf("@%s.%s", schema.ConnectionName, table.Name),
            })
        }
    }

    var conflicts []SchemaConflict
    for tableName, tables := range tableMap {
        if len(tables) > 1 {
            conflicts = append(conflicts, SchemaConflict{
                TableName:   tableName,
                Connections: tables,
                Resolution:  "Use @connection prefix to disambiguate",
            })
        }
    }

    return conflicts
}
```

### 5. Configuration Management

#### Backend Configuration

**Location**: `backend-go/configs/config.yaml`

Add multi-query configuration section:

```yaml
# Multi-database query settings
multiquery:
  enabled: true
  max_concurrent_connections: 10
  default_strategy: auto  # auto, federated, push_down
  timeout: 30s
  max_result_rows: 10000
  enable_cross_type_queries: true  # Allow PostgreSQL + MySQL in same query
  
  # Performance settings
  batch_size: 1000
  merge_buffer_size: 10000
  parallel_execution: true
  
  # Security settings
  require_explicit_connections: false  # If true, @connection must be specified
  allowed_operations:
    - SELECT
    - INSERT
    - UPDATE
    - DELETE
```

#### Load Configuration in Manager

**Location**: `backend-go/pkg/database/manager.go`

```go
// backend-go/pkg/database/manager.go

// Add configuration to Manager
type Manager struct {
    connections      map[string]Database
    mu               sync.RWMutex
    logger           *logrus.Logger
    multiQueryParser *multiquery.QueryParser
    multiQueryExec   *multiquery.Executor
    config           *ManagerConfig  // NEW
}

type ManagerConfig struct {
    MultiQuery *multiquery.Config
}

// Update NewManager to accept config
func NewManager(logger *logrus.Logger, config *ManagerConfig) *Manager {
    m := &Manager{
        connections: make(map[string]Database),
        logger:      logger,
        config:      config,
    }

    // Initialize multi-query components if enabled
    if config != nil && config.MultiQuery != nil && config.MultiQuery.Enabled {
        m.multiQueryParser = multiquery.NewQueryParser(config.MultiQuery, logger)
        m.multiQueryExec = multiquery.NewExecutor(config.MultiQuery, logger)
        logger.Info("Multi-query support enabled")
    }

    return m
}
```

### 6. Testing Strategy

#### Backend Tests

**Multi-Query Parser Tests**

**Location**: `backend-go/pkg/database/multiquery/parser_test.go`

```go
package multiquery

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestParser_ParseSimpleMultiQuery(t *testing.T) {
    parser := NewQueryParser(nil, nil)
    
    query := `
        SELECT u.name, o.total
        FROM @prod.users u
        JOIN @staging.orders o ON u.id = o.user_id
    `
    
    parsed, err := parser.Parse(query)
    assert.NoError(t, err)
    assert.Len(t, parsed.RequiredConnections, 2)
    assert.Contains(t, parsed.RequiredConnections, "prod")
    assert.Contains(t, parsed.RequiredConnections, "staging")
}

func TestParser_ParseConnectionSyntax(t *testing.T) {
    tests := []struct {
        name    string
        query   string
        wantErr bool
        connections []string
    }{
        {
            name:    "Single connection",
            query:   "SELECT * FROM @db1.users",
            wantErr: false,
            connections: []string{"db1"},
        },
        {
            name:    "Multiple connections",
            query:   "SELECT * FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
            wantErr: false,
            connections: []string{"db1", "db2"},
        },
        {
            name:    "With schema",
            query:   "SELECT * FROM @db1.public.users",
            wantErr: false,
            connections: []string{"db1"},
        },
        {
            name:    "No @connections",
            query:   "SELECT * FROM users",
            wantErr: false,
            connections: []string{},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewQueryParser(nil, nil)
            parsed, err := parser.Parse(tt.query)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.ElementsMatch(t, tt.connections, parsed.RequiredConnections)
            }
        })
    }
}
```

**Multi-Query Executor Tests**

**Location**: `backend-go/pkg/database/multiquery/executor_test.go`

```go
package multiquery

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestExecutor_FederatedStrategy(t *testing.T) {
    // Test federated execution across mock databases
    mockDB1 := new(MockDatabase)
    mockDB2 := new(MockDatabase)
    
    connections := map[string]Database{
        "db1": mockDB1,
        "db2": mockDB2,
    }
    
    executor := NewExecutor(&Config{Strategy: StrategyFederated}, nil)
    
    parsed := &ParsedQuery{
        RequiredConnections: []string{"db1", "db2"},
        Segments: []QuerySegment{
            {ConnectionID: "db1", SQL: "SELECT id, name FROM users"},
            {ConnectionID: "db2", SQL: "SELECT order_id, user_id FROM orders"},
        },
    }
    
    mockDB1.On("Execute", mock.Anything, mock.Anything).Return(&QueryResult{
        Columns: []string{"id", "name"},
        Rows:    [][]interface{}{{1, "Alice"}, {2, "Bob"}},
    }, nil)
    
    mockDB2.On("Execute", mock.Anything, mock.Anything).Return(&QueryResult{
        Columns: []string{"order_id", "user_id"},
        Rows:    [][]interface{}{{101, 1}, {102, 2}},
    }, nil)
    
    result, err := executor.Execute(context.Background(), parsed, connections, nil)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

#### Frontend Tests

**Multi-DB Editor Component Tests**

**Location**: `frontend/src/__tests__/multi-db-query-editor.test.tsx`

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MultiDBQueryEditor } from '../components/multi-db-query-editor'
import { ExecuteMultiDatabaseQuery, ValidateMultiQuery } from '../../wailsjs/go/main/App'

jest.mock('../../wailsjs/go/main/App')

describe('MultiDBQueryEditor', () => {
  it('renders connection pills', async () => {
    render(<MultiDBQueryEditor />)
    
    await waitFor(() => {
      expect(screen.getByText('Active Connections:')).toBeInTheDocument()
    })
  })
  
  it('validates query on input', async () => {
    const mockValidate = ValidateMultiQuery as jest.Mock
    mockValidate.mockResolvedValue({
      valid: true,
      requiredConnections: ['db1', 'db2']
    })
    
    render(<MultiDBQueryEditor />)
    
    const editor = screen.getByRole('textbox')
    fireEvent.change(editor, { 
      target: { value: 'SELECT * FROM @db1.users JOIN @db2.orders' }
    })
    
    await waitFor(() => {
      expect(mockValidate).toHaveBeenCalled()
    })
  })
  
  it('executes multi-database query', async () => {
    const mockExecute = ExecuteMultiDatabaseQuery as jest.Mock
    mockExecute.mockResolvedValue({
      columns: ['id', 'name'],
      rows: [[1, 'test']],
      rowCount: 1
    })
    
    render(<MultiDBQueryEditor />)
    
    const executeButton = screen.getByText('Execute Query')
    fireEvent.click(executeButton)
    
    await waitFor(() => {
      expect(mockExecute).toHaveBeenCalled()
    })
  })
})
```

### 7. Dependencies

#### Backend Dependencies

**Add to `backend-go/go.mod`:**

```go
require (
    // Existing dependencies...
    
    // Multi-query dependencies
    github.com/antlr/antlr4/runtime/Go/antlr v1.4.13  // SQL parsing
    github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2  // Alternative SQL parser
)
```

**Or use existing database/sql and parsing utilities without additional dependencies.**

#### Frontend Dependencies

**Add to `frontend/package.json`:**

```json
{
  "dependencies": {
    "@monaco-editor/react": "^4.6.0",
    "monaco-editor": "^0.45.0"
  },
  "devDependencies": {
    "@testing-library/react": "^14.1.2",
    "@testing-library/user-event": "^14.5.1",
    "vitest": "^1.0.4"
  }
}
```

### 8. Implementation Checklist

#### Phase 1: Core Parser (Week 1-2)
- [ ] Create `backend-go/pkg/database/multiquery/` package
- [ ] Implement `parser.go` with @connection syntax parsing
- [ ] Implement `types.go` with core data structures
- [ ] Write parser unit tests
- [ ] Add connection extraction logic
- [ ] Validate query structure

#### Phase 2: Executor (Week 3-4)
- [ ] Implement `executor.go` with strategy selection
- [ ] Implement `merger.go` for result combining
- [ ] Add federated execution strategy
- [ ] Add auto strategy selection logic
- [ ] Write executor tests
- [ ] Benchmark performance

#### Phase 3: Manager Integration (Week 5)
- [ ] Extend `database.Manager` with multi-query methods
- [ ] Add configuration loading
- [ ] Implement schema conflict detection
- [ ] Add combined schema endpoint
- [ ] Test integration with existing manager

#### Phase 4: Service Layer (Week 6)
- [ ] Extend `services/database.go` with wrappers
- [ ] Add Wails events for multi-query operations
- [ ] Implement response type conversions
- [ ] Add service layer tests

#### Phase 5: Wails Bindings (Week 7)
- [ ] Add multi-query methods to `app.go`
- [ ] Define request/response types
- [ ] Test Wails method calls
- [ ] Generate TypeScript bindings

#### Phase 6: Frontend Components (Week 8-9)
- [ ] Create `multi-db-query-editor.tsx`
- [ ] Implement Monaco language extensions
- [ ] Add connection selector UI
- [ ] Implement validation display
- [ ] Add result visualization

#### Phase 7: Testing & Polish (Week 10)
- [ ] End-to-end testing
- [ ] Performance benchmarking
- [ ] Error handling improvements
- [ ] Documentation updates
- [ ] User acceptance testing

### 9. Wails Integration

```go
// app.go - Add new methods for multi-database support
type App struct {
    // ... existing fields
    multiQueryService *services.MultiQueryService
    schemaService     *schema.MultiSchemaService
}

// Execute multi-database query
func (a *App) ExecuteMultiDatabaseQuery(req MultiQueryRequest) (*MultiQueryResponse, error) {
    return a.multiQueryService.Execute(req)
}

// Get combined schema for multiple connections
func (a *App) GetMultiConnectionSchema(connectionIDs []string) (*schema.CombinedSchema, error) {
    return a.schemaService.GetCombinedSchema(connectionIDs)
}

// Parse and validate multi-database query
func (a *App) ValidateMultiQuery(query string) (*ValidationResult, error) {
    return a.multiQueryService.Validate(query)
}
```

### 6. Implementation Timeline

#### Week 1-2: Query Parser
- [ ] Implement @connection syntax parser
- [ ] Create query segmentation logic
- [ ] Build connection validation
- [ ] Add unit tests

#### Week 3-4: Execution Engine
- [ ] Implement cross-database executor
- [ ] Build result merger
- [ ] Add data type harmonization
- [ ] Create temp table management

#### Week 5-6: Frontend UI
- [ ] Update query editor with multi-connection support
- [ ] Add connection selector/palette
- [ ] Implement connection status indicators
- [ ] Build query validation UI

#### Week 7-8: Autocomplete & Schema
- [ ] Multi-database autocomplete provider
- [ ] Schema conflict detection
- [ ] Enhanced syntax highlighting
- [ ] Connection-aware suggestions

#### Week 9-10: Testing & Polish
- [ ] Integration testing
- [ ] Performance optimization
- [ ] Error handling improvements
- [ ] Documentation

### 7. Testing Strategy

```go
// backend-go/pkg/database/multiquery/parser_test.go
func TestMultiQueryParsing(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        expected ParsedMultiQuery
    }{
        {
            name: "Cross-database JOIN",
            query: `SELECT * FROM @prod.users u
                   JOIN @staging.orders o ON u.id = o.user_id`,
            expected: ParsedMultiQuery{
                RequiredConns: []string{"prod", "staging"},
                // ... other fields
            },
        },
    }
    // ... test implementation
}
```

### 8. Migration Strategy

1. **Backward Compatibility**: All existing single-connection queries continue to work
2. **Opt-in Feature**: Multi-database queries only activated when @ syntax is used
3. **Gradual Rollout**: Feature flag to enable/disable multi-database support
4. **Performance**: Single-connection queries maintain current performance

### 9. Key Benefits

1. **Unified View**: Query across development, staging, and production simultaneously
2. **Data Comparison**: Easy comparison between environments
3. **Migration Support**: Validate data migrations across databases
4. **Analytics**: Combine operational and analytical databases in single queries
5. **Developer Productivity**: No need to switch between connections constantly

### 10. Success Metrics

- Query execution time overhead < 10% for single-connection queries
- Multi-database query success rate > 95%
- Autocomplete suggestion accuracy > 90%
- User adoption rate > 60% within 3 months
- Support ticket reduction for cross-database queries

## Next Steps

1. Review and approve design
2. Set up development branch
3. Begin Week 1 implementation
4. Schedule weekly progress reviews

---

**Status**: Ready for Implementation
**Estimated Duration**: 10 weeks
**Priority**: High
**Dependencies**: Current single-database query system must be stable