package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClubService struct {
	pb.UnimplementedClubServiceServer
	Clubs   *repo.ClubRepo
	Players *repo.PlayerRepo
	Series  *repo.SeriesRepo
}

func (s *ClubService) CreateClub(ctx context.Context, in *pb.CreateClubRequest) (*pb.CreateClubResponse, error) {
	// Check authentication
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate club name
	if in.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_REQUIRED")
	}

	// Check if user has completed their profile (first and last name required)
	if subject.GetFirstName() == "" || subject.GetLastName() == "" {
		return nil, status.Error(codes.FailedPrecondition, "PROFILE_COMPLETION_REQUIRED")
	}

	sports, err := normalizeClubSports(in.GetSupportedSports())
	if err != nil {
		return nil, err
	}

	// Create the club
	club, err := s.Clubs.Upsert(ctx, in.GetName(), sports)
	if err != nil {
		return nil, status.Error(codes.Internal, "CLUB_CREATE_FAILED")
	}

	// Add the creator as a club admin member
	membership := &repo.ClubMembership{
		ClubID:   club.ID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}

	err = s.Players.AddClubMembership(ctx, subject.GetEmail(), membership)
	if err != nil {
		// Log error but don't fail the club creation
		fmt.Printf("Warning: Failed to add club creator as member: %v\n", err)
	}

	return &pb.CreateClubResponse{
		Club: &pb.Club{
			Id:              club.ID.Hex(),
			Name:            club.Name,
			SupportedSports: pbSupportedSports(club.SupportedSports),
		},
	}, nil
}

func (s *ClubService) GetClub(ctx context.Context, in *pb.GetClubRequest) (*pb.GetClubResponse, error) {
	club, err := s.Clubs.GetByID(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	return &pb.GetClubResponse{
		Club: &pb.Club{
			Id:              club.ID.Hex(),
			Name:            club.Name,
			SupportedSports: pbSupportedSports(club.SupportedSports),
		},
	}, nil
}

func (s *ClubService) UpdateClub(ctx context.Context, in *pb.UpdateClubRequest) (*pb.UpdateClubResponse, error) {
	// Validate the request
	if in.GetClub() == nil {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_REQUIRED")
	}

	updates := map[string]interface{}{}
	if mask := in.GetUpdateMask(); mask != nil && len(mask.GetPaths()) > 0 {
		for _, path := range mask.GetPaths() {
			switch path {
			case "name":
				if in.GetClub().GetName() == "" {
					return nil, status.Error(codes.InvalidArgument, "CLUB_NAME_REQUIRED")
				}
				updates["name"] = in.GetClub().GetName()
			case "supported_sports":
				sports, err := normalizeClubSports(in.GetClub().GetSupportedSports())
				if err != nil {
					return nil, err
				}
				updates["supported_sports"] = sports
			default:
				return nil, status.Error(codes.InvalidArgument, "UNSUPPORTED_UPDATE_FIELD")
			}
		}
	} else {
		if in.GetClub().GetName() != "" {
			updates["name"] = in.GetClub().GetName()
		}
		sports, err := normalizeClubSports(in.GetClub().GetSupportedSports())
		if err != nil {
			return nil, err
		}
		updates["supported_sports"] = sports
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NO_FIELDS_TO_UPDATE")
	}

	club, err := s.Clubs.Update(ctx, in.GetId(), updates)
	if err != nil {
		return nil, status.Error(codes.Internal, "CLUB_UPDATE_FAILED")
	}

	return &pb.UpdateClubResponse{
		Club: &pb.Club{
			Id:              club.ID.Hex(),
			Name:            club.Name,
			SupportedSports: pbSupportedSports(club.SupportedSports),
		},
	}, nil
}

func (s *ClubService) DeleteClub(ctx context.Context, in *pb.DeleteClubRequest) (*pb.DeleteClubResponse, error) {
	// Check authentication
	subject := GetSubjectFromContext(ctx)
	if subject == nil {
		return nil, status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
	}

	// Validate club ID
	if in.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "CLUB_ID_REQUIRED")
	}

	// Check if club exists
	_, err := s.Clubs.GetByID(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "CLUB_NOT_FOUND")
	}

	// Check authorization: only club admins or platform owners can delete clubs
	isPlatformOwner, err := subject.IsPlatformOwner(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "PLATFORM_OWNER_CHECK_FAILED")
	}

	if !isPlatformOwner {
		isClubAdmin, err := subject.IsClubAdmin(ctx, in.GetId())
		if err != nil {
			return nil, status.Error(codes.Internal, "CLUB_ADMIN_CHECK_FAILED")
		}
		if !isClubAdmin {
			return nil, status.Error(codes.PermissionDenied, "ONLY_CLUB_ADMIN_CAN_DELETE")
		}
	}

	// Delete all memberships for this club
	err = s.Players.RemoveAllClubMemberships(ctx, in.GetId())
	if err != nil {
		// Log error but don't fail the club deletion
		fmt.Printf("Warning: Failed to remove club memberships: %v\n", err)
	}

	// Delete all club-specific series for this club
	err = s.Series.DeleteByClubID(ctx, in.GetId())
	if err != nil {
		// Log error but don't fail the club deletion
		fmt.Printf("Warning: Failed to delete club series: %v\n", err)
	}

	err = s.Clubs.Delete(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, "CLUB_DELETE_FAILED")
	}

	return &pb.DeleteClubResponse{
		Success: true,
	}, nil
}

