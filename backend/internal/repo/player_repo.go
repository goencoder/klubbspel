package repo

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// safeInt32 safely converts int64 to int32 with overflow protection
// This function helps resolve G115 security issues by preventing integer overflow
func safeInt32(value int64) (int32, error) {
	if value > math.MaxInt32 {
		return 0, fmt.Errorf("value %d exceeds int32 maximum %d", value, math.MaxInt32)
	}
	if value < math.MinInt32 {
		return 0, fmt.Errorf("value %d is below int32 minimum %d", value, math.MinInt32)
	}
	return int32(value), nil
}

// ClubMembership represents a player's membership in a club
type ClubMembership struct {
	ClubID    primitive.ObjectID `bson:"club_id"`
	Role      string             `bson:"role"` // "member", "admin"
	JoinedAt  time.Time          `bson:"joined_at"`
	InvitedBy primitive.ObjectID `bson:"invited_by,omitempty"` // Who invited this player
}

type Player struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	DisplayName string             `bson:"display_name"`
	Active      bool               `bson:"active"`

	// Authentication and multi-club support
	Email           string           `bson:"email"`
	FirstName       string           `bson:"first_name"`
	LastName        string           `bson:"last_name"`
	ClubMemberships []ClubMembership `bson:"club_memberships"`
	IsPlatformOwner bool             `bson:"is_platform_owner"`
	CreatedAt       time.Time        `bson:"created_at"`
	LastLoginAt     *time.Time       `bson:"last_login_at,omitempty"`

	// Enhanced search functionality
	SearchKeys *SearchKeys `bson:"search_keys,omitempty"`
}

// SearchKeys contains precomputed search keys for fuzzy matching
type SearchKeys struct {
	Normalized string   `bson:"normalized"`
	Prefixes   []string `bson:"prefixes"`
	Trigrams   []string `bson:"trigrams"`
	Consonants string   `bson:"consonants"`
	Phonetics  []string `bson:"phonetics"`
}

type PlayerRepo struct{ c *mongo.Collection }

func NewPlayerRepo(db *mongo.Database) *PlayerRepo {
	repo := &PlayerRepo{c: db.Collection("players")}

	// Create indexes for efficient lookups
	if err := repo.createIndexes(context.Background()); err != nil {
		fmt.Printf("Failed to create player indexes: %v\n", err)
	}

	return repo
}

// createIndexes creates necessary database indexes
func (r *PlayerRepo) createIndexes(ctx context.Context) error {
	_, err := r.c.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true), // Unique but allow missing
		},
		{
			Keys: bson.D{{Key: "club_memberships.club_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "club_memberships.role", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "display_name", Value: 1}}, // Index for name searches
		},
		{
			Keys: bson.D{{Key: "search_keys.normalized", Value: 1}}, // Index for fuzzy search
		},
		{
			Keys: bson.D{{Key: "search_keys.prefixes", Value: 1}}, // Index for prefix matching
		},
	})
	return err
}

func (r *PlayerRepo) Create(ctx context.Context, name, clubID string) (*Player, error) {
	// Create player with new club membership structure
	playerID := primitive.NewObjectID()
	p := &Player{
		ID:          playerID,
		DisplayName: name,
		Active:      true,
		// Generate synthetic email for email-less players to satisfy unique constraint
		Email:           fmt.Sprintf("noemail-%s@klubbspel.internal", playerID.Hex()),
		ClubMemberships: []ClubMembership{}, // Initialize empty, will be added via AddClubMembership
		CreatedAt:       time.Now(),
	}

	// If a club ID is provided, add initial membership
	if clubID != "" {
		clubObjID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, fmt.Errorf("invalid club ID: %w", err)
		}

		p.ClubMemberships = []ClubMembership{{
			ClubID:   clubObjID,
			Role:     "member",
			JoinedAt: time.Now(),
		}}
	}

	_, err := r.c.InsertOne(ctx, p)
	return p, err
}

