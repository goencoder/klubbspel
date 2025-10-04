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
import { deriveAutomaticClubId } from '@/lib/clubSelection'
import type { Series, SeriesVisibility, Club, Sport, SeriesFormat, LadderRules } from '@/types/api'
import { toast } from 'sonner'
import { SUPPORTED_SPORTS, DEFAULT_SPORT, SUPPORTED_SERIES_FORMATS, DEFAULT_SERIES_FORMAT, sportTranslationKey, seriesFormatTranslationKey } from '@/lib/sports'
import { useAuthStore } from '@/store/auth'

interface CreateSeriesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSeriesCreated: (series: Series) => void
}

function resolveClubIdForClubOnlyVisibility({
  previousClubId,
  hasManualClubSelection,
  selectedClubId,
  manageableClubs,
}: {
  previousClubId: string
  hasManualClubSelection: boolean
  selectedClubId?: string | null
  manageableClubs: Club[]
}) {
  if (hasManualClubSelection && previousClubId) {
    return previousClubId
  }

  return deriveAutomaticClubId({
    manageableClubs,
    selectedClubId,
    previousClubId,
  })
}

export function CreateSeriesDialog({
  open,
  onOpenChange,
  onSeriesCreated,
}: CreateSeriesDialogProps) {
  const { t } = useTranslation()
  const { isPlatformOwner, isClubAdmin, selectedClubId } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [clubs, setClubs] = useState<Club[]>([])
  const [availableSports, setAvailableSports] = useState<Sport[]>(SUPPORTED_SPORTS)
  const [manageableClubs, setManageableClubs] = useState<Club[]>([])
  const [formData, setFormData] = useState<{
    title: string
    visibility: SeriesVisibility
    clubId: string
    startsAt: string
    endsAt: string
    sport: Sport
    format: SeriesFormat
    ladderRules: LadderRules
    setsToPlay: number
  }>({
    title: '',
    visibility: 'SERIES_VISIBILITY_OPEN',
    clubId: '',
    startsAt: '',
    endsAt: '',
    sport: DEFAULT_SPORT,
    format: DEFAULT_SERIES_FORMAT,
    ladderRules: 'LADDER_RULES_CLASSIC',
    setsToPlay: 5,
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
          const nextClubId = deriveAutomaticClubId({
            manageableClubs: nextManageableClubs,
            selectedClubId,
            previousClubId: prev.clubId,
          })

          return nextClubId === prev.clubId ? prev : { ...prev, clubId: nextClubId }
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
        sport: DEFAULT_SPORT,
        format: DEFAULT_SERIES_FORMAT,
        ladderRules: 'LADDER_RULES_CLASSIC',
        setsToPlay: 5,
      })
      setAvailableSports(SUPPORTED_SPORTS)
      setClubs([])
      setManageableClubs([])
      setHasManualClubSelection(false)
    }
  }, [open, loadManageableClubs])

  const loadClubs = useCallback(async () => {
    try {
      const response = await apiClient.listClubs({ pageSize: 100 })
      setClubs(response.items)
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.unexpectedError'))
    }
  }, [t])

  useEffect(() => {
    if (open) {
      loadClubs()
    }
  }, [open, loadClubs])

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

      const clubIdPayload =
        formData.visibility === 'SERIES_VISIBILITY_CLUB_ONLY' && formData.clubId
          ? { clubId: formData.clubId }
          : {}

      const payload: {
        title: string
        visibility: SeriesVisibility
        startsAt: string
        endsAt: string
        clubId?: string
        sport: Sport
        format: SeriesFormat
        ladderRules?: LadderRules
        setsToPlay: number
      } = {
        title: formData.title,
        visibility: formData.visibility,
        startsAt,
        endsAt,
        sport: formData.sport,
        format: formData.format,
        setsToPlay: formData.setsToPlay,
        ...(formData.clubId && { clubId: formData.clubId }),
        ...(formData.format === 'SERIES_FORMAT_LADDER' && { ladderRules: formData.ladderRules }),
        ...clubIdPayload,
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
    const sports = club?.supportedSports?.length ? club.supportedSports : SUPPORTED_SPORTS
    setAvailableSports(sports)

    setFormData((prev) => {
      const nextSport = sports.includes(prev.sport) ? prev.sport : (sports[0] ?? DEFAULT_SPORT)
      return {
        ...prev,
        clubId: club?.id || '',
        sport: nextSport,
      }
    })
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
              onValueChange={(value) => {
                const nextVisibility = value as SeriesVisibility

                setFormData(prev => {
                  if (nextVisibility !== 'SERIES_VISIBILITY_CLUB_ONLY') {
                    return prev.visibility === nextVisibility
                      ? prev
                      : { ...prev, visibility: nextVisibility }
                  }

                  const nextClubId = resolveClubIdForClubOnlyVisibility({
                    previousClubId: prev.clubId,
                    hasManualClubSelection,
                    selectedClubId,
                    manageableClubs,
                  })

                  if (
                    prev.visibility === nextVisibility &&
                    nextClubId === prev.clubId
                  ) {
                    return prev
                  }

                  return {
                    ...prev,
                    visibility: nextVisibility,
                    clubId: nextClubId,
                  }
                })

                // Update available sports based on visibility and club
                if (nextVisibility === 'SERIES_VISIBILITY_OPEN') {
                  setAvailableSports(SUPPORTED_SPORTS)
                } else if (nextVisibility === 'SERIES_VISIBILITY_CLUB_ONLY' && formData.clubId) {
                  const selectedClub = clubs.find((clubItem) => clubItem.id === formData.clubId)
                  const sports = selectedClub?.supportedSports?.length ? selectedClub.supportedSports : SUPPORTED_SPORTS
                  setAvailableSports(sports)
                }
              }}
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

          {/* Sport */}
          <div className="space-y-2">
            <Label htmlFor="sport">{t('series.sportLabel')}</Label>
            {availableSports.length > 1 ? (
              <Select
                value={formData.sport}
                onValueChange={(value) =>
                  setFormData((prev) => ({
                    ...prev,
                    sport: value as Sport,
                  }))
                }
              >
                <SelectTrigger id="sport">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {availableSports.map((sport) => (
                    <SelectItem key={sport} value={sport}>
                      {t(sportTranslationKey(sport))}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <div className="text-sm text-muted-foreground border rounded-md p-2">
                {t(sportTranslationKey(formData.sport))}
              </div>
            )}
          </div>

          {/* Format */}
          <div className="space-y-2">
            <Label htmlFor="format">{t('series.formatLabel')}</Label>
            {SUPPORTED_SERIES_FORMATS.length > 1 ? (
              <Select
                value={formData.format}
                onValueChange={(value) =>
                  setFormData((prev) => ({
                    ...prev,
                    format: value as SeriesFormat,
                  }))
                }
              >
                <SelectTrigger id="format">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SUPPORTED_SERIES_FORMATS.map((format) => (
                    <SelectItem key={format} value={format}>
                      {t(seriesFormatTranslationKey(format))}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <div className="text-sm text-muted-foreground border rounded-md p-2">
                {t(seriesFormatTranslationKey(formData.format))}
              </div>
            )}
          </div>

          {/* Ladder Rules (only show when format is LADDER) */}
          {formData.format === 'SERIES_FORMAT_LADDER' && (
            <div className="space-y-2">
              <Label htmlFor="ladderRules">{t('series.ladderRulesLabel', 'Ladder Rules')}</Label>
              <Select
                value={formData.ladderRules}
                onValueChange={(value) =>
                  setFormData((prev) => ({
                    ...prev,
                    ladderRules: value as LadderRules,
                  }))
                }
              >
                <SelectTrigger id="ladderRules">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="LADDER_RULES_CLASSIC">
                    {t('series.ladderClassic', 'Klassisk stege (ingen straff)')}
                  </SelectItem>
                  <SelectItem value="LADDER_RULES_AGGRESSIVE">
                    {t('series.ladderAggressive', 'Aggressiv stege (straff vid förlust)')}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {formData.ladderRules === 'LADDER_RULES_CLASSIC' 
                  ? t('series.ladderClassicHelp', 'Klassisk: Bara vinnaren klättrar uppåt')
                  : t('series.ladderAggressiveHelp', 'Aggressiv: Förloraren faller nedåt')}
              </p>
            </div>
          )}

          {/* Sets to Play (for all racket/paddle sports) */}
          {(formData.sport === 'SPORT_TABLE_TENNIS' || 
            formData.sport === 'SPORT_TENNIS' ||
            formData.sport === 'SPORT_PADEL' ||
            formData.sport === 'SPORT_BADMINTON' ||
            formData.sport === 'SPORT_SQUASH' ||
            formData.sport === 'SPORT_PICKLEBALL') && (
            <div className="space-y-2">
              <Label htmlFor="setsToPlay">{t('series.setsToPlay', 'Sets to Play')} *</Label>
              <Select
                value={formData.setsToPlay.toString()}
                onValueChange={(value) =>
                  setFormData((prev) => ({
                    ...prev,
                    setsToPlay: parseInt(value, 10),
                  }))
                }
              >
                <SelectTrigger id="setsToPlay">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="3">{t('series.bestOf3', 'Best of 3')}</SelectItem>
                  <SelectItem value="5">{t('series.bestOf5', 'Best of 5')}</SelectItem>
                  <SelectItem value="7">{t('series.bestOf7', 'Best of 7')}</SelectItem>
                </SelectContent>
              </Select>
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
