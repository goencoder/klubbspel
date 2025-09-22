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
)

const LEADERBOARD_VERSION = 6

type LeaderboardService struct {
	pb.UnimplementedLeaderboardServiceServer
	Matches *repo.MatchRepo
	Players *repo.PlayerRepo
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, in *pb.GetLeaderboardRequest) (*pb.GetLeaderboardResponse, error) {
	log.Info().Str("seriesId", in.GetSeriesId()).Msg("GetLeaderboard called")
	log.Info().Str("seriesId", in.GetSeriesId()).Msg("GetLeaderboard called")
	fmt.Printf("DEBUG GetLeaderboard: seriesId=%s (LEADERBOARD_VERSION=%d)\n", in.GetSeriesId(), LEADERBOARD_VERSION)

	// Get all matches for the series
	matches, err := s.Matches.ListBySeries(ctx, in.GetSeriesId())
	if err != nil {
		log.Error().Str("seriesId", in.GetSeriesId()).Err(err).Msg("Failed to get matches")
		fmt.Printf("DEBUG GetLeaderboard: Failed to get matches for series %s: %v\n", in.GetSeriesId(), err)
		return nil, status.Error(codes.Internal, "LEADERBOARD_FETCH_FAILED")
	}
	log.Debug().Str("seriesId", in.GetSeriesId()).Int("matchCount", len(matches)).Msg("Retrieved matches")
	fmt.Printf("DEBUG GetLeaderboard: Found %d matches for series %s\n", len(matches), in.GetSeriesId())

	// Collect all unique player IDs first
	playerIDSet := make(map[string]bool)
        for _, match := range matches {
                participantIDs := extractPlayerIDs(match.Participants)
                fmt.Printf("DEBUG Match: Participant IDs='%v'\n", participantIDs)
                for _, id := range participantIDs {
                        if id != "" {
                                playerIDSet[id] = true
                        }
                }
        }

	// Convert map keys to slice for batch lookup
	playerIDs := make([]string, 0, len(playerIDSet))
	for playerID := range playerIDSet {
		playerIDs = append(playerIDs, playerID)
	}

	fmt.Printf("DEBUG PlayerIDs collected: %v\n", playerIDs)

	// Batch fetch all player names in a single database query
	log.Info().Int("playerCount", len(playerIDs)).Msg("Batch looking up players by IDs")
	playersMap, err := s.Players.FindByIDs(ctx, playerIDs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to batch lookup players")
		return nil, status.Error(codes.Internal, "LEADERBOARD_PLAYERS_FETCH_FAILED")
	}

	// Build player names map with fallback for missing players
	playerNames := make(map[string]string)
	for _, playerID := range playerIDs {
		if player, exists := playersMap[playerID]; exists {
			playerNames[playerID] = player.DisplayName
			log.Info().Str("playerId", playerID).Str("displayName", player.DisplayName).Msg("Found player")
			fmt.Printf("DEBUG LeaderboardService: Found player %s -> %s\n", playerID, player.DisplayName)
		} else {
			// If we can't find the player, use a fallback name
			playerNames[playerID] = "Unknown Player"
			log.Error().Str("playerId", playerID).Msg("Player not found in batch lookup")
			fmt.Printf("DEBUG LeaderboardService: Failed to find player %s\n", playerID)
		}
	}

	// Calculate ratings and stats for each player
	playerStats := make(map[string]*pb.LeaderboardEntry)

        for _, match := range matches {
                if len(match.Participants) < 2 {
                        continue
                }

                playerAID, playerBID := extractHeadToHeadParticipants(match.Participants)
                if playerAID == "" || playerBID == "" {
                        continue
                }

                tableTennis := match.Result
                if tableTennis == nil || tableTennis.TableTennis == nil || len(tableTennis.TableTennis.GamesWon) < 2 {
                        continue
                }

                scoreA := tableTennis.TableTennis.GamesWon[0]
                scoreB := tableTennis.TableTennis.GamesWon[1]

                // Initialize players if not seen before
                if _, exists := playerStats[playerAID]; !exists {
                        playerStats[playerAID] = &pb.LeaderboardEntry{
                                PlayerId:      playerAID,
                                PlayerName:    playerNames[playerAID],
                                EloRating:     1000,
                                MatchesPlayed: 0,
                                MatchesWon:    0,
                                MatchesLost:   0,
                                GamesWon:      0,
                                GamesLost:     0,
                        }
                }
                if _, exists := playerStats[playerBID]; !exists {
                        playerStats[playerBID] = &pb.LeaderboardEntry{
                                PlayerId:      playerBID,
                                PlayerName:    playerNames[playerBID],
                                EloRating:     1000,
                                MatchesPlayed: 0,
                                MatchesWon:    0,
                                MatchesLost:   0,
                                GamesWon:      0,
                                GamesLost:     0,
                        }
                }

                playerA := playerStats[playerAID]
                playerB := playerStats[playerBID]

                newRatingA, newRatingB := calculateELO(float64(playerA.EloRating), float64(playerB.EloRating), scoreA, scoreB)

                playerA.EloRating = int32(newRatingA)
                playerA.MatchesPlayed++
                playerA.GamesWon += scoreA
                playerA.GamesLost += scoreB
                if scoreA > scoreB {
                        playerA.MatchesWon++
                } else {
                        playerA.MatchesLost++
                }

                playerB.EloRating = int32(newRatingB)
                playerB.MatchesPlayed++
                playerB.GamesWon += scoreB
                playerB.GamesLost += scoreA
                if scoreB > scoreA {
                        playerB.MatchesWon++
                } else {
                        playerB.MatchesLost++
                }
        }

	// Convert to slice and sort by rating
	var entries []*pb.LeaderboardEntry
	for _, stats := range playerStats {
		// Calculate win rates
		if stats.MatchesPlayed > 0 {
			stats.WinRate = float32(stats.MatchesWon) / float32(stats.MatchesPlayed) * 100
		}
		if stats.GamesWon+stats.GamesLost > 0 {
			stats.GameWinRate = float32(stats.GamesWon) / float32(stats.GamesWon+stats.GamesLost) * 100
		}
		entries = append(entries, stats)
	}

	// Sort by ELO rating (highest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].EloRating > entries[j].EloRating
	})

	// Add ranks
	for i, entry := range entries {
		entry.Rank = int32(i + 1)
	}

	// Handle pagination with cursor support
	pageSize := in.GetPageSize()
	if pageSize == 0 {
		pageSize = 20
	}

	// Apply pagination to entries based on cursors
	totalPlayers := int32(len(entries))
	startIdx := 0
	endIdx := len(entries)

	// Handle cursor_after (forward pagination)
	if cursorAfter := in.GetCursorAfter(); cursorAfter != "" {
		for i, entry := range entries {
			if entry.PlayerId == cursorAfter {
				startIdx = i + 1 // Start after the cursor
				break
			}
		}
	}

	// Handle cursor_before (backward pagination)
	if cursorBefore := in.GetCursorBefore(); cursorBefore != "" {
		for i, entry := range entries {
			if entry.PlayerId == cursorBefore {
				endIdx = i // End before the cursor
				break
			}
		}
	}

	// Apply page size limit
	if endIdx-startIdx > int(pageSize) {
		endIdx = startIdx + int(pageSize)
	}

	// Ensure we don't go out of bounds
	if startIdx >= len(entries) {
		startIdx = len(entries)
		endIdx = len(entries)
		entries = []*pb.LeaderboardEntry{} // Empty result
	} else if endIdx > len(entries) {
		endIdx = len(entries)
		entries = entries[startIdx:endIdx]
	} else {
		entries = entries[startIdx:endIdx]
	}

	var startCursor, endCursor string
	hasNext := endIdx < len(entries) || (startIdx == 0 && len(entries) > int(pageSize))
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
		LastUpdated:     time.Now().Format(time.RFC3339),
	}, nil
}

