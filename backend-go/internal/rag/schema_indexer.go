package rag

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/sirupsen/logrus"
)

// EnhancedSchemaDocument models rich schema data suitable for embedding and retrieval.
type EnhancedSchemaDocument struct {
	TableName    string
	Schema       string
	ConnectionID string

	Columns []struct {
		Name         string
		Type         string
		IsNullable   bool
		IsPrimaryKey bool
		IsForeignKey bool
		Cardinality  int64
		Description  string
	}

	Relationships []struct {
		Type           string
		TargetTable    string
		LocalColumns   []string
		ForeignColumns []string
		JoinFrequency  int
	}

	Stats struct {
		RowCount         int64
		QueryFrequency   int
		LastQueried      time.Time
		CommonFilters    []string
		CommonAggregates []string
	}
}

// SchemaIndexer builds schema documents from live metadata without sampling values.
type SchemaIndexer struct {
	vectorStore      VectorStore
	embeddingService EmbeddingService
	logger           *logrus.Logger
	enricher         *SchemaEnricher
}

func NewSchemaIndexer(store VectorStore, embeddings EmbeddingService, logger *logrus.Logger) *SchemaIndexer {
	return &SchemaIndexer{vectorStore: store, embeddingService: embeddings, logger: logger}
}

// WithEnricher adds schema enrichment capability to the indexer
func (si *SchemaIndexer) WithEnricher(enricher *SchemaEnricher) *SchemaIndexer {
	si.enricher = enricher
	return si
}

