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

// APIToken represents an authentication token for a user
type APIToken struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Token      string             `bson:"token"`                  // UUID v4 token
	Email      string             `bson:"email"`                  // User's email address
	PlayerID   primitive.ObjectID `bson:"player_id"`              // Associated player ID
	IssuedAt   time.Time          `bson:"issued_at"`              // When the token was issued
	ExpiresAt  time.Time          `bson:"expires_at"`             // When the token expires
	LastUsedAt *time.Time         `bson:"last_used_at,omitempty"` // Last time token was used
	UserAgent  string             `bson:"user_agent,omitempty"`   // User agent when token was created
	IPAddress  string             `bson:"ip_address,omitempty"`   // IP address when token was created
	Revoked    bool               `bson:"revoked"`                // Whether the token has been revoked
	RevokedAt  *time.Time         `bson:"revoked_at,omitempty"`   // When the token was revoked
}

// MagicLinkToken represents a short-lived token for magic link authentication
type MagicLinkToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Token     string             `bson:"token"`                // UUID v4 token
	Email     string             `bson:"email"`                // Email address to authenticate
	ExpiresAt time.Time          `bson:"expires_at"`           // When the token expires (15 minutes)
	UsedAt    *time.Time         `bson:"used_at,omitempty"`    // When the token was consumed
	CreatedAt time.Time          `bson:"created_at"`           // When the token was created
	IPAddress string             `bson:"ip_address,omitempty"` // IP address when created
}

// TokenRepo handles API token and magic link token operations
type TokenRepo struct {
	apiTokens   *mongo.Collection
	magicTokens *mongo.Collection
}

// NewTokenRepo creates a new token repository
func NewTokenRepo(db *mongo.Database) *TokenRepo {
	repo := &TokenRepo{
		apiTokens:   db.Collection("api_tokens"),
		magicTokens: db.Collection("magic_link_tokens"),
	}

	// Create indexes for efficient lookups
	if err := repo.createIndexes(context.Background()); err != nil {
		fmt.Printf("Failed to create token indexes: %v\n", err)
	}

	return repo
}

// createIndexes creates necessary database indexes
func (r *TokenRepo) createIndexes(ctx context.Context) error {
	// API token indexes
	_, err := r.apiTokens.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "player_id", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // TTL index
		},
	})
	if err != nil {
		return err
	}

	// Magic link token indexes
	_, err = r.magicTokens.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // TTL index
		},
	})

	return err
}

// CreateMagicLinkToken creates a new magic link token
func (r *TokenRepo) CreateMagicLinkToken(ctx context.Context, token, email, ipAddress string) (*MagicLinkToken, error) {
	mlt := &MagicLinkToken{
		ID:        primitive.NewObjectID(),
		Token:     token,
		Email:     email,
		ExpiresAt: time.Now().Add(15 * time.Minute), // 15 minutes expiry
		CreatedAt: time.Now(),
		IPAddress: ipAddress,
	}

	_, err := r.magicTokens.InsertOne(ctx, mlt)
	return mlt, err
}

// GetMagicLinkToken retrieves a magic link token by token string
func (r *TokenRepo) GetMagicLinkToken(ctx context.Context, token string) (*MagicLinkToken, error) {
	var mlt MagicLinkToken
	err := r.magicTokens.FindOne(ctx, bson.M{
		"token":      token,
		"expires_at": bson.M{"$gt": time.Now()},
		"used_at":    bson.M{"$exists": false},
	}).Decode(&mlt)

	if err != nil {
		return nil, err
	}

	return &mlt, nil
}

// ConsumeMagicLinkToken marks a magic link token as used
func (r *TokenRepo) ConsumeMagicLinkToken(ctx context.Context, token string) error {
	now := time.Now()
	_, err := r.magicTokens.UpdateOne(ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{"used_at": now}},
	)
	return err
}

// CreateAPIToken creates a new long-lived API token
func (r *TokenRepo) CreateAPIToken(ctx context.Context, token, email string, playerID primitive.ObjectID, userAgent, ipAddress string) (*APIToken, error) {
	at := &APIToken{
		ID:        primitive.NewObjectID(),
		Token:     token,
		Email:     email,
		PlayerID:  playerID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days expiry
		UserAgent: userAgent,
		IPAddress: ipAddress,
		Revoked:   false,
	}

	_, err := r.apiTokens.InsertOne(ctx, at)
	return at, err
}

// GetAPIToken retrieves an API token by token string
func (r *TokenRepo) GetAPIToken(ctx context.Context, token string) (*APIToken, error) {
	var at APIToken
	err := r.apiTokens.FindOne(ctx, bson.M{
		"token":      token,
		"expires_at": bson.M{"$gt": time.Now()},
		"revoked":    false,
	}).Decode(&at)

	if err != nil {
		return nil, err
	}

	return &at, nil
}

// UpdateLastUsed updates the last used timestamp for an API token
func (r *TokenRepo) UpdateLastUsed(ctx context.Context, token string) error {
	now := time.Now()
	_, err := r.apiTokens.UpdateOne(ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{"last_used_at": now}},
	)
	return err
}

// RevokeAPIToken revokes an API token
func (r *TokenRepo) RevokeAPIToken(ctx context.Context, token string) error {
	now := time.Now()
	_, err := r.apiTokens.UpdateOne(ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{
			"revoked":    true,
			"revoked_at": now,
		}},
	)
	return err
}

// RevokeAllUserTokens revokes all API tokens for a specific user
func (r *TokenRepo) RevokeAllUserTokens(ctx context.Context, email string) error {
	now := time.Now()
	_, err := r.apiTokens.UpdateMany(ctx,
		bson.M{"email": email, "revoked": false},
		bson.M{"$set": bson.M{
			"revoked":    true,
			"revoked_at": now,
		}},
	)
	return err
}

// CleanupExpiredTokens removes expired and used tokens (cleanup job)
func (r *TokenRepo) CleanupExpiredTokens(ctx context.Context) error {
	// Clean up expired magic link tokens
	_, err := r.magicTokens.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})
	if err != nil {
		return err
	}

	// Clean up used magic link tokens older than 1 hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	_, err = r.magicTokens.DeleteMany(ctx, bson.M{
		"used_at": bson.M{"$lt": oneHourAgo},
	})
	if err != nil {
		return err
	}

	// Clean up revoked API tokens older than 7 days
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	_, err = r.apiTokens.DeleteMany(ctx, bson.M{
		"revoked":    true,
		"revoked_at": bson.M{"$lt": sevenDaysAgo},
	})

	return err
}
