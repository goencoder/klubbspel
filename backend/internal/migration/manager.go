package migration

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MigrationStatus represents the status of a migration task
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "pending"
	MigrationStatusRunning    MigrationStatus = "running"
	MigrationStatusCompleted  MigrationStatus = "completed"
	MigrationStatusFailed     MigrationStatus = "failed"
)

// MigrationTask represents a data migration task
type MigrationTask struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Status      MigrationStatus    `bson:"status"`
	StartedAt   *time.Time        `bson:"started_at,omitempty"`
	CompletedAt *time.Time        `bson:"completed_at,omitempty"`
	ErrorMsg    string             `bson:"error_msg,omitempty"`
	LeaseExpiry *time.Time        `bson:"lease_expiry,omitempty"`
	Version     int                `bson:"version"`
}

// MigrationManager handles database migrations with locking
type MigrationManager struct {
	db *mongo.Database
	collection *mongo.Collection
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *mongo.Database) *MigrationManager {
	return &MigrationManager{
		db: db,
		collection: db.Collection("migrations"),
	}
}

// RunMigration executes a migration with distributed locking
func (m *MigrationManager) RunMigration(ctx context.Context, name string, migrationFunc func(ctx context.Context, db *mongo.Database) error) error {
	log.Printf("🔄 Starting migration: %s", name)
	
	// Try to acquire lock
	acquired, err := m.acquireLock(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for migration %s: %w", name, err)
	}
	
	if !acquired {
		log.Printf("✅ Migration %s already completed or running on another instance", name)
		return nil
	}
	
	// Run the migration
	migrationErr := migrationFunc(ctx, m.db)
	
	// Update final status
	completedAt := time.Now()
	filter := bson.M{"name": name}
	if migrationErr != nil {
		update := bson.M{
			"$set": bson.M{
				"status":       MigrationStatusFailed,
				"completed_at": completedAt,
				"error_msg":    migrationErr.Error(),
			},
		}
		m.collection.UpdateOne(ctx, filter, update)
		log.Printf("❌ Migration %s failed: %v", name, migrationErr)
		return fmt.Errorf("migration %s failed: %w", name, migrationErr)
	}
	
	// Mark as completed
	update := bson.M{
		"$set": bson.M{
			"status":       MigrationStatusCompleted,
			"completed_at": completedAt,
		},
	}
	
	m.collection.UpdateOne(ctx, filter, update)
	log.Printf("✅ Migration %s completed successfully", name)
	return nil
}

// acquireLock attempts to acquire a distributed lock for the migration
func (m *MigrationManager) acquireLock(ctx context.Context, name string) (bool, error) {
	now := time.Now()
	
	// Check if migration is already completed
	var existingTask MigrationTask
	err := m.collection.FindOne(ctx, bson.M{"name": name}).Decode(&existingTask)
	if err == nil && existingTask.Status == MigrationStatusCompleted {
		log.Printf("Migration %s already completed", name)
		return false, nil
	}
	
	// Try to create the migration task
	filter := bson.M{"name": name}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":        primitive.NewObjectID(),
			"name":       name,
			"status":     MigrationStatusRunning,
			"started_at": now,
			"version":    1,
		},
	}
	
	opts := options.Update().SetUpsert(true)
	result, err := m.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	// If we created a new document, we got the lock
	return result.UpsertedCount > 0, nil
}