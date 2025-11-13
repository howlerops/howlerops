package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type DashboardService struct {
	db           *sql.DB
	queryMetrics *QueryMetrics
	logger       *logrus.Logger
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type DashboardData struct {
	TimeRange    TimeRange          `json:"time_range"`
	Overview     *OverviewStats     `json:"overview"`
	QueryStats   *QueryStatsData    `json:"query_stats"`
	UserActivity *UserActivityStats `json:"user_activity"`
	Performance  *PerformanceStats  `json:"performance"`
	Connections  *ConnectionStats   `json:"connections"`
}

type OverviewStats struct {
	TotalQueries     int     `json:"total_queries"`
	TotalUsers       int     `json:"total_users"`
	TotalConnections int     `json:"total_connections"`
	ActiveUsers24h   int     `json:"active_users_24h"`
	QuerySuccessRate float64 `json:"query_success_rate"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
}

type QueryStatsData struct {
	TopQueries     []*TopQuery       `json:"top_queries"`
	SlowQueries    []*QueryExecution `json:"slow_queries"`
	QueryFrequency []TimeSeriesPoint `json:"query_frequency"`
	PopularTables  []TableUsage      `json:"popular_tables"`
	ErrorQueries   []*QueryExecution `json:"error_queries"`
	QueryTypes     map[string]int    `json:"query_types"` // SELECT, INSERT, UPDATE, DELETE
}

type UserActivityStats struct {
	ActiveUsersByDay []TimeSeriesPoint `json:"active_users_by_day"`
	QueryCountByUser []UserQueryCount  `json:"query_count_by_user"`
	PeakHours        []HourlyActivity  `json:"peak_hours"`
	UserGrowth       []TimeSeriesPoint `json:"user_growth"`
}

type PerformanceStats struct {
	AvgQueryTime  float64           `json:"avg_query_time_ms"`
	P50QueryTime  float64           `json:"p50_query_time_ms"`
	P95QueryTime  float64           `json:"p95_query_time_ms"`
	P99QueryTime  float64           `json:"p99_query_time_ms"`
	ErrorRate     float64           `json:"error_rate"`
	TimeoutRate   float64           `json:"timeout_rate"`
	Throughput    float64           `json:"queries_per_second"`
	ResponseTrend []TimeSeriesPoint `json:"response_trend"`
}

type ConnectionStats struct {
	TotalConnections  int                `json:"total_connections"`
	ActiveConnections int                `json:"active_connections"`
	ConnectionsByType map[string]int     `json:"connections_by_type"`
	ConnectionHealth  []ConnectionHealth `json:"connection_health"`
	RecentConnections []RecentConnection `json:"recent_connections"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

type TableUsage struct {
	TableName  string  `json:"table_name"`
	QueryCount int     `json:"query_count"`
	Percentage float64 `json:"percentage"`
}

type UserQueryCount struct {
	UserID     string `json:"user_id"`
	UserEmail  string `json:"user_email,omitempty"`
	QueryCount int    `json:"query_count"`
}

type HourlyActivity struct {
	Hour       int     `json:"hour"`
	QueryCount int     `json:"query_count"`
	AvgTime    float64 `json:"avg_time_ms"`
}

type ConnectionHealth struct {
	ConnectionID string    `json:"connection_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // healthy, degraded, unhealthy
	LastChecked  time.Time `json:"last_checked"`
	ResponseTime float64   `json:"response_time_ms"`
	ErrorRate    float64   `json:"error_rate"`
}

type RecentConnection struct {
	ConnectionID string    `json:"connection_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsed     time.Time `json:"last_used"`
}

func NewDashboardService(db *sql.DB, queryMetrics *QueryMetrics, logger *logrus.Logger) *DashboardService {
	return &DashboardService{
		db:           db,
		queryMetrics: queryMetrics,
		logger:       logger,
	}
}

// GetDashboardData returns comprehensive dashboard data
func (s *DashboardService) GetDashboardData(ctx context.Context, orgID *string, timeRange TimeRange) (*DashboardData, error) {
	data := &DashboardData{
		TimeRange: timeRange,
	}

	// Fetch all components in parallel
	errChan := make(chan error, 5)

	// Overview stats
	go func() {
		overview, err := s.getOverviewStats(ctx, orgID, timeRange)
		data.Overview = overview
		errChan <- err
	}()

	// Query stats
	go func() {
		queryStats, err := s.getQueryStats(ctx, orgID, timeRange)
		data.QueryStats = queryStats
		errChan <- err
	}()

	// User activity
	go func() {
		userActivity, err := s.getUserActivity(ctx, orgID, timeRange)
		data.UserActivity = userActivity
		errChan <- err
	}()

	// Performance stats
	go func() {
		performance, err := s.getPerformanceStats(ctx, orgID, timeRange)
		data.Performance = performance
		errChan <- err
	}()

	// Connection stats
	go func() {
		connections, err := s.getConnectionStats(ctx, orgID, timeRange)
		data.Connections = connections
		errChan <- err
	}()

	// Check for errors
	for i := 0; i < 5; i++ {
		if err := <-errChan; err != nil {
			s.logger.WithError(err).Error("Failed to fetch dashboard data")
			// Continue with partial data
		}
	}

	return data, nil
}

// getOverviewStats fetches overview statistics
func (s *DashboardService) getOverviewStats(ctx context.Context, orgID *string, timeRange TimeRange) (*OverviewStats, error) {
	stats := &OverviewStats{}

	// Base query conditions
	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// Total queries and success rate
	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_queries,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
			AVG(execution_time_ms) as avg_time
		FROM query_metrics %s`, whereClause)

	var successCount int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalQueries,
		&successCount,
		&stats.AvgResponseTime,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if stats.TotalQueries > 0 {
		stats.QuerySuccessRate = float64(successCount) / float64(stats.TotalQueries) * 100
	}

	// Count unique users
	query = fmt.Sprintf(`
		SELECT COUNT(DISTINCT user_id) FROM query_metrics %s`, whereClause)
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&stats.TotalUsers); err != nil {
		s.logger.WithError(err).Warn("Failed to scan total users count")
	}

	// Active users in last 24h
	query = `
		SELECT COUNT(DISTINCT user_id)
		FROM query_metrics
		WHERE executed_at >= ?`
	if err := s.db.QueryRowContext(ctx, query, time.Now().Add(-24*time.Hour).Unix()).Scan(&stats.ActiveUsers24h); err != nil {
		s.logger.WithError(err).Warn("Failed to scan active users 24h count")
	}

	// Total connections
	query = fmt.Sprintf(`
		SELECT COUNT(DISTINCT connection_id) FROM query_metrics %s`, whereClause)
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&stats.TotalConnections); err != nil {
		s.logger.WithError(err).Warn("Failed to scan total connections count")
	}

	return stats, nil
}

// getQueryStats fetches query-related statistics
func (s *DashboardService) getQueryStats(ctx context.Context, orgID *string, timeRange TimeRange) (*QueryStatsData, error) {
	stats := &QueryStatsData{
		QueryTypes: make(map[string]int),
	}

	// Get top queries
	topQueries, err := s.queryMetrics.GetTopQueries(ctx, orgID, 10)
	if err == nil {
		stats.TopQueries = topQueries
	}

	// Get slow queries
	slowQueries, err := s.queryMetrics.GetSlowQueries(ctx, 1000, 10) // > 1 second
	if err == nil {
		stats.SlowQueries = slowQueries
	}

	// Get query frequency over time
	stats.QueryFrequency = s.getQueryFrequency(ctx, orgID, timeRange)

	// Get popular tables
	stats.PopularTables = s.getPopularTables(ctx, orgID, timeRange)

	// Get query types distribution
	stats.QueryTypes = s.getQueryTypes(ctx, orgID, timeRange)

	// Get recent error queries
	stats.ErrorQueries = s.getErrorQueries(ctx, orgID, 5)

	return stats, nil
}

// getQueryFrequency returns query frequency over time
func (s *DashboardService) getQueryFrequency(ctx context.Context, orgID *string, timeRange TimeRange) []TimeSeriesPoint {
	// Determine appropriate bucket size based on time range
	duration := timeRange.End.Sub(timeRange.Start)
	var bucketSize int64 // in seconds
	var format string

	switch {
	case duration <= 24*time.Hour:
		bucketSize = 3600 // 1 hour buckets
		format = "hour"
	case duration <= 7*24*time.Hour:
		bucketSize = 86400 // 1 day buckets
		format = "day"
	default:
		bucketSize = 86400 // 1 day buckets
		format = "day"
	}

	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// #nosec G201 - whereClause built from parameterized conditions, bucketSize is validated constant
	query := fmt.Sprintf(`
		SELECT
			(executed_at / %d) * %d as bucket,
			COUNT(*) as query_count
		FROM query_metrics
		%s
		GROUP BY bucket
		ORDER BY bucket`, bucketSize, bucketSize, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get query frequency")
		return nil
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var points []TimeSeriesPoint
	for rows.Next() {
		var bucket int64
		var count int
		if err := rows.Scan(&bucket, &count); err != nil {
			continue
		}
		points = append(points, TimeSeriesPoint{
			Timestamp: time.Unix(bucket, 0),
			Value:     float64(count),
			Label:     format,
		})
	}

	return points
}

// getPopularTables extracts and counts table references
func (s *DashboardService) getPopularTables(ctx context.Context, orgID *string, timeRange TimeRange) []TableUsage {
	// This is a simplified version - in production, you'd want proper SQL parsing
	// For now, return empty - would need SQL parser for accurate table extraction
	// TODO: Implement table extraction using SQL parser
	// When implemented, build whereClause and args for filtering by orgID and timeRange
	return []TableUsage{}
}

// getQueryTypes returns distribution of query types
func (s *DashboardService) getQueryTypes(ctx context.Context, orgID *string, timeRange TimeRange) map[string]int {
	types := make(map[string]int)

	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN UPPER(SUBSTR(TRIM(sql), 1, 6)) = 'SELECT' THEN 'SELECT'
				WHEN UPPER(SUBSTR(TRIM(sql), 1, 6)) = 'INSERT' THEN 'INSERT'
				WHEN UPPER(SUBSTR(TRIM(sql), 1, 6)) = 'UPDATE' THEN 'UPDATE'
				WHEN UPPER(SUBSTR(TRIM(sql), 1, 6)) = 'DELETE' THEN 'DELETE'
				ELSE 'OTHER'
			END as query_type,
			COUNT(*) as count
		FROM query_metrics
		%s
		GROUP BY query_type`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return types
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	for rows.Next() {
		var queryType string
		var count int
		if err := rows.Scan(&queryType, &count); err == nil {
			types[queryType] = count
		}
	}

	return types
}

// getErrorQueries returns recent queries with errors
func (s *DashboardService) getErrorQueries(ctx context.Context, orgID *string, limit int) []*QueryExecution {
	whereClause := "WHERE status = 'error'"
	var args []interface{}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	whereClause += " ORDER BY executed_at DESC LIMIT ?"
	args = append(args, limit)

	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT id, query_id, connection_id, user_id, organization_id,
			   sql, sql_hash, execution_time_ms, rows_returned,
			   status, error_message, executed_at
		FROM query_metrics
		%s`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
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
		if err == nil {
			exec.ExecutedAt = time.Unix(executedAtUnix, 0)
			executions = append(executions, exec)
		}
	}

	return executions
}

// getUserActivity fetches user activity statistics
func (s *DashboardService) getUserActivity(ctx context.Context, orgID *string, timeRange TimeRange) (*UserActivityStats, error) {
	stats := &UserActivityStats{}

	// Get active users by day
	stats.ActiveUsersByDay = s.getActiveUsersByDay(ctx, orgID, timeRange)

	// Get query count by user (top 10)
	stats.QueryCountByUser = s.getQueryCountByUser(ctx, orgID, timeRange, 10)

	// Get peak hours
	stats.PeakHours = s.getPeakHours(ctx, orgID, timeRange)

	// Get user growth trend
	stats.UserGrowth = s.getUserGrowth(ctx, orgID, timeRange)

	return stats, nil
}

// Additional helper methods would go here...

// getActiveUsersByDay returns daily active user count
func (s *DashboardService) getActiveUsersByDay(ctx context.Context, orgID *string, timeRange TimeRange) []TimeSeriesPoint {
	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT
			(executed_at / 86400) * 86400 as day,
			COUNT(DISTINCT user_id) as user_count
		FROM query_metrics
		%s
		GROUP BY day
		ORDER BY day`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var day int64
		var count int
		if err := rows.Scan(&day, &count); err == nil {
			points = append(points, TimeSeriesPoint{
				Timestamp: time.Unix(day, 0),
				Value:     float64(count),
				Label:     "users",
			})
		}
	}

	return points
}

// getQueryCountByUser returns top users by query count
func (s *DashboardService) getQueryCountByUser(ctx context.Context, orgID *string, timeRange TimeRange, limit int) []UserQueryCount {
	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	args = append(args, limit)

	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT user_id, COUNT(*) as query_count
		FROM query_metrics
		%s
		GROUP BY user_id
		ORDER BY query_count DESC
		LIMIT ?`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var users []UserQueryCount
	for rows.Next() {
		var user UserQueryCount
		if err := rows.Scan(&user.UserID, &user.QueryCount); err == nil {
			users = append(users, user)
		}
	}

	return users
}

// getPeakHours returns hourly activity patterns
func (s *DashboardService) getPeakHours(ctx context.Context, orgID *string, timeRange TimeRange) []HourlyActivity {
	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT
			CAST(strftime('%%H', datetime(executed_at, 'unixepoch')) AS INTEGER) as hour,
			COUNT(*) as query_count,
			AVG(execution_time_ms) as avg_time
		FROM query_metrics
		%s
		GROUP BY hour
		ORDER BY hour`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var hours []HourlyActivity
	for rows.Next() {
		var activity HourlyActivity
		if err := rows.Scan(&activity.Hour, &activity.QueryCount, &activity.AvgTime); err == nil {
			hours = append(hours, activity)
		}
	}

	return hours
}

// getUserGrowth returns user growth trend
func (s *DashboardService) getUserGrowth(ctx context.Context, orgID *string, timeRange TimeRange) []TimeSeriesPoint {
	// This would typically query a users table with registration dates
	// For now, return empty
	return []TimeSeriesPoint{}
}

// getPerformanceStats fetches performance metrics
func (s *DashboardService) getPerformanceStats(ctx context.Context, orgID *string, timeRange TimeRange) (*PerformanceStats, error) {
	stats := &PerformanceStats{}

	whereClause := "WHERE executed_at >= ? AND executed_at <= ?"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// Get basic performance metrics
	// #nosec G201 - whereClause built from parameterized conditions with placeholders
	query := fmt.Sprintf(`
		SELECT
			AVG(execution_time_ms) as avg_time,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors,
			SUM(CASE WHEN status = 'timeout' THEN 1 ELSE 0 END) as timeouts
		FROM query_metrics
		%s`, whereClause)

	var total, errors, timeouts int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.AvgQueryTime,
		&total,
		&errors,
		&timeouts,
	)

	if err == nil && total > 0 {
		stats.ErrorRate = float64(errors) / float64(total) * 100
		stats.TimeoutRate = float64(timeouts) / float64(total) * 100

		// Calculate throughput (queries per second)
		duration := timeRange.End.Sub(timeRange.Start).Seconds()
		if duration > 0 {
			stats.Throughput = float64(total) / duration
		}
	}

	// Get percentiles for successful queries
	query = fmt.Sprintf(`
		SELECT execution_time_ms
		FROM query_metrics
		%s AND status = 'success'
		ORDER BY execution_time_ms`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err == nil {
		defer rows.Close()

		var times []float64
		for rows.Next() {
			var execTime int64
			if err := rows.Scan(&execTime); err == nil {
				times = append(times, float64(execTime))
			}
		}

		if len(times) > 0 {
			stats.P50QueryTime = getPercentileValue(times, 50)
			stats.P95QueryTime = getPercentileValue(times, 95)
			stats.P99QueryTime = getPercentileValue(times, 99)
		}
	}

	// Get response time trend
	stats.ResponseTrend = s.getResponseTrend(ctx, orgID, timeRange)

	return stats, nil
}

// getResponseTrend returns response time trend over the period
func (s *DashboardService) getResponseTrend(ctx context.Context, orgID *string, timeRange TimeRange) []TimeSeriesPoint {
	duration := timeRange.End.Sub(timeRange.Start)
	var bucketSize int64

	switch {
	case duration <= 24*time.Hour:
		bucketSize = 3600 // 1 hour
	default:
		bucketSize = 86400 // 1 day
	}

	whereClause := "WHERE executed_at >= ? AND executed_at <= ? AND status = 'success'"
	args := []interface{}{timeRange.Start.Unix(), timeRange.End.Unix()}

	if orgID != nil {
		whereClause += " AND organization_id = ?"
		args = append(args, *orgID)
	}

	// #nosec G201 - whereClause built from parameterized conditions, bucketSize is validated constant
	query := fmt.Sprintf(`
		SELECT
			(executed_at / %d) * %d as bucket,
			AVG(execution_time_ms) as avg_time
		FROM query_metrics
		%s
		GROUP BY bucket
		ORDER BY bucket`, bucketSize, bucketSize, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var bucket int64
		var avgTime float64
		if err := rows.Scan(&bucket, &avgTime); err == nil {
			points = append(points, TimeSeriesPoint{
				Timestamp: time.Unix(bucket, 0),
				Value:     avgTime,
			})
		}
	}

	return points
}

// getConnectionStats fetches connection-related statistics
func (s *DashboardService) getConnectionStats(ctx context.Context, orgID *string, timeRange TimeRange) (*ConnectionStats, error) {
	// This would integrate with the connection management system
	// For now, return mock data
	return &ConnectionStats{
		TotalConnections:  0,
		ActiveConnections: 0,
		ConnectionsByType: make(map[string]int),
	}, nil
}

// HTTP Handlers

// DashboardHandler serves dashboard data
func (s *DashboardService) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Parse time range from query params
	timeRangeStr := r.URL.Query().Get("range")
	if timeRangeStr == "" {
		timeRangeStr = "7d"
	}

	var timeRange TimeRange
	now := time.Now()

	switch timeRangeStr {
	case "24h":
		timeRange.Start = now.Add(-24 * time.Hour)
	case "7d":
		timeRange.Start = now.Add(-7 * 24 * time.Hour)
	case "30d":
		timeRange.Start = now.Add(-30 * 24 * time.Hour)
	default:
		timeRange.Start = now.Add(-7 * 24 * time.Hour)
	}
	timeRange.End = now

	// Get organization ID from context (if authenticated)
	var orgID *string
	if org := r.Context().Value("organization_id"); org != nil {
		orgStr := org.(string)
		orgID = &orgStr
	}

	// Fetch dashboard data
	data, err := s.GetDashboardData(r.Context(), orgID, timeRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode dashboard data")
	}
}

// RegisterRoutes registers dashboard HTTP routes
func (s *DashboardService) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/analytics/dashboard", s.DashboardHandler).Methods("GET")
}

// Helper function to calculate percentile
func getPercentileValue(sortedValues []float64, percentile float64) float64 {
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
