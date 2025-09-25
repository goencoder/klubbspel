# System Patterns: Klubbspel

## System Architecture

### Microservices Architecture
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
│   (Vite Build)  │    │   Buffers       │    │  (Infrastructure│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Architecture Principles:**
- **API-First Design**: Protocol buffer definitions drive both backend and frontend types
- **Clean Architecture**: Clear separation between service, repository, and transport layers
- **Type Safety**: End-to-end type safety from protobuf to TypeScript
- **Stateless Services**: Backend services are stateless with all state in MongoDB

## Key Technical Decisions

### Backend Decisions
- **Go with gRPC**: Chosen for performance, type safety, and excellent protobuf integration
- **MongoDB**: Document database fits the flexible tournament/match data structure
- **Repository Pattern**: Clean separation between business logic and data access
- **Error Handling**: Structured error responses with localization support
- **Validation**: Both protobuf validation and business rule validation layers

### Frontend Decisions
- **React 19 with TypeScript**: Modern React features with strict type checking
- **Vite Build System**: Fast development server and optimized production builds
- **TailwindCSS**: Utility-first CSS with design system consistency
- **i18next**: Comprehensive internationalization with namespace organization
- **State Management**: Zustand for global state, React hooks for component state

### Infrastructure Decisions
- **Fly.io Deployment**: Multi-region deployment with edge proximity
- **Docker Containers**: Consistent deployment across environments
- **Host Development**: Fast iteration with services running on host, MongoDB in Docker
- **Protocol Buffer Generation**: Automated code generation with buf CLI

## Design Patterns in Use

### Backend Patterns
```go
// Repository Pattern
type PlayerRepo interface {
    Create(ctx context.Context, player *Player) error
    FindByID(ctx context.Context, id string) (*Player, error)
}

// Service Layer Pattern
type PlayerService struct {
    repo PlayerRepo
    validator Validator
}

// Error Wrapper Pattern
func (s *Service) Method() (*Response, error) {
    if err := s.validate(req); err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }
}
```

### Frontend Patterns
```typescript
// Custom Hooks Pattern
export function useMatchReporting(options: UseMatchReportingOptions) {
    // Encapsulate complex state management
}

// Service Layer Pattern
class ApiClient {
    async reportMatch(data: ReportMatchRequest): Promise<ReportMatchResponse>
}

// Component Composition Pattern
export function ReportMatchDialog({ onMatchReported }: Props) {
    // Dialog with embedded form and validation
}
```

## Component Relationships

### Data Flow Architecture
```
User Input → React Component → API Client → gRPC Service → Repository → MongoDB
                    ↓                           ↓              ↓
              State Update ← Response ← Business Logic ← Data Validation
```

### Critical Dependencies
- **Protocol Buffers**: Central contract between frontend and backend
- **Authentication Store**: Zustand store managing user session across components
- **Validation Layer**: Shared validation logic between client and server
- **Internationalization**: i18next integration in all user-facing components

## Critical Implementation Paths

### Match Reporting Flow
```
1. User selects players → Validation (no self-play, valid players)
2. User enters scores → Client validation (table tennis rules)
3. Submit request → Server validation (business rules, time windows)
4. Database update → Match record + ELO calculations
5. Real-time update → Leaderboard refresh + notifications
```

### Player Management Flow
```
1. Player registration → Duplicate detection (normalized names)
2. Club membership → Role assignment (member/admin)
3. Player merging → Authenticated user merges email-less duplicates
4. Data consistency → All match references updated atomically
```

### Series Management Flow
```
1. Series creation → Time validation (start < end)
2. Format configuration → Sport-specific rules (sets to play)
3. Match validation → Time window enforcement
4. Leaderboard calculation → Real-time ELO updates
```

### Authentication & Authorization
```
1. JWT token validation → Extract user claims
2. Role checking → Club admin permissions
3. Resource ownership → User can only edit own data
4. Audit logging → Track all administrative actions
```

### Development Workflow
```
1. Protobuf changes → buf generate → Go + TypeScript types
2. Backend changes → make be.build → Host restart
3. Frontend changes → Vite HMR → Instant browser update
4. Database changes → Migration scripts → Schema updates
```