package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements the Store interface using SQLite
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore creates a new SQLite catalog store
func NewSQLiteStore(dataDir string) (*SQLiteStore, error) {
	dbPath := filepath.Join(dataDir, "catalog.db")
	db, err := sql.Open("sqlite3", dbPath+"?_fk=1")
	if err != nil {
		return nil, fmt.Errorf("open catalog db: %w", err)
	}

	store := &SQLiteStore{db: db, path: dbPath}
	return store, nil
}

// NewStore creates a new SQLite catalog store from an existing database connection
// Useful for testing with in-memory databases
func NewStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db, path: ":memory:"}
}

// Initialize creates necessary tables and indexes
func (s *SQLiteStore) Initialize(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS table_catalog (
		id TEXT PRIMARY KEY,
		connection_id TEXT NOT NULL,
		schema_name TEXT NOT NULL,
		table_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		steward_user_id TEXT,
		tags_json TEXT DEFAULT '[]',
		organization_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		created_by TEXT NOT NULL,
		UNIQUE(connection_id, schema_name, table_name)
	);

	CREATE TABLE IF NOT EXISTS column_catalog (
		id TEXT PRIMARY KEY,
		table_catalog_id TEXT NOT NULL REFERENCES table_catalog(id) ON DELETE CASCADE,
		column_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		tags_json TEXT DEFAULT '[]',
		pii_type TEXT,
		pii_confidence REAL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(table_catalog_id, column_name)
	);

	CREATE TABLE IF NOT EXISTS catalog_tags (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		color TEXT DEFAULT '#808080',
		description TEXT DEFAULT '',
		organization_id TEXT,
		is_system INTEGER DEFAULT 0,
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_table_catalog_connection ON table_catalog(connection_id);
	CREATE INDEX IF NOT EXISTS idx_column_catalog_table ON column_catalog(table_catalog_id);
	CREATE INDEX IF NOT EXISTS idx_table_catalog_org ON table_catalog(organization_id);
	CREATE INDEX IF NOT EXISTS idx_catalog_tags_org ON catalog_tags(organization_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	// Insert system tags
	for _, tag := range SystemTags {
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO catalog_tags (id, name, color, description, is_system, created_at)
			VALUES (?, ?, ?, ?, 1, ?)
		`, tag.ID, tag.Name, tag.Color, tag.Description, time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("insert system tag %s: %w", tag.Name, err)
		}
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateTableEntry creates a new table catalog entry
func (s *SQLiteStore) CreateTableEntry(ctx context.Context, entry *TableCatalogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO table_catalog (id, connection_id, schema_name, table_name, description,
			steward_user_id, tags_json, organization_id, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.ConnectionID, entry.SchemaName, entry.TableName, entry.Description,
		entry.StewardUserID, string(tagsJSON), entry.OrganizationID,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339), entry.CreatedBy)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("table entry already exists: %s.%s", entry.SchemaName, entry.TableName)
		}
		return fmt.Errorf("insert table entry: %w", err)
	}

	return nil
}

// GetTableEntry retrieves a table catalog entry
func (s *SQLiteStore) GetTableEntry(ctx context.Context, connectionID, schema, table string) (*TableCatalogEntry, error) {
	var entry TableCatalogEntry
	var tagsJSON string
	var createdAt, updatedAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, connection_id, schema_name, table_name, description,
			steward_user_id, tags_json, organization_id, created_at, updated_at, created_by
		FROM table_catalog
		WHERE connection_id = ? AND schema_name = ? AND table_name = ?
	`, connectionID, schema, table).Scan(
		&entry.ID, &entry.ConnectionID, &entry.SchemaName, &entry.TableName, &entry.Description,
		&entry.StewardUserID, &tagsJSON, &entry.OrganizationID,
		&createdAt, &updatedAt, &entry.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get table entry: %w", err)
	}

	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	entry.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
		entry.Tags = []string{}
	}

	return &entry, nil
}

// UpdateTableEntry updates a table catalog entry
func (s *SQLiteStore) UpdateTableEntry(ctx context.Context, entry *TableCatalogEntry) error {
	entry.UpdatedAt = time.Now()

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE table_catalog
		SET description = ?, steward_user_id = ?, tags_json = ?, updated_at = ?
		WHERE id = ?
	`, entry.Description, entry.StewardUserID, string(tagsJSON),
		entry.UpdatedAt.Format(time.RFC3339), entry.ID)

	if err != nil {
		return fmt.Errorf("update table entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("table entry not found: %s", entry.ID)
	}

	return nil
}

// DeleteTableEntry deletes a table catalog entry
func (s *SQLiteStore) DeleteTableEntry(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM table_catalog WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete table entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("table entry not found: %s", id)
	}

	return nil
}

// ListTableEntries lists all table catalog entries for a connection
func (s *SQLiteStore) ListTableEntries(ctx context.Context, connectionID string) ([]*TableCatalogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, connection_id, schema_name, table_name, description,
			steward_user_id, tags_json, organization_id, created_at, updated_at, created_by
		FROM table_catalog
		WHERE connection_id = ?
		ORDER BY schema_name, table_name
	`, connectionID)
	if err != nil {
		return nil, fmt.Errorf("query table entries: %w", err)
	}
	defer rows.Close()

	var entries []*TableCatalogEntry
	for rows.Next() {
		var entry TableCatalogEntry
		var tagsJSON string
		var createdAt, updatedAt string

		err := rows.Scan(
			&entry.ID, &entry.ConnectionID, &entry.SchemaName, &entry.TableName, &entry.Description,
			&entry.StewardUserID, &tagsJSON, &entry.OrganizationID,
			&createdAt, &updatedAt, &entry.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("scan table entry: %w", err)
		}

		entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entry.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
			entry.Tags = []string{}
		}

		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// CreateColumnEntry creates a new column catalog entry
func (s *SQLiteStore) CreateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO column_catalog (id, table_catalog_id, column_name, description,
			tags_json, pii_type, pii_confidence, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.TableCatalogID, entry.ColumnName, entry.Description,
		string(tagsJSON), entry.PIIType, entry.PIIConfidence,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339))

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("column entry already exists: %s", entry.ColumnName)
		}
		return fmt.Errorf("insert column entry: %w", err)
	}

	return nil
}

