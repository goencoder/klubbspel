# System Patterns: Klubbspel v1.0.0

## System Architecture

### Production Microservices Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React SPA     │◄──►│   Go Backend    │◄──►│    MongoDB      │
│   (Frontend)    │    │   (gRPC/REST)   │    │   (Database)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                        ▲                        ▲
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Static Assets  │    │   Protocol      │    │   Fly.io        │
│   (Vite Build)  │    │   Buffers       │    │  (Production)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Architecture Principles:**
- **API-First Design**: Protocol buffer definitions serve as single source of truth for all interfaces
- **Clean Architecture**: Clear separation between service, repository, and transport layers
- **Type Safety**: End-to-end type safety from protobuf through Go to TypeScript
- **Stateless Services**: Backend services are horizontally scalable with all state in MongoDB
- **Security by Design**: JWT authentication, input validation, and defense in depth

## Key Technical Decisions

### Backend Architecture
- **Go with gRPC**: High-performance RPC with excellent protobuf integration and type safety
- **MongoDB**: Document database optimally fits flexible tournament/match data structures
- **Repository Pattern**: Clean data access abstraction enables comprehensive testing
- **Error Handling**: Structured error responses with internationalization support
- **Validation Layers**: Both protobuf field validation and business rule validation
- **Audit Logging**: Comprehensive logging of administrative actions for compliance

### Frontend Architecture
- **React 19 with TypeScript**: Latest React features with strict type checking
- **Vite Build System**: Fast development server and highly optimized production builds
- **TailwindCSS**: Utility-first CSS ensures design system consistency
- **i18next**: Complete internationalization with namespace organization
- **State Management**: Zustand for global state, React hooks for local component state
- **Code Splitting**: Route-based lazy loading for optimal initial load time

### Infrastructure Design
- **Fly.io Deployment**: Multi-region deployment with global edge proximity
- **Docker Containers**: Consistent builds and deployments across all environments
- **Protocol Buffer Generation**: Automated type generation with buf CLI toolchain
- **Host Development**: Fast local iteration with MongoDB in Docker, services on host
- **CI/CD Pipeline**: Automated testing, building, and deployment workflows

## Design Patterns Implementation

### Backend Patterns (Go)
```go
// Repository Pattern - Clean data access layer
type PlayerRepo interface {
    Create(ctx context.Context, player *Player) error
    FindByID(ctx context.Context, id string) (*Player, error)
    List(ctx context.Context, filters ListFilters) ([]*Player, error)
}

// Service Layer Pattern - Business logic encapsulation
type PlayerService struct {
    repo      PlayerRepo
    validator Validator
    auditor   AuditLogger
}

// Error Wrapper Pattern - Consistent error handling
func (s *Service) Method(ctx context.Context, req *Request) (*Response, error) {
    if err := s.validate(req); err != nil {
        return nil, status.Error(codes.InvalidArgument, "VALIDATION_ERROR")
    }
    // Business logic...
    return response, nil
}

// Context Pattern - Request-scoped values
func (s *Service) AuthorizedMethod(ctx context.Context) error {
    subject := GetSubjectFromContext(ctx)
    if subject == nil {
        return status.Error(codes.Unauthenticated, "LOGIN_REQUIRED")
    }
    // Authorization checks...
}

// Named Struct Pattern - Prefer named types over anonymous structs
type playerMatchStats struct {
    PlayerID      string
    Name          string
    Position      int
    MatchesPlayed int
    Wins          int
    Losses        int
    // Reusable definition improves maintainability
}

// Resource Cleanup Pattern - Proper error handling in defer
func (r *Repo) FindAll(ctx context.Context) ([]*Entity, error) {
    cursor, err := r.collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer func() {
        if err := cursor.Close(ctx); err != nil {
            log.Warn().Err(err).Msg("failed to close cursor")
        }
    }()
    // Process results...
}
```

