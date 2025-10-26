package analyzer

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/autocomplete"
	"github.com/sql-studio/backend-go/internal/nl2sql"
)

// Handler handles query analysis HTTP endpoints
type Handler struct {
	analyzer        *QueryAnalyzer
	nl2sqlConverter *nl2sql.NL2SQLConverter
	autocomplete    *autocomplete.AutocompleteService
	schemaService   SchemaService
	logger          *logrus.Logger
}

// SchemaService interface for retrieving schema information
type SchemaService interface {
	GetSchema(connectionID string) (*Schema, error)
}

// NewHandler creates a new query analysis handler
func NewHandler(
	schemaService SchemaService,
	logger *logrus.Logger,
) *Handler {
	if logger == nil {
		logger = logrus.New()
	}

	return &Handler{
		analyzer:      NewQueryAnalyzer(logger),
		schemaService: schemaService,
		logger:        logger,
	}
}

// RegisterRoutes registers the query analysis routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Apply authentication middleware to all routes
	api := router.PathPrefix("/api").Subrouter()
	// TODO: Add HTTP authentication middleware
	// api.Use(middleware.AuthMiddleware)

	// Query analysis endpoints
	api.HandleFunc("/query/analyze", h.AnalyzeQuery).Methods("POST", "OPTIONS")
	api.HandleFunc("/query/nl2sql", h.NaturalLanguageToSQL).Methods("POST", "OPTIONS")
	api.HandleFunc("/query/autocomplete", h.Autocomplete).Methods("POST", "OPTIONS")
	api.HandleFunc("/query/explain", h.ExplainQuery).Methods("POST", "OPTIONS")
	api.HandleFunc("/query/patterns", h.GetSupportedPatterns).Methods("GET", "OPTIONS")
}

// AnalyzeQueryRequest represents the analyze query request
type AnalyzeQueryRequest struct {
	SQL          string `json:"sql" validate:"required"`
	ConnectionID string `json:"connection_id,omitempty"`
}

