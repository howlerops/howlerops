//go:build !duckdb

package duckdb

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Engine manages the embedded DuckDB instance for federated queries
type Engine struct {
	logger *logrus.Logger
}

// NewEngine creates a new DuckDB federation engine
func NewEngine(logger *logrus.Logger, connectionManager interface{}) *Engine {
	return &Engine{
		logger: logger,
	}
}

// Initialize sets up the DuckDB instance and loads required extensions
func (e *Engine) Initialize(ctx context.Context) error {
	e.logger.Warn("DuckDB federation engine is disabled (build tag 'duckdb' not set)")
	return fmt.Errorf("DuckDB federation engine is disabled")
}

// ExecuteQuery executes a query against the DuckDB instance
func (e *Engine) ExecuteQuery(ctx context.Context, query string, timeout time.Duration) (*QueryResult, error) {
	return nil, fmt.Errorf("DuckDB federation engine is disabled")
}

// Close closes the DuckDB connection
func (e *Engine) Close() error {
	return nil
}

// IsInitialized returns true if the engine is ready
func (e *Engine) IsInitialized() bool {
	return false
}

// QueryResult represents the result of a DuckDB query
type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	RowCount int
	Duration time.Duration
}
