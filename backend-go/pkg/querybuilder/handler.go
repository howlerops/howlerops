package querybuilder

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for query builder
type Handler struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

// NewHandler creates a new query builder handler
func NewHandler(dbManager *database.Manager, logger *logrus.Logger) *Handler {
	return &Handler{
		dbManager: dbManager,
		logger:    logger,
	}
}

// RegisterRoutes registers the query builder routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Schema introspection
	router.HandleFunc("/api/connections/{connectionId}/schema", h.GetDatabaseSchema).Methods("GET")

	// SQL generation and validation
	router.HandleFunc("/api/querybuilder/generate", h.GenerateSQL).Methods("POST")
	router.HandleFunc("/api/querybuilder/validate", h.ValidateQuery).Methods("POST")

	// Query preview
	router.HandleFunc("/api/querybuilder/preview", h.PreviewQuery).Methods("POST")
}

// DatabaseSchemaResponse represents the schema introspection response
type DatabaseSchemaResponse struct {
	Tables []TableMetadataResponse `json:"tables"`
}

// TableMetadataResponse represents table metadata
type TableMetadataResponse struct {
	Schema      string                   `json:"schema"`
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Comment     string                   `json:"comment,omitempty"`
	RowCount    int64                    `json:"rowCount"`
	Columns     []ColumnMetadataResponse `json:"columns"`
	ForeignKeys []ForeignKeyResponse     `json:"foreignKeys"`
}

// ColumnMetadataResponse represents column metadata
type ColumnMetadataResponse struct {
	Name               string  `json:"name"`
	DataType           string  `json:"dataType"`
	Nullable           bool    `json:"nullable"`
	DefaultValue       *string `json:"defaultValue,omitempty"`
	PrimaryKey         bool    `json:"primaryKey"`
	Unique             bool    `json:"unique"`
	Indexed            bool    `json:"indexed"`
	Comment            string  `json:"comment,omitempty"`
	OrdinalPosition    int     `json:"ordinalPosition"`
	CharacterMaxLength *int64  `json:"characterMaxLength,omitempty"`
	NumericPrecision   *int    `json:"numericPrecision,omitempty"`
	NumericScale       *int    `json:"numericScale,omitempty"`
}

// ForeignKeyResponse represents foreign key metadata
type ForeignKeyResponse struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedSchema  string   `json:"referencedSchema"`
	ReferencedColumns []string `json:"referencedColumns"`
	OnDelete          string   `json:"onDelete"`
	OnUpdate          string   `json:"onUpdate"`
}

// GenerateSQLRequest represents the SQL generation request
type GenerateSQLRequest struct {
	QueryBuilder QueryBuilder `json:"queryBuilder"`
}

// GenerateSQLResponse represents the SQL generation response
type GenerateSQLResponse struct {
	SQL        string        `json:"sql"`
	Args       []interface{} `json:"args"`
	Parameters int           `json:"parameters"`
}

// ValidateQueryRequest represents the query validation request
type ValidateQueryRequest struct {
	QueryBuilder QueryBuilder `json:"queryBuilder"`
}

// PreviewQueryRequest represents the query preview request
type PreviewQueryRequest struct {
	ConnectionID string `json:"connectionId"`
	SQL          string `json:"sql"`
	Limit        int    `json:"limit"`
}

// PreviewQueryResponse represents the query preview response
type PreviewQueryResponse struct {
	SQL             string          `json:"sql"`
	EstimatedRows   int64           `json:"estimatedRows"`
	Columns         []string        `json:"columns"`
	Rows            [][]interface{} `json:"rows"`
	TotalRows       int64           `json:"totalRows"`
	ExecutionTimeMs int64           `json:"executionTimeMs"`
}

