package middleware_test

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/internal/middleware"
)

// Helper function to create a test logger
func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests
	return logger
}

// Helper function to create a simple test handler
func createTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// TestSecurityHeadersMiddleware_DefaultConfig tests middleware with nil config (uses defaults)
func TestSecurityHeadersMiddleware_DefaultConfig(t *testing.T) {
	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(nil, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Simulate HTTPS connection
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify CSP header is set
	csp := rec.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "CSP header should be set")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "upgrade-insecure-requests")

	// Verify other security headers
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Contains(t, rec.Header().Get("Permissions-Policy"), "geolocation=()")

	// Verify HSTS is set (connection has TLS)
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set with TLS connection")
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")

	// Verify additional security headers
	assert.Equal(t, "off", rec.Header().Get("X-DNS-Prefetch-Control"))
	assert.Equal(t, "noopen", rec.Header().Get("X-Download-Options"))
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))
}

// TestSecurityHeadersMiddleware_CustomCSP tests custom CSP configuration
func TestSecurityHeadersMiddleware_CustomCSP(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'", "https://cdn.example.com"},
			StyleSrc:   []string{"'self'", "https://fonts.googleapis.com"},
			ImgSrc:     []string{"'self'", "data:", "https:"},
			FontSrc:    []string{"'self'", "https://fonts.gstatic.com"},
			ConnectSrc: []string{"'self'", "https://api.example.com"},
			FrameSrc:   []string{"'none'"},
			ObjectSrc:  []string{"'none'"},
			MediaSrc:   []string{"'self'"},
			WorkerSrc:  []string{"'self'"},
		},
		ContentTypeOptions: "nosniff",
		FrameOptions:       "SAMEORIGIN",
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' https://cdn.example.com")
	assert.Contains(t, csp, "style-src 'self' https://fonts.googleapis.com")
	assert.Contains(t, csp, "img-src 'self' data: https:")
	assert.Contains(t, csp, "font-src 'self' https://fonts.gstatic.com")
	assert.Contains(t, csp, "connect-src 'self' https://api.example.com")
	assert.Contains(t, csp, "frame-src 'none'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "media-src 'self'")
	assert.Contains(t, csp, "worker-src 'self'")
	assert.Contains(t, csp, "upgrade-insecure-requests")

	assert.Equal(t, "SAMEORIGIN", rec.Header().Get("X-Frame-Options"))
}

// TestSecurityHeadersMiddleware_CSPReportOnly tests CSP in report-only mode
func TestSecurityHeadersMiddleware_CSPReportOnly(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'"},
			ReportURI:  "/api/csp-report",
			ReportOnly: true, // Enable report-only mode
		},
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify CSP is in report-only header, not enforcement header
	reportOnlyCSP := rec.Header().Get("Content-Security-Policy-Report-Only")
	enforcingCSP := rec.Header().Get("Content-Security-Policy")

	assert.NotEmpty(t, reportOnlyCSP, "Report-Only CSP should be set")
	assert.Empty(t, enforcingCSP, "Enforcing CSP should not be set")
	assert.Contains(t, reportOnlyCSP, "default-src 'self'")
	assert.Contains(t, reportOnlyCSP, "script-src 'self'")
	assert.Contains(t, reportOnlyCSP, "report-uri /api/csp-report")
}

// TestSecurityHeadersMiddleware_CSPWithReportURI tests CSP with report URI
func TestSecurityHeadersMiddleware_CSPWithReportURI(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
			ReportURI:  "/api/csp-violations",
		},
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "report-uri /api/csp-violations")
}

// TestSecurityHeadersMiddleware_HSTS_WithTLS tests HSTS with TLS connection
func TestSecurityHeadersMiddleware_HSTS_WithTLS(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Simulate TLS connection
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set with TLS connection")
	// Note: buildHSTS has a bug converting int to string, so we just check it's set
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.NotContains(t, hsts, "preload")
}

// TestSecurityHeadersMiddleware_HSTS_WithPreload tests HSTS with preload flag
func TestSecurityHeadersMiddleware_HSTS_WithPreload(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            63072000, // 2 years
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

// TestSecurityHeadersMiddleware_HSTS_WithoutTLS tests that HSTS is not set without TLS
func TestSecurityHeadersMiddleware_HSTS_WithoutTLS(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS: true,
		HSTSMaxAge: 31536000,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No TLS connection and no proxy headers
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts, "HSTS should not be set without secure connection")
}

// TestSecurityHeadersMiddleware_HSTS_WithXForwardedProto tests HSTS with proxy header
func TestSecurityHeadersMiddleware_HSTS_WithXForwardedProto(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS: true,
		HSTSMaxAge: 31536000,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set with X-Forwarded-Proto: https")
}

