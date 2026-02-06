package main

// dtos.go
// Data Transfer Objects (DTOs) - All type definitions used across the application
// Extracted from app.go, updater.go, and ai_query_agent.go for better organization

import (
	"encoding/json"

	"github.com/jbeck018/howlerops/pkg/database"
	"github.com/jbeck018/howlerops/services"
)

// ============================================================================
// Connection Types
// ============================================================================

// ConnectionRequest represents a database connection request
type ConnectionRequest struct {
	ID                string            `json:"id,omitempty"`   // Optional stored connection ID
	Name              string            `json:"name,omitempty"` // Connection display name
	Type              string            `json:"type"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	Database          string            `json:"database"`
	Username          string            `json:"username"`
	Password          string            `json:"password"`
	SSLMode           string            `json:"sslMode,omitempty"`
	ConnectionTimeout int               `json:"connectionTimeout,omitempty"`
	Parameters        map[string]string `json:"parameters,omitempty"`
}

// ConnectionInfo represents connection information
type ConnectionInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Username  string `json:"username"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

// HealthStatus represents a connection health check without time.Duration/time.Time fields
type HealthStatus struct {
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	Timestamp    string            `json:"timestamp"`
	ResponseTime int64             `json:"response_time"`
	Metrics      map[string]string `json:"metrics"`
}

// ============================================================================
// Query Types
// ============================================================================

// QueryRequest represents a query execution request
type QueryRequest struct {
	ConnectionID string `json:"connectionId"`
	Query        string `json:"query"`
	Limit        int    `json:"limit,omitempty"`    // Page size (default 1000)
	Offset       int    `json:"offset,omitempty"`   // NEW: Pagination offset
	Timeout      int    `json:"timeout,omitempty"`  // seconds
	IsExport     bool   `json:"isExport,omitempty"` // NEW: Bypass limits for exports
}

// QueryResponse represents a query execution response
type QueryResponse struct {
	Columns  []string                        `json:"columns"`
	Rows     [][]interface{}                 `json:"rows"`
	RowCount int64                           `json:"rowCount"` // Total rows in result
	Affected int64                           `json:"affected"`
	Duration string                          `json:"duration"`
	Error    string                          `json:"error,omitempty"`
	Editable *database.EditableQueryMetadata `json:"editable,omitempty"`
	// Pagination metadata
	TotalRows int64 `json:"totalRows,omitempty"` // NEW: Total rows available (unpaginated)
	PagedRows int64 `json:"pagedRows,omitempty"` // NEW: Rows in this page
	HasMore   bool  `json:"hasMore,omitempty"`   // NEW: More data available
	Offset    int   `json:"offset,omitempty"`    // NEW: Current offset
}

// EditableMetadataJobResponse represents the status of an editable metadata background job
type EditableMetadataJobResponse struct {
	ID           string                          `json:"id"`
	ConnectionID string                          `json:"connectionId"`
	Status       string                          `json:"status"`
	Metadata     *database.EditableQueryMetadata `json:"metadata,omitempty"`
	Error        string                          `json:"error,omitempty"`
	CreatedAt    string                          `json:"createdAt"`
	CompletedAt  string                          `json:"completedAt,omitempty"`
}

// ============================================================================
// Query Row Operations
// ============================================================================

// QueryRowUpdateRequest represents an inline edit save request
type QueryRowUpdateRequest struct {
	ConnectionID string                 `json:"connectionId"`
	Query        string                 `json:"query"`
	Columns      []string               `json:"columns"`
	Schema       string                 `json:"schema,omitempty"`
	Table        string                 `json:"table,omitempty"`
	PrimaryKey   map[string]interface{} `json:"primaryKey"`
	Values       map[string]interface{} `json:"values"`
}

// QueryRowUpdateResponse represents the outcome of a save operation
type QueryRowUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// QueryRowInsertRequest represents an inline insert request
type QueryRowInsertRequest struct {
	ConnectionID string                 `json:"connectionId"`
	Query        string                 `json:"query"`
	Columns      []string               `json:"columns"`
	Schema       string                 `json:"schema,omitempty"`
	Table        string                 `json:"table,omitempty"`
	Values       map[string]interface{} `json:"values"`
}

// QueryRowInsertResponse represents the inserted row payload
type QueryRowInsertResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Row     map[string]interface{} `json:"row,omitempty"`
}

