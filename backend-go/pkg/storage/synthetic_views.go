package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// SyntheticView represents a synthetic view in storage
type SyntheticView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Definition  string    `json:"definition"` // JSON string of ViewDefinition
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ViewDefinition represents the structure of a synthetic view definition
type ViewDefinition struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	Columns           []ColumnDefinition     `json:"columns"`
	IR                map[string]interface{} `json:"ir"` // JSON representation of QueryIR
	Sources           []SourceDefinition     `json:"sources"`
	CompiledDuckDBSQL string                 `json:"compiledDuckDBSQL"`
	Options           ViewOptions            `json:"options"`
}

// ColumnDefinition represents a column in a synthetic view
type ColumnDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SourceDefinition represents a source table for a synthetic view
type SourceDefinition struct {
	ConnectionIDOrName string `json:"connectionIdOrName"`
	Schema             string `json:"schema"`
	Table              string `json:"table"`
}

// ViewOptions contains configuration options for a synthetic view
type ViewOptions struct {
	RowLimitDefault int  `json:"rowLimitDefault"`
	MaterializeTemp bool `json:"materializeTemp"`
}

// ViewSummary represents a summary of a synthetic view
type ViewSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SyntheticViewStorage handles storage operations for synthetic views
type SyntheticViewStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewSyntheticViewStorage creates a new synthetic view storage
func NewSyntheticViewStorage(db *sql.DB, logger *logrus.Logger) *SyntheticViewStorage {
	return &SyntheticViewStorage{
		db:     db,
		logger: logger,
	}
}

// CreateTable creates the synthetic_views table if it doesn't exist
func (s *SyntheticViewStorage) CreateTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS synthetic_views (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		version TEXT NOT NULL DEFAULT '1.0.0',
		definition TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_synthetic_views_name ON synthetic_views(name);
	CREATE INDEX IF NOT EXISTS idx_synthetic_views_created_at ON synthetic_views(created_at);
	`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create synthetic_views table: %w", err)
	}

	s.logger.Info("Synthetic views table created successfully")
	return nil
}

// SaveSyntheticView saves a synthetic view definition
func (s *SyntheticViewStorage) SaveSyntheticView(viewDef *ViewDefinition) error {
	// Convert ViewDefinition to JSON
	definitionJSON, err := json.Marshal(viewDef)
	if err != nil {
		return fmt.Errorf("failed to marshal view definition: %w", err)
	}

	// Check if view exists
	exists, err := s.viewExists(viewDef.ID)
	if err != nil {
		return fmt.Errorf("failed to check if view exists: %w", err)
	}

	if exists {
		// Update existing view
		query := `
		UPDATE synthetic_views 
		SET name = ?, description = ?, version = ?, definition = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`

		_, err = s.db.Exec(query, viewDef.Name, viewDef.Description, viewDef.Version, string(definitionJSON), viewDef.ID)
		if err != nil {
			return fmt.Errorf("failed to update synthetic view: %w", err)
		}
	} else {
		// Insert new view
		query := `
		INSERT INTO synthetic_views (id, name, description, version, definition, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`

		_, err = s.db.Exec(query, viewDef.ID, viewDef.Name, viewDef.Description, viewDef.Version, string(definitionJSON))
		if err != nil {
			return fmt.Errorf("failed to insert synthetic view: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"view_id":   viewDef.ID,
		"view_name": viewDef.Name,
	}).Info("Synthetic view saved successfully")

	return nil
}

// GetSyntheticView retrieves a synthetic view by ID
func (s *SyntheticViewStorage) GetSyntheticView(id string) (*ViewDefinition, error) {
	query := `
	SELECT id, name, description, version, definition, created_at, updated_at
	FROM synthetic_views
	WHERE id = ?
	`

	row := s.db.QueryRow(query, id)

	var view SyntheticView
	err := row.Scan(&view.ID, &view.Name, &view.Description, &view.Version, &view.Definition, &view.CreatedAt, &view.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("synthetic view not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get synthetic view: %w", err)
	}

	// Unmarshal definition JSON
	var viewDef ViewDefinition
	err = json.Unmarshal([]byte(view.Definition), &viewDef)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal view definition: %w", err)
	}

	return &viewDef, nil
}

// ListSyntheticViews returns a list of all synthetic views
func (s *SyntheticViewStorage) ListSyntheticViews() ([]ViewSummary, error) {
	query := `
	SELECT id, name, description, version, created_at, updated_at
	FROM synthetic_views
	ORDER BY updated_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list synthetic views: %w", err)
	}
	defer rows.Close()

	var views []ViewSummary
	for rows.Next() {
		var view ViewSummary
		err := rows.Scan(&view.ID, &view.Name, &view.Description, &view.Version, &view.CreatedAt, &view.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan synthetic view: %w", err)
		}
		views = append(views, view)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return views, nil
}

// DeleteSyntheticView deletes a synthetic view by ID
func (s *SyntheticViewStorage) DeleteSyntheticView(id string) error {
	query := `DELETE FROM synthetic_views WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete synthetic view: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("synthetic view not found: %s", id)
	}

	s.logger.WithField("view_id", id).Info("Synthetic view deleted successfully")
	return nil
}

// GetSyntheticViewByName retrieves a synthetic view by name
func (s *SyntheticViewStorage) GetSyntheticViewByName(name string) (*ViewDefinition, error) {
	query := `
	SELECT id, name, description, version, definition, created_at, updated_at
	FROM synthetic_views
	WHERE name = ?
	`

	row := s.db.QueryRow(query, name)

	var view SyntheticView
	err := row.Scan(&view.ID, &view.Name, &view.Description, &view.Version, &view.Definition, &view.CreatedAt, &view.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("synthetic view not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get synthetic view by name: %w", err)
	}

	// Unmarshal definition JSON
	var viewDef ViewDefinition
	err = json.Unmarshal([]byte(view.Definition), &viewDef)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal view definition: %w", err)
	}

	return &viewDef, nil
}

// viewExists checks if a synthetic view exists by ID
func (s *SyntheticViewStorage) viewExists(id string) (bool, error) {
	query := `SELECT COUNT(*) FROM synthetic_views WHERE id = ?`

	var count int
	err := s.db.QueryRow(query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if view exists: %w", err)
	}

	return count > 0, nil
}

// GetSyntheticSchema returns the schema information for synthetic views
func (s *SyntheticViewStorage) GetSyntheticSchema() (map[string]interface{}, error) {
	views, err := s.ListSyntheticViews()
	if err != nil {
		return nil, fmt.Errorf("failed to list synthetic views: %w", err)
	}

	// Convert to schema format
	var schemaViews []map[string]interface{}
	for _, view := range views {
		// Get full view definition to extract columns
		viewDef, err := s.GetSyntheticView(view.ID)
		if err != nil {
			s.logger.WithError(err).WithField("view_id", view.ID).Warn("Failed to get view definition for schema")
			continue
		}

		// Convert columns to schema format
		var columns []map[string]interface{}
		for _, col := range viewDef.Columns {
			columns = append(columns, map[string]interface{}{
				"name":     col.Name,
				"type":     col.Type,
				"readOnly": true,
			})
		}

		schemaViews = append(schemaViews, map[string]interface{}{
			"name":     view.Name,
			"columns":  columns,
			"readOnly": true,
		})
	}

	return map[string]interface{}{
		"schema": "synthetic",
		"views":  schemaViews,
	}, nil
}
