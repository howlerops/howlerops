package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockHTTPClient creates a test HTTP client with a custom handler
func MockHTTPClient(handler http.HandlerFunc) (*http.Client, *httptest.Server) {
	server := httptest.NewServer(handler)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	return client, server
}

// NewTestRequest creates a test HTTP request
func NewTestRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Content-Type", "application/json")

	return req
}

// NewTestResponseRecorder creates a test response recorder
func NewTestResponseRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}
