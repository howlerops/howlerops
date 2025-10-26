package pii

import "time"

// PIIField represents a field that contains PII data
type PIIField struct {
	ID              string    `json:"id"`
	TableName       string    `json:"table_name"`
	FieldName       string    `json:"field_name"`
	PIIType         string    `json:"pii_type"`                   // 'email', 'phone', 'ssn', 'credit_card', 'address', 'name'
	DetectionMethod string    `json:"detection_method"`           // 'manual', 'pattern', 'ml'
	ConfidenceScore float64   `json:"confidence_score,omitempty"` // 0.0 to 1.0
	Verified        bool      `json:"verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PIIMatch represents a detected PII value in query results
type PIIMatch struct {
	Field           string      `json:"field"`
	Type            string      `json:"type"`
	Value           interface{} `json:"value"`
	ConfidenceScore float64     `json:"confidence_score"`
	Masked          bool        `json:"masked"`
	MaskedValue     string      `json:"masked_value,omitempty"`
}

// PIIScanResult contains results of PII scanning
type PIIScanResult struct {
	TotalFields    int        `json:"total_fields"`
	PIIFieldsFound int        `json:"pii_fields_found"`
	Matches        []PIIMatch `json:"matches"`
	ScannedAt      time.Time  `json:"scanned_at"`
}

// PIIPattern represents a pattern for detecting PII
type PIIPattern struct {
	Type        string  `json:"type"`
	Pattern     string  `json:"pattern"` // Regex pattern
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}
