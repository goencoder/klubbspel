.PHONY: generate be.run be.build swagger clean clean-all dev-clean pre-test rebuild validate-db deps tidy tools test lint security deps-update all help
.PHONY: up down logs docker-build test-integration test-api
.PHONY: dev-start dev-stop dev-restart dev-logs dev-status
.PHONY: host-dev host-be host-fe host-mongo host-services host-stop host-restart host-install
.PHONY: fe.install fe.build fe.dev fe.lint

GO ?= go
NODE ?= node
NPM ?= npm
GOBIN ?= $(CURDIR)/bin

# ----------------------------
# Host Development (Fast iteration)
# ----------------------------

# Install frontend dependencies
fe.install:
	cd frontend && $(NPM) install

# Build frontend for production
fe.build:
	cd frontend && $(NPM) run build

# Run frontend development server
fe.dev:
	cd frontend && $(NPM) run dev

# Lint frontend code
fe.lint:
	cd frontend && $(NPM) run lint

# Start full host development environment (MongoDB + MailHog in Docker, BE+FE on host)
host-dev: host-services host-install be.build
	@echo "🚀 Starting host development environment..."
	@echo ""
	@echo "✅ MongoDB and MailHog running in Docker"
	@echo "🔄 Starting backend and frontend on host..."
	@echo ""
	@echo "Backend will start on: http://localhost:8080"
	@echo "Frontend will start on: http://localhost:5000"
	@echo ""
	@echo "💡 Use 'make host-stop' to stop all services"
	@echo "💡 Use 'make host-restart' to restart backend/frontend"
	@$(MAKE) host-be &
	@sleep 2
	@$(MAKE) host-fe &
	@echo "🎉 Development environment ready!"
	@echo ""
	@echo "📊 Services:"
	@echo "  Frontend:      http://localhost:5000"
	@echo "  Backend API:   http://localhost:8080"
	@echo "  MongoDB:       localhost:27017"
	@echo "  MailHog UI:    http://localhost:8025"

# Install all dependencies (backend + frontend)
host-install: deps fe.install
	@echo "📦 All dependencies installed"

# Start only MongoDB in Docker (for host development)
host-mongo:
	@echo "🍃 Starting MongoDB in Docker..."
	@docker compose up -d mongodb
	@echo "✅ MongoDB ready on localhost:27017"

# Start MongoDB and MailHog in Docker (for development with email testing)
host-services:
	@echo "🍃 Starting MongoDB and MailHog in Docker..."
	@docker compose up -d mongodb mailhog
	@echo "✅ Services ready:"
	@echo "  📦 MongoDB:    localhost:27017"
	@echo "  📧 MailHog UI: http://localhost:8025"
	@echo "  📧 SMTP:      localhost:1025"

# Run backend on host (requires MongoDB running)
host-be: be.build
	@echo "🚀 Starting backend on host..."
	@cd backend && \
	MONGO_URI="mongodb://root:pingis123@localhost:27017/pingis?authSource=admin" \
	MONGO_DB="pingis" \
	GRPC_ADDR=":9090" \
	HTTP_ADDR=":8080" \
	SITE_ADDR=":8082" \
	DEFAULT_LOCALE="sv" \
	./bin/api

# Run frontend on host 
host-fe:
	@echo "⚛️ Starting frontend on host..."
	@cd frontend && $(NPM) run dev

# Stop host development environment
host-stop:
	@echo "🛑 Stopping host development environment..."
	@pkill -f "bin/api" || true
	@pkill -f "vite" || true
	@docker compose stop mongodb mailhog || true
	@echo "✅ Host development environment stopped"

# Restart backend and frontend (keep MongoDB running)
host-restart:
	@echo "🔄 Restarting backend and frontend..."
	@pkill -f "bin/api" || true
	@pkill -f "vite" || true
	@sleep 1
	@$(MAKE) host-be &
	@sleep 2
	@$(MAKE) host-fe &
	@echo "✅ Backend and frontend restarted"

# Quick host development test
host-test:
	@echo "🧪 Testing host development environment..."
	@sleep 3
	@curl -s http://localhost:8080/v1/clubs > /dev/null && echo "✅ Backend API is working" || echo "❌ Backend API failed"
	@curl -s http://localhost:5000 > /dev/null && echo "✅ Frontend is working" || echo "❌ Frontend failed"

# Show host development status
host-status:
	@echo "📊 Host Development Status:"
	@echo ""
	@echo "Backend (bin/api):"
	@pgrep -f "bin/api" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f 'bin/api'))" || echo "  ❌ Not running"
	@echo ""
	@echo "Frontend (vite):"
	@pgrep -f "vite" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f 'vite'))" || echo "  ❌ Not running"
	@echo ""
	@echo "MongoDB (Docker):"
	@docker compose ps mongodb | grep -q "Up" && echo "  ✅ Running in Docker" || echo "  ❌ Not running"
	@echo ""
	@echo "MailHog (Docker):"
	@docker compose ps mailhog | grep -q "Up" && echo "  ✅ Running in Docker" || echo "  ❌ Not running"
	@echo ""
	@echo "🌐 URLs:"
	@echo "  Frontend:      http://localhost:5000"
	@echo "  Backend API:   http://localhost:8080"
	@echo "  Backend gRPC:  localhost:9090"
	@echo "  MailHog UI:    http://localhost:8025"

