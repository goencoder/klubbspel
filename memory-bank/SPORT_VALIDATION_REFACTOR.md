# Sport-Specific Validation Architecture (High Priority Refactor)

## Problem Statement

**Current State**: All sports in Klubbspel share a single validation function (`validateTableTennisScore()`) which enforces identical rules for all racket/paddle sports:
- Best-of-N sets format (3, 5, or 7 sets)
- No ties/draws allowed
- Winner must reach ⌈N/2⌉ sets
- Loser cannot exceed winner's set count

**Impact**: This rigid approach prevents proper support for:
1. Sports with different scoring systems (dart, chess, fishing, golf)
2. Sports that allow draws (squash in some leagues, chess)
3. Sport-specific rules (tennis tiebreakers, dart checkouts, chess time controls)

## Current Implementation Analysis

### Location: `backend/internal/service/match_service.go`

```go
// Lines 62-97: validateTableTennisScore()
// Used by ALL sports: TABLE_TENNIS, TENNIS, PADEL, BADMINTON, SQUASH, PICKLEBALL

func validateTableTennisScore(setsA, setsB, setsToPlay int32) error {
    // No ties allowed - PROBLEM: Squash sometimes allows draws
    if setsA == setsB {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
    }
    
    // Calculate required wins - Works for racket sports, not for others
    requiredSets := (setsToPlay + 1) / 2
    
    // Validation logic...
}
```

### Switch Statement (Lines 155-169)
```go
switch series.Sport {
case int32(pb.Sport_SPORT_TABLE_TENNIS),
     int32(pb.Sport_SPORT_TENNIS),
     int32(pb.Sport_SPORT_PADEL),
     int32(pb.Sport_SPORT_BADMINTON),
     int32(pb.Sport_SPORT_SQUASH),
     int32(pb.Sport_SPORT_PICKLEBALL):
    // All use TABLE_TENNIS_SETS scoring profile
    // PROBLEM: No sport-specific differentiation
}
```

## Proposed Architecture

### Phase 1: Sport Validator Interface

**Create**: `backend/internal/validation/sport_validator.go`

```go
package validation

import (
    "context"
    pb "github.com/goencoder/klubbspel/backend/proto/gen/go/klubbspel/v1"
    "google.golang.org/grpc/status"
)

// SportValidator defines sport-specific scoring validation rules
type SportValidator interface {
    // ValidateScore checks if the reported score is valid for this sport
    ValidateScore(ctx context.Context, scoreA, scoreB int32, config *pb.SportConfig) error
    
    // AllowsDraws returns true if this sport can end in a draw/tie
    AllowsDraws() bool
    
    // GetScoringDescription returns human-readable scoring rules
    GetScoringDescription() string
    
    // GetDefaultConfig returns default sport configuration
    GetDefaultConfig() *pb.SportConfig
}

// Registry maps sports to their validators
var validatorRegistry = make(map[pb.Sport]SportValidator)

// RegisterValidator registers a sport-specific validator
func RegisterValidator(sport pb.Sport, validator SportValidator) {
    validatorRegistry[sport] = validator
}

// GetValidator returns the validator for a specific sport
func GetValidator(sport pb.Sport) (SportValidator, error) {
    validator, exists := validatorRegistry[sport]
    if !exists {
        return nil, status.Errorf(codes.Unimplemented, 
            "No validator registered for sport: %v", sport)
    }
    return validator, nil
}
```

### Phase 2: Implement Sport-Specific Validators

#### Table Tennis Validator
**File**: `backend/internal/validation/table_tennis_validator.go`

