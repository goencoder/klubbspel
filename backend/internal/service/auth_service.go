package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/goencoder/klubbspel/backend/internal/email"
	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
)

// AuthService handles authentication operations
type AuthService struct {
	TokenRepo  *repo.TokenRepo
	PlayerRepo *repo.PlayerRepo
	EmailSvc   email.Service
}

// SendMagicLink sends a magic link to the provided email address
func (s *AuthService) SendMagicLink(ctx context.Context, req *pb.SendMagicLinkRequest) (*pb.SendMagicLinkResponse, error) {
	// Validate input
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "EMAIL_REQUIRED")
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Generate magic link token
	magicToken := uuid.New().String()

	// Store magic link token in database
	_, err := s.TokenRepo.CreateMagicLinkToken(ctx, magicToken, email, "")
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_MAGIC_LINK")
	}

	// Send magic link via email
	err = s.EmailSvc.SendMagicLink(ctx, email, magicToken, req.ReturnUrl)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_SEND_EMAIL")
	}

	return &pb.SendMagicLinkResponse{
		Sent:             true,
		ExpiresInSeconds: 15 * 60, // 15 minutes
	}, nil
}

// ValidateToken validates a magic link token and returns an API token
func (s *AuthService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	// Validate input
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "TOKEN_REQUIRED")
	}

	// Get and validate magic link token
	magicToken, err := s.TokenRepo.GetMagicLinkToken(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "INVALID_OR_EXPIRED_TOKEN")
	}

	// Mark magic link token as used
	err = s.TokenRepo.ConsumeMagicLinkToken(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_CONSUME_TOKEN")
	}

	// Find or create player
	player, err := s.PlayerRepo.FindByEmail(ctx, magicToken.Email)
	if err != nil {
		// Player doesn't exist, create one
		player, err = s.PlayerRepo.CreateWithEmail(ctx, magicToken.Email, "", "", "")
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_PLAYER")
		}

		// First user becomes platform owner
		count, _, err := s.PlayerRepo.List(ctx, "", "", 1, "")
		if err == nil && len(count) <= 1 {
			err = s.PlayerRepo.SetPlatformOwner(ctx, magicToken.Email, true)
			if err != nil {
				// Log error but don't fail the authentication
				fmt.Printf("Warning: Failed to set platform owner for first user: %v\n", err)
			}
		}
	}

	// Update last login
	err = s.PlayerRepo.UpdateLastLogin(ctx, magicToken.Email)
	if err != nil {
		// Log error but don't fail the authentication
		fmt.Printf("Warning: Failed to update last login: %v\n", err)
	}

	// Generate API token
	apiToken := uuid.New().String()

	// Store API token
	apiTokenObj, err := s.TokenRepo.CreateAPIToken(ctx, apiToken, magicToken.Email, player.ID, "", "")
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_API_TOKEN")
	}

	// Convert club memberships
	var clubMemberships []*pb.ClubMembership
	for _, membership := range player.ClubMemberships {
		role := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
		if membership.Role == "admin" {
			role = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
		}

		clubMemberships = append(clubMemberships, &pb.ClubMembership{
			ClubId:   membership.ClubID.Hex(),
			Role:     role,
			JoinedAt: timestamppb.New(membership.JoinedAt),
		})
	}

	authUser := &pb.AuthUser{
		Id:              player.ID.Hex(),
		Email:           player.Email,
		FirstName:       player.FirstName,
		LastName:        player.LastName,
		ClubMemberships: clubMemberships,
		IsPlatformOwner: player.IsPlatformOwner,
		LastLoginAt:     timestampFromTimePtr(player.LastLoginAt),
	}

	return &pb.ValidateTokenResponse{
		ApiToken:  apiToken,
		User:      authUser,
		ExpiresAt: timestamppb.New(apiTokenObj.ExpiresAt),
	}, nil
}

// GetCurrentUser returns information about the current authenticated user
func (s *AuthService) GetCurrentUser(ctx context.Context, req *pb.GetCurrentUserRequest) (*pb.GetCurrentUserResponse, error) {
	// Get user from context (set by auth interceptor)
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "NOT_AUTHENTICATED")
	}

	// Ensure subject data is loaded (this will trigger the DB lookup)
	lazySubject, ok := subject.(*LazySubject)
	if !ok {
		return nil, status.Error(codes.Internal, "INVALID_SUBJECT_TYPE")
	}

	if err := lazySubject.ensureLoaded(ctx); err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	// Get player details from the loaded subject
	player := lazySubject.player

	// Convert club memberships
	var clubMemberships []*pb.ClubMembership
	for _, membership := range player.ClubMemberships {
		role := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
		if membership.Role == "admin" {
			role = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
		}

		clubMemberships = append(clubMemberships, &pb.ClubMembership{
			ClubId:   membership.ClubID.Hex(),
			Role:     role,
			JoinedAt: timestamppb.New(membership.JoinedAt),
		})
	}

	authUser := &pb.AuthUser{
		Id:              player.ID.Hex(),
		Email:           player.Email,
		FirstName:       player.FirstName,
		LastName:        player.LastName,
		ClubMemberships: clubMemberships,
		IsPlatformOwner: player.IsPlatformOwner,
		LastLoginAt:     timestampFromTimePtr(player.LastLoginAt),
	}

	return &pb.GetCurrentUserResponse{
		User: authUser,
	}, nil
}

