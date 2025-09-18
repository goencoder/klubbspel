package service

import (
	"context"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MatchService struct {
	pb.UnimplementedMatchServiceServer
	Matches *repo.MatchRepo
	Players *repo.PlayerRepo
	Series  *repo.SeriesRepo
}

func (s *MatchService) ReportMatch(ctx context.Context, in *pb.ReportMatchRequest) (*pb.ReportMatchResponse, error) {
	// Basic validation
	if in.GetSeriesId() == "" || in.GetPlayerAId() == "" || in.GetPlayerBId() == "" {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_REQUIRED")
	}

	// No ties allowed
	if in.GetScoreA() == in.GetScoreB() {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
	}

	// Best of 5 validation (winner must reach 3)
	maxScore := in.GetScoreA()
	if in.GetScoreB() > maxScore {
		maxScore = in.GetScoreB()
	}
	if maxScore < 3 {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_FIVE")
	}

	// Players cannot be the same
	if in.GetPlayerAId() == in.GetPlayerBId() {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_SAME_PLAYER")
	}

	playedAt := in.GetPlayedAt().AsTime()

		// Validate that match date is within series time window
	series, err := s.Series.FindByID(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find series: %v", err)
	}

	// Convert match date to start of day and series dates to inclusive ranges
	matchDate := in.GetPlayedAt().AsTime()
	seriesStart := series.StartsAt.Truncate(24 * time.Hour) // Start of start date
	seriesEnd := series.EndsAt.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond) // End of end date

	if matchDate.Before(seriesStart) || matchDate.After(seriesEnd) {
		return nil, status.Errorf(codes.InvalidArgument, 
			"match date %v must be between series start date %v and end date %v (inclusive)", 
			matchDate.Format("2006-01-02"), 
			series.StartsAt.Format("2006-01-02"), 
			series.EndsAt.Format("2006-01-02"))
	}

	// TODO: Add ELO rating calculation

	match, err := s.Matches.Create(ctx, in.GetSeriesId(), in.GetPlayerAId(), in.GetPlayerBId(), in.GetScoreA(), in.GetScoreB(), playedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_CREATE_FAILED")
	}

	return &pb.ReportMatchResponse{
		MatchId: match.ID.Hex(),
	}, nil
}

func (s *MatchService) ListMatches(ctx context.Context, in *pb.ListMatchesRequest) (*pb.ListMatchesResponse, error) {
	matches, nextPageToken, err := s.Matches.ListBySeriesID(ctx, in.GetSeriesId(), in.GetPageSize(), in.GetCursorAfter())
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_LIST_FAILED")
	}

	var pbMatches []*pb.MatchView
	for _, match := range matches {
		pbMatches = append(pbMatches, &pb.MatchView{
			Id:          match.ID,
			SeriesId:    match.SeriesID,
			PlayerAName: match.PlayerAName,
			PlayerBName: match.PlayerBName,
			ScoreA:      match.ScoreA,
			ScoreB:      match.ScoreB,
			PlayedAt:    timestamppb.New(match.PlayedAt),
		})
	}

	// Set pagination info
	var startCursor, endCursor string
	hasNext := nextPageToken != ""
	hasPrev := in.GetCursorAfter() != ""

	if len(pbMatches) > 0 {
		startCursor = pbMatches[0].Id
		endCursor = pbMatches[len(pbMatches)-1].Id
	}

	return &pb.ListMatchesResponse{
		Items:           pbMatches,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}, nil
}

