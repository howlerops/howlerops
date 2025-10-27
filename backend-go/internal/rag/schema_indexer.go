package rag

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/sql-studio/backend-go/pkg/database"
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
}

func NewSchemaIndexer(store VectorStore, embeddings EmbeddingService, logger *logrus.Logger) *SchemaIndexer {
    return &SchemaIndexer{vectorStore: store, embeddingService: embeddings, logger: logger}
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
                Metadata: map[string]interface{}{
                    "table_name":  table.Name,
                    "schema":      schemaName,
                    "column":      col.Name,
                    "data_type":   col.DataType,
                    "subtype":     "column",
                    "primary_key": col.PrimaryKey,
                    "unique":      col.Unique,
                    "not_null":    !col.Nullable,
                },
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
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
                    "schema":            schemaName,
                    "table_name":        table.Name,
                    "target_schema":     targetSchema,
                    "target_table":      targetTable,
                    "local_columns":     fk.Columns,
                    "foreign_columns":   fk.ReferencedColumns,
                    "subtype":           "relationship",
                },
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            })
        }
    }

    return nil
}


