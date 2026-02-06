package database

import (
	"context"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/pkg/federation/duckdb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFederationEngine(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)

	t.Run("creates engine with default config", func(t *testing.T) {
		fe := NewFederationEngine(manager, logger, nil)
		assert.NotNil(t, fe)
		assert.NotNil(t, fe.engine)
		assert.NotNil(t, fe.compiler)
		assert.True(t, fe.enabled)
	})

	t.Run("creates disabled engine when disabled in config", func(t *testing.T) {
		config := &FederationConfig{
			Enabled: false,
		}
		fe := NewFederationEngine(manager, logger, config)
		assert.NotNil(t, fe)
		assert.False(t, fe.enabled)
		assert.Nil(t, fe.engine)
	})

	t.Run("creates engine with custom config", func(t *testing.T) {
		config := &FederationConfig{
			Enabled:       true,
			QueryTimeout:  60 * time.Second,
			MaxRowLimit:   5000,
			EnableCaching: false,
		}
		fe := NewFederationEngine(manager, logger, config)
		assert.NotNil(t, fe)
		assert.True(t, fe.enabled)
	})
}

func TestDefaultFederationConfig(t *testing.T) {
	config := DefaultFederationConfig()
	assert.True(t, config.Enabled)
	assert.Equal(t, 30*time.Second, config.QueryTimeout)
	assert.Equal(t, 10000, config.MaxRowLimit)
	assert.True(t, config.EnableCaching)
}

func TestFederationEngine_Initialize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)
	ctx := context.Background()

	t.Run("gracefully handles initialization failure", func(t *testing.T) {
		// Without the duckdb build tag, initialization will fail but shouldn't error
		fe := NewFederationEngine(manager, logger, nil)
		err := fe.Initialize(ctx)
		// Should not return error even if DuckDB is not available
		assert.NoError(t, err)
	})

	t.Run("handles disabled engine", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)
		err := fe.Initialize(ctx)
		assert.NoError(t, err)
		assert.False(t, fe.IsAvailable())
	})
}

func TestFederationEngine_IsAvailable(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)

	t.Run("returns false when disabled", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)
		assert.False(t, fe.IsAvailable())
	})

	t.Run("returns false before initialization", func(t *testing.T) {
		fe := NewFederationEngine(manager, logger, nil)
		// Without duckdb build tag, engine won't be fully initialized
		assert.False(t, fe.IsAvailable())
	})
}

func TestFederationEngine_ExecuteFederatedQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)
	ctx := context.Background()

	t.Run("returns error when not available", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)

		viewDef := &duckdb.ViewDefinition{
			Name: "test_view",
		}

		result, err := fe.ExecuteFederatedQuery(ctx, viewDef, 10*time.Second)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "federation engine not available")
	})
}

func TestFederationEngine_ExecuteRawFederatedQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)
	ctx := context.Background()

	t.Run("returns error when not available", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)

		result, err := fe.ExecuteRawFederatedQuery(ctx, "SELECT 1", 10*time.Second)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "federation engine not available")
	})
}

func TestFederationEngine_CreateSyntheticView(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)
	ctx := context.Background()

	t.Run("returns error when not available", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)

		viewDef := &duckdb.ViewDefinition{
			Name: "test_view",
		}

		err := fe.CreateSyntheticView(ctx, viewDef)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "federation engine not available")
	})
}

func TestFederationEngine_Close(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)

	t.Run("closes without error when disabled", func(t *testing.T) {
		config := &FederationConfig{Enabled: false}
		fe := NewFederationEngine(manager, logger, config)
		err := fe.Close()
		assert.NoError(t, err)
	})

	t.Run("closes without error when enabled", func(t *testing.T) {
		fe := NewFederationEngine(manager, logger, nil)
		err := fe.Close()
		assert.NoError(t, err)
	})
}

func TestFederatedResult_ToQueryResult(t *testing.T) {
	t.Run("converts all fields correctly", func(t *testing.T) {
		fr := &FederatedResult{
			Columns:     []string{"id", "name"},
			Rows:        [][]interface{}{{1, "test"}},
			RowCount:    1,
			Duration:    100 * time.Millisecond,
			CompiledSQL: "SELECT id, name FROM users",
		}

		qr := fr.ToQueryResult()
		require.NotNil(t, qr)
		assert.Equal(t, fr.Columns, qr.Columns)
		assert.Equal(t, fr.Rows, qr.Rows)
		assert.Equal(t, fr.RowCount, qr.RowCount)
		assert.Equal(t, fr.Duration, qr.Duration)
	})
}
