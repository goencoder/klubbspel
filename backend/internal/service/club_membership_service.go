package service

import (
	"context"
	"strings"
	"time"

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
	EmailSvc   email.Service // Add email service for invitations
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

				members = append(members, &pb.ClubMemberInfo{
					PlayerId:    player.ID.Hex(),
					DisplayName: player.DisplayName,
					Email:       player.Email,
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
