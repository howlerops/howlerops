package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/organization"
)

// OrgAwareHandler handles organization-aware sync operations
type OrgAwareHandler struct {
	service          *Service
	orgRepo          organization.Repository
	conflictResolver *ConflictResolver
	metadataTracker  *MetadataTracker
	logger           *logrus.Logger
}

// NewOrgAwareHandler creates a new organization-aware sync handler
func NewOrgAwareHandler(
	service *Service,
	orgRepo organization.Repository,
	logger *logrus.Logger,
) *OrgAwareHandler {
	return &OrgAwareHandler{
		service:          service,
		orgRepo:          orgRepo,
		conflictResolver: NewConflictResolver(logger),
		metadataTracker:  NewMetadataTracker(service.store, logger),
		logger:           logger,
	}
}

// HandlePull handles sync pull requests with organization filtering
// GET /api/sync/pull?since={timestamp}&org_id={optional}
func (h *OrgAwareHandler) HandlePull(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters
	sinceStr := r.URL.Query().Get("since")
	deviceID := r.URL.Query().Get("device_id")
	orgIDParam := r.URL.Query().Get("org_id")
	clientVersion := r.Header.Get("X-Client-Version")

	var since time.Time
	var err error

	if sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'since' parameter")
			return
		}
	} else {
		// Default to 30 days ago
		since = time.Now().Add(-30 * 24 * time.Hour)
	}

	if deviceID == "" {
		h.respondError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": deviceID,
		"since":     since,
		"org_id":    orgIDParam,
	}).Info("Processing org-aware sync pull")

	// Get user's organizations
	orgs, err := h.orgRepo.GetByUserID(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user organizations")
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve organizations")
		return
	}

	orgIDs := extractOrgIDs(orgs)

	// If specific org requested, validate access
	if orgIDParam != "" {
		hasAccess := false
		for _, orgID := range orgIDs {
			if orgID == orgIDParam {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			h.respondError(w, http.StatusForbidden, "no access to specified organization")
			return
		}
		orgIDs = []string{orgIDParam}
	}

	// Get accessible connections (personal + shared in user's orgs)
	connections, err := h.service.store.ListAccessibleConnections(ctx, userID, orgIDs, since)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list accessible connections")
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve connections")
		return
	}

	// Get accessible queries (personal + shared in user's orgs)
	queries, err := h.service.store.ListAccessibleQueries(ctx, userID, orgIDs, since)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list accessible queries")
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve queries")
		return
	}

	// Get query history (always personal)
	queryHistory, err := h.service.store.ListQueryHistory(ctx, userID, since, h.service.config.MaxHistoryItems)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list query history")
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve query history")
		return
	}

	// Log the sync operation
	var orgIDPtr *string
	if orgIDParam != "" {
		orgIDPtr = &orgIDParam
	}

	syncLog := h.metadataTracker.CreatePullLog(
		userID,
		deviceID,
		orgIDPtr,
		len(connections)+len(queries)+len(queryHistory),
		0, // conflicts tracked separately
		clientVersion,
	)

	if err := h.metadataTracker.LogSyncOperation(ctx, syncLog); err != nil {
		h.logger.WithError(err).Warn("Failed to log sync operation")
	}

	// Update sync metadata
	metadata := &SyncMetadata{
		UserID:     userID,
		DeviceID:   deviceID,
		LastSyncAt: time.Now(),
		Version:    clientVersion,
	}

	if err := h.metadataTracker.UpdateSyncMetadata(ctx, metadata); err != nil {
		h.logger.WithError(err).Warn("Failed to update sync metadata")
	}

	h.logger.WithFields(logrus.Fields{
		"connections":   len(connections),
		"queries":       len(queries),
		"query_history": len(queryHistory),
	}).Info("Sync pull completed")

	// Send response
	resp := &SyncDownloadResponse{
		Connections:   connections,
		SavedQueries:  queries,
		QueryHistory:  queryHistory,
		Conflicts:     []ConflictInfo{}, // Empty for pull
		SyncTimestamp: time.Now(),
		HasMore:       len(queryHistory) >= h.service.config.MaxHistoryItems,
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// HandlePush handles sync push requests with organization permission validation
// POST /api/sync/push
func (h *OrgAwareHandler) HandlePush(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request
	var req SyncUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set user ID from auth context
	req.UserID = userID

	clientVersion := r.Header.Get("X-Client-Version")

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": req.DeviceID,
		"changes":   len(req.Changes),
	}).Info("Processing org-aware sync push")

	// Validate all changes have proper permissions
	conflicts := []ConflictInfo{}
	rejected := []RejectedChange{}
	successCount := 0

	for _, change := range req.Changes {
		// Validate organization permissions
		if err := h.validateChangePermissions(ctx, userID, &change); err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"item_type": change.ItemType,
				"item_id":   change.ItemID,
			}).Warn("Permission validation failed")

			rejected = append(rejected, RejectedChange{
				Change: change,
				Reason: err.Error(),
			})
			continue
		}

		// Process change with conflict detection
		conflictInfo, err := h.processChangeWithConflict(ctx, userID, &change)
		if err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"item_type": change.ItemType,
				"item_id":   change.ItemID,
			}).Warn("Failed to process change")

			rejected = append(rejected, RejectedChange{
				Change: change,
				Reason: err.Error(),
			})
			continue
		}

		if conflictInfo != nil {
			conflicts = append(conflicts, *conflictInfo)
		}

		successCount++
	}

	// Log the sync operation
	syncLog := h.metadataTracker.CreatePushLog(
		userID,
		req.DeviceID,
		nil, // org ID from changes if needed
		successCount,
		len(conflicts),
		clientVersion,
	)

	if err := h.metadataTracker.LogSyncOperation(ctx, syncLog); err != nil {
		h.logger.WithError(err).Warn("Failed to log sync operation")
	}

	h.logger.WithFields(logrus.Fields{
		"success":   successCount,
		"conflicts": len(conflicts),
		"rejected":  len(rejected),
	}).Info("Sync push completed")

	// Send response
	resp := &SyncUploadResponse{
		Success:   len(rejected) == 0,
		SyncedAt:  time.Now(),
		Conflicts: convertConflictInfoToConflict(conflicts),
		Rejected:  rejected,
		Message:   fmt.Sprintf("Synced %d items, %d conflicts, %d rejected", successCount, len(conflicts), len(rejected)),
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// validateChangePermissions validates that user has permission to make the change
func (h *OrgAwareHandler) validateChangePermissions(ctx context.Context, userID string, change *SyncChange) error {
	// For delete actions, no special validation needed
	if change.Action == SyncActionDelete {
		return nil
	}

	// Extract organization info from change data
	data, ok := change.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid change data")
	}

	visibility, _ := data["visibility"].(string)
	orgIDInterface, hasOrgID := data["organization_id"]

	// Personal resources - user can always modify their own
	if visibility == "personal" || !hasOrgID || orgIDInterface == nil {
		return nil
	}

	// Shared resource - validate org membership and permissions
	orgID, ok := orgIDInterface.(string)
	if !ok {
		return fmt.Errorf("invalid organization_id format")
	}

	// Check if user is member of the organization
	member, err := h.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("not a member of organization: %w", err)
	}

	// Validate permissions based on resource type
	var requiredPerm organization.Permission

	switch change.ItemType {
	case SyncItemTypeConnection:
		switch change.Action {
		case SyncActionCreate:
			requiredPerm = organization.PermCreateConnections
		case SyncActionUpdate:
			requiredPerm = organization.PermUpdateConnections
		}
	case SyncItemTypeSavedQuery:
		switch change.Action {
		case SyncActionCreate:
			requiredPerm = organization.PermCreateQueries
		case SyncActionUpdate:
			requiredPerm = organization.PermUpdateQueries
		}
	default:
		return fmt.Errorf("unknown item type: %s", change.ItemType)
	}

	if !organization.HasPermission(member.Role, requiredPerm) {
		return fmt.Errorf("insufficient permissions: %s required", requiredPerm)
	}

	// For updates, check if user can update resources they don't own
	if change.Action == SyncActionUpdate {
		resourceOwnerID, _ := data["user_id"].(string)
		if !organization.CanUpdateResource(member.Role, resourceOwnerID, userID) {
			return fmt.Errorf("cannot update resource owned by another user")
		}
	}

	return nil
}