func (s *MatchService) UpdateMatch(ctx context.Context, in *pb.UpdateMatchRequest) (*pb.UpdateMatchResponse, error) {
	// Basic validation
	if in.GetMatchId() == "" {
		return nil, status.Error(codes.InvalidArgument, "MATCH_ID_REQUIRED")
	}

	// Get existing match to check series and for validation
	existingMatch, err := s.Matches.FindByID(ctx, in.GetMatchId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "MATCH_NOT_FOUND")
	}

	// Extract optional fields
	var scoreA, scoreB *int32
	var playedAt *time.Time

	if in.ScoreA != nil {
		scoreA = in.ScoreA
	}
	if in.ScoreB != nil {
		scoreB = in.ScoreB
	}
	if in.PlayedAt != nil {
		t := in.GetPlayedAt().AsTime()
		playedAt = &t
		
		// Validate that updated match date is within series time window
		series, err := s.Series.FindByID(ctx, existingMatch.SeriesID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "SERIES_NOT_FOUND")
		}

		// Convert series dates to inclusive ranges
		seriesStart := series.StartsAt.Truncate(24 * time.Hour) // Start of start date
		seriesEnd := series.EndsAt.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond) // End of end date

		if t.Before(seriesStart) || t.After(seriesEnd) {
			return nil, status.Errorf(codes.InvalidArgument, 
				"match date %v must be between series start date %v and end date %v (inclusive)", 
				t.Format("2006-01-02"), 
				series.StartsAt.Format("2006-01-02"), 
				series.EndsAt.Format("2006-01-02"))
		}
	}

	// Validate scores if both are provided
	if scoreA != nil && scoreB != nil {
		// No ties allowed
		if *scoreA == *scoreB {
			return nil, status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
		}

		// Best of 5 validation (winner must reach 3)
		maxScore := *scoreA
		if *scoreB > maxScore {
			maxScore = *scoreB
		}
		if maxScore < 3 {
			return nil, status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_FIVE")
		}
	}

	// Update the match
	updatedMatch, err := s.Matches.Update(ctx, in.GetMatchId(), scoreA, scoreB, playedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_UPDATE_FAILED")
	}

	// Get player names for response
	players, err := s.Players.FindByIDs(ctx, []string{updatedMatch.PlayerAID, updatedMatch.PlayerBID})
	if err != nil {
		return nil, status.Error(codes.Internal, "PLAYER_LOOKUP_FAILED")
	}

	playerAName := "Unknown Player"
	playerBName := "Unknown Player"
	if playerA, exists := players[updatedMatch.PlayerAID]; exists {
		playerAName = playerA.DisplayName
	}
	if playerB, exists := players[updatedMatch.PlayerBID]; exists {
		playerBName = playerB.DisplayName
	}

	return &pb.UpdateMatchResponse{
		Match: &pb.MatchView{
			Id:          updatedMatch.ID.Hex(),
			SeriesId:    updatedMatch.SeriesID,
			PlayerAName: playerAName,
			PlayerBName: playerBName,
			ScoreA:      updatedMatch.ScoreA,
			ScoreB:      updatedMatch.ScoreB,
			PlayedAt:    timestamppb.New(updatedMatch.PlayedAt),
		},
	}, nil
}

func (s *MatchService) DeleteMatch(ctx context.Context, in *pb.DeleteMatchRequest) (*pb.DeleteMatchResponse, error) {
	// Basic validation
	if in.GetMatchId() == "" {
		return nil, status.Error(codes.InvalidArgument, "MATCH_ID_REQUIRED")
	}

	// Delete the match
	err := s.Matches.Delete(ctx, in.GetMatchId())
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_DELETE_FAILED")
	}

	return &pb.DeleteMatchResponse{
		Success: true,
	}, nil
}

func (s *MatchService) ReorderMatches(ctx context.Context, in *pb.ReorderMatchesRequest) (*pb.ReorderMatchesResponse, error) {
	// Basic validation
	if len(in.GetMatchIds()) < 2 {
		return nil, status.Error(codes.InvalidArgument, "AT_LEAST_TWO_MATCHES_REQUIRED")
	}

	// Reorder the matches
	err := s.Matches.ReorderMatches(ctx, in.GetMatchIds())
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_REORDER_FAILED")
	}

	return &pb.ReorderMatchesResponse{
		Success: true,
	}, nil
}