// GetDatabaseSchema handles GET /api/connections/{connectionId}/schema
func (h *Handler) GetDatabaseSchema(w http.ResponseWriter, r *http.Request) {
	connectionID := mux.Vars(r)["connectionId"]
	if connectionID == "" {
		h.respondError(w, http.StatusBadRequest, "connection ID is required")
		return
	}

	// Get database connection from manager
	db, err := h.dbManager.GetConnection(connectionID)
	if err != nil {
		h.logger.WithError(err).WithField("connection_id", connectionID).Error("Failed to get database")
		h.respondError(w, http.StatusNotFound, "database connection not found")
		return
	}

	// Get schemas
	schemas, err := db.GetSchemas(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get schemas")
		h.respondError(w, http.StatusInternalServerError, "failed to get schemas")
		return
	}

	// Get tables for each schema
	var allTables []TableMetadataResponse

	for _, schema := range schemas {
		tables, err := db.GetTables(r.Context(), schema)
		if err != nil {
			h.logger.WithError(err).WithField("schema", schema).Error("Failed to get tables")
			continue
		}

		for _, table := range tables {
			// Get full table structure including columns and foreign keys
			structure, err := db.GetTableStructure(r.Context(), schema, table.Name)
			if err != nil {
				h.logger.WithError(err).WithFields(logrus.Fields{
					"schema": schema,
					"table":  table.Name,
				}).Error("Failed to get table structure")
				continue
			}

			// Convert to response format
			tableResp := TableMetadataResponse{
				Schema:      schema,
				Name:        table.Name,
				Type:        table.Type,
				Comment:     table.Comment,
				RowCount:    table.RowCount,
				Columns:     convertColumns(structure.Columns),
				ForeignKeys: convertForeignKeys(structure.ForeignKeys),
			}

			allTables = append(allTables, tableResp)
		}
	}

	response := DatabaseSchemaResponse{
		Tables: allTables,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GenerateSQL handles POST /api/querybuilder/generate
func (h *Handler) GenerateSQL(w http.ResponseWriter, r *http.Request) {
	var req GenerateSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Generate SQL
	sql, args, err := req.QueryBuilder.ToSQL()
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate SQL")
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := GenerateSQLResponse{
		SQL:        sql,
		Args:       args,
		Parameters: len(args),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ValidateQuery handles POST /api/querybuilder/validate
func (h *Handler) ValidateQuery(w http.ResponseWriter, r *http.Request) {
	var req ValidateQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate query
	validation := req.QueryBuilder.GetValidationErrors()

	h.respondJSON(w, http.StatusOK, validation)
}

// PreviewQuery handles POST /api/querybuilder/preview
func (h *Handler) PreviewQuery(w http.ResponseWriter, r *http.Request) {
	var req PreviewQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get database connection
	db, err := h.dbManager.GetConnection(req.ConnectionID)
	if err != nil {
		h.logger.WithError(err).WithField("connection_id", req.ConnectionID).Error("Failed to get database")
		h.respondError(w, http.StatusNotFound, "database connection not found")
		return
	}

	// Execute query with limit
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	queryOpts := &database.QueryOptions{
		Limit:    limit,
		ReadOnly: true,
	}

	result, err := db.ExecuteWithOptions(r.Context(), req.SQL, queryOpts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to execute preview query")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := PreviewQueryResponse{
		SQL:             req.SQL,
		EstimatedRows:   result.RowCount,
		Columns:         result.Columns,
		Rows:            result.Rows,
		TotalRows:       result.RowCount,
		ExecutionTimeMs: result.Duration.Milliseconds(),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// Helper functions

func convertColumns(cols []database.ColumnInfo) []ColumnMetadataResponse {
	result := make([]ColumnMetadataResponse, len(cols))
	for i, col := range cols {
		result[i] = ColumnMetadataResponse{
			Name:               col.Name,
			DataType:           col.DataType,
			Nullable:           col.Nullable,
			DefaultValue:       col.DefaultValue,
			PrimaryKey:         col.PrimaryKey,
			Unique:             col.Unique,
			Indexed:            col.Indexed,
			Comment:            col.Comment,
			OrdinalPosition:    col.OrdinalPosition,
			CharacterMaxLength: col.CharacterMaxLength,
			NumericPrecision:   col.NumericPrecision,
			NumericScale:       col.NumericScale,
		}
	}
	return result
}

func convertForeignKeys(fks []database.ForeignKeyInfo) []ForeignKeyResponse {
	result := make([]ForeignKeyResponse, len(fks))
	for i, fk := range fks {
		result[i] = ForeignKeyResponse{
			Name:              fk.Name,
			Columns:           fk.Columns,
			ReferencedTable:   fk.ReferencedTable,
			ReferencedSchema:  fk.ReferencedSchema,
			ReferencedColumns: fk.ReferencedColumns,
			OnDelete:          fk.OnDelete,
			OnUpdate:          fk.OnUpdate,
		}
	}
	return result
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{
		"error": message,
	})
}
