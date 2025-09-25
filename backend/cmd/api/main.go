package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goencoder/klubbspel/backend/internal/config"
	"github.com/goencoder/klubbspel/backend/internal/migration"
	"github.com/goencoder/klubbspel/backend/internal/mongo"
	"github.com/goencoder/klubbspel/backend/internal/server"
	"github.com/rs/zerolog/log"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

const TEST_ITERATE_COUNTER = 10

func main() {
	// Add immediate startup logging to verify logs are working
	fmt.Println("========================================")
	fmt.Println("🚀 MATCHPOINT BACKEND STARTING UP")
	fmt.Printf("📅 Startup Time: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("🔄 Iteration: %d\n", TEST_ITERATE_COUNTER)
	fmt.Println("========================================")

	log.Info().
		Str("timestamp", time.Now().Format(time.RFC3339)).
		Int("iteration", TEST_ITERATE_COUNTER).
		Msg("🚀 MATCHPOINT BACKEND STARTING UP")

	log.Info().Int("iteration", TEST_ITERATE_COUNTER).Msg("=== BACKEND START ===")
	fmt.Printf("=== BACKEND START - ITERATION %d ===\n", TEST_ITERATE_COUNTER)
	cfg := config.FromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mc, err := mongo.NewClient(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal().Err(err).Msg("mongo")
	}
	defer func() {
		if err := mc.Close(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to close mongo client")
		}
	}()

	// Run database migrations on startup
	if err := runDatabaseMigrations(ctx, mc.DB); err != nil {
		log.Fatal().Err(err).Msg("database migrations failed")
	}

	gs, gw, httpSrv := server.Bootstrap(ctx, cfg, mc)

	// Start gRPC server first
	go func() {
		if err := gs.Serve(); err != nil {
			log.Fatal().Err(err).Msg("grpc")
		}
	}()

	// Wait a moment for gRPC server to be ready, then register gateway handlers
	time.Sleep(100 * time.Millisecond)
	grpcEndpoint := "localhost:9090" // Should match cfg.GRPCAddr without ":"
	if err := gw.RegisterHandlers(ctx, grpcEndpoint); err != nil {
		log.Fatal().Err(err).Msg("Failed to register gateway handlers")
	}

	// Now start the gateway HTTP server with registered handlers
	go func() {
		if err := gw.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("gateway")
		}
	}()
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http")
		}
	}()

	<-ctx.Done()
	if err := gs.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error shutting down gRPC server")
	}
	if err := gw.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("Error shutting down gateway")
	}
	if err := httpSrv.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("Error shutting down HTTP server")
	}
	os.Exit(0)
}

// MigrationDefinition defines a migration task
type MigrationDefinition struct {
	Name        string
	Description string
	Function    func(ctx context.Context, db *mongodriver.Database) error
}

// runDatabaseMigrations executes all registered migrations in order
func runDatabaseMigrations(ctx context.Context, db *mongodriver.Database) error {
	log.Info().Msg("🗄️ Starting database migrations...")

	// Create migration manager
	migrationManager := migration.NewMigrationManager(db)

	// Define all migrations in order
	migrations := []MigrationDefinition{
		{
			Name:        "fuzzy-search-keys",
			Description: "Add search keys for fuzzy matching",
			Function:    migration.RunSearchKeysMigration,
		},
		// Future migrations can be added here in order
	}

	// Run each migration
	for _, migration := range migrations {
		log.Info().
			Str("migration", migration.Name).
			Str("description", migration.Description).
			Msg("🔄 Starting migration")

		if err := migrationManager.RunMigration(ctx, migration.Name, migration.Function); err != nil {
			log.Error().
				Err(err).
				Str("migration", migration.Name).
				Msg("❌ Migration failed")
			return fmt.Errorf("migration '%s' failed: %w", migration.Name, err)
		}

		log.Info().
			Str("migration", migration.Name).
			Msg("✅ Migration completed")
	}

	log.Info().Msg("✅ All database migrations completed successfully")
	return nil
}
