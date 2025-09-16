package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthorizationService handles authorization decisions using Go code-based rules
type AuthorizationService struct{}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService() *AuthorizationService {
	return &AuthorizationService{}
}

// AuthorizationPattern defines the type of authorization check needed
type AuthorizationPattern int

const (
	AuthPatternPublic        AuthorizationPattern = iota // No authentication required
	AuthPatternAuthenticated                             // Basic authentication only
	AuthPatternPlatformOwner                             // Platform owner check required
	AuthPatternClubAdmin                                 // Club admin check required
	AuthPatternResourceBased                             // Custom resource-specific logic
)

// GetAuthorizationPattern returns the authorization pattern for a given gRPC method
func (a *AuthorizationService) GetAuthorizationPattern(method string) AuthorizationPattern {
	// Public methods - no authentication required
	publicMethods := map[string]bool{
		"/klubbspel.v1.ClubService/ListClubs":             true,
		"/klubbspel.v1.PlayerService/ListPlayers":         true,
		"/klubbspel.v1.SeriesService/ListSeries":          true,
		"/klubbspel.v1.LeaderboardService/GetLeaderboard": true,
		"/klubbspel.v1.MatchService/ListMatches":          true,
		"/klubbspel.v1.AuthService/SendMagicLink":         true,
		"/klubbspel.v1.AuthService/ValidateToken":         true,
	}

	if publicMethods[method] {
		return AuthPatternPublic
	}

	// Platform owner methods - highest privilege level
	platformOwnerMethods := map[string]bool{
		"/klubbspel.v1.ClubService/DeleteClub":      true,
		"/klubbspel.v1.PlayerService/DeletePlayer":  true,
		"/klubbspel.v1.AdminService/GetSystemStats": true,
		"/klubbspel.v1.AdminService/ManageUsers":    true,
	}

	if platformOwnerMethods[method] {
		return AuthPatternPlatformOwner
	}

	// Club admin methods - require club admin or platform owner
	clubAdminMethods := map[string]bool{
		"/klubbspel.v1.ClubService/UpdateClub":                 true,
		"/klubbspel.v1.SeriesService/CreateSeries":             true,
		"/klubbspel.v1.SeriesService/UpdateSeries":             true,
		"/klubbspel.v1.SeriesService/DeleteSeries":             true,
		"/klubbspel.v1.ClubMembershipService/InvitePlayer":     true,
		"/klubbspel.v1.ClubMembershipService/UpdateMemberRole": true,
		"/klubbspel.v1.ClubMembershipService/ListClubMembers":  true,
		"/klubbspel.v1.PlayerService/CreatePlayer":             true, // Require club admin for player creation
	}

	if clubAdminMethods[method] {
		return AuthPatternClubAdmin
	}

	// Resource-based methods - require custom authorization logic
	resourceBasedMethods := map[string]bool{
		"/klubbspel.v1.MatchService/ReportMatch":          true,
		"/klubbspel.v1.MatchService/UpdateMatch":          true,
		"/klubbspel.v1.MatchService/DeleteMatch":          true,
		"/klubbspel.v1.ClubMembershipService/LeaveClub":   true,
		"/klubbspel.v1.PlayerService/FindMergeCandidates": true, // Custom logic: authenticated users can find candidates
		"/klubbspel.v1.PlayerService/MergePlayer":         true, // Custom logic: users can merge email-less profiles to themselves
	}

	if resourceBasedMethods[method] {
		return AuthPatternResourceBased
	}

	// Default: require authentication for all other methods
	return AuthPatternAuthenticated
}

// RequiresAuthentication returns true if the method requires authentication
func (a *AuthorizationService) RequiresAuthentication(method string) bool {
	return a.GetAuthorizationPattern(method) != AuthPatternPublic
}

// CheckAuthorization performs authorization check based on the method's pattern
func (a *AuthorizationService) CheckAuthorization(ctx context.Context, method string, subject Subject) error {
	pattern := a.GetAuthorizationPattern(method)

	switch pattern {
	case AuthPatternPublic:
		// No authorization required
		return nil

	case AuthPatternAuthenticated:
		// Just check that user is authenticated (subject exists)
		if subject == nil {
			return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
		}
		return nil

	case AuthPatternPlatformOwner:
		// Check platform owner permission
		if subject == nil {
			return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
		}

		isPlatformOwner, err := subject.IsPlatformOwner(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "INVALID_TOKEN")
		}
		if !isPlatformOwner {
			return status.Error(codes.PermissionDenied, "PLATFORM_OWNER_REQUIRED")
		}
		return nil

	case AuthPatternClubAdmin:
		// Club admin check will be performed in the service method
		// since it requires club_id from the request
		if subject == nil {
			return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
		}
		return nil

	case AuthPatternResourceBased:
		// Resource-based authorization will be performed in the service method
		// since it requires request-specific data
		if subject == nil {
			return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
		}
		return nil

	default:
		return status.Error(codes.Internal, "UNKNOWN_AUTHORIZATION_PATTERN")
	}
}