func (r *PlayerRepo) FindSimilar(ctx context.Context, name, clubID string) ([]*Player, error) {
	// Use simple case-insensitive name search for similarity
	filter := bson.M{
		"display_name": bson.M{"$regex": name, "$options": "i"},
	}

	// If club ID is provided, filter by membership in that club
	if clubID != "" {
		clubObjectID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, fmt.Errorf("invalid club ID: %w", err)
		}
		filter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjectID,
			},
		}
	}

	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	for cursor.Next(ctx) {
		var p Player
		if err := cursor.Decode(&p); err != nil {
			continue
		}
		players = append(players, &p)
	}
	return players, nil
}

func (r *PlayerRepo) List(ctx context.Context, query, clubID string, pageSize int32, pageToken string) ([]*Player, string, error) {
	filter := bson.M{}
	if clubID != "" {
		// Check for membership in the specified club
		clubObjectID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, "", fmt.Errorf("invalid club ID: %w", err)
		}
		filter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjectID,
			},
		}
	}
	if query != "" {
		filter["display_name"] = bson.M{"$regex": query, "$options": "i"}
	}

	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	for cursor.Next(ctx) {
		var p Player
		if err := cursor.Decode(&p); err != nil {
			continue
		}
		players = append(players, &p)
	}

	return players, "", nil
}

// PlayerListFilters holds filtering options for listing players
type PlayerListFilters struct {
	SearchQuery string   // Search query for display name
	ClubIDs     []string // Club IDs to filter by
	IncludeOpen bool     // Whether to include players not associated with any club
}

// ListWithCursorAndFilters lists players with advanced filtering, similar to series
func (r *PlayerRepo) ListWithCursorAndFilters(ctx context.Context, pageSize int32, cursorAfter string, cursorBefore string, filters PlayerListFilters) ([]*Player, string, string, bool, bool, error) {
	// Set default page size if invalid
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	filter := bson.M{}

	// Handle search query with fuzzy matching using search keys
	if filters.SearchQuery != "" {
		searchKeys := util.GenerateSearchKeys(filters.SearchQuery)

		// Build fuzzy search conditions that match how the stored search keys work
		var searchConditions []bson.M

		// 1. Exact match on normalized text (highest priority)
		searchConditions = append(searchConditions, bson.M{
			"search_keys.normalized": searchKeys.Normalized,
		})

		// 2. Case-insensitive regex on normalized text for partial matches
		if searchKeys.Normalized != "" {
			searchConditions = append(searchConditions, bson.M{
				"search_keys.normalized": bson.M{"$regex": searchKeys.Normalized, "$options": "i"},
			})
		}

		// 3. Prefix matching - check if any stored prefixes start with our search
		if len(searchKeys.Prefixes) > 0 {
			for _, prefix := range searchKeys.Prefixes {
				searchConditions = append(searchConditions, bson.M{
					"search_keys.prefixes": prefix,
				})
			}
		}

		// 4. Fallback to display_name for backwards compatibility
		searchConditions = append(searchConditions, bson.M{
			"display_name": bson.M{"$regex": filters.SearchQuery, "$options": "i"},
		})

		// Use OR to match any of the search conditions
		filter["$or"] = searchConditions
	}

	// Handle club filtering
	if len(filters.ClubIDs) > 0 || filters.IncludeOpen {
		var clubConditions []bson.M

		// Add specific club IDs
		if len(filters.ClubIDs) > 0 {
			// Convert club IDs to ObjectIDs
			var clubObjectIDs []primitive.ObjectID
			for _, clubID := range filters.ClubIDs {
				if objectID, err := primitive.ObjectIDFromHex(clubID); err == nil {
					clubObjectIDs = append(clubObjectIDs, objectID)
				}
			}

			if len(clubObjectIDs) > 0 {
				clubConditions = append(clubConditions, bson.M{
					"club_memberships": bson.M{
						"$elemMatch": bson.M{
							"club_id": bson.M{"$in": clubObjectIDs},
						},
					},
				})
			}
		}

		// Add open players condition (players with no club memberships)
		if filters.IncludeOpen {
			clubConditions = append(clubConditions, bson.M{
				"$or": []bson.M{
					{"club_memberships": bson.M{"$exists": false}},
					{"club_memberships": bson.M{"$size": 0}},
				},
			})
		}

		if len(clubConditions) == 1 {
			// If only one condition, use it directly
			for k, v := range clubConditions[0] {
				filter[k] = v
			}
		} else if len(clubConditions) > 1 {
			// Multiple conditions, use $or
			filter["$or"] = clubConditions
		}
	}

	// Add cursor filtering logic
	if cursorAfter != "" {
		afterID, err := primitive.ObjectIDFromHex(cursorAfter)
		if err == nil {
			filter["_id"] = bson.M{"$gt": afterID}
		}
	}

	if cursorBefore != "" {
		beforeID, err := primitive.ObjectIDFromHex(cursorBefore)
		if err == nil {
			if existing, exists := filter["_id"]; exists {
				// Combine with existing cursor condition
				if existingMap, ok := existing.(bson.M); ok {
					existingMap["$lt"] = beforeID
				}
			} else {
				filter["_id"] = bson.M{"$lt": beforeID}
			}
		}
	}

	// Create find options with limit and sorting
	opts := options.Find().
		SetLimit(int64(pageSize + 1)).          // +1 to check if there are more results
		SetSort(bson.D{{Key: "_id", Value: 1}}) // Sort by ID for consistent ordering

	cursor, err := r.c.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", "", false, false, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	for cursor.Next(ctx) {
		var player Player
		if err := cursor.Decode(&player); err != nil {
			continue
		}
		players = append(players, &player)
	}

	if err := cursor.Err(); err != nil {
		return nil, "", "", false, false, err
	}

	// Determine pagination info
	hasNextPage := len(players) > int(pageSize)
	hasPreviousPage := cursorAfter != "" || cursorBefore != ""

	// Remove extra item used for pagination check
	if hasNextPage {
		players = players[:pageSize]
	}

	var startCursor, endCursor string
	if len(players) > 0 {
		startCursor = players[0].ID.Hex()
		endCursor = players[len(players)-1].ID.Hex()
	}

	return players, startCursor, endCursor, hasNextPage, hasPreviousPage, nil
}