// RevokeToken revokes the current API token (logout)
func (s *AuthService) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	// Get token from context (set by auth interceptor)
	token := GetTokenFromContext(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "NOT_AUTHENTICATED")
	}

	// Revoke the token
	err := s.TokenRepo.RevokeAPIToken(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_REVOKE_TOKEN")
	}

	return &pb.RevokeTokenResponse{
		Revoked: true,
	}, nil
}

// UpdateProfile updates the current user's profile (first name and last name)
func (s *AuthService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	// Get user from context (set by auth interceptor)
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "NOT_AUTHENTICATED")
	}

	// Validate input
	if req.FirstName == "" || req.LastName == "" {
		return nil, status.Error(codes.InvalidArgument, "FIRST_AND_LAST_NAME_REQUIRED")
	}

	// Ensure subject data is loaded
	lazySubject, ok := subject.(*LazySubject)
	if !ok {
		return nil, status.Error(codes.Internal, "INVALID_SUBJECT_TYPE")
	}

	if err := lazySubject.ensureLoaded(ctx); err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	// Update the player's first and last name
	err := s.PlayerRepo.UpdateProfile(ctx, lazySubject.player.Email, req.FirstName, req.LastName)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_UPDATE_PROFILE")
	}

	// Reload the player data to get the updated information
	updatedPlayer, err := s.PlayerRepo.GetByEmail(ctx, lazySubject.player.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_RELOAD_PLAYER")
	}

	// Convert club memberships
	var clubMemberships []*pb.ClubMembership
	for _, membership := range updatedPlayer.ClubMemberships {
		role := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
		if membership.Role == "admin" {
			role = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
		}

		clubMemberships = append(clubMemberships, &pb.ClubMembership{
			ClubId:   membership.ClubID.Hex(),
			Role:     role,
			JoinedAt: timestamppb.New(membership.JoinedAt),
		})
	}

	authUser := &pb.AuthUser{
		Id:              updatedPlayer.ID.Hex(),
		Email:           updatedPlayer.Email,
		FirstName:       updatedPlayer.FirstName,
		LastName:        updatedPlayer.LastName,
		ClubMemberships: clubMemberships,
		IsPlatformOwner: updatedPlayer.IsPlatformOwner,
		LastLoginAt:     timestampFromTimePtr(updatedPlayer.LastLoginAt),
	}

	return &pb.UpdateProfileResponse{
		User: authUser,
	}, nil
}

// Helper function to convert *time.Time to *timestamppb.Timestamp
func timestampFromTimePtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// Context keys for subject and token
type contextKey string

const (
	subjectContextKey contextKey = "subject"
	tokenContextKey   contextKey = "token"
)

// Subject interface for authorization
type Subject interface {
	GetEmail() string
	GetFirstName() string
	GetLastName() string
	GetDisplayName() string

	IsPlatformOwner(ctx context.Context) (bool, error)
	IsClubAdmin(ctx context.Context, clubID string) (bool, error)
	IsClubMember(ctx context.Context, clubID string) (bool, error)
	GetClubMemberships(ctx context.Context) ([]repo.ClubMembership, error)
	GetClubRole(ctx context.Context, clubID string) (string, error)

	// Convenience methods
	GetAdminClubIDs(ctx context.Context) ([]string, error)
	CanAccessClub(ctx context.Context, clubID string) (bool, error)
	CanManageClub(ctx context.Context, clubID string) (bool, error)
	CanReportMatch(ctx context.Context, playerAEmail, playerBEmail string) bool
}

// LazySubject provides late binding for subject data
type LazySubject struct {
	token      string
	tokenRepo  *repo.TokenRepo
	playerRepo *repo.PlayerRepo

	// Cached data
	email  *string
	player *repo.Player
	loaded bool
}

// NewLazySubject creates a new lazy subject
func NewLazySubject(token string, tokenRepo *repo.TokenRepo, playerRepo *repo.PlayerRepo) *LazySubject {
	return &LazySubject{
		token:      token,
		tokenRepo:  tokenRepo,
		playerRepo: playerRepo,
	}
}

// GetEmail returns the subject's email
func (ls *LazySubject) GetEmail() string {
	// Try to load data if not already loaded
	// Use a background context since this is a getter method
	if err := ls.ensureLoaded(context.Background()); err != nil {
		return ""
	}

	if ls.email != nil {
		return *ls.email
	}
	return ""
}

