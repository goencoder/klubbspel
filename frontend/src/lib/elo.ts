export const DEFAULT_RATING = 1000
export const K_FACTOR = 32

export interface EloResult {
  winnerNewRating: number
  loserNewRating: number
  winnerDelta: number
  loserDelta: number
}

/**
 * Calculate new ELO ratings for two players after a match
 * @param winnerRating Current rating of the winner
 * @param loserRating Current rating of the loser
 * @param kFactor K-factor for rating adjustment (default 32)
 * @returns New ratings and deltas for both players
 */
export function calculateEloRatings(
  winnerRating: number,
  loserRating: number,
  kFactor: number = K_FACTOR
): EloResult {
  // Expected score for winner (probability of winning)
  const expectedWinner = 1 / (1 + Math.pow(10, (loserRating - winnerRating) / 400))
  
  // Expected score for loser
  const expectedLoser = 1 / (1 + Math.pow(10, (winnerRating - loserRating) / 400))
  
  // Actual scores (1 for win, 0 for loss)
  const actualWinner = 1
  const actualLoser = 0
  
  // Calculate rating changes
  const winnerDelta = Math.round(kFactor * (actualWinner - expectedWinner))
  const loserDelta = Math.round(kFactor * (actualLoser - expectedLoser))
  
  // Calculate new ratings
  const winnerNewRating = winnerRating + winnerDelta
  const loserNewRating = loserRating + loserDelta
  
  return {
    winnerNewRating,
    loserNewRating,
    winnerDelta,
    loserDelta
  }
}

/**
 * Normalize a player name for duplicate detection
 * - Convert to lowercase
 * - Remove diacritics/accents
 * - Trim whitespace
 * - Remove extra spaces
 */
export function normalizePlayerName(name: string): string {
  return name
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '') // Remove diacritics
    .trim()
    .replace(/\s+/g, ' ') // Collapse multiple spaces
}

/**
 * Calculate Levenshtein distance between two strings
 */
export function levenshteinDistance(a: string, b: string): number {
  if (a.length === 0) return b.length
  if (b.length === 0) return a.length
  
  const matrix = Array(a.length + 1).fill(null).map(() => Array(b.length + 1).fill(null))
  
  for (let i = 0; i <= a.length; i++) matrix[i][0] = i
  for (let j = 0; j <= b.length; j++) matrix[0][j] = j
  
  for (let i = 1; i <= a.length; i++) {
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1
      matrix[i][j] = Math.min(
        matrix[i - 1][j] + 1,     // deletion
        matrix[i][j - 1] + 1,     // insertion
        matrix[i - 1][j - 1] + cost // substitution
      )
    }
  }
  
  return matrix[a.length][b.length]
}

/**
 * Find similar players based on normalized name comparison
 */
export function findSimilarPlayers(inputName: string, existingPlayers: { displayName: string; normalizedKey: string }[]): typeof existingPlayers {
  const inputNormalized = normalizePlayerName(inputName)
  
  return existingPlayers.filter(player => {
    // Exact match on normalized key
    if (player.normalizedKey === inputNormalized) {
      return true
    }
    
    // Levenshtein distance ≤ 2
    const distance = levenshteinDistance(inputNormalized, player.normalizedKey)
    return distance <= 2
  })
}