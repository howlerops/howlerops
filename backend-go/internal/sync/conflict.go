package sync

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// ConflictResolver handles sync conflicts using various strategies
type ConflictResolver struct {
	logger *logrus.Logger
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(logger *logrus.Logger) *ConflictResolver {
	return &ConflictResolver{
		logger: logger,
	}
}

// ResolveConnectionConflict handles conflicts when multiple users edit same shared connection
// Strategy: Last-write-wins based on updated_at timestamp
// Returns: winning version + conflict metadata
func (r *ConflictResolver) ResolveConnectionConflict(
	serverVersion *ConnectionTemplate,
	clientVersion *ConnectionTemplate,
) (*ConnectionTemplate, *ConflictMetadata, error) {
	r.logger.WithFields(logrus.Fields{
		"connection_id":    clientVersion.ID,
		"server_version":   serverVersion.SyncVersion,
		"client_version":   clientVersion.SyncVersion,
		"server_updated":   serverVersion.UpdatedAt,
		"client_updated":   clientVersion.UpdatedAt,
	}).Info("Resolving connection conflict")

	// Validate that we're comparing the same resource
	if serverVersion.ID != clientVersion.ID {
		return nil, nil, fmt.Errorf("cannot resolve conflict: resource IDs do not match")
	}

	metadata := &ConflictMetadata{
		ServerVersion: serverVersion.SyncVersion,
		ClientVersion: clientVersion.SyncVersion,
		ConflictedAt:  time.Now(),
	}

	// Last-write-wins: Compare timestamps
	if clientVersion.UpdatedAt.After(serverVersion.UpdatedAt) {
		metadata.Resolution = "client_wins"
		metadata.Reason = "client_newer"

		r.logger.WithFields(logrus.Fields{
			"connection_id": clientVersion.ID,
			"winner":        "client",
		}).Info("Conflict resolved: client wins")

		return clientVersion, metadata, nil
	}

	// Server version is newer or equal
	metadata.Resolution = "server_wins"
	metadata.Reason = "server_newer"

	r.logger.WithFields(logrus.Fields{
		"connection_id": serverVersion.ID,
		"winner":        "server",
	}).Info("Conflict resolved: server wins")

	return serverVersion, metadata, nil
}

// ResolveQueryConflict handles conflicts for saved queries
// Strategy: Last-write-wins based on updated_at timestamp
// Returns: winning version + conflict metadata
func (r *ConflictResolver) ResolveQueryConflict(
	serverVersion *SavedQuery,
	clientVersion *SavedQuery,
) (*SavedQuery, *ConflictMetadata, error) {
	r.logger.WithFields(logrus.Fields{
		"query_id":       clientVersion.ID,
		"server_version": serverVersion.SyncVersion,
		"client_version": clientVersion.SyncVersion,
		"server_updated": serverVersion.UpdatedAt,
		"client_updated": clientVersion.UpdatedAt,
	}).Info("Resolving query conflict")

	// Validate that we're comparing the same resource
	if serverVersion.ID != clientVersion.ID {
		return nil, nil, fmt.Errorf("cannot resolve conflict: resource IDs do not match")
	}

	metadata := &ConflictMetadata{
		ServerVersion: serverVersion.SyncVersion,
		ClientVersion: clientVersion.SyncVersion,
		ConflictedAt:  time.Now(),
	}

	// Last-write-wins: Compare timestamps
	if clientVersion.UpdatedAt.After(serverVersion.UpdatedAt) {
		metadata.Resolution = "client_wins"
		metadata.Reason = "client_newer"

		r.logger.WithFields(logrus.Fields{
			"query_id": clientVersion.ID,
			"winner":   "client",
		}).Info("Conflict resolved: client wins")

		return clientVersion, metadata, nil
	}

	// Server version is newer or equal
	metadata.Resolution = "server_wins"
	metadata.Reason = "server_newer"

	r.logger.WithFields(logrus.Fields{
		"query_id": serverVersion.ID,
		"winner":   "server",
	}).Info("Conflict resolved: server wins")

	return serverVersion, metadata, nil
}

// DetectConnectionConflict checks if a connection update would create a conflict
// Returns true if conflict detected
func (r *ConflictResolver) DetectConnectionConflict(
	serverVersion *ConnectionTemplate,
	clientVersion *ConnectionTemplate,
) bool {
	// Conflict exists if:
	// 1. Server has a newer sync version than client expected
	// 2. Server was updated after client's last sync
	if serverVersion.SyncVersion > clientVersion.SyncVersion {
		r.logger.WithFields(logrus.Fields{
			"connection_id":  clientVersion.ID,
			"server_version": serverVersion.SyncVersion,
			"client_version": clientVersion.SyncVersion,
		}).Warn("Sync version conflict detected")
		return true
	}

	return false
}

// DetectQueryConflict checks if a query update would create a conflict
// Returns true if conflict detected
func (r *ConflictResolver) DetectQueryConflict(
	serverVersion *SavedQuery,
	clientVersion *SavedQuery,
) bool {
	// Conflict exists if:
	// 1. Server has a newer sync version than client expected
	// 2. Server was updated after client's last sync
	if serverVersion.SyncVersion > clientVersion.SyncVersion {
		r.logger.WithFields(logrus.Fields{
			"query_id":       clientVersion.ID,
			"server_version": serverVersion.SyncVersion,
			"client_version": clientVersion.SyncVersion,
		}).Warn("Sync version conflict detected")
		return true
	}

	return false
}

// CreateConflictInfo creates conflict information for API response
func (r *ConflictResolver) CreateConflictInfo(
	resourceType string,
	resourceID string,
	metadata *ConflictMetadata,
) ConflictInfo {
	return ConflictInfo{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     *metadata,
	}
}

// ShouldRejectUpdate determines if an update should be rejected due to conflict
// Used when automatic resolution is not possible
func (r *ConflictResolver) ShouldRejectUpdate(
	serverUpdatedAt time.Time,
	clientUpdatedAt time.Time,
	serverSyncVersion int,
	clientSyncVersion int,
) (bool, string) {
	// Reject if client is trying to update with stale data
	if clientSyncVersion < serverSyncVersion {
		return true, fmt.Sprintf(
			"Update rejected: client sync version (%d) is behind server (%d). Pull latest changes first.",
			clientSyncVersion,
			serverSyncVersion,
		)
	}

	return false, ""
}

// MergeMetadata attempts to merge metadata from both versions
// Used for non-conflicting metadata fields
func (r *ConflictResolver) MergeMetadata(
	serverMetadata map[string]string,
	clientMetadata map[string]string,
) map[string]string {
	merged := make(map[string]string)

	// Start with server metadata
	for k, v := range serverMetadata {
		merged[k] = v
	}

	// Override with client metadata (client wins for metadata)
	for k, v := range clientMetadata {
		merged[k] = v
	}

	return merged
}
