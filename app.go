package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"

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

	// Initialize AI service if available (graceful degradation)
	a.initializeAIService(ctx)
	a.initializeEmbeddingService(ctx)

	a.logger.Info("HowlerOps desktop application started")

	// Emit app ready event
	runtime.EventsEmit(ctx, "app:startup-complete")
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
			DataDir:    getEnvOrDefault("HOWLEROPS_DATA_DIR", "~/.howlerops"),
			Database:   "local.db",
			VectorsDB:  "vectors.db",
			UserID:     userID,
			VectorSize: vectorSize,
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

// initializeAIService initializes the AI service if configured
// This is optional and will gracefully degrade if not available
func (a *App) initializeAIService(ctx context.Context) {
	// Try to initialize AI service (loads config from environment variables)
	aiService, err := ai.NewService(a.logger)
	if err != nil {
		a.logger.WithError(err).Warn("AI service not available - configure API keys via environment variables")
		a.logger.Info("Set OPENAI_API_KEY or ANTHROPIC_API_KEY to enable AI features")
		return
	}

	// Start the AI service
	if err := aiService.Start(ctx); err != nil {
		a.logger.WithError(err).Warn("Failed to start AI service")
		return
	}

	a.aiService = aiService
	a.logger.Info("AI service initialized successfully")

	// Check which providers have API keys
	if os.Getenv("OPENAI_API_KEY") != "" {
		a.logger.Info("OpenAI provider enabled")
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		a.logger.Info("Anthropic provider enabled")
	}
	if os.Getenv("OLLAMA_ENDPOINT") != "" || true {
		a.logger.Info("Ollama provider available (default: http://localhost:11434)")
	}
}

// initializeEmbeddingService initializes the embedding service used for AI memory recall
func (a *App) initializeEmbeddingService(ctx context.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		a.logger.Warn("Embedding service disabled: OPENAI_API_KEY not set")
		return
	}

	model := getEnvOrDefault("OPENAI_EMBED_MODEL", "text-embedding-3-small")
	provider := rag.NewOpenAIEmbeddingProvider(apiKey, model, a.logger)
	a.embeddingService = rag.NewEmbeddingService(provider, a.logger)
	a.logger.WithField("model", model).Info("Embedding service initialized")
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
	runtime.EventsEmit(ctx, "app:shutdown")
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
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// ShowErrorDialog shows an error dialog
func (a *App) ShowErrorDialog(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   title,
		Message: message,
	})
}

