import { useState, useEffect } from 'react'
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
  const [loading, setLoading] = useState(false)
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

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setFormData({
        title: '',
        visibility: 'SERIES_VISIBILITY_OPEN',
        clubId: '',
        startsAt: '',
        endsAt: '',
      })
    }
  }, [open])

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
        ...(formData.clubId && { clubId: formData.clubId }),
      }

      const series = await apiClient.createSeries(payload)
      onSeriesCreated(series)
      toast.success(t('series.created'))
      onOpenChange(false)
    } catch (error: any) {
      // grpc-gateway rpcStatus: { code, message, details }
      const msg = error?.message || t('errors.unexpectedError')
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleClubSelected = (club: Club | null) => {
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
                setFormData((prev) => ({
                  ...prev,
                  visibility: value as SeriesVisibility,
                  // reset club if switching away
                  clubId:
                    value === 'SERIES_VISIBILITY_CLUB_ONLY' ? prev.clubId : '',
                }))
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
                value={formData.clubId}
                onClubSelected={handleClubSelected} 
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
