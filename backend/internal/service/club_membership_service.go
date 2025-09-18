package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/goencoder/klubbspel/backend/internal/email"
	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
)

// ClubMembershipService handles club membership operations
type ClubMembershipService struct {
	PlayerRepo *repo.PlayerRepo
	ClubRepo   *repo.ClubRepo
	TokenRepo  *repo.TokenRepo // Add token repo for magic link generation
	EmailSvc   email.Service   // Add email service for invitations
}

// JoinClub allows a user to join a club (self-registration)
func (s *ClubMembershipService) JoinClub(ctx context.Context, req *pb.JoinClubRequest) (*pb.JoinClubResponse, error) {
	// Get authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate input
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_REQUIRED")
	}

	// Validate club exists
	_, err := s.ClubRepo.FindByID(ctx, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	// Check if already a member
	isMember, err := subject.IsClubMember(ctx, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.Internal, "MEMBERSHIP_CHECK_FAILED")
	}
	if isMember {
		return nil, status.Error(codes.AlreadyExists, "ALREADY_MEMBER")
	}

	// Create membership
	clubObjID, err := primitive.ObjectIDFromHex(req.ClubId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "INVALID_CLUB_ID")
	}

	membership := &repo.ClubMembership{
		ClubID:   clubObjID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	err = s.PlayerRepo.AddClubMembership(ctx, subject.GetEmail(), membership)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_JOIN_CLUB")
	}

	// Convert to protobuf
	pbMembership := &pb.ClubMembership{
		ClubId:   req.ClubId,
		Role:     pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER,
		JoinedAt: timestamppb.New(membership.JoinedAt),
	}

	return &pb.JoinClubResponse{
		Success:    true,
		Membership: pbMembership,
	}, nil
}

// LeaveClub allows a user to leave a club
func (s *ClubMembershipService) LeaveClub(ctx context.Context, req *pb.LeaveClubRequest) (*pb.LeaveClubResponse, error) {
	// Get authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate input
	if req.ClubId == "" || req.PlayerId == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_AND_PLAYER_ID_REQUIRED")
	}

	// Ensure subject data is loaded (this will populate the email)
	if lazySubject, ok := subject.(*LazySubject); ok {
		if err := lazySubject.ensureLoaded(ctx); err != nil {
			return nil, status.Error(codes.Unauthenticated, "FAILED_TO_LOAD_USER_DATA")
		}
	}

	// Check if user can perform this action (self or admin)
	userEmail := strings.ToLower(strings.TrimSpace(subject.GetEmail()))
	targetPlayer, err := s.PlayerRepo.FindByID(ctx, req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	targetEmail := strings.ToLower(strings.TrimSpace(targetPlayer.Email))

	if userEmail != targetEmail {
		// Check if user is club admin
		isAdmin, err := subject.IsClubAdmin(ctx, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "ADMIN_CHECK_FAILED")
		}
		if !isAdmin {
			return nil, status.Error(codes.PermissionDenied, "CAN_ONLY_REMOVE_SELF_OR_AS_ADMIN")
		}
	}

	// Remove membership completely
	err = s.PlayerRepo.RemoveClubMembership(ctx, targetPlayer.Email, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_LEAVE_CLUB")
	}

	return &pb.LeaveClubResponse{
		Success: true,
	}, nil
}

