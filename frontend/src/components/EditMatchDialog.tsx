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
  const [playedAt, setPlayedAt] = useState<string>('')

  useEffect(() => {
    if (match) {
      setScoreA(match.scoreA)
      setScoreB(match.scoreB)
      // Convert ISO date to input date format (YYYY-MM-DD)
      setPlayedAt(new Date(match.playedAt).toISOString().split('T')[0])
    }
  }, [match])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!match) return

    // Validation
    if (scoreA === scoreB) {
      toast.error(t('matches.validation.noTies'))
      return
    }

    if (scoreA < 0 || scoreA > 5 || scoreB < 0 || scoreB > 5) {
      toast.error(t('matches.validation.invalidScore'))
      return
    }

    if (Math.max(scoreA, scoreB) < 3) {
      toast.error(t('matches.validation.bestOfFive'))
      return
    }

    // Validate match date is within series window if being updated (inclusive)
    if (seriesStartDate && seriesEndDate && playedAt !== new Date(match.playedAt).toISOString().split('T')[0]) {
      const matchDate = new Date(playedAt)
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
      
      const updateRequest: UpdateMatchRequest = {
        matchId: match.id,
        scoreA: scoreA !== match.scoreA ? scoreA : undefined,
        scoreB: scoreB !== match.scoreB ? scoreB : undefined,
        playedAt: playedAt !== new Date(match.playedAt).toISOString().split('T')[0] 
          ? new Date(playedAt).toISOString() 
          : undefined,
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
                max="5"
                value={scoreA}
                onChange={(e) => setScoreA(parseInt(e.target.value, 10) || 0)}
                className="col-span-3"
                required
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
                max="5"
                value={scoreB}
                onChange={(e) => setScoreB(parseInt(e.target.value, 10) || 0)}
                className="col-span-3"
                required
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="playedAt" className="text-right">
                {t('matches.played_at')}
              </Label>
              <Input
                id="playedAt"
                type="date"
                value={playedAt}
                onChange={(e) => setPlayedAt(e.target.value)}
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