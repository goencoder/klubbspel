package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders provides comprehensive security headers for HTTP responses
type SecurityHeaders struct {
	config SecurityHeadersConfig
}

// SecurityHeadersConfig defines security headers configuration
type SecurityHeadersConfig struct {
	// Content Security Policy
	CSPDirectives map[string][]string

	// CORS configuration
	CORSOrigins     []string
	CORSMethods     []string
	CORSHeaders     []string
	CORSCredentials bool
	CORSMaxAge      int

	// HSTS configuration
	HSTSMaxAge            int
	HSTSIncludeSubDomains bool
	HSTSPreload           bool

	// Feature Policy/Permissions Policy
	PermissionsPolicy map[string][]string

	// Additional security settings
	FrameOptions       string // DENY, SAMEORIGIN, or ALLOW-FROM
	ContentTypeOptions bool   // nosniff
	ReferrerPolicy     string
	XSSProtection      string

	// Custom headers
	CustomHeaders map[string]string
}

// NewSecurityHeaders creates a new security headers middleware
func NewSecurityHeaders(config SecurityHeadersConfig) *SecurityHeaders {
	return &SecurityHeaders{
		config: config,
	}
}

// Middleware returns an HTTP middleware for setting security headers
func (sh *SecurityHeaders) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers first
		sh.setCORSHeaders(w, r)

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			sh.handlePreflight(w, r)
			return
		}

		// Set security headers
		sh.setSecurityHeaders(w)

		next.ServeHTTP(w, r)
	})
}

// setCORSHeaders sets Cross-Origin Resource Sharing headers
func (sh *SecurityHeaders) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed
	if sh.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else if len(sh.config.CORSOrigins) == 1 && sh.config.CORSOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// Set allowed methods
	if len(sh.config.CORSMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(sh.config.CORSMethods, ", "))
	}

	// Set allowed headers
	if len(sh.config.CORSHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(sh.config.CORSHeaders, ", "))
	}

	// Set credentials
	if sh.config.CORSCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Set max age for preflight caching
	if sh.config.CORSMaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", string(rune(sh.config.CORSMaxAge)))
	}
}

// setSecurityHeaders sets various security headers
func (sh *SecurityHeaders) setSecurityHeaders(w http.ResponseWriter) {
	// Content Security Policy
	if csp := sh.buildCSP(); csp != "" {
		w.Header().Set("Content-Security-Policy", csp)
	}

	// HTTP Strict Transport Security
	if hsts := sh.buildHSTS(); hsts != "" {
		w.Header().Set("Strict-Transport-Security", hsts)
	}

	// Permissions Policy
	if pp := sh.buildPermissionsPolicy(); pp != "" {
		w.Header().Set("Permissions-Policy", pp)
	}

	// X-Frame-Options
	if sh.config.FrameOptions != "" {
		w.Header().Set("X-Frame-Options", sh.config.FrameOptions)
	}

	// X-Content-Type-Options
	if sh.config.ContentTypeOptions {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	// Referrer-Policy
	if sh.config.ReferrerPolicy != "" {
		w.Header().Set("Referrer-Policy", sh.config.ReferrerPolicy)
	}

	// X-XSS-Protection
	if sh.config.XSSProtection != "" {
		w.Header().Set("X-XSS-Protection", sh.config.XSSProtection)
	}

	// Custom headers
	for key, value := range sh.config.CustomHeaders {
		w.Header().Set(key, value)
	}

	// Additional security headers
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// handlePreflight handles CORS preflight requests
func (sh *SecurityHeaders) handlePreflight(w http.ResponseWriter, r *http.Request) {
	// Set basic preflight headers
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(sh.config.CORSMethods, ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(sh.config.CORSHeaders, ", "))

	if sh.config.CORSMaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", string(rune(sh.config.CORSMaxAge)))
	}

	w.WriteHeader(http.StatusNoContent)
}

// isOriginAllowed checks if an origin is in the allowed list
func (sh *SecurityHeaders) isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range sh.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}

		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}

	return false
}

// buildCSP constructs the Content Security Policy header value
func (sh *SecurityHeaders) buildCSP() string {
	if len(sh.config.CSPDirectives) == 0 {
		return ""
	}

	var directives []string
	for directive, sources := range sh.config.CSPDirectives {
		if len(sources) > 0 {
			directives = append(directives, directive+" "+strings.Join(sources, " "))
		}
	}

	return strings.Join(directives, "; ")
}

// buildHSTS constructs the HTTP Strict Transport Security header value
func (sh *SecurityHeaders) buildHSTS() string {
	if sh.config.HSTSMaxAge <= 0 {
		return ""
	}

	hsts := "max-age=" + string(rune(sh.config.HSTSMaxAge))

	if sh.config.HSTSIncludeSubDomains {
		hsts += "; includeSubDomains"
	}

	if sh.config.HSTSPreload {
		hsts += "; preload"
	}

	return hsts
}