// InvitePlayer invites a player to join a club (admin only)
func (s *ClubMembershipService) InvitePlayer(ctx context.Context, req *pb.InvitePlayerRequest) (*pb.InvitePlayerResponse, error) {
	// Get authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate input
	if req.ClubId == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_AND_EMAIL_REQUIRED")
	}

	// Check if user is club admin or platform owner
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "PLATFORM_OWNER_CHECK_FAILED")
	}

	if !isPlatformOwner {
		isAdmin, err := subject.IsClubAdmin(ctx, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "ADMIN_CHECK_FAILED")
		}
		if !isAdmin {
			return nil, status.Error(codes.PermissionDenied, "ADMIN_REQUIRED")
		}
	}

	// Check if club exists
	club, err := s.ClubRepo.FindByID(ctx, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	// Find or create target player
	_, err = s.PlayerRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Player doesn't exist, create them
		_, err = s.PlayerRepo.CreateWithEmail(ctx, req.Email, "", "", "")
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_PLAYER")
		}
	}

	// Check if already a member
	isMember, err := s.PlayerRepo.IsClubMember(ctx, req.Email, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.Internal, "MEMBERSHIP_CHECK_FAILED")
	}
	if isMember {
		return nil, status.Error(codes.AlreadyExists, "ALREADY_MEMBER")
	}

	// Create membership
	clubObjID, err := primitive.ObjectIDFromHex(req.ClubId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "INVALID_CLUB_ID")
	}

	role := "member"
	if req.Role == pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN {
		role = "admin"
	}

	membership := &repo.ClubMembership{
		ClubID:   clubObjID,
		Role:     role,
		JoinedAt: time.Now(),
	}

	err = s.PlayerRepo.AddClubMembership(ctx, req.Email, membership)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_ADD_MEMBERSHIP")
	}

	// Send invitation email
	invitationSent := false
	if s.EmailSvc != nil {
		err = s.EmailSvc.SendInvitation(ctx, req.Email, club.Name, subject.GetDisplayName())
		if err != nil {
			// Log error but don't fail the invitation - membership was already created
			// TODO: Consider adding proper logging here
		} else {
			invitationSent = true
		}
	}

	return &pb.InvitePlayerResponse{
		Success:        true,
		InvitationSent: invitationSent,
	}, nil
}

