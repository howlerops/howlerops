package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ReportFolder represents a folder in the report hierarchy.
type ReportFolder struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ParentID  *string    `json:"parentId,omitempty"`
	Color     string     `json:"color,omitempty"`
	Icon      string     `json:"icon,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// FolderStorage manages report folder persistence.
type FolderStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewFolderStorage creates a new folder storage helper.
func NewFolderStorage(db *sql.DB, logger *logrus.Logger) *FolderStorage {
	return &FolderStorage{db: db, logger: logger}
}

// EnsureSchema creates the folders table.
func (s *FolderStorage) EnsureSchema() error {
	if s.db == nil {
		return errors.New("folder storage database not available")
	}

	statement := `
CREATE TABLE IF NOT EXISTS report_folders (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	parent_id TEXT,
	color TEXT,
	icon TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (parent_id) REFERENCES report_folders(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON report_folders(parent_id);
`

	if _, err := s.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure folder schema: %w", err)
	}

	return nil
}

// CreateFolder creates a new folder.
func (s *FolderStorage) CreateFolder(folder *ReportFolder) error {
	if s.db == nil {
		return errors.New("folder storage database not available")
	}
	if folder == nil {
		return errors.New("folder is nil")
	}

	if folder.ID == "" {
		folder.ID = uuid.NewString()
	}
	folder.CreatedAt = time.Now().UTC()
	folder.UpdatedAt = folder.CreatedAt

	query := `
INSERT INTO report_folders (id, name, parent_id, color, icon, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

	_, err := s.db.Exec(
		query,
		folder.ID,
		folder.Name,
		folder.ParentID,
		folder.Color,
		folder.Icon,
		folder.CreatedAt,
		folder.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}

// ListFolders returns all folders.
func (s *FolderStorage) ListFolders() ([]ReportFolder, error) {
	if s.db == nil {
		return nil, errors.New("folder storage database not available")
	}

	rows, err := s.db.Query(`
SELECT id, name, parent_id, color, icon, created_at, updated_at
FROM report_folders
ORDER BY name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []ReportFolder
	for rows.Next() {
		var folder ReportFolder
		var parentID, color, icon sql.NullString

		if err := rows.Scan(
			&folder.ID,
			&folder.Name,
			&parentID,
			&color,
			&icon,
			&folder.CreatedAt,
			&folder.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}

		if parentID.Valid {
			folder.ParentID = &parentID.String
		}
		if color.Valid {
			folder.Color = color.String
		}
		if icon.Valid {
			folder.Icon = icon.String
		}

		folders = append(folders, folder)
	}

	return folders, nil
}

// UpdateFolder updates an existing folder.
func (s *FolderStorage) UpdateFolder(folder *ReportFolder) error {
	if s.db == nil {
		return errors.New("folder storage database not available")
	}
	if folder == nil {
		return errors.New("folder is nil")
	}

	folder.UpdatedAt = time.Now().UTC()

	query := `
UPDATE report_folders
SET name = ?, parent_id = ?, color = ?, icon = ?, updated_at = ?
WHERE id = ?
`

	result, err := s.db.Exec(
		query,
		folder.Name,
		folder.ParentID,
		folder.Color,
		folder.Icon,
		folder.UpdatedAt,
		folder.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update folder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("folder not found: %s", folder.ID)
	}

	return nil
}

// DeleteFolder deletes a folder (cascades to children).
func (s *FolderStorage) DeleteFolder(id string) error {
	if s.db == nil {
		return errors.New("folder storage database not available")
	}

	// First check if any reports reference this folder
	var reportCount int
	err := s.db.QueryRow(`
SELECT COUNT(*) FROM reports WHERE folder = ?
`, id).Scan(&reportCount)
	if err != nil {
		return fmt.Errorf("failed to check folder usage: %w", err)
	}

	if reportCount > 0 {
		return fmt.Errorf("cannot delete folder: %d reports are using it", reportCount)
	}

	result, err := s.db.Exec(`DELETE FROM report_folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("folder not found: %s", id)
	}

	return nil
}

// MoveReportToFolder updates the folder field of a report.
func (s *FolderStorage) MoveReportToFolder(reportID string, folderID *string) error {
	if s.db == nil {
		return errors.New("folder storage database not available")
	}

	// Validate folder exists if provided
	if folderID != nil {
		var count int
		err := s.db.QueryRow(`SELECT COUNT(*) FROM report_folders WHERE id = ?`, *folderID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to validate folder: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("folder not found: %s", *folderID)
		}
	}

	result, err := s.db.Exec(`
UPDATE reports SET folder = ?, updated_at = ? WHERE id = ?
`, folderID, time.Now().UTC(), reportID)
	if err != nil {
		return fmt.Errorf("failed to move report: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check move result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("report not found: %s", reportID)
	}

	return nil
}

// Tag represents a tag with usage count.
type Tag struct {
	Name         string `json:"name"`
	Color        string `json:"color,omitempty"`
	ReportCount  int    `json:"reportCount"`
}

// TagStorage manages report tags.
type TagStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTagStorage creates a new tag storage helper.
func NewTagStorage(db *sql.DB, logger *logrus.Logger) *TagStorage {
	return &TagStorage{db: db, logger: logger}
}

// EnsureSchema creates the tags table.
func (s *TagStorage) EnsureSchema() error {
	if s.db == nil {
		return errors.New("tag storage database not available")
	}

	statement := `
CREATE TABLE IF NOT EXISTS report_tags (
	name TEXT PRIMARY KEY,
	color TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

	if _, err := s.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure tag schema: %w", err)
	}

	return nil
}

// ListTags returns all tags with their usage counts.
func (s *TagStorage) ListTags() ([]Tag, error) {
	if s.db == nil {
		return nil, errors.New("tag storage database not available")
	}

	// Get all distinct tags from reports
	rows, err := s.db.Query(`
WITH tag_counts AS (
	SELECT
		value AS name,
		COUNT(*) AS report_count
	FROM reports, json_each(tags)
	GROUP BY value
)
SELECT
	COALESCE(t.name, tc.name) AS name,
	COALESCE(t.color, '') AS color,
	COALESCE(tc.report_count, 0) AS report_count
FROM report_tags t
LEFT JOIN tag_counts tc ON t.name = tc.name
UNION
SELECT
	tc.name,
	'' AS color,
	tc.report_count
FROM tag_counts tc
LEFT JOIN report_tags t ON tc.name = t.name
WHERE t.name IS NULL
ORDER BY report_count DESC, name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.Name, &tag.Color, &tag.ReportCount); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// CreateOrUpdateTag creates or updates a tag.
func (s *TagStorage) CreateOrUpdateTag(name, color string) error {
	if s.db == nil {
		return errors.New("tag storage database not available")
	}

	query := `
INSERT INTO report_tags (name, color, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
	color = excluded.color,
	updated_at = excluded.updated_at
`

	now := time.Now().UTC()
	_, err := s.db.Exec(query, name, color, now, now)
	if err != nil {
		return fmt.Errorf("failed to create/update tag: %w", err)
	}

	return nil
}

// ReportTemplate represents a reusable report template.
type ReportTemplate struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Category    string           `json:"category"`
	Thumbnail   string           `json:"thumbnail,omitempty"`
	Icon        string           `json:"icon,omitempty"`
	Tags        []string         `json:"tags"`
	Definition  ReportDefinition `json:"definition"`
	Filter      ReportFilterDefinition `json:"filter,omitempty"`
	Featured    bool             `json:"featured"`
	UsageCount  int              `json:"usageCount"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// TemplateStorage manages report templates.
type TemplateStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTemplateStorage creates a new template storage helper.
func NewTemplateStorage(db *sql.DB, logger *logrus.Logger) *TemplateStorage {
	return &TemplateStorage{db: db, logger: logger}
}

// EnsureSchema creates the templates table.
func (s *TemplateStorage) EnsureSchema() error {
	if s.db == nil {
		return errors.New("template storage database not available")
	}

	statement := `
CREATE TABLE IF NOT EXISTS report_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	category TEXT NOT NULL,
	thumbnail TEXT,
	icon TEXT,
	tags TEXT,
	definition TEXT NOT NULL,
	filter TEXT,
	featured BOOLEAN DEFAULT FALSE,
	usage_count INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_templates_category ON report_templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_featured ON report_templates(featured);
`

	if _, err := s.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure template schema: %w", err)
	}

	return nil
}

// SaveTemplate creates or updates a template.
func (s *TemplateStorage) SaveTemplate(template *ReportTemplate) error {
	if s.db == nil {
		return errors.New("template storage database not available")
	}
	if template == nil {
		return errors.New("template is nil")
	}

	if template.ID == "" {
		template.ID = uuid.NewString()
	}
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now().UTC()
	}
	template.UpdatedAt = time.Now().UTC()

	definitionJSON, err := json.Marshal(template.Definition)
	if err != nil {
		return fmt.Errorf("failed to marshal template definition: %w", err)
	}
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	filterJSON, err := json.Marshal(template.Filter)
	if err != nil {
		return fmt.Errorf("failed to marshal filter: %w", err)
	}

	query := `
INSERT INTO report_templates (
	id, name, description, category, thumbnail, icon, tags, definition, filter, featured, usage_count, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	description = excluded.description,
	category = excluded.category,
	thumbnail = excluded.thumbnail,
	icon = excluded.icon,
	tags = excluded.tags,
	definition = excluded.definition,
	filter = excluded.filter,
	featured = excluded.featured,
	updated_at = excluded.updated_at
`

	_, err = s.db.Exec(
		query,
		template.ID,
		template.Name,
		template.Description,
		template.Category,
		template.Thumbnail,
		template.Icon,
		string(tagsJSON),
		string(definitionJSON),
		string(filterJSON),
		template.Featured,
		template.UsageCount,
		template.CreatedAt,
		template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	return nil
}

// ListTemplates returns all templates.
func (s *TemplateStorage) ListTemplates(category string) ([]ReportTemplate, error) {
	if s.db == nil {
		return nil, errors.New("template storage database not available")
	}

	query := `
SELECT id, name, description, category, thumbnail, icon, tags, definition, filter, featured, usage_count, created_at, updated_at
FROM report_templates
`
	args := []interface{}{}

	if category != "" && category != "all" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY featured DESC, usage_count DESC, name ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []ReportTemplate
	for rows.Next() {
		var template ReportTemplate
		var tagsJSON, definitionJSON, filterJSON string

		if err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.Category,
			&template.Thumbnail,
			&template.Icon,
			&tagsJSON,
			&definitionJSON,
			&filterJSON,
			&template.Featured,
			&template.UsageCount,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if tagsJSON != "" {
			_ = json.Unmarshal([]byte(tagsJSON), &template.Tags)
		}
		if definitionJSON != "" {
			_ = json.Unmarshal([]byte(definitionJSON), &template.Definition)
		}
		if filterJSON != "" {
			_ = json.Unmarshal([]byte(filterJSON), &template.Filter)
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// GetTemplate retrieves a template by ID.
func (s *TemplateStorage) GetTemplate(id string) (*ReportTemplate, error) {
	if s.db == nil {
		return nil, errors.New("template storage database not available")
	}

	row := s.db.QueryRow(`
SELECT id, name, description, category, thumbnail, icon, tags, definition, filter, featured, usage_count, created_at, updated_at
FROM report_templates
WHERE id = ?
`, id)

	var template ReportTemplate
	var tagsJSON, definitionJSON, filterJSON string

	if err := row.Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.Category,
		&template.Thumbnail,
		&template.Icon,
		&tagsJSON,
		&definitionJSON,
		&filterJSON,
		&template.Featured,
		&template.UsageCount,
		&template.CreatedAt,
		&template.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("template not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if tagsJSON != "" {
		_ = json.Unmarshal([]byte(tagsJSON), &template.Tags)
	}
	if definitionJSON != "" {
		_ = json.Unmarshal([]byte(definitionJSON), &template.Definition)
	}
	if filterJSON != "" {
		_ = json.Unmarshal([]byte(filterJSON), &template.Filter)
	}

	return &template, nil
}

// IncrementTemplateUsage increments the usage count for a template.
func (s *TemplateStorage) IncrementTemplateUsage(id string) error {
	if s.db == nil {
		return errors.New("template storage database not available")
	}

	_, err := s.db.Exec(`
UPDATE report_templates SET usage_count = usage_count + 1 WHERE id = ?
`, id)
	if err != nil {
		return fmt.Errorf("failed to increment template usage: %w", err)
	}

	return nil
}
