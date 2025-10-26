package sla

import "time"

// SLAMetrics represents SLA metrics for a day
type SLAMetrics struct {
	ID                string    `json:"id" db:"id"`
	OrganizationID    string    `json:"organization_id" db:"organization_id"`
	MetricDate        time.Time `json:"metric_date" db:"metric_date"`
	UptimePercentage  float64   `json:"uptime_percentage" db:"uptime_percentage"`
	AvgResponseTimeMS float64   `json:"avg_response_time_ms" db:"avg_response_time_ms"`
	ErrorRate         float64   `json:"error_rate" db:"error_rate"`
	P95ResponseTimeMS float64   `json:"p95_response_time_ms" db:"p95_response_time_ms"`
	P99ResponseTimeMS float64   `json:"p99_response_time_ms" db:"p99_response_time_ms"`
	TotalRequests     int       `json:"total_requests" db:"total_requests"`
	FailedRequests    int       `json:"failed_requests" db:"failed_requests"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// RequestLog represents a logged request for SLA calculation
type RequestLog struct {
	ID             string    `json:"id" db:"id"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	Endpoint       string    `json:"endpoint" db:"endpoint"`
	Method         string    `json:"method" db:"method"`
	ResponseTimeMS int       `json:"response_time_ms" db:"response_time_ms"`
	StatusCode     int       `json:"status_code" db:"status_code"`
	Success        bool      `json:"success" db:"success"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// SLAReport represents an SLA report for a period
type SLAReport struct {
	OrganizationID   string        `json:"organization_id"`
	StartDate        time.Time     `json:"start_date"`
	EndDate          time.Time     `json:"end_date"`
	OverallUptime    float64       `json:"overall_uptime"`
	AvgResponseTime  float64       `json:"avg_response_time"`
	OverallErrorRate float64       `json:"overall_error_rate"`
	TotalRequests    int           `json:"total_requests"`
	FailedRequests   int           `json:"failed_requests"`
	DailyMetrics     []SLAMetrics  `json:"daily_metrics"`
	WorstDays        []SLAMetrics  `json:"worst_days"` // Days with lowest uptime
	SLACompliance    SLACompliance `json:"sla_compliance"`
}

// SLACompliance represents SLA compliance status
type SLACompliance struct {
	TargetUptime       float64 `json:"target_uptime"` // e.g., 99.9
	ActualUptime       float64 `json:"actual_uptime"`
	CompliantDays      int     `json:"compliant_days"`
	NonCompliantDays   int     `json:"non_compliant_days"`
	ComplianceRate     float64 `json:"compliance_rate"`      // Percentage of days meeting SLA
	TargetResponseTime float64 `json:"target_response_time"` // e.g., 500ms
	ActualResponseTime float64 `json:"actual_response_time"`
	ResponseTimeOK     bool    `json:"response_time_ok"`
}
