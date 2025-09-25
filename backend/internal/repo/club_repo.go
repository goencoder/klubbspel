package repo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Club struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	SupportedSports []int32            `bson:"supported_sports"`
	
	// Enhanced search functionality
	SearchKeys *SearchKeys `bson:"search_keys,omitempty"`
}

type ClubRepo struct{ c *mongo.Collection }

func NewClubRepo(db *mongo.Database) *ClubRepo {
	return &ClubRepo{c: db.Collection("clubs")}
}

func (r *ClubRepo) Create(ctx context.Context, name string, supportedSports []int32) (*Club, error) {
	club := &Club{
		ID:              primitive.NewObjectID(),
		Name:            name,
		SupportedSports: supportedSports,
	}
	_, err := r.c.InsertOne(ctx, club)
	return club, err
}

func (r *ClubRepo) Upsert(ctx context.Context, name string, supportedSports []int32) (*Club, error) {
	// For simplicity, just create a new club
	return r.Create(ctx, name, supportedSports)
}

func (r *ClubRepo) GetByID(ctx context.Context, id string) (*Club, error) {
	return r.FindByID(ctx, id)
}

func (r *ClubRepo) FindByID(ctx context.Context, id string) (*Club, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var club Club
	err = r.c.FindOne(ctx, bson.M{"_id": objID}).Decode(&club)
	return &club, err
}

func (r *ClubRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*Club, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	if len(updates) > 0 {
		update := bson.M{"$set": updates}
		_, err = r.c.UpdateOne(ctx, bson.M{"_id": objID}, update)
		if err != nil {
			return nil, err
		}
	}

	return r.FindByID(ctx, id)
}

