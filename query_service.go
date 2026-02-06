package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jbeck018/howlerops/pkg/database"
	"github.com/jbeck018/howlerops/pkg/database/multiquery"
	"github.com/sirupsen/logrus"
)

// QueryService handles all query execution operations
type QueryService struct {
	deps         *SharedDeps
	duckdbEngine interface{} // *duckdb.Engine - stored separately for direct access
}

// NewQueryService creates a new QueryService instance
func NewQueryService(deps *SharedDeps) *QueryService {
	return &QueryService{
		deps:         deps,
		duckdbEngine: deps.DuckDBEngine,
	}
}

// ExecuteQuery executes a SQL query
func (s *QueryService) ExecuteQuery(req QueryRequest) (*QueryResponse, error) {
	s.deps.Logger.WithFields(logrus.Fields{
		"connection_id": req.ConnectionID,
		"query_length":  len(req.Query),
		"is_export":     req.IsExport,
	}).Info("Executing query")

	// Check if query targets synthetic schema
	if strings.Contains(strings.ToLower(req.Query), "synthetic.") {
		// Route to DuckDB federation engine - this would need to be handled separately
		// For now, return an error indicating this needs the App's ExecuteSyntheticQuery
		return nil, fmt.Errorf("synthetic queries must be routed through App.ExecuteSyntheticQuery")
	}

	// Set default limit for pagination if not specified
	// For exports, allow unlimited rows (or very large limit)
	limit := req.Limit
	if req.IsExport {
		// Export mode: Use a very large limit (1 million rows max for safety)
		// If limit is explicitly set and smaller, respect it
		if limit == 0 {
			limit = 1000000 // 1M rows max for exports
		}
	} else {
		// Normal query mode: Default to 1000 rows
		if limit == 0 {
			limit = 1000 // Default page size
		}
	}

	// Extend timeout for exports
	timeout := 30 * time.Second
	if req.IsExport {
		timeout = 5 * time.Minute // 5 minute timeout for exports
	}
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	options := &database.QueryOptions{
		Timeout:  timeout,
		ReadOnly: false,
		Limit:    limit,
		Offset:   req.Offset, // NEW: Pass pagination offset
	}

	result, err := s.deps.DatabaseService.ExecuteQuery(req.ConnectionID, req.Query, options)
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
		// NEW: Pagination metadata
		TotalRows: result.TotalRows,
		PagedRows: result.PagedRows,
		HasMore:   result.HasMore,
		Offset:    result.Offset,
	}, nil
}