func (r *PlayerRepo) ListWithCursor(ctx context.Context, searchQuery, clubID string, pageSize int32, cursorAfter string, cursorBefore string) ([]*Player, string, string, bool, bool, error) {
	// Set default page size if invalid
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	filter := bson.M{}
	if clubID != "" {
		// Check for membership in the specified club
		clubObjectID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, "", "", false, false, fmt.Errorf("invalid club ID: %w", err)
		}
		filter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjectID,
			},
		}
	}
	if searchQuery != "" {
		filter["display_name"] = bson.M{"$regex": searchQuery, "$options": "i"}
	}

	// Add cursor filtering logic
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

	if pageSize == 0 {
		pageSize = 20
	}

	// Create find options with limit and sorting
	opts := options.Find().
		SetLimit(int64(pageSize + 1)).          // +1 to check if there are more results
		SetSort(bson.D{{Key: "_id", Value: 1}}) // Sort by ID for consistent ordering

	cursor, err := r.c.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", "", false, false, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	for cursor.Next(ctx) {
		var player Player
		if err := cursor.Decode(&player); err != nil {
			continue
		}
		players = append(players, &player)
	}

	// Check if we have more results than pageSize
	hasNext := len(players) > int(pageSize)
	if hasNext && len(players) > int(pageSize) {
		// Remove the extra item we fetched to check for more results
		players = players[:pageSize]
	}

	// For backward pagination, we need to check if there are items before
	hasPrev := cursorAfter != "" || cursorBefore != ""

	var startCursor, endCursor string
	if len(players) > 0 {
		startCursor = players[0].ID.Hex()
		endCursor = players[len(players)-1].ID.Hex()
	}

	return players, startCursor, endCursor, hasNext, hasPrev, nil
}

func (r *PlayerRepo) FindByID(ctx context.Context, id string) (*Player, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var player Player
	err = r.c.FindOne(ctx, bson.M{"_id": objID}).Decode(&player)
	return &player, err
}

// Update applies partial updates to a player document and returns the updated player
func (r *PlayerRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*Player, error) {
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

// Delete performs a soft delete by setting active=false
func (r *PlayerRepo) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.c.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"active": false}})
	return err
}

