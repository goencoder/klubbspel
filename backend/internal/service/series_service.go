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

	series, err := s.Series.Create(ctx, in.GetClubId(), in.GetTitle(), startsAt, endsAt, int32(in.GetVisibility()))
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
		},
	}, nil
}

func (s *SeriesService) ListSeries(ctx context.Context, in *pb.ListSeriesRequest) (*pb.ListSeriesResponse, error) {
	// Use cursor_after for forward pagination
	cursor := in.GetCursorAfter()
	seriesList, hasNext, hasPrev, err := s.Series.ListWithCursor(ctx, in.GetPageSize(), cursor)
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
			}
		}
	} else {
		updates["title"] = in.GetSeries().GetTitle()
		updates["starts_at"] = in.GetSeries().GetStartsAt().AsTime()
		updates["ends_at"] = in.GetSeries().GetEndsAt().AsTime()
		updates["visibility"] = int32(in.GetSeries().GetVisibility())
		updates["club_id"] = in.GetSeries().GetClubId()
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
		},
	}, nil
}

func (s *SeriesService) DeleteSeries(ctx context.Context, in *pb.DeleteSeriesRequest) (*pb.DeleteSeriesResponse, error) {
	if err := s.Series.Delete(ctx, in.GetId()); err != nil {
		return nil, status.Error(codes.Internal, "SERIES_DELETE_FAILED")
	}

	return &pb.DeleteSeriesResponse{Success: true}, nil
}
