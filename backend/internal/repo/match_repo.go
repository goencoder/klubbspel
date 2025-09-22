package repo

import (
        "context"
        "fmt"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"
        "go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"
)

const defaultTableTennisBestOf int32 = 5

type MatchParticipant struct {
        PlayerID *string `bson:"player_id,omitempty"`
        TeamID   *string `bson:"team_id,omitempty"`
}

type TableTennisResult struct {
        BestOf   int32   `bson:"best_of"`
        GamesWon []int32 `bson:"games_won"`
}

type ScorelineResult struct {
        Scores []int32 `bson:"scores"`
}

type StrokeCardHoleScore struct {
        Hole    int32 `bson:"hole"`
        Strokes int32 `bson:"strokes"`
}

type StrokeCardResult struct {
        Holes []StrokeCardHoleScore `bson:"holes"`
}

type WeighInResult struct {
        TotalWeightGrams       float64   `bson:"total_weight_grams"`
        IndividualWeightsGrams []float64 `bson:"individual_weights_grams,omitempty"`
}

type MatchResult struct {
        TableTennis *TableTennisResult `bson:"table_tennis,omitempty"`
        Scoreline   *ScorelineResult   `bson:"scoreline,omitempty"`
        StrokeCard  *StrokeCardResult  `bson:"stroke_card,omitempty"`
        WeighIn     *WeighInResult     `bson:"weigh_in,omitempty"`
}

type Match struct {
        ID           primitive.ObjectID `bson:"_id,omitempty"`
        SeriesID     string             `bson:"series_id"`
        PlayedAt     time.Time          `bson:"played_at"`
        Participants []MatchParticipant `bson:"participants,omitempty"`
        Result       *MatchResult       `bson:"result,omitempty"`

        // Legacy fields kept for backward compatibility and migrations.
        PlayerAID string `bson:"player_a_id,omitempty"`
        PlayerBID string `bson:"player_b_id,omitempty"`
        ScoreA    int32  `bson:"score_a,omitempty"`
        ScoreB    int32  `bson:"score_b,omitempty"`
}

type MatchParticipantView struct {
        Participant MatchParticipant
        DisplayName string
}

type MatchView struct {
        ID           string
        SeriesID     string
        PlayedAt     time.Time
        Participants []MatchParticipantView
        Result       MatchResult
}

type MatchRepo struct {
        c       *mongo.Collection
        players *PlayerRepo
}

func NewMatchRepo(db *mongo.Database, players *PlayerRepo) *MatchRepo {
        return &MatchRepo{
                c:       db.Collection("matches"),
                players: players,
        }
}

func (r *MatchRepo) Create(ctx context.Context, seriesID string, playedAt time.Time, participants []MatchParticipant, result MatchResult) (*Match, error) {
        playerAID, playerBID := legacyParticipantIDs(participants)
        scoreA, scoreB := legacyScores(&result)

        match := &Match{
                ID:           primitive.NewObjectID(),
                SeriesID:     seriesID,
                PlayedAt:     playedAt,
                Participants: participants,
                Result:       &result,
                PlayerAID:    playerAID,
                PlayerBID:    playerBID,
                ScoreA:       scoreA,
                ScoreB:       scoreB,
        }

        ensureMatchDefaults(match)

        _, err := r.c.InsertOne(ctx, match)
        return match, err
}

