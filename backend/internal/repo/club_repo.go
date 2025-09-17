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
