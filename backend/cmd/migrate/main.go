package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	md "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/goencoder/klubbspel/backend/internal/config"
	"github.com/goencoder/klubbspel/backend/internal/mongo"
	"github.com/goencoder/klubbspel/backend/internal/repo"
)

type Migration struct {
	db *md.Database
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate <migration_name>")
	}

	migrationName := os.Args[1]

	// Load configuration
	cfg := config.FromEnv()

	// Connect to MongoDB
	ctx := context.Background()
	client, err := mongo.NewClient(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := client.Close(context.Background()); err != nil {
			log.Printf("Failed to close MongoDB client: %v", err)
		}
	}()

	migration := &Migration{db: client.DB}

	switch migrationName {
	case "single-to-multi-club":
		err = migration.SingleToMultiClub(context.Background())
	case "add-multi-club-indexes":
		err = migration.AddMultiClubIndexes(context.Background())
	case "verify-migration":
		err = migration.VerifyMigration(context.Background())
	case "add-scoring-profiles":
		err = migration.AddScoringProfiles(context.Background())
	case "add-search-keys":
		err = migration.AddSearchKeys(context.Background())
	default:
		log.Fatalf("Unknown migration: %s", migrationName)
	}

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Printf("Migration '%s' completed successfully", migrationName)
}

