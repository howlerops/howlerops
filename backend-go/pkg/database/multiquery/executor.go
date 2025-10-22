package multiquery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Executor executes multi-database queries
type Executor struct {
	config *Config
	logger *logrus.Logger
	merger *ResultMerger
}

// NewExecutor creates a new executor
func NewExecutor(config *Config, logger *logrus.Logger) *Executor {
	return &Executor{
		config: config,
		logger: logger,
		merger: NewResultMerger(logger),
	}
}

// Execute executes a parsed multi-database query
func (e *Executor) Execute(
	ctx context.Context,
	parsed *ParsedQuery,
	connections map[string]Database,
	options *Options,
) (*Result, error) {
	startTime := time.Now()

	// Apply default options
	if options == nil {
		options = &Options{
			Timeout:  e.config.Timeout,
			Strategy: parsed.SuggestedStrategy,
			Limit:    e.config.MaxResultRows,
		}
	}

	// Create timeout context
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// If single connection, execute directly
	if len(parsed.RequiredConnections) <= 1 {
		return e.executeSingle(ctx, parsed, connections, options)
	}

	// Execute based on strategy
	var result *Result
	var err error

	switch options.Strategy {
	case StrategyFederated:
		result, err = e.executeFederated(ctx, parsed, connections, options)
	case StrategyAuto:
		result, err = e.executeFederated(ctx, parsed, connections, options)
	default:
		result, err = e.executeFederated(ctx, parsed, connections, options)
	}

	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(startTime)
	result.ConnectionsUsed = parsed.RequiredConnections
	result.Strategy = options.Strategy

	return result, nil
}

// executeSingle executes a query on a single connection
func (e *Executor) executeSingle(
	ctx context.Context,
	parsed *ParsedQuery,
	connections map[string]Database,
	options *Options,
) (*Result, error) {
	// Get the connection (might be implicit)
	var db Database
	var connID string

	if len(parsed.RequiredConnections) == 1 {
		connID = parsed.RequiredConnections[0]
		var exists bool
		db, exists = connections[connID]
		if !exists {
			return nil, fmt.Errorf("connection not found: %s", connID)
		}
	} else {
		// No @connection specified, use first available
		for id, conn := range connections {
			db = conn
			connID = id
			break
		}
	}

	if db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// Execute the query (strip @connection prefixes if present)
	queryToExecute := parsed.OriginalSQL
	if len(parsed.Segments) > 0 {
		queryToExecute = e.replaceConnectionRefs(queryToExecute)
	}

	queryResult, err := db.Execute(ctx, queryToExecute)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return &Result{
		Columns:         queryResult.Columns,
		Rows:            queryResult.Rows,
		RowCount:        queryResult.RowCount,
		Duration:        queryResult.Duration,
		ConnectionsUsed: []string{connID},
		Strategy:        StrategyAuto,
		Editable:        queryResult.Editable,
	}, nil
}

// executeFederated executes queries across multiple connections and merges results
func (e *Executor) executeFederated(
	ctx context.Context,
	parsed *ParsedQuery,
	connections map[string]Database,
	options *Options,
) (*Result, error) {
	// Validate all connections exist
	for _, connID := range parsed.RequiredConnections {
		if _, exists := connections[connID]; !exists {
			return nil, fmt.Errorf("connection not found: %s", connID)
		}
	}

	// For now, we'll execute the full query on the first connection
	// and let it handle cross-database references (if supported)
	// In a full implementation, we would:
	// 1. Break down the query into per-connection segments
	// 2. Execute each segment in parallel
	// 3. Merge results based on JOIN conditions

	// Simple implementation: execute on first connection
	if len(parsed.RequiredConnections) > 0 {
		connID := parsed.RequiredConnections[0]
		db := connections[connID]

		// Replace @connection syntax with actual table references
		modifiedSQL := e.replaceConnectionRefs(parsed.OriginalSQL)

		queryResult, err := db.Execute(ctx, modifiedSQL)
		if err != nil {
			return nil, fmt.Errorf("federated query execution failed: %w", err)
		}

		return &Result{
			Columns:         queryResult.Columns,
			Rows:            queryResult.Rows,
			RowCount:        queryResult.RowCount,
			Duration:        queryResult.Duration,
			ConnectionsUsed: parsed.RequiredConnections,
			Strategy:        StrategyFederated,
			Editable:        queryResult.Editable,
		}, nil
	}

	return nil, fmt.Errorf("no connections specified")
}

// replaceConnectionRefs replaces @connection.table references with just table names
// This is a simplified implementation - a full version would need proper SQL parsing
func (e *Executor) replaceConnectionRefs(sql string) string {
	// Remove @connection_id. prefix, keeping schema.table or just table
	// Pattern: @connection_id.schema.table -> schema.table
	// Pattern: @connection_id.table -> table

	// For now, just remove the @connection. prefix
	// This is a simplified approach that works for basic queries
	result := sql

	// Use a simple approach: find and replace @word. patterns
	for {
		start := -1
		for i := 0; i < len(result); i++ {
			if result[i] == '@' {
				start = i
				break
			}
		}

		if start == -1 {
			break
		}

		// Find the end of the connection reference (first dot)
		end := start + 1
		for end < len(result) && result[end] != '.' {
			end++
		}

		if end >= len(result) {
			break
		}

		// Remove @connection part
		result = result[:start] + result[end+1:]
	}

	return result
}

// executeParallel executes query segments in parallel (future implementation)
func (e *Executor) executeParallel(
	ctx context.Context,
	segments []QuerySegment,
	connections map[string]Database,
) (map[string]*QueryResult, error) {
	results := make(map[string]*QueryResult)
	errors := make(chan error, len(segments))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, segment := range segments {
		wg.Add(1)
		go func(seg QuerySegment) {
			defer wg.Done()

			db, exists := connections[seg.ConnectionID]
			if !exists {
				errors <- fmt.Errorf("connection not found: %s", seg.ConnectionID)
				return
			}

			result, err := db.Execute(ctx, seg.SQL)
			if err != nil {
				errors <- fmt.Errorf("execution failed on %s: %w", seg.ConnectionID, err)
				return
			}

			mu.Lock()
			results[seg.ConnectionID] = result
			mu.Unlock()
		}(segment)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		return nil, err
	}

	return results, nil
}
