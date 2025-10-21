package testutil

import (
\t"net/http"
\t"net/http/httptest"
\t"testing"
)

// MockHTTPClient creates a test HTTP client with a custom handler
func MockHTTPClient(handler http.HandlerFunc) (*http.Client, *httptest.Server) {
\tserver := httptest.NewServer(handler)

\tclient := &http.Client{
\t\tTransport: &http.Transport{
\t\t\tProxy: http.ProxyFromEnvironment,
\t\t},
\t}

\treturn client, server
}

// NewTestRequest creates a test HTTP request
func NewTestRequest(t *testing.T, method, path string, body interface{}) *http.Request {
\tt.Helper()

\treq := httptest.NewRequest(method, path, nil)
\treq.Header.Set("Content-Type", "application/json")

\treturn req
}

// NewTestResponseRecorder creates a test response recorder
func NewTestResponseRecorder() *httptest.ResponseRecorder {
\treturn httptest.NewRecorder()
}
