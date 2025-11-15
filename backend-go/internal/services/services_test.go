package services

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/pkg/database"
)

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func newTestConfig() Config {
	return Config{
		Database: config.DatabaseConfig{},
		Auth: config.AuthConfig{
			BcryptCost:        12,
			JWTExpiration:     time.Hour,
			RefreshExpiration: 6 * time.Hour,
			MaxLoginAttempts:  5,
			LockoutDuration:   15 * time.Minute,
		},
		Security: config.SecurityConfig{},
	}
}

func TestNewServicesReturnsErrorWhenAIConfigInvalid(t *testing.T) {
	t.Setenv("AI_DEFAULT_PROVIDER", "invalid-provider")

	logger := newSilentLogger()
	dbManager := database.NewManager(logger)
	authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)

	_, err := NewServices(
		Config{},
		dbManager,
		authMiddleware,
		logger,
	)
	if err == nil {
		t.Fatalf("expected NewServices to return error when AI config invalid")
	}
}

func TestNewServicesCreatesDependencies(t *testing.T) {
	t.Skip("TODO: Fix this test - temporarily skipped for deployment")
	t.Setenv("OPENAI_API_KEY", "test-key")

	logger := newSilentLogger()
	dbManager := database.NewManager(logger)
	authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)

	svc, err := NewServices(
		Config{
			Database: config.DatabaseConfig{},
			Auth: config.AuthConfig{
				BcryptCost:        12,
				JWTExpiration:     time.Hour,
				RefreshExpiration: 6 * time.Hour,
				MaxLoginAttempts:  5,
				LockoutDuration:   15 * time.Minute,
			},
			Security: config.SecurityConfig{},
		},
		dbManager,
		authMiddleware,
		logger,
	)
	if err != nil {
		t.Fatalf("NewServices returned error: %v", err)
	}

	if svc.Auth == nil || svc.Database == nil || svc.Query == nil || svc.AI == nil {
		t.Fatalf("expected services dependencies to be initialized: %#v", svc)
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// PHASE 1 - Quick Wins: Service Constructor Tests

func TestNewQueryService(t *testing.T) {
	t.Run("successful creation with valid inputs", func(t *testing.T) {
		logger := newSilentLogger()
		dbManager := database.NewManager(logger)

		svc := NewQueryService(dbManager, logger)

		require.NotNil(t, svc)
		assert.Equal(t, dbManager, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})

	t.Run("accepts nil dbManager without validation", func(t *testing.T) {
		logger := newSilentLogger()

		svc := NewQueryService(nil, logger)

		require.NotNil(t, svc)
		assert.Nil(t, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})
}

func TestNewTableService(t *testing.T) {
	t.Run("successful creation with valid inputs", func(t *testing.T) {
		logger := newSilentLogger()
		dbManager := database.NewManager(logger)

		svc := NewTableService(dbManager, logger)

		require.NotNil(t, svc)
		assert.Equal(t, dbManager, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})

	t.Run("accepts nil dbManager without validation", func(t *testing.T) {
		logger := newSilentLogger()

		svc := NewTableService(nil, logger)

		require.NotNil(t, svc)
		assert.Nil(t, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})
}

func TestNewHealthService(t *testing.T) {
	t.Run("successful creation with valid inputs", func(t *testing.T) {
		logger := newSilentLogger()
		dbManager := database.NewManager(logger)

		svc := NewHealthService(dbManager, logger)

		require.NotNil(t, svc)
		assert.Equal(t, dbManager, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})

	t.Run("accepts nil dbManager without validation", func(t *testing.T) {
		logger := newSilentLogger()

		svc := NewHealthService(nil, logger)

		require.NotNil(t, svc)
		assert.Nil(t, svc.dbManager)
		assert.Equal(t, logger, svc.logger)
	})
}

func TestNewRealtimeService(t *testing.T) {
	t.Run("successful initialization with logger", func(t *testing.T) {
		logger := newSilentLogger()

		svc := NewRealtimeService(logger)

		require.NotNil(t, svc)
		assert.Equal(t, logger, svc.logger)
		assert.NotNil(t, svc.subscribers)
		assert.Empty(t, svc.subscribers)
	})

	t.Run("accepts nil logger without validation", func(t *testing.T) {
		svc := NewRealtimeService(nil)

		require.NotNil(t, svc)
		assert.Nil(t, svc.logger)
		assert.NotNil(t, svc.subscribers)
	})
}

func TestRealtimeService_Close(t *testing.T) {
	t.Run("close with no subscribers returns nil", func(t *testing.T) {
		logger := newSilentLogger()
		svc := NewRealtimeService(logger)

		err := svc.Close()

		assert.NoError(t, err)
	})

	t.Run("close with subscribers closes all channels", func(t *testing.T) {
		logger := newSilentLogger()
		svc := NewRealtimeService(logger)

		// Add three subscribers
		svc.subscribers["sub1"] = make(chan interface{}, 1)
		svc.subscribers["sub2"] = make(chan interface{}, 1)
		svc.subscribers["sub3"] = make(chan interface{}, 1)

		err := svc.Close()

		assert.NoError(t, err)

		// Verify all channels are closed by attempting to receive
		for id, ch := range svc.subscribers {
			_, ok := <-ch
			assert.False(t, ok, "channel %s should be closed", id)
		}
	})

	t.Run("verify channels actually closed and reading returns not ok", func(t *testing.T) {
		logger := newSilentLogger()
		svc := NewRealtimeService(logger)

		// Create channels and store references
		ch1 := make(chan interface{}, 1)
		ch2 := make(chan interface{}, 1)
		svc.subscribers["test1"] = ch1
		svc.subscribers["test2"] = ch2

		// Send some data before closing
		ch1 <- "data1"
		ch2 <- "data2"

		err := svc.Close()
		require.NoError(t, err)

		// First read should get the buffered data
		val1, ok1 := <-ch1
		assert.True(t, ok1)
		assert.Equal(t, "data1", val1)

		val2, ok2 := <-ch2
		assert.True(t, ok2)
		assert.Equal(t, "data2", val2)

		// Second read should indicate channel is closed
		_, ok1 = <-ch1
		assert.False(t, ok1, "channel should be closed on second read")

		_, ok2 = <-ch2
		assert.False(t, ok2, "channel should be closed on second read")
	})
}

// PHASE 2 - Core Lifecycle Tests

func TestNewServices(t *testing.T) {
	t.Run("successful initialization with valid config", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("AI_DEFAULT_PROVIDER", "openai")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)

		require.NoError(t, err)
		require.NotNil(t, svc)
	})

	t.Run("verify all service fields set correctly", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("AI_DEFAULT_PROVIDER", "openai")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)

		require.NoError(t, err)
		require.NotNil(t, svc)

		// Verify injected services are set
		assert.NotNil(t, svc.Database)
		assert.NotNil(t, svc.AI)
		assert.NotNil(t, svc.Query)
		assert.NotNil(t, svc.Table)
		assert.NotNil(t, svc.Health)
		assert.NotNil(t, svc.Realtime)

		// Verify services have correct references
		assert.Equal(t, dbManager, svc.Database)
		assert.Equal(t, dbManager, svc.Query.dbManager)
		assert.Equal(t, dbManager, svc.Table.dbManager)
		assert.Equal(t, dbManager, svc.Health.dbManager)
		assert.Equal(t, logger, svc.Realtime.logger)
	})

	t.Run("verify Auth Sync Org are nil injected later by main", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("AI_DEFAULT_PROVIDER", "openai")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)

		require.NoError(t, err)
		require.NotNil(t, svc)

		// These are intentionally nil, injected by main.go after Turso initialization
		assert.Nil(t, svc.Auth)
		assert.Nil(t, svc.Sync)
		assert.Nil(t, svc.Organization)
	})

	t.Run("AI config error with invalid provider", func(t *testing.T) {
		t.Setenv("AI_DEFAULT_PROVIDER", "invalid-provider")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)

		assert.Error(t, err)
		assert.Nil(t, svc)
	})
}

func TestServices_Close(t *testing.T) {
	t.Run("successful close with no errors", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("AI_DEFAULT_PROVIDER", "openai")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)
		require.NoError(t, err)

		ctx := context.Background()
		err = svc.Close(ctx)

		assert.NoError(t, err)
	})

	t.Run("close with subscribers in realtime service", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("AI_DEFAULT_PROVIDER", "openai")

		logger := newSilentLogger()
		dbManager := database.NewManager(logger)
		authMiddleware := middleware.NewAuthMiddleware("test-secret", logger)
		cfg := newTestConfig()

		svc, err := NewServices(cfg, dbManager, authMiddleware, logger)
		require.NoError(t, err)

		// Add subscribers to realtime service
		svc.Realtime.subscribers["sub1"] = make(chan interface{}, 1)
		svc.Realtime.subscribers["sub2"] = make(chan interface{}, 1)

		ctx := context.Background()
		err = svc.Close(ctx)

		assert.NoError(t, err)

		// Verify realtime channels were closed
		for _, ch := range svc.Realtime.subscribers {
			_, ok := <-ch
			assert.False(t, ok)
		}
	})
}
