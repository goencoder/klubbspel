package service

import (
	"context"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LeaderboardService struct {
	pb.UnimplementedLeaderboardServiceServer
	Leaderboard *repo.LeaderboardRepo
	Players     *repo.PlayerRepo
	Matches     *MatchService // For fallback recalculation
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, in *pb.GetLeaderboardRequest) (*pb.GetLeaderboardResponse, error) {
	log.Info().Str("seriesId", in.GetSeriesId()).Msg("GetLeaderboard called")

	// Read from pre-calculated leaderboard
	leaderboardEntries, err := s.Leaderboard.FindBySeriesOrdered(ctx, in.GetSeriesId())
	if err != nil {
		log.Error().Str("seriesId", in.GetSeriesId()).Err(err).Msg("Failed to get leaderboard")
		return nil, status.Error(codes.Internal, "LEADERBOARD_FETCH_FAILED")
	}

	if len(leaderboardEntries) == 0 {
		// Fallback: Trigger recalculation if no leaderboard exists yet
		if s.Matches != nil {
			log.Info().Str("seriesId", in.GetSeriesId()).Msg("Leaderboard empty, triggering recalculation")
			if err := s.Matches.RecalculateStandings(ctx, in.GetSeriesId()); err != nil {
				log.Error().Str("seriesId", in.GetSeriesId()).Err(err).Msg("Fallback recalculation failed")
				// Don't fail the request, just return empty leaderboard
			} else {
				// Recalculation succeeded, try fetching again
				leaderboardEntries, err = s.Leaderboard.FindBySeriesOrdered(ctx, in.GetSeriesId())
				if err != nil {
					log.Error().Str("seriesId", in.GetSeriesId()).Err(err).Msg("Failed to fetch after recalculation")
					return nil, status.Error(codes.Internal, "LEADERBOARD_FETCH_FAILED")
				}
				// If still empty after recalculation, series has no matches
				if len(leaderboardEntries) == 0 {
					return &pb.GetLeaderboardResponse{
						Entries:         []*pb.LeaderboardEntry{},
						StartCursor:     "",
						EndCursor:       "",
						HasNextPage:     false,
						HasPreviousPage: false,
						TotalPlayers:    0,
					}, nil
				}
				// Continue with populated leaderboard
			}
		} else {
			// No MatchService available, return empty
			return &pb.GetLeaderboardResponse{
				Entries:         []*pb.LeaderboardEntry{},
				StartCursor:     "",
				EndCursor:       "",
				HasNextPage:     false,
				HasPreviousPage: false,
				TotalPlayers:    0,
			}, nil
		}
	}

	// Collect player IDs for name lookup
	playerIDs := make([]string, len(leaderboardEntries))
	for i, entry := range leaderboardEntries {
		playerIDs[i] = entry.PlayerID
	}

	// Batch fetch player names
	playersMap, err := s.Players.FindByIDs(ctx, playerIDs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch players for leaderboard")
		return nil, status.Error(codes.Internal, "LEADERBOARD_PLAYERS_FETCH_FAILED")
	}

	// Build response entries
	var entries []*pb.LeaderboardEntry
	for _, entry := range leaderboardEntries {
		player, exists := playersMap[entry.PlayerID]
		playerName := "Unknown Player"
		if exists {
			playerName = player.DisplayName
		}

		pbEntry := &pb.LeaderboardEntry{
			PlayerId:      entry.PlayerID,
			PlayerName:    playerName,
			Rank:          entry.Rank,
			EloRating:     entry.Rating,
			MatchesPlayed: entry.MatchesPlayed,
			MatchesWon:    entry.MatchesWon,
			MatchesLost:   entry.MatchesLost,
			GamesWon:      entry.GamesWon,
			GamesLost:     entry.GamesLost,
		}

		// Calculate win rates
		if pbEntry.MatchesPlayed > 0 {
			pbEntry.WinRate = float32(pbEntry.MatchesWon) / float32(pbEntry.MatchesPlayed) * 100
		}
		if pbEntry.GamesWon+pbEntry.GamesLost > 0 {
			pbEntry.GameWinRate = float32(pbEntry.GamesWon) / float32(pbEntry.GamesWon+pbEntry.GamesLost) * 100
		}

		entries = append(entries, pbEntry)
	}

	// Handle pagination
	pageSize := in.GetPageSize()
	if pageSize == 0 {
		pageSize = 20
	}

	totalPlayers := int32(len(entries))
	startIdx := 0
	endIdx := len(entries)

	// Apply cursor pagination
	if cursorAfter := in.GetCursorAfter(); cursorAfter != "" {
		for i, entry := range entries {
			if entry.PlayerId == cursorAfter {
				startIdx = i + 1
				break
			}
		}
	}
	if cursorBefore := in.GetCursorBefore(); cursorBefore != "" {
		for i, entry := range entries {
			if entry.PlayerId == cursorBefore {
				endIdx = i
				break
			}
		}
	}

	if endIdx-startIdx > int(pageSize) {
		endIdx = startIdx + int(pageSize)
	}

	if startIdx >= len(entries) {
		entries = []*pb.LeaderboardEntry{}
	} else if endIdx > len(entries) {
		entries = entries[startIdx:]
	} else {
		entries = entries[startIdx:endIdx]
	}

	var startCursor, endCursor string
	hasNext := endIdx < int(totalPlayers)
	hasPrev := startIdx > 0

	if len(entries) > 0 {
		startCursor = entries[0].PlayerId
		endCursor = entries[len(entries)-1].PlayerId
	}

	return &pb.GetLeaderboardResponse{
		Entries:         entries,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
		TotalPlayers:    totalPlayers,
	}, nil
}
