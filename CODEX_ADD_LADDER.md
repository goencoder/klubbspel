# Codex Prompt: Implement Ladder Series Format

## Context

Klubbspel currently supports `SERIES_FORMAT_OPEN_PLAY` (free play) and has protobuf definitions for `SERIES_FORMAT_LADDER` and `SERIES_FORMAT_CUP`, but only OPEN_PLAY is implemented. We need to implement the ladder format functionality.

## Current State

- ✅ Protobuf enum `SERIES_FORMAT_LADDER` exists in `proto/klubbspel/v1/series.proto`
- ✅ Frontend shows "Fritt spel" (Free play) for open play format
- ✅ Match reporting works for free play
- ✅ ELO rankings are calculated and displayed in leaderboard
- ❌ Ladder-specific ranking logic is not implemented
- ❌ UI doesn't show ladder-specific information
- ❌ No position tracking for ladder format

## Objective

Implement a **Challenge Ladder** format where:
1. Players are ranked by position (1, 2, 3, etc.)
2. Any player can challenge any player above them (we handle scheduling externally)
3. Match outcomes affect ladder positions according to ladder rules
4. Leaderboard displays positions clearly for ladder series

## Ladder Rules to Implement

### **Rule Set: Classic Challenge Ladder**

**Initial Ranking:**
- When series starts, players are ranked by their current ELO rating
- Position 1 = highest ELO, Position 2 = second highest, etc.
- New players joining mid-series enter at the bottom

**Match Outcomes:**

1. **Challenger Wins**:
   - Challenger takes the challenged player's position
   - All players between them move down one position
   - Example: Player at #4 beats player at #1
     - Before: 1-Alice, 2-Bob, 3-Carol, 4-Dave
     - After: 1-Dave, 2-Alice, 3-Bob, 4-Carol

2. **Challenger Loses**:
   - Challenger drops one position (penalty for failed challenge)
   - Player below challenger moves up one position
   - Challenged player stays in same position
   - Example: Player at #3 loses to player at #1
     - Before: 1-Alice, 2-Bob, 3-Carol, 4-Dave
     - After: 1-Alice, 2-Bob, 3-Dave, 4-Carol

