import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card, CardContent, CardDescription, CardHeader, CardTitle
} from '@/components/ui/card'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from '@/components/ui/table'
import { apiClient } from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { Series, Club } from '@/types/api'
import { Chart, CloseCircle, Cup, Export, Medal, TickCircle } from 'iconsax-reactjs'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { exportLeaderboardToCSV, type LeaderboardCSVData } from '@/utils/csvExport'

interface SeriesByClub {
  [clubId: string]: {
    club: Club | null
    series: Series[]
  }
}

type UILBRow = {
  rank: number
  playerId: string
  displayName: string
  rating: number
  games: number
  wins: number
  losses: number
}

// ---- helpers to tolerate API shape differences (rows vs entries, snake_case vs camelCase)
function normalizeLeaderboard(resp: { rows?: unknown[]; entries?: unknown[] } | unknown): UILBRow[] {
  const raw = ((resp as { rows?: unknown[]; entries?: unknown[] })?.rows ?? 
               (resp as { rows?: unknown[]; entries?: unknown[] })?.entries ?? []) as Array<Record<string, unknown>>
  if (!Array.isArray(raw)) return []
  return raw.map((r: Record<string, unknown>) => ({
    rank: Number(r.rank ?? 0),
    playerId: String(r.playerId ?? r.player_id ?? ''),
    displayName: String(r.displayName ?? r.playerName ?? r.display_name ?? ''),
    rating: Number(r.rating ?? r.eloRating ?? 0),
    games: Number(r.games ?? r.matchesPlayed ?? 0),
    wins: Number(r.wins ?? r.matchesWon ?? 0),
    losses: Number(r.losses ?? r.matchesLost ?? 0),
  }))
}

