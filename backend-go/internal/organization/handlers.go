package organization

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
	"github.com/sirupsen/logrus"
)

// Handler handles organization HTTP requests
type Handler struct {
	service ServiceInterface
	logger  *logrus.Logger
}

// NewHandler creates a new organization handler
func NewHandler(service ServiceInterface, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers organization routes on the router
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Organization CRUD
	r.HandleFunc("/api/organizations", h.CreateOrganization).Methods("POST")
	r.HandleFunc("/api/organizations", h.ListOrganizations).Methods("GET")
	r.HandleFunc("/api/organizations/{id}", h.GetOrganization).Methods("GET")
	r.HandleFunc("/api/organizations/{id}", h.UpdateOrganization).Methods("PUT")
	r.HandleFunc("/api/organizations/{id}", h.DeleteOrganization).Methods("DELETE")

	// Member management
	r.HandleFunc("/api/organizations/{id}/members", h.ListMembers).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/members/{userId}", h.UpdateMemberRole).Methods("PUT")
	r.HandleFunc("/api/organizations/{id}/members/{userId}", h.RemoveMember).Methods("DELETE")

	// Invitation management
	r.HandleFunc("/api/organizations/{id}/invitations", h.CreateInvitation).Methods("POST")
	r.HandleFunc("/api/organizations/{id}/invitations", h.ListOrgInvitations).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/invitations/{inviteId}", h.RevokeInvitation).Methods("DELETE")
	r.HandleFunc("/api/invitations", h.ListUserInvitations).Methods("GET")
	r.HandleFunc("/api/invitations/{id}/accept", h.AcceptInvitation).Methods("POST")
	r.HandleFunc("/api/invitations/{id}/decline", h.DeclineInvitation).Methods("POST")

	// Audit logs
	r.HandleFunc("/api/organizations/{id}/audit-logs", h.GetAuditLogs).Methods("GET")
}

// ====================================================================
// Organization Endpoints
// ====================================================================

// CreateOrganization handles POST /api/organizations
func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var input CreateOrganizationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if input.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Create organization
	org, err := h.service.CreateOrganization(ctx, userID, &input)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create organization")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, org.ID, userID, "organization.created", "organization", org.ID, r, map[string]interface{}{
		"organization_name": org.Name,
	})

	h.respondJSON(w, http.StatusCreated, org)
}

// ListOrganizations handles GET /api/organizations
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user's organizations
	orgs, err := h.service.GetUserOrganizations(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list organizations")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"organizations": orgs,
		"count":         len(orgs),
	})
}

// GetOrganization handles GET /api/organizations/:id
func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Get organization
	org, err := h.service.GetOrganization(ctx, orgID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to get organization")
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, org)
}

// UpdateOrganization handles PUT /api/organizations/:id
func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Parse request body
	var input UpdateOrganizationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update organization
	org, err := h.service.UpdateOrganization(ctx, orgID, userID, &input)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to update organization")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, userID, "organization.updated", "organization", orgID, r, map[string]interface{}{
		"changes": input,
	})

	h.respondJSON(w, http.StatusOK, org)
}

// DeleteOrganization handles DELETE /api/organizations/:id
func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Delete organization
	err := h.service.DeleteOrganization(ctx, orgID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to delete organization")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, userID, "organization.deleted", "organization", orgID, r, nil)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "organization deleted successfully",
	})
}

// ====================================================================
// Member Endpoints
// ====================================================================

// ListMembers handles GET /api/organizations/:id/members
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Get members
	members, err := h.service.GetMembers(ctx, orgID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to list members")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
		"count":   len(members),
	})
}

// UpdateMemberRole handles PUT /api/organizations/:id/members/:userId
func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	actorUserID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || actorUserID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get parameters from URL
	vars := mux.Vars(r)
	orgID := vars["id"]
	targetUserID := vars["userId"]

	if orgID == "" || targetUserID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id and user id are required")
		return
	}

	// Parse request body
	var input UpdateMemberRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate role
	if !input.Role.Validate() {
		h.respondError(w, http.StatusBadRequest, "invalid role")
		return
	}

	// Update member role
	err := h.service.UpdateMemberRole(ctx, orgID, targetUserID, actorUserID, input.Role)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to update member role")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, actorUserID, "member.role_updated", "member", targetUserID, r, map[string]interface{}{
		"target_user_id": targetUserID,
		"new_role":       input.Role,
	})

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "member role updated successfully",
	})
}

// RemoveMember handles DELETE /api/organizations/:id/members/:userId
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	actorUserID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || actorUserID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get parameters from URL
	vars := mux.Vars(r)
	orgID := vars["id"]
	targetUserID := vars["userId"]

	if orgID == "" || targetUserID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id and user id are required")
		return
	}

	// Remove member
	err := h.service.RemoveMember(ctx, orgID, targetUserID, actorUserID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to remove member")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, actorUserID, "member.removed", "member", targetUserID, r, map[string]interface{}{
		"target_user_id": targetUserID,
	})

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "member removed successfully",
	})
}

// ====================================================================
// Invitation Endpoints
// ====================================================================