// FindByIDs efficiently retrieves multiple players by their IDs in a single database query
func (r *PlayerRepo) FindByIDs(ctx context.Context, ids []string) (map[string]*Player, error) {
	if len(ids) == 0 {
		return make(map[string]*Player), nil
	}

	// Convert string IDs to ObjectIDs
	var objIDs []primitive.ObjectID
	for _, id := range ids {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			// Skip invalid IDs rather than failing the entire operation
			continue
		}
		objIDs = append(objIDs, objID)
	}

	if len(objIDs) == 0 {
		return make(map[string]*Player), nil
	}

	// Query all players at once using $in operator
	filter := bson.M{"_id": bson.M{"$in": objIDs}}
	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	// Build map of ID -> Player for quick lookup
	result := make(map[string]*Player)
	for cursor.Next(ctx) {
		var player Player
		if err := cursor.Decode(&player); err != nil {
			continue // Skip invalid documents
		}
		result[player.ID.Hex()] = &player
	}

	return result, nil
}

// FindByEmail finds a player by email address
func (r *PlayerRepo) FindByEmail(ctx context.Context, email string) (*Player, error) {
	var player Player
	err := r.c.FindOne(ctx, bson.M{"email": email}).Decode(&player)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

// CreateWithEmail creates a new player with email and basic info
func (r *PlayerRepo) CreateWithEmail(ctx context.Context, email, firstName, lastName, displayName string) (*Player, error) {
	// Auto-generate display name if not provided
	if displayName == "" {
		if firstName != "" && lastName != "" {
			displayName = firstName + " " + lastName
		} else if firstName != "" {
			displayName = firstName
		} else {
			displayName = email
		}
	}

	p := &Player{
		ID:              primitive.NewObjectID(),
		DisplayName:     displayName,
		Active:          true,
		Email:           email,
		FirstName:       firstName,
		LastName:        lastName,
		ClubMemberships: []ClubMembership{}, // Start with no club memberships
		IsPlatformOwner: false,
		CreatedAt:       time.Now(),
	}

	_, err := r.c.InsertOne(ctx, p)
	return p, err
}

// UpdateLastLogin updates the last login timestamp for a player
func (r *PlayerRepo) UpdateLastLogin(ctx context.Context, email string) error {
	now := time.Now()
	_, err := r.c.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"last_login_at": now}},
	)
	return err
}

// SetPlatformOwner sets or unsets platform owner status
func (r *PlayerRepo) SetPlatformOwner(ctx context.Context, email string, isPlatformOwner bool) error {
	_, err := r.c.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"is_platform_owner": isPlatformOwner}},
	)
	return err
}

// AddClubMembership adds a club membership to a player
func (r *PlayerRepo) AddClubMembership(ctx context.Context, email string, membership *ClubMembership) error {
	// First, remove any existing membership for this club (to prevent duplicates)
	if _, err := r.c.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$pull": bson.M{"club_memberships": bson.M{"club_id": membership.ClubID}}},
	); err != nil {
		return err
	}

	// Add the new membership
	_, err := r.c.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$push": bson.M{"club_memberships": membership}},
		options.Update().SetUpsert(false),
	)
	return err
}

// UpdateClubMembershipRole updates the role of a club membership
func (r *PlayerRepo) UpdateClubMembershipRole(ctx context.Context, playerID, clubID, role string) error {
	playerObjID, err := primitive.ObjectIDFromHex(playerID)
	if err != nil {
		return err
	}

	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return err
	}

	_, err = r.c.UpdateOne(ctx,
		bson.M{
			"_id":                      playerObjID,
			"club_memberships.club_id": clubObjID,
		},
		bson.M{"$set": bson.M{"club_memberships.$.role": role}},
	)
	return err
}

// RemoveClubMembership removes a club membership completely (for leaving)
func (r *PlayerRepo) RemoveClubMembership(ctx context.Context, email, clubID string) error {
	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return err
	}

	_, err = r.c.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$pull": bson.M{"club_memberships": bson.M{"club_id": clubObjID}}},
	)
	return err
}

// IsClubMember checks if a player is a member of a club
func (r *PlayerRepo) IsClubMember(ctx context.Context, email, clubID string) (bool, error) {
	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return false, err
	}

	count, err := r.c.CountDocuments(ctx, bson.M{
		"email": email,
		"club_memberships": bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjID,
			},
		},
	})

	return count > 0, err
}