// QueryRowDeleteRequest represents a delete request (one or more rows)
type QueryRowDeleteRequest struct {
	ConnectionID string                   `json:"connectionId"`
	Query        string                   `json:"query"`
	Columns      []string                 `json:"columns"`
	Schema       string                   `json:"schema,omitempty"`
	Table        string                   `json:"table,omitempty"`
	PrimaryKeys  []map[string]interface{} `json:"primaryKeys"`
}

// QueryRowDeleteResponse represents delete results
type QueryRowDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Deleted int    `json:"deleted"`
}

// ============================================================================
// Database Operations
// ============================================================================

// ListDatabasesResponse represents available databases for a connection
type ListDatabasesResponse struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message,omitempty"`
	Databases []string `json:"databases,omitempty"`
}

// SwitchDatabaseRequest represents a request to switch the active database
type SwitchDatabaseRequest struct {
	ConnectionID string `json:"connectionId"`
	Database     string `json:"database"`
}

// SwitchDatabaseResponse represents the outcome of a database switch
type SwitchDatabaseResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
	Database    string `json:"database,omitempty"`
	Reconnected bool   `json:"reconnected"`
}

// ============================================================================
// Schema Types
// ============================================================================

// SyntheticViewSummary represents a synthetic view without backend-specific types
type SyntheticViewSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// TableInfo represents table metadata
type TableInfo struct {
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Comment   string `json:"comment"`
	RowCount  int64  `json:"rowCount"`
	SizeBytes int64  `json:"sizeBytes"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name               string            `json:"name"`
	DataType           string            `json:"data_type"`
	Nullable           bool              `json:"nullable"`
	DefaultValue       *string           `json:"default_value"`
	PrimaryKey         bool              `json:"primary_key"`
	Unique             bool              `json:"unique"`
	Indexed            bool              `json:"indexed"`
	Comment            string            `json:"comment"`
	OrdinalPosition    int               `json:"ordinal_position"`
	CharacterMaxLength *int64            `json:"character_maximum_length"`
	NumericPrecision   *int              `json:"numeric_precision"`
	NumericScale       *int              `json:"numeric_scale"`
	Metadata           map[string]string `json:"metadata"`
}

// IndexInfo represents index metadata without backend dependencies
type IndexInfo struct {
	Name     string            `json:"name"`
	Columns  []string          `json:"columns"`
	Unique   bool              `json:"unique"`
	Primary  bool              `json:"primary"`
	Type     string            `json:"type"`
	Method   string            `json:"method"`
	Metadata map[string]string `json:"metadata"`
}

// ForeignKeyInfo mirrors foreign key metadata with string-friendly fields
type ForeignKeyInfo struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referenced_table"`
	ReferencedSchema  string   `json:"referenced_schema"`
	ReferencedColumns []string `json:"referenced_columns"`
	OnDelete          string   `json:"on_delete"`
	OnUpdate          string   `json:"on_update"`
}

// TableStructure represents a table definition suitable for the frontend bindings
type TableStructure struct {
	Table       TableInfo         `json:"table"`
	Columns     []ColumnInfo      `json:"columns"`
	Indexes     []IndexInfo       `json:"indexes"`
	ForeignKeys []ForeignKeyInfo  `json:"foreign_keys"`
	Triggers    []string          `json:"triggers"`
	Statistics  map[string]string `json:"statistics"`
}

// ============================================================================
// Multi-Database Query Types
// ============================================================================

// MultiQueryRequest represents a multi-database query request
type MultiQueryRequest struct {
	Query    string            `json:"query"`
	Limit    int               `json:"limit,omitempty"`
	Timeout  int               `json:"timeout,omitempty"`  // seconds
	Strategy string            `json:"strategy,omitempty"` // "auto", "federated", "push_down"
	Options  map[string]string `json:"options,omitempty"`
}