```go
package validation

type TableTennisValidator struct{}

func (v *TableTennisValidator) ValidateScore(ctx context.Context, 
    scoreA, scoreB int32, config *pb.SportConfig) error {
    
    setsToPlay := config.GetSetsToPlay()
    if setsToPlay == 0 {
        setsToPlay = 5 // Default
    }
    
    // No draws in table tennis
    if scoreA == scoreB {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
    }
    
    requiredSets := (setsToPlay + 1) / 2
    
    // Check if either player reached required sets
    if scoreA < requiredSets && scoreB < requiredSets {
        return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_N")
    }
    
    // Check scores don't exceed possible values
    if scoreA > requiredSets || scoreB > requiredSets {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
    }
    
    return nil
}

func (v *TableTennisValidator) AllowsDraws() bool {
    return false
}

func (v *TableTennisValidator) GetScoringDescription() string {
    return "Best-of-N sets format. Each set won counts as 1 point. No draws allowed."
}

func (v *TableTennisValidator) GetDefaultConfig() *pb.SportConfig {
    return &pb.SportConfig{
        AllowsDraws: false,
        ScoringSystem: pb.ScoringSystem_SCORING_SYSTEM_SETS,
        SetsToPlay: 5,
        MinScore: 0,
        MaxScore: 5, // Best of 5 → max 5 sets
    }
}
```

#### Squash Validator (with draw support)
**File**: `backend/internal/validation/squash_validator.go`

```go
package validation

type SquashValidator struct{}

func (v *SquashValidator) ValidateScore(ctx context.Context, 
    scoreA, scoreB int32, config *pb.SportConfig) error {
    
    setsToPlay := config.GetSetsToPlay()
    if setsToPlay == 0 {
        setsToPlay = 5
    }
    
    // Squash CAN allow draws in some leagues
    if scoreA == scoreB {
        if config.GetAllowsDraws() {
            return nil // Draw is valid
        }
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
    }
    
    // Same best-of-N logic as table tennis for decisive results
    requiredSets := (setsToPlay + 1) / 2
    
    if scoreA < requiredSets && scoreB < requiredSets {
        return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_N")
    }
    
    if scoreA > requiredSets || scoreB > requiredSets {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
    }
    
    return nil
}

func (v *SquashValidator) AllowsDraws() bool {
    return true // Configurable per series
}

func (v *SquashValidator) GetScoringDescription() string {
    return "Best-of-N games. Draws allowed in some league formats."
}

func (v *SquashValidator) GetDefaultConfig() *pb.SportConfig {
    return &pb.SportConfig{
        AllowsDraws: true, // Can be configured per series
        ScoringSystem: pb.ScoringSystem_SCORING_SYSTEM_SETS,
        SetsToPlay: 5,
        MinScore: 0,
        MaxScore: 5,
    }
}
```

#### Chess Validator (simple W/D/L)
**File**: `backend/internal/validation/chess_validator.go`

```go
package validation

type ChessValidator struct{}

func (v *ChessValidator) ValidateScore(ctx context.Context, 
    scoreA, scoreB int32, config *pb.SportConfig) error {
    
    // Chess: 1 = win, 0.5 = draw, 0 = loss
    // In our system: scoreA=1, scoreB=0 means A wins
    //                scoreA=0, scoreB=1 means B wins
    //                scoreA=0, scoreB=0 with draw flag means draw
    
    // Must be either 1-0, 0-1, or 0-0 (draw)
    if (scoreA == 1 && scoreB == 0) || 
       (scoreA == 0 && scoreB == 1) ||
       (scoreA == 0 && scoreB == 0) {
        return nil
    }
    
    return status.Error(codes.InvalidArgument, 
        "VALIDATION_CHESS_INVALID: Must be 1-0, 0-1, or 0-0 (draw)")
}

func (v *ChessValidator) AllowsDraws() bool {
    return true // Chess commonly ends in draws
}

func (v *ChessValidator) GetScoringDescription() string {
    return "Win (1-0), Loss (0-1), or Draw (0-0). Time controls vary by tournament."
}

func (v *ChessValidator) GetDefaultConfig() *pb.SportConfig {
    return &pb.SportConfig{
        AllowsDraws: true,
        ScoringSystem: pb.ScoringSystem_SCORING_SYSTEM_WIN_DRAW_LOSS,
        SetsToPlay: 1,
        MinScore: 0,
        MaxScore: 1,
    }
}
```

