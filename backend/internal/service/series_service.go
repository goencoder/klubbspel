package service

import (
	"context"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SeriesService struct {
	pb.UnimplementedSeriesServiceServer
	Series *repo.SeriesRepo
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

        matchConfig, err := normalizeSeriesMatchConfiguration(in.GetMatchConfiguration())
        if err != nil {
                return nil, err
        }

        series, err := s.Series.Create(ctx, in.GetClubId(), in.GetTitle(), startsAt, endsAt, int32(in.GetVisibility()), int32(sport), int32(format), matchConfig)
        if err != nil {
                return nil, status.Error(codes.Internal, "SERIES_CREATE_FAILED")
        }

        return &pb.CreateSeriesResponse{
                Series: &pb.Series{
                        Id:         series.ID.Hex(),
                        ClubId:     series.ClubID,
                        Title:      series.Title,
                        StartsAt:   timestamppb.New(series.StartsAt),
                        EndsAt:     timestamppb.New(series.EndsAt),
                        Visibility: pb.SeriesVisibility(series.Visibility),
                        Sport:      pbSeriesSport(series.Sport),
                        Format:     pbSeriesFormat(series.Format),
                        MatchConfiguration: pbSeriesMatchConfiguration(series.MatchConfiguration),
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
                        Id:         series.ID.Hex(),
                        ClubId:     series.ClubID,
			Title:      series.Title,
                        StartsAt:   timestamppb.New(series.StartsAt),
                        EndsAt:     timestamppb.New(series.EndsAt),
                        Visibility: pb.SeriesVisibility(series.Visibility),
                        Sport:      pbSeriesSport(series.Sport),
                        Format:     pbSeriesFormat(series.Format),
                        MatchConfiguration: pbSeriesMatchConfiguration(series.MatchConfiguration),
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
                        Id:         series.ID.Hex(),
                        ClubId:     series.ClubID,
                        Title:      series.Title,
                        StartsAt:   timestamppb.New(series.StartsAt),
                        EndsAt:     timestamppb.New(series.EndsAt),
                        Visibility: pb.SeriesVisibility(series.Visibility),
                        Sport:      pbSeriesSport(series.Sport),
                        Format:     pbSeriesFormat(series.Format),
                        MatchConfiguration: pbSeriesMatchConfiguration(series.MatchConfiguration),
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
                        case "match_configuration":
                                matchConfig, err := normalizeSeriesMatchConfiguration(in.GetSeries().GetMatchConfiguration())
                                if err != nil {
                                        return nil, err
                                }
                                updates["match_configuration"] = matchConfig
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
                matchConfig, err := normalizeSeriesMatchConfiguration(in.GetSeries().GetMatchConfiguration())
                if err != nil {
                        return nil, err
                }
                updates["match_configuration"] = matchConfig
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
                        Id:         series.ID.Hex(),
                        ClubId:     series.ClubID,
                        Title:      series.Title,
                        StartsAt:   timestamppb.New(series.StartsAt),
                        EndsAt:     timestamppb.New(series.EndsAt),
                        Visibility: pb.SeriesVisibility(series.Visibility),
                        Sport:      pbSeriesSport(series.Sport),
                        Format:     pbSeriesFormat(series.Format),
                        MatchConfiguration: pbSeriesMatchConfiguration(series.MatchConfiguration),
                },
        }, nil
}

func (s *SeriesService) DeleteSeries(ctx context.Context, in *pb.DeleteSeriesRequest) (*pb.DeleteSeriesResponse, error) {
	if err := s.Series.Delete(ctx, in.GetId()); err != nil {
		return nil, status.Error(codes.Internal, "SERIES_DELETE_FAILED")
	}

	return &pb.DeleteSeriesResponse{Success: true}, nil
}

func normalizeSeriesSport(sport pb.Sport) (pb.Sport, error) {
	if sport == pb.Sport_SPORT_UNSPECIFIED {
		return pb.Sport_SPORT_TABLE_TENNIS, nil
	}

	if sport != pb.Sport_SPORT_TABLE_TENNIS {
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

	if format != pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY {
		return pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED, status.Error(codes.Unimplemented, "SERIES_FORMAT_NOT_SUPPORTED")
	}

	return format, nil
}

func pbSeriesFormat(value int32) pb.SeriesFormat {
        format := pb.SeriesFormat(value)
        if format == pb.SeriesFormat_SERIES_FORMAT_UNSPECIFIED {
                format = pb.SeriesFormat_SERIES_FORMAT_OPEN_PLAY
        }
        return format
}

func normalizeSeriesMatchConfiguration(cfg *pb.SeriesMatchConfiguration) (repo.SeriesMatchConfiguration, error) {
        mode := pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_INDIVIDUAL
        participantsPerSide := int32(1)
        profile := pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_TABLE_TENNIS

        if cfg != nil {
                if cfg.GetParticipantMode() != pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_UNSPECIFIED {
                        mode = cfg.GetParticipantMode()
                }
                if cfg.GetParticipantsPerSide() > 0 {
                        participantsPerSide = cfg.GetParticipantsPerSide()
                }
                if cfg.GetScoringProfile() != pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_UNSPECIFIED {
                        profile = cfg.GetScoringProfile()
                }
        }

        if mode != pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_INDIVIDUAL {
                return repo.SeriesMatchConfiguration{}, status.Error(codes.Unimplemented, "PARTICIPANT_MODE_NOT_SUPPORTED")
        }

        if participantsPerSide != 1 {
                return repo.SeriesMatchConfiguration{}, status.Error(codes.Unimplemented, "PARTICIPANT_COUNT_NOT_SUPPORTED")
        }

        if profile != pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_TABLE_TENNIS {
                return repo.SeriesMatchConfiguration{}, status.Error(codes.Unimplemented, "SCORING_PROFILE_NOT_SUPPORTED")
        }

        return repo.SeriesMatchConfiguration{
                ParticipantMode:     int32(mode),
                ParticipantsPerSide: participantsPerSide,
                ScoringProfile:      int32(profile),
        }, nil
}

func pbSeriesMatchConfiguration(cfg repo.SeriesMatchConfiguration) *pb.SeriesMatchConfiguration {
        mode := pb.SeriesParticipantMode(cfg.ParticipantMode)
        if mode == pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_UNSPECIFIED {
                mode = pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_INDIVIDUAL
        }

        participantsPerSide := cfg.ParticipantsPerSide
        if participantsPerSide == 0 {
                participantsPerSide = 1
        }

        profile := pb.SeriesScoringProfile(cfg.ScoringProfile)
        if profile == pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_UNSPECIFIED {
                profile = pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_TABLE_TENNIS
        }

        return &pb.SeriesMatchConfiguration{
                ParticipantMode:     mode,
                ParticipantsPerSide: participantsPerSide,
                ScoringProfile:      profile,
        }
}
