# 🎾 Add Support for Low-Hanging Racket/Paddle Sports

## Context
We successfully added tennis support to Klubbspel (v1.1.0), but encountered **three critical issues** during implementation that must be avoided:

### ⚠️ Issues Encountered with Tennis Implementation:
1. **Backend incomplete**: Match service didn't include the new sport in the switch statement → 501 error
2. **Wrong/missing icons**: Initially used non-existent Lucide icons (TableTennis, TennisBall)
3. **Translation namespace error**: Used `sport.*` instead of `sports.*` (incorrect namespace)

## 🎯 Objective
Add support for **racket/paddle sports** that use the same best-of-N sets scoring format as table tennis and tennis. These are "low-hanging fruit" because they require minimal changes.

## ✅ Sports to Add (in priority order)

### High Priority (Perfect Fit):
1. **SPORT_PADEL** (enum value 3 already exists in proto)
2. **SPORT_BADMINTON** (racket sport, sets-based)
3. **SPORT_SQUASH** (racket sport, sets-based)
4. **SPORT_PICKLEBALL** (paddle sport, sets-based, growing popularity)

### Medium Priority (Good Fit):
5. **SPORT_RACQUETBALL** (racket sport, sets-based)
6. **SPORT_BEACH_TENNIS** (variant of tennis, sets-based)

## 📋 Complete Implementation Checklist

### 1. Protocol Buffers (`proto/klubbspel/v1/common.proto`)

Add new enum values after SPORT_PADEL = 3:

```proto
enum Sport {
  SPORT_UNSPECIFIED = 0;
  SPORT_TABLE_TENNIS = 1;
  SPORT_TENNIS = 2;
  SPORT_PADEL = 3;
  SPORT_BADMINTON = 4;      // ADD
  SPORT_SQUASH = 5;         // ADD
  SPORT_PICKLEBALL = 6;     // ADD
  SPORT_RACQUETBALL = 7;    // ADD (optional)
  SPORT_BEACH_TENNIS = 8;   // ADD (optional)
}
```

### 2. Backend Match Service (`backend/internal/service/match_service.go`)

**CRITICAL**: Add ALL new sports to the existing switch case around line 156:

**Before:**
```go
switch series.Sport {
case int32(pb.Sport_SPORT_TABLE_TENNIS), int32(pb.Sport_SPORT_TENNIS):
    // For table tennis and tennis, use TABLE_TENNIS_SETS scoring
    // Both sports use the same best-of-N sets format
    ttResult := in.GetResult().GetTableTennis()
```

**After:**
```go
switch series.Sport {
case int32(pb.Sport_SPORT_TABLE_TENNIS), 
     int32(pb.Sport_SPORT_TENNIS),
     int32(pb.Sport_SPORT_PADEL),
     int32(pb.Sport_SPORT_BADMINTON),
     int32(pb.Sport_SPORT_SQUASH),
     int32(pb.Sport_SPORT_PICKLEBALL):
    // All racket/paddle sports use the same best-of-N sets scoring format
    ttResult := in.GetResult().GetTableTennis()
```

**⚠️ DO NOT FORGET THIS STEP** - This caused the 501 error with tennis!

### 3. Backend Club Service (`backend/internal/service/club_service.go`)

Add to `supportedClubSports` map around line 35:

**Before:**
```go
var supportedClubSports = map[pb.Sport]struct{}{
    pb.Sport_SPORT_TABLE_TENNIS: {},
    pb.Sport_SPORT_TENNIS:       {},
}
```

**After:**
```go
var supportedClubSports = map[pb.Sport]struct{}{
    pb.Sport_SPORT_TABLE_TENNIS: {},
    pb.Sport_SPORT_TENNIS:       {},
    pb.Sport_SPORT_PADEL:        {},
    pb.Sport_SPORT_BADMINTON:    {},
    pb.Sport_SPORT_SQUASH:       {},
    pb.Sport_SPORT_PICKLEBALL:   {},
}
```

### 4. Frontend Sports Config (`frontend/src/lib/sports.ts`)

#### 4a. Import new icons (verify they exist at https://lucide.dev):

**Before:**
```typescript
import { Circle, CircleDot } from 'lucide-react'
```

