package services

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/config"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/pkg/database"
)

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
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