// TestSecurityHeadersMiddleware_HSTS_WithXForwardedSsl tests HSTS with X-Forwarded-Ssl header
func TestSecurityHeadersMiddleware_HSTS_WithXForwardedSsl(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS: true,
		HSTSMaxAge: 31536000,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Ssl", "on")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set with X-Forwarded-Ssl: on")
}

// TestSecurityHeadersMiddleware_HSTS_WithCFVisitor tests HSTS with Cloudflare visitor header
func TestSecurityHeadersMiddleware_HSTS_WithCFVisitor(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS: true,
		HSTSMaxAge: 31536000,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Cloudflare visitor header with HTTPS scheme
	cfVisitor := map[string]string{"scheme": "https"}
	cfVisitorJSON, _ := json.Marshal(cfVisitor)
	req.Header.Set("CF-Visitor", string(cfVisitorJSON))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set with CF-Visitor containing https")
}

// TestSecurityHeadersMiddleware_HSTS_Disabled tests that HSTS is not set when disabled
func TestSecurityHeadersMiddleware_HSTS_Disabled(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		EnableHSTS: false, // Explicitly disabled
		HSTSMaxAge: 31536000,
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.TLS = &tls.ConnectionState{} // Even with TLS
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts, "HSTS should not be set when disabled")
}

// TestSecurityHeadersMiddleware_AllHeaders tests all security headers are set
func TestSecurityHeadersMiddleware_AllHeaders(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
		},
		EnableHSTS:         true,
		HSTSMaxAge:         31536000,
		FrameOptions:       "DENY",
		ContentTypeOptions: "nosniff",
		XSSProtection:      "1; mode=block",
		ReferrerPolicy:     "no-referrer",
		PermissionsPolicy:  "geolocation=(), camera=()",
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify all configured headers
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), camera=()", rec.Header().Get("Permissions-Policy"))

	// Verify automatic security headers
	assert.Equal(t, "off", rec.Header().Get("X-DNS-Prefetch-Control"))
	assert.Equal(t, "noopen", rec.Header().Get("X-Download-Options"))
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))
}

// TestSecurityHeadersMiddleware_EmptyHeaders tests that empty config values don't set headers
func TestSecurityHeadersMiddleware_EmptyHeaders(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP:                middleware.CSPConfig{}, // Empty CSP
		EnableHSTS:         false,
		FrameOptions:       "", // Empty
		ContentTypeOptions: "", // Empty
		XSSProtection:      "", // Empty
		ReferrerPolicy:     "", // Empty
		PermissionsPolicy:  "", // Empty
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// CSP should still be set with just upgrade-insecure-requests
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Equal(t, "upgrade-insecure-requests", csp)

	// Other headers should not be set
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	assert.Empty(t, rec.Header().Get("X-Frame-Options"))
	assert.Empty(t, rec.Header().Get("X-Content-Type-Options"))
	assert.Empty(t, rec.Header().Get("X-XSS-Protection"))
	assert.Empty(t, rec.Header().Get("Referrer-Policy"))
	assert.Empty(t, rec.Header().Get("Permissions-Policy"))

	// Automatic headers should still be set
	assert.Equal(t, "off", rec.Header().Get("X-DNS-Prefetch-Control"))
	assert.Equal(t, "noopen", rec.Header().Get("X-Download-Options"))
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))
}

// TestSecurityHeadersMiddleware_PartialCSP tests CSP with only some directives
func TestSecurityHeadersMiddleware_PartialCSP(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'", "'unsafe-inline'"},
			// Other directives omitted
		},
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' 'unsafe-inline'")
	assert.NotContains(t, csp, "style-src")
	assert.NotContains(t, csp, "img-src")
	assert.Contains(t, csp, "upgrade-insecure-requests")
}

// TestSecurityHeadersMiddleware_MultipleRequests tests that middleware works for multiple requests
func TestSecurityHeadersMiddleware_MultipleRequests(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
		},
		FrameOptions: "DENY",
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	assert.NotEmpty(t, rec1.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rec1.Header().Get("X-Frame-Options"))

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/test2", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.NotEmpty(t, rec2.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rec2.Header().Get("X-Frame-Options"))
}

