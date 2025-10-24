package sync

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Store defines the interface for sync data storage
type Store interface {
	// Connections
	GetConnection(ctx context.Context, userID, connectionID string) (*ConnectionTemplate, error)
	ListConnections(ctx context.Context, userID string, since time.Time) ([]ConnectionTemplate, error)
	// New: Organization-aware connection listing
	ListAccessibleConnections(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]ConnectionTemplate, error)
	SaveConnection(ctx context.Context, userID string, conn *ConnectionTemplate) error
	DeleteConnection(ctx context.Context, userID, connectionID string) error

	// Saved Queries
	GetSavedQuery(ctx context.Context, userID, queryID string) (*SavedQuery, error)
	ListSavedQueries(ctx context.Context, userID string, since time.Time) ([]SavedQuery, error)
	// New: Organization-aware query listing
	ListAccessibleQueries(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]SavedQuery, error)
	SaveQuery(ctx context.Context, userID string, query *SavedQuery) error
	DeleteQuery(ctx context.Context, userID, queryID string) error

	// Query History
	ListQueryHistory(ctx context.Context, userID string, since time.Time, limit int) ([]QueryHistory, error)
	SaveQueryHistory(ctx context.Context, userID string, history *QueryHistory) error

	// Conflicts
	SaveConflict(ctx context.Context, userID string, conflict *Conflict) error
	GetConflict(ctx context.Context, userID, conflictID string) (*Conflict, error)
	ListConflicts(ctx context.Context, userID string, resolved bool) ([]Conflict, error)
	ResolveConflict(ctx context.Context, userID, conflictID string, resolution ConflictResolutionStrategy) error

	// Metadata
	GetSyncMetadata(ctx context.Context, userID, deviceID string) (*SyncMetadata, error)
	UpdateSyncMetadata(ctx context.Context, metadata *SyncMetadata) error

	// Sync Logs
	SaveSyncLog(ctx context.Context, log *SyncLog) error
	ListSyncLogs(ctx context.Context, userID string, limit int) ([]SyncLog, error)
}

// SyncLog tracks sync operations for audit and debugging
type SyncLog struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	OrganizationID *string   `json:"organization_id,omitempty"`
	Action         string    `json:"action"` // "pull" | "push"
	ResourceCount  int       `json:"resource_count"`
	ConflictCount  int       `json:"conflict_count"`
	DeviceID       string    `json:"device_id"`
	ClientVersion  string    `json:"client_version"`
	SyncedAt       time.Time `json:"synced_at"`
}

// Service provides sync functionality
type Service struct {
	store    Store
	logger   *logrus.Logger
	config   Config
}

// Config holds sync service configuration
type Config struct {
	MaxUploadSize       int64  // Maximum upload size in bytes
	ConflictStrategy    ConflictResolutionStrategy
	RetentionDays       int    // How long to keep deleted items
	MaxHistoryItems     int    // Maximum history items per sync
	EnableSanitization  bool   // Verify no credentials in uploads
}

// NewService creates a new sync service
func NewService(store Store, config Config, logger *logrus.Logger) *Service {
	if config.MaxUploadSize == 0 {
		config.MaxUploadSize = 10 * 1024 * 1024 // 10MB default
	}
	if config.MaxHistoryItems == 0 {
		config.MaxHistoryItems = 1000
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = 30
	}
	if config.ConflictStrategy == "" {
		config.ConflictStrategy = ConflictResolutionLastWriteWins
	}

	return &Service{
		store:  store,
		logger: logger,
		config: config,
	}
}

