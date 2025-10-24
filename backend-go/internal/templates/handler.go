package templates

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// Handler handles HTTP requests for templates and schedules
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a new templates handler
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all template and schedule routes
func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/templates", func(r chi.Router) {
		r.Use(authMiddleware)

		// Template routes
		r.Post("/", h.CreateTemplate)
		r.Get("/", h.ListTemplates)
		r.Get("/{id}", h.GetTemplate)
		r.Put("/{id}", h.UpdateTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
		r.Post("/{id}/execute", h.ExecuteTemplate)
		r.Get("/popular", h.GetPopularTemplates)
	})

	r.Route("/api/schedules", func(r chi.Router) {
		r.Use(authMiddleware)

		// Schedule routes
		r.Post("/", h.CreateSchedule)
		r.Get("/", h.ListSchedules)
		r.Get("/{id}", h.GetSchedule)
		r.Put("/{id}", h.UpdateSchedule)
		r.Delete("/{id}", h.DeleteSchedule)
		r.Post("/{id}/pause", h.PauseSchedule)
		r.Post("/{id}/resume", h.ResumeSchedule)
		r.Get("/{id}/executions", h.GetExecutionHistory)
		r.Get("/{id}/stats", h.GetExecutionStats)
	})
}

// Template Handlers

// CreateTemplate creates a new template
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input CreateTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	template, err := h.service.CreateTemplate(r.Context(), input, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// ListTemplates lists templates with filters
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filters := turso.TemplateFilters{}

	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		filters.OrganizationID = &orgID
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filters.Category = &category
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filters.SearchTerm = &search
	}

	if isPublicStr := r.URL.Query().Get("is_public"); isPublicStr != "" {
		isPublic := isPublicStr == "true"
		filters.IsPublic = &isPublic
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = offset
		}
	}

	templates, err := h.service.ListTemplates(r.Context(), userID, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list templates")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// GetTemplate retrieves a template by ID
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		http.Error(w, "Template ID required", http.StatusBadRequest)
		return
	}

	template, err := h.service.GetTemplate(r.Context(), templateID, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get template")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// UpdateTemplate updates a template
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		http.Error(w, "Template ID required", http.StatusBadRequest)
		return
	}

	var template turso.QueryTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	template.ID = templateID

	if err := h.service.UpdateTemplate(r.Context(), &template, userID); err != nil {
		h.logger.WithError(err).Error("Failed to update template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplate deletes a template
func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		http.Error(w, "Template ID required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteTemplate(r.Context(), templateID, userID); err != nil {
		h.logger.WithError(err).Error("Failed to delete template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteTemplate executes a template with parameters
func (h *Handler) ExecuteTemplate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	templateID := chi.URLParam(r, "id")
	if templateID == "" {
		http.Error(w, "Template ID required", http.StatusBadRequest)
		return
	}

	var input struct {
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sql, err := h.service.ExecuteTemplate(r.Context(), templateID, input.Parameters, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to execute template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sql": sql,
	})
}

// GetPopularTemplates returns popular templates
func (h *Handler) GetPopularTemplates(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get organization ID if provided
	var orgID *string
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		orgID = &orgIDStr
	}

	// This would need to be implemented in the service
	// For now, return an error
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// Schedule Handlers

// CreateSchedule creates a new schedule
func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input CreateScheduleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	schedule, err := h.service.CreateSchedule(r.Context(), input, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create schedule")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schedule)
}

// ListSchedules lists schedules with filters
func (h *Handler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filters := turso.ScheduleFilters{}

	if templateID := r.URL.Query().Get("template_id"); templateID != "" {
		filters.TemplateID = &templateID
	}

	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		filters.OrganizationID = &orgID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = &status
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = offset
		}
	}

	// TODO: Implement ListSchedules in service
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// GetSchedule retrieves a schedule by ID
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	// TODO: Implement GetSchedule in service with permission checks
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// UpdateSchedule updates a schedule
func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	// TODO: Implement UpdateSchedule in service
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// DeleteSchedule deletes a schedule
func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	// TODO: Implement DeleteSchedule in service
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// PauseSchedule pauses a schedule
func (h *Handler) PauseSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	if err := h.service.PauseSchedule(r.Context(), scheduleID, userID); err != nil {
		h.logger.WithError(err).Error("Failed to pause schedule")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResumeSchedule resumes a schedule
func (h *Handler) ResumeSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	if err := h.service.ResumeSchedule(r.Context(), scheduleID, userID); err != nil {
		h.logger.WithError(err).Error("Failed to resume schedule")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetExecutionHistory returns execution history for a schedule
func (h *Handler) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	// TODO: Implement GetExecutionHistory in service with permission checks
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// GetExecutionStats returns execution statistics for a schedule
func (h *Handler) GetExecutionStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scheduleID := chi.URLParam(r, "id")
	if scheduleID == "" {
		http.Error(w, "Schedule ID required", http.StatusBadRequest)
		return
	}

	// TODO: Implement GetExecutionStats in service with permission checks
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