// GetColumnEntry retrieves a column catalog entry
func (s *SQLiteStore) GetColumnEntry(ctx context.Context, tableID, column string) (*ColumnCatalogEntry, error) {
	var entry ColumnCatalogEntry
	var tagsJSON string
	var createdAt, updatedAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, table_catalog_id, column_name, description, tags_json, pii_type, pii_confidence, created_at, updated_at
		FROM column_catalog
		WHERE table_catalog_id = ? AND column_name = ?
	`, tableID, column).Scan(
		&entry.ID, &entry.TableCatalogID, &entry.ColumnName, &entry.Description,
		&tagsJSON, &entry.PIIType, &entry.PIIConfidence, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get column entry: %w", err)
	}

	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	entry.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
		entry.Tags = []string{}
	}

	return &entry, nil
}

// UpdateColumnEntry updates a column catalog entry
func (s *SQLiteStore) UpdateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error {
	entry.UpdatedAt = time.Now()

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE column_catalog
		SET description = ?, tags_json = ?, pii_type = ?, pii_confidence = ?, updated_at = ?
		WHERE id = ?
	`, entry.Description, string(tagsJSON), entry.PIIType, entry.PIIConfidence,
		entry.UpdatedAt.Format(time.RFC3339), entry.ID)

	if err != nil {
		return fmt.Errorf("update column entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("column entry not found: %s", entry.ID)
	}

	return nil
}

