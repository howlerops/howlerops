package main

import (
	"context"
	"fmt"

	"github.com/jbeck018/howlerops/pkg/schemadiff"
)

// SchemaDiffService handles schema comparison operations
type SchemaDiffService struct {
	deps *SharedDeps
}

// NewSchemaDiffService creates a new schema diff service
func NewSchemaDiffService(deps *SharedDeps) *SchemaDiffService {
	return &SchemaDiffService{
		deps: deps,
	}
}

// SchemaDiff Service Wails Bindings
// These methods expose schema diff functionality to the frontend

// CompareConnectionSchemas compares schemas between two database connections
func (s *SchemaDiffService) CompareConnectionSchemas(sourceConnID, targetConnID string) (*schemadiff.SchemaDiff, error) {
	if s.deps.DatabaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}

	ctx := context.Background()

	// Get source and target database connections
	sourceDB, err := s.deps.DatabaseService.GetConnection(sourceConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source connection: %w", err)
	}

	targetDB, err := s.deps.DatabaseService.GetConnection(targetConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target connection: %w", err)
	}

	// Create comparator and perform comparison
	comparator := schemadiff.NewComparator()
	diff, err := comparator.CompareConnections(ctx, sourceDB, targetDB, sourceConnID, targetConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to compare schemas: %w", err)
	}

	return diff, nil
}

// CreateSchemaSnapshot creates a point-in-time snapshot of a database schema
func (s *SchemaDiffService) CreateSchemaSnapshot(connectionID, name string) (*schemadiff.SnapshotMetadata, error) {
	if s.deps.DatabaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	ctx := context.Background()

	// Get database connection
	db, err := s.deps.DatabaseService.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Create snapshot storage
	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	// Create snapshot
	snapshot, err := snapshotStore.CreateSnapshot(ctx, db, connectionID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Return metadata
	return &schemadiff.SnapshotMetadata{
		ID:           snapshot.ID,
		Name:         snapshot.Name,
		ConnectionID: snapshot.ConnectionID,
		DatabaseType: snapshot.DatabaseType,
		TableCount:   len(snapshot.Structures),
		CreatedAt:    snapshot.CreatedAt,
	}, nil
}

// ListSchemaSnapshots lists all saved schema snapshots
func (s *SchemaDiffService) ListSchemaSnapshots() ([]*schemadiff.SnapshotMetadata, error) {
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	return snapshotStore.ListSnapshots()
}

// GetSchemaSnapshot retrieves a specific snapshot
func (s *SchemaDiffService) GetSchemaSnapshot(snapshotID string) (*schemadiff.SchemaSnapshot, error) {
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	return snapshotStore.GetSnapshot(snapshotID)
}

// DeleteSchemaSnapshot deletes a saved snapshot
func (s *SchemaDiffService) DeleteSchemaSnapshot(snapshotID string) error {
	if s.deps.StorageManager == nil {
		return fmt.Errorf("storage manager not initialized")
	}

	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	return snapshotStore.DeleteSnapshot(snapshotID)
}

// CompareWithSnapshot compares a live connection against a saved snapshot
func (s *SchemaDiffService) CompareWithSnapshot(connectionID, snapshotID string) (*schemadiff.SchemaDiff, error) {
	if s.deps.DatabaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	ctx := context.Background()

	// Get live database connection
	db, err := s.deps.DatabaseService.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Get snapshot
	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	snapshot, err := snapshotStore.GetSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Create comparator and perform comparison
	comparator := schemadiff.NewComparator()
	diff, err := comparator.CompareWithSnapshot(ctx, db, connectionID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to compare with snapshot: %w", err)
	}

	return diff, nil
}

// GenerateMigrationSQL generates SQL statements to migrate from source to target schema
func (s *SchemaDiffService) GenerateMigrationSQL(sourceConnID, targetConnID string, reverse bool) (*schemadiff.MigrationScript, error) {
	if s.deps.DatabaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}

	ctx := context.Background()

	// Get database connections
	sourceDB, err := s.deps.DatabaseService.GetConnection(sourceConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source connection: %w", err)
	}

	targetDB, err := s.deps.DatabaseService.GetConnection(targetConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target connection: %w", err)
	}

	// Compare schemas
	comparator := schemadiff.NewComparator()
	diff, err := comparator.CompareConnections(ctx, sourceDB, targetDB, sourceConnID, targetConnID)
	if err != nil {
		return nil, fmt.Errorf("failed to compare schemas: %w", err)
	}

	// Generate migration
	generator := schemadiff.NewMigrationGenerator(sourceDB.GetDatabaseType())

	if reverse {
		return generator.GenerateReverseScript(diff)
	}
	return generator.GenerateForwardScript(diff)
}

// GenerateMigrationSQLFromSnapshot generates SQL to migrate from snapshot to live
func (s *SchemaDiffService) GenerateMigrationSQLFromSnapshot(connectionID, snapshotID string, reverse bool) (*schemadiff.MigrationScript, error) {
	if s.deps.DatabaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}
	if s.deps.StorageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	ctx := context.Background()

	// Get live database
	db, err := s.deps.DatabaseService.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Get snapshot
	snapshotStore, err := schemadiff.NewSnapshotStoreWithDir(s.deps.StorageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot store: %w", err)
	}

	snapshot, err := snapshotStore.GetSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Compare
	comparator := schemadiff.NewComparator()
	diff, err := comparator.CompareWithSnapshot(ctx, db, connectionID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to compare with snapshot: %w", err)
	}

	// Generate migration
	generator := schemadiff.NewMigrationGenerator(db.GetDatabaseType())

	if reverse {
		return generator.GenerateReverseScript(diff)
	}
	return generator.GenerateForwardScript(diff)
}