// ExtractClubIDFromMethod attempts to extract club_id from common request patterns
func (a *AuthorizationService) ExtractClubIDFromMethod(method string, req interface{}) string {
	// This is a helper method for club admin checks
	// In practice, each service method would extract the club_id from its specific request type

	// For now, we'll rely on individual service methods to perform club admin checks
	// since they have access to the typed request objects
	return ""
}

// Authorization helper functions for common patterns
func (a *AuthorizationService) CheckClubAccess(ctx context.Context, subject Subject, clubID string) error {
	if subject == nil {
		return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Platform owners can access all clubs
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, "INVALID_TOKEN")
	}
	if isPlatformOwner {
		return nil
	}

	// Check club membership
	canAccess, err := subject.CanAccessClub(ctx, clubID)
	if err != nil {
		return status.Error(codes.Unauthenticated, "INVALID_TOKEN")
	}
	if !canAccess {
		return status.Error(codes.PermissionDenied, "CLUB_ACCESS_REQUIRED")
	}

	return nil
}

func (a *AuthorizationService) CheckClubManagement(ctx context.Context, subject Subject, clubID string) error {
	if subject == nil {
		return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	canManage, err := subject.CanManageClub(ctx, clubID)
	if err != nil {
		return status.Error(codes.Unauthenticated, "INVALID_TOKEN")
	}
	if !canManage {
		return status.Error(codes.PermissionDenied, "CLUB_ADMIN_OR_PLATFORM_OWNER_REQUIRED")
	}

	return nil
}

// Middleware function for gRPC interceptor integration
func (a *AuthorizationService) AuthorizeRequest(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error {
	method := info.FullMethod
	subject := GetSubjectFromContext(ctx)

	return a.CheckAuthorization(ctx, method, subject)
}

// Business logic authorization helpers
func (a *AuthorizationService) CanReportMatch(ctx context.Context, subject Subject, playerAEmail, playerBEmail string) bool {
	if subject == nil {
		return false
	}
	return subject.CanReportMatch(ctx, playerAEmail, playerBEmail)
}

func (a *AuthorizationService) CanCreateSeries(ctx context.Context, subject Subject, clubID string) (bool, error) {
	if subject == nil {
		return false, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}
	return subject.CanCreateSeries(ctx, clubID)
}

func (a *AuthorizationService) CanInviteToClub(ctx context.Context, subject Subject, clubID string) (bool, error) {
	if subject == nil {
		return false, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}
	return subject.CanInviteToClub(ctx, clubID)
}

// Error helpers for consistent error messages
func AuthenticationError() error {
	return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
}

func InvalidTokenError() error {
	return status.Error(codes.Unauthenticated, "INVALID_TOKEN")
}

func PermissionDeniedError(message string) error {
	return status.Error(codes.PermissionDenied, message)
}

func PlatformOwnerRequiredError() error {
	return status.Error(codes.PermissionDenied, "PLATFORM_OWNER_REQUIRED")
}

func ClubAdminRequiredError() error {
	return status.Error(codes.PermissionDenied, "CLUB_ADMIN_OR_PLATFORM_OWNER_REQUIRED")
}

// IsAuthorizationError checks if an error is an authorization-related error
func IsAuthorizationError(err error) bool {
	if err == nil {
		return false
	}

	grpcErr, ok := status.FromError(err)
	if !ok {
		return false
	}

	code := grpcErr.Code()
	return code == codes.Unauthenticated || code == codes.PermissionDenied
}

// IsPublicMethod checks if a method is public (no auth required)
func (a *AuthorizationService) IsPublicMethod(method string) bool {
	return a.GetAuthorizationPattern(method) == AuthPatternPublic
}

// GetRequiredRole returns a human-readable description of the required role for a method
func (a *AuthorizationService) GetRequiredRole(method string) string {
	pattern := a.GetAuthorizationPattern(method)

	switch pattern {
	case AuthPatternPublic:
		return "Public access"
	case AuthPatternAuthenticated:
		return "Authenticated user"
	case AuthPatternPlatformOwner:
		return "Platform owner"
	case AuthPatternClubAdmin:
		return "Club admin or platform owner"
	case AuthPatternResourceBased:
		return "Resource-specific permissions"
	default:
		return "Unknown"
	}
}
