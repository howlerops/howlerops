package catalog

import "time"

// TableCatalogEntry represents catalog metadata for a table
type TableCatalogEntry struct {
	ID             string                `json:"id"`
	ConnectionID   string                `json:"connection_id"`
	SchemaName     string                `json:"schema_name"`
	TableName      string                `json:"table_name"`
	Description    string                `json:"description,omitempty"`
	StewardUserID  *string               `json:"steward_user_id,omitempty"`
	Tags           []string              `json:"tags,omitempty"`
	OrganizationID *string               `json:"organization_id,omitempty"`
	Columns        []*ColumnCatalogEntry `json:"columns,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	CreatedBy      string                `json:"created_by"`
}

// ColumnCatalogEntry represents catalog metadata for a column
type ColumnCatalogEntry struct {
	ID             string    `json:"id"`
	TableCatalogID string    `json:"table_catalog_id"`
	ColumnName     string    `json:"column_name"`
	Description    string    `json:"description,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	PIIType        *string   `json:"pii_type,omitempty"`
	PIIConfidence  *float64  `json:"pii_confidence,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CatalogTag represents a reusable tag
type CatalogTag struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Color          string    `json:"color"`
	Description    string    `json:"description,omitempty"`
	OrganizationID *string   `json:"organization_id,omitempty"`
	IsSystem       bool      `json:"is_system"`
	CreatedAt      time.Time `json:"created_at"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Type           string   `json:"type"` // 'table' or 'column'
	ID             string   `json:"id"`
	ConnectionID   string   `json:"connection_id"`
	SchemaName     string   `json:"schema_name"`
	TableName      string   `json:"table_name"`
	ColumnName     string   `json:"column_name,omitempty"`
	Description    string   `json:"description,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	RelevanceScore float64  `json:"relevance_score"`
}

// SearchResults represents a collection of search results
type SearchResults struct {
	Results []*SearchResult `json:"results"`
	Total   int             `json:"total"`
	Query   string          `json:"query"`
}

// SearchFilters represents search filtering options
type SearchFilters struct {
	ConnectionID   *string  `json:"connection_id,omitempty"`
	SchemaName     *string  `json:"schema_name,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	OrganizationID *string  `json:"organization_id,omitempty"`
	Limit          int      `json:"limit,omitempty"`
}

// System tag constants
const (
	TagPII        = "PII"
	TagSensitive  = "Sensitive"
	TagInternal   = "Internal"
	TagPublic     = "Public"
	TagDeprecated = "Deprecated"
)

// SystemTags defines all predefined system tags
var SystemTags = []CatalogTag{
	{
		ID:          "tag-pii",
		Name:        TagPII,
		Color:       "#ef4444",
		Description: "Personally Identifiable Information",
		IsSystem:    true,
	},
	{
		ID:          "tag-sensitive",
		Name:        TagSensitive,
		Color:       "#f97316",
		Description: "Sensitive business data",
		IsSystem:    true,
	},
	{
		ID:          "tag-internal",
		Name:        TagInternal,
		Color:       "#eab308",
		Description: "Internal use only",
		IsSystem:    true,
	},
	{
		ID:          "tag-public",
		Name:        TagPublic,
		Color:       "#22c55e",
		Description: "Safe for public access",
		IsSystem:    true,
	},
	{
		ID:          "tag-deprecated",
		Name:        TagDeprecated,
		Color:       "#6b7280",
		Description: "Scheduled for removal",
		IsSystem:    true,
	},
}
