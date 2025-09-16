package auth

import (
	"context"
	"fmt"

	"github.com/goencoder/klubbspel/backend/internal/repo"
)

// Subject represents an authenticated user with lazy-loaded authorization context
type Subject interface {
	// Basic identity
	GetEmail() string
	GetFirstName() string
	GetLastName() string
	GetDisplayName() string

	// Platform-level permissions (single DB lookup, cached)
	IsPlatformOwner(ctx context.Context) (bool, error)

	// Club-level permissions (DB lookup if not cached)
	IsClubMember(ctx context.Context, clubID string) (bool, error)
	IsClubAdmin(ctx context.Context, clubID string) (bool, error)
	GetClubRole(ctx context.Context, clubID string) (string, error)

	// Convenience methods
	GetAdminClubIDs(ctx context.Context) ([]string, error)
	CanAccessClub(ctx context.Context, clubID string) (bool, error)

	// Authorization helpers (business logic)
	CanReportMatch(ctx context.Context, playerAEmail, playerBEmail string) bool
	CanManageClub(ctx context.Context, clubID string) (bool, error)
	CanCreateSeries(ctx context.Context, clubID string) (bool, error)
	CanInviteToClub(ctx context.Context, clubID string) (bool, error)
}

// LazySubject implements Subject with lazy loading and context caching
type LazySubject struct {
	email     string
	firstName string
	lastName  string

	// Cached player data (loaded once per request)
	player *repo.Player
	loaded bool

	// Repository for DB lookups
	playerRepo *repo.PlayerRepo
}

// NewLazySubject creates a new LazySubject for the given email
func NewLazySubject(email, firstName, lastName string, playerRepo *repo.PlayerRepo) *LazySubject {
	return &LazySubject{
		email:      email,
		firstName:  firstName,
		lastName:   lastName,
		playerRepo: playerRepo,
	}
}

// loadPlayer loads player data from database if not already loaded
func (s *LazySubject) loadPlayer(ctx context.Context) error {
	if s.loaded {
		return nil
	}

	player, err := s.playerRepo.FindByEmail(ctx, s.email)
	if err != nil {
		return fmt.Errorf("failed to load player data: %w", err)
	}

	s.player = player
	s.loaded = true
	return nil
}

// Basic identity methods (no DB lookup required)
func (s *LazySubject) GetEmail() string {
	return s.email
}

func (s *LazySubject) GetFirstName() string {
	return s.firstName
}

func (s *LazySubject) GetLastName() string {
	return s.lastName
}

func (s *LazySubject) GetDisplayName() string {
	if s.firstName != "" && s.lastName != "" {
		return fmt.Sprintf("%s %s", s.firstName, s.lastName)
	}
	return s.email
}

// Platform-level permissions (single DB lookup, cached in context)
func (s *LazySubject) IsPlatformOwner(ctx context.Context) (bool, error) {
	if err := s.loadPlayer(ctx); err != nil {
		return false, err
	}
	return s.player.IsPlatformOwner, nil
}

// Club-level permissions (DB lookup if not cached)
func (s *LazySubject) IsClubMember(ctx context.Context, clubID string) (bool, error) {
	if err := s.loadPlayer(ctx); err != nil {
		return false, err
	}

	for _, membership := range s.player.ClubMemberships {
		if membership.ClubID.Hex() == clubID {
			return true, nil
		}
	}
	return false, nil
}

func (s *LazySubject) IsClubAdmin(ctx context.Context, clubID string) (bool, error) {
	if err := s.loadPlayer(ctx); err != nil {
		return false, err
	}

	for _, membership := range s.player.ClubMemberships {
		if membership.ClubID.Hex() == clubID && membership.Role == "admin" {
			return true, nil
		}
	}
	return false, nil
}

func (s *LazySubject) GetClubRole(ctx context.Context, clubID string) (string, error) {
	if err := s.loadPlayer(ctx); err != nil {
		return "", err
	}

	for _, membership := range s.player.ClubMemberships {
		if membership.ClubID.Hex() == clubID {
			return membership.Role, nil
		}
	}
	return "", nil
}

// Convenience methods
func (s *LazySubject) GetAdminClubIDs(ctx context.Context) ([]string, error) {
	if err := s.loadPlayer(ctx); err != nil {
		return nil, err
	}

	var adminClubs []string
	for _, membership := range s.player.ClubMemberships {
		if membership.Role == "admin" {
			adminClubs = append(adminClubs, membership.ClubID.Hex())
		}
	}
	return adminClubs, nil
}

func (s *LazySubject) CanAccessClub(ctx context.Context, clubID string) (bool, error) {
	// Platform owners can access all clubs
	isPlatformOwner, err := s.IsPlatformOwner(ctx)
	if err != nil {
		return false, err
	}
	if isPlatformOwner {
		return true, nil
	}

	// Check club membership
	return s.IsClubMember(ctx, clubID)
}

// Authorization helpers (business logic)
func (s *LazySubject) CanReportMatch(ctx context.Context, playerAEmail, playerBEmail string) bool {
	// Players can only report matches they participated in
	userEmail := s.GetEmail()
	return userEmail == playerAEmail || userEmail == playerBEmail
}

func (s *LazySubject) CanManageClub(ctx context.Context, clubID string) (bool, error) {
	// Platform owners can manage all clubs
	isPlatformOwner, err := s.IsPlatformOwner(ctx)
	if err != nil {
		return false, err
	}
	if isPlatformOwner {
		return true, nil
	}

	// Club admins can manage their clubs
	return s.IsClubAdmin(ctx, clubID)
}

func (s *LazySubject) CanCreateSeries(ctx context.Context, clubID string) (bool, error) {
	// Only club admins and platform owners can create series
	return s.CanManageClub(ctx, clubID)
}

func (s *LazySubject) CanInviteToClub(ctx context.Context, clubID string) (bool, error) {
	// Only club admins and platform owners can invite players
	return s.CanManageClub(ctx, clubID)
}

// Context keys for storing subject in request context
type contextKey string

const subjectContextKey contextKey = "auth_subject"

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

// GetLazySubjectFromContext retrieves LazySubject from context (for specific lazy features)
func GetLazySubjectFromContext(ctx context.Context) *LazySubject {
	if subject := ctx.Value(subjectContextKey); subject != nil {
		if s, ok := subject.(*LazySubject); ok {
			return s
		}
	}
	return nil
}
