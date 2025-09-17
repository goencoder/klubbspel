import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Badge } from '@/components/ui/badge'
import {
  Card, CardContent, CardDescription, CardHeader, CardTitle
} from '@/components/ui/card'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue
} from '@/components/ui/select'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from '@/components/ui/table'
import { apiClient } from '@/services/api'
import type { Series } from '@/types/api'
import { Chart, CloseCircle, Cup, Medal, TickCircle } from 'iconsax-reactjs'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'

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
function normalizeLeaderboard(resp: any): UILBRow[] {
  const raw = (resp?.rows ?? resp?.entries ?? []) as any[]
  if (!Array.isArray(raw)) return []
  return raw.map((r: any) => ({
    rank: r.rank ?? 0,
    playerId: r.playerId ?? r.player_id ?? '',
    displayName: r.displayName ?? r.playerName ?? r.display_name ?? '',
    rating: Number(r.rating ?? r.eloRating ?? 0),
    games: r.games ?? r.matchesPlayed ?? 0,
    wins: r.wins ?? r.matchesWon ?? 0,
    losses: r.losses ?? r.matchesLost ?? 0,
  }))
}

export function LeaderboardPage() {
  const { id: seriesIdFromParams } = useParams<{ id: string }>()
  const { t } = useTranslation()

  const [selectedSeriesId, setSelectedSeriesId] = useState<string>(seriesIdFromParams || '')
  const [leaderboard, setLeaderboard] = useState<UILBRow[]>([])
  const [series, setSeries] = useState<Series[]>([])
  const [loadingSeries, setLoadingSeries] = useState(true)
  const [loadingLeaderboard, setLoadingLeaderboard] = useState(false)

  useEffect(() => {
    loadSeries()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (selectedSeriesId) {
      loadLeaderboard(selectedSeriesId)
    } else {
      setLeaderboard([])
    }
  }, [selectedSeriesId])

  const loadSeries = async () => {
    try {
      setLoadingSeries(true)
      // Accept both {pageSize} and {page_size} depending on your client
      const response = await apiClient.listSeries?.({ pageSize: 100 }) ?? await apiClient.listSeries({ page_size: 100 })
      const items: Series[] = Array.isArray(response?.items) ? response.items : []
      setSeries(items)
      if (!seriesIdFromParams && items.length > 0) {
        setSelectedSeriesId(items[0].id as string)
      }
    } catch (error: any) {
      toast.error(error?.message || t('error.generic'))
      setSeries([])
    } finally {
      setLoadingSeries(false)
    }
  }

  const loadLeaderboard = async (seriesId: string) => {
    try {
      setLoadingLeaderboard(true)
      // Tolerate either {seriesId, pageSize} or {series_id, top_n}
      const resp =
        (await apiClient.getLeaderboard?.({ seriesId, pageSize: 50 }, 'leaderboard')) ??
        (await apiClient.getLeaderboard({ series_id: seriesId, top_n: 50 }, 'leaderboard'))

      setLeaderboard(normalizeLeaderboard(resp))
    } catch (error: any) {
      toast.error(error?.message || t('error.generic'))
      setLeaderboard([])
    } finally {
      setLoadingLeaderboard(false)
    }
  }

  const calculateWinRate = (wins: number, games: number) => {
    if (!games) return 0
    return Math.round((wins / games) * 100)
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

  const selectedSeries = series.find((s) => s.id === selectedSeriesId) || null

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
              {series.map((seriesItem) => (
                <SelectItem key={seriesItem.id as string} value={seriesItem.id as string}>
                  {seriesItem.title}
                </SelectItem>
              ))}
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
            {selectedSeries?.description && (
              <CardDescription>{selectedSeries.description}</CardDescription>
            )}
          </CardHeader>
        </Card>
      )}

      {/* Leaderboard */}
      {selectedSeriesId && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Chart size={20} className="text-blue-600" />
              <span>{t('leaderboard.rankings', 'Rankings')}</span>
            </CardTitle>
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
