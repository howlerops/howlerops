package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MetadataTracker tracks sync operations for audit and debugging
type MetadataTracker struct {
	store  Store
	logger *logrus.Logger
}

// NewMetadataTracker creates a new metadata tracker
func NewMetadataTracker(store Store, logger *logrus.Logger) *MetadataTracker {
	return &MetadataTracker{
		store:  store,
		logger: logger,
	}
}

// LogSyncOperation logs a sync operation for audit trail
func (m *MetadataTracker) LogSyncOperation(ctx context.Context, log *SyncLog) error {
	// Generate ID if not provided
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if log.SyncedAt.IsZero() {
		log.SyncedAt = time.Now()
	}

	m.logger.WithFields(logrus.Fields{
		"log_id":          log.ID,
		"user_id":         log.UserID,
		"organization_id": log.OrganizationID,
		"action":          log.Action,
		"resource_count":  log.ResourceCount,
		"conflict_count":  log.ConflictCount,
		"device_id":       log.DeviceID,
	}).Info("Logging sync operation")

	if err := m.store.SaveSyncLog(ctx, log); err != nil {
		return fmt.Errorf("failed to save sync log: %w", err)
	}

	return nil
}

// GetSyncHistory retrieves recent sync operations for a user
func (m *MetadataTracker) GetSyncHistory(ctx context.Context, userID string, limit int) ([]SyncLog, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	logs, err := m.store.ListSyncLogs(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync logs: %w", err)
	}

	return logs, nil
}

// GetLastSyncTime retrieves the last sync time for a user and device
func (m *MetadataTracker) GetLastSyncTime(ctx context.Context, userID, deviceID string) (time.Time, error) {
	metadata, err := m.store.GetSyncMetadata(ctx, userID, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// If metadata not found, return zero time
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get sync metadata: %w", err)
	}

	return metadata.LastSyncAt, nil
}

// UpdateSyncMetadata updates or creates sync metadata
func (m *MetadataTracker) UpdateSyncMetadata(ctx context.Context, metadata *SyncMetadata) error {
	m.logger.WithFields(logrus.Fields{
		"user_id":         metadata.UserID,
		"device_id":       metadata.DeviceID,
		"last_sync_at":    metadata.LastSyncAt,
		"total_synced":    metadata.TotalSynced,
		"conflicts_count": metadata.ConflictsCount,
	}).Info("Updating sync metadata")

	if err := m.store.UpdateSyncMetadata(ctx, metadata); err != nil {
		return fmt.Errorf("failed to update sync metadata: %w", err)
	}

	return nil
}

// CreatePullLog creates a log entry for a pull operation
func (m *MetadataTracker) CreatePullLog(
	userID string,
	deviceID string,
	orgID *string,
	resourceCount int,
	conflictCount int,
	clientVersion string,
) *SyncLog {
	return &SyncLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: orgID,
		Action:         "pull",
		ResourceCount:  resourceCount,
		ConflictCount:  conflictCount,
		DeviceID:       deviceID,
		ClientVersion:  clientVersion,
		SyncedAt:       time.Now(),
	}
}

// CreatePushLog creates a log entry for a push operation
func (m *MetadataTracker) CreatePushLog(
	userID string,
	deviceID string,
	orgID *string,
	resourceCount int,
	conflictCount int,
	clientVersion string,
) *SyncLog {
	return &SyncLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: orgID,
		Action:         "push",
		ResourceCount:  resourceCount,
		ConflictCount:  conflictCount,
		DeviceID:       deviceID,
		ClientVersion:  clientVersion,
		SyncedAt:       time.Now(),
	}
}

// GetSyncStatistics calculates sync statistics for a user
type SyncStatistics struct {
	TotalSyncs      int       `json:"total_syncs"`
	TotalPulls      int       `json:"total_pulls"`
	TotalPushes     int       `json:"total_pushes"`
	TotalConflicts  int       `json:"total_conflicts"`
	LastSyncAt      time.Time `json:"last_sync_at"`
	UniqueDevices   int       `json:"unique_devices"`
	ResourcesSynced int       `json:"resources_synced"`
}

// CalculateStatistics calculates sync statistics from sync logs
func (m *MetadataTracker) CalculateStatistics(ctx context.Context, userID string) (*SyncStatistics, error) {
	logs, err := m.store.ListSyncLogs(ctx, userID, 1000) // Get last 1000 syncs
	if err != nil {
		return nil, fmt.Errorf("failed to get sync logs: %w", err)
	}

	stats := &SyncStatistics{}
	deviceSet := make(map[string]bool)

	for _, log := range logs {
		stats.TotalSyncs++
		stats.TotalConflicts += log.ConflictCount
		stats.ResourcesSynced += log.ResourceCount

		switch log.Action {
		case "pull":
			stats.TotalPulls++
		case "push":
			stats.TotalPushes++
		}

		deviceSet[log.DeviceID] = true

		// Track most recent sync
		if log.SyncedAt.After(stats.LastSyncAt) {
			stats.LastSyncAt = log.SyncedAt
		}
	}

	stats.UniqueDevices = len(deviceSet)

	return stats, nil
}

// CleanupOldLogs removes sync logs older than specified duration
func (m *MetadataTracker) CleanupOldLogs(ctx context.Context, olderThan time.Duration) error {
	// This would require a new store method to delete old logs
	// For now, log the intent
	m.logger.WithFields(logrus.Fields{
		"older_than": olderThan,
	}).Info("Cleanup of old sync logs requested")

	// TODO: Implement cleanup in store
	return nil
}
