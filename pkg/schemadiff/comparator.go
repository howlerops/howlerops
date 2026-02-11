package schemadiff

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jbeck018/howlerops/pkg/database"
)

// Comparator compares database schemas
type Comparator struct {
	// No state needed - stateless comparison
}

// NewComparator creates a new schema comparator
func NewComparator() *Comparator {
	return &Comparator{}
}

// CompareConnections compares schemas between two live database connections
func (c *Comparator) CompareConnections(ctx context.Context, source, target database.Database, sourceID, targetID string) (*SchemaDiff, error) {
	startTime := time.Now()

	// Get source schema
	sourceSnapshot, err := c.captureSnapshot(ctx, source, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture source schema: %w", err)
	}

	// Get target schema
	targetSnapshot, err := c.captureSnapshot(ctx, target, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture target schema: %w", err)
	}

	// Perform comparison
	diff := c.compareSnapshots(sourceSnapshot, targetSnapshot, sourceID, targetID)
	diff.Duration = time.Since(startTime)

	return diff, nil
}

// CompareWithSnapshot compares a live connection against a saved snapshot
func (c *Comparator) CompareWithSnapshot(ctx context.Context, live database.Database, liveID string, snapshot *SchemaSnapshot) (*SchemaDiff, error) {
	startTime := time.Now()

	// Get live schema
	liveSnapshot, err := c.captureSnapshot(ctx, live, liveID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture live schema: %w", err)
	}

	// Perform comparison
	diff := c.compareSnapshots(liveSnapshot, snapshot, liveID, snapshot.ID)
	diff.Duration = time.Since(startTime)

	return diff, nil
}

// captureSnapshot captures the current schema of a database
func (c *Comparator) captureSnapshot(ctx context.Context, db database.Database, id string) (*SchemaSnapshot, error) {
	snapshot := &SchemaSnapshot{
		ID:           id,
		ConnectionID: id,
		DatabaseType: db.GetDatabaseType(),
		Tables:       make(map[string][]database.TableInfo),
		Structures:   make(map[string]*database.TableStructure),
		CreatedAt:    time.Now(),
	}

	// Get all schemas
	schemas, err := db.GetSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}
	snapshot.Schemas = schemas

	// Get tables for each schema
	for _, schema := range schemas {
		tables, err := db.GetTables(ctx, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to get tables for schema %s: %w", schema, err)
		}
		snapshot.Tables[schema] = tables

		// Get structure for each table
		for _, table := range tables {
			structure, err := db.GetTableStructure(ctx, schema, table.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get structure for table %s.%s: %w", schema, table.Name, err)
			}
			key := fmt.Sprintf("%s.%s", schema, table.Name)
			snapshot.Structures[key] = structure
		}
	}

	return snapshot, nil
}

// compareSnapshots performs the actual comparison between two snapshots
func (c *Comparator) compareSnapshots(source, target *SchemaSnapshot, sourceID, targetID string) *SchemaDiff {
	diff := &SchemaDiff{
		SourceID:  sourceID,
		TargetID:  targetID,
		Timestamp: time.Now(),
		Tables:    []TableDiff{},
		Summary:   DiffSummary{},
	}

	// Build maps for efficient lookup
	sourceTableMap := buildTableMap(source)
	targetTableMap := buildTableMap(target)

	// Find all unique table keys (schema.table)
	allKeys := make(map[string]bool)
	for key := range sourceTableMap {
		allKeys[key] = true
	}
	for key := range targetTableMap {
		allKeys[key] = true
	}

	// Sort keys for deterministic output
	sortedKeys := make([]string, 0, len(allKeys))
	for key := range allKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Compare each table
	for _, key := range sortedKeys {
		sourceStructure := sourceTableMap[key]
		targetStructure := targetTableMap[key]

		tableDiff := c.compareTable(key, sourceStructure, targetStructure)
		diff.Tables = append(diff.Tables, tableDiff)

		// Update summary
		c.updateSummary(&diff.Summary, tableDiff)
	}

	return diff
}

// compareTable compares a single table between source and target
func (c *Comparator) compareTable(key string, source, target *database.TableStructure) TableDiff {
	parts := strings.SplitN(key, ".", 2)
	schema := parts[0]
	name := parts[1]

	tableDiff := TableDiff{
		Schema: schema,
		Name:   name,
	}

	// Determine table status
	if source == nil && target != nil {
		tableDiff.Status = DiffAdded
		return tableDiff
	}
	if source != nil && target == nil {
		tableDiff.Status = DiffRemoved
		return tableDiff
	}

	// Table exists in both - compare details
	tableDiff.Status = DiffUnchanged
	tableDiff.Columns = c.compareColumns(source.Columns, target.Columns)
	tableDiff.Indexes = c.compareIndexes(source.Indexes, target.Indexes)
	tableDiff.ForeignKeys = c.compareForeignKeys(source.ForeignKeys, target.ForeignKeys)

	// If any child diff exists, mark table as modified
	if len(tableDiff.Columns) > 0 || len(tableDiff.Indexes) > 0 || len(tableDiff.ForeignKeys) > 0 {
		tableDiff.Status = DiffModified
	}

	return tableDiff
}

