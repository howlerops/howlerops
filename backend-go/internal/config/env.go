package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// LoadEnv loads environment variables from .env files
// Priority: .env.{ENVIRONMENT} > .env > system environment variables
func LoadEnv(logger *logrus.Logger) error {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
		if err := os.Setenv("ENVIRONMENT", env); err != nil {
			logger.WithError(err).Warn("Failed to set ENVIRONMENT variable")
		}
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try to load environment-specific file first
	envFile := filepath.Join(wd, fmt.Sprintf(".env.%s", env))
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("failed to load %s: %w", envFile, err)
		}
		if logger != nil {
			logger.WithField("file", envFile).Info("Loaded environment configuration")
		}
		return nil
	}

	// Fallback to .env
	defaultEnvFile := filepath.Join(wd, ".env")
	if _, err := os.Stat(defaultEnvFile); err == nil {
		if err := godotenv.Load(defaultEnvFile); err != nil {
			return fmt.Errorf("failed to load .env: %w", err)
		}
		if logger != nil {
			logger.WithField("file", defaultEnvFile).Info("Loaded environment configuration")
		}
		return nil
	}

	if logger != nil {
		logger.Debug("No .env file found, using system environment variables only")
	}
	return nil
}

// GetEnvString gets string env var with default
func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets int env var with default
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvDuration gets duration env var with default
// Accepts formats like "15m", "24h", "7d" (days are converted to hours)
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Handle day suffix (e.g., "7d" -> "168h")
		if len(value) > 1 && value[len(value)-1] == 'd' {
			if days, err := strconv.Atoi(value[:len(value)-1]); err == nil {
				return time.Duration(days) * 24 * time.Hour
			}
		}

		// Parse standard duration
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvBool gets bool env var with default
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