// Upload processes local changes and uploads them to the server
func (s *Service) Upload(ctx context.Context, req *SyncUploadRequest) (*SyncUploadResponse, error) {
	startTime := time.Now()

	s.logger.WithFields(logrus.Fields{
		"user_id":   req.UserID,
		"device_id": req.DeviceID,
		"changes":   len(req.Changes),
	}).Info("Processing sync upload")

	// Validate request
	if err := s.validateUploadRequest(req); err != nil {
		return nil, fmt.Errorf("invalid upload request: %w", err)
	}

	// Sanitize data if enabled
	if s.config.EnableSanitization {
		if err := s.sanitizeChanges(req.Changes); err != nil {
			return nil, fmt.Errorf("sanitization failed: %w", err)
		}
	}

	var conflicts []Conflict
	var rejected []RejectedChange
	successCount := 0

	// Process each change
	for _, change := range req.Changes {
		if err := s.processChange(ctx, req.UserID, &change, &conflicts); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"item_type": change.ItemType,
				"item_id":   change.ItemID,
				"action":    change.Action,
			}).Warn("Failed to process change")

			rejected = append(rejected, RejectedChange{
				Change: change,
				Reason: err.Error(),
			})
			continue
		}
		successCount++
	}

	// Update sync metadata
	metadata := &SyncMetadata{
		UserID:         req.UserID,
		DeviceID:       req.DeviceID,
		LastSyncAt:     time.Now(),
		TotalSynced:    int64(successCount),
		ConflictsCount: len(conflicts),
	}

	if err := s.store.UpdateSyncMetadata(ctx, metadata); err != nil {
		s.logger.WithError(err).Warn("Failed to update sync metadata")
	}

	duration := time.Since(startTime)
	s.logger.WithFields(logrus.Fields{
		"user_id":   req.UserID,
		"success":   successCount,
		"conflicts": len(conflicts),
		"rejected":  len(rejected),
		"duration":  duration.Milliseconds(),
	}).Info("Sync upload completed")

	return &SyncUploadResponse{
		Success:   len(rejected) == 0,
		SyncedAt:  time.Now(),
		Conflicts: conflicts,
		Rejected:  rejected,
		Message:   fmt.Sprintf("Synced %d items, %d conflicts, %d rejected", successCount, len(conflicts), len(rejected)),
	}, nil
}

// Download retrieves remote changes for the client
func (s *Service) Download(ctx context.Context, req *SyncDownloadRequest) (*SyncDownloadResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id":   req.UserID,
		"device_id": req.DeviceID,
		"since":     req.Since,
	}).Info("Processing sync download")

	// Get all changes since the requested time
	connections, err := s.store.ListConnections(ctx, req.UserID, req.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	savedQueries, err := s.store.ListSavedQueries(ctx, req.UserID, req.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved queries: %w", err)
	}

	queryHistory, err := s.store.ListQueryHistory(ctx, req.UserID, req.Since, s.config.MaxHistoryItems)
	if err != nil {
		return nil, fmt.Errorf("failed to list query history: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connections":    len(connections),
		"saved_queries":  len(savedQueries),
		"query_history":  len(queryHistory),
	}).Info("Sync download completed")

	return &SyncDownloadResponse{
		Connections:   connections,
		SavedQueries:  savedQueries,
		QueryHistory:  queryHistory,
		SyncTimestamp: time.Now(),
		HasMore:       len(queryHistory) >= s.config.MaxHistoryItems,
	}, nil
}

// ListConflicts returns all unresolved conflicts for a user
func (s *Service) ListConflicts(ctx context.Context, userID string) ([]Conflict, error) {
	return s.store.ListConflicts(ctx, userID, false)
}

// ResolveConflict resolves a conflict based on the chosen strategy
func (s *Service) ResolveConflict(ctx context.Context, userID string, req *ConflictResolutionRequest) (*ConflictResolutionResponse, error) {
	conflict, err := s.store.GetConflict(ctx, userID, req.ConflictID)
	if err != nil {
		return nil, fmt.Errorf("conflict not found: %w", err)
	}

	// Apply resolution strategy
	switch req.Strategy {
	case ConflictResolutionLastWriteWins:
		err = s.resolveLastWriteWins(ctx, userID, conflict)
	case ConflictResolutionKeepBoth:
		err = s.resolveKeepBoth(ctx, userID, conflict)
	case ConflictResolutionUserChoice:
		err = s.resolveUserChoice(ctx, userID, conflict, req.ChosenVersion)
	default:
		return nil, fmt.Errorf("invalid resolution strategy: %s", req.Strategy)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve conflict: %w", err)
	}

	// Mark conflict as resolved
	if err := s.store.ResolveConflict(ctx, userID, req.ConflictID, req.Strategy); err != nil {
		return nil, fmt.Errorf("failed to mark conflict as resolved: %w", err)
	}

	return &ConflictResolutionResponse{
		Success:    true,
		ResolvedAt: time.Now(),
		Message:    fmt.Sprintf("Conflict resolved using %s strategy", req.Strategy),
	}, nil
}

