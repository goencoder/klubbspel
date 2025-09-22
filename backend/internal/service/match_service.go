package service

import (
        "context"
        "fmt"
        "time"

        "github.com/goencoder/klubbspel/backend/internal/repo"
        pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
        "google.golang.org/grpc/codes"
        "google.golang.org/grpc/status"
        "google.golang.org/protobuf/types/known/timestamppb"
)

const defaultTableTennisBestOf = int32(5)

type MatchService struct {
        pb.UnimplementedMatchServiceServer
        Matches *repo.MatchRepo
        Players *repo.PlayerRepo
        Series  *repo.SeriesRepo
}

func (s *MatchService) ReportMatch(ctx context.Context, in *pb.ReportMatchRequest) (*pb.ReportMatchResponse, error) {
        if in.GetMetadata() == nil {
                return nil, status.Error(codes.InvalidArgument, "VALIDATION_REQUIRED")
        }

        metadata := in.GetMetadata()
        if metadata.GetSeriesId() == "" {
                return nil, status.Error(codes.InvalidArgument, "VALIDATION_REQUIRED")
        }
        playedAt := metadata.GetPlayedAt().AsTime()

        if len(in.GetParticipants()) != 2 {
                return nil, status.Error(codes.InvalidArgument, "MATCH_REQUIRES_TWO_PARTICIPANTS")
        }

        participants, err := convertParticipants(in.GetParticipants())
        if err != nil {
                return nil, status.Error(codes.InvalidArgument, err.Error())
        }

        playerAID, playerBID := repoParticipantIDs(participants)
        if playerAID == "" || playerBID == "" {
                return nil, status.Error(codes.InvalidArgument, "PLAYER_IDS_REQUIRED")
        }
        if playerAID == playerBID {
                return nil, status.Error(codes.InvalidArgument, "VALIDATION_SAME_PLAYER")
        }

        series, err := s.Series.FindByID(ctx, metadata.GetSeriesId())
        if err != nil {
                return nil, status.Errorf(codes.Internal, "failed to find series: %v", err)
        }

        if err := validateSeriesWindow(series, playedAt); err != nil {
                return nil, err
        }

        if err := ensureSeriesSupportsTableTennis(series); err != nil {
                return nil, err
        }

        result, scoreA, scoreB, err := convertTableTennisResult(in.GetResult())
        if err != nil {
                return nil, err
        }

        if err := validateTableTennisScores(scoreA, scoreB, result.TableTennis.BestOf); err != nil {
                return nil, err
        }

        repoMatch, err := s.Matches.Create(ctx, metadata.GetSeriesId(), playedAt, participants, *result)
        if err != nil {
                return nil, status.Error(codes.Internal, "MATCH_CREATE_FAILED")
        }

        return &pb.ReportMatchResponse{MatchId: repoMatch.ID.Hex()}, nil
}

func (s *MatchService) ListMatches(ctx context.Context, in *pb.ListMatchesRequest) (*pb.ListMatchesResponse, error) {
        matches, nextPageToken, err := s.Matches.ListBySeriesID(ctx, in.GetSeriesId(), in.GetPageSize(), in.GetCursorAfter())
        if err != nil {
                return nil, status.Error(codes.Internal, "MATCH_LIST_FAILED")
        }

        var pbMatches []*pb.MatchView
        for _, match := range matches {
                                pbMatches = append(pbMatches, toPBMatchView(match))
        }

        var startCursor, endCursor string
        if len(pbMatches) > 0 {
                startCursor = pbMatches[0].GetId()
                endCursor = pbMatches[len(pbMatches)-1].GetId()
        }

        return &pb.ListMatchesResponse{
                Items:           pbMatches,
                StartCursor:     startCursor,
                EndCursor:       endCursor,
                HasNextPage:     nextPageToken != "",
                HasPreviousPage: in.GetCursorAfter() != "",
        }, nil
}

