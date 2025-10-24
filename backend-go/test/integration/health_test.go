package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HealthTestSuite contains all health-related tests
type HealthTestSuite struct {
	baseURL string
	client  *http.Client
}

// NewHealthTestSuite creates a new health test suite
func NewHealthTestSuite() *HealthTestSuite {
	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8500"
	}

	return &HealthTestSuite{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// MetricsResponse represents metrics endpoint response
type MetricsResponse struct {
	// Prometheus metrics are in text format
	// We'll just check for valid response
}

// TestHealthCheck tests the basic health check endpoint
func TestHealthCheck(t *testing.T) {
	suite := NewHealthTestSuite()

	req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
	require.NoError(t, err)

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Health check should always return 200
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check response format
	var healthResp HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	assert.Equal(t, "healthy", healthResp.Status)
	assert.NotEmpty(t, healthResp.Service)
}

// TestHealthCheckResponseTime tests that health check responds quickly
func TestHealthCheckResponseTime(t *testing.T) {
	suite := NewHealthTestSuite()

	start := time.Now()

	req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
	require.NoError(t, err)

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	duration := time.Since(start)

	// Health check should respond in under 1 second
	assert.Less(t, duration, 1*time.Second, "Health check took too long: %v", duration)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestHealthCheckReliability tests health check reliability under load
func TestHealthCheckReliability(t *testing.T) {
	suite := NewHealthTestSuite()

	// Make 20 concurrent health check requests
	successCount := 0
	results := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
			if err != nil {
				results <- false
				return
			}

			resp, err := suite.client.Do(req)
			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()

			results <- resp.StatusCode == http.StatusOK
		}()
	}

	// Collect results
	for i := 0; i < 20; i++ {
		if <-results {
			successCount++
		}
	}

	// All health checks should succeed
	assert.Equal(t, 20, successCount, "Expected all 20 health checks to succeed")
}

// TestMetricsEndpoint tests the Prometheus metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	suite := NewHealthTestSuite()

	// Try the metrics endpoint (might be on a different port)
	metricsURL := os.Getenv("METRICS_URL")
	if metricsURL == "" {
		metricsURL = "http://localhost:9100/metrics"
	}

	req, err := http.NewRequest("GET", metricsURL, nil)
	require.NoError(t, err)

	resp, err := suite.client.Do(req)
	if err != nil {
		t.Skip("Metrics endpoint not available (might be disabled or on different port)")
		return
	}
	defer resp.Body.Close()

	// Metrics should return 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Content-Type should be text/plain for Prometheus format
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "text/plain")
}

// TestReadinessProbe tests if the service is ready to accept traffic
func TestReadinessProbe(t *testing.T) {
	suite := NewHealthTestSuite()

	// Many services have a separate readiness endpoint
	endpoints := []string{
		"/health",
		"/ready",
		"/readiness",
	}

	foundReady := false
	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", suite.baseURL+endpoint, nil)
		require.NoError(t, err)

		resp, err := suite.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			foundReady = true
			t.Logf("Found readiness endpoint at %s", endpoint)
			break
		}
	}

	assert.True(t, foundReady, "Should have at least one readiness endpoint")
}

// TestLivenessProbe tests if the service is alive
func TestLivenessProbe(t *testing.T) {
	suite := NewHealthTestSuite()

	// Many services have a separate liveness endpoint
	endpoints := []string{
		"/health",
		"/live",
		"/liveness",
	}

	foundLive := false
	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", suite.baseURL+endpoint, nil)
		require.NoError(t, err)

		resp, err := suite.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			foundLive = true
			t.Logf("Found liveness endpoint at %s", endpoint)
			break
		}
	}

	assert.True(t, foundLive, "Should have at least one liveness endpoint")
}

// TestCORS tests CORS headers on health endpoint
func TestCORS(t *testing.T) {
	suite := NewHealthTestSuite()

	req, err := http.NewRequest("OPTIONS", suite.baseURL+"/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 OK for OPTIONS
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check CORS headers
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, allowOrigin, "Should have Access-Control-Allow-Origin header")
}

// TestServiceDiscovery tests that all expected endpoints are available
func TestServiceDiscovery(t *testing.T) {
	suite := NewHealthTestSuite()

	// Test endpoints that should be available
	endpoints := map[string]int{
		"/health":            http.StatusOK,
		"/api/auth/login":    http.StatusBadRequest, // Should return 400 for empty body
		"/api/sync/download": http.StatusUnauthorized, // Should return 401 without auth
	}

	for endpoint, expectedStatus := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			method := "GET"
			if endpoint == "/api/auth/login" {
				method = "POST"
			}

			req, err := http.NewRequest(method, suite.baseURL+endpoint, nil)
			require.NoError(t, err)

			resp, err := suite.client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, expectedStatus, resp.StatusCode,
				"Endpoint %s returned unexpected status", endpoint)
		})
	}
}

// TestServerHeaders tests security headers
func TestServerHeaders(t *testing.T) {
	suite := NewHealthTestSuite()

	req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
	require.NoError(t, err)

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check for security headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	for header, expectedValue := range headers {
		value := resp.Header.Get(header)
		if expectedValue != "" {
			assert.Contains(t, value, expectedValue,
				"Header %s should contain %s", header, expectedValue)
		}
	}

	// Server header should not reveal version information
	serverHeader := resp.Header.Get("Server")
	if serverHeader != "" {
		assert.NotContains(t, serverHeader, "Go",
			"Server header should not reveal technology stack")
	}
}

// TestHealthCheckUnderLoad tests health check under sustained load
func TestHealthCheckUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	suite := NewHealthTestSuite()

	// Run health checks for 10 seconds
	duration := 10 * time.Second
	deadline := time.Now().Add(duration)

	successCount := 0
	failureCount := 0

	for time.Now().Before(deadline) {
		req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
		if err != nil {
			failureCount++
			continue
		}

		resp, err := suite.client.Do(req)
		if err != nil {
			failureCount++
			continue
		}

		if resp.StatusCode == http.StatusOK {
			successCount++
		} else {
			failureCount++
		}
		resp.Body.Close()

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Health check results: %d successes, %d failures", successCount, failureCount)

	// At least 95% of health checks should succeed
	successRate := float64(successCount) / float64(successCount+failureCount)
	assert.GreaterOrEqual(t, successRate, 0.95,
		"Health check success rate should be at least 95%%")
}
