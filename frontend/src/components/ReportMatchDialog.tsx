import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { PlayerSelector, type PlayerSelectorHandle } from '@/components/PlayerSelector'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
import type { Player, Series } from '@/types/api'
import { toast } from 'sonner'

interface MatchFormState {
  player_a_id: string
  player_b_id: string
  score_a: string
  score_b: string
  played_at_date: string
  played_at_time: string
}

interface ReportMatchDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  seriesId: string
  clubId?: string
  seriesStartDate?: string
  seriesEndDate?: string
  series?: Series  // Optional series object for advanced features
  onMatchReported: (match: { matchId: string; playedAt: string }) => void
}

export function ReportMatchDialog({
  open,
  onOpenChange,
  seriesId,
  clubId,
  seriesStartDate,
  seriesEndDate,
  series,
  onMatchReported
}: ReportMatchDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const playerASelectorRef = useRef<PlayerSelectorHandle | null>(null)

  // Helper function to validate table tennis scores
  const validateTableTennisScore = (scoreA: number, scoreB: number) => {
    // Get sets to play from series (default to 5 if not available)
    const setsToPlay = series?.setsToPlay || 5
    
    // Generate valid results based on setsToPlay
    const validResults: [number, number][] = []
    const requiredWins = Math.ceil(setsToPlay / 2)
    
    // Winner gets requiredWins, loser gets 0 to requiredWins-1
    for (let loserSets = 0; loserSets < requiredWins; loserSets++) {
      validResults.push([requiredWins, loserSets])
      validResults.push([loserSets, requiredWins])
    }
    
    return validResults.some(([a, b]) => a === scoreA && b === scoreB)
  }

  // Helper function to auto-complete score when one player has required wins and other is empty/0
  const getAutoCompletedScores = (scoreAStr: string, scoreBStr: string) => {
    const setsToPlay = series?.setsToPlay || 5
    const requiredWins = Math.ceil(setsToPlay / 2)
    
    // Parse scores, treating empty strings and invalid numbers as 0
    const scoreA = scoreAStr === '' || isNaN(parseInt(scoreAStr, 10)) ? 0 : parseInt(scoreAStr, 10)
    const scoreB = scoreBStr === '' || isNaN(parseInt(scoreBStr, 10)) ? 0 : parseInt(scoreBStr, 10)
    
    // If one player has required wins and the other is 0 (or empty), auto-complete to a shutout
    if (scoreA === requiredWins && (scoreBStr === '' || scoreB === 0)) {
      return { scoreA: requiredWins, scoreB: 0, autoCompleted: true }
    }
    if (scoreB === requiredWins && (scoreAStr === '' || scoreA === 0)) {
      return { scoreA: 0, scoreB: requiredWins, autoCompleted: true }
    }
    
    return { scoreA, scoreB, autoCompleted: false }
  }
  const createEmptyFormState = (): MatchFormState => ({
    player_a_id: '',
    player_b_id: '',
    score_a: '',
    score_b: '',
    played_at_date: '',
    played_at_time: ''
  })

  const [formData, setFormData] = useState<MatchFormState>(() => createEmptyFormState())

  // Multi-match session state
  const [keepDialogOpen, setKeepDialogOpen] = useState(false)
  const [sessionCount, setSessionCount] = useState(0)

  const formatDatePart = (date: Date) => date.toISOString().split('T')[0]
  const formatTimePart = (date: Date) => date.toTimeString().slice(0, 5)
  const addMinutes = (date: Date, minutes: number) => new Date(date.getTime() + minutes * 60 * 1000)

  // Reset form when dialog opens/closes and prime initial date/time suggestion
  useEffect(() => {
    if (!open) {
      setFormData(createEmptyFormState())
      return
    }

    const now = new Date()
    setFormData(prev => ({
      ...prev,
      played_at_date: formatDatePart(now),
      played_at_time: formatTimePart(now)
    }))
  }, [open])

  useEffect(() => {
    if (!open) {
      return
    }

    const focusTimer = setTimeout(() => {
      playerASelectorRef.current?.focus()
    }, 0)

    return () => clearTimeout(focusTimer)
  }, [open])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    // Only require players, date and time - scores will be auto-completed if needed
    if (!formData.player_a_id || !formData.player_b_id || !formData.played_at_date || !formData.played_at_time) {
      toast.error(t('form.validation.error'))
      return
    }

    if (formData.player_a_id === formData.player_b_id) {
      toast.error('A player cannot play against themselves')
      return
    }

    // Auto-complete scores if one player has 3 and other is empty/0
    const { scoreA, scoreB, autoCompleted } = getAutoCompletedScores(formData.score_a, formData.score_b)

    // Validate table tennis scores
    if (!validateTableTennisScore(scoreA, scoreB)) {
      toast.error('Invalid table tennis result. Valid results are: 3-0, 3-1, or 3-2')
      return
    }

    // If we auto-completed, update the form to show the completed score
    if (autoCompleted) {
      setFormData(prev => ({
        ...prev,
        score_a: scoreA.toString(),
        score_b: scoreB.toString()
      }))
    }

    // Validate that match date is within series time window (inclusive)
    if (seriesStartDate && seriesEndDate) {
      const matchDateTime = new Date(`${formData.played_at_date}T${formData.played_at_time}:00`)
      const seriesStart = new Date(seriesStartDate)
      const seriesEnd = new Date(seriesEndDate)
      
      // Set times to compare only dates (start of day for start, end of day for end)
      seriesStart.setHours(0, 0, 0, 0)
      seriesEnd.setHours(23, 59, 59, 999)
      
      if (matchDateTime < seriesStart || matchDateTime > seriesEnd) {
        toast.error(
          `${t('matches.validation.seriesWindow')} (${seriesStart.toLocaleDateString()} - ${seriesEnd.toLocaleDateString()})`
        )
        return
      }
    }

    try {
      setLoading(true)

      // Use V2 API for table tennis with scoring profile support
      const reportRequestV2 = {
        seriesId: seriesId,
        participantA: { playerId: formData.player_a_id },
        participantB: { playerId: formData.player_b_id },
        result: {
          tableTennis: {
            setsA: scoreA,
            setsB: scoreB
          }
        },
        playedAt: new Date(`${formData.played_at_date}T${formData.played_at_time}:00`).toISOString()
      }

      const response = await apiClient.reportMatchV2(reportRequestV2)
      
      // Update session tracking
      setSessionCount(prev => prev + 1)
      
      toast.success(t('matches.reported'))
      onMatchReported({
        matchId: response.matchId,
        playedAt: reportRequestV2.playedAt
      })

      if (keepDialogOpen) {
        // Prepare for next match - advance time by 5 minutes
        const nextSuggestedDate = addMinutes(new Date(reportRequestV2.playedAt), 5)
        setFormData({
          player_a_id: '',
          player_b_id: '',
          score_a: '',
          score_b: '',
          played_at_date: formatDatePart(nextSuggestedDate),
          played_at_time: formatTimePart(nextSuggestedDate)
        })

        setTimeout(() => {
          playerASelectorRef.current?.focus()
        }, 0)
      } else {
        // Close dialog and reset session
        onOpenChange(false)
        setSessionCount(0)
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : ''
      toast.error(message || t('error.generic'))
    } finally {
      setLoading(false)
    }
  }

  const handlePlayerASelected = (player: Player) => {
    setFormData(prev => ({ ...prev, player_a_id: player.id }))
  }

  const handlePlayerBSelected = (player: Player) => {
    setFormData(prev => ({ ...prev, player_b_id: player.id }))
  }

  const isFormValid = () => {
    // Basic required fields
    if (!formData.player_a_id || !formData.player_b_id || !formData.played_at_date || !formData.played_at_time) {
      return false
    }
    
    // Players must be different
    if (formData.player_a_id === formData.player_b_id) {
      return false
    }
    
    // Check if scores are valid (including auto-completion)
    const { scoreA, scoreB } = getAutoCompletedScores(formData.score_a, formData.score_b)
    return validateTableTennisScore(scoreA, scoreB)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {t('matches.report')}
            {sessionCount > 0 && (
              <Badge variant="secondary" className="text-xs">
                Session: {sessionCount}
              </Badge>
            )}
          </DialogTitle>
          <DialogDescription>
            Report the result of a table tennis match
            {keepDialogOpen && (
              <div className="text-xs text-muted-foreground mt-1">
                Multi-match mode active - dialog will stay open after reporting
              </div>
            )}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid gap-6">
            {/* Player Selection */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>
                  {series?.format === 'SERIES_FORMAT_LADDER' ? t('matches.challenger') : t('matches.player_a')} *
                </Label>
                <PlayerSelector
                  ref={playerASelectorRef}
                  value={formData.player_a_id}
                  onPlayerSelected={handlePlayerASelected}
                  clubId={clubId}
                  excludePlayerId={formData.player_b_id}
                />
              </div>
              <div className="space-y-2">
                <Label>
                  {series?.format === 'SERIES_FORMAT_LADDER' ? t('matches.defender') : t('matches.player_b')} *
                </Label>
                <PlayerSelector
                  value={formData.player_b_id}
                  onPlayerSelected={handlePlayerBSelected}
                  clubId={clubId}
                  excludePlayerId={formData.player_a_id}
                />
              </div>
            </div>

            {/* Score inputs */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="score_a">{t('matches.score_a')}</Label>
                <Input
                  id="score_a"
                  type="number"
                  min="0"
                  max={Math.ceil((series?.setsToPlay || 5) / 2)}
                  value={formData.score_a}
                  onChange={(e) => setFormData(prev => ({ ...prev, score_a: e.target.value }))}
                  placeholder="0"
                />
                <p className="text-xs text-muted-foreground">
                  {series?.setsToPlay === 3 
                    ? 'Valid results: 2-0, 2-1' 
                    : 'Valid results: 3-0, 3-1, 3-2'
                  }
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="score_b">{t('matches.score_b')}</Label>
                <Input
                  id="score_b"
                  type="number"
                  min="0"
                  max={Math.ceil((series?.setsToPlay || 5) / 2)}
                  value={formData.score_b}
                  onChange={(e) => setFormData(prev => ({ ...prev, score_b: e.target.value }))}
                  placeholder="0"
                />
              </div>
            </div>

            {/* Date and Time */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="played_at_date">{t('matches.played_at_date')} *</Label>
                <Input
                  id="played_at_date"
                  type="date"
                  value={formData.played_at_date}
                  onChange={(e) => setFormData(prev => ({ ...prev, played_at_date: e.target.value }))}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="played_at_time">{t('matches.played_at_time')} *</Label>
                <Input
                  id="played_at_time"
                  type="time"
                  value={formData.played_at_time}
                  onChange={(e) => setFormData(prev => ({ ...prev, played_at_time: e.target.value }))}
                  required
                />
              </div>
            </div>

            {/* Multi-match toggle */}
            <div className="flex items-center space-x-2 p-3 bg-muted/50 rounded-lg">
              <Switch
                id="keepOpen"
                checked={keepDialogOpen}
                onCheckedChange={setKeepDialogOpen}
              />
              <Label htmlFor="keepOpen" className="text-sm">
                Keep dialog open for multiple matches
                <div className="text-xs text-muted-foreground">
                  Time will advance by 5 minutes between matches
                </div>
              </Label>
            </div>

            {/* Validation hints */}
            <div className="text-sm text-muted-foreground bg-muted p-3 rounded-md">
              <ul className="space-y-1">
                <li>• Best of {series?.setsToPlay || 5} format (winner must reach {Math.ceil((series?.setsToPlay || 5) / 2)} sets)</li>
                <li>• Scores must be between 0 and {Math.ceil((series?.setsToPlay || 5) / 2)}</li>
                <li>• No ties allowed</li>
                <li>• Players must be different</li>
              </ul>
            </div>
          </div>

          <DialogFooter>
            <Button 
              type="button" 
              variant="outline" 
              onClick={() => {
                onOpenChange(false)
                setSessionCount(0)
                setKeepDialogOpen(false)
              }}
            >
              {keepDialogOpen ? 'Close Session' : t('common.done')}
            </Button>
            <Button type="submit" disabled={!isFormValid() || loading}>
              {loading ? (
                <LoadingSpinner size="sm" />
              ) : (
                <>
                  {t('matches.report')}
                  {keepDialogOpen && sessionCount > 0 && (
                    <Badge variant="secondary" className="ml-2 text-xs">
                      #{sessionCount + 1}
                    </Badge>
                  )}
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}