func (s *MatchService) UpdateMatch(ctx context.Context, in *pb.UpdateMatchRequest) (*pb.UpdateMatchResponse, error) {
        if in.GetMatchId() == "" {
                return nil, status.Error(codes.InvalidArgument, "MATCH_ID_REQUIRED")
        }

        existingMatch, err := s.Matches.FindByID(ctx, in.GetMatchId())
        if err != nil {
                return nil, status.Error(codes.NotFound, "MATCH_NOT_FOUND")
        }

        series, err := s.Series.FindByID(ctx, existingMatch.SeriesID)
        if err != nil {
                return nil, status.Error(codes.Internal, "SERIES_NOT_FOUND")
        }

        var playedAt *time.Time
        if in.GetPlayedAt() != nil {
                t := in.GetPlayedAt().AsTime()
                if err := validateSeriesWindow(series, t); err != nil {
                        return nil, err
                }
                playedAt = &t
        }

        var resultUpdate *repo.MatchResult
        if in.GetResult() != nil {
                if err := ensureSeriesSupportsTableTennis(series); err != nil {
                        return nil, err
                }

                result, scoreA, scoreB, err := convertTableTennisResult(in.GetResult())
                if err != nil {
                        return nil, err
                }

                if err := validateTableTennisScores(scoreA, scoreB, result.TableTennis.BestOf); err != nil {
                        return nil, err
                }

                resultUpdate = result
        }

        updatedMatch, err := s.Matches.Update(ctx, in.GetMatchId(), playedAt, resultUpdate)
        if err != nil {
                return nil, status.Error(codes.Internal, "MATCH_UPDATE_FAILED")
        }

        view, err := s.buildMatchView(ctx, updatedMatch)
        if err != nil {
                return nil, err
        }

        return &pb.UpdateMatchResponse{Match: view}, nil
}

func (s *MatchService) DeleteMatch(ctx context.Context, in *pb.DeleteMatchRequest) (*pb.DeleteMatchResponse, error) {
        if in.GetMatchId() == "" {
                return nil, status.Error(codes.InvalidArgument, "MATCH_ID_REQUIRED")
        }

        if err := s.Matches.Delete(ctx, in.GetMatchId()); err != nil {
                return nil, status.Error(codes.Internal, "MATCH_DELETE_FAILED")
        }

        return &pb.DeleteMatchResponse{Success: true}, nil
}

func (s *MatchService) ReorderMatches(ctx context.Context, in *pb.ReorderMatchesRequest) (*pb.ReorderMatchesResponse, error) {
        if len(in.GetMatchIds()) < 2 {
                return nil, status.Error(codes.InvalidArgument, "AT_LEAST_TWO_MATCHES_REQUIRED")
        }

        if err := s.Matches.ReorderMatches(ctx, in.GetMatchIds()); err != nil {
                return nil, status.Error(codes.Internal, "MATCH_REORDER_FAILED")
        }

        return &pb.ReorderMatchesResponse{Success: true}, nil
}

func (s *MatchService) buildMatchView(ctx context.Context, match *repo.Match) (*pb.MatchView, error) {
        participantIDs := map[string]struct{}{}
        for _, participant := range match.Participants {
                if participant.PlayerID != nil && *participant.PlayerID != "" {
                        participantIDs[*participant.PlayerID] = struct{}{}
                }
        }

        idList := make([]string, 0, len(participantIDs))
        for id := range participantIDs {
                idList = append(idList, id)
        }

        players, err := s.Players.FindByIDs(ctx, idList)
        if err != nil {
                return nil, status.Error(codes.Internal, "PLAYER_LOOKUP_FAILED")
        }

        var participantViews []*pb.MatchParticipantView
        for _, participant := range match.Participants {
                displayName := "Unknown Participant"
                if participant.PlayerID != nil {
                        if player, ok := players[*participant.PlayerID]; ok {
                                displayName = player.DisplayName
                        } else if *participant.PlayerID != "" {
                                displayName = "Unknown Player"
                        }
                } else if participant.TeamID != nil {
                        if *participant.TeamID != "" {
                                displayName = fmt.Sprintf("Team %s", *participant.TeamID)
                        } else {
                                displayName = "Team"
                        }
                }

                participantViews = append(participantViews, &pb.MatchParticipantView{
                        Participant: toPBMatchParticipant(participant),
                        DisplayName: displayName,
                })
        }

        return &pb.MatchView{
                Id: match.ID.Hex(),
                Metadata: &pb.MatchMetadata{
                        SeriesId: match.SeriesID,
                        PlayedAt: timestamppb.New(match.PlayedAt),
                },
                Participants: participantViews,
                Result:       toPBMatchResult(match.Result),
        }, nil
}