export function LeaderboardPage() {
  const { id: seriesIdFromParams } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const { selectedClubId, user } = useAuthStore()

  const [selectedSeriesId, setSelectedSeriesId] = useState<string>(seriesIdFromParams || '')
  const [leaderboard, setLeaderboard] = useState<UILBRow[]>([])
  const [seriesByClub, setSeriesByClub] = useState<SeriesByClub>({})
  const [openSeries, setOpenSeries] = useState<Series[]>([])
  const [_clubs, setClubs] = useState<{ [id: string]: Club }>({})
  const [loadingSeries, setLoadingSeries] = useState(true)
  const [loadingLeaderboard, setLoadingLeaderboard] = useState(false)

  const loadSeries = useCallback(async () => {
    try {
      setLoadingSeries(true)
      
      // Build filter based on selected club (same logic as SeriesListPage)
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
        pageSize: 100,
        clubFilter
      })
      
      const items: Series[] = Array.isArray(response?.items) ? response.items : []
      
      // Load clubs information for displaying club names
      const allClubIds = [...new Set(items.map(s => s.clubId).filter((id): id is string => Boolean(id)))]
      const clubsData: { [id: string]: Club } = {}
      
      await Promise.all(
        allClubIds.map(async (clubId) => {
          try {
            const club = await apiClient.getClub(clubId)
            clubsData[clubId] = club
          } catch (_error) {
            // Club might not exist or be accessible - silently skip
          }
        })
      )
      
      setClubs(clubsData)
      
      // Separate series by club and open series
      const seriesByClubData: SeriesByClub = {}
      const openSeriesData: Series[] = []
      
      items.forEach(series => {
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
      
      // Set default selected series if none selected and series exist
      if (!seriesIdFromParams && items.length > 0) {
        setSelectedSeriesId(items[0].id)
      }
    } catch (error: unknown) {
      toast.error((error as Error)?.message || t('error.generic'))
      setSeriesByClub({})
      setOpenSeries([])
    } finally {
      setLoadingSeries(false)
    }
  }, [selectedClubId, user, seriesIdFromParams, t])

  useEffect(() => {
    loadSeries()
  }, [loadSeries, selectedClubId])

  const loadLeaderboard = useCallback(async (seriesId: string) => {
    try {
      setLoadingLeaderboard(true)
      const resp = await apiClient.getLeaderboard({ seriesId, pageSize: 50 }, 'leaderboard')
      setLeaderboard(normalizeLeaderboard(resp))
    } catch (error: unknown) {
      toast.error((error as Error)?.message || t('error.generic'))
      setLeaderboard([])
    } finally {
      setLoadingLeaderboard(false)
    }
  }, [t])

  useEffect(() => {
    if (selectedSeriesId) {
      loadLeaderboard(selectedSeriesId)
    } else {
      setLeaderboard([])
    }
  }, [selectedSeriesId, loadLeaderboard])

  const calculateWinRate = (wins: number, games: number) => {
    if (!games) return 0
    return Math.round((wins / games) * 100)
  }

  const handleExportCSV = () => {
    const csvData: LeaderboardCSVData[] = leaderboard.map((row) => ({
      rank: row.rank,
      player: row.displayName,
      rating: Math.round(row.rating),
      games: row.games,
      wins: row.wins,
      losses: row.losses,
      winRate: `${calculateWinRate(row.wins, row.games)}%`
    }))
    
    exportLeaderboardToCSV(csvData, selectedSeries?.title)
    toast.success(t('leaderboard.exportSuccess'))
  }

  const getRankIcon = (rank: number) => {
    if (rank === 1) return <Medal size={20} className="text-yellow-500" variant="Bold" />
    if (rank === 2) return <Medal size={20} className="text-gray-400" variant="Bold" />
    if (rank === 3) return <Medal size={20} className="text-orange-600" variant="Bold" />
    return null
  }

  const getRankBadge = (rank: number) => {
    if (rank <= 3) return 'default' as const
    if (rank <= 10) return 'secondary' as const
    return 'outline' as const
  }

  // Get all series for finding selected series
  const allSeries = [
    ...Object.values(seriesByClub).flatMap(club => club.series),
    ...openSeries
  ]
  
  const selectedSeries = allSeries.find((s) => s.id === selectedSeriesId) || null

  if (loadingSeries) {
    return <LoadingSpinner />
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-foreground">{t('leaderboard.title')}</h1>
          <p className="text-muted-foreground">{t('leaderboard.subtitle', 'Player rankings and statistics')}</p>
        </div>
      </div>

      {/* Series Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2 text-lg">
            <Cup size={18} className="text-emerald-600" />
            <span>{t('leaderboard.select.series')}</span>
          </CardTitle>
          <CardDescription>
            {t('leaderboard.select.help')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Select value={selectedSeriesId || undefined} onValueChange={setSelectedSeriesId}>
            <SelectTrigger className="w-full max-w-md">
              <SelectValue placeholder={t('leaderboard.select.series')} />
            </SelectTrigger>
            <SelectContent>
              {/* Club Series Sections First */}
              {Object.entries(seriesByClub).map(([clubId, clubData]) => [
                // Club name header
                <div key={`header-${clubId}`} className="px-2 py-1.5 text-sm font-semibold text-muted-foreground bg-muted/50">
                  {clubData.club?.name || t('series.sections.unknownClub')}
                </div>,
                // Club series
                ...clubData.series.map((seriesItem) => (
                  <SelectItem key={seriesItem.id} value={seriesItem.id}>
                    {seriesItem.title}
                  </SelectItem>
                ))
              ]).flat()}
              
              {/* Separator if we have both club series and open series */}
              {Object.keys(seriesByClub).length > 0 && openSeries.length > 0 && (
                <Separator className="my-1" />
              )}
              
              {/* Open Series Section */}
              {openSeries.length > 0 && [
                <div key="open-header" className="px-2 py-1.5 text-sm font-semibold text-muted-foreground bg-muted/50">
                  {t('series.sections.openSeries')}
                </div>,
                ...openSeries.map((seriesItem) => (
                  <SelectItem key={seriesItem.id} value={seriesItem.id}>
                    {seriesItem.title}
                  </SelectItem>
                ))
              ]}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {/* Selected Series Info */}
      {selectedSeries && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Cup size={20} className="text-emerald-600" />
              <span>{selectedSeries.title}</span>
            </CardTitle>
          </CardHeader>
        </Card>
      )}

      {/* Leaderboard */}
      {selectedSeriesId && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <CardTitle className="flex items-center space-x-2">
              <Chart size={20} className="text-blue-600" />
              <span>{t('leaderboard.rankings', 'Rankings')}</span>
            </CardTitle>
            {leaderboard.length > 0 && !loadingLeaderboard && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleExportCSV}
                className="flex items-center gap-2"
              >
                <Export size={16} />
                {t('leaderboard.exportCSV')}
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {loadingLeaderboard ? (
              <LoadingSpinner />
            ) : leaderboard.length === 0 ? (
              <div className="text-center py-8">
                <Cup size={48} className="mb-4 mx-auto text-muted-foreground" />
                <h3 className="text-lg font-semibold text-foreground mb-2">
                  {t('leaderboard.empty')}
                </h3>
                <p className="text-muted-foreground">
                  {t('leaderboard.empty_help', 'No matches have been reported for this series yet.')}
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-20">{t('leaderboard.rank')}</TableHead>
                      <TableHead>{t('leaderboard.player')}</TableHead>
                      <TableHead className="text-center">{t('leaderboard.rating')}</TableHead>
                      <TableHead className="text-center">{t('leaderboard.games')}</TableHead>
                      <TableHead className="text-center">{t('leaderboard.wins')}</TableHead>
                      <TableHead className="text-center">{t('leaderboard.losses')}</TableHead>
                      <TableHead className="text-center">{t('leaderboard.winrate')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {leaderboard.map((row) => (
                      <TableRow key={row.playerId}>
                        <TableCell>
                          <div className="flex items-center space-x-2">
                            {getRankIcon(row.rank)}
                            <Badge variant={getRankBadge(row.rank)}>#{row.rank}</Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="font-medium">{row.displayName}</span>
                        </TableCell>
                        <TableCell className="text-center">
                          <span className="font-mono font-semibold">
                            {Math.round(row.rating)}
                          </span>
                        </TableCell>
                        <TableCell className="text-center">{row.games}</TableCell>
                        <TableCell className="text-center">
                          <div className="flex items-center justify-center space-x-1">
                            <TickCircle size={14} className="text-green-600" />
                            <span className="text-green-600 font-medium">{row.wins}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-center">
                          <div className="flex items-center justify-center space-x-1">
                            <CloseCircle size={14} className="text-red-600" />
                            <span className="text-red-600 font-medium">{row.losses}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-center">
                          <span
                            className={`font-medium ${calculateWinRate(row.wins, row.games) >= 60
                              ? 'text-green-600'
                              : calculateWinRate(row.wins, row.games) >= 40
                                ? 'text-yellow-600'
                                : 'text-red-600'
                              }`}
                          >
                            {calculateWinRate(row.wins, row.games)}%
                          </span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
