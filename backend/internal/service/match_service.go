package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MatchService struct {
	pb.UnimplementedMatchServiceServer
	Matches      *repo.MatchRepo
	Players      *repo.PlayerRepo
	Series       *repo.SeriesRepo
	Leaderboard  *repo.LeaderboardRepo
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

	// Recalculate and store leaderboard
	if err := s.RecalculateStandings(ctx, in.GetSeriesId()); err != nil {
		log.Error().Err(err).Str("seriesID", in.GetSeriesId()).Msg("Failed to recalculate standings")
		// Don't fail the match creation, just log the error
	}

	return &pb.ReportMatchResponse{
		MatchId: match.ID.Hex(),
	}, nil
}

// RecalculateStandings recalculates and stores the leaderboard for a series
func (s *MatchService) RecalculateStandings(ctx context.Context, seriesID string) error {
	// Fetch series to determine format
	series, err := s.Series.FindByID(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to fetch series: %w", err)
	}

	// Get all matches in chronological order
	matches, err := s.Matches.FindAllBySeriesChronological(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to fetch matches: %w", err)
	}

	// Clear existing leaderboard
	if err := s.Leaderboard.DeleteAllForSeries(ctx, seriesID); err != nil {
		return fmt.Errorf("failed to clear leaderboard: %w", err)
	}

	if len(matches) == 0 {
		return nil // No matches, nothing to calculate
	}

	now := time.Now()
	format := pb.SeriesFormat(series.Format)

	if format == pb.SeriesFormat_SERIES_FORMAT_LADDER {
		// For ladder series, calculate positions based on ladder rules
		return s.recalculateLadderStandings(ctx, seriesID, series.LadderRules, matches, now)
	}

	// For open series, calculate ELO ratings
	return s.recalculateEloStandings(ctx, seriesID, matches, now)
}

// recalculate EloStandings calculates ELO ratings for all players and stores in leaderboard
func (s *MatchService) recalculateEloStandings(ctx context.Context, seriesID string, matches []*repo.Match, now time.Time) error {
	// Calculate ELO ratings from matches
	eloRatings := make(map[string]int32)
	matchStats := make(map[string]*struct {
		played int32
		won    int32
		lost   int32
		gamesWon   int32
		gamesLost  int32
	})

	// Initialize all players at 1000 ELO
	for _, match := range matches {
		if _, exists := eloRatings[match.PlayerAID]; !exists {
			eloRatings[match.PlayerAID] = 1000
			matchStats[match.PlayerAID] = &struct {
				played int32
				won    int32
				lost   int32
				gamesWon   int32
				gamesLost  int32
			}{}
		}
		if _, exists := eloRatings[match.PlayerBID]; !exists {
			eloRatings[match.PlayerBID] = 1000
			matchStats[match.PlayerBID] = &struct {
				played int32
				won    int32
				lost   int32
				gamesWon   int32
				gamesLost  int32
			}{}
		}

		// Update match statistics
		matchStats[match.PlayerAID].played++
		matchStats[match.PlayerBID].played++
		matchStats[match.PlayerAID].gamesWon += match.ScoreA
		matchStats[match.PlayerAID].gamesLost += match.ScoreB
		matchStats[match.PlayerBID].gamesWon += match.ScoreB
		matchStats[match.PlayerBID].gamesLost += match.ScoreA

		// Skip ties
		if match.ScoreA == match.ScoreB {
			continue
		}

		// Update ELO
		ratingA := float64(eloRatings[match.PlayerAID])
		ratingB := float64(eloRatings[match.PlayerBID])
		
		newRatingA, newRatingB := calculateELO(ratingA, ratingB, match.ScoreA, match.ScoreB)
		
		eloRatings[match.PlayerAID] = int32(newRatingA)
		eloRatings[match.PlayerBID] = int32(newRatingB)

		// Update win/loss
		if match.ScoreA > match.ScoreB {
			matchStats[match.PlayerAID].won++
			matchStats[match.PlayerBID].lost++
		} else {
			matchStats[match.PlayerBID].won++
			matchStats[match.PlayerAID].lost++
		}
	}

	// Convert to slice and sort by rating
	type playerRating struct {
		playerID string
		rating   int32
		stats    *struct {
			played int32
			won    int32
			lost   int32
			gamesWon   int32
			gamesLost  int32
		}
	}
	
	var ratings []playerRating
	for playerID, rating := range eloRatings {
		ratings = append(ratings, playerRating{
			playerID: playerID,
			rating:   rating,
			stats:    matchStats[playerID],
		})
	}

	// Sort by rating (highest first)
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].rating > ratings[j].rating
	})

	// Store in leaderboard with ranks
	for rank, pr := range ratings {
		entry := &repo.LeaderboardEntry{
			SeriesID:      seriesID,
			PlayerID:      pr.playerID,
			Rank:          int32(rank + 1),
			Rating:        pr.rating,
			MatchesPlayed: pr.stats.played,
			MatchesWon:    pr.stats.won,
			MatchesLost:   pr.stats.lost,
			GamesWon:      pr.stats.gamesWon,
			GamesLost:     pr.stats.gamesLost,
			UpdatedAt:     now,
		}

		if err := s.Leaderboard.UpsertEntry(ctx, entry); err != nil {
			return fmt.Errorf("failed to upsert leaderboard entry for player %s: %w", pr.playerID, err)
		}
	}

	return nil
}