// IsClubAdmin checks if a player is an admin of a club
func (r *PlayerRepo) IsClubAdmin(ctx context.Context, email, clubID string) (bool, error) {
	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return false, err
	}

	count, err := r.c.CountDocuments(ctx, bson.M{
		"email": email,
		"club_memberships": bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjID,
				"role":    "admin",
			},
		},
	})

	return count > 0, err
}

// ListClubMembers lists all members of a club
func (r *PlayerRepo) ListClubMembers(ctx context.Context, clubID string, activeOnly bool) ([]*Player, error) {
	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"club_memberships.club_id": clubObjID,
	}

	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	for cursor.Next(ctx) {
		var player Player
		if err := cursor.Decode(&player); err != nil {
			continue
		}
		players = append(players, &player)
	}

	return players, nil
}

// GetPlayerMemberships gets all club memberships for a player
func (r *PlayerRepo) GetPlayerMemberships(ctx context.Context, playerID string, activeOnly bool) ([]ClubMembership, error) {
	player, err := r.FindByID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	return player.ClubMemberships, nil
}

// UpdateProfile updates a player's first name and last name
func (r *PlayerRepo) UpdateProfile(ctx context.Context, email, firstName, lastName string) error {
	filter := bson.M{"email": email}

	// Create the display name from first and last name
	displayName := fmt.Sprintf("%s %s", firstName, lastName)

	update := bson.M{
		"$set": bson.M{
			"first_name":   firstName,
			"last_name":    lastName,
			"display_name": displayName,
		},
	}

	result, err := r.c.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("player with email %s not found", email)
	}

	return nil
}

// GetByEmail is an alias for FindByEmail for consistency
func (r *PlayerRepo) GetByEmail(ctx context.Context, email string) (*Player, error) {
	return r.FindByEmail(ctx, email)
}

