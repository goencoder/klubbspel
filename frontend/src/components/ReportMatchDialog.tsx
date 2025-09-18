import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PlayerSelector } from '@/components/PlayerSelector'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
import type { Player } from '@/types/api'
import { toast } from 'sonner'

interface ReportMatchDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  seriesId: string
  clubId?: string
  onMatchReported: () => void
}

export function ReportMatchDialog({ 
  open, 
  onOpenChange, 
  seriesId, 
  clubId,
  onMatchReported 
}: ReportMatchDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    player_a_id: '',
    player_b_id: '',
    score_a: '',
    score_b: '',
    played_at: ''
  })

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setFormData({
        player_a_id: '',
        player_b_id: '',
        score_a: '',
        score_b: '',
        played_at: new Date().toISOString().split('T')[0]
      })
    } else {
      // Set default date to today
      setFormData(prev => ({
        ...prev,
        played_at: new Date().toISOString().split('T')[0]
      }))
    }
  }, [open])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!formData.player_a_id || !formData.player_b_id || 
        !formData.score_a || !formData.score_b || !formData.played_at) {
      toast.error(t('form.validation.error'))
      return
    }

    if (formData.player_a_id === formData.player_b_id) {
      toast.error('A player cannot play against themselves')
      return
    }

    const scoreA = parseInt(formData.score_a, 10)
    const scoreB = parseInt(formData.score_b, 10)

    if (scoreA < 0 || scoreB < 0 || scoreA > 5 || scoreB > 5) {
      toast.error(t('matches.validation.scoresRange'))
      return
    }

    if (scoreA === scoreB) {
      toast.error(t('matches.validation.noTie'))
      return
    }

    if (Math.max(scoreA, scoreB) < 3) {
      toast.error('Winner must reach at least 3 games')
      return
    }

    try {
      setLoading(true)
      
      const reportRequest = {
        seriesId: seriesId,
        playerAId: formData.player_a_id,
        playerBId: formData.player_b_id,
        scoreA: scoreA,
        scoreB: scoreB,
        playedAt: new Date(formData.played_at).toISOString()
      }

      await apiClient.reportMatch(reportRequest)
      onMatchReported()
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

  const isFormValid = formData.player_a_id && formData.player_b_id && 
    formData.score_a && formData.score_b && formData.played_at &&
    formData.player_a_id !== formData.player_b_id

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{t('matches.report')}</DialogTitle>
          <DialogDescription>
            Report the result of a table tennis match
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid gap-6">
            {/* Player Selection */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t('matches.player_a')} *</Label>
                <PlayerSelector 
                  onPlayerSelected={handlePlayerASelected}
                  clubId={clubId}
                  excludePlayerId={formData.player_b_id}
                />
              </div>
              <div className="space-y-2">
                <Label>{t('matches.player_b')} *</Label>
                <PlayerSelector 
                  onPlayerSelected={handlePlayerBSelected}
                  clubId={clubId}
                  excludePlayerId={formData.player_a_id}
                />
              </div>
            </div>

            {/* Score inputs */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="score_a">{t('matches.score_a')} *</Label>
                <Input
                  id="score_a"
                  type="number"
                  min="0"
                  max="5"
                  value={formData.score_a}
                  onChange={(e) => setFormData(prev => ({ ...prev, score_a: e.target.value }))}
                  placeholder="0"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="score_b">{t('matches.score_b')} *</Label>
                <Input
                  id="score_b"
                  type="number"
                  min="0"
                  max="5"
                  value={formData.score_b}
                  onChange={(e) => setFormData(prev => ({ ...prev, score_b: e.target.value }))}
                  placeholder="0"
                  required
                />
              </div>
            </div>

            {/* Date */}
            <div className="space-y-2">
              <Label htmlFor="played_at">{t('matches.played_at')} *</Label>
              <Input
                id="played_at"
                type="date"
                value={formData.played_at}
                onChange={(e) => setFormData(prev => ({ ...prev, played_at: e.target.value }))}
                required
              />
            </div>

            {/* Validation hints */}
            <div className="text-sm text-muted-foreground bg-muted p-3 rounded-md">
              <ul className="space-y-1">
                <li>• Best of 5 games format (winner must reach 3 games)</li>
                <li>• Scores must be between 0 and 5</li>
                <li>• No ties allowed</li>
                <li>• Players must be different</li>
              </ul>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!isFormValid || loading}>
              {loading ? <LoadingSpinner size="sm" /> : t('matches.report')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}