// recalculateLadderStandings calculates ladder positions and stores in leaderboard
func (s *MatchService) recalculateLadderStandings(ctx context.Context, seriesID string, ladderRulesValue int32, matches []*repo.Match, now time.Time) error {
	// Track positions: playerID -> position
	positions := make(map[string]int32)
	nextPosition := int32(1)

	ladderRules := pb.LadderRules(ladderRulesValue)

	// Track match statistics
	matchStats := make(map[string]*struct {
		played int32
		won    int32
		lost   int32
		gamesWon   int32
		gamesLost  int32
	})

	for _, match := range matches {
		// Skip ties
		if match.ScoreA == match.ScoreB {
			continue
		}

		winnerID := match.PlayerAID
		loserID := match.PlayerBID
		if match.ScoreB > match.ScoreA {
			winnerID = match.PlayerBID
			loserID = match.PlayerAID
		}

		// Ensure players have positions
		if _, exists := positions[winnerID]; !exists {
			positions[winnerID] = nextPosition
			nextPosition++
		}
		if _, exists := positions[loserID]; !exists {
			positions[loserID] = nextPosition
			nextPosition++
		}

		// Initialize stats if needed
		if matchStats[match.PlayerAID] == nil {
			matchStats[match.PlayerAID] = &struct {
				played int32
				won    int32
				lost   int32
				gamesWon   int32
				gamesLost  int32
			}{}
		}
		if matchStats[match.PlayerBID] == nil {
			matchStats[match.PlayerBID] = &struct {
				played int32
				won    int32
				lost   int32
				gamesWon   int32
				gamesLost  int32
			}{}
		}

		// Update stats
		matchStats[match.PlayerAID].played++
		matchStats[match.PlayerBID].played++
		matchStats[match.PlayerAID].gamesWon += match.ScoreA
		matchStats[match.PlayerAID].gamesLost += match.ScoreB
		matchStats[match.PlayerBID].gamesWon += match.ScoreB
		matchStats[match.PlayerBID].gamesLost += match.ScoreA

		if match.ScoreA > match.ScoreB {
			matchStats[match.PlayerAID].won++
			matchStats[match.PlayerBID].lost++
		} else {
			matchStats[match.PlayerBID].won++
			matchStats[match.PlayerAID].lost++
		}

		winnerPos := positions[winnerID]
		loserPos := positions[loserID]

		// Apply ladder climbing rules
		if winnerPos > loserPos {
			// Lower-ranked player beats higher-ranked player - winner climbs
			targetPos := loserPos

			// Shift everyone between targetPos and winnerPos down by 1
			for pid, pos := range positions {
				if pos >= targetPos && pos < winnerPos && pid != winnerID {
					positions[pid] = pos + 1
				}
			}

			positions[winnerID] = targetPos
		} else {
			// Higher-ranked player wins - apply penalty rules
			if ladderRules == pb.LadderRules_LADDER_RULES_AGGRESSIVE {
				// Loser drops one position (swap with player below)
				belowPos := loserPos + 1

				// Find player at belowPos and swap
				for pid, pos := range positions {
					if pos == belowPos {
						positions[pid] = loserPos
						positions[loserID] = belowPos
						break
					}
				}
			}
			// Classic rules: no penalty
		}
	}

	// Store in leaderboard
	for playerID, position := range positions {
		stats := matchStats[playerID]
		if stats == nil {
			stats = &struct {
				played int32
				won    int32
				lost   int32
				gamesWon   int32
				gamesLost  int32
			}{}
		}

		entry := &repo.LeaderboardEntry{
			SeriesID:      seriesID,
			PlayerID:      playerID,
			Rank:          position,
			Rating:        position, // For ladder, rating IS the position
			MatchesPlayed: stats.played,
			MatchesWon:    stats.won,
			MatchesLost:   stats.lost,
			GamesWon:      stats.gamesWon,
			GamesLost:     stats.gamesLost,
			UpdatedAt:     now,
		}

		if err := s.Leaderboard.UpsertEntry(ctx, entry); err != nil {
			return fmt.Errorf("failed to upsert leaderboard entry for player %s: %w", playerID, err)
		}
	}

	return nil
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
	case int32(pb.Sport_SPORT_TABLE_TENNIS),
		int32(pb.Sport_SPORT_TENNIS),
		int32(pb.Sport_SPORT_PADEL),
		int32(pb.Sport_SPORT_BADMINTON),
		int32(pb.Sport_SPORT_SQUASH),
		int32(pb.Sport_SPORT_PICKLEBALL):
		// All racket/paddle sports use TABLE_TENNIS_SETS scoring
		// They share the same best-of-N sets format
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

	// Recalculate and store leaderboard
	if err := s.RecalculateStandings(ctx, in.GetSeriesId()); err != nil {
		log.Error().Err(err).Str("seriesID", in.GetSeriesId()).Msg("Failed to recalculate standings")
		// Don't fail the match creation, just log the error
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

	// Recalculate and store leaderboard
	if err := s.RecalculateStandings(ctx, updatedMatch.SeriesID); err != nil {
		log.Error().Err(err).Str("seriesID", updatedMatch.SeriesID).Msg("Failed to recalculate standings")
		// Don't fail the update, just log the error
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

	// Get the match first to know which series to recalculate
	match, err := s.Matches.FindByID(ctx, in.GetMatchId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "MATCH_NOT_FOUND")
	}

	// Delete the match
	err = s.Matches.Delete(ctx, in.GetMatchId())
	if err != nil {
		return nil, status.Error(codes.Internal, "MATCH_DELETE_FAILED")
	}

	// Recalculate and store leaderboard
	if err := s.RecalculateStandings(ctx, match.SeriesID); err != nil {
		log.Error().Err(err).Str("seriesID", match.SeriesID).Msg("Failed to recalculate standings")
		// Don't fail the delete, just log the error
	}

	return &pb.DeleteMatchResponse{
		Success: true,
	}, nil
}

// calculateELO computes new ELO ratings for two players based on match result
func calculateELO(ratingA, ratingB float64, scoreA, scoreB int32) (newRatingA, newRatingB float64) {
	const K = 32.0

	// Calculate expected scores
	expectedA := 1 / (1 + math.Pow(10, (ratingB-ratingA)/400))
	expectedB := 1 / (1 + math.Pow(10, (ratingA-ratingB)/400))

	// Determine actual scores
	var actualA, actualB float64
	if scoreA > scoreB {
		actualA = 1
		actualB = 0
	} else {
		actualA = 0
		actualB = 1
	}

	// Calculate new ratings
	newRatingA = ratingA + K*(actualA-expectedA)
	newRatingB = ratingB + K*(actualB-expectedB)

	return newRatingA, newRatingB
}

