package middleware

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// SecurityHeadersConfig contains configuration for security headers
type SecurityHeadersConfig struct {
	// Content Security Policy directives
	CSP CSPConfig `yaml:"csp"`

	// Enable HSTS
	EnableHSTS bool `yaml:"enable_hsts"`

	// HSTS max age in seconds
	HSTSMaxAge int `yaml:"hsts_max_age"`

	// Include subdomains in HSTS
	HSTSIncludeSubdomains bool `yaml:"hsts_include_subdomains"`

	// Enable HSTS preload
	HSTSPreload bool `yaml:"hsts_preload"`

	// Frame options (DENY, SAMEORIGIN, ALLOW-FROM)
	FrameOptions string `yaml:"frame_options"`

	// Content type options
	ContentTypeOptions string `yaml:"content_type_options"`

	// XSS Protection
	XSSProtection string `yaml:"xss_protection"`

	// Referrer Policy
	ReferrerPolicy string `yaml:"referrer_policy"`

	// Permissions Policy
	PermissionsPolicy string `yaml:"permissions_policy"`
}

// CSPConfig contains Content Security Policy configuration
type CSPConfig struct {
	DefaultSrc  []string `yaml:"default_src"`
	ScriptSrc   []string `yaml:"script_src"`
	StyleSrc    []string `yaml:"style_src"`
	ImgSrc      []string `yaml:"img_src"`
	FontSrc     []string `yaml:"font_src"`
	ConnectSrc  []string `yaml:"connect_src"`
	FrameSrc    []string `yaml:"frame_src"`
	ObjectSrc   []string `yaml:"object_src"`
	MediaSrc    []string `yaml:"media_src"`
	WorkerSrc   []string `yaml:"worker_src"`
	ReportURI   string   `yaml:"report_uri"`
	ReportOnly  bool     `yaml:"report_only"`
}

// SecurityHeadersMiddleware adds security headers to HTTP responses
func SecurityHeadersMiddleware(config *SecurityHeadersConfig, logger *logrus.Logger) func(http.Handler) http.Handler {
	// Set defaults if config is nil
	if config == nil {
		config = getDefaultSecurityConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content Security Policy
			csp := buildCSP(config.CSP)
			if csp != "" {
				if config.CSP.ReportOnly {
					w.Header().Set("Content-Security-Policy-Report-Only", csp)
				} else {
					w.Header().Set("Content-Security-Policy", csp)
				}
			}

			// X-Content-Type-Options
			if config.ContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", config.ContentTypeOptions)
			}

			// X-Frame-Options
			if config.FrameOptions != "" {
				w.Header().Set("X-Frame-Options", config.FrameOptions)
			}

			// X-XSS-Protection
			if config.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", config.XSSProtection)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Strict-Transport-Security (HSTS)
			if config.EnableHSTS && (r.TLS != nil || isSecureConnection(r)) {
				hstsValue := buildHSTS(config)
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// Additional security headers
			w.Header().Set("X-DNS-Prefetch-Control", "off")
			w.Header().Set("X-Download-Options", "noopen")
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

			next.ServeHTTP(w, r)
		})
	}
}

// buildCSP builds a Content Security Policy string from configuration
func buildCSP(config CSPConfig) string {
	var directives []string

	// Add each directive if configured
	if len(config.DefaultSrc) > 0 {
		directives = append(directives, "default-src "+strings.Join(config.DefaultSrc, " "))
	}

	if len(config.ScriptSrc) > 0 {
		directives = append(directives, "script-src "+strings.Join(config.ScriptSrc, " "))
	}

	if len(config.StyleSrc) > 0 {
		directives = append(directives, "style-src "+strings.Join(config.StyleSrc, " "))
	}

	if len(config.ImgSrc) > 0 {
		directives = append(directives, "img-src "+strings.Join(config.ImgSrc, " "))
	}

	if len(config.FontSrc) > 0 {
		directives = append(directives, "font-src "+strings.Join(config.FontSrc, " "))
	}

	if len(config.ConnectSrc) > 0 {
		directives = append(directives, "connect-src "+strings.Join(config.ConnectSrc, " "))
	}

	if len(config.FrameSrc) > 0 {
		directives = append(directives, "frame-src "+strings.Join(config.FrameSrc, " "))
	}

	if len(config.ObjectSrc) > 0 {
		directives = append(directives, "object-src "+strings.Join(config.ObjectSrc, " "))
	}

	if len(config.MediaSrc) > 0 {
		directives = append(directives, "media-src "+strings.Join(config.MediaSrc, " "))
	}

	if len(config.WorkerSrc) > 0 {
		directives = append(directives, "worker-src "+strings.Join(config.WorkerSrc, " "))
	}

	// Add report-uri if configured
	if config.ReportURI != "" {
		directives = append(directives, "report-uri "+config.ReportURI)
	}

	// Add upgrade-insecure-requests
	directives = append(directives, "upgrade-insecure-requests")

	return strings.Join(directives, "; ")
}

// buildHSTS builds an HSTS header value from configuration
func buildHSTS(config *SecurityHeadersConfig) string {
	value := "max-age=" + string(rune(config.HSTSMaxAge))

	if config.HSTSIncludeSubdomains {
		value += "; includeSubDomains"
	}

	if config.HSTSPreload {
		value += "; preload"
	}

	return value
}

// isSecureConnection checks if the connection is secure (HTTPS)
func isSecureConnection(r *http.Request) bool {
	// Check common headers set by proxies
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}

	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return true
	}

	// Cloudflare
	if r.Header.Get("CF-Visitor") != "" && strings.Contains(r.Header.Get("CF-Visitor"), "https") {
		return true
	}

	return false
}

// getDefaultSecurityConfig returns default security headers configuration
func getDefaultSecurityConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		CSP: CSPConfig{
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"}, // May need adjustment for production
			StyleSrc:   []string{"'self'", "'unsafe-inline'"},
			ImgSrc:     []string{"'self'", "data:", "https:"},
			FontSrc:    []string{"'self'", "data:"},
			ConnectSrc: []string{"'self'", "https://api.turso.tech"},
			FrameSrc:   []string{"'none'"},
			ObjectSrc:  []string{"'none'"},
			MediaSrc:   []string{"'none'"},
			WorkerSrc:  []string{"'self'"},
		},
		EnableHSTS:            true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,
		FrameOptions:          "DENY",
		ContentTypeOptions:    "nosniff",
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
	}
}

// StrictSecurityConfig returns a strict security configuration for production
func StrictSecurityConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		CSP: CSPConfig{
			DefaultSrc: []string{"'none'"},
			ScriptSrc:  []string{"'self'"},
			StyleSrc:   []string{"'self'"},
			ImgSrc:     []string{"'self'"},
			FontSrc:    []string{"'self'"},
			ConnectSrc: []string{"'self'"},
			FrameSrc:   []string{"'none'"},
			ObjectSrc:  []string{"'none'"},
			MediaSrc:   []string{"'none'"},
			WorkerSrc:  []string{"'self'"},
			ReportURI:  "/api/csp-report",
		},
		EnableHSTS:            true,
		HSTSMaxAge:            63072000, // 2 years
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		FrameOptions:          "DENY",
		ContentTypeOptions:    "nosniff",
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "no-referrer",
		PermissionsPolicy:     "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), accelerometer=()",
	}
}