func (r *MatchRepo) ListBySeriesID(ctx context.Context, seriesID string, pageSize int32, pageToken string) ([]*MatchView, string, error) {
        filter := bson.M{"series_id": seriesID}

        if pageSize == 0 {
                pageSize = 20
        }

        if pageToken != "" {
                objID, err := primitive.ObjectIDFromHex(pageToken)
                if err != nil {
                        return nil, "", err
                }
                filter["_id"] = bson.M{"$gt": objID}
        }

        findOptions := options.Find().
                SetLimit(int64(pageSize + 1)).
                SetSort(bson.D{{Key: "played_at", Value: -1}, {Key: "_id", Value: 1}})

        cursor, err := r.c.Find(ctx, filter, findOptions)
        if err != nil {
                return nil, "", err
        }
        defer func() {
                _ = cursor.Close(ctx)
        }()

        var matches []*Match
        playerIDSet := make(map[string]struct{})

        for cursor.Next(ctx) {
                var m Match
                if err := cursor.Decode(&m); err != nil {
                        continue
                }

                ensureMatchDefaults(&m)
                matches = append(matches, &m)

                for _, participant := range m.Participants {
                        if participant.PlayerID != nil && *participant.PlayerID != "" {
                                playerIDSet[*participant.PlayerID] = struct{}{}
                        }
                }
        }

        hasMore := len(matches) > int(pageSize)
        if hasMore {
                matches = matches[:pageSize]
        }

        playerIDs := make([]string, 0, len(playerIDSet))
        for playerID := range playerIDSet {
                playerIDs = append(playerIDs, playerID)
        }

        playersMap, err := r.players.FindByIDs(ctx, playerIDs)
        if err != nil {
                return nil, "", err
        }

        var views []*MatchView
        for _, match := range matches {
                participantViews := make([]MatchParticipantView, 0, len(match.Participants))
                for _, participant := range match.Participants {
                        displayName := "Unknown Participant"
                        if participant.PlayerID != nil {
                                if player, ok := playersMap[*participant.PlayerID]; ok {
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

                        participantViews = append(participantViews, MatchParticipantView{
                                Participant: participant,
                                DisplayName: displayName,
                        })
                }

                viewResult := MatchResult{}
                if match.Result != nil {
                        viewResult = *match.Result
                }

                views = append(views, &MatchView{
                        ID:           match.ID.Hex(),
                        SeriesID:     match.SeriesID,
                        PlayedAt:     match.PlayedAt,
                        Participants: participantViews,
                        Result:       viewResult,
                })
        }

        var nextPageToken string
        if hasMore && len(matches) > 0 {
                nextPageToken = matches[len(matches)-1].ID.Hex()
        }

        return views, nextPageToken, nil
}

func (r *MatchRepo) ListBySeries(ctx context.Context, seriesID string) ([]*Match, error) {
        return r.FindBySeriesID(ctx, seriesID)
}

func (r *MatchRepo) FindBySeriesID(ctx context.Context, seriesID string) ([]*Match, error) {
        cursor, err := r.c.Find(ctx, bson.M{"series_id": seriesID})
        if err != nil {
                        return nil, err
        }
        defer func() {
                _ = cursor.Close(ctx)
        }()

        var matches []*Match
        for cursor.Next(ctx) {
                var m Match
                if err := cursor.Decode(&m); err != nil {
                        continue
                }
                ensureMatchDefaults(&m)
                matches = append(matches, &m)
        }

        return matches, nil
}

func (r *MatchRepo) FindByID(ctx context.Context, matchID string) (*Match, error) {
        objID, err := primitive.ObjectIDFromHex(matchID)
        if err != nil {
                return nil, err
        }

        var match Match
        if err := r.c.FindOne(ctx, bson.M{"_id": objID}).Decode(&match); err != nil {
                return nil, err
        }

        ensureMatchDefaults(&match)
        return &match, nil
}

func (r *MatchRepo) Update(ctx context.Context, matchID string, playedAt *time.Time, result *MatchResult) (*Match, error) {
        objID, err := primitive.ObjectIDFromHex(matchID)
        if err != nil {
                return nil, err
        }

        update := bson.M{}
        if playedAt != nil {
                update["played_at"] = *playedAt
        }
        if result != nil {
                ensureResultDefaults(result)
                update["result"] = result

                scoreA, scoreB := legacyScores(result)
                update["score_a"] = scoreA
                update["score_b"] = scoreB
        }

        if len(update) == 0 {
                return r.FindByID(ctx, matchID)
        }

        _, err = r.c.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
        if err != nil {
                return nil, err
        }

        return r.FindByID(ctx, matchID)
}

func (r *MatchRepo) Delete(ctx context.Context, matchID string) error {
        objID, err := primitive.ObjectIDFromHex(matchID)
        if err != nil {
                return err
        }

        _, err = r.c.DeleteOne(ctx, bson.M{"_id": objID})
        return err
}

func (r *MatchRepo) ReorderMatches(ctx context.Context, matchIDs []string) error {
        if len(matchIDs) < 2 {
                return nil
        }

        matches := make([]*Match, 0, len(matchIDs))
        for _, matchID := range matchIDs {
                match, err := r.FindByID(ctx, matchID)
                if err != nil {
                        return err
                }
                matches = append(matches, match)
        }

        baseDate := matches[0].PlayedAt.Truncate(24 * time.Hour)
        for _, match := range matches {
                if !match.PlayedAt.Truncate(24 * time.Hour).Equal(baseDate) {
                        return mongo.ErrNoDocuments
                }
        }

        for index, matchID := range matchIDs {
                newTime := baseDate.Add(time.Duration(index) * time.Minute)
                objID, err := primitive.ObjectIDFromHex(matchID)
                if err != nil {
                        return err
                }

                if _, err := r.c.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"played_at": newTime}}); err != nil {
                        return err
                }
        }

        return nil
}

