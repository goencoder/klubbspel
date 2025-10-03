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

type SeriesService struct {
	pb.UnimplementedSeriesServiceServer
	Series        *repo.SeriesRepo
	Matches       *repo.MatchRepo
	Players       *repo.PlayerRepo
	SeriesPlayers *repo.SeriesPlayerRepo
}

var supportedSeriesSports = map[pb.Sport]struct{}{
	pb.Sport_SPORT_TABLE_TENNIS: {},
	pb.Sport_SPORT_TENNIS:       {},
	pb.Sport_SPORT_PADEL:        {},
	pb.Sport_SPORT_BADMINTON:    {},
	pb.Sport_SPORT_SQUASH:       {},
	pb.Sport_SPORT_PICKLEBALL:   {},
}

func (s *SeriesService) CreateSeries(ctx context.Context, in *pb.CreateSeriesRequest) (*pb.CreateSeriesResponse, error) {
	startsAt := in.GetStartsAt().AsTime()
	endsAt := in.GetEndsAt().AsTime()

	sport, err := normalizeSeriesSport(in.GetSport())
	if err != nil {
		return nil, err
	}

	format, err := normalizeSeriesFormat(in.GetFormat())
	if err != nil {
		return nil, err
	}

	// Set ladder rules default for LADDER format
	ladderRules := in.GetLadderRules()
	if format == pb.SeriesFormat_SERIES_FORMAT_LADDER && ladderRules == pb.LadderRules_LADDER_RULES_UNSPECIFIED {
		ladderRules = pb.LadderRules_LADDER_RULES_CLASSIC // Default to classic (no penalty)
	}

	// Set scoring profile based on sport if not specified
	scoringProfile := in.GetScoringProfile()
	if scoringProfile == pb.ScoringProfile_SCORING_PROFILE_UNSPECIFIED {
		switch sport {
		case pb.Sport_SPORT_TABLE_TENNIS, pb.Sport_SPORT_BADMINTON, pb.Sport_SPORT_SQUASH, pb.Sport_SPORT_PADEL, pb.Sport_SPORT_PICKLEBALL:
			scoringProfile = pb.ScoringProfile_SCORING_PROFILE_TABLE_TENNIS_SETS
		case pb.Sport_SPORT_TENNIS:
			scoringProfile = pb.ScoringProfile_SCORING_PROFILE_SCORELINE
		default:
			return nil, status.Error(codes.InvalidArgument, "SCORING_PROFILE_REQUIRED_FOR_SPORT")
		}
	}

	// Set sets_to_play default for racket/paddle sports
	setsToPlay := in.GetSetsToPlay()
	if setsToPlay == 0 {
		switch sport {
		case pb.Sport_SPORT_TABLE_TENNIS, pb.Sport_SPORT_BADMINTON, pb.Sport_SPORT_SQUASH, pb.Sport_SPORT_PADEL, pb.Sport_SPORT_PICKLEBALL:
			setsToPlay = 5 // Default to best-of-5
		}
	}

	series, err := s.Series.Create(ctx, in.GetClubId(), in.GetTitle(), startsAt, endsAt, int32(in.GetVisibility()), int32(sport), int32(format), int32(ladderRules), int32(scoringProfile), setsToPlay)
	if err != nil {
		return nil, status.Error(codes.Internal, "SERIES_CREATE_FAILED")
	}

	return &pb.CreateSeriesResponse{
		Series: &pb.Series{
			Id:             series.ID.Hex(),
			ClubId:         series.ClubID,
			Title:          series.Title,
			StartsAt:       timestamppb.New(series.StartsAt),
			EndsAt:         timestamppb.New(series.EndsAt),
			Visibility:     pb.SeriesVisibility(series.Visibility),
			Sport:          pbSeriesSport(series.Sport),
			Format:         pbSeriesFormat(series.Format),
			LadderRules:    pb.LadderRules(series.LadderRules),
			ScoringProfile: pb.ScoringProfile(series.ScoringProfile),
			SetsToPlay:     series.SetsToPlay,
		},
	}, nil
}

