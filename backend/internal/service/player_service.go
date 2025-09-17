package service

import (
	"context"
	"sort"
	"strings"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	"github.com/goencoder/klubbspel/backend/internal/util"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PlayerService struct {
	pb.UnimplementedPlayerServiceServer
	Players *repo.PlayerRepo
}

// convertToProtobuf converts a repo.Player to pb.Player
func (s *PlayerService) convertToProtobuf(p *repo.Player) *pb.Player {
	// Convert club memberships
	var clubMemberships []*pb.ClubMembership
	for _, membership := range p.ClubMemberships {
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

	return &pb.Player{
		Id:              p.ID.Hex(),
		DisplayName:     p.DisplayName,
		Active:          p.Active,
		Email:           p.Email,
		FirstName:       p.FirstName,
		LastName:        p.LastName,
		ClubMemberships: clubMemberships,
		IsPlatformOwner: p.IsPlatformOwner,
		LastLoginAt:     timestampFromTimePtr(p.LastLoginAt),
	}
}

func (s *PlayerService) CreatePlayer(ctx context.Context, in *pb.CreatePlayerRequest) (*pb.CreatePlayerResponse, error) {
	// Authorization check: Ensure user can create players for the specified club
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// If a club ID is specified, verify the user can manage that club
	if clubID := in.GetInitialClubId(); clubID != "" {
		// Check if user is platform owner (can create players for any club)
		isPlatformOwner, err := subject.IsPlatformOwner(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "INVALID_TOKEN")
		}

		if !isPlatformOwner {
			// Check if user is admin of the specified club
			canManage, err := subject.CanManageClub(ctx, clubID)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "INVALID_TOKEN")
			}
			if !canManage {
				return nil, status.Error(codes.PermissionDenied, "CLUB_ADMIN_OR_PLATFORM_OWNER_REQUIRED")
			}
		}
	} else {
		// If no club ID specified, only platform owners can create "orphaned" players
		// This prevents creating players without club membership
		isPlatformOwner, err := subject.IsPlatformOwner(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "INVALID_TOKEN")
		}
		if !isPlatformOwner {
			return nil, status.Error(codes.InvalidArgument, "CLUB_ID_REQUIRED_FOR_NON_PLATFORM_OWNERS")
		}
	}

	// Find similar players first (only in specified club if provided)
	sim, _ := s.Players.FindSimilar(ctx, in.DisplayName, in.GetInitialClubId())

	// Create the player
	p, err := s.Players.Create(ctx, in.DisplayName, in.GetInitialClubId())
	if err != nil {
		// Provide more specific error messages based on the actual error
		if strings.Contains(err.Error(), "email") && strings.Contains(err.Error(), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "EMAIL_ALREADY_EXISTS: "+err.Error())
		}
		if strings.Contains(err.Error(), "validation") {
			return nil, status.Error(codes.InvalidArgument, "PLAYER_VALIDATION_FAILED: "+err.Error())
		}
		if strings.Contains(err.Error(), "ObjectID") {
			return nil, status.Error(codes.InvalidArgument, "INVALID_CLUB_ID")
		}
		// For any other error, include the actual error message
		return nil, status.Error(codes.Internal, "PLAYER_CREATE_FAILED: "+err.Error())
	}

	resp := &pb.CreatePlayerResponse{
		Player: s.convertToProtobuf(p),
	}

	// Add similar players to response
	for _, x := range sim {
		resp.Similar = append(resp.Similar, s.convertToProtobuf(x))
	}

	return resp, nil
}