// ensureLoaded loads subject data if not already loaded
func (ls *LazySubject) ensureLoaded(ctx context.Context) error {
	if ls.loaded {
		return nil
	}

	// Get API token
	apiToken, err := ls.tokenRepo.GetAPIToken(ctx, ls.token)
	if err != nil {
		return err
	}

	// Get player
	player, err := ls.playerRepo.FindByEmail(ctx, apiToken.Email)
	if err != nil {
		return err
	}

	// Cache the data
	ls.email = &apiToken.Email
	ls.player = player
	ls.loaded = true

	// Update last used timestamp (async, don't block)
	go func() {
		ls.tokenRepo.UpdateLastUsed(context.Background(), ls.token)
	}()

	return nil
}

// IsPlatformOwner checks if the subject is a platform owner
func (ls *LazySubject) IsPlatformOwner(ctx context.Context) (bool, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return false, err
	}
	return ls.player.IsPlatformOwner, nil
}

// IsClubAdmin checks if the subject is a club admin
func (ls *LazySubject) IsClubAdmin(ctx context.Context, clubID string) (bool, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return false, err
	}

	return ls.playerRepo.IsClubAdmin(ctx, ls.player.Email, clubID)
}

// IsClubMember checks if the subject is a club member
func (ls *LazySubject) IsClubMember(ctx context.Context, clubID string) (bool, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return false, err
	}

	return ls.playerRepo.IsClubMember(ctx, ls.player.Email, clubID)
}

// GetClubMemberships returns all club memberships
func (ls *LazySubject) GetClubMemberships(ctx context.Context) ([]repo.ClubMembership, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	return ls.player.ClubMemberships, nil
}

// GetFirstName returns the subject's first name
func (ls *LazySubject) GetFirstName() string {
	if err := ls.ensureLoaded(context.Background()); err != nil {
		return ""
	}
	return ls.player.FirstName
}

// GetLastName returns the subject's last name
func (ls *LazySubject) GetLastName() string {
	if err := ls.ensureLoaded(context.Background()); err != nil {
		return ""
	}
	return ls.player.LastName
}

// GetDisplayName returns a display name for the subject
func (ls *LazySubject) GetDisplayName() string {
	if err := ls.ensureLoaded(context.Background()); err != nil {
		return ls.GetEmail()
	}

	if ls.player.FirstName != "" && ls.player.LastName != "" {
		return fmt.Sprintf("%s %s", ls.player.FirstName, ls.player.LastName)
	}

	if ls.player.DisplayName != "" {
		return ls.player.DisplayName
	}

	return ls.GetEmail()
}

// GetClubRole returns the subject's role in a specific club
func (ls *LazySubject) GetClubRole(ctx context.Context, clubID string) (string, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return "", err
	}

	for _, membership := range ls.player.ClubMemberships {
		if membership.ClubID.Hex() == clubID {
			return membership.Role, nil
		}
	}
	return "", nil
}

// GetAdminClubIDs returns club IDs where the subject is an admin
func (ls *LazySubject) GetAdminClubIDs(ctx context.Context) ([]string, error) {
	if err := ls.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	var adminClubs []string
	for _, membership := range ls.player.ClubMemberships {
		if membership.Role == "admin" {
			adminClubs = append(adminClubs, membership.ClubID.Hex())
		}
	}
	return adminClubs, nil
}

// CanAccessClub checks if the subject can access a club
func (ls *LazySubject) CanAccessClub(ctx context.Context, clubID string) (bool, error) {
	// Platform owners can access all clubs
	isPlatformOwner, err := ls.IsPlatformOwner(ctx)
	if err != nil {
		return false, err
	}
	if isPlatformOwner {
		return true, nil
	}

	// Check club membership
	return ls.IsClubMember(ctx, clubID)
}

// CanManageClub checks if the subject can manage a club
func (ls *LazySubject) CanManageClub(ctx context.Context, clubID string) (bool, error) {
	// Platform owners can manage all clubs
	isPlatformOwner, err := ls.IsPlatformOwner(ctx)
	if err != nil {
		return false, err
	}
	if isPlatformOwner {
		return true, nil
	}

	// Club admins can manage their clubs
	return ls.IsClubAdmin(ctx, clubID)
}

// CanReportMatch checks if the subject can report a match
func (ls *LazySubject) CanReportMatch(ctx context.Context, playerAEmail, playerBEmail string) bool {
	// Players can only report matches they participated in
	userEmail := ls.GetEmail()
	return userEmail == playerAEmail || userEmail == playerBEmail
}

// WithSubject adds subject to context
func WithSubject(ctx context.Context, subject Subject) context.Context {
	return context.WithValue(ctx, subjectContextKey, subject)
}

// GetSubjectFromContext retrieves subject from context
func GetSubjectFromContext(ctx context.Context) Subject {
	if subject := ctx.Value(subjectContextKey); subject != nil {
		if s, ok := subject.(Subject); ok {
			return s
		}
	}
	return nil
}

// WithToken adds token to context
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenContextKey, token)
}

// GetTokenFromContext retrieves token from context
func GetTokenFromContext(ctx context.Context) string {
	if token := ctx.Value(tokenContextKey); token != nil {
		if t, ok := token.(string); ok {
			return t
		}
	}
	return ""
}
