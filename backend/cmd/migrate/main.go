package main

import (
	"context"
	"fmt"
	"log"
	"os"
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
	defer client.Close(ctx)

	migration := &Migration{db: client.DB}

	switch migrationName {
	case "single-to-multi-club":
		err = migration.SingleToMultiClub(context.Background())
	case "add-multi-club-indexes":
		err = migration.AddMultiClubIndexes(context.Background())
	case "verify-migration":
		err = migration.VerifyMigration(context.Background())
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
	defer cursor.Close(ctx)

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
			Role:     "member", // Default role for migrated players
			Active:   true,
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
			{Key: "club_memberships.active", Value: 1},
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
