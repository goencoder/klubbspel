# System Patterns: Klubbspel

## System Architecture

Klubbspel follows a microservices-inspired architecture with clear separation between frontend, backend, and data layers:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   Database      │
│   React/TS      │◄──►│   Go/gRPC       │◄──►│   MongoDB       │
│   Port: 5000    │    │   Port: 9090    │    │   Port: 27017   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  REST Gateway   │
                       │  Port: 8080     │
                       └─────────────────┘
```

### Core Architectural Principles
- **API-First Design**: gRPC services with automatic REST gateway generation
- **Protocol Buffers**: Strongly-typed API contracts shared between frontend and backend
- **Repository Pattern**: Clean separation between business logic and data access
- **Dependency Injection**: Configuration and dependencies injected at startup
- **Event-Driven Updates**: Real-time ELO calculations triggered by match events

## Key Technical Decisions

### Backend Architecture
- **Go + gRPC**: High performance, strongly-typed API with efficient serialization
- **MongoDB**: Document database optimal for flexible tournament data structures
- **Protocol Buffers**: Type-safe API contracts with automatic code generation
- **gRPC-Gateway**: Automatic REST API generation from gRPC definitions
- **Repository Pattern**: Clean abstraction between business logic and data persistence

### Frontend Architecture
- **React 19**: Modern React with latest hooks and concurrent features
- **TypeScript**: Full type safety with generated types from protobuf schemas
- **Vite**: Fast development server and optimized production builds
- **TailwindCSS**: Utility-first CSS framework for rapid UI development
- **GitHub Spark**: Component library for consistent, accessible UI components

### Development Infrastructure
- **Docker Compose**: Containerized development environment with MongoDB
- **Buf**: Modern protobuf toolchain for schema management and code generation
- **Make**: Build automation with clear commands for common workflows
- **Zerolog**: Structured logging with performance and searchability

## Design Patterns in Use

### Backend Patterns
- **Repository Pattern**: `ClubRepo`, `PlayerRepo`, `SeriesRepo`, `MatchRepo` interfaces with MongoDB implementations
- **Service Layer**: Business logic encapsulated in service classes (`ClubService`, `PlayerService`, etc.)
- **Dependency Injection**: Services receive repositories through constructor injection
- **Command Pattern**: gRPC service methods act as commands that orchestrate business operations
- **Event Sourcing (Limited)**: Audit trail maintains history of all changes for transparency

### Frontend Patterns
- **Component Composition**: Reusable UI components with clear prop interfaces
- **Custom Hooks**: Business logic abstracted into reusable hooks (`useClubs`, `usePlayerMatches`)
- **Service Layer**: API calls abstracted into service modules with error handling
- **Provider Pattern**: Context providers for global state (language settings, user context)
- **Controlled Components**: Form inputs with validation and real-time feedback

### Data Patterns
- **Aggregate Root**: Tournament series act as aggregate roots containing related matches
- **Value Objects**: Immutable objects for scores, ratings, and calculated statistics
- **Domain Events**: Match completion triggers ELO recalculation events
- **CQRS (Light)**: Separate read and write operations for complex leaderboard queries

## Component Relationships

### Backend Service Dependencies
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ gRPC Server │◄──►│  Services   │◄──►│ Repositories│
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ REST Gateway│    │   Config    │    │  MongoDB    │
└─────────────┘    └─────────────┘    └─────────────┘
```

### Frontend Component Hierarchy
```
App
├── Router
├── LanguageProvider
├── Pages
│   ├── ClubsPage
│   ├── PlayersPage
│   ├── SeriesPage
│   └── SeriesDetailPage
├── Components
│   ├── Navigation
│   ├── Forms (CreateClub, CreatePlayer, ReportMatch)
│   ├── Lists (ClubList, PlayerList, MatchList)
│   └── Display (Leaderboard, PlayerStats)
└── Services
    ├── ClubService
    ├── PlayerService
    └── SeriesService
```

### Cross-Service Communication
- **Frontend → Backend**: HTTP/REST calls to gRPC-Gateway on port 8080
- **Backend → Database**: Direct MongoDB driver connection with connection pooling
- **Development**: All services communicate via localhost with Docker networking

## Critical Implementation Paths

### Match Reporting Flow
1. **Frontend**: User submits match result via `ReportMatchDialog`
2. **Validation**: Client-side validation + server-side validation in `MatchService`
3. **Persistence**: Match stored in MongoDB via `MatchRepo.CreateMatch()`
4. **ELO Calculation**: `LeaderboardService.UpdateRatings()` triggered automatically
5. **Response**: Updated player ratings returned to frontend
6. **UI Update**: Leaderboard and player stats refresh in real-time

### Tournament Series Lifecycle
1. **Creation**: `SeriesService.CreateSeries()` with validation of dates and participants
2. **Player Registration**: Players added to series via `SeriesService.AddPlayer()`
3. **Match Generation**: Optional round-robin match generation for structured tournaments
4. **Progress Tracking**: Real-time calculation of standings and remaining matches
5. **Completion**: Series marked complete when all matches reported or end date reached

### ELO Rating System
1. **Initial Rating**: New players start with configurable default ELO rating
2. **K-Factor Calculation**: Dynamic K-factor based on player experience and rating difference
3. **Match Impact**: Win/loss probability calculated using standard ELO formula
4. **Rating Update**: Both players' ratings updated atomically in single transaction
5. **History Tracking**: All rating changes logged with match context for audit trail

### Error Handling Patterns
- **Backend**: Structured errors with gRPC status codes and detailed error messages
- **Frontend**: Centralized error handling with user-friendly error display
- **Validation**: Input validation at multiple layers (client, server, database)
- **Recovery**: Graceful degradation when services are unavailable
- **Monitoring**: Health checks and metrics collection for proactive issue detection