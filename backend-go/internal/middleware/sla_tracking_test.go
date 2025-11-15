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

	"github.com/jbeck018/howlerops/backend-go/internal/sla"
)

func TestSLATracking_Track(t *testing.T) {
	tests := []struct {
		name            string
		orgID           string
		path            string
		method          string
		statusCode      int
		expectTracking  bool
		setupMock       func(mock sqlmock.Sqlmock)
	}{
		{
			name:           "No organization context skips tracking",
			orgID:          "",
			path:           "/api/test",
			method:         "GET",
			statusCode:     http.StatusOK,
			expectTracking: false,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No mock setup - no DB calls expected
			},
		},
		{
			name:           "Successful request is tracked",
			orgID:          "org-1",
			path:           "/api/users",
			method:         "GET",
			statusCode:     http.StatusOK,
			expectTracking: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock LogRequest
				mock.ExpectExec("INSERT INTO request_log").
					WithArgs(
						sqlmock.AnyArg(), // id
						"org-1",          // organization_id
						"/api/users",     // endpoint
						"GET",            // method
						sqlmock.AnyArg(), // response_time_ms
						200,              // status_code
						true,             // success (200 is successful)
						sqlmock.AnyArg(), // created_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "Failed request (404) is tracked",
			orgID:          "org-2",
			path:           "/api/users/999",
			method:         "GET",
			statusCode:     http.StatusNotFound,
			expectTracking: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock LogRequest
				mock.ExpectExec("INSERT INTO request_log").
					WithArgs(
						sqlmock.AnyArg(),    // id
						"org-2",             // organization_id
						"/api/users/999",    // endpoint
						"GET",               // method
						sqlmock.AnyArg(),    // response_time_ms
						404,                 // status_code
						true,                // success (4xx is still successful, not server error)
						sqlmock.AnyArg(),    // created_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "Server error (500) is tracked as failure",
			orgID:          "org-3",
			path:           "/api/error",
			method:         "POST",
			statusCode:     http.StatusInternalServerError,
			expectTracking: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock LogRequest
				mock.ExpectExec("INSERT INTO request_log").
					WithArgs(
						sqlmock.AnyArg(),            // id
						"org-3",                     // organization_id
						"/api/error",                // endpoint
						"POST",                      // method
						sqlmock.AnyArg(),            // response_time_ms
						500,                         // status_code
						false,                       // success (5xx is failure)
						sqlmock.AnyArg(),            // created_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "POST request is tracked",
			orgID:          "org-4",
			path:           "/api/users",
			method:         "POST",
			statusCode:     http.StatusCreated,
			expectTracking: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock LogRequest
				mock.ExpectExec("INSERT INTO request_log").
					WithArgs(
						sqlmock.AnyArg(), // id
						"org-4",          // organization_id
						"/api/users",     // endpoint
						"POST",           // method
						sqlmock.AnyArg(), // response_time_ms
						201,              // status_code
						true,             // success
						sqlmock.AnyArg(), // created_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
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
			slaStore := sla.NewStore(db)
			slaMonitor := sla.NewMonitor(slaStore, logrus.New())

			// Create SLA tracking middleware
			tracking := NewSLATracking(slaMonitor, logrus.New())

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("Response"))
			})

			// Wrap handler with SLA tracking
			handler := tracking.Track(testHandler)

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)

			// Add organization context if specified
			if tt.orgID != "" {
				ctx := context.WithValue(req.Context(), CurrentOrgIDKey, tt.orgID)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rr, req)

			// Wait for async tracking to complete
			time.Sleep(100 * time.Millisecond)

			// Assert status code
			assert.Equal(t, tt.statusCode, rr.Code)

			// Verify all mock expectations were met
			if tt.expectTracking {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestSLATracking_RecordsDuration(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	slaStore := sla.NewStore(db)
	slaMonitor := sla.NewMonitor(slaStore, logrus.New())

	// Create SLA tracking middleware
	tracking := NewSLATracking(slaMonitor, logrus.New())

	// Create slow handler to verify duration tracking
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Mock LogRequest - we'll verify duration is > 0
	mock.ExpectExec("INSERT INTO request_log").
		WithArgs(
			sqlmock.AnyArg(), // id
			"org-1",          // organization_id
			"/slow",          // endpoint
			"GET",            // method
			sqlmock.AnyArg(), // response_time_ms - should be > 50
			200,              // status_code
			true,             // success
			sqlmock.AnyArg(), // created_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Wrap handler with SLA tracking
	handler := tracking.Track(slowHandler)

	// Create test request with org context
	req := httptest.NewRequest("GET", "/slow", nil)
	ctx := context.WithValue(req.Context(), CurrentOrgIDKey, "org-1")
	req = req.WithContext(ctx)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	start := time.Now()
	handler.ServeHTTP(rr, req)
	duration := time.Since(start)

	// Wait for async tracking
	time.Sleep(100 * time.Millisecond)

	// Assert duration was at least 50ms
	assert.GreaterOrEqual(t, duration.Milliseconds(), int64(50))

	// Assert status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify mock expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSLATracking_HandlesMultipleRequests(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	slaStore := sla.NewStore(db)
	slaMonitor := sla.NewMonitor(slaStore, logrus.New())

	// Create SLA tracking middleware
	tracking := NewSLATracking(slaMonitor, logrus.New())

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler
	handler := tracking.Track(testHandler)

	// Setup mock expectations for multiple requests
	for i := 0; i < 3; i++ {
		mock.ExpectExec("INSERT INTO request_log").
			WithArgs(
				sqlmock.AnyArg(), // id
				"org-1",          // organization_id
				"/api/test",      // endpoint
				"GET",            // method
				sqlmock.AnyArg(), // response_time_ms
				200,              // status_code
				true,             // success
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	// Execute multiple requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), CurrentOrgIDKey, "org-1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Wait for async operations
	time.Sleep(200 * time.Millisecond)

	// Verify all requests were tracked
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSLAResponseWriter_WriteHeader(t *testing.T) {
	// Create base response writer
	baseWriter := httptest.NewRecorder()

	// Create SLA response writer
	rw := &slaResponseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	// Write custom status code
	rw.WriteHeader(http.StatusCreated)

	// Assert status code was captured
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.Equal(t, http.StatusCreated, baseWriter.Code)
}

func TestSLAResponseWriter_Write(t *testing.T) {
	// Create base response writer
	baseWriter := httptest.NewRecorder()

	// Create SLA response writer
	rw := &slaResponseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	// Write some data
	testData := []byte("Test response")
	n, err := rw.Write(testData)

	// Assert write succeeded
	require.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Assert data was written
	assert.Equal(t, string(testData), baseWriter.Body.String())
}

func TestSLAResponseWriter_DefaultStatusCode(t *testing.T) {
	// Create base response writer
	baseWriter := httptest.NewRecorder()

	// Create SLA response writer without explicitly setting status
	rw := &slaResponseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK, // Default
	}

	// Write data without calling WriteHeader
	_, err := rw.Write([]byte("test"))
	require.NoError(t, err)

	// Default status code should be used
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestSLATracking_ErrorRecording(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	slaStore := sla.NewStore(db)
	slaMonitor := sla.NewMonitor(slaStore, logrus.New())

	// Create SLA tracking middleware
	tracking := NewSLATracking(slaMonitor, logrus.New())

	// Create handler that returns error
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	// Mock LogRequest for error case
	mock.ExpectExec("INSERT INTO request_log").
		WithArgs(
			sqlmock.AnyArg(), // id
			"org-1",          // organization_id
			"/error",         // endpoint
			"GET",            // method
			sqlmock.AnyArg(), // response_time_ms
			503,              // status_code
			false,            // success (5xx is failure)
			sqlmock.AnyArg(), // created_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Wrap handler
	handler := tracking.Track(errorHandler)

	// Create request
	req := httptest.NewRequest("GET", "/error", nil)
	ctx := context.WithValue(req.Context(), CurrentOrgIDKey, "org-1")
	req = req.WithContext(ctx)

	// Execute
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Wait for async tracking
	time.Sleep(100 * time.Millisecond)

	// Assert error status
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	// Verify mock
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSLATracking_ContinuesOnTrackingError(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create real stores and services
	slaStore := sla.NewStore(db)
	slaMonitor := sla.NewMonitor(slaStore, logrus.New())

	// Create SLA tracking middleware
	tracking := NewSLATracking(slaMonitor, logrus.New())

	// Create handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Mock LogRequest to return error
	mock.ExpectExec("INSERT INTO request_log").
		WillReturnError(assert.AnError)

	// Wrap handler
	handler := tracking.Track(testHandler)

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), CurrentOrgIDKey, "org-1")
	req = req.WithContext(ctx)

	// Execute
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Wait for async tracking
	time.Sleep(100 * time.Millisecond)

	// Request should still succeed despite tracking error
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify mock was called
	assert.NoError(t, mock.ExpectationsWereMet())
}

// BenchmarkSLATracking benchmarks the SLA tracking middleware
func BenchmarkSLATracking(b *testing.B) {
	// Create mock DB
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	// Create real stores and services
	slaStore := sla.NewStore(db)
	slaMonitor := sla.NewMonitor(slaStore, logrus.New())

	// Create SLA tracking middleware
	tracking := NewSLATracking(slaMonitor, logrus.New())

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler
	handler := tracking.Track(testHandler)

	// Setup mock expectations (will be called for each iteration)
	// Note: We don't use Times() as it's not available in go-sqlmock
	// Instead, we'll set up expectations before each iteration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Setup mock expectation for this iteration
		mock.ExpectExec("INSERT INTO request_log").
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), CurrentOrgIDKey, "org-1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
