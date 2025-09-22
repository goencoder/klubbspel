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
}

export function MatchesList({ seriesId, seriesStartDate, seriesEndDate, seriesName }: MatchesListProps) {
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
  }, [loadMatches])

  const getWinner = (match: MatchView) => {
    return match.scoreA > match.scoreB ? match.playerAName : match.playerBName
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
    const csvData: MatchCSVData[] = matches.map((match, index) => ({
      sequence: index + 1,
      playerA: match.playerAName,
      scoreA: match.scoreA,
      scoreB: match.scoreB,
      playerB: match.playerBName,
      winner: getWinner(match),
      date: new Date(match.playedAt).toLocaleDateString(),
      time: new Date(match.playedAt).toLocaleTimeString('sv-SE', { 
        hour: '2-digit', 
        minute: '2-digit' 
      }),
      playedAt: match.playedAt
    }))
    
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
                  const winner = getWinner(match)

                  return (
                    <TableRow key={match.id}>
                      <TableCell className="text-center text-muted-foreground">
                        {index + 1}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className={match.playerAName === winner ? 'font-semibold' : ''}>
                            {match.playerAName}
                          </span>
                          {match.playerAName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-center">
                        <div className="flex items-center justify-center space-x-2">
                          <span className={match.scoreA > match.scoreB ? 'font-bold text-foreground' : 'text-muted-foreground'}>
                            {match.scoreA}
                          </span>
                          <span className="text-muted-foreground">-</span>
                          <span className={match.scoreB > match.scoreA ? 'font-bold text-foreground' : 'text-muted-foreground'}>
                            {match.scoreB}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className={match.playerBName === winner ? 'font-semibold' : ''}>
                            {match.playerBName}
                          </span>
                          {match.playerBName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        <div className="flex flex-col items-start">
                          <span className="text-sm">
                            {new Date(match.playedAt).toLocaleDateString()}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {new Date(match.playedAt).toLocaleTimeString('sv-SE', { 
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
              const winner = getWinner(match)

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
                          <span className={match.playerAName === winner ? 'font-semibold' : ''}>
                            {match.playerAName}
                          </span>
                          {match.playerAName === winner ? (
                            <WinnerIcon size={16} />
                          ) : (
                            <LoserIcon size={16} />
                          )}
                        </div>
                        
                        <div className="flex items-center space-x-3">
                          <span className={match.scoreA > match.scoreB ? 'font-bold text-lg' : 'text-lg text-muted-foreground'}>
                            {match.scoreA}
                          </span>
                          <span className="text-muted-foreground">-</span>
                          <span className={match.scoreB > match.scoreA ? 'font-bold text-lg' : 'text-lg text-muted-foreground'}>
                            {match.scoreB}
                          </span>
                        </div>
                        
                        <div className="flex items-center space-x-2">
                          <span className={match.playerBName === winner ? 'font-semibold' : ''}>
                            {match.playerBName}
                          </span>
                          {match.playerBName === winner ? (
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
                          {new Date(match.playedAt).toLocaleDateString()}
                        </span>
                        <span className="mx-2">•</span>
                        <span>
                          {new Date(match.playedAt).toLocaleTimeString('sv-SE', { 
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