#### Dart Validator (leg-based with checkout)
**File**: `backend/internal/validation/dart_validator.go`

```go
package validation

type DartValidator struct{}

func (v *DartValidator) ValidateScore(ctx context.Context, 
    scoreA, scoreB int32, config *pb.SportConfig) error {
    
    legsToPlay := config.GetSetsToPlay() // Reuse field for "legs"
    if legsToPlay == 0 {
        legsToPlay = 5 // Best of 5 legs
    }
    
    // No draws in dart
    if scoreA == scoreB {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_TIE")
    }
    
    requiredLegs := (legsToPlay + 1) / 2
    
    // One player must have reached required legs
    if scoreA < requiredLegs && scoreB < requiredLegs {
        return status.Error(codes.InvalidArgument, "VALIDATION_BEST_OF_N_LEGS")
    }
    
    // Cannot exceed required legs
    if scoreA > requiredLegs || scoreB > requiredLegs {
        return status.Error(codes.InvalidArgument, "VALIDATION_SCORE_INVALID")
    }
    
    return nil
}

func (v *DartValidator) AllowsDraws() bool {
    return false
}

func (v *DartValidator) GetScoringDescription() string {
    return "Best-of-N legs. 301 or 501 format. Must finish on a double."
}

func (v *DartValidator) GetDefaultConfig() *pb.SportConfig {
    return &pb.SportConfig{
        AllowsDraws: false,
        ScoringSystem: pb.ScoringSystem_SCORING_SYSTEM_LEGS,
        SetsToPlay: 5, // 5 legs
        MinScore: 0,
        MaxScore: 5,
    }
}
```

### Phase 3: Protobuf Changes

**Add to**: `proto/klubbspel/v1/common.proto`

```protobuf
// Sport-specific configuration for tournaments
message SportConfig {
  // Whether draws/ties are allowed for this sport in this tournament
  bool allows_draws = 1;
  
  // Scoring system used by this sport
  ScoringSystem scoring_system = 2;
  
  // Number of sets/games/legs to play (interpretation depends on sport)
  int32 sets_to_play = 3 [(buf.validate.field).int32 = {gte: 1, lte: 11}];
  
  // Minimum valid score value
  int32 min_score = 4;
  
  // Maximum valid score value
  int32 max_score = 5;
  
  // Human-readable description of scoring rules
  string description = 6;
}

// Different scoring systems across sports
enum ScoringSystem {
  SCORING_SYSTEM_UNSPECIFIED = 0;
  SCORING_SYSTEM_SETS = 1;          // Table tennis, tennis, badminton
  SCORING_SYSTEM_LEGS = 2;          // Darts
  SCORING_SYSTEM_WIN_DRAW_LOSS = 3; // Chess, Go
  SCORING_SYSTEM_WEIGHT = 4;        // Fishing
  SCORING_SYSTEM_STROKES = 5;       // Golf, disc golf
  SCORING_SYSTEM_POINTS = 6;        // Generic point-based
}
```

**Update**: `proto/klubbspel/v1/series.proto`

```protobuf
message Series {
  // ... existing fields ...
  
  // Sport-specific configuration (optional, uses defaults if not set)
  SportConfig sport_config = 11;
}

message CreateSeriesRequest {
  // ... existing fields ...
  
  // Sport-specific configuration (optional)
  SportConfig sport_config = 10;
}
```

### Phase 4: Update Match Service

**Modify**: `backend/internal/service/match_service.go`