// compareColumns compares columns between source and target
func (c *Comparator) compareColumns(source, target []database.ColumnInfo) []ColumnDiff {
	diffs := []ColumnDiff{}

	// Build maps for lookup
	sourceMap := make(map[string]database.ColumnInfo)
	targetMap := make(map[string]database.ColumnInfo)
	for _, col := range source {
		sourceMap[col.Name] = col
	}
	for _, col := range target {
		targetMap[col.Name] = col
	}

	// Find all column names
	allNames := make(map[string]bool)
	for name := range sourceMap {
		allNames[name] = true
	}
	for name := range targetMap {
		allNames[name] = true
	}

	// Compare each column
	for name := range allNames {
		sourceCol, sourceExists := sourceMap[name]
		targetCol, targetExists := targetMap[name]

		if !sourceExists && targetExists {
			// Column added
			diffs = append(diffs, ColumnDiff{
				Name:       name,
				Status:     DiffAdded,
				NewType:    targetCol.DataType,
				NewNull:    &targetCol.Nullable,
				NewDefault: targetCol.DefaultValue,
			})
		} else if sourceExists && !targetExists {
			// Column removed
			diffs = append(diffs, ColumnDiff{
				Name:       name,
				Status:     DiffRemoved,
				OldType:    sourceCol.DataType,
				OldNull:    &sourceCol.Nullable,
				OldDefault: sourceCol.DefaultValue,
			})
		} else {
			// Column exists in both - check for modifications
			if !columnsEqual(sourceCol, targetCol) {
				diffs = append(diffs, ColumnDiff{
					Name:       name,
					Status:     DiffModified,
					OldType:    sourceCol.DataType,
					NewType:    targetCol.DataType,
					OldNull:    &sourceCol.Nullable,
					NewNull:    &targetCol.Nullable,
					OldDefault: sourceCol.DefaultValue,
					NewDefault: targetCol.DefaultValue,
				})
			}
		}
	}

	return diffs
}

// compareIndexes compares indexes between source and target
func (c *Comparator) compareIndexes(source, target []database.IndexInfo) []IndexDiff {
	diffs := []IndexDiff{}

	// Build maps for lookup
	sourceMap := make(map[string]database.IndexInfo)
	targetMap := make(map[string]database.IndexInfo)
	for _, idx := range source {
		sourceMap[idx.Name] = idx
	}
	for _, idx := range target {
		targetMap[idx.Name] = idx
	}

	// Find all index names
	allNames := make(map[string]bool)
	for name := range sourceMap {
		allNames[name] = true
	}
	for name := range targetMap {
		allNames[name] = true
	}

	// Compare each index
	for name := range allNames {
		sourceIdx, sourceExists := sourceMap[name]
		targetIdx, targetExists := targetMap[name]

		if !sourceExists && targetExists {
			// Index added
			diffs = append(diffs, IndexDiff{
				Name:       name,
				Status:     DiffAdded,
				NewColumns: targetIdx.Columns,
				NewUnique:  &targetIdx.Unique,
				NewMethod:  targetIdx.Method,
			})
		} else if sourceExists && !targetExists {
			// Index removed
			diffs = append(diffs, IndexDiff{
				Name:       name,
				Status:     DiffRemoved,
				OldColumns: sourceIdx.Columns,
				OldUnique:  &sourceIdx.Unique,
				OldMethod:  sourceIdx.Method,
			})
		} else {
			// Index exists in both - check for modifications
			if !indexesEqual(sourceIdx, targetIdx) {
				diffs = append(diffs, IndexDiff{
					Name:       name,
					Status:     DiffModified,
					OldColumns: sourceIdx.Columns,
					NewColumns: targetIdx.Columns,
					OldUnique:  &sourceIdx.Unique,
					NewUnique:  &targetIdx.Unique,
					OldMethod:  sourceIdx.Method,
					NewMethod:  targetIdx.Method,
				})
			}
		}
	}

	return diffs
}

