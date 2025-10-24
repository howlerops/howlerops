package queries

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// Handler handles HTTP requests for saved queries
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new queries handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// ShareQueryRequest represents the request to share a query
type ShareQueryRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
}

// ShareQuery handles POST /api/queries/{id}/share
// Share query in organization
func (h *Handler) ShareQuery(w http.ResponseWriter, r *http.Request) {
	queryID := mux.Vars(r)["id"]
	if queryID == "" {
		h.respondError(w, http.StatusBadRequest, "query ID is required")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ShareQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrganizationID == "" {
		h.respondError(w, http.StatusBadRequest, "organization_id is required")
		return
	}

	if err := h.service.ShareQuery(r.Context(), queryID, userID, req.OrganizationID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"query_id":        queryID,
			"user_id":         userID,
			"organization_id": req.OrganizationID,
		}).Error("Failed to share query")

		status := http.StatusInternalServerError
		if err.Error() == "query not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the creator can share this query" ||
			err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to share queries" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Query shared successfully",
	})
}

// UnshareQuery handles POST /api/queries/{id}/unshare
// Make query personal
func (h *Handler) UnshareQuery(w http.ResponseWriter, r *http.Request) {
	queryID := mux.Vars(r)["id"]
	if queryID == "" {
		h.respondError(w, http.StatusBadRequest, "query ID is required")
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.service.UnshareQuery(r.Context(), queryID, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"query_id": queryID,
			"user_id":  userID,
		}).Error("Failed to unshare query")

		status := http.StatusInternalServerError
		if err.Error() == "query not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only the creator can unshare this query" ||
			err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to unshare queries" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Query unshared successfully",
	})
}

// GetOrganizationQueries handles GET /api/organizations/{org_id}/queries
// List shared queries in org
func (h *Handler) GetOrganizationQueries(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["org_id"]
	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization ID is required")
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	queries, err := h.service.GetOrganizationQueries(r.Context(), orgID, userID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"organization_id": orgID,
			"user_id":         userID,
		}).Error("Failed to get organization queries")

		status := http.StatusInternalServerError
		if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to view queries" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"queries": queries,
		"count":   len(queries),
	})
}

// GetAccessibleQueries handles GET /api/queries/accessible
// Get all queries accessible to the user (personal + shared)
func (h *Handler) GetAccessibleQueries(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	queries, err := h.service.GetAccessibleQueries(r.Context(), userID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get accessible queries")
		h.respondError(w, http.StatusInternalServerError, "failed to get queries")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"queries": queries,
		"count":   len(queries),
	})
}

// CreateQuery handles POST /api/queries
func (h *Handler) CreateQuery(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var query turso.SavedQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateQuery(r.Context(), &query, userID); err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to create query")

		status := http.StatusInternalServerError
		if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to create queries in organization" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, query)
}

// UpdateQuery handles PUT /api/queries/{id}
func (h *Handler) UpdateQuery(w http.ResponseWriter, r *http.Request) {
	queryID := mux.Vars(r)["id"]
	if queryID == "" {
		h.respondError(w, http.StatusBadRequest, "query ID is required")
		return
	}

	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var query turso.SavedQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	query.ID = queryID

	if err := h.service.UpdateQuery(r.Context(), &query, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"query_id": queryID,
			"user_id":  userID,
		}).Error("Failed to update query")

		status := http.StatusInternalServerError
		if err.Error() == "query not found" {
			status = http.StatusNotFound
		} else if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to update this query" ||
			err.Error() == "cannot update another user's personal query" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, query)
}

// DeleteQuery handles DELETE /api/queries/{id}
func (h *Handler) DeleteQuery(w http.ResponseWriter, r *http.Request) {
	queryID := mux.Vars(r)["id"]
	if queryID == "" {
		h.respondError(w, http.StatusBadRequest, "query ID is required")
		return
	}

	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.service.DeleteQuery(r.Context(), queryID, userID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"query_id": queryID,
			"user_id":  userID,
		}).Error("Failed to delete query")

		status := http.StatusInternalServerError
		if err.Error() == "query not found" {
			status = http.StatusNotFound
		} else if err.Error() == "user not member of organization" ||
			err.Error() == "insufficient permissions to delete this query" ||
			err.Error() == "cannot delete another user's personal query" {
			status = http.StatusForbidden
		}

		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Query deleted successfully",
	})
}

// Helper methods

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}
