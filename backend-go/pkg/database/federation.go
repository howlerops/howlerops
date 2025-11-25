package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/federation/duckdb"
	"github.com/sirupsen/logrus"
)

// FederationEngine provides cross-database query capabilities using DuckDB
type FederationEngine struct {
	engine   *duckdb.Engine
	compiler *duckdb.Compiler
	manager  *Manager
	logger   *logrus.Logger
	enabled  bool
}

// FederationConfig configures the federation engine
type FederationConfig struct {
	Enabled        bool          // Whether federation is enabled
	QueryTimeout   time.Duration // Maximum query execution time
	MaxRowLimit    int           // Maximum rows returned
	EnableCaching  bool          // Whether to cache compiled queries
}

// DefaultFederationConfig returns default federation configuration
func DefaultFederationConfig() *FederationConfig {
	return &FederationConfig{
		Enabled:       true,
		QueryTimeout:  30 * time.Second,
		MaxRowLimit:   10000,
		EnableCaching: true,
	}
}

// NewFederationEngine creates a new federation engine
func NewFederationEngine(manager *Manager, logger *logrus.Logger, config *FederationConfig) *FederationEngine {
	if config == nil {
		config = DefaultFederationConfig()
	}

	fe := &FederationEngine{
		manager: manager,
		logger:  logger,
		enabled: config.Enabled,
	}

	if config.Enabled {
		fe.engine = duckdb.NewEngine(logger, manager)
		fe.compiler = duckdb.NewCompiler(manager)
	}

	return fe
}

// Initialize initializes the federation engine
func (fe *FederationEngine) Initialize(ctx context.Context) error {
	if !fe.enabled || fe.engine == nil {
		fe.logger.Info("Federation engine disabled or not available")
		return nil
	}

	if err := fe.engine.Initialize(ctx); err != nil {
		fe.logger.WithError(err).Warn("Failed to initialize federation engine, cross-database queries will be limited")
		fe.enabled = false
		return nil // Don't fail - gracefully degrade
	}

	fe.logger.Info("Federation engine initialized successfully")
	return nil
}

// IsAvailable returns true if the federation engine is ready for use
func (fe *FederationEngine) IsAvailable() bool {
	return fe.enabled && fe.engine != nil && fe.engine.IsInitialized()
}

// ExecuteFederatedQuery executes a query that may span multiple databases
func (fe *FederationEngine) ExecuteFederatedQuery(ctx context.Context, viewDef *duckdb.ViewDefinition, timeout time.Duration) (*FederatedResult, error) {
	if !fe.IsAvailable() {
		return nil, fmt.Errorf("federation engine not available")
	}

	// Compile the view definition to DuckDB SQL
	sql, err := fe.compiler.Compile(viewDef)
	if err != nil {
		return nil, fmt.Errorf("failed to compile federated query: %w", err)
	}

	fe.logger.WithFields(logrus.Fields{
		"view_name": viewDef.Name,
		"sources":   len(viewDef.Sources),
	}).Debug("Executing federated query")

	// Execute the compiled query
	result, err := fe.engine.ExecuteQuery(ctx, sql, timeout)
	if err != nil {
		return nil, fmt.Errorf("federated query execution failed: %w", err)
	}

	return &FederatedResult{
		Columns:     result.Columns,
		Rows:        result.Rows,
		RowCount:    int64(result.RowCount),
		Duration:    result.Duration,
		CompiledSQL: sql,
	}, nil
}

// ExecuteRawFederatedQuery executes a raw SQL query through the federation engine
func (fe *FederationEngine) ExecuteRawFederatedQuery(ctx context.Context, sql string, timeout time.Duration) (*FederatedResult, error) {
	if !fe.IsAvailable() {
		return nil, fmt.Errorf("federation engine not available")
	}

	result, err := fe.engine.ExecuteQuery(ctx, sql, timeout)
	if err != nil {
		return nil, fmt.Errorf("federated query execution failed: %w", err)
	}

	return &FederatedResult{
		Columns:     result.Columns,
		Rows:        result.Rows,
		RowCount:    int64(result.RowCount),
		Duration:    result.Duration,
		CompiledSQL: sql,
	}, nil
}

// CreateSyntheticView creates a temporary view for cross-database querying
func (fe *FederationEngine) CreateSyntheticView(ctx context.Context, viewDef *duckdb.ViewDefinition) error {
	if !fe.IsAvailable() {
		return fmt.Errorf("federation engine not available")
	}

	// Compile the view definition
	sql, err := fe.compiler.Compile(viewDef)
	if err != nil {
		return fmt.Errorf("failed to compile view definition: %w", err)
	}

	// Create the view in DuckDB
	if err := fe.engine.CreateView(ctx, viewDef.Name, sql); err != nil {
		return fmt.Errorf("failed to create synthetic view: %w", err)
	}

	fe.logger.WithField("view_name", viewDef.Name).Info("Synthetic view created")
	return nil
}

// Close closes the federation engine
func (fe *FederationEngine) Close() error {
	if fe.engine != nil {
		return fe.engine.Close()
	}
	return nil
}

// FederatedResult represents the result of a federated query
type FederatedResult struct {
	Columns     []string
	Rows        [][]interface{}
	RowCount    int64
	Duration    time.Duration
	CompiledSQL string // The SQL that was executed
}

// ToQueryResult converts FederatedResult to the standard QueryResult format
func (fr *FederatedResult) ToQueryResult() *QueryResult {
	return &QueryResult{
		Columns:  fr.Columns,
		Rows:     fr.Rows,
		RowCount: fr.RowCount,
		Duration: fr.Duration,
	}
}