// AddPlayerToClub allows a club admin to add a player to their club
func (s *ClubMembershipService) AddPlayerToClub(ctx context.Context, req *pb.AddPlayerToClubRequest) (*pb.AddPlayerToClubResponse, error) {
	// Get authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate input
	if req.ClubId == "" || req.FirstName == "" || req.LastName == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_FIRST_NAME_AND_LAST_NAME_REQUIRED")
	}

	// Check if user is club admin or platform owner
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "PLATFORM_OWNER_CHECK_FAILED")
	}

	if !isPlatformOwner {
		isAdmin, err := subject.IsClubAdmin(ctx, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "ADMIN_CHECK_FAILED")
		}
		if !isAdmin {
			return nil, status.Error(codes.PermissionDenied, "ADMIN_REQUIRED")
		}
	}

	// Check if club exists
	_, err = s.ClubRepo.FindByID(ctx, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	var player *repo.Player
	var wasNewPlayer bool
	notificationSent := false

	// Generate display name from first and last name
	displayName := strings.TrimSpace(req.FirstName + " " + req.LastName)

	if req.Email != "" {
		// Try to find existing player by email
		existingPlayer, err := s.PlayerRepo.FindByEmail(ctx, req.Email)
		if err != nil {
			// Player doesn't exist, create them with email
			player, err = s.PlayerRepo.CreateWithEmail(ctx, req.Email, req.FirstName, req.LastName, displayName)
			if err != nil {
				if strings.Contains(err.Error(), "email") && strings.Contains(err.Error(), "duplicate") {
					return nil, status.Error(codes.AlreadyExists, "EMAIL_ALREADY_EXISTS")
				}
				return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_PLAYER")
			}
			wasNewPlayer = true
		} else {
			// Player exists, use the existing player
			player = existingPlayer
			wasNewPlayer = false

			// Update player info if it's more complete than what we have
			shouldUpdate := false
			updates := make(map[string]interface{})

			if existingPlayer.FirstName == "" && req.FirstName != "" {
				updates["first_name"] = req.FirstName
				shouldUpdate = true
			}
			if existingPlayer.LastName == "" && req.LastName != "" {
				updates["last_name"] = req.LastName
				shouldUpdate = true
			}
			if existingPlayer.DisplayName == "" && displayName != "" {
				updates["display_name"] = displayName
				shouldUpdate = true
			}

			// Save updates if needed
			if shouldUpdate {
				_, err = s.PlayerRepo.Update(ctx, existingPlayer.ID.Hex(), updates)
				if err != nil {
					// Non-fatal error, continue with membership creation
				}
			}
		}
	} else {
		// Create player without email using the standard Create method
		// Pass the club ID so the player is automatically added to the club
		player, err = s.PlayerRepo.Create(ctx, displayName, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_CREATE_PLAYER")
		}

		// Update first and last names after creation
		updates := map[string]interface{}{
			"first_name": req.FirstName,
			"last_name":  req.LastName,
		}
		_, err = s.PlayerRepo.Update(ctx, player.ID.Hex(), updates)
		if err != nil {
			// Non-fatal error, continue with membership creation
		}

		wasNewPlayer = true
	}

	// Check if already a member
	if req.Email != "" {
		isMember, err := s.PlayerRepo.IsClubMember(ctx, req.Email, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "MEMBERSHIP_CHECK_FAILED")
		}
		if isMember {
			return nil, status.Error(codes.AlreadyExists, "ALREADY_MEMBER")
		}
	} else {
		// For players without email, we need to check manually by looking at their club memberships
		// Since we just created the player, they won't be a member yet, so we can skip this check
		// for new players. For existing email-less players (which shouldn't happen in this flow),
		// we'll rely on the AddClubMembership method to handle duplicates.
	}

	// Create membership (only needed for players with email or existing players without email)
	needsNewMembership := req.Email != "" || (req.Email == "" && !wasNewPlayer)
	
	if needsNewMembership {
		clubObjID, err := primitive.ObjectIDFromHex(req.ClubId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "INVALID_CLUB_ID")
		}

		membership := &repo.ClubMembership{
			ClubID:   clubObjID,
			Role:     "member", // Always add as member initially
			JoinedAt: time.Now(),
		}

		// Use the player's actual email (which might be synthetic for players without email)
		err = s.PlayerRepo.AddClubMembership(ctx, player.Email, membership)
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_ADD_MEMBERSHIP")
		}
	}

	// Send notification email if email provided
	if req.Email != "" && s.EmailSvc != nil {
		var err error

		// Get inviter information
		inviterEmail := subject.GetEmail()
		inviterPlayer, err := s.PlayerRepo.FindByEmail(ctx, inviterEmail)
		var inviterName string
		if err == nil && inviterPlayer != nil {
			if inviterPlayer.DisplayName != "" {
				inviterName = inviterPlayer.DisplayName
			} else if inviterPlayer.FirstName != "" || inviterPlayer.LastName != "" {
				inviterName = strings.TrimSpace(inviterPlayer.FirstName + " " + inviterPlayer.LastName)
			} else {
				inviterName = inviterEmail // Fallback to email
			}
		} else {
			inviterName = inviterEmail // Fallback to email if player not found
		}

		// Get club information
		club, err := s.ClubRepo.FindByID(ctx, req.ClubId)
		var clubName string
		if err == nil && club != nil {
			clubName = club.Name
		} else {
			clubName = "klubben" // Fallback
		}

		// Generate a club-specific return URL
		returnURL := fmt.Sprintf("/clubs/%s", req.ClubId)

		// Send enhanced club invitation email with magic link
		magicToken := uuid.New().String()

		// Create magic link token with 24-hour expiry
		_, err = s.TokenRepo.CreateMagicLinkTokenWithExpiry(ctx, magicToken, req.Email, "", 24*time.Hour)
		if err == nil {
			// Send club invitation magic link email with inviter and club context
			err = s.EmailSvc.SendClubInvitationMagicLink(ctx, req.Email, magicToken, returnURL, clubName, inviterName, inviterEmail)
		}

		if err != nil {
			// Log error but don't fail the operation - membership was already created
			// TODO: Consider adding proper logging here
		} else {
			notificationSent = true
		}
	}

	// Convert player to protobuf for response
	playerService := &PlayerService{Players: s.PlayerRepo}
	pbPlayer := playerService.convertToProtobuf(player)

	return &pb.AddPlayerToClubResponse{
		Success:          true,
		Player:           pbPlayer,
		NotificationSent: notificationSent,
		WasNewPlayer:     wasNewPlayer,
	}, nil
}

