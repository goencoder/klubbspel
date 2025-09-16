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
	"github.com/goencoder/klubbspel/backend/internal/mongo"
	"github.com/goencoder/klubbspel/backend/internal/server"
	"github.com/rs/zerolog/log"
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
	defer mc.Close(ctx)

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
