package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	Security SecurityConfig `mapstructure:"security"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Email    EmailConfig    `mapstructure:"email"`
	Sync     SyncConfig     `mapstructure:"sync"`
	Turso    TursoConfig    `mapstructure:"turso"`
	RAG      RAGConfig      `mapstructure:"rag"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	GRPCPort        int           `mapstructure:"grpc_port"`
	HTTPPort        int           `mapstructure:"http_port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	TLSEnabled      bool          `mapstructure:"tls_enabled"`
	TLSCertFile     string        `mapstructure:"tls_cert_file"`
	TLSKeyFile      string        `mapstructure:"tls_key_file"`
	Environment     string        `mapstructure:"environment"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	MaxConnections     int           `mapstructure:"max_connections"`
	MaxIdleConns       int           `mapstructure:"max_idle_connections"`
	ConnectionTimeout  time.Duration `mapstructure:"connection_timeout"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`
	ConnectionLifetime time.Duration `mapstructure:"connection_lifetime"`
	QueryTimeout       time.Duration `mapstructure:"query_timeout"`
	StreamingBatchSize int           `mapstructure:"streaming_batch_size"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret         string        `mapstructure:"jwt_secret"`
	JWTExpiration     time.Duration `mapstructure:"jwt_expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
	BcryptCost        int           `mapstructure:"bcrypt_cost"`
	SessionTimeout    time.Duration `mapstructure:"session_timeout"`
	MaxLoginAttempts  int           `mapstructure:"max_login_attempts"`
	LockoutDuration   time.Duration `mapstructure:"lockout_duration"`
	RequireStrongPass bool          `mapstructure:"require_strong_password"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	Database     int           `mapstructure:"database"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_connections"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	EnableCORS        bool          `mapstructure:"enable_cors"`
	CORSOrigins       []string      `mapstructure:"cors_origins"`
	CORSMethods       []string      `mapstructure:"cors_methods"`
	CORSHeaders       []string      `mapstructure:"cors_headers"`
	RateLimitEnabled  bool          `mapstructure:"rate_limit_enabled"`
	RateLimitRPS      int           `mapstructure:"rate_limit_rps"`
	RateLimitBurst    int           `mapstructure:"rate_limit_burst"`
	RequestTimeout    time.Duration `mapstructure:"request_timeout"`
	MaxRequestSize    int64         `mapstructure:"max_request_size"`
	EnableCompression bool          `mapstructure:"enable_compression"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider  string `mapstructure:"provider"` // resend, smtp
	APIKey    string `mapstructure:"api_key"`
	FromEmail string `mapstructure:"from_email"`
	FromName  string `mapstructure:"from_name"`
	BaseURL   string `mapstructure:"base_url"` // Base URL for email links
}

// SyncConfig holds sync service configuration
type SyncConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	MaxUploadSize      int64  `mapstructure:"max_upload_size"`   // bytes
	ConflictStrategy   string `mapstructure:"conflict_strategy"` // last_write_wins, keep_both, user_choice
	RetentionDays      int    `mapstructure:"retention_days"`
	MaxHistoryItems    int    `mapstructure:"max_history_items"`
	EnableSanitization bool   `mapstructure:"enable_sanitization"`
	RateLimitRPM       int    `mapstructure:"rate_limit_rpm"` // requests per minute
}

// TursoConfig holds Turso database configuration
type TursoConfig struct {
	URL            string `mapstructure:"url"`
	AuthToken      string `mapstructure:"auth_token"`
	MaxConnections int    `mapstructure:"max_connections"`
}

// RAGConfig holds RAG (Retrieval-Augmented Generation) configuration
type RAGConfig struct {
	Embedding EmbeddingConfig `mapstructure:"embedding"`
}

// EmbeddingConfig holds embedding provider configuration
type EmbeddingConfig struct {
	Provider string               `mapstructure:"provider"` // ollama, openai
	Ollama   OllamaEmbedConfig    `mapstructure:"ollama"`
	OpenAI   OpenAIEmbedConfig    `mapstructure:"openai"`
	Cache    EmbeddingCacheConfig `mapstructure:"cache"`
}

// OllamaEmbedConfig holds Ollama embedding configuration
type OllamaEmbedConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	Model     string `mapstructure:"model"`
	Dimension int    `mapstructure:"dimension"`
	AutoPull  bool   `mapstructure:"auto_pull"`
}

// OpenAIEmbedConfig holds OpenAI embedding configuration
type OpenAIEmbedConfig struct {
	Model     string `mapstructure:"model"`
	Dimension int    `mapstructure:"dimension"`
	APIKeyEnv string `mapstructure:"api_key_env"`
}

// EmbeddingCacheConfig holds embedding cache configuration
type EmbeddingCacheConfig struct {
	MaxSize int           `mapstructure:"max_size"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Create temporary logger for config loading
	// Note: We can't use logrus here yet as it creates a circular dependency
	// tempLogger := &logrus.Logger{
	// 	Out:       os.Stdout,
	// 	Formatter: &logrus.TextFormatter{},
	// 	Level:     logrus.InfoLevel,
	// }

	// Load environment variables from .env files
	if err := LoadEnv(nil); err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	// Set default values
	setDefaults()

	// Set up Viper
	if configFile := os.Getenv("SQL_STUDIO_CONFIG_FILE"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("/etc/sql-studio")
		viper.AddConfigPath("$HOME/.sql-studio")
	}

	// Enable environment variable support
	viper.SetEnvPrefix("SQL_STUDIO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read configuration file (optional)
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			// Real error (not just file not found), return it
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	// Unmarshal configuration
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with direct environment variables (higher priority)
	overrideFromEnv(&config)

	// Ensure critical values have defaults if empty
	config.Log.Output = strings.TrimSpace(config.Log.Output)
	if config.Log.Output == "" {
		config.Log.Output = "stdout"
	}
	config.Log.Format = strings.TrimSpace(config.Log.Format)
	if config.Log.Format == "" {
		config.Log.Format = "text"
	}
	config.Log.Level = strings.TrimSpace(config.Log.Level)
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// overrideFromEnv overrides config values with environment variables
func overrideFromEnv(config *Config) {
	// Server configuration
	config.Server.HTTPPort = GetEnvInt("SERVER_HTTP_PORT", config.Server.HTTPPort)
	config.Server.GRPCPort = GetEnvInt("SERVER_GRPC_PORT", config.Server.GRPCPort)
	config.Server.Environment = GetEnvString("ENVIRONMENT", config.Server.Environment)

	// Turso configuration
	if tursoURL := GetEnvString("TURSO_URL", ""); tursoURL != "" {
		config.Turso.URL = tursoURL
	}
	config.Turso.AuthToken = GetEnvString("TURSO_AUTH_TOKEN", config.Turso.AuthToken)

	// Email configuration
	if resendKey := GetEnvString("RESEND_API_KEY", ""); resendKey != "" {
		config.Email.APIKey = resendKey
	}
	if fromEmail := GetEnvString("RESEND_FROM_EMAIL", ""); fromEmail != "" {
		config.Email.FromEmail = fromEmail
	}

	// Auth configuration
	if jwtSecret := GetEnvString("JWT_SECRET", ""); jwtSecret != "" {
		config.Auth.JWTSecret = jwtSecret
	}
	config.Auth.JWTExpiration = GetEnvDuration("JWT_EXPIRATION", config.Auth.JWTExpiration)
	config.Auth.RefreshExpiration = GetEnvDuration("JWT_REFRESH_EXPIRATION", config.Auth.RefreshExpiration)

	// Logging configuration
	if logLevel := GetEnvString("LOG_LEVEL", ""); logLevel != "" {
		config.Log.Level = logLevel
	}
	if logFormat := GetEnvString("LOG_FORMAT", ""); logFormat != "" {
		config.Log.Format = logFormat
	}
	if logOutput := GetEnvString("LOG_OUTPUT", ""); logOutput != "" {
		config.Log.Output = logOutput
	}

	// Metrics configuration
	config.Metrics.Port = GetEnvInt("SERVER_METRICS_PORT", config.Metrics.Port)
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.shutdown_timeout", "30s")
	viper.SetDefault("server.tls_enabled", false)
	viper.SetDefault("server.environment", "development")

	// Database defaults
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_connections", 5)
	viper.SetDefault("database.connection_timeout", "30s")
	viper.SetDefault("database.idle_timeout", "5m")
	viper.SetDefault("database.connection_lifetime", "1h")
	viper.SetDefault("database.query_timeout", "30s")
	viper.SetDefault("database.streaming_batch_size", 1000)

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "change-me-in-production")
	viper.SetDefault("auth.jwt_expiration", "24h")
	viper.SetDefault("auth.refresh_expiration", "168h")
	viper.SetDefault("auth.bcrypt_cost", 12)
	viper.SetDefault("auth.session_timeout", "24h")
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.lockout_duration", "15m")
	viper.SetDefault("auth.require_strong_password", true)

	// Redis defaults
	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.database", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_connections", 5)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("redis.idle_timeout", "5m")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)

	// Security defaults
	viper.SetDefault("security.enable_cors", true)
	viper.SetDefault("security.cors_origins", []string{"*"})
	viper.SetDefault("security.cors_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors_headers", []string{"*"})
	viper.SetDefault("security.rate_limit_enabled", true)
	viper.SetDefault("security.rate_limit_rps", 100)
	viper.SetDefault("security.rate_limit_burst", 200)
	viper.SetDefault("security.request_timeout", "30s")
	viper.SetDefault("security.max_request_size", 10*1024*1024) // 10MB
	viper.SetDefault("security.enable_compression", true)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.port", 9100)
	viper.SetDefault("metrics.namespace", "sql_studio")
	viper.SetDefault("metrics.subsystem", "backend")

	// Email defaults
	viper.SetDefault("email.provider", "resend")
	viper.SetDefault("email.from_email", "noreply@sqlstudio.io")
	viper.SetDefault("email.from_name", "Howlerops")
	viper.SetDefault("email.base_url", "http://localhost:3000")

	// Sync defaults
	viper.SetDefault("sync.enabled", true)
	viper.SetDefault("sync.max_upload_size", 10*1024*1024) // 10MB
	viper.SetDefault("sync.conflict_strategy", "last_write_wins")
	viper.SetDefault("sync.retention_days", 30)
	viper.SetDefault("sync.max_history_items", 1000)
	viper.SetDefault("sync.enable_sanitization", true)
	viper.SetDefault("sync.rate_limit_rpm", 10)

	// Turso defaults
	viper.SetDefault("turso.url", "file:./data/development.db")
	viper.SetDefault("turso.max_connections", 25)

	// RAG defaults
	viper.SetDefault("rag.embedding.provider", "ollama")
	viper.SetDefault("rag.embedding.ollama.endpoint", "http://localhost:11434")
	viper.SetDefault("rag.embedding.ollama.model", "nomic-embed-text")
	viper.SetDefault("rag.embedding.ollama.dimension", 768)
	viper.SetDefault("rag.embedding.ollama.auto_pull", true)
	viper.SetDefault("rag.embedding.openai.model", "text-embedding-3-small")
	viper.SetDefault("rag.embedding.openai.dimension", 1536)
	viper.SetDefault("rag.embedding.openai.api_key_env", "OPENAI_API_KEY")
	viper.SetDefault("rag.embedding.cache.max_size", 10000)
	viper.SetDefault("rag.embedding.cache.ttl", "24h")
}