// processChangeWithConflict processes a change and detects conflicts
func (h *OrgAwareHandler) processChangeWithConflict(
	ctx context.Context,
	userID string,
	change *SyncChange,
) (*ConflictInfo, error) {
	switch change.ItemType {
	case SyncItemTypeConnection:
		return h.processConnectionChangeWithConflict(ctx, userID, change)
	case SyncItemTypeSavedQuery:
		return h.processQueryChangeWithConflict(ctx, userID, change)
	case SyncItemTypeQueryHistory:
		// Query history is append-only, no conflicts
		data := change.Data.(map[string]interface{})
		history, err := convertToQueryHistory(data)
		if err != nil {
			return nil, err
		}
		return nil, h.service.store.SaveQueryHistory(ctx, userID, history)
	default:
		return nil, fmt.Errorf("unknown item type: %s", change.ItemType)
	}
}

// processConnectionChangeWithConflict processes connection changes with conflict detection
func (h *OrgAwareHandler) processConnectionChangeWithConflict(
	ctx context.Context,
	userID string,
	change *SyncChange,
) (*ConflictInfo, error) {
	if change.Action == SyncActionDelete {
		return nil, h.service.store.DeleteConnection(ctx, userID, change.ItemID)
	}

	// Convert data to ConnectionTemplate
	data := change.Data.(map[string]interface{})
	conn, err := convertToConnectionTemplate(data)
	if err != nil {
		return nil, fmt.Errorf("invalid connection data: %w", err)
	}

	// Check for existing version
	existing, err := h.service.store.GetConnection(ctx, userID, conn.ID)
	if err == nil {
		// Resource exists - check for conflict
		if h.conflictResolver.DetectConnectionConflict(existing, conn) {
			// Resolve conflict
			winner, metadata, err := h.conflictResolver.ResolveConnectionConflict(existing, conn)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve conflict: %w", err)
			}

			// Save the winning version
			winner.SyncVersion = existing.SyncVersion + 1
			if err := h.service.store.SaveConnection(ctx, userID, winner); err != nil {
				return nil, fmt.Errorf("failed to save connection: %w", err)
			}

			// Return conflict info
			conflictInfo := h.conflictResolver.CreateConflictInfo(
				string(change.ItemType),
				change.ItemID,
				metadata,
			)
			return &conflictInfo, nil
		}
	}

	// No conflict - save normally
	conn.SyncVersion = change.SyncVersion + 1
	if err := h.service.store.SaveConnection(ctx, userID, conn); err != nil {
		return nil, fmt.Errorf("failed to save connection: %w", err)
	}

	return nil, nil
}