// MergePlayer merges source player into target player, updating all references
func (r *PlayerRepo) MergePlayer(ctx context.Context, targetID, sourceID string) (*Player, int32, int32, error) {
	// Convert IDs to ObjectIDs
	targetObjID, err := primitive.ObjectIDFromHex(targetID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid target player ID: %w", err)
	}

	sourceObjID, err := primitive.ObjectIDFromHex(sourceID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid source player ID: %w", err)
	}

	// Fetch both players to validate they exist
	targetPlayer, err := r.FindByID(ctx, targetID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("target player not found: %w", err)
	}

	sourcePlayer, err := r.FindByID(ctx, sourceID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("source player not found: %w", err)
	}

	// Start a transaction to ensure atomicity
	session, err := r.c.Database().Client().StartSession()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	var matchesUpdated, tokensUpdated int32
	var mergedPlayer *Player

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Update all matches that reference the source player
		matchesCollection := r.c.Database().Collection("matches")

		// Update player_a_id references
		resultA, err := matchesCollection.UpdateMany(sc,
			bson.M{"player_a_id": sourceID},
			bson.M{"$set": bson.M{"player_a_id": targetID}})
		if err != nil {
			return fmt.Errorf("failed to update player_a_id references: %w", err)
		}

		// Update player_b_id references
		resultB, err := matchesCollection.UpdateMany(sc,
			bson.M{"player_b_id": sourceID},
			bson.M{"$set": bson.M{"player_b_id": targetID}})
		if err != nil {
			return fmt.Errorf("failed to update player_b_id references: %w", err)
		}

		// Safe conversion with overflow check for G115 security fix
		total := resultA.ModifiedCount + resultB.ModifiedCount
		matchesUpdated, err = safeInt32(total)
		if err != nil {
			return fmt.Errorf("match update count conversion failed: %w", err)
		}

		// Update all API tokens that reference the source player
		tokensCollection := r.c.Database().Collection("api_tokens")
		tokenResult, err := tokensCollection.UpdateMany(sc,
			bson.M{"player_id": sourceObjID},
			bson.M{"$set": bson.M{"player_id": targetObjID}})
		if err != nil {
			return fmt.Errorf("failed to update token references: %w", err)
		}

		// Safe conversion with overflow check for G115 security fix
		tokensUpdated, err = safeInt32(tokenResult.ModifiedCount)
		if err != nil {
			return fmt.Errorf("token update count conversion failed: %w", err)
		}

		// INTELLIGENT FIELD MERGING - Following detailed merge rules:

		// SCALAR FIELDS: If target is empty/zero and source has value -> use source value
		// If both have values -> keep target value (target wins)

		// Email field
		if targetPlayer.Email == "" && sourcePlayer.Email != "" {
			targetPlayer.Email = sourcePlayer.Email
		}

		// First name field
		if targetPlayer.FirstName == "" && sourcePlayer.FirstName != "" {
			targetPlayer.FirstName = sourcePlayer.FirstName
		}

		// Last name field
		if targetPlayer.LastName == "" && sourcePlayer.LastName != "" {
			targetPlayer.LastName = sourcePlayer.LastName
		}

		// Display name - if target is empty, use source
		if targetPlayer.DisplayName == "" && sourcePlayer.DisplayName != "" {
			targetPlayer.DisplayName = sourcePlayer.DisplayName
		}

		// Platform owner status - target wins if both have values, source wins if target is false and source is true
		if !targetPlayer.IsPlatformOwner && sourcePlayer.IsPlatformOwner {
			targetPlayer.IsPlatformOwner = sourcePlayer.IsPlatformOwner
		}

		// Active status - if target is inactive but source is active, make result active
		if !targetPlayer.Active && sourcePlayer.Active {
			targetPlayer.Active = sourcePlayer.Active
		}

		// TIMESTAMP FIELDS: Use most recent for last_login_at
		if sourcePlayer.LastLoginAt != nil {
			if targetPlayer.LastLoginAt == nil || sourcePlayer.LastLoginAt.After(*targetPlayer.LastLoginAt) {
				targetPlayer.LastLoginAt = sourcePlayer.LastLoginAt
			}
		}

		// Created at - use earlier creation date (target's creation date is preserved)
		if sourcePlayer.CreatedAt.Before(targetPlayer.CreatedAt) {
			targetPlayer.CreatedAt = sourcePlayer.CreatedAt
		}

		// ARRAY FIELDS: Merge club memberships intelligently
		// Target memberships are preserved, source memberships are added if not duplicate
		targetMemberships := make(map[string]ClubMembership)
		for _, membership := range targetPlayer.ClubMemberships {
			targetMemberships[membership.ClubID.Hex()] = membership
		}

		// Add source memberships that don't exist in target (target wins for duplicates)
		for _, sourceMembership := range sourcePlayer.ClubMemberships {
			if _, exists := targetMemberships[sourceMembership.ClubID.Hex()]; !exists {
				targetPlayer.ClubMemberships = append(targetPlayer.ClubMemberships, sourceMembership)
			}
		}

		// Update the target player with all merged data
		updateFields := bson.M{
			"email":             targetPlayer.Email,
			"first_name":        targetPlayer.FirstName,
			"last_name":         targetPlayer.LastName,
			"display_name":      targetPlayer.DisplayName,
			"active":            targetPlayer.Active,
			"is_platform_owner": targetPlayer.IsPlatformOwner,
			"club_memberships":  targetPlayer.ClubMemberships,
			"created_at":        targetPlayer.CreatedAt,
		}

		if targetPlayer.LastLoginAt != nil {
			updateFields["last_login_at"] = targetPlayer.LastLoginAt
		}

		_, err = r.c.UpdateOne(sc,
			bson.M{"_id": targetObjID},
			bson.M{"$set": updateFields})
		if err != nil {
			return fmt.Errorf("failed to update target player with merged data: %w", err)
		}

		// Delete the source player
		_, err = r.c.DeleteOne(sc, bson.M{"_id": sourceObjID})
		if err != nil {
			return fmt.Errorf("failed to delete source player: %w", err)
		}

		// Fetch the updated target player
		mergedPlayer, err = r.FindByID(sc, targetID)
		if err != nil {
			return fmt.Errorf("failed to fetch merged player: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, 0, 0, err
	}

	return mergedPlayer, matchesUpdated, tokensUpdated, nil
}

// RemoveAllClubMemberships removes all memberships for a specific club
// This is used when a club is deleted to clean up all player memberships
func (r *PlayerRepo) RemoveAllClubMemberships(ctx context.Context, clubID string) error {
	clubObjID, err := primitive.ObjectIDFromHex(clubID)
	if err != nil {
		return err
	}

	// Remove all memberships for this club from all players
	_, err = r.c.UpdateMany(ctx,
		bson.M{"club_memberships.club_id": clubObjID},
		bson.M{"$pull": bson.M{"club_memberships": bson.M{"club_id": clubObjID}}},
	)
	return err
}

// IsSyntheticEmail checks if an email is a synthetic email for email-less players
func IsSyntheticEmail(email string) bool {
	return strings.HasPrefix(email, "noemail-") && strings.HasSuffix(email, "@klubbspel.internal")
}

// IsEmaillessPlayer checks if a player has no real email (either synthetic or empty)
func IsEmaillessPlayer(email string) bool {
	return email == "" || IsSyntheticEmail(email)
}

// CanPlayerLogin checks if a player can authenticate (has a real email address)
func (p *Player) CanLogin() bool {
	return !IsEmaillessPlayer(p.Email)
}

// HasRealEmail checks if a player has a real email address
func (p *Player) HasRealEmail() bool {
	return !IsEmaillessPlayer(p.Email)
}

// FindByRealEmail finds a player by a real (non-synthetic) email address
func (r *PlayerRepo) FindByRealEmail(ctx context.Context, email string) (*Player, error) {
	if IsEmaillessPlayer(email) {
		return nil, fmt.Errorf("cannot find player by synthetic email")
	}
	return r.FindByEmail(ctx, email)
}

// FindEmaillessPlayersByName finds players without real email addresses by display name
// This is useful for showing merge candidates to authenticated users
func (r *PlayerRepo) FindEmaillessPlayersByName(ctx context.Context, displayName, clubID string) ([]*Player, error) {
	filter := bson.M{
		"display_name": bson.M{"$regex": displayName, "$options": "i"},
		"$or": []bson.M{
			{"email": bson.M{"$regex": "^noemail-.*@klubbspel\\.internal$"}}, // Synthetic emails
			{"email": ""},                       // Empty emails (legacy)
			{"email": bson.M{"$exists": false}}, // Missing email field
		},
		"active": true,
	}

	// Filter by club membership if provided
	if clubID != "" {
		clubObjID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, fmt.Errorf("invalid club ID: %w", err)
		}
		filter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjID,
			},
		}
	}

	cursor, err := r.c.Find(ctx, filter, options.Find().SetLimit(10))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	if err = cursor.All(ctx, &players); err != nil {
		return nil, err
	}

	return players, nil
}