// ListColumnEntries lists all column catalog entries for a table
func (s *SQLiteStore) ListColumnEntries(ctx context.Context, tableID string) ([]*ColumnCatalogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_catalog_id, column_name, description, tags_json, pii_type, pii_confidence, created_at, updated_at
		FROM column_catalog
		WHERE table_catalog_id = ?
		ORDER BY column_name
	`, tableID)
	if err != nil {
		return nil, fmt.Errorf("query column entries: %w", err)
	}
	defer rows.Close()

	var entries []*ColumnCatalogEntry
	for rows.Next() {
		var entry ColumnCatalogEntry
		var tagsJSON string
		var createdAt, updatedAt string

		err := rows.Scan(
			&entry.ID, &entry.TableCatalogID, &entry.ColumnName, &entry.Description,
			&tagsJSON, &entry.PIIType, &entry.PIIConfidence, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan column entry: %w", err)
		}

		entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entry.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
			entry.Tags = []string{}
		}

		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// CreateTag creates a new custom tag
func (s *SQLiteStore) CreateTag(ctx context.Context, tag *CatalogTag) error {
	if tag.ID == "" {
		tag.ID = uuid.New().String()
	}
	tag.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO catalog_tags (id, name, color, description, organization_id, is_system, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tag.ID, tag.Name, tag.Color, tag.Description, tag.OrganizationID,
		boolToInt(tag.IsSystem), tag.CreatedAt.Format(time.RFC3339))

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("tag already exists: %s", tag.Name)
		}
		return fmt.Errorf("insert tag: %w", err)
	}

	return nil
}

// ListTags lists all tags for an organization (including system tags)
func (s *SQLiteStore) ListTags(ctx context.Context, orgID *string) ([]*CatalogTag, error) {
	var rows *sql.Rows
	var err error

	if orgID != nil {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, color, description, organization_id, is_system, created_at
			FROM catalog_tags
			WHERE organization_id IS NULL OR organization_id = ?
			ORDER BY is_system DESC, name
		`, *orgID)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, color, description, organization_id, is_system, created_at
			FROM catalog_tags
			WHERE organization_id IS NULL
			ORDER BY is_system DESC, name
		`)
	}

	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	var tags []*CatalogTag
	for rows.Next() {
		var tag CatalogTag
		var isSystem int
		var createdAt string

		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description,
			&tag.OrganizationID, &isSystem, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}

		tag.IsSystem = isSystem == 1
		tag.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tags = append(tags, &tag)
	}

	return tags, rows.Err()
}

// DeleteTag deletes a custom tag
func (s *SQLiteStore) DeleteTag(ctx context.Context, id string) error {
	// Check if it's a system tag
	var isSystem int
	err := s.db.QueryRowContext(ctx, `SELECT is_system FROM catalog_tags WHERE id = ?`, id).Scan(&isSystem)
	if err == sql.ErrNoRows {
		return fmt.Errorf("tag not found: %s", id)
	}
	if err != nil {
		return fmt.Errorf("check tag: %w", err)
	}

	if isSystem == 1 {
		return fmt.Errorf("cannot delete system tag")
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM catalog_tags WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	return nil
}

// SearchCatalog performs search across catalog entries
func (s *SQLiteStore) SearchCatalog(ctx context.Context, query string, filters *SearchFilters) (*SearchResults, error) {
	results := &SearchResults{
		Results: []*SearchResult{},
		Query:   query,
	}

	if filters == nil {
		filters = &SearchFilters{Limit: 50}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	// Search pattern using LIKE for simple text search
	searchPattern := "%" + query + "%"

	// Search tables
	tableQuery := `
		SELECT id, connection_id, schema_name, table_name, description, tags_json
		FROM table_catalog
		WHERE (table_name LIKE ? OR schema_name LIKE ? OR description LIKE ?)
	`
	args := []interface{}{searchPattern, searchPattern, searchPattern}

	if filters.ConnectionID != nil {
		tableQuery += " AND connection_id = ?"
		args = append(args, *filters.ConnectionID)
	}
	if filters.SchemaName != nil {
		tableQuery += " AND schema_name = ?"
		args = append(args, *filters.SchemaName)
	}
	if filters.OrganizationID != nil {
		tableQuery += " AND organization_id = ?"
		args = append(args, *filters.OrganizationID)
	}

	tableQuery += " ORDER BY table_name LIMIT ?"
	args = append(args, filters.Limit)

	rows, err := s.db.QueryContext(ctx, tableQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search tables: %w", err)
	}

	for rows.Next() {
		var result SearchResult
		var tagsJSON string

		err := rows.Scan(&result.ID, &result.ConnectionID, &result.SchemaName,
			&result.TableName, &result.Description, &tagsJSON)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan table result: %w", err)
		}

		result.Type = "table"
		if err := json.Unmarshal([]byte(tagsJSON), &result.Tags); err != nil {
			result.Tags = []string{}
		}
		result.RelevanceScore = calculateRelevance(query, result.TableName, result.Description)

		results.Results = append(results.Results, &result)
	}
	rows.Close()

	// Search columns
	columnQuery := `
		SELECT c.id, t.connection_id, t.schema_name, t.table_name, c.column_name, c.description, c.tags_json
		FROM column_catalog c
		INNER JOIN table_catalog t ON c.table_catalog_id = t.id
		WHERE (c.column_name LIKE ? OR c.description LIKE ?)
	`
	colArgs := []interface{}{searchPattern, searchPattern}

	if filters.ConnectionID != nil {
		columnQuery += " AND t.connection_id = ?"
		colArgs = append(colArgs, *filters.ConnectionID)
	}
	if filters.SchemaName != nil {
		columnQuery += " AND t.schema_name = ?"
		colArgs = append(colArgs, *filters.SchemaName)
	}

	columnQuery += " ORDER BY c.column_name LIMIT ?"
	colArgs = append(colArgs, filters.Limit)

	colRows, err := s.db.QueryContext(ctx, columnQuery, colArgs...)
	if err != nil {
		return nil, fmt.Errorf("search columns: %w", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var result SearchResult
		var tagsJSON string

		err := colRows.Scan(&result.ID, &result.ConnectionID, &result.SchemaName,
			&result.TableName, &result.ColumnName, &result.Description, &tagsJSON)
		if err != nil {
			return nil, fmt.Errorf("scan column result: %w", err)
		}

		result.Type = "column"
		if err := json.Unmarshal([]byte(tagsJSON), &result.Tags); err != nil {
			result.Tags = []string{}
		}
		result.RelevanceScore = calculateRelevance(query, result.ColumnName, result.Description)

		results.Results = append(results.Results, &result)
	}

	results.Total = len(results.Results)

	// Filter by tags if specified
	if len(filters.Tags) > 0 {
		filtered := []*SearchResult{}
		for _, result := range results.Results {
			for _, filterTag := range filters.Tags {
				for _, resultTag := range result.Tags {
					if filterTag == resultTag {
						filtered = append(filtered, result)
						break
					}
				}
			}
		}
		results.Results = filtered
		results.Total = len(filtered)
	}

	return results, nil
}

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func calculateRelevance(query, name, description string) float64 {
	query = strings.ToLower(query)
	name = strings.ToLower(name)
	description = strings.ToLower(description)

	score := 0.0

	// Exact name match
	if name == query {
		score += 1.0
	} else if strings.Contains(name, query) {
		score += 0.7
	}

	// Description match
	if strings.Contains(description, query) {
		score += 0.3
	}

	return score
}

// Verify SQLiteStore implements Store interface
var _ Store = (*SQLiteStore)(nil)
