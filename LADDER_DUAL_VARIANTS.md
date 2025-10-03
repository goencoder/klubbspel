# Ladder Dual Variants Implementation

## Overview
Extended Codex's ladder implementation to support **TWO ladder rule variants**:
1. **Classic (Default)**: Loser keeps position (no penalty)
2. **Aggressive**: Loser drops one position (penalty)

## Changes Made

### 1. Protocol Buffers (`proto/klubbspel/v1/series.proto`)

#### Added LadderRules Enum
```protobuf
enum LadderRules {
  LADDER_RULES_UNSPECIFIED = 0;
  LADDER_RULES_CLASSIC = 1;      // No penalty on loss
  LADDER_RULES_AGGRESSIVE = 2;   // Penalty on loss
}
```

#### Added Fields
- `Series.ladder_rules` (field 11): Specifies which variant to use
- `CreateSeriesRequest.ladder_rules` (field 10): Optional, defaults to CLASSIC

### 2. Backend Repository (`backend/internal/repo/series_repo.go`)

#### Updated Series Model
```go
type Series struct {
    // ... existing fields ...
    LadderRules    int32  `bson:"ladder_rules"`    // NEW
    // ... rest of fields ...
}
```

#### Updated Create Method Signature
```go
func (r *SeriesRepo) Create(ctx context.Context, clubID, title string, 
    startsAt, endsAt time.Time, visibility int32, 
    sport, format, ladderRules, scoringProfile, setsToPlay int32) (*Series, error)
```

### 3. Backend Service (`backend/internal/service/series_service.go`)

#### CreateSeries Enhancement
- Defaults `ladder_rules` to `LADDER_RULES_CLASSIC` for LADDER format series
- Passes `ladder_rules` to repository Create method
- Returns `ladder_rules` in response

#### All Series Responses Updated
Added `LadderRules: pb.LadderRules(series.LadderRules)` to:
- `CreateSeries`
- `ListSeries`
- `GetSeries`
- `UpdateSeries`

### 4. Match Service (`backend/internal/service/match_service.go`)

#### Updated updateLadderPositions Logic
```go
// Fetch series to check ladder rules
series, err := s.Series.FindByID(ctx, seriesID)

// Challenger lost: apply ladder rules
ladderRules := pb.LadderRules(series.LadderRules)
if ladderRules == pb.LadderRules_LADDER_RULES_UNSPECIFIED || 
   ladderRules == pb.LadderRules_LADDER_RULES_CLASSIC {
    // Classic: loser keeps position (just touch timestamp)
    return s.SeriesPlayers.TouchPlayer(sessCtx, seriesID, challenger.PlayerID, now)
}

// Aggressive: loser drops one position
newPosition := challenger.Position + 1
// ... swap positions ...
```

## Behavior Specification

### Classic Rules (LADDER_RULES_CLASSIC)
**Challenger Wins**: Takes challenged position, others shift down ✅  
**Challenger Loses**: Keeps current position (no penalty) ✅

### Aggressive Rules (LADDER_RULES_AGGRESSIVE)
**Challenger Wins**: Takes challenged position, others shift down ✅  
**Challenger Loses**: Drops one position (swaps with player below) ✅

### Common Behavior
- If positions are equal: Both players just update timestamp
- Transaction-safe: All position updates are atomic
- Indexes: Auto-created on `(series_id, player_id)` and `(series_id, position)`

## Database Migration

**NO MIGRATION REQUIRED** ✅

Reasons:
1. **Indexes**: Auto-created in `NewSeriesPlayerRepo()` constructor
2. **New Collection**: `series_players` created on-demand when first player joins
3. **New Field**: `ladder_rules` in `series` collection defaults to `0` (UNSPECIFIED), which we treat as CLASSIC

Existing LADDER series without `ladder_rules` will default to CLASSIC behavior.

## Default Behavior

- **Format**: When not specified → `SERIES_FORMAT_OPEN_PLAY` (existing)
- **LadderRules**: When format is LADDER and not specified → `LADDER_RULES_CLASSIC` (NEW)