func (s *SeriesService) ListSeries(ctx context.Context, in *pb.ListSeriesRequest) (*pb.ListSeriesResponse, error) {
	// Use cursor_after for forward pagination
	cursor := in.GetCursorAfter()
	filters := repo.SeriesListFilters{}

	if in.GetSportFilter() != pb.Sport_SPORT_UNSPECIFIED {
		sport, err := normalizeSeriesSport(in.GetSportFilter())
		if err != nil {
			return nil, err
		}
		sportValue := int32(sport)
		filters.Sport = &sportValue
	}

	// Handle club filtering
	clubFilters := in.GetClubFilter()
	if len(clubFilters) > 0 {
		var clubIDs []string
		for _, clubFilter := range clubFilters {
			if clubFilter == "OPEN" {
				filters.IncludeOpen = true
			} else {
				clubIDs = append(clubIDs, clubFilter)
			}
		}
		filters.ClubIDs = clubIDs
	}

	seriesList, hasNext, hasPrev, err := s.Series.ListWithCursor(ctx, in.GetPageSize(), cursor, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "SERIES_LIST_FAILED")
	}

	var pbSeries []*pb.Series
	for _, series := range seriesList {
		pbSeries = append(pbSeries, &pb.Series{
			Id:             series.ID.Hex(),
			ClubId:         series.ClubID,
			Title:          series.Title,
			StartsAt:       timestamppb.New(series.StartsAt),
			EndsAt:         timestamppb.New(series.EndsAt),
			Visibility:     pb.SeriesVisibility(series.Visibility),
			Sport:          pbSeriesSport(series.Sport),
			Format:         pbSeriesFormat(series.Format),
			LadderRules:    pb.LadderRules(series.LadderRules),
			ScoringProfile: pb.ScoringProfile(series.ScoringProfile),
			SetsToPlay:     series.SetsToPlay,
		})
	}

	// Simplified pagination info
	var startCursor, endCursor string
	if len(pbSeries) > 0 {
		startCursor = pbSeries[0].Id
		endCursor = pbSeries[len(pbSeries)-1].Id
	}

	return &pb.ListSeriesResponse{
		Items:           pbSeries,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}, nil
}

func (s *SeriesService) GetSeries(ctx context.Context, in *pb.GetSeriesRequest) (*pb.GetSeriesResponse, error) {
	series, err := s.Series.FindByID(ctx, in.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "SERIES_NOT_FOUND")
	}

	return &pb.GetSeriesResponse{
		Series: &pb.Series{
			Id:             series.ID.Hex(),
			ClubId:         series.ClubID,
			Title:          series.Title,
			StartsAt:       timestamppb.New(series.StartsAt),
			EndsAt:         timestamppb.New(series.EndsAt),
			Visibility:     pb.SeriesVisibility(series.Visibility),
			Sport:          pbSeriesSport(series.Sport),
			Format:         pbSeriesFormat(series.Format),
			LadderRules:    pb.LadderRules(series.LadderRules),
			ScoringProfile: pb.ScoringProfile(series.ScoringProfile),
			SetsToPlay:     series.SetsToPlay,
		},
	}, nil
}

