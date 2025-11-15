package config_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/jbeck018/howlerops/backend-go/internal/config"
)

func resetViper() {
	viper.Reset()
}

func TestLoadReadsConfigFileDefaults(t *testing.T) {
	t.Setenv("SQL_STUDIO_CONFIG_PATH", "")
	resetViper()
	t.Setenv("SQL_STUDIO_CONFIG_FILE", filepath.Join("..", "..", "configs", "config.yaml"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 8500 {
		t.Fatalf("expected server port 8500 from config file, got %d", cfg.Server.Port)
	}
	if cfg.Database.StreamingBatchSize != 1000 {
		t.Fatalf("expected streaming batch size 1000, got %d", cfg.Database.StreamingBatchSize)
	}
	if cfg.Auth.BcryptCost != 12 {
		t.Fatalf("expected bcrypt cost 12, got %d", cfg.Auth.BcryptCost)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	resetViper()
	t.Setenv("SQL_STUDIO_CONFIG_FILE", filepath.Join("..", "..", "configs", "config.yaml"))
	t.Setenv("SQL_STUDIO_SERVER_PORT", "9100")
	t.Setenv("SQL_STUDIO_DATABASE_STREAMING_BATCH_SIZE", "2048")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 9100 {
		t.Fatalf("expected server port override 9100, got %d", cfg.Server.Port)
	}
	if cfg.Database.StreamingBatchSize != 2048 {
		t.Fatalf("expected streaming batch size override 2048, got %d", cfg.Database.StreamingBatchSize)
	}
}

func TestLoadFailsOnInvalidConfiguration(t *testing.T) {
	resetViper()
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  port: \"invalid\"\n"), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	_, err = config.Load()
	if err == nil {
		t.Fatalf("expected Load to return error for invalid config")
	}
}

func init() {
	// Silence logrus output during tests that load configuration.
	logrus.StandardLogger().SetOutput(io.Discard)
}
