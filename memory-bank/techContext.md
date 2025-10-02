# Tech Context: Klubbspel v1.0.0

## Production Technology Stack

### Backend Stack (Production-Ready)
- **Go 1.25+**: High-performance backend with excellent concurrency and type safety
- **gRPC**: High-performance RPC framework with protocol buffer integration
- **gRPC-Gateway**: HTTP/JSON REST API gateway for broad client compatibility
- **MongoDB Go Driver**: Official driver with connection pooling and query optimization
- **Protocol Buffers**: Type-safe API contracts with automated code generation
- **Zerolog**: Structured, high-performance JSON logging
- **Buf CLI**: Protocol buffer linting, formatting, breaking change detection, and code generation

### Frontend Stack (Production-Ready)
- **React 19**: Latest React with concurrent rendering and improved performance
- **TypeScript 5.0+**: Strict type checking with advanced type system features
- **Vite**: Lightning-fast build tool with HMR and optimized production bundles
- **TailwindCSS**: Utility-first CSS framework ensuring design consistency
- **i18next**: Complete internationalization with React integration and namespace organization
- **Zustand**: Lightweight, performant state management for global application state
- **React Router v6**: Client-side routing with code splitting and lazy loading

### Development & Quality Tools (Production-Grade)
- **golangci-lint**: Comprehensive Go linting with 50+ linters enabled
- **ESLint**: JavaScript/TypeScript linting with custom rules for code consistency
- **Playwright**: End-to-end browser testing with multi-browser support
- **Docker & Docker Compose**: Containerization for development and production consistency
- **Make**: Build automation orchestrating entire development workflow
- **gosec**: Security-focused static analysis for Go code

### Production Infrastructure
- **Fly.io**: Multi-region application platform with edge deployment
- **MongoDB Atlas**: Managed MongoDB with automated backups, monitoring, and scaling
- **Docker Multi-stage Builds**: Optimized production containers (minimal attack surface)
- **SendGrid**: Reliable email delivery service with tracking and analytics
- **MailHog**: Development SMTP server for local email testing

## Development Setup (Optimized Workflow)

### Prerequisites
```bash
# Go 1.25+ with module support
go version  # Should be 1.25 or higher

# Node.js 18+ with npm
node --version && npm --version  # Node 18+ required

# Docker and Docker Compose (v2 syntax)
docker --version && docker compose version

# Buf CLI for protobuf management
buf --version

# Protocol buffer plugins (installed via make host-install)
# - protoc-gen-go
# - protoc-gen-go-grpc
# - protoc-gen-grpc-gateway
# - protoc-gen-openapiv2
```

### Streamlined Host Development (Recommended)
```bash
# One-time setup: Install all dependencies
make host-install  # Backend (~20s) + Frontend (~40s)

# Generate protobuf code (NEVER CANCEL - network dependent, 30+ seconds)
export PATH=$PATH:$(go env GOPATH)/bin
make generate

# Build backend (NEVER CANCEL - 30-60 seconds for clean build)
make be.build

# Start complete development environment
# MongoDB in Docker, Backend+Frontend on host for fast iteration
make host-dev  # First run: 60+ seconds, subsequent: 10+ seconds

# Development URLs:
# Frontend:     http://localhost:5000
# Backend REST: http://localhost:8080
# Backend gRPC: localhost:9090
# Health Check: http://localhost:8081/healthz
# MongoDB:      localhost:27017 (root:pingis123)
```

### Production Build & Deployment
```bash
# Production builds (with comprehensive validation)
make generate      # Regenerate protobuf code
make lint          # Backend + Frontend linting
make test          # Unit test suite
make test-integration  # Integration tests with MongoDB
cd frontend && npm run test  # Playwright UI tests

# Build production artifacts
make be.build      # Backend binary
make fe.build      # Optimized frontend bundle

# Deploy to production (automated with CI/CD)
./deploy-production.sh  # Fly.io deployment
```

## Build Timing Expectations (Critical for CI/CD)