```go
import (
    "github.com/goencoder/klubbspel/backend/internal/validation"
)

func (s *MatchService) ReportMatchV2(ctx context.Context, 
    in *pb.ReportMatchV2Request) (*pb.ReportMatchV2Response, error) {
    
    // ... existing validation ...
    
    series, err := s.Series.FindByID(ctx, in.GetSeriesId())
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to find series: %v", err)
    }
    
    // Get sport-specific validator
    validator, err := validation.GetValidator(pb.Sport(series.Sport))
    if err != nil {
        return nil, err
    }
    
    // Get sport config (use defaults if not set)
    sportConfig := series.SportConfig
    if sportConfig == nil {
        sportConfig = validator.GetDefaultConfig()
    }
    
    // Extract scores based on result type
    var scoreA, scoreB int32
    switch result := in.GetResult().(type) {
    case *pb.MatchResult_TableTennis:
        scoreA = result.TableTennis.GetSetsA()
        scoreB = result.TableTennis.GetSetsB()
    default:
        return nil, status.Error(codes.InvalidArgument, "VALIDATION_RESULT_TYPE_INVALID")
    }
    
    // Use sport-specific validation
    if err := validator.ValidateScore(ctx, scoreA, scoreB, sportConfig); err != nil {
        return nil, err
    }
    
    // ... rest of implementation ...
}
```

## Implementation Plan

### Step 1: Foundation (Week 1)
- [ ] Create `validation` package with `SportValidator` interface
- [ ] Implement validator registry
- [ ] Add `SportConfig` and `ScoringSystem` to protobuf
- [ ] Generate protobuf code

### Step 2: Core Validators (Week 2)
- [ ] Implement `TableTennisValidator` (extract from existing code)
- [ ] Implement `TennisValidator` (same as table tennis initially)
- [ ] Implement `SquashValidator` (with draw support)
- [ ] Write comprehensive unit tests

### Step 3: Integration (Week 3)
- [ ] Update `MatchService.ReportMatchV2()` to use validators
- [ ] Add default sport configs for all existing sports
- [ ] Update series creation to accept `SportConfig`
- [ ] Integration tests for all validators

### Step 4: Frontend Support (Week 4)
- [ ] Update `CreateSeriesDialog` to show sport-specific options
- [ ] Add "Allow Draws" checkbox for applicable sports
- [ ] Update `ReportMatchDialog` validation hints per sport
- [ ] Add sport-specific help text

### Step 5: Future Sports (Ongoing)
- [ ] Implement `ChessValidator`
- [ ] Implement `DartValidator`
- [ ] Implement `FishingValidator` (weight-based)
- [ ] Implement `GolfValidator` (stroke/match play)

## Benefits

1. **Extensibility**: Adding new sports becomes straightforward - just implement interface
2. **Maintainability**: Sport rules isolated in dedicated validators
3. **Testability**: Each validator can be unit tested independently
4. **Flexibility**: Tournament organizers can configure sport-specific rules
5. **Correctness**: Proper enforcement of sport-specific scoring rules

## Migration Strategy

1. **Backward Compatibility**: Existing series without `SportConfig` use defaults
2. **Gradual Rollout**: Start with racket sports, add others incrementally
3. **Data Migration**: Optional migration to add `SportConfig` to existing series
4. **UI Updates**: Phased UI improvements to expose new configuration options

## Testing Requirements

### Unit Tests
- Each validator with comprehensive test cases
- Edge cases (minimum scores, maximum scores, draws)
- Invalid input handling

### Integration Tests
- Match reporting with different sport configs
- Series creation with custom sport configs
- Backward compatibility with existing series

### UI Tests (Playwright)
- Sport-specific series creation
- Draw handling in applicable sports
- Validation message correctness

## Documentation Updates

- [ ] Update `CODEX_ADD_SPORTS.md` with validator requirements
- [ ] Create sport validator implementation guide
- [ ] Document `SportConfig` options in API docs
- [ ] Add sport-specific scoring rules to user documentation

---

**Priority**: HIGH - Blocks proper implementation of chess, dart, fishing, golf, and other non-racket sports

**Estimated Effort**: 4 weeks (1 developer)

**Dependencies**: None - can be implemented independently

**Breaking Changes**: None if implemented with backward compatibility