// AnalyzeQuery analyzes a SQL query and provides optimization suggestions
func (h *Handler) AnalyzeQuery(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get schema if connection ID provided
	var schema *Schema
	if req.ConnectionID != "" {
		var err error
		schema, err = h.schemaService.GetSchema(req.ConnectionID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get schema for analysis")
			// Continue without schema - will provide generic suggestions
		}
	}

	// Analyze the query
	result, err := h.analyzer.Analyze(req.SQL, schema)
	if err != nil {
		h.logger.WithError(err).Error("Failed to analyze query")
		http.Error(w, "Failed to analyze query", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// NL2SQLRequest represents the natural language to SQL request
type NL2SQLRequest struct {
	Query        string `json:"query" validate:"required"`
	ConnectionID string `json:"connection_id,omitempty"`
}

// NaturalLanguageToSQL converts natural language to SQL
func (h *Handler) NaturalLanguageToSQL(w http.ResponseWriter, r *http.Request) {
	var req NL2SQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get schema if connection ID provided
	var nlSchema *nl2sql.Schema
	if req.ConnectionID != "" {
		schema, err := h.schemaService.GetSchema(req.ConnectionID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get schema for NL2SQL")
		} else {
			// Convert schema format
			nlSchema = convertToNL2SQLSchema(schema)
		}
	}

	// Initialize converter if not already done
	if h.nl2sqlConverter == nil {
		h.nl2sqlConverter = nl2sql.NewNL2SQLConverter(nlSchema, h.logger)
	}

	// Convert natural language to SQL
	result, err := h.nl2sqlConverter.Convert(req.Query)
	if err != nil {
		// Return result even on error (contains suggestions)
		respondJSON(w, http.StatusOK, result)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// AutocompleteRequest represents the autocomplete request
type AutocompleteRequest struct {
	SQL          string `json:"sql" validate:"required"`
	CursorPos    int    `json:"cursor" validate:"min=0"`
	ConnectionID string `json:"connection_id,omitempty"`
}

// Autocomplete provides SQL autocomplete suggestions
func (h *Handler) Autocomplete(w http.ResponseWriter, r *http.Request) {
	var req AutocompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get schema if connection ID provided
	var acSchema *autocomplete.Schema
	if req.ConnectionID != "" {
		schema, err := h.schemaService.GetSchema(req.ConnectionID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get schema for autocomplete")
		} else {
			// Convert schema format
			acSchema = convertToAutocompleteSchema(schema)
		}
	}

	// Initialize autocomplete if not already done
	if h.autocomplete == nil {
		h.autocomplete = autocomplete.NewAutocompleteService(acSchema, h.logger)
	}

	// Get autocomplete suggestions
	suggestions, err := h.autocomplete.GetSuggestions(req.SQL, req.CursorPos)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get autocomplete suggestions")
		http.Error(w, "Failed to get suggestions", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
	})
}

// ExplainQueryRequest represents the explain query request
type ExplainQueryRequest struct {
	SQL     string `json:"sql" validate:"required"`
	Verbose bool   `json:"verbose,omitempty"`
}

// ExplainQuery explains what a SQL query does in plain English
func (h *Handler) ExplainQuery(w http.ResponseWriter, r *http.Request) {
	var req ExplainQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Verbose {
		// Return detailed explanation
		result, err := ExplainComplex(req.SQL)
		if err != nil {
			h.logger.WithError(err).Error("Failed to explain query")
			http.Error(w, "Failed to explain query", http.StatusInternalServerError)
			return
		}
		respondJSON(w, http.StatusOK, result)
	} else {
		// Return simple explanation
		explanation, err := Explain(req.SQL)
		if err != nil {
			h.logger.WithError(err).Error("Failed to explain query")
			http.Error(w, "Failed to explain query", http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"explanation": explanation,
		})
	}
}

// GetSupportedPatterns returns the supported natural language patterns
func (h *Handler) GetSupportedPatterns(w http.ResponseWriter, r *http.Request) {
	// Initialize converter if needed
	if h.nl2sqlConverter == nil {
		h.nl2sqlConverter = nl2sql.NewNL2SQLConverter(nil, h.logger)
	}

	patterns := h.nl2sqlConverter.GetSupportedPatterns()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"patterns": patterns,
		"total":    len(patterns),
	})
}

// Helper functions to convert schema formats

func convertToNL2SQLSchema(schema *Schema) *nl2sql.Schema {
	if schema == nil {
		return nil
	}

	nlSchema := &nl2sql.Schema{
		Tables: make(map[string]*nl2sql.Table),
	}

	for tableName, table := range schema.Tables {
		nlTable := &nl2sql.Table{
			Name:    tableName,
			Columns: []string{},
		}

		for colName := range table.Columns {
			nlTable.Columns = append(nlTable.Columns, colName)
		}

		nlSchema.Tables[tableName] = nlTable
	}

	return nlSchema
}

func convertToAutocompleteSchema(schema *Schema) *autocomplete.Schema {
	if schema == nil {
		return nil
	}

	acSchema := &autocomplete.Schema{
		Tables: make(map[string]*autocomplete.Table),
	}

	for tableName, table := range schema.Tables {
		acTable := &autocomplete.Table{
			Name:    tableName,
			Columns: make(map[string]string),
			Indexes: []string{},
		}

		for colName, col := range table.Columns {
			acTable.Columns[colName] = col.Type
			if col.Indexed {
				acTable.Indexes = append(acTable.Indexes, colName)
			}
		}

		acSchema.Tables[tableName] = acTable
	}

	return acSchema
}

// MockSchemaService is a mock implementation for testing
type MockSchemaService struct{}

// GetSchema returns a mock schema for testing
func (m *MockSchemaService) GetSchema(connectionID string) (*Schema, error) {
	// Return a sample schema for testing
	return &Schema{
		Tables: map[string]*Table{
			"users": {
				Name: "users",
				Columns: map[string]*Column{
					"id":         {Name: "id", Type: "INTEGER", Indexed: true},
					"name":       {Name: "name", Type: "VARCHAR(255)"},
					"email":      {Name: "email", Type: "VARCHAR(255)", Indexed: true},
					"created_at": {Name: "created_at", Type: "TIMESTAMP"},
					"status":     {Name: "status", Type: "VARCHAR(50)"},
				},
				Indexes: map[string]*Index{
					"PRIMARY": {
						Name:    "PRIMARY",
						Columns: []string{"id"},
						Primary: true,
					},
					"idx_email": {
						Name:    "idx_email",
						Columns: []string{"email"},
						Unique:  true,
					},
				},
				RowCount: 50000,
			},
			"orders": {
				Name: "orders",
				Columns: map[string]*Column{
					"id":         {Name: "id", Type: "INTEGER", Indexed: true},
					"user_id":    {Name: "user_id", Type: "INTEGER", Indexed: true},
					"total":      {Name: "total", Type: "DECIMAL(10,2)"},
					"status":     {Name: "status", Type: "VARCHAR(50)"},
					"created_at": {Name: "created_at", Type: "TIMESTAMP"},
				},
				Indexes: map[string]*Index{
					"PRIMARY": {
						Name:    "PRIMARY",
						Columns: []string{"id"},
						Primary: true,
					},
					"idx_user_id": {
						Name:    "idx_user_id",
						Columns: []string{"user_id"},
					},
				},
				RowCount: 100000,
			},
			"products": {
				Name: "products",
				Columns: map[string]*Column{
					"id":          {Name: "id", Type: "INTEGER", Indexed: true},
					"name":        {Name: "name", Type: "VARCHAR(255)"},
					"description": {Name: "description", Type: "TEXT"},
					"price":       {Name: "price", Type: "DECIMAL(10,2)"},
					"category":    {Name: "category", Type: "VARCHAR(100)", Indexed: true},
					"stock":       {Name: "stock", Type: "INTEGER"},
				},
				Indexes: map[string]*Index{
					"PRIMARY": {
						Name:    "PRIMARY",
						Columns: []string{"id"},
						Primary: true,
					},
					"idx_category": {
						Name:    "idx_category",
						Columns: []string{"category"},
					},
				},
				RowCount: 5000,
			},
		},
	}, nil
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