// CreateInvitation handles POST /api/organizations/:id/invitations
func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Parse request body
	var input CreateInvitationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if input.Email == "" {
		h.respondError(w, http.StatusBadRequest, "email is required")
		return
	}

	if !input.Role.Validate() {
		h.respondError(w, http.StatusBadRequest, "invalid role")
		return
	}

	// Create invitation
	invitation, err := h.service.CreateInvitation(ctx, orgID, userID, &input)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			h.respondError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to create invitation")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, userID, "invitation.created", "invitation", invitation.ID, r, map[string]interface{}{
		"email": invitation.Email,
		"role":  invitation.Role,
	})

	h.respondJSON(w, http.StatusCreated, invitation)
}

// ListOrgInvitations handles GET /api/organizations/:id/invitations
func (h *Handler) ListOrgInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Get invitations
	invitations, err := h.service.GetInvitations(ctx, orgID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to list invitations")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"invitations": invitations,
		"count":       len(invitations),
	})
}

// ListUserInvitations handles GET /api/invitations
func (h *Handler) ListUserInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user email from query parameter or context
	// In a real implementation, you'd get the email from the user's profile
	email := r.URL.Query().Get("email")
	if email == "" {
		h.respondError(w, http.StatusBadRequest, "email parameter is required")
		return
	}

	// Get pending invitations for user's email
	invitations, err := h.service.GetPendingInvitationsForEmail(ctx, email)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list user invitations")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"invitations": invitations,
		"count":       len(invitations),
	})
}

// AcceptInvitation handles POST /api/invitations/:id/accept
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get invitation token from URL (using ID as token)
	vars := mux.Vars(r)
	token := vars["id"]

	if token == "" {
		h.respondError(w, http.StatusBadRequest, "invitation token is required")
		return
	}

	// Accept invitation
	org, err := h.service.AcceptInvitation(ctx, token, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "expired") {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already accepted") || strings.Contains(err.Error(), "already a member") {
			h.respondError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to accept invitation")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, org.ID, userID, "invitation.accepted", "invitation", token, r, map[string]interface{}{
		"organization_id": org.ID,
	})

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      "invitation accepted successfully",
		"organization": org,
	})
}

// DeclineInvitation handles POST /api/invitations/:id/decline
func (h *Handler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (optional for decline)
	userID, _ := ctx.Value(middleware.UserIDKey).(string)

	// Get invitation token from URL (using ID as token)
	vars := mux.Vars(r)
	token := vars["id"]

	if token == "" {
		h.respondError(w, http.StatusBadRequest, "invitation token is required")
		return
	}

	// Decline invitation
	err := h.service.DeclineInvitation(ctx, token)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to decline invitation")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log if user is authenticated
	if userID != "" {
		h.createAuditLog(ctx, "", userID, "invitation.declined", "invitation", token, r, nil)
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "invitation declined successfully",
	})
}

// RevokeInvitation handles DELETE /api/organizations/:id/invitations/:inviteId
func (h *Handler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get parameters from URL
	vars := mux.Vars(r)
	orgID := vars["id"]
	invitationID := vars["inviteId"]

	if orgID == "" || invitationID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id and invitation id are required")
		return
	}

	// Revoke invitation
	err := h.service.RevokeInvitation(ctx, orgID, invitationID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to revoke invitation")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create audit log
	h.createAuditLog(ctx, orgID, userID, "invitation.revoked", "invitation", invitationID, r, nil)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "invitation revoked successfully",
	})
}

// ====================================================================
// Audit Log Endpoints
// ====================================================================

// GetAuditLogs handles GET /api/organizations/:id/audit-logs
func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization ID from URL
	vars := mux.Vars(r)
	orgID := vars["id"]

	if orgID == "" {
		h.respondError(w, http.StatusBadRequest, "organization id is required")
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get audit logs
	logs, err := h.service.GetAuditLogs(ctx, orgID, userID, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") || strings.Contains(err.Error(), "not a member") {
			h.respondError(w, http.StatusForbidden, err.Error())
			return
		}
		h.logger.WithError(err).Error("Failed to get audit logs")
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}

// ====================================================================
// Helper Functions
// ====================================================================

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}

// createAuditLog creates an audit log entry
func (h *Handler) createAuditLog(ctx context.Context, orgID, userID, action, resourceType, resourceID string, r *http.Request, details map[string]interface{}) {
	// Extract IP address
	ipAddress := h.extractIPAddress(r)

	// Extract user agent
	userAgent := r.UserAgent()

	// Create audit log
	log := &AuditLog{
		OrganizationID: nil,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		IPAddress:      &ipAddress,
		UserAgent:      &userAgent,
		Details:        details,
	}

	// Set organization ID if provided
	if orgID != "" {
		log.OrganizationID = &orgID
	}

	// Create the log (errors are logged but don't fail the request)
	if err := h.service.CreateAuditLog(ctx, log); err != nil {
		h.logger.WithError(err).Warn("Failed to create audit log")
	}
}

// extractIPAddress extracts the client IP address from the request
func (h *Handler) extractIPAddress(r *http.Request) string {
	// Try X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
