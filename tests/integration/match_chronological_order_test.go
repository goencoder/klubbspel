package integration
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/repo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestMatchChronologicalOrder verifies that matches are returned in chronological order
// This is critical for correct ELO calculations
func TestMatchChronologicalOrder(t *testing.T) {
	// Setup test database
	ctx := context.Background()
	
	// Connect to test MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:pingis123@localhost:27017/pingis_test?authSource=admin"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)
	
	db := client.Database("pingis_test")
	
	// Clean up test data
	defer func() {
		db.Collection("matches").Drop(ctx)
		db.Collection("players").Drop(ctx)
		db.Collection("series").Drop(ctx)
		db.Collection("clubs").Drop(ctx)
	}()
	
	// Setup repositories
	clubRepo := repo.NewClubRepo(db)
	playerRepo := repo.NewPlayerRepo(db)
	seriesRepo := repo.NewSeriesRepo(db)
	matchRepo := repo.NewMatchRepo(db, playerRepo)
	
	// Create test data
	clubID, err := clubRepo.Create(ctx, "Test Club", "test@example.com")
	require.NoError(t, err)
	
	playerAID, err := playerRepo.Create(ctx, clubID, "Alice", "alice@example.com")
	require.NoError(t, err)
	
	playerBID, err := playerRepo.Create(ctx, clubID, "Bob", "bob@example.com")
	require.NoError(t, err)
	
	seriesStart := time.Now().Add(-7 * 24 * time.Hour)
	seriesEnd := time.Now().Add(7 * 24 * time.Hour)
	seriesID, err := seriesRepo.Create(ctx, clubID, "Test Tournament", seriesStart, seriesEnd)
	require.NoError(t, err)
	
	// Create matches with specific timestamps
	baseTime := time.Now().Add(-24 * time.Hour)
	
	// Match 1: Earliest
	match1Time := baseTime
	match1, err := matchRepo.Create(ctx, seriesID, playerAID, playerBID, 3, 1, match1Time)
	require.NoError(t, err)
	
	// Match 2: Middle 
	match2Time := baseTime.Add(1 * time.Hour)
	match2, err := matchRepo.Create(ctx, seriesID, playerBID, playerAID, 3, 2, match2Time)
	require.NoError(t, err)
	
	// Match 3: Latest
	match3Time := baseTime.Add(2 * time.Hour)
	match3, err := matchRepo.Create(ctx, seriesID, playerAID, playerBID, 3, 0, match3Time)
	require.NoError(t, err)
	
	t.Logf("Created matches at times: %v, %v, %v", match1Time, match2Time, match3Time)
	
	// Retrieve matches and verify chronological order
	matches, err := matchRepo.FindBySeriesID(ctx, seriesID)
	require.NoError(t, err)
	require.Len(t, matches, 3)
	
	// Verify they are in chronological order (earliest first)
	require.True(t, matches[0].PlayedAt.Equal(match1Time) || matches[0].PlayedAt.Before(match1Time.Add(time.Second)), 
		"First match should be the earliest, got %v, expected %v", matches[0].PlayedAt, match1Time)
	
	require.True(t, matches[1].PlayedAt.Equal(match2Time) || matches[1].PlayedAt.Before(match2Time.Add(time.Second)), 
		"Second match should be the middle, got %v, expected %v", matches[1].PlayedAt, match2Time)
	
	require.True(t, matches[2].PlayedAt.Equal(match3Time) || matches[2].PlayedAt.Before(match3Time.Add(time.Second)), 
		"Third match should be the latest, got %v, expected %v", matches[2].PlayedAt, match3Time)
	
	// Verify chronological progression
	require.True(t, matches[0].PlayedAt.Before(matches[1].PlayedAt) || matches[0].PlayedAt.Equal(matches[1].PlayedAt), 
		"Matches should be in chronological order: %v should be before/equal %v", matches[0].PlayedAt, matches[1].PlayedAt)
	
	require.True(t, matches[1].PlayedAt.Before(matches[2].PlayedAt) || matches[1].PlayedAt.Equal(matches[2].PlayedAt), 
		"Matches should be in chronological order: %v should be before/equal %v", matches[1].PlayedAt, matches[2].PlayedAt)
	
	t.Logf("✅ Matches correctly ordered chronologically:")
	for i, match := range matches {
		t.Logf("  Match %d: ID=%s, Time=%v", i+1, match.ID.Hex(), match.PlayedAt)
	}
}