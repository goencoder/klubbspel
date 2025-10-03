# Ladder Logic Test Scenarios

## Your Original Question: Alice (#1) vs Morgan (#3), Alice Loses

**Initial State:**
1. Alice
2. Peter
3. Morgan

**Match:** Alice plays Morgan, Morgan wins

### Analysis with New Logic:

```go
// betterPositionPlayer = Alice (position 1 < 3)
// worsePositionPlayer = Morgan (position 3 > 1)
// winnerID = Morgan

// Case check: worsePositionPlayer.PlayerID == winnerID?
// YES! Morgan (worse position) won
```

**Code Path:** Case 1 - Worse position player wins (climbs ladder)

**Action:**
- Morgan takes Alice's position (#1)
- Everyone between positions 1 and 3 shifts down by 1

**Result:**
1. **Morgan** (winner, moved from #3 to #1)
2. **Alice** (was #1, shifted down to #2)
3. **Peter** (was #2, shifted down to #3)

✅ **This is CORRECT!** Morgan climbed the ladder by winning.

---

## Reverse Scenario: Morgan (#3) vs Alice (#1), Morgan Loses

**Initial State:**
1. Alice
2. Peter
3. Morgan

**Match:** Morgan plays Alice, Alice wins

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Morgan (position 3)
// winnerID = Alice

// Case check: worsePositionPlayer.PlayerID == winnerID?
// NO! Alice (better position) won
```

**Code Path:** Case 2 - Better position player wins (defends position)

### Classic Rules (LADDER_RULES_CLASSIC):

**Action:**
- Morgan keeps position (no penalty)
- Just update timestamps

**Result:**
1. Alice
2. Peter
3. Morgan

✅ No changes, loser keeps position

### Aggressive Rules (LADDER_RULES_AGGRESSIVE):

**Action:**
- Morgan drops one position (penalty)
- Player at position #4 (if exists) moves up to #3
- If no #4, Morgan stays at #3 (can't drop further)

**Result (if 4 players exist):**
1. Alice
2. Peter
3. Dave (was #4, moved up)
4. Morgan (dropped due to penalty)

**Result (if only 3 players):**
1. Alice
2. Peter
3. Morgan (can't drop, already at bottom)

✅ Penalty applied correctly

---

## Test Case 3: Adjacent Positions - Peter (#2) vs Alice (#1), Peter Wins

**Initial State:**
1. Alice
2. Peter
3. Morgan

**Match:** Peter plays Alice, Peter wins

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Peter (position 2)
// winnerID = Peter

// Case: worsePositionPlayer.PlayerID == winnerID? YES
```

**Code Path:** Case 1 - Worse position player wins

**Action:**
- Peter takes Alice's position (#1)
- Everyone between 1 and 2 shifts down... but there's no one between!

**Result:**
1. **Peter** (winner, moved from #2 to #1)
2. **Alice** (was #1, shifted to #2)
3. Morgan (unchanged)

✅ Simple swap, works correctly

---

## Test Case 4: Adjacent Positions - Alice (#1) vs Peter (#2), Alice Loses

**Initial State:**
1. Alice
2. Peter
3. Morgan
4. Dave

**Match:** Alice plays Peter, Peter wins

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Peter (position 2)
// winnerID = Peter

// Case: worsePositionPlayer.PlayerID == winnerID? YES
```

**Code Path:** Case 1 - Worse position player wins

**Action:**
- Peter takes position #1
- Alice shifts to #2

**Result:**
1. **Peter** (winner, climbed from #2)
2. **Alice** (was #1, dropped to #2)
3. Morgan (unchanged)
4. Dave (unchanged)

✅ Adjacent swap works

---

## Test Case 5: Downward Challenge Loss - Alice (#1) vs Dave (#4), Alice Loses

**Initial State:**
1. Alice
2. Peter
3. Morgan
4. Dave

**Match:** Alice plays Dave, Dave wins

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Dave (position 4)
// winnerID = Dave

// Case: worsePositionPlayer.PlayerID == winnerID? YES
```

**Code Path:** Case 1 - Worse position player wins

**Action:**
- Dave takes position #1
- Everyone between positions 1 and 4 shifts down

**Result:**
1. **Dave** (winner, climbed from #4 to #1)
2. **Alice** (was #1, shifted to #2)
3. **Peter** (was #2, shifted to #3)
4. **Morgan** (was #3, shifted to #4)

✅ Big climb works correctly!

---

## Test Case 6: Downward Challenge Win - Alice (#1) vs Dave (#4), Alice Wins

**Initial State:**
1. Alice
2. Peter
3. Morgan
4. Dave

**Match:** Alice plays Dave, Alice wins

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Dave (position 4)
// winnerID = Alice

// Case: worsePositionPlayer.PlayerID == winnerID? NO
```

**Code Path:** Case 2 - Better position player wins

### Classic Rules:

**Action:**
- Dave keeps position (no penalty)

**Result:**
1. Alice (unchanged)
2. Peter (unchanged)
3. Morgan (unchanged)
4. Dave (unchanged)

✅ No penalty, everyone stays

### Aggressive Rules:

**Action:**
- Dave drops one position (to #5)
- Player at #5 (if exists) moves to #4

**Result (if 5 players):**
1. Alice
2. Peter
3. Morgan
4. Eve (was #5, moved up)
5. Dave (dropped due to penalty)

**Result (if only 4 players):**
1. Alice
2. Peter
3. Morgan
4. Dave (can't drop, already at bottom)

✅ Penalty logic correct

---

## Edge Case 7: Bottom Player Loses with Aggressive Rules

**Initial State:**
1. Alice
2. Peter
3. Morgan

**Match:** Morgan (#3) plays Alice (#1), Morgan loses
**Format:** Aggressive ladder

### Analysis:

```go
// betterPositionPlayer = Alice (position 1)
// worsePositionPlayer = Morgan (position 3)
// winnerID = Alice

// Case: worsePositionPlayer.PlayerID == winnerID? NO
// Apply aggressive penalty: newPosition = 3 + 1 = 4
// FindBySeriesAndPosition(4) → mongo.ErrNoDocuments
```

**Code Path:** Aggressive penalty, but no position #4 exists

**Action:**
- Try to drop Morgan to #4
- No player at #4 found
- Fallback: Just touch timestamp, no position change

**Result:**
1. Alice
2. Peter
3. Morgan (stays, can't drop further)

✅ Edge case handled gracefully!

---

## Summary of Logic Correctness

| Scenario | Better Pos | Worse Pos | Winner | Expected Result | ✅ |
|----------|-----------|-----------|--------|-----------------|---|
| Alice #1 vs Morgan #3, Morgan wins | Alice | Morgan | Morgan | Morgan → #1, shift down | ✅ |
| Alice #1 vs Morgan #3, Alice wins (Classic) | Alice | Morgan | Alice | No change (no penalty) | ✅ |
| Alice #1 vs Morgan #3, Alice wins (Aggressive) | Alice | Morgan | Alice | Morgan drops to #4 (penalty) | ✅ |
| Peter #2 vs Alice #1, Peter wins | Alice | Peter | Peter | Peter → #1, Alice → #2 | ✅ |
| Alice #1 vs Dave #4, Dave wins | Alice | Dave | Dave | Dave → #1, all shift down | ✅ |
| Alice #1 vs Dave #4, Alice wins (Classic) | Alice | Dave | Alice | No change | ✅ |
| Alice #1 vs Dave #4, Alice wins (Aggressive) | Alice | Dave | Alice | Dave drops to #5 (penalty) | ✅ |
| Morgan #3 (bottom) loses (Aggressive) | Alice | Morgan | Alice | Morgan stays #3 (can't drop) | ✅ |

## Conclusion

**The new logic is CORRECT! ✅**

Key improvements:
1. **Neutral terminology**: betterPosition/worsePosition instead of challenger/challenged
2. **Bidirectional support**: Works regardless of who initiated the match
3. **Clear case distinction**: Winner with worse position vs winner with better position
4. **Proper penalty application**: Only affects loser in aggressive mode
5. **Edge case handling**: Bottom position can't drop further

The system now properly handles:
- ✅ Upward challenges (worse → better)
- ✅ Downward challenges (better → worse) ← **This was broken before!**
- ✅ Adjacent positions
- ✅ Long-distance positions
- ✅ Classic rules (no penalty)
- ✅ Aggressive rules (penalty)
- ✅ Bottom position edge case
