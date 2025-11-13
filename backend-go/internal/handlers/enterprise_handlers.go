package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/domains"
	"github.com/sql-studio/backend-go/internal/middleware"
	"github.com/sql-studio/backend-go/internal/quotas"
	"github.com/sql-studio/backend-go/internal/sla"
	"github.com/sql-studio/backend-go/internal/whitelabel"
)

// EnterpriseHandlers handles all enterprise feature endpoints
type EnterpriseHandlers struct {
	whiteLabelService *whitelabel.Service
	domainService     *domains.Service
	quotaService      *quotas.Service
	slaMonitor        *sla.Monitor
	logger            *logrus.Logger
}

// NewEnterpriseHandlers creates a new enterprise handlers instance
func NewEnterpriseHandlers(
	whiteLabelService *whitelabel.Service,
	domainService *domains.Service,
	quotaService *quotas.Service,
	slaMonitor *sla.Monitor,
	logger *logrus.Logger,
) *EnterpriseHandlers {
	return &EnterpriseHandlers{
		whiteLabelService: whiteLabelService,
		domainService:     domainService,
		quotaService:      quotaService,
		slaMonitor:        slaMonitor,
		logger:            logger,
	}
}

// RegisterRoutes registers all enterprise routes
func (h *EnterpriseHandlers) RegisterRoutes(r *mux.Router) {
	// White-labeling routes
	r.HandleFunc("/api/organizations/{id}/white-label", h.GetWhiteLabelConfig).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/white-label", h.UpdateWhiteLabelConfig).Methods("PUT")
	r.HandleFunc("/api/white-label/css", h.GetBrandedCSS).Methods("GET")

	// Custom domains routes
	r.HandleFunc("/api/organizations/{id}/domains", h.ListDomains).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/domains", h.AddDomain).Methods("POST")
	r.HandleFunc("/api/organizations/{id}/domains/{domain}/verify", h.VerifyDomain).Methods("POST")
	r.HandleFunc("/api/organizations/{id}/domains/{domain}", h.RemoveDomain).Methods("DELETE")

	// Quotas and usage routes
	r.HandleFunc("/api/organizations/{id}/quotas", h.GetQuotas).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/quotas", h.UpdateQuotas).Methods("PUT")
	r.HandleFunc("/api/organizations/{id}/usage", h.GetUsageStatistics).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/usage/export", h.ExportUsageData).Methods("GET")

	// SLA routes
	r.HandleFunc("/api/organizations/{id}/sla", h.GetSLAMetrics).Methods("GET")
	r.HandleFunc("/api/organizations/{id}/sla/report", h.DownloadSLAReport).Methods("GET")
}

// White-labeling handlers

func (h *EnterpriseHandlers) GetWhiteLabelConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	config, err := h.whiteLabelService.GetConfigByOrganization(r.Context(), orgID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get white-label config")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

func (h *EnterpriseHandlers) UpdateWhiteLabelConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	var req whitelabel.UpdateWhiteLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, err := h.whiteLabelService.UpdateConfig(r.Context(), orgID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update white-label config")
		http.Error(w, fmt.Sprintf("Failed to update config: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

func (h *EnterpriseHandlers) GetBrandedCSS(w http.ResponseWriter, r *http.Request) {
	// Get domain from query parameter or Host header
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		domain = r.Host
	}

	config, err := h.whiteLabelService.GetConfigByDomain(r.Context(), domain)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get white-label config by domain")
		// Return default CSS
		config = &whitelabel.WhiteLabelConfig{}
	}

	css := h.whiteLabelService.GenerateBrandedCSS(config)

	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	if _, err := w.Write([]byte(css)); err != nil {
		h.logger.WithError(err).Error("Failed to write CSS response")
	}
}

// Custom domains handlers

func (h *EnterpriseHandlers) ListDomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	domains, err := h.domainService.ListDomains(r.Context(), orgID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list domains")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, domains)
}

func (h *EnterpriseHandlers) AddDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	var req domains.AddDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	verification, err := h.domainService.InitiateVerification(r.Context(), orgID, req.Domain)
	if err != nil {
		h.logger.WithError(err).Error("Failed to add domain")
		http.Error(w, fmt.Sprintf("Failed to add domain: %v", err), http.StatusBadRequest)
		return
	}

	// Include instructions in response
	response := map[string]interface{}{
		"verification": verification,
		"instructions": h.domainService.GetVerificationInstructions(verification),
	}

	respondJSON(w, http.StatusCreated, response)
}

func (h *EnterpriseHandlers) VerifyDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	domain := vars["domain"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	verification, err := h.domainService.VerifyDomain(r.Context(), orgID, domain)
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify domain")
		http.Error(w, fmt.Sprintf("Verification failed: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, verification)
}

func (h *EnterpriseHandlers) RemoveDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]
	domain := vars["domain"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := h.domainService.RemoveDomain(r.Context(), orgID, domain); err != nil {
		h.logger.WithError(err).Error("Failed to remove domain")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Quotas and usage handlers

func (h *EnterpriseHandlers) GetQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	quota, err := h.quotaService.GetQuota(r.Context(), orgID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get quotas")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, quota)
}

func (h *EnterpriseHandlers) UpdateQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	var req quotas.UpdateQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	quota, err := h.quotaService.UpdateQuota(r.Context(), orgID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update quotas")
		http.Error(w, fmt.Sprintf("Failed to update quotas: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, quota)
}

func (h *EnterpriseHandlers) GetUsageStatistics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get days parameter (default 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	stats, err := h.quotaService.GetUsageStatistics(r.Context(), orgID, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get usage statistics")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (h *EnterpriseHandlers) ExportUsageData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get days parameter (default 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	stats, err := h.quotaService.GetUsageStatistics(r.Context(), orgID, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get usage statistics")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate CSV
	csv := "Date,Queries,API Calls,Connections,Storage (MB)\n"
	for _, daily := range stats.DailyUsage {
		csv += fmt.Sprintf("%s,%d,%d,%d,%.2f\n",
			daily.Date,
			daily.QueriesCount,
			daily.APICallsCount,
			daily.ConnectionsCount,
			daily.StorageUsedMB,
		)
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=usage-%s.csv", time.Now().Format("2006-01-02")))
	if _, err := w.Write([]byte(csv)); err != nil {
		h.logger.WithError(err).Error("Failed to write CSV response")
	}
}

// SLA handlers

func (h *EnterpriseHandlers) GetSLAMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get days parameter (default 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	metrics, err := h.slaMonitor.GetLatestMetrics(r.Context(), orgID, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get SLA metrics")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

func (h *EnterpriseHandlers) DownloadSLAReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgID := vars["id"]

	// Verify access
	if err := middleware.VerifyOrgAccess(r.Context(), orgID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get date range (default last 30 days)
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get target SLA parameters
	targetUptime := 99.9
	if uptimeStr := r.URL.Query().Get("target_uptime"); uptimeStr != "" {
		if u, err := strconv.ParseFloat(uptimeStr, 64); err == nil {
			targetUptime = u
		}
	}

	targetResponseTime := 500.0 // 500ms
	if responseStr := r.URL.Query().Get("target_response_time"); responseStr != "" {
		if rt, err := strconv.ParseFloat(responseStr, 64); err == nil {
			targetResponseTime = rt
		}
	}

	report, err := h.slaMonitor.GenerateSLAReport(r.Context(), orgID, startDate, endDate, targetUptime, targetResponseTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate SLA report")
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	// Return as JSON
	respondJSON(w, http.StatusOK, report)
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