// GetEditableMetadata returns the status of an editable metadata job
func (s *QueryService) GetEditableMetadata(jobID string) (*EditableMetadataJobResponse, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, fmt.Errorf("jobId is required")
	}

	job, err := s.deps.DatabaseService.GetEditableMetadataJob(jobID)
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
func (s *QueryService) UpdateQueryRow(req QueryRowUpdateRequest) (*QueryRowUpdateResponse, error) {
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

	s.deps.Logger.WithFields(logrus.Fields{
		"connection_id": req.ConnectionID,
		"schema":        req.Schema,
		"table":         req.Table,
	}).Info("Applying row update")

	if err := s.deps.DatabaseService.UpdateRow(req.ConnectionID, params); err != nil {
		s.deps.Logger.WithError(err).Error("Row update failed")
		return &QueryRowUpdateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &QueryRowUpdateResponse{
		Success: true,
	}, nil
}

// InsertQueryRow inserts a new row via editable metadata guarantees
func (s *QueryService) InsertQueryRow(req QueryRowInsertRequest) (*QueryRowInsertResponse, error) {
	if req.ConnectionID == "" {
		return &QueryRowInsertResponse{Success: false, Message: "connectionId is required"}, nil
	}
	if len(req.Values) == 0 {
		return &QueryRowInsertResponse{Success: false, Message: "no column values provided"}, nil
	}

	params := database.InsertRowParams{
		Schema:        req.Schema,
		Table:         req.Table,
		Values:        req.Values,
		OriginalQuery: req.Query,
		Columns:       req.Columns,
	}

	row, err := s.deps.DatabaseService.InsertRow(req.ConnectionID, params)
	if err != nil {
		s.deps.Logger.WithError(err).Error("Row insert failed")
		return &QueryRowInsertResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &QueryRowInsertResponse{
		Success: true,
		Row:     row,
	}, nil
}

// DeleteQueryRows deletes one or more rows via editable metadata guarantees
func (s *QueryService) DeleteQueryRows(req QueryRowDeleteRequest) (*QueryRowDeleteResponse, error) {
	if req.ConnectionID == "" {
		return &QueryRowDeleteResponse{Success: false, Message: "connectionId is required"}, nil
	}
	if len(req.PrimaryKeys) == 0 {
		return &QueryRowDeleteResponse{Success: false, Message: "no primary keys provided"}, nil
	}

	deleted := 0
	for _, pk := range req.PrimaryKeys {
		params := database.DeleteRowParams{
			Schema:        req.Schema,
			Table:         req.Table,
			PrimaryKey:    pk,
			OriginalQuery: req.Query,
			Columns:       req.Columns,
		}

		if err := s.deps.DatabaseService.DeleteRow(req.ConnectionID, params); err != nil {
			s.deps.Logger.WithError(err).Error("Row delete failed")
			return &QueryRowDeleteResponse{
				Success: false,
				Message: err.Error(),
				Deleted: deleted,
			}, nil
		}
		deleted++
	}

	return &QueryRowDeleteResponse{
		Success: true,
		Deleted: deleted,
	}, nil
}

// GetSchemas returns available schemas for a connection
func (s *QueryService) GetSchemas(connectionID string) ([]string, error) {
	return s.deps.DatabaseService.GetSchemas(connectionID)
}

// GetTables returns tables in a schema
func (s *QueryService) GetTables(connectionID, schema string) ([]TableInfo, error) {
	tables, err := s.deps.DatabaseService.GetTables(connectionID, schema)
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
func (s *QueryService) GetTableStructure(connectionID, schema, table string) (*TableStructure, error) {
	structure, err := s.deps.DatabaseService.GetTableStructure(connectionID, schema, table)
	if err != nil {
		return nil, err
	}

	columns := make([]ColumnInfo, 0, len(structure.Columns))
	for _, column := range structure.Columns {
		columns = append(columns, ColumnInfo{
			Name:               column.Name,
			DataType:           column.DataType,
			Nullable:           column.Nullable,
			DefaultValue:       column.DefaultValue,
			PrimaryKey:         column.PrimaryKey,
			Unique:             column.Unique,
			Indexed:            column.Indexed,
			Comment:            column.Comment,
			OrdinalPosition:    column.OrdinalPosition,
			CharacterMaxLength: column.CharacterMaxLength,
			NumericPrecision:   column.NumericPrecision,
			NumericScale:       column.NumericScale,
			Metadata:           column.Metadata,
		})
	}

	indexes := make([]IndexInfo, 0, len(structure.Indexes))
	for _, idx := range structure.Indexes {
		indexes = append(indexes, IndexInfo{
			Name:     idx.Name,
			Columns:  idx.Columns,
			Unique:   idx.Unique,
			Primary:  idx.Primary,
			Type:     idx.Type,
			Method:   idx.Method,
			Metadata: idx.Metadata,
		})
	}

	fks := make([]ForeignKeyInfo, 0, len(structure.ForeignKeys))
	for _, fk := range structure.ForeignKeys {
		fks = append(fks, ForeignKeyInfo{
			Name:              fk.Name,
			Columns:           fk.Columns,
			ReferencedTable:   fk.ReferencedTable,
			ReferencedSchema:  fk.ReferencedSchema,
			ReferencedColumns: fk.ReferencedColumns,
			OnDelete:          fk.OnDelete,
			OnUpdate:          fk.OnUpdate,
		})
	}

	return &TableStructure{
		Table: TableInfo{
			Schema:    structure.Table.Schema,
			Name:      structure.Table.Name,
			Type:      structure.Table.Type,
			Comment:   structure.Table.Comment,
			RowCount:  structure.Table.RowCount,
			SizeBytes: structure.Table.SizeBytes,
		},
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: fks,
		Triggers:    structure.Triggers,
		Statistics:  structure.Statistics,
	}, nil
}

// ExplainQuery returns query execution plan
func (s *QueryService) ExplainQuery(connectionID, query string) (string, error) {
	return s.deps.DatabaseService.ExplainQuery(connectionID, query)
}

// ExecuteQueryStream executes a query with streaming results
func (s *QueryService) ExecuteQueryStream(connectionID, query string, batchSize int) (string, error) {
	return s.deps.DatabaseService.ExecuteQueryStream(connectionID, query, batchSize)
}

// CancelQueryStream cancels a streaming query
func (s *QueryService) CancelQueryStream(streamID string) error {
	return s.deps.DatabaseService.CancelQueryStream(streamID)
}

// ExecuteMultiDatabaseQuery executes a query across multiple databases
func (s *QueryService) ExecuteMultiDatabaseQuery(req MultiQueryRequest) (*MultiQueryResponse, error) {
	s.deps.Logger.WithFields(logrus.Fields{
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
	result, err := s.deps.DatabaseService.ExecuteMultiDatabaseQuery(req.Query, options)
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
		Editable:        result.Editable,
	}, nil
}

// ValidateMultiQuery validates a multi-database query without executing it
func (s *QueryService) ValidateMultiQuery(query string) (*ValidationResult, error) {
	s.deps.Logger.WithField("query_length", len(query)).Debug("Validating multi-query")

	validation, err := s.deps.DatabaseService.ValidateMultiQuery(query)
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
func (s *QueryService) GetMultiConnectionSchema(connectionIDs []string) (*CombinedSchema, error) {
	s.deps.Logger.WithField("connection_count", len(connectionIDs)).Debug("Fetching combined schema")

	schema, err := s.deps.DatabaseService.GetCombinedSchema(connectionIDs)
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

	// Note: Synthetic views would need to be added by the App layer
	// since QueryService doesn't have access to syntheticViews

	return result, nil
}

// ParseQueryConnections extracts connection IDs from a query without validating
func (s *QueryService) ParseQueryConnections(query string) ([]string, error) {
	s.deps.Logger.Debug("Parsing query for connections")

	// Use the database service's manager to parse the query
	// This is a simplified version - the full implementation would need access to the manager
	validation, err := s.deps.DatabaseService.ValidateMultiQuery(query)
	if err != nil {
		return []string{}, nil // Return empty array instead of error for parsing
	}

	return validation.RequiredConnections, nil
}
