// Match validation utilities for table tennis scoring

/**
 * Generates valid table tennis results for a given number of sets to play
 */
export function getValidTableTennisResults(setsToPlay: number): [number, number][] {
  const requiredWins = Math.ceil(setsToPlay / 2)
  const validResults: [number, number][] = []
  
  // Winner gets requiredWins, loser gets 0 to requiredWins-1
  for (let loserSets = 0; loserSets < requiredWins; loserSets++) {
    validResults.push([requiredWins, loserSets])
    validResults.push([loserSets, requiredWins])
  }
  
  return validResults
}

// Backward compatibility: best-of-5 results
export const VALID_TABLE_TENNIS_RESULTS = getValidTableTennisResults(5)

export type TableTennisScore = [number, number]

export interface ScoreValidationResult {
  isValid: boolean
  canAutoComplete: boolean
  suggestedScore?: TableTennisScore
  errorMessage?: string
}

export interface ScoreAutoCompletion {
  scoreA: number
  scoreB: number
  autoCompleted: boolean
}

/**
 * Validates if a table tennis score is valid according to sets-to-play rules
 */
export function validateTableTennisScore(scoreA: number, scoreB: number, setsToPlay: number = 5): boolean {
  const validResults = getValidTableTennisResults(setsToPlay)
  return validResults.some(([a, b]) => a === scoreA && b === scoreB)
}

/**
 * Checks if scores can be auto-completed and provides suggestions
 */
export function analyzeScoreInput(scoreAStr: string, scoreBStr: string): ScoreValidationResult {
  const scoreA = scoreAStr === '' ? null : parseInt(scoreAStr, 10)
  const scoreB = scoreBStr === '' ? null : parseInt(scoreBStr, 10)
  
  // Handle invalid number inputs
  if ((scoreAStr !== '' && scoreA !== null && (isNaN(scoreA) || scoreA < 0 || scoreA > 3)) ||
      (scoreBStr !== '' && scoreB !== null && (isNaN(scoreB) || scoreB < 0 || scoreB > 3))) {
    return {
      isValid: false,
      canAutoComplete: false,
      errorMessage: 'Scores must be between 0 and 3'
    }
  }
  
  // Both scores provided
  if (scoreA !== null && scoreB !== null) {
    const isValid = validateTableTennisScore(scoreA, scoreB)
    return {
      isValid,
      canAutoComplete: false,
      errorMessage: isValid ? undefined : 'Invalid table tennis result. Valid results: 3-0, 3-1, 3-2'
    }
  }
  
  // Auto-completion scenarios
  if (scoreA === 3 && (scoreB === null || scoreB === 0)) {
    return {
      isValid: false,
      canAutoComplete: true,
      suggestedScore: [3, 0]
    }
  }
  
  if (scoreB === 3 && (scoreA === null || scoreA === 0)) {
    return {
      isValid: false,
      canAutoComplete: true,
      suggestedScore: [0, 3]
    }
  }
  
  // Partial input, not ready for validation
  return {
    isValid: false,
    canAutoComplete: false
  }
}

/**
 * Auto-completes scores when one player has 3 and other is empty/0
 */
export function getAutoCompletedScores(scoreAStr: string, scoreBStr: string): ScoreAutoCompletion {
  const scoreA = scoreAStr === '' || isNaN(parseInt(scoreAStr, 10)) ? 0 : parseInt(scoreAStr, 10)
  const scoreB = scoreBStr === '' || isNaN(parseInt(scoreBStr, 10)) ? 0 : parseInt(scoreBStr, 10)
  
  // Auto-complete 3-0 scenarios
  if (scoreA === 3 && (scoreBStr === '' || scoreB === 0)) {
    return { scoreA: 3, scoreB: 0, autoCompleted: true }
  }
  if (scoreB === 3 && (scoreAStr === '' || scoreA === 0)) {
    return { scoreA: 0, scoreB: 3, autoCompleted: true }
  }
  
  return { scoreA, scoreB, autoCompleted: false }
}

/**
 * Validates that two players are different
 */
export function validateDifferentPlayers(playerAId: string, playerBId: string): boolean {
  return playerAId !== '' && playerBId !== '' && playerAId !== playerBId
}

/**
 * Validates match date is within series window
 */
export function validateMatchDateWindow(
  matchDate: string,
  matchTime: string,
  seriesStartDate?: string,
  seriesEndDate?: string
): { isValid: boolean; errorMessage?: string } {
  if (!seriesStartDate || !seriesEndDate) {
    return { isValid: true }
  }
  
  try {
    const matchDateTime = new Date(`${matchDate}T${matchTime}:00`)
    const seriesStart = new Date(seriesStartDate)
    const seriesEnd = new Date(seriesEndDate)
    
    // Set times to compare only dates (start of day for start, end of day for end)
    seriesStart.setHours(0, 0, 0, 0)
    seriesEnd.setHours(23, 59, 59, 999)
    
    if (matchDateTime < seriesStart || matchDateTime > seriesEnd) {
      return {
        isValid: false,
        errorMessage: `Match date must be within series period (${seriesStart.toLocaleDateString()} - ${seriesEnd.toLocaleDateString()})`
      }
    }
    
    return { isValid: true }
  } catch {
    return { isValid: false, errorMessage: 'Invalid date format' }
  }
}

/**
 * Comprehensive form validation for match reporting
 */
export interface MatchFormData {
  player_a_id: string
  player_b_id: string
  score_a: string
  score_b: string
  played_at_date: string
  played_at_time: string
}

export interface FormValidationResult {
  isValid: boolean
  errors: Record<string, string>
  canSubmit: boolean
  autoCompletionAvailable: boolean
}

export function validateMatchForm(
  formData: MatchFormData,
  seriesStartDate?: string,
  seriesEndDate?: string
): FormValidationResult {
  const errors: Record<string, string> = {}
  
  // Required fields
  if (!formData.player_a_id) errors.player_a_id = 'Player A is required'
  if (!formData.player_b_id) errors.player_b_id = 'Player B is required'
  if (!formData.played_at_date) errors.played_at_date = 'Date is required'
  if (!formData.played_at_time) errors.played_at_time = 'Time is required'
  
  // Player validation
  if (formData.player_a_id && formData.player_b_id) {
    if (!validateDifferentPlayers(formData.player_a_id, formData.player_b_id)) {
      errors.players = 'Players must be different'
    }
  }
  
  // Score validation
  const scoreValidation = analyzeScoreInput(formData.score_a, formData.score_b)
  if (scoreValidation.errorMessage) {
    errors.scores = scoreValidation.errorMessage
  }
  
  // Date validation
  if (formData.played_at_date && formData.played_at_time) {
    const dateValidation = validateMatchDateWindow(
      formData.played_at_date,
      formData.played_at_time,
      seriesStartDate,
      seriesEndDate
    )
    if (!dateValidation.isValid && dateValidation.errorMessage) {
      errors.date = dateValidation.errorMessage
    }
  }
  
  const hasRequiredFields = !!(formData.player_a_id && formData.player_b_id && 
                           formData.played_at_date && formData.played_at_time)
  const hasValidScores = scoreValidation.isValid || scoreValidation.canAutoComplete
  const hasNoErrors = Object.keys(errors).length === 0
  
  return {
    isValid: hasNoErrors && hasValidScores && hasRequiredFields,
    errors,
    canSubmit: hasRequiredFields && hasValidScores && hasNoErrors,
    autoCompletionAvailable: scoreValidation.canAutoComplete
  }
}