# Build backend binary for host development
host-build: be.build
	@echo "🔨 Backend binary built for host development"

# Generate protobuf code and OpenAPI spec
generate:
	PATH=$$PATH:~/go/bin buf generate

# Build the backend
be.build: generate
	cd backend && $(GO) build -o bin/api ./cmd/api

# Run the backend
be.run:
	cd backend && $(GO) run ./cmd/api

# Run integration tests with MongoDB
test-integration:
	@echo "Running integration tests..."
	cd tests && ./run-integration-tests.sh

# Start test database only
test-db-up:
	cd tests && docker compose -f docker compose.test.yml up -d mongodb-test

# Stop test database
test-db-down:
	cd tests && docker compose -f docker compose.test.yml down -v

# Test API endpoints (requires running server)
test-api:
	@echo "Testing API endpoints..."
	cd tests && go test ./integration -v -short

# Clean test environment
test-clean:
	cd tests && docker compose -f docker compose.test.yml down -v
	docker system prune -f

# Clean development environment completely
dev-clean:
	docker compose down -v
	docker system prune -f
	cd tests && docker compose -f docker compose.test.yml down -v

# Full clean - everything (build artifacts + containers + volumes)
clean-all: clean test-clean dev-clean
	@echo "🧹 All build artifacts, containers, and volumes cleaned"

# Rebuild from completely clean state
rebuild: clean-all all
	@echo "🔄 Complete rebuild from clean state completed"

# Pre-test clean cycle - run before important tests
pre-test: clean test-clean
	@echo "🧪 Environment cleaned for testing"

# Validate database state for stale data issues
validate-db:
	@./scripts/validate-db-state.sh

# Show the generated swagger file
swagger:
	@ls -la backend/openapi/pingis.swagger.json 2>/dev/null || echo "Swagger file not found. Run 'make generate' first."

# Download Go dependencies
deps:
	cd backend && $(GO) mod download

# Tidy Go modules
tidy:
	cd backend && $(GO) mod tidy

