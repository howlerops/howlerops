package analytics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type QueryMetrics struct {
	db     *sql.DB
	logger *logrus.Logger
}

type QueryExecution struct {
	ID             string    `json:"id"`
	QueryID        string    `json:"query_id"`
	ConnectionID   string    `json:"connection_id"`
	UserID         string    `json:"user_id"`
	OrganizationID *string   `json:"organization_id,omitempty"`
	SQL            string    `json:"sql"`
	SQLHash        string    `json:"sql_hash"`
	ExecutionTime  int64     `json:"execution_time_ms"`
	RowsReturned   int       `json:"rows_returned"`
	Status         string    `json:"status"` // 'success', 'error', 'timeout'
	ErrorMessage   string    `json:"error_message,omitempty"`
	ExecutedAt     time.Time `json:"executed_at"`
}

type QueryStats struct {
	QueryID         string    `json:"query_id"`
	TotalExecutions int       `json:"total_executions"`
	SuccessCount    int       `json:"success_count"`
	ErrorCount      int       `json:"error_count"`
	TimeoutCount    int       `json:"timeout_count"`
	AvgExecutionMs  float64   `json:"avg_execution_ms"`
	MinExecutionMs  int64     `json:"min_execution_ms"`
	MaxExecutionMs  int64     `json:"max_execution_ms"`
	P50ExecutionMs  float64   `json:"p50_execution_ms"`
	P95ExecutionMs  float64   `json:"p95_execution_ms"`
	P99ExecutionMs  float64   `json:"p99_execution_ms"`
	SuccessRate     float64   `json:"success_rate"`
	LastExecutedAt  time.Time `json:"last_executed_at"`
}

type TopQuery struct {
	SQLHash        string  `json:"sql_hash"`
	SQL            string  `json:"sql"`
	ExecutionCount int     `json:"execution_count"`
	AvgTimeMs      float64 `json:"avg_time_ms"`
	SuccessRate    float64 `json:"success_rate"`
}

func NewQueryMetrics(db *sql.DB, logger *logrus.Logger) *QueryMetrics {
	return &QueryMetrics{
		db:     db,
		logger: logger,
	}
}

// Initialize creates the query_metrics table
func (m *QueryMetrics) Initialize() error {
	query := `
	CREATE TABLE IF NOT EXISTS query_metrics (
		id TEXT PRIMARY KEY,
		query_id TEXT,
		connection_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		organization_id TEXT,
		sql TEXT NOT NULL,
		sql_hash TEXT NOT NULL,
		execution_time_ms INTEGER NOT NULL,
		rows_returned INTEGER,
		status TEXT NOT NULL,
		error_message TEXT,
		executed_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_user ON query_metrics(user_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_org ON query_metrics(organization_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_sql_hash ON query_metrics(sql_hash);
	CREATE INDEX IF NOT EXISTS idx_metrics_executed_at ON query_metrics(executed_at);
	CREATE INDEX IF NOT EXISTS idx_metrics_status ON query_metrics(status);
	CREATE INDEX IF NOT EXISTS idx_metrics_execution_time ON query_metrics(execution_time_ms);
	`

	_, err := m.db.Exec(query)
	return err
}