// buildPermissionsPolicy constructs the Permissions Policy header value
func (sh *SecurityHeaders) buildPermissionsPolicy() string {
	if len(sh.config.PermissionsPolicy) == 0 {
		return ""
	}

	var policies []string
	for feature, allowlist := range sh.config.PermissionsPolicy {
		if len(allowlist) > 0 {
			policies = append(policies, feature+"=("+strings.Join(allowlist, " ")+")")
		} else {
			policies = append(policies, feature+"=()")
		}
	}

	return strings.Join(policies, ", ")
}

// GetSecureConfig returns a secure default configuration
func GetSecureConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		// Content Security Policy - Very restrictive
		CSPDirectives: map[string][]string{
			"default-src":               {"'self'"},
			"script-src":                {"'self'", "'unsafe-inline'", "'unsafe-eval'"}, // Allow inline for React
			"style-src":                 {"'self'", "'unsafe-inline'"},                  // Allow inline styles
			"img-src":                   {"'self'", "data:", "https:"},
			"font-src":                  {"'self'", "https://fonts.gstatic.com"},
			"connect-src":               {"'self'"},
			"media-src":                 {"'none'"},
			"object-src":                {"'none'"},
			"child-src":                 {"'none'"},
			"frame-src":                 {"'none'"},
			"worker-src":                {"'self'"},
			"frame-ancestors":           {"'none'"},
			"form-action":               {"'self'"},
			"upgrade-insecure-requests": {},
		},

		// CORS configuration for development and production
		CORSOrigins: []string{
			"http://localhost:5000", // Development frontend
			"http://localhost:3000", // Alternative dev port
			"https://klubbspel.app", // Production domain (example)
		},
		CORSMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD",
		},
		CORSHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Requested-With",
		},
		CORSCredentials: true,
		CORSMaxAge:      86400, // 24 hours

		// HSTS configuration
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubDomains: true,
		HSTSPreload:           true,

		// Permissions Policy - Deny all dangerous features
		PermissionsPolicy: map[string][]string{
			"camera":                          {},         // Deny camera access
			"microphone":                      {},         // Deny microphone access
			"geolocation":                     {},         // Deny location access
			"notifications":                   {},         // Deny notifications
			"persistent-storage":              {},         // Deny persistent storage
			"push":                            {},         // Deny push notifications
			"speaker-selection":               {},         // Deny speaker selection
			"accelerometer":                   {},         // Deny accelerometer
			"ambient-light-sensor":            {},         // Deny ambient light sensor
			"autoplay":                        {},         // Deny autoplay
			"battery":                         {},         // Deny battery API
			"display-capture":                 {},         // Deny display capture
			"document-domain":                 {},         // Deny document.domain
			"encrypted-media":                 {},         // Deny encrypted media
			"execution-while-not-rendered":    {},         // Deny execution while not rendered
			"execution-while-out-of-viewport": {},         // Deny execution while out of viewport
			"fullscreen":                      {"'self'"}, // Allow fullscreen on same origin
			"gyroscope":                       {},         // Deny gyroscope
			"magnetometer":                    {},         // Deny magnetometer
			"payment":                         {},         // Deny payment API
			"picture-in-picture":              {},         // Deny picture-in-picture
			"publickey-credentials-get":       {},         // Deny WebAuthn
			"screen-wake-lock":                {},         // Deny screen wake lock
			"sync-xhr":                        {},         // Deny synchronous XHR
			"usb":                             {},         // Deny USB API
			"web-share":                       {},         // Deny Web Share API
			"xr-spatial-tracking":             {},         // Deny XR spatial tracking
		},

		// Frame options
		FrameOptions: "DENY",

		// Content type options
		ContentTypeOptions: true,

		// Referrer policy
		ReferrerPolicy: "strict-origin-when-cross-origin",

		// XSS protection
		XSSProtection: "1; mode=block",

		// Custom security headers
		CustomHeaders: map[string]string{
			"X-Permitted-Cross-Domain-Policies": "none",
			"X-Download-Options":                "noopen",
			"X-DNS-Prefetch-Control":            "off",
		},
	}
}

// GetDevelopmentConfig returns a more permissive configuration for development
func GetDevelopmentConfig() SecurityHeadersConfig {
	config := GetSecureConfig()

	// More permissive CSP for development
	config.CSPDirectives = map[string][]string{
		"default-src": {"'self'"},
		"script-src":  {"'self'", "'unsafe-inline'", "'unsafe-eval'", "blob:"},
		"style-src":   {"'self'", "'unsafe-inline'"},
		"img-src":     {"'self'", "data:", "blob:", "*"},
		"font-src":    {"'self'", "data:", "*"},
		"connect-src": {"'self'", "ws:", "wss:", "*"}, // Allow WebSocket connections
		"media-src":   {"'self'", "blob:", "data:"},
		"object-src":  {"'none'"},
		"frame-src":   {"'self'"},
		"worker-src":  {"'self'", "blob:"},
	}

	// More permissive CORS for development
	config.CORSOrigins = []string{"*"}

	// Disable HSTS for development (allows HTTP)
	config.HSTSMaxAge = 0

	return config
}
