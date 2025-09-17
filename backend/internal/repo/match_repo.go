package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Match struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	SeriesID  string             `bson:"series_id"`
	PlayerAID string             `bson:"player_a_id"`
	PlayerBID string             `bson:"player_b_id"`
	ScoreA    int32              `bson:"score_a"`
	ScoreB    int32              `bson:"score_b"`
	PlayedAt  time.Time          `bson:"played_at"`
}

type MatchView struct {
	ID          string    `bson:"_id"`
	SeriesID    string    `bson:"series_id"`
	PlayerAName string    `bson:"player_a_name"`
	PlayerBName string    `bson:"player_b_name"`
	ScoreA      int32     `bson:"score_a"`
	ScoreB      int32     `bson:"score_b"`
	PlayedAt    time.Time `bson:"played_at"`
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

func (r *MatchRepo) Create(ctx context.Context, seriesID, playerAID, playerBID string, scoreA, scoreB int32, playedAt time.Time) (*Match, error) {
	m := &Match{
		ID:        primitive.NewObjectID(),
		SeriesID:  seriesID,
		PlayerAID: playerAID,
		PlayerBID: playerBID,
		ScoreA:    scoreA,
		ScoreB:    scoreB,
		PlayedAt:  playedAt,
	}
	_, err := r.c.InsertOne(ctx, m)
	return m, err
}

func (r *MatchRepo) ListBySeriesID(ctx context.Context, seriesID string, pageSize int32, pageToken string) ([]*MatchView, string, error) {
	filter := bson.M{"series_id": seriesID}

	// Set default page size if not specified
	if pageSize == 0 {
		pageSize = 20
	}

	// Apply cursor-based pagination
	if pageToken != "" {
		objID, err := primitive.ObjectIDFromHex(pageToken)
		if err != nil {
			return nil, "", err
		}
		filter["_id"] = bson.M{"$gt": objID}
	}

	// Apply pagination with limit and sorting
	findOptions := options.Find().
		SetLimit(int64(pageSize + 1)).                                         // +1 to check for more results
		SetSort(bson.D{{Key: "played_at", Value: -1}, {Key: "_id", Value: 1}}) // Sort by played_at descending, then ID ascending

	cursor, err := r.c.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	// First, collect all matches and unique player IDs
	var matches []*Match
	playerIDSet := make(map[string]bool)

	for cursor.Next(ctx) {
		var m Match
		if err := cursor.Decode(&m); err != nil {
			continue
		}
		matches = append(matches, &m)
		playerIDSet[m.PlayerAID] = true
		playerIDSet[m.PlayerBID] = true
	}

	// Check if we have more results than pageSize
	hasMore := len(matches) > int(pageSize)
	if hasMore {
		// Remove the extra item we fetched to check for more results
		matches = matches[:pageSize]
	}

	// Convert player ID set to slice for batch lookup
	playerIDs := make([]string, 0, len(playerIDSet))
	for playerID := range playerIDSet {
		playerIDs = append(playerIDs, playerID)
	}

	// Batch lookup all player names in a single database query
	playersMap, err := r.players.FindByIDs(ctx, playerIDs)
	if err != nil {
		return nil, "", err
	}

	// Build MatchView list with resolved player names
	var matchViews []*MatchView
	for _, m := range matches {
		// Get player names from the map, with fallback for missing players
		playerAName := "Unknown Player"
		playerBName := "Unknown Player"

		if playerA, exists := playersMap[m.PlayerAID]; exists {
			playerAName = playerA.DisplayName
		}

		if playerB, exists := playersMap[m.PlayerBID]; exists {
			playerBName = playerB.DisplayName
		}

		matchView := &MatchView{
			ID:          m.ID.Hex(),
			SeriesID:    m.SeriesID,
			PlayerAName: playerAName,
			PlayerBName: playerBName,
			ScoreA:      m.ScoreA,
			ScoreB:      m.ScoreB,
			PlayedAt:    m.PlayedAt,
		}
		matchViews = append(matchViews, matchView)
	}

	// Set next page token if there are more results
	var nextPageToken string
	if hasMore && len(matchViews) > 0 {
		lastMatch := matches[len(matches)-1]
		nextPageToken = lastMatch.ID.Hex()
	}

	return matchViews, nextPageToken, nil
}

func (r *MatchRepo) ListBySeries(ctx context.Context, seriesID string) ([]*Match, error) {
	return r.FindBySeriesID(ctx, seriesID)
}

func (r *MatchRepo) FindBySeriesID(ctx context.Context, seriesID string) ([]*Match, error) {
	filter := bson.M{"series_id": seriesID}

	cursor, err := r.c.Find(ctx, filter)
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
		matches = append(matches, &m)
	}

	return matches, nil
}
