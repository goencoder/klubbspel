package service

import (
	"context"

	"github.com/goencoder/klubbspel/backend/internal/i18n"
	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SeriesService struct {
	pb.UnimplementedSeriesServiceServer
	Series  *repo.SeriesRepo
	Matches *repo.MatchRepo
	Players *repo.PlayerRepo
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

	// Ladder standings are no longer stored separately
	// Use the leaderboard service instead for all series types
	return nil, status.Error(codes.Unimplemented, "LADDER_STANDINGS_DEPRECATED")
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

	// Get locale from request, default to Swedish
	locale := in.GetLocale()
	if locale == "" {
		locale = "sv"
	}
	// Normalize locale to supported values
	if locale != "sv" && locale != "en" {
		locale = "sv" // Default to Swedish
	}

	var rules *pb.RulesDescription

	switch format {
	case pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY:
		rulesContent, err := i18n.GetFreePlayRules(locale)
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_LOAD_RULES")
		}
		rules = &pb.RulesDescription{
			Title:   rulesContent.Title,
			Summary: rulesContent.Summary,
			Rules:   rulesContent.Rules,
		}
		for _, ex := range rulesContent.Examples {
			rules.Examples = append(rules.Examples, &pb.RuleExample{
				Scenario: ex.Scenario,
				Outcome:  ex.Outcome,
			})
		}

	case pb.SeriesFormat_SERIES_FORMAT_LADDER:
		ladderRules := in.GetLadderRules()
		if ladderRules == pb.LadderRules_LADDER_RULES_UNSPECIFIED {
			ladderRules = pb.LadderRules_LADDER_RULES_CLASSIC
		}

		isAggressive := ladderRules == pb.LadderRules_LADDER_RULES_AGGRESSIVE
		rulesContent, err := i18n.GetLadderRules(locale, isAggressive)
		if err != nil {
			return nil, status.Error(codes.Internal, "FAILED_TO_LOAD_RULES")
		}

		rules = &pb.RulesDescription{
			Title:   rulesContent.Title,
			Summary: rulesContent.Summary,
			Rules:   rulesContent.Rules,
		}
		for _, ex := range rulesContent.Examples {
			rules.Examples = append(rules.Examples, &pb.RuleExample{
				Scenario: ex.Scenario,
				Outcome:  ex.Outcome,
			})
		}

	default:
		return nil, status.Error(codes.Unimplemented, "SERIES_FORMAT_NOT_SUPPORTED")
	}

	return &pb.GetSeriesRulesResponse{
		Rules: rules,
	}, nil
}
