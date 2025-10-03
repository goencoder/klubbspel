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

	// Players cannot be the same
	if in.GetPlayerAId() == in.GetPlayerBId() {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_SAME_PLAYER")
	}

	// Validate table tennis scores using new helper
	if err := validateTableTennisScore(in.GetScoreA(), in.GetScoreB(), 5); err != nil {
		return nil, err
	}

	playedAt := in.GetPlayedAt().AsTime()

	// Validate that match date is within series time window
	series, err := s.Series.FindByID(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find series: %v", err)
	}

	// Use common time validation helper
	if err := validateMatchTimeWindow(playedAt, series.StartsAt, series.EndsAt); err != nil {
		return nil, err
	}

	// Create the match record
	match, err := s.Matches.Create(ctx, in.GetSeriesId(), in.GetPlayerAId(), in.GetPlayerBId(), in.GetScoreA(), in.GetScoreB(), playedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_CREATE_FAILED")
	}

	return &pb.ReportMatchResponse{
		MatchId: match.ID.Hex(),
	}, nil
}

// validateTableTennisScore validates table tennis scoring rules
func validateTableTennisScore(setsA, setsB, setsToPlay int32) error {
	// No ties allowed
	if setsA == setsB {
		return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
	}

	// Calculate required wins to win the match
	requiredSets := (setsToPlay + 1) / 2

	// Check if either player has reached the required sets to win
	if setsA < requiredSets && setsB < requiredSets {
		if setsToPlay == 3 {
			return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_THREE")
		}
		return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_FIVE")
	}

	// Check that scores don't exceed what's possible
	if setsA > requiredSets || setsB > requiredSets {
		return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
	}

	// Check that the loser didn't get too many sets
	// In a valid match, the loser can have at most requiredSets-1 sets
	if setsA == requiredSets && setsB >= requiredSets {
		return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
	}
	if setsB == requiredSets && setsA >= requiredSets {
		return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
	}

	return nil
}

// validateMatchTimeWindow validates that a match was played within series bounds
func validateMatchTimeWindow(matchTime, seriesStart, seriesEnd time.Time) error {
	// Convert to inclusive date ranges
	seriesStartDate := seriesStart.Truncate(24 * time.Hour)
	seriesEndDate := seriesEnd.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)

	if matchTime.Before(seriesStartDate) || matchTime.After(seriesEndDate) {
		return status.Errorf(codes.InvalidArgument,
			"match date %v must be between series start date %v and end date %v (inclusive)",
			matchTime.Format("2006-01-02"),
			seriesStart.Format("2006-01-02"),
			seriesEnd.Format("2006-01-02"))
	}
	return nil
}

func (s *MatchService) ReportMatchV2(ctx context.Context, in *pb.ReportMatchV2Request) (*pb.ReportMatchV2Response, error) {
	// Basic validation
	if in.GetSeriesId() == "" {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_SERIES_ID_REQUIRED")
	}

	if in.GetParticipantA() == nil || in.GetParticipantB() == nil {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_PARTICIPANTS_REQUIRED")
	}

	if in.GetResult() == nil {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_RESULT_REQUIRED")
	}

	// Get series to check scoring profile and validation rules
	series, err := s.Series.FindByID(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find series: %v", err)
	}

	// Extract participant IDs (only support individual players for now)
	var playerAId, playerBId string

	if pA := in.GetParticipantA().GetPlayerId(); pA != "" {
		playerAId = pA
	} else {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_ONLY_INDIVIDUAL_PLAYERS_SUPPORTED")
	}

	if pB := in.GetParticipantB().GetPlayerId(); pB != "" {
		playerBId = pB
	} else {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_ONLY_INDIVIDUAL_PLAYERS_SUPPORTED")
	}

	// Players cannot be the same
	if playerAId == playerBId {
		return nil, status.Error(codes.InvalidArgument, "VALIDATION_SAME_PLAYER")
	}

	// Validate result according to series scoring profile
	var scoreA, scoreB int32

	switch series.Sport {
	case int32(pb.Sport_SPORT_TABLE_TENNIS), int32(pb.Sport_SPORT_TENNIS):
		// For table tennis and tennis, use TABLE_TENNIS_SETS scoring
		// Both sports use the same best-of-N sets format
		ttResult := in.GetResult().GetTableTennis()
		if ttResult == nil {
			return nil, status.Error(codes.InvalidArgument, "VALIDATION_TABLE_TENNIS_RESULT_REQUIRED")
		}

		scoreA = ttResult.GetSetsA()
		scoreB = ttResult.GetSetsB()

		// Determine sets to play (default to 5 if not set)
		setsToPlay := series.SetsToPlay
		if setsToPlay == 0 {
			setsToPlay = 5
		}

		// Validate table tennis scores
		if err := validateTableTennisScore(scoreA, scoreB, setsToPlay); err != nil {
			return nil, err
		}

	default:
		return nil, status.Error(codes.Unimplemented, "SPORT_NOT_SUPPORTED")
	}

	playedAt := in.GetPlayedAt().AsTime()

	// Validate match time window
	if err := validateMatchTimeWindow(playedAt, series.StartsAt, series.EndsAt); err != nil {
		return nil, err
	}

	// Create match using existing repository method
	match, err := s.Matches.Create(ctx, in.GetSeriesId(), playerAId, playerBId, scoreA, scoreB, playedAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_CREATE_FAILED")
	}

	return &pb.ReportMatchV2Response{
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
		seriesStart := series.StartsAt.Truncate(24 * time.Hour)                                 // Start of start date
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