// UpdateMemberRole updates a member's role (promote/demote)
func (s *ClubMembershipService) UpdateMemberRole(ctx context.Context, req *pb.UpdateMemberRoleRequest) (*pb.UpdateMemberRoleResponse, error) {
	// Get authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate input
	if req.ClubId == "" || req.PlayerId == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_AND_PLAYER_ID_REQUIRED")
	}

	// Check if user is club admin or platform owner
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "PLATFORM_OWNER_CHECK_FAILED")
	}

	if !isPlatformOwner {
		isAdmin, err := subject.IsClubAdmin(ctx, req.ClubId)
		if err != nil {
			return nil, status.Error(codes.Internal, "ADMIN_CHECK_FAILED")
		}
		if !isAdmin {
			return nil, status.Error(codes.PermissionDenied, "ADMIN_REQUIRED")
		}
	}

	// Validate target player exists
	targetPlayer, err := s.PlayerRepo.FindByID(ctx, req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	// Convert role
	role := "member"
	if req.Role == pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN {
		role = "admin"
	}

	// Update role
	err = s.PlayerRepo.UpdateClubMembershipRole(ctx, req.PlayerId, req.ClubId, role)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_UPDATE_ROLE")
	}

	// Get updated membership for response
	memberships, err := s.PlayerRepo.GetPlayerMemberships(ctx, req.PlayerId, true)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_GET_MEMBERSHIPS")
	}

	// Find the updated membership
	var updatedMembership *pb.ClubMembership
	for _, membership := range memberships {
		if membership.ClubID.Hex() == req.ClubId {
			pbRole := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
			if membership.Role == "admin" {
				pbRole = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
			}

			updatedMembership = &pb.ClubMembership{
				ClubId:   req.ClubId,
				Role:     pbRole,
				JoinedAt: timestamppb.New(membership.JoinedAt),
			}
			break
		}
	}

	// Use target player details for response
	_ = targetPlayer.Email // Avoid unused variable error

	return &pb.UpdateMemberRoleResponse{
		Success:    true,
		Membership: updatedMembership,
	}, nil
}

// ListClubMembers lists all members of a club
func (s *ClubMembershipService) ListClubMembers(ctx context.Context, req *pb.ListClubMembersRequest) (*pb.ListClubMembersResponse, error) {
	// Validate input
	if req.ClubId == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_REQUIRED")
	}

	// Check if club exists
	_, err := s.ClubRepo.FindByID(ctx, req.ClubId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	// List members (activeOnly parameter is no longer relevant)
	players, err := s.PlayerRepo.ListClubMembers(ctx, req.ClubId, false)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_LIST_MEMBERS")
	}

	// Convert to response
	var members []*pb.ClubMemberInfo
	for _, player := range players {
		// Find the relevant membership
		for _, membership := range player.ClubMemberships {
			if membership.ClubID.Hex() == req.ClubId {
				pbRole := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
				if membership.Role == "admin" {
					pbRole = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
				}

				pbMembership := &pb.ClubMembership{
					ClubId:   req.ClubId,
					Role:     pbRole,
					JoinedAt: timestamppb.New(membership.JoinedAt),
				}

				// Hide synthetic emails from API responses
				email := player.Email
				if repo.IsSyntheticEmail(email) {
					email = ""
				}

				members = append(members, &pb.ClubMemberInfo{
					PlayerId:    player.ID.Hex(),
					DisplayName: player.DisplayName,
					Email:       email,
					Membership:  pbMembership,
				})
				break
			}
		}
	}

	return &pb.ListClubMembersResponse{
		Members:       members,
		NextPageToken: "", // TODO: Implement pagination if needed
	}, nil
}

// ListPlayerMemberships lists all club memberships for a player
func (s *ClubMembershipService) ListPlayerMemberships(ctx context.Context, req *pb.ListPlayerMembershipsRequest) (*pb.ListPlayerMembershipsResponse, error) {
	// Validate input
	if req.PlayerId == "" {
		return nil, status.Error(codes.InvalidArgument, "PLAYER_ID_REQUIRED")
	}

	// Get player (just validate it exists)
	_, err := s.PlayerRepo.FindByID(ctx, req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	// Get memberships (activeOnly parameter is no longer relevant)
	memberships, err := s.PlayerRepo.GetPlayerMemberships(ctx, req.PlayerId, false)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_GET_MEMBERSHIPS")
	}

	// Convert to response
	var pbMemberships []*pb.PlayerMembershipInfo
	for _, membership := range memberships {
		// Get club details
		club, err := s.ClubRepo.FindByID(ctx, membership.ClubID.Hex())
		if err != nil {
			continue // Skip if club not found
		}

		pbRole := pb.MembershipRole_MEMBERSHIP_ROLE_MEMBER
		if membership.Role == "admin" {
			pbRole = pb.MembershipRole_MEMBERSHIP_ROLE_ADMIN
		}

		pbMembership := &pb.ClubMembership{
			ClubId:   membership.ClubID.Hex(),
			Role:     pbRole,
			JoinedAt: timestamppb.New(membership.JoinedAt),
		}

		pbMemberships = append(pbMemberships, &pb.PlayerMembershipInfo{
			ClubId:     membership.ClubID.Hex(),
			ClubName:   club.Name,
			Membership: pbMembership,
		})
	}

	return &pb.ListPlayerMembershipsResponse{
		Memberships: pbMemberships,
	}, nil
}