func (s *PlayerService) ListPlayers(ctx context.Context, in *pb.ListPlayersRequest) (*pb.ListPlayersResponse, error) {
	// Handle both old club_id and new club_filter for backward compatibility
	var clubIDs []string
	var includeOpen bool

	// Use new club_filter if provided
	if len(in.GetClubFilter()) > 0 {
		for _, filter := range in.GetClubFilter() {
			if filter == "OPEN" {
				includeOpen = true
			} else {
				clubIDs = append(clubIDs, filter)
			}
		}
	} else if in.GetClubId() != "" {
		// Fallback to old club_id for backward compatibility
		clubIDs = []string{in.GetClubId()}
	}

	// Use the new filtering method
	filters := repo.PlayerListFilters{
		SearchQuery: in.GetSearchQuery(),
		ClubIDs:     clubIDs,
		IncludeOpen: includeOpen,
	}

	players, startCursor, endCursor, hasNext, hasPrev, err := s.Players.ListWithCursorAndFilters(ctx, in.GetPageSize(), in.GetCursorAfter(), in.GetCursorBefore(), filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "PLAYER_LIST_FAILED")
	}

	var pbPlayers []*pb.Player
	for _, player := range players {
		pbPlayers = append(pbPlayers, s.convertToProtobuf(player))
	}

	return &pb.ListPlayersResponse{
		Items:           pbPlayers,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}, nil
}

func (s *PlayerService) GetPlayer(ctx context.Context, in *pb.GetPlayerRequest) (*pb.GetPlayerResponse, error) {
	player, err := s.Players.FindByID(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "PLAYER_NOT_FOUND")
	}

	return &pb.GetPlayerResponse{Player: s.convertToProtobuf(player)}, nil
}

func (s *PlayerService) UpdatePlayer(ctx context.Context, in *pb.UpdatePlayerRequest) (*pb.UpdatePlayerResponse, error) {
	updates := map[string]interface{}{}
	if mask := in.GetUpdateMask(); mask != nil && len(mask.GetPaths()) > 0 {
		for _, path := range mask.GetPaths() {
			switch path {
			case "display_name":
				updates["display_name"] = in.GetPlayer().GetDisplayName()
			case "active":
				updates["active"] = in.GetPlayer().GetActive()
			case "email":
				updates["email"] = in.GetPlayer().GetEmail()
			case "first_name":
				updates["first_name"] = in.GetPlayer().GetFirstName()
			case "last_name":
				updates["last_name"] = in.GetPlayer().GetLastName()
			}
		}
	} else {
		updates["display_name"] = in.GetPlayer().GetDisplayName()
		updates["active"] = in.GetPlayer().GetActive()
		updates["email"] = in.GetPlayer().GetEmail()
		updates["first_name"] = in.GetPlayer().GetFirstName()
		updates["last_name"] = in.GetPlayer().GetLastName()
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NO_FIELDS_TO_UPDATE")
	}

	player, err := s.Players.Update(ctx, in.GetId(), updates)
	if err != nil {
		if strings.Contains(err.Error(), "ObjectID") {
			return nil, status.Error(codes.InvalidArgument, "INVALID_PLAYER_ID")
		}
		return nil, status.Error(codes.Internal, "PLAYER_UPDATE_FAILED")
	}

	return &pb.UpdatePlayerResponse{Player: s.convertToProtobuf(player)}, nil
}

func (s *PlayerService) DeletePlayer(ctx context.Context, in *pb.DeletePlayerRequest) (*pb.DeletePlayerResponse, error) {
	if err := s.Players.Delete(ctx, in.GetId()); err != nil {
		if strings.Contains(err.Error(), "ObjectID") {
			return nil, status.Error(codes.InvalidArgument, "INVALID_PLAYER_ID")
		}
		return nil, status.Error(codes.Internal, "PLAYER_DELETE_FAILED")
	}

	return &pb.DeletePlayerResponse{Success: true}, nil
}

