import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Add, SearchNormal1, Calendar, Cup } from 'iconsax-reactjs'
import { CreateSeriesDialog } from '@/components/CreateSeriesDialog'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { useDebounce } from '@/hooks/useDebounce'
import { apiClient } from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { Series, SeriesVisibility, Club } from '@/types/api'
import { toast } from 'sonner'
import { PageWrapper, PageHeaderSection, HeaderContent, SearchSection } from './Styles'
import { sportTranslationKey, seriesFormatTranslationKey } from '@/lib/sports'

interface SeriesByClub {
  [clubId: string]: {
    club: Club | null
    series: Series[]
  }
}

export function SeriesListPage() {
  const { t } = useTranslation()
  const { user: _user, selectedClubId } = useAuthStore()
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [seriesByClub, setSeriesByClub] = useState<SeriesByClub>({})
  const [openSeries, setOpenSeries] = useState<Series[]>([])
  const [clubs, setClubs] = useState<{ [id: string]: Club }>({})
  const [loading, setLoading] = useState(true)
  
  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  const loadSeries = useCallback(async () => {
    try {
      setLoading(true)
      
      // Get selected club and user from auth store
      const { selectedClubId, user } = useAuthStore.getState()
      
      // Build filter based on selected club
      let clubFilter: string[] = []
      
      if (!selectedClubId || selectedClubId === 'all-clubs') {
        // "Alla klubbar" - send empty list to show all series
        clubFilter = []
      } else if (selectedClubId === 'my-clubs') {
        // "Mina klubbar" - resolve to actual club IDs + open series
        const userClubIds = user?.memberships?.map(m => m.clubId).filter(Boolean) || []
        clubFilter = [...userClubIds, 'OPEN']
      } else {
        // Specific club selected - send that club ID + open series
        clubFilter = [selectedClubId, 'OPEN']
      }
      
      const response = await apiClient.listSeries({
        pageSize: 50,
        clubFilter
      }, 'series-list')
      
      // Load clubs information for displaying club names
      const allClubIds = [...new Set(response.items.map(s => s.clubId).filter((id): id is string => Boolean(id)))]
      const clubsData: { [id: string]: Club } = {}
      
      await Promise.all(
        allClubIds.map(async (clubId) => {
          try {
            const club = await apiClient.getClub(clubId)
            clubsData[clubId] = club
          } catch (_error) {
            // Club might not exist or be accessible - silently skip
            // Ignore error
          }
        })
      )
      
      setClubs(clubsData)
      
      // Separate series by club and open series
      const seriesByClubData: SeriesByClub = {}
      const openSeriesData: Series[] = []
      
      response.items.forEach(series => {
        if (series.visibility === 'SERIES_VISIBILITY_OPEN') {
          openSeriesData.push(series)
        } else if (series.clubId) {
          if (!seriesByClubData[series.clubId]) {
            seriesByClubData[series.clubId] = {
              club: clubsData[series.clubId] || null,
              series: []
            }
          }
          seriesByClubData[series.clubId].series.push(series)
        }
      })
      
      setSeriesByClub(seriesByClubData)
      setOpenSeries(openSeriesData)
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.unexpectedError'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    loadSeries()
  }, [debouncedSearchQuery, loadSeries, selectedClubId])

  const handleSeriesCreated = (newSeries: Series) => {
    // Add new series to the appropriate section
    if (newSeries.visibility === 'SERIES_VISIBILITY_OPEN') {
      setOpenSeries(prev => [newSeries, ...prev])
    } else if (newSeries.clubId) {
      const clubId = newSeries.clubId
      setSeriesByClub(prev => ({
        ...prev,
        [clubId]: {
          club: clubs[clubId] || null,
          series: [newSeries, ...(prev[clubId]?.series || [])]
        }
      }))
    }
    setShowCreateDialog(false)
    toast.success(t('series.created'))
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

  // Get all series for searching
  const allSeries = [
    ...openSeries,
    ...Object.values(seriesByClub).flatMap(club => club.series)
  ]

  // Filter series by search query (client-side for now)
  const filteredSeries = debouncedSearchQuery
    ? allSeries.filter(s => s.title.toLowerCase().includes(debouncedSearchQuery.toLowerCase()))
    : allSeries

  const renderSeriesCard = (seriesItem: Series) => (
    <Card key={seriesItem.id} className="hover:shadow-md transition-shadow">
      <CardHeader>
        <div className="flex justify-between items-start">
          <CardTitle className="text-lg">{seriesItem.title}</CardTitle>
          <Badge variant={getVisibilityVariant(seriesItem.visibility)}>
            {getVisibilityLabel(seriesItem.visibility)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <div className="flex items-center text-sm text-muted-foreground">
            <Calendar size={16} className="mr-2 text-current" />
            <span>
              {formatDateRange(seriesItem.startsAt, seriesItem.endsAt)}
            </span>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <Badge variant="outline">
              {t(sportTranslationKey(seriesItem.sport))}
            </Badge>
            <Badge variant="outline">
              {t(seriesFormatTranslationKey(seriesItem.format))}
            </Badge>
          </div>
          <Link to={`/series/${seriesItem.id}`}>
            <Button variant="outline" className="w-full">
              {t('series.viewDetails')}
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  )

  return (
    <PageWrapper>
      <PageHeaderSection>
        <HeaderContent>
          <h1 className="text-3xl font-bold text-foreground">{t('series.title')}</h1>
          <p className="text-muted-foreground">{t('series.subtitle')}</p>
        </HeaderContent>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Add size={16} className="text-current" />
          <span className="ml-2">{t('series.createNew')}</span>
        </Button>
      </PageHeaderSection>

      <SearchSection>
        <div className="relative flex-1 max-w-md">
          <SearchNormal1 className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground" size={16} />
          <Input
            placeholder={t('common.search') + '...'}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
      </SearchSection>

      {loading ? (
        <LoadingSpinner />
      ) : filteredSeries.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Cup size={48} className="text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold text-foreground mb-2">{t('series.noSeries')}</h3>
            <p className="text-muted-foreground text-center mb-4">
              {searchQuery ? t('series.searchAdjust') : t('series.createFirst')}
            </p>
            {!searchQuery && (
              <Button onClick={() => setShowCreateDialog(true)}>
                <Add size={16} className="text-current" />
                <span className="ml-2">{t('series.createNew')}</span>
              </Button>
            )}
          </CardContent>
        </Card>
      ) : debouncedSearchQuery ? (
        // Show filtered results without sections when searching
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredSeries.map(renderSeriesCard)}
          </div>
        </div>
      ) : (
        // Show sectioned results when not searching
        <div className="space-y-8">
          {/* Club Series Sections First */}
          {Object.entries(seriesByClub).map(([clubId, clubData]) => (
            <div key={clubId}>
              <h2 className="text-xl font-semibold mb-4">
                {clubData.club?.name || t('series.sections.unknownClub')}
              </h2>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {clubData.series.map(renderSeriesCard)}
              </div>
            </div>
          ))}

          {/* Open Series Section Last */}
          {openSeries.length > 0 && (
            <div>
              <h2 className="text-xl font-semibold mb-4">{t('series.sections.openSeries')}</h2>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {openSeries.map(renderSeriesCard)}
              </div>
            </div>
          )}
        </div>
      )}

      <CreateSeriesDialog 
        open={showCreateDialog} 
        onOpenChange={setShowCreateDialog}
        onSeriesCreated={handleSeriesCreated}
      />
    </PageWrapper>
  )
}