// processChange processes a single sync change
func (s *Service) processChange(ctx context.Context, userID string, change *SyncChange, conflicts *[]Conflict) error {
	// Calculate checksum if not provided
	if change.Checksum == "" {
		change.Checksum = s.calculateChecksum(change.Data)
	}

	switch change.ItemType {
	case SyncItemTypeConnection:
		return s.processConnectionChange(ctx, userID, change, conflicts)
	case SyncItemTypeSavedQuery:
		return s.processSavedQueryChange(ctx, userID, change, conflicts)
	case SyncItemTypeQueryHistory:
		return s.processQueryHistoryChange(ctx, userID, change, conflicts)
	default:
		return fmt.Errorf("unknown item type: %s", change.ItemType)
	}
}

// processConnectionChange processes a connection change
func (s *Service) processConnectionChange(ctx context.Context, userID string, change *SyncChange, conflicts *[]Conflict) error {
	conn, ok := change.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid connection data")
	}

	// Convert to ConnectionTemplate
	connData, err := s.convertToConnectionTemplate(conn)
	if err != nil {
		return fmt.Errorf("failed to convert connection data: %w", err)
	}

	switch change.Action {
	case SyncActionCreate, SyncActionUpdate:
		// Check for conflicts
		existing, err := s.store.GetConnection(ctx, userID, connData.ID)
		if err == nil && existing.SyncVersion > change.SyncVersion {
			// Conflict detected
			conflict := s.createConflict(change.ItemType, change.ItemID, existing, connData, change.DeviceID)
			*conflicts = append(*conflicts, *conflict)

			// Store conflict for later resolution
			if err := s.store.SaveConflict(ctx, userID, conflict); err != nil {
				s.logger.WithError(err).Warn("Failed to save conflict")
			}

			// Apply default conflict resolution strategy
			if s.config.ConflictStrategy == ConflictResolutionLastWriteWins {
				if existing.UpdatedAt.After(connData.UpdatedAt) {
					// Keep existing version
					return nil
				}
			}
		}

		// Save connection
		connData.SyncVersion = change.SyncVersion + 1
		return s.store.SaveConnection(ctx, userID, connData)

	case SyncActionDelete:
		return s.store.DeleteConnection(ctx, userID, change.ItemID)

	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}
}

// processSavedQueryChange processes a saved query change
func (s *Service) processSavedQueryChange(ctx context.Context, userID string, change *SyncChange, conflicts *[]Conflict) error {
	queryData, ok := change.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid query data")
	}

	savedQuery, err := s.convertToSavedQuery(queryData)
	if err != nil {
		return fmt.Errorf("failed to convert query data: %w", err)
	}

	switch change.Action {
	case SyncActionCreate, SyncActionUpdate:
		// Check for conflicts
		existing, err := s.store.GetSavedQuery(ctx, userID, savedQuery.ID)
		if err == nil && existing.SyncVersion > change.SyncVersion {
			conflict := s.createConflict(change.ItemType, change.ItemID, existing, savedQuery, change.DeviceID)
			*conflicts = append(*conflicts, *conflict)

			if err := s.store.SaveConflict(ctx, userID, conflict); err != nil {
				s.logger.WithError(err).Warn("Failed to save conflict")
			}

			if s.config.ConflictStrategy == ConflictResolutionLastWriteWins {
				if existing.UpdatedAt.After(savedQuery.UpdatedAt) {
					return nil
				}
			}
		}

		savedQuery.SyncVersion = change.SyncVersion + 1
		return s.store.SaveQuery(ctx, userID, savedQuery)

	case SyncActionDelete:
		return s.store.DeleteQuery(ctx, userID, change.ItemID)

	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}
}

// processQueryHistoryChange processes a query history change
func (s *Service) processQueryHistoryChange(ctx context.Context, userID string, change *SyncChange, conflicts *[]Conflict) error {
	historyData, ok := change.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid history data")
	}

	history, err := s.convertToQueryHistory(historyData)
	if err != nil {
		return fmt.Errorf("failed to convert history data: %w", err)
	}

	// Query history is append-only, no conflicts
	return s.store.SaveQueryHistory(ctx, userID, history)
}

