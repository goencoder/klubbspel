package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/goencoder/klubbspel/backend/internal/audit"
	"github.com/goencoder/klubbspel/backend/internal/auth"
	"github.com/goencoder/klubbspel/backend/internal/config"
	"github.com/goencoder/klubbspel/backend/internal/email"
	"github.com/goencoder/klubbspel/backend/internal/middleware"
	"github.com/goencoder/klubbspel/backend/internal/mongo"
	"github.com/goencoder/klubbspel/backend/internal/monitoring"
	"github.com/goencoder/klubbspel/backend/internal/repo"
	"github.com/goencoder/klubbspel/backend/internal/service"
	"github.com/goencoder/klubbspel/backend/internal/validate"
	"github.com/goencoder/klubbspel/backend/internal/validation"
	pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
)

// Bootstrap wires gRPC, gateway, and an extra site mux for /healthz and serving swagger with security hardening
func Bootstrap(ctx context.Context, cfg config.Config, mc *mongo.Client) (*GRPCServer, *Gateway, *http.Server) {
	// Create logger for security components
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize security components
	rateLimiter := middleware.NewRateLimiter(middleware.GetDefaultConfig())
	httpRateLimiter := middleware.NewHTTPRateLimiter(middleware.GetDefaultHTTPConfig())

	// Security headers middleware
	var securityHeaders *middleware.SecurityHeaders
	if cfg.Environment == "development" {
		securityHeaders = middleware.NewSecurityHeaders(middleware.GetDevelopmentConfig())
	} else {
		securityHeaders = middleware.NewSecurityHeaders(middleware.GetSecureConfig())
	}

	// Audit logging
	auditLogger := audit.NewAuditLogger(logger, audit.GetDefaultAuditConfig())

	// Input validation
	validator := validation.NewSecurityValidator(validation.GetDefaultValidationConfig())
	_ = validator // Will be used in service layer

	// Monitoring and health checks
	// metricsCollector := monitoring.NewMetricsCollector(logger, monitoring.GetDefaultMetricsConfig())

	// Create custom health config that disables the health checker's own HTTP server
	// since we handle /healthz in the main site server
	healthConfig := monitoring.GetDefaultHealthConfig()
	healthConfig.HTTPEnabled = false // Disable health checker's HTTP server
	healthChecker := monitoring.NewHealthChecker(logger, healthConfig)

	// Register health checks
	healthChecker.RegisterCheck(monitoring.NewDatabaseHealthCheck("mongodb", "MongoDB connection health", true))
	healthChecker.RegisterCheck(monitoring.NewSystemResourcesHealthCheck("system", "System resource health", false))

	// Repositories
	clubRepo := repo.NewClubRepo(mc.DB)
	playerRepo := repo.NewPlayerRepo(mc.DB)
	seriesRepo := repo.NewSeriesRepo(mc.DB)
	matchRepo := repo.NewMatchRepo(mc.DB, playerRepo)
	tokenRepo := repo.NewTokenRepo(mc.DB)

	// Email service - use configuration from environment
	var emailSvc email.Service

	// Convert SMTP port string to int
	smtpPort, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		panic(fmt.Sprintf("Invalid SMTP port: %v", err))
	}

	if cfg.EmailProvider == "mailhog" || (cfg.Environment == "development" && cfg.EmailProvider == "") {
		// MailHog/SMTP configuration for development
		emailConfig := email.EmailConfig{
			Provider:     email.ProviderMailHog,
			FromName:     cfg.EmailFromName,
			FromEmail:    cfg.EmailFromAddress,
			BaseURL:      cfg.EmailBaseURL,
			SMTPHost:     cfg.SMTPHost,
			SMTPPort:     smtpPort,
			SMTPUsername: cfg.SMTPUsername,
			SMTPPassword: cfg.SMTPPassword,
			SMTPTLSMode:  cfg.SMTPTLSMode,
		}
		emailAdapter, err := email.NewEmailAdapter(emailConfig)
		if err != nil {
			panic(fmt.Sprintf("Failed to create MailHog email adapter: %v", err))
		}
		emailSvc = emailAdapter
	} else if cfg.EmailProvider == "sendgrid" || cfg.Environment == "production" {
		// SendGrid configuration for production
		emailConfig := email.EmailConfig{
			Provider:       email.ProviderSendGrid,
			FromName:       cfg.EmailFromName,
			FromEmail:      cfg.EmailFromAddress,
			BaseURL:        cfg.EmailBaseURL,
			SendGridAPIKey: cfg.SendGridAPIKey,
		}
		emailAdapter, err := email.NewEmailAdapter(emailConfig)
		if err != nil {
			panic(fmt.Sprintf("Failed to create SendGrid email adapter: %v", err))
		}
		emailSvc = emailAdapter
	} else {
		// Mock service for testing
		emailConfig := email.EmailConfig{
			Provider:  email.ProviderMock,
			FromName:  cfg.EmailFromName,
			FromEmail: cfg.EmailFromAddress,
			BaseURL:   cfg.EmailBaseURL,
		}
		emailAdapter, err := email.NewEmailAdapter(emailConfig)
		if err != nil {
			panic(fmt.Sprintf("Failed to create mock email adapter: %v", err))
		}
		emailSvc = emailAdapter
	}

	// Services with security enhancements
	clubSvc := &service.ClubService{Clubs: clubRepo, Players: playerRepo, Series: seriesRepo}
	playerSvc := &service.PlayerService{Players: playerRepo}
	seriesSvc := &service.SeriesService{Series: seriesRepo}
	matchSvc := &service.MatchService{Matches: matchRepo, Players: playerRepo, Series: seriesRepo}
	leaderboardSvc := &service.LeaderboardService{Matches: matchRepo, Players: playerRepo}
	authSvc := &service.AuthService{TokenRepo: tokenRepo, PlayerRepo: playerRepo, EmailSvc: emailSvc}
	clubMembershipSvc := &service.ClubMembershipService{PlayerRepo: playerRepo, ClubRepo: clubRepo}

	// Authentication interceptor with audit logging
	authInterceptor := auth.NewAuthInterceptor(tokenRepo, playerRepo)

	// gRPC Server with security interceptors
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			validate.ValidationInterceptor,
			rateLimiter.UnaryInterceptor(),
			authInterceptor.UnaryInterceptor,
			createAuditInterceptor(auditLogger), // Audit logging provides timing + success metrics
		),
	)
	pb.RegisterClubServiceServer(grpcServer, clubSvc)
	pb.RegisterPlayerServiceServer(grpcServer, playerSvc)
	pb.RegisterSeriesServiceServer(grpcServer, seriesSvc)
	pb.RegisterMatchServiceServer(grpcServer, matchSvc)
	pb.RegisterLeaderboardServiceServer(grpcServer, leaderboardSvc)
	pb.RegisterAuthServiceServer(grpcServer, authSvc)
	pb.RegisterClubMembershipServiceServer(grpcServer, clubMembershipSvc)

	gs := &GRPCServer{s: grpcServer, lis: lis}

	// gRPC Gateway with error handling and header matching
	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(LocalizedErrorHandler()),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			return key, true
		}),
	)

	// Create gateway with allowCORS middleware applied to the mux
	gatewayHandler := allowCORS(mux)

	gateway := &Gateway{
		http: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: gatewayHandler,
		},
		mux:         mux,
		environment: cfg.Environment,
	}

	// Chi router for healthz and swagger with security middleware
	r := chi.NewRouter()

	// Security middleware
	r.Use(securityHeaders.Middleware)
	r.Use(httpRateLimiter.Middleware)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(chimiddleware.RequestID)

	// Health endpoints
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := healthChecker.GetStatus()
		w.Header().Set("Content-Type", "application/json")

		httpStatus := http.StatusOK
		if status.Status == "unhealthy" {
			httpStatus = http.StatusServiceUnavailable
		}

		w.WriteHeader(httpStatus)
		json.NewEncoder(w).Encode(status)
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		status := healthChecker.GetStatus()
		if status.Status == "healthy" || status.Status == "degraded" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
		}
	})

	r.Get("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Health check provides system monitoring - dedicated metrics endpoint removed
	// since metrics are captured in audit logs

	// OpenAPI documentation endpoint
	r.Get("/openapi/pingis.swagger.json", func(w http.ResponseWriter, r *http.Request) {
		swaggerPath := "backend/openapi/pingis.swagger.json"
		if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
			http.Error(w, "Swagger file not found. Run 'make generate' first.", http.StatusNotFound)
			return
		}

		data, err := os.ReadFile(swaggerPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read swagger file: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	httpSrv := &http.Server{
		Addr:    cfg.SiteAddr,
		Handler: r,
		// Security-enhanced server settings
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Log security initialization
	logger.Info().
		Str("environment", cfg.Environment).
		Str("grpc_addr", cfg.GRPCAddr).
		Str("http_addr", cfg.HTTPAddr).
		Str("site_addr", cfg.SiteAddr).
		Msg("Security-hardened server initialized")

	return gs, gateway, httpSrv
}

// createAuditInterceptor creates a gRPC interceptor for audit logging
func createAuditInterceptor(auditLogger *audit.AuditLogger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Log request
		event := audit.AuditEvent{
			Type:     getEventTypeFromMethod(info.FullMethod),
			Action:   info.FullMethod,
			Method:   "gRPC",
			Endpoint: info.FullMethod,
		}

		resp, err := handler(ctx, req)

		// Update event with result
		if err != nil {
			event.Result = "FAILURE"
			event.Message = fmt.Sprintf("gRPC call failed: %v", err)
		} else {
			event.Result = "SUCCESS"
			event.Message = fmt.Sprintf("gRPC call succeeded in %v", time.Since(start))
		}

		auditLogger.LogEvent(ctx, event)

		return resp, err
	}
}

// getEventTypeFromMethod maps gRPC method names to audit event types
func getEventTypeFromMethod(method string) audit.EventType {
	switch {
	case strings.Contains(method, "Auth"):
		return audit.EventAuthTokenValidated
	case strings.Contains(method, "Create"):
		return audit.EventDataCreate
	case strings.Contains(method, "Update"):
		return audit.EventDataUpdate
	case strings.Contains(method, "Delete"):
		return audit.EventDataDelete
	case strings.Contains(method, "List"), strings.Contains(method, "Get"):
		return audit.EventDataRead
	default:
		return audit.EventDataRead
	}
}