func (s *PlayerService) MergePlayer(ctx context.Context, in *pb.MergePlayerRequest) (*pb.MergePlayerResponse, error) {
	// Validate that the players are different
	if in.GetTargetPlayerId() == in.GetSourcePlayerId() {
		return nil, status.Error(codes.InvalidArgument, "CANNOT_MERGE_SAME_PLAYER")
	}

	// Get the authenticated user
	subject := GetSubjectFromContext(ctx)
	if subject.GetEmail() == "" {
		return nil, status.Error(codes.Unauthenticated, "USER_NOT_AUTHENTICATED")
	}

	// Get both players to validate merge restrictions
	targetPlayer, err := s.Players.FindByID(ctx, in.GetTargetPlayerId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "TARGET_PLAYER_NOT_FOUND")
	}

	sourcePlayer, err := s.Players.FindByID(ctx, in.GetSourcePlayerId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "SOURCE_PLAYER_NOT_FOUND")
	}

	// Check if user is platform owner
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_CHECK_PLATFORM_OWNER")
	}

	// AUTHORIZATION RULES:
	// 1. Platform owners can merge any players
	// 2. Regular users can only merge email-less players into their own account
	if !isPlatformOwner {
		// Get authenticated user's player record
		authenticatedPlayer, err := s.Players.FindByEmail(ctx, subject.GetEmail())
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_FIND_AUTHENTICATED_PLAYER")
		}

		// Target must be the authenticated user's own player
		if targetPlayer.ID.Hex() != authenticatedPlayer.ID.Hex() {
			return nil, status.Error(codes.PermissionDenied, "CAN_ONLY_MERGE_INTO_OWN_ACCOUNT")
		}

		// Source must be an email-less player (synthetic email)
		if !repo.IsSyntheticEmail(sourcePlayer.Email) {
			return nil, status.Error(codes.PermissionDenied, "CAN_ONLY_MERGE_EMAIL_LESS_PLAYERS")
		}
	}

	// Perform the merge
	mergedPlayer, matchesUpdated, tokensUpdated, err := s.Players.MergePlayer(ctx, in.GetTargetPlayerId(), in.GetSourcePlayerId())
	if err != nil {
		return nil, status.Error(codes.Internal, "MERGE_FAILED: "+err.Error())
	}

	return &pb.MergePlayerResponse{
		Player:         s.convertToProtobuf(mergedPlayer),
		MatchesUpdated: matchesUpdated,
		TokensUpdated:  tokensUpdated,
	}, nil
}

func (s *PlayerService) FindMergeCandidates(ctx context.Context, in *pb.FindMergeCandidatesRequest) (*pb.FindMergeCandidatesResponse, error) {
	// Get the authenticated user from the auth interceptor
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "SUBJECT_NOT_FOUND_IN_CONTEXT")
	}

	email := subject.GetEmail()
	if email == "" {
		return nil, status.Error(codes.Unauthenticated, "SUBJECT_EMAIL_EMPTY")
	}

	// Check if user is platform owner or club member/admin
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_CHECK_PLATFORM_OWNER")
	}

	if !isPlatformOwner {
		// Check if user is a member or admin of the specified club
		isMember, err := subject.IsClubMember(ctx, in.GetClubId())
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_CHECK_CLUB_MEMBERSHIP")
		}
		if !isMember {
			return nil, status.Error(codes.PermissionDenied, "MUST_BE_CLUB_MEMBER_TO_FIND_MERGE_CANDIDATES")
		}
	}

	// Get the authenticated user's player record
	authenticatedPlayer, err := s.Players.FindByEmail(ctx, email)
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_FIND_AUTHENTICATED_PLAYER")
	}

	// Get all email-less players in the club for similarity scoring
	allEmaillessPlayers, err := s.Players.FindAllEmaillessPlayersInClub(ctx, in.GetClubId())
	if err != nil {
		return nil, status.Error(codes.Internal, "FAILED_TO_FIND_EMAILLESS_PLAYERS")
	}

	// Score each candidate and create MergeCandidate objects
	type candidateWithScore struct {
		player *repo.Player
		score  float64
	}

	var scoredCandidates []candidateWithScore
	targetName := authenticatedPlayer.DisplayName

	for _, candidate := range allEmaillessPlayers {
		score := util.StringSimilarity(targetName, candidate.DisplayName)
		scoredCandidates = append(scoredCandidates, candidateWithScore{
			player: candidate,
			score:  score,
		})
	}

	// Sort by similarity score (highest first)
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].score > scoredCandidates[j].score
	})

	// Convert to protobuf MergeCandidate objects
	var pbCandidates []*pb.MergeCandidate
	for _, candidate := range scoredCandidates {
		pbCandidates = append(pbCandidates, &pb.MergeCandidate{
			Player:          s.convertToProtobuf(candidate.player),
			SimilarityScore: candidate.score,
		})
	}

	return &pb.FindMergeCandidatesResponse{
		Candidates: pbCandidates,
	}, nil
}
