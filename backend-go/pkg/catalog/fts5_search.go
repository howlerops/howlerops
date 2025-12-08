package catalog

import (
	"context"
	"fmt"
	"strings"
)

// EnableFTS5 adds FTS5 full-text search tables to the catalog database
// This should be called after Initialize() to add enhanced search capabilities
func (s *SQLiteStore) EnableFTS5(ctx context.Context) error {
	schema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS catalog_fts USING fts5(
		entity_id UNINDEXED,
		entity_type UNINDEXED,
		connection_id UNINDEXED,
		schema_name,
		table_name,
		column_name,
		description,
		tags,
		content='',
		contentless_delete=1
	);

	-- Triggers to keep FTS index in sync with table_catalog
	CREATE TRIGGER IF NOT EXISTS table_catalog_ai AFTER INSERT ON table_catalog BEGIN
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		VALUES (new.id, 'table', new.connection_id, new.schema_name, new.table_name, '', new.description, new.tags_json);
	END;

	CREATE TRIGGER IF NOT EXISTS table_catalog_au AFTER UPDATE ON table_catalog BEGIN
		DELETE FROM catalog_fts WHERE entity_id = old.id AND entity_type = 'table';
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		VALUES (new.id, 'table', new.connection_id, new.schema_name, new.table_name, '', new.description, new.tags_json);
	END;

	CREATE TRIGGER IF NOT EXISTS table_catalog_ad AFTER DELETE ON table_catalog BEGIN
		DELETE FROM catalog_fts WHERE entity_id = old.id AND entity_type = 'table';
	END;

	-- Triggers for column_catalog
	CREATE TRIGGER IF NOT EXISTS column_catalog_ai AFTER INSERT ON column_catalog BEGIN
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		SELECT new.id, 'column', t.connection_id, t.schema_name, t.table_name, new.column_name, new.description, new.tags_json
		FROM table_catalog t WHERE t.id = new.table_catalog_id;
	END;

	CREATE TRIGGER IF NOT EXISTS column_catalog_au AFTER UPDATE ON column_catalog BEGIN
		DELETE FROM catalog_fts WHERE entity_id = old.id AND entity_type = 'column';
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		SELECT new.id, 'column', t.connection_id, t.schema_name, t.table_name, new.column_name, new.description, new.tags_json
		FROM table_catalog t WHERE t.id = new.table_catalog_id;
	END;

	CREATE TRIGGER IF NOT EXISTS column_catalog_ad AFTER DELETE ON column_catalog BEGIN
		DELETE FROM catalog_fts WHERE entity_id = old.id AND entity_type = 'column';
	END;
	`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("enable FTS5: %w", err)
	}

	// Populate existing data
	if err := s.rebuildFTS5Index(ctx); err != nil {
		return fmt.Errorf("rebuild FTS5 index: %w", err)
	}

	return nil
}

// rebuildFTS5Index rebuilds the FTS5 index from existing data
func (s *SQLiteStore) rebuildFTS5Index(ctx context.Context) error {
	// Clear existing index
	_, err := s.db.ExecContext(ctx, "DELETE FROM catalog_fts")
	if err != nil {
		return err
	}

	// Index tables
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		SELECT id, 'table', connection_id, schema_name, table_name, '', description, tags_json
		FROM table_catalog
	`)
	if err != nil {
		return fmt.Errorf("index tables: %w", err)
	}

	// Index columns
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_fts(entity_id, entity_type, connection_id, schema_name, table_name, column_name, description, tags)
		SELECT c.id, 'column', t.connection_id, t.schema_name, t.table_name, c.column_name, c.description, c.tags_json
		FROM column_catalog c
		INNER JOIN table_catalog t ON c.table_catalog_id = t.id
	`)
	if err != nil {
		return fmt.Errorf("index columns: %w", err)
	}

	return nil
}

// SearchCatalogFTS5 performs FTS5-powered search with BM25 ranking
func (s *SQLiteStore) SearchCatalogFTS5(ctx context.Context, query string, filters *SearchFilters) (*SearchResults, error) {
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

	// Build FTS5 query
	ftsQuery := buildFTS5Query(query)

	// Build SQL with filters
	sqlQuery := `
		SELECT 
			entity_id,
			entity_type,
			connection_id,
			schema_name,
			table_name,
			column_name,
			description,
			tags,
			bm25(catalog_fts) as score
		FROM catalog_fts
		WHERE catalog_fts MATCH ?
	`
	args := []interface{}{ftsQuery}

	if filters.ConnectionID != nil {
		sqlQuery += " AND connection_id = ?"
		args = append(args, *filters.ConnectionID)
	}
	if filters.SchemaName != nil {
		sqlQuery += " AND schema_name = ?"
		args = append(args, *filters.SchemaName)
	}

	sqlQuery += " ORDER BY score LIMIT ?"
	args = append(args, filters.Limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("FTS5 search: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result SearchResult
		var score float64
		var entityType, tags string

		err := rows.Scan(
			&result.ID,
			&entityType,
			&result.ConnectionID,
			&result.SchemaName,
			&result.TableName,
			&result.ColumnName,
			&result.Description,
			&tags,
			&score,
		)
		if err != nil {
			return nil, fmt.Errorf("scan FTS result: %w", err)
		}

		result.Type = entityType
		result.RelevanceScore = -score // BM25 returns negative scores

		// Parse tags JSON
		if tags != "" && tags != "[]" {
			var tagList []string
			if err := parseJSONArray(tags, &tagList); err == nil {
				result.Tags = tagList
			}
		}

		results.Results = append(results.Results, &result)
	}

	results.Total = len(results.Results)

	// Apply tag filter if specified
	if len(filters.Tags) > 0 {
		filtered := []*SearchResult{}
		for _, result := range results.Results {
			if hasAnyTag(result.Tags, filters.Tags) {
				filtered = append(filtered, result)
			}
		}
		results.Results = filtered
		results.Total = len(filtered)
	}

	return results, rows.Err()
}

// buildFTS5Query converts user query to FTS5 MATCH syntax
func buildFTS5Query(query string) string {
	// Escape quotes
	query = strings.ReplaceAll(query, `"`, `""`)

	// If already quoted, use as-is
	if strings.HasPrefix(query, `"`) && strings.HasSuffix(query, `"`) {
		return query
	}

	// Split into terms and build OR query with prefix matching
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return query
	}

	parts := make([]string, len(terms))
	for i, term := range terms {
		parts[i] = term + "*" // Prefix matching
	}

	return strings.Join(parts, " OR ")
}

// hasAnyTag checks if any of the filterTags are in resultTags
func hasAnyTag(resultTags, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, rt := range resultTags {
			if ft == rt {
				return true
			}
		}
	}
	return false
}

// parseJSONArray is a simple JSON array parser
func parseJSONArray(s string, out *[]string) error {
	s = strings.TrimSpace(s)
	if s == "[]" {
		*out = []string{}
		return nil
	}

	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			result = append(result, part)
		}
	}

	*out = result
	return nil
}