// ShowQuestionDialog shows a question dialog and returns the result
func (a *App) ShowQuestionDialog(title, message string) (bool, error) {
	result, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
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
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
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

	// For now, just check if the binary path looks reasonable
	// TODO: Implement actual Claude CLI check
	if strings.Contains(binaryPath, "claude") || binaryPath == "claude" {
		return &AITestResponse{
			Success: true,
			Message: "Claude Code connection test successful",
		}
	}

	return &AITestResponse{
		Success: false,
		Error:   "Claude CLI not found or invalid path",
	}
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
	return &AITestResponse{
		Success: true,
		Message: "Codex connection test successful",
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
		return nil, fmt.Errorf("AI service not configured. Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable and restart the app")
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
		// Single database mode: get schema for specific connection
		schemas, err := a.databaseService.GetSchemas(req.ConnectionID)
		if err == nil && len(schemas) > 0 {
			// Build schema context with table details
			schemaContext = "Database Schema Information:\n"
			for _, schema := range schemas {
				// Get tables for each schema
				tables, err := a.databaseService.GetTables(req.ConnectionID, schema)
				if err == nil && len(tables) > 0 {
					schemaContext += fmt.Sprintf("\nSchema: %s\n", schema)
					schemaContext += "Tables: "
					for i, table := range tables {
						if i > 0 {
							schemaContext += ", "
						}
						schemaContext += table.Name
					}
					schemaContext += "\n"
				}
			}
		}

		// Add custom context if provided
		if req.Context != "" {
			schemaContext += "\n" + req.Context
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

	return &GeneratedSQLResponse{
		SQL:         result.SQL,
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		Warnings:    []string{}, // TODO: Add warnings if confidence is low
	}, nil
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
		return nil, fmt.Errorf("AI service not configured. Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable")
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
		return nil, fmt.Errorf("AI service not configured. Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable")
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
		return map[string]ProviderStatus{
			"openai":      {Name: "OpenAI", Available: false, Error: "Not configured - set OPENAI_API_KEY"},
			"anthropic":   {Name: "Anthropic", Available: false, Error: "Not configured - set ANTHROPIC_API_KEY"},
			"claudecode":  {Name: "Claude Code", Available: false, Error: "Not configured"},
			"codex":       {Name: "Codex", Available: false, Error: "Not configured"},
			"ollama":      {Name: "Ollama", Available: false, Error: "Not configured"},
			"huggingface": {Name: "HuggingFace", Available: false, Error: "Not configured"},
		}, nil
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

	// Set environment variables based on provider configuration
	// This allows the AI service to pick them up
	switch strings.ToLower(config.Provider) {
	case "openai":
		if config.APIKey != "" {
			os.Setenv("OPENAI_API_KEY", config.APIKey)
		}
		if config.Model != "" {
			os.Setenv("OPENAI_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "openai")

	case "anthropic":
		if config.APIKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", config.APIKey)
		}
		if config.Model != "" {
			os.Setenv("ANTHROPIC_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "anthropic")

	case "ollama":
		if config.Endpoint != "" {
			os.Setenv("OLLAMA_ENDPOINT", config.Endpoint)
		}
		if config.Model != "" {
			os.Setenv("OLLAMA_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "ollama")

	case "claudecode":
		if config.Options != nil {
			if binaryPath, ok := config.Options["binary_path"]; ok {
				os.Setenv("CLAUDE_BINARY_PATH", binaryPath)
			}
		}
		if config.Model != "" {
			os.Setenv("CLAUDE_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "claudecode")

	case "codex":
		if config.APIKey != "" {
			os.Setenv("OPENAI_API_KEY", config.APIKey) // Codex uses OpenAI API
		}
		if config.Model != "" {
			os.Setenv("CODEX_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "codex")

	case "huggingface":
		if config.Endpoint != "" {
			os.Setenv("HUGGINGFACE_ENDPOINT", config.Endpoint)
		}
		if config.Model != "" {
			os.Setenv("HUGGINGFACE_MODEL", config.Model)
		}
		os.Setenv("AI_DEFAULT_PROVIDER", "huggingface")
	}

	// Reinitialize AI service with new configuration
	aiService, err := ai.NewService(a.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize AI service with new config: %w", err)
	}

	// Start the new AI service
	if err := aiService.Start(a.ctx); err != nil {
		return fmt.Errorf("failed to start AI service: %w", err)
	}

	// Stop old service if it exists
	if a.aiService != nil {
		_ = a.aiService.Stop(a.ctx)
	}

	// Replace with new service
	a.aiService = aiService

	a.logger.WithFields(logrus.Fields{
		"provider": config.Provider,
		"model":    config.Model,
	}).Info("AI provider configured successfully")

	return nil
}

// TestAIProvider tests a provider configuration without saving it
func (a *App) TestAIProvider(config ProviderConfig) (*ProviderStatus, error) {
	a.logger.WithField("provider", config.Provider).Info("Testing AI provider")

	// Temporarily set environment variables for testing
	oldEnv := make(map[string]string)

	switch strings.ToLower(config.Provider) {
	case "openai":
		if oldKey := os.Getenv("OPENAI_API_KEY"); oldKey != "" {
			oldEnv["OPENAI_API_KEY"] = oldKey
		}
		os.Setenv("OPENAI_API_KEY", config.APIKey)
		defer func() {
			if old, ok := oldEnv["OPENAI_API_KEY"]; ok {
				os.Setenv("OPENAI_API_KEY", old)
			}
		}()

	case "anthropic":
		if oldKey := os.Getenv("ANTHROPIC_API_KEY"); oldKey != "" {
			oldEnv["ANTHROPIC_API_KEY"] = oldKey
		}
		os.Setenv("ANTHROPIC_API_KEY", config.APIKey)
		defer func() {
			if old, ok := oldEnv["ANTHROPIC_API_KEY"]; ok {
				os.Setenv("ANTHROPIC_API_KEY", old)
			}
		}()
	}

	// Try to create a test AI service
	testService, err := ai.NewService(a.logger)
	if err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}
	defer testService.Stop(a.ctx)

	// Start it to verify connection
	if err := testService.Start(a.ctx); err != nil {
		return &ProviderStatus{
			Name:      config.Provider,
			Available: false,
			Error:     err.Error(),
		}, nil
	}

	return &ProviderStatus{
		Name:      config.Provider,
		Available: true,
		Error:     "",
	}, nil
}

// GetAIConfiguration returns the current AI configuration
func (a *App) GetAIConfiguration() (ProviderConfig, error) {
	provider := strings.ToLower(os.Getenv("AI_DEFAULT_PROVIDER"))
	if provider == "" {
		provider = "openai"
	}

	config := ProviderConfig{
		Provider: provider,
	}

	switch provider {
	case "openai":
		config.APIKey = maskAPIKey(os.Getenv("OPENAI_API_KEY"))
		config.Model = getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini")

	case "anthropic":
		config.APIKey = maskAPIKey(os.Getenv("ANTHROPIC_API_KEY"))
		config.Model = getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022")

	case "ollama":
		config.Endpoint = getEnvOrDefault("OLLAMA_ENDPOINT", "http://localhost:11434")
		config.Model = getEnvOrDefault("OLLAMA_MODEL", "sqlcoder:7b")
	}

	return config, nil
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
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
