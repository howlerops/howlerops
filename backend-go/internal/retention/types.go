package retention

import "time"

// RetentionPolicy defines data retention rules for an organization
type RetentionPolicy struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	ResourceType    string    `json:"resource_type"` // 'query_history', 'audit_logs', 'connections', 'templates'
	RetentionDays   int       `json:"retention_days"`
	AutoArchive     bool      `json:"auto_archive"`
	ArchiveLocation string    `json:"archive_location,omitempty"` // 's3://bucket/path' or 'local'
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by"`
}

// ArchiveLog records when data was archived
type ArchiveLog struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	ResourceType    string    `json:"resource_type"`
	RecordsArchived int       `json:"records_archived"`
	ArchiveLocation string    `json:"archive_location"`
	ArchiveDate     time.Time `json:"archive_date"`
	CutoffDate      time.Time `json:"cutoff_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// ArchiveData represents data to be archived
type ArchiveData struct {
	ResourceType string                   `json:"resource_type"`
	Records      []map[string]interface{} `json:"records"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

// RetentionStats provides statistics about data retention
type RetentionStats struct {
	ResourceType     string    `json:"resource_type"`
	TotalRecords     int       `json:"total_records"`
	OldestRecord     time.Time `json:"oldest_record"`
	RecordsToArchive int       `json:"records_to_archive"`
	EstimatedSize    int64     `json:"estimated_size_bytes"`
}
