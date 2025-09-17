import ClubMembersManager from '@/components/ClubMembersManager'
import { ClubMergeManager } from '@/components/ClubMergeManager'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import apiClient from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { ApiError, Club, Series } from '@/types/api'
import { ArrowLeft, Calendar, Trophy, Users, UserMinus } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'

export function ClubDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { user, isAuthenticated, isClubMember, isClubAdmin } = useAuthStore()
  const [club, setClub] = useState<Club | null>(null)
  const [series, setSeries] = useState<Series[]>([])
  const [loading, setLoading] = useState(true)
  const [joining, setJoining] = useState(false)
  const [leaving, setLeaving] = useState(false)
  const [showLeaveDialog, setShowLeaveDialog] = useState(false)

  const loadClubDetails = useCallback(async () => {
    if (!id) return

    try {
      setLoading(true)

      // Load club details
      const clubData = await apiClient.getClub(id)
      setClub(clubData)

      // Load club series
      const seriesResponse = await apiClient.listSeries({
        pageSize: 50
      })

      // Filter series for this club (assuming series have clubId)
      const clubSeries = seriesResponse.items.filter(s =>
        s.clubId === id || s.visibility === 'SERIES_VISIBILITY_OPEN'
      )
      setSeries(clubSeries)

    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || 'Failed to load club details')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    if (id) {
      loadClubDetails()
    }
  }, [id, loadClubDetails])

  const handleJoinClub = async () => {
    if (!id || !user) return

    try {
      setJoining(true)
      await apiClient.joinClub({ clubId: id })
      toast.success(`Successfully joined ${club?.name}!`)

      // Refresh user memberships
      useAuthStore.getState().refreshUserMemberships()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('clubs.joinFailed'))
    } finally {
      setJoining(false)
    }
  }

  const handleLeaveClub = async () => {
    if (!id || !user) return

    if (!user.playerId) {
      toast.error(t('clubs.leaveUserNotFound'))
      return
    }

    try {
      setLeaving(true)
      await apiClient.leaveClub({ clubId: id, playerId: user.playerId })
      toast.success(`You have left ${club?.name}`)
      setShowLeaveDialog(false)

      // Refresh user memberships
      useAuthStore.getState().refreshUserMemberships()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('clubs.leaveFailed'))
    } finally {
      setLeaving(false)
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse">
          <div className="h-8 bg-muted rounded w-1/3 mb-4"></div>
          <div className="h-4 bg-muted rounded w-2/3 mb-2"></div>
          <div className="h-4 bg-muted rounded w-1/2"></div>
        </div>
      </div>
    )
  }

  if (!club) {
    return (
      <div className="text-center py-12">
        <h2 className="text-2xl font-bold mb-4">{t('clubs.detail.notFound')}</h2>
        <p className="text-muted-foreground mb-6">{t('clubs.detail.notFoundMessage')}</p>
        <Button asChild>
          <Link to="/clubs">{t('common.backToClubs')}</Link>
        </Button>
      </div>
    )
  }

  const isMember = isAuthenticated() && user && isClubMember(club.id)
  const isAdmin = isAuthenticated() && user && isClubAdmin(club.id)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/clubs">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Clubs
            </Link>
          </Button>
        </div>
      </div>

      {/* Club Info */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div>
              <CardTitle className="text-3xl">{club.name}</CardTitle>
              <CardDescription className="text-lg mt-2">
                {t('clubs.detail.description')}
              </CardDescription>
            </div>

            <div className="flex items-center space-x-2">
              {isMember && (
                <>
                  <Badge variant="default" className="flex items-center gap-1">
                    <Users className="h-3 w-3" />
                    {isAdmin ? t('clubs.detail.adminBadge') : t('clubs.detail.memberBadge')}
                  </Badge>
                  
                  <Dialog open={showLeaveDialog} onOpenChange={setShowLeaveDialog}>
                    <DialogTrigger asChild>
                      <Button variant="outline" className="flex items-center gap-2">
                        <UserMinus className="h-4 w-4" />
                        Leave Club
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Leave {club?.name}?</DialogTitle>
                        <DialogDescription>
                          Are you sure you want to leave this club? You can always rejoin later.
                        </DialogDescription>
                      </DialogHeader>
                      <DialogFooter>
                        <Button
                          variant="outline"
                          onClick={() => setShowLeaveDialog(false)}
                          disabled={leaving}
                        >
                          Cancel
                        </Button>
                        <Button
                          variant="destructive"
                          onClick={handleLeaveClub}
                          disabled={leaving}
                        >
                          {leaving ? t('clubs.leaving') : t('clubs.leaveClub')}
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </>
              )}

              {isAuthenticated() && !isMember && (
                <Button onClick={handleJoinClub} disabled={joining}>
                  {joining ? t('common.joining') : t('common.joinClub')}
                </Button>
              )}

              {!isAuthenticated() && (
                <Button asChild>
                  <Link to={`/login?returnTo=${encodeURIComponent(window.location.pathname)}`}>
                    Sign In to Join
                  </Link>
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">{t('clubs.detail.tabs.overview')}</TabsTrigger>
          <TabsTrigger value="series">{t('clubs.detail.tabs.series')}</TabsTrigger>
          {(isMember || isAdmin) && (
            <TabsTrigger value="members">{t('clubs.detail.tabs.members')}</TabsTrigger>
          )}
          {user && user.playerId && (
            <TabsTrigger value="merge">{t('clubs.detail.tabs.mergeePlayers')}</TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{t('clubs.detail.stats.activeSeries')}</CardTitle>
                <Trophy className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {series.filter(s => new Date(s.endsAt) > new Date()).length}
                </div>
                <p className="text-xs text-muted-foreground">
                  {t('clubs.detail.stats.currentlyRunning')}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{t('clubs.detail.stats.totalSeries')}</CardTitle>
                <Calendar className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{series.length}</div>
                <p className="text-xs text-muted-foreground">
                  {t('clubs.detail.stats.allTimeTournaments')}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{t('clubs.detail.stats.clubType')}</CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{t('clubs.detail.stats.open')}</div>
                <p className="text-xs text-muted-foreground">
                  {t('clubs.detail.stats.anyoneCanJoin')}
                </p>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="series" className="space-y-6">
          {series.length === 0 ? (
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-8">
                  <Trophy className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
                  <h3 className="text-lg font-medium mb-2">{t('clubs.detail.series.noSeries')}</h3>
                  <p className="text-muted-foreground">
                    {isAdmin
                      ? t('clubs.detail.series.createFirst')
                      : t('clubs.detail.series.noneCreated')
                    }
                  </p>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {series.map((s) => (
                <Card key={s.id}>
                  <CardHeader>
                    <CardTitle className="text-lg">{s.title}</CardTitle>
                    <CardDescription>
                      {new Date(s.startsAt).toLocaleDateString()} - {new Date(s.endsAt).toLocaleDateString()}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center justify-between">
                      <Badge variant={new Date(s.endsAt) > new Date() ? 'default' : 'secondary'}>
                        {new Date(s.endsAt) > new Date() ? t('common.active') : t('common.completed')}
                      </Badge>
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/series/${s.id}`}>View</Link>
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {(isMember || isAdmin) && (
          <TabsContent value="members" className="space-y-6">
            <ClubMembersManager clubId={club.id} clubName={club.name} />
          </TabsContent>
        )}

        {user && user.playerId && (
          <TabsContent value="merge" className="space-y-6">
            <ClubMergeManager clubId={club.id} />
          </TabsContent>
        )}
      </Tabs>
    </div>
  )
}

export default ClubDetailPage