**After:**
```typescript
import { Circle, CircleDot, Swords, Wind, Zap } from 'lucide-react'
```

#### 4b. Update SUPPORTED_SPORTS array:

**Before:**
```typescript
export const SUPPORTED_SPORTS: Sport[] = [DEFAULT_SPORT, 'SPORT_TENNIS']
```

**After:**
```typescript
export const SUPPORTED_SPORTS: Sport[] = [
  DEFAULT_SPORT,
  'SPORT_TENNIS',
  'SPORT_PADEL',
  'SPORT_BADMINTON',
  'SPORT_SQUASH',
  'SPORT_PICKLEBALL'
]
```

#### 4c. Update sportTranslationKey() - Use correct namespace `sports.*`:

**Before:**
```typescript
export function sportTranslationKey(sport: Sport): string {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return 'sports.table_tennis'
    case 'SPORT_TENNIS':
      return 'sports.tennis'
    default:
      return 'sports.unknown'
  }
}
```

**After:**
```typescript
export function sportTranslationKey(sport: Sport): string {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return 'sports.table_tennis'
    case 'SPORT_TENNIS':
      return 'sports.tennis'
    case 'SPORT_PADEL':
      return 'sports.padel'
    case 'SPORT_BADMINTON':
      return 'sports.badminton'
    case 'SPORT_SQUASH':
      return 'sports.squash'
    case 'SPORT_PICKLEBALL':
      return 'sports.pickleball'
    default:
      return 'sports.unknown'
  }
}
```

#### 4d. Update sportIconComponent() - Use EXISTING Lucide icons only:

**CRITICAL**: Verify icons exist at https://lucide.dev before using!

**Before:**
```typescript
export function sportIconComponent(sport: Sport): LucideIcon {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return CircleDot  // Represents ping pong ball
    case 'SPORT_TENNIS':
      return Circle     // Represents tennis ball
    default:
      return Circle
  }
}
```

**After:**
```typescript
export function sportIconComponent(sport: Sport): LucideIcon {
  switch (sport) {
    case 'SPORT_TABLE_TENNIS':
      return CircleDot      // Ping pong ball
    case 'SPORT_TENNIS':
      return Circle         // Tennis ball
    case 'SPORT_PADEL':
      return Swords         // Crossed paddles/rackets
    case 'SPORT_BADMINTON':
      return Wind           // Shuttlecock/speed
    case 'SPORT_SQUASH':
      return Zap            // Fast-paced
    case 'SPORT_PICKLEBALL':
      return CircleDot      // Similar to table tennis
    default:
      return Circle
  }
}
```

**Alternative icon options** (verify they exist): Trophy, Target, Activity, Disc, Dumbbell

### 5. Translations (`frontend/src/i18n/locales/`)

#### Swedish (`frontend/src/i18n/locales/sv.json`)

Add to the existing `sports` object (around line 371):

**Before:**
```json
{
  "sports": {
    "table_tennis": "Bordtennis",
    "tennis": "Tennis",
    "padel": "Padel",
    "unknown": "Okänd sport"
  }
}
```

**After:**
```json
{
  "sports": {
    "table_tennis": "Bordtennis",
    "tennis": "Tennis",
    "padel": "Padel",
    "badminton": "Badminton",
    "squash": "Squash",
    "pickleball": "Pickleball",
    "racquetball": "Racquetball",
    "beach_tennis": "Beachtennis",
    "unknown": "Okänd sport"
  }
}
```

#### English (`frontend/src/i18n/locales/en.json`)

Add to the existing `sports` object (around line 371):

**Before:**
```json
{
  "sports": {
    "table_tennis": "Table tennis",
    "tennis": "Tennis",
    "padel": "Padel",
    "unknown": "Unknown sport"
  }
}
```

**After:**
```json
{
  "sports": {
    "table_tennis": "Table tennis",
    "tennis": "Tennis",
    "padel": "Padel",
    "badminton": "Badminton",
    "squash": "Squash",
    "pickleball": "Pickleball",
    "racquetball": "Racquetball",
    "beach_tennis": "Beach tennis",
    "unknown": "Unknown sport"
  }
}
```

