# Tech Context: Klubbspel

## Technologies Used

### Backend Stack
- **Go 1.25+**: Primary backend language with excellent concurrency and performance
- **gRPC**: High-performance RPC framework with protocol buffer integration
- **gRPC-Gateway**: HTTP/JSON gateway for REST API compatibility
- **MongoDB Go Driver**: Official MongoDB client with connection pooling
- **Protocol Buffers**: Type-safe API contract and code generation
- **Zerolog**: Structured logging with performance focus
- **Buf CLI**: Protocol buffer linting, formatting, and code generation

### Frontend Stack
- **React 19**: Latest React with concurrent features and server components
- **TypeScript 5.0+**: Strict type checking with advanced type features
- **Vite**: Ultra-fast build tool with HMR and optimized production builds
- **TailwindCSS**: Utility-first CSS framework with design system
- **i18next**: Comprehensive internationalization with React integration
- **Zustand**: Lightweight state management for global application state
- **React Router**: Client-side routing with lazy loading support

### Development Tools
- **golangci-lint**: Comprehensive Go linting with multiple analyzers
- **ESLint**: JavaScript/TypeScript linting with custom rules
- **Playwright**: End-to-end testing framework for UI validation
- **Docker & Docker Compose**: Containerization for consistent environments
- **Make**: Build automation and development workflow orchestration

### Infrastructure & Deployment
- **Fly.io**: Multi-region application deployment platform
- **MongoDB Atlas**: Managed MongoDB with automated backups and scaling
- **Docker Multi-stage**: Optimized container builds for production
- **SendGrid**: Email delivery service for production notifications
- **MailHog**: Development SMTP server for email testing

## Development Setup

### Prerequisites Installation
```bash
# Go 1.25+ with module support
go version

# Node.js 18+ with npm
node --version && npm --version

# Docker and Docker Compose
docker --version && docker compose version

# Buf CLI for protobuf generation
buf --version
```

### Host Development Workflow (Recommended)
```bash
# 1. Install all dependencies (Backend + Frontend)
make host-install

# 2. Generate protobuf code (CRITICAL: Never cancel, can take 30+ seconds)
make generate

# 3. Build backend (CRITICAL: Never cancel, takes 30-60 seconds)
make be.build

# 4. Start development environment (MongoDB in Docker, services on host)
make host-dev

# Services available:
# - Frontend: http://localhost:5000
# - Backend API: http://localhost:8080  
# - MongoDB: localhost:27017
# - MailHog UI: http://localhost:8025
```

### Build Timing Expectations
- `make generate`: 30+ seconds (network-dependent, buf.build registry)
- `make be.build`: 30-60 seconds (Go compilation and dependencies)
- `make fe.build`: 15-30 seconds (Vite production build)
- `make host-dev`: 60+ seconds first time, 10+ seconds subsequent
- `make test`: 30-60 seconds (unit tests)
- `make test-integration`: 2-5 minutes (full database tests)

## Technical Constraints

### Performance Requirements
- **API Response Time**: < 200ms for typical requests under normal load
- **Database Queries**: Proper indexing for O(log n) lookups on player/match data
- **Frontend Bundle Size**: < 1MB compressed for initial page load
- **ELO Calculations**: Must complete within transaction for data consistency

### Scalability Constraints
- **Single MongoDB Instance**: Current architecture assumes single database
- **Session State**: JWT tokens enable horizontal scaling without session storage
- **File Storage**: No large file uploads, text-only data model
- **Memory Usage**: Backend designed for < 256MB memory per instance

### Security Requirements
- **Authentication**: JWT-based authentication with proper expiration
- **Input Validation**: All user inputs validated on both client and server
- **SQL Injection Prevention**: MongoDB parameterized queries (no string concatenation)
- **XSS Prevention**: React's built-in escaping + CSP headers
- **Rate Limiting**: API rate limiting to prevent abuse

### Internationalization Constraints
- **Supported Locales**: Swedish (primary) and English (secondary)
- **Text Direction**: Left-to-right only (no RTL support planned)
- **Date/Time Formats**: Swedish conventions with locale-aware formatting
- **Number Formats**: European decimal notation (comma as decimal separator)

## Dependencies

### Critical Backend Dependencies
```go
// Protocol Buffers and gRPC
google.golang.org/grpc
google.golang.org/protobuf
github.com/grpc-ecosystem/grpc-gateway/v2

// Database
go.mongodb.org/mongo-driver

// Logging and Configuration  
github.com/rs/zerolog
github.com/spf13/viper

// Validation
github.com/bufbuild/protovalidate-go
```

### Critical Frontend Dependencies
```json
{
  "react": "^19.0.0",
  "typescript": "^5.0.0", 
  "vite": "^5.0.0",
  "tailwindcss": "^3.0.0",
  "i18next": "^23.0.0",
  "zustand": "^4.0.0",
  "react-router-dom": "^6.0.0"
}
```

### Development Dependencies
- **golangci-lint**: Backend code quality enforcement
- **buf**: Protocol buffer toolchain management
- **eslint**: Frontend code quality and consistency
- **playwright**: End-to-end testing framework
- **@types/***: TypeScript definitions for all major libraries

## Tool Usage Patterns

### Make-based Workflow
```bash
# Development iteration cycle
make host-dev          # Start environment
make host-restart      # Restart after backend changes
make host-stop         # Clean shutdown

# Quality assurance
make lint             # Backend linting
cd frontend && npm run lint  # Frontend linting
make test             # Unit tests
make test-integration # Integration tests

# Build and deployment
make generate         # Regenerate protobuf code
make be.build         # Backend compilation
make fe.build         # Frontend production build
```

### Protocol Buffer Workflow
```bash
# 1. Edit .proto files in proto/klubbspel/v1/
# 2. Run code generation (NEVER CANCEL)
make generate
# 3. Update backend Go code to use new types
# 4. Update frontend TypeScript code to use new types
# 5. Test integration between frontend and backend
```

### Testing Strategy
- **Unit Tests**: Go testing framework for business logic
- **Integration Tests**: Full database round-trip testing
- **UI Tests**: Playwright for end-to-end user workflows
- **Manual Testing**: Complete user scenarios after changes

### Error Handling Pattern
```typescript
// Frontend API calls always handle errors
try {
  const result = await apiClient.reportMatch(data)
  toast.success(t('match.reportSuccess'))
  onMatchReported?.(result)
} catch (error) {
  const apiError = error as ApiError
  toast.error(t(apiError.code) || t('common.error'))
}
```

### Internationalization Pattern
```typescript
// All user-facing strings use translation keys
const { t } = useTranslation()
return (
  <Button>{t('common.save')}</Button>
  <p>{t('match.scoreValidation', { playerA: 'Erik', playerB: 'Anna' })}</p>
)
```