// validate validates the configuration
func validate(config *Config) error {
	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Server.GRPCPort <= 0 || config.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", config.Server.GRPCPort)
	}

	if config.Server.HTTPPort <= 0 || config.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.Server.HTTPPort)
	}

	// Validate auth configuration
	if len(config.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	if config.Auth.BcryptCost < 4 || config.Auth.BcryptCost > 31 {
		return fmt.Errorf("bcrypt cost must be between 4 and 31")
	}

	// Validate database configuration
	if config.Database.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive")
	}

	if config.Database.MaxIdleConns < 0 || config.Database.MaxIdleConns > config.Database.MaxConnections {
		return fmt.Errorf("max_idle_connections must be between 0 and max_connections")
	}

	// Validate Redis configuration
	if config.Redis.Enabled {
		if config.Redis.Port <= 0 || config.Redis.Port > 65535 {
			return fmt.Errorf("invalid Redis port: %d", config.Redis.Port)
		}
	}

	// Validate log configuration
	validLogLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}

	if !validLogLevels[strings.ToLower(config.Log.Level)] {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}

	if !validLogFormats[strings.ToLower(config.Log.Format)] {
		return fmt.Errorf("invalid log format: %s", config.Log.Format)
	}

	// Validate metrics configuration
	if config.Metrics.Enabled {
		if config.Metrics.Port <= 0 || config.Metrics.Port > 65535 {
			return fmt.Errorf("invalid metrics port: %d", config.Metrics.Port)
		}
	}

	return nil
}

// GetEnv returns environment name
func (c *Config) GetEnv() string {
	return c.Server.Environment
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Server.Environment) == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Server.Environment) == "production"
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetGRPCAddress returns the gRPC server address
func (c *Config) GetGRPCAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.GRPCPort)
}

// GetHTTPAddress returns the HTTP server address
func (c *Config) GetHTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.HTTPPort)
}

// GetMetricsAddress returns the metrics server address
func (c *Config) GetMetricsAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Metrics.Port)
}

// GetRedisAddress returns the Redis address
func (c *Config) GetRedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
