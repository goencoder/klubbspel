# GitHub Copilot Instructions for Klubbspel

**ALWAYS FOLLOW THESE INSTRUCTIONS FIRST** and only fallback to additional search and context gathering if the information in these instructions is incomplete or found to be in error.

Klubbspel is a table tennis tournament management system with Go backend (gRPC + MongoDB), React TypeScript frontend, and comprehensive testing infrastructure.

## 🚀 Quick Bootstrap & Build (Essential Commands)

Run these commands in order for a fresh clone:

### 1. Install All Dependencies
```bash
# Install buf CLI (required for protobuf generation)
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o ~/buf
chmod +x ~/buf && sudo mv ~/buf /usr/local/bin/

# Install Go protoc plugins  
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest  
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# Install all project dependencies (Backend: ~20 seconds, Frontend: ~40 seconds)
make host-install
```

### 2. Generate Protobuf Code and Build
```bash
# Generate API code - NEVER CANCEL: Can take 30+ seconds depending on network
# Set timeout to 120+ seconds
export PATH=$PATH:$(go env GOPATH)/bin
make generate

# Build backend - NEVER CANCEL: Takes 30-60 seconds for clean build  
# Set timeout to 120+ seconds
make be.build

# Build frontend for production - NEVER CANCEL: Takes 15-30 seconds
# Set timeout to 60+ seconds  
make fe.build
```

### 3. Start Development Environment
```bash
# Start MongoDB + Full Development Stack - NEVER CANCEL: Initial start takes 60+ seconds
# Set timeout to 180+ seconds for first run (Docker image download)
make host-dev

# Alternative: Docker-only development (if host development fails)
make dev-start
```

## ⚡ Development Workflow (Fast Iteration)

### Host Development (Recommended - Fastest)
```bash
# Start (MongoDB in Docker, Backend+Frontend on host)
make host-dev        # NEVER CANCEL: Takes 60+ seconds first time, 10+ seconds subsequent

# Check status 
make host-status     # Shows what's running

# Restart backend/frontend (keep MongoDB running)
make host-restart    # Takes ~5 seconds

# Stop everything
make host-stop       # Takes ~3 seconds
```

### Manual Validation After Changes
**ALWAYS run these end-to-end scenarios after making changes:**

1. **Club Management Workflow**:
   ```bash
   # Ensure services are running first
   make host-status
   
   # Test REST API manually
   curl -X POST http://localhost:8080/v1/clubs \
     -H "Content-Type: application/json" \
     -d '{"name": "Test Club"}'
   
   # View in browser - ALWAYS verify in UI
   # Open: http://localhost:5000
   # Navigate to Clubs page
   # Create a new club via UI
   # Verify it appears in the list
   ```

2. **Player and Series Workflow**:
   ```bash
   # Open: http://localhost:5000
   # 1. Create a club if none exists
   # 2. Navigate to Players page  
   # 3. Create 2+ players for the club
   # 4. Navigate to Series page
   # 5. Create a new series
   # 6. Navigate to series detail  
   # 7. Report a match between players
   # 8. View updated leaderboard
   ```

## 🧪 Testing & Validation (With Timing Expectations)

### Backend Testing
```bash
# Unit tests - NEVER CANCEL: Takes 30-60 seconds  
# Set timeout to 120+ seconds
make test

# Integration tests - NEVER CANCEL: Takes 2-5 minutes
# Set timeout to 10+ minutes (includes database setup/teardown)
make pre-test         # Clean environment first
make test-integration
```

### Frontend Testing  
```bash
# Install Playwright browsers (first time only) - NEVER CANCEL: Takes 5-15 minutes
# Set timeout to 30+ minutes 
cd frontend && npx playwright install

# Run UI tests - NEVER CANCEL: Takes 2-5 minutes
# Set timeout to 10+ minutes
cd frontend && npm run test

# Run in headed mode for debugging
cd frontend && npm run test:headed
```