// processQueryChangeWithConflict processes query changes with conflict detection
func (h *OrgAwareHandler) processQueryChangeWithConflict(
	ctx context.Context,
	userID string,
	change *SyncChange,
) (*ConflictInfo, error) {
	if change.Action == SyncActionDelete {
		return nil, h.service.store.DeleteQuery(ctx, userID, change.ItemID)
	}

	// Convert data to SavedQuery
	data := change.Data.(map[string]interface{})
	query, err := convertToSavedQuery(data)
	if err != nil {
		return nil, fmt.Errorf("invalid query data: %w", err)
	}

	// Check for existing version
	existing, err := h.service.store.GetSavedQuery(ctx, userID, query.ID)
	if err == nil {
		// Resource exists - check for conflict
		if h.conflictResolver.DetectQueryConflict(existing, query) {
			// Resolve conflict
			winner, metadata, err := h.conflictResolver.ResolveQueryConflict(existing, query)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve conflict: %w", err)
			}

			// Save the winning version
			winner.SyncVersion = existing.SyncVersion + 1
			if err := h.service.store.SaveQuery(ctx, userID, winner); err != nil {
				return nil, fmt.Errorf("failed to save query: %w", err)
			}

			// Return conflict info
			conflictInfo := h.conflictResolver.CreateConflictInfo(
				string(change.ItemType),
				change.ItemID,
				metadata,
			)
			return &conflictInfo, nil
		}
	}

	// No conflict - save normally
	query.SyncVersion = change.SyncVersion + 1
	if err := h.service.store.SaveQuery(ctx, userID, query); err != nil {
		return nil, fmt.Errorf("failed to save query: %w", err)
	}

	return nil, nil
}

// Helper functions

func extractOrgIDs(orgs []*organization.Organization) []string {
	orgIDs := make([]string, len(orgs))
	for i, org := range orgs {
		orgIDs[i] = org.ID
	}
	return orgIDs
}

func convertConflictInfoToConflict(conflicts []ConflictInfo) []Conflict {
	result := make([]Conflict, len(conflicts))
	for i, info := range conflicts {
		result[i] = Conflict{
			ID:         fmt.Sprintf("conflict-%d", i),
			ItemType:   SyncItemType(info.ResourceType),
			ItemID:     info.ResourceID,
			DetectedAt: info.Metadata.ConflictedAt,
		}
	}
	return result
}

func convertToConnectionTemplate(data map[string]interface{}) (*ConnectionTemplate, error) {
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

func convertToSavedQuery(data map[string]interface{}) (*SavedQuery, error) {
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

func convertToQueryHistory(data map[string]interface{}) (*QueryHistory, error) {
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

func (h *OrgAwareHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (h *OrgAwareHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}