func (s *SeriesService) UpdateSeries(ctx context.Context, in *pb.UpdateSeriesRequest) (*pb.UpdateSeriesResponse, error) {
	updates := map[string]interface{}{}
	if mask := in.GetUpdateMask(); mask != nil && len(mask.GetPaths()) > 0 {
		for _, path := range mask.GetPaths() {
			switch path {
			case "title":
				updates["title"] = in.GetSeries().GetTitle()
			case "starts_at":
				updates["starts_at"] = in.GetSeries().GetStartsAt().AsTime()
			case "ends_at":
				updates["ends_at"] = in.GetSeries().GetEndsAt().AsTime()
			case "visibility":
				updates["visibility"] = int32(in.GetSeries().GetVisibility())
			case "club_id":
				updates["club_id"] = in.GetSeries().GetClubId()
			case "sport":
				sport, err := normalizeSeriesSport(in.GetSeries().GetSport())
				if err != nil {
					return nil, err
				}
				updates["sport"] = int32(sport)
			case "format":
				format, err := normalizeSeriesFormat(in.GetSeries().GetFormat())
				if err != nil {
					return nil, err
				}
				updates["format"] = int32(format)
			case "scoring_profile":
				updates["scoring_profile"] = int32(in.GetSeries().GetScoringProfile())
			case "sets_to_play":
				updates["sets_to_play"] = in.GetSeries().GetSetsToPlay()
			}
		}
	} else {
		updates["title"] = in.GetSeries().GetTitle()
		updates["starts_at"] = in.GetSeries().GetStartsAt().AsTime()
		updates["ends_at"] = in.GetSeries().GetEndsAt().AsTime()
		updates["visibility"] = int32(in.GetSeries().GetVisibility())
		updates["club_id"] = in.GetSeries().GetClubId()
		sport, err := normalizeSeriesSport(in.GetSeries().GetSport())
		if err != nil {
			return nil, err
		}
		updates["sport"] = int32(sport)
		format, err := normalizeSeriesFormat(in.GetSeries().GetFormat())
		if err != nil {
			return nil, err
		}
		updates["format"] = int32(format)
		updates["scoring_profile"] = int32(in.GetSeries().GetScoringProfile())
		updates["sets_to_play"] = in.GetSeries().GetSetsToPlay()
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NO_FIELDS_TO_UPDATE")
	}

	series, err := s.Series.Update(ctx, in.GetId(), updates)
	if err != nil {
		return nil, status.Error(codes.Internal, "SERIES_UPDATE_FAILED")
	}

	return &pb.UpdateSeriesResponse{
		Series: &pb.Series{
			Id:             series.ID.Hex(),
			ClubId:         series.ClubID,
			Title:          series.Title,
			StartsAt:       timestamppb.New(series.StartsAt),
			EndsAt:         timestamppb.New(series.EndsAt),
			Visibility:     pb.SeriesVisibility(series.Visibility),
			Sport:          pbSeriesSport(series.Sport),
			Format:         pbSeriesFormat(series.Format),
			LadderRules:    pb.LadderRules(series.LadderRules),
			ScoringProfile: pb.ScoringProfile(series.ScoringProfile),
			SetsToPlay:     series.SetsToPlay,
		},
	}, nil
}

func (s *SeriesService) DeleteSeries(ctx context.Context, in *pb.DeleteSeriesRequest) (*pb.DeleteSeriesResponse, error) {
	if err := s.Series.Delete(ctx, in.GetId()); err != nil {
		return nil, status.Error(codes.Internal, "SERIES_DELETE_FAILED")
	}

	return &pb.DeleteSeriesResponse{Success: true}, nil
}