## 🧪 Testing Requirements

### Sanity Checks (MUST ALL PASS):

```bash
# Run all sanity checks - ALL must pass with 0 errors
make lint           # Backend + frontend linting (0 issues)
make be.build       # Backend compilation
make test           # Backend unit tests
cd frontend && npm run build  # Frontend compilation
```

### Manual Validation for EACH Sport:

1. Start dev environment: `make host-dev`
2. Open http://localhost:5000
3. **Create series** with the new sport (e.g., Badminton Championship 2025)
4. **Report a match** between two players (e.g., 3-1 score)
5. **Verify match appears** in match list
6. **Check leaderboard** - ELO ratings updated correctly (winner gains, loser loses)
7. **Navigate to Clubs page** - verify sport icon displays correctly
8. **Check translations** - Swedish and English labels show correctly (NOT `sport.badminton`)

### End-to-End Test Scenarios:

For each sport (badminton, squash, padel, pickleball), perform this test:

```bash
# Scenario: Create series, report match, verify leaderboard

# 1. Navigate to Series page
# 2. Click "Skapa ny serie" (Create new series)
# 3. Fill in form:
#    - Name: "Badminton Championship 2025"
#    - Sport: Select "Badminton" from dropdown
#    - Dates: Valid date range
# 4. Click "Skapa" (Create)
# 5. Should see success message
# 6. Navigate to series detail
# 7. Click "Rapportera matcher" (Report matches)
# 8. Select Player A and Player B
# 9. Enter scores: Player A: 3, Player B: 1
# 10. Click "Rapportera matcher" (Report match)
# 11. Should see success message (NOT 501 error!)
# 12. Match should appear in match list
# 13. Click "Resultattabell" tab
# 14. Verify Player A has higher rating than Player B
# 15. Navigate to Clubs page
# 16. Verify sport icon displays next to club name
```

### API Test (Optional - for debugging):

```bash
# Test match reporting endpoint directly
curl -X POST http://localhost:8080/v1/matches \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "series_id": "YOUR_SERIES_ID",
    "player_a_id": "PLAYER_A_ID",
    "player_b_id": "PLAYER_B_ID",
    "result": {
      "table_tennis": {
        "sets_a": 3,
        "sets_b": 1
      }
    },
    "played_at": "2025-01-05T10:00:00Z"
  }'

# Should return 200 OK with match details
# Should NOT return 501 Not Implemented
```

## 🚫 Common Mistakes to Avoid

| Mistake | Impact | Solution |
|---------|--------|----------|
| ❌ Forgetting match service switch case | 501 error when reporting matches | Add ALL sports to the case statement |
| ❌ Using non-existent Lucide icons | Runtime import errors, page crashes | Verify at https://lucide.dev first |
| ❌ Wrong translation namespace (`sport.*` vs `sports.*`) | Translation keys show instead of text | Use `sports.*` namespace |
| ❌ Not adding to supportedClubSports map | Sport not selectable in UI | Add to map in club_service.go |
| ❌ Not adding to SUPPORTED_SPORTS array | Sport doesn't appear in dropdowns | Add to array in sports.ts |
| ❌ Inconsistent naming (snake_case) | Translation failures | Use snake_case consistently |
| ❌ Not running `make generate` after proto changes | Type errors in frontend | Always run after proto changes |

## ✅ Success Criteria

- [ ] All 6 files modified (proto, 2 backend service files, 1 frontend sports file, 2 i18n files)
- [ ] Run `make generate` after proto changes
- [ ] All sanity checks pass (lint, build, test)
- [ ] Can create series for each new sport via UI
- [ ] Can report matches for each new sport (no 501 errors)
- [ ] Match appears in match list after reporting
- [ ] Leaderboards calculate correctly for all sports
- [ ] Icons display properly in clubs page (verified in Lucide library beforehand)
- [ ] Translations work in both Swedish and English (no raw keys like `sport.badminton`)
- [ ] Backward compatible (existing tennis/table tennis data still works)
- [ ] No database migration needed (sports share same scoring format)

## 📦 Deliverable

### Branch Name
```
codex/add-racket-paddle-sports
```

### PR Title
```
Add support for badminton, squash, padel, and pickleball
```

