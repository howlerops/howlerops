package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/backend-go/internal/services"
	"github.com/jbeck018/howlerops/backend-go/internal/sync"
)

// registerSyncRoutes registers HTTP routes for sync operations
func registerSyncRoutes(router *mux.Router, services *services.Services, logger *logrus.Logger) {
	// Create sync handler
	handler := &syncHandler{
		syncService: services.Sync,
		logger:      logger,
	}

	// Sync routes (all require authentication)
	syncRouter := router.PathPrefix("/api/sync").Subrouter()

	// Upload local changes to cloud
	syncRouter.HandleFunc("/upload", handler.handleUpload).Methods("POST", "OPTIONS")

	// Download remote changes from cloud
	syncRouter.HandleFunc("/download", handler.handleDownload).Methods("GET", "OPTIONS")

	// Get list of unresolved conflicts
	syncRouter.HandleFunc("/conflicts", handler.handleListConflicts).Methods("GET", "OPTIONS")

	// Resolve a specific conflict
	syncRouter.HandleFunc("/conflicts/{id}/resolve", handler.handleResolveConflict).Methods("POST", "OPTIONS")
}

// syncHandler handles sync-related HTTP requests
type syncHandler struct {
	syncService *sync.Service
	logger      *logrus.Logger
}

// handleUpload handles uploading local changes to the cloud
func (h *syncHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("Upload attempted without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req sync.SyncUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to parse upload request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set user ID from authenticated context
	req.UserID = userID

	// Validate request
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	if len(req.Changes) == 0 {
		http.Error(w, "no changes provided", http.StatusBadRequest)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": req.DeviceID,
		"changes":   len(req.Changes),
	}).Info("Processing sync upload")

	// Upload changes
	resp, err := h.syncService.Upload(r.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upload changes")
		http.Error(w, "Failed to upload changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.WithError(err).Error("Failed to encode upload response")
	}
}

// handleDownload handles downloading remote changes from the cloud
func (h *syncHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("Download attempted without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id query parameter is required", http.StatusBadRequest)
		return
	}

	sinceStr := r.URL.Query().Get("since")
	var since sync.Time
	if sinceStr != "" {
		if err := since.UnmarshalText([]byte(sinceStr)); err != nil {
			http.Error(w, "Invalid since parameter (use RFC3339 format)", http.StatusBadRequest)
			return
		}
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": deviceID,
		"since":     since,
	}).Info("Processing sync download")

	// Create download request
	req := &sync.SyncDownloadRequest{
		UserID:   userID,
		DeviceID: deviceID,
		Since:    since.Time, // Access embedded time.Time field
	}

	// Download changes
	resp, err := h.syncService.Download(r.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to download changes")
		http.Error(w, "Failed to download changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.WithError(err).Error("Failed to encode download response")
	}
}

// handleListConflicts lists all unresolved conflicts for the user
func (h *syncHandler) handleListConflicts(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("List conflicts attempted without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.WithField("user_id", userID).Info("Listing sync conflicts")

	// Get conflicts
	conflicts, err := h.syncService.ListConflicts(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list conflicts")
		http.Error(w, "Failed to list conflicts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"conflicts": conflicts,
		"count":     len(conflicts),
	}); err != nil {
		h.logger.WithError(err).Error("Failed to encode conflicts response")
	}
}

// handleResolveConflict resolves a specific conflict
func (h *syncHandler) handleResolveConflict(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("Resolve conflict attempted without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get conflict ID from URL
	vars := mux.Vars(r)
	conflictID := vars["id"]
	if conflictID == "" {
		http.Error(w, "conflict_id is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req sync.ConflictResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to parse conflict resolution request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set conflict ID from URL
	req.ConflictID = conflictID

	// Validate resolution strategy
	validStrategies := map[sync.ConflictResolutionStrategy]bool{
		sync.ConflictResolutionLastWriteWins: true,
		sync.ConflictResolutionKeepBoth:      true,
		sync.ConflictResolutionUserChoice:    true,
	}

	if !validStrategies[req.Strategy] {
		http.Error(w, "Invalid resolution strategy", http.StatusBadRequest)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"conflict_id": conflictID,
		"strategy":    req.Strategy,
	}).Info("Resolving sync conflict")

	// Resolve conflict
	resp, err := h.syncService.ResolveConflict(r.Context(), userID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to resolve conflict")
		http.Error(w, "Failed to resolve conflict: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.WithError(err).Error("Failed to encode conflict resolution response")
	}
}