// MultiQueryResponse represents a multi-database query response
type MultiQueryResponse struct {
	Columns         []string                           `json:"columns"`
	Rows            [][]interface{}                    `json:"rows"`
	RowCount        int64                              `json:"rowCount"`
	Duration        string                             `json:"duration"`
	ConnectionsUsed []string                           `json:"connectionsUsed"`
	Strategy        string                             `json:"strategy"`
	Error           string                             `json:"error,omitempty"`
	Editable        *services.EditableMetadataResponse `json:"editable,omitempty"`
}

// ValidationResult represents validation result for a multi-query
type ValidationResult struct {
	Valid               bool     `json:"valid"`
	Errors              []string `json:"errors,omitempty"`
	RequiredConnections []string `json:"requiredConnections,omitempty"`
	Tables              []string `json:"tables,omitempty"`
	EstimatedStrategy   string   `json:"estimatedStrategy,omitempty"`
}

// CombinedSchema represents combined schema from multiple connections
type CombinedSchema struct {
	Connections map[string]ConnectionSchema `json:"connections"`
	Conflicts   []SchemaConflict            `json:"conflicts"`
}

// ConnectionSchema represents schema info for a connection
type ConnectionSchema struct {
	ConnectionID string      `json:"connectionId"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Schemas      []string    `json:"schemas"`
	Tables       []TableInfo `json:"tables"`
}

// SchemaConflict represents a table name conflict
type SchemaConflict struct {
	TableName   string             `json:"tableName"`
	Connections []ConflictingTable `json:"connections"`
	Resolution  string             `json:"resolution"`
}

// ConflictingTable represents a table in a conflict
type ConflictingTable struct {
	ConnectionID string `json:"connectionId"`
	TableName    string `json:"tableName"`
	Schema       string `json:"schema"`
}

// ============================================================================
// AI/RAG Types
// ============================================================================

// NLQueryRequest represents a natural language query request
type NLQueryRequest struct {
	Prompt       string  `json:"prompt"`
	ConnectionID string  `json:"connectionId"`
	Context      string  `json:"context,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	Model        string  `json:"model,omitempty"`
	MaxTokens    int     `json:"maxTokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
}

// FixSQLRequest represents a request to fix an SQL statement
type FixSQLRequest struct {
	Query        string  `json:"query"`
	Error        string  `json:"error"`
	ConnectionID string  `json:"connectionId"`
	Provider     string  `json:"provider,omitempty"`
	Model        string  `json:"model,omitempty"`
	MaxTokens    int     `json:"maxTokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	Context      string  `json:"context,omitempty"`
}

