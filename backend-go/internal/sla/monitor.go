package sla

import (
	"log"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Monitor handles SLA monitoring and calculation
type Monitor struct {
	store  *Store
	logger *logrus.Logger
}

// NewMonitor creates a new SLA monitor
func NewMonitor(store *Store, logger *logrus.Logger) *Monitor {
	return &Monitor{
		store:  store,
		logger: logger,
	}
}

// RecordRequest records a request for SLA calculation
func (m *Monitor) RecordRequest(ctx context.Context, orgID, endpoint, method string, duration time.Duration, statusCode int) error {
	log := CreateRequestLog(
		orgID,
		endpoint,
		method,
		int(duration.Milliseconds()),
		statusCode,
	)

	if err := m.store.LogRequest(ctx, log); err != nil {
		m.logger.WithError(err).Error("Failed to log request for SLA")
		return err
	}

	return nil
}

// CalculateDailySLA calculates SLA metrics for a specific day
func (m *Monitor) CalculateDailySLA(ctx context.Context, orgID string, date time.Time) (*SLAMetrics, error) {
	// Get all requests for the day
	requests, err := m.store.GetRequestsForDay(ctx, orgID, date)
	if err != nil {
		return nil, fmt.Errorf("get requests: %w", err)
	}

	if len(requests) == 0 {
		m.logger.WithFields(logrus.Fields{
			"organization_id": orgID,
			"date":            date.Format("2006-01-02"),
		}).Debug("No requests found for SLA calculation")
		return nil, nil
	}

	// Calculate metrics
	totalRequests := len(requests)
	failedRequests := 0
	var totalDuration int64
	var durations []int

	for _, req := range requests {
		if !req.Success {
			failedRequests++
		}
		totalDuration += int64(req.ResponseTimeMS)
		durations = append(durations, req.ResponseTimeMS)
	}

	// Sort durations for percentile calculation
	sort.Ints(durations)

	// Calculate percentiles
	p95Index := int(float64(len(durations)) * 0.95)
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}

	p99Index := int(float64(len(durations)) * 0.99)
	if p99Index >= len(durations) {
		p99Index = len(durations) - 1
	}

	// Calculate uptime and error rate
	successfulRequests := totalRequests - failedRequests
	uptimePercentage := float64(successfulRequests) / float64(totalRequests) * 100
	errorRate := float64(failedRequests) / float64(totalRequests) * 100
	avgResponseTime := float64(totalDuration) / float64(totalRequests)

	metrics := &SLAMetrics{
		ID:                uuid.New().String(),
		OrganizationID:    orgID,
		MetricDate:        date,
		UptimePercentage:  uptimePercentage,
		AvgResponseTimeMS: avgResponseTime,
		ErrorRate:         errorRate,
		P95ResponseTimeMS: float64(durations[p95Index]),
		P99ResponseTimeMS: float64(durations[p99Index]),
		TotalRequests:     totalRequests,
		FailedRequests:    failedRequests,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Save metrics
	if err := m.store.SaveSLAMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("save metrics: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"date":            date.Format("2006-01-02"),
		"uptime":          fmt.Sprintf("%.2f%%", uptimePercentage),
		"avg_response":    fmt.Sprintf("%.2fms", avgResponseTime),
		"total_requests":  totalRequests,
	}).Info("SLA metrics calculated")

	return metrics, nil
}

