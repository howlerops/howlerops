package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionPool manages database connections with pooling
type ConnectionPool struct {
	config ConnectionConfig
	db     *sql.DB
	mu     sync.RWMutex
	closed bool
	logger *logrus.Logger
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config ConnectionConfig, logger *logrus.Logger) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		config: config,
		logger: logger,
	}

	if err := pool.connect(); err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return pool, nil
}

// connect establishes the database connection
func (p *ConnectionPool) connect() error {
	dsn, err := p.buildDSN()
	if err != nil {
		return fmt.Errorf("failed to build DSN: %w", err)
	}

	driverName, err := driverNameForType(p.config.Type)
	if err != nil {
		return err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if p.config.MaxConnections > 0 {
		db.SetMaxOpenConns(p.config.MaxConnections)
	} else {
		db.SetMaxOpenConns(25) // Default
	}

	if p.config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(p.config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5) // Default
	}

	if p.config.IdleTimeout > 0 {
		db.SetConnMaxIdleTime(p.config.IdleTimeout)
	} else {
		db.SetConnMaxIdleTime(5 * time.Minute) // Default
	}

	// Set connection max lifetime
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), p.getConnectionTimeout())
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	p.logger.WithFields(logrus.Fields{
		"type":     p.config.Type,
		"database": p.config.Database,
		"host":     p.config.Host,
	}).Info("Database connection pool created successfully")

	return nil
}

// buildDSN builds the data source name for the connection
func (p *ConnectionPool) buildDSN() (string, error) {
	switch p.config.Type {
	case PostgreSQL:
		return p.buildPostgresDSN(), nil
	case MySQL, MariaDB:
		return p.buildMySQLDSN(), nil
	case SQLite:
		return p.buildSQLiteDSN(), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", p.config.Type)
	}
}

// buildPostgresDSN builds PostgreSQL DSN
func (p *ConnectionPool) buildPostgresDSN() string {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
		p.config.Host, p.config.Port, p.config.Database, p.config.Username, p.config.Password)

	if p.config.SSLMode != "" {
		dsn += fmt.Sprintf(" sslmode=%s", p.config.SSLMode)
	} else {
		dsn += " sslmode=prefer"
	}

	if p.config.ConnectionTimeout > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", int(p.config.ConnectionTimeout.Seconds()))
	}

	// Add custom parameters
	for key, value := range p.config.Parameters {
		dsn += fmt.Sprintf(" %s=%s", key, value)
	}

	return dsn
}

// buildMySQLDSN builds MySQL/MariaDB DSN
func (p *ConnectionPool) buildMySQLDSN() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		p.config.Username, p.config.Password, p.config.Host, p.config.Port, p.config.Database)

	params := make(map[string]string)

	// Set default parameters
	params["parseTime"] = "true"
	params["loc"] = "UTC"

	if p.config.ConnectionTimeout > 0 {
		params["timeout"] = p.config.ConnectionTimeout.String()
	}

	// Add SSL configuration
	if p.config.SSLMode != "" {
		switch p.config.SSLMode {
		case "disable":
			params["tls"] = "false"
		case "require":
			params["tls"] = "true"
		default:
			params["tls"] = "preferred"
		}
	}

	// Add custom parameters
	for key, value := range p.config.Parameters {
		params[key] = value
	}

	// Build parameter string
	var paramStr string
	first := true
	for key, value := range params {
		if first {
			paramStr += "?"
			first = false
		} else {
			paramStr += "&"
		}
		paramStr += fmt.Sprintf("%s=%s", key, value)
	}

	return dsn + paramStr
}

// buildSQLiteDSN builds SQLite DSN
func (p *ConnectionPool) buildSQLiteDSN() string {
	dsn := p.config.Database

	if len(p.config.Parameters) > 0 {
		dsn += "?"
		first := true
		for key, value := range p.config.Parameters {
			if !first {
				dsn += "&"
			}
			dsn += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	return dsn
}

// getConnectionTimeout returns the connection timeout or a default value
func (p *ConnectionPool) getConnectionTimeout() time.Duration {
	if p.config.ConnectionTimeout > 0 {
		return p.config.ConnectionTimeout
	}
	return 30 * time.Second
}

func driverNameForType(dbType DatabaseType) (string, error) {
	switch dbType {
	case PostgreSQL:
		return "postgres", nil
	case MySQL, MariaDB:
		return "mysql", nil
	case SQLite:
		return "sqlite3", nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// Get returns a database connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*sql.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("connection pool is closed")
	}

	if p.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	return p.db, nil
}

// Close closes the connection pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if p.db != nil {
		err := p.db.Close()
		p.logger.WithFields(logrus.Fields{
			"type":     p.config.Type,
			"database": p.config.Database,
		}).Info("Database connection pool closed")
		return err
	}

	return nil
}

// Stats returns connection pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return PoolStats{}
	}

	stats := p.db.Stats()
	return PoolStats{
		OpenConnections:   stats.OpenConnections,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}
}

// Ping tests the database connection
func (p *ConnectionPool) Ping(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return fmt.Errorf("connection pool is closed")
	}

	if p.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	return p.db.PingContext(ctx)
}

// GetHealth returns the health status of the connection pool
func (p *ConnectionPool) GetHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Timestamp: start,
		Metrics:   make(map[string]string),
	}

	// Add pool statistics to metrics
	stats := p.Stats()
	status.Metrics["open_connections"] = fmt.Sprintf("%d", stats.OpenConnections)
	status.Metrics["in_use"] = fmt.Sprintf("%d", stats.InUse)
	status.Metrics["idle"] = fmt.Sprintf("%d", stats.Idle)

	// Test connection
	if err := p.Ping(ctx); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Failed to ping database: %v", err)
		status.ResponseTime = time.Since(start)
		return status
	}

	status.Status = "healthy"
	status.Message = "Database connection is healthy"
	status.ResponseTime = time.Since(start)

	return status
}

// Reconnect attempts to reconnect to the database
func (p *ConnectionPool) Reconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close existing connection
	if p.db != nil {
		p.db.Close()
		p.db = nil
	}

	// Reconnect
	return p.connect()
}
