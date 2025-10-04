# Leaderboard System Analysis

**Date:** October 4, 2025  
**Status:** Under Development - Recalculation Logic Has Bug

---

## Executive Summary

The system has been redesigned to use a **recalculation-based architecture** where leaderboards are rebuilt from scratch after every match operation. However, the implementation has a critical bug causing the backend to crash with a 500 error during the recalculation phase.

### Key Finding
- ✅ Matches are successfully created and stored
- ❌ `RecalculateStandings` causes a 500 error (likely panic)

---

## Architecture Overview

### Data Flow

```
Match Operation (Create/Update/Delete)
    ↓
1. Store/Modify Match in MongoDB
    ↓
2. Call RecalculateStandings(seriesID)
    ↓
3. Fetch ALL matches chronologically
    ↓
6. Store calculated standings in leader board collection
```

```
Leaderboard Operation (Get)
    ↓
1. read from leader board
```


---

## Series Types & Behavior

### 1. Open Series (SERIES_FORMAT_OPEN_PLAY)

**Format Value:** `1`  
**Ladder Rules:** N/A

#### Current Implementation

**Match Reporting:**
```go
// In ReportMatch/UpdateMatch/DeleteMatch:
match = CreateMatch(...)
RecalculateStandings(seriesID) // Currently fails with 500 error
```

**Standings Calculation:**
```go
// In RecalculateStandings:
if format == SERIES_FORMAT_LADDER_CLASSIC {
    recalculateLadderStandings(aggressive:false)
} else if format == SERIES_FORMAT_LADDER_AGGRESSIVE {
    recalculateLadderStandings(aggressive:true)
} else {
    recalculateEloStandings(...) // For open series
}
```

**Elo Standings Method:**
```go
func recalculateEloStandings(ctx, seriesID, matches, now) error {
    // should update leader board, based on ELO calculation
    return nil
}
```

**Leaderboard Display:**
```go
// In GetLeaderboard:
// just read from the pre-calculated leaderboard
```

