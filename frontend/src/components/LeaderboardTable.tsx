import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card, CardContent, CardDescription, CardHeader, CardTitle
} from '@/components/ui/card'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from '@/components/ui/table'
import { Cup, Export, Medal, TickCircle, CloseCircle } from 'iconsax-reactjs'
import { useTranslation } from 'react-i18next'
import { exportLeaderboardToCSV, type LeaderboardCSVData } from '@/utils/csvExport'
import { toast } from 'sonner'
import { testIds } from '@/lib/testIds'

export type UILBRow = {
  rank: number
  playerId: string
  displayName: string
  rating: number
  games: number
  wins: number
  losses: number
}

interface LeaderboardTableProps {
  leaderboard: UILBRow[]
  loading: boolean
  seriesTitle?: string
  showExport?: boolean
  title?: string
  description?: string
}

export function LeaderboardTable({
  leaderboard,
  loading,
  seriesTitle,
  showExport = true,
  title,
  description
}: LeaderboardTableProps) {
  const { t } = useTranslation()

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
    
    exportLeaderboardToCSV(csvData, seriesTitle)
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
    return 'secondary' as const
  }

  return (
    <Card id={testIds.leaderboard.container}>
      <CardHeader id={testIds.leaderboard.header}>
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <CardTitle id={testIds.leaderboard.title} className="flex items-center gap-2">
              <Cup size={20} className="text-primary" />
              {title || t('leaderboard.title')}
            </CardTitle>
            {description && (
              <CardDescription id={testIds.leaderboard.description}>{description}</CardDescription>
            )}
          </div>
          {showExport && leaderboard.length > 0 && (
            <Button
              id={testIds.leaderboard.exportBtn}
              variant="outline"
              size="sm"
              onClick={handleExportCSV}
              className="flex items-center gap-2"
            >
              <Export size={16} />
              {t('leaderboard.exportCSV')}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadingSpinner />
        ) : leaderboard.length === 0 ? (
          <div id={testIds.leaderboard.emptyState} className="text-center py-8">
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
            <Table id={testIds.leaderboard.table}>
              <TableHeader id={testIds.leaderboard.tableHeader}>
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
              <TableBody id={testIds.leaderboard.tableBody}>
                {leaderboard.map((row, index) => (
                  <TableRow key={row.playerId} id={testIds.leaderboard.row(index)} data-rank={row.rank} data-player-id={row.playerId}>
                    <TableCell id={testIds.leaderboard.rankCell(index)}>
                      <div className="flex items-center space-x-2">
                        {getRankIcon(row.rank) && (
                          <div id={testIds.leaderboard.rankIcon(index)}>
                            {getRankIcon(row.rank)}
                          </div>
                        )}
                        <Badge variant={getRankBadge(row.rank)}>#{row.rank}</Badge>
                      </div>
                    </TableCell>
                    <TableCell id={testIds.leaderboard.playerCell(index)}>
                      <span className="font-medium">{row.displayName}</span>
                    </TableCell>
                    <TableCell id={testIds.leaderboard.ratingCell(index)} className="text-center">
                      <span className="font-mono font-semibold">
                        {Math.round(row.rating)}
                      </span>
                    </TableCell>
                    <TableCell id={testIds.leaderboard.gamesCell(index)} className="text-center">{row.games}</TableCell>
                    <TableCell id={testIds.leaderboard.winsCell(index)} className="text-center">
                      <div className="flex items-center justify-center space-x-1">
                        <TickCircle size={14} className="text-green-600" />
                        <span className="text-green-600 font-medium">{row.wins}</span>
                      </div>
                    </TableCell>
                    <TableCell id={testIds.leaderboard.lossesCell(index)} className="text-center">
                      <div className="flex items-center justify-center space-x-1">
                        <CloseCircle size={14} className="text-red-600" />
                        <span className="text-red-600 font-medium">{row.losses}</span>
                      </div>
                    </TableCell>
                    <TableCell id={testIds.leaderboard.winrateCell(index)} className="text-center">
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
  )
}

// Helper function to normalize API responses (same as in LeaderboardPage)
export function normalizeLeaderboard(resp: { rows?: unknown[]; entries?: unknown[] } | unknown): UILBRow[] {
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