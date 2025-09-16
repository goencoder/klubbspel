package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
)

type Gateway struct {
	http        *http.Server
	mux         *runtime.ServeMux
	environment string
}

func (g *Gateway) ListenAndServe() error              { return g.http.ListenAndServe() }
func (g *Gateway) Shutdown(ctx context.Context) error { return g.http.Shutdown(ctx) }

// RegisterHandlers registers all gRPC gateway handlers with the mux
func (g *Gateway) RegisterHandlers(ctx context.Context, grpcEndpoint string) error {
	var opts []grpc.DialOption

	// Use secure credentials in production, insecure for development/testing
	if g.environment == "production" {
		// Use TLS credentials for production
		creds := credentials.NewTLS(&tls.Config{ServerName: "localhost"})
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
		log.Info().Msg("Using TLS credentials for gRPC gateway connection")
	} else {
		// Use insecure credentials for development/testing
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		log.Warn().Msg("Using insecure credentials for gRPC gateway connection (development mode)")
	}

	// Register all service handlers
	if err := pb.RegisterClubServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register ClubService: %w", err)
	}
	if err := pb.RegisterPlayerServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register PlayerService: %w", err)
	}
	if err := pb.RegisterSeriesServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register SeriesService: %w", err)
	}
	if err := pb.RegisterMatchServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register MatchService: %w", err)
	}
	if err := pb.RegisterLeaderboardServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register LeaderboardService: %w", err)
	}
	if err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register AuthService: %w", err)
	}
	if err := pb.RegisterClubMembershipServiceHandlerFromEndpoint(ctx, g.mux, grpcEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register ClubMembershipService: %w", err)
	}

	log.Info().Msg("gRPC Gateway handlers registered successfully")
	return nil
}
