package turso

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// QueryTemplate represents a reusable query template with parameters
type QueryTemplate struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description,omitempty"`
	SQLTemplate    string              `json:"sql_template"`
	Parameters     []TemplateParameter `json:"parameters,omitempty"`
	Tags           []string            `json:"tags,omitempty"`
	Category       string              `json:"category"`
	OrganizationID *string             `json:"organization_id,omitempty"`
	CreatedBy      string              `json:"created_by"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	IsPublic       bool                `json:"is_public"`
	UsageCount     int                 `json:"usage_count"`
	DeletedAt      *time.Time          `json:"deleted_at,omitempty"`
}

// TemplateParameter defines a parameter in a query template
type TemplateParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // 'string', 'number', 'date', 'boolean'
	DefaultValue interface{} `json:"default,omitempty"`
	Required     bool        `json:"required"`
	Description  string      `json:"description,omitempty"`
	Validation   string      `json:"validation,omitempty"` // regex or validation rule
}

// TemplateFilters for querying templates
type TemplateFilters struct {
	OrganizationID *string
	CreatedBy      *string
	Category       *string
	Tags           []string
	IsPublic       *bool
	SearchTerm     *string
	Limit          int
	Offset         int
}

// TemplateStore handles database operations for query templates
type TemplateStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTemplateStore creates a new template store
func NewTemplateStore(db *sql.DB, logger *logrus.Logger) *TemplateStore {
	return &TemplateStore{
		db:     db,
		logger: logger,
	}
}

// Create creates a new query template
func (s *TemplateStore) Create(ctx context.Context, template *QueryTemplate) error {
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now
	template.UsageCount = 0

	// Marshal JSON fields
	parametersJSON, err := json.Marshal(template.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO query_templates (
			id, name, description, sql_template, parameters, tags, category,
			organization_id, created_by, created_at, updated_at, is_public, usage_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Description, template.SQLTemplate,
		string(parametersJSON), string(tagsJSON), template.Category,
		template.OrganizationID, template.CreatedBy,
		template.CreatedAt.Unix(), template.UpdatedAt.Unix(),
		template.IsPublic, template.UsageCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"template_id": template.ID,
		"name":        template.Name,
		"created_by":  template.CreatedBy,
	}).Info("Template created")

	return nil
}

// GetByID retrieves a template by ID
func (s *TemplateStore) GetByID(ctx context.Context, id string) (*QueryTemplate, error) {
	query := `
		SELECT id, name, description, sql_template, parameters, tags, category,
		       organization_id, created_by, created_at, updated_at, is_public, usage_count, deleted_at
		FROM query_templates
		WHERE id = ?
	`

	var template QueryTemplate
	var parametersJSON, tagsJSON sql.NullString
	var orgID sql.NullString
	var description sql.NullString
	var createdAt, updatedAt int64
	var deletedAt sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID, &template.Name, &description, &template.SQLTemplate,
		&parametersJSON, &tagsJSON, &template.Category,
		&orgID, &template.CreatedBy,
		&createdAt, &updatedAt,
		&template.IsPublic, &template.UsageCount, &deletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	// Convert timestamps
	template.CreatedAt = time.Unix(createdAt, 0)
	template.UpdatedAt = time.Unix(updatedAt, 0)

	if deletedAt.Valid {
		deletedTime := time.Unix(deletedAt.Int64, 0)
		template.DeletedAt = &deletedTime
	}

	// Convert nullable fields
	if description.Valid {
		template.Description = description.String
	}

	if orgID.Valid {
		template.OrganizationID = &orgID.String
	}

	// Unmarshal JSON fields
	if parametersJSON.Valid && parametersJSON.String != "" && parametersJSON.String != "null" {
		if err := json.Unmarshal([]byte(parametersJSON.String), &template.Parameters); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal parameters")
		}
	}

	if tagsJSON.Valid && tagsJSON.String != "" && tagsJSON.String != "null" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &template.Tags); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal tags")
		}
	}

	return &template, nil
}

// List retrieves templates with filters
func (s *TemplateStore) List(ctx context.Context, filters TemplateFilters) ([]*QueryTemplate, error) {
	query := `
		SELECT id, name, description, sql_template, parameters, tags, category,
		       organization_id, created_by, created_at, updated_at, is_public, usage_count
		FROM query_templates
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}

	// Apply filters
	if filters.OrganizationID != nil {
		query += " AND organization_id = ?"
		args = append(args, *filters.OrganizationID)
	}

	if filters.CreatedBy != nil {
		query += " AND created_by = ?"
		args = append(args, *filters.CreatedBy)
	}

	if filters.Category != nil {
		query += " AND category = ?"
		args = append(args, *filters.Category)
	}

	if filters.IsPublic != nil {
		query += " AND is_public = ?"
		args = append(args, *filters.IsPublic)
	}

	if filters.SearchTerm != nil && *filters.SearchTerm != "" {
		query += " AND (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + *filters.SearchTerm + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Order by usage and recent updates
	query += " ORDER BY usage_count DESC, updated_at DESC"

	// Apply pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	} else {
		query += " LIMIT 100" // Default limit
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	return s.queryTemplates(ctx, query, args...)
}

