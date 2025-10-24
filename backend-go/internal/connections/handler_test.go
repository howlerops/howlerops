package connections_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/connections"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// HTTP Handler Tests for API Endpoints
// ====================================================================

func TestShareConnectionEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}/share", handler.ShareConnection).Methods("POST")

	// Setup request
	reqBody := map[string]string{
		"organization_id": "org-test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/connections/conn-123/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user context
	ctx := context.WithValue(req.Context(), "user_id", "user-admin")
	req = req.WithContext(ctx)

	// Mock expectations
	conn := &turso.Connection{
		ID:        "conn-123",
		Name:      "Test DB",
		CreatedBy: "user-admin",
	}

	member := &organization.OrganizationMember{
		UserID:         "user-admin",
		OrganizationID: "org-test",
		Role:           organization.RoleAdmin,
	}

	mockStore.On("GetByID", mock.Anything, "conn-123").Return(conn, nil)
	mockOrgRepo.On("GetMember", mock.Anything, "org-test", "user-admin").Return(member, nil)
	mockStore.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockOrgRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	mockStore.AssertExpectations(t)
	mockOrgRepo.AssertExpectations(t)
}

func TestShareConnectionEndpoint_Unauthorized(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}/share", handler.ShareConnection).Methods("POST")

	reqBody := map[string]string{
		"organization_id": "org-test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/connections/conn-123/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), "user_id", "user-member")
	req = req.WithContext(ctx)

	// Connection owned by different user
	conn := &turso.Connection{
		ID:        "conn-123",
		Name:      "Test DB",
		CreatedBy: "user-other",
	}

	mockStore.On("GetByID", mock.Anything, "conn-123").Return(conn, nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify: 403 Forbidden
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestGetOrgConnectionsEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/organizations/{orgId}/connections", handler.GetOrgConnections).Methods("GET")

	req := httptest.NewRequest("GET", "/api/organizations/org-123/connections", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user-member")
	req = req.WithContext(ctx)

	// Mock data
	member := &organization.OrganizationMember{
		UserID:         "user-member",
		OrganizationID: "org-123",
		Role:           organization.RoleMember,
	}

	connections := []*turso.Connection{
		{
			ID:         "conn-1",
			Name:       "Shared DB 1",
			Visibility: "shared",
		},
		{
			ID:         "conn-2",
			Name:       "Shared DB 2",
			Visibility: "shared",
		},
	}

	mockOrgRepo.On("GetMember", mock.Anything, "org-123", "user-member").Return(member, nil)
	mockStore.On("GetConnectionsByOrganization", mock.Anything, "org-123").Return(connections, nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	conns := response["connections"].([]interface{})
	assert.Len(t, conns, 2)
}

func TestGetOrgConnectionsEndpoint_NotMember(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/organizations/{orgId}/connections", handler.GetOrgConnections).Methods("GET")

	req := httptest.NewRequest("GET", "/api/organizations/org-123/connections", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user-outsider")
	req = req.WithContext(ctx)

	// User is NOT a member
	mockOrgRepo.On("GetMember", mock.Anything, "org-123", "user-outsider").Return(nil, assert.AnError)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify: 403 Forbidden
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestUnshareConnectionEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}/unshare", handler.UnshareConnection).Methods("POST")

	req := httptest.NewRequest("POST", "/api/connections/conn-123/unshare", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user-owner")
	req = req.WithContext(ctx)

	orgID := "org-test"
	conn := &turso.Connection{
		ID:             "conn-123",
		Name:           "Shared DB",
		CreatedBy:      "user-owner",
		Visibility:     "shared",
		OrganizationID: &orgID,
	}

	member := &organization.OrganizationMember{
		UserID:         "user-owner",
		OrganizationID: orgID,
		Role:           organization.RoleAdmin,
	}

	mockStore.On("GetByID", mock.Anything, "conn-123").Return(conn, nil)
	mockOrgRepo.On("GetMember", mock.Anything, orgID, "user-owner").Return(member, nil)
	mockStore.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockOrgRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetAccessibleConnectionsEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections", handler.GetAccessibleConnections).Methods("GET")

	req := httptest.NewRequest("GET", "/api/connections", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user-test")
	req = req.WithContext(ctx)

	connections := []*turso.Connection{
		{ID: "conn-1", Name: "Personal DB", Visibility: "personal"},
		{ID: "conn-2", Name: "Org DB 1", Visibility: "shared"},
		{ID: "conn-3", Name: "Org DB 2", Visibility: "shared"},
	}

	mockStore.On("GetSharedConnections", mock.Anything, "user-test").Return(connections, nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	conns := response["connections"].([]interface{})
	assert.Len(t, conns, 3)
}

func TestUpdateConnectionVisibilityEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}/visibility", handler.UpdateVisibility).Methods("PATCH")

	reqBody := map[string]string{
		"visibility":      "shared",
		"organization_id": "org-123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", "/api/connections/conn-123/visibility", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", "user-owner")
	req = req.WithContext(ctx)

	conn := &turso.Connection{
		ID:        "conn-123",
		Name:      "DB",
		CreatedBy: "user-owner",
	}

	mockStore.On("GetByID", mock.Anything, "conn-123").Return(conn, nil)
	mockStore.On("UpdateConnectionVisibility", mock.Anything, "conn-123", "user-owner", "shared").Return(nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreateConnectionEndpoint_WithOrg(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections", handler.CreateConnection).Methods("POST")

	orgID := "org-create"
	reqBody := map[string]interface{}{
		"name":            "New Shared DB",
		"type":            "postgres",
		"host":            "localhost",
		"port":            5432,
		"database":        "mydb",
		"username":        "user",
		"visibility":      "shared",
		"organization_id": orgID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", "user-admin")
	req = req.WithContext(ctx)

	member := &organization.OrganizationMember{
		UserID:         "user-admin",
		OrganizationID: orgID,
		Role:           organization.RoleAdmin,
	}

	mockOrgRepo.On("GetMember", mock.Anything, orgID, "user-admin").Return(member, nil)
	mockStore.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockOrgRepo.On("CreateAuditLog", mock.Anything, mock.Anything).Return(nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestDeleteConnectionEndpoint(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}", handler.DeleteConnection).Methods("DELETE")

	req := httptest.NewRequest("DELETE", "/api/connections/conn-123", nil)
	ctx := context.WithValue(req.Context(), "user_id", "user-owner")
	req = req.WithContext(ctx)

	conn := &turso.Connection{
		ID:         "conn-123",
		Name:       "To Delete",
		CreatedBy:  "user-owner",
		Visibility: "personal",
	}

	mockStore.On("GetByID", mock.Anything, "conn-123").Return(conn, nil)
	mockStore.On("Delete", mock.Anything, "conn-123").Return(nil)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAPIValidation_MissingOrganizationID(t *testing.T) {
	mockStore := new(MockConnectionStore)
	mockOrgRepo := new(MockOrgRepository)
	service := connections.NewService(mockStore, mockOrgRepo, testLogger())

	router := mux.NewRouter()
	handler := connections.NewHandler(service, testLogger())
	router.HandleFunc("/api/connections/{id}/share", handler.ShareConnection).Methods("POST")

	// Missing organization_id in request
	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/connections/conn-123/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", "user-test")
	req = req.WithContext(ctx)

	// Execute
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify: 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAPIRateLimiting(t *testing.T) {
	// Placeholder for rate limiting test
	// Would test max requests per user/org
	t.Skip("Rate limiting not yet implemented")
}

// ====================================================================
// Mock Handler (placeholder - actual handler would be in handler.go)
// ====================================================================

// This is a placeholder showing what the actual Handler struct might look like
type Handler struct {
	service *connections.Service
	logger  *logrus.Logger
}

func NewHandler(service *connections.Service, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) ShareConnection(w http.ResponseWriter, r *http.Request) {
	// Implementation would be in actual handler.go
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) GetOrgConnections(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections": []map[string]string{},
	})
}

func (h *Handler) UnshareConnection(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) GetAccessibleConnections(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections": []map[string]string{},
	})
}

func (h *Handler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": "new-conn-id"})
}

func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
