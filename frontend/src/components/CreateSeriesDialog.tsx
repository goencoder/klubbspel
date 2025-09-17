import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ClubSelector } from '@/components/ClubSelector'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
import type { Series, SeriesVisibility, Club } from '@/types/api'
import { toast } from 'sonner'
import { useAuthStore } from '@/store/auth'

interface CreateSeriesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSeriesCreated: (series: Series) => void
}

export function CreateSeriesDialog({
  open,
  onOpenChange,
  onSeriesCreated,
}: CreateSeriesDialogProps) {
  const { t } = useTranslation()
  const { isPlatformOwner, isClubAdmin, selectedClubId } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [manageableClubs, setManageableClubs] = useState<Club[]>([])
  const [formData, setFormData] = useState<{
    title: string
    visibility: SeriesVisibility
    clubId: string
    startsAt: string
    endsAt: string
  }>({
    title: '',
    visibility: 'SERIES_VISIBILITY_OPEN',
    clubId: '',
    startsAt: '',
    endsAt: '',
  })
  const [hasManualClubSelection, setHasManualClubSelection] = useState(false)

  const loadManageableClubs = useCallback(async () => {
    try {
      const response = await apiClient.listClubs({ pageSize: 100 })

      const nextManageableClubs: Club[] = []
      const userIsPlatformOwner = isPlatformOwner()

      for (const club of response.items) {
        if (userIsPlatformOwner || isClubAdmin(club.id)) {
          nextManageableClubs.push(club)
        }
      }

      setManageableClubs(nextManageableClubs)

      if (!hasManualClubSelection) {
        setFormData(prev => {
          const selectedClubExists = selectedClubId
            ? nextManageableClubs.some(club => club.id === selectedClubId)
            : false

          if (selectedClubExists) {
            return prev.clubId === selectedClubId
              ? prev
              : { ...prev, clubId: selectedClubId }
          }

          if (nextManageableClubs.length === 1) {
            const [onlyClub] = nextManageableClubs
            return prev.clubId === onlyClub.id
              ? prev
              : { ...prev, clubId: onlyClub.id }
          }

          if (prev.clubId && !nextManageableClubs.some(club => club.id === prev.clubId)) {
            return { ...prev, clubId: '' }
          }

          return prev
        })
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : ''
      toast.error(message || t('errors.unexpectedError'))
    }
  }, [hasManualClubSelection, isClubAdmin, isPlatformOwner, selectedClubId, t])

  // Reset form when dialog closes
  useEffect(() => {
    if (open) {
      loadManageableClubs()
    } else {
      setFormData({
        title: '',
        visibility: 'SERIES_VISIBILITY_OPEN',
        clubId: '',
        startsAt: '',
        endsAt: '',
      })
      setManageableClubs([])
      setHasManualClubSelection(false)
    }
  }, [open, loadManageableClubs])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Basic client-side checks (server is source of truth)
    if (!formData.title || !formData.startsAt || !formData.endsAt) {
      toast.error(t('form.validation.error'))
      return
    }

    if (new Date(formData.startsAt) >= new Date(formData.endsAt)) {
      toast.error(t('series.validation.start_before_end'))
      return
    }

    if (
      formData.visibility === 'SERIES_VISIBILITY_CLUB_ONLY' &&
      !formData.clubId
    ) {
      toast.error(t('series.validation.club_required'))
      return
    }

    try {
      setLoading(true)

      // Convert datetime-local to RFC3339
      const startsAt = new Date(formData.startsAt).toISOString()
      const endsAt = new Date(formData.endsAt).toISOString()

      const payload: {
        title: string
        visibility: SeriesVisibility
        startsAt: string
        endsAt: string
        clubId?: string
      } = {
        title: formData.title,
        visibility: formData.visibility,
        startsAt,
        endsAt,
        ...(formData.visibility === 'SERIES_VISIBILITY_CLUB_ONLY' && formData.clubId
          ? { clubId: formData.clubId }
          : {}),
      }

      const series = await apiClient.createSeries(payload)
      onSeriesCreated(series)
      toast.success(t('series.created'))
      onOpenChange(false)
    } catch (error: unknown) {
      // grpc-gateway rpcStatus: { code, message, details }
      const message = error instanceof Error ? error.message : ''
      toast.error(message || t('errors.unexpectedError'))
    } finally {
      setLoading(false)
    }
  }

  const handleClubSelected = (club: Club | null) => {
    setHasManualClubSelection(true)
    setFormData((prev) => ({ ...prev, clubId: club?.id || '' }))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{t('series.createNew')}</DialogTitle>
          <DialogDescription>
            {t('series.create_help', 'Create a new tournament series.')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="title">{t('series.name')} *</Label>
            <Input
              id="title"
              value={formData.title}
              onChange={(e) =>
                setFormData((prev) => ({ ...prev, title: e.target.value }))
              }
              placeholder={t('series.name_placeholder')}
              required
            />
          </div>

          {/* Visibility */}
          <div className="space-y-2">
            <Label htmlFor="visibility">{t('series.visibility')} *</Label>
            <Select
              value={formData.visibility}
              onValueChange={(value) =>
                setFormData((prev) => {
                  const nextVisibility = value as SeriesVisibility
                  const nextState = {
                    ...prev,
                    visibility: nextVisibility,
                  }

                  if (nextVisibility !== 'SERIES_VISIBILITY_CLUB_ONLY') {
                    return nextState
                  }

                  if (hasManualClubSelection && prev.clubId) {
                    return nextState
                  }

                  if (selectedClubId && prev.clubId !== selectedClubId) {
                    return { ...nextState, clubId: selectedClubId }
                  }

                  if (!prev.clubId && manageableClubs.length === 1) {
                    return { ...nextState, clubId: manageableClubs[0].id }
                  }

                  if (prev.clubId && manageableClubs.some(club => club.id === prev.clubId)) {
                    return { ...nextState, clubId: prev.clubId }
                  }

                  return { ...nextState, clubId: '' }
                })
              }
            >
              <SelectTrigger id="visibility">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="SERIES_VISIBILITY_OPEN">
                  {t('series.visibility.open')}
                </SelectItem>
                <SelectItem value="SERIES_VISIBILITY_CLUB_ONLY">
                  {t('series.visibility.club_only')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Club (only when CLUB_ONLY) */}
          {formData.visibility === 'SERIES_VISIBILITY_CLUB_ONLY' && (
            <div className="space-y-2">
              <Label>{t('players.club')} *</Label>
              <ClubSelector
                clubs={manageableClubs}
                selectedClubId={formData.clubId}
                onClubSelected={handleClubSelected}
                disabled={loading}
              />
            </div>
          )}

          {/* Dates */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="startsAt">{t('series.starts_at')} *</Label>
              <Input
                id="startsAt"
                type="datetime-local"
                value={formData.startsAt}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    startsAt: e.target.value,
                  }))
                }
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="endsAt">{t('series.ends_at')} *</Label>
              <Input
                id="endsAt"
                type="datetime-local"
                value={formData.endsAt}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, endsAt: e.target.value }))
                }
                required
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? <LoadingSpinner size="sm" /> : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