// Update updates a template
func (s *TemplateStore) Update(ctx context.Context, template *QueryTemplate) error {
	template.UpdatedAt = time.Now()

	// Marshal JSON fields
	parametersJSON, err := json.Marshal(template.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE query_templates
		SET name = ?, description = ?, sql_template = ?, parameters = ?, tags = ?,
		    category = ?, updated_at = ?, is_public = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		template.Name, template.Description, template.SQLTemplate,
		string(parametersJSON), string(tagsJSON), template.Category,
		template.UpdatedAt.Unix(), template.IsPublic,
		template.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found or already deleted")
	}

	s.logger.WithField("template_id", template.ID).Info("Template updated")
	return nil
}

// Delete soft-deletes a template
func (s *TemplateStore) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE query_templates
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found or already deleted")
	}

	s.logger.WithField("template_id", id).Info("Template deleted")
	return nil
}

// IncrementUsage increments the usage counter for a template
func (s *TemplateStore) IncrementUsage(ctx context.Context, id string) error {
	query := `
		UPDATE query_templates
		SET usage_count = usage_count + 1
		WHERE id = ? AND deleted_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	return nil
}

// GetPopularTemplates returns the most used templates
func (s *TemplateStore) GetPopularTemplates(ctx context.Context, organizationID *string, limit int) ([]*QueryTemplate, error) {
	query := `
		SELECT id, name, description, sql_template, parameters, tags, category,
		       organization_id, created_by, created_at, updated_at, is_public, usage_count
		FROM query_templates
		WHERE deleted_at IS NULL AND usage_count > 0
	`

	args := []interface{}{}

	if organizationID != nil {
		query += " AND organization_id = ?"
		args = append(args, *organizationID)
	}

	query += " ORDER BY usage_count DESC LIMIT ?"
	args = append(args, limit)

	return s.queryTemplates(ctx, query, args...)
}

// queryTemplates is a helper method to query and scan multiple templates
func (s *TemplateStore) queryTemplates(ctx context.Context, query string, args ...interface{}) ([]*QueryTemplate, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var templates []*QueryTemplate
	for rows.Next() {
		var template QueryTemplate
		var parametersJSON, tagsJSON sql.NullString
		var orgID sql.NullString
		var description sql.NullString
		var createdAt, updatedAt int64

		err := rows.Scan(
			&template.ID, &template.Name, &description, &template.SQLTemplate,
			&parametersJSON, &tagsJSON, &template.Category,
			&orgID, &template.CreatedBy,
			&createdAt, &updatedAt,
			&template.IsPublic, &template.UsageCount,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		// Convert timestamps
		template.CreatedAt = time.Unix(createdAt, 0)
		template.UpdatedAt = time.Unix(updatedAt, 0)

		// Convert nullable fields
		if description.Valid {
			template.Description = description.String
		}

		if orgID.Valid {
			template.OrganizationID = &orgID.String
		}

		// Unmarshal JSON fields
		if parametersJSON.Valid && parametersJSON.String != "" && parametersJSON.String != "null" {
			if err := json.Unmarshal([]byte(parametersJSON.String), &template.Parameters); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal parameters")
			}
		}

		if tagsJSON.Valid && tagsJSON.String != "" && tagsJSON.String != "null" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &template.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal tags")
			}
		}

		templates = append(templates, &template)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating templates: %w", err)
	}

	return templates, nil
}