// FindAllEmaillessPlayersInClub finds all email-less players in a specific club
func (r *PlayerRepo) FindAllEmaillessPlayersInClub(ctx context.Context, clubID string) ([]*Player, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"email": bson.M{"$regex": "^noemail-.*@klubbspel\\.internal$"}}, // Synthetic emails
			{"email": ""},                       // Empty emails (legacy)
			{"email": bson.M{"$exists": false}}, // Missing email field
		},
		"active": true,
	}

	// Filter by club membership if provided
	if clubID != "" {
		clubObjID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			return nil, fmt.Errorf("invalid club ID: %w", err)
		}
		filter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjID,
			},
		}
	}

	// Debug: Let's also try to find ALL players in the club first to see what we have
	if clubID != "" {
		debugFilter := bson.M{"active": true}
		clubObjID, _ := primitive.ObjectIDFromHex(clubID)
		debugFilter["club_memberships"] = bson.M{
			"$elemMatch": bson.M{
				"club_id": clubObjID,
			},
		}
		debugCursor, err := r.c.Find(ctx, debugFilter)
		if err == nil {
			defer func() {
				_ = debugCursor.Close(ctx)
			}()
			var allPlayers []*Player
			if err := debugCursor.All(ctx, &allPlayers); err != nil {
				fmt.Printf("DEBUG: Failed to read players in club %s: %v\n", clubID, err)
			} else {
				fmt.Printf("DEBUG: Found %d total players in club %s\n", len(allPlayers), clubID)
				for _, p := range allPlayers {
					fmt.Printf("DEBUG: Player %s (%s) email: '%s', synthetic: %v, empty: %v\n",
						p.DisplayName, p.ID.Hex(), p.Email, IsSyntheticEmail(p.Email), p.Email == "")
				}
			}
		}
	}

	// Remove the limit since we want all email-less players for scoring
	cursor, err := r.c.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var players []*Player
	if err = cursor.All(ctx, &players); err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG: Found %d email-less players with filter: %v\n", len(players), filter)

	return players, nil
}

