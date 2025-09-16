import { useState, useEffect } from 'react'
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
import type { Series, SeriesVisibility } from '@/types/api'
import { toast } from 'sonner'
import { PageWrapper, PageHeaderSection, HeaderContent, SearchSection, ContentGrid } from './Styles'

export function SeriesListPage() {
  const { t } = useTranslation()
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [series, setSeries] = useState<Series[]>([])
  const [loading, setLoading] = useState(true)
  const [endCursor, setEndCursor] = useState<string | undefined>()
  const [hasNextPage, setHasNextPage] = useState(false)
  
  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  useEffect(() => {
    loadSeries()
  }, [debouncedSearchQuery])

  const loadSeries = async (cursorAfter?: string) => {
    try {
      setLoading(true)
      const response = await apiClient.listSeries({
        pageSize: 20,
        cursorAfter
      }, 'series-list')
      
      if (cursorAfter) {
        setSeries(prev => [...prev, ...response.items])
      } else {
        setSeries(response.items)
      }
      setEndCursor(response.endCursor)
      setHasNextPage(response.hasNextPage)
    } catch (error: any) {
      toast.error(error.message || t('errors.unexpectedError'))
    } finally {
      setLoading(false)
    }
  }

  const handleSeriesCreated = (newSeries: Series) => {
    setSeries(prev => [newSeries, ...prev])
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
    } catch (error) {
      console.error('Date parsing error:', error, { startDate, endDate })
      return t('errors.invalidDates')
    }
  }

  // Filter series by search query (client-side for now)
  const filteredSeries = debouncedSearchQuery
    ? series.filter(s => s.title.toLowerCase().includes(debouncedSearchQuery.toLowerCase()))
    : series

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

      {loading && series.length === 0 ? (
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
      ) : (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredSeries.map((seriesItem) => (
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
                    <Link to={`/series/${seriesItem.id}`}>
                      <Button variant="outline" className="w-full">
                        {t('series.viewDetails')}
                      </Button>
                    </Link>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {hasNextPage && !searchQuery && (
            <div className="flex justify-center">
              <Button
                variant="outline"
                onClick={() => loadSeries(endCursor)}
                disabled={loading}
              >
                {loading ? <LoadingSpinner size="sm" /> : t('common.next')}
              </Button>
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