// IndexTable indexes one table as a RAG document.
func (si *SchemaIndexer) IndexTable(ctx context.Context, connID, schema, table string, metadata map[string]interface{}) error {
	content := fmt.Sprintf("table %s.%s", schema, table)
	doc := &Document{
		ID:           fmt.Sprintf("schema:%s:%s.%s", connID, schema, table),
		ConnectionID: connID,
		Type:         DocumentTypeSchema,
		Content:      content,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := si.embeddingService.EmbedDocument(ctx, doc); err != nil {
		return err
	}
	return si.vectorStore.IndexDocument(ctx, doc)
}

// SchemaProvider abstracts schema discovery for a connection.
type SchemaProvider interface {
	GetSchemas(connID string) ([]string, error)
	GetTables(connID, schema string) ([]database.TableInfo, error)
	GetTableStructure(connID, schema, table string) (*database.TableStructure, error)
}

// IndexConnection walks schemas/tables and indexes table/column/relationship docs.
func (si *SchemaIndexer) IndexConnection(ctx context.Context, provider SchemaProvider, connID string) error {
	if si.embeddingService == nil || si.vectorStore == nil || provider == nil {
		return fmt.Errorf("indexer not initialized")
	}

	schemas, err := provider.GetSchemas(connID)
	if err != nil || len(schemas) == 0 {
		return err
	}

	// Limits to keep indexing fast
	const (
		maxTablesPerSchema = 50
	)

	for _, schemaName := range schemas {
		tables, err := provider.GetTables(connID, schemaName)
		if err != nil || len(tables) == 0 {
			continue
		}
		limit := len(tables)
		if limit > maxTablesPerSchema {
			limit = maxTablesPerSchema
		}
		for i := 0; i < limit; i++ {
			tbl := tables[i]
			structure, err := provider.GetTableStructure(connID, schemaName, tbl.Name)
			if err != nil {
				continue
			}
			_ = si.IndexTableDetails(ctx, connID, schemaName, tbl, structure)
		}
	}
	return nil
}

// IndexTableDetails creates table, column, and relationship documents and indexes them.
// This only uses metadata and never inspects data values.
func (si *SchemaIndexer) IndexTableDetails(
	ctx context.Context,
	connID string,
	schemaName string,
	table database.TableInfo,
	structure *database.TableStructure,
) error {
	if si.embeddingService == nil || si.vectorStore == nil {
		return fmt.Errorf("indexer not initialized")
	}

	const maxColumnsPerTable = 100

	// Build table-level text
	var b strings.Builder
	b.WriteString("table: ")
	b.WriteString(schemaName)
	b.WriteString(".")
	b.WriteString(table.Name)
	if table.Type != "" {
		b.WriteString(" type: ")
		b.WriteString(strings.ToLower(table.Type))
	}
	if table.Comment != "" {
		b.WriteString(" description: ")
		b.WriteString(table.Comment)
	}
	if table.RowCount > 0 {
		b.WriteString(" rows: ")
		b.WriteString(strconv.FormatInt(table.RowCount, 10))
	}

	// Append columns brief
	if structure != nil && len(structure.Columns) > 0 {
		count := len(structure.Columns)
		if count > maxColumnsPerTable {
			count = maxColumnsPerTable
		}
		b.WriteString(" columns: ")
		for ci := 0; ci < count; ci++ {
			col := structure.Columns[ci]
			b.WriteString(col.Name)
			b.WriteString(" (")
			b.WriteString(strings.ToLower(col.DataType))
			attrs := make([]string, 0, 3)
			if col.PrimaryKey {
				attrs = append(attrs, "pk")
			}
			if col.Unique {
				attrs = append(attrs, "unique")
			}
			if !col.Nullable {
				attrs = append(attrs, "not null")
			}
			if len(attrs) > 0 {
				b.WriteString(" ")
				b.WriteString(strings.Join(attrs, "/"))
			}
			b.WriteString(")")
			if ci < count-1 {
				b.WriteString(", ")
			}
		}
	}

	tableText := b.String()
	emb, err := si.embeddingService.EmbedText(ctx, tableText)
	if err == nil {
		_ = si.vectorStore.IndexDocument(ctx, &Document{
			ID:           fmt.Sprintf("schema:table:%s:%s.%s", connID, schemaName, table.Name),
			ConnectionID: connID,
			Type:         DocumentTypeSchema,
			Content:      tableText,
			Embedding:    emb,
			Metadata: map[string]interface{}{
				"table_name":  table.Name,
				"schema":      schemaName,
				"subtype":     "table",
				"row_count":   table.RowCount,
				"table_type":  table.Type,
				"description": table.Comment,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	// Column-level docs
	if structure != nil && len(structure.Columns) > 0 {
		count := len(structure.Columns)
		if count > maxColumnsPerTable {
			count = maxColumnsPerTable
		}
		for ci := 0; ci < count; ci++ {
			col := structure.Columns[ci]
			colText := fmt.Sprintf("column %s.%s.%s type %s", schemaName, table.Name, col.Name, strings.ToLower(col.DataType))
			if col.PrimaryKey {
				colText += " pk"
			}
			if col.Unique {
				colText += " unique"
			}
			if !col.Nullable {
				colText += " not_null"
			}

			// Add enrichment if available
			metadata := map[string]interface{}{
				"table_name":  table.Name,
				"schema":      schemaName,
				"column":      col.Name,
				"data_type":   col.DataType,
				"subtype":     "column",
				"primary_key": col.PrimaryKey,
				"unique":      col.Unique,
				"not_null":    !col.Nullable,
			}

			if si.enricher != nil {
				stats, err := si.enricher.EnrichColumn(ctx, schemaName, table.Name, col.Name, col.DataType)
				if err == nil && stats != nil {
					// Add sample values to content for better embeddings
					if len(stats.SampleValues) > 0 {
						colText += fmt.Sprintf(" examples: %s", strings.Join(stats.SampleValues, ", "))
					}

					// Add numeric range to content
					if stats.MinValue != nil && stats.MaxValue != nil {
						colText += fmt.Sprintf(" range: %v to %v", stats.MinValue, stats.MaxValue)
					}

					// Add distinct count hint
					if stats.DistinctCount > 0 {
						colText += fmt.Sprintf(" distinct_values: %d", stats.DistinctCount)
					}

					// Add to metadata
					metadata["distinct_count"] = stats.DistinctCount
					metadata["null_count"] = stats.NullCount
					metadata["sample_values"] = stats.SampleValues

					if len(stats.TopValues) > 0 {
						metadata["top_values"] = stats.TopValues
					}

					if stats.MinValue != nil {
						metadata["min_value"] = stats.MinValue
						metadata["max_value"] = stats.MaxValue
						metadata["avg_value"] = stats.AvgValue
					}
				}
			}

			cemb, err := si.embeddingService.EmbedText(ctx, colText)
			if err != nil {
				continue
			}
			_ = si.vectorStore.IndexDocument(ctx, &Document{
				ID:           fmt.Sprintf("schema:column:%s:%s.%s.%s", connID, schemaName, table.Name, col.Name),
				ConnectionID: connID,
				Type:         DocumentTypeSchema,
				Content:      colText,
				Embedding:    cemb,
				Metadata:     metadata,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		}
	}

	// Relationship-level docs
	if structure != nil && len(structure.ForeignKeys) > 0 {
		for _, fk := range structure.ForeignKeys {
			targetSchema := fk.ReferencedSchema
			targetTable := fk.ReferencedTable
			relText := fmt.Sprintf(
				"relationship %s.%s -> %s.%s by (%s -> %s)",
				schemaName, table.Name, targetSchema, targetTable,
				strings.Join(fk.Columns, ","), strings.Join(fk.ReferencedColumns, ","),
			)
			remb, err := si.embeddingService.EmbedText(ctx, relText)
			if err != nil {
				continue
			}
			_ = si.vectorStore.IndexDocument(ctx, &Document{
				ID:           fmt.Sprintf("schema:rel:%s:%s.%s->%s.%s:%s", connID, schemaName, table.Name, targetSchema, targetTable, strings.Join(fk.Columns, ",")),
				ConnectionID: connID,
				Type:         DocumentTypeSchema,
				Content:      relText,
				Embedding:    remb,
				Metadata: map[string]interface{}{
					"schema":          schemaName,
					"table_name":      table.Name,
					"target_schema":   targetSchema,
					"target_table":    targetTable,
					"local_columns":   fk.Columns,
					"foreign_columns": fk.ReferencedColumns,
					"subtype":         "relationship",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
		}
	}

	return nil
}

// IndexTableHierarchical creates a hierarchical structure with parent table and child documents
func (si *SchemaIndexer) IndexTableHierarchical(
	ctx context.Context,
	connectionID string,
	schema string,
	table database.TableInfo,
	structure *database.TableStructure,
) error {
	if si.embeddingService == nil || si.vectorStore == nil {
		return fmt.Errorf("indexer not initialized")
	}

	// Generate table summary (concise for vector search)
	tableSummary := si.generateTableSummary(schema, table, structure)

	// Generate table content (full details)
	tableContent := si.generateTableContent(schema, table, structure)

	// Create parent table document
	tableDoc := &Document{
		ID:           fmt.Sprintf("table:%s:%s.%s", connectionID, schema, table.Name),
		ConnectionID: connectionID,
		Type:         DocumentTypeSchema,
		Level:        DocumentLevelTable,
		Summary:      tableSummary,
		Content:      tableContent,
		Metadata: map[string]interface{}{
			"schema":       schema,
			"table":        table.Name,
			"column_count": len(structure.Columns),
			"row_count":    table.RowCount,
			"type":         "table_summary",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Embed parent document
	if err := si.embeddingService.EmbedDocument(ctx, tableDoc); err != nil {
		return fmt.Errorf("failed to embed table document: %w", err)
	}

	// Index parent
	if err := si.vectorStore.IndexDocument(ctx, tableDoc); err != nil {
		return fmt.Errorf("failed to index table document: %w", err)
	}

	// Create child documents for columns (not embedded initially)
	childIDs := make([]string, 0, len(structure.Columns))

	for _, col := range structure.Columns {
		colDoc := si.createColumnDocument(connectionID, schema, table.Name, col, tableDoc.ID)
		childIDs = append(childIDs, colDoc.ID)

		// Store without embedding (lazy loading)
		if err := si.vectorStore.StoreDocumentWithoutEmbedding(ctx, colDoc); err != nil {
			return fmt.Errorf("failed to store column document: %w", err)
		}
	}

	// Create child documents for indexes
	if structure.Indexes != nil {
		for _, idx := range structure.Indexes {
			idxDoc := si.createIndexDocument(connectionID, schema, table.Name, idx, tableDoc.ID)
			childIDs = append(childIDs, idxDoc.ID)

			if err := si.vectorStore.StoreDocumentWithoutEmbedding(ctx, idxDoc); err != nil {
				return fmt.Errorf("failed to store index document: %w", err)
			}
		}
	}

	// Create child documents for foreign keys
	if structure.ForeignKeys != nil {
		for _, fk := range structure.ForeignKeys {
			fkDoc := si.createForeignKeyDocument(connectionID, schema, table.Name, fk, tableDoc.ID)
			childIDs = append(childIDs, fkDoc.ID)

			if err := si.vectorStore.StoreDocumentWithoutEmbedding(ctx, fkDoc); err != nil {
				return fmt.Errorf("failed to store foreign key document: %w", err)
			}
		}
	}

	// Update parent with child references
	if err := si.vectorStore.UpdateDocumentMetadata(ctx, tableDoc.ID, map[string]interface{}{
		"schema":       schema,
		"table":        table.Name,
		"column_count": len(structure.Columns),
		"row_count":    table.RowCount,
		"type":         "table_summary",
		"child_ids":    childIDs,
	}); err != nil {
		return fmt.Errorf("failed to update parent with children: %w", err)
	}

	si.logger.WithFields(logrus.Fields{
		"table":     table.Name,
		"schema":    schema,
		"children":  len(childIDs),
		"parent_id": tableDoc.ID,
	}).Debug("Hierarchical table indexing completed")

	return nil
}

// generateTableSummary creates a concise summary for vector search
func (si *SchemaIndexer) generateTableSummary(
	schema string,
	table database.TableInfo,
	structure *database.TableStructure,
) string {
	keyColumns := si.getKeyColumns(structure.Columns, 5) // Top 5 important columns

	return fmt.Sprintf(
		"Table %s.%s (%d rows) with %d columns including: %s",
		schema,
		table.Name,
		table.RowCount,
		len(structure.Columns),
		strings.Join(keyColumns, ", "),
	)
}

// generateTableContent creates full table description
func (si *SchemaIndexer) generateTableContent(
	schema string,
	table database.TableInfo,
	structure *database.TableStructure,
) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Table: %s.%s\n\n", schema, table.Name))

	if table.Comment != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n\n", table.Comment))
	}

	b.WriteString(fmt.Sprintf("Row count: %d\n", table.RowCount))
	b.WriteString(fmt.Sprintf("Columns: %d\n\n", len(structure.Columns)))

	// Primary key
	if pk := si.findPrimaryKey(structure); pk != "" {
		b.WriteString(fmt.Sprintf("Primary key: %s\n", pk))
	}

	// Foreign keys
	if len(structure.ForeignKeys) > 0 {
		b.WriteString("Foreign keys:\n")
		for _, fk := range structure.ForeignKeys {
			b.WriteString(fmt.Sprintf("  - %s → %s.%s\n",
				strings.Join(fk.Columns, ", "), fk.ReferencedTable, strings.Join(fk.ReferencedColumns, ", ")))
		}
	}

	// Key columns overview
	b.WriteString("\nKey columns:\n")
	for i, col := range structure.Columns {
		if i >= 10 { // Limit to top 10
			b.WriteString(fmt.Sprintf("  ... and %d more columns\n", len(structure.Columns)-10))
			break
		}

		b.WriteString(fmt.Sprintf("  - %s (%s)", col.Name, col.DataType))
		if col.PrimaryKey {
			b.WriteString(" [PK]")
		}
		if !col.Nullable {
			b.WriteString(" [NOT NULL]")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// getKeyColumns returns the most important columns in priority order
func (si *SchemaIndexer) getKeyColumns(columns []database.ColumnInfo, limit int) []string {
	// Prioritize important columns
	priority := make([]database.ColumnInfo, 0, len(columns))

	// Primary keys first
	for _, col := range columns {
		if col.PrimaryKey {
			priority = append(priority, col)
		}
	}

	// Then common ID/name columns
	for _, col := range columns {
		if !col.PrimaryKey && (strings.Contains(strings.ToLower(col.Name), "id") ||
			strings.Contains(strings.ToLower(col.Name), "name")) {
			priority = append(priority, col)
		}
	}

	// Then other columns
	for _, col := range columns {
		if !col.PrimaryKey &&
			!strings.Contains(strings.ToLower(col.Name), "id") &&
			!strings.Contains(strings.ToLower(col.Name), "name") {
			priority = append(priority, col)
		}
	}

	// Take top N
	names := make([]string, 0, limit)
	for i := 0; i < len(priority) && i < limit; i++ {
		names = append(names, priority[i].Name)
	}

	return names
}

// findPrimaryKey finds the primary key column(s)
func (si *SchemaIndexer) findPrimaryKey(structure *database.TableStructure) string {
	var pkColumns []string
	for _, col := range structure.Columns {
		if col.PrimaryKey {
			pkColumns = append(pkColumns, col.Name)
		}
	}
	return strings.Join(pkColumns, ", ")
}

// createColumnDocument creates a column document
func (si *SchemaIndexer) createColumnDocument(
	connectionID, schema, table string,
	col database.ColumnInfo,
	parentID string,
) *Document {
	content := fmt.Sprintf(
		"Column %s.%s.%s: %s%s%s",
		schema, table, col.Name,
		col.DataType,
		func() string {
			if col.PrimaryKey {
				return " (Primary Key)"
			}
			return ""
		}(),
		func() string {
			if !col.Nullable {
				return " NOT NULL"
			}
			return ""
		}(),
	)

	if col.DefaultValue != nil && *col.DefaultValue != "" {
		content += fmt.Sprintf(", default: %s", *col.DefaultValue)
	}

	return &Document{
		ID:           fmt.Sprintf("column:%s:%s.%s.%s", connectionID, schema, table, col.Name),
		ParentID:     parentID,
		ConnectionID: connectionID,
		Type:         DocumentTypeSchema,
		Level:        DocumentLevelColumn,
		Content:      content,
		Metadata: map[string]interface{}{
			"schema":        schema,
			"table":         table,
			"column":        col.Name,
			"data_type":     col.DataType,
			"is_primary":    col.PrimaryKey,
			"nullable":      col.Nullable,
			"default_value": col.DefaultValue,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createIndexDocument creates an index document
func (si *SchemaIndexer) createIndexDocument(
	connectionID, schema, table string,
	idx database.IndexInfo,
	parentID string,
) *Document {
	content := fmt.Sprintf(
		"Index %s on %s.%s (%s)%s",
		idx.Name, schema, table,
		strings.Join(idx.Columns, ", "),
		func() string {
			if idx.Unique {
				return " UNIQUE"
			}
			return ""
		}(),
	)

	return &Document{
		ID:           fmt.Sprintf("index:%s:%s.%s.%s", connectionID, schema, table, idx.Name),
		ParentID:     parentID,
		ConnectionID: connectionID,
		Type:         DocumentTypeSchema,
		Level:        DocumentLevelIndex,
		Content:      content,
		Metadata: map[string]interface{}{
			"schema":    schema,
			"table":     table,
			"index":     idx.Name,
			"columns":   idx.Columns,
			"is_unique": idx.Unique,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createForeignKeyDocument creates a foreign key document
func (si *SchemaIndexer) createForeignKeyDocument(
	connectionID, schema, table string,
	fk database.ForeignKeyInfo,
	parentID string,
) *Document {
	content := fmt.Sprintf(
		"Foreign Key: %s.%s.%s → %s.%s.%s",
		schema, table, strings.Join(fk.Columns, ", "),
		fk.ReferencedSchema, fk.ReferencedTable, strings.Join(fk.ReferencedColumns, ", "),
	)

	return &Document{
		ID:           fmt.Sprintf("fk:%s:%s.%s->%s.%s", connectionID, schema, table, fk.ReferencedTable, strings.Join(fk.Columns, ",")),
		ParentID:     parentID,
		ConnectionID: connectionID,
		Type:         DocumentTypeSchema,
		Level:        DocumentLevelRelationship,
		Content:      content,
		Metadata: map[string]interface{}{
			"schema":             schema,
			"table":              table,
			"columns":            fk.Columns,
			"referenced_schema":  fk.ReferencedSchema,
			"referenced_table":   fk.ReferencedTable,
			"referenced_columns": fk.ReferencedColumns,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
