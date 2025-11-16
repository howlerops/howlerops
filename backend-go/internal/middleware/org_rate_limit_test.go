package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/internal/quotas"
)

func TestOrgRateLimiter_Limit(t *testing.T) {
	tests := []struct {
		name          string
		orgID         string
		setupMock     func(mock sqlmock.Sqlmock)
		expectAllow   bool
		expectStatus  int
		expectHeaders bool
		requestCount  int
	}{
		{
			name:  "No organization context allows request through",
			orgID: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No mock setup needed - no DB calls when orgID is empty
			},
			expectAllow:  true,
			expectStatus: http.StatusOK,
			requestCount: 1,
		},
		{
			name:  "Request within quota and rate limit",
			orgID: "org-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock quota check (GetQuota + GetTodayUsage)
				quotaRows := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-1").
					WillReturnRows(quotaRows)

				usageRows := sqlmock.NewRows([]string{
					"id", "organization_id", "usage_date", "connections_count",
					"queries_count", "storage_used_mb", "api_calls_count",
					"concurrent_queries_peak", "created_at", "updated_at",
				}).AddRow("usage-1", "org-1", time.Now().Unix(), 5, 500, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_usage").
					WithArgs("org-1", sqlmock.AnyArg()).
					WillReturnRows(usageRows)

				// Mock hourly API usage check - within limit (100/1000)
				hourlyUsageRows := sqlmock.NewRows([]string{"COALESCE(SUM(api_calls_count), 0)"}).
					AddRow(100)

				mock.ExpectQuery("SELECT COALESCE\\(SUM\\(api_calls_count\\), 0\\) FROM organization_usage_hourly").
					WithArgs("org-1", sqlmock.AnyArg()).
					WillReturnRows(hourlyUsageRows)

				// Mock GetQuota for rate limiter setup
				quotaRows2 := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-1").
					WillReturnRows(quotaRows2)

				// Mock IncrementUsage (happens async, but we can expect it)
				mock.ExpectExec("INSERT INTO organization_usage").
					WithArgs(
						sqlmock.AnyArg(), // id
						"org-1",          // organization_id
						sqlmock.AnyArg(), // usage_date
						1,                // amount
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						1,                // amount (for update)
						sqlmock.AnyArg(), // updated_at (for update)
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectAllow:   true,
			expectStatus:  http.StatusOK,
			expectHeaders: true,
			requestCount:  1,
		},
		{
			name:  "Request with quota exceeded",
			orgID: "org-2",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock quota check - at limit
				quotaRows := sqlmock.NewRows([]string{
					"organization_id", "max_connections", "max_queries_per_day",
					"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
					"max_team_members", "features_enabled", "created_at", "updated_at",
				}).AddRow("org-2", 10, 1000, 100, 100, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_quotas").
					WithArgs("org-2").
					WillReturnRows(quotaRows)

				usageRows := sqlmock.NewRows([]string{
					"id", "organization_id", "usage_date", "connections_count",
					"queries_count", "storage_used_mb", "api_calls_count",
					"concurrent_queries_peak", "created_at", "updated_at",
				}).AddRow("usage-2", "org-2", time.Now().Unix(), 5, 500, 50.0, 100, 2, time.Now().Unix(), time.Now().Unix())

				mock.ExpectQuery("SELECT .* FROM organization_usage").
					WithArgs("org-2", sqlmock.AnyArg()).
					WillReturnRows(usageRows)

				// Mock hourly API usage check - at limit (100/100)
				hourlyUsageRows := sqlmock.NewRows([]string{"COALESCE(SUM(api_calls_count), 0)"}).
					AddRow(100)

				mock.ExpectQuery("SELECT COALESCE\\(SUM\\(api_calls_count\\), 0\\) FROM organization_usage_hourly").
					WithArgs("org-2", sqlmock.AnyArg()).
					WillReturnRows(hourlyUsageRows)
			},
			expectAllow:   false,
			expectStatus:  http.StatusTooManyRequests,
			expectHeaders: true,
			requestCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DB
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()

			// Setup mock expectations
			tt.setupMock(mock)

			// Create real stores and services
			quotaStore := quotas.NewStore(db)
			quotaService := quotas.NewService(quotaStore, logrus.New())

			// Create rate limiter
			rateLimiter := NewOrgRateLimiter(quotaService, logrus.New())

			// Create test handler
			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Wrap handler with rate limiter
			handler := rateLimiter.Limit(testHandler)

			// Create test request
			req := httptest.NewRequest("GET", "/api/test", nil)

			// Add organization context if specified
			if tt.orgID != "" {
				ctx := context.WithValue(req.Context(), CurrentOrgIDKey, tt.orgID)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rr, req)

			// Wait a bit for async operations
			time.Sleep(50 * time.Millisecond)

			// Assert status code
			assert.Equal(t, tt.expectStatus, rr.Code)

			// Assert handler was called (or not)
			if tt.expectAllow {
				assert.True(t, handlerCalled, "Handler should have been called")
			} else {
				assert.False(t, handlerCalled, "Handler should not have been called")
			}

			// Assert rate limit headers are present
			if tt.expectHeaders {
				assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
				assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
			}

			// Verify all mock expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestOrgRateLimiter_GetLimiter(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	quotaStore := quotas.NewStore(db)
	quotaService := quotas.NewService(quotaStore, logrus.New())

	// Create rate limiter
	rateLimiter := NewOrgRateLimiter(quotaService, logrus.New())

	// Mock GetQuota
	quotaRows := sqlmock.NewRows([]string{
		"organization_id", "max_connections", "max_queries_per_day",
		"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
		"max_team_members", "features_enabled", "created_at", "updated_at",
	}).AddRow("org-1", 10, 1000, 100, 3600, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

	mock.ExpectQuery("SELECT .* FROM organization_quotas").
		WithArgs("org-1").
		WillReturnRows(quotaRows)

	// Get limiter
	limiter := rateLimiter.getLimiter("org-1")

	// Assert limiter was created
	assert.NotNil(t, limiter)

	// Expected rate: 3600 calls per hour = 1 call per second
	expectedRate := 1.0
	assert.Equal(t, expectedRate, float64(limiter.Limit()))

	// Expected burst: 10% of 3600 = 360
	assert.Equal(t, 360, limiter.Burst())

	// Verify mock expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrgRateLimiter_GetLimiter_CachesLimiter(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	quotaStore := quotas.NewStore(db)
	quotaService := quotas.NewService(quotaStore, logrus.New())

	// Create rate limiter
	rateLimiter := NewOrgRateLimiter(quotaService, logrus.New())

	// Mock GetQuota - should only be called once
	quotaRows := sqlmock.NewRows([]string{
		"organization_id", "max_connections", "max_queries_per_day",
		"max_storage_mb", "max_api_calls_per_hour", "max_concurrent_queries",
		"max_team_members", "features_enabled", "created_at", "updated_at",
	}).AddRow("org-1", 10, 1000, 100, 1000, 5, 5, "basic", time.Now().Unix(), time.Now().Unix())

	mock.ExpectQuery("SELECT .* FROM organization_quotas").
		WithArgs("org-1").
		WillReturnRows(quotaRows)

	// Get limiter first time
	limiter1 := rateLimiter.getLimiter("org-1")
	assert.NotNil(t, limiter1)

	// Get limiter second time - should use cached version
	limiter2 := rateLimiter.getLimiter("org-1")
	assert.NotNil(t, limiter2)

	// Should be the same limiter instance
	assert.Same(t, limiter1, limiter2)

	// Verify mock expectations - should only query once
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrgRateLimiter_DefaultsOnQuotaError(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	quotaStore := quotas.NewStore(db)
	quotaService := quotas.NewService(quotaStore, logrus.New())

	// Create rate limiter
	rateLimiter := NewOrgRateLimiter(quotaService, logrus.New())

	// Mock GetQuota to return error
	mock.ExpectQuery("SELECT .* FROM organization_quotas").
		WithArgs("org-error").
		WillReturnError(assert.AnError)

	// Get limiter
	limiter := rateLimiter.getLimiter("org-error")

	// Assert limiter was created with defaults
	assert.NotNil(t, limiter)

	// Default: 1000 calls per hour = 0.277... calls per second
	expectedRate := float64(1000) / 3600.0
	assert.InDelta(t, expectedRate, float64(limiter.Limit()), 0.001)

	// Default burst: 10% of 1000 = 100
	assert.Equal(t, 100, limiter.Burst())

	// Verify mock expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}