### Commit Message Template
```
feat: Add support for racket/paddle sports (badminton, squash, padel, pickleball)

All sports use the same best-of-N sets scoring format as table tennis/tennis,
making them low-hanging fruit for multi-sport support expansion.

Changes:
- proto: Add SPORT_BADMINTON, SPORT_SQUASH, SPORT_PICKLEBALL enum values
- backend/match_service: Extend switch case to support all new sports
- backend/club_service: Add sports to supportedClubSports validation map
- frontend/sports.ts: Update SUPPORTED_SPORTS array and icon/translation mappings
- i18n: Add translations for Swedish and English

Icons used (all verified in Lucide library):
- Badminton: Wind (represents shuttlecock speed)
- Squash: Zap (represents fast-paced gameplay)
- Padel: Swords (represents crossed paddles)
- Pickleball: CircleDot (similar visual to table tennis)

Testing:
- ✅ All sanity checks pass (lint, build, test)
- ✅ Created series for each sport via UI
- ✅ Reported matches for each sport successfully
- ✅ Verified leaderboard calculations work correctly
- ✅ Confirmed icons display and translations render properly
- ✅ Backward compatible with existing tennis/table tennis data

No breaking changes. No database migration required.
```

## 🔍 Review Checklist for Codex

Before submitting the PR, verify:

1. **Proto changes**:
   - [ ] New enum values added sequentially (4, 5, 6...)
   - [ ] Ran `make generate` to regenerate code
   - [ ] No build errors in backend

2. **Backend match_service.go**:
   - [ ] ALL new sports added to switch case (comma-separated)
   - [ ] Comment updated to mention all sports
   - [ ] No duplicate case statements

3. **Backend club_service.go**:
   - [ ] ALL new sports added to supportedClubSports map
   - [ ] Proper formatting (aligned with existing entries)

4. **Frontend sports.ts**:
   - [ ] Icons imported at top (verified they exist in Lucide)
   - [ ] SUPPORTED_SPORTS array includes all new sports
   - [ ] sportTranslationKey() has case for each sport
   - [ ] sportIconComponent() has case for each sport
   - [ ] Uses correct namespace: `sports.*` not `sport.*`

5. **Translations**:
   - [ ] Swedish translations added to sv.json
   - [ ] English translations added to en.json
   - [ ] Consistent formatting (comma after each entry except last)
   - [ ] Proper Swedish characters (ä, ö, å) used where needed

6. **Testing**:
   - [ ] `make lint` passes (0 issues)
   - [ ] `make be.build` succeeds
   - [ ] `make test` passes (all tests)
   - [ ] `cd frontend && npm run build` succeeds
   - [ ] Manual test: Created series for at least one sport
   - [ ] Manual test: Reported match successfully (no 501 error)
   - [ ] Manual test: Verified icon displays
   - [ ] Manual test: Verified translation shows correctly

## 📚 Reference Files

Key files to modify (in order):

1. `proto/klubbspel/v1/common.proto` - Add enum values
2. `backend/internal/service/match_service.go` - Add to switch case (line ~156)
3. `backend/internal/service/club_service.go` - Add to map (line ~35)
4. `frontend/src/lib/sports.ts` - Update all functions
5. `frontend/src/i18n/locales/sv.json` - Add Swedish translations (line ~371)
6. `frontend/src/i18n/locales/en.json` - Add English translations (line ~371)

## 🎓 Learning from Tennis Implementation

The tennis feature implementation taught us:

1. **Backend completeness is critical** - Always add new sports to match service switch
2. **Icon verification is essential** - Check Lucide library before importing
3. **Translation namespaces matter** - Use correct `sports.*` prefix
4. **Testing saves time** - Manual validation catches issues early
5. **Documentation helps** - Clear commit messages aid future debugging

These lessons are now encoded in this implementation guide to prevent repeating mistakes.

---

## 🚀 Ready to Start?

Use this guide as a complete checklist. Work through each section systematically, and verify each step before moving to the next. The goal is to add all four sports (badminton, squash, padel, pickleball) in a single, well-tested PR that maintains backward compatibility and code quality standards.

Good luck! 🎾🏸🎾
