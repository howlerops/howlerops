package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sql-studio/backend-go/pkg/ai"
	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/backend-go/pkg/database/multiquery"
	"github.com/sql-studio/backend-go/pkg/rag"
	"github.com/sql-studio/backend-go/pkg/storage"
	"github.com/sql-studio/sql-studio/services"
)

//go:embed howlerops-light.png howlerops-dark.png howlerops-transparent.png
var iconFS embed.FS

// App struct
type App struct {
	ctx              context.Context
	logger           *logrus.Logger
	storageManager   *storage.Manager // Storage for connections, queries, and RAG
	databaseService  *services.DatabaseService
	fileService      *services.FileService
	keyboardService  *services.KeyboardService
	aiService        *ai.Service
	aiConfig         *ai.RuntimeConfig
	embeddingService rag.EmbeddingService
}

// ConnectionRequest represents a database connection request
type ConnectionRequest struct {
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

// QueryRequest represents a query execution request
type QueryRequest struct {
	ConnectionID string `json:"connectionId"`
	Query        string `json:"query"`
	Limit        int    `json:"limit,omitempty"`
}

// QueryResponse represents a query execution response
type QueryResponse struct {
	Columns  []string                        `json:"columns"`
	Rows     [][]interface{}                 `json:"rows"`
	RowCount int64                           `json:"rowCount"`
	Affected int64                           `json:"affected"`
	Duration string                          `json:"duration"`
	Error    string                          `json:"error,omitempty"`
	Editable *database.EditableQueryMetadata `json:"editable,omitempty"`
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

// ConnectionInfo represents connection information
type ConnectionInfo struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Database  string    `json:"database"`
	Username  string    `json:"username"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
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
	Name               string  `json:"name"`
	DataType           string  `json:"dataType"`
	Nullable           bool    `json:"nullable"`
	DefaultValue       *string `json:"defaultValue"`
	PrimaryKey         bool    `json:"primaryKey"`
	Unique             bool    `json:"unique"`
	OrdinalPosition    int     `json:"ordinalPosition"`
	CharacterMaxLength *int64  `json:"characterMaxLength"`
	NumericPrecision   *int    `json:"numericPrecision"`
	NumericScale       *int    `json:"numericScale"`
}

// Multi-Database Query Types

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
	Columns         []string        `json:"columns"`
	Rows            [][]interface{} `json:"rows"`
	RowCount        int64           `json:"rowCount"`
	Duration        string          `json:"duration"`
	ConnectionsUsed []string        `json:"connectionsUsed"`
	Strategy        string          `json:"strategy"`
	Error           string          `json:"error,omitempty"`
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

// AI/RAG Types

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

// NewApp creates a new App application struct
func NewApp() *App {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Create services
	databaseService := services.NewDatabaseService(logger)
	fileService := services.NewFileService(logger)
	keyboardService := services.NewKeyboardService(logger)

	return &App{
		logger:          logger,
		databaseService: databaseService,
		fileService:     fileService,
		keyboardService: keyboardService,
		aiConfig:        ai.DefaultRuntimeConfig(),
	}
}

// OnStartup is called when the app starts, before the frontend is loaded
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Set context for services
	a.databaseService.SetContext(ctx)
	a.fileService.SetContext(ctx)
	a.keyboardService.SetContext(ctx)

	// Initialize storage manager
	if err := a.initializeStorageManager(ctx); err != nil {
		a.logger.WithError(err).Error("Failed to initialize storage manager")
		// Continue without storage - graceful degradation
	}

	a.logger.Info("HowlerOps desktop application started")

	// Emit app ready event
	wailsRuntime.EventsEmit(ctx, "app:startup-complete")
}

// initializeStorageManager initializes the storage manager for local data
func (a *App) initializeStorageManager(ctx context.Context) error {
	// Get user ID from environment or generate one
	userID := getEnvOrDefault("HOWLEROPS_USER_ID", "local-user")

	vectorStoreType := strings.ToLower(getEnvOrDefault("VECTOR_STORE_TYPE", "sqlite"))
	mysqlVectorDSN := os.Getenv("MYSQL_VECTOR_DSN")
	mysqlVectorSize := getEnvOrDefault("MYSQL_VECTOR_SIZE", "1536")
	vectorSize := 1536
	if parsed, err := strconv.Atoi(mysqlVectorSize); err == nil {
		vectorSize = parsed
	}

	// Configure storage
	storageConfig := &storage.Config{
		Mode: storage.ModeSolo,
		Local: storage.LocalStorageConfig{
			DataDir:         getEnvOrDefault("HOWLEROPS_DATA_DIR", "~/.howlerops"),
			Database:        "local.db",
			VectorsDB:       "vectors.db",
			UserID:          userID,
			VectorSize:      vectorSize,
			VectorStoreType: vectorStoreType,
		},
		UserID: userID,
	}

	if vectorStoreType == "mysql" && mysqlVectorDSN != "" {
		storageConfig.Local.MySQLVector = &storage.MySQLVectorConfig{
			DSN:        mysqlVectorDSN,
			VectorSize: vectorSize,
		}
	}

	// Check if team mode is enabled
	if os.Getenv("HOWLEROPS_MODE") == "team" && os.Getenv("TURSO_URL") != "" {
		storageConfig.Mode = storage.ModeTeam
		storageConfig.Team = &storage.TursoConfig{
			Enabled:        true,
			URL:            os.Getenv("TURSO_URL"),
			AuthToken:      os.Getenv("TURSO_AUTH_TOKEN"),
			LocalReplica:   getEnvOrDefault("TURSO_LOCAL_REPLICA", "~/.howlerops/team-replica.db"),
			SyncInterval:   getEnvOrDefault("TURSO_SYNC_INTERVAL", "30s"),
			ShareHistory:   true,
			ShareQueries:   true,
			ShareLearnings: true,
			TeamID:         os.Getenv("TEAM_ID"),
		}
	}

	// Create storage manager
	manager, err := storage.NewManager(ctx, storageConfig, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create storage manager: %w", err)
	}

	a.storageManager = manager
	a.logger.WithFields(logrus.Fields{
		"mode":    manager.GetMode().String(),
		"user_id": manager.GetUserID(),
	}).Info("Storage manager initialized")

	return nil
}

// hasConfiguredProvider returns true if the current AI config has at least one provider enabled.
func (a *App) hasConfiguredProvider() bool {
	if a.aiConfig == nil {
		return false
	}

	if a.aiConfig.OpenAI.APIKey != "" {
		return true
	}

	if a.aiConfig.Anthropic.APIKey != "" {
		return true
	}

	if a.aiConfig.Codex.APIKey != "" {
		return true
	}

	if a.aiConfig.Ollama.Endpoint != "" {
		return true
	}

	if a.aiConfig.HuggingFace.Endpoint != "" {
		return true
	}

	if a.aiConfig.ClaudeCode.ClaudePath != "" {
		return true
	}

	return false
}

// applyAIConfiguration rebuilds the AI service using the current runtime configuration.
func (a *App) applyAIConfiguration() error {
	if a.ctx == nil {
		return fmt.Errorf("application context not initialised")
	}

	if a.aiConfig == nil {
		a.aiConfig = ai.DefaultRuntimeConfig()
	}

	if !a.hasConfiguredProvider() {
		a.logger.Info("No AI providers configured; AI features remain disabled")
		if a.aiService != nil {
			if err := a.aiService.Stop(a.ctx); err != nil {
				a.logger.WithError(err).Warn("Failed to stop existing AI service")
			}
			a.aiService = nil
		}
		a.embeddingService = nil
		return fmt.Errorf("no AI providers configured")
	}

	// Shut down any existing instance before reconfiguring
	if a.aiService != nil {
		if err := a.aiService.Stop(a.ctx); err != nil {
			a.logger.WithError(err).Warn("Failed to stop existing AI service during reconfiguration")
		}
		a.aiService = nil
	}

	service, err := ai.NewServiceWithConfig(a.aiConfig, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	if err := service.Start(a.ctx); err != nil {
		return fmt.Errorf("failed to start AI service: %w", err)
	}

	a.aiService = service
	a.logger.Info("AI service configured successfully")

	a.rebuildEmbeddingService()
	return nil
}

// rebuildEmbeddingService establishes the embedding service if OpenAI credentials are available.
func (a *App) rebuildEmbeddingService() {
	a.embeddingService = nil

	if a.aiConfig == nil || a.aiConfig.OpenAI.APIKey == "" {
		return
	}

	model := "text-embedding-3-small"
	provider := rag.NewOpenAIEmbeddingProvider(a.aiConfig.OpenAI.APIKey, model, a.logger)
	a.embeddingService = rag.NewEmbeddingService(provider, a.logger)
	a.logger.WithField("model", model).Info("Embedding service initialised")
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// OnShutdown is called when the app is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	a.logger.Info("HowlerOps desktop application shutting down")

	// Close storage manager
	if a.storageManager != nil {
		if err := a.storageManager.Close(); err != nil {
			a.logger.WithError(err).Error("Failed to close storage manager")
		}
	}

	// Stop AI service
	if a.aiService != nil {
		if err := a.aiService.Stop(ctx); err != nil {
			a.logger.WithError(err).Error("Failed to stop AI service")
		}
	}

	// Close database service
	if a.databaseService != nil {
		a.databaseService.Close()
	}

	// Emit shutdown event
	wailsRuntime.EventsEmit(ctx, "app:shutdown")
}

// CreateConnection creates a new database connection
func (a *App) CreateConnection(req ConnectionRequest) (*ConnectionInfo, error) {
	a.logger.WithFields(logrus.Fields{
		"type":     req.Type,
		"host":     req.Host,
		"port":     req.Port,
		"database": req.Database,
		"username": req.Username,
	}).Info("Creating database connection")

	// Convert request to internal config
	config := database.ConnectionConfig{
		Type:     database.DatabaseType(req.Type),
		Host:     req.Host,
		Port:     req.Port,
		Database: req.Database,
		Username: req.Username,
		Password: req.Password,
		SSLMode:  req.SSLMode,
	}

	// Set default timeout if not provided
	if req.ConnectionTimeout > 0 {
		config.ConnectionTimeout = time.Duration(req.ConnectionTimeout) * time.Second
	} else {
		config.ConnectionTimeout = 30 * time.Second
	}

	// Set default parameters
	if config.Parameters == nil {
		config.Parameters = make(map[string]string)
	}
	for k, v := range req.Parameters {
		config.Parameters[k] = v
	}

	// Create connection
	connection, err := a.databaseService.CreateConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return &ConnectionInfo{
		ID:        connection.ID,
		Type:      string(config.Type),
		Host:      config.Host,
		Port:      config.Port,
		Database:  config.Database,
		Username:  config.Username,
		Active:    connection.Active,
		CreatedAt: connection.CreatedAt,
	}, nil
}

// TestConnection tests a database connection without creating it
func (a *App) TestConnection(req ConnectionRequest) error {
	a.logger.WithFields(logrus.Fields{
		"type":     req.Type,
		"host":     req.Host,
		"port":     req.Port,
		"database": req.Database,
		"username": req.Username,
	}).Info("Testing database connection")

	config := database.ConnectionConfig{
		Type:              database.DatabaseType(req.Type),
		Host:              req.Host,
		Port:              req.Port,
		Database:          req.Database,
		Username:          req.Username,
		Password:          req.Password,
		SSLMode:           req.SSLMode,
		ConnectionTimeout: time.Duration(req.ConnectionTimeout) * time.Second,
		Parameters:        req.Parameters,
	}

	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 10 * time.Second
	}

	return a.databaseService.TestConnection(config)
}

// ListConnections returns all active connections
func (a *App) ListConnections() ([]string, error) {
	return a.databaseService.ListConnections(), nil
}

// RemoveConnection removes a database connection
func (a *App) RemoveConnection(connectionID string) error {
	a.logger.WithField("connection_id", connectionID).Info("Removing database connection")
	return a.databaseService.RemoveConnection(connectionID)
}

// ExecuteQuery executes a SQL query
func (a *App) ExecuteQuery(req QueryRequest) (*QueryResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"connection_id": req.ConnectionID,
		"query_length":  len(req.Query),
	}).Info("Executing query")

	options := &database.QueryOptions{
		Timeout:  30 * time.Second,
		ReadOnly: false,
		Limit:    req.Limit,
	}

	if options.Limit == 0 {
		options.Limit = 1000
	}

	result, err := a.databaseService.ExecuteQuery(req.ConnectionID, req.Query, options)
	if err != nil {
		return &QueryResponse{
			Error: err.Error(),
		}, nil
	}

	return &QueryResponse{
		Columns:  result.Columns,
		Rows:     result.Rows,
		RowCount: result.RowCount,
		Affected: result.Affected,
		Duration: result.Duration.String(),
		Editable: result.Editable,
	}, nil
}

// GetEditableMetadata returns the status of an editable metadata job
func (a *App) GetEditableMetadata(jobID string) (*EditableMetadataJobResponse, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, fmt.Errorf("jobId is required")
	}

	job, err := a.databaseService.GetEditableMetadataJob(jobID)
	if err != nil {
		return nil, err
	}

	response := &EditableMetadataJobResponse{
		ID:           job.ID,
		ConnectionID: job.ConnectionID,
		Status:       job.Status,
		Metadata:     job.Metadata,
		Error:        job.Error,
		CreatedAt:    job.CreatedAt.Format(time.RFC3339Nano),
	}

	if job.Metadata != nil {
		job.Metadata.JobID = job.ID
		job.Metadata.Pending = job.Status == "pending"
	}

	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format(time.RFC3339Nano)
	}

	return response, nil
}

// UpdateQueryRow persists edits made to a query result row
func (a *App) UpdateQueryRow(req QueryRowUpdateRequest) (*QueryRowUpdateResponse, error) {
	if req.ConnectionID == "" {
		return &QueryRowUpdateResponse{
			Success: false,
			Message: "connectionId is required",
		}, nil
	}

	if len(req.PrimaryKey) == 0 {
		return &QueryRowUpdateResponse{
			Success: false,
			Message: "primary key values are required",
		}, nil
	}

	if len(req.Values) == 0 {
		return &QueryRowUpdateResponse{
			Success: false,
			Message: "no changes were provided",
		}, nil
	}

	params := database.UpdateRowParams{
		Schema:        req.Schema,
		Table:         req.Table,
		PrimaryKey:    req.PrimaryKey,
		Values:        req.Values,
		OriginalQuery: req.Query,
		Columns:       req.Columns,
	}

	a.logger.WithFields(logrus.Fields{
		"connection_id": req.ConnectionID,
		"schema":        req.Schema,
		"table":         req.Table,
	}).Info("Applying row update")

	if err := a.databaseService.UpdateRow(req.ConnectionID, params); err != nil {
		a.logger.WithError(err).Error("Row update failed")
		return &QueryRowUpdateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &QueryRowUpdateResponse{
		Success: true,
	}, nil
}

// GetSchemas returns available schemas for a connection
func (a *App) GetSchemas(connectionID string) ([]string, error) {
	return a.databaseService.GetSchemas(connectionID)
}

// GetTables returns tables in a schema
func (a *App) GetTables(connectionID, schema string) ([]TableInfo, error) {
	tables, err := a.databaseService.GetTables(connectionID, schema)
	if err != nil {
		return nil, err
	}

	result := make([]TableInfo, len(tables))
	for i, table := range tables {
		result[i] = TableInfo{
			Schema:    table.Schema,
			Name:      table.Name,
			Type:      table.Type,
			Comment:   table.Comment,
			RowCount:  table.RowCount,
			SizeBytes: table.SizeBytes,
		}
	}

	return result, nil
}

// GetTableStructure returns the structure of a table
func (a *App) GetTableStructure(connectionID, schema, table string) (*database.TableStructure, error) {
	return a.databaseService.GetTableStructure(connectionID, schema, table)
}

// OpenFileDialog opens a file dialog and returns the selected file path
func (a *App) OpenFileDialog() (string, error) {
	return a.fileService.OpenFile(nil)
}

// SaveFileDialog opens a save file dialog and returns the selected file path
func (a *App) SaveFileDialog() (string, error) {
	return a.fileService.SaveFile("query.sql")
}

// ReadFile reads a file and returns its content
func (a *App) ReadFile(filePath string) (string, error) {
	return a.fileService.ReadFile(filePath)
}

// WriteFile writes content to a file
func (a *App) WriteFile(filePath, content string) error {
	return a.fileService.WriteFile(filePath, content)
}

// ShowInfoDialog shows an information dialog
func (a *App) ShowInfoDialog(title, message string) {
	wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// ShowErrorDialog shows an error dialog
func (a *App) ShowErrorDialog(title, message string) {
	wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.ErrorDialog,
		Title:   title,
		Message: message,
	})
}

// ShowQuestionDialog shows a question dialog and returns the result
func (a *App) ShowQuestionDialog(title, message string) (bool, error) {
	result, err := wailsRuntime.MessageDialog(a.ctx, wailsRuntime.MessageDialogOptions{
		Type:    wailsRuntime.QuestionDialog,
		Title:   title,
		Message: message,
	})
	return result == "Yes", err
}

// GetConnectionHealth returns health status for a connection
func (a *App) GetConnectionHealth(connectionID string) (*database.HealthStatus, error) {
	return a.databaseService.GetConnectionHealth(connectionID)
}

// GetDatabaseVersion returns the database version
func (a *App) GetDatabaseVersion(connectionID string) (string, error) {
	info, err := a.databaseService.GetConnectionInfo(connectionID)
	if err != nil {
		return "", err
	}

	if version, ok := info["version"].(string); ok {
		return version, nil
	}

	return "Unknown", nil
}

// ExplainQuery returns query execution plan
func (a *App) ExplainQuery(connectionID, query string) (string, error) {
	return a.databaseService.ExplainQuery(connectionID, query)
}

// GetAppVersion returns the application version
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// GetHomePath returns the user's home directory
func (a *App) GetHomePath() (string, error) {
	return a.fileService.GetHomePath()
}

// ===============================
// Service Delegation Methods
// ===============================

// Database Service Methods
func (a *App) ExecuteQueryStream(connectionID, query string, batchSize int) (string, error) {
	return a.databaseService.ExecuteQueryStream(connectionID, query, batchSize)
}

func (a *App) CancelQueryStream(streamID string) error {
	return a.databaseService.CancelQueryStream(streamID)
}

func (a *App) GetConnectionStats() map[string]database.PoolStats {
	return a.databaseService.GetConnectionStats()
}

func (a *App) HealthCheckAll() map[string]*database.HealthStatus {
	return a.databaseService.HealthCheckAll()
}

func (a *App) GetSupportedDatabaseTypes() []string {
	return a.databaseService.GetSupportedDatabaseTypes()
}

func (a *App) GetDatabaseTypeInfo(dbType string) map[string]interface{} {
	return a.databaseService.GetDatabaseTypeInfo(dbType)
}

// File Service Methods
func (a *App) GetFileInfo(filePath string) (*services.FileInfo, error) {
	return a.fileService.GetFileInfo(filePath)
}

func (a *App) FileExists(filePath string) bool {
	return a.fileService.FileExists(filePath)
}

func (a *App) GetRecentFiles() ([]services.RecentFile, error) {
	return a.fileService.GetRecentFiles()
}

func (a *App) ClearRecentFiles() {
	a.fileService.ClearRecentFiles()
}

func (a *App) RemoveFromRecentFiles(filePath string) {
	a.fileService.RemoveFromRecentFiles(filePath)
}

func (a *App) GetWorkspaceFiles(dirPath string, extensions []string) ([]services.FileInfo, error) {
	return a.fileService.GetWorkspaceFiles(dirPath, extensions)
}

func (a *App) CreateDirectory(dirPath string) error {
	return a.fileService.CreateDirectory(dirPath)
}

func (a *App) DeleteFile(filePath string) error {
	return a.fileService.DeleteFile(filePath)
}

func (a *App) CopyFile(srcPath, destPath string) error {
	return a.fileService.CopyFile(srcPath, destPath)
}

func (a *App) GetTempDir() string {
	return a.fileService.GetTempDir()
}

func (a *App) CreateTempFile(content, prefix, suffix string) (string, error) {
	return a.fileService.CreateTempFile(content, prefix, suffix)
}

func (a *App) GetDownloadsPath() (string, error) {
	return a.fileService.GetDownloadsPath()
}

func (a *App) SaveToDownloads(filename, content string) (string, error) {
	return a.fileService.SaveToDownloads(filename, content)
}

// Keyboard Service Methods
func (a *App) HandleKeyboardEvent(event services.KeyboardEvent) {
	a.keyboardService.HandleKeyboardEvent(event)
}

func (a *App) GetAllKeyboardBindings() map[string]services.KeyboardAction {
	return a.keyboardService.GetAllBindings()
}

func (a *App) GetKeyboardBindingsByCategory() map[string][]services.KeyboardAction {
	return a.keyboardService.GetBindingsByCategory()
}

func (a *App) AddKeyboardBinding(key string, action services.KeyboardAction) {
	a.keyboardService.AddBinding(key, action)
}

func (a *App) RemoveKeyboardBinding(key string) {
	a.keyboardService.RemoveBinding(key)
}

func (a *App) ResetKeyboardBindings() {
	a.keyboardService.ResetToDefaults()
}

func (a *App) ExportKeyboardBindings() map[string]services.KeyboardAction {
	return a.keyboardService.ExportBindings()
}

func (a *App) ImportKeyboardBindings(bindings map[string]services.KeyboardAction) {
	a.keyboardService.ImportBindings(bindings)
}

// ===============================
// Icon Management Methods
// ===============================

// GetAppIcon returns the main application icon
func (a *App) GetAppIcon() ([]byte, error) {
	return iconFS.ReadFile("howlerops-transparent.png")
}

// GetLightIcon returns the light theme icon
func (a *App) GetLightIcon() ([]byte, error) {
	return iconFS.ReadFile("howlerops-light.png")
}

// GetDarkIcon returns the dark theme icon
func (a *App) GetDarkIcon() ([]byte, error) {
	return iconFS.ReadFile("howlerops-dark.png")
}

// ===============================
// AI Provider Test Methods
// ===============================

// AITestRequest represents an AI provider test request
type AITestRequest struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"apiKey,omitempty"`
	Model        string `json:"model,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	Organization string `json:"organization,omitempty"`
	BinaryPath   string `json:"binaryPath,omitempty"`
}

// AITestResponse represents an AI provider test response
type AITestResponse struct {
	Success  bool              `json:"success"`
	Message  string            `json:"message"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

var (
	loginURLRegex       = regexp.MustCompile(`https?://[^\s]+`)
	loginCodeRegex      = regexp.MustCompile(`(?i)(?:code|token|key)[^A-Z0-9]*([A-Z0-9-]{4,})`)
	loginCodeValueRegex = regexp.MustCompile(`^[A-Z0-9-]{4,}$`)
	ansiEscapeRegex     = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\-_]`)
	emailRegex          = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	oscHyperlinkRegex   = regexp.MustCompile(`\x1b]8;;([^\x07\x1b]*)(?:\x07|\x1b\\)([^\x1b]*?)(?:\x1b]8;;(?:\x07|\x1b\\))`)
)

type deviceLoginResult struct {
	Link             string
	UserCode         string
	DeviceCode       string
	Message          string
	RawOutput        string
	OriginalOutput   string
	Err              error
	expectUserCode   bool
	expectDeviceCode bool
}

func startLoginStream(cmd *exec.Cmd) (io.ReadCloser, func(), func([]byte), bool, error) {
	if runtime.GOOS == "windows" {
		reader, cleanup, writeFn, err := startPipeStream(cmd)
		return reader, cleanup, writeFn, false, err
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		reader, cleanup, writeFn, pipeErr := startPipeStream(cmd)
		if pipeErr != nil {
			return nil, nil, nil, false, pipeErr
		}
		return reader, cleanup, writeFn, false, nil
	}

	cleanup := func() {
		_ = ptmx.Close()
	}

	writeFn := func(data []byte) {
		_, _ = ptmx.Write(data)
	}

	return ptmx, cleanup, writeFn, true, nil
}

func startPipeStream(cmd *exec.Cmd) (io.ReadCloser, func(), func([]byte), error) {
	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter

	stdin, err := cmd.StdinPipe()
	if err != nil {
		stdin = nil
	}

	cleanup := func() {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		if stdin != nil {
			_ = stdin.Close()
		}
	}

	writeFn := func(data []byte) {
		if stdin != nil {
			_, _ = stdin.Write(data)
		}
	}

	return pipeReader, cleanup, writeFn, nil
}

func stripANSICodes(input string) string {
	if input == "" {
		return ""
	}
	expanded := expandOSC8Links(input)
	return ansiEscapeRegex.ReplaceAllString(expanded, "")
}

func expandOSC8Links(input string) string {
	if input == "" {
		return ""
	}
	return oscHyperlinkRegex.ReplaceAllStringFunc(input, func(match string) string {
		subs := oscHyperlinkRegex.FindStringSubmatch(match)
		if len(subs) < 3 {
			return ""
		}
		url := strings.TrimSpace(subs[1])
		text := strings.TrimSpace(subs[2])
		if text == "" || text == url {
			return url
		}
		return fmt.Sprintf("%s (%s)", text, url)
	})
}

func updateLoginInfoFromLine(line string, info *deviceLoginResult) {
	clean := stripANSICodes(line)
	trimmed := strings.TrimSpace(clean)
	if trimmed == "" {
		return
	}

	lower := strings.ToLower(trimmed)

	if info.Link == "" {
		if match := loginURLRegex.FindString(trimmed); match != "" {
			info.Link = strings.TrimRight(match, ".,\"')")
		}
	}

	if (strings.Contains(lower, "code") || strings.Contains(lower, "token")) && info.UserCode == "" {
		if m := loginCodeRegex.FindStringSubmatch(strings.ReplaceAll(strings.ToUpper(trimmed), " ", "")); len(m) > 1 {
			info.UserCode = strings.Trim(m[1], ".\"')")
			info.expectUserCode = false
		} else {
			info.expectUserCode = true
		}
	} else {
		tryAssignPendingCode(trimmed, &info.UserCode, &info.expectUserCode)
	}

	if strings.Contains(lower, "device") && info.DeviceCode == "" {
		if m := loginCodeRegex.FindStringSubmatch(strings.ReplaceAll(strings.ToUpper(trimmed), " ", "")); len(m) > 1 {
			info.DeviceCode = strings.Trim(m[1], ".\"')")
			info.expectDeviceCode = false
		} else {
			info.expectDeviceCode = true
		}
	} else {
		tryAssignPendingCode(trimmed, &info.DeviceCode, &info.expectDeviceCode)
	}

	if info.Message == "" && (strings.Contains(lower, "visit") || strings.Contains(lower, "open")) && strings.Contains(lower, "http") {
		info.Message = trimmed
	}

	if strings.HasPrefix(trimmed, "{") {
		applyLoginJSON([]byte(trimmed), info)
	}
}

func applyLoginJSON(data []byte, info *deviceLoginResult) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	if info.Link == "" {
		if v, ok := payload["verification_uri_complete"].(string); ok && v != "" {
			info.Link = v
		} else if v, ok := payload["verification_uri"].(string); ok && v != "" {
			info.Link = v
		} else if v, ok := payload["login_url"].(string); ok && v != "" {
			info.Link = v
		}
	}

	if info.UserCode == "" {
		if v, ok := payload["user_code"].(string); ok && v != "" {
			info.UserCode = v
		} else if v, ok := payload["code"].(string); ok && v != "" {
			info.UserCode = v
		}
	}

	if info.DeviceCode == "" {
		if v, ok := payload["device_code"].(string); ok && v != "" {
			info.DeviceCode = v
		}
	}

	if info.Message == "" {
		if v, ok := payload["message"].(string); ok && v != "" {
			info.Message = v
		}
	}
}

func tryAssignPendingCode(trimmed string, target *string, pending *bool) {
	if !*pending || *target != "" {
		return
	}

	token := strings.TrimSpace(strings.ToUpper(trimmed))
	token = strings.Trim(token, ".\"')")
	if token == "" {
		return
	}

	if loginCodeValueRegex.MatchString(token) {
		*target = token
		*pending = false
	}
}

func isUnknownOptionError(message string) bool {
	if message == "" {
		return false
	}

	lower := strings.ToLower(message)
	return strings.Contains(lower, "unknown option") ||
		strings.Contains(lower, "unknown flag") ||
		strings.Contains(lower, "flag provided but not defined") ||
		strings.Contains(lower, "did you mean")
}

func parseClaudeWhoAmIPlain(output string) *AITestResponse {
	clean := strings.TrimSpace(stripANSICodes(output))
	if clean == "" {
		return &AITestResponse{
			Success: true,
			Message: "Claude CLI responded successfully. Complete the login flow if prompted.",
		}
	}

	lower := strings.ToLower(clean)

	metadata := map[string]string{
		"raw_output": clean,
	}

	if strings.Contains(lower, "not logged") || strings.Contains(lower, "please run") && strings.Contains(lower, "login") {
		return &AITestResponse{
			Success:  false,
			Error:    "Claude CLI is not logged in. Run 'claude login' to link your account.",
			Metadata: metadata,
		}
	}

	if email := emailRegex.FindString(clean); email != "" {
		metadata["email"] = email
		return &AITestResponse{
			Success:  true,
			Message:  fmt.Sprintf("Claude CLI authenticated as %s", email),
			Metadata: metadata,
		}
	}

	if strings.Contains(lower, "logged in") || strings.Contains(lower, "authenticated") {
		return &AITestResponse{
			Success:  true,
			Message:  clean,
			Metadata: metadata,
		}
	}

	return &AITestResponse{
		Success:  true,
		Message:  clean,
		Metadata: metadata,
	}
}

func parseClaudeWhoAmIOutput(output string) *AITestResponse {
	clean := strings.TrimSpace(stripANSICodes(output))
	if clean == "" {
		return &AITestResponse{
			Success: true,
			Message: "Claude CLI responded successfully. Complete the login flow if prompted.",
		}
	}

	var whoami struct {
		LoggedIn bool `json:"loggedIn"`
		Account  struct {
			Email string `json:"email"`
		} `json:"account"`
	}

	if err := json.Unmarshal([]byte(clean), &whoami); err == nil {
		metadata := map[string]string{
			"raw_output": clean,
		}

		if whoami.Account.Email != "" {
			metadata["email"] = whoami.Account.Email
		}

		if !whoami.LoggedIn {
			return &AITestResponse{
				Success:  false,
				Error:    "Claude CLI is not logged in. Run 'claude login' to link your account.",
				Metadata: metadata,
			}
		}

		message := "Claude CLI authenticated"
		if whoami.Account.Email != "" {
			message = fmt.Sprintf("Claude CLI authenticated as %s", whoami.Account.Email)
		}

		return &AITestResponse{
			Success:  true,
			Message:  message,
			Metadata: metadata,
		}
	}

	return parseClaudeWhoAmIPlain(clean)
}

func buildLoginMessage(defaultMessage string, result deviceLoginResult) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = strings.TrimSpace(defaultMessage)
	}

	var lines []string
	if message != "" {
		lines = append(lines, message)
	}

	if result.Link != "" && !strings.Contains(strings.ToLower(message), "http") {
		lines = append(lines, fmt.Sprintf("Verification URL: %s", result.Link))
	}

	if result.UserCode != "" {
		lines = append(lines, fmt.Sprintf("Code: %s", result.UserCode))
	}

	if result.DeviceCode != "" && strings.ToUpper(result.DeviceCode) != strings.ToUpper(result.UserCode) {
		lines = append(lines, fmt.Sprintf("Device Code: %s", result.DeviceCode))
	}

	seen := map[string]struct{}{}
	dedup := make([]string, 0, len(lines))
	for _, line := range lines {
		key := strings.ToLower(strings.TrimSpace(line))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dedup = append(dedup, line)
	}

	if len(dedup) == 0 {
		return defaultMessage
	}

	return strings.Join(dedup, "\n")
}

func sanitizeLoginOutput(output string) string {
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))

	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(stripANSICodes(line))
		if trimmed == "" {
			continue
		}

		if isDecorativeLoginLine(trimmed) {
			continue
		}

		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		filtered = append(filtered, trimmed)
	}

	return strings.Join(filtered, "\n")
}

func isDecorativeLoginLine(line string) bool {
	if line == "" {
		return true
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	isDecorativeChars := true
	for _, r := range trimmed {
		switch r {
		case '│', '╭', '╰', '─', '╮', '╯', '╲', '╱', '╳', '┼', '┌', '┐', '└', '┘', '━', '┃', '┏', '┓', '┗', '┛', '╸', '╹', '╺', '╻':
			continue
		case ' ', '\t':
			continue
		default:
			isDecorativeChars = false
			break
		}
	}
	if isDecorativeChars {
		return true
	}

	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "/help") ||
		strings.Contains(lower, "/status") ||
		strings.Contains(lower, "cwd:") ||
		strings.Contains(lower, "welcome to claude code") ||
		strings.Contains(lower, "mcp server needs auth") ||
		strings.Contains(lower, "? for shortcuts") ||
		strings.Contains(lower, "for shortcuts") && strings.Contains(lower, "/ide") ||
		strings.Contains(lower, "hatching") ||
		strings.Contains(lower, "beaming") ||
		strings.HasPrefix(lower, "try \"") ||
		strings.HasPrefix(lower, ">") && !strings.Contains(lower, "http") && !strings.Contains(lower, "code") && !strings.Contains(lower, "device") ||
		strings.HasPrefix(lower, "·") && !strings.Contains(lower, "http") && !strings.Contains(lower, "code") && !strings.Contains(lower, "device") ||
		strings.EqualFold(trimmed, "code") {
		return true
	}

	return false
}

func ensureLoginInfoFromRaw(raw string, info *deviceLoginResult) {
	if raw == "" || info == nil {
		return
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		clean := strings.TrimSpace(stripANSICodes(line))
		if clean == "" {
			continue
		}
		updateLoginInfoFromLine(clean, info)
		if info.Link != "" && info.UserCode != "" && info.DeviceCode != "" {
			break
		}
	}

	if info.Link == "" {
		if url := extractLoginURL(raw); url != "" {
			info.Link = url
		}
	}
}

func extractLoginURL(text string) string {
	if text == "" {
		return ""
	}
	if match := loginURLRegex.FindString(text); match != "" {
		return strings.TrimRight(match, ".,\"')")
	}
	return ""
}

func (a *App) runClaudeLoginJSON(binaryPath string, defaultMessage string) (*AITestResponse, bool) {
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "/login", "--json")
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		if isUnknownOptionError(message) || strings.Contains(strings.ToLower(message), "unknown command") {
			return nil, false
		}
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Claude CLI JSON login failed: %s", message),
		}, true
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return &AITestResponse{
			Success: false,
			Error:   "Claude CLI JSON login returned no data",
		}, true
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		metadata := map[string]string{"raw_output": output}
		return &AITestResponse{
			Success:  false,
			Error:    fmt.Sprintf("Failed to parse Claude CLI JSON login output: %v", err),
			Metadata: metadata,
		}, true
	}

	info := deviceLoginResult{OriginalOutput: output, RawOutput: output}
	applyLoginJSON([]byte(output), &info)
	ensureLoginInfoFromRaw(output, &info)

	message := buildLoginMessage(defaultMessage, info)
	if message == "" {
		message = defaultMessage
	}

	metadata := loginMetadata(info)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["raw_output"] = output

	return &AITestResponse{
		Success:  true,
		Message:  message,
		Metadata: metadata,
	}, true
}

func (a *App) runDeviceLoginCommand(binaryPath string, args []string, provider string, defaultMessage string) *AITestResponse {
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(baseCtx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Env = os.Environ()

	reader, cleanupStream, writeInput, alreadyStarted, err := startLoginStream(cmd)
	if err != nil {
		a.logger.WithError(err).WithField("provider", provider).Error("Failed to configure login stream")
		return &AITestResponse{Success: false, Error: "unable to prepare login stream"}
	}

	if !alreadyStarted {
		if err := cmd.Start(); err != nil {
			if cleanupStream != nil {
				cleanupStream()
			}
			a.logger.WithError(err).WithField("provider", provider).Error("Failed to start login command")
			return &AITestResponse{Success: false, Error: fmt.Sprintf("unable to start %s login: %v", provider, err)}
		}
	}

	if writeInput != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			writeInput([]byte("\n"))
		}()
	}

	resultChan := make(chan deviceLoginResult, 1)
	infoLatest := &deviceLoginResult{}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if cleanupStream != nil {
				cleanupStream()
			}
		})
	}

	go func() {
		defer cleanup()

		var builder strings.Builder
		var rawAll strings.Builder
		info := deviceLoginResult{}
		sent := false
		residual := ""
		buf := make([]byte, 2048)

		emit := func(text string) {
			clean := strings.TrimSpace(stripANSICodes(text))
			if clean == "" {
				return
			}

			if rawAll.Len() > 0 {
				rawAll.WriteString("\n")
			}
			rawAll.WriteString(clean)
			updateLoginInfoFromLine(clean, &info)
			info.OriginalOutput = rawAll.String()
			*infoLatest = info

			if isDecorativeLoginLine(clean) {
				return
			}

			keep := false
			for _, r := range clean {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					keep = true
					break
				}
			}
			if !keep && !strings.Contains(clean, "http") {
				return
			}

			lower := strings.ToLower(clean)
			if strings.HasPrefix(clean, "╭") ||
				strings.HasPrefix(clean, "╰") ||
				strings.HasPrefix(clean, "│") ||
				strings.HasPrefix(clean, "─") {
				if !(strings.Contains(lower, "http") ||
					strings.Contains(lower, "code") ||
					strings.Contains(lower, "token") ||
					strings.Contains(lower, "visit") ||
					strings.Contains(lower, "open")) {
					return
				}
			}

			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(clean)
			info.RawOutput = builder.String()
			*infoLatest = info
			if !sent && (info.Link != "" || info.UserCode != "" || info.DeviceCode != "") {
				select {
				case resultChan <- info:
					sent = true
				default:
				}
			}
		}

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				cleanChunk := stripANSICodes(chunk)
				if cleanChunk != "" {
					residual += cleanChunk
				}

				normalized := strings.ReplaceAll(residual, "\r\n", "\n")
				normalized = strings.ReplaceAll(normalized, "\r", "\n")
				parts := strings.Split(normalized, "\n")
				if !strings.HasSuffix(normalized, "\n") {
					residual = parts[len(parts)-1]
					parts = parts[:len(parts)-1]
				} else {
					residual = ""
				}

				for _, line := range parts {
					emit(line)
				}
			}

			if err != nil {
				if strings.TrimSpace(residual) != "" {
					emit(residual)
					residual = ""
				}

				if err != io.EOF {
					info.Err = err
					info.RawOutput = builder.String()
					*infoLatest = info
				}
				break
			}
		}

		if waitErr := cmd.Wait(); waitErr != nil {
			info.Err = waitErr
		}

		info.OriginalOutput = rawAll.String()
		info.RawOutput = builder.String()
		*infoLatest = info

		if !sent {
			select {
			case resultChan <- info:
			default:
			}
		}
	}()

	var result deviceLoginResult
	select {
	case result = <-resultChan:
	case <-time.After(120 * time.Second):
		_ = cmd.Process.Kill()
		info := *infoLatest
		if info.Message == "" {
			info.Message = "Login command timed out before producing instructions"
		}
		cleanup()
		return &AITestResponse{
			Success:  false,
			Error:    info.Message,
			Metadata: loginMetadata(info),
		}
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		info := *infoLatest
		if info.Message == "" {
			info.Message = "Login command cancelled"
		}
		cleanup()
		return &AITestResponse{
			Success:  false,
			Error:    info.Message,
			Metadata: loginMetadata(info),
		}
	}

	if cmd.ProcessState == nil {
		_ = cmd.Process.Kill()
	}

	originalRaw := result.OriginalOutput
	if originalRaw == "" {
		originalRaw = result.RawOutput
	}

	sanitizedOutput := sanitizeLoginOutput(originalRaw)
	result.RawOutput = sanitizedOutput
	ensureLoginInfoFromRaw(originalRaw, &result)
	if result.Link == "" {
		ensureLoginInfoFromRaw(result.RawOutput, &result)
	}

	message := buildLoginMessage(defaultMessage, result)
	if message == "" && result.RawOutput != "" {
		message = result.RawOutput
	}

	result.RawOutput = message
	metadata := loginMetadata(result)

	if result.Link == "" {
		if a.logger != nil {
			a.logger.WithFields(logrus.Fields{
				"provider":         provider,
				"raw_output":       originalRaw,
				"sanitized_output": sanitizedOutput,
			}).Warn("Claude login output missing verification URL")
		}
	}

	if result.Err != nil && result.Link == "" && result.UserCode == "" && result.DeviceCode == "" {
		errMsg := result.Err.Error()
		if message != "" {
			errMsg = message
		}
		cleanup()
		return &AITestResponse{Success: false, Error: errMsg, Metadata: metadata}
	}

	cleanup()
	return &AITestResponse{Success: true, Message: message, Metadata: metadata}
}

func loginMetadata(info deviceLoginResult) map[string]string {
	metadata := map[string]string{}
	if info.Link != "" {
		metadata["verification_url"] = info.Link
	}
	if info.UserCode != "" {
		metadata["user_code"] = info.UserCode
	}
	if info.DeviceCode != "" {
		metadata["device_code"] = info.DeviceCode
	}
	if info.RawOutput != "" {
		metadata["raw_output"] = info.RawOutput
	}
	if info.OriginalOutput != "" {
		metadata["original_raw_output"] = info.OriginalOutput
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

// TestOpenAIConnection tests OpenAI provider connection
func (a *App) TestOpenAIConnection(apiKey, model string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider": "openai",
		"model":    model,
	}).Info("Testing OpenAI connection")

	if apiKey == "" {
		return &AITestResponse{
			Success: false,
			Error:   "OpenAI API key is required",
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid OpenAI API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to OpenAI
	return &AITestResponse{
		Success: true,
		Message: "OpenAI connection test successful",
	}
}

// TestAnthropicConnection tests Anthropic provider connection
func (a *App) TestAnthropicConnection(apiKey, model string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider": "anthropic",
		"model":    model,
	}).Info("Testing Anthropic connection")

	if apiKey == "" {
		return &AITestResponse{
			Success: false,
			Error:   "Anthropic API key is required",
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid Anthropic API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to Anthropic
	return &AITestResponse{
		Success: true,
		Message: "Anthropic connection test successful",
	}
}

// TestOllamaConnection tests Ollama provider connection
func (a *App) TestOllamaConnection(endpoint, model string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider": "ollama",
		"endpoint": endpoint,
		"model":    model,
	}).Info("Testing Ollama connection")

	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// Try to connect to Ollama endpoint
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to Ollama at %s: %v", endpoint, err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Ollama endpoint returned status %d", resp.StatusCode),
		}
	}

	return &AITestResponse{
		Success: true,
		Message: "Ollama connection test successful",
	}
}

// TestClaudeCodeConnection tests Claude Code provider connection
func (a *App) TestClaudeCodeConnection(binaryPath, model string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider":   "claudecode",
		"binaryPath": binaryPath,
		"model":      model,
	}).Info("Testing Claude Code connection")

	if binaryPath == "" {
		binaryPath = "claude"
	}

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	runCommand := func(args ...string) (string, string, error) {
		cmd := exec.CommandContext(ctx, binaryPath, args...)
		cmd.Env = os.Environ()

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
	}

	output, errMsg, err := runCommand("whoami", "--json")
	if err != nil && isUnknownOptionError(errMsg) {
		output, errMsg, err = runCommand("whoami", "--format", "json")
	}

	if err != nil && isUnknownOptionError(errMsg) {
		plainOutput, plainErrMsg, plainErr := runCommand("whoami")
		if plainErr != nil && strings.TrimSpace(plainOutput) == "" {
			if plainErrMsg == "" {
				plainErrMsg = plainErr.Error()
			}
			return &AITestResponse{
				Success: false,
				Error:   fmt.Sprintf("Claude CLI check failed: %s", plainErrMsg),
			}
		}
		return parseClaudeWhoAmIOutput(plainOutput)
	}

	if err != nil {
		if output != "" {
			return parseClaudeWhoAmIOutput(output)
		}

		if errMsg == "" {
			errMsg = err.Error()
		}

		return &AITestResponse{
			Success: false,
			Error:   fmt.Sprintf("Claude CLI check failed: %s", errMsg),
		}
	}

	return parseClaudeWhoAmIOutput(output)
}

// StartClaudeCodeLogin begins the Claude CLI login flow.
func (a *App) StartClaudeCodeLogin(binaryPath string) *AITestResponse {
	a.logger.WithField("binaryPath", binaryPath).Info("Launching Claude CLI login flow")

	if binaryPath == "" {
		binaryPath = "claude"
	}

	defaultMessage := "Open the link and authorise Claude Code using the displayed code."
	if response, handled := a.runClaudeLoginJSON(binaryPath, defaultMessage); handled {
		if response != nil {
			return response
		}
	}

	return a.runDeviceLoginCommand(binaryPath, []string{"/login"}, "claudecode", defaultMessage)
}

// StartCodexLogin begins the OpenAI CLI login flow for Codex access.
func (a *App) StartCodexLogin(binaryPath string) *AITestResponse {
	a.logger.WithField("binaryPath", binaryPath).Info("Launching OpenAI CLI login flow")

	if binaryPath == "" {
		binaryPath = "openai"
	}

	return a.runDeviceLoginCommand(binaryPath, []string{"login"}, "codex", "Open the link and authorise OpenAI using the displayed code.")
}

// TestCodexConnection tests Codex provider connection
func (a *App) TestCodexConnection(apiKey, model, organization string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider":     "codex",
		"model":        model,
		"organization": organization,
	}).Info("Testing Codex connection")

	if apiKey == "" {
		return &AITestResponse{
			Success: false,
			Error:   "Codex API key is required",
		}
	}

	// Basic validation - check if API key format is correct
	if !strings.HasPrefix(apiKey, "sk-") {
		return &AITestResponse{
			Success: false,
			Error:   "Invalid Codex API key format",
		}
	}

	// For now, return success for valid-looking keys
	// TODO: Implement actual API call to OpenAI Codex
	metadata := map[string]string{}
	if organization != "" {
		metadata["organization"] = organization
	}

	return &AITestResponse{
		Success:  true,
		Message:  "Codex connection test successful",
		Metadata: metadata,
	}
}

// TestHuggingFaceConnection tests HuggingFace provider connection
func (a *App) TestHuggingFaceConnection(endpoint, model string) *AITestResponse {
	a.logger.WithFields(logrus.Fields{
		"provider": "huggingface",
		"endpoint": endpoint,
		"model":    model,
	}).Info("Testing HuggingFace connection")

	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// For HuggingFace via Ollama, test the Ollama endpoint
	return a.TestOllamaConnection(endpoint, model)
}

// ===============================
// AI/RAG Methods
// ===============================

// GenerateSQLFromNaturalLanguage generates SQL from a natural language prompt
func (a *App) GenerateSQLFromNaturalLanguage(req NLQueryRequest) (*GeneratedSQLResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"prompt":       req.Prompt,
		"connectionId": req.ConnectionID,
		"hasContext":   req.Context != "",
	}).Info("Generating SQL from natural language")

	if a.aiService == nil {
		if err := a.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if a.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	// Build comprehensive schema context
	var schemaContext string

	// Check if this is a multi-database query based on context
	isMultiDB := strings.Contains(req.Context, "Multi-DB Mode") || strings.Contains(req.Context, "@connection_name")

	if isMultiDB {
		// Multi-database mode: context already contains comprehensive schema info from frontend
		schemaContext = req.Context

		// Add additional instructions for multi-DB SQL generation
		schemaContext += "\n\nIMPORTANT SQL Generation Rules for Multi-Database Mode:\n"
		schemaContext += "1. Use @connection_name.table_name syntax for all table references\n"
		schemaContext += "2. Use @connection_name.schema_name.table_name for non-public schemas\n"
		schemaContext += "3. Table aliases are recommended for readability\n"
		schemaContext += "4. Cross-database JOINs are supported\n"
		schemaContext += "5. Ensure connection names match exactly (case-sensitive)\n"
	} else if req.ConnectionID != "" {
		// Single database mode: build detailed schema context for the active connection
		detailedContext := a.buildDetailedSchemaContext(req.ConnectionID)

		if detailedContext != "" {
			schemaContext = detailedContext
		}

		// Add custom context if provided
		if req.Context != "" {
			if schemaContext != "" {
				schemaContext += "\n\n"
			}
			schemaContext += req.Context
		}
	} else {
		// No connection specified, use provided context only
		schemaContext = req.Context
	}

	// Enhance prompt with mode-specific instructions
	enhancedPrompt := req.Prompt
	if isMultiDB {
		enhancedPrompt = fmt.Sprintf(
			"Generate a SQL query for the following request in MULTI-DATABASE mode. "+
				"Use @connection_name.table_name syntax for all tables. "+
				"Request: %s",
			req.Prompt,
		)
	}

	// Generate SQL using AI service with provider configuration
	request := &ai.SQLRequest{
		Prompt:      enhancedPrompt,
		Schema:      schemaContext,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	result, err := a.aiService.GenerateSQLWithRequest(a.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	sanitizeSQLResponse(result)

	return &GeneratedSQLResponse{
		SQL:         strings.TrimSpace(result.SQL),
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		Warnings:    []string{}, // TODO: Add warnings if confidence is low
	}, nil
}

// GenericChat handles generic conversational AI requests without SQL-specific expectations
func (a *App) GenericChat(req GenericChatRequest) (*GenericChatResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"hasContext": req.Context != "",
		"provider":   req.Provider,
		"model":      req.Model,
	}).Info("Handling generic AI chat request")

	if a.aiService == nil {
		if err := a.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if a.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	chatReq := &ai.ChatRequest{
		Prompt:      req.Prompt,
		Context:     req.Context,
		System:      req.System,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Metadata:    req.Metadata,
	}

	response, err := a.aiService.Chat(a.ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chat response: %w", err)
	}

	return &GenericChatResponse{
		Content:    response.Content,
		Provider:   response.Provider,
		Model:      response.Model,
		TokensUsed: response.TokensUsed,
		Metadata:   response.Metadata,
	}, nil
}

// buildDetailedSchemaContext constructs a concise schema summary with column details for a connection.
func (a *App) buildDetailedSchemaContext(connectionID string) string {
	const (
		maxTablesPerSchema = 10
		maxTotalTables     = 40
		maxColumnsPerTable = 25
	)

	schemas, err := a.databaseService.GetSchemas(connectionID)
	if err != nil {
		a.logger.WithError(err).WithField("connection_id", connectionID).
			Warn("Failed to load schemas for AI context")
		return ""
	}

	if len(schemas) == 0 {
		return ""
	}

	sort.Strings(schemas)

	var builder strings.Builder
	builder.WriteString("Database Schema Information:\n")

	totalTables := 0
	for _, schemaName := range schemas {
		if totalTables >= maxTotalTables {
			break
		}

		tables, err := a.databaseService.GetTables(connectionID, schemaName)
		if err != nil {
			a.logger.WithError(err).WithFields(logrus.Fields{
				"connection_id": connectionID,
				"schema":        schemaName,
			}).Warn("Failed to load tables for AI context")
			continue
		}

		if len(tables) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("\nSchema: %s\n", schemaName))

		tableLimit := len(tables)
		if tableLimit > maxTablesPerSchema {
			tableLimit = maxTablesPerSchema
		}

		for i := 0; i < tableLimit && totalTables < maxTotalTables; i++ {
			table := tables[i]
			tableType := strings.ToLower(table.Type)
			if tableType == "" {
				tableType = "table"
			}

			builder.WriteString(fmt.Sprintf("Table: %s (%s)\n", table.Name, tableType))

			structure, err := a.databaseService.GetTableStructure(connectionID, schemaName, table.Name)
			if err != nil {
				a.logger.WithError(err).WithFields(logrus.Fields{
					"connection_id": connectionID,
					"schema":        schemaName,
					"table":         table.Name,
				}).Debug("Failed to load table structure for AI context")
				totalTables++
				continue
			}

			if structure != nil && len(structure.Columns) > 0 {
				columns := formatColumnsForAI(structure.Columns, maxColumnsPerTable)
				builder.WriteString("Columns: " + strings.Join(columns, ", ") + "\n")
			}

			totalTables++
		}

		if len(tables) > tableLimit {
			builder.WriteString(fmt.Sprintf("... %d more tables in schema %s omitted for brevity\n", len(tables)-tableLimit, schemaName))
		}
	}

	return strings.TrimSpace(builder.String())
}

// formatColumnsForAI shortens column metadata for prompt consumption.
func formatColumnsForAI(columns []database.ColumnInfo, maxColumns int) []string {
	formatted := make([]string, 0, len(columns))

	for _, column := range columns {
		columnType := strings.ToLower(column.DataType)
		if columnType == "" {
			columnType = "unknown"
		}

		attributes := make([]string, 0, 3)
		if column.PrimaryKey {
			attributes = append(attributes, "pk")
		}
		if column.Unique {
			attributes = append(attributes, "unique")
		}
		if !column.Nullable {
			attributes = append(attributes, "not null")
		}

		if len(attributes) > 0 {
			columnType = fmt.Sprintf("%s %s", columnType, strings.Join(attributes, "/"))
		}

		formatted = append(formatted, fmt.Sprintf("%s (%s)", column.Name, columnType))

		if maxColumns > 0 && len(formatted) >= maxColumns {
			break
		}
	}

	if maxColumns > 0 && len(columns) > maxColumns {
		formatted = append(formatted, "... additional columns omitted")
	}

	return formatted
}

// sanitizeSQLResponse removes duplicate SQL blocks and strips code from explanations to reduce UI noise.
func sanitizeSQLResponse(resp *ai.SQLResponse) {
	if resp == nil {
		return
	}

	resp.SQL = deduplicateSequentialSQL(resp.SQL)
	resp.SQL = strings.TrimSpace(resp.SQL)

	resp.Explanation = sanitizeExplanation(resp.Explanation, resp.SQL)
}

// deduplicateSequentialSQL collapses responses where the model repeats the same SQL twice.
func deduplicateSequentialSQL(sql string) string {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return trimmed
	}

	halfStart := len(trimmed) / 2
	for i := halfStart; i < len(trimmed); i++ {
		if trimmed[i] != '\n' && trimmed[i] != '\r' {
			continue
		}

		first := strings.TrimSpace(trimmed[:i])
		second := strings.TrimSpace(trimmed[i:])
		if first != "" && first == second {
			return first
		}
	}

	return trimmed
}

// sanitizeExplanation strips code blocks and duplicate SQL from the explanation text.
func sanitizeExplanation(explanation string, sql string) string {
	if explanation == "" {
		return ""
	}

	cleaned := removeCodeBlocks(explanation)

	if sql != "" {
		cleaned = strings.ReplaceAll(cleaned, sql, "")
		// Also remove compressed versions of the SQL where extra whitespace may have been collapsed.
		compressedSQL := strings.Join(strings.Fields(sql), " ")
		if compressedSQL != "" {
			cleaned = strings.ReplaceAll(cleaned, compressedSQL, "")
		}
	}

	return strings.TrimSpace(cleaned)
}

// removeCodeBlocks removes fenced code blocks (``` ... ```) from a string.
func removeCodeBlocks(text string) string {
	result := text

	for {
		start := strings.Index(result, "```")
		if start == -1 {
			break
		}

		end := strings.Index(result[start+3:], "```")
		if end == -1 {
			// Unclosed code block; drop everything after start
			result = result[:start]
			break
		}

		closing := start + 3 + end + 3
		if closing > len(result) {
			closing = len(result)
		}

		result = result[:start] + result[closing:]
	}

	return strings.TrimSpace(result)
}

