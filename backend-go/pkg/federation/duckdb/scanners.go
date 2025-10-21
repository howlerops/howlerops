package duckdb

import (
	"fmt"
	"strings"
)

// ScannerConfig represents configuration for database scanners
type ScannerConfig struct {
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	SSLMode  string            `json:"sslMode"`
	Options  map[string]string `json:"options"`
}

// ScannerBuilder builds scanner function calls for different database types
type ScannerBuilder struct{}

// NewScannerBuilder creates a new scanner builder
func NewScannerBuilder() *ScannerBuilder {
	return &ScannerBuilder{}
}

// BuildPostgresScanner builds a postgres_scanner function call
func (sb *ScannerBuilder) BuildPostgresScanner(config *ScannerConfig, schema, table string) string {
	// Build connection string
	connStr := sb.buildPostgresConnectionString(config)
	
	// Build scanner call
	return fmt.Sprintf("postgres_scan('%s', '%s', '%s')", connStr, schema, table)
}

// BuildMySQLScanner builds a mysql_scanner function call
func (sb *ScannerBuilder) BuildMySQLScanner(config *ScannerConfig, schema, table string) string {
	// Build connection string
	connStr := sb.buildMySQLConnectionString(config)
	
	// Build scanner call
	return fmt.Sprintf("mysql_scan('%s', '%s', '%s')", connStr, schema, table)
}

// BuildSQLiteScanner builds a sqlite_scanner function call
func (sb *ScannerBuilder) BuildSQLiteScanner(config *ScannerConfig, schema, table string) string {
	// For SQLite, the database path is the file path
	databasePath := config.Database
	
	// Build scanner call
	return fmt.Sprintf("sqlite_scan('%s', '%s', '%s')", databasePath, schema, table)
}

// BuildScannerCall builds the appropriate scanner call based on connection type
func (sb *ScannerBuilder) BuildScannerCall(config *ScannerConfig, schema, table string) (string, error) {
	switch strings.ToLower(config.Type) {
	case "postgres", "postgresql":
		return sb.BuildPostgresScanner(config, schema, table), nil
	case "mysql", "mariadb":
		return sb.BuildMySQLScanner(config, schema, table), nil
	case "sqlite", "sqlite3":
		return sb.BuildSQLiteScanner(config, schema, table), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// buildPostgresConnectionString builds a PostgreSQL connection string
func (sb *ScannerBuilder) buildPostgresConnectionString(config *ScannerConfig) string {
	var parts []string
	
	if config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	}
	if config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", config.Database))
	}
	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
	}
	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}
	if config.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", config.SSLMode))
	}
	
	// Add custom options
	for key, value := range config.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	
	return strings.Join(parts, " ")
}

// buildMySQLConnectionString builds a MySQL connection string
func (sb *ScannerBuilder) buildMySQLConnectionString(config *ScannerConfig) string {
	var parts []string
	
	if config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	}
	if config.Database != "" {
		parts = append(parts, fmt.Sprintf("database=%s", config.Database))
	}
	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
	}
	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}
	
	// Add custom options
	for key, value := range config.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	
	return strings.Join(parts, " ")
}

// ValidateScannerConfig validates scanner configuration
func (sb *ScannerBuilder) ValidateScannerConfig(config *ScannerConfig) error {
	if config.Type == "" {
		return fmt.Errorf("database type is required")
	}
	
	switch strings.ToLower(config.Type) {
	case "postgres", "postgresql":
		return sb.validatePostgresConfig(config)
	case "mysql", "mariadb":
		return sb.validateMySQLConfig(config)
	case "sqlite", "sqlite3":
		return sb.validateSQLiteConfig(config)
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// validatePostgresConfig validates PostgreSQL configuration
func (sb *ScannerBuilder) validatePostgresConfig(config *ScannerConfig) error {
	if config.Host == "" {
		return fmt.Errorf("host is required for PostgreSQL")
	}
	if config.Database == "" {
		return fmt.Errorf("database is required for PostgreSQL")
	}
	if config.Username == "" {
		return fmt.Errorf("username is required for PostgreSQL")
	}
	return nil
}

// validateMySQLConfig validates MySQL configuration
func (sb *ScannerBuilder) validateMySQLConfig(config *ScannerConfig) error {
	if config.Host == "" {
		return fmt.Errorf("host is required for MySQL")
	}
	if config.Database == "" {
		return fmt.Errorf("database is required for MySQL")
	}
	if config.Username == "" {
		return fmt.Errorf("username is required for MySQL")
	}
	return nil
}

// validateSQLiteConfig validates SQLite configuration
func (sb *ScannerBuilder) validateSQLiteConfig(config *ScannerConfig) error {
	if config.Database == "" {
		return fmt.Errorf("database path is required for SQLite")
	}
	return nil
}
