package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jbeck018/howlerops/backend-go/internal/config"
	"github.com/jbeck018/howlerops/backend-go/pkg/logger"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create logger
	appLogger, err := logger.NewLogger(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		File:       cfg.Log.File,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	appLogger.WithFields(logrus.Fields{
		"environment": cfg.GetEnv(),
		"database":    cfg.Turso.URL,
	}).Info("Starting database migration")

	// Create database directory if using local file
	if dirErr := ensureDatabaseDirectory(cfg.Turso.URL, appLogger); dirErr != nil {
		appLogger.WithError(dirErr).Fatal("Failed to create database directory")
	}

	// Connect to database
	db, err := turso.NewClient(&turso.Config{
		URL:       cfg.Turso.URL,
		AuthToken: cfg.Turso.AuthToken,
		MaxConns:  5,
	}, appLogger)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			appLogger.WithError(err).Error("Failed to close database")
		}
	}()

	// Run migrations
	appLogger.Info("Running database migrations...")
	if err := turso.InitializeSchema(db, appLogger); err != nil {
		appLogger.WithError(err).Fatal("Failed to run migrations")
	}

	// Verify schema
	ctx := context.Background()
	if err := verifySchema(ctx, db, appLogger); err != nil {
		appLogger.WithError(err).Fatal("Schema verification failed")
	}

	appLogger.Info("Migration completed successfully")
}

// ensureDatabaseDirectory creates the database directory if using a local file
func ensureDatabaseDirectory(dbURL string, logger *logrus.Logger) error {
	// Check if this is a local file path
	if len(dbURL) > 5 && dbURL[:5] == "file:" {
		// Extract the directory path
		filePath := dbURL[5:] // Remove "file:" prefix

		// Get directory from file path
		dir := filePath
		for i := len(filePath) - 1; i >= 0; i-- {
			if filePath[i] == '/' {
				dir = filePath[:i]
				break
			}
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}

		logger.WithField("directory", dir).Debug("Database directory ensured")
	}

	return nil
}

// verifySchema checks that critical tables exist
func verifySchema(ctx context.Context, db turso.DB, logger *logrus.Logger) error {
	requiredTables := []string{
		"users",
		"sessions",
		"login_attempts",
		"email_verification_tokens",
		"password_reset_tokens",
		"license_keys",
		"connection_templates",
		"saved_queries_sync",
		"query_history_sync",
		"sync_metadata",
	}

	for _, table := range requiredTables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) > 0
			FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("required table %s does not exist", table)
		}

		logger.WithField("table", table).Debug("Table verified")
	}

	logger.WithField("tables_verified", len(requiredTables)).Info("All required tables verified")
	return nil
}