### Linting & Quality Checks  
```bash
# Backend linting - NEVER CANCEL: Takes 10-30 seconds
# Set timeout to 60+ seconds
make lint

# Frontend linting - NEVER CANCEL: Takes 5-15 seconds  
# Set timeout to 60+ seconds
cd frontend && npm run lint

# Fix linting issues automatically where possible
cd frontend && npm run lint -- --fix
```

## 🔧 Build Timing Expectations & Timeouts

| Command | Expected Time | Recommended Timeout | Never Cancel |
|---------|---------------|-------------------|--------------|
| `make deps` | 20 seconds | 60 seconds | ✅ |
| `make fe.install` | 40 seconds | 120 seconds | ✅ |  
| `make generate` | 30 seconds | 120 seconds | ✅ |
| `make be.build` | 30-60 seconds | 120 seconds | ✅ |
| `make fe.build` | 15-30 seconds | 60 seconds | ✅ |
| `make host-dev` | 60+ seconds (first), 10+ seconds (subsequent) | 180 seconds | ✅ |
| `make test` | 30-60 seconds | 120 seconds | ✅ |
| `make test-integration` | 2-5 minutes | 10 minutes | ✅ |
| `npx playwright install` | 5-15 minutes | 30 minutes | ✅ |
| `npm run test` | 2-5 minutes | 10 minutes | ✅ |

## 📋 Required Quality Checks Before Commits

**ALWAYS run these in order before any commit:**

```bash
# 1. Backend validation 
make lint                              # Must pass
make test                              # Must pass  
make be.build                          # Must compile

# 2. Frontend validation
cd frontend && npm run lint            # Must pass
cd frontend && npm run type-check      # TypeScript validation
cd frontend && npm run build           # Must build

# 3. Integration validation
make pre-test                          # Clean environment
make test-integration                  # Full test suite

# 4. Manual validation - CRITICAL STEP
make host-dev                          # Start full environment
# Open http://localhost:5000 and test complete user workflows
```

## 🚨 Common Issues & Solutions

### Protobuf Generation Fails
```bash
# Network connectivity issues with buf.build
# Solution: Ensure stable internet, retry with longer timeout
export PATH=$PATH:$(go env GOPATH)/bin
buf mod update --timeout=300

# Alternative: Use Docker for complete environment
make dev-start
```

### Backend Build Fails  
```bash
# Missing generated protobuf code
# Always run generation first:
make generate
make be.build
```

### Frontend Linting Errors
```bash
# Fix automatically where possible
cd frontend && npm run lint -- --fix

# Common issues to fix manually:
# - Add curly braces around if statements
# - Remove unused variables  
# - Replace 'any' types with specific types
# - Add missing dependencies to useEffect
```

### Tests Fail Mysteriously
```bash
# Stale data issues - clean environment
make clean-all                         # Nuclear clean
make rebuild                           # Fresh rebuild from scratch
make pre-test && make test-integration # Test with clean slate
```

### Docker Issues
```bash
# Use new docker compose syntax (not docker-compose)
docker compose up -d mongodb          # Not: docker-compose up -d mongodb

# Clean Docker state
docker compose down -v                # Remove volumes  
docker system prune -f               # Clean everything
```

## 🏗 Architecture & Code Organization

### Backend Structure (`backend/`)
- **`cmd/api/`** - Main application entry point
- **`internal/service/`** - Business logic, gRPC service implementations  
- **`internal/repo/`** - Data access layer, MongoDB operations
- **`internal/server/`** - gRPC/HTTP server setup, middleware
- **`internal/config/`** - Configuration management
- **`proto/gen/go/`** - Generated protobuf Go code

### Frontend Structure (`frontend/src/`)
- **`components/`** - Reusable UI components  
- **`pages/`** - Route-level page components
- **`services/`** - API client code
- **`types/`** - TypeScript type definitions
- **`lib/`** - Utility functions and helpers

