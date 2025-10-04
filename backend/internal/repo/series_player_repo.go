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

// SeriesPlayer tracks ladder-specific information for players within a series.
type SeriesPlayer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	SeriesID  string             `bson:"series_id"`
	PlayerID  string             `bson:"player_id"`
	Position  int32              `bson:"position"`
	JoinedAt  time.Time          `bson:"joined_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// SeriesPlayerRepo manages ladder player documents.
type SeriesPlayerRepo struct {
	db *mongo.Database
	c  *mongo.Collection
}

// NewSeriesPlayerRepo creates the repository and ensures required indexes exist.
func NewSeriesPlayerRepo(db *mongo.Database) *SeriesPlayerRepo {
	repo := &SeriesPlayerRepo{
		db: db,
		c:  db.Collection("series_players"),
	}

	if err := repo.createIndexes(context.Background()); err != nil {
		fmt.Printf("Failed to create series player indexes: %v\n", err)
	}

	return repo
}

func (r *SeriesPlayerRepo) createIndexes(ctx context.Context) error {
	_, err := r.c.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "series_id", Value: 1}, {Key: "player_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Non-unique index on (series_id, position) for sorting queries
		// Position is just a rank number, not a unique identifier
		{
			Keys: bson.D{{Key: "series_id", Value: 1}, {Key: "position", Value: 1}},
		},
	})
	return err
}

// EnsurePlayer ensures a ladder entry exists for the given series/player pair, creating it at the bottom if missing.
// Uses a retry loop to handle race conditions when multiple players are added simultaneously.
func (r *SeriesPlayerRepo) EnsurePlayer(ctx context.Context, seriesID, playerID string) (*SeriesPlayer, error) {
	existing, err := r.FindBySeriesAndPlayer(ctx, seriesID, playerID)
	if err == nil {
		return existing, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	// Retry up to 5 times to handle race conditions on position allocation
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Determine next position (bottom of ladder).
		var last SeriesPlayer
		opts := options.FindOne().SetSort(bson.D{{Key: "position", Value: -1}})
		position := int32(1)
		if err := r.c.FindOne(ctx, bson.M{"series_id": seriesID}, opts).Decode(&last); err == nil {
			position = last.Position + 1
		}

		now := time.Now().UTC()
		sp := &SeriesPlayer{
			ID:        primitive.NewObjectID(),
			SeriesID:  seriesID,
			PlayerID:  playerID,
			Position:  position,
			JoinedAt:  now,
			UpdatedAt: now,
		}

		if _, err := r.c.InsertOne(ctx, sp); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// Check if it's a duplicate on series_id + player_id (player already exists)
				existing, findErr := r.FindBySeriesAndPlayer(ctx, seriesID, playerID)
				if findErr == nil {
					return existing, nil
				}
				// Otherwise it's a position conflict, retry with new position
				if attempt < maxRetries-1 {
					continue
				}
			}
			return nil, err
		}

		return sp, nil
	}

	return nil, fmt.Errorf("failed to ensure player after %d retries due to position conflicts", maxRetries)
}

// FindBySeriesOrdered returns all players for a series ordered by ascending position.
func (r *SeriesPlayerRepo) FindBySeriesOrdered(ctx context.Context, seriesID string) ([]*SeriesPlayer, error) {
	opts := options.Find().SetSort(bson.D{{Key: "position", Value: 1}})
	cursor, err := r.c.Find(ctx, bson.M{"series_id": seriesID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []*SeriesPlayer
	for cursor.Next(ctx) {
		var sp SeriesPlayer
		if err := cursor.Decode(&sp); err != nil {
			continue
		}
		results = append(results, &sp)
	}
	return results, nil
}

// FindBySeriesAndPlayer retrieves a ladder entry for a specific player.
func (r *SeriesPlayerRepo) FindBySeriesAndPlayer(ctx context.Context, seriesID, playerID string) (*SeriesPlayer, error) {
	var sp SeriesPlayer
	err := r.c.FindOne(ctx, bson.M{"series_id": seriesID, "player_id": playerID}).Decode(&sp)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// FindBySeriesAndPosition returns the player currently occupying the provided position.
func (r *SeriesPlayerRepo) FindBySeriesAndPosition(ctx context.Context, seriesID string, position int32) (*SeriesPlayer, error) {
	var sp SeriesPlayer
	err := r.c.FindOne(ctx, bson.M{"series_id": seriesID, "position": position}).Decode(&sp)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// ShiftPlayersInRange increments positions for all players in the given range.
// Used when a lower-ranked player climbs the ladder by beating a higher-ranked player.
// Since position is no longer unique, we can update all players at once.
func (r *SeriesPlayerRepo) ShiftPlayersInRange(ctx context.Context, seriesID string, minPosition, maxPosition int32, delta int32, now time.Time) error {
	filter := bson.M{
		"series_id": seriesID,
		"position": bson.M{
			"$gte": minPosition,
			"$lt":  maxPosition,
		},
	}
	update := bson.M{
		"$inc": bson.M{"position": delta},
		"$set": bson.M{"updated_at": now},
	}

	_, err := r.c.UpdateMany(ctx, filter, update)
	return err
}

// UpdatePosition sets a player's position and updates the timestamp.
func (r *SeriesPlayerRepo) UpdatePosition(ctx context.Context, seriesID, playerID string, position int32, now time.Time) error {
	_, err := r.c.UpdateOne(ctx, bson.M{
		"series_id": seriesID,
		"player_id": playerID,
	}, bson.M{
		"$set": bson.M{
			"position":   position,
			"updated_at": now,
		},
	})
	return err
}

// TouchPlayer updates only the timestamp for a player without changing position.
func (r *SeriesPlayerRepo) TouchPlayer(ctx context.Context, seriesID, playerID string, now time.Time) error {
	_, err := r.c.UpdateOne(ctx, bson.M{
		"series_id": seriesID,
		"player_id": playerID,
	}, bson.M{
		"$set": bson.M{"updated_at": now},
	})
	return err
}

// DeleteAllForSeries removes all series_players entries for a given series.
// Used when recalculating standings from scratch.
func (r *SeriesPlayerRepo) DeleteAllForSeries(ctx context.Context, seriesID string) error {
	_, err := r.c.DeleteMany(ctx, bson.M{"series_id": seriesID})
	return err
}

// UpsertPlayer creates or updates a series player entry with the given position.
func (r *SeriesPlayerRepo) UpsertPlayer(ctx context.Context, seriesID, playerID string, position int32, now time.Time) error {
	filter := bson.M{
		"series_id": seriesID,
		"player_id": playerID,
	}
	update := bson.M{
		"$set": bson.M{
			"position":   position,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"joined_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.c.UpdateOne(ctx, filter, update, opts)
	return err
}
