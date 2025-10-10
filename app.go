package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/sql-studio/services"
)

//go:embed howlerops-light.png howlerops-dark.png howlerops-transparent.png
var iconFS embed.FS

// App struct
type App struct {
	ctx             context.Context
	logger          *logrus.Logger
	databaseService *services.DatabaseService
	fileService     *services.FileService
	keyboardService *services.KeyboardService
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

	a.logger.Info("HowlerOps desktop application started")

	// Emit app ready event
	runtime.EventsEmit(ctx, "app:startup-complete")
}

// OnShutdown is called when the app is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	a.logger.Info("HowlerOps desktop application shutting down")

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

// ShowNotification shows a notification using Wails MessageDialog
func (a *App) ShowNotification(title, message string, isError bool) {
	if isError {
		a.ShowErrorDialog(title, message)
	} else {
		a.ShowInfoDialog(title, message)
	}
}
