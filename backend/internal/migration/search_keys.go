package migration

import (
	"context"
	"fmt"
	"log"

	"github.com/goencoder/klubbspel/backend/internal/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AddSearchKeysToPlayers migrates existing players to include search keys
func AddSearchKeysToPlayers(ctx context.Context, db *mongo.Database) error {
	log.Println("Starting search keys migration for players...")

	collection := db.Collection("players")

	// Find all players without search_keys field
	filter := bson.M{
		"search_keys": bson.M{"$exists": false},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find players for migration: %w", err)
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			log.Printf("Failed to close cursor: %v", closeErr)
		}
	}()

	var processed, updated int

	for cursor.Next(ctx) {
		var player bson.M
		if err := cursor.Decode(&player); err != nil {
			log.Printf("Failed to decode player: %v", err)
			continue
		}

		processed++

		// Extract display name
		displayName, ok := player["display_name"].(string)
		if !ok || displayName == "" {
			log.Printf("Player %v has no display_name, skipping", player["_id"])
			continue
		}

		// Generate search keys
		searchKeys := util.GenerateSearchKeys(displayName)

		// Update the player with search keys
		updateFilter := bson.M{"_id": player["_id"]}
		update := bson.M{
			"$set": bson.M{
				"search_keys": bson.M{
					"normalized": searchKeys.Normalized,
					"prefixes":   searchKeys.Prefixes,
					"trigrams":   searchKeys.Trigrams,
					"consonants": searchKeys.Consonants,
					"phonetics":  searchKeys.Phonetics,
				},
			},
		}

		result, err := collection.UpdateOne(ctx, updateFilter, update)
		if err != nil {
			log.Printf("Failed to update player %v: %v", player["_id"], err)
			continue
		}

		if result.ModifiedCount > 0 {
			updated++
			if updated%100 == 0 {
				log.Printf("Updated %d players so far...", updated)
			}
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error during migration: %w", err)
	}

	log.Printf("Search keys migration for players completed: processed %d players, updated %d players", processed, updated)
	return nil
}

// AddSearchKeysToClubs migrates existing clubs to include search keys
func AddSearchKeysToClubs(ctx context.Context, db *mongo.Database) error {
	log.Println("Starting search keys migration for clubs...")

	collection := db.Collection("clubs")

	// Find all clubs without search_keys field
	filter := bson.M{
		"search_keys": bson.M{"$exists": false},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find clubs for migration: %w", err)
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			log.Printf("Failed to close cursor: %v", closeErr)
		}
	}()

	var processed, updated int

	for cursor.Next(ctx) {
		var club bson.M
		if err := cursor.Decode(&club); err != nil {
			log.Printf("Failed to decode club: %v", err)
			continue
		}

		processed++

		// Extract club name
		clubName, ok := club["name"].(string)
		if !ok || clubName == "" {
			log.Printf("Club %v has no name, skipping", club["_id"])
			continue
		}

		// Generate search keys
		searchKeys := util.GenerateSearchKeys(clubName)

		// Update the club with search keys
		updateFilter := bson.M{"_id": club["_id"]}
		update := bson.M{
			"$set": bson.M{
				"search_keys": bson.M{
					"normalized": searchKeys.Normalized,
					"prefixes":   searchKeys.Prefixes,
					"trigrams":   searchKeys.Trigrams,
					"consonants": searchKeys.Consonants,
					"phonetics":  searchKeys.Phonetics,
				},
			},
		}

		result, err := collection.UpdateOne(ctx, updateFilter, update)
		if err != nil {
			log.Printf("Failed to update club %v: %v", club["_id"], err)
			continue
		}

		if result.ModifiedCount > 0 {
			updated++
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error during migration: %w", err)
	}

	log.Printf("Search keys migration for clubs completed: processed %d clubs, updated %d clubs", processed, updated)
	return nil
}

// AddSearchKeysIndexes creates indexes for fuzzy search on search keys
func AddSearchKeysIndexes(ctx context.Context, db *mongo.Database) error {
	log.Println("Creating search keys indexes...")

	// Player indexes
	playersCollection := db.Collection("players")
	playerIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "search_keys.normalized", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.prefixes", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.trigrams", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.consonants", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.phonetics", Value: 1},
			},
		},
	}

	_, err := playersCollection.Indexes().CreateMany(ctx, playerIndexes)
	if err != nil {
		return fmt.Errorf("failed to create player search indexes: %w", err)
	}

	// Club indexes
	clubsCollection := db.Collection("clubs")
	clubIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "search_keys.normalized", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.prefixes", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.trigrams", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.consonants", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "search_keys.phonetics", Value: 1},
			},
		},
	}

	_, err = clubsCollection.Indexes().CreateMany(ctx, clubIndexes)
	if err != nil {
		return fmt.Errorf("failed to create club search indexes: %w", err)
	}

	log.Printf("Search keys indexes created successfully")
	return nil
}

// RunSearchKeysMigration runs the complete search keys migration
func RunSearchKeysMigration(ctx context.Context, db *mongo.Database) error {
	// Add search keys to players
	if err := AddSearchKeysToPlayers(ctx, db); err != nil {
		return fmt.Errorf("failed to add search keys to players: %w", err)
	}

	// Add search keys to clubs
	if err := AddSearchKeysToClubs(ctx, db); err != nil {
		return fmt.Errorf("failed to add search keys to clubs: %w", err)
	}

	// Create indexes
	if err := AddSearchKeysIndexes(ctx, db); err != nil {
		return fmt.Errorf("failed to create search keys indexes: %w", err)
	}

	return nil
}