func convertParticipants(input []*pb.MatchParticipant) ([]repo.MatchParticipant, error) {
        participants := make([]repo.MatchParticipant, len(input))
        for i, participant := range input {
                switch value := participant.GetParticipant().(type) {
                case *pb.MatchParticipant_PlayerId:
                        id := value.PlayerId
                        if id == "" {
                                return nil, fmt.Errorf("player_id is required for participant")
                        }
                        participants[i] = repo.MatchParticipant{PlayerID: &id}
                case *pb.MatchParticipant_TeamId:
                        id := value.TeamId
                        if id == "" {
                                return nil, fmt.Errorf("team_id is required for participant")
                        }
                        participants[i] = repo.MatchParticipant{TeamID: &id}
                default:
                        return nil, fmt.Errorf("participant must include player_id or team_id")
                }
        }
        return participants, nil
}

func convertTableTennisResult(result *pb.MatchResult) (*repo.MatchResult, int32, int32, error) {
        if result == nil {
                return nil, 0, 0, status.Error(codes.InvalidArgument, "RESULT_REQUIRED")
        }

        tableTennis := result.GetTableTennis()
        if tableTennis == nil {
                return nil, 0, 0, status.Error(codes.Unimplemented, "RESULT_NOT_SUPPORTED")
        }

        games := tableTennis.GetGamesWon()
        if len(games) != 2 {
                return nil, 0, 0, status.Error(codes.InvalidArgument, "TABLE_TENNIS_REQUIRES_TWO_SCORES")
        }

        bestOf := tableTennis.GetBestOf()
        if bestOf == 0 {
                bestOf = defaultTableTennisBestOf
        }

        repoResult := &repo.MatchResult{
                TableTennis: &repo.TableTennisResult{
                        BestOf:   bestOf,
                        GamesWon: append([]int32(nil), games...),
                },
        }

        return repoResult, games[0], games[1], nil
}

func validateTableTennisScores(scoreA, scoreB, bestOf int32) error {
        if scoreA < 0 || scoreB < 0 {
                return status.Error(codes.InvalidArgument, "VALIDATION_NEGATIVE_SCORE")
        }
        if scoreA == scoreB {
                return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
        }

        if bestOf != 3 && bestOf != 5 {
                bestOf = defaultTableTennisBestOf
        }
        requiredWins := bestOf/2 + 1

        maxScore := scoreA
        if scoreB > maxScore {
                maxScore = scoreB
        }
        if maxScore < requiredWins {
                return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_FIVE")
        }

        return nil
}

func validateSeriesWindow(series *repo.Series, playedAt time.Time) error {
        start := series.StartsAt.Truncate(24 * time.Hour)
        end := series.EndsAt.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)

        if playedAt.Before(start) || playedAt.After(end) {
                return status.Errorf(codes.InvalidArgument,
                        "match date %v must be between series start date %v and end date %v (inclusive)",
                        playedAt.Format("2006-01-02"),
                        series.StartsAt.Format("2006-01-02"),
                        series.EndsAt.Format("2006-01-02"))
        }

        return nil
}

func ensureSeriesSupportsTableTennis(series *repo.Series) error {
        if series.MatchConfiguration.ScoringProfile != int32(pb.SeriesScoringProfile_SERIES_SCORING_PROFILE_TABLE_TENNIS) {
                return status.Error(codes.Unimplemented, "SCORING_PROFILE_NOT_SUPPORTED")
        }

        if series.MatchConfiguration.ParticipantMode != int32(pb.SeriesParticipantMode_SERIES_PARTICIPANT_MODE_INDIVIDUAL) {
                return status.Error(codes.Unimplemented, "PARTICIPANT_MODE_NOT_SUPPORTED")
        }

        if series.MatchConfiguration.ParticipantsPerSide != 1 {
                return status.Error(codes.InvalidArgument, "PARTICIPANT_COUNT_NOT_SUPPORTED")
        }

        return nil
}

