package middleware

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter provides rate limiting functionality for gRPC services
type RateLimiter struct {
	// Global rate limiter
	globalLimiter *rate.Limiter

	// Per-IP rate limiters
	ipLimiters map[string]*rate.Limiter
	ipMutex    sync.RWMutex

	// Cleanup goroutine control
	cleanupInterval time.Duration
	stopCleanup     chan struct{}

	// Configuration
	config RateLimitConfig
}

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	// Global limits
	GlobalRequestsPerSecond float64
	GlobalBurstSize         int

	// Per-IP limits
	IPRequestsPerSecond float64
	IPBurstSize         int

	// Per-endpoint limits
	EndpointLimits map[string]EndpointLimit

	// Cleanup settings
	IPLimiterTTL    time.Duration
	CleanupInterval time.Duration
}

// EndpointLimit defines limits for specific endpoints
type EndpointLimit struct {
	RequestsPerSecond float64
	BurstSize         int

	// Special limits for authenticated vs unauthenticated users
	AuthenticatedMultiplier float64
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		globalLimiter:   rate.NewLimiter(rate.Limit(config.GlobalRequestsPerSecond), config.GlobalBurstSize),
		ipLimiters:      make(map[string]*rate.Limiter),
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
		config:          config,
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

// UnaryInterceptor returns a gRPC unary interceptor for rate limiting
func (rl *RateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract client IP
		clientIP := rl.extractClientIP(ctx)

		// Check global rate limit
		if !rl.globalLimiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "GLOBAL_RATE_LIMIT_EXCEEDED")
		}

		// Check per-IP rate limit
		if !rl.checkIPRateLimit(clientIP) {
			return nil, status.Error(codes.ResourceExhausted, "IP_RATE_LIMIT_EXCEEDED")
		}

		// Check endpoint-specific rate limit
		if !rl.checkEndpointRateLimit(ctx, info.FullMethod, clientIP) {
			return nil, status.Error(codes.ResourceExhausted, "ENDPOINT_RATE_LIMIT_EXCEEDED")
		}

		return handler(ctx, req)
	}
}

// extractClientIP extracts the client IP address from the gRPC context
func (rl *RateLimiter) extractClientIP(ctx context.Context) string {
	// First try to get IP from X-Forwarded-For header (for load balancers)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			// X-Forwarded-For can contain multiple IPs, take the first one
			ips := strings.Split(xff[0], ",")
			if len(ips) > 0 {
				ip := strings.TrimSpace(ips[0])
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}

		// Try X-Real-IP header
		if xri := md.Get("x-real-ip"); len(xri) > 0 {
			if net.ParseIP(xri[0]) != nil {
				return xri[0]
			}
		}
	}

	// Fall back to peer address
	if peer, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := peer.Addr.(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
	}

	return "unknown"
}

// checkIPRateLimit checks and updates the rate limit for a specific IP
func (rl *RateLimiter) checkIPRateLimit(ip string) bool {
	rl.ipMutex.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.ipMutex.RUnlock()

	if !exists {
		// Create new limiter for this IP
		rl.ipMutex.Lock()
		// Double-check after acquiring write lock
		if limiter, exists = rl.ipLimiters[ip]; !exists {
			limiter = rate.NewLimiter(
				rate.Limit(rl.config.IPRequestsPerSecond),
				rl.config.IPBurstSize,
			)
			rl.ipLimiters[ip] = limiter
		}
		rl.ipMutex.Unlock()
	}

	return limiter.Allow()
}

