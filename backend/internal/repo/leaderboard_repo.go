package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LeaderboardEntry represents a cached leaderboard entry for a player in a series
type LeaderboardEntry struct {
	SeriesID      string    `bson:"series_id"`
	PlayerID      string    `bson:"player_id"`
	Rank          int32     `bson:"rank"`
	Rating        int32     `bson:"rating"`         // ELO rating or ladder position
	MatchesPlayed int32     `bson:"matches_played"`
	MatchesWon    int32     `bson:"matches_won"`
	MatchesLost   int32     `bson:"matches_lost"`
	GamesWon      int32     `bson:"games_won"`
	GamesLost     int32     `bson:"games_lost"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

type LeaderboardRepo struct {
	c *mongo.Collection
}

func NewLeaderboardRepo(db *mongo.Database) *LeaderboardRepo {
	r := &LeaderboardRepo{
		c: db.Collection("leaderboard"),
	}
	if err := r.createIndexes(context.Background()); err != nil {
		panic(err)
	}
	return r
}

func (r *LeaderboardRepo) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "series_id", Value: 1},
				{Key: "player_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "series_id", Value: 1},
				{Key: "rank", Value: 1},
			},
		},
	}

	_, err := r.c.Indexes().CreateMany(ctx, indexes)
	return err
}

// UpsertEntry creates or updates a leaderboard entry
func (r *LeaderboardRepo) UpsertEntry(ctx context.Context, entry *LeaderboardEntry) error {
	filter := bson.M{
		"series_id": entry.SeriesID,
		"player_id": entry.PlayerID,
	}

	update := bson.M{
		"$set": bson.M{
			"rank":           entry.Rank,
			"rating":         entry.Rating,
			"matches_played": entry.MatchesPlayed,
			"matches_won":    entry.MatchesWon,
			"matches_lost":   entry.MatchesLost,
			"games_won":      entry.GamesWon,
			"games_lost":     entry.GamesLost,
			"updated_at":     entry.UpdatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.c.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindBySeriesOrdered returns leaderboard entries sorted by rank
func (r *LeaderboardRepo) FindBySeriesOrdered(ctx context.Context, seriesID string) ([]*LeaderboardEntry, error) {
	filter := bson.M{"series_id": seriesID}
	opts := options.Find().SetSort(bson.D{{Key: "rank", Value: 1}})

	cursor, err := r.c.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []*LeaderboardEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// DeleteAllForSeries removes all leaderboard entries for a series
func (r *LeaderboardRepo) DeleteAllForSeries(ctx context.Context, seriesID string) error {
	_, err := r.c.DeleteMany(ctx, bson.M{"series_id": seriesID})
	return err
}
