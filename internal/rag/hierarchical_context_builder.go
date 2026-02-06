package rag

import (
	"context"
	"fmt"
)

// FetchRelevantSchemasHierarchical searches only parent table documents,
// then expands top matches to include child details
func (cb *ContextBuilder) FetchRelevantSchemasHierarchical(
	ctx context.Context,
	embedding []float32,
	connectionID string,
	limit int,
) ([]SchemaContext, error) {
	// Step 1: Search parent documents only (table level)
	filter := map[string]interface{}{
		"connection_id": connectionID,
		"level":         string(DocumentLevelTable),
	}

	parents, err := cb.vectorStore.SearchSimilar(ctx, embedding, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("parent search failed: %w", err)
	}

	// Step 2: Expand top-3 parents to get full details
	schemas := make([]SchemaContext, 0, len(parents))

	for i, parent := range parents {
		// For top 3, get full details including children
		if i < 3 {
			fullSchema, err := cb.expandTableDocument(ctx, parent)
			if err != nil {
				cb.logger.WithError(err).Warn("Failed to expand table document")
				// Use parent-only schema as fallback
				fullSchema = cb.parentToSchemaContext(parent)
			}
			schemas = append(schemas, fullSchema)
		} else {
			// For remaining, use summary only
			schemas = append(schemas, cb.parentToSchemaContext(parent))
		}
	}

	return schemas, nil
}

// expandTableDocument fetches child documents and assembles full schema context
func (cb *ContextBuilder) expandTableDocument(
	ctx context.Context,
	parent *Document,
) (SchemaContext, error) {
	// Get child IDs from metadata
	childIDsRaw, ok := parent.Metadata["child_ids"]
	if !ok {
		return cb.parentToSchemaContext(parent), nil
	}

	// Try to convert to []string (may be stored as []interface{})
	var childIDs []string
	switch v := childIDsRaw.(type) {
	case []string:
		childIDs = v
	case []interface{}:
		childIDs = make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				childIDs = append(childIDs, str)
			}
		}
	default:
		cb.logger.WithField("type", fmt.Sprintf("%T", childIDsRaw)).Warn("Unexpected child_ids type")
		return cb.parentToSchemaContext(parent), nil
	}

	if len(childIDs) == 0 {
		return cb.parentToSchemaContext(parent), nil
	}

	// Fetch children in batch
	children, err := cb.vectorStore.GetDocumentsBatch(ctx, childIDs)
	if err != nil {
		return SchemaContext{}, fmt.Errorf("failed to fetch children: %w", err)
	}

	// Assemble full schema context
	schema := cb.assembleSchemaContext(parent, children)
	return schema, nil
}

// assembleSchemaContext combines parent and child documents into SchemaContext
func (cb *ContextBuilder) assembleSchemaContext(
	parent *Document,
	children []*Document,
) SchemaContext {
	schema := cb.parentToSchemaContext(parent)

	// Add column details from children
	for _, child := range children {
		switch child.Level {
		case DocumentLevelColumn:
			colName, _ := child.Metadata["column"].(string)
			dataType, _ := child.Metadata["data_type"].(string)
			isPrimary, _ := child.Metadata["is_primary"].(bool)
			nullable, _ := child.Metadata["nullable"].(bool)

			schema.Columns = append(schema.Columns, ColumnInfo{
				Name:         colName,
				DataType:     dataType,
				IsPrimaryKey: isPrimary,
				IsNullable:   nullable,
			})

		case DocumentLevelIndex:
			idxName, _ := child.Metadata["index"].(string)
			isUnique, _ := child.Metadata["is_unique"].(bool)

			// Extract columns (may be []interface{} or []string)
			var cols []string
			if colsRaw, ok := child.Metadata["columns"]; ok {
				switch v := colsRaw.(type) {
				case []string:
					cols = v
				case []interface{}:
					for _, item := range v {
						if str, ok := item.(string); ok {
							cols = append(cols, str)
						}
					}
				}
			}

			schema.Indexes = append(schema.Indexes, IndexInfo{
				Name:     idxName,
				Columns:  cols,
				IsUnique: isUnique,
			})

		case DocumentLevelRelationship:
			targetTable, _ := child.Metadata["referenced_table"].(string)
			var localCol, foreignCol []string

			// Handle potential []interface{} format for columns
			if localRaw, ok := child.Metadata["columns"].([]interface{}); ok {
				for _, item := range localRaw {
					if str, ok := item.(string); ok {
						localCol = append(localCol, str)
					}
				}
			} else if localDirect, ok := child.Metadata["columns"].([]string); ok {
				localCol = localDirect
			}

			if foreignRaw, ok := child.Metadata["referenced_columns"].([]interface{}); ok {
				for _, item := range foreignRaw {
					if str, ok := item.(string); ok {
						foreignCol = append(foreignCol, str)
					}
				}
			} else if foreignDirect, ok := child.Metadata["referenced_columns"].([]string); ok {
				foreignCol = foreignDirect
			}

			if len(localCol) > 0 && len(foreignCol) > 0 {
				schema.Relationships = append(schema.Relationships, RelationshipInfo{
					Type:          "foreign_key",
					TargetTable:   targetTable,
					LocalColumn:   localCol[0],
					ForeignColumn: foreignCol[0],
				})
			}
		}
	}

	return schema
}

// parentToSchemaContext converts a parent document to SchemaContext (summary only)
func (cb *ContextBuilder) parentToSchemaContext(parent *Document) SchemaContext {
	meta := parent.Metadata
	var tableName string
	var rowCount int64

	if meta != nil {
		if name, ok := meta["table"].(string); ok {
			tableName = name
		}
		switch v := meta["row_count"].(type) {
		case int:
			rowCount = int64(v)
		case int64:
			rowCount = v
		case float64:
			rowCount = int64(v)
		}
	}

	// Use summary if available, otherwise content
	displayContent := parent.Summary
	if displayContent == "" {
		displayContent = parent.Content
	}

	return SchemaContext{
		TableName:     tableName,
		Description:   displayContent,
		RowCount:      rowCount,
		Relevance:     parent.Score,
		Columns:       []ColumnInfo{},
		Indexes:       []IndexInfo{},
		Relationships: []RelationshipInfo{},
	}
}