func (s *ClubService) ListClubs(ctx context.Context, in *pb.ListClubsRequest) (*pb.ListClubsResponse, error) {
	// DEBUG: Log the request parameters
	fmt.Printf("DEBUG ClubService.ListClubs: pageSize=%d, searchQuery='%s', cursorAfter='%s', cursorBefore='%s'\n",
		in.GetPageSize(), in.GetSearchQuery(), in.GetCursorAfter(), in.GetCursorBefore())

	// Defensive page size handling
	pageSize := in.GetPageSize()
	if pageSize <= 0 {
		fmt.Printf("DEBUG ClubService.ListClubs: Invalid pageSize %d, setting to 20\n", pageSize)
		pageSize = 20
	}
	if pageSize > 100 {
		fmt.Printf("DEBUG ClubService.ListClubs: PageSize %d too large, setting to 100\n", pageSize)
		pageSize = 100
	}

	fmt.Printf("DEBUG ClubService.ListClubs: Final pageSize=%d\n", pageSize)
	clubs, startCursor, endCursor, hasNext, hasPrev, err := s.Clubs.ListWithCursor(ctx, in.GetSearchQuery(), pageSize, in.GetCursorAfter(), in.GetCursorBefore())
	if err != nil {
		return nil, status.Error(codes.Internal, "CLUB_LIST_FAILED")
	}

	var pbClubs []*pb.Club
	for _, club := range clubs {
		pbClubs = append(pbClubs, &pb.Club{
			Id:              club.ID.Hex(),
			Name:            club.Name,
			SupportedSports: pbSupportedSports(club.SupportedSports),
		})
	}

	return &pb.ListClubsResponse{
		Items:           pbClubs,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}, nil
}

func normalizeClubSports(input []pb.Sport) ([]int32, error) {
	seen := map[int32]struct{}{}
	var sports []int32

	for _, sport := range input {
		if sport == pb.Sport_SPORT_UNSPECIFIED {
			continue
		}

		if sport != pb.Sport_SPORT_TABLE_TENNIS {
			return nil, status.Error(codes.Unimplemented, "SPORT_NOT_SUPPORTED")
		}

		key := int32(sport)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		sports = append(sports, key)
	}

	if len(sports) == 0 {
		sports = []int32{int32(pb.Sport_SPORT_TABLE_TENNIS)}
	}

	sort.Slice(sports, func(i, j int) bool { return sports[i] < sports[j] })
	return sports, nil
}

func pbSupportedSports(values []int32) []pb.Sport {
	if len(values) == 0 {
		return []pb.Sport{pb.Sport_SPORT_TABLE_TENNIS}
	}

	seen := map[pb.Sport]struct{}{}
	var sports []pb.Sport

	for _, value := range values {
		sport := pb.Sport(value)
		if sport == pb.Sport_SPORT_UNSPECIFIED {
			sport = pb.Sport_SPORT_TABLE_TENNIS
		}

		if _, ok := seen[sport]; ok {
			continue
		}
		seen[sport] = struct{}{}
		sports = append(sports, sport)
	}

	sort.Slice(sports, func(i, j int) bool { return sports[i] < sports[j] })
	return sports
}