// compareForeignKeys compares foreign keys between source and target
func (c *Comparator) compareForeignKeys(source, target []database.ForeignKeyInfo) []FKDiff {
	diffs := []FKDiff{}

	// Build maps for lookup
	sourceMap := make(map[string]database.ForeignKeyInfo)
	targetMap := make(map[string]database.ForeignKeyInfo)
	for _, fk := range source {
		sourceMap[fk.Name] = fk
	}
	for _, fk := range target {
		targetMap[fk.Name] = fk
	}

	// Find all FK names
	allNames := make(map[string]bool)
	for name := range sourceMap {
		allNames[name] = true
	}
	for name := range targetMap {
		allNames[name] = true
	}

	// Compare each FK
	for name := range allNames {
		sourceFK, sourceExists := sourceMap[name]
		targetFK, targetExists := targetMap[name]

		if !sourceExists && targetExists {
			// FK added
			diffs = append(diffs, FKDiff{
				Name:          name,
				Status:        DiffAdded,
				NewColumns:    targetFK.Columns,
				NewRefTable:   targetFK.ReferencedTable,
				NewRefColumns: targetFK.ReferencedColumns,
				NewOnDelete:   targetFK.OnDelete,
				NewOnUpdate:   targetFK.OnUpdate,
			})
		} else if sourceExists && !targetExists {
			// FK removed
			diffs = append(diffs, FKDiff{
				Name:          name,
				Status:        DiffRemoved,
				OldColumns:    sourceFK.Columns,
				OldRefTable:   sourceFK.ReferencedTable,
				OldRefColumns: sourceFK.ReferencedColumns,
				OldOnDelete:   sourceFK.OnDelete,
				OldOnUpdate:   sourceFK.OnUpdate,
			})
		} else {
			// FK exists in both - check for modifications
			if !foreignKeysEqual(sourceFK, targetFK) {
				diffs = append(diffs, FKDiff{
					Name:          name,
					Status:        DiffModified,
					OldColumns:    sourceFK.Columns,
					NewColumns:    targetFK.Columns,
					OldRefTable:   sourceFK.ReferencedTable,
					NewRefTable:   targetFK.ReferencedTable,
					OldRefColumns: sourceFK.ReferencedColumns,
					NewRefColumns: targetFK.ReferencedColumns,
					OldOnDelete:   sourceFK.OnDelete,
					NewOnDelete:   targetFK.OnDelete,
					OldOnUpdate:   sourceFK.OnUpdate,
					NewOnUpdate:   targetFK.OnUpdate,
				})
			}
		}
	}

	return diffs
}

// updateSummary updates the diff summary based on a table diff
func (c *Comparator) updateSummary(summary *DiffSummary, tableDiff TableDiff) {
	switch tableDiff.Status {
	case DiffAdded:
		summary.TablesAdded++
	case DiffRemoved:
		summary.TablesRemoved++
	case DiffModified:
		summary.TablesModified++
	}

	for _, colDiff := range tableDiff.Columns {
		switch colDiff.Status {
		case DiffAdded:
			summary.ColumnsAdded++
		case DiffRemoved:
			summary.ColumnsRemoved++
		case DiffModified:
			summary.ColumnsModified++
		}
	}

	summary.IndexesChanged += len(tableDiff.Indexes)
	summary.FKsChanged += len(tableDiff.ForeignKeys)
}

// Helper functions

func buildTableMap(snapshot *SchemaSnapshot) map[string]*database.TableStructure {
	tableMap := make(map[string]*database.TableStructure)
	for key, structure := range snapshot.Structures {
		tableMap[key] = structure
	}
	return tableMap
}

func columnsEqual(a, b database.ColumnInfo) bool {
	if a.Name != b.Name || a.DataType != b.DataType || a.Nullable != b.Nullable {
		return false
	}
	if (a.DefaultValue == nil) != (b.DefaultValue == nil) {
		return false
	}
	if a.DefaultValue != nil && b.DefaultValue != nil && *a.DefaultValue != *b.DefaultValue {
		return false
	}
	return true
}

func indexesEqual(a, b database.IndexInfo) bool {
	if a.Name != b.Name || a.Unique != b.Unique || a.Method != b.Method {
		return false
	}
	if len(a.Columns) != len(b.Columns) {
		return false
	}
	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}
	return true
}

func foreignKeysEqual(a, b database.ForeignKeyInfo) bool {
	if a.Name != b.Name || a.ReferencedTable != b.ReferencedTable {
		return false
	}
	if a.OnDelete != b.OnDelete || a.OnUpdate != b.OnUpdate {
		return false
	}
	if len(a.Columns) != len(b.Columns) || len(a.ReferencedColumns) != len(b.ReferencedColumns) {
		return false
	}
	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}
	for i := range a.ReferencedColumns {
		if a.ReferencedColumns[i] != b.ReferencedColumns[i] {
			return false
		}
	}
	return true
}