// Calculate ELO rating changes based on match result
func calculateELO(ratingA, ratingB float64, scoreA, scoreB int32) (float64, float64) {
        // K-factor for rating changes
        const K = 32

        // Expected scores
        expectedA := 1 / (1 + math.Pow(10, (ratingB-ratingA)/400))
        expectedB := 1 / (1 + math.Pow(10, (ratingA-ratingB)/400))

        // Actual scores (1 for win, 0 for loss)
        var actualA, actualB float64
        if scoreA > scoreB {
                actualA, actualB = 1, 0
        } else {
                actualA, actualB = 0, 1
        }

        // New ratings
        newRatingA := ratingA + K*(actualA-expectedA)
        newRatingB := ratingB + K*(actualB-expectedB)

        return newRatingA, newRatingB
}

func extractPlayerIDs(participants []repo.MatchParticipant) []string {
        ids := make([]string, 0, len(participants))
        for _, participant := range participants {
                if participant.PlayerID != nil {
                        ids = append(ids, *participant.PlayerID)
                }
        }
        return ids
}

func extractHeadToHeadParticipants(participants []repo.MatchParticipant) (string, string) {
        var playerAID, playerBID string
        if len(participants) > 0 && participants[0].PlayerID != nil {
                playerAID = *participants[0].PlayerID
        }
        if len(participants) > 1 && participants[1].PlayerID != nil {
                playerBID = *participants[1].PlayerID
        }
        return playerAID, playerBID
}
