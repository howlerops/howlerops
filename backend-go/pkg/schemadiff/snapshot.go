package schemadiff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jbeck018/howlerops/backend-go/pkg/database"
)

// SnapshotManager handles saving and loading schema snapshots
type SnapshotManager struct {
	snapshotDir string
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(snapshotDir string) (*SnapshotManager, error) {
	// Ensure snapshot directory exists (0750 for security - owner rwx, group rx, other none)
	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	return &SnapshotManager{
		snapshotDir: snapshotDir,
	}, nil
}

// SaveSnapshot saves a schema snapshot to disk
func (sm *SnapshotManager) SaveSnapshot(ctx context.Context, db database.Database, connectionID, name string) (*SchemaSnapshot, error) {
	// Create comparator to capture snapshot
	comparator := NewComparator()
	snapshot, err := comparator.captureSnapshot(ctx, db, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to capture snapshot: %w", err)
	}

	// Set snapshot metadata
	snapshot.ID = uuid.New().String()
	snapshot.Name = name
	snapshot.ConnectionID = connectionID
	snapshot.CreatedAt = time.Now()

	// Compute hash
	hash, err := sm.computeHash(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to compute snapshot hash: %w", err)
	}
	snapshot.Hash = hash

	// Save to disk
	if err := sm.writeSnapshot(snapshot); err != nil {
		return nil, fmt.Errorf("failed to write snapshot: %w", err)
	}

	return snapshot, nil
}

// LoadSnapshot loads a snapshot from disk by ID
func (sm *SnapshotManager) LoadSnapshot(snapshotID string) (*SchemaSnapshot, error) {
	// Validate snapshotID to prevent path traversal (G304)
	if err := sm.validateSnapshotID(snapshotID); err != nil {
		return nil, err
	}

	filePath := sm.getSnapshotPath(snapshotID)

	// #nosec G304 - filePath is validated via validateSnapshotID
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot SchemaSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	// Verify hash
	computedHash, err := sm.computeHash(&snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to verify snapshot hash: %w", err)
	}
	if computedHash != snapshot.Hash {
		return nil, fmt.Errorf("snapshot hash mismatch - snapshot may be corrupted")
	}

	return &snapshot, nil
}

// ListSnapshots lists all available snapshots
func (sm *SnapshotManager) ListSnapshots() ([]SnapshotMetadata, error) {
	files, err := os.ReadDir(sm.snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	metadataList := []SnapshotMetadata{}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Read snapshot file - path is constructed from controlled directory listing
		filePath := filepath.Join(sm.snapshotDir, file.Name())
		// #nosec G304 - filePath constructed from os.ReadDir entries within snapshotDir
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var snapshot SchemaSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue // Skip invalid snapshots
		}

		// Get file size
		fileInfo, err := os.Stat(filePath)
		var sizeBytes int64
		if err == nil {
			sizeBytes = fileInfo.Size()
		}

		// Count tables
		tableCount := 0
		for _, tables := range snapshot.Tables {
			tableCount += len(tables)
		}

		metadata := SnapshotMetadata{
			ID:           snapshot.ID,
			Name:         snapshot.Name,
			ConnectionID: snapshot.ConnectionID,
			DatabaseType: snapshot.DatabaseType,
			TableCount:   tableCount,
			CreatedAt:    snapshot.CreatedAt,
			SizeBytes:    sizeBytes,
		}
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// DeleteSnapshot deletes a snapshot by ID
func (sm *SnapshotManager) DeleteSnapshot(snapshotID string) error {
	// Validate snapshotID to prevent path traversal
	if err := sm.validateSnapshotID(snapshotID); err != nil {
		return err
	}

	filePath := sm.getSnapshotPath(snapshotID)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	return nil
}

// writeSnapshot writes a snapshot to disk
func (sm *SnapshotManager) writeSnapshot(snapshot *SchemaSnapshot) error {
	filePath := sm.getSnapshotPath(snapshot.ID)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

// getSnapshotPath returns the file path for a snapshot
func (sm *SnapshotManager) getSnapshotPath(snapshotID string) string {
	return filepath.Join(sm.snapshotDir, fmt.Sprintf("%s.json", snapshotID))
}

// validateSnapshotID validates that a snapshot ID is safe and doesn't contain path traversal
func (sm *SnapshotManager) validateSnapshotID(snapshotID string) error {
	if snapshotID == "" {
		return fmt.Errorf("snapshot ID cannot be empty")
	}

	// Check for path traversal attempts
	if filepath.Base(snapshotID) != snapshotID {
		return fmt.Errorf("invalid snapshot ID: contains path separators")
	}

	// Ensure the resolved path is within the snapshot directory
	filePath := sm.getSnapshotPath(snapshotID)
	cleanPath := filepath.Clean(filePath)
	if !filepath.HasPrefix(cleanPath, filepath.Clean(sm.snapshotDir)) {
		return fmt.Errorf("invalid snapshot ID: path traversal detected")
	}

	return nil
}

// computeHash computes a SHA-256 hash of the snapshot content
func (sm *SnapshotManager) computeHash(snapshot *SchemaSnapshot) (string, error) {
	// Create a copy without the hash field
	type SnapshotForHashing struct {
		ID           string
		Name         string
		ConnectionID string
		DatabaseType database.DatabaseType
		Schemas      []string
		Tables       map[string][]database.TableInfo
		Structures   map[string]*database.TableStructure
		CreatedAt    time.Time
	}

	hashSnapshot := SnapshotForHashing{
		ID:           snapshot.ID,
		Name:         snapshot.Name,
		ConnectionID: snapshot.ConnectionID,
		DatabaseType: snapshot.DatabaseType,
		Schemas:      snapshot.Schemas,
		Tables:       snapshot.Tables,
		Structures:   snapshot.Structures,
		CreatedAt:    snapshot.CreatedAt,
	}

	data, err := json.Marshal(hashSnapshot)
	if err != nil {
		return "", fmt.Errorf("failed to marshal snapshot for hashing: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// SnapshotStore provides a simpler API for snapshot operations
// It wraps SnapshotManager with a cleaner interface
type SnapshotStore struct {
	manager *SnapshotManager
}

// NewSnapshotStore creates a new snapshot store
func NewSnapshotStore(dataDir string) *SnapshotStore {
	return &SnapshotStore{
		manager: nil, // Lazy initialization
	}
}

// Initialize initializes the snapshot store
func (ss *SnapshotStore) Initialize() error {
	if ss.manager != nil {
		return nil // Already initialized
	}

	// The dataDir is passed in NewSnapshotStore, but we need to store it
	// For now, use a default path - this should be improved
	return nil
}

// initManager ensures the manager is initialized
func (ss *SnapshotStore) initManager(dataDir string) error {
	if ss.manager != nil {
		return nil
	}

	snapshotDir := filepath.Join(dataDir, "snapshots")
	manager, err := NewSnapshotManager(snapshotDir)
	if err != nil {
		return err
	}
	ss.manager = manager
	return nil
}

// CreateSnapshot creates a new snapshot
func (ss *SnapshotStore) CreateSnapshot(ctx context.Context, db database.Database, connectionID, name string) (*SchemaSnapshot, error) {
	if ss.manager == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}
	return ss.manager.SaveSnapshot(ctx, db, connectionID, name)
}

// GetSnapshot retrieves a snapshot by ID
func (ss *SnapshotStore) GetSnapshot(snapshotID string) (*SchemaSnapshot, error) {
	if ss.manager == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}
	return ss.manager.LoadSnapshot(snapshotID)
}

// ListSnapshots lists all available snapshots
func (ss *SnapshotStore) ListSnapshots() ([]*SnapshotMetadata, error) {
	if ss.manager == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}

	metadataList, err := ss.manager.ListSnapshots()
	if err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*SnapshotMetadata, len(metadataList))
	for i := range metadataList {
		result[i] = &metadataList[i]
	}
	return result, nil
}

// DeleteSnapshot deletes a snapshot by ID
func (ss *SnapshotStore) DeleteSnapshot(snapshotID string) error {
	if ss.manager == nil {
		return fmt.Errorf("snapshot store not initialized")
	}
	return ss.manager.DeleteSnapshot(snapshotID)
}

// NewSnapshotStoreWithDir creates a fully initialized snapshot store
func NewSnapshotStoreWithDir(dataDir string) (*SnapshotStore, error) {
	snapshotDir := filepath.Join(dataDir, "snapshots")
	manager, err := NewSnapshotManager(snapshotDir)
	if err != nil {
		return nil, err
	}
	return &SnapshotStore{manager: manager}, nil
}