// checkEndpointRateLimit checks endpoint-specific rate limits
func (rl *RateLimiter) checkEndpointRateLimit(ctx context.Context, method, ip string) bool {
	endpointLimit, exists := rl.config.EndpointLimits[method]
	if !exists {
		// No specific limit for this endpoint
		return true
	}

	// Check if user is authenticated for multiplier
	multiplier := 1.0
	if rl.isAuthenticated(ctx) {
		multiplier = endpointLimit.AuthenticatedMultiplier
		if multiplier == 0 {
			multiplier = 1.0
		}
	}

	// Create endpoint-specific limiter key
	limiterKey := fmt.Sprintf("endpoint:%s:%s", method, ip)

	rl.ipMutex.RLock()
	limiter, exists := rl.ipLimiters[limiterKey]
	rl.ipMutex.RUnlock()

	if !exists {
		rl.ipMutex.Lock()
		if limiter, exists = rl.ipLimiters[limiterKey]; !exists {
			limiter = rate.NewLimiter(
				rate.Limit(endpointLimit.RequestsPerSecond*multiplier),
				int(float64(endpointLimit.BurstSize)*multiplier),
			)
			rl.ipLimiters[limiterKey] = limiter
		}
		rl.ipMutex.Unlock()
	}

	return limiter.Allow()
}

// isAuthenticated checks if the request is from an authenticated user
func (rl *RateLimiter) isAuthenticated(ctx context.Context) bool {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		authHeaders := md.Get("authorization")
		return len(authHeaders) > 0 && strings.HasPrefix(authHeaders[0], "Bearer ")
	}
	return false
}

// cleanupRoutine periodically cleans up old IP limiters
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes old and unused IP limiters
func (rl *RateLimiter) cleanup() {
	rl.ipMutex.Lock()
	defer rl.ipMutex.Unlock()

	// For now, we'll keep all limiters to maintain rate limiting state
	// In a production environment, you might want to remove limiters
	// that haven't been used for a certain period

	// Example cleanup logic (commented out):
	// cutoff := time.Now().Add(-rl.config.IPLimiterTTL)
	// for ip, limiter := range rl.ipLimiters {
	//     if limiter.LastUsed().Before(cutoff) {
	//         delete(rl.ipLimiters, ip)
	//     }
	// }
}

// Stop stops the rate limiter cleanup routine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// GetDefaultConfig returns a default rate limiting configuration
func GetDefaultConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRequestsPerSecond: 1000.0, // 1000 requests per second globally
		GlobalBurstSize:         2000,   // Allow bursts up to 2000 requests

		IPRequestsPerSecond: 10.0, // 10 requests per second per IP
		IPBurstSize:         20,   // Allow bursts up to 20 requests per IP

		EndpointLimits: map[string]EndpointLimit{
			// Authentication endpoints - more restrictive
			"/klubbspel.v1.AuthService/SendMagicLink": {
				RequestsPerSecond:       0.1, // 1 request per 10 seconds
				BurstSize:               3,   // Allow up to 3 requests in burst
				AuthenticatedMultiplier: 0.2, // Slightly higher for authenticated users
			},
			"/klubbspel.v1.AuthService/ValidateToken": {
				RequestsPerSecond:       1.0, // 1 request per second
				BurstSize:               5,   // Allow up to 5 requests in burst
				AuthenticatedMultiplier: 1.0, // Same for authenticated users
			},

			// Club management endpoints
			"/klubbspel.v1.ClubService/CreateClub": {
				RequestsPerSecond:       0.1, // 1 request per 10 seconds
				BurstSize:               2,   // Very limited
				AuthenticatedMultiplier: 1.0,
			},
			"/klubbspel.v1.ClubMembershipService/InvitePlayer": {
				RequestsPerSecond:       0.2, // 1 request per 5 seconds
				BurstSize:               3,   // Limited invitations
				AuthenticatedMultiplier: 1.0,
			},

			// Read endpoints - more permissive
			"/klubbspel.v1.ClubService/ListClubs": {
				RequestsPerSecond:       5.0, // 5 requests per second
				BurstSize:               10,  // Allow bursts
				AuthenticatedMultiplier: 2.0, // Higher limits for authenticated users
			},
			"/klubbspel.v1.PlayerService/ListPlayers": {
				RequestsPerSecond:       5.0,
				BurstSize:               10,
				AuthenticatedMultiplier: 2.0,
			},
		},

		IPLimiterTTL:    time.Hour,        // Keep IP limiters for 1 hour
		CleanupInterval: 10 * time.Minute, // Cleanup every 10 minutes
	}
}
