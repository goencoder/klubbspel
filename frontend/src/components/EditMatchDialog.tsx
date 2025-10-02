import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { apiClient } from '@/services/api'
import type { MatchView, UpdateMatchRequest } from '@/types/api'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

interface EditMatchDialogProps {
  match: MatchView | null
  isOpen: boolean
  onClose: () => void
  onMatchUpdated: (updatedMatch: MatchView) => void
  seriesStartDate?: string
  seriesEndDate?: string
}

export function EditMatchDialog({
  match,
  isOpen,
  onClose,
  onMatchUpdated,
  seriesStartDate,
  seriesEndDate,
}: EditMatchDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [scoreA, setScoreA] = useState<number>(0)
  const [scoreB, setScoreB] = useState<number>(0)
  const [playedAtDate, setPlayedAtDate] = useState<string>('')
  const [playedAtTime, setPlayedAtTime] = useState<string>('')

  // Helper function to validate table tennis scores
  const validateTableTennisScore = (scoreA: number, scoreB: number) => {
    // Valid table tennis match results: 3-0, 3-1, 3-2 (or reversed)
    const validResults = [
      [3, 0], [0, 3], [3, 1], [1, 3], [3, 2], [2, 3]
    ]
    
    return validResults.some(([a, b]) => a === scoreA && b === scoreB)
  }

  useEffect(() => {
    if (match) {
      setScoreA(match.scoreA)
      setScoreB(match.scoreB)
      // Convert ISO date to input date format (YYYY-MM-DD)
      const matchDate = new Date(match.playedAt)
      setPlayedAtDate(matchDate.toISOString().split('T')[0])
      // Convert to HH:MM format
      setPlayedAtTime(matchDate.toTimeString().slice(0, 5))
    }
  }, [match])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!match) return

    // Validate table tennis scores
    if (!validateTableTennisScore(scoreA, scoreB)) {
      toast.error('Invalid table tennis result. Valid results are: 3-0, 3-1, or 3-2')
      return
    }

    // Validate match date is within series window if being updated (inclusive)
    if (seriesStartDate && seriesEndDate && playedAtDate !== new Date(match.playedAt).toISOString().split('T')[0]) {
      const matchDate = new Date(playedAtDate)
      const startDate = new Date(seriesStartDate)
      const endDate = new Date(seriesEndDate)
      
      // Set times to compare only dates (start of day for start, end of day for end)
      startDate.setHours(0, 0, 0, 0)
      endDate.setHours(23, 59, 59, 999)
      matchDate.setHours(12, 0, 0, 0) // Set to middle of day for comparison
      
      if (matchDate < startDate || matchDate > endDate) {
        toast.error(
          `${t('matches.validation.seriesWindow')} (${startDate.toLocaleDateString()} - ${endDate.toLocaleDateString()})`
        )
        return
      }
    }

    try {
      setLoading(true)
      
      // Combine date and time for comparison and API call
      const originalDate = new Date(match.playedAt)
      const newDateTime = `${playedAtDate}T${playedAtTime}:00`
      const hasDateTimeChanged = playedAtDate !== originalDate.toISOString().split('T')[0] || 
                                 playedAtTime !== originalDate.toTimeString().slice(0, 5)
      
      const updateRequest: UpdateMatchRequest = {
        matchId: match.id,
        scoreA: scoreA !== match.scoreA ? scoreA : undefined,
        scoreB: scoreB !== match.scoreB ? scoreB : undefined,
        playedAt: hasDateTimeChanged ? new Date(newDateTime).toISOString() : undefined,
      }

      const response = await apiClient.updateMatch(updateRequest)
      onMatchUpdated(response.match)
      toast.success(t('matches.updated'))
      onClose()
    } catch (error: unknown) {
      toast.error((error as Error).message || t('error.generic'))
    } finally {
      setLoading(false)
    }
  }

  if (!match) return null

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{t('matches.editTitle')}</DialogTitle>
          <DialogDescription>
            {match.playerAName} vs {match.playerBName}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="scoreA" className="text-right">
                {match.playerAName}
              </Label>
              <Input
                id="scoreA"
                type="number"
                min="0"
                max="3"
                value={scoreA}
                onChange={(e) => setScoreA(parseInt(e.target.value, 10) || 0)}
                className="col-span-3"
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="scoreB" className="text-right">
                {match.playerBName}
              </Label>
              <Input
                id="scoreB"
                type="number"
                min="0"
                max="3"
                value={scoreB}
                onChange={(e) => setScoreB(parseInt(e.target.value, 10) || 0)}
                className="col-span-3"
              />
            </div>
            <div className="text-xs text-muted-foreground text-center">
              Valid table tennis results: 3-0, 3-1, 3-2
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="playedAtDate" className="text-right">
                {t('matches.played_at_date')}
              </Label>
              <Input
                id="playedAtDate"
                type="date"
                value={playedAtDate}
                onChange={(e) => setPlayedAtDate(e.target.value)}
                className="col-span-3"
                required
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="playedAtTime" className="text-right">
                {t('matches.played_at_time')}
              </Label>
              <Input
                id="playedAtTime"
                type="time"
                value={playedAtTime}
                onChange={(e) => setPlayedAtTime(e.target.value)}
                className="col-span-3"
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? t('common.loading') : t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}