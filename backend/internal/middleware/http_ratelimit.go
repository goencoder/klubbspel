package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HTTPRateLimiter provides rate limiting for HTTP requests
type HTTPRateLimiter struct {
	// Per-IP rate limiters
	ipLimiters map[string]*rate.Limiter
	mutex      sync.RWMutex

	// Configuration
	config HTTPRateLimitConfig
}

// HTTPRateLimitConfig defines HTTP rate limiting parameters
type HTTPRateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int

	// Per-endpoint limits
	EndpointLimits map[string]HTTPEndpointLimit

	// Cleanup settings
	CleanupInterval time.Duration
}

// HTTPEndpointLimit defines limits for specific HTTP endpoints
type HTTPEndpointLimit struct {
	RequestsPerSecond float64
	BurstSize         int
	Methods           []string // HTTP methods this limit applies to
}

// NewHTTPRateLimiter creates a new HTTP rate limiter
func NewHTTPRateLimiter(config HTTPRateLimitConfig) *HTTPRateLimiter {
	return &HTTPRateLimiter{
		ipLimiters: make(map[string]*rate.Limiter),
		config:     config,
	}
}

// Middleware returns an HTTP middleware for rate limiting
func (rl *HTTPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := rl.extractHTTPClientIP(r)

		// Check general rate limit
		if !rl.checkHTTPRateLimit(clientIP) {
			rl.writeRateLimitError(w, "RATE_LIMIT_EXCEEDED")
			return
		}

		// Check endpoint-specific rate limit
		if !rl.checkHTTPEndpointRateLimit(r, clientIP) {
			rl.writeRateLimitError(w, "ENDPOINT_RATE_LIMIT_EXCEEDED")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractHTTPClientIP extracts the client IP from HTTP request
func (rl *HTTPRateLimiter) extractHTTPClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// checkHTTPRateLimit checks the general rate limit for an IP
func (rl *HTTPRateLimiter) checkHTTPRateLimit(ip string) bool {
	rl.mutex.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.mutex.RUnlock()

	if !exists {
		rl.mutex.Lock()
		if limiter, exists = rl.ipLimiters[ip]; !exists {
			limiter = rate.NewLimiter(
				rate.Limit(rl.config.RequestsPerSecond),
				rl.config.BurstSize,
			)
			rl.ipLimiters[ip] = limiter
		}
		rl.mutex.Unlock()
	}

	return limiter.Allow()
}

// checkHTTPEndpointRateLimit checks endpoint-specific rate limits
func (rl *HTTPRateLimiter) checkHTTPEndpointRateLimit(r *http.Request, ip string) bool {
	path := r.URL.Path
	method := r.Method

	for pattern, limit := range rl.config.EndpointLimits {
		if rl.matchesPattern(path, pattern) && rl.matchesMethod(method, limit.Methods) {
			limiterKey := fmt.Sprintf("http:%s:%s:%s", method, pattern, ip)

			rl.mutex.RLock()
			limiter, exists := rl.ipLimiters[limiterKey]
			rl.mutex.RUnlock()

			if !exists {
				rl.mutex.Lock()
				if limiter, exists = rl.ipLimiters[limiterKey]; !exists {
					limiter = rate.NewLimiter(
						rate.Limit(limit.RequestsPerSecond),
						limit.BurstSize,
					)
					rl.ipLimiters[limiterKey] = limiter
				}
				rl.mutex.Unlock()
			}

			return limiter.Allow()
		}
	}

	return true // No specific limit found
}

// matchesPattern checks if a path matches a pattern (simple prefix matching for now)
func (rl *HTTPRateLimiter) matchesPattern(path, pattern string) bool {
	return strings.HasPrefix(path, pattern)
}

// matchesMethod checks if a method is in the allowed methods list
func (rl *HTTPRateLimiter) matchesMethod(method string, allowedMethods []string) bool {
	if len(allowedMethods) == 0 {
		return true // No method restriction
	}

	for _, allowed := range allowedMethods {
		if strings.EqualFold(method, allowed) {
			return true
		}
	}
	return false
}

// writeRateLimitError writes a rate limit error response
func (rl *HTTPRateLimiter) writeRateLimitError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60") // Suggest retry after 60 seconds
	w.WriteHeader(http.StatusTooManyRequests)

	response := fmt.Sprintf(`{
		"error": {
			"code": "RATE_LIMIT_EXCEEDED",
			"message": "%s",
			"details": "Too many requests. Please try again later."
		}
	}`, message)

	if _, err := w.Write([]byte(response)); err != nil {
		fmt.Printf("HTTP rate limiter failed to write response: %v\n", err)
	}
}

// GetDefaultHTTPConfig returns default HTTP rate limiting configuration
func GetDefaultHTTPConfig() HTTPRateLimitConfig {
	return HTTPRateLimitConfig{
		RequestsPerSecond: 20.0, // 20 requests per second per IP
		BurstSize:         40,   // Allow bursts up to 40 requests

		EndpointLimits: map[string]HTTPEndpointLimit{
			// Authentication endpoints - very restrictive
			"/v1/auth/magic-link": {
				RequestsPerSecond: 0.1, // 1 request per 10 seconds
				BurstSize:         2,   // Very limited
				Methods:           []string{"POST"},
			},
			"/v1/auth/validate": {
				RequestsPerSecond: 1.0, // 1 request per second
				BurstSize:         3,
				Methods:           []string{"POST"},
			},

			// Club creation - restrictive
			"/v1/clubs POST": {
				RequestsPerSecond: 0.1, // 1 request per 10 seconds
				BurstSize:         2,
				Methods:           []string{"POST"},
			},

			// Invitations and modifications - restrictive
			"/v1/clubs/members": { // This will match membership operations
				RequestsPerSecond: 0.2, // 1 request per 5 seconds
				BurstSize:         3,
				Methods:           []string{"POST", "PUT", "DELETE"},
			},

			// Read operations - more permissive
			"/v1/clubs GET": {
				RequestsPerSecond: 10.0, // 10 requests per second
				BurstSize:         20,
				Methods:           []string{"GET"},
			},
			"/v1/players": {
				RequestsPerSecond: 10.0,
				BurstSize:         20,
				Methods:           []string{"GET"},
			},
			"/v1/series": {
				RequestsPerSecond: 10.0,
				BurstSize:         20,
				Methods:           []string{"GET"},
			},
		},

		CleanupInterval: 10 * time.Minute,
	}
}
