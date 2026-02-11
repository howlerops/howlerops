//go:build !duckdb

package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDuckDBStub_IsDisabledWithoutBuildTag tests that DuckDB is properly stubbed when build tag is absent
func TestDuckDBStub_IsDisabledWithoutBuildTag(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	require.NotNil(t, engine)

	ctx := context.Background()
	err := engine.Initialize(ctx)
	assert.Error(t, err, "Should error when duckdb build tag is not set")
	assert.Contains(t, err.Error(), "disabled")
}

// TestDuckDBStub_ExecuteQueryFails tests that query execution fails gracefully
func TestDuckDBStub_ExecuteQueryFails(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	ctx := context.Background()

	result, err := engine.ExecuteQuery(ctx, "SELECT 1", 5*time.Second)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDuckDBStub_IsNotInitialized tests that engine reports as not initialized
func TestDuckDBStub_IsNotInitialized(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	assert.False(t, engine.IsInitialized())
}

// TestDuckDBStub_CloseIsNoOp tests that Close() is a no-op in stub
func TestDuckDBStub_CloseIsNoOp(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Close()
	assert.NoError(t, err, "Close should not error in stub")
}

// TestDuckDBStub_AllOperationsFail tests that all operations fail consistently
func TestDuckDBStub_AllOperationsFail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	ctx := context.Background()

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "Initialize",
			test: func() error {
				return engine.Initialize(ctx)
			},
		},
		{
			name: "ExecuteQuery",
			test: func() error {
				_, err := engine.ExecuteQuery(ctx, "SELECT 1", time.Second)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			assert.Error(t, err, "Operation should fail: %s", tt.name)
		})
	}
}

// TestDuckDBStub_ProvidesHelpfulErrors tests that errors are informative
func TestDuckDBStub_ProvidersHelpfulErrors(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	ctx := context.Background()

	err := engine.Initialize(ctx)
	require.Error(t, err)

	// Error should mention that build tag is required
	assert.Contains(t, err.Error(), "disabled", "Error should indicate feature is disabled")
}

// TestDuckDB_FallbackBehavior tests that queries fall back gracefully when DuckDB is unavailable
func TestDuckDB_FallbackBehavior(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)

	// Attempting to use engine should indicate it's not available
	ctx := context.Background()

	err := engine.Initialize(ctx)
	assert.Error(t, err)

	// After failed init, operations should still fail
	result, err := engine.ExecuteQuery(ctx, "SELECT 1", 5*time.Second)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDuckDB_SafeMultipleCalls tests that multiple operations don't cause panics
func TestDuckDB_SafeMultipleCalls(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	ctx := context.Background()

	// Multiple initialization attempts should not panic
	for i := 0; i < 5; i++ {
		engine.Initialize(ctx)
	}

	// Multiple close calls should not panic
	for i := 0; i < 3; i++ {
		engine.Close()
	}

	// Multiple query attempts should not panic
	for i := 0; i < 3; i++ {
		engine.ExecuteQuery(ctx, "SELECT 1", 5*time.Second)
	}

	assert.False(t, engine.IsInitialized())
}