// FixSQLError attempts to fix a SQL error
func (a *App) FixSQLError(query string, error string, connectionID string) (*FixedSQLResponse, error) {
	return a.FixSQLErrorWithOptions(FixSQLRequest{
		Query:        query,
		Error:        error,
		ConnectionID: connectionID,
	})
}

// OptimizeQuery optimizes a SQL query
func (a *App) OptimizeQuery(query string, connectionID string) (*OptimizationResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"query":        query,
		"connectionId": connectionID,
	}).Info("Optimizing query")

	if a.aiService == nil {
		if err := a.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if a.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	result, err := a.aiService.OptimizeQuery(a.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize query: %w", err)
	}

	return &OptimizationResponse{
		SQL:              result.OptimizedSQL,
		EstimatedSpeedup: result.Impact,
		Explanation:      result.Explanation,
		Suggestions:      []Suggestion{},
	}, nil
}

// FixSQLErrorWithOptions applies AI-based fixes with provider-specific configuration
func (a *App) FixSQLErrorWithOptions(req FixSQLRequest) (*FixedSQLResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"query":        req.Query,
		"error":        req.Error,
		"connectionId": req.ConnectionID,
		"provider":     req.Provider,
		"model":        req.Model,
	}).Info("Fixing SQL error with options")

	if a.aiService == nil {
		if err := a.applyAIConfiguration(); err != nil {
			return nil, fmt.Errorf("AI service not configured: %w", err)
		}
	}

	if a.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if strings.TrimSpace(req.Error) == "" {
		return nil, fmt.Errorf("error message cannot be empty")
	}

	isMultiDB := strings.Contains(req.Query, "@") && (strings.Contains(req.Error, "multi-database") || strings.Contains(req.Error, "@connection_name"))

	enhancedError := req.Error
	if isMultiDB {
		enhancedError = fmt.Sprintf(
			"%s\n\n"+
				"Context: This appears to be a multi-database query. "+
				"Ensure that:\n"+
				"1. Table references use @connection_name.table_name syntax\n"+
				"2. Connection names are spelled correctly (case-sensitive)\n"+
				"3. All referenced connections are properly connected\n"+
				"4. Schema names are included for non-public schemas (@conn.schema.table)",
			req.Error,
		)
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	model := strings.TrimSpace(req.Model)

	aiRequest := &ai.SQLRequest{
		Query:       req.Query,
		Error:       enhancedError,
		Schema:      req.Context,
		Provider:    provider,
		Model:       model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	result, err := a.aiService.FixQueryWithRequest(a.ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fix query: %w", err)
	}

	return &FixedSQLResponse{
		SQL:         result.SQL,
		Explanation: result.Explanation,
		Changes:     []string{"Fixed query based on error message"},
	}, nil
}

// SaveAIMemorySessions persists AI memory sessions and indexes them for recall
func (a *App) SaveAIMemorySessions(sessions []AIMemorySessionPayload) error {
	if a.storageManager == nil {
		return fmt.Errorf("storage manager not initialized")
	}

	previousSessions, err := a.LoadAIMemorySessions()
	if err != nil {
		return err
	}

	data, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal AI memory sessions: %w", err)
	}

	if err := a.storageManager.SetSetting(a.ctx, "ai_memory_sessions", string(data)); err != nil {
		return fmt.Errorf("failed to save AI memory sessions: %w", err)
	}

	if err := a.indexAIMemorySessions(a.ctx, sessions); err != nil {
		a.logger.WithError(err).Warn("Failed to index AI memory sessions")
	}

	a.pruneAIMemoryDocuments(previousSessions, sessions)

	return nil
}