func (r *ClubRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.c.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *ClubRepo) ListWithCursor(ctx context.Context, searchQuery string, pageSize int32, cursorAfter string, cursorBefore string) ([]*Club, string, string, bool, bool, error) {
	fmt.Printf("DEBUG ClubRepo.ListWithCursor: pageSize=%d, searchQuery='%s', cursorAfter='%s', cursorBefore='%s'\n", pageSize, searchQuery, cursorAfter, cursorBefore)

	if pageSize <= 0 || pageSize > 100 {
		fmt.Printf("DEBUG ClubRepo.ListWithCursor: Invalid pageSize %d, setting to 20\n", pageSize)
		pageSize = 20
	}

	filter := bson.M{}
	if searchQuery != "" {
		filter["name"] = bson.M{"$regex": searchQuery, "$options": "i"}
	}

	if cursorAfter != "" {
		afterID, err := primitive.ObjectIDFromHex(cursorAfter)
		if err == nil {
			filter["_id"] = bson.M{"$gt": afterID}
		}
	}

	if cursorBefore != "" {
		beforeID, err := primitive.ObjectIDFromHex(cursorBefore)
		if err == nil {
			filter["_id"] = bson.M{"$lt": beforeID}
		}
	}

	opts := options.Find().SetLimit(int64(pageSize + 1)).SetSort(bson.D{{Key: "_id", Value: 1}})
	cursor, err := r.c.Find(ctx, filter, opts)
	if err != nil {
		fmt.Printf("DEBUG ClubRepo.ListWithCursor: Error in Find: %v\n", err)
		return nil, "", "", false, false, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var clubs []*Club
	for cursor.Next(ctx) {
		var club Club
		if err := cursor.Decode(&club); err != nil {
			continue
		}
		clubs = append(clubs, &club)
	}

	fmt.Printf("DEBUG ClubRepo.ListWithCursor: Found %d clubs, pageSize=%d\n", len(clubs), pageSize)

	hasNext := len(clubs) > int(pageSize)
	if hasNext && len(clubs) > int(pageSize) && pageSize > 0 {
		fmt.Printf("DEBUG ClubRepo.ListWithCursor: Trimming clubs from %d to %d\n", len(clubs), pageSize)
		clubs = clubs[:pageSize]
	}

	hasPrev := cursorAfter != "" || cursorBefore != ""

	var startCursor, endCursor string
	if len(clubs) > 0 {
		startCursor = clubs[0].ID.Hex()
		endCursor = clubs[len(clubs)-1].ID.Hex()
	}

	fmt.Printf("DEBUG ClubRepo.ListWithCursor: Returning %d clubs, hasNext=%t, hasPrev=%t\n", len(clubs), hasNext, hasPrev)
	return clubs, startCursor, endCursor, hasNext, hasPrev, nil
}

func (r *ClubRepo) List(ctx context.Context, query string, pageSize int32, pageToken string) ([]*Club, string, error) {
	filter := bson.M{}
	if query != "" {
		filter["name"] = bson.M{"$regex": query, "$options": "i"}
	}

	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var clubs []*Club
	for cursor.Next(ctx) {
		var club Club
		if err := cursor.Decode(&club); err != nil {
			continue
		}
		clubs = append(clubs, &club)
	}

	return clubs, "", nil
}

// FuzzySearchClubs performs fuzzy search for clubs using precomputed search keys
func (r *ClubRepo) FuzzySearchClubs(ctx context.Context, query string, pageSize int32, pageToken string) ([]*Club, string, bool, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	
	pipeline := mongo.Pipeline{}
	
	// Search functionality
	if query != "" {
		searchConditions := []bson.M{
			// Simple regex search on name
			{"name": bson.M{"$regex": query, "$options": "i"}},
			// If search_keys exist, add fuzzy matching conditions
			{"search_keys.normalized": bson.M{"$regex": query, "$options": "i"}},
			{"search_keys.prefixes": bson.M{"$regex": "^" + query, "$options": "i"}},
			{"search_keys.trigrams": bson.M{"$regex": query, "$options": "i"}},
		}
		
		matchStage := bson.D{{Key: "$match", Value: bson.D{
			{Key: "$or", Value: searchConditions},
		}}}
		pipeline = append(pipeline, matchStage)
		
		// Add scoring stage
		scoringStage := bson.D{
			{Key: "$addFields", Value: bson.D{
				{Key: "score", Value: bson.D{
					{Key: "$cond", Value: bson.D{
						{Key: "if", Value: bson.D{
							{Key: "$regexMatch", Value: bson.D{
								{Key: "input", Value: "$name"},
								{Key: "regex", Value: query},
								{Key: "options", Value: "i"},
							}},
						}},
						{Key: "then", Value: 1.0},
						{Key: "else", Value: 0.7},
					}},
				}},
			}},
		}
		pipeline = append(pipeline, scoringStage)
	}
	
	// Sort by score and then by name
	sortStage := bson.D{{Key: "$sort", Value: bson.D{
		{Key: "score", Value: -1},
		{Key: "name", Value: 1},
		{Key: "_id", Value: 1},
	}}}
	pipeline = append(pipeline, sortStage)
	
	// Handle pagination
	if pageToken != "" {
		if objID, err := primitive.ObjectIDFromHex(pageToken); err == nil {
			paginationStage := bson.D{{Key: "$match", Value: bson.D{
				{Key: "_id", Value: bson.D{{Key: "$gt", Value: objID}}},
			}}}
			pipeline = append(pipeline, paginationStage)
		}
	}
	
	// Limit results (add 1 to check for more pages)
	limitStage := bson.D{{Key: "$limit", Value: int64(pageSize + 1)}}
	pipeline = append(pipeline, limitStage)
	
	// Execute aggregation
	cursor, err := r.c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, "", false, fmt.Errorf("aggregation failed: %w", err)
	}
	defer cursor.Close(ctx)
	
	var clubs []*Club
	for cursor.Next(ctx) {
		var club Club
		if err := cursor.Decode(&club); err != nil {
			continue
		}
		clubs = append(clubs, &club)
	}
	
	// Check for more pages and set next page token
	hasNextPage := len(clubs) > int(pageSize)
	if hasNextPage {
		clubs = clubs[:pageSize]
	}
	
	var nextPageToken string
	if hasNextPage && len(clubs) > 0 {
		nextPageToken = clubs[len(clubs)-1].ID.Hex()
	}
	
	return clubs, nextPageToken, hasNextPage, nil
}