// SingleToMultiClub migrates existing players from single club_id to club_memberships
func (m *Migration) SingleToMultiClub(ctx context.Context) error {
	log.Println("Starting single-to-multi-club migration...")

	collection := m.db.Collection("players")

	// Find all players with club_id but no club_memberships
	filter := bson.M{
		"club_id": bson.M{"$exists": true, "$ne": ""},
		"$or": []bson.M{
			{"club_memberships": bson.M{"$exists": false}},
			{"club_memberships": bson.M{"$size": 0}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find players for migration: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var processed, updated int

	for cursor.Next(ctx) {
		var player bson.M
		if err := cursor.Decode(&player); err != nil {
			log.Printf("Failed to decode player: %v", err)
			continue
		}

		processed++
		playerID := player["_id"].(primitive.ObjectID)
		clubID := player["club_id"].(string)

		log.Printf("Migrating player %s with club_id %s", playerID.Hex(), clubID)

		// Convert club_id string to ObjectID
		clubObjectID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			log.Printf("Invalid club_id format for player %s: %s", playerID.Hex(), clubID)
			continue
		}

		// Create club membership
		membership := repo.ClubMembership{
			ClubID:   clubObjectID,
			Role:     "member",   // Default role for migrated players
			JoinedAt: time.Now(), // Use current time since we don't have historical data
		}

		// Check if player was a club admin (this would need custom logic based on your existing data)
		// For now, we'll assume all existing players are members

		// Update player with club membership
		update := bson.M{
			"$set": bson.M{
				"club_memberships": []repo.ClubMembership{membership},
			},
		}

		result, err := collection.UpdateOne(ctx, bson.M{"_id": playerID}, update)
		if err != nil {
			log.Printf("Failed to update player %s: %v", playerID.Hex(), err)
			continue
		}

		if result.ModifiedCount > 0 {
			updated++
			log.Printf("Successfully migrated player %s", playerID.Hex())
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error during migration: %w", err)
	}

	log.Printf("Migration completed: processed %d players, updated %d players", processed, updated)
	return nil
}

// AddMultiClubIndexes creates indexes optimized for multi-club queries
func (m *Migration) AddMultiClubIndexes(ctx context.Context) error {
	log.Println("Adding multi-club indexes...")

	collection := m.db.Collection("players")

	// Index for email lookups (authentication)
	emailIndex := md.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	// Compound index for club membership queries
	clubMembershipIndex := md.IndexModel{
		Keys: bson.D{
			{Key: "club_memberships.club_id", Value: 1},
			{Key: "club_memberships.role", Value: 1},
		},
	}

	// Index for platform owner queries
	platformOwnerIndex := md.IndexModel{
		Keys:    bson.D{{Key: "is_platform_owner", Value: 1}},
		Options: options.Index().SetSparse(true),
	}

	// Index for normalized key lookups (duplicate detection)
	normalizedKeyIndex := md.IndexModel{
		Keys: bson.D{
			{Key: "normalized_key", Value: 1},
			{Key: "club_memberships.club_id", Value: 1},
		},
	}

	indexes := []md.IndexModel{
		emailIndex,
		clubMembershipIndex,
		platformOwnerIndex,
		normalizedKeyIndex,
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Successfully created multi-club indexes")
	return nil
}

// VerifyMigration checks that the migration was successful
func (m *Migration) VerifyMigration(ctx context.Context) error {
	log.Println("Verifying migration...")

	collection := m.db.Collection("players")

	// Count players with old club_id but no club_memberships
	unmigrated, err := collection.CountDocuments(ctx, bson.M{
		"club_id": bson.M{"$exists": true, "$ne": ""},
		"$or": []bson.M{
			{"club_memberships": bson.M{"$exists": false}},
			{"club_memberships": bson.M{"$size": 0}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to count unmigrated players: %w", err)
	}

	// Count players with club_memberships
	migrated, err := collection.CountDocuments(ctx, bson.M{
		"club_memberships": bson.M{"$exists": true, "$not": bson.M{"$size": 0}},
	})
	if err != nil {
		return fmt.Errorf("failed to count migrated players: %w", err)
	}

	// Total players
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count total players: %w", err)
	}

	log.Printf("Migration verification results:")
	log.Printf("  Total players: %d", total)
	log.Printf("  Migrated players: %d", migrated)
	log.Printf("  Unmigrated players: %d", unmigrated)

	if unmigrated > 0 {
		log.Printf("WARNING: %d players still need migration", unmigrated)
		return fmt.Errorf("migration incomplete: %d players still need migration", unmigrated)
	}

	log.Println("Migration verification successful: all players migrated")
	return nil
}

// AddScoringProfiles migrates existing series to include scoring_profile and sets_to_play fields
func (m *Migration) AddScoringProfiles(ctx context.Context) error {
	log.Println("Starting scoring profiles migration...")

	collection := m.db.Collection("series")

	// Find all series without scoring_profile field
	filter := bson.M{
		"$or": []bson.M{
			{"scoring_profile": bson.M{"$exists": false}},
			{"sets_to_play": bson.M{"$exists": false}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find series for migration: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var processed, updated int

	for cursor.Next(ctx) {
		var series bson.M
		if err := cursor.Decode(&series); err != nil {
			log.Printf("Failed to decode series: %v", err)
			continue
		}

		processed++
		seriesID := series["_id"].(primitive.ObjectID)
		
		// Get sport value, default to table tennis if not set
		sport := int32(1) // SPORT_TABLE_TENNIS
		if sportVal, exists := series["sport"]; exists {
			if sportInt, ok := sportVal.(int32); ok {
				sport = sportInt
			}
		}

		log.Printf("Migrating series %s with sport %d", seriesID.Hex(), sport)

		update := bson.M{"$set": bson.M{}}
		needsUpdate := false

		// Set scoring_profile if missing
		if _, exists := series["scoring_profile"]; !exists {
			if sport == 1 { // SPORT_TABLE_TENNIS
				update["$set"].(bson.M)["scoring_profile"] = int32(1) // SCORING_PROFILE_TABLE_TENNIS_SETS
				needsUpdate = true
			}
		}

		// Set sets_to_play if missing (default to 5 for table tennis)
		if _, exists := series["sets_to_play"]; !exists {
			if sport == 1 { // SPORT_TABLE_TENNIS
				update["$set"].(bson.M)["sets_to_play"] = int32(5)
				needsUpdate = true
			}
		}

		if needsUpdate {
			result, err := collection.UpdateOne(ctx, bson.M{"_id": seriesID}, update)
			if err != nil {
				log.Printf("Failed to update series %s: %v", seriesID.Hex(), err)
				continue
			}

			if result.ModifiedCount > 0 {
				updated++
				log.Printf("Successfully migrated series %s", seriesID.Hex())
			}
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error during migration: %w", err)
	}

	log.Printf("Scoring profiles migration completed: processed %d series, updated %d series", processed, updated)
	return nil
}

// AddSearchKeys migrates existing players and clubs to include search keys
func (m *Migration) AddSearchKeys(ctx context.Context) error {
	log.Println("Starting search keys migration...")
	
	// Add search keys to players
	if err := m.addSearchKeysToPlayers(ctx); err != nil {
		return fmt.Errorf("failed to add search keys to players: %w", err)
	}
	
	// Add search keys to clubs
	if err := m.addSearchKeysToClubs(ctx); err != nil {
		return fmt.Errorf("failed to add search keys to clubs: %w", err)
	}
	
	// Create indexes for search keys
	if err := m.addSearchKeysIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create search keys indexes: %w", err)
	}
	
	log.Println("Search keys migration completed successfully")
	return nil
}

// addSearchKeysToPlayers migrates existing players to include search keys
func (m *Migration) addSearchKeysToPlayers(ctx context.Context) error {
	log.Println("Adding search keys to players...")
	
	collection := m.db.Collection("players")
	
	// Find all players without search_keys field
	filter := bson.M{
		"search_keys": bson.M{"$exists": false},
	}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find players for migration: %w", err)
	}
	defer cursor.Close(ctx)
	
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
		
		// Generate search keys (simplified version without util package dependency)
		searchKeys := generateSimpleSearchKeys(displayName)
		
		// Update the player with search keys
		updateFilter := bson.M{"_id": player["_id"]}
		update := bson.M{
			"$set": bson.M{
				"search_keys": searchKeys,
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

// addSearchKeysToClubs migrates existing clubs to include search keys
func (m *Migration) addSearchKeysToClubs(ctx context.Context) error {
	log.Println("Adding search keys to clubs...")
	
	collection := m.db.Collection("clubs")
	
	// Find all clubs without search_keys field
	filter := bson.M{
		"search_keys": bson.M{"$exists": false},
	}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find clubs for migration: %w", err)
	}
	defer cursor.Close(ctx)
	
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
		searchKeys := generateSimpleSearchKeys(clubName)
		
		// Update the club with search keys
		updateFilter := bson.M{"_id": club["_id"]}
		update := bson.M{
			"$set": bson.M{
				"search_keys": searchKeys,
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

// addSearchKeysIndexes creates indexes for fuzzy search on search keys
func (m *Migration) addSearchKeysIndexes(ctx context.Context) error {
	log.Println("Creating search keys indexes...")
	
	// Player indexes
	playersCollection := m.db.Collection("players")
	playerIndexes := []md.IndexModel{
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
	}
	
	_, err := playersCollection.Indexes().CreateMany(ctx, playerIndexes)
	if err != nil {
		return fmt.Errorf("failed to create player search indexes: %w", err)
	}
	
	// Club indexes
	clubsCollection := m.db.Collection("clubs")
	clubIndexes := []md.IndexModel{
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
	}
	
	_, err = clubsCollection.Indexes().CreateMany(ctx, clubIndexes)
	if err != nil {
		return fmt.Errorf("failed to create club search indexes: %w", err)
	}
	
	log.Printf("Search keys indexes created successfully")
	return nil
}

// generateSimpleSearchKeys creates basic search keys without external dependencies
func generateSimpleSearchKeys(text string) bson.M {
	normalized := strings.ToLower(text)
	
	// Simple diacritic folding for common Swedish characters
	normalized = strings.ReplaceAll(normalized, "å", "a")
	normalized = strings.ReplaceAll(normalized, "ä", "a")
	normalized = strings.ReplaceAll(normalized, "ö", "o")
	normalized = strings.ReplaceAll(normalized, "é", "e")
	
	// Generate simple prefixes
	var prefixes []string
	words := strings.Fields(normalized)
	for _, word := range words {
		for i := 2; i <= len(word) && i <= 6; i++ {
			prefixes = append(prefixes, word[:i])
		}
	}
	
	// Generate trigrams
	var trigrams []string
	for _, word := range words {
		padded := "  " + word + "  "
		for i := 0; i <= len(padded)-3; i++ {
			trigrams = append(trigrams, padded[i:i+3])
		}
	}
	
	return bson.M{
		"normalized": normalized,
		"prefixes":   prefixes,
		"trigrams":   trigrams,
	}
}