func (s *SeriesService) GetLadderStandings(ctx context.Context, in *pb.GetLadderStandingsRequest) (*pb.GetLadderStandingsResponse, error) {
	if in.GetSeriesId() == "" {
		return nil, status.Error(codes.InvalidArgument, "SERIES_ID_REQUIRED")
	}

	series, err := s.Series.FindByID(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "SERIES_NOT_FOUND")
	}

	if pb.SeriesFormat(series.Format) != pb.SeriesFormat_SERIES_FORMAT_LADDER {
		return nil, status.Error(codes.FailedPrecondition, "SERIES_NOT_LADDER")
	}

	if s.SeriesPlayers == nil {
		return nil, status.Error(codes.Internal, "LADDER_REPOSITORY_UNAVAILABLE")
	}
	if s.Players == nil {
		return nil, status.Error(codes.Internal, "PLAYER_REPOSITORY_UNAVAILABLE")
	}
	if s.Matches == nil {
		return nil, status.Error(codes.Internal, "MATCH_REPOSITORY_UNAVAILABLE")
	}

	ladderEntries, err := s.SeriesPlayers.FindBySeriesOrdered(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Error(codes.Internal, "LADDER_FETCH_FAILED")
	}

	// Collect player IDs for name lookup.
	playerIDs := make([]string, 0, len(ladderEntries))
	for _, entry := range ladderEntries {
		playerIDs = append(playerIDs, entry.PlayerID)
	}

	playersMap := map[string]*repo.Player{}
	if len(playerIDs) > 0 {
		playersMap, err = s.Players.FindByIDs(ctx, playerIDs)
		if err != nil {
			return nil, status.Error(codes.Internal, "LADDER_PLAYER_LOOKUP_FAILED")
		}
	}

	// Prepare statistics from matches.
	matches, err := s.Matches.FindBySeriesID(ctx, in.GetSeriesId())
	if err != nil {
		return nil, status.Error(codes.Internal, "LADDER_MATCH_FETCH_FAILED")
	}

	type ladderStats struct {
		matchesPlayed int32
		matchesWon    int32
		lastMatch     time.Time
	}

	stats := make(map[string]*ladderStats)

	for _, match := range matches {
		// Ensure stats entries exist for both players participating in the match.
		if _, ok := stats[match.PlayerAID]; !ok {
			stats[match.PlayerAID] = &ladderStats{}
		}
		if _, ok := stats[match.PlayerBID]; !ok {
			stats[match.PlayerBID] = &ladderStats{}
		}

		stats[match.PlayerAID].matchesPlayed++
		stats[match.PlayerBID].matchesPlayed++

		if match.ScoreA > match.ScoreB {
			stats[match.PlayerAID].matchesWon++
		} else {
			stats[match.PlayerBID].matchesWon++
		}

		if match.PlayedAt.After(stats[match.PlayerAID].lastMatch) {
			stats[match.PlayerAID].lastMatch = match.PlayedAt
		}
		if match.PlayedAt.After(stats[match.PlayerBID].lastMatch) {
			stats[match.PlayerBID].lastMatch = match.PlayedAt
		}
	}

	response := &pb.GetLadderStandingsResponse{}

	for _, entry := range ladderEntries {
		name := "Unknown Player"
		if player, ok := playersMap[entry.PlayerID]; ok {
			name = player.DisplayName
		}

		stat := stats[entry.PlayerID]

		ladderEntry := &pb.LadderEntry{
			PlayerId:      entry.PlayerID,
			PlayerName:    name,
			Position:      entry.Position,
			MatchesPlayed: 0,
			MatchesWon:    0,
		}

		if stat != nil {
			ladderEntry.MatchesPlayed = stat.matchesPlayed
			ladderEntry.MatchesWon = stat.matchesWon
			if !stat.lastMatch.IsZero() {
				ladderEntry.LastMatchAt = timestamppb.New(stat.lastMatch)
			}
		}

		response.Entries = append(response.Entries, ladderEntry)
	}

	return response, nil
}

func normalizeSeriesSport(sport pb.Sport) (pb.Sport, error) {
	if sport == pb.Sport_SPORT_UNSPECIFIED {
		return pb.Sport_SPORT_TABLE_TENNIS, nil
	}

	if _, ok := supportedSeriesSports[sport]; !ok {
		return pb.Sport_SPORT_UNSPECIFIED, status.Error(codes.Unimplemented, "SPORT_NOT_SUPPORTED")
	}

	return sport, nil
}

func pbSeriesSport(value int32) pb.Sport {
	sport := pb.Sport(value)
	if sport == pb.Sport_SPORT_UNSPECIFIED {
		sport = pb.Sport_SPORT_TABLE_TENNIS
	}
	return sport
}

func normalizeSeriesFormat(format pb.SeriesFormat) (pb.SeriesFormat, error) {
	if format == pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED {
		return pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY, nil
	}

	switch format {
	case pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY, pb.SeriesFormat_SERIES_FORMAT_LADDER:
		return format, nil
	default:
		return pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED, status.Error(codes.Unimplemented, "SERIES_FORMAT_NOT_SUPPORTED")
	}
}

func pbSeriesFormat(value int32) pb.SeriesFormat {
	format := pb.SeriesFormat(value)
	if format == pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED {
		format = pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY
	}
	return format
}

