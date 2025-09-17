import { CreatePlayerDialog } from '@/components/CreatePlayerDialog'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useDebounce } from '@/hooks/useDebounce'
import { apiClient } from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { Club, Player } from '@/types/api'
import { Add, People, SearchNormal1 } from 'iconsax-reactjs'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { PageWrapper, PageHeaderSection, HeaderContent, SearchSection } from './Styles'

export function PlayersPage() {
  const { t } = useTranslation()
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [players, setPlayers] = useState<Player[]>([])
  const [clubs, setClubs] = useState<Club[]>([])
  const [loading, setLoading] = useState(true)
  const [hasNextPage, setHasNextPage] = useState(false)
  const [nextPageToken, setNextPageToken] = useState<string | undefined>()

  // Use global club selection from auth store
  const { selectedClubId } = useAuthStore()

  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  const loadClubs = useCallback(async () => {
    try {
      const response = await apiClient.listClubs({ pageSize: 100 })
      setClubs(response.items)
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.generic'))
    }
  }, [t])

  const loadPlayers = useCallback(async (pageToken?: string) => {
    try {
      setLoading(true)
      const response = await apiClient.listPlayers({
        searchQuery: debouncedSearchQuery || undefined,
        clubId: selectedClubId || undefined,
        pageSize: 20,
        cursorAfter: pageToken
      }, 'players-list')

      if (pageToken) {
        setPlayers(prev => [...prev, ...response.items])
      } else {
        setPlayers(response.items)
      }
      setNextPageToken(response.endCursor)
      setHasNextPage(response.hasNextPage)
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.generic'))
    } finally {
      setLoading(false)
    }
  }, [debouncedSearchQuery, selectedClubId, t])

  useEffect(() => {
    loadClubs()
  }, [loadClubs])

  useEffect(() => {
    loadPlayers()
  }, [loadPlayers])

  const handlePlayerCreated = (newPlayer: Player) => {
    setPlayers(prev => [newPlayer, ...prev])
    setShowCreateDialog(false)
    toast.success(t('common.success'))
  }

  const getClubName = (clubId: string) => {
    const club = clubs.find(c => c.id === clubId)
    return club?.name || t('players.unknownClub')
  }

  // Get the primary club for a player (first active membership)
  const getPrimaryClub = (player: Player) => {
    // First try to use the new club memberships
    if (player.clubMemberships && player.clubMemberships.length > 0) {
      const activeMembership = player.clubMemberships.find(m => m.active)
      if (activeMembership) {
        return getClubName(activeMembership.clubId)
      }
    }
    
    // Fallback to deprecated clubId field
    if (player.clubId) {
      return getClubName(player.clubId)
    }
    
    return t('players.unknownClub')
  }

  return (
    <PageWrapper>
      <PageHeaderSection>
        <HeaderContent>
          <h1>{t('players.title')}</h1>
          <p>{t('players.subtitle')}</p>
        </HeaderContent>
        <div className="flex gap-2">
          <Button onClick={() => setShowCreateDialog(true)} className="flex items-center space-x-2">
            <Add size={18} color="white" />
            <span>{t('players.create')}</span>
          </Button>
        </div>
      </PageHeaderSection>

      <div className="flex flex-col sm:flex-row gap-4">
        <SearchSection className="flex-1">
          <SearchNormal1 size={16} />
          <Input
            placeholder={t('common.search') + '...'}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </SearchSection>
        {/* Removed the redundant club selector - now using global club navigation in header */}
      </div>

      {loading && players.length === 0 ? (
        <LoadingSpinner />
      ) : players.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <People size={48} className="text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold text-foreground mb-2">{t('players.empty')}</h3>
            <p className="text-muted-foreground text-center mb-4">
              {searchQuery || selectedClubId ? t('players.searchAdjust') : t('players.getStarted')}
            </p>
            {!searchQuery && !selectedClubId && (
              <Button onClick={() => setShowCreateDialog(true)} className="mt-4">
                <Add size={18} color="white" className="mr-2" />
                {t('players.create')}
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {players.map((player) => (
              <Card key={player.id} className="hover:shadow-md transition-shadow">
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <CardTitle className="text-lg">{player.displayName}</CardTitle>
                    <Badge variant={player.active ? 'default' : 'secondary'}>
                      {player.active ? t('common.active') : t('common.inactive')}
                    </Badge>
                  </div>
                  <CardDescription>
                    <div className="text-sm font-medium text-foreground">
                      {getPrimaryClub(player)}
                    </div>
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="text-sm text-muted-foreground">
                    {t('common.joined')} {new Date().toLocaleDateString()}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {hasNextPage && (
            <div className="flex justify-center">
              <Button
                variant="outline"
                onClick={() => loadPlayers(nextPageToken)}
                disabled={loading}
              >
                {loading ? <LoadingSpinner size="sm" /> : t('common.showMore')}
              </Button>
            </div>
          )}
        </div>
      )}

      <CreatePlayerDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onPlayerCreated={handlePlayerCreated}
      />
    </PageWrapper>
  )
}