3. **Same-Position Challenges** (edge case):
   - If positions are equal (shouldn't happen), treat as free play
   - No position changes

**Important Notes:**
- Position changes happen immediately when match is reported
- ELO is still calculated but **position determines ranking**, not ELO
- Position is **series-specific** - a player can be #1 in one ladder and #5 in another

## Technical Implementation Requirements

### 1. Backend Changes

#### Database Schema (`backend/internal/repo/`)
Add to `SeriesPlayer` document (or create new collection):
```go
type SeriesPlayer struct {
    SeriesID   primitive.ObjectID `bson:"series_id"`
    PlayerID   string             `bson:"player_id"`
    Position   int                `bson:"position"`      // NEW: Ladder position
    JoinedAt   time.Time          `bson:"joined_at"`
    UpdatedAt  time.Time          `bson:"updated_at"`
}
```

Create index:
```go
db.series_players.createIndex(
    { "series_id": 1, "position": 1 }, 
    { unique: true }
)
```

#### Match Service (`backend/internal/service/match_service.go`)
Modify `ReportMatch` to:
1. Check if series format is `SERIES_FORMAT_LADDER`
2. After validating and storing match, call ladder position update logic
3. Use transaction to ensure atomicity

Add new function:
```go
func (s *MatchService) updateLadderPositions(
    ctx context.Context, 
    seriesID primitive.ObjectID,
    winnerID string,
    loserID string,
) error {
    // 1. Get current positions
    // 2. Determine who challenged whom (lower position challenges higher)
    // 3. Apply ladder rules
    // 4. Update all affected positions atomically
}
```

#### Series Service (`backend/internal/service/series_service.go`)
- When player joins ladder series, assign position (bottom of ladder)
- When series starts, initialize positions based on ELO
- Add RPC to get ladder standings

#### New RPC Methods (add to `series.proto`)
```protobuf
// Get ladder standings for a series
rpc GetLadderStandings(GetLadderStandingsRequest) returns (GetLadderStandingsResponse) {
  option (google.api.http) = {
    get: "/v1/series/{series_id}/ladder"
  };
}

message LadderEntry {
  string player_id = 1;
  string player_name = 2;
  int32 position = 3;
  int32 matches_played = 4;
  int32 matches_won = 5;
  google.protobuf.Timestamp last_match_at = 6;
}

message GetLadderStandingsRequest {
  string series_id = 1;
}

message GetLadderStandingsResponse {
  repeated LadderEntry entries = 1;
}
```

### 2. Frontend Changes

#### UI Components

**Series List Page (`frontend/src/pages/SeriesListPage.tsx`)**
- Show format badge: "Stege" for ladder, "Fritt spel" for open play

**Create Series Dialog (`frontend/src/components/CreateSeriesDialog.tsx`)**
- Add format selector dropdown:
  ```tsx
  <Select value={format} onValueChange={setFormat}>
    <SelectItem value="SERIES_FORMAT_OPEN_PLAY">
      {t('series.format.openPlay')}
    </SelectItem>
    <SelectItem value="SERIES_FORMAT_LADDER">
      {t('series.format.ladder')}
    </SelectItem>
  </Select>
  ```

**Series Detail Page (`frontend/src/pages/SeriesDetailPage.tsx`)**
- For ladder series, fetch and display ladder standings instead of ELO leaderboard
- Show position column prominently
- Show up/down arrows indicating recent position changes
- Add "Challenge" context to match history

**Leaderboard Display for Ladder**
Create new component: `LadderStandings.tsx`
```tsx
interface LadderStandingsProps {
  seriesId: string;
}

// Display:
// Position | Player | Matches | W/L | Last Match
//    1     | Alice  |   12    | 9-3 | 2 days ago
//    2     | Bob    |   10    | 7-3 | 1 day ago
```

**Match Reporting (`frontend/src/components/ReportMatchDialog.tsx`)**
- Show current positions when reporting match in ladder
- Example: "Dave (#4) challenges Alice (#1)"
- After submission, show position changes as notification

#### Translations (`frontend/src/i18n/locales/`)

**Swedish (`sv.json`)**:
```json
{
  "series": {
    "format": {
      "openPlay": "Fritt spel",
      "ladder": "Stege",
      "cup": "Cup"
    },
    "ladder": {
      "position": "Position",
      "standings": "Stegeställning",
      "challenge": "Utmaning",
      "positionChange": "Positionsändring",
      "movedUp": "Klättrade {{positions}} position(er)",
      "movedDown": "Föll {{positions}} position(er)"
    }
  }
}
```

**English (`en.json`)**:
```json
{
  "series": {
    "format": {
      "openPlay": "Free Play",
      "ladder": "Ladder",
      "cup": "Cup"
    },
    "ladder": {
      "position": "Position",
      "standings": "Ladder Standings",
      "challenge": "Challenge",
      "positionChange": "Position Change",
      "movedUp": "Moved up {{positions}} position(s)",
      "movedDown": "Moved down {{positions}} position(s)"
    }
  }
}
```

### 3. Migration

Create migration to:
1. Initialize positions for existing ladder series (if any exist)
2. Create `series_players` collection with proper indexes

Location: `backend/internal/migration/init_ladder_positions.go`

## Implementation Steps (Priority Order)

### Phase 1: Backend Foundation
1. ✅ Create `series_players` collection schema and indexes
2. ✅ Implement ladder position calculation logic
3. ✅ Add `GetLadderStandings` RPC method
4. ✅ Update `ReportMatch` to handle ladder position changes
5. ✅ Write unit tests for ladder logic

### Phase 2: Frontend Basic Support
6. ✅ Add format selector to CreateSeriesDialog
7. ✅ Add translations for ladder terminology
8. ✅ Create LadderStandings component
9. ✅ Update SeriesDetailPage to show ladder when format is LADDER

### Phase 3: Enhanced UX
10. ✅ Show position indicators in match reporting
11. ✅ Add position change notifications
12. ✅ Show format badge on series cards
13. ✅ Add "Challenge" context to match history

### Phase 4: Testing & Polish
14. ✅ Integration tests for ladder scenarios
15. ✅ UI testing with Playwright
16. ✅ Documentation updates

## Testing Scenarios

### Test Case 1: Basic Challenge Win
```
Given: 4 players - Alice(#1), Bob(#2), Carol(#3), Dave(#4)
When: Dave challenges and beats Alice
Then: New order - Dave(#1), Alice(#2), Bob(#3), Carol(#4)
```

### Test Case 2: Challenge Loss with Penalty
```
Given: 4 players - Alice(#1), Bob(#2), Carol(#3), Dave(#4)
When: Carol challenges Alice and loses
Then: New order - Alice(#1), Bob(#2), Dave(#3), Carol(#4)
```

### Test Case 3: Adjacent Position Challenge
```
Given: 4 players - Alice(#1), Bob(#2), Carol(#3), Dave(#4)
When: Bob challenges and beats Alice
Then: New order - Bob(#1), Alice(#2), Carol(#3), Dave(#4)
```

### Test Case 4: Multiple Matches in Quick Succession
```
Given: 4 players - Alice(#1), Bob(#2), Carol(#3), Dave(#4)
When: Dave beats Alice (becomes #1)
And: Carol beats Bob (becomes #2, Bob becomes #3)
And: Bob beats Dave (Bob becomes #1, Dave becomes #2)
Then: Verify all position changes are atomic and consistent
```

## Edge Cases to Handle

1. **Concurrent Match Reports**: Use database transactions to prevent race conditions
2. **Player Joins Mid-Series**: Add to bottom of ladder
3. **Player Leaves Series**: Remove from ladder, others maintain positions (no cascade)
4. **Tie/Draw Matches**: Not applicable for racket sports (always a winner)
5. **First Match in Series**: Initialize positions if not already set

## Database Queries to Implement

```javascript
// Get ladder standings
db.series_players.find({ series_id: ObjectId("...") })
  .sort({ position: 1 })
  .lookup({ from: "players", localField: "player_id", foreignField: "_id" })

// Update positions atomically (transaction required)
session.startTransaction()
try {
  // Update winner
  db.series_players.updateOne(
    { series_id: sid, player_id: winner },
    { $set: { position: challenged_position, updated_at: now } }
  )
  
  // Update all players in range
  db.series_players.updateMany(
    { 
      series_id: sid, 
      position: { $gte: challenged_position, $lt: challenger_position } 
    },
    { $inc: { position: 1 }, $set: { updated_at: now } }
  )
  
  session.commitTransaction()
} catch (error) {
  session.abortTransaction()
}
```

## Success Criteria

✅ Can create a ladder series via UI  
✅ Players have positions in ladder series  
✅ Match reports update positions correctly according to ladder rules  
✅ Ladder standings display correctly sorted by position  
✅ Position changes show in match history  
✅ All edge cases handled gracefully  
✅ Unit tests cover ladder logic  
✅ Integration tests verify end-to-end flow  
✅ UI is clear and intuitive for ladder format  

## Files to Modify

**Backend:**
- `proto/klubbspel/v1/series.proto` (add RPC methods)
- `backend/internal/repo/series_repo.go` (ladder queries)
- `backend/internal/service/series_service.go` (ladder standings RPC)
- `backend/internal/service/match_service.go` (position update logic)
- `backend/internal/migration/init_ladder_positions.go` (NEW)

**Frontend:**
- `frontend/src/components/CreateSeriesDialog.tsx` (format selector)
- `frontend/src/components/LadderStandings.tsx` (NEW)
- `frontend/src/components/ReportMatchDialog.tsx` (position context)
- `frontend/src/pages/SeriesDetailPage.tsx` (conditional ladder display)
- `frontend/src/pages/SeriesListPage.tsx` (format badge)
- `frontend/src/i18n/locales/sv.json` (translations)
- `frontend/src/i18n/locales/en.json` (translations)
- `frontend/src/types/api.ts` (TypeScript types)

## Reference: Existing Code Patterns

Follow these patterns from the existing codebase:

1. **Match Reporting**: See `match_service.go` `ReportMatch` function
2. **Leaderboard Display**: See `frontend/src/components/Leaderboard.tsx` (if exists)
3. **Series Creation**: See `series_service.go` `CreateSeries` function
4. **ELO Calculation**: Leverage existing ELO logic, but positions override for ladder

## Notes for Codex

- This is a **MINOR version feature** (v1.3.0)
- Follow existing code patterns from CODEX_ADD_SPORTS.md
- Use transactions for position updates to ensure data consistency
- Test thoroughly - ladder logic is complex and prone to edge cases
- Consider performance: updating many positions should be efficient
- UI should clearly distinguish ladder series from free play series

## Questions to Consider

1. Should we limit challenge range (e.g., can only challenge within 3 positions)?
   - **Recommendation**: Start without limits, add as configuration later
   
2. Should there be a challenge cooldown period?
   - **Recommendation**: Out of scope for v1, handle externally
   
3. What happens to ELO in ladder series?
   - **Recommendation**: Still calculate ELO, but position is primary ranking
   
4. Can a player participate in both free play and ladder series simultaneously?
   - **Answer**: Yes, they're independent series

## Alternative: Simpler "Soft Ladder" Approach

If full position management is too complex for first iteration, consider:
- Keep ELO-based ranking
- Add "position history" to track movements
- Show trend arrows (↑↓) based on ELO changes
- Label as "Ladder" but use ELO under the hood

This gives ladder "feel" without complex position management. Can evolve to full ladder later.

---

**Priority**: Medium-High  
**Complexity**: High (database transactions, position management, UI changes)  
**Estimated Effort**: 2-3 days for experienced developer  
**Dependencies**: None (ladder enum already exists)  
**Risk**: Position race conditions if not using transactions properly