// FuzzySearchResult represents a player search result with score
type FuzzySearchResult struct {
	Player *Player `bson:"player"`
	Score  float64 `bson:"score"`
}

// FuzzySearchPlayers performs fuzzy search using precomputed search keys
func (r *PlayerRepo) FuzzySearchPlayers(ctx context.Context, query string, clubIDs []string, includeOpen bool, pageSize int32, pageToken string) ([]*FuzzySearchResult, string, bool, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	pipeline := mongo.Pipeline{}

	// Match stage for basic filtering
	matchStage := bson.D{}

	// Club filtering
	if len(clubIDs) > 0 || includeOpen {
		var clubConditions []bson.M

		if len(clubIDs) > 0 {
			var clubObjectIDs []primitive.ObjectID
			for _, clubID := range clubIDs {
				if objectID, err := primitive.ObjectIDFromHex(clubID); err == nil {
					clubObjectIDs = append(clubObjectIDs, objectID)
				}
			}

			if len(clubObjectIDs) > 0 {
				clubConditions = append(clubConditions, bson.M{
					"club_memberships": bson.M{
						"$elemMatch": bson.M{
							"club_id": bson.M{"$in": clubObjectIDs},
						},
					},
				})
			}
		}

		if includeOpen {
			clubConditions = append(clubConditions, bson.M{
				"$or": []bson.M{
					{"club_memberships": bson.M{"$exists": false}},
					{"club_memberships": bson.M{"$size": 0}},
				},
			})
		}

		if len(clubConditions) > 0 {
			matchStage = append(matchStage, bson.E{Key: "$or", Value: clubConditions})
		}
	}

	// Search functionality
	var searchConditions []bson.M

	if query != "" {
		// Simple regex search on display_name
		searchConditions = append(searchConditions, bson.M{
			"display_name": bson.M{"$regex": query, "$options": "i"},
		})

		// If search_keys exist, add fuzzy matching conditions
		searchConditions = append(searchConditions, bson.M{
			"search_keys.normalized": bson.M{"$regex": query, "$options": "i"},
		})

		// Prefix match
		searchConditions = append(searchConditions, bson.M{
			"search_keys.prefixes": bson.M{"$regex": "^" + query, "$options": "i"},
		})

		// Trigram overlap (simplified)
		searchConditions = append(searchConditions, bson.M{
			"search_keys.trigrams": bson.M{"$regex": query, "$options": "i"},
		})
	}

	if len(searchConditions) > 0 {
		matchStage = append(matchStage, bson.E{Key: "$or", Value: searchConditions})
	}

	if len(matchStage) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchStage}})
	}

	// Add scoring stage (simplified scoring)
	if query != "" {
		scoringStage := bson.D{
			{Key: "$addFields", Value: bson.D{
				{Key: "score", Value: bson.D{
					{Key: "$cond", Value: bson.D{
						{Key: "if", Value: bson.D{
							{Key: "$regexMatch", Value: bson.D{
								{Key: "input", Value: "$display_name"},
								{Key: "regex", Value: query},
								{Key: "options", Value: "i"},
							}},
						}},
						{Key: "then", Value: 1.0},
						{Key: "else", Value: 0.5},
					}},
				}},
			}},
		}
		pipeline = append(pipeline, scoringStage)
	}

	// Sort by score and then by display_name
	sortStage := bson.D{{Key: "$sort", Value: bson.D{
		{Key: "score", Value: -1},
		{Key: "display_name", Value: 1},
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

	var results []*FuzzySearchResult
	for cursor.Next(ctx) {
		var doc struct {
			Player
			Score float64 `bson:"score"`
		}

		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		result := &FuzzySearchResult{
			Player: &doc.Player,
			Score:  doc.Score,
		}
		results = append(results, result)
	}

	// Check for more pages and set next page token
	hasNextPage := len(results) > int(pageSize)
	if hasNextPage {
		results = results[:pageSize]
	}

	var nextPageToken string
	if hasNextPage && len(results) > 0 {
		nextPageToken = results[len(results)-1].Player.ID.Hex()
	}

	return results, nextPageToken, hasNextPage, nil
}