// RecordExecution records a query execution
func (m *QueryMetrics) RecordExecution(ctx context.Context, exec *QueryExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	// Generate SQL hash for grouping similar queries
	if exec.SQLHash == "" {
		exec.SQLHash = m.hashSQL(exec.SQL)
	}

	query := `
	INSERT INTO query_metrics (
		id, query_id, connection_id, user_id, organization_id,
		sql, sql_hash, execution_time_ms, rows_returned,
		status, error_message, executed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		exec.ID,
		exec.QueryID,
		exec.ConnectionID,
		exec.UserID,
		exec.OrganizationID,
		exec.SQL,
		exec.SQLHash,
		exec.ExecutionTime,
		exec.RowsReturned,
		exec.Status,
		exec.ErrorMessage,
		exec.ExecutedAt.Unix(),
	)

	if err != nil {
		m.logger.WithError(err).Error("Failed to record query execution")
		return err
	}

	// Log slow queries
	if exec.ExecutionTime > 1000 { // > 1 second
		m.logger.WithFields(logrus.Fields{
			"query_id":     exec.QueryID,
			"execution_ms": exec.ExecutionTime,
			"user_id":      exec.UserID,
			"status":       exec.Status,
		}).Warn("Slow query detected")
	}

	return nil
}

// GetSlowQueries returns queries taking longer than threshold milliseconds
func (m *QueryMetrics) GetSlowQueries(ctx context.Context, thresholdMs int64, limit int) ([]*QueryExecution, error) {
	query := `
	SELECT id, query_id, connection_id, user_id, organization_id,
		   sql, sql_hash, execution_time_ms, rows_returned,
		   status, error_message, executed_at
	FROM query_metrics
	WHERE execution_time_ms > ?
	ORDER BY execution_time_ms DESC
	LIMIT ?`

	rows, err := m.db.QueryContext(ctx, query, thresholdMs, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var executions []*QueryExecution
	for rows.Next() {
		exec := &QueryExecution{}
		var executedAtUnix int64
		err := rows.Scan(
			&exec.ID,
			&exec.QueryID,
			&exec.ConnectionID,
			&exec.UserID,
			&exec.OrganizationID,
			&exec.SQL,
			&exec.SQLHash,
			&exec.ExecutionTime,
			&exec.RowsReturned,
			&exec.Status,
			&exec.ErrorMessage,
			&executedAtUnix,
		)
		if err != nil {
			return nil, err
		}
		exec.ExecutedAt = time.Unix(executedAtUnix, 0)
		executions = append(executions, exec)
	}

	return executions, nil
}

// GetQueryStats returns statistics for a specific query
func (m *QueryMetrics) GetQueryStats(ctx context.Context, sqlHash string) (*QueryStats, error) {
	// Get basic stats
	query := `
	SELECT
		COUNT(*) as total,
		SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
		SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
		SUM(CASE WHEN status = 'timeout' THEN 1 ELSE 0 END) as timeout_count,
		AVG(execution_time_ms) as avg_time,
		MIN(execution_time_ms) as min_time,
		MAX(execution_time_ms) as max_time,
		MAX(executed_at) as last_executed
	FROM query_metrics
	WHERE sql_hash = ?`

	stats := &QueryStats{}
	var lastExecutedUnix int64

	err := m.db.QueryRowContext(ctx, query, sqlHash).Scan(
		&stats.TotalExecutions,
		&stats.SuccessCount,
		&stats.ErrorCount,
		&stats.TimeoutCount,
		&stats.AvgExecutionMs,
		&stats.MinExecutionMs,
		&stats.MaxExecutionMs,
		&lastExecutedUnix,
	)
	if err != nil {
		return nil, err
	}

	stats.LastExecutedAt = time.Unix(lastExecutedUnix, 0)
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
	}

	// Calculate percentiles
	percentiles, err := m.calculatePercentiles(ctx, sqlHash)
	if err == nil && percentiles != nil {
		stats.P50ExecutionMs = percentiles["p50"]
		stats.P95ExecutionMs = percentiles["p95"]
		stats.P99ExecutionMs = percentiles["p99"]
	}

	return stats, nil
}

// GetTopQueries returns the most frequently executed queries
func (m *QueryMetrics) GetTopQueries(ctx context.Context, orgID *string, limit int) ([]*TopQuery, error) {
	var query string
	var args []interface{}

	if orgID != nil {
		query = `
		SELECT sql_hash, sql, COUNT(*) as exec_count,
			   AVG(execution_time_ms) as avg_time,
			   CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS REAL) / COUNT(*) * 100 as success_rate
		FROM query_metrics
		WHERE organization_id = ?
		GROUP BY sql_hash, sql
		ORDER BY exec_count DESC
		LIMIT ?`
		args = []interface{}{*orgID, limit}
	} else {
		query = `
		SELECT sql_hash, sql, COUNT(*) as exec_count,
			   AVG(execution_time_ms) as avg_time,
			   CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS REAL) / COUNT(*) * 100 as success_rate
		FROM query_metrics
		GROUP BY sql_hash, sql
		ORDER BY exec_count DESC
		LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var topQueries []*TopQuery
	for rows.Next() {
		q := &TopQuery{}
		err := rows.Scan(&q.SQLHash, &q.SQL, &q.ExecutionCount, &q.AvgTimeMs, &q.SuccessRate)
		if err != nil {
			return nil, err
		}
		topQueries = append(topQueries, q)
	}

	return topQueries, nil
}

// CleanupOldMetrics removes metrics older than the retention period
func (m *QueryMetrics) CleanupOldMetrics(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).Unix()

	query := `DELETE FROM query_metrics WHERE executed_at < ?`
	result, err := m.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	m.logger.WithField("rows_deleted", rowsAffected).Info("Cleaned up old query metrics")

	return nil
}

// calculatePercentiles calculates execution time percentiles
func (m *QueryMetrics) calculatePercentiles(ctx context.Context, sqlHash string) (map[string]float64, error) {
	query := `
	SELECT execution_time_ms
	FROM query_metrics
	WHERE sql_hash = ? AND status = 'success'
	ORDER BY execution_time_ms`

	rows, err := m.db.QueryContext(ctx, query, sqlHash)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var times []float64
	for rows.Next() {
		var execTime int64
		if err := rows.Scan(&execTime); err != nil {
			return nil, err
		}
		times = append(times, float64(execTime))
	}

	if len(times) == 0 {
		return nil, fmt.Errorf("no execution times found")
	}

	sort.Float64s(times)

	percentiles := make(map[string]float64)
	percentiles["p50"] = m.getPercentile(times, 50)
	percentiles["p95"] = m.getPercentile(times, 95)
	percentiles["p99"] = m.getPercentile(times, 99)

	return percentiles, nil
}

// getPercentile calculates a specific percentile from sorted slice
func (m *QueryMetrics) getPercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := (percentile / 100) * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// hashSQL generates a hash of the SQL query for grouping
func (m *QueryMetrics) hashSQL(sql string) string {
	// Normalize SQL: remove extra whitespace, convert to lowercase
	normalized := strings.TrimSpace(strings.ToLower(sql))
	normalized = strings.Join(strings.Fields(normalized), " ")

	// Generate hash
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}