### Frontend Patterns (TypeScript/React)
```typescript
// Custom Hooks Pattern - Reusable stateful logic
export function useMatchReporting(options: Options) {
    const [state, setState] = useState(initialState)
    // Complex state management encapsulated
    return { reportMatch, isLoading, error }
}

// Mediator Pattern - Breaking circular dependencies
// sessionManager.ts - Mediator between API client and auth store
let sessionExpiredHandler: (() => void) | null = null

export function registerSessionExpiredHandler(handler: () => void) {
    sessionExpiredHandler = handler
}

export function handleSessionExpired() {
    if (sessionExpiredHandler) {
        sessionExpiredHandler()
    }
}

// Service Layer Pattern - API abstraction
class ApiClient {
    async reportMatch(data: ReportMatchRequest): Promise<ReportMatchResponse> {
        // HTTP/gRPC communication layer
    }
}

// Component Composition Pattern - Reusable UI components
export function ReportMatchDialog({ onMatchReported }: Props) {
    return (
        <Dialog>
            <ReportMatchForm onSubmit={handleSubmit} />
        </Dialog>
    )
}

// Error Boundary Pattern - Graceful error handling
export function ErrorBoundary({ children }: Props) {
    // Catch React errors and display fallback UI
}
```

## Component Relationships

### Data Flow Architecture (Production System)
```
User Input → React Component → API Client → gRPC Gateway → gRPC Service
                    ↓                                           ↓
              Local State ←─────── JSON/HTTP ←──────── Protocol Buffer
                    ↓                                           ↓
              UI Update                                   Repository Layer
                                                               ↓
                                                           MongoDB
                                                               ↓
                                                        Persistent Storage
```

### Critical Dependencies and Integration Points
- **Protocol Buffers**: Central contract enforcing type safety across entire stack
- **Authentication Store**: Zustand store managing user session state globally
- **Validation Layer**: Shared validation logic between client and server
- **i18next Integration**: Internationalization woven through all user-facing components
- **MongoDB Indexes**: Query optimization through strategic index design
- **Health Checks**: Monitoring endpoints for infrastructure health tracking
## Critical Implementation Patterns

### Match Reporting Flow (Production)
```
1. User Interface → Player selection with validation (prevents self-play, invalid players)
2. Score Entry → Client-side validation (enforces table tennis rules)
3. API Request → Server validation (business rules, time window enforcement)
4. Database Transaction → Atomic match record creation + ELO calculation
5. Real-time Response → Immediate leaderboard refresh + success confirmation
6. Audit Log → Administrative action tracking for compliance
```

### Player Management Flow (Production)
```
1. Player Registration → Intelligent duplicate detection (normalized name comparison)
2. Club Membership → Role-based assignment (member/admin permissions)
3. Authentication → JWT token validation and refresh mechanism
4. Player Merging → Authorized users merge email-less duplicates with accounts
5. Data Consistency → All match references updated atomically in transaction
6. Search Indexing → Maintains search keys for fast player lookup
```

### Series Management Flow (Production)
```
1. Series Creation → Comprehensive validation (dates, format, permissions)
2. Format Configuration → Sport-specific rules (sets to play, scoring system)
3. Time Window → Automatic enforcement (matches must be within date boundaries)
4. Match Reporting → Real-time validation and ELO updates
5. Leaderboard → Automatic ranking calculation and caching
6. CSV Export → Generate reports for external analysis
```

### Authentication & Authorization Flow (Production)
```
1. Login Request → JWT token generation with user claims
2. Token Validation → Middleware extracts and validates token
3. Role Verification → Check club admin or platform owner permissions
4. Resource Authorization → Verify user ownership or permissions
5. Audit Logging → Track all administrative and sensitive actions
6. Token Refresh → Seamless session management without interruption
```

### Development Workflow (Optimized for Speed)
```
1. Protobuf Changes → buf generate → Automatic Go + TypeScript type generation
2. Backend Changes → make be.build → Binary compilation (30-60 seconds)
3. Backend Restart → make host-restart → Hot reload (5 seconds)
4. Frontend Changes → Vite HMR → Instant browser update (milliseconds)
5. Database Migrations → Migration manager → Distributed lock, safe execution
6. Testing → make test → Comprehensive validation before commit
```

## Extensibility Patterns

### Multi-Sport Framework (Ready for Extension)
```go
// Sport-specific validation interface
type SportValidator interface {
    ValidateMatch(match *Match) error
    CalculateWinner(match *Match) (*Player, error)
}

// Currently implemented:
type TableTennisValidator struct{}

// Ready for community implementation:
// type TennisValidator struct{}
// type PadelValidator struct{}
```

### Plugin Architecture for Features
- **Email Adapters**: Pluggable email providers (SendGrid, MailHog, custom)
- **Authentication Providers**: Extensible auth mechanisms (JWT, OAuth future)
- **Storage Backends**: Abstract storage interfaces for future cloud storage
- **Notification Channels**: Framework ready for multiple notification types

This architecture enables community contributions while maintaining system stability and consistency.