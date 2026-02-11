package audit

import "time"

// AuditLogDetailed represents field-level audit tracking
type AuditLogDetailed struct {
	ID         string    `json:"id"`
	AuditLogID string    `json:"audit_log_id"`
	TableName  string    `json:"table_name"`
	RecordID   string    `json:"record_id"`
	FieldName  string    `json:"field_name"`
	OldValue   string    `json:"old_value,omitempty"`
	NewValue   string    `json:"new_value,omitempty"`
	FieldType  string    `json:"field_type"` // 'pii', 'sensitive', 'normal'
	CreatedAt  time.Time `json:"created_at"`
}

// AuditChange represents a single field change
type AuditChange struct {
	TableName string      `json:"table_name"`
	RecordID  string      `json:"record_id"`
	FieldName string      `json:"field_name"`
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value,omitempty"`
	FieldType string      `json:"field_type"` // 'pii', 'sensitive', 'normal'
}

// ChangeHistory represents the complete change history for a record
type ChangeHistory struct {
	TableName string                   `json:"table_name"`
	RecordID  string                   `json:"record_id"`
	Fields    map[string][]FieldChange `json:"fields"`
}

// FieldChange represents a historical change to a field
type FieldChange struct {
	FieldName string    `json:"field_name"`
	OldValue  string    `json:"old_value,omitempty"`
	NewValue  string    `json:"new_value,omitempty"`
	FieldType string    `json:"field_type"`
	ChangedAt time.Time `json:"changed_at"`
	ChangedBy string    `json:"changed_by,omitempty"`
	AuditID   string    `json:"audit_id"`
}

// PIIAccessLog tracks access to PII fields
type PIIAccessLog struct {
	UserID     string    `json:"user_id"`
	TableName  string    `json:"table_name"`
	FieldName  string    `json:"field_name"`
	RecordID   string    `json:"record_id"`
	AccessType string    `json:"access_type"` // 'read', 'write', 'export'
	AccessedAt time.Time `json:"accessed_at"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
}
