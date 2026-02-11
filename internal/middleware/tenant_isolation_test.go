package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestTenantIsolationMiddleware(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	logger := logrus.New()
	middleware := NewTenantIsolationMiddleware(db, logger)

	tests := []struct {
		name           string
		userID         string
		setupMock      func()
		expectedStatus int
		expectOrgIDs   []string
	}{
		{
			name:   "User with multiple organizations",
			userID: "user-123",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id"}).
					AddRow("org-1").
					AddRow("org-2")
				mock.ExpectQuery("SELECT o.id FROM organizations").
					WithArgs("user-123").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectOrgIDs:   []string{"org-1", "org-2"},
		},
		{
			name:   "User with single organization",
			userID: "user-456",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id"}).
					AddRow("org-3")
				mock.ExpectQuery("SELECT o.id FROM organizations").
					WithArgs("user-456").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectOrgIDs:   []string{"org-3"},
		},
		{
			name:   "User with no organizations",
			userID: "user-789",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"id"})
				mock.ExpectQuery("SELECT o.id FROM organizations").
					WithArgs("user-789").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "No user ID in context",
			userID:         "",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			// Create test handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectOrgIDs != nil {
					orgIDs := GetUserOrganizationIDs(r.Context())
					assert.Equal(t, tt.expectOrgIDs, orgIDs)

					currentOrgID := GetCurrentOrgID(r.Context())
					assert.Equal(t, tt.expectOrgIDs[0], currentOrgID)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create request with user context
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			// Record response
			rr := httptest.NewRecorder()

			// Execute middleware
			handler := middleware.EnforceTenantIsolation(nextHandler)
			handler.ServeHTTP(rr, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBuildOrgFilterQuery(t *testing.T) {
	tests := []struct {
		name         string
		orgIDs       []string
		columnName   string
		expectedSQL  string
		expectedArgs int
	}{
		{
			name:         "Single organization",
			orgIDs:       []string{"org-1"},
			columnName:   "organization_id",
			expectedSQL:  "organization_id = ?",
			expectedArgs: 1,
		},
		{
			name:         "Multiple organizations",
			orgIDs:       []string{"org-1", "org-2", "org-3"},
			columnName:   "org_id",
			expectedSQL:  "org_id IN (?,?,?)",
			expectedArgs: 3,
		},
		{
			name:         "No organizations",
			orgIDs:       []string{},
			columnName:   "organization_id",
			expectedSQL:  "organization_id = ?",
			expectedArgs: 1, // __no_access__
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), UserOrganizationsKey, tt.orgIDs)

			query, args := BuildOrgFilterQuery(ctx, tt.columnName)

			assert.Contains(t, query, tt.columnName)
			assert.Equal(t, tt.expectedArgs, len(args))
		})
	}
}

func TestVerifyOrgAccess(t *testing.T) {
	tests := []struct {
		name        string
		userOrgs    []string
		targetOrg   string
		expectError bool
	}{
		{
			name:        "User has access",
			userOrgs:    []string{"org-1", "org-2"},
			targetOrg:   "org-1",
			expectError: false,
		},
		{
			name:        "User does not have access",
			userOrgs:    []string{"org-1", "org-2"},
			targetOrg:   "org-3",
			expectError: true,
		},
		{
			name:        "No organizations",
			userOrgs:    []string{},
			targetOrg:   "org-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), UserOrganizationsKey, tt.userOrgs)

			err := VerifyOrgAccess(ctx, tt.targetOrg)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test for data isolation - ensure queries are properly filtered
func TestDataIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }() // Best-effort close in test

	ctx := context.WithValue(context.Background(), UserOrganizationsKey, []string{"org-1"})

	// Example query that should be filtered by organization
	_ = `SELECT * FROM connections WHERE organization_id = ?` // Example for documentation

	rows := sqlmock.NewRows([]string{"id", "name", "organization_id"}).
		AddRow("conn-1", "My Connection", "org-1")

	mock.ExpectQuery("SELECT .* FROM connections WHERE organization_id").
		WithArgs("org-1").
		WillReturnRows(rows)

	// Execute query with organization filter
	whereClause, args := BuildOrgFilterQuery(ctx, "organization_id")
	// #nosec G202 - test query with parameterized whereClause, safe string concatenation
	fullQuery := "SELECT * FROM connections WHERE " + whereClause

	result, err := db.QueryContext(ctx, fullQuery, args...)
	assert.NoError(t, err)
	defer func() { _ = result.Close() }() // Best-effort close in test

	// Verify data is returned
	assert.True(t, result.Next())

	var id, name, orgID string
	err = result.Scan(&id, &name, &orgID)
	assert.NoError(t, err)
	assert.Equal(t, "org-1", orgID)

	// Verify expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Benchmark tenant isolation middleware
func BenchmarkTenantIsolation(b *testing.B) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }() // Best-effort close in test

	logger := logrus.New()
	middleware := NewTenantIsolationMiddleware(db, logger)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("org-1")
	mock.ExpectQuery("SELECT o.id FROM organizations").
		WillReturnRows(rows).
		RowsWillBeClosed()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.EnforceTenantIsolation(nextHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), UserIDKey, "user-123")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