// GenerateSLAReport generates a comprehensive SLA report for a period
func (m *Monitor) GenerateSLAReport(ctx context.Context, orgID string, startDate, endDate time.Time, targetUptime, targetResponseTime float64) (*SLAReport, error) {
	// Get metrics for period
	metrics, err := m.store.GetMetricsForPeriod(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found for period")
	}

	// Calculate overall statistics
	var totalRequests, totalFailed int
	var totalResponseTime float64
	var worstDays []*SLAMetrics
	compliantDays := 0
	nonCompliantDays := 0

	for _, m := range metrics {
		totalRequests += m.TotalRequests
		totalFailed += m.FailedRequests
		totalResponseTime += m.AvgResponseTimeMS * float64(m.TotalRequests)

		// Check SLA compliance
		if m.UptimePercentage >= targetUptime {
			compliantDays++
		} else {
			nonCompliantDays++
			worstDays = append(worstDays, m)
		}
	}

	// Sort worst days by uptime (lowest first)
	sort.Slice(worstDays, func(i, j int) bool {
		return worstDays[i].UptimePercentage < worstDays[j].UptimePercentage
	})

	// Keep only top 5 worst days
	if len(worstDays) > 5 {
		worstDays = worstDays[:5]
	}

	// Calculate overall metrics
	overallUptime := float64(totalRequests-totalFailed) / float64(totalRequests) * 100
	avgResponseTime := totalResponseTime / float64(totalRequests)
	overallErrorRate := float64(totalFailed) / float64(totalRequests) * 100

	// SLA compliance
	complianceRate := float64(compliantDays) / float64(len(metrics)) * 100
	responseTimeOK := avgResponseTime <= targetResponseTime

	report := &SLAReport{
		OrganizationID:   orgID,
		StartDate:        startDate,
		EndDate:          endDate,
		OverallUptime:    overallUptime,
		AvgResponseTime:  avgResponseTime,
		OverallErrorRate: overallErrorRate,
		TotalRequests:    totalRequests,
		FailedRequests:   totalFailed,
		DailyMetrics:     convertMetricsSlice(metrics),
		WorstDays:        convertMetricsSlice(worstDays),
		SLACompliance: SLACompliance{
			TargetUptime:       targetUptime,
			ActualUptime:       overallUptime,
			CompliantDays:      compliantDays,
			NonCompliantDays:   nonCompliantDays,
			ComplianceRate:     complianceRate,
			TargetResponseTime: targetResponseTime,
			ActualResponseTime: avgResponseTime,
			ResponseTimeOK:     responseTimeOK,
		},
	}

	return report, nil
}

// GetLatestMetrics retrieves recent SLA metrics
func (m *Monitor) GetLatestMetrics(ctx context.Context, orgID string, days int) ([]*SLAMetrics, error) {
	metrics, err := m.store.GetLatestMetrics(ctx, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("get latest metrics: %w", err)
	}
	return metrics, nil
}

// StartScheduler starts the background scheduler for SLA calculation
func (m *Monitor) StartScheduler(ctx context.Context) {
	// Calculate SLA for all organizations daily at 1 AM
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case now := <-ticker.C:
				// Only run at 1 AM
				if now.Hour() == 1 {
					m.calculateAllOrgSLAs(ctx, now.AddDate(0, 0, -1))
				}
			}
		}
	}()

	m.logger.Info("SLA monitoring scheduler started")
}

// calculateAllOrgSLAs calculates SLA for all organizations for a given date
func (m *Monitor) calculateAllOrgSLAs(ctx context.Context, date time.Time) {
	m.logger.WithField("date", date.Format("2006-01-02")).Info("Calculating SLA metrics for all organizations")

	// Get all organization IDs
	query := `SELECT DISTINCT organization_id FROM request_log WHERE created_at >= ? AND created_at < ?`
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := m.store.db.QueryContext(ctx, query, startOfDay.Unix(), endOfDay.Unix())
	if err != nil {
		m.logger.WithError(err).Error("Failed to get organization IDs")
		return
	}
	defer func() { if err := rows.Close(); err != nil { log.Printf("Failed to close rows: %v", err) } }()

	var orgIDs []string
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			m.logger.WithError(err).Error("Failed to scan organization ID")
			continue
		}
		orgIDs = append(orgIDs, orgID)
	}

	// Calculate SLA for each organization
	for _, orgID := range orgIDs {
		if _, err := m.CalculateDailySLA(ctx, orgID, date); err != nil {
			m.logger.WithError(err).WithField("organization_id", orgID).Error("Failed to calculate SLA")
		}
	}

	m.logger.WithFields(logrus.Fields{
		"date":          date.Format("2006-01-02"),
		"organizations": len(orgIDs),
	}).Info("SLA calculation completed")
}

// StartCleanupScheduler starts background cleanup of old logs
func (m *Monitor) StartCleanupScheduler(ctx context.Context, retentionDays int) {
	// Run cleanup daily at 2 AM
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case now := <-ticker.C:
				if now.Hour() == 2 {
					if err := m.store.CleanupOldLogs(ctx, retentionDays); err != nil {
						m.logger.WithError(err).Error("Failed to cleanup old logs")
					} else {
						m.logger.Info("Old request logs cleaned up")
					}
				}
			}
		}
	}()

	m.logger.WithField("retention_days", retentionDays).Info("SLA cleanup scheduler started")
}

// Helper function to convert []*SLAMetrics to []SLAMetrics
func convertMetricsSlice(metrics []*SLAMetrics) []SLAMetrics {
	result := make([]SLAMetrics, len(metrics))
	for i, m := range metrics {
		result[i] = *m
	}
	return result
}