// LoadAIMemorySessions retrieves previously stored AI memory sessions
func (a *App) LoadAIMemorySessions() ([]AIMemorySessionPayload, error) {
	if a.storageManager == nil {
		return []AIMemorySessionPayload{}, nil
	}

	value, err := a.storageManager.GetSetting(a.ctx, "ai_memory_sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to load AI memory sessions: %w", err)
	}

	if strings.TrimSpace(value) == "" {
		return []AIMemorySessionPayload{}, nil
	}

	var sessions []AIMemorySessionPayload
	if err := json.Unmarshal([]byte(value), &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode AI memory sessions: %w", err)
	}

	return sessions, nil
}

// ClearAIMemorySessions removes stored AI memory sessions
func (a *App) ClearAIMemorySessions() error {
	if a.storageManager == nil {
		return nil
	}

	if err := a.storageManager.DeleteSetting(a.ctx, "ai_memory_sessions"); err != nil {
		return fmt.Errorf("failed to clear AI memory sessions: %w", err)
	}

	return nil
}

// RecallAIMemorySessions returns the most relevant stored memories for the given prompt
func (a *App) RecallAIMemorySessions(prompt string, limit int) ([]AIMemoryRecallResult, error) {
	if strings.TrimSpace(prompt) == "" || limit == 0 {
		return []AIMemoryRecallResult{}, nil
	}

	if limit < 0 {
		limit = 5
	}
	if limit == 0 {
		limit = 5
	}

	if a.embeddingService == nil || a.storageManager == nil {
		return []AIMemoryRecallResult{}, nil
	}

	embedding, err := a.embeddingService.EmbedText(a.ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to embed prompt for memory recall: %w", err)
	}

	docs, err := a.storageManager.SearchDocuments(a.ctx, embedding, &storage.DocumentFilters{
		Type:  string(rag.DocumentTypeMemory),
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search AI memories: %w", err)
	}

	results := make([]AIMemoryRecallResult, 0, len(docs))
	for _, doc := range docs {
		sessionID := ""
		title := ""
		summary := ""

		if doc.Metadata != nil {
			if val, ok := doc.Metadata["session_id"].(string); ok {
				sessionID = val
			}
			if val, ok := doc.Metadata["title"].(string); ok {
				title = val
			}
			if val, ok := doc.Metadata["summary"].(string); ok {
				summary = val
			}
		}

		results = append(results, AIMemoryRecallResult{
			SessionID: sessionID,
			Title:     title,
			Summary:   summary,
			Content:   doc.Content,
			Score:     doc.Score,
		})
	}

	return results, nil
}

func (a *App) indexAIMemorySessions(ctx context.Context, sessions []AIMemorySessionPayload) error {
	if a.embeddingService == nil || a.storageManager == nil {
		return nil
	}

	for _, session := range sessions {
		doc, err := a.buildMemoryDocument(ctx, session)
		if err != nil {
			a.logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to build memory document")
			continue
		}
		if doc == nil {
			continue
		}

		if err := a.storageManager.IndexDocument(ctx, doc); err != nil {
			a.logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to index AI memory document")
		}
	}

	return nil
}

func (a *App) buildMemoryDocument(ctx context.Context, session AIMemorySessionPayload) (*storage.Document, error) {
	if len(session.Messages) == 0 {
		return nil, nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Session: %s\n", session.Title))
	if session.Summary != "" {
		builder.WriteString(fmt.Sprintf("Summary: %s\n", session.Summary))
	}

	const maxMessages = 6
	start := len(session.Messages) - maxMessages
	if start < 0 {
		start = 0
	}

	builder.WriteString("Recent conversation:\n")
	for _, msg := range session.Messages[start:] {
		builder.WriteString(fmt.Sprintf("[%s] %s\n", strings.ToUpper(msg.Role), msg.Content))
	}

	content := builder.String()
	embedding, err := a.embeddingService.EmbedText(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to embed AI memory: %w", err)
	}

	createdAt := time.UnixMilli(session.CreatedAt)
	if session.CreatedAt == 0 {
		createdAt = time.Now()
	}
	updatedAt := time.UnixMilli(session.UpdatedAt)
	if session.UpdatedAt == 0 {
		updatedAt = createdAt
	}

	metadata := map[string]interface{}{
		"session_id":    session.ID,
		"title":         session.Title,
		"summary":       session.Summary,
		"message_count": len(session.Messages),
	}

	return &storage.Document{
		ID:           fmt.Sprintf("ai_memory:%s", session.ID),
		ConnectionID: "",
		Type:         string(rag.DocumentTypeMemory),
		Content:      content,
		Embedding:    embedding,
		Metadata:     metadata,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// DeleteAIMemorySession removes a single session by ID
func (a *App) DeleteAIMemorySession(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	sessions, err := a.LoadAIMemorySessions()
	if err != nil {
		return err
	}

	filtered := make([]AIMemorySessionPayload, 0, len(sessions))
	for _, session := range sessions {
		if session.ID != sessionID {
			filtered = append(filtered, session)
		}
	}

	if len(filtered) == len(sessions) {
		return fmt.Errorf("session not found")
	}

	if err := a.SaveAIMemorySessions(filtered); err != nil {
		return err
	}

	if a.storageManager != nil {
		docID := fmt.Sprintf("ai_memory:%s", sessionID)
		if err := a.storageManager.DeleteDocument(a.ctx, docID); err != nil {
			a.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to delete memory document")
		}
	}

	return nil
}

func (a *App) pruneAIMemoryDocuments(previous, current []AIMemorySessionPayload) {
	if a.storageManager == nil {
		return
	}

	currentSet := make(map[string]struct{}, len(current))
	for _, session := range current {
		currentSet[session.ID] = struct{}{}
	}

	for _, session := range previous {
		if _, exists := currentSet[session.ID]; !exists {
			docID := fmt.Sprintf("ai_memory:%s", session.ID)
			if err := a.storageManager.DeleteDocument(a.ctx, docID); err != nil {
				a.logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to delete AI memory document")
			}
		}
	}
}

// GetQuerySuggestions provides autocomplete suggestions for a partial query
func (a *App) GetQuerySuggestions(partialQuery string, connectionID string) ([]Suggestion, error) {
	a.logger.WithFields(logrus.Fields{
		"query":        partialQuery,
		"connectionId": connectionID,
	}).Debug("Getting query suggestions")

	// TODO: Integrate with backend-go AI service when properly exposed
	return []Suggestion{}, nil
}

// SuggestVisualization suggests appropriate visualizations for query results
func (a *App) SuggestVisualization(resultData ResultData) (*VizSuggestion, error) {
	a.logger.WithFields(logrus.Fields{
		"columns":  len(resultData.Columns),
		"rowCount": resultData.RowCount,
	}).Debug("Suggesting visualization")

	// TODO: Integrate with backend-go AI service when properly exposed
	return nil, fmt.Errorf("AI service not yet integrated")
}

// GetAIProviderStatus returns the status of available AI providers
func (a *App) GetAIProviderStatus() (map[string]ProviderStatus, error) {
	a.logger.Debug("Getting AI provider status")

	if a.aiService == nil {
		statuses := map[string]ProviderStatus{
			"openai":      {Name: "OpenAI", Available: false, Error: "Configure this provider in Settings"},
			"anthropic":   {Name: "Anthropic", Available: false, Error: "Configure this provider in Settings"},
			"claudecode":  {Name: "Claude Code", Available: false, Error: "Configure this provider in Settings"},
			"codex":       {Name: "Codex", Available: false, Error: "Configure this provider in Settings"},
			"ollama":      {Name: "Ollama", Available: false, Error: "Configure this provider in Settings"},
			"huggingface": {Name: "HuggingFace", Available: false, Error: "Configure this provider in Settings"},
		}

		if a.aiConfig != nil {
			if a.aiConfig.OpenAI.APIKey != "" {
				statuses["openai"] = ProviderStatus{Name: "OpenAI", Available: true}
			}
			if a.aiConfig.Anthropic.APIKey != "" {
				statuses["anthropic"] = ProviderStatus{Name: "Anthropic", Available: true}
			}
			if a.aiConfig.Codex.APIKey != "" {
				statuses["codex"] = ProviderStatus{Name: "Codex", Available: true}
			}
			if a.aiConfig.ClaudeCode.ClaudePath != "" {
				statuses["claudecode"] = ProviderStatus{Name: "Claude Code", Available: true}
			}
			if a.aiConfig.Ollama.Endpoint != "" {
				statuses["ollama"] = ProviderStatus{Name: "Ollama", Available: true}
			}
			if a.aiConfig.HuggingFace.Endpoint != "" {
				statuses["huggingface"] = ProviderStatus{Name: "HuggingFace", Available: true}
			}
		}

		return statuses, nil
	}

	// Get provider statuses from AI service
	providers, err := a.aiService.GetProviders(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider status: %w", err)
	}

	// Convert to map
	result := make(map[string]ProviderStatus)
	for _, p := range providers {
		result[strings.ToLower(p.Name)] = ProviderStatus{
			Name:      p.Name,
			Available: p.Available,
			Error:     "",
		}
	}

	// Fill in missing providers as unavailable
	allProviders := []string{"openai", "anthropic", "claudecode", "codex", "ollama", "huggingface"}
	for _, name := range allProviders {
		if _, exists := result[name]; !exists {
			result[name] = ProviderStatus{
				Name:      strings.Title(name),
				Available: false,
				Error:     "Not configured",
			}
		}
	}

	return result, nil
}

// ConfigureAIProvider configures an AI provider dynamically from UI
func (a *App) ConfigureAIProvider(config ProviderConfig) error {
	a.logger.WithField("provider", config.Provider).Info("Configuring AI provider")

	if a.aiConfig == nil {
		a.aiConfig = ai.DefaultRuntimeConfig()
	}

	provider := strings.ToLower(config.Provider)

	switch provider {
	case "openai":
		a.aiConfig.OpenAI.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			a.aiConfig.OpenAI.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			a.aiConfig.DefaultProvider = ai.ProviderOpenAI
		}

	case "anthropic":
		a.aiConfig.Anthropic.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			a.aiConfig.Anthropic.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			a.aiConfig.DefaultProvider = ai.ProviderAnthropic
		}

	case "ollama":
		a.aiConfig.Ollama.Endpoint = strings.TrimSpace(config.Endpoint)
		if config.Model != "" && !containsModel(a.aiConfig.Ollama.Models, config.Model) {
			a.aiConfig.Ollama.Models = append([]string{config.Model}, a.aiConfig.Ollama.Models...)
		}
		a.aiConfig.DefaultProvider = ai.ProviderOllama

	case "huggingface":
		a.aiConfig.HuggingFace.Endpoint = strings.TrimSpace(config.Endpoint)
		if config.Model != "" && !containsModel(a.aiConfig.HuggingFace.Models, config.Model) {
			a.aiConfig.HuggingFace.Models = append([]string{config.Model}, a.aiConfig.HuggingFace.Models...)
		}
		a.aiConfig.DefaultProvider = ai.ProviderHuggingFace

	case "claudecode":
		claudePath := ""
		if config.Options != nil {
			claudePath = strings.TrimSpace(config.Options["binary_path"])
		}
		if claudePath == "" {
			claudePath = "claude"
		}
		a.aiConfig.ClaudeCode.ClaudePath = claudePath
		if config.Model != "" {
			a.aiConfig.ClaudeCode.Model = config.Model
		}
		a.aiConfig.DefaultProvider = ai.ProviderClaudeCode

	case "codex":
		a.aiConfig.Codex.APIKey = strings.TrimSpace(config.APIKey)
		if config.Model != "" {
			a.aiConfig.Codex.Model = config.Model
		}
		if config.Endpoint != "" {
			a.aiConfig.Codex.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Options != nil {
			if org, ok := config.Options["organization"]; ok {
				a.aiConfig.Codex.Organization = strings.TrimSpace(org)
			}
		}
		a.aiConfig.DefaultProvider = ai.ProviderCodex

	default:
		return fmt.Errorf("unknown AI provider: %s", config.Provider)
	}

	if err := a.applyAIConfiguration(); err != nil {
		return fmt.Errorf("failed to apply AI configuration: %w", err)
	}

	return nil
}

func containsModel(models []string, candidate string) bool {
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if candidate == "" {
		return false
	}

	for _, model := range models {
		if strings.TrimSpace(strings.ToLower(model)) == candidate {
			return true
		}
	}

	return false
}

// GetAIConfiguration returns the currently active AI provider configuration with masked secrets.
func (a *App) GetAIConfiguration() (ProviderConfig, error) {
	if a.aiConfig == nil {
		return ProviderConfig{}, fmt.Errorf("AI configuration not initialised")
	}

	provider := strings.ToLower(string(a.aiConfig.DefaultProvider))
	if provider == "" {
		provider = "openai"
	}

	config := ProviderConfig{
		Provider: provider,
	}

	switch provider {
	case "openai":
		config.APIKey = maskSecret(a.aiConfig.OpenAI.APIKey)
		config.Endpoint = a.aiConfig.OpenAI.BaseURL
		if len(a.aiConfig.OpenAI.Models) > 0 {
			config.Model = a.aiConfig.OpenAI.Models[0]
		}

	case "anthropic":
		config.APIKey = maskSecret(a.aiConfig.Anthropic.APIKey)
		config.Endpoint = a.aiConfig.Anthropic.BaseURL
		if len(a.aiConfig.Anthropic.Models) > 0 {
			config.Model = a.aiConfig.Anthropic.Models[0]
		}

	case "codex":
		config.APIKey = maskSecret(a.aiConfig.Codex.APIKey)
		config.Endpoint = a.aiConfig.Codex.BaseURL
		config.Model = a.aiConfig.Codex.Model
		config.Options = map[string]string{
			"organization": a.aiConfig.Codex.Organization,
		}

	case "claudecode":
		config.Endpoint = a.aiConfig.ClaudeCode.ClaudePath
		config.Model = a.aiConfig.ClaudeCode.Model

	case "ollama":
		config.Endpoint = a.aiConfig.Ollama.Endpoint
		if len(a.aiConfig.Ollama.Models) > 0 {
			config.Model = a.aiConfig.Ollama.Models[0]
		}

	case "huggingface":
		config.Endpoint = a.aiConfig.HuggingFace.Endpoint
		config.Model = a.aiConfig.HuggingFace.RecommendedModel
	}

	return config, nil
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if len(value) <= 8 {
		return "********"
	}

	return fmt.Sprintf("%s****%s", value[:4], value[len(value)-4:])
}

// TestAIProvider tests a provider configuration without saving it
func (a *App) TestAIProvider(config ProviderConfig) (*ProviderStatus, error) {
	a.logger.WithField("provider", config.Provider).Info("Testing AI provider")

	testConfig := ai.DefaultRuntimeConfig()
	provider := strings.ToLower(config.Provider)

	switch provider {
	case "openai":
		testConfig.OpenAI.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.OpenAI.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		testConfig.DefaultProvider = ai.ProviderOpenAI

	case "anthropic":
		testConfig.Anthropic.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.Anthropic.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		testConfig.DefaultProvider = ai.ProviderAnthropic

	case "ollama":
		testConfig.Ollama.Endpoint = strings.TrimSpace(config.Endpoint)
		testConfig.DefaultProvider = ai.ProviderOllama

	case "huggingface":
		testConfig.HuggingFace.Endpoint = strings.TrimSpace(config.Endpoint)
		testConfig.DefaultProvider = ai.ProviderHuggingFace

	case "claudecode":
		path := ""
		if config.Options != nil {
			path = strings.TrimSpace(config.Options["binary_path"])
		}
		if path == "" {
			path = "claude"
		}
		testConfig.ClaudeCode.ClaudePath = path
		if config.Model != "" {
			testConfig.ClaudeCode.Model = config.Model
		}
		testConfig.DefaultProvider = ai.ProviderClaudeCode

	case "codex":
		testConfig.Codex.APIKey = strings.TrimSpace(config.APIKey)
		if config.Endpoint != "" {
			testConfig.Codex.BaseURL = strings.TrimSpace(config.Endpoint)
		}
		if config.Model != "" {
			testConfig.Codex.Model = config.Model
		}
		if config.Options != nil {
			if org, ok := config.Options["organization"]; ok {
				testConfig.Codex.Organization = strings.TrimSpace(org)
			}
		}
		testConfig.DefaultProvider = ai.ProviderCodex

	default:
		return nil, fmt.Errorf("unknown AI provider: %s", config.Provider)
	}

	// Remove other providers to avoid validation noise
	if provider != "openai" {
		testConfig.OpenAI.APIKey = ""
	}
	if provider != "anthropic" {
		testConfig.Anthropic.APIKey = ""
	}
	if provider != "codex" {
		testConfig.Codex.APIKey = ""
		testConfig.Codex.Organization = ""
	}
	if provider != "claudecode" {
		testConfig.ClaudeCode.ClaudePath = ""
	}
	if provider != "ollama" {
		testConfig.Ollama.Endpoint = ""
	}
	if provider != "huggingface" {
		testConfig.HuggingFace.Endpoint = ""
	}

	testService, err := ai.NewServiceWithConfig(testConfig, a.logger)
	if err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	defer func() {
		_ = testService.Stop(a.ctx)
	}()

	if err := testService.Start(a.ctx); err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	status := &ProviderStatus{
		Name:      config.Provider,
		Available: true,
		Error:     "",
		Model:     config.Model,
	}

	return status, nil
}

// ===============================
// Multi-Database Query Methods
// ===============================

// ExecuteMultiDatabaseQuery executes a query that spans multiple database connections
func (a *App) ExecuteMultiDatabaseQuery(req MultiQueryRequest) (*MultiQueryResponse, error) {
	a.logger.WithFields(logrus.Fields{
		"query_length": len(req.Query),
		"limit":        req.Limit,
		"strategy":     req.Strategy,
	}).Info("Executing multi-database query")

	// Parse strategy
	var strategy multiquery.ExecutionStrategy
	switch req.Strategy {
	case "federated":
		strategy = multiquery.StrategyFederated
	case "push_down":
		strategy = multiquery.StrategyPushDown
	case "auto":
		strategy = multiquery.StrategyAuto
	default:
		strategy = multiquery.StrategyAuto
	}

	// Build options
	options := &multiquery.Options{
		Timeout:  time.Duration(req.Timeout) * time.Second,
		Strategy: strategy,
		Limit:    req.Limit,
	}

	// Apply defaults
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if options.Limit == 0 {
		options.Limit = 1000
	}

	// Execute via database service
	result, err := a.databaseService.ExecuteMultiDatabaseQuery(req.Query, options)
	if err != nil {
		return &MultiQueryResponse{
			Error: err.Error(),
		}, nil
	}

	// Convert from services.MultiQueryResponse to app.MultiQueryResponse
	return &MultiQueryResponse{
		Columns:         result.Columns,
		Rows:            result.Rows,
		RowCount:        result.RowCount,
		Duration:        result.Duration,
		ConnectionsUsed: result.ConnectionsUsed,
		Strategy:        result.Strategy,
		Error:           result.Error,
	}, nil
}

// ValidateMultiQuery validates a multi-database query without executing it
func (a *App) ValidateMultiQuery(query string) (*ValidationResult, error) {
	a.logger.WithField("query_length", len(query)).Debug("Validating multi-query")

	validation, err := a.databaseService.ValidateMultiQuery(query)
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	return &ValidationResult{
		Valid:               validation.Valid,
		Errors:              validation.Errors,
		RequiredConnections: validation.RequiredConnections,
		Tables:              validation.Tables,
		EstimatedStrategy:   validation.EstimatedStrategy,
	}, nil
}

// GetMultiConnectionSchema returns combined schema information for multiple connections
func (a *App) GetMultiConnectionSchema(connectionIDs []string) (*CombinedSchema, error) {
	a.logger.WithField("connection_count", len(connectionIDs)).Debug("Fetching combined schema")

	schema, err := a.databaseService.GetCombinedSchema(connectionIDs)
	if err != nil {
		return nil, err
	}

	// Convert to app-level types
	result := &CombinedSchema{
		Connections: make(map[string]ConnectionSchema),
		Conflicts:   make([]SchemaConflict, len(schema.Conflicts)),
	}

	for connID, connSchema := range schema.Connections {
		tables := make([]TableInfo, len(connSchema.Tables))
		for i, table := range connSchema.Tables {
			tables[i] = TableInfo{
				Schema:    table.Schema,
				Name:      table.Name,
				Type:      table.Type,
				Comment:   table.Comment,
				RowCount:  table.RowCount,
				SizeBytes: table.SizeBytes,
			}
		}

		result.Connections[connID] = ConnectionSchema{
			ConnectionID: connSchema.ConnectionID,
			Schemas:      connSchema.Schemas,
			Tables:       tables,
		}
	}

	for i, conflict := range schema.Conflicts {
		conflictingTables := make([]ConflictingTable, len(conflict.Connections))
		for j, ct := range conflict.Connections {
			conflictingTables[j] = ConflictingTable{
				ConnectionID: ct.ConnectionID,
				TableName:    ct.TableName,
				Schema:       ct.Schema,
			}
		}

		result.Conflicts[i] = SchemaConflict{
			TableName:   conflict.TableName,
			Connections: conflictingTables,
			Resolution:  conflict.Resolution,
		}
	}

	return result, nil
}

// ParseQueryConnections extracts connection IDs from a query without validating
func (a *App) ParseQueryConnections(query string) ([]string, error) {
	a.logger.Debug("Parsing query for connections")

	// Use the database service's manager to parse the query
	// This is a simplified version - the full implementation would need access to the manager
	validation, err := a.databaseService.ValidateMultiQuery(query)
	if err != nil {
		return []string{}, nil // Return empty array instead of error for parsing
	}

	return validation.RequiredConnections, nil
}

// ShowNotification shows a notification using Wails MessageDialog
func (a *App) ShowNotification(title, message string, isError bool) {
	if isError {
		a.ShowErrorDialog(title, message)
	} else {
		a.ShowInfoDialog(title, message)
	}
}

// ==================== Schema Cache Management ====================

// InvalidateSchemaCache invalidates the cached schema for a specific connection
func (a *App) InvalidateSchemaCache(connectionID string) error {
	a.logger.WithField("connection", connectionID).Info("Invalidating schema cache")
	a.databaseService.InvalidateSchemaCache(connectionID)
	return nil
}

// InvalidateAllSchemas invalidates all cached schemas
func (a *App) InvalidateAllSchemas() error {
	a.logger.Info("Invalidating all schema caches")
	a.databaseService.InvalidateAllSchemas()
	return nil
}

// RefreshSchema forces a refresh of the schema for a connection
func (a *App) RefreshSchema(connectionID string) error {
	a.logger.WithField("connection", connectionID).Info("Refreshing schema")
	return a.databaseService.RefreshSchema(a.ctx, connectionID)
}

// GetSchemaCacheStats returns statistics about the schema cache
func (a *App) GetSchemaCacheStats() map[string]interface{} {
	return a.databaseService.GetSchemaCacheStats()
}

// ==================== Connection Management ====================

// GetConnectionCount returns the number of active database connections
func (a *App) GetConnectionCount() int {
	return a.databaseService.GetConnectionCount()
}

// GetConnectionIDs returns a list of all connection IDs
func (a *App) GetConnectionIDs() []string {
	return a.databaseService.GetConnectionIDs()
}

// GetAvailableEnvironments returns all unique environment tags across connections
func (a *App) GetAvailableEnvironments() ([]string, error) {
	if a.storageManager == nil {
		return []string{}, nil
	}
	return a.storageManager.GetAvailableEnvironments(a.ctx)
}