This makes Classic the **safe default** (no penalty on loss).

## API Usage Examples

### Create Classic Ladder (Default)
```bash
curl -X POST http://localhost:8080/v1/series \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Classic Ladder",
    "club_id": "club123",
    "format": "SERIES_FORMAT_LADDER",
    "starts_at": "2025-01-01T00:00:00Z",
    "ends_at": "2025-12-31T23:59:59Z"
  }'
# ladder_rules defaults to LADDER_RULES_CLASSIC
```

### Create Aggressive Ladder (Explicit)
```bash
curl -X POST http://localhost:8080/v1/series \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Aggressive Ladder",
    "club_id": "club123",
    "format": "SERIES_FORMAT_LADDER",
    "ladder_rules": "LADDER_RULES_AGGRESSIVE",
    "starts_at": "2025-01-01T00:00:00Z",
    "ends_at": "2025-12-31T23:59:59Z"
  }'
```

### Get Ladder Standings
```bash
# Same endpoint for both variants
curl http://localhost:8080/v1/series/67976fba24aa64bde5a0b6b0/ladder
```

## Testing Checklist

### Unit Tests Needed
- [ ] Classic rules: Challenger loses → position unchanged
- [ ] Aggressive rules: Challenger loses → drops one position
- [ ] Both rules: Challenger wins → takes challenged position
- [ ] Edge case: Bottom position loses in aggressive (no position below)
- [ ] Transaction rollback on error

### Integration Tests Needed
- [ ] Create series with Classic rules via API
- [ ] Create series with Aggressive rules via API
- [ ] Report matches and verify position changes
- [ ] Verify GetLadderStandings returns correct order

### Manual Testing
- [ ] Create both ladder variants via UI (when frontend ready)
- [ ] Report matches and observe position changes
- [ ] Verify ladder standings display correctly
- [ ] Test concurrent match reporting (transaction safety)

## Frontend Changes Required (Future PR)

### CreateSeriesDialog Component
- Add "Ladder Rules" selector when format is LADDER
- Options: "Classic (No penalty)" and "Aggressive (Penalty on loss)"
- Default to Classic
- Help text explaining the difference

### SeriesCard Component
- Show ladder rules badge/tag for ladder series
- Example: "Ladder · Classic" or "Ladder · Aggressive"

### Translations
```json
{
  "series.ladder_rules": "Ladder Rules",
  "series.ladder_rules.classic": "Classic",
  "series.ladder_rules.aggressive": "Aggressive",
  "series.ladder_rules.classic.description": "Challenger keeps position if they lose",
  "series.ladder_rules.aggressive.description": "Challenger drops one position if they lose"
}
```

## Files Modified

**Protobuf**:
- `proto/klubbspel/v1/series.proto` (+9 lines)

**Backend**:
- `backend/internal/repo/series_repo.go` (+1 field, +1 parameter)
- `backend/internal/service/series_service.go` (+13 lines)
- `backend/internal/service/match_service.go` (+13 lines)

**Generated**:
- `backend/proto/gen/go/klubbspel/v1/*.go` (auto-generated)
- `backend/openapi/klubbspel.swagger.json` (auto-generated)

## Build & Lint Status

✅ Backend builds successfully  
✅ No linting errors  
✅ Protobuf generation successful  
✅ OpenAPI spec generated

## Next Steps

1. ✅ Commit changes to `codex/implement-ladder-series-format` branch
2. ⏳ Add backend tests for dual variants
3. ⏳ Implement frontend UI for ladder rules selection
4. ⏳ Add frontend tests (Playwright)
5. ⏳ Update translations (en.json, sv.json)
6. ⏳ Manual end-to-end testing
7. ⏳ Merge PR #22

## Notes

- **Backward Compatibility**: Existing LADDER series without `ladder_rules` field will default to Classic (safe default)
- **Transaction Safety**: Maintained from Codex's original implementation
- **Index Performance**: No additional indexes needed, existing indexes sufficient
- **Code Quality**: 0 linting errors, follows existing patterns