// GenericChatRequest represents a generic AI chat request
type GenericChatRequest struct {
	Prompt      string            `json:"prompt"`
	Context     string            `json:"context,omitempty"`
	System      string            `json:"system,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// GenericChatResponse represents a generic AI chat response
type GenericChatResponse struct {
	Content    string            `json:"content"`
	Provider   string            `json:"provider"`
	Model      string            `json:"model"`
	TokensUsed int               `json:"tokensUsed,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AIMemoryMessagePayload represents a single conversational turn stored for memory
type AIMemoryMessagePayload struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AIMemorySessionPayload represents a persisted AI memory session
type AIMemorySessionPayload struct {
	ID            string                   `json:"id"`
	Title         string                   `json:"title"`
	CreatedAt     int64                    `json:"createdAt"`
	UpdatedAt     int64                    `json:"updatedAt"`
	Summary       string                   `json:"summary,omitempty"`
	SummaryTokens int                      `json:"summaryTokens,omitempty"`
	Metadata      map[string]interface{}   `json:"metadata,omitempty"`
	Messages      []AIMemoryMessagePayload `json:"messages"`
}

// GeneratedSQLResponse represents a generated SQL query
type GeneratedSQLResponse struct {
	SQL          string             `json:"sql"`
	Confidence   float64            `json:"confidence"`
	Explanation  string             `json:"explanation"`
	Warnings     []string           `json:"warnings,omitempty"`
	Alternatives []AlternativeQuery `json:"alternatives,omitempty"`
}

// AlternativeQuery represents an alternative query option
type AlternativeQuery struct {
	SQL         string  `json:"sql"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

// FixedSQLResponse represents a fixed SQL query
type FixedSQLResponse struct {
	SQL         string   `json:"sql"`
	Explanation string   `json:"explanation"`
	Changes     []string `json:"changes"`
}

// AIMemoryRecallResult represents a recalled AI memory snippet
type AIMemoryRecallResult struct {
	SessionID string  `json:"sessionId"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary,omitempty"`
	Content   string  `json:"content"`
	Score     float32 `json:"score"`
}

// OptimizationResponse represents an optimized query
type OptimizationResponse struct {
	SQL              string       `json:"sql"`
	EstimatedSpeedup string       `json:"estimatedSpeedup"`
	Explanation      string       `json:"explanation"`
	Suggestions      []Suggestion `json:"suggestions"`
}

// Suggestion represents an autocomplete or optimization suggestion
type Suggestion struct {
	Text        string  `json:"text"`
	Type        string  `json:"type"`
	Detail      string  `json:"detail,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	Description string  `json:"description,omitempty"`
	SQL         string  `json:"sql,omitempty"`
}

// VizSuggestion represents a visualization suggestion
type VizSuggestion struct {
	ChartType   string            `json:"chartType"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Config      map[string]string `json:"config"`
	Confidence  float64           `json:"confidence"`
}

// ResultData represents query result data for AI processing
type ResultData struct {
	Columns  []string        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int64           `json:"rowCount"`
}

// ProviderStatus represents AI provider status
type ProviderStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
	Model     string `json:"model,omitempty"`
}

