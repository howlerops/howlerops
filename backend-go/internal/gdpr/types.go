package gdpr

import "time"

// DataExportRequest represents a user's request to export or delete their data
type DataExportRequest struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	RequestType    string    `json:"request_type"` // 'export', 'delete'
	Status         string    `json:"status"`       // 'pending', 'processing', 'completed', 'failed'
	ExportURL      string    `json:"export_url,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	Metadata       string    `json:"metadata,omitempty"` // JSON
}

// UserDataExport contains all user data for GDPR export
type UserDataExport struct {
	User          interface{}            `json:"user"`
	Connections   []interface{}          `json:"connections"`
	Queries       []interface{}          `json:"queries"`
	QueryHistory  []interface{}          `json:"query_history"`
	Templates     []interface{}          `json:"templates"`
	Schedules     []interface{}          `json:"schedules"`
	Organizations []interface{}          `json:"organizations"`
	AuditLogs     []interface{}          `json:"audit_logs"`
	ExportedAt    time.Time              `json:"exported_at"`
	ExportVersion string                 `json:"export_version"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// DeletionReport provides details about what was deleted
type DeletionReport struct {
	UserID              string         `json:"user_id"`
	ConnectionsDeleted  int            `json:"connections_deleted"`
	QueriesDeleted      int            `json:"queries_deleted"`
	HistoryDeleted      int            `json:"history_deleted"`
	TemplatesDeleted    int            `json:"templates_deleted"`
	SchedulesDeleted    int            `json:"schedules_deleted"`
	AuditLogsAnonymized int            `json:"audit_logs_anonymized"`
	DeletedAt           time.Time      `json:"deleted_at"`
	Details             map[string]int `json:"details"`
}