| Command | Expected Time | Recommended Timeout | Notes |
|---------|---------------|---------------------|-------|
| `make deps` | 20 seconds | 60 seconds | Go dependency resolution |
| `make fe.install` | 40 seconds | 120 seconds | npm install with package-lock |
| `make generate` | 30+ seconds | 120 seconds | **Network-dependent** (buf.build) |
| `make be.build` | 30-60 seconds | 120 seconds | Go compilation with CGO |
| `make fe.build` | 15-30 seconds | 60 seconds | Vite production build |
| `make host-dev` | 60+ / 10+ seconds | 180 seconds | First run / subsequent |
| `make test` | 30-60 seconds | 120 seconds | Backend unit tests |
| `make test-integration` | 2-5 minutes | 10 minutes | Full database testing |
| `npx playwright install` | 5-15 minutes | 30 minutes | **One-time browser install** |
| `npm run test` (UI) | 2-5 minutes | 10 minutes | Playwright test suite |

**Important**: Never cancel operations marked as critical - they involve network operations or complex builds that can leave the system in an inconsistent state.

## Technical Constraints & Performance

### Performance Requirements (Production Validated)
- **API Response Time**: < 200ms for 95th percentile under normal load
- **Database Queries**: Indexed queries achieve O(log n) lookup performance
- **Frontend Bundle**: < 1MB compressed JavaScript for initial page load
- **ELO Calculations**: Complete within MongoDB transaction (< 100ms)
- **Real-time Updates**: Leaderboard refresh < 500ms after match report

### Scalability Characteristics
- **Horizontal Scaling**: Stateless backend services scale horizontally
- **Database**: MongoDB Atlas supports read replicas and sharding
- **Session Management**: JWT tokens enable zero-state backend scaling
- **File Storage**: Text-only data model, no large file upload requirements
- **Memory Footprint**: Backend operates efficiently with < 256MB per instance

### Security Implementation (Production-Hardened)
- **Authentication**: JWT-based with secure token generation and validation
- **Input Validation**: Comprehensive validation on both client and server
- **Injection Prevention**: MongoDB parameterized queries (no string concatenation)
- **XSS Protection**: React's built-in escaping + Content Security Policy headers
- **Rate Limiting**: API rate limiting prevents abuse and ensures availability
- **HTTPS Enforcement**: TLS termination at edge, secure cookie flags
- **CORS Configuration**: Strict origin validation for API access
- **XSS Prevention**: React's built-in escaping + CSP headers
- **Rate Limiting**: API rate limiting to prevent abuse

### Internationalization Constraints
- **Supported Locales**: Swedish (primary) and English (secondary)
- **Text Direction**: Left-to-right only (no RTL support planned)
- **Date/Time Formats**: Swedish conventions with locale-aware formatting
### Internationalization Support (Production-Validated)
- **Supported Locales**: Swedish (sv) as primary, English (en) as secondary
- **Text Direction**: Left-to-right (LTR) only
- **Date/Time Formats**: Swedish conventions with locale-aware formatting via i18next
- **Number Formats**: European decimal notation (comma as decimal separator)
- **Translation Coverage**: 100% coverage for all user-facing strings
- **Fallback Strategy**: English fallback for missing Swedish translations

## Critical Dependencies (Production Versions)

### Backend Dependencies (Go)
```go
// Protocol Buffers and gRPC (v1.60+)
google.golang.org/grpc v1.60.1
google.golang.org/protobuf v1.32.0
github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0

// Database (v1.13+)
go.mongodb.org/mongo-driver v1.13.1

// Logging and Configuration
github.com/rs/zerolog v1.31.0
github.com/spf13/viper v1.18.0

// Security and Validation
github.com/bufbuild/protovalidate-go v0.5.0
github.com/golang-jwt/jwt/v5 v5.2.0
```

### Frontend Dependencies (npm)
```json
{
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "typescript": "^5.3.3",
    "vite": "^5.0.10",
    "tailwindcss": "^3.4.0",
    "i18next": "^23.7.0",
    "react-i18next": "^14.0.0",
    "zustand": "^4.4.7",
    "react-router-dom": "^6.21.0"
  },
  "devDependencies": {
    "@playwright/test": "^1.40.0",
    "@types/react": "^18.2.45",
    "@types/react-dom": "^18.2.18",
    "eslint": "^8.56.0",
    "@typescript-eslint/parser": "^6.15.0"
  }
}
```