func repoParticipantIDs(participants []repo.MatchParticipant) (string, string) {
        var playerAID, playerBID string
        if len(participants) > 0 && participants[0].PlayerID != nil {
                playerAID = *participants[0].PlayerID
        }
        if len(participants) > 1 && participants[1].PlayerID != nil {
                playerBID = *participants[1].PlayerID
        }
        return playerAID, playerBID
}

func toPBMatchView(match *repo.MatchView) *pb.MatchView {
        var participantViews []*pb.MatchParticipantView
        for _, participant := range match.Participants {
                participantViews = append(participantViews, &pb.MatchParticipantView{
                        Participant: toPBMatchParticipant(participant.Participant),
                        DisplayName: participant.DisplayName,
                })
        }

        var result *pb.MatchResult
        if match.Result.TableTennis != nil || match.Result.Scoreline != nil || match.Result.StrokeCard != nil || match.Result.WeighIn != nil {
                result = toPBMatchResult(&match.Result)
        }

        return &pb.MatchView{
                Id: match.ID,
                Metadata: &pb.MatchMetadata{
                        SeriesId: match.SeriesID,
                        PlayedAt: timestamppb.New(match.PlayedAt),
                },
                Participants: participantViews,
                Result:       result,
        }
}

func toPBMatchParticipant(participant repo.MatchParticipant) *pb.MatchParticipant {
        switch {
        case participant.PlayerID != nil && *participant.PlayerID != "":
                return &pb.MatchParticipant{
                        Participant: &pb.MatchParticipant_PlayerId{PlayerId: *participant.PlayerID},
                }
        case participant.TeamID != nil && *participant.TeamID != "":
                return &pb.MatchParticipant{
                        Participant: &pb.MatchParticipant_TeamId{TeamId: *participant.TeamID},
                }
        default:
                return &pb.MatchParticipant{}
        }
}

func toPBMatchResult(result *repo.MatchResult) *pb.MatchResult {
        if result == nil {
                        return nil
        }

        switch {
        case result.TableTennis != nil:
                games := append([]int32(nil), result.TableTennis.GamesWon...)
                return &pb.MatchResult{
                        Result: &pb.MatchResult_TableTennis{
                                TableTennis: &pb.TableTennisResult{
                                        BestOf:   result.TableTennis.BestOf,
                                        GamesWon: games,
                                },
                        },
                }
        case result.Scoreline != nil:
                scores := append([]int32(nil), result.Scoreline.Scores...)
                return &pb.MatchResult{
                        Result: &pb.MatchResult_Scoreline{
                                Scoreline: &pb.ScorelineResult{Scores: scores},
                        },
                }
        case result.StrokeCard != nil:
                holes := make([]*pb.StrokeCardResult_HoleScore, 0, len(result.StrokeCard.Holes))
                for _, hole := range result.StrokeCard.Holes {
                        holes = append(holes, &pb.StrokeCardResult_HoleScore{
                                Hole:    hole.Hole,
                                Strokes: hole.Strokes,
                        })
                }
                return &pb.MatchResult{
                        Result: &pb.MatchResult_StrokeCard{
                                StrokeCard: &pb.StrokeCardResult{Holes: holes},
                        },
                }
        case result.WeighIn != nil:
                weights := append([]float64(nil), result.WeighIn.IndividualWeightsGrams...)
                return &pb.MatchResult{
                        Result: &pb.MatchResult_WeighIn{
                                WeighIn: &pb.WeighInResult{
                                        TotalWeightGrams:       result.WeighIn.TotalWeightGrams,
                                        IndividualWeightsGrams: weights,
                                },
                        },
                }
        default:
                return nil
        }
}