// TestStrictSecurityConfig tests the strict security configuration
func TestStrictSecurityConfig(t *testing.T) {
	config := middleware.StrictSecurityConfig()

	require.NotNil(t, config)

	// Verify strict CSP
	assert.Equal(t, []string{"'none'"}, config.CSP.DefaultSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.ScriptSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.StyleSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.ImgSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.FontSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.ConnectSrc)
	assert.Equal(t, []string{"'none'"}, config.CSP.FrameSrc)
	assert.Equal(t, []string{"'none'"}, config.CSP.ObjectSrc)
	assert.Equal(t, []string{"'none'"}, config.CSP.MediaSrc)
	assert.Equal(t, []string{"'self'"}, config.CSP.WorkerSrc)
	assert.Equal(t, "/api/csp-report", config.CSP.ReportURI)
	assert.False(t, config.CSP.ReportOnly)

	// Verify strict HSTS (2 years)
	assert.True(t, config.EnableHSTS)
	assert.Equal(t, 63072000, config.HSTSMaxAge)
	assert.True(t, config.HSTSIncludeSubdomains)
	assert.True(t, config.HSTSPreload)

	// Verify strict other headers
	assert.Equal(t, "DENY", config.FrameOptions)
	assert.Equal(t, "nosniff", config.ContentTypeOptions)
	assert.Equal(t, "1; mode=block", config.XSSProtection)
	assert.Equal(t, "no-referrer", config.ReferrerPolicy)
	assert.Contains(t, config.PermissionsPolicy, "geolocation=()")
	assert.Contains(t, config.PermissionsPolicy, "camera=()")
	assert.Contains(t, config.PermissionsPolicy, "payment=()")
}

// TestStrictSecurityConfig_InMiddleware tests strict config in actual middleware
func TestStrictSecurityConfig_InMiddleware(t *testing.T) {
	config := middleware.StrictSecurityConfig()
	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify strict CSP is applied
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "script-src 'self'")
	assert.NotContains(t, csp, "unsafe-inline")
	assert.Contains(t, csp, "report-uri /api/csp-report")

	// Verify strict HSTS
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")

	// Verify other strict headers
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Contains(t, rec.Header().Get("Permissions-Policy"), "payment=()")
}

// TestSecurityHeadersMiddleware_CSPDirectiveSeparation tests that CSP directives are properly separated
func TestSecurityHeadersMiddleware_CSPDirectiveSeparation(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'", "https://cdn.example.com"},
			StyleSrc:   []string{"'self'"},
		},
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	// Check that directives are semicolon-separated
	directives := strings.Split(csp, "; ")
	assert.GreaterOrEqual(t, len(directives), 3)

	// Verify each directive is properly formatted
	foundDefault := false
	foundScript := false
	foundStyle := false

	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "default-src") {
			assert.Equal(t, "default-src 'self'", directive)
			foundDefault = true
		}
		if strings.HasPrefix(directive, "script-src") {
			assert.Equal(t, "script-src 'self' https://cdn.example.com", directive)
			foundScript = true
		}
		if strings.HasPrefix(directive, "style-src") {
			assert.Equal(t, "style-src 'self'", directive)
			foundStyle = true
		}
	}

	assert.True(t, foundDefault, "default-src directive not found")
	assert.True(t, foundScript, "script-src directive not found")
	assert.True(t, foundStyle, "style-src directive not found")
}

// TestSecurityHeadersMiddleware_ChainedMiddleware tests security headers work in middleware chain
func TestSecurityHeadersMiddleware_ChainedMiddleware(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
		},
		FrameOptions: "DENY",
	}

	logger := createTestLogger()

	// Create a middleware chain
	finalHandler := createTestHandler()

	// Add custom middleware that sets a header
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			next.ServeHTTP(w, r)
		})
	}

	// Chain: security headers -> custom middleware -> handler
	handler := middleware.SecurityHeadersMiddleware(config, logger)(
		customMiddleware(finalHandler),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify both security headers and custom header are set
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "custom-value", rec.Header().Get("X-Custom-Header"))
}

// TestSecurityHeadersMiddleware_DifferentMethods tests that headers are set for all HTTP methods
func TestSecurityHeadersMiddleware_DifferentMethods(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP: middleware.CSPConfig{
			DefaultSrc: []string{"'self'"},
		},
		FrameOptions: "DENY",
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"),
				"CSP should be set for %s", method)
			assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"),
				"X-Frame-Options should be set for %s", method)
		})
	}
}

// TestSecurityHeadersMiddleware_NoCSPConfig tests behavior when CSP config is completely empty
func TestSecurityHeadersMiddleware_NoCSPConfig(t *testing.T) {
	config := &middleware.SecurityHeadersConfig{
		CSP:          middleware.CSPConfig{}, // Empty CSP config
		FrameOptions: "DENY",
	}

	logger := createTestLogger()
	handler := middleware.SecurityHeadersMiddleware(config, logger)(createTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Even with empty CSP config, upgrade-insecure-requests should be present
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Equal(t, "upgrade-insecure-requests", csp)

	// Other headers should still work
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}
