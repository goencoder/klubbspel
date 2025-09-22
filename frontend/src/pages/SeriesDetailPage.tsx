import { LoadingSpinner } from '@/components/LoadingSpinner'
import { MatchesList } from '@/components/MatchesList'
import { ReportMatchDialog } from '@/components/ReportMatchDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { apiClient } from '@/services/api'
import type { Series, SeriesVisibility } from '@/types/api'
import { Add, ArrowLeft2, Calendar, Chart, ClipboardTick, Cup } from 'iconsax-reactjs'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { sportTranslationKey, seriesFormatTranslationKey } from '@/lib/sports'
import { useDebounce } from '@/hooks/useDebounce'
import type { ReportedMatch } from '@/hooks/useMatchReporting'

export function SeriesDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const [series, setSeries] = useState<Series | null>(null)
  const [clubName, setClubName] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [showReportDialog, setShowReportDialog] = useState(false)
  const [matchesRefreshKey, setMatchesRefreshKey] = useState(0)

  const loadSeries = useCallback(async (seriesId: string) => {
    try {
      setLoading(true)
      const seriesData = await apiClient.getSeries(seriesId)
      setSeries(seriesData)

      // Load club name if clubId is present
      if (seriesData.clubId) {
        loadClubName(seriesData.clubId)
      }
    } catch (error: unknown) {
      const errorMessage = (error as Error).message || '';
      // If getSeries is not implemented, show a helpful error
      if (errorMessage.includes('not implemented')) {
        toast.error('Series details endpoint not yet implemented in backend')
      } else {
        toast.error(errorMessage || t('errors.generic'))
      }
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (id) {
      loadSeries(id)
    }
  }, [id, loadSeries])

  const loadClubName = async (clubId: string) => {
    try {
      const club = await apiClient.getClub(clubId)
      setClubName(club.name)
    } catch (_error: unknown) {
      // Keep the ID as fallback if club name fetch fails
      setClubName(clubId)
    }
  }

  const getVisibilityLabel = (visibility: SeriesVisibility) => {
    return visibility === 'SERIES_VISIBILITY_OPEN' ? t('series.visibility.open') : t('series.visibility.club_only')
  }

  const getVisibilityVariant = (visibility: SeriesVisibility) => {
    return visibility === 'SERIES_VISIBILITY_OPEN' ? 'default' : 'secondary'
  }

  const formatDateRange = (startDate: string, endDate: string) => {
    try {
      const start = new Date(startDate)
      const end = new Date(endDate)

      // Check if dates are valid
      if (isNaN(start.getTime()) || isNaN(end.getTime())) {
        return t('errors.invalidDates')
      }

      return `${start.toLocaleDateString()} - ${end.toLocaleDateString()}`
    } catch (_error) {
      return t('errors.invalidDates')
    }
  }

  // Debounce the refresh to avoid excessive API calls during rapid match entry
  const debouncedRefreshKey = useDebounce(matchesRefreshKey, 500)
  const [pendingRefresh, setPendingRefresh] = useState(false)

  // Memoize handlers to prevent unnecessary re-renders
  const handleMatchReported = useCallback((_reportedMatch: ReportedMatch) => {
    setPendingRefresh(true)
    setMatchesRefreshKey((prev) => prev + 1)
  }, [])

  // Clear pending state when debounced refresh completes
  useEffect(() => {
    if (pendingRefresh && debouncedRefreshKey > 0) {
      setPendingRefresh(false)
    }
  }, [debouncedRefreshKey, pendingRefresh])

  if (loading) {
    return <LoadingSpinner />
  }

  if (!series) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-foreground mb-2">Series not found</h2>
        <p className="text-muted-foreground mb-4">The series you're looking for doesn't exist.</p>
        <Link to="/">
          <Button>
            <ArrowLeft2 size={16} />
            <span className="ml-2">Back to Series</span>
          </Button>
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center space-x-4">
        <Link to="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft2 size={16} className="text-muted-foreground" />
            <span className="ml-2">{t('common.back')}</span>
          </Button>
        </Link>
      </div>

      {/* Series Info */}
      <Card>
        <CardHeader>
          <div className="flex flex-col space-y-4 sm:flex-row sm:justify-between sm:items-start sm:space-y-0">
            <div className="space-y-2 flex-1 min-w-0">
              <div className="flex items-center space-x-3">
                <CardTitle className="text-2xl">{series.title}</CardTitle>
                <Badge variant={getVisibilityVariant(series.visibility)}>
                  {getVisibilityLabel(series.visibility)}
                </Badge>
              </div>
              <CardDescription className="text-base">
                {series.visibility === 'SERIES_VISIBILITY_OPEN' ? t('series.visibility.open_description') : t('series.visibility.club_only_description')}
              </CardDescription>
            </div>
            <div className="flex flex-col space-y-2 sm:flex-row sm:space-y-0 sm:space-x-2 sm:flex-shrink-0">
              <Button 
                onClick={() => setShowReportDialog(true)}
                className="w-full sm:w-auto whitespace-nowrap"
                size="sm"
              >
                <Add size={16} className="text-current" />
                <span className="ml-2">{t('series.report.match')}</span>
              </Button>
              <Link to={`/series/${series.id}/leaderboard`}>
                <Button 
                  variant="outline"
                  className="w-full sm:w-auto whitespace-nowrap"
                  size="sm"
                >
                  <ClipboardTick size={16} className="text-blue-600" />
                  <span className="ml-2">{t('series.leaderboard')}</span>
                </Button>
              </Link>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div className="flex items-center text-sm">
                <Calendar size={16} className="mr-2 text-muted-foreground" />
                <span className="text-muted-foreground mr-2">Duration:</span>
                <span className="font-medium">
                  {formatDateRange(series.startsAt, series.endsAt)}
                </span>
              </div>
              <div className="flex items-center text-sm">
                <Cup size={16} className="mr-2 text-muted-foreground" />
                <span className="text-muted-foreground mr-2">{t('series.sportLabel')}:</span>
                <span className="font-medium">{t(sportTranslationKey(series.sport))}</span>
              </div>
              <div className="flex items-center text-sm">
                <ClipboardTick size={16} className="mr-2 text-muted-foreground" />
                <span className="text-muted-foreground mr-2">{t('series.formatLabel')}:</span>
                <span className="font-medium">{t(seriesFormatTranslationKey(series.format))}</span>
              </div>
              {series.clubId && (
                <div className="flex items-center text-sm">
                  <Cup size={16} className="mr-2 text-muted-foreground" />
                  <span className="text-muted-foreground mr-2">{t('players.club')}:</span>
                  <span className="font-medium">{clubName || series.clubId}</span>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Content Tabs */}
      <Tabs defaultValue="matches" className="space-y-6">
        <TabsList>
          <TabsTrigger value="matches">{t('series.matches')}</TabsTrigger>
          <TabsTrigger value="leaderboard">{t('series.leaderboard')}</TabsTrigger>
        </TabsList>

        <TabsContent value="matches" className="space-y-6">
          <MatchesList
            seriesId={series.id}
            seriesStartDate={series.startsAt}
            seriesEndDate={series.endsAt}
            seriesName={series.title}
            refreshKey={debouncedRefreshKey}
          />
        </TabsContent>

        <TabsContent value="leaderboard" className="space-y-6">
          <div className="text-center py-8">
            <p className="text-muted-foreground mb-4">
              View the full leaderboard for detailed rankings and statistics
            </p>
            <Link to={`/series/${series.id}/leaderboard`}>
              <Button>
                <Chart size={16} className="text-current" />
                <span className="ml-2">View Full Leaderboard</span>
              </Button>
            </Link>
          </div>
        </TabsContent>
      </Tabs>

      <ReportMatchDialog
        open={showReportDialog}
        onOpenChange={setShowReportDialog}
        seriesId={series.id}
        clubId={series.visibility === 'SERIES_VISIBILITY_CLUB_ONLY' ? series.clubId : undefined}
        seriesStartDate={series.startsAt}
        seriesEndDate={series.endsAt}
        onMatchReported={handleMatchReported}
      />
    </div>
  )
}