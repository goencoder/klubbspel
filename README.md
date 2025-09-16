# Klubbspel

A Swedish table tennis club management system built with Go, React, MongoDB, and Protocol Buffers.

## Architecture

**Backend**: Go microservice with gRPC/REST APIs, MongoDB database, protobuf code generation  
**Frontend**: React TypeScript application with Vite, internationalization (Swedish/English)  
**Infrastructure**: Fly.io deployment, Docker containers, MongoDB Atlas database  
**Development**: Protocol buffer code generation, host-based development for fast iteration

### Technology Stack

- **Backend**: Go 1.25+, gRPC/REST, MongoDB driver, protobuf, golangci-lint
- **Frontend**: React 18, TypeScript, Vite, TailwindCSS, i18next
- **Database**: MongoDB with authentication and health checks
- **Deployment**: Fly.io platform with Docker multi-stage builds
- **Email**: SendGrid (production) / MailHog (development)
- **Build System**: Make, buf CLI for protobuf generation

## Key Features

- ⚽ **Series Management** - Create time-bound tournament series with clear start/end boundaries
- 👥 **Player Registration** - Intelligent duplicate detection using normalized names (Erik/Eric prevention)
- 🏓 **Match Reporting** - Report match results with game scores and automatic ELO rating updates
- 🏆 **Live Leaderboards** - Public, real-time ranking display sorted by ELO rating
- 🌍 **Full Internationalization** - Complete Swedish/English language support including error messages

## 🚀 Quick Start

### Prerequisites

