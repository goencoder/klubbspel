package service

import (
	"context"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MatchService struct {
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

	// TODO: Add CEL validation for series time window
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

func (s *MatchService) mustEmbedUnimplementedMatchServiceServer() {}