### Key Technologies
- **Backend**: Go 1.25, gRPC, MongoDB, Protocol Buffers, Zerolog
- **Frontend**: React 19, TypeScript, Vite, TailwindCSS, GitHub Spark
- **Testing**: Go testing, Playwright for UI, MongoDB integration tests
- **Tools**: Buf for protobuf, Docker Compose for development

## 🌐 Service URLs & Endpoints

When development environment is running:

| Service | URL | Purpose |
|---------|-----|---------|
| Frontend | http://localhost:5000 | React development server |
| Backend REST | http://localhost:8080 | gRPC-Gateway REST API |
| Backend gRPC | localhost:9090 | Direct gRPC interface |
| Health Check | http://localhost:8081/healthz | Backend health status |
| OpenAPI Spec | http://localhost:8081/openapi/pingis.swagger.json | API documentation |
| MongoDB | localhost:27017 | Database (root:pingis123) |

## 💡 Best Practices for Code Changes

### Always Follow This Pattern:
1. **Start clean environment**: `make host-dev`
2. **Make minimal changes** to achieve your goal
3. **Test immediately**: Restart relevant services
4. **Validate manually**: Test the affected user workflows in browser  
5. **Run quality checks**: Linting, tests, builds
6. **Manual end-to-end test**: Complete user scenario from browser

### Code Standards:
- **Go**: Include proper error handling, use context, follow repository pattern
- **TypeScript**: Use strict types, avoid 'any', include proper error handling  
- **React**: Use hooks properly, include dependencies in useEffect
- **Testing**: Add tests for new functionality, maintain existing test patterns

### Security Guidelines:
- Validate all inputs on backend
- Sanitize user data
- Use parameterized MongoDB queries
- Don't expose internal errors to frontend
- Use environment variables for configuration

## 🔄 Dependency Management

### Safe Updates (Patch versions):
```bash
# Backend
cd backend && go get -u=patch ./...    # 1.2.3 -> 1.2.4
go mod tidy

# Frontend  
cd frontend && npm update              # Updates within semver range
```

### After Updates - ALWAYS Validate:
```bash
make clean-all                         # Clean everything
make rebuild                           # Fresh rebuild  
make test-integration                  # Full test suite
cd frontend && npm run build           # Frontend build check
```

## 🎯 Manual Validation Checklist

After any significant change, **ALWAYS** manually test these scenarios:

### Scenario 1: Club & Player Management
- [ ] Open http://localhost:5000
- [ ] Navigate to Clubs page  
- [ ] Create a new club
- [ ] Navigate to Players page
- [ ] Create 2 players for the club
- [ ] Verify players appear in list
- [ ] Search for players

### Scenario 2: Tournament Series
- [ ] Navigate to Series page
- [ ] Create a new tournament series  
- [ ] Set start/end dates appropriately
- [ ] Navigate to series detail page
- [ ] Verify series details are correct

### Scenario 3: Match Reporting & Leaderboard  
- [ ] In series detail, report a match
- [ ] Enter scores for 2 players
- [ ] Submit match report
- [ ] Navigate to leaderboard
- [ ] Verify ELO ratings updated correctly
- [ ] Verify match appears in match history

### Scenario 4: API Validation
```bash
# Test REST endpoints directly
curl -X GET http://localhost:8080/v1/clubs
curl -X GET http://localhost:8080/v1/players  
curl -X GET http://localhost:8080/v1/series
```

## 🚦 Environment Variables & Configuration

### Development (Automatic with make host-dev):
```bash
MONGO_URI=mongodb://root:pingis123@localhost:27017/pingis?authSource=admin
MONGO_DB=pingis
GRPC_ADDR=:9090
HTTP_ADDR=:8080  
SITE_ADDR=:8081
DEFAULT_LOCALE=sv
```

### Frontend Build Variables:
```bash
# Create frontend/.env if needed
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_TITLE="Klubbspel"
```

This comprehensive guide ensures consistent, reliable development with proper timing expectations and complete validation workflows. Always follow these instructions for predictable results.