- [Go 1.22+](https://golang.org/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [Buf CLI](https://docs.buf.build/installation) (for API development)

### Development Setup

```bash
# 1. Clone and enter directory
git clone https://github.com/goencoder/motionsserien.git
cd motionsserien

# 2. Install dependencies
make host-install

# 3. Start development environment
make host-dev
```

### Daily Development Workflow

The recommended development approach uses **host development** for fast iteration:

1. **Start development environment** (MongoDB + MailHog in Docker, backend + frontend on host):
   ```bash
   make host-dev
   ```

2. **Make code changes** - follow this order for builds:
   ```bash
   make generate    # Generate protobuf code (required before backend build)
   make be.build    # Build backend
   make fe.build    # Build frontend
   ```

3. **View the application**:
   - **Frontend**: http://localhost:5173 (React dev server)
   - **Backend API**: http://localhost:8080 (Go server)
   - **MailHog**: http://localhost:8025 (email testing)

4. **Stop development**:
   ```bash
   make host-stop
   ```

### Alternative: Docker Development

For isolated development environment:

```bash
# Start everything in Docker
make docker-dev

# Stop Docker environment
make docker-stop
```

## 🛠 Build Commands

### Code Generation
```bash
make generate        # Generate protobuf code (Go + TypeScript)
make proto.clean     # Clean generated files
```

### Backend
```bash
make be.build        # Build backend binary
make be.test         # Run backend tests
make be.lint         # Lint backend code
```

### Frontend
```bash
make fe.install      # Install dependencies
make fe.build        # Build for production
make fe.dev          # Development server
make fe.test         # Run tests
make fe.lint         # Lint frontend code
```

### Full Pipeline
```bash
make lint           # Lint both backend and frontend
make build          # Build both backend and frontend
make test           # Run all tests
```

## 📊 Database Management

### Development Database
```bash
make db-up          # Start MongoDB in Docker
make db-down        # Stop MongoDB
make db-reset       # Reset development data
```

### Test Database
```bash
make test-db-up     # Start test database
make test-db-down   # Stop test database
make validate-db    # Check for stale data issues
```

## 📁 Project Structure

```
klubbspel/
├── backend/                # Go backend service
│   ├── cmd/               # Application entry points
│   ├── internal/          # Private application code
│   │   ├── auth/         # Authentication & authorization
│   │   ├── repo/         # Database repositories
│   │   ├── service/      # Business logic
│   │   └── server/       # HTTP/gRPC servers
│   ├── proto/gen/go/     # Generated Go code
│   └── openapi/          # OpenAPI specifications
├── frontend/              # React TypeScript frontend
│   ├── src/              # Source code
│   ├── tests/            # Playwright UI tests
│   └── dist/             # Built frontend (after build)
├── proto/                # Protocol Buffer definitions
│   └── pingis/v1/        # API v1 definitions
├── tests/                # Integration tests
├── docs/                 # Additional documentation
├── bin/                  # Built binaries
├── Makefile              # Build automation
└── README.md             # This file
```

## 🧰 Tech Stack

### Backend
- **Go 1.22+** - Main programming language
- **gRPC** - High-performance RPC framework
- **gRPC-Gateway** - REST API gateway
- **MongoDB** - Document database
- **Protocol Buffers** - API definition and serialization
- **Buf** - Protocol Buffer toolchain

### Frontend
- **React 19** - UI library with latest features
- **TypeScript** - Type-safe JavaScript
- **Vite** - Fast build tool and dev server
- **GitHub Spark** - Component framework
- **Tailwind CSS** - Utility-first CSS framework
- **Playwright** - End-to-end testing

## 🚀 Deployment

### Deployment to Fly.io

#### First-Time Setup

1. **Install and authenticate with Fly.io**:
   ```bash
   # Install flyctl
   curl -L https://fly.io/install.sh | sh
   
   # Login to Fly.io
   flyctl auth login
   ```

2. **Set up MongoDB Atlas**:
   - Create a MongoDB Atlas cluster
   - Create a database user
   - Get the connection string (format: `mongodb+srv://username:password@cluster.mongodb.net/pingis`)

3. **Deploy applications**:
   ```bash
   # Deploy both backend and frontend
   ./deploy.sh full
   ```

4. **Set up secrets** (the deploy script will show you these commands):
   ```bash
   # Backend secrets
   flyctl secrets set MONGO_URI='mongodb+srv://username:password@cluster.mongodb.net/pingis?retryWrites=true&w=majority' --app klubbspel-backend
   flyctl secrets set MONGO_DB='pingis' --app klubbspel-backend
   
   # Email secrets (if using SendGrid)
   flyctl secrets set SENDGRID_API_KEY='your-sendgrid-api-key' --app klubbspel-backend
   flyctl secrets set EMAIL_PROVIDER='sendgrid' --app klubbspel-backend
   
   # Security secrets
   flyctl secrets set GDPR_ENCRYPTION_KEY='your-32-character-encryption-key' --app klubbspel-backend
   ```

#### Iterative Deployments

For ongoing development and deployments:

```bash
# Deploy only backend changes
./deploy.sh backend

# Deploy only frontend changes  
./deploy.sh frontend

# Deploy both (full deployment)
./deploy.sh full
```

## 🔍 Troubleshooting

### Log Levels

Set environment variable `LOG_LEVEL` to control logging:
```bash
export LOG_LEVEL=debug  # debug, info, warn, error
```

### Log Locations
- **Host development**: Console output
- **Docker development**: `docker-compose logs [service]`
- **Production**: Configure log output via Docker or systemd

### Common Issues

#### MongoDB Connection Issues
```bash
# Check MongoDB status
make db-logs

# Reset database if corrupted
make db-reset
```

#### Frontend Build Issues
```bash
# Clear node modules and reinstall
rm -rf frontend/node_modules
make fe.install

# Check for TypeScript errors
make fe.lint
```

#### Protocol Buffer Issues
```bash
# Regenerate all protobuf code
make proto.clean
make generate
```

## 🧪 Testing

### Backend Tests
```bash
make be.test                    # Run all backend tests
go test ./backend/internal/...  # Run specific package tests
```

### Frontend Tests
```bash
make fe.test                    # Run unit tests
make fe.test.ui                 # Run Playwright UI tests
```

### Integration Tests
```bash
make test                       # Run full test suite
make validate-db                # Validate database integrity
```

## 📚 Documentation

- [Email Setup Guide](docs/EMAIL_SETUP.md)
- [Authorization System](docs/authz.md)
- [Stale Data Prevention](docs/STALE_DATA_PREVENTION.md)
- [Phase Implementation Summaries](PHASE*.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Follow the build workflow:
   ```bash
   make lint      # Lint code
   make generate  # Generate protobuf code
   make be.build  # Build backend
   make fe.build  # Build frontend
   make test      # Run tests
   ```
4. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.