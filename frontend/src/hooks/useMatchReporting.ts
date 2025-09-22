import { useState, useCallback, useMemo, useRef } from 'react'
import { toast } from 'sonner'
import { apiClient } from '@/services/api'
import type { Player, ReportMatchRequest } from '@/types/api'
import { 
  validateMatchForm, 
  getAutoCompletedScores,
  type MatchFormData,
  type FormValidationResult 
} from '@/utils/matchValidation'

export interface PlayerPair {
  playerAId: string
  playerBId: string
  playerAName: string
  playerBName: string
  frequency: number
  lastUsed: Date
}

export interface ReportedMatch {
  matchId: string
  playedAt: string
  playerAId: string
  playerBId: string
  scoreA: number
  scoreB: number
}

export interface UseMatchReportingOptions {
  seriesId: string
  clubId?: string
  seriesStartDate?: string
  seriesEndDate?: string
  onMatchReported: (match: ReportedMatch) => void
  bulkMode?: boolean
}

export interface UseMatchReportingReturn {
  // Form state
  formData: MatchFormData
  setFormData: React.Dispatch<React.SetStateAction<MatchFormData>>
  
  // Validation
  validation: FormValidationResult
  
  // Actions
  reportMatch: (e?: React.FormEvent) => Promise<void>
  resetForm: () => void
  setPlayerA: (player: Player) => void
  setPlayerB: (player: Player) => void
  
  // Bulk entry state
  matchesReported: number
  recentPairs: PlayerPair[]
  selectPlayerPair: (pair: PlayerPair) => void
  
  // Loading state
  isLoading: boolean
  
  // Utilities
  formatDatePart: (date: Date) => string
  formatTimePart: (date: Date) => string
  addMinutes: (date: Date, minutes: number) => Date
}

const createEmptyFormState = (): MatchFormData => ({
  player_a_id: '',
  player_b_id: '',
  score_a: '',
  score_b: '',
  played_at_date: '',
  played_at_time: ''
})

