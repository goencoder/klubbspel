import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { apiClient } from '@/services/api'
import type { MatchView } from '@/types/api'
import { CloseCircle, Cup, TickCircle } from 'iconsax-reactjs'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SharedEmptyState } from './Styles'
import styled from 'styled-components'
import { colors } from '@/Styles'

// Status Icons with semantic colors using tokens
const SuccessIcon = styled(TickCircle)`
  color: ${colors.ui.success};
`

const DangerIcon = styled(CloseCircle)`
  color: ${colors.ui.danger};
`

interface MatchesListProps {
  seriesId: string
}

export function MatchesList({ seriesId }: MatchesListProps) {
  const { t } = useTranslation()
  const [matches, setMatches] = useState<MatchView[]>([])
  const [loading, setLoading] = useState(true)
  const [hasNextPage, setHasNextPage] = useState(false)
  const [nextPageToken, setNextPageToken] = useState<string | undefined>()

  useEffect(() => {
    loadMatches()
  }, [seriesId])

  const loadMatches = async (pageToken?: string) => {
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
    } catch (error: any) {
      toast.error(error.message || t('error.generic'))
    } finally {
      setLoading(false)
    }
  }

  const getWinner = (match: MatchView) => {
    return match.scoreA > match.scoreB ? match.playerAName : match.playerBName
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
            <p>No matches have been reported yet. Report the first match to get started!</p>
          </SharedEmptyState>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('matches.player_a')}</TableHead>
                <TableHead className="text-center">Score</TableHead>
                <TableHead>{t('matches.player_b')}</TableHead>
                <TableHead>{t('matches.played_at')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {matches.map((match) => {
                const winner = getWinner(match)

                return (
                  <TableRow key={match.id}>
                    <TableCell>
                      <div className="flex items-center space-x-2">
                        <span className={match.playerAName === winner ? 'font-semibold' : ''}>
                          {match.playerAName}
                        </span>
                        {match.playerAName === winner ? (
                          <SuccessIcon size={16} variant="Bold" />
                        ) : (
                          <DangerIcon size={16} />
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
                          <SuccessIcon size={16} variant="Bold" />
                        ) : (
                          <DangerIcon size={16} />
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(match.playedAt).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
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
    </div>
  )
}