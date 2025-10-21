package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Engine manages the embedded DuckDB instance for federated queries
type Engine struct {
	db     *sql.DB
	logger *logrus.Logger
	mu     sync.RWMutex

	// Connection manager for resolving DSNs
	connectionManager interface{}

	// Extensions loaded
	extensions map[string]bool
}

// NewEngine creates a new DuckDB federation engine
func NewEngine(logger *logrus.Logger, connectionManager interface{}) *Engine {
	return &Engine{
		logger:            logger,
		connectionManager: connectionManager,
		extensions:        make(map[string]bool),
	}
}

// Initialize sets up the DuckDB instance and loads required extensions
func (e *Engine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db != nil {
		return nil // Already initialized
	}

	// Open DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping DuckDB: %w", err)
	}

	e.db = db

	// Load required extensions
	if err := e.loadExtensions(ctx); err != nil {
		e.logger.WithError(err).Warn("Failed to load some DuckDB extensions")
		// Continue anyway - some extensions might not be available
	}

	e.logger.Info("DuckDB federation engine initialized")
	return nil
}

// loadExtensions loads required DuckDB extensions for database scanners
func (e *Engine) loadExtensions(ctx context.Context) error {
	extensions := []string{
		"postgres_scanner",
		"mysql_scanner",
		"sqlite_scanner",
		"httpfs",
	}

	for _, ext := range extensions {
		if err := e.loadExtension(ctx, ext); err != nil {
			e.logger.WithField("extension", ext).WithError(err).Warn("Failed to load extension")
			continue
		}
		e.extensions[ext] = true
	}

	return nil
}

// loadExtension loads a specific DuckDB extension
func (e *Engine) loadExtension(ctx context.Context, extension string) error {
	query := fmt.Sprintf("INSTALL %s", extension)
	if _, err := e.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to install extension %s: %w", extension, err)
	}

	query = fmt.Sprintf("LOAD %s", extension)
	if _, err := e.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to load extension %s: %w", extension, err)
	}

	e.logger.WithField("extension", extension).Debug("Loaded DuckDB extension")
	return nil
}

// CreateView creates a temporary view in DuckDB from compiled SQL
func (e *Engine) CreateView(ctx context.Context, viewName, compiledSQL string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.db == nil {
		return fmt.Errorf("engine not initialized")
	}

	// Drop existing view if it exists
	dropQuery := fmt.Sprintf("DROP VIEW IF EXISTS %s", viewName)
	e.db.ExecContext(ctx, dropQuery) // Ignore error if view doesn't exist

	// Create the view
	createQuery := fmt.Sprintf("CREATE TEMP VIEW %s AS %s", viewName, compiledSQL)
	if _, err := e.db.ExecContext(ctx, createQuery); err != nil {
		return fmt.Errorf("failed to create view %s: %w", viewName, err)
	}

	e.logger.WithField("view", viewName).Debug("Created DuckDB view")
	return nil
}

// ExecuteQuery executes a query against the DuckDB instance
func (e *Engine) ExecuteQuery(ctx context.Context, query string, timeout time.Duration) (*QueryResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.db == nil {
		return nil, fmt.Errorf("engine not initialized")
	}

	// Validate query for safety
	if err := e.validateQuery(query); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Create timeout context
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Execute query
	rows, err := e.db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan rows
	var resultRows [][]interface{}
	rowCount := 0

	for rows.Next() {
		// Create slice for row values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		resultRows = append(resultRows, values)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	duration := time.Since(start)

	return &QueryResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: rowCount,
		Duration: duration,
	}, nil
}

// validateQuery validates a query for safety and performance
func (e *Engine) validateQuery(query string) error {
	queryLower := strings.ToLower(strings.TrimSpace(query))

	// Check for DML operations
	if strings.HasPrefix(queryLower, "insert") ||
		strings.HasPrefix(queryLower, "update") ||
		strings.HasPrefix(queryLower, "delete") ||
		strings.HasPrefix(queryLower, "create") ||
		strings.HasPrefix(queryLower, "alter") ||
		strings.HasPrefix(queryLower, "drop") {
		return fmt.Errorf("DML/DDL operations are not allowed on synthetic views")
	}

	// Check for potentially expensive operations
	if strings.Contains(queryLower, "cross join") {
		return fmt.Errorf("CROSS JOIN operations are not allowed - use explicit JOIN conditions")
	}

	// Check for missing WHERE clause on large tables (basic heuristic)
	if !strings.Contains(queryLower, "where") &&
		(strings.Contains(queryLower, "select *") || strings.Contains(queryLower, "select count(*)")) {
		e.logger.Warn("Query without WHERE clause may be expensive")
	}

	return nil
}

// SetRowLimit sets a row limit for queries
func (e *Engine) SetRowLimit(limit int) {
	// This would be used to automatically add LIMIT clauses to queries
	// Implementation would depend on the specific requirements
}

// Close closes the DuckDB connection
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db != nil {
		err := e.db.Close()
		e.db = nil
		return err
	}

	return nil
}

// IsInitialized returns true if the engine is ready
func (e *Engine) IsInitialized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.db != nil
}

// QueryResult represents the result of a DuckDB query
type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	RowCount int
	Duration time.Duration
}