// validateUploadRequest validates the upload request
func (s *Service) validateUploadRequest(req *SyncUploadRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if len(req.Changes) == 0 {
		return fmt.Errorf("no changes to sync")
	}

	// Estimate size
	data, err := json.Marshal(req.Changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	if int64(len(data)) > s.config.MaxUploadSize {
		return fmt.Errorf("upload size exceeds maximum allowed (%d bytes)", s.config.MaxUploadSize)
	}

	return nil
}

// sanitizeChanges ensures no sensitive data is being synced
func (s *Service) sanitizeChanges(changes []SyncChange) error {
	for _, change := range changes {
		if change.ItemType == SyncItemTypeConnection {
			conn, ok := change.Data.(map[string]interface{})
			if !ok {
				continue
			}

			// Check for password field
			if password, exists := conn["password"]; exists && password != "" {
				return fmt.Errorf("connection data contains password - credentials should not be synced")
			}

			// Check for SSH key
			if sshKey, exists := conn["ssh_key"]; exists && sshKey != "" {
				return fmt.Errorf("connection data contains SSH key - credentials should not be synced")
			}
		}
	}
	return nil
}

// createConflict creates a conflict object
func (s *Service) createConflict(itemType SyncItemType, itemID string, remote, local interface{}, deviceID string) *Conflict {
	return &Conflict{
		ID:       uuid.New().String(),
		ItemType: itemType,
		ItemID:   itemID,
		LocalVersion: &ConflictVersion{
			Data:      local,
			UpdatedAt: time.Now(),
			DeviceID:  deviceID,
		},
		RemoteVersion: &ConflictVersion{
			Data:      remote,
			UpdatedAt: time.Now(),
		},
		DetectedAt: time.Now(),
	}
}

// calculateChecksum calculates SHA-256 checksum of data
func (s *Service) calculateChecksum(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// Helper functions for type conversion
func (s *Service) convertToConnectionTemplate(data map[string]interface{}) (*ConnectionTemplate, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var conn ConnectionTemplate
	if err := json.Unmarshal(jsonData, &conn); err != nil {
		return nil, err
	}
	return &conn, nil
}

func (s *Service) convertToSavedQuery(data map[string]interface{}) (*SavedQuery, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var query SavedQuery
	if err := json.Unmarshal(jsonData, &query); err != nil {
		return nil, err
	}
	return &query, nil
}

func (s *Service) convertToQueryHistory(data map[string]interface{}) (*QueryHistory, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var history QueryHistory
	if err := json.Unmarshal(jsonData, &history); err != nil {
		return nil, err
	}
	return &history, nil
}

// Conflict resolution strategies
func (s *Service) resolveLastWriteWins(ctx context.Context, userID string, conflict *Conflict) error {
	// Compare timestamps and keep the most recent version
	if conflict.LocalVersion.UpdatedAt.After(conflict.RemoteVersion.UpdatedAt) {
		// Local is newer, no action needed
		return nil
	}
	// Remote is newer, already applied
	return nil
}

func (s *Service) resolveKeepBoth(ctx context.Context, userID string, conflict *Conflict) error {
	// Create a copy of the local version with a new ID
	switch conflict.ItemType {
	case SyncItemTypeConnection:
		conn, _ := conflict.LocalVersion.Data.(*ConnectionTemplate)
		newConn := *conn
		newConn.ID = uuid.New().String()
		newConn.Name = conn.Name + " (copy)"
		return s.store.SaveConnection(ctx, userID, &newConn)

	case SyncItemTypeSavedQuery:
		query, _ := conflict.LocalVersion.Data.(*SavedQuery)
		newQuery := *query
		newQuery.ID = uuid.New().String()
		newQuery.Name = query.Name + " (copy)"
		return s.store.SaveQuery(ctx, userID, &newQuery)
	}
	return nil
}

func (s *Service) resolveUserChoice(ctx context.Context, userID string, conflict *Conflict, choice string) error {
	if choice == "local" {
		// Keep local version
		switch conflict.ItemType {
		case SyncItemTypeConnection:
			conn, _ := conflict.LocalVersion.Data.(*ConnectionTemplate)
			return s.store.SaveConnection(ctx, userID, conn)
		case SyncItemTypeSavedQuery:
			query, _ := conflict.LocalVersion.Data.(*SavedQuery)
			return s.store.SaveQuery(ctx, userID, query)
		}
	}
	// Remote is already applied
	return nil
}