# Clean build artifacts
clean:
	rm -f backend/bin/*
	rm -rf backend/proto/gen/
	rm -f backend/openapi/*.json

# Install required tools (run once)
tools:
	@echo "Installing buf..."
	@which buf > /dev/null || (echo "Please install buf: https://docs.buf.build/installation" && exit 1)
	@echo "Installing Go..."
	@which go > /dev/null || (echo "Please install Go: https://golang.org/dl/" && exit 1)

# Test (placeholder)
test:
	cd backend && $(GO) test ./...

# Lint code with proper linting tools
lint:
	@echo "🔍 Running Go linting with golangci-lint..."
	@mkdir -p $(GOBIN)
	@cd backend && GOBIN=$(GOBIN) GOTOOLCHAIN=go1.25.0 $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
	cd backend && GOTOOLCHAIN=go1.25.0 $(GOBIN)/golangci-lint run ./...
	@echo "✅ Backend linting complete"
	@echo "🔍 Running frontend linting..."
	cd frontend && $(NPM) run lint
	@echo "✅ Frontend linting complete"

# Security scanning
security:
	@echo "🔒 Running security scans..."
	@echo "🔍 Running gosec security scanner..."
	@mkdir -p $(GOBIN)
	@cd backend && GOBIN=$(GOBIN) GOTOOLCHAIN=go1.25.0 $(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	cd backend && $(GOBIN)/gosec ./...
	@echo "🔍 Scanning frontend dependencies..."
	cd frontend && $(NPM) audit --audit-level=moderate
	@echo "✅ Security scans complete"

# Update dependencies (patch and minor versions only)
deps-update:
	@echo "📦 Updating dependencies (patch + minor only)..."
	@echo "🔄 Updating Go dependencies..."
	cd backend && $(GO) get -u=patch ./...
	cd backend && $(GO) mod tidy
	@echo "🔄 Updating frontend dependencies..."
	cd frontend && $(NPM) update
	@echo "✅ Dependencies updated"
	@echo "⚠️  Please test thoroughly after dependency updates!"

# All-in-one: generate, deps, build
all: generate deps be.build

# ----------------------------
# Development Environment
# ----------------------------

# Start development environment
dev-start: up
	@echo "🚀 Development environment started!"
	@echo ""
	@echo "Services available at:"
	@echo "  REST API:      http://localhost:8080"
	@echo "  gRPC:          localhost:9090"
	@echo "  Health/OpenAPI: http://localhost:8081"
	@echo "  MailHog UI:    http://localhost:8025"
	@echo "  MongoDB:       localhost:27017"
	@echo ""
	@echo "Use 'make dev-stop' to stop all services"
	@echo "Use 'make dev-logs' to view logs"

# Stop development environment
dev-stop: down
	@echo "🛑 Development environment stopped"

# Restart development environment
dev-restart: dev-stop dev-start

# View development logs
dev-logs: logs

# Check development environment status
dev-status: status

# Quick API test
dev-test:
	@echo "🧪 Testing API..."
	@curl -s http://localhost:8080/v1/clubs > /dev/null && echo "✅ REST API is working" || echo "❌ REST API failed"
	@curl -s http://localhost:8081/healthz > /dev/null && echo "✅ Health endpoint is working" || echo "❌ Health endpoint failed"

# ----------------------------
# Docker Compose Commands
# ----------------------------

# Bring up all services in the background
up:
	docker compose up -d

# Tear down all services (the -v option removes volumes, effectively resetting the database state)
down:
	docker compose down -v

# Show logs from all services
logs:
	docker compose logs -f --tail=200

# Show logs from specific service
logs-backend:
	docker compose logs -f backend

logs-mongodb:
	docker compose logs -f mongodb

logs-mailhog:
	docker compose logs -f mailhog

# Build Docker images
docker-build:
	docker compose build

# ----------------------------
# Integration Testing
# ----------------------------

# ----------------------------
# Development helpers
# ----------------------------

# Connect to MongoDB shell
mongo-shell:
	docker compose exec mongodb mongosh -u root -p pingis123 --authenticationDatabase admin pingis

# Reset database (WARNING: deletes all data)
reset-db: down up

# Populate development environment with test data
dev-populate:
	@echo "Populating development environment with test data..."
	@./scripts/populate-dev-data.sh

# Show service status
status:
	docker compose ps

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "🚀 Host Development (Recommended for fast iteration):"
	@echo "  host-dev       - Start full dev environment (MongoDB in Docker, BE+FE on host)"
	@echo "  host-be        - Start only backend on host (requires MongoDB)"
	@echo "  host-fe        - Start only frontend on host"
	@echo "  host-mongo     - Start only MongoDB in Docker"
	@echo "  host-stop      - Stop all host development services"
	@echo "  host-restart   - Restart backend and frontend (keep MongoDB)"
	@echo "  host-status    - Show status of all services"
	@echo "  host-test      - Quick test of host development environment"
	@echo "  host-install   - Install all dependencies (backend + frontend)"
	@echo ""
	@echo "⚛️ Frontend:"
	@echo "  fe.install     - Install frontend dependencies"
	@echo "  fe.build       - Build frontend for production"
	@echo "  fe.dev         - Run frontend development server"
	@echo "  fe.lint        - Lint frontend code"
	@echo ""
	@echo "🐳 Docker Development (For production-like environment):"
	@echo "  dev-start      - Start complete development environment (MongoDB, MailHog, Backend)"
	@echo "  dev-stop       - Stop development environment and clean up"
	@echo "  dev-restart    - Restart development environment"
	@echo "  dev-logs       - View logs from all services"
	@echo "  dev-status     - Show service status"
	@echo "  dev-test       - Quick API health check"
	@echo "  dev-populate   - Populate development environment with test data"
	@echo ""
	@echo "🔧 Development:"
	@echo "  generate       - Generate protobuf code and OpenAPI spec"
	@echo "  be.build       - Build the backend binary"
	@echo "  be.run         - Run the backend server locally"
	@echo "  deps           - Download Go dependencies"
	@echo "  tidy           - Tidy Go modules"
	@echo "  test           - Run unit tests"
	@echo "  lint           - Run linting"
	@echo "  all            - Generate, deps, and build"
	@echo ""
	@echo "🐳 Docker Compose (low-level):"
	@echo "  up             - Start all services (use dev-start instead)"
	@echo "  down           - Stop all services and remove volumes"
	@echo "  logs           - Show logs from all services"
	@echo "  logs-backend   - Show backend logs only"
	@echo "  logs-mongodb   - Show MongoDB logs only"
	@echo "  docker-build   - Build Docker images"
	@echo "  status         - Show service status"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  test-integration - Run full integration tests with MongoDB"
	@echo "  test-api       - Test API endpoints (requires running server)"
	@echo "  test-db-up     - Start test database only"
	@echo "  test-db-down   - Stop test database"
	@echo "  test-clean     - Clean test environment"
	@echo "  pre-test       - Clean environment before testing"
	@echo ""
	@echo "🗄️ Database:"
	@echo "  mongo-shell    - Connect to MongoDB shell"
	@echo "  reset-db       - Reset database (WARNING: deletes all data)"
	@echo "  validate-db    - Check database state for stale data issues"
	@echo ""
	@echo "📄 Other:"
	@echo "  swagger        - Show the generated swagger file"
	@echo "  clean          - Clean build artifacts"
	@echo "  clean-all      - Clean everything (build + containers + volumes)"
	@echo "  dev-clean      - Clean development containers and volumes"
	@echo "  rebuild        - Complete rebuild from clean state"
	@echo "  tools          - Check required tools"
	@echo "  help           - Show this help"
