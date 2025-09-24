# Tech Context: Klubbspel

## Technologies Used

### Backend Stack
- **Go 1.25**: Primary backend language for high performance and strong typing
- **gRPC**: High-performance RPC framework for API services
- **Protocol Buffers**: Interface definition language for API contracts
- **MongoDB**: Document database for flexible tournament data storage
- **gRPC-Gateway**: Automatic REST API generation from gRPC services
- **Zerolog**: Structured logging with high performance
- **Buf**: Modern protobuf toolchain for schema management

### Frontend Stack
- **React 19**: Modern frontend framework with latest concurrent features
- **TypeScript**: Type-safe JavaScript with full IDE support
- **Vite**: Fast development server and build tool
- **TailwindCSS**: Utility-first CSS framework for rapid styling
- **GitHub Spark**: Component library for consistent UI components
- **React Router**: Client-side routing for single-page application

### Development & Testing
- **Docker & Docker Compose**: Containerized development environment
- **Playwright**: End-to-end testing framework for UI validation
- **Go Testing**: Built-in testing framework for backend unit tests
- **Make**: Build automation and workflow management
- **ESLint**: TypeScript/React code linting and formatting
- **golangci-lint**: Go code linting with multiple analyzers

### Infrastructure & Deployment
- **Fly.io**: Production hosting platform for both frontend and backend
- **GitHub Actions**: CI/CD pipeline for automated testing and deployment
- **Docker**: Containerization for consistent deployment environments
- **Nginx**: Reverse proxy and static file serving in production

## Development Setup

### Prerequisites
- **Go 1.25+**: Backend development and building
- **Node.js 18+**: Frontend development and npm packages
- **Docker**: Local development environment with MongoDB
- **Buf CLI**: Protocol buffer code generation
- **Make**: Build automation (pre-installed on macOS/Linux)

### Quick Start Commands
```bash
# 1. Install dependencies (20-60 seconds)
make host-install

# 2. Generate protobuf code (30+ seconds)
make generate

# 3. Build all components (45-90 seconds)
make be.build && make fe.build

# 4. Start development environment (60+ seconds first time)
make host-dev
```

### Development Ports
- **Frontend**: http://localhost:5000 (Vite dev server)
- **Backend gRPC**: localhost:9090 (direct gRPC)
- **Backend REST**: http://localhost:8080 (gRPC-Gateway)
- **Health/Metrics**: http://localhost:8081 (monitoring endpoints)
- **MongoDB**: localhost:27017 (database, auth: root:pingis123)

### Key Development Files
- **Makefile**: All build and development commands
- **docker-compose.yml**: Local development services
- **buf.yaml**: Protobuf generation configuration
- **.github/copilot-instructions.md**: Comprehensive development guide

## Technical Constraints

### Performance Requirements
- **API Response Time**: Sub-second response for all user interactions
- **ELO Calculation**: Rating updates within 5 seconds of match completion
- **Concurrent Users**: Support 100+ simultaneous users per tournament
- **Database Queries**: Optimized queries with proper indexing strategies

### Platform Constraints
- **Go Version**: Must use Go 1.25+ for latest features and security
- **Browser Support**: Modern browsers with ES2020+ support (no IE support)
- **Mobile Responsive**: Must work on mobile devices but no native app requirement
- **Network**: Designed for reliable internet connection (no offline support)

### Security Requirements
- **Input Validation**: All user inputs validated on both client and server
- **SQL Injection Prevention**: Parameterized MongoDB queries only
- **CORS Policy**: Strict cross-origin resource sharing configuration
- **Error Handling**: No internal system details exposed to frontend
- **Audit Trail**: Complete logging of all system changes

### Scalability Constraints
- **Single Database**: MongoDB single instance (no sharding requirement initially)
- **Stateless Backend**: All services must be stateless for horizontal scaling
- **File Storage**: Local file system only (no external file storage initially)
- **Caching**: In-memory caching acceptable (no external cache requirement)

## Dependencies

### Backend Dependencies (Go Modules)
```go
// Core gRPC and API
google.golang.org/grpc v1.60+
google.golang.org/protobuf v1.31+
github.com/grpc-ecosystem/grpc-gateway/v2 v2.18+

// Database and data handling
go.mongodb.org/mongo-driver v1.13+
github.com/google/uuid v1.4+

// Logging and monitoring
github.com/rs/zerolog v1.31+
github.com/prometheus/client_golang v1.17+

// Configuration and utilities
github.com/spf13/viper v1.17+
github.com/stretchr/testify v1.8+ (testing)
```

### Frontend Dependencies (package.json)
```json
{
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^6.20.0",
    "typescript": "^5.3.0",
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0"
  },
  "devDependencies": {
    "vite": "^5.0.0",
    "tailwindcss": "^3.3.0",
    "eslint": "^8.54.0",
    "playwright": "^1.40.0",
    "@typescript-eslint/parser": "^6.12.0"
  }
}
```

### Development Dependencies
- **Buf CLI**: Protocol buffer toolchain
- **Docker Compose**: Multi-container development environment
- **golangci-lint**: Go code quality and formatting
- **Playwright Browsers**: Automated testing browsers (Chrome, Firefox, Safari)

## Tool Usage Patterns

### Build Automation (Make)
- **`make host-install`**: Install all dependencies (Go modules + npm packages)
- **`make generate`**: Generate protobuf code for both Go and TypeScript
- **`make be.build`**: Build Go backend binary
- **`make fe.build`**: Build production frontend assets
- **`make host-dev`**: Start complete development environment
- **`make test`**: Run backend unit tests
- **`make test-integration`**: Run full integration test suite

### Protobuf Workflow (Buf)
- **Schema Definition**: Proto files in `proto/klubbspel/v1/`
- **Code Generation**: Automatic Go and TypeScript client generation
- **Versioning**: Buf handles schema evolution and breaking change detection
- **Validation**: Lint rules ensure consistent API design patterns

### Testing Strategy
- **Unit Tests**: Go testing framework for business logic validation
- **Integration Tests**: Full stack tests with real MongoDB instance
- **E2E Tests**: Playwright tests covering complete user workflows
- **Manual Testing**: Browser-based validation of user scenarios

### Development Workflow
1. **Code Generation**: Always run `make generate` after proto changes
2. **Incremental Building**: Use `make host-restart` for quick development cycles
3. **Quality Checks**: `make lint` before commits
4. **Testing**: `make test` for unit tests, `make test-integration` for full validation
5. **Manual Validation**: Test complete user workflows in browser at http://localhost:5000

### Production Deployment
- **Container Building**: Docker multi-stage builds for optimized images
- **Environment Configuration**: Environment variables for all runtime configuration
- **Health Monitoring**: Built-in health checks and metrics endpoints
- **Zero-Downtime**: Blue-green deployment strategy with health check validation