// ProviderConfig represents AI provider configuration
type ProviderConfig struct {
	Provider string            `json:"provider"`
	APIKey   string            `json:"apiKey,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
	Model    string            `json:"model,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

// ============================================================================
// AI Query Agent Types (from ai_query_agent.go)
// ============================================================================

// AIQueryAgentRequest represents a request to the AI query agent workflow
type AIQueryAgentRequest struct {
	SessionID     string                   `json:"sessionId"`
	Message       string                   `json:"message"`
	Provider      string                   `json:"provider"`
	Model         string                   `json:"model"`
	ConnectionID  string                   `json:"connectionId,omitempty"`
	ConnectionIDs []string                 `json:"connectionIds,omitempty"`
	SchemaContext string                   `json:"schemaContext,omitempty"`
	Context       string                   `json:"context,omitempty"`
	History       []AIMemoryMessagePayload `json:"history,omitempty"`
	SystemPrompt  string                   `json:"systemPrompt,omitempty"`
	Temperature   float64                  `json:"temperature,omitempty"`
	MaxTokens     int                      `json:"maxTokens,omitempty"`
	MaxRows       int                      `json:"maxRows,omitempty"`
	Page          int                      `json:"page,omitempty"`     // Current page number (1-indexed)
	PageSize      int                      `json:"pageSize,omitempty"` // Rows per page
}

// AIQueryAgentResponse aggregates the generated artefacts for a single turn
type AIQueryAgentResponse struct {
	SessionID   string                 `json:"sessionId"`
	TurnID      string                 `json:"turnId"`
	Provider    string                 `json:"provider"`
	Model       string                 `json:"model"`
	Messages    []AIQueryAgentMessage  `json:"messages"`
	Error       string                 `json:"error,omitempty"`
	DurationMs  int64                  `json:"durationMs"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExecutedSQL string                 `json:"executedSql,omitempty"`
}

// AIQueryAgentMessage represents a single agent response (orchestrator, sql generator, etc)
type AIQueryAgentMessage struct {
	ID          string                     `json:"id"`
	Agent       string                     `json:"agent"`
	Role        string                     `json:"role"`
	Title       string                     `json:"title,omitempty"`
	Content     string                     `json:"content"`
	CreatedAt   int64                      `json:"createdAt"`
	Attachments []AIQueryAgentAttachment   `json:"attachments,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
	Warnings    []string                   `json:"warnings,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Provider    string                     `json:"provider,omitempty"`
	Model       string                     `json:"model,omitempty"`
	TokensUsed  int                        `json:"tokensUsed,omitempty"`
	ElapsedMs   int64                      `json:"elapsedMs,omitempty"`
	Context     map[string]json.RawMessage `json:"context,omitempty"`
}

// AIQueryAgentAttachment represents rich content associated with a message
type AIQueryAgentAttachment struct {
	Type       string                         `json:"type"`
	SQL        *AIQueryAgentSQLAttachment     `json:"sql,omitempty"`
	Result     *AIQueryAgentResultAttachment  `json:"result,omitempty"`
	Chart      *AIQueryAgentChartAttachment   `json:"chart,omitempty"`
	Report     *AIQueryAgentReportAttachment  `json:"report,omitempty"`
	Insight    *AIQueryAgentInsightAttachment `json:"insight,omitempty"`
	RawPayload map[string]interface{}         `json:"rawPayload,omitempty"`
}

// AIQueryAgentSQLAttachment contains generated SQL information
type AIQueryAgentSQLAttachment struct {
	Query        string   `json:"query"`
	Explanation  string   `json:"explanation,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	ConnectionID string   `json:"connectionId,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// AIQueryAgentResultAttachment contains a lightweight data preview
type AIQueryAgentResultAttachment struct {
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int64                    `json:"rowCount"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	Limited         bool                     `json:"limited"`
	ConnectionID    string                   `json:"connectionId,omitempty"`
	TotalRows       int64                    `json:"totalRows,omitempty"`
	Page            int                      `json:"page,omitempty"`
	PageSize        int                      `json:"pageSize,omitempty"`
	TotalPages      int                      `json:"totalPages,omitempty"`
	HasMore         bool                     `json:"hasMore,omitempty"`
}

// AIQueryAgentChartAttachment represents a chart suggestion produced by the agent
type AIQueryAgentChartAttachment struct {
	Type          string           `json:"type"`
	XField        string           `json:"xField"`
	YFields       []string         `json:"yFields"`
	SeriesField   string           `json:"seriesField,omitempty"`
	Title         string           `json:"title,omitempty"`
	Description   string           `json:"description,omitempty"`
	Recommended   bool             `json:"recommended"`
	PreviewValues []map[string]any `json:"previewValues,omitempty"`
}

// AIQueryAgentReportAttachment is a formatted report
type AIQueryAgentReportAttachment struct {
	Format string `json:"format"`
	Body   string `json:"body"`
	Title  string `json:"title,omitempty"`
}

// AIQueryAgentInsightAttachment holds structured insights
type AIQueryAgentInsightAttachment struct {
	Highlights []string `json:"highlights"`
}

// ReadOnlyQueryResult represents a guarded SELECT output
type ReadOnlyQueryResult struct {
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int64                    `json:"rowCount"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	Limited         bool                     `json:"limited"`
	ConnectionID    string                   `json:"connectionId"`
	TotalRows       int64                    `json:"totalRows,omitempty"`
	Page            int                      `json:"page,omitempty"`
	PageSize        int                      `json:"pageSize,omitempty"`
	TotalPages      int                      `json:"totalPages,omitempty"`
	HasMore         bool                     `json:"hasMore,omitempty"`
	Offset          int                      `json:"offset,omitempty"`
}

// ============================================================================
// AI Response Types
// ============================================================================

// ModelInfoResponse represents a model available from a provider
type ModelInfoResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description,omitempty"`
	MaxTokens   int    `json:"maxTokens,omitempty"`
	Source      string `json:"source,omitempty"`
}

// AITestResponse represents the result of testing an AI provider connection
type AITestResponse struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpdateInfo and GitHubRelease are defined in updater.go