export function useMatchReporting(options: UseMatchReportingOptions): UseMatchReportingReturn {
  const { 
    seriesId, 
    seriesStartDate, 
    seriesEndDate, 
    onMatchReported, 
    bulkMode = false 
  } = options
  
  // Form state
  const [formData, setFormData] = useState<MatchFormData>(createEmptyFormState)
  const [isLoading, setIsLoading] = useState(false)
  
  // Bulk entry state
  const [matchesReported, setMatchesReported] = useState(0)
  const [recentPairs, setRecentPairs] = useState<PlayerPair[]>([])
  const [sessionMatches, setSessionMatches] = useState<ReportedMatch[]>([])
  
  // Player name cache for recent pairs
  const playerNamesRef = useRef<Map<string, string>>(new Map())
  
  // Utility functions
  const formatDatePart = useCallback((date: Date) => date.toISOString().split('T')[0], [])
  const formatTimePart = useCallback((date: Date) => date.toTimeString().slice(0, 5), [])
  const addMinutes = useCallback((date: Date, minutes: number) => 
    new Date(date.getTime() + minutes * 60 * 1000), [])
  
  // Validation
  const validation = useMemo(() => 
    validateMatchForm(formData, seriesStartDate, seriesEndDate),
    [formData, seriesStartDate, seriesEndDate]
  )
  
  // Player selection handlers
  const setPlayerA = useCallback((player: Player) => {
    playerNamesRef.current.set(player.id, player.displayName)
    setFormData(prev => ({ ...prev, player_a_id: player.id }))
  }, [])
  
  const setPlayerB = useCallback((player: Player) => {
    playerNamesRef.current.set(player.id, player.displayName)
    setFormData(prev => ({ ...prev, player_b_id: player.id }))
  }, [])
  
  // Update recent pairs
  const updateRecentPairs = useCallback((playerAId: string, playerBId: string) => {
    const playerAName = playerNamesRef.current.get(playerAId) || playerAId
    const playerBName = playerNamesRef.current.get(playerBId) || playerBId
    
    setRecentPairs(prev => {
      const existingIndex = prev.findIndex(pair => 
        (pair.playerAId === playerAId && pair.playerBId === playerBId) ||
        (pair.playerAId === playerBId && pair.playerBId === playerAId)
      )
      
      const newPair: PlayerPair = {
        playerAId,
        playerBId,
        playerAName,
        playerBName,
        frequency: existingIndex >= 0 ? prev[existingIndex].frequency + 1 : 1,
        lastUsed: new Date()
      }
      
      if (existingIndex >= 0) {
        // Update existing pair
        const updated = [...prev]
        updated[existingIndex] = newPair
        return updated.sort((a, b) => b.frequency - a.frequency).slice(0, 5) // Keep top 5
      } else {
        // Add new pair
        return [newPair, ...prev].sort((a, b) => b.frequency - a.frequency).slice(0, 5)
      }
    })
  }, [])
  
  // Select a recent player pair
  const selectPlayerPair = useCallback((pair: PlayerPair) => {
    setFormData(prev => ({
      ...prev,
      player_a_id: pair.playerAId,
      player_b_id: pair.playerBId
    }))
  }, [])
  
  // Reset form with smart defaults
  const resetForm = useCallback((advanceTime = false) => {
    const now = new Date()
    const baseTime = advanceTime && sessionMatches.length > 0 
      ? addMinutes(new Date(sessionMatches[sessionMatches.length - 1].playedAt), 5)
      : now
      
    setFormData({
      player_a_id: '',
      player_b_id: '',
      score_a: '',
      score_b: '',
      played_at_date: formatDatePart(baseTime),
      played_at_time: formatTimePart(baseTime)
    })
  }, [sessionMatches, addMinutes, formatDatePart, formatTimePart])
  
  // Report match
  const reportMatch = useCallback(async (e?: React.FormEvent) => {
    if (e) {
      e.preventDefault()
    }
    
    if (!validation.canSubmit) {
      // Show specific error messages
      const firstError = Object.values(validation.errors)[0]
      if (firstError) {
        toast.error(firstError)
      }
      return
    }
    
    setIsLoading(true)
    
    try {
      // Auto-complete scores if needed
      const { scoreA, scoreB, autoCompleted } = getAutoCompletedScores(
        formData.score_a, 
        formData.score_b
      )
      
      // Update form to show completed scores
      if (autoCompleted) {
        setFormData(prev => ({
          ...prev,
          score_a: scoreA.toString(),
          score_b: scoreB.toString()
        }))
      }
      
      const playedAt = new Date(`${formData.played_at_date}T${formData.played_at_time}:00`).toISOString()

      const reportRequest: ReportMatchRequest = {
        metadata: {
          seriesId,
          playedAt
        },
        participants: [
          { playerId: formData.player_a_id },
          { playerId: formData.player_b_id }
        ],
        result: {
          tableTennis: {
            bestOf: 5,
            gamesWon: [scoreA, scoreB]
          }
        }
      }

      const response = await apiClient.reportMatch(reportRequest)

      const reportedMatch: ReportedMatch = {
        matchId: response.matchId,
        playedAt: reportRequest.metadata.playedAt,
        playerAId: formData.player_a_id,
        playerBId: formData.player_b_id,
        scoreA,
        scoreB
      }
      
      // Update session state
      setSessionMatches(prev => [...prev, reportedMatch])
      setMatchesReported(prev => prev + 1)
      updateRecentPairs(formData.player_a_id, formData.player_b_id)
      
      // Notify parent
      onMatchReported(reportedMatch)
      
      // Show success message
      const playerAName = playerNamesRef.current.get(formData.player_a_id) || 'Player A'
      const playerBName = playerNamesRef.current.get(formData.player_b_id) || 'Player B'
      toast.success(`Match reported: ${playerAName} ${scoreA}-${scoreB} ${playerBName}`)
      
      // Reset form for next match in bulk mode
      if (bulkMode) {
        resetForm(true) // Advance time by 5 minutes
      }
      
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to report match'
      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }, [
    validation,
    formData,
    seriesId,
    bulkMode,
    onMatchReported,
    updateRecentPairs,
    resetForm
  ])
  
  return {
    // Form state
    formData,
    setFormData,
    
    // Validation
    validation,
    
    // Actions
    reportMatch,
    resetForm,
    setPlayerA,
    setPlayerB,
    
    // Bulk entry state
    matchesReported,
    recentPairs,
    selectPlayerPair,
    
    // Loading state
    isLoading,
    
    // Utilities
    formatDatePart,
    formatTimePart,
    addMinutes
  }
}