package quotas

import "time"

// OrganizationQuota represents resource limits for an organization
type OrganizationQuota struct {
	OrganizationID        string    `json:"organization_id" db:"organization_id"`
	MaxConnections        int       `json:"max_connections" db:"max_connections"`
	MaxQueriesPerDay      int       `json:"max_queries_per_day" db:"max_queries_per_day"`
	MaxStorageMB          int       `json:"max_storage_mb" db:"max_storage_mb"`
	MaxAPICallsPerHour    int       `json:"max_api_calls_per_hour" db:"max_api_calls_per_hour"`
	MaxConcurrentQueries  int       `json:"max_concurrent_queries" db:"max_concurrent_queries"`
	MaxTeamMembers        int       `json:"max_team_members" db:"max_team_members"`
	FeaturesEnabled       string    `json:"features_enabled" db:"features_enabled"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationUsage represents current resource usage
type OrganizationUsage struct {
	ID                   string    `json:"id" db:"id"`
	OrganizationID       string    `json:"organization_id" db:"organization_id"`
	UsageDate            time.Time `json:"usage_date" db:"usage_date"`
	ConnectionsCount     int       `json:"connections_count" db:"connections_count"`
	QueriesCount         int       `json:"queries_count" db:"queries_count"`
	StorageUsedMB        float64   `json:"storage_used_mb" db:"storage_used_mb"`
	APICallsCount        int       `json:"api_calls_count" db:"api_calls_count"`
	ConcurrentQueriesPeak int      `json:"concurrent_queries_peak" db:"concurrent_queries_peak"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// UsageStatistics aggregated usage statistics
type UsageStatistics struct {
	OrganizationID        string               `json:"organization_id"`
	Period                string               `json:"period"` // e.g., "last_7_days"
	TotalQueries          int                  `json:"total_queries"`
	TotalAPIcalls         int                  `json:"total_api_calls"`
	AverageQueriesPerDay  float64              `json:"average_queries_per_day"`
	AverageAPICallsPerDay float64              `json:"average_api_calls_per_day"`
	PeakConcurrentQueries int                  `json:"peak_concurrent_queries"`
	CurrentStorageUsedMB  float64              `json:"current_storage_used_mb"`
	DailyUsage            []DailyUsageSummary  `json:"daily_usage"`
	QuotaLimits           OrganizationQuota    `json:"quota_limits"`
}

// DailyUsageSummary summarizes usage for a single day
type DailyUsageSummary struct {
	Date              string  `json:"date"`
	QueriesCount      int     `json:"queries_count"`
	APICallsCount     int     `json:"api_calls_count"`
	ConnectionsCount  int     `json:"connections_count"`
	StorageUsedMB     float64 `json:"storage_used_mb"`
}

// UpdateQuotaRequest represents a request to update quotas
type UpdateQuotaRequest struct {
	MaxConnections       *int    `json:"max_connections,omitempty"`
	MaxQueriesPerDay     *int    `json:"max_queries_per_day,omitempty"`
	MaxStorageMB         *int    `json:"max_storage_mb,omitempty"`
	MaxAPICallsPerHour   *int    `json:"max_api_calls_per_hour,omitempty"`
	MaxConcurrentQueries *int    `json:"max_concurrent_queries,omitempty"`
	MaxTeamMembers       *int    `json:"max_team_members,omitempty"`
	FeaturesEnabled      *string `json:"features_enabled,omitempty"`
}

// ResourceType represents different types of resources
type ResourceType string

const (
	ResourceConnection      ResourceType = "connection"
	ResourceQuery           ResourceType = "query"
	ResourceStorage         ResourceType = "storage"
	ResourceAPI             ResourceType = "api"
	ResourceConcurrentQuery ResourceType = "concurrent_query"
	ResourceTeamMember      ResourceType = "team_member"
)
