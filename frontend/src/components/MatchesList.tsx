import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { apiClient } from '@/services/api'
import type { MatchView } from '@/types/api'
import { CloseCircle, Cup, Edit2, Export, TickCircle, Trash, Calendar } from 'iconsax-reactjs'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SharedEmptyState } from './Styles'
import styled from 'styled-components'
import { colors } from '@/Styles'
import { EditMatchDialog } from './EditMatchDialog'
import { DeleteMatchDialog } from './DeleteMatchDialog'
import { exportMatchesToCSV, type MatchCSVData } from '@/utils/csvExport'

// Status Icons with semantic colors using tokens
const WinnerIcon = styled(TickCircle)`
  color: ${colors.ui.success};
`

const LoserIcon = styled(CloseCircle)`
  color: ${colors.ui.danger};
`

interface MatchesListProps {
  seriesId: string
  seriesStartDate?: string
  seriesEndDate?: string
  seriesName?: string
  refreshKey?: number
}

export function MatchesList({ seriesId, seriesStartDate, seriesEndDate, seriesName, refreshKey }: MatchesListProps) {
  const { t } = useTranslation()
  const [matches, setMatches] = useState<MatchView[]>([])
  const [loading, setLoading] = useState(true)
  const [hasNextPage, setHasNextPage] = useState(false)
  const [nextPageToken, setNextPageToken] = useState<string | undefined>()
  const [editingMatch, setEditingMatch] = useState<MatchView | null>(null)
  const [deletingMatch, setDeletingMatch] = useState<MatchView | null>(null)

  const loadMatches = useCallback(async (pageToken?: string) => {
    try {
      setLoading(true)
      const response = await apiClient.listMatches({
        seriesId: seriesId,
        pageSize: 20,
        cursorAfter: pageToken
      }, 'matches-list')

      if (pageToken) {
        setMatches(prev => [...prev, ...response.items])
      } else {
        setMatches(response.items)
      }
      setNextPageToken(response.endCursor)
      setHasNextPage(response.hasNextPage)
    } catch (error: unknown) {
      toast.error((error as Error).message || t('error.generic'))
    } finally {
      setLoading(false)
    }
  }, [seriesId, t])

  useEffect(() => {
    loadMatches()
  }, [loadMatches, refreshKey])

  const getScores = (match: MatchView): [number, number] => {
    const games = match.result?.tableTennis?.gamesWon ?? []
    return [games[0] ?? 0, games[1] ?? 0]
  }

  const getPlayerName = (match: MatchView, index: number) =>
    match.participants[index]?.displayName || t('matches.unknownPlayer')

  const getWinner = (match: MatchView) => {
    const [scoreA, scoreB] = getScores(match)
    if (scoreA === scoreB) {
      return undefined
    }
    return scoreA > scoreB ? getPlayerName(match, 0) : getPlayerName(match, 1)
  }

  const handleMatchUpdated = (updatedMatch: MatchView) => {
    setMatches(prev => prev.map(m => m.id === updatedMatch.id ? updatedMatch : m))
  }

  const handleMatchDeleted = (matchId: string) => {
    setMatches(prev => prev.filter(m => m.id !== matchId))
  }

  const handleEditMatch = (match: MatchView) => {
    setEditingMatch(match)
  }

  const handleDeleteMatch = (match: MatchView) => {
    setDeletingMatch(match)
  }

  const handleExportCSV = () => {
    const csvData: MatchCSVData[] = matches.map((match, index) => {
      const [scoreA, scoreB] = getScores(match)
      const playerAName = getPlayerName(match, 0)
      const playerBName = getPlayerName(match, 1)
      const playedAt = new Date(match.metadata.playedAt)

      return {
        sequence: index + 1,
        playerA: playerAName,
        scoreA,
        scoreB,
        playerB: playerBName,
        winner: getWinner(match) ?? '',
        date: playedAt.toLocaleDateString(),
        time: playedAt.toLocaleTimeString('sv-SE', {
          hour: '2-digit',
          minute: '2-digit'
        }),
        playedAt: match.metadata.playedAt
      }
    })

    exportMatchesToCSV(csvData, seriesName)
    toast.success(t('matches.exportSuccess'))
  }

  if (loading && matches.length === 0) {
    return <LoadingSpinner />
  }

  if (matches.length === 0) {
    return (
      <Card>
        <CardContent className="p-0">
          <SharedEmptyState>
            <Cup size={48} />
            <h3>{t('matches.empty')}</h3>
            <p>{t('matches.emptyDescription')}</p>
          </SharedEmptyState>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle>{t('matches.title')}</CardTitle>
          {matches.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleExportCSV}
              className="flex items-center gap-2"
            >
              <Export size={16} />
              {t('matches.exportCSV')}
            </Button>
          )}
        </CardHeader>
        <CardContent className="p-0">
          {/* Desktop Table View */}
          <div className="hidden md:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">#</TableHead>
                  <TableHead>{t('matches.player_a')}</TableHead>
                  <TableHead className="text-center">Score</TableHead>
                  <TableHead>{t('matches.player_b')}</TableHead>
                  <TableHead>{t('matches.played_at')}</TableHead>
                  <TableHead className="text-center">{t('matches.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {matches.map((match, index) => {
                  const [scoreA, scoreB] = getScores(match)
                  const playerAName = getPlayerName(match, 0)
                  const playerBName = getPlayerName(match, 1)
                  const winner = getWinner(match)
                  const playedAt = new Date(match.metadata.playedAt)

                  return (
                    <TableRow key={match.id}>
                      <TableCell className="text-center text-muted-foreground">
                        {index + 1}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className={playerAName === winner ? 'font-semibold' : ''}>
                            {playerAName}
                          </span>
                          {playerAName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-center">
                        <div className="flex items-center justify-center space-x-2">
                          <span className={scoreA > scoreB ? 'font-bold text-foreground' : 'text-muted-foreground'}>
                            {scoreA}
                          </span>
                          <span className="text-muted-foreground">-</span>
                          <span className={scoreB > scoreA ? 'font-bold text-foreground' : 'text-muted-foreground'}>
                            {scoreB}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className={playerBName === winner ? 'font-semibold' : ''}>
                            {playerBName}
                          </span>
                          {playerBName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        <div className="flex flex-col items-start">
                          <span className="text-sm">
                            {playedAt.toLocaleDateString()}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {playedAt.toLocaleTimeString('sv-SE', {
                              hour: '2-digit',
                              minute: '2-digit'
                            })}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-center space-x-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEditMatch(match)}
                            className="h-8 w-8 p-0"
                          >
                            <Edit2 className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteMatch(match)}
                            className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                          >
                            <Trash className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>

          {/* Mobile Card View */}
          <div className="md:hidden space-y-3 p-4">
            {matches.map((match, index) => {
              const [scoreA, scoreB] = getScores(match)
              const playerAName = getPlayerName(match, 0)
              const playerBName = getPlayerName(match, 1)
              const winner = getWinner(match)
              const playedAt = new Date(match.metadata.playedAt)

              return (
                <Card key={match.id} className="border-l-4 border-l-blue-500">
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm text-muted-foreground">#{index + 1}</span>
                      <div className="flex items-center space-x-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditMatch(match)}
                          className="h-8 w-8 p-0"
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeleteMatch(match)}
                          className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                        >
                          <Trash className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      {/* Players and Score */}
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-2">
                          <span className={playerAName === winner ? 'font-semibold' : ''}>
                            {playerAName}
                          </span>
                          {playerAName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>

                        <div className="flex items-center space-x-3">
                          <span className={scoreA > scoreB ? 'font-bold text-lg' : 'text-lg text-muted-foreground'}>
                            {scoreA}
                          </span>
                          <span className="text-muted-foreground">-</span>
                          <span className={scoreB > scoreA ? 'font-bold text-lg' : 'text-lg text-muted-foreground'}>
                            {scoreB}
                          </span>
                        </div>

                        <div className="flex items-center space-x-2">
                          <span className={playerBName === winner ? 'font-semibold' : ''}>
                            {playerBName}
                          </span>
                          {playerBName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                      </div>
                      
                      {/* Date and Time */}
                      <div className="flex items-center justify-center text-sm text-muted-foreground">
                        <Calendar size={14} className="mr-2" />
                        <span>
                          {playedAt.toLocaleDateString()}
                        </span>
                        <span className="mx-2">•</span>
                        <span>
                          {playedAt.toLocaleTimeString('sv-SE', {
                            hour: '2-digit',
                            minute: '2-digit'
                          })}
                        </span>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {hasNextPage && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            onClick={() => loadMatches(nextPageToken)}
            disabled={loading}
          >
            {loading ? <LoadingSpinner size="sm" /> : t('common.showMore')}
          </Button>
        </div>
      )}

      <EditMatchDialog
        match={editingMatch}
        isOpen={!!editingMatch}
        onClose={() => setEditingMatch(null)}
        onMatchUpdated={handleMatchUpdated}
        seriesStartDate={seriesStartDate}
        seriesEndDate={seriesEndDate}
      />

      <DeleteMatchDialog
        match={deletingMatch}
        isOpen={!!deletingMatch}
        onClose={() => setDeletingMatch(null)}
        onMatchDeleted={handleMatchDeleted}
      />
    </div>
  )
}