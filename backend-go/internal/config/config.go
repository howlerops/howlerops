package config

import (
	"fmt"
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
	JWTSecret          string        `mapstructure:"jwt_secret"`
	JWTExpiration      time.Duration `mapstructure:"jwt_expiration"`
	RefreshExpiration  time.Duration `mapstructure:"refresh_expiration"`
	BcryptCost         int           `mapstructure:"bcrypt_cost"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout"`
	MaxLoginAttempts   int           `mapstructure:"max_login_attempts"`
	LockoutDuration    time.Duration `mapstructure:"lockout_duration"`
	RequireStrongPass  bool          `mapstructure:"require_strong_password"`
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
	Enabled    bool   `mapstructure:"enabled"`
	Path       string `mapstructure:"path"`
	Port       int    `mapstructure:"port"`
	Namespace  string `mapstructure:"namespace"`
	Subsystem  string `mapstructure:"subsystem"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Set default values
	setDefaults()

	// Set up Viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/sql-studio")
	viper.AddConfigPath("$HOME/.sql-studio")

	// Enable environment variable support
	viper.SetEnvPrefix("SQL_STUDIO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	// Unmarshal configuration
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
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