func ensureMatchDefaults(match *Match) {
        if match == nil {
                return
        }

        if len(match.Participants) == 0 {
                if match.PlayerAID != "" {
                        id := match.PlayerAID
                        match.Participants = append(match.Participants, MatchParticipant{PlayerID: &id})
                }
                if match.PlayerBID != "" {
                        id := match.PlayerBID
                        match.Participants = append(match.Participants, MatchParticipant{PlayerID: &id})
                }
        }

        if match.Result == nil {
                match.Result = &MatchResult{}
        }

        if match.Result.TableTennis == nil {
                if match.ScoreA != 0 || match.ScoreB != 0 || (match.PlayerAID != "" && match.PlayerBID != "") {
                        match.Result.TableTennis = &TableTennisResult{
                                BestOf:   defaultTableTennisBestOf,
                                GamesWon: []int32{match.ScoreA, match.ScoreB},
                        }
                }
        }

        ensureResultDefaults(match.Result)

        if match.Result.TableTennis != nil {
                if len(match.Result.TableTennis.GamesWon) == 0 {
                        match.Result.TableTennis.GamesWon = []int32{match.ScoreA, match.ScoreB}
                } else if len(match.Result.TableTennis.GamesWon) == 1 {
                        match.Result.TableTennis.GamesWon = append(match.Result.TableTennis.GamesWon, 0)
                }
                match.ScoreA, match.ScoreB = legacyScores(match.Result)
        }

        if match.PlayerAID == "" || match.PlayerBID == "" {
                playerAID, playerBID := legacyParticipantIDs(match.Participants)
                if match.PlayerAID == "" {
                        match.PlayerAID = playerAID
                }
                if match.PlayerBID == "" {
                        match.PlayerBID = playerBID
                }
        }
}

func ensureResultDefaults(result *MatchResult) {
        if result == nil {
                return
        }

        if result.TableTennis != nil {
                if result.TableTennis.BestOf == 0 {
                        result.TableTennis.BestOf = defaultTableTennisBestOf
                }
                if result.TableTennis.GamesWon == nil {
                        result.TableTennis.GamesWon = []int32{0, 0}
                }
        }
}

func legacyParticipantIDs(participants []MatchParticipant) (string, string) {
        var playerAID, playerBID string
        if len(participants) > 0 && participants[0].PlayerID != nil {
                playerAID = *participants[0].PlayerID
        }
        if len(participants) > 1 && participants[1].PlayerID != nil {
                playerBID = *participants[1].PlayerID
        }
        return playerAID, playerBID
}

func legacyScores(result *MatchResult) (int32, int32) {
        if result != nil && result.TableTennis != nil && len(result.TableTennis.GamesWon) >= 2 {
                return result.TableTennis.GamesWon[0], result.TableTennis.GamesWon[1]
        }
        return 0, 0
}
