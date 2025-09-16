package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	"github.com/goencoder/klubbspel/backend/internal/service"
)

// AuthInterceptor handles gRPC authentication and authorization
type AuthInterceptor struct {
	TokenRepo  *repo.TokenRepo
	PlayerRepo *repo.PlayerRepo
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(tokenRepo *repo.TokenRepo, playerRepo *repo.PlayerRepo) *AuthInterceptor {
	return &AuthInterceptor{
		TokenRepo:  tokenRepo,
		PlayerRepo: playerRepo,
	}
}

// UnaryInterceptor implements gRPC unary interceptor for authentication
func (a *AuthInterceptor) UnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	method := info.FullMethod

	// Check if the method is public (doesn't require authentication)
	if a.isPublicMethod(method) {
		return handler(ctx, req)
	}

	// Extract and validate API token
	token, err := a.extractToken(ctx)
	if err != nil {
		return nil, err
	}

	// Create lazy subject for authorization
	subject := service.NewLazySubject(token, a.TokenRepo, a.PlayerRepo)

	// Add subject and token to context
	ctx = service.WithSubject(ctx, subject)
	ctx = service.WithToken(ctx, token)

	return handler(ctx, req)
}

// extractToken extracts and validates the API token from the request
func (a *AuthInterceptor) extractToken(ctx context.Context) (string, error) {
	// Extract metadata from context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "MISSING_METADATA")
	}

	// Get authorization header
	authHeaders := md["authorization"]
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "MISSING_AUTHORIZATION_HEADER")
	}

	// Parse Bearer token
	const prefix = "Bearer "
	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, prefix) {
		return "", status.Error(codes.Unauthenticated, "INVALID_AUTHORIZATION_FORMAT")
	}

	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "EMPTY_TOKEN")
	}

	// Validate token exists and is not expired (this will trigger DB lookup)
	_, err := a.TokenRepo.GetAPIToken(ctx, token)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "INVALID_OR_EXPIRED_TOKEN")
	}

	return token, nil
}

// isPublicMethod determines if a method can be called without authentication
func (a *AuthInterceptor) isPublicMethod(method string) bool {
	publicMethods := map[string]bool{
		// Club service - public read access
		"/klubbspel.v1.ClubService/ListClubs": true,
		"/klubbspel.v1.ClubService/GetClub":   true,

		// Player service - public read access
		"/klubbspel.v1.PlayerService/ListPlayers": true,
		"/klubbspel.v1.PlayerService/GetPlayer":   true,

		// Series service - public read access
		"/klubbspel.v1.SeriesService/ListSeries": true,
		"/klubbspel.v1.SeriesService/GetSeries":  true,

		// Leaderboard service - public read access
		"/klubbspel.v1.LeaderboardService/GetLeaderboard": true,

		// Match service - public read access
		"/klubbspel.v1.MatchService/ListMatches": true,
		"/klubbspel.v1.MatchService/GetMatch":    true,

		// Auth service - public for magic link flow
		"/klubbspel.v1.AuthService/SendMagicLink": true,
		"/klubbspel.v1.AuthService/ValidateToken": true,
	}

	return publicMethods[method]
}