### Development Tools
- **golangci-lint v1.55+**: Backend code quality enforcement with comprehensive linter suite
- **buf v1.28+**: Protocol buffer toolchain management and code generation
- **eslint v8+**: Frontend code quality and consistency enforcement
- **playwright v1.40+**: Comprehensive end-to-end testing across browsers
- **@types/***: TypeScript definitions ensuring type safety for all dependencies

## Production Tool Usage Patterns

### Make-based Development Workflow
```bash
# Daily development cycle
make host-dev          # Start full environment (MongoDB + services)
make host-status       # Check what's running
make host-restart      # Quick restart after backend changes (5 seconds)
make host-stop         # Clean shutdown of all services

# Quality assurance (pre-commit)
make lint              # Backend golangci-lint + Frontend ESLint
make test              # Backend unit tests
make test-integration  # Full database integration tests
cd frontend && npm run lint        # Frontend-specific linting
cd frontend && npm run type-check  # TypeScript validation

# Build artifacts
make generate          # Regenerate protobuf code (after .proto changes)
make be.build          # Compile backend binary
make fe.build          # Production-optimized frontend bundle

# Clean and rebuild (when things go wrong)
make clean-all         # Nuclear clean of all artifacts
make rebuild           # Complete rebuild from scratch
```

### Protocol Buffer Development Workflow
```bash
# 1. Edit protocol buffer definitions
vim proto/klubbspel/v1/service.proto

# 2. Generate code (NEVER CANCEL - can take 30+ seconds)
make generate

# 3. Update Go service implementations
# - Implement new RPC methods in backend/internal/service/
# - Update repository layer if needed

# 4. Update TypeScript client code
# - API client will have new generated types
# - Update React components to use new types

# 5. Test integration
make test              # Backend tests
cd frontend && npm run test  # UI tests
```

### Testing Strategy (Production-Grade)
- **Unit Tests**: Go testing framework for business logic isolation
- **Integration Tests**: Full database round-trip with MongoDB test containers
- **UI Tests**: Playwright for complete end-to-end user workflows across browsers
- **Manual Validation**: Critical user scenarios tested manually before releases
- **Performance Tests**: Load testing with realistic data volumes
- **Security Scans**: Automated security scanning with gosec

### Error Handling Pattern (Production Best Practice)
```typescript
// Frontend API calls with comprehensive error handling
try {
  const result = await apiClient.reportMatch(data)
  toast.success(t('match.reportSuccess'))
  onMatchReported?.(result)
} catch (error) {
  if (isRequestCancelledError(error)) {
    // Silently ignore cancelled requests (user navigation)
    return
  }
  const apiError = error as ApiError
  const errorMessage = t(apiError.code) || t('common.unexpectedError')
  toast.error(errorMessage)
  console.error('Match report failed:', apiError)
}
```

### Internationalization Pattern (Production Standard)
```typescript
// All user-facing strings use translation keys with namespaces
import { useTranslation } from 'react-i18next'

export function Component() {
  const { t } = useTranslation()
  
  return (
    <div>
      <h1>{t('common.title')}</h1>
      <Button>{t('common.save')}</Button>
      <p>{t('match.scoreDisplay', { 
        playerA: 'Erik', 
        scoreA: 11, 
        playerB: 'Anna', 
        scoreB: 9 
      })}</p>
    </div>
  )
}
```

## Extension Points for Community Contributions

### Multi-Sport Framework
- **Interface**: `SportValidator` interface ready for implementation
- **Currently**: Table tennis fully implemented
- **Opportunities**: Tennis, padel, badminton, squash validators

### Authentication Providers
- **Interface**: Pluggable authentication adapter pattern
- **Currently**: JWT with email/password
- **Opportunities**: OAuth providers (Google, GitHub), SAML for enterprises

### Notification Channels
- **Interface**: Notification sender abstraction
- **Currently**: Email via SendGrid
- **Opportunities**: SMS, push notifications, webhooks

### Storage Backends
- **Interface**: File storage abstraction (currently not used)
- **Future**: Player photos, tournament documents
- **Opportunities**: S3, Azure Blob Storage, local filesystem

This production-ready architecture provides a solid foundation for community contributions while maintaining system reliability and code quality.