func (s *SeriesService) GetSeriesRules(ctx context.Context, in *pb.GetSeriesRulesRequest) (*pb.GetSeriesRulesResponse, error) {
	format := in.GetFormat()
	if format == pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED {
		format = pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY
	}

	var rules *pb.RulesDescription

	switch format {
	case pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY:
		rules = &pb.RulesDescription{
			Title:   "Free Play Rules",
			Summary: "Play matches freely with any player. Rankings are determined by ELO rating.",
			Rules: []string{
				"Play matches against any player in the series",
				"No position tracking - rankings based on ELO rating only",
				"Winner gains ELO points, loser loses ELO points",
				"ELO changes depend on rating difference between players",
				"All matches count equally toward your rating",
			},
			Examples: []*pb.RuleExample{
				{
					Scenario: "Higher-rated player (ELO 1500) beats lower-rated player (ELO 1200)",
					Outcome:  "Winner gains ~8 points, loser loses ~8 points (small change due to expected outcome)",
				},
				{
					Scenario: "Lower-rated player (ELO 1200) beats higher-rated player (ELO 1500)",
					Outcome:  "Winner gains ~24 points, loser loses ~24 points (large change due to upset)",
				},
			},
		}

	case pb.SeriesFormat_SERIES_FORMAT_LADDER:
		ladderRules := in.GetLadderRules()
		if ladderRules == pb.LadderRules_LADDER_RULES_UNSPECIFIED {
			ladderRules = pb.LadderRules_LADDER_RULES_CLASSIC
		}

		if ladderRules == pb.LadderRules_LADDER_RULES_AGGRESSIVE {
			rules = &pb.RulesDescription{
				Title:   "Aggressive Ladder Rules",
				Summary: "Challenge any player to climb the ladder. Winner improves position, loser drops one position (penalty).",
				Rules: []string{
					"Players are ranked by position (1, 2, 3, etc.)",
					"Play matches against any player regardless of position",
					"Position determines ranking, not ELO",
					"Winner with worse position takes the better player's position",
					"All players between swap positions (shift down one)",
					"Loser with worse position drops one additional position (penalty)",
					"Player below loser moves up to fill the gap",
					"ELO is still calculated but doesn't affect ladder position",
				},
				Examples: []*pb.RuleExample{
					{
						Scenario: "Player at position #3 beats player at position #1",
						Outcome:  "Winner → position #1, positions #1 and #2 → shift down (#2 and #3)",
					},
					{
						Scenario: "Player at position #3 loses to player at position #1",
						Outcome:  "Loser drops to position #4 (penalty), player at #4 → moves up to #3",
					},
					{
						Scenario: "Player at position #2 beats player at position #1",
						Outcome:  "Winner → position #1, previous #1 → position #2",
					},
					{
						Scenario: "Player at position #1 loses to player at position #3",
						Outcome:  "Winner → position #1, loser (#1) → drops to position #2, previous #2 → position #3",
					},
				},
			}
		} else {
			// LADDER_RULES_CLASSIC
			rules = &pb.RulesDescription{
				Title:   "Classic Ladder Rules",
				Summary: "Challenge any player to climb the ladder. Winner improves position, loser keeps their position (no penalty).",
				Rules: []string{
					"Players are ranked by position (1, 2, 3, etc.)",
					"Play matches against any player regardless of position",
					"Position determines ranking, not ELO",
					"Winner with worse position takes the better player's position",
					"All players between swap positions (shift down one)",
					"Loser with worse position keeps their position (no penalty)",
					"Loser with better position drops to where winner was",
					"ELO is still calculated but doesn't affect ladder position",
				},
				Examples: []*pb.RuleExample{
					{
						Scenario: "Player at position #3 beats player at position #1",
						Outcome:  "Winner → position #1, positions #1 and #2 → shift down (#2 and #3)",
					},
					{
						Scenario: "Player at position #3 loses to player at position #1",
						Outcome:  "Loser keeps position #3 (no penalty), winner keeps position #1",
					},
					{
						Scenario: "Player at position #2 beats player at position #1",
						Outcome:  "Winner → position #1, previous #1 → position #2",
					},
					{
						Scenario: "Player at position #1 loses to player at position #3",
						Outcome:  "Winner → position #1, loser (#1) → drops to position #3, previous #2 → position #2",
					},
				},
			}
		}

	default:
		return nil, status.Error(codes.Unimplemented, "SERIES_FORMAT_NOT_SUPPORTED")
	}

	return &pb.GetSeriesRulesResponse{
		Rules: rules,
	}, nil
}
