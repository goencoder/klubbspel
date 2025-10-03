package repo

import (
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Series struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ClubID         string             `bson:"club_id"`
	Title          string             `bson:"title"`
	StartsAt       time.Time          `bson:"starts_at"`
	EndsAt         time.Time          `bson:"ends_at"`
	Visibility     int32              `bson:"visibility"` // SeriesVisibility enum value
	Sport          int32              `bson:"sport"`
	Format         int32              `bson:"format"`
	LadderRules    int32              `bson:"ladder_rules"`    // LadderRules enum value (only for LADDER format)
	ScoringProfile int32              `bson:"scoring_profile"` // ScoringProfile enum value
	SetsToPlay     int32              `bson:"sets_to_play"`    // For table tennis: 3 or 5
}

type SeriesRepo struct{ c *mongo.Collection }

func NewSeriesRepo(db *mongo.Database) *SeriesRepo {
	return &SeriesRepo{c: db.Collection("series")}
}

func (r *SeriesRepo) Create(ctx context.Context, clubID, title string, startsAt, endsAt time.Time, visibility int32, sport, format, ladderRules, scoringProfile, setsToPlay int32) (*Series, error) {
	s := &Series{
		ID:             primitive.NewObjectID(),
		ClubID:         clubID,
		Title:          title,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		Visibility:     visibility,
		Sport:          sport,
		Format:         format,
		LadderRules:    ladderRules,
		ScoringProfile: scoringProfile,
		SetsToPlay:     setsToPlay,
	}
	_, err := r.c.InsertOne(ctx, s)
	return s, err
}

func (r *SeriesRepo) List(ctx context.Context, pageSize int32, pageToken string) ([]*Series, string, error) {
	cursor, err := r.c.Find(ctx, bson.M{})
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var series []*Series
	for cursor.Next(ctx) {
		var s Series
		if err := cursor.Decode(&s); err != nil {
			continue
		}
		series = append(series, &s)
	}

	return series, "", nil
}

func (r *SeriesRepo) DistinctSportsByClubIDs(ctx context.Context, clubIDs []string) (map[string][]int32, error) {
	if len(clubIDs) == 0 {
		return map[string][]int32{}, nil
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"club_id": bson.M{"$in": clubIDs}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$club_id"},
			{Key: "sports", Value: bson.D{{Key: "$addToSet", Value: "$sport"}}},
		}}},
	}

	cursor, err := r.c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	results := make(map[string][]int32, len(clubIDs))
	for cursor.Next(ctx) {
		var doc struct {
			ID     string  `bson:"_id"`
			Sports []int32 `bson:"sports"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if doc.ID == "" {
			continue
		}

		sort.Slice(doc.Sports, func(i, j int) bool { return doc.Sports[i] < doc.Sports[j] })
		results[doc.ID] = doc.Sports
	}

	return results, nil
}

type SeriesListFilters struct {
	Sport       *int32
	ClubIDs     []string // Club IDs to filter by
	IncludeOpen bool     // Whether to include open series
}

func (r *SeriesRepo) ListWithCursor(ctx context.Context, pageSize int32, cursor string, filters SeriesListFilters) ([]*Series, bool, bool, error) {
	// Set default page size if invalid
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	filter := bson.M{}

	if filters.Sport != nil {
		filter["sport"] = *filters.Sport
	}

	// Handle club filtering
	if len(filters.ClubIDs) > 0 || filters.IncludeOpen {
		var clubConditions []bson.M

		// Add specific club IDs
		if len(filters.ClubIDs) > 0 {
			clubConditions = append(clubConditions, bson.M{"club_id": bson.M{"$in": filters.ClubIDs}})
		}

		// Add open series condition
		if filters.IncludeOpen {
			clubConditions = append(clubConditions, bson.M{"visibility": 2}) // SERIES_VISIBILITY_OPEN = 2
		}

		if len(clubConditions) == 1 {
			// If only one condition, use it directly
			for k, v := range clubConditions[0] {
				filter[k] = v
			}
		} else {
			// Multiple conditions, use $or
			filter["$or"] = clubConditions
		}
	}

	// Add cursor condition for pagination
	if cursor != "" {
		cursorObjID, err := primitive.ObjectIDFromHex(cursor)
		if err == nil {
			filter["_id"] = bson.M{"$gt": cursorObjID}
		}
	}

	// Get pageSize + 1 items to determine if there's a next page
	opts := options.Find().SetLimit(int64(pageSize + 1)).SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor_result, err := r.c.Find(ctx, filter, opts)
	if err != nil {
		return nil, false, false, err
	}
	defer func() {
		_ = cursor_result.Close(ctx)
	}()

	var series []*Series
	for cursor_result.Next(ctx) {
		var s Series
		if err := cursor_result.Decode(&s); err != nil {
			continue
		}
		series = append(series, &s)
	}

	// Determine pagination info
	hasNext := len(series) > int(pageSize)
	if hasNext && len(series) > int(pageSize) {
		series = series[:pageSize] // Remove the extra item
	}

	hasPrev := cursor != ""

	return series, hasNext, hasPrev, nil
}

func (r *SeriesRepo) FindByID(ctx context.Context, id string) (*Series, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var series Series
	err = r.c.FindOne(ctx, bson.M{"_id": objID}).Decode(&series)
	return &series, err
}

// Update applies partial updates to a series document and returns the updated series
func (r *SeriesRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*Series, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	_, err = r.c.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updates})
	if err != nil {
		return nil, err
	}

	return r.FindByID(ctx, id)
}

// Delete removes a series document by ID
func (r *SeriesRepo) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.c.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

// DeleteByClubID removes all series for a specific club
// This is used when a club is deleted to clean up all club-specific series
func (r *SeriesRepo) DeleteByClubID(ctx context.Context, clubID string) error {
	// Only delete series that are club-specific (not open to all clubs)
	// Assuming visibility 0 = club-specific, visibility 1 = open to all
	_, err := r.c.DeleteMany(ctx, bson.M{
		"club_id":    clubID,
		"visibility": bson.M{"$ne": 1}, // Only delete non-open series
	})
	return err
}
